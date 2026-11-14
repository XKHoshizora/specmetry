package chain

import (
	"errors"
	"strings"
	"testing"
)

// The seven walkthroughs of plan §6.6 and the five states of the parent's
// final consumption. Times are seconds from TestMain's start: W = 110,
// D = 120. Every ordering below comes from an explicit synchronization
// point.

// §6.6 case 1: m.Run returns at 109; the snapshot wait reaches W but the
// parent resumes only at 121. Step 1's check passed; step 2's check takes
// the expiry exit: no provisional conclusion, no seal, no closeout signal,
// no diagnostic task.
func TestCase1SnapshotWaitHitsWParentResumesAfterD(t *testing.T) {
	h := newHarness(t)
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(109))
	open := h.gate(pointSnapshotWaitDone)
	cfg := h.cfg(Consistent, nil, nil)
	cfg.Terminal = []TerminalSource{func() (bool, <-chan struct{}) { return false, nil }} // never terminal
	done := h.run(cfg)

	h.waitEvent("step 1 pass", func(ev traceEvent) bool { return ev.step == pointMRunReturn && ev.branch == "pass" })
	h.clk.Set(at(110)) // W: the wait ends
	h.waitArrived(pointSnapshotWaitDone)
	h.clk.Set(at(121)) // the parent is descheduled until 121
	open()
	f := h.final(done)
	wantFinal(t, f, Incomplete, PathExpiredAtSnapshot)
	if f.Provisional != 0 {
		t.Errorf("provisional %v, want none", f.Provisional)
	}
	if h.copyCount() != 1 || len(h.provInputs()) != 0 {
		t.Errorf("copies %d, provisional calls %d; want 1 and 0", h.copyCount(), len(h.provInputs()))
	}
	ev := h.waitEvent("step 2 expired", func(ev traceEvent) bool { return ev.step == pointSnapshotWaitDone })
	if ev.branch != "expired" || !ev.now.Equal(at(121)) || !ev.latch {
		t.Errorf("step 2 event %+v, want expired at 121 with the latch set", ev)
	}
	s := h.o.Snapshot()
	if s.Diag != nil || s.Closeout.Signaled {
		t.Errorf("diagnostic task registered or writer signaled: %+v", s.Closeout)
	}
	if sealed, _ := h.log.Sealed(); sealed {
		t.Error("log sealed on the expiry exit")
	}
	if s.Writer.Terminal != nil {
		t.Errorf("writer terminal %v; the writer never got the closeout signal", s.Writer.Terminal.State)
	}
	if strings.Contains(h.stderr.text(), "provisional") {
		t.Errorf("stderr has a provisional line: %q", h.stderr.text())
	}

	// The idle writer, woken by a late item, takes the expiry exit at its
	// next decision point without a new external call.
	writes := h.fs.count("write", "raw.jsonl")
	h.log.Append("raw.jsonl", []byte(`{"late":1}`))
	s = h.waitFor("writer expiry", writerTerminal)
	if st := s.Writer.Terminal; st.State != WriterExpired || st.Qual != Late {
		t.Errorf("writer terminal %v/%v, want expired-incomplete/late", st.State, st.Qual)
	}
	if got := h.fs.count("write", "raw.jsonl"); got != writes {
		t.Errorf("raw.jsonl writes %d after D, want %d", got, writes)
	}
	for _, name := range recordFiles {
		if n := h.fs.count("close", name); n != 0 {
			t.Errorf("%s closed %d times after D", name, n)
		}
	}
}

// manifestCloseLate sets up §6.6 case 2: provisional 4; the manifest's
// Close is admitted at 119.5 and blocks.
func manifestCloseLate(t *testing.T, gateParent bool) (*harness, *ctl, <-chan Final, func()) {
	h := newHarness(t)
	mw := h.fs.set(t, "write", ManifestName, 1, blocking())
	mc := h.fs.set(t, "close", ManifestName, 1, blocking())
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	open := func() {}
	if gateParent {
		open = h.gate(pointFinalWake)
	}
	done := h.run(h.cfg(Consistent, nil, nil))
	waitCh(t, mw.entered, "manifest write")
	h.clk.Set(at(119.5))
	mw.free()
	waitCh(t, mc.entered, "manifest close")
	s := h.o.Snapshot()
	if oc := s.Writer.Manifest.Write.Outcome; !oc.OK() || !oc.At.Equal(at(119.5)) {
		t.Fatalf("manifest write %+v, want timely at 119.5", oc)
	}
	return h, mc, done, open
}

// §6.6 case 2, order (a); five-state row 3: the parent's D timer takes the
// lock first; the diagnostic task has no terminal state → branch (b).
func TestCase2aManifestCloseLateParentFirst(t *testing.T) {
	h, mc, done, _ := manifestCloseLate(t, false)
	h.clk.Set(at(120.5))
	f := h.final(done)
	wantFinal(t, f, Incomplete, PathExpiredFinal)
	if f.Provisional != Consistent {
		t.Errorf("provisional %v, want 4", f.Provisional)
	}
	if br := h.finalBranches(); br[len(br)-1] != "b" {
		t.Errorf("final branches %v, want last b", br)
	}

	// The close returns nil at 121: late; the late success does not change
	// the decided conclusion, and the diagnostic task takes the expiry exit.
	h.clk.Set(at(121))
	mc.free()
	s := h.waitFor("diagnostic terminal", diagTerminal)
	if oc := s.Writer.Manifest.Close; oc.Err != nil || oc.Qual != Late {
		t.Errorf("manifest close %+v, want nil/late", oc)
	}
	if st := s.Writer.Terminal; st.State != WriterSuccess || st.Qual != Late {
		t.Errorf("writer terminal %v/%v, want success/late", st.State, st.Qual)
	}
	if st := s.Diag.Terminal; st.State != DiagExpiredStopped || st.Qual != Late {
		t.Errorf("diagnostic terminal %v/%v, want expired-stopped/late", st.State, st.Qual)
	}
	if got, want := h.stderr.text(), snapshotLines(Consistent, nil, nil); got != want {
		t.Errorf("stderr = %q, want only the snapshot lines %q", got, want)
	}
}

// §6.6 case 2, order (b); five-state row 2: the writer publishes the
// close and "success" at 121 first (late, latch); the diagnostic task's
// re-check takes the expiry exit; the parent then consumes it via (a).
func TestCase2bManifestCloseLateWriterFirst(t *testing.T) {
	h, mc, done, open := manifestCloseLate(t, true)
	h.waitArrived(pointFinalWake)
	h.clk.Set(at(121))
	mc.free()
	s := h.waitFor("diagnostic terminal", diagTerminal)
	if st := s.Writer.Terminal; st.State != WriterSuccess || st.Qual != Late || !st.At.Equal(at(121)) {
		t.Errorf("writer terminal %+v, want success/late at 121", st)
	}
	if st := s.Diag.Terminal; st.State != DiagExpiredStopped || st.Qual != Late {
		t.Errorf("diagnostic terminal %v/%v, want expired-stopped/late", st.State, st.Qual)
	}
	open()
	f := h.final(done)
	wantFinal(t, f, Incomplete, PathConsumed)
	if !hasReason(f.Reasons, "close manifest.txt: late") || !hasReason(f.Reasons, "expired-stopped") {
		t.Errorf("reasons %q", f.Reasons)
	}
	if br := h.finalBranches(); len(br) != 1 || br[0] != "a" {
		t.Errorf("final branches %v, want [a]", br)
	}
}

// §6.6 case 3; five-state row 1: every writer and diagnostic result is
// published timely (writer at 116, diagnostic at 118); the parent is
// scheduled only at 121. It sets the latch, takes branch (a) and keeps 4.
func TestCase3AllTimelyParentWakesAfterD(t *testing.T) {
	h := newHarness(t)
	mc := h.fs.set(t, "close", ManifestName, 1, blocking())
	term := h.stderr.on(t, 0, " end ", blocking())
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	open := h.gate(pointFinalWake)
	done := h.run(h.cfg(Consistent, nil, nil))
	h.waitArrived(pointFinalWake)
	waitCh(t, mc.entered, "manifest close")
	h.clk.Set(at(116))
	mc.free()
	waitCh(t, term.entered, "terminator write")
	h.clk.Set(at(118))
	term.free()
	s := h.waitFor("diagnostic terminal", diagTerminal)
	if st := s.Writer.Terminal; st.State != WriterSuccess || !st.At.Equal(at(116)) || st.Qual != Timely {
		t.Errorf("writer terminal %+v, want success/timely at 116", st)
	}
	if st := s.Diag.Terminal; st.State != DiagCompleteWithTerminator || !st.At.Equal(at(118)) || st.Qual != Timely {
		t.Errorf("diagnostic terminal %+v, want complete-with-terminator/timely at 118", st)
	}
	if s.Latch {
		t.Fatal("latch set before D")
	}

	h.clk.Set(at(121))
	open()
	f := h.final(done)
	wantFinal(t, f, Consistent, PathConsumed)
	if !f.DecidedAt.Equal(at(121)) {
		t.Errorf("decided at %v, want 121", f.DecidedAt)
	}
	s = h.o.Snapshot()
	if !s.Latch || !s.LatchAt.Equal(at(121)) {
		t.Errorf("latch %v at %v, want set by the parent at 121", s.Latch, s.LatchAt)
	}
	allQualified(t, s, Timely) // setting the latch did not rewrite a qualification
	ev := h.waitEvent("branch a", func(ev traceEvent) bool { return ev.step == pointFinalWake })
	if ev.branch != "a" || !ev.now.Equal(at(121)) || !ev.latch {
		t.Errorf("first final event %+v, want a at 121 with the latch set", ev)
	}
	content, mf := h.checkCompleteRecords()
	want := snapshotLines(Consistent, nil, nil) + successLines(content, mf, Consistent)
	if got := h.stderr.text(); got != want {
		t.Errorf("stderr =\n%s\nwant\n%s", got, want)
	}
}

// terminatorBytesBeforeD sets up §6.6 case 4: provisional 4; the
// terminator's write is admitted at 119.7, its bytes are in the output at
// 119.8, and the call has not returned.
func terminatorBytesBeforeD(t *testing.T, gateParent bool) (*harness, *ctl, <-chan Final, func()) {
	h := newHarness(t)
	mline := h.stderr.on(t, 0, LinePrefix+"manifest ", blocking())
	term := h.stderr.on(t, 0, " end ", blocking())
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	open := func() {}
	if gateParent {
		open = h.gate(pointFinalWake)
	}
	done := h.run(h.cfg(Consistent, nil, nil))
	waitCh(t, mline.entered, "manifest hash line")
	h.clk.Set(at(119.7))
	mline.free()
	waitCh(t, term.entered, "terminator write")
	h.clk.Set(at(119.8))
	if !strings.Contains(h.stderr.text(), " end manifest-sha256 ") {
		t.Fatal("terminator bytes not in the output before D")
	}
	return h, term, done, open
}

// §6.6 case 4, the parent first (120.1); five-state row 3.
func TestCase4aTerminatorBeforeDParentFirst(t *testing.T) {
	h, term, done, _ := terminatorBytesBeforeD(t, false)
	h.clk.Set(at(120.1))
	f := h.final(done)
	wantFinal(t, f, Incomplete, PathExpiredFinal)

	h.clk.Set(at(120.3))
	term.free()
	s := h.waitFor("diagnostic terminal", diagTerminal)
	last := s.Diag.Writes[len(s.Diag.Writes)-1]
	if !strings.Contains(last.Line, " end ") || last.Err != nil || last.Qual != Late || !last.At.Equal(at(120.3)) {
		t.Errorf("terminator write %+v, want nil/late at 120.3", last)
	}
	if st := s.Diag.Terminal; st.State != DiagCompleteWithTerminator || st.Qual != Late {
		t.Errorf("diagnostic terminal %v/%v, want complete-with-terminator/late", st.State, st.Qual)
	}
}

// §6.6 case 4, the diagnostic task first (120.3); five-state row 2: the
// log has the terminator line, yet the conclusion is 2.
func TestCase4bTerminatorBeforeDDiagFirst(t *testing.T) {
	h, term, done, open := terminatorBytesBeforeD(t, true)
	h.waitArrived(pointFinalWake)
	h.clk.Set(at(120.3))
	term.free()
	h.waitFor("diagnostic terminal", diagTerminal)
	open()
	f := h.final(done)
	wantFinal(t, f, Incomplete, PathConsumed)
	if !hasReason(f.Reasons, "diagnostic terminal state: late") {
		t.Errorf("reasons %q", f.Reasons)
	}
	if !strings.Contains(h.stderr.text(), " end manifest-sha256 ") {
		t.Error("terminator line missing from the output")
	}
}

// §6.6 case 5; five-state row 2: provisional 3; a record hash line's write
// returns nil at 120.2; in the same lock the result is published (late) and
// the re-check stops the task before the next write: no terminator.
func TestCase5HashLineReturnsAfterD(t *testing.T) {
	h := newHarness(t)
	hash := h.stderr.on(t, 0, "record controls.jsonl", blocking())
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	open := h.gate(pointFinalWake)
	done := h.run(h.cfg(Inconsistent, nil, nil))
	waitCh(t, hash.entered, "hash line write")
	h.clk.Set(at(120.2))
	hash.free()
	s := h.waitFor("diagnostic terminal", diagTerminal)
	if st := s.Diag.Terminal; st.State != DiagExpiredStopped || st.Qual != Late {
		t.Errorf("diagnostic terminal %v/%v, want expired-stopped/late", st.State, st.Qual)
	}
	last := s.Diag.Writes[len(s.Diag.Writes)-1]
	if !strings.Contains(last.Line, "record controls.jsonl") || last.Err != nil || last.Qual != Late {
		t.Errorf("last write %+v, want the hash line, nil/late", last)
	}
	out := h.stderr.text()
	if strings.Contains(out, "record derived.jsonl") || strings.Contains(out, " end ") {
		t.Errorf("writes after the expiry exit: %q", out)
	}
	if !strings.HasPrefix(out, ProvisionalLine(Inconsistent)) {
		t.Errorf("output %q does not start with the provisional line", out)
	}
	open()
	wantFinal(t, h.final(done), Incomplete, PathConsumed)
}

// §6.6 case 6 (first part); five-state row 2: the writer's write of the
// derived records fails at 60. It stops writing, closes every open file
// once, creates no manifest and publishes "failure", all timely. The
// snapshot at 100 gets provisional 2 from the published failure, and the
// final conclusion stays 2.
func TestCase6WriterWriteFails(t *testing.T) {
	h := newHarness(t)
	diskFull := errors.New("disk full")
	h.fs.set(t, "write", "derived.jsonl", 2, failing(diskFull, 3))
	h.startWriter()
	h.waitFor("headers", func(s OuterSnapshot) bool {
		for _, f := range s.Writer.Files {
			if !f.HeaderWritten {
				return false
			}
		}
		return true
	})
	h.clk.Set(at(60))
	h.log.Append("derived.jsonl", []byte(`{"derived":1}`))
	s := h.waitFor("writer terminal", writerTerminal)
	w := s.Writer
	if w.Terminal.State != WriterFailure || w.Terminal.Qual != Timely || !w.Terminal.At.Equal(at(60)) {
		t.Errorf("writer terminal %+v, want failure/timely at 60", w.Terminal)
	}
	if len(w.Errors) != 1 || w.Errors[0].Site != "write derived.jsonl" || !errors.Is(w.Errors[0].Err, diskFull) {
		t.Errorf("errors %v", w.Errors)
	}
	header := HeaderLine("derived.jsonl")
	if d := w.Files[1]; d.Bytes != int64(len(header)+3) || d.SHA256 != sha(append(header, `{"d`...)) {
		t.Errorf("derived bytes %d sha %s; the hash must cover the bytes written", d.Bytes, d.SHA256)
	}
	for _, name := range recordFiles {
		if n := h.fs.count("close", name); n != 1 {
			t.Errorf("%s closed %d times, want 1", name, n)
		}
	}
	if n := h.fs.count("create", ManifestName); n != 0 {
		t.Error("manifest created after a write failure")
	}
	rawWrites := h.fs.count("write", "raw.jsonl")
	h.log.Append("raw.jsonl", []byte(`{"after":1}`)) // stays in memory

	h.clk.Set(at(100))
	done := h.run(h.cfg(Consistent, []string{"R ok"}, nil))
	f := h.final(done)
	wantFinal(t, f, Incomplete, PathConsumed)
	if f.CallerProvisional != Consistent || f.Provisional != Incomplete || !hasReason(f.Caps, "write derived.jsonl: disk full") {
		t.Errorf("caller %v provisional %v caps %q", f.CallerProvisional, f.Provisional, f.Caps)
	}
	in := h.provInputs()
	if len(in) != 1 || in[0].Outer.Writer.Terminal == nil || in[0].Outer.Writer.Terminal.State != WriterFailure {
		t.Error("the provisional function did not receive the published writer failure")
	}
	if got := h.fs.count("write", "raw.jsonl"); got != rawWrites {
		t.Errorf("raw.jsonl written after the failure: %d writes, want %d", got, rawWrites)
	}
	s = h.o.Snapshot()
	if st := s.Diag.Terminal; st.State != DiagCompleteWithoutTerminator || st.Qual != Timely {
		t.Errorf("diagnostic terminal %v/%v, want complete-without-terminator/timely", st.State, st.Qual)
	}
	failures := []string{"R ok", "closeout: record chain: write derived.jsonl: disk full", "closeout: record chain terminal state failure"}
	want := snapshotLines(Incomplete, failures, nil) +
		CountLine("record-chain-error", 1) + TextLine("record-chain-error", "write derived.jsonl: disk full")
	if got := h.stderr.text(); got != want {
		t.Errorf("stderr =\n%s\nwant\n%s", got, want)
	}
}

// §6.6 case 6 (second part): a diagnostic write fails → "write-failed";
// provisional 3 or 4 becomes 2, provisional 1 or 2 is unchanged.
func TestCase6DiagnosticWriteFails(t *testing.T) {
	for _, tc := range []struct {
		prov, want Conclusion
		nth        int
	}{
		{Consistent, Incomplete, 1},
		{Inconsistent, Incomplete, 5},
		{Incomplete, Incomplete, 1},
		{Invalid, Invalid, 3},
	} {
		t.Run(tc.prov.Name(), func(t *testing.T) {
			h := newHarness(t)
			h.stderr.on(t, tc.nth, "", failing(errors.New("EPIPE"), 0))
			h.startWriter()
			h.appendItems()
			h.clk.Set(at(100))
			f := h.final(h.run(h.cfg(tc.prov, nil, nil)))
			wantFinal(t, f, tc.want, PathConsumed)
			s := h.o.Snapshot()
			if st := s.Diag.Terminal; st.State != DiagWriteFailed || st.Qual != Timely {
				t.Errorf("diagnostic terminal %v/%v, want write-failed/timely", st.State, st.Qual)
			}
			if n := len(s.Diag.Writes); n != tc.nth || s.Diag.Writes[n-1].Err == nil {
				t.Errorf("diagnostic writes %d, want %d ending with the error", n, tc.nth)
			}
		})
	}
}

// §6.6 case 7; five-state row 5: provisional 1 or 2 with every result
// timely: the final conclusion is the provisional one, exit nonzero, and
// the terminator line carries it.
func TestCase7ProvisionalOneOrTwoNeverImproves(t *testing.T) {
	for _, prov := range []Conclusion{Invalid, Incomplete} {
		t.Run(prov.Name(), func(t *testing.T) {
			h := newHarness(t)
			h.startWriter()
			h.appendItems()
			h.clk.Set(at(100))
			f := h.final(h.run(h.cfg(prov, nil, []string{"F LC-9 not executed"})))
			wantFinal(t, f, prov, PathConsumed)
			s := h.o.Snapshot()
			allQualified(t, s, Timely)
			if st := s.Diag.Terminal; st.State != DiagCompleteWithTerminator {
				t.Errorf("diagnostic terminal %v", st.State)
			}
			content, mf := h.checkCompleteRecords()
			want := snapshotLines(prov, nil, []string{"F LC-9 not executed"}) + successLines(content, mf, prov)
			if got := h.stderr.text(); got != want {
				t.Errorf("stderr =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// Five-state row 4 (and rows 5's waiting): on a wake before D without a
// diagnostic terminal state the parent keeps waiting (branch c) and does
// not exit early, for a provisional 4 and for a provisional 2; a later
// success does not improve the 2.
func TestFinalConsumptionKeepsWaitingBeforeD(t *testing.T) {
	for _, prov := range []Conclusion{Consistent, Incomplete} {
		t.Run(prov.Name(), func(t *testing.T) {
			h := newHarness(t)
			first := h.stderr.on(t, 1, "", blocking())
			h.startWriter()
			h.appendItems()
			h.clk.Set(at(100))
			done := h.run(h.cfg(prov, nil, nil))
			waitCh(t, first.entered, "first diagnostic write")
			s := h.waitFor("writer terminal", writerTerminal)
			// After the writer's last publication the parent evaluates at
			// least once more; nothing else is published until the
			// diagnostic write returns.
			h.waitEvent("branch c after the writer finished", func(ev traceEvent) bool {
				return ev.step == pointFinalWake && ev.branch == "c" && ev.gen >= s.Gen
			})
			for _, ev := range h.trace() {
				if ev.step == pointFinalWake && (ev.branch != "c" || !ev.now.Before(h.d)) {
					t.Fatalf("early exit branch %+v", ev)
				}
			}
			select {
			case f := <-done:
				t.Fatalf("Closeout returned before the diagnostic terminal state: %+v", f)
			default:
			}
			first.free()
			wantFinal(t, h.final(done), prov, PathConsumed)
			if br := h.finalBranches(); br[len(br)-1] != "a" {
				t.Errorf("branches %v, want last a", br)
			}
		})
	}
}

// Five-state row 5 with expiry: provisional 1, the diagnostic output blocks
// past D; the parent exits at D with 1 (unchanged), nonzero.
func TestProvisionalOneExpiresUnchanged(t *testing.T) {
	h := newHarness(t)
	first := h.stderr.on(t, 1, "", blocking())
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	done := h.run(h.cfg(Invalid, nil, nil))
	waitCh(t, first.entered, "first diagnostic write")
	h.clk.Set(at(120))
	wantFinal(t, h.final(done), Invalid, PathExpiredFinal)
}

// A diagnostic output that blocks past D: Closeout still returns at D,
// nonzero, without waiting for it; the blocked task later publishes a late
// result and stops.
func TestDiagnosticBlocksPastD(t *testing.T) {
	h := newHarness(t)
	first := h.stderr.on(t, 1, "", blocking())
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	done := h.run(h.cfg(Consistent, nil, nil))
	waitCh(t, first.entered, "first diagnostic write")
	h.waitFor("writer terminal", writerTerminal)
	h.clk.Set(at(120))
	f := h.final(done)
	wantFinal(t, f, Incomplete, PathExpiredFinal)
	if !f.DecidedAt.Equal(at(120)) {
		t.Errorf("decided at %v, want D", f.DecidedAt)
	}
	first.free()
	s := h.waitFor("diagnostic terminal", diagTerminal)
	if st := s.Diag.Terminal; st.State != DiagExpiredStopped || st.Qual != Late {
		t.Errorf("diagnostic terminal %v/%v", st.State, st.Qual)
	}
	if len(s.Diag.Writes) != 1 || s.Diag.Writes[0].Qual != Late {
		t.Errorf("diagnostic writes %+v", s.Diag.Writes)
	}
}
