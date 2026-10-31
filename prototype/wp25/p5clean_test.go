//go:build darwin || wp25p5local

package wp25

// P-5's cleanup chains and the real-child holder (docs/wp2.5-prototype.md
// §16.17.3, the fourth draft), apart from the platform like the rest of the
// protocol core (p5proto_test.go): darwin gives them their reads and calls
// (p5gate_darwin_test.go, p5real_darwin_test.go), the model checks fakes.
//
// The cleanup chain is what p5EndProven and p5Remediate were: a set of
// this round's processes, each read once per round by its recorded
// identity, ended only while it runs, proven this round's and carrying
// the tag in the same step; every read, tag read, census and signal that
// fails is kept to the call's end, apart from what is known of the ends.
//
// The holder is one side of TestP5CleanupReal: its child, the tested
// chain's task, the request tasks (R0 to R5) and the one Wait task (W1 to
// W6) under one control lock, as the gate's inner run holds its own; its
// decision steps, the normal path's steps 3 and 4 and the cleanup's steps
// 1 to 6, re-read the clock at every decision point (a new request, the
// unstarted Wait, the chain's start) and take the expiry exit once a
// deadline has passed. What is unconfirmed at the cleanup's end stays held
// by p5RealHeld.

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// The classes of one read of a recorded identity v = (pid, start time).
const (
	p5Absent  = "absent"           // v has ended
	p5Changed = "identity changed" // v has ended; its PID is another process's
	p5Zombie  = "zombie"           // v no longer runs (recordAlive's rule)
	p5Alive   = "alive"            // v runs: the only class an action may follow
	p5Unknown = "unknown"          // not known: an error, kept; never an end
)

// p5Ended says whether a class is a reliable end.
func p5Ended(class string) bool { return class == p5Absent || class == p5Changed || class == p5Zombie }

// p5CleanOps are a cleanup chain's reads and calls. darwin binds the real
// ones; a regression's seam wraps a real read, which still happens, and
// answers for it once.
type p5CleanOps struct {
	// read reads v once and gives its class, with unknown its error. at
	// says which read it is: "entry", "round" or "recheck" (right after a
	// tag read failed).
	read func(v survivor, at string) (string, error)
	// tagged reads v's environment for this round's tag: found or not, or
	// the read's error.
	tagged func(pid int) (bool, error)
	// census lists this round's processes by the tag; nil for the
	// remediation, which has none.
	census func() ([]survivor, error)
	// kill is one kill(pid, SIGKILL); its raw errno is kept.
	kill func(pid int) error
	// skip is the inner run: never in the set, never acted on here (only
	// its request tasks signal it). nil: nothing is skipped.
	skip  func(v survivor) bool
	now   func() time.Time
	every time.Duration
}

// p5CleanResult keeps the knowledge of the ends apart from the validity.
type p5CleanResult struct {
	Found   int        `json:"found"`            // in the set at the start: alive or unknown
	Ended   []string   `json:"ended,omitempty"`  // "key: class", in the order they ended
	Alive   []survivor `json:"-"`                // alive at the deadline
	Unknown []survivor `json:"-"`                // unknown at the deadline
	Kills   int        `json:"kills"`            // kill returned 0: requests, not ends
	Notes   string     `json:"notes,omitempty"`  // ESRCH, an end during a tag read: no errors
	Errs    error      `json:"-"`                // every failure of the call, kept to its end
	Left    []string   `json:"left,omitempty"`   // Alive and Unknown, for the record
	ErrText string     `json:"errors,omitempty"` // Errs, for the record
	// Failures are the same failures as entries, each with its source
	// (read, tag, kill, census) and whether a seam planned it (§16.21.3).
	Failures []p5Failure `json:"failures,omitempty"`
	Classes  []string    `json:"classes,omitempty"` // every read's class in order: "key at: class"
}

// p5EndChain is the cleanup chain (§16.17.3). The set: each known process
// unless its first read is a reliable end (an unknown one enters, its
// error kept), then what the census proves this round's (its error kept).
// Each round reads every process in the set once: an end leaves it; an
// unknown stays, its error kept; one alive is checked for the tag and only
// then sent one kill. It returns once the set is empty or at the deadline,
// what is alive or unknown then with it. An unknown adds no process and
// authorizes no action. One accumulator serves the whole call; a later
// success clears nothing.
func p5EndChain(known []survivor, deadline time.Time, ops p5CleanOps) p5CleanResult {
	var res p5CleanResult
	var errs, notes p5Errs
	set := map[string]survivor{}
	var order []string
	enter := func(v survivor) {
		if _, ok := set[v.key()]; ok || ops.skip != nil && ops.skip(v) {
			return
		}
		set[v.key()] = v
		order = append(order, v.key())
	}
	read := func(v survivor, at string) (string, error) {
		class, err := ops.read(v, at)
		res.Classes = append(res.Classes, v.key()+" "+at+": "+class)
		return class, err
	}
	for _, v := range known {
		if ops.skip != nil && ops.skip(v) {
			continue
		}
		switch class, err := read(v, "entry"); {
		case class == p5Unknown:
			errs.note("read", err)
			enter(v)
		case !p5Ended(class):
			enter(v)
		}
	}
	if ops.census != nil {
		ss, err := ops.census()
		errs.note("census", err)
		for _, v := range ss {
			enter(v)
		}
	}
	res.Found = len(set)
	for {
		var alive, unknown []survivor
		var next []string
		for _, key := range order {
			v := set[key]
			class, err := read(v, "round")
			switch {
			case p5Ended(class):
				res.Ended = append(res.Ended, key+": "+class)
				continue
			case class == p5Unknown:
				errs.note("read", err)
				unknown = append(unknown, v)
			default:
				alive = append(alive, v)
				act, note, err := p5TagCheck(v, ops, read)
				notes.note("", note)
				errs.note("tag", err)
				if act {
					switch err := ops.kill(v.pid); {
					case err == nil:
						res.Kills++
					case errors.Is(err, syscall.ESRCH):
						notes.note("", fmt.Errorf("process %s: kill gave ESRCH: it ended after the check; its end is still read", key))
					default:
						errs.note("kill", fmt.Errorf("process %s: kill(%d, SIGKILL) failed: %w", key, v.pid, err))
					}
				}
			}
			next = append(next, key)
		}
		order = next
		if len(order) == 0 || !ops.now().Before(deadline) {
			res.Alive, res.Unknown, res.Errs = alive, unknown, errs.err()
			for _, v := range append(append([]survivor(nil), alive...), unknown...) {
				res.Left = append(res.Left, v.key())
			}
			if res.Errs != nil {
				res.ErrText = res.Errs.Error()
			}
			res.Failures = errs.entries("")
			if n := notes.err(); n != nil {
				res.Notes = n.Error()
			}
			return res
		}
		time.Sleep(ops.every)
	}
}

// p5TagCheck is the check before an action on a process read alive: its
// tag read now. Found: it may be acted on. Read but not found: no action,
// an error (a process proven this round's no longer carries the tag). The
// read failed: its identity read again at once, censusTag's rule; a
// reliable end then is an end during the read (a note, no error), anything
// else an error. Either way no action.
func p5TagCheck(v survivor, ops p5CleanOps, read func(survivor, string) (string, error)) (act bool, note, err error) {
	ok, terr := ops.tagged(v.pid)
	switch {
	case terr == nil && ok:
		return true, nil, nil
	case terr == nil:
		return false, nil, fmt.Errorf("process %s: proven this round's, its environment no longer carries the tag: not acted on", v.key())
	}
	class, rerr := read(v, "recheck")
	if p5Ended(class) {
		return false, fmt.Errorf("process %s: ended during the read of its tag (%s)", v.key(), class), nil
	}
	if rerr != nil {
		return false, nil, fmt.Errorf("process %s: its tag cannot be read (%v), and its state is unknown (%v): not acted on", v.key(), terr, rerr)
	}
	return false, nil, fmt.Errorf("process %s: its tag cannot be read (%v), and it is %s: not acted on", v.key(), terr, class)
}

func p5Keys(vs []survivor) []string {
	var keys []string
	for _, v := range vs {
		keys = append(keys, v.key())
	}
	return keys
}

// The chain's result, consumed (§16.17.3): small functions the callers use
// as they are, and the regressions check.

// p5InsureAccount is the insurance cleanup's account: FAILED, with every
// cause, when the inner run's exit is not confirmed, a process is alive or
// unknown, or the call met any error, even with every process ended; only
// without any is it "ended n" or "nothing left".
func p5InsureAccount(innerDone bool, res p5CleanResult) string {
	var parts []string
	if !innerDone {
		parts = append(parts, "the inner run's exit not confirmed")
	}
	if len(res.Alive) > 0 {
		parts = append(parts, fmt.Sprintf("%d still alive: %v", len(res.Alive), p5Keys(res.Alive)))
	}
	if len(res.Unknown) > 0 {
		parts = append(parts, fmt.Sprintf("%d of unknown state: %v", len(res.Unknown), p5Keys(res.Unknown)))
	}
	if res.Errs != nil {
		parts = append(parts, "reads, signals or census: "+res.Errs.Error())
	}
	switch {
	case len(parts) > 0:
		return "FAILED: " + strings.Join(parts, "; ")
	case res.Found > 0:
		return fmt.Sprintf("ended %d", res.Found)
	}
	return "nothing left"
}

// p5AfterInsurance is the gate's judgment after its insurance cleanup: the
// tested path's ended state contradicted, or the insurance FAILED, is an
// "insurance" problem, which no expected rejection excuses, and the gate
// is judged again.
func p5AfterInsurance(c gateCase, g *gateResult, verdictOK bool, ins string) {
	g.Insurance = ins
	switch {
	case g.Final == "ended" && ins != "nothing left":
		g.Problems = append(g.Problems, "insurance: the tested path found the round ended, the insurance cleanup did not: "+ins)
		judge(c, g, verdictOK)
	case strings.HasPrefix(ins, "FAILED"):
		g.Problems = append(g.Problems, "insurance: not confirmed by d_g: "+ins)
		judge(c, g, verdictOK)
	}
}

// p5Remedy is the remediation's outcome.
type p5Remedy struct {
	Remedy   string   // succeeded or failed
	Err      string   // the call's errors, for RemedyErr
	Problems []string // still-alive problems, which no expected rejection excuses
	Still    []int    // alive at the deadline
	Final    string   // what it leaves the final state: alive, unknown or ended
}

// p5RemedyOutcome judges the remediation's result: its errors, a process
// still alive or unknown, or the inner run's exit not confirmed make it
// failed; errors and unknowns make the final state unknown, a process
// alive makes it alive.
func p5RemedyOutcome(res p5CleanResult, innerOpen bool) p5Remedy {
	out := p5Remedy{Remedy: "succeeded", Final: "ended"}
	for _, v := range res.Alive {
		out.Still = append(out.Still, v.pid)
	}
	if res.Errs != nil {
		out.Err = res.Errs.Error()
		out.Problems = append(out.Problems, "still-alive: the remediation could not confirm its end: "+out.Err)
	}
	if len(out.Still) > 0 {
		out.Problems = append(out.Problems, fmt.Sprintf("still-alive: %d processes still alive after the remediation: %v", len(out.Still), out.Still))
	}
	if len(res.Unknown) > 0 {
		out.Problems = append(out.Problems, fmt.Sprintf("still-alive: %d processes of unknown state after the remediation: %v", len(res.Unknown), p5Keys(res.Unknown)))
	}
	if innerOpen {
		out.Problems = append(out.Problems, "still-alive: the inner run's exit was not confirmed by the remediation's deadline")
	}
	switch {
	case len(out.Still) > 0:
		out.Final = "alive"
	case res.Errs != nil || len(res.Unknown) > 0 || innerOpen:
		out.Final = "unknown"
	}
	if len(out.Problems) > 0 {
		out.Remedy = "failed"
	}
	return out
}

// p5FinalProcs is the processes' part of a final cleanup item: what is
// alive or unknown at the deadline, and the call's errors, each an
// unknown of the item, which make final_T true.
func p5FinalProcs(res p5CleanResult) (processes, unknown []string) {
	processes = append(processes, p5Keys(res.Alive)...)
	for _, v := range res.Unknown {
		processes = append(processes, v.key()+" (unknown)")
	}
	if res.Errs != nil {
		unknown = append(unknown, "the processes of the round: "+res.Errs.Error())
	}
	if len(res.Alive) > 0 {
		unknown = append(unknown, fmt.Sprintf("%d processes of the round still alive", len(res.Alive)))
	}
	if len(res.Unknown) > 0 {
		unknown = append(unknown, fmt.Sprintf("%d processes of the round of unknown state", len(res.Unknown)))
	}
	return processes, unknown
}

// p5Task is a task of its own (the tested chain): registered first, then
// started; its result written under its state lock before its completion
// is published, and kept.
type p5Task struct {
	done chan struct{}
	mu   sync.Mutex
	fin  bool
	val  any
	err  error
}

func (k *p5Task) run(f func() (any, error)) {
	v, err := f()
	k.mu.Lock()
	k.fin, k.val, k.err = true, v, err
	k.mu.Unlock()
	close(k.done)
}

func (k *p5Task) snap() (bool, any, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.fin, k.val, k.err
}

func (k *p5Task) until(deadline time.Time) (bool, any, error) {
	p5Until(k.done, deadline)
	return k.snap()
}

// p5HolderOps are a holder's reads and calls.
type p5HolderOps struct {
	// now is the clock read at every decision point.
	now func() time.Time
	// probe is the parent's own synchronous read: true when the child is
	// this process's reliable unreaped zombie; false when it is not; an
	// error when not known.
	probe func() (bool, error)
	// observe, wait, state and kill are the Wait task's and the request
	// tasks' (p5Ops).
	observe func() (bool, error)
	wait    func() error
	state   func() p5Exit
	kill    func() error
	every   time.Duration
}

// p5Holder is one side's holder (§16.17.3 R-B2-2): registered right after
// its child started, before anything else is read or asserted. Only the
// side's test goroutine and its cleanup use it, one after the other; its
// tasks never touch the test.
type p5Holder struct {
	side       string
	pid        int
	cmd        any // the child's Cmd: held, used only by the Wait task's ops
	ops        p5HolderOps
	ctl        *p5Ctl
	wait       *p5Wait // nil: not started
	reqs       []*p5Req
	chain      *p5Task // nil: not started
	c, q, w, d time.Time
	held       *p5HeldReg
	entered    bool
	fails      []string // what made the side fail, in order
	expired    []string // the decision points where the expiry exit was taken
	handed     *p5HeldItem
	status     string      // the reaped status, once confirmed by the cleanup
	probes     []p5Failure // its own probe's failures, appended, never cleared
}

// probeFailed keeps a failure of the holder's own probe (§16.21.2).
func (h *p5Holder) probeFailed(site string, err error) {
	h.probes = append(h.probes, p5Failure{Source: "probe", Site: site, Text: err.Error(), Count: 1, Planned: planned(err)})
}

// p5HolderFailures is the holder's one consumption of its tool failures
// (§16.21.2), from its own record and its tasks' snapshots: its probe's
// failures, the Wait task's failed reads (W1, kept by the task), Cmd.Wait's
// own error (an exit status is no error of it), and each completed request
// task's raw error. Each source only grows: no later success clears an
// entry. The side's judgement, the hand-over's knowledge and the held
// registry's current view all use it; the knowledge of the reap is kept
// apart from it.
func p5HolderFailures(probes []p5Failure, wait *p5Wait, reqs []*p5Req) []p5Failure {
	out := append([]p5Failure(nil), probes...)
	if wait != nil {
		s := wait.snap()
		out = append(out, s.Fails...)
		if s.Done && s.WaitErr != nil {
			out = append(out, p5Failure{Source: "wait", Text: s.WaitErr.Error(), Count: 1, Planned: planned(s.WaitErr)})
		}
	}
	for _, q := range reqs {
		if r := q.snap(); r.Err != "" {
			out = append(out, p5Failure{Source: "request", Site: r.Holder + "/" + r.Step, Text: r.Err, Count: 1})
		}
	}
	return out
}

func (h *p5Holder) failures() []p5Failure { return p5HolderFailures(h.probes, h.wait, h.reqs) }

func (h *p5Holder) fail(format string, args ...any) {
	h.fails = append(h.fails, fmt.Sprintf(format, args...))
}

// decide is a decision point: the clock read again now and compared with
// the path's deadline. Nothing read before stands in for it: a query that
// returned done, or a read that began in time. Past the deadline (now not
// before it) the caller takes the expiry exit.
func (h *p5Holder) decide(deadline time.Time) bool { return h.ops.now().Before(deadline) }

func (h *p5Holder) pendingReq() *p5Req {
	if n := len(h.reqs); n > 0 {
		select {
		case <-h.reqs[n-1].done:
		default:
			return h.reqs[n-1]
		}
	}
	return nil
}

// request is R0 by the holder: none needed once the latest snapshot of the
// Wait task shows reaping started (a Wait not started has not); a pending
// one is only queried, not joined by another; otherwise a new task, its
// creation a decision point: registered first, then started, only while
// the deadline has not passed. expired: the deadline had passed, nothing
// was started.
func (h *p5Holder) request(step string, deadline time.Time) (*p5Req, bool) {
	if h.wait != nil && !h.wait.snap().Seen.IsZero() {
		return nil, false
	}
	if q := h.pendingReq(); q != nil {
		return q, false
	}
	if !h.decide(deadline) {
		return nil, true
	}
	q := &p5Req{holder: h.side, step: step, done: make(chan struct{})}
	h.reqs = append(h.reqs, q)
	go q.run(h.ctl, h.ops.kill)
	return q, false
}

// startWait starts the one Wait task, its start a decision point.
func (h *p5Holder) startWait(deadline time.Time) bool {
	if h.wait != nil {
		return false
	}
	if !h.decide(deadline) {
		return true
	}
	h.wait = p5StartWait(h.ctl, p5Ops{observe: h.ops.observe, wait: h.ops.wait, state: h.ops.state, kill: h.ops.kill, every: h.ops.every})
	return false
}

// chainStep is the normal path's step 2: the tested chain in its own task,
// registered first, started only before c_k (a decision point), queried by
// q_k. Until it returns no request is made and no Wait started: the chain
// signals by PID, and the child is not reaped while it runs. It says
// whether the chain returned by q_k.
func (h *p5Holder) chainStep(f func() (any, error)) bool {
	if !h.decide(h.c) {
		h.fail("not executed: c_k passed at the decision point of the chain's start")
		return false
	}
	k := &p5Task{done: make(chan struct{})}
	h.chain = k
	go k.run(f)
	if done, _, _ := k.until(h.q); !done {
		h.fail("the tested chain had not returned by q_k: no request, no Wait; the cleanup holds it")
		return false
	}
	return true
}

// tail is the normal path's steps 3 and 4. probeFirst: after the chain,
// the child is read first, and only one still running or unknown needs the
// holder's request (the chain did not end it); without it (the identity
// read failed, no chain ran) the request is made at once. Each new task is
// a decision point against w_k. expired: the expiry exit was taken; what
// is unconfirmed stays with the holder for the cleanup, decided against
// d_k.
func (h *p5Holder) tail(probeFirst bool) bool {
	need := true
	if probeFirst {
		switch zombie, err := h.ops.probe(); {
		case err != nil:
			h.probeFailed("normal", err)
			h.fail("the child's state after the chain is unknown (%v): the chain did not end it", err)
		case !zombie:
			h.fail("the child still runs after the chain: the chain did not end it")
		default:
			need = false
		}
	}
	if need {
		if _, expired := h.request("normal", h.w); expired {
			h.expire("the normal path", "w_k", "a new request")
			return true
		}
	}
	if h.startWait(h.w) {
		h.expire("the normal path", "w_k", "the Wait task's start")
		return true
	}
	h.wait.until(h.w)
	if n := len(h.reqs); n > 0 {
		h.reqs[n-1].until(h.w)
	}
	return false
}

// expire records the expiry exit taken at a decision point: a snapshot
// only, a T; no new request, no Wait, no new budget.
func (h *p5Holder) expire(path, deadline, what string) {
	h.expired = append(h.expired, path+": "+what)
	h.fail("%s: deadline %s passed at the decision point (%s): the expiry exit; %s", path, deadline, what, h.know().line())
}

// cleanup is the side's cleanup (§16.17.3), registered on the side's own t
// right after the holder: run once (its entered mark guards a second
// call). Every decision point is against d_k; past it, the expiry exit
// hands what is unconfirmed to p5RealHeld. It never waits beyond d_k, and
// never retries.
func (h *p5Holder) cleanup() {
	if h.entered {
		return
	}
	h.entered = true
	// 1. On entry: past d_k is the expiry exit.
	if !h.decide(h.d) {
		h.cleanupExpire("on entry")
		return
	}
	// 2. The chain, started and not returned, queried once more. A done
	// query is no proof that time is left.
	if h.chain != nil {
		if done, _, _ := h.chain.until(h.d); !done {
			h.fail("the cleanup: the tested chain had still not returned by d_k: no request, no Wait")
			h.handOver("the tested chain had not returned by d_k")
			return
		}
	}
	// 3. The Wait task not started: the child read once; a request only
	// if it is not a reliable zombie and none is pending, then the Wait,
	// each a decision point.
	if h.wait == nil {
		zombie, err := h.ops.probe()
		if err != nil {
			h.probeFailed("cleanup", err)
		}
		if err != nil || !zombie {
			if _, expired := h.request("cleanup", h.d); expired {
				h.cleanupExpire("a new request")
				return
			}
		}
		if h.startWait(h.d) {
			h.cleanupExpire("the Wait task's start")
			return
		}
	}
	// 4. Queries only, by d_k.
	ws := h.wait.until(h.d)
	for _, q := range h.reqs {
		q.until(h.d)
	}
	// 5. All confirmed: the knowledge of the reap only; the tools' validity
	// is the judgement's, from failures().
	if ws.Done && h.pendingReq() == nil {
		h.status = ws.Exit.Status
		return
	}
	// 6. Otherwise what is unconfirmed is handed on.
	h.fail("the cleanup: not confirmed by d_k: %s", strings.Join(h.unconfirmed(), "; "))
	h.handOver("not confirmed by d_k")
}

// cleanupExpire is the cleanup's expiry exit, the same one on entry and
// when a step crosses d_k.
func (h *p5Holder) cleanupExpire(what string) {
	h.expire("the cleanup", "d_k", what)
	if u := h.unconfirmed(); len(u) > 0 {
		h.handOver("the expiry exit (" + what + "): " + strings.Join(u, "; "))
	}
}

// unconfirmed lists the duties not discharged.
func (h *p5Holder) unconfirmed() []string {
	var u []string
	if h.chain != nil {
		if done, _, _ := h.chain.snap(); !done {
			u = append(u, "the tested chain has not returned")
		}
	}
	switch {
	case h.wait == nil:
		u = append(u, "the Wait task was not started: the child is not reaped")
	case !h.wait.snap().Done:
		u = append(u, "the Wait task has not completed")
	}
	if h.pendingReq() != nil {
		u = append(u, "a request task is pending")
	}
	return u
}

// handOver hands the side's unconfirmed duties to the held registry: its
// Cmd, pid and tasks, with the knowledge at that moment, fixed.
func (h *p5Holder) handOver(reason string) {
	it := &p5HeldItem{Side: h.side, Pid: h.pid, cmd: h.cmd, Reason: reason, chain: h.chain, reqs: append([]*p5Req(nil), h.reqs...), wait: h.wait,
		probes: append([]p5Failure(nil), h.probes...), Handed: h.know()}
	h.held.add(it)
	h.handed = it
}

// The four reap states (§16.17.3, N-114-2), from the Wait task's own
// state: the holder never takes the control lock, so the instant between
// W2's mark and W3's record shows as not started.
const (
	p5ReapNone    = "Wait not started"
	p5ReapWaiting = "Wait started, reaping not started"
	p5ReapStarted = "reaping started, not confirmed"
	p5ReapDone    = "reliably reaped"
)

// p5HeldKnow is a holder's knowledge, from snapshots only.
type p5HeldKnow struct {
	At       string      `json:"at"`
	Reap     string      `json:"reap"`
	Exit     string      `json:"exit,omitempty"`
	Chain    string      `json:"chain"` // not started, pending, returned
	ChainErr string      `json:"chain_error,omitempty"`
	Requests []p5ReqSnap `json:"requests"`
	// Failures is the unified consumption at that moment (§16.21.2).
	Failures []p5Failure `json:"failures,omitempty"`
}

func (k p5HeldKnow) line() string {
	return fmt.Sprintf("reap: %s; chain: %s; requests: %d", k.Reap, k.Chain, len(k.Requests))
}

func p5Knowledge(chain *p5Task, reqs []*p5Req, wait *p5Wait, probes []p5Failure) p5HeldKnow {
	k := p5HeldKnow{At: time.Now().Format(time.RFC3339Nano), Reap: p5ReapNone, Chain: "not started", Requests: []p5ReqSnap{},
		Failures: p5HolderFailures(probes, wait, reqs)}
	if wait != nil {
		s := wait.snap()
		switch {
		case s.Done:
			k.Reap, k.Exit = p5ReapDone, s.Exit.Status
		case !s.Seen.IsZero():
			k.Reap = p5ReapStarted
		default:
			k.Reap = p5ReapWaiting
		}
	}
	if chain != nil {
		k.Chain = "pending"
		if done, _, err := chain.snap(); done {
			k.Chain = "returned"
			if err != nil {
				k.ChainErr = err.Error()
			}
		}
	}
	for _, q := range reqs {
		k.Requests = append(k.Requests, q.snap())
	}
	return k
}

func (h *p5Holder) know() p5HeldKnow { return p5Knowledge(h.chain, h.reqs, h.wait, h.probes) }

// p5HeldItem is one side handed to the registry: it keeps the references
// to the side's Cmd and tasks until the test process ends. Its knowledge
// at the hand-over is never rewritten; the current snapshot is taken apart.
type p5HeldItem struct {
	Side   string
	Pid    int
	Reason string
	Handed p5HeldKnow
	cmd    any
	chain  *p5Task
	reqs   []*p5Req
	wait   *p5Wait
	probes []p5Failure
}

// p5HeldReg is the held registry: its lock guards only the list (no task
// is read under it).
type p5HeldReg struct {
	mu    sync.Mutex
	items []*p5HeldItem
}

// p5RealHeld is this test process's registry of TestP5CleanupReal (apart
// from p5.handed: TestP5's final cleanup and that check run in no fixed
// order).
var p5RealHeld p5HeldReg

func (r *p5HeldReg) add(it *p5HeldItem) {
	r.mu.Lock()
	r.items = append(r.items, it)
	r.mu.Unlock()
}

func (r *p5HeldReg) list() []*p5HeldItem {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*p5HeldItem(nil), r.items...)
}

// p5HeldView is what is printed of one item: the knowledge at its
// hand-over and a current snapshot, which never waits. A late reap in the
// current snapshot restores nothing: the side's T was decided at the
// hand-over. The views are for the record only: no PID action or verdict
// reads them.
type p5HeldView struct {
	Side   string     `json:"side"`
	Pid    int        `json:"pid"`
	Reason string     `json:"reason"`
	Handed p5HeldKnow `json:"handed"`
	Now    p5HeldKnow `json:"now"`
}

func (r *p5HeldReg) view() []p5HeldView {
	out := []p5HeldView{}
	for _, it := range r.list() {
		out = append(out, p5HeldView{it.Side, it.Pid, it.Reason, it.Handed, p5Knowledge(it.chain, it.reqs, it.wait, it.probes)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Side < out[j].Side })
	return out
}

// p5ReadSeam is the real regression's seam on a chain's read
// (§16.21.3): the real read happens first and is kept; only when it gave
// alive without an error is its answer replaced, once, by an unknown with
// a planned error. Otherwise nothing is replaced: the real result goes on
// as it is, a real failure stays unplanned, and the premise's failure is
// kept. The chain runs in its own task; the side reads the seam after.
type p5ReadSeam struct {
	mu      sync.Mutex
	v       survivor
	hits    int
	real    string // the real read the seam met at its site
	premise string // why the premise failed, "" while it holds
}

func (s *p5ReadSeam) wrap(read func(survivor, string) (string, error)) func(survivor, string) (string, error) {
	return func(v survivor, at string) (string, error) {
		class, err := read(v, at)
		s.mu.Lock()
		defer s.mu.Unlock()
		if v != s.v || at != "round" || s.hits > 0 || s.real != "" {
			return class, err
		}
		s.real = class
		if err != nil {
			s.real += " (" + err.Error() + ")"
		}
		if err != nil || class != p5Alive {
			s.premise = "the real read at the seam's site gave " + s.real + ", not alive: nothing replaced"
			return class, err
		}
		s.hits++
		return p5Unknown, &p5Planned{"the first round's read of " + v.key() + " answered unknown"}
	}
}

func (s *p5ReadSeam) state() (hits int, real, premise string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits, s.real, s.premise
}

// p5ObserveSeam is the seam on the Wait task's W1 read (§16.21.2): the
// real read happens first; only when it succeeded is its answer replaced,
// once, by a planned error. A real failure is never replaced.
type p5ObserveSeam struct {
	mu      sync.Mutex
	hits    int
	real    string
	premise string
}

func (s *p5ObserveSeam) wrap(observe func() (bool, error)) func() (bool, error) {
	return func() (bool, error) {
		z, err := observe()
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.hits > 0 || s.real != "" {
			return z, err
		}
		s.real = fmt.Sprintf("zombie %v", z)
		if err != nil {
			s.real = "failed (" + err.Error() + ")"
			s.premise = "the real W1 read at the seam's site failed: nothing replaced"
			return z, err
		}
		s.hits++
		return false, &p5Planned{"the Wait task's first read failed"}
	}
}

func (s *p5ObserveSeam) state() (hits int, real, premise string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits, s.real, s.premise
}

// p5RealSpec is what a side of the real regression plans: how many times
// each seam is to be hit.
type p5RealSpec struct {
	ChainSeam int // the chain's read seam: 1, or 0 for none
	WaitSeam  int // the W1 seam: 1, or 0 for none
}

// p5RealFacts are what a side found, each from its own record.
type p5RealFacts struct {
	Chain        *p5CleanResult // nil: the chain did not return
	ChainHits    int
	ChainPremise string
	WaitHits     int
	WaitPremise  string
	Holder       []p5Failure // the holder's unified consumption
	Protocol     []string    // the holder's own T: not executed, not returned, expiry, unconfirmed
	Reaped       bool
	Status       string
	Requests     int
}

// p5Problem is one reason a side does not hold, with its code.
type p5Problem struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// p5RealJudge is a side's judgement (§16.21.3), the same for the real
// regression and the model checks. Planned failures are known by their
// type and source only, and must match the seam's hits; every unplanned
// failure of the chain, and every failure of the holder itself (planned
// or not), is a problem. Neither a text, nor a count of aggregated
// problems, nor the final success stands for "only the expected error".
func p5RealJudge(spec p5RealSpec, f p5RealFacts) []p5Problem {
	var out []p5Problem
	add := func(code, format string, args ...any) {
		out = append(out, p5Problem{code, fmt.Sprintf(format, args...)})
	}
	for _, x := range f.Protocol {
		add("protocol", "%s", x)
	}
	if f.ChainPremise != "" {
		add("seam-premise", "the chain's seam: %s", f.ChainPremise)
	}
	if f.WaitPremise != "" {
		add("seam-premise", "the W1 seam: %s", f.WaitPremise)
	}
	if f.ChainHits != spec.ChainSeam {
		add("seam-hits", "the chain's seam was hit %d times, planned %d", f.ChainHits, spec.ChainSeam)
	}
	if f.WaitHits != spec.WaitSeam {
		add("seam-hits", "the W1 seam was hit %d times, planned %d", f.WaitHits, spec.WaitSeam)
	}
	if f.Chain == nil {
		add("chain", "the chain gave no result")
	} else {
		plannedN := 0
		for _, x := range f.Chain.Failures {
			if x.Planned {
				plannedN += x.Count
				continue
			}
			add("unplanned", "the chain's %s failure (%d times): %s", x.Source, x.Count, x.Text)
		}
		if plannedN != f.ChainHits {
			add("planned-unmatched", "the chain kept %d planned failures, the seam made %d", plannedN, f.ChainHits)
		}
		if c := f.Chain; c.Found != 1 || len(c.Ended) != 1 || len(c.Left) != 0 || c.Kills < 1 {
			add("chain", "found %d, ended %v, left %v, kills %d; want 1 found, 1 ended, none left, a kill", c.Found, c.Ended, c.Left, c.Kills)
		}
	}
	holderPlanned := 0
	for _, x := range f.Holder {
		if x.Planned {
			holderPlanned += x.Count
		}
		add("holder", "the holder's %s failure%s (%d times, planned %v): %s", x.Source, map[bool]string{true: " at " + x.Site, false: ""}[x.Site != ""], x.Count, x.Planned, x.Text)
	}
	if holderPlanned != f.WaitHits {
		add("planned-unmatched", "the holder kept %d planned failures, the W1 seam made %d", holderPlanned, f.WaitHits)
	}
	if !f.Reaped || f.Status != "signal 9" {
		add("reap", "reaped %v as %q; want signal 9", f.Reaped, f.Status)
	}
	if f.Requests != 0 {
		add("requests", "the holder made %d termination requests: signal 9 proves nothing of the chain then", f.Requests)
	}
	return out
}

// p5Codes are the problems' codes, sorted.
func p5Codes(ps []p5Problem) []string {
	var cs []string
	for _, p := range ps {
		cs = append(cs, p.Code)
	}
	sort.Strings(cs)
	return cs
}
