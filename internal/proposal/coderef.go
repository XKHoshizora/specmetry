package proposal

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/gitx"
	"github.com/XKHoshizora/specmetry/internal/project"
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
// no file to point at. Outside git, the path must be a regular file under
// the project root, also after resolving links. Nothing of a file but its
// line count is read, and nothing is kept.
func CheckCodeRefs(ctx context.Context, w project.Where, refs [][]CodeRef) ([][]CheckedCodeRef, error) {
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
	exists, err := fileChecker(ctx, w)
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
			p := path.Clean(strings.ReplaceAll(r.Path, `\`, "/"))
			lines, ok, err := exists.lines(ctx, p)
			if err != nil {
				return nil, err
			}
			if ok {
				_, to, ranged := lineRange(r.Lines)
				c.Exists = !ranged || to <= lines
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

func fileChecker(ctx context.Context, w project.Where) (*files, error) {
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
			f, err := os.Open(real)
			if err != nil {
				return 0, false, nil
			}
			defer func() { _ = f.Close() }()
			if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() {
				return 0, false, nil
			}
			n, err := countLines(ctx, f)
			return n, err == nil, err
		}}, nil
	}
	head, err := gitx.CommitOf(ctx, w.Current, "HEAD")
	if err != nil {
		return nil, err
	}
	if head == "" {
		return &files{lines: func(context.Context, string) (int, bool, error) { return 0, false, nil }}, nil
	}
	tree, err := gitx.TreeFiles(ctx, w.Current, head)
	if err != nil {
		return nil, err
	}
	counted := map[string]int{}
	return &files{head: &head, lines: func(ctx context.Context, p string) (int, bool, error) {
		object, ok := tree[p]
		if !ok {
			return 0, false, nil
		}
		if n, done := counted[object]; done {
			return n, true, nil
		}
		n, err := gitx.CountLines(ctx, w.Current, object)
		if err != nil {
			return 0, false, err
		}
		counted[object] = n
		return n, true, nil
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
