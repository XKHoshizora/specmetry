// Package sqlite is Specmetry's SQLite storage foundation: opening the
// database with WAL and a single writer connection, the fixed open order
// (permission check → version guard → tombstones → migration backup →
// migration), the global fact sequence, and append-only enforcement
// (architecture.md §11, §13).
package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
	sqlitedrv "modernc.org/sqlite" // also registers the "sqlite" driver

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
	// LockTimeout bounds the wait for the open lock, which another process
	// holds while it opens or migrates the database.
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
	// target and binVersion are the schema version and binary version this
	// process was opened with; WriteTx re-checks them under the write lock.
	target     int
	binVersion string
}

// OpenResult reports what Open did.
type OpenResult struct {
	FromVersion       int
	ToVersion         int
	BackupPath        string
	TombstoneRowsGone int64
}

// Open opens (creating if necessary) the database in dir and brings it to
// the latest schema. Order (architecture.md §11.1, §11.2):
//  1. data directory: create it private, or verify that the existing one,
//     everything in it and the directories on its path are safe — before
//     anything is written;
//  2. take the open lock, held until Open returns;
//  3. version guard, decided by a read-only connection that changes no file,
//     before any connection that could;
//  4. apply tombstones;
//  5. if migration is needed: VACUUM INTO backup, then all pending
//     migrations in one BEGIN EXCLUSIVE transaction.
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
	// From here until the database is open and migrated, no other process
	// can migrate it: the compatibility decision cannot go stale before the
	// read-write connection exists (codex-review-005 B3).
	lock, err := acquireOpenLock(ctx, dir.Join(datadir.MigrateLockFile), opts.LockTimeout)
	if err != nil {
		return nil, res, err
	}
	defer lock.release()

	dbPath := dir.Join(datadir.DatabaseFile)
	// Decided by a read-only SQLite connection that changes no file, so a
	// database this binary must not open is refused untouched (MVP-STO-02).
	ro, err := readMetaReadOnly(ctx, dbPath, false)
	if err != nil {
		var u *uninspectableError
		if errors.As(err, &u) {
			return nil, res, apperr.Newf(apperr.Environment,
				"cannot confirm that this specmetry binary may open the database without modifying it: %s", u.reason)
		}
		return nil, res, apperr.Wrap(apperr.Environment, err, "cannot read the database version")
	}
	if ro.exists {
		if err := guardVersion(ro.version, target, ro.minBin, binVersion); err != nil {
			return nil, res, err
		}
	}
	if openHook != nil {
		openHook()
	}
	// Validate tombstones before anything is written, so that a bad file
	// leaves the data directory untouched.
	tombstones, err := readTombstones(dir.Join(datadir.TombstonesFile))
	if err != nil {
		return nil, res, apperr.Wrap(apperr.Environment, err, "cannot apply tombstones")
	}
	if err := precreate(dbPath); err != nil {
		return nil, res, apperr.Wrap(apperr.Environment, err, "cannot create the database file")
	}
	connector, err := sqlitedrv.NewConnector(dsn(dbPath))
	if err != nil {
		return nil, res, apperr.Wrap(apperr.Environment, err, "cannot open the database")
	}
	sdb := sql.OpenDB(persistentWAL{connector})
	// A single connection serializes all writes from this process
	// (architecture.md §13); SQLite locking serializes across processes.
	sdb.SetMaxOpenConns(1)
	db := &DB{sql: sdb, dir: dir, path: dbPath, target: target, binVersion: binVersion}
	fail := func(err error) (*DB, OpenResult, error) {
		_ = sdb.Close()
		return nil, res, err
	}

	if err := checkConnection(ctx, sdb); err != nil {
		return fail(err)
	}
	if err := checkPersistentWAL(ctx, sdb); err != nil {
		return fail(err)
	}
	// Under the lock this sees what the read-only check saw; it is repeated
	// on the connection that will write.
	current, minBin, err := readMeta(ctx, sdb)
	if err != nil {
		return fail(apperr.Wrap(apperr.Environment, err, "cannot read the database version"))
	}
	if err := guardVersion(current, target, minBin, binVersion); err != nil {
		return fail(err)
	}
	if res.TombstoneRowsGone, err = applyTombstones(ctx, sdb, dir, tombstones); err != nil {
		return fail(apperr.Wrap(apperr.Environment, err, "cannot apply tombstones"))
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

// persistentWAL sets SQLITE_FCNTL_PERSIST_WAL on every connection the pool
// opens: when the last connection closes, SQLite checkpoints the WAL and
// truncates it (journal_size_limit 0) but keeps the -wal and -shm files. A
// closed specmetry database therefore stays in a state that a read-only
// connection can read without creating files (readonly.go).
type persistentWAL struct{ driver.Connector }

func (c persistentWAL) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	fc, ok := conn.(sqlitedrv.FileControl)
	if !ok {
		_ = conn.Close()
		return nil, errors.New("the SQLite driver does not support file control")
	}
	if _, err := fc.FileControlPersistWAL("main", 1); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

// openHook, if set, runs right after Open decided that the database is
// compatible and before it opens a read-write connection (tests use it to
// let another process act in between).
var openHook func()

// guardVersion refuses databases written by a newer binary (TD-16). A
// minimum binary version that cannot be compared is refused too: the
// database asks for something this binary cannot confirm.
func guardVersion(current, target int, minBin, binVersion string) error {
	if current > target {
		return apperr.Newf(apperr.Environment,
			"the database uses schema v%d but this specmetry binary only supports up to v%d; upgrade specmetry", current, target)
	}
	if minBin == "" {
		return nil
	}
	if !semver.IsValid("v" + minBin) {
		return apperr.Newf(apperr.Environment,
			"the database requires specmetry version %q, which this binary cannot interpret; upgrade specmetry", minBin)
	}
	if !semver.IsValid("v" + binVersion) {
		return apperr.Newf(apperr.Environment,
			"the database requires specmetry %s or newer, but this binary's version %q cannot be compared with it", minBin, binVersion)
	}
	if semver.Compare("v"+binVersion, "v"+minBin) < 0 {
		return apperr.Newf(apperr.Environment,
			"the database requires specmetry %s or newer (this binary is %s); upgrade specmetry", minBin, binVersion)
	}
	return nil
}

// checkConnection verifies the per-connection settings that protect the
// data: with recursive_triggers off, REPLACE conflict resolution deletes rows
// without firing the append-only delete trigger (codex-review-004 B4).
func checkConnection(ctx context.Context, q queryer) error {
	var on int
	if err := q.QueryRowContext(ctx, `PRAGMA recursive_triggers`).Scan(&on); err != nil {
		return apperr.Wrap(apperr.Environment, err, "cannot read the database connection settings")
	}
	if on != 1 {
		return apperr.New(apperr.Internal,
			"the database connection has recursive_triggers off, so append-only tables would not be protected against REPLACE; refusing to write")
	}
	return nil
}

// checkPersistentWAL verifies that the pool's connections keep the WAL
// files on close (persistentWAL).
func checkPersistentWAL(ctx context.Context, sdb *sql.DB) error {
	conn, err := sdb.Conn(ctx)
	if err != nil {
		return apperr.Wrap(apperr.Environment, err, "cannot open the database")
	}
	defer func() { _ = conn.Close() }()
	return conn.Raw(func(dc any) error {
		fc, ok := dc.(sqlitedrv.FileControl)
		if !ok {
			return apperr.New(apperr.Internal, "the SQLite driver does not support file control")
		}
		mode, err := fc.FileControlPersistWAL("main", -1)
		if err != nil {
			return apperr.Wrap(apperr.Environment, err, "cannot read the database connection settings")
		}
		if mode != 1 {
			return apperr.New(apperr.Internal, "the database connection does not keep its WAL files on close")
		}
		return nil
	})
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
	// The migration lock coordinates specmetry processes; this re-check
	// under the exclusive lock also covers anything that does not use it.
	if v, _, err := readMeta(ctx, conn); err != nil {
		return rollback(err)
	} else if v != current {
		return rollback(fmt.Errorf("the database changed to schema v%d while waiting to migrate it", v))
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

// dsn is used only after the version guard has passed: journal_mode(WAL)
// rewrites the header of a rollback-journal database (such as a restored
// VACUUM INTO backup). The driver applies every _pragma to each new
// connection, so connections the pool opens later get them too.
func dsn(path string) string {
	return "file:" + escapePath(path) +
		"?_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		// REPLACE conflict resolution fires delete triggers only with
		// recursive_triggers on (https://sqlite.org/lang_conflict.html).
		"&_pragma=recursive_triggers(1)" +
		// With persistentWAL, truncate the WAL to zero bytes when the last
		// connection checkpoints it on close, so no old page images stay in it.
		"&_pragma=journal_size_limit(0)" +
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
// first statement (BEGIN IMMEDIATE). Before fn runs, and under that lock, it
// re-checks that this binary may still write: another process may have
// migrated the schema or raised min_binary_version since Open, or the
// connection may have lost its append-only protection (codex-review-004 B3,
// B4). If not, fn is not called and nothing is written. fn's error rolls the
// transaction back.
func (db *DB) WriteTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	return withTx(ctx, db.sql, func(tx *sql.Tx) error {
		if err := db.checkWritable(ctx, tx); err != nil {
			return err
		}
		return fn(tx)
	})
}

func (db *DB) checkWritable(ctx context.Context, tx *sql.Tx) error {
	current, minBin, err := readMeta(ctx, tx)
	if err != nil {
		return apperr.Wrap(apperr.Environment, err, "cannot read the database version")
	}
	if err := guardVersion(current, db.target, minBin, db.binVersion); err != nil {
		return err
	}
	if current != db.target {
		return apperr.Newf(apperr.Environment,
			"the database schema changed from v%d to v%d since it was opened; run the command again", db.target, current)
	}
	return checkConnection(ctx, tx)
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
