package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// The CLI has no human-only command until WP1.4, WP1.6 and WP1.8. These
// tests add one that exists only here, so the whole path is exercised in a
// real terminal: channel detection in Execute, the guard, the exit code, and
// that nothing is written when the guard refuses.

const (
	argsEnv   = "SPECMETRY_TEST_ARGS"
	markerEnv = "SPECMETRY_TEST_MARKER"
)

// testHumanOnlyCommand is a stand-in for a human-only command: it writes
// the marker file only after the guard lets it proceed.
func testHumanOnlyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "test-human-op",
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			op := channel.Operation{Name: "test-human-op", Summary: []string{"would write the marker"}, Confirm: "REQ-001"}
			if err := channel.Guard(channel.FromContext(cmd.Context()), os.Stdin, cmd.OutOrStdout(), op); err != nil {
				return err
			}
			return os.WriteFile(os.Getenv(markerEnv), []byte("written"), 0o600)
		}),
	}
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().String("file", "", "")
	return cmd
}

// cliHelper runs the CLI with the extra command, the way main does.
func cliHelper() int {
	root := newRoot(os.Stdout, os.Stderr)
	root.AddCommand(testHumanOnlyCommand())
	code := executeRoot(context.Background(), root, strings.Fields(os.Getenv(argsEnv)), os.Stderr)
	fmt.Fprintf(os.Stderr, "\nexit=%d\n", code)
	return code
}

// cliRedirectedHelper is cliHelper with stdout going to a file while stdin
// stays the terminal, as in "specmetry ... > file".
func cliRedirectedHelper() int {
	f, err := os.CreateTemp("", "stdout")
	if err != nil {
		return 1
	}
	defer func() { _ = f.Close(); _ = os.Remove(f.Name()) }()
	terminal := os.Stdout
	os.Stdout = f
	root := newRoot(f, terminal)
	root.AddCommand(testHumanOnlyCommand())
	code := executeRoot(context.Background(), root, strings.Fields(os.Getenv(argsEnv)), terminal)
	fmt.Fprintf(terminal, "\nexit=%d\n", code)
	return code
}

// helperEnv is the helper's environment without the Agent variables (the
// test may itself run inside an Agent session), plus extra.
func helperEnv(t *testing.T, name string, extra ...string) []string {
	t.Helper()
	var env []string
	for _, kv := range testutil.HelperCommand(t, name).Env {
		key, _, _ := strings.Cut(kv, "=")
		if strings.EqualFold(key, "CLAUDECODE") || strings.EqualFold(key, "CLAUDE_CODE_ENTRYPOINT") {
			continue
		}
		env = append(env, kv)
	}
	return append(env, extra...)
}

type run1 struct {
	output  string
	written bool
}

// runCLI runs the CLI helper in a terminal (or not) with args and extra
// environment, answering the confirmation prompt with answer when given.
func runCLI(t *testing.T, helper string, terminal bool, args, answer string, extra ...string) run1 {
	t.Helper()
	marker := filepath.Join(t.TempDir(), "written")
	env := helperEnv(t, helper, append(extra, argsEnv+"="+args, markerEnv+"="+marker)...)
	cmd := testutil.HelperCommand(t, helper)
	var out string
	if terminal {
		var steps []testutil.PTYStep
		if answer != "" {
			steps = []testutil.PTYStep{{Expect: "press Enter to cancel:", Send: answer}}
		}
		out = testutil.RunInPTYSession(t, 60*time.Second, env, steps, cmd.Path, cmd.Args[1:]...).Output
	} else {
		cmd.Env = env
		b, _ := cmd.CombinedOutput()
		out = string(b)
	}
	_, err := os.Stat(marker)
	return run1{output: out, written: err == nil}
}

// MVP-CH-01 (CLI part, with a stand-in command): in a terminal without
// Agent variables the command asks for confirmation and proceeds only on
// the exact text.
func TestHumanOnlyCommandConfirmsInTerminal(t *testing.T) {
	testutil.TempHome(t)
	if r := runCLI(t, "cli", true, "test-human-op", "REQ-001\r"); !r.written || !strings.Contains(r.output, "exit=0") {
		t.Errorf("confirmed: not written\n%s", r.output)
	}
	for _, answer := range []string{"\r", "no\r"} {
		r := runCLI(t, "cli", true, "test-human-op", answer)
		if r.written || !strings.Contains(r.output, "was not confirmed") || !strings.Contains(r.output, "exit=3") {
			t.Errorf("answer %q: written=%v\n%s", answer, r.written, r.output)
		}
	}
	// --json does not skip the confirmation.
	if r := runCLI(t, "cli", true, "test-human-op --json", "\r"); r.written || !strings.Contains(r.output, `Type "REQ-001" to confirm`) {
		t.Errorf("--json: written=%v\n%s", r.written, r.output)
	}
}

// MVP-CH-01 (CLI part): an Agent variable refuses the operation with exit
// code 3 before any prompt; nothing is written and the variable's value
// appears neither in the output nor in the data directory.
func TestHumanOnlyCommandRefusesAgentInTerminal(t *testing.T) {
	home := testutil.TempHome(t)
	canaryValue := testutil.NewCanary(t)
	for _, name := range channel.AgentVariables() {
		r := runCLI(t, "cli", true, "test-human-op --json", "", name+"="+canaryValue)
		if r.written || strings.Contains(r.output, "to confirm") || !strings.Contains(r.output, "human-only operation") || !strings.Contains(r.output, "exit=3") {
			t.Errorf("%s: written=%v\n%s", name, r.written, r.output)
		}
		if strings.Contains(r.output, canaryValue) {
			t.Errorf("%s: the value was printed", name)
		}
	}
	if _, err := os.Stat(home.Path); err == nil {
		if hits := testutil.FindInTree(t, home.Path, canaryValue); len(hits) > 0 {
			t.Errorf("the value reached the data directory: %v", hits)
		}
	}
}

// MVP-CH-02 (CLI part): without a terminal on stdin or stdout the command
// is refused with exit code 3, whatever its flags.
func TestHumanOnlyCommandRefusesWithoutTerminal(t *testing.T) {
	testutil.TempHome(t)
	for _, args := range []string{"test-human-op", "test-human-op --json", "test-human-op --file -"} {
		r := runCLI(t, "cli", false, args, "")
		if r.written || !strings.Contains(r.output, "stdin is not a terminal") || !strings.Contains(r.output, "exit=3") {
			t.Errorf("pipes, %q: written=%v\n%s", args, r.written, r.output)
		}
	}
	r := runCLI(t, "cli-redirected", true, "test-human-op", "")
	if r.written || !strings.Contains(r.output, "stdout is not a terminal") || !strings.Contains(r.output, "exit=3") {
		t.Errorf("stdout redirected: written=%v\n%s", r.written, r.output)
	}
}

// MVP-CH-02: read-only commands run in either channel.
func TestReadOnlyCommandsRunInEitherChannel(t *testing.T) {
	testutil.TempHome(t)
	canaryValue := testutil.NewCanary(t)
	for _, args := range []string{"version", "doctor"} {
		for _, c := range []struct {
			name     string
			terminal bool
			extra    []string
		}{
			{"terminal", true, nil},
			{"terminal with CLAUDECODE", true, []string{"CLAUDECODE=" + canaryValue}},
			{"pipes", false, nil},
		} {
			r := runCLI(t, "cli", c.terminal, args, "", c.extra...)
			if !strings.Contains(r.output, "exit=0") || strings.Contains(r.output, "human-only") || strings.Contains(r.output, canaryValue) {
				t.Errorf("%s, %s:\n%s", args, c.name, r.output)
			}
		}
	}
}

// There is no flag that skips the confirmation.
func TestNoConfirmationSkipFlag(t *testing.T) {
	for _, flag := range []string{"--yes", "-y", "--force", "--no-confirm"} {
		if code, _, _ := execute(t, flag, "version"); code != apperr.ExitInvalidInput {
			t.Errorf("%s accepted", flag)
		}
	}
}
