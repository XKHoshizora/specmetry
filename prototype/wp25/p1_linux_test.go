//go:build linux

package wp25

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
)

// p1Exclusive runs fn in the exclusive interval of a gate of its own: the
// baseline check, the subreaper set for fn only. The label lets a gate
// regression fail this interval's own steps, not the barrier's.
func p1Exclusive(fn func() error) error {
	g := newGate2()
	g.label = "p1.inner"
	return g.exclusive(fn)
}

func p1Subreaper(s *scen) int { return subreaper(s) }

// p1Barrier asks for the next exclusive interval while bypassing the
// host's admission: Linux's own barrier, recorded apart from Q-1. Only a
// baseline read and found not empty is that barrier's refusal; one that
// cannot be read, or any other error, is a failed observation of this
// run (Review 095 B1), whatever it would do to a real check.
//
// The setting is read again after the interval (Review 096 B1-R): what
// the barrier's own clear left, recorded apart from what it returned.
func p1Barrier(s *scen) (refused, entered bool, detail string) {
	g := newGate2()
	g.label = "p1.barrier"
	err := g.exclusive(func() error { entered = true; return nil })
	sub := subreaper(s)
	s.expect("subreaper after the barrier's interval", 0, sub)
	switch {
	case errors.Is(err, errBaselineUnread):
		s.flag("observation failed: the barrier's baseline: %v", err)
	case err != nil && !errors.Is(err, errBaseline):
		s.flag("observation failed: the barrier's interval: %v", err)
	}
	refused = errors.Is(err, errBaseline) && !errors.Is(err, errBaselineUnread) && !entered
	return refused, entered, fmt.Sprintf("refused by a baseline that has children: %v; entered: %v; error: %v; subreaper after it: %d", refused, entered, err, sub)
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
