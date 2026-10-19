package gitx

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func watchConsumerExit(pid int) (func() error, func(), error) {
	fd, err := unix.Kqueue()
	if err != nil {
		return nil, nil, err
	}
	change := []unix.Kevent_t{{Ident: uint64(pid), Filter: unix.EVFILT_PROC, Flags: unix.EV_ADD | unix.EV_ENABLE | unix.EV_ONESHOT, Fflags: unix.NOTE_EXIT}}
	if _, err := unix.Kevent(fd, change, nil, nil); err != nil {
		_ = unix.Close(fd)
		return nil, nil, err
	}
	wait := func() error {
		events := make([]unix.Kevent_t, 1)
		timeout := unix.Timespec{Sec: 30}
		n, err := unix.Kevent(fd, nil, events, &timeout)
		if err != nil {
			return err
		}
		if n != 1 || events[0].Fflags&unix.NOTE_EXIT == 0 {
			return fmt.Errorf("Git exit was not observed: %d", n)
		}
		return nil
	}
	return wait, func() { _ = unix.Close(fd) }, nil
}
