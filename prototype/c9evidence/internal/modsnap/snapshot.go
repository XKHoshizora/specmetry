package modsnap

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
)

// FileRecord is one entry of a module's M: the relative name, the name
// with the Hash1 prefix, the bytes read, their SHA-256, and the real Close
// result of that one read (method record §2.4 I-4).
type FileRecord struct {
	Rel      string // slash-separated, relative to the module root
	Name     string // Path@Version/Rel, the Hash1 name
	Size     int64
	SHA256   string
	CloseErr string // "" when the real Close returned nil
}

// Snapshot is the result of I-4 for one module: M, the computed h1 ("" when
// it was not computed), and every failure. Failures use the classes of
// DS-7: walk, open, read and close errors J-1; a non-regular entry or
// handle J-2; an h1 different from go.sum J-4. A name containing LF is J-2
// here (the method record says only that it fails; the tool classes it as a
// source anomaly, see the implementation record).
type Snapshot struct {
	Path     string // module path
	Version  string
	Dir      string // Module.Dir as reported
	Files    []FileRecord
	byRel    map[string]FileRecord
	dirs     map[string][]string // directory (slash rel, "" for the root) -> regular file names directly in it
	H1       string
	WantH1   string
	Failures []judge.Failure
}

// Prefix returns the Hash1 name prefix: the unescaped path@version.
func (s *Snapshot) Prefix() string { return s.Path + "@" + s.Version }

// File returns the record of a relative name.
func (s *Snapshot) File(rel string) (FileRecord, bool) {
	r, ok := s.byRel[rel]
	return r, ok
}

// Holds reports whether I-4 holds for this module.
func (s *Snapshot) Holds() bool { return len(s.Failures) == 0 && s.H1 != "" && s.H1 == s.WantH1 }

// Take runs I-4 on one module: walk Dir without following symbolic links,
// take every entry's type with Lstat semantics, require every non-directory
// entry to be a regular file, open each file once, check the handle's type,
// read to EOF, close once and keep the real result; then compute h1 with
// dirhash.Hash1 over the bytes already read and compare it with wantH1,
// the go.sum record of I-1. Every input that can be evaluated is evaluated:
// the walk and the reads continue after a failure; only h1 is skipped once
// a type failure or a read failure makes it meaningless.
func Take(fsys FS, modPath, version, dir, wantH1 string) *Snapshot {
	s := &Snapshot{Path: modPath, Version: version, Dir: dir, WantH1: wantH1, byRel: map[string]FileRecord{}, dirs: map[string][]string{}}
	site := "I-4 " + s.Prefix()
	fail := func(c judge.Class, item, kind, detail string) {
		s.Failures = append(s.Failures, judge.Real(c, item, site, kind, detail))
	}

	fi, err := fsys.Lstat(dir)
	switch {
	case err != nil:
		fail(judge.J1, ".", "lstat error", err.Error())
		return s
	case !fi.IsDir():
		fail(judge.J2, ".", "not a directory", fi.Mode().Type().String())
		return s
	}

	data := map[string][]byte{}
	var names []string
	hashable := true
	var walk func(abs, rel string)
	walk = func(abs, rel string) {
		ents, err := fsys.ReadDir(abs)
		if err != nil {
			fail(judge.J1, relOrRoot(rel), "readdir error", err.Error())
			hashable = false
			return
		}
		if _, ok := s.dirs[rel]; !ok {
			s.dirs[rel] = nil // the directory is recorded even with no regular file in it
		}
		sort.Slice(ents, func(i, j int) bool { return ents[i].Name() < ents[j].Name() })
		for _, e := range ents {
			childRel := e.Name()
			if rel != "" {
				childRel = rel + "/" + e.Name()
			}
			childAbs := filepath.Join(abs, e.Name())
			li, err := fsys.Lstat(childAbs)
			if err != nil {
				fail(judge.J1, childRel, "lstat error", err.Error())
				hashable = false
				continue
			}
			switch {
			case li.IsDir():
				walk(childAbs, childRel)
				continue
			case !li.Mode().IsRegular():
				fail(judge.J2, childRel, "not a regular file", li.Mode().Type().String())
				hashable = false
				continue
			}
			name := s.Prefix() + "/" + childRel
			if strings.Contains(name, "\n") {
				fail(judge.J2, childRel, "name contains LF", "")
				hashable = false
				continue
			}
			b, regular, openErr, statErr, readErr, closeErr := readOnce(fsys, childAbs)
			rec := FileRecord{Rel: childRel, Name: name}
			switch {
			case openErr != nil:
				fail(judge.J1, childRel, "open error", openErr.Error())
				hashable = false
				continue
			case statErr != nil:
				fail(judge.J1, childRel, "stat error", statErr.Error())
				hashable = false
			case !regular:
				fail(judge.J2, childRel, "handle not a regular file", "")
				hashable = false
			case readErr != nil:
				fail(judge.J1, childRel, "read error", readErr.Error())
				hashable = false
			}
			if closeErr != nil {
				rec.CloseErr = closeErr.Error()
				fail(judge.J1, childRel, "close error", closeErr.Error())
			}
			if regular && readErr == nil && statErr == nil {
				sum := sha256.Sum256(b)
				rec.Size = int64(len(b))
				rec.SHA256 = hex.EncodeToString(sum[:])
				data[name] = b
				names = append(names, name)
				s.byRel[childRel] = rec
				s.dirs[rel] = append(s.dirs[rel], e.Name())
			}
			s.Files = append(s.Files, rec)
		}
	}
	walk(dir, "")

	if hashable {
		h, err := hash1(names, data)
		if err != nil {
			fail(judge.J2, ".", "hash1 error", err.Error())
		} else {
			s.H1 = h
			if h != wantH1 {
				fail(judge.J4, ".", "h1 differs from go.sum", fmt.Sprintf("computed %s, go.sum %s", h, wantH1))
			}
		}
	}
	return s
}

func relOrRoot(rel string) string {
	if rel == "" {
		return "."
	}
	return rel
}

// DirNames returns the regular file names recorded directly in a directory
// of the module (rel is slash-separated, "" for the root), sorted.
func (s *Snapshot) DirNames(rel string) ([]string, bool) {
	rel = strings.TrimSuffix(path.Clean("/" + rel)[1:], "/")
	names, ok := s.dirs[rel]
	if !ok {
		return nil, false
	}
	out := append([]string(nil), names...)
	sort.Strings(out)
	return out, true
}
