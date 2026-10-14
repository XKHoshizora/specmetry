// Package channel tells whether the CLI is working for a person at an
// interactive terminal (human_cli) or for an Agent or a script (agent_cli),
// and guards human-only operations (docs/architecture.md §5.1).
//
// This prevents an Agent from performing a human-only operation by mistake.
// It is not authentication: an Agent that runs commands in a
// pseudo-terminal and clears the marker variables is taken for a person.
package channel

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// Channel is a CLI channel.
type Channel string

// CLI channels.
const (
	HumanCLI Channel = "human_cli"
	AgentCLI Channel = "agent_cli"
)

// agentVariables are the verified environment variables that mark a process
// run by an Agent (architecture §5.1). Only whether one is set and not empty
// matters; its value is never kept, logged or printed, because environment
// values are C0 data (architecture §10.1).
var agentVariables = []string{"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT"}

// AgentVariables returns the verified Agent environment variables.
func AgentVariables() []string { return append([]string(nil), agentVariables...) }

// Detection is the channel of one command and, for agent_cli, why. Reasons
// name streams and variables, never values.
type Detection struct {
	Channel Channel
	Reasons []string
}

// Decide identifies the channel: human_cli when stdin and stdout are
// terminals and no Agent variable is set, agent_cli otherwise. stderr plays
// no part. lookupEnv is os.LookupEnv outside tests.
func Decide(stdinTerminal, stdoutTerminal bool, lookupEnv func(string) (string, bool)) Detection {
	var reasons []string
	if !stdinTerminal {
		reasons = append(reasons, "stdin is not a terminal")
	}
	if !stdoutTerminal {
		reasons = append(reasons, "stdout is not a terminal")
	}
	for _, name := range agentVariables {
		if v, ok := lookupEnv(name); ok && v != "" {
			reasons = append(reasons, "the Agent environment variable "+name+" is set")
		}
	}
	if len(reasons) > 0 {
		return Detection{Channel: AgentCLI, Reasons: reasons}
	}
	return Detection{Channel: HumanCLI}
}

// RequireHuman refuses a human-only operation outside human_cli with
// human_required (exit code 3).
func (d Detection) RequireHuman(operation string) error {
	if d.Channel == HumanCLI {
		return nil
	}
	return apperr.New(apperr.HumanRequired, fmt.Sprintf(
		"%s is a human-only operation and this command is not running for a person at an interactive terminal (%s); run it yourself in a separate terminal",
		operation, strings.Join(d.Reasons, "; ")))
}

// Operation describes a human-only operation for its confirmation: what is
// about to happen, and the text the person must type to go ahead (an
// object key or a project name).
type Operation struct {
	Name    string
	Summary []string
	Confirm string
}

// Guard lets a human-only operation proceed: it refuses outside human_cli
// without asking anything, and in human_cli shows the operation on out and
// reads one line from in, which in human_cli is the terminal. Only the exact
// confirmation text goes ahead; anything else, an empty line or the end of
// input cancels with human_required (exit code 3). There is no way to skip
// the confirmation.
func Guard(d Detection, in io.Reader, out io.Writer, op Operation) error {
	if op.Confirm == "" {
		return apperr.New(apperr.Internal, "human-only operation "+op.Name+" has no confirmation text")
	}
	if err := d.RequireHuman(op.Name); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(out, op.Name)
	for _, line := range op.Summary {
		_, _ = fmt.Fprintln(out, "  "+line)
	}
	_, _ = fmt.Fprintf(out, "Type %q to confirm, or press Enter to cancel: ", op.Confirm)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return apperr.Wrap(apperr.Environment, err, "cannot read the confirmation")
	}
	if strings.TrimSpace(line) != op.Confirm {
		return apperr.New(apperr.HumanRequired, op.Name+" was not confirmed; nothing was changed")
	}
	return nil
}

// Detect identifies the channel of a process from its stdin and stdout and
// the environment. A terminal is a tty on Unix and a console on Windows; a
// terminal without a console, such as mintty without winpty, is not one.
func Detect(stdin, stdout *os.File) Detection {
	return Decide(isTerminal(stdin), isTerminal(stdout), os.LookupEnv)
}

func isTerminal(f *os.File) bool { return f != nil && term.IsTerminal(int(f.Fd())) }

type contextKey struct{}

// NewContext returns ctx carrying d, the channel detected for the command.
func NewContext(ctx context.Context, d Detection) context.Context {
	return context.WithValue(ctx, contextKey{}, d)
}

// FromContext returns the detection stored by NewContext. Without one it
// returns agent_cli, the safe side: a human-only operation is refused.
func FromContext(ctx context.Context) Detection {
	if d, ok := ctx.Value(contextKey{}).(Detection); ok {
		return d
	}
	return Detection{Channel: AgentCLI, Reasons: []string{"the channel was not detected"}}
}
