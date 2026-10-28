//go:build windows

package wp25

// P-4 (spec v7 §19.7 and §19.12, G7, Q-1 and Q-2; plan fixed in
// docs/wp2.5-prototype.md §14.5): the four results of ResumeThread, the
// failures before the resume, and an end not confirmed before and after
// it. The inner call is the model's start and teardown chain with its
// knowledge states, record decisions and ownership branches; the outer,
// with handles of its own, confirms what was hit, keeps the inner state
// and only then cleans up. The outer does not nest a Job of its own
// (nesting changes the tested Job); the helpers create no descendants, so
// its process handle confirms the end of all of this run's processes.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procSuspendThread        = kernel32.NewProc("SuspendThread")
	procQueryThreadCycleTime = kernel32.NewProc("QueryThreadCycleTime")
)

// p4Seams are the injections and constructions of one scenario.
type p4Seams struct {
	// assign, snapshot, openThread fail a step before the resume; the
	// real call is not made.
	assign, snapshot, openThread bool
	// resumeAccess replaces the access the Runner opens the thread with
	// for its resume (a seam that changes the premise; the call is real).
	resumeAccess uint32
	// swallowTerminate swallows TerminateProcess (reported sent, never
	// sent); query fails the Job's query.
	swallowTerminate, query bool
	// created runs right after the process is created, before the
	// Runner's first step: the outer opens its own handles.
	created func(pid int)
	// before runs after the Runner has opened the main thread and just
	// before its ResumeThread: the fixture's own construction on that
	// thread (the snapshot still shows the one suspended thread).
	before func(tid uint32) string
}

// p4Inner is what the inner call reports.
type p4Inner struct {
	Knowledge string `json:"knowledge"` // "definitely not run", "possibly run", "started by protocol"
	Step      string `json:"failed_step,omitempty"`
	Resume    string `json:"resume,omitempty"` // the raw result of the Runner's ResumeThread
	Case      string `json:"case"`             // A4, A5, A6, A10, A12
	TStart    bool   `json:"t_start"`
	TAnom     bool   `json:"t_anom"`
	Ended     string `json:"ended"`
	Exit      string `json:"exit,omitempty"` // the status the Runner reliably read
	// ExitHow says why Exit is what it is: "obtained", "not obtained: the
	// wait timed out", or "read failed" (an observation failure, flagged).
	ExitHow     string   `json:"exit_how,omitempty"`
	Record      *record  `json:"record"`
	JobClose    string   `json:"job_close"`
	ProcClose   string   `json:"proc_close,omitempty"`
	ThreadClose string   `json:"thread_close,omitempty"`
	Handed      bool     `json:"handed_over"`
	Waited      bool     `json:"waited"`
	Hits        []string `json:"seam_hits,omitempty"`
	handle      windows.Handle
}

func runnerAccess() uint32 {
	return windows.PROCESS_SET_QUOTA | windows.PROCESS_TERMINATE | windows.PROCESS_QUERY_LIMITED_INFORMATION | windows.SYNCHRONIZE
}

// p4Run is the model's chain: create suspended, join the Job, confirm,
// snapshot, open the thread, resume, then the teardown and the record.
func p4Run(s *scen, h *host, cmd *exec.Cmd, seams p4Seams) (*p4Inner, *exec.Cmd) {
	in := &p4Inner{}
	job, err := newJob()
	if err != nil {
		s.invalidf("job: %v", err)
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_SUSPENDED
	if err := start(cmd); err != nil {
		_ = windows.CloseHandle(job)
		s.invalidf("create: %v", err)
	}
	s.launched(1)
	s.own(childOf(cmd))
	pid := cmd.Process.Pid
	if seams.created != nil {
		seams.created(pid)
	}
	proc, err := windows.OpenProcess(runnerAccess(), false, uint32(pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		s.invalidf("the Runner's process handle: %v", err)
	}
	clock := time.Now() // the run's monotonic clock
	var tAnom, tS3 time.Duration
	inJob := false
	failed := ""
	// Before the resume.
	switch {
	case seams.assign:
		in.Hits = append(in.Hits, "assign: injected failure, not called")
		failed = "assign"
	default:
		if err := windows.AssignProcessToJobObject(job, proc); err != nil {
			failed = "assign: " + err.Error()
			s.flag("unplanned failure of the start chain: %s", failed)
			break
		}
		ok, err := isProcessInJob(proc, job)
		if err != nil || !ok {
			failed = fmt.Sprintf("confirm: %v %v", ok, err)
			s.flag("unplanned failure of the start chain: %s", failed)
			break
		}
		inJob = true
	}
	var tid uint32
	threads := 0
	if failed == "" {
		if seams.snapshot {
			in.Hits = append(in.Hits, "snapshot: injected failure, not called")
			failed = "snapshot"
		} else {
			tid, threads, err = p4Threads(uint32(pid))
			if err != nil || threads == 0 {
				failed = fmt.Sprintf("snapshot: %d threads, %v", threads, err)
				s.flag("unplanned failure of the start chain: %s", failed)
			}
			// More than one thread (5') is a resume anomaly without a
			// ResumeThread; no scenario here constructs it.
		}
	}
	var th windows.Handle
	if failed == "" {
		switch {
		case seams.openThread:
			in.Hits = append(in.Hits, "open thread: injected failure, not called")
			failed = "open thread"
		default:
			access := uint32(windows.THREAD_SUSPEND_RESUME)
			if seams.resumeAccess != 0 {
				access = seams.resumeAccess
				in.Hits = append(in.Hits, fmt.Sprintf("open thread: the access replaced by %#x (no THREAD_SUSPEND_RESUME)", access))
			}
			th, err = windows.OpenThread(access, false, tid)
			if err != nil {
				failed = "open thread: " + err.Error()
				s.flag("unplanned failure of the start chain: %s", failed)
			}
		}
	}
	if failed != "" {
		// Never resumed: definitely not run (the engineering premise of
		// §19.7). End it and confirm within T_kill, by the Job when it is
		// known to be in it, else by its own handle.
		in.Knowledge, in.Step = "definitely not run", failed
		confirmed := false
		if inJob {
			_ = windows.TerminateJobObject(job, 1)
			confirmed = p4Confirm(s, job, seams.query, time.Now().Add(tKill))
			in.Ended = "ActiveProcesses=0"
		} else {
			if seams.swallowTerminate {
				in.Hits = append(in.Hits, "TerminateProcess: swallowed, not sent")
			} else {
				_ = windows.TerminateProcess(proc, 1)
			}
			ev, werr := windows.WaitForSingleObject(proc, uint32(tKill.Milliseconds()))
			if werr != nil {
				s.flag("observation failed: waiting for the process handle: %v", werr)
			}
			confirmed = werr == nil && ev == windows.WAIT_OBJECT_0
			in.Ended = "process handle"
		}
		if confirmed {
			in.Case = "A4"
			p4Close(in, th, proc, job)
			in.Waited = childOf(cmd).reap(time.Second)
			return in, cmd
		}
		in.Case, in.Ended = "A5", "not confirmed"
		p4HandOver(in, h, cmd, th, proc, job)
		return in, cmd
	}
	var ret uint32
	var rerr error
	if threads > 1 {
		in.Resume = fmt.Sprintf("not called: %d threads", threads)
	} else {
		if seams.before != nil {
			seams.before(tid)
		}
		ret, rerr = windows.ResumeThread(th)
	}
	at := time.Since(clock)
	switch {
	case threads > 1:
	case ret == 0xFFFFFFFF:
		in.Resume = "-1 (" + errnoText(rerr) + ")"
	default:
		in.Resume = strconv.FormatUint(uint64(ret), 10)
	}
	if ret == 1 && threads == 1 {
		// Started by protocol: t_start; S2 waits for E-main.
		in.Knowledge, in.TStart = "started by protocol", true
		ev, werr := windows.WaitForSingleObject(proc, 10000)
		tMain := time.Duration(-1)
		in.ExitHow = p4ReadExit(s, in, proc, ev, werr)
		if in.ExitHow == "obtained" {
			tMain = time.Since(clock)
		}
		_ = windows.TerminateJobObject(job, 1) // D11: what is left in the Job
		if !p4Confirm(s, job, seams.query, time.Now().Add(tKill)) {
			in.Case, in.Ended = "A6", "not confirmed"
			p4HandOver(in, h, cmd, th, proc, job)
			return in, cmd
		}
		in.Case, in.Ended = "A10", "ActiveProcesses=0"
		if in.Exit == "exit 0" {
			zero := 0
			in.Record = &record{Attempted: true, Result: "pass", ExitCode: &zero, Qualifying: true, DurationMs: (tMain - at).Milliseconds(), StartedAt: "t_start", Batch: "continue", Exit: 0}
		}
		p4Close(in, th, proc, job)
		in.Waited = childOf(cmd).reap(time.Second)
		return in, cmd
	}
	// A resume anomaly: possibly run; t_anom; straight to S3.
	in.Knowledge, in.TAnom = "possibly run", true
	tAnom = at
	tS3 = time.Since(clock)
	_ = windows.TerminateJobObject(job, 1)
	deadline := time.Now().Add(tKill)
	confirmed := p4Confirm(s, job, seams.query, deadline)
	// The exit status, if it can be reliably read within what is left.
	left := time.Until(deadline)
	if left < 0 {
		left = 0
	}
	ev, werr := windows.WaitForSingleObject(proc, uint32(left.Milliseconds()))
	if injectOnce("p4-anomaly-wait-fail-once", "p4Run.anomaly-wait", pid) {
		werr = errors.New("injected: the wait failed once") // the real call was made
	}
	in.ExitHow = p4ReadExit(s, in, proc, ev, werr)
	run := anomalyRun{treeConfirmed: confirmed, tAnom: tAnom, tS3: tS3}
	if code, ok := p4Code(in.Exit); ok {
		run.exitObtained, run.exitCode = true, code
	}
	rec := decideAnomaly(run)
	in.Record = &rec
	if confirmed {
		in.Case, in.Ended = "A12", "ActiveProcesses=0"
		p4Close(in, th, proc, job)
		in.Waited = childOf(cmd).reap(time.Second)
		return in, cmd
	}
	in.Case, in.Ended = "A6", "not confirmed"
	p4HandOver(in, h, cmd, th, proc, job)
	return in, cmd
}

func errnoText(err error) string {
	var en syscall.Errno
	if errors.As(err, &en) {
		switch en {
		case windows.ERROR_ACCESS_DENIED:
			return "ERROR_ACCESS_DENIED"
		}
		return "errno " + strconv.Itoa(int(en))
	}
	return fmt.Sprint(err)
}

// p4Threads enumerates the process's threads: the first one's ID and
// how many there are.
func p4Threads(pid uint32) (uint32, int, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return 0, 0, err
	}
	defer windows.CloseHandle(snap)
	var e windows.ThreadEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	var first uint32
	n := 0
	for err = windows.Thread32First(snap, &e); err == nil; err = windows.Thread32Next(snap, &e) {
		if e.OwnerProcessID != pid {
			continue
		}
		if n == 0 {
			first = e.ThreadID
		}
		n++
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return 0, 0, err
	}
	return first, n, nil
}

// p4Confirm waits until ActiveProcesses = 0, at most until deadline; a
// failed query (injected or real) confirms nothing.
func p4Confirm(s *scen, job windows.Handle, fail bool, deadline time.Time) bool {
	for {
		if fail {
			return false
		}
		var a basicAccounting
		if err := windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation,
			uintptr(unsafe.Pointer(&a)), uint32(unsafe.Sizeof(a)), nil); err != nil {
			s.flag("observation failed: the Job's active processes: %v", err)
			return false
		}
		if a.ActiveProcesses == 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Millisecond)
	}
}

// p4Exit reads a signalled process's exit code; a read that fails is an
// error, never an empty value that could pass for "not obtained".
func p4Exit(h windows.Handle) (string, error) {
	var code uint32
	err := windows.GetExitCodeProcess(h, &code)
	if injectOnce("p4-exitcode-fail-once", "p4Exit", 0) {
		err = errors.New("injected: GetExitCodeProcess failed once") // the real call was made
	}
	if err != nil {
		return "", err
	}
	return "exit " + strconv.Itoa(int(code)), nil
}

// p4ReadExit is the Runner's E-main read after a bounded wait: a timeout
// is a legitimate "not obtained"; a failed wait or read is an observation
// failure of the run, kept whatever follows (Review 095 B1).
func p4ReadExit(s *scen, in *p4Inner, proc windows.Handle, ev uint32, werr error) string {
	switch {
	case werr != nil:
		s.flag("observation failed: waiting for the main process: %v", werr)
		return "read failed"
	case ev != windows.WAIT_OBJECT_0:
		return "not obtained: the wait timed out"
	}
	exit, err := p4Exit(proc)
	if err != nil {
		s.flag("observation failed: the exit code of the main process: %v", err)
		return "read failed"
	}
	in.Exit = exit
	return "obtained"
}

func p4Code(exit string) (int, bool) {
	n, err := strconv.Atoi(exit[min(len(exit), len("exit ")):])
	return n, err == nil && exit != ""
}

// p4Close is the branch "end confirmed": every handle closed.
func p4Close(in *p4Inner, th, proc, job windows.Handle) {
	if th != 0 {
		in.ThreadClose = errOK(windows.CloseHandle(th))
	}
	in.ProcClose = errOK(windows.CloseHandle(proc))
	in.JobClose = errOK(windows.CloseHandle(job))
}

// p4HandOver is the branch "end not confirmed" (A5, A6): the Job handle
// closed (KILL_ON_JOB_CLOSE asks again), the thread handle closed, the
// Runner's process handle not closed but handed to the inventory with the
// Cmd, no Wait, the host disabled (Q-1).
func p4HandOver(in *p4Inner, h *host, cmd *exec.Cmd, th, proc, job windows.Handle) {
	if th != 0 {
		in.ThreadClose = errOK(windows.CloseHandle(th))
	}
	in.JobClose = errOK(windows.CloseHandle(job))
	in.handle, in.Handed = proc, true
	h.handOver(invItem{kind: "unconfirmed-process", cmd: cmd, handle: uintptr(proc), pid: cmd.Process.Pid})
}

// p4Scenario is one row of §14.5.
type p4Scenario struct {
	id, source, knowledge, caseRow string
	marker                         bool // the helper writes a marker and exits 0; else it writes one and runs on
	seams                          func(fix *p4Fixture) p4Seams
	record                         string // "none", "pass", "anomaly"
	disabled                       bool
	resume                         string // the Runner's ResumeThread result, "" when not called
}

// p4Fixture is the outer's own handles and fixture events.
type p4Fixture struct {
	events []string
	thread windows.Handle // the outer's read-only handle to the main thread
}

var p4Scenarios = []p4Scenario{
	{id: "healthy", source: "real API", knowledge: "started by protocol", caseRow: "A10", marker: true, record: "pass", resume: "1",
		seams: func(*p4Fixture) p4Seams { return p4Seams{} }},
	{id: "resume.0", source: "construction + real API", knowledge: "possibly run", caseRow: "A12", record: "anomaly", resume: "0",
		seams: func(f *p4Fixture) p4Seams {
			return p4Seams{before: func(tid uint32) string {
				return f.construct(tid, "ResumeThread", func(h windows.Handle) (uint32, error) { return windows.ResumeThread(h) })
			}}
		}},
	{id: "resume.gt1", source: "construction + real API", knowledge: "possibly run", caseRow: "A12", record: "anomaly", resume: "2",
		seams: func(f *p4Fixture) p4Seams {
			return p4Seams{before: func(tid uint32) string { return f.construct(tid, "SuspendThread", suspendThread) }}
		}},
	{id: "resume.-1", source: "seam on the premise + real API failure", knowledge: "possibly run", caseRow: "A12", record: "anomaly", resume: "-1 (ERROR_ACCESS_DENIED)",
		seams: func(*p4Fixture) p4Seams { return p4Seams{resumeAccess: windows.THREAD_QUERY_LIMITED_INFORMATION} }},
	{id: "pre.assign", source: "seam injection", knowledge: "definitely not run", caseRow: "A4", record: "none",
		seams: func(*p4Fixture) p4Seams { return p4Seams{assign: true} }},
	{id: "pre.snapshot", source: "seam injection", knowledge: "definitely not run", caseRow: "A4", record: "none",
		seams: func(*p4Fixture) p4Seams { return p4Seams{snapshot: true} }},
	{id: "pre.openthread", source: "seam injection", knowledge: "definitely not run", caseRow: "A4", record: "none",
		seams: func(*p4Fixture) p4Seams { return p4Seams{openThread: true} }},
	{id: "pre.unconfirmed", source: "seam injection", knowledge: "definitely not run", caseRow: "A5", record: "none", disabled: true,
		seams: func(*p4Fixture) p4Seams { return p4Seams{assign: true, swallowTerminate: true} }},
	{id: "anomaly.unconfirmed", source: "construction + real API + seam injection", knowledge: "possibly run", caseRow: "A6", record: "anomaly", disabled: true, resume: "2",
		seams: func(f *p4Fixture) p4Seams {
			return p4Seams{query: true, before: func(tid uint32) string { return f.construct(tid, "SuspendThread", suspendThread) }}
		}},
}

func suspendThread(h windows.Handle) (uint32, error) {
	r, _, err := procSuspendThread.Call(uintptr(h))
	if uint32(r) == 0xFFFFFFFF {
		return uint32(r), err
	}
	return uint32(r), nil
}

// construct is the fixture's own call on the main thread before the
// Runner's resume, with a handle of its own; its result is the fixture
// event.
func (f *p4Fixture) construct(tid uint32, name string, call func(windows.Handle) (uint32, error)) string {
	th, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, tid)
	if err != nil {
		ev := fmt.Sprintf("fixture: opening thread %d for %s failed: %v", tid, name, err)
		f.events = append(f.events, ev)
		return ev
	}
	r, cerr := call(th)
	_ = windows.CloseHandle(th)
	ev := fmt.Sprintf("fixture: its own %s on thread %d returned %d (%v)", name, tid, int32(r), cerr)
	f.events = append(f.events, ev)
	return ev
}

// cycles is an auxiliary read only: a thread's cycle time, never taken
// as evidence that it ran or did not.
func cycles(th windows.Handle) string {
	if th == 0 {
		return "unsupported: no handle"
	}
	var c uint64
	r, _, err := procQueryThreadCycleTime.Call(uintptr(th), uintptr(unsafe.Pointer(&c)))
	if r == 0 {
		return "unsupported: " + err.Error()
	}
	return strconv.FormatUint(c, 10)
}

var p4Budget = &budget{name: "P-4", total: 4 * time.Minute, per: 30 * time.Second}

func TestP4(t *testing.T) {
	for _, sc := range p4Scenarios {
		for n := 1; n <= 2; n++ {
			t.Run(fmt.Sprintf("%s.%d", sc.id, n), func(t *testing.T) { runP4(t, sc, n) })
		}
	}
}

func runP4(t *testing.T, sc p4Scenario, n int) {
	id := fmt.Sprintf("P4.%s.%d", sc.id, n)
	k := fault
	if sc.id == "healthy" {
		k = normal
	}
	s := begin(t, "G7", id, k, true, true)
	defer s.done()
	if ok, left := p4Budget.allow(); !ok {
		s.invalidf("not executed: the total deadline of P-4 has %v left, less than one run's %v", left.Round(time.Second), p4Budget.per)
	}
	marker := filepath.Join(s.t.TempDir(), "marker")
	cmd := helper("marker", []string{markerEnv + "=" + marker})
	if sc.marker {
		cmd = helper("marker-exit", []string{markerEnv + "=" + marker})
	}
	fix := &p4Fixture{}
	seams := sc.seams(fix)
	var oh windows.Handle
	seams.created = func(pid int) {
		// The outer's own handles, read-only and for waiting, opened
		// before the Runner's first step.
		h, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_TERMINATE, false, uint32(pid))
		if err != nil {
			s.invalidf("the outer's process handle: %v", err)
		}
		oh = h
		s.defer_(func() { _ = windows.CloseHandle(h) })
		if tid, cnt, err := p4Threads(uint32(pid)); err == nil && cnt == 1 {
			if th, err := windows.OpenThread(windows.THREAD_QUERY_LIMITED_INFORMATION, false, tid); err == nil {
				fix.thread = th
				s.defer_(func() { _ = windows.CloseHandle(th) })
			}
		}
	}
	h := &host{}
	if err := h.admit(); err != nil {
		s.invalidf("a fresh host refused: %v", err)
	}
	base := maxGoroutine()
	began := time.Now()
	in, _ := p4Run(s, h, cmd, seams)
	h.leave()
	if d := time.Since(began); d > 20*time.Second {
		s.flag("the inner call took %v, over its 20 s", d)
	}

	// The outer, before any cleanup.
	gs := goroutines()
	waiting, fresh := waitingGoroutines(gs), freshProto(gs, base, goid())
	disabled, reason, items := h.state()
	ev, werr := windows.WaitForSingleObject(oh, 0)
	if injectOnce("p4-outer-wait-fail-once", "runP4.outer-wait", cmd.Process.Pid) {
		werr = errors.New("injected: the outer's wait failed once") // the real call was made
	}
	if werr != nil {
		s.flag("observation failed: the outer's wait on %d: %v", cmd.Process.Pid, werr)
	}
	alive := werr == nil && ev == uint32(windows.WAIT_TIMEOUT)
	ended := werr == nil && ev == windows.WAIT_OBJECT_0
	outerExit := ""
	if ended {
		x, xerr := p4Exit(oh)
		if xerr != nil {
			s.flag("observation failed: the outer's read of the exit code of %d: %v", cmd.Process.Pid, xerr)
		}
		outerExit = x
	}
	itemOK, itemDetail := false, fmt.Sprintf("%d items", len(items))
	if len(items) == 1 {
		hv := windows.Handle(items[0].handle)
		iev, ierr := windows.WaitForSingleObject(hv, 0)
		if injectOnce("p4-item-wait-fail-once", "runP4.item-wait", cmd.Process.Pid) {
			ierr = errors.New("injected: the wait on the handed handle failed once") // the real call was made
		}
		if ierr != nil {
			s.flag("observation failed: the wait on the handed handle: %v", ierr)
		}
		valid := ierr == nil && (iev == windows.WAIT_OBJECT_0 || iev == uint32(windows.WAIT_TIMEOUT))
		itemOK = items[0].cmd == cmd && hv == in.handle && valid
		itemDetail = fmt.Sprintf("the run's Cmd %v, the Runner's handle %v, a zero wait on it %s", items[0].cmd == cmd, hv == in.handle, waitResult(iev, ierr))
	}
	_, markErr := os.Stat(marker)
	aux := cycles(fix.thread)
	acc, aerr := h.request(func(a *access) { a.db, a.git, a.procs = 1, 1, 1 })
	pline(id, map[string]any{
		"source": sc.source, "inner": in, "fixture": fix.events, "outer_alive": alive, "outer_exit": outerExit,
		"inventory": itemDetail, "disabled": disabled, "reason": reason, "waiting": waiting, "fresh": fresh,
		"marker": markErr == nil, "aux_cycle_time": aux, "admission": admission(acc, aerr),
	})
	for _, e := range fix.events {
		s.note("%s", e)
	}
	for _, e := range in.Hits {
		s.note("seam: %s", e)
	}
	s.note("source: %s; aux cycle time %s (not evidence); marker %v (not evidence)", sc.source, aux, markErr == nil)
	// The target was hit: the fixture's construction answered as planned,
	// or the seam acted, or the real API returned what the row is about.
	switch sc.id {
	case "resume.0":
		s.needLoad(len(fix.events) == 1 && strings.Contains(fix.events[0], "ResumeThread on thread") && strings.Contains(fix.events[0], "returned 1 "), "%v", fix.events)
	case "resume.gt1", "anomaly.unconfirmed":
		s.needLoad(len(fix.events) == 1 && strings.Contains(fix.events[0], "SuspendThread on thread") && strings.Contains(fix.events[0], "returned 1 "), "%v", fix.events)
	case "resume.-1", "pre.assign", "pre.snapshot", "pre.openthread", "pre.unconfirmed":
		s.needLoad(len(in.Hits) >= 1, "seam hits %v", in.Hits)
	}
	if sc.id == "pre.unconfirmed" {
		s.needLoad(alive, "the suspended process still exists before the cleanup (outer handle not signalled)")
	}

	s.needLoad(in.Resume == sc.resume, "the Runner's ResumeThread result %q (the row is about %q)", in.Resume, sc.resume)
	s.expect("knowledge", sc.knowledge, in.Knowledge)
	s.expect("case", sc.caseRow, in.Case)
	s.expect("t_start", sc.knowledge == "started by protocol", in.TStart)
	s.expect("t_anom", sc.knowledge == "possibly run", in.TAnom)
	s.expect("Job handle closed", "ok", in.JobClose)
	switch sc.caseRow {
	case "A5", "A6":
		s.expectTrue("the Runner's process handle handed over and valid", in.Handed && itemOK, itemDetail)
		s.expect("Go's process object waited", false, in.Waited)
		s.expectTrue("no goroutine waits for a process", len(waiting) == 0, fmt.Sprint(waiting))
	default:
		s.expect("the Runner's process handle closed", "ok", in.ProcClose)
		s.expect("Go's process object waited", true, in.Waited)
		s.expect("inventory items", 0, len(items))
	}
	if in.ThreadClose != "" {
		s.expect("thread handle closed", "ok", in.ThreadClose)
	}
	s.expectTrue("no new prototype goroutine", len(fresh) == 0, fmt.Sprint(fresh))
	s.expect("host disabled", sc.disabled, disabled)
	if sc.disabled {
		s.expect("disabled reason", "unconfirmed_end", reason)
		s.expect("next check", "refused (0 db, 0 git, 0 processes)", admission(acc, aerr))
	} else {
		s.expect("next check", "admitted (1 db, 1 git, 1 processes)", admission(acc, aerr))
	}
	switch sc.record {
	case "none":
		s.expectTrue("no record", in.Record == nil, fmt.Sprint(in.Record))
	case "pass":
		s.expectTrue("record pass, exit 0", in.Record != nil && in.Record.Result == "pass" && in.Record.ExitCode != nil && *in.Record.ExitCode == 0, fmt.Sprintf("%+v", in.Record))
	case "anomaly":
		r := in.Record
		ok := r != nil && r.Attempted && r.Result == "error" && r.FailureKind == "context_changed" && !r.Qualifying &&
			r.StartedAt == "t_anom" && r.HeadEnd == "NULL" && r.TreeEnd == "unknown" && r.Batch == "stop" && r.Exit == 6 && r.DurationMs >= 0
		s.expectTrue("record: error/context_changed, unqualified, t_anom, t_S3 - t_anom, end context not read, stop, 6", ok, fmt.Sprintf("%+v", r))
		// The exit code is kept whenever the Runner reliably read it, and
		// is then the outer's own reading.
		want := "NULL"
		switch in.ExitHow {
		case "obtained":
			want = in.Exit
			s.expect("the Runner's exit reading agrees with the outer's", in.Exit, outerExit)
		case "not obtained: the wait timed out":
			s.note("the Runner did not obtain the exit status: its wait timed out; the outer read %q", outerExit)
		}
		got := "NULL"
		if r != nil && r.ExitCode != nil {
			got = "exit " + strconv.Itoa(*r.ExitCode)
		}
		s.expect("record exit code", want, got)
	}

	// The cleanup, after the evidence is kept.
	if !ended {
		_ = windows.TerminateProcess(oh, 1)
	}
	cev, cerr := windows.WaitForSingleObject(oh, 10000)
	if cerr != nil || cev != windows.WAIT_OBJECT_0 {
		s.flag("cleanup: the outer could not confirm the end of %d (%v)", cmd.Process.Pid, cerr)
	}
	if !childOf(cmd).reap(5 * time.Second) {
		s.flag("cleanup: %d was not reaped", cmd.Process.Pid)
	}
	for _, it := range h.release() {
		if err := windows.CloseHandle(windows.Handle(it.handle)); err != nil {
			s.flag("the inventory's handle could not be released: %v", err)
		}
	}
	if sc.disabled {
		_, aerr2 := h.request(func(*access) {})
		s.expect("next check after the end and the release", true, errors.Is(aerr2, errHostDisabled))
	}
	s.worked(1)
}
