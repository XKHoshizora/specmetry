// Package datadir locates and secures Specmetry's local data directory
// ($SPECMETRY_HOME, default ~/.specmetry; docs/architecture.md §8 and §11.1).
package datadir

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// EnvHome overrides the data directory location.
const EnvHome = "SPECMETRY_HOME"

// File and directory names inside the data directory.
const (
	DatabaseFile    = "specmetry.db"
	ConfigFile      = "config.toml"
	TombstonesFile  = "tombstones.json"
	MigrateLockFile = ".migrate.lock"
	BackupsDir      = "backups"
)

// Dir is the resolved data directory.
type Dir struct {
	Path    string
	FromEnv bool
}

// Resolve returns the data directory: $SPECMETRY_HOME if set, otherwise
// ~/.specmetry. It does not touch the file system.
func Resolve() (Dir, error) {
	if v := os.Getenv(EnvHome); v != "" {
		abs, err := filepath.Abs(v)
		if err != nil {
			return Dir{}, apperr.Wrap(apperr.Environment, err, EnvHome+" is not a usable path")
		}
		return Dir{Path: abs, FromEnv: true}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Dir{}, apperr.Wrap(apperr.Environment, err, "cannot determine the home directory; set "+EnvHome)
	}
	return Dir{Path: filepath.Join(home, ".specmetry")}, nil
}

// Join returns a path inside the data directory.
func (d Dir) Join(name string) string { return filepath.Join(d.Path, name) }

// Exists reports whether the data directory exists.
func (d Dir) Exists() (bool, error) {
	fi, err := os.Stat(d.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, apperr.Wrap(apperr.Environment, err, "cannot read the data directory")
	}
	if !fi.IsDir() {
		return false, apperr.Newf(apperr.Environment, "data directory %s is not a directory", d.Path)
	}
	return true, nil
}

// Ensure creates the data directory with owner-only permissions if it does
// not exist. An existing directory is left unchanged; Check reports any
// permission problems with it.
func (d Dir) Ensure() error {
	ok, err := d.Exists()
	if err != nil || ok {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(d.Path), 0o700); err != nil {
		return apperr.Wrap(apperr.Environment, err, "cannot create the parent of the data directory")
	}
	if err := os.Mkdir(d.Path, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return apperr.Wrap(apperr.Environment, err, "cannot create the data directory")
	}
	if err := secureDir(d.Path); err != nil {
		return apperr.Wrap(apperr.Environment, err, "cannot restrict data directory permissions")
	}
	return nil
}

// EnsureSub creates a subdirectory of the data directory. It inherits the
// data directory's protection (mode 0700 on Unix, inherited DACL on Windows).
func (d Dir) EnsureSub(name string) (string, error) {
	p := d.Join(name)
	if err := os.Mkdir(p, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return "", apperr.Wrap(apperr.Environment, err, "cannot create "+p)
	}
	if err := securePrivatePath(p, true); err != nil {
		return "", apperr.Wrap(apperr.Environment, err, "cannot restrict permissions of "+p)
	}
	return p, nil
}

// CreatePrivateFile creates path for writing with owner-only permissions.
// With exclusive set it fails if the file already exists.
func CreatePrivateFile(path string, exclusive bool) (*os.File, error) {
	flags := os.O_RDWR | os.O_CREATE
	if exclusive {
		flags |= os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		return nil, err
	}
	if err := securePrivatePath(path, false); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

// Check returns human-readable permission problems for the data directory
// and everything inside it. An empty result means the directory is only
// accessible to the current user (and SYSTEM on Windows).
func Check(d Dir) ([]string, error) {
	return checkPermissions(d.Path)
}
