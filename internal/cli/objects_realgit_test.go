package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// gitObjectsEnv says what the git on PATH is expected to be for object
// reads, "safe" or "unsafe", when a test run is made with a chosen real
// git; safeGitEnv names a git that never fetches, which sets up the
// project for the code reference check of an "unsafe" run.
const (
	gitObjectsEnv = "SPECMETRY_TEST_GIT_OBJECTS"
	safeGitEnv    = "SPECMETRY_TEST_SAFE_GIT"
)

// PD-26 with a real git that fetches although GIT_NO_LAZY_FETCH is set
// (git 2.32.0), through the public commands: a git project's init is
// refused (exit 6) with no data directory and the repository as it was; a
// project outside git is initialized; checking code references is refused
// with nothing written; a proposal without code references and the source
// commands work. Run with SPECMETRY_TEST_GIT_OBJECTS=unsafe and
// SPECMETRY_TEST_SAFE_GIT set to a git that never fetches; CI runs the
// stand-in tests instead (objects_test.go).
func TestARealGitThatFetches(t *testing.T) {
	if os.Getenv(gitObjectsEnv) != "unsafe" {
		t.Skipf("set %s=unsafe, with such a git first on PATH, to run this compatibility check", gitObjectsEnv)
	}
	safe := os.Getenv(safeGitEnv)
	if safe == "" {
		t.Fatalf("set %s to a git that never fetches, to set the project up", safeGitEnv)
	}
	home := testutil.TempHome(t)
	f := testutil.BuildFixture(t, "fx-docs-only") // before any t.Chdir
	repo := committedRepo(t)
	before := testutil.SettledSnapshot(t, repo.Dir)
	initRefused(t, repo.Dir, "does not honour GIT_NO_LAZY_FETCH")
	if exists(home.Path) {
		t.Error("the data directory was created")
	}
	testutil.AssertTreeUnchanged(t, repo.Dir, before)
	if out := mustInit(t, t.TempDir()); !out.Created || out.Target != nil {
		t.Errorf("init outside git = %+v", out)
	}

	// The project, set up with a git that never fetches.
	path := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Dir(safe)+string(os.PathListSeparator)+path)
	mustInit(t, f.Repo.Dir)
	sourceAdd(t, f.Repo.Dir, "docs/client-brief.md")
	f.Repo.WriteFile("internal/auth/login.go", "package auth\n\nfunc Login() {}\n")
	f.Repo.Commit("add login")
	t.Setenv("PATH", path)

	gitDir := filepath.Join(f.Repo.Dir, ".git")
	beforeGit := testutil.SettledSnapshot(t, gitDir)
	withRefs := map[string]any{"schema": "specmetry.proposal/v1", "origin": "skill_init", "base_version": 0, "summary": "inferred",
		"items": []any{map[string]any{"op": "add", "type": "requirement", "ref": "r1", "origin": "inferred_from_code",
			"rationale": "the login handler locks accounts",
			"code_refs": []any{map[string]any{"path": "internal/auth/login.go", "lines": "1-3"}},
			"data":      map[string]any{"title": "log in", "kind": "functional", "priority": "must"}}}}
	env := proposeRefused(t, home, f.Repo.Dir, withRefs, apperr.ExitEnvironment, "")
	if !strings.Contains(env.Error.Message, "does not honour GIT_NO_LAZY_FETCH") {
		t.Errorf("error %+v", env.Error)
	}
	withoutRefs := map[string]any{"schema": "specmetry.proposal/v1", "origin": "cli", "base_version": 0, "summary": "one goal",
		"items": []any{map[string]any{"op": "add", "type": "goal", "ref": "g1", "data": map[string]any{"title": "launch"}, "origin": "manual"}}}
	proposeOK(t, f.Repo.Dir, withoutRefs)
	if added := sourceAdd(t, f.Repo.Dir, "docs/client-brief.md"); added.Created {
		t.Errorf("source add = %+v", added)
	}
	if list := sourceList(t, f.Repo.Dir); len(list.Sources) != 1 {
		t.Errorf("source list = %+v", list)
	}
	brief := filepath.Join(f.Repo.Dir, "docs", "client-brief.md")
	if err := os.WriteFile(brief, []byte("# brief, revised\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if out := inTerminal(t, f.Repo.Dir, []string{"source", "ack", "docs/client-brief.md", "--observed", fileSHA(t, brief), "-m", "typo"}, "docs/client-brief.md"); !strings.Contains(out, "exit=0") {
		t.Errorf("source ack:\n%s", out)
	}
	if out := inTerminal(t, f.Repo.Dir, []string{"source", "remove", "docs/client-brief.md"}, "docs/client-brief.md"); !strings.Contains(out, "exit=0") {
		t.Errorf("source remove:\n%s", out)
	}
	testutil.AssertTreeUnchanged(t, gitDir, beforeGit)
}
