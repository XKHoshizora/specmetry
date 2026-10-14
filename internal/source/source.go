// Package source registers and observes requirement sources
// (data-model.md §4.3; architecture §6.1.3). A source's content is only
// ever read to hash it, a chunk at a time, and is never kept.
package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/gitx"
	"github.com/XKHoshizora/specmetry/internal/project"
)

// ErrMissing means a source file cannot be read: it does not exist, is not
// a regular file, or reading it failed.
var ErrMissing = errors.New("the source file cannot be read")

// Member is a source as the source set hash sees it.
type Member struct{ ID, AckedSHA256 string }

// SetHash is the source set hash (data-model §4.3): the sha256 of
// "<source_id>:<acked_sha256>\n" for the active sources, sorted by ID. The
// same sources (IDs and acknowledged hashes) in any order give the same
// hash.
func SetHash(members []Member) string {
	sorted := slices.Clone(members)
	slices.SortFunc(sorted, func(a, b Member) int { return strings.Compare(a.ID, b.ID) })
	h := sha256.New()
	for _, m := range sorted {
		_, _ = io.WriteString(h, m.ID+":"+m.AckedSHA256+"\n")
	}
	return hex.EncodeToString(h.Sum(nil))
}

// chunkSize bounds what is held in memory while hashing.
const chunkSize = 1 << 20

// HashFile returns the sha256 and size of a regular file, read a chunk at
// a time. It checks for cancellation before every chunk and never returns
// the hash of part of a file. Errors never quote the file.
func HashFile(ctx context.Context, path string) (string, int64, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, interrupted(err)
	}
	// A FIFO or a device must not block the hash (OpenRegular).
	f, err := OpenRegular(path)
	if err != nil {
		return "", 0, ErrMissing
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	buf := make([]byte, chunkSize)
	var size int64
	for {
		if err := ctx.Err(); err != nil {
			return "", 0, interrupted(err)
		}
		n, err := f.Read(buf)
		_, _ = h.Write(buf[:n])
		size += int64(n)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", 0, ErrMissing
		}
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

func interrupted(err error) error {
	return apperr.Wrap(apperr.Environment, err, "reading the source was interrupted; nothing was written")
}

// Kind is a source kind (data-model §4.3).
type Kind string

// ParseKind parses a source kind.
func ParseKind(s string) (Kind, bool) {
	switch s {
	case "client_requirement", "product_doc", "design_doc", "issue", "note", "other":
		return Kind(s), true
	}
	return "", false
}

// Location says where a source lives.
type Location string

// Locations.
const (
	InRepo   Location = "in_repo"
	External Location = "external"
)

// Resolved is a file named as a source: where it is, the path to store and
// the file to read.
type Resolved struct {
	Location Location
	// Path is relative to the project root, with forward slashes, for an
	// in_repo source; the canonical absolute path for an external one.
	Path string
	// File is the canonical absolute path to read.
	File string
}

// Resolve resolves arg, relative to cwd, to a source of the project at w.
// Symbolic links are resolved: a source is where its content is. A file in
// the main worktree is in_repo; one in a linked worktree of the project is
// refused, whichever worktree the command runs in, because sources are
// observed in the main worktree (inLinkedWorktree); anything else is
// external. A missing file, a directory, a file inside a git directory, or
// one for which git cannot tell which worktree holds it, is invalid input
// (exit code 2).
//
// Errors name only arg, which the caller has scanned for secrets, never
// the resolved path: a link with a harmless name can lead to a file whose
// path holds a secret (codex-review-018 B1). The caller scans the resolved
// path before it uses it.
func Resolve(ctx context.Context, arg, cwd string, w project.Where) (Resolved, error) {
	p := arg
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	fi, err := os.Stat(p)
	if err != nil {
		return Resolved{}, apperr.Newf(apperr.InvalidInput, "%s does not exist or cannot be read", arg)
	}
	if !fi.Mode().IsRegular() {
		return Resolved{}, apperr.Newf(apperr.InvalidInput, "%s is not a file; a source is a document file", arg)
	}
	canon, err := filepath.EvalSymlinks(p)
	if err != nil {
		return Resolved{}, apperr.Newf(apperr.InvalidInput, "%s cannot be resolved", arg)
	}
	if w.CommonDir != "" {
		if _, ok := within(w.CommonDir, canon); ok {
			return Resolved{}, apperr.Newf(apperr.InvalidInput, "%s is inside the git directory, which holds no sources", arg)
		}
	}
	if w.Git {
		linked, err := inLinkedWorktree(ctx, canon, w)
		switch {
		case ctx.Err() != nil:
			return Resolved{}, interrupted(ctx.Err())
		case err != nil:
			return Resolved{}, apperr.Newf(apperr.InvalidInput,
				"%s cannot be a source: git cannot tell which worktree holds it (it may be inside a git directory, or in a repository git does not read), so it is not taken to be outside the project", arg)
		case linked:
			return Resolved{}, apperr.Newf(apperr.InvalidInput,
				"%s is in a linked worktree of the project; sources are read in the main worktree, so register the file there instead", arg)
		}
	}
	if rel, ok := within(w.Root, canon); ok {
		slash := filepath.ToSlash(rel)
		if slash == ".git" || strings.HasPrefix(slash, ".git/") {
			return Resolved{}, apperr.Newf(apperr.InvalidInput, "%s is inside the git directory, which holds no sources", arg)
		}
		return Resolved{Location: InRepo, Path: slash, File: canon}, nil
	}
	return Resolved{Location: External, Path: canon, File: canon}, nil
}

// inLinkedWorktree reports whether file is in a linked worktree of the
// git project at w rather than in its main worktree. git says which
// worktree holds the file's directory and that worktree's git common
// directory; the worktree is one of the project's if that directory is the
// project's. Directories are compared as files (os.SameFile), not by how
// their paths are spelled, and a linked worktree inside the main one is
// still a linked worktree. A file in no git repository is in none of the
// project's worktrees. When git cannot tell (the file is inside a git
// directory, or in a repository git refuses to read), the error says so:
// the file is not assumed to be outside the project. Errors may name
// paths; the caller does not pass them on.
func inLinkedWorktree(ctx context.Context, file string, w project.Where) (bool, error) {
	loc, err := gitx.Locate(ctx, filepath.Dir(file))
	if errors.Is(err, gitx.ErrNotRepository) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	ours, err := sameFile(loc.CommonDir, w.CommonDir)
	if err != nil || !ours {
		return false, err
	}
	main, err := sameFile(loc.Worktree, w.Root)
	return !main, err
}

func sameFile(a, b string) (bool, error) {
	fa, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	fb, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(fa, fb), nil
}

// within returns p relative to root if p is inside root.
func within(root, p string) (string, bool) {
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}
