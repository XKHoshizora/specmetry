package gitx

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func watchConsumerExit(pid int) (func() error, func(), error) {
	fd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		return nil, nil, err
	}
	wait := func() error {
		poll := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		for {
			n, err := unix.Poll(poll, 30000)
			if err == unix.EINTR {
				continue
			}
			if err != nil {
				return err
			}
			if n != 1 || poll[0].Revents&unix.POLLIN == 0 {
				return fmt.Errorf("Git exit was not observed: %d/%x", n, poll[0].Revents)
			}
			return nil
		}
	}
	return wait, func() { _ = unix.Close(fd) }, nil
}
