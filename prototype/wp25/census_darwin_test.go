//go:build darwin

package wp25

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// censusTag lists the live processes of this user, started no earlier
// than since (the inner run's start time), whose arguments and environment
// (sysctl kern.procargs2) carry tag. A process that ends during the scan
// is left out; one whose environment cannot be read is an error.
func censusTag(tag, since string) ([]survivor, error) {
	ks, err := unix.SysctlKinfoProcSlice("kern.proc.all")
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
		b, err := unix.SysctlRaw("kern.procargs2", pid)
		if err != nil || len(b) < 4 {
			// Ended meanwhile, or a zombie now: left out; otherwise an error.
			if s2, pr := procInfo(pid); pr == absent || pr == present && (s2.zombie || s2.start != st) {
				continue
			}
			return nil, fmt.Errorf("%w: process %d: the environment cannot be read: %v", errCensus, pid, err)
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
