package chain

import (
	"fmt"
	"sync"
	"time"
)

// Outer is the outer registry (外层登记表, plan §6.4): the deadline D, the
// expiry latch, the change notification and the records of the writer, the
// diagnostic task and the outer (O-domain) operations. It holds every record
// until the process ends (the persistent holder); records are append-only.
type Outer struct {
	clock Clock
	d     time.Time

	mu       sync.Mutex
	changed  chan struct{}
	gen      uint64
	latch    bool
	latchAt  time.Time
	writer   *Writer
	diag     *diag
	ops      []opRec
	defects  []string
	closeout closeoutRec
}

type closeoutRec struct {
	started     bool
	signaled    bool
	provisional Conclusion
	sealSeq     int
}

// NewOuter makes the outer registry with the absolute deadline d.
func NewOuter(clock Clock, d time.Time) *Outer {
	if clock == nil {
		panic("chain: NewOuter with a nil clock")
	}
	return &Outer{clock: clock, d: d, changed: make(chan struct{})}
}

// Deadline returns D.
func (o *Outer) Deadline() time.Time { return o.d }

// Clock returns the registry's clock.
func (o *Outer) Clock() Clock { return o.clock }

// checkLocked is a logical decision point: it reads the clock under o.mu,
// sets the latch if the reading is at or past D (it is never cleared), and
// returns the reading with the qualification a result published here gets:
// Timely if the latch is unset and the reading is before D, Late otherwise.
// For a decision about a new action, Late means "take the expiry exit".
func (o *Outer) checkLocked() (time.Time, Qualification) {
	now := o.clock.Now()
	if !now.Before(o.d) && !o.latch {
		o.latch = true
		o.latchAt = now
	}
	if o.latch {
		return now, Late
	}
	return now, Timely
}

// publishLocked announces a publication: it closes the current changed
// channel and replaces it.
func (o *Outer) publishLocked() {
	o.gen++
	close(o.changed)
	o.changed = make(chan struct{})
}

func (o *Outer) defectLocked(msg string) { o.defects = append(o.defects, msg) }

// Changed returns the channel that is closed at the next publication.
func (o *Outer) Changed() <-chan struct{} {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.changed
}

// Latched reports whether the expiry latch is set.
func (o *Outer) Latched() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.latch
}

// OpState is the state of an outer operation.
type OpState int

// The operation states. Every state except OpPending is terminal.
const (
	OpPending OpState = iota
	OpDone            // succeeded
	OpFailed          // a real error
	OpUnknown         // the result cannot be known (e.g. cleanup unknown)
	// OpRefused: the operation was not admitted (its deadline had passed),
	// so it never started. It is neither a success nor the operation's
	// failure (Review 148 R-c).
	OpRefused
)

func (s OpState) String() string {
	switch s {
	case OpPending:
		return "pending"
	case OpDone:
		return "done"
	case OpFailed:
		return "failed"
	case OpUnknown:
		return "unknown"
	case OpRefused:
		return "refused"
	}
	return "invalid"
}

type opRec struct {
	name    string
	state   OpState
	reason  string
	outcome *Outcome
}

// Op is a registered outer operation (a fixture preparation or cleanup step
// of the O domain, plan §5.4). It is pending until Done or Unknown publishes
// its terminal result; the parent's snapshot point waits for every Op.
type Op struct {
	o *Outer
	i int
}

// RegisterOp registers a pending outer operation. Registration is not
// admission: an operation that must not start after a deadline is
// registered through AdmitOp.
func (o *Outer) RegisterOp(name string) *Op {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.ops = append(o.ops, opRec{name: name})
	o.publishLocked()
	return &Op{o: o, i: len(o.ops) - 1}
}

// AdmitOp decides, right before a new outer operation starts, whether it
// may start (Review 148 R-c). Under the mutex it reads the clock (a
// decision point: a reading at or past D sets the latch) and admits the
// operation only when the reading is before until and the latch is not set;
// until is W for a preparation and D for a cleanup (a later until is capped
// by D). An admitted operation is registered pending, as RegisterOp does,
// and its caller publishes the real result with Done or Unknown. A refused
// one is registered terminal (OpRefused) with the reading and the deadline,
// returns nil and false, and nothing may start.
func (o *Outer) AdmitOp(name string, until time.Time) (*Op, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	now, q := o.checkLocked()
	if q == Timely && now.Before(until) {
		o.ops = append(o.ops, opRec{name: name})
		o.publishLocked()
		return &Op{o: o, i: len(o.ops) - 1}, true
	}
	o.ops = append(o.ops, opRec{
		name:    name,
		state:   OpRefused,
		reason:  fmt.Sprintf("not admitted: the reading %s is not before the deadline %s (latch %v); nothing was started", now.Format(time.RFC3339Nano), until.Format(time.RFC3339Nano), o.latch),
		outcome: &Outcome{At: now, Qual: q},
	})
	o.publishLocked()
	return nil, false
}

// Done publishes the operation's real result: OpDone when err is nil,
// OpFailed otherwise.
func (op *Op) Done(err error) {
	st := OpDone
	if err != nil {
		st = OpFailed
	}
	op.finish(st, err, "")
}

// Unknown publishes that the operation's result cannot be known.
func (op *Op) Unknown(reason string) { op.finish(OpUnknown, nil, reason) }

func (op *Op) finish(st OpState, err error, reason string) {
	o := op.o
	o.mu.Lock()
	defer o.mu.Unlock()
	r := &o.ops[op.i]
	now, q := o.checkLocked()
	if r.state != OpPending {
		o.defectLocked(fmt.Sprintf("outer op %q: second terminal result %s ignored; first was %s", r.name, st, r.state))
		o.publishLocked()
		return
	}
	r.state = st
	r.reason = reason
	r.outcome = &Outcome{Err: err, At: now, Qual: q}
	o.publishLocked()
}

// OpsTerminal reports whether every outer operation is terminal; when not,
// it also returns the channel closed at the next publication. Its signature
// is a TerminalSource.
func (o *Outer) OpsTerminal() (bool, <-chan struct{}) {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, r := range o.ops {
		if r.state == OpPending {
			return false, o.changed
		}
	}
	return true, nil
}

// OpSnapshot is a copy of one outer operation's record.
type OpSnapshot struct {
	Name    string
	State   OpState
	Reason  string
	Outcome *Outcome // nil while pending
}

// CloseoutSnapshot is the parent's published closeout state.
type CloseoutSnapshot struct {
	Started     bool       // Closeout has passed its first lock
	Signaled    bool       // the log is sealed and the writer signaled
	Provisional Conclusion // 0 until the snapshot point
	SealSeq     int
}

// OuterSnapshot is a consistent copy of the outer registry, taken under its
// mutex. Gen counts publications.
type OuterSnapshot struct {
	Deadline time.Time
	Gen      uint64
	Latch    bool
	LatchAt  time.Time // the reading that set the latch
	Writer   *WriterSnapshot
	Diag     *DiagSnapshot
	Ops      []OpSnapshot
	Defects  []string
	Closeout CloseoutSnapshot
}

// Snapshot returns a copy of the registry.
func (o *Outer) Snapshot() OuterSnapshot {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.snapshotLocked()
}

// Watch returns a copy of the registry and, from the same critical section,
// the channel closed at the next publication.
func (o *Outer) Watch() (OuterSnapshot, <-chan struct{}) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.snapshotLocked(), o.changed
}

func (o *Outer) snapshotLocked() OuterSnapshot {
	s := OuterSnapshot{
		Deadline: o.d,
		Gen:      o.gen,
		Latch:    o.latch,
		LatchAt:  o.latchAt,
		Defects:  append([]string(nil), o.defects...),
		Closeout: CloseoutSnapshot{
			Started:     o.closeout.started,
			Signaled:    o.closeout.signaled,
			Provisional: o.closeout.provisional,
			SealSeq:     o.closeout.sealSeq,
		},
	}
	if o.writer != nil {
		ws := o.writer.snapshotLocked()
		s.Writer = &ws
	}
	if o.diag != nil {
		ds := o.diag.snapshotLocked()
		s.Diag = &ds
	}
	for _, r := range o.ops {
		s.Ops = append(s.Ops, OpSnapshot{Name: r.name, State: r.state, Reason: r.reason, Outcome: copyOutcome(r.outcome)})
	}
	return s
}

func copyOutcome(p *Outcome) *Outcome {
	if p == nil {
		return nil
	}
	c := *p
	return &c
}
