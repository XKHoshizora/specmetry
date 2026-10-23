//go:build windows

package wp25

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func creationTime(h windows.Handle) (string, error) {
	var c, e, k, u windows.Filetime
	if err := windows.GetProcessTimes(h, &c, &e, &k, &u); err != nil {
		return "", err
	}
	return strconv.FormatUint(uint64(c.HighDateTime)<<32|uint64(c.LowDateTime), 10), nil
}

// selfIdentity is this process's creation time.
func selfIdentity() (string, error) {
	h, err := windows.GetCurrentProcess()
	if err != nil {
		return "", err
	}
	s, err := creationTime(h)
	if inHelper() && injected("witness-identity-unknown") {
		injectHit("selfIdentity", int(windows.GetCurrentProcessId()))
		err = errors.New("injected: the creation time could not be read")
	}
	return s, err
}

// childIdentity is the creation time of a child this process has just
// started; Go holds its handle, so its PID still names it.
func childIdentity(cmd *exec.Cmd) (string, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(cmd.Process.Pid))
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)
	return creationTime(h)
}

// raiseWitness tells the outer check, by its named event, that a record
// could not be made.
func raiseWitness() {
	name := os.Getenv(raiseEnv)
	if name == "" {
		return
	}
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return
	}
	h, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, p)
	if err != nil {
		return
	}
	_ = windows.SetEvent(h)
	_ = windows.CloseHandle(h)
}

var raiseSeq atomic.Int64

// raiseWatch is the outer check's event for one gate check.
type raiseWatch struct {
	name string
	ev   windows.Handle
}

func newRaiseWatch() (raiseWatch, error) {
	name := fmt.Sprintf(`Local\wp25-witness-%d-%d`, os.Getpid(), raiseSeq.Add(1))
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return raiseWatch{}, err
	}
	ev, err := windows.CreateEvent(nil, 1, 0, p)
	if err != nil {
		return raiseWatch{}, err
	}
	return raiseWatch{name: name, ev: ev}, nil
}

func (w raiseWatch) env() string { return raiseEnv + "=" + w.name }

func (w raiseWatch) raised() bool {
	ev, err := windows.WaitForSingleObject(w.ev, 200)
	return err != nil || ev == windows.WAIT_OBJECT_0
}

func (w raiseWatch) close() { _ = windows.CloseHandle(w.ev) }

// innerProc is the inner test process as the outer check started it: in a
// Job of the outer check's own, from which no process of the run can break
// away. The Job's accounting counts every process the run created, and
// lists the ones still active, independently of the records.
type innerProc struct {
	cmd     *exec.Cmd
	pid     int
	start   string
	job     windows.Handle
	fault   string // an outer-side fault of the tested path (TestGateCombos)
	faulted bool
}

func launchInner(cmd *exec.Cmd, _ string) (*innerProc, error) {
	job, err := newJob()
	if err != nil {
		return nil, err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_SUSPENDED}
	if err := cmd.Start(); err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	p := &innerProc{cmd: cmd, pid: cmd.Process.Pid, job: job}
	fail := func(err error) (*innerProc, error) {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = windows.CloseHandle(job)
		return nil, err
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(p.pid))
	if err != nil {
		return fail(err)
	}
	defer windows.CloseHandle(h)
	if p.start, err = creationTime(h); err != nil {
		return fail(err)
	}
	if err := windows.AssignProcessToJobObject(job, h); err != nil {
		return fail(fmt.Errorf("assign: %w", err))
	}
	if in, err := isProcessInJob(h, job); err != nil || !in {
		return fail(fmt.Errorf("in the Job %v: %v", in, err))
	}
	if n, err := resumeThreads(uint32(p.pid)); err != nil || n != 1 {
		return fail(fmt.Errorf("resumed %d threads: %v", n, err))
	}
	return p, nil
}

type pidList struct {
	assigned, listed uint32
	ids              [4096]uintptr
}

// census lists the processes still active in the Job, and gives the number
// it saw created, the inner run included.
func (p *innerProc) census() ([]survivor, int, error) {
	var a basicAccounting
	if err := windows.QueryInformationJobObject(p.job, windows.JobObjectBasicAccountingInformation,
		uintptr(unsafe.Pointer(&a)), uint32(unsafe.Sizeof(a)), nil); err != nil {
		return nil, -1, fmt.Errorf("%w: %v", errCensus, err)
	}
	var l pidList
	if err := windows.QueryInformationJobObject(p.job, windows.JobObjectBasicProcessIdList,
		uintptr(unsafe.Pointer(&l)), uint32(unsafe.Sizeof(l)), nil); err != nil {
		return nil, -1, fmt.Errorf("%w: %v", errCensus, err)
	}
	if l.listed != l.assigned {
		return nil, -1, fmt.Errorf("%w: %d processes active, %d listed", errCensus, l.assigned, l.listed)
	}
	var out []survivor
	for _, id := range l.ids[:l.listed] {
		v, ok, err := activeMember(int(id))
		if err != nil {
			return nil, -1, fmt.Errorf("%w: process %d: %v", errCensus, id, err)
		}
		if ok {
			out = append(out, v)
		}
	}
	return out, int(a.TotalProcesses), nil
}

// activeMember reads a listed member's identity; one that ended since the
// list was taken is left out.
func activeMember(pid int) (survivor, bool, error) {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return survivor{}, false, nil
	}
	if err != nil {
		return survivor{}, false, err
	}
	defer windows.CloseHandle(h)
	st, err := creationTime(h)
	if err != nil {
		return survivor{}, false, err
	}
	ev, err := windows.WaitForSingleObject(h, 0)
	if err != nil {
		return survivor{}, false, err
	}
	return survivor{pid, st}, ev != windows.WAIT_OBJECT_0, nil
}

// remediate terminates the Job: its members are exactly this run's
// processes. It returns the ones still active after a bounded wait.
func (p *innerProc) remediate(ss []survivor) ([]survivor, error) {
	if len(ss) == 0 {
		return nil, nil
	}
	_ = windows.TerminateJobObject(p.job, 1)
	deadline := time.Now().Add(10 * time.Second)
	for {
		left, _, err := p.census()
		if err != nil {
			return nil, err
		}
		if len(left) == 0 || time.Now().After(deadline) {
			return left, nil
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (p *innerProc) close() { _ = windows.CloseHandle(p.job) }

// insure is the experiment's own last cleanup, after the tested path was
// judged and apart from it: the Job's members are this round's processes,
// so whatever is still active in it is terminated, and the Job is then
// confirmed empty. Its result never makes a check pass.
func (p *innerProc) insure() string {
	ss, _, err := p.census()
	if err != nil {
		return "FAILED: " + err.Error()
	}
	if len(ss) == 0 {
		return "nothing left"
	}
	var pids []int
	for _, v := range ss {
		pids = append(pids, v.pid)
	}
	_ = windows.TerminateJobObject(p.job, 1)
	deadline := time.Now().Add(10 * time.Second)
	for {
		left, _, err := p.census()
		switch {
		case err != nil:
			return "FAILED: " + err.Error()
		case len(left) == 0:
			return fmt.Sprintf("ended %d: %v", len(pids), pids)
		case time.Now().After(deadline):
			return fmt.Sprintf("FAILED: %d still active", len(left))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// openRecorded opens the process a record names and checks that it is the
// one recorded (same creation time). The identity has the platform's form
// (parseWitness), so a different creation time is reliably another process.
func openRecorded(w witnessRec, access uint32) (windows.Handle, bool, error) {
	h, err := windows.OpenProcess(access|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(w.pid))
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return 0, false, nil // no such process
		}
		return 0, false, err
	}
	s, err := creationTime(h)
	if err != nil {
		_ = windows.CloseHandle(h)
		return 0, false, err
	}
	if s != w.start {
		_ = windows.CloseHandle(h)
		return 0, false, nil // the PID names another process now
	}
	return h, true, nil
}

// recordAlive reports whether the process a record names still runs. A
// state that cannot be read is an error, never an end.
func recordAlive(w witnessRec) (bool, error) {
	h, same, err := openRecorded(w, windows.SYNCHRONIZE)
	if err != nil || !same {
		return false, err
	}
	defer windows.CloseHandle(h)
	ev, err := windows.WaitForSingleObject(h, 0)
	if err != nil {
		return false, err
	}
	return ev != windows.WAIT_OBJECT_0, nil
}

var errCensus = errors.New("census")
