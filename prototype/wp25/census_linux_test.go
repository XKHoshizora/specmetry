//go:build linux

package wp25

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// censusTag lists the live processes of this user, started no earlier
// than since (the inner run's start ticks), whose environment carries tag.
// A process that ends during the scan is left out; one whose state or
// environment cannot be read is an error, never a process left out.
func censusTag(tag, since string) ([]survivor, error) {
	floor, err := strconv.ParseUint(since, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: the inner run's start %q", errCensus, since)
	}
	pids, err := allPIDs()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errCensus, err)
	}
	var out []survivor
	for _, pid := range pids {
		if pid == os.Getpid() {
			continue
		}
		v, ok, err := censusOne(pid, tag, floor)
		if err != nil {
			return nil, fmt.Errorf("%w: process %d: %v", errCensus, pid, err)
		}
		if ok {
			out = append(out, v)
		}
	}
	return out, nil
}

// censusOne reads one process: its state, its owner and its environment,
// then its state again; if the PID changed hands meanwhile, it is read
// again.
func censusOne(pid int, tag string, floor uint64) (survivor, bool, error) {
	for try := 0; try < 3; try++ {
		st, pr := procInfo(pid)
		switch {
		case pr == unknown:
			return survivor{}, false, errUnknownState
		case pr == absent, st.zombie:
			return survivor{}, false, nil
		}
		n, err := strconv.ParseUint(st.start, 10, 64)
		if err != nil {
			return survivor{}, false, errUnknownState
		}
		if n < floor {
			return survivor{}, false, nil
		}
		fi, err := os.Stat("/proc/" + strconv.Itoa(pid))
		if gone(err) {
			return survivor{}, false, nil
		}
		if err != nil {
			return survivor{}, false, err
		}
		if sys, ok := fi.Sys().(*syscall.Stat_t); !ok || sys.Uid != uint32(os.Getuid()) {
			return survivor{}, false, nil
		}
		env, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/environ")
		if gone(err) {
			return survivor{}, false, nil
		}
		if err != nil {
			return survivor{}, false, err
		}
		st2, pr2 := procInfo(pid)
		switch {
		case pr2 == unknown:
			return survivor{}, false, errUnknownState
		case pr2 == absent:
			return survivor{}, false, nil
		case st2.start != st.start:
			continue
		}
		return survivor{pid, st.start}, hasToken(env, tag), nil
	}
	return survivor{}, false, errors.New("the PID kept changing hands")
}

func gone(err error) bool { return errors.Is(err, fs.ErrNotExist) || errors.Is(err, unix.ESRCH) }

// environOf reads a process's environment, NUL-separated.
func environOf(pid int) ([]byte, error) {
	return os.ReadFile("/proc/" + strconv.Itoa(pid) + "/environ")
}

// tagged reports whether the process's environment still carries tag.
func tagged(pid int, tag string) bool {
	env, err := environOf(pid)
	return err == nil && hasToken(env, tag)
}
