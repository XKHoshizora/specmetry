package controls

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
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
// holder H) and judges only once the call and H are terminal; the complete
// expectations below are written independently of toolctl.
//
// The judgement consumes the structured fault set (Review 154 B-154-4):
// every error the protocol recorded, by site and source, with its kind
// given by the error's identity (a planned seam fault, an injection and a
// real error never compare equal); every real result a seam replaced; and
// H's faults. Each is a line of the complete expectation, so an unexpected
// real error, or a planned fault whose source did not happen, is a miss.

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

// faultKind is an error's kind by its identity, never by its text: the
// planned seam faults and the injection each have their own kind, apart
// from every real error.
func faultKind(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, errSeamPipe):
		return "seam-pipe"
	case errors.Is(err, errSeamRead):
		return "seam-read"
	case errors.Is(err, errSeamClose):
		return "seam-close"
	case errors.Is(err, errSeamPidfd):
		return "seam-pidfd"
	case errors.Is(err, errInjected):
		return "injected"
	case errors.Is(err, ErrNotAdmitted):
		return "not-admitted"
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return "exit-status"
	}
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "not-exist"
	case errors.Is(err, fs.ErrPermission):
		return "permission"
	}
	var en syscall.Errno
	if errors.As(err, &en) {
		if en == syscall.ESRCH {
			return "ESRCH"
		}
		return "errno " + strconv.FormatUint(uint64(en), 10)
	}
	return "other"
}

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
	h        *holder
	fx       *fixture
	mu       sync.Mutex
	problems []string
	prepErr  error
	saved    bool    // the tested decision was saved (before any release)
	seamLog  []Fault // what the seams replaced
}

type lc9Case struct {
	name   string
	linux  bool // only on Linux: the pidfd exists only there (§12.14 item 8)
	mode   string
	stdin  bool
	limit  int64
	bytes  int
	gates  []string
	hooks  map[string]string // hook point -> gate
	holder bool              // the case starts the output holder H
	absent bool              // start an absolute path that does not exist
	badExe bool              // start an existing file that is not an executable
	seams  func(c *lc9Run, s *toolctl.Seams)
	drive  func(c *lc9Run)
	want   func(goos string) []string
	// realClose and realSend replace the real operation a seam calls, and
	// holderSeams H's operations; only the regressions of Review 154 set
	// them (nil: the real operations).
	realClose   func(d int, f *os.File) error
	realSend    func(fd int) error
	holderSeams func(h *holder)
}

func (c *lc9Run) realClose(d int, f *os.File) error {
	if c.k.realClose != nil {
		return c.k.realClose(d, f)
	}
	return f.Close()
}

func (c *lc9Run) realSend(fd int) error {
	if c.k.realSend != nil {
		return c.k.realSend(fd)
	}
	return toolctl.RealSendKill(fd)
}

// seam records what a seam did at a site in place of the real operation:
// the real result it hid ("not-called" when it did not call it) and what
// it returned.
func (c *lc9Run) seam(site, real string, returned error) {
	c.mu.Lock()
	c.seamLog = append(c.seamLog, Fault{Site: site, Source: "seam", Kind: "real " + real + " returned " + faultKind(returned)})
	c.mu.Unlock()
}

func (c *lc9Run) problem(s string) {
	c.mu.Lock()
	c.problems = append(c.problems, s)
	c.mu.Unlock()
}

func (c *lc9Run) prepFail(err error) {
	c.mu.Lock()
	if c.prepErr == nil {
		c.prepErr = err
	}
	c.mu.Unlock()
}

func (c *lc9Run) decisionSaved() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.saved
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
		c.h = newHolder(c)
		if k.holderSeams != nil {
			k.holderSeams(c.h)
		}
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
	// The decision is saved (copied) before anything is released.
	c.mu.Lock()
	c.saved = true
	c.mu.Unlock()
	cerr := c.cleanup()
	sn := c.snap()
	lines = append(lines, toolctl.Lines(sn)...)
	inst := &Instance{Extra: map[string]any{"call": sn}}
	for _, f := range sn.Faults {
		inst.Faults = append(inst.Faults, Fault{Site: f.Site, Source: f.Source, Kind: faultKind(f.Err), Detail: f.Detail})
	}
	c.mu.Lock()
	inst.Faults = append(inst.Faults, c.seamLog...)
	c.mu.Unlock()
	terminal := sn.Terminal
	if h := c.h; h != nil {
		hl := h.lines()
		lines = append(lines, hl...)
		inst.Faults = append(inst.Faults, h.faultSet()...)
		terminal = terminal && h.terminal()
		inst.Extra["holder"] = hl
	}
	for _, f := range inst.Faults {
		lines = append(lines, f.Line())
	}
	c.mu.Lock()
	for _, p := range c.problems {
		lines = append(lines, "problem "+p)
	}
	prepErr := c.prepErr
	c.mu.Unlock()
	inst.Extra["decision_lines"] = lines
	var entered []string
	if len(sn.Events) > 0 && sn.Events[0].What == "registered" {
		entered = []string{"toolctl registration"}
	}
	if prepErr != nil {
		return Outcome{PrepErr: prepErr, Instance: inst, Terminal: terminal, CleanupErr: cerr}
	}
	return Outcome{Entered: entered, Lines: lines, Instance: inst, Terminal: terminal, CleanupErr: cerr}
}

func (c *lc9Run) closeStdinR() error {
	if c.stdinR == nil {
		return nil
	}
	return cleanupStep(c.e, "LC-9 "+c.k.name+": close the parent's copy of the stdin read end", c.stdinR.Close)
}

// cleanup releases everything the case holds and waits, within the guard,
// until H and the call are terminal; the decision was saved before.
func (c *lc9Run) cleanup() error {
	var errs []error
	for _, g := range c.gates {
		g.release()
	}
	if err := c.releaseStdin(); err != nil {
		errs = append(errs, err)
	}
	if h := c.h; h != nil {
		h.stop()
		if !h.waitTerminal(lc9Guard) {
			errs = append(errs, errors.New("the holder is not terminal within the guard: held as unknown"))
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

// The expected lines, written independently of toolctl.

const emptySHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func shaOf(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

type lc9Want struct {
	decision   string
	extra      []string
	class      string
	state      string
	admissions []string // the points admitted, in order (nil: all four)
	start      string
	cmd        bool
	desc       [4]string
	injected   [4]int
	out        [2]string
	wait       string
	pidfd      string // Linux only
	req        string // after "request <mode> "
	notAdmit   []string
	unknown    []string
	expired    bool
	late       bool
	success    bool
	holder     []string
	faults     []string // "<site> <source> <kind>"
	tail       []string
}

var allPoints = []string{"entry", "stdout pipe", "stderr pipe", "start"}

// descLine is a descriptor's expected line: its close state and, from it,
// what is known of its release and whether the record holds the object.
func descLine(name, st string) string {
	switch st {
	case "closed ok":
		return name + " closed ok release confirmed object held"
	case "closed error":
		return name + " closed error release unknown object held"
	case "closing":
		return name + " closing release pending object held"
	case "none":
		return name + " none release none object none"
	}
	return name + " " + st + " (unknown expectation)"
}

func (w lc9Want) lines(goos string) []string {
	out := []string{"decision " + w.decision}
	out = append(out, w.extra...)
	out = append(out, "class "+w.class, "state "+w.state)
	adm := w.admissions
	if adm == nil {
		adm = allPoints
	}
	for _, p := range adm {
		out = append(out, "admission "+p+" admitted")
	}
	out = append(out, "start "+w.start, "cmd held "+yesNo(w.cmd))
	for i, d := range w.desc {
		out = append(out, descLine(toolctl.DescNames[i], d))
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
	out = append(out, "panic no", "expired "+yesNo(w.expired), "late "+yesNo(w.late), "terminal yes", "success "+yesNo(w.success))
	out = append(out, w.holder...)
	for _, f := range w.faults {
		out = append(out, "fault "+f)
	}
	return append(out, w.tail...)
}

var closedOK = [4]string{"closed ok", "closed ok", "closed ok", "closed ok"}

func eofKept(b []byte) string { return fmt.Sprintf("eof kept %d sha256 %s", len(b), shaOf(b)) }

var eofEmpty = "eof kept 0 sha256 " + emptySHA

const darwinUnknown = "no termination request on this platform (darwin: os/wait_unimp.go); the tool may outlive the test process"

// healthy is the expectation of a pseudo-tool that exited 0 and was reaped
// before W, with nothing written.
func healthy() lc9Want {
	return lc9Want{decision: "success", class: "B", state: "waited", start: "ok", cmd: true, desc: closedOK, out: [2]string{eofEmpty, eofEmpty},
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
	byOS := func(goos, linux, windows, other string) string {
		switch goos {
		case "linux":
			return linux
		case "windows":
			return windows
		}
		return other
	}
	// killedWait: the process was ended by the termination request (Linux:
	// SIGKILL through the pidfd; Windows: Kill, exit code 1); its one Wait
	// returns an *exec.ExitError. darwin sends no request.
	killedWait := func(goos string) []string {
		if goos == "linux" || goos == "windows" {
			return []string{"wait operation exit-status"}
		}
		return nil
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
			seams: func(c *lc9Run, s *toolctl.Seams) {
				s.Pipe = func(_, n int) (*os.File, *os.File, error) {
					if n == 1 {
						c.seam("pipe stderr", "not-called", errSeamPipe)
						return nil, nil, errSeamPipe
					}
					return os.Pipe()
				}
			},
			want: func(goos string) []string {
				w := lc9Want{decision: "fail", class: "A phase pipe", state: "start-failed", admissions: allPoints[:3], start: "not-initiated",
					desc: [4]string{"closed ok", "closed ok", "none", "none"}, out: [2]string{"not-started", "not-started"},
					wait: "not-called", pidfd: "none closes 0", req: "none sends 0",
					faults: []string{"pipe stderr operation seam-pipe", "pipe stderr seam real not-called returned seam-pipe"}}
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
				// A ② would apply. The real error is "does not exist" on
				// every platform.
				w := lc9Want{decision: "fail", class: "C phase start", state: "start-failed", start: "error", cmd: true, desc: closedOK,
					out: [2]string{eofEmpty, eofEmpty}, wait: "not-called", pidfd: "none closes 0", req: "none sends 0",
					faults: []string{"start operation not-exist"}}
				return w.lines(goos)
			},
		},
		{
			name: "start-failure-not-executable", mode: pseudo.Silent, badExe: true,
			want: func(goos string) []string {
				// The real error: execve's EACCES on Linux and darwin (a
				// file without the execute permission); on Windows
				// CreateProcess's ERROR_EXE_MACHINE_TYPE_MISMATCH (216) for
				// a file that is not a PE image (observed on the local
				// Windows 11 x64; an x64 Windows has no 16-bit subsystem to
				// run it as a DOS program). The first expectation written,
				// ERROR_BAD_EXE_FORMAT (193), did not hold locally.
				w := lc9Want{decision: "fail", class: "C phase start", state: "start-failed", start: "error", cmd: true, desc: closedOK,
					out: [2]string{eofEmpty, eofEmpty}, wait: "not-called", pidfd: "none closes 0", req: "none sends 0",
					faults: []string{"start operation " + byOS(goos, "permission", "errno 216", "permission")}}
				return w.lines(goos)
			},
		},
		{
			name: "read-error", mode: pseudo.Silent,
			seams: func(c *lc9Run, s *toolctl.Seams) {
				s.Read = func(_, stream int, f *os.File, p []byte) (int, error) {
					if stream == 0 {
						c.seam("read stdout", "not-called", errSeamRead)
						return 0, errSeamRead
					}
					return f.Read(p)
				}
			},
			want: func(goos string) []string {
				w := failed(healthy())
				w.out[0] = "read-error kept 0 sha256 " + emptySHA
				w.faults = []string{"read stdout operation seam-read", "read stdout seam real not-called returned seam-read"}
				return w.lines(goos)
			},
		},
		{
			name: "overflow", mode: pseudo.Flood, bytes: 1 << 20, limit: 4096,
			want: func(goos string) []string {
				// After the read end is closed, the pseudo-tool's next write
				// gets EPIPE: on Unix the Go runtime then ends it with
				// SIGPIPE (a write to a broken pipe on fd 1, os/signal's
				// documentation), so its Wait returns an *exec.ExitError
				// and the exit code is -1; on Windows the write error is
				// ignored and it exits 0. 1 MiB never fits a pipe buffer,
				// so a write after the close always happens.
				w := failed(healthy())
				w.out[0] = fmt.Sprintf("overflow kept %d sha256 %s", len(x4k), shaOf(x4k))
				if goos != "windows" {
					w.wait = "exit -1"
					w.faults = []string{"wait operation exit-status"}
				}
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
				w.faults = []string{"close stdout-r injected injected"}
				return w.lines(goos)
			},
		},
		{
			name: "close-real-error", mode: pseudo.Silent,
			seams: func(c *lc9Run, s *toolctl.Seams) {
				s.CloseFile = func(_, d int, f *os.File) error {
					err := c.realClose(d, f)
					if d != toolctl.StdoutR {
						return err
					}
					// The planned fault presupposes that the real Close
					// returned nil; otherwise the real result is returned
					// as it came and the plan is not applied.
					if err == nil {
						c.seam("close stdout-r", "ok", errSeamClose)
						return errSeamClose
					}
					c.seam("close stdout-r", faultKind(err), err)
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
				w.faults = []string{"close stdout-r operation seam-close", "close stdout-r injected injected", "close stdout-r seam real ok returned seam-close"}
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
				w.wait = byOS(goos, "exit -1", "exit 1", "exit 0")
				w.req = byOS(goos, "done sends 1 ok", "done sends 1 ok", "none sends 0")
				if goos != "linux" && goos != "windows" {
					w.unknown = []string{darwinUnknown}
				}
				w.faults = killedWait(goos)
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
				w.holder = []string{"holder start started", "holder ready ok", "holder wait done exit 0", "holder release by-cleanup after-saved-decision yes",
					"holder ready-r closed ok", "holder ready-w closed ok", "holder rel-r closed ok", "holder rel-w closed ok", "holder tasks 0"}
				return w.lines(goos)
			},
		},
		// Linux only: the pidfd state machine (§12.14 item 9 and its table).
		{
			name: "pidfd-open-failed", linux: true, mode: pseudo.Block, stdin: true,
			seams: func(c *lc9Run, s *toolctl.Seams) {
				s.PidfdOpen = func(int, int) (int, error) {
					c.seam("pidfd open", "not-called", errSeamPidfd)
					return -1, errSeamPidfd
				}
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
				w.faults = []string{"pidfd open operation seam-pidfd", "pidfd open seam real not-called returned seam-pidfd"}
				return w.lines(goos)
			},
		},
		{
			name: "blocked-send", linux: true, mode: pseudo.Block, stdin: true, gates: []string{"send"},
			seams: func(c *lc9Run, s *toolctl.Seams) {
				s.SendSignal = func(_, fd int) error {
					c.gates["send"].wait()
					return c.realSend(fd)
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
				// The request runs after the process was reaped: the real
				// pidfd_send_signal returns ESRCH, and no other kind.
				w := expired(healthy())
				w.extra = []string{"returned while the request runs running", "while-blocked pidfd closing request running"}
				w.req = "done sends 1 error"
				w.faults = []string{"request operation ESRCH"}
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
				w.faults = []string{"request operation ESRCH"}
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
				w.faults = killedWait(goos)
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
