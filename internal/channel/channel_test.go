package channel_test

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/termtext"
)

// env returns a lookup function over vars, as os.LookupEnv would see them.
func env(vars map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		v, ok := vars[name]
		return v, ok
	}
}

const canary = "SPECMETRY-CANARY-channel-value"

// architecture §5.1 rule 1: human_cli needs stdin and stdout to be
// terminals and no verified Agent variable set; anything else is agent_cli.
func TestDecide(t *testing.T) {
	cases := []struct {
		name          string
		stdin, stdout bool
		vars          map[string]string
		want          channel.Channel
		reasons       []string
	}{
		{"terminal, no marker", true, true, nil, channel.HumanCLI, nil},
		{"stdin not a terminal", false, true, nil, channel.AgentCLI, []string{"stdin is not a terminal"}},
		{"stdout not a terminal", true, false, nil, channel.AgentCLI, []string{"stdout is not a terminal"}},
		{"neither", false, false, nil, channel.AgentCLI, []string{"stdin is not a terminal", "stdout is not a terminal"}},
		{"CLAUDECODE set", true, true, map[string]string{"CLAUDECODE": canary}, channel.AgentCLI,
			[]string{"the Agent environment variable CLAUDECODE is set"}},
		{"CLAUDE_CODE_ENTRYPOINT set", true, true, map[string]string{"CLAUDE_CODE_ENTRYPOINT": canary}, channel.AgentCLI,
			[]string{"the Agent environment variable CLAUDE_CODE_ENTRYPOINT is set"}},
		{"both set, not a terminal", false, true, map[string]string{"CLAUDECODE": "1", "CLAUDE_CODE_ENTRYPOINT": canary}, channel.AgentCLI,
			[]string{"stdin is not a terminal", "the Agent environment variable CLAUDECODE is set", "the Agent environment variable CLAUDE_CODE_ENTRYPOINT is set"}},
		{"set but empty", true, true, map[string]string{"CLAUDECODE": "", "CLAUDE_CODE_ENTRYPOINT": ""}, channel.HumanCLI, nil},
		{"unrelated variables", true, true, map[string]string{"CLAUDE_CONFIG_DIR": canary, "TERM": "xterm"}, channel.HumanCLI, nil},
	}
	for _, c := range cases {
		d := channel.Decide(c.stdin, c.stdout, env(c.vars))
		if d.Channel != c.want || !reflect.DeepEqual(d.Reasons, c.reasons) {
			t.Errorf("%s: Decide = %+v, want %s %v", c.name, d, c.want, c.reasons)
		}
	}
}

// The variables' values are never part of the result.
func TestDecideNeverKeepsVariableValues(t *testing.T) {
	d := channel.Decide(true, true, env(map[string]string{"CLAUDECODE": canary, "CLAUDE_CODE_ENTRYPOINT": canary}))
	if strings.Contains(strings.Join(d.Reasons, " "), canary) {
		t.Fatalf("reasons contain a value: %v", d.Reasons)
	}
	err := d.RequireHuman("scope confirm")
	if err == nil || strings.Contains(err.Error(), canary) {
		t.Fatalf("RequireHuman = %v", err)
	}
}

func TestAgentVariablesAreTheVerifiedOnes(t *testing.T) {
	if got := channel.AgentVariables(); !reflect.DeepEqual(got, []string{"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT"}) {
		t.Fatalf("AgentVariables = %v", got)
	}
}

// architecture §5.1 rule 2: a human-only operation in agent_cli is refused
// with exit code 3 and names why, without values.
func TestRequireHuman(t *testing.T) {
	if err := channel.Decide(true, true, env(nil)).RequireHuman("scope confirm"); err != nil {
		t.Fatalf("human_cli refused: %v", err)
	}
	err := channel.Decide(false, true, env(map[string]string{"CLAUDECODE": canary})).RequireHuman("scope confirm")
	if apperr.ExitCode(err) != apperr.ExitHumanRequired {
		t.Fatalf("exit code %d for %v", apperr.ExitCode(err), err)
	}
	for _, want := range []string{"scope confirm", "human-only", "stdin is not a terminal", "CLAUDECODE is set"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err.Error(), want)
		}
	}
}

var op = channel.Operation{
	Name:    "scope confirm",
	Summary: []string{"target: refs/heads/main", "requirements: REQ-001, REQ-002"},
	Confirm: "REQ-001",
}

func guard(t *testing.T, d channel.Detection, input string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := channel.Guard(d, strings.NewReader(input), &out, op)
	return out.String(), err
}

// architecture §5.1 rule 2: in human_cli the operation proceeds only after
// the person types the confirmation text.
func TestGuardConfirmsInHumanChannel(t *testing.T) {
	human := channel.Decide(true, true, env(nil))
	for _, input := range []string{"REQ-001\n", "REQ-001\r\n", "  REQ-001  \n", "\tREQ-001\t\r\n"} {
		out, err := guard(t, human, input)
		if err != nil {
			t.Errorf("input %q: %v", input, err)
		}
		for _, want := range []string{"scope confirm", "target: refs/heads/main", "requirements: REQ-001, REQ-002", `Type "REQ-001" to confirm`} {
			if !strings.Contains(out, want) {
				t.Errorf("prompt lacks %q:\n%s", want, out)
			}
		}
	}
}

// The prompt is safe on any writer: Guard escapes the operation's name and
// summary for the terminal (termtext) and indents the lines a summary line
// continues on, so that a text from a proposal cannot pass for a line of
// the summary. The confirmation text is shown quoted and compared as it
// is.
func TestGuardShowsTheSummarySafely(t *testing.T) {
	human := channel.Decide(true, true, env(nil))
	rlo := string(rune(0x202e))
	op := channel.Operation{
		Name:    "review",
		Summary: []string{"proposal by agent:x: fix\x1b[2J\x07" + rlo + "\r\n  item 0: reject add requirement ref r1", "second\rline"},
		Confirm: "01J9Z3K5Q8R2T4V6W8X0Y2Z4A5",
	}
	var out bytes.Buffer
	if err := channel.Guard(human, strings.NewReader(op.Confirm+"\n"), &out, op); err != nil {
		t.Fatalf("confirmed: %v", err)
	}
	prompt := out.String()
	for _, r := range prompt {
		if r < 0x20 && r != '\n' || r == 0x7f || r >= 0x80 && r <= 0x9f || r >= 0x202a && r <= 0x202e {
			t.Fatalf("the prompt holds %U:\n%q", r, prompt)
		}
	}
	want := "review\n  proposal by agent:x: fix" + `\x1b[2J\a\u202e` + "\n      item 0: reject add requirement ref r1\n  second" + `\r` + "line\n" +
		`Type "01J9Z3K5Q8R2T4V6W8X0Y2Z4A5" to confirm, or press Enter to cancel: `
	if prompt != want {
		t.Fatalf("prompt:\n%q\nwant\n%q", prompt, want)
	}
	if err := channel.Guard(human, strings.NewReader(`\x1b`+"\n"), io.Discard, op); apperr.ExitCode(err) != apperr.ExitHumanRequired {
		t.Fatalf("the escaped form confirmed: %v", err)
	}
}

// Anything but the exact confirmation text, ended by Enter, cancels: exit
// code 3, nothing may be written. End of input cancels even after the
// confirmation text (architecture §5.1).
func TestGuardCancels(t *testing.T) {
	human := channel.Decide(true, true, env(nil))
	for _, input := range []string{"\n", "", "req-001\n", "REQ-0011\n", "yes\n", "y\n", "REQ-001", "  REQ-001  ", "REQ-001\r"} {
		_, err := guard(t, human, input)
		if apperr.ExitCode(err) != apperr.ExitHumanRequired || !strings.Contains(err.Error(), "not confirmed") {
			t.Errorf("input %q: %v", input, err)
		}
	}
}

// In agent_cli the guard refuses before asking anything.
func TestGuardRefusesAgentWithoutPrompting(t *testing.T) {
	agent := channel.Decide(true, true, env(map[string]string{"CLAUDECODE": canary}))
	out, err := guard(t, agent, "REQ-001\n")
	if apperr.ExitCode(err) != apperr.ExitHumanRequired {
		t.Fatalf("exit code %d: %v", apperr.ExitCode(err), err)
	}
	if out != "" {
		t.Fatalf("prompted an Agent:\n%s", out)
	}
}

// An operation needs confirmation text; an empty one is a programming error.
func TestGuardNeedsConfirmationText(t *testing.T) {
	err := channel.Guard(channel.Decide(true, true, env(nil)), strings.NewReader("\n"), &bytes.Buffer{}, channel.Operation{Name: "x"})
	if apperr.ExitCode(err) != apperr.ExitInternal {
		t.Fatalf("Guard without confirmation text = %v", err)
	}
}

// faultyWriter answers every write with write; writes counts the calls.
type faultyWriter struct {
	write  func(p []byte) (int, error)
	writes int
}

func (w *faultyWriter) Write(p []byte) (int, error) {
	w.writes++
	return w.write(p)
}

// readCounter counts the reads made from it.
type readCounter struct {
	r     io.Reader
	reads int
}

func (c *readCounter) Read(p []byte) (int, error) {
	c.reads++
	return c.r.Read(p)
}

// When the operation, its summary or the prompt cannot be shown in full,
// the guard fails with exit code 6 before reading anything, and the
// operation does not go ahead even though the confirmation text is
// waiting. Every write fails, so the fault is certain to happen. The same
// holds through the terminal escaping every command's output goes through
// (termtext.Writer; codex-review-031 B1): a short write without an error
// is still a short write there.
func TestGuardFailsWhenThePromptCannotBeShown(t *testing.T) {
	errOutput := errors.New("output closed")
	cases := []struct {
		name  string
		write func(p []byte) (int, error)
	}{
		{"write error", func([]byte) (int, error) { return 0, errOutput }},
		{"write error after writing everything", func(p []byte) (int, error) { return len(p), errOutput }},
		{"short write", func(p []byte) (int, error) { return len(p) / 2, nil }},
		{"nothing written", func([]byte) (int, error) { return 0, nil }},
	}
	for _, c := range cases {
		for _, escaped := range []bool{false, true} {
			name := c.name
			if escaped {
				name += ", through termtext.Writer"
			}
			t.Run(name, func(t *testing.T) {
				out := &faultyWriter{write: c.write}
				var w io.Writer = out
				if escaped {
					w = termtext.NewWriter(out)
				}
				in := &readCounter{r: strings.NewReader("REQ-001\n")}
				err := channel.Guard(channel.Decide(true, true, env(nil)), in, w, op)
				if out.writes == 0 {
					t.Fatal("Guard wrote nothing, so the fault did not happen")
				}
				if apperr.ExitCode(err) != apperr.ExitEnvironment {
					t.Errorf("Guard = %v, want exit code %d", err, apperr.ExitEnvironment)
				}
				if in.reads != 0 {
					t.Errorf("Guard read the confirmation %d times", in.reads)
				}
			})
		}
	}
	// Written whole, directly or escaped, the confirmation goes ahead.
	for _, escaped := range []bool{false, true} {
		var buf bytes.Buffer
		var w io.Writer = &buf
		if escaped {
			w = termtext.NewWriter(&buf)
		}
		in := &readCounter{r: strings.NewReader("REQ-001\n")}
		if err := channel.Guard(channel.Decide(true, true, env(nil)), in, w, op); err != nil || in.reads == 0 {
			t.Errorf("written whole (escaped %v): %v after %d reads", escaped, err, in.reads)
		}
	}
}

// A failure to read the confirmation is not a confirmation.
func TestGuardFailsWhenTheConfirmationCannotBeRead(t *testing.T) {
	in := io.MultiReader(strings.NewReader("REQ-001"), iotest.ErrReader(errors.New("input closed")))
	err := channel.Guard(channel.Decide(true, true, env(nil)), in, &bytes.Buffer{}, op)
	if apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("Guard = %v, want exit code %d", err, apperr.ExitEnvironment)
	}
}

// The Agent is named when one of its variables is set (actor
// agent:claude_code), and only then.
func TestDecideNamesTheAgent(t *testing.T) {
	env := func(vars map[string]string) func(string) (string, bool) {
		return func(k string) (string, bool) { v, ok := vars[k]; return v, ok }
	}
	for _, c := range []struct {
		vars map[string]string
		want string
	}{
		{map[string]string{"CLAUDECODE": "1"}, "claude_code"},
		{map[string]string{"CLAUDE_CODE_ENTRYPOINT": "cli"}, "claude_code"},
		{map[string]string{"CLAUDECODE": ""}, ""},
		{nil, ""},
	} {
		if d := channel.Decide(false, false, env(c.vars)); d.Agent != c.want {
			t.Errorf("%v: Agent %q, want %q", c.vars, d.Agent, c.want)
		}
	}
}
