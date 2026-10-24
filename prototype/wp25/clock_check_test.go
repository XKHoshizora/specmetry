package wp25

// Direct checks of the output step's clock (Review 078 N6). runS6On runs on
// a clock the test moves itself, with in-memory pipes whose write ends the
// test holds, so no copier ends before the stop. Closing such a pipe's read
// end ends a pending or a later Read at once. An OS pipe does not do that on
// Windows: Close cancels only a read already issued (CancelIoEx) and then
// waits for the reads in progress, so a read issued just after it blocks
// Close; with phase D taking no real time, that happened in 12 of 180 local
// runs of these checks. The real pipes are the scenarios' own (pipe_test.go).
// Phase D must end on its
// deadline only once its own clock reads at least start + T_drain, and its
// length must be read on that same clock: the boundary, a phase D that
// ends exactly at T_drain, has used T_drain. A wake-up that comes before
// the clock reads the deadline is not the deadline.

import (
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"
)

// fakeClock is a phaseClock that moves only when the test moves it.
type fakeClock struct {
	mu    sync.Mutex
	t     time.Time
	ws    []*fakeWake
	asked chan time.Duration // the d of every wake asked for, in order
}

type fakeWake struct {
	at    time.Time
	ch    chan time.Time
	fired bool // fired or stopped
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Unix(1_000_000, 0), asked: make(chan time.Duration, 64)}
}

func (f *fakeClock) now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

func (f *fakeClock) wake(d time.Duration) (<-chan time.Time, func() bool) {
	f.mu.Lock()
	w := &fakeWake{at: f.t.Add(d), ch: make(chan time.Time, 1)}
	f.ws = append(f.ws, w)
	f.mu.Unlock()
	f.asked <- d
	return w.ch, func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		was := !w.fired
		w.fired = true
		return was
	}
}

// advance moves the clock by d and fires the wakes then due.
func (f *fakeClock) advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(d)
	for _, w := range f.ws {
		if !w.fired && !w.at.After(f.t) {
			w.fired = true
			w.ch <- f.t
		}
	}
}

// fireEarly fires the pending wakes without moving the clock: a wake-up
// that comes before the clock reads its time.
func (f *fakeClock) fireEarly() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, w := range f.ws {
		if !w.fired {
			w.fired = true
			w.ch <- f.t
		}
	}
}

// startS6On runs runS6On (phase D not skipped) on clk with n in-memory
// pipes, in a goroutine; the returned function closes the write ends.
func startS6On(clk *fakeClock, n int) (<-chan s6Result, func()) {
	d := &dest{}
	var ss []*stream
	var ws []*io.PipeWriter
	for i := 0; i < n; i++ {
		r, w := io.Pipe()
		name := fmt.Sprintf("s%d", i)
		p := d.copyPort(name)
		ss = append(ss, &stream{name: name, r: r, c: startCopy(r, p, false), p: p})
		ws = append(ws, w)
	}
	done := make(chan s6Result, 1)
	go func() { done <- runS6On(clk, d, ss, tDrain, tJoin, false) }()
	return done, func() {
		for _, w := range ws {
			_ = w.Close()
		}
	}
}

// asked waits for the next wake asked for, within 5 s of real time.
func asked(t *testing.T, clk *fakeClock) time.Duration {
	t.Helper()
	select {
	case d := <-clk.asked:
		return d
	case <-time.After(5 * time.Second):
		t.Fatal("no wake was asked for within 5 s")
		return 0
	}
}

func s6Done(t *testing.T, res <-chan s6Result) s6Result {
	t.Helper()
	select {
	case r := <-res:
		return r
	case <-time.After(5 * time.Second):
		t.Fatal("runS6On did not return within 5 s")
		return s6Result{}
	}
}

// checkStopped checks a phase D that ended on its deadline, exactly T_drain
// on its clock, with n copiers blocked at the stop.
func checkStopped(t *testing.T, r s6Result, n int) {
	t.Helper()
	if r.drain != tDrain {
		t.Errorf("phase D's recorded length is %v; its clock moved exactly T_drain (%v)", r.drain, tDrain)
	}
	if r.drain < tDrain {
		t.Errorf("phase D used T_drain: %v < %v", r.drain, tDrain)
	}
	if !r.stopped || r.pending != n {
		t.Errorf("stopped %v with %d copiers pending, want true with %d", r.stopped, r.pending, n)
	}
	for name, o := range r.out {
		if o != o2 {
			t.Errorf("%s: %s, want O2 (returned at the stop)", name, o)
		}
	}
}

func TestPhaseClock(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	t.Run("deadline-exact", func(t *testing.T) {
		clk := newFakeClock()
		res, closeW := startS6On(clk, 2)
		defer closeW()
		if d := asked(t, clk); d != tDrain {
			t.Fatalf("phase D asked for a wake after %v, want T_drain %v", d, tDrain)
		}
		clk.advance(tDrain) // the clock reads the deadline exactly
		checkStopped(t, s6Done(t, res), 2)
	})
	t.Run("one-nanosecond-short", func(t *testing.T) {
		clk := newFakeClock()
		res, closeW := startS6On(clk, 2)
		defer closeW()
		asked(t, clk)
		clk.advance(tDrain - time.Nanosecond)
		select {
		case r := <-res:
			t.Fatalf("phase D ended before its clock read T_drain (it read %v)", r.drain)
		case <-time.After(200 * time.Millisecond):
		}
		clk.advance(time.Nanosecond)
		checkStopped(t, s6Done(t, res), 2)
	})
	t.Run("early-wake", func(t *testing.T) {
		// The wake-up comes 450 us before the clock reads the deadline,
		// about the shortfall Windows showed (999.5519 ms on the
		// performance counter).
		const early = 450 * time.Microsecond
		clk := newFakeClock()
		res, closeW := startS6On(clk, 1)
		defer closeW()
		asked(t, clk)
		clk.advance(tDrain - early)
		clk.fireEarly()
		select {
		case r := <-res:
			t.Fatalf("phase D ended on a wake before its clock read the deadline (it read %v)", r.drain)
		case d := <-clk.asked:
			if d == tJoin {
				t.Fatalf("phase D ended at the early wake: phase J's wake (%v) was asked for", d)
			}
			if d != early {
				t.Fatalf("after the early wake, a wake after %v was asked for, want the %v left of phase D", d, early)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("after the early wake, neither a new wake nor the end within 5 s")
		}
		clk.advance(early)
		checkStopped(t, s6Done(t, res), 1)
	})
}
