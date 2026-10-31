//go:build darwin || wp25p5local

package wp25

// Direct checks of P-5's pure parts (docs/wp2.5-prototype.md §16.7,
// §16.8): the order and the locks of the Wait task and the request tasks,
// the query by a deadline, the copy and the stop, the classification of a
// call of entry E2 and the check of the call counts, all with fakes. They
// count no scenario and are no native evidence: model checks (§16.11).

import (
	"errors"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// fakeInner is an inner run of fakes: observe answers from a script, wait
// and kill block until released.
type fakeInner struct {
	mu       sync.Mutex
	reads    int
	zombieAt int // the read from which the inner run is a zombie
	failAt   int // the read that fails, 0 for none
	waitGo   chan struct{}
	killGo   chan struct{}
	killed   int
	killSeen []int // len(p.reqs) as each kill saw it
	p        *p5Inner
}

func newFake(zombieAt, failAt int) *fakeInner {
	f := &fakeInner{zombieAt: zombieAt, failAt: failAt, waitGo: make(chan struct{}), killGo: make(chan struct{})}
	return f
}

func (f *fakeInner) ops() p5Ops {
	return p5Ops{
		observe: func() (bool, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.reads++
			if f.reads == f.failAt {
				return false, errors.New("injected: the read failed")
			}
			return f.reads >= f.zombieAt, nil
		},
		wait:  func() error { <-f.waitGo; return nil },
		state: func() p5Exit { return p5Exit{Status: "exit 1", Code: 1} },
		kill: func() error {
			<-f.killGo
			f.mu.Lock()
			defer f.mu.Unlock()
			f.killed++
			return nil
		},
		every: time.Millisecond,
	}
}

func (f *fakeInner) start() *p5Inner {
	ctl := &p5Ctl{}
	ops := f.ops()
	p := &p5Inner{pid: 4242, ops: ops, ctl: ctl}
	p.wait = p5StartWait(ctl, ops)
	f.p = p
	return p
}

// within fails unless f returns within d.
func within(t *testing.T, d time.Duration, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { f(); close(done) }()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s did not return within %v", what, d)
	}
}

func TestP5ProtoUntil(t *testing.T) {
	modelCheck(t)
	never := make(chan struct{})
	t0 := time.Now()
	if p5Until(never, t0.Add(50*time.Millisecond)) {
		t.Fatal("a task never done was reported done")
	}
	if d := time.Since(t0); d < 40*time.Millisecond || d > time.Second {
		t.Errorf("the query took %v, want about the 50 ms to its deadline", d)
	}
	t1 := time.Now()
	if p5Until(never, t1.Add(-time.Second)) {
		t.Fatal("a task never done was reported done past its deadline")
	}
	if d := time.Since(t1); d > 20*time.Millisecond {
		t.Errorf("past its deadline the query took %v, want no wait", d)
	}
	done := make(chan struct{})
	time.AfterFunc(10*time.Millisecond, func() { close(done) })
	t2 := time.Now()
	if !p5Until(done, t2.Add(5*time.Second)) || time.Since(t2) > time.Second {
		t.Errorf("a task done at 10 ms was not reported done promptly (%v)", time.Since(t2))
	}
}

// The Wait task releases the control lock before Cmd.Wait (W2 before W4):
// while Cmd.Wait does not return, a request takes the lock at once, finds
// the mark and sends nothing; the snapshots never wait. A read that failed
// before the zombie was seen stays kept.
func TestP5ProtoWaitOrder(t *testing.T) {
	modelCheck(t)
	f := newFake(3, 2)
	close(f.killGo)
	p := f.start()
	deadline := time.Now().Add(5 * time.Second)
	for p.wait.snap().Seen.IsZero() {
		if time.Now().After(deadline) {
			t.Fatal("the zombie was never seen")
		}
		time.Sleep(time.Millisecond)
	}
	var s p5WaitSnap
	within(t, time.Second, "a snapshot while Cmd.Wait blocks", func() { s = p.wait.snap() })
	if s.Done || s.Errs == nil || !strings.Contains(s.Errs.Error(), "injected: the read failed") {
		t.Fatalf("snapshot while Cmd.Wait blocks: %+v, want not done, the failed read kept", s)
	}
	// The mark is set, the lock free: R0 needs no request.
	if q := p.request("gate", "k"); q != nil || len(p.reqs) != 0 || len(p.notNeeded) != 1 {
		t.Fatalf("R0 after the observation: task %v, table %d, not needed %v", q, len(p.reqs), p.notNeeded)
	}
	// A request task that runs all the same (an older snapshot) takes the
	// control lock at once and sends nothing.
	q := &p5Req{holder: "gate", step: "old-snapshot", done: make(chan struct{})}
	go q.run(p.ctl, p.ops.kill)
	var r p5ReqSnap
	within(t, time.Second, "a request while Cmd.Wait blocks", func() { r = q.until(time.Now().Add(5 * time.Second)) })
	if r.Result != "not needed" || f.killed != 0 {
		t.Fatalf("a request after the mark: %+v, %d kills; want not needed, none", r, f.killed)
	}
	if got := p.wait.until(time.Now().Add(20 * time.Millisecond)); got.Done {
		t.Fatal("the Wait task is done while Cmd.Wait still blocks")
	}
	close(f.waitGo)
	s = p.wait.until(time.Now().Add(5 * time.Second))
	if !s.Done || s.Exit.Status != "exit 1" || s.Errs == nil {
		t.Fatalf("after Cmd.Wait: %+v, want done, exit 1, the failed read still kept", s)
	}
}

// A request task that stops inside the control lock (its kill does not
// return): the holder's queries still return at their deadlines, R0 starts
// no second task while it is pending, and the Wait task waits at W2 with
// Cmd.Wait not begun. Released, both finish late, and the request was
// registered before it ran.
func TestP5ProtoRequestHung(t *testing.T) {
	modelCheck(t)
	f := newFake(1<<30, 0)
	p := f.start()
	q := p.request("gate", "k")
	if q == nil || len(p.reqs) != 1 {
		t.Fatalf("R0 before any exit: task %v, table %d", q, len(p.reqs))
	}
	var r p5ReqSnap
	t0 := time.Now()
	within(t, time.Second, "the query of a hung request", func() { r = q.until(t0.Add(50 * time.Millisecond)) })
	if r.Result != "pending" {
		t.Fatalf("a hung request: %+v, want pending", r)
	}
	// The inner run becomes a zombie: the Wait task stops at W2.
	f.mu.Lock()
	f.zombieAt = f.reads + 1
	f.mu.Unlock()
	time.Sleep(20 * time.Millisecond)
	var s p5WaitSnap
	within(t, time.Second, "the Wait task's query while W2 waits", func() { s = p.wait.until(time.Now().Add(30 * time.Millisecond)) })
	if s.Done || !s.Seen.IsZero() {
		t.Fatalf("the Wait task while the request holds the lock: %+v, want neither observed nor done", s)
	}
	if again := p.request("remedy", "remedy"); again != q || len(p.reqs) != 1 {
		t.Fatalf("R0 with a request pending: task %p (want %p), table %d", again, q, len(p.reqs))
	}
	if p.pendingReq() != q {
		t.Fatal("the pending request is not the one registered")
	}
	close(f.killGo)
	if r = q.until(time.Now().Add(5 * time.Second)); r.Result != "sent" {
		t.Fatalf("the request released: %+v, want sent", r)
	}
	close(f.waitGo)
	if s = p.wait.until(time.Now().Add(5 * time.Second)); !s.Done {
		t.Fatal("the Wait task did not finish once the lock was free")
	}
	if p.request("final cleanup", "final") != nil {
		t.Fatal("R0 after the observation started a request")
	}
}

// Requests in turn: a new task only once the previous one is done, each
// registered before its kill runs.
func TestP5ProtoRequestSerial(t *testing.T) {
	modelCheck(t)
	f := newFake(1<<30, 0)
	close(f.killGo)
	ops := f.ops()
	inner := ops.kill
	p := f.start()
	p.ops.kill = func() error {
		f.mu.Lock()
		f.killSeen = append(f.killSeen, len(p.reqs))
		f.mu.Unlock()
		return inner()
	}
	q1 := p.request("gate", "k")
	q1.until(time.Now().Add(5 * time.Second))
	q2 := p.request("gate", "remedy")
	if q2 == nil || q2 == q1 || len(p.reqs) != 2 {
		t.Fatalf("a second request after the first was done: %p, %p, table %d", q1, q2, len(p.reqs))
	}
	q2.until(time.Now().Add(5 * time.Second))
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.killSeen) != 2 || f.killSeen[0] != 1 || f.killSeen[1] != 2 {
		t.Fatalf("the table as each kill saw it: %v, want [1 2]", f.killSeen)
	}
	if s := q2.snap(); s.Result != "sent" || s.Holder != "gate" || s.Step != "remedy" {
		t.Fatalf("the second request: %+v", s)
	}
}

// The copy ends at EOF with the whole output; stopped, it returns what it
// read so far with the close's error, and the stop keeps its own. Both read
// ends are closed only through the inner run's stop task (R-B1-3).
func TestP5ProtoCopyStop(t *testing.T) {
	modelCheck(t)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	p := &p5Inner{rd: r.Close}
	p.copy = p5StartCopy(r)
	if _, err := w.WriteString("WP25RESULT {}\n"); err != nil {
		t.Fatal(err)
	}
	if s := p.copy.until(time.Now().Add(30 * time.Millisecond)); s.Done || s.Buf != nil {
		t.Fatalf("the copy with the write end open: %+v, want not done and no buffer", s)
	}
	ss := p.closeRead("the query by w").until(time.Now().Add(5 * time.Second))
	s := p.copy.until(time.Now().Add(5 * time.Second))
	if !ss.Done || ss.Real != nil || ss.Reported != nil || !ss.Released() || ss.Calls != 1 || !s.Done || string(s.Buf) != "WP25RESULT {}\n" || s.Err == nil {
		t.Fatalf("stopped: stop %+v, copy %+v; want one close released, the bytes so far and the close's error", ss, s)
	}
	_ = w.Close()
	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	p2 := &p5Inner{rd: r2.Close}
	p2.copy = p5StartCopy(r2)
	_, _ = w2.WriteString("a\nb\n")
	_ = w2.Close()
	if s := p2.copy.until(time.Now().Add(5 * time.Second)); !s.Done || s.Err != nil || string(s.Buf) != "a\nb\n" {
		t.Fatalf("to EOF: %+v", s)
	}
	if ss := p2.closeRead("EOF").until(time.Now().Add(5 * time.Second)); !ss.Done || !ss.Released() || ss.Calls != 1 || ss.Origin != "EOF" {
		t.Fatalf("the close after EOF: %+v, want one close, released", ss)
	}
}

// R-B1-3: closeRead called twice gives the same task; the close function
// runs once.
func TestP5ProtoCloseOnce(t *testing.T) {
	modelCheck(t)
	var mu sync.Mutex
	n := 0
	p := &p5Inner{rd: func() error {
		mu.Lock()
		n++
		mu.Unlock()
		return nil
	}}
	s1 := p.closeRead("EOF")
	s2 := p.closeRead("the final cleanup")
	if s1 != s2 {
		t.Fatal("a second closeRead started another stop task")
	}
	ss := s1.until(time.Now().Add(5 * time.Second))
	mu.Lock()
	defer mu.Unlock()
	if !ss.Done || n != 1 || ss.Calls != 1 || ss.Origin != "EOF" || !ss.Released() {
		t.Fatalf("the stop task %+v, the close called %d times; want done once, from EOF, released", ss, n)
	}
}

// R-B1-4: the close pending at first, then completing; with an error, the
// duty is handed over and the final cleanup's unknown carries it; with nil
// (the control), the read end is released, the late success counts in no
// T. Either way the first query's problem stays, and the close runs once.
func TestP5ProtoCloseLate(t *testing.T) {
	modelCheck(t)
	for _, c := range []struct {
		name string
		err  error
	}{{"error", errors.New("E: the fake close failed")}, {"nil", nil}} {
		t.Run(c.name, func(t *testing.T) {
			entered, gate := make(chan struct{}), make(chan struct{})
			var once sync.Once
			release := func() { once.Do(func() { close(gate) }) }
			t.Cleanup(release) // a failed exit releases it too
			var mu sync.Mutex
			calls := 0
			p := &p5Inner{rd: func() error {
				mu.Lock()
				calls++
				mu.Unlock()
				close(entered)
				<-gate
				return c.err
			}}
			stop := p.closeRead("EOF")
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("the close never began")
			}
			// 1. Pending at a near deadline: the gate's problem.
			ss := stop.until(time.Now().Add(20 * time.Millisecond))
			prob := p5CloseProblem(ss)
			if ss.Done || !strings.Contains(prob, "did not return") {
				t.Fatalf("pending: %+v, problem %q", ss, prob)
			}
			// 2. Released, its completion awaited.
			release()
			ss = stop.until(time.Now().Add(5 * time.Second))
			if !ss.Done {
				t.Fatal("the close did not complete once released")
			}
			// 3 and 4. The new snapshot: the hand-over and the final cleanup.
			why := p5HandOver(true, true, false, ss)
			out, unk := p5FinalRead(ss)
			if c.err != nil {
				if len(why) != 1 || !strings.Contains(why[0], "read end") || !strings.Contains(unk, "E: the fake close failed") || out != "" {
					t.Fatalf("a close completed with E: hand over %q, outcome %q, unknown %q; want the read end handed over, E in the unknown", why, out, unk)
				}
			} else if len(why) != 0 || out == "" || unk != "" {
				t.Fatalf("a close completed with nil: hand over %q, outcome %q, unknown %q; want none, a late success", why, out, unk)
			}
			if !strings.Contains(prob, "did not return") {
				t.Fatalf("the first query's problem was lost: %q", prob)
			}
			// 5. Once.
			if p.closeRead("the final cleanup") != stop {
				t.Fatal("a second closeRead started another stop task")
			}
			mu.Lock()
			defer mu.Unlock()
			if calls != 1 || ss.Calls != 1 {
				t.Fatalf("the close was called %d times (task %d), want once", calls, ss.Calls)
			}
		})
	}
}

func TestP5ProtoErrs(t *testing.T) {
	modelCheck(t)
	var e p5Errs
	if e.err() != nil {
		t.Fatal("no errors gave one")
	}
	e.add(errors.New("a"))
	e.add(errors.New("a"))
	e.add(errors.New("b"))
	e.add(nil)
	e.add(errors.New("a"))
	if got := e.err().Error(); got != "a (2 times); b; a" {
		t.Fatalf("got %q", got)
	}
}

func TestE2Forms(t *testing.T) {
	modelCheck(t)
	const echild, einval, eintr = syscall.Errno(10), syscall.Errno(22), syscall.Errno(4)
	for _, c := range []struct {
		r    int32
		err  syscall.Errno
		form string
	}{{0, 0, "X1"}, {-1, echild, "X2"}, {-1, 0, "X3"}, {0, eintr, "X4"}, {1, 0, "X5"}, {-2, einval, "X5"}, {7, eintr, "X5"}} {
		if got := e2Form(c.r, c.err); got != c.form {
			t.Errorf("e2Form(%d, %d) = %s, want %s", c.r, c.err, got, c.form)
		}
	}
	for _, c := range []struct {
		form       string
		r          int32
		err, want  syscall.Errno
		seam       bool
		class, why string
	}{
		{"X1", 0, 0, 0, false, e2Consistent, "compared with the prediction"},
		{"X2", -1, eintr, 0, false, e2Native, "the healthy call failed"},
		{"X3", -1, 0, 0, false, e2Tool, "E2 protocol: X3"},
		{"X4", 0, eintr, 0, false, e2Tool, "E2 protocol: X4"},
		{"X5", 3, 0, 0, false, e2Tool, "E2 protocol: X5"},
		{"X1", 0, 0, echild, false, e2Native, "expected to fail"},
		{"X2", -1, echild, echild, false, e2Consistent, "as expected"},
		{"X2", -1, einval, echild, false, e2Native, "not 10"},
		{"X3", -1, 0, einval, false, e2Tool, "E2 protocol: X3"},
		{"X1", 0, 0, 0, true, e2Consistent, "the seam replaces"},
		{"X2", -1, eintr, 0, true, e2Tool, "the real waitid at the seam's site failed"},
		{"X4", 0, echild, 0, true, e2Tool, "E2 protocol: X4"},
	} {
		class, why := e2Class(c.form, c.r, c.err, c.want, c.seam)
		if class != c.class || !strings.Contains(why, c.why) {
			t.Errorf("e2Class(%s, %d, %d, want %d, seam %v) = %s, %q; want %s, %q", c.form, c.r, c.err, c.want, c.seam, class, why, c.class, c.why)
		}
	}
}

func TestP5CallsCheck(t *testing.T) {
	modelCheck(t)
	for _, c := range []struct {
		actual, rule, plan int
		ok                 bool
		want               string
	}{
		{2, 2, 2, true, ""},
		{1, 1, 2, false, ""}, // a failed first read: its class explains it
		{1, 1, 2, true, "no call's class explains it"},
		{3, 2, 2, true, "is not the rule count"},
		{3, 2, 2, false, "is not the rule count"},
		{0, 0, 0, true, ""},
	} {
		got := p5CallsCheck(c.actual, c.rule, c.plan, c.ok)
		if c.want == "" && got != "" || c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("p5CallsCheck(%d, %d, %d, %v) = %q, want %q", c.actual, c.rule, c.plan, c.ok, got, c.want)
		}
	}
}

// The model checks' outcomes, for the WP25P5 summary (§16.11).
var (
	modelMu  sync.Mutex
	modelRes = map[string]bool{}
)

// modelCheck records, at the test's end, whether it passed.
func modelCheck(t *testing.T) {
	t.Cleanup(func() {
		modelMu.Lock()
		modelRes[t.Name()] = !t.Failed()
		modelMu.Unlock()
	})
}
