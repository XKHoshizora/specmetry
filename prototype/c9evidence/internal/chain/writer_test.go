package chain

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// runFailingChain runs a full closeout with provisional 4 and returns the
// final conclusion and the registry once the diagnostic task is terminal.
func runFailingChain(t *testing.T, h *harness) (Final, OuterSnapshot) {
	t.Helper()
	h.startWriter()
	h.addItems() // a failure may keep them from being written
	h.clk.Set(at(100))
	f := h.final(h.run(h.cfg(Consistent, nil, nil)))
	return f, h.o.Snapshot()
}

// failedChainChecks asserts the common shape of a record-chain failure:
// writer "failure" (timely), every error kept, no terminator, final 2.
func failedChainChecks(t *testing.T, h *harness, f Final, s OuterSnapshot, site string) {
	t.Helper()
	wantFinal(t, f, Incomplete, PathConsumed)
	if st := s.Writer.Terminal; st == nil || st.State != WriterFailure || st.Qual != Timely {
		t.Fatalf("writer terminal %+v, want failure/timely", st)
	}
	found := false
	for _, e := range s.Writer.Terminal.Errors {
		found = found || e.Site == site
	}
	if !found {
		t.Errorf("writer errors %v lack %q", s.Writer.Terminal.Errors, site)
	}
	if st := s.Diag.Terminal; st.State != DiagCompleteWithoutTerminator || st.Qual != Timely {
		t.Errorf("diagnostic terminal %v/%v, want complete-without-terminator/timely", st.State, st.Qual)
	}
	out := h.stderr.text()
	if strings.Contains(out, " end ") || !strings.Contains(out, LinePrefix+"record-chain-error-count ") {
		t.Errorf("stderr %q: want error lines and no terminator", out)
	}
}

func TestWriterMkdirFails(t *testing.T) {
	h := newHarness(t)
	h.fs.set(t, "mkdir", recDir, 1, failing(errors.New("permission denied"), 0))
	f, s := runFailingChain(t, h)
	failedChainChecks(t, h, f, s, "mkdir "+recDir)
	if n := len(h.fs.callLog()); n != 1 {
		t.Errorf("FS calls %v, want only the mkdir", h.fs.callLog())
	}
}

// The k-th create fails: the first k-1 files are closed once each, nothing
// is written, no manifest.
func TestWriterKthCreateFails(t *testing.T) {
	for k := 1; k <= len(recordFiles); k++ {
		t.Run(fmt.Sprint(k), func(t *testing.T) {
			h := newHarness(t)
			h.fs.set(t, "create", recordFiles[k-1], 1, failing(errors.New("exists"), 0))
			h.startWriter()
			s := h.waitFor("writer terminal", writerTerminal)
			if st := s.Writer.Terminal; st.State != WriterFailure || st.Qual != Timely {
				t.Fatalf("writer terminal %v/%v", st.State, st.Qual)
			}
			for i, name := range recordFiles {
				wantCreate, wantClose := 1, 0
				if i < k-1 {
					wantClose = 1
				}
				if i > k-1 {
					wantCreate = 0
				}
				if n := h.fs.count("create", name); n != wantCreate {
					t.Errorf("%s created %d times, want %d", name, n, wantCreate)
				}
				if n := h.fs.count("close", name); n != wantClose {
					t.Errorf("%s closed %d times, want %d", name, n, wantClose)
				}
				if n := h.fs.count("write", name); n != 0 {
					t.Errorf("%s written %d times, want 0", name, n)
				}
			}
			if h.fs.count("create", ManifestName) != 0 {
				t.Error("manifest created")
			}
			if len(s.Writer.Errors) != 1 || s.Writer.Errors[0].Site != "create "+recordFiles[k-1] {
				t.Errorf("errors %v", s.Writer.Errors)
			}
		})
	}
}

// A write fails (a short write with an error): the hash covers exactly the
// bytes written, writing stops for every file, every open file is closed
// once, no manifest.
func TestWriterPartialWriteFails(t *testing.T) {
	h := newHarness(t)
	h.fs.set(t, "write", "raw.jsonl", 1, failing(errors.New("EIO"), 5))
	h.startWriter()
	s := h.waitFor("writer terminal", writerTerminal)
	raw := s.Writer.Files[0]
	header := HeaderLine("raw.jsonl")
	if raw.Bytes != 5 || raw.SHA256 != sha(header[:5]) || raw.HeaderWritten {
		t.Errorf("raw.jsonl bytes %d sha %s header %v", raw.Bytes, raw.SHA256, raw.HeaderWritten)
	}
	if w := raw.Writes[0]; w.Requested != len(header) || w.N != 5 || w.Err == nil || !w.Header {
		t.Errorf("write outcome %+v", w)
	}
	for _, name := range recordFiles {
		if n := h.fs.count("close", name); n != 1 {
			t.Errorf("%s closed %d times", name, n)
		}
		if name != "raw.jsonl" && h.fs.count("write", name) != 0 {
			t.Errorf("%s written after the failure", name)
		}
	}
	if s.Writer.Terminal.State != WriterFailure || h.fs.count("create", ManifestName) != 0 {
		t.Errorf("terminal %v; manifest creates %d", s.Writer.Terminal.State, h.fs.count("create", ManifestName))
	}
}

// A record file's Close fails: the release is unknown, it is recorded and
// not retried, the other files are still closed once, no manifest.
func TestWriterRecordCloseFails(t *testing.T) {
	h := newHarness(t)
	h.fs.set(t, "close", "derived.jsonl", 1, failing(errors.New("EIO on close"), 0))
	f, s := runFailingChain(t, h)
	failedChainChecks(t, h, f, s, "close derived.jsonl")
	for _, name := range recordFiles {
		if n := h.fs.count("close", name); n != 1 {
			t.Errorf("%s closed %d times, want 1", name, n)
		}
	}
	if h.fs.count("create", ManifestName) != 0 {
		t.Error("manifest created after a close error")
	}
	if oc := s.Writer.Files[1].Close; oc == nil || oc.Err == nil {
		t.Errorf("derived close outcome %+v", oc)
	}
	if oc := s.Writer.Files[2].Close; oc == nil || oc.Err != nil {
		t.Errorf("controls close outcome %+v (closed after the failure)", oc)
	}
}

func TestWriterManifestFailures(t *testing.T) {
	for _, tc := range []struct {
		kind   string
		closes int
	}{
		{"create", 0},
		{"write", 1}, // the open manifest is closed once
		{"close", 1}, // not retried
	} {
		t.Run(tc.kind, func(t *testing.T) {
			h := newHarness(t)
			h.fs.set(t, tc.kind, ManifestName, 1, failing(errors.New("manifest "+tc.kind+" failed"), 0))
			f, s := runFailingChain(t, h)
			failedChainChecks(t, h, f, s, tc.kind+" "+ManifestName)
			if n := h.fs.count("close", ManifestName); n != tc.closes {
				t.Errorf("manifest closed %d times, want %d", n, tc.closes)
			}
			for _, name := range recordFiles {
				if n := h.fs.count("close", name); n != 1 {
					t.Errorf("%s closed %d times", name, n)
				}
			}
		})
	}
}

// The writer is in a record file's Close when D passes; the close returns
// nil afterwards (late), and the next decision point takes the expiry exit:
// no further close, no manifest. Closeout has already returned at D, while
// the Close is still blocked: the parent never waits for it.
//
// Past D the parent's D timer and the diagnostic task's re-check race
// (review-resolution §193): the parent either finds no diagnostic terminal
// state (step 5 branch b, expired-final) or consumes the diagnostic task's
// expired-stopped state (branch a, consumed). Both are legal by the
// five-state table and both give 2; no other exit is accepted. The
// background results are checked only after both the writer and the
// diagnostic task have published their terminal states.
func TestWriterExpiresMidway(t *testing.T) {
	h := newHarness(t)
	dc := h.fs.set(t, "close", "derived.jsonl", 1, blocking())
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	done := h.run(h.cfg(Consistent, nil, nil))
	waitCh(t, dc.entered, "derived close")
	h.clk.Set(at(121))
	f := h.final(done) // the derived Close is still blocked
	if f.Conclusion != Incomplete || f.ExitCode != ExitCode(Incomplete) {
		t.Fatalf("final %v exit %d via %s (reasons %q), want 2", f.Conclusion, f.ExitCode, f.Path, f.Reasons)
	}
	if f.DecidedAt.Before(h.d) {
		t.Errorf("decided at %v, before D", f.DecidedAt)
	}
	last := ""
	if bs := h.finalBranches(); len(bs) > 0 {
		last = bs[len(bs)-1]
	}
	t.Logf("exit %s, last final branch %q", f.Path, last) // which legal exit this run took
	switch f.Path {
	case PathConsumed: // (a): the published expired-stopped state was consumed
		if last != "a" || !hasReason(f.Reasons, "diagnostic terminal state: "+DiagExpiredStopped.String()) {
			t.Errorf("consumed exit: last branch %q, reasons %q", last, f.Reasons)
		}
	case PathExpiredFinal: // (b): no diagnostic terminal state, D passed
		if last != "b" || !hasReason(f.Reasons, "the diagnostic task had no terminal state at D") {
			t.Errorf("expired-final exit: last branch %q, reasons %q", last, f.Reasons)
		}
	default:
		t.Fatalf("exit %s, want consumed (a) or expired-final (b)", f.Path)
	}
	dc.free()
	s := h.waitFor("writer and diagnostic terminal states", func(s OuterSnapshot) bool { return writerTerminal(s) && diagTerminal(s) })
	if st := s.Writer.Terminal; st.State != WriterExpired || st.Qual != Late {
		t.Errorf("writer terminal %v/%v, want expired-incomplete/late", st.State, st.Qual)
	}
	// raw.jsonl was closed before D; the admitted derived Close returned nil
	// after D and its real, late result is kept; both are released.
	raw, derived := fileNamed(s, "raw.jsonl"), fileNamed(s, "derived.jsonl")
	if oc := raw.Close; oc == nil || oc.Err != nil || oc.Qual != Timely {
		t.Errorf("raw close %+v, want nil/timely", oc)
	}
	if oc := derived.Close; oc == nil || oc.Err != nil || oc.Qual != Late {
		t.Errorf("derived close %+v, want nil/late", oc)
	}
	for _, x := range []FileSnapshot{raw, derived} {
		if hd := x.Holding; hd.Held() || !hd.CloseStarted || !hd.CloseReturned || hd.CloseErr != nil || !hd.Released {
			t.Errorf("%s holding %+v, want released after its one nil Close", x.Name, hd)
		}
		if n := h.fs.count("close", x.Name); n != 1 {
			t.Errorf("%s closed %d times, want 1", x.Name, n)
		}
	}
	// After D no further Close starts: the other record files stay held by
	// the writer, with no Close started.
	for _, name := range []string{"controls.jsonl", "closeout.jsonl"} {
		if n := h.fs.count("close", name); n != 0 {
			t.Errorf("%s closed after D", name)
		}
		fs := fileNamed(s, name)
		if hd := fs.Holding; hd.Holder != HolderWriter || hd.CloseStarted || hd.CloseReturned || hd.Released || fs.Close != nil {
			t.Errorf("%s holding %+v close %+v, want held by the writer, no Close started", name, hd, fs.Close)
		}
	}
	if h.fs.count("create", ManifestName) != 0 {
		t.Error("manifest created after D")
	}
	if m := s.Writer.Manifest; m.Create != nil || m.Holding.Held() {
		t.Errorf("manifest create %+v holding %+v, want none", m.Create, m.Holding)
	}
	if got := h.wr.Held(); len(got) != 2 || got[0] != "controls.jsonl" || got[1] != "closeout.jsonl" {
		t.Errorf("Held() %v, want the two record files whose Close never started", got)
	}
	if st := s.Diag.Terminal; st.State != DiagExpiredStopped || st.Qual != Late {
		t.Errorf("diagnostic terminal %v/%v, want expired-stopped/late", st.State, st.Qual)
	}
}

// fileNamed returns the writer's published state of one record file.
func fileNamed(s OuterSnapshot, name string) FileSnapshot {
	for _, f := range s.Writer.Files {
		if f.Name == name {
			return f
		}
	}
	return FileSnapshot{Name: name}
}

// The seal: a producer keeps appending during and after the closeout; the
// writer still terminates, and the record files contain exactly the items
// up to the seal. The later items stay in memory only. The writer's and
// diagnostic task's own publications are not log items.
func TestSealNoEndlessAppend(t *testing.T) {
	h := newHarness(t)
	h.startWriter()
	h.appendItems()
	stop := make(chan struct{})
	producerDone := make(chan int)
	go func() {
		after := 0
		for i := 0; ; i++ {
			if sealed, _ := h.log.Sealed(); sealed {
				after++
				if after > 1000 {
					break
				}
			}
			select {
			case <-stop:
				producerDone <- after
				return
			default:
			}
			h.log.Append("raw.jsonl", []byte(fmt.Sprintf(`{"producer":%d}`, i)))
			runtime.Gosched()
		}
		producerDone <- after
	}()
	h.clk.Set(at(100))
	f := h.final(h.run(h.cfg(Consistent, nil, nil)))
	wantFinal(t, f, Consistent, PathConsumed)
	after := <-producerDone
	close(stop)
	if after <= 1000 {
		t.Fatalf("producer appended %d items after the seal, want > 1000", after)
	}
	h.checkCompleteRecords()
	_, seal := h.log.Sealed()
	items := h.log.Items()
	if len(items) <= seal {
		t.Fatalf("no items after the seal (%d items, seal %d)", len(items), seal)
	}
	// Inside the seal: the producer items, the caller's closeout record and
	// the parent's own provisional item — nothing from the writer or the
	// diagnostic task.
	last := items[seal-1]
	if last.File != "closeout.jsonl" || !strings.HasPrefix(string(last.Line), `{"closeout":"provisional","conclusion":4,`) {
		t.Errorf("last sealed item %s %s", last.File, last.Line)
	}
	if prev := items[seal-2]; string(prev.Line) != `{"summary":"test"}` {
		t.Errorf("caller closeout record %s", prev.Line)
	}
	for _, it := range items {
		if it.File != "raw.jsonl" && it.File != "closeout.jsonl" && it.Seq > 4 {
			t.Errorf("unexpected item %d in %s: %s", it.Seq, it.File, it.Line)
		}
	}
}

// A qualification decided at exactly D is late: the terminator's write
// returns at exactly 120.
func TestQualificationAtExactlyDIsLate(t *testing.T) {
	h := newHarness(t)
	term := h.stderr.on(t, 0, " end ", blocking())
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	open := h.gate(pointFinalWake)
	done := h.run(h.cfg(Consistent, nil, nil))
	waitCh(t, term.entered, "terminator write")
	h.clk.Set(h.d)
	term.free()
	s := h.waitFor("diagnostic terminal", diagTerminal)
	last := s.Diag.Writes[len(s.Diag.Writes)-1]
	if last.Qual != Late || !last.At.Equal(h.d) || s.Diag.Terminal.Qual != Late {
		t.Errorf("terminator write %+v terminal %+v, want late at D", last, s.Diag.Terminal)
	}
	if !s.Latch || !s.LatchAt.Equal(h.d) {
		t.Errorf("latch %v at %v, want set at D", s.Latch, s.LatchAt)
	}
	open()
	wantFinal(t, h.final(done), Incomplete, PathConsumed)
}

// backClock reads whatever the test sets, even an earlier instant, so the
// latch can be tested independently of the reading. Its timers never fire.
type backClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *backClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *backClock) set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now
}

func (c *backClock) Timer(time.Time) (<-chan struct{}, func()) { return make(chan struct{}), func() {} }

func TestLatchNeverResetsAndLateNeverRestores(t *testing.T) {
	clk := NewFakeClock(t0)
	_, d := Deadlines(t0)
	o := NewOuter(clk, d)

	o.mu.Lock()
	if _, q := o.checkLocked(); q != Timely || o.latch {
		t.Fatal("timely before D")
	}
	clk.Set(d.Add(-1))
	if now, q := o.checkLocked(); q != Timely || !now.Equal(d.Add(-1)) {
		t.Fatal("one nanosecond before D is timely")
	}
	clk.Set(d)
	if _, q := o.checkLocked(); q != Late || !o.latch || !o.latchAt.Equal(d) {
		t.Fatal("a reading equal to D must be late and set the latch")
	}
	o.mu.Unlock()

	// A clock that reads before D again (not monotonic): the latch still
	// makes every later publication late and is never cleared.
	back := &backClock{now: d.Add(-60 * time.Second)}
	o2 := NewOuter(back, d)
	op1 := o2.RegisterOp("first")
	op2 := o2.RegisterOp("second")
	op1.Done(errors.New("cleanup failed"))
	back.set(d)
	o2.mu.Lock()
	o2.checkLocked() // a reading at D sets the latch
	o2.mu.Unlock()
	back.set(d.Add(-30 * time.Second))
	op2.Done(nil) // a late success
	op1.Done(nil) // a second result is a defect; the failure stays
	s := o2.Snapshot()
	if !s.Latch {
		t.Fatal("latch cleared")
	}
	if s.Ops[0].State != OpFailed || s.Ops[0].Outcome.Qual != Timely {
		t.Errorf("first op %+v: the failure and its qualification must stay", s.Ops[0])
	}
	if s.Ops[1].Outcome.Qual != Late {
		t.Errorf("second op %+v: published after the latch, must be late", s.Ops[1].Outcome)
	}
	if len(s.Defects) != 1 {
		t.Errorf("defects %v", s.Defects)
	}
}
