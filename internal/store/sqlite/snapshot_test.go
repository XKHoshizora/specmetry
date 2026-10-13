package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// stateQuery summarizes everything the tests write.
const stateQuery = `SELECT (SELECT schema_version FROM meta) || '/' || (SELECT seq_counter FROM meta) || '/' ||
  (SELECT count(*) FROM facts) || '/' || (SELECT coalesce(max(seq), 0) FROM facts)`

func scanState(sdb *sql.DB) (string, error) {
	var s string
	err := sdb.QueryRowContext(ctx, stateQuery).Scan(&s)
	return s, err
}

// snapshotState reads the database through the snapshot reader.
func snapshotState(t *testing.T, dbPath string) string {
	t.Helper()
	var s string
	if _, err := readSnapshot(ctx, dbPath, func(sdb *sql.DB) error {
		var err error
		s, err = scanState(sdb)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return s
}

// sqliteState lets SQLite itself recover a copy of the database files and
// reads the same state: the reference the snapshot reader must match.
func sqliteState(t *testing.T, dbPath string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "copy.db")
	for _, suffix := range []string{"", "-wal", "-shm"} {
		src, err := os.Open(dbPath + suffix)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		out, err := os.Create(dst + suffix)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(out, src); err != nil {
			t.Fatal(err)
		}
		_ = src.Close()
		_ = out.Close()
	}
	sdb, err := sql.Open("sqlite", dsn(dst))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sdb.Close() }()
	s, err := scanState(sdb)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func walFrames(t *testing.T, dbPath string) (frames int64, pageSize int64) {
	t.Helper()
	fi, err := os.Stat(dbPath + "-wal")
	if err != nil {
		t.Fatal(err)
	}
	pageSize = 4096
	return (fi.Size() - walHeaderSize) / (walFrameHeaderSize + pageSize), pageSize
}

// The snapshot reader applies the WAL as SQLite's own recovery would, when
// the last frame is damaged (a crash in the middle of writing it).
func TestSnapshotIgnoresDamagedLastFrame(t *testing.T) {
	dir := testutil.TempHome(t)
	crashWithWAL(t, "CRASH_FACTS=5", "CRASH_PAYLOAD=x")
	dbPath := dir.Join(datadir.DatabaseFile)
	frames, pageSize := walFrames(t, dbPath)
	f, err := os.OpenFile(dbPath+"-wal", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Flip a byte in the page data of the last frame.
	off := walHeaderSize + (frames-1)*(walFrameHeaderSize+pageSize) + walFrameHeaderSize + 100
	b := make([]byte, 1)
	if _, err := f.ReadAt(b, off); err != nil {
		t.Fatal(err)
	}
	b[0] ^= 0xff
	if _, err := f.WriteAt(b, off); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	want := sqliteState(t, dbPath)
	if got := snapshotState(t, dbPath); got != want {
		t.Fatalf("snapshot %s, SQLite recovery %s", got, want)
	}
	if !strings.HasSuffix(want, "/4/4") {
		t.Fatalf("damaging the last frame should lose the last fact; SQLite sees %s", want)
	}
}

// ... and when the WAL ends in the middle of a frame.
func TestSnapshotIgnoresTruncatedLastFrame(t *testing.T) {
	dir := testutil.TempHome(t)
	crashWithWAL(t, "CRASH_FACTS=5", "CRASH_PAYLOAD=x")
	dbPath := dir.Join(datadir.DatabaseFile)
	fi, err := os.Stat(dbPath + "-wal")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(dbPath+"-wal", fi.Size()-1000); err != nil {
		t.Fatal(err)
	}
	want := sqliteState(t, dbPath)
	if got := snapshotState(t, dbPath); got != want {
		t.Fatalf("snapshot %s, SQLite recovery %s", got, want)
	}
}

// spillCrashHelper starts a transaction large enough to spill uncommitted
// frames into the WAL, then exits before committing. The transaction first
// changes a row in a small table that is never touched again, so that page
// is evicted and its uncommitted version lands in the WAL.
func spillCrashHelper() int {
	dir, err := datadir.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	migs, err := helperMigrations(false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	db, _, err := Open(context.Background(), dir, Options{Migrations: migs})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	c := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := writeFact(c, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "committed"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	if _, err := db.sql.ExecContext(c, `CREATE TABLE scratch (id INTEGER PRIMARY KEY, v TEXT NOT NULL); INSERT INTO scratch VALUES (1, 'committed')`); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if _, err := db.sql.ExecContext(c, `PRAGMA cache_size = 2`); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_ = db.WriteTx(c, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(c, `UPDATE scratch SET v = 'uncommitted' WHERE id = 1`); err != nil {
			return err
		}
		for i := 0; i < 300; i++ {
			seq, err := NextSeq(c, tx)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(c, `INSERT INTO facts (seq, project_id, payload) VALUES (?, 'p', hex(randomblob(2000)))`, seq); err != nil {
				return err
			}
		}
		os.Exit(0) // crash before COMMIT
		return nil
	})
	return 1
}

// Frames written by a transaction that never committed are ignored.
func TestSnapshotIgnoresUncommittedFrames(t *testing.T) {
	dir := testutil.TempHome(t)
	if out, err := testutil.HelperCommand(t, "spillcrash").CombinedOutput(); err != nil {
		t.Fatalf("spill crash: %v\n%s", err, out)
	}
	dbPath := dir.Join(datadir.DatabaseFile)
	if frames, _ := walFrames(t, dbPath); frames < 50 {
		t.Fatalf("only %d WAL frames; the transaction did not spill", frames)
	}
	want := sqliteState(t, dbPath)
	if want != "2/3/3/3" {
		t.Fatalf("SQLite recovery sees %s, want only the 3 committed facts", want)
	}
	if got := snapshotState(t, dbPath); got != want {
		t.Fatalf("snapshot %s, SQLite recovery %s", got, want)
	}
	var v string
	if _, err := readSnapshot(ctx, dbPath, func(sdb *sql.DB) error {
		return sdb.QueryRowContext(ctx, `SELECT v FROM scratch`).Scan(&v)
	}); err != nil || v != "committed" {
		t.Fatalf("snapshot sees scratch = %q (%v), want the committed value", v, err)
	}
}

// restartCrashHelper fills the WAL, checkpoints it so the next write starts
// over at its beginning with a new salt, writes one more fact and crashes:
// stale frames of the previous generation remain after the new ones.
func restartCrashHelper() int {
	dir, err := datadir.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	migs, err := helperMigrations(false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	db, _, err := Open(context.Background(), dir, Options{Migrations: migs})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	c := context.Background()
	for i := 0; i < 20; i++ {
		if _, err := writeFact(c, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", strings.Repeat("old", 500)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	if _, err := db.sql.ExecContext(c, `PRAGMA wal_checkpoint(RESTART)`); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if _, err := writeFact(c, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "new generation"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	os.Exit(0)
	return 0
}

// After a WAL restart, frames of the previous generation are ignored.
func TestSnapshotIgnoresPreviousWALGeneration(t *testing.T) {
	dir := testutil.TempHome(t)
	if out, err := testutil.HelperCommand(t, "restartcrash").CombinedOutput(); err != nil {
		t.Fatalf("restart crash: %v\n%s", err, out)
	}
	dbPath := dir.Join(datadir.DatabaseFile)
	want := sqliteState(t, dbPath)
	if want != "2/21/21/21" {
		t.Fatalf("SQLite recovery sees %s, want 21 facts", want)
	}
	if got := snapshotState(t, dbPath); got != want {
		t.Fatalf("snapshot %s, SQLite recovery %s", got, want)
	}
}

// A write that lands while the snapshot is being read is detected, and the
// read is repeated on the new state.
func TestSnapshotRetriesWhenDatabaseChanges(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{Migrations: append(embedded(t), factsMigration(t))})
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "first")
	calls := 0
	snapshotHook = func() {
		calls++
		if calls == 1 {
			insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "during the read")
		}
	}
	t.Cleanup(func() { snapshotHook = nil })
	if got := snapshotState(t, db.Path()); got != "2/2/2/2" || calls < 2 {
		t.Fatalf("snapshot %s after %d reads, want the state after the concurrent write", got, calls)
	}
}

// A database that never stops changing is reported, not read inconsistently.
func TestSnapshotGivesUpOnContinuousChanges(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{Migrations: append(embedded(t), factsMigration(t))})
	snapshotHook = func() { insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "again") }
	t.Cleanup(func() { snapshotHook = nil })
	_, err := readSnapshotMeta(ctx, db.Path())
	var u *uninspectableError
	if !errors.As(err, &u) || !strings.Contains(u.reason, "kept changing") {
		t.Fatalf("err = %v", err)
	}
}

// A rollback journal that would have to be played back is a limitation:
// doctor reports it and Open refuses, and neither touches anything.
func TestHotJournalIsReportedNotRecovered(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{})
	_ = db.Close()
	journal := append(append([]byte{}, journalMagic...), make([]byte, 504)...)
	if err := os.WriteFile(dir.Join(datadir.DatabaseFile)+"-journal", journal, 0o600); err != nil {
		t.Fatal(err)
	}
	before := testutil.SnapshotTree(t, dir.Path)
	in, err := Inspect(ctx, dir)
	if err != nil || !strings.Contains(in.Limitation, "rollback journal") {
		t.Fatalf("Inspect = %+v, %v", in, err)
	}
	openRefused(t, dir, Options{}, "rollback journal")
	testutil.AssertTreeUnchanged(t, dir.Path, before)
}

// The preflight does not stand in the way of normal crash recovery: a
// compatible binary opens a crashed database and sees every committed fact.
func TestCompatibleBinaryRecoversCrashedDatabase(t *testing.T) {
	dir := testutil.TempHome(t)
	crashWithWAL(t, "CRASH_FACTS=4", "CRASH_PAYLOAD=x")
	db, _ := open(t, dir, Options{Migrations: append(embedded(t), factsMigration(t))})
	if n := countRows(t, db, `SELECT count(*) FROM facts`); n != 4 {
		t.Fatalf("%d facts after recovery, want 4", n)
	}
}

// codex-review-004 N1: tombstone compaction that was interrupted after the
// rows were deleted is finished on the next open, although nothing is left
// to delete by then.
func TestTombstoneCompactionResumesAfterCrash(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(embedded(t), factsMigration(t))
	canary := testutil.NewCanary(t)
	db, _ := open(t, dir, Options{Migrations: migs})
	insertFact(t, db, purgedProject, canary)
	insertFact(t, db, keptProject, "kept")
	_ = db.Close()

	writeTombstones(t, dir, purgedProject)
	compactionHook = func() error { return errors.New("simulated crash before compaction") }
	_, _, err := Open(ctx, dir, Options{Migrations: migs})
	compactionHook = nil
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "simulated crash") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(dir.Join(datadir.CompactionFile)); err != nil {
		t.Fatalf("no compaction marker after the interrupted purge: %v", err)
	}
	if hits := testutil.FindInTree(t, dir.Path, canary); len(hits) == 0 {
		t.Fatal("the simulated crash already removed the data; the test does not exercise recovery")
	}

	db, res := open(t, dir, Options{Migrations: migs})
	_ = db.Close()
	if res.TombstoneRowsGone != 0 {
		t.Fatalf("rows were deleted again: %+v", res)
	}
	if hits := testutil.FindInTree(t, dir.Path, canary); len(hits) != 0 {
		t.Fatalf("purged data still on disk after the resumed compaction: %v", hits)
	}
	if _, err := os.Stat(dir.Join(datadir.CompactionFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("compaction marker left behind: %v", err)
	}
}

// Characterization of the driver in use (codex-review-004 B2): each way of
// opening the database read-only through SQLite itself modifies the files in
// at least one common state, which is why snapshot.go does not use them. If
// this test starts failing, the driver changed and the design reason in
// snapshot.go and architecture.md §11.5 should be revisited.
func TestSQLiteReadOnlyModesModifyFiles(t *testing.T) {
	cases := []struct {
		name, params string
		crashed      bool
	}{
		{"read-write with query_only, crashed WAL", "_pragma=query_only(1)", true},
		{"mode=ro, crashed WAL", "mode=ro", true},
		{"mode=ro, cleanly closed", "mode=ro", false},
		{"mode=ro with readonly_shm, cleanly closed", "mode=ro&readonly_shm=1", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := testutil.TempHome(t)
			if c.crashed {
				crashWithWAL(t, "CRASH_FACTS=3", "CRASH_PAYLOAD=x")
			} else {
				db, _ := open(t, dir, Options{})
				_ = db.Close()
			}
			before := testutil.SnapshotTree(t, dir.Path)
			sdb, err := sql.Open("sqlite", "file:"+escapePath(dir.Join(datadir.DatabaseFile))+"?"+c.params)
			if err != nil {
				t.Fatal(err)
			}
			var v int
			_ = sdb.QueryRowContext(ctx, `SELECT schema_version FROM meta`).Scan(&v)
			_ = sdb.Close()
			if testutil.DiffSnapshots(before, testutil.SnapshotTree(t, dir.Path)) == "" {
				t.Fatalf("%s left the files unchanged; the driver's behavior changed", c.params)
			}
		})
	}
}
