//go:build windows

package wp25

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

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

// closeRead closes a stream's read end (Review 081 B1). os.Pipe's handles
// are synchronous: Close cancels the reads already issued (CancelIoEx),
// then waits for every read holding the file, and a read that makes its
// system call just after that blocks until data or EOF, with Close waiting
// for it. So while Close waits, the stop cancels the pipe's I/O again, once
// a millisecond, until Close returns: a read that reaches its system call
// is cancelled then, and Close returns.
//
// The cancellation goes through a handle of the stop's own to the same
// pipe, taken before Close, so it never names a handle Close may already
// have freed; it is closed once Close has returned. A read that never
// reaches its system call (one that holds the file and waits elsewhere)
// keeps Close waiting: the stop is then pending, and its run hands it on.
//
// The calls' results (Review 082 B1-Rb):
//
//   - CancelIoEx fails with ERROR_NOT_FOUND when no request of the pipe is
//     outstanding: the race the loop expects, before the read reaches its
//     system call or after it was cancelled. It succeeds when it asked a
//     request to cancel, which is not yet its end: Close's return is.
//   - Any other failure of CancelIoEx means the stop cannot cancel through
//     its handle. It goes to fail at once, so a holder of the pending stop
//     sees it; the loop goes on, since Close may still return, and later
//     failures are counted, not repeated. It is never cleared.
//   - CloseHandle on the stop's own handle failing leaves its release
//     unconfirmed: an error of the stop, even if Close succeeded. The
//     handle is not closed again, as it may already be another's.
//
// Errors name the call and Windows' error, never a handle's value.
func closeRead(r io.Closer, fail func(error)) error {
	f, ok := r.(*os.File)
	if !ok {
		return r.Close()
	}
	own, derr := ownHandle(f)
	if derr != nil {
		// No handle to cancel through: the close is all that is left, and
		// the stop reports why it could not cancel.
		return errors.Join(fmt.Errorf("no handle of the stop's own: %w", derr), f.Close())
	}
	closed := make(chan error, 1)
	go func() { closed <- f.Close() }()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	failed := 0
	for {
		select {
		case err := <-closed:
			if failed > 1 {
				err = errors.Join(err, fmt.Errorf("%d cancellations through the stop's own handle failed in all", failed))
			}
			if cerr := closeOwn(own); cerr != nil {
				err = errors.Join(err, fmt.Errorf("closing the stop's own handle: %w", cerr))
			}
			return err
		case <-tick.C:
			if err := cancelOwn(own, nil); err != nil && !errors.Is(err, windows.ERROR_NOT_FOUND) {
				failed++
				if failed == 1 {
					fail(fmt.Errorf("cancelling through the stop's own handle: %w", err))
				}
			}
		}
	}
}

// The calls the stop makes on its own handle; a direct check
// (stop_api_windows_test.go) replaces them with faults.
var (
	cancelOwn = windows.CancelIoEx
	closeOwn  = windows.CloseHandle
)

// ownHandle duplicates f's handle (non-inheritable) for the caller to own.
func ownHandle(f *os.File) (windows.Handle, error) {
	rc, err := f.SyscallConn()
	if err != nil {
		return 0, err
	}
	var h windows.Handle
	var derr error
	cur := windows.CurrentProcess()
	if err := rc.Control(func(s uintptr) {
		derr = windows.DuplicateHandle(cur, windows.Handle(s), cur, &h, 0, false, windows.DUPLICATE_SAME_ACCESS)
	}); err != nil {
		return 0, err
	}
	return h, derr
}

// inSystemRead marks a goroutine blocked in its pipe's read: on Windows it
// is in the system call, in ReadFile.
var inSystemRead = []string{"[syscall", "syscall.ReadFile"}
