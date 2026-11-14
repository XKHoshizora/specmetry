package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// safety bounds every wait in the tests so that a defect fails the test
// instead of hanging it. It is never used to order events: every ordering
// in these tests comes from an explicit synchronization point (a blocked
// fake call, a parent gate, a published state or a trace event).
const safety = 20 * time.Second

var t0 = time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)

// at is t0 plus s seconds: the plan's relative times (W = 110 s, D = 120 s).
func at(s float64) time.Time { return t0.Add(time.Duration(s * float64(time.Second))) }

var recordFiles = []string{"raw.jsonl", "derived.jsonl", "controls.jsonl", "closeout.jsonl"}

const recDir = "/rec"

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func waitCh(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(safety):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// ctl controls one fake call: an injected error (with n bytes accepted for
// a write), and an optional block. entered is closed once the call has
// applied its effect (bytes appended, file created or closed); the call then
// returns only after release is closed.
type ctl struct {
	err     error
	n       int
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func blocking() *ctl { return &ctl{entered: make(chan struct{}), release: make(chan struct{})} }

func failing(err error, n int) *ctl { return &ctl{err: err, n: n} }

func (c *ctl) free() { c.once.Do(func() { close(c.release) }) }

func (c *ctl) wait() {
	if c == nil {
		return
	}
	if c.entered != nil {
		close(c.entered)
	}
	if c.release != nil {
		<-c.release
	}
}

// fakeFS is an in-memory FS with per-call fault injection and blocking.
type fakeFS struct {
	mu     sync.Mutex
	dirs   map[string]bool
	files  map[string]*fakeFile
	counts map[string]int
	ctls   map[string]*ctl
	calls  []string
}

type fakeFile struct {
	fs   *fakeFS
	name string
	data []byte
}

func newFakeFS() *fakeFS {
	return &fakeFS{dirs: map[string]bool{}, files: map[string]*fakeFile{}, counts: map[string]int{}, ctls: map[string]*ctl{}}
}

// set installs c for the nth call of kind ("mkdir", "create", "write",
// "close") on name (the directory for mkdir, the base name otherwise).
func (fs *fakeFS) set(t *testing.T, kind, name string, nth int, c *ctl) *ctl {
	fs.mu.Lock()
	fs.ctls[fmt.Sprintf("%s %s #%d", kind, name, nth)] = c
	fs.mu.Unlock()
	if c.release != nil {
		t.Cleanup(c.free)
	}
	return c
}

func (fs *fakeFS) begin(kind, name string) *ctl {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	k := kind + " " + name
	fs.counts[k]++
	fs.calls = append(fs.calls, k)
	return fs.ctls[fmt.Sprintf("%s #%d", k, fs.counts[k])]
}

func (fs *fakeFS) Mkdir(dir string) error {
	c := fs.begin("mkdir", dir)
	var err error
	fs.mu.Lock()
	switch {
	case c != nil && c.err != nil:
		err = c.err
	case fs.dirs[filepath.Clean(dir)]:
		err = os.ErrExist
	default:
		fs.dirs[filepath.Clean(dir)] = true
	}
	fs.mu.Unlock()
	c.wait()
	return err
}

func (fs *fakeFS) Create(path string) (File, error) {
	name := filepath.Base(path)
	c := fs.begin("create", name)
	var f *fakeFile
	var err error
	fs.mu.Lock()
	switch {
	case c != nil && c.err != nil:
		err = c.err
	case !fs.dirs[filepath.Dir(path)]:
		err = os.ErrNotExist
	case fs.files[name] != nil:
		err = os.ErrExist
	default:
		f = &fakeFile{fs: fs, name: name}
		fs.files[name] = f
	}
	fs.mu.Unlock()
	c.wait()
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (f *fakeFile) Write(p []byte) (int, error) {
	c := f.fs.begin("write", f.name)
	n, err := len(p), error(nil)
	if c != nil && c.err != nil {
		n, err = c.n, c.err
	}
	f.fs.mu.Lock()
	f.data = append(f.data, p[:n]...)
	f.fs.mu.Unlock()
	c.wait()
	return n, err
}

func (f *fakeFile) Close() error {
	c := f.fs.begin("close", f.name)
	c.wait()
	if c != nil {
		return c.err
	}
	return nil
}

func (fs *fakeFS) count(kind, name string) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.counts[kind+" "+name]
}

func (fs *fakeFS) content(name string) ([]byte, bool) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	f := fs.files[name]
	if f == nil {
		return nil, false
	}
	return append([]byte(nil), f.data...), true
}

func (fs *fakeFS) callLog() []string {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return append([]string(nil), fs.calls...)
}

// fakeStderr is the diagnostic output with per-write fault injection and
// blocking: a rule matches the nth write or the first write containing a
// substring.
type fakeStderr struct {
	mu    sync.Mutex
	buf   []byte
	lines []string
	rules []*stderrRule
}

type stderrRule struct {
	nth      int
	contains string
	c        *ctl
	used     bool
}

func (e *fakeStderr) on(t *testing.T, nth int, contains string, c *ctl) *ctl {
	e.mu.Lock()
	e.rules = append(e.rules, &stderrRule{nth: nth, contains: contains, c: c})
	e.mu.Unlock()
	if c.release != nil {
		t.Cleanup(c.free)
	}
	return c
}

func (e *fakeStderr) Write(p []byte) (int, error) {
	e.mu.Lock()
	nth := len(e.lines) + 1
	line := string(p)
	var c *ctl
	for _, r := range e.rules {
		if !r.used && ((r.nth > 0 && r.nth == nth) || (r.contains != "" && strings.Contains(line, r.contains))) {
			r.used = true
			c = r.c
			break
		}
	}
	n, err := len(p), error(nil)
	if c != nil && c.err != nil {
		n, err = c.n, c.err
	}
	e.buf = append(e.buf, p[:n]...)
	e.lines = append(e.lines, line)
	e.mu.Unlock()
	c.wait()
	return n, err
}

func (e *fakeStderr) text() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return string(e.buf)
}

// harness wires a fake clock, a log with the four record files, the outer
// registry, a fake FS and a fake stderr, and records the parent's trace.
type harness struct {
	t      *testing.T
	clk    *FakeClock
	w, d   time.Time
	log    *Log
	o      *Outer
	fs     *fakeFS
	stderr *fakeStderr
	wr     *Writer

	mu       sync.Mutex
	gates    map[point]chan struct{}
	arrived  map[point]chan struct{}
	events   []traceEvent
	evNotify chan struct{}
	provIn   []ProvisionalInput[int]
	copies   int
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	clk := NewFakeClock(t0)
	w, d := Deadlines(t0)
	log, err := NewLog(recordFiles...)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{
		t: t, clk: clk, w: w, d: d, log: log,
		o:        NewOuter(clk, d),
		fs:       newFakeFS(),
		stderr:   &fakeStderr{},
		gates:    map[point]chan struct{}{},
		arrived:  map[point]chan struct{}{},
		evNotify: make(chan struct{}),
	}
	h.fs.dirs[filepath.Clean("/")] = true
	return h
}

// startWriter registers the writer (before "m.Run") and starts it.
func (h *harness) startWriter() {
	h.t.Helper()
	wr, err := h.o.RegisterWriter(WriterConfig{FS: h.fs, Dir: recDir, Log: h.log})
	if err != nil {
		h.t.Fatal(err)
	}
	if err := wr.Start(); err != nil {
		h.t.Fatal(err)
	}
	h.wr = wr
}

// addItems appends a fixed set of producer items and returns the last
// sequence number.
func (h *harness) addItems() int {
	h.log.Append("raw.jsonl", []byte(`{"raw":1}`))
	h.log.Append("derived.jsonl", []byte(`{"derived":1}`))
	h.log.Append("raw.jsonl", []byte(`{"raw":2}`))
	return h.log.Append("controls.jsonl", []byte(`{"controls":1}`))
}

// appendItems appends the producer items and waits until the writer has
// written them.
func (h *harness) appendItems() {
	h.t.Helper()
	last := h.addItems()
	h.waitFor("producer items written", func(s OuterSnapshot) bool {
		if s.Writer == nil || len(s.Writer.Files) != len(recordFiles) {
			return false
		}
		through := 0
		for _, f := range s.Writer.Files {
			if !f.HeaderWritten {
				return false
			}
			through = max(through, f.WrittenThrough)
		}
		return through >= last
	})
}

func (h *harness) cfg(prov Conclusion, failures, pending []string) Config[int] {
	return Config[int]{
		Outer: h.o,
		Log:   h.log,
		W:     h.w,
		Copy: func() int {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.copies++
			return h.copies
		},
		Provisional: func(in ProvisionalInput[int]) Snapshot {
			h.mu.Lock()
			h.provIn = append(h.provIn, in)
			h.mu.Unlock()
			return Snapshot{
				Conclusion: prov,
				Records:    []Record{{File: "closeout.jsonl", Line: []byte(`{"summary":"test"}`)}},
				Failures:   failures,
				Pending:    pending,
			}
		},
		Level1:       func(OuterSnapshot) bool { return false },
		CloseoutFile: "closeout.jsonl",
		Stderr:       h.stderr,
		hook:         h.hook,
		trace:        h.record,
	}
}

func (h *harness) run(cfg Config[int]) <-chan Final {
	ch := make(chan Final, 1)
	go func() { ch <- Closeout(cfg) }()
	return ch
}

func (h *harness) final(ch <-chan Final) Final {
	h.t.Helper()
	select {
	case f := <-ch:
		return f
	case <-time.After(safety):
		h.t.Fatal("timed out waiting for Closeout to return")
	}
	return Final{}
}

// gate makes the parent stop at p (each time it reaches it) until the
// returned function is called.
func (h *harness) gate(p point) func() {
	g := make(chan struct{})
	h.mu.Lock()
	h.gates[p] = g
	h.arrived[p] = make(chan struct{}, 1)
	h.mu.Unlock()
	var once sync.Once
	open := func() { once.Do(func() { close(g) }) }
	h.t.Cleanup(open)
	return open
}

func (h *harness) hook(p point) {
	h.mu.Lock()
	g, a := h.gates[p], h.arrived[p]
	h.mu.Unlock()
	if g == nil {
		return
	}
	select {
	case a <- struct{}{}:
	default:
	}
	<-g
}

func (h *harness) waitArrived(p point) {
	h.t.Helper()
	h.mu.Lock()
	a := h.arrived[p]
	h.mu.Unlock()
	waitCh(h.t, a, fmt.Sprintf("parent at gate %d", p))
}

func (h *harness) record(ev traceEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, ev)
	close(h.evNotify)
	h.evNotify = make(chan struct{})
}

func (h *harness) trace() []traceEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]traceEvent(nil), h.events...)
}

func (h *harness) waitEvent(what string, pred func(traceEvent) bool) traceEvent {
	h.t.Helper()
	timeout := time.After(safety)
	for {
		h.mu.Lock()
		evs, ch := append([]traceEvent(nil), h.events...), h.evNotify
		h.mu.Unlock()
		for _, ev := range evs {
			if pred(ev) {
				return ev
			}
		}
		select {
		case <-ch:
		case <-timeout:
			h.t.Fatalf("timed out waiting for trace event %s; have %+v", what, evs)
		}
	}
}

// finalEvents returns the step-5 branches in order.
func (h *harness) finalBranches() []string {
	var out []string
	for _, ev := range h.trace() {
		if ev.step == pointFinalWake {
			out = append(out, ev.branch)
		}
	}
	return out
}

func (h *harness) waitFor(what string, cond func(OuterSnapshot) bool) OuterSnapshot {
	h.t.Helper()
	timeout := time.After(safety)
	for {
		s, ch := h.o.Watch()
		if cond(s) {
			return s
		}
		select {
		case <-ch:
		case <-timeout:
			h.t.Fatalf("timed out waiting for %s; writer %v", what, s.Writer)
		}
	}
}

func writerTerminal(s OuterSnapshot) bool { return s.Writer != nil && s.Writer.Terminal != nil }

func diagTerminal(s OuterSnapshot) bool { return s.Diag != nil && s.Diag.Terminal != nil }

func (h *harness) provInputs() []ProvisionalInput[int] {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]ProvisionalInput[int](nil), h.provIn...)
}

func (h *harness) copyCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.copies
}

// sealedContent is the expected bytes of each record file: the header and
// every item up to the seal.
func (h *harness) sealedContent() map[string][]byte {
	_, seal := h.log.Sealed()
	out := map[string][]byte{}
	for _, f := range recordFiles {
		out[f] = HeaderLine(f)
	}
	for _, it := range h.log.Items() {
		if it.Seq <= seal {
			out[it.File] = append(append(out[it.File], it.Line...), '\n')
		}
	}
	return out
}

// expectedManifest is the manifest for these record file contents.
func expectedManifest(content map[string][]byte) []byte {
	names := []string{"closeout.jsonl", "controls.jsonl", "derived.jsonl", "raw.jsonl"}
	var b []byte
	for _, n := range names {
		b = append(b, ManifestLine(n, int64(len(content[n])), sha(content[n]))...)
	}
	return b
}

// snapshotLines are the diagnostic lines before the chain lines.
func snapshotLines(prov Conclusion, failures, pending []string) string {
	s := ProvisionalLine(prov) + CountLine("failure", len(failures))
	for _, f := range failures {
		s += TextLine("failure", f)
	}
	s += CountLine("pending", len(pending))
	for _, p := range pending {
		s += TextLine("pending", p)
	}
	return s
}

// successLines are the chain lines of a successful writer.
func successLines(content map[string][]byte, manifest []byte, prov Conclusion) string {
	var s string
	for _, n := range []string{"closeout.jsonl", "controls.jsonl", "derived.jsonl", "raw.jsonl"} {
		s += RecordLine(n, int64(len(content[n])), sha(content[n]))
	}
	return s + ManifestHashLine(int64(len(manifest)), sha(manifest)) + TerminatorLine(sha(manifest), prov)
}

// checkCompleteRecords asserts that the record files and the manifest in
// the fake FS are exactly the sealed items, and returns them.
func (h *harness) checkCompleteRecords() (map[string][]byte, []byte) {
	h.t.Helper()
	want := h.sealedContent()
	for _, f := range recordFiles {
		got, ok := h.fs.content(f)
		if !ok || string(got) != string(want[f]) {
			h.t.Errorf("%s = %q, want %q", f, got, want[f])
		}
		if n := h.fs.count("close", f); n != 1 {
			h.t.Errorf("%s closed %d times, want 1", f, n)
		}
	}
	mf, ok := h.fs.content(ManifestName)
	if wantMF := expectedManifest(want); !ok || string(mf) != string(wantMF) {
		h.t.Errorf("manifest = %q, want %q", mf, wantMF)
	}
	if n := h.fs.count("close", ManifestName); n != 1 {
		h.t.Errorf("manifest closed %d times, want 1", n)
	}
	return want, mf
}

// allQualified asserts that every writer and diagnostic result has this
// qualification.
func allQualified(t *testing.T, s OuterSnapshot, q Qualification) {
	t.Helper()
	w := s.Writer
	check := func(what string, oc *Outcome) {
		if oc != nil && oc.Qual != q {
			t.Errorf("%s qualification %s, want %s", what, oc.Qual, q)
		}
	}
	check("mkdir", w.Mkdir)
	for _, f := range w.Files {
		check("create "+f.Name, f.Create)
		for _, wr := range f.Writes {
			oc := wr.Outcome
			check("write "+f.Name, &oc)
		}
		check("close "+f.Name, f.Close)
	}
	check("manifest create", w.Manifest.Create)
	if w.Manifest.Write != nil {
		oc := w.Manifest.Write.Outcome
		check("manifest write", &oc)
	}
	check("manifest close", w.Manifest.Close)
	if w.Terminal != nil && w.Terminal.Qual != q {
		t.Errorf("writer terminal %s, want %s", w.Terminal.Qual, q)
	}
	if s.Diag != nil {
		for i, d := range s.Diag.Writes {
			if d.Qual != q {
				t.Errorf("diagnostic write %d %q qualification %s, want %s", i+1, d.Line, d.Qual, q)
			}
		}
		if s.Diag.Terminal != nil && s.Diag.Terminal.Qual != q {
			t.Errorf("diagnostic terminal %s, want %s", s.Diag.Terminal.Qual, q)
		}
	}
}

func wantFinal(t *testing.T, f Final, c Conclusion, path ExitPath) {
	t.Helper()
	if f.Conclusion != c || f.Path != path {
		t.Fatalf("final = %v via %s (reasons %q, caps %q), want %v via %s", f.Conclusion, f.Path, f.Reasons, f.Caps, c, path)
	}
	if want := ExitCode(c); f.ExitCode != want {
		t.Fatalf("exit code %d, want %d", f.ExitCode, want)
	}
}

func hasReason(rs []string, sub string) bool {
	for _, r := range rs {
		if strings.Contains(r, sub) {
			return true
		}
	}
	return false
}
