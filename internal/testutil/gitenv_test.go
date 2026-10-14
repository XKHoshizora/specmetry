package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// codex-review-012 B1: git environment variables inherited by the test
// process must not redirect fixture operations to another repository or
// change the fixture's commits. Each case injects variables that point at
// a separate temporary "outer" repository, builds a fixture through the
// public BuildFixture path, and checks that the outer repository is
// untouched, the fixture repository is where it should be, and the commit
// IDs are the pinned ones.
func TestBuildFixtureIgnoresInheritedGitEnvironment(t *testing.T) {
	cases := map[string]func(outer string, hooks string) map[string]string{
		"GIT_DIR and GIT_WORK_TREE": func(outer, _ string) map[string]string {
			return map[string]string{"GIT_DIR": filepath.Join(outer, ".git"), "GIT_WORK_TREE": outer}
		},
		"GIT_COMMON_DIR": func(outer, _ string) map[string]string {
			return map[string]string{"GIT_COMMON_DIR": filepath.Join(outer, ".git")}
		},
		"GIT_INDEX_FILE": func(outer, _ string) map[string]string {
			return map[string]string{"GIT_INDEX_FILE": filepath.Join(outer, ".git", "index")}
		},
		"GIT_OBJECT_DIRECTORY": func(outer, _ string) map[string]string {
			return map[string]string{"GIT_OBJECT_DIRECTORY": filepath.Join(outer, ".git", "objects")}
		},
		"GIT_DEFAULT_HASH": func(_, _ string) map[string]string {
			return map[string]string{"GIT_DEFAULT_HASH": "sha256"}
		},
		"GIT_CONFIG_COUNT hooks path": func(_, hooks string) map[string]string {
			return map[string]string{"GIT_CONFIG_COUNT": "1", "GIT_CONFIG_KEY_0": "core.hooksPath", "GIT_CONFIG_VALUE_0": hooks}
		},
		"GIT_CONFIG_PARAMETERS hooks path": func(_, hooks string) map[string]string {
			return map[string]string{"GIT_CONFIG_PARAMETERS": "'core.hooksPath'='" + filepath.ToSlash(hooks) + "'"}
		},
		"GIT_TEMPLATE_DIR with a hook": func(_, hooks string) map[string]string {
			return map[string]string{"GIT_TEMPLATE_DIR": filepath.Dir(hooks)}
		},
		"all at once": func(outer, hooks string) map[string]string {
			return map[string]string{
				"GIT_DIR":              filepath.Join(outer, ".git"),
				"GIT_WORK_TREE":        outer,
				"GIT_COMMON_DIR":       filepath.Join(outer, ".git"),
				"GIT_INDEX_FILE":       filepath.Join(outer, ".git", "index"),
				"GIT_OBJECT_DIRECTORY": filepath.Join(outer, ".git", "objects"),
				"GIT_DEFAULT_HASH":     "sha256",
				"GIT_CONFIG_COUNT":     "1",
				"GIT_CONFIG_KEY_0":     "core.hooksPath",
				"GIT_CONFIG_VALUE_0":   hooks,
				"GIT_TEMPLATE_DIR":     filepath.Dir(hooks),
				"GIT_AUTHOR_DATE":      "1700000000 +0000",
				"GIT_COMMITTER_NAME":   "Injected",
			}
		},
	}
	for name, inject := range cases {
		t.Run(name, func(t *testing.T) {
			outer := NewGitRepo(t)
			outer.WriteFile("README.md", "outer repository\n")
			outer.Commit("outer: initial commit")
			marker := filepath.Join(t.TempDir(), "hook-ran")
			hooks := hookTemplate(t, marker)
			before := SettledSnapshot(t, outer.Dir)

			for k, v := range inject(outer.Dir, hooks) {
				t.Setenv(k, v)
			}
			f := BuildFixture(t, "fx-docs-only")

			AssertTreeUnchanged(t, outer.Dir, before)
			if _, err := os.Stat(filepath.Join(f.Repo.Dir, ".git")); err != nil {
				t.Fatalf("the fixture repository has no .git of its own: %v", err)
			}
			if outerPath, fixturePath := filepath.Clean(outer.Dir), filepath.Clean(f.Repo.Dir); strings.EqualFold(outerPath, fixturePath) {
				t.Fatalf("the fixture was built in the outer repository %s", outerPath)
			}
			if got := f.Refs(); !reflect.DeepEqual(got, f.Manifest.ExpectedRefs) {
				t.Errorf("refs %v, manifest expects %v", got, f.Manifest.ExpectedRefs)
			}
			if _, err := os.Stat(marker); err == nil {
				t.Error("an injected hook ran")
			}
		})
	}
}

// hookTemplate creates a git template directory whose hooks/ holds
// post-commit and pre-commit hooks that create marker, and returns the
// hooks directory (usable as core.hooksPath too).
func hookTemplate(t *testing.T, marker string) string {
	t.Helper()
	hooks := filepath.Join(t.TempDir(), "template", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\necho ran > '" + filepath.ToSlash(marker) + "'\n"
	for _, h := range []string{"pre-commit", "post-commit"} {
		if err := os.WriteFile(filepath.Join(hooks, h), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return hooks
}

// fatalRecorder records a Fatal call instead of failing the test, and stops
// the calling goroutine as testing.T does.
type fatalRecorder struct {
	testing.TB
	msg string
}

func (f *fatalRecorder) Fatalf(format string, args ...any) {
	f.msg = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

func (f *fatalRecorder) Fatal(args ...any) {
	f.msg = fmt.Sprint(args...)
	runtime.Goexit()
}

// codex-review-012 B1: before a commit, GitRepo checks that git resolves
// the repository to its own .git. Here .git is replaced by a gitfile that
// points at another repository: the commit must stop before writing.
func TestGitRepoRefusesAForeignRepository(t *testing.T) {
	outer := NewGitRepo(t)
	outer.WriteFile("README.md", "outer repository\n")
	outer.Commit("outer: initial commit")
	before := SettledSnapshot(t, outer.Dir)

	r := NewGitRepo(t)
	if err := os.RemoveAll(filepath.Join(r.Dir, ".git")); err != nil {
		t.Fatal(err)
	}
	gitfile := "gitdir: " + filepath.ToSlash(filepath.Join(outer.Dir, ".git")) + "\n"
	if err := os.WriteFile(filepath.Join(r.Dir, ".git"), []byte(gitfile), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := &fatalRecorder{TB: t}
	r.t = rec
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.WriteFile("other.txt", "must not be committed\n")
		r.Commit("must not reach the outer repository")
	}()
	<-done
	if !strings.Contains(rec.msg, "not the test repository") {
		t.Fatalf("commit was not refused: %q", rec.msg)
	}
	AssertTreeUnchanged(t, outer.Dir, before)
}
