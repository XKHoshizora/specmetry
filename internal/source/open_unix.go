//go:build !windows

package source

import (
	"os"
	"syscall"
)

// openRegular opens with O_NONBLOCK, which returns at once for a FIFO
// without a writer, and O_NOCTTY, so that a terminal device never becomes
// the controlling terminal; fstat on the descriptor then tells what was
// opened. A regular file gets blocking mode back, which it reads the same
// with either way.
func openRegular(path string) (*os.File, error) {
	var fd int
	var err error
	for {
		fd, err = syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
		if err != syscall.EINTR {
			break
		}
	}
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		_ = syscall.Close(fd)
		return nil, &os.PathError{Op: "fstat", Path: path, Err: err}
	}
	if st.Mode&syscall.S_IFMT != syscall.S_IFREG {
		_ = syscall.Close(fd)
		return nil, &os.PathError{Op: "open", Path: path, Err: errNotRegular}
	}
	if err := syscall.SetNonblock(fd, false); err != nil {
		_ = syscall.Close(fd)
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(fd), path), nil
}
