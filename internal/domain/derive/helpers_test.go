package derive_test

import (
	"slices"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/toderive"
)

const projectID = "01JAAAAAAAPR0JECT000000000"

// Commits of the tests.
var (
	commitA = factgen.Commit("A")
	commitB = factgen.Commit("B")
	commitC = factgen.Commit("C")
)

func check(command string, inputs ...string) *baseline.Check {
	return &baseline.Check{Command: command, Workdir: ".", TimeoutSec: 300, Inputs: inputs, Rationale: "it tests the criterion"}
}

// fixture is a project with a target on main, one source, and one goal
// whose requirement has one automated criterion with a check, entered by a
// person's edit (propose --apply) with the scope attached. S is commit A.
type fixture struct {
	h              *factgen.History
	goal, req, ac  string
	src            string
	s, stage       string
	sourcesSkipped bool
	// inventorySkipped keeps INVENTORY_INCOMPLETE out of tests about
	// other gaps.
	inventorySkipped bool
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{h: factgen.New(projectID), s: commitA, stage: "in_development"}
	f.h.SetTarget("default", "refs/heads/main")
	f.src = f.h.AddSource("docs/brief.md", factgen.Hash("brief v1"))
	f.h.Apply(func(r *factgen.Review) {
		f.goal = r.AddGoal("Orders")
		f.req = r.AddRequirement(f.goal, "must")
		f.ac = r.AddCriterion(f.req, "automated", check("go test ./orders"))
		r.Ref(f.req, f.src, "verified")
		r.Scope(f.req)
	})
	return f
}

func (f *fixture) current() factgen.Current {
	c := f.h.Current(f.s, f.stage)
	c.SourcesSkipped, c.InventorySkipped = f.sourcesSkipped, f.inventorySkipped
	return c
}

func (f *fixture) input() derive.Input {
	return toderive.Input(f.h.Dump(), f.current(), f.h.Seq())
}

func (f *fixture) derive(t *testing.T) *derive.Result {
	t.Helper()
	return mustDerive(t, f.input())
}

// toInput is a history's input at its latest seq, with every source
// observed unchanged.
func toInput(h *factgen.History, s string) derive.Input {
	return toderive.Input(h.Dump(), h.Current(s, "in_development"), h.Seq())
}

func mustDerive(t *testing.T, in derive.Input) *derive.Result {
	t.Helper()
	res, err := derive.Derive(in)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if res == nil {
		t.Fatal("Derive returned no result")
	}
	return res
}

// pass records a qualifying observed pass of the criterion on a commit.
func (f *fixture) pass(ac, commit string) string {
	return f.h.Run(ac, factgen.RunSpec{Head: commit})
}

func (f *fixture) fail(ac, commit string) string {
	return f.h.Run(ac, factgen.RunSpec{Head: commit, Result: "fail", FailureKind: "nonzero_exit"})
}

// accept is a normal acceptance on a commit relying on a run.
func (f *fixture) accept(ac, commit, run string) string {
	return f.h.Decide(ac, factgen.DecisionSpec{Decision: "accepted", Mode: "normal", Snapshot: commit, Evidence: run})
}

func criterionOf(t *testing.T, res *derive.Result, key string) derive.CriterionView {
	t.Helper()
	for _, r := range res.Requirements {
		for _, c := range r.Criteria {
			if c.Key == key {
				return c
			}
		}
	}
	t.Fatalf("no active criterion %s", key)
	return derive.CriterionView{}
}

func requirementOf(t *testing.T, res *derive.Result, key string) derive.RequirementView {
	t.Helper()
	for _, r := range res.Requirements {
		if r.Key == key {
			return r
		}
	}
	t.Fatalf("no active requirement %s", key)
	return derive.RequirementView{}
}

type statusReason struct {
	status derive.CriterionStatus
	reason string
}

func (f *fixture) statusOf(t *testing.T, ac string) statusReason {
	t.Helper()
	c := criterionOf(t, f.derive(t), f.h.Key(ac))
	return statusReason{c.Status, c.Reason}
}

func wantStatus(t *testing.T, got statusReason, status derive.CriterionStatus, reason string) {
	t.Helper()
	if got.status != status || got.reason != reason {
		t.Errorf("criterion = %s/%s, want %s/%s", got.status, got.reason, status, reason)
	}
}

func hasFlag(flags []derive.Flag, f derive.Flag) bool { return slices.Contains(flags, f) }

func ptr[T any](v T) *T { return &v }

func gapsOf(res *derive.Result, typ derive.GapType) []derive.Gap {
	var out []derive.Gap
	for _, g := range res.Gaps {
		if g.Type == typ {
			out = append(out, g)
		}
	}
	return out
}

func gapTypes(gaps []derive.Gap) []derive.GapType {
	var out []derive.GapType
	for _, g := range gaps {
		out = append(out, g.Type)
	}
	return out
}
