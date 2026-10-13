package sqlite

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

// rollbackModeCopy writes a VACUUM INTO copy of the database in src — a
// rollback-journal database, like a restored backup — to a new file.
func rollbackModeCopy(t *testing.T, src datadir.Dir) string {
	t.Helper()
	out := filepath.Join(testdir.New(t), "copy.db")
	raw, err := sql.Open("sqlite", "file:"+escapePath(src.Join(datadir.DatabaseFile)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	if _, err := raw.ExecContext(ctx, `VACUUM INTO ?`, out); err != nil {
		t.Fatal(err)
	}
	return out
}

// installDatabase puts file in a new, private data directory as its database,
// the way a user restores a backup into an empty data directory: there is no
// -wal, -shm or lock file.
func installDatabase(t *testing.T, home datadir.Dir, file string) {
	t.Helper()
	if err := home.Ensure(); err != nil {
		t.Fatal(err)
	}
	src, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = src.Close() }()
	dst, err := datadir.CreatePrivateFile(home.Join(datadir.DatabaseFile), true)
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

// databaseFacts reads the meta row and the number of facts from a copy of the
// main database file, so the original is not opened at all.
func databaseFacts(t *testing.T, dbFile string) string {
	t.Helper()
	b, err := os.ReadFile(dbFile)
	if err != nil {
		t.Fatal(err)
	}
	cp := filepath.Join(testdir.New(t), "check.db")
	if err := os.WriteFile(cp, b, 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite", "file:"+escapePath(cp))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	var s string
	q := `SELECT schema_version || '/' || min_binary_version || '/' || seq_counter FROM meta`
	if err := raw.QueryRowContext(ctx, q).Scan(&s); err != nil {
		t.Fatal(err)
	}
	var hasFacts, facts int
	if err := raw.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE name = 'facts'`).Scan(&hasFacts); err != nil {
		t.Fatal(err)
	}
	if hasFacts > 0 {
		if err := raw.QueryRowContext(ctx, `SELECT count(*) FROM facts`).Scan(&facts); err != nil {
			t.Fatal(err)
		}
	}
	return s + "/" + strconv.Itoa(facts)
}

// assertOnlyCreated requires that the only difference between before and
// after is the creation of the named entries — each empty — and the data
// directory's own modification time, which creating an entry updates.
func assertOnlyCreated(t *testing.T, dir datadir.Dir, before, after map[string]string, allowed ...string) {
	t.Helper()
	ok := map[string]bool{}
	for _, a := range allowed {
		ok["created "+a] = true
	}
	for _, line := range strings.Split(testutil.DiffSnapshots(before, after), "\n") {
		switch {
		case line == "", strings.HasPrefix(line, "changed .:"):
		case ok[line]:
			name := strings.TrimPrefix(line, "created ")
			if fi, err := os.Stat(dir.Join(name)); err != nil || fi.Size() != 0 {
				t.Fatalf("%s is not an empty file (%v)", name, err)
			}
		default:
			t.Fatalf("change outside the allowed side effects (%v): %s", allowed, line)
		}
	}
}

// MVP-STO-02 (adjusted 2026-10-13, codex-review-006 R1): refusing an
// incompatible database may create the coordination file .migrate.lock —
// empty, private, not left held — and nothing else; the schema, business
// data, facts, seq and every existing database file stay as they were.
func TestIncompatibleRefusalMayOnlyCreateCoordinationFile(t *testing.T) {
	requires := Migration{Version: 2, Name: "requires_new", SQL: `UPDATE meta SET min_binary_version = '9.9.9' WHERE id = 1;`}
	cases := []struct {
		name   string
		source func(t *testing.T) datadir.Dir
		binary func(t *testing.T) Options // the binary that must refuse
		want   string
	}{
		{"newer schema", func(t *testing.T) datadir.Dir {
			src := datadir.Dir{Path: filepath.Join(testdir.New(t), "src")}
			db, _ := open(t, src, Options{Migrations: append(embedded(t), factsMigration(t))})
			insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "kept")
			insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "kept")
			_ = db.Close()
			return src
		}, func(t *testing.T) Options { return Options{Migrations: embedded(t)} }, "upgrade specmetry"},
		{"min_binary_version", func(t *testing.T) datadir.Dir {
			src := datadir.Dir{Path: filepath.Join(testdir.New(t), "src")}
			db, _ := open(t, src, Options{Migrations: append(embedded(t), requires), BinaryVersion: "9.9.9"})
			_ = db.Close()
			return src
		}, func(t *testing.T) Options {
			// Knows the same schema, but is older than min_binary_version.
			return Options{Migrations: append(embedded(t), requires), BinaryVersion: "0.1.0"}
		}, "9.9.9"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := c.source(t)
			home := testutil.TempHome(t)
			installDatabase(t, home, rollbackModeCopy(t, src))
			dbFile := home.Join(datadir.DatabaseFile)
			wantFacts := databaseFacts(t, dbFile)
			if _, err := os.Stat(home.Join(datadir.MigrateLockFile)); !os.IsNotExist(err) {
				t.Fatalf("the data directory must start without a lock file (%v)", err)
			}
			opts := c.binary(t)
			before := testutil.SnapshotTree(t, home.Path)
			openRefused(t, home, opts, c.want)
			after := testutil.SnapshotTree(t, home.Path)

			// Allowed: the coordination file, empty, private and not held.
			assertOnlyCreated(t, home, before, after, datadir.MigrateLockFile)
			if problems, err := datadir.Check(home); err != nil || len(problems) != 0 {
				t.Fatalf("permissions after the refusal: %v %v", problems, err)
			}
			if held, err := lockHeld(home.Join(datadir.MigrateLockFile)); err != nil || held {
				t.Fatalf("lock still held after the refusal (%v)", err)
			}
			// Forbidden: any change to the schema, data, facts or seq.
			if got := databaseFacts(t, dbFile); got != wantFacts {
				t.Fatalf("database content %s, want %s", got, wantFacts)
			}
			// Refusing again, now with the lock file present, changes nothing.
			again := testutil.SnapshotTree(t, home.Path)
			openRefused(t, home, opts, c.want)
			testutil.AssertTreeUnchanged(t, home.Path, again)
		})
	}
}
