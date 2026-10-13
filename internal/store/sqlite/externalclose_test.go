package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// plainCloseHelper opens the database with an ordinary SQLite connection,
// reads it and closes it. Being the last connection, and without a persistent
// WAL, SQLite deletes the -wal and -shm files on close.
func plainCloseHelper() int {
	dir, err := datadir.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	raw, err := sql.Open("sqlite", "file:"+escapePath(dir.Join(datadir.DatabaseFile)))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	var n int
	if err := raw.QueryRow(`SELECT count(*) FROM meta`).Scan(&n); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := raw.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// codex-review-006 R2, kept as a regression test of the adjusted MVP-STO-07
// (2026-10-13). doctor has judged the files readable (-wal and -shm present)
// when another, ordinary SQLite program closes the database and removes them.
// SQLite's read-only connection may then create empty -wal/-shm files; that is
// the documented, limited known behavior. Required:
//   - the version is reported as unknown, never as a version or as healthy;
//   - the diagnosis itself adds at most empty, private -wal/-shm files, and
//     changes or removes nothing (the main database file stays byte for
//     byte as the other program left it);
//   - the committed data is intact.
func TestInspectDuringExternalCloseReportsUnknown(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{Migrations: append(embedded(t), factsMigration(t))})
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "committed")
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "committed")
	_ = db.Close() // persistent WAL: -wal and -shm stay
	dbFile := dir.Join(datadir.DatabaseFile)
	wantFacts := databaseFacts(t, dbFile)

	var afterExternal map[string]string
	readOnlyHook = func() {
		if out, err := testutil.HelperCommand(t, "plainclose").CombinedOutput(); err != nil {
			t.Errorf("external close: %v\n%s", err, out)
		}
		for _, suffix := range []string{"-wal", "-shm"} {
			if _, err := os.Stat(dbFile + suffix); !os.IsNotExist(err) {
				t.Errorf("the external close left %s (%v); the race was not reproduced", suffix, err)
			}
		}
		afterExternal = testutil.SettledSnapshot(t, dir.Path)
	}
	t.Cleanup(func() { readOnlyHook = nil })

	in, err := Inspect(ctx, dir)
	readOnlyHook = nil
	if err != nil {
		t.Fatal(err)
	}
	if in.Limitation == "" || in.SchemaVersion != 0 || in.MinBinaryVersion != "" {
		t.Fatalf("inspection = %+v, want the version reported as unknown", in)
	}
	if in.Compatibility("0.0.0-dev") != nil {
		t.Fatal("an unknown version must not be judged incompatible either")
	}
	after := testutil.SnapshotTree(t, dir.Path)
	assertOnlyCreated(t, dir, afterExternal, after, datadir.DatabaseFile+"-wal", datadir.DatabaseFile+"-shm")
	if problems, err := datadir.Check(dir); err != nil || len(problems) != 0 {
		t.Fatalf("permissions after the check: %v %v", problems, err)
	}
	if got := databaseFacts(t, dbFile); got != wantFacts {
		t.Fatalf("database content %s, want %s", got, wantFacts)
	}
	t.Logf("files the read-only connection added: %q", testutil.DiffSnapshots(afterExternal, after))
}
