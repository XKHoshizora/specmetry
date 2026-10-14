package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// secretLike is a synthetic GitHub token, built at run time, that the
// secret rules (TD-21) recognise.
var secretLike = "ghp_" + strings.Repeat("A1b2", 9)

// patched returns the valid example base with ops applied, as JSON.
func patched(t *testing.T, base string, ops ...patchOp) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(examplesDir, "valid", base))
	if err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, op := range ops {
		if doc, err = applyPatch(doc, op); err != nil {
			t.Fatal(err)
		}
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func value(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// requireRedacted checks that nothing of the error a document produced —
// message, details, the error.v1 envelope — holds any of the needles.
func requireRedacted(t *testing.T, name string, err error, needles ...string) apperr.Envelope {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: the document was accepted", name)
	}
	if apperr.ExitCode(err) != apperr.ExitInvalidInput {
		t.Fatalf("%s: exit code %d for %v", name, apperr.ExitCode(err), err)
	}
	env := apperr.ToEnvelope(err)
	b, merr := json.Marshal(env)
	if merr != nil {
		t.Fatal(merr)
	}
	text := err.Error() + string(b)
	for _, d := range apperr.From(err).Details {
		text += d.Pointer + d.Problem
	}
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			t.Errorf("%s: the error echoes %q:\n%s", name, needle, b)
		}
	}
	return env
}

// Schema validation errors never echo the submitted values or unexpected
// keys: they name the field (JSON Pointer) and the constraint only
// (WP1.4 pre-review finding; architecture §10.8).
func TestValidationErrorsDoNotEchoInput(t *testing.T) {
	v := validator(t)
	canary := "SPECMETRY-CANARY-" + strings.Repeat("Zz9", 4)
	cases := []struct {
		name, schema string
		doc          []byte
		pointer      string
		needles      []string
	}{
		{"pattern", "source-ack.v1",
			patched(t, "source-ack.json", patchOp{Op: "replace", Path: "/observed_sha256", Value: value(t, secretLike)}),
			"/observed_sha256", []string{secretLike}},
		{"const", "source-ack.v1",
			patched(t, "source-ack.json", patchOp{Op: "replace", Path: "/schema", Value: value(t, canary)}),
			"/schema", []string{canary}},
		{"enum", "source-list.v1",
			patched(t, "source-list.with-refs.json", patchOp{Op: "replace", Path: "/excerpt_storage", Value: value(t, canary)}),
			"/excerpt_storage", []string{canary}},
		{"unexpected property named like a secret", "source-ack.v1",
			patched(t, "source-ack.json", patchOp{Op: "add", Path: "/" + secretLike, Value: value(t, canary)}),
			"/", []string{secretLike, canary}},
		{"maxLength", "source-ack.v1",
			patched(t, "source-ack.json", patchOp{Op: "replace", Path: "/rationale", Value: value(t, strings.Repeat(canary+" ", 80))}),
			"/rationale", []string{canary}},
		{"pattern on free text", "source-ack.v1",
			patched(t, "source-ack.json", patchOp{Op: "replace", Path: "/rationale", Value: value(t, "```\n"+secretLike+"\n```")}),
			"/rationale", []string{secretLike}},
		{"type", "source-ack.v1",
			patched(t, "source-ack.json", patchOp{Op: "replace", Path: "/rationale", Value: value(t, map[string]string{secretLike: canary})}),
			"/rationale", []string{secretLike, canary}},
		{"minimum", "source-list.v1",
			patched(t, "source-list.with-refs.json", patchOp{Op: "replace", Path: "/sources/0/size_bytes", Value: value(t, -987654321)}),
			// The library prints numbers reformatted (-9.87654321×10⁰⁸).
			"/sources/0/size_bytes", []string{"987654321", "87654321"}},
		{"uniqueItems", "init.v1",
			patched(t, "init.json", patchOp{Op: "replace", Path: "/candidate_sources", Value: value(t, []string{canary, canary})}),
			"/candidate_sources", []string{canary}},
		{"oneOf", "init.v1",
			patched(t, "init.json", patchOp{Op: "replace", Path: "/target", Value: value(t, canary)}),
			"/target", []string{canary}},
		{"map key in the location", "status.v1",
			patched(t, "status.minimal.json", patchOp{Op: "add", Path: "/metrics/ac_status_counts/" + secretLike, Value: value(t, -5)}),
			"/metrics/ac_status_counts", []string{secretLike}},
	}
	for _, c := range cases {
		err := v.ValidateJSON(c.schema, c.doc)
		env := requireRedacted(t, c.name, err, c.needles...)
		found := false
		for _, d := range env.Error.Details {
			found = found || d.Pointer == c.pointer
			if d.Problem == "" {
				t.Errorf("%s: a detail without a problem: %+v", c.name, d)
			}
		}
		if !found {
			t.Errorf("%s: no detail points at %q: %+v", c.name, c.pointer, env.Error.Details)
		}
	}
}

// A document that is not JSON is refused without quoting it.
func TestInvalidJSONIsNotEchoed(t *testing.T) {
	v := validator(t)
	for _, doc := range []string{
		`{"rationale": "` + secretLike,
		`{"rationale": "x"} ` + secretLike,
		secretLike,
		`{"rationale": "x", ` + secretLike + `}`,
	} {
		requireRedacted(t, doc[:12], v.ValidateJSON("source-ack.v1", []byte(doc)), secretLike)
	}
}

// The safe messages still say what is wrong.
func TestValidationErrorsStayUseful(t *testing.T) {
	v := validator(t)
	cases := []struct {
		schema string
		doc    []byte
		want   string
	}{
		{"source-ack.v1", patched(t, "source-ack.json", patchOp{Op: "replace", Path: "/observed_sha256", Value: value(t, "x")}), "pattern ^[0-9a-f]{64}$"},
		{"source-ack.v1", patched(t, "source-ack.json", patchOp{Op: "remove", Path: "/rationale"}), "rationale"},
		{"source-ack.v1", patched(t, "source-ack.json", patchOp{Op: "replace", Path: "/rationale", Value: value(t, 1)}), "string"},
		{"source-list.v1", patched(t, "source-list.with-refs.json", patchOp{Op: "replace", Path: "/excerpt_storage", Value: value(t, "x")}), `"hash_only"`},
	}
	for _, c := range cases {
		err := v.ValidateJSON(c.schema, c.doc)
		details := apperr.From(err).Details
		text := ""
		for _, d := range details {
			text += d.Pointer + " " + d.Problem + "\n"
		}
		if !strings.Contains(text, c.want) {
			t.Errorf("%s: details lack %q:\n%s", c.schema, c.want, text)
		}
	}
}
