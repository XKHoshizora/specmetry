package decode

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// EmptyGoModHash is cmd/go's hash of a one-file tree holding an empty
// go.mod (go1.27.3 cmd/go/internal/modfetch/fetch.go:694-697). cmd/go drops
// such lines (:725-728); I-1 fails on them (method record §2.4 row I-1).
const EmptyGoModHash = "h1:G7mAYYxgmS0lVkHyy2hEOLQCFB0DlQFTMLWggykrydY="

// SumKey is the key of a go.sum line: the module path, the version without
// a "/go.mod" suffix, and whether the line had that suffix.
type SumKey struct {
	Path    string
	Version string
	GoMod   bool
}

func (k SumKey) String() string {
	if k.GoMod {
		return k.Path + " " + k.Version + "/go.mod"
	}
	return k.Path + " " + k.Version
}

// SumProblemKind is the kind of a go.sum problem. Every problem fails I-1
// (J-2 at the caller, method record §3.5 DS-5).
type SumProblemKind string

// The problem kinds of I-1.
const (
	// SumFields: a non-empty line does not have exactly three fields.
	SumFields SumProblemKind = "fields"
	// SumHashPrefix: the hash does not begin with "h1:".
	SumHashPrefix SumProblemKind = "hash-prefix"
	// SumEmptyGoMod: the hash is EmptyGoModHash.
	SumEmptyGoMod SumProblemKind = "empty-gomod-hash"
	// SumConflict: the key already has a different h1 on an earlier line.
	SumConflict SumProblemKind = "conflict"
)

// SumProblem is one problem of one line. Line is 1-based and counted as
// cmd/go counts it (fetch.go:701-713).
type SumProblem struct {
	Line   int
	Kind   SumProblemKind
	Detail string
}

func (p SumProblem) String() string {
	return fmt.Sprintf("go.sum:%d: %s: %s", p.Line, p.Kind, p.Detail)
}

// SumDuplicate records a line whose three fields equal those of an earlier
// line: it is recorded, not a conflict (I-1).
type SumDuplicate struct {
	Line  int // the repeated line
	First int // the earliest line with the same fields
	Key   SumKey
}

// SumEntry is the state of one key over all lines that passed the per-line
// checks. Hashes are the distinct hashes in order of first appearance, Lines
// every line of the key. Usable is true when the key has exactly one hash and
// no line of the key had a problem; only then do H1 and GoModH1 give it.
type SumEntry struct {
	Key    SumKey
	Hashes []string
	Lines  []int
	Usable bool
}

type sumEntry struct {
	hashes []string
	first  map[string]int // hash -> first line
	lines  []int
	bad    bool // a line of this key had a problem (hash-prefix, empty-gomod-hash)
}

// GoSum is the parsed R/go.sum (I-1). Size and SHA256 identify the bytes
// parsed; Lines is the number of lines as cmd/go counts them. The table is
// unexported so a conflicting or problematic key cannot be read by mistake:
// use H1, GoModH1 or Entries.
type GoSum struct {
	Size       int
	SHA256     string
	Lines      int
	Duplicates []SumDuplicate
	entries    map[SumKey]*sumEntry
}

// ParseGoSum parses go.sum bytes strictly (method record §2.4 row I-1, §2.2
// item 9). Lines are split on LF and fields with strings.Fields, as cmd/go
// does (go1.27.3 cmd/go/internal/modfetch/fetch.go:701-731); lines with no
// fields are skipped. Every non-empty line is evaluated and every problem
// kept, in line order; parsing never stops at the first problem:
//
//   - a line without exactly three fields is SumFields;
//   - the third field must begin with "h1:" (SumHashPrefix) and must not be
//     EmptyGoModHash (SumEmptyGoMod);
//   - a line passing those checks whose key already has a different hash on
//     an earlier line is SumConflict;
//   - a line whose three fields equal an earlier line's is recorded in
//     Duplicates and is not a problem.
//
// CR: strings.Fields treats '\r' as white space (unicode.IsSpace), exactly
// as cmd/go does, so a CRLF go.sum parses to the same fields as its LF form
// and a CR is not a problem. Comparing fields rather than line bytes also
// means repeated lines that differ only in blanks count as identical.
//
// A key is usable only when every line of it passed the per-line checks and
// they all carry one hash: a conflicting key, or one that also appears on a
// hash-prefix or empty-gomod-hash line, has no lookup result.
func ParseGoSum(b []byte) (GoSum, []SumProblem) {
	sum := sha256.Sum256(b)
	g := GoSum{Size: len(b), SHA256: hex.EncodeToString(sum[:]), entries: map[SumKey]*sumEntry{}}
	var problems []SumProblem
	data := b
	lineno := 0
	for len(data) > 0 {
		var line []byte
		lineno++
		if i := bytes.IndexByte(data, '\n'); i < 0 {
			line, data = data, nil
		} else {
			line, data = data[:i], data[i+1:]
		}
		f := strings.Fields(string(line))
		if len(f) == 0 {
			continue
		}
		if len(f) != 3 {
			problems = append(problems, SumProblem{lineno, SumFields, fmt.Sprintf("%d fields, want 3", len(f))})
			continue
		}
		key := SumKey{Path: f[0], Version: f[1]}
		if v, ok := strings.CutSuffix(f[1], "/go.mod"); ok {
			key.Version, key.GoMod = v, true
		}
		e := g.entries[key]
		if e == nil {
			e = &sumEntry{first: map[string]int{}}
			g.entries[key] = e
		}
		hash := f[2]
		switch {
		case !strings.HasPrefix(hash, "h1:"):
			e.bad = true
			problems = append(problems, SumProblem{lineno, SumHashPrefix, fmt.Sprintf("%s: hash %q does not begin with \"h1:\"", key, hash)})
			continue
		case hash == EmptyGoModHash:
			e.bad = true
			problems = append(problems, SumProblem{lineno, SumEmptyGoMod, fmt.Sprintf("%s: hash is the empty go.mod hash", key)})
			continue
		}
		e.lines = append(e.lines, lineno)
		if first, ok := e.first[hash]; ok {
			g.Duplicates = append(g.Duplicates, SumDuplicate{Line: lineno, First: first, Key: key})
			continue
		}
		if len(e.hashes) > 0 {
			problems = append(problems, SumProblem{lineno, SumConflict, fmt.Sprintf("%s: hash %s differs from %s on line %d", key, hash, e.hashes[0], e.first[e.hashes[0]])})
		}
		e.first[hash] = lineno
		e.hashes = append(e.hashes, hash)
	}
	g.Lines = lineno
	// Keys seen only on problem lines have no hash and are not entries.
	for k, e := range g.entries {
		if len(e.hashes) == 0 {
			delete(g.entries, k)
		}
	}
	return g, problems
}

func (g GoSum) lookup(k SumKey) (string, bool) {
	e := g.entries[k]
	if e == nil || e.bad || len(e.hashes) != 1 {
		return "", false
	}
	return e.hashes[0], true
}

// H1 returns the recorded h1 of the module content (path, version), the
// value Module.Sum must equal (I-2). ok is false when there is no usable
// record.
func (g GoSum) H1(path, version string) (string, bool) {
	return g.lookup(SumKey{Path: path, Version: version})
}

// GoModH1 returns the recorded h1 of (path, version/go.mod), the value
// Module.GoModSum must equal (I-2).
func (g GoSum) GoModH1(path, version string) (string, bool) {
	return g.lookup(SumKey{Path: path, Version: version, GoMod: true})
}

// Entries returns every key that has at least one hash, sorted by path,
// version and GoMod (false first), with copies of its hashes and lines.
func (g GoSum) Entries() []SumEntry {
	out := make([]SumEntry, 0, len(g.entries))
	for k, e := range g.entries {
		out = append(out, SumEntry{
			Key:    k,
			Hashes: append([]string(nil), e.hashes...),
			Lines:  append([]int(nil), e.lines...),
			Usable: !e.bad && len(e.hashes) == 1,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].Key, out[j].Key
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		return !a.GoMod && b.GoMod
	})
	return out
}
