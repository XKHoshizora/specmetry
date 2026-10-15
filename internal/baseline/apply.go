package baseline

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/ulid"
)

// SourceRef is a checked source reference to bind to an entity's revision
// (data-model §4.5): under hash_only its locator and the excerpt's hash,
// never the excerpt. The JSON form is the one proposal items store.
type SourceRef struct {
	SourceID      string  `json:"source_id"`
	Locator       *string `json:"locator"`
	ExcerptSHA256 string  `json:"excerpt_sha256"`
	Verification  string  `json:"verification"`
	SourceSHA256  *string `json:"source_sha256"`
}

// CodeRef is a checked code reference to bind to an entity's revision.
type CodeRef struct {
	Path        string  `json:"path"`
	Lines       *string `json:"lines"`
	Exists      bool    `json:"exists"`
	CheckedHead *string `json:"checked_head"`
}

// Change is one accepted proposal item to apply.
type Change struct {
	ItemID string
	// Index is the item's item_index: the order within a step.
	Index int
	// Pointer locates the item in the submitted document, for errors, and
	// DataPointer its data there; empty when the data is not in it (an
	// item accepted as proposed).
	Pointer     string
	DataPointer string
	Op          string
	Type        Type
	TargetID    string
	Ref         string
	// Data is the data applied: everything for add, the fields to change
	// for update, nil for retire and reaffirm. Goals and requirements are
	// named by key or by a ref of the proposal.
	Data map[string]json.RawMessage
	// Edited marks accepted_with_edits.
	Edited bool
	// ChangeClass is the reviewer's class for an update; fields that force
	// material win.
	ChangeClass        string
	Origin, Confidence string
	Note               string
	// SourceRefs and CodeRefs are bound to the entity's revision after the
	// change: an add's references, an update's new ones or, for a material
	// update without new ones, its previous references checked again.
	SourceRefs []SourceRef
	CodeRefs   []CodeRef
	// CarrySourceRefs and CarryCodeRefs are an update target's current
	// references, checked again: a material update without new references
	// of a kind binds these to its new revision (data-model §4.5).
	CarrySourceRefs []SourceRef
	CarryCodeRefs   []CodeRef
}

// Rejection is one rejected proposal item.
type Rejection struct {
	ItemID string
	Note   string
}

// Followup is what a review decides for a criterion whose requirement it
// revised materially (review.v1 criterion_followups).
type Followup struct {
	Pointer      string
	Criterion    string
	SeenRevision int
	Action       string
	Rationale    string
}

// Plan is one review, or one human edit, to apply in a write transaction.
type Plan struct {
	ProjectID  string
	ProposalID string
	// ExpectedVersion is the baseline version the person read, at
	// VersionPointer (seen_base_version, or a human edit's base_version).
	ExpectedVersion int
	VersionPointer  string
	Seq             int64
	Actor, Channel  string
	// Summary is the new version's summary.
	Summary   string
	Accepted  []Change
	Rejected  []Rejection
	Followups []Followup
	// Applied maps the refs of this proposal's items accepted by earlier
	// reviews to their entities.
	Applied map[string]string
}

// Result is what applying a plan did.
type Result struct {
	// Version is the new baseline version, 0 when nothing was accepted.
	Version int
	// Keys are the keys assigned to accepted add items, by item ID.
	Keys map[string]string
	// Changes counts the changes recorded, cascades and follow-ups
	// included.
	Changes int
}

// Apply applies a review or a human edit in the caller's write transaction,
// in the fixed order of data-model §5.5. Every accepted item changes the
// baseline as one new version with every change recorded; items are
// marked reviewed once; pending items of other proposals whose targets
// changed are marked conflict. Nothing is accepted: no version, only the
// rejections. A stale version or follow-up revision is stale_precondition
// (exit code 5); anything that cannot apply is invalid input (exit code 2),
// every problem at its pointer. Either way the caller rolls back.
func Apply(ctx context.Context, tx *sql.Tx, plan Plan) (Result, error) {
	current, err := CurrentVersion(ctx, tx, plan.ProjectID)
	if err != nil {
		return Result{}, err
	}
	if current != plan.ExpectedVersion {
		return Result{}, StaleVersion(current, plan.ExpectedVersion, plan.VersionPointer)
	}
	a := &applier{ctx: ctx, tx: tx, plan: plan, version: current + 1, result: Result{Keys: map[string]string{}},
		refs: map[string]string{}, final: map[string]any{}, classes: map[string]string{}, changed: map[string]bool{},
		materialRequirements: map[string]bool{}, touched: map[string]bool{}, retiredHere: map[string]bool{}}
	for ref, id := range plan.Applied {
		a.refs[ref] = id
	}
	if len(plan.Accepted) == 0 {
		if len(plan.Followups) > 0 {
			a.problem(plan.Followups[0].Pointer, "follow-ups are for criteria whose requirement this review revises materially, and it accepts no item")
			return Result{}, a.err()
		}
		return Result{}, a.reviewed()
	}
	a.versionID = ulid.New()
	if _, err := tx.ExecContext(ctx, `INSERT INTO baseline_versions (id, seq, project_id, number, summary, actor, channel)
VALUES (?, ?, ?, ?, ?, ?, ?)`, a.versionID, plan.Seq, plan.ProjectID, a.version, plan.Summary, plan.Actor, plan.Channel); err != nil {
		return Result{}, err
	}
	accepted := slices.Clone(plan.Accepted)
	slices.SortStableFunc(accepted, func(x, y Change) int { return x.Index - y.Index })
	steps := []func(Change) (bool, error){
		a.step(Goal, "add", "update", "retire"),
		a.step(Requirement, "add", "update", "retire"),
		a.step(Criterion, "add"),
		a.step(Criterion, "update"),
		a.step(Criterion, "reaffirm"),
		a.step(Criterion, "retire"),
	}
	for _, step := range steps {
		for _, c := range accepted {
			if _, err := step(c); err != nil {
				return Result{}, err
			}
		}
	}
	if err := a.followups(); err != nil {
		return Result{}, err
	}
	if err := a.err(); err != nil {
		return Result{}, err
	}
	if err := a.questions(accepted); err != nil {
		return Result{}, err
	}
	if err := a.conflicts(); err != nil {
		return Result{}, err
	}
	if err := a.reviewed(); err != nil {
		return Result{}, err
	}
	a.result.Version = a.version
	return a.result, nil
}

// StaleVersion is the stale_precondition (exit code 5) of a review or edit
// made against version read when the baseline is at current: it points at
// the version read, and gives the current one.
func StaleVersion(current, read int, pointer string) error {
	e := apperr.Newf(apperr.StalePrecondition,
		"the baseline is at version %d, not %d; read it again (proposal show, baseline show) and decide against it", current, read).
		WithDetails(apperr.Detail{Pointer: pointer, Problem: fmt.Sprintf("is stale: the current baseline version is %d", current)})
	e.Current = map[string]any{"baseline_version": current}
	return e
}

type applier struct {
	ctx       context.Context
	tx        *sql.Tx
	plan      Plan
	version   int
	versionID string
	result    Result
	problems  []apperr.Detail
	stale     []apperr.Detail
	// refs maps the proposal's refs to the entities their items added.
	refs map[string]string
	// final holds each accepted item's applied data, and classes each
	// update's recorded class, by item ID.
	final   map[string]any
	classes map[string]string
	// changed holds every entity this version changed; touched every
	// criterion an accepted item changed; retiredHere every entity this
	// version retired; materialRequirements every requirement it revised
	// materially.
	changed, touched, retiredHere, materialRequirements map[string]bool
	changeIndex                                         int
}

func (a *applier) problem(pointer, format string, args ...any) {
	a.problems = append(a.problems, apperr.Detail{Pointer: pointer, Problem: fmt.Sprintf(format, args...)})
}

func (a *applier) err() error {
	if len(a.problems) > 0 {
		return apperr.New(apperr.InvalidInput, "the decisions cannot be applied; nothing was written").WithDetails(a.problems...)
	}
	if len(a.stale) > 0 {
		return apperr.New(apperr.StalePrecondition, "a criterion has changed since it was read; read the proposal again and decide against it").
			WithDetails(a.stale...)
	}
	return nil
}

// step applies the accepted items of type t whose op is one of ops.
func (a *applier) step(t Type, ops ...string) func(Change) (bool, error) {
	return func(c Change) (bool, error) {
		if c.Type != t || !slices.Contains(ops, c.Op) {
			return false, nil
		}
		var err error
		switch c.Op {
		case "add":
			err = a.add(c)
		case "update":
			err = a.update(c)
		case "retire":
			err = a.retire(c)
		case "reaffirm":
			err = a.reaffirm(c)
		}
		return true, err
	}
}

// record writes one baseline change.
func (a *applier) record(r Record, op, class string, before *Record, itemID, rationale string) error {
	var beforeJSON any
	if before != nil {
		b, err := json.Marshal(before.Snapshot())
		if err != nil {
			return err
		}
		beforeJSON = string(b)
	}
	after, err := json.Marshal(r.Snapshot())
	if err != nil {
		return err
	}
	if _, err := a.tx.ExecContext(a.ctx, `INSERT INTO baseline_changes (id, seq, project_id, version_id, change_index, entity_type,
    entity_id, op, change_class, before_json, after_json, proposal_item_id, rationale)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, ulid.New(), a.plan.Seq, a.plan.ProjectID, a.versionID, a.changeIndex, string(r.Type),
		r.ID, op, nullable(class), beforeJSON, string(after), nullable(itemID), nullable(rationale)); err != nil {
		return err
	}
	a.changeIndex++
	a.result.Changes++
	if op != "add" {
		a.changed[r.ID] = true
	}
	return nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// data reads the data field key: its JSON, and whether it is there.
func data(c Change, key string) (json.RawMessage, bool) {
	raw, ok := c.Data[key]
	return raw, ok
}

func isNull(raw json.RawMessage) bool { return string(bytes.TrimSpace(raw)) == "null" }

// str is the string raw holds, or "".
func str(raw json.RawMessage) string {
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}

func (a *applier) dataPointer(c Change, key string) string {
	if c.DataPointer != "" {
		return c.DataPointer + "/" + key
	}
	return c.Pointer
}

// link resolves a goal or requirement named by data: by key, or by a ref of
// the proposal, whose item must be accepted (now or by an earlier review).
// It must be active and not retired by this version.
func (a *applier) link(c Change, field string, t Type) (Record, bool, error) {
	raw, _ := data(c, field)
	name := str(raw)
	pointer := a.dataPointer(c, field)
	var rec Record
	var found bool
	var err error
	if _, isKey := TypeOfKey(name); isKey {
		rec, found, err = LoadKey(a.ctx, a.tx, a.plan.ProjectID, name)
		if err != nil {
			return Record{}, false, err
		}
		if !found || rec.Type != t {
			a.problem(pointer, "%s names no %s of the baseline", field, t)
			return Record{}, false, nil
		}
	} else {
		id, ok := a.refs[name]
		if !ok {
			a.problem(pointer, "%s names %s, an item of this proposal that is not accepted; accept it as well, or reject this item", field, name)
			return Record{}, false, nil
		}
		if rec, found, err = Load(a.ctx, a.tx, a.plan.ProjectID, t, id); err != nil || !found {
			return Record{}, false, err
		}
	}
	if !rec.Active() {
		reason := "it is retired"
		if a.retiredHere[rec.ID] {
			reason = "it is retired by this review"
		}
		a.problem(pointer, "%s names %s, but %s; reject this item", field, rec.Key, reason)
		return Record{}, false, nil
	}
	return rec, true, nil
}

func (a *applier) add(c Change) error {
	r := Record{ID: ulid.New(), Type: c.Type, Status: "active", Revision: 1, RevisionSeq: a.plan.Seq, Origin: c.Origin,
		CreatedVersion: a.version, UpdatedVersion: a.version, UpdatedSeq: a.plan.Seq}
	if c.Confidence != "" {
		conf := c.Confidence
		r.Confidence = &conf
	}
	final := map[string]any{}
	if err := a.setFields(c, &r, final); err != nil {
		return err
	}
	var requirement *Entity
	switch c.Type {
	case Requirement:
		if raw, ok := data(c, "goal"); ok && isNull(raw) {
			final["goal"] = nil
		} else if ok {
			goal, ok, err := a.link(c, "goal", Goal)
			if err != nil || !ok {
				return err
			}
			r.GoalID, r.GoalKey = &goal.ID, &goal.Key
			final["goal"] = goal.Key
		}
	case Criterion:
		req, ok, err := a.link(c, "requirement", Requirement)
		if err != nil || !ok {
			return err
		}
		// After step 3: the requirement's revision this version gives it.
		r.RequirementID, r.RequirementKey, r.RequirementRevision = req.ID, req.Key, req.Revision
		final["requirement"] = req.Key
		requirement = &Entity{ID: req.ID, Key: req.Key}
		if !a.consistentCheck(c, r) {
			return nil
		}
	}
	key, err := NextKey(a.ctx, a.tx, a.plan.ProjectID, c.Type, requirement)
	if err != nil {
		return err
	}
	r.Key = key
	if err := insert(a.ctx, a.tx, a.plan.ProjectID, r); err != nil {
		return err
	}
	a.refs[c.Ref] = r.ID
	a.result.Keys[c.ItemID] = key
	a.final[c.ItemID] = final
	if c.Type == Criterion {
		a.touched[r.ID] = true
	}
	if err := a.record(r, "add", "", nil, c.ItemID, ""); err != nil {
		return err
	}
	return a.bindRefs(r, c)
}

// setFields sets the fields data gives (all but links) on r, and copies
// them to final.
func (a *applier) setFields(c Change, r *Record, final map[string]any) error {
	for key, raw := range c.Data {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}
		switch key {
		case "goal", "requirement":
			continue
		case "title":
			r.Title = str(raw)
		case "description":
			d := str(raw)
			r.Description = &d
		case "kind":
			r.Kind = str(raw)
		case "priority":
			r.Priority = str(raw)
		case "statement":
			r.Statement = str(raw)
		case "verification":
			r.Verification = str(raw)
		case "check":
			if isNull(raw) {
				r.Check = nil
				break
			}
			ch := Check{Workdir: ".", TimeoutSec: 300}
			if err := json.Unmarshal(raw, &ch); err != nil {
				return err
			}
			r.Check = &ch
		default:
			return fmt.Errorf("unknown data field %q", key)
		}
		final[key] = v
	}
	return nil
}

// consistentCheck refuses a criterion with a check that is not automated
// (data-model §4.4, V-P04).
func (a *applier) consistentCheck(c Change, r Record) bool {
	if r.Check != nil && r.Verification != "automated" {
		field := "verification"
		if _, ok := data(c, "check"); ok {
			field = "check"
		}
		a.problem(a.dataPointer(c, field), "a check needs an automated criterion; %s would be %s", keyOr(r, "the criterion"), r.Verification)
		return false
	}
	return true
}

func keyOr(r Record, other string) string {
	if r.Key != "" {
		return r.Key
	}
	return other
}

// target loads an item's target, which must still be active.
func (a *applier) target(c Change) (Record, bool, error) {
	r, found, err := Load(a.ctx, a.tx, a.plan.ProjectID, c.Type, c.TargetID)
	if err != nil {
		return Record{}, false, err
	}
	if !found {
		a.problem(c.Pointer, "the item's target is not in the baseline")
		return Record{}, false, nil
	}
	if !r.Active() {
		if a.retiredHere[r.ID] {
			a.problem(c.Pointer, "%s is retired by this review with the entity it belongs to; reject this item", r.Key)
		} else {
			a.problem(c.Pointer, "%s is retired; only rejecting this item is possible", r.Key)
		}
		return Record{}, false, nil
	}
	return r, true, nil
}

// forcedMaterial reports whether a criterion update changes a field that
// always makes it material (data-model §7.2): verification, or the check's
// command, workdir, shell or inputs.
func forcedMaterial(before, after Record) bool {
	if before.Type != Criterion {
		return false
	}
	if before.Verification != after.Verification || (before.Check == nil) != (after.Check == nil) {
		return true
	}
	if before.Check == nil {
		return false
	}
	b, c := before.Check, after.Check
	inputsB, inputsC := slices.Clone(b.Inputs), slices.Clone(c.Inputs)
	slices.Sort(inputsB)
	slices.Sort(inputsC)
	return b.Command != c.Command || b.Workdir != c.Workdir || !equalShell(b.Shell, c.Shell) || !slices.Equal(inputsB, inputsC)
}

func equalShell(a, b *string) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}

func (a *applier) update(c Change) error {
	before, ok, err := a.target(c)
	if err != nil || !ok {
		return err
	}
	after := before
	if before.Check != nil {
		ch := *before.Check
		after.Check = &ch
	}
	final := map[string]any{}
	if err := a.setFields(c, &after, final); err != nil {
		return err
	}
	if c.Type == Requirement {
		if raw, ok := data(c, "goal"); ok {
			if isNull(raw) {
				after.GoalID, after.GoalKey = nil, nil
				final["goal"] = nil
			} else {
				goal, ok, err := a.link(c, "goal", Goal)
				if err != nil || !ok {
					return err
				}
				after.GoalID, after.GoalKey = &goal.ID, &goal.Key
				final["goal"] = goal.Key
			}
		}
	}
	if c.Type == Criterion {
		if !a.consistentCheck(c, after) {
			return nil
		}
		// An accepted update was decided under the requirement as it is
		// now (data-model §5.5 step 5).
		parent, _, err := Load(a.ctx, a.tx, a.plan.ProjectID, Requirement, after.RequirementID)
		if err != nil {
			return err
		}
		after.RequirementRevision = parent.Revision
		a.touched[after.ID] = true
	}
	class := c.ChangeClass
	if forcedMaterial(before, after) {
		class = "material"
	}
	if class == "material" {
		after.Revision++
		after.RevisionSeq = a.plan.Seq
		if c.Type == Requirement {
			a.materialRequirements[after.ID] = true
		}
	}
	after.UpdatedVersion, after.UpdatedSeq = a.version, a.plan.Seq
	if err := update(a.ctx, a.tx, a.plan.ProjectID, after); err != nil {
		return err
	}
	a.final[c.ItemID], a.classes[c.ItemID] = final, class
	if err := a.record(after, "update", class, &before, c.ItemID, ""); err != nil {
		return err
	}
	refs := c
	if class == "material" {
		if len(refs.SourceRefs) == 0 {
			refs.SourceRefs = c.CarrySourceRefs
		}
		if len(refs.CodeRefs) == 0 {
			refs.CodeRefs = c.CarryCodeRefs
		}
	}
	return a.bindRefs(after, refs)
}

// retireRecord retires r in this version and records it.
func (a *applier) retireRecord(r Record, itemID, rationale string) error {
	before := r
	v := a.version
	r.Status, r.RetiredVersion, r.UpdatedVersion, r.UpdatedSeq = "retired", &v, a.version, a.plan.Seq
	if err := update(a.ctx, a.tx, a.plan.ProjectID, r); err != nil {
		return err
	}
	a.retiredHere[r.ID] = true
	return a.record(r, "retire", "", &before, itemID, rationale)
}

func (a *applier) retire(c Change) error {
	r, ok, err := a.target(c)
	if err != nil || !ok {
		return err
	}
	a.final[c.ItemID] = nil
	if c.Type == Criterion {
		a.touched[r.ID] = true
	}
	if err := a.retireRecord(r, c.ItemID, ""); err != nil {
		return err
	}
	switch c.Type {
	case Goal:
		// Its requirements lose their goal (data-model §5.2): an editorial
		// change, their revision stays.
		reqs, err := LoadAll(a.ctx, a.tx, a.plan.ProjectID, Requirement)
		if err != nil {
			return err
		}
		for _, req := range reqs {
			if req.Active() && req.GoalID != nil && *req.GoalID == r.ID {
				before := req
				req.GoalID, req.GoalKey = nil, nil
				req.UpdatedVersion, req.UpdatedSeq = a.version, a.plan.Seq
				if err := update(a.ctx, a.tx, a.plan.ProjectID, req); err != nil {
					return err
				}
				if err := a.record(req, "update", "editorial", &before, "", ""); err != nil {
					return err
				}
			}
		}
	case Requirement:
		// Its criteria are retired with it, in the same version.
		criteria, err := LoadAll(a.ctx, a.tx, a.plan.ProjectID, Criterion)
		if err != nil {
			return err
		}
		for _, ac := range criteria {
			if ac.Active() && ac.RequirementID == r.ID {
				if err := a.retireRecord(ac, "", ""); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (a *applier) reaffirm(c Change) error {
	r, ok, err := a.target(c)
	if err != nil || !ok {
		return err
	}
	a.final[c.ItemID] = nil
	a.touched[r.ID] = true
	return a.reaffirmRecord(r, c.ItemID, "")
}

// reaffirmRecord confirms that a criterion applies to its requirement's
// current revision; its own revision stays.
func (a *applier) reaffirmRecord(r Record, itemID, rationale string) error {
	parent, _, err := Load(a.ctx, a.tx, a.plan.ProjectID, Requirement, r.RequirementID)
	if err != nil {
		return err
	}
	before := r
	r.RequirementRevision = parent.Revision
	r.UpdatedVersion, r.UpdatedSeq = a.version, a.plan.Seq
	if err := update(a.ctx, a.tx, a.plan.ProjectID, r); err != nil {
		return err
	}
	return a.record(r, "reaffirm", "", &before, itemID, rationale)
}

// followups applies the review's criterion follow-ups (step 8): only for a
// criterion of a requirement this version revised materially, that no
// accepted item changed, at the revision the person read.
func (a *applier) followups() error {
	seen := map[string]int{}
	for i, f := range a.plan.Followups {
		if prev, dup := seen[f.Criterion]; dup {
			a.problem(f.Pointer+"/criterion", "%s has a follow-up already (criterion_followups/%d)", f.Criterion, prev)
			continue
		}
		seen[f.Criterion] = i
		r, found, err := LoadKey(a.ctx, a.tx, a.plan.ProjectID, f.Criterion)
		if err != nil {
			return err
		}
		switch {
		case !found || r.Type != Criterion:
			a.problem(f.Pointer+"/criterion", "names no criterion of the baseline")
			continue
		case !r.Active():
			a.problem(f.Pointer+"/criterion", "%s is retired", f.Criterion)
			continue
		case !a.materialRequirements[r.RequirementID]:
			a.problem(f.Pointer+"/criterion", "%s belongs to a requirement this review does not revise materially; follow-ups are only for those", f.Criterion)
			continue
		case a.touched[r.ID]:
			a.problem(f.Pointer+"/criterion", "an accepted item of this review changes %s already", f.Criterion)
			continue
		case r.Revision != f.SeenRevision:
			a.stale = append(a.stale, apperr.Detail{Pointer: f.Pointer + "/seen_revision",
				Problem: fmt.Sprintf("is stale: %s is at revision %d", f.Criterion, r.Revision)})
			continue
		}
		var err2 error
		switch f.Action {
		case "reaffirm":
			err2 = a.reaffirmRecord(r, "", f.Rationale)
		case "retire":
			err2 = a.retireRecord(r, "", f.Rationale)
		case "defer":
			// It stays parent_revised until a later edit.
		}
		if err2 != nil {
			return err2
		}
	}
	return nil
}

// bindRefs binds an item's references to the entity's revision.
func (a *applier) bindRefs(r Record, c Change) error {
	for _, s := range c.SourceRefs {
		if _, err := a.tx.ExecContext(a.ctx, `INSERT INTO source_refs (id, seq, project_id, source_id, entity_type, entity_id,
    entity_revision, locator, excerpt, excerpt_sha256, verification, source_sha256)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?)`, ulid.New(), a.plan.Seq, a.plan.ProjectID, s.SourceID, string(r.Type), r.ID,
			r.Revision, s.Locator, s.ExcerptSHA256, s.Verification, s.SourceSHA256); err != nil {
			return err
		}
	}
	for _, cr := range c.CodeRefs {
		if _, err := a.tx.ExecContext(a.ctx, `INSERT INTO code_refs (id, seq, project_id, entity_type, entity_id, entity_revision,
    path, lines, checked_head, "exists")
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, ulid.New(), a.plan.Seq, a.plan.ProjectID, string(r.Type), r.ID, r.Revision,
			cr.Path, cr.Lines, cr.CheckedHead, cr.Exists); err != nil {
			return err
		}
	}
	return nil
}

// questions relates the open questions about an accepted add item to the
// entity it became (step 9). A question answered or dismissed meanwhile
// keeps naming the item.
func (a *applier) questions(accepted []Change) error {
	for _, c := range accepted {
		if c.Op != "add" {
			continue
		}
		if _, err := a.tx.ExecContext(a.ctx, `UPDATE questions SET related_type = ?, related_id = ?
WHERE project_id = ? AND proposal_item_id = ? AND status = 'open' AND related_id IS NULL`,
			string(c.Type), a.refs[c.Ref], a.plan.ProjectID, c.ItemID); err != nil {
			return err
		}
	}
	return nil
}

// conflicts marks pending items whose target this version changed: they
// were proposed against an earlier version (data-model §5.1).
func (a *applier) conflicts() error {
	reviewed := map[string]bool{}
	for _, c := range a.plan.Accepted {
		reviewed[c.ItemID] = true
	}
	for _, r := range a.plan.Rejected {
		reviewed[r.ItemID] = true
	}
	for id := range a.changed {
		rows, err := a.tx.QueryContext(a.ctx, `SELECT id FROM proposal_items WHERE project_id = ? AND target_id = ? AND review_state = 'pending'`,
			a.plan.ProjectID, id)
		if err != nil {
			return err
		}
		var items []string
		for rows.Next() {
			var item string
			if err := rows.Scan(&item); err != nil {
				_ = rows.Close()
				return err
			}
			if !reviewed[item] {
				items = append(items, item)
			}
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, item := range items {
			if _, err := a.tx.ExecContext(a.ctx, `UPDATE proposal_items SET review_state = 'conflict' WHERE id = ?`, item); err != nil {
				return err
			}
		}
	}
	return nil
}

// reviewed writes the review fields of every decided item, once.
func (a *applier) reviewed() error {
	var versionID any
	if a.versionID != "" {
		versionID = a.versionID
	}
	write := func(itemID, state string, final any, class, note string, version any) error {
		var finalJSON any
		if final != nil {
			b, err := json.Marshal(final)
			if err != nil {
				return err
			}
			finalJSON = string(b)
		}
		res, err := a.tx.ExecContext(a.ctx, `UPDATE proposal_items SET review_state = ?, final_payload_json = ?, change_class = ?,
    reviewed_by = ?, reviewed_channel = ?, reviewed_seq = ?, review_note = ?, applied_version_id = ?
WHERE project_id = ? AND id = ? AND review_state IN ('pending', 'conflict')`,
			state, finalJSON, nullable(class), a.plan.Actor, a.plan.Channel, a.plan.Seq, nullable(note), version, a.plan.ProjectID, itemID)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil || n != 1 {
			return apperr.New(apperr.InvalidInput, "an item was reviewed meanwhile; nothing was written")
		}
		return nil
	}
	for _, c := range a.plan.Accepted {
		state := "accepted"
		if c.Edited {
			state = "accepted_with_edits"
		}
		var final any
		if f, ok := a.final[c.ItemID]; ok && f != nil {
			final = f
		}
		if err := write(c.ItemID, state, final, a.classes[c.ItemID], c.Note, versionID); err != nil {
			return err
		}
	}
	for _, r := range a.plan.Rejected {
		if err := write(r.ItemID, "rejected", nil, "", r.Note, nil); err != nil {
			return err
		}
	}
	return nil
}

// ChangedSince reports whether a version after version changed the entity.
func ChangedSince(ctx context.Context, q Querier, projectID, entityID string, version int) (bool, error) {
	var n int
	err := q.QueryRowContext(ctx, `SELECT count(*) FROM baseline_changes c JOIN baseline_versions v ON v.id = c.version_id
WHERE c.project_id = ? AND c.entity_id = ? AND v.number > ?`, projectID, entityID, version).Scan(&n)
	return n > 0, err
}

// SnapshotAt returns the entity as it was in version: the after snapshot
// of its last change up to that version.
func SnapshotAt(ctx context.Context, q Querier, projectID, entityID string, version int) (map[string]any, bool, error) {
	var raw string
	err := q.QueryRowContext(ctx, `SELECT c.after_json FROM baseline_changes c JOIN baseline_versions v ON v.id = c.version_id
WHERE c.project_id = ? AND c.entity_id = ? AND v.number <= ? ORDER BY v.number DESC, c.change_index DESC LIMIT 1`,
		projectID, entityID, version).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var s map[string]any
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return nil, false, err
	}
	return s, true, nil
}
