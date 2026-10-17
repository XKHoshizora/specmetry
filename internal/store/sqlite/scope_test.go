package sqlite

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// scope_confirmations and scope_items (0006, data-model §4.2): made by a
// person only, bound to a target and a requirement of the same project,
// with complete_scope true and a count of excluded items; append-only;
// removed by purge with their project.
func TestScopeTablesConstraints(t *testing.T) {
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
	const (
		targetA = "01J9Z3K5Q8R2T4V6W8X0Y2Z4C1"
		targetB = "01J9Z3K5Q8R2T4V6W8X0Y2Z4C2"
		first   = "01J9Z3K5Q8R2T4V6W8X0Y2Z4D1"
		second  = "01J9Z3K5Q8R2T4V6W8X0Y2Z4D2"
	)
	hash := strings.Repeat("ab", 32)
	var ids rowIDs
	ok("projects, targets, a version and a requirement", func(tx *sql.Tx) error {
		for _, f := range []func() error{
			func() error { return insertProject(tx, projectA, "/r/a/.git", worktreeA, "/r/a") },
			func() error { return insertProject(tx, projectB, "/r/b/.git", worktreeB, "/r/b") },
			func() error { return insertTarget(tx, targetA, projectA, "refs/heads/main", false) },
			func() error { return insertTarget(tx, targetB, projectB, "refs/heads/main", false) },
			func() error { return insertVersion(tx, versionA, 1, "human:me") },
			func() error { return insertRequirement(tx, reqA, "REQ-001", nil) },
		} {
			if err := f(); err != nil {
				return err
			}
		}
		return nil
	})
	seq := int64(100)
	confirmation := func(id, project, target string, version int, attestations, actor, channel string, seen any) func(tx *sql.Tx) error {
		seq++
		return stmt(`INSERT INTO scope_confirmations (id, seq, project_id, target_id, baseline_version, source_set_hash, attestations_json,
    acknowledged_waivers_json, seen_scope_confirmation_id, actor, channel, confirmed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, '[]', ?, ?, ?, 1)`, id, seq, project, target, version, hash, attestations, seen, actor, channel)
	}
	const attested = `{"complete_scope":true,"pending_items_excluded":0}`
	ok("a confirmation", confirmation(first, projectA, targetA, 1, attested, "human:me", "human_cli", nil))
	ok("its requirement", stmt(`INSERT INTO scope_items (confirmation_id, project_id, requirement_id, requirement_revision) VALUES (?, ?, ?, 1)`,
		first, projectA, reqA))
	ok("the next, naming the first", confirmation(second, projectA, targetA, 1,
		`{"complete_scope":true,"pending_items_excluded":2,"missing_goals_reviewed":true}`, "human:me", "dashboard", first))
	for _, c := range []struct {
		name, why string
		fn        func(tx *sql.Tx) error
	}{
		{"an incomplete scope", check, confirmation(ids.next(), projectA, targetA, 1, `{"complete_scope":false,"pending_items_excluded":0}`, "human:me", "human_cli", nil)},
		{"no complete_scope", check, confirmation(ids.next(), projectA, targetA, 1, `{"pending_items_excluded":0}`, "human:me", "human_cli", nil)},
		{"a negative count", check, confirmation(ids.next(), projectA, targetA, 1, `{"complete_scope":true,"pending_items_excluded":-1}`, "human:me", "human_cli", nil)},
		{"a count that is not a number", check, confirmation(ids.next(), projectA, targetA, 1, `{"complete_scope":true,"pending_items_excluded":"0"}`, "human:me", "human_cli", nil)},
		{"missing goals not a bool", check, confirmation(ids.next(), projectA, targetA, 1, `{"complete_scope":true,"pending_items_excluded":0,"missing_goals_reviewed":1}`, "human:me", "human_cli", nil)},
		{"before any baseline", check, confirmation(ids.next(), projectA, targetA, 0, attested, "human:me", "human_cli", nil)},
		{"by an Agent", check, confirmation(ids.next(), projectA, targetA, 1, attested, "agent:claude_code", "agent_cli", nil)},
		{"on the Agent channel", check, confirmation(ids.next(), projectA, targetA, 1, attested, "human:me", "agent_cli", nil)},
		{"another project's target", fk, confirmation(ids.next(), projectA, targetB, 1, attested, "human:me", "human_cli", nil)},
		{"a target that does not exist", fk, confirmation(ids.next(), projectA, ids.next(), 1, attested, "human:me", "human_cli", nil)},
		{"naming a confirmation that does not exist", fk, confirmation(ids.next(), projectA, targetA, 1, attested, "human:me", "human_cli", ids.next())},
		{"another project's requirement", fk, stmt(`INSERT INTO scope_items (confirmation_id, project_id, requirement_id, requirement_revision) VALUES (?, ?, ?, 1)`,
			second, projectB, reqA)},
		{"a requirement twice", unique, stmt(`INSERT INTO scope_items (confirmation_id, project_id, requirement_id, requirement_revision) VALUES (?, ?, ?, 1)`,
			first, projectA, reqA)},
		{"revision 0", check, stmt(`INSERT INTO scope_items (confirmation_id, project_id, requirement_id, requirement_revision) VALUES (?, ?, ?, 0)`,
			second, projectA, reqA)},
		{"the same seq twice", unique, stmt(`INSERT INTO scope_confirmations (id, seq, project_id, target_id, baseline_version, source_set_hash,
    attestations_json, acknowledged_waivers_json, actor, channel, confirmed_at) VALUES (?, 101, ?, ?, 1, ?, ?, '[]', 'human:me', 'human_cli', 1)`,
			ids.next(), projectA, targetA, hash, attested)},
		{"changing a confirmation", "append-only table scope_confirmations", stmt(`UPDATE scope_confirmations SET note = 'x'`)},
		{"deleting a confirmation", "append-only table scope_confirmations", stmt(`DELETE FROM scope_confirmations`)},
		{"changing a scope item", "append-only table scope_items", stmt(`UPDATE scope_items SET requirement_revision = 2`)},
		{"deleting a scope item", "append-only table scope_items", stmt(`DELETE FROM scope_items`)},
	} {
		refused(c.name, c.why, c.fn)
	}

	_ = db.Close()
	writeTombstones(t, dir, projectA)
	db, _ = open(t, dir, Options{})
	for _, table := range []string{"scope_confirmations", "scope_items"} {
		if n := countRows(t, db, `SELECT count(*) FROM `+table+` WHERE project_id = '`+projectA+`'`); n != 0 {
			t.Errorf("%s has %d rows of the purged project", table, n)
		}
	}
}
