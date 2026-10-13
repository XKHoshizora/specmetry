package sqlite

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// inspectUnchanged runs Inspect and requires that nothing in the data
// directory changed: no file content, size or time, and no entry created or
// removed.
func inspectUnchanged(t *testing.T, dir datadir.Dir) Inspection {
	t.Helper()
	before := testutil.SettledSnapshot(t, dir.Path)
	in, err := Inspect(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	testutil.AssertTreeUnchanged(t, dir.Path, before)
	return in
}

func requireVersion(t *testing.T, in Inspection, want int) {
	t.Helper()
	if !in.Exists || in.Limitation != "" || in.SchemaVersion != want {
		t.Fatalf("inspection = %+v, want schema v%d", in, want)
	}
}

func requireUnknown(t *testing.T, in Inspection, reason string) {
	t.Helper()
	if !in.Exists || !strings.Contains(in.Limitation, reason) {
		t.Fatalf("inspection = %+v, want an unknown version because %q", in, reason)
	}
}

// closeLikeAnotherProgram opens and closes the database with a plain SQLite
// connection, which deletes the -wal and -shm files on close.
func closeLikeAnotherProgram(t *testing.T, dir datadir.Dir) {
	t.Helper()
	raw, err := sql.Open("sqlite", "file:"+escapePath(dir.Join(datadir.DatabaseFile)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `SELECT count(*) FROM meta`); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()
	if _, err := os.Stat(dir.Join(datadir.DatabaseFile) + "-wal"); !os.IsNotExist(err) {
		t.Fatalf("-wal still exists after a plain close (%v)", err)
	}
}

// A database specmetry closed keeps its -wal and -shm (persistent WAL), so
// it can be read without creating them.
func TestInspectLeavesClosedDatabaseUntouched(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{})
	_ = db.Close()
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(dir.Join(datadir.DatabaseFile) + suffix); err != nil {
			t.Fatalf("closing did not keep %s: %v", suffix, err)
		}
	}
	if fi, _ := os.Stat(dir.Join(datadir.DatabaseFile) + "-wal"); fi.Size() != 0 {
		t.Fatalf("the WAL kept %d bytes after close, want 0", fi.Size())
	}
	requireVersion(t, inspectUnchanged(t, dir), SchemaVersion())
}

// codex-review-004 B2: a crash left an unmerged WAL and its -shm. The
// committed version (schema v2 lives only in the WAL) is reported without
// recovering, checkpointing or deleting anything.
func TestInspectLeavesCrashedWALUntouched(t *testing.T) {
	dir := testutil.TempHome(t)
	crashWithWAL(t, "CRASH_FACTS=3", "CRASH_PAYLOAD=x")
	requireVersion(t, inspectUnchanged(t, dir), 2)
}

// While another process has the database open.
func TestInspectLeavesLiveDatabaseUntouched(t *testing.T) {
	dir := testutil.TempHome(t)
	startLiveWriter(t)
	requireVersion(t, inspectUnchanged(t, dir), 2)
}

// A restored VACUUM INTO backup is a rollback-journal database; with or
// without the -wal and -shm a specmetry connection left behind.
func TestInspectLeavesRestoredBackupUntouched(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(embedded(t), factsMigration(t))
	db, _ := open(t, dir, Options{Migrations: migs[:1]})
	_ = db.Close()
	db, res := open(t, dir, Options{Migrations: migs})
	_ = db.Close()
	walFiles := map[string][]byte{}
	for _, suffix := range []string{"-wal", "-shm"} {
		b, err := os.ReadFile(dir.Join(datadir.DatabaseFile) + suffix)
		if err != nil {
			t.Fatal(err)
		}
		walFiles[suffix] = b
	}
	restoreBackup(t, dir, res.BackupPath) // also removes -wal and -shm
	requireVersion(t, inspectUnchanged(t, dir), 1)
	for suffix, b := range walFiles {
		if err := os.WriteFile(dir.Join(datadir.DatabaseFile)+suffix, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	requireVersion(t, inspectUnchanged(t, dir), 1)
}

// codex-review-005 B2: when reading would create or rewrite files, the
// version is unknown, and nothing is touched.
func TestInspectReportsUnknownInsteadOfChangingFiles(t *testing.T) {
	t.Run("closed by another program", func(t *testing.T) {
		dir := testutil.TempHome(t)
		db, _ := open(t, dir, Options{})
		_ = db.Close()
		closeLikeAnotherProgram(t, dir)
		requireUnknown(t, inspectUnchanged(t, dir), "-wal and -shm files are missing")
	})
	t.Run("WAL without -shm", func(t *testing.T) {
		dir := testutil.TempHome(t)
		crashWithWAL(t, "CRASH_FACTS=3", "CRASH_PAYLOAD=x")
		if err := os.Remove(dir.Join(datadir.DatabaseFile) + "-shm"); err != nil {
			t.Fatal(err)
		}
		requireUnknown(t, inspectUnchanged(t, dir), "only one of")
	})
	t.Run("rollback journal", func(t *testing.T) {
		dir := testutil.TempHome(t)
		db, _ := open(t, dir, Options{})
		_ = db.Close()
		journal := append([]byte{0xd9, 0xd5, 0x05, 0xf9, 0x20, 0xa1, 0x63, 0xd7}, make([]byte, 504)...)
		if err := os.WriteFile(dir.Join(datadir.DatabaseFile)+"-journal", journal, 0o600); err != nil {
			t.Fatal(err)
		}
		requireUnknown(t, inspectUnchanged(t, dir), "rollback journal")
		before := testutil.SnapshotTree(t, dir.Path)
		openRefused(t, dir, Options{}, "rollback journal")
		testutil.AssertTreeUnchanged(t, dir.Path, before)
	})
}

// Inspect never creates the database.
func TestInspectMissingDatabase(t *testing.T) {
	dir := testutil.TempHome(t)
	if err := dir.Ensure(); err != nil {
		t.Fatal(err)
	}
	if in := inspectUnchanged(t, dir); in.Exists {
		t.Fatalf("inspection = %+v", in)
	}
}

// The read-only check does not stand in the way of normal crash recovery: a
// compatible binary opens a crashed database and sees every committed fact.
func TestCompatibleBinaryRecoversCrashedDatabase(t *testing.T) {
	dir := testutil.TempHome(t)
	crashWithWAL(t, "CRASH_FACTS=4", "CRASH_PAYLOAD=x")
	db, _ := open(t, dir, Options{Migrations: append(embedded(t), factsMigration(t))})
	if n := countRows(t, db, `SELECT count(*) FROM facts`); n != 4 {
		t.Fatalf("%d facts after recovery, want 4", n)
	}
}

// Characterization of the driver in use (codex-review-005 B2): what a
// read-only SQLite connection (mode=ro, readonly_shm=1) does to the files in
// each state. readonly.go reads only in the states listed as unchanged here
// and reports the version as unknown in the others. If this test fails, the
// driver changed and readonly.go must be revisited.
func TestReadOnlyConnectionByFileState(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(t *testing.T, dir datadir.Dir)
		unchanged bool
	}{
		{"closed by specmetry (persistent WAL)", func(t *testing.T, dir datadir.Dir) {
			db, _ := open(t, dir, Options{})
			_ = db.Close()
		}, true},
		{"crashed, -wal and -shm left", func(t *testing.T, dir datadir.Dir) {
			crashWithWAL(t, "CRASH_FACTS=3", "CRASH_PAYLOAD=x")
		}, true},
		{"rollback journal mode, no -wal or -shm", func(t *testing.T, dir datadir.Dir) {
			migs := append(embedded(t), factsMigration(t))
			db, _ := open(t, dir, Options{Migrations: migs[:1]})
			_ = db.Close()
			db, res := open(t, dir, Options{Migrations: migs})
			_ = db.Close()
			restoreBackup(t, dir, res.BackupPath)
		}, true},
		{"closed by another program", func(t *testing.T, dir datadir.Dir) {
			db, _ := open(t, dir, Options{})
			_ = db.Close()
			closeLikeAnotherProgram(t, dir)
		}, false},
		{"crashed, -shm deleted", func(t *testing.T, dir datadir.Dir) {
			crashWithWAL(t, "CRASH_FACTS=3", "CRASH_PAYLOAD=x")
			if err := os.Remove(dir.Join(datadir.DatabaseFile) + "-shm"); err != nil {
				t.Fatal(err)
			}
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := testutil.TempHome(t)
			c.setup(t, dir)
			before := testutil.SettledSnapshot(t, dir.Path)
			sdb, err := sql.Open("sqlite", "file:"+escapePath(dir.Join(datadir.DatabaseFile))+"?mode=ro&readonly_shm=1")
			if err != nil {
				t.Fatal(err)
			}
			var v int
			readErr := sdb.QueryRowContext(ctx, `SELECT schema_version FROM meta`).Scan(&v)
			_ = sdb.Close()
			diff := testutil.DiffSnapshots(before, testutil.SnapshotTree(t, dir.Path))
			if c.unchanged && (diff != "" || readErr != nil) {
				t.Fatalf("expected a clean read; err=%v, changes:\n%s", readErr, diff)
			}
			if !c.unchanged && diff == "" && readErr == nil {
				t.Fatal("the read neither failed nor changed anything; this state could be read directly")
			}
			t.Logf("err=%v changes=%q", readErr, diff)
		})
	}
}
