package chain

import (
	"sync"
	"time"
)

// Clock is the time source of the record chain. Every logical decision
// point calls Now under the outer mutex. Timer returns a channel that is
// closed once the clock has reached at (immediately if it already has) and
// a stop function; waiting on the channel is how tasks wait for W or D.
type Clock interface {
	Now() time.Time
	Timer(at time.Time) (fired <-chan struct{}, stop func())
}

// RealClock is the clock of the tool: time.Now, whose monotonic reading
// makes the comparisons with D monotonic.
type RealClock struct{}

// Now returns time.Now().
func (RealClock) Now() time.Time { return time.Now() }

// Timer closes the channel when time.Until(at) has elapsed.
func (RealClock) Timer(at time.Time) (<-chan struct{}, func()) {
	ch := make(chan struct{})
	t := time.AfterFunc(time.Until(at), func() { close(ch) })
	return ch, func() { t.Stop() }
}

// FakeClock is a deterministic clock for tests: time moves only when the
// test calls Set or Advance, and timers fire then (a timer fires when the
// clock reaches its instant, so a timer at D fires at exactly D). It never
// moves backwards.
type FakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}

type fakeTimer struct {
	at time.Time
	ch chan struct{}
}

// NewFakeClock returns a fake clock reading start.
func NewFakeClock(start time.Time) *FakeClock { return &FakeClock{now: start} }

// Now returns the current fake time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Timer arms a timer at the instant at.
func (c *FakeClock) Timer(at time.Time) (<-chan struct{}, func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{at: at, ch: make(chan struct{})}
	if !at.After(c.now) {
		close(t.ch)
		return t.ch, func() {}
	}
	c.timers = append(c.timers, t)
	return t.ch, func() { c.stop(t) }
}

func (c *FakeClock) stop(t *fakeTimer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, x := range c.timers {
		if x == t {
			c.timers = append(c.timers[:i], c.timers[i+1:]...)
			return
		}
	}
}

// Set moves the clock to now and fires every timer whose instant has been
// reached. It panics if now is before the current time.
func (c *FakeClock) Set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setLocked(now)
}

func (c *FakeClock) setLocked(now time.Time) {
	if now.Before(c.now) {
		panic("chain: FakeClock moved backwards")
	}
	c.now = now
	keep := c.timers[:0]
	for _, t := range c.timers {
		if !t.at.After(now) {
			close(t.ch)
			continue
		}
		keep = append(keep, t)
	}
	for i := len(keep); i < len(c.timers); i++ {
		c.timers[i] = nil
	}
	c.timers = keep
}

// Advance moves the clock forward by d.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setLocked(c.now.Add(d))
}

// Armed returns the number of timers that have not fired or been stopped.
func (c *FakeClock) Armed() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.timers)
}
