//go:build windows

package wp25

import (
	"errors"
	"io"

	"golang.org/x/sys/windows"
)

// sysRead is one system read of a pipe's read end: os.Pipe's handles are
// synchronous, so ReadFile blocks until data, EOF, or a cancellation.
func sysRead(fd uintptr, p []byte) (n int, err error, again bool) {
	var got uint32
	err = windows.ReadFile(windows.Handle(fd), p, &got, nil)
	switch {
	case errors.Is(err, windows.ERROR_BROKEN_PIPE):
		return 0, io.EOF, false
	case err != nil:
		return 0, err, false
	case got == 0:
		return 0, io.EOF, false
	}
	return int(got), nil, false
}

// inSystemRead marks a goroutine blocked in its pipe's read: on Windows it
// is in the system call, in ReadFile.
var inSystemRead = []string{"[syscall", "syscall.ReadFile"}
