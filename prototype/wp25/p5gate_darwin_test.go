//go:build darwin

package wp25

// P-5's gates (docs/wp2.5-prototype.md §16.6, §16.8; role B′): each runs
// P5.layout.1 and P5.w.exit7.1 again in a new process of this test binary
// (role B, under d_i), and judges it as runGate judges P-3's gates, with
// exact. What differs is how the inner run is started, waited for and
// handed on, by the fourth draft's protocol (p5proto_test.go):
//
//   - its output goes to a pipe of the gate's own, read by a copy task;
//     no WaitDelay, no CommandContext, no wait without a deadline;
//   - from the start, before anything else: the parent's write end closed,
//     the pid read once, the control lock and its mark made, the Wait and
//     copy tasks started; only then the inner run's identity;
//   - every wait is a query by an absolute deadline: k = d_i + 5 s, w =
//     d_i + 10 s, d_g = s + 60 s; the gate never takes the control lock;
//   - the inner run itself is signalled only by request tasks (R0), never
//     by the census, the remediation or the insurance by its PID;
//   - what is not confirmed by d_g is handed on to the final cleanup.

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// p5GateCase is a gate with the target's planned calls (§16.6). fault is
// an outer seam of one combination's run (§16.17.2 R-B1-2: "close-error");
// the two gates have none, and only they check R-B1-1.
type p5GateCase struct {
	gateCase
	calls int
	fault string
}

// p5Gates are §16.6's gate and its control. TestGate's list is unchanged.
var p5Gates = []p5GateCase{
	{gateCase{id: "p5.eintr-once", test: "TestP5$/^(layout.1|w.exit7.1)", scenario: "P5.w.exit7.1", inject: "p5-eintr-once", want: invalid, goos: []string{"darwin"}, hit: true, hits: 1, exact: true, because: "observation failed: waitid 1: interrupted system call"}, 1, ""},
	{gateCase{id: "control.p5-w-exit7", test: "TestP5$/^(layout.1|w.exit7.1)", scenario: "P5.w.exit7.1", want: confirmed, goos: []string{"darwin"}, exact: true}, 2, ""},
}

// p5GateInfo is a P-5 gate's own record (§16.11): its deadlines and its
// knowledge of the inner run when it returned.
type p5GateInfo struct {
	S        string  `json:"s"`
	Dg       string  `json:"d_g"`
	Di       string  `json:"d_i"`
	K        string  `json:"k"`
	W        string  `json:"w"`
	Pid      int     `json:"pid,omitempty"`
	Identity string  `json:"identity,omitempty"` // the inner run's start time, "" if unknown
	Know     *p5Know `json:"inner_state,omitempty"`
	Exit     string  `json:"exit"` // the inner run's exit, or unknown
	Output   string  `json:"output"`
	// ReadClose is the read end's close (§16.17.2 item 8).
	ReadClose *p5ReadClose `json:"read_close,omitempty"`
	// HandedWhy are the hand-over predicate's reasons, on the snapshot after
	// the insurance (§16.17.2 item 4); Handed, what was handed on.
	HandedWhy []string `json:"handed_why,omitempty"`
	Handed    []string `json:"handed,omitempty"`
}

// checkP5Gate runs one P-5 gate and records it.
func checkP5Gate(t *testing.T, c p5GateCase) {
	g := runP5Gate(t, c)
	b, _ := json.Marshal(g)
	fmt.Println("WP25GATE " + string(b))
	gateMu.Lock()
	gateResults = append(gateResults, g)
	gateMu.Unlock()
	p5.mu.Lock()
	p5.gates = append(p5.gates, g)
	p5.mu.Unlock()
	if !g.Pass {
		t.Errorf("%s: want %s, got %s (exit %d): %s; %v", c.id, c.want, g.Got, g.InnerExit, g.Reason, g.Problems)
	}
}

// p5Launch is the start (§16.8, the start steps 2 to 4). It gives the
// inner run as the gate holds it, nil if there is no child, and the
// problems met; a closed write end's error is a T, the start goes on. The
// read end goes into the inner run: only its stop task closes it
// (§16.17.2).
func p5Launch(test, inject string, di time.Time, env []string) (*p5Inner, *exec.Cmd, []string) {
	self, err := os.Executable()
	if err != nil {
		return nil, nil, []string{"inner: os.Executable: " + err.Error()}
	}
	rd, wr, err := os.Pipe()
	if err != nil {
		return nil, nil, []string{"inner: os.Pipe: " + err.Error()}
	}
	timeout := int(math.Ceil(time.Until(di).Seconds())) + 5
	cmd := exec.Command(self, "-test.run", "^"+test+"$", "-test.count=1", "-test.v", fmt.Sprintf("-test.timeout=%ds", timeout))
	cmd.Env = append(append(os.Environ(), innerEnv+"=1", injectEnv+"="+inject, p5DeadlineEnv+"="+strconv.FormatInt(di.UnixNano(), 10)), env...)
	cmd.Stdout, cmd.Stderr = wr, wr
	if err := cmd.Start(); err != nil {
		probs := []string{"inner: the inner run did not start: " + err.Error()}
		for _, f := range []struct {
			name string
			f    *os.File
		}{{"read", rd}, {"write", wr}} {
			if cerr := f.f.Close(); cerr != nil {
				probs = append(probs, fmt.Sprintf("inner: closing the %s end after the failed start: %v", f.name, cerr))
			}
		}
		return nil, nil, probs
	}
	var probs []string
	// The parent's write end, closed by the gate itself, holding no lock:
	// nothing else of this process uses it, and Cmd does not take a
	// caller's *os.File as one of its own to close.
	if err := wr.Close(); err != nil {
		probs = append(probs, "inner: closing the parent's write end failed: "+err.Error())
	}
	pid := cmd.Process.Pid // read once
	ops := p5Ops{
		observe: func() (bool, error) {
			st, pr := procInfo(pid)
			switch {
			case pr == unknown:
				return false, fmt.Errorf("the state of the inner run %d cannot be read", pid)
			case pr == absent:
				return false, fmt.Errorf("the inner run %d is absent, and the Wait task has not reaped it", pid)
			case st.ppid != os.Getpid():
				return false, fmt.Errorf("%d is not this process's child (its parent is %d)", pid, st.ppid)
			}
			return st.zombie, nil
		},
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
	p := &p5Inner{pid: pid, ops: ops, ctl: &p5Ctl{}, rd: rd.Close}
	p.wait = p5StartWait(p.ctl, ops)
	p.copy = p5StartCopy(rd)
	return p, cmd, probs
}

// runP5Gate is one P-5 gate. Its judgment is runGate's: the inner verdict
// (verdictOf, exact and §16.6's evidence and calls), the state left, the
// remediation, the witness, the hits and the registrations, and after it
// the insurance cleanup; all of it within d_g.
func runP5Gate(t *testing.T, c p5GateCase) gateResult {
	began := time.Now()
	g := gateResult{ID: c.id, Test: c.test, Inject: c.inject, Want: c.want, Reject: c.reject, InnerExit: -1, Total: -1}
	info := &p5GateInfo{S: began.Format(time.RFC3339Nano), Exit: "unknown", Output: "none"}
	g.P5 = info
	p5.mu.Lock()
	D := p5.D
	p5.mu.Unlock()
	if left := D.Sub(began); left < p5Window {
		g.Reason = fmt.Sprintf("not executed: D is %v away, less than a gate's %v", left.Round(time.Millisecond), p5Window)
		g.Problems = append(g.Problems, "inner: "+g.Reason)
		g.Remedy, g.Final = "not-run", "unknown"
		return finishGate(c.gateCase, g, false, began)
	}
	dg := minTime(began.Add(p5Window), D)
	di := dg.Add(-30 * time.Second)
	k, w := di.Add(5*time.Second), di.Add(10*time.Second)
	info.Dg, info.Di, info.K, info.W = dg.Format(time.RFC3339Nano), di.Format(time.RFC3339Nano), k.Format(time.RFC3339Nano), w.Format(time.RFC3339Nano)
	g0 := time.Now()
	dir := t.TempDir()
	wf := emptyFile(t, dir, "witness-*")
	il := emptyFile(t, dir, "inject-*")
	rw, err := newRaiseWatch()
	if err != nil {
		t.Fatal(err)
	}
	defer rw.close()
	tag := witnessEnv + "=" + wf
	inner, cmd, probs := p5Launch(c.test, c.inject, di, []string{tag, injectLogEnv + "=" + il, rw.env()})
	g.Problems = append(g.Problems, probs...)
	if inner == nil {
		g.Reason = "inner run: " + strings.Join(probs, "; ")
		g.Remedy, g.Final = "not-run", "unknown"
		return finishGate(c.gateCase, g, false, began)
	}
	inner.closeSeam = c.fault == "close-error" // R-B1-2: this run's stop task only
	info.Pid = inner.pid
	// Only now the inner run's identity. A read that fails is a T; the
	// request needs no identity, and is made at once (R0).
	start := ""
	if st, pr := procInfo(inner.pid); pr == present {
		start = st.start
	} else {
		g.Problems = append(g.Problems, "inner: the inner run's identity: unknown")
		inner.request("gate", "identity")
	}
	info.Identity = start

	// k: the inner run has not ended by itself; R0, the gate takes no
	// control lock.
	ws := inner.wait.until(k)
	if ws.Seen.IsZero() {
		inner.request("gate", "k")
		g.Problems = append(g.Problems, "inner: the inner run did not end by itself by d_i + 5 s")
	}
	// w: the request task, the Wait task and the copy, each queried by w.
	if q := inner.pendingReq(); q != nil {
		q.until(w)
	}
	ws = inner.wait.until(w)
	cs := inner.copy.until(w)
	queried := time.Now() // the copy's query returned: perhaps before w
	complete := cs.Done
	// The read end, closed on every path once the copy's query returned,
	// whether the copy ended or not (§16.17.2 item 2), by the one stop task.
	origin := "the query by w"
	switch {
	case cs.Done && cs.Err != nil:
		origin = "a read error"
	case cs.Done:
		origin = "EOF"
	}
	stop := inner.closeRead(origin)
	var ss p5StopSnap
	if cs.Done {
		ss = stop.until(w)
	} else {
		// One query each that does not wait.
		cs = inner.copy.snap()
		ss = stop.snap()
		g.Problems = append(g.Problems, "inner: the output was not complete by d_i + 10 s")
	}
	// The close's result, consumed before the verdict (§16.17.2 item 3).
	if p := p5CloseProblem(ss); p != "" {
		g.Problems = append(g.Problems, p)
	}
	rc := p5ReadCloseOf(ss)
	rc.Consumed = true
	info.ReadClose = &rc
	if c.fault == "" {
		for _, why := range p5CloseNormal(ss, queried, w) {
			g.Problems = append(g.Problems, "inner: R-B1-1: "+why)
		}
	}
	if !ws.Done {
		g.Problems = append(g.Problems, "inner: exit not confirmed by d_i + 10 s; "+p5Requests(inner))
	}
	r := innerRun{results: map[string]result{}, exit: -1}
	switch {
	case cs.Done:
		// What the copy took is safe to read now; only whole lines.
		b := cs.Buf
		if i := strings.LastIndexByte(string(b), '\n'); i+1 < len(b) {
			info.Output = fmt.Sprintf("%d bytes, the last %d of a line cut", len(b), len(b)-i-1)
			b = b[:i+1]
		} else {
			info.Output = fmt.Sprintf("%d bytes", len(b))
		}
		if complete {
			info.Output = "complete: " + info.Output
			if cs.Err != nil {
				g.Problems = append(g.Problems, "inner: reading the output failed: "+cs.Err.Error())
			}
		} else {
			info.Output = "partial: " + info.Output
		}
		parseInner(b, &r)
	default:
		info.Output = "none: the copy had not returned"
		g.Problems = append(g.Problems, "inner: no output could be read safely")
	}
	if !complete && r.err == nil {
		r.err = errors.New("the output is not complete")
	}
	if ws.Done {
		r.exit = ws.Exit.Code
		info.Exit = ws.Exit.Status
		if ws.WaitErr != nil {
			g.Problems = append(g.Problems, "inner: Cmd.Wait failed: "+ws.WaitErr.Error())
		}
	}
	if ws.Errs != nil {
		g.Problems = append(g.Problems, "inner: the Wait task's reads of the inner run failed: "+ws.Errs.Error())
	}
	g.InnerExit, g.Hits, g.Sites, g.Inner = r.exit, r.hits, r.sites, r.all
	verdictOK := verdictOf(c.gateCase, r, &g)
	switch {
	case !ws.Done:
		verdictOK = false
		g.Reason = "the inner run's exit is unknown: " + g.Reason
	case verdictOK && c.exact:
		if why := p5Exact(c, r); why != "" {
			verdictOK = false
			g.Reason = why
		}
	}

	// The state the inner run left, within d_g: the census by this round's
	// tag (time bound: the inner run's start, or g0 without it), the inner
	// run apart from the rest; then the recorded processes.
	since := start
	if since == "" {
		since = fmt.Sprintf("%d.%06d", g0.Unix(), g0.Nanosecond()/1000)
	}
	survivors, cerr := censusTag(tag, since, minTime(time.Now().Add(tKill), dg))
	if cerr != nil {
		g.Problems = append(g.Problems, "census: "+cerr.Error())
	}
	innerAlive := false
	var others []survivor
	for _, v := range survivors {
		if v.pid == inner.pid {
			innerAlive = true
			continue
		}
		others = append(others, v)
	}
	file, rerr := os.ReadFile(wf)
	recs, _ := parseWitness(file)
	if rerr != nil {
		recs = nil
	}
	inCensus := map[string]bool{}
	for _, v := range others {
		inCensus[v.key()] = true
	}
	created := map[string]witnessRec{}
	receipts := map[string]bool{}
	for _, x := range recs {
		switch x.kind {
		case "created":
			created[x.key()] = x
		case "launched":
			receipts[x.key()] = true
			g.Receipts++
		}
	}
	g.Created = len(created)
	state := map[string]string{}
	var claimed []survivor
	for key, x := range created {
		a, err := recordAlive(x)
		switch {
		case err != nil:
			state[key] = "unknown"
			g.Problems = append(g.Problems, fmt.Sprintf("state-unknown: process %s: %v", key, err))
		case a:
			state[key] = "alive"
			if !inCensus[key] {
				g.Problems = append(g.Problems, fmt.Sprintf("census: process %s alive by its record, not found by the census", key))
				claimed = append(claimed, survivor{x.pid, x.start})
			}
		default:
			state[key] = "ended"
			if !receipts[key] {
				g.NoReceipt++
			}
		}
	}
	ip := &innerProc{cmd: cmd, pid: inner.pid, start: start, tag: tag}
	act, refused, oprobs := sortCandidates(others, claimed, ip.ownedNow)
	g.Problems = append(g.Problems, oprobs...)
	g.Refused = refused
	for _, v := range act {
		g.Left = append(g.Left, v.pid)
	}
	if innerAlive {
		g.Left = append(g.Left, inner.pid)
	}
	for _, v := range others {
		if _, ok := created[v.key()]; !ok {
			g.Unrecord = append(g.Unrecord, v.pid)
		}
	}
	sort.Ints(g.Left)
	if len(g.Left) > 0 {
		g.Problems = append(g.Problems, fmt.Sprintf("left: %d processes of the inner run alive when it was over: %v", len(g.Left), g.Left))
	}
	if len(g.Unrecord) > 0 {
		g.Problems = append(g.Problems, fmt.Sprintf("unrecorded: %d of them without a creation record: %v", len(g.Unrecord), g.Unrecord))
	}

	// The remediation, within min(now + 10 s, d_g): this round's proven
	// processes by their identity and tag; the inner run only by R0.
	for _, v := range act {
		g.Acted = append(g.Acted, v.key())
	}
	sort.Strings(g.Acted)
	g.Remedy = "not-needed"
	remedyFinal := "ended"
	innerOpen := !inner.wait.snap().Done
	if len(act) > 0 || innerAlive || innerOpen {
		g.Remedy = "not-run"
		rdl := minTime(time.Now().Add(10*time.Second), dg)
		res := p5Remediate(act, tag, rdl)
		if innerOpen {
			if q := inner.request("gate", "remedy"); q != nil {
				q.until(rdl)
			}
			innerOpen = !inner.wait.until(rdl).Done
		}
		out := p5RemedyOutcome(res, innerOpen)
		g.Remedy, g.RemedyErr, g.Still, remedyFinal = out.Remedy, out.Err, out.Still, out.Final
		g.Problems = append(g.Problems, out.Problems...)
	}
	unknownN := 0
	for _, st := range state {
		if st == "unknown" {
			unknownN++
		}
	}
	switch {
	case remedyFinal == "alive":
		g.Final = "alive"
	case cerr != nil || unknownN > 0 || remedyFinal == "unknown" || len(refused) > 0:
		g.Final = "unknown"
	default:
		g.Final = "ended"
	}

	// The witness; a raise is read last.
	g.Raised = rw.raised()
	_, wprobs := judgeWitness(evidence{file: file, readErr: rerr, seals: r.seals, inner: inner.pid, start: start, raised: g.Raised, total: -1})
	g.Problems = append(g.Problems, wprobs...)
	if b, err := os.ReadFile(il); err == nil {
		g.Hits += strings.Count(string(b), "\n")
		for _, l := range strings.Split(string(b), "\n") {
			var h struct {
				Site string `json:"site"`
			}
			if l != "" && json.Unmarshal([]byte(l), &h) == nil {
				g.Sites = append(g.Sites, "helper: "+h.Site)
			}
		}
	}
	if c.hit && g.Hits == 0 {
		g.Problems = append(g.Problems, "hit: the fault never reached its site")
	}
	if c.hits > 0 && g.Hits != c.hits {
		g.Problems = append(g.Problems, fmt.Sprintf("hit: the fault reached its sites %d times, want %d", g.Hits, c.hits))
	}
	owned := map[string]bool{}
	for _, res := range r.results {
		for _, key := range res.Owned {
			owned[key] = true
		}
		g.Shortfall += res.Shortfall
	}
	for key, x := range created {
		if !owned[key] {
			g.Unowned = append(g.Unowned, fmt.Sprintf("%d:%s", x.pid, state[key]))
		}
	}
	sort.Strings(g.Unowned)
	if len(g.Unowned) > g.Shortfall {
		g.Problems = append(g.Problems, fmt.Sprintf("unaccounted: %d created processes not registered by the inner run, %d declared short", len(g.Unowned), g.Shortfall))
	}
	g = finishGate(c.gateCase, g, verdictOK, began)

	// The insurance, after the judgment, within min(now + 10 s, d_g); it
	// never makes the gate pass.
	idl := minTime(time.Now().Add(10*time.Second), dg)
	left, ins := p5Insure(inner, act, tag, since, idl)
	p5AfterInsurance(c.gateCase, &g, verdictOK, ins)

	// What is not confirmed is handed on to the final cleanup, whole: the
	// hand-over predicate on a new snapshot (§16.17.2 item 4). A close that
	// completed after the verdict is recorded as late; it changes no T.
	k2 := inner.know()
	info.Know = &k2
	ss2 := inner.stop.snap()
	if !rc.Done && ss2.Done {
		late := p5ReadCloseOf(ss2)
		info.ReadClose.Late = fmt.Sprintf("completed after the verdict: real %s, reported %s, %s", late.Real, late.Reported, late.Release)
	}
	why := p5HandOver(k2.WaitDone, k2.OutputDone, inner.pendingReq() != nil, ss2)
	info.HandedWhy = why
	if c.fault == "" && !ss2.Released() {
		g.Problems = append(g.Problems, "inner: R-B1-1: the inner run is handed on for its read end, its release not confirmed")
		judge(c.gateCase, &g, verdictOK)
	}
	if len(why) > 0 || len(left) > 0 {
		h := &p5Hand{gate: c.id, tag: tag, since: since, others: left}
		if len(why) > 0 {
			h.inner = inner
			info.Handed = append(info.Handed, "the inner run: Cmd, Process, its control lock, the Wait task, the copy and stop tasks and the request tasks")
		}
		for _, v := range left {
			info.Handed = append(info.Handed, "process "+v.key())
		}
		p5.mu.Lock()
		p5.handed = append(p5.handed, h)
		p5.mu.Unlock()
	}
	g.Millis = time.Since(began).Milliseconds()
	return g
}

// p5Requests describes the inner run's request tasks.
func p5Requests(p *p5Inner) string {
	if len(p.reqs) == 0 {
		return "no termination request"
	}
	var parts []string
	for _, q := range p.reqs {
		s := q.snap()
		part := fmt.Sprintf("request %s/%s: %s", s.Holder, s.Step, s.Result)
		if s.Result == "pending" {
			part += " (the request has not completed: it may wait for the control lock or in a system call)"
		}
		if s.At != "" {
			part += " at " + s.At
		}
		if s.Err != "" {
			part += ", error " + s.Err
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
}

// p5Exact is §16.6's addition to exact: the final results of the inner
// run's layout probe and target, their evidence, and their calls, present
// and as planned; calls missing are no 0.
func p5Exact(c p5GateCase, r innerRun) string {
	lay, okL := r.results["P5.layout.1"]
	tgt, okT := r.results[c.scenario]
	switch {
	case !okL:
		return "exact: the inner run gave no final result for P5.layout.1"
	case !okT:
		return "exact: the inner run gave no final result for " + c.scenario
	case lay.Evidence != "layout":
		return fmt.Sprintf("exact: P5.layout.1's evidence is %q, not layout", lay.Evidence)
	case lay.Calls == nil:
		return "exact: P5.layout.1 has no calls"
	case *lay.Calls != 0:
		return fmt.Sprintf("exact: P5.layout.1 made %d calls, not 0", *lay.Calls)
	case tgt.Evidence != "seam":
		return fmt.Sprintf("exact: %s's evidence is %q, not seam", c.scenario, tgt.Evidence)
	case tgt.Calls == nil:
		return "exact: " + c.scenario + " has no calls"
	case *tgt.Calls != c.calls:
		return fmt.Sprintf("exact: %s made %d calls, not %d", c.scenario, *tgt.Calls, c.calls)
	}
	return ""
}

// p5ReadReal is a cleanup chain's real read of a recorded identity
// (§16.17.3's table): procInfo once.
func p5ReadReal(v survivor, _ string) (string, error) {
	st, pr := procInfo(v.pid)
	switch {
	case pr == unknown:
		return p5Unknown, fmt.Errorf("process %s: its state cannot be read", v.key())
	case pr == absent:
		return p5Absent, nil
	case st.start != v.start:
		return p5Changed, nil
	case st.zombie:
		return p5Zombie, nil
	}
	return p5Alive, nil
}

// p5TagReal reads a process's environment for the tag, environOf directly
// (tagged keeps its old rule).
func p5TagReal(tag string) func(int) (bool, error) {
	return func(pid int) (bool, error) {
		env, err := environOf(pid)
		if err != nil {
			return false, err
		}
		return hasToken(env, tag), nil
	}
}

// p5EndProvenOps are the real reads and calls of p5EndProven: the census
// by the round's tag within the call's deadline, kill directly, the inner
// run skipped.
func p5EndProvenOps(tag, since string, deadline time.Time, inner *p5Inner) p5CleanOps {
	ops := p5CleanOps{
		read:   p5ReadReal,
		tagged: p5TagReal(tag),
		census: func() ([]survivor, error) { return censusTag(tag, since, deadline) },
		kill:   func(pid int) error { return syscall.Kill(pid, syscall.SIGKILL) },
		now:    time.Now,
		every:  20 * time.Millisecond,
	}
	if inner != nil {
		ops.skip = func(v survivor) bool { return v.pid == inner.pid }
	}
	return ops
}

// p5EndProven ends what is still alive of a round (§16.17.3): the
// processes proven its own and what a census of its tag finds, never the
// inner run itself (only its request tasks signal it); each only while it
// runs, is proven this round's and carries the tag in the same step.
func p5EndProven(known []survivor, tag, since string, deadline time.Time, inner *p5Inner) p5CleanResult {
	return p5EndChain(known, deadline, p5EndProvenOps(tag, since, deadline, inner))
}

// p5RemediateOps are the remediation's: no census, the outer signal.
func p5RemediateOps(tag string) p5CleanOps {
	return p5CleanOps{read: p5ReadReal, tagged: p5TagReal(tag), kill: outerSignal, now: time.Now, every: 20 * time.Millisecond}
}

// p5Remediate ends this round's proven processes (never the inner run) by
// the same chain, without a census.
func p5Remediate(ss []survivor, tag string, deadline time.Time) p5CleanResult {
	return p5EndChain(ss, deadline, p5RemediateOps(tag))
}

// p5Insure is the insurance cleanup of a P-5 gate: within the gate's
// deadline, the inner run apart: it is never signalled by its PID here,
// only by a request task (R0), and its end is the Wait task's. It gives
// what is alive or unknown of the rest, to hand on, and its account.
func p5Insure(inner *p5Inner, known []survivor, tag, since string, deadline time.Time) ([]survivor, string) {
	if !inner.wait.snap().Done {
		if q := inner.request("gate", "insurance"); q != nil {
			q.until(deadline)
		}
	}
	res := p5EndProven(known, tag, since, deadline, inner)
	innerDone := inner.wait.until(deadline).Done
	return append(append([]survivor(nil), res.Alive...), res.Unknown...), p5InsureAccount(innerDone, res)
}

// p5CloseCombo is R-B1-2's combination (§16.17.2): one real run of
// control.p5-w-exit7's gate, the outer seam on its stop task.
var p5CloseCombo = p5GateCase{gateCase: p5Gates[1].gateCase, calls: p5Gates[1].calls, fault: "close-error"}

// checkP5CloseCombo runs it and checks every expectation; it is counted
// among the combinations (WP25COMBO), never as a gate passed.
func checkP5CloseCombo(t *testing.T) {
	c := p5CloseCombo
	g := runP5Gate(t, c)
	res := comboResult{ID: "p5combo.close-error", Gate: c.id, Fault: c.fault, WantPass: false, Self: g}
	var why []string
	info, _ := g.P5.(*p5GateInfo)
	var rc p5ReadClose
	if info != nil && info.ReadClose != nil {
		rc = *info.ReadClose
	}
	switch {
	case strings.HasPrefix(g.Reason, "not executed"):
		why = append(why, g.Reason)
	case !rc.Injected:
		why = append(why, "the seam on the stop task was not hit (its real close: "+rc.Real+")")
	}
	if g.Pass {
		why = append(why, "the gate passed")
	}
	if len(g.Problems) != 1 || !strings.HasPrefix(g.Problems[0], "inner: closing the read end failed: injected: ") {
		why = append(why, fmt.Sprintf("problems %q, want exactly the injected close's", g.Problems))
	}
	if len(g.ToolErrors) != 1 || len(g.Unexpected) != 0 {
		why = append(why, fmt.Sprintf("tool errors %d and other problems %d, want 1 and 0", len(g.ToolErrors), len(g.Unexpected)))
	}
	lay, tgt := p5InnerResult(g, "P5.layout.1"), p5InnerResult(g, c.scenario)
	switch {
	case lay == nil || lay.Calls == nil || *lay.Calls != 0:
		why = append(why, "the inner layout probe's calls are not 0")
	case tgt == nil || tgt.Verdict != confirmed || tgt.Calls == nil || *tgt.Calls != 2:
		why = append(why, "the inner target is not confirmed with 2 calls")
	case g.InnerExit != 0:
		why = append(why, fmt.Sprintf("the inner run exited %d", g.InnerExit))
	}
	if g.Remedy != "not-needed" || g.Final != "ended" || g.Insurance != "nothing left" {
		why = append(why, fmt.Sprintf("remediation %s, final %s, insurance %q; want not-needed, ended, nothing left", g.Remedy, g.Final, g.Insurance))
	}
	if rc.Real != "nil" || rc.Release != "released" || !strings.HasPrefix(rc.Reported, "injected: ") {
		why = append(why, fmt.Sprintf("read_close real %s (%s), reported %s; want nil (released) and the injected error", rc.Real, rc.Release, rc.Reported))
	}
	if info == nil || len(info.HandedWhy) > 0 || len(info.Handed) > 0 {
		why = append(why, "something was handed on")
	}
	res.Pass, res.Reason = len(why) == 0, strings.Join(why, "; ")
	b, _ := json.Marshal(res)
	fmt.Println("WP25COMBO " + string(b))
	comboMu.Lock()
	comboResults = append(comboResults, res)
	comboMu.Unlock()
	p5.mu.Lock()
	p5.combos = append(p5.combos, res)
	p5.mu.Unlock()
	if !res.Pass {
		t.Errorf("p5combo.close-error: %s; gate problems %v", res.Reason, g.Problems)
	}
}

// p5InnerResult is a result the gate kept of its inner run, nil if none.
func p5InnerResult(g gateResult, id string) *result {
	for i := range g.Inner {
		if g.Inner[i].ID == id {
			return &g.Inner[i]
		}
	}
	return nil
}
