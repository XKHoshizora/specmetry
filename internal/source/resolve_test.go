package source_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/source"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

func writeFile(t *testing.T, p, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func where(t *testing.T, dir string) project.Where {
	t.Helper()
	w, err := project.Locate(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// A source inside the main worktree is in_repo with a slash-separated
// path relative to it, from wherever it is named; anything else is
// external with its canonical absolute path (data-model §4.3).
func TestResolve(t *testing.T) {
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("docs/client brief 需求.md", "brief")
	repo.Commit("init")
	w := where(t, repo.Dir)
	outsideDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside := writeFile(t, filepath.Join(outsideDir, "外部 文档", "客户 需求.md"), "external")

	cases := []struct {
		arg, cwd       string
		location, path string
	}{
		{"docs/client brief 需求.md", repo.Dir, "in_repo", "docs/client brief 需求.md"},
		{"client brief 需求.md", filepath.Join(repo.Dir, "docs"), "in_repo", "docs/client brief 需求.md"},
		{filepath.Join(repo.Dir, "docs", "client brief 需求.md"), outsideDir, "in_repo", "docs/client brief 需求.md"},
		{outside, repo.Dir, "external", outside},
		{filepath.Join("..", filepath.Base(outsideDir), "外部 文档", "客户 需求.md"), filepath.Join(filepath.Dir(outsideDir), filepath.Base(outsideDir)), "external", outside},
	}
	for _, c := range cases {
		got, err := source.Resolve(ctx, c.arg, c.cwd, w)
		if err != nil {
			t.Errorf("Resolve(%q in %s): %v", c.arg, c.cwd, err)
			continue
		}
		if string(got.Location) != c.location || got.Path != c.path {
			t.Errorf("Resolve(%q in %s) = %+v, want %s %s", c.arg, c.cwd, got, c.location, c.path)
		}
		if _, err := os.Stat(got.File); err != nil {
			t.Errorf("Resolve(%q).File %s: %v", c.arg, got.File, err)
		}
	}
}

// MVP-SRC-05 and the paths that are not a source: invalid input (exit 2).
func TestResolveRefuses(t *testing.T) {
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("docs/brief.md", "brief")
	repo.Commit("init")
	wt := repo.AddWorktree("feature-x")
	writeFile(t, filepath.Join(wt, "docs", "only-here.md"), "x")
	w := where(t, repo.Dir)
	for _, arg := range []string{
		filepath.Join(repo.Dir, "docs", "missing.md"),
		filepath.Join(repo.Dir, "docs"),
		filepath.Join(repo.Dir, ".git", "HEAD"),
		filepath.Join(wt, "docs", "only-here.md"),
	} {
		if _, err := source.Resolve(ctx, arg, repo.Dir, where(t, wt)); apperr.ExitCode(err) != apperr.ExitInvalidInput {
			t.Errorf("Resolve(%s) = %v", arg, err)
		}
	}
	// The same file named from the main worktree (codex-review-018 B3).
	if _, err := source.Resolve(ctx, filepath.Join(wt, "docs", "only-here.md"), repo.Dir, w); apperr.ExitCode(err) != apperr.ExitInvalidInput {
		t.Errorf("Resolve from the main worktree = %v", err)
	}
	if _, err := source.Resolve(ctx, "docs/brief.md", repo.Dir, w); err != nil {
		t.Fatal(err)
	}
}

// A symbolic link is resolved: the source is where its content is.
func TestResolveFollowsLinks(t *testing.T) {
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("README.md", "x")
	repo.Commit("init")
	outsideDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := writeFile(t, filepath.Join(outsideDir, "brief.md"), "brief")
	if err := os.Symlink(target, filepath.Join(repo.Dir, "linked.md")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("cannot create symbolic links: %v", err)
		}
		t.Fatal(err)
	}
	got, err := source.Resolve(ctx, "linked.md", repo.Dir, where(t, repo.Dir))
	if err != nil || got.Location != source.External || got.Path != target {
		t.Fatalf("Resolve = %+v, %v", got, err)
	}
}

// Outside git the project directory is the root.
func TestResolveOutsideGit(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "docs", "brief.md"), "brief")
	if err := project.WriteMarker(root, project.Marker{ProjectID: idA, Name: "plain"}); err != nil {
		t.Fatal(err)
	}
	got, err := source.Resolve(ctx, "brief.md", filepath.Join(root, "docs"), where(t, filepath.Join(root, "docs")))
	if err != nil || got.Location != source.InRepo || got.Path != "docs/brief.md" {
		t.Fatalf("Resolve = %+v, %v", got, err)
	}
}
