//go:build darwin || wp25p5local

package wp25

// P-5's gate inner run as its holders hold it (docs/wp2.5-prototype.md
// §16.8, the fourth draft): the five events kept apart, the one Wait task
// (W1 to W6), the request tasks (R0 to R5), the two kinds of lock and the
// queries by an absolute deadline. It is written apart from the platform:
// the darwin gate (p5gate_darwin_test.go) gives it the inner run's reads
// and calls, and the direct checks (p5proto_check_test.go) give it fakes.
// The build tag wp25p5local builds both on another platform, for the
// author's local runs only; the diagnostic's Linux and Windows jobs build
// neither.
//
// The two kinds of lock never nest:
//
//   - the control lock (p5Ctl.mu), one per inner run, guards only the
//     reaping mark. Only the Wait task (W2) and the request tasks (R1 to
//     R3) take it; no holder (the gate, its census, remediation and
//     insurance, the final cleanup, any query) ever does. Inside it there
//     is W2's one assignment, or R2's read of the mark and at most one
//     kill;
//   - a state lock, one per task, guards only that task's own result
//     fields. While one is held there is no system call, no channel
//     operation and no other lock.
//
// Here the five events have their producers: the termination request, the
// request task; the exit reliably observed and Cmd.Wait completed, the
// Wait task; the pipe closed and the copy ended, the copy task (or the
// holder's stop of the read end).

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"syscall"
	"time"
)

// p5Ops are the inner run's reads and calls, all made by its tasks.
type p5Ops struct {
	// observe is W1's one read, which does not reap: true when the inner
	// run is this process's own unreaped zombie. An error is a read that
	// failed, or an answer that is not that (absent, another parent).
	observe func() (bool, error)
	// wait is W4's one Cmd.Wait; an exit status of its own is not an error
	// of it. state is W5's read of what it left (cmd.ProcessState): plain
	// field reads, by the Wait task only.
	wait  func() error
	state func() p5Exit
	// kill is R2's one kill(pid, SIGKILL); its raw errno is kept.
	kill func() error
	// every is W1's interval between two reads.
	every time.Duration
}

// p5Exit is what Cmd.Wait left: the status as waitStatus writes it, and
// the exit code (-1 for a signal).
type p5Exit struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// p5Ctl is one inner run's control lock and its reaping mark: "not
// started" (false) at first, it changes once, to "reaping" (true).
type p5Ctl struct {
	mu      sync.Mutex
	reaping bool
}

// p5Errs keeps the errors a task met, in order, each text once with the
// number of times it came: a read retried every few milliseconds must not
// grow without end, and a later success clears none.
type p5Errs struct {
	texts []string
	count []int
}

func (e *p5Errs) add(err error) {
	if err == nil {
		return
	}
	t := err.Error()
	if n := len(e.texts); n > 0 && e.texts[n-1] == t {
		e.count[n-1]++
		return
	}
	e.texts = append(e.texts, t)
	e.count = append(e.count, 1)
}

// err joins them; nil if there were none.
func (e p5Errs) err() error {
	if len(e.texts) == 0 {
		return nil
	}
	parts := make([]string, len(e.texts))
	for i, t := range e.texts {
		parts[i] = t
		if e.count[i] > 1 {
			parts[i] += fmt.Sprintf(" (%d times)", e.count[i])
		}
	}
	return errors.New(strings.Join(parts, "; "))
}

// p5Until is the query of §16.8: the done channel checked without
// waiting, then, if the deadline has not passed, one select between it and
// the deadline's timer. It never waits again: past the deadline it returns
// at once, whatever the task is doing, and the caller only takes a
// snapshot.
func p5Until(done <-chan struct{}, deadline time.Time) bool {
	select {
	case <-done:
		return true
	default:
	}
	d := time.Until(deadline)
	if d <= 0 {
		return false
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-done:
		return true
	case <-t.C:
		return false
	}
}

// p5Wait is the one Wait task: the only reaper of the inner run.
type p5Wait struct {
	done chan struct{} // closed by W6
	mu   sync.Mutex    // its state lock: the fields below only
	seen time.Time     // W3: the exit reliably observed, reaping started
	errs p5Errs        // W1's failed reads
	fin  bool          // W5: Cmd.Wait completed
	exit p5Exit        // with fin
	werr error         // Cmd.Wait's own error, with fin
}

// p5WaitSnap is a copy of the Wait task's fields; Exit and WaitErr mean
// something only with Done.
type p5WaitSnap struct {
	Seen    time.Time
	Done    bool
	Exit    p5Exit
	WaitErr error
	Errs    error
}

// p5StartWait starts the Wait task. It is started once, right after the
// start and the close of the write end, before anything else.
func p5StartWait(ctl *p5Ctl, ops p5Ops) *p5Wait {
	w := &p5Wait{done: make(chan struct{})}
	go w.run(ctl, ops)
	return w
}

func (w *p5Wait) run(ctl *p5Ctl, ops p5Ops) {
	// W1, no lock: read until the inner run is seen as this process's own
	// unreaped zombie. A read that fails is kept, under the state lock
	// released at once; the interval before the next read holds no lock.
	// Running, unknown, someone else's or absent: none is an exit.
	for {
		zombie, err := ops.observe()
		if err != nil {
			w.mu.Lock()
			w.errs.add(err)
			w.mu.Unlock()
		}
		if zombie && err == nil {
			break
		}
		time.Sleep(ops.every)
	}
	// W2: the mark, the one assignment in the control lock, which is
	// released at once.
	ctl.mu.Lock()
	ctl.reaping = true
	ctl.mu.Unlock()
	// W3: the observation, under the state lock.
	w.mu.Lock()
	w.seen = time.Now()
	w.mu.Unlock()
	// W4: the one Cmd.Wait, holding no lock. If it does not return, only
	// this task stops.
	err := ops.wait()
	// W5: its error, its state and the completion flag, under the state
	// lock; only this task reads cmd.ProcessState, here.
	w.mu.Lock()
	w.werr, w.exit, w.fin = err, ops.state(), true
	w.mu.Unlock()
	// W6: the completion published.
	close(w.done)
}

func (w *p5Wait) snap() p5WaitSnap {
	w.mu.Lock()
	defer w.mu.Unlock()
	s := p5WaitSnap{Seen: w.seen, Done: w.fin, Errs: w.errs.err()}
	if w.fin {
		s.Exit, s.WaitErr = w.exit, w.werr
	}
	return s
}

// until is the Wait task's query.
func (w *p5Wait) until(deadline time.Time) p5WaitSnap {
	p5Until(w.done, deadline)
	return w.snap()
}

// p5Req is one request task: the only one that signals the inner run.
type p5Req struct {
	holder string        // who registered it: the gate or the final cleanup
	step   string        // at which step: k, identity, remedy, insurance, final
	done   chan struct{} // closed by R5
	mu     sync.Mutex    // its state lock
	fin    bool
	sent   bool      // R2 called kill; false: the mark said reaping, not needed
	at     time.Time // R2's moment
	err    error     // kill's raw errno
}

// p5ReqSnap is a copy of a request task's fields.
type p5ReqSnap struct {
	Holder string `json:"holder"`
	Step   string `json:"step"`
	Result string `json:"result"` // sent, not needed, pending
	At     string `json:"at,omitempty"`
	Err    string `json:"error,omitempty"`
}

func (q *p5Req) run(ctl *p5Ctl, kill func() error) {
	// R1: the control lock. Only this task waits here.
	ctl.mu.Lock()
	// R2: the mark, and at most one kill, in the same critical section:
	// before the mark the PID is still this round's unreaped child.
	at := time.Now()
	sent := !ctl.reaping
	var err error
	if sent {
		err = kill()
	}
	// R3: the control lock released.
	ctl.mu.Unlock()
	// R4: the result, under the state lock.
	q.mu.Lock()
	q.fin, q.sent, q.at, q.err = true, sent, at, err
	q.mu.Unlock()
	// R5: the completion published.
	close(q.done)
}

func (q *p5Req) snap() p5ReqSnap {
	q.mu.Lock()
	defer q.mu.Unlock()
	s := p5ReqSnap{Holder: q.holder, Step: q.step, Result: "pending"}
	if q.fin {
		s.Result, s.At = "not needed", q.at.Format(time.RFC3339Nano)
		if q.sent {
			s.Result = "sent"
		}
		if q.err != nil {
			s.Err = q.err.Error()
		}
	}
	return s
}

func (q *p5Req) until(deadline time.Time) p5ReqSnap {
	p5Until(q.done, deadline)
	return q.snap()
}

// p5Copy is the copy task of the inner run's output: its buffer is stable
// only once the task has ended, and is read only then.
type p5Copy struct {
	done chan struct{}
	mu   sync.Mutex
	fin  bool
	buf  []byte
	err  error // a read error; a closed read end gives one
}

type p5CopySnap struct {
	Done bool
	Buf  []byte // with Done only
	Err  error
}

func p5StartCopy(r io.Reader) *p5Copy {
	c := &p5Copy{done: make(chan struct{})}
	go func() {
		var b bytes.Buffer
		_, err := b.ReadFrom(r) // to EOF, or to an error such as a closed read end
		c.mu.Lock()
		c.fin, c.buf, c.err = true, b.Bytes(), err
		c.mu.Unlock()
		close(c.done)
	}()
	return c
}

func (c *p5Copy) snap() p5CopySnap {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.fin {
		return p5CopySnap{}
	}
	return p5CopySnap{Done: true, Buf: c.buf, Err: c.err}
}

func (c *p5Copy) until(deadline time.Time) p5CopySnap {
	p5Until(c.done, deadline)
	return c.snap()
}

// p5Stop is the stop task of the read end: its close, in a task of its own
// (a close may wait for the copy's read), its error kept.
type p5Stop struct {
	done chan struct{}
	mu   sync.Mutex
	fin  bool
	err  error
}

func p5StartStop(closeRead func() error) *p5Stop {
	s := &p5Stop{done: make(chan struct{})}
	go func() {
		err := closeRead()
		s.mu.Lock()
		s.fin, s.err = true, err
		s.mu.Unlock()
		close(s.done)
	}()
	return s
}

func (s *p5Stop) snap() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fin, s.err
}

func (s *p5Stop) until(deadline time.Time) (bool, error) {
	p5Until(s.done, deadline)
	return s.snap()
}

// p5Inner is one gate's inner run, with everything that is the round's
// from its start: its tasks and the holder table of its request tasks. The
// gate holds it, then hands it on whole to the final cleanup; each holder
// uses it from its own goroutine, one after the other.
type p5Inner struct {
	pid       int
	ops       p5Ops
	ctl       *p5Ctl
	wait      *p5Wait
	copy      *p5Copy
	stop      *p5Stop
	reqs      []*p5Req // the request tasks, in the order registered
	notNeeded []string // the steps at which R0 found that none was needed
}

// request is R0, by the holder, holding no lock. A request is not needed
// once the latest snapshot of the Wait task shows reaping started; while
// the previous request task is still pending, it is queried, not joined by
// another, so there is at most one at a time; otherwise a new task is
// registered in the table first, then started: none has no owner. It gives
// the task to query, nil when none is needed.
func (p *p5Inner) request(holder, step string) *p5Req {
	if !p.wait.snap().Seen.IsZero() {
		p.notNeeded = append(p.notNeeded, holder+"/"+step)
		return nil
	}
	if n := len(p.reqs); n > 0 {
		select {
		case <-p.reqs[n-1].done:
		default:
			return p.reqs[n-1]
		}
	}
	q := &p5Req{holder: holder, step: step, done: make(chan struct{})}
	p.reqs = append(p.reqs, q)
	go q.run(p.ctl, p.ops.kill)
	return q
}

// pendingReq is the request task still pending, if any.
func (p *p5Inner) pendingReq() *p5Req {
	if n := len(p.reqs); n > 0 {
		select {
		case <-p.reqs[n-1].done:
		default:
			return p.reqs[n-1]
		}
	}
	return nil
}

// p5Know is a holder's knowledge of the inner run, from snapshots only.
type p5Know struct {
	Requests   []p5ReqSnap `json:"requests"`
	NotNeeded  []string    `json:"not_needed,omitempty"`
	ExitSeen   string      `json:"exit_seen,omitempty"` // W3's moment
	WaitDone   bool        `json:"wait_done"`
	Exit       string      `json:"exit"` // the status, or unknown
	WaitErr    string      `json:"wait_error,omitempty"`
	ObserveErr string      `json:"observe_errors,omitempty"`
	OutputDone bool        `json:"output_done"`
	OutputLen  int         `json:"output_bytes"`
	OutputErr  string      `json:"output_error,omitempty"`
	Stop       string      `json:"stop"` // none, done, pending
	StopErr    string      `json:"stop_error,omitempty"`
}

func (p *p5Inner) know() p5Know {
	k := p5Know{Requests: []p5ReqSnap{}, NotNeeded: p.notNeeded, Exit: "unknown", Stop: "none"}
	for _, q := range p.reqs {
		k.Requests = append(k.Requests, q.snap())
	}
	w := p.wait.snap()
	if !w.Seen.IsZero() {
		k.ExitSeen = w.Seen.Format(time.RFC3339Nano)
	}
	if w.Errs != nil {
		k.ObserveErr = w.Errs.Error()
	}
	if w.Done {
		k.WaitDone, k.Exit = true, w.Exit.Status
		if w.WaitErr != nil {
			k.WaitErr = w.WaitErr.Error()
		}
	}
	if p.copy != nil {
		c := p.copy.snap()
		k.OutputDone, k.OutputLen = c.Done, len(c.Buf)
		if c.Err != nil {
			k.OutputErr = c.Err.Error()
		}
	}
	if p.stop != nil {
		done, err := p.stop.snap()
		k.Stop = map[bool]string{true: "done", false: "pending"}[done]
		if err != nil {
			k.StopErr = err.Error()
		}
	}
	return k
}

// The classification of one call of entry E2 (§16.7).

// e2Form is the form of a call's result: r is int32(r1). The five are
// exclusive and cover every combination.
func e2Form(r int32, err syscall.Errno) string {
	switch {
	case r == 0 && err == 0:
		return "X1"
	case r == -1 && err != 0:
		return "X2"
	case r == -1:
		return "X3"
	case r == 0:
		return "X4"
	}
	return "X5"
}

// e2Protocol describes a protocol failure (X3 to X5): a T of entry E2
// itself, never compared with a prediction.
func e2Protocol(form string, r int32, err syscall.Errno) string {
	switch form {
	case "X3":
		return fmt.Sprintf("E2 protocol: X3: waitid returned %d without an errno", r)
	case "X4":
		return fmt.Sprintf("E2 protocol: X4: waitid returned %d with errno %d (%v)", r, int(err), err)
	}
	return fmt.Sprintf("E2 protocol: X5: waitid returned %d, neither 0 nor -1 (errno %d)", r, int(err))
}

// The classes of a real call: consistent with what it was expected to
// give, a native counterexample, or a T.
const (
	e2Consistent = "consistent"
	e2Native     = "native counterexample"
	e2Tool       = "T"
)

// e2Class gives the one class of a real call and its reason (§16.7's
// table): want is 0 for a healthy call (0 / 0 expected), the expected
// errno for a constructed error; seam marks the real call at the seam's
// site, before the seam acts. For a healthy X1 the class is consistent so
// far: its raw values are compared with the prediction apart.
func e2Class(form string, r int32, err, want syscall.Errno, seam bool) (string, string) {
	if form != "X1" && form != "X2" {
		return e2Tool, e2Protocol(form, r, err)
	}
	switch {
	case seam && form == "X1":
		return e2Consistent, "X1: the seam replaces this result by (-1, EINTR)"
	case seam:
		return e2Tool, fmt.Sprintf("the real waitid at the seam's site failed: errno %d (%v)", int(err), err)
	case want == 0 && form == "X1":
		return e2Consistent, "X1: its values are compared with the prediction"
	case want == 0:
		return e2Native, fmt.Sprintf("X2: the healthy call failed: errno %d (%v)", int(err), err)
	case form == "X1":
		return e2Native, fmt.Sprintf("X1: the call expected to fail with errno %d (%v) succeeded", int(want), want)
	case err != want:
		return e2Native, fmt.Sprintf("X2: errno %d (%v), not %d (%v)", int(err), err, int(want), want)
	}
	return e2Consistent, fmt.Sprintf("X2: errno %d (%v) as expected", int(err), err)
}

// p5CallsCheck judges a run's calls of entry E2 (§16.7): the actual count
// against the rule count (the calls the consumer's rules made), then the
// planned one. A count that differs from the rule's is a T; one equal to
// the rule's but not to the plan's is explained by that call's class, and
// gets a reason of its own only if nothing else keeps the run from being
// confirmed (confirmed needs the planned count).
func p5CallsCheck(actual, rule, plan int, otherwiseConfirmed bool) string {
	switch {
	case actual != rule:
		return fmt.Sprintf("calls: the actual count %d is not the rule count %d (planned %d)", actual, rule, plan)
	case actual != plan && otherwiseConfirmed:
		return fmt.Sprintf("calls: the actual count %d equals the rule count but not the planned %d, and no call's class explains it", actual, plan)
	}
	return ""
}
