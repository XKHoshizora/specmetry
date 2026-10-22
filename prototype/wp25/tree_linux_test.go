//go:build linux

package wp25

import (
	"encoding/binary"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

func osVersion() string {
	var u unix.Utsname
	if err := unix.Uname(&u); err != nil {
		return "linux"
	}
	return "linux " + unix.ByteSliceToString(u.Release[:])
}

// procInfo reads /proc/<pid>/stat: state, parent, group and start time.
func procInfo(pid int) (pinfo, bool) {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return pinfo{}, false
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return pinfo{}, false
	}
	f := strings.Fields(s[i+1:])
	if len(f) < 20 {
		return pinfo{}, false
	}
	ppid, _ := strconv.Atoi(f[1])
	pgid, _ := strconv.Atoi(f[2])
	return pinfo{zombie: f[0] == "Z" || f[0] == "X", ppid: ppid, pgid: pgid, start: f[19]}, true
}

func allPIDs() []int {
	if injected("readdir-fail") {
		return nil // ReadDir failed; its error is dropped, as below
	}
	ents, _ := os.ReadDir("/proc")
	var out []int
	for _, e := range ents {
		if pid, err := strconv.Atoi(e.Name()); err == nil {
			out = append(out, pid)
		}
	}
	return out
}

func groupLive(pgid int) ([]int, error) {
	var out []int
	for _, pid := range allPIDs() {
		if st, ok := procInfo(pid); ok && st.pgid == pgid && !st.zombie {
			out = append(out, pid)
		}
	}
	return out, nil
}

// childrenOf lists the children of pid, zombies included.
func childrenOf(pid int) []int {
	var out []int
	for _, p := range allPIDs() {
		if st, ok := procInfo(p); ok && st.ppid == pid {
			out = append(out, p)
		}
	}
	return out
}

// exitObserver is the Linux E-main: a bounded poll on the main process's
// pidfd, then waitid(P_PID, WEXITED|WNOWAIT) for the status, which leaves
// the process unreaped and no goroutine waiting.
type exitObserver struct {
	pid, fd int
}

func newExitObserver(pid int) (exitObserver, error) {
	fd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		return exitObserver{}, err
	}
	return exitObserver{pid: pid, fd: fd}, nil
}

func (o exitObserver) close() { _ = unix.Close(o.fd) }

func (o exitObserver) wait(d time.Duration) (bool, string, error) {
	deadline := time.Now().Add(d)
	for {
		n, err := unix.Poll([]unix.PollFd{{Fd: int32(o.fd), Events: unix.POLLIN}}, pollMillis(deadline))
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return false, "", err
		}
		if n == 0 {
			return false, "", nil
		}
		st, err := waitidNoWait(o.pid)
		return err == nil, st, err
	}
}

// pollMillis is the time left until deadline in whole milliseconds,
// rounded up, so that a bounded wait never ends before its limit.
func pollMillis(deadline time.Time) int {
	left := time.Until(deadline)
	if left <= 0 {
		return 0
	}
	return int((left + time.Millisecond - 1) / time.Millisecond)
}

// waitidNoWait takes a child's exit status without reaping it.
func waitidNoWait(pid int) (string, error) {
	var info unix.Siginfo
	if err := unix.Waitid(unix.P_PID, pid, &info, unix.WEXITED|unix.WNOWAIT, nil); err != nil {
		return "", err
	}
	return siginfoStatus(&info), nil
}

// siginfoStatus reads si_code and si_status of a SIGCHLD siginfo (the
// union starts with si_pid, si_uid, si_status after the 16-byte header).
func siginfoStatus(info *unix.Siginfo) string {
	raw := (*[128]byte)(unsafe.Pointer(info))[:]
	code := int32(binary.LittleEndian.Uint32(raw[8:12]))
	status := int32(binary.LittleEndian.Uint32(raw[24:28]))
	switch code {
	case 1: // CLD_EXITED
		return "exit " + strconv.Itoa(int(status))
	case 2, 3: // CLD_KILLED, CLD_DUMPED
		return "signal " + strconv.Itoa(int(status))
	}
	return "code " + strconv.Itoa(int(code))
}

// sigchldDisposition reads whether the kernel has SIGCHLD ignored for this
// process (SigIgn in /proc/self/status).
func sigchldDisposition() string {
	b, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return "unknown"
	}
	for _, l := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(l, "SigIgn:"); ok {
			mask, err := strconv.ParseUint(strings.TrimSpace(v), 16, 64)
			if err != nil {
				return "unknown"
			}
			if mask&(1<<(uint(unix.SIGCHLD)-1)) != 0 {
				return "SIG_IGN"
			}
			return "not-ignored"
		}
	}
	return "unknown"
}
