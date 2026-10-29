//go:build darwin

package wp25

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

func osVersion() string {
	v, _ := unix.Sysctl("kern.osproductversion")
	r, _ := unix.Sysctl("kern.osrelease")
	return "macOS " + v + " (Darwin " + r + ")"
}

const (
	sZomb  = 5      // SZOMB in <sys/proc.h>
	pWexit = 0x2000 // P_WEXIT in <sys/proc.h>: working on exiting
)

// procInfo reads a process's state, parent, group and start time with
// sysctl kern.proc.pid; zombies are included. An empty answer is absent; a
// failed call or an answer of another size is unknown.
func procInfo(pid int) (pinfo, presence) {
	b, err := unix.SysctlRaw("kern.proc.pid", pid)
	if err != nil {
		return pinfo{}, unknown
	}
	if len(b) == 0 {
		return pinfo{}, absent
	}
	if len(b) != int(unsafe.Sizeof(unix.KinfoProc{})) {
		return pinfo{}, unknown
	}
	k := (*unix.KinfoProc)(unsafe.Pointer(&b[0]))
	if int(k.Proc.P_pid) != pid {
		return pinfo{}, unknown
	}
	return pinfo{
		zombie: k.Proc.P_stat == sZomb,
		ppid:   int(k.Eproc.Ppid),
		pgid:   int(k.Eproc.Pgid),
		start:  fmt.Sprintf("%d.%06d", k.Proc.P_starttime.Sec, k.Proc.P_starttime.Usec),
	}, present
}

// groupLive lists the group's members that are not zombies, with sysctl
// kern.proc.pgrp, within the caller's deadline (kinfoProcs).
func groupLive(pgid int, deadline time.Time) ([]int, error) {
	ks, err := kinfoProcs(deadline, "kern.proc.pgrp", pgid)
	if err != nil {
		return nil, err
	}
	var out []int
	for _, k := range ks {
		if k.Proc.P_stat != sZomb {
			out = append(out, int(k.Proc.P_pid))
		}
	}
	return out, nil
}

// The system call of a kinfo_proc list, which a direct check
// (enum_check_darwin_test.go) replaces; the real one otherwise.
var sysctlRaw = unix.SysctlRaw

// kinfoProcs reads a kinfo_proc list (sysctl kern.proc.*). It is x/sys's
// SysctlKinfoProcSlice, its loop unchanged, through the seams: ENOMEM, the
// table having grown between the size and the read, is tried again.
func kinfoProcs(_ time.Time, name string, args ...int) ([]unix.KinfoProc, error) {
	for {
		b, err := enumCall(name, args)
		if errors.Is(err, unix.ENOMEM) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return kinfoOf(b)
	}
}

// enumCall is one call: a gate regression's construction answers it
// instead (p3Seam.enum, the call is then not made), or the real one.
func enumCall(name string, args []int) ([]byte, error) {
	if h := p3Seam.enum; h != nil {
		if err := h(name); err != nil {
			return nil, err
		}
	}
	return sysctlRaw(name, args...)
}

// kinfoOf makes the list of what the call returned: whole entries only.
func kinfoOf(b []byte) ([]unix.KinfoProc, error) {
	if len(b)%unix.SizeofKinfoProc != 0 {
		return nil, fmt.Errorf("sysctl returned %d bytes, not a multiple of %d", len(b), unix.SizeofKinfoProc)
	}
	ks := make([]unix.KinfoProc, len(b)/unix.SizeofKinfoProc)
	if len(ks) > 0 {
		copy(unsafe.Slice((*byte)(unsafe.Pointer(&ks[0])), len(b)), b)
	}
	return ks, nil
}

// exitObserver is the macOS candidate for E-main: a kqueue EVFILT_PROC
// NOTE_EXIT event, waited for with a timeout, which does not reap. A
// process that exited before the registration is seen as a zombie. What
// the kernel returned is kept before it is decoded (P-3, spec §19.12).
type exitObserver struct{ s *obsState }

type obsState struct {
	pid, kq int
	early   bool
	exited  bool
	status  string
	regErr  error     // the registration's own error; nil if it registered
	lastK   *kRaw     // the event that came, before decoding
	lastW   *wRaw     // the waitid read of an early exit, before decoding
	end     *exitInfo // what the decoder made of it
}

func newExitObserver(pid int) (exitObserver, error) {
	kq, err := unix.Kqueue()
	if err != nil {
		return exitObserver{}, err
	}
	s := &obsState{pid: pid, kq: kq}
	ev := unix.Kevent_t{Ident: uint64(pid), Filter: unix.EVFILT_PROC,
		Flags: unix.EV_ADD | unix.EV_ONESHOT, Fflags: unix.NOTE_EXIT | unix.NOTE_EXITSTATUS}
	if _, err := unix.Kevent(kq, []unix.Kevent_t{ev}, nil, nil); err != nil {
		s.regErr = err
		if st, pr := procInfo(pid); errors.Is(err, unix.ESRCH) && pr == present && st.zombie {
			s.early = true
		} else {
			_ = unix.Close(kq)
			return exitObserver{}, fmt.Errorf("kevent register: %w", err)
		}
	}
	return exitObserver{s: s}, nil
}

func (o exitObserver) close() {
	err := unix.Close(o.s.kq)
	if h := p3Seam.close; h != nil {
		err = h(err)
	}
	_ = err
}

// wait observes the exit within d. An event, or an early exit's status,
// that the decoder rejects is an observation failure (spec §19.5): the
// exit is not reliably observed.
func (o exitObserver) wait(d time.Duration) (bool, string, error) {
	s := o.s
	if s.exited {
		return true, s.status, nil
	}
	if s.early {
		// It exited before the registration: its status is read without
		// reaping it. A read that fails is an observation failure, not a
		// status (Review 077 B1-R1-S).
		r, err := readWaitid(s.pid)
		if err != nil {
			return true, "", err
		}
		s.lastW = &r
		e, err := decodeW(r, s.pid)
		if err != nil {
			return true, "", err
		}
		s.end = &e
		s.exited, s.status = true, e.String()
		return true, s.status, nil
	}
	ev, ok, err := kqWaitRaw(s.kq, d)
	if err != nil || !ok {
		return false, "", err
	}
	s.lastK = &ev
	e, err := decodeK(ev, s.pid)
	if err != nil {
		return false, "", err
	}
	s.end = &e
	s.exited, s.status = true, e.String()
	return true, s.status, nil
}

// The decoding of an exit status on macOS (spec v8 second draft §19.5):
// what the kernel returned is kept whole (R), and the decoder (M) accepts
// a normalized end or rejects it by the first rule that does not hold.

const (
	pPID      = 1 // P_PID in <sys/wait.h>
	cldExited = 1 // CLD_EXITED in <sys/signal.h>
	cldKilled = 2
	cldDumped = 3
	siSize    = 128 // the waitid buffer; user64_siginfo is 104 bytes
)

// wOff are the offsets of si_signo, si_errno, si_code, si_pid, si_uid and
// si_status that the decoder reads, 4 bytes each.
var wOff = [6]int{0, 4, 8, 12, 16, 20}

// wRaw is one waitid read: the first six fields of siginfo_t as the kernel
// wrote them, each read as an unsigned 32-bit value (W2).
type wRaw struct {
	Signo  uint32 `json:"si_signo"`
	Errno  uint32 `json:"si_errno"`
	Code   uint32 `json:"si_code"`
	Pid    uint32 `json:"si_pid"`
	Uid    uint32 `json:"si_uid"`
	Status uint32 `json:"si_status"`
}

// kRaw is one kqueue event before any decoding; data is kept signed and
// whole (K1).
type kRaw struct {
	Ident  uint64 `json:"ident"`
	Filter int16  `json:"filter"`
	Flags  uint16 `json:"flags"`
	Fflags uint32 `json:"fflags"`
	Data   int64  `json:"data"`
}

// exitInfo is a normalized end: its kind ("exit", "signal" or "core", a
// signal with WCOREFLAG) and its value.
type exitInfo struct {
	Kind string `json:"kind"`
	Val  int    `json:"val"`
}

func (e exitInfo) String() string {
	switch e.Kind {
	case "exit":
		return "exit " + strconv.Itoa(e.Val)
	case "signal":
		return "signal " + strconv.Itoa(e.Val)
	}
	return "signal " + strconv.Itoa(e.Val) + " core"
}

// ruleErr is the decoder's rejection: the rule that does not hold.
type ruleErr struct{ rule, detail string }

func (e *ruleErr) Error() string { return e.rule + ": " + e.detail }

func reject(rule, f string, a ...any) error { return &ruleErr{rule, fmt.Sprintf(f, a...)} }

// ruleOf names the rule an error rejects by; "" for any other error.
func ruleOf(err error) string {
	var r *ruleErr
	if errors.As(err, &r) {
		return r.rule
	}
	return ""
}

// p3Seam rewrites what a read returned, after the real call: P-3's gate
// regressions only (p3_darwin_test.go), nil otherwise. n counts the waitid
// reads while w is set, from 1. enum answers a call of a kinfo_proc list
// in its place (a non-nil error: the call is not made); close replaces
// the result of the observer's real close.
var p3Seam struct {
	w     func(n int, si *[siSize]byte)
	k     func(ev *unix.Kevent_t)
	n     int
	enum  func(name string) error
	close func(err error) error
}

// The kqueue wait's system call, which a direct check
// (enum_check_darwin_test.go) replaces; the real one otherwise.
var keventCall = unix.Kevent

// readWaitid calls waitid(P_PID, pid, WEXITED|WNOWAIT) through the raw
// system call (E1; x/sys has no darwin wrapper) and returns its fields
// before any decoding.
func readWaitid(pid int) (wRaw, error) {
	var si [siSize]byte
	_, _, e := unix.Syscall6(unix.SYS_WAITID, pPID, uintptr(pid), uintptr(unsafe.Pointer(&si[0])),
		unix.WEXITED|unix.WNOWAIT, 0, 0)
	if e != 0 {
		return wRaw{}, e
	}
	if p3Seam.w != nil {
		p3Seam.n++
		p3Seam.w(p3Seam.n, &si)
	}
	f := func(i int) uint32 { return binary.LittleEndian.Uint32(si[wOff[i]:]) }
	return wRaw{f(0), f(1), f(2), f(3), f(4), f(5)}, nil
}

// decodeW decodes one waitid read of pid: W1, then W3a for an exit (whose
// value is the low 8 bits of si_status, W3b, Q-3 (a)) or W4 for a signal.
func decodeW(r wRaw, pid int) (exitInfo, error) {
	switch {
	case r.Signo != uint32(unix.SIGCHLD):
		return exitInfo{}, reject("W1", "si_signo %d is not SIGCHLD", r.Signo)
	case r.Pid != uint32(pid):
		return exitInfo{}, reject("W1", "si_pid %d is not %d", r.Pid, pid)
	}
	switch r.Code {
	case cldExited:
		if r.Status&0xff000000 != 0 {
			return exitInfo{}, reject("W3a", "si_status %#x has bits 24 to 31 set", r.Status)
		}
		return exitInfo{"exit", int(r.Status & 0xff)}, nil
	case cldKilled, cldDumped:
		if r.Status < 1 || r.Status > 31 {
			return exitInfo{}, reject("W4", "si_status %#x is not a signal from 1 to 31", r.Status)
		}
		if r.Code == cldDumped {
			return exitInfo{"core", int(r.Status)}, nil
		}
		return exitInfo{"signal", int(r.Status)}, nil
	}
	return exitInfo{}, reject("W1", "si_code %d is not CLD_EXITED, CLD_KILLED or CLD_DUMPED", r.Code)
}

// decodeW2 decodes two waitid reads of the same unreaped process: the
// first decides, and the second must be the same in every field (W5).
func decodeW2(a, b wRaw, pid int) (exitInfo, error) {
	e, err := decodeW(a, pid)
	if err != nil {
		return exitInfo{}, err
	}
	if a != b {
		return exitInfo{}, reject("W5", "the second read %+v differs from the first %+v", b, a)
	}
	return e, nil
}

// kqWaitRaw waits at most d for one event on kq, retrying EINTR within
// what is left of d; ok is false when none came.
func kqWaitRaw(kq int, d time.Duration) (kRaw, bool, error) {
	deadline := time.Now().Add(d)
	evs := make([]unix.Kevent_t, 1)
	for {
		left := time.Until(deadline)
		if left < 0 {
			left = 0
		}
		ts := unix.NsecToTimespec(left.Nanoseconds())
		n, err := keventCall(kq, nil, evs, &ts)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return kRaw{}, false, err
		}
		if n == 0 {
			return kRaw{}, false, nil
		}
		if p3Seam.k != nil {
			p3Seam.k(&evs[0])
		}
		e := evs[0]
		return kRaw{e.Ident, e.Filter, e.Flags, e.Fflags, e.Data}, true, nil
	}
}

// decodeK decodes one kqueue event of pid. An event of another process or
// filter is an error of the call, not a K rule; then K1 and K2.
func decodeK(r kRaw, pid int) (exitInfo, error) {
	if r.Ident != uint64(pid) || r.Filter != unix.EVFILT_PROC {
		return exitInfo{}, fmt.Errorf("kevent returned an event of ident %d, filter %d, not of process %d", r.Ident, r.Filter, pid)
	}
	const both = unix.NOTE_EXIT | unix.NOTE_EXITSTATUS
	switch {
	case r.Fflags&both != both:
		return exitInfo{}, reject("K1", "the returned fflags %#x lack NOTE_EXIT or NOTE_EXITSTATUS", r.Fflags)
	case r.Data < 0 || r.Data > 0xffff:
		return exitInfo{}, reject("K1", "data %d is outside 0 to 0xffff", r.Data)
	}
	ws := uint32(r.Data)
	sig := ws & 0x7f
	switch {
	case ws&0xff == 0:
		return exitInfo{"exit", int(ws >> 8)}, nil
	case sig >= 1 && sig <= 31 && ws>>8 == 0:
		if ws&0x80 != 0 {
			return exitInfo{"core", int(sig)}, nil
		}
		return exitInfo{"signal", int(sig)}, nil
	}
	return exitInfo{}, reject("K2", "data %#x is neither an exit nor a signal (stopped, or bits 8 to 15 beside a signal)", ws)
}

// sameEnd compares the two paths' normalized ends (C1).
func sameEnd(k, w exitInfo) error {
	if k != w {
		return reject("C1", "the kqueue end %s is not the waitid end %s", k, w)
	}
	return nil
}

// waitidNoWait takes a child's exit status without reaping it, decoded.
func waitidNoWait(pid int) (string, error) {
	r, err := readWaitid(pid)
	if err != nil {
		return "", err
	}
	e, err := decodeW(r, pid)
	if err != nil {
		return "", err
	}
	return e.String(), nil
}

// sigchldDisposition is not read on macOS (no /proc); the probe relies on
// signal.Ignored and the observed reaping.
func sigchldDisposition() string { return "n/a" }

// ownChildren and newChildren: macOS has no simple list of a process's
// children here; the leftover check is Linux's (see tree_linux_test.go).
func ownChildren() ([]int, error) { return nil, nil }
func newChildren([]int) string    { return "" }
