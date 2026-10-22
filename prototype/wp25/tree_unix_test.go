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

func (c *childProc) ended() bool { return c.reaped() }

// stop kills the child: its PID cannot have been reused, it is not reaped.
func (c *childProc) stop() {
	if !c.reaped() {
		_ = syscall.Kill(c.cmd.Process.Pid, syscall.SIGKILL)
		c.reap(5 * time.Second)
	}
}

// pidProc is a process learned from a helper's report; its identity is
// its start time, read when it was learned and again before every action.
type pidProc struct {
	pid int
	id  string
	err error // the last reaping error, for the report
}

func ownPID(pid int) (owned, error) {
	st, ok := procInfo(pid)
	if !ok {
		return nil, fmt.Errorf("process %d is not there", pid)
	}
	return &pidProc{pid: pid, id: st.start}, nil
}

func (p *pidProc) String() string {
	if p.err != nil {
		return "pid " + strconv.Itoa(p.pid) + " (reap: " + p.err.Error() + ")"
	}
	return "pid " + strconv.Itoa(p.pid)
}

func (p *pidProc) ended() bool {
	st, ok := procInfo(p.pid)
	if !ok || st.start != p.id {
		return true
	}
	if st.zombie {
		if st.ppid != os.Getpid() {
			return true // someone else's zombie: it has ended
		}
		// A zombie child of this process, one the scenario owns: it has
		// ended once exactly this PID is reaped.
		var ws syscall.WaitStatus
		wpid, err := syscall.Wait4(p.pid, &ws, syscall.WNOHANG, nil)
		p.err = err
		return err == nil && wpid == p.pid
	}
	return false
}

func (p *pidProc) stop() {
	if st, ok := procInfo(p.pid); ok && st.start == p.id && !st.zombie {
		_ = syscall.Kill(p.pid, syscall.SIGKILL)
	}
}

type pinfo struct {
	zombie     bool
	ppid, pgid int
	start      string
}

// unixTree is one check's process group; the leader is the main process.
type unixTree struct {
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
	t := &unixTree{cmd: cmd, c: childOf(cmd), pid: cmd.Process.Pid}
	if st, ok := procInfo(t.pid); ok {
		t.start = st.start
	}
	obs, err := newExitObserver(t.pid)
	if err != nil {
		s.invalidf("exit observer: %v", err)
	}
	t.obs = obs
	s.defer_(obs.close)
	return t
}

// waitExit observes the main process's exit within d, without reaping it.
func (t *unixTree) waitExit(d time.Duration) (bool, string, error) { return t.obs.wait(d) }

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
	st, ok := procInfo(t.pid)
	return ok && st.start == t.start && st.ppid == os.Getpid()
}

// live lists the group's members that are not zombies.
func (t *unixTree) live() ([]int, error) { return groupLive(t.pid) }

// confirm repeats the kill until the group has no live member, within d.
func (t *unixTree) confirm(d time.Duration) bool {
	deadline := time.Now().Add(d)
	for {
		m, err := t.live()
		if err == nil && len(m) == 0 {
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
	st, ok := procInfo(t.pid)
	return ok && st.zombie
}

func (t *unixTree) leaderPgid() int {
	st, _ := procInfo(t.pid)
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
