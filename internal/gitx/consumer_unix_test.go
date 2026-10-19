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
		_, err := unix.Wait4(pid, &status, unix.WUNTRACED, nil)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return err
		}
		if !status.Stopped() {
			return fmt.Errorf("Git exited instead of acknowledging stop: %v", status)
		}
		return nil
	}
}

func resumeConsumer(pid int) error { return unix.Kill(pid, unix.SIGCONT) }
