package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// Locate finds a directory's project root without the database: the main
// worktree of a git repository, or outside git the nearest directory with
// a marker (or the directory itself).
func TestLocate(t *testing.T) {
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("docs/brief.md", "brief")
	repo.Commit("init")
	wt := repo.AddWorktree("feature-x")
	gitDir := filepath.Join(repo.Dir, ".git")
	// A linked worktree's own git directory is named after its directory.
	wtGitDir := filepath.Join(gitDir, "worktrees", filepath.Base(wt))
	for dir, want := range map[string]project.Where{
		repo.Dir:                        {Git: true, Root: repo.Dir, Current: repo.Dir, CommonDir: gitDir, GitDir: gitDir},
		filepath.Join(repo.Dir, "docs"): {Git: true, Root: repo.Dir, Current: repo.Dir, CommonDir: gitDir, GitDir: gitDir},
		wt:                              {Git: true, Root: repo.Dir, Current: wt, CommonDir: gitDir, GitDir: wtGitDir},
	} {
		if got, err := project.Locate(ctx, dir); err != nil || got != want {
			t.Errorf("Locate(%s) = %+v, %v; want %+v", dir, got, err, want)
		}
	}

	plain, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(plain, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := project.Locate(ctx, sub); err != nil || got != (project.Where{Root: sub, Current: sub}) {
		t.Errorf("Locate without a marker = %+v, %v", got, err)
	}
	if err := project.WriteMarker(plain, project.Marker{ProjectID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4C1", Name: "plain"}); err != nil {
		t.Fatal(err)
	}
	if got, err := project.Locate(ctx, sub); err != nil || got != (project.Where{Root: plain, Current: plain}) {
		t.Errorf("Locate with a marker above = %+v, %v", got, err)
	}
}

// The source type whitelist (data-model §4.3), shared with the candidates.
func TestIsDocumentType(t *testing.T) {
	for _, name := range []string{"a.md", "B.MD", "x.markdown", "x.txt", "x.rst", "x.adoc", "x.pdf", "x.docx", "x.doc", "x.odt", "x.rtf", "x.html", "x.htm", "x.eml", "客户 需求.md"} {
		if !project.IsDocumentType(name) {
			t.Errorf("%s refused", name)
		}
	}
	for _, name := range []string{"login.go", "x.json", "README", "x.md.go", ".md"} {
		if project.IsDocumentType(name) {
			t.Errorf("%s accepted", name)
		}
	}
}
