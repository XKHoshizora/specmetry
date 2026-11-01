package checkrun

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/contract"
	"github.com/XKHoshizora/specmetry/internal/project"
)

// Document is the check-run.v1 document (spec §11.1).
type Document struct {
	Schema           string    `json:"schema"`
	WorktreeID       string    `json:"worktree_id"`
	IsTargetSnapshot bool      `json:"is_target_snapshot"`
	Isolation        string    `json:"isolation"`
	Runs             []RunItem `json:"runs"`
}

// RunItem is one run of the document, as check-run.v1 has it (spec
// §11.1). A later batch makes it from the row written (§4.2 step 6).
type RunItem struct {
	Criterion         string  `json:"criterion"`
	CriterionRevision int     `json:"criterion_revision"`
	EvidenceID        string  `json:"evidence_id"`
	Seq               int64   `json:"seq"`
	Result            string  `json:"result"`
	FailureKind       *string `json:"failure_kind"`
	ExitCode          *int64  `json:"exit_code"`
	DurationMS        int64   `json:"duration_ms"`
	HeadStart         *string `json:"head_start"`
	TreeStateStart    string  `json:"tree_state_start"`
	EvidenceGrade     string  `json:"evidence_grade"`
	Qualifying        bool    `json:"qualifying"`
}

var validator = sync.OnceValues(contract.New)

// Test hook: testHookRegister runs in the empty batch's registration
// transaction, once the worktree is registered.
var testHookRegister func(*sql.Tx) error

// emptyBatch is step 10 when --all finds nothing to run (D18, E3; spec
// §8.6 D table): no check runs; a registered W is written nothing at all,
// and an unregistered one is registered alone, with its seq. The document
// has no runs, and is_target_snapshot compares HEAD at the command's start
// with S (D9). It is checked against check-run.v1 before it is returned; a
// document that does not conform is an internal error (C1), the
// registration kept.
func (b *Batch) emptyBatch(ctx context.Context, head string) (*Document, error) {
	if b.WorktreeID == "" {
		id, err := b.register(ctx)
		if err != nil {
			return nil, err
		}
		b.WorktreeID = id
	}
	doc := &Document{Schema: "specmetry.check-run/v1", WorktreeID: b.WorktreeID, IsTargetSnapshot: TargetSnapshot(b.Snapshot, nil, head),
		Isolation: "off", Runs: []RunItem{}}
	val, err := validator()
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, err, "cannot load the exchange schemas")
	}
	if err := val.ValidateValue("check-run.v1", doc); err != nil {
		e := apperr.From(err)
		return nil, apperr.Newf(apperr.Internal, "the result does not conform to check-run.v1; no check was run: %s", e.Message).
			WithDetails(e.Details...)
	}
	return doc, nil
}

// register is the empty batch's registration transaction (spec §10.3).
// W registered meanwhile by another call is left as it is, nothing
// written. A failure is told apart as §8.6 rule 3 does: before the body
// returned, the registration was not committed (D3); after it, the error
// came from the commit, and whether W was registered is not known (D4).
func (b *Batch) register(ctx context.Context) (string, error) {
	var id string
	bodyDone := false
	err := b.db.WriteTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `SELECT id FROM worktrees WHERE project_id = ? AND path = ?`, b.ProjectID, b.Where.Current).Scan(&id)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if id, err = project.EnsureWorktree(ctx, tx, b.ProjectID, b.Where, time.Now()); err != nil {
				return err
			}
		case err != nil:
			return err
		}
		if testHookRegister != nil {
			if err := testHookRegister(tx); err != nil {
				return err
			}
		}
		bodyDone = true
		return nil
	})
	switch {
	case err == nil:
		return id, nil
	case bodyDone:
		return "", apperr.New(apperr.Environment,
			"the worktree may have been registered, or not: committing its registration gave an error; no check was run")
	}
	return "", apperr.New(apperr.Environment, "the worktree was not registered, and no check was run")
}
