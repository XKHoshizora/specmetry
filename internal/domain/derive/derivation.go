package derive

import (
	"cmp"
	"slices"
	"strings"
)

// derivation holds one full derivation while its result is assembled.
type derivation struct {
	st       *state
	in       Input
	s        string
	settings Settings
	obs      map[string]string // source ID -> observed hash; "" missing

	reqs     []*entityAt
	criteria map[string][]*criterionEval // requirement ID -> its criteria
	rs, as   map[string]bool             // RS and AS (data-model §6.8), by ID
	scoped   bool                        // a scope confirmation exists
}

func assemble(st *state, in Input) *Result {
	d := &derivation{
		st: st, in: in, s: effectiveS(st, in.Current.Snapshot.Commit), settings: *in.Current.Settings,
		obs: map[string]string{}, criteria: map[string][]*criterionEval{}, rs: map[string]bool{}, as: map[string]bool{},
	}
	for _, o := range in.Current.Observations {
		d.obs[o.SourceID] = o.SHA256
	}
	d.reqs = st.activeRequirements()
	for _, req := range d.reqs {
		for _, e := range st.activeCriteria(req) {
			d.criteria[req.ID] = append(d.criteria[req.ID], evalCriterion(st, e, req, d.s, d.settings))
		}
	}
	if st.scope != nil {
		d.scoped = true
		for _, it := range st.scopeItems {
			if e := st.entities[it.RequirementID]; e != nil && e.Type == RequirementEntity && e.active() {
				d.rs[e.ID] = true
				for _, c := range d.criteria[e.ID] {
					d.as[c.ent.ID] = true
				}
			}
		}
	}

	res := &Result{
		AsOfSeq: in.AsOfSeq, Snapshot: *in.Current.Snapshot, Settings: d.settings,
		BaselineVersion: st.version, SourceSetHash: st.setHash,
	}
	res.Observations = slices.Clone(in.Current.Observations)
	slices.SortFunc(res.Observations, func(a, b Observation) int { return strings.Compare(a.SourceID, b.SourceID) })
	if st.target != nil {
		res.Target = &TargetView{ID: st.target.ID, Name: st.target.Name, Ref: st.target.Ref, Snapshot: d.s}
	}
	if st.scope != nil {
		res.Scope = &ScopeView{
			ConfirmationID: st.scope.ID, BaselineVersion: st.scope.BaselineVersion, SourceSetHash: st.scope.SourceSetHash,
			Stale: d.scopeStale(),
		}
		for _, it := range st.scopeItems {
			if e := st.entities[it.RequirementID]; e != nil {
				res.Scope.Requirements = append(res.Scope.Requirements, e.Key)
			}
		}
		slices.SortFunc(res.Scope.Requirements, compareKeys)
	}
	if w := in.Current.Workspace; w != nil {
		res.Workspace = &WorkspaceView{WorktreeID: w.WorktreeID, Head: w.Head, TreeState: w.TreeState, IsTargetSnapshot: w.Head != "" && w.Head == d.s}
	}
	for _, g := range st.activeGoals() {
		res.Goals = append(res.Goals, GoalView{Key: g.Key, Revision: g.Revision, Title: g.Title})
	}
	for _, s := range st.sources {
		res.Sources = append(res.Sources, SourceView{
			ID: s.ID, Path: s.Path, Kind: s.Kind, AckedSHA256: s.acked, CurrentSHA256: d.obs[s.ID], State: d.sourceState(s),
		})
	}
	for _, req := range d.reqs {
		res.Requirements = append(res.Requirements, d.requirementView(req))
	}
	for _, q := range st.questions {
		if !q.open {
			continue
		}
		v := QuestionView{Key: q.Key, Blocking: q.Blocking, ItemRejected: q.itemRejected}
		if q.linked != nil {
			v.About = q.linked.Key
		}
		res.OpenQuestions = append(res.OpenQuestions, v)
	}
	res.InventoryCoverage = d.coverage()
	res.InitProgress = InitProgressOf(InitFacts{
		HasSources: len(st.sources) > 0, SourcesSkipped: d.settings.SourcesSkipped, HasProposal: st.hasProposal,
		HasBaseline: st.version > 0, ScopeConfirmed: st.scope != nil,
		InventoryComplete: res.InventoryCoverage != nil && res.InventoryCoverage.Num == res.InventoryCoverage.Den,
		InventorySkipped:  d.settings.InventorySkipped, DocsOnly: d.settings.Stage == DocsOnly,
	})
	res.Gaps = d.gaps(res.InventoryCoverage)
	res.Delivery = d.delivery(res.Gaps)
	res.Metrics = d.metrics(res.Requirements)
	return res
}

func (d *derivation) scopeStale() bool {
	sc := d.st.scope
	return sc != nil && (sc.BaselineVersion != d.st.version || sc.SourceSetHash != d.st.setHash)
}

func (d *derivation) sourceState(s sourceAt) SourceState {
	switch h := d.obs[s.ID]; {
	case h == "":
		return SourceMissing
	case h != s.acked:
		return SourceChanged
	}
	return SourceUnchanged
}

// grade is the grade of the run a decision relied on (decisionSummary of
// status.v1): a normal acceptance's or a rejection's evidence; none for
// manual, exception and waived decisions, and without evidence.
func (d *derivation) grade(dec *Decision) Grade {
	if dec.EvidenceID == "" || !(dec.Decision == "rejected" || (dec.Decision == "accepted" && dec.Mode == "normal")) {
		return NoEvidence
	}
	if dec.EvidenceGrade != "" {
		return dec.EvidenceGrade
	}
	if run := d.st.runByID[dec.EvidenceID]; run != nil {
		return run.EvidenceGrade
	}
	return NoEvidence
}

// acceptanceGrade is the grade of the run an accepted criterion's
// acceptance relies on (data-model §6.8): the decision's evidence for a
// direct normal acceptance, V's run when carried forward, none otherwise.
func (d *derivation) acceptanceGrade(c *criterionEval) Grade {
	switch {
	case c.status != CriterionAccepted:
		return NoEvidence
	case c.carried:
		return c.vRun.EvidenceGrade
	case c.reason == "direct" && c.d.Mode == "normal":
		if run := d.st.runByID[c.d.EvidenceID]; run != nil {
			return run.EvidenceGrade
		}
		return d.grade(c.d)
	}
	return NoEvidence
}

func (d *derivation) blockingQuestionAbout(ids ...string) bool {
	return slices.ContainsFunc(d.st.questions, func(q questionAt) bool {
		return q.open && q.Blocking && !q.itemRejected && q.linked != nil && slices.Contains(ids, q.linked.ID)
	})
}

// flags are a criterion's flags (data-model §6.6), in the order of the
// table.
func (d *derivation) flags(c *criterionEval) []Flag {
	var out []Flag
	add := func(cond bool, f Flag) {
		if cond {
			out = append(out, f)
		}
	}
	add((c.progress == ProgressClaimed && c.status == CriterionFailing) || (c.progress == ProgressNotStarted && c.v == pass), FlagConflict)
	add(c.progress == ProgressClaimed && c.status != CriterionReadyForReview && c.status != CriterionAccepted && c.status != CriterionWaived,
		FlagClaimedWithoutEvidence)
	add(c.inconsistent, FlagInconsistentResults)
	add(c.applicable && c.d.Mode == "exception", FlagExceptionAcceptance)
	add(d.acceptanceGrade(c) == Observed || (c.status == CriterionReadyForReview && c.vRun != nil && c.vRun.EvidenceGrade == Observed),
		FlagObservedEvidence)
	add(d.blockingQuestionAbout(c.ent.ID, c.ent.RequirementID), FlagBlockingQuestion)
	add(d.st.refNotFound(c.ent), FlagRefNotFound)
	add(c.ent.Origin == "inferred_from_code", FlagInferred)
	add(d.st.citesRemoved(c.ent), FlagSourceRemoved)
	return out
}

func (d *derivation) requirementFlags(req *entityAt) []Flag {
	var out []Flag
	if d.st.refNotFound(req) {
		out = append(out, FlagRefNotFound)
	}
	if req.Origin == "inferred_from_code" {
		out = append(out, FlagInferred)
	}
	if d.st.citesRemoved(req) {
		out = append(out, FlagSourceRemoved)
	}
	return out
}

// requirementStatus aggregates a requirement's active criteria
// (data-model §6.7).
func (d *derivation) requirementStatus(req *entityAt) RequirementStatus {
	cs := d.criteria[req.ID]
	has := func(s CriterionStatus) bool {
		return slices.ContainsFunc(cs, func(c *criterionEval) bool { return c.status == s })
	}
	switch {
	case len(cs) == 0:
		return RequirementNeedsCriteria
	case d.s == "":
		return RequirementNotAssessed
	case has(CriterionFailing):
		return RequirementFailing
	case has(CriterionRejected):
		return RequirementRejected
	case has(CriterionInProgress):
		return RequirementInProgress
	case has(CriterionNeedsReverification):
		return RequirementNeedsReverification
	case has(CriterionReadyForReview):
		return RequirementReadyForReview
	case !slices.ContainsFunc(cs, func(c *criterionEval) bool { return c.status != CriterionWaived }):
		return RequirementWaived
	}
	return RequirementAccepted
}

func (d *derivation) requirementView(req *entityAt) RequirementView {
	v := RequirementView{
		Key: req.Key, Revision: req.Revision, Goal: req.GoalKey, Title: req.Title, Priority: req.Priority,
		InScope: d.rs[req.ID], Status: d.requirementStatus(req), Flags: d.requirementFlags(req),
	}
	for _, c := range d.criteria[req.ID] {
		v.Criteria = append(v.Criteria, d.criterionView(c))
	}
	return v
}

func runView(r *CheckRun) *RunView {
	if r == nil {
		return nil
	}
	return &RunView{ID: r.ID, Seq: r.Seq, Result: r.Result, EvidenceGrade: r.EvidenceGrade, FailureKind: r.FailureKind, Qualifying: qualifying(r)}
}

func (d *derivation) criterionView(c *criterionEval) CriterionView {
	v := CriterionView{
		Key: c.ent.Key, Revision: c.r, RequirementRevision: c.ar, Verification: c.ent.Verification,
		HasCheck: c.hasCheck, CheckConfigHash: c.h, Status: c.status, Reason: c.reason, Progress: c.progress,
		Flags: d.flags(c), SnapshotRun: runView(c.vRun), WorkspaceRun: runView(d.workspaceRun(c)),
	}
	if dec := c.d; dec != nil {
		v.CurrentDecision = &DecisionView{
			ID: dec.ID, Seq: dec.Seq, Decision: dec.Decision, Mode: dec.Mode, Snapshot: dec.Snapshot,
			EvidenceGrade: d.grade(dec), Rationale: dec.Rationale,
		}
	}
	return v
}

// workspaceRun is the latest run of the criterion's current revision and
// configuration that started in the current worktree at its HEAD.
func (d *derivation) workspaceRun(c *criterionEval) *CheckRun {
	w := d.in.Current.Workspace
	if w == nil || w.Head == "" {
		return nil
	}
	var latest *CheckRun
	for _, r := range d.st.runs[c.ent.ID] {
		if r.WorktreeID == w.WorktreeID && r.HeadStart == w.Head && r.CriterionRevision == c.r && r.CheckConfigHash == c.h {
			latest = r
		}
	}
	return latest
}

// coverage is the inventory coverage (data-model §6.11): the share of the
// active criteria that have an audit claim of their current revision; nil
// without an active criterion.
func (d *derivation) coverage() *Ratio {
	var r Ratio
	for _, req := range d.reqs {
		for _, c := range d.criteria[req.ID] {
			r.Den++
			if slices.ContainsFunc(d.st.claims[c.ent.ID], func(cl claimAt) bool { return cl.kind == "audit" && cl.CriterionRevision == c.r }) {
				r.Num++
			}
		}
	}
	if r.Den == 0 {
		return nil
	}
	return &r
}

// delivery is the target delivery table (data-model §6.8).
func (d *derivation) delivery(gaps []Gap) Delivery {
	out := Delivery{EvidenceGrade: NoEvidence}
	has := func(t GapType) bool { return slices.ContainsFunc(gaps, func(g Gap) bool { return g.Type == t }) }
	var as []*criterionEval
	for _, req := range d.reqs {
		for _, c := range d.criteria[req.ID] {
			if d.as[c.ent.ID] {
				as = append(as, c)
			}
		}
	}
	anyAS := func(s CriterionStatus) bool {
		return slices.ContainsFunc(as, func(c *criterionEval) bool { return c.status == s })
	}
	needsCriteria := slices.ContainsFunc(d.reqs, func(r *entityAt) bool { return d.rs[r.ID] && len(d.criteria[r.ID]) == 0 })
	switch {
	case has(GapNoBaseline) || has(GapNoTarget) || has(GapTargetUnresolved) || has(GapNoScope) || has(GapEmptyScope):
		out.Status = TargetNotAssessed
	case needsCriteria || anyAS(CriterionInProgress) || anyAS(CriterionFailing) || anyAS(CriterionRejected) || has(GapBlockingQuestion):
		out.Status = TargetInProgress
	case has(GapScopeStale) || has(GapSourceChanged) || anyAS(CriterionNeedsReverification):
		out.Status = TargetNeedsReverification
	case anyAS(CriterionReadyForReview):
		out.Status = TargetReadyForReview
	case !slices.ContainsFunc(as, func(c *criterionEval) bool { return c.status != CriterionWaived }):
		out.Status = TargetNotAssessed // ALL_WAIVED
	default:
		out.Status = TargetReady
	}
	for _, c := range as {
		switch d.acceptanceGrade(c) {
		case Observed:
			out.EvidenceGrade = Observed
		case SnapshotIsolated:
			if out.EvidenceGrade == NoEvidence {
				out.EvidenceGrade = SnapshotIsolated
			}
		}
	}
	for _, g := range gaps {
		if g.Blocking {
			out.BlockingGaps = append(out.BlockingGaps, g)
		}
	}
	return out
}

// metrics are the counts of data-model §6.11.
func (d *derivation) metrics(reqs []RequirementView) Metrics {
	m := Metrics{ACStatusCounts: map[CriterionStatus]int{}, PendingReviewItems: d.st.unreviewed}
	var ratio Ratio
	for _, r := range reqs {
		if !r.InScope {
			continue
		}
		if r.Status != RequirementWaived {
			ratio.Den++
		}
		if r.Status == RequirementAccepted {
			ratio.Num++
		}
		for _, c := range r.Criteria {
			m.ACStatusCounts[c.Status]++
			if slices.Contains(c.Flags, FlagClaimedWithoutEvidence) {
				m.ClaimedWithoutEvidence++
			}
			if slices.Contains(c.Flags, FlagExceptionAcceptance) {
				m.ExceptionAcceptances++
			}
		}
	}
	if ratio.Den > 0 {
		m.ScopeAcceptanceRatio = &ratio
	}
	for _, req := range d.reqs {
		for _, c := range d.criteria[req.ID] {
			if d.as[c.ent.ID] && d.acceptanceGrade(c) == Observed {
				m.ObservedEvidenceAcceptances++
			}
		}
	}
	for _, q := range d.st.questions {
		if q.open {
			m.OpenQuestions++
			if q.Blocking && !q.itemRejected {
				m.BlockingQuestions++
			}
		}
	}
	return m
}

// gapOrder is the order of data-model §6.10: the project table, the entity
// table, then the warnings.
var gapOrder = func() map[GapType]int {
	order := []GapType{
		GapNoBaseline, GapNoTarget, GapTargetUnresolved, GapNoScope, GapEmptyScope, GapAllWaived, GapScopeStale,
		GapSourceChanged, GapBlockingQuestion, GapPendingReview, GapInventoryIncomplete,
		GapNeedsCriteria, GapNoCheck, GapClaimedWithoutEvidence, GapNotVerified, GapFailing, GapRejected,
		GapParentRevised, GapNeedsReverification, GapAwaitingReview,
		GapConflict, GapExceptionAcceptance, GapInconsistentResults, GapSourceRemoved,
	}
	m := map[GapType]int{}
	for i, t := range order {
		m[t] = i
	}
	return m
}()

// nextActions are the suggested next steps of data-model §6.10 (the
// warnings without one there take the wording of the schema examples).
var nextActions = map[GapType]string{
	GapNoBaseline: "审核提案", GapNoTarget: "人设置交付目标", GapTargetUnresolved: "检查分支或标签",
	GapNoScope: "人确认交付范围", GapEmptyScope: "重新确认范围", GapAllWaived: "重新确认范围", GapScopeStale: "重新确认范围",
	GapSourceChanged: "处理变化，然后由人确认", GapBlockingQuestion: "回答问题", GapPendingReview: "审核提案",
	GapInventoryIncomplete: "补充 audit", GapNeedsCriteria: "补充 AC", GapNoCheck: "定义检查，或改为 manual",
	GapClaimedWithoutEvidence: "在快照上运行检查", GapNotVerified: "实现并验证", GapFailing: "修复", GapRejected: "返工",
	GapParentRevised: "重新确认、修订或退役 AC", GapNeedsReverification: "在快照上重新运行检查，或人工复核",
	GapAwaitingReview: "人工验收", GapConflict: "核实自述", GapExceptionAcceptance: "在快照上补充自动化检查",
	GapInconsistentResults: "排查结果不一致的检查", GapSourceRemoved: "检查条目的来源引用",
}

// stateGap maps a criterion's delivery status and reason to its state gap
// (data-model §6.10); accepted, waived and not assessed criteria have none.
func stateGap(c *criterionEval) (GapType, bool) {
	switch c.status {
	case CriterionInProgress:
		switch c.reason {
		case "no_check":
			return GapNoCheck, true
		case "claimed_without_evidence":
			return GapClaimedWithoutEvidence, true
		}
		return GapNotVerified, true
	case CriterionFailing:
		return GapFailing, true
	case CriterionRejected:
		return GapRejected, true
	case CriterionNeedsReverification:
		if c.reason == "parent_revised" {
			return GapParentRevised, true
		}
		return GapNeedsReverification, true
	case CriterionReadyForReview:
		return GapAwaitingReview, true
	}
	return "", false
}

// gaps are the gaps of data-model §6.10, in order: blocking first, then by
// the table order, then by key.
func (d *derivation) gaps(coverage *Ratio) []Gap {
	var out []Gap
	add := func(t GapType, blocking bool, key, reason string) {
		out = append(out, Gap{Type: t, Blocking: blocking, TargetKey: key, Reason: reason, NextAction: nextActions[t]})
	}
	st := d.st
	// The project table.
	if st.version == 0 {
		add(GapNoBaseline, true, "", "")
	}
	if st.target == nil {
		add(GapNoTarget, true, "", "")
	} else {
		if d.s == "" {
			add(GapTargetUnresolved, true, "", "")
		}
		if st.scope == nil {
			add(GapNoScope, true, "", "")
		}
	}
	if d.scoped && len(d.rs) == 0 {
		add(GapEmptyScope, true, "", "")
	}
	if len(d.as) > 0 && !d.anyAS(func(c *criterionEval) bool { return c.status != CriterionWaived }) {
		add(GapAllWaived, true, "", "")
	}
	if d.scopeStale() {
		add(GapScopeStale, true, "", "")
	}
	for _, s := range st.sources {
		if state := d.sourceState(s); state != SourceUnchanged {
			add(GapSourceChanged, true, s.Path, string(state))
		}
	}
	for _, q := range st.questions {
		if !q.open || !q.Blocking || q.itemRejected || q.itemPending {
			continue
		}
		if q.linked == nil || d.rs[q.linked.ID] || d.as[q.linked.ID] {
			add(GapBlockingQuestion, true, q.Key, "")
		}
	}
	if st.unreviewed > 0 {
		add(GapPendingReview, false, "", "")
	}
	if d.settings.Stage != DocsOnly && !d.settings.InventorySkipped && coverage != nil && coverage.Num < coverage.Den {
		add(GapInventoryIncomplete, false, "", "")
	}
	// The entity gaps: with a scope, only for RS and AS, blocking; without
	// one, for every active entity, not blocking.
	listed := func(id string) bool { return !d.scoped || d.rs[id] || d.as[id] }
	for _, req := range d.reqs {
		if !listed(req.ID) {
			continue
		}
		if len(d.criteria[req.ID]) == 0 {
			add(GapNeedsCriteria, d.scoped, req.Key, "")
		}
		if st.citesRemoved(req) {
			add(GapSourceRemoved, false, req.Key, "")
		}
		for _, c := range d.criteria[req.ID] {
			if !listed(c.ent.ID) {
				continue
			}
			if t, ok := stateGap(c); ok {
				add(t, d.scoped, c.ent.Key, c.reason)
			}
			flags := d.flags(c)
			if slices.Contains(flags, FlagConflict) {
				add(GapConflict, false, c.ent.Key, "")
			}
			if slices.Contains(flags, FlagExceptionAcceptance) {
				add(GapExceptionAcceptance, false, c.ent.Key, "exception")
			}
			if c.inconsistent {
				add(GapInconsistentResults, false, c.ent.Key, "")
			}
			if slices.Contains(flags, FlagSourceRemoved) {
				add(GapSourceRemoved, false, c.ent.Key, "")
			}
		}
	}
	for _, g := range st.activeGoals() {
		if !st.citesRemoved(g) {
			continue
		}
		if !d.scoped || slices.ContainsFunc(d.reqs, func(r *entityAt) bool { return d.rs[r.ID] && r.GoalKey == g.Key }) {
			add(GapSourceRemoved, false, g.Key, "")
		}
	}
	slices.SortStableFunc(out, func(a, b Gap) int {
		if a.Blocking != b.Blocking {
			if a.Blocking {
				return -1
			}
			return 1
		}
		return cmp.Or(cmp.Compare(gapOrder[a.Type], gapOrder[b.Type]), compareKeys(a.TargetKey, b.TargetKey))
	})
	return out
}

func (d *derivation) anyAS(fn func(c *criterionEval) bool) bool {
	for _, cs := range d.criteria {
		for _, c := range cs {
			if d.as[c.ent.ID] && fn(c) {
				return true
			}
		}
	}
	return false
}
