package decode

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// encodeEnv writes a map the way go env -json does (go1.27.3
// cmd/go/internal/envcmd/env.go:605-620).
func encodeEnv(t *testing.T, m map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "\t")
	if err := enc.Encode(m); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestEnvEncoderOutput decodes the encoder's own output, including values
// the encoder escapes (HTML characters, control bytes, U+2028) or coerces
// (invalid UTF-8 becomes U+FFFD); encoding/json's Unmarshal is the oracle
// for the values.
func TestEnvEncoderOutput(t *testing.T) {
	m := map[string]string{
		"GOROOT":      `C:\Program Files\Go`,
		"GOVERSION":   "go1.27.3",
		"GOOS":        "<linux>&",
		"GOARCH":      "a\"b\nc\u2028d\x01",
		"CGO_ENABLED": "0",
		"GOFLAGS":     "",
		"GOWORK":      "bad\xffutf8 r\u00fcnner \U0001F600",
		"GOTOOLCHAIN": "local",
	}
	out := encodeEnv(t, m)
	got, err := DecodeEnv(out, spec.EnvNames)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	var want map[string]string
	if err := json.Unmarshal(out, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	if got["GOWORK"] != "bad\uFFFDutf8 r\u00fcnner \U0001F600" {
		t.Fatalf("GOWORK %q", got["GOWORK"])
	}
}

func envErr(t *testing.T, in string, names []string) *Error {
	t.Helper()
	_, err := DecodeEnv([]byte(in), names)
	var de *Error
	if !errors.As(err, &de) {
		t.Fatalf("want *Error, got %v", err)
	}
	if de.Call != spec.CallEnv {
		t.Fatalf("call %v", de.Call)
	}
	return de
}

func TestEnvErrors(t *testing.T) {
	two := []string{"A", "B"}
	cases := []struct {
		name  string
		in    string
		names []string
		kind  Kind
		at    string
	}{
		{"no-names", "{}\n", nil, KindUsage, "$"},
		{"repeated-name", "{}\n", []string{"A", "A"}, KindUsage, "$"},
		{"empty-object", "{}\n", two, KindMissing, "$.A"},
		{"missing-in-names-order", "{\"B\":\"\"}\n", two, KindMissing, "$.A"},
		{"null-before-type", "{\"A\":1,\"B\":null}\n", two, KindNull, "$.B"},
		{"missing-before-type", "{\"A\":1}\n", two, KindMissing, "$.B"},
		{"key-before-missing", "{\"A\":\"\",\"C\":\"\"}\n", two, KindKey, "$.C"},
		{"case-of-requested", "{\"a\":\"\",\"B\":\"\"}\n", two, KindCase, "$.a"},
		{"bool-value", "{\"A\":true,\"B\":\"\"}\n", two, KindType, "$.A"},
		{"array-value", "{\"A\":[],\"B\":\"\"}\n", two, KindType, "$.A"},
		{"object-value", "{\"A\":{},\"B\":\"\"}\n", two, KindType, "$.A"},
		{"odd-key-path", "{\"A\":\"\",\"B\":\"\",\"x y\":\"\"}\n", two, KindKey, `$["x y"]`},
		{"compact-ok-then-tab", "{\"A\":\"\",\"B\":\"\"}\t", two, KindTrailing, "$"},
		{"second-object-after-space", "{\"A\":\"\",\"B\":\"\"}\n {}\n", two, KindCount, "$"},
		{"string-top-level", "\"x\"\n", two, KindForm, "$"},
		{"unclosed", "{\"A\":\"\",\"B\":\"\"\n", two, KindSyntax, "$"},
		{"trailing-comma", "{\"A\":\"\",\"B\":\"\",}\n", two, KindSyntax, "$"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := envErr(t, c.in, c.names)
			if e.Kind != c.kind || e.At != c.at {
				t.Fatalf("got %s at %s (%s), want %s at %s", e.Kind, e.At, e.Detail, c.kind, c.at)
			}
		})
	}
}

// TestEnvEmptyIsObtained: an empty string is a value, and whitespace
// inside the object (including CRLF) is JSON whitespace.
func TestEnvEmptyIsObtained(t *testing.T) {
	in := "{\r\n  \"A\" : \"\" ,\r\n\t\"B\":\"x\"}\n"
	got, err := DecodeEnv([]byte(in), []string{"A", "B"})
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := got["A"]; !ok || v != "" || got["B"] != "x" || len(got) != 2 {
		t.Fatalf("%q", got)
	}
}

func TestEnvErrorOffsets(t *testing.T) {
	in := strings.Replace(envLinux, `"GOFLAGS": ""`, `"GOFLAGS": null`, 1)
	e := envErr(t, in, spec.EnvNames)
	if e.Kind != KindNull || !strings.HasPrefix(in[e.Offset:], "null") {
		t.Fatalf("%v", e)
	}
	in = strings.Replace(envLinux, `"GOOS"`, `"Goos"`, 1)
	e = envErr(t, in, spec.EnvNames)
	if e.Kind != KindCase || !strings.HasPrefix(in[e.Offset:], `"Goos"`) {
		t.Fatalf("%v", e)
	}
	e = envErr(t, envLinux+"zz", spec.EnvNames)
	if e.Kind != KindTrailing || e.Offset != len(envLinux)-1 {
		t.Fatalf("%v", e)
	}
}
