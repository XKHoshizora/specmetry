//go:build darwin

package datadir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

// On macOS the ACLs are listed with /bin/ls after the walk; an entry that
// vanishes in between is skipped there too, while the others are still
// checked.
func TestDarwinACLCheckSkipsEntriesThatVanish(t *testing.T) {
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
	beforeACLCheck = func([]string) { _ = os.Remove(d.Join("specmetry.db-journal")) }
	t.Cleanup(func() { beforeACLCheck = nil })
	if problems, err := Check(d); err != nil || len(problems) != 0 {
		t.Fatalf("Check = %v, %v", problems, err)
	}
}

// Without gone (the directories on the path to the data directory), a
// path that vanished is still an error.
func TestDarwinACLCheckOfAncestorsIsStrict(t *testing.T) {
	missing := filepath.Join(testdir.New(t), "missing")
	if _, err := extendedACLProblems([]string{missing}, nil); err == nil {
		t.Fatal("a missing ancestor was skipped")
	}
}
