package derive_test

import (
	"reflect"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/toderive"
)

// The counterexamples of acceptance-criteria §4, at the level of the
// derivation: the facts each scenario records, and what the derivation must
// conclude from them. The writes themselves (preconditions, exit codes)
// belong to the work packages of the commands.

type outcome struct {
	criterion derive.CriterionStatus
	reason    string
	delivery  derive.TargetStatus
}

func (f *fixture) outcome(t *testing.T) outcome {
	t.Helper()
	res := f.derive(t)
	c := criterionOf(t, res, f.h.Key(f.ac))
	return outcome{c.Status, c.Reason, res.Delivery.Status}
}

func wantOutcome(t *testing.T, step string, got, want outcome) {
	t.Helper()
	if got != want {
		t.Errorf("%s: %+v, want %+v", step, got, want)
	}
}

// CX-01: the code changes and the check is not run again.
func TestCX01CodeChangedWithoutARun(t *testing.T) {
	f := readyFixture(t)
	f.inventorySkipped = true
	wantOutcome(t, "accepted on A", f.outcome(t), outcome{derive.CriterionAccepted, "direct", derive.TargetReady})
	f.s = commitB
	wantOutcome(t, "the target moves to B", f.outcome(t), outcome{derive.CriterionNeedsReverification, "code_changed", derive.TargetNeedsReverification})
	f.fail(f.ac, commitB)
	wantOutcome(t, "the check fails on B", f.outcome(t), outcome{derive.CriterionFailing, "regressed", derive.TargetInProgress})
}

// CX-02 (derivation part): a run in a worktree with uncommitted changes is
// not qualifying, and the judgement stays on commit A.
func TestCX02DirtyRunDoesNotCount(t *testing.T) {
	f := readyFixture(t)
	f.h.Run(f.ac, factgen.RunSpec{Head: commitA, TreeStart: "dirty", TreeEnd: "dirty", Result: "fail", FailureKind: "nonzero_exit"})
	wantOutcome(t, "after the dirty failure", f.outcome(t), outcome{derive.CriterionAccepted, "direct", derive.TargetReady})
}

// CX-03: two worktrees with opposite results; only the snapshot counts.
func TestCX03TwoWorktrees(t *testing.T) {
	f := newFixture(t)
	f.fail(f.ac, commitA)
	f.h.Run(f.ac, factgen.RunSpec{Head: commitB, Worktree: factgen.FeatureWorktree})
	wantOutcome(t, "main fails, feature passes", f.outcome(t), outcome{derive.CriterionFailing, "check_failed", derive.TargetInProgress})
	f.s = commitB // main fast-forwards to the feature commit
	wantOutcome(t, "after the merge", f.outcome(t), outcome{derive.CriterionReadyForReview, "first_review", derive.TargetReadyForReview})
}

// CX-05: only part of the baseline confirmed; without a scope never ready,
// with a scope of that part it can be.
func TestCX05PartialBaseline(t *testing.T) {
	g := factgen.New(projectID)
	g.SetTarget("default", "refs/heads/main")
	items, _ := g.Propose(8)
	var req, ac string
	g.Review(func(r *factgen.Review) {
		req = r.AddRequirement("", "must")
		r.FromItem(items[0])
		ac = r.AddCriterion(req, "manual", nil)
	})
	g.Decide(ac, factgen.DecisionSpec{Decision: "accepted", Mode: "manual", Snapshot: commitA})
	f := &fixture{h: g, req: req, ac: ac, s: commitA, stage: "in_development", inventorySkipped: true}
	wantOutcome(t, "no scope", f.outcome(t), outcome{derive.CriterionAccepted, "manual", derive.TargetNotAssessed})
	g.ConfirmScope(req)
	got := f.outcome(t)
	wantOutcome(t, "the scope is the one requirement", got, outcome{derive.CriterionAccepted, "manual", derive.TargetReady})
	if res := f.derive(t); res.Metrics.PendingReviewItems != 7 || len(gapsOf(res, derive.GapPendingReview)) != 1 {
		t.Errorf("pending items = %d", res.Metrics.PendingReviewItems)
	}
}

// CX-07: the parent requirement is revised materially.
func TestCX07ParentRevised(t *testing.T) {
	f := newFixture(t)
	second := addAuto(f, f.req)
	f.h.ConfirmScope(f.req)
	f.accept(f.ac, commitA, f.pass(f.ac, commitA))
	f.accept(second, commitA, f.pass(second, commitA))
	f.inventorySkipped = true
	wantOutcome(t, "accepted", f.outcome(t), outcome{derive.CriterionAccepted, "direct", derive.TargetReady})
	f.h.Review(func(r *factgen.Review) { r.Update(f.req, true, nil); r.Scope(f.req) }) // defer both
	wantOutcome(t, "deferred", f.outcome(t), outcome{derive.CriterionNeedsReverification, "parent_revised", derive.TargetNeedsReverification})
	if g := gapsOf(f.derive(t), derive.GapParentRevised); len(g) != 2 {
		t.Errorf("PARENT_REVISED = %+v", g)
	}
	var added string
	f.h.Review(func(r *factgen.Review) {
		r.Reaffirm(f.ac)
		r.Retire(second)
		added = r.AddCriterion(f.req, "automated", check("go test ./new"))
		r.Scope(f.req)
	})
	wantOutcome(t, "reaffirmed", f.outcome(t), outcome{derive.CriterionAccepted, "direct", derive.TargetInProgress})
	if c := criterionOf(t, f.derive(t), f.h.Key(added)); c.Status != derive.CriterionInProgress {
		t.Errorf("the new criterion is %s", c.Status)
	}
}

// CX-09: order is only seq. The model has no time: a later fact by seq wins
// whatever its row position (the shuffle property covers the rows).
func TestCX09OrderIsSeq(t *testing.T) {
	f := newFixture(t)
	f.fail(f.ac, commitA)
	f.pass(f.ac, commitA)
	wantOutcome(t, "fail then pass", f.outcome(t), outcome{derive.CriterionReadyForReview, "first_review", derive.TargetReadyForReview})
	f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
	wantOutcome(t, "then a rejection", f.outcome(t), outcome{derive.CriterionRejected, "rework_requested", derive.TargetInProgress})
}

// CX-10: a partial audit.
func TestCX10PartialAudit(t *testing.T) {
	f := newFixture(t)
	var other, otherAC string
	f.h.Review(func(r *factgen.Review) {
		other = r.AddRequirement(f.goal, "must")
		otherAC = r.AddCriterion(other, "automated", check("go test ./other"))
		r.Scope(f.req, other)
	})
	f.h.Claims("audit", factgen.ClaimSpec{Criterion: f.ac, Assertion: "partial"})
	res := f.derive(t)
	if res.InventoryCoverage == nil || *res.InventoryCoverage != (derive.Ratio{Num: 1, Den: 2}) || res.InitProgress == derive.Inventoried {
		t.Errorf("coverage %+v, init progress %s", res.InventoryCoverage, res.InitProgress)
	}
	if p := criterionOf(t, res, f.h.Key(otherAC)).Progress; p != derive.ProgressUnknown {
		t.Errorf("the other criterion's progress is %s", p)
	}
	if g := gapsOf(res, derive.GapInventoryIncomplete); len(g) != 1 || g[0].Blocking {
		t.Errorf("INVENTORY_INCOMPLETE = %+v", g)
	}
	if res.Delivery.Status != derive.TargetInProgress {
		t.Errorf("delivery = %s", res.Delivery.Status)
	}
}

// CX-12: the agent claims done without valid evidence.
func TestCX12ClaimedWithoutEvidence(t *testing.T) {
	cases := []struct {
		name  string
		setup func(f *fixture)
		want  statusReason
		flag  derive.Flag
	}{
		{"a: no run", func(*fixture) {}, statusReason{derive.CriterionInProgress, "claimed_without_evidence"}, derive.FlagClaimedWithoutEvidence},
		{"b: a pass in a dirty worktree", func(f *fixture) {
			f.h.Run(f.ac, factgen.RunSpec{Head: commitA, TreeStart: "dirty", TreeEnd: "dirty"})
		}, statusReason{derive.CriterionInProgress, "claimed_without_evidence"}, derive.FlagClaimedWithoutEvidence},
		{"c: a pass on another commit", func(f *fixture) { f.pass(f.ac, commitB) },
			statusReason{derive.CriterionNeedsReverification, "code_changed"}, derive.FlagClaimedWithoutEvidence},
		{"d: a failure on S", func(f *fixture) { f.fail(f.ac, commitA) },
			statusReason{derive.CriterionFailing, "check_failed"}, derive.FlagConflict},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			c.setup(f)
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"})
			res := f.derive(t)
			cr := criterionOf(t, res, f.h.Key(f.ac))
			wantStatus(t, statusReason{cr.Status, cr.Reason}, c.want.status, c.want.reason)
			if !hasFlag(cr.Flags, c.flag) || res.Delivery.Status == derive.TargetReady {
				t.Errorf("flags %v, delivery %s", cr.Flags, res.Delivery.Status)
			}
			if c.want.reason == "claimed_without_evidence" && len(gapsOf(res, derive.GapClaimedWithoutEvidence)) != 1 {
				t.Errorf("no CLAIMED_WITHOUT_EVIDENCE gap: %v", gapTypes(res.Gaps))
			}
		})
	}
	t.Run("e: a manual criterion", func(t *testing.T) {
		f := newFixture(t)
		f.ac = manualCriterion(f)
		f.h.ConfirmScope(f.req)
		f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"})
		got := f.outcome(t)
		if got.criterion != derive.CriterionReadyForReview || got.reason != "manual_claimed" || got.delivery == derive.TargetReady {
			t.Errorf("%+v", got)
		}
	})
}

// CX-15: the criterion is revised during a run; the run belongs to the old
// revision.
func TestCX15RevisedDuringARun(t *testing.T) {
	f := newFixture(t)
	f.h.Review(func(r *factgen.Review) { r.Update(f.ac, true, nil); r.Scope(f.req) })
	f.h.Run(f.ac, factgen.RunSpec{Head: commitA, Revision: 1, ConfigHash: f.h.ConfigHash(f.ac)})
	c := criterionOf(t, f.derive(t), f.h.Key(f.ac))
	if c.SnapshotRun != nil || c.Status != derive.CriterionNeedsReverification || c.Reason != "ac_revised" {
		t.Errorf("criterion = %+v", c)
	}
}

// CX-20: an exception acceptance is not carried forward.
func TestCX20ExceptionNotCarried(t *testing.T) {
	f := newFixture(t)
	f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
	f.s = commitB
	wantOutcome(t, "the target moves", f.outcome(t), outcome{derive.CriterionNeedsReverification, "code_changed", derive.TargetNeedsReverification})
}

// CX-21: inconsistent results on one snapshot, with an acceptance elsewhere.
func TestCX21InconsistentResults(t *testing.T) {
	f := newFixture(t)
	f.accept(f.ac, commitA, f.pass(f.ac, commitA))
	f.pass(f.ac, commitB)
	f.fail(f.ac, commitB)
	f.pass(f.ac, commitB)
	f.s = commitB
	res := f.derive(t)
	c := criterionOf(t, res, f.h.Key(f.ac))
	if c.Status != derive.CriterionReadyForReview || c.Reason != "inconsistent_results" || len(gapsOf(res, derive.GapInconsistentResults)) != 1 {
		t.Errorf("criterion %s/%s, gaps %v", c.Status, c.Reason, gapTypes(res.Gaps))
	}
}

// CX-22: the target does not resolve, or the project is not a git project.
func TestCX22NotAssessed(t *testing.T) {
	f := readyFixture(t)
	f.s = ""
	wantOutcome(t, "S is NULL", f.outcome(t), outcome{derive.CriterionNotAssessed, "target_unresolved", derive.TargetNotAssessed})
}

// CX-25: an agent tries to lift a rejection without rework.
func TestCX25RejectionLiftedOnlyByRework(t *testing.T) {
	f := newFixture(t)
	f.inventorySkipped = true
	s1, s2 := commitA, commitB
	f.pass(f.ac, s2) // before the rejection
	f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "rejected", Snapshot: s1})
	f.pass(f.ac, s1)
	wantOutcome(t, "a: a pass on the same commit", f.outcome(t), outcome{derive.CriterionRejected, "rework_requested", derive.TargetInProgress})
	f.s = s2
	wantOutcome(t, "b: the target moves to an older pass", f.outcome(t), outcome{derive.CriterionRejected, "rework_requested", derive.TargetInProgress})
	f.pass(f.ac, s2)
	wantOutcome(t, "a new pass on S2", f.outcome(t), outcome{derive.CriterionReadyForReview, "rework_submitted", derive.TargetReadyForReview})
	f.s = s1
	runs := f.h.Runs(f.ac)
	f.accept(f.ac, s1, runs[len(runs)-2].ID) // the pass on S1 after the rejection
	wantOutcome(t, "the person accepts on S1", f.outcome(t), outcome{derive.CriterionAccepted, "direct", derive.TargetReady})
}

// CX-26, CX-27 (derivation part): an observed pass on a new snapshot after
// an acceptance elsewhere is only ready for review, with grade none until a
// person accepts it; then ready, grade observed.
func TestCX26ObservedEvidence(t *testing.T) {
	f := newFixture(t)
	f.inventorySkipped = true
	p := commitC
	f.accept(f.ac, p, f.pass(f.ac, p))
	run := f.pass(f.ac, commitA)
	res := f.derive(t)
	c := criterionOf(t, res, f.h.Key(f.ac))
	if c.Status != derive.CriterionReadyForReview || c.Reason != "observed_evidence" || res.Delivery.Status != derive.TargetReadyForReview ||
		res.Delivery.EvidenceGrade != derive.NoEvidence || !hasFlag(c.Flags, derive.FlagObservedEvidence) {
		t.Errorf("before: %s/%s, delivery %s/%s, flags %v", c.Status, c.Reason, res.Delivery.Status, res.Delivery.EvidenceGrade, c.Flags)
	}
	f.accept(f.ac, commitA, run)
	res = f.derive(t)
	c = criterionOf(t, res, f.h.Key(f.ac))
	if c.Status != derive.CriterionAccepted || c.Reason != "direct" || res.Delivery.Status != derive.TargetReady ||
		res.Delivery.EvidenceGrade != derive.Observed || !hasFlag(c.Flags, derive.FlagObservedEvidence) {
		t.Errorf("after: %s/%s, delivery %s/%s, flags %v", c.Status, c.Reason, res.Delivery.Status, res.Delivery.EvidenceGrade, c.Flags)
	}
}

// MVP-ACC-08: accept, reject and accept again on one snapshot; the current
// decision is the latest.
func TestACC08LatestDecision(t *testing.T) {
	f := newFixture(t)
	run := f.pass(f.ac, commitA)
	f.accept(f.ac, commitA, run)
	f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA, Evidence: run})
	last := f.accept(f.ac, commitA, run)
	c := criterionOf(t, f.derive(t), f.h.Key(f.ac))
	if c.Status != derive.CriterionAccepted || c.CurrentDecision == nil || c.CurrentDecision.ID != last {
		t.Errorf("criterion = %+v", c)
	}
}

// The workspace view: the latest run of the current revision in the
// worktree at its HEAD, and whether that HEAD is S.
func TestWorkspaceView(t *testing.T) {
	f := newFixture(t)
	f.pass(f.ac, commitB)
	dirty := f.h.Run(f.ac, factgen.RunSpec{Head: commitB, TreeStart: "dirty", TreeEnd: "dirty", Result: "fail"})
	f.h.Run(f.ac, factgen.RunSpec{Head: commitB, Worktree: factgen.FeatureWorktree})
	in := f.input()
	in.Current.Workspace = &derive.Workspace{WorktreeID: factgen.MainWorktree, Head: commitB, TreeState: "dirty"}
	res := mustDerive(t, in)
	c := criterionOf(t, res, f.h.Key(f.ac))
	if c.WorkspaceRun == nil || c.WorkspaceRun.ID != dirty || c.WorkspaceRun.Qualifying {
		t.Errorf("workspace run = %+v", c.WorkspaceRun)
	}
	if res.Workspace == nil || res.Workspace.IsTargetSnapshot || res.Workspace.TreeState != "dirty" {
		t.Errorf("workspace = %+v", res.Workspace)
	}
	in.Current.Workspace.Head = commitA
	if res := mustDerive(t, in); !res.Workspace.IsTargetSnapshot || criterionOf(t, res, f.h.Key(f.ac)).WorkspaceRun != nil {
		t.Errorf("at A: %+v", res.Workspace)
	}
	// The workspace never changes the delivery judgement.
	if !reflect.DeepEqual(mustDerive(t, in).Delivery, f.derive(t).Delivery) {
		t.Error("the workspace changed the delivery")
	}
}

// data-model §6.13: S is NULL at a cutoff without an active target.
func TestNoTargetMeansNoSnapshot(t *testing.T) {
	g := factgen.New(projectID)
	var ac string
	g.Review(func(r *factgen.Review) { ac = r.AddCriterion(r.AddRequirement("", "must"), "manual", nil) })
	res := mustDerive(t, toderive.Input(g.Dump(), g.Current(commitA, "in_development"), g.Seq()))
	c := criterionOf(t, res, g.Key(ac))
	if c.Status != derive.CriterionNotAssessed || res.Target != nil {
		t.Errorf("criterion %s, target %+v", c.Status, res.Target)
	}
}
