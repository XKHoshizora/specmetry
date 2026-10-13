// Package cli wires the specmetry command line. Commands return
// *apperr.Error values; Execute maps them to exit codes and prints them
// (architecture.md §9.1).
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// Execute runs the CLI with args and returns the process exit code.
func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	root := newRoot(stdout, stderr)
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	if err == nil {
		return apperr.ExitOK
	}
	var e *apperr.Error
	if !errors.As(err, &e) {
		// Errors that are not *apperr.Error come from cobra's own argument
		// and flag parsing: unknown command, unknown flag, wrong arity.
		e = apperr.Wrap(apperr.InvalidInput, err, "invalid command line")
	}
	printError(stderr, e)
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
	root.AddCommand(newVersionCmd(), newDoctorCmd())
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

// printError writes a human-readable error. Commands with --json output
// (from M1 on) print the error.v1 envelope to stdout instead.
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
