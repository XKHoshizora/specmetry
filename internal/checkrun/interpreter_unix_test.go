//go:build !windows

package checkrun_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/checkrun"
)

// file makes an executable (or not) file at path, with its directory.
func file(t *testing.T, path string, mode os.FileMode) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), mode); err != nil {
		t.Fatal(err)
	}
	return path
}

// WP2.5 spec §5.3, §19.8 on Unix: bash and pwsh are the first acceptable
// candidate on PATH, skipping empty and relative entries ("." too), a
// missing file, a directory and a file without execute permission, and
// any whose final path is in a worktree, reached by an absolute PATH entry
// or a symbolic link; the result is the final path. sh is /bin/sh,
// whatever PATH says; cmd and powershell are not available.
func TestLookupUnix(t *testing.T) {
	base := t.TempDir()
	worktree := filepath.Join(base, "worktree")
	inside := file(t, filepath.Join(worktree, "bin", "bash"), 0o755)
	relative := filepath.Join(base, "cwd")
	file(t, filepath.Join(relative, "bash"), 0o755)
	file(t, filepath.Join(relative, "bin", "bash"), 0o755)
	t.Chdir(relative)
	asDir := filepath.Join(base, "dir", "bash")
	if err := os.MkdirAll(asDir, 0o755); err != nil {
		t.Fatal(err)
	}
	notExec := file(t, filepath.Join(base, "noexec", "bash"), 0o644)
	linked := filepath.Join(base, "link", "bash")
	if err := os.MkdirAll(filepath.Dir(linked), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(inside, linked); err != nil {
		t.Fatal(err)
	}
	good := file(t, filepath.Join(base, "good", "bash"), 0o755)
	alias := filepath.Join(base, "alias")
	if err := os.Symlink(filepath.Dir(good), alias); err != nil {
		t.Fatal(err)
	}
	path := strings.Join([]string{"", ".", "bin", filepath.Join(base, "missing"), filepath.Dir(asDir), filepath.Dir(notExec),
		filepath.Join(worktree, "bin"), filepath.Dir(linked), alias, filepath.Dir(good)}, string(os.PathListSeparator))
	f := checkrun.NewFinder([]string{worktree}, path, nil)
	in, problem, err := checkrun.Lookup(ctx, f, "bash")
	want, _ := filepath.EvalSymlinks(good)
	if err != nil || problem != "" || in.Path != want || in.Shell != "bash" || in.PathPrefix != "" {
		t.Fatalf("bash = %+v, %q, %v; want %s", in, problem, err, want)
	}

	// Nothing acceptable: the worktree's own, through a link, is not used.
	f = checkrun.NewFinder([]string{worktree}, strings.Join([]string{filepath.Join(worktree, "bin"), filepath.Dir(linked), "."},
		string(os.PathListSeparator)), nil)
	if in, problem, err := checkrun.Lookup(ctx, f, "bash"); err != nil || in.Path != "" || !strings.Contains(problem, "no bash") {
		t.Errorf("bash in the worktree only = %+v, %q, %v", in, problem, err)
	}
	// Each way into the worktree on its own: excluded, while without the
	// worktree to exclude the same entry finds the worktree's bash.
	inWorktree, _ := filepath.EvalSymlinks(inside)
	for _, entry := range []string{filepath.Join(worktree, "bin"), filepath.Dir(linked)} {
		if in, _, _ := checkrun.Lookup(ctx, checkrun.NewFinder(nil, entry, nil), "bash"); in.Path != inWorktree {
			t.Errorf("%s without exclusion = %+v; want %s", entry, in, inWorktree)
		}
		if in, problem, _ := checkrun.Lookup(ctx, checkrun.NewFinder([]string{worktree}, entry, nil), "bash"); in.Path != "" || problem == "" {
			t.Errorf("%s = %+v; want it excluded", entry, in)
		}
	}
	file(t, filepath.Join(base, "pw", "pwsh"), 0o755)
	f = checkrun.NewFinder([]string{worktree}, filepath.Join(base, "pw"), nil)
	if in, _, _ := checkrun.Lookup(ctx, f, "pwsh"); filepath.Base(in.Path) != "pwsh" {
		t.Fatalf("pwsh = %+v", in)
	}

	// sh is /bin/sh, not PATH's.
	file(t, filepath.Join(base, "good", "sh"), 0o755)
	in, problem, err = checkrun.Lookup(ctx, checkrun.NewFinder([]string{worktree}, filepath.Dir(good), nil), "sh")
	binSh, statErr := os.Stat("/bin/sh")
	got, gotErr := os.Stat(in.Path)
	if err != nil || problem != "" || statErr != nil || gotErr != nil || !os.SameFile(binSh, got) {
		t.Fatalf("sh = %+v, %q, %v", in, problem, err)
	}
	// A worktree that holds /bin/sh's final path excludes it.
	if in, problem, _ := checkrun.Lookup(ctx, checkrun.NewFinder([]string{"/"}, "", nil), "sh"); in.Path != "" || problem == "" {
		t.Fatalf("sh inside a worktree = %+v", in)
	}
	for _, shell := range []string{"cmd", "powershell"} {
		if in, problem, err := checkrun.Lookup(ctx, f, shell); err != nil || in.Path != "" || !strings.Contains(problem, "no "+shell) {
			t.Errorf("%s = %+v, %q, %v", shell, in, problem, err)
		}
	}
}

// WP2.5 spec §5.3 (MVP-EV-20): on Unix a cmd or powershell check refuses
// the batch, at its criterion only.
func TestPrepareCmdOnUnix(t *testing.T) {
	w := newWorkspace(t)
	w.criteria(t, ac{key: "AC-001-1", check: sh("echo one")}, ac{key: "AC-001-2", check: shell("cmd", "echo two")},
		ac{key: "AC-001-3", check: shell("powershell", "echo three")})
	h := checkrun.NewHost()
	_, err := run(t, h, all())
	refused(t, h, err, apperr.ExitEnvironment, "AC-001-2", "AC-001-3")
	for _, d := range apperr.From(err).Details {
		if !strings.HasPrefix(d.Problem, "shell_unavailable: ") {
			t.Errorf("%s: %q", d.Pointer, d.Problem)
		}
	}
}
