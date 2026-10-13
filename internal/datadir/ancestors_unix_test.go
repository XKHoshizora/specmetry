//go:build !windows

package datadir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

func mkdirMode(t *testing.T, p string, mode os.FileMode) string {
	t.Helper()
	if err := os.Mkdir(p, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil { // not subject to the umask
		t.Fatal(err)
	}
	return p
}

func requireRefusal(t *testing.T, d Dir, want string) {
	t.Helper()
	err := d.Ensure()
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error()+detailText(err), want) {
		t.Fatalf("Ensure() = %v (%s), want a refusal mentioning %q", err, detailText(err), want)
	}
	if _, err := os.Lstat(d.Path); !os.IsNotExist(err) {
		t.Fatalf("the refused data directory was created (%v)", err)
	}
}

// codex-review-005 B1: another user who can write to a directory on the path
// can rename the data directory away and put a link in its place after the
// checks. Such locations are refused before anything is created.
func TestEnsureRefusesAncestorWritableByOthers(t *testing.T) {
	shared := mkdirMode(t, filepath.Join(testdir.New(t), "shared"), 0o777)
	requireRefusal(t, Dir{Path: filepath.Join(shared, "home")}, "writable by other users")
}

func TestEnsureRefusesGroupWritableAncestor(t *testing.T) {
	shared := mkdirMode(t, filepath.Join(testdir.New(t), "shared"), 0o770)
	requireRefusal(t, Dir{Path: filepath.Join(shared, "nested", "home")}, "writable by other users")
}

// A sticky directory such as /tmp is writable by everyone, but only an
// entry's owner can rename or remove it; a data directory the current user
// owns there cannot be replaced.
func TestEnsureAcceptsStickyAncestor(t *testing.T) {
	sticky := mkdirMode(t, filepath.Join(testdir.New(t), "sticky"), os.ModeSticky|0o777)
	d := Dir{Path: filepath.Join(sticky, "home")}
	if err := d.Ensure(); err != nil {
		t.Fatalf("Ensure() = %v (%s)", err, detailText(err))
	}
}

// Symbolic links on the way are followed the way the kernel follows them,
// and the directories they lead through are checked too.
func TestEnsureChecksDirectoriesBehindLinksOnThePath(t *testing.T) {
	base := testdir.New(t)
	unsafe := mkdirMode(t, filepath.Join(base, "unsafe"), 0o777)
	safe := mkdirMode(t, filepath.Join(base, "safe"), 0o700)
	if err := os.Symlink(unsafe, filepath.Join(safe, "link")); err != nil {
		t.Fatal(err)
	}
	requireRefusal(t, Dir{Path: filepath.Join(safe, "link", "home")}, "writable by other users")

	private := mkdirMode(t, filepath.Join(base, "private"), 0o700)
	if err := os.Symlink(private, filepath.Join(safe, "good")); err != nil {
		t.Fatal(err)
	}
	if err := (Dir{Path: filepath.Join(safe, "good", "home")}).Ensure(); err != nil {
		t.Fatalf("a link to a private directory was refused: %v (%s)", err, detailText(err))
	}
}
