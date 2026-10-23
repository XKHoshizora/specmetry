//go:build windows

package wp25

import (
	"errors"
	"strconv"

	"golang.org/x/sys/windows"
)

func creationTime(h windows.Handle) (string, error) {
	var c, e, k, u windows.Filetime
	if err := windows.GetProcessTimes(h, &c, &e, &k, &u); err != nil {
		return "", err
	}
	return strconv.FormatUint(uint64(c.HighDateTime)<<32|uint64(c.LowDateTime), 10), nil
}

// selfStart is this process's creation time.
func selfStart() string {
	h, err := windows.GetCurrentProcess()
	if err != nil {
		return "unknown"
	}
	s, err := creationTime(h)
	if err != nil {
		return "unknown"
	}
	return s
}

// openWitnessed opens a witnessed helper by its PID and checks that the
// process found there is the one witnessed (same creation time).
func openWitnessed(w witnessLine, access uint32) (windows.Handle, bool, error) {
	h, err := windows.OpenProcess(access|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(w.pid))
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return 0, false, nil // no such process
		}
		return 0, false, err
	}
	s, err := creationTime(h)
	if err != nil {
		_ = windows.CloseHandle(h)
		return 0, false, err
	}
	if s != w.start {
		_ = windows.CloseHandle(h)
		return 0, false, nil // the PID names another process now
	}
	return h, true, nil
}

func witnessAlive(w witnessLine) (bool, error) {
	h, same, err := openWitnessed(w, windows.SYNCHRONIZE)
	if err != nil || !same {
		return false, err
	}
	defer windows.CloseHandle(h)
	ev, err := windows.WaitForSingleObject(h, 0)
	if err != nil {
		return false, err
	}
	return ev != windows.WAIT_OBJECT_0, nil
}

func witnessStop(w witnessLine) {
	h, same, err := openWitnessed(w, windows.PROCESS_TERMINATE)
	if err != nil || !same {
		return
	}
	_ = windows.TerminateProcess(h, 1)
	_ = windows.CloseHandle(h)
}
