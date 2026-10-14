package gitx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

var ctx = context.Background()

func committedRepo(t *testing.T) *testutil.GitRepo {
	t.Helper()
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("README.md", "readme")
	repo.Commit("init")
	return repo
}

func mkdir(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(parts...)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLocateMainWorktree(t *testing.T) {
	repo := committedRepo(t)
	want := Location{
		Worktree:  repo.Dir,
		GitDir:    filepath.Join(repo.Dir, ".git"),
		CommonDir: filepath.Join(repo.Dir, ".git"),
		Main:      repo.Dir,
	}
	for _, dir := range []string{repo.Dir, mkdir(t, repo.Dir, "a", "b")} {
		got, err := Locate(ctx, dir)
		if err != nil || got != want {
			t.Errorf("Locate(%s) = %+v, %v; want %+v", dir, got, err, want)
		}
	}
}

func TestLocateLinkedWorktree(t *testing.T) {
	repo := committedRepo(t)
	wt := repo.AddWorktree("feature-x")
	got, err := Locate(ctx, wt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Worktree != wt || got.Main != repo.Dir || got.CommonDir != filepath.Join(repo.Dir, ".git") {
		t.Fatalf("Locate = %+v", got)
	}
	if got.GitDir == got.CommonDir || filepath.Dir(filepath.Dir(got.GitDir)) != got.CommonDir {
		t.Fatalf("linked worktree git dir = %s", got.GitDir)
	}
}

func TestLocateOutsideRepository(t *testing.T) {
	if _, err := Locate(ctx, t.TempDir()); !errors.Is(err, ErrNotRepository) {
		t.Fatalf("Locate = %v, want ErrNotRepository", err)
	}
}

// Inside a git directory or a bare repository there is no worktree: an
// environment error, never "not a repository" (which would make it a
// non-git project).
func TestLocateRefusesGitDirectories(t *testing.T) {
	repo := committedRepo(t)
	bare := filepath.Join(t.TempDir(), "bare.git")
	repo.Git("clone", "-q", "--bare", repo.Dir, bare)
	for _, dir := range []string{filepath.Join(repo.Dir, ".git"), filepath.Join(repo.Dir, ".git", "refs"), bare, filepath.Join(bare, "refs")} {
		_, err := Locate(ctx, dir)
		if err == nil || errors.Is(err, ErrNotRepository) || apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Errorf("Locate(%s) = %v", dir, err)
		}
	}
}

// The location comes from the directory, not from variables that point git
// at another repository.
func TestLocateIgnoresRepositoryVariables(t *testing.T) {
	repo, other := committedRepo(t), committedRepo(t)
	t.Setenv("GIT_DIR", filepath.Join(other.Dir, ".git"))
	t.Setenv("GIT_WORK_TREE", other.Dir)
	t.Setenv("GIT_COMMON_DIR", filepath.Join(other.Dir, ".git"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(other.Dir, ".git", "index"))
	t.Setenv("GIT_CEILING_DIRECTORIES", repo.Dir)
	got, err := Locate(ctx, mkdir(t, repo.Dir, "sub"))
	if err != nil || got.Worktree != repo.Dir || got.CommonDir != filepath.Join(repo.Dir, ".git") {
		t.Fatalf("Locate = %+v, %v", got, err)
	}
	files, err := ListFiles(ctx, repo.Dir)
	if err != nil || !reflect.DeepEqual(files, []string{"README.md"}) {
		t.Fatalf("ListFiles = %q, %v", files, err)
	}
}

// codex-review-015 B1: git gets none of the inherited GIT_* variables —
// configuration injection (GIT_CONFIG_COUNT/KEY/VALUE, GIT_CONFIG_PARAMETERS,
// GIT_CONFIG_GLOBAL), tracing, repository location, exec path — whatever
// their letter case, and only the controlled ones are added back: system and
// global configuration replaced by an empty file, no optional locks, no
// prompts.
func TestFilterEnv(t *testing.T) {
	in := []string{
		"PATH=/bin", "HOME=/h", "LANG=C",
		"GIT_DIR=/x", "git_work_tree=/y", "GIT_AUTHOR_NAME=a", "GIT_INDEX_FILE=/i", "GIT_CEILING_DIRECTORIES=/c",
		"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=core.fsmonitor", "GIT_CONFIG_VALUE_0=x", "GIT_CONFIG_PARAMETERS='a'='b'",
		"GIT_CONFIG_GLOBAL=/g", "GIT_CONFIG_SYSTEM=/s", "GIT_CONFIG_NOSYSTEM=0", "GIT_TRACE=/t", "GIT_TRACE2_EVENT=/e",
		"GIT_EXEC_PATH=/p", "GIT_OPTIONAL_LOCKS=1", "Git_Trace_Setup=/u",
	}
	got := filterEnv(in)
	want := []string{
		"PATH=/bin", "HOME=/h", "LANG=C",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0", "GIT_DISCOVERY_ACROSS_FILESYSTEM=1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterEnv =\n%q\nwant\n%q", got, want)
	}
}

// codex-review-015 B3: an error never carries git's own message, which can
// quote configuration values, nor the directory git ran in.
func TestErrorsDoNotQuoteGit(t *testing.T) {
	repo := committedRepo(t)
	canary := testutil.NewCanary(t)
	_, err := run(ctx, repo.Dir, "rev-parse", "--verify", canary)
	if err == nil || apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("run = %v", err)
	}
	if msg := err.Error(); strings.Contains(msg, canary) || strings.Contains(msg, repo.Dir) || !strings.Contains(msg, "rev-parse") {
		t.Fatalf("error = %q", msg)
	}

	bad := testutil.NewGitRepo(t)
	bad.Git("config", "core.bare", canary)
	_, err = Locate(ctx, bad.Dir)
	if err == nil || apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("Locate with an invalid configuration = %v", err)
	}
	if msg := err.Error(); strings.Contains(msg, canary) || strings.Contains(msg, bad.Dir) {
		t.Fatalf("error = %q", msg)
	}
}

// codex-review-015 B1: git 2.32 is the oldest that reads GIT_CONFIG_GLOBAL
// and GIT_CONFIG_SYSTEM, without which the user's configuration cannot be
// kept out.
func TestParseGitVersion(t *testing.T) {
	cases := map[string]bool{
		"git version 2.32.0":                 true,
		"git version 2.56.0.windows.2":       true,
		"git version 2.39.5 (Apple Git-154)": true,
		"git version 3.0.0":                  true,
		"git version 2.31.8":                 false,
		"git version 1.99.0":                 false,
		"not git":                            false,
	}
	for out, want := range cases {
		if got := supportedVersion(out); got != want {
			t.Errorf("supportedVersion(%q) = %v, want %v", out, got, want)
		}
	}
}

// codex-review-015 N2: a cancelled context stops git.
func TestCanceledContextStopsGit(t *testing.T) {
	repo := committedRepo(t)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := ListFiles(canceled, repo.Dir); err == nil {
		t.Fatal("ListFiles ran with a cancelled context")
	}
}

func TestHeadBranchAndCommit(t *testing.T) {
	repo := committedRepo(t)
	head := repo.Git("rev-parse", "HEAD")
	if ref, err := HeadBranch(ctx, repo.Dir); err != nil || ref != "refs/heads/main" {
		t.Fatalf("HeadBranch = %q, %v", ref, err)
	}
	if sha, err := CommitOf(ctx, repo.Dir, "refs/heads/main"); err != nil || sha != head {
		t.Fatalf("CommitOf = %q, %v; want %s", sha, err, head)
	}
	repo.Git("checkout", "-q", "--detach")
	if ref, err := HeadBranch(ctx, repo.Dir); err != nil || ref != "" {
		t.Fatalf("detached HeadBranch = %q, %v", ref, err)
	}
	empty := testutil.NewGitRepo(t)
	if ref, err := HeadBranch(ctx, empty.Dir); err != nil || ref != "refs/heads/main" {
		t.Fatalf("unborn HeadBranch = %q, %v", ref, err)
	}
	if sha, err := CommitOf(ctx, empty.Dir, "refs/heads/main"); err != nil || sha != "" {
		t.Fatalf("unborn CommitOf = %q, %v", sha, err)
	}
}

func TestGitPath(t *testing.T) {
	repo := committedRepo(t)
	wt := repo.AddWorktree("feature-x")
	want := filepath.Join(repo.Dir, ".git", "info", "exclude")
	for _, dir := range []string{repo.Dir, wt} {
		if got, err := GitPath(ctx, dir, "info/exclude"); err != nil || got != want {
			t.Errorf("GitPath(%s) = %s, %v; want %s", dir, got, err, want)
		}
	}
}
