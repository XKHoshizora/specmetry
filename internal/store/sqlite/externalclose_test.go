package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"runtime"
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
	db, _ := open(t, dir, Options{Migrations: append(foundation(t), factsMigration(t))})
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
		migs := append(foundation(t), factsMigration(t))
		db, _ := open(t, dir, Options{Migrations: migs})
		insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "committed")
		insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "committed")
		_ = db.Close()
		return dir, migs, databaseFacts(t, dir.Join(datadir.DatabaseFile))
	}

	t.Run("incompatible database is still refused", func(t *testing.T) {
		dir, _, want := setup(t)
		afterExternal := externalCloseDuringCheck(t, dir)
		db, _, err := Open(ctx, dir, Options{Migrations: foundation(t)}) // knows schema v1 only
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
		// The version was read from committed data: it must be the right one,
		// and opening must not have changed the data, the permissions or left
		// the lock held (codex-review-007 non-blocker).
		defer func() { _ = db.Close() }()
		if v, _ := db.SchemaVersion(ctx); v != 2 || countRows(t, db, `SELECT count(*) FROM facts`) != 2 {
			t.Fatalf("opened with schema v%d and %d facts", v, countRows(t, db, `SELECT count(*) FROM facts`))
		}
		dbFile := dir.Join(datadir.DatabaseFile)
		if got := databaseFacts(t, dbFile); got != want {
			t.Fatalf("schema/min_binary_version/seq/facts %s after opening, want %s", got, want)
		}
		if (*afterExternal)[datadir.DatabaseFile] != testutil.SnapshotTree(t, dir.Path)[datadir.DatabaseFile] {
			t.Fatal("opening changed the main database file before anything was written")
		}
		if problems, err := datadir.Check(dir); err != nil || len(problems) != 0 {
			t.Fatalf("permissions: %v %v", problems, err)
		}
		if held, err := lockHeld(dir.Join(datadir.MigrateLockFile)); err != nil || held {
			t.Fatalf("open lock still held after Open returned (%v)", err)
		}
		if seq := insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "after"); seq != 3 {
			t.Fatalf("the next write got seq %d, want 3", seq)
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
		openRefused(t, dir, Options{Migrations: append(foundation(t), factsMigration(t))}, "cannot confirm")
		testutil.AssertTreeUnchanged(t, dir.Path, before)
	})
}

// armedClose records whether an armed outside close actually happened.
type armedClose struct {
	happened      bool
	afterExternal map[string]string
}

// armExternalClose arranges that, if the next read-only check gets as far as
// opening its connection, an independent process first closes the database
// with an ordinary SQLite connection (removing -wal and -shm). If the check
// decides "unknown" before reading, nothing races and nothing may change.
func armExternalClose(t *testing.T, dir datadir.Dir) *armedClose {
	t.Helper()
	a := &armedClose{}
	dbFile := dir.Join(datadir.DatabaseFile)
	readOnlyHook = func() {
		readOnlyHook = nil
		if out, err := testutil.HelperCommand(t, "plainclose").CombinedOutput(); err != nil {
			t.Errorf("external close: %v\n%s", err, out)
		}
		for _, suffix := range []string{"-wal", "-shm"} {
			if _, err := os.Stat(dbFile + suffix); !os.IsNotExist(err) {
				t.Errorf("the external close left %s (%v)", suffix, err)
			}
		}
		a.happened = true
		a.afterExternal = testutil.SettledSnapshot(t, dir.Path)
	}
	t.Cleanup(func() { readOnlyHook = nil })
	return a
}

// codex-review-007 B2: the external-close race twice in a row. On Windows the
// first check leaves an empty -wal and an empty -shm; in the second race the
// outside close removes them and SQLite recreates them with the same names
// and sizes, which a comparison of existence and sizes cannot see. Every
// cycle must report the version as unknown; Specmetry may add only empty,
// private -wal/-shm files, and only when the race happened; the data stays
// intact; and once specmetry has opened the database normally, an
// uncontended check reports the version again.
func TestInspectRepeatedExternalCloseReportsUnknown(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(foundation(t), factsMigration(t))
	db, _ := open(t, dir, Options{Migrations: migs})
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "committed")
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "committed")
	_ = db.Close()
	dbFile := dir.Join(datadir.DatabaseFile)
	wantFacts := databaseFacts(t, dbFile)

	for cycle := 0; cycle < 2; cycle++ {
		before := testutil.SettledSnapshot(t, dir.Path)
		race := armExternalClose(t, dir)
		in, err := Inspect(ctx, dir)
		readOnlyHook = nil
		if err != nil {
			t.Fatal(err)
		}
		if in.Limitation == "" || in.SchemaVersion != 0 || in.MinBinaryVersion != "" {
			t.Fatalf("cycle %d (outside close during the check: %v): reported %+v, want the version unknown", cycle, race.happened, in)
		}
		after := testutil.SnapshotTree(t, dir.Path)
		if race.happened {
			assertOnlyCreated(t, dir, race.afterExternal, after, datadir.DatabaseFile+"-wal", datadir.DatabaseFile+"-shm")
		} else {
			testutil.AssertTreeUnchanged(t, dir.Path, before)
		}
		if problems, err := datadir.Check(dir); err != nil || len(problems) != 0 {
			t.Fatalf("cycle %d: permissions: %v %v", cycle, problems, err)
		}
		if got := databaseFacts(t, dbFile); got != wantFacts {
			t.Fatalf("cycle %d: database content %s, want %s", cycle, got, wantFacts)
		}
		t.Logf("cycle %d: outside close during the check: %v; %s", cycle, race.happened, in.Limitation)
		if cycle == 0 {
			if !race.happened {
				t.Fatal("the first cycle did not reproduce the race")
			}
			if runtime.GOOS == "windows" {
				// The state of the reported counterexample, reached naturally.
				for _, suffix := range []string{"-wal", "-shm"} {
					if fi, err := os.Stat(dbFile + suffix); err != nil || fi.Size() != 0 {
						t.Fatalf("after the first race %s is %v (%v), want an empty file", suffix, fi, err)
					}
				}
			}
		}
	}

	// A normal open rebuilds the shared-memory index; afterwards the
	// uncontended check reports the version and changes nothing. On Linux
	// the empty -wal left alone keeps the database in a PD-23 state, which
	// Open refuses.
	db, _, err := Open(ctx, dir, Options{Migrations: migs})
	if err != nil {
		if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "only one of") {
			t.Fatalf("reopening: %v", err)
		}
		t.Logf("reopening refused: %v", err)
		return
	}
	_ = db.Close()
	requireVersion(t, inspectUnchanged(t, dir), 2)
	if got := databaseFacts(t, dbFile); got != wantFacts {
		t.Fatalf("database content %s, want %s", got, wantFacts)
	}
}
