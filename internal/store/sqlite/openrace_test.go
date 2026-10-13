package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

// upgraderHelper migrates the database to schema v3 and exits without
// closing it, leaving the upgrade in the WAL (as codex-review-005 did).
func upgraderHelper() int {
	dir, err := datadir.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	migs, err := helperMigrations(false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	migs = append(migs, Migration{Version: 3, Name: "newer", SQL: `CREATE TABLE newer (id INTEGER PRIMARY KEY);`})
	// Signal, from inside the lock wait, that this process is blocked on the
	// open lock (codex-review-006 N3: a handshake instead of a fixed wait).
	if waiting := os.Getenv("UPGRADER_WAITING"); waiting != "" {
		lockWaitHook = func() { _ = os.WriteFile(waiting, nil, 0o600) }
	}
	if _, _, err := Open(context.Background(), dir, Options{Migrations: migs}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	os.Exit(0)
	return 0
}

// codex-review-005 B3, as reported: an older binary has decided the database
// is compatible but has not opened its read-write connection yet, and a newer
// binary tries to upgrade the database in that window. The upgrade must wait
// on the open lock until the older binary has finished opening; the older
// binary must not refuse after having recovered or checkpointed a newer
// database. The newer binary reports from inside the lock wait that it is
// blocked (codex-review-006 N3), so the test does not depend on timing.
func TestOpenAndConcurrentUpgradeDoNotInterleave(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(embedded(t), factsMigration(t))
	db, _ := open(t, dir, Options{Migrations: migs})
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "before")
	_ = db.Close()

	paused, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	openHook = func() { once.Do(func() { close(paused); <-resume }) }
	t.Cleanup(func() { openHook = nil })
	type result struct {
		db  *DB
		err error
	}
	oldDone := make(chan result, 1)
	go func() {
		d, _, err := Open(ctx, dir, Options{Migrations: slices.Clone(migs)})
		oldDone <- result{d, err}
	}()
	<-paused

	waiting := filepath.Join(testdir.New(t), "waiting")
	upgrader := testutil.HelperCommand(t, "upgrader", "UPGRADER_WAITING="+waiting)
	upgrader.Stderr = os.Stderr
	if err := upgrader.Start(); err != nil {
		t.Fatal(err)
	}
	upgraded := make(chan error, 1)
	go func() { upgraded <- upgrader.Wait() }()
	// Wait for the handshake. The upgrader finishing first would mean it got
	// past the open lock while the older binary held it.
	for {
		if _, err := os.Stat(waiting); err == nil {
			break
		}
		select {
		case err := <-upgraded:
			close(resume)
			<-oldDone
			t.Fatalf("the upgrade finished (%v) while the older binary was between its check and its connection", err)
		case <-time.After(10 * time.Millisecond):
		}
	}
	select {
	case err := <-upgraded:
		t.Fatalf("the upgrader stopped waiting on the lock (%v) before the older binary resumed", err)
	default:
	}
	close(resume)
	old := <-oldDone
	if old.err != nil {
		t.Fatalf("the older binary failed to open a compatible database: %v", old.err)
	}
	if err := <-upgraded; err != nil {
		t.Fatalf("upgrader: %v", err)
	}
	// The older binary's connection is still open; it must not write under
	// the newer schema (WriteTx re-check).
	assertWriteRefused(t, old.db, apperr.ExitEnvironment, "upgrade specmetry")
	// Closing it as the last connection checkpoints the newer binary's
	// committed WAL; the upgraded database must stay intact.
	if err := old.db.Close(); err != nil {
		t.Fatal(err)
	}
	v3 := append(slices.Clone(migs), Migration{Version: 3, Name: "newer", SQL: `CREATE TABLE newer (id INTEGER PRIMARY KEY);`})
	newer, _ := open(t, dir, Options{Migrations: v3})
	if v, _ := newer.SchemaVersion(ctx); v != 3 || countRows(t, newer, `SELECT count(*) FROM facts`) != 1 {
		t.Fatalf("after the older binary closed: schema v%d, %d facts", v, countRows(t, newer, `SELECT count(*) FROM facts`))
	}
}
