//go:build darwin

package wp25

// P-5 (docs/wp2.5-prototype.md §16, the fourth draft; its implementation
// registered in §16.14 before this code): entry E2, the dynamically
// imported waitid of libSystem called through syscall.syscall6
// (e2waitid), on arm64 and x86_64. TestP5 runs, within one budget D:
//
//   - P5.static.entry: the source check that entry E2 is the only waitid
//     entry and that its one caller is waitidCall (§16.4);
//   - P5.layout three times, P-3's layout probe, the premise of the rest;
//   - 39 native runs (§16.5), each call of entry E2 kept whole and
//     classified once (§16.7);
//   - two gates (§16.6), whose inner runs TestP5 starts again in a new
//     process (role B) and waits for by the fourth draft's protocol
//     (p5gate_darwin_test.go, p5proto_test.go).
//
// The roles (§16.8): A, the runs of TestP5 in the outer test process,
// under D; B, the two runs of a gate's inner run, under the d_i the gate
// passes in WP25_P5_DEADLINE; B′, the gates themselves, under D. P-3's own
// runs (role C) keep their rules.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

const (
	p5Total    = 180 * time.Second // D: S plus this, no later than T less p5Margin
	p5Margin   = 30 * time.Second  // D before T; the final cleanup after D
	p5Window   = 60 * time.Second  // a role A run's window and admission; a gate's
	p5InnerWin = 12 * time.Second  // a role B run's
	p5InnerObs = 6 * time.Second   // its observation
	p5Every    = 5 * time.Millisecond

	p5DeadlineEnv = "WP25_P5_DEADLINE" // d_i, Unix nanoseconds, from the gate to its inner run
	p5StaticEnv   = "WP25_P5_STATIC"   // the workflow's file of the static steps' conclusions
	p5Expected    = 39
)

// The two limitation statements, verbatim (§16.11).
const (
	p5H4 = "H4：静态依据是固定的 Go 1.27.3 过滤源码；动态覆盖的是所测的真实错误（`ECHILD`、`EINVAL`）与错误之后的健康恢复；没有动态取样的是健康调用时过滤之前的 errno 是否非零。"
	p5H6 = "H6：只有接缝证据（消费者的不重试与失败）；真实的 EINTR 穿过跳板没有原生验证。"
)

// p5 is this test process's P-5 state.
var p5 struct {
	mu      sync.Mutex
	ran     bool   // TestP5 began (P5.static.load)
	roleErr string // the outer has WP25_P5_DEADLINE: TestP5 does not run
	role    string
	S, T, D time.Time
	di      time.Time
	gates   []gateResult
	handed  []*p5Hand // what the gates handed on to the final cleanup
	final   string    // the final cleanup's account
	finalT  bool      // it left something unconfirmed (a T)
}

// p5Layouts are the verdicts of the P-5 layout probes of this process (P-3
// keeps its own), under p3LayoutMu.
var p5Layouts []verdict

// p5Ctx is one P-5 run's role, deadlines and calls.
type p5Ctx struct {
	role     string    // A or B
	D        time.Time // A: the total budget's end
	di       time.Time // B: the gate's deadline for its inner run
	roleErr  string    // B: why that deadline is missing, unreadable or past
	d        time.Time // the run's own deadline, set by admit
	plan     int       // the planned calls of entry E2
	rule     int       // the calls the consumer's rules made
	evidence string
}

// admit runs the scenario only within its role's budget, otherwise it is
// not executed, which is invalid (§16.8). Not the P-3 budget, not the 120 s
// to go test's deadline.
func (x *p5Ctx) admit(s *scen) {
	now := time.Now()
	switch x.role {
	case "A":
		if left := x.D.Sub(now); left < p5Window {
			s.invalidf("not executed: D is %v away, less than one run's %v", left.Round(time.Millisecond), p5Window)
		}
		x.d = minTime(now.Add(p5Window), x.D)
		s.cleanLimit = p3Clean
	case "B":
		if x.roleErr != "" {
			s.invalidf("not executed: %s", x.roleErr)
		}
		if left := x.di.Sub(now); left < p5InnerWin {
			s.invalidf("not executed: the inner run's deadline d_i is %v away, less than one run's %v", left.Round(time.Millisecond), p5InnerWin)
		}
		x.d = minTime(now.Add(p5InnerWin), x.di)
	}
	s.cap = x.d
}

// obsEnd is the end of a run's observation: role A min(start + 20 s,
// d - 30 s); role B min(start + 6 s, d - 6 s). Neither looks at the test's
// own deadline.
func (x *p5Ctx) obsEnd(began time.Time) time.Time {
	if x.role == "B" {
		return minTime(began.Add(p5InnerObs), x.d.Add(-6*time.Second))
	}
	return minTime(began.Add(p3Inner), x.d.Add(-30*time.Second))
}

// final runs at the run's end, after its cleanup and before its verdict:
// the calls, read now (§16.11), against the rule and the plan (§16.7);
// and the run's own deadline: one that ends after d is invalid (§16.8).
func (x *p5Ctx) final(s *scen) {
	n := int(e2Calls.Load())
	s.r.Calls = &n
	ok := len(s.invalid) == 0 && len(s.mismatch) == 0 && s.unsup == "" && !s.t.Failed() && len(s.want) > 0
	if why := p5CallsCheck(n, x.rule, x.plan, ok); why != "" {
		s.flag("%s", why)
	}
	if !x.d.IsZero() && time.Now().After(x.d) {
		s.flag("the run ended at %s, after its deadline d %s", time.Now().Format(time.RFC3339Nano), x.d.Format(time.RFC3339Nano))
	}
}

func minTime(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

// e2Begin starts a run's count and log of the calls of entry E2; e2End
// stops the log and gives it.
func e2Begin() {
	e2Calls.Store(0)
	e2Log.Lock()
	e2Log.on, e2Log.calls = true, nil
	e2Log.Unlock()
}

func e2Snapshot() []e2Call {
	e2Log.Lock()
	defer e2Log.Unlock()
	return append([]e2Call(nil), e2Log.calls...)
}

func e2End() {
	e2Log.Lock()
	e2Log.on, e2Log.calls = false, nil
	e2Log.Unlock()
}

// e2Last is the current run's last call.
func e2Last() (e2Call, bool) {
	e2Log.Lock()
	defer e2Log.Unlock()
	if n := len(e2Log.calls); n > 0 {
		return e2Log.calls[n-1], true
	}
	return e2Call{}, false
}

// p5RoleOf reads the role (§16.8): A without innerEnv, B with it. The
// outer with WP25_P5_DEADLINE is a T; an inner run whose deadline is
// missing, unreadable or past gets B with the reason, and its runs are not
// executed.
func p5RoleOf() (role string, di time.Time, why string) {
	v := os.Getenv(p5DeadlineEnv)
	if os.Getenv(innerEnv) == "" {
		if v != "" {
			return "", time.Time{}, "the outer test process has " + p5DeadlineEnv + " in its environment"
		}
		return "A", time.Time{}, ""
	}
	n, err := strconv.ParseInt(v, 10, 64)
	switch {
	case v == "":
		return "B", time.Time{}, p5DeadlineEnv + " is missing"
	case err != nil:
		return "B", time.Time{}, p5DeadlineEnv + " cannot be read: " + err.Error()
	case !time.Unix(0, n).After(time.Now()):
		return "B", time.Unix(0, n), p5DeadlineEnv + " has passed"
	}
	return "B", time.Unix(0, n), ""
}

type p5Case struct {
	name string
	plan int // the planned calls of entry E2 (§16.7)
	run  func(*p3Run)
}

// p5Cases are §16.5's scenarios, in its order, the layout probe first.
func p5Cases() []p5Case {
	cs := []p5Case{{"layout", 0, p3Layout}, {"w.exit7", 2, p5W(7)}}
	for _, x := range p3Exits {
		cs = append(cs, p5Case{"w.exit-" + x.name, 2, p5W(x.code)})
	}
	return append(cs,
		p5Case{"w.signal9", 2, p5WSignal},
		p5Case{"c.exit7", 2, p5C(7)}, p5Case{"c.exit-107", 2, p5C(0x107)}, p5Case{"c.exit-ff", 2, p5C(-1)},
		p5Case{"k.esrch", 2, p5KEsrch},
		p5Case{"err.echild", 3, p5Err(unix.ECHILD)}, p5Case{"err.einval", 3, p5Err(unix.EINVAL)})
}

// TestP5 is P-5 in either role.
func TestP5(t *testing.T) {
	S := time.Now()
	role, di, why := p5RoleOf()
	if role == "" {
		p5.mu.Lock()
		p5.roleErr = why
		p5.mu.Unlock()
		t.Fatalf("P-5: %s: a T; TestP5 does not run", why)
	}
	ctx := func(plan int, evidence string) *p5Ctx {
		p5.mu.Lock()
		defer p5.mu.Unlock()
		return &p5Ctx{role: role, D: p5.D, di: di, roleErr: why, plan: plan, evidence: evidence}
	}
	p5.mu.Lock()
	p5.ran, p5.role, p5.S, p5.di = role == "A", role, S, di
	p5.mu.Unlock()
	if role == "A" {
		T := S.Add(100 * 365 * 24 * time.Hour)
		if dl, ok := t.Deadline(); ok {
			T = dl
		}
		D := minTime(S.Add(p5Total), T.Add(-p5Margin))
		p5.mu.Lock()
		p5.T, p5.D = T, D
		p5.mu.Unlock()
		p5Meta(S, T, D)
		// The final cleanup, registered at S: after every run of TestP5.
		t.Cleanup(func() { p5FinalCleanup(t, minTime(D.Add(p5Margin), T)) })
		t.Run("static.entry", func(t *testing.T) { runP5Static(t, ctx(0, "static")) })
	} else {
		t.Cleanup(func() { p5FinalCleanup(t, di.Add(3*time.Second)) })
	}
	for _, c := range p5Cases() {
		for n := 1; n <= 3; n++ {
			t.Run(fmt.Sprintf("%s.%d", c.name, n), func(t *testing.T) {
				ev := "native"
				switch {
				case c.name == "layout":
					ev = "layout"
				case role == "B":
					ev = "seam"
				}
				runP5(t, c, n, ctx(c.plan, ev))
			})
		}
	}
	if role == "A" {
		for _, c := range p5Gates {
			t.Run(c.id, func(t *testing.T) { checkP5Gate(t, c) })
		}
	}
}

// p5Meta prints the P-5 metadata of this job (§16.10): the toolchain, its
// environment and the build settings this test binary carries.
func p5Meta(S, T, D time.Time) {
	m := map[string]any{
		"go": runtime.Version(), "goos": runtime.GOOS, "goarch": runtime.GOARCH,
		"CGO_ENABLED": envOr("CGO_ENABLED", "(unset)"), "GOTOOLCHAIN": envOr("GOTOOLCHAIN", "(unset)"),
		"S": S.Format(time.RFC3339Nano), "T": T.Format(time.RFC3339Nano), "D": D.Format(time.RFC3339Nano),
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		settings := map[string]string{}
		for _, s := range bi.Settings {
			settings[s.Key] = s.Value
		}
		m["build"] = settings
		m["build_go"] = bi.GoVersion
	} else {
		m["build"] = "unknown: no build information"
	}
	b, _ := json.Marshal(m)
	fmt.Println("WP25P5META " + string(b))
}

// runP5 is one P-5 run (roles A and B). Its evidence line comes first, then
// its end (§19.12), as in P-3.
func runP5(t *testing.T, c p5Case, n int, x *p5Ctx) {
	id := fmt.Sprintf("P5.%s.%d", c.name, n)
	s := begin(t, "G5", id, normal, true, true)
	s.r.Evidence = x.evidence
	r := &p3Run{s: s, t: t, began: time.Now(), ev: p3Evidence{E: map[string]string{}}, p5: x}
	e2Begin()
	s.final = func() { x.final(s) }
	defer func() {
		r.limits()
		r.ev.E2 = e2Snapshot()
		s.locked(func() { r.ev.T = append([]string(nil), s.invalid...) })
		pline(id, r.ev)
		s.done()
		e2End()
		p5Seam.eintr = false
		if c.name == "layout" {
			p3LayoutMu.Lock()
			p5Layouts = append(p5Layouts, s.r.Verdict)
			p3LayoutMu.Unlock()
		}
	}()
	r.admit()
	r.environment()
	if c.name != "layout" {
		r.layoutPremise()
	}
	// The gate's seam (§16.6), in its inner run's target only.
	if x.role == "B" && c.name == "w.exit7" && n == 1 && injected("p5-eintr-once") {
		r.seam = "p5-eintr-once"
		r.ev.F = append(r.ev.F, "planned fault p5-eintr-once: the first real success of waitid is replaced by (-1, EINTR) after the call")
		x.plan = 1
		p5Seam.eintr = true
	}
	c.run(r)
}

// e2Read is one healthy read of pid through entry E2, as the consumer
// makes it, judged by e2Judge. It says whether the read gave X1: the
// consumer goes on only then (§16.3, §16.7).
func (r *p3Run) e2Read(pid int, what string, recovery bool) (wRaw, bool) {
	r.p5.rule++
	w, err := readWaitid(pid)
	return w, r.e2Judge(what, w, err, recovery)
}

// e2Judge classifies the call of entry E2 just made for a healthy read
// (§16.7) and records it (R, and its class in the call's record). err is
// the read's own error, nil when it gave its fields. A protocol failure or
// the seam's real failure is a T, kept to the verdict; a healthy X2 without
// the seam is a native counterexample, and the consumer's observation
// failure that follows is its correct handling, recorded in M; the seam's
// replaced (-1, EINTR) is not classified, and the consumer rejects it by
// §19.5's error exit, the one reason the gate expects.
func (r *p3Run) e2Judge(what string, w wRaw, err error, recovery bool) bool {
	c, ok := e2Last()
	if !ok {
		r.s.flag("calls: %s made no call of entry E2", what)
		return false
	}
	seam := c.Seam != ""
	class, why := e2Class(c.Form, c.R, syscall.Errno(c.Errno), 0, seam)
	if recovery && class == e2Native {
		why = "the healthy recovery after the error failed: " + why
	}
	e2Note(func(x *e2Call) { x.Class, x.Why = class, why })
	rd := p3Read{What: what}
	if err != nil {
		rd.Err = err.Error()
	} else {
		rd.W = &w
	}
	r.ev.R = append(r.ev.R, rd)
	switch class {
	case e2Tool:
		r.s.flag("%s: %s", what, why)
	case e2Native:
		r.s.expect(what+": the call's r / errno / form", "0 / 0 / X1", fmt.Sprintf("%d / %d / %s", c.R, c.Errno, c.Form))
	}
	if err == nil {
		return true
	}
	if seam {
		r.s.flag("observation failed: %s: %v", what, err)
	} else {
		r.ev.M = append(r.ev.M, what+": an observation failure, §19.5's error exit: "+err.Error())
	}
	return false
}

// p5Waitid2 is the consumer's two reads of an unreaped zombie through entry
// E2; a first read that is not X1 ends it (§16.3).
func (r *p3Run) p5Waitid2(tr *unixTree, code, status uint32, want string, recovery bool) {
	a, ok := r.e2Read(tr.pid, "waitid 1", recovery)
	if !ok {
		r.mark()
		r.s.worked(1)
		return
	}
	b, ok := r.e2Read(tr.pid, "waitid 2", recovery)
	r.mark()
	r.s.worked(1)
	if !ok {
		return
	}
	e, err := decodeW2(a, b, tr.pid)
	r.answer("waitid, two reads", &e, err)
	r.predictW(a, &b, tr.pid, code, status)
	r.normal("waitid", p3NormW(a, tr.pid), want, &e, err)
}

// p5W: P-3's p3W, with the consumer's reads through entry E2.
func p5W(code int) func(*p3Run) {
	return func(r *p3Run) {
		pr := p3Pred[code]
		cmd := helper("exit-after", nil, "0", strconv.Itoa(code))
		tr := startGroup(r.s, cmd)
		r.zombie(tr)
		r.p5Waitid2(tr, cldExited, pr.w, fmt.Sprintf("exit %d", pr.n), false)
		r.reap(tr, cmd, fmt.Sprintf("exit %d", pr.code))
	}
}

// p5WSignal: P-3's p3WSignal, the same way.
func p5WSignal(r *p3Run) {
	cmd := helper("sleep", nil)
	tr := startGroup(r.s, cmd)
	if err := tr.kill(); err != nil {
		r.s.invalidf("load not established: kill(-pgid): %v", err)
	}
	r.zombie(tr)
	r.p5Waitid2(tr, cldKilled, 9, "signal 9", false)
	r.reap(tr, cmd, "signal 9")
}

// p5C: the kqueue event, then the consumer's two reads of the same
// unreaped zombie through entry E2, and C1 between the event's end and the
// two reads' (decodeW2). P-3's p3C reads once and stays as it is.
func p5C(code int) func(*p3Run) {
	return func(r *p3Run) {
		pr := p3Pred[code]
		cmd := helper("exit-after", nil, "200", strconv.Itoa(code))
		tr := startGroup(r.s, cmd)
		obs := r.observer(tr)
		r.s.needLoad(!obs.s.early, "the kqueue registered while the process ran")
		_, _, kerr := obs.wait(r.capped(3 * time.Second))
		k := r.kread(obs, kerr)
		r.answer("kqueue", obs.s.end, kerr)
		r.zombie(tr)
		a, ok := r.e2Read(tr.pid, "waitid 1", false)
		var b wRaw
		if ok {
			b, ok = r.e2Read(tr.pid, "waitid 2", false)
		}
		r.mark()
		r.s.worked(1)
		want := fmt.Sprintf("exit %d", pr.n)
		r.predictK(k, pr.k)
		r.normal("kqueue", p3NormK(*k, tr.pid), want, obs.s.end, kerr)
		if ok {
			we, werr := decodeW2(a, b, tr.pid)
			r.answer("waitid, two reads", &we, werr)
			same := "not compared (a path did not give an end)"
			if kerr == nil && werr == nil && obs.s.end != nil {
				if cerr := sameEnd(*obs.s.end, we); cerr != nil {
					same = "different"
					r.ev.M = append(r.ev.M, "C1: rejected, "+cerr.Error()+" (an observation failure: C-19, F1)")
				} else {
					same = "the same"
					r.ev.M = append(r.ev.M, "C1: the same end")
				}
			}
			r.predictW(a, &b, tr.pid, cldExited, pr.w)
			r.normal("waitid", p3NormW(a, tr.pid), want, &we, werr)
			r.predict("C1: the two ends", "the same", same)
		}
		r.reap(tr, cmd, fmt.Sprintf("exit %d", pr.code))
	}
}

// p5KEsrch: P-3's p3KEsrch: the registration after the exit; the
// candidate's early path makes the first read, through entry E2, and the
// consumer's second read follows only if it gave X1.
func p5KEsrch(r *p3Run) {
	cmd := helper("exit-after", nil, "0", "7")
	tr := startGroup(r.s, cmd)
	r.zombie(tr)
	obs := r.observer(tr)
	r.ev.R = append(r.ev.R, p3Read{What: "kevent registration", Err: errText(obs.s.regErr)})
	r.predict("the registration of an unreaped zombie", "no such process", errText(obs.s.regErr))
	if !obs.s.early {
		r.ev.M = append(r.ev.M, "the candidate did not take its early path (registered: "+errText(obs.s.regErr)+")")
		r.mark()
		r.s.worked(1)
		r.reap(tr, cmd, "exit 7")
		return
	}
	r.p5.rule++
	_, _, err := obs.wait(r.capped(3 * time.Second))
	// The early path's read gave its fields (lastW) or failed; an error
	// with fields is the decoder's, its answer.
	var a wRaw
	var readErr error
	if obs.s.lastW != nil {
		a = *obs.s.lastW
	} else {
		readErr = err
	}
	if r.e2Judge("waitid 1 (the candidate's)", a, readErr, false) {
		b, ok := r.e2Read(tr.pid, "waitid 2", false)
		r.answer("waitid (the candidate's early path)", obs.s.end, err)
		r.mark()
		r.s.worked(1)
		if ok {
			r.predictW(a, &b, tr.pid, cldExited, 0x7)
			r.normal("waitid", p3NormW(a, tr.pid), "exit 7", obs.s.end, err)
		}
	} else {
		r.mark()
		r.s.worked(1)
	}
	r.reap(tr, cmd, "exit 7")
}

// p5Err: a constructed error through waitidCall, then the consumer's two
// healthy reads of a controlled zombie, all on one locked OS thread
// (§16.5). The thread only gives a thread-local context: it is no proof
// that a residual errno was set (§21.9).
func p5Err(want syscall.Errno) func(*p3Run) {
	return func(r *p3Run) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		cmd := helper("exit-after", nil, "0", "7")
		tr := startGroup(r.s, cmd)
		r.zombie(tr)
		r.p5ErrCall(tr, want)
		r.p5Waitid2(tr, cldExited, 0x7, "exit 7", true)
		r.reap(tr, cmd, "exit 7")
	}
}

// p5ErrCall is P5.err's constructed error, through waitidCall: ECHILD by
// asking for this process itself as its own child (kern_exit.c:3501 to
// 3503), EINVAL by options 0 (kern_exit.c:3316 to 3318, before any child
// is looked at). Its buffer is kept raw (the call's record) and never
// decoded. It neither acts on another process nor changes the child.
func (r *p3Run) p5ErrCall(tr *unixTree, want syscall.Errno) {
	id, opts := uint32(os.Getpid()), uint32(unix.WEXITED|unix.WNOHANG|unix.WNOWAIT)
	if want == unix.EINVAL {
		id, opts = uint32(tr.pid), 0
	}
	r.p5.rule++
	ret, e, _, _ := waitidCall(pPID, id, opts)
	form := e2Form(ret, e)
	class, why := e2Class(form, ret, e, want, false)
	e2Note(func(x *e2Call) { x.Class, x.Why = class, why })
	what := fmt.Sprintf("the constructed error waitid(P_PID, %d, %#x)", id, opts)
	r.ev.R = append(r.ev.R, p3Read{What: what, Err: fmt.Sprintf("r %d, errno %d (%v), %s: %s", ret, int(e), e, form, why)})
	if class == e2Tool {
		r.s.flag("%s: %s", what, why)
		return
	}
	r.s.expect(what+": r / errno / form", fmt.Sprintf("-1 / %d / X2", int(want)), fmt.Sprintf("%d / %d / %s", ret, int(e), form))
}

// runP5Static is P5.static.entry (§16.4): the module's Go sources parsed,
// its assembly read as text, and the four statements checked.
func runP5Static(t *testing.T, x *p5Ctx) {
	s := begin(t, "G5", "P5.static.entry", normal, false, true)
	s.r.Evidence = "static"
	e2Begin()
	s.final = func() { x.final(s) }
	defer func() {
		s.done()
		e2End()
	}()
	x.admit(s)
	st, err := p5Sources(".")
	if err != nil {
		s.invalidf("the sources: %v", err)
	}
	if st.files < 20 {
		s.invalidf("the sources: only %d files read (a copied test binary?)", st.files)
	}
	s.worked(int64(st.files))
	s.note("files read: %d", st.files)
	in := func(dir string, files []string) bool {
		for _, f := range files {
			if filepath.Dir(f) != dir {
				return false
			}
		}
		return true
	}
	s.expectTrue("1. the dynamic import's directive only in e2waitid", len(st.directive) > 0 && in("e2waitid", st.directive), strings.Join(st.directive, ", "))
	s.expectTrue("1. the symbol libc_waitid only in e2waitid", len(st.symbol) > 0 && in("e2waitid", st.symbol), strings.Join(st.symbol, ", "))
	s.expectTrue("2. no Syscall or RawSyscall of unix or syscall with SYS_WAITID", len(st.raw) == 0, "found: "+strings.Join(st.raw, ", "))
	s.expectTrue("3. e2waitid.Waitid called only by waitidCall", len(st.waitid) > 0 && allEqual(st.waitid, "waitidCall"), "callers: "+strings.Join(st.waitid, ", "))
	s.expectTrue("4. readWaitid calls waitidCall", st.through["readWaitid"], "")
	s.expectTrue("4. P5.err's call (p5ErrCall) calls waitidCall", st.through["p5ErrCall"], "")
}

func allEqual(xs []string, v string) bool {
	for _, x := range xs {
		if x != v {
			return false
		}
	}
	return true
}

// p5Static is what the source check found.
type p5Static struct {
	files     int
	directive []string        // files with the dynamic import's directive
	symbol    []string        // files with an identifier or text libc_waitid
	raw       []string        // raw waitid system calls: file:line
	waitid    []string        // the functions calling e2waitid.Waitid
	through   map[string]bool // readWaitid, p5ErrCall: they call waitidCall
}

// p5Sources walks the module's sources from dir.
func p5Sources(dir string) (p5Static, error) {
	st := p5Static{through: map[string]bool{}}
	fset := token.NewFileSet()
	add := func(list *[]string, f string) {
		if len(*list) == 0 || (*list)[len(*list)-1] != f {
			*list = append(*list, f)
		}
	}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := filepath.ToSlash(filepath.Clean(path))
		switch filepath.Ext(path) {
		case ".s":
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			st.files++
			if strings.Contains(string(b), "cgo_import_dynamic") {
				add(&st.directive, rel)
			}
			if strings.Contains(string(b), "libc_waitid") {
				add(&st.symbol, rel)
			}
		case ".go":
			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			st.files++
			for _, cg := range f.Comments {
				for _, c := range cg.List {
					if strings.HasPrefix(c.Text, "//go:cgo_import_dynamic") {
						add(&st.directive, rel)
					}
				}
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && strings.Contains(id.Name, "libc_waitid") {
					add(&st.symbol, rel)
				}
				return true
			})
			for _, decl := range f.Decls {
				fn := ""
				if fd, ok := decl.(*ast.FuncDecl); ok {
					fn = fd.Name.Name
				}
				ast.Inspect(decl, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					switch fun := call.Fun.(type) {
					case *ast.SelectorExpr:
						pkg, _ := fun.X.(*ast.Ident)
						switch {
						case pkg != nil && pkg.Name == "e2waitid" && fun.Sel.Name == "Waitid":
							st.waitid = append(st.waitid, fn)
						case pkg != nil && (pkg.Name == "unix" || pkg.Name == "syscall") &&
							(strings.HasPrefix(fun.Sel.Name, "Syscall") || strings.HasPrefix(fun.Sel.Name, "RawSyscall")) && mentions(call.Args, "SYS_WAITID"):
							st.raw = append(st.raw, fmt.Sprintf("%s:%d", rel, fset.Position(call.Pos()).Line))
						}
					case *ast.Ident:
						if fun.Name == "waitidCall" && (fn == "readWaitid" || fn == "p5ErrCall") {
							st.through[fn] = true
						}
						if fun.Name == "Waitid" && f.Name.Name == "e2waitid" {
							st.waitid = append(st.waitid, "e2waitid."+fn)
						}
					}
					return true
				})
			}
		}
		return nil
	})
	return st, err
}

// mentions says whether any of the expressions names the identifier.
func mentions(args []ast.Expr, name string) bool {
	found := false
	for _, a := range args {
		ast.Inspect(a, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == name {
				found = true
			}
			return !found
		})
	}
	return found
}

// p5Hand is what one gate handed on to the final cleanup (§16.8): its
// inner run, if anything of it is not confirmed, and the processes the
// census proved this round's whose end was not confirmed.
type p5Hand struct {
	gate   string
	inner  *p5Inner
	tag    string
	since  string
	others []survivor
}

// p5FinalCleanup is the final cleanup (§16.8): one deadline shared by
// every handed item; each task only queried by it, never waited for again;
// a request task only by R0 (none while one is pending, none once reaping
// started); the census by each gate's tag, acting only on what it proves.
// What ends is a late end, which changes no gate; what does not is
// unknown, and P-5 is a T. TestP5 then returns, leaving any task running.
func p5FinalCleanup(t *testing.T, deadline time.Time) {
	p5.mu.Lock()
	handed := p5.handed
	p5.mu.Unlock()
	type item struct {
		Gate      string   `json:"gate"`
		Inner     *p5Know  `json:"inner,omitempty"`
		Outcome   []string `json:"outcome"`
		Unknown   []string `json:"unknown,omitempty"`
		Processes []string `json:"processes,omitempty"`
	}
	var items []item
	unknown := 0
	for _, h := range handed {
		it := item{Gate: h.gate}
		if p := h.inner; p != nil {
			p.request("final cleanup", "final")
			for _, q := range p.reqs {
				q.until(deadline)
			}
			ws := p.wait.until(deadline)
			cs := p.copy.until(deadline)
			stopDone := true
			if p.stop != nil {
				stopDone, _ = p.stop.until(deadline)
			}
			k := p.know()
			it.Inner = &k
			if ws.Done {
				it.Outcome = append(it.Outcome, fmt.Sprintf("a late end: %s, observed %s", ws.Exit.Status, ws.Seen.Format(time.RFC3339Nano)))
			} else {
				it.Unknown = append(it.Unknown, "the inner run's exit is not confirmed")
			}
			if q := p.pendingReq(); q != nil {
				it.Unknown = append(it.Unknown, "a request task is still pending ("+q.holder+"/"+q.step+")")
			}
			if !cs.Done {
				it.Unknown = append(it.Unknown, "the copy task has not returned")
			} else if p.stop != nil || len(cs.Buf) > 0 {
				it.Outcome = append(it.Outcome, fmt.Sprintf("the copy returned: %d bytes, error %v (not parsed into the gate)", len(cs.Buf), cs.Err))
			}
			if !stopDone {
				it.Unknown = append(it.Unknown, "the stop of the read end has not returned")
			}
		}
		if len(h.others) > 0 || h.tag != "" {
			left, _, err := p5EndProven(h.others, h.tag, h.since, deadline, h.inner)
			for _, v := range left {
				it.Processes = append(it.Processes, v.key())
			}
			if err != nil {
				it.Unknown = append(it.Unknown, "the processes of the round: "+err.Error())
			}
			if len(left) > 0 {
				it.Unknown = append(it.Unknown, fmt.Sprintf("%d processes of the round still alive", len(left)))
			}
		}
		unknown += len(it.Unknown)
		items = append(items, it)
	}
	acc := "nothing handed over"
	if len(handed) > 0 {
		acc = fmt.Sprintf("%d items handed over, %d unconfirmed", len(handed), unknown)
	}
	b, _ := json.Marshal(map[string]any{"deadline": deadline.Format(time.RFC3339Nano), "account": acc, "items": items})
	fmt.Println("WP25P5FINAL " + string(b))
	p5.mu.Lock()
	p5.final, p5.finalT = acc, unknown > 0
	p5.mu.Unlock()
	if unknown > 0 {
		t.Errorf("P-5: the final cleanup left %d items unconfirmed by its deadline: a T (§16.12)", unknown)
	}
}

// p5EndProven ends what is still alive of a round: the processes proven
// its own that still run (by their start time), and what a census of its
// tag finds, never the inner run itself (only its request tasks signal
// it), each only while its identity and the tag still hold; then it
// confirms their end by identity, within deadline. It gives what is still
// alive, how many it found alive, and a read or census that failed.
func p5EndProven(known []survivor, tag, since string, deadline time.Time, inner *p5Inner) ([]survivor, int, error) {
	alive := map[string]survivor{}
	notInner := func(v survivor) bool { return inner == nil || v.pid != inner.pid }
	for _, v := range known {
		if st, pr := procInfo(v.pid); notInner(v) && pr == present && st.start == v.start && !st.zombie {
			alive[v.key()] = v
		}
	}
	ss, cerr := censusTag(tag, since, deadline)
	for _, v := range ss {
		if notInner(v) {
			alive[v.key()] = v
		}
	}
	found := len(alive)
	for {
		var still []survivor
		var rerr error
		for _, v := range alive {
			if st, pr := procInfo(v.pid); pr == present && st.start == v.start && !st.zombie && tagged(v.pid, tag) {
				_ = syscall.Kill(v.pid, syscall.SIGKILL)
			}
			a, err := recordAlive(witnessRec{pid: v.pid, start: v.start})
			switch {
			case err != nil:
				rerr = err
			case a:
				still = append(still, v)
			}
		}
		if len(still) == 0 && rerr == nil || !time.Now().Before(deadline) {
			if rerr == nil {
				rerr = cerr
			}
			return still, found, rerr
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// p5Summary prints this job's WP25P5 line (§16.11), in the outer test
// process: the toolchain and cgo, the static checks one by one, the native
// confirmed count against 39, the gates, the model checks, the runs not
// executed, D, the final cleanup, and the H4 and H6 statements. It decides
// nothing.
func p5Summary() {
	p5.mu.Lock()
	ran, roleErr, D, final, finalT := p5.ran, p5.roleErr, p5.D, p5.final, p5.finalT
	gates := append([]gateResult(nil), p5.gates...)
	p5.mu.Unlock()
	static := p5StaticFile()
	static["P5.static.load"] = map[bool]string{true: "ok: TestP5 began", false: "T: TestP5 did not begin"}[ran]
	if roleErr != "" {
		static["P5.static.load"] = "T: " + roleErr
	}
	native := map[verdict]int{}
	nativeOK, notExec := 0, 0
	var layout []string
	resultsMu.Lock()
	for _, r := range results {
		if !strings.HasPrefix(r.ID, "P5.") {
			continue
		}
		if strings.Contains(r.Reason, "not executed") {
			notExec++
		}
		switch r.Evidence {
		case "native":
			native[r.Verdict]++
			if r.Verdict == confirmed {
				nativeOK++
			}
		case "layout":
			layout = append(layout, r.ID+" "+string(r.Verdict))
		case "static":
			static[r.ID] = string(r.Verdict)
			if r.Verdict != confirmed {
				static[r.ID] += ": " + r.Reason
			}
		}
	}
	resultsMu.Unlock()
	g := map[string]string{}
	for _, x := range gates {
		g[x.ID] = "pass"
		if !x.Pass {
			g[x.ID] = "FAIL: " + x.Reason + "; " + strings.Join(x.Problems, "; ")
		}
	}
	model := map[string]string{}
	modelMu.Lock()
	for k, ok := range modelRes {
		model[k] = map[bool]string{true: "pass", false: "FAIL"}[ok]
	}
	modelMu.Unlock()
	d := ""
	if !D.IsZero() {
		d = D.Format(time.RFC3339Nano)
	}
	b, _ := json.Marshal(map[string]any{
		"toolchain": runtime.Version(), "CGO_ENABLED": envOr("CGO_ENABLED", "(unset)"), "GOTOOLCHAIN": envOr("GOTOOLCHAIN", "(unset)"),
		"static": static, "native_confirmed": nativeOK, "native_expected": p5Expected, "native": native,
		"layout": layout, "gates": g, "model": model, "not_executed": notExec, "D": d,
		"final": final, "final_T": finalT, "H4": p5H4, "H6": p5H6,
	})
	fmt.Println("WP25P5 " + string(b))
}

// p5StaticFile reads the workflow's conclusions of the static steps
// (WP25_P5_STATIC: one "ID conclusion" line each); a file that is not
// there or cannot be read leaves them unknown.
func p5StaticFile() map[string]string {
	m := map[string]string{}
	ids := []string{"P5.static.toolchain", "P5.static.vet", "P5.static.build", "P5.static.settings", "P5.static.symbols"}
	for _, id := range ids {
		m[id] = "unknown: no conclusion"
	}
	path := os.Getenv(p5StaticEnv)
	if path == "" {
		return m
	}
	f, err := os.Open(path)
	if err != nil {
		for _, id := range ids {
			m[id] = "unknown: " + err.Error()
		}
		return m
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		id, v, _ := strings.Cut(sc.Text(), " ")
		if strings.HasPrefix(id, "P5.static.") {
			m[id] = v
		}
	}
	return m
}
