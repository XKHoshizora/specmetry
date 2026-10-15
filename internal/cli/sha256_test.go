package cli

import (
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// sha256Repos returns real SHA-256 repositories: one with a commit on
// main, one whose main has no commit yet, and one with a detached HEAD.
func sha256Repos(t *testing.T) map[string]*testutil.GitRepo {
	t.Helper()
	committed := testutil.NewSHA256GitRepo(t)
	committed.WriteFile("README.md", "# demo\n")
	committed.Commit("init")
	unborn := testutil.NewSHA256GitRepo(t)
	unborn.WriteFile("README.md", "# demo\n")
	detached := testutil.NewSHA256GitRepo(t)
	detached.WriteFile("README.md", "# demo\n")
	detached.Commit("init")
	detached.Git("checkout", "-q", "--detach")
	return map[string]*testutil.GitRepo{"with a commit": committed, "without a commit": unborn, "with a detached HEAD": detached}
}

// TD-26: Specmetry v0.1.0 does not support SHA-256 object IDs. init in a
// SHA-256 repository is refused (exit 6) before anything is written: no
// data directory, database, project, marker or exclude entry, and no seq.
// It is refused whether or not a snapshot would be reported (a branch
// without commits, a detached HEAD), so leaving the snapshot out is no way
// around it. SHA-1 repositories are initialized as before, a branch
// without commits still with snapshot null.
func TestInitRefusesSHA256Repositories(t *testing.T) {
	home := testutil.TempHome(t)
	repos := sha256Repos(t)

	t.Run("no data directory yet", func(t *testing.T) {
		for name, repo := range repos {
			before := testutil.SettledSnapshot(t, repo.Dir)
			initRefused(t, repo.Dir, "SHA-256")
			if exists(home.Path) {
				t.Fatalf("%s: the data directory was created", name)
			}
			testutil.AssertTreeUnchanged(t, repo.Dir, before)
		}
	})

	t.Run("a data directory with projects", func(t *testing.T) {
		sha1 := committedRepo(t)
		head := sha1.Git("rev-parse", "HEAD")
		if out := mustInit(t, sha1.Dir); out.Target == nil || out.Target.Snapshot == nil || *out.Target.Snapshot != head {
			t.Fatalf("SHA-1 repository: target %+v", out.Target)
		}
		unborn := testutil.NewGitRepo(t)
		if out := mustInit(t, unborn.Dir); out.Target == nil || out.Target.Snapshot != nil {
			t.Fatalf("SHA-1 branch without commits: target %+v", out.Target)
		}
		mustInit(t, t.TempDir())
		for name, repo := range repos {
			beforeSeq := seq(t, home) // opens the database: before the snapshot
			beforeHome := testutil.SettledSnapshot(t, home.Path)
			beforeRepo := testutil.SettledSnapshot(t, repo.Dir)
			initRefused(t, repo.Dir, "SHA-256")
			testutil.AssertTreeUnchanged(t, home.Path, beforeHome)
			testutil.AssertTreeUnchanged(t, repo.Dir, beforeRepo)
			if after := seq(t, home); after != beforeSeq {
				t.Errorf("%s: seq %s -> %s", name, beforeSeq, after)
			}
		}
		if n := count(t, home, `SELECT count(*) FROM projects`); n != 3 {
			t.Errorf("projects = %d, want the 3 registered before", n)
		}
	})
}
