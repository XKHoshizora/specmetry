//go:build linux

package toolctl

import (
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// platformMode on Linux: one request through the pidfd the start task
// opened after Start and before Wait (§12.14 items 8 and 9). Process.Kill
// is not used.
const platformMode = ModePidfd

func pidfdOpen(pid int) (int, error) { return unix.PidfdOpen(pid, 0) }

func pidfdSendKill(fd int) error { return unix.PidfdSendSignal(fd, unix.SIGKILL, nil, 0) }

func closeFD(fd int) error { return unix.Close(fd) }

// sigchldIgnored is the check of §12.14 item 8 on automatic reaping (it is
// only this check, not a proof of every signal disposition).
func sigchldIgnored() bool { return signal.Ignored(syscall.SIGCHLD) }
