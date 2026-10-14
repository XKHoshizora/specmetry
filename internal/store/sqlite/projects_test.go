package sqlite

import (
	"database/sql"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil"
)

const (
	projectA  = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"
	projectB  = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A7"
	worktreeA = "01J9Z3K5Q8R2T4V6W8X0Y2Z4B6"
	worktreeB = "01J9Z3K5Q8R2T4V6W8X0Y2Z4B7"
)

// insertProject registers a project with its main worktree, as init does.
func insertProject(tx *sql.Tx, id, commonDir, worktree, path string) error {
	seq, err := NextSeq(ctx, tx)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO projects (id, name, git_common_dir, main_worktree_id, stage, settings_json, created_at)
VALUES (?1, 'demo', ?2, ?3, 'in_development', '{}', 1);
INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES (?3, ?1, ?4, ?2, 1, ?5, 1);`, id, commonDir, worktree, path, seq)
	return err
}

func insertTarget(tx *sql.Tx, id, project, ref string, archived bool) error {
	seq, err := NextSeq(ctx, tx)
	if err != nil {
		return err
	}
	var archivedSeq any
	if archived {
		if archivedSeq, err = NextSeq(ctx, tx); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO delivery_targets (id, project_id, name, ref, created_seq, archived_seq, actor, channel)
VALUES (?, ?, 'default', ?, ?, ?, 'system', 'system')`, id, project, ref, seq, archivedSeq)
	return err
}

// insertWorktree registers a further worktree.
func insertWorktree(tx *sql.Tx, id, project, path string, gitDir any, isMain int) error {
	seq, err := NextSeq(ctx, tx)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES (?, ?, ?, ?, ?, ?, 1)`, id, project, path, gitDir, isMain, seq)
	return err
}

// data-model §4.1, §4.2 and PD-19, enforced by the schema as well as by
// init: one project per git common directory, one main worktree per
// project, worktree paths unique in a project, one active delivery target,
// branch or tag refs only, and references within one project.
func TestProjectTablesConstraints(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{})
	defer func() { _ = db.Close() }()
	ok := func(name string, fn func(tx *sql.Tx) error) {
		t.Helper()
		if err := db.WriteTx(ctx, fn); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	refused := func(name string, fn func(tx *sql.Tx) error) {
		t.Helper()
		err := db.WriteTx(ctx, fn)
		if err == nil {
			t.Errorf("%s was accepted", name)
		}
		t.Logf("%s: %v", name, err)
	}
	ok("project A", func(tx *sql.Tx) error { return insertProject(tx, projectA, "/r/a/.git", worktreeA, "/r/a") })
	ok("active target", func(tx *sql.Tx) error {
		return insertTarget(tx, "01J9Z3K5Q8R2T4V6W8X0Y2Z4C1", projectA, "refs/heads/main", false)
	})
	ok("archived target", func(tx *sql.Tx) error {
		return insertTarget(tx, "01J9Z3K5Q8R2T4V6W8X0Y2Z4C2", projectA, "refs/tags/v1", true)
	})

	refused("a second project for the same common directory", func(tx *sql.Tx) error {
		return insertProject(tx, projectB, "/r/a/.git", worktreeB, "/r/b")
	})
	refused("a second active target", func(tx *sql.Tx) error {
		return insertTarget(tx, "01J9Z3K5Q8R2T4V6W8X0Y2Z4C3", projectA, "refs/heads/dev", false)
	})
	for _, ref := range []string{"refs/remotes/origin/main", "main", "refs/heads/", "HEAD"} {
		refused("target ref "+ref, func(tx *sql.Tx) error {
			return insertTarget(tx, "01J9Z3K5Q8R2T4V6W8X0Y2Z4C4", projectA, ref, true)
		})
	}
	refused("a second main worktree", func(tx *sql.Tx) error {
		return insertWorktree(tx, "01J9Z3K5Q8R2T4V6W8X0Y2Z4D1", projectA, "/r/a2", "/r/a/.git/worktrees/x", 1)
	})
	refused("the same worktree path twice", func(tx *sql.Tx) error {
		return insertWorktree(tx, "01J9Z3K5Q8R2T4V6W8X0Y2Z4D2", projectA, "/r/a", "/r/a/.git/worktrees/y", 0)
	})
	refused("a main worktree of another project", func(tx *sql.Tx) error {
		if err := insertProject(tx, projectB, "/r/b/.git", worktreeB, "/r/b"); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE projects SET main_worktree_id = ? WHERE id = ?`, worktreeA, projectB)
		return err
	})
	refused("a worktree of a project that does not exist", func(tx *sql.Tx) error {
		return insertWorktree(tx, "01J9Z3K5Q8R2T4V6W8X0Y2Z4D3", projectB, "/r/z", nil, 0)
	})
	ok("a project outside git", func(tx *sql.Tx) error {
		if err := insertProject(tx, projectB, "", worktreeB, "/r/plain"); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
UPDATE projects SET git_common_dir = NULL WHERE id = ?1;
UPDATE worktrees SET git_dir = NULL WHERE project_id = ?1;`, projectB)
		return err
	})
	refused("a second project outside git in the same directory", func(tx *sql.Tx) error {
		if err := insertProject(tx, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A8", "", "01J9Z3K5Q8R2T4V6W8X0Y2Z4B8", "/r/plain"); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
UPDATE projects SET git_common_dir = NULL WHERE id = ?1;
UPDATE worktrees SET git_dir = NULL WHERE project_id = ?1;`, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A8")
		return err
	})
	refused("a target of another project's", func(tx *sql.Tx) error {
		return insertTarget(tx, "01J9Z3K5Q8R2T4V6W8X0Y2Z4C5", "01J9Z3K5Q8R2T4V6W8X0Y2Z4A9", "refs/heads/main", false)
	})
	if n := countRows(t, db, `SELECT count(*) FROM projects`); n != 2 {
		t.Fatalf("%d projects after the refused writes, want 2", n)
	}
}

// Tombstones remove a registered project with its worktrees and targets.
func TestTombstonesRemoveRegisteredProject(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{})
	for _, p := range []struct{ id, worktree, path string }{{projectA, worktreeA, "/r/a"}, {projectB, worktreeB, "/r/b"}} {
		if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
			if err := insertProject(tx, p.id, p.path+"/.git", p.worktree, p.path); err != nil {
				return err
			}
			return insertTarget(tx, "T"+p.worktree[1:], p.id, "refs/heads/main", false)
		}); err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Close()
	writeTombstones(t, dir, projectA)
	db, res := open(t, dir, Options{})
	defer func() { _ = db.Close() }()
	if res.TombstoneRowsGone != 3 {
		t.Fatalf("deleted %d rows, want 3 (project, worktree, target)", res.TombstoneRowsGone)
	}
	for _, table := range []string{"projects", "worktrees", "delivery_targets"} {
		if n := countRows(t, db, `SELECT count(*) FROM `+table); n != 1 {
			t.Errorf("%s has %d rows, want only project B's", table, n)
		}
	}
}
