package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

// uncommittedWriterHelper opens the database with a plain SQLite connection
// in the journal mode $UNCOMMITTED_JOURNAL, starts a transaction that sets
// schema_version to 9, writes enough to spill the uncommitted pages to disk,
// signals readiness and rolls back only when told to stop.
func uncommittedWriterHelper() int {
	c := context.Background()
	dir, err := datadir.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	db, err := sql.Open("sqlite", "file:"+escapePath(dir.Join(datadir.DatabaseFile))+
		"?_pragma=journal_mode("+os.Getenv("UNCOMMITTED_JOURNAL")+")&_pragma=cache_size(2)&_pragma=busy_timeout(5000)")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	conn, err := db.Conn(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = conn.Close() }()
	steps := []string{
		`CREATE TABLE IF NOT EXISTS scratch (x TEXT)`,
		`BEGIN IMMEDIATE`,
		`UPDATE meta SET schema_version = 9 WHERE id = 1`,
	}
	for i := 0; i < 300; i++ {
		steps = append(steps, `INSERT INTO scratch VALUES (hex(randomblob(2000)))`)
	}
	for _, s := range steps {
		if _, err := conn.ExecContext(c, s); err != nil {
			fmt.Fprintln(os.Stderr, s, err)
			return 1
		}
	}
	if err := os.WriteFile(os.Getenv("UNCOMMITTED_READY"), nil, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for {
		if _, err := os.Stat(os.Getenv("UNCOMMITTED_STOP")); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := conn.ExecContext(c, `ROLLBACK`); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// startUncommittedWriter runs uncommittedWriterHelper and waits until its
// uncommitted transaction is on disk.
func startUncommittedWriter(t *testing.T, journal string) {
	t.Helper()
	ready := filepath.Join(testdir.New(t), "ready")
	stop := filepath.Join(testdir.New(t), "stop")
	cmd := testutil.HelperCommand(t, "uncommitted", "UNCOMMITTED_JOURNAL="+journal, "UNCOMMITTED_READY="+ready, "UNCOMMITTED_STOP="+stop)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.WriteFile(stop, nil, 0o600)
		_ = cmd.Wait()
	})
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("uncommitted writer did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// restoredRollbackDatabase leaves a schema-v2 rollback-journal database (a
// restored VACUUM INTO backup) in dir.
func restoredRollbackDatabase(t *testing.T, dir datadir.Dir) {
	t.Helper()
	migs := append(foundation(t), factsMigration(t))
	db, _ := open(t, dir, Options{Migrations: migs})
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "committed")
	_ = db.Close()
	v3 := Migration{Version: 3, Name: "extra", SQL: `CREATE TABLE extra (id INTEGER PRIMARY KEY);`}
	db, res := open(t, dir, Options{Migrations: append(migs, v3)})
	_ = db.Close()
	restoreBackup(t, dir, res.BackupPath)
}

// codex-review-005 B2, as reported: a rollback-journal transaction that has
// not committed has already written its pages into the database file (the
// journal is kept in memory, so no -journal file warns the reader). The
// diagnosis must never report the uncommitted schema_version.
func TestInspectNeverReportsUncommittedRollbackWrite(t *testing.T) {
	dir := testutil.TempHome(t)
	restoredRollbackDatabase(t, dir)
	startUncommittedWriter(t, "MEMORY")
	in, err := Inspect(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if in.Limitation == "" && in.SchemaVersion != 2 {
		t.Fatalf("Inspect reported schema v%d during an uncommitted write: %+v", in.SchemaVersion, in)
	}
	if in.SchemaVersion == 9 {
		t.Fatalf("Inspect reported uncommitted data: %+v", in)
	}
}

// The same with a WAL-mode transaction whose uncommitted frames are in the
// WAL: only the committed version may be reported.
func TestInspectNeverReportsUncommittedWALWrite(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{Migrations: append(foundation(t), factsMigration(t))})
	_ = db.Close()
	startUncommittedWriter(t, "WAL")
	in, err := Inspect(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if in.SchemaVersion == 9 || (in.Limitation == "" && in.SchemaVersion != 2) {
		t.Fatalf("Inspect during an uncommitted WAL write: %+v", in)
	}
}
