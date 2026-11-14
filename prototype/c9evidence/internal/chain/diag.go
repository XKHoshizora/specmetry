package chain

import (
	"io"
	"sort"
	"time"
)

// DiagState is the diagnostic task's terminal state.
type DiagState int

// The diagnostic task's terminal states.
const (
	DiagCompleteWithTerminator    DiagState = iota + 1 // every line, ending with the terminator line
	DiagCompleteWithoutTerminator                      // every line; the writer failed, so no terminator
	DiagWriteFailed                                    // a write returned an error; writing stopped
	DiagExpiredStopped                                 // expiry exit before a write that was still due
)

func (s DiagState) String() string {
	switch s {
	case DiagCompleteWithTerminator:
		return "complete-with-terminator"
	case DiagCompleteWithoutTerminator:
		return "complete-without-terminator"
	case DiagWriteFailed:
		return "write-failed"
	case DiagExpiredStopped:
		return "expired-stopped"
	}
	return "none"
}

// DiagWrite is the published result of one diagnostic write.
type DiagWrite struct {
	Outcome
	Line string // the line asked to be written, LF included
	N    int    // bytes the write reported (clamped to the line)
}

// DiagTerminal is the diagnostic task's published terminal state.
type DiagTerminal struct {
	State DiagState
	At    time.Time
	Qual  Qualification
}

// DiagSnapshot is a copy of the diagnostic task's record.
type DiagSnapshot struct {
	Provisional Conclusion
	Writes      []DiagWrite
	Terminal    *DiagTerminal
}

const (
	diagPhaseSnapshot = iota + 1 // provisional, failure and pending lines
	diagPhaseChain               // hash lines and terminator, or writer errors
)

// diag is the unique diagnostic task (§6.4 step 4; implementation plan
// §12.14 item 13). It is registered by the parent at the snapshot point and
// started after the parent releases the outer mutex. Each write is preceded
// by a decision point under the outer mutex; the result of a write and the
// decision about the next one are made in the same lock. Between the
// snapshot lines and the chain lines it waits for the writer's terminal
// state on the registry's changes (never on an external call).
type diag struct {
	o    *Outer
	out  io.Writer
	prov Conclusion

	// Task state, used only by the diagnostic goroutine (set before it starts).
	queue      []string
	phase      int
	terminator bool

	// Published state, guarded by o.mu.
	writes   []DiagWrite
	terminal *DiagTerminal
}

func newDiag(o *Outer, out io.Writer, prov Conclusion, failures, pending []string) *diag {
	q := []string{ProvisionalLine(prov), CountLine("failure", len(failures))}
	for _, f := range failures {
		q = append(q, TextLine("failure", f))
	}
	q = append(q, CountLine("pending", len(pending)))
	for _, p := range pending {
		q = append(q, TextLine("pending", p))
	}
	return &diag{o: o, out: out, prov: prov, queue: q, phase: diagPhaseSnapshot}
}

func (d *diag) snapshotLocked() DiagSnapshot {
	s := DiagSnapshot{Provisional: d.prov, Writes: append([]DiagWrite(nil), d.writes...)}
	if d.terminal != nil {
		t := *d.terminal
		s.Terminal = &t
	}
	return s
}

// chainLines are the lines that follow the writer's terminal state: the
// record and manifest hashes and the terminator for success; the published
// errors for failure (and, defensively, for an expired writer, which the
// latch normally stops before).
func (d *diag) chainLinesLocked(w *Writer) []string {
	t := w.terminal
	if t.State != WriterSuccess {
		lines := []string{CountLine("record-chain-error", len(w.errs))}
		for _, e := range w.errs {
			lines = append(lines, TextLine("record-chain-error", e.Error()))
		}
		return lines
	}
	files := append([]FileSnapshot(nil), w.files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	var lines []string
	for _, f := range files {
		lines = append(lines, RecordLine(f.Name, f.Bytes, f.SHA256))
	}
	lines = append(lines, ManifestHashLine(w.manifest.Bytes, w.manifest.SHA256))
	d.terminator = true
	return append(lines, TerminatorLine(w.manifest.SHA256, d.prov))
}

type diagResult struct {
	line string
	n    int
	err  error
}

func (d *diag) run() {
	o := d.o
	var res *diagResult
	for {
		o.mu.Lock()
		now, q := o.checkLocked()
		if res != nil {
			d.writes = append(d.writes, DiagWrite{Outcome: Outcome{Err: res.err, At: now, Qual: q}, Line: res.line, N: res.n})
			if res.err != nil {
				d.finishLocked(DiagWriteFailed, now, q)
				o.mu.Unlock()
				return
			}
			res = nil
			o.publishLocked()
		}
		if len(d.queue) == 0 && d.phase == diagPhaseChain {
			// Nothing more to write: the terminal state carries this
			// lock's qualification.
			st := DiagCompleteWithoutTerminator
			if d.terminator {
				st = DiagCompleteWithTerminator
			}
			d.finishLocked(st, now, q)
			o.mu.Unlock()
			return
		}
		if len(d.queue) == 0 && (o.writer == nil || o.writer.terminal == nil) {
			// The snapshot lines are written and the writer has no
			// terminal state yet: keep waiting on the registry's changes.
			// The wait is not a decision point; the re-check comes in the
			// lock where the wait ends.
			ch := o.changed
			o.mu.Unlock()
			<-ch
			continue
		}
		if q == Late {
			// A write is due (the next snapshot line, or the first chain
			// line after the wait ended): expiry exit.
			d.finishLocked(DiagExpiredStopped, now, q)
			o.mu.Unlock()
			return
		}
		if len(d.queue) == 0 {
			d.queue = d.chainLinesLocked(o.writer)
			d.phase = diagPhaseChain
		}
		line := d.queue[0]
		d.queue = d.queue[1:]
		o.mu.Unlock()

		n, err := d.out.Write([]byte(line))
		n, err = normWrite(n, err, len(line))
		res = &diagResult{line: line, n: n, err: err}
	}
}

func (d *diag) finishLocked(st DiagState, now time.Time, q Qualification) {
	d.terminal = &DiagTerminal{State: st, At: now, Qual: q}
	d.o.publishLocked()
}
