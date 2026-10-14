package gitx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
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

func TestFilterEnv(t *testing.T) {
	in := []string{"PATH=/bin", "GIT_DIR=/x", "git_work_tree=/y", "GIT_AUTHOR_NAME=a", "GIT_INDEX_FILE=/i", "HOME=/h", "GIT_CEILING_DIRECTORIES=/c"}
	got := filterEnv(in)
	for _, kept := range []string{"PATH=/bin", "GIT_AUTHOR_NAME=a", "HOME=/h", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0"} {
		if !slices.Contains(got, kept) {
			t.Errorf("%s missing from %q", kept, got)
		}
	}
	for _, v := range got {
		name, _, _ := strings.Cut(v, "=")
		switch strings.ToUpper(name) {
		case "GIT_DIR", "GIT_INDEX_FILE", "GIT_CEILING_DIRECTORIES":
			t.Errorf("%s kept", v)
		case "GIT_WORK_TREE":
			if runtime.GOOS == "windows" {
				t.Errorf("%s kept (variable names are case-insensitive on Windows)", v)
			}
		}
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
