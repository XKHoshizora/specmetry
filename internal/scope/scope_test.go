package scope_test

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/scope"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

var ctx = context.Background()

const (
	projectA = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"
	targetA  = "01J9Z3K5Q8R2T4V6W8X0Y2Z4C1"
	c1       = "01J9Z3K5Q8R2T4V6W8X0Y2Z4D1"
	c2       = "01J9Z3K5Q8R2T4V6W8X0Y2Z4D2"
	hashA    = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hashB    = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// state is a project at baseline version 3 with REQ-001 and REQ-002 must,
// REQ-003 should, 2 items pending, target A and its confirmation C1.
func state() scope.State {
	return scope.State{
		Target:          &scope.Target{ID: targetA, Name: "default", Ref: "refs/heads/main", CreatedSeq: 2},
		Latest:          &scope.Confirmation{ID: c1, Seq: 9, BaselineVersion: 3, SourceSetHash: hashA},
		BaselineVersion: 3,
		SourceSetHash:   hashA,
		PendingItems:    2,
		Stage:           "docs_only",
		Requirements: map[string]scope.Requirement{
			"REQ-001": {ID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4N1", Key: "REQ-001", Priority: "must", Revision: 1},
			"REQ-002": {ID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4N2", Key: "REQ-002", Priority: "must", Revision: 2},
			"REQ-010": {ID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4N3", Key: "REQ-010", Priority: "should", Revision: 1},
		},
		Waivers: []scope.Waiver{},
	}
}

func input() scope.Input {
	v, seen := 3, c1
	return scope.Input{TargetID: targetA, BaselineVersion: &v, SourceSetHash: hashA, Requirements: []string{"REQ-002", "REQ-001"},
		Attestations: scope.Attestations{CompleteScope: true, PendingItemsExcluded: 2}, AcknowledgedWaivers: []string{},
		SeenScopeConfirmationID: &seen}
}

func ptr[T any](v T) *T { return &v }

// expect checks err: exit code, the pointer of its detail, and a current
// value when it is stale.
func expect(t *testing.T, name string, err error, code int, pointer, currentKey string) {
	t.Helper()
	if apperr.ExitCode(err) != code {
		t.Errorf("%s: %v, want exit %d", name, err, code)
		return
	}
	if code == apperr.ExitOK {
		return
	}
	e := apperr.From(err)
	found := false
	for _, d := range e.Details {
		found = found || d.Pointer == pointer
	}
	if !found {
		t.Errorf("%s: no detail at %s: %+v", name, pointer, e.Details)
	}
	if currentKey != "" {
		if _, ok := e.Current[currentKey]; !ok {
			t.Errorf("%s: current lacks %s: %v", name, currentKey, e.Current)
		}
	}
}

// data-model §7.1: every value read is compared with the current one, and
// any that moved on is stale (exit 5) with it; then the requirements and
// the attestations must be valid (exit 2). Pointers sit under base.
func TestCheck(t *testing.T) {
	reqs, err := scope.Check(state(), input(), "")
	if err != nil || len(reqs) != 2 || reqs[0].Key != "REQ-001" || reqs[1].Key != "REQ-002" {
		t.Fatalf("Check = %+v, %v", reqs, err)
	}
	for _, c := range []struct {
		name      string
		st        func(*scope.State)
		in        func(*scope.Input)
		base      string
		code      int
		pointer   string
		current   string
		wantValue any
	}{
		{"another target", nil, func(in *scope.Input) { in.TargetID = "01J9Z3K5Q8R2T4V6W8X0Y2Z4C2" }, "", 5, "/target_id", "target_id", targetA},
		{"no target now", func(st *scope.State) { st.Target = nil; st.Latest = nil }, nil, "", 5, "/target_id", "target_id", nil},
		{"an older version", nil, func(in *scope.Input) { in.BaselineVersion = ptr(2) }, "", 5, "/baseline_version", "baseline_version", 3},
		{"another source set", nil, func(in *scope.Input) { in.SourceSetHash = hashB }, "", 5, "/source_set_hash", "source_set_hash", hashA},
		{"confirmed meanwhile (V-T02)", func(st *scope.State) { st.Latest.ID = c2 }, nil, "", 5, "/seen_scope_confirmation_id", "scope_confirmation_id", c2},
		{"a first confirmation, but one exists (V-T02)", nil, func(in *scope.Input) { in.SeenScopeConfirmationID = nil }, "", 5,
			"/seen_scope_confirmation_id", "scope_confirmation_id", c1},
		{"naming one, but none exists (V-T02)", func(st *scope.State) { st.Latest = nil }, nil, "", 5, "/seen_scope_confirmation_id",
			"scope_confirmation_id", nil},
		{"another pending count", nil, func(in *scope.Input) { in.Attestations.PendingItemsExcluded = 1 }, "", 5,
			"/attestations/pending_items_excluded", "pending_items", 2},
		{"in a review: another pending count", nil, func(in *scope.Input) { in.BaselineVersion = nil; in.Attestations.PendingItemsExcluded = 0 }, "/scope", 5,
			"/scope/attestations/pending_items_excluded", "pending_items", 2},
		{"a retired or unknown requirement", nil, func(in *scope.Input) { in.Requirements = []string{"REQ-001", "REQ-099"} }, "", 2, "/requirements/1", "", nil},
		{"a requirement twice", nil, func(in *scope.Input) { in.Requirements = []string{"REQ-001", "REQ-001"} }, "", 2, "/requirements/1", "", nil},
		{"an empty scope", nil, func(in *scope.Input) { in.Requirements = nil }, "", 2, "/requirements", "", nil},
		{"not attested complete", nil, func(in *scope.Input) { in.Attestations.CompleteScope = false }, "", 2, "/attestations/complete_scope", "", nil},
		{"incomplete_docs without the missing goals", func(st *scope.State) { st.Stage = scope.IncompleteDocs }, nil, "", 2,
			"/attestations/missing_goals_reviewed", "", nil},
		{"incomplete_docs, goals not reviewed", func(st *scope.State) { st.Stage = scope.IncompleteDocs },
			func(in *scope.Input) { in.Attestations.MissingGoalsReviewed = ptr(false) }, "", 2, "/attestations/missing_goals_reviewed", "", nil},
		{"incomplete_docs, goals reviewed", func(st *scope.State) { st.Stage = scope.IncompleteDocs },
			func(in *scope.Input) { in.Attestations.MissingGoalsReviewed = ptr(true) }, "", 0, "", "", nil},
		{"no baseline version yet", func(st *scope.State) { st.BaselineVersion = 0 }, func(in *scope.Input) { in.BaselineVersion = nil }, "/scope", 2,
			"/scope/requirements", "", nil},
		{"in a review, bound to the version it leaves", nil, func(in *scope.Input) { in.BaselineVersion = nil }, "/scope", 0, "", "", nil},
	} {
		st, in := state(), input()
		if c.st != nil {
			c.st(&st)
		}
		if c.in != nil {
			c.in(&in)
		}
		_, err := scope.Check(st, in, c.base)
		expect(t, c.name, err, c.code, c.pointer, c.current)
		if c.current != "" && err != nil {
			if got := apperr.From(err).Current[c.current]; got != c.wantValue {
				t.Errorf("%s: current %s = %v, want %v", c.name, c.current, got, c.wantValue)
			}
		}
	}
}

// V-T01 (PD-21), with waivers in effect: acknowledged_waivers must be
// exactly those under the confirmed requirements, in any order; one
// missing, one extra (under a requirement left out), or one no longer in
// effect is stale with the set in effect. This version records no
// decision, so a project's set is empty (TestRead), but the check is the
// same.
func TestCheckWaivers(t *testing.T) {
	const w1, w2, w3, old = "01J9Z3K5Q8R2T4V6W8X0Y2Z4G1", "01J9Z3K5Q8R2T4V6W8X0Y2Z4G2", "01J9Z3K5Q8R2T4V6W8X0Y2Z4G3", "01J9Z3K5Q8R2T4V6W8X0Y2Z4G9"
	st := state()
	st.Waivers = []scope.Waiver{
		{DecisionID: w2, Criterion: "AC-001-2", CriterionRevision: 1, Requirement: "REQ-001", Rationale: "later", Seq: 7},
		{DecisionID: w1, Criterion: "AC-001-1", CriterionRevision: 2, Requirement: "REQ-001", Rationale: "not needed", Seq: 6},
		{DecisionID: w3, Criterion: "AC-010-1", CriterionRevision: 1, Requirement: "REQ-010", Rationale: "out", Seq: 8},
	}
	for _, c := range []struct {
		name  string
		reqs  []string
		acked []string
		ok    bool
	}{
		{"exactly, in another order", []string{"REQ-001", "REQ-002"}, []string{w2, w1}, true},
		{"one missing", []string{"REQ-001", "REQ-002"}, []string{w1}, false},
		{"one extra, under a requirement left out", []string{"REQ-001", "REQ-002"}, []string{w1, w2, w3}, false},
		{"one no longer in effect", []string{"REQ-001", "REQ-002"}, []string{w1, w2, old}, false},
		{"none under the confirmed requirements", []string{"REQ-002"}, []string{}, true},
		{"none acknowledged, but some in effect", []string{"REQ-010"}, []string{}, false},
		{"all three", []string{"REQ-001", "REQ-010"}, []string{w1, w2, w3}, true},
	} {
		in := input()
		in.Requirements, in.AcknowledgedWaivers = c.reqs, c.acked
		_, err := scope.Check(st, in, "")
		if c.ok {
			if err != nil {
				t.Errorf("%s: %v", c.name, err)
			}
			continue
		}
		expect(t, c.name, err, 5, "/acknowledged_waivers", "effective_waivers")
		if err == nil {
			continue
		}
		got, _ := apperr.From(err).Current["effective_waivers"].([]scope.Waiver)
		want := st.WaiversUnder(c.reqs)
		if len(got) != len(want) {
			t.Errorf("%s: current %v, want %v", c.name, got, want)
		}
	}
}

// store opens a database with project A (git, docs_only), its default
// target, baseline version 1 with REQ-001 (must), REQ-002 (should) and the
// retired REQ-003 (must), and a proposal with 2 items pending.
func store(t *testing.T) (datadir.Dir, *sqlite.DB) {
	t.Helper()
	home := testutil.TempHome(t)
	db := open(t, home)
	write(t, db, `
INSERT INTO projects (id, name, git_common_dir, main_worktree_id, stage, settings_json, created_at)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 'a', '/r/a/.git', '01J9Z3K5Q8R2T4V6W8X0Y2Z4B6', 'docs_only', '{}', 1);
INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4B6', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', '/r/a', '/r/a/.git', 1, 1, 1);
INSERT INTO delivery_targets (id, project_id, name, ref, created_seq, actor, channel)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4C1', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 'default', 'refs/heads/main', 2, 'system', 'system');
INSERT INTO baseline_versions (id, seq, project_id, number, summary, actor, channel)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4M1', 3, '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 1, 'v1', 'human:me', 'human_cli');
INSERT INTO requirements (id, project_id, key, status, revision, revision_seq, origin, created_version, updated_version, updated_seq,
    title, kind, priority, retired_version)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4N1', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 'REQ-001', 'active', 2, 3, 'extracted', 1, 1, 3, 'log in', 'functional', 'must', NULL),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4N2', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 'REQ-002', 'active', 1, 3, 'extracted', 1, 1, 3, 'export', 'functional', 'should', NULL),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4N3', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 'REQ-003', 'retired', 1, 3, 'extracted', 1, 1, 3, 'gone', 'functional', 'must', 1);
INSERT INTO proposals (id, seq, project_id, origin, actor, channel, worktree_id, summary, base_version)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4P1', 4, '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 'skill_goals', 'agent:claude_code', 'agent_cli',
        '01J9Z3K5Q8R2T4V6W8X0Y2Z4B6', 'more', 1);
INSERT INTO proposal_items (id, proposal_id, project_id, item_index, op, entity_type, ref, payload_json, sources_json, code_refs_json)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4J1', '01J9Z3K5Q8R2T4V6W8X0Y2Z4P1', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 0, 'add', 'requirement', 'r1',
        '{"title":"t"}', '[]', '[]'),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4J2', '01J9Z3K5Q8R2T4V6W8X0Y2Z4P1', '01J9Z3K5Q8R2T4V6W8X0Y2Z4A6', 1, 'add', 'requirement', 'r2',
        '{"title":"t"}', '[]', '[]');
UPDATE meta SET seq_counter = 4 WHERE id = 1;`)
	return home, db
}

func open(t *testing.T, home datadir.Dir) *sqlite.DB {
	t.Helper()
	db, _, err := sqlite.Open(ctx, home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
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

func read(t *testing.T, db *sqlite.DB) scope.State {
	t.Helper()
	var st scope.State
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		var err error
		st, err = scope.Read(ctx, tx, projectA)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return st
}

// confirm checks in against the state read under the write lock and writes
// it, as scope confirm does after the person confirmed.
func confirm(db *sqlite.DB, in scope.Input) (string, error) {
	var id string
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		st, err := scope.Read(ctx, tx, projectA)
		if err != nil {
			return err
		}
		reqs, err := scope.Check(st, in, "")
		if err != nil {
			return err
		}
		seq, err := sqlite.NextSeq(ctx, tx)
		if err != nil {
			return err
		}
		id, err = scope.Write(ctx, tx, projectA, st, in, reqs, seq, "human:me", "human_cli", 1)
		return err
	})
	return id, err
}

// inputFrom is a confirmation of keys built from st, as scope show gives it.
func inputFrom(st scope.State, keys ...string) scope.Input {
	v := st.BaselineVersion
	in := scope.Input{TargetID: st.Target.ID, BaselineVersion: &v, SourceSetHash: st.SourceSetHash, Requirements: keys,
		Attestations: scope.Attestations{CompleteScope: true, PendingItemsExcluded: st.PendingItems}, AcknowledgedWaivers: []string{}}
	if st.Latest != nil {
		id := st.Latest.ID
		in.SeenScopeConfirmationID = &id
	}
	return in
}

// Read gives the state a confirmation is checked against; Write records
// one, and the next Read has it as the latest, not stale, with the
// requirements at their revisions; a new baseline version or source set
// makes it stale. The document has every field target-scope.v1 needs.
func TestReadAndWrite(t *testing.T) {
	_, db := store(t)
	st := read(t, db)
	if st.Target == nil || st.Target.ID != targetA || st.Latest != nil || st.BaselineVersion != 1 || st.PendingItems != 2 || st.Stage != "docs_only" ||
		len(st.Requirements) != 2 || len(st.Waivers) != 0 {
		t.Fatalf("state = %+v", st)
	}
	if got := strings.Join(st.MustRequirements(), " "); got != "REQ-001" {
		t.Fatalf("must = %s", got)
	}
	id, err := confirm(db, inputFrom(st, "REQ-001", "REQ-002"))
	if err != nil {
		t.Fatal(err)
	}
	st = read(t, db)
	if st.Latest == nil || st.Latest.ID != id || st.Latest.Seq != 5 || st.Stale() || len(st.Latest.Requirements) != 2 ||
		st.Latest.Requirements[0] != (scope.ConfirmedRequirement{Key: "REQ-001", Revision: 2}) || st.Latest.Attestations.PendingItemsExcluded != 2 {
		t.Fatalf("after the confirmation: %+v", st.Latest)
	}
	doc := scope.Document(st, "")
	if doc.Target == nil || doc.Target.Snapshot != nil || doc.Scope == nil || doc.Scope.ConfirmationID != id || *doc.ConfirmInputs.SeenScopeConfirmationID != id ||
		*doc.ConfirmInputs.TargetID != targetA || len(doc.ConfirmInputs.EffectiveWaivers) != 0 {
		t.Fatalf("document = %+v", doc)
	}
	write(t, db, `INSERT INTO baseline_versions (id, seq, project_id, number, summary, actor, channel)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4M2', 6, ?, 2, 'v2', 'human:me', 'human_cli'); UPDATE meta SET seq_counter = 6 WHERE id = 1`, projectA)
	if st = read(t, db); !st.Stale() || !scope.Document(st, "").Scope.Stale {
		t.Fatal("a new baseline version does not make the scope stale")
	}
	// A retired requirement cannot be in a scope.
	if _, err := confirm(db, inputFrom(st, "REQ-003")); apperr.ExitCode(err) != apperr.ExitInvalidInput {
		t.Fatalf("a retired requirement: %v", err)
	}
}

// V-T02 between two connections writing at once with the same reading:
// one confirmation is written, the other is stale with it, never written
// over it. Then a confirmation read after it is written.
func TestConfirmRace(t *testing.T) {
	home, first := store(t)
	second := open(t, home)
	in := inputFrom(read(t, first), "REQ-001")
	var wg sync.WaitGroup
	start := make(chan struct{})
	ids, errs := make([]string, 2), make([]error, 2)
	for i, db := range []*sqlite.DB{first, second} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ids[i], errs[i] = confirm(db, in)
		}()
	}
	close(start)
	wg.Wait()
	won := -1
	for i, err := range errs {
		switch {
		case err == nil:
			won = i
		case apperr.ExitCode(err) != apperr.ExitStalePrecondition:
			t.Fatalf("connection %d: %v", i, err)
		}
	}
	if won < 0 || (errs[0] == nil) == (errs[1] == nil) {
		t.Fatalf("results %v, want one written and one stale", errs)
	}
	lost := apperr.From(errs[1-won])
	if lost.Current["scope_confirmation_id"] != ids[won] {
		t.Fatalf("the stale one is told %v, want %s", lost.Current, ids[won])
	}
	if n := count(t, first, `SELECT count(*) FROM scope_confirmations`); n != 1 {
		t.Fatalf("%d confirmations written", n)
	}
	if _, err := confirm(second, inputFrom(read(t, second), "REQ-001", "REQ-002")); err != nil {
		t.Fatalf("a confirmation read after it: %v", err)
	}
}

func count(t *testing.T, db *sqlite.DB, query string) int {
	t.Helper()
	var n int
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error { return tx.QueryRowContext(ctx, query).Scan(&n) }); err != nil {
		t.Fatal(err)
	}
	return n
}

// SetTarget archives the active target and creates the new one at one seq;
// the new target has no scope; a target changed meanwhile is stale.
// codex-review-034 B2: a target name is one line of 1 to 200 characters
// without control characters or a code fence (architecture §10.2);
// Chinese, Japanese, emoji and single or double backticks are valid.
func TestCheckName(t *testing.T) {
	for _, name := range []string{"首期交付", "第一リリース", "🚀 launch", "`v1`", "``v1`` final", strings.Repeat("x", 200)} {
		if err := scope.CheckName(name); err != nil {
			t.Errorf("%q: %v", name, err)
		}
	}
	for _, name := range []string{"", " ", "a\nb", "a\x1bb", strings.Repeat("x", 201), "```", "rel ```go x``` ", "a````b", "\xff"} {
		err := scope.CheckName(name)
		if e := apperr.From(err); err == nil || e.Code != apperr.InvalidInput || len(e.Details) != 1 || e.Details[0].Pointer != "--name" {
			t.Errorf("%q: %+v", name, err)
		}
	}
}

func TestSetTarget(t *testing.T) {
	_, db := store(t)
	if _, err := confirm(db, inputFrom(read(t, db), "REQ-001")); err != nil {
		t.Fatal(err)
	}
	var made scope.Target
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		var err error
		made, err = scope.SetTarget(ctx, tx, projectA, ptr(targetA), "refs/tags/v1.0", "first release", "human:me", "human_cli")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if n := count(t, db, `SELECT count(*) FROM delivery_targets WHERE id = '`+targetA+`' AND archived_seq = `+strconv.FormatInt(made.CreatedSeq, 10)); n != 1 {
		t.Fatalf("the old target is not archived at the new one's seq %d", made.CreatedSeq)
	}
	st := read(t, db)
	if st.Target == nil || st.Target.ID != made.ID || st.Target.Ref != "refs/tags/v1.0" || st.Target.Name != "first release" || st.Latest != nil {
		t.Fatalf("after setting the target: %+v %+v", st.Target, st.Latest)
	}
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := scope.SetTarget(ctx, tx, projectA, ptr(targetA), "refs/heads/main", "main", "human:me", "human_cli")
		return err
	})
	if apperr.ExitCode(err) != apperr.ExitStalePrecondition {
		t.Fatalf("a target changed meanwhile: %v", err)
	}
	// The name is checked where it is written too.
	err = db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := scope.SetTarget(ctx, tx, projectA, ptr(made.ID), "refs/heads/main", "main ```", "human:me", "human_cli")
		return err
	})
	if apperr.ExitCode(err) != apperr.ExitInvalidInput {
		t.Fatalf("a name with a code fence: %v", err)
	}
	if n := count(t, db, `SELECT count(*) FROM delivery_targets`); n != 2 {
		t.Fatalf("%d targets, want 2", n)
	}
}
