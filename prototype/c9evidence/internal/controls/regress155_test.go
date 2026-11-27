package controls

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"weak"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/pseudo"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/toolctl"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

// The Review 155 regressions of LC-9 that run on the old implementation
// too (review-resolution §204: H3, S2 to S5). They use only what the old
// code already had, and the pseudo-tool's new modes (test fixtures). Each
// states its premise and checks it on the instance before it judges.

var (
	errSim155    = errors.New("r155: a holder close error the seam stands in for")
	errActual155 = errors.New("r155: a real close result the plan did not expect")
)

// runJudged runs one case as its variant and judges it.
func runJudged(t *testing.T, k lc9Case) (Outcome, verdict.Variant) {
	t.Helper()
	v := k.variant(runtime.GOOS)
	env, _ := newEnv(t)
	o := v.Run(env)
	j := Judge(v, runtime.GOOS, o, true)
	t.Logf("%s: %s %v", k.name, j.Judgement, j.Reasons)
	return o, j
}

// callSnap is the call's published copy kept in the instance.
func callSnap(t *testing.T, o Outcome) toolctl.Snapshot {
	t.Helper()
	if o.Instance == nil {
		t.Fatalf("no instance: prep %v", o.PrepErr)
	}
	s, ok := o.Instance.Extra["call"].(toolctl.Snapshot)
	if !ok {
		t.Fatal("no call snapshot in the instance")
	}
	return s
}

// holderRefs are weak pointers to what one output-not-closed run made.
type holderRefs struct {
	inst  weak.Pointer[lc9Run]
	reg   weak.Pointer[toolctl.Registry]
	h     weak.Pointer[holder]
	hcmd  weak.Pointer[exec.Cmd]
	ready weak.Pointer[os.File]
}

// runHolderCase runs output-not-closed in its own frame with H's ready
// read end's real close result replaced by an error (the seam stands in
// for a real failure): H's tasks end, the end's release is unknown. It
// returns weak pointers and the outcome (a copy): when it returns, nothing
// in the test holds the run, its registry or H.
//
//go:noinline
func runHolderCase(t *testing.T) (*holderRefs, Outcome, Variant) {
	t.Helper()
	k := lc9CaseNamed(t, "output-not-closed")
	refs := &holderRefs{}
	k.holderSeams = func(h *holder) {
		refs.h, refs.inst = weak.Make(h), weak.Make(h.c)
		h.closeOp = func(name string, f *os.File) error {
			err := f.Close()
			if name == "ready-r" && err == nil {
				return errSim155
			}
			return err
		}
	}
	drive := k.drive
	k.drive = func(c *lc9Run) {
		refs.reg = weak.Make(c.reg)
		drive(c)
	}
	v := k.variant(runtime.GOOS)
	env, _ := newEnv(t)
	o := v.Run(env)
	if h := refs.h.Value(); h != nil {
		h.mu.Lock()
		refs.hcmd = weak.Make(h.cmd)
		if e := h.ends["ready-r"]; e != nil {
			refs.ready = weak.Make(e.f)
		}
		h.mu.Unlock()
	}
	return refs, o, v
}

// TestR155HolderOutlivesTheVariant is H3: after the LC-9 run returned its
// outcome, with H's tasks ended and a release unknown, a complete
// collection must not reclaim the run, its registry, H, H's Cmd or the
// ready read end. On the old implementation nothing held them (the
// expected failure; as in H1, the weak pointers' becoming nil is the
// runtime's behaviour).
func TestR155HolderOutlivesTheVariant(t *testing.T) {
	refs, o, _ := runHolderCase(t)
	premise := false
	if l, ok := o.Instance.Extra["holder"].([]string); ok {
		for _, s := range l {
			premise = premise || strings.HasPrefix(s, "holder ready-r closed error")
		}
	}
	if !o.Terminal || !premise {
		t.Fatalf("premise: terminal %v, ready-r closed with an error %v", o.Terminal, premise)
	}
	fullGC()
	for what, alive := range map[string]bool{"run": refs.inst.Value() != nil, "registry": refs.reg.Value() != nil, "holder": refs.h.Value() != nil,
		"holder cmd": refs.hcmd.Value() != nil, "ready-r": refs.ready.Value() != nil} {
		if !alive {
			t.Errorf("%s reclaimed after the run returned (a release unknown): it had no owner", what)
		}
	}
}

func fullGC() {
	for range 3 {
		runtime.GC()
	}
}

// TestR155OverflowOtherSignal is S2 (Unix): the pseudo-tool writes 8192
// bytes in one write and then ends itself by SIGTERM; it never writes
// again, so no write meets the closed read end. Everything else is the
// overflow variant's case, judged against its complete expectation (a
// SIGPIPE end): it must not be a hit.
func TestR155OverflowOtherSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no signals on Windows")
	}
	k := lc9CaseNamed(t, "overflow")
	k.mode, k.bytes = pseudo.FloodTerm, 8192
	o, j := runJudged(t, k)
	s := callSnap(t, o)
	if s.Exit == nil || s.Exit.String != "signal: terminated" || !s.Out[0].Overflow || len(s.Out[0].Data) != 4096 {
		t.Fatalf("premise: exit %+v, stdout overflow %v kept %d", s.Exit, s.Out[0].Overflow, len(s.Out[0].Data))
	}
	if j.Judgement == verdict.Hit {
		t.Errorf("a SIGTERM end was taken for the expected SIGPIPE end: %s", j.Judgement)
	}
}

// TestR155OverflowNormalExit is S3: the pseudo-tool writes 8192 bytes in
// one write and then exits normally with 141; judged against the overflow
// variant's complete expectation, it must not be a hit (a normal non-zero
// exit is not a signal end).
func TestR155OverflowNormalExit(t *testing.T) {
	k := lc9CaseNamed(t, "overflow")
	k.mode, k.bytes = pseudo.FloodExit, 8192
	o, j := runJudged(t, k)
	s := callSnap(t, o)
	if s.Exit == nil || s.Exit.String != "exit status 141" || !s.Out[0].Overflow {
		t.Fatalf("premise: exit %+v, stdout overflow %v", s.Exit, s.Out[0].Overflow)
	}
	if j.Judgement == verdict.Hit {
		t.Errorf("a normal exit 141 was taken for the expected end: %s", j.Judgement)
	}
}

// TestR155OverflowExtraRealError is S4: the overflow variant with one more
// real result the plan did not expect (stderr-r's Close; the seam stands
// in for it): it must not be a hit.
func TestR155OverflowExtraRealError(t *testing.T) {
	k := lc9CaseNamed(t, "overflow")
	k.seams = func(_ *lc9Run, s *toolctl.Seams) {
		s.CloseFile = func(_, d int, f *os.File) error {
			err := f.Close()
			if d == toolctl.StderrR && err == nil {
				return errActual155
			}
			return err
		}
	}
	o, j := runJudged(t, k)
	s := callSnap(t, o)
	if s.Desc[toolctl.StderrR].CloseErr == "" || !s.Out[0].Overflow {
		t.Fatalf("premise: stderr-r close %q, stdout overflow %v", s.Desc[toolctl.StderrR].CloseErr, s.Out[0].Overflow)
	}
	if j.Judgement == verdict.Hit {
		t.Errorf("an extra real error was not rejected: %s", j.Judgement)
	}
}

// TestR155ExpiryOtherSignal is S5 (Linux): the pseudo-tool ends itself by
// SIGTERM at once; the start task is held before its one Wait; the outer
// layer moves the clock to W only after both outputs reached EOF (the
// kernel closed the descriptors in the exit, after the group exit by
// SIGTERM was marked), so the request's SIGKILL that follows is ignored
// and the request returns nil. Judged against the expiry variant's
// complete expectation (ended by the request's SIGKILL), it must not be a
// hit.
func TestR155ExpiryOtherSignal(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the pidfd exists only on Linux")
	}
	k := lc9CaseNamed(t, "expiry")
	k.mode = pseudo.Term
	k.gates = []string{"wait"}
	k.hooks = map[string]string{toolctl.HookBeforeWait: "wait"}
	k.drive = func(c *lc9Run) {
		if !c.entered("wait") || !c.until("both outputs at EOF with the pidfd open", func(s toolctl.Snapshot) bool {
			return s.Pidfd.State == toolctl.PidfdOpen && s.Out[0].Done && s.Out[0].EOF && s.Out[1].Done && s.Out[1].EOF
		}) {
			return
		}
		c.clock.Set(c.W)
		if !c.waitReturned() || !c.until("the request done", func(s toolctl.Snapshot) bool { return s.Req.State == toolctl.ReqDone }) {
			return
		}
		c.release("wait")
	}
	o, j := runJudged(t, k)
	s := callSnap(t, o)
	if s.Exit == nil || s.Exit.String != "signal: terminated" || s.Req.State != toolctl.ReqDone || s.Req.Err != "" {
		t.Fatalf("premise: exit %+v request %+v", s.Exit, s.Req)
	}
	if j.Judgement == verdict.Hit {
		t.Errorf("a SIGTERM end was taken for the request's SIGKILL: %s", j.Judgement)
	}
}
