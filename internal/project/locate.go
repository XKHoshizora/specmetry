package project

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/gitx"
)

// Where places a directory in its project, found without the database
// (architecture §6.1.2): for a git project the main worktree is the root;
// outside git the root is the nearest directory with a marker, or the
// directory itself. Paths are canonical.
type Where struct {
	Git bool
	// Root is the main worktree, or the project directory outside git.
	Root string
	// Current is the worktree the directory is in (Root outside git).
	Current string
	// CommonDir is the git common directory ("" outside git).
	CommonDir string
	// GitDir is the git directory of the Current worktree ("" outside
	// git).
	GitDir string
}

// Locate finds the project root of dir.
func Locate(ctx context.Context, dir string) (Where, error) {
	dir = gitx.Canonical(dir)
	loc, err := gitx.Locate(ctx, dir)
	switch {
	case errors.Is(err, gitx.ErrNotRepository):
		root := dir
		if r, found := FindMarker(dir); found {
			root = r
		}
		return Where{Root: root, Current: root}, nil
	case err != nil:
		return Where{}, err
	}
	return Where{Git: true, Root: loc.Main, Current: loc.Worktree, CommonDir: loc.CommonDir, GitDir: loc.GitDir}, nil
}

// IsDocumentType reports whether a file name has a document type: the
// source type whitelist (data-model §4.3), compared ignoring case.
func IsDocumentType(name string) bool {
	ext := path.Ext(name)
	return ext != "" && ext != name && documentExtensions[strings.ToLower(ext)]
}

// Registered is a registered project.
type Registered struct {
	ID, Name string
	Where    Where
	// ExcerptStorage is hash_only or full (PD-24).
	ExcerptStorage string
}

// Lookup finds the registered project of w in the database. A directory
// that belongs to none is not_initialized (exit code 4).
func Lookup(ctx context.Context, tx *sql.Tx, w Where) (Registered, error) {
	var r Registered
	var raw string
	var err error
	if w.Git {
		err = tx.QueryRowContext(ctx, `SELECT id, name, settings_json FROM projects WHERE git_common_dir = ?`, w.CommonDir).
			Scan(&r.ID, &r.Name, &raw)
	} else {
		err = tx.QueryRowContext(ctx, `SELECT p.id, p.name, p.settings_json
FROM projects p JOIN worktrees w ON w.id = p.main_worktree_id
WHERE p.git_common_dir IS NULL AND w.path = ?`, w.Root).Scan(&r.ID, &r.Name, &raw)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return Registered{}, ErrNotInitialized()
	}
	if err != nil {
		return Registered{}, err
	}
	var s settings
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return Registered{}, apperr.Wrap(apperr.Internal, err, "the project settings are not valid JSON")
	}
	r.Where = w
	r.ExcerptStorage = "hash_only"
	var mode string
	if json.Unmarshal(s["excerpt_storage"], &mode) == nil && mode != "" {
		r.ExcerptStorage = mode
	}
	return r, nil
}

// ErrNotInitialized is the error for a directory outside any registered
// project.
func ErrNotInitialized() error {
	return apperr.New(apperr.NotInitialized,
		"the current directory does not belong to an initialized project; run specmetry init first")
}
