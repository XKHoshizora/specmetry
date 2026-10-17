package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// criterionState returns a criterion's revision and requirement revision.
func criterionState(t *testing.T, dir, key string) (revision, requirementRevision int) {
	t.Helper()
	for _, r := range baselineShow(t, dir, "--include-retired").Requirements {
		for _, c := range r.Criteria {
			if c.Key == key {
				return c.Revision, c.RequirementRevision
			}
		}
	}
	t.Fatalf("no criterion %s", key)
	return 0, 0
}

// MVP-BL-12 and MVP-BL-27: a statement change is material or editorial as
// the reviewer chooses; verification, the check's command, workdir, shell
// or inputs force material whatever is chosen, recorded as material; the
// check's timeout and rationale may stay editorial. An accepted update
// without the reviewer's change_class is refused at its pointer, and the
// class the proposal suggests is never used.
func TestReviewChangeClasses(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	acceptAll(t, dir, proposeOK(t, dir, small(0, addReq("r", "sign up", nil), addAC("c", "r", "a new account can sign in"))), 0, nil)
	version := 1
	step := func(data map[string]any, chosen string, wantRevision int, wantClass string) {
		t.Helper()
		rev, _ := criterionState(t, dir, "AC-001-1")
		r := proposeOK(t, dir, small(version, update("criterion", "AC-001-1", rev, data)))
		reviewOK(t, dir, reviewDoc(r.ProposalID, version, decide(r.Items[0].ItemID, "accepted", "change_class", chosen)))
		version++
		if got, _ := criterionState(t, dir, "AC-001-1"); got != wantRevision {
			t.Errorf("%v as %s: revision %d, want %d", data, chosen, got, wantRevision)
		}
		got := rows(t, home, `SELECT i.change_class, c.change_class FROM proposal_items i JOIN baseline_changes c ON c.proposal_item_id = i.id
WHERE i.id = ?`, r.Items[0].ItemID)
		if len(got) != 1 || got[0][0] != wantClass || got[0][1] != wantClass {
			t.Errorf("%v as %s: recorded %v, want %s", data, chosen, got, wantClass)
		}
	}
	check := map[string]any{"command": "go test ./...", "rationale": "the tests sign up and sign in"}
	step(map[string]any{"statement": "a new account can sign in at once"}, "material", 2, "material")
	step(map[string]any{"statement": "a new account can sign in at once."}, "editorial", 2, "editorial")
	step(map[string]any{"verification": "automated", "check": check}, "editorial", 3, "material")
	step(map[string]any{"check": map[string]any{"command": "go test ./...", "rationale": "sign up, then sign in", "timeout_sec": 60}}, "editorial", 3, "editorial")
	for _, changed := range []map[string]any{
		{"command": "go test ./internal/...", "rationale": "x"},
		{"command": "go test ./internal/...", "rationale": "x", "workdir": "docs"},
		{"command": "go test ./internal/...", "rationale": "x", "workdir": "docs", "shell": "bash"},
		{"command": "go test ./internal/...", "rationale": "x", "workdir": "docs", "shell": "bash", "inputs": []any{"README.md"}},
	} {
		rev, _ := criterionState(t, dir, "AC-001-1")
		step(map[string]any{"check": changed}, "editorial", rev+1, "material")
	}
	if b := baselineShow(t, dir); b.Requirements[0].Criteria[0].Verification != "automated" {
		t.Fatalf("criterion = %+v", b.Requirements[0].Criteria[0])
	}
	var doc struct {
		Requirements []struct {
			Criteria []struct {
				CheckConfigHash *string `json:"check_config_hash"`
			} `json:"criteria"`
		} `json:"requirements"`
	}
	_, stdout, _ := specmetry(t, dir, "baseline-show.v1", "baseline", "show", "--json")
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil || doc.Requirements[0].Criteria[0].CheckConfigHash == nil {
		t.Fatalf("check_config_hash missing: %v %s", err, stdout)
	}

	// V-R01: no class, also when the proposal suggests one, and also for an
	// edited update.
	rev, _ := criterionState(t, dir, "AC-001-1")
	suggested := update("criterion", "AC-001-1", rev, map[string]any{"statement": "changed"})
	suggested["change_class"] = "editorial"
	r := proposeOK(t, dir, small(version, suggested))
	reviewRefused(t, home, dir, reviewDoc(r.ProposalID, version, decide(r.Items[0].ItemID, "accepted")), apperr.ExitInvalidInput,
		"/items/0/change_class")
	reviewRefused(t, home, dir, reviewDoc(r.ProposalID, version, decide(r.Items[0].ItemID, "accepted_with_edits",
		"final_data", map[string]any{"statement": "changed again"})), apperr.ExitInvalidInput, "/items/0/change_class")
	reviewRefused(t, home, dir, reviewDoc(r.ProposalID, version, decide(r.Items[0].ItemID, "rejected"), decide(r.Items[0].ItemID, "rejected")),
		apperr.ExitInvalidInput, "/items/1/item_id")
}

// MVP-BL-14: an item whose target changed after the proposal's base
// version, by an editorial change at the same revision too, is in conflict:
// proposal show gives the target's values then, now and as proposed;
// deciding it needs acknowledged_conflict. A target retired meanwhile can
// only be rejected.
func TestReviewConflicts(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	acceptAll(t, dir, proposeOK(t, dir, small(0, addReq("a", "sign up", nil), addReq("b", "export", nil))), 0, nil)
	first := proposeOK(t, dir, small(1, update("requirement", "REQ-001", 1, map[string]any{"title": "sign up with e-mail"})))
	second := proposeOK(t, dir, small(1, update("requirement", "REQ-001", 1, map[string]any{"priority": "should"}),
		update("requirement", "REQ-002", 1, map[string]any{"title": "export as CSV"})))
	third := proposeOK(t, dir, small(1, retire("requirement", "REQ-002", 1)))
	acceptAll(t, dir, first, 1, nil) // editorial: REQ-001 stays at revision 1

	show := showProposal(t, dir, second.ProposalID)
	c := show.Items[0].Conflict
	if show.Items[0].ReviewState != "conflict" || c == nil || c.BaseRevision != 1 || c.CurrentRevision != 1 ||
		c.Base["title"] != "sign up" || c.Current["title"] != "sign up with e-mail" || c.Proposed["priority"] != "should" {
		t.Fatalf("item 0 = %+v, conflict %+v", show.Items[0], c)
	}
	if show.Items[1].ReviewState != "pending" || show.Items[1].Conflict != nil {
		t.Fatalf("item 1 = %+v", show.Items[1])
	}
	if got := rows(t, home, `SELECT review_state FROM proposal_items WHERE id = ?`, second.Items[0].ItemID); got[0][0] != "conflict" {
		t.Fatalf("stored state %v", got)
	}
	reviewRefused(t, home, dir, reviewDoc(second.ProposalID, 2, decide(second.Items[0].ItemID, "accepted", "change_class", "editorial")),
		apperr.ExitInvalidInput, "/items/0/acknowledged_conflict")
	reviewRefused(t, home, dir, reviewDoc(second.ProposalID, 2, decide(second.Items[0].ItemID, "rejected")),
		apperr.ExitInvalidInput, "/items/0/acknowledged_conflict")
	reviewOK(t, dir, reviewDoc(second.ProposalID, 2,
		decide(second.Items[0].ItemID, "accepted", "change_class", "editorial", "acknowledged_conflict", true)))
	if r := baselineShow(t, dir).Requirements[0]; r.Title != "sign up with e-mail" {
		t.Fatalf("REQ-001 = %+v", r)
	}

	// REQ-002 retired: an update of it can only be rejected.
	acceptAll(t, dir, third, 3, nil)
	show = showProposal(t, dir, second.ProposalID)
	if c := show.Items[1].Conflict; c == nil || c.CurrentStatus != "retired" {
		t.Fatalf("item 1 = %+v", show.Items[1])
	}
	reviewRefused(t, home, dir, reviewDoc(second.ProposalID, 4,
		decide(second.Items[1].ItemID, "accepted", "change_class", "editorial", "acknowledged_conflict", true)), apperr.ExitInvalidInput, "/items/0")
	reviewOK(t, dir, reviewDoc(second.ProposalID, 4, decide(second.Items[1].ItemID, "rejected", "acknowledged_conflict", true)))
}

// MVP-BL-19, BL-20, BL-24: the fixed order of application. A material
// requirement revision makes its criteria parent_revised; follow-ups in the
// same review reaffirm, retire or defer them; a reaffirm item and an
// accepted criterion update refresh the requirement revision, and a new
// criterion takes the new one. A deferred criterion stays behind until an
// Agent's reaffirm is accepted, which leaves its own revision. All items
// rejected: no version.
func TestReviewApplicationOrder(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	acceptAll(t, dir, proposeOK(t, dir, small(0, addReq("r", "sign up", nil), addAC("c1", "r", "one"), addAC("c2", "r", "two"),
		addAC("c3", "r", "three"), addAC("c4", "r", "four"), addAC("c5", "r", "five"))), 0, nil)
	r := proposeOK(t, dir, small(1, update("requirement", "REQ-001", 1, map[string]any{"title": "sign up with e-mail"}),
		reaffirm("AC-001-1", 1), addAC("c6", "REQ-001", "six"), update("criterion", "AC-001-2", 1, map[string]any{"statement": "two."})))
	ids := itemIDs(r)
	doc := reviewDoc(r.ProposalID, 1, decide(ids["REQ-001"], "accepted", "change_class", "material"), decide(ids["AC-001-1"], "accepted"),
		decide(ids["c6"], "accepted"), decide(ids["AC-001-2"], "accepted", "change_class", "editorial"))
	doc["criterion_followups"] = []any{
		map[string]any{"criterion": "AC-001-3", "seen_revision": 1, "action": "reaffirm"},
		map[string]any{"criterion": "AC-001-4", "seen_revision": 1, "action": "retire", "rationale": "covered by AC-001-6"},
		map[string]any{"criterion": "AC-001-5", "seen_revision": 1, "action": "defer"},
	}
	// Follow-ups need a material revision of their requirement, the
	// revision read, once each, and no other change to the criterion.
	for _, bad := range []struct {
		followup map[string]any
		code     int
		pointer  string
	}{
		{map[string]any{"criterion": "AC-001-3", "seen_revision": 2, "action": "reaffirm"}, apperr.ExitStalePrecondition, "/criterion_followups/0/seen_revision"},
		{map[string]any{"criterion": "AC-001-1", "seen_revision": 1, "action": "retire", "rationale": "x"}, apperr.ExitInvalidInput, "/criterion_followups/0/criterion"},
		{map[string]any{"criterion": "AC-009-1", "seen_revision": 1, "action": "defer"}, apperr.ExitInvalidInput, "/criterion_followups/0/criterion"},
	} {
		b := reviewDoc(r.ProposalID, 1, decide(ids["REQ-001"], "accepted", "change_class", "material"), decide(ids["AC-001-1"], "accepted"))
		b["criterion_followups"] = []any{bad.followup}
		reviewRefused(t, home, dir, b, bad.code, bad.pointer)
	}
	editorial := reviewDoc(r.ProposalID, 1, decide(ids["REQ-001"], "accepted", "change_class", "editorial"))
	editorial["criterion_followups"] = []any{map[string]any{"criterion": "AC-001-3", "seen_revision": 1, "action": "reaffirm"}}
	reviewRefused(t, home, dir, editorial, apperr.ExitInvalidInput, "/criterion_followups/0/criterion")

	reviewOK(t, dir, doc)
	b := baselineShow(t, dir, "--include-retired")
	if b.Requirements[0].Revision != 2 {
		t.Fatalf("REQ-001 = %+v", b.Requirements[0])
	}
	got := map[string][3]any{}
	for _, c := range b.Requirements[0].Criteria {
		got[c.Key] = [3]any{c.Revision, c.RequirementRevision, c.Status}
	}
	want := map[string][3]any{
		"AC-001-1": {1, 2, "active"}, "AC-001-2": {1, 2, "active"}, "AC-001-3": {1, 2, "active"},
		"AC-001-4": {1, 1, "retired"}, "AC-001-5": {1, 1, "active"}, "AC-001-6": {1, 2, "active"},
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s = %v, want %v (revision, requirement revision, status)", k, got[k], w)
		}
	}
	if got := rows(t, home, `SELECT op, rationale FROM baseline_changes WHERE proposal_item_id IS NULL ORDER BY change_index`); !equalRows(got,
		[][]string{{"reaffirm", "NULL"}, {"retire", "covered by AC-001-6"}}) {
		t.Fatalf("follow-up changes = %v", got)
	}

	// BL-20: the Agent proposes to reaffirm the deferred criterion.
	agent := proposeOK(t, dir, small(2, reaffirm("AC-001-5", 1)))
	reviewOK(t, dir, reviewDoc(agent.ProposalID, 2, decide(agent.Items[0].ItemID, "accepted")))
	if rev, reqRev := criterionState(t, dir, "AC-001-5"); rev != 1 || reqRev != 2 {
		t.Fatalf("AC-001-5: revision %d, requirement revision %d", rev, reqRev)
	}

	// All rejected: no version.
	before := count(t, home, `SELECT count(*) FROM baseline_versions`)
	rejected := proposeOK(t, dir, small(3, addReq("x", "x", nil), addReq("y", "y", nil)))
	out := reviewOK(t, dir, reviewDoc(rejected.ProposalID, 3, decide(rejected.Items[0].ItemID, "rejected"), decide(rejected.Items[1].ItemID, "rejected")))
	if n := count(t, home, `SELECT count(*) FROM baseline_versions`); n != before || !strings.Contains(out, "no baseline version") {
		t.Fatalf("versions %d -> %d\n%s", before, n, out)
	}
}

// MVP-BL-25 (M1): a question about an item of its proposal is related to
// the entity the item becomes when accepted; one about a rejected item
// keeps naming the item.
func TestReviewRelatesQuestions(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	doc := small(0, addReq("r1", "sign up", nil), addReq("r2", "export", nil))
	doc["questions"] = []any{
		map[string]any{"text": "e-mail or phone?", "about": "r1", "blocking": true},
		map[string]any{"text": "which format?", "about": "r2", "blocking": true},
	}
	r := proposeOK(t, dir, doc)
	ids := itemIDs(r)
	reviewOK(t, dir, reviewDoc(r.ProposalID, 0, decide(ids["r1"], "accepted"), decide(ids["r2"], "rejected")))
	got := rows(t, home, `SELECT q.key, q.related_type, coalesce(req.key, 'NULL'), q.status FROM questions q
LEFT JOIN requirements req ON req.id = q.related_id ORDER BY q.key`)
	if want := [][]string{{"Q-001", "requirement", "REQ-001", "open"}, {"Q-002", "NULL", "NULL", "open"}}; !equalRows(got, want) {
		t.Fatalf("questions = %v, want %v", got, want)
	}
	about := map[string]string{}
	for _, q := range showProposal(t, dir, r.ProposalID).Questions {
		about[q.Key] = *q.About
	}
	if about["Q-001"] != "REQ-001" || about["Q-002"] != "r2" {
		t.Fatalf("about = %v", about)
	}
}

// A review is refused, with nothing written: a delivery scope (WP1.8), an
// item of another proposal or reviewed already, edits to a retire item,
// edits that are not valid data, a proposal that does not exist, and a
// person who does not confirm (exit 3).
func TestReviewRefusals(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	r := proposeOK(t, dir, small(0, addReq("r1", "sign up", nil), addReq("r2", "export", nil)))
	other := proposeOK(t, dir, small(0, addReq("r1", "other", nil)))
	ids := itemIDs(r)
	// A scope for another target (WP1.8; TestScopeInReview has the rest).
	scope := reviewDoc(r.ProposalID, 0, decide(ids["r1"], "accepted"))
	scope["scope"] = map[string]any{"target_id": r.ProposalID, "source_set_hash": strings.Repeat("0", 64), "requirements": []any{"REQ-001"},
		"attestations": map[string]any{"complete_scope": true, "pending_items_excluded": 0}, "acknowledged_waivers": []any{},
		"seen_scope_confirmation_id": nil}
	reviewRefused(t, home, dir, scope, apperr.ExitStalePrecondition, "/scope/target_id")
	reviewRefused(t, home, dir, reviewDoc(r.ProposalID, 0, decide(other.Items[0].ItemID, "accepted")), apperr.ExitInvalidInput, "/items/0/item_id")
	reviewRefused(t, home, dir, reviewDoc(r.ProposalID, 0, decide(ids["r1"], "accepted_with_edits", "final_data", map[string]any{"title": "a\nb"})),
		apperr.ExitInvalidInput, "/items/0/final_data/title")
	reviewRefused(t, home, dir, reviewDoc(r.ProposalID, 0, decide(ids["r1"], "accepted_with_edits",
		"final_data", map[string]any{"title": "x", "kind": "functional", "priority": "must", "goal": "zz"})),
		apperr.ExitInvalidInput, "/items/0/final_data/goal")
	reviewRefused(t, home, dir, reviewDoc(r.ProposalID, 0, decide(ids["r1"], "accepted", "final_data", map[string]any{"title": "x"})),
		apperr.ExitInvalidInput, "/items/0/final_data")
	reviewRefused(t, home, dir, reviewDoc("01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", 0, decide(ids["r1"], "accepted")), apperr.ExitInvalidInput, "/proposal_id")

	before := state(t, home)
	code, stdout, stderr := asHuman(t, dir, "", "review", "--file", writeReview(t, reviewDoc(r.ProposalID, 0, decide(ids["r1"], "accepted"))))
	if code != apperr.ExitHumanRequired || state(t, home) != before {
		t.Fatalf("declined: exit %d\n%s%s", code, stdout, stderr)
	}
	reviewOK(t, dir, reviewDoc(r.ProposalID, 0, decide(ids["r1"], "accepted")))
	reviewRefused(t, home, dir, reviewDoc(r.ProposalID, 1, decide(ids["r1"], "rejected")), apperr.ExitInvalidInput, "/items/0/item_id")
	retirement := proposeOK(t, dir, small(1, retire("requirement", "REQ-001", 1)))
	reviewRefused(t, home, dir, reviewDoc(retirement.ProposalID, 1, decide(retirement.Items[0].ItemID, "accepted_with_edits",
		"final_data", map[string]any{"title": "x"})), apperr.ExitInvalidInput, "/items/0/decision")
}

// Any failure rolls everything back: keys handed out before it are not
// used up, and the next review gives the same keys.
func TestReviewRollsBack(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	acceptAll(t, dir, proposeOK(t, dir, small(0, addReq("a", "a", nil))), 0, nil)
	r := proposeOK(t, dir, small(1, addGoal("g", "launch"), addReq("b", "b", "g"), addAC("c", "b", "one"), addAC("d", "REQ-001", "late")))
	// REQ-001 is retired before the review: the last item cannot apply,
	// after keys were handed out for the others.
	acceptAll(t, dir, proposeOK(t, dir, small(1, retire("requirement", "REQ-001", 1))), 1, nil)
	ids := itemIDs(r)
	reviewRefused(t, home, dir, reviewDoc(r.ProposalID, 2, decide(ids["g"], "accepted"), decide(ids["b"], "accepted"),
		decide(ids["c"], "accepted"), decide(ids["d"], "accepted")), apperr.ExitInvalidInput, "/items/3")
	reviewOK(t, dir, reviewDoc(r.ProposalID, 2, decide(ids["g"], "accepted"), decide(ids["b"], "accepted"),
		decide(ids["c"], "accepted"), decide(ids["d"], "rejected")))
	if got := baselineShow(t, dir).keys(); got != "GOAL-1 REQ-002 AC-002-1" {
		t.Fatalf("baseline = %s", got)
	}
}

// MVP-BL-22: a material update without new references carries the
// previous revision's over, checked again, as new rows of the new
// revision; the old rows stay. hash_only: a source still with the hash it
// was checked against keeps its result; a changed one is not_checkable.
// An editorial update without references adds none.
func TestReviewCarriesReferences(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	add := addReq("r", "log in", nil)
	add["sources"] = []any{map[string]any{"source": "docs/client-brief.md", "locator": "R1", "excerpt": "连续 5 次输错密码后，账号锁定 15 分钟。"}}
	add["code_refs"] = []any{map[string]any{"path": "README.md"}}
	acceptAll(t, dir, proposeOK(t, dir, small(0, add)), 0, nil)
	refs := func() [][]string {
		return rows(t, home, `SELECT 'source', entity_revision, verification FROM source_refs WHERE entity_type = 'requirement'
UNION ALL SELECT 'code', entity_revision, "exists" FROM code_refs ORDER BY 1, 2`)
	}
	if got := refs(); !equalRows(got, [][]string{{"code", "1", "1"}, {"source", "1", "verified"}}) {
		t.Fatalf("refs = %v", got)
	}
	acceptAll(t, dir, proposeOK(t, dir, small(1, update("requirement", "REQ-001", 1, map[string]any{"title": "log in with a password"}))),
		1, map[string]string{"REQ-001": "material"})
	if got := refs(); !equalRows(got, [][]string{{"code", "1", "1"}, {"code", "2", "1"}, {"source", "1", "verified"}, {"source", "2", "verified"}}) {
		t.Fatalf("refs after a material update = %v", got)
	}
	acceptAll(t, dir, proposeOK(t, dir, small(2, update("requirement", "REQ-001", 2, map[string]any{"title": "log in with a password."}))), 2, nil)
	if n := len(refs()); n != 4 {
		t.Fatalf("an editorial update added references: %v", refs())
	}
	brief := filepath.Join(dir, "docs", "client-brief.md")
	if err := os.WriteFile(brief, []byte("# brief, revised\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	acceptAll(t, dir, proposeOK(t, dir, small(3, update("requirement", "REQ-001", 2, map[string]any{"priority": "should"}))),
		3, map[string]string{"REQ-001": "material"})
	if got := refs(); got[len(got)-1][1] != "3" || got[len(got)-1][2] != "not_checkable" {
		t.Fatalf("refs after the source changed = %v", got)
	}
}

// hash_only (PD-24, MVP-PRV-08): reviewing never stores or prints an
// excerpt: the entity's references keep its hash only.
func TestReviewNeverStoresExcerpts(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	canary := testutil.NewCanary(t)
	add := addReq("r", "log in", nil)
	add["sources"] = []any{map[string]any{"source": "docs/client-brief.md", "excerpt": "客户口令规则：" + canary}}
	r := proposeOK(t, dir, small(0, add))
	out := acceptAll(t, dir, r, 0, nil)
	_, show, _ := specmetry(t, dir, "proposal-show.v1", "proposal", "show", r.ProposalID, "--json")
	_, history, _ := specmetry(t, dir, "baseline-history.v1", "baseline", "history", "--json")
	for name, text := range map[string]string{"review": out, "proposal show": show, "baseline history": history} {
		if strings.Contains(text, canary) {
			t.Errorf("%s prints the excerpt", name)
		}
	}
	if got := rows(t, home, `SELECT count(*) FROM source_refs WHERE excerpt IS NULL AND entity_type = 'requirement'`); got[0][0] != "1" {
		t.Fatalf("references = %v", got)
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the excerpt reached the data directory: %v", hits)
	}
}

// Retiring a goal clears its requirements' goal in the same version (data-
// model §5.2): an editorial change of each, their revision kept, recorded
// as a cascade.
func TestReviewRetiresAGoal(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	acceptAll(t, dir, proposeOK(t, dir, small(0, addGoal("g", "launch"), addReq("a", "sign up", "g"), addReq("b", "export", "g"))), 0, nil)
	acceptAll(t, dir, proposeOK(t, dir, small(1, retire("goal", "GOAL-1", 1))), 1, nil)
	for _, r := range baselineShow(t, dir).Requirements {
		if r.Goal != nil || r.Revision != 1 {
			t.Errorf("%s: goal %v, revision %d", r.Key, r.Goal, r.Revision)
		}
	}
	got := rows(t, home, `SELECT c.op, c.entity_type, coalesce(c.change_class, ''), c.proposal_item_id IS NULL FROM baseline_changes c
JOIN baseline_versions v ON v.id = c.version_id WHERE v.number = 2 ORDER BY c.change_index`)
	want := [][]string{{"retire", "goal", "", "0"}, {"update", "requirement", "editorial", "1"}, {"update", "requirement", "editorial", "1"}}
	if !equalRows(got, want) {
		t.Fatalf("changes = %v, want %v", got, want)
	}
}
