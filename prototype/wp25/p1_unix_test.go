//go:build linux || darwin

package wp25

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// p1Unix is one P-1 run on Linux or macOS: the main process leads its
// own group; E-main is the platform's observer (pidfd or kqueue).
type p1Unix struct {
	p1Common
	start    string // the main process's start time, read at its start
	obsFD    int    // the observer's descriptor, closed by the inner call
	subAfter int    // Linux: the subreaper after the inner call; -1 elsewhere
	reaped   bool
}

func p1Inner(s *scen, h *host, variant string) p1Run {
	r := &p1Unix{subAfter: -1, obsFD: -1}
	var cmd *exec.Cmd
	rep := ""
	switch variant {
	case "alive":
		rep = s.newReport(reportSpec{"pid": 1})
		cmd = helper("sleep", []string{reportEnv + "=" + rep})
	case "exited":
		cmd = helper("exit-after", nil, "0", "7")
	default:
		cmd = helper("exit-after", nil, "100", "0")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := p1Exclusive(func() error {
		startHelper(s, cmd)
		r.cmd, r.PID = cmd, cmd.Process.Pid
		st, pr := procInfo(r.PID)
		if pr != present {
			s.invalidf("the main process %d cannot be read", r.PID)
		}
		r.start = st.start
		obs, err := newExitObserver(r.PID)
		if err != nil {
			s.invalidf("exit observer: %v", err)
		}
		r.obsFD = obsFD(obs)
		// The fixture's release point.
		switch variant {
		case "alive":
			s.awaitReport(rep, "pid", 1, 5*time.Second)
			r.Facts = append(r.Facts, "fixture: the main process reported itself running")
		case "exited":
			if !p1AwaitZombie(r.PID, r.start, 5*time.Second) {
				obs.close()
				s.invalidf("load not established: the main process did not exit within 5 s")
			}
			r.Facts = append(r.Facts, "fixture: the main process had exited (an unreaped zombie, same start time) before the observation")
		}
		// S2: for the faulted variants the first observation fails, and
		// that cannot be undone (C-19): the run goes to S3 and ends in F1.
		failed := false
		if variant == "healthy" {
			ok, status, oerr := obs.wait(10 * time.Second)
			r.Status = status
			failed = oerr != nil || !ok
		} else {
			r.ObsHits++
			failed = true
		}
		// S3, then S4 and S5 within T_kill. The leader is never reaped
		// before the end is confirmed, so the group ID is this run's.
		confirmed := false
		deadline := time.Now().Add(tKill)
		for {
			if variant == "alive" {
				r.KillHits++ // the request is swallowed: reported sent, never sent
			} else {
				_ = syscall.Kill(-r.PID, syscall.SIGKILL)
			}
			m, gerr := groupLive(r.PID)
			if gerr != nil {
				s.flag("observation failed: the members of group %d: %v", r.PID, gerr)
				failed = true
				break
			}
			if len(m) == 0 {
				confirmed = true
				break
			}
			if time.Now().After(deadline) {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		obs.close()
		if failed || !confirmed {
			// F1 (spec §19.3): no Wait, the observer closed, the process
			// handed to the inventory with the host disabled (Q-1).
			r.Class = "F1"
			h.handOver(invItem{kind: "unconfirmed-process", cmd: cmd, pid: r.PID})
			return nil
		}
		r.Class = "normal"
		r.reaped = childOf(cmd).reap(time.Second)
		return nil
	})
	if err != nil {
		s.invalidf("the exclusive interval: %v", err)
	}
	r.subAfter = p1Subreaper(s)
	return r
}

// p1AwaitZombie waits at most d for pid to be an unreaped zombie child of
// this process with the given start time.
func p1AwaitZombie(pid int, start string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for {
		st, pr := procInfo(pid)
		if pr == present && st.zombie && st.start == start && st.ppid == os.Getpid() {
			return true
		}
		if pr != present || time.Now().After(deadline) {
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (r *p1Unix) closed(s *scen) string {
	ok, detail, known := fdClosed(r.obsFD)
	if !known {
		s.flag("observation failed: whether the observer's descriptor %d is closed: %s", r.obsFD, detail)
	}
	s.expectTrue("the E-main observer's descriptor is closed", ok, detail)
	return detail
}

func (r *p1Unix) branch(s *scen, variant string) (bool, string) {
	if variant == "healthy" {
		return r.reaped, fmt.Sprintf("reaped by the inner call after its exit was observed: %v", r.reaped)
	}
	st, pr := procInfo(r.PID)
	switch {
	case pr == unknown:
		s.flag("observation failed: the state of the main process %d cannot be read", r.PID)
		return false, "the main process's state cannot be read"
	case pr == absent:
		return false, "the main process is gone"
	case st.start != r.start:
		return false, "the PID names another process now"
	case st.ppid != os.Getpid():
		return false, fmt.Sprintf("its parent is %d, not this process", st.ppid)
	case variant == "alive" && st.zombie:
		return false, "a zombie: it has exited"
	case variant == "alive":
		return true, fmt.Sprintf("alive: pid %d, the same start time, this process its parent, not a zombie", r.PID)
	case !st.zombie:
		return false, "running: it has not exited"
	}
	return true, fmt.Sprintf("exited: pid %d is this process's unreaped zombie, the same start time", r.PID)
}

func (r *p1Unix) item(it invItem) (bool, string) {
	ok := it.kind == "unconfirmed-process" && it.cmd == r.cmd && it.pid == r.PID && it.handle == 0
	return ok, fmt.Sprintf("kind %s, the run's Cmd %v, pid %d", it.kind, it.cmd == r.cmd, it.pid)
}

func (r *p1Unix) extra(s *scen, _ string) {
	if r.subAfter >= 0 {
		s.expect("subreaper after the inner call", 0, r.subAfter)
	}
}

// remedy ends the main process after checking its identity (its group
// while the leader is held), then reaps it.
func (r *p1Unix) remedy(_ *scen, _ string) error {
	if r.Class != "F1" {
		return nil
	}
	c := childOf(r.cmd)
	st, pr := procInfo(r.PID)
	switch {
	case pr == unknown:
		return errors.New("the main process's state cannot be read")
	case pr == present && st.start == r.start && st.ppid == os.Getpid():
		_ = syscall.Kill(-r.PID, syscall.SIGKILL)
		_ = syscall.Kill(r.PID, syscall.SIGKILL)
	}
	if !c.reap(10 * time.Second) {
		return errors.New("the main process was not reaped within 10 s")
	}
	return nil
}

func (r *p1Unix) release(invItem) error { return nil }
