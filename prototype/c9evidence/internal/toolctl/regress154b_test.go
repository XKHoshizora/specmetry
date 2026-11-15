package toolctl

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/pseudo"
)

// The Review 154 regressions that need the new seams or the new observable
// state (review-resolution §202: R1-d, R1-e, R2-d, R2-e, R2-f). They never
// ran on the old implementation.

// TestR154AdmitDelayedStartTask is R1-d: the start task is scheduled only
// after W (the hook at its beginning holds it); it is refused before the
// first pipe, so nothing is acquired or started.
func TestR154AdmitDelayedStartTask(t *testing.T) {
	clk := chain.NewFakeClock(t154)
	w := t154.Add(100 * time.Second)
	held := newLatch()
	t.Cleanup(held.release)
	var mu sync.Mutex
	pipes := 0
	reg := New(Options{Clock: clk, W: w, D: w.Add(10 * time.Second), Seams: &Seams{
		Hook: func(point string, _ int) {
			if point == HookStartTask {
				held.wait()
			}
		},
		Pipe: func(_, _ int) (*os.File, *os.File, error) {
			mu.Lock()
			pipes++
			mu.Unlock()
			return os.Pipe()
		},
		Prepare: func(_ int, cmd *exec.Cmd) { silentPrepare(cmd) },
	}})
	ch := invokeAsync(reg, pseudoSpec(t))
	held.waitEntered(t, "the start task")
	clk.Set(w)
	waitResult(t, ch, "R1-d")
	held.release()
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	s, _ := reg.Snapshot(1)
	noStart(t, s)
	mu.Lock()
	n := pipes
	mu.Unlock()
	if n != 0 {
		t.Errorf("%d pipes built after W", n)
	}
	wantRefusal(t, s, PointStdout, nil, []string{PointStdout, PointStderr, PointStart})
	if !s.Expired {
		t.Errorf("the call reached W unfinished: expired %v", s.Expired)
	}
}

// TestR154AdmittedStartReturnsLate is R1-e: Start was admitted and
// initiated before W and returns after it. It is neither refused nor
// undone; its result is late; the termination then follows the platform
// (Linux: through the pidfd; Windows: Kill; darwin: none, the cleanup is
// unknown and the test releases the pseudo-tool's stdin).
func TestR154AdmittedStartReturnsLate(t *testing.T) {
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	var inOnce sync.Once
	releaseIn := func() { inOnce.Do(func() { inW.Close() }) }
	t.Cleanup(func() { releaseIn(); inR.Close() })
	clk := chain.NewFakeClock(t154)
	w := t154.Add(100 * time.Second)
	held := newLatch()
	t.Cleanup(held.release)
	reg := New(Options{Clock: clk, W: w, D: w.Add(10 * time.Second), Seams: &Seams{
		Prepare: func(_ int, cmd *exec.Cmd) {
			cmd.Env = append(cmd.Env, pseudo.EnvMode+"="+pseudo.Block)
			cmd.Stdin = inR
		},
		Start: func(_ int, cmd *exec.Cmd) error {
			err := cmd.Start()
			held.wait()
			return err
		},
	}})
	ch := invokeAsync(reg, pseudoSpec(t))
	held.waitEntered(t, "Start")
	clk.Set(w)
	waitResult(t, ch, "R1-e")
	held.release()
	if !reg.WaitFor(1, after(guard), func(s Snapshot) bool { return s.State == StStarted || s.State == StWaited }) {
		t.Fatal("the late Start was not published")
	}
	if reg.Mode() == ModeNone {
		releaseIn()
	}
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	releaseIn()
	s, _ := reg.Snapshot(1)
	for _, a := range s.Admissions {
		if !a.Admitted {
			t.Errorf("admission %s refused", a.Point)
		}
	}
	if len(s.Admissions) != 4 || len(s.Skipped) != 0 || !s.StartCalled || !s.StartReturned || s.StartErr != "" || !lateEvent(s, "started") ||
		s.State != StWaited || s.Class != "B" || !s.Expired || s.Success {
		t.Errorf("lines %q", Lines(s))
	}
	switch reg.Mode() {
	case ModePidfd, ModeKill:
		if s.Req.State != ReqDone || s.Req.Sends != 1 || s.Req.Err != "" || s.Exit == nil || s.Exit.Code == 0 {
			t.Errorf("request %+v exit %+v", s.Req, s.Exit)
		}
	default:
		if s.Req.State != ReqNone || len(s.Unknown) != 1 || s.Exit == nil || s.Exit.Code != 0 {
			t.Errorf("request %+v unknown %q exit %+v", s.Req, s.Unknown, s.Exit)
		}
	}
}

var errSim = errors.New("r154: a close error the seam stands in for")

// TestR154ObjectsHeld is R2-d: the record holds the very objects the pipes
// returned. A close error leaves the release unknown, the object held and
// the close not retried; in the half-built case and after a normal read.
func TestR154ObjectsHeld(t *testing.T) {
	for _, half := range []bool{true, false} {
		name := "after-the-read"
		if half {
			name = "half-built"
		}
		t.Run(name, func(t *testing.T) {
			now := time.Now()
			var mu sync.Mutex
			var got [4]*os.File
			closes := map[int]int{}
			reg := New(Options{Clock: chain.RealClock{}, W: now.Add(time.Hour), D: now.Add(time.Hour + 10*time.Second), Seams: &Seams{
				Pipe: func(_, n int) (*os.File, *os.File, error) {
					if n == 1 && half {
						return nil, nil, errSim
					}
					r, w, err := os.Pipe()
					mu.Lock()
					got[2*n+1], got[2*n] = r, w
					mu.Unlock()
					return r, w, err
				},
				CloseFile: func(_, d int, f *os.File) error {
					mu.Lock()
					closes[d]++
					mu.Unlock()
					err := f.Close()
					if d == StdoutR && err == nil {
						return errSim
					}
					return err
				},
				Prepare: func(_ int, cmd *exec.Cmd) { silentPrepare(cmd) },
			}})
			waitResult(t, invokeAsync(reg, pseudoSpec(t)), name)
			if !reg.WaitTerminal(1, after(guard)) {
				t.Fatal("not terminal")
			}
			s, _ := reg.Snapshot(1)
			reg.mu.Lock()
			files := reg.recs[0].files
			reg.mu.Unlock()
			mu.Lock()
			defer mu.Unlock()
			ds := []int{StdoutR, StdoutW}
			if !half {
				ds = append(ds, StderrR, StderrW)
			}
			for _, d := range ds {
				if files[d] == nil || files[d] != got[d] || !s.Desc[d].Held || closes[d] != 1 {
					t.Errorf("%s: held %v same object %v closes %d", DescNames[d], s.Desc[d].Held, files[d] == got[d], closes[d])
				}
			}
			if s.Desc[StdoutR].Release != ReleaseUnknown || s.Desc[StdoutR].CloseErr == "" {
				t.Errorf("stdout-r: release %s err %q", s.Desc[StdoutR].Release, s.Desc[StdoutR].CloseErr)
			}
			released(t, s, StdoutW)
			var sites []string
			for _, f := range s.Faults {
				sites = append(sites, f.Site+" "+f.Source)
			}
			want := "close stdout-r operation"
			if half {
				want = "pipe stderr operation;" + want
			}
			if strings.Join(sites, ";") != want || s.Success {
				t.Errorf("faults %q, want %q", sites, want)
			}
		})
	}
}

// TestR154PanicInStart is R2-e: Start panics (the seam panics without
// starting anything): class C, std-panic; the Cmd is held; the write ends
// are closed by the panic exit; the read tasks reach EOF.
func TestR154PanicInStart(t *testing.T) {
	now := time.Now()
	reg := New(Options{Clock: chain.RealClock{}, W: now.Add(time.Hour), D: now.Add(time.Hour + 10*time.Second), Seams: &Seams{
		Start: func(int, *exec.Cmd) error { panic("r154: a panic in Start") },
	}})
	waitResult(t, invokeAsync(reg, pseudoSpec(t)), "R2-e")
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	s, _ := reg.Snapshot(1)
	wantPanic(t, s, "C", "std-panic")
	released(t, s, StdoutW, StderrW, StdoutR, StderrR)
	if !s.StartCalled || s.StartReturned || !s.CmdHeld || s.ProcessHeld || len(s.Unknown) != 1 || s.WaitCalled {
		t.Errorf("lines %q", Lines(s))
	}
}

// TestR154PanicBeforeWait is R2-f: the start task panics after a
// successful Start and before its one Wait. The reap is unknown; the
// process, the Cmd and (Linux) the open pidfd are held; no Wait and no
// termination are added. The pseudo-tool exits by itself, so the read
// tasks reach EOF.
func TestR154PanicBeforeWait(t *testing.T) {
	now := time.Now()
	reg := New(Options{Clock: chain.RealClock{}, W: now.Add(time.Hour), D: now.Add(time.Hour + 10*time.Second), Seams: &Seams{
		Prepare: func(_ int, cmd *exec.Cmd) { silentPrepare(cmd) },
		Hook: func(point string, _ int) {
			if point == HookBeforeWait {
				panic("r154: a panic before Wait")
			}
		},
	}})
	waitResult(t, invokeAsync(reg, pseudoSpec(t)), "R2-f")
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal")
	}
	s, _ := reg.Snapshot(1)
	wantPanic(t, s, "C", "panic-before-wait")
	released(t, s, StdoutW, StderrW, StdoutR, StderrR)
	if s.State != StStarted || s.WaitCalled || !s.ProcessHeld || !s.CmdHeld || s.Pid == 0 || len(s.Unknown) != 1 {
		t.Errorf("lines %q", Lines(s))
	}
	if reg.Mode() == ModePidfd && (s.Pidfd.State != PidfdOpen || s.Pidfd.CloseCalled || s.Pidfd.Closes != 0) {
		t.Errorf("pidfd %+v", s.Pidfd)
	}
}
