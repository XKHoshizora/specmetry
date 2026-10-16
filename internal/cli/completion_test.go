package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// buildSpecmetry builds the real CLI binary (cmd/specmetry) into a
// temporary directory and returns its path. It is called before the test
// changes directory: the module is two levels above this package.
func buildSpecmetry(t *testing.T) string {
	t.Helper()
	module, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "specmetry")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-tags", "nowebui", "-o", bin, "./cmd/specmetry")
	build.Dir = module
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// runSpecmetry runs the binary in dir as its own process, with the test's
// environment (its data directory) without any BASH_COMP_DEBUG_FILE, plus
// env, and returns the exit code and the process's own stdout and stderr.
func runSpecmetry(t *testing.T, bin, dir string, env []string, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(kv), "BASH_COMP_DEBUG_FILE=") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	cmd.Env = append(cmd.Env, env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0, stdout.String(), stderr.String()
	case errors.As(err, &exit):
		return exit.ExitCode(), stdout.String(), stderr.String()
	}
	t.Fatalf("run %v: %v", args, err)
	return 0, "", ""
}

// treeDigest is every regular file under root with the hash of its content.
func treeDigest(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		lines = append(lines, p+" "+hex.EncodeToString(sum[:]))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// codex-review-029 B1: cobra adds its hidden shell completion command
// (__complete, alias __completeNoDesc) only while it executes, after
// secureCommandLine, and that command writes cobra's raw errors to the
// process's own stderr and to BASH_COMP_DEBUG_FILE. Specmetry provides no
// shell completion: run as a real process, a command line that would run
// it is refused with invalid_input (exit 2) and nothing is completed (no
// directive such as ":0"). A secret canary in it is in neither the
// process's stdout nor its stderr, BASH_COMP_DEBUG_FILE is not created,
// nor changed when it exists, and the data directory does not change by a
// byte. --json before the completion command asks for error.v1; one in the
// command line to complete belongs to that line and does not.
func TestShellCompletionIsRefused(t *testing.T) {
	bin := buildSpecmetry(t)
	home, dir, _ := questionsProject(t)
	canary := testutil.NewSecretCanary(t)
	debug := t.TempDir()
	existing := filepath.Join(debug, "existing.log")
	if err := os.WriteFile(existing, []byte("before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := []string{"__complete", "proposal", "list", "--pending=" + canary, ""}
	b := []string{"__completeNoDesc", "review", "--" + canary, ""}
	cases := []struct {
		args   []string
		asJSON bool
		debug  string // BASH_COMP_DEBUG_FILE, if set
	}{
		{a, false, ""},
		{b, false, ""},
		{append([]string{"--json"}, a...), true, ""},
		{append([]string{"--json=true"}, b...), true, ""},
		{append([]string{"--json=false"}, a...), false, ""},
		{append([]string{"--json", "--json=false"}, b...), false, ""},
		{append([]string{"--json"}, "__complete", "question", "answer", "Q-001", "-m", canary, ""), true, ""},
		{[]string{"__complete", "proposal", "list", "--json", "--pending=" + canary, ""}, false, ""},
		{[]string{"__completeNoDesc"}, false, ""},
		{a, false, filepath.Join(debug, "a.log")},
		{b, false, filepath.Join(debug, "b.log")},
		{append([]string{"--json"}, a...), true, filepath.Join(debug, "json.log")},
		{a, false, existing},
		{b, false, existing},
	}
	for _, c := range cases {
		var env []string
		if c.debug != "" {
			env = append(env, "BASH_COMP_DEBUG_FILE="+c.debug)
		}
		before := treeDigest(t, home.Path)
		code, stdout, stderr := runSpecmetry(t, bin, dir, env, c.args...)
		what := strings.ReplaceAll(strings.Join(c.args, " "), canary, "<canary>")
		if c.debug != "" {
			what += " (BASH_COMP_DEBUG_FILE)"
		}
		if code != apperr.ExitInvalidInput {
			t.Errorf("%s: exit %d, want 2\n%s%s", what, code, stdout, stderr)
		}
		if strings.Contains(stdout+stderr, canary) {
			t.Errorf("%s: the canary is in the output:\n%s%s", what, stdout, stderr)
		}
		if strings.Contains(stdout, ":0") || strings.Contains(stderr, "Completion ended") || strings.Contains(stderr, "[Debug]") {
			t.Errorf("%s: completion ran:\n%s%s", what, stdout, stderr)
		}
		if c.asJSON {
			errorV1(t, what, stdout, apperr.InvalidInput)
			if stderr != "" {
				t.Errorf("%s: stderr with error.v1:\n%s", what, stderr)
			}
		} else if stdout != "" || !strings.HasPrefix(stderr, "error: ") || !strings.Contains(stderr, "shell completion") {
			t.Errorf("%s: want the refusal as text only, got stdout %q stderr %q", what, stdout, stderr)
		}
		switch b, err := os.ReadFile(c.debug); {
		case c.debug == existing && string(b) != "before\n":
			t.Errorf("%s: the existing debug file changed: %q", what, b)
		case c.debug != "" && c.debug != existing && !errors.Is(err, fs.ErrNotExist):
			t.Errorf("%s: the debug file was written: %q (%v)", what, b, err)
		}
		if after := treeDigest(t, home.Path); after != before {
			t.Errorf("%s: the data directory changed", what)
		}
	}
}

// A business argument that happens to be __complete or __completeNoDesc
// is not taken for the completion command: what cobra would run decides,
// not the word. These reach their own command and its own error; nothing
// is written.
func TestShellCompletionWordIsData(t *testing.T) {
	bin := buildSpecmetry(t)
	home, dir, _ := questionsProject(t)
	before := closeState(t, home)
	for _, c := range []struct {
		args []string
		exit int
		want string
	}{
		{[]string{"question", "answer", "Q-001", "-m", "__complete"}, apperr.ExitHumanRequired, "human-only"},
		{[]string{"proposal", "show", "__completeNoDesc", "--json"}, apperr.ExitInvalidInput, "not a ULID"},
		{[]string{"source", "add", "__complete", "--json"}, apperr.ExitInvalidInput, "does not exist"},
		{[]string{"--", "__complete", "proposal"}, apperr.ExitInvalidInput, "unknown command"},
		{[]string{"--unknown", "__complete", "proposal"}, apperr.ExitInvalidInput, "an unknown flag"},
		{[]string{"help", "__complete"}, apperr.ExitInvalidInput, "unknown help topic"},
	} {
		code, stdout, stderr := runSpecmetry(t, bin, dir, nil, c.args...)
		if code != c.exit || !strings.Contains(stdout+stderr, c.want) || strings.Contains(stdout+stderr, "shell completion") {
			t.Errorf("%v: exit %d, want %d and %q\n%s%s", c.args, code, c.exit, c.want, stdout, stderr)
		}
	}
	if after := closeState(t, home); strings.Join(after, " ") != strings.Join(before, " ") {
		t.Fatalf("written:\n%v\n%v", before, after)
	}
}
