package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
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

// externalCloseDuringCheck makes the next read-only check pause after it has
// judged the files readable, while an independent process closes the
// database with an ordinary SQLite connection (removing -wal and -shm). It
// returns the snapshot taken once that process has finished, so that what
// the outside process did is not counted as Specmetry's side effect.
func externalCloseDuringCheck(t *testing.T, dir datadir.Dir) *map[string]string {
	t.Helper()
	dbFile := dir.Join(datadir.DatabaseFile)
	var afterExternal map[string]string
	readOnlyHook = func() {
		readOnlyHook = nil // only the first check
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
	return &afterExternal
}

// requireOnlyEmptyAuxiliaryFiles checks Specmetry's own side effects after
// the outside close: at most empty -wal/-shm files, with private
// permissions; the schema, seq, facts and min_binary_version unchanged.
func requireOnlyEmptyAuxiliaryFiles(t *testing.T, dir datadir.Dir, afterExternal map[string]string, wantFacts string) {
	t.Helper()
	assertOnlyCreated(t, dir, afterExternal, testutil.SnapshotTree(t, dir.Path), datadir.DatabaseFile+"-wal", datadir.DatabaseFile+"-shm")
	if problems, err := datadir.Check(dir); err != nil || len(problems) != 0 {
		t.Fatalf("permissions: %v %v", problems, err)
	}
	if got := databaseFacts(t, dir.Join(datadir.DatabaseFile)); got != wantFacts {
		t.Fatalf("database content %s, want %s", got, wantFacts)
	}
	if held, err := lockHeld(dir.Join(datadir.MigrateLockFile)); err != nil || held {
		t.Fatalf("open lock still held (%v)", err)
	}
}

// MVP-STO-02 (product decision of 2026-10-13, option A): the same outside
// close during Open's version check. Specmetry may add empty, private
// -wal/-shm files and nothing else; it never changes the schema, data,
// facts or seq; an incompatible database is still refused; and a version it
// could not establish is never taken as compatible.
func TestOpenDuringExternalCloseOnlyAddsEmptyAuxiliaryFiles(t *testing.T) {
	setup := func(t *testing.T) (datadir.Dir, []Migration, string) {
		dir := testutil.TempHome(t)
		migs := append(embedded(t), factsMigration(t))
		db, _ := open(t, dir, Options{Migrations: migs})
		insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "committed")
		insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "committed")
		_ = db.Close()
		return dir, migs, databaseFacts(t, dir.Join(datadir.DatabaseFile))
	}

	t.Run("incompatible database is still refused", func(t *testing.T) {
		dir, _, want := setup(t)
		afterExternal := externalCloseDuringCheck(t, dir)
		db, _, err := Open(ctx, dir, Options{Migrations: embedded(t)}) // knows schema v1 only
		if err == nil {
			_ = db.Close()
			t.Fatal("a binary that knows schema v1 opened a schema v2 database")
		}
		// Either the committed version was read and refused as newer, or the
		// change was noticed and the version left unconfirmed.
		refusedAsNewer := strings.Contains(err.Error(), "upgrade specmetry")
		refusedAsUnconfirmed := strings.Contains(err.Error(), "cannot confirm")
		if apperr.ExitCode(err) != apperr.ExitEnvironment || (!refusedAsNewer && !refusedAsUnconfirmed) {
			t.Fatalf("err = %v", err)
		}
		requireOnlyEmptyAuxiliaryFiles(t, dir, *afterExternal, want)
		t.Logf("refused: %v; added: %q", err, testutil.DiffSnapshots(*afterExternal, testutil.SnapshotTree(t, dir.Path)))
	})

	t.Run("compatible database is not misjudged", func(t *testing.T) {
		dir, migs, want := setup(t)
		afterExternal := externalCloseDuringCheck(t, dir)
		db, _, err := Open(ctx, dir, Options{Migrations: migs})
		if err != nil {
			// The version could not be established: refused, not assumed.
			if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "cannot confirm") {
				t.Fatalf("err = %v", err)
			}
			requireOnlyEmptyAuxiliaryFiles(t, dir, *afterExternal, want)
			t.Logf("refused as unconfirmed: %v", err)
			return
		}
		// The version was read from committed data: it must be the right one.
		defer func() { _ = db.Close() }()
		if v, _ := db.SchemaVersion(ctx); v != 2 || countRows(t, db, `SELECT count(*) FROM facts`) != 2 {
			t.Fatalf("opened with schema v%d and %d facts", v, countRows(t, db, `SELECT count(*) FROM facts`))
		}
		t.Log("opened after reading the committed version")
	})
}

// A version Open cannot establish is never treated as compatible: without
// any race, Open refuses and changes nothing (MVP-STO-02; PD-23 states).
func TestOpenRefusesWhenVersionIsUnknown(t *testing.T) {
	t.Run("closed by another program", func(t *testing.T) {
		dir := testutil.TempHome(t)
		db, _ := open(t, dir, Options{})
		_ = db.Close()
		closeLikeAnotherProgram(t, dir)
		before := testutil.SnapshotTree(t, dir.Path)
		openRefused(t, dir, Options{}, "cannot confirm")
		testutil.AssertTreeUnchanged(t, dir.Path, before)
	})
	t.Run("WAL without -shm", func(t *testing.T) {
		dir := testutil.TempHome(t)
		crashWithWAL(t, "CRASH_FACTS=3", "CRASH_PAYLOAD=x")
		if err := os.Remove(dir.Join(datadir.DatabaseFile) + "-shm"); err != nil {
			t.Fatal(err)
		}
		before := testutil.SnapshotTree(t, dir.Path)
		openRefused(t, dir, Options{Migrations: append(embedded(t), factsMigration(t))}, "cannot confirm")
		testutil.AssertTreeUnchanged(t, dir.Path, before)
	})
}
