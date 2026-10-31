package checkrun

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/gitx"
)

// Interpreter is the interpreter step 7 chose for a shell (spec §5.3,
// §19.8): Path is its final path, which the check is started with.
// PathPrefix is set for Windows' sh and bash only: <R>\usr\bin, which D2
// puts first on that check's PATH alone.
type Interpreter struct {
	Shell      string
	Path       string
	PathPrefix string
}

// foldCase is whether paths are compared without letter case when an
// interpreter is excluded (spec §19.8): on Windows and macOS. A
// case-sensitive volume on macOS then excludes more, never less.
var foldCase = runtime.GOOS == "windows" || runtime.GOOS == "darwin"

// finder looks the interpreters of one batch up (spec §5.3, §19.8). roots
// are the final paths of every registered worktree and W, inside which no
// interpreter is used (G11); when one cannot be resolved, where an
// interpreter is cannot be verified and none is used. path is the PATH
// searched; execPath gives the exec path of the git gitx runs (Windows'
// sh and bash). released holds the first close of finalPath's own that
// did not confirm its release (Review 123 B-123-2): the lookup then fails,
// whatever else it found.
type finder struct {
	roots      []string
	unverified bool
	path       string
	execPath   func(context.Context) (string, error)
	released   error
}

func newFinder(worktrees []string, path string, execPath func(context.Context) (string, error)) *finder {
	f := &finder{path: path, execPath: execPath}
	for _, w := range worktrees {
		root, err := f.final(w)
		if err != nil {
			f.unverified = true
			continue
		}
		f.roots = append(f.roots, root)
	}
	return f
}

// final is finalPath, a close of its own that did not confirm its release
// kept as the lookup's failure; for the caller it is a path not had, so
// nothing found after it can make the lookup succeed.
func (f *finder) final(p string) (string, error) {
	final, err := finalPath(p)
	var rel *ReleaseError
	if errors.As(err, &rel) && f.released == nil {
		f.released = err
	}
	return final, err
}

// excluded reports whether the final path p is inside one of the roots.
func (f *finder) excluded(p string) bool {
	for _, root := range f.roots {
		if within(p, root, foldCase) {
			return true
		}
	}
	return false
}

// within reports whether p is root or inside it: a prefix that ends at a
// path separator, letter case folded when fold is set.
func within(p, root string, fold bool) bool {
	if fold {
		p, root = strings.ToLower(p), strings.ToLower(root)
	}
	if p == root {
		return true
	}
	if !strings.HasSuffix(root, string(filepath.Separator)) {
		root += string(filepath.Separator)
	}
	return strings.HasPrefix(p, root)
}

// accept returns the final path of candidate when it is a regular file
// (executable, on Unix) outside every root; a candidate whose final path
// cannot be had is skipped.
func (f *finder) accept(candidate string) (string, bool) {
	final, err := f.final(candidate)
	if err != nil {
		return "", false
	}
	fi, err := os.Stat(final) //nolint:gosec // G703: interpreter discovery checks resolved external candidate paths, as spec §5.3 and §19.8 require; this call only reads metadata.
	if err != nil || !fi.Mode().IsRegular() || !executable(fi) || f.excluded(final) {
		return "", false
	}
	return final, true
}

// onPath returns the first acceptable name in a PATH entry. Empty and
// relative entries, "." among them, are skipped (Review 071 N1): Go's
// exec.LookPath and its ErrDot are not this rule.
func (f *finder) onPath(name string) (string, bool) {
	for _, dir := range filepath.SplitList(f.path) {
		if dir == "" || !filepath.IsAbs(dir) {
			continue
		}
		if p, ok := f.accept(filepath.Join(dir, name)); ok {
			return p, true
		}
	}
	return "", false
}

// lookup returns shell's interpreter, or why there is none (a problem for
// shell_unavailable, without a path). A cancelled context is an error, and
// so is a handle of finalPath's own that did not confirm its release: an
// environment error, never a candidate skipped or a location unverified.
func (f *finder) lookup(ctx context.Context, shell string) (Interpreter, string, error) {
	if err := f.releaseFailure(); err != nil {
		return Interpreter{}, "", err
	}
	if f.unverified {
		return Interpreter{}, "where an interpreter is cannot be verified: the path of a worktree of the project cannot be resolved", nil
	}
	in, ok := f.find(ctx, shell)
	if err := f.releaseFailure(); err != nil {
		return Interpreter{}, "", err
	}
	if ok {
		return in, "", nil
	}
	if err := ctx.Err(); err != nil {
		return Interpreter{}, "", err
	}
	return Interpreter{}, "no " + shell + " that Specmetry may use is available here", nil
}

// releaseFailure is the lookup's failure once a close of finalPath's own
// did not confirm its release: environment (exit 6), its text controlled,
// the close's error in its chain.
func (f *finder) releaseFailure() error {
	if f.released == nil {
		return nil
	}
	return errors.Join(apperr.New(apperr.Environment,
		"a handle opened to resolve an interpreter's path closed with an error and is not confirmed released; no check was run"), f.released)
}

func shellOf(c baseline.Check) string {
	if c.Shell == nil {
		return "sh"
	}
	return *c.Shell
}

// interpreters is step 7 (spec §4.1, §5.3; D1): every shell the items use
// is looked up once, before anything runs. When any is unavailable the
// whole batch is refused, shell_unavailable at each item that needs one,
// without a path; otherwise each item gets its shell's interpreter.
func interpreters(ctx context.Context, items []Item, f *finder) error {
	found := map[string]Interpreter{}
	missing := map[string]string{}
	for _, it := range items {
		shell := shellOf(it.Check)
		if _, done := found[shell]; done || missing[shell] != "" {
			continue
		}
		in, problem, err := f.lookup(ctx, shell)
		if err != nil {
			return err
		}
		if problem != "" {
			missing[shell] = problem
			continue
		}
		found[shell] = in
	}
	if len(missing) > 0 {
		var details []apperr.Detail
		for _, it := range items {
			if p := missing[shellOf(it.Check)]; p != "" {
				details = append(details, apperr.Detail{Pointer: it.Criterion, Problem: "shell_unavailable: " + shellOf(it.Check) + ": " + p})
			}
		}
		shells := make([]string, 0, len(missing))
		for s := range missing {
			shells = append(shells, s)
		}
		sort.Strings(shells)
		return refusal(apperr.Environment, "an interpreter the checks need is not available ("+strings.Join(shells, ", ")+"); no check was run",
			details)
	}
	for i := range items {
		items[i].Interpreter = found[shellOf(items[i].Check)]
	}
	return nil
}

// gitExecPath is the exec path of the git gitx runs, in dir.
func gitExecPath(dir string) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) { return gitx.ExecPath(ctx, dir) }
}
