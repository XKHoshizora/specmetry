package controls

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/pseudo"
)

// The controlled output holder H of LC-9's output-not-closed variant
// (implementation plan §12.12 "输出不关闭的拒绝路径"), with its whole life
// cycle held (Review 154 B-154-3): the start intent is registered before
// any work; the ready pipe and the release pipe are each their own admitted
// outer preparation, and each end is registered when it exists; the Cmd is
// registered when built; the ready task and the one Wait are counted before
// they start; the one release request is decided under the holder's lock,
// by the outer cleanup when H has started, or by the start path when the
// cleanup stopped while H's Start was in flight. No reference is ever
// cleared: an end whose close was refused, did not return or failed stays
// held. H is terminal only when no necessary task runs, every acquired end's
// close has returned, and a started H was released and waited. H is the
// test's direct child, never the tool's descendant; no PID action is ever
// taken on it.

// The holder's start states.
const (
	hNone       = "none"        // nothing done yet
	hPreparing  = "preparing"   // the start intent is registered; the pipes are being built
	hStarting   = "starting"    // H's Start was initiated and has not returned
	hStarted    = "started"     // H's Start returned nil
	hFailed     = "failed"      // a preparation or Start failed or was not admitted
	hNotStarted = "not-started" // the outer cleanup had stopped before H's Start was initiated
)

// hEndNames are the parent's four ends: the ready pipe's read and write
// ends, the release pipe's read and write ends.
var hEndNames = []string{"ready-r", "ready-w", "rel-r", "rel-w"}

// The states of an end.
const (
	endOpen        = "open"
	endClosing     = "closing"
	endClosed      = "closed"       // the close returned; its real result is kept
	endNotAdmitted = "not-admitted" // the close was refused: the end stays open and held; never retried
)

type hend struct {
	f     *os.File
	state string
	err   error
}

type holder struct {
	c       *lc9Run
	mu      sync.Mutex
	changed chan struct{} // closed and renewed on every publication
	stopped bool          // the outer cleanup stopped new work
	start   string
	ends    map[string]*hend
	cmd     *exec.Cmd
	release string // none, by-cleanup, by-start
	saved   bool   // the release came after the tested decision was saved
	ready   string // none, running, ok, wrong, error
	wait    string // none, running, done
	exit    int
	tasks   int // the necessary tasks still running: the ready task, the one Wait, each close in flight
	faults  []Fault

	// The regressions replace these (nil: the real operations).
	pipeOp     func(n int) (*os.File, *os.File, error)
	closeOp    func(name string, f *os.File) error
	afterStart func() // after H's Start returned, before its result is published
}

func newHolder(c *lc9Run) *holder {
	return &holder{c: c, changed: make(chan struct{}), start: hNone, ends: map[string]*hend{}, release: "none", ready: "none", wait: "none", exit: -2}
}

func (h *holder) publishLocked() {
	close(h.changed)
	h.changed = make(chan struct{})
}

func holderSource(err error) string {
	if errors.Is(err, ErrNotAdmitted) {
		return "admission"
	}
	return "real"
}

func (h *holder) faultLocked(site string, err error) {
	h.faults = append(h.faults, Fault{Site: "holder " + site, Source: holderSource(err), Kind: faultKind(err), Detail: err.Error()})
}

func (h *holder) pipe(n int) (*os.File, *os.File, error) {
	if h.pipeOp != nil {
		return h.pipeOp(n)
	}
	return os.Pipe()
}

func (h *holder) closeFile(name string, f *os.File) error {
	if h.closeOp != nil {
		return h.closeOp(name, f)
	}
	return f.Close()
}

func (h *holder) file(name string) *os.File {
	h.mu.Lock()
	defer h.mu.Unlock()
	if e := h.ends[name]; e != nil {
		return e.f
	}
	return nil
}

func (h *holder) setStart(s string) {
	h.mu.Lock()
	h.start = s
	h.publishLocked()
	h.mu.Unlock()
}

func (h *holder) isStopped() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.stopped
}

// buildPipe builds one pipe (0: ready, 1: release) as its own admitted
// outer preparation; its two ends are registered as soon as they exist.
func (h *holder) buildPipe(n int) error {
	names := [2][2]string{{"ready-r", "ready-w"}, {"rel-r", "rel-w"}}[n]
	what := [2]string{"ready pipe", "release pipe"}[n]
	err := step(h.c.e, "prepare LC-9 "+h.c.k.name+": holder "+what, func() error {
		r, w, err := h.pipe(n)
		if err == nil {
			h.mu.Lock()
			h.ends[names[0]] = &hend{f: r, state: endOpen}
			h.ends[names[1]] = &hend{f: w, state: endOpen}
			h.publishLocked()
			h.mu.Unlock()
		}
		return err
	})
	if err != nil {
		h.mu.Lock()
		h.faultLocked(what, err)
		h.publishLocked()
		h.mu.Unlock()
	}
	return err
}

// closeEnd closes one open end once, as an admitted outer cleanup. A
// refused close leaves the end open and held; an end not open is never
// closed again.
func (h *holder) closeEnd(name string) {
	h.mu.Lock()
	e := h.ends[name]
	if e == nil || e.state != endOpen {
		h.mu.Unlock()
		return
	}
	e.state = endClosing
	h.tasks++
	h.publishLocked()
	h.mu.Unlock()
	err := cleanupStep(h.c.e, "LC-9 "+h.c.k.name+": close the holder's "+name, func() error { return h.closeFile(name, e.f) })
	h.mu.Lock()
	h.tasks--
	if errors.Is(err, ErrNotAdmitted) {
		e.state = endNotAdmitted
	} else {
		e.state, e.err = endClosed, err
	}
	if err != nil {
		h.faultLocked("close "+name, err)
	}
	h.publishLocked()
	h.mu.Unlock()
}

func (h *holder) closeEnds(names ...string) {
	for _, n := range names {
		h.closeEnd(n)
	}
}

// startHolder starts H in the after-handover seam: before the pseudo-tool
// starts and before the start task closes its copy of the stdout write end
// (§12.12). H inherits that write end as its stdout and never writes to it;
// its stderr is the ready pipe, its stdin the release pipe. It returns once
// the ready reader published its result, or at the guard.
func (c *lc9Run) startHolder(stdoutW *os.File) {
	h := c.h
	h.mu.Lock()
	if h.stopped {
		h.start = hNotStarted
		h.publishLocked()
		h.mu.Unlock()
		c.prepFail(errors.New("the holder was not started: the outer cleanup had already stopped"))
		return
	}
	h.start = hPreparing
	h.publishLocked()
	h.mu.Unlock()
	exe, err := os.Executable()
	if err != nil {
		h.mu.Lock()
		h.faultLocked("executable", err)
		h.mu.Unlock()
		h.setStart(hFailed)
		c.prepFail(err)
		return
	}
	if err := h.buildPipe(0); err != nil {
		h.setStart(hFailed)
		c.prepFail(err)
		return
	}
	if h.isStopped() {
		h.closeEnds("ready-r", "ready-w")
		h.setStart(hNotStarted)
		c.prepFail(errors.New("the holder was not started: the outer cleanup stopped during its preparation"))
		return
	}
	if err := h.buildPipe(1); err != nil {
		h.closeEnds("ready-r", "ready-w")
		h.setStart(hFailed)
		c.prepFail(err)
		return
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), pseudo.EnvMode+"="+pseudo.Hold)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = stdoutW, h.file("ready-w"), h.file("rel-r")
	h.mu.Lock()
	h.cmd = cmd
	if h.stopped {
		h.start = hNotStarted
		h.publishLocked()
		h.mu.Unlock()
		h.closeEnds(hEndNames...)
		c.prepFail(errors.New("the holder was not started: the outer cleanup stopped before its Start"))
		return
	}
	h.start = hStarting
	h.publishLocked()
	h.mu.Unlock()
	serr := step(c.e, "prepare LC-9 "+c.k.name+": start the holder", cmd.Start)
	if h.afterStart != nil {
		h.afterStart()
	}
	h.mu.Lock()
	releaseNow := false
	if serr != nil {
		h.start = hFailed
		h.faultLocked("start", serr)
	} else {
		h.start = hStarted
		h.wait, h.ready = "running", "running"
		h.tasks += 2 // the one Wait and the ready reader, counted before they start
		if h.stopped && h.release == "none" {
			// The cleanup stopped while Start was in flight: the start path
			// makes the one release request.
			h.release, h.saved, releaseNow = "by-start", c.decisionSaved(), true
		}
	}
	h.publishLocked()
	h.mu.Unlock()
	if serr == nil {
		go h.waitTask(cmd)
		go h.readyTask()
	}
	// The parent's copies of the ready write end and the release read end,
	// once each, after H's Start returned.
	h.closeEnds("ready-w", "rel-r")
	if serr != nil {
		h.closeEnds("ready-r", "rel-w")
		c.prepFail(serr)
		return
	}
	if releaseNow {
		h.closeEnd("rel-w")
	}
	ch, stop := guardCh()
	defer stop()
	for {
		h.mu.Lock()
		r, changed := h.ready, h.changed
		h.mu.Unlock()
		if r != "running" {
			if r != "ok" {
				c.prepFail(fmt.Errorf("the holder is not ready: %s", r))
			}
			return
		}
		select {
		case <-changed:
		case <-ch:
			c.prepFail(errors.New("the holder did not report ready within the guard"))
			return
		}
	}
}

// readyTask reads H's fixed ready line, publishes the result, then closes
// the ready read end once; it is a necessary task until both are done.
func (h *holder) readyTask() {
	f := h.file("ready-r")
	b := make([]byte, len(pseudo.ReadyLine))
	n, err := io.ReadFull(f, b)
	h.mu.Lock()
	switch {
	case err != nil:
		h.ready = "error"
		h.faultLocked("ready read", err)
	case string(b[:n]) != pseudo.ReadyLine:
		h.ready = "wrong"
	default:
		h.ready = "ok"
	}
	h.publishLocked()
	h.mu.Unlock()
	h.closeEnd("ready-r")
	h.mu.Lock()
	h.tasks--
	h.publishLocked()
	h.mu.Unlock()
}

// waitTask is H's one Wait.
func (h *holder) waitTask(cmd *exec.Cmd) {
	werr := cmd.Wait()
	h.mu.Lock()
	h.wait = "done"
	if ps := cmd.ProcessState; ps != nil {
		h.exit = ps.ExitCode()
	}
	if werr != nil {
		h.faultLocked("wait", werr)
	}
	h.tasks--
	h.publishLocked()
	h.mu.Unlock()
}

// stop is the outer cleanup's part, after the tested decision was saved:
// no new holder work starts after it; when H has started, the cleanup makes
// the one release request; when H's Start is in flight, the start path will.
func (h *holder) stop() {
	h.mu.Lock()
	h.stopped = true
	take := h.start == hStarted && h.release == "none"
	if take {
		h.release, h.saved = "by-cleanup", h.c.decisionSaved()
	}
	h.publishLocked()
	h.mu.Unlock()
	if take {
		h.closeEnd("rel-w")
	}
}

func (h *holder) terminalLocked() bool {
	if h.start == hPreparing || h.start == hStarting || h.tasks != 0 {
		return false
	}
	for _, e := range h.ends {
		if e.state != endClosed {
			return false
		}
	}
	return h.start != hStarted || (h.release != "none" && h.wait == "done")
}

// waitTerminal waits, at most d, until H is terminal.
func (h *holder) waitTerminal(d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	for {
		h.mu.Lock()
		done, ch := h.terminalLocked(), h.changed
		h.mu.Unlock()
		if done {
			return true
		}
		select {
		case <-ch:
		case <-t.C:
			return false
		}
	}
}

func (h *holder) terminal() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.terminalLocked()
}

// lines are H's canonical lines; its faults are in faultSet.
func (h *holder) lines() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := []string{"holder start " + h.start, "holder ready " + h.ready}
	w := "holder wait " + h.wait
	if h.wait == "done" {
		w += fmt.Sprintf(" exit %d", h.exit)
	}
	out = append(out, w, "holder release "+h.release+" after-saved-decision "+yesNo(h.saved))
	for _, n := range hEndNames {
		e := h.ends[n]
		if e == nil {
			out = append(out, "holder "+n+" none")
			continue
		}
		l := "holder " + n + " " + e.state
		if e.state == endClosed {
			if e.err == nil {
				l += " ok"
			} else {
				l += " error"
			}
		}
		out = append(out, l)
	}
	return append(out, fmt.Sprintf("holder tasks %d", h.tasks))
}

func (h *holder) faultSet() []Fault {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]Fault(nil), h.faults...)
}
