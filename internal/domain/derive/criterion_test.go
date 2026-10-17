package derive_test

import (
	"testing"

	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
)

// MVP-ST-01: every rule of the criterion decision table (data-model §6.3)
// is hit, each pair of adjacent rules that can hold together has a
// priority case, and the reason-code priorities of rules 7, 10 and 11 have
// cases.
func TestCriterionDecisionTable(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(f *fixture)
		status derive.CriterionStatus
		reason string
	}{
		// Rule 1.
		{"1: S is NULL", func(f *fixture) { f.s = "" }, derive.CriterionNotAssessed, "target_unresolved"},
		{"1 before 2: S is NULL and the parent is revised", func(f *fixture) {
			f.s = ""
			f.h.Review(func(r *factgen.Review) { r.Update(f.req, true, nil) })
		}, derive.CriterionNotAssessed, "target_unresolved"},
		// Rule 2.
		{"2: the parent is revised (deferred)", func(f *fixture) {
			f.h.Review(func(r *factgen.Review) { r.Update(f.req, true, nil) })
		}, derive.CriterionNeedsReverification, "parent_revised"},
		{"2 before 3: waived, then the parent is revised", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "waived"})
			f.h.Review(func(r *factgen.Review) { r.Update(f.req, true, nil) })
		}, derive.CriterionNeedsReverification, "parent_revised"},
		{"an editorial parent update is not a revision", func(f *fixture) {
			f.h.Review(func(r *factgen.Review) { r.Update(f.req, false, func(x *factgen.Fields) { x.Title = "Renamed" }) })
		}, derive.CriterionInProgress, "unknown"},
		{"a reaffirmed criterion applies to the revised parent", func(f *fixture) {
			f.h.Review(func(r *factgen.Review) { r.Update(f.req, true, nil); r.Reaffirm(f.ac) })
		}, derive.CriterionInProgress, "unknown"},
		// Rule 3.
		{"3: waived", func(f *fixture) { f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "waived"}) }, derive.CriterionWaived, ""},
		{"3 before 4: waived, then a failure on S", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "waived"})
			f.fail(f.ac, commitA)
		}, derive.CriterionWaived, ""},
		// Rule 4.
		{"4: the latest qualifying run on S fails", func(f *fixture) { f.fail(f.ac, commitA) }, derive.CriterionFailing, "check_failed"},
		{"4: an error counts as a failure", func(f *fixture) {
			f.h.Run(f.ac, factgen.RunSpec{Head: commitA, Result: "error", FailureKind: "timeout"})
		}, derive.CriterionFailing, "check_failed"},
		{"4: input_missing counts as a failure", func(f *fixture) {
			f.h.Run(f.ac, factgen.RunSpec{Head: commitA, Result: "error", FailureKind: "input_missing"})
		}, derive.CriterionFailing, "check_failed"},
		{"4 before 5: rejected, then a failure", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
			f.fail(f.ac, commitA)
		}, derive.CriterionFailing, "check_failed"},
		{"4 before 6: accepted, then a failure on S is a regression", func(f *fixture) {
			f.accept(f.ac, commitA, f.pass(f.ac, commitA))
			f.fail(f.ac, commitA)
		}, derive.CriterionFailing, "regressed"},
		{"4: ACC-05: a failure after an exception acceptance is a regression", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
			f.fail(f.ac, commitA)
		}, derive.CriterionFailing, "regressed"},
		// Rule 5.
		{"5: rejected", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
		}, derive.CriterionRejected, "rework_requested"},
		{"5 before 7: a pass on the same snapshot after the rejection", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
			f.pass(f.ac, commitA)
		}, derive.CriterionRejected, "rework_requested"},
		{"5: a pass older than the rejection does not lift it", func(f *fixture) {
			f.pass(f.ac, commitB)
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
			f.s = commitB
		}, derive.CriterionRejected, "rework_requested"},
		{"7: a new snapshot and a pass after the rejection lift it", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
			f.pass(f.ac, commitB)
			f.s = commitB
		}, derive.CriterionReadyForReview, "rework_submitted"},
		// Rule 6.
		{"6: normal acceptance on S", func(f *fixture) { f.accept(f.ac, commitA, f.pass(f.ac, commitA)) }, derive.CriterionAccepted, "direct"},
		{"6: exception acceptance on S", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
		}, derive.CriterionAccepted, "exception"},
		{"6 before 7: accepted on S, with a later pass", func(f *fixture) {
			f.accept(f.ac, commitA, f.pass(f.ac, commitA))
			f.pass(f.ac, commitA)
		}, derive.CriterionAccepted, "direct"},
		// Rule 7.
		{"7: first review", func(f *fixture) { f.pass(f.ac, commitA) }, derive.CriterionReadyForReview, "first_review"},
		{"7: an observed pass does not carry an acceptance forward", func(f *fixture) {
			f.accept(f.ac, commitA, f.pass(f.ac, commitA))
			f.pass(f.ac, commitB)
			f.s = commitB
		}, derive.CriterionReadyForReview, "observed_evidence"},
		{"7: an exception is never carried forward", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
			f.h.Run(f.ac, factgen.RunSpec{Head: commitB, Grade: "snapshot_isolated"})
			f.s = commitB
		}, derive.CriterionReadyForReview, "exception_not_carried"},
		{"7: exception_not_carried before inconsistent_results", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
			f.pass(f.ac, commitB)
			f.fail(f.ac, commitB)
			f.pass(f.ac, commitB)
			f.s = commitB
		}, derive.CriterionReadyForReview, "exception_not_carried"},
		{"7: inconsistent_results before observed_evidence", func(f *fixture) {
			f.accept(f.ac, commitA, f.pass(f.ac, commitA))
			f.pass(f.ac, commitB)
			f.fail(f.ac, commitB)
			f.pass(f.ac, commitB)
			f.s = commitB
		}, derive.CriterionReadyForReview, "inconsistent_results"},
		{"7: inconsistent results block carrying an isolated pass", func(f *fixture) {
			f.accept(f.ac, commitA, f.pass(f.ac, commitA))
			f.h.Run(f.ac, factgen.RunSpec{Head: commitB, Result: "fail", FailureKind: "nonzero_exit", Grade: "snapshot_isolated"})
			f.h.Run(f.ac, factgen.RunSpec{Head: commitB, Grade: "snapshot_isolated"})
			f.s = commitB
		}, derive.CriterionReadyForReview, "inconsistent_results"},
		// Rule 8.
		{"8: a manual acceptance on another snapshot", func(f *fixture) {
			m := manualCriterion(f)
			f.h.Decide(m, factgen.DecisionSpec{Decision: "accepted", Mode: "manual", Snapshot: commitA})
			f.s = commitB
			f.ac = m
		}, derive.CriterionNeedsReverification, "code_changed"},
		{"8 before 9: a manual acceptance on another snapshot and a done claim", func(f *fixture) {
			m := manualCriterion(f)
			f.h.Decide(m, factgen.DecisionSpec{Decision: "accepted", Mode: "manual", Snapshot: commitA})
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: m, Assertion: "done"})
			f.s = commitB
			f.ac = m
		}, derive.CriterionNeedsReverification, "code_changed"},
		{"6: a manual acceptance on S", func(f *fixture) {
			m := manualCriterion(f)
			f.h.Decide(m, factgen.DecisionSpec{Decision: "accepted", Mode: "manual", Snapshot: commitA})
			f.ac = m
		}, derive.CriterionAccepted, "manual"},
		// Rule 9.
		{"9: a manual criterion claimed done", func(f *fixture) {
			m := manualCriterion(f)
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: m, Assertion: "done"})
			f.ac = m
		}, derive.CriterionReadyForReview, "manual_claimed"},
		{"5: a manual rejection holds on the same snapshot", func(f *fixture) {
			m := manualCriterion(f)
			f.h.Decide(m, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: m, Assertion: "done"})
			f.ac = m
		}, derive.CriterionRejected, "rework_requested"},
		{"9: a manual rejection lifted by a new snapshot and a later done claim", func(f *fixture) {
			m := manualCriterion(f)
			f.h.Decide(m, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: m, Assertion: "done"})
			f.s = commitB
			f.ac = m
		}, derive.CriterionReadyForReview, "manual_claimed"},
		{"5: a done claim older than the manual rejection does not lift it", func(f *fixture) {
			m := manualCriterion(f)
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: m, Assertion: "done"})
			f.h.Decide(m, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
			f.s = commitB
			f.ac = m
		}, derive.CriterionRejected, "rework_requested"},
		// Rule 10.
		{"10: a pass on another commit", func(f *fixture) { f.pass(f.ac, commitB) }, derive.CriterionNeedsReverification, "code_changed"},
		{"10: an acceptance on another snapshot", func(f *fixture) {
			f.accept(f.ac, commitB, f.pass(f.ac, commitB))
		}, derive.CriterionNeedsReverification, "code_changed"},
		{"10: a pass of an older revision is ac_revised", func(f *fixture) {
			f.pass(f.ac, commitA)
			f.h.Review(func(r *factgen.Review) {
				r.Update(f.ac, true, func(x *factgen.Fields) { x.Statement = "stricter" })
			})
		}, derive.CriterionNeedsReverification, "ac_revised"},
		{"10: a pass of an older configuration is ac_revised before code_changed", func(f *fixture) {
			f.pass(f.ac, commitB)
			f.pass(f.ac, commitA)
			f.h.Review(func(r *factgen.Review) {
				r.Update(f.ac, true, func(x *factgen.Fields) { x.Check = check("go test -race ./orders") })
			})
		}, derive.CriterionNeedsReverification, "ac_revised"},
		{"10: an acceptance of an older revision is ac_revised", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
			f.h.Review(func(r *factgen.Review) { r.Update(f.ac, true, nil) })
		}, derive.CriterionNeedsReverification, "ac_revised"},
		{"7: a pass of the new revision after an older acceptance is a first review", func(f *fixture) {
			f.accept(f.ac, commitA, f.pass(f.ac, commitA))
			f.h.Review(func(r *factgen.Review) { r.Update(f.ac, true, nil) })
			f.pass(f.ac, commitB)
			f.s = commitB
		}, derive.CriterionReadyForReview, "first_review"},
		{"10 before 11: history and a done claim", func(f *fixture) {
			f.pass(f.ac, commitB)
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"})
		}, derive.CriterionNeedsReverification, "code_changed"},
		{"a non-qualifying pass is no history", func(f *fixture) {
			f.h.Run(f.ac, factgen.RunSpec{Head: commitB, TreeStart: "dirty", TreeEnd: "dirty"})
		}, derive.CriterionInProgress, "unknown"},
		{"a failure on another commit is no history", func(f *fixture) { f.fail(f.ac, commitB) }, derive.CriterionInProgress, "unknown"},
		// Rule 11.
		{"11: no check", func(f *fixture) {
			var a string
			f.h.Review(func(r *factgen.Review) { a = r.AddCriterion(f.req, "automated", nil) })
			f.ac = a
		}, derive.CriterionInProgress, "no_check"},
		{"11: no_check before the progress", func(f *fixture) {
			var a string
			f.h.Review(func(r *factgen.Review) { a = r.AddCriterion(f.req, "automated", nil) })
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: a, Assertion: "done"})
			f.ac = a
		}, derive.CriterionInProgress, "no_check"},
		{"11: claimed without evidence", func(f *fixture) {
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"})
		}, derive.CriterionInProgress, "claimed_without_evidence"},
		{"11: blocked", func(f *fixture) {
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "blocked"})
		}, derive.CriterionInProgress, "blocked"},
		{"11: partial is the agent in progress", func(f *fixture) {
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "partial"})
		}, derive.CriterionInProgress, "agent_in_progress"},
		{"11: not_done", func(f *fixture) {
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "not_done"})
		}, derive.CriterionInProgress, "not_started"},
		{"11: docs_only without a claim", func(f *fixture) { f.stage = "docs_only" }, derive.CriterionInProgress, "not_started"},
		{"11: unknown", func(*fixture) {}, derive.CriterionInProgress, "unknown"},
		{"11: a manual criterion without a claim", func(f *fixture) { f.ac = manualCriterion(f) }, derive.CriterionInProgress, "unknown"},
		// Runs that are not qualifying, or of another revision, do not count.
		{"a dirty run on S does not count", func(f *fixture) {
			f.h.Run(f.ac, factgen.RunSpec{Head: commitA, TreeStart: "dirty", TreeEnd: "dirty", Result: "fail"})
		}, derive.CriterionInProgress, "unknown"},
		{"a context_changed run does not count", func(f *fixture) {
			f.h.Run(f.ac, factgen.RunSpec{Head: commitA, Result: "error", FailureKind: "context_changed"})
		}, derive.CriterionInProgress, "unknown"},
		{"a run whose HEAD changed does not count", func(f *fixture) {
			f.h.Run(f.ac, factgen.RunSpec{Head: commitA, HeadEnd: commitB, Result: "fail"})
		}, derive.CriterionInProgress, "unknown"},
		{"an unknown tree state does not count", func(f *fixture) {
			f.h.Run(f.ac, factgen.RunSpec{Head: commitA, TreeStart: "unknown", TreeEnd: "unknown", Result: "fail"})
		}, derive.CriterionInProgress, "unknown"},
		{"a run of an older revision does not count for V", func(f *fixture) {
			f.fail(f.ac, commitA)
			f.h.Review(func(r *factgen.Review) { r.Update(f.ac, true, nil) })
		}, derive.CriterionInProgress, "unknown"},
		{"the latest run by seq decides V", func(f *fixture) {
			f.fail(f.ac, commitA)
			f.pass(f.ac, commitA)
		}, derive.CriterionReadyForReview, "first_review"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			c.setup(f)
			wantStatus(t, f.statusOf(t, f.ac), c.status, c.reason)
		})
	}
}

// manualCriterion adds a manual criterion to the fixture's requirement.
func manualCriterion(f *fixture) string {
	var m string
	f.h.Review(func(r *factgen.Review) { m = r.AddCriterion(f.req, "manual", nil) })
	return m
}

// isolatedFixture has the criterion accepted on A, the target on B and an
// isolated pass on B (the general legal domain, data-model §6.14).
func isolatedFixture(t *testing.T, inputsA, inputsB []factgen.InputEntry, onB func(f *fixture)) *fixture {
	t.Helper()
	f := newFixture(t)
	if inputsA != nil {
		f.h.Review(func(r *factgen.Review) {
			r.Update(f.ac, true, func(x *factgen.Fields) { x.Check = check("go test ./orders", "orders") })
		})
	}
	f.accept(f.ac, commitA, f.h.Run(f.ac, factgen.RunSpec{Head: commitA, Inputs: inputsA}))
	f.h.Run(f.ac, factgen.RunSpec{Head: commitB, Grade: "snapshot_isolated", Inputs: inputsB})
	if onB != nil {
		onB(f)
	}
	f.s = commitB
	return f
}

var (
	ordersTree   = []factgen.InputEntry{{Path: "orders", Mode: "040000", Type: "tree", OID: factgen.Commit("tree 1")}}
	ordersTree2  = []factgen.InputEntry{{Path: "orders", Mode: "040000", Type: "tree", OID: factgen.Commit("tree 2")}}
	ordersSymlnk = []factgen.InputEntry{{Path: "orders", Mode: "120000", Type: "blob", OID: factgen.Commit("tree 1")}}
)

// MVP-ACC-09 (unit), data-model §6.2: a normal acceptance is carried forward
// only by an isolated qualifying pass, consistent results, the same
// configuration and, with declared inputs, equal input records.
func TestCarryForward(t *testing.T) {
	t.Run("an isolated pass carries the acceptance", func(t *testing.T) {
		f := isolatedFixture(t, nil, nil, nil)
		wantStatus(t, f.statusOf(t, f.ac), derive.CriterionAccepted, "carried_forward")
	})
	t.Run("equal inputs carry it", func(t *testing.T) {
		f := isolatedFixture(t, ordersTree, ordersTree, nil)
		wantStatus(t, f.statusOf(t, f.ac), derive.CriterionAccepted, "carried_forward")
	})
	t.Run("CX-19: changed inputs", func(t *testing.T) {
		f := isolatedFixture(t, ordersTree, ordersTree2, nil)
		wantStatus(t, f.statusOf(t, f.ac), derive.CriterionReadyForReview, "inputs_changed")
	})
	t.Run("CX-19: a mode change with the same object ID", func(t *testing.T) {
		f := isolatedFixture(t, ordersTree, ordersSymlnk, nil)
		wantStatus(t, f.statusOf(t, f.ac), derive.CriterionReadyForReview, "inputs_changed")
	})
	t.Run("a missing input record with declared inputs", func(t *testing.T) {
		f := isolatedFixture(t, ordersTree, nil, nil)
		wantStatus(t, f.statusOf(t, f.ac), derive.CriterionReadyForReview, "inputs_changed")
	})
	t.Run("inputs_changed before observed_evidence", func(t *testing.T) {
		f := newFixture(t)
		f.h.Review(func(r *factgen.Review) {
			r.Update(f.ac, true, func(x *factgen.Fields) { x.Check = check("go test ./orders", "orders") })
		})
		f.accept(f.ac, commitA, f.h.Run(f.ac, factgen.RunSpec{Head: commitA, Inputs: ordersTree}))
		f.h.Run(f.ac, factgen.RunSpec{Head: commitB, Inputs: ordersTree2})
		f.s = commitB
		wantStatus(t, f.statusOf(t, f.ac), derive.CriterionReadyForReview, "inputs_changed")
	})
	t.Run("a later failure on the new snapshot", func(t *testing.T) {
		f := isolatedFixture(t, nil, nil, func(f *fixture) { f.fail(f.ac, commitB) })
		wantStatus(t, f.statusOf(t, f.ac), derive.CriterionFailing, "regressed")
	})
	t.Run("the delivery grade of a carried acceptance is isolated", func(t *testing.T) {
		f := isolatedFixture(t, nil, nil, nil)
		res := f.derive(t)
		if res.Delivery.EvidenceGrade != derive.SnapshotIsolated {
			t.Errorf("evidence grade = %s, want snapshot_isolated", res.Delivery.EvidenceGrade)
		}
	})
}

// MVP-ST-04: the eight cases of the progress table (data-model §6.5).
func TestProgress(t *testing.T) {
	cases := []struct {
		name  string
		setup func(f *fixture)
		want  derive.Progress
	}{
		{"done", func(f *fixture) { f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"}) }, derive.ProgressClaimed},
		{"blocked", func(f *fixture) { f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "blocked"}) }, derive.ProgressBlocked},
		{"partial", func(f *fixture) { f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "partial"}) }, derive.ProgressInProgress},
		{"not_done", func(f *fixture) { f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "not_done"}) }, derive.ProgressNotStarted},
		{"unknown", func(f *fixture) { f.h.Claims("audit", factgen.ClaimSpec{Criterion: f.ac, Assertion: "unknown"}) }, derive.ProgressUnknown},
		{"a commit found after the revision", func(f *fixture) {
			f.h.Scan(factgen.CommitSpec{SHA: commitB, Links: []string{f.ac}})
		}, derive.ProgressInProgress},
		{"docs_only", func(f *fixture) { f.stage = "docs_only" }, derive.ProgressNotStarted},
		{"otherwise unknown", func(*fixture) {}, derive.ProgressUnknown},
		// Priorities and bindings.
		{"a claim before a commit", func(f *fixture) {
			f.h.Scan(factgen.CommitSpec{SHA: commitB, Links: []string{f.ac}})
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "not_done"})
		}, derive.ProgressNotStarted},
		{"a commit before docs_only", func(f *fixture) {
			f.stage = "docs_only"
			f.h.Scan(factgen.CommitSpec{SHA: commitB, Links: []string{f.ac}})
		}, derive.ProgressInProgress},
		{"the latest claim by seq", func(f *fixture) {
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"})
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "blocked"})
		}, derive.ProgressBlocked},
		{"a claim of an older revision does not count", func(f *fixture) {
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"})
			f.h.Review(func(r *factgen.Review) { r.Update(f.ac, true, nil) })
		}, derive.ProgressUnknown},
		{"a commit found before the revision does not count", func(f *fixture) {
			f.h.Scan(factgen.CommitSpec{SHA: commitB, Links: []string{f.ac}})
			f.h.Review(func(r *factgen.Review) { r.Update(f.ac, true, nil) })
		}, derive.ProgressUnknown},
		{"an editorial update keeps the revision start", func(f *fixture) {
			f.h.Scan(factgen.CommitSpec{SHA: commitB, Links: []string{f.ac}})
			f.h.Review(func(r *factgen.Review) { r.Update(f.ac, false, func(x *factgen.Fields) { x.Statement = "reworded" }) })
		}, derive.ProgressInProgress},
		{"a commit linked only to the requirement does not count", func(f *fixture) {
			f.h.Scan(factgen.CommitSpec{SHA: commitB, Links: []string{f.req}})
		}, derive.ProgressUnknown},
		{"unmapped work counts for no criterion", func(f *fixture) {
			f.h.Claims("sync", factgen.ClaimSpec{Assertion: "done"})
		}, derive.ProgressUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			c.setup(f)
			if got := criterionOf(t, f.derive(t), f.h.Key(f.ac)).Progress; got != c.want {
				t.Errorf("progress = %s, want %s", got, c.want)
			}
		})
	}
}

// data-model §6.6: the criterion flags.
func TestCriterionFlags(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(f *fixture)
		want    []derive.Flag
		notWant []derive.Flag
	}{
		{"conflict: claimed done and failing", func(f *fixture) {
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"})
			f.fail(f.ac, commitA)
		}, []derive.Flag{derive.FlagConflict, derive.FlagClaimedWithoutEvidence}, nil},
		{"conflict: not started and V = pass", func(f *fixture) {
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "not_done"})
			f.pass(f.ac, commitA)
		}, []derive.Flag{derive.FlagConflict}, nil},
		{"claimed without evidence", func(f *fixture) {
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"})
		}, []derive.Flag{derive.FlagClaimedWithoutEvidence}, []derive.Flag{derive.FlagConflict}},
		{"claimed with evidence", func(f *fixture) {
			f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"})
			f.pass(f.ac, commitA)
		}, []derive.Flag{derive.FlagObservedEvidence}, []derive.Flag{derive.FlagClaimedWithoutEvidence}},
		{"inconsistent results", func(f *fixture) {
			f.pass(f.ac, commitA)
			f.fail(f.ac, commitA)
		}, []derive.Flag{derive.FlagInconsistentResults}, nil},
		{"exception acceptance", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
		}, []derive.Flag{derive.FlagExceptionAcceptance}, []derive.Flag{derive.FlagObservedEvidence}},
		{"an exception on another snapshot does not apply", func(f *fixture) {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitB})
		}, nil, []derive.Flag{derive.FlagExceptionAcceptance}},
		{"observed evidence of a normal acceptance", func(f *fixture) {
			f.accept(f.ac, commitA, f.pass(f.ac, commitA))
		}, []derive.Flag{derive.FlagObservedEvidence}, nil},
		{"BL-05: inferred", func(f *fixture) {
			var a string
			f.h.Review(func(r *factgen.Review) { a = r.AddCriterionWithOrigin(f.req, "manual", nil, "inferred_from_code") })
			f.ac = a
		}, []derive.Flag{derive.FlagInferred}, nil},
		{"ref_not_found", func(f *fixture) {
			f.h.Review(func(r *factgen.Review) { r.Ref(f.ac, f.src, "not_found") })
		}, []derive.Flag{derive.FlagRefNotFound}, nil},
		{"a reference of an older revision does not count", func(f *fixture) {
			f.h.Review(func(r *factgen.Review) { r.Ref(f.ac, f.src, "not_found") })
			f.h.Review(func(r *factgen.Review) { r.Update(f.ac, true, nil) })
		}, nil, []derive.Flag{derive.FlagRefNotFound}},
		{"SRC-04: source_removed", func(f *fixture) {
			f.h.Review(func(r *factgen.Review) { r.Ref(f.ac, f.src, "verified") })
			f.h.RemoveSource(f.src)
		}, []derive.Flag{derive.FlagSourceRemoved}, nil},
		{"blocking question about the criterion", func(f *fixture) {
			f.h.Propose(0, factgen.QuestionSpec{Blocking: true, AboutEntity: f.ac})
		}, []derive.Flag{derive.FlagBlockingQuestion}, nil},
		{"blocking question about the requirement", func(f *fixture) {
			f.h.Propose(0, factgen.QuestionSpec{Blocking: true, AboutEntity: f.req})
		}, []derive.Flag{derive.FlagBlockingQuestion}, nil},
		{"an answered question", func(f *fixture) {
			_, qs := f.h.Propose(0, factgen.QuestionSpec{Blocking: true, AboutEntity: f.ac})
			f.h.Answer(qs[0])
		}, nil, []derive.Flag{derive.FlagBlockingQuestion}},
		{"a non-blocking question", func(f *fixture) {
			f.h.Propose(0, factgen.QuestionSpec{AboutEntity: f.ac})
		}, nil, []derive.Flag{derive.FlagBlockingQuestion}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			c.setup(f)
			flags := criterionOf(t, f.derive(t), f.h.Key(f.ac)).Flags
			for _, w := range c.want {
				if !hasFlag(flags, w) {
					t.Errorf("flags %v lack %s", flags, w)
				}
			}
			for _, n := range c.notWant {
				if hasFlag(flags, n) {
					t.Errorf("flags %v have %s", flags, n)
				}
			}
		})
	}
}

// BL-05, SRC-04: the requirement flags (M2-Q8).
func TestRequirementFlags(t *testing.T) {
	f := newFixture(t)
	var inferred string
	f.h.Review(func(r *factgen.Review) {
		inferred = r.AddRequirementWithOrigin(f.goal, "should", "inferred_from_code")
		r.Ref(inferred, f.src, "not_found")
		r.AddCriterion(inferred, "manual", nil)
	})
	f.h.RemoveSource(f.src)
	res := f.derive(t)
	got := requirementOf(t, res, f.h.Key(inferred)).Flags
	want := []derive.Flag{derive.FlagRefNotFound, derive.FlagInferred, derive.FlagSourceRemoved}
	if len(got) != len(want) {
		t.Fatalf("flags = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("flags = %v, want %v", got, want)
		}
	}
	if fl := requirementOf(t, res, f.h.Key(f.req)).Flags; !hasFlag(fl, derive.FlagSourceRemoved) || len(fl) != 1 {
		t.Errorf("flags of %s = %v, want [source_removed]", f.h.Key(f.req), fl)
	}
}

// MVP-ST-02: the nine rules of the requirement aggregation (data-model
// §6.7), in order.
func TestRequirementStatus(t *testing.T) {
	// Each case adds criteria to a new requirement and sets their states.
	cases := []struct {
		name  string
		s     string
		build func(f *fixture, r string) // r is the requirement
		want  derive.RequirementStatus
	}{
		{"1: no active criterion", commitA, func(*fixture, string) {}, derive.RequirementNeedsCriteria},
		{"1: all criteria retired", commitA, func(f *fixture, r string) {
			a := addAuto(f, r)
			f.h.Review(func(rv *factgen.Review) { rv.Retire(a) })
		}, derive.RequirementNeedsCriteria},
		{"1 before 2", "", func(*fixture, string) {}, derive.RequirementNeedsCriteria},
		{"2: S is NULL", "", func(f *fixture, r string) { addAuto(f, r) }, derive.RequirementNotAssessed},
		{"3: a failing criterion before a rejected one", commitA, func(f *fixture, r string) {
			f.fail(addAuto(f, r), commitA)
			f.h.Decide(addAuto(f, r), factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
		}, derive.RequirementFailing},
		{"4: a rejected criterion before one in progress", commitA, func(f *fixture, r string) {
			f.h.Decide(addAuto(f, r), factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
			addAuto(f, r)
		}, derive.RequirementRejected},
		{"5: in progress before needs_reverification", commitA, func(f *fixture, r string) {
			addAuto(f, r)
			f.pass(addAuto(f, r), commitB)
		}, derive.RequirementInProgress},
		{"6: needs_reverification before ready_for_review", commitA, func(f *fixture, r string) {
			f.pass(addAuto(f, r), commitB)
			f.pass(addAuto(f, r), commitA)
		}, derive.RequirementNeedsReverification},
		{"7: ready_for_review before accepted", commitA, func(f *fixture, r string) {
			f.pass(addAuto(f, r), commitA)
			a := addAuto(f, r)
			f.accept(a, commitA, f.pass(a, commitA))
		}, derive.RequirementReadyForReview},
		{"8: all waived", commitA, func(f *fixture, r string) {
			f.h.Decide(addAuto(f, r), factgen.DecisionSpec{Decision: "waived"})
			f.h.Decide(addAuto(f, r), factgen.DecisionSpec{Decision: "waived"})
		}, derive.RequirementWaived},
		{"9: accepted and waived", commitA, func(f *fixture, r string) {
			f.h.Decide(addAuto(f, r), factgen.DecisionSpec{Decision: "waived"})
			a := addAuto(f, r)
			f.accept(a, commitA, f.pass(a, commitA))
		}, derive.RequirementAccepted},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			var r string
			f.h.Review(func(rv *factgen.Review) { r = rv.AddRequirement(f.goal, "should") })
			c.build(f, r)
			f.s = c.s
			if got := requirementOf(t, f.derive(t), f.h.Key(r)).Status; got != c.want {
				t.Errorf("requirement = %s, want %s", got, c.want)
			}
		})
	}
}

func addAuto(f *fixture, requirement string) string {
	var a string
	f.h.Review(func(r *factgen.Review) { a = r.AddCriterion(requirement, "automated", check("go test ./x")) })
	return a
}

// data-model §4.4, §6.13: the engine computes the check configuration hash
// of a historical revision with the formula M1 stores.
func TestCheckConfigHashMatchesTheBaseline(t *testing.T) {
	sh := "bash"
	for _, c := range []baseline.Check{
		{Command: "go test ./...", Workdir: "."},
		{Command: "go test ./...", Workdir: "sub", Shell: &sh, Inputs: []string{"b", "a/c"}},
		{Command: "echo \"quoted\" \\ \t\n\u0001 ünïcode", Workdir: ".", Inputs: []string{"z"}},
	} {
		got := derive.CheckConfigHash(derive.Check{Command: c.Command, Workdir: c.Workdir, Shell: c.Shell, Inputs: c.Inputs})
		if want := baseline.CheckConfigHash(c); got != want {
			t.Errorf("CheckConfigHash(%+v) = %s, want %s", c, got, want)
		}
	}
}

// data-model §4.11.2: input records are equal when their {path, mode,
// type, oid} sets are; NULL equals nothing.
func TestInputsEqual(t *testing.T) {
	a := derive.InputEntry{Path: "a", Mode: "100644", Type: "blob", OID: "1"}
	b := derive.InputEntry{Path: "b", Mode: "040000", Type: "tree", OID: "2"}
	cases := []struct {
		name string
		x, y derive.InputRecord
		want bool
	}{
		{"same", derive.InputRecord{a, b}, derive.InputRecord{a, b}, true},
		{"order does not matter", derive.InputRecord{a, b}, derive.InputRecord{b, a}, true},
		{"empty records", derive.InputRecord{}, derive.InputRecord{}, true},
		{"another object ID", derive.InputRecord{a}, derive.InputRecord{{Path: "a", Mode: "100644", Type: "blob", OID: "9"}}, false},
		{"executable bit", derive.InputRecord{a}, derive.InputRecord{{Path: "a", Mode: "100755", Type: "blob", OID: "1"}}, false},
		{"a symlink with the same text", derive.InputRecord{a}, derive.InputRecord{{Path: "a", Mode: "120000", Type: "blob", OID: "1"}}, false},
		{"another saved path", derive.InputRecord{a}, derive.InputRecord{{Path: "./a", Mode: "100644", Type: "blob", OID: "1"}}, false},
		{"an extra entry", derive.InputRecord{a}, derive.InputRecord{a, b}, false},
		{"NULL", nil, derive.InputRecord{a}, false},
		{"both NULL", nil, nil, false},
	}
	for _, c := range cases {
		if got := derive.InputsEqual(c.x, c.y); got != c.want {
			t.Errorf("%s: InputsEqual = %v, want %v", c.name, got, c.want)
		}
	}
}
