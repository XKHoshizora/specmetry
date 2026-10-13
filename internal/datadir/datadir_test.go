package datadir

import (
	"os"
	"path/filepath"
	"testing"
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
