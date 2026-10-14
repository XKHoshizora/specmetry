package cli

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/contract"
)

// validator compiles the exchange schemas once per process.
var validator = sync.OnceValues(contract.New)

// writeJSON validates v against the named schema and prints it. Output
// meant for an Agent must conform to schemas/ (AGENTS.md); a document that
// does not is an internal error and is not printed.
func writeJSON(w io.Writer, schema string, v any) error {
	val, err := validator()
	if err != nil {
		return apperr.Wrap(apperr.Internal, err, "cannot load the exchange schemas")
	}
	if err := val.ValidateValue(schema, v); err != nil {
		e := apperr.From(err)
		return apperr.Newf(apperr.Internal, "the %s output does not conform to its schema: %s", schema, e.Message).WithDetails(e.Details...)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, err, "cannot encode the output")
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return apperr.Wrap(apperr.Environment, err, "cannot write the output")
	}
	return nil
}

// wantsJSON reports whether the command was asked for --json output.
func wantsJSON(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	f := cmd.Flags().Lookup("json")
	return f != nil && f.Value.String() == "true"
}

// printErrorJSON prints the error.v1 envelope. If even that cannot be
// done, the error goes to stderr as text.
func printErrorJSON(stdout, stderr io.Writer, e *apperr.Error) {
	if err := writeJSON(stdout, "error.v1", apperr.ToEnvelope(e)); err != nil {
		printError(stderr, e)
	}
}
