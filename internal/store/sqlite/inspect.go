package sqlite

import (
	"context"
	"errors"

	"github.com/XKHoshizora/specmetry/internal/datadir"
)

// Inspection is a read-only view of the database for `specmetry doctor`.
type Inspection struct {
	Exists           bool
	SchemaVersion    int
	MinBinaryVersion string
	SupportedVersion int
	// Locked reports that another process is opening or migrating the
	// database right now.
	Locked bool
	// Limitation explains why the database could not be read without
	// changing it, or not consistently; the version fields are then unknown
	// and must not be reported.
	Limitation string
}

// Inspect reports the database state without creating, migrating,
// recovering, checkpointing or writing anything: the database, its -wal and
// -shm and the directory are left as they were (codex-review-004 B2,
// codex-review-005 B2; see readonly.go). The version is reported only when a
// read-only SQLite connection could read it in a file state where it changes
// nothing, and no database file changed in size or appeared or disappeared
// meanwhile; otherwise Limitation says why it is unknown.
func Inspect(ctx context.Context, dir datadir.Dir) (Inspection, error) {
	in := Inspection{SupportedVersion: SchemaVersion()}
	locked, err := lockHeld(dir.Join(datadir.MigrateLockFile))
	if err != nil {
		return in, err
	}
	in.Locked = locked
	m, err := readMetaReadOnly(ctx, dir.Join(datadir.DatabaseFile), true)
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
// version: nil means such a binary may open the database. It says nothing
// when the version is unknown (Limitation).
func (in Inspection) Compatibility(binVersion string) error {
	if !in.Exists || in.Limitation != "" {
		return nil
	}
	return guardVersion(in.SchemaVersion, in.SupportedVersion, in.MinBinaryVersion, binVersion)
}
