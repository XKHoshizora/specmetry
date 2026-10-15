package gitx

import (
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// TD-26: the object format is what git reports for the repository: sha1 or
// sha256, also in a SHA-256 repository without commits. An answer that is
// neither is an environment error, never taken for SHA-1.
func TestObjectFormat(t *testing.T) {
	sha1 := committedRepo(t)
	sha256 := testutil.NewSHA256GitRepo(t)
	for want, repo := range map[string]*testutil.GitRepo{"sha1": sha1, "sha256": sha256} {
		if got, err := ObjectFormat(ctx, repo.Dir); err != nil || got != want {
			t.Errorf("ObjectFormat = %q, %v; want %s", got, err, want)
		}
	}
	dir, env, _ := testutil.FakeGit(t, "git version 2.56.0")
	useGit(t, dir, env)
	if got, err := ObjectFormat(ctx, sha1.Dir); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("ObjectFormat with a git that answers something else = %q, %v; want exit 6", got, err)
	}
}
