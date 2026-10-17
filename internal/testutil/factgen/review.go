package factgen

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/XKHoshizora/specmetry/internal/baseline"
)

// Review is a review of one proposal's pending items, or a person's edit
// (propose --apply), being planned. Its methods only plan; when the
// function given to History.Review or History.Apply returns, the plan is
// checked and applied as baseline.Apply applies one (data-model §5.5):
//
//   - every explicit change comes from a proposal item: in a review, the
//     item named with FromItem right after the change is planned; in an
//     edit, a new item of the edit's proposal, in planning order;
//   - one proposal is reviewed, and it names a target at most once
//     (V-P03);
//   - the changes apply by step (goals; requirements; criterion adds,
//     updates, reaffirms, retires) and within a step by item_index, then
//     the follow-ups of step 8; a cascade follows its cause;
//   - references are written for the entities the write changes, at their
//     new revision; questions about an accepted add item are linked to its
//     entity if they are still open (step 9); an attached scope binds to
//     the new version.
//
// A plan that breaks a rule panics: a test or the generator cannot write
// it as a legal history.
type Review struct {
	h        *History
	seq      int64
	apply    bool
	proposal string
	ops      []*planned
	rejected []string
	refs     []plannedRef
	scope    []string
	scoped   bool
}

type planned struct {
	op        string // add / update / retire / reaffirm
	e         *entity
	material  bool
	edit      func(f *Fields)
	item      string
	index     int
	followup  bool
	rationale string
}

type plannedRef struct {
	entity, source, verification string
}

// Review reviews pending items of one proposal. A version is written when
// at least one item is accepted.
func (h *History) Review(fn func(r *Review)) int64 {
	r := &Review{h: h, seq: h.next()}
	fn(r)
	r.finish()
	return r.seq
}

// Apply is a person's edit (propose --apply): the proposal, its accepted
// items and the version share one seq (architecture §6.1.5).
func (h *History) Apply(fn func(r *Review)) int64 {
	r := &Review{h: h, seq: h.next(), apply: true, proposal: h.id()}
	fn(r)
	r.finish()
	return r.seq
}

// Seq is the write's seq.
func (r *Review) Seq() int64 { return r.seq }

func (r *Review) plan(op string, e *entity) {
	r.ops = append(r.ops, &planned{op: op, e: e})
}

func (r *Review) last() *planned {
	for i := len(r.ops) - 1; i >= 0; i-- {
		if !r.ops[i].followup {
			return r.ops[i]
		}
	}
	panic("factgen: FromItem without a planned change")
}

func (r *Review) add(e *entity) string {
	e.id = r.h.id()
	if e.origin == "" {
		e.origin = "extracted"
	}
	r.h.ents[e.id] = e
	r.plan("add", e)
	return e.id
}

// AddGoal plans a goal's addition.
func (r *Review) AddGoal(title string) string {
	return r.add(&entity{typ: "goal", title: title})
}

// AddRequirement plans a requirement's addition, under a goal or none ("").
func (r *Review) AddRequirement(goal, priority string) string {
	return r.AddRequirementWithOrigin(goal, priority, "extracted")
}

// AddRequirementWithOrigin plans a requirement's addition with an origin.
func (r *Review) AddRequirementWithOrigin(goal, priority, origin string) string {
	return r.add(&entity{typ: "requirement", origin: origin, goalID: goal, priority: priority})
}

// AddCriterion plans a criterion's addition to a requirement; it applies to
// the requirement's revision after the requirements' step.
func (r *Review) AddCriterion(requirement, verification string, check *baseline.Check) string {
	return r.AddCriterionWithOrigin(requirement, verification, check, "extracted")
}

// AddCriterionWithOrigin plans a criterion's addition with an origin.
func (r *Review) AddCriterionWithOrigin(requirement, verification string, check *baseline.Check, origin string) string {
	return r.add(&entity{typ: "criterion", origin: origin, reqID: requirement, statement: "statement", verification: verification, check: check})
}

// Fields are the editable fields of an entity.
type Fields struct {
	Title        string
	Goal         string // requirements
	Priority     string // requirements
	Statement    string // criteria
	Verification string // criteria
	Check        *baseline.Check
}

func (e *entity) fields() Fields {
	return Fields{Title: e.title, Goal: e.goalID, Priority: e.priority, Statement: e.statement, Verification: e.verification, Check: e.check}
}

func (e *entity) set(f Fields) {
	e.title, e.goalID, e.priority, e.statement, e.verification, e.check = f.Title, f.Goal, f.Priority, f.Statement, f.Verification, f.Check
}

// Update plans an entity's update. A material update increases its
// revision; a change of a criterion's verification or check is material
// whatever the class given (data-model §7.2); an update of a criterion, of
// either class, confirms it for its requirement's revision (§5.5).
func (r *Review) Update(id string, material bool, edit func(f *Fields)) {
	r.ops = append(r.ops, &planned{op: "update", e: r.h.ents[id], material: material, edit: edit})
}

// Reaffirm plans a criterion's reaffirmation for its requirement's
// revision.
func (r *Review) Reaffirm(criterion string) { r.plan("reaffirm", r.h.ents[criterion]) }

// Retire plans an entity's retirement, with the cascade of data-model
// §5.2: a requirement's criteria are retired, a goal's requirements lose
// it.
func (r *Review) Retire(id string) { r.plan("retire", r.h.ents[id]) }

// FollowUp plans a criterion follow-up of a review (step 8): reaffirm or
// retire a criterion whose requirement this review revises materially and
// that no item changes. It has no item; its rationale is recorded.
func (r *Review) FollowUp(criterion, action string) {
	if r.apply {
		panic("factgen: a person's edit has no follow-ups")
	}
	r.ops = append(r.ops, &planned{op: action, e: r.h.ents[criterion], followup: true, rationale: "the reviewer's follow-up"})
}

// FromItem names the pending proposal item the last planned change
// applies; the item is accepted. Every explicit change of a review needs
// one.
func (r *Review) FromItem(item string) {
	if r.apply {
		panic("factgen: an edit's changes get their own items")
	}
	r.useItem(item)
	r.last().item = item
}

// RejectItem rejects a pending proposal item.
func (r *Review) RejectItem(item string) {
	r.useItem(item)
	r.rejected = append(r.rejected, item)
}

func (r *Review) useItem(item string) {
	p, ok := r.h.itemProposal[item]
	switch {
	case !ok:
		panic("factgen: an unknown proposal item")
	case r.proposal != "" && p != r.proposal:
		panic("factgen: a review reviews one proposal")
	}
	r.proposal = p
	for _, it := range r.h.d.ProposalItems {
		if it.ID == item && it.ReviewState != "pending" {
			panic("factgen: an item is reviewed once")
		}
	}
}

// Ref plans a source reference of an entity this write changes, at its new
// revision.
func (r *Review) Ref(entity, src, verification string) {
	r.refs = append(r.refs, plannedRef{entity: entity, source: src, verification: verification})
}

// Scope attaches a scope confirmation of these requirements, confirmed
// after the write's changes.
func (r *Review) Scope(requirements ...string) {
	r.scoped, r.scope = true, requirements
}

// step is the order of data-model §5.5 (baseline.Apply's steps).
func step(op *planned) int {
	switch {
	case op.followup:
		return 6
	case op.e.typ == "goal":
		return 0
	case op.e.typ == "requirement":
		return 1
	case op.op == "add":
		return 2
	case op.op == "update":
		return 3
	case op.op == "reaffirm":
		return 4
	}
	return 5
}

type applying struct {
	r            *Review
	versionID    string
	changes      int
	changed      map[string]bool
	material     map[string]bool // requirements revised materially here
	touched      map[string]bool // criteria an item changes here
	retiredHere  map[string]bool
	appliedItems map[string]string // item ID -> entity it produced
	addItems     map[string]bool
}

func (r *Review) finish() {
	h := r.h
	if r.apply {
		h.d.Proposals = append(h.d.Proposals, ProposalRow{ID: r.proposal, Seq: r.seq})
	}
	// Bind the explicit changes to items and check V-P03.
	if h.targeted[r.proposal] == nil {
		h.targeted[r.proposal] = map[string]bool{}
	}
	index := map[string]int{}
	for _, it := range h.d.ProposalItems {
		index[it.ID] = it.ItemIndex
	}
	n := 0
	for _, op := range r.ops {
		if op.followup {
			continue
		}
		if r.apply {
			op.item, op.index = h.id(), n
			h.itemProposal[op.item] = r.proposal
			h.d.ProposalItems = append(h.d.ProposalItems, ProposalItemRow{ID: op.item, ProposalID: r.proposal, ItemIndex: n, ReviewState: "pending"})
			n++
		} else {
			if op.item == "" {
				panic(fmt.Sprintf("factgen: a review's %s of a %s needs a proposal item", op.op, op.e.typ))
			}
			op.index = index[op.item]
		}
		if op.op != "add" {
			if h.targeted[r.proposal][op.e.id] {
				panic(fmt.Sprintf("factgen: the proposal names %s twice (V-P03)", op.e.key))
			}
			h.targeted[r.proposal][op.e.id] = true
		}
	}
	ops := slices.Clone(r.ops)
	slices.SortStableFunc(ops, func(a, b *planned) int {
		if c := cmp.Compare(step(a), step(b)); c != 0 || a.followup {
			return c
		}
		return cmp.Compare(a.index, b.index)
	})
	a := &applying{
		r: r, versionID: h.id(), changed: map[string]bool{}, material: map[string]bool{}, touched: map[string]bool{},
		retiredHere: map[string]bool{}, appliedItems: map[string]string{}, addItems: map[string]bool{},
	}
	var lines []string
	for _, op := range ops {
		a.apply(op)
		lines = append(lines, fmt.Sprintf("%s %s %s", op.op, op.e.typ, op.e.key))
	}
	for _, ref := range r.refs {
		e := h.ents[ref.entity]
		if !a.changed[ref.entity] {
			panic(fmt.Sprintf("factgen: a reference of %s, which the write does not change", e.key))
		}
		h.d.SourceRefs = append(h.d.SourceRefs, SourceRefRow{
			ID: h.id(), Seq: r.seq, SourceID: ref.source, EntityType: e.typ, EntityID: e.id, EntityRevision: e.revision, Verification: ref.verification,
		})
	}
	// The items: accepted with their change, or rejected.
	for i := range h.d.ProposalItems {
		it := &h.d.ProposalItems[i]
		_, accepted := a.appliedItems[it.ID]
		switch {
		case accepted:
			it.ReviewState, it.ReviewedSeq = "accepted", ptr(r.seq)
		case slices.Contains(r.rejected, it.ID):
			it.ReviewState, it.ReviewedSeq = "rejected", ptr(r.seq)
		}
	}
	// Step 9: an open question about an accepted add item is linked to its
	// entity. A question about an item that is accepted as no add is no
	// legal question (only an add item has a proposal-local ref).
	for i := range h.d.Questions {
		q := &h.d.Questions[i]
		if q.ProposalItemID == nil {
			continue
		}
		entity, accepted := a.appliedItems[*q.ProposalItemID]
		switch {
		case !accepted:
		case !a.addItems[*q.ProposalItemID]:
			panic("factgen: a question about an item that is accepted as no add")
		case q.AnsweredSeq == nil && q.RelatedID == nil:
			q.RelatedType, q.RelatedID = ptr(h.ents[entity].typ), ptr(entity)
		}
	}
	if a.changes > 0 {
		h.version++
		h.d.BaselineVersions = append(h.d.BaselineVersions, VersionRow{ID: a.versionID, Seq: r.seq, Number: h.version})
	}
	if r.scoped {
		for _, req := range r.scope {
			if !h.Active(req) {
				panic("factgen: a scope of a requirement that is not active")
			}
		}
		h.confirm(r.seq, r.scope)
	}
	kind := "review of " + r.proposal
	if r.apply {
		kind = "edit " + r.proposal
	}
	h.log("%s: %v; rejected %d; scope %v", kind, lines, len(r.rejected), h.keysOf(r.scope))
}

// apply applies one planned change, as baseline.Apply's steps do.
func (a *applying) apply(op *planned) {
	h, e := a.r.h, op.e
	if op.op != "add" && e.status != "active" {
		why := "is not active"
		if a.retiredHere[e.id] {
			why = "is retired by this write with the entity it belongs to"
		}
		panic(fmt.Sprintf("factgen: %s %s: %s %s", op.op, e.typ, e.key, why))
	}
	switch op.op {
	case "add":
		a.addEntity(e)
		a.record(e, "add", nil, op)
	case "update":
		if e.typ == "requirement" && e.goalID != "" && !h.Active(e.goalID) {
			panic("factgen: an update of a requirement whose goal this write retires")
		}
		before := e.fields()
		f := e.fields()
		if op.edit != nil {
			op.edit(&f)
		}
		e.set(f)
		material := op.material || before.Verification != f.Verification || !sameCheck(before.Check, f.Check)
		class := "editorial"
		if material {
			class = "material"
			e.revision++
			e.revisionSeq = a.r.seq
			if e.typ == "requirement" {
				a.material[e.id] = true
			}
		}
		if e.typ == "criterion" {
			e.reqRevision = h.ents[e.reqID].revision
		}
		a.record(e, "update", ptr(class), op)
	case "reaffirm":
		e.reqRevision = h.ents[e.reqID].revision
		a.record(e, "reaffirm", nil, op)
	case "retire":
		e.status = "retired"
		a.retiredHere[e.id] = true
		a.record(e, "retire", nil, op)
		a.cascade(e)
	default:
		panic("factgen: unknown op " + op.op)
	}
}

func (a *applying) addEntity(e *entity) {
	h := a.r.h
	switch e.typ {
	case "goal":
		h.keys["goal"]++
		e.key = fmt.Sprintf("GOAL-%d", h.keys["goal"])
	case "requirement":
		if e.goalID != "" && !h.Active(e.goalID) {
			panic("factgen: a requirement under a goal that is not active")
		}
		h.keys["requirement"]++
		e.reqNumber = h.keys["requirement"]
		e.key, e.title = fmt.Sprintf("REQ-%03d", e.reqNumber), fmt.Sprintf("Requirement %d", e.reqNumber)
	case "criterion":
		p := h.ents[e.reqID]
		if p == nil || p.status != "active" {
			panic("factgen: a criterion added to a requirement that is not active")
		}
		k := "criterion:" + e.reqID
		h.keys[k]++
		e.key, e.reqRevision = fmt.Sprintf("AC-%03d-%d", p.reqNumber, h.keys[k]), p.revision
	}
	e.status, e.revision, e.revisionSeq = "active", 1, a.r.seq
	h.order = append(h.order, e.id)
}

func sameCheck(x, y *baseline.Check) bool {
	if x == nil || y == nil {
		return x == y
	}
	return baseline.CheckConfigHash(*x) == baseline.CheckConfigHash(*y)
}

// record writes a change: an item's, or a follow-up's.
func (a *applying) record(e *entity, op string, class *string, p *planned) {
	h := a.r.h
	if p.followup {
		switch {
		case p.op != "reaffirm" && p.op != "retire":
			panic("factgen: a follow-up reaffirms or retires")
		case e.typ != "criterion" || !a.material[e.reqID]:
			panic("factgen: a follow-up of a criterion whose requirement this review does not revise materially")
		case a.touched[e.id] || a.changed[e.id]:
			panic("factgen: a follow-up of a criterion an item changes")
		}
	}
	if a.changed[e.id] {
		panic(fmt.Sprintf("factgen: %s changes twice in one write", e.key))
	}
	a.changed[e.id] = true
	row := ChangeRow{
		ID: h.id(), Seq: a.r.seq, VersionID: a.versionID, ChangeIndex: a.changes,
		EntityType: e.typ, EntityID: e.id, Op: op, ChangeClass: class, AfterJSON: h.snapshot(e),
	}
	switch {
	case p.followup:
		row.Rationale = ptr(p.rationale)
	case p.item != "":
		row.ProposalItemID = ptr(p.item)
		a.appliedItems[p.item] = e.id
		if op == "add" {
			a.addItems[p.item] = true
		}
		if e.typ == "criterion" {
			a.touched[e.id] = true
		}
	}
	h.d.BaselineChanges = append(h.d.BaselineChanges, row)
	a.changes++
}

// cascade writes the cascade of a retirement (data-model §5.2) right after
// it, without an item.
func (a *applying) cascade(e *entity) {
	h := a.r.h
	for _, id := range h.order {
		c := h.ents[id]
		if c.status != "active" {
			continue
		}
		switch {
		case e.typ == "requirement" && c.typ == "criterion" && c.reqID == e.id:
			c.status = "retired"
			a.retiredHere[c.id] = true
			a.write(c, "retire", nil)
		case e.typ == "goal" && c.typ == "requirement" && c.goalID == e.id:
			c.goalID = ""
			a.write(c, "update", ptr("editorial"))
		}
	}
}

func (a *applying) write(e *entity, op string, class *string) {
	h := a.r.h
	if a.changed[e.id] {
		panic(fmt.Sprintf("factgen: %s changes twice in one write", e.key))
	}
	a.changed[e.id] = true
	h.d.BaselineChanges = append(h.d.BaselineChanges, ChangeRow{
		ID: h.id(), Seq: a.r.seq, VersionID: a.versionID, ChangeIndex: a.changes,
		EntityType: e.typ, EntityID: e.id, Op: op, ChangeClass: class, AfterJSON: h.snapshot(e),
	})
	a.changes++
}
