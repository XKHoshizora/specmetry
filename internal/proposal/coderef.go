package proposal

import (
	"bytes"
	"context"
	"io"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/gitx"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/source"
)

// CheckedCodeRef is a code reference as checked and stored (V-P08). Exists
// is a mark: it shows that the file, and the lines, are there, never that
// a criterion is met.
type CheckedCodeRef struct {
	Path   string  `json:"path"`
	Lines  *string `json:"lines"`
	Exists bool    `json:"exists"`
	// CheckedHead is the commit the reference was checked against; null
	// outside git or before the first commit.
	CheckedHead *string `json:"checked_head"`
}

// CheckCodeRefs checks code references where the command runs. In a git
// project they are checked against the HEAD commit of the current worktree:
// the path must be a regular file of its tree (not a directory, symbolic
// link or submodule) and a line range must lie within the file; a file
// that is not committed is not there, and a worktree without a commit has
// no file to point at. Every object the answer rests on (the commit, the
// trees on the way to each path, each file's blob) is read whole and must
// be the object its ID names (gitx.FileLines, review 068 B3): one that is
// damaged, holds another object, or cannot be read is an environment error
// (exit 6), never a mark. Outside git, the path must be a regular file
// under the project root, also after resolving links. Nothing of a file
// but its line count is read, and nothing is kept.
func CheckCodeRefs(ctx context.Context, w project.Where, refs [][]CodeRef) ([][]CheckedCodeRef, error) {
	return checkCodeRefs(ctx, w, refs, func() (string, error) { return gitx.CommitOf(ctx, w.Current, "HEAD") })
}

// fullCommit is a full commit ID (TD-26: SHA-1).
var fullCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)

// CheckCodeRefsAt is CheckCodeRefs against head, the HEAD commit the caller
// captured before (WP2.4 spec §7): HEAD is not resolved again, so every
// reference is checked against, and names, the commit the caller records
// with them, even if HEAD has moved since. head "" is a worktree with no
// commit, and the only head outside git; a head that is not a full commit
// ID is an internal error.
func CheckCodeRefsAt(ctx context.Context, w project.Where, head string, refs [][]CodeRef) ([][]CheckedCodeRef, error) {
	if (!w.Git && head != "") || (head != "" && !fullCommit.MatchString(head)) {
		return nil, apperr.New(apperr.Internal, "CheckCodeRefsAt needs the full ID of a commit of a git project, or no head")
	}
	return checkCodeRefs(ctx, w, refs, func() (string, error) { return head, nil })
}

// checkCodeRefs checks refs, with headOf giving the commit of a git
// project once the object reads are known to be safe.
func checkCodeRefs(ctx context.Context, w project.Where, refs [][]CodeRef, headOf func() (string, error)) ([][]CheckedCodeRef, error) {
	out := make([][]CheckedCodeRef, len(refs))
	total := 0
	for _, r := range refs {
		total += len(r)
	}
	if total == 0 {
		for i := range out {
			out[i] = []CheckedCodeRef{}
		}
		return out, nil
	}
	clean := func(p string) string { return path.Clean(strings.ReplaceAll(p, `\`, "/")) }
	var paths []string
	for _, list := range refs {
		for _, r := range list {
			if p := clean(r.Path); !slices.Contains(paths, p) {
				paths = append(paths, p)
			}
		}
	}
	exists, err := fileChecker(ctx, w, headOf, paths)
	if err != nil {
		return nil, err
	}
	for i, list := range refs {
		out[i] = make([]CheckedCodeRef, len(list))
		for k, r := range list {
			c := CheckedCodeRef{Path: r.Path, CheckedHead: exists.head}
			if r.Lines != "" {
				lines := r.Lines
				c.Lines = &lines
			}
			lines, ok, err := exists.lines(ctx, clean(r.Path))
			if err != nil {
				return nil, err
			}
			// A malformed range is refused before this (CheckStatic); here
			// it is never taken for no range either.
			if span, valid := parseLines(r.Lines); ok && valid {
				c.Exists = !span.set || span.to <= lines
			}
			out[i][k] = c
		}
	}
	return out, nil
}

// files answers whether a path is a file where the references are checked,
// and how many lines it has.
type files struct {
	head  *string
	lines func(ctx context.Context, p string) (int, bool, error)
}

// fileChecker readies the check of paths, every path its lines will be
// asked for.
func fileChecker(ctx context.Context, w project.Where, headOf func() (string, error), paths []string) (*files, error) {
	if !w.Git {
		root, err := filepath.EvalSymlinks(w.Root)
		if err != nil {
			root = w.Root
		}
		return &files{lines: func(ctx context.Context, p string) (int, bool, error) {
			real, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(p)))
			if err != nil {
				return 0, false, nil
			}
			if rel, err := filepath.Rel(root, real); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return 0, false, nil
			}
			f, err := source.OpenRegular(real)
			if err != nil {
				return 0, false, nil
			}
			defer func() { _ = f.Close() }()
			n, err := countLines(ctx, f)
			return n, err == nil, err
		}}, nil
	}
	// PD-26: a git that could fetch an object the repository lacks reads
	// none.
	if err := gitx.CheckObjectReads(ctx); err != nil {
		return nil, err
	}
	// git would fetch the objects a partial clone lacks, from its promisor
	// remote; Specmetry never does (codex-review-020 B1), so it reads no
	// object there at all.
	partial, err := gitx.PartialClone(ctx, w.Current)
	if err != nil {
		return nil, err
	}
	if partial {
		return nil, apperr.New(apperr.Environment,
			"this repository is a partial clone: git would fetch the objects it lacks from the promisor remote, which Specmetry never does, so code references cannot be checked here; nothing was written; check them in a full clone, or submit without code references")
	}
	head, err := headOf()
	if err != nil {
		return nil, err
	}
	if head == "" {
		return &files{lines: func(context.Context, string) (int, bool, error) { return 0, false, nil }}, nil
	}
	// The line counts of a checked read, not the objects themselves, are
	// what is kept (review 068 B3).
	counted, err := gitx.FileLines(ctx, w.Current, head, paths)
	if err != nil {
		return nil, err
	}
	return &files{head: &head, lines: func(_ context.Context, p string) (int, bool, error) {
		n, ok := counted[p]
		return n, ok, nil
	}}, nil
}

// countLines counts the lines of r a chunk at a time: its line breaks, and
// one more for a last line without one.
func countLines(ctx context.Context, r io.Reader) (int, error) {
	buf := make([]byte, chunkSize)
	n, size := 0, 0
	var last byte
	for {
		if err := ctx.Err(); err != nil {
			return 0, interrupted(err)
		}
		k, err := r.Read(buf)
		chunk := buf[:k]
		n += bytes.Count(chunk, []byte{'\n'})
		if len(chunk) > 0 {
			last, size = chunk[len(chunk)-1], size+len(chunk)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}
	if size > 0 && last != '\n' {
		n++
	}
	return n, nil
}
