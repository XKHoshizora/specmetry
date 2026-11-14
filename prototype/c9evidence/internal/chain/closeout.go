package chain

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Record is a record item the parent appends at the snapshot point.
type Record struct {
	File string
	Line []byte
}

// Snapshot is what the caller's Provisional function computes at the
// snapshot point: the provisional conclusion by §7.1, the closeout record
// items (the snapshot's summary), and the failure and pending lines the
// diagnostic task writes.
type Snapshot struct {
	Conclusion Conclusion
	Records    []Record
	Failures   []string
	Pending    []string
}

// TerminalSource reports whether one registry's records are all terminal;
// when not, it also returns a channel closed at that registry's next change
// (a nil channel means "no notification": only W ends the wait).
type TerminalSource func() (done bool, changed <-chan struct{})

// ProvisionalInput is what the Provisional function receives, under the
// outer mutex: the copies taken before the lock, a copy of the outer
// registry (with the writer's published state, so a record-chain failure
// published before the snapshot counts), whether every non-closeout record
// was terminal when the wait ended, and m.Run's code.
type ProvisionalInput[S any] struct {
	Copied      S
	Outer       OuterSnapshot
	AllTerminal bool
	MRunCode    int
}

// Config is the parent's input to Closeout. D and the clock are the
// Outer's (one absolute deadline for everything).
type Config[S any] struct {
	Outer *Outer
	Log   *Log
	// W is the work deadline, D - CleanupMargin.
	W time.Time
	// MRunCode is the value m.Run returned. Non-zero is a level-1 condition.
	MRunCode int
	// Terminal are the R and F registries (and any other non-closeout
	// records). The outer operations are always included.
	Terminal []TerminalSource
	// Copy takes the in-memory copies of the R registry and the controls
	// table after the snapshot wait, before the outer mutex is taken.
	Copy func() S
	// Provisional computes the snapshot result by §7.1. It runs with the
	// outer mutex held and must not call methods of Outer or Op.
	Provisional func(ProvisionalInput[S]) Snapshot
	// Level1 reports, from already-published state, whether a level-1
	// (invalid) condition holds; it is used by the expiry exits that come
	// before a provisional conclusion. It runs with the outer mutex held and
	// must not call methods of Outer or Op.
	Level1 func(OuterSnapshot) bool
	// CloseoutFile is the record file of the parent's own closeout item.
	CloseoutFile string
	// Stderr is the diagnostic task's output (os.Stderr in the tool).
	Stderr io.Writer

	// Test seams: hook runs outside the lock at the parent's wake points;
	// trace runs under the outer mutex at each parent decision.
	hook  func(point)
	trace func(traceEvent)
}

type point int

const (
	pointMRunReturn       point = iota + 1 // before step 1's lock
	pointSnapshotWaitDone                  // after step 2's wait, before the copies
	pointFinalWake                         // before each lock of step 5
)

type traceEvent struct {
	step         point
	branch       string // "pass" or "expired" (steps 1, 2); "a", "b" or "c" (step 5)
	now          time.Time
	gen          uint64
	latch        bool
	diagTerminal bool
}

// ExitPath says which exit Closeout took.
type ExitPath int

// The exits.
const (
	PathConfig            ExitPath = iota + 1 // invalid input; nothing was done
	PathExpiredAtMRun                         // step 1: D had passed when m.Run returned
	PathExpiredAtSnapshot                     // step 2: D had passed when the snapshot wait ended
	PathConsumed                              // step 5 (a): the diagnostic terminal state was consumed
	PathExpiredFinal                          // step 5 (b): no diagnostic terminal state, D passed
)

func (p ExitPath) String() string {
	switch p {
	case PathConfig:
		return "config"
	case PathExpiredAtMRun:
		return "expired-at-m.Run"
	case PathExpiredAtSnapshot:
		return "expired-at-snapshot"
	case PathConsumed:
		return "consumed"
	case PathExpiredFinal:
		return "expired-final"
	}
	return "none"
}

// Final is Closeout's decision. ExitCode is 0 if and only if Conclusion is
// 3 or 4. Provisional is 0 when the exit came before the snapshot point.
// CallerProvisional is what the Provisional function returned and Caps the
// conditions that lowered it; Reasons says why the final conclusion is what
// it is.
type Final struct {
	Conclusion        Conclusion
	ExitCode          int
	Path              ExitPath
	Provisional       Conclusion
	CallerProvisional Conclusion
	Caps              []string
	Reasons           []string
	DecidedAt         time.Time
}

func configDefect(msg string) Final {
	return Final{Conclusion: Invalid, ExitCode: ExitNonzero, Path: PathConfig, Reasons: []string{"closeout: " + msg}}
}

func (cfg *Config[S]) at(p point) {
	if cfg.hook != nil {
		cfg.hook(p)
	}
}

func (cfg *Config[S]) emitLocked(p point, branch string, now time.Time) {
	if cfg.trace == nil {
		return
	}
	o := cfg.Outer
	cfg.trace(traceEvent{step: p, branch: branch, now: now, gen: o.gen, latch: o.latch, diagTerminal: o.diag != nil && o.diag.terminal != nil})
}

func (cfg *Config[S]) validate() string {
	switch {
	case cfg.Outer == nil || cfg.Log == nil:
		return "no outer registry or log"
	case cfg.Copy == nil || cfg.Provisional == nil || cfg.Level1 == nil:
		return "Copy, Provisional and Level1 are required"
	case cfg.Stderr == nil:
		return "no diagnostic output"
	case cfg.W.After(cfg.Outer.d):
		return "W is after D"
	}
	known := false
	for _, f := range cfg.Log.Files() {
		known = known || f == cfg.CloseoutFile
	}
	if !known {
		return fmt.Sprintf("closeout file %q is not a record file", cfg.CloseoutFile)
	}
	for _, s := range cfg.Terminal {
		if s == nil {
			return "nil terminal source"
		}
	}
	return ""
}

func (cfg *Config[S]) validateLocked() string {
	o := cfg.Outer
	switch {
	case o.closeout.started:
		return "closeout already ran"
	case o.writer == nil:
		return "no writer registered"
	case o.writer.log != cfg.Log:
		return "the writer writes another log"
	case !o.writer.started:
		return "the writer was not started"
	}
	if sealed, _ := cfg.Log.Sealed(); sealed {
		return "the log is already sealed"
	}
	return ""
}

// Closeout is the parent's closeout after m.Run returns (plan §6.4, the
// five steps; §7.1). It returns the final conclusion and the exit code; it
// never calls os.Exit. Once it has decided it returns at once: it does not
// print, flush, close, query or wait for anything.
//
//  1. m.Run returned: decision point; past D → expiry exit (no provisional).
//  2. Snapshot point: wait until every non-closeout record is terminal or W
//     (no wait if m.Run returned at or after W); take the copies; decision
//     point (past D → expiry exit, even if step 1 passed); in the same lock
//     compute the provisional conclusion, append the closeout items, seal
//     the log (the writer's closeout signal) and register the diagnostic
//     task; release the lock and start it.
//  3. and 4. are the writer's and the diagnostic task's.
//  5. Final consumption: wait on the registry's changes and the original D
//     timer; on every wake, under the lock read the clock (which sets the
//     latch at or past D; that alone is not the failure exit) and then, in
//     this order only: (a) the diagnostic task has a terminal state →
//     consume it; (b) else past D or latch set → expiry exit; (c) else keep
//     waiting.
func Closeout[S any](cfg Config[S]) Final {
	if msg := cfg.validate(); msg != "" {
		return configDefect(msg)
	}
	o := cfg.Outer

	// Step 1.
	cfg.at(pointMRunReturn)
	o.mu.Lock()
	if msg := cfg.validateLocked(); msg != "" {
		o.defectLocked("closeout: " + msg)
		o.publishLocked()
		o.mu.Unlock()
		return configDefect(msg)
	}
	o.closeout.started = true
	now, q := o.checkLocked()
	if q == Late {
		f := cfg.expiredLocked(PathExpiredAtMRun, now)
		cfg.emitLocked(pointMRunReturn, "expired", now)
		o.mu.Unlock()
		return f
	}
	cfg.emitLocked(pointMRunReturn, "pass", now)
	o.publishLocked()
	o.mu.Unlock()

	// Step 2.
	allTerminal := cfg.waitSnapshot(now)
	cfg.at(pointSnapshotWaitDone)
	copied := cfg.Copy()
	o.mu.Lock()
	now, q = o.checkLocked()
	if q == Late {
		f := cfg.expiredLocked(PathExpiredAtSnapshot, now)
		cfg.emitLocked(pointSnapshotWaitDone, "expired", now)
		o.mu.Unlock()
		return f
	}
	snap := o.snapshotLocked()
	s := cfg.Provisional(ProvisionalInput[S]{Copied: copied, Outer: snap, AllTerminal: allTerminal, MRunCode: cfg.MRunCode})
	var good []Record
	var rejected []string
	for _, r := range s.Records {
		if err := cfg.Log.checkItem(r.File, r.Line); err != nil {
			rejected = append(rejected, err.Error())
			continue
		}
		good = append(good, r)
	}
	prov, caps := cfg.capProvisional(s.Conclusion, snap, allTerminal, rejected)
	own := closeoutItem(prov, s, cfg.MRunCode, allTerminal, caps, o.d.Sub(now))
	sealSeq, lastPerFile, _ := cfg.Log.sealWith(append(good, Record{File: cfg.CloseoutFile, Line: own}))
	failures := append([]string(nil), s.Failures...)
	for _, c := range caps {
		failures = append(failures, "closeout: "+c)
	}
	d := newDiag(o, cfg.Stderr, prov, failures, s.Pending)
	o.closeout.signaled = true
	o.closeout.provisional = prov
	o.closeout.sealSeq = sealSeq
	o.diag = d
	o.publishLocked()
	cfg.emitLocked(pointSnapshotWaitDone, "pass", now)
	o.mu.Unlock()
	go d.run()

	// Step 5.
	dFired, _ := o.clock.Timer(o.d)
	for {
		cfg.at(pointFinalWake)
		o.mu.Lock()
		now, _ := o.checkLocked()
		if o.diag.terminal != nil {
			f := consumeLocked(o, prov, lastPerFile)
			f.CallerProvisional, f.Caps, f.DecidedAt = s.Conclusion, caps, now
			cfg.emitLocked(pointFinalWake, "a", now)
			o.mu.Unlock()
			return f
		}
		if o.latch {
			f := Final{Conclusion: prov, Path: PathExpiredFinal, Provisional: prov}
			if prov == Inconsistent || prov == Consistent {
				f.Conclusion = Incomplete
				f.Reasons = []string{"the diagnostic task had no terminal state at D"}
			}
			f.ExitCode = ExitCode(f.Conclusion)
			f.CallerProvisional, f.Caps, f.DecidedAt = s.Conclusion, caps, now
			cfg.emitLocked(pointFinalWake, "b", now)
			o.mu.Unlock()
			return f
		}
		ch := o.changed
		cfg.emitLocked(pointFinalWake, "c", now)
		o.mu.Unlock()
		select {
		case <-ch:
		case <-dFired:
		}
	}
}

// waitSnapshot waits until every terminal source is done or W, whichever
// comes first, and reports whether every source was done when it ended. It
// does not wait when m.Run returned at or after W.
func (cfg *Config[S]) waitSnapshot(now time.Time) bool {
	sources := append([]TerminalSource{cfg.Outer.OpsTerminal}, cfg.Terminal...)
	firstPending := func() (bool, <-chan struct{}) {
		for _, src := range sources {
			if done, ch := src(); !done {
				return false, ch
			}
		}
		return true, nil
	}
	if !now.Before(cfg.W) {
		done, _ := firstPending()
		return done
	}
	wFired, stop := cfg.Outer.clock.Timer(cfg.W)
	defer stop()
	for {
		done, ch := firstPending()
		if done {
			return true
		}
		select {
		case <-ch:
		case <-wFired:
			done, _ := firstPending()
			return done
		}
	}
}

// expiredLocked is the expiry exit before a provisional conclusion exists:
// from the published state only, 1 when a level-1 condition holds, else 2.
func (cfg *Config[S]) expiredLocked(path ExitPath, now time.Time) Final {
	snap := cfg.Outer.snapshotLocked()
	var why []string
	if cfg.MRunCode != 0 {
		why = append(why, fmt.Sprintf("m.Run returned %d", cfg.MRunCode))
	}
	if len(snap.Defects) > 0 {
		why = append(why, fmt.Sprintf("%d outer registry defects", len(snap.Defects)))
	}
	if n := len(cfg.Log.Defects()); n > 0 {
		why = append(why, fmt.Sprintf("%d log defects", n))
	}
	if cfg.Level1(snap) {
		why = append(why, "level-1 condition in the published state")
	}
	c := Incomplete
	if len(why) > 0 {
		c = Invalid
	}
	why = append(why, "deadline passed before a provisional conclusion")
	return Final{Conclusion: c, ExitCode: ExitCode(c), Path: path, Reasons: why, DecidedAt: now}
}

// capProvisional lowers the caller's conclusion by the conditions the chain
// itself can see. It only lowers.
func (cfg *Config[S]) capProvisional(raw Conclusion, snap OuterSnapshot, allTerminal bool, rejected []string) (Conclusion, []string) {
	c := raw
	caps := []string{}
	lower := func(to Conclusion, why string) {
		caps = append(caps, why)
		if !c.Valid() || c > to {
			c = to
		}
	}
	if !raw.Valid() {
		lower(Invalid, fmt.Sprintf("provisional function returned %d", int(raw)))
	}
	if cfg.MRunCode != 0 {
		lower(Invalid, fmt.Sprintf("m.Run returned %d", cfg.MRunCode))
	}
	for _, d := range snap.Defects {
		lower(Invalid, "outer registry defect: "+d)
	}
	for _, d := range cfg.Log.Defects() {
		lower(Invalid, "log defect: "+d)
	}
	for _, r := range rejected {
		lower(Invalid, "closeout record rejected: "+r)
	}
	if !allTerminal {
		lower(Incomplete, "not every non-closeout record was terminal at the snapshot point")
	}
	for _, op := range snap.Ops {
		if op.State != OpDone {
			lower(Incomplete, fmt.Sprintf("outer op %q is %s", op.Name, op.State))
		}
	}
	if w := snap.Writer; w != nil {
		for _, e := range w.Errors {
			lower(Incomplete, "record chain: "+e.Error())
		}
		if w.Terminal != nil && w.Terminal.State != WriterSuccess {
			lower(Incomplete, "record chain terminal state "+w.Terminal.State.String())
		}
	}
	return c, caps
}

type closeoutLine struct {
	Closeout    string   `json:"closeout"`
	Conclusion  int      `json:"conclusion"`
	Name        string   `json:"name"`
	Caller      int      `json:"caller"`
	MRunCode    int      `json:"m_run_code"`
	AllTerminal bool     `json:"all_terminal"`
	Caps        []string `json:"caps"`
	Failures    int      `json:"failures"`
	Pending     int      `json:"pending"`
	RemainingNS int64    `json:"remaining_ns"`
}

// closeoutItem is the parent's own closeout item, the last item inside the
// seal:
//
//	{"closeout":"provisional","conclusion":N,"name":"...","caller":N,"m_run_code":N,
//	 "all_terminal":B,"caps":[...],"failures":N,"pending":N,"remaining_ns":N}
//
// remaining_ns is D minus the snapshot decision point's reading.
func closeoutItem(prov Conclusion, s Snapshot, code int, allTerminal bool, caps []string, remaining time.Duration) []byte {
	b, err := json.Marshal(closeoutLine{
		Closeout:    "provisional",
		Conclusion:  int(prov),
		Name:        prov.Name(),
		Caller:      int(s.Conclusion),
		MRunCode:    code,
		AllTerminal: allTerminal,
		Caps:        caps,
		Failures:    len(s.Failures),
		Pending:     len(s.Pending),
		RemainingNS: int64(remaining),
	})
	if err != nil {
		// Unreachable for this type; keep a valid line regardless.
		return []byte(`{"closeout":"provisional","conclusion":` + fmt.Sprint(int(prov)) + `}`)
	}
	return b
}

// consumeLocked is branch (a): a provisional 1 or 2 stays; a provisional 3
// or 4 stays if and only if every required writer and diagnostic result was
// published, succeeded and is timely, otherwise it becomes 2.
func consumeLocked(o *Outer, prov Conclusion, lastPerFile map[string]int) Final {
	f := Final{Conclusion: prov, Path: PathConsumed, Provisional: prov}
	if prov == Inconsistent || prov == Consistent {
		ws := o.writer.snapshotLocked()
		ds := o.diag.snapshotLocked()
		if why := requiredReasons(&ws, &ds, lastPerFile); len(why) > 0 {
			f.Conclusion = Incomplete
			f.Reasons = why
		}
	}
	f.ExitCode = ExitCode(f.Conclusion)
	return f
}

// requiredReasons lists every required result (plan §7.1) that was not
// published, failed or is late. An empty list means 3 or 4 is kept.
func requiredReasons(w *WriterSnapshot, d *DiagSnapshot, lastPerFile map[string]int) []string {
	var why []string
	need := func(what string, oc *Outcome) {
		switch {
		case oc == nil:
			why = append(why, what+": not published")
		case oc.Err != nil:
			why = append(why, what+": "+oc.Err.Error())
		case oc.Qual != Timely:
			why = append(why, what+": "+oc.Qual.String())
		}
	}
	need("mkdir", w.Mkdir)
	for _, f := range w.Files {
		need("create "+f.Name, f.Create)
		for k, wr := range f.Writes {
			oc := wr.Outcome
			need(fmt.Sprintf("write %s #%d", f.Name, k+1), &oc)
			if wr.N != wr.Requested {
				why = append(why, fmt.Sprintf("write %s #%d: %d of %d bytes", f.Name, k+1, wr.N, wr.Requested))
			}
		}
		if !f.HeaderWritten {
			why = append(why, "header of "+f.Name+" not written")
		}
		if f.WrittenThrough < lastPerFile[f.Name] {
			why = append(why, fmt.Sprintf("%s written through item %d of %d", f.Name, f.WrittenThrough, lastPerFile[f.Name]))
		}
		need("close "+f.Name, f.Close)
	}
	need("create "+ManifestName, w.Manifest.Create)
	if w.Manifest.Write == nil {
		why = append(why, "write "+ManifestName+": not published")
	} else {
		oc := w.Manifest.Write.Outcome
		need("write "+ManifestName, &oc)
		if w.Manifest.Write.N != w.Manifest.Write.Requested {
			why = append(why, "write "+ManifestName+": short")
		}
	}
	need("close "+ManifestName, w.Manifest.Close)
	switch t := w.Terminal; {
	case t == nil:
		why = append(why, "writer terminal state: not published")
	case t.State != WriterSuccess:
		why = append(why, "writer terminal state: "+t.State.String())
	case t.Qual != Timely:
		why = append(why, "writer terminal state: "+t.Qual.String())
	}
	for _, e := range w.Errors {
		why = append(why, "record chain: "+e.Error())
	}
	for k, wr := range d.Writes {
		oc := wr.Outcome
		need(fmt.Sprintf("diagnostic write #%d", k+1), &oc)
		if wr.N != len(wr.Line) {
			why = append(why, fmt.Sprintf("diagnostic write #%d: %d of %d bytes", k+1, wr.N, len(wr.Line)))
		}
	}
	switch t := d.Terminal; {
	case t == nil:
		why = append(why, "diagnostic terminal state: not published")
	case t.State != DiagCompleteWithTerminator:
		why = append(why, "diagnostic terminal state: "+t.State.String())
	case t.Qual != Timely:
		why = append(why, "diagnostic terminal state: "+t.Qual.String())
	}
	return why
}
