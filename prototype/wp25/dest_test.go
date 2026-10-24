package wp25

// The controlled destination and the pipe path's copy and teardown of
// spec §8.5 S6 and §9.1, as a prototype: one lock serializes every write,
// writes never block, copy ports can be sealed, and the owner's port is
// separate. The copier and the S6 driver are what the pipe scenarios run.

import (
	"errors"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	errSealed   = errors.New("port sealed")
	errCapacity = errors.New("destination full")
)

type span struct {
	port   string
	off, n int
}

// dest is the controlled in-memory destination.
type dest struct {
	mu       sync.Mutex
	buf      []byte
	log      []span
	copyCap  int // bytes the copy ports may add in all; 0: no limit
	ownerCap int // bytes the owner may add; 0: no limit (fault injection)
	copied   int
	owned    int
}

type port struct {
	d        *dest
	name     string
	owner    bool
	sealed   bool // guarded by d.mu
	rejected int  // writes refused after the seal, guarded by d.mu
}

func (d *dest) copyPort(name string) *port { return &port{d: d, name: name} }
func (d *dest) ownerPort() *port           { return &port{d: d, name: "owner", owner: true} }

// Write appends under the lock and never blocks. A sealed port refuses
// everything; a full destination takes what fits and reports errCapacity.
func (p *port) Write(b []byte) (int, error) {
	d := p.d
	d.mu.Lock()
	defer d.mu.Unlock()
	if p.sealed {
		p.rejected++
		return 0, errSealed
	}
	n := len(b)
	switch {
	case p.owner && d.ownerCap > 0 && d.owned+n > d.ownerCap:
		n = d.ownerCap - d.owned
	case !p.owner && d.copyCap > 0 && d.copied+n > d.copyCap:
		n = d.copyCap - d.copied
	}
	if n < 0 {
		n = 0
	}
	if n > 0 {
		d.log = append(d.log, span{p.name, len(d.buf), n})
		d.buf = append(d.buf, b[:n]...)
		if p.owner {
			d.owned += n
		} else {
			d.copied += n
		}
	}
	if n < len(b) {
		return n, errCapacity
	}
	return n, nil
}

// seal closes the ports to further writes; it is one setting under the
// lock and does no I/O.
func (d *dest) seal(ps ...*port) {
	d.mu.Lock()
	for _, p := range ps {
		p.sealed = true
	}
	d.mu.Unlock()
}

func (d *dest) rejectedBy(p *port) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return p.rejected
}

func (d *dest) snapshot() ([]byte, []span) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]byte(nil), d.buf...), append([]span(nil), d.log...)
}

// streamOf gives the bytes one port added, in order.
func streamOf(buf []byte, log []span, name string) []byte {
	var out []byte
	for _, s := range log {
		if s.port == name {
			out = append(out, buf[s.off:s.off+s.n]...)
		}
	}
	return out
}

// lastIsWhole reports whether the owner's last write is the document,
// whole, in one span, and nothing came after it.
func lastIsWhole(buf []byte, log []span, doc []byte) bool {
	if len(log) == 0 {
		return false
	}
	l := log[len(log)-1]
	return l.port == "owner" && l.n == len(doc) && l.off+l.n == len(buf) && string(buf[l.off:]) == string(doc)
}

type copyEnd string

const (
	endEOF   copyEnd = "eof"   // read EOF with every byte written: E-drain
	endRead  copyEnd = "read"  // a read error, the stopped read included
	endWrite copyEnd = "write" // a write error: capacity, sealed, short
)

type copyResult struct {
	end       copyEnd
	written   int64
	discarded int64
	err       error
}

// copier copies one stream to its port. With discard set, a write error
// does not stop the reading: the rest is read and dropped, so the check is
// not blocked on a full pipe (a candidate; see the capacity scenarios).
type copier struct {
	done chan copyResult
	res  *copyResult
}

func startCopy(src io.Reader, dst io.Writer, discard bool) *copier {
	c := &copier{done: make(chan copyResult, 1)}
	go func() {
		buf := make([]byte, 32<<10)
		var r copyResult
		for {
			n, rerr := src.Read(buf)
			if n > 0 {
				if r.end == endWrite {
					r.discarded += int64(n)
				} else {
					w, werr := dst.Write(buf[:n])
					r.written += int64(w)
					if werr == nil && w < n {
						werr = io.ErrShortWrite
					}
					if werr != nil {
						r.end, r.err = endWrite, werr
						r.discarded += int64(n - w)
						if !discard {
							c.done <- r
							return
						}
					}
				}
			}
			if rerr != nil {
				if r.end != endWrite {
					if errors.Is(rerr, io.EOF) {
						r.end = endEOF
					} else {
						r.end, r.err = endRead, rerr
					}
				}
				c.done <- r
				return
			}
		}
	}()
	return c
}

// phaseClock bounds the output step's phases and reads their lengths
// (Review 078 N6). wake gives a channel that fires after about d, and a
// function that stops it.
type phaseClock interface {
	now() time.Time
	wake(d time.Duration) (<-chan time.Time, func() bool)
}

// goClock is Go's clock: time.Now's monotonic reading and the runtime's
// timers, the clock T_drain and T_join are defined on. On Windows it
// advances with the clock interrupt; a finer reading of the same phase
// (the stopwatch's performance counter) can be shorter by up to one of its
// ticks, and the tick can change while a run goes on. So a phase is bounded
// and judged on this clock, never on the stopwatch.
type goClock struct{}

func (goClock) now() time.Time { return time.Now() }

func (goClock) wake(d time.Duration) (<-chan time.Time, func() bool) {
	t := time.NewTimer(d)
	return t.C, t.Stop
}

// wait waits until the copier returns or the deadline passes.
func (c *copier) wait(deadline time.Time) bool { return c.waitOn(goClock{}, deadline) }

// waitOn waits until the copier returns, or until clk reads at least the
// deadline (Review 078 N6): a wake-up is only a cue to read the clock
// again, so a phase never ends before its own clock says so.
func (c *copier) waitOn(clk phaseClock, deadline time.Time) bool {
	for {
		if c.res != nil {
			return true
		}
		select {
		case r := <-c.done:
			c.res = &r
			return true
		default:
		}
		left := deadline.Sub(clk.now())
		if left <= 0 {
			return false
		}
		ch, stop := clk.wake(left)
		select {
		case r := <-c.done:
			stop()
			c.res = &r
			return true
		case <-ch:
		}
	}
}

type outcome string

const (
	o1 outcome = "O1" // ended at EOF, every byte written
	o2 outcome = "O2" // returned, but incomplete
	o3 outcome = "O3" // not confirmed to have returned (its port is sealed)
)

// stream is one output stream of one run on the pipe path.
type stream struct {
	name string
	r    io.Closer
	c    *copier
	p    *port
}

type s6Result struct {
	out     map[string]outcome
	stopped bool          // the read was stopped
	pending int           // copiers not returned when the read was stopped
	drain   time.Duration // time spent in phase D, on the clock that bounds it
	join    time.Duration // time spent in phase J, on the same clock
	// The stopwatch's readings of the same phases, kept to compare with
	// the earlier runs' readings; nothing is judged on them (Review 078 N6).
	drainFine, joinFine time.Duration
	// The stop of each stopped stream's read (Review 081 B1): "done" once
	// closing its read end has returned, "pending" if it had not when the
	// run returned. A stream that ended in phase D is not stopped.
	stop map[string]string
	// stopErr is the error of a stop that is done, if any.
	stopErr map[string]error
	// What the run still owned when it returned: copiers not confirmed to
	// have returned, stops not done. Its caller keeps them until they end.
	left []leftover
}

// stopTask is the stop of one stream's read (Review 081 B1): closing its
// read end, in a task of its own started with phase J, so that a Close
// that waits for a read holding the file keeps neither phase J nor the
// seal waiting. The run that starts it owns it; if it is not done when the
// run returns, the run hands it on in s6Result.left. It ends when the
// close returns, and keeps the close's error.
type stopTask struct {
	done chan struct{}
	err  error // valid once done is closed
}

func startStop(s *stream) *stopTask {
	t := &stopTask{done: make(chan struct{})}
	go func() {
		defer close(t.done)
		t.err = closeRead(s.r)
	}()
	return t
}

// until waits until the stop is done or the deadline passes.
func (t *stopTask) until(deadline time.Time) bool {
	select {
	case <-t.done:
		return true
	default:
	}
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	select {
	case <-t.done:
		return true
	case <-timer.C:
		return false
	}
}

// awaitOn waits until ch is closed, or until clk reads at least the
// deadline.
func awaitOn(clk phaseClock, ch <-chan struct{}, deadline time.Time) bool {
	for {
		select {
		case <-ch:
			return true
		default:
		}
		left := deadline.Sub(clk.now())
		if left <= 0 {
			return false
		}
		w, stop := clk.wake(left)
		select {
		case <-ch:
			stop()
			return true
		case <-w:
		}
	}
}

// leftover is one thing a run still owns after runS6 returned.
type leftover struct {
	what string
	// until waits until it has ended or the deadline passes, and says
	// whether it has ended.
	until func(deadline time.Time) bool
}

// runS6 is the output step: phase D (skipped when the tree is not
// confirmed), stopping the read, phase J, then sealing every copy port of
// the run, whatever came before. Its phases are bounded and measured on
// Go's clock.
func runS6(d *dest, ss []*stream, tDrain, tJoin time.Duration, skipDrain bool) s6Result {
	return runS6On(goClock{}, d, ss, tDrain, tJoin, skipDrain)
}

// runS6On is runS6 on clk: each phase ends on its deadline only once clk
// reads at least its start plus its length, and its length is read on clk.
//
// Phase J starts with the stop of the reads (Review 081 B1): T_join covers
// closing the read ends as well as the copiers' return, and the seal comes
// at its end whatever is still running. When the run returns, for each
// stopped stream:
//
//   - the stop was requested (always);
//   - the stop is done or pending (res.stop);
//   - the copier has returned, or not, which is O3 (res.out);
//   - the copy port is sealed (always): nothing more reaches the
//     destination from this run;
//   - what is still running is handed to the caller (res.left), who holds
//     it until it ends; whether a later run may start meanwhile is the
//     caller's policy (lingering, C-4 / D20).
func runS6On(clk phaseClock, d *dest, ss []*stream, tDrain, tJoin time.Duration, skipDrain bool) s6Result {
	res := s6Result{out: map[string]outcome{}, stop: map[string]string{}, stopErr: map[string]error{}}
	all := func(deadline time.Time) bool {
		ok := true
		for _, s := range ss {
			if !s.c.waitOn(clk, deadline) {
				ok = false
			}
		}
		return ok
	}
	ended := false
	if !skipDrain {
		start, fine := clk.now(), stopwatch()
		ended = all(start.Add(tDrain))
		res.drain, res.drainFine = clk.now().Sub(start), fine()
	}
	stops := map[string]*stopTask{}
	if !ended {
		res.stopped = true
		for _, s := range ss {
			if !s.c.waitOn(clk, clk.now()) {
				res.pending++
			}
		}
		start, fine := clk.now(), stopwatch()
		deadline := start.Add(tJoin)
		for _, s := range ss {
			stops[s.name] = startStop(s)
		}
		all(deadline)
		for _, s := range ss {
			awaitOn(clk, stops[s.name].done, deadline)
		}
		res.join, res.joinFine = clk.now().Sub(start), fine()
	}
	ps := make([]*port, 0, len(ss))
	for _, s := range ss {
		ps = append(ps, s.p)
	}
	d.seal(ps...)
	for _, s := range ss {
		switch {
		case s.c.res == nil:
			res.out[s.name] = o3
			res.left = append(res.left, leftover{what: "the copier of " + s.name, until: s.c.wait})
		case s.c.res.end == endEOF:
			res.out[s.name] = o1
		default:
			res.out[s.name] = o2
		}
		t := stops[s.name]
		switch {
		case t == nil:
			// It ended in phase D: no read holds the file, so the close
			// returns at once.
			_ = s.r.Close()
		case t.until(time.Now()):
			res.stop[s.name] = "done"
			if t.err != nil {
				res.stopErr[s.name] = t.err
			}
		default:
			res.stop[s.name] = "pending"
			res.left = append(res.left, leftover{what: "the stop of " + s.name, until: t.until})
		}
	}
	return res
}

// gateReader blocks in Read until released and ignores Close: the copier
// the spec cannot rule out (§8.5 S6, O3). Released, it returns one more
// chunk, so the copier's next write meets the sealed port, then EOF.
type gateReader struct {
	gate  chan struct{}
	once  sync.Once
	given bool
}

func newGate() *gateReader { return &gateReader{gate: make(chan struct{})} }

func (g *gateReader) Read(p []byte) (int, error) {
	<-g.gate
	if !g.given {
		g.given = true
		return copy(p, "late-after-release\n"), nil
	}
	return 0, io.EOF
}

func (g *gateReader) Close() error { return nil }
func (g *gateReader) release()     { g.once.Do(func() { close(g.gate) }) }

// endless keeps producing and ignores Close: a copier that never returns
// while its port takes the writes.
type endless struct {
	n    int
	stop chan struct{}
}

func (e *endless) Read(p []byte) (int, error) {
	select {
	case <-e.stop:
		return 0, io.EOF
	default:
	}
	time.Sleep(time.Millisecond)
	e.n++
	return copy(p, "chunk\n"), nil
}

func (e *endless) Close() error { return nil }

// goroutinesSettle waits for the goroutine count to fall back to at most n.
func goroutinesSettle(n int, d time.Duration) int {
	deadline := time.Now().Add(d)
	for {
		g := runtime.NumGoroutine()
		if g <= n || time.Now().After(deadline) {
			return g
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// pipePair gives an os.Pipe, or stops the scenario.
func pipePair(s *scen) (*os.File, *os.File) {
	r, w, err := os.Pipe()
	if err != nil {
		s.invalidf("pipe: %v", err)
	}
	return r, w
}
