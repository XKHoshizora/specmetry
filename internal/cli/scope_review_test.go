package cli

import (
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// reviewScope is a review's scope from what ts says a confirmation must
// carry, with pending items excluded and requirements (keys or item IDs).
func reviewScope(ts targetScopeOut, pending int, requirements ...any) map[string]any {
	s := scopeFields(ts, requirements...)
	s["attestations"].(map[string]any)["pending_items_excluded"] = pending
	return s
}

// MVP-TGT-07, and TGT-10 and TGT-11 in a review: the first review accepts
// three new requirements and confirms the scope by their item IDs, in the
// same transaction and at the same seq: the IDs resolve to the new keys,
// the scope binds to the new version and excludes the items left pending
// after the review. The whole review fails, nothing written, for a scope
// naming an item the review rejects or no add requirement item of it,
// excluding the items pending before the review, acknowledging a waiver
// not in effect, naming a stale confirmation or none when one exists, or a
// stale source set. A later review with existing keys and an item ID
// replaces the confirmation; a review accepting nothing binds its scope to
// the current version; a review without a scope leaves the confirmation,
// which goes stale. A review without a scope works as before.
func TestScopeInReview(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	r := proposeOK(t, dir, small(0, addGoal("g", "launch"), addReq("a", "sign up", "g"), addReq("b", "export", "g"), addReq("c", "audit log", "g"),
		addReq("d", "import", "g"), addReq("e", "reports", nil)))
	ids := itemIDs(r)
	ts := targetScope(t, dir)
	decisions := []map[string]any{decide(ids["g"], "accepted"), decide(ids["a"], "accepted"), decide(ids["b"], "accepted"),
		decide(ids["c"], "accepted"), decide(ids["d"], "rejected")}
	withScope := func(scope map[string]any) map[string]any {
		doc := reviewDoc(r.ProposalID, 0, decisions...)
		doc["scope"] = scope
		return doc
	}
	// e is left pending: one item after the review, six before.
	reviewRefused(t, home, dir, withScope(reviewScope(ts, 1, ids["a"], ids["b"], ids["d"])), apperr.ExitInvalidInput, "/scope/requirements/2")
	reviewRefused(t, home, dir, withScope(reviewScope(ts, 1, ids["a"], ids["g"])), apperr.ExitInvalidInput, "/scope/requirements/1")
	reviewRefused(t, home, dir, withScope(reviewScope(ts, 1, ids["a"], ids["e"])), apperr.ExitInvalidInput, "/scope/requirements/1")
	reviewRefused(t, home, dir, withScope(reviewScope(ts, 1, ids["a"], "REQ-009")), apperr.ExitInvalidInput, "/scope/requirements/1")
	env := reviewRefused(t, home, dir, withScope(reviewScope(ts, 6, ids["a"], ids["b"], ids["c"])), apperr.ExitStalePrecondition,
		"/scope/attestations/pending_items_excluded")
	if env.Error.Current["pending_items"] != float64(1) {
		t.Fatalf("current = %v", env.Error.Current)
	}
	waived := reviewScope(ts, 1, ids["a"])
	waived["acknowledged_waivers"] = []any{"01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"}
	reviewRefused(t, home, dir, withScope(waived), apperr.ExitStalePrecondition, "/scope/acknowledged_waivers")
	seen := reviewScope(ts, 1, ids["a"])
	seen["seen_scope_confirmation_id"] = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"
	reviewRefused(t, home, dir, withScope(seen), apperr.ExitStalePrecondition, "/scope/seen_scope_confirmation_id")
	hash := reviewScope(ts, 1, ids["a"])
	hash["source_set_hash"] = strings.Repeat("0", 64)
	reviewRefused(t, home, dir, withScope(hash), apperr.ExitStalePrecondition, "/scope/source_set_hash")
	other := reviewScope(ts, 1, ids["a"])
	other["target_id"] = r.ProposalID
	reviewRefused(t, home, dir, withScope(other), apperr.ExitStalePrecondition, "/scope/target_id")

	out := reviewOK(t, dir, withScope(reviewScope(ts, 1, ids["c"], ids["a"], ids["b"])))
	if !strings.Contains(out, "delivery scope: 3 requirements at baseline version 1: REQ-001, REQ-002, REQ-003") ||
		!strings.Contains(out, "1 proposal items pending or in conflict are not in it") ||
		!strings.Contains(out, "Delivery scope confirmed: 3 requirements at baseline version 1.") {
		t.Fatalf("review:\n%s", out)
	}
	c1 := targetScope(t, dir)
	if c1.scoped() != "REQ-001@1 REQ-002@1 REQ-003@1" || c1.Scope.BaselineVersion != 1 || c1.Scope.Attestations["pending_items_excluded"] != float64(1) ||
		c1.Scope.Stale {
		t.Fatalf("scope = %+v", c1.Scope)
	}
	got := rows(t, home, `SELECT c.seq = v.seq, c.baseline_version, c.channel, c.actor LIKE 'human:_%', coalesce(c.seen_scope_confirmation_id, '-')
FROM scope_confirmations c JOIN baseline_versions v ON v.number = c.baseline_version AND v.project_id = c.project_id`)
	if want := [][]string{{"1", "1", "human_cli", "1", "-"}}; !equalRows(got, want) {
		t.Fatalf("confirmation = %v, want %v (the review's seq)", got, want)
	}

	// A later review: an existing key and an item ID; no confirmation
	// named while one exists is stale.
	later := proposeOK(t, dir, small(1, addReq("f", "billing", nil)))
	fresh := targetScope(t, dir)
	laterDoc := func(scope map[string]any) map[string]any {
		doc := reviewDoc(later.ProposalID, 1, decide(later.Items[0].ItemID, "accepted"))
		doc["scope"] = scope
		return doc
	}
	none := reviewScope(fresh, 1, "REQ-001", later.Items[0].ItemID)
	none["seen_scope_confirmation_id"] = nil
	env = reviewRefused(t, home, dir, laterDoc(none), apperr.ExitStalePrecondition, "/scope/seen_scope_confirmation_id")
	if env.Error.Current["scope_confirmation_id"] != c1.Scope.ConfirmationID {
		t.Fatalf("current = %v", env.Error.Current)
	}
	reviewOK(t, dir, laterDoc(reviewScope(fresh, 1, "REQ-001", later.Items[0].ItemID)))
	c2 := targetScope(t, dir)
	if c2.scoped() != "REQ-001@1 REQ-004@1" || c2.Scope.BaselineVersion != 2 {
		t.Fatalf("scope = %+v", c2.Scope)
	}

	// Nothing accepted: the scope binds to the current version.
	nothing := proposeOK(t, dir, small(2, addReq("h", "help", nil)))
	doc := reviewDoc(nothing.ProposalID, 2, decide(nothing.Items[0].ItemID, "rejected"))
	doc["scope"] = reviewScope(c2, 1, "REQ-002")
	out = reviewOK(t, dir, doc)
	if !strings.Contains(out, "Delivery scope confirmed: 1 requirements at baseline version 2.") {
		t.Fatalf("review:\n%s", out)
	}
	c3 := targetScope(t, dir)
	if c3.scoped() != "REQ-002@1" || c3.Scope.BaselineVersion != 2 || c3.ConfirmInputs.BaselineVersion != 2 {
		t.Fatalf("scope = %+v", c3.Scope)
	}

	// A review without a scope: as before; the confirmation goes stale.
	acceptAll(t, dir, proposeOK(t, dir, small(2, addReq("i", "search", nil))), 2, nil)
	if got := targetScope(t, dir); got.Scope.ConfirmationID != c3.Scope.ConfirmationID || !got.Scope.Stale || got.ConfirmInputs.BaselineVersion != 3 {
		t.Fatalf("after a review without a scope: %+v", got.Scope)
	}
}
