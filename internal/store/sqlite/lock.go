package sqlite

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
)

// openLock serializes, across processes, everything from deciding that the
// database is compatible to having it open: the compatibility check, the
// read-write connection, tombstones and migrations (codex-review-005 B3). No
// process can migrate the database while another is between its check and
// its connection. It is an operating-system lock on .migrate.lock (flock on
// Unix, LockFileEx on Windows), so a process that crashes cannot leave it
// held; the file itself stays.
type openLock struct{ f *os.File }

func acquireOpenLock(ctx context.Context, path string, timeout time.Duration) (*openLock, error) {
	f, err := datadir.CreatePrivateFile(path, false)
	if err != nil {
		return nil, apperr.Wrap(apperr.Environment, err, "cannot open the database lock "+path)
	}
	deadline := time.Now().Add(timeout)
	for {
		ok, err := tryLock(f, true)
		if err != nil {
			_ = f.Close()
			return nil, apperr.Wrap(apperr.Environment, err, "cannot lock "+path)
		}
		if ok {
			return &openLock{f: f}, nil
		}
		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, apperr.Newf(apperr.Environment,
				"another specmetry process is opening or migrating the database (waited %s for the lock on %s); try again", timeout, path)
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, apperr.Wrap(apperr.Environment, ctx.Err(), "interrupted while waiting for the database lock")
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func (l *openLock) release() {
	_ = unlock(l.f)
	_ = l.f.Close()
}

// lockHeld reports whether another process holds the lock at path, without
// creating the file or waiting.
func lockHeld(path string) (bool, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()
	ok, err := tryLock(f, false)
	if err != nil || !ok {
		return !ok, err
	}
	return false, unlock(f)
}
