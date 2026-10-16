package scope

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"strconv"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/ulid"
)

// IncompleteDocs is the stage whose scope needs missing_goals_reviewed.
const IncompleteDocs = "incomplete_docs"

// Input is a scope confirmation as a person submits it: a scope.v1
// document, or the scope of a review.v1 with its item IDs resolved to
// keys.
type Input struct {
	TargetID string `json:"target_id"`
	// BaselineVersion is the version the person read; nil in a review,
	// where the confirmation binds to the version the review leaves
	// (data-model §5.5 step 11).
	BaselineVersion         *int         `json:"baseline_version"`
	SourceSetHash           string       `json:"source_set_hash"`
	Requirements            []string     `json:"requirements"`
	Attestations            Attestations `json:"attestations"`
	AcknowledgedWaivers     []string     `json:"acknowledged_waivers"`
	SeenScopeConfirmationID *string      `json:"seen_scope_confirmation_id"`
	// Note is nil without one.
	Note *string `json:"note,omitempty"`
}

// Document is in as the scope.v1 document it stands for, to be checked
// against that schema: a confirmation built from the command line is
// checked like one from a file (codex-review-034 B1). in carries the
// baseline version it was read at.
func (in Input) Document() ([]byte, error) {
	return json.Marshal(struct {
		Schema string `json:"schema"`
		Input
	}{"specmetry.scope/v1", in})
}

// Check checks in against the current state (data-model §7.1), with the
// JSON pointers of its problems under base ("" for scope.v1, "/scope" in a
// review), and returns the confirmed requirements in key order.
//
// The values read are compared first, and any that moved on is
// stale_precondition (exit code 5) with the current value: the target, the
// baseline version, the source set hash, the target's current confirmation
// (V-T02) and the count of items still pending or in conflict. Then the
// requirements must be active and the attestations complete, or it is
// invalid_input (exit code 2). Last, acknowledged_waivers must be exactly
// the waivers in effect under the confirmed requirements (V-T01), or it is
// stale with that set.
func Check(st State, in Input, base string) ([]Requirement, error) {
	if st.Target == nil || in.TargetID != st.Target.ID {
		return nil, stale("the delivery target is not the one the scope was read for; read it again (scope show)", base+"/target_id",
			map[string]any{"target_id": targetID(st.Target)})
	}
	if in.BaselineVersion != nil && *in.BaselineVersion != st.BaselineVersion {
		return nil, stale("the baseline version is "+strconv.Itoa(st.BaselineVersion)+", not the one the scope was read at; read it again (scope show)",
			base+"/baseline_version", map[string]any{"baseline_version": st.BaselineVersion})
	}
	if st.BaselineVersion < 1 {
		return nil, apperr.New(apperr.InvalidInput, "there is no baseline version yet: a scope holds confirmed requirements").
			WithDetails(apperr.Detail{Pointer: base + "/requirements", Problem: "has no baseline version to be confirmed against"})
	}
	if in.SourceSetHash != st.SourceSetHash {
		return nil, stale("the source set changed since the scope was read; read it again (scope show)", base+"/source_set_hash",
			map[string]any{"source_set_hash": st.SourceSetHash})
	}
	if !sameConfirmation(st.Latest, in.SeenScopeConfirmationID) {
		return nil, stale("the target's scope was confirmed meanwhile; read it again (scope show) and confirm again", base+"/seen_scope_confirmation_id",
			map[string]any{"scope_confirmation_id": confirmationID(st.Latest)})
	}
	if in.Attestations.PendingItemsExcluded != st.PendingItems {
		return nil, stale(strconv.Itoa(st.PendingItems)+" proposal items are pending or in conflict now, not the number the scope excludes; read it again (scope show)",
			base+"/attestations/pending_items_excluded", map[string]any{"pending_items": st.PendingItems})
	}

	var problems []apperr.Detail
	reqs := make([]Requirement, 0, len(in.Requirements))
	seen := map[string]bool{}
	for i, key := range in.Requirements {
		r, ok := st.Requirements[key]
		switch {
		case !ok:
			problems = append(problems, apperr.Detail{Pointer: base + "/requirements/" + strconv.Itoa(i), Problem: "is not an active requirement"})
		case seen[key]:
			problems = append(problems, apperr.Detail{Pointer: base + "/requirements/" + strconv.Itoa(i), Problem: "is listed twice"})
		default:
			seen[key] = true
			reqs = append(reqs, r)
		}
	}
	if len(in.Requirements) == 0 {
		problems = append(problems, apperr.Detail{Pointer: base + "/requirements", Problem: "is empty; a scope holds at least one requirement"})
	}
	if !in.Attestations.CompleteScope {
		problems = append(problems, apperr.Detail{Pointer: base + "/attestations/complete_scope", Problem: "must be true"})
	}
	if st.Stage == IncompleteDocs && (in.Attestations.MissingGoalsReviewed == nil || !*in.Attestations.MissingGoalsReviewed) {
		problems = append(problems, apperr.Detail{Pointer: base + "/attestations/missing_goals_reviewed",
			Problem: "must be true for a project at stage incomplete_docs: the goals the code cannot reveal have been added"})
	}
	if len(problems) > 0 {
		return nil, apperr.New(apperr.InvalidInput, "the scope is not valid; nothing was written").WithDetails(problems...)
	}
	slices.SortFunc(reqs, func(a, b Requirement) int { return compareKeys(a.Key, b.Key) })

	keys := make([]string, len(reqs))
	for i, r := range reqs {
		keys[i] = r.Key
	}
	effective := st.WaiversUnder(keys)
	want := make([]string, len(effective))
	for i, w := range effective {
		want[i] = w.DecisionID
	}
	got := slices.Clone(in.AcknowledgedWaivers)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		return nil, stale("acknowledged_waivers is not exactly the waivers in effect under the confirmed requirements; read them again (scope show)",
			base+"/acknowledged_waivers", map[string]any{"effective_waivers": effective})
	}
	return reqs, nil
}

func sameConfirmation(latest *Confirmation, seen *string) bool {
	if latest == nil || seen == nil {
		return latest == nil && seen == nil
	}
	return latest.ID == *seen
}

func targetID(t *Target) any {
	if t == nil {
		return nil
	}
	return t.ID
}

func confirmationID(c *Confirmation) any {
	if c == nil {
		return nil
	}
	return c.ID
}

func stale(message, pointer string, current map[string]any) *apperr.Error {
	e := apperr.New(apperr.StalePrecondition, message).WithDetails(apperr.Detail{Pointer: pointer, Problem: "is no longer current"})
	e.Current = current
	return e
}

// Write records the confirmation Check accepted, in the caller's write
// transaction and at seq: bound to the state's target, baseline version
// and source set hash, replacing the target's latest confirmation, with
// the requirements at their current revisions. It returns its ID.
func Write(ctx context.Context, tx *sql.Tx, projectID string, st State, in Input, reqs []Requirement, seq int64, actor, channel string,
	confirmedAt int64) (string, error) {
	attestations, err := json.Marshal(Attestations{CompleteScope: true, PendingItemsExcluded: st.PendingItems,
		MissingGoalsReviewed: in.Attestations.MissingGoalsReviewed})
	if err != nil {
		return "", err
	}
	acknowledged := slices.Clone(in.AcknowledgedWaivers)
	if acknowledged == nil {
		acknowledged = []string{}
	}
	slices.Sort(acknowledged)
	waivers, err := json.Marshal(acknowledged)
	if err != nil {
		return "", err
	}
	var note any
	if in.Note != nil {
		note = *in.Note
	}
	id := ulid.New()
	if _, err := tx.ExecContext(ctx, `INSERT INTO scope_confirmations (id, seq, project_id, target_id, baseline_version, source_set_hash,
    attestations_json, acknowledged_waivers_json, seen_scope_confirmation_id, note, actor, channel, confirmed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, seq, projectID, st.Target.ID, st.BaselineVersion, st.SourceSetHash,
		string(attestations), string(waivers), confirmationID(st.Latest), note, actor, channel, confirmedAt); err != nil {
		return "", err
	}
	for _, r := range reqs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO scope_items (confirmation_id, project_id, requirement_id, requirement_revision) VALUES (?, ?, ?, ?)`,
			id, projectID, r.ID, r.Revision); err != nil {
			return "", err
		}
	}
	return id, nil
}
