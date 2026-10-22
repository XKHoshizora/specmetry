package gitx

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// gitObjectsEnv says what the git on PATH is expected to be for object
// reads, "safe" or "unsafe", when a test run is made with a chosen real git
// (git 2.32.0, a distribution's 2.43, 2.56). Without it the probe is
// checked against the independent oracle only.
const gitObjectsEnv = "SPECMETRY_TEST_GIT_OBJECTS"

// requireSafeGit skips a test that reads objects when the git on PATH is
// expected not to be allowed to.
func requireSafeGit(t *testing.T) {
	t.Helper()
	if os.Getenv(gitObjectsEnv) == "unsafe" {
		t.Skip("the git on PATH may not read objects (" + gitObjectsEnv + "=unsafe)")
	}
}

// useGit puts dir first on PATH and sets env, for the rest of the test.
func useGit(t *testing.T, dir string, env []string) {
	t.Helper()
	for _, kv := range env {
		name, value, _ := strings.Cut(kv, "=")
		t.Setenv(name, value)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// kept returns what gitx keeps about the git it checked last.
func kept() objectSafety {
	stateMu.Lock()
	defer stateMu.Unlock()
	return state.objects
}

// oracle reads a blob that a partial clone lacks with plain git, which has
// no protocol allow list, and reports whether the promisor remote's
// upload-pack stand-in ran: whether git fetched. GIT_NO_LAZY_FETCH=1 is set
// when noLazyFetch. It shares nothing with the probe: another repository,
// made by git, and a program that runs instead of trace2 events.
func oracle(t *testing.T, noLazyFetch bool) bool {
	t.Helper()
	repo, head, ran := promisorRepo(t)
	blob := repo.Git("rev-parse", head+":internal/auth/login.go")
	removeObject(t, repo, blob)
	var extra []string
	if noLazyFetch {
		extra = []string{"GIT_NO_LAZY_FETCH=1"}
	}
	_, _ = repo.GitResultEnv(extra, "cat-file", "blob", blob)
	return ranAt(ran)
}

// PD-26: the probe tells, from git's behaviour, whether the git on PATH
// fetches a missing object although GIT_NO_LAZY_FETCH is set, and never
// finds nothing out about a real git. Its answer agrees with an
// independent oracle, and with the expectation of a run made with a chosen
// git (git 2.32.0 fetches; a distribution's 2.43 with the CVE-2024-32465
// fixes and 2.56 do not).
func TestProbeAgreesWithTheGitOnPath(t *testing.T) {
	err := CheckObjectReads(ctx)
	got := kept()
	switch {
	case err == nil && got == neverFetches:
	case apperr.ExitCode(err) == apperr.ExitEnvironment && got == fetches:
		if !strings.Contains(err.Error(), "does not honour GIT_NO_LAZY_FETCH") {
			t.Errorf("message = %v", err)
		}
	default:
		t.Fatalf("the probe found nothing out about the git on PATH: %v", err)
	}
	version, _, _ := controlled(t, "", "--version")
	t.Logf("%s: fetches anyway = %v", strings.TrimSpace(string(version)), got == fetches)

	if !oracle(t, false) {
		t.Fatal("oracle control: plain git did not run the promisor remote's upload-pack, so the oracle proves nothing")
	}
	if anyway := oracle(t, true); anyway != (got == fetches) {
		t.Fatalf("the probe says fetches anyway = %v, the oracle says %v", got == fetches, anyway)
	}
	switch want := os.Getenv(gitObjectsEnv); want {
	case "":
	case "safe", "unsafe":
		if (want == "unsafe") != (got == fetches) {
			t.Fatalf("%s=%s, but the probe says fetches anyway = %v", gitObjectsEnv, want, got == fetches)
		}
	default:
		t.Fatalf("%s=%q: want safe or unsafe", gitObjectsEnv, want)
	}
}

// sparseRepo is a repository with a sparse index (cone mode, only a/ in
// the worktree): b/ is one entry of the index, which git expands from the
// tree it names. It returns the repository, its commit and the tree of b/.
func sparseRepo(t *testing.T) (repo *testutil.GitRepo, head, treeB string) {
	t.Helper()
	repo = testutil.NewGitRepo(t)
	repo.WriteFile("a/x.txt", "one\n")
	repo.WriteFile("b/y.txt", "two\n")
	head = repo.Commit("init")
	repo.Git("sparse-checkout", "init", "--cone", "--sparse-index")
	repo.Git("sparse-checkout", "set", "a")
	// A sparse index has the "sdir" extension (no path here holds it).
	if index, err := os.ReadFile(filepath.Join(repo.Dir, ".git", "index")); err != nil || !bytes.Contains(index, []byte("sdir")) {
		t.Skip("this git keeps no sparse index")
	}
	return repo, head, repo.Git("rev-parse", head+":b")
}

// makePartial makes repo a partial clone whose promisor remote's
// upload-pack is a stand-in, and returns the stand-in's marker.
func makePartial(t *testing.T, repo *testutil.GitRepo) (ran string) {
	t.Helper()
	script, ran := standIn(t)
	repo.Git("config", "core.repositoryformatversion", "1")
	repo.Git("config", "extensions.partialClone", "origin")
	repo.Git("config", "remote.origin.url", filepath.ToSlash(t.TempDir()))
	repo.Git("config", "remote.origin.promisor", "true")
	repo.Git("config", "remote.origin.uploadpack", "sh "+filepath.ToSlash(script))
	return ran
}

// PD-26: with a git that fetches although GIT_NO_LAZY_FETCH is set (a
// stand-in that hides the variable from the real git, as git 2.32 ignores
// it), every command that may read an object is refused with exit 6
// before it runs, in every repository: a full clone, a partial clone that
// lacks a blob, and a sparse index. The only git run is the version check
// and the probe, in its own directory; nothing in the repository changes,
// and no promisor remote runs. Commands that read no object still run.
func TestAGitThatFetchesReadsNoObject(t *testing.T) {
	full := committedRepo(t)
	partial, partialHead, ran := promisorRepo(t)
	blob := partial.Git("rev-parse", partialHead+":internal/auth/login.go")
	removeObject(t, partial, blob)
	sparse, sparseHead, _ := sparseRepo(t)
	sparseRan := makePartial(t, sparse)
	repos := []struct {
		name string
		repo *testutil.GitRepo
		head string
	}{{"a full clone", full, full.Git("rev-parse", "HEAD")}, {"a partial clone", partial, partialHead}, {"a sparse index", sparse, sparseHead}}

	// From here on, git (repo.Git's too) is the stand-in.
	dir, env, calls := testutil.PassthroughGit(t, testutil.PassWithoutNoLazyFetch)
	useGit(t, dir, env)
	for _, r := range repos {
		t.Run(r.name, func(t *testing.T) {
			gitDir := filepath.Join(r.repo.Dir, ".git")
			before := testutil.SettledSnapshot(t, gitDir)
			_, errCommit := CommitOf(ctx, r.repo.Dir, "HEAD")
			_, errLines := FileLines(ctx, r.repo.Dir, r.head, []string{"internal/auth/login.go"})
			_, errFiles := ListFiles(ctx, r.repo.Dir)
			for name, err := range map[string]error{"CommitOf": errCommit, "FileLines": errLines, "ListFiles": errFiles, "CheckObjectReads": CheckObjectReads(ctx)} {
				if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "does not honour GIT_NO_LAZY_FETCH") {
					t.Errorf("%s: %v", name, err)
				}
			}
			if kept() != fetches {
				t.Errorf("kept = %v, want fetches", kept())
			}
			testutil.AssertTreeUnchanged(t, gitDir, before)
			// Reading refs, the configuration and the layout goes on.
			if _, err := Locate(ctx, r.repo.Dir); err != nil {
				t.Errorf("Locate: %v", err)
			}
			if _, err := HeadBranch(ctx, r.repo.Dir); err != nil {
				t.Errorf("HeadBranch: %v", err)
			}
			if _, err := PartialClone(ctx, r.repo.Dir); err != nil {
				t.Errorf("PartialClone: %v", err)
			}
		})
	}
	if ranAt(ran) || ranAt(sparseRan) {
		t.Error("a promisor remote's upload-pack ran")
	}
	probes := 0
	for _, c := range calls() {
		switch {
		case slices.Equal(c.Args, []string{"--version"}):
		case slices.Contains(c.Args, probeObject):
			probes++
		case slices.ContainsFunc([]string{"cat-file", "ls-tree", "ls-files", "--verify"}, func(a string) bool { return slices.Contains(c.Args, a) }):
			t.Errorf("git ran a command that may read an object: %q", c.Args)
		}
	}
	if probes != 2 {
		t.Errorf("the probe ran git %d times, want 2 (once, then kept)", probes)
	}
}

// PD-26: a git about which the probe finds nothing out is never taken for
// one that may read objects, and nothing is kept, so the next read probes
// again: a stand-in that prints a version for any command and writes no
// event, and the real git with its trace2 events hidden.
func TestProbeFindsNothingOut(t *testing.T) {
	for _, c := range []struct {
		name string
		make func(t *testing.T) (string, []string, func() []testutil.FakeGitCall)
	}{
		{"a stand-in that only prints a version", func(t *testing.T) (string, []string, func() []testutil.FakeGitCall) {
			return testutil.FakeGit(t, "git version 2.56.0")
		}},
		{"git without trace2 events", func(t *testing.T) (string, []string, func() []testutil.FakeGitCall) {
			return testutil.PassthroughGit(t, testutil.PassWithoutTrace)
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := committedRepo(t)
			dir, env, calls := c.make(t)
			useGit(t, dir, env)
			for i := range 2 {
				_, err := CommitOf(ctx, repo.Dir, "HEAD")
				if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "cannot tell whether this git") {
					t.Fatalf("read %d: %v", i+1, err)
				}
				if kept() != unprobed {
					t.Fatalf("read %d: kept = %v, want nothing", i+1, kept())
				}
			}
			probes := 0
			for _, call := range calls() {
				if slices.Contains(call.Args, probeObject) {
					probes++
				} else if !slices.Equal(call.Args, []string{"--version"}) {
					t.Errorf("git ran %q", call.Args)
				}
			}
			if probes < 2 {
				t.Errorf("the probe ran %d times for two reads; nothing should have been kept", probes)
			}
		})
	}
}

// probeDirs lists the probe's directories left in dir.
func probeDirs(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var left []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "specmetry-git-probe-") {
			left = append(left, e.Name())
		}
	}
	return left
}

// setTemp makes dir the temporary directory for the rest of the test.
func setTemp(t *testing.T, dir string) {
	t.Helper()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, dir)
	}
}

// PD-26: the probe's temporary directory is removed whatever the outcome:
// a git that may read objects, one that fetches, one about which nothing
// is found out, a cancelled context, a cancellation while git runs, and a
// probe that does not finish in time. What does not finish is not kept.
func TestProbeCleansUp(t *testing.T) {
	asIs, asIsEnv, _ := testutil.PassthroughGit(t, testutil.PassAsIs)
	fetching, fetchingEnv, _ := testutil.PassthroughGit(t, testutil.PassWithoutNoLazyFetch)
	opaque, opaqueEnv, _ := testutil.FakeGit(t, "git version 2.56.0")
	hang1, hang1Env, _ := testutil.PassthroughGit(t, testutil.PassHang)
	hang2, hang2Env, _ := testutil.PassthroughGit(t, testutil.PassHang)
	cancelled, cancelledEnv, _ := testutil.PassthroughGit(t, testutil.PassAsIs)
	repo := committedRepo(t)
	temp := t.TempDir()
	path := os.Getenv("PATH")
	setTemp(t, temp)
	with := func(dir string, env []string) {
		for _, kv := range env {
			name, value, _ := strings.Cut(kv, "=")
			t.Setenv(name, value)
		}
		t.Setenv("PATH", dir+string(os.PathListSeparator)+path)
	}
	check := func(name string) {
		t.Helper()
		if left := probeDirs(t, temp); len(left) > 0 {
			t.Errorf("%s: the probe left %q", name, left)
		}
	}

	with(asIs, asIsEnv)
	if err := CheckObjectReads(ctx); err != nil && kept() != fetches {
		t.Fatalf("the git on PATH: %v", err)
	}
	check("the git on PATH")
	if kept() == neverFetches {
		if _, err := ListFiles(ctx, repo.Dir); err != nil {
			t.Fatal(err)
		}
		check("ls-files --sparse")
	}

	with(fetching, fetchingEnv)
	if err := CheckObjectReads(ctx); kept() != fetches {
		t.Fatalf("a git that fetches: %v, kept %v", err, kept())
	}
	check("a git that fetches")

	with(opaque, opaqueEnv)
	if err := CheckObjectReads(ctx); err == nil || kept() != unprobed {
		t.Fatalf("a git about which nothing is found out: %v, kept %v", err, kept())
	}
	check("a git about which nothing is found out")

	with(cancelled, cancelledEnv)
	if _, err := gitToRun(ctx); err != nil {
		t.Fatal(err)
	}
	done, cancel := context.WithCancel(ctx)
	cancel()
	if err := CheckObjectReads(done); apperr.ExitCode(err) != apperr.ExitEnvironment || kept() != unprobed {
		t.Fatalf("a cancelled context: %v, kept %v", err, kept())
	}
	check("a cancelled context")

	with(hang1, hang1Env)
	running, cancel := context.WithCancel(ctx)
	timer := time.AfterFunc(500*time.Millisecond, cancel)
	defer timer.Stop()
	start := time.Now()
	if err := CheckObjectReads(running); apperr.ExitCode(err) != apperr.ExitEnvironment || kept() != unprobed {
		t.Fatalf("a cancellation while git runs: %v, kept %v", err, kept())
	}
	if time.Since(start) > 20*time.Second {
		t.Errorf("the cancelled probe took %v", time.Since(start))
	}
	check("a cancellation while git runs")

	with(hang2, hang2Env)
	was := probeTimeout
	probeTimeout = 500 * time.Millisecond
	t.Cleanup(func() { probeTimeout = was })
	err := CheckObjectReads(ctx)
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "did not finish in time") || kept() != unprobed {
		t.Fatalf("a probe that does not finish in time: %v, kept %v", err, kept())
	}
	check("a probe that does not finish in time")
}

// PD-26: what the probe showed is kept for the git it ran, so a second
// read runs no probe, and only while PATH finds the same, unchanged file:
// a git that has changed is checked anew, its version included.
func TestProbeIsKeptForTheSameGit(t *testing.T) {
	dir, env, calls := testutil.PassthroughGit(t, testutil.PassAsIs)
	useGit(t, dir, env)
	first := CheckObjectReads(ctx)
	n := len(calls())
	if n != 3 {
		t.Fatalf("the first check ran git %d times, want 3 (version, control, test): %+v", n, calls())
	}
	if err := CheckObjectReads(ctx); (err == nil) != (first == nil) || len(calls()) != n {
		t.Fatalf("the second check ran git %d more times, or answered otherwise (%v, then %v)", len(calls())-n, first, err)
	}
	bin := filepath.Join(dir, "git")
	if _, err := os.Stat(bin); err != nil {
		bin += ".exe"
	}
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(bin, later, later); err != nil {
		t.Fatal(err)
	}
	_ = CheckObjectReads(ctx)
	if got := len(calls()) - n; got != 3 {
		t.Fatalf("after git changed, the check ran git %d times, want 3", got)
	}
}

// PD-26: CommitOf tells a ref without a commit (a branch without commits,
// a ref that does not exist), which is "", from a commit that cannot be
// read, which is an environment error: missing from a partial clone
// (fetched from nowhere, the repository unchanged), missing from a full
// clone, or not a commit.
func TestCommitOfTellsAMissingCommitFromNoCommit(t *testing.T) {
	requireSafeGit(t)
	unborn := testutil.NewGitRepo(t)
	for _, ref := range []string{"HEAD", "refs/heads/main", "refs/heads/none"} {
		if c, err := CommitOf(ctx, unborn.Dir, ref); err != nil || c != "" {
			t.Errorf("CommitOf(%s) without commits = %q, %v; want no commit", ref, c, err)
		}
	}

	partial, head, ran := promisorRepo(t)
	removeObject(t, partial, head)
	gitDir := filepath.Join(partial.Dir, ".git")
	before := testutil.SettledSnapshot(t, gitDir)
	for _, ref := range []string{"HEAD", "refs/heads/main"} {
		if c, err := CommitOf(ctx, partial.Dir, ref); apperr.ExitCode(err) != apperr.ExitEnvironment || c != "" {
			t.Errorf("CommitOf(%s) of a missing commit in a partial clone = %q, %v; want exit 6", ref, c, err)
		}
	}
	if ranAt(ran) {
		t.Error("the promisor remote's upload-pack ran")
	}
	testutil.AssertTreeUnchanged(t, gitDir, before)

	full := testutil.NewGitRepo(t)
	full.WriteFile("a.md", "a\n")
	fullHead := full.Commit("init")
	blob := full.Git("rev-parse", fullHead+":a.md")
	full.Git("tag", "a-blob", blob)
	if _, err := CommitOf(ctx, full.Dir, "refs/tags/a-blob"); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("CommitOf of a ref to a blob = %v; want exit 6", err)
	}
	removeObject(t, full, fullHead)
	if _, err := CommitOf(ctx, full.Dir, "refs/heads/main"); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("CommitOf of a commit missing from a full clone = %v; want exit 6", err)
	}
}
