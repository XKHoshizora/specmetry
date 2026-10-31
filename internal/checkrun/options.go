package checkrun

import (
	"io"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// Options is a check run's command line once the CLI has parsed it (spec
// §3): the criteria given, in the order given, or --all; --isolation, off
// when it is left out; and --json.
type Options struct {
	Criteria  []string
	All       bool
	Isolation string
	JSON      bool
}

// Output is the raw stdout and stderr cli.Execute receives (spec §9.1,
// §19.9), never the termtext Writers it wraps them in: a check's output
// goes to them, and Specmetry's own text does not.
type Output struct {
	Stdout, Stderr io.Writer
}

// check is step 1 of spec §4.1 past the parsing: criteria or --all, not
// both and not neither, and an isolation mode of the MVP (D16: auto is not
// one, so it is a command line error). required passes here and is
// refused at step 2. Nothing a caller gave is quoted.
func (o Options) check() error {
	switch {
	case len(o.Criteria) > 0 && o.All:
		return apperr.New(apperr.InvalidInput, "give the acceptance criteria to check or --all, not both; no check was run")
	case len(o.Criteria) == 0 && !o.All:
		return apperr.New(apperr.InvalidInput, "give the acceptance criteria to check, or --all; no check was run")
	}
	switch o.Isolation {
	case "off", "required":
		return nil
	case "auto":
		return apperr.New(apperr.InvalidInput, "--isolation auto is not available: it needs the isolated executor, which the MVP does not have; no check was run").
			WithDetails(apperr.Detail{Pointer: "--isolation", Problem: "must be off or required"})
	}
	return apperr.New(apperr.InvalidInput, "invalid --isolation; no check was run").
		WithDetails(apperr.Detail{Pointer: "--isolation", Problem: "must be off or required"})
}

// isolationUnavailable is step 2's refusal of --isolation required (data
// model §7.4, MVP-EV-13, CX-28(a)): the MVP has no isolated executor.
func isolationUnavailable() error {
	return apperr.New(apperr.Environment,
		"--isolation required cannot be met: the MVP has no isolated executor; no check was run and none was recorded").
		WithDetails(apperr.Detail{Pointer: "--isolation", Problem: "isolation_unavailable: no isolated executor in the MVP"})
}
