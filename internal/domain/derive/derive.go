package derive

import (
	"errors"
	"strings"
)

// InputError is a derivation refused because its input is incomplete or
// invalid: a missing current-value input, a missing observation, facts that
// do not reach the cutoff. The derivation never fills a missing input in
// (data-model §6.13.1).
type InputError struct {
	Problems []string
}

func (e *InputError) Error() string {
	return "derivation input: " + strings.Join(e.Problems, "; ")
}

// errNotImplemented marks the test-first stubs.
var errNotImplemented = errors.New("not implemented")

// Derive is the full derivation at in.AsOfSeq (data-model §6).
func Derive(Input) (*Result, error) {
	return nil, errNotImplemented
}

// ChangesSince is the --diff-since projection (data-model §6.13.1).
func ChangesSince(Input, int64) ([]Change, error) {
	return nil, errNotImplemented
}
