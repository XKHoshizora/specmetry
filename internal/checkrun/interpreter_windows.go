package checkrun

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/sys/windows"
)

// finalPath is the path GetFinalPathNameByHandle gives the file at p:
// links and junctions resolved, short names expanded, letter case as
// stored, without the \\?\ prefix. The handle it opens for that is closed
// before it returns, and that close's result is consumed (Review 123
// B-123-2): a close that fails returns a *ReleaseError, never the path.
func finalPath(p string) (string, error) {
	name, err := windows.UTF16PtrFromString(p)
	if err != nil {
		return "", err
	}
	h, err := windows.CreateFile(name, windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", err
	}
	final, err := finalName(h)
	if testHookHandle != nil {
		testHookHandle(p, h)
	}
	if cerr := released("handle", windows.CloseHandle(h)); cerr != nil {
		return "", errors.Join(&ReleaseError{Resources: []string{"a handle opened to resolve a path"}, Errs: []error{cerr}}, err)
	}
	return final, err
}

// Test hook: testHookHandle runs with finalPath's handle of the path p
// just before it is closed.
var testHookHandle func(p string, h windows.Handle)

// finalName reads the final path of the file open as h.
func finalName(h windows.Handle) (string, error) {
	// FILE_NAME_NORMALIZED | VOLUME_NAME_DOS, both 0.
	const flags = 0
	buf := make([]uint16, windows.MAX_PATH)
	for {
		n, err := windows.GetFinalPathNameByHandle(h, &buf[0], uint32(len(buf)), flags)
		if err != nil {
			return "", err
		}
		if int(n) < len(buf) {
			s := windows.UTF16ToString(buf[:n])
			if rest, ok := strings.CutPrefix(s, `\\?\UNC\`); ok {
				return `\\` + rest, nil
			}
			return strings.TrimPrefix(s, `\\?\`), nil
		}
		buf = make([]uint16, n+1)
	}
}

// executable: on Windows any regular file found by its full name.
func executable(fs.FileInfo) bool { return true }

// gitEnvironments are the directories of Git for Windows that hold
// libexec\git-core (spec §19.8 step 2).
var gitEnvironments = []string{"mingw64", "ucrt64", "clang64", "clangarm64", "mingw32"}

// find is the Windows table of spec §5.3 and §19.8: sh and bash come only
// from the Git for Windows of the git gitx runs (D2); cmd and powershell
// from the system directory, not from ComSpec or SystemRoot; pwsh is the
// first acceptable pwsh.exe on PATH, PATHEXT not applied.
func (f *finder) find(ctx context.Context, shell string) (Interpreter, bool) {
	switch shell {
	case "sh", "bash":
		return f.gitForWindows(ctx, shell)
	case "cmd", "powershell":
		dir, err := windows.GetSystemDirectory()
		if err != nil {
			return Interpreter{}, false
		}
		name := `cmd.exe`
		if shell == "powershell" {
			name = `WindowsPowerShell\v1.0\powershell.exe`
		}
		p, ok := f.accept(filepath.Join(dir, name))
		return Interpreter{Shell: shell, Path: p}, ok
	case "pwsh":
		p, ok := f.onPath("pwsh.exe")
		return Interpreter{Shell: shell, Path: p}, ok
	}
	return Interpreter{}, false
}

// gitForWindows is spec §19.8 steps 1 to 3: git --exec-path, in gitx's
// controlled environment, normalized, must be <R>\<env>\libexec\git-core;
// <R>\usr\bin must hold the interpreter and msys-2.0.dll, the mark of Git
// for Windows' MSYS runtime, both regular files; and <R> must be outside
// every root. Anything else is not available; so is a failed exec path
// read, a cancelled one too (lookup reports the cancellation).
func (f *finder) gitForWindows(ctx context.Context, shell string) (Interpreter, bool) {
	p, err := f.execPath(ctx)
	if err != nil {
		return Interpreter{}, false
	}
	core, err := f.final(filepath.FromSlash(p))
	if err != nil {
		return Interpreter{}, false
	}
	libexec := filepath.Dir(core)
	env := filepath.Dir(libexec)
	root := filepath.Dir(env)
	if !strings.EqualFold(filepath.Base(core), "git-core") || !strings.EqualFold(filepath.Base(libexec), "libexec") ||
		!slices.Contains(gitEnvironments, strings.ToLower(filepath.Base(env))) || root == env || f.excluded(root) {
		return Interpreter{}, false
	}
	bin := filepath.Join(root, "usr", "bin")
	if _, ok := f.accept(filepath.Join(bin, "msys-2.0.dll")); !ok {
		return Interpreter{}, false
	}
	exe, ok := f.accept(filepath.Join(bin, shell+".exe"))
	if !ok {
		return Interpreter{}, false
	}
	return Interpreter{Shell: shell, Path: exe, PathPrefix: bin}, true
}
