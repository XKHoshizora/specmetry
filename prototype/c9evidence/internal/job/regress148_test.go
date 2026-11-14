package job

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/controls"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

// Review 148 regressions (R-a, R-b, R-c of B-147-1; R-a is shared with
// B-147-4). Each runs the real RunDomains → Finish wiring in-process, with
// the fake clock, the fixture repository's fake caller, an in-memory record
// file system (so a writer that ends on an expiry exit leaves no open file
// behind) and controlled blocking at named points. Every wait has its own
// bounded failure exit, every block is released in t.Cleanup, and the
// cleanup waits (bounded) for the work task to finish before the
// temporary directories go. No helper process is started.

// memFS is an in-memory chain.FS.
type memFS struct {
	mu    sync.Mutex
	dirs  map[string]bool
	files map[string]*memFile
}

func newMemFS() *memFS { return &memFS{dirs: map[string]bool{}, files: map[string]*memFile{}} }

func (m *memFS) Mkdir(dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.dirs[dir] {
		return os.ErrExist
	}
	m.dirs[dir] = true
	return nil
}

func (m *memFS) Create(p string) (chain.File, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.files[p]; ok {
		return nil, os.ErrExist
	}
	f := &memFile{}
	m.files[p] = f
	return f, nil
}

type memFile struct {
	mu     sync.Mutex
	b      bytes.Buffer
	closed bool
}

func (f *memFile) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, os.ErrClosed
	}
	return f.b.Write(p)
}

func (f *memFile) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return os.ErrClosed
	}
	f.closed = true
	return nil
}

// opCall is one call of the outer layer's directory operations.
type opCall struct {
	op, path string // "mkdir" (path is the created directory, or the pattern on failure) or "remove"
	at       time.Time
	err      error
}

// opSeams count the outer layer's directory operations with the fake
// clock's reading, create every directory below base, and inject failures.
type opSeams struct {
	clk   *chain.FakeClock
	base  string
	mu    sync.Mutex
	calls []opCall
	nfx   int
	// failFx makes the n-th fixture creation (pattern "fx-", from 1) fail
	// without creating anything; failRoot makes the fixture root's removal
	// fail without removing anything.
	failFx   int
	failRoot bool
	root     string
}

func (s *opSeams) mkdirTemp(dir, pattern string) (string, error) {
	if dir == "" {
		dir = s.base
	}
	s.mu.Lock()
	if pattern == "fx-" {
		s.nfx++
		if s.nfx == s.failFx {
			err := errors.New("injected mkdir failure")
			s.calls = append(s.calls, opCall{op: "mkdir", path: pattern, at: s.clk.Now(), err: err})
			s.mu.Unlock()
			return "", err
		}
	}
	s.mu.Unlock()
	d, err := os.MkdirTemp(dir, pattern)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, opCall{op: "mkdir", path: d, at: s.clk.Now(), err: err})
	if pattern == "c9evidence-fixtures-" {
		s.root = d
	}
	return d, err
}

func (s *opSeams) removeAll(p string) error {
	s.mu.Lock()
	inject := s.failRoot && p == s.root
	s.mu.Unlock()
	var err error
	if inject {
		err = errors.New("injected remove failure")
	} else {
		err = os.RemoveAll(p)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, opCall{op: "remove", path: p, at: s.clk.Now(), err: err})
	return err
}

func (s *opSeams) list(op string) []opCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []opCall
	for _, c := range s.calls {
		if c.op == op {
			out = append(out, c)
		}
	}
	return out
}

// gate blocks the first arrival at one named point until released.
type gate struct {
	reached, release chan struct{}
	once, rel        sync.Once
	do               func() // runs at the first arrival, before it blocks
}

func (g *gate) free() { g.rel.Do(func() { close(g.release) }) }

type gates struct {
	mu sync.Mutex
	m  map[string]*gate
}

func newGates() *gates { return &gates{m: map[string]*gate{}} }

func (g *gates) add(point string) *gate {
	g.mu.Lock()
	defer g.mu.Unlock()
	x := &gate{reached: make(chan struct{}), release: make(chan struct{})}
	g.m[point] = x
	return x
}

func (g *gates) hook(point string) {
	g.mu.Lock()
	x := g.m[point]
	g.mu.Unlock()
	if x == nil {
		return
	}
	first := false
	x.once.Do(func() {
		first = true
		if x.do != nil {
			x.do()
		}
		close(x.reached)
	})
	if first {
		<-x.release
	}
}

func (g *gates) freeAll() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, x := range g.m {
		x.free()
	}
}

// verifier stands in for collect.VerifyProvenance: it records each call
// with the clock's reading, returns the real result plus extra problems,
// and when block is set blocks its first call until released. It calls no
// method of the job (copyState may hold the job's mutex around it).
type verifier struct {
	clk              *chain.FakeClock
	block            bool
	extra            []string
	entered, release chan struct{}
	ent, rel         sync.Once
	mu               sync.Mutex
	calls            []time.Time
}

func newVerifier(clk *chain.FakeClock, block bool, extra ...string) *verifier {
	return &verifier{clk: clk, block: block, extra: extra, entered: make(chan struct{}), release: make(chan struct{})}
}

func (v *verifier) free() { v.rel.Do(func() { close(v.release) }) }

func (v *verifier) verify(items []collect.Item) []string {
	v.mu.Lock()
	v.calls = append(v.calls, v.clk.Now())
	first := len(v.calls) == 1
	v.mu.Unlock()
	if v.block && first {
		v.ent.Do(func() { close(v.entered) })
		<-v.release
	}
	return append(collect.VerifyProvenance(items), v.extra...)
}

func (v *verifier) callTimes() []time.Time {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([]time.Time(nil), v.calls...)
}

// newJob148 is a job on the fixture repository with the fake clock, the
// in-memory record file system, the given controls and type checker, and
// the counting directory seams below a test directory.
func newJob148(t *testing.T, vs []controls.Variant, tc collect.TypeChecker) (*Job, *chain.FakeClock, *opSeams, *gates) {
	t.Helper()
	cfg, _ := newConfig(t, vs)
	clk := chain.NewFakeClock(time.Unix(1_000_000, 0))
	cfg.Clock, cfg.Start = clk, clk.Now()
	r, inv, guard, err := controls.FixtureRepo(filepath.Join(t.TempDir(), "fx"), runtime.GOOS, clk)
	if err != nil {
		t.Fatal(err)
	}
	cfg.R, cfg.Invoker, cfg.Guard, cfg.WriteFS = r, inv, guard, newMemFS()
	if tc != nil {
		cfg.TypeCheck = tc
	}
	j, err := Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ops := &opSeams{clk: clk, base: t.TempDir()}
	g := newGates()
	j.s.mkdirTemp, j.s.removeAll, j.s.hook = ops.mkdirTemp, ops.removeAll, g.hook
	return j, clk, ops, g
}

// settle releases every block and waits, bounded, for the work task to
// finish; registered after the temporary directories, it runs before they
// are removed.
func settle(t *testing.T, j *Job, frees ...func()) {
	t.Cleanup(func() {
		for _, f := range frees {
			f()
		}
		if ch := j.WorkDone(); ch != nil {
			select {
			case <-ch:
			case <-time.After(bound):
				t.Errorf("the work task did not finish within %s after the release", bound)
			}
		}
	})
}

func goRun(j *Job) <-chan struct{} {
	ch := make(chan struct{})
	go func() { j.RunDomains(); close(ch) }()
	return ch
}

func lastConfig() string {
	cs := spec.ConfigsFor(runtime.GOOS)
	return cs[len(cs)-1].String()
}

func lc1Of(in verdict.Inputs) *verdict.Variant {
	for i := range in.Variants {
		if in.Variants[i].Control == "LC-1" {
			return &in.Variants[i]
		}
	}
	return nil
}

func opsNamed(o chain.OuterSnapshot, name string) []chain.OpSnapshot {
	var out []chain.OpSnapshot
	for _, op := range o.Ops {
		if op.Name == name {
			out = append(out, op)
		}
	}
	return out
}

// ---- R-a: the source-chain verification is work; the parent only
// consumes its published state.

// R-a 1: the verification stays stuck; the parent still leaves at W and
// its closeout decides, without running the verification itself.
func TestB148RaStuckVerificationParentStillCloses(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	v := newVerifier(clk, true)
	j.s.verify = v.verify
	settle(t, j, v.free, g.freeAll)
	ran := goRun(j)
	select {
	case <-v.entered:
	case <-ran:
	case <-time.After(bound):
		t.Fatal("neither the verification was entered nor RunDomains returned")
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	done := make(chan chain.Final, 1)
	go func() { done <- j.Finish(0) }()
	select {
	case f := <-done:
		if !notValid(f) {
			t.Errorf("a stuck verification gave %v exit %d", f.Conclusion, f.ExitCode)
		}
	case <-time.After(bound):
		t.Errorf("Finish did not return within %s: the parent is running the verification", bound)
		v.free()
		waitFor(t, toStruct(done), "Finish after the release")
		return
	}
	if n := len(v.callTimes()); n != 1 {
		t.Errorf("verification calls: %d, want only the work task's one", n)
	}
}

// R-a 1 at D: the verification stays stuck and m.Run returns only after D;
// the expiry exit decides from the published state without calling the
// verifier again.
func TestB148RaStuckVerificationExpiredParent(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	v := newVerifier(clk, true)
	j.s.verify = v.verify
	settle(t, j, v.free, g.freeAll)
	ran := goRun(j)
	select {
	case <-v.entered:
	case <-ran:
	case <-time.After(bound):
		t.Fatal("neither the verification was entered nor RunDomains returned")
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	clk.Set(j.D)
	done := make(chan chain.Final, 1)
	go func() { done <- j.Finish(0) }()
	select {
	case f := <-done:
		if !notValid(f) || f.Path != chain.PathExpiredAtMRun {
			t.Errorf("final %v exit %d path %v", f.Conclusion, f.ExitCode, f.Path)
		}
	case <-time.After(bound):
		t.Errorf("Finish did not return within %s: the expiry exit is running the verification", bound)
		v.free()
		waitFor(t, toStruct(done), "Finish after the release")
		return
	}
	for _, at := range v.callTimes() {
		if !at.Before(j.D) {
			t.Errorf("the verifier was called at %v, at or after D", at)
		}
	}
}

func toStruct(ch <-chan chain.Final) <-chan struct{} {
	out := make(chan struct{})
	go func() { <-ch; close(out) }()
	return out
}

// R-a 2: the verification returns success only after W; a late success
// does not restore validity.
func TestB148RaLateVerificationSuccessDoesNotRestore(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	v := newVerifier(clk, true)
	j.s.verify = v.verify
	settle(t, j, v.free, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, v.entered, "the work task entered the verification") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	clk.Set(j.W.Add(time.Second))
	v.free()
	if !waitFor(t, j.WorkDone(), "the work task finished") {
		return
	}
	f, ok := finish(t, j)
	if !ok {
		return
	}
	if !notValid(f) {
		t.Fatalf("a late verification success gave %v exit %d", f.Conclusion, f.ExitCode)
	}
}

// R-a 3: a real verification error published after the parent stopped
// waiting, but before its snapshot, is consumed: the job is invalid.
func TestB148RaLateVerificationErrorIsConsumed(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	v := newVerifier(clk, true, "injected verification problem")
	j.s.verify = v.verify
	settle(t, j, v.free, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, v.entered, "the work task entered the verification") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	clk.Set(j.W.Add(time.Second))
	v.free()
	if !waitFor(t, j.WorkDone(), "the work task finished") {
		return
	}
	f, ok := finish(t, j)
	if !ok {
		return
	}
	if f.Conclusion != chain.Invalid {
		t.Fatalf("a real verification error gave %v %v", f.Conclusion, f.Reasons)
	}
	found := false
	for _, d := range j.copyState(0).in.Defects {
		found = found || strings.Contains(d, "injected verification problem")
	}
	if !found {
		t.Error("the late verification error is not among the defects")
	}
}

// R-a 4: healthy: every record is verified once, by the work task, before
// it finishes; the parent's closeout does not verify again.
func TestB148RaHealthyVerifiedOnceByTheWork(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	v := newVerifier(clk, false)
	j.s.verify = v.verify
	settle(t, j, g.freeAll)
	j.RunDomains()
	if n := len(v.callTimes()); n != 1 {
		t.Errorf("verification calls when the work task finished: %d, want 1", n)
	}
	f, ok := finish(t, j)
	if !ok {
		return
	}
	if f.Conclusion != chain.Consistent || f.ExitCode != 0 {
		t.Fatalf("healthy: %v exit %d %v", f.Conclusion, f.ExitCode, f.Reasons)
	}
	if n := len(v.callTimes()); n != 1 {
		t.Errorf("verification calls after Finish: %d, want 1 (the parent must not verify)", n)
	}
}

// R-a 5: m.Run returns after D; the expiry exit consumes the published
// state and never calls the verifier.
func TestB148RaExpiredParentDoesNotVerify(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	v := newVerifier(clk, false)
	j.s.verify = v.verify
	settle(t, j, g.freeAll)
	j.RunDomains()
	clk.Set(j.D)
	f, ok := finish(t, j)
	if !ok {
		return
	}
	if !notValid(f) || f.Path != chain.PathExpiredAtMRun {
		t.Errorf("final %v exit %d path %v", f.Conclusion, f.ExitCode, f.Path)
	}
	for _, at := range v.callTimes() {
		if !at.Before(j.D) {
			t.Errorf("the verifier was called at %v, at or after D", at)
		}
	}
}

// R-a (prefix rule's healthy counterpart): a broken record appended inside
// the work task before its verification is found by that verification.
func TestB148RaBrokenRecordBeforeTheVerification(t *testing.T) {
	j, _, _, g := newJob148(t, controls.LC3(), nil)
	x := g.add("R done")
	x.do = func() {
		j.rec.Derived(spec.ConfigsFor(runtime.GOOS)[0].String()+"/closure", "B3", nil, map[string]any{"closure": []string{"forged"}})
	}
	x.free()
	settle(t, j, g.freeAll)
	j.RunDomains()
	f, ok := finish(t, j)
	if !ok {
		return
	}
	if f.Conclusion != chain.Invalid {
		t.Fatalf("final %v %v", f.Conclusion, f.Reasons)
	}
}

// ---- R-b: the work task's abandonment and completion qualification are
// consumed, and LC-1 has its own.

// R-b window (Review 148 §3.3): the last R result is published before W;
// the work task pauses before its following publications; the parent stops
// waiting at W; the work task completes after W and before D; only then
// does the parent run its closeout. The completed work cannot turn the
// established failure into 3 or 4.
func TestB148RbAbandonedThenCompletedBeforeFinish(t *testing.T) {
	j, clk, _, g := newJob148(t, controls.LC3(), nil)
	last := g.add("R published " + lastConfig())
	settle(t, j, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, last.reached, "the last R result was published") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	clk.Set(j.W.Add(time.Second))
	last.free()
	if !waitFor(t, j.WorkDone(), "the work task completed after W") {
		return
	}
	s := j.Snapshot()
	if s.Work == nil || !s.Work.Abandoned || s.Work.State != WorkDone || s.Work.Timely {
		t.Fatalf("work %+v", s.Work)
	}
	for _, r := range s.RRecords {
		if !r.Timely {
			t.Fatalf("%s: the R results must be timely in this window", r.Result.Config)
		}
	}
	f, ok := finish(t, j)
	if !ok {
		return
	}
	if !notValid(f) {
		t.Errorf("completion after the parent stopped waiting gave %v exit %d", f.Conclusion, f.ExitCode)
	}
	st := j.copyState(0)
	if v := lc1Of(st.in); v == nil || v.Judgement == verdict.Hit {
		t.Errorf("LC-1 published after W must not be a timely hit: %+v", v)
	}
	if len(st.pending) == 0 {
		t.Error("the abandoned and late work task is not consumed")
	}
}

// R-b healthy branch: every necessary publication was made before W and only
// the parent is scheduled late, with the completion and W both readable.
// The repetitions only vary the scheduler's choice; each run asserts the
// same events.
func TestB148RbTimelyWorkParentLate(t *testing.T) {
	for i := 0; i < 10; i++ {
		j, clk, _, g := newJob148(t, controls.LC3(), nil)
		parent := g.add("parent wait")
		settle(t, j, g.freeAll)
		ran := goRun(j)
		if !waitFor(t, parent.reached, "the parent reached its wait") || !waitFor(t, j.WorkDone(), "the work task finished") {
			return
		}
		clk.Set(j.W)
		parent.free()
		if !waitFor(t, ran, "RunDomains returned") {
			return
		}
		s := j.Snapshot()
		if !s.Work.Timely || s.Work.Abandoned {
			t.Fatalf("run %d: work %+v", i, s.Work)
		}
		f, ok := finish(t, j)
		if !ok {
			return
		}
		if f.Conclusion != chain.Consistent || f.ExitCode != 0 {
			t.Fatalf("run %d: timely work with a late parent gave %v exit %d %v", i, f.Conclusion, f.ExitCode, f.Reasons)
		}
	}
}

// R-b: the work task publishes done exactly at W, or after it, while the
// parent is held; when the parent waits, the completion and W are both
// readable. Either choice of the scheduler gives a non-valid job.
func TestB148RbWorkDoneAtOrAfterW(t *testing.T) {
	for _, after := range []time.Duration{0, time.Second} {
		j, clk, _, g := newJob148(t, controls.LC3(), nil)
		parent := g.add("parent wait")
		before := g.add("before done")
		settle(t, j, g.freeAll)
		ran := goRun(j)
		if !waitFor(t, parent.reached, "the parent reached its wait") || !waitFor(t, before.reached, "the work task reached done") {
			return
		}
		clk.Set(j.W.Add(after))
		before.free()
		if !waitFor(t, j.WorkDone(), "the work task finished") {
			return
		}
		parent.free()
		if !waitFor(t, ran, "RunDomains returned") {
			return
		}
		if s := j.Snapshot(); s.Work.Timely {
			t.Fatalf("W+%v: done at or after W is not timely: %+v", after, s.Work)
		}
		f, ok := finish(t, j)
		if !ok {
			return
		}
		if !notValid(f) {
			t.Errorf("W+%v: the work task done at or after W gave %v exit %d", after, f.Conclusion, f.ExitCode)
		}
	}
}

// ---- R-c: the outer layer's own operations are admitted against the
// outer deadlines (preparation W, cleanup D) right before they start.

// R-c 1: the work task is registered but only scheduled after D: it creates
// no fixture root and starts no cleanup.
func TestB148RcLateWorkCreatesNoRoot(t *testing.T) {
	j, clk, ops, g := newJob148(t, controls.LC3(), nil)
	start := g.add("work start")
	settle(t, j, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, start.reached, "the work task was scheduled") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	clk.Set(j.D)
	start.free()
	if !waitFor(t, j.WorkDone(), "the work task finished") {
		return
	}
	if c := ops.list("mkdir"); len(c) != 0 {
		t.Errorf("directories created after D: %+v", c)
	}
	if c := ops.list("remove"); len(c) != 0 {
		t.Errorf("cleanups started after D: %+v", c)
	}
	if f, ok := finish(t, j); ok && !notValid(f) {
		t.Errorf("final %v exit %d", f.Conclusion, f.ExitCode)
	}
}

// lc5Then3 is LC5-0 (its analysis runs the job's type checker) followed by
// one LC-3 variant.
func lc5Then3() []controls.Variant {
	return []controls.Variant{controls.LC5()[0], controls.LC3()[0]}
}

// R-c 2: an LC-5 instance admitted before W returns only after D: it starts
// no cleanup of its fixture, no further instance is prepared, and the
// fixture root's deferred cleanup does not start either.
func TestB148RcAnalysisReturnsAfterD(t *testing.T) {
	tc := newBlockingTC()
	j, clk, ops, g := newJob148(t, lc5Then3(), tc)
	settle(t, j, tc.free, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, tc.entered, "the LC-5 instance entered the type check") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	clk.Set(j.D)
	if f, ok := finish(t, j); ok && !notValid(f) {
		t.Errorf("final %v exit %d", f.Conclusion, f.ExitCode)
	}
	tc.free()
	if !waitFor(t, j.WorkDone(), "the work task finished") {
		return
	}
	mk := ops.list("mkdir")
	if len(mk) != 2 {
		t.Errorf("directories created: %+v, want the fixture root and the LC-5 fixture only", mk)
	}
	if c := ops.list("remove"); len(c) != 0 {
		t.Errorf("cleanups started after D: %+v", c)
	}
	for _, c := range mk {
		if _, err := os.Stat(c.path); err != nil {
			t.Errorf("%s: %v (a directory whose cleanup was not admitted must still exist)", c.path, err)
		}
	}
}

// R-c 3: the same instance returns after W and before D: its cleanup and
// the root's are admitted and work as before; no further instance starts.
func TestB148RcCleanupBetweenWAndD(t *testing.T) {
	tc := newBlockingTC()
	j, clk, ops, g := newJob148(t, lc5Then3(), tc)
	settle(t, j, tc.free, g.freeAll)
	ran := goRun(j)
	if !waitFor(t, tc.entered, "the LC-5 instance entered the type check") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	clk.Set(j.W.Add(time.Second))
	tc.free()
	if !waitFor(t, j.WorkDone(), "the work task finished") {
		return
	}
	if mk := ops.list("mkdir"); len(mk) != 2 {
		t.Errorf("directories created: %+v, want the fixture root and the LC-5 fixture only", mk)
	}
	rm := ops.list("remove")
	if len(rm) != 2 {
		t.Fatalf("cleanups: %+v, want the LC-5 fixture's and the root's", rm)
	}
	for _, c := range rm {
		if c.err != nil || !c.at.Before(j.D) {
			t.Errorf("cleanup %+v", c)
		}
		if _, err := os.Stat(c.path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s still exists: %v", c.path, err)
		}
	}
	o := j.outer.Snapshot()
	for _, name := range []string{"cleanup LC-5 " + controls.LC5()[0].Name, "cleanup fixture root"} {
		if xs := opsNamed(o, name); len(xs) != 1 || xs[0].State != chain.OpDone {
			t.Errorf("%s: %+v", name, xs)
		}
	}
	if f, ok := finish(t, j); ok && !notValid(f) {
		t.Errorf("a late instance gave %v exit %d", f.Conclusion, f.ExitCode)
	}
}

// R-c 4 (failures): a fixture's creation and the root's removal fail; each
// is recorded as that operation's real failure.
func TestB148RcPrepareAndCleanupFailuresAreRecorded(t *testing.T) {
	vs := controls.LC3()
	j, _, ops, g := newJob148(t, vs, nil)
	ops.failFx, ops.failRoot = 2, true
	settle(t, j, g.freeAll)
	j.RunDomains()
	o := j.outer.Snapshot()
	prep := opsNamed(o, "prepare LC-3 "+vs[1].Name)
	if len(prep) != 1 || prep[0].State != chain.OpFailed || prep[0].Outcome == nil || prep[0].Outcome.Err == nil || prep[0].Outcome.Err.Error() != "injected mkdir failure" {
		t.Errorf("preparation: %+v", prep)
	}
	root := opsNamed(o, "cleanup fixture root")
	if len(root) != 1 || root[0].State != chain.OpFailed || root[0].Outcome == nil || root[0].Outcome.Err == nil || root[0].Outcome.Err.Error() != "injected remove failure" {
		t.Errorf("root cleanup: %+v", root)
	}
	for _, r := range j.Snapshot().FRecords {
		if r.Variant.Variant == vs[1].Name && (r.Outcome.PrepErr == nil || r.Variant.Judgement != verdict.Miss) {
			t.Errorf("%s: %+v", vs[1].Name, r)
		}
	}
	if f, ok := finish(t, j); ok && !notValid(f) {
		t.Errorf("final %v exit %d", f.Conclusion, f.ExitCode)
	}
}

// R-c 5 (healthy): every preparation and cleanup is done before W, and
// every directory created is removed.
func TestB148RcHealthyPrepareAndCleanup(t *testing.T) {
	j, _, ops, g := newJob148(t, controls.LC3(), nil)
	settle(t, j, g.freeAll)
	j.RunDomains()
	mk, rm := ops.list("mkdir"), ops.list("remove")
	if len(mk) != 5 || len(rm) != 5 {
		t.Fatalf("mkdir %d remove %d, want 5 and 5 (the root and four fixtures)", len(mk), len(rm))
	}
	for _, c := range append(mk, rm...) {
		if c.err != nil || !c.at.Before(j.W) {
			t.Errorf("%+v", c)
		}
	}
	for _, c := range mk {
		if _, err := os.Stat(c.path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s still exists: %v", c.path, err)
		}
	}
	for _, op := range j.outer.Snapshot().Ops {
		if op.State != chain.OpDone {
			t.Errorf("op %s: %v", op.Name, op.State)
		}
	}
	f, ok := finish(t, j)
	if ok && (f.Conclusion != chain.Consistent || f.ExitCode != 0) {
		t.Errorf("healthy: %v exit %d %v", f.Conclusion, f.ExitCode, f.Reasons)
	}
}
