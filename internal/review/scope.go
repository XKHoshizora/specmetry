package review

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/scope"
	"github.com/XKHoshizora/specmetry/internal/ulid"
)

// ScopeInput is the delivery scope the review confirms, nil without one.
// Its requirements are as written: keys, or item IDs of the review.
func (d Document) ScopeInput() (*scope.Input, error) {
	if d.Scope == nil || string(d.Scope) == "null" {
		return nil, nil
	}
	var in scope.Input
	if err := json.Unmarshal(d.Scope, &in); err != nil {
		return nil, apperr.New(apperr.InvalidInput, "the review's scope is not valid").WithDetails(apperr.Detail{Pointer: "/scope", Problem: "cannot be read"})
	}
	return &in, nil
}

// ScopeResult is the scope a review confirmed.
type ScopeResult struct {
	ID              string
	BaselineVersion int
	Requirements    []scope.Requirement
	PendingItems    int
}

// ConfirmScope confirms the delivery scope of a review that was just
// applied (baseline.Apply with plan gave result), in the same write
// transaction and at the review's seq: data-model §5.5 step 11. A
// requirement written as an item ID must be an add requirement item this
// review accepted, and stands for the key it was given; any other ID is
// invalid input. The scope is then checked like a standalone one, against
// the state the review leaves: it binds to the version the review made
// (the current one when it accepted nothing), and pending_items_excluded
// and the waivers are those after the review. Any problem is returned, and
// the caller's transaction, the review with it, is rolled back.
func ConfirmScope(ctx context.Context, tx *sql.Tx, projectID string, in scope.Input, plan baseline.Plan, result baseline.Result,
	confirmedAt int64) (ScopeResult, error) {
	added := map[string]bool{}
	for _, c := range plan.Accepted {
		if c.Op == "add" && c.Type == baseline.Requirement {
			added[c.ItemID] = true
		}
	}
	resolved := in
	resolved.BaselineVersion = nil
	resolved.Requirements = make([]string, len(in.Requirements))
	var problems []apperr.Detail
	for i, r := range in.Requirements {
		if !ulid.Valid(r) {
			resolved.Requirements[i] = r
			continue
		}
		key, ok := result.Keys[r]
		if !added[r] || !ok {
			problems = append(problems, apperr.Detail{Pointer: "/scope/requirements/" + strconv.Itoa(i),
				Problem: "is not an add requirement item this review accepts"})
			continue
		}
		resolved.Requirements[i] = key
	}
	if len(problems) > 0 {
		return ScopeResult{}, apperr.New(apperr.InvalidInput, "the review's scope is not valid; nothing was written").WithDetails(problems...)
	}
	st, err := scope.Read(ctx, tx, projectID)
	if err != nil {
		return ScopeResult{}, err
	}
	reqs, err := scope.Check(st, resolved, "/scope")
	if err != nil {
		return ScopeResult{}, err
	}
	id, err := scope.Write(ctx, tx, projectID, st, resolved, reqs, plan.Seq, plan.Actor, plan.Channel, confirmedAt)
	if err != nil {
		return ScopeResult{}, err
	}
	return ScopeResult{ID: id, BaselineVersion: st.BaselineVersion, Requirements: reqs, PendingItems: st.PendingItems}, nil
}
