// Package decode holds the syntax layer of the C-9 limited evidence tool:
// the strict parser of R/go.sum (method record docs/wp2.5-c9-exception-
// identity-method.md §2.4 row I-1, §2.2 item 9) and the decoders of the
// outputs of the three go calls (method record §3.1 steps 1 to 4, §3.2,
// §3.3.2, §3.4), with the byte variants of the decoder control LC-4
// (docs/wp2.5-c9-limited-evidence-plan.md §5.2 row LC-4, §8.1).
//
// The package only decodes. It never applies the value requirements of
// §3.2, §3.3.3 or the branch rules of §3.4 (method record §3.1 step 5,
// stage B): an omitted key takes its Go zero value and is not an error
// here, and a decoded value that a later requirement rejects is still a
// successful decode. Every decode error makes the whole output of that
// call unusable (method record §3.1); the caller classifies it J-3, and
// every go.sum problem J-2 (method record §6.3, DS-5).
//
// The package has no state outside the values it returns, starts no
// process and reads no file: its only inputs are the byte slices it is
// given.
package decode

import (
	"fmt"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// Kind is the precise kind of a decode error. It refines J-3 for records;
// it is not a J class.
type Kind string

// The kinds. Each error has exactly one kind; which one is fixed by the
// decode order documented on DecodeEnv, DecodeContext and DecodePackages.
const (
	// KindCount: the number of objects or texts is wrong (zero-byte output,
	// a second object or text), or a list of the context text does not
	// have the number of element lines its header declares.
	KindCount Kind = "count"
	// KindForm: the output does not have the required overall shape: it
	// does not begin with a JSON object, the first call's object is not
	// followed by its LF, or the context text ends before its end line.
	KindForm Kind = "form"
	// KindSyntax: malformed JSON (including invalid UTF-8, a lone UTF-16
	// surrogate escape and nesting deeper than maxDepth).
	KindSyntax Kind = "syntax"
	// KindTrailing: content after the last object or after the end line.
	KindTrailing Kind = "trailing"
	// KindKey: a key outside the fixed definition, a key written with
	// escapes instead of the literal field name, or a key that is not one
	// of the requested names.
	KindKey Kind = "key"
	// KindCase: a key equal to a field name only when case is ignored.
	KindCase Kind = "case"
	// KindDuplicate: a key that appears twice in one object.
	KindDuplicate Kind = "duplicate"
	// KindNull: an explicit null anywhere.
	KindNull Kind = "null"
	// KindType: a JSON value whose type does not match the field's type
	// (string, bool, array of strings, map of string to string, object,
	// array of objects, or an RFC 3339 time string).
	KindType Kind = "type"
	// KindMissing: a requested name of the first call has no key.
	KindMissing Kind = "missing"
	// KindLine: a line of the context text that is not the expected line:
	// a CR, a blank line, a wrong name or order, or a wrong separator.
	KindLine Kind = "line"
	// KindValue: a value of the context text that is not of its form: <Q>
	// not a Go interpreted string literal, <B> not true or false, <N> not
	// 0 or [1-9][0-9]*.
	KindValue Kind = "value"
	// KindCanonical: a <Q> that is a valid literal but not the one
	// strconv.Quote gives for the string it denotes (for example "\x41").
	KindCanonical Kind = "canonical"
	// KindUsage: the decoder was called with arguments it cannot use (a
	// tool defect, not an observation of the output).
	KindUsage Kind = "usage"
)

// Error is a decode error. Call says which output; Kind is the precise
// kind; At names where: a JSON path ("$", "$.GOOS", "$[2].Module.Sum") for
// the first and third calls, "line N" (1-based) for the second call; Offset
// is the byte offset in the output where the problem was found. Detail is a
// deterministic explanation. The outcome of the LC-4 variants compares
// Call, Kind and At; Offset and Detail are diagnostics.
type Error struct {
	Call   spec.CallKind
	Kind   Kind
	At     string
	Offset int
	Detail string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s at %s (byte %d): %s", e.Call, e.Kind, e.At, e.Offset, e.Detail)
}

func errorf(call spec.CallKind, kind Kind, at string, off int, format string, args ...any) *Error {
	return &Error{Call: call, Kind: kind, At: at, Offset: off, Detail: fmt.Sprintf(format, args...)}
}
