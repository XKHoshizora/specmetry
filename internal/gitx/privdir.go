package gitx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// Private directories (WP2.3 spec §4.3, "the temporary directory's
// lifecycle"): Clean's private git directories, and the directory that
// holds the missing grafts and shallow files of the scan's reads. Each is
// a new directory in the system's temporary directory, readable only by
// the current user, named with privatePrefix, and locked by the process
// that uses it for as long as it does, so that a directory left behind by
// a process that crashed can be told from one still in use: the next
// Clean removes the first kind, never the second. Nothing in them is a
// project fact; they never reach a log, an error or a backup.

const (
	privatePrefix = "specmetry-private-"
	privateLock   = "lock"
	// staleUnlocked is how old a directory without a lock file must be to
	// be taken for one whose process stopped before it locked it.
	staleUnlocked = time.Hour
)

// removeAll removes a directory tree; tests replace it to make removal
// fail.
var removeAll = os.RemoveAll

// privateDir is a private directory in use.
type privateDir struct {
	path string
	lock *os.File
}

// newPrivateDir makes and locks a private directory.
func newPrivateDir() (*privateDir, error) {
	dir, err := os.MkdirTemp("", privatePrefix)
	if err != nil {
		return nil, apperr.New(apperr.Environment, "cannot make a temporary directory to read git with ("+osProblem(err)+")")
	}
	lock, err := os.OpenFile(filepath.Join(dir, privateLock), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err == nil {
		var locked bool
		if locked, err = tryLock(lock); err == nil && !locked {
			err = errors.New("the lock is held")
		}
		if err != nil {
			_ = lock.Close()
		}
	}
	if err != nil {
		_ = removeAll(dir)
		return nil, apperr.New(apperr.Environment, "cannot lock a temporary directory to read git with")
	}
	return &privateDir{path: dir, lock: lock}, nil
}

// join names a file in the directory.
func (d *privateDir) join(elem ...string) string {
	return filepath.Join(append([]string{d.path}, elem...)...)
}

// remove unlocks and removes the directory. On Windows a file a git process
// has just closed can stay in use for a moment, so a failure is tried again
// briefly; one that persists is an environment error (the spec: a read
// whose directory cannot be removed is not reported clean).
func (d *privateDir) remove() error {
	_ = unlock(d.lock)
	_ = d.lock.Close()
	var err error
	for range 20 {
		if err = removeAll(d.path); err == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return apperr.New(apperr.Environment, "cannot remove a temporary directory Specmetry read git with ("+osProblem(err)+")")
}

// removeStalePrivateDirs removes the private directories left behind by
// processes that stopped: those whose lock no process holds, and those
// without a lock file that are older than staleUnlocked. A directory in
// use, or one it cannot tell, is left alone; failures are not errors.
func removeStalePrivateDirs() {
	tmp := os.TempDir()
	entries, err := os.ReadDir(tmp)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), privatePrefix) || !e.IsDir() {
			continue // a symbolic link is not followed
		}
		dir := filepath.Join(tmp, e.Name())
		lock, err := os.OpenFile(filepath.Join(dir, privateLock), os.O_RDWR, 0)
		if errors.Is(err, fs.ErrNotExist) {
			if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > staleUnlocked {
				_ = removeAll(dir)
			}
			continue
		}
		if err != nil {
			continue
		}
		locked, err := tryLock(lock)
		if err != nil || !locked {
			_ = lock.Close()
			continue
		}
		_ = unlock(lock)
		_ = lock.Close()
		_ = removeAll(dir)
	}
}
