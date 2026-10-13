package sqlite

import (
	"context"
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
	// Limitation explains why the database could not be read without
	// modifying it (for example, a rollback journal that would have to be
	// played back first); the version fields are then unknown.
	Limitation string
}

// Inspect reports the database state without creating, migrating, recovering
// or writing anything: the database, its WAL and -shm, and the directory are
// left exactly as they were (codex-review-004 B2; see snapshot.go).
func Inspect(ctx context.Context, dir datadir.Dir) (Inspection, error) {
	in := Inspection{SupportedVersion: SchemaVersion()}
	if _, err := os.Lstat(dir.Join(datadir.MigrateLockFile)); err == nil {
		in.MigrationLocked = true
	} else if !errors.Is(err, fs.ErrNotExist) {
		return in, err
	}
	m, err := readSnapshotMeta(ctx, dir.Join(datadir.DatabaseFile))
	var u *uninspectableError
	if errors.As(err, &u) {
		in.Exists, in.Limitation = true, u.reason
		return in, nil
	}
	if err != nil {
		return in, err
	}
	in.Exists, in.SchemaVersion, in.MinBinaryVersion = m.exists, m.version, m.minBin
	return in, nil
}

// Compatibility applies Open's version guard for a binary of the given
// version: nil means such a binary may open the database.
func (in Inspection) Compatibility(binVersion string) error {
	if !in.Exists || in.Limitation != "" {
		return nil
	}
	return guardVersion(in.SchemaVersion, in.SupportedVersion, in.MinBinaryVersion, binVersion)
}
