package secrets_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// Every string value of a write input is scanned, wherever it sits
// (architecture §10.8); findings carry the field's JSON Pointer.
func TestJSONFindingsPointToEachField(t *testing.T) {
	s := scanner(t)
	token := "ghp_" + rep("A1b2", 9)
	var doc any
	raw := `{
		"summary": "plain text",
		"items": [
			{"data": {"statement": "no secret here"}},
			{"data": {"check": {"command": "curl -H 'Authorization: Bearer ` + rep("abc123", 4) + `' x", "rationale": "ok"}}}
		],
		"a/b": "` + token + `",
		"c~d": ["fine", "` + token + `"],
		"count": 3,
		"flag": true,
		"none": null
	}`
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	got := s.JSON(doc)
	want := []secrets.Finding{
		{Pointer: "/a~1b", RuleID: "github_token"},
		{Pointer: "/c~0d/1", RuleID: "github_token"},
		{Pointer: "/items/1/data/check/command", RuleID: "bearer_token"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON findings = %+v, want %+v", got, want)
	}
}

// Persisted command-line arguments are scanned too; the pointer names the
// flag.
func TestFieldFindings(t *testing.T) {
	s := scanner(t)
	got := s.Fields(
		secrets.Field{Pointer: "--name", Value: "demo"},
		secrets.Field{Pointer: "-m", Value: "token sk_live_" + rep("4eC3", 4)},
	)
	want := []secrets.Finding{{Pointer: "-m", RuleID: "stripe_secret_key"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Fields = %+v, want %+v", got, want)
	}
}

// Each field reports the first rule in table order with a hit, not the
// first match in the text.
func TestFirstRuleInTableOrderIsReported(t *testing.T) {
	s := scanner(t)
	jwt := "eyJ" + rep("a", 10) + ".eyJ" + rep("b", 10) + "." + rep("c", 10)
	v := jwt + " then " + "ghp_" + rep("A1b2", 9)
	if got, _ := s.Match(v); got != "github_token" {
		t.Fatalf("Match = %q, want github_token (earlier in the table than jwt)", got)
	}
}

func TestNoFindingsIsNoError(t *testing.T) {
	if err := secrets.Reject(nil); err != nil {
		t.Fatalf("Reject(nil) = %v", err)
	}
}

// MVP-PRV-07: a hit rejects the write with exit code 2; the error names the
// field and the rule only, and never contains the matched text.
func TestRejectionNeverEchoesTheMatchedText(t *testing.T) {
	s := scanner(t)
	canary := testutil.NewCanary(t)
	secret := "postgres://app:" + canary + "9x@db.internal/app"
	doc := map[string]any{"items": []any{map[string]any{"rationale": "see " + secret}}}
	err := secrets.Reject(s.JSON(doc))
	if err == nil {
		t.Fatal("expected a rejection")
	}
	e := apperr.From(err)
	if e.Code != apperr.InvalidInput || apperr.ExitCode(err) != apperr.ExitInvalidInput {
		t.Fatalf("code = %s, exit = %d", e.Code, apperr.ExitCode(err))
	}
	want := []apperr.Detail{{Pointer: "/items/0/rationale", Problem: "secret_detected: url_credentials"}}
	if !reflect.DeepEqual(e.Details, want) {
		t.Fatalf("details = %+v, want %+v", e.Details, want)
	}
	envelope, _ := json.Marshal(apperr.ToEnvelope(err))
	for _, out := range []string{err.Error(), string(envelope)} {
		if strings.Contains(out, canary) || strings.Contains(out, "postgres://") {
			t.Fatalf("error output echoes the matched text: %s", out)
		}
	}
}
