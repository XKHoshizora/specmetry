package gitx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	testutil.DispatchFakeGit()
	testutil.DispatchHelper(map[string]func() int{"scratch-lease": helperScratchLease, "review61-consumer": helperReview61Consumer})
	os.Exit(m.Run())
}

func committedRepo(t *testing.T) *testutil.GitRepo {
	t.Helper()
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("README.md", "readme")
	repo.Commit("init")
	return repo
}

func mkdir(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(parts...)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLocateMainWorktree(t *testing.T) {
	repo := committedRepo(t)
	want := Location{
		Worktree:  repo.Dir,
		GitDir:    filepath.Join(repo.Dir, ".git"),
		CommonDir: filepath.Join(repo.Dir, ".git"),
		Main:      repo.Dir,
	}
	for _, dir := range []string{repo.Dir, mkdir(t, repo.Dir, "a", "b")} {
		got, err := Locate(ctx, dir)
		if err != nil || got != want {
			t.Errorf("Locate(%s) = %+v, %v; want %+v", dir, got, err, want)
		}
	}
}

func TestLocateLinkedWorktree(t *testing.T) {
	repo := committedRepo(t)
	wt := repo.AddWorktree("feature-x")
	got, err := Locate(ctx, wt)
	if err != nil {
		t.Fatal(err)
	}
	if got.Worktree != wt || got.Main != repo.Dir || got.CommonDir != filepath.Join(repo.Dir, ".git") {
		t.Fatalf("Locate = %+v", got)
	}
	if got.GitDir == got.CommonDir || filepath.Dir(filepath.Dir(got.GitDir)) != got.CommonDir {
		t.Fatalf("linked worktree git dir = %s", got.GitDir)
	}
}

func TestLocateOutsideRepository(t *testing.T) {
	if _, err := Locate(ctx, t.TempDir()); !errors.Is(err, ErrNotRepository) {
		t.Fatalf("Locate = %v, want ErrNotRepository", err)
	}
}

// Inside a git directory or a bare repository there is no worktree: an
// environment error, never "not a repository" (which would make it a
// non-git project).
func TestLocateRefusesGitDirectories(t *testing.T) {
	repo := committedRepo(t)
	bare := filepath.Join(t.TempDir(), "bare.git")
	repo.Git("clone", "-q", "--bare", repo.Dir, bare)
	for _, dir := range []string{filepath.Join(repo.Dir, ".git"), filepath.Join(repo.Dir, ".git", "refs"), bare, filepath.Join(bare, "refs")} {
		_, err := Locate(ctx, dir)
		if err == nil || errors.Is(err, ErrNotRepository) || apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Errorf("Locate(%s) = %v", dir, err)
		}
	}
}

// The location comes from the directory, not from variables that point git
// at another repository.
func TestLocateIgnoresRepositoryVariables(t *testing.T) {
	repo, other := committedRepo(t), committedRepo(t)
	t.Setenv("GIT_DIR", filepath.Join(other.Dir, ".git"))
	t.Setenv("GIT_WORK_TREE", other.Dir)
	t.Setenv("GIT_COMMON_DIR", filepath.Join(other.Dir, ".git"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(other.Dir, ".git", "index"))
	t.Setenv("GIT_CEILING_DIRECTORIES", repo.Dir)
	got, err := Locate(ctx, mkdir(t, repo.Dir, "sub"))
	if err != nil || got.Worktree != repo.Dir || got.CommonDir != filepath.Join(repo.Dir, ".git") {
		t.Fatalf("Locate = %+v, %v", got, err)
	}
	files, err := ListFiles(ctx, repo.Dir)
	if err != nil || !reflect.DeepEqual(files, []string{"README.md"}) {
		t.Fatalf("ListFiles = %q, %v", files, err)
	}
}

// codex-review-015 B1: git gets none of the inherited GIT_* variables —
// configuration injection (GIT_CONFIG_COUNT/KEY/VALUE, GIT_CONFIG_PARAMETERS,
// GIT_CONFIG_GLOBAL), tracing, repository location, exec path — whatever
// their letter case, and only the controlled ones are added back: system and
// global configuration replaced by an empty file, no optional locks, no
// prompts.
func TestFilterEnv(t *testing.T) {
	in := []string{
		"PATH=/bin", "HOME=/h", "LANG=C",
		"GIT_DIR=/x", "git_work_tree=/y", "GIT_AUTHOR_NAME=a", "GIT_INDEX_FILE=/i", "GIT_CEILING_DIRECTORIES=/c",
		"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=core.fsmonitor", "GIT_CONFIG_VALUE_0=x", "GIT_CONFIG_PARAMETERS='a'='b'",
		"GIT_CONFIG_GLOBAL=/g", "GIT_CONFIG_SYSTEM=/s", "GIT_CONFIG_NOSYSTEM=0", "GIT_TRACE=/t", "GIT_TRACE2_EVENT=/e",
		"GIT_EXEC_PATH=/p", "GIT_OPTIONAL_LOCKS=1", "Git_Trace_Setup=/u",
		"GIT_ALLOW_PROTOCOL=file:ext", "GIT_NO_LAZY_FETCH=0",
	}
	got := filterEnv(in)
	want := []string{
		"PATH=/bin", "HOME=/h", "LANG=C",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_TRACE2=0", "GIT_TRACE2_EVENT=0", "GIT_TRACE2_PERF=0",
		"GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0", "GIT_DISCOVERY_ACROSS_FILESYSTEM=1",
		"GIT_NO_LAZY_FETCH=1", "GIT_ALLOW_PROTOCOL=",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterEnv =\n%q\nwant\n%q", got, want)
	}
}

// codex-review-015 B3: an error never carries git's own message, which can
// quote configuration values, nor the directory git ran in.
func TestErrorsDoNotQuoteGit(t *testing.T) {
	repo := committedRepo(t)
	canary := testutil.NewCanary(t)
	_, err := run(ctx, repo.Dir, "rev-parse", "--verify", canary)
	if err == nil || apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("run = %v", err)
	}
	if msg := err.Error(); strings.Contains(msg, canary) || strings.Contains(msg, repo.Dir) || !strings.Contains(msg, "rev-parse") {
		t.Fatalf("error = %q", msg)
	}

	bad := testutil.NewGitRepo(t)
	bad.Git("config", "core.bare", canary)
	_, err = Locate(ctx, bad.Dir)
	if err == nil || apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("Locate with an invalid configuration = %v", err)
	}
	if msg := err.Error(); strings.Contains(msg, canary) || strings.Contains(msg, bad.Dir) {
		t.Fatalf("error = %q", msg)
	}
}

// codex-review-015 B1: git 2.32 is the oldest that reads GIT_CONFIG_GLOBAL
// and GIT_CONFIG_SYSTEM, without which the user's configuration cannot be
// kept out.
func TestParseGitVersion(t *testing.T) {
	cases := map[string]bool{
		"git version 2.32.0":                 true,
		"git version 2.56.0.windows.2":       true,
		"git version 2.39.5 (Apple Git-154)": true,
		"git version 3.0.0":                  true,
		"git version 2.31.8":                 false,
		"git version 1.99.0":                 false,
		"not git":                            false,
	}
	for out, want := range cases {
		if got := supportedVersion(out); got != want {
			t.Errorf("supportedVersion(%q) = %v, want %v", out, got, want)
		}
	}
}

// gitVariables returns the GIT_* entries of env, sorted.
func gitVariables(env []string) []string {
	var out []string
	for _, kv := range env {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(strings.ToUpper(name), "GIT_") {
			out = append(out, kv)
		}
	}
	slices.Sort(out)
	return out
}

// codex-review-016 B1: the version probe is the first git that runs, and
// runs before the version is known, so it must already run in the
// controlled environment: with trace2 off, which git 2.31 would otherwise
// enable from the user's global configuration before it even prints its
// version. A git older than 2.32 is refused after the probe, before any
// other command. (A stand-in for git 2.31.1 records how it was run; the
// real git 2.31.1 is the cli package's TestInitRefusesRealOldGit.)
func TestVersionProbeRunsInTheControlledEnvironment(t *testing.T) {
	// The stand-in is a git gitx has not seen, so its version is checked.
	dir, env, calls := testutil.FakeGit(t, "git version 2.31.1")
	for _, kv := range env {
		name, value, _ := strings.Cut(kv, "=")
		t.Setenv(name, value)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIT_TRACE2_EVENT", filepath.Join(t.TempDir(), "event"))
	t.Setenv("GIT_TRACE2", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))

	_, err := gitToRun(ctx)
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "2.32 or newer") {
		t.Fatalf("gitToRun = %v", err)
	}
	got := calls()
	if len(got) != 1 || !reflect.DeepEqual(got[0].Args, []string{"--version"}) {
		t.Fatalf("git ran %d times: %+v", len(got), got)
	}
	want := slices.Clone(controlledEnv)
	slices.Sort(want)
	if vars := gitVariables(got[0].Env); !reflect.DeepEqual(vars, want) {
		t.Fatalf("the probe ran with\n%q\nwant\n%q", vars, want)
	}
	for _, kv := range []string{"GIT_TRACE2=0", "GIT_TRACE2_EVENT=0", "GIT_TRACE2_PERF=0"} {
		if !slices.Contains(got[0].Env, kv) {
			t.Errorf("the probe ran without %s", kv)
		}
	}
}

// codex-review-015 N2: a cancelled context stops git.
func TestCanceledContextStopsGit(t *testing.T) {
	repo := committedRepo(t)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := ListFiles(canceled, repo.Dir); err == nil {
		t.Fatal("ListFiles ran with a cancelled context")
	}
}

func TestHeadBranchAndCommit(t *testing.T) {
	repo := committedRepo(t)
	head := repo.Git("rev-parse", "HEAD")
	if ref, err := HeadBranch(ctx, repo.Dir); err != nil || ref != "refs/heads/main" {
		t.Fatalf("HeadBranch = %q, %v", ref, err)
	}
	if sha, err := CommitOf(ctx, repo.Dir, "refs/heads/main"); err != nil || sha != head {
		t.Fatalf("CommitOf = %q, %v; want %s", sha, err, head)
	}
	repo.Git("checkout", "-q", "--detach")
	if ref, err := HeadBranch(ctx, repo.Dir); err != nil || ref != "" {
		t.Fatalf("detached HeadBranch = %q, %v", ref, err)
	}
	empty := testutil.NewGitRepo(t)
	if ref, err := HeadBranch(ctx, empty.Dir); err != nil || ref != "refs/heads/main" {
		t.Fatalf("unborn HeadBranch = %q, %v", ref, err)
	}
	if sha, err := CommitOf(ctx, empty.Dir, "refs/heads/main"); err != nil || sha != "" {
		t.Fatalf("unborn CommitOf = %q, %v", sha, err)
	}
}

// ValidRefName asks git whether a name is a well-formed ref, without the
// ref existing; an option-like name is never passed to git.
func TestValidRefName(t *testing.T) {
	repo := committedRepo(t)
	for ref, want := range map[string]bool{
		"refs/heads/main":         true,
		"refs/heads/release/1.0":  true,
		"refs/tags/v1.0.0":        true,
		"refs/heads/not-yet-made": true,
		"refs/heads/a..b":         false,
		"refs/heads/a b":          false,
		"refs/heads/a~1":          false,
		"refs/heads/a:b":          false,
		"refs/heads/.hidden":      false,
		"refs/heads/a.lock":       false,
		"refs/heads/a/":           false,
		"-refs/heads/main":        false,
		"":                        false,
	} {
		if got, err := ValidRefName(ctx, repo.Dir, ref); err != nil || got != want {
			t.Errorf("ValidRefName(%q) = %v, %v; want %v", ref, got, err, want)
		}
	}
}

func TestGitPath(t *testing.T) {
	repo := committedRepo(t)
	wt := repo.AddWorktree("feature-x")
	want := filepath.Join(repo.Dir, ".git", "info", "exclude")
	for _, dir := range []string{repo.Dir, wt} {
		if got, err := GitPath(ctx, dir, "info/exclude"); err != nil || got != want {
			t.Errorf("GitPath(%s) = %s, %v; want %s", dir, got, err, want)
		}
	}
}

// Code references are checked against a commit's tree: its regular files,
// by path from the top, and their line counts. A symbolic link, a
// directory or a file that is not committed is not a file of the tree.
// Expected values are written out. (TreeFiles and CountLines before
// review 068 B3; the same expectations through FileLines.)
func TestFileLines(t *testing.T) {
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("internal/auth/login.go", "package auth\n\nfunc Login() {}\n")
	repo.WriteFile("docs/客户 需求.md", "第一行\r\n第二行\r\n第三行")
	repo.WriteFile("empty.txt", "")
	repo.WriteFile("run.sh", "#!/bin/sh\necho hi\n")
	repo.WriteFile("link-target.txt", "internal/auth/login.go")
	repo.Git("add", "-A")
	repo.Git("update-index", "--chmod=+x", "run.sh")
	// A symbolic link and a submodule exist only in the index; committing
	// without add -A keeps them (codex-review-020 N2).
	object := repo.Git("hash-object", "-w", "link-target.txt")
	repo.Git("update-index", "--add", "--cacheinfo", "120000,"+object+",link.go")
	repo.Git("update-index", "--add", "--cacheinfo", "160000,"+strings.Repeat("1", 40)+",vendor/sub")
	repo.Git("commit", "-q", "-m", "init")
	head := repo.Git("rev-parse", "HEAD")
	tree := repo.Git("ls-tree", "-r", head)
	for _, entry := range []string{"120000 blob " + object + "\tlink.go", "160000 commit " + strings.Repeat("1", 40) + "\tvendor/sub",
		"100755 blob"} {
		if !strings.Contains(tree, entry) {
			t.Fatalf("the commit's tree lacks %q:\n%s", entry, tree)
		}
	}
	repo.WriteFile("later.go", "package later\n")

	asked := []string{"docs/客户 需求.md", "empty.txt", "internal/auth/login.go", "link-target.txt", "run.sh",
		"link.go", "vendor/sub", "vendor", "internal", "internal/auth", "internal/auth/login.go/x", "later.go", "none", ".", "/run.sh"}
	files, err := FileLines(ctx, repo.Dir, head, asked)
	want := map[string]int{"docs/客户 需求.md": 3, "empty.txt": 0, "internal/auth/login.go": 3, "link-target.txt": 1, "run.sh": 2}
	if err != nil || !reflect.DeepEqual(files, want) {
		t.Fatalf("FileLines = %v, %v; want %v", files, err, want)
	}
	if files, err := FileLines(ctx, repo.Dir, head, nil); err != nil || len(files) != 0 {
		t.Errorf("FileLines(no paths) = %v, %v", files, err)
	}
	// Only a full commit ID is accepted: an argument never reaches git as
	// an option.
	for _, commit := range []string{"--output=x", head[:12], strings.ToUpper(head) + "0"} {
		if _, err := FileLines(ctx, repo.Dir, commit, []string{"run.sh"}); apperr.ExitCode(err) != apperr.ExitInternal {
			t.Errorf("FileLines(%q) = %v; want an internal error", commit, err)
		}
	}
}
