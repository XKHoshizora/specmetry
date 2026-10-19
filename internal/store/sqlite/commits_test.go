package sqlite

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// commits, commit_links and git_scan_cursors (0007, data-model §4.12, WP2.3
// spec §7): a commit is stored once per project, with its links, by the
// scan that first finds it; commits and links are append-only; a link
// names a requirement or criterion of the commit's own project; the
// cursors are the scan's own, written only by it; all three go with their
// project when it is purged.
func TestCommitTablesConstraints(t *testing.T) {
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
	const target = "commit_links target must be a requirement or criterion of the link's project"
	const (
		commitA = "01J9Z3K5Q8R2T4V6W8X0Y2Z4E1"
		commitB = "01J9Z3K5Q8R2T4V6W8X0Y2Z4E2"
		commitC = "01J9Z3K5Q8R2T4V6W8X0Y2Z4E3"
		reqB    = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N9"
	)
	sha := func(c string) string { return strings.Repeat(c, 40) }
	ok("projects, a version, requirements and a criterion", func(tx *sql.Tx) error {
		for _, f := range []func() error{
			func() error { return insertProject(tx, projectA, "/r/a/.git", worktreeA, "/r/a") },
			func() error { return insertProject(tx, projectB, "/r/b/.git", worktreeB, "/r/b") },
			func() error { return insertVersion(tx, versionA, 1, "human:me") },
			func() error { return insertRequirement(tx, reqA, "REQ-001", nil) },
			func() error { return insertCriterion(tx, criterion, "AC-001-1", reqA, "manual", nil) },
			func() error {
				return stmt(`INSERT INTO requirements (id, project_id, key, status, revision, revision_seq, origin, created_version,
    updated_version, updated_seq, title, kind, priority)
VALUES (?, ?, 'REQ-001', 'active', 1, 1, 'extracted', 1, 1, 1, 'log in', 'functional', 'must')`, reqB, projectB)(tx)
			},
		} {
			if err := f(); err != nil {
				return err
			}
		}
		return nil
	})
	commit := func(id, project, sha string, seq int64) func(tx *sql.Tx) error {
		return stmt(`INSERT INTO commits (id, seq, project_id, sha, committed_at) VALUES (?, ?, ?, ?, 1700000000000)`, id, seq, project, sha)
	}
	link := func(commitID, project, targetType, targetID, source string) func(tx *sql.Tx) error {
		return stmt(`INSERT INTO commit_links (commit_id, project_id, target_type, target_id, link_source) VALUES (?, ?, ?, ?, ?)`,
			commitID, project, targetType, targetID, source)
	}
	cursor := func(kind, name, tip string, scannedSeq any) func(tx *sql.Tx) error {
		return stmt(`INSERT INTO git_scan_cursors (project_id, kind, name, last_tip, scanned_seq, scanned_at) VALUES (?, ?, ?, ?, ?, 1)`,
			projectA, kind, name, tip, scannedSeq)
	}
	ok("a commit with a trailer link and a message link", func(tx *sql.Tx) error {
		for _, f := range []func(tx *sql.Tx) error{
			commit(commitA, projectA, sha("a"), 50),
			link(commitA, projectA, "requirement", reqA, "trailer"),
			link(commitA, projectA, "criterion", criterion, "message"),
		} {
			if err := f(tx); err != nil {
				return err
			}
		}
		return nil
	})
	ok("another commit of the same scan shares its seq", commit(commitB, projectA, sha("b"), 50))
	ok("the same SHA in another project", commit(commitC, projectB, sha("a"), 51))
	ok("a branch cursor", cursor("branch", "refs/heads/main", sha("a"), 50))
	ok("a detached HEAD's cursor, never with new commits", cursor("detached", "/r/a-wt", sha("b"), nil))
	ok("a cursor moves", stmt(`UPDATE git_scan_cursors SET last_tip = ?, scanned_at = 2 WHERE kind = 'branch'`, sha("c")))
	var ids rowIDs
	for _, c := range []struct {
		name, why string
		fn        func(tx *sql.Tx) error
	}{
		{"a short ID", check, commit("01J9Z3K5Q8R2T4V6W8X0Y2Z4E", projectA, sha("c"), 52)},
		{"an upper-case SHA", check, commit(ids.next(), projectA, sha("C"), 52)},
		{"a short SHA", check, commit(ids.next(), projectA, sha("c")[1:], 52)},
		{"a SHA that is not hexadecimal", check, commit(ids.next(), projectA, sha("g"), 52)},
		{"a SHA-256 commit ID", check, commit(ids.next(), projectA, strings.Repeat("c", 64), 52)},
		{"the same SHA twice in a project", unique, commit(ids.next(), projectA, sha("a"), 52)},
		{"no seq", "NOT NULL constraint failed", stmt(`INSERT INTO commits (id, project_id, sha, committed_at) VALUES (?, ?, ?, 1)`,
			ids.next(), projectA, sha("d"))},
		{"no project", fk, commit(ids.next(), ids.next(), sha("d"), 52)},
		{"another project's requirement", target, link(commitB, projectA, "requirement", reqB, "message")},
		{"a criterion taken for a requirement", target, link(commitB, projectA, "requirement", criterion, "message")},
		{"a requirement taken for a criterion", target, link(commitB, projectA, "criterion", reqA, "message")},
		{"a goal", target, link(commitB, projectA, "goal", reqA, "message")},
		{"a target that does not exist", target, link(commitB, projectA, "requirement", ids.next(), "message")},
		{"another source", check, link(commitB, projectA, "requirement", reqA, "body")},
		{"the same target twice", unique, link(commitA, projectA, "requirement", reqA, "message")},
		{"a commit of another project", fk, link(commitC, projectA, "requirement", reqA, "message")},
		{"a commit that does not exist", fk, link(ids.next(), projectA, "requirement", reqA, "message")},
		{"changing a commit", "append-only table commits", stmt(`UPDATE commits SET committed_at = 2`)},
		{"deleting a commit", "append-only table commits", stmt(`DELETE FROM commits WHERE id = ?`, commitB)},
		{"replacing a commit", "append-only table commits", stmt(`INSERT OR REPLACE INTO commits (id, seq, project_id, sha, committed_at)
VALUES (?, 60, ?, ?, 1)`, commitB, projectA, sha("b"))},
		{"changing a link", "append-only table commit_links", stmt(`UPDATE commit_links SET link_source = 'message'`)},
		{"deleting a link", "append-only table commit_links", stmt(`DELETE FROM commit_links`)},
		{"replacing a link", "append-only table commit_links", stmt(`INSERT OR REPLACE INTO commit_links (commit_id, project_id, target_type, target_id, link_source)
VALUES (?, ?, 'requirement', ?, 'message')`, commitA, projectA, reqA)},
		{"a cursor of another kind", check, cursor("tag", "refs/tags/v1", sha("a"), nil)},
		{"a cursor without a name", check, cursor("branch", "", sha("a"), nil)},
		{"a cursor at a short tip", check, cursor("branch", "refs/heads/x", sha("a")[1:], nil)},
		{"a cursor at an upper-case tip", check, cursor("branch", "refs/heads/x", sha("A"), nil)},
		{"the same cursor twice", unique, cursor("branch", "refs/heads/main", sha("d"), nil)},
		{"a cursor of no project", fk, stmt(`INSERT INTO git_scan_cursors (project_id, kind, name, last_tip, scanned_at) VALUES (?, 'branch', 'refs/heads/main', ?, 1)`,
			ids.next(), sha("a"))},
		{"a commit removed while its links stay", fk, func(tx *sql.Tx) error {
			return Maintenance(ctx, tx, func() error { return stmt(`DELETE FROM commits WHERE id = ?`, commitA)(tx) })
		}},
	} {
		refused(c.name, c.why, c.fn)
	}
	ok("maintenance removes links, then their commit", func(tx *sql.Tx) error {
		return Maintenance(ctx, tx, func() error {
			if err := link(commitB, projectA, "requirement", reqA, "message")(tx); err != nil {
				return err
			}
			if err := stmt(`DELETE FROM commit_links WHERE commit_id = ?`, commitB)(tx); err != nil {
				return err
			}
			return stmt(`DELETE FROM commits WHERE id = ?`, commitB)(tx)
		})
	})
	for _, index := range []string{"commits_by_seq", "commit_links_by_target"} {
		if n := countRows(t, db, `SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, index); n != 1 {
			t.Errorf("index %s: %d", index, n)
		}
	}

	_ = db.Close()
	writeTombstones(t, dir, projectA)
	db, _ = open(t, dir, Options{})
	for _, table := range []string{"commits", "commit_links", "git_scan_cursors"} {
		if n := countRows(t, db, `SELECT count(*) FROM `+table+` WHERE project_id = '`+projectA+`'`); n != 0 {
			t.Errorf("%s has %d rows of the purged project", table, n)
		}
	}
	if n := countRows(t, db, `SELECT count(*) FROM commits WHERE project_id = '`+projectB+`'`); n != 1 {
		t.Errorf("project B has %d commits after A was purged, want 1", n)
	}
}
