package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// crashWriterHelper opens the database with the facts table, writes facts and
// exits without closing it, leaving an unmerged WAL and its -shm behind as a
// crash would.
func crashWriterHelper() int {
	dir, err := datadir.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	migs, err := helperMigrations(os.Getenv("CRASH_SLOW") == "1")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if n, _ := strconv.Atoi(os.Getenv("CRASH_SCHEMA")); n > 0 {
		migs = migs[:n]
	}
	db, _, err := Open(context.Background(), dir, Options{Migrations: migs})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	n, _ := strconv.Atoi(os.Getenv("CRASH_FACTS"))
	for i := 0; i < n; i++ {
		if _, err := writeFact(context.Background(), db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", os.Getenv("CRASH_PAYLOAD")); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	os.Exit(0) // no Close: the WAL is not checkpointed
	return 0
}

// liveWriterHelper writes facts, signals readiness and keeps the database
// open until told to stop.
func liveWriterHelper() int {
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
	db, _, err := Open(context.Background(), dir, Options{Migrations: migs})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = db.Close() }()
	for i := 0; i < 5; i++ {
		if _, err := writeFact(context.Background(), db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "live"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	if err := os.WriteFile(os.Getenv("LIVE_READY"), nil, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for {
		if _, err := os.Stat(os.Getenv("LIVE_STOP")); err == nil {
			return 0
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// crashWithWAL runs crashWriterHelper and checks that it left a WAL behind.
func crashWithWAL(t *testing.T, env ...string) {
	t.Helper()
	if out, err := testutil.HelperCommand(t, "crashwriter", env...).CombinedOutput(); err != nil {
		t.Fatalf("crash writer: %v\n%s", err, out)
	}
	dir, err := datadir.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		fi, err := os.Stat(dir.Join(datadir.DatabaseFile) + suffix)
		if err != nil || fi.Size() == 0 {
			t.Fatalf("crash writer left no %s file (%v)", suffix, err)
		}
	}
}

// startLiveWriter runs liveWriterHelper until the test ends.
func startLiveWriter(t *testing.T) {
	t.Helper()
	ready := filepath.Join(t.TempDir(), "ready")
	stop := filepath.Join(t.TempDir(), "stop")
	cmd := testutil.HelperCommand(t, "livewriter", "LIVE_READY="+ready, "LIVE_STOP="+stop)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.WriteFile(stop, nil, 0o600)
		_ = cmd.Wait()
	})
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("live writer did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// openRefused opens dir and requires an environment error (exit 6) that
// mentions want.
func openRefused(t *testing.T, dir datadir.Dir, opts Options, want string) {
	t.Helper()
	db, _, err := Open(ctx, dir, opts)
	if err == nil {
		_ = db.Close()
		t.Fatalf("Open succeeded, want a refusal mentioning %q", want)
	}
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error()+detailsOf(err), want) {
		t.Fatalf("err = %v (details: %s), want exit 6 mentioning %q", err, detailsOf(err), want)
	}
}

func detailsOf(err error) string {
	if err == nil {
		return ""
	}
	var parts []string
	for _, d := range apperr.From(err).Details {
		parts = append(parts, d.Problem)
	}
	return strings.Join(parts, "; ")
}

// codex-review-004 B1: an existing data directory that is not private is
// refused before anything is written into it.
func TestOpenRefusesInsecureExistingDirectory(t *testing.T) {
	dir := testutil.TempHome(t)
	if err := dir.Ensure(); err != nil {
		t.Fatal(err)
	}
	testutil.LoosenPermissions(t, dir.Path)
	before := testutil.SnapshotTree(t, dir.Path)
	openRefused(t, dir, Options{}, "not private")
	testutil.AssertTreeUnchanged(t, dir.Path, before)
}

// codex-review-004 B1: an existing database file that is not private is
// refused; neither it nor the directory changes.
func TestOpenRefusesInsecureExistingDatabase(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{Migrations: append(embedded(t), factsMigration(t))})
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "secret")
	_ = db.Close()
	testutil.LoosenPermissions(t, dir.Join(datadir.DatabaseFile))
	before := testutil.SnapshotTree(t, dir.Path)
	openRefused(t, dir, Options{Migrations: append(embedded(t), factsMigration(t))}, datadir.DatabaseFile)
	testutil.AssertTreeUnchanged(t, dir.Path, before)
}

// codex-review-004 B1: a WAL left by a crash with loosened permissions is
// refused as well.
func TestOpenRefusesInsecureLeftoverWAL(t *testing.T) {
	dir := testutil.TempHome(t)
	crashWithWAL(t, "CRASH_FACTS=3", "CRASH_PAYLOAD=secret")
	testutil.LoosenPermissions(t, dir.Join(datadir.DatabaseFile)+"-wal")
	before := testutil.SnapshotTree(t, dir.Path)
	openRefused(t, dir, Options{Migrations: append(embedded(t), factsMigration(t))}, datadir.DatabaseFile+"-wal")
	testutil.AssertTreeUnchanged(t, dir.Path, before)
}

// codex-review-004 B1: a symbolic link inside the data directory could send
// database writes anywhere; it is refused.
func TestOpenRefusesSymlinkedDatabase(t *testing.T) {
	dir := testutil.TempHome(t)
	if err := dir.Ensure(); err != nil {
		t.Fatal(err)
	}
	elsewhere := filepath.Join(t.TempDir(), "elsewhere.db")
	if err := os.WriteFile(elsewhere, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, dir.Join(datadir.DatabaseFile)); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("creating symbolic links needs Developer Mode or administrator rights:", err)
		}
		t.Fatal(err)
	}
	before := testutil.SnapshotTree(t, dir.Path)
	openRefused(t, dir, Options{}, "symbolic link")
	testutil.AssertTreeUnchanged(t, dir.Path, before)
	if fi, err := os.Stat(elsewhere); err != nil || fi.Size() != 0 {
		t.Fatalf("link target was written: %v %v", fi, err)
	}
}
