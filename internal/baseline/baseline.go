// Package baseline reads the confirmed baseline (data-model.md §4.4, §4.7)
// and hands out keys (TD-14). Proposals only read the baseline; a person's
// review changes it (WP1.6).
package baseline

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Type is the type of a baseline entity, or a question.
type Type string

// Types.
const (
	Goal        Type = "goal"
	Requirement Type = "requirement"
	Criterion   Type = "criterion"
	Question    Type = "question"
)

// CurrentVersion returns the project's current baseline version: the
// highest version number, or 0 before the first.
func CurrentVersion(ctx context.Context, q Querier, projectID string) (int, error) {
	var v int
	err := q.QueryRowContext(ctx, `SELECT coalesce(max(number), 0) FROM baseline_versions WHERE project_id = ?`, projectID).Scan(&v)
	return v, err
}

// Entity is a baseline entity as a proposal sees it.
type Entity struct {
	ID     string
	Type   Type
	Key    string
	Active bool
	// Revision is the entity's current revision.
	Revision int
	// RequirementID and RequirementRevision are a criterion's requirement
	// and the requirement revision it was confirmed for; Verification is
	// how it is verified, and HasCheck whether it has a check.
	RequirementID       string
	RequirementRevision int
	Verification        string
	HasCheck            bool
}

var keyPatterns = []struct {
	pattern *regexp.Regexp
	typ     Type
	table   string
}{
	{regexp.MustCompile(`^GOAL-[1-9][0-9]*$`), Goal, "goals"},
	{regexp.MustCompile(`^REQ-[0-9]{3,}$`), Requirement, "requirements"},
	{regexp.MustCompile(`^AC-[0-9]{3,}-[1-9][0-9]*$`), Criterion, "acceptance_criteria"},
}

// TypeOfKey returns the type of entity a key names.
func TypeOfKey(key string) (Type, bool) {
	for _, k := range keyPatterns {
		if k.pattern.MatchString(key) {
			return k.typ, true
		}
	}
	return "", false
}

// Find finds a baseline entity of the project by its key, active or
// retired.
func Find(ctx context.Context, tx *sql.Tx, projectID, key string) (Entity, bool, error) {
	for _, k := range keyPatterns {
		if !k.pattern.MatchString(key) {
			continue
		}
		e := Entity{Type: k.typ, Key: key}
		var status string
		var err error
		if k.typ == Criterion {
			err = tx.QueryRowContext(ctx, `SELECT id, status, revision, requirement_id, requirement_revision, verification,
    check_command IS NOT NULL
FROM acceptance_criteria WHERE project_id = ? AND key = ?`, projectID, key).
				Scan(&e.ID, &status, &e.Revision, &e.RequirementID, &e.RequirementRevision, &e.Verification, &e.HasCheck)
		} else {
			err = tx.QueryRowContext(ctx, `SELECT id, status, revision FROM `+k.table+` WHERE project_id = ? AND key = ?`, projectID, key).
				Scan(&e.ID, &status, &e.Revision)
		}
		if errors.Is(err, sql.ErrNoRows) {
			return Entity{}, false, nil
		}
		if err != nil {
			return Entity{}, false, err
		}
		e.Active = status == "active"
		return e, true, nil
	}
	return Entity{}, false, nil
}

// NextKey hands out the next key of a type (TD-14): GOAL-n, REQ-nnn and
// Q-nnn, numbered within the project, and AC-<requirement number>-n,
// numbered within requirement. Numbers have at least three digits where
// the format pads them and simply grow past that. A number is never handed
// out twice: the counters only grow, also when an entity is retired; a
// transaction that rolls back hands nothing out.
func NextKey(ctx context.Context, tx *sql.Tx, projectID string, t Type, requirement *Entity) (string, error) {
	scope := ""
	if t == Criterion {
		if requirement == nil || requirement.ID == "" || !strings.HasPrefix(requirement.Key, "REQ-") {
			return "", errors.New("a criterion key is numbered within its requirement, which is missing")
		}
		scope = requirement.ID
	}
	var n int
	err := tx.QueryRowContext(ctx, `SELECT next_value FROM key_counters WHERE project_id = ? AND entity_type = ? AND scope = ?`,
		projectID, string(t), scope).Scan(&n)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		n = 1
		_, err = tx.ExecContext(ctx, `INSERT INTO key_counters (project_id, entity_type, scope, next_value) VALUES (?, ?, ?, 2)`,
			projectID, string(t), scope)
	case err == nil:
		_, err = tx.ExecContext(ctx, `UPDATE key_counters SET next_value = next_value + 1 WHERE project_id = ? AND entity_type = ? AND scope = ?`,
			projectID, string(t), scope)
	}
	if err != nil {
		return "", err
	}
	switch t {
	case Goal:
		return fmt.Sprintf("GOAL-%d", n), nil
	case Requirement:
		return fmt.Sprintf("REQ-%03d", n), nil
	case Criterion:
		return fmt.Sprintf("AC-%s-%d", strings.TrimPrefix(requirement.Key, "REQ-"), n), nil
	case Question:
		return fmt.Sprintf("Q-%03d", n), nil
	}
	return "", fmt.Errorf("no keys for %q", t)
}
