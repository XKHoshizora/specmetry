package sqlite

import (
	"os"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// inspectUnchanged runs Inspect and requires that nothing in the data
// directory changed: no file content, size or time, and no entry created or
// removed.
func inspectUnchanged(t *testing.T, dir datadir.Dir) Inspection {
	t.Helper()
	before := testutil.SnapshotTree(t, dir.Path)
	in, err := Inspect(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	testutil.AssertTreeUnchanged(t, dir.Path, before)
	return in
}

// codex-review-004 B2: a crash left an unmerged WAL and its -shm. Inspect
// reads the committed state (schema v2 lives only in the WAL) without
// recovering, checkpointing or deleting anything.
func TestInspectLeavesCrashedWALUntouched(t *testing.T) {
	dir := testutil.TempHome(t)
	crashWithWAL(t, "CRASH_FACTS=3", "CRASH_PAYLOAD=x")
	in := inspectUnchanged(t, dir)
	if !in.Exists || in.SchemaVersion != 2 {
		t.Fatalf("inspection = %+v, want schema v2 from the WAL", in)
	}
}

// codex-review-004 B2: the same when the -shm file is gone as well.
func TestInspectLeavesWALWithoutSHMUntouched(t *testing.T) {
	dir := testutil.TempHome(t)
	crashWithWAL(t, "CRASH_FACTS=3", "CRASH_PAYLOAD=x")
	if err := os.Remove(dir.Join(datadir.DatabaseFile) + "-shm"); err != nil {
		t.Fatal(err)
	}
	in := inspectUnchanged(t, dir)
	if in.SchemaVersion != 2 {
		t.Fatalf("inspection = %+v, want schema v2 from the WAL", in)
	}
}

// codex-review-004 B2: a cleanly closed WAL database has no -wal or -shm;
// Inspect must not create them.
func TestInspectLeavesCleanDatabaseUntouched(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{})
	_ = db.Close()
	in := inspectUnchanged(t, dir)
	if in.SchemaVersion != SchemaVersion() {
		t.Fatalf("inspection = %+v", in)
	}
}

// codex-review-004 B2: while another process has the database open.
func TestInspectLeavesLiveDatabaseUntouched(t *testing.T) {
	dir := testutil.TempHome(t)
	startLiveWriter(t)
	in := inspectUnchanged(t, dir)
	if in.SchemaVersion != 2 {
		t.Fatalf("inspection = %+v, want schema v2", in)
	}
}

// codex-review-004 B2: a restored backup is a rollback-journal database.
func TestInspectLeavesRestoredBackupUntouched(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(embedded(t), factsMigration(t))
	db, _ := open(t, dir, Options{Migrations: migs[:1]})
	_ = db.Close()
	db, res := open(t, dir, Options{Migrations: migs})
	_ = db.Close()
	restoreBackup(t, dir, res.BackupPath)
	in := inspectUnchanged(t, dir)
	if in.SchemaVersion != 1 {
		t.Fatalf("inspection = %+v, want the backup's schema v1", in)
	}
}

// Inspect never creates the database.
func TestInspectMissingDatabase(t *testing.T) {
	dir := testutil.TempHome(t)
	if err := dir.Ensure(); err != nil {
		t.Fatal(err)
	}
	in := inspectUnchanged(t, dir)
	if in.Exists {
		t.Fatalf("inspection = %+v", in)
	}
}
