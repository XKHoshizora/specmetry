package sqlite

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
)

// migrateLock is the cross-process migration lock (.migrate.lock), created
// with O_EXCL. A crashed process leaves the file behind; the error message
// tells the user how to recover rather than guessing that it is stale.
type migrateLock struct{ path string }

func acquireMigrateLock(ctx context.Context, path string, timeout time.Duration) (*migrateLock, error) {
	deadline := time.Now().Add(timeout)
	for {
		f, err := datadir.CreatePrivateFile(path, true)
		if err == nil {
			_, _ = fmt.Fprintf(f, "pid=%d\nstarted=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
			_ = f.Close()
			return &migrateLock{path: path}, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, apperr.Wrap(apperr.Environment, err, "cannot create the migration lock")
		}
		if time.Now().After(deadline) {
			return nil, apperr.Newf(apperr.Environment,
				"another specmetry process is migrating the database (lock file %s); if no other process is running, delete the lock file and retry", path)
		}
		select {
		case <-ctx.Done():
			return nil, apperr.Wrap(apperr.Environment, ctx.Err(), "interrupted while waiting for the migration lock")
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (l *migrateLock) release() {
	_ = os.Remove(l.path)
}
