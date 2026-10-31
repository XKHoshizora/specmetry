//go:build darwin

package wp25

// R-B2-2 (docs/wp2.5-prototype.md §16.17.3, the fourth draft) and its
// fixes after Review 116 (§16.21): the cleanup chains on a real child each,
// direct checks of the root test process outside D (a gate's inner run
// selects only TestP5's runs, never these).
//
//   - TestP5CleanupReal: two subtests in turn, never parallel and with no
//     subtest of their own: "insurance" tests p5EndProven's chain,
//     "remediation" p5Remediate's, each with a seam that answers unknown
//     for its child's first round read, only when that real read gave
//     alive (the real read happens first and is kept). Healthy: each side
//     must hold with no problem at all.
//   - TestP5CleanupRealWait: one side, "wait-observe", the insurance chain
//     without its seam, and a seam on the Wait task's first read instead,
//     only when that real read succeeded. A negative self-check: the side's
//     judgement must reject it for exactly that failure of the holder,
//     with the child reaped as signal 9 and nothing else wrong.
//
// Each side starts one controlled child once, registers its holder
// (p5clean_test.go) and the holder's cleanup on its own t before anything
// else, runs the chain, and is judged by p5RealJudge after its cleanup,
// from the chain's structured failures, the seams' own records, the
// holder's unified consumption (p5HolderFailures) and its knowledge of the
// reap.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	p5RealTotal = 60 * time.Second // T_r: S_r plus this, no later than T_bin less p5Margin
	p5RealSide  = 25 * time.Second // a side's nominal window and admission
	p5RealEnv   = "WP25_P5REAL"    // a side's own mark in its child's environment
)

// p5RealCase is one side: its chain, the seams it plans, and the codes of
// the problems its judgement must give (none for a healthy side).
type p5RealCase struct {
	side   string
	chain  string // insurance or remediation
	spec   p5RealSpec
	expect []string // sorted
}

// p5RealSeamRec is a seam's own record.
type p5RealSeamRec struct {
	Hits    int    `json:"hits"`
	Real    string `json:"real,omitempty"`    // what the real read at its site gave
	Premise string `json:"premise,omitempty"` // why it replaced nothing
}

// p5RealRec is one side's record (WP25P5REAL), printed by its cleanup.
type p5RealRec struct {
	Side      string         `json:"side"`
	Plan      p5RealSpec     `json:"plan"`
	Expect    []string       `json:"expect,omitempty"`
	S         string         `json:"s_k"`
	C         string         `json:"c_k,omitempty"`
	Q         string         `json:"q_k,omitempty"`
	W         string         `json:"w_k,omitempty"`
	D         string         `json:"d_k,omitempty"`
	Pid       int            `json:"pid,omitempty"`
	Identity  string         `json:"identity,omitempty"`
	ChainSeam *p5RealSeamRec `json:"chain_seam,omitempty"`
	WaitSeam  *p5RealSeamRec `json:"wait_seam,omitempty"`
	Chain     *p5CleanResult `json:"chain,omitempty"`
	Consumed  []string       `json:"consumed,omitempty"` // the callers' consumption, as checked
	Know      *p5HeldKnow    `json:"know,omitempty"`     // after the cleanup
	Status    string         `json:"status,omitempty"`   // the reaped status, confirmed by the cleanup
	Expired   []string       `json:"expired,omitempty"`
	Handed    string         `json:"handed,omitempty"`
	Fails     []string       `json:"fails,omitempty"`  // the holder's own T
	Holder    []p5Failure    `json:"holder,omitempty"` // the unified consumption
	Problems  []p5Problem    `json:"problems,omitempty"`
	Pass      bool           `json:"pass"`
	Reason    string         `json:"reason,omitempty"`
}

// TestP5CleanupReal is R-B2-2's two healthy sides.
func TestP5CleanupReal(t *testing.T) {
	p5RealTop(t, []p5RealCase{
		{side: "insurance", chain: "insurance", spec: p5RealSpec{ChainSeam: 1}},
		{side: "remediation", chain: "remediation", spec: p5RealSpec{ChainSeam: 1}},
	})
}

// TestP5CleanupRealWait is B-116-1's real regression (§16.21.2): its own
// S_r and T_r by the same rule, one side.
func TestP5CleanupRealWait(t *testing.T) {
	p5RealTop(t, []p5RealCase{
		{side: "wait-observe", chain: "insurance", spec: p5RealSpec{WaitSeam: 1}, expect: []string{"holder"}},
	})
}

// p5RealTop runs the sides in turn: the next is admitted only once the
// previous side's t.Run has returned, its cleanup done.
func p5RealTop(t *testing.T, cases []p5RealCase) {
	Sr := time.Now()
	Tbin := Sr.Add(p5RealTotal)
	if dl, ok := t.Deadline(); ok {
		Tbin = dl
	}
	Tr := minTime(Sr.Add(p5RealTotal), Tbin.Add(-p5Margin))
	for _, c := range cases {
		t.Run(c.side, func(t *testing.T) { p5RealCheck(t, c, Tr) })
	}
}

// p5RealOps are a side's real reads and calls: procInfo for the parent's
// probe and W1's read, Cmd.Wait and its state for W4 and W5, kill for R2.
func p5RealOps(cmd *exec.Cmd, pid int) p5HolderOps {
	zombie := func() (bool, error) {
		st, pr := procInfo(pid)
		switch {
		case pr == unknown:
			return false, fmt.Errorf("the state of %d cannot be read", pid)
		case pr == absent:
			return false, fmt.Errorf("%d is absent, and the Wait task has not reaped it", pid)
		case st.ppid != os.Getpid():
			return false, fmt.Errorf("%d is not this process's child (its parent is %d)", pid, st.ppid)
		}
		return st.zombie, nil
	}
	return p5HolderOps{
		now:     time.Now,
		probe:   zombie,
		observe: zombie,
		wait: func() error {
			err := cmd.Wait()
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				return nil // an exit status, which state reads: signal 9 is no failure
			}
			return err
		},
		state: func() p5Exit {
			if cmd.ProcessState == nil {
				return p5Exit{Status: "no state", Code: -1}
			}
			ws, _ := cmd.ProcessState.Sys().(syscall.WaitStatus)
			return p5Exit{Status: waitStatus(ws), Code: cmd.ProcessState.ExitCode()}
		},
		kill:  func() error { return syscall.Kill(pid, syscall.SIGKILL) },
		every: p5Every,
	}
}

// p5RealCheck is one side.
func p5RealCheck(t *testing.T, c p5RealCase, Tr time.Time) {
	sk := time.Now()
	rec := &p5RealRec{Side: c.side, Plan: c.spec, Expect: c.expect, S: sk.Format(time.RFC3339Nano)}
	notRun := func(why string) {
		rec.Fails = append(rec.Fails, "not executed: "+why)
		rec.Reason = "not executed: " + why
		p5RealPrint(rec)
		t.Errorf("%s: not executed: %s", c.side, why)
	}
	// The premises and the admission: none holding, no child is started.
	if left := Tr.Sub(sk); left < p5RealSide {
		notRun(fmt.Sprintf("T_r is %v away, less than a side's %v", left.Round(time.Millisecond), p5RealSide))
		return
	}
	switch {
	case os.Getenv(innerEnv) != "":
		notRun("an inner run")
		return
	case witnessOn():
		notRun("the witness is on: start would read the child's identity and may wait for it")
		return
	}
	if _, err := os.Executable(); err != nil {
		notRun("os.Executable: " + err.Error())
		return
	}
	cq, q, w := sk.Add(10*time.Second), sk.Add(12*time.Second), sk.Add(20*time.Second)
	d := minTime(sk.Add(p5RealSide), Tr)
	rec.C, rec.Q, rec.W, rec.D = cq.Format(time.RFC3339Nano), q.Format(time.RFC3339Nano), w.Format(time.RFC3339Nano), d.Format(time.RFC3339Nano)
	tag := p5RealEnv + "=" + c.side + "." + strconv.FormatInt(sk.UnixNano(), 10)
	cmd := helper("exit-after", []string{tag}, "60000", "0")
	if err := start(cmd); err != nil {
		rec.Fails = append(rec.Fails, "start: "+err.Error())
		rec.Reason = "start: " + err.Error()
		p5RealPrint(rec)
		t.Errorf("%s: start: %v (no child)", c.side, err)
		return
	}
	// Registered first: the pid read once, the holder, its cleanup on this
	// side's t; only then any read or assertion. The W1 seam, if planned,
	// wraps the Wait task's read only; the parent's probe stays real.
	pid := cmd.Process.Pid
	ops := p5RealOps(cmd, pid)
	rseam, wseam := &p5ReadSeam{}, &p5ObserveSeam{}
	if c.spec.WaitSeam > 0 {
		ops.observe = wseam.wrap(ops.observe)
	}
	h := &p5Holder{side: c.side, pid: pid, cmd: cmd, ops: ops, ctl: &p5Ctl{}, c: cq, q: q, w: w, d: d, held: &p5RealHeld}
	rec.Pid = pid
	var callers []string // the callers' checks that did not hold
	t.Cleanup(func() {
		h.cleanup()
		// A chain that returned only after q_k: its late result recorded.
		if rec.Chain == nil && h.chain != nil {
			if done, val, _ := h.chain.snap(); done {
				if res, ok := val.(p5CleanResult); ok {
					rec.Chain = &res
				}
			}
		}
		f := p5RealFacts{Chain: rec.Chain, Holder: h.failures(), Protocol: h.fails, Requests: len(h.reqs)}
		f.ChainHits, rec.ChainSeam = p5RealSeam(rseam.state())
		f.WaitHits, rec.WaitSeam = p5RealSeam(wseam.state())
		f.ChainPremise, f.WaitPremise = rec.ChainSeam.Premise, rec.WaitSeam.Premise
		if h.wait != nil {
			ws := h.wait.snap()
			f.Reaped, f.Status = ws.Done, ws.Exit.Status
		}
		k := h.know()
		rec.Know, rec.Status, rec.Expired, rec.Fails, rec.Holder = &k, h.status, h.expired, h.fails, f.Holder
		if h.handed != nil {
			rec.Handed = h.handed.Reason
		}
		rec.Problems = p5RealJudge(c.spec, f)
		for _, x := range callers {
			rec.Problems = append(rec.Problems, p5Problem{"caller", x})
		}
		rec.Pass, rec.Reason = p5RealVerdict(c, f, rec.Problems)
		p5RealPrint(rec)
		if !rec.Pass {
			t.Errorf("%s: %s; problems %v", c.side, rec.Reason, rec.Problems)
		}
	})

	// 1. The identity.
	st, pr := procInfo(pid)
	if pr != present || st.ppid != os.Getpid() || st.zombie {
		h.fail("the child's identity cannot be read as this process's running child (presence %d, parent %d, zombie %v): no chain", pr, st.ppid, st.zombie)
		h.tail(false)
		return
	}
	v := survivor{pid, st.start}
	rec.Identity = st.start

	// 2. The chain, its first round read answered unknown once if planned.
	var cops p5CleanOps
	if c.chain == "insurance" {
		cops = p5EndProvenOps(tag, st.start, cq, nil)
	} else {
		cops = p5RemediateOps(tag)
	}
	if c.spec.ChainSeam > 0 {
		rseam.v = v // before the chain's task starts
		cops.read = rseam.wrap(cops.read)
	}
	if !h.chainStep(func() (any, error) { return p5EndChain([]survivor{v}, cq, cops), nil }) {
		return
	}
	_, val, _ := h.chain.snap()
	res, _ := val.(p5CleanResult)
	rec.Chain = &res

	// 3 and 4. The child read; a request only if the chain did not end it;
	// the Wait task; each new task a decision point against w_k.
	if h.tail(true) {
		return
	}
	if ws := h.wait.snap(); !ws.Done {
		h.fail("the Wait task has not completed by w_k (%s)", h.know().Reap)
	}
	// The callers' consumption of a chain that met its planned failure.
	if c.spec.ChainSeam > 0 {
		var fails []string
		if c.chain == "insurance" {
			rec.Consumed, fails = p5RealInsurance(res)
		} else {
			rec.Consumed, fails = p5RealRemedy(res)
		}
		callers = append(callers, fails...)
	}
}

func p5RealSeam(hits int, real, premise string) (int, *p5RealSeamRec) {
	return hits, &p5RealSeamRec{hits, real, premise}
}

// p5RealVerdict decides the side: its judgement's codes must be the ones
// the case expects; a negative self-check (the W1 seam) holds only on the
// correct rejection: exactly one failure of the holder, from the Wait
// task's read, planned, once, with the child reaped as signal 9.
func p5RealVerdict(c p5RealCase, f p5RealFacts, ps []p5Problem) (bool, string) {
	got := p5Codes(ps)
	if strings.Join(got, ",") != strings.Join(c.expect, ",") {
		return false, fmt.Sprintf("the judgement gave problem codes %v, want %v", got, c.expect)
	}
	if c.spec.WaitSeam > 0 {
		h := f.Holder
		if len(h) != 1 || h[0].Source != "wait-observe" || !h[0].Planned || h[0].Count != 1 || !f.Reaped || f.Status != "signal 9" {
			return false, fmt.Sprintf("the rejection is not the planned one: holder failures %v, reaped %v as %q", h, f.Reaped, f.Status)
		}
	}
	return true, ""
}

// p5RealInsurance checks the insurance's consumption of the chain's result
// with the callers' own small functions: each caller fails closed. That
// only the planned failure was met is the judgement's, never these.
func p5RealInsurance(res p5CleanResult) (record, fails []string) {
	acc := p5InsureAccount(true, res)
	if !strings.HasPrefix(acc, "FAILED") {
		fails = append(fails, fmt.Sprintf("the insurance's account %q, want FAILED", acc))
	}
	for _, x := range res.Failures {
		if !strings.Contains(acc, x.Text) {
			fails = append(fails, fmt.Sprintf("the insurance's account does not carry the chain's %s failure %q", x.Source, x.Text))
		}
	}
	gc := gateCase{id: "p5real.insurance", want: confirmed}
	g := gateResult{ID: gc.id, Want: confirmed, Remedy: "not-needed", Final: "ended"}
	judge(gc, &g, true)
	before := g.Pass
	p5AfterInsurance(gc, &g, true, acc)
	n := 0
	for _, p := range g.Problems {
		if strings.HasPrefix(p, "insurance: ") {
			n++
		}
	}
	if !before || g.Pass || n != 1 || len(g.ToolErrors) != 1 {
		fails = append(fails, fmt.Sprintf("after the insurance a passing gate gave pass %v (before %v), %d insurance problems, %d tool errors; want false, 1, 1", g.Pass, before, n, len(g.ToolErrors)))
	}
	procs, unk := p5FinalProcs(res)
	if len(unk) < 1 {
		fails = append(fails, "the final cleanup's processes part gave no unknown")
	}
	return []string{"account: " + acc, fmt.Sprintf("after the insurance: pass %v, problems %q", g.Pass, g.Problems), fmt.Sprintf("final: processes %v, unknown %q", procs, unk)}, fails
}

// p5RealRemedy checks the remediation's outcome.
func p5RealRemedy(res p5CleanResult) (record, fails []string) {
	out := p5RemedyOutcome(res, false)
	sa := false
	for _, p := range out.Problems {
		sa = sa || strings.HasPrefix(p, "still-alive: ")
	}
	if out.Remedy != "failed" || !sa || out.Final != "unknown" {
		fails = append(fails, fmt.Sprintf("the remediation's outcome %s, problems %q, final %s; want failed, a still-alive problem, unknown", out.Remedy, out.Problems, out.Final))
	}
	return []string{fmt.Sprintf("remediation: %s, problems %q, final %s", out.Remedy, out.Problems, out.Final)}, fails
}

func p5RealPrint(rec *p5RealRec) {
	b, _ := json.Marshal(rec)
	fmt.Println("WP25P5REAL " + string(b))
	p5.mu.Lock()
	p5.real = append(p5.real, *rec)
	p5.mu.Unlock()
}
