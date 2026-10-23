//go:build linux || darwin

package wp25

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// presence is what a read of a process's state says about it. A state
// that cannot be read is unknown, never absent.
type presence int

const (
	absent presence = iota
	present
	unknown
)

var errUnknownState = errors.New("the process's state could not be read")

// childProc is a direct child; it is reaped once, by reap, and never
// before the scenario asks for it (the group identity rule, spec §8.2).
type childProc struct {
	cmd  *exec.Cmd
	once sync.Once
	done chan struct{}
	err  error
}

func ownChild(cmd *exec.Cmd) owned { return childOf(cmd) }

var (
	childrenMu sync.Mutex
	children   = map[*exec.Cmd]*childProc{}
)

func childOf(cmd *exec.Cmd) *childProc {
	childrenMu.Lock()
	defer childrenMu.Unlock()
	if c, ok := children[cmd]; ok {
		return c
	}
	c := &childProc{cmd: cmd, done: make(chan struct{})}
	children[cmd] = c
	return c
}

func (c *childProc) String() string { return "child " + strconv.Itoa(c.cmd.Process.Pid) }

// reap waits (and reaps) at most d; the Wait goroutine stays only while
// the process lives, and stop ends it.
func (c *childProc) reap(d time.Duration) bool {
	c.once.Do(func() {
		go func() {
			c.err = c.cmd.Wait()
			close(c.done)
		}()
	})
	select {
	case <-c.done:
		return true
	case <-time.After(d):
		return false
	}
}

func (c *childProc) reaped() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

func (c *childProc) ended() bool       { return c.reaped() }
func (c *childProc) release()          {}
func (c *childProc) observeErr() error { return nil }
func (c *childProc) pidOf() int {
	if c.cmd.Process == nil {
		return 0
	}
	return c.cmd.Process.Pid
}
func (c *childProc) identity() string { return idKey(c.pidOf(), identityOf(c.cmd)) }

// stop kills the child: its PID cannot have been reused, it is not reaped.
func (c *childProc) stop() {
	if !c.reaped() {
		_ = syscall.Kill(c.cmd.Process.Pid, syscall.SIGKILL)
		c.reap(5 * time.Second)
	}
}

// pidProc is a reported process the scenario verified as its own; its
// identity is its start time, read when it was verified and again before
// every action.
type pidProc struct {
	pid      int
	id       string
	err      error // why the last check could not confirm an end
	obsErr   error // the first observation that failed; never cleared
	injected bool  // the observe-fail-once fault has acted on it
}

// ownDescendant verifies a reported PID before the scenario may act on
// it: the process must be in the group whose leader (anchor) the scenario
// started and still holds unreaped, or its chain of parents must reach
// the anchor. The PID in a report is never enough on its own.
func ownDescendant(s *scen, anchor *unixTree, pid int) owned {
	st, pr := procInfo(pid)
	if pr != present {
		s.invalidf("ownership not established: process %d cannot be read", pid)
	}
	if st.pgid == anchor.pid && anchor.leaderHeld() {
		p := &pidProc{pid: pid, id: st.start}
		s.own(p)
		return p
	}
	cur := st
	for i := 0; i < 32; i++ {
		if cur.ppid == anchor.pid {
			p := &pidProc{pid: pid, id: st.start}
			s.own(p)
			return p
		}
		if cur.ppid <= 1 {
			break
		}
		next, pr := procInfo(cur.ppid)
		if pr == unknown {
			s.invalidf("ownership not established: the state of %d, a parent of %d, cannot be read", cur.ppid, pid)
		}
		if pr != present {
			break
		}
		cur = next
	}
	s.invalidf("ownership not established: process %d is not a descendant of %d", pid, anchor.pid)
	return nil
}

func (p *pidProc) String() string {
	if p.err != nil {
		return "pid " + strconv.Itoa(p.pid) + " (" + p.err.Error() + ")"
	}
	return "pid " + strconv.Itoa(p.pid)
}

func (p *pidProc) ended() bool {
	if injected("cleanup-unconfirmed") {
		return false
	}
	st, pr := procInfo(p.pid)
	if injected("procinfo-unknown") {
		pr = unknown // the read of the process's state failed
	}
	if injected("observe-fail-once") && !p.injected {
		p.injected = true // this one read fails; the next ones do not
		injectHit("pidProc.ended", p.pid)
		pr = unknown
	}
	switch {
	case pr == absent:
		return true
	case pr == unknown:
		p.err = errUnknownState
		if p.obsErr == nil {
			p.obsErr = errUnknownState
		}
		return false
	case st.start != p.id:
		return true // the PID names another process now: this one has ended
	case st.zombie && st.ppid != os.Getpid():
		return true // someone else's zombie: it has ended
	case st.zombie:
		// A zombie child of this process, one the scenario owns: it has
		// ended once exactly this PID is reaped.
		var ws syscall.WaitStatus
		wpid, err := syscall.Wait4(p.pid, &ws, syscall.WNOHANG, nil)
		p.err = err
		p.failed(err) // a reap that fails is not an end, and not a "not yet"
		return err == nil && wpid == p.pid
	}
	return false
}

// failed keeps the first observation that failed.
func (p *pidProc) failed(err error) {
	if err != nil && p.obsErr == nil {
		p.obsErr = err
	}
}

func (p *pidProc) stop() {
	st, pr := procInfo(p.pid)
	if pr == unknown {
		p.failed(errUnknownState) // not signalled: its identity cannot be checked
	}
	if pr == present && st.start == p.id && !st.zombie {
		_ = syscall.Kill(p.pid, syscall.SIGKILL)
	}
}

func (p *pidProc) release()          {}
func (p *pidProc) pidOf() int        { return p.pid }
func (p *pidProc) identity() string  { return idKey(p.pid, p.id) }
func (p *pidProc) observeErr() error { return p.obsErr }

type pinfo struct {
	zombie     bool
	ppid, pgid int
	start      string
}

// unixTree is one check's process group; the leader is the main process.
type unixTree struct {
	s     *scen
	cmd   *exec.Cmd
	c     *childProc
	pid   int
	start string // the leader's start time
	obs   exitObserver
}

var errGroupGone = errors.New("the group leader is reaped: no more signals to the group")

func startTree(s *scen, cmd *exec.Cmd) *unixTree {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	startHelper(s, cmd)
	t := &unixTree{s: s, cmd: cmd, c: childOf(cmd), pid: cmd.Process.Pid}
	st, pr := procInfo(t.pid)
	if pr != present {
		s.invalidf("the main process %d cannot be read", t.pid)
	}
	t.start = st.start
	obs, err := newExitObserver(t.pid)
	if err != nil {
		s.invalidf("exit observer: %v", err)
	}
	t.obs = obs
	s.defer_(obs.close)
	// If the scenario stops early, its group is still its own while the
	// leader is held: end the group before the leader is reaped.
	s.defer_(func() {
		if t.c.reaped() || !t.leaderHeld() {
			return
		}
		_ = syscall.Kill(-t.pid, syscall.SIGKILL)
		deadline := time.Now().Add(tKill)
		for {
			m, err := t.live()
			if err != nil {
				s.flag("observation failed while ending the group %d: %v", t.pid, err)
				return
			}
			if len(m) == 0 {
				return
			}
			if time.Now().After(deadline) {
				s.flag("cleanup: the group %d could not be confirmed empty", t.pid)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	})
	return t
}

// ownReported verifies and owns a PID its helpers reported.
func (t *unixTree) ownReported(pid int) owned { return ownDescendant(t.s, t, pid) }

// waitExit observes the main process's exit within d, without reaping it.
// A wait that fails is an observation failure of the run, whatever the
// caller makes of the answer.
func (t *unixTree) waitExit(d time.Duration) (bool, string, error) {
	ok, st, err := t.obs.wait(d)
	if err != nil {
		t.s.flag("observation failed: waiting for the main process %d: %v", t.pid, err)
	}
	return ok, st, err
}

// kill signals the whole group; only while the leader is unreaped, so the
// group ID still names this check's group.
func (t *unixTree) kill() error {
	if t.c.reaped() || !t.leaderHeld() {
		return errGroupGone
	}
	return syscall.Kill(-t.pid, syscall.SIGKILL)
}

// leaderHeld reports whether the leader's PID still names this process's
// own, unreaped child: same start time, this process its parent. It
// detects a leader reaped by someone else, but only up to the moment of
// the check; a concurrent reaper can still act between the check and the
// signal (T2.other-reaper), which is why no other reaper may exist.
func (t *unixTree) leaderHeld() bool {
	st, pr := procInfo(t.pid)
	if pr == unknown {
		t.s.flag("observation failed: the state of the leader %d cannot be read", t.pid)
	}
	return pr == present && st.start == t.start && st.ppid == os.Getpid()
}

// live lists the group's members that are not zombies.
func (t *unixTree) live() ([]int, error) { return groupLive(t.pid) }

// confirm repeats the kill until the group has no live member, within d.
// A member list that cannot be read is not an empty group: the scenario
// cannot observe what it predicts, so it is invalid.
func (t *unixTree) confirm(d time.Duration) bool {
	deadline := time.Now().Add(d)
	for {
		m, err := t.live()
		if err != nil {
			t.s.invalidf("observation failed: %v", err)
		}
		if len(m) == 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		_ = t.kill()
		time.Sleep(5 * time.Millisecond)
	}
}

func (t *unixTree) reap(d time.Duration) bool { return t.c.reap(d) }

func (t *unixTree) zombieLeader() bool {
	st, pr := procInfo(t.pid)
	if pr == unknown {
		t.s.flag("observation failed: the state of the leader %d cannot be read", t.pid)
	}
	return pr == present && st.zombie
}

func (t *unixTree) leaderPgid() int {
	st, pr := procInfo(t.pid)
	if pr == unknown {
		t.s.flag("observation failed: the state of the leader %d cannot be read", t.pid)
	}
	return st.pgid
}

func ignoreSIGPIPE() { signal.Ignore(syscall.SIGPIPE) }

// waitStatus describes a wait status.
func waitStatus(ws syscall.WaitStatus) string {
	switch {
	case ws.Exited():
		return "exit " + strconv.Itoa(ws.ExitStatus())
	case ws.Signaled():
		return "signal " + strconv.Itoa(int(ws.Signal()))
	}
	return fmt.Sprintf("status %#x", uint32(ws))
}
