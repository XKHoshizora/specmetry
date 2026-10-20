package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/claim"
)

// PD-26 with a real git that fetches although GIT_NO_LAZY_FETCH is set
// (git 2.32.0), through the claim command (WP2.4 spec §15): a batch with
// file or commit references, and one without (its HEAD is a commit to
// read), is exit 6 with nothing written. Run with
// SPECMETRY_TEST_GIT_OBJECTS=unsafe and SPECMETRY_TEST_SAFE_GIT set to a
// git that never fetches, which sets the project up; CI runs the stand-in
// tests instead (TestClaimNeedsASafeGit).
func TestClaimWithARealGitThatFetches(t *testing.T) {
	if os.Getenv(gitObjectsEnv) != "unsafe" {
		t.Skipf("set %s=unsafe, with such a git first on PATH, to run this compatibility check", gitObjectsEnv)
	}
	safe := os.Getenv(safeGitEnv)
	if safe == "" {
		t.Fatalf("set %s to a git that never fetches, to set the project up", safeGitEnv)
	}
	path := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Dir(safe)+string(os.PathListSeparator)+path)
	home, f, dir := claimsProject(t)
	head := f.Repo.Git("rev-parse", "HEAD")
	t.Setenv("PATH", path)
	for _, refs := range []map[string]any{nil, {"files": []any{fileRef("docs/client-brief.md", "")}}, {"commits": []any{head[:7]}}} {
		env := claimRefused(t, home, dir, claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", refs)), 6, "")
		if !strings.Contains(env.Error.Message, "GIT_NO_LAZY_FETCH") {
			t.Errorf("refs %v: %s", refs, env.Error.Message)
		}
	}
}
