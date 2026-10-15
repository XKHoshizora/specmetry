package testutil

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	fakeGitRecordEnv  = "SPECMETRY_FAKE_GIT_RECORD"
	fakeGitVersionEnv = "SPECMETRY_FAKE_GIT_VERSION"
)

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
	record := filepath.Join(t.TempDir(), "calls.jsonl")
	env = []string{fakeGitRecordEnv + "=" + record, fakeGitVersionEnv + "=" + version}
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
	return dir, env, calls
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
// uses FakeGit or FakeRemoteHelper. When the test binary runs as a fake
// git, it records the call, prints the version and exits; as a remote
// helper, it appends to its marker and fails.
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
	fmt.Println(os.Getenv(fakeGitVersionEnv))
	os.Exit(0)
}
