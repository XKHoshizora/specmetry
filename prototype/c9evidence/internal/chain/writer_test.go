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
// no further close, no manifest. Closeout has already returned at D.
func TestWriterExpiresMidway(t *testing.T) {
	h := newHarness(t)
	dc := h.fs.set(t, "close", "derived.jsonl", 1, blocking())
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	done := h.run(h.cfg(Consistent, nil, nil))
	waitCh(t, dc.entered, "derived close")
	h.clk.Set(at(121))
	wantFinal(t, h.final(done), Incomplete, PathExpiredFinal)
	dc.free()
	s := h.waitFor("diagnostic terminal", diagTerminal)
	if st := s.Writer.Terminal; st.State != WriterExpired || st.Qual != Late {
		t.Errorf("writer terminal %v/%v, want expired-incomplete/late", st.State, st.Qual)
	}
	if oc := s.Writer.Files[1].Close; oc == nil || oc.Err != nil || oc.Qual != Late {
		t.Errorf("derived close %+v, want nil/late", oc)
	}
	for _, name := range []string{"controls.jsonl", "closeout.jsonl"} {
		if n := h.fs.count("close", name); n != 0 {
			t.Errorf("%s closed after D", name)
		}
	}
	if h.fs.count("create", ManifestName) != 0 {
		t.Error("manifest created after D")
	}
	if st := s.Diag.Terminal; st.State != DiagExpiredStopped {
		t.Errorf("diagnostic terminal %v", st.State)
	}
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
