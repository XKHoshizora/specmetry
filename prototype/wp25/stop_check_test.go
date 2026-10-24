package wp25

// Direct checks of the output step's stop (Review 081 B1), with a real
// os.Pipe, its real File.Close, and runS6On on a clock the test moves. The
// write end is the test's own and is never written, so no copier ends by
// itself. Stopping the read must not hold up phase J or the seal, whatever
// the copier's read is doing:
//
//   - its system read is already in progress (the control);
//   - it holds the file's read reference but has not made its system read
//     yet, and makes it while phase J runs;
//   - it is still there when phase J ends.
//
// The window is held with the file's RawConn: the read takes the same read
// reference File.Read takes, then waits at a gate the test opens, then
// makes the real system read. Nothing but this test's own pipe and tasks is
// touched, and every wait is bounded.

import (
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// gatedRead reads a real os.File through its RawConn, waiting at a gate
// between taking the file's read reference and its system read.
type gatedRead struct {
	rc      syscall.RawConn
	entered chan struct{} // closed once the read holds the reference
	gate    chan struct{}
	enter   sync.Once
	opened  sync.Once
}

func newGatedRead(f *os.File) (*gatedRead, error) {
	rc, err := f.SyscallConn()
	if err != nil {
		return nil, err
	}
	return &gatedRead{rc: rc, entered: make(chan struct{}), gate: make(chan struct{})}, nil
}

func (g *gatedRead) Read(p []byte) (int, error) {
	var n int
	var rerr error
	err := g.rc.Read(func(fd uintptr) bool {
		g.enter.Do(func() { close(g.entered) })
		<-g.gate
		var again bool
		n, rerr, again = sysRead(fd, p)
		return !again
	})
	if err != nil {
		return 0, err
	}
	return n, rerr
}

func (g *gatedRead) open() { g.opened.Do(func() { close(g.gate) }) }

// stopRig is one stream on a real pipe, its copier, and runS6On on a fake
// clock, phase D skipped (the F1 path, the stop at once).
type stopRig struct {
	t    *testing.T
	clk  *fakeClock
	d    *dest
	ss   []*stream
	w    *os.File
	g    *gatedRead // nil: the copier reads the file itself
	res  chan s6Result
	got  *s6Result
	once sync.Once
}

func newStopRig(t *testing.T, gated bool) *stopRig {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	rig := &stopRig{t: t, clk: newFakeClock(), d: &dest{}, w: w, res: make(chan s6Result, 1)}
	var src io.Reader = r
	if gated {
		if rig.g, err = newGatedRead(r); err != nil {
			t.Fatal(err)
		}
		src = rig.g
	}
	p := rig.d.copyPort("out")
	rig.ss = []*stream{{name: "out", r: r, c: startCopy(src, p, false), p: p}}
	t.Cleanup(rig.release)
	return rig
}

func (g *stopRig) start() {
	go func() { g.res <- runS6On(g.clk, g.d, g.ss, tDrain, tJoin, true) }()
}

// result waits for runS6On to return, within d of real time.
func (g *stopRig) result(d time.Duration) (s6Result, bool) {
	if g.got != nil {
		return *g.got, true
	}
	select {
	case r := <-g.res:
		g.got = &r
		return r, true
	case <-time.After(d):
		return s6Result{}, false
	}
}

// release ends what the test started: the gate opens, the write end closes
// (EOF for a read in progress), the clock moves past every deadline; then
// runS6On and the copier must have returned.
func (g *stopRig) release() {
	g.once.Do(func() {
		if g.g != nil {
			g.g.open()
		}
		_ = g.w.Close()
		for i := 0; i < 10; i++ {
			if _, ok := g.result(200 * time.Millisecond); ok {
				break
			}
			g.clk.advance(tJoin)
		}
		if _, ok := g.result(0); !ok {
			g.t.Errorf("cleanup: runS6On did not return")
		}
		if !g.ss[0].c.wait(time.Now().Add(5 * time.Second)) {
			g.t.Errorf("cleanup: the copier did not return")
		}
	})
}

// awaitAsked waits, within 5 s of real time, for the next wake asked of the
// fake clock: phase J's, once it has started.
func awaitAsked(clk *fakeClock) (time.Duration, bool) {
	select {
	case d := <-clk.asked:
		return d, true
	case <-time.After(5 * time.Second):
		return 0, false
	}
}

// goroutineWith reports whether one goroutine's stack has every frame.
func goroutineWith(frames ...string) bool {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	for _, g := range strings.Split(string(buf[:n]), "\n\n") {
		all := true
		for _, f := range frames {
			if !strings.Contains(g, f) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

// awaitGoroutine waits, within 5 s, for a goroutine with every frame.
func awaitGoroutine(frames ...string) bool {
	deadline := time.Now().Add(5 * time.Second)
	for {
		if goroutineWith(frames...) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Millisecond)
	}
}

// checkSealed checks that the run's copy port refuses writes.
func checkSealed(t *testing.T, g *stopRig) {
	t.Helper()
	if _, err := g.ss[0].p.Write([]byte("late\n")); !errors.Is(err, errSealed) {
		t.Errorf("the copy port takes writes after the run returned (%v)", err)
	}
}

func TestStopRead(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	t.Run("read-in-system-call", func(t *testing.T) {
		// The control: the copier's system read is in progress when the
		// stop comes; closing the file cancels it.
		g := newStopRig(t, false)
		if !awaitGoroutine(append([]string{"wp25.startCopy"}, inSystemRead...)...) {
			t.Fatal("the copier never reached its system read within 5 s")
		}
		g.start()
		// The copier may return before phase J waits for it, so no wake
		// need be asked: the run must return with the clock not moved.
		r, ok := g.result(5 * time.Second)
		if !ok {
			t.Fatal("the stop did not complete within 5 s, the clock not moved")
		}
		if r.stop["out"] != "done" || r.out["out"] != o2 || r.join != 0 || len(r.left) != 0 {
			t.Errorf("stop %q, outcome %s, phase J %v, %d left; want done, O2, 0, none", r.stop["out"], r.out["out"], r.join, len(r.left))
		}
		if end := endOf(g.ss[0]); end != endRead {
			t.Errorf("the copier ended by %s, want a read error", end)
		}
		checkSealed(t, g)
	})
	t.Run("window-released-in-J", func(t *testing.T) {
		// The read holds the file's read reference when the stop comes; it
		// makes its system read while phase J runs.
		g := newStopRig(t, true)
		select {
		case <-g.g.entered:
		case <-time.After(5 * time.Second):
			t.Fatal("the copier never took its read reference within 5 s")
		}
		g.start()
		if d, ok := awaitAsked(g.clk); !ok || d != tJoin {
			t.Fatalf("phase J did not start while the read held its reference: the stop held it up (wake %v, asked %v)", d, ok)
		}
		if !awaitGoroutine("internal/poll.(*FD).Close") {
			t.Fatal("no Close was seen waiting for the read's reference within 5 s")
		}
		g.g.open()
		r, ok := g.result(5 * time.Second)
		if !ok {
			t.Fatal("the stop did not complete within 5 s after the read made its system read, the clock not moved")
		}
		if r.stop["out"] != "done" || r.out["out"] != o2 || r.join != 0 || len(r.left) != 0 {
			t.Errorf("stop %q, outcome %s, phase J %v, %d left; want done, O2, 0, none", r.stop["out"], r.out["out"], r.join, len(r.left))
		}
		checkSealed(t, g)
	})
	t.Run("window-held-past-J", func(t *testing.T) {
		// The read is still in its window when phase J ends: the run must
		// still seal, report O3 and the stop pending, and hand both on.
		g := newStopRig(t, true)
		select {
		case <-g.g.entered:
		case <-time.After(5 * time.Second):
			t.Fatal("the copier never took its read reference within 5 s")
		}
		g.start()
		if d, ok := awaitAsked(g.clk); !ok || d != tJoin {
			t.Fatalf("phase J did not start while the read held its reference: the stop held it up (wake %v, asked %v)", d, ok)
		}
		if !awaitGoroutine("internal/poll.(*FD).Close") {
			t.Fatal("no Close was seen waiting for the read's reference within 5 s")
		}
		g.clk.advance(tJoin)
		r, ok := g.result(5 * time.Second)
		if !ok {
			t.Fatal("runS6On did not return at the end of phase J")
		}
		if r.stop["out"] != "pending" || r.out["out"] != o3 || r.join != tJoin || len(r.left) != 2 {
			t.Errorf("stop %q, outcome %s, phase J %v, %d left; want pending, O3, %v, the stop and the copier", r.stop["out"], r.out["out"], r.join, len(r.left), tJoin)
		}
		checkSealed(t, g)
		// Whether a later run may start meanwhile is C-4 / D20; both
		// policies see what is left.
		reg := &lingering{}
		reg.add(g.ss, r)
		reg.refuse = true
		if !errors.Is(reg.admit(), errLingering) {
			t.Error("policy refuse: a later run was admitted while the stop was pending")
		}
		reg.refuse = false
		if reg.admit() != nil {
			t.Error("policy allow: a later run was refused")
		}
		// The read leaves its window; nothing else is done, no byte is
		// written: what was left must end by itself.
		g.g.open()
		for _, l := range r.left {
			if !l.until(time.Now().Add(5 * time.Second)) {
				t.Errorf("%s did not end within 5 s after the read left its window", l.what)
			}
		}
		if n := reg.pending(); n != 0 {
			t.Errorf("%d still pending in the register", n)
		}
	})
}
