//go:build linux || darwin

package wp25

import (
	"errors"
	"io"

	"golang.org/x/sys/unix"
)

// sysRead is one system read of a pipe's read end; again reports that the
// descriptor had no data yet (it is non-blocking), so Go's poller waits.
func sysRead(fd uintptr, p []byte) (n int, err error, again bool) {
	n, err = unix.Read(int(fd), p)
	switch {
	case errors.Is(err, unix.EAGAIN):
		return 0, nil, true
	case err == nil && n == 0:
		return 0, io.EOF, false
	case err != nil:
		return 0, err, false
	}
	return n, nil, false
}

// inSystemRead marks a goroutine blocked in its pipe's read: on Unix it is
// parked in Go's poller.
var inSystemRead = []string{"[IO wait", "internal/poll.runtime_pollWait"}
