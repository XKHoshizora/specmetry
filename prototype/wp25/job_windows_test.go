//go:build windows

package wp25

// T4 on Windows: the start chain (suspended create, join the Job, confirm,
// resume) and the teardown chain (terminate, query the end), with a
// failure injected at each step. After every failure: the process is never
// resumed, its end is confirmed or reported as not confirmed, every handle
// has an owner, and a successful terminate request is not taken for an
// ended process.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// markerPath is where a "marker" helper's user code leaves its marker.
func markerPath(s *scen) string { return filepath.Join(s.t.TempDir(), "marker") }

// failStart runs the start chain with one injected failure and handles it
// as spec §8.5 S1 says: never resume, end the process, confirm the end.
func failStart(t *testing.T, id string, seams jobSeams, wantStep string) {
	s := begin(t, "T4", id, fault, true, true)
	defer s.done()
	marker := markerPath(s)
	cmd := helper("marker", []string{markerEnv + "=" + marker})
	tr, step, err := startJob(s, cmd, seams)
	defer tr.close()
	s.expect("failed step", wantStep, step)
	s.expectTrue("error reported", err != nil, fmt.Sprint(err))
	inJob := false
	if tr.proc != 0 {
		inJob, _ = isProcessInJob(tr.proc, tr.job)
	}
	var how string
	var ended bool
	switch {
	case step == "resume" && inJob:
		// In the Job: terminate the Job, confirm ActiveProcesses = 0.
		how = "TerminateJobObject"
		_ = tr.kill()
		ended = tr.confirm(tKill)
	default:
		// Not (known to be) in the Job: terminate the process itself.
		how = "TerminateProcess"
		_ = windows.TerminateProcess(tr.proc, 1)
		ev, _ := windows.WaitForSingleObject(tr.proc, uint32(tKill.Milliseconds()))
		ended = ev == windows.WAIT_OBJECT_0
	}
	time.Sleep(50 * time.Millisecond)
	_, statErr := os.Stat(marker)
	reaped := tr.reap(time.Second)
	s.worked(1)
	s.note("ended by %s; in the Job: %v", how, inJob)
	s.expect("end confirmed", true, ended)
	s.expect("user code ran (marker)", false, statErr == nil)
	s.expect("Go's handle released (Wait returned)", true, reaped)
	// Every handle has an owner: the Job and the process handle are this
	// runner's and are closed here; Go closed the main thread's handle at
	// creation and releases its process handle in Wait.
	jobH, procH := tr.job, tr.proc
	tr.close()
	s.expect("Job handle still open", false, handleValid(jobH))
	s.expect("process handle still open", false, handleValid(procH))
}

func errClass(err error) string {
	switch {
	case err == nil:
		return "none"
	case errors.Is(err, fs.ErrNotExist):
		return "not exist"
	}
	return "other"
}

// handleValid reports whether h is still an open handle of this process.
func handleValid(h windows.Handle) bool {
	if h == 0 {
		return false
	}
	var flags uint32
	r, _, _ := procGetHandleInformation.Call(uintptr(h), uintptr(unsafe.Pointer(&flags)))
	return r != 0
}

var procGetHandleInformation = kernel32.NewProc("GetHandleInformation")

func TestJobFailAssign(t *testing.T) {
	failStart(t, "T4.start.fail-assign", jobSeams{assign: errInjected}, "assign")
}

func TestJobFailConfirm(t *testing.T) {
	failStart(t, "T4.start.fail-confirm", jobSeams{confirm: errInjected}, "confirm")
}

func TestJobFailResume(t *testing.T) {
	failStart(t, "T4.start.fail-resume", jobSeams{resume: errInjected}, "resume")
}

func TestJobFailCreate(t *testing.T) {
	// The process is not created: nothing to end; only the Job handle,
	// which this process owns and closes.
	s := begin(t, "T4", "T4.start.fail-create", fault, false, true)
	defer s.done()
	cmd := helper("marker", nil)
	cmd.Path = filepath.Join(s.t.TempDir(), "missing.exe")
	tr, step, err := startJob(s, cmd, jobSeams{})
	s.worked(1)
	s.expect("failed step", "create", step)
	// Only the class of the error, not its text, which names a path.
	s.expect("error class", "not exist", errClass(err))
	s.expect("process created", false, cmd.Process != nil)
	s.expect("launches", 0, s.r.Launches)
	tr.close()
	s.expect("Job handle left", windows.Handle(0), tr.job)
}

func TestJobNormal(t *testing.T) {
	s := begin(t, "T4", "T4.chain.normal", normal, true, true)
	defer s.done()
	marker := markerPath(s)
	cmd := helper("marker", []string{markerEnv + "=" + marker})
	tr, step, err := startJob(s, cmd, jobSeams{})
	defer tr.close()
	if step != "" {
		s.invalidf("start failed at %s: %v", step, err)
	}
	ran := false
	for i := 0; i < 200 && !ran; i++ {
		_, e := os.Stat(marker)
		ran = e == nil
		time.Sleep(10 * time.Millisecond)
	}
	n0, _ := tr.active()
	since := stopwatch()
	kerr := tr.kill()
	ended := tr.confirm(tKill)
	took := since()
	s.worked(int64(n0))
	s.expect("user code ran after the resume", true, ran)
	s.expect("active before the kill", 1, n0)
	s.expect("terminate request", "ok", errOK(kerr))
	s.expect("ActiveProcesses reached 0", true, ended)
	s.note("terminate to 0: %v", took)
	s.expect("reaped", true, tr.reap(time.Second))
}

func errOK(err error) string {
	if err == nil {
		return "ok"
	}
	return err.Error()
}

func TestJobTerminateFails(t *testing.T) {
	// The terminate request fails (injected): nothing is confirmed (F1).
	// Closing the Job handle asks again (KILL_ON_JOB_CLOSE); the scenario
	// then observes with its own handle whether the process ended, which
	// the protocol could not have claimed.
	s := begin(t, "T4", "T4.teardown.terminate-fails", fault, true, true)
	defer s.done()
	cmd := helper("sleep", nil)
	tr, step, err := startJob(s, cmd, jobSeams{terminate: errInjected})
	if step != "" {
		s.invalidf("start failed at %s: %v", step, err)
	}
	kerr := tr.kill()
	conf := tr.confirm(300 * time.Millisecond)
	stillActive, _ := tr.active()
	h := tr.proc
	tr.proc = 0
	_ = windows.CloseHandle(tr.job)
	tr.job = 0
	ev, _ := windows.WaitForSingleObject(h, uint32(tKill.Milliseconds()))
	_ = windows.CloseHandle(h)
	s.worked(1)
	s.expectTrue("terminate request failed", errors.Is(kerr, errInjected), fmt.Sprint(kerr))
	s.expect("end confirmed by the protocol", false, conf)
	s.expect("still active", 1, stillActive)
	s.expect("ended after the Job handle was closed (observed, not claimed)", true, ev == windows.WAIT_OBJECT_0)
	s.expect("reaped", true, tr.reap(time.Second))
}

func TestJobQueryFails(t *testing.T) {
	// The end query fails (injected) after a successful terminate request:
	// the end is not confirmed (F1), although the process may well have
	// ended, which the scenario checks with its own handle.
	s := begin(t, "T4", "T4.teardown.query-fails", fault, true, true)
	defer s.done()
	cmd := helper("sleep", nil)
	tr, step, err := startJob(s, cmd, jobSeams{query: errInjected})
	defer tr.close()
	if step != "" {
		s.invalidf("start failed at %s: %v", step, err)
	}
	kerr := tr.kill()
	conf := tr.confirm(300 * time.Millisecond)
	ev, _ := windows.WaitForSingleObject(tr.proc, uint32(tKill.Milliseconds()))
	s.worked(1)
	s.expect("terminate request", "ok", errOK(kerr))
	s.expect("end confirmed by the protocol", false, conf)
	s.expect("process ended (observed by its handle)", true, ev == windows.WAIT_OBJECT_0)
	s.expect("reaped", true, tr.reap(time.Second))
}

func TestJobTerminateNotEnded(t *testing.T) {
	// A Job of 31 processes, five times: how many are still active right
	// after a successful TerminateJobObject, and how long until 0.
	s := begin(t, "T4", "T4.teardown.request-is-not-end", normal, true, true)
	defer s.done()
	var first []int
	var took []time.Duration
	ended := 0
	for i := 0; i < 5; i++ {
		rep := s.newReport()
		cmd := helper("fanout", []string{reportEnv + "=" + rep}, "30")
		tr, step, err := startJob(s, cmd, jobSeams{})
		if step != "" {
			s.invalidf("start failed at %s: %v", step, err)
		}
		if len(waitReport(rep, "fanout", 1, 20*time.Second)) != 1 {
			tr.close()
			s.invalidf("fanout did not start its children")
		}
		for _, pid := range waitReport(rep, "pid", 30, time.Second) {
			if p, err := ownPID(pid); err == nil {
				s.own(p)
			}
		}
		n0, _ := tr.active()
		since := stopwatch()
		if err := tr.kill(); err != nil {
			tr.close()
			s.invalidf("terminate: %v", err)
		}
		n1, _ := tr.active()
		first = append(first, n1)
		if tr.confirm(tKill) {
			ended++
			took = append(took, since())
		}
		s.worked(int64(n0))
		tr.reap(time.Second)
		tr.close()
	}
	s.expect("Jobs that reached ActiveProcesses = 0", 5, ended)
	s.note("active right after the request: %v; request to 0: %s", first, durations(took))
}
