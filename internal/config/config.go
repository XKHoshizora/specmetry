// Package config loads $SPECMETRY_HOME/config.toml. Unknown keys are
// rejected so that typos never silently change behaviour.
package config

import (
	"bytes"
	"errors"
	"io/fs"
	"os"

	"github.com/pelletier/go-toml/v2"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// Config is the user configuration. Fields are added only when a milestone
// needs them.
type Config struct {
	User User `toml:"user"`
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
		return Config{}, true, apperr.Wrap(apperr.Environment, err, "invalid configuration file "+path)
	}
	return cfg, true, nil
}
