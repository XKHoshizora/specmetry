package secrets_test

import (
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/secrets"
)

// rep repeats s n times. Hit samples are assembled at run time so that the
// repository never contains a literal that looks like a secret
// (architecture §10.8).
func rep(s string, n int) string { return strings.Repeat(s, n) }

func scanner(t *testing.T) *secrets.Scanner {
	t.Helper()
	s, err := secrets.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

type ruleSamples struct {
	rule    string
	hits    []string
	nonHits []string
}

// The sample table of architecture §10.8, in rule order.
var builtinSamples = []ruleSamples{
	{"private_key",
		[]string{"-----BEGIN " + "RSA PRIVATE KEY-----", "-----BEGIN " + "OPENSSH PRIVATE KEY-----"},
		[]string{"-----BEGIN PUBLIC KEY-----", "-----BEGIN CERTIFICATE-----"}},
	{"aws_access_key_id",
		[]string{"AKIA" + rep("Q7", 8), "ASIA" + rep("ZX12", 4)},
		[]string{"AKIA" + rep("Q7", 7) + "Q", "akia" + rep("q7", 8)}},
	{"github_token",
		[]string{"ghp_" + rep("A1b2", 9), "github_pat_" + rep("11AB", 6)},
		[]string{"ghp_" + rep("A1b2", 8), "ghx_" + rep("A1b2", 9)}},
	{"gitlab_token",
		[]string{"glpat-" + rep("Ab3-", 5), "glpat-" + rep("x", 20)},
		[]string{"glpat-" + rep("x", 19), "gl-pat-" + rep("x", 20)}},
	{"slack_token",
		[]string{"xoxb-" + "1234567890-abc", "xoxp-" + rep("a1", 6)},
		[]string{"xoxz-" + rep("a1", 6), "xoxb-short"}},
	{"stripe_secret_key",
		[]string{"sk_live_" + rep("4eC3", 4), "rk_test_" + rep("Zz9", 6)},
		[]string{"pk_live_" + rep("4eC3", 4), "sk_live_abc"}},
	{"google_api_key",
		[]string{"AIza" + rep("b", 35), "AIza" + rep("Xy-_", 8) + "Q9z"},
		[]string{"AIza" + rep("b", 34), "AIZA" + rep("b", 35)}},
	{"anthropic_api_key",
		[]string{"sk-ant-api03-" + rep("Ab9_", 6), "sk-ant-admin01-" + rep("x", 20)},
		[]string{"sk-ant-short", "sk_ant_api03" + rep("x", 20)}},
	{"openai_api_key",
		[]string{"sk-" + rep("A1b2", 12), "sk-proj-" + rep("Ab_9-", 5)},
		[]string{"sk-short1", "task-" + rep("a1", 10)}},
	{"npm_token",
		[]string{"npm_" + rep("Ab12", 9), "npm_" + rep("z9", 18)},
		[]string{"npm_" + rep("Ab12", 8), "npm-" + rep("Ab12", 9)}},
	{"jwt",
		[]string{
			"eyJ" + rep("a", 10) + ".eyJ" + rep("b", 10) + "." + rep("c", 10),
			"eyJ" + rep("hb", 8) + ".eyJ" + rep("zd", 8) + "." + rep("S1", 8),
		},
		[]string{
			"eyJ" + rep("a", 10) + ".eyJ" + rep("b", 10),
			"eyJ" + rep("a", 5) + ".eyJ" + rep("b", 10) + "." + rep("c", 10),
		}},
	{"url_credentials",
		[]string{"postgres://app:" + "s3cr3tPass" + "@db.internal:5432/app", "https://deploy:" + rep("x9", 6) + "@example.com/repo.git"},
		[]string{"postgres://app:${DB_PASSWORD}@db.internal/app", "https://example.com/path:with:colons"}},
	{"bearer_token",
		[]string{"Authorization: Bearer " + rep("abc123", 4), "bearer " + rep("Z9y8", 6)},
		[]string{"Bearer $API_TOKEN", "bearer token-based-authentication-flow"}},
	{"generic_assignment",
		[]string{"DB_PASSWORD=" + "s3cr3t" + "Pass9", "api_key: " + rep("a1B2", 5)},
		[]string{"password: $DB_PASSWORD", "password=<your-password>"}},
}

func TestBuiltinRulesFollowTheDocumentedOrder(t *testing.T) {
	var want []string
	for _, r := range builtinSamples {
		want = append(want, r.rule)
	}
	if got := secrets.BuiltinRules(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("BuiltinRules() = %v, want %v", got, want)
	}
}

// MVP-PRV-07: every built-in rule has at least two hit and two non-hit
// samples; each hit is reported under its own rule, and no non-hit sample
// matches any rule.
func TestBuiltinRuleSamples(t *testing.T) {
	s := scanner(t)
	for _, r := range builtinSamples {
		if len(r.hits) < 2 || len(r.nonHits) < 2 {
			t.Errorf("%s: needs at least 2 hit and 2 non-hit samples", r.rule)
		}
		for i, v := range r.hits {
			if got, ok := s.Match(v); !ok || got != r.rule {
				t.Errorf("%s hit sample %d: Match = %q, %v", r.rule, i, got, ok)
			}
		}
		for i, v := range r.nonHits {
			if got, ok := s.Match(v); ok {
				t.Errorf("%s non-hit sample %d matched rule %q", r.rule, i, got)
			}
		}
	}
}

func TestOrdinaryTextIsNotASecret(t *testing.T) {
	s := scanner(t)
	for _, v := range []string{
		"用户只能查看自己的订单",
		"API key: configurable per tenant",
		"go test ./internal/orders -run TestIsolation",
		"export API_TOKEN=$API_TOKEN && ./deploy.sh",
		"git@github.com:org/repo.git",
		"risk-assessment-document-2026-final",
		"the refresh token rotation policy",
		"secretary: Jane",
		"",
	} {
		if got, ok := s.Match(v); ok {
			t.Errorf("%q matched rule %q", v, got)
		}
	}
}
