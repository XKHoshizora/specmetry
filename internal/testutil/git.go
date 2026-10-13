package testutil

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// GitRepo is a throwaway git repository isolated from the user's git
// configuration (no global/system config, no autocrlf, fixed identity).
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
	root := t.TempDir()
	gitHome := filepath.Join(root, "git-home")
	if err := os.MkdirAll(gitHome, 0o700); err != nil {
		t.Fatal(err)
	}
	globalConfig := filepath.Join(gitHome, "gitconfig")
	if err := os.WriteFile(globalConfig, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	r := &GitRepo{
		t:   t,
		Dir: filepath.Join(root, "repo"),
		env: append(os.Environ(),
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_CONFIG_GLOBAL="+globalConfig,
			"HOME="+gitHome,
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
	r.Git("init", "-q", "-b", "main")
	r.Git("config", "core.autocrlf", "false")
	return r
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
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = r.env
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
	r.Git("add", "-A")
	r.Git("commit", "-q", "--allow-empty", "-m", message)
	return r.Git("rev-parse", "HEAD")
}

// AddWorktree creates a linked worktree on a new branch and returns its path.
func (r *GitRepo) AddWorktree(branch string) string {
	r.t.Helper()
	p := filepath.Join(filepath.Dir(r.Dir), "wt-"+branch)
	r.Git("worktree", "add", "-q", "-b", branch, p)
	return p
}
