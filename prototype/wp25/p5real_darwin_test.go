//go:build darwin

package wp25

// R-B2-2 (docs/wp2.5-prototype.md §16.17.3, the fourth draft): the two
// cleanup chains on a real child each, in TestP5CleanupReal, a direct check
// of the root test process outside D (a gate's inner run selects only
// TestP5's runs, never this). Two subtests in turn, never parallel and
// with no subtest of their own: "insurance" tests p5EndProven's chain,
// "remediation" p5Remediate's. Each side starts one controlled child once,
// registers its holder (p5clean_test.go) and the holder's cleanup on its
// own t before anything else, then runs the chain with a seam that answers
// unknown for its child's first round read (the real read still happens),
// and checks the chain's result, its callers' consumption and the child's
// end: the chain's own SIGKILL, reaped by the one Wait task as signal 9,
// with no request of the holder.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const (
	p5RealTotal = 60 * time.Second // T_r: S_r plus this, no later than T_bin less p5Margin
	p5RealSide  = 25 * time.Second // a side's nominal window and admission
	p5RealEnv   = "WP25_P5REAL"    // a side's own mark in its child's environment
)

// p5RealRec is one side's record (WP25P5REAL), printed by its cleanup.
type p5RealRec struct {
	Side     string         `json:"side"`
	S        string         `json:"s_k"`
	C        string         `json:"c_k,omitempty"`
	Q        string         `json:"q_k,omitempty"`
	W        string         `json:"w_k,omitempty"`
	D        string         `json:"d_k,omitempty"`
	Pid      int            `json:"pid,omitempty"`
	Identity string         `json:"identity,omitempty"`
	SeamHits int            `json:"seam_hits"`
	SeamReal string         `json:"seam_real,omitempty"` // what the real read the seam answered for gave
	Chain    *p5CleanResult `json:"chain,omitempty"`
	Consumed []string       `json:"consumed,omitempty"` // the callers' consumption, as checked
	Know     *p5HeldKnow    `json:"know,omitempty"`     // after the cleanup
	Status   string         `json:"status,omitempty"`   // the reaped status, confirmed by the cleanup
	Expired  []string       `json:"expired,omitempty"`
	Handed   string         `json:"handed,omitempty"`
	Fails    []string       `json:"fails,omitempty"`
}

// p5ReadSeam answers unknown once for its child's first round read: the
// real read happens all the same, its answer kept apart. The chain runs in
// its own task, the side reads the hits after it returned.
type p5ReadSeam struct {
	mu   sync.Mutex
	v    survivor
	hits int
	real string
}

func (s *p5ReadSeam) wrap(read func(survivor, string) (string, error)) func(survivor, string) (string, error) {
	return func(v survivor, at string) (string, error) {
		class, err := read(v, at)
		s.mu.Lock()
		defer s.mu.Unlock()
		if v != s.v || at != "round" || s.hits > 0 {
			return class, err
		}
		s.hits++
		s.real = class
		if err != nil {
			s.real += " (" + err.Error() + ")"
		}
		return p5Unknown, fmt.Errorf("injected: the first round's read of %s answered unknown", v.key())
	}
}

func (s *p5ReadSeam) count() (int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits, s.real
}

// TestP5CleanupReal is R-B2-2.
func TestP5CleanupReal(t *testing.T) {
	Sr := time.Now()
	Tbin := Sr.Add(p5RealTotal)
	if dl, ok := t.Deadline(); ok {
		Tbin = dl
	}
	Tr := minTime(Sr.Add(p5RealTotal), Tbin.Add(-p5Margin))
	for _, side := range []string{"insurance", "remediation"} {
		// The second side is admitted only once the first side's t.Run has
		// returned, its cleanup done.
		t.Run(side, func(t *testing.T) { p5RealCheck(t, side, Tr) })
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
				return nil // an exit status, which state reads
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
func p5RealCheck(t *testing.T, side string, Tr time.Time) {
	sk := time.Now()
	rec := &p5RealRec{Side: side, S: sk.Format(time.RFC3339Nano)}
	notRun := func(why string) {
		rec.Fails = append(rec.Fails, "not executed: "+why)
		p5RealPrint(rec)
		t.Errorf("%s: not executed: %s", side, why)
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
	c, q, w := sk.Add(10*time.Second), sk.Add(12*time.Second), sk.Add(20*time.Second)
	d := minTime(sk.Add(p5RealSide), Tr)
	rec.C, rec.Q, rec.W, rec.D = c.Format(time.RFC3339Nano), q.Format(time.RFC3339Nano), w.Format(time.RFC3339Nano), d.Format(time.RFC3339Nano)
	tag := p5RealEnv + "=" + side + "." + strconv.FormatInt(sk.UnixNano(), 10)
	cmd := helper("exit-after", []string{tag}, "60000", "0")
	if err := start(cmd); err != nil {
		rec.Fails = append(rec.Fails, "start: "+err.Error())
		p5RealPrint(rec)
		t.Errorf("%s: start: %v (no child)", side, err)
		return
	}
	// Registered first: the pid read once, the holder, its cleanup on this
	// side's t; only then any read or assertion.
	pid := cmd.Process.Pid
	h := &p5Holder{side: side, pid: pid, cmd: cmd, ops: p5RealOps(cmd, pid), ctl: &p5Ctl{}, c: c, q: q, w: w, d: d, held: &p5RealHeld}
	rec.Pid = pid
	seam := &p5ReadSeam{}
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
		rec.SeamHits, rec.SeamReal = seam.count()
		k := h.know()
		rec.Know, rec.Status, rec.Expired, rec.Fails = &k, h.status, h.expired, h.fails
		if h.handed != nil {
			rec.Handed = h.handed.Reason
		}
		p5RealPrint(rec)
		for _, f := range h.unreported() {
			t.Errorf("%s: %s", side, f)
		}
	})
	defer func() {
		for _, f := range h.unreported() {
			t.Errorf("%s: %s", side, f)
		}
	}()

	// 1. The identity.
	st, pr := procInfo(pid)
	if pr != present || st.ppid != os.Getpid() || st.zombie {
		h.fail("the child's identity cannot be read as this process's running child (presence %d, parent %d, zombie %v): no chain", pr, st.ppid, st.zombie)
		h.tail(false)
		return
	}
	v := survivor{pid, st.start}
	rec.Identity = st.start

	// 2. The chain, its first round read answered unknown once.
	seam.v = v // before the chain's task starts
	var ops p5CleanOps
	if side == "insurance" {
		ops = p5EndProvenOps(tag, st.start, c, nil)
	} else {
		ops = p5RemediateOps(tag)
	}
	ops.read = seam.wrap(ops.read)
	if !h.chainStep(func() (any, error) { return p5EndChain([]survivor{v}, c, ops), nil }) {
		return
	}
	_, val, _ := h.chain.snap()
	res, _ := val.(p5CleanResult)
	rec.Chain = &res
	rec.SeamHits, rec.SeamReal = seam.count()

	// 3 and 4. The child read; a request only if the chain did not end it;
	// the Wait task; each new task a decision point against w_k.
	if h.tail(true) {
		return
	}

	// 5. The end is the chain's: signal 9 with no request of the holder.
	ws := h.wait.snap()
	switch {
	case !ws.Done:
		h.fail("the Wait task has not completed by w_k (%s)", p5Knowledge(h.chain, h.reqs, h.wait).Reap)
	case ws.Exit.Status != "signal 9":
		h.fail("the child was reaped as %s, not signal 9", ws.Exit.Status)
	}
	if len(h.reqs) != 0 {
		h.fail("the holder made %d termination requests: signal 9 proves nothing of the chain then", len(h.reqs))
	}
	if rec.SeamHits != 1 {
		h.fail("the seam was hit %d times, want 1", rec.SeamHits)
	}
	if res.Found != 1 || len(res.Ended) != 1 || len(res.Left) != 0 || !strings.Contains(res.ErrText, "injected: ") {
		h.fail("the chain's result: found %d, ended %v, left %v, errors %q; want 1 found, 1 ended, none left, the injected error kept", res.Found, res.Ended, res.Left, res.ErrText)
	}
	if res.Kills < 1 {
		h.fail("the chain sent no kill")
	}
	if side == "insurance" {
		rec.Consumed = p5RealInsurance(h, res)
	} else {
		rec.Consumed = p5RealRemedy(h, res)
	}
}

// p5RealInsurance checks the insurance's consumption of the chain's result
// with the callers' own small functions.
func p5RealInsurance(h *p5Holder, res p5CleanResult) []string {
	acc := p5InsureAccount(true, res)
	if !strings.HasPrefix(acc, "FAILED") || !strings.Contains(acc, "injected: ") {
		h.fail("the insurance's account %q, want FAILED with the injected error", acc)
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
		h.fail("after the insurance a passing gate gave pass %v (before %v), %d insurance problems, %d tool errors; want false, 1, 1", g.Pass, before, n, len(g.ToolErrors))
	}
	procs, unk := p5FinalProcs(res)
	if len(unk) < 1 {
		h.fail("the final cleanup's processes part gave no unknown")
	}
	return []string{"account: " + acc, fmt.Sprintf("after the insurance: pass %v, problems %q", g.Pass, g.Problems), fmt.Sprintf("final: processes %v, unknown %q", procs, unk)}
}

// p5RealRemedy checks the remediation's outcome.
func p5RealRemedy(h *p5Holder, res p5CleanResult) []string {
	out := p5RemedyOutcome(res, false)
	sa := false
	for _, p := range out.Problems {
		sa = sa || strings.HasPrefix(p, "still-alive: ")
	}
	if out.Remedy != "failed" || !sa || out.Final != "unknown" {
		h.fail("the remediation's outcome %s, problems %q, final %s; want failed, a still-alive problem, unknown", out.Remedy, out.Problems, out.Final)
	}
	return []string{fmt.Sprintf("remediation: %s, problems %q, final %s", out.Remedy, out.Problems, out.Final)}
}

func p5RealPrint(rec *p5RealRec) {
	b, _ := json.Marshal(rec)
	fmt.Println("WP25P5REAL " + string(b))
	p5.mu.Lock()
	p5.real = append(p5.real, *rec)
	p5.mu.Unlock()
}
