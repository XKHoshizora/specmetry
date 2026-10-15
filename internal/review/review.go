// Package review checks a person's review of a proposal (review.v1) against
// the stored proposal and turns it into a baseline plan (data-model §5.1,
// §5.5; architecture §6.6). Applying it is baseline.Apply's.
package review

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/proposal"
)

// Document is a review.v1 submission that passed schema validation.
type Document struct {
	ProposalID      string          `json:"proposal_id"`
	SeenBaseVersion int             `json:"seen_base_version"`
	Summary         string          `json:"summary"`
	Items           []Decision      `json:"items"`
	Followups       []Followup      `json:"criterion_followups"`
	Scope           json.RawMessage `json:"scope"`
}

// Decision is the decision on one item.
type Decision struct {
	ItemID               string                     `json:"item_id"`
	Decision             string                     `json:"decision"`
	FinalData            map[string]json.RawMessage `json:"final_data"`
	ChangeClass          string                     `json:"change_class"`
	AcknowledgedConflict bool                       `json:"acknowledged_conflict"`
	Note                 string                     `json:"note"`
}

// Followup is what the review decides for a criterion of a requirement it
// revises materially.
type Followup struct {
	Criterion    string `json:"criterion"`
	SeenRevision int    `json:"seen_revision"`
	Action       string `json:"action"`
	Rationale    string `json:"rationale"`
}

// Parse decodes a document that passed review.v1 validation.
func Parse(raw []byte) (Document, error) {
	var d Document
	if err := json.Unmarshal(raw, &d); err != nil {
		return Document{}, apperr.New(apperr.InvalidInput, "the review is not valid")
	}
	return d, nil
}

// Validator validates a document against an exchange schema.
type Validator interface {
	ValidateValue(name string, value any) error
}

// Checked is a review checked against the stored proposal.
type Checked struct {
	Proposal proposal.Stored
	// Plan applies the review; the caller sets its seq, actor and channel.
	Plan baseline.Plan
	// Carry are the references that material updates without new ones
	// would carry over, to check again outside the transaction.
	Carry []Carry
	// Lines summarize the review for the person to confirm.
	Lines []string
}

// Carry is an entity's current references, which a material update
// without new references binds to its new revision once checked again
// (data-model §4.5, MVP-BL-22).
type Carry struct {
	ItemID     string
	EntityID   string
	Revision   int
	SourceRefs []baseline.SourceRef
	CodeRefs   []baseline.CodeRef
}

type problems []apperr.Detail

func (ps *problems) add(pointer, format string, args ...any) {
	*ps = append(*ps, apperr.Detail{Pointer: pointer, Problem: fmt.Sprintf(format, args...)})
}

func itemPointer(i int, rest ...string) string {
	return "/items/" + strconv.Itoa(i) + strings.Join(append([]string{""}, rest...), "/")
}

// CheckVersion compares the current baseline version, in the caller's
// transaction, with the review's seen_base_version: a stale review is
// stale_precondition (exit code 5) whatever else is wrong with it, so it
// comes before every other check (codex-review-025 B2). baseline.Apply
// checks it again.
func CheckVersion(ctx context.Context, q baseline.Querier, projectID string, seen int) error {
	current, err := baseline.CurrentVersion(ctx, q, projectID)
	if err != nil {
		return err
	}
	if current != seen {
		return baseline.StaleVersion(current, seen, "/seen_base_version")
	}
	return nil
}

// Check checks the review against the proposal as stored, in the caller's
// transaction: first that seen_base_version is current (CheckVersion);
// then every item is the proposal's and still undecided, decided once; an item in conflict is acknowledged; an accepted update carries
// the reviewer's change_class (V-R01); edited data is valid item data; a
// delivery scope cannot be confirmed yet (WP1.8). Problems are invalid
// input (exit code 2) at their pointers; nothing is written.
func Check(ctx context.Context, tx *sql.Tx, projectID string, doc Document, val Validator) (Checked, error) {
	if err := CheckVersion(ctx, tx, projectID, doc.SeenBaseVersion); err != nil {
		return Checked{}, err
	}
	if doc.Scope != nil && string(doc.Scope) != "null" {
		return Checked{}, apperr.New(apperr.InvalidInput,
			"confirming the delivery scope in a review is not available in this version (it arrives with scope confirm, WP1.8); review without scope, and confirm the scope later").
			WithDetails(apperr.Detail{Pointer: "/scope", Problem: "is not supported yet; remove it"})
	}
	p, found, err := proposal.Load(ctx, tx, projectID, doc.ProposalID)
	if err != nil {
		return Checked{}, err
	}
	if !found {
		return Checked{}, apperr.New(apperr.NotFound, "the project has no such proposal (see proposal list)").
			WithDetails(apperr.Detail{Pointer: "/proposal_id", Problem: "names no proposal of this project"})
	}
	items := map[string]proposal.StoredItem{}
	added := map[string]baseline.Type{}
	for _, it := range p.Items {
		items[it.ID] = it
		if it.Op == "add" {
			added[it.Ref] = it.Type
		}
	}
	appliedIDs, _, err := proposal.AppliedRefs(ctx, tx, projectID, p.ID)
	if err != nil {
		return Checked{}, err
	}
	c := Checked{Proposal: p, Plan: baseline.Plan{ProjectID: projectID, ProposalID: p.ID, ExpectedVersion: doc.SeenBaseVersion,
		VersionPointer: "/seen_base_version", Summary: doc.Summary, Applied: appliedIDs}}
	if c.Plan.Summary == "" {
		c.Plan.Summary = p.Summary
	}
	var ps problems
	seen := map[string]int{}
	pending := 0
	for _, it := range p.Items {
		if !it.Reviewed() {
			pending++
		}
	}
	for i, d := range doc.Items {
		if prev, dup := seen[d.ItemID]; dup {
			ps.add(itemPointer(i, "item_id"), "is decided by items/%d already", prev)
			continue
		}
		seen[d.ItemID] = i
		it, ok := items[d.ItemID]
		if !ok {
			ps.add(itemPointer(i, "item_id"), "names no item of proposal %s", p.ID)
			continue
		}
		if it.Reviewed() {
			ps.add(itemPointer(i, "item_id"), "was %s already (seq %d); an item is reviewed once", it.State, it.ReviewedSeq)
			continue
		}
		conflict, err := proposal.InConflict(ctx, tx, projectID, p, it)
		if err != nil {
			return Checked{}, err
		}
		if conflict && !d.AcknowledgedConflict {
			ps.add(itemPointer(i, "acknowledged_conflict"),
				"%s changed after the proposal's base version %d; look at the three-way comparison (proposal show) and set acknowledged_conflict to true",
				it.TargetKey, p.BaseVersion)
		}
		if d.Decision == "rejected" {
			c.Plan.Rejected = append(c.Plan.Rejected, baseline.Rejection{ItemID: it.ID, Note: d.Note})
			c.Lines = append(c.Lines, line(it, "reject", ""))
			continue
		}
		ch, ok := change(i, it, d, added, val, &ps)
		if !ok {
			continue
		}
		c.Plan.Accepted = append(c.Plan.Accepted, ch)
		verb := "accept"
		if ch.Edited {
			verb = "accept with edits"
		}
		if conflict {
			verb += " (conflict acknowledged)"
		}
		c.Lines = append(c.Lines, line(it, verb, ch.ChangeClass))
		if it.Op == "update" && (len(it.Sources) == 0 || len(it.CodeRefs) == 0) {
			carry, err := CurrentRefs(ctx, tx, projectID, it.ID, it.Type, it.TargetID)
			if err != nil {
				return Checked{}, err
			}
			if len(it.Sources) > 0 {
				carry.SourceRefs = nil
			}
			if len(it.CodeRefs) > 0 {
				carry.CodeRefs = nil
			}
			if len(carry.SourceRefs)+len(carry.CodeRefs) > 0 {
				c.Carry = append(c.Carry, carry)
			}
		}
	}
	for k, f := range doc.Followups {
		c.Plan.Followups = append(c.Plan.Followups, baseline.Followup{Pointer: "/criterion_followups/" + strconv.Itoa(k),
			Criterion: f.Criterion, SeenRevision: f.SeenRevision, Action: f.Action, Rationale: f.Rationale})
		c.Lines = append(c.Lines, fmt.Sprintf("follow-up: %s %s", f.Action, f.Criterion))
	}
	if len(ps) > 0 {
		return Checked{}, apperr.New(apperr.InvalidInput, "the review is not valid; nothing was written").WithDetails(ps...)
	}
	accepted, rejected := len(c.Plan.Accepted), len(c.Plan.Rejected)
	head := []string{fmt.Sprintf("proposal %s by %s: %s", p.ID, p.Actor, p.Summary)}
	if accepted > 0 {
		head = append(head, fmt.Sprintf("%d accepted and %d rejected become baseline version %d (now %d)",
			accepted, rejected, doc.SeenBaseVersion+1, doc.SeenBaseVersion))
	} else {
		head = append(head, fmt.Sprintf("%d rejected; nothing is accepted, so the baseline stays at version %d", rejected, doc.SeenBaseVersion))
	}
	if left := pending - accepted - rejected; left > 0 {
		head = append(head, fmt.Sprintf("%d items of the proposal stay undecided", left))
	}
	c.Lines = append(head, c.Lines...)
	return c, nil
}

func line(it proposal.StoredItem, verb, class string) string {
	subject := it.TargetKey
	if it.Op == "add" {
		subject = "ref " + it.Ref
	}
	s := fmt.Sprintf("item %d: %s %s %s %s", it.Index, verb, it.Op, it.Type, subject)
	if class != "" {
		s += " (" + class + ")"
	}
	return s
}

// change turns an accepted decision into a baseline change, or reports its
// problems.
func change(i int, it proposal.StoredItem, d Decision, added map[string]baseline.Type, val Validator, ps *problems) (baseline.Change, bool) {
	before := len(*ps)
	ch := baseline.Change{ItemID: it.ID, Index: it.Index, Pointer: itemPointer(i), Op: it.Op, Type: it.Type, TargetID: it.TargetID,
		Ref: it.Ref, Data: it.Payload, Origin: it.Origin, Confidence: it.Confidence, Note: d.Note,
		SourceRefs: it.Sources, CodeRefs: it.CodeRefs}
	switch {
	case d.Decision == "accepted" && d.FinalData != nil:
		ps.add(itemPointer(i, "final_data"), "is for accepted_with_edits; accepted applies the item as proposed")
	case d.Decision == "accepted_with_edits" && (it.Op == "retire" || it.Op == "reaffirm"):
		ps.add(itemPointer(i, "decision"), "a %s item has no data to edit; accept or reject it", it.Op)
	case d.Decision == "accepted_with_edits":
		ch.Data, ch.Edited, ch.DataPointer = d.FinalData, true, itemPointer(i, "final_data")
		checkFinalData(i, it, d.FinalData, added, val, ps)
	}
	switch {
	case it.Op == "update" && d.ChangeClass == "":
		ps.add(itemPointer(i, "change_class"),
			"an accepted update needs the reviewer's change_class, material or editorial (V-R01); the class the proposal suggests is not used")
	case it.Op != "update" && d.ChangeClass != "":
		ps.add(itemPointer(i, "change_class"), "is only for update items")
	}
	ch.ChangeClass = d.ChangeClass
	return ch, len(*ps) == before
}

// checkFinalData validates edited data as the item's data: against
// proposal.v1, with the rules on fences, paths and refs.
func checkFinalData(i int, it proposal.StoredItem, data map[string]json.RawMessage, added map[string]baseline.Type, val Validator, ps *problems) {
	pointer := itemPointer(i, "final_data")
	item := map[string]any{"op": it.Op, "type": string(it.Type), "data": data}
	if it.Op == "add" {
		item["ref"], item["origin"] = it.Ref, "manual"
	} else {
		item["target"], item["target_revision"] = it.TargetKey, it.TargetRevision
	}
	doc := map[string]any{"schema": "specmetry.proposal/v1", "origin": "cli", "base_version": 0, "summary": "review", "items": []any{item}}
	if err := val.ValidateValue("proposal.v1", doc); err != nil {
		details := apperr.From(err).Details
		if len(details) == 0 {
			ps.add(pointer, "is not valid %s data for a %s item", it.Type, it.Op)
		}
		for _, d := range details {
			at := pointer
			if rest, ok := strings.CutPrefix(d.Pointer, "/items/0/data"); ok {
				at += rest
			}
			ps.add(at, "%s", d.Problem)
		}
		return
	}
	*ps = append(*ps, proposal.CheckEditedData(data, pointer)...)
	for field, want := range map[string]baseline.Type{"goal": baseline.Goal, "requirement": baseline.Requirement} {
		raw, ok := data[field]
		if !ok {
			continue
		}
		var name string
		if json.Unmarshal(raw, &name) != nil || !proposal.IsLocalRef(name) {
			continue
		}
		switch typ, ok := added[name]; {
		case !ok:
			ps.add(pointer+"/"+field, "names no item added by this proposal")
		case typ != want:
			ps.add(pointer+"/"+field, "names a %s; it must name a %s", typ, want)
		}
	}
}

// CurrentRefs reads the references an update item's target has at its
// current revision.
func CurrentRefs(ctx context.Context, tx *sql.Tx, projectID, itemID string, t baseline.Type, targetID string) (Carry, error) {
	c := Carry{ItemID: itemID, EntityID: targetID}
	r, found, err := baseline.Load(ctx, tx, projectID, t, targetID)
	if err != nil || !found {
		return c, err
	}
	c.Revision = r.Revision
	rows, err := tx.QueryContext(ctx, `SELECT source_id, locator, excerpt_sha256, verification, source_sha256 FROM source_refs
WHERE project_id = ? AND entity_id = ? AND entity_revision = ? ORDER BY seq, rowid`, projectID, targetID, r.Revision)
	if err != nil {
		return c, err
	}
	for rows.Next() {
		var s baseline.SourceRef
		var locator, sha sql.NullString
		if err := rows.Scan(&s.SourceID, &locator, &s.ExcerptSHA256, &s.Verification, &sha); err != nil {
			_ = rows.Close()
			return c, err
		}
		if locator.Valid {
			s.Locator = &locator.String
		}
		if sha.Valid {
			s.SourceSHA256 = &sha.String
		}
		c.SourceRefs = append(c.SourceRefs, s)
	}
	if err := rows.Close(); err != nil {
		return c, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT path, lines FROM code_refs
WHERE project_id = ? AND entity_id = ? AND entity_revision = ? ORDER BY seq, rowid`, projectID, targetID, r.Revision)
	if err != nil {
		return c, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cr baseline.CodeRef
		var lines sql.NullString
		if err := rows.Scan(&cr.Path, &lines); err != nil {
			return c, err
		}
		if lines.Valid {
			cr.Lines = &lines.String
		}
		c.CodeRefs = append(c.CodeRefs, cr)
	}
	return c, rows.Err()
}

// Carried applies the references checked again (by item) to the plan's
// material updates: an update without new references of a kind binds the
// carried ones of that kind if it turns out material.
func (c *Checked) Carried(checked map[string]Carry) error {
	for _, need := range c.Carry {
		got, ok := checked[need.ItemID]
		if !ok || got.EntityID != need.EntityID || got.Revision != need.Revision ||
			len(got.SourceRefs) != len(need.SourceRefs) || len(got.CodeRefs) != len(need.CodeRefs) {
			return apperr.New(apperr.StalePrecondition, "the baseline changed while the review was being confirmed; read the proposal again")
		}
		for k := range c.Plan.Accepted {
			if c.Plan.Accepted[k].ItemID == need.ItemID {
				c.Plan.Accepted[k].CarrySourceRefs = slices.Clone(got.SourceRefs)
				c.Plan.Accepted[k].CarryCodeRefs = slices.Clone(got.CodeRefs)
			}
		}
	}
	return nil
}
