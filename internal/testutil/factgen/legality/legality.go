// Package legality checks that a factgen dump is a legal fact history
// (data-model §6.14): one the production writes could have made. It reads
// only the dump's rows and checks them against the frozen write rules; it
// uses neither the history builder that wrote them, nor the derivation
// engine, nor the ready checker, so a history they agree on is not taken
// as legal for that reason.
//
// The rules checked:
//
//   - seq: the columns unique per table hold each seq once, and the rows
//     sharing a seq are one logical write of an allowed kind (§6.13);
//   - reviews and person's edits: one proposal per review; every explicit
//     change comes from one accepted item, reviewed in that write; a
//     proposal names a target at most once (V-P03); the changes follow the
//     order of §5.5, by step and item_index; a change without an item is a
//     cascade right after its cause, and every cascade a retirement owes
//     is written (§5.2), or a follow-up of step 8; each change is valid on
//     the entity as the earlier changes of the write left it, so an
//     entity can change more than once in a write (a cascade, then an
//     item), but by one item, one cascade and one follow-up at most; a
//     link names an entity active when its item applies; the revision
//     rules (§5.5, §7.2);
//   - questions (§4.8): one about a proposal item names an item of its
//     proposal and is linked only once the item is accepted as a
//     requirement or criterion add (a question's local about names one,
//     proposal.CheckStatic); one about an entity names an active
//     requirement or criterion;
//   - scope confirmations, claims, runs and decisions: their write
//     preconditions (§7.1, §7.3, §7.4, V-C03), with one immutable tree
//     per commit: a normalized path at a head_start has one input entry,
//     and an input_missing run there needs a declared path no record shows
//     at that commit (§4.11.2, §7.4 step 2).
//
// Not checked, because the dump does not carry them: a scope's
// acknowledged waivers and seen_scope_confirmation_id, a decision's
// acknowledged risks, the actors and channels of human-only writes, and a
// proposal item's op, type and target before it is accepted (so the type
// of a pending or rejected item a question is about). Which paths a
// commit's tree has is not in the dump either: the rows are checked for a
// contradiction among themselves, and CheckTrees checks them against a
// generator's trees.
package legality

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/source"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
)

// Problem is one broken rule, at the seq of the write that breaks it.
type Problem struct {
	Seq    int64
	Rule   string
	Detail string
}

func (p Problem) String() string { return fmt.Sprintf("seq %d: %s: %s", p.Seq, p.Rule, p.Detail) }

// Options describe the legal domain: Isolated allows synthetic isolated
// runs (the general legal domain); without it every run is observed (the
// MVP domain).
type Options struct {
	Isolated bool
}

type change struct {
	row   factgen.ChangeRow
	state state
}

// state is an entity's snapshot after a change.
type state struct {
	Revision            int    `json:"revision"`
	Status              string `json:"status"`
	RequirementRevision int    `json:"requirement_revision"`
	Values              struct {
		Goal         *string         `json:"goal"`
		Requirement  *string         `json:"requirement"`
		Verification string          `json:"verification"`
		Check        *baseline.Check `json:"check"`
	} `json:"values"`
	values map[string]any // every value, to compare them
}

func (s state) checkHash() string {
	if s.Values.Check == nil {
		return ""
	}
	return baseline.CheckConfigHash(*s.Values.Check)
}

type checker struct {
	d        factgen.Dump
	opts     Options
	problems []Problem

	kind      map[string]string // entity ID -> goal / requirement / criterion
	parent    map[string]string // criterion -> requirement
	keyOf     map[string]string
	idOfKey   map[string]string
	changes   []change // in (seq, change_index) order
	items     map[string]factgen.ProposalItemRow
	proposals map[string]int64
}

func (c *checker) fail(seq int64, rule, format string, args ...any) {
	c.problems = append(c.problems, Problem{Seq: seq, Rule: rule, Detail: fmt.Sprintf(format, args...)})
}

// Check returns every problem of a dump, in seq order; none for a legal
// history.
func Check(d factgen.Dump, o Options) []Problem {
	c := &checker{
		d: d, opts: o, kind: map[string]string{}, parent: map[string]string{}, keyOf: map[string]string{}, idOfKey: map[string]string{},
		items: map[string]factgen.ProposalItemRow{}, proposals: map[string]int64{},
	}
	for t, rows := range map[string][]factgen.EntityRow{"goal": d.Goals, "requirement": d.Requirements, "criterion": d.Criteria} {
		for _, e := range rows {
			c.kind[e.ID], c.keyOf[e.ID], c.idOfKey[e.Key] = t, e.Key, e.ID
			if t == "criterion" {
				c.parent[e.ID] = e.RequirementID
			}
		}
	}
	for _, it := range d.ProposalItems {
		c.items[it.ID] = it
	}
	for _, p := range d.Proposals {
		c.proposals[p.ID] = p.Seq
	}
	for _, r := range d.BaselineChanges {
		var s state
		var all struct {
			Values map[string]any `json:"values"`
		}
		if err := cmp.Or(json.Unmarshal([]byte(r.AfterJSON), &s), json.Unmarshal([]byte(r.AfterJSON), &all)); err != nil {
			c.fail(r.Seq, "after_json", "change %s: %v", r.ID, err)
			continue
		}
		s.values = all.Values
		c.changes = append(c.changes, change{row: r, state: s})
	}
	slices.SortFunc(c.changes, func(a, b change) int {
		return cmp.Or(cmp.Compare(a.row.Seq, b.row.Seq), cmp.Compare(a.row.ChangeIndex, b.row.ChangeIndex))
	})
	c.checkSeqs()
	c.checkReviews()
	c.checkQuestions()
	c.checkScopes()
	c.checkClaims()
	c.checkRuns()
	c.checkDecisions()
	slices.SortStableFunc(c.problems, func(a, b Problem) int { return cmp.Compare(a.Seq, b.Seq) })
	return c.problems
}

// at is the state of every entity after the changes with seq <= upTo.
func (c *checker) at(upTo int64) map[string]state {
	out := map[string]state{}
	for _, ch := range c.changes {
		if ch.row.Seq > upTo {
			break
		}
		out[ch.row.EntityID] = ch.state
	}
	return out
}

func (c *checker) version(upTo int64) int {
	v := 0
	for _, r := range c.d.BaselineVersions {
		if r.Seq <= upTo {
			v = max(v, r.Number)
		}
	}
	return v
}

func (c *checker) activeTarget(upTo int64) string {
	id := ""
	for _, t := range c.d.DeliveryTargets {
		if t.CreatedSeq <= upTo && (t.ArchivedSeq == nil || *t.ArchivedSeq > upTo) {
			if id != "" {
				c.fail(upTo, "one target", "two delivery targets are active")
			}
			id = t.ID
		}
	}
	return id
}

// setHash is the source set hash at a seq, from the acknowledgements.
func (c *checker) setHash(upTo int64) string {
	var members []source.Member
	for _, s := range c.d.Sources {
		if s.RegisteredSeq > upTo || (s.RemovedSeq != nil && *s.RemovedSeq <= upTo) {
			continue
		}
		hash, first := s.AckedSHA256, true
		for _, a := range c.sortedAcks(s.ID) {
			if first {
				hash, first = a.PreviousSHA256, false
			}
			if a.Seq <= upTo {
				hash = a.AckedSHA256
			}
		}
		members = append(members, source.Member{ID: s.ID, AckedSHA256: hash})
	}
	return source.SetHash(members)
}

func (c *checker) sortedAcks(id string) []factgen.SourceAckRow {
	var out []factgen.SourceAckRow
	for _, a := range c.d.SourceAcks {
		if a.SourceID == id {
			out = append(out, a)
		}
	}
	slices.SortFunc(out, func(a, b factgen.SourceAckRow) int { return cmp.Compare(a.Seq, b.Seq) })
	return out
}

// checkSeqs checks that the columns unique per table are, and that the
// rows sharing a seq form one logical write of a kind §6.13 allows.
func (c *checker) checkSeqs() {
	kinds := map[int64]map[string]int{}
	add := func(seq int64, k string) {
		if kinds[seq] == nil {
			kinds[seq] = map[string]int{}
		}
		kinds[seq][k]++
	}
	for _, t := range c.d.DeliveryTargets {
		add(t.CreatedSeq, "target")
		if t.ArchivedSeq != nil {
			add(*t.ArchivedSeq, "archive")
		}
	}
	for _, s := range c.d.Sources {
		add(s.RegisteredSeq, "source")
		if s.RemovedSeq != nil {
			add(*s.RemovedSeq, "remove")
		}
	}
	for _, a := range c.d.SourceAcks {
		add(a.Seq, "ack")
	}
	for _, p := range c.d.Proposals {
		add(p.Seq, "proposal")
	}
	for _, v := range c.d.BaselineVersions {
		add(v.Seq, "version")
	}
	for _, ch := range c.changes {
		add(ch.row.Seq, "change")
	}
	for _, r := range c.d.SourceRefs {
		add(r.Seq, "ref")
	}
	for _, s := range c.d.ScopeConfirmations {
		add(s.Seq, "scope")
	}
	for _, it := range c.d.ProposalItems {
		if it.ReviewedSeq != nil {
			add(*it.ReviewedSeq, "reviewed")
		}
	}
	for _, q := range c.d.Questions {
		add(q.CreatedSeq, "question")
		if q.AnsweredSeq != nil {
			add(*q.AnsweredSeq, "answer")
		}
	}
	for _, b := range c.d.ClaimBatches {
		add(b.Seq, "batch")
	}
	for _, r := range c.d.CheckRuns {
		add(r.Seq, "run")
	}
	for _, x := range c.d.Decisions {
		add(x.Seq, "decision")
	}
	for _, x := range c.d.Commits {
		add(x.Seq, "commit")
	}
	// The kinds of write and the rows each one may share its seq with;
	// "unique" are the kinds with a column unique per table.
	writes := [][]string{
		{"target", "archive"}, {"source"}, {"ack"}, {"remove"},
		{"proposal", "question"},
		{"version", "change", "ref", "reviewed", "scope"},
		{"proposal", "version", "change", "ref", "reviewed", "scope"},
		{"reviewed", "scope"}, {"reviewed"}, {"scope"},
		{"answer"}, {"batch"}, {"run"}, {"decision"}, {"commit"},
	}
	unique := []string{"target", "archive", "source", "ack", "remove", "proposal", "version", "scope", "batch", "run", "decision"}
	seqs := make([]int64, 0, len(kinds))
	for s := range kinds {
		seqs = append(seqs, s)
	}
	slices.Sort(seqs)
	for _, seq := range seqs {
		got := kinds[seq]
		if seq < 1 || seq > c.d.LatestSeq {
			c.fail(seq, "seq", "outside 1..%d", c.d.LatestSeq)
		}
		for _, k := range unique {
			if got[k] > 1 {
				c.fail(seq, "unique seq", "%d rows of %s", got[k], k)
			}
		}
		ok := slices.ContainsFunc(writes, func(w []string) bool {
			for k := range got {
				if !slices.Contains(w, k) {
					return false
				}
			}
			return true
		})
		if !ok {
			names := make([]string, 0, len(got))
			for k := range got {
				names = append(names, k)
			}
			slices.Sort(names)
			c.fail(seq, "one logical write", "rows of %v share the seq", names)
		}
		if got["change"] > 0 && got["version"] != 1 {
			c.fail(seq, "version", "changes without one version")
		}
		if got["ref"] > 0 && got["change"] == 0 {
			c.fail(seq, "references", "references written without a change")
		}
	}
}

func step(kind, op string) int {
	switch {
	case kind == "goal":
		return 0
	case kind == "requirement":
		return 1
	case op == "add":
		return 2
	case op == "update":
		return 3
	case op == "reaffirm":
		return 4
	}
	return 5 // a criterion's retire
}

func forcedMaterial(prev, next state) bool {
	if prev.Values.Verification != next.Values.Verification {
		return true
	}
	return prev.checkHash() != next.checkHash()
}

// checkReviews checks every review and person's edit: the version, the
// proposal and its items, V-P03, the order of §5.5, cascades and
// follow-ups, and the revision rules.
func (c *checker) checkReviews() {
	targeted := map[string]map[string]bool{} // proposal -> explicit targets
	usedItems := map[string]bool{}
	states := map[string]state{}
	lastVersion := 0
	versions := slices.Clone(c.d.BaselineVersions)
	slices.SortFunc(versions, func(a, b factgen.VersionRow) int { return cmp.Compare(a.Seq, b.Seq) })
	for _, v := range versions {
		if v.Number != lastVersion+1 {
			c.fail(v.Seq, "version", "number %d after %d", v.Number, lastVersion)
		}
		lastVersion = v.Number
	}
	for i := 0; i < len(c.changes); {
		seq := c.changes[i].row.Seq
		j := i
		for j < len(c.changes) && c.changes[j].row.Seq == seq {
			j++
		}
		c.checkReview(seq, c.changes[i:j], states, targeted, usedItems)
		i = j
	}
	// Items reviewed in a write without a version reject everything.
	for _, it := range c.d.ProposalItems {
		if it.ReviewedSeq != nil && it.ReviewState != "rejected" && !usedItems[it.ID] {
			c.fail(*it.ReviewedSeq, "item", "item %s is %s but applies no change", it.ID, it.ReviewState)
		}
		if it.ReviewedSeq == nil && it.ReviewState != "pending" {
			c.fail(c.proposals[it.ProposalID], "item", "item %s is %s without a review", it.ID, it.ReviewState)
		}
	}
	c.checkOneProposalPerReview()
}

func (c *checker) checkOneProposalPerReview() {
	byReview := map[int64]map[string]bool{}
	for _, it := range c.d.ProposalItems {
		if it.ReviewedSeq == nil {
			continue
		}
		if byReview[*it.ReviewedSeq] == nil {
			byReview[*it.ReviewedSeq] = map[string]bool{}
		}
		byReview[*it.ReviewedSeq][it.ProposalID] = true
		if p, ok := c.proposals[it.ProposalID]; !ok || p > *it.ReviewedSeq {
			c.fail(*it.ReviewedSeq, "item", "item %s is reviewed before its proposal", it.ID)
		}
	}
	for seq, ps := range byReview {
		if len(ps) > 1 {
			c.fail(seq, "one proposal per review", "%d proposals reviewed in one write", len(ps))
		}
	}
}

func (c *checker) checkReview(seq int64, cs []change, states map[string]state, targeted map[string]map[string]bool, usedItems map[string]bool) {
	step0, lastIndex := -1, -1
	followups := false
	changed := map[string]bool{}
	origins := map[string]map[string]bool{} // entity -> an item / a cascade / a follow-up
	materialReq := map[string]bool{}
	touched := map[string]bool{} // criteria an item of this review changes
	var cause *change            // the last item change, for its cascades
	owed := map[string]bool{}    // the entities the cause's cascade has still to change
	// settle ends the cause's cascade, which must change every entity it owes.
	settle := func() {
		for _, id := range slices.Sorted(maps.Keys(owed)) {
			c.fail(seq, "cascade", "%s is retired, but its cascade does not change %s (§5.2)", c.keyOf[cause.row.EntityID], c.keyOf[id])
		}
		clear(owed)
	}
	for k := range cs {
		ch := &cs[k]
		r, s := ch.row, ch.state
		kind := c.kind[r.EntityID]
		if kind == "" {
			c.fail(seq, "entity", "change %s of an unknown entity", r.ID)
			continue
		}
		if r.ChangeIndex != k {
			c.fail(seq, "change_index", "change %s has index %d, at position %d", r.ID, r.ChangeIndex, k)
		}
		changed[r.EntityID] = true
		// Each change applies to the entity as the earlier changes of this
		// write left it: a cascade, then an item, is two valid changes.
		prev, existed := states[r.EntityID]
		c.checkRevision(seq, r, kind, prev, existed, s, states)
		origin := "an item"
		if r.ProposalItemID != nil {
			settle()
			item, ok := c.items[*r.ProposalItemID]
			switch {
			case !ok:
				c.fail(seq, "item", "change %s names an unknown item", r.ID)
			case usedItems[item.ID]:
				c.fail(seq, "item", "item %s applies two changes", item.ID)
			case item.ReviewedSeq == nil || *item.ReviewedSeq != seq || (item.ReviewState != "accepted" && item.ReviewState != "accepted_with_edits"):
				c.fail(seq, "item", "item %s is not accepted in this write", item.ID)
			}
			usedItems[item.ID] = true
			c.checkNames(seq, r, prev, s, kind, item, cs, states)
			if targeted[item.ProposalID] == nil {
				targeted[item.ProposalID] = map[string]bool{}
			}
			if targeted[item.ProposalID][r.EntityID] {
				c.fail(seq, "V-P03", "proposal %s names %s twice", item.ProposalID, c.keyOf[r.EntityID])
			}
			targeted[item.ProposalID][r.EntityID] = true
			if followups {
				c.fail(seq, "§5.5 order", "item change %s after a follow-up", r.ID)
			}
			switch st := step(kind, r.Op); {
			case st < step0:
				c.fail(seq, "§5.5 order", "%s %s (step %d) after step %d", kind, r.Op, st, step0)
			case st == step0 && item.ItemIndex <= lastIndex:
				c.fail(seq, "§5.5 order", "item_index %d after %d within step %d", item.ItemIndex, lastIndex, st)
			default:
				step0, lastIndex = st, item.ItemIndex
			}
			if kind == "requirement" && r.Op == "update" && existed && s.Revision == prev.Revision+1 {
				materialReq[r.EntityID] = true
			}
			if kind == "criterion" {
				touched[r.EntityID] = true
			}
			cause = ch
			if r.Op == "retire" {
				owed = c.cascadeOf(r.EntityID, kind, states)
			}
		} else {
			switch {
			case kind == "criterion" && r.Op == "retire" && owed[r.EntityID]:
				// The cascade of its requirement's retirement.
				origin = "a cascade"
			case kind == "requirement" && r.Op == "update" && owed[r.EntityID] && r.ChangeClass != nil && *r.ChangeClass == "editorial" &&
				s.Values.Goal == nil && s.Revision == prev.Revision && sameValues(prev.values, s.values, "goal"):
				// The cascade of its goal's retirement: it loses the goal.
				origin = "a cascade"
			case kind == "criterion" && (r.Op == "reaffirm" || r.Op == "retire") && r.Rationale != nil &&
				materialReq[c.parent[r.EntityID]] && !touched[r.EntityID]:
				settle()
				origin, followups = "a follow-up", true // step 8
			default:
				c.fail(seq, "explicit change", "%s %s of %s without a proposal item is no cascade or follow-up", kind, r.Op, c.keyOf[r.EntityID])
				origin = ""
			}
			delete(owed, r.EntityID)
		}
		// An entity changes by one item at most (V-P03, one proposal per
		// review), by one cascade and by one follow-up.
		if origins[r.EntityID] == nil {
			origins[r.EntityID] = map[string]bool{}
		}
		if origin != "" && origins[r.EntityID][origin] {
			c.fail(seq, "changes of an entity", "%s changes twice by %s", c.keyOf[r.EntityID], origin)
		}
		origins[r.EntityID][origin] = true
		states[r.EntityID] = s
	}
	settle()
	// References are written for the entities this write changes, at their
	// new revision (§4.5).
	for _, ref := range c.d.SourceRefs {
		if ref.Seq == seq && ref.EntityType != "question" && (!changed[ref.EntityID] || states[ref.EntityID].Revision != ref.EntityRevision) {
			c.fail(seq, "references", "a reference of %s revision %d it does not write", c.keyOf[ref.EntityID], ref.EntityRevision)
		}
	}
	// One proposal; a person's edit has its proposal in the same write.
	for _, ch := range cs {
		if ch.row.ProposalItemID == nil {
			continue
		}
		if p := c.items[*ch.row.ProposalItemID].ProposalID; c.proposals[p] == seq {
			for _, it := range c.d.ProposalItems {
				if it.ProposalID == p && (it.ReviewedSeq == nil || *it.ReviewedSeq != seq || it.ReviewState == "rejected") {
					c.fail(seq, "person's edit", "item %s of the edit is not applied", it.ID)
				}
			}
			if followups {
				c.fail(seq, "person's edit", "an edit has no follow-ups")
			}
		}
		break
	}
}

// cascadeOf are the entities the cascade of a retirement changes (§5.2,
// baseline.Apply): a goal's active requirements lose it, a requirement's
// active criteria are retired with it.
func (c *checker) cascadeOf(id, kind string, states map[string]state) map[string]bool {
	out := map[string]bool{}
	for e, st := range states {
		switch {
		case st.Status != "active":
		case kind == "goal" && c.kind[e] == "requirement" && st.Values.Goal != nil && c.idOfKey[*st.Values.Goal] == id:
			out[e] = true
		case kind == "requirement" && c.kind[e] == "criterion" && c.parent[e] == id:
			out[e] = true
		}
	}
	return out
}

// sameValues reports whether two entity values are equal but for the
// field except ("" for none).
func sameValues(a, b map[string]any, except string) bool {
	a, b = maps.Clone(a), maps.Clone(b)
	delete(a, except)
	delete(b, except)
	return reflect.DeepEqual(a, b)
}

func sameKey(a, b *string) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }

// checkNames checks what an item names (V-P03): a target is a key active
// when the proposal was submitted. A link (a criterion add's requirement,
// the goal a requirement add or update sets) is such a key, or an entity an
// add item of the same proposal adds in the same write; it must be active
// when the item applies, so not retired by an earlier step of the write
// (baseline.Apply's link). An update does not move a criterion to another
// requirement.
func (c *checker) checkNames(seq int64, r factgen.ChangeRow, prev, s state, kind string, item factgen.ProposalItemRow, cs []change, states map[string]state) {
	submitted := c.at(c.proposals[item.ProposalID] - 1)
	activeThen := func(id string) bool { return submitted[id].Status == "active" }
	addedHere := func(id string) bool {
		return slices.ContainsFunc(cs, func(o change) bool {
			return o.row.EntityID == id && o.row.Op == "add" && o.row.ProposalItemID != nil &&
				c.items[*o.row.ProposalItemID].ProposalID == item.ProposalID
		})
	}
	if r.Op != "add" && !activeThen(r.EntityID) {
		c.fail(seq, "V-P03 target", "item %s names %s, which was not an active key when the proposal was submitted", item.ID, c.keyOf[r.EntityID])
	}
	var link, linkKind string
	switch {
	case kind == "criterion" && r.Op == "add":
		link, linkKind = c.parent[r.EntityID], "requirement"
	case kind == "criterion" && r.Op == "update" && !sameKey(prev.Values.Requirement, s.Values.Requirement):
		c.fail(seq, "link", "item %s moves %s to another requirement", item.ID, c.keyOf[r.EntityID])
	case kind == "requirement" && s.Values.Goal != nil && (r.Op == "add" || (r.Op == "update" && !sameKey(prev.Values.Goal, s.Values.Goal))):
		link, linkKind = c.idOfKey[*s.Values.Goal], "goal"
	}
	switch {
	case link == "":
	case c.kind[link] != linkKind:
		c.fail(seq, "link", "item %s links %s to no %s", item.ID, c.keyOf[r.EntityID], linkKind)
	case states[link].Status != "active":
		c.fail(seq, "link", "item %s links %s to %s, which is not active when the item applies", item.ID, c.keyOf[r.EntityID], c.keyOf[link])
	case !activeThen(link) && !addedHere(link):
		c.fail(seq, "V-P03 target", "item %s links to %s, which is neither an active key of the submission nor added by the proposal here", item.ID, c.keyOf[link])
	}
}

// checkRevision checks one change against the entity's previous state
// (§5.5, §7.2).
func (c *checker) checkRevision(seq int64, r factgen.ChangeRow, kind string, prev state, existed bool, s state, states map[string]state) {
	key := c.keyOf[r.EntityID]
	parentRevision := func() int { return states[c.parent[r.EntityID]].Revision }
	if r.Op != "add" && (!existed || prev.Status != "active") {
		c.fail(seq, "target", "%s %s of an entity that is not active", r.Op, key)
		return
	}
	switch r.Op {
	case "add":
		if existed || s.Revision != 1 || s.Status != "active" {
			c.fail(seq, "add", "%s is added twice or not at revision 1", key)
		}
		if kind == "criterion" && s.RequirementRevision != parentRevision() {
			c.fail(seq, "add", "%s applies to revision %d of its requirement, at %d", key, s.RequirementRevision, parentRevision())
		}
	case "update":
		switch {
		case s.Status != "active" || (s.Revision != prev.Revision && s.Revision != prev.Revision+1):
			c.fail(seq, "update", "%s goes from revision %d to %d", key, prev.Revision, s.Revision)
		case kind == "criterion" && forcedMaterial(prev, s) && s.Revision == prev.Revision:
			c.fail(seq, "§7.2", "%s changes its check or verification editorially", key)
		}
		if kind == "criterion" && s.RequirementRevision != parentRevision() {
			c.fail(seq, "update", "%s is updated without applying to revision %d of its requirement", key, parentRevision())
		}
	case "reaffirm":
		if kind != "criterion" || s.Revision != prev.Revision || s.RequirementRevision != parentRevision() || !sameValues(prev.values, s.values, "") {
			c.fail(seq, "reaffirm", "%s is not reaffirmed for its requirement's revision, its values unchanged", key)
		}
	case "retire":
		if s.Status != "retired" || s.Revision != prev.Revision || !sameValues(prev.values, s.values, "") {
			c.fail(seq, "retire", "%s is not retired at its revision, its values unchanged", key)
		}
	default:
		c.fail(seq, "op", "unknown op %s", r.Op)
	}
}

// checkQuestions checks §4.8. A question about a proposal item names an
// item of its proposal, a requirement or criterion add: a question's
// proposal-local about names one of those (proposal.CheckStatic), and only
// an add item has a ref. It is linked to the entity only once the item is
// accepted, if it is open then. A question about an entity names an active
// requirement or criterion (proposal's store). The type of an item that is
// not accepted is not in the dump.
func (c *checker) checkQuestions() {
	produced := map[string]change{}
	for _, ch := range c.changes {
		if ch.row.ProposalItemID != nil {
			produced[*ch.row.ProposalItemID] = ch
		}
	}
	for _, q := range c.d.Questions {
		if q.ProposalItemID == nil {
			if q.RelatedID == nil {
				continue
			}
			kind := c.kind[*q.RelatedID]
			if (kind != "requirement" && kind != "criterion") || orEmpty(q.RelatedType) != kind || c.at(q.CreatedSeq)[*q.RelatedID].Status != "active" {
				c.fail(q.CreatedSeq, "question", "%s is about a %s that is not an active requirement or criterion", q.Key, kind)
			}
			continue
		}
		item, ok := c.items[*q.ProposalItemID]
		if !ok || c.proposals[item.ProposalID] != q.CreatedSeq {
			c.fail(q.CreatedSeq, "question", "%s is about an item of another proposal", q.Key)
			continue
		}
		ch, accepted := produced[item.ID]
		switch {
		case accepted && (ch.row.Op != "add" || (c.kind[ch.row.EntityID] != "requirement" && c.kind[ch.row.EntityID] != "criterion")):
			c.fail(q.CreatedSeq, "question", "%s is about item %s, accepted as a %s %s: a question's local about names a requirement or criterion add",
				q.Key, item.ID, c.kind[ch.row.EntityID], ch.row.Op)
		case accepted:
			openThen := q.AnsweredSeq == nil || *q.AnsweredSeq > ch.row.Seq
			linked := q.RelatedID != nil && *q.RelatedID == ch.row.EntityID
			if openThen != linked || (q.RelatedID != nil && !linked) {
				c.fail(ch.row.Seq, "question", "%s's link after its item's acceptance", q.Key)
			}
		case q.RelatedID != nil:
			c.fail(q.CreatedSeq, "question", "%s is linked although its item is not accepted", q.Key)
		}
	}
}

// checkScopes checks the scope confirmations: the active target, the
// version and source set hash at the write, active requirements at their
// revision.
func (c *checker) checkScopes() {
	for _, s := range c.d.ScopeConfirmations {
		if t := c.activeTarget(s.Seq); t == "" || t != s.TargetID {
			c.fail(s.Seq, "scope", "confirmed for a target that is not active")
		}
		if v := c.version(s.Seq); s.BaselineVersion != v {
			c.fail(s.Seq, "scope", "binds to version %d, the baseline is at %d", s.BaselineVersion, v)
		}
		if h := c.setHash(s.Seq); s.SourceSetHash != h {
			c.fail(s.Seq, "scope", "binds to another source set")
		}
		states := c.at(s.Seq)
		n := 0
		for _, it := range c.d.ScopeItems {
			if it.ConfirmationID != s.ID {
				continue
			}
			n++
			st := states[it.RequirementID]
			if c.kind[it.RequirementID] != "requirement" || st.Status != "active" || st.Revision != it.RequirementRevision {
				c.fail(s.Seq, "scope", "%s is not an active requirement at revision %d", c.keyOf[it.RequirementID], it.RequirementRevision)
			}
		}
		if n == 0 {
			c.fail(s.Seq, "scope", "no requirement")
		}
	}
}

// checkClaims checks V-C03: an active criterion at its current revision,
// once per batch, unknown only in an audit.
func (c *checker) checkClaims() {
	kinds := map[string]factgen.ClaimBatchRow{}
	for _, b := range c.d.ClaimBatches {
		kinds[b.ID] = b
		if b.Kind == "audit" && b.AuditScopeJSON == nil {
			c.fail(b.Seq, "V-C05", "an audit batch without a scope")
		}
	}
	seen := map[string]bool{}
	for _, x := range c.d.Claims {
		b, ok := kinds[x.BatchID]
		if !ok || b.Seq != x.Seq {
			c.fail(x.Seq, "claim", "claim %s is not written with its batch", x.ID)
		}
		if x.Assertion == "unknown" && b.Kind != "audit" {
			c.fail(x.Seq, "V-C03", "unknown outside an audit")
		}
		if x.TargetType != "criterion" {
			continue
		}
		st := c.at(x.Seq)[*x.CriterionID]
		if c.kind[*x.CriterionID] != "criterion" || st.Status != "active" || st.Revision != *x.CriterionRevision {
			c.fail(x.Seq, "V-C03", "a claim on %s that is not active at revision %d", c.keyOf[*x.CriterionID], *x.CriterionRevision)
		}
		k := x.BatchID + "/" + *x.CriterionID
		if seen[k] {
			c.fail(x.Seq, "V-C03", "%s twice in one batch", c.keyOf[*x.CriterionID])
		}
		seen[k] = true
	}
}

func orEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func qualifying(r factgen.CheckRunRow) bool {
	return r.HeadStart != nil && r.HeadEnd != nil && *r.HeadStart == *r.HeadEnd &&
		r.TreeStateStart == "clean" && r.TreeStateEnd == "clean" && orEmpty(r.FailureKind) != "context_changed"
}

// lookupPath is the path an input is looked up by in a commit's tree
// (data-model §4.11.2, as V-P04 normalizes it): \ as /, then path.Clean.
func lookupPath(p string) string { return path.Clean(strings.ReplaceAll(p, `\`, "/")) }

type treeKey struct{ commit, path string }

// entries are the input entries the runs record, by commit and lookup
// path; a commit's tree is immutable, so two records of one path at one
// commit that differ are a problem whatever their seqs.
func (c *checker) entries() map[treeKey]factgen.InputEntry {
	runs := slices.Clone(c.d.CheckRuns)
	slices.SortFunc(runs, func(a, b factgen.CheckRunRow) int { return cmp.Compare(a.Seq, b.Seq) })
	out := map[treeKey]factgen.InputEntry{}
	for _, r := range runs {
		if r.HeadStart == nil || r.InputsJSON == nil {
			continue
		}
		var entries []factgen.InputEntry
		_ = json.Unmarshal([]byte(*r.InputsJSON), &entries)
		for _, e := range entries {
			k := treeKey{*r.HeadStart, lookupPath(e.Path)}
			if was, ok := out[k]; ok && (was.Mode != e.Mode || was.Type != e.Type || was.OID != e.OID) {
				c.fail(r.Seq, "commit tree", "%s at commit %s is recorded as %s %s %s, and before as %s %s %s", k.path, short(k.commit),
					e.Mode, e.Type, e.OID, was.Mode, was.Type, was.OID)
				continue
			}
			out[k] = e
		}
	}
	return out
}

func short(s string) string { return s[:min(8, len(s))] }

// checkRuns checks §7.4: the run binds the criterion's revision and
// configuration of its start, records its declared inputs, and a context
// change is an error; the grade is the domain's. An input_missing run at a
// head_start needs a declared input no record shows at that commit: the
// tree of head_start, not the worktree, is read (§4.11.2), so a later
// context change or a dirty worktree does not change it.
func (c *checker) checkRuns() {
	recorded := c.entries()
	for _, r := range c.d.CheckRuns {
		// The run binds the revision, configuration and requirement
		// revision it read at its start, which can be before a revision
		// written while it ran (CX-15): one of the criterion's states
		// before the run's write, active with a check.
		var st *state
		for i := range c.changes {
			ch := &c.changes[i]
			if ch.row.Seq < r.Seq && ch.row.EntityID == r.CriterionID && ch.state.Status == "active" && ch.state.Values.Check != nil &&
				ch.state.Revision == r.CriterionRevision && ch.state.checkHash() == r.CheckConfigHash &&
				ch.state.RequirementRevision == r.RequirementRevision {
				st = &ch.state
			}
		}
		if c.kind[r.CriterionID] != "criterion" || st == nil {
			c.fail(r.Seq, "run", "a run of %s that binds no revision and configuration it had", c.keyOf[r.CriterionID])
			continue
		}
		kind := orEmpty(r.FailureKind)
		// §7.4 step 4: clean at the start and not at the end, or another
		// HEAD, is error / context_changed.
		moved := (r.TreeStateStart == "clean" && r.TreeStateEnd != "clean") || orEmpty(r.HeadStart) != orEmpty(r.HeadEnd)
		if (kind == "context_changed") != moved {
			c.fail(r.Seq, "§7.4", "a run whose context changed = %v has failure kind %q", moved, kind)
		}
		if (r.Result == "pass") != (kind == "") {
			c.fail(r.Seq, "run", "result %s with failure kind %q", r.Result, kind)
		}
		declared := slices.Clone(st.Values.Check.Inputs)
		slices.Sort(declared)
		if kind == "input_missing" && r.HeadStart != nil && len(declared) > 0 && !slices.ContainsFunc(declared, func(p string) bool {
			_, ok := recorded[treeKey{*r.HeadStart, lookupPath(p)}]
			return !ok
		}) {
			c.fail(r.Seq, "commit tree", "input_missing at commit %s, where a record shows every declared input %v", short(*r.HeadStart), declared)
		}
		switch {
		case kind == "input_missing" || len(declared) == 0:
			if r.InputsJSON != nil {
				c.fail(r.Seq, "§4.11.2", "an input record without declared inputs, or for a missing input")
			}
		case r.InputsJSON == nil:
			c.fail(r.Seq, "§4.11.2", "no input record for the declared inputs")
		default:
			var entries []factgen.InputEntry
			_ = json.Unmarshal([]byte(*r.InputsJSON), &entries)
			var paths []string
			for _, e := range entries {
				paths = append(paths, e.Path)
			}
			if !slices.Equal(paths, declared) {
				c.fail(r.Seq, "§4.11.2", "the input record has %v, the check declares %v", paths, declared)
			}
		}
		switch r.EvidenceGrade {
		case "observed":
		case "snapshot_isolated":
			if !c.opts.Isolated || !qualifying(r) {
				c.fail(r.Seq, "grade", "an isolated run outside the general domain, or not qualifying")
			}
		default:
			c.fail(r.Seq, "grade", "unknown grade %s", r.EvidenceGrade)
		}
	}
}

// checkDecisions checks §7.1 and §7.3 for every decision, with the state
// at its write and the snapshot it records as S.
func (c *checker) checkDecisions() {
	latest := map[string]string{} // criterion@revision -> latest decision
	decisions := slices.Clone(c.d.Decisions)
	slices.SortFunc(decisions, func(a, b factgen.DecisionRow) int { return cmp.Compare(a.Seq, b.Seq) })
	for _, x := range decisions {
		states := c.at(x.Seq)
		st := states[x.CriterionID]
		parent := states[c.parent[x.CriterionID]]
		k := fmt.Sprintf("%s@%d", x.CriterionID, x.CriterionRevision)
		if orEmpty(x.SeenDecisionID) != latest[k] {
			c.fail(x.Seq, "§7.1", "seen_decision_id %q, the latest is %q", orEmpty(x.SeenDecisionID), latest[k])
		}
		latest[k] = x.ID
		if t := c.activeTarget(x.Seq); t == "" || x.TargetID != t {
			c.fail(x.Seq, "§7.1", "a decision for a target that is not active")
		}
		if c.kind[x.CriterionID] != "criterion" || st.Status != "active" || st.Revision != x.CriterionRevision || parent.Revision != x.RequirementRevision {
			c.fail(x.Seq, "§7.1", "a decision on %s that does not bind its current revisions", c.keyOf[x.CriterionID])
			continue
		}
		parentOK := st.RequirementRevision == parent.Revision
		snapshot, mode := orEmpty(x.Snapshot), orEmpty(x.Mode)
		needsReason := x.Decision != "accepted" || mode == "exception"
		if needsReason && orEmpty(x.Rationale) == "" {
			c.fail(x.Seq, "§7.3", "a %s %s without a rationale", x.Decision, mode)
		}
		// runs(S) and V at the decision, from the runs written before it.
		var runs []factgen.CheckRunRow
		for _, r := range c.d.CheckRuns {
			if r.CriterionID == x.CriterionID && r.Seq < x.Seq && qualifying(r) && orEmpty(r.HeadStart) == snapshot && snapshot != "" &&
				r.CriterionRevision == st.Revision && r.CheckConfigHash == st.checkHash() {
				runs = append(runs, r)
			}
		}
		slices.SortFunc(runs, func(a, b factgen.CheckRunRow) int { return cmp.Compare(a.Seq, b.Seq) })
		v := ""
		if n := len(runs); n > 0 {
			v = map[bool]string{true: "pass", false: "fail"}[runs[n-1].Result == "pass"]
		}
		automated := st.Values.Verification == "automated"
		switch x.Decision {
		case "waived":
			if !parentOK {
				c.fail(x.Seq, "§7.3", "a waiver while the parent's applicability is not confirmed")
			}
		case "rejected":
			if snapshot == "" {
				c.fail(x.Seq, "§7.3", "a rejection without a snapshot")
			}
		case "accepted":
			switch {
			case snapshot == "":
				c.fail(x.Seq, "§7.3", "an acceptance without a snapshot")
			case !parentOK:
				c.fail(x.Seq, "§7.3", "an acceptance while the parent's applicability is not confirmed")
			case v == "fail":
				c.fail(x.Seq, "§7.3", "an acceptance while V is a failure")
			case mode == "normal" && (!automated || v != "pass" || orEmpty(x.EvidenceID) != runs[len(runs)-1].ID ||
				orEmpty(x.CheckConfigHash) != st.checkHash() || orEmpty(x.InputsJSON) != orEmpty(runs[len(runs)-1].InputsJSON) ||
				orEmpty(x.EvidenceGrade) != runs[len(runs)-1].EvidenceGrade):
				c.fail(x.Seq, "§7.3", "a normal acceptance that does not rely on the latest qualifying pass on its snapshot")
			case mode == "manual" && automated:
				c.fail(x.Seq, "§7.3", "a manual acceptance of an automated criterion")
			case mode == "exception" && (!automated || v != ""):
				c.fail(x.Seq, "§7.3", "an exception acceptance with a qualifying run on its snapshot, or of a manual criterion")
			case mode != "normal" && mode != "manual" && mode != "exception":
				c.fail(x.Seq, "§7.3", "an acceptance without a mode")
			}
		default:
			c.fail(x.Seq, "decision", "unknown decision %s", x.Decision)
		}
		if x.EvidenceID != nil && (len(runs) == 0 || runs[len(runs)-1].ID != *x.EvidenceID) {
			c.fail(x.Seq, "§7.1", "the evidence %s is not the latest qualifying run on the snapshot", *x.EvidenceID)
		}
	}
}

// Report formats the problems with the history's trace around their seqs.
func Report(problems []Problem, trace []string) string {
	var b strings.Builder
	for _, p := range problems {
		b.WriteString(p.String() + "\n")
	}
	if len(trace) > 0 {
		b.WriteString("trace:\n")
		for _, t := range trace {
			b.WriteString("  " + t + "\n")
		}
	}
	return b.String()
}

// CheckTrees checks the input records of a dump against the trees of its
// commits, a generator's witness (factgen.Case.Trees): every entry is the
// one of its lookup path in the tree of the run's head_start, and an
// input_missing run declares a path that tree does not have. It shows the
// records are explained by one tree per commit as the generator declares
// them; it says nothing of how a Git reader computes an entry.
func CheckTrees(d factgen.Dump, trees factgen.Trees) []Problem {
	c := &checker{d: d}
	declared := map[string][]string{} // criterion/config hash -> inputs
	for _, ch := range d.BaselineChanges {
		var s state
		if json.Unmarshal([]byte(ch.AfterJSON), &s) == nil && s.Values.Check != nil {
			declared[ch.EntityID+"/"+s.checkHash()] = s.Values.Check.Inputs
		}
	}
	for _, r := range d.CheckRuns {
		inputs := declared[r.CriterionID+"/"+r.CheckConfigHash]
		if r.HeadStart == nil || len(inputs) == 0 {
			continue
		}
		tree, ok := trees[*r.HeadStart]
		if !ok {
			c.fail(r.Seq, "commit tree", "no tree of commit %s", short(*r.HeadStart))
			continue
		}
		has := func(p string) bool { _, ok := tree[lookupPath(p)]; return ok }
		switch {
		case orEmpty(r.FailureKind) == "input_missing":
			if !slices.ContainsFunc(inputs, func(p string) bool { return !has(p) }) {
				c.fail(r.Seq, "commit tree", "input_missing at %s, whose tree has every declared input %v", short(*r.HeadStart), inputs)
			}
		case r.InputsJSON != nil:
			var entries []factgen.InputEntry
			_ = json.Unmarshal([]byte(*r.InputsJSON), &entries)
			for _, e := range entries {
				if want, ok := tree[lookupPath(e.Path)]; !ok || want.Mode != e.Mode || want.Type != e.Type || want.OID != e.OID {
					c.fail(r.Seq, "commit tree", "the entry of %s at %s is not the tree's", e.Path, short(*r.HeadStart))
				}
			}
		}
	}
	return c.problems
}
