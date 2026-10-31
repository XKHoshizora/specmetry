package checkrun_test

import (
	"fmt"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/checkrun"
	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/toderive"
)

// WP2.5 spec §7.4: the Runner's qualifying is the derive engine's formula
// (WP2.1, accepted). The two are separate implementations, so this
// compares them on every row the Runner can write: each observation of a
// table of starts, ends and outcomes is classified, made a row of a
// criterion with a check, and derived. The engine's own judgement is
// RunView.Qualifying of the worktree's run; a row on S of the current
// revision and configuration is V's run (SnapshotRun) exactly when it
// qualifies. The facts go straight to toderive: factgen/legality predates
// D6, D19 and A12 and calls some of these rows illegal (implementation
// plan §10.7).
func TestQualifyingMatchesTheDeriveEngine(t *testing.T) {
	const projectID = "01JAAAAAAAPR0JECT000000000"
	h := factgen.New(projectID)
	h.SetTarget("default", "refs/heads/main")
	src := h.AddSource("docs/brief.md", factgen.Hash("brief v1"))
	var ac string
	h.Apply(func(r *factgen.Review) {
		goal := r.AddGoal("Orders")
		req := r.AddRequirement(goal, "must")
		ac = r.AddCriterion(req, "automated", &baseline.Check{Command: "go test ./orders", Workdir: ".", TimeoutSec: 300, Rationale: "it tests the criterion"})
		r.Ref(req, src, "verified")
		r.Scope(req)
	})
	s, other := factgen.Commit("A"), factgen.Commit("B")
	base := h.Dump()
	starts := []checkrun.Context{contextOf(s, checkrun.Clean), contextOf(s, checkrun.Dirty), contextOf(s, checkrun.Unknown)}
	ends := []checkrun.Context{
		contextOf(s, checkrun.Clean), contextOf(s, checkrun.Dirty), contextOf(s, checkrun.Unknown),
		contextOf(other, checkrun.Clean), contextOf("", checkrun.Unknown),
	}
	var observations []checkrun.Observation
	for _, start := range starts {
		for _, e := range ends {
			read := e
			for _, o := range []checkrun.Observation{
				{Launch: checkrun.Started, Exit: exited(0)},
				{Launch: checkrun.Started, Exit: exited(1)},
				{Launch: checkrun.Started, Exit: signaled},
				{Launch: checkrun.Started, TimedOut: true, Exit: signaled},
				{NotStarted: checkrun.InputMissing},
				{NotStarted: checkrun.SpawnError},
				{Launch: checkrun.Started, Exit: exited(0), EndUnreadable: true},
			} {
				o.Start, o.End = start, &read
				observations = append(observations, o)
			}
			observations = append(observations,
				checkrun.Observation{Launch: checkrun.Started, TreeNotConfirmed: true, Exit: exited(0), Start: start},
				checkrun.Observation{Launch: checkrun.ResumeAnomaly, Exit: exited(0), Start: start})
		}
	}
	tested := 0
	for i, o := range observations {
		out, err := checkrun.Classify(o)
		if err != nil {
			t.Fatalf("observation %d: %v", i, err)
		}
		row := checkrun.Row{
			Run: checkrun.Run{
				HeadStart: o.Start.Head, HeadEnd: out.End.Head, TreeStateStart: o.Start.TreeState, TreeStateEnd: out.End.TreeState,
				Result: out.Result, FailureKind: out.FailureKind,
			},
			EvidenceGrade: checkrun.Observed,
		}
		d := base
		d.CheckRuns = nil
		seq := d.LatestSeq + 1
		fact := factgen.CheckRunRow{
			ID: fmt.Sprintf("01JAAAAAAARVN%013d", i), Seq: seq, CriterionID: ac, CriterionRevision: h.Revision(ac),
			RequirementRevision: h.Revision(h.Parent(ac)), CheckConfigHash: h.ConfigHash(ac), WorktreeID: factgen.MainWorktree,
			EvidenceGrade: checkrun.Observed, TreeStateStart: row.TreeStateStart, TreeStateEnd: row.TreeStateEnd, Result: row.Result,
		}
		if row.HeadStart != "" {
			fact.HeadStart = &row.HeadStart
		}
		if row.HeadEnd != "" {
			fact.HeadEnd = &row.HeadEnd
		}
		if row.FailureKind != "" {
			fact.FailureKind = &row.FailureKind
		}
		d.CheckRuns, d.LatestSeq = []factgen.CheckRunRow{fact}, seq
		c := h.Current(s, "in_development")
		c.Workspace = &factgen.Workspace{WorktreeID: factgen.MainWorktree, Head: s, TreeState: checkrun.Clean}
		res, err := derive.Derive(toderive.Input(d, c, seq))
		if err != nil {
			t.Fatalf("observation %d (%s): Derive: %v", i, out.Case, err)
		}
		view := criterionView(t, res, h.Key(ac))
		want := checkrun.Qualifying(row)
		switch {
		case view.WorkspaceRun == nil || view.WorkspaceRun.ID != fact.ID:
			t.Errorf("observation %d (%s): the engine does not see the run in the worktree", i, out.Case)
		case view.WorkspaceRun.Qualifying != want:
			t.Errorf("observation %d (%s, %+v): the engine's qualifying is %v, the Runner's %v", i, out.Case, row.Run, view.WorkspaceRun.Qualifying, want)
		case (view.SnapshotRun != nil && view.SnapshotRun.ID == fact.ID) != want:
			t.Errorf("observation %d (%s, %+v): V's run is %+v, the Runner's qualifying %v", i, out.Case, row.Run, view.SnapshotRun, want)
		}
		tested++
	}
	t.Logf("%d rows compared", tested)
}

func criterionView(t *testing.T, res *derive.Result, key string) derive.CriterionView {
	t.Helper()
	for _, r := range res.Requirements {
		for _, c := range r.Criteria {
			if c.Key == key {
				return c
			}
		}
	}
	t.Fatalf("no criterion %s in the result", key)
	return derive.CriterionView{}
}
