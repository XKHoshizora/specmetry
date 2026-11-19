package toolctl

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
)

// The Review 155 regressions of the control protocol that run on the old
// implementation too (review-resolution §204: H1 and C2). They use only
// what the old code already had.

// r2fRefs are weak pointers to what R2-f's record holds: they do not keep
// anything reachable.
type r2fRefs struct {
	reg   weak.Pointer[Registry]
	rec   weak.Pointer[Record]
	cmd   weak.Pointer[exec.Cmd]
	proc  weak.Pointer[os.Process]
	files [4]weak.Pointer[os.File]
	pid   int
}

// runR2F runs R2-f's case (the start task panics after a successful Start
// and before its one Wait; the pseudo-tool exits by itself) in its own
// frame, waits until every task of the call has returned, and returns only
// weak pointers: when it returns, nothing in the test holds the registry.
//
//go:noinline
func runR2F(t *testing.T) r2fRefs {
	t.Helper()
	now := time.Now()
	reg := New(Options{Clock: chain.RealClock{}, W: now.Add(time.Hour), D: now.Add(time.Hour + 10*time.Second), Seams: &Seams{
		Prepare: func(_ int, cmd *exec.Cmd) { silentPrepare(cmd) },
		Hook: func(point string, _ int) {
			if point == HookBeforeWait {
				panic("r155: a panic before Wait")
			}
		},
	}})
	waitResult(t, invokeAsync(reg, pseudoSpec(t)), "H1")
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	reg.mu.Lock()
	defer reg.mu.Unlock()
	rec := reg.recs[0]
	if rec.cmd == nil || rec.proc == nil || rec.WaitCalled || rec.Phase != "panic-before-wait" {
		t.Fatalf("premise: cmd %v process %v wait-called %v phase %s", rec.cmd != nil, rec.proc != nil, rec.WaitCalled, rec.Phase)
	}
	refs := r2fRefs{reg: weak.Make(reg), rec: weak.Make(rec), cmd: weak.Make(rec.cmd), proc: weak.Make(rec.proc), pid: rec.Pid}
	for i, f := range rec.files {
		refs.files[i] = weak.Make(f)
	}
	return refs
}

// fullGC runs complete garbage collections (each blocks until its sweep is
// done).
func fullGC() {
	for range 3 {
		runtime.GC()
	}
}

// TestR155RegistryOutlivesItsCaller is H1: after R2-f's caller returned and
// every task of the call ended, a complete collection must not reclaim the
// registry, the record, its Cmd, its Process or its four files: the reap
// is unknown and the responsibility must stay with an owner until the test
// process ends (implementation plan §12.14 items 3 and 10). On the old
// implementation nothing held them (the expected failure: the weak
// pointers become nil; that they do is the runtime's behaviour, not a
// guarantee of the language).
func TestR155RegistryOutlivesItsCaller(t *testing.T) {
	refs := runR2F(t)
	fullGC()
	got := map[string]bool{"registry": refs.reg.Value() != nil, "record": refs.rec.Value() != nil, "cmd": refs.cmd.Value() != nil, "process": refs.proc.Value() != nil}
	for i, w := range refs.files {
		got[DescNames[i]] = w.Value() != nil
	}
	for what, alive := range got {
		if !alive {
			t.Errorf("%s reclaimed after its caller returned (pid %d, reap unknown): it had no owner", what, refs.pid)
		}
	}
}

// TestR155ExitedBeforeRequest is C2 (Linux): the pseudo-tool exits by
// itself (exit 0) while the start task is held before its one Wait; the
// outer layer moves the clock to W only after both outputs reached EOF. The
// premise of this order: the pseudo-tool never closes its standard output
// or error itself, so EOF on both means the kernel closed them in its exit
// (do_exit's exit_files), after exit_group had marked the group exit with
// the code 0. The request that follows reaches a process not yet reaped:
// the signal is accepted and ignored (prepare_signal on a group exit), the
// call returns 0, and the one Wait then reaps exited 0. This is the legal
// branch "exited before the request" of RC-3 (§204); it is the protocol's
// path with a pseudo-tool, never evidence of the real go's lifetime.
func TestR155ExitedBeforeRequest(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the pidfd exists only on Linux")
	}
	clk := chain.NewFakeClock(t154)
	w := t154.Add(100 * time.Second)
	held := newLatch()
	t.Cleanup(held.release)
	reg := New(Options{Clock: clk, W: w, D: w.Add(10 * time.Second), Seams: &Seams{
		Prepare: func(_ int, cmd *exec.Cmd) { silentPrepare(cmd) },
		Hook: func(point string, _ int) {
			if point == HookBeforeWait {
				held.wait()
			}
		},
	}})
	ch := invokeAsync(reg, pseudoSpec(t))
	held.waitEntered(t, "the one Wait")
	if !reg.WaitFor(1, after(guard), func(s Snapshot) bool {
		return s.Pidfd.State == PidfdOpen && s.Out[0].Done && s.Out[0].EOF && s.Out[1].Done && s.Out[1].EOF
	}) {
		t.Fatal("premise: the outputs did not both reach EOF with the pidfd open")
	}
	clk.Set(w)
	waitResult(t, ch, "C2")
	if !reg.WaitFor(1, after(guard), func(s Snapshot) bool { return s.Req.State == ReqDone || s.Req.State == ReqAbandoned }) {
		t.Fatal("the request did not finish")
	}
	held.release()
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	s, _ := reg.Snapshot(1)
	t.Logf("%q", Lines(s))
	if s.Req.State != ReqDone || s.Req.Sends != 1 || s.Req.Err != "" || s.Exit == nil || !s.Exit.Exited || s.Exit.Code != 0 || len(s.Faults) != 0 {
		t.Errorf("request %+v exit %+v faults %+v: not the branch exited before the request", s.Req, s.Exit, s.Faults)
	}
	if !s.Expired || s.Success || s.State != StWaited || s.Class != "B" || len(s.Unknown) != 0 || len(s.NotAdmit) != 0 {
		t.Errorf("expired %v success %v state %s class %s unknown %q not-admitted %q", s.Expired, s.Success, s.State, s.Class, s.Unknown, s.NotAdmit)
	}
}
