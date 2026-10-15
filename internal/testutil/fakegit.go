package testutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	fakeGitRecordEnv   = "SPECMETRY_FAKE_GIT_RECORD"
	fakeGitVersionEnv  = "SPECMETRY_FAKE_GIT_VERSION"
	passthroughRealEnv = "SPECMETRY_FAKE_GIT_REAL"
	passthroughModeEnv = "SPECMETRY_FAKE_GIT_MODE"
	traceControlEnv    = "SPECMETRY_FAKE_GIT_TRACE_CONTROL"
	traceTestEnv       = "SPECMETRY_FAKE_GIT_TRACE_TEST"
	traceCodeEnv       = "SPECMETRY_FAKE_GIT_TRACE_CODE"
)

// clearModes are variables that override the modes of fake gits set
// earlier in the test.
var clearModes = []string{passthroughRealEnv + "=", traceControlEnv + "="}

// FakeGitCall is one run of a fake git: its arguments and environment.
type FakeGitCall struct {
	Args []string `json:"args"`
	Env  []string `json:"env"`
}

// FakeGit makes a stand-in for git, not a real one: a directory holding a
// copy of the test binary named git, which, run under that name, records
// how it was run and prints version (for example "git version 2.31.1") for
// any command. It returns the directory to put first on PATH, the
// variables the process that runs it needs, and a function that returns
// the recorded calls. The package's TestMain must call DispatchFakeGit
// first.
func FakeGit(t testing.TB, version string) (dir string, env []string, calls func() []FakeGitCall) {
	t.Helper()
	dir, record, calls := fakeGitDir(t)
	return dir, append([]string{fakeGitRecordEnv + "=" + record, fakeGitVersionEnv + "=" + version}, clearModes...), calls
}

// FakeTraceGit makes a stand-in for git that runs no git: it reports
// version 2.56.0 to git --version and, run as git cat-file, writes a
// scripted trace2 event stream to the file GIT_TRACE2_EVENT names (control
// without GIT_NO_LAZY_FETCH=1, test with it) and exits with code. Any
// other command exits 0. It returns what FakeGit returns. The package's
// TestMain must call DispatchFakeGit first.
func FakeTraceGit(t testing.TB, control, test string, code int) (dir string, env []string, calls func() []FakeGitCall) {
	t.Helper()
	dir, record, calls := fakeGitDir(t)
	files := t.TempDir()
	controlFile, testFile := filepath.Join(files, "control.jsonl"), filepath.Join(files, "test.jsonl")
	for path, content := range map[string]string{controlFile: control, testFile: test} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir, []string{fakeGitRecordEnv + "=" + record, fakeGitVersionEnv + "=git version 2.56.0", passthroughRealEnv + "=",
		traceControlEnv + "=" + controlFile, traceTestEnv + "=" + testFile, traceCodeEnv + "=" + strconv.Itoa(code)}, calls
}

// Modes of a PassthroughGit.
const (
	// PassAsIs runs the real git as it was run.
	PassAsIs = "as-is"
	// PassWithoutNoLazyFetch hides GIT_NO_LAZY_FETCH from the real git,
	// which then fetches a missing object as a git that does not honour
	// the variable (git 2.32) does.
	PassWithoutNoLazyFetch = "without-no-lazy-fetch"
	// PassWithoutTrace hides GIT_TRACE2_EVENT from the real git, which
	// then writes no trace2 event.
	PassWithoutTrace = "without-trace"
	// PassWithoutLsFilesSparse refuses ls-files --sparse with a usage
	// error (exit 129) without running git, as a git older than the option
	// does.
	PassWithoutLsFilesSparse = "without-ls-files-sparse"
	// PassHang answers git --version and, run in any other way, waits a
	// minute and fails, without running git.
	PassHang = "hang"
)

// PassthroughGit makes a stand-in for git around the real one: a
// directory holding a copy of the test binary named git, which, run under
// that name, records how it was run and runs the git PATH finds now with
// the same arguments, its environment changed as mode says. It returns
// the directory to put first on PATH, the variables the process that runs
// it needs, and a function that returns the recorded calls. The package's
// TestMain must call DispatchFakeGit first.
func PassthroughGit(t testing.TB, mode string) (dir string, env []string, calls func() []FakeGitCall) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	if real, err = filepath.Abs(real); err != nil {
		t.Fatal(err)
	}
	dir, record, calls := fakeGitDir(t)
	return dir, []string{fakeGitRecordEnv + "=" + record, passthroughRealEnv + "=" + real, passthroughModeEnv + "=" + mode, traceControlEnv + "="}, calls
}

// fakeGitDir copies the test binary as git into a new directory and
// returns it, the file the copy records its calls in, and their reader.
func fakeGitDir(t testing.TB) (dir, record string, calls func() []FakeGitCall) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir = t.TempDir()
	name := "git"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	copyFile(t, self, filepath.Join(dir, name))
	record = filepath.Join(t.TempDir(), "calls.jsonl")
	calls = func() []FakeGitCall {
		b, err := os.ReadFile(record)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			t.Fatal(err)
		}
		var out []FakeGitCall
		for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			var c FakeGitCall
			if err := json.Unmarshal([]byte(line), &c); err != nil {
				t.Fatal(err)
			}
			out = append(out, c)
		}
		return out
	}
	return dir, record, calls
}

func copyFile(t testing.TB, from, to string) {
	t.Helper()
	src, err := os.Open(from)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = src.Close() }()
	dst, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		t.Fatal(err)
	}
	if err := dst.Close(); err != nil {
		t.Fatal(err)
	}
}

const fakeHelperMarkerEnv = "SPECMETRY_FAKE_REMOTE_HELPER_MARKER"

// FakeRemoteHelper puts a stand-in git remote helper named
// git-remote-<name> in dir, a copy of the test binary (so that git runs it
// on Windows too), and returns the marker file it appends to whenever git
// runs it; it does nothing else and fails. The marker goes to git through
// the environment, which this sets for the rest of the test. The package's
// TestMain must call DispatchFakeGit first.
func FakeRemoteHelper(t *testing.T, dir, name string) (marker string) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	file := "git-remote-" + name
	if runtime.GOOS == "windows" {
		file += ".exe"
	}
	copyFile(t, self, filepath.Join(dir, file))
	marker = filepath.Join(t.TempDir(), "remote-helper-ran")
	t.Setenv(fakeHelperMarkerEnv, marker)
	return marker
}

// DispatchFakeGit must be called first in the TestMain of a package that
// uses FakeGit, FakeTraceGit, PassthroughGit or FakeRemoteHelper. When the test binary
// runs as a fake git, it records the call, then prints the version and
// exits, or runs the real git; as a remote helper, it appends to its
// marker and fails.
func DispatchFakeGit() {
	name := strings.TrimSuffix(strings.ToLower(filepath.Base(os.Args[0])), ".exe")
	if strings.HasPrefix(name, "git-remote-") {
		if f, err := os.OpenFile(os.Getenv(fakeHelperMarkerEnv), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600); err == nil {
			_, _ = f.WriteString("ran\n")
			_ = f.Close()
		}
		os.Exit(1)
	}
	if name != "git" {
		return
	}
	b, _ := json.Marshal(FakeGitCall{Args: os.Args[1:], Env: os.Environ()})
	if f, err := os.OpenFile(os.Getenv(fakeGitRecordEnv), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600); err == nil {
		_, _ = f.Write(append(b, '\n'))
		_ = f.Close()
	}
	if os.Getenv(traceControlEnv) != "" {
		os.Exit(fakeTrace())
	}
	if real := os.Getenv(passthroughRealEnv); real != "" {
		os.Exit(passthrough(real, os.Getenv(passthroughModeEnv)))
	}
	fmt.Println(os.Getenv(fakeGitVersionEnv))
	os.Exit(0)
}

// fakeTrace is FakeTraceGit's git: it returns the exit status.
func fakeTrace() int {
	args := os.Args[1:]
	switch {
	case slices.Equal(args, []string{"--version"}):
		fmt.Println(os.Getenv(fakeGitVersionEnv))
		return 0
	case !slices.Contains(args, "cat-file"):
		return 0
	}
	script := os.Getenv(traceControlEnv)
	if os.Getenv("GIT_NO_LAZY_FETCH") == "1" {
		script = os.Getenv(traceTestEnv)
	}
	events, err := os.ReadFile(script)
	if err != nil {
		return 127
	}
	f, err := os.OpenFile(os.Getenv("GIT_TRACE2_EVENT"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return 127
	}
	_, err = f.Write(events)
	if cerr := f.Close(); err != nil || cerr != nil {
		return 127
	}
	code, err := strconv.Atoi(os.Getenv(traceCodeEnv))
	if err != nil {
		return 127
	}
	return code
}

// passthrough runs the real git as PassthroughGit's mode says and returns
// its exit status.
func passthrough(real, mode string) int {
	args := os.Args[1:]
	if mode == PassHang && !slices.Equal(args, []string{"--version"}) {
		time.Sleep(time.Minute)
		return 1
	}
	if mode == PassWithoutLsFilesSparse && slices.Contains(args, "ls-files") && slices.Contains(args, "--sparse") {
		fmt.Fprintln(os.Stderr, "error: unknown option `sparse'")
		return 129
	}
	hide := map[string]string{PassWithoutNoLazyFetch: "GIT_NO_LAZY_FETCH", PassWithoutTrace: "GIT_TRACE2_EVENT"}[mode]
	var env []string
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); hide == "" || !strings.EqualFold(name, hide) {
			env = append(env, kv)
		}
	}
	cmd := exec.Command(real, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = env
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		return exit.ExitCode()
	case err != nil:
		return 127
	}
	return 0
}
