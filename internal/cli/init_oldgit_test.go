package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// oldGitEnv names a real git 2.31.1 binary for TestInitRefusesRealOldGit.
const oldGitEnv = "SPECMETRY_TEST_GIT_2_31"

// initInFreshProcess runs `init --json --no-exclude` in dir in a new
// process (so git's version has not been checked yet) with extra
// environment, and returns the exit code, stdout and stderr.
func initInFreshProcess(t *testing.T, dir string, env ...string) (int, string, string) {
	t.Helper()
	cmd := testutil.HelperCommand(t, "cli", append(env, argsEnv+"=init --json --no-exclude")...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0, stdout.String(), stderr.String()
	case errors.As(err, &exit):
		return exit.ExitCode(), stdout.String(), stderr.String()
	default:
		t.Fatal(err)
		return 0, "", ""
	}
}

// requireOldGitRefusal checks the refusal of an old git: exit 6, the
// error.v1 that names the minimum version, and nothing written.
func requireOldGitRefusal(t *testing.T, code int, stdout, stderr string, home, repo string) {
	t.Helper()
	if code != apperr.ExitEnvironment {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	conforms(t, code, stdout)
	var env apperr.Envelope
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Error.Message, "git 2.32 or newer is required") {
		t.Fatalf("message = %q", env.Error.Message)
	}
	if exists(home) {
		t.Fatal("the data directory was created")
	}
	if exists(filepath.Join(repo, ".specmetry")) {
		t.Fatal("a marker was written")
	}
}

// codex-review-016 B1, through the public init path: with a git older
// than 2.32, the version probe is the only git that runs, and it runs in
// the controlled environment, trace2 off. This uses a stand-in that
// reports version 2.31.1 and records how it was run; it is not a real old
// git (that is TestInitRefusesRealOldGit).
func TestInitRefusesOldGitBeforeAnythingElse(t *testing.T) {
	home := testutil.TempHome(t)
	repo := committedRepo(t)
	dir, env, calls := testutil.FakeGit(t, "git version 2.31.1")
	trace := filepath.Join(t.TempDir(), "event")
	code, stdout, stderr := initInFreshProcess(t, repo.Dir, append(env,
		"PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GIT_TRACE2_EVENT="+trace, "GIT_TRACE2=1", "GIT_CONFIG_PARAMETERS='trace2.eventtarget'='"+trace+"'")...)
	requireOldGitRefusal(t, code, stdout, stderr, home.Path, repo.Dir)
	got := calls()
	if len(got) != 1 || !reflect.DeepEqual(got[0].Args, []string{"--version"}) {
		t.Fatalf("git ran %d times before the refusal: %+v", len(got), got)
	}
	var vars []string
	for _, kv := range got[0].Env {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(strings.ToUpper(name), "GIT_") {
			vars = append(vars, kv)
		}
	}
	slices.Sort(vars)
	want := []string{
		"GIT_ALLOW_PROTOCOL=",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_DISCOVERY_ACROSS_FILESYSTEM=1", "GIT_NO_LAZY_FETCH=1", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0",
		"GIT_TRACE2=0", "GIT_TRACE2_EVENT=0", "GIT_TRACE2_PERF=0",
	}
	if !reflect.DeepEqual(vars, want) {
		t.Fatalf("the version probe ran with\n%q\nwant\n%q", vars, want)
	}
}

// codex-review-016 B1 against a real git 2.31.1, the last version without
// GIT_CONFIG_GLOBAL: it reads the user's global configuration, where
// trace2 targets and trace2.envVars would make even `git --version` write
// a trace holding an environment variable's value. Set SPECMETRY_TEST_GIT_2_31
// to such a binary to run it; CI has none, and runs the stand-in above.
func TestInitRefusesRealOldGit(t *testing.T) {
	old := os.Getenv(oldGitEnv)
	if old == "" {
		t.Skipf("set %s to a git 2.31.1 binary to run this compatibility check", oldGitEnv)
	}
	if runtime.GOOS == "windows" {
		t.Skip("the compatibility check runs on Linux or macOS")
	}
	if out, err := exec.Command(old, "--version").Output(); err != nil || strings.TrimSpace(string(out)) != "git version 2.31.1" {
		t.Fatalf("%s is not git 2.31.1: %q, %v", old, out, err)
	}
	home := testutil.TempHome(t)
	repo := committedRepo(t) // built with the git on PATH, before the old one replaces it

	bin := t.TempDir()
	if err := os.Symlink(old, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	gitHome, traces := t.TempDir(), t.TempDir()
	targets := map[string]string{"eventTarget": "event", "normalTarget": "normal", "perfTarget": "perf"}
	config := "[trace2]\n\tenvVars = SPECMETRY_TRACE_CANARY\n\tconfigParams = *\n"
	for key, file := range targets {
		config += "\t" + key + " = " + filepath.Join(traces, file) + "\n"
	}
	if err := os.WriteFile(filepath.Join(gitHome, ".gitconfig"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	canary := testutil.NewCanary(t)
	env := []string{
		"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + gitHome, "XDG_CONFIG_HOME=" + gitHome, "SPECMETRY_TRACE_CANARY=" + canary,
	}

	// The configuration is live: the old git traces its own --version,
	// with the canary.
	control := exec.Command(filepath.Join(bin, "git"), "--version")
	control.Env = append(os.Environ(), env...)
	if err := control.Run(); err != nil {
		t.Fatal(err)
	}
	if hits := testutil.FindInTree(t, traces, canary); len(hits) == 0 {
		t.Fatal("git 2.31.1 wrote no trace with the canary, so this test would prove nothing")
	}
	for _, file := range targets {
		_ = os.Remove(filepath.Join(traces, file))
	}
	before, refs := gitState(t, repo, true)

	code, stdout, stderr := initInFreshProcess(t, repo.Dir, env...)
	requireOldGitRefusal(t, code, stdout, stderr, home.Path, repo.Dir)
	if entries, _ := os.ReadDir(traces); len(entries) != 0 {
		t.Fatalf("init made git 2.31.1 write %d trace files", len(entries))
	}
	if strings.Contains(stdout+stderr, canary) {
		t.Fatal("the canary reached the output")
	}
	after, afterRefs := gitState(t, repo, false)
	if d := testutil.DiffSnapshots(before, after); d != "" || afterRefs != refs {
		t.Fatalf("the repository changed:\n%s", d)
	}
}
