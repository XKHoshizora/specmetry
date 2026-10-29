//go:build linux

package wp25

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
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
// Only "no such file" or "no such process" is absent; any other failure,
// or a line that does not parse, is unknown.
func procInfo(pid int) (pinfo, presence) {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, unix.ESRCH) {
			return pinfo{}, absent
		}
		return pinfo{}, unknown
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return pinfo{}, unknown
	}
	f := strings.Fields(s[i+1:])
	if len(f) < 20 {
		return pinfo{}, unknown
	}
	ppid, err1 := strconv.Atoi(f[1])
	pgid, err2 := strconv.Atoi(f[2])
	if err1 != nil || err2 != nil {
		return pinfo{}, unknown
	}
	return pinfo{zombie: f[0] == "Z" || f[0] == "X", ppid: ppid, pgid: pgid, start: f[19]}, present
}

// allPIDs lists /proc; a failed read is an error, never an empty list.
func allPIDs() ([]int, error) {
	if injected("readdir-fail") {
		return nil, errors.New("injected: reading /proc failed")
	}
	if injectOnce("readdir-fail-once", "allPIDs", os.Getpid()) {
		return nil, errors.New("injected: reading /proc failed once")
	}
	ents, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var out []int
	for _, e := range ents {
		if pid, err := strconv.Atoi(e.Name()); err == nil {
			out = append(out, pid)
		}
	}
	return out, nil
}

// processes lists every process whose state was read; a process that
// ended during the scan is left out, one that cannot be read is an error.
func processes() (map[int]pinfo, error) {
	pids, err := allPIDs()
	if err != nil {
		return nil, err
	}
	out := map[int]pinfo{}
	for _, pid := range pids {
		st, pr := procInfo(pid)
		switch pr {
		case present:
			out[pid] = st
		case unknown:
			return nil, fmt.Errorf("process %d: %w", pid, errUnknownState)
		}
	}
	return out, nil
}

// groupLive lists the group's members that are not zombies. The deadline
// is macOS's (its enumeration's retries); processes reads /proc once.
func groupLive(pgid int, _ time.Time) ([]int, error) {
	ps, err := processes()
	if err != nil {
		return nil, err
	}
	var out []int
	for pid, st := range ps {
		if st.pgid == pgid && !st.zombie {
			out = append(out, pid)
		}
	}
	return out, nil
}

// childrenOf lists the children of pid, zombies included.
func childrenOf(pid int) ([]int, error) {
	ps, err := processes()
	if err != nil {
		return nil, err
	}
	var out []int
	for p, st := range ps {
		if st.ppid == pid {
			out = append(out, p)
		}
	}
	return out, nil
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

// siginfoPID reads si_pid of a SIGCHLD siginfo: 0 when a WNOHANG wait
// found nothing to reap.
func siginfoPID(info *unix.Siginfo) int {
	raw := (*[128]byte)(unsafe.Pointer(info))[:]
	return int(int32(binary.LittleEndian.Uint32(raw[16:20])))
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

// ownChildren lists this process's children, zombies included. A list
// that cannot be read is an error, never an empty list (Review 077
// B1-R1-S).
func ownChildren() ([]int, error) { return childrenOf(os.Getpid()) }

// newChildren describes the children of this process that were not there
// at the start (base): a scenario must not leave any, alive or a zombie
// it did not reap. "" when there are none.
func newChildren(base []int) string {
	kids, err := childrenOf(os.Getpid())
	if err != nil {
		return "unreadable: " + err.Error()
	}
	before := map[int]bool{}
	for _, k := range base {
		before[k] = true
	}
	var out []string
	for _, k := range kids {
		if before[k] {
			continue
		}
		st, pr := procInfo(k)
		if pr != present {
			out = append(out, fmt.Sprintf("%d state unreadable", k))
			continue
		}
		out = append(out, fmt.Sprintf("%d zombie=%v", k, st.zombie))
	}
	return strings.Join(out, ", ")
}
