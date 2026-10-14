// Package gitx runs the git CLI for Specmetry (TD-05). It only reads: no
// command it runs changes the repository, its index or its configuration.
//
// The repository is always found from the directory, as the Hook finds it
// (architecture §12.2): variables that point git at another repository,
// such as GIT_DIR, are not passed on.
package gitx

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// ErrNotRepository means that neither the directory nor any of its parents
// holds a .git entry.
var ErrNotRepository = errors.New("not a git repository")

// Location places a directory in its git repository. Every path is
// absolute and canonical (see Canonical).
type Location struct {
	// Worktree is the root of the worktree that holds the directory.
	Worktree string
	// GitDir is that worktree's own git directory (HEAD, index).
	GitDir string
	// CommonDir is the git directory shared by all worktrees.
	CommonDir string
	// Main is the root of the main worktree.
	Main string
}

// Locate finds the git repository of dir. It returns ErrNotRepository when
// there is none. Inside a git directory or a bare repository there is no
// worktree, which is an environment error, as is a repository git refuses
// to read (for example one owned by another user).
func Locate(ctx context.Context, dir string) (Location, error) {
	found, err := findDotGit(dir)
	if err != nil {
		return Location{}, err
	}
	if !found {
		return Location{}, ErrNotRepository
	}
	out, err := run(ctx, dir, "rev-parse", "--show-toplevel", "--absolute-git-dir", "--git-common-dir")
	if err != nil {
		return Location{}, err
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) != 3 {
		return Location{}, apperr.Newf(apperr.Environment, "git rev-parse printed %d lines, want 3", len(lines))
	}
	common := filepath.FromSlash(lines[2])
	if !filepath.IsAbs(common) {
		common = filepath.Join(dir, common)
	}
	loc := Location{
		Worktree:  Canonical(lines[0]),
		GitDir:    Canonical(lines[1]),
		CommonDir: Canonical(common),
	}
	if loc.Main, err = mainWorktree(ctx, dir); err != nil {
		return Location{}, err
	}
	return loc, nil
}

// findDotGit reports whether dir or one of its parents holds a .git entry.
// A git directory met on the way (dir inside .git, or a bare repository)
// is an error: there is no worktree to work in.
func findDotGit(dir string) (bool, error) {
	for d := filepath.Clean(dir); ; {
		if _, err := os.Lstat(filepath.Join(d, ".git")); err == nil {
			return true, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return false, apperr.Wrap(apperr.Environment, err, "cannot look for a git repository")
		}
		if isGitDir(d) {
			return false, apperr.Newf(apperr.Environment,
				"%s is inside a git directory or a bare repository, which has no worktree; run the command in a worktree", dir)
		}
		parent := filepath.Dir(d)
		if parent == d {
			return false, nil
		}
		d = parent
	}
}

// isGitDir reports whether dir looks like a git directory: a HEAD file and
// objects and refs directories.
func isGitDir(dir string) bool {
	head, err := os.Lstat(filepath.Join(dir, "HEAD"))
	if err != nil || !head.Mode().IsRegular() {
		return false
	}
	for _, sub := range []string{"objects", "refs"} {
		if fi, err := os.Lstat(filepath.Join(dir, sub)); err != nil || !fi.IsDir() {
			return false
		}
	}
	return true
}

// mainWorktree returns the main worktree of the repository that holds dir:
// the first entry of `git worktree list`. A bare repository has none.
// (-z would need git 2.36.)
func mainWorktree(ctx context.Context, dir string) (string, error) {
	out, err := run(ctx, dir, "worktree", "list", "--porcelain")
	if err != nil {
		return "", err
	}
	var main string
	for i, field := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		if field == "" {
			break // the end of the first entry
		}
		if i == 0 {
			p, ok := strings.CutPrefix(field, "worktree ")
			if !ok {
				return "", apperr.New(apperr.Environment, "cannot read the worktree list of the repository")
			}
			main = p
		} else if field == "bare" {
			return "", apperr.New(apperr.Environment,
				"the repository is bare and has no main worktree; Specmetry needs one for its marker (architecture §12.2)")
		}
	}
	if main == "" {
		return "", apperr.New(apperr.Environment, "cannot read the worktree list of the repository")
	}
	return Canonical(main), nil
}

// HeadBranch returns the full name of the branch checked out in worktree,
// such as refs/heads/main, or "" when its HEAD is detached. A branch
// without commits yet is still returned.
func HeadBranch(ctx context.Context, worktree string) (string, error) {
	out, code, err := runCode(ctx, worktree, "symbolic-ref", "-q", "HEAD")
	if err != nil {
		return "", err
	}
	if code == 1 {
		return "", nil
	}
	ref := strings.TrimSpace(string(out))
	if !strings.HasPrefix(ref, "refs/heads/") {
		return "", nil
	}
	return ref, nil
}

// CommitOf resolves ref to the full ID of a commit, or "" when the ref does
// not exist or has no commit yet.
func CommitOf(ctx context.Context, dir, ref string) (string, error) {
	out, code, err := runCode(ctx, dir, "rev-parse", "-q", "--verify", ref+"^{commit}")
	if err != nil {
		return "", err
	}
	if code == 1 {
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// ListFiles lists the files of worktree that git sees: tracked files and
// untracked files that are not ignored, relative to the worktree root with
// forward slashes, each once.
func ListFiles(ctx context.Context, worktree string) ([]string, error) {
	out, err := run(ctx, worktree, "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--full-name", "--", ":/")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f != "" && !seen[f] {
			seen[f] = true
			files = append(files, f)
		}
	}
	return files, nil
}

// GitPath resolves name inside the git directories of worktree, as
// `git rev-parse --git-path` does: shared files such as info/exclude
// resolve into the common directory.
func GitPath(ctx context.Context, worktree, name string) (string, error) {
	out, err := run(ctx, worktree, "rev-parse", "--git-path", name)
	if err != nil {
		return "", err
	}
	p := filepath.FromSlash(strings.TrimRight(string(out), "\n"))
	if !filepath.IsAbs(p) {
		p = filepath.Join(worktree, p)
	}
	return Canonical(p), nil
}

// Canonical cleans an absolute path from git or the operating system and
// resolves symbolic links (and on Windows short names and letter case), so
// that one directory always has one spelling. A path that does not exist
// is only cleaned; for one whose last element does not exist yet, the
// parent is resolved.
func Canonical(p string) string {
	p = filepath.Clean(filepath.FromSlash(p))
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	if r, err := filepath.EvalSymlinks(filepath.Dir(p)); err == nil {
		return filepath.Join(r, filepath.Base(p))
	}
	return p
}

// locationVariables are the variables that make git use another
// repository, worktree, index or object store than the directory's own, or
// stop it from finding the repository above the directory.
var locationVariables = map[string]bool{
	"GIT_DIR": true, "GIT_WORK_TREE": true, "GIT_COMMON_DIR": true, "GIT_INDEX_FILE": true,
	"GIT_OBJECT_DIRECTORY": true, "GIT_ALTERNATE_OBJECT_DIRECTORIES": true, "GIT_NAMESPACE": true,
	"GIT_PREFIX": true, "GIT_IMPLICIT_WORK_TREE": true, "GIT_SHALLOW_FILE": true, "GIT_GRAFT_FILE": true,
	"GIT_REPLACE_REF_BASE": true, "GIT_NO_REPLACE_OBJECTS": true, "GIT_CEILING_DIRECTORIES": true,
	"GIT_DISCOVERY_ACROSS_FILESYSTEM": true, "GIT_OPTIONAL_LOCKS": true, "GIT_TERMINAL_PROMPT": true,
}

// filterEnv drops the location variables from env and adds settings that
// keep git from taking optional locks (so reading never writes the index)
// and from prompting. Variable names are case-insensitive on Windows.
func filterEnv(env []string) []string {
	out := make([]string, 0, len(env)+3)
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if runtime.GOOS == "windows" {
			name = strings.ToUpper(name)
		}
		if !locationVariables[name] {
			out = append(out, kv)
		}
	}
	return append(out,
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		// The .git entry was found by walking up without a filesystem
		// limit; git must find the same one.
		"GIT_DISCOVERY_ACROSS_FILESYSTEM=1")
}

func run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	out, code, err := runCode(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, apperr.Newf(apperr.Environment, "git %s failed in %s", args[0], dir)
	}
	return out, nil
}

// runCode runs git in dir. A non-zero exit status other than 1 is an
// error that carries git's message; status 1 is returned for the caller to
// interpret.
func runCode(ctx context.Context, dir string, args ...string) ([]byte, int, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = filterEnv(os.Environ())
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return stdout.Bytes(), 0, nil
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		return stdout.Bytes(), 1, nil
	case errors.As(err, &exit):
		return nil, exit.ExitCode(), apperr.Newf(apperr.Environment, "git %s failed in %s: %s", args[0], dir, firstLine(stderr.Bytes()))
	case errors.Is(err, exec.ErrNotFound):
		return nil, -1, apperr.New(apperr.Environment, "git is required but was not found on PATH")
	default:
		return nil, -1, apperr.Wrap(apperr.Environment, err, fmt.Sprintf("cannot run git %s", args[0]))
	}
}

func firstLine(b []byte) string {
	s := bufio.NewScanner(bytes.NewReader(b))
	if s.Scan() {
		return strings.TrimSpace(s.Text())
	}
	return "(no message)"
}
