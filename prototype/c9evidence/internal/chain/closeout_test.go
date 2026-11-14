package chain

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// m.Run returns at or after D: step 1 takes the expiry exit. No copies, no
// provisional conclusion, no seal, no closeout signal, no diagnostic task;
// 1 if a level-1 condition holds, else 2.
func TestMRunReturnsAfterD(t *testing.T) {
	for _, tc := range []struct {
		name   string
		when   float64
		code   int
		level1 bool
		want   Conclusion
	}{
		{"at-D-incomplete", 120, 0, false, Incomplete},
		{"after-D-incomplete", 121, 0, false, Incomplete},
		{"level1", 121, 0, true, Invalid},
		{"m.Run-nonzero", 121, 1, false, Invalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			h.startWriter()
			h.appendItems()
			h.clk.Set(at(tc.when))
			cfg := h.cfg(Consistent, nil, nil)
			cfg.MRunCode = tc.code
			cfg.Level1 = func(OuterSnapshot) bool { return tc.level1 }
			f := h.final(h.run(cfg))
			wantFinal(t, f, tc.want, PathExpiredAtMRun)
			if f.Provisional != 0 || h.copyCount() != 0 || len(h.provInputs()) != 0 {
				t.Errorf("provisional %v, copies %d, provisional calls %d", f.Provisional, h.copyCount(), len(h.provInputs()))
			}
			s := h.o.Snapshot()
			if s.Diag != nil || s.Closeout.Signaled || !s.Latch {
				t.Errorf("diag %v signaled %v latch %v", s.Diag, s.Closeout.Signaled, s.Latch)
			}
			if sealed, _ := h.log.Sealed(); sealed {
				t.Error("log sealed")
			}
			if h.stderr.text() != "" {
				t.Errorf("stderr %q", h.stderr.text())
			}
		})
	}
}

// The snapshot wait reaches W with a record still pending: the parent
// proceeds at W (before D), and the provisional 4 is lowered to 2.
func TestSnapshotWaitReachesW(t *testing.T) {
	h := newHarness(t)
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	cfg := h.cfg(Consistent, nil, nil)
	cfg.Terminal = []TerminalSource{func() (bool, <-chan struct{}) { return false, make(chan struct{}) }}
	done := h.run(cfg)
	h.waitEvent("step 1 pass", func(ev traceEvent) bool { return ev.step == pointMRunReturn })
	h.clk.Set(at(110))
	f := h.final(done)
	wantFinal(t, f, Incomplete, PathConsumed)
	if f.CallerProvisional != Consistent || f.Provisional != Incomplete || !hasReason(f.Caps, "not every non-closeout record") {
		t.Errorf("caller %v provisional %v caps %q", f.CallerProvisional, f.Provisional, f.Caps)
	}
	in := h.provInputs()
	if len(in) != 1 || in[0].AllTerminal {
		t.Errorf("provisional input %+v", in)
	}
	ev := h.waitEvent("snapshot", func(ev traceEvent) bool { return ev.step == pointSnapshotWaitDone })
	if ev.branch != "pass" || !ev.now.Equal(at(110)) {
		t.Errorf("snapshot event %+v", ev)
	}
	if !strings.Contains(h.stderr.text(), TextLine("failure", "closeout: not every non-closeout record was terminal at the snapshot point")) {
		t.Errorf("stderr lacks the cap: %q", h.stderr.text())
	}
}

// m.Run returns after W: the parent does not wait for pending records.
func TestMRunAfterWDoesNotWait(t *testing.T) {
	h := newHarness(t)
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(112))
	cfg := h.cfg(Consistent, nil, nil)
	cfg.Terminal = []TerminalSource{func() (bool, <-chan struct{}) { return false, make(chan struct{}) }}
	f := h.final(h.run(cfg)) // no clock movement is needed
	wantFinal(t, f, Incomplete, PathConsumed)
}

// The snapshot point waits for every pending outer operation and for the
// caller's sources; once they are terminal it proceeds without W.
func TestSnapshotWaitsForTerminalRecords(t *testing.T) {
	h := newHarness(t)
	op := h.o.RegisterOp("fixture cleanup")
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	src := make(chan struct{})
	var mu sync.Mutex
	srcDone := false
	cfg := h.cfg(Consistent, nil, nil)
	cfg.Terminal = []TerminalSource{func() (bool, <-chan struct{}) {
		mu.Lock()
		defer mu.Unlock()
		return srcDone, src
	}}
	done := h.run(cfg)
	h.waitEvent("step 1 pass", func(ev traceEvent) bool { return ev.step == pointMRunReturn })
	op.Done(nil)
	mu.Lock()
	srcDone = true
	mu.Unlock()
	close(src)
	f := h.final(done)
	wantFinal(t, f, Consistent, PathConsumed)
	in := h.provInputs()
	if len(in) != 1 || !in[0].AllTerminal || in[0].Outer.Ops[0].State != OpDone {
		t.Errorf("provisional input %+v", in)
	}
	if ev := h.waitEvent("snapshot", func(ev traceEvent) bool { return ev.step == pointSnapshotWaitDone }); !ev.now.Equal(at(100)) {
		t.Errorf("snapshot at %v, want 100 (no wait for W)", ev.now)
	}
}

// Outer operations that failed, are unknown or got a second result lower
// the provisional conclusion.
func TestOuterOperationsCap(t *testing.T) {
	for _, tc := range []struct {
		name string
		fin  func(*Op)
		want Conclusion
	}{
		{"failed", func(op *Op) { op.Done(errors.New("rm failed")) }, Incomplete},
		{"unknown", func(op *Op) { op.Unknown("darwin cleanup at expiry") }, Incomplete},
		{"second-result", func(op *Op) { op.Done(nil); op.Done(nil) }, Invalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			tc.fin(h.o.RegisterOp("fixture"))
			h.startWriter()
			h.appendItems()
			h.clk.Set(at(100))
			wantFinal(t, h.final(h.run(h.cfg(Consistent, nil, nil))), tc.want, PathConsumed)
		})
	}
}

// Rejected log items (tool defects) and invalid caller results are level 1.
func TestToolDefectsAreLevelOne(t *testing.T) {
	t.Run("log-defect", func(t *testing.T) {
		h := newHarness(t)
		if seq := h.log.Append("unknown.jsonl", []byte("{}")); seq != 0 {
			t.Fatalf("append to an unknown file returned %d", seq)
		}
		h.startWriter()
		h.appendItems()
		h.clk.Set(at(100))
		wantFinal(t, h.final(h.run(h.cfg(Consistent, nil, nil))), Invalid, PathConsumed)
	})
	t.Run("bad-closeout-record", func(t *testing.T) {
		h := newHarness(t)
		h.startWriter()
		h.appendItems()
		h.clk.Set(at(100))
		cfg := h.cfg(Consistent, nil, nil)
		prov := cfg.Provisional
		cfg.Provisional = func(in ProvisionalInput[int]) Snapshot {
			s := prov(in)
			s.Records = append(s.Records, Record{File: "closeout.jsonl", Line: []byte("two\nlines")})
			return s
		}
		f := h.final(h.run(cfg))
		wantFinal(t, f, Invalid, PathConsumed)
		if !hasReason(f.Caps, "closeout record rejected") {
			t.Errorf("caps %q", f.Caps)
		}
		h.checkCompleteRecords() // the rejected record is not written
	})
	t.Run("caller-out-of-range", func(t *testing.T) {
		h := newHarness(t)
		h.startWriter()
		h.appendItems()
		h.clk.Set(at(100))
		wantFinal(t, h.final(h.run(h.cfg(Conclusion(7), nil, nil))), Invalid, PathConsumed)
	})
	t.Run("m.Run-nonzero", func(t *testing.T) {
		h := newHarness(t)
		h.startWriter()
		h.appendItems()
		h.clk.Set(at(100))
		cfg := h.cfg(Consistent, nil, nil)
		cfg.MRunCode = 1
		wantFinal(t, h.final(h.run(cfg)), Invalid, PathConsumed)
	})
}

func TestCloseoutConfigDefects(t *testing.T) {
	t.Run("no-writer", func(t *testing.T) {
		h := newHarness(t)
		wantFinal(t, Closeout(h.cfg(Consistent, nil, nil)), Invalid, PathConfig)
	})
	t.Run("writer-not-started", func(t *testing.T) {
		h := newHarness(t)
		if _, err := h.o.RegisterWriter(WriterConfig{FS: h.fs, Dir: recDir, Log: h.log}); err != nil {
			t.Fatal(err)
		}
		wantFinal(t, Closeout(h.cfg(Consistent, nil, nil)), Invalid, PathConfig)
	})
	t.Run("twice", func(t *testing.T) {
		h := newHarness(t)
		h.startWriter()
		h.appendItems()
		h.clk.Set(at(100))
		wantFinal(t, h.final(h.run(h.cfg(Consistent, nil, nil))), Consistent, PathConsumed)
		wantFinal(t, Closeout(h.cfg(Consistent, nil, nil)), Invalid, PathConfig)
	})
	t.Run("unknown-closeout-file", func(t *testing.T) {
		h := newHarness(t)
		h.startWriter()
		cfg := h.cfg(Consistent, nil, nil)
		cfg.CloseoutFile = ManifestName
		wantFinal(t, Closeout(cfg), Invalid, PathConfig)
	})
	t.Run("second-writer", func(t *testing.T) {
		h := newHarness(t)
		h.startWriter()
		if _, err := h.o.RegisterWriter(WriterConfig{FS: h.fs, Dir: "/other", Log: h.log}); err == nil {
			t.Fatal("a second writer was registered")
		}
		if err := h.wr.Start(); err == nil {
			t.Fatal("the writer started twice")
		}
	})
}

// The fixed formats: header, manifest, the parent's closeout item and the
// complete diagnostic output with its terminator.
func TestFormats(t *testing.T) {
	if got := string(HeaderLine("raw.jsonl")); got != "{\"c9evidence\":\"raw.jsonl\",\"version\":1}\n" {
		t.Errorf("header %q", got)
	}
	var v map[string]any
	if err := json.Unmarshal(HeaderLine("closeout.jsonl"), &v); err != nil || v["version"] != float64(1) {
		t.Errorf("header is not the expected JSON: %v %v", v, err)
	}
	if got := ManifestLine("raw.jsonl", 12, "ab"); got != "raw.jsonl 12 ab\n" {
		t.Errorf("manifest line %q", got)
	}
	if got := TerminatorLine("ab", Consistent); got != "c9evidence: end manifest-sha256 ab provisional 4 consistent-with-candidates\n" {
		t.Errorf("terminator %q", got)
	}
	if got := TextLine("failure", "a\nb \"c\""); got != "c9evidence: failure \"a\\nb \\\"c\\\"\"\n" {
		t.Errorf("text line %q", got)
	}

	h := newHarness(t)
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	wantFinal(t, h.final(h.run(h.cfg(Inconsistent, []string{"x"}, []string{"y"}))), Inconsistent, PathConsumed)
	content, mf := h.checkCompleteRecords()
	want := "c9evidence: provisional 3 inconsistent-with-candidates\n" +
		"c9evidence: failure-count 1\n" +
		"c9evidence: failure \"x\"\n" +
		"c9evidence: pending-count 1\n" +
		"c9evidence: pending \"y\"\n" +
		successLines(content, mf, Inconsistent)
	if got := h.stderr.text(); got != want {
		t.Errorf("stderr =\n%s\nwant\n%s", got, want)
	}
	lines := strings.Split(strings.TrimSuffix(string(content["closeout.jsonl"]), "\n"), "\n")
	var item closeoutLine
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &item); err != nil {
		t.Fatal(err)
	}
	if item.Closeout != "provisional" || item.Conclusion != 3 || item.Caller != 3 || !item.AllTerminal ||
		len(item.Caps) != 0 || item.Failures != 1 || item.Pending != 1 || item.RemainingNS != int64(20*time.Second) {
		t.Errorf("closeout item %+v", item)
	}
	if lines[len(lines)-2] != `{"summary":"test"}` {
		t.Errorf("caller closeout record %q", lines[len(lines)-2])
	}
}

// The real file system and the real clock, in a temporary directory: the
// files on disk match the manifest, the directory holds nothing else, and
// the terminator carries the manifest's SHA-256.
func TestRealFileSystemAndClock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "records")
	clk := RealClock{}
	w, d := Deadlines(clk.Now())
	log, err := NewLog(recordFiles...)
	if err != nil {
		t.Fatal(err)
	}
	o := NewOuter(clk, d)
	wr, err := o.RegisterWriter(WriterConfig{FS: OSFS{}, Dir: dir, Log: log})
	if err != nil {
		t.Fatal(err)
	}
	if err := wr.Start(); err != nil {
		t.Fatal(err)
	}
	log.Append("raw.jsonl", []byte(`{"raw":"é"}`))
	log.Append("derived.jsonl", []byte(`{"derived":1}`))
	var out lockedBuffer
	f := Closeout(Config[struct{}]{
		Outer: o, Log: log, W: w,
		Copy: func() struct{} { return struct{}{} },
		Provisional: func(ProvisionalInput[struct{}]) Snapshot {
			return Snapshot{Conclusion: Consistent}
		},
		Level1:       func(OuterSnapshot) bool { return false },
		CloseoutFile: "closeout.jsonl",
		Stderr:       &out,
	})
	wantFinal(t, f, Consistent, PathConsumed)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	wantNames := append([]string{ManifestName}, recordFiles...)
	sort.Strings(wantNames)
	if strings.Join(names, ",") != strings.Join(wantNames, ",") {
		t.Errorf("directory %v, want %v", names, wantNames)
	}
	content := map[string][]byte{}
	for _, n := range recordFiles {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(b, HeaderLine(n)) {
			t.Errorf("%s lacks its header", n)
		}
		content[n] = b
	}
	mf, err := os.ReadFile(filepath.Join(dir, ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if string(mf) != string(expectedManifest(content)) {
		t.Errorf("manifest %q", mf)
	}
	if !strings.HasSuffix(out.String(), TerminatorLine(sha(mf), Consistent)) {
		t.Errorf("stderr %q lacks the terminator", out.String())
	}
	if !bytes.Contains(content["raw.jsonl"], []byte(`{"raw":"é"}`+"\n")) {
		t.Error("raw item missing")
	}

	if err := (OSFS{}).Mkdir(dir); err == nil {
		t.Error("Mkdir of an existing directory succeeded")
	}
	if _, err := (OSFS{}).Create(filepath.Join(dir, ManifestName)); err == nil {
		t.Error("exclusive Create of an existing file succeeded")
	}
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func TestLog(t *testing.T) {
	for _, bad := range [][]string{nil, {""}, {"Raw.jsonl"}, {"a/b"}, {".x"}, {ManifestName}, {"a", "a"}} {
		if _, err := NewLog(bad...); err == nil {
			t.Errorf("NewLog(%q) accepted", bad)
		}
	}
	l, err := NewLog("a.jsonl", "b.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if l.Append("a.jsonl", []byte("1")) != 1 || l.Append("b.jsonl", []byte("2")) != 2 {
		t.Fatal("sequence numbers")
	}
	for _, bad := range []struct {
		file string
		line []byte
	}{{"c.jsonl", []byte("x")}, {"a.jsonl", []byte("x\ny")}, {"a.jsonl", []byte{0xff}}} {
		if seq := l.Append(bad.file, bad.line); seq != 0 {
			t.Errorf("Append(%q, %q) = %d", bad.file, bad.line, seq)
		}
	}
	if len(l.Defects()) != 3 {
		t.Errorf("defects %v", l.Defects())
	}
	seal, last, ok := l.sealWith([]Record{{File: "a.jsonl", Line: []byte("3")}})
	if !ok || seal != 3 || last["a.jsonl"] != 3 || last["b.jsonl"] != 2 {
		t.Fatalf("seal %d %v %v", seal, last, ok)
	}
	if _, _, ok := l.sealWith(nil); ok {
		t.Error("sealed twice")
	}
	if l.Append("b.jsonl", []byte("4")) != 4 {
		t.Error("append after the seal")
	}
	items, sealed, sealSeq, _ := l.take(1)
	if !sealed || sealSeq != 3 || len(items) != 3 {
		t.Errorf("take after the seal: %d items, sealed %v at %d", len(items), sealed, sealSeq)
	}
	if items, _, _, _ := l.take(4); len(items) != 0 {
		t.Error("take past the seal returned items")
	}
	if all := l.Items(); len(all) != 4 || string(all[3].Line) != "4" {
		t.Errorf("items %v", all)
	}
}

func TestFakeClock(t *testing.T) {
	c := NewFakeClock(t0)
	fired, _ := c.Timer(at(10))
	stopped, stop := c.Timer(at(10))
	stop()
	past, _ := c.Timer(t0)
	waitCh(t, past, "timer at the current instant")
	c.Set(at(9.999))
	select {
	case <-fired:
		t.Fatal("fired early")
	default:
	}
	c.Advance(time.Millisecond) // exactly at(10)
	waitCh(t, fired, "timer at its instant")
	select {
	case <-stopped:
		t.Fatal("stopped timer fired")
	default:
	}
	if c.Armed() != 0 {
		t.Errorf("armed %d", c.Armed())
	}
	defer func() {
		if recover() == nil {
			t.Error("moving backwards did not panic")
		}
	}()
	c.Set(t0)
}

func TestRealClockTimer(t *testing.T) {
	c := RealClock{}
	ch, stop := c.Timer(c.Now().Add(-time.Second))
	defer stop()
	waitCh(t, ch, "a real timer in the past")
	if w, d := Deadlines(t0); d.Sub(t0) != 120*time.Second || d.Sub(w) != 10*time.Second {
		t.Errorf("W %v D %v", w, d)
	}
}

func TestExitCode(t *testing.T) {
	for c, want := range map[Conclusion]int{Invalid: 1, Incomplete: 1, Inconsistent: 0, Consistent: 0, 0: 1, 5: 1} {
		if got := ExitCode(c); got != want {
			t.Errorf("ExitCode(%d) = %d, want %d", c, got, want)
		}
	}
}
