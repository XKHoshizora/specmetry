//go:build darwin

package wp25

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

// macOS has no subreaper and no baseline barrier.
func p1Exclusive(fn func() error) error { return fn() }

func p1Subreaper(*scen) int { return -1 }

func p1Barrier(*scen) (bool, string) { return false, "no baseline barrier on this platform" }

func obsFD(o exitObserver) int { return o.s.kq }

// fdClosed asks fcntl: EBADF is a closed descriptor; an open one may be
// the kqueue or a reused number, which says nothing either way.
func fdClosed(fd int) (closed bool, detail string, known bool) {
	_, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0)
	switch {
	case errors.Is(err, unix.EBADF):
		return true, fmt.Sprintf("descriptor %d closed", fd), true
	case err != nil:
		return false, err.Error(), false
	}
	return false, fmt.Sprintf("descriptor %d is open (the kqueue, or a reused number)", fd), false
}
