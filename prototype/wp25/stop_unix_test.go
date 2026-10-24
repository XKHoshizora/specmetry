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

// closeRead closes a stream's read end. On Unix the read end is in Go's
// poller: Close marks the file closing and wakes a read parked in the
// poller, and a read that has not reached it yet finds the mark there, so
// the close waits only for a read to leave its system call.
func closeRead(r io.Closer, _ func(error)) error { return r.Close() }

// inSystemRead marks a goroutine blocked in its pipe's read: on Unix it is
// parked in Go's poller.
var inSystemRead = []string{"[IO wait", "internal/poll.runtime_pollWait"}
