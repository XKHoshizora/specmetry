package checkrun_test

import (
	"context"
	"errors"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/XKHoshizora/specmetry/internal/checkrun"
	"github.com/XKHoshizora/specmetry/internal/gitx"
)

func file(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func final(t *testing.T, p string) string {
	t.Helper()
	f, err := checkrun.FinalPath(p)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// gitLayout makes a Git for Windows tree under a new root: <env>\libexec
// \git-core, and in usr\bin the files named. It returns the root and the
// exec path as git prints it, with forward slashes.
func gitLayout(t *testing.T, env string, usrBin ...string) (root, execPath string) {
	t.Helper()
	root = filepath.Join(t.TempDir(), "Git")
	core := filepath.Join(root, env, "libexec", "git-core")
	if err := os.MkdirAll(core, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range usrBin {
		file(t, filepath.Join(root, "usr", "bin", name))
	}
	return root, filepath.ToSlash(core)
}

func execPathOf(p string) func(context.Context) (string, error) {
	return func(context.Context) (string, error) { return p, nil }
}

// WP2.5 spec §19.8 steps 1 to 3 (D2): sh and bash are <R>\usr\bin's, for
// an exec path <R>\<env>\libexec\git-core with env one of Git for
// Windows' (letter case aside) and msys-2.0.dll beside them, both regular
// files; PathPrefix is <R>\usr\bin. Another env directory, a missing
// runtime mark or interpreter, a distribution without usr\bin (MinGit), an
// exec path that cannot be read, or <R> inside a worktree: not available.
func TestLookupGitForWindowsLayouts(t *testing.T) {
	full := []string{"sh.exe", "bash.exe", "msys-2.0.dll"}
	for _, env := range []string{"mingw64", "ucrt64", "clang64", "clangarm64", "mingw32", "UCRT64"} {
		root, p := gitLayout(t, env, full...)
		for _, shell := range []string{"sh", "bash"} {
			in, problem, err := checkrun.Lookup(ctx, checkrun.NewFinder([]string{t.TempDir()}, "", execPathOf(p)), shell)
			want := final(t, filepath.Join(root, "usr", "bin", shell+".exe"))
			if err != nil || problem != "" || in.Path != want || in.PathPrefix != filepath.Dir(want) || in.Shell != shell {
				t.Errorf("%s %s: %+v, %q, %v; want %s", env, shell, in, problem, err, want)
			}
		}
	}
	unavailable := func(name string, f *checkrun.Finder, shell string) {
		t.Helper()
		if in, problem, err := checkrun.Lookup(ctx, f, shell); err != nil || in.Path != "" || problem == "" {
			t.Errorf("%s: %+v, %q, %v", name, in, problem, err)
		}
	}
	_, p := gitLayout(t, "usr64", full...)
	unavailable("another env directory", checkrun.NewFinder(nil, "", execPathOf(p)), "sh")
	_, p = gitLayout(t, "mingw64", "sh.exe", "bash.exe")
	unavailable("no msys-2.0.dll", checkrun.NewFinder(nil, "", execPathOf(p)), "sh")
	_, p = gitLayout(t, "mingw64", "bash.exe", "msys-2.0.dll")
	unavailable("no sh.exe", checkrun.NewFinder(nil, "", execPathOf(p)), "sh")
	_, p = gitLayout(t, "mingw64")
	unavailable("MinGit, without usr\\bin", checkrun.NewFinder(nil, "", execPathOf(p)), "bash")
	unavailable("no exec path", checkrun.NewFinder(nil, "", func(context.Context) (string, error) { return "", errors.New("no git") }), "sh")
	root, p := gitLayout(t, "mingw64", full...)
	unavailable("Git inside a worktree", checkrun.NewFinder([]string{filepath.Dir(root)}, "", execPathOf(p)), "sh")
}

// WP2.5 spec §19.8 step 1 with the git gitx runs (here Git for Windows on
// the machine): sh is <R>\usr\bin\sh.exe for the <R> of git --exec-path.
// cmd and powershell come from the system directory.
func TestLookupOnThisMachine(t *testing.T) {
	dir := t.TempDir()
	p, err := gitx.ExecPath(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(final(t, filepath.FromSlash(p)))))
	f := checkrun.NewFinder([]string{dir}, os.Getenv("PATH"), func(ctx context.Context) (string, error) { return gitx.ExecPath(ctx, dir) })
	in, problem, err := checkrun.Lookup(ctx, f, "sh")
	if err != nil || problem != "" || !strings.EqualFold(in.Path, filepath.Join(root, "usr", "bin", "sh.exe")) {
		t.Fatalf("sh = %+v, %q, %v; Git at %s", in, problem, err, root)
	}
	system, err := windows.GetSystemDirectory()
	if err != nil {
		t.Fatal(err)
	}
	for shell, name := range map[string]string{"cmd": `cmd.exe`, "powershell": `WindowsPowerShell\v1.0\powershell.exe`} {
		in, problem, err := checkrun.Lookup(ctx, f, shell)
		if err != nil || problem != "" || in.Path != final(t, filepath.Join(system, name)) {
			t.Errorf("%s = %+v, %q, %v", shell, in, problem, err)
		}
	}
}

// WP2.5 spec §5.3, §19.8 on Windows: pwsh is the first acceptable
// pwsh.exe on PATH (PATHEXT not applied: a pwsh.cmd is not one); empty and
// relative entries are skipped; a candidate whose final path is in a
// worktree is excluded, whether reached by an absolute entry, another
// letter case, a junction or a short name.
func TestLookupPwshOnWindows(t *testing.T) {
	base := t.TempDir()
	worktree := filepath.Join(base, "worktree")
	bin := filepath.Dir(file(t, filepath.Join(worktree, "Tools Bin", "pwsh.exe")))
	cmdOnly := filepath.Dir(file(t, filepath.Join(base, "cmdonly", "pwsh.cmd")))
	relative := filepath.Join(base, "cwd")
	file(t, filepath.Join(relative, "pwsh.exe"))
	t.Chdir(relative)
	junction := filepath.Join(base, "junction")
	if out, err := osexec.Command("cmd", "/d", "/c", "mklink", "/J", junction, bin).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v\n%s", err, out)
	}
	good := file(t, filepath.Join(base, "good", "pwsh.exe"))
	entries := []string{"", ".", "cwd", cmdOnly, bin, strings.ToUpper(bin), junction}
	short := make([]uint16, windows.MAX_PATH)
	longPtr, err := windows.UTF16PtrFromString(bin)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := windows.GetShortPathName(longPtr, &short[0], uint32(len(short))); err == nil && n > 0 && windows.UTF16ToString(short[:n]) != bin {
		entries = append(entries, windows.UTF16ToString(short[:n]))
	} else {
		t.Log("no short name on this volume: the short-name exclusion is not exercised here")
	}
	path := strings.Join(append(entries, filepath.Dir(good)), string(os.PathListSeparator))
	in, problem, err := checkrun.Lookup(ctx, checkrun.NewFinder([]string{worktree}, path, nil), "pwsh")
	if err != nil || problem != "" || in.Path != final(t, good) {
		t.Errorf("pwsh = %+v, %q, %v; want %s", in, problem, err, final(t, good))
	}
	in, problem, _ = checkrun.Lookup(ctx, checkrun.NewFinder([]string{worktree}, strings.Join(entries, string(os.PathListSeparator)), nil), "pwsh")
	if in.Path != "" || problem == "" {
		t.Errorf("pwsh in the worktree only = %+v", in)
	}
	// Each way into the worktree on its own: excluded, while without the
	// worktree to exclude the same entry finds the worktree's pwsh.exe.
	inWorktree := final(t, filepath.Join(bin, "pwsh.exe"))
	for _, entry := range entries[4:] {
		if in, _, _ := checkrun.Lookup(ctx, checkrun.NewFinder(nil, entry, nil), "pwsh"); in.Path != inWorktree {
			t.Errorf("%s without exclusion = %+v; want %s", entry, in, inWorktree)
		}
		if in, problem, _ := checkrun.Lookup(ctx, checkrun.NewFinder([]string{worktree}, entry, nil), "pwsh"); in.Path != "" || problem == "" {
			t.Errorf("%s = %+v; want it excluded", entry, in)
		}
	}
}
