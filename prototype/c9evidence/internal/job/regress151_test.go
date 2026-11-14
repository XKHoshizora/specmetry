package job

import (
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/controls"
)

// Review 151 B-151-1 regressions: the tests' closeout is one request per
// job. A wait that times out is only that wait's failure; the cleanup
// issues Finish only when it was never issued, waits for the same request
// otherwise, and classifies the tasks only after the original call
// returned, from its original Final.
//
// Each case holds the first call at a controlled point that the test
// releases itself; the short first wait is expected to fail because of
// that hold, not because of scheduling. Every other wait is bounded, and
// every hold is released in t.Cleanup too.

// failingWait is the length of a first wait that must fail: the call is
// held until the test releases it after this wait.
const failingWait = 200 * time.Millisecond

// goEndTasks runs endTasks in its own goroutine (as the cleanup would).
func goEndTasks(t *testing.T, j *Job) <-chan chain.OuterSnapshot {
	ch := make(chan chain.OuterSnapshot, 1)
	go func() { ch <- endTasks(t, j) }()
	return ch
}

// awaitEnd waits for goEndTasks; endTasks itself waits at most bound for
// the request and bound for the tasks.
func awaitEnd(t *testing.T, ch <-chan chain.OuterSnapshot) (chain.OuterSnapshot, bool) {
	t.Helper()
	select {
	case s := <-ch:
		return s, true
	case <-time.After(3 * bound):
		t.Errorf("endTasks did not return within %s", 3*bound)
		return chain.OuterSnapshot{}, false
	}
}

// notReturned checks, at that moment only, that endTasks has not delivered
// a result while the original request is still held. It does not show that
// endTasks has started waiting: its goroutine may run only after the
// release (Review 152 N-152-1).
func notReturned(t *testing.T, ch <-chan chain.OuterSnapshot) {
	t.Helper()
	select {
	case <-ch:
		t.Error("endTasks returned while the original Finish had not returned")
	default:
	}
}

// originalFinal checks that the job's request holds the original Final of
// a healthy job (4, consumed) and that no further call was made for it.
func originalFinal(t *testing.T, j *Job) {
	t.Helper()
	f, ok := tryFinish(j, bound)
	if !ok {
		t.Error("the original request did not return")
		return
	}
	if f.Conclusion != chain.Consistent || f.ExitCode != 0 || f.Path != chain.PathConsumed {
		t.Errorf("Final %v exit %d via %v (reasons %q), want the original 4 / 0 / consumed", f.Conclusion, f.ExitCode, f.Path, f.Reasons)
	}
	if n := finishCalls(j); n != 1 {
		t.Errorf("Finish called %d times, want exactly 1", n)
	}
}

// waitStarted waits, bounded, until the closeout published Started (a
// precondition the test sets up, never a decision of the helpers).
func waitStarted(t *testing.T, j *Job) bool {
	t.Helper()
	timeout := time.After(bound)
	for {
		s, ch := j.outer.Watch()
		if s.Closeout.Started {
			return true
		}
		select {
		case <-ch:
		case <-timeout:
			t.Errorf("the closeout did not publish Started within %s", bound)
			return false
		}
	}
}

// B-151-1 (1): the request is registered and its goroutine has not called
// Finish yet when the first wait fails. The cleanup must not issue a second
// call; it waits for the same request.
func TestB151RegisteredNotYetCalled(t *testing.T) {
	j, _, _, g := newJob148(t, controls.LC3(), nil)
	gate, entered := make(chan struct{}), make(chan struct{})
	var first, rel sync.Once
	release := func() { rel.Do(func() { close(gate) }) }
	setFinishHook(j, func() {
		held := false
		first.Do(func() { held = true; close(entered) })
		if held {
			<-gate
		}
	})
	settle(t, j, release, g.freeAll)
	j.RunDomains()
	if _, ok := tryFinish(j, failingWait); ok {
		t.Fatal("Finish returned while its goroutine was held before the call")
	}
	if !waitFor(t, entered, "the request's goroutine reached the point before Finish") {
		return
	}
	if n := finishCalls(j); n != 0 {
		t.Fatalf("Finish called %d times while held before the call", n)
	}
	end := goEndTasks(t, j)
	notReturned(t, end)
	release()
	s, ok := awaitEnd(t, end)
	if n := finishCalls(j); n != 1 {
		t.Errorf("Finish called %d times, want exactly 1", n)
	}
	originalFinal(t, j)
	if ok {
		wantTasks(t, s, writerSuccess, diagComplete)
	}
}

// B-151-1 (2): the test holds the job's lock and the hook that runs right
// before the call has been reached (entered), so the call cannot pass
// Job.Finish's lock and neither finalized nor Started can be set when the
// first wait fails. entered shows the hook was reached and the lock held;
// it does not show the call observed waiting at the lock inside Job.Finish,
// and the exact window with finalized set and Started not yet published is
// not reached (Review 152 N-152-1). The cleanup must wait for that call,
// not make another.
func TestB151CallEnteredBeforeStarted(t *testing.T) {
	j, _, _, g := newJob148(t, controls.LC3(), nil)
	settle(t, j, g.freeAll)
	j.RunDomains() // the work task is done before the lock is taken
	entered := make(chan struct{})
	var once sync.Once
	setFinishHook(j, func() { once.Do(func() { close(entered) }) })
	j.mu.Lock()
	var unlockOnce sync.Once
	unlock := func() { unlockOnce.Do(j.mu.Unlock) }
	t.Cleanup(unlock) // runs before settle's cleanup, which needs the lock
	if _, ok := tryFinish(j, failingWait); ok {
		t.Fatal("Finish returned while the job's lock was held")
	}
	if !waitFor(t, entered, "the hook before the first call was reached") {
		return
	}
	if j.outer.Snapshot().Closeout.Started {
		t.Fatal("Started was published while Job.Finish was held at its lock")
	}
	end := goEndTasks(t, j)
	notReturned(t, end)
	unlock()
	s, ok := awaitEnd(t, end)
	if n := finishCalls(j); n != 1 {
		t.Errorf("Finish called %d times, want exactly 1", n)
	}
	originalFinal(t, j)
	if ok {
		wantTasks(t, s, writerSuccess, diagComplete)
	}
}

// B-151-1 (3): Started is published but Finish has not returned (it waits
// at the snapshot point for the work task, held in its verification, before
// W). The cleanup must not take that moment's "not sealed, no latch, no
// diagnostic task" for the state after the return.
func TestB151StartedButNotReturned(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	v := newVerifier(clk, true)
	j.s.verify = v.verify
	settle(t, j, v.free, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, v.entered, "the work task entered the verification") {
		return
	}
	if _, ok := tryFinish(j, failingWait); ok {
		t.Fatal("Finish returned while the work task was held before W")
	}
	if !waitStarted(t, j) {
		return
	}
	if sealed, _ := j.log.Sealed(); sealed {
		t.Fatal("the log was sealed while the closeout waited for the work task")
	}
	end := goEndTasks(t, j)
	notReturned(t, end)
	v.free()
	if !waitFor(t, ran, "RunDomains returned") {
		return
	}
	s, ok := awaitEnd(t, end)
	if n := finishCalls(j); n != 1 {
		t.Errorf("Finish called %d times, want exactly 1", n)
	}
	originalFinal(t, j)
	if ok {
		wantTasks(t, s, writerSuccess, diagComplete)
	}
}

// B-151-1 (4): the original call returns late, after a failed wait and a
// release (the failure branch of the two hand-written asynchronous Finish
// calls). A second wait gets the original Final of the same request, no
// second call is made, and only then are the tasks ended.
func TestB151LateReturnKeepsTheOriginalFinal(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	v := newVerifier(clk, true)
	j.s.verify = v.verify
	settle(t, j, v.free, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, v.entered, "the work task entered the verification") {
		return
	}
	if _, ok := tryFinish(j, failingWait); ok {
		t.Fatal("Finish returned while the work task was held before W")
	}
	v.free()
	if !waitFor(t, ran, "RunDomains returned") {
		return
	}
	originalFinal(t, j)
	wantTasks(t, endTasks(t, j), writerSuccess, diagComplete)
	if n := finishCalls(j); n != 1 {
		t.Errorf("Finish called %d times after endTasks, want exactly 1", n)
	}
}

// B-151-1 (5): a failure exit that never issued Finish: the cleanup issues
// it exactly once, and a second cleanup issues nothing more.
func TestB151NeverIssuedCleanupIssuesOnce(t *testing.T) {
	j, _, _, g := newJob148(t, controls.LC3(), nil)
	settle(t, j, g.freeAll)
	j.RunDomains()
	if n := finishCalls(j); n != 0 {
		t.Fatalf("Finish called %d times before the cleanup", n)
	}
	wantTasks(t, endTasks(t, j), writerSuccess, diagComplete)
	wantTasks(t, endTasks(t, j), writerSuccess, diagComplete)
	originalFinal(t, j)
}

// B-151-1 (6) controls: a normal closeout, and an expiry exit before the
// seal whose writer is still waiting until the cleanup wakes it. Since
// Review 152 B-152-1 the expiry control first establishes, from the
// writer's published state, that the writer has written every record and
// is waiting (see regress152_test.go).
func TestB151ControlsNormalAndExpiryBeforeTheSeal(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		j, _, _, g := newJob148(t, controls.LC3(), nil)
		settle(t, j, g.freeAll)
		j.RunDomains()
		finishWant(t, j, chain.Consistent, chain.PathConsumed)
		wantTasks(t, endTasks(t, j), writerSuccess, diagComplete)
		if n := finishCalls(j); n != 1 {
			t.Errorf("Finish called %d times, want exactly 1", n)
		}
	})
	t.Run("expiry before the seal", func(t *testing.T) {
		// Review 152 B-152-1: RunDomains returning does not mean the writer
		// has written the records. Before D, wait (bounded) until the
		// writer's own published state shows every existing item written and
		// nothing left to write (regress152_test.go); only then is a writer
		// without a terminal state after Finish a consequence of the
		// precondition, which expireWaitingWriter checks before and after
		// endTasks's wake.
		fs := newCountFS(false)
		j, clk, _, g := newJob148FS(t, controls.LC3(), nil, fs)
		settle(t, j, g.freeAll)
		j.RunDomains()
		idle, why, ok := waitWriterIdle(j, fs, bound)
		if !ok {
			t.Fatalf("the writer was not idle within %s before D: %s", bound, why)
		}
		expireWaitingWriter(t, j, clk, fs, idle)
	})
}
