//go:build darwin

package wp25

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

func osVersion() string {
	v, _ := unix.Sysctl("kern.osproductversion")
	r, _ := unix.Sysctl("kern.osrelease")
	return "macOS " + v + " (Darwin " + r + ")"
}

const sZomb = 5 // SZOMB in <sys/proc.h>

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
// kern.proc.pgrp.
func groupLive(pgid int) ([]int, error) {
	ks, err := unix.SysctlKinfoProcSlice("kern.proc.pgrp", pgid)
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

// groupAll lists every entry kern.proc.pgrp gives, zombies included, as
// "pid:state".
func groupAll(pgid int) []string {
	ks, err := unix.SysctlKinfoProcSlice("kern.proc.pgrp", pgid)
	if err != nil {
		return []string{"error " + err.Error()}
	}
	var out []string
	for _, k := range ks {
		out = append(out, fmt.Sprintf("%d:%d", k.Proc.P_pid, k.Proc.P_stat))
	}
	return out
}

// exitObserver is the macOS candidate for E-main: a kqueue EVFILT_PROC
// NOTE_EXIT event, waited for with a timeout, which does not reap. A
// process that exited before the registration is seen as a zombie.
type exitObserver struct{ s *obsState }

type obsState struct {
	pid, kq int
	early   bool
	exited  bool
	status  string
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
		if st, pr := procInfo(pid); errors.Is(err, unix.ESRCH) && pr == present && st.zombie {
			s.early = true
		} else {
			_ = unix.Close(kq)
			return exitObserver{}, fmt.Errorf("kevent register: %w", err)
		}
	}
	return exitObserver{s: s}, nil
}

func (o exitObserver) close() { _ = unix.Close(o.s.kq) }

func (o exitObserver) wait(d time.Duration) (bool, string, error) {
	s := o.s
	if s.exited {
		return true, s.status, nil
	}
	if s.early {
		st, err := waitidNoWait(s.pid)
		if err != nil {
			st = "unknown (exited before the registration)"
		}
		s.exited, s.status = true, st
		return true, st, nil
	}
	deadline := time.Now().Add(d)
	evs := make([]unix.Kevent_t, 1)
	for {
		left := time.Until(deadline)
		if left < 0 {
			left = 0
		}
		ts := unix.NsecToTimespec(left.Nanoseconds())
		n, err := unix.Kevent(s.kq, nil, evs, &ts)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return false, "", err
		}
		if n == 0 {
			return false, "", nil
		}
		if evs[0].Fflags&unix.NOTE_EXIT != 0 {
			s.exited, s.status = true, waitStatus(syscall.WaitStatus(evs[0].Data))
			return true, s.status, nil
		}
	}
}

// waitidNoWait calls waitid(P_PID, pid, WEXITED|WNOWAIT) through the raw
// system call; x/sys has no darwin wrapper. Darwin's siginfo_t has
// si_code at 8 and si_status at 20.
func waitidNoWait(pid int) (string, error) {
	var si [128]byte
	const pPID = 1
	_, _, e := unix.Syscall6(unix.SYS_WAITID, pPID, uintptr(pid), uintptr(unsafe.Pointer(&si[0])),
		unix.WEXITED|unix.WNOWAIT, 0, 0)
	if e != 0 {
		return "", e
	}
	code := int32(binary.LittleEndian.Uint32(si[8:12]))
	status := int32(binary.LittleEndian.Uint32(si[20:24]))
	switch code {
	case 1:
		return "exit " + strconv.Itoa(int(status)), nil
	case 2, 3:
		return "signal " + strconv.Itoa(int(status)), nil
	}
	return "code " + strconv.Itoa(int(code)), nil
}

// sigchldDisposition is not read on macOS (no /proc); the probe relies on
// signal.Ignored and the observed reaping.
func sigchldDisposition() string { return "n/a" }
