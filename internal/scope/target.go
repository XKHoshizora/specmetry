// Package scope holds the delivery target of a project and the delivery
// scope confirmed for it (data-model §4.2, §5.3, §7.1; WP1.8).
//
// A project has at most one active target, a branch or a tag ref
// (PD-19); its snapshot is resolved from the ref whenever it is read, never
// stored. Changing the target archives the old one and creates a new one:
// a scope confirmed for the old target does not carry over. A scope
// confirmation is a person's, append-only, bound to the target, the
// baseline version and the source set hash it was made against, and
// replaces the target's confirmation it names (V-T02).
package scope

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/ulid"
)

// Target is a delivery target.
type Target struct {
	ID, Name, Ref string
	CreatedSeq    int64
}

// ActiveTarget reads the project's active target, nil when it has none.
func ActiveTarget(ctx context.Context, q baseline.Querier, projectID string) (*Target, error) {
	var t Target
	err := q.QueryRowContext(ctx, `SELECT id, name, ref, created_seq FROM delivery_targets WHERE project_id = ? AND archived_seq IS NULL`,
		projectID).Scan(&t.ID, &t.Name, &t.Ref, &t.CreatedSeq)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// RefKind is what a target ref may track: a branch or a tag (PD-19).
func RefKind(ref string) (string, bool) {
	for _, prefix := range []string{"refs/heads/", "refs/tags/"} {
		if rest, ok := strings.CutPrefix(ref, prefix); ok && rest != "" {
			return prefix, true
		}
	}
	return "", false
}

// ShortName is a ref without its refs/heads/ or refs/tags/ prefix, the
// default name of a target.
func ShortName(ref string) string {
	if prefix, ok := RefKind(ref); ok {
		return strings.TrimPrefix(ref, prefix)
	}
	return ref
}

// CheckName checks a target's name: one line of at most 200 characters,
// without control characters or a code fence, which no free text may hold
// (architecture §10.2; codex-review-034 B2).
func CheckName(name string) error {
	problem := ""
	switch {
	case !utf8.ValidString(name):
		problem = "is not valid UTF-8"
	case strings.TrimSpace(name) == "":
		problem = "is empty"
	case utf8.RuneCountInString(name) > 200:
		problem = "is longer than 200 characters"
	case strings.ContainsFunc(name, unicode.IsControl):
		problem = "contains a line break or another control character"
	case strings.Contains(name, "```"):
		problem = "contains a code fence (three backticks), which free text may not"
	default:
		return nil
	}
	return apperr.New(apperr.InvalidInput, "invalid target name").WithDetails(apperr.Detail{Pointer: "--name", Problem: problem})
}

// SetTarget makes ref the project's target, named name, by actor on
// channel, in the caller's write transaction: the active target is
// archived and the new one created, at one seq. seen is the active target
// the person was shown (nil for none); another one is stale, so that two
// people changing the target at once do not archive each other's.
func SetTarget(ctx context.Context, tx *sql.Tx, projectID string, seen *string, ref, name, actor, channel string) (Target, error) {
	if err := CheckName(name); err != nil {
		return Target{}, err
	}
	current, err := ActiveTarget(ctx, tx, projectID)
	if err != nil {
		return Target{}, err
	}
	if !sameTarget(current, seen) {
		e := apperr.New(apperr.StalePrecondition, "the delivery target changed meanwhile; run target show and set it again")
		e.Current = map[string]any{"target": targetValue(current)}
		return Target{}, e
	}
	seq, err := sqlite.NextSeq(ctx, tx)
	if err != nil {
		return Target{}, err
	}
	if current != nil {
		if _, err := tx.ExecContext(ctx, `UPDATE delivery_targets SET archived_seq = ? WHERE id = ? AND project_id = ? AND archived_seq IS NULL`,
			seq, current.ID, projectID); err != nil {
			return Target{}, err
		}
	}
	t := Target{ID: ulid.New(), Name: name, Ref: ref, CreatedSeq: seq}
	if _, err := tx.ExecContext(ctx, `INSERT INTO delivery_targets (id, project_id, name, ref, created_seq, actor, channel) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, projectID, name, ref, seq, actor, channel); err != nil {
		return Target{}, err
	}
	return t, nil
}

func sameTarget(current *Target, seen *string) bool {
	if current == nil || seen == nil {
		return current == nil && seen == nil
	}
	return current.ID == *seen
}

// targetValue is a target in error.current: its ID and ref, or nil.
func targetValue(t *Target) any {
	if t == nil {
		return nil
	}
	return map[string]any{"id": t.ID, "ref": t.Ref}
}
