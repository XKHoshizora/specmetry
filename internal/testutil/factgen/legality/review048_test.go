package legality_test

import (
	"testing"

	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/legality"
)

// Review 048, B1-R2's residue: a run is input_missing only when a declared
// input is missing at head_start (data-model §7.4 step 2), so a check that
// declares no input never gives one. The two gates are checked one by one:
// legality.Check on the rows alone, legality.CheckTrees against the trees
// of the commits, so neither relies on the other having run.
//
// The cases tell apart a check that declares no input, a declared input
// missing at the commit, and a declared input the run does not record.
func TestInputMissingNeedsADeclaredInput(t *testing.T) {
	x := factgen.Commit("x")
	entry := func(p string) factgen.InputEntry {
		return factgen.InputEntry{Path: p, Mode: "100644", Type: "blob", OID: factgen.Commit(p + "@x")}
	}
	missing := factgen.RunSpec{Head: x, Result: "error", FailureKind: "input_missing"}
	cases := []struct {
		name   string
		inputs []string             // the check's declared inputs
		tree   []string             // the paths of x's tree, the witness
		runs   []factgen.RunSpec    // of the criterion
		other  []factgen.InputEntry // a pass of another criterion declaring "a", at x
		check  string               // legality.Check: "" legal, else the rule
		trees  string               // legality.CheckTrees: "" legal, else the rule
	}{
		{name: "no declared input: a pass, without an input record", runs: []factgen.RunSpec{{Head: x}}},
		{name: "no declared input: input_missing (review 048)", runs: []factgen.RunSpec{missing},
			check: "input_missing", trees: "input_missing"},
		{name: "no declared input: input_missing, whatever the tree has", tree: []string{"a"}, runs: []factgen.RunSpec{missing},
			check: "input_missing", trees: "input_missing"},
		{name: "a declared input the tree lacks: input_missing", inputs: []string{"a"}, runs: []factgen.RunSpec{missing}},
		{name: "a declared input another record shows: input_missing", inputs: []string{"a"}, tree: []string{"a"},
			runs: []factgen.RunSpec{missing}, other: []factgen.InputEntry{entry("a")}, check: "commit tree", trees: "commit tree"},
		{name: "a declared input only the tree shows: input_missing", inputs: []string{"a"}, tree: []string{"a"},
			runs: []factgen.RunSpec{missing}, trees: "commit tree"},
		{name: "a declared input without the run's record", inputs: []string{"a"}, tree: []string{"a"},
			runs: []factgen.RunSpec{{Head: x}}, check: "§4.11.2", trees: "§4.11.2"},
		{name: "two declared inputs, the tree lacks one: input_missing", inputs: []string{"a", "b"}, tree: []string{"a"},
			runs: []factgen.RunSpec{missing}, other: []factgen.InputEntry{entry("a")}},
		{name: "two declared inputs, the tree has both: input_missing", inputs: []string{"a", "b"}, tree: []string{"a", "b"},
			runs: []factgen.RunSpec{missing}, trees: "commit tree"},
		{name: "./a is a, which the tree has: input_missing", inputs: []string{"./a"}, tree: []string{"a"},
			runs: []factgen.RunSpec{missing}, trees: "commit tree"},
	}
	for _, c := range cases {
		h := factgen.New("01JAAAAAAAPR0JECT000000000")
		h.SetTarget("default", "refs/heads/main")
		var ac, other string
		h.Apply(func(r *factgen.Review) {
			req := r.AddRequirement("", "must")
			ac = r.AddCriterion(req, "automated", &baseline.Check{Command: "go test ./...", Workdir: ".", TimeoutSec: 300, Rationale: "r",
				Inputs: c.inputs})
			if c.other != nil {
				other = r.AddCriterion(req, "automated", &baseline.Check{Command: "go test ./a", Workdir: ".", TimeoutSec: 300,
					Rationale: "r", Inputs: []string{"a"}})
			}
		})
		if c.other != nil {
			h.Run(other, factgen.RunSpec{Head: x, Inputs: c.other})
		}
		for _, s := range c.runs {
			if s.FailureKind == "" && len(c.inputs) > 0 && c.check == "" {
				for _, p := range c.inputs {
					s.Inputs = append(s.Inputs, entry(p))
				}
			}
			h.Run(ac, s)
		}
		d := h.Dump()
		tree := map[string]factgen.InputEntry{}
		for _, p := range c.tree {
			tree[p] = entry(p)
		}
		gate := func(gate string, got []legality.Problem, rule string) {
			t.Helper()
			switch {
			case rule == "" && len(got) > 0:
				t.Errorf("%s: %s refuses the legal history:\n%s", c.name, gate, legality.Report(got, nil))
			case rule != "" && !hasRule(got, rule):
				t.Errorf("%s: %s does not find %s:\n%s", c.name, gate, rule, legality.Report(got, nil))
			}
		}
		gate("Check", legality.Check(d, legality.Options{}), c.check)
		gate("CheckTrees", legality.CheckTrees(d, factgen.Trees{x: tree}), c.trees)
	}
}
