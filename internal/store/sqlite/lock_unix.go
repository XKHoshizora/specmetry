//go:build !windows

package sqlite

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// tryLock takes an exclusive or shared flock on f without waiting; false
// means another process holds a conflicting lock.
func tryLock(f *os.File, exclusive bool) (bool, error) {
	how := unix.LOCK_SH
	if exclusive {
		how = unix.LOCK_EX
	}
	err := unix.Flock(int(f.Fd()), how|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) {
		return false, nil
	}
	return err == nil, err
}

func unlock(f *os.File) error { return unix.Flock(int(f.Fd()), unix.LOCK_UN) }
