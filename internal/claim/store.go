package claim

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/ulid"
)

// SchemaID is the receipt's "schema" field.
const SchemaID = "specmetry.claim-receipt/v1"

// Submission is a batch checked outside the database, with the code
// context captured and the references checked, ready to store.
type Submission struct {
	Batch   *Batch
	Context Context
	Checked Checked
	Where   project.Where
	Actor   string
	Channel string
}

// staleTarget is a claim whose target_revision is not the criterion's
// current revision (error.current.targets, as propose reports it).
type staleTarget struct {
	Pointer  string `json:"pointer"`
	Target   string `json:"target"`
	Revision int    `json:"revision"`
}

// Store checks the batch against the baseline and stores it, in the
// caller's write transaction (BEGIN IMMEDIATE: everything is read under
// the write lock), in the order of WP2.4 spec §5 step 10:
//
//  1. base_version must be the current baseline version, or the batch is
//     stale (exit 5) with the current version. This comes first: a lawful
//     change after the batch was read (a retirement included) moved the
//     version on, so it is never reported as exit 2 (review 065 B1);
//  2. every target is an active criterion (V-C03); every requirement an
//     audit scope names is active (D2); every claim of such an audit names
//     a criterion of one of them (D3), without the scope having to be
//     covered. These are invalid input (exit 2), reported together;
//  3. every target_revision is the criterion's current revision, or the
//     batch is stale (exit 5) with the current revisions;
//  4. the worktree is registered the first time it is seen, with a seq of
//     its own (data-model §4.10);
//  5. a sync batch's comparison start (§6.3);
//  6. one seq for the batch, its claims and its unmapped work.
//
// The receipt is then built from the rows written, read back. Any error
// leaves the caller to roll the transaction back.
func Store(ctx context.Context, tx *sql.Tx, projectID string, s Submission) (Receipt, error) {
	b := s.Batch
	current, err := baseline.CurrentVersion(ctx, tx, projectID)
	if err != nil {
		return Receipt{}, err
	}
	if current != b.BaseVersion {
		e := apperr.Newf(apperr.StalePrecondition,
			"the baseline is at version %d, not %d; read it again (specmetry status --json) and claim against it; nothing was written", current, b.BaseVersion).
			WithDetails(apperr.Detail{Pointer: "/base_version", Problem: fmt.Sprintf("is stale: the current baseline version is %d", current)})
		e.Current = map[string]any{"base_version": current}
		return Receipt{}, e
	}

	var ps problems
	criteria := make([]baseline.Entity, len(b.Claims))
	valid := make([]bool, len(b.Claims))
	for i, c := range b.Claims {
		e, ok, err := baseline.Find(ctx, tx, projectID, c.Target)
		if err != nil {
			return Receipt{}, err
		}
		pointer := "/claims/" + strconv.Itoa(i) + "/target"
		switch {
		case !ok:
			ps.add(pointer, "names no criterion of the baseline; only an active criterion of the baseline can be claimed")
		case e.Type != baseline.Criterion:
			ps.add(pointer, "names a %s; a claim names a criterion", e.Type)
		case !e.Active:
			ps.add(pointer, "names a retired criterion; only an active criterion can be claimed")
		default:
			criteria[i], valid[i] = e, true
		}
	}
	if b.Kind == Audit && b.AuditScope != nil && !b.AuditScope.All {
		declared := map[string]bool{}
		for j, key := range b.AuditScope.Requirements {
			e, ok, err := baseline.Find(ctx, tx, projectID, key)
			if err != nil {
				return Receipt{}, err
			}
			pointer := "/audit_scope/requirements/" + strconv.Itoa(j)
			switch {
			case !ok || e.Type != baseline.Requirement:
				ps.add(pointer, "names no requirement of the baseline")
			case !e.Active:
				ps.add(pointer, "names a retired requirement; an audit scope names active requirements")
			default:
				declared[e.ID] = true
			}
		}
		for i := range b.Claims {
			if valid[i] && !declared[criteria[i].RequirementID] {
				ps.add("/claims/"+strconv.Itoa(i)+"/target", "is a criterion of a requirement the audit scope does not declare")
			}
		}
	}
	if err := ps.err(); err != nil {
		return Receipt{}, err
	}
	var stale []staleTarget
	for i, c := range b.Claims {
		if criteria[i].Revision != c.TargetRevision {
			stale = append(stale, staleTarget{"/claims/" + strconv.Itoa(i) + "/target_revision", c.Target, criteria[i].Revision})
		}
	}
	if len(stale) > 0 {
		e := apperr.New(apperr.StalePrecondition, "a target has changed since it was read; read the baseline again and claim against it; nothing was written")
		for _, st := range stale {
			e.Details = append(e.Details, apperr.Detail{Pointer: st.Pointer, Problem: fmt.Sprintf("is stale: %s is at revision %d", st.Target, st.Revision)})
		}
		e.Current = map[string]any{"base_version": current, "targets": stale}
		return Receipt{}, e
	}

	worktree, err := project.EnsureWorktree(ctx, tx, projectID, s.Where, time.Now())
	if err != nil {
		return Receipt{}, err
	}
	var compareFrom, compareSource any
	if b.Kind == Sync {
		compareSource = "none"
		if commit, ok, err := previousBatch(ctx, tx, projectID, worktree); err != nil {
			return Receipt{}, err
		} else if ok {
			compareFrom, compareSource = commit, "previous_batch"
		}
	}
	seq, err := sqlite.NextSeq(ctx, tx)
	if err != nil {
		return Receipt{}, err
	}
	var scope any
	if b.Kind == Audit {
		if scope, err = scopeJSON(b.AuditScope); err != nil {
			return Receipt{}, err
		}
	}
	batchID := ulid.New()
	if _, err := tx.ExecContext(ctx, `INSERT INTO claim_batches (id, seq, project_id, kind, audit_scope_json, base_version, worktree_id, head,
    tree_state, compare_from, compare_from_source, session_id, summary, actor, channel)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?)`, batchID, seq, projectID, b.Kind, scope, b.BaseVersion, worktree, nullable(s.Context.Head),
		s.Context.TreeState, compareFrom, compareSource, b.Summary, s.Actor, s.Channel); err != nil {
		return Receipt{}, err
	}
	// The IDs of what was written, by input index: the receipt is built in
	// input order from these, not from the order SQL returns rows in or
	// the order of the IDs.
	claimIDs := make([]string, len(b.Claims))
	for i, c := range b.Claims {
		claimIDs[i] = ulid.New()
		if err := insertClaim(ctx, tx, claimIDs[i], seq, projectID, batchID, &criteria[i], c.TargetRevision, c.Assertion, c.Summary, c.Refs, s.Checked.Claims[i]); err != nil {
			return Receipt{}, err
		}
	}
	unmappedIDs := make([]string, len(b.UnmappedWork))
	for i, u := range b.UnmappedWork {
		unmappedIDs[i] = ulid.New()
		if err := insertClaim(ctx, tx, unmappedIDs[i], seq, projectID, batchID, nil, 0, "", u.Summary, u.Refs, s.Checked.Unmapped[i]); err != nil {
			return Receipt{}, err
		}
	}
	return readReceipt(ctx, tx, batchID, claimIDs, unmappedIDs)
}

// previousBatch is the comparison start of a sync batch in worktree (E3):
// the head of the latest sync or audit batch there, by seq, when that one
// was clean with a head; nothing otherwise, and no earlier batch is looked
// at. A manual batch never counts.
func previousBatch(ctx context.Context, tx *sql.Tx, projectID, worktree string) (string, bool, error) {
	var head sql.NullString
	var state string
	err := tx.QueryRowContext(ctx, `SELECT head, tree_state FROM claim_batches
WHERE project_id = ? AND worktree_id = ? AND kind IN ('sync', 'audit') ORDER BY seq DESC LIMIT 1`, projectID, worktree).Scan(&head, &state)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return head.String, state == Clean && head.Valid, nil
}

func scopeJSON(s *Scope) (string, error) {
	var v any = map[string]any{"all": true}
	if !s.All {
		v = map[string]any{"requirements": s.Requirements}
	}
	b, err := json.Marshal(v)
	return string(b), err
}

// storedFile is a file reference as refs_json keeps it.
type storedFile struct {
	Path  string `json:"path"`
	Lines string `json:"lines,omitempty"`
}

// storedRefs is refs_json: the references as submitted, each list there
// even when empty.
type storedRefs struct {
	Files   []storedFile `json:"files"`
	Tests   []string     `json:"tests"`
	Commits []string     `json:"commits"`
}

func insertClaim(ctx context.Context, tx *sql.Tx, id string, seq int64, projectID, batchID string, criterion *baseline.Entity, revision int,
	assertion, summary string, refs Refs, checked RefResults) error {
	stored := storedRefs{Files: []storedFile{}, Tests: []string{}, Commits: []string{}}
	for _, f := range refs.Files {
		stored.Files = append(stored.Files, storedFile{Path: f.Path, Lines: f.Lines})
	}
	stored.Tests = append(stored.Tests, refs.Tests...)
	stored.Commits = append(stored.Commits, refs.Commits...)
	refsJSON, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	checkJSON, err := json.Marshal(checked)
	if err != nil {
		return err
	}
	targetType, criterionID, criterionRevision, assertionValue := "none", any(nil), any(nil), any(nil)
	if criterion != nil {
		targetType, criterionID, criterionRevision, assertionValue = "criterion", criterion.ID, revision, assertion
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO claims (id, seq, project_id, batch_id, target_type, criterion_id, criterion_revision, assertion,
    summary, refs_json, refs_check_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, seq, projectID, batchID, targetType, criterionID, criterionRevision, assertionValue, summary,
		string(refsJSON), string(checkJSON))
	return err
}

// readReceipt builds the receipt from the rows written (WP2.4 spec §9.3
// step 1): the batch, and each claim by its ID, in input order.
func readReceipt(ctx context.Context, tx *sql.Tx, batchID string, claimIDs, unmappedIDs []string) (Receipt, error) {
	r := Receipt{Schema: SchemaID, BatchID: batchID, Claims: []ReceiptClaim{}, UnmappedWork: []ReceiptUnmapped{}}
	var head, compareFrom, compareSource sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT seq, kind, base_version, worktree_id, head, tree_state, compare_from, compare_from_source
FROM claim_batches WHERE id = ?`, batchID).Scan(&r.Seq, &r.Kind, &r.BaseVersion, &r.Context.WorktreeID, &head, &r.Context.TreeState,
		&compareFrom, &compareSource); err != nil {
		return Receipt{}, err
	}
	if head.Valid {
		r.Context.Head = &head.String
	}
	if compareSource.Valid {
		r.CompareFrom = &CompareFrom{Source: compareSource.String}
		if compareFrom.Valid {
			r.CompareFrom.Commit = &compareFrom.String
		}
	}
	type row struct {
		seq                 int64
		targetType          string
		key                 sql.NullString
		revision            sql.NullInt64
		assertion           sql.NullString
		refsJSON, refsCheck string
	}
	read := func(id string) (row, error) {
		var c row
		err := tx.QueryRowContext(ctx, `SELECT c.seq, c.target_type, a.key, c.criterion_revision, c.assertion, c.refs_json, c.refs_check_json
FROM claims c LEFT JOIN acceptance_criteria a ON a.id = c.criterion_id AND a.project_id = c.project_id WHERE c.id = ? AND c.batch_id = ?`,
			id, batchID).Scan(&c.seq, &c.targetType, &c.key, &c.revision, &c.assertion, &c.refsJSON, &c.refsCheck)
		if err == nil && c.seq != r.Seq {
			err = apperr.New(apperr.Internal, "a claim was written with another seq than its batch")
		}
		return c, err
	}
	refsOf := func(c row) (ReceiptRefs, error) {
		var stored storedRefs
		var checked RefResults
		if err := json.Unmarshal([]byte(c.refsJSON), &stored); err != nil {
			return ReceiptRefs{}, err
		}
		if err := json.Unmarshal([]byte(c.refsCheck), &checked); err != nil {
			return ReceiptRefs{}, err
		}
		out := ReceiptRefs{Files: []ReceiptFile{}, Commits: slices.Clone(checked.Commits), Tests: len(stored.Tests)}
		if out.Commits == nil {
			out.Commits = []CommitResult{}
		}
		for _, f := range checked.Files {
			out.Files = append(out.Files, ReceiptFile{Path: f.Path, Lines: f.Lines, Exists: f.Exists})
		}
		return out, nil
	}
	for i, id := range claimIDs {
		c, err := read(id)
		if err != nil {
			return Receipt{}, err
		}
		if c.targetType != "criterion" || !c.key.Valid || !c.revision.Valid || !c.assertion.Valid {
			return Receipt{}, apperr.New(apperr.Internal, "a claim was not written as a claim on a criterion")
		}
		refs, err := refsOf(c)
		if err != nil {
			return Receipt{}, err
		}
		r.Claims = append(r.Claims, ReceiptClaim{Index: i, ClaimID: id, Target: c.key.String, TargetRevision: int(c.revision.Int64),
			Assertion: c.assertion.String, Refs: refs})
	}
	for i, id := range unmappedIDs {
		c, err := read(id)
		if err != nil {
			return Receipt{}, err
		}
		if c.targetType != "none" {
			return Receipt{}, apperr.New(apperr.Internal, "unmapped work was not written as such")
		}
		refs, err := refsOf(c)
		if err != nil {
			return Receipt{}, err
		}
		r.UnmappedWork = append(r.UnmappedWork, ReceiptUnmapped{Index: i, ClaimID: id, Refs: refs})
	}
	return r, nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
