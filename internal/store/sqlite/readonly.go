package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	sqlitedrv "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// This file reads the database version without writing anything, using
// SQLite itself — its locks and transaction isolation decide what is
// committed (codex-review-005 B2). SQLite only writes nothing in some file
// states, measured with the driver in use (modernc.org/sqlite 1.60.1, SQLite
// 3.53.4) on Windows and Linux (TestReadOnlyConnectionByFileState):
//
//   - with both -wal and -shm present, a connection with mode=ro and
//     readonly_shm=1 changes nothing — after a crash, while another process
//     writes, and in the state a specmetry connection leaves on close
//     (persistent WAL, see dsn);
//   - a rollback-journal database (such as a restored VACUUM INTO backup)
//     with no -wal, -shm or -journal is read without changes;
//   - in every other state a read-only connection creates or rewrites the
//     -wal or -shm, so the version is reported as unknown instead.
//
// Uncommitted data is never read: in WAL mode readers see only committed
// frames, and in rollback mode a writer that has put uncommitted pages into
// the database file holds a lock that makes the read fail with SQLITE_BUSY,
// which is reported as unknown too.

// uninspectableError explains why the database cannot be read without
// changing it; the version is then unknown.
type uninspectableError struct{ reason string }

func (e *uninspectableError) Error() string { return e.reason }

// readOnlyMeta is what a read-only inspection found.
type readOnlyMeta struct {
	exists  bool
	version int
	minBin  string
}

// fileState records which database files exist and their sizes.
type fileState struct {
	exists [4]bool
	sizes  [4]int64
}

var databaseSuffixes = [4]string{"", "-wal", "-shm", "-journal"}

const (
	dbFile = iota
	walFile
	shmFile
	journalFile
)

func statDatabaseFiles(dbPath string) (fileState, error) {
	var s fileState
	for i, suffix := range databaseSuffixes {
		fi, err := os.Lstat(dbPath + suffix)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return s, err
		}
		if !fi.Mode().IsRegular() {
			return s, fmt.Errorf("%s is not a regular file", dbPath+suffix)
		}
		s.exists[i], s.sizes[i] = true, fi.Size()
	}
	return s, nil
}

// readMetaReadOnly reads the schema version and minimum binary version
// without modifying any file. A missing database reports exists=false. It
// returns an *uninspectableError when that is not possible. With
// sameSizes, a change in the size of any database file during the read is
// also reported as uninspectable (doctor); otherwise only files appearing or
// disappearing are (Open, where other processes may be writing).
func readMetaReadOnly(ctx context.Context, dbPath string, sameSizes bool) (readOnlyMeta, error) {
	before, err := statDatabaseFiles(dbPath)
	if err != nil {
		return readOnlyMeta{}, err
	}
	if !before.exists[dbFile] {
		return readOnlyMeta{}, nil
	}
	if before.exists[journalFile] && before.sizes[journalFile] > 0 {
		return readOnlyMeta{}, &uninspectableError{"a rollback journal (" + dbPath + "-journal) is present: another program is writing the database, or an interrupted write must be played back first"}
	}
	if before.sizes[dbFile] == 0 {
		if before.exists[walFile] {
			return readOnlyMeta{}, &uninspectableError{"the database file is empty but has a -wal file"}
		}
		return readOnlyMeta{exists: true}, nil // created, never written
	}
	wal, err := walModeHeader(dbPath)
	if err != nil {
		return readOnlyMeta{}, err
	}
	switch {
	case before.exists[walFile] && before.exists[shmFile]:
	case !before.exists[walFile] && !before.exists[shmFile] && !wal:
	case !before.exists[walFile] && !before.exists[shmFile]:
		return readOnlyMeta{}, &uninspectableError{"the database's -wal and -shm files are missing (it was last closed by a program other than specmetry, or they were deleted); reading it would create them"}
	default:
		return readOnlyMeta{}, &uninspectableError{"only one of the database's -wal and -shm files exists; reading it would recover the database"}
	}

	m, readErr := queryReadOnly(ctx, dbPath)
	after, err := statDatabaseFiles(dbPath)
	if err != nil {
		return readOnlyMeta{}, err
	}
	if after.exists != before.exists || (sameSizes && after.sizes != before.sizes) {
		return readOnlyMeta{}, &uninspectableError{"the database files changed while they were being read; run the command again"}
	}
	if readErr != nil {
		var se *sqlitedrv.Error
		if errors.As(readErr, &se) && (se.Code()&0xff == sqlite3.SQLITE_BUSY || se.Code()&0xff == sqlite3.SQLITE_LOCKED) {
			return readOnlyMeta{}, &uninspectableError{"another program holds a write lock on the database"}
		}
		return readOnlyMeta{}, fmt.Errorf("cannot read the database: %w", readErr)
	}
	m.exists = true
	return m, nil
}

func queryReadOnly(ctx context.Context, dbPath string) (readOnlyMeta, error) {
	sdb, err := sql.Open("sqlite", "file:"+escapePath(dbPath)+"?mode=ro&readonly_shm=1&_pragma=busy_timeout(1000)")
	if err != nil {
		return readOnlyMeta{}, err
	}
	defer func() { _ = sdb.Close() }()
	sdb.SetMaxOpenConns(1)
	tx, err := sdb.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return readOnlyMeta{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var m readOnlyMeta
	m.version, m.minBin, err = readMeta(ctx, tx)
	return m, err
}

// walModeHeader reports whether the database header marks a WAL database
// (file format bytes 18 and 19).
func walModeHeader(dbPath string) (bool, error) {
	f, err := os.Open(dbPath)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()
	hdr := make([]byte, 20)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return false, fmt.Errorf("cannot read the database header: %w", err)
	}
	if !bytes.Equal(hdr[:16], []byte("SQLite format 3\x00")) {
		return false, errors.New("the database file is not a SQLite database")
	}
	return hdr[18] == 2 || hdr[19] == 2, nil
}
