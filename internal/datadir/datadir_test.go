package datadir

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

func TestResolveUsesEnvironment(t *testing.T) {
	want := filepath.Join(t.TempDir(), "home with space", "数据")
	t.Setenv(EnvHome, want)
	d, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if d.Path != want || !d.FromEnv {
		t.Fatalf("Resolve() = %+v, want %s from env", d, want)
	}
}

func TestResolveDefaultsToHome(t *testing.T) {
	t.Setenv(EnvHome, "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory:", err)
	}
	d, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if d.Path != filepath.Join(home, ".specmetry") || d.FromEnv {
		t.Fatalf("Resolve() = %+v", d)
	}
}

func TestEnsureCreatesPrivateDirectory(t *testing.T) {
	d := Dir{Path: filepath.Join(t.TempDir(), "nested", "家 dir")}
	if ok, _ := d.Exists(); ok {
		t.Fatal("directory should not exist yet")
	}
	if err := d.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := d.Ensure(); err != nil { // idempotent
		t.Fatal(err)
	}
	f, err := CreatePrivateFile(d.Join(DatabaseFile), true)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if _, err := CreatePrivateFile(d.Join(DatabaseFile), true); err == nil {
		t.Fatal("exclusive create must fail for an existing file")
	}
	if _, err := d.EnsureSub(BackupsDir); err != nil {
		t.Fatal(err)
	}
	problems, err := Check(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("fresh data directory has permission problems: %v", problems)
	}
}

func TestExistsRejectsFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(p, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (Dir{Path: p}).Exists(); err == nil {
		t.Fatal("Exists must fail when the path is a regular file")
	}
}

// codex-review-004 B1: Ensure verifies an existing directory instead of
// trusting it, and refuses one that is not private.
func TestEnsureRefusesInsecureExistingDirectory(t *testing.T) {
	d := Dir{Path: filepath.Join(t.TempDir(), "home")}
	if err := d.Ensure(); err != nil {
		t.Fatal(err)
	}
	loosen(t, d.Path)
	err := d.Ensure()
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "not private") {
		t.Fatalf("Ensure() = %v", err)
	}
	if len(apperr.From(err).Details) == 0 {
		t.Fatal("refusal lists no problems")
	}
}

// codex-review-004 B1: a file inside the data directory that is not private
// makes Ensure refuse.
func TestEnsureRefusesInsecureFile(t *testing.T) {
	d := Dir{Path: filepath.Join(t.TempDir(), "home")}
	if err := d.Ensure(); err != nil {
		t.Fatal(err)
	}
	f, err := CreatePrivateFile(d.Join(DatabaseFile), true)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	loosen(t, d.Join(DatabaseFile))
	if err := d.Ensure(); apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(detailText(err), DatabaseFile) {
		t.Fatalf("Ensure() = %v (%s)", err, detailText(err))
	}
}

// codex-review-004 B1: links inside the data directory are reported.
func TestCheckReportsSymlinkEntry(t *testing.T) {
	d := Dir{Path: filepath.Join(t.TempDir(), "home")}
	if err := d.Ensure(); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, d.Join(DatabaseFile)); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("creating symbolic links needs Developer Mode or administrator rights:", err)
		}
		t.Fatal(err)
	}
	problems, err := Check(d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(problems, "\n"), "symbolic link") {
		t.Fatalf("symlink not reported: %v", problems)
	}
}

func detailText(err error) string {
	if err == nil {
		return ""
	}
	var parts []string
	for _, d := range apperr.From(err).Details {
		parts = append(parts, d.Problem)
	}
	return strings.Join(parts, "; ")
}
