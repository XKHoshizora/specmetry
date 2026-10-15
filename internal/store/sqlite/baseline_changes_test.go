package sqlite

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// baseline_changes (0005, data-model §4.7): append-only; before only for
// changes other than add; a class exactly for updates; bound to a version
// of its project and ordered within it; removed by purge with its project.
func TestBaselineChangesConstraints(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{})
	defer func() { _ = db.Close() }()
	ok := func(name string, fn func(tx *sql.Tx) error) {
		t.Helper()
		if err := db.WriteTx(ctx, fn); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	refused := func(name, why string, fn func(tx *sql.Tx) error) {
		t.Helper()
		err := db.WriteTx(ctx, fn)
		switch {
		case err == nil:
			t.Errorf("%s was accepted", name)
		case !strings.Contains(err.Error(), why):
			t.Errorf("%s was refused for another reason: %v (want %q)", name, err, why)
		}
	}
	const check, unique, fk = "CHECK constraint failed", "UNIQUE constraint failed", "FOREIGN KEY constraint failed"
	var ids rowIDs
	ok("projects, a version and a goal", func(tx *sql.Tx) error {
		if err := insertProject(tx, projectA, "/r/a/.git", worktreeA, "/r/a"); err != nil {
			return err
		}
		if err := insertProject(tx, projectB, "/r/b/.git", worktreeB, "/r/b"); err != nil {
			return err
		}
		if err := insertVersion(tx, versionA, 1, "human:me"); err != nil {
			return err
		}
		return insertGoal(tx, goalA, "GOAL-1")
	})
	change := func(project, version string, index int, op string, class, before any) func(tx *sql.Tx) error {
		return stmt(`INSERT INTO baseline_changes (id, seq, project_id, version_id, change_index, entity_type, entity_id, op, change_class,
    before_json, after_json) VALUES (?, 1, ?, ?, ?, 'goal', ?, ?, ?, ?, '{"revision":1}')`, ids.next(), project, version, index, goalA, op, class, before)
	}
	ok("an add", change(projectA, versionA, 0, "add", nil, nil))
	ok("an update", change(projectA, versionA, 1, "update", "editorial", `{"revision":1}`))
	refused("an add with a before", check, change(projectA, versionA, 2, "add", nil, `{}`))
	refused("an update without before", check, change(projectA, versionA, 2, "update", "material", nil))
	refused("an update without a class", check, change(projectA, versionA, 2, "update", nil, `{}`))
	refused("a retire with a class", check, change(projectA, versionA, 2, "retire", "material", `{}`))
	refused("the same index twice", unique, change(projectA, versionA, 1, "retire", nil, `{}`))
	refused("another project's version", fk, change(projectB, versionA, 2, "retire", nil, `{}`))
	refused("a version that does not exist", fk, change(projectA, ids.next(), 0, "retire", nil, `{}`))
	refused("changing a change", "append-only table baseline_changes", stmt(`UPDATE baseline_changes SET change_index = 9`))
	refused("deleting a change", "append-only table baseline_changes", stmt(`DELETE FROM baseline_changes`))

	_ = db.Close()
	writeTombstones(t, dir, projectA)
	db, _ = open(t, dir, Options{})
	if n := countRows(t, db, `SELECT count(*) FROM baseline_changes`); n != 0 {
		t.Errorf("baseline_changes has %d rows of the purged project", n)
	}
}
