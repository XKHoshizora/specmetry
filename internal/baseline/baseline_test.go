package baseline_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

var ctx = context.Background()

const (
	projectA = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"
	projectB = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A7"
	reqOne   = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N2"
	reqTwo   = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N4"
)

// store opens a fresh database with projects A and B.
func store(t *testing.T) *sqlite.DB {
	t.Helper()
	db, _, err := sqlite.Open(ctx, testutil.TempHome(t), sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	write(t, db, `
INSERT INTO projects (id, name, git_common_dir, main_worktree_id, stage, settings_json, created_at)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 'a', '/r/a/.git', '01J9Z3K5Q8R2T4V6W8X0Y2Z4B6', 'docs_only', '{}', 1),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4A7', 'b', '/r/b/.git', '01J9Z3K5Q8R2T4V6W8X0Y2Z4B7', 'docs_only', '{}', 1);
INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4B6', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', '/r/a', '/r/a/.git', 1, 1, 1),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4B7', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A7', '/r/b', '/r/b/.git', 1, 2, 1);`)
	return db
}

func write(t *testing.T, db *sqlite.DB, query string, args ...any) {
	t.Helper()
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func inTx[T any](t *testing.T, db *sqlite.DB, fn func(tx *sql.Tx) (T, error)) T {
	t.Helper()
	var out T
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		var err error
		out, err = fn(tx)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

// The current baseline version is the highest number of the project, and 0
// before the first; other projects' versions do not count.
func TestCurrentVersion(t *testing.T) {
	db := store(t)
	version := func(project string) int {
		return inTx(t, db, func(tx *sql.Tx) (int, error) { return baseline.CurrentVersion(ctx, tx, project) })
	}
	if v := version(projectA); v != 0 {
		t.Fatalf("version before the first = %d, want 0", v)
	}
	write(t, db, `INSERT INTO baseline_versions (id, seq, project_id, number, summary, actor, channel) VALUES
('01J9Z3K5Q8R2T4V6W8X0Y2Z4M1', 10, ?1, 1, 'v1', 'human:me', 'human_cli'),
('01J9Z3K5Q8R2T4V6W8X0Y2Z4M2', 11, ?1, 2, 'v2', 'human:me', 'human_cli'),
('01J9Z3K5Q8R2T4V6W8X0Y2Z4M3', 12, ?2, 7, 'other', 'human:me', 'human_cli')`, projectA, projectB)
	if v := version(projectA); v != 2 {
		t.Fatalf("version = %d, want 2", v)
	}
}

// Find looks a key up in the table of its type, within the project, and
// reports its status and revision; a key of another type, project or
// shape is not found.
func TestFind(t *testing.T) {
	db := store(t)
	write(t, db, `
INSERT INTO goals (id, project_id, key, status, revision, revision_seq, origin, created_version, updated_version, updated_seq, title)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4N1', ?1, 'GOAL-1', 'active', 1, 1, 'extracted', 1, 1, 1, 'launch');
INSERT INTO requirements (id, project_id, key, status, revision, revision_seq, origin, created_version, updated_version,
    retired_version, updated_seq, title, kind, priority)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4N2', ?1, 'REQ-001', 'active', 3, 1, 'extracted', 1, 1, NULL, 1, 'log in', 'functional', 'must'),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4N4', ?1, 'REQ-002', 'retired', 1, 1, 'extracted', 1, 2, 2, 1, 'old', 'functional', 'must');
INSERT INTO acceptance_criteria (id, project_id, key, status, revision, revision_seq, origin, created_version, updated_version,
    updated_seq, requirement_id, requirement_revision, statement, verification)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4N3', ?1, 'AC-001-1', 'active', 2, 1, 'extracted', 1, 1, 1, '01J9Z3K5Q8R2T4V6W8X0Y2Z4N2', 3,
    'locked after five failures', 'manual');`, projectA)
	cases := []struct {
		project, key string
		want         baseline.Entity
		found        bool
	}{
		{projectA, "GOAL-1", baseline.Entity{ID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4N1", Type: baseline.Goal, Key: "GOAL-1", Active: true, Revision: 1}, true},
		{projectA, "REQ-001", baseline.Entity{ID: reqOne, Type: baseline.Requirement, Key: "REQ-001", Active: true, Revision: 3}, true},
		{projectA, "REQ-002", baseline.Entity{ID: reqTwo, Type: baseline.Requirement, Key: "REQ-002", Active: false, Revision: 1}, true},
		{projectA, "AC-001-1", baseline.Entity{ID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4N3", Type: baseline.Criterion, Key: "AC-001-1",
			Active: true, Revision: 2, RequirementID: reqOne, RequirementRevision: 3, Verification: "manual"}, true},
		{projectA, "REQ-003", baseline.Entity{}, false},
		{projectB, "REQ-001", baseline.Entity{}, false},
		{projectA, "Q-001", baseline.Entity{}, false},
		{projectA, "req-001", baseline.Entity{}, false},
	}
	for _, c := range cases {
		type result struct {
			e     baseline.Entity
			found bool
		}
		got := inTx(t, db, func(tx *sql.Tx) (result, error) {
			e, found, err := baseline.Find(ctx, tx, c.project, c.key)
			return result{e, found}, err
		})
		if got.found != c.found || got.e != c.want {
			t.Errorf("Find(%s) = %+v, %v; want %+v, %v", c.key, got.e, got.found, c.want, c.found)
		}
	}
}

// TD-14: GOAL-1, REQ-001, Q-001 and AC-<requirement number>-n; numbers are
// handed out in order, criteria within their requirement, each project on
// its own, and never twice. Expected keys are written out, not computed.
func TestNextKey(t *testing.T) {
	db := store(t)
	next := func(project string, typ baseline.Type, req *baseline.Entity) string {
		t.Helper()
		return inTx(t, db, func(tx *sql.Tx) (string, error) { return baseline.NextKey(ctx, tx, project, typ, req) })
	}
	one := &baseline.Entity{ID: reqOne, Type: baseline.Requirement, Key: "REQ-001"}
	two := &baseline.Entity{ID: reqTwo, Type: baseline.Requirement, Key: "REQ-002"}
	steps := []struct {
		project string
		typ     baseline.Type
		req     *baseline.Entity
		want    string
	}{
		{projectA, baseline.Goal, nil, "GOAL-1"},
		{projectA, baseline.Goal, nil, "GOAL-2"},
		{projectA, baseline.Requirement, nil, "REQ-001"},
		{projectA, baseline.Requirement, nil, "REQ-002"},
		{projectA, baseline.Criterion, one, "AC-001-1"},
		{projectA, baseline.Criterion, one, "AC-001-2"},
		{projectA, baseline.Criterion, two, "AC-002-1"},
		{projectA, baseline.Criterion, one, "AC-001-3"},
		{projectA, baseline.Question, nil, "Q-001"},
		{projectA, baseline.Question, nil, "Q-002"},
		{projectB, baseline.Requirement, nil, "REQ-001"},
		{projectB, baseline.Question, nil, "Q-001"},
		{projectA, baseline.Requirement, nil, "REQ-003"},
	}
	for i, s := range steps {
		if got := next(s.project, s.typ, s.req); got != s.want {
			t.Fatalf("step %d: NextKey(%s) = %s, want %s", i, s.typ, got, s.want)
		}
	}
	// Past three digits the number simply grows.
	write(t, db, `UPDATE key_counters SET next_value = 999 WHERE project_id = ? AND entity_type = 'requirement'`, projectA)
	for _, want := range []string{"REQ-999", "REQ-1000"} {
		if got := next(projectA, baseline.Requirement, nil); got != want {
			t.Fatalf("NextKey = %s, want %s", got, want)
		}
	}
	thousand := &baseline.Entity{ID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4N5", Type: baseline.Requirement, Key: "REQ-1000"}
	if got := next(projectA, baseline.Criterion, thousand); got != "AC-1000-1" {
		t.Fatalf("NextKey of a criterion of REQ-1000 = %s", got)
	}
	// A rolled-back transaction hands nothing out.
	_ = db.WriteTx(ctx, func(tx *sql.Tx) error {
		if _, err := baseline.NextKey(ctx, tx, projectA, baseline.Question, nil); err != nil {
			t.Fatal(err)
		}
		return context.Canceled
	})
	if got := next(projectA, baseline.Question, nil); got != "Q-003" {
		t.Fatalf("after a rollback NextKey = %s, want Q-003", got)
	}
}

// A criterion is numbered within a requirement, so it needs one.
func TestNextKeyOfACriterionNeedsItsRequirement(t *testing.T) {
	db := store(t)
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := baseline.NextKey(ctx, tx, projectA, baseline.Criterion, nil)
		return err
	})
	if err == nil {
		t.Fatal("a criterion key without its requirement was handed out")
	}
}
