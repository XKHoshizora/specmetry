//go:build windows

package wp25

import (
	"errors"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// golang.org/x/sys/windows does not wrap PeekNamedPipe.
var procPeekNamedPipe = kernel32.NewProc("PeekNamedPipe")

// kept is a second, non-inheritable handle of a pipe's read end, held by a
// scenario: while it is open, closing the first one leaves the pipe open.
type kept struct{ h windows.Handle }

func keepOpen(f *os.File) (*kept, error) {
	h, err := ownHandle(f)
	if err != nil {
		return nil, err
	}
	return &kept{h: h}, nil
}

func (k *kept) close() { _ = windows.CloseHandle(k.h) }

// available is how many bytes wait in the pipe, read without consuming
// them.
func (k *kept) available() (int, error) {
	var n uint32
	r, _, err := procPeekNamedPipe.Call(uintptr(k.h), 0, 0, 0, uintptr(unsafe.Pointer(&n)), 0)
	if r == 0 {
		return 0, err
	}
	return int(n), nil
}

// drain reads what waits in the pipe until EOF, every writer of it gone,
// or the deadline. It reads only what PeekNamedPipe says is there, so a
// read never waits; an empty pipe whose writers are all gone is
// ERROR_BROKEN_PIPE, which is the EOF.
func (k *kept) drain(deadline time.Time) (n int, eof bool, err error) {
	buf := make([]byte, 32<<10)
	for {
		var avail uint32
		r, _, perr := procPeekNamedPipe.Call(uintptr(k.h), 0, 0, 0, uintptr(unsafe.Pointer(&avail)), 0)
		switch {
		case r == 0 && errors.Is(perr, windows.ERROR_BROKEN_PIPE):
			return n, true, nil
		case r == 0:
			return n, false, perr
		case avail > 0:
			want := int(avail)
			if want > len(buf) {
				want = len(buf)
			}
			var got uint32
			if rerr := windows.ReadFile(k.h, buf[:want], &got, nil); rerr != nil {
				if errors.Is(rerr, windows.ERROR_BROKEN_PIPE) {
					return n, true, nil
				}
				return n, false, rerr
			}
			n += int(got)
		case time.Now().After(deadline):
			return n, false, nil
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
