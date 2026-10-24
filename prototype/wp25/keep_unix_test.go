//go:build linux || darwin

package wp25

import (
	"os"

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
