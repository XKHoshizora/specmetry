// Package contract validates documents against the embedded exchange schemas
// (schemas/*.schema.json). Validation failures are reported as invalid_input
// errors whose details carry JSON Pointers (data-model.md §8.3).
package contract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/schemas"
)

// Validator holds the compiled schemas.
type Validator struct {
	schemas map[string]*jsonschema.Schema
}

// New compiles every embedded schema. Each schema is itself checked against
// the JSON Schema 2020-12 meta-schema. References resolve only among the
// embedded files; nothing is fetched from the network.
func New() (*Validator, error) {
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	// The schemas use ECMA-262 regular expressions (lookaheads), which Go's
	// RE2-based regexp does not support.
	c.UseRegexpEngine(ecmaRegexp)

	names, err := fs.Glob(schemas.FS, "*.schema.json")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	for _, file := range names {
		b, err := schemas.FS.ReadFile(file)
		if err != nil {
			return nil, err
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if id, _ := doc.(map[string]any)["$id"].(string); id != schemas.BaseURL+file {
			return nil, fmt.Errorf("%s: $id %q does not match its file name", file, id)
		}
		if err := c.AddResource(schemas.BaseURL+file, doc); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
	}
	v := &Validator{schemas: map[string]*jsonschema.Schema{}}
	for _, file := range names {
		sch, err := c.Compile(schemas.BaseURL + file)
		if err != nil {
			return nil, fmt.Errorf("compile %s: %w", file, err)
		}
		v.schemas[strings.TrimSuffix(file, ".schema.json")] = sch
	}
	return v, nil
}

// Names lists the schema names, e.g. "proposal.v1".
func (v *Validator) Names() []string {
	out := make([]string, 0, len(v.schemas))
	for n := range v.schemas {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// ValidateJSON validates raw JSON against the named schema ("proposal.v1").
func (v *Validator) ValidateJSON(name string, data []byte) error {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return apperr.Wrap(apperr.InvalidInput, err, "input is not valid JSON")
	}
	return v.validate(name, doc)
}

// ValidateValue marshals value and validates it (used for outputs).
func (v *Validator) ValidateValue(name string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return apperr.Wrap(apperr.Internal, err, "cannot encode document")
	}
	return v.ValidateJSON(name, b)
}

func (v *Validator) validate(name string, doc any) error {
	sch, ok := v.schemas[name]
	if !ok {
		return apperr.Newf(apperr.Internal, "unknown schema %q", name)
	}
	err := sch.Validate(doc)
	if err == nil {
		return nil
	}
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return apperr.Wrap(apperr.Internal, err, "schema validation failed to run")
	}
	return apperr.Newf(apperr.InvalidInput, "document does not conform to %s", name).
		WithDetails(leafErrors(ve)...)
}

var printer = message.NewPrinter(language.English)

// leafErrors flattens the validation error tree into pointer/problem pairs.
func leafErrors(ve *jsonschema.ValidationError) []apperr.Detail {
	if len(ve.Causes) == 0 {
		return []apperr.Detail{{
			Pointer: "/" + strings.Join(escapeTokens(ve.InstanceLocation), "/"),
			Problem: ve.ErrorKind.LocalizedString(printer),
		}}
	}
	var out []apperr.Detail
	for _, c := range ve.Causes {
		out = append(out, leafErrors(c)...)
	}
	return out
}

func escapeTokens(tokens []string) []string {
	out := make([]string, len(tokens))
	for i, t := range tokens {
		out[i] = strings.NewReplacer("~", "~0", "/", "~1").Replace(t)
	}
	return out
}

type ecmaRE struct{ re *regexp2.Regexp }

func (r ecmaRE) MatchString(s string) bool {
	ok, err := r.re.MatchString(s)
	return err == nil && ok
}

func (r ecmaRE) String() string { return r.re.String() }

func ecmaRegexp(s string) (jsonschema.Regexp, error) {
	re, err := regexp2.Compile(s, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	return ecmaRE{re: re}, nil
}
