package collect

import (
	"sync"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
)

// Clock is the monotonic clock the tool reads; the formal run uses the real
// one, tests and controls a fake.
type Clock interface{ Now() time.Time }

// Budget is the shared budget of a job (implementation plan §12.14): the
// single absolute deadline D taken at TestMain's start, W = D − 10 s, and
// the admission of work units. A refused unit is never started; it records
// "budget insufficient" (J-1, source budget) and everything that depends
// on it stays unevaluated (LC-8).
type Budget struct {
	Clock Clock
	W, D  time.Time

	mu      sync.Mutex
	refused []judge.Failure
}

// CallMargin is the admission margin of a tool call: W − now ≥ 5 s.
const CallMargin = 5 * time.Second

// NewBudget makes a budget whose D is start + total.
func NewBudget(c Clock, start time.Time, total time.Duration) *Budget {
	d := start.Add(total)
	return &Budget{Clock: c, D: d, W: d.Add(-10 * time.Second)}
}

// AdmitCall admits a tool call when at least CallMargin remains before W.
func (b *Budget) AdmitCall(unit string) bool {
	if b.W.Sub(b.Clock.Now()) >= CallMargin {
		return true
	}
	b.refuse(unit)
	return false
}

// AdmitUnit admits a package, module, fixture or configuration unit while
// time remains before W.
func (b *Budget) AdmitUnit(unit string) bool {
	if b.W.Sub(b.Clock.Now()) > 0 {
		return true
	}
	b.refuse(unit)
	return false
}

// PastW reports whether the clock reads at or after W (a result published
// then is late).
func (b *Budget) PastW(t time.Time) bool { return !t.Before(b.W) }

func (b *Budget) refuse(unit string) {
	b.mu.Lock()
	b.refused = append(b.refused, judge.Failure{Class: judge.J1, Item: unit, Site: "admission " + unit, Kind: "budget insufficient", Source: judge.SourceBudget})
	b.mu.Unlock()
}

// Refused returns every refusal, in order.
func (b *Budget) Refused() []judge.Failure {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]judge.Failure(nil), b.refused...)
}
