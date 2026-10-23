//go:build windows

package wp25

import (
	"os"

	"golang.org/x/sys/windows"
)

// keepOpen holds a second, non-inheritable handle of f's open file until
// the returned function closes it: closing f then leaves the file open.
func keepOpen(f *os.File) (func(), error) {
	rc, err := f.SyscallConn()
	if err != nil {
		return nil, err
	}
	var h windows.Handle
	var derr error
	cur := windows.CurrentProcess()
	if err := rc.Control(func(s uintptr) {
		derr = windows.DuplicateHandle(cur, windows.Handle(s), cur, &h, 0, false, windows.DUPLICATE_SAME_ACCESS)
	}); err != nil {
		return nil, err
	}
	if derr != nil {
		return nil, derr
	}
	return func() { _ = windows.CloseHandle(h) }, nil
}
