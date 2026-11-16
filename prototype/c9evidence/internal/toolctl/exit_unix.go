//go:build unix

package toolctl

import (
	"os"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// exitOf copies a ProcessState with its end's identity: on Unix a signal
// end has the code -1 whatever the signal (os/exec_posix.go), so the
// signal's name is kept (Review 155 B-154-4-R-S).
func exitOf(ps *os.ProcessState) Exit {
	e := Exit{Code: ps.ExitCode(), Exited: ps.Exited(), String: ps.String()}
	if ws, ok := ps.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		e.Signaled = true
		if e.Signal = unix.SignalName(ws.Signal()); e.Signal == "" {
			e.Signal = "signal-" + strconv.Itoa(int(ws.Signal()))
		}
	}
	return e
}
