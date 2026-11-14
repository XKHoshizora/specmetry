package job

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/controls"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

// Review 149 B-147-1-R-c/P regressions: every new file operation of an
// outer preparation or change must be admitted against W right before it
// starts, not covered by one admission at the entry. They run the real
// RunDomains → Finish wiring with the job's outer registry, the fake clock
// and controlled blocking of the fixture file operations; each wait has its
// own bounded failure exit and every block is released in t.Cleanup.

// fileCall is one fixture file operation: its kind, its path, the clock's
// reading when it returned, and its result.
type fileCall struct {
	kind, path string
	at         time.Time
	err        error
}

// fileCtl blocks one call after its effect is applied (entered is closed,
// the call returns only after release) and may make it return err without
// doing anything.
type fileCtl struct {
	block            bool
	err              error
	entered, release chan struct{}
	ent, rel         sync.Once
}

func blockCall() *fileCtl {
	return &fileCtl{block: true, entered: make(chan struct{}), release: make(chan struct{})}
}

func (c *fileCtl) free() { c.rel.Do(func() { close(c.release) }) }

// fileSeams replace every fixture file operation of the job: the fixture
// root and fixture directories (created below base), MkdirAll, WriteFile,
// Remove, Mkdir, Symlink and RemoveAll. Kinds: "mkdirtemp-root",
// "mkdirtemp", "mkdirall", "write", "remove", "mkdir", "symlink",
// "removeall".
type fileSeams struct {
	clk   *chain.FakeClock
	base  string
	mu    sync.Mutex
	calls []fileCall
	n     map[string]int
	ctl   map[string]*fileCtl
}

func installFileSeams(t *testing.T, j *Job, clk *chain.FakeClock) *fileSeams {
	s := &fileSeams{clk: clk, base: t.TempDir(), n: map[string]int{}, ctl: map[string]*fileCtl{}}
	j.s.mkdirTemp = func(dir, pattern string) (string, error) {
		kind := "mkdirtemp"
		if dir == "" {
			kind, dir = "mkdirtemp-root", s.base
		}
		var d string
		err := s.do(kind, pattern, func() error {
			var err error
			d, err = os.MkdirTemp(dir, pattern)
			return err
		}, func() string { return d })
		return d, err
	}
	j.s.mkdirAll = func(p string) error { return s.do("mkdirall", p, func() error { return os.MkdirAll(p, 0o755) }, nil) }
	j.s.writeFile = func(p string, b []byte) error {
		return s.do("write", p, func() error { return os.WriteFile(p, b, 0o644) }, nil)
	}
	j.s.remove = func(p string) error { return s.do("remove", p, func() error { return os.Remove(p) }, nil) }
	j.s.mkdir = func(p string) error { return s.do("mkdir", p, func() error { return os.Mkdir(p, 0o755) }, nil) }
	j.s.symlink = func(o, n string) error { return s.do("symlink", n, func() error { return os.Symlink(o, n) }, nil) }
	j.s.removeAll = func(p string) error { return s.do("removeall", p, func() error { return os.RemoveAll(p) }, nil) }
	return s
}

// on installs c for the nth call (from 1) of kind; a blocking one is also
// released in t.Cleanup.
func (s *fileSeams) on(t *testing.T, kind string, nth int, c *fileCtl) *fileCtl {
	s.mu.Lock()
	s.ctl[fmt.Sprintf("%s#%d", kind, nth)] = c
	s.mu.Unlock()
	if c.block {
		t.Cleanup(c.free)
	}
	return c
}

func (s *fileSeams) do(kind, path string, op func() error, result func() string) error {
	s.mu.Lock()
	s.n[kind]++
	c := s.ctl[fmt.Sprintf("%s#%d", kind, s.n[kind])]
	s.mu.Unlock()
	var err error
	if c != nil && c.err != nil && !c.block {
		err = c.err
	} else {
		err = op()
	}
	if c != nil && c.block {
		c.ent.Do(func() { close(c.entered) })
		<-c.release
		if c.err != nil {
			err = c.err
		}
	}
	if result != nil && err == nil {
		path = result()
	}
	s.mu.Lock()
	s.calls = append(s.calls, fileCall{kind: kind, path: path, at: s.clk.Now(), err: err})
	s.mu.Unlock()
	return err
}

func (s *fileSeams) count(kind string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.calls {
		if c.kind == kind {
			n++
		}
	}
	return n
}

func (s *fileSeams) list(kind string) []fileCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []fileCall
	for _, c := range s.calls {
		if c.kind == kind {
			out = append(out, c)
		}
	}
	return out
}

// newJob149 is newJob148 with the file seams installed.
func newJob149(t *testing.T, vs []controls.Variant) (*Job, *chain.FakeClock, *fileSeams, *gates) {
	t.Helper()
	j, clk, _, g := newJob148(t, vs, nil)
	return j, clk, installFileSeams(t, j, clk), g
}

// opsWithPrefix returns the outer operations whose names start with p.
func opsWithPrefix(o chain.OuterSnapshot, p string) []chain.OpSnapshot {
	var out []chain.OpSnapshot
	for _, op := range o.Ops {
		if strings.HasPrefix(op.Name, p) {
			out = append(out, op)
		}
	}
	return out
}

func refusedOps(ops []chain.OpSnapshot) []chain.OpSnapshot {
	var out []chain.OpSnapshot
	for _, op := range ops {
		if op.State == chain.OpRefused {
			out = append(out, op)
		}
	}
	return out
}

func fRecord(t *testing.T, j *Job, variant string) FRecord {
	t.Helper()
	for _, r := range j.Snapshot().FRecords {
		if r.Variant.Variant == variant {
			return r
		}
	}
	t.Fatalf("no F record for %s", variant)
	return FRecord{}
}

func heldDir(j *Job, dir string) *HeldDir {
	for _, h := range j.Snapshot().Held {
		if h.Dir == dir {
			h := h
			return &h
		}
	}
	return nil
}

// crossW blocks the nth call of kind in the preparation of v, moves the
// clock to at (at or after W, so the parent stops waiting) while it is
// blocked, releases it, and waits for the work task. It returns the job,
// the clock and the seams.
func crossW(t *testing.T, v controls.Variant, kind string, nth int, at time.Duration, err error) (*Job, *chain.FakeClock, *fileSeams) {
	t.Helper()
	j, clk, fs, g := newJob149(t, []controls.Variant{v})
	c := blockCall()
	c.err = err
	fs.on(t, kind, nth, c)
	settle(t, j, c.free, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, c.entered, kind+" entered") {
		t.FailNow()
	}
	clk.Set(j.W.Add(at))
	if !waitFor(t, ran, "RunDomains returned at W") {
		t.FailNow()
	}
	c.free()
	if !waitFor(t, j.WorkDone(), "the work task finished") {
		t.FailNow()
	}
	return j, clk, fs
}

// R-c/P 1: the fixture directory's MkdirTemp is admitted before W and
// returns success only after D. The directory is held; no MkdirAll or
// WriteFile starts after it; the later step is refused (registered, not
// started); no cleanup starts after D.
func TestB149MkdirTempReturnsAfterD(t *testing.T) {
	v := controls.LC3()[0]
	j, clk, fs, g := newJob149(t, []controls.Variant{v})
	c := fs.on(t, "mkdirtemp", 1, blockCall())
	settle(t, j, c.free, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, c.entered, "the fixture directory was created") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	clk.Set(j.D)
	f, ok := finish(t, j)
	if !ok {
		return
	}
	if !notValid(f) || f.Path != chain.PathExpiredAtMRun {
		t.Errorf("final %v exit %d path %v", f.Conclusion, f.ExitCode, f.Path)
	}
	c.free()
	if !waitFor(t, j.WorkDone(), "the work task finished") {
		return
	}
	if n, m := fs.count("mkdirall"), fs.count("write"); n != 0 || m != 0 {
		t.Errorf("after the late MkdirTemp: %d MkdirAll and %d WriteFile started, want none", n, m)
	}
	if n := fs.count("removeall"); n != 0 {
		t.Errorf("%d cleanups started after D", n)
	}
	mk := fs.list("mkdirtemp")
	if len(mk) != 1 || mk[0].err != nil || mk[0].at.Before(j.D) {
		t.Fatalf("fixture MkdirTemp %+v, want one success returned at or after D", mk)
	}
	if h := heldDir(j, mk[0].path); h == nil || h.Removed {
		t.Errorf("the created directory %s is not held: %+v", mk[0].path, h)
	}
	if _, err := os.Stat(mk[0].path); err != nil {
		t.Errorf("the created directory must stay: %v", err)
	}
	ops := opsWithPrefix(j.outer.Snapshot(), "prepare LC-3 "+v.Name)
	if len(ops) == 0 || ops[0].State != chain.OpDone || ops[0].Outcome == nil || ops[0].Outcome.Qual != chain.Late {
		t.Errorf("the directory creation's late result is not kept: %+v", ops)
	}
	if r := refusedOps(ops); len(r) != 1 {
		t.Errorf("refused steps %+v, want exactly the next step", r)
	}
	if r := fRecord(t, j, v.Name); !errors.Is(r.Outcome.PrepErr, controls.ErrNotAdmitted) || r.Variant.Judgement != verdict.Miss {
		t.Errorf("F record %+v", r)
	}
}

// R-c/P 2: a MkdirAll admitted before W returns after W; the WriteFile
// after it does not start.
func TestB149MkdirAllCrossesW(t *testing.T) {
	v := controls.LC3()[0]
	j, _, fs := crossW(t, v, "mkdirall", 1, time.Second, nil)
	if n := fs.count("write"); n != 0 {
		t.Errorf("%d WriteFile started after a MkdirAll that returned after W, want none", n)
	}
	if n := fs.count("mkdirall"); n != 1 {
		t.Errorf("%d MkdirAll, want 1", n)
	}
	ops := opsWithPrefix(j.outer.Snapshot(), "prepare LC-3 "+v.Name)
	if r := refusedOps(ops); len(r) != 1 || r[0].Outcome == nil || r[0].Outcome.Err != nil {
		t.Errorf("refused steps %+v, want exactly one, never started", r)
	}
	if r := fRecord(t, j, v.Name); !errors.Is(r.Outcome.PrepErr, controls.ErrNotAdmitted) || r.Variant.Judgement != verdict.Miss {
		t.Errorf("F record %+v", r)
	}
	if mk := fs.list("mkdirtemp"); len(mk) != 1 || heldDir(j, mk[0].path) == nil || heldDir(j, mk[0].path).Removed {
		t.Errorf("the partly prepared directory is not held by the job: %+v", mk)
	}
}

// R-c/P 3: the first file's WriteFile returns after W; the next file's
// operations do not start.
func TestB149WriteCrossesW(t *testing.T) {
	v := controls.LC3()[0]
	j, _, fs := crossW(t, v, "write", 1, time.Second, nil)
	if n, m := fs.count("mkdirall"), fs.count("write"); n != 1 || m != 1 {
		t.Errorf("MkdirAll %d WriteFile %d, want 1 and 1: the next file must not start", n, m)
	}
	ops := opsWithPrefix(j.outer.Snapshot(), "prepare LC-3 "+v.Name)
	if r := refusedOps(ops); len(r) != 1 {
		t.Errorf("refused steps %+v, want exactly one", r)
	}
	if w := fs.list("write"); len(w) != 1 || w[0].err != nil || w[0].at.Before(j.W) {
		t.Errorf("write %+v, want one nil result returned after W", w)
	}
}

// R-c/P 4: LC-3's Remove returns after W; the Mkdir after it does not
// start. The fixture's own cleanup, between W and D, is still admitted.
func TestB149RemoveCrossesW(t *testing.T) {
	v := controls.LC3()[2] // handle not a regular file: Remove, then Mkdir
	j, _, fs := crossW(t, v, "remove", 1, time.Second, nil)
	if n := fs.count("mkdir"); n != 0 {
		t.Errorf("%d Mkdir started after a Remove that returned after W, want none", n)
	}
	ops := opsWithPrefix(j.outer.Snapshot(), "prepare LC-3 "+v.Name)
	if r := refusedOps(ops); len(r) != 1 {
		t.Errorf("refused steps %+v, want exactly the Mkdir", r)
	}
	r := fRecord(t, j, v.Name)
	if !errors.Is(r.Outcome.PrepErr, controls.ErrNotAdmitted) || r.Outcome.CleanupErr != nil || r.Variant.Judgement != verdict.Miss {
		t.Errorf("F record %+v", r)
	}
	rm := fs.list("removeall")
	if len(rm) != 2 { // the fixture's cleanup and the root's, both before D
		t.Fatalf("cleanups %+v, want the fixture's and the root's", rm)
	}
	for _, c := range rm {
		if c.err != nil || !c.at.Before(j.D) {
			t.Errorf("cleanup %+v", c)
		}
		if h := heldDir(j, c.path); h == nil || !h.Removed {
			t.Errorf("%s not released after its cleanup: %+v", c.path, h)
		}
	}
}

// R-c/P 5: a new preparation step at exactly W is refused (W is late); a
// cleanup at W, before D, is still admitted.
func TestB149StepAtWIsRefusedCleanupAdmitted(t *testing.T) {
	v := controls.LC3()[0]
	j, _, fs := crossW(t, v, "mkdirall", 1, 0, nil)
	if n := fs.count("write"); n != 0 {
		t.Errorf("%d WriteFile started at W, want none", n)
	}
	ops := opsWithPrefix(j.outer.Snapshot(), "prepare LC-3 "+v.Name)
	r := refusedOps(ops)
	if len(r) != 1 || r[0].Outcome == nil || !r[0].Outcome.At.Equal(j.W) {
		t.Errorf("refused steps %+v, want one refused at W", r)
	}
	root := opsNamed(j.outer.Snapshot(), "cleanup fixture root")
	if len(root) != 1 || root[0].State != chain.OpDone || root[0].Outcome == nil || !root[0].Outcome.At.Equal(j.W) {
		t.Errorf("root cleanup %+v, want admitted and done at W", root)
	}
}

// R-c/P 6: a real failure of a step is kept as that step's failure (also
// when it returns after W), nothing after it starts, and it is consumed as
// a preparation failure, apart from a refusal.
func TestB149RealFailureKeptApartFromRefusal(t *testing.T) {
	v := controls.LC3()[0]
	werr := errors.New("injected write failure")

	// Before W.
	j, _, fs, g := newJob149(t, []controls.Variant{v})
	fs.on(t, "write", 1, &fileCtl{err: werr})
	settle(t, j, g.freeAll)
	j.RunDomains()
	if n, m := fs.count("mkdirall"), fs.count("write"); n != 1 || m != 1 {
		t.Errorf("MkdirAll %d WriteFile %d after a failed write, want 1 and 1", n, m)
	}
	// afterW: the step returned at or after W. The outer registry's
	// qualification is relative to D, so a result returned between W and D
	// is timely there; what matters here is that it is kept as the step's
	// real failure.
	failedOne := func(j *Job, afterW bool) {
		t.Helper()
		ops := opsWithPrefix(j.outer.Snapshot(), "prepare LC-3 "+v.Name)
		var failed []chain.OpSnapshot
		for _, op := range ops {
			if op.State == chain.OpFailed {
				failed = append(failed, op)
			}
		}
		if len(failed) != 1 || failed[0].Outcome == nil || !errors.Is(failed[0].Outcome.Err, werr) || failed[0].Outcome.At.Before(j.W) == afterW {
			t.Errorf("failed steps %+v, want the write's real error (returned at or after W: %v)", failed, afterW)
		}
		if r := refusedOps(ops); len(r) != 0 {
			t.Errorf("refused steps %+v after a real failure: nothing after it is attempted", r)
		}
		r := fRecord(t, j, v.Name)
		if !errors.Is(r.Outcome.PrepErr, werr) || errors.Is(r.Outcome.PrepErr, controls.ErrNotAdmitted) || r.Variant.Judgement != verdict.Miss ||
			len(r.Variant.Reasons) != 1 || !strings.HasPrefix(r.Variant.Reasons[0], "outer preparation failed: ") {
			t.Errorf("F record %+v", r)
		}
	}
	failedOne(j, false)

	// Returned after W: the real error stays, not replaced by a refusal.
	j2, _, _ := crossW(t, v, "write", 1, time.Second, werr)
	failedOne(j2, true)

	// A refusal after a partial preparation is consumed as "not admitted".
	j3, _, _ := crossW(t, v, "write", 1, time.Second, nil)
	r := fRecord(t, j3, v.Name)
	if len(r.Variant.Reasons) != 1 || !strings.HasPrefix(r.Variant.Reasons[0], "outer preparation not admitted") {
		t.Errorf("refusal consumed as %v", r.Variant.Reasons)
	}
}

// R-c/P 7 (healthy): LC-3 with every variant prepares, changes and cleans up
// before W with every step done; the controls hit and the job is 4.
func TestB149HealthyStepsAllDone(t *testing.T) {
	j, _, fs, g := newJob149(t, controls.LC3())
	settle(t, j, g.freeAll)
	j.RunDomains()
	want := map[string]int{"mkdirtemp-root": 1, "mkdirtemp": 4, "mkdirall": 8, "write": 9, "remove": 1, "mkdir": 1, "removeall": 5}
	for kind, n := range want {
		if got := fs.count(kind); got != n {
			t.Errorf("%s: %d calls, want %d", kind, got, n)
		}
		for _, c := range fs.list(kind) {
			if c.err != nil || !c.at.Before(j.W) {
				t.Errorf("%+v", c)
			}
		}
	}
	for _, op := range j.outer.Snapshot().Ops {
		if op.State != chain.OpDone {
			t.Errorf("op %s: %v", op.Name, op.State)
		}
	}
	for _, r := range j.Snapshot().FRecords {
		if r.Variant.Judgement != verdict.Hit {
			t.Errorf("%s: %s %v", r.Variant.Variant, r.Variant.Judgement, r.Variant.Reasons)
		}
	}
	for _, h := range j.Snapshot().Held {
		if !h.Removed {
			t.Errorf("held %+v", h)
		}
	}
	if f, ok := finish(t, j); ok && (f.Conclusion != chain.Consistent || f.ExitCode != 0) {
		t.Errorf("healthy: %v exit %d %v", f.Conclusion, f.ExitCode, f.Reasons)
	}
}
