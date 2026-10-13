package sqlite

import (
	"database/sql"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// restoreBackup replaces the database with a backup file, as a user would
// when restoring by hand, and removes the WAL files of the replaced database.
func restoreBackup(t *testing.T, dir datadir.Dir, backupPath string) {
	t.Helper()
	dbPath := dir.Join(datadir.DatabaseFile)
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.Remove(dbPath + suffix); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	src, err := os.Open(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = src.Close() }()
	dst, err := os.OpenFile(dbPath, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatal(err)
	}
	if err := dst.Close(); err != nil {
		t.Fatal(err)
	}
}

// MVP-STO-02 / codex-review-004 B3: a VACUUM INTO backup is a rollback-journal
// database. Restoring one written by a newer binary and opening it with an
// older binary must be refused without switching it to WAL or touching any
// file.
func TestOlderBinaryLeavesRestoredNewerBackupUntouched(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(embedded(t), factsMigration(t))
	v3 := Migration{Version: 3, Name: "extra", SQL: `CREATE TABLE extra (id INTEGER PRIMARY KEY);`}
	v4 := Migration{Version: 4, Name: "more", SQL: `CREATE TABLE more (id INTEGER PRIMARY KEY);`}
	db, _ := open(t, dir, Options{Migrations: append(slices.Clone(migs), v3)})
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "v3 data")
	_ = db.Close()
	// Migrating to v4 backs up the schema-v3 database.
	db, res := open(t, dir, Options{Migrations: append(slices.Clone(migs), v3, v4)})
	_ = db.Close()
	if res.BackupPath == "" || res.FromVersion != 3 {
		t.Fatalf("open result = %+v", res)
	}
	restoreBackup(t, dir, res.BackupPath)

	before := testutil.SnapshotTree(t, dir.Path)
	openRefused(t, dir, Options{Migrations: migs}, "upgrade specmetry")
	testutil.AssertTreeUnchanged(t, dir.Path, before)
}

// MVP-STO-02 / codex-review-004 B3: a newer binary crashed and left an
// unmerged WAL. An older binary refuses it without recovering or
// checkpointing the WAL.
func TestOlderBinaryLeavesCrashedNewerDatabaseUntouched(t *testing.T) {
	dir := testutil.TempHome(t)
	crashWithWAL(t, "CRASH_FACTS=3", "CRASH_PAYLOAD=newer")
	before := testutil.SnapshotTree(t, dir.Path)
	openRefused(t, dir, Options{Migrations: embedded(t)}, "upgrade specmetry")
	testutil.AssertTreeUnchanged(t, dir.Path, before)
}

// codex-review-004 B3: a connection opened before another process upgraded
// the schema must not write: WriteTx re-checks compatibility after taking the
// write lock and before running the callback.
func TestWriteTxRefusesAfterConcurrentSchemaUpgrade(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(embedded(t), factsMigration(t))
	old, _ := open(t, dir, Options{Migrations: migs})
	insertFact(t, old, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "before")

	v3 := Migration{Version: 3, Name: "newer", SQL: `CREATE TABLE newer (id INTEGER PRIMARY KEY);`}
	newer, _ := open(t, dir, Options{Migrations: append(migs, v3)})
	_ = newer.Close()

	assertWriteRefused(t, old, apperr.ExitEnvironment, "upgrade specmetry")
}

// codex-review-004 B3: the same when another process raised
// min_binary_version without changing the schema version.
func TestWriteTxRefusesAfterMinBinaryVersionRaised(t *testing.T) {
	dir := testutil.TempHome(t)
	old, _ := open(t, dir, Options{Migrations: append(embedded(t), factsMigration(t)), BinaryVersion: "0.1.0"})
	insertFact(t, old, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "before")

	other, err := sql.Open("sqlite", dsn(dir.Join(datadir.DatabaseFile)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.ExecContext(ctx, `UPDATE meta SET min_binary_version = '9.9.9' WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	_ = other.Close()

	assertWriteRefused(t, old, apperr.ExitEnvironment, "9.9.9")
}

func assertWriteRefused(t *testing.T, db *DB, wantExit int, want string) {
	t.Helper()
	seqBefore := countRows(t, db, `SELECT seq_counter FROM meta`)
	factsBefore := countRows(t, db, `SELECT count(*) FROM facts`)
	called := false
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		called = true
		seq, err := NextSeq(ctx, tx)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO facts (seq, project_id, payload) VALUES (?, ?, 'after')`, seq, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6")
		return err
	})
	if called {
		t.Fatal("WriteTx ran its callback on an incompatible database")
	}
	if apperr.ExitCode(err) != wantExit || !contains(err, want) {
		t.Fatalf("err = %v, want exit %d mentioning %q", err, wantExit, want)
	}
	if got := countRows(t, db, `SELECT seq_counter FROM meta`); got != seqBefore {
		t.Fatalf("seq_counter %d -> %d", seqBefore, got)
	}
	if got := countRows(t, db, `SELECT count(*) FROM facts`); got != factsBefore {
		t.Fatalf("facts %d -> %d", factsBefore, got)
	}
}

func contains(err error, s string) bool {
	return err != nil && (strings.Contains(err.Error(), s) || strings.Contains(detailsOf(err), s))
}

// A min_binary_version that is not a version number cannot be honoured, so
// the database is refused rather than opened (fail closed).
func TestUnreadableMinBinaryVersionIsRefused(t *testing.T) {
	dir := testutil.TempHome(t)
	bad := Migration{Version: 2, Name: "bad_min", SQL: `UPDATE meta SET min_binary_version = 'next' WHERE id = 1;`}
	migs := append(embedded(t), bad)
	db, _ := open(t, dir, Options{Migrations: migs})
	_ = db.Close()
	before := testutil.SnapshotTree(t, dir.Path)
	openRefused(t, dir, Options{Migrations: migs}, "next")
	testutil.AssertTreeUnchanged(t, dir.Path, before)
}
