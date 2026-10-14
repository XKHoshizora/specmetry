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
	ctx = channel.NewContext(ctx, channel.Detect(os.Stdin, os.Stdout))
	root.SetArgs(args)
	cmd, err := root.ExecuteContextC(ctx)
	if err == nil {
		return apperr.ExitOK
	}
	var e *apperr.Error
	if !errors.As(err, &e) {
		// Errors that are not *apperr.Error come from cobra's own argument
		// and flag parsing: unknown command, unknown flag, wrong arity.
		e = apperr.Wrap(apperr.InvalidInput, err, "invalid command line")
	}
	// With --json the error is the error.v1 document on stdout
	// (architecture §9.1).
	if wantsJSON(cmd) {
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
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newVersionCmd(), newDoctorCmd(), newInitCmd())
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

// printError writes a human-readable error. Commands run with --json print
// the error.v1 envelope to stdout instead (printErrorJSON).
func printError(w io.Writer, e *apperr.Error) {
	_, _ = fmt.Fprintf(w, "error: %s\n", e.Error())
	for _, d := range e.Details {
		if d.Pointer != "" {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", d.Pointer, d.Problem)
		} else {
			_, _ = fmt.Fprintf(w, "  %s\n", d.Problem)
		}
	}
}
