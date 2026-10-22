package gitx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// appendConfig adds raw lines to the repository's configuration file, so
// that a value is written exactly as given (git config would normalize a
// bare key).
func appendConfig(t *testing.T, repo *testutil.GitRepo, lines string) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(repo.Dir, ".git", "config"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(lines); err != nil {
		t.Fatal(err)
	}
}

// standIn writes a shell script that only appends a line to a marker file
// in its own temporary directory, and returns the script and the marker.
func standIn(t *testing.T) (script, ran string) {
	t.Helper()
	dir := t.TempDir()
	ran = filepath.Join(dir, "ran")
	script = filepath.Join(dir, "stand-in.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ran >> '"+filepath.ToSlash(ran)+"'\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(script, " '") {
		t.Skipf("temporary path %q needs quoting", script)
	}
	return script, ran
}

func ranAt(path string) bool { _, err := os.Stat(path); return err == nil }

// controlled runs git once in dir in the controlled environment.
func controlled(t *testing.T, dir string, args ...string) ([]byte, int, error) {
	t.Helper()
	git, err := gitToRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return exec1(ctx, git, dir, args...)
}

// codex-review-021 B1-1: remote.<name>.promisor is read as git reads a
// boolean (git 2.32 takes promisor = 2 for true); a value git cannot read
// as one makes the check fail (exit 6) rather than answer "not a partial
// clone". extensions.partialClone names a remote and is never read as a
// boolean: any value makes a partial clone.
func TestPartialCloneReadsBooleansAsGitDoes(t *testing.T) {
	type want int
	const (
		partial want = iota
		full
		refused
	)
	cases := []struct {
		name, config string
		want         want
	}{
		{"true", "[remote \"origin\"]\n\tpromisor = true\n", partial},
		{"yes", "[remote \"origin\"]\n\tpromisor = yes\n", partial},
		{"on", "[remote \"origin\"]\n\tpromisor = on\n", partial},
		{"1", "[remote \"origin\"]\n\tpromisor = 1\n", partial},
		{"2", "[remote \"origin\"]\n\tpromisor = 2\n", partial},
		{"TRUE", "[remote \"origin\"]\n\tpromisor = TRUE\n", partial},
		{"a bare key", "[remote \"origin\"]\n\tpromisor\n", partial},
		{"false", "[remote \"origin\"]\n\tpromisor = false\n", full},
		{"no", "[remote \"origin\"]\n\tpromisor = no\n", full},
		{"off", "[remote \"origin\"]\n\tpromisor = off\n", full},
		{"0", "[remote \"origin\"]\n\tpromisor = 0\n", full},
		{"an empty value", "[remote \"origin\"]\n\tpromisor =\n", full},
		{"one of two remotes", "[remote \"a\"]\n\tpromisor = false\n[remote \"b\"]\n\tpromisor = 2\n", partial},
		{"not a boolean", "[remote \"origin\"]\n\tpromisor = maybe\n", refused},
		{"extensions.partialClone naming a remote", "[extensions]\n\tpartialClone = origin\n", partial},
		{"extensions.partialClone = false (a remote's name)", "[extensions]\n\tpartialClone = false\n", partial},
		{"extensions.partialClone = 2", "[extensions]\n\tpartialClone = 2\n", partial},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := committedRepo(t)
			appendConfig(t, repo, c.config)
			got, err := PartialClone(ctx, repo.Dir)
			switch c.want {
			case partial:
				if err != nil || !got {
					t.Errorf("PartialClone = %v, %v; want a partial clone", got, err)
				}
			case full:
				if err != nil || got {
					t.Errorf("PartialClone = %v, %v; want a full clone", got, err)
				}
			case refused:
				if apperr.ExitCode(err) != apperr.ExitEnvironment {
					t.Errorf("PartialClone = %v, %v; want exit 6", got, err)
				}
				if err != nil && strings.Contains(err.Error(), "maybe") {
					t.Errorf("the error quotes the configuration value: %v", err)
				}
			}
		})
	}
}

// codex-review-021 B1-2: the git Specmetry runs is allowed no transport at
// all: GIT_ALLOW_PROTOCOL is set and empty. A remote helper (also one with
// the name the earlier allow list held), the file transport's upload-pack,
// ext and ssh all fail before they run anything; each is shown to run with
// plain git first.
func TestNoTransportIsAllowed(t *testing.T) {
	for _, name := range []string{"specmetry-no-transport", "specmetry-probe"} {
		t.Run("remote helper "+name, func(t *testing.T) {
			pathDir := t.TempDir()
			ran := testutil.FakeRemoteHelper(t, pathDir, name)
			t.Setenv("PATH", pathDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			// The repository's git takes its environment (PATH, the marker)
			// when it is made.
			repo := committedRepo(t)
			_, _ = repo.GitResult("ls-remote", name+"::unused")
			if !ranAt(ran) {
				t.Fatal("control: plain git did not run the helper, so the case proves nothing")
			}
			if err := os.Remove(ran); err != nil {
				t.Fatal(err)
			}
			if _, code, err := controlled(t, repo.Dir, "ls-remote", name+"::unused"); err == nil && code == 0 {
				t.Error("ls-remote succeeded")
			}
			if ranAt(ran) {
				t.Error("the remote helper ran")
			}
		})
	}
	repo := committedRepo(t)
	bare := t.TempDir()
	repo.Git("init", "-q", "--bare", bare)
	for _, c := range []struct {
		name string
		args func(script string) []string
	}{
		{"file and its upload-pack", func(s string) []string {
			return []string{"ls-remote", "--upload-pack=sh " + filepath.ToSlash(s), filepath.ToSlash(bare)}
		}},
		{"ext", func(s string) []string {
			return []string{"-c", "protocol.ext.allow=always", "ls-remote", "ext::sh " + filepath.ToSlash(s)}
		}},
		{"ssh", func(s string) []string {
			return []string{"-c", "core.sshCommand=sh " + filepath.ToSlash(s), "ls-remote", "ssh://example.invalid/repo.git"}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			script, ran := standIn(t)
			_, _ = repo.GitResult(c.args(script)...)
			if !ranAt(ran) {
				t.Fatal("control: plain git did not run the stand-in, so the case proves nothing")
			}
			if err := os.Remove(ran); err != nil {
				t.Fatal(err)
			}
			if _, code, err := controlled(t, repo.Dir, c.args(script)...); err == nil && code == 0 {
				t.Error("ls-remote succeeded")
			}
			if ranAt(ran) {
				t.Error("the transport ran its program")
			}
		})
	}
	// https needs a helper git may not have been built with; it must fail
	// either way, without a connection being tried for it to fail on.
	if _, code, err := controlled(t, repo.Dir, "ls-remote", "https://127.0.0.1:9/repo.git"); err == nil && code == 0 {
		t.Error("ls-remote over https succeeded")
	}
}

// codex-review-022 BLOCKER: a repository that becomes a partial clone
// between two reads (promisor = true or 2, a blob removed) is not fetched
// from: no check of the repository comes before a read, so no change can
// slip in between; git itself, which honours GIT_NO_LAZY_FETCH, refuses to
// fetch while it reads (PD-26). The read fails with exit 6, the promisor
// remote does not run, and nothing in the repository changes.
func TestARepositoryThatBecomesAPartialCloneIsNotFetchedFrom(t *testing.T) {
	requireSafeGit(t)
	for _, value := range []string{"true", "2"} {
		t.Run("promisor = "+value, func(t *testing.T) {
			repo := testutil.NewGitRepo(t)
			repo.WriteFile("internal/auth/login.go", "package auth\n")
			head := repo.Commit("init")
			blob := repo.Git("rev-parse", head+":internal/auth/login.go")
			if c, err := CommitOf(ctx, repo.Dir, "HEAD"); err != nil || c != head {
				t.Fatalf("CommitOf before = %q, %v", c, err)
			}
			if n, err := FileLines(ctx, repo.Dir, head, []string{"internal/auth/login.go"}); err != nil || n["internal/auth/login.go"] != 1 {
				t.Fatalf("FileLines before = %v, %v", n, err)
			}
			script, ran := standIn(t)
			repo.Git("config", "remote.origin.url", filepath.ToSlash(t.TempDir()))
			repo.Git("config", "remote.origin.uploadpack", "sh "+filepath.ToSlash(script))
			repo.Git("config", "remote.origin.promisor", value)
			removeObject(t, repo, blob)
			gitDir := filepath.Join(repo.Dir, ".git")
			before := testutil.SettledSnapshot(t, gitDir)
			if _, err := FileLines(ctx, repo.Dir, head, []string{"internal/auth/login.go"}); apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Errorf("FileLines with the blob removed = %v; want exit 6", err)
			}
			if ranAt(ran) {
				t.Error("the promisor remote's upload-pack ran")
			}
			testutil.AssertTreeUnchanged(t, gitDir, before)
		})
	}
}
