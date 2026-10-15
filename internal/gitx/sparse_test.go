package gitx

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// PD-26: a sparse index is listed without being expanded. Expanding it,
// git reads the trees it names and rewrites the timestamps of object files
// (git 2.56 and 2.43 both do, without --sparse). A directory outside the
// sparse checkout is listed as itself, its trees unread, so one that a
// partial clone lacks is not fetched: the promisor remote does not run and
// nothing in the repository changes, timestamps included.
func TestListFilesWithASparseIndex(t *testing.T) {
	requireSafeGit(t)
	for _, missing := range []bool{false, true} {
		repo, _, treeB := sparseRepo(t)
		ran := ""
		if missing {
			ran = makePartial(t, repo)
			removeObject(t, repo, treeB)
		}
		gitDir := filepath.Join(repo.Dir, ".git")
		before := testutil.SettledSnapshot(t, gitDir)
		files, err := ListFiles(ctx, repo.Dir)
		if err != nil {
			t.Fatalf("tree of b/ missing = %v: %v", missing, err)
		}
		slices.Sort(files)
		if !slices.Equal(files, []string{"a/x.txt", "b/"}) {
			t.Errorf("tree of b/ missing = %v: ListFiles = %q", missing, files)
		}
		if missing && ranAt(ran) {
			t.Error("the promisor remote's upload-pack ran")
		}
		testutil.AssertTreeUnchanged(t, gitDir, before)
	}
}

// PD-26: in a sparse checkout, git reads a .gitignore outside it from the
// index: an object read. One whose blob a partial clone lacks is not
// fetched, and nothing in the repository changes (git then applies none of
// its patterns, as measured with git 2.56 and 2.43).
func TestListFilesDoesNotFetchAGitignore(t *testing.T) {
	requireSafeGit(t)
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("a/x.txt", "one\n")
	repo.WriteFile("a/.gitignore", "secret.txt\n")
	head := repo.Commit("init")
	repo.Git("config", "core.sparseCheckout", "true")
	info := filepath.Join(repo.Dir, ".git", "info")
	if err := os.MkdirAll(info, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(info, "sparse-checkout"), []byte("/*\n!/a/.gitignore\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo.Git("read-tree", "-mu", "HEAD")
	if _, err := os.Lstat(filepath.Join(repo.Dir, "a", ".gitignore")); err == nil {
		t.Skip("this git left a/.gitignore in the worktree")
	}
	repo.WriteFile("a/secret.txt", "s\n")
	blob := repo.Git("rev-parse", head+":a/.gitignore")
	ran := makePartial(t, repo)
	removeObject(t, repo, blob)
	gitDir := filepath.Join(repo.Dir, ".git")
	before := testutil.SettledSnapshot(t, gitDir)
	if _, err := ListFiles(ctx, repo.Dir); err != nil && apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("ListFiles = %v", err)
	}
	if ranAt(ran) {
		t.Error("the promisor remote's upload-pack ran")
	}
	testutil.AssertTreeUnchanged(t, gitDir, before)
}

// PD-26: with a git whose ls-files does not know --sparse (a stand-in that
// refuses the option as such a git does), no file is listed: such a git
// reads and expands a sparse index before it looks at its options, so the
// option is tried only outside the user's repository, once, and
// ListFiles is refused (exit 6) without running ls-files in the
// repository, which is left as it was. Reading objects goes on: the git
// still never fetches.
func TestListFilesNeedsLsFilesSparse(t *testing.T) {
	requireSafeGit(t)
	repo, head, _ := sparseRepo(t)
	gitDir := filepath.Join(repo.Dir, ".git")
	dir, env, calls := testutil.PassthroughGit(t, testutil.PassWithoutLsFilesSparse)
	useGit(t, dir, env)
	before := testutil.SettledSnapshot(t, gitDir)
	for i := range 2 {
		_, err := ListFiles(ctx, repo.Dir)
		if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "does not know --sparse") {
			t.Fatalf("ListFiles %d = %v", i+1, err)
		}
	}
	testutil.AssertTreeUnchanged(t, gitDir, before)
	if c, err := CommitOf(ctx, repo.Dir, "HEAD"); err != nil || c != head {
		t.Errorf("CommitOf = %q, %v", c, err)
	}
	tries := 0
	for _, c := range calls() {
		if !slices.Contains(c.Args, "ls-files") {
			continue
		}
		if slices.Contains(c.Args, "--cached") {
			t.Errorf("ls-files ran in the repository: %q", c.Args)
		}
		tries++
	}
	if tries != 1 {
		t.Errorf("ls-files --sparse was tried %d times, want once", tries)
	}
}
