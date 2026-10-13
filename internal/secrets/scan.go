package secrets

import (
	"slices"
	"strconv"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// Field is a value to scan and where it came from: a JSON Pointer into a
// submitted document, or a flag such as "--name".
type Field struct {
	Pointer string
	Value   string
}

// Finding is a field whose value has a hit. It never holds the matched
// text.
type Finding struct {
	Pointer string
	RuleID  string
}

// Fields scans each field, in order.
func (s *Scanner) Fields(fields ...Field) []Finding {
	var out []Finding
	for _, f := range fields {
		if id, ok := s.Match(f.Value); ok {
			out = append(out, Finding{Pointer: f.Pointer, RuleID: id})
		}
	}
	return out
}

// JSON scans every string value of a document decoded by encoding/json
// (objects as map[string]any, arrays as []any). Object keys are visited in
// sorted order, so the findings are deterministic.
func (s *Scanner) JSON(doc any) []Finding {
	var out []Finding
	s.walk(doc, "", &out)
	return out
}

func (s *Scanner) walk(v any, pointer string, out *[]Finding) {
	switch v := v.(type) {
	case string:
		if id, ok := s.Match(v); ok {
			*out = append(*out, Finding{Pointer: pointer, RuleID: id})
		}
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			s.walk(v[k], pointer+"/"+escapePointer(k), out)
		}
	case []any:
		for i, e := range v {
			s.walk(e, pointer+"/"+strconv.Itoa(i), out)
		}
	}
}

// escapePointer escapes a JSON Pointer reference token (RFC 6901).
func escapePointer(k string) string {
	return strings.ReplaceAll(strings.ReplaceAll(k, "~", "~0"), "/", "~1")
}

// Reject turns findings into an invalid_input error (exit code 2), or nil
// when there are none. The error names each field and rule only; the
// matched text is never part of it.
func Reject(findings []Finding) error {
	if len(findings) == 0 {
		return nil
	}
	details := make([]apperr.Detail, len(findings))
	for i, f := range findings {
		details[i] = apperr.Detail{Pointer: f.Pointer, Problem: "secret_detected: " + f.RuleID}
	}
	return apperr.New(apperr.InvalidInput,
		"the input contains text that looks like a secret; remove it, or reference an environment variable by name instead (for example $API_TOKEN)").
		WithDetails(details...)
}
