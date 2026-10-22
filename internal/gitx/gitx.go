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
//   - A command that may read a git object runs only with a git shown,
//     by running it, never to fetch a missing one (objects.go, PD-26).
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

// ObjectFormat returns the object format of the repository of worktree,
// "sha1" or "sha256", as git reports it (rev-parse --show-object-format
// reads the configuration, not an object). Any other answer is an
// environment error.
func ObjectFormat(ctx context.Context, worktree string) (string, error) {
	out, err := run(ctx, worktree, "rev-parse", "--show-object-format")
	if err != nil {
		return "", err
	}
	switch format := strings.TrimSpace(string(out)); format {
	case "sha1", "sha256":
		return format, nil
	}
	return "", apperr.New(apperr.Environment, "cannot tell the object format of the repository")
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

// ValidRefName reports whether ref is a well-formed full ref name, as git
// check-ref-format judges it. It reads no object and needs no repository;
// whether the ref exists is not asked.
func ValidRefName(ctx context.Context, dir, ref string) (bool, error) {
	if ref == "" || strings.HasPrefix(ref, "-") {
		return false, nil
	}
	_, code, err := runCode(ctx, dir, "check-ref-format", ref)
	if err != nil {
		return false, err
	}
	return code == 0, nil
}

// CommitOf resolves ref to the full ID of a commit, or "" when the ref does
// not exist or has no commit yet (a branch without commits). A ref that
// exists but whose commit cannot be read, because it is missing (as in a
// partial clone) or is not a commit, is an environment error, never ""
// (PD-26): the ref is resolved first, which reads no object, and only then
// is its object read.
func CommitOf(ctx context.Context, dir, ref string) (string, error) {
	out, code, err := runObjectsCode(ctx, dir, "rev-parse", "-q", "--verify", ref)
	if err != nil {
		return "", err
	}
	if code == 1 {
		return "", nil
	}
	id := strings.TrimSpace(string(out))
	if !objectID.MatchString(id) {
		return "", apperr.New(apperr.Environment, "git rev-parse printed something other than an object ID")
	}
	out, code, err = runObjectsCode(ctx, dir, "rev-parse", "-q", "--verify", id+"^{commit}")
	if err != nil {
		return "", err
	}
	commit := strings.TrimSpace(string(out))
	if code == 1 || !objectID.MatchString(commit) {
		return "", apperr.New(apperr.Environment,
			"the commit a ref points to cannot be read from the repository: it is missing (Specmetry never fetches objects, as from a partial clone's remote) or is not a commit; run git log in this directory to see why")
	}
	return commit, nil
}

// ListFiles lists the files of worktree that git sees: tracked files and
// untracked files that are not ignored, relative to the worktree root with
// forward slashes, each once. A sparse index is not expanded: a directory
// outside the sparse checkout is listed as itself, with a trailing slash
// (git would otherwise read its trees and rewrite the timestamps of object
// files, PD-26). It may still read objects (a .gitignore outside the
// sparse checkout is read from the index), so it runs only with a git that
// never fetches.
func ListFiles(ctx context.Context, worktree string) ([]string, error) {
	git, err := listGit(ctx)
	if err != nil {
		return nil, err
	}
	out, code, err := runWith(ctx, git, worktree, "ls-files", "-z", "--sparse", "--cached", "--others", "--exclude-standard", "--full-name", "--", ":/")
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, failed("ls-files", code)
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

// objectID matches a full object or commit ID (SHA-1 or SHA-256), as
// git rev-parse prints one.
var objectID = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

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
// runs, the version check and the object read probe included: no system
// or global configuration (an empty file instead; git maps /dev/null on
// Windows too); trace2 off, which a git too old for GIT_CONFIG_GLOBAL
// would otherwise enable from the user's global configuration while it
// only prints its version (codex-review-016 B1; the variables override
// trace2.*Target); no optional locks, so reading never writes the index;
// no prompts; the .git entry that findDotGit found by walking up without a
// filesystem limit; and no object is ever fetched (codex-review-020 B1,
// PD-26). A partial clone fetches a missing object from its promisor
// remote on demand, which could reach a network, run the remote's
// upload-pack or write the repository's configuration. GIT_NO_LAZY_FETCH
// stops that in a git that honours it, and commands that may read an
// object run only with a git shown to (objects.go). For every version
// from 2.32 on, GIT_ALLOW_PROTOCOL set and empty is an allow list with no
// protocol in it: every transport (file, ssh, git, http and https, ext,
// any remote helper) is refused before it connects or runs a program,
// whatever the protocol.* configuration says. It must be set, not left
// out: unset means git's default policy, which allows most transports
// (codex-review-021 B1-2; an earlier value named a protocol, so a remote
// helper of that name was allowed).
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
	"GIT_NO_LAZY_FETCH=1",
	"GIT_ALLOW_PROTOCOL=",
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

// gitBinary is the file PATH finds for git. What gitx found out about a
// git, its version and whether it may read objects, is kept only while
// PATH finds the same file, unchanged; any other git is checked anew. The
// file found is the one run.
type gitBinary struct {
	path     string
	size     int64
	modTime  int64
	identity fileKey
}

var (
	stateMu sync.Mutex
	// state is what gitx knows about bin. Only what a check showed is
	// kept: a check that failed, was cancelled or found nothing out is
	// made again next time.
	state struct {
		bin       gitBinary
		versionOK bool
		objects   objectSafety
		sparse    sparseListing
		scanEmpty bool
	}
)

// findGit returns the git that PATH finds.
func findGit() (gitBinary, error) {
	p, err := exec.LookPath("git")
	if err != nil {
		return gitBinary{}, apperr.New(apperr.Environment, "git is required but was not found on PATH")
	}
	f, err := os.Open(p)
	if err != nil {
		return gitBinary{}, apperr.New(apperr.Environment, "cannot identify git")
	}
	defer func() { _ = f.Close() }()
	identity, _, err := keyOf(f)
	if err != nil {
		return gitBinary{}, apperr.New(apperr.Environment, "cannot identify git")
	}
	fi, err := f.Stat()
	if err != nil {
		return gitBinary{}, apperr.New(apperr.Environment, "cannot run git ("+osProblem(err)+")")
	}
	return gitBinary{path: p, size: fi.Size(), modTime: fi.ModTime().UnixNano(), identity: identity}, nil
}

// gitToRun returns the git to run after refusing one too old to be
// isolated.
func gitToRun(ctx context.Context) (string, error) {
	stateMu.Lock()
	defer stateMu.Unlock()
	return gitToRunLocked(ctx)
}

// gitToRunLocked is gitToRun with stateMu held.
func gitToRunLocked(ctx context.Context) (string, error) {
	bin, err := findGit()
	if err != nil {
		return "", err
	}
	if bin != state.bin {
		state.bin, state.versionOK, state.objects, state.sparse, state.scanEmpty = bin, false, unprobed, sparseUnprobed, false
	}
	if !state.versionOK {
		out, code, err := exec1(ctx, bin.path, "", "--version")
		if err != nil {
			return "", err
		}
		if code != 0 || !supportedVersion(string(out)) {
			return "", apperr.New(apperr.Environment, "git "+minimumVersion+" or newer is required")
		}
		state.versionOK = true
	}
	return bin.path, nil
}

// run runs a git command that reads no object and must succeed.
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

// runCode runs a git command that reads no object in dir, after checking
// git's version. A non-zero exit status other than 1 is an error; status 1
// is returned for the caller to interpret.
func runCode(ctx context.Context, dir string, args ...string) ([]byte, int, error) {
	git, err := gitToRun(ctx)
	if err != nil {
		return nil, -1, err
	}
	return runWith(ctx, git, dir, args...)
}

func runWith(ctx context.Context, git, dir string, args ...string) ([]byte, int, error) {
	out, code, err := exec1(ctx, git, dir, append(slices.Clone(globalArgs), args...)...)
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
func exec1(ctx context.Context, git, dir string, args ...string) ([]byte, int, error) {
	var stdout bytes.Buffer
	code, err := execTo(ctx, git, dir, &stdout, args...)
	if err != nil {
		return nil, -1, err
	}
	return stdout.Bytes(), code, nil
}

// execTo is exec1 with git's output written to stdout as it comes.
func execTo(ctx context.Context, git, dir string, stdout io.Writer, args ...string) (int, error) {
	return command(ctx, git, dir, filterEnv(os.Environ()), stdout, args...)
}

// command runs git once in dir with env. Its output goes to stdout, or
// nowhere when stdout is nil; its messages go nowhere.
func command(ctx context.Context, git, dir string, env []string, stdout io.Writer, args ...string) (int, error) {
	return commandIO(ctx, git, dir, env, nil, stdout, nil, args...)
}

// testHookCommand brackets the actual Git process for deterministic lifecycle tests.
var testHookCommand func(stage string, pid int, args []string)
var testHookCommandSetup func(cmd *exec.Cmd)

// commandIO is command with stdin as git's input, and git's messages
// written to stderr, or nowhere when it is nil.
func commandIO(ctx context.Context, git, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer, args ...string) (int, error) {
	cmd := exec.CommandContext(ctx, git, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if testHookCommandSetup != nil {
		testHookCommandSetup(cmd)
	}
	var err error
	if testHookCommand == nil {
		err = cmd.Run()
	} else {
		err = cmd.Start()
		if err == nil {
			testHookCommand("started", cmd.Process.Pid, slices.Clone(cmd.Args))
			err = cmd.Wait()
			if testHookCommand != nil {
				testHookCommand("waited", cmd.Process.Pid, slices.Clone(cmd.Args))
			}
		}
	}
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

// PartialClone reports whether the repository of worktree is a partial
// clone: one with a promisor remote, from which git fetches the objects it
// lacks on demand. Either setting makes one: extensions.partialClone,
// which names a remote and counts whatever its value; or a
// remote.<name>.promisor that is true. promisor is read by git itself as
// a boolean (git config --bool), so every value git takes for true counts
// (git 2.32 takes promisor = 2 for true; codex-review-021 B1-1). A value
// git cannot read as a boolean makes this fail (exit 6) rather than answer
// "not a partial clone". Only the configuration is read, never an object,
// and errors never quote it.
func PartialClone(ctx context.Context, worktree string) (bool, error) {
	_, code, err := runCode(ctx, worktree, "config", "--get-regexp", `^extensions\.partialclone$`)
	if err != nil {
		return false, unreadable(ctx, err)
	}
	if code == 0 {
		return true, nil
	}
	out, code, err := runCode(ctx, worktree, "config", "--bool", "--get-regexp", `^remote\..*\.promisor$`)
	if err != nil {
		return false, unreadable(ctx, err)
	}
	if code == 1 {
		return false, nil // no remote has the setting
	}
	for _, line := range strings.Split(strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n")), "\n") {
		_, value, _ := strings.Cut(line, " ")
		switch value {
		case "true":
			return true, nil
		case "false":
		default:
			return false, unreadable(ctx, nil)
		}
	}
	return false, nil
}

// unreadable is the error of a partial clone check that could not be made:
// no object is read then either.
func unreadable(ctx context.Context, err error) error {
	if ctx.Err() != nil && err != nil {
		return err
	}
	return apperr.New(apperr.Environment,
		"cannot tell whether this repository is a partial clone (git cannot read its extensions.partialClone or remote.<name>.promisor setting), so Specmetry reads none of its objects; check the repository's configuration")
}
