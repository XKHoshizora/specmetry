package modsnap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
)

// Group is a read's group (method record §2.3): C for the conditional
// classification and import reads, P for the final prefilter, parse and
// type analysis.
type Group string

// The groups.
const (
	GroupC Group = "C"
	GroupP Group = "P"
)

// Consumption records one handed-out read: which file, for which group and
// purpose, and the SHA-256 of the bytes handed out (method record §2.5).
// Origin is the module (path@version), "repo" or "std".
type Consumption struct {
	Origin  string
	Rel     string
	Group   Group
	Purpose string
	Size    int64
	SHA256  string
}

// Injected is an error a test seam appended after a real Close returned
// nil. It is kept apart from the real result (implementation plan §12.14
// item 12; method record V-29).
type Injected struct {
	ID   int
	Site string
	Err  string
}

// Injector is the seam: asked after a real Close returned nil, it may
// append an error with its serial. It is never asked when the real Close
// failed, so it can neither replace nor hide a real error.
type Injector func(site string) (id int, err error)

// ReadError is a failed controlled or plain read; it carries the failure
// the caller records.
type ReadError struct{ Failure judge.Failure }

func (e *ReadError) Error() string { return e.Failure.String() }

// FailureOf returns the failure of a read error.
func FailureOf(err error) (judge.Failure, bool) {
	var re *ReadError
	if errors.As(err, &re) {
		return re.Failure, true
	}
	return judge.Failure{}, false
}

// Reader is the controlled read over the modules whose I-4 holds. All of
// groups C and P go through it: a name must be in M; the bytes are reread
// (open once, the handle must be regular, read to EOF, close once, the
// real Close result kept) and their SHA-256 must equal M's; the first
// bytes handed out for a name are kept and handed out again, so C and P
// consume the same B (method record §2.5, implementation 2).
type Reader struct {
	mu       sync.Mutex
	fsys     FS
	mods     []*Snapshot
	cache    map[string][]byte
	consumed []Consumption
	failures []judge.Failure
	injected []Injected
	inject   Injector
}

// NewReader makes a controlled read over the verified modules.
func NewReader(fsys FS, verified []*Snapshot, inject Injector) *Reader {
	return &Reader{fsys: fsys, mods: append([]*Snapshot(nil), verified...), cache: map[string][]byte{}, inject: inject}
}

func (r *Reader) fail(f judge.Failure) error {
	r.mu.Lock()
	r.failures = append(r.failures, f)
	r.mu.Unlock()
	return &ReadError{Failure: f}
}

// Read hands out the bytes of rel in module m.
func (r *Reader) Read(m *Snapshot, rel string, g Group, purpose string) ([]byte, error) {
	site := fmt.Sprintf("I-5 %s/%s (%s %s)", m.Prefix(), rel, g, purpose)
	rec, ok := m.File(rel)
	if !ok {
		return nil, r.fail(judge.Real(judge.J4, rel, site, "name not in M", ""))
	}
	r.mu.Lock()
	b, cached := r.cache[rec.Name]
	r.mu.Unlock()
	if !cached {
		abs := filepath.Join(m.Dir, filepath.FromSlash(rel))
		nb, regular, openErr, statErr, readErr, closeErr := readOnce(r.fsys, abs)
		var first *judge.Failure
		switch {
		case openErr != nil:
			return nil, r.fail(judge.Real(judge.J1, rel, site, "open error", openErr.Error()))
		case statErr != nil:
			x := judge.Real(judge.J1, rel, site, "stat error", statErr.Error())
			first = &x
		case !regular:
			x := judge.Real(judge.J2, rel, site, "handle not a regular file", "")
			first = &x
		case readErr != nil:
			x := judge.Real(judge.J1, rel, site, "read error", readErr.Error())
			first = &x
		}
		if closeErr != nil {
			cf := judge.Real(judge.J1, rel, site, "close error", closeErr.Error())
			if first == nil {
				return nil, r.fail(cf)
			}
			_ = r.fail(cf) // a second, separate real failure of the same read
		}
		if first != nil {
			return nil, r.fail(*first)
		}
		if r.inject != nil {
			if id, err := r.inject(site); err != nil {
				r.mu.Lock()
				r.injected = append(r.injected, Injected{ID: id, Site: site, Err: err.Error()})
				r.mu.Unlock()
				return nil, r.fail(judge.Failure{Class: judge.J1, Item: rel, Site: site, Kind: "close error", Source: judge.SourceInjected, Inject: id, Detail: err.Error()})
			}
		}
		sum := sha256.Sum256(nb)
		if got := hex.EncodeToString(sum[:]); got != rec.SHA256 {
			return nil, r.fail(judge.Real(judge.J4, rel, site, "sha256 differs from M", fmt.Sprintf("read %s, M %s", got, rec.SHA256)))
		}
		r.mu.Lock()
		if prev, ok := r.cache[rec.Name]; ok {
			nb = prev // another request won: hand out the first bytes
		} else {
			r.cache[rec.Name] = nb
		}
		r.mu.Unlock()
		b = nb
	}
	r.mu.Lock()
	r.consumed = append(r.consumed, Consumption{Origin: m.Prefix(), Rel: rel, Group: g, Purpose: purpose, Size: int64(len(b)), SHA256: rec.SHA256})
	r.mu.Unlock()
	return b, nil
}

// Locate maps an absolute path to a verified module and the slash-relative
// name inside it, lexically (no symbolic link is resolved).
func (r *Reader) Locate(abs string) (*Snapshot, string, bool) {
	var best *Snapshot
	var bestRel string
	for _, m := range r.mods {
		rel, err := filepath.Rel(m.Dir, abs)
		if err != nil || filepath.IsAbs(rel) {
			continue
		}
		up := false
		for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
			if part == ".." {
				up = true
				break
			}
		}
		if up {
			continue
		}
		if best == nil || len(m.Dir) > len(best.Dir) {
			best, bestRel = m, filepath.ToSlash(rel)
		}
	}
	if best == nil {
		return nil, "", false
	}
	if bestRel == "." {
		bestRel = ""
	}
	return best, bestRel, true
}

// OpenFile is go/build's Context.OpenFile for a group and purpose: the
// controlled read; a path outside every verified module is J-3. The
// returned Close has no real resource.
func (r *Reader) OpenFile(g Group, purpose string) func(string) (io.ReadCloser, error) {
	return func(abs string) (io.ReadCloser, error) {
		m, rel, ok := r.Locate(abs)
		if !ok {
			return nil, r.fail(judge.Real(judge.J3, abs, "I-5 OpenFile", "path outside the verified modules", ""))
		}
		b, err := r.Read(m, rel, g, purpose)
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bytes.NewReader(b)), nil
	}
}

// ReadDir is go/build's Context.ReadDir: the view of the current use, not a
// live listing (method record §2.5, N-141-1). It hands out the regular file
// names recorded by H directly in that directory, sorted by name, with the
// size I-4 read; no mode bits, a zero modification time and no Sys. A
// directory outside every verified module is J-3; nothing falls back to
// os.ReadDir.
func (r *Reader) ReadDir(abs string) ([]fs.FileInfo, error) {
	m, rel, ok := r.Locate(abs)
	if !ok {
		return nil, r.fail(judge.Real(judge.J3, abs, "I-5 ReadDir", "directory outside the verified modules", ""))
	}
	names, ok := m.DirNames(rel)
	if !ok {
		return nil, r.fail(judge.Real(judge.J3, abs, "I-5 ReadDir", "directory not in H", ""))
	}
	out := make([]fs.FileInfo, 0, len(names))
	for _, n := range names {
		p := n
		if rel != "" {
			p = rel + "/" + n
		}
		rec, _ := m.File(p)
		out = append(out, viewInfo{name: n, size: rec.Size})
	}
	return out, nil
}

// Consumed returns the consumption records.
func (r *Reader) Consumed() []Consumption {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Consumption(nil), r.consumed...)
}

// Failures returns every failure of the controlled read.
func (r *Reader) Failures() []judge.Failure {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]judge.Failure(nil), r.failures...)
}

// Injections returns the injected errors, kept apart from real results.
func (r *Reader) Injections() []Injected {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Injected(nil), r.injected...)
}

// WasConsumed reports whether rel of module m was handed out for group g.
func (r *Reader) WasConsumed(m *Snapshot, rel string, g Group) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.consumed {
		if c.Origin == m.Prefix() && c.Rel == rel && c.Group == g {
			return true
		}
	}
	return false
}

// viewInfo is a FileInfo of the ReadDir view: every field comes from H.
type viewInfo struct {
	name string
	size int64
}

func (v viewInfo) Name() string       { return v.name }
func (v viewInfo) Size() int64        { return v.size }
func (v viewInfo) Mode() fs.FileMode  { return 0 }
func (v viewInfo) ModTime() time.Time { return time.Time{} }
func (v viewInfo) IsDir() bool        { return false }
func (v viewInfo) Sys() any           { return nil }

// Plain is the recorded read of files outside the go.sum chain: the
// repository checkout and the standard library (method record §2.3: no
// content check, G4-Q3). It keeps the same discipline: open once, the
// handle must be regular, read to EOF, close once, the real Close result
// kept; the first bytes of a path are handed out again.
type Plain struct {
	mu       sync.Mutex
	fsys     FS
	origin   string
	cache    map[string][]byte
	consumed []Consumption
	failures []judge.Failure
}

// NewPlain makes a plain recorded read; origin is "repo" or "std".
func NewPlain(fsys FS, origin string) *Plain {
	return &Plain{fsys: fsys, origin: origin, cache: map[string][]byte{}}
}

func (p *Plain) fail(f judge.Failure) error {
	p.mu.Lock()
	p.failures = append(p.failures, f)
	p.mu.Unlock()
	return &ReadError{Failure: f}
}

// Read hands out the bytes of an absolute path; rel names it in records.
func (p *Plain) Read(abs, rel string, g Group, purpose string) ([]byte, error) {
	site := fmt.Sprintf("read %s %s (%s %s)", p.origin, rel, g, purpose)
	p.mu.Lock()
	b, ok := p.cache[abs]
	p.mu.Unlock()
	if !ok {
		nb, regular, openErr, statErr, readErr, closeErr := readOnce(p.fsys, abs)
		var f *judge.Failure
		switch {
		case openErr != nil:
			x := judge.Real(judge.J1, rel, site, "open error", openErr.Error())
			f = &x
		case statErr != nil:
			x := judge.Real(judge.J1, rel, site, "stat error", statErr.Error())
			f = &x
		case !regular:
			x := judge.Real(judge.J2, rel, site, "handle not a regular file", "")
			f = &x
		case readErr != nil:
			x := judge.Real(judge.J1, rel, site, "read error", readErr.Error())
			f = &x
		}
		if closeErr != nil {
			cf := judge.Real(judge.J1, rel, site, "close error", closeErr.Error())
			if f == nil {
				f = &cf
			} else {
				p.mu.Lock()
				p.failures = append(p.failures, cf)
				p.mu.Unlock()
			}
		}
		if f != nil {
			return nil, p.fail(*f)
		}
		p.mu.Lock()
		if prev, ok := p.cache[abs]; ok {
			nb = prev
		} else {
			p.cache[abs] = nb
		}
		p.mu.Unlock()
		b = nb
	}
	sum := sha256.Sum256(b)
	p.mu.Lock()
	p.consumed = append(p.consumed, Consumption{Origin: p.origin, Rel: rel, Group: g, Purpose: purpose, Size: int64(len(b)), SHA256: hex.EncodeToString(sum[:])})
	p.mu.Unlock()
	return b, nil
}

// List returns the regular file names directly in an absolute directory,
// sorted, by a real listing with Lstat semantics. Entries that are not
// regular files are skipped (they cannot be source files of a package).
func (p *Plain) List(abs, rel string) ([]string, error) {
	ents, err := p.fsys.ReadDir(abs)
	if err != nil {
		return nil, p.fail(judge.Real(judge.J1, rel, "list "+p.origin+" "+rel, "readdir error", err.Error()))
	}
	var out []string
	for _, e := range ents {
		if e.Type().IsRegular() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// Consumed returns the consumption records.
func (p *Plain) Consumed() []Consumption {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]Consumption(nil), p.consumed...)
}

// Failures returns every failure.
func (p *Plain) Failures() []judge.Failure {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]judge.Failure(nil), p.failures...)
}
