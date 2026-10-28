//go:build linux

package wp25

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"sync/atomic"
)

// p1Probing is set while p1Barrier asks for an interval, so that a gate
// regression can fail the baseline read of that call only.
var p1Probing atomic.Bool

// p1Exclusive runs fn in the exclusive interval of a gate of its own: the
// baseline check, the subreaper set for fn only.
func p1Exclusive(fn func() error) error { return newGate2().exclusive(fn) }

func p1Subreaper(s *scen) int { return subreaper(s) }

// p1Barrier asks for the next exclusive interval while bypassing the
// host's admission: Linux's own barrier, recorded apart from Q-1.
func p1Barrier(*scen) (bool, string) {
	called := false
	p1Probing.Store(true)
	err := newGate2().exclusive(func() error { called = true; return nil })
	p1Probing.Store(false)
	refused := errors.Is(err, errBaseline) && !called
	return refused, fmt.Sprintf("refused by the baseline: %v; entered: %v", refused, called)
}

func obsFD(o exitObserver) int { return o.fd }

// fdClosed reads /proc/self/fd: no link is a closed descriptor; a pidfd
// link is an open one; any other link is a reused number, which says
// nothing either way.
func fdClosed(fd int) (closed bool, detail string, known bool) {
	l, err := os.Readlink("/proc/self/fd/" + strconv.Itoa(fd))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return true, fmt.Sprintf("descriptor %d closed", fd), true
	case err != nil:
		return false, err.Error(), false
	case l == "anon_inode:[pidfd]":
		return false, fmt.Sprintf("descriptor %d is still a pidfd", fd), true
	}
	return false, fmt.Sprintf("descriptor %d now names %q", fd, l), false
}
