// Package apperr defines Specmetry's error codes, their CLI exit codes and
// HTTP statuses, and the error.v1 envelope (schemas/error.v1.schema.json).
package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"unicode/utf8"
)

// Code is an error.v1 error code.
type Code string

// Error codes defined by schemas/error.v1.schema.json.
const (
	InvalidInput      Code = "invalid_input"
	Unauthorized      Code = "unauthorized"
	Forbidden         Code = "forbidden"
	HumanRequired     Code = "human_required"
	NotInitialized    Code = "not_initialized"
	StalePrecondition Code = "stale_precondition"
	Environment       Code = "environment"
	NotFound          Code = "not_found"
	Internal          Code = "internal"
)

// CLI exit codes (docs/architecture.md §9.1).
const (
	ExitOK                = 0
	ExitInternal          = 1
	ExitInvalidInput      = 2
	ExitHumanRequired     = 3
	ExitNotInitialized    = 4
	ExitStalePrecondition = 5
	ExitEnvironment       = 6
)

// SchemaID is the value of the envelope's "schema" field.
const SchemaID = "specmetry.error/v1"

const maxMessageLen = 2000

// Detail points at one problem in a submitted document.
type Detail struct {
	Pointer string `json:"pointer,omitempty"`
	Problem string `json:"problem"`
}

// Error is the single error type returned across package boundaries.
type Error struct {
	Code    Code
	Message string
	Details []Detail
	// Current carries the values a caller must re-read after a
	// stale_precondition error.
	Current map[string]any
	cause   error
}

// New returns an error with the given code and message.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Newf is New with formatting.
func Newf(code Code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap returns an error with the given code that wraps cause.
func Wrap(code Code, cause error, message string) *Error {
	return &Error{Code: code, Message: message, cause: cause}
}

// WithDetails appends details and returns e.
func (e *Error) WithDetails(details ...Detail) *Error {
	e.Details = append(e.Details, details...)
	return e
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.cause }

// From returns the *Error in err's chain. Any other error is reported as an
// internal error that wraps it.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return &Error{Code: Internal, Message: "internal error", cause: err}
}

// ExitCode maps err to a CLI exit code. A nil error maps to ExitOK.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	switch From(err).Code {
	case InvalidInput, NotFound:
		return ExitInvalidInput
	case HumanRequired:
		return ExitHumanRequired
	case NotInitialized:
		return ExitNotInitialized
	case StalePrecondition:
		return ExitStalePrecondition
	case Environment:
		return ExitEnvironment
	default:
		return ExitInternal
	}
}

// HTTPStatus maps an error code to an HTTP status (docs/architecture.md §9.2).
func HTTPStatus(code Code) int {
	switch code {
	case InvalidInput:
		return http.StatusBadRequest
	case Unauthorized:
		return http.StatusUnauthorized
	case Forbidden, HumanRequired:
		return http.StatusForbidden
	case NotFound, NotInitialized:
		return http.StatusNotFound
	case StalePrecondition:
		return http.StatusConflict
	case Environment:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Envelope is the error.v1 document.
type Envelope struct {
	Schema string `json:"schema"`
	Error  Body   `json:"error"`
}

// Body is the "error" object of an Envelope.
type Body struct {
	Code    Code           `json:"code"`
	Message string         `json:"message"`
	Details []Detail       `json:"details,omitempty"`
	Current map[string]any `json:"current,omitempty"`
}

// ToEnvelope converts err into an error.v1 envelope.
func ToEnvelope(err error) Envelope {
	e := From(err)
	return Envelope{
		Schema: SchemaID,
		Error: Body{
			Code:    e.Code,
			Message: truncate(e.Error(), maxMessageLen),
			Details: e.Details,
			Current: e.Current,
		},
	}
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}
