//go:build linux || darwin

package wp25

import "syscall"

// selfStart is this process's start time as procInfo reads it.
func selfStart() string {
	st, pr := procInfo(syscall.Getpid())
	if pr != present {
		return "unknown"
	}
	return st.start
}

// witnessAlive reports whether a witnessed helper still runs: the same PID
// with the same start time, not a zombie.
func witnessAlive(w witnessLine) (bool, error) {
	st, pr := procInfo(w.pid)
	switch {
	case pr == unknown:
		return false, errUnknownState
	case pr == absent, st.start != w.start, st.zombie:
		return false, nil
	}
	return true, nil
}

// witnessStop ends a witnessed helper that is still alive, while its
// identity holds: an experiment object this round created.
func witnessStop(w witnessLine) {
	if a, err := witnessAlive(w); err == nil && a {
		_ = syscall.Kill(w.pid, syscall.SIGKILL)
	}
}
