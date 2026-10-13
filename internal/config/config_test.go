package config

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadMissingFile(t *testing.T) {
	cfg, found, err := Load(filepath.Join(t.TempDir(), "absent.toml"))
	if err != nil || found || !reflect.DeepEqual(cfg, Config{}) {
		t.Fatalf("Load(missing) = %+v, %v, %v", cfg, found, err)
	}
}

func TestLoadValidFile(t *testing.T) {
	cfg, found, err := Load(write(t, "[user]\ndisplay_name = \"Hoshi 星\"\n"))
	if err != nil || !found || cfg.User.DisplayName != "Hoshi 星" {
		t.Fatalf("Load = %+v, %v, %v", cfg, found, err)
	}
}

func TestLoadRejectsUnknownKeysAndBadSyntax(t *testing.T) {
	for _, content := range []string{"[user]\ndisplay_nam = \"x\"\n", "[telemetry]\nenabled = true\n", "[user\n"} {
		_, _, err := Load(write(t, content))
		if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "config.toml") {
			t.Errorf("Load(%q) err = %v", content, err)
		}
	}
}

// architecture §10.8: extra secret rules are added under [secrets].
func TestLoadSecretsExtraRules(t *testing.T) {
	cfg, _, err := Load(write(t, "[secrets]\nextra_rules = [\n  { id = \"custom_acme_token\", pattern = \"acme_[0-9a-f]{32}\" },\n]\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []secrets.ExtraRule{{ID: "custom_acme_token", Pattern: "acme_[0-9a-f]{32}"}}
	if !reflect.DeepEqual(cfg.Secrets.ExtraRules, want) {
		t.Fatalf("extra rules = %+v", cfg.Secrets.ExtraRules)
	}
}

// MVP-PRV-07: an invalid extra rule makes loading fail (exit code 6); the
// error does not echo the pattern (codex-review-009).
func TestLoadRejectsInvalidExtraRuleWithoutEchoingIt(t *testing.T) {
	canary := testutil.NewCanary(t)
	_, _, err := Load(write(t, "[secrets]\nextra_rules = [ { id = \"custom_a\", pattern = \"("+canary+"\" } ]\n"))
	if apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("err = %v, want an environment error", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "config.toml") || !strings.Contains(msg, "extra_rules[0]") || strings.Contains(msg, canary) {
		t.Fatalf("err = %q", msg)
	}
}

var safeLoadError = regexp.MustCompile(`config\.toml: (parse error|unknown key) at line \d+, column \d+$`)

// Loading errors give the position only, never the file's content: the
// file may hold patterns built around secrets.
func TestLoadErrorsNeverEchoTheFileContent(t *testing.T) {
	canary := testutil.NewCanary(t)
	marker := "Ж" + canary
	for _, content := range []string{
		"[secrets]\nextra_rules = [ { id = \"custom_a\", pattern = " + marker + " } ]\n",
		"[secrets]\nextra_rules = [ { id = \"custom_a\", pattern = \"\\Ж" + canary + "\" } ]\n",
		"[secrets]\nextra_rules = [ { id = \"custom_a\", pattern = '" + marker + " } ]\n",
		"[secrets]\n" + "k" + canary + " = \"" + marker + "\"\n",
		"[secrets]\nextra_rules = [ { id = \"custom_a\", pattern = 12 } ]\n",
	} {
		_, _, err := Load(write(t, content))
		if apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Errorf("Load(%q) err = %v", content, err)
			continue
		}
		// Only the position may follow the path: no parser text, which
		// quotes characters of the file.
		msg := err.Error()
		if !safeLoadError.MatchString(msg) || strings.Contains(msg, canary) {
			t.Errorf("Load(%q) err = %q", content, msg)
		}
	}
}
