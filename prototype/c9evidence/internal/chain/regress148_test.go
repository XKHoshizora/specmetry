package chain

import (
	"strings"
	"testing"
	"time"
)

// Review 148 R-c: admission of a new outer operation is a decision point of
// the outer registry. Registration alone is not admission.
func TestB148AdmitOp(t *testing.T) {
	start := time.Unix(1_000_000, 0)
	clk := NewFakeClock(start)
	d := start.Add(Budget)
	w := d.Add(-CleanupMargin)
	o := NewOuter(clk, d)

	// Before the operation's deadline: admitted and pending until its result.
	op, ok := o.AdmitOp("prepare a", w)
	if !ok || op == nil {
		t.Fatal("not admitted before W")
	}
	if done, _ := o.OpsTerminal(); done {
		t.Error("an admitted operation must be pending until its result")
	}
	op.Done(nil)

	// At W a preparation is refused: registered terminal as refused, with
	// the reading, never as a success or as the operation's failure.
	clk.Set(w)
	if op, ok := o.AdmitOp("prepare b", w); ok || op != nil {
		t.Fatal("a preparation admitted at W")
	}
	// Between W and D a cleanup is still admitted.
	c, ok := o.AdmitOp("cleanup a", d)
	if !ok {
		t.Fatal("a cleanup refused before D")
	}
	c.Done(nil)
	// At D a cleanup is refused and the expiry latch is set.
	clk.Set(d)
	if _, ok := o.AdmitOp("cleanup b", d); ok {
		t.Fatal("a cleanup admitted at D")
	}
	s := o.Snapshot()
	if !s.Latch {
		t.Error("the refusal at D did not set the latch")
	}
	want := map[string]OpState{"prepare a": OpDone, "prepare b": OpRefused, "cleanup a": OpDone, "cleanup b": OpRefused}
	if len(s.Ops) != len(want) {
		t.Fatalf("ops %+v", s.Ops)
	}
	for _, x := range s.Ops {
		if x.State != want[x.Name] {
			t.Errorf("%s: %v, want %v", x.Name, x.State, want[x.Name])
		}
		if x.State == OpRefused && (x.Outcome == nil || x.Outcome.Err != nil || !strings.Contains(x.Reason, "not admitted")) {
			t.Errorf("%s: refusal record %+v", x.Name, x)
		}
	}
	if done, _ := o.OpsTerminal(); !done {
		t.Error("a refused operation is terminal")
	}
	if OpRefused.String() != "refused" {
		t.Errorf("OpRefused is %q", OpRefused.String())
	}
}
