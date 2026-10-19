package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

func TestMain(m *testing.M) {
	testutil.DispatchHelper(map[string]func() int{
		"seqwriter":   seqWriterHelper,
		"migrator":    migratorHelper,
		"crashwriter": crashWriterHelper,
		"livewriter":  liveWriterHelper,
		"victimopen":  victimOpenHelper,
		"victimclean": victimCleanHelper,
		"attacker":    attackerHelper,
		"uncommitted": uncommittedWriterHelper,
		"upgrader":    upgraderHelper,
		"lockholder":  lockHolderHelper,
		"plainclose":  plainCloseHelper,
		"m14open":     m14OpenHelper,
		"m14probe":    m14ProbeHelper,
	})
	os.Exit(m.Run())
}

var ctx = context.Background()

// foundation returns the storage foundation migration (0001) only. The
// synthetic migrations of these tests are numbered from 2 on top of it and
// create tables of their own, so they must not follow the business tables
// that later embedded migrations add.
func foundation(t *testing.T) []Migration {
	t.Helper()
	ms, err := Migrations()
	if err != nil {
		t.Fatal(err)
	}
	return ms[:1]
}

// factsMigration creates an append-only table with a project_id column, the
// shape of future fact tables.
func factsMigration(t *testing.T) Migration {
	t.Helper()
	m, err := newFactsMigration()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func newFactsMigration() (Migration, error) {
	ddl, err := AppendOnlyTriggers("facts")
	return Migration{Version: 2, Name: "facts", SQL: `
CREATE TABLE facts (id INTEGER PRIMARY KEY, seq INTEGER NOT NULL UNIQUE, project_id TEXT NOT NULL, payload TEXT NOT NULL);
` + ddl}, err
}

// slowMigration is deliberately slow to widen the window in which two
// processes race to migrate.
var slowMigration = Migration{Version: 3, Name: "slow", SQL: `
CREATE TABLE numbers (n INTEGER PRIMARY KEY);
WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x + 1 FROM c WHERE x < 200000)
INSERT INTO numbers SELECT x FROM c;`}

// helperMigrations returns the embedded migrations plus the facts table, and
// the slow migration if requested; helper processes cannot use *testing.T.
func helperMigrations(slow bool) ([]Migration, error) {
	migs, err := Migrations()
	if err != nil {
		return nil, err
	}
	migs = migs[:1] // the foundation, as foundation(t) returns
	facts, err := newFactsMigration()
	if err != nil {
		return nil, err
	}
	migs = append(migs, facts)
	if slow {
		migs = append(migs, slowMigration)
	}
	return migs, nil
}

func open(t *testing.T, dir datadir.Dir, opts Options) (*DB, OpenResult) {
	t.Helper()
	db, res, err := Open(ctx, dir, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, res
}

func insertFact(t *testing.T, db *DB, project, payload string) int64 {
	t.Helper()
	seq, err := writeFact(ctx, db, project, payload)
	if err != nil {
		t.Fatal(err)
	}
	return seq
}

// writeFact is safe to call from any goroutine or helper process.
func writeFact(ctx context.Context, db *DB, project, payload string) (int64, error) {
	var seq int64
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		var err error
		if seq, err = NextSeq(ctx, tx); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO facts (seq, project_id, payload) VALUES (?, ?, ?)`, seq, project, payload)
		return err
	})
	return seq, err
}

func countRows(t *testing.T, db *DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.sql.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func backups(t *testing.T, dir datadir.Dir) []string {
	t.Helper()
	entries, err := os.ReadDir(dir.Join(datadir.BackupsDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, filepath.Join(dir.Join(datadir.BackupsDir), e.Name()))
	}
	return out
}

func TestEmbeddedMigrations(t *testing.T) {
	ms, err := Migrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 || ms[0].Version != 1 || SchemaVersion() != len(ms) {
		t.Fatalf("embedded migrations = %+v", ms)
	}
}

func TestOpenCreatesEmptySchemaV1(t *testing.T) {
	dir := testutil.TempHome(t)
	db, res := open(t, dir, Options{})
	if res.FromVersion != 0 || res.ToVersion != SchemaVersion() || res.BackupPath != "" {
		t.Fatalf("open result = %+v", res)
	}
	if v, err := db.SchemaVersion(ctx); err != nil || v != SchemaVersion() {
		t.Fatalf("schema version = %d, %v", v, err)
	}
	var mode string
	if err := db.sql.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("journal_mode = %q, %v", mode, err)
	}
	if n := countRows(t, db, `SELECT seq_counter FROM meta WHERE id = 1`); n != 0 {
		t.Fatalf("seq_counter = %d", n)
	}
	_ = db.Close()
	// Re-opening an up-to-date database neither migrates nor backs up.
	_, res = open(t, dir, Options{})
	if res.FromVersion != res.ToVersion || res.BackupPath != "" || len(backups(t, dir)) != 0 {
		t.Fatalf("second open = %+v", res)
	}
}

// Paths with spaces and CJK characters must work (NFR-10).
func TestOpenPathWithSpacesAndCJK(t *testing.T) {
	dir := datadir.Dir{Path: filepath.Join(testdir.New(t), "my data", "规格 数据")}
	db, _ := open(t, dir, Options{})
	if _, err := os.Stat(db.Path()); err != nil {
		t.Fatal(err)
	}
}

// MVP-PRV-05 / MVP-PRV-06: every file the store creates — database, WAL,
// SHM, backups, lock — is private to the current user.
func TestStoreFilesArePrivate(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(foundation(t), factsMigration(t))
	db, _ := open(t, dir, Options{Migrations: migs[:1]})
	_ = db.Close()
	db, res := open(t, dir, Options{Migrations: migs})
	if res.BackupPath == "" {
		t.Fatal("expected a pre-migration backup")
	}
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "x")
	problems, err := datadir.Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("permission problems: %v", problems)
	}
}

// MVP-STO-01 (part): a migration of an existing database first writes a
// consistent, openable VACUUM INTO backup.
func TestMigrationWritesConsistentBackup(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(foundation(t), factsMigration(t))
	db, _ := open(t, dir, Options{Migrations: migs})
	for i := 0; i < 50; i++ {
		insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", fmt.Sprint(i))
	}
	_ = db.Close()

	v3 := Migration{Version: 3, Name: "extra", SQL: `CREATE TABLE extra (id INTEGER PRIMARY KEY);`}
	db, res := open(t, dir, Options{Migrations: append(migs, v3)})
	if res.FromVersion != 2 || res.ToVersion != 3 || res.BackupPath == "" {
		t.Fatalf("open result = %+v", res)
	}
	_ = db.Close()

	bak, err := sql.Open("sqlite", dsn(res.BackupPath))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bak.Close() }()
	var rows, version int
	if err := bak.QueryRowContext(ctx, `SELECT count(*) FROM facts`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if err := bak.QueryRowContext(ctx, `SELECT schema_version FROM meta`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if rows != 50 || version != 2 {
		t.Fatalf("backup has %d rows at schema v%d, want 50 at v2", rows, version)
	}
}

// MVP-STO-01: two processes migrating at once — one migrates, the other waits
// and then finds the database up to date; exactly one backup is written, and
// it holds the pre-migration content.
func TestConcurrentMigrationIsCoordinated(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{Migrations: append(foundation(t), factsMigration(t))})
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "before")
	_ = db.Close()

	type outcome struct {
		res OpenResult
		err error
	}
	outcomes := make([]chan outcome, 2)
	for i := range outcomes {
		outcomes[i] = make(chan outcome, 1)
		cmd := testutil.HelperCommand(t, "migrator")
		go func(ch chan outcome) {
			var o outcome
			out, err := cmd.Output()
			if err != nil {
				o.err = fmt.Errorf("%w: %s", err, out)
			} else {
				o.err = json.Unmarshal(out, &o.res)
			}
			ch <- o
		}(outcomes[i])
	}
	var migrated []OpenResult
	for i, ch := range outcomes {
		o := <-ch
		if o.err != nil {
			t.Fatalf("process %d: %v", i, o.err)
		}
		t.Logf("process %d: %+v", i, o.res)
		switch {
		case o.res.BackupPath != "" && o.res.FromVersion == 2 && o.res.ToVersion == 3:
			migrated = append(migrated, o.res)
		case o.res.BackupPath == "" && o.res.FromVersion == 3 && o.res.ToVersion == 3:
			// Waited for the other process, then found nothing to do.
		default:
			t.Fatalf("process %d: unexpected result %+v", i, o.res)
		}
	}
	if len(migrated) != 1 || len(backups(t, dir)) != 1 {
		t.Fatalf("migrations performed = %d, backups = %v", len(migrated), backups(t, dir))
	}

	bak, err := sql.Open("sqlite", dsn(migrated[0].BackupPath))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bak.Close() }()
	var payload string
	var version int
	if err := bak.QueryRowContext(ctx, `SELECT payload FROM facts`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if err := bak.QueryRowContext(ctx, `SELECT schema_version FROM meta`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if payload != "before" || version != 2 {
		t.Fatalf("backup holds %q at schema v%d, want \"before\" at v2", payload, version)
	}
}

// migratorHelper opens the database with the slow migration pending and
// prints the OpenResult as JSON.
func migratorHelper() int {
	dir, err := datadir.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	migs, err := helperMigrations(true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	db, res, err := Open(context.Background(), dir, Options{Migrations: migs})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_ = db.Close()
	if err := json.NewEncoder(os.Stdout).Encode(res); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// lockHolderHelper takes the open lock and keeps it until $LOCK_STOP exists,
// or exits while holding it if $LOCK_CRASH is set.
//
// The lock lasts as long as its file stays open. A dropped *openLock leaves
// the *os.File unreachable, and its finalizer closes the file at the next
// garbage collection, releasing the lock: TestOpenLockTimeout then failed
// only when a collection happened to run while it waited (seen in CI on
// Linux). The helper keeps the lock referenced, and collects garbage on
// every turn of its wait so that a dropped reference fails at once.
func lockHolderHelper() int {
	dir, err := datadir.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := dir.Ensure(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	lock, err := acquireOpenLock(context.Background(), dir.Join(datadir.MigrateLockFile), time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if os.Getenv("LOCK_CRASH") != "" {
		os.Exit(0) // the operating system releases the lock
	}
	defer lock.release()
	if err := os.WriteFile(os.Getenv("LOCK_READY"), nil, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for {
		if _, err := os.Stat(os.Getenv("LOCK_STOP")); err == nil {
			return 0
		}
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
}

// MVP-STO-01: while another process holds the open lock (it is opening or
// migrating the database), a process that wants to open the database waits
// and then fails with an environment error (exit 6), touching nothing.
func TestOpenLockTimeout(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{Migrations: foundation(t)})
	_ = db.Close()
	ready := filepath.Join(testdir.New(t), "ready")
	stop := filepath.Join(testdir.New(t), "stop")
	holder := testutil.HelperCommand(t, "lockholder", "LOCK_READY="+ready, "LOCK_STOP="+stop)
	holder.Stderr = os.Stderr
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.WriteFile(stop, nil, 0o600); _ = holder.Wait() })
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if in, err := Inspect(ctx, dir); err != nil || !in.Locked {
		t.Fatalf("Inspect = %+v, %v; want the lock reported as held", in, err)
	}
	migs := append(foundation(t), factsMigration(t))
	before := testutil.SnapshotTree(t, dir.Path)
	_, _, err := Open(ctx, dir, Options{Migrations: migs, LockTimeout: 200 * time.Millisecond})
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "another specmetry process") {
		t.Fatalf("err = %v", err)
	}
	testutil.AssertTreeUnchanged(t, dir.Path, before)

	_ = os.WriteFile(stop, nil, 0o600)
	if err := holder.Wait(); err != nil {
		t.Fatal(err)
	}
	db, _ = open(t, dir, Options{Migrations: migs})
	if v, _ := db.SchemaVersion(ctx); v != 2 {
		t.Fatalf("schema v%d after the lock was released", v)
	}
}

// A process that dies while holding the lock does not leave it held, and a
// lock file left behind blocks nobody.
func TestCrashedLockHolderDoesNotBlock(t *testing.T) {
	dir := testutil.TempHome(t)
	if out, err := testutil.HelperCommand(t, "lockholder", "LOCK_CRASH=1").CombinedOutput(); err != nil {
		t.Fatalf("lock holder: %v\n%s", err, out)
	}
	if _, err := os.Stat(dir.Join(datadir.MigrateLockFile)); err != nil {
		t.Fatalf("lock file: %v", err)
	}
	if in, err := Inspect(ctx, dir); err != nil || in.Locked {
		t.Fatalf("Inspect = %+v, %v; want the lock free", in, err)
	}
	db, _ := open(t, dir, Options{LockTimeout: 200 * time.Millisecond})
	_ = db.Close()
}

// MVP-STO-02: an older binary refuses a newer schema with exit 6 and writes
// nothing.
func TestOlderBinaryRefusesNewerSchema(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{Migrations: append(foundation(t), factsMigration(t))})
	_ = db.Close()
	before := testutil.SnapshotTree(t, dir.Path)
	openRefused(t, dir, Options{Migrations: foundation(t)}, "upgrade specmetry")
	testutil.AssertTreeUnchanged(t, dir.Path, before)
}

func TestMinBinaryVersionGuard(t *testing.T) {
	dir := testutil.TempHome(t)
	requires := Migration{Version: 2, Name: "requires_new", SQL: `UPDATE meta SET min_binary_version = '9.9.9' WHERE id = 1;`}
	migs := append(foundation(t), requires)
	db, _ := open(t, dir, Options{Migrations: migs, BinaryVersion: "9.9.9"})
	_ = db.Close()
	before := testutil.SnapshotTree(t, dir.Path)
	openRefused(t, dir, Options{Migrations: migs, BinaryVersion: "0.1.0"}, "9.9.9")
	testutil.AssertTreeUnchanged(t, dir.Path, before)
	db, _ = open(t, dir, Options{Migrations: migs, BinaryVersion: "10.0.0"})
	_ = db.Close()
}

// MVP-STO-03: a migration that fails part-way rolls back completely.
func TestFailingMigrationRollsBack(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{Migrations: foundation(t)})
	_ = db.Close()
	bad := Migration{Version: 2, Name: "bad", SQL: `
CREATE TABLE half_done (id INTEGER PRIMARY KEY);
INSERT INTO table_that_does_not_exist VALUES (1);`}
	_, _, err := Open(ctx, dir, Options{Migrations: append(foundation(t), bad)})
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("err = %v", err)
	}
	db, _ = open(t, dir, Options{Migrations: foundation(t)})
	if v, _ := db.SchemaVersion(ctx); v != 1 {
		t.Fatalf("schema v%d after failed migration, want v1", v)
	}
	if n := countRows(t, db, `SELECT count(*) FROM sqlite_master WHERE name = 'half_done'`); n != 0 {
		t.Fatal("partial migration was not rolled back")
	}
	if held, err := lockHeld(dir.Join(datadir.MigrateLockFile)); err != nil || held {
		t.Fatalf("open lock still held after a failed migration (%v)", err)
	}
}

func TestBackupRetentionKeepsNewest(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := foundation(t)
	clock := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	db, _ := open(t, dir, Options{Migrations: migs})
	_ = db.Close()
	var last string
	for v := 2; v <= 6; v++ {
		migs = append(migs, Migration{Version: v, Name: "noop", SQL: `SELECT 1;`})
		clock = clock.Add(time.Minute)
		now := clock
		db, res := open(t, dir, Options{Migrations: migs, Now: func() time.Time { return now }})
		_ = db.Close()
		last = res.BackupPath
	}
	got := backups(t, dir)
	if len(got) != 3 {
		t.Fatalf("kept %d backups, want 3: %v", len(got), got)
	}
	if got[len(got)-1] != last {
		t.Fatalf("newest backup %s not kept: %v", last, got)
	}
}

// MVP-STO-04: concurrent writers from separate connections get strictly
// increasing, gap-free seq values.
func TestSeqIsUniqueAcrossConnections(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(foundation(t), factsMigration(t))
	a, _ := open(t, dir, Options{Migrations: migs})
	b, _ := open(t, dir, Options{Migrations: migs})
	var wg sync.WaitGroup
	errs := make(chan error, 300)
	for _, db := range []*DB{a, b} {
		wg.Add(1)
		go func(db *DB) {
			defer wg.Done()
			for i := 0; i < 150; i++ {
				if _, err := writeFact(ctx, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "w"); err != nil {
					errs <- err
				}
			}
		}(db)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	assertContiguousSeq(t, a, 300)
}

// MVP-STO-04: the same across operating-system processes.
func TestSeqIsUniqueAcrossProcesses(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(foundation(t), factsMigration(t))
	db, _ := open(t, dir, Options{Migrations: migs})
	const perProcess = 100
	procs := make([]chan error, 3)
	for i := range procs {
		procs[i] = make(chan error, 1)
		cmd := testutil.HelperCommand(t, "seqwriter", "SEQWRITER_N="+strconv.Itoa(perProcess))
		go func(ch chan error) {
			out, err := cmd.CombinedOutput()
			if err != nil {
				err = fmt.Errorf("%w: %s", err, out)
			}
			ch <- err
		}(procs[i])
	}
	for _, ch := range procs {
		if err := <-ch; err != nil {
			t.Fatal(err)
		}
	}
	assertContiguousSeq(t, db, len(procs)*perProcess)
}

func seqWriterHelper() int {
	dir, err := datadir.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	n, _ := strconv.Atoi(os.Getenv("SEQWRITER_N"))
	migs, err := helperMigrations(false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	db, _, err := Open(context.Background(), dir, Options{Migrations: migs})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = db.Close() }()
	for i := 0; i < n; i++ {
		if _, err := writeFact(context.Background(), db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "p"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	return 0
}

func assertContiguousSeq(t *testing.T, db *DB, want int) {
	t.Helper()
	var n, distinct, minSeq, maxSeq int
	err := db.sql.QueryRowContext(ctx, `SELECT count(*), count(DISTINCT seq), min(seq), max(seq) FROM facts`).Scan(&n, &distinct, &minSeq, &maxSeq)
	if err != nil {
		t.Fatal(err)
	}
	if n != want || distinct != want || minSeq != 1 || maxSeq != want {
		t.Fatalf("rows=%d distinct=%d min=%d max=%d, want %d contiguous", n, distinct, minSeq, maxSeq, want)
	}
	if got := countRows(t, db, `SELECT seq_counter FROM meta`); got != want {
		t.Fatalf("seq_counter = %d, want %d", got, want)
	}
}

func TestAppendOnlyTriggersAndMaintenance(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{Migrations: append(foundation(t), factsMigration(t))})
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "keep")
	for _, stmt := range []string{`UPDATE facts SET payload = 'changed'`, `DELETE FROM facts`} {
		if _, err := db.sql.ExecContext(ctx, stmt); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Fatalf("%s: err = %v", stmt, err)
		}
	}
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		return Maintenance(ctx, tx, func() error {
			_, err := tx.ExecContext(ctx, `DELETE FROM facts`)
			return err
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, db, `SELECT count(*) FROM facts`); n != 0 {
		t.Fatalf("maintenance delete left %d rows", n)
	}
	if n := countRows(t, db, `SELECT active FROM maintenance`); n != 0 {
		t.Fatal("maintenance switch left on")
	}
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "again")
	if _, err := db.sql.ExecContext(ctx, `DELETE FROM facts`); err == nil {
		t.Fatal("append-only enforcement not restored after maintenance")
	}
	if _, err := AppendOnlyTriggers("Bad-Name"); err == nil {
		t.Fatal("invalid table name accepted")
	}
}

// Tombstones run before any migration backup, so a restored old backup
// cannot bring a purged project back, not even inside the new backup
// (architecture.md §11.2/§11.4; foundation for CX-11).
func TestTombstonesAppliedBeforeMigrationBackup(t *testing.T) {
	dir := testutil.TempHome(t)
	const purged, kept = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "01J9Z3K5Q8R2T4V6W8X0Y2Z4A7"
	canaryPurged, canaryKept := testutil.NewCanary(t), testutil.NewCanary(t)
	migs := append(foundation(t), factsMigration(t))
	db, _ := open(t, dir, Options{Migrations: migs})
	insertFact(t, db, purged, canaryPurged)
	insertFact(t, db, kept, canaryKept)
	_ = db.Close()

	b, _ := json.Marshal([]Tombstone{{ProjectID: purged, PurgedAt: 1}})
	if err := os.WriteFile(dir.Join(datadir.TombstonesFile), b, 0o600); err != nil {
		t.Fatal(err)
	}
	v3 := Migration{Version: 3, Name: "next", SQL: `CREATE TABLE next_table (id INTEGER PRIMARY KEY);`}
	db, res := open(t, dir, Options{Migrations: append(migs, v3)})
	if res.TombstoneRowsGone != 1 || res.BackupPath == "" {
		t.Fatalf("open result = %+v", res)
	}
	if n := countRows(t, db, `SELECT count(*) FROM facts WHERE project_id = ?`, purged); n != 0 {
		t.Fatalf("%d purged rows remain", n)
	}
	if n := countRows(t, db, `SELECT count(*) FROM facts WHERE project_id = ?`, kept); n != 1 {
		t.Fatal("rows of another project were deleted")
	}
	_ = db.Close()
	if hits := testutil.FindInTree(t, dir.Path, canaryPurged); len(hits) != 0 {
		t.Fatalf("purged data still on disk: %v", hits)
	}
	if hits := testutil.FindInTree(t, dir.Path, canaryKept); len(hits) == 0 {
		t.Fatal("kept project's data missing")
	}
}

// codex-review-004 N2: reopen with the same migrations, so the error comes
// from tombstone validation rather than from the version guard.
func TestInvalidTombstonesFileIsAnError(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(foundation(t), factsMigration(t))
	db, _ := open(t, dir, Options{Migrations: migs})
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "kept")
	_ = db.Close()
	if err := os.WriteFile(dir.Join(datadir.TombstonesFile), []byte(`[{"project_id":"not-a-ulid","purged_at":1}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	before := testutil.SnapshotTree(t, dir.Path)
	_, _, err := Open(ctx, dir, Options{Migrations: migs})
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), `invalid project_id "not-a-ulid"`) {
		t.Fatalf("err = %v", err)
	}
	testutil.AssertTreeUnchanged(t, dir.Path, before)
}

func TestMigrationsMustBeContiguous(t *testing.T) {
	dir := testutil.TempHome(t)
	gap := []Migration{foundation(t)[0], {Version: 3, Name: "gap", SQL: `SELECT 1;`}}
	if _, _, err := Open(ctx, dir, Options{Migrations: gap}); apperr.ExitCode(err) != apperr.ExitInternal {
		t.Fatalf("err = %v", err)
	}
}
