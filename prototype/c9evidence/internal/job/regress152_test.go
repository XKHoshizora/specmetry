package job

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/controls"
)

// Review 152 B-152-1: the expiry control of TestB151ControlsNormalAndExpiry
// BeforeTheSeal may only require a writer that is still waiting after an
// expiry exit if, before D, the writer was shown to have written every
// existing record and to be waiting. RunDomains returning does not show it:
// the work task publishes done without waiting for the writer.
//
// The handshake reads only what the writer itself publishes in the outer
// registry and the log's items (writerIdle). The writer wakes only when an
// item is appended or the log is sealed (chain.Log.take, Writer.writeUntil
// Seal), calls admit and a file operation only for bytes it has to write,
// and does not wake for D. The closeout's expiry exit at m.Run neither
// appends nor seals, and Job.Finish writes nothing to the log before it.
// So once the writer is idle and the log keeps its items and stays
// unsealed, the writer cannot move by itself; a writer without a terminal
// state after Finish is then a consequence of that precondition, not of
// scheduling.

// fsCounts counts the calls of the record file system, each counted when
// it starts.
type fsCounts struct{ Mkdir, Create, Write, Close int }

// countFS wraps the in-memory record file system: it counts every call
// and, when hold is set, holds the first Write until release (entered is
// closed when that Write is reached). release is idempotent and is also
// registered in the test's cleanup.
type countFS struct {
	mem     *memFS
	hold    bool
	gate    chan struct{}
	entered chan struct{}
	first   sync.Once
	rel     sync.Once

	mu sync.Mutex
	n  fsCounts
}

func newCountFS(hold bool) *countFS {
	return &countFS{mem: newMemFS(), hold: hold, gate: make(chan struct{}), entered: make(chan struct{})}
}

func (c *countFS) add(f func(*fsCounts)) {
	c.mu.Lock()
	f(&c.n)
	c.mu.Unlock()
}

func (c *countFS) counts() fsCounts {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

func (c *countFS) release() { c.rel.Do(func() { close(c.gate) }) }

func (c *countFS) Mkdir(dir string) error {
	c.add(func(n *fsCounts) { n.Mkdir++ })
	return c.mem.Mkdir(dir)
}

func (c *countFS) Create(p string) (chain.File, error) {
	c.add(func(n *fsCounts) { n.Create++ })
	f, err := c.mem.Create(p)
	if err != nil {
		return f, err
	}
	return &countFile{File: f, fs: c}, nil
}

// containing returns the record files whose bytes contain s.
func (c *countFS) containing(s string) []string {
	c.mem.mu.Lock()
	defer c.mem.mu.Unlock()
	var out []string
	for p, f := range c.mem.files {
		f.mu.Lock()
		if bytes.Contains(f.b.Bytes(), []byte(s)) {
			out = append(out, p)
		}
		f.mu.Unlock()
	}
	return out
}

type countFile struct {
	chain.File
	fs *countFS
}

func (f *countFile) Write(b []byte) (int, error) {
	f.fs.add(func(n *fsCounts) { n.Write++ })
	if f.fs.hold {
		held := false
		f.fs.first.Do(func() { held = true; close(f.fs.entered) })
		if held {
			<-f.fs.gate
		}
	}
	return f.File.Write(b)
}

func (f *countFile) Close() error {
	f.fs.add(func(n *fsCounts) { n.Close++ })
	return f.File.Close()
}

// writerIdle reports whether the job's record writer has written every item
// now in the log and has nothing left to write, from the registry copy o
// taken before items and sealed were read; why says what is missing. It
// requires: the writer started, its directory created, no terminal state
// and no chain error; every record file created, its header written, its
// WrittenThrough equal to the log's last item for that file (0 when none)
// and no Close started; and the log not sealed. WrittenThrough moves only
// after a write returned without error and was published, and items only
// grow, so an item appended after o has a sequence number above every
// WrittenThrough in o and makes the result false: when it is true, every
// item that existed when items were read was written before o.
func writerIdle(o chain.OuterSnapshot, items []chain.Item, sealed bool) (bool, string) {
	w := o.Writer
	switch {
	case w == nil || !w.Started:
		return false, "the writer is not started"
	case w.Terminal != nil:
		return false, "the writer has a terminal state"
	case len(w.Errors) > 0:
		return false, fmt.Sprintf("the writer has chain errors %v", w.Errors)
	case w.Mkdir == nil || w.Mkdir.Err != nil:
		return false, "the record directory is not created"
	case sealed:
		return false, "the log is sealed"
	}
	last := map[string]int{}
	for _, it := range items {
		last[it.File] = it.Seq
	}
	known := map[string]bool{}
	for _, f := range w.Files {
		known[f.Name] = true
		switch {
		case f.Create == nil || f.Create.Err != nil:
			return false, f.Name + " is not created"
		case !f.HeaderWritten:
			return false, f.Name + " has no header written"
		case f.WrittenThrough != last[f.Name]:
			return false, fmt.Sprintf("%s is written through %d, the log's last item for it is %d", f.Name, f.WrittenThrough, last[f.Name])
		case f.Close != nil || f.Holding.CloseStarted:
			return false, f.Name + " is being closed"
		}
	}
	for name := range last {
		if !known[name] {
			return false, "the log has items for " + name + ", which the writer does not have"
		}
	}
	return true, ""
}

// fileWrites is one record file's published write state.
type fileWrites struct{ Writes, Through int }

func writesOf(o chain.OuterSnapshot) map[string]fileWrites {
	out := map[string]fileWrites{}
	if o.Writer != nil {
		for _, f := range o.Writer.Files {
			out[f.Name] = fileWrites{len(f.Writes), f.WrittenThrough}
		}
	}
	return out
}

func sameWrites(a, b map[string]fileWrites) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// idleState is what the handshake fixed: the log's item count, the
// published writes per record file and the file system's call counts.
type idleState struct {
	items  int
	writes map[string]fileWrites
	ops    fsCounts
}

// waitWriterIdle waits at most d for writerIdle, judging again at every
// publication of the outer registry: only a publication of the writer can
// make it true (a new item only makes it false), so no change is missed.
// It returns what it fixed, or false with the last reason.
func waitWriterIdle(j *Job, fs *countFS, d time.Duration) (idleState, string, bool) {
	timeout := time.After(d)
	for {
		s, ch := j.outer.Watch()
		items := j.log.Items()
		sealed, _ := j.log.Sealed()
		ok, why := writerIdle(s, items, sealed)
		if ok {
			return idleState{items: len(items), writes: writesOf(s), ops: fs.counts()}, "", true
		}
		select {
		case <-ch:
		case <-timeout:
			return idleState{}, why, false
		}
	}
}

// expireWaitingWriter is the wake control, from a writer the handshake
// found idle: D, the closeout's expiry result, "still waiting" derived
// from the unchanged precondition, endTasks's wake, and the checks that the
// wake added nothing to the record files and started no file operation.
func expireWaitingWriter(t *testing.T, j *Job, clk *chain.FakeClock, fs *countFS, idle idleState) {
	t.Helper()
	clk.Set(j.D)
	finishWant(t, j, chain.Incomplete, chain.PathExpiredAtMRun)
	o := j.outer.Snapshot()
	items := j.log.Items()
	sealed, _ := j.log.Sealed()
	if len(items) != idle.items || sealed || !o.Latch {
		t.Fatalf("after Finish: %d log items (handshake %d), sealed %v, latch %v: the precondition no longer holds", len(items), idle.items, sealed, o.Latch)
	}
	if w, d := taskStates(o); w != "" || d != diagNone {
		t.Fatalf("after Finish: writer %q diagnostic %q, want the writer still waiting (nothing woke it since the handshake) and no diagnostic task", w, d)
	}
	if got := writesOf(o); !sameWrites(got, idle.writes) {
		t.Fatalf("after Finish: published writes %v, handshake %v", got, idle.writes)
	}
	if c := fs.counts(); c != idle.ops {
		t.Fatalf("after Finish: file system calls %+v, handshake %+v", c, idle.ops)
	}

	end := endTasks(t, j)
	wantTasks(t, end, writerExpired, diagNone)
	items = j.log.Items()
	if len(items) != idle.items+1 || items[len(items)-1].File != FileCloseout || string(items[len(items)-1].Line) != wakeLine {
		t.Errorf("after endTasks: %d log items (handshake %d), want exactly the test's wake item appended to %s", len(items), idle.items, FileCloseout)
	}
	if got := writesOf(end); !sameWrites(got, idle.writes) {
		t.Errorf("after the wake: published writes %v, handshake %v", got, idle.writes)
	}
	if c := fs.counts(); c != idle.ops {
		t.Errorf("after the wake: file system calls %+v, handshake %+v: a file operation started after the expiry", c, idle.ops)
	}
	if p := fs.containing(wakeLine); len(p) > 0 {
		t.Errorf("the wake item reached the record files %v", p)
	}
	if end.Writer != nil {
		for _, f := range end.Writer.Files {
			if f.Close != nil || f.Holding.CloseStarted {
				t.Errorf("%s: a Close started after the expiry", f.Name)
			}
		}
		if end.Writer.Manifest.Create != nil {
			t.Error("the manifest was created after the expiry")
		}
	}
	if n := finishCalls(j); n != 1 {
		t.Errorf("Finish called %d times, want exactly 1", n)
	}
}

// B-152-1: the writer lags behind the work task (its first Write is held,
// entry handshake). After RunDomains the handshake is false, and a short
// wait fails because of the hold, not because of scheduling; after the
// release the handshake holds and the wake control runs as in the expiry
// control.
func TestB152HandshakeWaitsForALaggingWriter(t *testing.T) {
	fs := newCountFS(true)
	j, clk, _, g := newJob148FS(t, controls.LC3(), nil, fs)
	settle(t, j, fs.release, g.freeAll)
	if !waitFor(t, fs.entered, "the writer's first Write is held") {
		return
	}
	j.RunDomains()
	if w := j.Snapshot().Work; w == nil || w.State != WorkDone {
		t.Fatalf("work %+v after RunDomains, want done", w)
	}
	sealed, _ := j.log.Sealed()
	if ok, _ := writerIdle(j.outer.Snapshot(), j.log.Items(), sealed); ok {
		t.Fatal("the handshake held while the writer's first Write was held")
	}
	if _, _, ok := waitWriterIdle(j, fs, failingWait); ok {
		t.Fatal("the handshake passed while the writer's first Write was held")
	}
	if c := fs.counts(); c.Write != 1 {
		t.Fatalf("%d Write calls while the first was held, want 1", c.Write)
	}
	fs.release()
	idle, why, ok := waitWriterIdle(j, fs, bound)
	if !ok {
		t.Fatalf("the writer was not idle within %s after the release: %s", bound, why)
	}
	if idle.ops.Write < 2 {
		t.Fatalf("idle after %d Write calls: the held Write's file and the others must have been written", idle.ops.Write)
	}
	expireWaitingWriter(t, j, clk, fs, idle)
}

// B-152-1, a separate healthy branch that needs no wake. It is NOT the wake
// control: the writer is held inside its first Write until after the
// closeout's expiry exit; released, it publishes that write and takes its
// expiry exit at its next decision point by itself (a legal early expiry),
// and endTasks appends nothing.
func TestB152WriterExpiredByItselfNeedsNoWake(t *testing.T) {
	fs := newCountFS(true)
	j, clk, _, g := newJob148FS(t, controls.LC3(), nil, fs)
	settle(t, j, fs.release, g.freeAll)
	if !waitFor(t, fs.entered, "the writer's first Write is held") {
		return
	}
	j.RunDomains()
	clk.Set(j.D)
	finishWant(t, j, chain.Incomplete, chain.PathExpiredAtMRun)
	if w, _ := taskStates(j.outer.Snapshot()); w != "" {
		t.Fatalf("writer %q while held inside its first Write", w)
	}
	n := len(j.log.Items())
	fs.release()
	timeout := time.After(bound)
	for {
		s, ch := j.outer.Watch()
		if s.Writer != nil && s.Writer.Terminal != nil {
			break
		}
		select {
		case <-ch:
		case <-timeout:
			t.Fatalf("the released writer did not end within %s", bound)
		}
	}
	ops := fs.counts()
	wantTasks(t, endTasks(t, j), writerExpired, diagNone)
	if m := len(j.log.Items()); m != n {
		t.Errorf("%d log items after endTasks, %d before: a wake item was appended for a writer that had ended", m, n)
	}
	if c := fs.counts(); c != ops {
		t.Errorf("file system calls %+v after endTasks, %+v before", c, ops)
	}
	if p := fs.containing(wakeLine); len(p) > 0 {
		t.Errorf("a wake item reached the record files %v", p)
	}
	if k := finishCalls(j); k != 1 {
		t.Errorf("Finish called %d times, want exactly 1", k)
	}
}
