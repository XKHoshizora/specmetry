package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"os"

	"github.com/XKHoshizora/specmetry/internal/datadir"
)

// Inspection is a read-only view of the database for `specmetry doctor`.
type Inspection struct {
	Exists           bool
	SchemaVersion    int
	MinBinaryVersion string
	SupportedVersion int
	MigrationLocked  bool
}

// Inspect reports the database state without creating, migrating or writing
// anything.
func Inspect(ctx context.Context, dir datadir.Dir) (Inspection, error) {
	in := Inspection{SupportedVersion: SchemaVersion()}
	if _, err := os.Stat(dir.Join(datadir.MigrateLockFile)); err == nil {
		in.MigrationLocked = true
	}
	dbPath := dir.Join(datadir.DatabaseFile)
	if _, err := os.Stat(dbPath); errors.Is(err, fs.ErrNotExist) {
		return in, nil
	} else if err != nil {
		return in, err
	}
	in.Exists = true
	// mode=rw never creates the file; query_only forbids writes.
	sdb, err := sql.Open("sqlite", "file:"+escapePath(dbPath)+"?mode=rw&_pragma=query_only(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return in, err
	}
	defer func() { _ = sdb.Close() }()
	in.SchemaVersion, in.MinBinaryVersion, err = readMeta(ctx, sdb)
	return in, err
}
