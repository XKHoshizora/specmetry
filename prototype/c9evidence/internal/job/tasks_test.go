package job

import (
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
)

// The background tasks of a test job (Review 150 N-150-1). Begin starts the
// record writer; the closeout registers the diagnostic task. Neither ends
// by itself while it waits: the writer waits for a log change until the log
// is sealed, and the diagnostic task waits for the writer's terminal state.
// So a test that never runs Finish, or whose Finish takes an expiry exit
// before the seal, leaves the writer waiting. These helpers end both tasks,
// bounded, after the test's own assertions, and check their actual terminal
// states. They change nothing in the tool.

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
// both terminal states, bounded; it returns the registry's copy.
//
//   - The closeout never ran (only on a failure exit, since every test calls
//     Finish once): it runs Finish once here, only to seal the log and end
//     the tasks; it does not judge the result.
//   - The log is sealed: the writer finishes writing and closing, and the
//     diagnostic task finishes after it; this only waits.
//   - The closeout took an expiry exit before the seal (the latch is set):
//     the writer waits for a log change and does not wake for D by itself.
//     One wake item is appended to the log after the decision; at its next
//     decision point the writer takes the expiry exit without writing it.
//
// It never calls Finish twice and never releases a block of the test.
func endTasks(t *testing.T, j *Job) chain.OuterSnapshot {
	t.Helper()
	if !j.outer.Snapshot().Closeout.Started {
		finish(t, j)
	}
	o := j.outer.Snapshot()
	if sealed, _ := j.log.Sealed(); !sealed && o.Writer != nil && o.Writer.Terminal == nil {
		if !o.Latch {
			t.Errorf("the closeout left the log unsealed without the expiry latch: %+v", o.Closeout)
			return o
		}
		j.log.Append(FileCloseout, []byte(`{"test":"lifecycle wake after the decision (N-150-1)"}`))
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
