//go:build windows

package wp25

import (
	"os"
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
