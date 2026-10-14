package testutil

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// GitRepo is a throwaway git repository isolated from the user's git setup:
// git runs without any inherited GIT_* variable, without system or global
// configuration, templates or hooks, with a fixed identity and SHA-1
// objects (codex-review-012 B1).
type GitRepo struct {
	t   testing.TB
	Dir string
	env []string
}

// NewGitRepo creates an empty repository on branch main.
func NewGitRepo(t testing.TB) *GitRepo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	// git prints canonical paths. Resolve the temp directory the same way
	// (macOS: /var -> /private/var; Windows runners: 8.3 short names such
	// as RUNNER~1) so Dir compares equal to git's output.
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	gitHome := filepath.Join(root, "git-home")
	if err := os.MkdirAll(gitHome, 0o700); err != nil {
		t.Fatal(err)
	}
	globalConfig := filepath.Join(gitHome, "gitconfig")
	if err := os.WriteFile(globalConfig, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	template := filepath.Join(root, "git-template")
	if err := os.MkdirAll(template, 0o700); err != nil {
		t.Fatal(err)
	}
	r := &GitRepo{
		t:   t,
		Dir: filepath.Join(root, "repo"),
		env: append(inheritedEnvWithoutGit(),
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_CONFIG_GLOBAL="+globalConfig,
			"HOME="+gitHome,
			"XDG_CONFIG_HOME="+gitHome,
			// An empty template: nothing (hooks included) is copied into .git.
			"GIT_TEMPLATE_DIR="+template,
			// Never look for a repository above the temporary directory.
			"GIT_CEILING_DIRECTORIES="+root,
			"GIT_TERMINAL_PROMPT=0",
			"GIT_AUTHOR_NAME=Specmetry Test",
			"GIT_AUTHOR_EMAIL=test@specmetry.invalid",
			"GIT_COMMITTER_NAME=Specmetry Test",
			"GIT_COMMITTER_EMAIL=test@specmetry.invalid",
		),
	}
	if err := os.MkdirAll(r.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("init", "-q", "-b", "main", "--object-format=sha1")
	r.requireOwnGitDir()
	r.Git("config", "core.autocrlf", "false")
	return r
}

// inheritedEnvWithoutGit returns the process environment without any
// variable git reads to locate or configure a repository: every GIT_*
// variable (GIT_DIR, GIT_WORK_TREE, GIT_INDEX_FILE, GIT_OBJECT_DIRECTORY,
// GIT_DEFAULT_HASH, GIT_CONFIG_*, GIT_TEMPLATE_DIR, ...) and the HOME and
// XDG_CONFIG_HOME it takes user configuration from. NewGitRepo sets the
// ones it needs itself. Names compare case-insensitively, as on Windows.
func inheritedEnvWithoutGit() []string {
	var out []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		upper := strings.ToUpper(name)
		if strings.HasPrefix(upper, "GIT_") || upper == "HOME" || upper == "XDG_CONFIG_HOME" {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// requireOwnGitDir stops the test unless git resolves the repository to
// Dir/.git: no configuration change or commit may reach another repository.
func (r *GitRepo) requireOwnGitDir() {
	r.t.Helper()
	got := filepath.Clean(filepath.FromSlash(r.Git("rev-parse", "--absolute-git-dir")))
	want := filepath.Join(r.Dir, ".git")
	if got != want && (runtime.GOOS != "windows" || !strings.EqualFold(got, want)) {
		r.t.Fatalf("git uses %s, not the test repository %s", got, want)
	}
}

// Git runs git in the repository and returns trimmed stdout.
func (r *GitRepo) Git(args ...string) string {
	r.t.Helper()
	return r.GitIn(r.Dir, args...)
}

// GitIn runs git in dir (e.g. a linked worktree) with the repo's isolated
// environment.
func (r *GitRepo) GitIn(dir string, args ...string) string {
	r.t.Helper()
	return r.gitEnvIn(dir, nil, args...)
}

// gitEnvIn runs git in dir with the repo's environment plus extra
// variables, which take precedence.
func (r *GitRepo) gitEnvIn(dir string, extra []string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(slices.Clip(r.env), extra...)
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			stderr = string(ee.Stderr)
		}
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, stderr)
	}
	return strings.TrimSpace(string(out))
}

// WriteFile writes a file relative to the repository root.
func (r *GitRepo) WriteFile(rel, content string) {
	r.t.Helper()
	p := filepath.Join(r.Dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// Commit stages everything and commits, returning the commit SHA.
func (r *GitRepo) Commit(message string) string {
	r.t.Helper()
	r.requireOwnGitDir()
	r.Git("add", "-A")
	r.Git("commit", "-q", "--allow-empty", "-m", message)
	return r.Git("rev-parse", "HEAD")
}

// CommitAt is Commit with a fixed author and committer date, so the commit
// ID depends only on the content, the parents and the message.
func (r *GitRepo) CommitAt(message string, date time.Time) string {
	r.t.Helper()
	stamp := strconv.FormatInt(date.Unix(), 10) + " +0000"
	r.requireOwnGitDir()
	r.Git("add", "-A")
	r.gitEnvIn(r.Dir, []string{"GIT_AUTHOR_DATE=" + stamp, "GIT_COMMITTER_DATE=" + stamp},
		"commit", "-q", "--allow-empty", "-m", message)
	return r.Git("rev-parse", "HEAD")
}

// AddWorktree creates a linked worktree on a new branch and returns its path.
func (r *GitRepo) AddWorktree(branch string) string {
	r.t.Helper()
	p := filepath.Join(filepath.Dir(r.Dir), "wt-"+branch)
	r.Git("worktree", "add", "-q", "-b", branch, p)
	return p
}

// WorktreeFor returns a linked worktree that checks out an existing branch,
// adding it on the first call. A branch can be checked out in one worktree
// only, so later calls return the same path.
func (r *GitRepo) WorktreeFor(branch string) string {
	r.t.Helper()
	p := filepath.Join(filepath.Dir(r.Dir), "wt-"+strings.ReplaceAll(branch, "/", "-"))
	if _, err := os.Stat(p); err == nil {
		return p
	}
	r.Git("worktree", "add", "-q", p, branch)
	return p
}
