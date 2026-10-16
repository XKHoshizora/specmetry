package cli

import (
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// cmdline is a command line that must be refused with exit 2, and whether
// its error is error.v1 (true) or text.
type cmdline struct {
	args   []string
	asJSON bool
}

// codex-review-027 N1 (implementation-plan M6): no command line error quotes
// what was typed. A secret canary, as a positional argument, a flag name or
// a flag value, is refused with invalid_input (exit 2) and is in no stdout,
// stderr or error.v1 field (message, details, current), and never in the
// data directory. With --json, wherever it is and whether or not parsing
// reached it, the error is a whole error.v1 and stderr is empty; without it,
// or when --json is only a value, after "--", or false, the error is text
// and stdout is empty. Nothing is written. Every kind of error:
//   - the root's unknown command, help for an unknown topic;
//   - a group's unknown command;
//   - extra arguments to a command that takes none, or a fixed number;
//   - an unknown flag or shorthand, a bad bool, a flag without its value,
//     a bad number (baseline history --since).
func TestCommandLineErrorsQuoteNothing(t *testing.T) {
	home, dir, _ := questionsProject(t)
	token := testutil.NewSecretCanary(t)
	scanner, err := secrets.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(scanner.Fields(secrets.Field{Pointer: "/x", Value: token})) == 0 {
		t.Fatal("the canary is not a secret to the rules")
	}
	cases := []cmdline{
		// The root.
		{[]string{token}, false},
		{[]string{token, "--json"}, true},
		{[]string{"--json", token}, true},
		{[]string{"--json"}, true},
		{[]string{"--" + token}, false},
		{[]string{"-" + token}, false},
		{[]string{"--" + token, "--json"}, true},
		{[]string{"help", token}, false},
		{[]string{"help", "source", token}, false},
		// Every group.
		{[]string{"source", token}, false},
		{[]string{"proposal", token}, false},
		{[]string{"baseline", token}, false},
		{[]string{"question", token}, false},
		{[]string{"source", token, "--json"}, true},
		{[]string{"proposal", "--json", token}, true},
		{[]string{"baseline", "--" + token, "--json"}, true},
		{[]string{"question", token, "--json"}, true},
		// Every command that takes no argument, or a fixed number.
		{[]string{"version", token}, false},
		{[]string{"doctor", token}, false},
		{[]string{"version", token, "--json"}, false}, // version has no --json
		{[]string{"init", token}, false},
		{[]string{"init", token, "--json"}, true},
		{[]string{"source", "list", token, "--json"}, true},
		{[]string{"proposal", "list", token}, false},
		{[]string{"proposal", "list", token, "--json"}, true},
		{[]string{"propose", token, "--json"}, true},
		{[]string{"review", token, "--json"}, true},
		{[]string{"baseline", "show", token, "--json"}, true},
		{[]string{"baseline", "history", token}, false},
		{[]string{"question", "list", token, "--json"}, true},
		{[]string{"proposal", "show", "a", token, "--json"}, true},
		{[]string{"source", "add", "a", token}, false},
		{[]string{"source", "remove", "a", token, "--json"}, true},
		{[]string{"source", "ack", "a", token, "--json"}, true},
		// Unknown flags and shorthands, before and after --json.
		{[]string{"proposal", "list", "--" + token}, false},
		{[]string{"proposal", "list", "--" + token, "--json"}, true},
		{[]string{"proposal", "list", "--json", "--" + token}, true},
		{[]string{"source", "list", "-" + token}, false},
		{[]string{"source", "list", "-" + token, "--json"}, true},
		{[]string{"review", "-" + token, "--json"}, true},
		{[]string{"doctor", "--" + token}, false},
		// Bad bools, before and after --json; --json=false, repeated --json.
		{[]string{"proposal", "list", "--pending=" + token}, false},
		{[]string{"proposal", "list", "--pending=" + token, "--json"}, true},
		{[]string{"proposal", "list", "--json", "--pending=" + token}, true},
		{[]string{"source", "list", "--json=false", "--refs=" + token}, false},
		{[]string{"source", "list", "--json", "--json=false", "--refs=" + token}, false},
		{[]string{"source", "list", "--json=false", "--json", "--refs=" + token}, true},
		{[]string{"baseline", "show", "--json=true", "--include-retired=" + token}, true},
		{[]string{"init", "--json=" + token}, false},
		// A flag without its value.
		{[]string{"init", "--name"}, false},
		{[]string{"init", "--json", "--name"}, true},
		{[]string{"source", "ack", "a", "--observed"}, false},
		// A bad number: --since is read by the command.
		{[]string{"baseline", "history", "--since", token}, false},
		{[]string{"baseline", "history", "--since", token, "--json"}, true},
		// --json as the value of a string flag, or after "--".
		{[]string{"init", "--name", "--json", "--" + token}, false},
		{[]string{"init", "--name=--json", "--" + token}, false},
		{[]string{"review", "--file", "--json", "--" + token}, false},
		{[]string{"propose", "--file=--json", "-" + token}, false},
		{[]string{"source", "ack", "a", "-m", "--json", "--" + token}, false},
		{[]string{"source", "ack", "a", "-m=--json", "--" + token}, false},
		{[]string{"proposal", "list", "--", "--json"}, false},
		{[]string{"proposal", "list", "--pending=" + token, "--", "--json"}, false},
		{[]string{"source", "list", "--unknown=--json"}, false},
	}
	before := closeState(t, home)
	for _, c := range cases {
		code, stdout, stderr := specmetry(t, dir, "", c.args...)
		what := strings.ReplaceAll(strings.Join(c.args, " "), token, "<token>")
		if code != apperr.ExitInvalidInput {
			t.Errorf("%s: exit %d, want 2\n%s%s", what, code, stdout, stderr)
		}
		if strings.Contains(stdout+stderr, token) {
			t.Errorf("%s quotes the canary:\n%s%s", what, stdout, stderr)
		}
		if c.asJSON {
			errorV1(t, what, stdout, apperr.InvalidInput)
			if stderr != "" {
				t.Errorf("%s: stderr with error.v1:\n%s", what, stderr)
			}
			continue
		}
		if stdout != "" || !strings.HasPrefix(stderr, "error: ") {
			t.Errorf("%s: want a text error only, got stdout %q stderr %q", what, stdout, stderr)
		}
	}
	if after := closeState(t, home); strings.Join(after, " ") != strings.Join(before, " ") {
		t.Fatalf("written:\n%v\n%v", before, after)
	}
	if hits := testutil.FindInTree(t, home.Path, token); len(hits) != 0 {
		t.Fatalf("the canary reached the data directory: %v", hits)
	}
}

// What works keeps working: the help of every level (exit 0, Usage), a
// group alone shows its help, help names commands, a suggestion names an
// existing command only, and a valid command with --json before its
// arguments.
func TestCommandLineStillWorks(t *testing.T) {
	_, dir, _ := questionsProject(t)
	for _, args := range [][]string{
		{}, {"--help"}, {"-h"}, {"help"}, {"help", "source"}, {"help", "source", "list"},
		{"source"}, {"source", "--help"}, {"proposal"}, {"baseline", "--help"}, {"question"}, {"question", "list", "--help"},
	} {
		code, stdout, stderr := specmetry(t, dir, "", args...)
		if code != apperr.ExitOK || !strings.Contains(stdout, "Usage:") {
			t.Errorf("%v: exit %d\n%s%s", args, code, stdout, stderr)
		}
	}
	if code, stdout, _ := specmetry(t, dir, "", "version"); code != apperr.ExitOK || stdout == "" {
		t.Errorf("version: exit %d", code)
	}
	// The root and the groups know --json now, so it may also come before
	// the command it is for, as any flag may.
	for _, args := range [][]string{{"source", "list", "--json"}, {"proposal", "list", "--json", "--pending"}, {"question", "list", "--json", "--all"},
		{"--json", "question", "list"}, {"question", "--json", "list", "--all"}} {
		if code, stdout, stderr := specmetry(t, dir, "", args...); code != apperr.ExitOK || !strings.HasPrefix(stdout, "{") {
			t.Errorf("%v: exit %d\n%s%s", args, code, stdout, stderr)
		}
	}
	for _, c := range []struct {
		args []string
		hint string
	}{
		{[]string{"sorce"}, "did you mean source?"},
		{[]string{"source", "lst"}, "did you mean list?"},
		{[]string{"proposal", "shw"}, "did you mean show?"},
		{[]string{"question", "answr"}, "did you mean answer?"},
	} {
		code, stdout, stderr := specmetry(t, dir, "", c.args...)
		if code != apperr.ExitInvalidInput || !strings.Contains(stderr, c.hint) || stdout != "" {
			t.Errorf("%v: exit %d, want 2 and %q\n%s%s", c.args, code, c.hint, stdout, stderr)
		}
	}
}
