package secrets_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/secrets"
)

// rejectField runs one value through the public path a write command uses,
// New → JSON → Reject, as the "command" field of a submitted document.
func rejectField(t *testing.T, value string) error {
	t.Helper()
	s, err := secrets.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return secrets.Reject(s.JSON(map[string]any{"command": value}))
}

// assertRejected checks the rejection and that its output never contains
// the secret.
func assertRejected(t *testing.T, name, value, rule, secret string) {
	t.Helper()
	err := rejectField(t, value)
	if err == nil {
		t.Errorf("%s: %q was not rejected", name, value)
		return
	}
	want := []apperr.Detail{{Pointer: "/command", Problem: "secret_detected: " + rule}}
	if got := apperr.From(err).Details; !reflect.DeepEqual(got, want) {
		t.Errorf("%s: details = %+v, want %+v", name, got, want)
	}
	envelope, _ := json.Marshal(apperr.ToEnvelope(err))
	for _, out := range []string{err.Error(), string(envelope)} {
		if strings.Contains(out, secret) {
			t.Errorf("%s: error output contains the secret: %s", name, out)
		}
	}
}

// codex-review-010 B1: a command substitution is not an environment
// reference. Neither it nor the literal next to it may exempt a password.
func TestCommandSubstitutionDoesNotExemptAPassword(t *testing.T) {
	password := "s3cr3t" + "Pass9"
	assertRejected(t, "control", "password="+password, "generic_assignment", password)
	assertRejected(t, "after a complete substitution", "password=$(true)"+password, "generic_assignment", password)
	assertRejected(t, "URL password after a substitution", "postgres://app:$(true)"+password+"@db/app", "url_credentials", password)
	assertRejected(t, "unclosed substitution", "password=$("+password, "generic_assignment", password)
}

// The four complete environment reference forms stay exempt; a reference
// followed by a literal password is not a reference.
func TestOnlyCompleteEnvironmentReferencesAreExempt(t *testing.T) {
	for _, v := range []string{
		"password=$DB_PASSWORD",
		"password=${DB_PASSWORD}",
		"password=%DB_PASSWORD%",
		"password=$env:DB_PASSWORD",
		"postgres://app:$DB_PASSWORD@db/app",
		"postgres://app:${DB_PASSWORD}@db/app",
		"postgres://app:%DB_PASSWORD%@db/app",
		"postgres://app:$env:DB_PASSWORD@db/app",
	} {
		if err := rejectField(t, v); err != nil {
			t.Errorf("%q rejected: %v", v, err)
		}
	}
	password := "s3cr3t" + "Pass9"
	assertRejected(t, "reference then password", "password=${DB_PASSWORD}"+password, "generic_assignment", password)
	assertRejected(t, "URL reference then password", "postgres://app:${DB_PASSWORD}"+password+"@db/app", "url_credentials", password)
}

// codex-review-009, through the public path: a reference elsewhere in the
// field, or an exempt earlier candidate, never hides a secret.
func TestPublicPathChecksEveryCandidate(t *testing.T) {
	password := "hunter2" + "Pass"
	token := "ghp_" + rep("A1b2", 9)
	assertRejected(t, "reference and token", "API_KEY=$API_KEY ./deploy.sh --token "+token, "github_token", token)
	assertRejected(t, "reference then password", "DB_PASSWORD=$DB_PASSWORD ADMIN_PASSWORD="+password, "generic_assignment", password)
	assertRejected(t, "placeholder then password", "password=<your-password>; password="+password, "generic_assignment", password)
}

// codex-review-010 N1: a rule without a condition stops at the first match
// instead of collecting every match of the text.
func TestRuleWithoutConditionDoesNotCollectEveryMatch(t *testing.T) {
	s, err := secrets.New([]secrets.ExtraRule{{ID: "custom_any", Pattern: "."}})
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("a", 4000)
	if got, ok := s.Match(text); !ok || got != "custom_any" {
		t.Fatalf("Match = %q, %v", got, ok)
	}
	if allocs := testing.AllocsPerRun(20, func() { s.Match(text) }); allocs > 50 {
		t.Fatalf("Match allocated %.0f times per run; an unconditioned rule should not collect every match", allocs)
	}
}
