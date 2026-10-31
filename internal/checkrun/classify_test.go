package checkrun_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/checkrun"
)

const (
	headA = "8f3c2a1b9d7e6f5a4b3c2d1e0f9a8b7c6d5e4f3a"
	headB = "2b7d4e9f1a3c5e7092b4d6f8a0c2e4f6a8b0d2e4"
)

func contextOf(head, tree string) checkrun.Context {
	return checkrun.Context{Head: head, TreeState: tree}
}

func end(head, tree string) *checkrun.Context { c := contextOf(head, tree); return &c }

func exited(code int64) checkrun.Exit { return checkrun.Exit{Obtained: true, Code: code} }

var signaled = checkrun.Exit{Obtained: true, Signaled: true}

func code(n int64) *int64 { return &n }

// notRead is the end context recorded when it is not read (§10.1: NULL /
// unknown).
var notRead = contextOf("", checkrun.Unknown)

// WP2.5 spec §6.2 in its priority, with the exit code of §10.1 for the row
// hit and the end context of §7.1 and §10.1; the rows of the truth table
// of §19.10 (D14) by name. Each expectation is written from the frozen
// text, not from the implementation.
func TestClassify(t *testing.T) {
	cleanA := contextOf(headA, checkrun.Clean)
	started := func(exit checkrun.Exit, e *checkrun.Context) checkrun.Observation {
		return checkrun.Observation{Launch: checkrun.Started, Exit: exit, Start: cleanA, End: e}
	}
	type want struct {
		c      checkrun.Case
		record bool
		result string
		kind   string
		code   *int64
		end    checkrun.Context
	}
	recorded := func(c checkrun.Case, result, kind string, code *int64, e checkrun.Context) want {
		return want{c, true, result, kind, code, e}
	}
	with := func(o checkrun.Observation, f func(*checkrun.Observation)) checkrun.Observation { f(&o); return o }
	for _, tc := range []struct {
		name string
		o    checkrun.Observation
		want want
	}{
		// §19.10 "最终 0", "最终非零", "Unix 信号"; §6.2 rows 5 and 6.
		{"row 5: final 0 is pass, exit 0", started(exited(0), end(headA, checkrun.Clean)),
			recorded(checkrun.CasePass, checkrun.Pass, "", code(0), cleanA)},
		{"row 6: final nonzero is fail/nonzero_exit with the code", started(exited(1), end(headA, checkrun.Clean)),
			recorded(checkrun.CaseFail, checkrun.Fail, checkrun.NonzeroExit, code(1), cleanA)},
		{"row 6: a Windows code is kept as it is (D17)", started(exited(3221225477), end(headA, checkrun.Clean)),
			recorded(checkrun.CaseFail, checkrun.Fail, checkrun.NonzeroExit, code(3221225477), cleanA)},
		{"row 6: a Unix signal is fail/nonzero_exit with NULL (D12)", started(signaled, end(headA, checkrun.Clean)),
			recorded(checkrun.CaseFail, checkrun.Fail, checkrun.NonzeroExit, nil, cleanA)},
		// §7.2 row 4: a start that is not clean is no change; the run does
		// not qualify, but its result stands.
		{"row 5: dirty at both ends is no context change", with(started(exited(0), end(headA, checkrun.Dirty)), func(o *checkrun.Observation) {
			o.Start = contextOf(headA, checkrun.Dirty)
		}), recorded(checkrun.CasePass, checkrun.Pass, "", code(0), contextOf(headA, checkrun.Dirty))},
		{"row 5: unknown then clean on one HEAD is no context change", with(started(exited(0), end(headA, checkrun.Clean)), func(o *checkrun.Observation) {
			o.Start = contextOf(headA, checkrun.Unknown)
		}), recorded(checkrun.CasePass, checkrun.Pass, "", code(0), cleanA)},
		{"row 5: no commit at both ends is no HEAD change", with(started(exited(0), end("", checkrun.Clean)), func(o *checkrun.Observation) {
			o.Start = contextOf("", checkrun.Clean)
		}), recorded(checkrun.CasePass, checkrun.Pass, "", code(0), contextOf("", checkrun.Clean))},
		// §19.10 "超时，树已确认"; §6.2 row 4: NULL even when an exit was read.
		{"row 4: timeout with the tree confirmed, killed", with(started(signaled, end(headA, checkrun.Clean)), func(o *checkrun.Observation) { o.TimedOut = true }),
			recorded(checkrun.CaseTimeout, checkrun.Error, checkrun.Timeout, nil, cleanA)},
		{"row 4: timeout keeps NULL though the main process exited 0", with(started(exited(0), end(headA, checkrun.Clean)), func(o *checkrun.Observation) { o.TimedOut = true }),
			recorded(checkrun.CaseTimeout, checkrun.Error, checkrun.Timeout, nil, cleanA)},
		// §19.10 "上下文确实变化，启动了进程"; row 1 before rows 4 to 6
		// (Review 091 N5): exit 0 does not imply pass.
		{"row 1: final 0 and clean to dirty is context_changed with 0", started(exited(0), end(headA, checkrun.Dirty)),
			recorded(checkrun.CaseContextChanged, checkrun.Error, checkrun.ContextChanged, code(0), contextOf(headA, checkrun.Dirty))},
		{"row 1: clean to unknown counts as a change", started(exited(0), end(headA, checkrun.Unknown)),
			recorded(checkrun.CaseContextChanged, checkrun.Error, checkrun.ContextChanged, code(0), contextOf(headA, checkrun.Unknown))},
		{"row 1: another HEAD keeps the exit code", started(exited(1), end(headB, checkrun.Clean)),
			recorded(checkrun.CaseContextChanged, checkrun.Error, checkrun.ContextChanged, code(1), contextOf(headB, checkrun.Clean))},
		{"row 1: a first commit during the run is a HEAD change", with(started(exited(0), end(headA, checkrun.Clean)), func(o *checkrun.Observation) {
			o.Start = contextOf("", checkrun.Clean)
		}), recorded(checkrun.CaseContextChanged, checkrun.Error, checkrun.ContextChanged, code(0), cleanA)},
		{"row 1: a signal with a change is NULL", started(signaled, end(headB, checkrun.Clean)),
			recorded(checkrun.CaseContextChanged, checkrun.Error, checkrun.ContextChanged, nil, contextOf(headB, checkrun.Clean))},
		{"row 1 before row 4: a timeout with a change keeps the code read", with(started(exited(1), end(headB, checkrun.Clean)), func(o *checkrun.Observation) { o.TimedOut = true }),
			recorded(checkrun.CaseContextChanged, checkrun.Error, checkrun.ContextChanged, code(1), contextOf(headB, checkrun.Clean))},
		// §19.10 "D6，启动了进程": what was read stays, the rest is NULL /
		// unknown; it does not claim a change.
		{"row 1/D6: an unreadable end keeps what was read", with(started(exited(0), end(headA, checkrun.Unknown)), func(o *checkrun.Observation) { o.EndUnreadable = true }),
			recorded(checkrun.CaseEndUnreadable, checkrun.Error, checkrun.ContextChanged, code(0), contextOf(headA, checkrun.Unknown))},
		{"row 1/D6: nothing read, even from a dirty start", with(started(exited(2), end("", checkrun.Unknown)), func(o *checkrun.Observation) {
			o.Start, o.EndUnreadable = contextOf(headA, checkrun.Dirty), true
		}), recorded(checkrun.CaseEndUnreadable, checkrun.Error, checkrun.ContextChanged, code(2), notRead)},
		// §19.10 "D19，正常启动": the end is not read; the code reliably read
		// is kept whenever it was read.
		{"row 0a: tree not confirmed, no exit read", checkrun.Observation{Launch: checkrun.Started, TreeNotConfirmed: true, TimedOut: true, Start: cleanA},
			recorded(checkrun.CaseTreeNotConfirmed, checkrun.Error, checkrun.ContextChanged, nil, notRead)},
		{"row 0a: tree not confirmed after the main process exited 0", checkrun.Observation{Launch: checkrun.Started, TreeNotConfirmed: true, Exit: exited(0), Start: cleanA},
			recorded(checkrun.CaseTreeNotConfirmed, checkrun.Error, checkrun.ContextChanged, code(0), notRead)},
		{"row 0a: tree not confirmed after a signal", checkrun.Observation{Launch: checkrun.Started, TreeNotConfirmed: true, Exit: signaled, Start: cleanA},
			recorded(checkrun.CaseTreeNotConfirmed, checkrun.Error, checkrun.ContextChanged, nil, notRead)},
		{"row 0a before 0b: tree not confirmed with a cancellation is recorded", checkrun.Observation{Launch: checkrun.Started, TreeNotConfirmed: true, Canceled: true, Start: cleanA},
			recorded(checkrun.CaseTreeNotConfirmed, checkrun.Error, checkrun.ContextChanged, nil, notRead)},
		{"row 0a: in a project outside git", checkrun.Observation{Launch: checkrun.Started, TreeNotConfirmed: true, Exit: exited(3), Start: contextOf("", checkrun.Unknown)},
			recorded(checkrun.CaseTreeNotConfirmed, checkrun.Error, checkrun.ContextChanged, code(3), notRead)},
		// §19.10 "D19，Windows 恢复异常" (A6) and A12 (Q-2 (a)).
		{"row 0a/A6: abnormal resume, tree not confirmed, exit read", checkrun.Observation{Launch: checkrun.ResumeAnomaly, TreeNotConfirmed: true, Exit: exited(7), Start: cleanA},
			recorded(checkrun.CaseTreeNotConfirmed, checkrun.Error, checkrun.ContextChanged, code(7), notRead)},
		{"row 0a/A6: abnormal resume, tree not confirmed, no exit", checkrun.Observation{Launch: checkrun.ResumeAnomaly, TreeNotConfirmed: true, Start: cleanA},
			recorded(checkrun.CaseTreeNotConfirmed, checkrun.Error, checkrun.ContextChanged, nil, notRead)},
		{"A12: abnormal resume, tree confirmed, exit read", checkrun.Observation{Launch: checkrun.ResumeAnomaly, Exit: exited(5), Start: cleanA},
			recorded(checkrun.CaseResumeAnomaly, checkrun.Error, checkrun.ContextChanged, code(5), notRead)},
		{"A12: abnormal resume, tree confirmed, exit 0 is not pass", checkrun.Observation{Launch: checkrun.ResumeAnomaly, Exit: exited(0), Start: cleanA},
			recorded(checkrun.CaseResumeAnomaly, checkrun.Error, checkrun.ContextChanged, code(0), notRead)},
		{"A12: abnormal resume, tree confirmed, no exit", checkrun.Observation{Launch: checkrun.ResumeAnomaly, Start: cleanA},
			recorded(checkrun.CaseResumeAnomaly, checkrun.Error, checkrun.ContextChanged, nil, notRead)},
		// §6.2 row 0b and §7.5: a cancellation before the write is not
		// recorded, an abnormal resume included.
		{"row 0b: canceled after the exit", with(started(exited(0), end(headA, checkrun.Clean)), func(o *checkrun.Observation) { o.Canceled = true }), want{c: checkrun.CaseCanceled}},
		{"row 0b: canceled before the end was read", checkrun.Observation{Launch: checkrun.Started, Canceled: true, Start: cleanA}, want{c: checkrun.CaseCanceled}},
		{"row 0b: canceled abnormal resume", checkrun.Observation{Launch: checkrun.ResumeAnomaly, Canceled: true, Exit: exited(1), Start: cleanA}, want{c: checkrun.CaseCanceled}},
		{"row 0b: canceled with no process", checkrun.Observation{NotStarted: checkrun.InputMissing, Canceled: true, Start: cleanA}, want{c: checkrun.CaseCanceled}},
		// §19.10 "进程无法创建", "输入路径不存在", and the two rows with no
		// process and a change or an unreadable end.
		{"row 2: input_missing has no exit code", checkrun.Observation{NotStarted: checkrun.InputMissing, Start: cleanA, End: end(headA, checkrun.Clean)},
			recorded(checkrun.CaseInputMissing, checkrun.Error, checkrun.InputMissing, nil, cleanA)},
		{"row 3: spawn_error has no exit code", checkrun.Observation{NotStarted: checkrun.SpawnError, Start: cleanA, End: end(headA, checkrun.Clean)},
			recorded(checkrun.CaseSpawnError, checkrun.Error, checkrun.SpawnError, nil, cleanA)},
		{"row 1 before row 2: no process, and another process moved HEAD", checkrun.Observation{NotStarted: checkrun.InputMissing, Start: cleanA, End: end(headB, checkrun.Clean)},
			recorded(checkrun.CaseContextChanged, checkrun.Error, checkrun.ContextChanged, nil, contextOf(headB, checkrun.Clean))},
		{"row 1/D6 before row 3: no process, end unreadable", checkrun.Observation{NotStarted: checkrun.SpawnError, Start: cleanA, End: end("", checkrun.Unknown), EndUnreadable: true},
			recorded(checkrun.CaseEndUnreadable, checkrun.Error, checkrun.ContextChanged, nil, notRead)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := checkrun.Classify(tc.o)
			if err != nil {
				t.Fatalf("Classify: %v", err)
			}
			w := tc.want
			if got.Case != w.c || got.Record != w.record || got.Result != w.result || got.FailureKind != w.kind || !sameCode(got.ExitCode, w.code) || got.End != w.end {
				t.Errorf("Classify = %+v (exit %s); want %+v (exit %s)", got, show(got.ExitCode), w, show(w.code))
			}
		})
	}
}

func sameCode(a, b *int64) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }

func show(c *int64) string {
	if c == nil {
		return "NULL"
	}
	return strconv.FormatInt(*c, 10)
}

// No lawful sequence of the Runner's events gives these observations; each
// is refused, never classified.
func TestClassifyRefusesInconsistentObservations(t *testing.T) {
	cleanA := contextOf(headA, checkrun.Clean)
	for _, tc := range []struct {
		name string
		o    checkrun.Observation
	}{
		{"no process and no reason", checkrun.Observation{Start: cleanA, End: end(headA, checkrun.Clean)}},
		{"no process and another reason", checkrun.Observation{NotStarted: checkrun.Timeout, Start: cleanA, End: end(headA, checkrun.Clean)}},
		{"no process, timed out", checkrun.Observation{NotStarted: checkrun.SpawnError, TimedOut: true, Start: cleanA, End: end(headA, checkrun.Clean)}},
		{"no process, an exit", checkrun.Observation{NotStarted: checkrun.SpawnError, Exit: exited(0), Start: cleanA, End: end(headA, checkrun.Clean)}},
		{"no process, a tree", checkrun.Observation{NotStarted: checkrun.InputMissing, TreeNotConfirmed: true, Start: cleanA}},
		{"a process with a no-process reason", checkrun.Observation{Launch: checkrun.Started, NotStarted: checkrun.SpawnError, Exit: exited(0), Start: cleanA, End: end(headA, checkrun.Clean)}},
		{"S2 woke for nothing", checkrun.Observation{Launch: checkrun.Started, Start: cleanA, End: end(headA, checkrun.Clean)}},
		{"a confirmed tree and no end", checkrun.Observation{Launch: checkrun.Started, Exit: exited(0), Start: cleanA}},
		{"an end read under D19", checkrun.Observation{Launch: checkrun.Started, TreeNotConfirmed: true, Start: cleanA, End: end(headA, checkrun.Clean)}},
		{"an unreadable end under D19", checkrun.Observation{Launch: checkrun.Started, TreeNotConfirmed: true, Start: cleanA, EndUnreadable: true}},
		{"an end read under A12", checkrun.Observation{Launch: checkrun.ResumeAnomaly, Exit: exited(0), Start: cleanA, End: end(headA, checkrun.Clean)}},
		{"an unreadable end with nothing kept", checkrun.Observation{Launch: checkrun.Started, Exit: exited(0), Start: cleanA, EndUnreadable: true}},
		{"a signal not read", checkrun.Observation{Launch: checkrun.Started, Exit: checkrun.Exit{Signaled: true}, Start: cleanA, End: end(headA, checkrun.Clean)}},
		{"a signal with a code", checkrun.Observation{Launch: checkrun.Started, Exit: checkrun.Exit{Obtained: true, Signaled: true, Code: 9}, Start: cleanA, End: end(headA, checkrun.Clean)}},
		{"a code not read", checkrun.Observation{Launch: checkrun.Started, TimedOut: true, Exit: checkrun.Exit{Code: 1}, Start: cleanA, End: end(headA, checkrun.Clean)}},
		{"a start tree state outside the enum", checkrun.Observation{Launch: checkrun.Started, Exit: exited(0), Start: contextOf(headA, "undecidable"), End: end(headA, checkrun.Clean)}},
		{"an end tree state outside the enum", checkrun.Observation{Launch: checkrun.Started, Exit: exited(0), Start: cleanA, End: end(headA, "")}},
		{"an unknown launch", checkrun.Observation{Launch: checkrun.Launch(9), Exit: exited(0), Start: cleanA, End: end(headA, checkrun.Clean)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := checkrun.Classify(tc.o)
			if err == nil || !errors.Is(err, checkrun.ErrObservation) {
				t.Errorf("Classify = %+v, %v; want an inconsistent observation", got, err)
			}
		})
	}
}

// WP2.5 spec §7.2 (data-model §7.4 step 4), its four rows and the notes.
func TestContextChanged(t *testing.T) {
	for _, tc := range []struct {
		start, end checkrun.Context
		want       bool
	}{
		{contextOf(headA, checkrun.Clean), contextOf(headA, checkrun.Clean), false},
		{contextOf(headA, checkrun.Clean), contextOf(headA, checkrun.Dirty), true},
		{contextOf(headA, checkrun.Clean), contextOf(headA, checkrun.Unknown), true},
		{contextOf(headA, checkrun.Clean), contextOf(headB, checkrun.Clean), true},
		{contextOf(headA, checkrun.Dirty), contextOf(headB, checkrun.Dirty), true},
		{contextOf(headA, checkrun.Clean), contextOf("", checkrun.Clean), true},
		{contextOf("", checkrun.Clean), contextOf(headA, checkrun.Clean), true},
		{contextOf(headA, checkrun.Dirty), contextOf(headA, checkrun.Clean), false},
		{contextOf(headA, checkrun.Dirty), contextOf(headA, checkrun.Dirty), false},
		{contextOf(headA, checkrun.Unknown), contextOf(headA, checkrun.Dirty), false},
		{contextOf("", checkrun.Unknown), contextOf("", checkrun.Unknown), false},
		{contextOf("", checkrun.Clean), contextOf("", checkrun.Clean), false},
	} {
		if got := checkrun.ContextChanged(tc.start, tc.end); got != tc.want {
			t.Errorf("ContextChanged(%+v, %+v) = %v; want %v", tc.start, tc.end, got, tc.want)
		}
	}
}
