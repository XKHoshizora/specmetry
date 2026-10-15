package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// fetchingGit puts a git that fetches a missing object although
// GIT_NO_LAZY_FETCH is set first on PATH, for the rest of the test: a
// stand-in that hides the variable from the real git, as git 2.32 ignores
// it. It returns the stand-in's recorded calls. Every git command the test
// runs itself (repo.Git) must come before: the stand-in only works with
// the variables this sets.
func fetchingGit(t *testing.T) func() []testutil.FakeGitCall {
	t.Helper()
	dir, env, calls := testutil.PassthroughGit(t, testutil.PassWithoutNoLazyFetch)
	for _, kv := range env {
		name, value, _ := strings.Cut(kv, "=")
		t.Setenv(name, value)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return calls
}

// probeObject is the object gitx's probe asks for, in a repository of its
// own.
const probeObject = "0123456789abcdef0123456789abcdef01234567"

// noObjectReads fails the test if git ran a command that may read an
// object of a user's repository: anything but the version check, the
// probe, and commands that read refs, the configuration or the layout.
func noObjectReads(t *testing.T, calls []testutil.FakeGitCall) {
	t.Helper()
	for _, c := range calls {
		if slices.Contains(c.Args, probeObject) {
			continue
		}
		for _, cmd := range []string{"cat-file", "ls-tree", "ls-files", "--verify", "status", "log"} {
			if slices.Contains(c.Args, cmd) {
				t.Errorf("git ran a command that may read an object: %q", c.Args)
			}
		}
	}
}

// initRefused runs `init --json` in dir and checks the refusal: exit 6 and
// an error.v1 whose message holds want.
func initRefused(t *testing.T, dir, want string) {
	t.Helper()
	t.Chdir(dir)
	code, stdout, stderr := execute(t, "init", "--json")
	conforms(t, code, stdout)
	if code != apperr.ExitEnvironment {
		t.Fatalf("exit %d, want 6\n%s%s", code, stdout, stderr)
	}
	var env apperr.Envelope
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != "environment" || !strings.Contains(env.Error.Message, want) {
		t.Fatalf("error %+v, want a message with %q", env.Error, want)
	}
}

// PD-26, WP1.1: with a git that could fetch a missing object, a git
// project's init stops before it lists files, opens the database or writes
// anything: exit 6, no data directory, and the repository (marker,
// exclude, .git) as it was. With a data directory that already holds
// projects, nothing in it changes either (seq included), whether the
// project is new or registered. No git command that may read an object
// runs.
func TestInitRefusesAGitThatFetches(t *testing.T) {
	home := testutil.TempHome(t)
	fresh := committedRepo(t)
	registered := committedRepo(t)
	other := t.TempDir()

	t.Run("no data directory yet", func(t *testing.T) {
		calls := fetchingGit(t)
		before := testutil.SettledSnapshot(t, fresh.Dir)
		initRefused(t, fresh.Dir, "does not honour GIT_NO_LAZY_FETCH")
		if exists(home.Path) {
			t.Error("the data directory was created")
		}
		testutil.AssertTreeUnchanged(t, fresh.Dir, before)
		noObjectReads(t, calls())
	})

	t.Run("a data directory with projects", func(t *testing.T) {
		mustInit(t, other) // not a git project: no git object is read
		mustInit(t, registered.Dir)
		calls := fetchingGit(t)
		for _, repo := range []*testutil.GitRepo{fresh, registered} {
			beforeHome := testutil.SettledSnapshot(t, home.Path)
			beforeRepo := testutil.SettledSnapshot(t, repo.Dir)
			initRefused(t, repo.Dir, "does not honour GIT_NO_LAZY_FETCH")
			testutil.AssertTreeUnchanged(t, home.Path, beforeHome)
			testutil.AssertTreeUnchanged(t, repo.Dir, beforeRepo)
		}
		noObjectReads(t, calls())
		// A project outside git is initialized as before.
		if out := mustInit(t, other); out.Created || out.Target != nil {
			t.Errorf("init outside git = %+v", out)
		}
	})
}

// PD-26, WP1.1: init reports a commit it cannot read as an error (exit 6),
// never as a branch without commits (snapshot null), and writes nothing:
// the commit of the main worktree's branch is missing (a partial clone
// that lacks it; its promisor remote does not run), or, for a registered
// project, the commit of the ref its target keeps, read in the write
// transaction, which then writes nothing.
func TestInitRefusesACommitItCannotRead(t *testing.T) {
	home := testutil.TempHome(t)
	repo := committedRepo(t)
	head := repo.Git("rev-parse", "HEAD")
	script := filepath.Join(t.TempDir(), "upload-pack.sh")
	ran := filepath.Join(filepath.Dir(script), "ran")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ran >> '"+filepath.ToSlash(ran)+"'\nexit 1\n"), 0o755); err != nil {
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
	object := filepath.Join(repo.Dir, ".git", "objects", head[:2], head[2:])
	commit, err := os.ReadFile(object)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(object); err != nil {
		t.Fatal(err)
	}
	before := testutil.SettledSnapshot(t, repo.Dir)
	initRefused(t, repo.Dir, "cannot be read")
	if exists(home.Path) {
		t.Error("the data directory was created")
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("the promisor remote's upload-pack ran")
	}
	testutil.AssertTreeUnchanged(t, repo.Dir, before)

	// Registered on main; main's commit then goes missing while another
	// branch is checked out.
	if err := os.WriteFile(object, commit, 0o444); err != nil {
		t.Fatal(err)
	}
	if out := mustInit(t, repo.Dir); out.Target == nil || out.Target.Snapshot == nil || *out.Target.Snapshot != head {
		t.Fatalf("init = %+v", out.Target)
	}
	repo.Git("checkout", "-q", "-b", "other")
	repo.WriteFile("other.md", "other\n")
	repo.Commit("other")
	repo.AddWorktree("feature")
	if err := os.Remove(object); err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(filepath.Dir(repo.Dir), "wt-feature")
	beforeSeq := seq(t, home)
	initRefused(t, wt, "cannot be read")
	if after := seq(t, home); after != beforeSeq {
		t.Errorf("seq %s -> %s", beforeSeq, after)
	}
	if n := count(t, home, `SELECT count(*) FROM worktrees`); n != 1 {
		t.Errorf("worktrees = %d: the linked worktree was registered", n)
	}
}

// PD-26, WP1.1: with a git that never fetches, init is as before: the
// candidates, and the snapshot of the main worktree's branch.
func TestInitWithAGitThatNeverFetches(t *testing.T) {
	testutil.TempHome(t)
	repo := committedRepo(t)
	repo.WriteFile("docs/prd.md", "# PRD\n")
	head := repo.Commit("docs")
	out := mustInit(t, repo.Dir)
	if out.Target == nil || out.Target.Snapshot == nil || *out.Target.Snapshot != head {
		t.Fatalf("target = %+v", out.Target)
	}
	if !slices.Contains(out.CandidateSources, "docs/prd.md") {
		t.Fatalf("candidates = %q", out.CandidateSources)
	}
}

// PD-26: with a git that could fetch a missing object, checking code
// references is refused (exit 6) before any object is read, and nothing is
// written or allocated a seq; a proposal without code references, and the
// source commands, are as before.
func TestProposeAndSourcesWithAGitThatFetches(t *testing.T) {
	home, f, _ := proposalsProject(t)
	repo := f.Repo
	repo.WriteFile("internal/auth/login.go", "package auth\n\nfunc Login() {}\n")
	repo.Commit("add login")
	gitDir := filepath.Join(repo.Dir, ".git")
	calls := fetchingGit(t)
	before := testutil.SettledSnapshot(t, gitDir)

	withRefs := map[string]any{"schema": "specmetry.proposal/v1", "origin": "skill_init", "base_version": 0, "summary": "inferred",
		"items": []any{map[string]any{"op": "add", "type": "requirement", "ref": "r1", "origin": "inferred_from_code",
			"rationale": "the login handler locks accounts",
			"code_refs": []any{map[string]any{"path": "internal/auth/login.go", "lines": "1-3"}},
			"data":      map[string]any{"title": "log in", "kind": "functional", "priority": "must"}}}}
	env := proposeRefused(t, home, repo.Dir, withRefs, apperr.ExitEnvironment, "")
	if env.Error.Code != "environment" || !strings.Contains(env.Error.Message, "does not honour GIT_NO_LAZY_FETCH") {
		t.Errorf("error %+v", env.Error)
	}

	withoutRefs := map[string]any{"schema": "specmetry.proposal/v1", "origin": "cli", "base_version": 0, "summary": "one goal",
		"items": []any{map[string]any{"op": "add", "type": "goal", "ref": "g1", "data": map[string]any{"title": "launch"}, "origin": "manual"}}}
	if r := proposeOK(t, repo.Dir, withoutRefs); r.ProposalID == "" {
		t.Errorf("receipt = %+v", r)
	}

	added := sourceAdd(t, repo.Dir, "docs/client-brief.md")
	if added.Created {
		t.Errorf("source add of a registered source = %+v", added)
	}
	if list := sourceList(t, repo.Dir); len(list.Sources) != 1 {
		t.Errorf("source list = %+v", list)
	}
	brief := filepath.Join(repo.Dir, "docs", "client-brief.md")
	if err := os.WriteFile(brief, []byte("# brief, revised\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if out := inTerminal(t, repo.Dir, []string{"source", "ack", "docs/client-brief.md", "--observed", fileSHA(t, brief), "-m", "typo"}, "docs/client-brief.md"); !strings.Contains(out, "exit=0") {
		t.Errorf("source ack:\n%s", out)
	}
	if out := inTerminal(t, repo.Dir, []string{"source", "remove", "docs/client-brief.md"}, "docs/client-brief.md"); !strings.Contains(out, "exit=0") {
		t.Errorf("source remove:\n%s", out)
	}
	testutil.AssertTreeUnchanged(t, gitDir, before)
	noObjectReads(t, calls())
}
