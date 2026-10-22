//go:build linux || darwin

package wp25

// E-main and the group identity rule on Unix (T1 on Linux, T3 on macOS):
// observe the main process's exit with a bounded wait that does not reap
// it, keep the leader unreaped while the group is signalled, reap only
// after the group is confirmed empty, and never signal it after that.

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func gate13() string {
	if runtime.GOOS == "darwin" {
		return "T3"
	}
	return "T1"
}

func TestEMainExitUnreaped(t *testing.T) {
	s := begin(t, gate13(), "emain.exit-unreaped", normal, true, true)
	defer s.done()
	cmd := helper("exit-after", nil, "300", "7")
	tr := startTree(s, cmd)
	g0 := runtime.NumGoroutine()
	t0 := time.Now()
	exited, status, err := tr.waitExit(3 * time.Second)
	took := time.Since(t0)
	s.worked(1)
	s.expect("exit observed", true, exited)
	s.expect("status", "exit 7", status)
	s.expectTrue("observed near the exit", err == nil && took < 2*time.Second, took.String())
	s.expect("leader unreaped (zombie)", true, tr.zombieLeader())
	s.expect("leader's group unchanged", tr.pid, tr.leaderPgid())
	s.expectTrue("no goroutine left waiting", runtime.NumGoroutine() <= g0, fmt.Sprintf("%d -> %d", g0, runtime.NumGoroutine()))
	// Recorded, not predicted: the safety of the rule does not depend on
	// it, since only a process with this PID can make a new group with
	// this ID, and the unreaped leader holds the PID.
	s.note("kill(-pgid, 0) with only the zombie leader: %s", errText(syscall.Kill(-tr.pid, 0)))
	s.expect("reaped", true, tr.reap(time.Second))
	s.expect("exit code after reaping", 7, cmd.ProcessState.ExitCode())
	s.expect("kill(-pgid, 0) after reaping", "no such process", errText(syscall.Kill(-tr.pid, 0)))
	s.expect("group signal refused after reaping", true, errors.Is(tr.kill(), errGroupGone))
}

func TestEMainBoundedTimeout(t *testing.T) {
	s := begin(t, gate13(), "emain.bounded-timeout", normal, true, true)
	defer s.done()
	cmd := helper("sleep", nil)
	tr := startTree(s, cmd)
	g0 := runtime.NumGoroutine()
	t0 := time.Now()
	exited, _, err := tr.waitExit(500 * time.Millisecond)
	took := time.Since(t0)
	s.worked(1)
	s.expect("exit observed", false, exited)
	s.expectTrue("returned at the limit", err == nil && took >= 500*time.Millisecond && took < 1500*time.Millisecond, took.String())
	s.expectTrue("no goroutine left waiting", runtime.NumGoroutine() <= g0, fmt.Sprintf("%d -> %d", g0, runtime.NumGoroutine()))
	s.expect("kill(-pgid) while the leader runs", "no error", errText(tr.kill()))
	exited, status, _ := tr.waitExit(3 * time.Second)
	s.expect("exit observed after the kill", true, exited)
	s.expect("status", "signal 9", status)
	s.expect("group confirmed empty", true, tr.confirm(tKill))
	s.expect("reaped", true, tr.reap(time.Second))
}

func TestGroupZombieLeader(t *testing.T) {
	// The leader exits while a member lives: the unreaped leader keeps the
	// group ID naming this group; the kill reaches the member; the leader
	// is reaped only after the group is confirmed empty.
	s := begin(t, gate13(), "group.zombie-leader", normal, true, true)
	defer s.done()
	rep := s.newReport()
	cmd := helper("member-parent", []string{reportEnv + "=" + rep})
	tr := startTree(s, cmd)
	exited, status, _ := tr.waitExit(5 * time.Second)
	members := waitReport(rep, "pid", 1, 5*time.Second)
	if len(members) != 1 {
		s.invalidf("member did not report")
	}
	if p, err := ownPID(members[0]); err == nil {
		s.own(p)
	}
	live, _ := tr.live()
	s.worked(int64(len(live)))
	s.expect("leader exited", true, exited)
	s.expect("status", "exit 0", status)
	s.expect("leader unreaped (zombie)", true, tr.zombieLeader())
	s.expect("live members besides the leader", fmt.Sprint(members), fmt.Sprint(live))
	s.expect("kill(-pgid, 0)", "no error", errText(syscall.Kill(-tr.pid, 0)))
	s.expect("group confirmed empty after the kill", true, tr.confirm(tKill))
	s.expect("leader still unreaped after the kill", true, tr.zombieLeader())
	s.expect("reaped", true, tr.reap(time.Second))
	s.expect("kill(-pgid, 0) after reaping", "no such process", errText(syscall.Kill(-tr.pid, 0)))
}

func TestGroupEscape(t *testing.T) {
	// Boundary 1 of §8.2: a descendant that leaves the group (setsid) is
	// not reached by the group kill, and the group still reads as empty.
	// On macOS there is no subreaper to find it.
	s := begin(t, gate13(), "group.escape-boundary", normal, true, true)
	defer s.done()
	rep := s.newReport()
	cmd := helper("escape-parent", []string{reportEnv + "=" + rep})
	tr := startTree(s, cmd)
	exited, _, _ := tr.waitExit(5 * time.Second)
	esc := waitReport(rep, "pid", 1, 5*time.Second)
	if len(esc) != 1 {
		s.invalidf("escaped process did not report")
	}
	p, err := ownPID(esc[0])
	if err != nil {
		s.invalidf("escaped process: %v", err)
	}
	s.own(p)
	conf := tr.confirm(tKill)
	s.worked(1)
	s.expect("leader exited", true, exited)
	s.expect("group confirmed empty", true, conf)
	s.expect("escaped process still alive", false, p.ended())
	st, _ := procInfo(esc[0])
	s.expectTrue("it is in another group", st.pgid != tr.pid, fmt.Sprintf("pgid %d vs %d", st.pgid, tr.pid))
	s.expect("reaped", true, tr.reap(time.Second))
}

func TestWaitidNoWait(t *testing.T) {
	// waitid(P_PID, WEXITED|WNOWAIT) gives the status and leaves the
	// process unreaped (on macOS through the raw system call).
	s := begin(t, gate13(), "emain.waitid-wnowait", normal, true, true)
	defer s.done()
	cmd := helper("exit-after", nil, "100", "5")
	tr := startTree(s, cmd)
	if ok, _, _ := tr.waitExit(3 * time.Second); !ok {
		s.invalidf("helper did not exit")
	}
	st, err := waitidNoWait(tr.pid)
	if errors.Is(err, syscall.ENOSYS) || errors.Is(err, syscall.EINVAL) {
		s.unsupported("waitid: %v", err)
	}
	s.worked(1)
	s.expect("waitid error", "no error", errText(err))
	s.expect("status", "exit 5", st)
	s.expect("still unreaped", true, tr.zombieLeader())
	st2, err2 := waitidNoWait(tr.pid)
	s.expect("a second waitid sees the same", "exit 5 no error", st2+" "+errText(err2))
	s.expect("reaped", true, tr.reap(time.Second))
	s.expect("exit code", 5, cmd.ProcessState.ExitCode())
}

func errText(err error) string {
	switch {
	case err == nil:
		return "no error"
	case errors.Is(err, syscall.ESRCH):
		return "no such process"
	case errors.Is(err, syscall.EPERM):
		return "not permitted"
	}
	return err.Error()
}
