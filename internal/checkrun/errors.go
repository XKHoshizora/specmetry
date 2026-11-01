package checkrun

import (
	"errors"
	"fmt"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// maxDetails is the most details an error.v1 document holds (its
// maxItems, schemas/error.v1.schema.json).
const maxDetails = 200

// refusal is a step's refusal with every problem it found (Review 123
// B-123-1): the first maxDetails as details, in their order, each at its
// own argument or criterion; when there are more, the message says how
// many problems there are and how many are left out. Every argument and
// criterion was still checked; the code, and so the exit code, is the
// step's.
func refusal(code apperr.Code, message string, problems []apperr.Detail) error {
	if n := len(problems); n > maxDetails {
		message = fmt.Sprintf("%s; %d problems: the first %d are listed, %d left out", message, n, maxDetails, n-maxDetails)
		problems = problems[:maxDetails]
	}
	return apperr.New(code, message).WithDetails(problems...)
}

// ReleaseError is a close that did not confirm its release (Review 123
// B-123-2): what was being closed, in Specmetry's words, and each close's
// own error. Whether the resource is still open is not known. Nothing
// closes it again: the process's exit releases it at the latest, and the
// caller reports it. Its text is controlled; the close errors are reached
// with errors.Is and errors.As.
type ReleaseError struct {
	Resources []string
	Errs      []error
}

func (e *ReleaseError) Error() string {
	return strings.Join(e.Resources, " and ") + " closed with an error: not confirmed released"
}

func (e *ReleaseError) Unwrap() []error { return e.Errs }

func (e *ReleaseError) add(resource string, err error) {
	e.Resources = append(e.Resources, resource)
	e.Errs = append(e.Errs, err)
}

// Test hook: testHookRelease adds an injected error to a close's own
// result, by the resource closed ("scratch", "database", "handle"); it
// never replaces that result.
var testHookRelease func(resource string) error

// released is a close's result: its own, with an injected error joined
// to it in tests.
func released(resource string, err error) error {
	if testHookRelease != nil {
		err = errors.Join(err, testHookRelease(resource))
	}
	return err
}

const noCheck = "no check was run"

// failed is a refused Prepare's error (D10, Review 123 B-123-2): the
// step's own error, copied with its code, exit code and details, its
// message saying that no check was run and, when closing what Prepare had
// opened did not confirm every release, a controlled note on it after
// any count of problems left out. The step's error and the close errors
// stay in the chain; apperr.From and ToEnvelope find the copy, note and
// all.
func failed(step, closeErr error) error {
	e := *apperr.From(step)
	if !strings.Contains(e.Message, noCheck) {
		e.Message += "; " + noCheck
	}
	errs := []error{&e, step}
	if closeErr != nil {
		e.Message += "; cleanup: " + closeErr.Error() + "; nothing closes it again, and the process's exit releases it"
		errs = append(errs, closeErr)
	}
	return &prepareError{env: &e, errs: errs}
}

// prepareError is failed's error: env, which the envelope is made from,
// first in its chain.
type prepareError struct {
	env  *apperr.Error
	errs []error
}

func (e *prepareError) Error() string   { return e.env.Error() }
func (e *prepareError) Unwrap() []error { return e.errs }
