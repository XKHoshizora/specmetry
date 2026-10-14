package sqlite

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil"
)

const (
	proposalA = "01J9Z3K5Q8R2T4V6W8X0Y2Z4H1"
	proposalB = "01J9Z3K5Q8R2T4V6W8X0Y2Z4H2"
	itemA     = "01J9Z3K5Q8R2T4V6W8X0Y2Z4J1"
	questionA = "01J9Z3K5Q8R2T4V6W8X0Y2Z4K1"
	versionA  = "01J9Z3K5Q8R2T4V6W8X0Y2Z4M1"
	goalA     = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N1"
	reqA      = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N2"
	criterion = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N3"
)

// rowIDs hands out distinct ULIDs for rows a test does not refer to again.
type rowIDs int

func (n *rowIDs) next() string {
	*n++
	const digits = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	return "01J9Z3K5Q8R2T4V6W8X0Y2Z" + string(digits[int(*n)/1024%32]) + string(digits[int(*n)/32%32]) + string(digits[int(*n)%32])
}

func insertProposal(tx *sql.Tx, id, project, worktree, origin, actor, channel string) error {
	seq, err := NextSeq(ctx, tx)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO proposals (id, seq, project_id, origin, actor, channel, worktree_id, summary, base_version)
VALUES (?, ?, ?, ?, ?, ?, ?, 'initial baseline', 0)`, id, seq, project, origin, actor, channel, worktree)
	return err
}

// item is a proposal item row; the zero values are those of a pending
// add item.
type item struct {
	id, proposal, project, op, entityType string
	index                                 int
	target                                any
	targetRevision                        any
	ref                                   any
	sources, codeRefs                     string
	rationale, origin                     any
	suggested                             any
}

func insertItem(tx *sql.Tx, it item) error {
	if it.project == "" {
		it.project = projectA
	}
	if it.proposal == "" {
		it.proposal = proposalA
	}
	if it.op == "" {
		it.op, it.entityType, it.ref = "add", "requirement", "r1"
	}
	if it.sources == "" {
		it.sources = "[]"
	}
	if it.codeRefs == "" {
		it.codeRefs = "[]"
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO proposal_items (id, proposal_id, project_id, item_index, op, entity_type, target_id, target_revision,
    ref, payload_json, suggested_change_class, sources_json, code_refs_json, rationale, origin)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '{"title":"t"}', ?, ?, ?, ?, ?)`,
		it.id, it.proposal, it.project, it.index, it.op, it.entityType, it.target, it.targetRevision,
		it.ref, it.suggested, it.sources, it.codeRefs, it.rationale, it.origin)
	return err
}

func insertQuestion(tx *sql.Tx, id, key string, relatedType, relatedID any) error {
	seq, err := NextSeq(ctx, tx)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO questions (id, project_id, key, text, related_type, related_id, blocking, origin, proposal_id, created_seq)
VALUES (?, ?, ?, 'how many per page?', ?, ?, 0, 'skill_init', ?, ?)`, id, projectA, key, relatedType, relatedID, proposalA, seq)
	return err
}

func insertSourceRef(tx *sql.Tx, id, source string, excerpt any, verification string, sourceSHA any) error {
	seq, err := NextSeq(ctx, tx)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO source_refs (id, seq, project_id, source_id, entity_type, entity_id, entity_revision, locator,
    excerpt, excerpt_sha256, verification, source_sha256)
VALUES (?, ?, ?, ?, 'question', ?, 1, 'R3', ?, ?, ?, ?)`, id, seq, projectA, source, questionA, excerpt, shaA, verification, sourceSHA)
	return err
}

func insertVersion(tx *sql.Tx, id string, number int, actor string) error {
	seq, err := NextSeq(ctx, tx)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO baseline_versions (id, seq, project_id, number, summary, actor, channel)
VALUES (?, ?, ?, ?, 'v', ?, 'human_cli')`, id, seq, projectA, number, actor)
	return err
}

func insertGoal(tx *sql.Tx, id, key string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO goals (id, project_id, key, status, revision, revision_seq, origin, created_version,
    updated_version, updated_seq, title)
VALUES (?, ?, ?, 'active', 1, 1, 'extracted', 1, 1, 1, 'launch')`, id, projectA, key)
	return err
}

func insertRequirement(tx *sql.Tx, id, key string, goal any) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO requirements (id, project_id, key, status, revision, revision_seq, origin, created_version,
    updated_version, updated_seq, goal_id, title, kind, priority)
VALUES (?, ?, ?, 'active', 1, 1, 'extracted', 1, 1, 1, ?, 'log in', 'functional', 'must')`, id, projectA, key, goal)
	return err
}

func insertCriterion(tx *sql.Tx, id, key, requirement, verification string, command any) error {
	var rationale, hash any
	if command != nil {
		rationale, hash = "the test logs in five times", shaA
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO acceptance_criteria (id, project_id, key, status, revision, revision_seq, origin,
    created_version, updated_version, updated_seq, requirement_id, requirement_revision, statement, verification,
    check_command, check_rationale, check_config_hash)
VALUES (?, ?, ?, 'active', 1, 1, 'extracted', 1, 1, 1, ?, 1, 'locked after five failures', ?, ?, ?, ?)`,
		id, projectA, key, requirement, verification, command, rationale, hash)
	return err
}

func stmt(query string, args ...any) func(tx *sql.Tx) error {
	return func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}
}

// data-model §4.4 – §4.8 and §4.14, enforced by the schema as well as by
// the commands: each statement below is refused, and for the intended
// reason.
func TestProposalTablesConstraints(t *testing.T) {
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
	var ids rowIDs
	const check, unique, fk = "CHECK constraint failed", "UNIQUE constraint failed", "FOREIGN KEY constraint failed"

	ok("projects and a source", func(tx *sql.Tx) error {
		if err := insertProject(tx, projectA, "/r/a/.git", worktreeA, "/r/a"); err != nil {
			return err
		}
		if err := insertProject(tx, projectB, "/r/b/.git", worktreeB, "/r/b"); err != nil {
			return err
		}
		if err := insertSource(tx, sourceA, projectA); err != nil {
			return err
		}
		return insertSource(tx, sourceB, projectB)
	})

	// Baseline versions: append-only, made by a person.
	ok("version 1", func(tx *sql.Tx) error { return insertVersion(tx, versionA, 1, "human:me") })
	refused("a version by an Agent", check, func(tx *sql.Tx) error { return insertVersion(tx, ids.next(), 2, "agent:claude_code") })
	refused("version 1 again", unique, func(tx *sql.Tx) error { return insertVersion(tx, ids.next(), 1, "human:me") })
	refused("changing a version", "append-only table baseline_versions", stmt(`UPDATE baseline_versions SET summary = 'x'`))
	refused("deleting a version", "append-only table baseline_versions", stmt(`DELETE FROM baseline_versions`))

	// Baseline entities.
	ok("a goal, a requirement and a criterion", func(tx *sql.Tx) error {
		if err := insertGoal(tx, goalA, "GOAL-1"); err != nil {
			return err
		}
		if err := insertRequirement(tx, reqA, "REQ-001", goalA); err != nil {
			return err
		}
		return insertCriterion(tx, criterion, "AC-001-1", reqA, "automated", "go test ./...")
	})
	refused("a malformed goal key", check, func(tx *sql.Tx) error { return insertGoal(tx, ids.next(), "GOAL-01") })
	refused("a malformed requirement key", check, func(tx *sql.Tx) error { return insertRequirement(tx, ids.next(), "REQ-1", nil) })
	refused("the same requirement key twice", unique, func(tx *sql.Tx) error { return insertRequirement(tx, ids.next(), "REQ-001", nil) })
	refused("a manual criterion with a check", check, func(tx *sql.Tx) error {
		return insertCriterion(tx, ids.next(), "AC-001-2", reqA, "manual", "go test ./...")
	})
	refused("a criterion of a missing requirement", fk, func(tx *sql.Tx) error {
		return insertCriterion(tx, ids.next(), "AC-001-3", ids.next(), "manual", nil)
	})
	refused("a multi-line title", check, stmt(`UPDATE goals SET title = 'a' || char(10) || 'b' WHERE id = ?`, goalA))
	refused("changing a key", "a baseline key never changes", stmt(`UPDATE requirements SET key = 'REQ-002' WHERE id = ?`, reqA))
	refused("moving a criterion", "a criterion does not move to another requirement",
		stmt(`UPDATE acceptance_criteria SET requirement_id = ? WHERE id = ?`, goalA, criterion))
	refused("deleting a requirement", "baseline entities are retired, not deleted", stmt(`DELETE FROM requirements WHERE id = ?`, reqA))
	ok("retiring the goal", stmt(`UPDATE goals SET status = 'retired', retired_version = 1 WHERE id = ?`, goalA))
	refused("changing a retired goal", "a retired baseline entity does not change", stmt(`UPDATE goals SET title = 'x' WHERE id = ?`, goalA))

	// Proposals.
	ok("an Agent proposal", func(tx *sql.Tx) error {
		return insertProposal(tx, proposalA, projectA, worktreeA, "skill_init", "agent:claude_code", "agent_cli")
	})
	refused("human_edit from the Agent channel", check, func(tx *sql.Tx) error {
		return insertProposal(tx, ids.next(), projectA, worktreeA, "human_edit", "human:me", "agent_cli")
	})
	refused("an Agent actor on a human channel", check, func(tx *sql.Tx) error {
		return insertProposal(tx, ids.next(), projectA, worktreeA, "cli", "agent:claude_code", "human_cli")
	})
	refused("a proposal from another project's worktree", fk, func(tx *sql.Tx) error {
		return insertProposal(tx, ids.next(), projectA, worktreeB, "cli", "agent:unknown", "agent_cli")
	})
	refused("changing a proposal", "a submitted proposal does not change", stmt(`UPDATE proposals SET summary = 'x'`))
	refused("deleting a proposal", "proposals are never deleted", stmt(`DELETE FROM proposals`))

	// Items.
	ok("an add item", func(tx *sql.Tx) error {
		return insertItem(tx, item{id: itemA, sources: `[{"source_id":"` + sourceA + `","locator":"R1","excerpt_sha256":"` + shaA + `","verification":"verified"}]`})
	})
	refused("an add item without a ref", check, func(tx *sql.Tx) error {
		return insertItem(tx, item{id: ids.next(), index: 1, op: "add", entityType: "goal"})
	})
	refused("the same ref twice in a proposal", unique, func(tx *sql.Tx) error { return insertItem(tx, item{id: ids.next(), index: 1}) })
	refused("the same index twice", unique, func(tx *sql.Tx) error {
		return insertItem(tx, item{id: ids.next(), op: "add", entityType: "goal", ref: "g1"})
	})
	refused("an update without a target", check, func(tx *sql.Tx) error {
		return insertItem(tx, item{id: ids.next(), index: 2, op: "update", entityType: "goal"})
	})
	refused("reaffirming a requirement", check, func(tx *sql.Tx) error {
		return insertItem(tx, item{id: ids.next(), index: 2, op: "reaffirm", entityType: "requirement", target: reqA, targetRevision: 1})
	})
	refused("retiring without a rationale", check, func(tx *sql.Tx) error {
		return insertItem(tx, item{id: ids.next(), index: 2, op: "retire", entityType: "requirement", target: reqA, targetRevision: 1})
	})
	refused("a suggested class on an add item", check, func(tx *sql.Tx) error {
		return insertItem(tx, item{id: ids.next(), index: 2, op: "add", entityType: "goal", ref: "g2", suggested: "material"})
	})
	refused("inferred_from_code without code references", check, func(tx *sql.Tx) error {
		return insertItem(tx, item{id: ids.next(), index: 2, op: "add", entityType: "goal", ref: "g2", origin: "inferred_from_code"})
	})
	refused("an item of another project's proposal", fk, func(tx *sql.Tx) error {
		return insertItem(tx, item{id: ids.next(), index: 2, project: projectB, op: "add", entityType: "goal", ref: "g2"})
	})
	refused("an excerpt in an item under hash_only", "excerpt text is not stored under hash_only", func(tx *sql.Tx) error {
		return insertItem(tx, item{id: ids.next(), index: 2, op: "add", entityType: "goal", ref: "g2", sources: `[{"source_id":"` + sourceA + `","excerpt":"text"}]`})
	})
	refused("changing what was submitted", "a submitted proposal item does not change",
		stmt(`UPDATE proposal_items SET payload_json = '{}' WHERE id = ?`, itemA))
	refused("a review without a reviewer", check, stmt(`UPDATE proposal_items SET review_state = 'rejected' WHERE id = ?`, itemA))
	ok("the review", stmt(`UPDATE proposal_items SET review_state = 'rejected', reviewed_by = 'human:me', reviewed_channel = 'human_cli',
    reviewed_seq = 99 WHERE id = ?`, itemA))
	refused("reviewing again", "a reviewed proposal item does not change",
		stmt(`UPDATE proposal_items SET review_note = 'x' WHERE id = ?`, itemA))
	refused("deleting an item", "proposal items are never deleted", stmt(`DELETE FROM proposal_items`))

	// Questions.
	ok("a question about the requirement", func(tx *sql.Tx) error { return insertQuestion(tx, questionA, "Q-001", "requirement", reqA) })
	refused("a malformed question key", check, func(tx *sql.Tx) error { return insertQuestion(tx, ids.next(), "Q-1", nil, nil) })
	refused("the same question key twice", unique, func(tx *sql.Tx) error { return insertQuestion(tx, ids.next(), "Q-001", nil, nil) })
	refused("a related type without an entity", check, func(tx *sql.Tx) error {
		return insertQuestion(tx, ids.next(), "Q-002", "requirement", nil)
	})
	refused("an answer without the answering fact", check, stmt(`UPDATE questions SET status = 'answered' WHERE id = ?`, questionA))
	ok("the answer", stmt(`UPDATE questions SET status = 'answered', answer = '20', answered_by = 'human:me',
    answered_channel = 'human_cli', answered_seq = 99 WHERE id = ?`, questionA))
	refused("answering again", "an answered or dismissed question does not change",
		stmt(`UPDATE questions SET answer = '30' WHERE id = ?`, questionA))
	refused("deleting a question", "questions are never deleted", stmt(`DELETE FROM questions`))

	// Source references.
	ok("a hash_only reference", func(tx *sql.Tx) error {
		return insertSourceRef(tx, ids.next(), sourceA, nil, "verified", shaB)
	})
	refused("excerpt text under hash_only", "excerpt text is not stored under hash_only", func(tx *sql.Tx) error {
		return insertSourceRef(tx, ids.next(), sourceA, "text", "verified", shaB)
	})
	refused("a missing source with a hash", check, func(tx *sql.Tx) error {
		return insertSourceRef(tx, ids.next(), sourceA, nil, "source_missing", shaB)
	})
	refused("a read source without a hash", check, func(tx *sql.Tx) error {
		return insertSourceRef(tx, ids.next(), sourceA, nil, "not_found", nil)
	})
	refused("a reference to another project's source", fk, func(tx *sql.Tx) error {
		return insertSourceRef(tx, ids.next(), sourceB, nil, "verified", shaB)
	})
	refused("changing a reference", "append-only table source_refs", stmt(`UPDATE source_refs SET locator = 'x'`))
	refused("deleting a reference", "append-only table source_refs", stmt(`DELETE FROM source_refs`))
	ok("full mode", stmt(`UPDATE projects SET settings_json = json_set(settings_json, '$.excerpt_storage', 'full') WHERE id = ?`, projectA))
	ok("excerpt text under full", func(tx *sql.Tx) error {
		return insertSourceRef(tx, ids.next(), sourceA, "text", "verified", shaB)
	})

	// Code references.
	ok("a code reference", stmt(`INSERT INTO code_refs (id, seq, project_id, entity_type, entity_id, entity_revision, path, lines, checked_head, "exists")
VALUES (?, 1, ?, 'requirement', ?, 1, 'internal/auth/login.go', '40-62', ?, 1)`, ids.next(), projectA, reqA, strings.Repeat("c", 40)))
	refused("a malformed line range", check, stmt(`INSERT INTO code_refs (id, seq, project_id, entity_type, entity_id, entity_revision, path, lines, "exists")
VALUES (?, 1, ?, 'requirement', ?, 1, 'a.go', '0-3', 0)`, ids.next(), projectA, reqA))
	refused("changing a code reference", "append-only table code_refs", stmt(`UPDATE code_refs SET "exists" = 0`))

	// Key counters.
	ok("a counter", stmt(`INSERT INTO key_counters (project_id, entity_type, scope, next_value) VALUES (?, 'question', '', 2)`, projectA))
	ok("the counter grows", stmt(`UPDATE key_counters SET next_value = 3`))
	refused("the counter going back", "a key counter only grows", stmt(`UPDATE key_counters SET next_value = 2`))
	refused("deleting a counter", "key counters are never deleted", stmt(`DELETE FROM key_counters`))
	refused("a criterion counter without its requirement", check,
		stmt(`INSERT INTO key_counters (project_id, entity_type, scope, next_value) VALUES (?, 'criterion', '', 1)`, projectA))
}

// Tombstones remove every table this migration adds, despite their
// append-only and never-deleted triggers (maintenance mode).
func TestTombstonesRemoveProposals(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{})
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		for _, step := range []func() error{
			func() error { return insertProject(tx, projectA, "/r/a/.git", worktreeA, "/r/a") },
			func() error { return insertProject(tx, projectB, "/r/b/.git", worktreeB, "/r/b") },
			func() error { return insertSource(tx, sourceA, projectA) },
			func() error { return insertVersion(tx, versionA, 1, "human:me") },
			func() error { return insertGoal(tx, goalA, "GOAL-1") },
			func() error { return insertRequirement(tx, reqA, "REQ-001", goalA) },
			func() error { return insertCriterion(tx, criterion, "AC-001-1", reqA, "manual", nil) },
			func() error {
				return insertProposal(tx, proposalA, projectA, worktreeA, "skill_init", "agent:claude_code", "agent_cli")
			},
			func() error {
				return insertProposal(tx, proposalB, projectB, worktreeB, "skill_init", "agent:claude_code", "agent_cli")
			},
			func() error { return insertItem(tx, item{id: itemA}) },
			func() error { return insertQuestion(tx, questionA, "Q-001", nil, nil) },
			func() error {
				return insertSourceRef(tx, "01J9Z3K5Q8R2T4V6W8X0Y2Z4P1", sourceA, nil, "verified", shaB)
			},
			func() error {
				return stmt(`INSERT INTO key_counters (project_id, entity_type, scope, next_value) VALUES (?, 'question', '', 2)`, projectA)(tx)
			},
		} {
			if err := step(); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	writeTombstones(t, dir, projectA)
	db, _ = open(t, dir, Options{})
	defer func() { _ = db.Close() }()
	for _, table := range []string{"baseline_versions", "goals", "requirements", "acceptance_criteria", "proposal_items",
		"questions", "source_refs", "key_counters"} {
		if n := countRows(t, db, `SELECT count(*) FROM `+table); n != 0 {
			t.Errorf("%s has %d rows of the purged project", table, n)
		}
	}
	if n := countRows(t, db, `SELECT count(*) FROM proposals`); n != 1 {
		t.Errorf("%d proposals, want the other project's 1", n)
	}
}
