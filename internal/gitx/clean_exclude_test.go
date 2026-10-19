package gitx

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// definitionStatus runs the definition's status (data-model §1, D1) on the
// repository itself, in the controlled environment: the reference Clean
// must agree with. It may run the repository's filters, which these tests
// make harmless (cat).
func definitionStatus(t *testing.T, worktree string) string {
	t.Helper()
	r, err := call{label: "status", dir: worktree, objects: true, args: []string{"status", "--porcelain", "--untracked-files=all",
		"--ignore-submodules=none", "--", ".", ":(exclude).specmetry"}}.run(ctx)
	if err != nil || r.code != 0 {
		t.Fatalf("the definition's status: %v, exit %d", err, r.code)
	}
	return string(r.out)
}

// excludeRepo is a repository with the same filtered file inside
// .specmetry/ and inside .specmetry-not/ (a name that only starts the
// same), committed, settled, then a harmless local driver (cat) for them.
func excludeRepo(t *testing.T) *testutil.GitRepo {
	t.Helper()
	repo := baseRepo(t)
	repo.WriteFile(".gitattributes", ".specmetry/probe.txt filter=r058\n.specmetry-not/probe.txt filter=r058\n")
	repo.WriteFile(".specmetry/probe.txt", "original\n")
	repo.WriteFile(".specmetry/plain.txt", "plain\n")
	repo.WriteFile(".specmetry-not/probe.txt", "original\n")
	repo.Commit("probes")
	at := time.Now().Add(-time.Hour)
	for _, f := range []string{".specmetry/probe.txt", ".specmetry/plain.txt", ".specmetry-not/probe.txt", "README.md", "dir/x.txt", ".gitignore", ".gitattributes"} {
		touch(t, filepath.Join(repo.Dir, filepath.FromSlash(f)), at)
	}
	repo.Git("update-index", "--refresh")
	repo.Git("config", "filter.r058.clean", "cat")
	return repo
}

// Review 058 B3, as reported, and its neighbours: what the definition's
// pathspec leaves out (.specmetry, the directory and everything in it, at
// the top of the superproject) changes nothing Clean says, through any of
// its reads (status, the index listing, the filter and racy rule,
// diff-index, diff-files, the gitlinks); the same change anywhere else,
// .specmetry-not/ included, still counts. Each case is checked against
// the definition's status itself.
func TestCleanLeavesOutSpecmetry(t *testing.T) {
	requireSafeGit(t)
	future := time.Now().Add(time.Hour)
	for _, c := range []struct {
		name   string
		change func(t *testing.T, r *testutil.GitRepo)
		empty  bool  // the definition's status prints nothing
		want   State // what Clean must say
	}{
		{"a filtered file in .specmetry/ changed (the review's case)", func(t *testing.T, r *testutil.GitRepo) {
			r.WriteFile(".specmetry/probe.txt", "changed\n")
			touch(t, filepath.Join(r.Dir, ".specmetry", "probe.txt"), future)
		}, true, StateClean},
		{"a filtered file in .specmetry/, only its modification time", func(t *testing.T, r *testutil.GitRepo) {
			touch(t, filepath.Join(r.Dir, ".specmetry", "probe.txt"), future)
		}, true, StateClean},
		{"a filtered file in .specmetry/, racy", func(t *testing.T, r *testutil.GitRepo) {
			idx, err := os.Stat(filepath.Join(r.Dir, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			touch(t, filepath.Join(r.Dir, ".specmetry", "probe.txt"), idx.ModTime())
		}, true, StateClean},
		{"a tracked file in .specmetry/ modified and staged", func(t *testing.T, r *testutil.GitRepo) {
			r.WriteFile(".specmetry/plain.txt", "staged\n")
			r.Git("add", ".specmetry/plain.txt")
		}, true, StateClean},
		{"a tracked file in .specmetry/ deleted", func(t *testing.T, r *testutil.GitRepo) {
			if err := os.Remove(filepath.Join(r.Dir, ".specmetry", "plain.txt")); err != nil {
				t.Fatal(err)
			}
		}, true, StateClean},
		{"the same filtered change in .specmetry-not/", func(t *testing.T, r *testutil.GitRepo) {
			r.WriteFile(".specmetry-not/probe.txt", "changed\n")
			touch(t, filepath.Join(r.Dir, ".specmetry-not", "probe.txt"), future)
		}, false, StateUndecidable},
		{"only the modification time in .specmetry-not/", func(t *testing.T, r *testutil.GitRepo) {
			touch(t, filepath.Join(r.Dir, ".specmetry-not", "probe.txt"), future)
		}, true, StateUndecidable},
		{"an untracked file in .specmetry-not/", func(t *testing.T, r *testutil.GitRepo) {
			r.WriteFile(".specmetry-not/new.txt", "n\n")
		}, false, StateDirty},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := excludeRepo(t)
			c.change(t, repo)
			check := unchanged(t, filepath.Join(repo.Dir, ".git"))
			got, err := cleanForTest(t, repo.Dir)
			check()
			if err != nil || got != c.want {
				t.Errorf("Clean = %v, %v; want %v", got, err, c.want)
			}
			if def := definitionStatus(t, repo.Dir); (def == "") != c.empty {
				t.Errorf("the definition's status printed %q; the case expects it empty = %v", def, c.empty)
			}
		})
	}
}

// The gitlinks: a submodule inside .specmetry/ is left out like any path
// there, removed or with its gitlink staged; one at .specmetry-sub is not.
// Inside a submodule the definition has no such pathspec (the superproject
// asks the submodule's own status), so a submodule's .specmetry/ counts.
func TestCleanLeavesOutSpecmetryGitlinks(t *testing.T) {
	requireSafeGit(t)
	withSub := func(t *testing.T, path string) *testutil.GitRepo {
		t.Helper()
		sub := testutil.NewGitRepo(t)
		sub.WriteFile("s.txt", "s\n")
		sub.Commit("sub")
		super := baseRepo(t)
		super.Git("-c", "protocol.file.allow=always", "submodule", "add", "-q", filepath.ToSlash(sub.Dir), path)
		super.Commit("add " + path)
		return super
	}
	for _, c := range []struct {
		name, path string
		change     func(t *testing.T, r *testutil.GitRepo, path string)
		empty      bool
		want       State
	}{
		{"a submodule in .specmetry/, its directory removed", ".specmetry/sub", func(t *testing.T, r *testutil.GitRepo, path string) {
			if err := os.RemoveAll(filepath.Join(r.Dir, filepath.FromSlash(path))); err != nil {
				t.Fatal(err)
			}
		}, true, StateClean},
		{"a submodule in .specmetry/, its gitlink staged", ".specmetry/sub", func(t *testing.T, r *testutil.GitRepo, path string) {
			r.GitIn(filepath.Join(r.Dir, filepath.FromSlash(path)), "commit", "-q", "--allow-empty", "-m", "more")
			r.Git("add", path)
		}, true, StateClean},
		{"a submodule at .specmetry-sub, its directory removed", ".specmetry-sub", func(t *testing.T, r *testutil.GitRepo, path string) {
			if err := os.RemoveAll(filepath.Join(r.Dir, filepath.FromSlash(path))); err != nil {
				t.Fatal(err)
			}
		}, false, StateDirty},
		{"a .specmetry/ inside a submodule", "sub", func(t *testing.T, r *testutil.GitRepo, path string) {
			r.WriteFile(path+"/.specmetry/marker.json", "{}\n")
		}, false, StateDirty},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := withSub(t, c.path)
			c.change(t, repo, c.path)
			check := unchanged(t, filepath.Join(repo.Dir, ".git"))
			got, err := cleanForTest(t, repo.Dir)
			check()
			if err != nil || got != c.want {
				t.Errorf("Clean = %v, %v; want %v", got, err, c.want)
			}
			if def := definitionStatus(t, repo.Dir); (def == "") != c.empty {
				t.Errorf("the definition's status printed %q; the case expects it empty = %v", def, c.empty)
			}
		})
	}
}

// The whole-index gates come before any pathspec (P2, review 057 §三 3):
// a sparse directory entry for .specmetry/ itself is still a sparse index.
func TestCleanSparseGateIgnoresTheExclusion(t *testing.T) {
	requireSafeGit(t)
	repo := baseRepo(t)
	repo.WriteFile(".specmetry/d/x.txt", "x\n")
	repo.Commit("specmetry files")
	repo.Git("sparse-checkout", "init", "--cone", "--sparse-index")
	repo.Git("sparse-checkout", "set", "dir")
	if b, err := os.ReadFile(filepath.Join(repo.Dir, ".git", "index")); err != nil || !bytes.Contains(b, []byte("sdir")) {
		t.Skip("this git keeps no sparse index")
	}
	check := unchanged(t, filepath.Join(repo.Dir, ".git"))
	if got, err := cleanForTest(t, repo.Dir); err != nil || got != StateUndecidable {
		t.Errorf("Clean = %v, %v; want undecidable", got, err)
	}
	check()
}
