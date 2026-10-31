package checkrun

import "time"

// Events are the Runner's readings of the events of spec §10.1, on one
// clock (G2). A zero time is an event that did not happen or was not
// obtained. Their order is the order the Runner handled them in.
type Events struct {
	Launch Launch
	// Item is when the item began, before its start context was read;
	// started_at of a run that started no process.
	Item time.Time
	// Start is t_start (event 1), Anomaly t_anom (event 2), Main t_main
	// (event 4: the exit reliably observed), S3 t_S3 (event 5: the
	// decision to terminate).
	Start, Anomaly, Main, S3 time.Time
	// Attempt is event 8, before the write transaction: finished_at.
	Attempt time.Time
}

// Times are a row's time fields (spec §10.1).
type Times struct {
	// StartedAt and FinishedAt are UTC Unix milliseconds, for display
	// only.
	StartedAt, FinishedAt int64
	DurationMS            int64
	// Interval says DurationMS is a defined interval, not the main
	// process's running time nor a bound on it: D19 with no exit read,
	// and every abnormal resume.
	Interval bool
}

// Timing computes the time fields by the start's class alone, whatever the
// result (§10.1): a normal start runs from t_start to t_main, or to t_S3
// when no exit was read; an abnormal resume always runs from t_anom to
// t_S3, an exit read later included; a run with no process has 0.
func Timing(e Events) (Times, error) {
	if e.Attempt.IsZero() {
		return Times{}, invalid("no record attempt time")
	}
	t := Times{FinishedAt: e.Attempt.UnixMilli()}
	var from, to time.Time
	switch e.Launch {
	case NotStarted:
		if e.Item.IsZero() || !e.Start.IsZero() || !e.Anomaly.IsZero() || !e.Main.IsZero() || !e.S3.IsZero() {
			return Times{}, invalid("a run with no process has only its item start")
		}
		t.StartedAt = e.Item.UnixMilli()
		return t, nil
	case Started:
		if e.Start.IsZero() || !e.Anomaly.IsZero() {
			return Times{}, invalid("a normal start has t_start and no t_anom")
		}
		from, to = e.Start, e.Main
		if to.IsZero() {
			to, t.Interval = e.S3, true
		}
	case ResumeAnomaly:
		if e.Anomaly.IsZero() || !e.Start.IsZero() {
			return Times{}, invalid("an abnormal resume has t_anom and no t_start")
		}
		from, to, t.Interval = e.Anomaly, e.S3, true
	default:
		return Times{}, invalid("launch %d", e.Launch)
	}
	if to.IsZero() {
		return Times{}, invalid("neither an exit read nor S3 entered")
	}
	d := to.Sub(from)
	if d < 0 {
		return Times{}, invalid("an interval that ends before it starts")
	}
	t.StartedAt, t.DurationMS = from.UnixMilli(), d.Milliseconds()
	return t, nil
}
