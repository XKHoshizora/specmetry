package sqlite

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// lockOffsetHigh places the locked byte far beyond the end of the (empty)
// lock file. Windows byte-range locks are mandatory: a locked byte cannot be
// read by anyone else, so locking byte 0 would make the file unreadable to
// backup tools and tests while a process holds it.
const lockOffsetHigh = 0x7fffffff

// tryLock locks one byte of f, exclusively or shared, without waiting;
// false means another process holds a conflicting lock.
func tryLock(f *os.File, exclusive bool) (bool, error) {
	flags := uint32(windows.LOCKFILE_FAIL_IMMEDIATELY)
	if exclusive {
		flags |= windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	err := windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, &windows.Overlapped{OffsetHigh: lockOffsetHigh})
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return false, nil
	}
	return err == nil, err
}

func unlock(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &windows.Overlapped{OffsetHigh: lockOffsetHigh})
}
