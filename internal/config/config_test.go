package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
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
	if err != nil || found || cfg != (Config{}) {
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
