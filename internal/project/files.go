package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/gitx"
)

// ListFiles lists the regular files of the worktree at root with their
// sizes, never their content (architecture §6.1.1). In a git project these
// are the files git sees: tracked ones and untracked ones that are not
// ignored. Otherwise the directory is walked. Symbolic links are not
// followed, and the excluded directories are skipped.
//
// A cancelled context stops the listing as soon as it is noticed, with an
// error: a listing cut short is never returned as if it were complete
// (codex-review-015 N2).
func ListFiles(ctx context.Context, root string, git bool) ([]File, error) {
	if err := ctx.Err(); err != nil {
		return nil, interrupted(err)
	}
	if git {
		names, err := gitx.ListFiles(ctx, root)
		if err != nil {
			return nil, err
		}
		var out []File
		for _, name := range names {
			if err := ctx.Err(); err != nil {
				return nil, interrupted(err)
			}
			if inExcludedDir(name) {
				continue
			}
			// Deleted, sparse or submodule entries are not regular files.
			fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(name)))
			if err != nil || !fi.Mode().IsRegular() {
				continue
			}
			out = append(out, File{Path: name, Size: fi.Size()})
		}
		return out, nil
	}
	var out []File
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if cerr := ctx.Err(); cerr != nil {
			return cerr
		}
		if p == root {
			return err
		}
		if err != nil {
			// An unreadable directory is skipped, not fatal.
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if excludedDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		out = append(out, File{Path: filepath.ToSlash(rel), Size: fi.Size()})
		return nil
	})
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, interrupted(err)
	}
	if err != nil {
		return nil, apperr.Wrap(apperr.Environment, err, "cannot list the files of "+root)
	}
	return out, nil
}

func interrupted(err error) error {
	return apperr.Wrap(apperr.Environment, err, "the file scan was interrupted before it finished; nothing was registered")
}

// The marker: <main worktree>/.specmetry/project.json (architecture §8).
const (
	MarkerSchema  = "specmetry.project/v1"
	markerDir     = ".specmetry"
	markerFile    = "project.json"
	maxMarkerSize = 64 << 10
)

// Marker is the content of the marker file. It points at the project in
// the database; the database, not the marker, is the authority.
type Marker struct {
	Schema    string `json:"schema"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
}

var ulidPattern = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

// MarkerPath returns the marker path of the project rooted at root.
func MarkerPath(root string) string { return filepath.Join(root, markerDir, markerFile) }

// markerDirExists checks <root>/.specmetry: it must be a real directory,
// not a file or a link that could lead outside the project.
func markerDirExists(root string) (bool, error) {
	fi, err := os.Lstat(filepath.Join(root, markerDir))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, apperr.Wrap(apperr.Environment, err, "cannot read "+filepath.Join(root, markerDir))
	}
	if !fi.IsDir() {
		return false, apperr.Newf(apperr.Environment, "%s is not a directory; Specmetry keeps its marker there", filepath.Join(root, markerDir))
	}
	return true, nil
}

// ReadMarker reads the marker of the project rooted at root. A missing
// marker is not an error; one that cannot be read or trusted is. Errors
// never quote the file.
func ReadMarker(root string) (Marker, bool, error) {
	exists, err := markerDirExists(root)
	if err != nil || !exists {
		return Marker{}, false, err
	}
	p := MarkerPath(root)
	fi, err := os.Lstat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return Marker{}, false, nil
	}
	if err != nil {
		return Marker{}, false, apperr.Wrap(apperr.Environment, err, "cannot read the marker "+p)
	}
	invalid := func(why string) (Marker, bool, error) {
		return Marker{}, false, apperr.Newf(apperr.Environment,
			"the marker %s is not valid (%s); check it, or remove it to register the directory as a new project", p, why)
	}
	if !fi.Mode().IsRegular() {
		return invalid("not a regular file")
	}
	if fi.Size() > maxMarkerSize {
		return invalid("too large")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return Marker{}, false, apperr.Wrap(apperr.Environment, err, "cannot read the marker "+p)
	}
	var m Marker
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return invalid("not a specmetry.project/v1 document")
	}
	if m.Schema != MarkerSchema {
		return invalid("not a specmetry.project/v1 document")
	}
	if !ulidPattern.MatchString(m.ProjectID) {
		return invalid("the project_id is not a ULID")
	}
	return m, true, nil
}

// WriteMarker writes the marker of the project rooted at root, replacing
// the file atomically. It refuses to write through anything but a real
// .specmetry directory.
func WriteMarker(root string, m Marker) error {
	exists, err := markerDirExists(root)
	if err != nil {
		return err
	}
	dir := filepath.Join(root, markerDir)
	if !exists {
		// Repository files keep git's usual modes: the marker holds only the
		// project ID and name.
		if err := os.Mkdir(dir, 0o755); err != nil { //nolint:gosec // a directory in the user's repository
			return apperr.Wrap(apperr.Environment, err, "cannot create "+dir)
		}
	}
	m.Schema = MarkerSchema
	b, err := json.Marshal(m)
	if err != nil {
		return apperr.Wrap(apperr.Internal, err, "cannot encode the marker")
	}
	tmp, err := os.CreateTemp(dir, ".project.json.*")
	if err != nil {
		return apperr.Wrap(apperr.Environment, err, "cannot write the marker in "+dir)
	}
	_, werr := tmp.Write(append(b, '\n'))
	if werr == nil {
		// CreateTemp makes the file private; the marker is an ordinary
		// repository file that Hooks of any local checkout may read.
		werr = tmp.Chmod(0o644)
	}
	if werr == nil {
		werr = tmp.Sync()
	}
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp.Name(), MarkerPath(root))
	}
	if werr != nil {
		_ = os.Remove(tmp.Name())
		return apperr.Wrap(apperr.Environment, werr, "cannot write the marker "+MarkerPath(root))
	}
	return nil
}

// FindMarker returns the nearest directory, from dir upwards, that holds a
// marker file. ReadMarker then decides whether the marker can be trusted.
func FindMarker(dir string) (string, bool) {
	for d := filepath.Clean(dir); ; {
		if _, err := os.Lstat(MarkerPath(d)); err == nil {
			return d, true
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", false
		}
		d = parent
	}
}

// ExcludeEntry is the line init adds to the repository's exclude file
// (PD-07, architecture §10.6).
const ExcludeEntry = "/.specmetry/"

// EnsureExcluded adds ExcludeEntry to the git exclude file at path unless
// an entry for the marker directory is already there. It only appends, so
// the user's entries are kept byte for byte. Errors never quote the file.
func EnsureExcluded(path string) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for _, line := range strings.Split(string(b), "\n") {
		switch strings.TrimSpace(line) {
		case "/.specmetry/", "/.specmetry", ".specmetry/", ".specmetry":
			return nil
		}
	}
	// The modes git itself gives info/ and info/exclude.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // git's own directory
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644) //nolint:gosec // git's own file
	if err != nil {
		return err
	}
	entry := ExcludeEntry + "\n"
	if len(b) > 0 && b[len(b)-1] != '\n' {
		entry = "\n" + entry
	}
	_, err = f.WriteString(entry)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
