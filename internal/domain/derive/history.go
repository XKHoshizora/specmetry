package derive

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"slices"
	"strings"
)

// state is the project at a cutoff, rebuilt from the facts with seq <=
// asOf (data-model §6.13). Facts that share a seq are kept or dropped
// together; within one seq, only change_index and item_index order rows.
// Ties the facts cannot have (a column unique per table holding one seq
// twice) are broken by ID, so even malformed input gives one result.
type state struct {
	asOf    int64
	target  *Target
	version int

	entities map[string]*entityAt // that exist at the cutoff
	byKey    map[string]*entityAt

	scope      *ScopeConfirmation
	scopeItems []ScopeItem

	sources     []sourceAt      // the source set, in path order
	sourceByID  map[string]bool // members of the source set
	removed     map[string]bool // sources removed at the cutoff
	setHash     string
	refs        []SourceRef // seq <= asOf
	hasProposal bool
	unreviewed  int

	questions []questionAt // created at the cutoff, open or not

	claims     map[string][]claimAt // criterion ID -> claims, by seq
	runs       map[string][]*CheckRun
	runByID    map[string]*CheckRun
	commitSeqs map[string][]int64 // criterion ID -> discovery seqs of linked commits
	decisions  map[string][]*Decision
}

// entityAt is a baseline entity at the cutoff.
type entityAt struct {
	Entity
	EntityState
	// revisionSeq is the seq of the change where the current revision first
	// appeared (data-model §6.13): an editorial change does not move it.
	revisionSeq int64
}

func (e *entityAt) active() bool { return e.Status == "active" }

type sourceAt struct {
	Source
	acked string // acknowledged at the cutoff
}

type questionAt struct {
	Question
	open         bool
	itemRejected bool // about a proposal item rejected at the cutoff
	itemPending  bool // about a proposal item not reviewed at the cutoff
	linked       *entityAt
}

type claimAt struct {
	Claim
	kind string // of its batch
}

func bySeqID[T any](seq func(T) int64, id func(T) string) func(a, b T) int {
	return func(a, b T) int { return cmp.Or(cmp.Compare(seq(a), seq(b)), strings.Compare(id(a), id(b))) }
}

// at rebuilds the state at a cutoff.
func at(f *Facts, asOf int64) *state {
	st := &state{
		asOf: asOf, entities: map[string]*entityAt{}, byKey: map[string]*entityAt{},
		sourceByID: map[string]bool{}, removed: map[string]bool{},
		claims: map[string][]claimAt{}, runs: map[string][]*CheckRun{}, runByID: map[string]*CheckRun{},
		commitSeqs: map[string][]int64{}, decisions: map[string][]*Decision{},
	}
	st.target = activeTarget(f.Targets, asOf)
	for _, v := range f.BaselineVersions {
		if v.Seq <= asOf && v.Number > st.version {
			st.version = v.Number
		}
	}
	st.rebuildEntities(f)
	st.rebuildScope(f)
	st.rebuildSources(f)
	for _, r := range f.SourceRefs {
		if r.Seq <= asOf {
			st.refs = append(st.refs, r)
		}
	}
	slices.SortFunc(st.refs, bySeqID(func(r SourceRef) int64 { return r.Seq }, func(r SourceRef) string { return r.ID }))
	st.rebuildProposals(f)
	st.rebuildQuestions(f)
	st.rebuildClaims(f)
	for i := range f.CheckRuns {
		r := &f.CheckRuns[i]
		if r.Seq <= asOf {
			st.runs[r.CriterionID] = append(st.runs[r.CriterionID], r)
			if prev, ok := st.runByID[r.ID]; !ok || r.Seq < prev.Seq {
				st.runByID[r.ID] = r
			}
		}
	}
	for _, rs := range st.runs {
		slices.SortFunc(rs, bySeqID(func(r *CheckRun) int64 { return r.Seq }, func(r *CheckRun) string { return r.ID }))
	}
	found := map[string]int64{}
	for _, c := range f.Commits {
		if c.Seq <= asOf {
			if prev, ok := found[c.ID]; !ok || c.Seq < prev {
				found[c.ID] = c.Seq
			}
		}
	}
	for _, l := range f.CommitLinks {
		if s, ok := found[l.CommitID]; ok && l.TargetType == CriterionEntity {
			st.commitSeqs[l.TargetID] = append(st.commitSeqs[l.TargetID], s)
		}
	}
	for i := range f.Decisions {
		d := &f.Decisions[i]
		if d.Seq <= asOf {
			st.decisions[d.CriterionID] = append(st.decisions[d.CriterionID], d)
		}
	}
	for _, ds := range st.decisions {
		slices.SortFunc(ds, bySeqID(func(d *Decision) int64 { return d.Seq }, func(d *Decision) string { return d.ID }))
	}
	return st
}

// activeTarget is the target whose interval contains the cutoff. The MVP
// has at most one; malformed input with more gets the latest created.
func activeTarget(targets []Target, asOf int64) *Target {
	var best *Target
	for i := range targets {
		t := &targets[i]
		if t.CreatedSeq > asOf || (t.ArchivedSeq != 0 && t.ArchivedSeq <= asOf) {
			continue
		}
		if best == nil || cmp.Or(cmp.Compare(t.CreatedSeq, best.CreatedSeq), strings.Compare(t.ID, best.ID)) > 0 {
			best = t
		}
	}
	return best
}

func (st *state) rebuildEntities(f *Facts) {
	rows := map[string]Entity{}
	for _, e := range f.Entities {
		if prev, ok := rows[e.ID]; !ok || strings.Compare(e.Key, prev.Key) < 0 {
			rows[e.ID] = e
		}
	}
	changes := map[string][]BaselineChange{}
	for _, c := range f.BaselineChanges {
		if c.Seq <= st.asOf {
			changes[c.EntityID] = append(changes[c.EntityID], c)
		}
	}
	for id, cs := range changes {
		row, ok := rows[id]
		if !ok {
			continue
		}
		slices.SortFunc(cs, func(a, b BaselineChange) int {
			return cmp.Or(cmp.Compare(a.Seq, b.Seq), cmp.Compare(a.ChangeIndex, b.ChangeIndex), strings.Compare(a.ID, b.ID))
		})
		last := cs[len(cs)-1]
		e := &entityAt{Entity: row, EntityState: last.After, revisionSeq: last.Seq}
		for _, c := range cs {
			if c.After.Revision == last.After.Revision {
				e.revisionSeq = c.Seq
				break
			}
		}
		st.entities[id] = e
		if prev, ok := st.byKey[row.Key]; !ok || strings.Compare(id, prev.ID) < 0 {
			st.byKey[row.Key] = e
		}
	}
}

func (st *state) rebuildScope(f *Facts) {
	if st.target == nil {
		return
	}
	for i := range f.ScopeConfirmations {
		c := &f.ScopeConfirmations[i]
		if c.TargetID != st.target.ID || c.Seq > st.asOf {
			continue
		}
		if st.scope == nil || cmp.Or(cmp.Compare(c.Seq, st.scope.Seq), strings.Compare(c.ID, st.scope.ID)) > 0 {
			st.scope = c
		}
	}
	if st.scope == nil {
		return
	}
	seen := map[string]bool{}
	for _, it := range f.ScopeItems {
		if it.ConfirmationID == st.scope.ID && !seen[it.RequirementID] {
			seen[it.RequirementID] = true
			st.scopeItems = append(st.scopeItems, it)
		}
	}
	slices.SortFunc(st.scopeItems, func(a, b ScopeItem) int { return strings.Compare(a.RequirementID, b.RequirementID) })
}

// rebuildSources finds the source set and each member's hash acknowledged
// at the cutoff: the latest acknowledgement up to the cutoff, or else the
// hash at registration, which is the first acknowledgement's previous hash
// when there is one (the row holds the latest hash).
func (st *state) rebuildSources(f *Facts) {
	acks := map[string][]SourceAck{}
	for _, a := range f.SourceAcks {
		acks[a.SourceID] = append(acks[a.SourceID], a)
	}
	seen := map[string]bool{}
	for _, s := range f.Sources {
		if seen[s.ID] || s.RegisteredSeq > st.asOf {
			continue
		}
		seen[s.ID] = true
		if s.RemovedSeq != 0 && s.RemovedSeq <= st.asOf {
			st.removed[s.ID] = true
			continue
		}
		member := sourceAt{Source: s, acked: s.AckedSHA256}
		if as := acks[s.ID]; len(as) > 0 {
			slices.SortFunc(as, bySeqID(func(a SourceAck) int64 { return a.Seq }, func(a SourceAck) string { return a.ID }))
			member.acked = as[0].PreviousSHA256
			for _, a := range as {
				if a.Seq <= st.asOf {
					member.acked = a.AckedSHA256
				}
			}
		}
		st.sources = append(st.sources, member)
		st.sourceByID[s.ID] = true
	}
	slices.SortFunc(st.sources, func(a, b sourceAt) int { return cmp.Or(strings.Compare(a.Path, b.Path), strings.Compare(a.ID, b.ID)) })
	st.setHash = setHash(st.sources)
}

// setHash is the source set hash of data-model §4.3.
func setHash(members []sourceAt) string {
	byID := slices.Clone(members)
	slices.SortFunc(byID, func(a, b sourceAt) int { return strings.Compare(a.ID, b.ID) })
	h := sha256.New()
	for _, m := range byID {
		_, _ = io.WriteString(h, m.ID+":"+m.acked+"\n")
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (st *state) rebuildProposals(f *Facts) {
	exists := map[string]bool{}
	for _, p := range f.Proposals {
		if p.Seq <= st.asOf {
			exists[p.ID] = true
			st.hasProposal = true
		}
	}
	seen := map[string]bool{}
	for _, it := range f.ProposalItems {
		if seen[it.ID] || !exists[it.ProposalID] {
			continue
		}
		seen[it.ID] = true
		if it.ReviewedSeq == 0 || it.ReviewedSeq > st.asOf {
			st.unreviewed++
		}
	}
}

// rebuildQuestions finds the questions and what each one is about at the
// cutoff (data-model §4.8). A question about a proposal item is linked to
// the entity the item produced only once a review up to the cutoff
// accepted the item; the current, backfilled link is not used for it.
func (st *state) rebuildQuestions(f *Facts) {
	items := map[string]ProposalItem{}
	for _, it := range f.ProposalItems {
		if prev, ok := items[it.ID]; !ok || it.ReviewedSeq < prev.ReviewedSeq {
			items[it.ID] = it
		}
	}
	produced := map[string]BaselineChange{}
	for _, c := range f.BaselineChanges {
		if c.ProposalItemID == "" || c.Seq > st.asOf {
			continue
		}
		prev, ok := produced[c.ProposalItemID]
		if !ok || cmp.Or(cmp.Compare(c.Seq, prev.Seq), cmp.Compare(c.ChangeIndex, prev.ChangeIndex), strings.Compare(c.ID, prev.ID)) < 0 {
			produced[c.ProposalItemID] = c
		}
	}
	seen := map[string]bool{}
	for _, q := range f.Questions {
		if seen[q.ID] || q.CreatedSeq > st.asOf {
			continue
		}
		seen[q.ID] = true
		qa := questionAt{Question: q, open: q.AnsweredSeq == 0 || q.AnsweredSeq > st.asOf}
		switch {
		case q.ProposalItemID != "":
			it, ok := items[q.ProposalItemID]
			reviewed := ok && it.ReviewedSeq != 0 && it.ReviewedSeq <= st.asOf
			switch {
			case reviewed && (it.ReviewState == "accepted" || it.ReviewState == "accepted_with_edits"):
				if c, ok := produced[it.ID]; ok {
					qa.linked = st.entities[c.EntityID]
				}
			case reviewed && it.ReviewState == "rejected":
				qa.itemRejected = true
			default:
				qa.itemPending = true
			}
		case q.RelatedID != "":
			qa.linked = st.entities[q.RelatedID]
			if qa.linked == nil {
				qa.itemPending = true // about an entity that does not exist at the cutoff
			}
		}
		st.questions = append(st.questions, qa)
	}
	slices.SortFunc(st.questions, func(a, b questionAt) int { return cmp.Or(compareKeys(a.Key, b.Key), strings.Compare(a.ID, b.ID)) })
}

func (st *state) rebuildClaims(f *Facts) {
	kinds := map[string]string{}
	for _, b := range f.ClaimBatches {
		if b.Seq <= st.asOf {
			if prev, ok := kinds[b.ID]; !ok || b.Kind < prev {
				kinds[b.ID] = b.Kind
			}
		}
	}
	for _, c := range f.Claims {
		if c.Seq > st.asOf || c.TargetType != string(CriterionEntity) {
			continue
		}
		st.claims[c.CriterionID] = append(st.claims[c.CriterionID], claimAt{Claim: c, kind: kinds[c.BatchID]})
	}
	for _, cs := range st.claims {
		slices.SortFunc(cs, bySeqID(func(c claimAt) int64 { return c.Seq }, func(c claimAt) string { return c.ID }))
	}
}

// currentRefs are an entity's references of its revision at the cutoff
// (data-model §4.5).
func (st *state) currentRefs(e *entityAt) []SourceRef {
	var out []SourceRef
	for _, r := range st.refs {
		if r.EntityID == e.ID && r.EntityRevision == e.Revision {
			out = append(out, r)
		}
	}
	return out
}

func (st *state) refNotFound(e *entityAt) bool {
	return slices.ContainsFunc(st.currentRefs(e), func(r SourceRef) bool { return r.Verification == "not_found" })
}

func (st *state) citesRemoved(e *entityAt) bool {
	return slices.ContainsFunc(st.currentRefs(e), func(r SourceRef) bool { return st.removed[r.SourceID] })
}
