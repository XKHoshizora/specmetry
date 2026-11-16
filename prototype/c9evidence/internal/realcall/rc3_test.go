package realcall

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"slices"
	"syscall"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/toolctl"
)

// RC-3's independent expectation (Review 155 B-154-4-R-C, review-resolution
// §204). This file has no build tag: its judgement is a pure function of a
// published copy, so its unit test (constructed copies, never a real call)
// runs in step A too.
//
// The start task is held before its one Wait while the clock reaches W.
// That holds the parent, not the child: go may finish by itself before the
// request. So the facts are judged apart (the parent's expiry; whether the
// request was admitted, made and what it returned; how the child ended;
// what the one Wait reaped; every fault and the lateness) and only these
// branches are legal, by the current protocol and the fixed platforms'
// behaviour:
//
//   - Linux, ended by the request: the request through the pidfd made once
//     and nil; the end SIGKILL; the faults exactly the Wait's exit status.
//   - Linux, exited before the request: the request made once and nil (a
//     process not yet reaped accepts the signal and ignores it: the
//     mainline kernel's kill_pid_info still finds it and prepare_signal
//     ignores a signal after a group exit); the end exited 0; no fault.
//   - Windows, ended by the request: Kill made once and nil; the end the
//     exit code 1 of TerminateProcess; the faults exactly the Wait's exit
//     status.
//   - Windows, exited before the request: Kill made once and failed with
//     TerminateProcess's access denied (Go 1.27.3's os/exec_windows.go
//     calls TerminateProcess(h, 1); Microsoft documents that it fails with
//     ERROR_ACCESS_DENIED once the process has terminated); the end exited
//     0; the faults exactly that request error.
//   - darwin: no request (§12.14 item 8), the cleanup unknown by the one
//     fixed reason; the end exited 0; no fault.
//
// Everything else (another signal, a normal non-zero exit, another request
// error, an extra fault, the call counted as a success, no expiry) is not a
// legal branch. Whatever the child's end, the expired call is never a
// success.

// The legal branches.
const (
	rc3EndedByRequest = "ended by the request"
	rc3ExitedBefore   = "exited before the request"
	rc3NoRequest      = "no request (darwin)"
)

// darwinNoRequest is the one cleanup-unknown reason on darwin, written
// independently of toolctl.
const darwinNoRequest = "no termination request on this platform (darwin: os/wait_unimp.go); the tool may outlive the test process"

// faultKind is a fault's kind by its identity: an exit status with the
// end's identity, TerminateProcess's access denied, or "other" (every other
// fault is unexpected in the real-call tests).
func faultKind(f toolctl.Fault) string {
	if f.Exit != nil {
		return "exit-status " + toolctl.ExitKind(f.Exit)
	}
	var ee *exec.ExitError
	if errors.As(f.Err, &ee) {
		return "exit-status " + toolctl.ExitKindOf(ee.ProcessState)
	}
	var se *os.SyscallError
	if errors.As(f.Err, &se) && se.Syscall == "TerminateProcess" && errors.Is(se.Err, fs.ErrPermission) {
		return "TerminateProcess access-denied"
	}
	return "other"
}

// faultSet is the record's structured fault set as "<site> <source>
// <kind>", in the record's order.
func faultSet(s toolctl.Snapshot) []string {
	var out []string
	for _, f := range s.Faults {
		out = append(out, f.Site+" "+f.Source+" "+faultKind(f))
	}
	return out
}

func anyLate(es []toolctl.Event) bool {
	for _, e := range es {
		if e.Late {
			return true
		}
	}
	return false
}

// judgeRC3 returns RC-3's branch, or the reasons the copy is in none.
// succeeded is the collector's decision on the call (CallResult.Succeeded
// at W).
func judgeRC3(s toolctl.Snapshot, succeeded bool) (string, []string) {
	var p []string
	if succeeded || s.Success {
		p = append(p, "the expired call is counted as a success")
	}
	if !s.Expired {
		p = append(p, "the parent's expiry at W is not recorded")
	}
	if !anyLate(s.Events) {
		p = append(p, "no result is marked late")
	}
	if s.State != toolctl.StWaited || s.Class != "B" || !s.WaitCalled || !s.WaitReturned || s.Exit == nil {
		p = append(p, fmt.Sprintf("the one Wait's reap is not confirmed: state %s class %s wait %v/%v exit %v", s.State, s.Class, s.WaitCalled, s.WaitReturned, s.Exit != nil))
	}
	for i, o := range s.Out {
		if !o.Done || !o.EOF || o.ReadErr != "" || o.Overflow {
			p = append(p, fmt.Sprintf("%s: done %v eof %v read error %q overflow %v", toolctl.StreamNames[i], o.Done, o.EOF, o.ReadErr, o.Overflow))
		}
	}
	if len(s.NotAdmit) > 0 || len(s.Injected) > 0 || s.Panic != "" || len(s.Skipped) > 0 {
		p = append(p, fmt.Sprintf("not-admitted %q injected %q panic %q skipped %q", s.NotAdmit, s.Injected, s.Panic, s.Skipped))
	}
	faults, end, req := faultSet(s), toolctl.ExitKind(s.Exit), s.Req
	made := req.State == toolctl.ReqDone && req.Sends == 1
	branch := ""
	switch s.Mode {
	case toolctl.ModePidfd:
		switch {
		case made && req.Err == "" && end == "signaled SIGKILL" && slices.Equal(faults, []string{"wait operation exit-status signaled SIGKILL"}) && len(s.Unknown) == 0:
			branch = rc3EndedByRequest
		case made && req.Err == "" && end == "exited 0" && len(faults) == 0 && len(s.Unknown) == 0:
			branch = rc3ExitedBefore
		}
	case toolctl.ModeKill:
		switch {
		case made && req.Err == "" && end == "exited 1" && slices.Equal(faults, []string{"wait operation exit-status exited 1"}) && len(s.Unknown) == 0:
			branch = rc3EndedByRequest
		case made && req.Err != "" && end == "exited 0" && slices.Equal(faults, []string{"request operation TerminateProcess access-denied"}) && len(s.Unknown) == 0:
			branch = rc3ExitedBefore
		}
	case toolctl.ModeNone:
		if req.State == toolctl.ReqNone && req.Sends == 0 && end == "exited 0" && len(faults) == 0 && slices.Equal(s.Unknown, []string{darwinNoRequest}) {
			branch = rc3NoRequest
		}
	}
	if branch == "" {
		p = append(p, fmt.Sprintf("no legal branch (mode %s): request %s sends %d error %q, end %s, faults %q, unknown %q",
			s.Mode, req.State, req.Sends, req.Err, end, faults, s.Unknown))
	}
	if len(p) > 0 {
		return "", p
	}
	return branch, nil
}

// rc3Facts are RC-3's facts, each recorded on its own for the evidence.
func rc3Facts(s toolctl.Snapshot, succeeded bool) map[string]any {
	expiredAt := ""
	var events []string
	for _, e := range s.Events {
		if e.What == "expired at W" && expiredAt == "" {
			expiredAt = e.At.Format("2006-01-02T15:04:05.999999999Z07:00")
		}
		l := e.What
		if e.Late {
			l += " (late)"
		}
		events = append(events, l)
	}
	reqKind := ""
	for _, f := range s.Faults {
		if f.Site == "request" {
			reqKind = faultKind(f)
		}
	}
	branch, problems := judgeRC3(s, succeeded)
	return map[string]any{
		"parent":    map[string]any{"expired": s.Expired, "expired_at": expiredAt, "late": anyLate(s.Events), "decision_success": succeeded, "item11_success": s.Success},
		"request":   map[string]any{"mode": s.Req.Mode, "admitted": len(s.NotAdmit) == 0 && s.Req.State != toolctl.ReqNone, "not_admitted": s.NotAdmit, "state": s.Req.State, "sends": s.Req.Sends, "error": s.Req.Err, "error_kind": reqKind},
		"child_end": map[string]any{"identity": toolctl.ExitKind(s.Exit), "exit": s.Exit},
		"wait":      map[string]any{"called": s.WaitCalled, "returned": s.WaitReturned, "state": s.State, "class": s.Class},
		"faults":    faultSet(s),
		"unknown":   s.Unknown,
		"events":    events,
		"branch":    branch,
		"problems":  problems,
	}
}

// TestRC3Branches is C1 (review-resolution §204): RC-3's judgement on
// constructed copies (never a real call): each legal branch is accepted
// with its name; every other combination is rejected with a reason.
func TestRC3Branches(t *testing.T) {
	base := func(mode string) toolctl.Snapshot {
		s := toolctl.Snapshot{Mode: mode, State: toolctl.StWaited, Class: "B", WaitCalled: true, WaitReturned: true, Expired: true,
			Events: []toolctl.Event{{What: "expired at W", Late: true}, {What: "waited", Late: true}}, Req: toolctl.Request{Mode: mode, State: toolctl.ReqNone}}
		for i := range s.Out {
			s.Out[i] = toolctl.Stream{Started: true, Done: true, EOF: true}
		}
		return s
	}
	exited := func(code int) *toolctl.Exit { return &toolctl.Exit{Exited: true, Code: code} }
	signaled := func(name string) *toolctl.Exit { return &toolctl.Exit{Signaled: true, Signal: name, Code: -1} }
	waitFault := func(e *toolctl.Exit) toolctl.Fault {
		return toolctl.Fault{Site: "wait", Source: toolctl.SourceOperation, Exit: e, Err: &exec.ExitError{}}
	}
	reqFault := func(err error) toolctl.Fault {
		return toolctl.Fault{Site: "request", Source: toolctl.SourceOperation, Err: err}
	}
	denied := &os.SyscallError{Syscall: "TerminateProcess", Err: fs.ErrPermission}
	done := func(err error) toolctl.Request {
		r := toolctl.Request{State: toolctl.ReqDone, Sends: 1}
		if err != nil {
			r.Err = err.Error()
		}
		return r
	}
	type tc struct {
		name      string
		mode      string
		req       toolctl.Request
		end       *toolctl.Exit
		faults    []toolctl.Fault
		unknown   []string
		succeeded bool
		change    func(*toolctl.Snapshot)
		want      string // "" when rejected
	}
	sigkill, exit1 := signaled("SIGKILL"), exited(1)
	cases := []tc{
		{name: "linux ended by the request", mode: toolctl.ModePidfd, req: done(nil), end: sigkill, faults: []toolctl.Fault{waitFault(sigkill)}, want: rc3EndedByRequest},
		{name: "linux exited before the request", mode: toolctl.ModePidfd, req: done(nil), end: exited(0), want: rc3ExitedBefore},
		{name: "linux request error ESRCH", mode: toolctl.ModePidfd, req: done(syscall.ESRCH), end: exited(0), faults: []toolctl.Fault{reqFault(syscall.ESRCH)}},
		{name: "linux another signal", mode: toolctl.ModePidfd, req: done(nil), end: signaled("SIGTERM"), faults: []toolctl.Fault{waitFault(signaled("SIGTERM"))}},
		{name: "linux a normal non-zero exit", mode: toolctl.ModePidfd, req: done(nil), end: exit1, faults: []toolctl.Fault{waitFault(exit1)}},
		{name: "linux an extra fault", mode: toolctl.ModePidfd, req: done(nil), end: sigkill,
			faults: []toolctl.Fault{waitFault(sigkill), {Site: "close stderr-r", Source: toolctl.SourceOperation, Err: errors.New("r155: constructed")}}},
		{name: "linux an extra cleanup unknown", mode: toolctl.ModePidfd, req: done(nil), end: exited(0), unknown: []string{"something"}},
		{name: "linux no request", mode: toolctl.ModePidfd, end: exited(0)},
		{name: "linux counted as a success", mode: toolctl.ModePidfd, req: done(nil), end: exited(0), succeeded: true},
		{name: "linux not expired", mode: toolctl.ModePidfd, req: done(nil), end: exited(0), change: func(s *toolctl.Snapshot) { s.Expired = false }},
		{name: "linux nothing late", mode: toolctl.ModePidfd, req: done(nil), end: exited(0), change: func(s *toolctl.Snapshot) { s.Events = nil }},
		{name: "linux reap not returned", mode: toolctl.ModePidfd, req: done(nil), end: exited(0), change: func(s *toolctl.Snapshot) { s.WaitReturned = false }},
		{name: "windows ended by the request", mode: toolctl.ModeKill, req: done(nil), end: exit1, faults: []toolctl.Fault{waitFault(exit1)}, want: rc3EndedByRequest},
		{name: "windows exited before the request", mode: toolctl.ModeKill, req: done(denied), end: exited(0), faults: []toolctl.Fault{reqFault(denied)}, want: rc3ExitedBefore},
		{name: "windows access denied but exit 1", mode: toolctl.ModeKill, req: done(denied), end: exit1, faults: []toolctl.Fault{reqFault(denied), waitFault(exit1)}},
		{name: "windows Kill nil but exit 0", mode: toolctl.ModeKill, req: done(nil), end: exited(0)},
		{name: "windows another request error", mode: toolctl.ModeKill, req: done(errors.New("DuplicateHandle")), end: exited(0),
			faults: []toolctl.Fault{reqFault(&os.SyscallError{Syscall: "DuplicateHandle", Err: fs.ErrPermission})}},
		{name: "windows exit 2", mode: toolctl.ModeKill, req: done(nil), end: exited(2), faults: []toolctl.Fault{waitFault(exited(2))}},
		{name: "darwin no request", mode: toolctl.ModeNone, end: exited(0), unknown: []string{darwinNoRequest}, want: rc3NoRequest},
		{name: "darwin a normal non-zero exit", mode: toolctl.ModeNone, end: exited(3), faults: []toolctl.Fault{waitFault(exited(3))}, unknown: []string{darwinNoRequest}},
		{name: "darwin a request", mode: toolctl.ModeNone, req: done(nil), end: exited(0), unknown: []string{darwinNoRequest}},
		{name: "darwin no cleanup unknown", mode: toolctl.ModeNone, end: exited(0)},
		{name: "darwin a read error", mode: toolctl.ModeNone, end: exited(0), unknown: []string{darwinNoRequest},
			change: func(s *toolctl.Snapshot) { s.Out[1].ReadErr = "constructed" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := base(c.mode)
			if c.req.State != "" {
				c.req.Mode = c.mode
				s.Req = c.req
			}
			s.Exit, s.Faults, s.Unknown = c.end, c.faults, c.unknown
			if c.change != nil {
				c.change(&s)
			}
			got, why := judgeRC3(s, c.succeeded)
			// A legal branch has no reason; a rejection has at least one.
			if got != c.want || (c.want == "") != (len(why) > 0) {
				t.Errorf("branch %q (reasons %q), want %q", got, why, c.want)
			}
		})
	}
}
