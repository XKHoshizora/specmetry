package sqlite

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// claim_batches and claims (0008, data-model §4.10, WP2.4 spec §8, §15):
// a batch and all its claims share one seq; a manual batch only comes from
// human_cli (E2); a claim names an active criterion of its own project at
// its current revision, or is unmapped work with no criterion, revision
// or assertion; unknown only in an audit batch; at most one claim per
// criterion in a batch; both tables are append-only, REPLACE and upserts
// included; both go with their project when it is purged.
func TestClaimTablesConstraints(t *testing.T) {
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
		shared   = "a claim belongs to a batch of its project and shares its seq"
		target   = "a claim names an active criterion of its project at its current revision"
		unknown  = "an unknown assertion is only made in an audit batch"
		batchSyn = "01J9Z3K5Q8R2T4V6W8X0Y2Z4P1"
		batchAud = "01J9Z3K5Q8R2T4V6W8X0Y2Z4P2"
		batchMan = "01J9Z3K5Q8R2T4V6W8X0Y2Z4P3"
		batchB   = "01J9Z3K5Q8R2T4V6W8X0Y2Z4P4"
		claimA   = "01J9Z3K5Q8R2T4V6W8X0Y2Z4Q1"
		retired  = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N7"
		reqB     = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N8"
		critB    = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N9"
	)
	sha := func(c string) string { return strings.Repeat(c, 40) }
	ok("projects, a version, requirements and criteria", func(tx *sql.Tx) error {
		for _, f := range []func() error{
			func() error { return insertProject(tx, projectA, "/r/a/.git", worktreeA, "/r/a") },
			func() error { return insertProject(tx, projectB, "/r/b/.git", worktreeB, "/r/b") },
			func() error { return insertVersion(tx, versionA, 1, "human:me") },
			func() error { return insertRequirement(tx, reqA, "REQ-001", nil) },
			func() error { return insertCriterion(tx, criterion, "AC-001-1", reqA, "manual", nil) },
			func() error {
				return stmt(`INSERT INTO acceptance_criteria (id, project_id, key, status, revision, revision_seq, origin,
    created_version, updated_version, retired_version, updated_seq, requirement_id, requirement_revision, statement, verification)
VALUES (?, ?, 'AC-001-2', 'retired', 1, 1, 'extracted', 1, 1, 1, 1, ?, 1, 'retired one', 'manual')`, retired, projectA, reqA)(tx)
			},
			func() error {
				return stmt(`INSERT INTO requirements (id, project_id, key, status, revision, revision_seq, origin, created_version,
    updated_version, updated_seq, title, kind, priority)
VALUES (?, ?, 'REQ-001', 'active', 1, 1, 'extracted', 1, 1, 1, 'log in', 'functional', 'must')`, reqB, projectB)(tx)
			},
			func() error {
				return stmt(`INSERT INTO acceptance_criteria (id, project_id, key, status, revision, revision_seq, origin,
    created_version, updated_version, updated_seq, requirement_id, requirement_revision, statement, verification)
VALUES (?, ?, 'AC-001-1', 'active', 1, 1, 'extracted', 1, 1, 1, ?, 1, 'in B', 'manual')`, critB, projectB, reqB)(tx)
			},
		} {
			if err := f(); err != nil {
				return err
			}
		}
		return nil
	})
	type batchRow struct {
		id, project, kind, worktree, actor, channel, tree, summary string
		scope, head, compareFrom, compareSource                    any
		seq                                                        int64
		base                                                       int
	}
	batch := func(b batchRow) func(tx *sql.Tx) error {
		if b.project == "" {
			b.project = projectA
		}
		if b.worktree == "" {
			b.worktree = worktreeA
		}
		if b.actor == "" {
			b.actor, b.channel = "agent:claude_code", "agent_cli"
		}
		if b.tree == "" {
			b.tree = "dirty"
		}
		if b.summary == "" {
			b.summary = "orders isolated"
		}
		return stmt(`INSERT INTO claim_batches (id, seq, project_id, kind, audit_scope_json, base_version, worktree_id, head, tree_state,
    compare_from, compare_from_source, session_id, summary, actor, channel)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?)`, b.id, b.seq, b.project, b.kind, b.scope, b.base, b.worktree, b.head, b.tree,
			b.compareFrom, b.compareSource, b.summary, b.actor, b.channel)
	}
	type claimRow struct {
		id, project, batch, targetType, refs, refsCheck string
		criterion, revision, assertion                  any
		seq                                             int64
		summary                                         string
	}
	claim := func(c claimRow) func(tx *sql.Tx) error {
		if c.project == "" {
			c.project = projectA
		}
		if c.refs == "" {
			c.refs = `{}`
		}
		if c.refsCheck == "" {
			c.refsCheck = `{"files":[],"commits":[]}`
		}
		if c.summary == "" {
			c.summary = "owner filter added"
		}
		return stmt(`INSERT INTO claims (id, seq, project_id, batch_id, target_type, criterion_id, criterion_revision, assertion, summary,
    refs_json, refs_check_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, c.id, c.seq, c.project, c.batch, c.targetType, c.criterion, c.revision, c.assertion, c.summary,
			c.refs, c.refsCheck)
	}
	sync := batchRow{id: batchSyn, kind: "sync", seq: 60, head: sha("a"), compareFrom: sha("b"), compareSource: "previous_batch"}
	ok("a sync batch with a done claim and unmapped work, one seq", func(tx *sql.Tx) error {
		for _, f := range []func(tx *sql.Tx) error{
			batch(sync),
			claim(claimRow{id: claimA, batch: batchSyn, seq: 60, targetType: "criterion", criterion: criterion, revision: 1, assertion: "done",
				refs:      `{"files":[{"path":"a.go","lines":"1-2"}],"tests":["TestA"],"commits":["abcdef1"]}`,
				refsCheck: `{"files":[{"path":"a.go","lines":"1-2","exists":true,"checked_head":"` + sha("a") + `"}],"commits":[{"commit":"abcdef1","resolved":false}]}`}),
			claim(claimRow{id: "01J9Z3K5Q8R2T4V6W8X0Y2Z4Q2", batch: batchSyn, seq: 60, targetType: "none"}),
			claim(claimRow{id: "01J9Z3K5Q8R2T4V6W8X0Y2Z4Q3", batch: batchSyn, seq: 60, targetType: "none"}),
		} {
			if err := f(tx); err != nil {
				return err
			}
		}
		return nil
	})
	ok("a sync batch from none, outside git", batch(batchRow{id: "01J9Z3K5Q8R2T4V6W8X0Y2Z4P5", kind: "sync", seq: 61, tree: "unknown", compareSource: "none"}))
	ok("an audit batch of all requirements with an unknown claim", func(tx *sql.Tx) error {
		if err := batch(batchRow{id: batchAud, kind: "audit", seq: 62, scope: `{"all":true}`, head: sha("c"), tree: "clean"})(tx); err != nil {
			return err
		}
		return claim(claimRow{id: "01J9Z3K5Q8R2T4V6W8X0Y2Z4Q4", batch: batchAud, seq: 62, targetType: "criterion", criterion: criterion, revision: 1,
			assertion: "unknown"})(tx)
	})
	ok("an audit batch of some requirements", batch(batchRow{id: "01J9Z3K5Q8R2T4V6W8X0Y2Z4P6", kind: "audit", seq: 63,
		scope: `{"requirements":["REQ-001"]}`, tree: "clean"}))
	ok("a manual batch from a person", batch(batchRow{id: batchMan, kind: "manual", seq: 64, actor: "human:me", channel: "human_cli", tree: "clean"}))
	ok("a sync batch of project B", batch(batchRow{id: batchB, project: projectB, worktree: worktreeB, kind: "sync", seq: 65, compareSource: "none"}))

	var ids rowIDs
	next := func(b batchRow) batchRow { b.id = ids.next(); b.seq = 100 + int64(ids); return b }
	for _, c := range []struct {
		name, why string
		fn        func(tx *sql.Tx) error
	}{
		{"a short batch ID", check, batch(batchRow{id: "01J9Z3K5Q8R2T4V6W8X0Y2Z4P", kind: "manual", seq: 90, actor: "human:me", channel: "human_cli"})},
		{"another kind", check, batch(batchRow{id: ids.next(), kind: "review", seq: 90, actor: "human:me", channel: "human_cli"})},
		{"an audit batch without a scope", check, batch(next(batchRow{kind: "audit"}))},
		{"a manual batch with a scope", check, batch(next(batchRow{kind: "manual", scope: `{"all":true}`, actor: "human:me", channel: "human_cli"}))},
		{"a scope that is not an object", check, batch(next(batchRow{kind: "audit", scope: `["REQ-001"]`}))},
		{"a scope that is not JSON", check, batch(next(batchRow{kind: "audit", scope: `all`}))},
		{"E2: a manual batch from the Agent channel", check, batch(next(batchRow{kind: "manual"}))},
		{"a person's actor on the Agent channel", check, batch(next(batchRow{kind: "audit", scope: `{"all":true}`, actor: "human:me", channel: "agent_cli"}))},
		{"an Agent's actor on the human channel", check, batch(next(batchRow{kind: "manual", actor: "agent:claude_code", channel: "human_cli"}))},
		{"the dashboard channel", check, batch(next(batchRow{kind: "manual", actor: "human:me", channel: "dashboard"}))},
		{"another actor", check, batch(next(batchRow{kind: "manual", actor: "system", channel: "human_cli"}))},
		{"a short head", check, batch(next(batchRow{kind: "manual", actor: "human:me", channel: "human_cli", head: sha("a")[1:]}))},
		{"an upper-case head", check, batch(next(batchRow{kind: "manual", actor: "human:me", channel: "human_cli", head: sha("A")}))},
		{"a SHA-256 head", check, batch(next(batchRow{kind: "manual", actor: "human:me", channel: "human_cli", head: strings.Repeat("a", 64)}))},
		{"another tree state", check, batch(next(batchRow{kind: "manual", actor: "human:me", channel: "human_cli", tree: "undecidable"}))},
		{"a sync batch without a comparison source", check, batch(next(batchRow{kind: "sync"}))},
		{"an audit batch with a comparison source", check, batch(next(batchRow{kind: "audit", scope: `{"all":true}`, compareSource: "none"}))},
		{"a manual batch with a comparison start", check, batch(next(batchRow{kind: "manual", actor: "human:me", channel: "human_cli", compareFrom: sha("b")}))},
		{"none with a commit", check, batch(next(batchRow{kind: "sync", compareFrom: sha("b"), compareSource: "none"}))},
		{"previous_batch without a commit", check, batch(next(batchRow{kind: "sync", compareSource: "previous_batch"}))},
		{"session_start without a commit", check, batch(next(batchRow{kind: "sync", compareSource: "session_start"}))},
		{"another comparison source", check, batch(next(batchRow{kind: "sync", compareFrom: sha("b"), compareSource: "merge_base"}))},
		{"an upper-case comparison start", check, batch(next(batchRow{kind: "sync", compareFrom: sha("B"), compareSource: "previous_batch"}))},
		{"an empty summary", check, stmt(`INSERT INTO claim_batches (id, seq, project_id, kind, base_version, worktree_id, tree_state,
    compare_from_source, summary, actor, channel)
VALUES (?, 92, ?, 'sync', 0, ?, 'dirty', 'none', '', 'agent:unknown', 'agent_cli')`, ids.next(), projectA, worktreeA)},
		{"a summary of 501 characters", check, batch(next(batchRow{kind: "sync", compareSource: "none", summary: strings.Repeat("x", 501)}))},
		{"a negative base version", check, batch(next(batchRow{kind: "sync", compareSource: "none", base: -1}))},
		{"the seq of another batch", unique, batch(batchRow{id: ids.next(), kind: "sync", compareSource: "none", seq: 60})},
		{"the ID of another batch", unique, batch(batchRow{id: batchSyn, kind: "sync", compareSource: "none", seq: 91})},
		{"a worktree of another project", fk, batch(next(batchRow{kind: "sync", compareSource: "none", worktree: worktreeB}))},
		{"no worktree", fk, batch(next(batchRow{kind: "sync", compareSource: "none", worktree: ids.next()}))},
		{"no project", fk, batch(next(batchRow{kind: "sync", compareSource: "none", project: ids.next()}))},

		{"another target type", check, claim(claimRow{id: ids.next(), batch: batchSyn, seq: 60, targetType: "requirement", criterion: criterion, revision: 1, assertion: "done"})},
		{"a criterion claim without a criterion", check, claim(claimRow{id: ids.next(), batch: batchSyn, seq: 60, targetType: "criterion", revision: 1, assertion: "done"})},
		{"a criterion claim without a revision", check, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "criterion", criterion: criterion, assertion: "done"})},
		{"a criterion claim without an assertion", check, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "criterion", criterion: criterion, revision: 1})},
		{"unmapped work with a criterion", check, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "none", criterion: criterion})},
		{"unmapped work with a revision", check, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "none", revision: 1})},
		{"unmapped work with an assertion", check, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "none", assertion: "done"})},
		{"another assertion", check, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "criterion", criterion: criterion, revision: 1, assertion: "maybe"})},
		{"a claim summary of 501 characters", check, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "none", summary: strings.Repeat("x", 501)})},
		{"references that are not an object", check, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "none", refs: `[]`})},
		{"reference results that are not an object", check, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "none", refsCheck: `[]`})},
		{"a claim with another seq than its batch", shared, claim(claimRow{id: ids.next(), batch: batchMan, seq: 65, targetType: "none"})},
		{"a claim of a batch that does not exist", shared, claim(claimRow{id: ids.next(), batch: ids.next(), seq: 64, targetType: "none"})},
		{"a claim of another project's batch", shared, claim(claimRow{id: ids.next(), batch: batchB, seq: 65, targetType: "none"})},
		{"unknown in a sync batch", unknown, claim(claimRow{id: ids.next(), batch: "01J9Z3K5Q8R2T4V6W8X0Y2Z4P5", seq: 61, targetType: "criterion", criterion: criterion, revision: 1, assertion: "unknown"})},
		{"unknown in a manual batch", unknown, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "criterion", criterion: criterion, revision: 1, assertion: "unknown"})},
		{"a retired criterion", target, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "criterion", criterion: retired, revision: 1, assertion: "done"})},
		{"a criterion at another revision", target, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "criterion", criterion: criterion, revision: 2, assertion: "done"})},
		{"another project's criterion", target, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "criterion", criterion: critB, revision: 1, assertion: "done"})},
		{"a criterion that does not exist", target, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "criterion", criterion: ids.next(), revision: 1, assertion: "done"})},
		{"a requirement taken for a criterion", target, claim(claimRow{id: ids.next(), batch: batchMan, seq: 64, targetType: "criterion", criterion: reqA, revision: 1, assertion: "done"})},
		{"the same criterion twice in a batch", unique, claim(claimRow{id: ids.next(), batch: batchSyn, seq: 60, targetType: "criterion", criterion: criterion, revision: 1, assertion: "partial"})},
		{"the ID of another claim", unique, claim(claimRow{id: claimA, batch: batchMan, seq: 64, targetType: "none"})},

		{"changing a batch", "append-only table claim_batches", stmt(`UPDATE claim_batches SET summary = 'changed'`)},
		{"deleting a batch", "append-only table claim_batches", stmt(`DELETE FROM claim_batches WHERE id = ?`, batchMan)},
		{"replacing a batch", "append-only table claim_batches", stmt(`INSERT OR REPLACE INTO claim_batches (id, seq, project_id, kind, base_version,
    worktree_id, tree_state, summary, actor, channel)
VALUES (?, 64, ?, 'manual', 0, ?, 'clean', 'replaced', 'human:me', 'human_cli')`, batchMan, projectA, worktreeA)},
		{"replacing a batch by its seq", "append-only table claim_batches", stmt(`REPLACE INTO claim_batches (id, seq, project_id, kind, base_version,
    worktree_id, tree_state, summary, actor, channel)
VALUES (?, 64, ?, 'manual', 0, ?, 'clean', 'replaced', 'human:me', 'human_cli')`, ids.next(), projectA, worktreeA)},
		{"an upsert of a batch", "append-only table claim_batches", stmt(`INSERT INTO claim_batches (id, seq, project_id, kind, base_version,
    worktree_id, tree_state, summary, actor, channel)
VALUES (?, 99, ?, 'manual', 0, ?, 'clean', 'new', 'human:me', 'human_cli') ON CONFLICT (id) DO UPDATE SET summary = excluded.summary`,
			batchMan, projectA, worktreeA)},
		{"changing a claim", "append-only table claims", stmt(`UPDATE claims SET assertion = 'not_done' WHERE id = ?`, claimA)},
		{"deleting a claim", "append-only table claims", stmt(`DELETE FROM claims WHERE id = ?`, claimA)},
		{"replacing a claim", "append-only table claims", stmt(`INSERT OR REPLACE INTO claims (id, seq, project_id, batch_id, target_type, summary,
    refs_json, refs_check_json)
VALUES (?, 60, ?, ?, 'none', 'replaced', '{}', '{}')`, claimA, projectA, batchSyn)},
		{"replacing a claim by its criterion", "append-only table claims", stmt(`INSERT OR REPLACE INTO claims (id, seq, project_id, batch_id,
    target_type, criterion_id, criterion_revision, assertion, summary, refs_json, refs_check_json)
VALUES (?, 60, ?, ?, 'criterion', ?, 1, 'blocked', 'replaced', '{}', '{}')`, ids.next(), projectA, batchSyn, criterion)},
		{"an upsert of a claim", "append-only table claims", stmt(`INSERT INTO claims (id, seq, project_id, batch_id, target_type, summary,
    refs_json, refs_check_json)
VALUES (?, 60, ?, ?, 'none', 'new', '{}', '{}') ON CONFLICT (id) DO UPDATE SET summary = excluded.summary`, claimA, projectA, batchSyn)},
		{"a batch removed while its claims stay", fk, func(tx *sql.Tx) error {
			return Maintenance(ctx, tx, func() error { return stmt(`DELETE FROM claim_batches WHERE id = ?`, batchSyn)(tx) })
		}},
	} {
		refused(c.name, c.why, c.fn)
	}
	if got := countRows(t, db, `SELECT count(*) FROM claims WHERE id = ? AND assertion = 'done' AND summary = 'owner filter added'`, claimA); got != 1 {
		t.Errorf("the first claim changed: %d", got)
	}
	ok("maintenance removes claims, then their batch", func(tx *sql.Tx) error {
		return Maintenance(ctx, tx, func() error {
			if err := stmt(`DELETE FROM claims WHERE batch_id = ?`, batchAud)(tx); err != nil {
				return err
			}
			return stmt(`DELETE FROM claim_batches WHERE id = ?`, batchAud)(tx)
		})
	})
	for _, index := range []string{"claim_batches_by_worktree", "claims_by_criterion", "claims_one_per_criterion"} {
		if n := countRows(t, db, `SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, index); n != 1 {
			t.Errorf("index %s: %d", index, n)
		}
	}

	_ = db.Close()
	writeTombstones(t, dir, projectA)
	db, _ = open(t, dir, Options{})
	for _, table := range []string{"claim_batches", "claims"} {
		if n := countRows(t, db, `SELECT count(*) FROM `+table+` WHERE project_id = '`+projectA+`'`); n != 0 {
			t.Errorf("%s has %d rows of the purged project", table, n)
		}
	}
	if n := countRows(t, db, `SELECT count(*) FROM claim_batches WHERE project_id = '`+projectB+`'`); n != 1 {
		t.Errorf("project B has %d batches after A was purged, want 1", n)
	}
}
