package job

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/controls"
)

// Review 149 R-c/P: the exact per-step outer operations. These expectations
// were written before the fix; they name each step, so they are not part of
// the pre-fix run (the old code registers one operation per preparation).

func opStates(o chain.OuterSnapshot, prefix string) []string {
	var out []string
	for _, op := range o.Ops {
		if strings.HasPrefix(op.Name, prefix) {
			out = append(out, op.Name+" = "+op.State.String())
		}
	}
	return out
}

func sameList(t *testing.T, what string, got, want []string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("%s:\n got  %q\n want %q", what, got, want)
	}
}

// Healthy: each external call of a preparation and of a change is its own
// admitted, done operation, in order.
func TestB149StepOpsHealthy(t *testing.T) {
	vs := controls.LC3()
	j, _, _, g := newJob149(t, vs)
	settle(t, j, g.freeAll)
	j.RunDomains()
	o := j.outer.Snapshot()
	p := "prepare LC-3 name not in M"
	sameList(t, p, opStates(o, p), []string{
		p + " = done",
		p + ": mkdir parent of go.mod = done",
		p + ": write go.mod = done",
		p + ": mkdir parent of lc3.go = done",
		p + ": write lc3.go = done",
	})
	p = "prepare LC-3 handle not a regular file"
	sameList(t, p, opStates(o, p), []string{
		p + " = done",
		p + ": mkdir parent of go.mod = done",
		p + ": write go.mod = done",
		p + ": mkdir parent of lc3.go = done",
		p + ": write lc3.go = done",
		p + " change: remove lc3.go = done",
		p + " change: mkdir lc3.go = done",
	})
	p = "prepare LC-3 reread differs change"
	sameList(t, p, opStates(o, p), []string{p + ": write lc3.go = done"})
}

// A step that would start after a late return is refused, never started,
// and named; nothing after it is even registered. The preparation's error
// names the refused step.
func TestB149RefusedStepIsNamed(t *testing.T) {
	v := controls.LC3()[0]
	j, _, _ := crossW(t, v, "mkdirall", 1, time.Second, nil)
	p := "prepare LC-3 " + v.Name
	o := j.outer.Snapshot()
	sameList(t, p, opStates(o, p), []string{
		p + " = done",
		p + ": mkdir parent of go.mod = done",
		p + ": write go.mod = refused",
	})
	for _, op := range o.Ops {
		// The registry's qualification is relative to D: returned between W
		// and D, the admitted MkdirAll's result is kept with its reading.
		if op.Name == p+": mkdir parent of go.mod" && (op.Outcome == nil || op.Outcome.Err != nil || op.Outcome.At.Before(j.W)) {
			t.Errorf("the admitted MkdirAll's result after W is not kept: %+v", op)
		}
		if op.Name == p+": write go.mod" && (op.Outcome == nil || op.Outcome.Err != nil || !strings.Contains(op.Reason, "not admitted")) {
			t.Errorf("refusal record %+v", op)
		}
	}
	r := fRecord(t, j, v.Name)
	want := "outer preparation not admitted: the work deadline W has passed (" + p + ": write go.mod)"
	if r.Outcome.PrepErr == nil || r.Outcome.PrepErr.Error() != want || len(r.Variant.Reasons) != 1 || r.Variant.Reasons[0] != want {
		t.Errorf("F record %+v, want the reason %q", r, want)
	}
}

// LC-3's Remove → Mkdir: two steps, the second refused after the first
// returned late.
func TestB149RemoveThenMkdirSteps(t *testing.T) {
	v := controls.LC3()[2]
	j, _, _ := crossW(t, v, "remove", 1, time.Second, nil)
	p := "prepare LC-3 " + v.Name + " change"
	sameList(t, p, opStates(j.outer.Snapshot(), p), []string{
		p + ": remove lc3.go = done",
		p + ": mkdir lc3.go = refused",
	})
	if r := fRecord(t, j, v.Name); !errors.Is(r.Outcome.PrepErr, controls.ErrNotAdmitted) || !strings.HasSuffix(r.Outcome.PrepErr.Error(), "("+p+": mkdir lc3.go)") {
		t.Errorf("F record %+v", r)
	}
}

// The other change closures are single steps with their own names.
func TestB149LC2ChangeSteps(t *testing.T) {
	vs := controls.LC2()[:2] // one byte changed, one more file
	j, _, _, g := newJob149(t, vs)
	settle(t, j, g.freeAll)
	j.RunDomains()
	o := j.outer.Snapshot()
	sameList(t, "LC-2 one byte changed", opStates(o, "prepare LC-2 one byte changed change"), []string{"prepare LC-2 one byte changed change: write lc2.go = done"})
	sameList(t, "LC-2 one more file", opStates(o, "prepare LC-2 one more file change"), []string{"prepare LC-2 one more file change: write extra.txt = done"})
}
