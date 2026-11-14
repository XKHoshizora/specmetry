package job

import (
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
)

// The background tasks of a test job (Review 150 N-150-1). Begin starts the
// record writer; the closeout registers the diagnostic task. Neither ends
// by itself while it waits: while the log is not sealed, the writer waits
// for a log change, and the diagnostic task waits for the writer's terminal
// state. So a writer whose log is not sealed and that no later record has
// woken is still waiting for a record change; a test that never ran Finish
// or whose Finish took an expiry exit before the seal does not by itself
// mean that (a record appended after D may already have woken it). These
// helpers end both tasks, bounded, after the test's own assertions, and
// check their published terminal states; a published terminal state is
// the task's last state, not an observed exit of its goroutine. They change
// nothing in the tool.

// The terminal states the tests expect, as "state/qualification".
const (
	writerSuccess = "success/timely"
	writerExpired = "expired-incomplete/late"
	diagComplete  = "complete-with-terminator/timely"
	diagNone      = "none" // the closeout never registered a diagnostic task
)

// taskStates returns the writer's and the diagnostic task's terminal states
// as "state/qualification"; "" is a task without a terminal state, and
// "none" a diagnostic task that was never registered.
func taskStates(o chain.OuterSnapshot) (writer, diag string) {
	if o.Writer != nil && o.Writer.Terminal != nil {
		writer = o.Writer.Terminal.State.String() + "/" + o.Writer.Terminal.Qual.String()
	}
	diag = diagNone
	if o.Diag != nil {
		diag = ""
		if o.Diag.Terminal != nil {
			diag = o.Diag.Terminal.State.String() + "/" + o.Diag.Terminal.Qual.String()
		}
	}
	return writer, diag
}

func tasksEnded(o chain.OuterSnapshot) bool {
	w, d := taskStates(o)
	return w != "" && d != ""
}

// endTasks ends the job's record writer and diagnostic task and waits for
// both terminal states, bounded; it returns the registry's copy. It works
// from the job's one closeout request (finish_test.go; Review 151
// B-151-1), never from Closeout.Started:
//
//   - No request (Finish was never issued; only on a failure exit, since
//     every test issues it once): it issues the request once here, only to
//     seal the log and end the tasks; it does not judge the result.
//   - A request: it waits, bounded, for that same request. When the wait
//     fails it reports it and returns; the request stays registered and
//     pending, and nothing is classified or claimed to have ended.
//   - Only after the original call returned, by the original Final's exit:
//     consumed or expired-final sealed the log, so it only waits for the
//     writer and any registered diagnostic task; expired-at-m.Run or
//     expired-at-snapshot came before the seal with the latch set, so a
//     writer still waiting for a log change gets one wake item, appended
//     after the decision, and takes the expiry exit at its next decision
//     point without writing it. Any other exit is a failure.
//
// It never issues a second Finish and never releases a block of the test.
func endTasks(t *testing.T, j *Job) chain.OuterSnapshot {
	t.Helper()
	r := requestOf(j)
	if r == nil {
		r = requestFinish(j)
	}
	f, ok := r.wait(bound)
	if !ok {
		t.Errorf("the job's closeout request did not return within %s; it stays registered and pending, and the tasks are not classified", bound)
		return j.outer.Snapshot()
	}
	o := j.outer.Snapshot()
	sealed, _ := j.log.Sealed()
	switch f.Path {
	case chain.PathConsumed, chain.PathExpiredFinal:
		if !sealed {
			t.Errorf("the closeout returned via %v without sealing the log", f.Path)
			return o
		}
	case chain.PathExpiredAtMRun, chain.PathExpiredAtSnapshot:
		if sealed || !o.Latch {
			t.Errorf("the closeout returned via %v with sealed %v and latch %v", f.Path, sealed, o.Latch)
			return o
		}
		if o.Writer != nil && o.Writer.Terminal == nil {
			j.log.Append(FileCloseout, []byte(`{"test":"lifecycle wake after the decision (N-150-1)"}`))
		}
	default:
		t.Errorf("the closeout returned via %v (reasons %q): it did not run", f.Path, f.Reasons)
		return o
	}
	timeout := time.After(bound)
	for {
		s, ch := j.outer.Watch()
		if tasksEnded(s) {
			return s
		}
		select {
		case <-ch:
		case <-timeout:
			w, d := taskStates(s)
			t.Errorf("the job's tasks did not end within %s: writer %q diagnostic %q", bound, w, d)
			return s
		}
	}
}

// wantTasks checks the terminal states endTasks returned.
func wantTasks(t *testing.T, o chain.OuterSnapshot, writer, diag string) {
	t.Helper()
	if w, d := taskStates(o); w != writer || d != diag {
		t.Errorf("writer %q diagnostic %q, want %q %q", w, d, writer, diag)
	}
}

// finishWant is a test's final judgment: it runs the closeout once and
// checks the conclusion, its exit status and the exit path.
func finishWant(t *testing.T, j *Job, c chain.Conclusion, path chain.ExitPath) {
	t.Helper()
	f, ok := finish(t, j)
	if !ok {
		return
	}
	if f.Conclusion != c || f.ExitCode != chain.ExitCode(c) || f.Path != path {
		t.Errorf("final %v exit %d via %v (reasons %q), want %v exit %d via %v", f.Conclusion, f.ExitCode, f.Path, f.Reasons, c, chain.ExitCode(c), path)
	}
}
