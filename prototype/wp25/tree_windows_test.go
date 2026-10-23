//go:build windows

package wp25

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func osVersion() string {
	v := windows.RtlGetVersion()
	return fmt.Sprintf("windows %d.%d.%d", v.MajorVersion, v.MinorVersion, v.BuildNumber)
}

func ignoreSIGPIPE() {}

// childProc is a direct child, waited for once; Go holds its handle, so a
// kill through it cannot reach another process.
type childProc struct {
	cmd  *exec.Cmd
	once sync.Once
	done chan struct{}
	err  error
}

var (
	childrenMu sync.Mutex
	children   = map[*exec.Cmd]*childProc{}
)

func ownChild(cmd *exec.Cmd) owned { return childOf(cmd) }

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

func (c *childProc) stop() {
	if !c.reaped() {
		_ = c.cmd.Process.Kill()
		c.reap(5 * time.Second)
	}
}

// pidProc is a reported process the scenario verified as its own, held by
// a handle opened when it was verified; the handle is closed at the end.
type pidProc struct {
	pid      int
	h        windows.Handle
	err      error // why the last check could not confirm an end
	obsErr   error // the first observation that failed; never cleared
	injected bool  // the observe-fail-once fault has acted on it
}

// ownReported verifies a reported PID before the scenario may act on it:
// the process must be in this check's Job, which holds only processes
// the check started. The PID in a report is never enough on its own.
func (t *winTree) ownReported(pid int) owned {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE|
		windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		t.s.invalidf("ownership not established: process %d cannot be opened: %v", pid, err)
	}
	in, err := isProcessInJob(h, t.job)
	if err != nil || !in {
		_ = windows.CloseHandle(h)
		t.s.invalidf("ownership not established: process %d is not in the check's Job (%v)", pid, err)
	}
	p := &pidProc{pid: pid, h: h}
	t.s.own(p)
	return p
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
	ev, err := windows.WaitForSingleObject(p.h, 0)
	if injected("observe-fail-once") && !p.injected {
		p.injected = true // this one wait fails; the next ones do not
		injectHit("pidProc.ended", p.pid)
		err = errors.New("injected: one observation failed")
	}
	if err != nil {
		p.err = err // a wait that fails is not an end
		if p.obsErr == nil {
			p.obsErr = err
		}
		return false
	}
	return ev == windows.WAIT_OBJECT_0
}

func (p *pidProc) stop() { _ = windows.TerminateProcess(p.h, 1) }

func (p *pidProc) pidOf() int        { return p.pid }
func (p *pidProc) observeErr() error { return p.obsErr }

func (p *pidProc) release() {
	if p.h != 0 {
		_ = windows.CloseHandle(p.h)
		p.h = 0
	}
}

// jobSeams injects a failure at one step of the Windows start and
// teardown chain; the real call is then not made.
type jobSeams struct {
	assign, confirm, resume, terminate, query error
}

var errInjected = errors.New("injected failure")

// winTree is one check's Job.
type winTree struct {
	s     *scen
	cmd   *exec.Cmd
	c     *childProc
	job   windows.Handle
	proc  windows.Handle
	pid   int
	seams jobSeams
}

func newJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

// startJob creates the child suspended, assigns it to a new Job, confirms
// the assignment and resumes it. It returns the step that failed; after a
// failure the child is never resumed and the caller handles it.
func startJob(s *scen, cmd *exec.Cmd, seams jobSeams) (*winTree, string, error) {
	job, err := newJob()
	if err != nil {
		s.invalidf("job: %v", err)
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_SUSPENDED
	t := &winTree{s: s, cmd: cmd, job: job, seams: seams}
	if err := cmd.Start(); err != nil {
		return t, "create", err
	}
	s.launched(1)
	t.c = childOf(cmd)
	s.own(t.c)
	t.pid = cmd.Process.Pid
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|
		windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(t.pid))
	if err != nil {
		return t, "open", err
	}
	t.proc = h
	if seams.assign != nil {
		return t, "assign", seams.assign
	}
	if err := windows.AssignProcessToJobObject(job, h); err != nil {
		return t, "assign", err
	}
	in, err := isProcessInJob(h, job)
	if seams.confirm != nil {
		in, err = false, seams.confirm
	}
	if err != nil || !in {
		return t, "confirm", fmt.Errorf("in job %v: %w", in, err)
	}
	if seams.resume != nil {
		return t, "resume", seams.resume
	}
	if n, err := resumeThreads(uint32(t.pid)); err != nil || n != 1 {
		return t, "resume", fmt.Errorf("resumed %d threads: %v", n, err)
	}
	return t, "", nil
}

func startTree(s *scen, cmd *exec.Cmd) *winTree {
	t, step, err := startJob(s, cmd, jobSeams{})
	if step != "" {
		s.invalidf("start failed at %s: %v", step, err)
	}
	s.defer_(func() {
		if t.job != 0 {
			t.endJob()
		}
		_, _ = t.close()
	})
	return t
}

// endJob ends what is left in the Job when the scenario ends and confirms
// it with the real query (not a fault seam): a Job that cannot be
// confirmed empty, or a query that fails, is flagged.
func (t *winTree) endJob() {
	_ = windows.TerminateJobObject(t.job, 1)
	deadline := time.Now().Add(tKill)
	for {
		var a basicAccounting
		err := windows.QueryInformationJobObject(t.job, windows.JobObjectBasicAccountingInformation,
			uintptr(unsafe.Pointer(&a)), uint32(unsafe.Sizeof(a)), nil)
		if err != nil {
			t.s.flag("observation failed while ending the Job: %v", err)
			return
		}
		if a.ActiveProcesses == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.s.flag("cleanup: the Job could not be confirmed empty (%d active)", a.ActiveProcesses)
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func (t *winTree) waitExit(d time.Duration) (bool, string, error) {
	ev, err := windows.WaitForSingleObject(t.proc, uint32(d.Milliseconds()))
	if err != nil {
		t.s.flag("observation failed: waiting for the main process %d: %v", t.pid, err)
		return false, "", err
	}
	if ev != windows.WAIT_OBJECT_0 {
		return false, "", nil
	}
	var code uint32
	if err := windows.GetExitCodeProcess(t.proc, &code); err != nil {
		t.s.flag("observation failed: the exit code of %d: %v", t.pid, err)
		return true, "", err
	}
	return true, "exit " + strconv.Itoa(int(code)), nil
}

func (t *winTree) kill() error {
	if t.seams.terminate != nil {
		return t.seams.terminate
	}
	return windows.TerminateJobObject(t.job, 1)
}

type basicAccounting struct {
	TotalUserTime, TotalKernelTime, ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime int64
	TotalPageFaultCount, TotalProcesses, ActiveProcesses, TotalTerminatedProcesses     uint32
}

// active gives the Job's ActiveProcesses.
func (t *winTree) active() (int, error) {
	if t.seams.query != nil {
		return 0, t.seams.query
	}
	var a basicAccounting
	if err := windows.QueryInformationJobObject(t.job, windows.JobObjectBasicAccountingInformation,
		uintptr(unsafe.Pointer(&a)), uint32(unsafe.Sizeof(a)), nil); err != nil {
		return 0, err
	}
	return int(a.ActiveProcesses), nil
}

// confirm waits within d for ActiveProcesses to be 0; a failed query
// confirms nothing. A query that fails without an injected fault means
// the scenario cannot observe what it predicts: it is invalid.
func (t *winTree) confirm(d time.Duration) bool {
	deadline := time.Now().Add(d)
	for {
		n, err := t.active()
		if err != nil && t.seams.query == nil {
			t.s.invalidf("observation failed: %v", err)
		}
		if err == nil && n == 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Millisecond)
	}
}

func (t *winTree) reap(d time.Duration) bool { return t.c.reap(d) }

// close closes this runner's own handles, the process's and the Job's,
// and returns the results of the closes.
func (t *winTree) close() (procErr, jobErr error) {
	if t.proc != 0 {
		procErr = windows.CloseHandle(t.proc)
		t.proc = 0
	}
	if t.job != 0 {
		jobErr = windows.CloseHandle(t.job)
		t.job = 0
	}
	return procErr, jobErr
}

// resumeThreads resumes the threads of a suspended process; Go closes the
// main thread's handle, so they are found through a snapshot.
func resumeThreads(pid uint32) (int, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snap)
	var e windows.ThreadEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	n := 0
	for err = windows.Thread32First(snap, &e); err == nil; err = windows.Thread32Next(snap, &e) {
		if e.OwnerProcessID != pid {
			continue
		}
		th, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, e.ThreadID)
		if err != nil {
			return n, err
		}
		_, rerr := windows.ResumeThread(th)
		_ = windows.CloseHandle(th)
		if rerr != nil {
			return n, rerr
		}
		n++
	}
	return n, nil
}

var procIsProcessInJob = windows.NewLazySystemDLL("kernel32.dll").NewProc("IsProcessInJob")

func isProcessInJob(h, job windows.Handle) (bool, error) {
	var b int32
	r, _, err := procIsProcessInJob.Call(uintptr(h), uintptr(job), uintptr(unsafe.Pointer(&b)))
	if r == 0 {
		return false, err
	}
	return b != 0, nil
}

func runHelperPlatform(string, []string) (int, bool) { return 0, false }

// ownChildren and newChildren: the leftover-children check is Linux's;
// on Windows every check's processes are in its Job.
func ownChildren() []int       { return nil }
func newChildren([]int) string { return "" }
