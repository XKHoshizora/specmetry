package contract

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// Validation errors describe what is wrong without echoing what was
// submitted (architecture §10.8): a submitted value or key may hold a
// secret. The validation library's own messages quote the offending value
// for some keywords (pattern, format, the numeric bounds) and list
// unexpected property names, so they are never passed on. Each problem is
// written here from the schema side of the error only — the keyword and
// what the schema expects — and an unknown kind gets a generic message.
// JSON Pointers keep only array indexes and property names the schemas
// declare; a pointer that would pass through any other key (a map key the
// submitter chose) stops at that key's parent.

// propertyNames collects every property name the schemas declare.
func propertyNames(doc any, into map[string]bool) {
	switch d := doc.(type) {
	case map[string]any:
		if props, ok := d["properties"].(map[string]any); ok {
			for name := range props {
				into[name] = true
			}
		}
		for _, v := range d {
			propertyNames(v, into)
		}
	case []any:
		for _, v := range d {
			propertyNames(v, into)
		}
	}
}

// details flattens the validation error tree into pointer/problem pairs,
// without repeating one.
func (v *Validator) details(ve *jsonschema.ValidationError) []apperr.Detail {
	var out []apperr.Detail
	seen := map[apperr.Detail]bool{}
	var walk func(*jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) > 0 {
			for _, c := range e.Causes {
				walk(c)
			}
			return
		}
		d := apperr.Detail{Pointer: v.pointer(e.InstanceLocation), Problem: describe(e.ErrorKind)}
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	walk(ve)
	return out
}

// pointer is the JSON Pointer of a location, cut short before the first
// token that is neither an array index nor a declared property name. The
// root is "/".
func (v *Validator) pointer(location []string) string {
	var tokens []string
	for _, t := range location {
		if _, err := strconv.Atoi(t); err != nil && !v.names[t] {
			break
		}
		tokens = append(tokens, strings.NewReplacer("~", "~0", "/", "~1").Replace(t))
	}
	return "/" + strings.Join(tokens, "/")
}

// describe states the problem of one error kind from the schema's side.
func describe(k jsonschema.ErrorKind) string {
	switch k := k.(type) {
	case *kind.Type:
		return "must be of type " + strings.Join(k.Want, " or ") + ", not " + k.Got
	case *kind.Enum:
		return "must be one of " + constants(k.Want...)
	case *kind.Const:
		return "must be " + constants(k.Want)
	case *kind.Pattern:
		return "does not match pattern " + k.Want
	case *kind.Format:
		return "is not a valid " + k.Want
	case *kind.MinLength:
		return fmt.Sprintf("must be at least %d characters long", k.Want)
	case *kind.MaxLength:
		return fmt.Sprintf("must be at most %d characters long", k.Want)
	case *kind.MinItems:
		return fmt.Sprintf("must have at least %d items", k.Want)
	case *kind.MaxItems:
		return fmt.Sprintf("must have at most %d items", k.Want)
	case *kind.MinProperties:
		return fmt.Sprintf("must have at least %d properties", k.Want)
	case *kind.MaxProperties:
		return fmt.Sprintf("must have at most %d properties", k.Want)
	case *kind.Required:
		return "is missing required properties: " + strings.Join(k.Missing, ", ")
	case *kind.Dependency:
		return "is missing properties another property requires: " + strings.Join(k.Missing, ", ")
	case *kind.DependentRequired:
		return "is missing properties another property requires: " + strings.Join(k.Missing, ", ")
	case *kind.AdditionalProperties:
		if len(k.Properties) == 1 {
			return "has a property the schema does not allow"
		}
		return fmt.Sprintf("has %d properties the schema does not allow", len(k.Properties))
	case *kind.PropertyNames:
		return "has a property name the schema does not allow"
	case *kind.AdditionalItems:
		return "has more items than the schema allows"
	case *kind.UniqueItems:
		return fmt.Sprintf("items %d and %d are equal; items must be unique", k.Duplicates[0], k.Duplicates[1])
	case *kind.Minimum:
		return "must be at least " + number(k.Want)
	case *kind.Maximum:
		return "must be at most " + number(k.Want)
	case *kind.ExclusiveMinimum:
		return "must be greater than " + number(k.Want)
	case *kind.ExclusiveMaximum:
		return "must be less than " + number(k.Want)
	case *kind.MultipleOf:
		return "must be a multiple of " + number(k.Want)
	case *kind.OneOf:
		return "must match exactly one of the allowed forms"
	case *kind.AnyOf:
		return "does not match any of the allowed forms"
	case *kind.Not:
		return "matches a form the schema forbids"
	case *kind.FalseSchema:
		return "is not allowed here"
	case *kind.Contains, *kind.MinContains, *kind.MaxContains:
		return "does not contain the items the schema requires"
	case *kind.InvalidJsonValue:
		return "is not a JSON value"
	}
	keyword := "schema"
	if path := k.KeywordPath(); len(path) > 0 {
		keyword = path[len(path)-1]
	}
	return "does not satisfy the " + keyword + " constraint"
}

// constants writes schema constants (from enum and const) when they are
// primitive; they come from the schema, never from the input.
func constants(values ...any) string {
	var out []string
	for _, v := range values {
		switch v.(type) {
		case map[string]any, []any:
			return "a value the schema allows"
		}
		b, err := json.Marshal(v)
		if err != nil {
			return "a value the schema allows"
		}
		out = append(out, string(b))
	}
	return strings.Join(out, ", ")
}

func number(r *big.Rat) string {
	if r == nil {
		return "the schema's bound"
	}
	return r.RatString()
}
