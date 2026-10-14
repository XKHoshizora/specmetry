package channel_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
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
	for _, input := range []string{"REQ-001\n", "REQ-001\r\n", "  REQ-001  \n", "REQ-001"} {
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

// Anything but the exact confirmation text cancels: exit code 3, nothing
// may be written.
func TestGuardCancels(t *testing.T) {
	human := channel.Decide(true, true, env(nil))
	for _, input := range []string{"\n", "", "req-001\n", "REQ-0011\n", "yes\n", "y\n"} {
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
