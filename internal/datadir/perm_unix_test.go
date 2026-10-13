//go:build !windows

package datadir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

// MVP-PRV-05: on Unix the data directory is 0700 and its files are 0600.
func TestUnixModes(t *testing.T) {
	d := Dir{Path: filepath.Join(testdir.New(t), "home")}
	if err := d.Ensure(); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(d.Path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o700 {
		t.Fatalf("data directory mode %04o, want 0700", got)
	}
	f, err := CreatePrivateFile(d.Join(ConfigFile), false)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	fi, err = os.Stat(d.Join(ConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Fatalf("file mode %04o, want 0600", got)
	}
}

func TestUnixCheckReportsLoosenedPermissions(t *testing.T) {
	d := Dir{Path: filepath.Join(testdir.New(t), "home")}
	if err := d.Ensure(); err != nil {
		t.Fatal(err)
	}
	p := d.Join("loose.txt")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(d.Path, 0o755); err != nil {
		t.Fatal(err)
	}
	problems, err := Check(d)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(problems, "\n")
	if !strings.Contains(joined, "want 0700") || !strings.Contains(joined, "loose.txt: mode 0644, want 0600") {
		t.Fatalf("problems = %v", problems)
	}
}

// loosen makes path readable by other users.
func loosen(t *testing.T, path string) {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mode := os.FileMode(0o644)
	if fi.IsDir() {
		mode = 0o755
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

// codex-review-004 B1: a data directory that is a symbolic link is refused.
func TestUnixCheckRefusesSymlinkedRoot(t *testing.T) {
	target := Dir{Path: filepath.Join(testdir.New(t), "real")}
	if err := target.Ensure(); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(testdir.New(t), "link")
	if err := os.Symlink(target.Path, link); err != nil {
		t.Fatal(err)
	}
	problems, err := Check(Dir{Path: link})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(problems, "\n"), "symbolic link") {
		t.Fatalf("symlinked root not reported: %v", problems)
	}
}
