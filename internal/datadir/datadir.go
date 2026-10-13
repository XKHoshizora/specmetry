// Package datadir locates and secures Specmetry's local data directory
// ($SPECMETRY_HOME, default ~/.specmetry; docs/architecture.md §8 and §11.1).
package datadir

import (
	"errors"
	"fmt"
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
	// CompactionFile marks a tombstone deletion whose file compaction
	// (checkpoint and VACUUM) has not finished yet.
	CompactionFile = ".compaction-pending"
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

// Exists reports whether the data directory exists. A link in its place
// counts as existing, so that Check reports it instead of Ensure creating a
// directory behind it.
func (d Dir) Exists() (bool, error) {
	fi, err := os.Lstat(d.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, apperr.Wrap(apperr.Environment, err, "cannot read the data directory")
	}
	if !fi.IsDir() && !isLink(fi.Mode()) {
		return false, apperr.Newf(apperr.Environment, "data directory %s is not a directory", d.Path)
	}
	return true, nil
}

// Ensure makes the data directory ready to be written to (architecture.md
// §11.1). A missing directory is created with owner-only permissions. An
// existing one is not trusted: it and everything in it are checked, and if
// any of it is not private to the current user Ensure refuses before
// anything is written. Every writer calls Ensure first; doctor's Check
// reports the same problems but is not a substitute for it.
func (d Dir) Ensure() error {
	ok, err := d.Exists()
	if err != nil {
		return err
	}
	if !ok {
		if err := os.MkdirAll(filepath.Dir(d.Path), 0o700); err != nil {
			return apperr.Wrap(apperr.Environment, err, "cannot create the parent of the data directory")
		}
		if err := os.Mkdir(d.Path, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
			return apperr.Wrap(apperr.Environment, err, "cannot create the data directory")
		}
		if err := secureDir(d.Path); err != nil {
			return apperr.Wrap(apperr.Environment, err, "cannot restrict data directory permissions")
		}
	}
	problems, err := Check(d)
	if err != nil {
		return apperr.Wrap(apperr.Environment, err, "cannot check the permissions of the data directory")
	}
	if len(problems) == 0 {
		return nil
	}
	details := make([]apperr.Detail, len(problems))
	for i, p := range problems {
		details[i] = apperr.Detail{Problem: p}
	}
	return apperr.Newf(apperr.Environment,
		"the data directory %s is not private to the current user; refusing to write to it. Restrict it to the current user%s, or set %s to another location",
		d.Path, systemNote, EnvHome).WithDetails(details...)
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
// accessible to the current user (and SYSTEM on Windows). Check only reads.
//
// Links are problems wherever they appear, the data directory itself
// included: a link inside it could send writes to a location that is not
// private, and following one would check something other than what SQLite
// writes to.
func Check(d Dir) ([]string, error) {
	fi, err := os.Lstat(d.Path)
	if err != nil {
		return nil, err
	}
	if isLink(fi.Mode()) {
		return []string{d.Path + ": " + linkProblem}, nil
	}
	return checkPermissions(d.Path)
}

const linkProblem = "is a symbolic link or junction; Specmetry does not follow links in its data directory (set " +
	EnvHome + " to the real location)"

func isLink(m fs.FileMode) bool { return m&(fs.ModeSymlink|fs.ModeIrregular) != 0 }

// entryProblem reports an entry that is neither a regular file nor a
// directory; "" means its permissions still need checking.
func entryProblem(p string, d fs.DirEntry) string {
	switch t := d.Type(); {
	case isLink(t):
		return p + ": " + linkProblem
	case t.IsDir() || t.IsRegular():
		return ""
	default:
		return fmt.Sprintf("%s: unexpected file type %v", p, t)
	}
}
