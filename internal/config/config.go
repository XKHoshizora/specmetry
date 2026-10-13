// Package config loads $SPECMETRY_HOME/config.toml. Unknown keys are
// rejected so that typos never silently change behaviour.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/pelletier/go-toml/v2"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/secrets"
)

// Config is the user configuration. Fields are added only when a milestone
// needs them.
type Config struct {
	User    User    `toml:"user"`
	Secrets Secrets `toml:"secrets"`
}

// Secrets holds the secret rules added to the built-in ones
// (docs/architecture.md §10.8).
type Secrets struct {
	ExtraRules []secrets.ExtraRule `toml:"extra_rules"`
}

// User describes the local human operator.
type User struct {
	// DisplayName is used for actor "human:<display_name>" (data-model.md §1).
	DisplayName string `toml:"display_name"`
}

// Load reads path. A missing file yields the zero Config and found=false.
func Load(path string) (cfg Config, found bool, err error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, false, nil
	}
	if err != nil {
		return Config{}, false, apperr.Wrap(apperr.Environment, err, "cannot read "+path)
	}
	dec := toml.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, true, apperr.New(apperr.Environment, "invalid configuration file "+path+": "+decodeProblem(err))
	}
	if err := secrets.ValidateExtraRules(cfg.Secrets.ExtraRules); err != nil {
		return Config{}, true, apperr.New(apperr.Environment, "invalid configuration file "+path+": "+apperr.From(err).Message)
	}
	return cfg, true, nil
}

// decodeProblem locates a decoding error without quoting the file. The
// parser's own messages include characters of the file, which may hold
// patterns built around secrets (docs/architecture.md §10.8).
func decodeProblem(err error) string {
	var strict *toml.StrictMissingError
	if errors.As(err, &strict) && len(strict.Errors) > 0 {
		row, col := strict.Errors[0].Position()
		return fmt.Sprintf("unknown key at line %d, column %d", row, col)
	}
	var de *toml.DecodeError
	if errors.As(err, &de) {
		row, col := de.Position()
		return fmt.Sprintf("parse error at line %d, column %d", row, col)
	}
	return "the file cannot be parsed"
}
