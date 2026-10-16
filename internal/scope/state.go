package scope

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/source"
)

// Requirement is an active requirement a scope can hold.
type Requirement struct {
	ID, Key, Title, Priority string
	Revision                 int
}

// Waiver is a waiver in effect (PD-21): the current decision of an active
// criterion is waived, and its parent's revision has not moved on.
type Waiver struct {
	DecisionID        string `json:"decision_id"`
	Criterion         string `json:"criterion"`
	CriterionRevision int    `json:"criterion_revision"`
	Requirement       string `json:"requirement"`
	Rationale         string `json:"rationale"`
	Seq               int64  `json:"seq"`
}

// Attestations are what a person states with a scope (scope.v1).
type Attestations struct {
	CompleteScope        bool  `json:"complete_scope"`
	PendingItemsExcluded int   `json:"pending_items_excluded"`
	MissingGoalsReviewed *bool `json:"missing_goals_reviewed,omitempty"`
}

// ConfirmedRequirement is a requirement of a confirmation, at the revision
// it had then.
type ConfirmedRequirement struct {
	Key      string `json:"key"`
	Revision int    `json:"revision"`
}

// Confirmation is a stored scope confirmation.
type Confirmation struct {
	ID                  string
	Seq                 int64
	BaselineVersion     int
	SourceSetHash       string
	Requirements        []ConfirmedRequirement
	Attestations        Attestations
	AcknowledgedWaivers []string
	Note                *string
}

// State is what a scope confirmation is checked against, read in one
// transaction: the active target and its latest confirmation, the current
// baseline version and source set hash, the proposal items still pending or
// in conflict, the project's stage, the active requirements and the
// waivers in effect.
type State struct {
	Target          *Target
	Latest          *Confirmation
	BaselineVersion int
	SourceSetHash   string
	PendingItems    int
	Stage           string
	// Requirements are the active requirements by key.
	Requirements map[string]Requirement
	Waivers      []Waiver
}

// Read reads the project's state in the caller's transaction.
func Read(ctx context.Context, tx *sql.Tx, projectID string) (State, error) {
	var st State
	var err error
	if st.Target, err = ActiveTarget(ctx, tx, projectID); err != nil {
		return st, err
	}
	if st.Target != nil {
		if st.Latest, err = latest(ctx, tx, projectID, st.Target.ID); err != nil {
			return st, err
		}
	}
	if st.BaselineVersion, err = baseline.CurrentVersion(ctx, tx, projectID); err != nil {
		return st, err
	}
	if st.SourceSetHash, err = source.ProjectSetHash(ctx, tx, projectID); err != nil {
		return st, err
	}
	// Not yet reviewed: pending or in conflict.
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM proposal_items WHERE project_id = ? AND reviewed_seq IS NULL`, projectID).
		Scan(&st.PendingItems); err != nil {
		return st, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT stage FROM projects WHERE id = ?`, projectID).Scan(&st.Stage); err != nil {
		return st, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id, key, title, priority, revision FROM requirements WHERE project_id = ? AND status = 'active'`, projectID)
	if err != nil {
		return st, err
	}
	defer func() { _ = rows.Close() }()
	st.Requirements = map[string]Requirement{}
	for rows.Next() {
		var r Requirement
		if err := rows.Scan(&r.ID, &r.Key, &r.Title, &r.Priority, &r.Revision); err != nil {
			return st, err
		}
		st.Requirements[r.Key] = r
	}
	if err := rows.Err(); err != nil {
		return st, err
	}
	st.Waivers, err = effectiveWaivers(ctx, tx, projectID)
	return st, err
}

// effectiveWaivers reads the waivers in effect. Acceptance decisions, and
// so waivers, are recorded from M2 on (data-model §4.13, WP2.7); a project
// of this version has none. The checks that use the set are the same
// either way (TestCheckWaivers).
func effectiveWaivers(context.Context, *sql.Tx, string) ([]Waiver, error) {
	return []Waiver{}, nil
}

func latest(ctx context.Context, tx *sql.Tx, projectID, targetID string) (*Confirmation, error) {
	var c Confirmation
	var attestations, waivers string
	var note sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT id, seq, baseline_version, source_set_hash, attestations_json, acknowledged_waivers_json, note
FROM scope_confirmations WHERE project_id = ? AND target_id = ? ORDER BY seq DESC LIMIT 1`, projectID, targetID).
		Scan(&c.ID, &c.Seq, &c.BaselineVersion, &c.SourceSetHash, &attestations, &waivers, &note)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(attestations), &c.Attestations); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(waivers), &c.AcknowledgedWaivers); err != nil {
		return nil, err
	}
	if note.Valid {
		c.Note = &note.String
	}
	rows, err := tx.QueryContext(ctx, `SELECT r.key, i.requirement_revision FROM scope_items i
JOIN requirements r ON r.project_id = i.project_id AND r.id = i.requirement_id
WHERE i.project_id = ? AND i.confirmation_id = ?`, projectID, c.ID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var r ConfirmedRequirement
		if err := rows.Scan(&r.Key, &r.Revision); err != nil {
			return nil, err
		}
		c.Requirements = append(c.Requirements, r)
	}
	slices.SortFunc(c.Requirements, func(a, b ConfirmedRequirement) int { return compareKeys(a.Key, b.Key) })
	return &c, rows.Err()
}

// Stale reports whether the latest confirmation no longer holds: the
// baseline version or the source set hash moved on (SCOPE_STALE).
func (st State) Stale() bool {
	return st.Latest != nil && (st.Latest.BaselineVersion != st.BaselineVersion || st.Latest.SourceSetHash != st.SourceSetHash)
}

// MustRequirements are the active requirements with priority must, in key
// order: the default selection of an interactive confirmation (FR-TGT-07).
func (st State) MustRequirements() []string {
	var keys []string
	for k, r := range st.Requirements {
		if r.Priority == "must" {
			keys = append(keys, k)
		}
	}
	slices.SortFunc(keys, compareKeys)
	return keys
}

// WaiversUnder are the waivers in effect under the requirements keys, by
// decision ID order.
func (st State) WaiversUnder(keys []string) []Waiver {
	out := []Waiver{}
	for _, w := range st.Waivers {
		if slices.Contains(keys, w.Requirement) {
			out = append(out, w)
		}
	}
	slices.SortFunc(out, func(a, b Waiver) int { return strings.Compare(a.DecisionID, b.DecisionID) })
	return out
}

// compareKeys orders keys of one kind by number: REQ-002 before REQ-010.
func compareKeys(a, b string) int {
	if len(a) != len(b) {
		return len(a) - len(b)
	}
	return strings.Compare(a, b)
}
