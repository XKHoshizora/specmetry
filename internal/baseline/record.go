package baseline

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Check is a criterion's check (data-model §4.4), in the checkSpec shape
// of common.v1.
type Check struct {
	Command    string   `json:"command"`
	Workdir    string   `json:"workdir"`
	Shell      *string  `json:"shell,omitempty"`
	TimeoutSec int      `json:"timeout_sec"`
	Inputs     []string `json:"inputs,omitempty"`
	Rationale  string   `json:"rationale"`
}

// Record is a baseline entity with every field (data-model §4.4). Keys of
// the entities it is linked to are read with it, for its values.
type Record struct {
	ID             string
	Type           Type
	Key            string
	Status         string
	Revision       int
	RevisionSeq    int64
	Origin         string
	Confidence     *string
	CreatedVersion int
	UpdatedVersion int
	RetiredVersion *int
	UpdatedSeq     int64

	// Goals and requirements.
	Title       string
	Description *string
	// Requirements.
	GoalID, GoalKey *string
	Kind, Priority  string
	// Criteria.
	RequirementID       string
	RequirementKey      string
	RequirementRevision int
	Statement           string
	Verification        string
	Check               *Check
}

// Active reports whether the entity is not retired.
func (r Record) Active() bool { return r.Status == "active" }

const commonColumns = `id, key, status, revision, revision_seq, origin, confidence, created_version, updated_version,
    retired_version, updated_seq`

// Load reads an entity of the project by its ID; found is false when there
// is none of that type.
func Load(ctx context.Context, q Querier, projectID string, t Type, id string) (Record, bool, error) {
	return load(ctx, q, projectID, t, "id", id)
}

// LoadKey reads an entity of the project by its key, whatever its type.
func LoadKey(ctx context.Context, q Querier, projectID, key string) (Record, bool, error) {
	t, ok := TypeOfKey(key)
	if !ok {
		return Record{}, false, nil
	}
	return load(ctx, q, projectID, t, "key", key)
}

// Querier is a *sql.Tx or another handle that runs queries.
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func load(ctx context.Context, q Querier, projectID string, t Type, by, value string) (Record, bool, error) {
	rows, err := q.QueryContext(ctx, selectRecords(t)+` AND x.`+by+` = ?`, projectID, value)
	if err != nil {
		return Record{}, false, err
	}
	recs, err := scanRecords(rows, t)
	if err != nil || len(recs) == 0 {
		return Record{}, false, err
	}
	return recs[0], true, nil
}

// LoadAll reads every entity of a type of the project, in key order.
func LoadAll(ctx context.Context, q Querier, projectID string, t Type) ([]Record, error) {
	rows, err := q.QueryContext(ctx, selectRecords(t), projectID)
	if err != nil {
		return nil, err
	}
	recs, err := scanRecords(rows, t)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(recs, func(a, b Record) int { return compareKeys(a.Key, b.Key) })
	return recs, nil
}

func selectRecords(t Type) string {
	cols := strings.ReplaceAll(commonColumns, "\n    ", " ")
	cols = "x." + strings.ReplaceAll(cols, ", ", ", x.")
	switch t {
	case Goal:
		return `SELECT ` + cols + `, x.title, x.description FROM goals x WHERE x.project_id = ?`
	case Requirement:
		return `SELECT ` + cols + `, x.title, x.description, x.goal_id, g.key, x.kind, x.priority
FROM requirements x LEFT JOIN goals g ON g.id = x.goal_id WHERE x.project_id = ?`
	}
	return `SELECT ` + cols + `, x.requirement_id, r.key, x.requirement_revision, x.statement, x.verification,
    x.check_command, x.check_workdir, x.check_shell, x.check_inputs_json, x.check_rationale, x.check_timeout_sec
FROM acceptance_criteria x JOIN requirements r ON r.id = x.requirement_id WHERE x.project_id = ?`
}

func scanRecords(rows *sql.Rows, t Type) ([]Record, error) {
	defer func() { _ = rows.Close() }()
	var out []Record
	for rows.Next() {
		r := Record{Type: t}
		var retired sql.NullInt64
		var confidence sql.NullString
		dest := []any{&r.ID, &r.Key, &r.Status, &r.Revision, &r.RevisionSeq, &r.Origin, &confidence,
			&r.CreatedVersion, &r.UpdatedVersion, &retired, &r.UpdatedSeq}
		var description, goalID, goalKey, command, shell, inputs, rationale sql.NullString
		var workdir string
		var timeout int
		switch t {
		case Goal:
			dest = append(dest, &r.Title, &description)
		case Requirement:
			dest = append(dest, &r.Title, &description, &goalID, &goalKey, &r.Kind, &r.Priority)
		case Criterion:
			dest = append(dest, &r.RequirementID, &r.RequirementKey, &r.RequirementRevision, &r.Statement, &r.Verification,
				&command, &workdir, &shell, &inputs, &rationale, &timeout)
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		if retired.Valid {
			v := int(retired.Int64)
			r.RetiredVersion = &v
		}
		r.Confidence = nullString(confidence)
		r.Description = nullString(description)
		r.GoalID, r.GoalKey = nullString(goalID), nullString(goalKey)
		if command.Valid {
			c := &Check{Command: command.String, Workdir: workdir, Shell: nullString(shell), TimeoutSec: timeout, Rationale: rationale.String}
			if inputs.Valid {
				if err := json.Unmarshal([]byte(inputs.String), &c.Inputs); err != nil {
					return nil, fmt.Errorf("check inputs of %s: %w", r.Key, err)
				}
			}
			r.Check = c
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func nullString(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}

// compareKeys orders keys by their numbers: GOAL-2 before GOAL-10,
// AC-002-10 after AC-002-9.
func compareKeys(a, b string) int {
	na, nb := keyNumbers(a), keyNumbers(b)
	if c := slices.Compare(na, nb); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}

func keyNumbers(key string) []int {
	var out []int
	for _, part := range strings.Split(key, "-")[1:] {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	return out
}

// Values are the entity's fields in the proposal data shape of its type
// (goalData, requirementData, criterionData), with keys for the entities
// it is linked to.
func (r Record) Values() map[string]any {
	v := map[string]any{}
	switch r.Type {
	case Goal, Requirement:
		v["title"] = r.Title
		if r.Description != nil {
			v["description"] = *r.Description
		}
		if r.Type == Requirement {
			v["goal"] = r.GoalKey
			v["kind"], v["priority"] = r.Kind, r.Priority
		}
	case Criterion:
		v["requirement"] = r.RequirementKey
		v["statement"], v["verification"] = r.Statement, r.Verification
		v["check"] = r.Check
	}
	return v
}

// Snapshot is the entity as baseline-history.v1 records it before and
// after a change.
func (r Record) Snapshot() map[string]any {
	s := map[string]any{"revision": r.Revision, "status": r.Status, "values": r.Values()}
	if r.Type == Criterion {
		s["requirement_revision"] = r.RequirementRevision
	}
	return s
}

// CheckConfigHash is the sha256 of the check's configuration in JSON
// canonical form (RFC 8785): {"command","inputs","shell","workdir"}, with
// the inputs sorted and absent inputs as an empty list, and a shell that is
// not set as null. The timeout and the rationale are not part of it
// (data-model §4.4).
func CheckConfigHash(c Check) string {
	inputs := slices.Clone(c.Inputs)
	slices.Sort(inputs)
	var b strings.Builder
	b.WriteString(`{"command":`)
	jcsString(&b, c.Command)
	b.WriteString(`,"inputs":[`)
	for i, in := range inputs {
		if i > 0 {
			b.WriteByte(',')
		}
		jcsString(&b, in)
	}
	b.WriteString(`],"shell":`)
	if c.Shell == nil {
		b.WriteString("null")
	} else {
		jcsString(&b, *c.Shell)
	}
	b.WriteString(`,"workdir":`)
	jcsString(&b, c.Workdir)
	b.WriteByte('}')
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// LineBreakProblem is what WP2.5 D3 and D4 refuse in c's command, or "".
// cmd runs only the first line of a command, without a word, so a cmd
// command holds no CR or LF (D3); an sh or bash command, or one with no
// shell, which runs as sh (D1), holds no CR, quoted or not (D4). Nothing is
// normalized: that would change the saved command and its hash. pwsh and
// powershell commands are not concerned. Every path that writes a check
// applies it to the check it writes (V-P04), and check run to a saved one
// before it runs anything; a saved check stays readable as it is.
func (c Check) LineBreakProblem() string {
	shell := "sh"
	if c.Shell != nil {
		shell = *c.Shell
	}
	switch {
	case shell == "cmd" && strings.ContainsAny(c.Command, "\r\n"):
		return "has a line break (CR or LF): cmd runs only the first line of a command, so the command of a cmd check is one line"
	case (shell == "sh" || shell == "bash") && strings.Contains(c.Command, "\r"):
		return "has a carriage return (CR), quoted or not: the command of an sh or bash check, or one with no shell, takes LF line breaks only"
	}
	return ""
}

// jcsString writes s as RFC 8785 serializes a string: quote and backslash
// escaped, control characters as \b \t \n \f \r or \u00xx, everything else
// as it is.
func jcsString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\f':
			b.WriteString(`\f`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r < 0x20 {
				fmt.Fprintf(b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}

// insert writes a new entity.
func insert(ctx context.Context, tx *sql.Tx, projectID string, r Record) error {
	common := []any{r.ID, projectID, r.Key, r.Status, r.Revision, r.RevisionSeq, r.Origin, r.Confidence,
		r.CreatedVersion, r.UpdatedVersion, r.RetiredVersion, r.UpdatedSeq}
	cols := `id, project_id, key, status, revision, revision_seq, origin, confidence, created_version, updated_version,
    retired_version, updated_seq`
	var query string
	var args []any
	switch r.Type {
	case Goal:
		query = `INSERT INTO goals (` + cols + `, title, description) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		args = slices.Concat(common, []any{r.Title, r.Description})
	case Requirement:
		query = `INSERT INTO requirements (` + cols + `, goal_id, title, description, kind, priority)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		args = slices.Concat(common, []any{r.GoalID, r.Title, r.Description, r.Kind, r.Priority})
	case Criterion:
		check, err := checkColumns(r.Check)
		if err != nil {
			return err
		}
		query = `INSERT INTO acceptance_criteria (` + cols + `, requirement_id, requirement_revision, statement, verification,
    check_command, check_workdir, check_shell, check_inputs_json, check_rationale, check_timeout_sec, check_config_hash)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		args = slices.Concat(common, []any{r.RequirementID, r.RequirementRevision, r.Statement, r.Verification}, check)
	default:
		return errors.New("no table for " + string(r.Type))
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

// update writes an entity's changed fields; its key, type and links to a
// requirement never change.
func update(ctx context.Context, tx *sql.Tx, projectID string, r Record) error {
	common := []any{r.Status, r.Revision, r.RevisionSeq, r.UpdatedVersion, r.RetiredVersion, r.UpdatedSeq}
	set := `status = ?, revision = ?, revision_seq = ?, updated_version = ?, retired_version = ?, updated_seq = ?`
	var query string
	var args []any
	switch r.Type {
	case Goal:
		query = `UPDATE goals SET ` + set + `, title = ?, description = ? WHERE project_id = ? AND id = ?`
		args = slices.Concat(common, []any{r.Title, r.Description})
	case Requirement:
		query = `UPDATE requirements SET ` + set + `, goal_id = ?, title = ?, description = ?, kind = ?, priority = ?
WHERE project_id = ? AND id = ?`
		args = slices.Concat(common, []any{r.GoalID, r.Title, r.Description, r.Kind, r.Priority})
	case Criterion:
		check, err := checkColumns(r.Check)
		if err != nil {
			return err
		}
		query = `UPDATE acceptance_criteria SET ` + set + `, requirement_revision = ?, statement = ?, verification = ?,
    check_command = ?, check_workdir = ?, check_shell = ?, check_inputs_json = ?, check_rationale = ?, check_timeout_sec = ?,
    check_config_hash = ?
WHERE project_id = ? AND id = ?`
		args = slices.Concat(common, []any{r.RequirementRevision, r.Statement, r.Verification}, check)
	default:
		return errors.New("no table for " + string(r.Type))
	}
	res, err := tx.ExecContext(ctx, query, append(args, projectID, r.ID)...)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		return fmt.Errorf("updating %s changed %d rows", r.Key, n)
	}
	return nil
}

// checkColumns are a criterion's check columns: command, workdir, shell,
// inputs, rationale, timeout and configuration hash.
func checkColumns(c *Check) ([]any, error) {
	if c == nil {
		return []any{nil, ".", nil, nil, nil, 300, nil}, nil
	}
	var inputs any
	if c.Inputs != nil {
		b, err := json.Marshal(c.Inputs)
		if err != nil {
			return nil, err
		}
		inputs = string(b)
	}
	return []any{c.Command, c.Workdir, c.Shell, inputs, c.Rationale, c.TimeoutSec, CheckConfigHash(*c)}, nil
}
