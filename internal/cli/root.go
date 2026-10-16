// Package cli wires the specmetry command line. Commands return
// *apperr.Error values; Execute maps them to exit codes and prints them
// (architecture.md §9.1).
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/termtext"
)

// Execute runs the CLI with args and returns the process exit code.
func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return executeRoot(ctx, newRoot(stdout, stderr), args, stderr)
}

// executeRoot runs root with args. The channel (architecture §5.1) is
// detected once per command, from the process's own stdin and stdout, and
// handed to commands in the context; human-only commands pass it to
// channel.Guard before writing anything.
func executeRoot(ctx context.Context, root *cobra.Command, args []string, stderr io.Writer) int {
	d := channel.Detect(os.Stdin, os.Stdout)
	d.StderrTerminal = channel.IsTerminal(os.Stderr)
	return executeAs(ctx, d, root, args, stderr)
}

// executeAs is executeRoot with the channel given.
func executeAs(ctx context.Context, d channel.Detection, root *cobra.Command, args []string, stderr io.Writer) int {
	ctx = channel.NewContext(ctx, d)
	if args == nil {
		args = []string{} // cobra would read os.Args instead
	}
	// cobra's hidden shell completion command would bypass every check
	// above and write the command line to os.Stderr: it is refused before
	// cobra executes (completionRequest; codex-review-029 B1).
	if flags, ok := completionRequest(root, args); ok {
		e := errNoCompletion()
		if jsonInArgs(root, flags) {
			printErrorJSON(root.OutOrStdout(), stderr, e)
		} else {
			printError(stderr, e)
		}
		return apperr.ExitCode(e)
	}
	root.SetArgs(args)
	cmd, err := root.ExecuteContextC(ctx)
	if err == nil {
		return apperr.ExitOK
	}
	var e *apperr.Error
	if !errors.As(err, &e) {
		// An error that is not an *apperr.Error is cobra's own about the
		// command line, one secureCommandLine does not replace. Its text
		// quotes what was typed, so it is dropped, not kept as the cause
		// (codex-review-027 N1).
		e = apperr.New(apperr.InvalidInput, "invalid command line (see specmetry --help)")
	}
	// With --json the error is the error.v1 document on stdout
	// (architecture §9.1). After a flag error, the flags after it were not
	// parsed: whether --json was asked for is read from the arguments
	// (codex-review-027 B3).
	asJSON := wantsJSON(cmd)
	var unparsed unparsedFlags
	if errors.As(err, &unparsed) {
		asJSON = jsonInArgs(cmd, args)
	}
	if asJSON {
		printErrorJSON(root.OutOrStdout(), stderr, e)
	} else {
		printError(stderr, e)
	}
	return apperr.ExitCode(e)
}

func newRoot(stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:   "specmetry",
		Short: "Local-first progress and delivery oversight for AI coding agent projects",
		// Usage is printed only for --help, and errors only by Execute.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// Every text a command writes reaches the terminal escaped (termtext):
	// it may hold a proposal's, a person's or the configuration's text.
	// JSON documents keep their own encoding (writeJSON).
	root.SetOut(termtext.NewWriter(stdout))
	root.SetErr(termtext.NewWriter(stderr))
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newVersionCmd(), newDoctorCmd(), newInitCmd(), newSourceCmd(), newProposeCmd(), newProposalCmd(),
		newReviewCmd(), newBaselineCmd(), newQuestionCmd(), newTargetCmd(), newScopeCmd())
	secureCommandLine(root)
	return root
}

// run adapts a command body so that every error it returns is an
// *apperr.Error; anything else is reported as an internal error.
func run(fn func(cmd *cobra.Command, args []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := fn(cmd, args); err != nil {
			return apperr.From(err)
		}
		return nil
	}
}

// printError writes a human-readable error, escaped for the terminal: it may
// quote an argument or a stored text. Commands run with --json print the
// error.v1 envelope to stdout instead (printErrorJSON).
func printError(w io.Writer, e *apperr.Error) {
	w = termtext.NewWriter(w)
	_, _ = fmt.Fprintf(w, "error: %s\n", e.Error())
	for _, d := range e.Details {
		if d.Pointer != "" {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", d.Pointer, d.Problem)
		} else {
			_, _ = fmt.Fprintf(w, "  %s\n", d.Problem)
		}
	}
}
