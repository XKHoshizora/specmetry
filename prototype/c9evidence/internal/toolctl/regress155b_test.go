package toolctl

import (
	"os/exec"
	"strings"
	"testing"
	"time"
	"weak"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/keep"
)

// The Review 155 regression of the control protocol that needs the new
// process-lifetime owner (review-resolution §204: H2). It never ran on the
// old implementation.

// runR2FOwned is runR2F that also returns the registry's entry in the
// process owner and Linux's pidfd number; it too keeps nothing reachable
// when it returns.
//
//go:noinline
func runR2FOwned(t *testing.T) (r2fRefs, keep.Entry, int) {
	t.Helper()
	now := time.Now()
	reg := New(Options{Clock: chain.RealClock{}, W: now.Add(time.Hour), D: now.Add(time.Hour + 10*time.Second), Name: "H2", Seams: &Seams{
		Prepare: func(_ int, cmd *exec.Cmd) { silentPrepare(cmd) },
		Hook: func(point string, _ int) {
			if point == HookBeforeWait {
				panic("r155: a panic before Wait")
			}
		},
	}})
	waitResult(t, invokeAsync(reg, pseudoSpec(t)), "H2")
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	reg.mu.Lock()
	defer reg.mu.Unlock()
	rec := reg.recs[0]
	refs := r2fRefs{reg: weak.Make(reg), rec: weak.Make(rec), cmd: weak.Make(rec.cmd), proc: weak.Make(rec.proc), pid: rec.Pid}
	for i, f := range rec.files {
		refs.files[i] = weak.Make(f)
	}
	return refs, reg.Owner(), rec.Pidfd.FD
}

// TestR155OwnerHoldsR2F is H2: after R2-f's caller returned, every task
// ended and complete collections ran, the process owner's entry gives the
// very registry; its record holds the very Cmd, Process and four files;
// the record still says what is known: class C, panic-before-wait, the
// reap unknown, no Wait initiated, (Linux) the pidfd open with the same
// number and never closed; and the panic's fault keeps its original error.
// Nothing was waited for, retried or marked reaped.
func TestR155OwnerHoldsR2F(t *testing.T) {
	refs, owner, fd := runR2FOwned(t)
	fullGC()
	obj, ent, ok := keep.Get(owner.Seq)
	reg, isReg := obj.(*Registry)
	if !ok || !isReg || ent != owner || ent.Kind != KindRegistry || ent.Name != "H2" || reg != refs.reg.Value() {
		t.Fatalf("owner entry %+v ok %v registry %v same %v", ent, ok, isReg, isReg && reg == refs.reg.Value())
	}
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if len(reg.recs) != 1 {
		t.Fatalf("%d records", len(reg.recs))
	}
	rec := reg.recs[0]
	if rec != refs.rec.Value() || rec.cmd == nil || rec.cmd != refs.cmd.Value() || rec.proc == nil || rec.proc != refs.proc.Value() || rec.proc.Pid != refs.pid {
		t.Errorf("record same %v cmd same %v process same %v", rec == refs.rec.Value(), rec.cmd == refs.cmd.Value(), rec.proc == refs.proc.Value())
	}
	for i, f := range rec.files {
		if f == nil || f != refs.files[i].Value() || rec.Desc[i].Release != ReleaseConfirmed {
			t.Errorf("%s: same object %v release %s", DescNames[i], f != nil && f == refs.files[i].Value(), rec.Desc[i].Release)
		}
	}
	unknown := strings.Join(rec.Unknown, "; ")
	if rec.Class != "C" || rec.Phase != "panic-before-wait" || rec.WaitCalled || rec.Exit != nil || rec.State != StStarted ||
		!strings.Contains(unknown, "the reap is unknown") || rec.Tasks != 0 {
		t.Errorf("knowledge: class %s phase %s wait-called %v exit %+v state %s unknown %q tasks %d", rec.Class, rec.Phase, rec.WaitCalled, rec.Exit, rec.State, unknown, rec.Tasks)
	}
	if reg.mode == ModePidfd && (rec.Pidfd.State != PidfdOpen || rec.Pidfd.FD != fd || rec.Pidfd.CloseCalled || rec.Pidfd.Closes != 0) {
		t.Errorf("pidfd %+v (fd %d)", rec.Pidfd, fd)
	}
	panics := 0
	for _, f := range rec.Faults {
		if f.Site == "panic" && f.Source == SourcePanic && f.Err != nil && strings.Contains(f.Err.Error(), "r155: a panic before Wait") {
			panics++
		}
	}
	if panics != 1 || len(rec.Faults) != 1 {
		t.Errorf("faults %+v", rec.Faults)
	}
}
