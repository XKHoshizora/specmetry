package checkrun_test

import (
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/checkrun"
)

// WP2.5 spec §19.10 (G12, D14), row by row: the columns result,
// failure_kind, exit_code, duration_ms, head_end / tree_state_end and
// qualifying of each row, from Classify, Timing and Qualifying together.
// Every run starts clean on commit A; "按公式" rows end the same way, so
// they qualify. The expectations are the table's.
func TestTruthTable(t *testing.T) {
	base := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	at := func(ms int64) time.Time { return base.Add(time.Duration(ms) * time.Millisecond) }
	cleanA := contextOf(headA, checkrun.Clean)
	// One schedule for every row: the item at 0, a start at 10, the exit
	// read at 1010, S3 entered at 2010, the record attempt at 3010.
	normal := checkrun.Events{Launch: checkrun.Started, Item: at(0), Start: at(10), Main: at(1010), Attempt: at(3010)}
	noExit := checkrun.Events{Launch: checkrun.Started, Item: at(0), Start: at(10), S3: at(2010), Attempt: at(3010)}
	killed := checkrun.Events{Launch: checkrun.Started, Item: at(0), Start: at(10), S3: at(2010), Main: at(2500), Attempt: at(3010)}
	none := checkrun.Events{Launch: checkrun.NotStarted, Item: at(0), Attempt: at(3010)}
	anomaly := func(main time.Time) checkrun.Events {
		return checkrun.Events{Launch: checkrun.ResumeAnomaly, Item: at(0), Anomaly: at(15), Main: main, S3: at(2010), Attempt: at(3010)}
	}
	type cols struct {
		result, kind string
		code         *int64
		duration     int64
		interval     bool
		end          checkrun.Context
		qualifying   bool
	}
	for _, tc := range []struct {
		row  string
		o    checkrun.Observation
		e    checkrun.Events
		want cols
	}{
		{"最终 0", checkrun.Observation{Launch: checkrun.Started, Exit: exited(0), Start: cleanA, End: end(headA, checkrun.Clean)}, normal,
			cols{checkrun.Pass, "", code(0), 1000, false, cleanA, true}},
		{"最终非零", checkrun.Observation{Launch: checkrun.Started, Exit: exited(2), Start: cleanA, End: end(headA, checkrun.Clean)}, normal,
			cols{checkrun.Fail, checkrun.NonzeroExit, code(2), 1000, false, cleanA, true}},
		{"Unix 信号", checkrun.Observation{Launch: checkrun.Started, Exit: signaled, Start: cleanA, End: end(headA, checkrun.Clean)}, normal,
			cols{checkrun.Fail, checkrun.NonzeroExit, nil, 1000, false, cleanA, true}},
		{"超时，树已确认", checkrun.Observation{Launch: checkrun.Started, TimedOut: true, Exit: signaled, Start: cleanA, End: end(headA, checkrun.Clean)}, killed,
			cols{checkrun.Error, checkrun.Timeout, nil, 2490, false, cleanA, true}},
		{"进程无法创建", checkrun.Observation{NotStarted: checkrun.SpawnError, Start: cleanA, End: end(headA, checkrun.Clean)}, none,
			cols{checkrun.Error, checkrun.SpawnError, nil, 0, false, cleanA, true}},
		{"输入路径不存在", checkrun.Observation{NotStarted: checkrun.InputMissing, Start: cleanA, End: end(headA, checkrun.Clean)}, none,
			cols{checkrun.Error, checkrun.InputMissing, nil, 0, false, cleanA, true}},
		{"上下文确实变化，启动了进程", checkrun.Observation{Launch: checkrun.Started, Exit: exited(0), Start: cleanA, End: end(headB, checkrun.Clean)}, normal,
			cols{checkrun.Error, checkrun.ContextChanged, code(0), 1000, false, contextOf(headB, checkrun.Clean), false}},
		{"上下文确实变化，没有启动进程", checkrun.Observation{NotStarted: checkrun.InputMissing, Start: cleanA, End: end(headA, checkrun.Dirty)}, none,
			cols{checkrun.Error, checkrun.ContextChanged, nil, 0, false, contextOf(headA, checkrun.Dirty), false}},
		{"D6，启动了进程", checkrun.Observation{Launch: checkrun.Started, Exit: exited(1), Start: cleanA, End: end(headA, checkrun.Unknown), EndUnreadable: true}, normal,
			cols{checkrun.Error, checkrun.ContextChanged, code(1), 1000, false, contextOf(headA, checkrun.Unknown), false}},
		{"D6，没有启动进程", checkrun.Observation{NotStarted: checkrun.SpawnError, Start: cleanA, End: end("", checkrun.Unknown), EndUnreadable: true}, none,
			cols{checkrun.Error, checkrun.ContextChanged, nil, 0, false, notRead, false}},
		{"D19，正常启动，取得了退出状态", checkrun.Observation{Launch: checkrun.Started, TreeNotConfirmed: true, Exit: exited(0), Start: cleanA},
			checkrun.Events{Launch: checkrun.Started, Item: at(0), Start: at(10), Main: at(1010), S3: at(1010), Attempt: at(3010)},
			cols{checkrun.Error, checkrun.ContextChanged, code(0), 1000, false, notRead, false}},
		{"D19，正常启动，没有取得退出状态", checkrun.Observation{Launch: checkrun.Started, TreeNotConfirmed: true, TimedOut: true, Start: cleanA}, noExit,
			cols{checkrun.Error, checkrun.ContextChanged, nil, 2000, true, notRead, false}},
		{"D19，Windows 恢复异常，取得了退出状态（A6）", checkrun.Observation{Launch: checkrun.ResumeAnomaly, TreeNotConfirmed: true, Exit: exited(4), Start: cleanA}, anomaly(at(2500)),
			cols{checkrun.Error, checkrun.ContextChanged, code(4), 1995, true, notRead, false}},
		{"D19，Windows 恢复异常，没有取得退出状态（A6）", checkrun.Observation{Launch: checkrun.ResumeAnomaly, TreeNotConfirmed: true, Start: cleanA}, anomaly(time.Time{}),
			cols{checkrun.Error, checkrun.ContextChanged, nil, 1995, true, notRead, false}},
		{"Windows 恢复的结果异常，终止已确认，取得了退出状态（A12）", checkrun.Observation{Launch: checkrun.ResumeAnomaly, Exit: exited(1), Start: cleanA}, anomaly(at(500)),
			cols{checkrun.Error, checkrun.ContextChanged, code(1), 1995, true, notRead, false}},
		{"Windows 恢复的结果异常，终止已确认，没有取得退出状态（A12）", checkrun.Observation{Launch: checkrun.ResumeAnomaly, Start: cleanA}, anomaly(time.Time{}),
			cols{checkrun.Error, checkrun.ContextChanged, nil, 1995, true, notRead, false}},
	} {
		t.Run(tc.row, func(t *testing.T) {
			out, err := checkrun.Classify(tc.o)
			if err != nil {
				t.Fatalf("Classify: %v", err)
			}
			times, err := checkrun.Timing(tc.e)
			if err != nil {
				t.Fatalf("Timing: %v", err)
			}
			if !out.Record {
				t.Fatal("not recorded")
			}
			r := checkrun.Row{
				Run: checkrun.Run{
					HeadStart: tc.o.Start.Head, HeadEnd: out.End.Head, TreeStateStart: tc.o.Start.TreeState, TreeStateEnd: out.End.TreeState,
					Result: out.Result, FailureKind: out.FailureKind, ExitCode: out.ExitCode, DurationMS: times.DurationMS,
				},
				EvidenceGrade: checkrun.Observed,
			}
			got := cols{out.Result, out.FailureKind, out.ExitCode, times.DurationMS, times.Interval, out.End, checkrun.Qualifying(r)}
			w := tc.want
			if got.result != w.result || got.kind != w.kind || !sameCode(got.code, w.code) || got.duration != w.duration ||
				got.interval != w.interval || got.end != w.end || got.qualifying != w.qualifying {
				t.Errorf("row = %+v (exit %s); want %+v (exit %s)", got, show(got.code), w, show(w.code))
			}
			// §19.10 "not_found": a new write never produces it (D5).
			if got.kind == "not_found" {
				t.Error("the observed runner wrote not_found")
			}
		})
	}
}
