// Package sqlite is Specmetry's SQLite storage foundation: opening the
// database with WAL and a single writer connection, the fixed open order
// (version guard → tombstones → migration backup → migration), the global
// fact sequence, and append-only enforcement (architecture.md §11, §13).
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
	_ "modernc.org/sqlite" // registers the "sqlite" driver

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/buildinfo"
	"github.com/XKHoshizora/specmetry/internal/datadir"
)

// Options configures Open. The zero value uses the embedded migrations and
// the running binary's version.
type Options struct {
	// Migrations overrides the embedded migrations (tests only).
	Migrations []Migration
	// BinaryVersion overrides buildinfo.Version (tests only).
	BinaryVersion string
	// LockTimeout bounds the wait for another process's migration.
	LockTimeout time.Duration
	// KeepBackups is the number of pre-migration backups to keep.
	KeepBackups int
	// Now overrides the clock used for backup names (tests only).
	Now func() time.Time
}

// DB is an open Specmetry database.
type DB struct {
	sql  *sql.DB
	dir  datadir.Dir
	path string
}

// OpenResult reports what Open did.
type OpenResult struct {
	FromVersion       int
	ToVersion         int
	BackupPath        string
	TombstoneRowsGone int64
}

// Open opens (creating if necessary) the database in dir and brings it to
// the latest schema. Order (architecture.md §11.2):
//  1. version guard — refuse databases newer than this binary;
//  2. apply tombstones;
//  3. if migration is needed: migration lock, VACUUM INTO backup, then all
//     pending migrations in one BEGIN EXCLUSIVE transaction.
func Open(ctx context.Context, dir datadir.Dir, opts Options) (*DB, OpenResult, error) {
	var res OpenResult
	migs := opts.Migrations
	if migs == nil {
		var err error
		if migs, err = Migrations(); err != nil {
			return nil, res, apperr.Wrap(apperr.Internal, err, "embedded migrations are invalid")
		}
	} else if err := validateSequence(migs); err != nil {
		return nil, res, apperr.Wrap(apperr.Internal, err, "migrations are invalid")
	}
	target := len(migs)
	binVersion := opts.BinaryVersion
	if binVersion == "" {
		binVersion = buildinfo.Version
	}
	if opts.LockTimeout == 0 {
		opts.LockTimeout = 30 * time.Second
	}
	if opts.KeepBackups == 0 {
		opts.KeepBackups = 3
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}

	if err := dir.Ensure(); err != nil {
		return nil, res, err
	}
	dbPath := dir.Join(datadir.DatabaseFile)
	if err := precreate(dbPath); err != nil {
		return nil, res, apperr.Wrap(apperr.Environment, err, "cannot create the database file")
	}
	sdb, err := sql.Open("sqlite", dsn(dbPath))
	if err != nil {
		return nil, res, apperr.Wrap(apperr.Environment, err, "cannot open the database")
	}
	// A single connection serializes all writes from this process
	// (architecture.md §13); SQLite locking serializes across processes.
	sdb.SetMaxOpenConns(1)
	db := &DB{sql: sdb, dir: dir, path: dbPath}
	fail := func(err error) (*DB, OpenResult, error) {
		_ = sdb.Close()
		return nil, res, err
	}

	current, minBin, err := readMeta(ctx, sdb)
	if err != nil {
		return fail(apperr.Wrap(apperr.Environment, err, "cannot read the database version"))
	}
	if err := guardVersion(current, target, minBin, binVersion); err != nil {
		return fail(err)
	}
	if res.TombstoneRowsGone, err = applyTombstones(ctx, sdb, dir.Join(datadir.TombstonesFile)); err != nil {
		return fail(apperr.Wrap(apperr.Environment, err, "cannot apply tombstones"))
	}
	res.FromVersion, res.ToVersion = current, current
	if current == target {
		return db, res, nil
	}

	lock, err := acquireMigrateLock(ctx, dir.Join(datadir.MigrateLockFile), opts.LockTimeout)
	if err != nil {
		return fail(err)
	}
	defer lock.release()
	// Another process may have migrated while we waited for the lock.
	if current, minBin, err = readMeta(ctx, sdb); err != nil {
		return fail(apperr.Wrap(apperr.Environment, err, "cannot read the database version"))
	}
	if err := guardVersion(current, target, minBin, binVersion); err != nil {
		return fail(err)
	}
	res.FromVersion, res.ToVersion = current, current
	if current == target {
		return db, res, nil
	}
	if current > 0 {
		if res.BackupPath, err = backup(ctx, sdb, dir, current, target, opts.Now(), opts.KeepBackups); err != nil {
			return fail(apperr.Wrap(apperr.Environment, err, "cannot back up the database before migrating"))
		}
	}
	if err := migrate(ctx, sdb, migs, current); err != nil {
		return fail(apperr.Wrap(apperr.Environment, err, fmt.Sprintf("migration from schema v%d to v%d failed and was rolled back", current, target)))
	}
	res.ToVersion = target
	return db, res, nil
}

// guardVersion refuses databases written by a newer binary (TD-16).
func guardVersion(current, target int, minBin, binVersion string) error {
	if current > target {
		return apperr.Newf(apperr.Environment,
			"the database uses schema v%d but this specmetry binary only supports up to v%d; upgrade specmetry", current, target)
	}
	if minBin != "" && semver.IsValid("v"+minBin) && semver.IsValid("v"+binVersion) &&
		semver.Compare("v"+binVersion, "v"+minBin) < 0 {
		return apperr.Newf(apperr.Environment,
			"the database requires specmetry %s or newer (this binary is %s); upgrade specmetry", minBin, binVersion)
	}
	return nil
}

func migrate(ctx context.Context, sdb *sql.DB, migs []Migration, current int) error {
	conn, err := sdb.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(ctx, `BEGIN EXCLUSIVE`); err != nil {
		return err
	}
	rollback := func(err error) error {
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		return err
	}
	for _, m := range migs[current:] {
		if _, err := conn.ExecContext(ctx, m.SQL); err != nil {
			return rollback(fmt.Errorf("migration %04d_%s: %w", m.Version, m.Name, err))
		}
	}
	// min_binary_version is changed only by a migration that needs it, never
	// set to the migrating binary's version (that would lock out older
	// binaries that understand the same schema).
	if _, err := conn.ExecContext(ctx, `UPDATE meta SET schema_version = ? WHERE id = 1`, len(migs)); err != nil {
		return rollback(err)
	}
	_, err = conn.ExecContext(ctx, `COMMIT`)
	if err != nil {
		return rollback(err)
	}
	return nil
}

// precreate creates the database file with owner-only permissions so that
// SQLite's -wal and -shm files, which copy the database file's mode, are
// private as well.
func precreate(path string) error {
	f, err := datadir.CreatePrivateFile(path, true)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return f.Close()
}

// escapePath makes a file path safe inside a SQLite URI filename.
func escapePath(path string) string {
	return strings.NewReplacer("%", "%25", "?", "%3f", "#", "%23").Replace(filepath.ToSlash(path))
}

func dsn(path string) string {
	return "file:" + escapePath(path) +
		"?_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		// BeginTx issues BEGIN IMMEDIATE: the write lock is taken before any
		// read, so read-compare-write sequences cannot act on a stale
		// snapshot (codex-review-003; data-model.md §7.1).
		"&_txlock=immediate"
}

// Close closes the database.
func (db *DB) Close() error { return db.sql.Close() }

// Path returns the database file path.
func (db *DB) Path() string { return db.path }

// SchemaVersion returns the database's current schema version.
func (db *DB) SchemaVersion(ctx context.Context) (int, error) {
	v, _, err := readMeta(ctx, db.sql)
	return v, err
}

// WriteTx runs fn in a write transaction that holds the write lock from its
// first statement (BEGIN IMMEDIATE). fn's error rolls the transaction back.
func (db *DB) WriteTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	return withTx(ctx, db.sql, fn)
}

func withTx(ctx context.Context, sdb *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := sdb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// NextSeq allocates the next global fact sequence number inside tx. Because
// write transactions are serialized, seq order equals commit order
// (data-model.md §1).
func NextSeq(ctx context.Context, tx *sql.Tx) (int64, error) {
	var seq int64
	err := tx.QueryRowContext(ctx, `UPDATE meta SET seq_counter = seq_counter + 1 WHERE id = 1 RETURNING seq_counter`).Scan(&seq)
	return seq, err
}

// Maintenance runs fn with append-only enforcement suspended inside tx. It is
// reserved for purge, retention cleanup and excerpt erasure.
func Maintenance(ctx context.Context, tx *sql.Tx, fn func() error) error {
	return withMaintenance(ctx, tx, fn)
}
