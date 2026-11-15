package toolctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/pseudo"
)

// The regressions of Review 154 (review-resolution §202, table "回归与预期").
// R1-a, R1-b, R1-c, R2-a, R2-b and R2-c are written against the fields the
// old implementation (8e228f8) already has, so they also run on it, where
// they are expected to fail.

var t154 = time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)

// latch is a one-shot gate a seam can block in.
type latch struct {
	entered, open chan struct{}
	eo, oo        sync.Once
}

func newLatch() *latch { return &latch{entered: make(chan struct{}), open: make(chan struct{})} }

func (l *latch) wait() {
	l.eo.Do(func() { close(l.entered) })
	<-l.open
}

func (l *latch) release() { l.oo.Do(func() { close(l.open) }) }

func (l *latch) waitEntered(t *testing.T, what string) {
	t.Helper()
	select {
	case <-l.entered:
	case <-time.After(guard):
		t.Fatalf("%s: not entered within the guard", what)
	}
}

// invokeAsync runs Invoke on its own goroutine; the result channel has
// room for it.
func invokeAsync(reg *Registry, cs collect.CallSpec) <-chan collect.CallResult {
	ch := make(chan collect.CallResult, 1)
	go func() { ch <- reg.Invoke(cs) }()
	return ch
}

func waitResult(t *testing.T, ch <-chan collect.CallResult, what string) collect.CallResult {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(guard):
		t.Fatalf("%s: Invoke did not return within the guard", what)
	}
	return collect.CallResult{}
}

func silentPrepare(cmd *exec.Cmd) { cmd.Env = append(cmd.Env, pseudo.EnvMode+"="+pseudo.Silent) }

// noStart checks that no process was started for the call.
func noStart(t *testing.T, s Snapshot) {
	t.Helper()
	if s.Pid != 0 || s.WaitCalled || s.State == StStarted || s.State == StWaited {
		t.Errorf("a process was started: state %s pid %d wait-called %v; lines %q", s.State, s.Pid, s.WaitCalled, Lines(s))
	}
}

func closedOK(t *testing.T, s Snapshot, ds ...int) {
	t.Helper()
	for _, d := range ds {
		if s.Desc[d].State != DescClosed || s.Desc[d].CloseErr != "" {
			t.Errorf("%s: %s %q, want closed with a nil result; lines %q", DescNames[d], s.Desc[d].State, s.Desc[d].CloseErr, Lines(s))
		}
	}
}

// TestR154AdmitEntry is R1-a: a call that reaches the real entry with less
// than the call margin before W starts nothing.
func TestR154AdmitEntry(t *testing.T) {
	clk := chain.NewFakeClock(t154)
	w := t154.Add(4 * time.Second)
	var mu sync.Mutex
	pipes := 0
	reg := New(Options{Clock: clk, W: w, D: w.Add(10 * time.Second), Seams: &Seams{
		Pipe: func(_, _ int) (*os.File, *os.File, error) {
			mu.Lock()
			pipes++
			mu.Unlock()
			return os.Pipe()
		},
		Prepare: func(_ int, cmd *exec.Cmd) { silentPrepare(cmd) },
	}})
	r := waitResult(t, invokeAsync(reg, pseudoSpec(t)), "R1-a")
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	s, _ := reg.Snapshot(1)
	noStart(t, s)
	mu.Lock()
	n := pipes
	mu.Unlock()
	if r.Started || r.Succeeded(w) || n != 0 {
		t.Errorf("started %v succeeded %v pipes %d; lines %q", r.Started, r.Succeeded(w), n, Lines(s))
	}
}

// TestR154AdmitPipeCrossesD is R1-b: the first pipe's call returns after D;
// the start task does not build the second pipe or Start, and closes the
// first pipe's two ends once each.
func TestR154AdmitPipeCrossesD(t *testing.T) {
	clk := chain.NewFakeClock(t154)
	w := t154.Add(100 * time.Second)
	d := w.Add(10 * time.Second)
	first := newLatch()
	t.Cleanup(first.release)
	var mu sync.Mutex
	second := 0
	reg := New(Options{Clock: clk, W: w, D: d, Seams: &Seams{
		Pipe: func(_, n int) (*os.File, *os.File, error) {
			if n == 0 {
				r, wf, err := os.Pipe()
				first.wait()
				return r, wf, err
			}
			mu.Lock()
			second++
			mu.Unlock()
			return os.Pipe()
		},
		Prepare: func(_ int, cmd *exec.Cmd) { silentPrepare(cmd) },
	}})
	ch := invokeAsync(reg, pseudoSpec(t))
	first.waitEntered(t, "the first pipe")
	clk.Set(w)
	waitResult(t, ch, "R1-b")
	clk.Set(d.Add(time.Second))
	first.release()
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	s, _ := reg.Snapshot(1)
	noStart(t, s)
	mu.Lock()
	n := second
	mu.Unlock()
	if n != 0 {
		t.Errorf("the second pipe was built %d times after D", n)
	}
	closedOK(t, s, StdoutR, StdoutW)
}

// TestR154AdmitStartNotInitiated is R1-c: the Cmd is complete when the
// clock has passed W; Start is not initiated; the start task closes the
// write ends and the read tasks reach EOF and close the read ends.
func TestR154AdmitStartNotInitiated(t *testing.T) {
	clk := chain.NewFakeClock(t154)
	w := t154.Add(100 * time.Second)
	held := newLatch()
	t.Cleanup(held.release)
	reg := New(Options{Clock: clk, W: w, D: w.Add(10 * time.Second), Seams: &Seams{
		Prepare: func(_ int, cmd *exec.Cmd) {
			silentPrepare(cmd)
			held.wait()
		},
	}})
	ch := invokeAsync(reg, pseudoSpec(t))
	held.waitEntered(t, "Prepare")
	clk.Set(w)
	waitResult(t, ch, "R1-c")
	held.release()
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	s, _ := reg.Snapshot(1)
	noStart(t, s)
	closedOK(t, s, StdoutW, StderrW, StdoutR, StderrR)
	for i, o := range s.Out {
		if !o.Done || !o.EOF {
			t.Errorf("%s: done %v eof %v", StreamNames[i], o.Done, o.EOF)
		}
	}
}

// TestR154CmdHeldAfterStartFailure is R2-a: a start failure of class C
// keeps this call's Cmd in the record.
func TestR154CmdHeldAfterStartFailure(t *testing.T) {
	now := time.Now()
	reg := New(Options{Clock: chain.RealClock{}, W: now.Add(time.Hour), D: now.Add(time.Hour + 10*time.Second)})
	cs := pseudoSpec(t)
	cs.Exe = filepath.Join(cs.Dir, "absent", "tool.exe")
	reg.Invoke(cs)
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	s, _ := reg.Snapshot(1)
	reg.mu.Lock()
	cmd := reg.recs[0].cmd
	reg.mu.Unlock()
	if s.Class != "C" || cmd == nil || cmd.Path != cs.Exe || cmd.Process != nil {
		t.Errorf("class %s; cmd held %v; lines %q", s.Class, cmd != nil, Lines(s))
	}
}

// TestR154PanicAfterHandover is R2-b: a panic in the start task after the
// hand-over and before any start function. The read tasks hold the read
// ends; the write ends must still be closed, so the call ends.
func TestR154PanicAfterHandover(t *testing.T) {
	now := time.Now()
	reg := New(Options{Clock: chain.RealClock{}, W: now.Add(time.Hour), D: now.Add(time.Hour + 10*time.Second), Seams: &Seams{
		Prepare: func(int, *exec.Cmd) { panic("r154: a panic before Start") },
	}})
	waitResult(t, invokeAsync(reg, pseudoSpec(t)), "R2-b")
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	s, _ := reg.Snapshot(1)
	noStart(t, s)
	closedOK(t, s, StdoutW, StderrW, StdoutR, StderrR)
	if s.Success {
		t.Error("a panic must not be a success")
	}
}

// TestR154PanicBetweenPipes is R2-c: a panic while the second pipe is
// built; the first pipe's two ends are closed once each.
func TestR154PanicBetweenPipes(t *testing.T) {
	now := time.Now()
	reg := New(Options{Clock: chain.RealClock{}, W: now.Add(time.Hour), D: now.Add(time.Hour + 10*time.Second), Seams: &Seams{
		Pipe: func(_, n int) (*os.File, *os.File, error) {
			if n == 1 {
				panic("r154: a panic in the second pipe")
			}
			return os.Pipe()
		},
	}})
	waitResult(t, invokeAsync(reg, pseudoSpec(t)), "R2-c")
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	s, _ := reg.Snapshot(1)
	noStart(t, s)
	closedOK(t, s, StdoutR, StdoutW)
	if s.Success {
		t.Error("a panic must not be a success")
	}
}
