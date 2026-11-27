package controls

import (
	"errors"
	"runtime"
	"slices"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/keep"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/pseudo"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/toolctl"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

// The Review 155 regressions of LC-9 that need the new owner or the new
// lines (review-resolution §204: H4, S1, S6). They never ran on the old
// implementation.

// TestR155OwnerHoldsTheHolder is H4: after the output-not-closed run
// returned (H's ready read end's close failed, its release unknown, every
// task of H ended) and complete collections ran, the outcome's entries in
// the process owner give the very run, registry and H; H holds the very
// Cmd and ready read end; the end keeps its original error and its release
// unknown; H's one Wait was reaped (confirmed) with exit 0.
func TestR155OwnerHoldsTheHolder(t *testing.T) {
	refs, o, _ := runHolderCase(t)
	fullGC()
	owner, ok := o.Instance.Extra["owner"].(map[string]int)
	if !ok {
		t.Fatalf("no owner entries in the outcome: %v", o.Instance.Extra["owner"])
	}
	get := func(what, kind, name string) any {
		obj, e, ok := keep.Get(owner[what])
		if !ok || e.Kind != kind || e.Name != name {
			t.Fatalf("%s: entry %+v ok %v, want kind %s name %s", what, e, ok, kind, name)
		}
		return obj
	}
	c, _ := get("instance", kindInstance, "output-not-closed").(*lc9Run)
	reg, _ := get("registry", toolctl.KindRegistry, "LC-9 output-not-closed").(*toolctl.Registry)
	h, _ := get("holder", kindHolder, "output-not-closed").(*holder)
	if c == nil || c != refs.inst.Value() || reg == nil || reg != refs.reg.Value() || h == nil || h != refs.h.Value() || c.reg != reg || c.h != h {
		t.Fatalf("not the same objects: run %v registry %v holder %v", c == refs.inst.Value(), reg == refs.reg.Value(), h == refs.h.Value())
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	e := h.ends["ready-r"]
	if h.cmd == nil || h.cmd != refs.hcmd.Value() || e == nil || e.f != refs.ready.Value() {
		t.Fatalf("H's Cmd same %v, ready-r same %v", h.cmd == refs.hcmd.Value(), e != nil && e.f == refs.ready.Value())
	}
	if e.state != endClosed || !errors.Is(e.err, errSim155) || e.release != relUnknown {
		t.Errorf("ready-r: state %s err %v release %s", e.state, e.err, e.release)
	}
	if h.wait != "done" || h.reap != "confirmed" || h.end != "exited 0" || h.waitErr != nil || h.tasks != 0 {
		t.Errorf("H's Wait: %s reap %s end %s err %v tasks %d", h.wait, h.reap, h.end, h.waitErr, h.tasks)
	}
	n := 0
	for _, f := range h.faults {
		if errors.Is(f.Err, errSim155) {
			n++
		}
	}
	if n != 1 {
		t.Errorf("H's faults keep the original error %d times: %+v", n, h.faults)
	}
	if s, ok := reg.Snapshot(1); !ok || !s.Terminal || toolctl.ExitKind(s.Exit) != "exited 0" {
		t.Errorf("the registry's record: %v %q", ok, toolctl.Lines(s))
	}
}

// TestR155OverflowSIGPIPEHit is S1 (Unix): the overflow variant, run as it
// is (the pseudo-tool's write after the close meets the closed read end),
// is a hit, and the judged lines carry the end's identity.
func TestR155OverflowSIGPIPEHit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no signals on Windows: the overflow's end is exited 0 there (the variant's own expectation)")
	}
	o, j := runJudged(t, lc9CaseNamed(t, "overflow"))
	if j.Judgement != verdict.Hit || !slices.Contains(o.Lines, "wait signaled SIGPIPE") ||
		!slices.Contains(o.Lines, "fault wait operation exit-status signaled SIGPIPE") {
		t.Errorf("%s %v; lines %q", j.Judgement, j.Reasons, o.Lines)
	}
}

// TestR155ExitKindLines is S6: the ends of S1, S2 and S3 are told apart in
// the judged lines and the fault kinds: SIGPIPE, SIGTERM, a normal exit
// 141 (on Windows only the normal exit applies).
func TestR155ExitKindLines(t *testing.T) {
	type c struct {
		mode  string
		bytes int
		want  string
	}
	cases := []c{{pseudo.FloodExit, 8192, "exited 141"}}
	if runtime.GOOS != "windows" {
		cases = append(cases, c{pseudo.FloodTerm, 8192, "signaled SIGTERM"}, c{pseudo.Flood, 1 << 20, "signaled SIGPIPE"})
	}
	seen := map[string]bool{}
	for _, x := range cases {
		k := lc9CaseNamed(t, "overflow")
		k.mode, k.bytes = x.mode, x.bytes
		o, _ := runJudged(t, k)
		if !slices.Contains(o.Lines, "wait "+x.want) || !slices.Contains(o.Lines, "fault wait operation exit-status "+x.want) {
			t.Errorf("%s: lines %q, want the end %s", x.mode, o.Lines, x.want)
		}
		seen[x.want] = true
	}
	if len(seen) != len(cases) {
		t.Errorf("the ends are not told apart: %v", seen)
	}
}
