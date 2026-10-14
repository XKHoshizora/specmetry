// Package gitx runs the git CLI for Specmetry (TD-05). It only reads: no
// command it runs changes the repository, its index or its configuration,
// runs another program or writes a file (codex-review-015 B1):
//
//   - git gets none of the inherited GIT_* variables, so neither the
//     environment's configuration (GIT_CONFIG_COUNT, GIT_CONFIG_PARAMETERS,
//     ...) nor tracing nor a repository named by GIT_DIR applies. The
//     repository is found from the directory, as the Hook finds it
//     (architecture §12.2).
//   - The system and global configuration are replaced by an empty file, so
//     only the repository's own configuration applies, and in it
//     core.fsmonitor, the one setting that makes a read command run a
//     program, is turned off on the command line.
//   - The commands run are read commands that run no hooks, without
//     optional locks, a pager or replace objects.
//
// Errors never carry git's own messages, which can quote configuration
// values, nor the directory (codex-review-015 B3).
package gitx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

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
			return false, apperr.New(apperr.Environment, "cannot look for a git repository ("+osProblem(err)+")")
		}
		if isGitDir(d) {
			return false, apperr.New(apperr.Environment,
				"the current directory is inside a git directory or a bare repository, which has no worktree; run the command in a worktree")
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

// objectID matches a full object or commit ID (SHA-1 or SHA-256), the only
// form TreeFiles and CountLines accept, so that no argument reaches git as
// an option.
var objectID = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

// TreeFiles lists the regular files of commit in the repository of
// worktree: the blob of each, by its path from the top with forward
// slashes. Symbolic links, submodules and directories are not files.
func TreeFiles(ctx context.Context, worktree, commit string) (map[string]string, error) {
	if !objectID.MatchString(commit) {
		return nil, apperr.New(apperr.Internal, "TreeFiles needs a full commit ID")
	}
	out, err := run(ctx, worktree, "ls-tree", "-r", "-z", "--full-tree", commit)
	if err != nil {
		return nil, err
	}
	files := map[string]string{}
	for _, entry := range strings.Split(string(out), "\x00") {
		if entry == "" {
			continue
		}
		meta, path, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 {
			return nil, apperr.New(apperr.Environment, "cannot read the tree of the commit")
		}
		if fields[1] == "blob" && (fields[0] == "100644" || fields[0] == "100755") {
			files[path] = fields[2]
		}
	}
	return files, nil
}

// CountLines counts the lines of a blob without keeping its content: its
// line breaks, and one more for a last line without one.
func CountLines(ctx context.Context, worktree, object string) (int, error) {
	if !objectID.MatchString(object) {
		return 0, apperr.New(apperr.Internal, "CountLines needs a full object ID")
	}
	if err := checkVersion(ctx); err != nil {
		return 0, err
	}
	var c lineCounter
	code, err := execTo(ctx, worktree, &c, append(slices.Clone(globalArgs), "cat-file", "blob", object)...)
	if err != nil {
		return 0, err
	}
	if code != 0 {
		return 0, failed("cat-file", code)
	}
	return c.lines(), nil
}

// lineCounter counts line breaks in what is written to it and keeps
// nothing else.
type lineCounter struct {
	breaks int
	last   byte
	size   int64
}

func (c *lineCounter) Write(p []byte) (int, error) {
	c.breaks += bytes.Count(p, []byte{'\n'})
	if len(p) > 0 {
		c.last = p[len(p)-1]
	}
	c.size += int64(len(p))
	return len(p), nil
}

func (c *lineCounter) lines() int {
	if c.size > 0 && c.last != '\n' {
		return c.breaks + 1
	}
	return c.breaks
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

// controlledEnv is the git environment Specmetry sets for every git it
// runs, the version probe included: no system or global configuration (an
// empty file instead; git maps /dev/null on Windows too); trace2 off, which
// a git too old for GIT_CONFIG_GLOBAL would otherwise enable from the
// user's global configuration while it only prints its version
// (codex-review-016 B1; the variables override trace2.*Target); no optional
// locks, so reading never writes the index; no prompts; and the .git entry
// that findDotGit found by walking up without a filesystem limit.
var controlledEnv = []string{
	"GIT_CONFIG_NOSYSTEM=1",
	"GIT_CONFIG_SYSTEM=/dev/null",
	"GIT_CONFIG_GLOBAL=/dev/null",
	"GIT_TRACE2=0",
	"GIT_TRACE2_EVENT=0",
	"GIT_TRACE2_PERF=0",
	"GIT_OPTIONAL_LOCKS=0",
	"GIT_TERMINAL_PROMPT=0",
	"GIT_DISCOVERY_ACROSS_FILESYSTEM=1",
}

// globalArgs come before every git command. An empty core.fsmonitor turns
// the file system monitor off in every git version: a hook command from the
// repository's configuration is never run.
var globalArgs = []string{"--no-pager", "--no-replace-objects", "-c", "core.fsmonitor="}

// filterEnv drops every GIT_* variable from env, whatever its letter case,
// and adds controlledEnv. Other variables, such as HOME (git still reads
// the default global excludes file) and PATH, are kept.
func filterEnv(env []string) []string {
	out := make([]string, 0, len(env)+len(controlledEnv))
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if !strings.HasPrefix(strings.ToUpper(name), "GIT_") {
			out = append(out, kv)
		}
	}
	return append(out, controlledEnv...)
}

// minimumVersion is the oldest git that reads GIT_CONFIG_GLOBAL and
// GIT_CONFIG_SYSTEM; an older one would read the user's configuration.
const minimumVersion = "2.32"

var versionPattern = regexp.MustCompile(`^git version (\d+)\.(\d+)`)

// supportedVersion reports whether `git --version` printed minimumVersion
// or newer.
func supportedVersion(out string) bool {
	m := versionPattern.FindStringSubmatch(strings.TrimSpace(out))
	if m == nil {
		return false
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return major > 2 || major == 2 && minor >= 32
}

var (
	versionMu sync.Mutex
	versionOK bool
)

// checkVersion refuses a git too old to be isolated. Only a successful
// check is remembered, so a cancelled one is repeated.
func checkVersion(ctx context.Context) error {
	versionMu.Lock()
	defer versionMu.Unlock()
	if versionOK {
		return nil
	}
	out, code, err := exec1(ctx, "", "--version")
	if err != nil {
		return err
	}
	if code != 0 || !supportedVersion(string(out)) {
		return apperr.New(apperr.Environment, "git "+minimumVersion+" or newer is required")
	}
	versionOK = true
	return nil
}

func run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	out, code, err := runCode(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, failed(args[0], code)
	}
	return out, nil
}

// runCode runs git in dir after checking its version. A non-zero exit
// status other than 1 is an error; status 1 is returned for the caller to
// interpret.
func runCode(ctx context.Context, dir string, args ...string) ([]byte, int, error) {
	if err := checkVersion(ctx); err != nil {
		return nil, -1, err
	}
	out, code, err := exec1(ctx, dir, append(slices.Clone(globalArgs), args...)...)
	if err != nil {
		return nil, -1, err
	}
	if code != 0 && code != 1 {
		return nil, code, failed(args[0], code)
	}
	return out, code, nil
}

// exec1 runs git once in the controlled environment. git's own messages
// are discarded, not kept: they may quote configuration values.
func exec1(ctx context.Context, dir string, args ...string) ([]byte, int, error) {
	var stdout bytes.Buffer
	code, err := execTo(ctx, dir, &stdout, args...)
	if err != nil {
		return nil, -1, err
	}
	return stdout.Bytes(), code, nil
}

// execTo is exec1 with git's output written to stdout as it comes.
func execTo(ctx context.Context, dir string, stdout io.Writer, args ...string) (int, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = filterEnv(os.Environ())
	cmd.Stdout = stdout
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case ctx.Err() != nil:
		return -1, apperr.Wrap(apperr.Environment, ctx.Err(), "git was interrupted")
	case err == nil:
		return 0, nil
	case errors.As(err, &exit):
		return exit.ExitCode(), nil
	case errors.Is(err, exec.ErrNotFound):
		return -1, apperr.New(apperr.Environment, "git is required but was not found on PATH")
	default:
		return -1, apperr.New(apperr.Environment, "cannot run git")
	}
}

// failed is the error of a git command that did not succeed: the command
// and its exit status only, never git's message or the directory.
func failed(command string, code int) error {
	return apperr.Newf(apperr.Environment,
		"git %s failed with exit status %d; run git in this directory (for example git status) to see why", command, code)
}

// osProblem describes a file system error without its path.
func osProblem(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return "file system error"
}
