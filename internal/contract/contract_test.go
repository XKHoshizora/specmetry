package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

const examplesDir = "../../schemas/examples"

// Example file name prefix → schema name.
var prefixes = map[string]string{
	"proposal":        "proposal.v1",
	"claims":          "claims.v1",
	"review":          "review.v1",
	"decision":        "decision.v1",
	"scope":           "scope.v1",
	"source-ack":      "source-ack.v1",
	"error":           "error.v1",
	"spool":           "spool-event.v1",
	"status":          "status.v1",
	"init":            "init.v1",
	"check-run":       "check-run.v1",
	"gaps":            "gaps.v1",
	"question-answer": "question-answer.v1",
}

func validator(t *testing.T) *Validator {
	t.Helper()
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// WP0.4: every schema compiles against the 2020-12 meta-schema.
func TestAllSchemasCompile(t *testing.T) {
	v := validator(t)
	want := []string{
		"check-run.v1", "claims.v1", "common.v1", "decision.v1", "error.v1", "gaps.v1", "init.v1",
		"proposal.v1", "question-answer.v1", "review.v1", "scope.v1", "source-ack.v1", "spool-event.v1", "status.v1",
	}
	if got := strings.Join(v.Names(), ","); got != strings.Join(want, ",") {
		t.Fatalf("compiled schemas = %s", got)
	}
}

func schemaFor(t *testing.T, file string) string {
	t.Helper()
	for prefix, name := range prefixes {
		if strings.HasPrefix(file, prefix+".") {
			return name
		}
	}
	t.Fatalf("no schema mapping for example %s", file)
	return ""
}

// WP0.4: every example in schemas/examples/valid passes.
func TestValidExamplesPass(t *testing.T) {
	v := validator(t)
	files, err := filepath.Glob(filepath.Join(examplesDir, "valid", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no valid examples found: %v", err)
	}
	covered := map[string]bool{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		name := schemaFor(t, filepath.Base(f))
		covered[name] = true
		if err := v.ValidateJSON(name, b); err != nil {
			t.Errorf("%s should be valid against %s: %v %+v", filepath.Base(f), name, err, apperr.From(err).Details)
		}
	}
	for _, name := range v.Names() {
		if name != "common.v1" && !covered[name] {
			t.Errorf("schema %s has no valid example", name)
		}
	}
	t.Logf("%d valid examples checked", len(files))
}

type invalidCase struct {
	ID     string    `json:"id"`
	Schema string    `json:"schema"`
	Base   string    `json:"base"`
	Patch  []patchOp `json:"patch"`
	Why    string    `json:"why"`
}

type patchOp struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value"`
}

// WP0.4: every case in invalid-cases.json (an RFC 6902 patch applied to a
// valid example) must fail validation.
func TestInvalidCasesFail(t *testing.T) {
	v := validator(t)
	raw, err := os.ReadFile(filepath.Join(examplesDir, "invalid-cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Cases []invalidCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Cases) == 0 {
		t.Fatal("no invalid cases")
	}
	seen := map[string]bool{}
	for _, c := range doc.Cases {
		if seen[c.ID] {
			t.Fatalf("duplicate case id %s", c.ID)
		}
		seen[c.ID] = true
		base, err := os.ReadFile(filepath.Join(examplesDir, filepath.FromSlash(c.Base)))
		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		var target any
		if err := json.Unmarshal(base, &target); err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		// The unpatched base must be valid, so the patch is what breaks it.
		if err := v.ValidateJSON(c.Schema, base); err != nil {
			t.Fatalf("%s: base %s is not valid: %v", c.ID, c.Base, err)
		}
		for _, op := range c.Patch {
			if target, err = applyPatch(target, op); err != nil {
				t.Fatalf("%s: patch %+v: %v", c.ID, op, err)
			}
		}
		patched, _ := json.Marshal(target)
		err = v.ValidateJSON(c.Schema, patched)
		if err == nil {
			t.Errorf("%s accepted but must be rejected: %s", c.ID, c.Why)
			continue
		}
		if e := apperr.From(err); e.Code != apperr.InvalidInput || len(e.Details) == 0 {
			t.Errorf("%s: error %+v lacks invalid_input details", c.ID, e)
		}
	}
	t.Logf("%d invalid cases checked", len(doc.Cases))
}

// MVP-ST-10 foundation: the error envelope the CLI prints validates against
// error.v1, including details and current values.
func TestErrorEnvelopeConformsToErrorV1(t *testing.T) {
	v := validator(t)
	errs := []error{
		apperr.New(apperr.InvalidInput, "bad").WithDetails(apperr.Detail{Pointer: "/items/0", Problem: "missing title"}),
		&apperr.Error{Code: apperr.StalePrecondition, Message: "stale", Current: map[string]any{"base_version": 4}},
		fmt.Errorf("plain failure"),
		apperr.New(apperr.Environment, strings.Repeat("x", 5000)),
	}
	for _, err := range errs {
		if verr := v.ValidateValue("error.v1", apperr.ToEnvelope(err)); verr != nil {
			t.Errorf("envelope for %v does not conform: %v %+v", err, verr, apperr.From(verr).Details)
		}
	}
}

func TestValidationReportsJSONPointers(t *testing.T) {
	v := validator(t)
	err := v.ValidateJSON("source-ack.v1", []byte(`{"schema":"specmetry.source-ack/v1","source":"a.md","observed_sha256":"xyz"}`))
	e := apperr.From(err)
	if e == nil || e.Code != apperr.InvalidInput {
		t.Fatalf("err = %v", err)
	}
	var pointers []string
	for _, d := range e.Details {
		pointers = append(pointers, d.Pointer)
	}
	joined := strings.Join(pointers, " ")
	if !strings.Contains(joined, "/observed_sha256") || !strings.Contains(joined, "/") {
		t.Fatalf("pointers = %v", pointers)
	}
	if err := v.ValidateJSON("source-ack.v1", []byte(`{not json`)); apperr.ExitCode(err) != apperr.ExitInvalidInput {
		t.Fatalf("malformed JSON: %v", err)
	}
}

// applyPatch implements the RFC 6902 add/remove/replace subset used by
// invalid-cases.json.
func applyPatch(doc any, op patchOp) (any, error) {
	if !strings.HasPrefix(op.Path, "/") {
		return nil, fmt.Errorf("bad path %q", op.Path)
	}
	tokens := strings.Split(op.Path[1:], "/")
	for i, tk := range tokens {
		tokens[i] = strings.NewReplacer("~1", "/", "~0", "~").Replace(tk)
	}
	var value any
	if op.Op != "remove" {
		if err := json.Unmarshal(op.Value, &value); err != nil {
			return nil, err
		}
	}
	parent := doc
	for _, tk := range tokens[:len(tokens)-1] {
		switch p := parent.(type) {
		case map[string]any:
			parent = p[tk]
		case []any:
			i, err := strconv.Atoi(tk)
			if err != nil || i < 0 || i >= len(p) {
				return nil, fmt.Errorf("bad index %q", tk)
			}
			parent = p[i]
		default:
			return nil, fmt.Errorf("cannot descend into %T", parent)
		}
	}
	last := tokens[len(tokens)-1]
	switch p := parent.(type) {
	case map[string]any:
		_, exists := p[last]
		switch op.Op {
		case "remove":
			if !exists {
				return nil, fmt.Errorf("remove: %s missing", op.Path)
			}
			delete(p, last)
		case "replace":
			if !exists {
				return nil, fmt.Errorf("replace: %s missing", op.Path)
			}
			p[last] = value
		case "add":
			p[last] = value
		default:
			return nil, fmt.Errorf("unsupported op %q", op.Op)
		}
		return doc, nil
	case []any:
		// Arrays are replaced in their parent, so mutate through a copy.
		i, err := strconv.Atoi(last)
		if err != nil || i < 0 || i >= len(p) {
			return nil, fmt.Errorf("bad index %q", last)
		}
		switch op.Op {
		case "replace":
			p[i] = value
			return doc, nil
		case "remove":
			copy(p[i:], p[i+1:])
			p[len(p)-1] = nil
			return setParentSlice(doc, tokens[:len(tokens)-1], p[:len(p)-1])
		}
		return nil, fmt.Errorf("unsupported array op %q", op.Op)
	}
	return nil, fmt.Errorf("cannot patch into %T", parent)
}

func setParentSlice(doc any, path []string, s []any) (any, error) {
	if len(path) == 0 {
		return s, nil
	}
	parent := doc
	for _, tk := range path[:len(path)-1] {
		switch p := parent.(type) {
		case map[string]any:
			parent = p[tk]
		case []any:
			i, _ := strconv.Atoi(tk)
			parent = p[i]
		}
	}
	last := path[len(path)-1]
	switch p := parent.(type) {
	case map[string]any:
		p[last] = s
	case []any:
		i, _ := strconv.Atoi(last)
		p[i] = s
	}
	return doc, nil
}
