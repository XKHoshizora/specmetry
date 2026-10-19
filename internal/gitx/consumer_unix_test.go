//go:build !windows

package gitx

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func pauseConsumer(pid int) error {
	if err := unix.Kill(pid, unix.SIGSTOP); err != nil {
		return err
	}
	var status unix.WaitStatus
	for {
		got, err := unix.Wait4(pid, &status, unix.WUNTRACED, nil)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return err
		}
		// Darwin's native WIFSTOPPED distinguishes SIGCONT (19), whereas
		// x/sys v0.49.0's shared BSD Stopped excludes SIGSTOP instead. Require
		// the exact native stop reply for our signal; WCONTINUED is not requested.
		// https://github.com/apple-oss-distributions/xnu/blob/main/bsd/sys/wait.h
		if got != pid || uint32(status)&0x7f != 0x7f || unix.Signal(status>>8) != unix.SIGSTOP {
			return fmt.Errorf("Git exited instead of acknowledging stop: %v", status)
		}
		return nil
	}
}

func resumeConsumer(pid int) error { return unix.Kill(pid, unix.SIGCONT) }
