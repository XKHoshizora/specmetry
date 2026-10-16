package cli

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// The errors about the command line itself quote nothing that was typed
// (codex-review-025 B1, codex-review-027 N1; architecture §9.1): what was
// typed is input not yet scanned for secrets. cobra and pflag quote what
// they cannot use: an unknown command or flag, a value a flag does not
// take, an extra argument, a help topic. So none of their messages reaches
// the output; secureCommandLine sets fixed ones up for every command, in
// one place.

// secureCommandLine sets up root and every command under it:
//   - a flag that cannot be parsed is flagError, for every command;
//   - a command with subcommands (the root, every group) refuses an
//     unknown command, shows its help when given none, and takes --json
//     so that the refusal can be error.v1;
//   - a command's own argument check (cobra.NoArgs, cobra.ExactArgs, ...)
//     keeps its rule, but not its message;
//   - help refuses a topic it does not know.
func secureCommandLine(root *cobra.Command) {
	root.SetFlagErrorFunc(flagError)
	root.SetHelpCommand(newHelpCmd())
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		switch {
		case c.HasSubCommands():
			secureGroup(c)
		case c.Args != nil:
			c.Args = quotingNothing(c.Args)
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}

// secureGroup makes a command with subcommands refuse an unknown one.
// Without Args, cobra quotes it (the root), or shows the help and exits 0
// (a group; codex-review-027 B2).
func secureGroup(c *cobra.Command) {
	if c.Flags().Lookup("json") == nil {
		c.Flags().Bool("json", false, "print errors as error.v1")
	}
	c.SuggestionsMinimumDistance = 2 // cobra's default, which SuggestionsFor does not apply
	c.Args = unknownCommand
	if c.Run == nil && c.RunE == nil {
		c.RunE = func(cmd *cobra.Command, _ []string) error {
			if wantsJSON(cmd) {
				return apperr.New(apperr.InvalidInput, "name a command (see "+cmd.CommandPath()+" --help)")
			}
			return cmd.Help()
		}
	}
}

// unknownCommand refuses a command that does not exist, without quoting
// it; it may suggest the names of those that do.
func unknownCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	msg := "unknown command"
	if names := cmd.SuggestionsFor(args[0]); len(names) > 0 {
		msg += "; did you mean " + strings.Join(names, " or ") + "?"
	}
	return apperr.New(apperr.InvalidInput, msg+" (see "+cmd.CommandPath()+" --help)")
}

// quotingNothing is check with cobra's message replaced: cobra.NoArgs
// quotes the first argument. An *apperr.Error of the command's own passes.
func quotingNothing(check cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		err := check(cmd, args)
		var e *apperr.Error
		if err == nil || errors.As(err, &e) {
			return err
		}
		return apperr.New(apperr.InvalidInput, "wrong arguments; usage: "+cmd.UseLine())
	}
}

// flagError reports a flag that cannot be parsed without quoting the
// command line. pflag stops at that flag, so the error is marked
// unparsedFlags: whether --json was asked for is read from the arguments
// (jsonInArgs; codex-review-027 B3).
func flagError(cmd *cobra.Command, err error) error {
	problem := "an unknown flag, or a flag without its value"
	switch msg := err.Error(); {
	case strings.HasPrefix(msg, "unknown"):
		problem = "an unknown flag"
	case strings.HasPrefix(msg, "flag needs an argument"):
		problem = "a flag without its value"
	case strings.HasPrefix(msg, "invalid argument"):
		problem = "a flag with a value it does not take"
	}
	return unparsedFlags{apperr.New(apperr.InvalidInput, "invalid command line: "+problem+" (see "+cmd.CommandPath()+" --help)")}
}

// unparsedFlags is an error after which the command's flags were not all
// parsed (flagError).
type unparsedFlags struct{ err *apperr.Error }

func (u unparsedFlags) Error() string { return u.err.Error() }
func (u unparsedFlags) Unwrap() error { return u.err }

// completionRequest reports whether cobra would run its hidden shell
// completion command (__complete, or its alias __completeNoDesc) for args,
// and returns the arguments before that command: the root's own flags
// (codex-review-029 B1).
//
// cobra adds that command only while executing, after secureCommandLine
// set the tree up, and it writes cobra's raw errors, quoting the command
// line, to os.Stderr and to BASH_COMP_DEBUG_FILE. Specmetry provides no
// shell completion, so such a command line must be refused before cobra
// executes. cobra's initCompleteCmd keeps its command exactly when
// root.Find(args) resolves to a command of that name; otherwise it removes
// it again. completionRequest asks the same question in the same state:
// the help command added as ExecuteC adds it first, and a stand-in of the
// same name and alias at the root, then cobra's own Find, which reads the
// root's flags, values and "--" as cobra does. The stand-in never runs and
// is removed again. A word __complete elsewhere (a flag's value, after
// "--", an argument of another command) does not resolve to it.
func completionRequest(root *cobra.Command, args []string) ([]string, bool) {
	root.InitDefaultHelpCmd()
	probe := &cobra.Command{
		Use:     cobra.ShellCompRequestCmd,
		Aliases: []string{cobra.ShellCompNoDescRequestCmd},
		Hidden:  true,
	}
	root.AddCommand(probe)
	defer root.RemoveCommand(probe)
	found, rest, err := root.Find(args)
	if err != nil || found != probe {
		return nil, false
	}
	// Find drops the command's name from args: what is before it are the
	// root's flags. If the name follows itself, the prefix may hold one
	// more of it, which is no flag.
	pos := 0
	for pos < len(rest) && rest[pos] == args[pos] {
		pos++
	}
	return args[:pos], true
}

// errNoCompletion is the refusal of a completion request.
func errNoCompletion() *apperr.Error {
	return apperr.New(apperr.InvalidInput, "shell completion is not provided (see specmetry --help)")
}

// newHelpCmd is cobra's help command, refusing a topic it does not know:
// cobra's quotes it and exits 0.
func newHelpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long: `Help provides help for any command in the application.
Simply type specmetry help [path to command] for full details.`,
		RunE: func(c *cobra.Command, args []string) error {
			target, rest, err := c.Root().Find(args)
			if err != nil || target == nil || len(rest) > 0 {
				return apperr.New(apperr.InvalidInput, "unknown help topic (see specmetry --help)")
			}
			return target.Help()
		},
	}
}
