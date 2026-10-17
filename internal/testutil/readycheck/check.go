package readycheck

import "errors"

var errNotImplemented = errors.New("not implemented")

// Check evaluates the ready predicate at a cutoff.
func Check(Facts, Current, int64) Verdict {
	return Verdict{Failed: map[Condition][]string{"stub": nil}}
}

// Consistent is R10: an output's ready verdict and delivery evidence grade
// agree with the checker's.
func Consistent(Verdict, bool, string) error { return errNotImplemented }

// ConfigHash is a check configuration's hash (data-model §4.4).
func ConfigHash(CheckConfig) string { return "" }
