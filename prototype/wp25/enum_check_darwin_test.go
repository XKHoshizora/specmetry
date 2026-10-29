//go:build darwin

package wp25

// Direct checks of the retries that wait on the system (Review 101 B3):
// a kinfo_proc list's ENOMEM, within §19.5's enumeration budget (at most
// 3 retries, each before the caller's deadline), and the kqueue wait's
// EINTR, within what is left of its deadline. The system calls are
// replaced (sysctlRaw, keventCall): this is no native evidence, and no
// scenario. Every call under test runs under a hard limit set before it
// starts. One still running at its limit is a failure; its fake then
// answers a final result at once, so the call returns and no goroutine
// is left behind.

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// bounded runs call under a hard limit. At the limit, it stops the fake
// and waits for the call to return; ok is false then.
func bounded(t *testing.T, limit time.Duration, stop func(), call func()) (took time.Duration, ok bool) {
	t.Helper()
	done := make(chan struct{})
	t0 := time.Now()
	go func() {
		defer close(done)
		call()
	}()
	select {
	case <-done:
		return time.Since(t0), true
	case <-time.After(limit):
	}
	stop()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("the call did not return even after its fake was stopped: a goroutine is left")
	}
	return time.Since(t0), false
}

// enumFake replaces sysctlRaw: its n-th call (from 1) waits, then gives
// step(n); once stopped, every call answers EIO at once.
type enumFake struct {
	mu    sync.Mutex
	calls int
	stop  bool
	step  func(n int) (time.Duration, []byte, error)
}

func (f *enumFake) call(string, ...int) ([]byte, error) {
	f.mu.Lock()
	f.calls++
	n, stop := f.calls, f.stop
	f.mu.Unlock()
	if stop {
		return nil, unix.EIO
	}
	d, b, err := f.step(n)
	time.Sleep(d)
	return b, err
}

func (f *enumFake) halt() {
	f.mu.Lock()
	f.stop = true
	f.mu.Unlock()
}

func (f *enumFake) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func useEnum(t *testing.T, f *enumFake) {
	prev := sysctlRaw
	sysctlRaw = f.call
	t.Cleanup(func() { sysctlRaw = prev })
}

func enomem(d time.Duration) func(int) (time.Duration, []byte, error) {
	return func(int) (time.Duration, []byte, error) { return d, nil, unix.ENOMEM }
}

// entry is one kinfo_proc entry of a live process.
func entry() []byte { return make([]byte, unix.SizeofKinfoProc) }

func TestEnumBudget(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	ms := time.Millisecond
	cases := []struct {
		name    string
		step    func(n int) (time.Duration, []byte, error)
		within  time.Duration // the caller's deadline, from the call
		calls   int
		entries int    // -1: an error, never a list
		errHas  string // in the error
		atLeast time.Duration
	}{
		// Instant ENOMEMs: the 3 retries are used up before the deadline.
		{"enomem-used-up", enomem(0), time.Second, 4, -1, "the 3 retries are used up", 0},
		// Slow ENOMEMs: the deadline comes first, after the second call.
		{"enomem-to-deadline", enomem(60 * ms), 100 * ms, 2, -1, "at the deadline", 100 * ms},
		// The first call ends past the deadline: no retry at all.
		{"enomem-at-boundary", enomem(80 * ms), 50 * ms, 1, -1, "at the deadline", 50 * ms},
		// Two ENOMEMs, then an answer: legal retries before the deadline.
		{"enomem-then-success", func(n int) (time.Duration, []byte, error) {
			if n <= 2 {
				return 0, nil, unix.ENOMEM
			}
			return 0, entry(), nil
		}, time.Second, 3, 1, "", 0},
		// The first call is always made, and its answer kept, even late.
		{"success-past-deadline", func(int) (time.Duration, []byte, error) { return 80 * ms, entry(), nil }, 50 * ms, 1, 1, "", 50 * ms},
		{"other-error", func(int) (time.Duration, []byte, error) { return 0, nil, unix.EIO }, time.Second, 1, -1, unix.EIO.Error(), 0},
		{"empty", func(int) (time.Duration, []byte, error) { return 0, nil, nil }, time.Second, 1, 0, "", 0},
		{"partial-entry", func(int) (time.Duration, []byte, error) { return 0, make([]byte, 10), nil }, time.Second, 1, -1, "not a multiple", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &enumFake{step: c.step}
			useEnum(t, f)
			var ks []unix.KinfoProc
			var err error
			deadline := time.Now().Add(c.within)
			took, ok := bounded(t, c.within+2*time.Second, f.halt, func() { ks, err = kinfoProcs(deadline, "kern.proc.pgrp", 1) })
			if !ok {
				t.Fatalf("still retrying %v past its deadline, after %d calls: the retries are not bounded", took-c.within, f.count())
			}
			if got := f.count(); got != c.calls {
				t.Errorf("%d calls, want %d", got, c.calls)
			}
			budget := strings.Contains(c.errHas, "retries") || strings.Contains(c.errHas, "deadline")
			switch {
			case c.entries < 0 && err == nil:
				t.Errorf("no error: a list of %d", len(ks))
			case c.entries < 0 && !strings.Contains(err.Error(), c.errHas):
				t.Errorf("the error %q does not say %q", err, c.errHas)
			case c.entries < 0 && budget && !errors.Is(err, unix.ENOMEM):
				t.Errorf("the error %q does not carry ENOMEM", err)
			case c.entries >= 0 && err != nil:
				t.Errorf("error %v, want %d entries", err, c.entries)
			case c.entries >= 0 && len(ks) != c.entries:
				t.Errorf("%d entries, want %d", len(ks), c.entries)
			}
			if took < c.atLeast {
				t.Errorf("returned after %v, before %v", took, c.atLeast)
			}
		})
	}
}

// TestEnumChain checks that the readers of P-3 and of the census go
// through the budget: an ENOMEM that never ends gives an error after 3
// retries, never an empty list, from each.
func TestEnumChain(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	reads := []struct {
		name string
		read func(time.Time) error
	}{
		{"groupLive", func(dl time.Time) error {
			m, err := groupLive(1, dl)
			if err == nil {
				return fmt.Errorf("no error: members %v", m)
			}
			return err
		}},
		{"procList", func(dl time.Time) error {
			ks, err := procList(dl)
			if err == nil {
				return fmt.Errorf("no error: %d processes", len(ks))
			}
			return err
		}},
		{"censusTag", func(dl time.Time) error {
			ss, err := censusTag(p3ToolEnv+"=0", "0.000000", dl)
			if err == nil {
				return fmt.Errorf("no error: %d survivors", len(ss))
			}
			if !errors.Is(err, errCensus) {
				return fmt.Errorf("not a census error: %w", err)
			}
			return err
		}},
	}
	for _, rd := range reads {
		t.Run(rd.name, func(t *testing.T) {
			f := &enumFake{step: enomem(0)}
			useEnum(t, f)
			var err error
			took, ok := bounded(t, 3*time.Second, f.halt, func() { err = rd.read(time.Now().Add(time.Second)) })
			if !ok {
				t.Fatalf("still retrying after %v and %d calls: the retries are not bounded", took, f.count())
			}
			if !strings.Contains(err.Error(), "the 3 retries are used up") {
				t.Errorf("the error %q does not say the retries are used up", err)
			}
			if got := f.count(); got != 4 {
				t.Errorf("%d calls, want 4", got)
			}
		})
	}
}

// kevFake replaces keventCall: its n-th call (from 1) waits, then gives
// step(n); once stopped, every call answers no event at once. It keeps
// the timeouts it was given: the first, and whether a later one grew.
type kevFake struct {
	mu    sync.Mutex
	calls int
	stop  bool
	first time.Duration
	prev  time.Duration
	grew  bool
	step  func(n int) (time.Duration, int, error)
}

func (f *kevFake) call(_ int, _, evs []unix.Kevent_t, ts *unix.Timespec) (int, error) {
	f.mu.Lock()
	f.calls++
	n, stop, left := f.calls, f.stop, time.Duration(ts.Nano())
	if n == 1 {
		f.first = left
	} else if left > f.prev {
		f.grew = true
	}
	f.prev = left
	f.mu.Unlock()
	if stop {
		return 0, nil
	}
	d, k, err := f.step(n)
	time.Sleep(d)
	if k > 0 {
		evs[0] = unix.Kevent_t{Ident: 1, Filter: unix.EVFILT_PROC, Fflags: unix.NOTE_EXIT | unix.NOTE_EXITSTATUS, Data: 0x700}
	}
	return k, err
}

func (f *kevFake) halt() {
	f.mu.Lock()
	f.stop = true
	f.mu.Unlock()
}

func TestKqWaitDeadline(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	ms := time.Millisecond
	eintr := func(d time.Duration) func(int) (time.Duration, int, error) {
		return func(int) (time.Duration, int, error) { return d, 0, unix.EINTR }
	}
	cases := []struct {
		name   string
		step   func(n int) (time.Duration, int, error)
		wait   time.Duration
		calls  int // 0: not counted
		event  bool
		errHas string
	}{
		// EINTR that never ends: no retry past the deadline.
		{"eintr-always", eintr(0), 50 * ms, 0, false, "EINTR"},
		// The first call ends past the deadline with EINTR: no retry.
		{"eintr-at-boundary", eintr(60 * ms), 50 * ms, 1, false, "EINTR"},
		// EINTR, then the rest of the wait without an event.
		{"eintr-then-timeout", func(n int) (time.Duration, int, error) {
			if n == 1 {
				return 10 * ms, 0, unix.EINTR
			}
			return 0, 0, nil
		}, 100 * ms, 2, false, ""},
		// EINTR, then the event.
		{"eintr-then-event", func(n int) (time.Duration, int, error) {
			if n == 1 {
				return 10 * ms, 0, unix.EINTR
			}
			return 0, 1, nil
		}, 100 * ms, 2, true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &kevFake{step: c.step}
			prev := keventCall
			keventCall = f.call
			t.Cleanup(func() { keventCall = prev })
			var ok bool
			var err error
			took, returned := bounded(t, c.wait+time.Second, f.halt, func() { _, ok, err = kqWaitRaw(-1, c.wait) })
			f.mu.Lock()
			calls, first, grew := f.calls, f.first, f.grew
			f.mu.Unlock()
			if !returned {
				t.Fatalf("still waiting %v past its deadline, after %d calls: the EINTR retries are not bounded", took-c.wait, calls)
			}
			if c.calls > 0 && calls != c.calls {
				t.Errorf("%d calls, want %d", calls, c.calls)
			}
			if first > c.wait || grew {
				t.Errorf("a retry was given more than what was left of the wait (first %v of %v, grew %v)", first, c.wait, grew)
			}
			switch {
			case c.errHas != "" && (err == nil || !strings.Contains(err.Error(), c.errHas)):
				t.Errorf("error %v, want one that says %q", err, c.errHas)
			case c.errHas == "" && err != nil:
				t.Errorf("error %v", err)
			case ok != c.event:
				t.Errorf("an event %v, want %v", ok, c.event)
			}
		})
	}
}
