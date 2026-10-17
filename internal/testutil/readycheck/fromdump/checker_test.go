package fromdump_test

import (
	"slices"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/legality"
	"github.com/XKHoshizora/specmetry/internal/testutil/readycheck"
	"github.com/XKHoshizora/specmetry/internal/testutil/readycheck/fromdump"
)

var (
	commitA = factgen.Commit("A")
	commitB = factgen.Commit("B")
)

// world is a legal history for the checker's cases: a target on main, one
// source, one requirement in scope with one automated criterion that has a
// check, entered by a person's edit with the scope attached. S is A.
type world struct {
	h             *factgen.History
	goal, req, ac string
	src           string
	s             string
	observed      map[string]string // overrides of the observations
}

func newWorld(t *testing.T) *world {
	t.Helper()
	w := &world{h: factgen.New("01JAAAAAAAPR0JECT000000000"), s: commitA, observed: map[string]string{}}
	w.h.SetTarget("default", "refs/heads/main")
	w.src = w.h.AddSource("docs/brief.md", factgen.Hash("brief"))
	w.h.Apply(func(r *factgen.Review) {
		w.goal = r.AddGoal("Orders")
		w.req = r.AddRequirement(w.goal, "must")
		w.ac = r.AddCriterion(w.req, "automated", &baseline.Check{Command: "go test ./orders", Workdir: ".", TimeoutSec: 300, Rationale: "r"})
		r.Scope(w.req)
	})
	return w
}

func (w *world) current() factgen.Current {
	c := w.h.Current(w.s, "in_development")
	for i, o := range c.Observations {
		if h, ok := w.observed[o.SourceID]; ok {
			c.Observations[i].SHA256 = h
		}
	}
	return c
}

// verdict is the checker's verdict on the world, which must be a legal
// history: F5's counterexamples and positive cases are built from legal
// histories (data-model §6.14).
func (w *world) verdict(t *testing.T) readycheck.Verdict {
	t.Helper()
	return readycheck.Check(fromdump.Facts(legalDump(t, w.h)), fromdump.Current(w.current()), w.h.Seq())
}

func legalDump(t *testing.T, h *factgen.History) factgen.Dump {
	t.Helper()
	d := h.Dump()
	if p := legality.Check(d, legality.Options{Isolated: true}); len(p) > 0 {
		t.Fatalf("not a legal history:\n%s", legality.Report(p, h.Trace()))
	}
	return d
}

func (w *world) pass(ac, commit string) string { return w.h.Run(ac, factgen.RunSpec{Head: commit}) }

func (w *world) accept(ac, commit, run string) {
	w.h.Decide(ac, factgen.DecisionSpec{Decision: "accepted", Mode: "normal", Snapshot: commit, Evidence: run})
}

// ready is the world with its criterion accepted on A.
func readyWorld(t *testing.T) *world {
	t.Helper()
	w := newWorld(t)
	w.accept(w.ac, commitA, w.pass(w.ac, commitA))
	return w
}

func addCriterion(w *world, verification string) string {
	var a string
	w.h.Apply(func(r *factgen.Review) {
		var c *baseline.Check
		if verification == "automated" {
			c = &baseline.Check{Command: "go test ./x", Workdir: ".", TimeoutSec: 300, Rationale: "r"}
		}
		a = r.AddCriterion(w.req, verification, c)
	})
	w.h.ConfirmScope(w.req)
	return a
}

func failedConditions(v readycheck.Verdict) []readycheck.Condition {
	var out []readycheck.Condition
	for c := range v.Failed {
		out = append(out, c)
	}
	slices.Sort(out)
	return out
}

// F5, the counterexamples that must be found: each built from a legal
// history, aimed at one condition. The checker must name it; it may also
// name only the conditions that fail with it by structure.
func TestCheckerFindsEachCounterexample(t *testing.T) {
	cases := []struct {
		name       string
		build      func(t *testing.T) *world
		target     readycheck.Condition
		dependents []readycheck.Condition
	}{
		{"the current decision is a rejection after an acceptance", func(t *testing.T) *world {
			w := readyWorld(t)
			w.h.Decide(w.ac, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
			return w
		}, readycheck.R6, nil},
		{"a qualifying failure after the acceptance on S", func(t *testing.T) *world {
			w := readyWorld(t)
			w.h.Run(w.ac, factgen.RunSpec{Head: commitA, Result: "fail", FailureKind: "nonzero_exit"})
			return w
		}, readycheck.R6, nil},
		{"the parent revised, the criterion not handled", func(t *testing.T) *world {
			w := readyWorld(t)
			w.h.Apply(func(r *factgen.Review) { r.Update(w.req, true, nil); r.Scope(w.req) })
			return w
		}, readycheck.R6, nil},
		{"the scope is stale by version", func(t *testing.T) *world {
			w := readyWorld(t)
			w.h.Apply(func(r *factgen.Review) { r.AddRequirement("", "could") })
			return w
		}, readycheck.R3, nil},
		{"the scope is stale by the source set", func(t *testing.T) *world {
			w := readyWorld(t)
			w.h.AddSource("docs/new.md", factgen.Hash("new"))
			return w
		}, readycheck.R3, nil},
		{"every requirement of the scope is retired", func(t *testing.T) *world {
			w := readyWorld(t)
			w.h.Apply(func(r *factgen.Review) { r.Retire(w.req) })
			return w
		}, readycheck.R4, []readycheck.Condition{readycheck.R3}},
		{"a requirement without a criterion", func(t *testing.T) *world {
			w := readyWorld(t)
			w.h.Apply(func(r *factgen.Review) { r.Scope(w.req, r.AddRequirement("", "must")) })
			return w
		}, readycheck.R5, nil},
		{"AS all waived", func(t *testing.T) *world {
			w := newWorld(t)
			w.h.Decide(w.ac, factgen.DecisionSpec{Decision: "waived"})
			return w
		}, readycheck.R7, nil},
		{"a blocking question about a requirement of RS", func(t *testing.T) *world {
			w := readyWorld(t)
			w.h.Propose(0, factgen.QuestionSpec{Blocking: true, AboutEntity: w.req})
			return w
		}, readycheck.R8, nil},
		{"a blocking question about a criterion of AS", func(t *testing.T) *world {
			w := readyWorld(t)
			w.h.Propose(0, factgen.QuestionSpec{Blocking: true, AboutEntity: w.ac})
			return w
		}, readycheck.R8, nil},
		{"a blocking question about nothing", func(t *testing.T) *world {
			w := readyWorld(t)
			w.h.Propose(0, factgen.QuestionSpec{Blocking: true})
			return w
		}, readycheck.R8, nil},
		{"a source's content changed", func(t *testing.T) *world {
			w := readyWorld(t)
			w.observed[w.src] = factgen.Hash("edited")
			return w
		}, readycheck.R9, nil},
		{"a source file is missing", func(t *testing.T) *world {
			w := readyWorld(t)
			w.observed[w.src] = ""
			return w
		}, readycheck.R9, nil},
		{"S is NULL", func(t *testing.T) *world {
			w := readyWorld(t)
			w.s = ""
			return w
		}, readycheck.R2, []readycheck.Condition{readycheck.R6}},
		{"a manual acceptance of another snapshot", func(t *testing.T) *world {
			w := newWorld(t)
			m := addCriterion(w, "manual")
			w.h.Decide(w.ac, factgen.DecisionSpec{Decision: "waived"})
			w.h.Decide(m, factgen.DecisionSpec{Decision: "accepted", Mode: "manual", Snapshot: commitA})
			w.s = commitB
			return w
		}, readycheck.R6, nil},
		{"an exception acceptance of another snapshot", func(t *testing.T) *world {
			w := newWorld(t)
			w.h.Decide(w.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
			w.s = commitB
			return w
		}, readycheck.R6, nil},
		{"a normal acceptance not carried forward (observed)", func(t *testing.T) *world {
			w := readyWorld(t)
			w.pass(w.ac, commitB)
			w.s = commitB
			return w
		}, readycheck.R6, nil},
		{"only a decision of an older revision", func(t *testing.T) *world {
			w := readyWorld(t)
			w.h.Apply(func(r *factgen.Review) { r.Update(w.ac, true, nil); r.Scope(w.req) })
			w.pass(w.ac, commitA)
			return w
		}, readycheck.R6, nil},
		{"no baseline version", func(t *testing.T) *world {
			w := &world{h: factgen.New("01JAAAAAAAPR0JECT000000000"), s: commitA, observed: map[string]string{}}
			w.h.SetTarget("default", "refs/heads/main")
			return w
		}, readycheck.R1, []readycheck.Condition{readycheck.R3, readycheck.R4, readycheck.R7}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := c.build(t).verdict(t)
			if _, ok := v.Failed[c.target]; !ok {
				t.Fatalf("the checker did not find %s; failed %v", c.target, failedConditions(v))
			}
			allowed := append([]readycheck.Condition{c.target}, c.dependents...)
			for _, f := range failedConditions(v) {
				if !slices.Contains(allowed, f) {
					t.Errorf("%s also failed (%v); only %v may", f, v.Failed[f], allowed)
				}
			}
		})
	}
}

// F5: a question about a rejected item does not block (the control of the
// blocking-question counterexamples).
func TestRejectedItemQuestionDoesNotBlock(t *testing.T) {
	w := readyWorld(t)
	items, _ := w.h.Propose(1, factgen.QuestionSpec{Blocking: true, AboutItem: 1})
	w.h.Review(func(r *factgen.Review) { r.RejectItem(items[0]) })
	if v := w.verdict(t); !v.Ready() {
		t.Errorf("failed %v", failedConditions(v))
	}
}

// F5, the positive cases.
func TestCheckerPositiveCases(t *testing.T) {
	t.Run("a normal acceptance on observed evidence", func(t *testing.T) {
		v := readyWorld(t).verdict(t)
		if !v.Ready() || v.Grade != "observed" {
			t.Errorf("ready %v, grade %s, failed %v", v.Ready(), v.Grade, failedConditions(v))
		}
	})
	t.Run("manual, exception and waived acceptances", func(t *testing.T) {
		w := newWorld(t)
		m, x := addCriterion(w, "manual"), addCriterion(w, "manual")
		w.h.Decide(w.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
		w.h.Decide(m, factgen.DecisionSpec{Decision: "accepted", Mode: "manual", Snapshot: commitA})
		w.h.Decide(x, factgen.DecisionSpec{Decision: "waived"})
		v := w.verdict(t)
		if !v.Ready() || v.Grade != "none" {
			t.Errorf("ready %v, grade %s, failed %v", v.Ready(), v.Grade, failedConditions(v))
		}
	})
	t.Run("a review sharing one seq, at that seq", func(t *testing.T) {
		w := readyWorld(t)
		before := w.h.Seq()
		items, _ := w.h.Propose(2)
		seq := w.h.Review(func(r *factgen.Review) {
			r.Update(w.req, false, func(f *factgen.Fields) { f.Title = "renamed" })
			r.FromItem(items[0])
			r.Update(w.ac, false, func(f *factgen.Fields) { f.Statement = "reworded" })
			r.FromItem(items[1])
			r.Ref(w.req, w.src, "verified")
			r.Ref(w.ac, w.src, "verified")
			r.Scope(w.req)
		})
		for _, asOf := range []int64{before, seq} {
			v := readycheck.Check(fromdump.Facts(legalDump(t, w.h)), fromdump.Current(w.current()), asOf)
			if !v.Ready() {
				t.Errorf("at %d: failed %v", asOf, failedConditions(v))
			}
		}
	})
	t.Run("carried forward by a synthetic isolated pass (general domain)", func(t *testing.T) {
		w := readyWorld(t)
		w.h.Run(w.ac, factgen.RunSpec{Head: commitB, Grade: "snapshot_isolated"})
		w.s = commitB
		v := w.verdict(t)
		if !v.Ready() || v.Grade != "snapshot_isolated" {
			t.Errorf("ready %v, grade %s, failed %v", v.Ready(), v.Grade, failedConditions(v))
		}
	})
}

// R10: the output's grade is checked against the recomputed one; it is an
// output consistency assertion, not a fact counterexample.
func TestR10OutputConsistency(t *testing.T) {
	v := readyWorld(t).verdict(t)
	if err := readycheck.Consistent(v, true, "observed"); err != nil {
		t.Errorf("a consistent output: %v", err)
	}
	if err := readycheck.Consistent(v, true, "snapshot_isolated"); err == nil {
		t.Error("a tampered grade passed")
	}
	if err := readycheck.Consistent(v, false, "observed"); err == nil {
		t.Error("a ready verdict the output denies passed")
	}
}

// Review 047 B1-R1: one edit retires the goal, whose cascade loses the
// requirement its goal, and revises the requirement materially. The
// checker reads both changes: the criterion no longer applies to the
// requirement's revision (R6) until a reaffirmation, which makes the
// delivery ready again on the same acceptance. The engine agrees on the
// verdict and the grade at each step.
func TestGoalCascadeThenARequirementRevision(t *testing.T) {
	w := readyWorld(t)
	agree := func(step string, ready bool, grade string, failed []readycheck.Condition) {
		t.Helper()
		v := w.verdict(t)
		c := factgen.Case{Dump: legalDump(t, w.h), Current: w.current(), AsOf: w.h.Seq()}
		engine, res := engineReady(t, c, c.Current)
		if v.Ready() != ready || v.Grade != grade || !slices.Equal(failedConditions(v), failed) {
			t.Errorf("%s: ready %v, grade %s, failed %v", step, v.Ready(), v.Grade, failedConditions(v))
		}
		if engine != v.Ready() || readycheck.GradeConsistent(v, string(res.Delivery.EvidenceGrade)) != nil {
			t.Errorf("%s: the engine says %s, %s", step, res.Delivery.Status, res.Delivery.EvidenceGrade)
		}
	}
	agree("accepted", true, "observed", nil)
	w.h.Apply(func(r *factgen.Review) {
		r.Retire(w.goal)
		r.Update(w.req, true, func(f *factgen.Fields) { f.Title = "revised" })
		r.Scope(w.req)
	})
	agree("the goal retired and the requirement revised", false, "none", []readycheck.Condition{readycheck.R6})
	w.h.Apply(func(r *factgen.Review) { r.Reaffirm(w.ac); r.Scope(w.req) })
	agree("reaffirmed", true, "observed", nil)
}

// Review 046 N2: a delivery that is not ready still reports the grade of
// its normal acceptances; GradeConsistent checks it.
func TestGradeOfADeliveryThatIsNotReady(t *testing.T) {
	w := readyWorld(t)
	w.observed[w.src] = factgen.Hash("edited") // R9 fails; the acceptance still applies
	v := w.verdict(t)
	if v.Ready() || v.Grade != "observed" {
		t.Fatalf("ready %v, grade %s, failed %v", v.Ready(), v.Grade, failedConditions(v))
	}
	if err := readycheck.GradeConsistent(v, "observed"); err != nil {
		t.Error(err)
	}
	if err := readycheck.GradeConsistent(v, "none"); err == nil {
		t.Error("a wrong grade of a delivery that is not ready passed")
	}
}

// F5: the checker's converter keeps every row the predicate reads and the
// timing F5 lists, NULL apart from a value.
func TestConverterKeepsTheTiming(t *testing.T) {
	var nullSeen, seen int
	for seed := range uint64(100) {
		c := factgen.Random(seed, factgen.Options{Steps: 40, Deliver: 0.5, Isolated: seed%2 == 1})
		f := fromdump.Facts(c.Dump)
		if len(f.Decisions) != len(c.Dump.Decisions) || len(f.Runs) != len(c.Dump.CheckRuns) || len(f.Changes) != len(c.Dump.BaselineChanges) {
			t.Fatalf("seed %d: rows are lost", seed)
		}
		for i, x := range c.Dump.Decisions {
			y := f.Decisions[i]
			want := ""
			if x.SeenDecisionID != nil {
				want = *x.SeenDecisionID
				seen++
			} else {
				nullSeen++
			}
			if y.SeenDecision != want || y.Seq != x.Seq || y.RequirementRevision != x.RequirementRevision {
				t.Fatalf("seed %d: decision %s is not kept: %+v", seed, x.ID, y)
			}
		}
		for i, r := range c.Dump.CheckRuns {
			if y := f.Runs[i]; y.Worktree != r.WorktreeID || y.RequirementRevision != r.RequirementRevision || y.Seq != r.Seq {
				t.Fatalf("seed %d: run %s is not kept: %+v", seed, r.ID, y)
			}
		}
	}
	if nullSeen == 0 || seen == 0 {
		t.Errorf("decisions: %d with no predecessor, %d with one", nullSeen, seen)
	}
}

// The checker's own configuration hash agrees with the formula M1 stores.
func TestConfigHash(t *testing.T) {
	sh := "pwsh"
	for _, c := range []baseline.Check{
		{Command: "go test ./...", Workdir: "."},
		{Command: "a \"b\"\n\\c", Workdir: "w", Shell: &sh, Inputs: []string{"z", "a"}},
	} {
		got := readycheck.ConfigHash(readycheck.CheckConfig{Command: c.Command, Workdir: c.Workdir, Shell: c.Shell, Inputs: c.Inputs})
		if want := baseline.CheckConfigHash(c); got != want {
			t.Errorf("ConfigHash(%+v) = %s, want %s", c, got, want)
		}
	}
}
