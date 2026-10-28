//go:build windows

package wp25

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// p1Win is one P-1 run on Windows: the main process in a Job of its own,
// created suspended and resumed (startJob); E-main is a wait on the
// Runner's process handle.
type p1Win struct {
	p1Common
	tr        *winTree
	outerProc windows.Handle // the outer's own handle, opened at the start
	outerJob  windows.Handle // alive: the outer's second handle to the tested Job
	handed    windows.Handle // the Runner's process handle, handed to the inventory
	jobClose  string
	procClose string
	reaped    bool
}

func p1Inner(s *scen, h *host, variant string) p1Run {
	r := &p1Win{}
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
	tr, step, err := startJob(s, cmd, jobSeams{})
	if step != "" {
		s.invalidf("start failed at %s: %v", step, err)
	}
	r.tr, r.cmd, r.PID = tr, cmd, tr.pid
	s.defer_(func() { _, _ = tr.close() })
	oh, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(r.PID))
	if err != nil {
		s.invalidf("the outer's handle to %d: %v", r.PID, err)
	}
	r.outerProc = oh
	s.defer_(func() { _ = windows.CloseHandle(oh) })
	if variant == "alive" {
		// A construction that changes a premise: with a second handle the
		// Job outlives the inner call's close, so KILL_ON_JOB_CLOSE does
		// not end the process there (docs/wp2.5-prototype.md §14.3).
		cur := windows.CurrentProcess()
		var dup windows.Handle
		if err := windows.DuplicateHandle(cur, tr.job, cur, &dup, 0, false, windows.DUPLICATE_SAME_ACCESS); err != nil {
			s.invalidf("the outer's handle to the Job: %v", err)
		}
		r.outerJob = dup
		s.defer_(func() {
			if r.outerJob != 0 {
				_ = windows.TerminateJobObject(r.outerJob, 1)
				_ = windows.CloseHandle(r.outerJob)
			}
		})
		r.Facts = append(r.Facts, "construction: the outer holds a second handle to the tested Job, so KILL_ON_JOB_CLOSE does not act when the inner call closes its own")
	}
	// The fixture's release point.
	switch variant {
	case "alive":
		s.awaitReport(rep, "pid", 1, 5*time.Second)
		r.Facts = append(r.Facts, "fixture: the main process reported itself running")
	case "exited":
		ev, werr := windows.WaitForSingleObject(oh, 5000)
		if werr != nil || ev != windows.WAIT_OBJECT_0 {
			s.invalidf("load not established: the main process did not exit within 5 s (%v)", werr)
		}
		r.Facts = append(r.Facts, "fixture: the main process had exited (the outer's handle signalled) before the observation")
	}
	// S2: for the faulted variants the first observation fails (C-19).
	failed := false
	if variant == "healthy" {
		ev, werr := windows.WaitForSingleObject(tr.proc, 10000)
		if injectOnce("p1-observe-fail-once", "p1Inner.observe", r.PID) {
			werr = errors.New("injected: the observation failed once") // the real wait was made
		}
		failed = werr != nil || ev != windows.WAIT_OBJECT_0
		if !failed {
			var code uint32
			if gerr := windows.GetExitCodeProcess(tr.proc, &code); gerr != nil {
				failed = true
			} else {
				r.Status = "exit " + strconv.Itoa(int(code))
			}
		}
	} else {
		r.ObsHits++
		failed = true
	}
	// S3, then S4 and S5 within T_kill: the end is ActiveProcesses = 0.
	confirmed := false
	deadline := time.Now().Add(tKill)
	for {
		if variant == "alive" {
			r.KillHits++ // the request is swallowed: reported sent, never sent
		} else {
			_ = windows.TerminateJobObject(tr.job, 1)
		}
		n, qerr := tr.active()
		if qerr != nil {
			s.flag("observation failed: the Job's active processes: %v", qerr)
			failed = true
			break
		}
		if n == 0 {
			confirmed = true
			break
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if failed || !confirmed {
		// F1 (spec §19.3, §19.7): the Job handle closed (which asks for
		// the end again), the Runner's process handle not closed but
		// handed to the inventory with the Cmd, no Wait; the host
		// disabled (Q-1).
		r.Class = "F1"
		r.jobClose = errOK(windows.CloseHandle(tr.job))
		tr.job = 0
		r.handed, tr.proc = tr.proc, 0
		h.handOver(invItem{kind: "unconfirmed-process", cmd: cmd, handle: uintptr(r.handed), pid: r.PID})
		return r
	}
	r.Class = "normal"
	pe, je := tr.close()
	r.procClose, r.jobClose = errOK(pe), errOK(je)
	r.reaped = tr.reap(time.Second)
	return r
}

func (r *p1Win) closed(*scen) string { return "" }

func (r *p1Win) branch(s *scen, variant string) (bool, string) {
	if variant == "healthy" {
		return r.reaped, fmt.Sprintf("reaped by the inner call after its exit was observed: %v", r.reaped)
	}
	ev, err := windows.WaitForSingleObject(r.outerProc, 0)
	if err != nil {
		s.flag("observation failed: the outer's handle: %v", err)
		return false, "the outer's handle cannot be waited on"
	}
	if variant == "alive" {
		if ev != uint32(windows.WAIT_TIMEOUT) {
			return false, "the outer's handle is signalled: the process has ended"
		}
		in, ierr := isProcessInJob(r.outerProc, r.outerJob)
		if ierr != nil {
			s.flag("observation failed: whether the process is in the Job: %v", ierr)
			return false, "the Job membership cannot be read"
		}
		return in, fmt.Sprintf("alive: the outer's handle not signalled; in the tested Job: %v", in)
	}
	if ev != windows.WAIT_OBJECT_0 {
		return false, "the outer's handle is not signalled: the process runs"
	}
	var code uint32
	if err := windows.GetExitCodeProcess(r.outerProc, &code); err != nil {
		s.flag("observation failed: the exit code: %v", err)
		return false, "the exit code cannot be read"
	}
	return code == 7, fmt.Sprintf("exited: the outer's handle signalled, exit code %d", code)
}

func (r *p1Win) item(it invItem) (bool, string) {
	h := windows.Handle(it.handle)
	ev, err := windows.WaitForSingleObject(h, 0)
	if injectOnce("p1-item-wait-fail-once", "p1.item-wait", r.PID) {
		err = errors.New("injected: the wait on the handed handle failed once") // the real call was made
	}
	valid := err == nil && (ev == windows.WAIT_OBJECT_0 || ev == uint32(windows.WAIT_TIMEOUT))
	ok := it.kind == "unconfirmed-process" && it.cmd == r.cmd && it.pid == r.PID && h == r.handed && h != 0 && valid
	return ok, fmt.Sprintf("kind %s, the run's Cmd %v, the Runner's handle %v, a zero wait on it %s", it.kind, it.cmd == r.cmd, h == r.handed && h != 0, waitResult(ev, err))
}

func waitResult(ev uint32, err error) string {
	switch {
	case err != nil:
		return "failed: " + err.Error()
	case ev == windows.WAIT_OBJECT_0:
		return "signalled"
	case ev == uint32(windows.WAIT_TIMEOUT):
		return "not signalled"
	}
	return fmt.Sprintf("returned %#x", ev)
}

func (r *p1Win) extra(s *scen, variant string) {
	s.expect("the Runner's Job handle closed", "ok", r.jobClose)
	if variant == "healthy" {
		s.expect("the Runner's process handle closed", "ok", r.procClose)
	}
}

// remedy ends what is left through the outer's own handle to the Job and
// confirms it empty, then reaps the main process.
func (r *p1Win) remedy(_ *scen, _ string) error {
	if r.Class != "F1" {
		return nil
	}
	if r.outerJob != 0 {
		if err := windows.TerminateJobObject(r.outerJob, 1); err != nil {
			return fmt.Errorf("terminating the Job: %w", err)
		}
		deadline := time.Now().Add(10 * time.Second)
		for {
			var a basicAccounting
			if err := windows.QueryInformationJobObject(r.outerJob, windows.JobObjectBasicAccountingInformation,
				uintptr(unsafe.Pointer(&a)), uint32(unsafe.Sizeof(a)), nil); err != nil {
				return fmt.Errorf("the Job's active processes: %w", err)
			}
			if a.ActiveProcesses == 0 {
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("the Job still has %d active processes", a.ActiveProcesses)
			}
			time.Sleep(5 * time.Millisecond)
		}
		err := windows.CloseHandle(r.outerJob)
		r.outerJob = 0
		if err != nil {
			return fmt.Errorf("closing the outer's Job handle: %w", err)
		}
	}
	if !childOf(r.cmd).reap(10 * time.Second) {
		return errors.New("the main process was not reaped within 10 s")
	}
	return nil
}

// release closes the Runner's handle held by the inventory, as the
// host's exit would.
func (r *p1Win) release(it invItem) error {
	return windows.CloseHandle(windows.Handle(it.handle))
}

// The baseline barrier is Linux's; Windows has none.
func p1Barrier(*scen) (bool, string) { return false, "no baseline barrier on this platform" }
