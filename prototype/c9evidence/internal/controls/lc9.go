package controls

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/pseudo"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/toolctl"
)

// LC-9 is the control-protocol control (plan §5.2 LC-9; implementation
// plan §12.12 "工具的控制协议" and "输出不关闭的拒绝路径", §12.14 items 3
// to 13; phase 2, review-resolution §200). Each variant runs one call of
// the real control protocol (toolctl) on a pseudo-tool, a re-execution of
// the test binary (internal/pseudo), with its own registry and its own fake
// clock, so its W and D are the case's, not the job's. The seams fix the
// interleavings and inject; the process start, the reads, the reap and the
// termination are real. The outer layer saves the decision Invoke returned,
// then releases what the case holds (the pseudo-tool's stdin, the gates, the
// holder H) and judges only once the call is terminal; the complete
// expectations below are written independently of toolctl.

// lc9Guard bounds each real-time wait of the outer layer. It is a backstop
// against a defect, not a budget: a case that hits it is not terminal or
// does not match, never a hit.
const lc9Guard = 20 * time.Second

var (
	errSeamPipe   = errors.New("seam: pipe failure")
	errSeamRead   = errors.New("seam: read failure")
	errSeamClose  = errors.New("seam: real close failure")
	errInjected   = errors.New("injected close error")
	errSeamPidfd  = errors.New("seam: pidfd_open failure")
	errNoExecPath = errors.New("os.Executable failed")
)

type gate struct {
	entered, open chan struct{}
	eo, oo        sync.Once
}

func newGate() *gate { return &gate{entered: make(chan struct{}), open: make(chan struct{})} }

func (g *gate) wait() {
	g.eo.Do(func() { close(g.entered) })
	<-g.open
}

func (g *gate) release() { g.oo.Do(func() { close(g.open) }) }

// holder is the controlled output holder H (§12.12): the test's own direct
// child, never the tool's descendant.
type holder struct {
	mu                     sync.Mutex
	relW                   *os.File
	started, ready, waited bool
	closed                 bool // the outer cleanup ran: H is never started after it
	exit                   int
	errs                   []string
	done                   chan struct{} // closed when H's one Wait returned
}

func (h *holder) err(s string) {
	h.mu.Lock()
	h.errs = append(h.errs, s)
	h.mu.Unlock()
}

type lc9Run struct {
	e        *Env
	k        lc9Case
	clock    *chain.FakeClock
	W, D     time.Time
	reg      *toolctl.Registry
	gates    map[string]*gate
	stdinR   *os.File
	stdinW   *os.File
	inOnce   sync.Once
	inErr    error
	returned chan struct{}
	res      collect.CallResult
	extra    []string
	problems []string
	prepErr  error
	h        *holder
	fx       *fixture
	mu       sync.Mutex
}

type lc9Case struct {
	name    string
	linux   bool // only on Linux: the pidfd exists only there (§12.14 item 8)
	mode    string
	stdin   bool
	limit   int64
	bytes   int
	gates   []string
	hooks   map[string]string // hook point -> gate
	holder  bool              // the case starts the output holder H
	absent  bool              // start an absolute path that does not exist
	badExe  bool              // start an existing file that is not an executable
	seams   func(c *lc9Run, s *toolctl.Seams)
	drive   func(c *lc9Run)
	exitAny bool // the exit status is recorded, not compared
	want    func(goos string) []string
}

func (c *lc9Run) problem(s string) {
	c.mu.Lock()
	c.problems = append(c.problems, s)
	c.mu.Unlock()
}

func (c *lc9Run) note(s string) { c.extra = append(c.extra, s) }

func guardCh() (<-chan struct{}, func()) {
	ch := make(chan struct{})
	t := time.AfterFunc(lc9Guard, func() { close(ch) })
	return ch, func() { t.Stop() }
}

// until waits on the call's published copy.
func (c *lc9Run) until(what string, cond func(toolctl.Snapshot) bool) bool {
	ch, stop := guardCh()
	defer stop()
	if !c.reg.WaitFor(1, ch, cond) {
		c.problem("timeout waiting for " + what)
		return false
	}
	return true
}

func (c *lc9Run) snap() toolctl.Snapshot {
	s, _ := c.reg.Snapshot(1)
	return s
}

func (c *lc9Run) waitReturned() bool {
	ch, stop := guardCh()
	defer stop()
	select {
	case <-c.returned:
		return true
	case <-ch:
		c.problem("timeout waiting for Invoke to return")
		return false
	}
}

func (c *lc9Run) entered(name string) bool {
	ch, stop := guardCh()
	defer stop()
	select {
	case <-c.gates[name].entered:
		return true
	case <-ch:
		c.problem("timeout waiting for the gate " + name)
		return false
	}
}

func (c *lc9Run) release(name string) { c.gates[name].release() }

// releaseStdin closes the pseudo-tool's stdin write end once: the
// pseudo-tool in the block mode then reads EOF and exits 0.
func (c *lc9Run) releaseStdin() error {
	c.inOnce.Do(func() {
		if c.stdinW == nil {
			return
		}
		c.inErr = cleanupStep(c.e, "LC-9 "+c.k.name+": close the pseudo-tool's stdin", c.stdinW.Close)
	})
	return c.inErr
}

// startedPidfd: Start published and, on Linux, the pidfd open.
func startedPidfd(s toolctl.Snapshot) bool {
	return s.State == toolctl.StStarted && (s.Mode != toolctl.ModePidfd || s.Pidfd.State == toolctl.PidfdOpen)
}

func cleanupStep(e *Env, op string, run func() error) error {
	done, ok := e.Outer.Cleanup(op, "")
	if !ok {
		return fmt.Errorf("%w (%s)", errCleanupNotAdmitted, op)
	}
	err := run()
	done(err)
	return err
}

func (k lc9Case) variant(goos string) Variant {
	applies := always
	if k.linux {
		applies = func(goos string) (bool, string) {
			if goos == "linux" {
				return true, ""
			}
			return false, "the pidfd exists only on Linux (§12.14 item 8)"
		}
	}
	v := Variant{Control: "LC-9", Name: k.name, Applies: applies, MustEnter: []string{"toolctl registration"}}
	v.Want = k.want(goos)
	v.Run = k.run
	return v
}

func (k lc9Case) run(e *Env) Outcome {
	exe, err := os.Executable()
	if err != nil {
		return Outcome{PrepErr: fmt.Errorf("%w: %w", errNoExecPath, err)}
	}
	t0 := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	c := &lc9Run{e: e, k: k, clock: chain.NewFakeClock(t0), W: t0.Add(100 * time.Second), D: t0.Add(110 * time.Second),
		gates: map[string]*gate{}, returned: make(chan struct{})}
	for _, g := range k.gates {
		c.gates[g] = newGate()
	}
	if k.holder {
		c.h = &holder{done: make(chan struct{}), exit: -2}
	}
	if k.stdin {
		if err := step(e, "prepare LC-9 "+k.name+": stdin pipe", func() error {
			r, w, err := os.Pipe()
			c.stdinR, c.stdinW = r, w
			return err
		}); err != nil {
			return Outcome{PrepErr: err}
		}
	}
	if k.badExe {
		name := "tool"
		if runtime.GOOS == "windows" {
			name = "tool.exe"
		}
		fx, err := prepare(e, "LC-9 "+k.name, map[string]string{name: "not an executable\n"})
		if err != nil {
			c.closeStdinR()
			return Outcome{PrepErr: err}
		}
		c.fx, exe = fx, filepath.Join(fx.dir, name)
	}
	if k.absent {
		name := "no-such-tool"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		exe = filepath.Join(e.TempRoot, "lc9-absent", name)
	}
	s := &toolctl.Seams{Prepare: func(_ int, cmd *exec.Cmd) {
		cmd.Env = append(cmd.Env, pseudo.EnvMode+"="+k.mode, pseudo.EnvBytes+"="+strconv.Itoa(k.bytes))
		if c.stdinR != nil {
			cmd.Stdin = c.stdinR
		}
	}}
	if len(k.hooks) > 0 {
		s.Hook = func(point string, _ int) {
			if g, ok := k.hooks[point]; ok {
				c.gates[g].wait()
			}
		}
	}
	if k.seams != nil {
		k.seams(c, s)
	}
	c.reg = toolctl.New(toolctl.Options{Clock: c.clock, W: c.W, D: c.D, Limit: k.limit, Seams: s})
	cfg := spec.Config{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
	cs := collect.CallSpec{Config: cfg, Kind: spec.CallContext, Exe: exe, Args: spec.Args(spec.CallContext), Dir: e.TempRoot, Env: spec.ToolEnv(cfg)}
	go func() {
		c.res = c.reg.Invoke(cs)
		close(c.returned)
	}()
	if k.drive != nil {
		k.drive(c)
	}
	var lines []string
	if c.waitReturned() {
		d := "fail"
		if c.res.Succeeded(c.W) {
			d = "success"
		}
		lines = append(lines, "decision "+d)
	} else {
		lines = append(lines, "decision not-returned")
	}
	lines = append(lines, c.extra...)
	cerr := c.cleanup()
	sn := c.snap()
	for _, l := range toolctl.Lines(sn) {
		if k.exitAny && strings.HasPrefix(l, "wait exit ") {
			l = "wait exit *"
		}
		lines = append(lines, l)
	}
	if h := c.h; h != nil {
		h.mu.Lock()
		lines = append(lines, fmt.Sprintf("holder started %s ready %s waited %s exit %d", yesNo(h.started), yesNo(h.ready), yesNo(h.waited), h.exit))
		h.mu.Unlock()
	}
	c.mu.Lock()
	for _, p := range c.problems {
		lines = append(lines, "problem "+p)
	}
	c.mu.Unlock()
	inst := &Instance{Extra: map[string]any{"call": sn, "decision_lines": lines}}
	if h := c.h; h != nil {
		h.mu.Lock()
		inst.Extra["holder_errors"] = append([]string(nil), h.errs...)
		h.mu.Unlock()
	}
	var entered []string
	if len(sn.Events) > 0 && sn.Events[0].What == "registered" {
		entered = []string{"toolctl registration"}
	}
	c.mu.Lock()
	prepErr := c.prepErr
	c.mu.Unlock()
	if prepErr != nil {
		return Outcome{PrepErr: prepErr, Instance: inst, Terminal: sn.Terminal, CleanupErr: cerr}
	}
	return Outcome{Entered: entered, Lines: lines, Instance: inst, Terminal: sn.Terminal, CleanupErr: cerr}
}

func (c *lc9Run) closeStdinR() error {
	if c.stdinR == nil {
		return nil
	}
	return cleanupStep(c.e, "LC-9 "+c.k.name+": close the parent's copy of the stdin read end", c.stdinR.Close)
}

// cleanup releases everything the case holds and waits, within the guard,
// until the call is terminal; the decision was saved before.
func (c *lc9Run) cleanup() error {
	var errs []error
	for _, g := range c.gates {
		g.release()
	}
	if err := c.releaseStdin(); err != nil {
		errs = append(errs, err)
	}
	if h := c.h; h != nil {
		h.mu.Lock()
		h.closed = true
		relW, started := h.relW, h.started
		h.relW = nil
		h.mu.Unlock()
		if relW != nil {
			if err := cleanupStep(c.e, "LC-9 "+c.k.name+": release the holder", relW.Close); err != nil {
				errs = append(errs, err)
			}
		}
		if started {
			ch, stop := guardCh()
			select {
			case <-h.done:
			case <-ch:
				errs = append(errs, errors.New("the holder's Wait did not return: its state is unknown"))
			}
			stop()
		}
	}
	ch, stop := guardCh()
	terminal := c.reg.WaitTerminal(1, ch)
	stop()
	if !terminal {
		errs = append(errs, errors.New("the call did not become terminal within the guard: held as unknown"))
	} else if err := c.closeStdinR(); err != nil {
		errs = append(errs, err)
	}
	if c.fx != nil {
		if err := c.fx.cleanup(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// startHolder starts H in the after-handover seam: before the pseudo-tool
// starts and before the start task closes its copy of the stdout write end
// (§12.12). H inherits that write end as its stdout and never writes to it;
// its stderr is the ready pipe, its stdin the release pipe.
func (c *lc9Run) startHolder(stdoutW *os.File) {
	h := c.h
	fail := func(err error) {
		c.mu.Lock()
		if c.prepErr == nil {
			c.prepErr = err
		}
		c.mu.Unlock()
	}
	h.mu.Lock()
	closed := h.closed
	h.mu.Unlock()
	if closed {
		fail(errors.New("the holder was not started: the outer cleanup already ran"))
		return
	}
	exe, err := os.Executable()
	if err != nil {
		fail(err)
		return
	}
	var readyR, readyW, relR, relW *os.File
	if err := step(c.e, "prepare LC-9 "+c.k.name+": holder pipes", func() error {
		var err error
		if readyR, readyW, err = os.Pipe(); err != nil {
			return err
		}
		relR, relW, err = os.Pipe()
		return err
	}); err != nil {
		fail(err)
		return
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), pseudo.EnvMode+"="+pseudo.Hold)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = stdoutW, readyW, relR
	serr := step(c.e, "prepare LC-9 "+c.k.name+": start the holder", cmd.Start)
	h.mu.Lock()
	h.relW = relW
	h.started = serr == nil
	h.mu.Unlock()
	if serr != nil {
		fail(serr)
	} else {
		go func() {
			werr := cmd.Wait()
			h.mu.Lock()
			if cmd.ProcessState != nil {
				h.exit = cmd.ProcessState.ExitCode()
			}
			if werr != nil {
				h.errs = append(h.errs, "wait: "+werr.Error())
			}
			h.waited = cmd.ProcessState != nil
			h.mu.Unlock()
			close(h.done)
		}()
	}
	for _, f := range []*os.File{readyW, relR} {
		if err := cleanupStep(c.e, "LC-9 "+c.k.name+": close the parent's copy of a holder pipe end", f.Close); err != nil {
			h.err(err.Error())
		}
	}
	if serr != nil {
		if err := cleanupStep(c.e, "LC-9 "+c.k.name+": close the ready pipe", readyR.Close); err != nil {
			h.err(err.Error())
		}
		return
	}
	got := make(chan string, 1)
	go func() {
		b := make([]byte, len(pseudo.ReadyLine))
		n, err := io.ReadFull(readyR, b)
		if err != nil {
			h.err("ready: " + err.Error())
		}
		got <- string(b[:n])
		if err := cleanupStep(c.e, "LC-9 "+c.k.name+": close the ready pipe", readyR.Close); err != nil {
			h.err(err.Error())
		}
	}()
	ch, stop := guardCh()
	defer stop()
	select {
	case s := <-got:
		h.mu.Lock()
		h.ready = s == pseudo.ReadyLine
		ready := h.ready
		h.mu.Unlock()
		if !ready {
			fail(fmt.Errorf("the holder is not ready: %q", s))
		}
	case <-ch:
		fail(errors.New("the holder did not report ready within the guard"))
	}
}

// The expected lines, written independently of toolctl.

const emptySHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func shaOf(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

type lc9Want struct {
	decision string
	extra    []string
	class    string
	state    string
	start    string
	desc     [4]string
	injected [4]int
	out      [2]string
	wait     string
	pidfd    string // Linux only
	req      string // after "request <mode> "
	notAdmit []string
	unknown  []string
	expired  bool
	late     bool
	success  bool
	tail     []string
}

func (w lc9Want) lines(goos string) []string {
	out := []string{"decision " + w.decision}
	out = append(out, w.extra...)
	out = append(out, "class "+w.class, "state "+w.state, "start "+w.start)
	for i, d := range w.desc {
		out = append(out, toolctl.DescNames[i]+" "+d)
		for range w.injected[i] {
			out = append(out, toolctl.DescNames[i]+" injected")
		}
	}
	out = append(out, "stdout "+w.out[0], "stderr "+w.out[1], "wait "+w.wait)
	mode := toolctl.ModeNone
	switch goos {
	case "linux":
		mode = toolctl.ModePidfd
		out = append(out, "pidfd "+w.pidfd)
	case "windows":
		mode = toolctl.ModeKill
	}
	out = append(out, "request "+mode+" "+w.req)
	for _, n := range w.notAdmit {
		out = append(out, "not-admitted "+n)
	}
	for _, u := range w.unknown {
		out = append(out, "cleanup-unknown "+u)
	}
	out = append(out, "expired "+yesNo(w.expired), "late "+yesNo(w.late), "terminal yes", "success "+yesNo(w.success))
	return append(out, w.tail...)
}

var closedOK = [4]string{"closed ok", "closed ok", "closed ok", "closed ok"}

func eofKept(b []byte) string { return fmt.Sprintf("eof kept %d sha256 %s", len(b), shaOf(b)) }

var eofEmpty = "eof kept 0 sha256 " + emptySHA

const darwinUnknown = "no termination request on this platform (darwin: os/wait_unimp.go); the tool may outlive the test process"

// healthy is the expectation of a pseudo-tool that exited 0 and was reaped
// before W, with nothing written.
func healthy() lc9Want {
	return lc9Want{decision: "success", class: "B", state: "waited", start: "ok", desc: closedOK, out: [2]string{eofEmpty, eofEmpty},
		wait: "exit 0", pidfd: "closed ok closes 1", req: "none sends 0", success: true}
}

func expired(w lc9Want) lc9Want {
	w.decision, w.expired, w.late, w.success = "fail", true, true, false
	return w
}

func failed(w lc9Want) lc9Want {
	w.decision, w.success = "fail", false
	return w
}

// LC9 returns the LC-9 variants (phase 2): the common ones on every
// platform, and those of the pidfd state machine on Linux only (not
// applicable elsewhere, with the reason).
func LC9(goos string) []Variant {
	out := make([]Variant, 0, len(lc9Cases()))
	for _, k := range lc9Cases() {
		out = append(out, k.variant(goos))
	}
	return out
}

func lc9Cases() []lc9Case {
	echoArgs := spec.Args(spec.CallContext)
	echo := []byte(pseudo.EchoOutput(echoArgs))
	t2line := fmt.Sprintf("arg 4 %d %s", spec.T2Len, spec.T2SHA256)
	x4k := []byte(strings.Repeat("x", 4096))
	reqKill := func(goos, linux, windows, other string) string {
		switch goos {
		case "linux":
			return linux
		case "windows":
			return windows
		}
		return other
	}
	return []lc9Case{
		{
			name: "normal-eof", mode: pseudo.EchoArgs,
			want: func(goos string) []string {
				w := healthy()
				w.out = [2]string{eofKept(echo), eofKept([]byte(pseudo.StderrLine))}
				// The fixed T2 binding is the fourth argument the child
				// received: its length and SHA-256 (LC-10's values).
				if !strings.Contains(string(echo), "\n"+t2line+"\n") {
					w.tail = append(w.tail, "problem the echo expectation lacks T2's line")
				}
				return w.lines(goos)
			},
		},
		{
			name: "half-construction", mode: pseudo.Silent,
			seams: func(_ *lc9Run, s *toolctl.Seams) {
				s.Pipe = func(_, n int) (*os.File, *os.File, error) {
					if n == 1 {
						return nil, nil, errSeamPipe
					}
					return os.Pipe()
				}
			},
			want: func(goos string) []string {
				w := lc9Want{decision: "fail", class: "A phase pipe", state: "start-failed", start: "error",
					desc: [4]string{"closed ok", "closed ok", "none", "none"}, out: [2]string{"not-started", "not-started"},
					wait: "not-called", pidfd: "none closes 0", req: "none sends 0"}
				return w.lines(goos)
			},
		},
		{
			name: "start-failure", mode: pseudo.Silent, absent: true,
			want: func(goos string) []string {
				// C, phase start, on the three platforms (§12.12). On
				// Windows the name ends in ".exe": exec.Command's
				// lookExtensions accepts a known extension without looking
				// at the file (os/exec/exec.go:466 ~ 480 and lp_windows.go),
				// so Cmd.Err stays nil and CreateProcess fails; without a
				// known extension the lookup would fail first and §12.6's
				// A ② would apply.
				w := lc9Want{decision: "fail", class: "C phase start", state: "start-failed", start: "error", desc: closedOK,
					out: [2]string{eofEmpty, eofEmpty}, wait: "not-called", pidfd: "none closes 0", req: "none sends 0"}
				return w.lines(goos)
			},
		},
		{
			name: "start-failure-not-executable", mode: pseudo.Silent, badExe: true,
			want: func(goos string) []string {
				w := lc9Want{decision: "fail", class: "C phase start", state: "start-failed", start: "error", desc: closedOK,
					out: [2]string{eofEmpty, eofEmpty}, wait: "not-called", pidfd: "none closes 0", req: "none sends 0"}
				return w.lines(goos)
			},
		},
		{
			name: "read-error", mode: pseudo.Silent,
			seams: func(_ *lc9Run, s *toolctl.Seams) {
				s.Read = func(_, stream int, f *os.File, p []byte) (int, error) {
					if stream == 0 {
						return 0, errSeamRead
					}
					return f.Read(p)
				}
			},
			want: func(goos string) []string {
				w := failed(healthy())
				w.out[0] = "read-error kept 0 sha256 " + emptySHA
				return w.lines(goos)
			},
		},
		{
			name: "overflow", mode: pseudo.Flood, bytes: 1 << 20, limit: 4096, exitAny: true,
			want: func(goos string) []string {
				w := failed(healthy())
				w.out[0] = fmt.Sprintf("overflow kept %d sha256 %s", len(x4k), shaOf(x4k))
				w.wait = "exit *"
				return w.lines(goos)
			},
		},
		{
			name: "close-injected", mode: pseudo.Silent,
			seams: func(_ *lc9Run, s *toolctl.Seams) {
				s.InjectClose = func(_, d int) error {
					if d == toolctl.StdoutR {
						return errInjected
					}
					return nil
				}
			},
			want: func(goos string) []string {
				w := failed(healthy())
				w.injected[toolctl.StdoutR] = 1
				return w.lines(goos)
			},
		},
		{
			name: "close-real-error", mode: pseudo.Silent,
			seams: func(_ *lc9Run, s *toolctl.Seams) {
				s.CloseFile = func(_, d int, f *os.File) error {
					err := f.Close()
					if d == toolctl.StdoutR && err == nil {
						return errSeamClose
					}
					return err
				}
				s.InjectClose = func(_, d int) error {
					if d == toolctl.StdoutR {
						return errInjected
					}
					return nil
				}
			},
			want: func(goos string) []string {
				w := failed(healthy())
				w.desc[toolctl.StdoutR] = "closed error"
				w.injected[toolctl.StdoutR] = 1
				return w.lines(goos)
			},
		},
		{
			name: "expiry", mode: pseudo.Block, stdin: true,
			drive: func(c *lc9Run) {
				if !c.until("the pseudo-tool started", startedPidfd) {
					return
				}
				c.clock.Set(c.W)
				if !c.waitReturned() {
					return
				}
				if m := c.reg.Mode(); m == toolctl.ModePidfd || m == toolctl.ModeKill {
					c.until("the request done", func(s toolctl.Snapshot) bool { return s.Req.State == toolctl.ReqDone })
				}
			},
			want: func(goos string) []string {
				w := expired(healthy())
				w.wait = reqKill(goos, "exit -1", "exit 1", "exit 0")
				w.req = reqKill(goos, "done sends 1 ok", "done sends 1 ok", "none sends 0")
				if goos != "linux" && goos != "windows" {
					w.unknown = []string{darwinUnknown}
				}
				return w.lines(goos)
			},
		},
		{
			name: "blocked-close", mode: pseudo.Silent, gates: []string{"close"},
			seams: func(c *lc9Run, s *toolctl.Seams) {
				s.CloseFile = func(_, d int, f *os.File) error {
					if d == toolctl.StdoutR {
						c.gates["close"].wait()
					}
					return f.Close()
				}
			},
			drive: func(c *lc9Run) {
				if !c.until("only the stdout close pending", func(s toolctl.Snapshot) bool {
					return s.Tasks == 1 && s.State == toolctl.StWaited && s.Desc[toolctl.StdoutR].State == toolctl.DescClosing &&
						s.Desc[toolctl.StderrR].State == toolctl.DescClosed && (s.Mode != toolctl.ModePidfd || s.Pidfd.State == toolctl.PidfdClosed)
				}) {
					return
				}
				c.clock.Set(c.W)
				if c.waitReturned() {
					c.note("at-return stdout-r " + c.snap().Desc[toolctl.StdoutR].State)
				}
			},
			want: func(goos string) []string {
				w := expired(healthy())
				w.extra = []string{"at-return stdout-r closing"}
				return w.lines(goos)
			},
		},
		{
			name: "output-not-closed", mode: pseudo.Silent, holder: true,
			seams: func(c *lc9Run, s *toolctl.Seams) {
				s.AfterHandover = func(_ int, stdoutW, _ *os.File) { c.startHolder(stdoutW) }
			},
			drive: func(c *lc9Run) {
				if !c.until("only the held output pending", func(s toolctl.Snapshot) bool {
					return s.Tasks == 1 && s.State == toolctl.StWaited && !s.Out[0].Done && s.Out[1].Done &&
						s.Desc[toolctl.StderrR].State == toolctl.DescClosed && (s.Mode != toolctl.ModePidfd || s.Pidfd.State == toolctl.PidfdClosed)
				}) {
					return
				}
				c.clock.Set(c.W)
				if c.waitReturned() {
					o := c.snap().Out[0]
					c.note(fmt.Sprintf("at-return stdout done %s", yesNo(o.Done)))
				}
			},
			want: func(goos string) []string {
				w := expired(healthy())
				w.extra = []string{"at-return stdout done no"}
				w.tail = []string{"holder started yes ready yes waited yes exit 0"}
				return w.lines(goos)
			},
		},
		// Linux only: the pidfd state machine (§12.14 item 9 and its table).
		{
			name: "pidfd-open-failed", linux: true, mode: pseudo.Block, stdin: true,
			seams: func(_ *lc9Run, s *toolctl.Seams) {
				s.PidfdOpen = func(int, int) (int, error) { return -1, errSeamPidfd }
			},
			drive: func(c *lc9Run) {
				if !c.until("the pidfd open failed", func(s toolctl.Snapshot) bool {
					return s.State == toolctl.StStarted && s.Pidfd.State == toolctl.PidfdOpenFailed
				}) {
					return
				}
				c.clock.Set(c.W)
				c.waitReturned()
			},
			want: func(goos string) []string {
				w := expired(healthy())
				w.pidfd, w.req = "open-failed closes 0", "none sends 0"
				w.notAdmit = []string{"the pidfd could not be opened"}
				w.unknown = []string{"no termination was sent and the reap is not published"}
				return w.lines(goos)
			},
		},
		{
			name: "blocked-send", linux: true, mode: pseudo.Block, stdin: true, gates: []string{"send"},
			seams: func(c *lc9Run, s *toolctl.Seams) {
				s.SendSignal = func(_, fd int) error {
					c.gates["send"].wait()
					return toolctl.RealSendKill(fd)
				}
			},
			drive: func(c *lc9Run) {
				if !c.until("the pseudo-tool started", startedPidfd) {
					return
				}
				c.clock.Set(c.W)
				if !c.waitReturned() || !c.entered("send") {
					return
				}
				c.note("returned while the request runs " + c.snap().Req.State)
				if c.releaseStdin() != nil {
					return
				}
				if c.until("sealed while the request runs", func(s toolctl.Snapshot) bool {
					return s.State == toolctl.StWaited && s.Pidfd.State == toolctl.PidfdClosing
				}) {
					s := c.snap()
					c.note("while-blocked pidfd " + s.Pidfd.State + " request " + s.Req.State)
				}
			},
			want: func(goos string) []string {
				w := expired(healthy())
				w.extra = []string{"returned while the request runs running", "while-blocked pidfd closing request running"}
				w.req = "done sends 1 error"
				return w.lines(goos)
			},
		},
		{
			name: "sealed-before-reserve", linux: true, mode: pseudo.Block, stdin: true, gates: []string{"decided", "close"},
			hooks: map[string]string{toolctl.HookExpiryDecided: "decided", toolctl.HookBeforePidfdClose: "close"},
			drive: func(c *lc9Run) {
				if !c.until("the pseudo-tool started", startedPidfd) {
					return
				}
				c.clock.Set(c.W)
				if !c.entered("decided") || c.releaseStdin() != nil {
					return
				}
				if !c.until("sealed", func(s toolctl.Snapshot) bool {
					return s.State == toolctl.StWaited && s.Pidfd.State == toolctl.PidfdClosing
				}) || !c.entered("close") {
					return
				}
				c.release("decided")
				if c.waitReturned() {
					c.note("at-return pidfd " + c.snap().Pidfd.State)
				}
			},
			want: func(goos string) []string {
				w := expired(healthy())
				w.extra = []string{"at-return pidfd closing"}
				w.notAdmit = []string{"the pidfd is sealed (closing)"}
				return w.lines(goos)
			},
		},
		{
			name: "reserved-then-sealed-in-window", linux: true, mode: pseudo.Block, stdin: true, gates: []string{"req"},
			hooks: map[string]string{toolctl.HookRequestStart: "req"},
			drive: func(c *lc9Run) { driveReservedThenSealed(c, false) },
			want: func(goos string) []string {
				w := expired(healthy())
				w.extra = []string{"closer waits pidfd closing request reserved"}
				w.req = "done sends 1 error"
				return w.lines(goos)
			},
		},
		{
			name: "reserved-then-sealed-out-of-window", linux: true, mode: pseudo.Block, stdin: true, gates: []string{"req"},
			hooks: map[string]string{toolctl.HookRequestStart: "req"},
			drive: func(c *lc9Run) { driveReservedThenSealed(c, true) },
			want: func(goos string) []string {
				w := expired(healthy())
				w.extra = []string{"closer waits pidfd closing request reserved"}
				w.req = "abandoned sends 0"
				return w.lines(goos)
			},
		},
		{
			name: "wait-first", linux: true, mode: pseudo.Silent,
			drive: func(c *lc9Run) {
				if !c.waitReturned() {
					return
				}
				c.clock.Set(c.W)
				ok, why := c.reg.Reserve(1)
				c.note(fmt.Sprintf("reserve-after %s %s", admitted(ok), why))
			},
			want: func(goos string) []string {
				w := healthy()
				w.extra = []string{"reserve-after refused the pidfd is sealed (closed)"}
				return w.lines(goos)
			},
		},
		{
			name: "pidfd-late-publish", linux: true, mode: pseudo.Block, stdin: true, gates: []string{"open"},
			seams: func(c *lc9Run, s *toolctl.Seams) {
				s.PidfdOpen = func(_, pid int) (int, error) {
					c.gates["open"].wait()
					return toolctl.RealPidfdOpen(pid)
				}
			},
			drive: func(c *lc9Run) {
				if !c.until("started with the pidfd unpublished", func(s toolctl.Snapshot) bool {
					return s.State == toolctl.StStarted && s.Pidfd.State == toolctl.PidfdUnpublished
				}) || !c.entered("open") {
					return
				}
				c.clock.Set(c.W)
				if !c.waitReturned() {
					return
				}
				s := c.snap()
				c.note("at-return pidfd " + s.Pidfd.State + " request " + s.Req.State)
				c.release("open")
				c.until("the request done", func(s toolctl.Snapshot) bool { return s.Req.State == toolctl.ReqDone })
			},
			want: func(goos string) []string {
				w := expired(healthy())
				w.extra = []string{"at-return pidfd unpublished request none"}
				w.wait, w.req = "exit -1", "done sends 1 ok"
				return w.lines(goos)
			},
		},
		{
			name: "abandoned-before-wait", linux: true, mode: pseudo.Block, stdin: true, gates: []string{"req"},
			hooks: map[string]string{toolctl.HookRequestStart: "req"},
			drive: func(c *lc9Run) {
				if !c.until("the pseudo-tool started", startedPidfd) {
					return
				}
				c.clock.Set(c.W)
				if !c.waitReturned() || !c.entered("req") {
					return
				}
				c.clock.Set(c.D)
				c.release("req")
				if c.until("the request abandoned", func(s toolctl.Snapshot) bool { return s.Req.State == toolctl.ReqAbandoned }) {
					s := c.snap()
					c.note("after-abandon pidfd " + s.Pidfd.State + " state " + s.State)
				}
			},
			want: func(goos string) []string {
				w := expired(healthy())
				w.extra = []string{"after-abandon pidfd open state started"}
				w.req = "abandoned sends 0"
				w.unknown = []string{"the termination request was abandoned; no termination was sent"}
				return w.lines(goos)
			},
		},
		{
			name: "closing-old-snapshot", linux: true, mode: pseudo.Block, stdin: true, gates: []string{"close"},
			hooks: map[string]string{toolctl.HookBeforePidfdClose: "close"},
			drive: func(c *lc9Run) {
				if !c.until("the pseudo-tool started", startedPidfd) {
					return
				}
				c.note("old-snapshot pidfd " + c.snap().Pidfd.State)
				if c.releaseStdin() != nil {
					return
				}
				if !c.until("sealed", func(s toolctl.Snapshot) bool {
					return s.State == toolctl.StWaited && s.Pidfd.State == toolctl.PidfdClosing
				}) || !c.entered("close") {
					return
				}
				c.clock.Set(c.W)
				if !c.waitReturned() {
					return
				}
				ok, why := c.reg.Reserve(1)
				c.note(fmt.Sprintf("reserve-with-old-snapshot %s %s", admitted(ok), why))
			},
			want: func(goos string) []string {
				w := expired(healthy())
				w.extra = []string{"old-snapshot pidfd open", "reserve-with-old-snapshot refused the pidfd is sealed (closing)"}
				return w.lines(goos)
			},
		},
	}
}

// driveReservedThenSealed: the request is reserved at W and held before
// its window check; the pseudo-tool is released, reaped and the pidfd
// sealed, so the closer waits; then the request runs in the window or, out
// of it, is abandoned.
func driveReservedThenSealed(c *lc9Run, outOfWindow bool) {
	if !c.until("the pseudo-tool started", startedPidfd) {
		return
	}
	c.clock.Set(c.W)
	if !c.waitReturned() || !c.entered("req") || c.releaseStdin() != nil {
		return
	}
	if !c.until("the closer waiting", func(s toolctl.Snapshot) bool {
		return s.State == toolctl.StWaited && s.Pidfd.State == toolctl.PidfdClosing && s.Req.State == toolctl.ReqReserved
	}) {
		return
	}
	s := c.snap()
	c.note("closer waits pidfd " + s.Pidfd.State + " request " + s.Req.State)
	if outOfWindow {
		c.clock.Set(c.D)
	}
	c.release("req")
	c.until("the request finished", func(s toolctl.Snapshot) bool {
		return s.Req.State == toolctl.ReqDone || s.Req.State == toolctl.ReqAbandoned
	})
}

func admitted(ok bool) string {
	if ok {
		return "admitted"
	}
	return "refused"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
