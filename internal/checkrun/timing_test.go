package checkrun_test

import (
	"errors"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/checkrun"
)

// WP2.5 spec §10.1 (D19, the event dictionary and the field table) and the
// duration_ms column of §19.10: the start's class alone decides the time
// fields. A normal start runs from t_start to t_main, to t_S3 (a defined
// interval) when no exit was read; an abnormal resume always runs from
// t_anom to t_S3 (A6, A12), an exit read later included; a run with no
// process has 0, and started_at is the item's start. finished_at is the
// record attempt.
func TestTiming(t *testing.T) {
	base := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	at := func(ms int64) time.Time { return base.Add(time.Duration(ms) * time.Millisecond) }
	ms := base.UnixMilli()
	for _, tc := range []struct {
		name string
		e    checkrun.Events
		want checkrun.Times
	}{
		{"no process: 0 from the item start", checkrun.Events{Launch: checkrun.NotStarted, Item: at(0), Attempt: at(40)},
			checkrun.Times{StartedAt: ms, FinishedAt: ms + 40}},
		{"normal start, exit read: t_main - t_start", checkrun.Events{Launch: checkrun.Started, Item: at(0), Start: at(10), Main: at(8431), Attempt: at(8500)},
			checkrun.Times{StartedAt: ms + 10, FinishedAt: ms + 8500, DurationMS: 8421}},
		{"normal start, exit read after S3 (timeout, E9): still t_main - t_start", checkrun.Events{Launch: checkrun.Started, Start: at(0), S3: at(120000), Main: at(120250), Attempt: at(121000)},
			checkrun.Times{StartedAt: ms, FinishedAt: ms + 121000, DurationMS: 120250}},
		{"normal start, D19 with an exit read: t_main - t_start", checkrun.Events{Launch: checkrun.Started, Start: at(0), Main: at(500), S3: at(500), Attempt: at(9000)},
			checkrun.Times{StartedAt: ms, FinishedAt: ms + 9000, DurationMS: 500}},
		{"normal start, D19 with no exit read: t_S3 - t_start, an interval", checkrun.Events{Launch: checkrun.Started, Start: at(0), S3: at(300000), Attempt: at(305000)},
			checkrun.Times{StartedAt: ms, FinishedAt: ms + 305000, DurationMS: 300000, Interval: true}},
		{"abnormal resume, exit read (A6, A12): t_S3 - t_anom", checkrun.Events{Launch: checkrun.ResumeAnomaly, Anomaly: at(20), Main: at(25), S3: at(30), Attempt: at(90)},
			checkrun.Times{StartedAt: ms + 20, FinishedAt: ms + 90, DurationMS: 10, Interval: true}},
		{"abnormal resume, exit read after S3: still t_S3 - t_anom", checkrun.Events{Launch: checkrun.ResumeAnomaly, Anomaly: at(20), S3: at(30), Main: at(70), Attempt: at(90)},
			checkrun.Times{StartedAt: ms + 20, FinishedAt: ms + 90, DurationMS: 10, Interval: true}},
		{"abnormal resume, no exit: t_S3 - t_anom", checkrun.Events{Launch: checkrun.ResumeAnomaly, Anomaly: at(20), S3: at(20), Attempt: at(90)},
			checkrun.Times{StartedAt: ms + 20, FinishedAt: ms + 90, Interval: true}},
		{"equal readings are 0", checkrun.Events{Launch: checkrun.Started, Start: at(5), Main: at(5), Attempt: at(5)},
			checkrun.Times{StartedAt: ms + 5, FinishedAt: ms + 5}},
		{"milliseconds are truncated", checkrun.Events{Launch: checkrun.Started, Start: at(0), Main: base.Add(1999 * time.Microsecond), Attempt: at(3)},
			checkrun.Times{StartedAt: ms, FinishedAt: ms + 3, DurationMS: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := checkrun.Timing(tc.e)
			if err != nil {
				t.Fatalf("Timing: %v", err)
			}
			if got != tc.want {
				t.Errorf("Timing = %+v; want %+v", got, tc.want)
			}
		})
	}
}

// Readings no lawful run has are refused, never turned into a duration.
func TestTimingRefusesInconsistentEvents(t *testing.T) {
	base := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	at := func(ms int64) time.Time { return base.Add(time.Duration(ms) * time.Millisecond) }
	for _, tc := range []struct {
		name string
		e    checkrun.Events
	}{
		{"no attempt time", checkrun.Events{Launch: checkrun.Started, Start: at(0), Main: at(1)}},
		{"no process, no item start", checkrun.Events{Launch: checkrun.NotStarted, Attempt: at(1)}},
		{"no process with t_start", checkrun.Events{Launch: checkrun.NotStarted, Item: at(0), Start: at(0), Attempt: at(1)}},
		{"no process with t_S3", checkrun.Events{Launch: checkrun.NotStarted, Item: at(0), S3: at(0), Attempt: at(1)}},
		{"normal start without t_start", checkrun.Events{Launch: checkrun.Started, Main: at(1), Attempt: at(2)}},
		{"normal start with t_anom", checkrun.Events{Launch: checkrun.Started, Start: at(0), Anomaly: at(0), Main: at(1), Attempt: at(2)}},
		{"normal start with neither t_main nor t_S3", checkrun.Events{Launch: checkrun.Started, Start: at(0), Attempt: at(2)}},
		{"abnormal resume without t_anom", checkrun.Events{Launch: checkrun.ResumeAnomaly, S3: at(1), Attempt: at(2)}},
		{"abnormal resume with t_start", checkrun.Events{Launch: checkrun.ResumeAnomaly, Start: at(0), Anomaly: at(0), S3: at(1), Attempt: at(2)}},
		{"abnormal resume without t_S3", checkrun.Events{Launch: checkrun.ResumeAnomaly, Anomaly: at(0), Main: at(1), Attempt: at(2)}},
		{"t_main before t_start", checkrun.Events{Launch: checkrun.Started, Start: at(10), Main: at(9), Attempt: at(11)}},
		{"t_S3 before t_anom", checkrun.Events{Launch: checkrun.ResumeAnomaly, Anomaly: at(10), S3: at(9), Attempt: at(11)}},
		{"an unknown launch", checkrun.Events{Launch: checkrun.Launch(-1), Start: at(0), Main: at(1), Attempt: at(2)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := checkrun.Timing(tc.e)
			if err == nil || !errors.Is(err, checkrun.ErrObservation) {
				t.Errorf("Timing = %+v, %v; want an inconsistent observation", got, err)
			}
		})
	}
}
