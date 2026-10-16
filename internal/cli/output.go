package cli

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
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

// jsonInArgs reports whether args ask cmd for --json output, for a command
// whose flags pflag stopped parsing (unparsedFlags; codex-review-027 B3).
// The arguments are read as pflag reads them, against cmd's flags:
//   - --json and --json=<bool> count, the last one wins; a --json=<value>
//     that is not a bool leaves the result as it was;
//   - the value of a flag that takes one (-m x, -mx, -m=x, --message x,
//     --file x) is not a flag;
//   - an unknown flag takes no value: pflag stops at it, and so it does
//     not take the argument after it;
//   - nothing after "--" is a flag.
//
// A command without a json flag never asks for it.
func jsonInArgs(cmd *cobra.Command, args []string) bool {
	if cmd == nil || cmd.Flags().Lookup("json") == nil {
		return false
	}
	flags := cmd.Flags()
	asJSON := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			return asJSON
		case strings.HasPrefix(a, "--"):
			name, value, hasValue := strings.Cut(a[2:], "=")
			f := flags.Lookup(name)
			switch {
			case f == nil:
			case name == "json" && !hasValue:
				asJSON = true
			case name == "json":
				if v, err := strconv.ParseBool(value); err == nil {
					asJSON = v
				}
			case f.NoOptDefVal == "" && !hasValue:
				i++ // its value
			}
		case strings.HasPrefix(a, "-") && len(a) > 1:
			// Shorthands, possibly several: -h, -m x, -mx, -m=x.
			for j := 1; j < len(a); j++ {
				f := flags.ShorthandLookup(a[j : j+1])
				if f == nil {
					break
				}
				if f.NoOptDefVal == "" {
					if j == len(a)-1 {
						i++ // its value is the next argument
					}
					break
				}
			}
		}
	}
	return asJSON
}

// printErrorJSON prints the error.v1 envelope. If even that cannot be
// done, the error goes to stderr as text.
func printErrorJSON(stdout, stderr io.Writer, e *apperr.Error) {
	if err := writeJSON(stdout, "error.v1", apperr.ToEnvelope(e)); err != nil {
		printError(stderr, e)
	}
}
