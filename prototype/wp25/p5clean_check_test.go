//go:build darwin || wp25p5local

package wp25

// Model checks of the cleanup chain and the holder (docs/wp2.5-prototype.md
// §16.17.3): R-B2-1 and R-B2-3, with fakes that record every call, so each
// injected failure is seen hit once, at the step it was meant for. Blocking
// fakes signal when entered and wait for a release gate that the test's
// cleanup also opens; every task a check starts is awaited (§16.17.5).
// Model checks: no native evidence (§16.11).

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// fakeProcs scripts a chain's reads: each key's answers in order (the last
// one repeats), the tag reads' and the kills'; it logs every call.
type fakeProcs struct {
	mu    sync.Mutex
	reads map[string][]string // key -> classes, in order
	tags  map[string][]error  // key -> tag read errors (nil: found), in order
	kills map[string][]error  // key -> kill results, in order
	log   []string
	n     map[string]int
}

func newFakeProcs() *fakeProcs {
	return &fakeProcs{reads: map[string][]string{}, tags: map[string][]error{}, kills: map[string][]error{}, n: map[string]int{}}
}

func (f *fakeProcs) next(kind, key string, n int) int {
	i := f.n[kind+key]
	f.n[kind+key]++
	if i >= n {
		i = n - 1
	}
	return i
}

func (f *fakeProcs) ops(deadline time.Time) p5CleanOps {
	return p5CleanOps{
		read: func(v survivor, at string) (string, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			s := f.reads[v.key()]
			class := s[f.next("read", v.key(), len(s))]
			f.log = append(f.log, "read "+at+" "+class)
			if class == p5Unknown {
				return class, fmt.Errorf("injected: the read of %s failed", v.key())
			}
			return class, nil
		},
		tagged: func(pid int) (bool, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			key := fmt.Sprint(pid)
			s := f.tags[key]
			err := error(nil)
			if len(s) > 0 {
				err = s[f.next("tag", key, len(s))]
			}
			f.log = append(f.log, fmt.Sprintf("tag %v", err))
			return err == nil, err
		},
		kill: func(pid int) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			key := fmt.Sprint(pid)
			s := f.kills[key]
			err := error(nil)
			if len(s) > 0 {
				err = s[f.next("kill", key, len(s))]
			}
			f.log = append(f.log, fmt.Sprintf("kill %v", err))
			return err
		},
		now:   time.Now,
		every: time.Millisecond,
	}
}

// R-B2-1: the chain's six cases.
func TestP5CleanChain(t *testing.T) {
	modelCheck(t)
	v := survivor{101, "1700000000.000101"}
	esrch, eperm := syscall.ESRCH, syscall.EPERM
	for _, c := range []struct {
		name  string
		reads []string
		tags  []error
		kills []error
		short bool // a deadline shortly
		log   []string
		check func(r p5CleanResult) string
	}{
		{name: "1 a round read unknown once, then absent",
			reads: []string{p5Alive, p5Unknown, p5Absent},
			log:   []string{"read entry alive", "read round unknown", "read round absent"},
			check: func(r p5CleanResult) string {
				if r.Errs == nil || !strings.Contains(r.Errs.Error(), "injected") || len(r.Left) != 0 || len(r.Ended) != 1 || r.Found != 1 || r.Kills != 0 {
					return "want the error kept, none left, 1 ended, no kill"
				}
				return ""
			}},
		{name: "2 the entry read unknown, then alive",
			reads: []string{p5Unknown, p5Alive, p5Zombie},
			log:   []string{"read entry unknown", "read round alive", "tag <nil>", "kill <nil>", "read round zombie"},
			check: func(r p5CleanResult) string {
				if r.Errs == nil || r.Kills != 1 || len(r.Left) != 0 || len(r.Ended) != 1 || r.Found != 1 {
					return "want it entered, one kill after the recovered read, then ended, the error kept"
				}
				return ""
			}},
		{name: "3 the tag read fails once, the identity still alive",
			reads: []string{p5Alive, p5Alive, p5Alive, p5Alive, p5Absent},
			tags:  []error{errors.New("injected: the tag read failed"), nil},
			log:   []string{"read entry alive", "read round alive", "tag injected: the tag read failed", "read recheck alive", "read round alive", "tag <nil>", "kill <nil>", "read round absent"},
			check: func(r p5CleanResult) string {
				if r.Errs == nil || !strings.Contains(r.Errs.Error(), "its tag cannot be read") || r.Kills != 1 || len(r.Left) != 0 {
					return "want no action that step, the error kept, the action the next step"
				}
				return ""
			}},
		{name: "4 the tag read fails, the identity read again absent",
			reads: []string{p5Alive, p5Alive, p5Absent, p5Absent},
			tags:  []error{errors.New("injected: the tag read failed")},
			log:   []string{"read entry alive", "read round alive", "tag injected: the tag read failed", "read recheck absent", "read round absent"},
			check: func(r p5CleanResult) string {
				if r.Errs != nil || r.Kills != 0 || !strings.Contains(r.Notes, "ended during the read of its tag") || len(r.Left) != 0 {
					return "want no error (censusTag's rule), no action, the end noted"
				}
				return ""
			}},
		{name: "5a kill gives ESRCH",
			reads: []string{p5Alive, p5Alive, p5Absent},
			kills: []error{esrch},
			log:   []string{"read entry alive", "read round alive", "tag <nil>", "kill " + esrch.Error(), "read round absent"},
			check: func(r p5CleanResult) string {
				if r.Errs != nil || !strings.Contains(r.Notes, "ESRCH") || r.Kills != 0 {
					return "want a note only, no error"
				}
				return ""
			}},
		{name: "5b kill gives EPERM",
			reads: []string{p5Alive, p5Alive, p5Absent},
			kills: []error{eperm},
			log:   []string{"read entry alive", "read round alive", "tag <nil>", "kill " + eperm.Error(), "read round absent"},
			check: func(r p5CleanResult) string {
				if r.Errs == nil || !strings.Contains(r.Errs.Error(), "kill(101, SIGKILL) failed") || r.Kills != 0 {
					return "want the error kept"
				}
				return ""
			}},
		{name: "6 unknown to the deadline",
			reads: []string{p5Alive, p5Unknown},
			short: true,
			check: func(r p5CleanResult) string {
				left := append(append([]survivor(nil), r.Alive...), r.Unknown...)
				_, unk := p5FinalProcs(r)
				if len(r.Unknown) != 1 || len(left) != 1 || r.Errs == nil || !strings.HasPrefix(p5InsureAccount(true, r), "FAILED") || len(unk) < 2 {
					return "want it returned unknown, to hand on, the error kept, a FAILED account and unknowns in the final cleanup"
				}
				return ""
			}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFakeProcs()
			f.reads[v.key()] = c.reads
			f.tags["101"] = c.tags
			f.kills["101"] = c.kills
			deadline := time.Now().Add(5 * time.Second)
			if c.short {
				deadline = time.Now().Add(50 * time.Millisecond)
			}
			var r p5CleanResult
			within(t, 10*time.Second, "the chain", func() { r = p5EndChain([]survivor{v}, deadline, f.ops(deadline)) })
			f.mu.Lock()
			log := append([]string(nil), f.log...)
			f.mu.Unlock()
			if c.log != nil && strings.Join(log, " | ") != strings.Join(c.log, " | ") {
				t.Errorf("the calls:\n got %q\nwant %q", log, c.log)
			}
			if why := c.check(r); why != "" {
				t.Errorf("%s: %s; got %+v (errors %v)", c.name, why, r, r.Errs)
			}
		})
	}
}

// fakeHold is a holder's fakes: a clock the test sets, a probe that may
// block until released, a child that becomes a zombie once killed, a Wait
// that returns at once.
type fakeHold struct {
	mu     sync.Mutex
	clock  time.Time
	clocks int
	zombie bool
	killed int
	block  bool
	enter  chan struct{}
	go_    chan struct{}
	stop   chan struct{}
}

func newFakeHold(t *testing.T, now time.Time) *fakeHold {
	f := &fakeHold{clock: now, enter: make(chan struct{}, 4), go_: make(chan struct{}, 4), stop: make(chan struct{})}
	t.Cleanup(func() { close(f.stop) }) // releases a blocked probe on any exit
	return f
}

func (f *fakeHold) blocking() {
	f.mu.Lock()
	f.block = true
	f.mu.Unlock()
}

func (f *fakeHold) set(c time.Time) {
	f.mu.Lock()
	f.clock = c
	f.mu.Unlock()
}

func (f *fakeHold) ops() p5HolderOps {
	return p5HolderOps{
		now: func() time.Time {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.clocks++
			return f.clock
		},
		probe: func() (bool, error) {
			f.mu.Lock()
			block := f.block
			f.mu.Unlock()
			if block {
				f.enter <- struct{}{}
				select {
				case <-f.go_:
				case <-f.stop:
				}
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			return f.zombie, nil
		},
		observe: func() (bool, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			return f.zombie, nil
		},
		wait:  func() error { return nil },
		state: func() p5Exit { return p5Exit{Status: "signal 9", Code: -1} },
		kill: func() error {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.killed++
			f.zombie = true
			return nil
		},
		every: time.Millisecond,
	}
}

// R-B2-3: the one expiry exit, at decision points the real side uses too.
func TestP5HoldExpiry(t *testing.T) {
	modelCheck(t)
	newHolder := func(f *fakeHold, base time.Time, reg *p5HeldReg) *p5Holder {
		return &p5Holder{side: "model", pid: 4242, cmd: "the fake Cmd", ops: f.ops(), ctl: &p5Ctl{},
			c: base.Add(10 * time.Second), q: base.Add(50 * time.Millisecond), w: base.Add(20 * time.Second), d: base.Add(25 * time.Second), held: reg}
	}
	hasFail := func(h *p5Holder, s string) bool {
		for _, x := range h.fails {
			if strings.Contains(x, s) {
				return true
			}
		}
		return false
	}
	// release waits for the blocked probe (its handshake), then sets the
	// clock and releases it; it runs beside the step, so it reports by its
	// channel, never by t.
	release := func(f *fakeHold, clock time.Time) <-chan error {
		errc := make(chan error, 1)
		go func() {
			select {
			case <-f.enter:
			case <-time.After(5 * time.Second):
				errc <- errors.New("the probe was never entered")
				return
			}
			f.set(clock)
			f.go_ <- struct{}{}
			errc <- nil
		}()
		return errc
	}
	handshake := func(t *testing.T, errc <-chan error) {
		t.Helper()
		if err := <-errc; err != nil {
			t.Fatal(err)
		}
	}
	run := func(t *testing.T, what string, fn func()) {
		t.Helper()
		done := make(chan struct{})
		go func() { fn(); close(done) }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatalf("%s did not return", what)
		}
	}

	// Control one (timeline 己): the chain pending at q_k; it then returns
	// with an error; the cleanup's query gives done, but the clock read at
	// the next decision point is past d_k.
	t.Run("one: done, decided after d_k", func(t *testing.T) {
		base := time.Now()
		f := newFakeHold(t, base)
		var reg p5HeldReg
		h := newHolder(f, base, &reg)
		gate := make(chan struct{})
		var once sync.Once
		t.Cleanup(func() { once.Do(func() { close(gate) }) })
		if h.chainStep(func() (any, error) { <-gate; return nil, errors.New("E: the chain failed") }) {
			t.Fatal("the chain returned by q_k")
		}
		once.Do(func() { close(gate) })
		if done, _, _ := h.chain.until(time.Now().Add(5 * time.Second)); !done {
			t.Fatal("the chain did not return once released")
		}
		f.mu.Lock()
		f.clocks = 0
		f.mu.Unlock()
		// The entry reads d_k - 1 s; the next decision reads d_k + 1 ms.
		reads := []time.Time{h.d.Add(-time.Second), h.d.Add(time.Millisecond)}
		h.ops.now = func() time.Time {
			f.mu.Lock()
			defer f.mu.Unlock()
			i := f.clocks
			f.clocks++
			if i >= len(reads) {
				i = len(reads) - 1
			}
			return reads[i]
		}
		run(t, "the cleanup", h.cleanup)
		items := reg.list()
		switch {
		case len(h.reqs) != 0 || h.wait != nil:
			t.Errorf("a new task after d_k: %d requests, Wait started %v", len(h.reqs), h.wait != nil)
		case f.clocks != 2:
			t.Errorf("the clock was read %d times by the cleanup, want 2 (entry, the request's decision)", f.clocks)
		case !hasFail(h, "q_k") || !hasFail(h, "the cleanup: deadline d_k passed at the decision point (a new request)"):
			t.Errorf("the T: %q", h.fails)
		case len(items) != 1 || items[0].Side != "model" || items[0].Pid != 4242 || items[0].cmd != "the fake Cmd":
			t.Errorf("the registry: %+v, want this side's one item holding its Cmd and pid", items)
		case items[0].Handed.Reap != p5ReapNone || items[0].Handed.Chain != "returned" || !strings.Contains(items[0].Handed.ChainErr, "E: the chain failed"):
			t.Errorf("the knowledge at the hand-over: %+v", items[0].Handed)
		}
	})

	// Control two (timeline 庚): a read that began in time returns past the
	// deadline, on the normal path (w_k), then in the cleanup (d_k).
	t.Run("two: a read returning past the deadline", func(t *testing.T) {
		base := time.Now()
		f := newFakeHold(t, base)
		var reg p5HeldReg
		h := newHolder(f, base, &reg)
		if !h.chainStep(func() (any, error) { return "the chain's result", nil }) {
			t.Fatal("the chain did not return by q_k")
		}
		f.blocking()
		var expired bool
		errc := release(f, h.w.Add(time.Millisecond))
		run(t, "the normal path's steps 3 and 4", func() { expired = h.tail(true) })
		handshake(t, errc)
		if !expired || len(h.reqs) != 0 || h.wait != nil || !hasFail(h, "the normal path: deadline w_k passed at the decision point (a new request)") {
			t.Fatalf("the normal path: expired %v, %d requests, Wait %v, T %q", expired, len(h.reqs), h.wait != nil, h.fails)
		}
		if len(reg.list()) != 0 {
			t.Fatal("the normal path handed on: its duty stays with the holder for the cleanup")
		}
		errc = release(f, h.d.Add(time.Millisecond))
		run(t, "the cleanup", h.cleanup)
		handshake(t, errc)
		items := reg.list()
		switch {
		case len(h.reqs) != 0 || h.wait != nil:
			t.Errorf("a new task after d_k: %d requests, Wait started %v", len(h.reqs), h.wait != nil)
		case !hasFail(h, "deadline w_k passed") || !hasFail(h, "the cleanup: deadline d_k passed at the decision point (a new request)"):
			t.Errorf("the T: %q", h.fails)
		case len(items) != 1 || items[0].Handed.Reap != p5ReapNone || items[0].Handed.Chain != "returned":
			t.Errorf("the registry: %+v", items)
		}
	})

	// Control three: the same steps before every deadline: the request and
	// the Wait as usual, everything confirmed, nothing handed on.
	t.Run("three: decided in time", func(t *testing.T) {
		base := time.Now()
		f := newFakeHold(t, base)
		var reg p5HeldReg
		h := newHolder(f, base, &reg)
		if !h.chainStep(func() (any, error) { return "the chain's result", nil }) {
			t.Fatal("the chain did not return by q_k")
		}
		f.blocking()
		var expired bool
		errc := release(f, base)
		run(t, "the normal path's steps 3 and 4", func() { expired = h.tail(true) })
		handshake(t, errc)
		if expired || len(h.reqs) != 1 || h.wait == nil {
			t.Fatalf("in time: expired %v, %d requests, Wait %v; T %q", expired, len(h.reqs), h.wait != nil, h.fails)
		}
		if q := h.reqs[0].until(time.Now().Add(5 * time.Second)); q.Result != "sent" {
			t.Fatalf("the request: %+v", q)
		}
		if ws := h.wait.until(time.Now().Add(5 * time.Second)); !ws.Done {
			t.Fatal("the Wait task did not complete")
		}
		run(t, "the cleanup", h.cleanup)
		if len(reg.list()) != 0 || h.status != "signal 9" || len(h.expired) != 0 {
			t.Errorf("the cleanup: registry %d, status %q, expired %q; T %q", len(reg.list()), h.status, h.expired, h.fails)
		}
	})
}
