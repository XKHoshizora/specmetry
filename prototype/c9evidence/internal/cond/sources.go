package cond

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
)

// ModuleSource lists and reads a package of a module whose I-4 holds,
// through the controlled read: the names come from H (no live listing) and
// every byte from I-5 (method record §2.3, group C).
type ModuleSource struct {
	R      *modsnap.Reader
	M      *modsnap.Snapshot
	RelDir string // the package directory inside the module, slash-separated
}

// List returns the regular file names H recorded in the directory.
func (s ModuleSource) List() ([]string, error) {
	names, ok := s.M.DirNames(s.RelDir)
	if !ok {
		f := judge.Real(judge.J3, s.RelDir, "list "+s.M.Prefix()+"/"+s.RelDir, "directory not in H", "")
		return nil, &modsnap.ReadError{Failure: f}
	}
	return names, nil
}

// Read is the controlled read of a file of the package.
func (s ModuleSource) Read(name string) ([]byte, error) {
	return s.R.Read(s.M, joinRel(s.RelDir, name), modsnap.GroupC, "classify")
}

// OpenFile is the controlled read for go/build.
func (s ModuleSource) OpenFile(abs string) (io.ReadCloser, error) {
	return s.R.OpenFile(modsnap.GroupC, "classify")(abs)
}

// ReadDir is the controlled view for go/build.
func (s ModuleSource) ReadDir(abs string) ([]fs.FileInfo, error) { return s.R.ReadDir(abs) }

// PlainSource lists and reads a directory outside the go.sum chain (the
// repository checkout or the standard library) with the plain recorded
// read. go/build may open only files directly in Dir.
type PlainSource struct {
	P      *modsnap.Plain
	Dir    string // absolute
	RelDir string // for records: repository-relative or the import path
}

// List returns the regular file names in the directory.
func (s PlainSource) List() ([]string, error) { return s.P.List(s.Dir, s.RelDir) }

// Read reads a file of the directory.
func (s PlainSource) Read(name string) ([]byte, error) {
	return s.P.Read(filepath.Join(s.Dir, name), joinRel(s.RelDir, name), modsnap.GroupC, "classify")
}

// OpenFile reads a file of the directory for go/build; any other path is
// refused (J-3), never opened.
func (s PlainSource) OpenFile(abs string) (io.ReadCloser, error) {
	rel, err := filepath.Rel(s.Dir, abs)
	if err != nil || strings.ContainsAny(filepath.ToSlash(rel), "/") || rel == "." || rel == ".." {
		f := judge.Real(judge.J3, abs, "OpenFile "+s.RelDir, "path outside the package directory", "")
		return nil, &modsnap.ReadError{Failure: f}
	}
	b, err := s.Read(rel)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// ReadDir is not part of this use for plain sources.
func (s PlainSource) ReadDir(string) ([]fs.FileInfo, error) { return nil, errNoReadDir }

var errNoReadDir = &modsnap.ReadError{Failure: judge.Real(judge.J3, "ReadDir", "ReadDir", "ReadDir is not part of this use", "")}

// RepoDirs walks the repository root and returns its production
// directories (implementation plan §12.10 step 2): directories of the
// main module that hold a candidate .go file, outside internal/testutil and
// any testdata, skipping names starting with . or _, vendor, and nested
// modules (a directory with its own go.mod, such as prototype/wp25 and
// this tool's module). Paths are slash-separated and relative ("" is the
// root). The walk does not follow symbolic links.
func RepoDirs(fsys modsnap.FS, root string, skip []string) ([]string, []judge.Failure) {
	var out []string
	var fails []judge.Failure
	skipSet := map[string]bool{}
	for _, s := range skip {
		skipSet[s] = true
	}
	var walk func(abs, rel string)
	walk = func(abs, rel string) {
		ents, err := fsys.ReadDir(abs)
		if err != nil {
			fails = append(fails, judge.Real(judge.J1, relName(rel), "walk repository", "readdir error", err.Error()))
			return
		}
		has := false
		for _, e := range ents {
			if e.Type().IsRegular() && IsCandidate(e.Name()) {
				has = true
			}
		}
		if has {
			out = append(out, rel)
		}
		for _, e := range ents {
			if !e.IsDir() || e.Type()&fs.ModeSymlink != 0 {
				continue
			}
			n := e.Name()
			if strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_") || n == "testdata" || n == "vendor" {
				continue
			}
			childRel := joinRel(rel, n)
			if skipSet[childRel] {
				continue
			}
			childAbs := filepath.Join(abs, n)
			if _, err := fsys.Lstat(filepath.Join(childAbs, "go.mod")); err == nil {
				continue // a nested module
			} else if !errors.Is(err, fs.ErrNotExist) {
				fails = append(fails, judge.Real(judge.J1, childRel, "walk repository", "lstat error", err.Error()))
				continue
			}
			walk(childAbs, childRel)
		}
	}
	walk(root, "")
	return out, fails
}
