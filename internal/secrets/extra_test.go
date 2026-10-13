package secrets_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// Extra rules from config.toml apply after the built-in ones, which they
// can neither remove nor override.
func TestExtraRulesApplyAfterTheBuiltinOnes(t *testing.T) {
	s, err := secrets.New([]secrets.ExtraRule{{ID: "custom_acme_token", Pattern: `acme_[0-9a-f]{32}`}})
	if err != nil {
		t.Fatal(err)
	}
	acme := "acme_" + rep("0f", 16)
	if got, ok := s.Match("deploy with " + acme); !ok || got != "custom_acme_token" {
		t.Fatalf("Match(extra) = %q, %v", got, ok)
	}
	if got, _ := s.Match(acme + " " + "ghp_" + rep("A1b2", 9)); got != "github_token" {
		t.Fatalf("a built-in hit must be reported first, got %q", got)
	}
	for _, r := range builtinSamples {
		if got, ok := s.Match(r.hits[0]); !ok || got != r.rule {
			t.Errorf("built-in %s no longer hits with extra rules: %q, %v", r.rule, got, ok)
		}
	}
}

func TestInvalidExtraRulesAreRejected(t *testing.T) {
	valid := secrets.ExtraRule{ID: "custom_ok", Pattern: "ok_[0-9]{4}"}
	many := make([]secrets.ExtraRule, 51)
	for i := range many {
		many[i] = secrets.ExtraRule{ID: fmt.Sprintf("custom_r%d", i), Pattern: fmt.Sprintf("r%d_[0-9]{4}", i)}
	}
	cases := map[string][]secrets.ExtraRule{
		"id without prefix":    {{ID: "acme", Pattern: "acme_[0-9]{4}"}},
		"id with only prefix":  {{ID: "custom_", Pattern: "acme_[0-9]{4}"}},
		"id with upper case":   {{ID: "custom_Acme", Pattern: "acme_[0-9]{4}"}},
		"id too long":          {{ID: "custom_" + strings.Repeat("a", 41), Pattern: "acme_[0-9]{4}"}},
		"built-in id":          {{ID: "github_token", Pattern: "acme_[0-9]{4}"}},
		"duplicate id":         {valid, valid},
		"more than 50 rules":   many,
		"empty pattern":        {{ID: "custom_a", Pattern: ""}},
		"pattern too long":     {{ID: "custom_a", Pattern: strings.Repeat("a", 501)}},
		"matches empty string": {{ID: "custom_a", Pattern: "a*"}},
		"matches only empty":   {{ID: "custom_a", Pattern: "^$"}},
		"invalid regexp":       {{ID: "custom_a", Pattern: "acme_(["}},
	}
	for name, rules := range cases {
		err := secrets.ValidateExtraRules(rules)
		if apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Errorf("%s: ValidateExtraRules = %v, want an environment error (exit 6)", name, err)
			continue
		}
		if !strings.Contains(err.Error(), "extra_rules[") {
			t.Errorf("%s: error does not say which rule: %v", name, err)
		}
		if _, err := secrets.New(rules); apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Errorf("%s: New accepted invalid rules: %v", name, err)
		}
	}
	if err := secrets.ValidateExtraRules([]secrets.ExtraRule{valid}); err != nil {
		t.Errorf("valid rule rejected: %v", err)
	}
	if err := secrets.ValidateExtraRules(many[:50]); err != nil {
		t.Errorf("50 rules rejected: %v", err)
	}
}

// codex-review-009: an invalid extra rule must not leak its pattern (or an
// invalid id) through the error, which doctor and the CLI print.
func TestInvalidExtraRuleErrorsNeverEchoThePattern(t *testing.T) {
	canary := testutil.NewCanary(t)
	cases := map[string]secrets.ExtraRule{
		"unclosed group":      {ID: "custom_a", Pattern: "(" + canary},
		"bad character class": {ID: "custom_a", Pattern: canary + "[z-a]"},
		"bad repetition":      {ID: "custom_a", Pattern: canary + "**"},
		"too long":            {ID: "custom_a", Pattern: canary + strings.Repeat("a", 500)},
		"matches empty":       {ID: "custom_a", Pattern: "(" + canary + ")?"},
		"invalid id":          {ID: "custom_" + canary, Pattern: "acme_[0-9]{4}"},
	}
	for name, r := range cases {
		err := secrets.ValidateExtraRules([]secrets.ExtraRule{r})
		if err == nil {
			t.Errorf("%s: accepted", name)
			continue
		}
		envelope, _ := json.Marshal(apperr.ToEnvelope(err))
		low := strings.ToLower(canary)
		for _, out := range []string{err.Error(), string(envelope)} {
			if strings.Contains(out, canary) || strings.Contains(strings.ToLower(out), low) {
				t.Errorf("%s: error output contains the pattern or id: %s", name, out)
			}
		}
	}
	// The error still says what is wrong, without the pattern.
	err := secrets.ValidateExtraRules([]secrets.ExtraRule{cases["unclosed group"]})
	if err == nil || !strings.Contains(err.Error(), "missing closing )") {
		t.Errorf("error should name the regexp error kind: %v", err)
	}
}
