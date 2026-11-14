package job

import (
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/controls"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

// Review 148 regressions that need the published state added by the fix
// (the verification record, LC-1's qualification, the refused outer
// operations and the held directories). They could not compile before the
// fix; the event orders are those of regress148_test.go.

func hasSub(xs []string, sub string) bool {
	for _, x := range xs {
		if strings.Contains(x, sub) {
			return true
		}
	}
	return false
}

// R-a: the healthy verification covers every record the job holds, and its
// coverage, result and qualification are published.
func TestB148RaPublishedVerificationCoversEveryRecord(t *testing.T) {
	j, _, _, g := newJob148(t, controls.LC3(), nil)
	settle(t, j, g.freeAll)
	j.RunDomains()
	p := j.Snapshot().Prov
	items := j.rec.Items()
	if p == nil || p.State != ProvDone || !p.Timely || len(p.Problems) != 0 {
		t.Fatalf("verification %+v", p)
	}
	if p.Covered != len(items) || p.Last != items[len(items)-1].Ref {
		t.Errorf("covered %d (last %v) of %d items (last %v)", p.Covered, p.Last, len(items), items[len(items)-1].Ref)
	}
}

// R-a: a verification still running at the snapshot is pending: the parent
// does not run it and does not treat it as success.
func TestB148RaStuckVerificationIsPending(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	v := newVerifier(clk, true)
	j.s.verify = v.verify
	settle(t, j, v.free, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, v.entered, "the work task entered the verification") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	if p := j.Snapshot().Prov; p == nil || p.State != ProvRunning {
		t.Fatalf("verification %+v", p)
	}
	st := j.copyState(0)
	if !hasSub(st.pending, "source-chain verification") {
		t.Errorf("pending %v", st.pending)
	}
	if f, ok := finish(t, j); ok && f.Conclusion != chain.Incomplete {
		t.Errorf("final %v %v", f.Conclusion, f.Reasons)
	}
}

// R-a: a success published after W is held as late and consumed as such.
func TestB148RaLateSuccessIsNotTimely(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	v := newVerifier(clk, true)
	j.s.verify = v.verify
	settle(t, j, v.free, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, v.entered, "the work task entered the verification") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	clk.Set(j.W.Add(time.Second))
	v.free()
	if !waitFor(t, j.WorkDone(), "the work task finished") {
		return
	}
	p := j.Snapshot().Prov
	if p == nil || p.State != ProvDone || p.Timely || len(p.Problems) != 0 {
		t.Fatalf("verification %+v", p)
	}
	if st := j.copyState(0); !hasSub(st.pending, "source-chain verification published at or after W") {
		t.Errorf("pending %v", st.pending)
	}
}

// R-a: a real verification error published after the parent decided stays
// held by the job.
func TestB148RaLateVerificationErrorHeldAfterFinish(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	v := newVerifier(clk, true, "injected verification problem")
	j.s.verify = v.verify
	settle(t, j, v.free, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, v.entered, "the work task entered the verification") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	f, ok := finish(t, j)
	if !ok {
		return
	}
	if !notValid(f) {
		t.Fatalf("final %v exit %d", f.Conclusion, f.ExitCode)
	}
	clk.Set(j.W.Add(time.Second))
	v.free()
	if !waitFor(t, j.WorkDone(), "the work task finished") {
		return
	}
	p := j.Snapshot().Prov
	if p == nil || p.State != ProvDone || p.Timely || !hasSub(p.Problems, "injected verification problem") {
		t.Errorf("verification %+v", p)
	}
}

// R-a prefix rule: a record appended after the verification is not covered
// by it; the old success is not reused and the job is invalid.
func TestB148RaAppendedAfterTheVerificationIsADefect(t *testing.T) {
	j, _, _, g := newJob148(t, controls.LC3(), nil)
	settle(t, j, g.freeAll)
	j.RunDomains()
	j.rec.Derived(spec.ConfigsFor(runtime.GOOS)[0].String()+"/closure", "B3", nil, map[string]any{"closure": []string{"forged"}})
	if p := j.Snapshot().Prov; p == nil || len(p.Problems) != 0 {
		t.Fatalf("the verification itself saw no problem: %+v", p)
	}
	if st := j.copyState(0); !hasSub(st.in.Defects, "after the source-chain verification") {
		t.Errorf("defects %v", st.in.Defects)
	}
	if f, ok := finish(t, j); ok && f.Conclusion != chain.Invalid {
		t.Errorf("final %v %v", f.Conclusion, f.Reasons)
	}
}

// R-a: a broken record appended inside the work task before the
// verification is found and covered by it.
func TestB148RaBrokenRecordFoundByTheWork(t *testing.T) {
	j, _, _, g := newJob148(t, controls.LC3(), nil)
	x := g.add("R done")
	x.do = func() {
		j.rec.Derived(spec.ConfigsFor(runtime.GOOS)[0].String()+"/closure", "B3", nil, map[string]any{"closure": []string{"forged"}})
	}
	x.free()
	settle(t, j, g.freeAll)
	j.RunDomains()
	p := j.Snapshot().Prov
	if p == nil || p.Covered != len(j.rec.Items()) || !hasSub(p.Problems, "closure: no parent") {
		t.Errorf("verification %+v", p)
	}
}

// R-b: in the Review 148 window, LC-1 is published after W (its own
// qualification is late) and the work task's abandonment stays consumed
// after it published done.
func TestB148RbLC1AndWorkQualification(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	last := g.add("R published " + lastConfig())
	settle(t, j, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, last.reached, "the last R result was published") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	clk.Set(j.W.Add(time.Second))
	last.free()
	if !waitFor(t, j.WorkDone(), "the work task completed after W") {
		return
	}
	s := j.Snapshot()
	if s.LC1 == nil || s.LC1.Timely || s.LC1.Variant.Judgement != verdict.Hit {
		t.Fatalf("LC-1 record %+v", s.LC1)
	}
	st := j.copyState(0)
	if v := lc1Of(st.in); v == nil || v.Judgement != verdict.Pending || !hasSub(v.Reasons, "published at or after W") {
		t.Errorf("LC-1 consumed as %+v", v)
	}
	if !hasSub(st.pending, "after the parent stopped waiting") {
		t.Errorf("pending %v", st.pending)
	}
	if f, ok := finish(t, j); ok && f.Conclusion != chain.Incomplete {
		t.Errorf("final %v %v", f.Conclusion, f.Reasons)
	}
}

// R-c: refusals are recorded apart from failures. The work task scheduled
// after D refuses the fixture root's preparation; the F records say "not
// admitted", never a success.
func TestB148RcRefusedPreparationIsRecordedApart(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	start := g.add("work start")
	settle(t, j, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, start.reached, "the work task was scheduled") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	clk.Set(j.D)
	start.free()
	if !waitFor(t, j.WorkDone(), "the work task finished") {
		return
	}
	root := opsNamed(j.outer.Snapshot(), "prepare fixture root")
	if len(root) != 1 || root[0].State != chain.OpRefused || root[0].Outcome == nil || root[0].Outcome.Err != nil || !strings.Contains(root[0].Reason, "not admitted") {
		t.Errorf("root preparation %+v", root)
	}
	s := j.Snapshot()
	if len(s.Held) != 0 {
		t.Errorf("held %+v: nothing was created", s.Held)
	}
	for _, r := range s.FRecords {
		if !errors.Is(r.Outcome.PrepErr, controls.ErrNotAdmitted) || r.Variant.Judgement != verdict.Miss || !hasSub(r.Variant.Reasons, "not admitted") {
			t.Errorf("%s %s: %+v", r.Variant.Control, r.Variant.Variant, r)
		}
	}
}

// R-c: an instance returning after D: its cleanup and the root's are
// refused and recorded as refused; both directories stay held, not removed.
func TestB148RcRefusedCleanupKeepsTheHold(t *testing.T) {
	tc := newBlockingTC()
	j, clk, _, g := newJob148(t, lc5Then3(), tc)
	settle(t, j, tc.free, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, tc.entered, "the LC-5 instance entered the type check") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	clk.Set(j.D)
	tc.free()
	if !waitFor(t, j.WorkDone(), "the work task finished") {
		return
	}
	o := j.outer.Snapshot()
	lc5 := controls.LC5()[0].Name
	for _, name := range []string{"cleanup LC-5 " + lc5, "cleanup fixture root"} {
		if xs := opsNamed(o, name); len(xs) != 1 || xs[0].State != chain.OpRefused || !strings.Contains(xs[0].Reason, "not admitted") {
			t.Errorf("%s: %+v", name, xs)
		}
	}
	s := j.Snapshot()
	if len(s.Held) != 2 {
		t.Fatalf("held %+v, want the fixture root and the LC-5 fixture", s.Held)
	}
	for _, h := range s.Held {
		if h.Removed || !strings.Contains(h.State, "not admitted") {
			t.Errorf("held %+v", h)
		}
	}
	for _, r := range s.FRecords {
		if r.Variant.Variant == lc5 && (!errors.Is(r.Outcome.CleanupErr, controls.ErrNotAdmitted) || r.Variant.Judgement == verdict.Hit) {
			t.Errorf("LC-5 record %+v", r)
		}
	}
	if st := j.copyState(0); len(st.held) != 2 {
		t.Errorf("the held directories are not in the copied state: %+v", st.held)
	}
}

// R-c: a real failure stays a failure, distinct from a refusal.
func TestB148RcFailureIsNotARefusal(t *testing.T) {
	vs := controls.LC3()
	j, _, ops, g := newJob148(t, vs, nil)
	ops.failRoot = true
	settle(t, j, g.freeAll)
	j.RunDomains()
	root := opsNamed(j.outer.Snapshot(), "cleanup fixture root")
	if len(root) != 1 || root[0].State != chain.OpFailed {
		t.Errorf("root cleanup %+v", root)
	}
	for _, h := range j.Snapshot().Held {
		if h.Name == "fixture root" && (h.Removed || !strings.Contains(h.State, "cleanup failed: injected remove failure")) {
			t.Errorf("held %+v", h)
		}
	}
}

// R-c healthy: every held directory is released by a confirmed removal.
func TestB148RcHealthyHoldsReleased(t *testing.T) {
	j, _, _, g := newJob148(t, controls.LC3(), nil)
	settle(t, j, g.freeAll)
	j.RunDomains()
	s := j.Snapshot()
	if len(s.Held) != 5 {
		t.Fatalf("held %+v, want the root and four fixtures", s.Held)
	}
	for _, h := range s.Held {
		if !h.Removed || h.State != "removed" {
			t.Errorf("held %+v", h)
		}
	}
}
