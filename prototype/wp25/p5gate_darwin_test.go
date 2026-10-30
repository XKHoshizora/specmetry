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

// p5GateCase is a gate with the target's planned calls (§16.6).
type p5GateCase struct {
	gateCase
	calls int
}

// p5Gates are §16.6's gate and its control. TestGate's list is unchanged.
var p5Gates = []p5GateCase{
	{gateCase{id: "p5.eintr-once", test: "TestP5$/^(layout.1|w.exit7.1)", scenario: "P5.w.exit7.1", inject: "p5-eintr-once", want: invalid, goos: []string{"darwin"}, hit: true, hits: 1, exact: true, because: "observation failed: waitid 1: interrupted system call"}, 1},
	{gateCase{id: "control.p5-w-exit7", test: "TestP5$/^(layout.1|w.exit7.1)", scenario: "P5.w.exit7.1", want: confirmed, goos: []string{"darwin"}, exact: true}, 2},
}

// p5GateInfo is a P-5 gate's own record (§16.11): its deadlines and its
// knowledge of the inner run when it returned.
type p5GateInfo struct {
	S        string   `json:"s"`
	Dg       string   `json:"d_g"`
	Di       string   `json:"d_i"`
	K        string   `json:"k"`
	W        string   `json:"w"`
	Pid      int      `json:"pid,omitempty"`
	Identity string   `json:"identity,omitempty"` // the inner run's start time, "" if unknown
	Know     *p5Know  `json:"inner_state,omitempty"`
	Exit     string   `json:"exit"` // the inner run's exit, or unknown
	Output   string   `json:"output"`
	Handed   []string `json:"handed,omitempty"`
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
// problems met; a closed write end's error is a T, the start goes on.
func p5Launch(test, inject string, di time.Time, env []string) (*p5Inner, *exec.Cmd, *os.File, []string) {
	self, err := os.Executable()
	if err != nil {
		return nil, nil, nil, []string{"inner: os.Executable: " + err.Error()}
	}
	rd, wr, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, []string{"inner: os.Pipe: " + err.Error()}
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
		return nil, nil, nil, probs
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
	p := &p5Inner{pid: pid, ops: ops, ctl: &p5Ctl{}}
	p.wait = p5StartWait(p.ctl, ops)
	p.copy = p5StartCopy(rd)
	return p, cmd, rd, probs
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
	inner, cmd, rd, probs := p5Launch(c.test, c.inject, di, []string{tag, injectLogEnv + "=" + il, rw.env()})
	g.Problems = append(g.Problems, probs...)
	if inner == nil {
		g.Reason = "inner run: " + strings.Join(probs, "; ")
		g.Remedy, g.Final = "not-run", "unknown"
		return finishGate(c.gateCase, g, false, began)
	}
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
	complete := cs.Done
	if !cs.Done {
		// The read end's stop, registered and started; one query that does
		// not wait.
		inner.stop = p5StartStop(rd.Close)
		cs = inner.copy.snap()
		inner.stop.snap()
		g.Problems = append(g.Problems, "inner: the output was not complete by d_i + 10 s")
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
	var remedyErr error
	innerOpen := !inner.wait.snap().Done
	if len(act) > 0 || innerAlive || innerOpen {
		g.Remedy = "not-run"
		rdl := minTime(time.Now().Add(10*time.Second), dg)
		still, err := p5Remediate(act, tag, rdl)
		remedyErr = err
		if innerOpen {
			if q := inner.request("gate", "remedy"); q != nil {
				q.until(rdl)
			}
			innerOpen = !inner.wait.until(rdl).Done
		}
		for _, v := range still {
			g.Still = append(g.Still, v.pid)
		}
		switch {
		case err != nil, len(g.Still) > 0, innerOpen:
			g.Remedy = "failed"
		default:
			g.Remedy = "succeeded"
		}
		if err != nil {
			g.RemedyErr = err.Error()
			g.Problems = append(g.Problems, "still-alive: the remediation could not confirm its end: "+err.Error())
		}
		if len(g.Still) > 0 {
			g.Problems = append(g.Problems, fmt.Sprintf("still-alive: %d processes still alive after the remediation: %v", len(g.Still), g.Still))
		}
		if innerOpen {
			g.Problems = append(g.Problems, "still-alive: the inner run's exit was not confirmed by the remediation's deadline")
		}
	}
	unknownN := 0
	for _, st := range state {
		if st == "unknown" {
			unknownN++
		}
	}
	switch {
	case len(g.Still) > 0:
		g.Final = "alive"
	case cerr != nil || unknownN > 0 || remedyErr != nil || len(refused) > 0 || innerOpen:
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
	g.Insurance = ins
	switch {
	case g.Final == "ended" && ins != "nothing left":
		g.Problems = append(g.Problems, "insurance: the tested path found the round ended, the insurance cleanup did not: "+ins)
		judge(c.gateCase, &g, verdictOK)
	case strings.HasPrefix(ins, "FAILED"):
		g.Problems = append(g.Problems, "insurance: not confirmed by d_g: "+ins)
		judge(c.gateCase, &g, verdictOK)
	}

	// What is not confirmed is handed on to the final cleanup, whole.
	k2 := inner.know()
	info.Know = &k2
	pending := !k2.WaitDone || !k2.OutputDone || k2.Stop == "pending" || inner.pendingReq() != nil
	if pending || len(left) > 0 {
		h := &p5Hand{gate: c.id, tag: tag, since: since, others: left}
		if pending {
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

// p5Remediate ends this round's proven processes (never the inner run),
// each only while its identity holds and it carries the tag, and gives the
// ones still alive at the deadline.
func p5Remediate(ss []survivor, tag string, deadline time.Time) ([]survivor, error) {
	for _, v := range ss {
		if st, pr := procInfo(v.pid); pr == present && st.start == v.start && !st.zombie && tagged(v.pid, tag) {
			_ = outerSignal(v.pid)
		}
	}
	for {
		var still []survivor
		for _, v := range ss {
			a, err := recordAlive(witnessRec{pid: v.pid, start: v.start})
			if err != nil {
				return nil, err
			}
			if a {
				still = append(still, v)
			}
		}
		if len(still) == 0 || !time.Now().Before(deadline) {
			return still, nil
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// p5Insure is the insurance cleanup of a P-5 gate: insure's rules, within
// the gate's deadline, the inner run apart: it is never signalled by its
// PID here, only by a request task (R0), and its end is the Wait task's.
// It gives what is still alive of the rest, to hand on, and its account.
func p5Insure(inner *p5Inner, known []survivor, tag, since string, deadline time.Time) ([]survivor, string) {
	if !inner.wait.snap().Done {
		if q := inner.request("gate", "insurance"); q != nil {
			q.until(deadline)
		}
	}
	left, found, err := p5EndProven(known, tag, since, deadline, inner)
	innerDone := inner.wait.until(deadline).Done
	var parts []string
	if !innerDone {
		parts = append(parts, "the inner run's exit not confirmed")
	}
	if len(left) > 0 {
		var keys []string
		for _, v := range left {
			keys = append(keys, v.key())
		}
		parts = append(parts, fmt.Sprintf("%d still alive: %v", len(left), keys))
	}
	if err != nil {
		parts = append(parts, "read or census: "+err.Error())
	}
	switch {
	case len(parts) > 0:
		return left, "FAILED: " + strings.Join(parts, "; ")
	case found > 0:
		return nil, fmt.Sprintf("ended %d", found)
	}
	return nil, "nothing left"
}
