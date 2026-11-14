// Package cond implements the conditional difference checks of the
// implementation plan §12.10 (steps 1 to 5) over the controlled read:
// every candidate file is classified with go/build's Context.MatchFile
// under the analysis condition and under every production variant, with the
// target's ToolTags and ReleaseTags taken from the second call, and every
// byte go/build sees comes from the caller's read (method record §2.5).
package cond

import (
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// Kind is a package's kind for the checks.
type Kind int

// The kinds.
const (
	Main Kind = iota
	ThirdParty
	Std
	XSys
)

func (k Kind) String() string {
	return [...]string{"main", "third-party", "std", "x/sys"}[k]
}

// Source is how a package's files are listed and read; every read is a
// group C read.
type Source interface {
	// List returns the candidate file names directly in the directory.
	List() ([]string, error)
	// Read returns a file's bytes.
	Read(name string) ([]byte, error)
	// OpenFile is the Context.OpenFile for this package's directory.
	OpenFile(abs string) (io.ReadCloser, error)
	// ReadDir is the Context.ReadDir, or nil when the source has none.
	ReadDir(abs string) ([]fs.FileInfo, error)
}

// Pkg is one package to classify.
type Pkg struct {
	ImportPath string
	Dir        string // absolute
	Kind       Kind
	Listed     bool     // the third call reported it (step 1 applies)
	GoFiles    []string // as the third call reported (analysis condition)
	CgoFiles   []string
	Src        Source
}

// Condition is one build condition.
type Condition struct {
	Name       string // "analysis", "production-cgo0", "production-cgo1"
	CgoEnabled bool
	Tags       []string
}

// Conditions returns the analysis condition and the production variants.
func Conditions() []Condition {
	out := []Condition{{Name: "analysis", CgoEnabled: false, Tags: append([]string(nil), spec.AnalysisTags...)}}
	for _, cgo := range spec.ProductionCgo {
		n := "production-cgo0"
		if cgo {
			n = "production-cgo1"
		}
		out = append(out, Condition{Name: n, CgoEnabled: cgo})
	}
	return out
}

// Target is what the second call gave for the configuration.
type Target struct {
	Config      spec.Config
	Compiler    string
	ToolTags    []string
	ReleaseTags []string
	ContextSHA  string // SHA-256 of the second call's standard output
}

// ContextRecord is the derived record of a Context handed to MatchFile
// (plan §4.2 "实际的消费者").
type ContextRecord struct {
	Condition   string
	GOOS        string
	GOARCH      string
	CgoEnabled  bool
	BuildTags   []string
	ToolTags    []string
	ReleaseTags []string
	Compiler    string
	ContextSHA  string
}

// NewContext builds the one Context used for a condition and source: the
// same value is recorded and handed to MatchFile.
func NewContext(t Target, c Condition, src Source) (*build.Context, ContextRecord) {
	ctx := &build.Context{
		GOOS:        t.Config.GOOS,
		GOARCH:      t.Config.GOARCH,
		CgoEnabled:  c.CgoEnabled,
		BuildTags:   append([]string(nil), c.Tags...),
		ToolTags:    append([]string(nil), t.ToolTags...),
		ReleaseTags: append([]string(nil), t.ReleaseTags...),
		Compiler:    t.Compiler,
		OpenFile:    src.OpenFile,
		ReadDir:     src.ReadDir,
	}
	if ctx.ReadDir == nil {
		ctx.ReadDir = func(string) ([]fs.FileInfo, error) {
			return nil, fmt.Errorf("cond: ReadDir is not part of this use")
		}
	}
	rec := ContextRecord{
		Condition: c.Name, GOOS: ctx.GOOS, GOARCH: ctx.GOARCH, CgoEnabled: ctx.CgoEnabled,
		BuildTags: append([]string(nil), ctx.BuildTags...), ToolTags: append([]string(nil), ctx.ToolTags...),
		ReleaseTags: append([]string(nil), ctx.ReleaseTags...), Compiler: ctx.Compiler, ContextSHA: t.ContextSHA,
	}
	return ctx, rec
}

// Classes is one classification of a directory.
type Classes struct {
	GoFiles  []string
	CgoFiles []string
	Imports  map[string][]string // file -> imports, for matched files
	// Judged are the candidates handed to MatchFile, for I-6 (c): no
	// candidate is skipped (method record V-11).
	Judged []string
}

// IsCandidate reports whether go/build's Import looks at a file name
// (go/build/build.go:948, :985-1020): .go, not _test.go, not starting with
// _ or . .
func IsCandidate(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") &&
		!strings.HasPrefix(name, "_") && !strings.HasPrefix(name, ".")
}

// Classify classifies the candidates of a package under one condition: a
// file go/build matches and that imports "C" goes to CgoFiles when cgo is
// enabled and is ignored otherwise; the rest go to GoFiles. Every failure
// (a read failure, a MatchFile error, an import parse error) is returned;
// the classification is then incomplete and must not be used.
func Classify(ctx *build.Context, p *Pkg, names []string) (Classes, []judge.Failure) {
	var fails []judge.Failure
	cl := Classes{Imports: map[string][]string{}}
	for _, name := range names {
		if !IsCandidate(name) {
			continue
		}
		site := fmt.Sprintf("classify %s %s", p.ImportPath, name)
		cl.Judged = append(cl.Judged, name)
		match, err := ctx.MatchFile(p.Dir, name)
		if err != nil {
			fails = append(fails, readOrMatchFailure(err, name, site))
			continue
		}
		if !match {
			continue
		}
		b, err := p.Src.Read(name)
		if err != nil {
			fails = append(fails, readOrMatchFailure(err, name, site))
			continue
		}
		imps, err := Imports(b)
		if err != nil {
			fails = append(fails, judge.Real(judge.J3, name, site, "parse imports", err.Error()))
			continue
		}
		cl.Imports[name] = imps
		if contains(imps, "C") {
			if ctx.CgoEnabled {
				cl.CgoFiles = append(cl.CgoFiles, name)
			}
			continue
		}
		cl.GoFiles = append(cl.GoFiles, name)
	}
	sort.Strings(cl.GoFiles)
	sort.Strings(cl.CgoFiles)
	return cl, fails
}

func readOrMatchFailure(err error, name, site string) judge.Failure {
	if f, ok := modsnap.FailureOf(err); ok {
		return f
	}
	return judge.Real(judge.J3, name, site, "match error", err.Error())
}

// Imports parses a file's import paths in source order.
func Imports(b []byte) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "", b, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(f.Imports))
	for _, s := range f.Imports {
		p, err := strconv.Unquote(s.Path.Value)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// IsStd reports whether an import path is a standard library path by
// cmd/go's rule: the first element has no dot (this covers "vendor/...").
func IsStd(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return path != "C" && !strings.Contains(first, ".")
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func equal(a, b []string) bool {
	a, b = sorted(a), sorted(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sorted(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}
