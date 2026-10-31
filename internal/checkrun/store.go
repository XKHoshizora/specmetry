package checkrun

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/ulid"
)

// Input is one entry of a run's input record (data-model §4.11.2): the
// saved path and the tree entry's mode, type and object ID at head_start.
type Input struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	OID  string `json:"oid"`
}

// Run is what one execution records, before it is written (data-model
// §4.11, spec §10.1). The criterion revision, requirement revision and
// configuration hash are those read when the run started.
type Run struct {
	CriterionID         string
	CriterionRevision   int
	RequirementRevision int
	CheckConfigHash     string
	// Inputs is the input record; nil is NULL (no inputs declared, or not a
	// git project).
	Inputs         []Input
	Workdir        string
	HeadStart      string // "" is NULL
	HeadEnd        string // "" is NULL
	TreeStateStart string
	TreeStateEnd   string
	Result         string
	FailureKind    string // "" is NULL
	ExitCode       *int64 // nil is NULL
	DurationMS     int64
	// Channel is the channel that triggered the run: human_cli or
	// agent_cli.
	Channel               string
	StartedAt, FinishedAt int64
}

// Row is a written check_runs row, read back.
type Row struct {
	ID         string
	Seq        int64
	WorktreeID string
	Run
	EvidenceGrade string
	IsolationMode string
}

// Write records run in tx, the caller's write transaction (DB.WriteTx,
// BEGIN IMMEDIATE), as spec §10.3 orders it: the worktree where is
// registered the first time it is seen, with a seq of its own
// (project.EnsureWorktree), then the run gets the next seq. There is no
// version precondition: the run keeps the revisions and configuration of
// its start, and a later baseline change does not refuse it (D13, CX-15).
// The observed runner writes the observed grade and isolation off, and
// nothing in the isolation columns or session_id (M3). It returns the row
// as written, read back. Any error leaves the caller to roll tx back.
func Write(ctx context.Context, tx *sql.Tx, projectID string, where project.Where, run Run) (Row, error) {
	worktree, err := project.EnsureWorktree(ctx, tx, projectID, where, time.Now())
	if err != nil {
		return Row{}, err
	}
	var inputs any
	if run.Inputs != nil {
		sorted := slices.Clone(run.Inputs)
		slices.SortFunc(sorted, func(a, b Input) int { return strings.Compare(a.Path, b.Path) })
		b, err := json.Marshal(sorted)
		if err != nil {
			return Row{}, err
		}
		inputs = string(b)
	}
	seq, err := sqlite.NextSeq(ctx, tx)
	if err != nil {
		return Row{}, err
	}
	id := ulid.New()
	if _, err := tx.ExecContext(ctx, `INSERT INTO check_runs (id, seq, project_id, criterion_id, criterion_revision, requirement_revision,
    check_config_hash, inputs_json, worktree_id, workdir, evidence_grade, isolation_mode, isolation_backend, isolation_fallback_reason,
    head_start, head_end, tree_state_start, tree_state_end, result, failure_kind, exit_code, duration_ms, triggered_channel, session_id,
    started_at, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)`,
		id, seq, projectID, run.CriterionID, run.CriterionRevision, run.RequirementRevision, run.CheckConfigHash, inputs, worktree,
		run.Workdir, Observed, Off, nullable(run.HeadStart), nullable(run.HeadEnd), run.TreeStateStart, run.TreeStateEnd, run.Result,
		nullable(run.FailureKind), run.ExitCode, run.DurationMS, run.Channel, run.StartedAt, run.FinishedAt); err != nil {
		return Row{}, err
	}
	return read(ctx, tx, id)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// read reads one row back by its ID.
func read(ctx context.Context, tx *sql.Tx, id string) (Row, error) {
	var r Row
	var inputs, headStart, headEnd, kind sql.NullString
	var code sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT id, seq, worktree_id, criterion_id, criterion_revision, requirement_revision, check_config_hash,
    inputs_json, workdir, evidence_grade, isolation_mode, head_start, head_end, tree_state_start, tree_state_end, result, failure_kind,
    exit_code, duration_ms, triggered_channel, started_at, finished_at
FROM check_runs WHERE id = ?`, id).Scan(&r.ID, &r.Seq, &r.WorktreeID, &r.CriterionID, &r.CriterionRevision, &r.RequirementRevision,
		&r.CheckConfigHash, &inputs, &r.Workdir, &r.EvidenceGrade, &r.IsolationMode, &headStart, &headEnd, &r.TreeStateStart,
		&r.TreeStateEnd, &r.Result, &kind, &code, &r.DurationMS, &r.Channel, &r.StartedAt, &r.FinishedAt)
	if err != nil {
		return Row{}, err
	}
	r.HeadStart, r.HeadEnd, r.FailureKind = headStart.String, headEnd.String, kind.String
	if code.Valid {
		c := code.Int64
		r.ExitCode = &c
	}
	if inputs.Valid {
		if err := json.Unmarshal([]byte(inputs.String), &r.Inputs); err != nil {
			return Row{}, err
		}
	}
	return r, nil
}
