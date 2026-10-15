package gitx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// promisorRepo is a repository git treats as a partial clone: an object
// missing from it would be fetched from the promisor remote "origin". The
// remote's upload-pack is a stand-in that only appends to a marker file
// outside the repository; nothing goes over a network. It returns the repository, its commit and the marker.
func promisorRepo(t *testing.T) (repo *testutil.GitRepo, head, ran string) {
	t.Helper()
	repo = testutil.NewGitRepo(t)
	repo.WriteFile("internal/auth/login.go", "package auth\n\nfunc Login() {}\n")
	head = repo.Commit("init")
	dir := t.TempDir()
	ran = filepath.Join(dir, "upload-pack-ran")
	script := filepath.Join(dir, "upload-pack.sh")
	// It runs nothing else: a git it ran would look for the missing object
	// itself and fetch again.
	body := "#!/bin/sh\necho ran >> '" + filepath.ToSlash(ran) + "'\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(script, " '") {
		t.Skipf("temporary path %q needs quoting", script)
	}
	repo.Git("config", "core.repositoryformatversion", "1")
	repo.Git("config", "extensions.partialClone", "origin")
	repo.Git("config", "remote.origin.url", filepath.ToSlash(t.TempDir()))
	repo.Git("config", "remote.origin.promisor", "true")
	repo.Git("config", "remote.origin.uploadpack", "sh "+filepath.ToSlash(script))
	return repo, head, ran
}

// removeObject deletes a loose object, as a partial clone lacks it.
func removeObject(t *testing.T, repo *testutil.GitRepo, oid string) {
	t.Helper()
	if err := os.Remove(filepath.Join(repo.Dir, ".git", "objects", oid[:2], oid[2:])); err != nil {
		t.Fatal(err)
	}
}

// codex-review-020 B1: reading an object that a partial clone lacks never
// fetches it. git would run the promisor remote's upload-pack (shown live
// by a control with plain git), which may reach a network or run a
// program; Specmetry's git does not run it, the repository (objects, refs,
// index, configuration) is left as it was, and the missing object is never
// reported as present.
func TestMissingObjectsAreNotFetched(t *testing.T) {
	cases := []struct {
		name    string
		missing func(repo *testutil.GitRepo, head string) string
		read    func(repo *testutil.GitRepo, head, oid string) (present bool)
		control []string
	}{
		{"a missing blob",
			func(repo *testutil.GitRepo, head string) string {
				return repo.Git("rev-parse", head+":internal/auth/login.go")
			},
			func(repo *testutil.GitRepo, _, oid string) bool {
				_, err := CountLines(ctx, repo.Dir, oid)
				return err == nil
			},
			[]string{"cat-file", "blob"}},
		{"a missing tree",
			func(repo *testutil.GitRepo, head string) string { return repo.Git("rev-parse", head+":internal/auth") },
			func(repo *testutil.GitRepo, head, _ string) bool {
				files, err := TreeFiles(ctx, repo.Dir, head)
				return err == nil && len(files) > 0
			},
			[]string{"ls-tree", "-r"}},
		{"a missing commit",
			func(_ *testutil.GitRepo, head string) string { return head },
			func(repo *testutil.GitRepo, _, _ string) bool {
				c, err := CommitOf(ctx, repo.Dir, "HEAD")
				return err == nil && c != ""
			},
			[]string{"cat-file", "commit"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Control: plain git does run the stand-in here.
			control, head, ran := promisorRepo(t)
			oid := c.missing(control, head)
			removeObject(t, control, oid)
			_, _ = control.GitResult(append(c.control, oid)...)
			if _, err := os.Stat(ran); err != nil {
				t.Fatalf("control: plain git did not run the promisor remote's upload-pack, so the case proves nothing (%v)", err)
			}

			repo, head, ran := promisorRepo(t)
			oid = c.missing(repo, head)
			removeObject(t, repo, oid)
			// The snapshot covers every file of the git directory: objects,
			// refs and packed-refs, HEAD, the index and the configuration.
			// (Listing refs with git would read the missing commit.)
			gitDir := filepath.Join(repo.Dir, ".git")
			before := testutil.SettledSnapshot(t, gitDir)
			if c.read(repo, head, oid) {
				t.Error("the missing object was reported as present")
			}
			if _, err := os.Stat(ran); err == nil {
				t.Error("the promisor remote's upload-pack ran")
			}
			testutil.AssertTreeUnchanged(t, gitDir, before)
		})
	}
}

// A partial clone is recognised from its configuration before any object
// is read; a full clone is not one.
func TestPartialClone(t *testing.T) {
	repo, _, ran := promisorRepo(t)
	if partial, err := PartialClone(ctx, repo.Dir); err != nil || !partial {
		t.Errorf("PartialClone of a partial clone = %v, %v", partial, err)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("recognising the partial clone fetched something")
	}
	// Only the promisor remote makes one: extensions.partialClone alone or
	// remote.<name>.promisor alone.
	for _, keys := range [][]string{{"extensions.partialClone", "origin"}, {"remote.origin.promisor", "true"}} {
		r := testutil.NewGitRepo(t)
		r.WriteFile("a.md", "a")
		r.Commit("init")
		if keys[0] == "extensions.partialClone" {
			r.Git("config", "core.repositoryformatversion", "1")
		}
		r.Git("config", keys[0], keys[1])
		if partial, err := PartialClone(ctx, r.Dir); err != nil || !partial {
			t.Errorf("%s alone: PartialClone = %v, %v", keys[0], partial, err)
		}
	}
	full := committedRepo(t)
	full.Git("config", "remote.origin.url", filepath.ToSlash(t.TempDir()))
	full.Git("config", "remote.origin.promisor", "false")
	if partial, err := PartialClone(ctx, full.Dir); err != nil || partial {
		t.Errorf("PartialClone of a full clone = %v, %v", partial, err)
	}
}
