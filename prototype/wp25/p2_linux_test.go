//go:build linux

package wp25

// P-2 (spec v7 §19.6 and §19.12, G6; plan fixed in
// docs/wp2.5-prototype.md §14.4): the Linux capability probe where seccomp
// denies one of the calls it needs. Each group runs in an experiment
// process of its own, this test binary run again: it installs its filter
// on itself (no_new_privs, then SECCOMP_FILTER_FLAG_TSYNC, so every thread
// of that process and nothing else), probes in the order of §19.6, and,
// when the probe holds, sets the subreaper, starts, observes and reaps a
// minimal check, and clears the subreaper. It reports every native return
// and waits for its release; meanwhile the outer reads the process's own
// /proc entries. Nothing here changes the runner's configuration.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const p2OutEnv = "WP25_P2_OUT"

// p2Step is one native call of the experiment and its errno.
type p2Step struct {
	Call  string `json:"call"`
	Errno string `json:"errno,omitempty"` // "" when it succeeded
	Value string `json:"value,omitempty"`
	// EINTR is how often the call was interrupted and retried within its
	// deadline, the only retry the experiment makes.
	EINTR int `json:"eintr,omitempty"`
}

// p2Report is what the experiment process reports before its release.
type p2Report struct {
	Group       string   `json:"group"`
	PID         int      `json:"pid"`
	Unsupported string   `json:"unsupported,omitempty"`
	Status      string   `json:"status"` // its own read of /proc/self/status
	StatusErr   string   `json:"status_error,omitempty"`
	Steps       []p2Step `json:"steps"`
	Refused     string   `json:"refused,omitempty"` // the capability that did not hold
	CheckPID    int      `json:"check_pid,omitempty"`
	CheckStatus string   `json:"check_status,omitempty"`
	CheckReaped bool     `json:"check_reaped"`
	ClearErr    string   `json:"clear_error,omitempty"`
	SubEnd      int      `json:"subreaper_end"`
	Next        string   `json:"next_probe,omitempty"` // after a failed clear
	Exit        int      `json:"exit"`
}

// p2Rule denies one system call, or one call with given arguments (their
// low 32 bits at the offsets of struct seccomp_data), with an errno.
type p2Rule struct {
	nr    uint32
	args  [][2]uint32
	errno syscall.Errno
}

var p2Rules = map[string]p2Rule{
	"pidfd_open.EPERM":        {nr: unix.SYS_PIDFD_OPEN, errno: unix.EPERM},
	"pidfd_open.ENOSYS":       {nr: unix.SYS_PIDFD_OPEN, errno: unix.ENOSYS},
	"pidfd_send_signal.EPERM": {nr: unix.SYS_PIDFD_SEND_SIGNAL, errno: unix.EPERM},
	"waitid.EPERM":            {nr: unix.SYS_WAITID, errno: unix.EPERM},
	"prctl.set1":              {nr: unix.SYS_PRCTL, args: [][2]uint32{{16, unix.PR_SET_CHILD_SUBREAPER}, {24, 1}}, errno: unix.EPERM},
	"prctl.set0":              {nr: unix.SYS_PRCTL, args: [][2]uint32{{16, unix.PR_SET_CHILD_SUBREAPER}, {24, 0}}, errno: unix.EPERM},
}

func p2Arch() (uint32, bool) {
	switch runtime.GOARCH {
	case "amd64":
		return unix.AUDIT_ARCH_X86_64, true
	case "arm64":
		return unix.AUDIT_ARCH_AARCH64, true
	}
	return 0, false
}

// p2Filter is a classic BPF program: another architecture or another
// call is allowed; the rule's call with the rule's arguments returns the
// errno.
func p2Filter(arch uint32, r p2Rule) []unix.SockFilter {
	ld := func(off uint32) unix.SockFilter {
		return unix.SockFilter{Code: unix.BPF_LD | unix.BPF_W | unix.BPF_ABS, K: off}
	}
	jeq := func(k uint32) unix.SockFilter {
		return unix.SockFilter{Code: unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K, K: k}
	}
	ret := func(k uint32) unix.SockFilter { return unix.SockFilter{Code: unix.BPF_RET | unix.BPF_K, K: k} }
	var prog []unix.SockFilter
	var jumps []int
	prog = append(prog, ld(4))
	jumps = append(jumps, len(prog))
	prog = append(prog, jeq(arch), ld(0))
	jumps = append(jumps, len(prog))
	prog = append(prog, jeq(r.nr))
	for _, a := range r.args {
		prog = append(prog, ld(a[0]))
		jumps = append(jumps, len(prog))
		prog = append(prog, jeq(a[1]))
	}
	prog = append(prog, ret(unix.SECCOMP_RET_ERRNO|uint32(r.errno)&unix.SECCOMP_RET_DATA))
	allow := len(prog)
	prog = append(prog, ret(unix.SECCOMP_RET_ALLOW))
	for _, j := range jumps {
		prog[j].Jf = uint8(allow - j - 1)
	}
	return prog
}

// p2Install sets no_new_privs and installs the filter on every thread of
// this process.
func p2Install(prog []unix.SockFilter) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("PR_SET_NO_NEW_PRIVS: %w", err)
	}
	fp := unix.SockFprog{Len: uint16(len(prog)), Filter: &prog[0]}
	r1, _, e := unix.Syscall(unix.SYS_SECCOMP, unix.SECCOMP_SET_MODE_FILTER, unix.SECCOMP_FILTER_FLAG_TSYNC, uintptr(unsafe.Pointer(&fp)))
	if e != 0 {
		return fmt.Errorf("seccomp: %w", e)
	}
	if r1 != 0 {
		return fmt.Errorf("seccomp: thread %d could not be synchronised", r1)
	}
	return nil
}

// seccompStatus reads NoNewPrivs, Seccomp and Seccomp_filters of a
// process's status file; a read that fails is an error, not a status.
func seccompStatus(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	v := map[string]string{}
	for _, l := range strings.Split(string(b), "\n") {
		if k, val, ok := strings.Cut(l, ":"); ok {
			v[k] = strings.TrimSpace(val)
		}
	}
	return fmt.Sprintf("NoNewPrivs=%s Seccomp=%s Seccomp_filters=%s", v["NoNewPrivs"], v["Seccomp"], v["Seccomp_filters"]), nil
}

func errnoOf(err error) string {
	if err == nil {
		return ""
	}
	return errnoName(err)
}

func (r *p2Report) step(call string, err error, value string) {
	r.Steps = append(r.Steps, p2Step{Call: call, Errno: errnoOf(err), Value: value})
}

func (r *p2Report) get(when string) int {
	var v int32
	err := unix.Prctl(unix.PR_GET_CHILD_SUBREAPER, uintptr(unsafe.Pointer(&v)), 0, 0, 0)
	r.step("prctl(PR_GET_CHILD_SUBREAPER) "+when, err, strconv.Itoa(int(v)))
	if err != nil {
		return -1
	}
	return int(v)
}

// p2Probe is §19.6, in order; it returns the capability that did not
// hold, "" when all held. Its own pidfd is closed on every path.
func p2Probe(r *p2Report) string {
	switch v := r.get("(probe)"); v {
	case -1:
		return "subreaper (read)"
	case 0:
	default:
		return "subreaper (already set)"
	}
	fd, err := unix.PidfdOpen(os.Getpid(), 0)
	r.step("pidfd_open(self)", err, "")
	if err != nil {
		return "pidfd_open"
	}
	closeFD := func() bool {
		cerr := unix.Close(fd)
		if injectOnce("p2-close-fail-once", "p2Probe.close", os.Getpid()) {
			cerr = errors.New("injected: close failed once") // the real call was made
		}
		r.step("close(own pidfd)", cerr, "")
		return cerr == nil
	}
	err = unix.PidfdSendSignal(fd, 0, nil, 0)
	r.step("pidfd_send_signal(own, 0)", err, "")
	if err != nil {
		closeFD()
		return "pidfd_send_signal"
	}
	var info unix.Siginfo
	err = unix.Waitid(unix.P_PIDFD, fd, &info, unix.WEXITED|unix.WNOHANG, nil)
	r.step("waitid(P_PIDFD, own, WEXITED|WNOHANG)", err, "")
	if !errors.Is(err, unix.ECHILD) {
		closeFD()
		return "waitid(P_PIDFD)"
	}
	n, err := unix.Poll([]unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}, 0)
	r.step("poll(own pidfd, 0)", err, strconv.Itoa(n))
	if err != nil || n != 0 {
		closeFD()
		return "poll(pidfd)"
	}
	_, pr := procInfo(os.Getpid())
	_, derr := os.ReadDir("/proc")
	r.step("/proc", derr, fmt.Sprint(pr == present))
	if pr != present || derr != nil {
		closeFD()
		return "/proc"
	}
	if !closeFD() {
		return "pidfd close"
	}
	return ""
}

// p2Check starts a minimal check that reports itself, observes its exit
// without reaping it (a bounded poll on its pidfd, then waitid WNOWAIT,
// each retried on EINTR within its deadline), waits until the outer has
// registered it, then reaps it. A step that fails stops the observation;
// once registered, the check, this process's own unreaped child, is ended
// and reaped within a bound, never through a wait without one (Review 095
// B1).
func p2Check(r *p2Report, ack string) {
	cmd := helper("report-exit", nil)
	if err := start(cmd); err != nil {
		r.step("start the check", err, "")
		return
	}
	r.CheckPID = cmd.Process.Pid
	r.step("start the check", nil, "")
	ok := p2Observe(r)
	// Registered whatever the observation gave: the outer verifies the
	// check while it is still this process's child, and only then may it
	// be ended and reaped.
	if !p2AwaitAck(r, ack, 10*time.Second) {
		ok = false
	}
	if !ok {
		_ = unix.Kill(r.CheckPID, unix.SIGKILL)
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
		r.step("reap(check)", nil, cmd.ProcessState.String())
		r.CheckReaped = true
	case <-time.After(5 * time.Second):
		r.step("reap(check)", errors.New("not reaped within 5 s"), "")
	}
}

// p2Observe is the check's E-main: the pidfd, a bounded poll, the status
// without reaping; its pidfd is closed on every path.
func p2Observe(r *p2Report) bool {
	fd, err := unix.PidfdOpen(r.CheckPID, 0)
	r.step("pidfd_open(check)", err, "")
	if err != nil {
		return false
	}
	ok := false
	deadline := time.Now().Add(5 * time.Second)
	n, eintr, perr := 0, 0, error(nil)
	for {
		n, perr = unix.Poll([]unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}, pollMillis(deadline))
		if !errors.Is(perr, unix.EINTR) || time.Now().After(deadline) {
			break
		}
		eintr++
	}
	if injectOnce("p2-check-poll-fail-once", "p2Check.poll", r.CheckPID) {
		perr = errors.New("injected: poll failed once") // the real call was made
	}
	r.Steps = append(r.Steps, p2Step{"poll(check pidfd, 5 s)", errnoOf(perr), strconv.Itoa(n), eintr})
	if perr == nil && n == 1 {
		st, werr := "", error(nil)
		eintr = 0
		for {
			st, werr = waitidNoWait(r.CheckPID)
			if !errors.Is(werr, unix.EINTR) || time.Now().After(deadline) {
				break
			}
			eintr++
		}
		r.Steps = append(r.Steps, p2Step{"waitid(P_PID, check, WEXITED|WNOWAIT)", errnoOf(werr), st, eintr})
		r.CheckStatus = st
		ok = werr == nil
	}
	cerr := unix.Close(fd)
	r.step("close(check pidfd)", cerr, "")
	return ok && cerr == nil
}

// p2AwaitAck waits for the outer to register the check (its file at
// ack), while the check is still this process's unreaped child.
func p2AwaitAck(r *p2Report, ack string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for {
		_, err := os.Stat(ack)
		switch {
		case err == nil:
			r.step("await the outer's registration", nil, "")
			return true
		case !errors.Is(err, os.ErrNotExist):
			r.step("await the outer's registration", err, "")
			return false
		case time.Now().After(deadline):
			r.step("await the outer's registration", errors.New("timed out"), "")
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func p2Experiment(r *p2Report) int {
	if rule, ok := p2Rules[r.Group]; ok {
		arch, ok := p2Arch()
		if !ok {
			r.Unsupported = "no audit architecture for " + runtime.GOARCH
			return 0
		}
		if err := p2Install(p2Filter(arch, rule)); err != nil {
			r.Unsupported = err.Error()
			return 0
		}
	} else if r.Group != "normal" {
		r.Unsupported = "unknown group"
		return 0
	}
	st, serr := seccompStatus("/proc/self/status")
	r.Status = st
	if serr != nil {
		r.StatusErr = serr.Error()
	}
	if refused := p2Probe(r); refused != "" {
		r.Refused, r.SubEnd = refused, r.get("(end)")
		return 6
	}
	// S0, step 5: set, then read back; then the check; then the clear.
	err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0)
	r.step("prctl(PR_SET_CHILD_SUBREAPER, 1)", err, "")
	if err != nil {
		r.Refused, r.SubEnd = "subreaper (set)", r.get("(end)")
		return 6
	}
	if v := r.get("(after the set)"); v != 1 {
		r.step("prctl(PR_SET_CHILD_SUBREAPER, 0)", unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 0, 0, 0, 0), "")
		r.Refused, r.SubEnd = "subreaper (not read back)", r.get("(end)")
		return 6
	}
	p2Check(r, os.Getenv(p2OutEnv)+".ack")
	err = unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 0, 0, 0, 0)
	r.step("prctl(PR_SET_CHILD_SUBREAPER, 0)", err, "")
	if err != nil {
		// The attribute stays: an environment error, the batch stops, and
		// a later probe refuses on GET = 1.
		r.ClearErr, r.SubEnd = errnoOf(err), r.get("(after the failed clear)")
		var next p2Report
		if ref := p2Probe(&next); ref != "" {
			r.Next = "refused: " + ref
		} else {
			r.Next = "admitted"
		}
		return 6
	}
	r.SubEnd = r.get("(end)")
	return 0
}

func p2Helper(mode string, args []string) (int, bool) {
	if mode != "p2-probe" {
		return 0, false
	}
	r := &p2Report{PID: os.Getpid(), SubEnd: -1}
	if len(args) > 0 {
		r.Group = args[0]
	}
	code := p2Experiment(r)
	r.Exit = code
	b, err := json.Marshal(r)
	if err != nil {
		return exitReport, true
	}
	out := os.Getenv(p2OutEnv)
	if os.WriteFile(out+".tmp", b, 0o600) != nil || os.Rename(out+".tmp", out) != nil {
		return exitReport, true
	}
	awaitRelease()
	return code, true
}

// p2Result is one step's result: "ok" and its value, or its errno.
func p2Result(r p2Report, call string) string {
	for _, st := range r.Steps {
		if st.Call == call {
			if st.Errno != "" {
				return st.Errno
			}
			return strings.TrimSpace("ok " + st.Value)
		}
	}
	return "not reached"
}

// pidfdsOf counts the pidfds among a process's descriptors.
func pidfdsOf(pid int) (int, error) {
	dir := "/proc/" + strconv.Itoa(pid) + "/fd"
	ents, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range ents {
		l, err := os.Readlink(filepath.Join(dir, e.Name()))
		if err != nil {
			return 0, err
		}
		if l == "anon_inode:[pidfd]" {
			n++
		}
	}
	return n, nil
}

var p2Budget = &budget{name: "P-2", total: 3 * time.Minute, per: 30 * time.Second}

func TestP2(t *testing.T) {
	runs := []string{"normal.1", "normal.2", "normal.3", "pidfd_open.EPERM", "pidfd_open.ENOSYS",
		"pidfd_send_signal.EPERM", "waitid.EPERM", "prctl.set1", "prctl.set0"}
	for _, id := range runs {
		t.Run(id, func(t *testing.T) { runP2(t, id) })
	}
}

// p2Exp is one declared step of a group: the call, its planned errno and
// value. An observation step's value is the experiment's own load (a
// different value is a failed observation); a probe step's value is the
// mechanism's (a different value is a finding).
type p2Exp struct {
	call, errno, value string
	obs                bool
}

var (
	p2ProbeSteps = []p2Exp{
		{"prctl(PR_GET_CHILD_SUBREAPER) (probe)", "", "0", false},
		{"pidfd_open(self)", "", "", false},
		{"pidfd_send_signal(own, 0)", "", "", false},
		{"waitid(P_PIDFD, own, WEXITED|WNOHANG)", "ECHILD", "", false},
		{"poll(own pidfd, 0)", "", "0", false},
		{"/proc", "", "true", false},
		{"close(own pidfd)", "", "", true},
	}
	p2SetSteps = []p2Exp{
		{"prctl(PR_SET_CHILD_SUBREAPER, 1)", "", "", false},
		{"prctl(PR_GET_CHILD_SUBREAPER) (after the set)", "", "1", false},
	}
	p2CheckSteps = []p2Exp{
		{"start the check", "", "", true},
		{"pidfd_open(check)", "", "", true},
		{"poll(check pidfd, 5 s)", "", "1", true},
		{"waitid(P_PID, check, WEXITED|WNOWAIT)", "", "exit 0", true},
		{"close(check pidfd)", "", "", true},
		{"await the outer's registration", "", "", true},
		{"reap(check)", "", "exit status 0", true},
	}
	getEnd0 = p2Exp{"prctl(PR_GET_CHILD_SUBREAPER) (end)", "", "0", false}
)

func p2Seq(parts ...[]p2Exp) []p2Exp {
	var out []p2Exp
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

var p2Steps = map[string][]p2Exp{
	"normal":                  p2Seq(p2ProbeSteps, p2SetSteps, p2CheckSteps, []p2Exp{{"prctl(PR_SET_CHILD_SUBREAPER, 0)", "", "", false}, getEnd0}),
	"pidfd_open.EPERM":        p2Seq(p2ProbeSteps[:1], []p2Exp{{"pidfd_open(self)", "EPERM", "", false}, getEnd0}),
	"pidfd_open.ENOSYS":       p2Seq(p2ProbeSteps[:1], []p2Exp{{"pidfd_open(self)", "ENOSYS", "", false}, getEnd0}),
	"pidfd_send_signal.EPERM": p2Seq(p2ProbeSteps[:2], []p2Exp{{"pidfd_send_signal(own, 0)", "EPERM", "", false}, p2ProbeSteps[6], getEnd0}),
	"waitid.EPERM":            p2Seq(p2ProbeSteps[:3], []p2Exp{{"waitid(P_PIDFD, own, WEXITED|WNOHANG)", "EPERM", "", false}, p2ProbeSteps[6], getEnd0}),
	"prctl.set1":              p2Seq(p2ProbeSteps, []p2Exp{{"prctl(PR_SET_CHILD_SUBREAPER, 1)", "EPERM", "", false}, getEnd0}),
	"prctl.set0":              p2Seq(p2ProbeSteps, p2SetSteps, p2CheckSteps, []p2Exp{{"prctl(PR_SET_CHILD_SUBREAPER, 0)", "EPERM", "", false}, {"prctl(PR_GET_CHILD_SUBREAPER) (after the failed clear)", "", "1", false}}),
}

// p2Judge checks every step against the group's declaration (Review 095
// B1): an errno other than the planned one, at any step, is a failed
// observation of the run (the planned denial that did not happen is a
// fault that did not hit); so is an observation step's other value. A
// different sequence of calls, or a probe step's other value, is the
// mechanism taking another path: a finding.
func p2Judge(s *scen, group string, r p2Report) {
	exp := p2Steps[group]
	var wantSeq, gotSeq, probeDiff []string
	for _, e := range exp {
		wantSeq = append(wantSeq, e.call)
	}
	for i, st := range r.Steps {
		gotSeq = append(gotSeq, st.Call)
		if i >= len(exp) || exp[i].call != st.Call {
			continue
		}
		e := exp[i]
		switch {
		case st.Errno != e.errno && e.errno != "" && e.errno != "ECHILD" && st.Errno == "":
			s.flag("the fault did not hit: %s succeeded, planned %s", st.Call, e.errno)
		case st.Errno != e.errno:
			s.flag("observation failed: unplanned result of %s: %q, planned %q", st.Call, st.Errno, e.errno)
		case st.Value != e.value && e.obs:
			s.flag("load not established: %s gave %q, planned %q", st.Call, st.Value, e.value)
		case st.Value != e.value:
			probeDiff = append(probeDiff, fmt.Sprintf("%s=%q", st.Call, st.Value))
		}
	}
	s.expect("step sequence", strings.Join(wantSeq, " > "), strings.Join(gotSeq, " > "))
	s.expectTrue("probe values as planned", len(probeDiff) == 0, strings.Join(probeDiff, ", "))
}

// p2Want is a group's fixed expectation (§14.4).
type p2Want struct {
	status, refused, denied, deniedErrno string
	check                                bool
	subEnd                               int
	clearErr, next                       string
	exit                                 int
}

var p2Wants = map[string]p2Want{
	"normal":                  {status: "NoNewPrivs=0 Seccomp=0 Seccomp_filters=0", check: true},
	"pidfd_open.EPERM":        {refused: "pidfd_open", denied: "pidfd_open(self)", deniedErrno: "EPERM", exit: 6},
	"pidfd_open.ENOSYS":       {refused: "pidfd_open", denied: "pidfd_open(self)", deniedErrno: "ENOSYS", exit: 6},
	"pidfd_send_signal.EPERM": {refused: "pidfd_send_signal", denied: "pidfd_send_signal(own, 0)", deniedErrno: "EPERM", exit: 6},
	"waitid.EPERM":            {refused: "waitid(P_PIDFD)", denied: "waitid(P_PIDFD, own, WEXITED|WNOHANG)", deniedErrno: "EPERM", exit: 6},
	"prctl.set1":              {refused: "subreaper (set)", denied: "prctl(PR_SET_CHILD_SUBREAPER, 1)", deniedErrno: "EPERM", exit: 6},
	"prctl.set0":              {denied: "prctl(PR_SET_CHILD_SUBREAPER, 0)", deniedErrno: "EPERM", check: true, subEnd: 1, clearErr: "EPERM", next: "refused: subreaper (already set)", exit: 6},
}

func runP2(t *testing.T, id string) {
	group, _, _ := strings.Cut(id, ".")
	if !strings.HasPrefix(id, "normal") {
		group = id
	}
	w := p2Wants[group]
	if w.status == "" {
		w.status = "NoNewPrivs=1 Seccomp=2 Seccomp_filters=1"
	}
	k := refusal
	switch group {
	case "normal":
		k = normal
	case "prctl.set0":
		k = fault
	}
	s := begin(t, "G6", "P2."+id, k, true, true)
	defer s.done()
	if ok, left := p2Budget.allow(); !ok {
		s.invalidf("not executed: the total deadline of P-2 has %v left, less than one run's %v", left.Round(time.Second), p2Budget.per)
	}
	out := filepath.Join(s.t.TempDir(), "p2.json")
	checks := 0
	if w.check {
		checks = 1
	}
	rep := s.newReport(reportSpec{"pid": checks})
	cmd := helper("p2-probe", []string{p2OutEnv + "=" + out, reportEnv + "=" + rep}, group)
	release := gated(s, cmd)
	startHelper(s, cmd)
	pid := cmd.Process.Pid
	if checks == 1 {
		// The check is the experiment process's own child; the outer
		// registers it from its record while it can still verify that
		// (Review 095: every created process is registered).
		s.awaitOwned(rep, 1, 10*time.Second, func(cpid int) {
			st, pr := procInfo(cpid)
			switch {
			case pr != present:
				s.invalidf("ownership not established: the check %d cannot be read", cpid)
			case st.ppid != pid:
				s.invalidf("ownership not established: %d is not a child of the experiment process %d", cpid, pid)
			}
			s.own(&pidProc{pid: cpid, id: st.start})
		})
		if err := os.WriteFile(out+".ack", nil, 0o600); err != nil {
			s.invalidf("the acknowledgement of the registration: %v", err)
		}
	}
	var r p2Report
	deadline := time.Now().Add(15 * time.Second)
	for {
		b, err := os.ReadFile(out)
		if err == nil {
			if jerr := json.Unmarshal(b, &r); jerr != nil {
				s.invalidf("load not established: the experiment's report does not parse: %v", jerr)
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			s.invalidf("observation failed: the experiment's report: %v", err)
		}
		if time.Now().After(deadline) {
			s.invalidf("load not established: no report from the experiment process within 15 s")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if r.PID != pid || r.Group != group {
		s.invalidf("load not established: a report of pid %d, group %q", r.PID, r.Group)
	}
	if r.Unsupported != "" {
		s.unsupported("the filter cannot be installed here: %s", r.Unsupported)
	}
	// The outer, while the experiment process waits for its release.
	outer, oerr := seccompStatus("/proc/" + strconv.Itoa(pid) + "/status")
	if injectOnce("p2-status-unreadable-once", "runP2.outer-status", pid) {
		outer, oerr = "", errors.New("injected: the read failed once") // the real read was made
	}
	if oerr != nil {
		s.flag("observation failed: the outer's read of the experiment process's status: %v", oerr)
	}
	if r.StatusErr != "" {
		s.flag("observation failed: the experiment process's read of its own status: %s", r.StatusErr)
	}
	kids, kerr := childrenOf(pid)
	if kerr != nil {
		s.flag("observation failed: the experiment process's children: %v", kerr)
	}
	pidfds, ferr := pidfdsOf(pid)
	if ferr != nil {
		s.flag("observation failed: the experiment process's descriptors: %v", ferr)
	}
	pline("P2."+id, map[string]any{"report": r, "outer_status": outer, "children": kids, "pidfds": pidfds})
	release()
	c := childOf(cmd)
	if !c.reap(5 * time.Second) {
		s.invalidf("the experiment process did not exit within 5 s of its release")
	}
	code := cmd.ProcessState.ExitCode()

	denied := "not reached"
	for _, st := range r.Steps {
		if st.Call == w.denied {
			denied = st.Errno
			if denied == "" {
				denied = "succeeded"
			}
			break
		}
	}
	s.note("experiment pid %d, check pid %d, %d steps", pid, r.CheckPID, len(r.Steps))
	for _, st := range r.Steps {
		s.note("%s: %s %s", st.Call, map[bool]string{true: "ok", false: st.Errno}[st.Errno == ""], st.Value)
	}
	s.worked(int64(len(r.Steps)))
	p2Judge(s, group, r)
	s.expect("filter, its own read", w.status, r.Status)
	s.expect("filter, the outer's read", w.status, outer)
	if w.denied != "" {
		s.expect("the denied call's errno", w.deniedErrno, denied)
		// Every call before the denied one held: the filter hit that call.
		var before []string
		for _, st := range r.Steps {
			if st.Call == w.denied {
				break
			}
			if st.Errno != "" && !(st.Errno == "ECHILD" && strings.HasPrefix(st.Call, "waitid(P_PIDFD")) {
				before = append(before, st.Call+" "+st.Errno)
			}
		}
		s.expectTrue("the calls before the denied one held", len(before) == 0, strings.Join(before, ", "))
	}
	switch group {
	case "normal":
		s.expect("waitid(P_PIDFD) on its own pidfd", "ECHILD", p2Result(r, "waitid(P_PIDFD, own, WEXITED|WNOHANG)"))
	case "prctl.set1":
		s.expect("argument control: the GET under the same filter", "ok 0", p2Result(r, "prctl(PR_GET_CHILD_SUBREAPER) (probe)"))
	case "prctl.set0":
		s.expect("argument control: SET 1 under the same filter", "ok", p2Result(r, "prctl(PR_SET_CHILD_SUBREAPER, 1)"))
		s.expect("argument control: the GET after the set", "ok 1", p2Result(r, "prctl(PR_GET_CHILD_SUBREAPER) (after the set)"))
	}
	s.expect("refused capability", w.refused, r.Refused)
	s.expect("check started", w.check, r.CheckPID > 0)
	if w.check {
		s.expect("check status", "exit 0", r.CheckStatus)
		s.expect("check reaped", true, r.CheckReaped)
	}
	s.expect("subreaper at the end", w.subEnd, r.SubEnd)
	s.expect("clear error", w.clearErr, r.ClearErr)
	s.expect("the next probe", w.next, r.Next)
	s.expect("reported exit", w.exit, r.Exit)
	s.expect("exit status", w.exit, code)
	s.expect("children of the experiment process", 0, len(kids))
	s.expect("pidfds left in the experiment process", 0, pidfds)
	if group == "prctl.set0" {
		s.note("the attribute stayed set (subreaper %d) until the experiment process exited; that exit is not a clear by the inner code", r.SubEnd)
	}
}
