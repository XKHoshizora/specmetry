package legality_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/legality"
)

// The random histories of the derivation tests and of the ready checker's
// comparison, with the options those tests use.
var (
	derivationOptions = func(seed uint64) factgen.Options {
		return factgen.Options{Steps: 40, Deliver: 0.5, Cutoff: 0.2, Isolated: seed%3 == 0}
	}
	comparisonOptions = func(seed uint64) factgen.Options {
		return factgen.Options{Steps: 40, Deliver: 0.55, Cutoff: 0.15, Isolated: seed%2 == 1}
	}
)

func problemsOf(c factgen.Case, o factgen.Options, rule string) []legality.Problem {
	var out []legality.Problem
	for _, p := range append(legality.Check(c.Dump, legality.Options{Isolated: o.Isolated}), legality.CheckTrees(c.Dump, c.Trees)...) {
		if rule == "" || p.Rule == rule {
			out = append(out, p)
		}
	}
	return out
}

// Review 046 B1, seed 2 of the derivation tests: one proposal named one
// criterion in two items (a reaffirm, then an update). V-P03 allows a
// target once per proposal.
func TestReview046Seed2NamesATargetOnce(t *testing.T) {
	o := derivationOptions(2)
	if p := problemsOf(factgen.Random(2, o), o, "V-P03"); len(p) > 0 {
		t.Errorf("seed 2:\n%s", legality.Report(p, nil))
	}
}

// Review 046 B1, seed 5 of the comparison: one proposal updated one
// requirement in two items, and one criterion in two others.
func TestReview046Seed5NamesATargetOnce(t *testing.T) {
	o := comparisonOptions(5)
	if p := problemsOf(factgen.Random(5, o), o, "V-P03"); len(p) > 0 {
		t.Errorf("seed 5:\n%s", legality.Report(p, nil))
	}
}

// Review 046 B1, seed 3 of the comparison: a review applied a goal add, a
// requirement update, then another goal add. §5.5 applies every goal item
// before any requirement item.
func TestReview046Seed3FollowsTheApplicationOrder(t *testing.T) {
	o := comparisonOptions(3)
	if p := problemsOf(factgen.Random(3, o), o, "§5.5 order"); len(p) > 0 {
		t.Errorf("seed 3:\n%s", legality.Report(p, nil))
	}
}

// Every random history the derivation tests and the comparison use is
// legal: the generator's claim is checked, not assumed, by the checker and
// by the scans of review047_test.go.
func TestRandomHistoriesAreLegal(t *testing.T) {
	sets := []struct {
		name  string
		seeds uint64
		opts  func(uint64) factgen.Options
	}{
		{"derivation", 400, derivationOptions},
		{"comparison", 600, comparisonOptions},
		{"shuffle", 4, func(seed uint64) factgen.Options {
			return factgen.Options{Steps: 60, Deliver: 0.5, Isolated: seed%2 == 0}
		}},
		{"malformed base", 200, func(seed uint64) factgen.Options {
			return factgen.Options{Steps: 40, Deliver: 0.5, Isolated: seed%2 == 1}
		}},
	}
	for _, set := range sets {
		bad := 0
		covered := map[string]int{}
		for seed := range set.seeds {
			o := set.opts(seed)
			c := factgen.Random(seed, o)
			p, scan := problemsOf(c, o, ""), independent(c.Dump)
			if len(p) > 0 || len(scan) > 0 {
				bad++
				if bad <= 3 {
					t.Errorf("%s seed=%d:\n%s%s", set.name, seed, legality.Report(p, nil), strings.Join(scan, "\n"))
				}
			}
			for k := range coverage(c.Dump) {
				covered[k]++
			}
		}
		if bad > 0 {
			t.Errorf("%s: %d of %d histories are not legal", set.name, bad, set.seeds)
		}
		t.Logf("%s: histories with %v, of %d", set.name, covered, set.seeds)
		// The legal combinations review 047 is about are generated, not
		// avoided.
		if set.seeds >= 200 {
			for _, k := range []string{"a cascade then an item", "input_missing", "a question linked to its accepted item", "a question about a rejected item"} {
				if covered[k] == 0 {
					t.Errorf("%s: no history with %s", set.name, k)
				}
			}
		}
	}
}

// coverage names the legal combinations of review 047 B1 a history has.
func coverage(d factgen.Dump) map[string]bool {
	out := map[string]bool{}
	seen := map[string]bool{}
	for _, ch := range d.BaselineChanges {
		k := fmt.Sprintf("%d/%s", ch.Seq, ch.EntityID)
		if seen[k] && ch.ProposalItemID != nil {
			out["a cascade then an item"] = true
		}
		seen[k] = true
	}
	for _, r := range d.CheckRuns {
		if r.FailureKind != nil && *r.FailureKind == "input_missing" {
			out["input_missing"] = true
		}
	}
	state := map[string]string{}
	for _, it := range d.ProposalItems {
		state[it.ID] = it.ReviewState
	}
	for _, q := range d.Questions {
		switch {
		case q.ProposalItemID == nil:
		case q.RelatedID != nil:
			out["a question linked to its accepted item"] = true
		case state[*q.ProposalItemID] == "rejected":
			out["a question about a rejected item"] = true
		}
	}
	return out
}

// The checker finds a broken rule when one is put into a legal history.
func TestCheckerFindsBrokenRules(t *testing.T) {
	legal := func() factgen.Dump {
		h := factgen.New("01JAAAAAAAPR0JECT000000000")
		h.SetTarget("default", "refs/heads/main")
		h.Apply(func(r *factgen.Review) {
			g := r.AddGoal("g")
			req := r.AddRequirement(g, "must")
			r.AddCriterion(req, "manual", nil)
			r.AddCriterion(req, "manual", nil)
			r.Scope(req)
		})
		return h.Dump()
	}
	if p := legality.Check(legal(), legality.Options{}); len(p) > 0 {
		t.Fatalf("the legal history:\n%s", legality.Report(p, nil))
	}
	cases := []struct {
		rule   string
		mutate func(d *factgen.Dump)
	}{
		{"§5.5 order", func(d *factgen.Dump) {
			// The goal after the requirement.
			d.BaselineChanges[0].ChangeIndex, d.BaselineChanges[1].ChangeIndex = 1, 0
		}},
		{"V-P03", func(d *factgen.Dump) {
			// Two items of the proposal apply to one entity.
			d.BaselineChanges[3].EntityID = d.BaselineChanges[2].EntityID
		}},
		{"explicit change", func(d *factgen.Dump) { d.BaselineChanges[2].ProposalItemID = nil }},
		{"references", func(d *factgen.Dump) {
			d.SourceRefs = append(d.SourceRefs, factgen.SourceRefRow{ID: "01JAAAAAAAREF0000000000000", Seq: 1, EntityType: "goal",
				EntityID: d.Goals[0].ID, EntityRevision: 1, Verification: "verified"})
		}},
		{"unique seq", func(d *factgen.Dump) {
			d.BaselineVersions = append(d.BaselineVersions, factgen.VersionRow{ID: "01JAAAAAAAVERS10N000000000", Seq: 2, Number: 2})
		}},
		{"scope", func(d *factgen.Dump) { d.ScopeConfirmations[0].BaselineVersion = 0 }},
		{"item", func(d *factgen.Dump) { d.ProposalItems[0].ReviewState = "rejected" }},
	}
	for _, c := range cases {
		d := legal()
		c.mutate(&d)
		got := legality.Check(d, legality.Options{})
		if !slices.ContainsFunc(got, func(p legality.Problem) bool { return p.Rule == c.rule }) {
			t.Errorf("%s not found:\n%s", c.rule, legality.Report(got, nil))
		}
	}
}

// Decisions, claims and runs: their preconditions are checked too.
func TestCheckerFindsBrokenPreconditions(t *testing.T) {
	type setup func(h *factgen.History, ac string)
	build := func(extra setup) factgen.Dump {
		h := factgen.New("01JAAAAAAAPR0JECT000000000")
		h.SetTarget("default", "refs/heads/main")
		var ac string
		h.Apply(func(r *factgen.Review) {
			req := r.AddRequirement("", "must")
			ac = r.AddCriterion(req, "automated", nil)
			r.Scope(req)
		})
		extra(h, ac)
		return h.Dump()
	}
	commit := factgen.Commit("A")
	cases := []struct {
		name, rule string
		extra      setup
	}{
		{"a normal acceptance without a pass", "§7.3", func(h *factgen.History, ac string) {
			h.Decide(ac, factgen.DecisionSpec{Decision: "accepted", Mode: "normal", Snapshot: commit})
		}},
		{"a manual acceptance of an automated criterion", "§7.3", func(h *factgen.History, ac string) {
			h.Decide(ac, factgen.DecisionSpec{Decision: "accepted", Mode: "manual", Snapshot: commit})
		}},
		{"a claim of an older revision", "V-C03", func(h *factgen.History, ac string) {
			h.Claims("sync", factgen.ClaimSpec{Criterion: ac, Assertion: "done", Revision: 2})
		}},
		{"unknown outside an audit", "V-C03", func(h *factgen.History, ac string) {
			h.Claims("sync", factgen.ClaimSpec{Criterion: ac, Assertion: "unknown"})
		}},
		{"a run of a criterion without a check", "run", func(h *factgen.History, ac string) {
			h.Run(ac, factgen.RunSpec{Head: commit})
		}},
	}
	for _, c := range cases {
		got := legality.Check(build(c.extra), legality.Options{})
		if !slices.ContainsFunc(got, func(p legality.Problem) bool { return p.Rule == c.rule }) {
			t.Errorf("%s: %s not found:\n%s", c.name, c.rule, legality.Report(got, nil))
		}
	}
	if got := legality.Check(build(func(*factgen.History, string) {}), legality.Options{}); len(got) > 0 {
		t.Errorf("the legal base:\n%s", legality.Report(got, nil))
	}
}
