package verdict

import (
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
)

func healthy() Inputs {
	return Inputs{
		T2OK:     true,
		Variants: []Variant{{Control: "LC-2", Variant: "byte", Judgement: Hit}, {Control: "LC-9", Variant: "x", Judgement: NotApplicable}},
		R: []RConfig{
			{Config: "linux/amd64", ABHold: true, Compared: true, Consistent: true},
			{Config: "linux/arm64", ABHold: true, Compared: true, Consistent: true},
		},
	}
}

func TestDecide(t *testing.T) {
	j1 := judge.Real(judge.J1, "x", "s", "k", "")
	cases := []struct {
		name string
		mod  func(*Inputs)
		want Conclusion
	}{
		{"consistent", func(*Inputs) {}, Consistent},
		{"inconsistent", func(in *Inputs) { in.R[1].Consistent = false }, Inconsistent},
		{"a pending variant", func(in *Inputs) { in.Variants[0].Judgement = Pending }, Incomplete},
		{"an R failure", func(in *Inputs) {
			in.R[0].Failures = []judge.Failure{j1}
			in.R[0].ABHold = false
			in.R[0].Compared = false
		}, Incomplete},
		{"an R failure beats inconsistency", func(in *Inputs) { in.R[0].Failures = []judge.Failure{j1}; in.R[1].Consistent = false }, Incomplete},
		{"unevaluated", func(in *Inputs) { in.R[0].Unevaluated = []string{"K"}; in.R[0].Compared = false }, Incomplete},
		{"no configuration", func(in *Inputs) { in.R = nil }, Incomplete},
		{"O cleanup", func(in *Inputs) { in.OFailures = []string{"remove temp dir: EBUSY"} }, Incomplete},
		{"O pending", func(in *Inputs) { in.OPending = []string{"cleanup LC-2"} }, Incomplete},
		{"a miss beats everything", func(in *Inputs) {
			in.Variants[0].Judgement = Miss
			in.R[0].Failures = []judge.Failure{j1}
		}, Invalid},
		{"T2", func(in *Inputs) { in.T2OK = false }, Invalid},
		{"m.Run", func(in *Inputs) { in.MRunCode = 1 }, Invalid},
		{"a defect", func(in *Inputs) { in.Defects = []string{"result for unknown check B'1"} }, Invalid},
	}
	for _, c := range cases {
		in := healthy()
		c.mod(&in)
		got, reasons := Decide(in)
		if got != c.want {
			t.Errorf("%s: %v (%v), want %v", c.name, got, reasons, c.want)
		}
		if (got == Inconsistent || got == Consistent) != (ExitCode(got) == 0) {
			t.Errorf("%s: exit code %d", c.name, ExitCode(got))
		}
	}
}
