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
// host's admission: Linux's own barrier, recorded apart from Q-1. Only a
// baseline read and found not empty is that barrier's refusal; one that
// cannot be read, or any other error, is a failed observation of this
// run (Review 095 B1), whatever it would do to a real check.
func p1Barrier(s *scen) (refused, entered bool, detail string) {
	p1Probing.Store(true)
	err := newGate2().exclusive(func() error { entered = true; return nil })
	p1Probing.Store(false)
	switch {
	case errors.Is(err, errBaselineUnread):
		s.flag("observation failed: the barrier's baseline: %v", err)
	case err != nil && !errors.Is(err, errBaseline):
		s.flag("observation failed: the barrier's interval: %v", err)
	}
	refused = errors.Is(err, errBaseline) && !errors.Is(err, errBaselineUnread) && !entered
	return refused, entered, fmt.Sprintf("refused by a baseline that has children: %v; entered: %v; error: %v", refused, entered, err)
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
