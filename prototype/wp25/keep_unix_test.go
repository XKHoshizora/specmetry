//go:build linux || darwin

package wp25

import (
	"os"

	"golang.org/x/sys/unix"
)

// keepOpen holds a second descriptor of f's open file, close-on-exec, until
// the returned function closes it: closing f then leaves the file open.
func keepOpen(f *os.File) (func(), error) {
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
	return func() { _ = unix.Close(fd) }, nil
}
