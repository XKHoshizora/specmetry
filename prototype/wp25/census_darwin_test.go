//go:build darwin

package wp25

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// censusTag lists the live processes of this user, started no earlier
// than since (the inner run's start time), whose arguments and environment
// (sysctl kern.procargs2) carry tag. A process that ends during the scan
// is left out; one whose environment cannot be read is an error.
func censusTag(tag, since string) ([]survivor, error) {
	ks, err := procList()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errCensus, err)
	}
	var out []survivor
	for _, k := range ks {
		pid := int(k.Proc.P_pid)
		if pid <= 0 || pid == os.Getpid() || k.Proc.P_stat == sZomb || k.Eproc.Ucred.Uid != uint32(os.Getuid()) {
			continue
		}
		st := fmt.Sprintf("%d.%06d", k.Proc.P_starttime.Sec, k.Proc.P_starttime.Usec)
		if !notBefore(st, since) {
			continue
		}
		b, err := procArgs(pid)
		if err != nil || len(b) < 4 {
			// Ended meanwhile, or a zombie now: left out; otherwise an error,
			// which says what is known of the process (Review 083 N-MAC).
			s2, pr := censusRecheck(pid)
			if pr == absent || pr == present && (s2.zombie || s2.start != st) {
				continue
			}
			if err == nil {
				err = fmt.Errorf("%d bytes", len(b))
			}
			return nil, fmt.Errorf("%w: process %d: the environment cannot be read: %w (%s)", errCensus, pid, err, unreadableDetail(k, st, s2, pr))
		}
		if s2, pr := procInfo(pid); pr != present || s2.start != st || s2.zombie {
			if pr == unknown {
				return nil, fmt.Errorf("%w: process %d: %v", errCensus, pid, errUnknownState)
			}
			continue // ended meanwhile
		}
		if hasToken(b[4:], tag) {
			out = append(out, survivor{pid, st})
		}
	}
	return out, nil
}

// unreadableDetail says what is known of a process whose environment the
// census could not read (Review 083 N-MAC): from the list entry the census
// read, its identity, state, whether it is exiting, its parent and group,
// and whether it runs this test's program; and the state read after the
// failure. Never its arguments, environment or name. It is evidence for a
// reader, not of ownership: the census fails all the same.
func unreadableDetail(k unix.KinfoProc, st string, s2 pinfo, pr presence) string {
	const pWexit = 0x2000 // P_WEXIT in <sys/proc.h>: working on exiting
	after := "unreadable"
	switch pr {
	case absent:
		after = "gone"
	case present:
		after = fmt.Sprintf("present, start %s, zombie %v", s2.start, s2.zombie)
	}
	return fmt.Sprintf("start %s, state %d, exiting %v, parent %d, group %d, same program as this test: %s; state after the failure: %s",
		st, k.Proc.P_stat, k.Proc.P_flag&pWexit != 0, k.Eproc.Ppid, k.Eproc.Pgid, sameProgram(k), after)
}

// sameProgram compares a process's short command name (at most
// MAXCOMLEN bytes) with this test binary's; only the answer is kept.
func sameProgram(k unix.KinfoProc) string {
	self, err := os.Executable()
	comm := unix.ByteSliceToString(k.Proc.P_comm[:])
	switch {
	case err != nil || comm == "":
		return "unknown"
	case strings.HasPrefix(filepath.Base(self), comm):
		return "yes"
	}
	return "no"
}

// The census's reads, which a direct check (census_check_darwin_test.go)
// replaces; the real ones otherwise.
var (
	procList      = func() ([]unix.KinfoProc, error) { return unix.SysctlKinfoProcSlice("kern.proc.all") }
	procArgs      = func(pid int) ([]byte, error) { return unix.SysctlRaw("kern.procargs2", pid) }
	censusRecheck = procInfo
)

// notBefore compares two "sec.usec" times.
func notBefore(a, b string) bool {
	as, au, _ := strings.Cut(a, ".")
	bs, bu, _ := strings.Cut(b, ".")
	x, _ := strconv.ParseUint(as, 10, 64)
	y, _ := strconv.ParseUint(bs, 10, 64)
	if x != y {
		return x > y
	}
	return au >= bu
}

// environOf reads a process's arguments and environment (sysctl
// kern.procargs2, after its argument count), NUL-separated.
func environOf(pid int) ([]byte, error) {
	b, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return nil, err
	}
	if len(b) < 4 {
		return nil, fmt.Errorf("kern.procargs2 of %d: %d bytes", pid, len(b))
	}
	return b[4:], nil
}

func tagged(pid int, tag string) bool {
	env, err := environOf(pid)
	return err == nil && hasToken(env, tag)
}
