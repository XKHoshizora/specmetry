package sqlite

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
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
	if _, _, err := Open(context.Background(), dir, Options{Migrations: migs}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	os.Exit(0)
	return 0
}

// codex-review-005 B3, as reported: an older binary has decided the database
// is compatible but has not opened its read-write connection yet, and a newer
// binary upgrades the database in that window. Either the upgrade waits until
// the older binary has finished opening, or the older binary refuses without
// changing any file (MVP-STO-02); it must not refuse after it has already
// recovered or checkpointed the newer database.
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

	upgrader := testutil.HelperCommand(t, "upgrader")
	upgrader.Stderr = os.Stderr
	if err := upgrader.Start(); err != nil {
		t.Fatal(err)
	}
	upgraded := make(chan error, 1)
	go func() { upgraded <- upgrader.Wait() }()
	var upgradedFirst bool
	select {
	case err := <-upgraded:
		if err != nil {
			t.Fatalf("upgrader: %v", err)
		}
		upgradedFirst = true
	case <-time.After(2 * time.Second):
	}
	before := testutil.SnapshotTree(t, dir.Path)
	close(resume)
	old := <-oldDone

	switch {
	case old.err == nil && !upgradedFirst:
		// The upgrade waited for the older binary's open to finish.
		t.Log("the upgrade waited until the older binary had opened the database")
		if err := <-upgraded; err != nil {
			t.Fatalf("upgrader: %v", err)
		}
		// The older binary's connection is still open; it must not write
		// under the newer schema (WriteTx re-check).
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
	case old.err != nil:
		// A refusal must leave every file as it was when the older binary
		// resumed.
		testutil.AssertTreeUnchanged(t, dir.Path, before)
		if !upgradedFirst {
			<-upgraded
		}
	default:
		_ = old.db.Close()
		t.Fatal("the older binary opened a database that had been upgraded underneath it")
	}
}
