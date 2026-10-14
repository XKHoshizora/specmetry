package datadir

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

// Another process may be creating the database at the moment a check
// runs: SQLite's transient files (the rollback journal that switching a
// new database to WAL uses) appear and disappear. An entry gone by the
// time it is inspected cannot expose anything; the check skips it instead
// of failing (seen as an exit 6 of one of several concurrent first init
// runs). An entry that stays is still checked.
func TestCheckSkipsEntriesThatVanish(t *testing.T) {
	d := Dir{Path: filepath.Join(testdir.New(t), "home")}
	if err := d.Ensure(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"specmetry.db-journal", "kept"} {
		f, err := CreatePrivateFile(d.Join(name), true)
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}
	if err := os.Mkdir(d.Join("transient-dir"), 0o700); err != nil {
		t.Fatal(err)
	}
	vanishing := map[string]bool{d.Join("specmetry.db-journal"): true, d.Join("transient-dir"): true}
	beforeEntryCheck = func(p string) {
		if vanishing[p] {
			if err := os.RemoveAll(p); err != nil {
				t.Error(err)
			}
		}
	}
	t.Cleanup(func() { beforeEntryCheck = nil })
	problems, err := Check(d)
	if err != nil || len(problems) != 0 {
		t.Fatalf("Check with vanishing entries = %v, %v", problems, err)
	}
	if err := d.Ensure(); err != nil {
		t.Fatalf("Ensure with vanishing entries: %v", err)
	}
	if _, err := os.Stat(d.Join("kept")); err != nil {
		t.Fatal(err)
	}

	// The data directory itself missing is still an error.
	beforeEntryCheck = nil
	gone := Dir{Path: filepath.Join(testdir.New(t), "missing")}
	if _, err := checkPermissions(gone.Path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("checkPermissions of a missing directory = %v", err)
	}
}
