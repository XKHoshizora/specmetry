//go:build linux || darwin

package wp25

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// kept is a second descriptor of a pipe's read end, close-on-exec, held by
// a scenario: while it is open, closing the first one leaves the pipe open.
type kept struct{ fd int }

func keepOpen(f *os.File) (*kept, error) {
	rc, err := f.SyscallConn()
	if err != nil {
		return nil, err
	}
	fd, derr := -1, error(nil)
	if err := rc.Control(func(s uintptr) { fd, derr = unix.FcntlInt(s, unix.F_DUPFD_CLOEXEC, 0) }); err != nil {
		return nil, err
	}
	if derr != nil {
		return nil, derr
	}
	return &kept{fd: fd}, nil
}

func (k *kept) close() { _ = unix.Close(k.fd) }

// available is how many bytes wait in the pipe, read without consuming
// them.
func (k *kept) available() (int, error) { return unix.IoctlGetInt(k.fd, fionread) }

// drain reads what waits in the pipe until EOF, every writer of it gone,
// or the deadline. The kept descriptor shares the read end's non-blocking
// mode, so a read never waits: no data yet is EAGAIN, EOF is a read of 0.
func (k *kept) drain(deadline time.Time) (n int, eof bool, err error) {
	buf := make([]byte, 32<<10)
	for {
		m, rerr := unix.Read(k.fd, buf)
		switch {
		case errors.Is(rerr, unix.EAGAIN):
			if time.Now().After(deadline) {
				return n, false, nil
			}
			time.Sleep(time.Millisecond)
		case rerr != nil:
			return n, false, rerr
		case m == 0:
			return n, true, nil
		default:
			n += m
		}
	}
}
