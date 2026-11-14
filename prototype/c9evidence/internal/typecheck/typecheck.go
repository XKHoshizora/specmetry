// Package typecheck is the loading, type checking and third-party object
// scan of one configuration (implementation plan §12.9 and §12.10 "装载"
// steps 3 to 9; method record §2.4 I-6 (d) and I-7; limited evidence plan
// §4.3, §4.4 and LC-7).
//
// The caller gives the configuration, the toolchain version, the package
// table of the third call, a byte source (the controlled read of method
// record §2.5) and an admission function (§12.14). The package never opens a
// file, never runs a program and never asks go/build, go/importer or the go
// command for anything: every package's metadata comes from the table and
// every byte from the source.
//
// What it does, per configuration and in one pass:
//
//   - orders the non-main packages by their Imports and finds cycles;
//   - reads, prefilters (third-party only), parses and type-checks each of
//     them in that order: "unsafe" is never parsed and imports as
//     types.Unsafe; the standard library and golang.org/x/sys are checked
//     for declarations only; a third-party package is checked in full only
//     when the lexical prefilter hits, otherwise for declarations only;
//   - for third-party and golang.org/x/sys packages, checks I-6 (d) (the
//     parsed imports, mapped through ImportMap, equal Imports as a set) and
//     derives each file's I-7 constraint line;
//   - scans every fully checked third-party package for references to the
//     object table: frozen objects give findings, X1 and X4 objects give
//     diagnostics, and every reference to an os/signal object gets the
//     signal-argument classification (diagnostic only: limited evidence plan
//     §2.3 and §4.4; the rule of §12.9 is not applied as a judgement here);
//   - runs the table self-check of §12.9.
//
// The repository's own packages (Kind Main) are not type-checked: third-party
// code does not import them, and the limited evidence tool does not run the
// repository rules (limited evidence plan §4.3).
//
// Results follow method record §6.1: a check holds, fails with judge
// failures, or is not evaluated with a reason. A package whose import did
// not load is not evaluated ("import incomplete"), not failed a second time.
// Every ordering in the report is deterministic.
package typecheck

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"go/version"
	"sort"
	"strconv"
	"strings"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// Kind is a package's place in the analysis (implementation plan §12.10,
// "范围" and loading step 9).
type Kind int

// The kinds. The caller assigns them; Run checks them against the record
// (Standard; Module.Main; the golang.org/x/sys path).
const (
	Std        Kind = iota + 1 // standard library (Standard): declarations only
	XSys                       // golang.org/x/sys/...: declarations only
	ThirdParty                 // every other dependency: prefilter, then full check and scan or declarations only
	Main                       // the repository's own packages: not type-checked here
)

func (k Kind) String() string {
	switch k {
	case Std:
		return "std"
	case XSys:
		return "xsys"
	case ThirdParty:
		return "third-party"
	case Main:
		return "main"
	}
	return fmt.Sprintf("kind(%d)", int(k))
}

// Module is the part of a package record's Module that loading uses.
type Module struct {
	Path      string
	Version   string
	GoVersion string // "" means go1.16 (cmd/go DefaultGoModVersion)
	Main      bool
}

// Package is one record of the configuration's package table (the third
// call, method record §3.4), already decoded and checked by the caller.
type Package struct {
	ImportPath string
	Dir        string // carried for the caller's records; positions use ImportPath and the file name
	Name       string
	Standard   bool
	Module     *Module
	GoFiles    []string
	Imports    []string
	ImportMap  map[string]string
	Kind       Kind
}

// Source returns the bytes of one file (a GoFiles name) of one package
// (its ImportPath). It is the caller's controlled read (method record §2.5);
// an error is a J-1 read failure of that package.
type Source func(importPath, file string) ([]byte, error)

// Admit is the admission of one unit of work (implementation plan §12.14):
// Run calls it before each package (PackageUnit) and before each file is
// read and parsed (FileUnit). When it returns false Run starts no new work:
// the refused unit and every package not yet finished are not evaluated
// ("budget"), and Admit is not called again.
type Admit func(unit string) bool

// PackageUnit is the admission unit of one package.
func PackageUnit(importPath string) string { return "typecheck package " + importPath }

// FileUnit is the admission unit of reading and parsing one file.
func FileUnit(importPath, file string) string {
	return "typecheck file " + importPath + " " + file
}

// Input is one configuration's input.
type Input struct {
	Config spec.Config
	// Toolchain is the toolchain's version: the GOVERSION the caller has
	// checked against runtime.Version(), e.g. "go1.27.3". Its language part
	// is the standard library's language version; a module's go version
	// above it (the full version, as cmd/go's allowedVersion compares) fails.
	Toolchain string
	Packages  []Package
	Source    Source
	Admit     Admit
}

// Depth is how a package was type-checked.
type Depth int

// The depths.
const (
	Skipped      Depth = iota // not type-checked ("unsafe", or loading did not reach the check)
	Declarations              // declarations only (IgnoreFuncBodies; bodies stripped)
	Full                      // with function bodies
)

func (d Depth) String() string {
	switch d {
	case Declarations:
		return "declarations-only"
	case Full:
		return "full"
	}
	return "skipped"
}

// Prefilter is the lexical prefilter's result (third-party packages only).
type Prefilter int

// The prefilter results.
const (
	PrefilterNA          Prefilter = iota // not a third-party package
	PrefilterHit                          // an object name occurs as an identifier token
	PrefilterMiss                         // no object name occurs as an identifier token
	PrefilterUnevaluated                  // not every file's bytes were obtained
)

func (p Prefilter) String() string {
	switch p {
	case PrefilterHit:
		return "hit"
	case PrefilterMiss:
		return "miss"
	case PrefilterUnevaluated:
		return "unevaluated"
	}
	return "n/a"
}

// FileResult is what Run recorded about one file it read.
type FileResult struct {
	Name   string
	Size   int
	SHA256 string // of the bytes parsed; "" when the read failed
	// Constraint is the I-7 constraint line: the //go:build line found by
	// go/build's file-header rules, its own bytes with the line end's LF and
	// CR removed; "" when the file has none. Derived for third-party and
	// golang.org/x/sys packages only; "" also when the derivation failed
	// (see the package's Header result).
	Constraint   string
	PrefilterHit bool // third-party packages only
}

// PackageResult is the outcome of one non-main package.
type PackageResult struct {
	ImportPath string
	Kind       Kind
	Depth      Depth
	Prefilter  Prefilter
	GoVersion  string // types.Config.GoVersion used; "" when undeterminable or not reached
	Files      []FileResult
	// SourceImports are the imports parsed from the files, each mapped
	// through ImportMap, sorted and without duplicates. Set when every file
	// was read and parsed.
	SourceImports []string
	// Load is the result of loading: the bytes, parsing, `import "C"`,
	// import resolution, the language version and the type check.
	Load judge.Result
	// ImportSet is I-6 (d), for third-party and golang.org/x/sys packages;
	// Unset for the others.
	ImportSet judge.Result
	// Header is the I-7 constraint derivation of every file, for
	// third-party and golang.org/x/sys packages; Unset for the others.
	Header  judge.Result
	Scanned bool // the object scan ran (fully checked third-party package)
}

// PackageLevel is the caller of a reference outside any function
// declaration: in a package-level variable's initializer (including a
// function literal there), a type or a constant. It cannot be a FullName.
const PackageLevel = "<package-level>"

// Ref is one source reference to an object of the table.
type Ref struct {
	Basis   spec.Basis // frozen (finding), x1 or x4 (diagnostic)
	Package string     // ImportPath
	File    string     // GoFiles name
	// Caller is the full name of the enclosing top-level function or
	// method (function literals belong to it), or PackageLevel.
	Caller string
	Object string // full name, after Origin() and alias expansion
	Line   int    // token.File.PositionFor(p, false): not adjusted by //line
	Column int    // in bytes, 1-based
	// Constraint and FileSHA256 are the file's I-7 constraint line ("" when
	// none) and the SHA-256 of the bytes parsed.
	Constraint string
	FileSHA256 string
}

func (r Ref) String() string {
	return fmt.Sprintf("%s %s/%s:%d:%d %s in %s", r.Basis, r.Package, r.File, r.Line, r.Column, r.Object, r.Caller)
}

// The classes of a signal call (limited evidence plan §2.3, LQ-5).
const (
	SignalConstSIGCHLD = "constant SIGCHLD"    // a constant of type syscall.Signal equal to the configuration's SIGCHLD
	SignalOtherConst   = "other constant"      // any other constant of type syscall.Signal
	SignalRuntime      = "runtime value"       // anything else, including a `...` expansion and interface values
	SignalNoArgs       = "no signal arguments" // a direct call without a signal argument
)

// SignalUse is the diagnostic of one reference to an os/signal object of
// the table. It classifies; it does not judge (§12.9's rule is not applied).
type SignalUse struct {
	Ref
	// Direct reports whether the reference is the function of a call
	// expression (parentheses around it allowed); method values, function
	// values and other references are not direct.
	Direct bool
	// Spread reports a `...` argument.
	Spread bool
	// Classes are, for a direct call, the class of each signal argument in
	// order (the first argument of Notify and NotifyContext is not a signal),
	// or the single SignalNoArgs; nil when the reference is not direct.
	Classes []string
}

// The self-check statuses.
const (
	SelfFound       = "found"
	SelfMissing     = "missing"
	SelfNotRequired = "not required"  // the object's package is not in this configuration's table
	SelfUnevaluated = "not evaluated" // the package is in the table but did not load
)

// SelfCheckItem is the table self-check of one object of ObjectsFor(GOOS).
type SelfCheckItem struct {
	Object string
	Status string
	Detail string
}

// Report is the outcome of one configuration.
type Report struct {
	Config    spec.Config
	Toolchain string
	// Setup is the configuration-level check: the input functions, the
	// GOOS, the toolchain version, types.SizesFor and the table's identity
	// and kinds. When it fails nothing else runs and every other check is
	// not evaluated.
	Setup judge.Result
	// Budget holds when every unit asked for was admitted; otherwise it
	// fails with one J-1 "budget insufficient" failure (Source budget) for
	// the refused unit, Refused.
	Budget  judge.Result
	Refused string
	// Order is the order the packages were processed in: every non-main
	// package except "unsafe", dependencies first.
	Order    []string
	Packages []PackageResult // by ImportPath; main packages are not listed
	// Findings are the references to frozen objects; Diagnostics those to
	// X1 and X4 objects; Signals the signal diagnostics. All sorted by
	// package, file, line, column and object.
	Findings    []Ref
	Diagnostics []Ref
	Signals     []SignalUse
	SelfCheck   []SelfCheckItem // in object table order
	// Scan is the third-party scan with the table self-check: it fails on a
	// missing table object or a reference to a table object on a GOOS the
	// table does not list; it is not evaluated when a third-party package's
	// prefilter or a hit package's scan, or a self-check item, could not be
	// evaluated.
	Scan judge.Result
}

// Package returns the result of one package, or false.
func (r *Report) Package(importPath string) (PackageResult, bool) {
	i := sort.Search(len(r.Packages), func(i int) bool { return r.Packages[i].ImportPath >= importPath })
	if i < len(r.Packages) && r.Packages[i].ImportPath == importPath {
		return r.Packages[i], true
	}
	return PackageResult{}, false
}

// Failure kinds (judge.Failure.Kind).
const (
	KindReadError             = "read error"
	KindParseError            = "parse error"
	KindImportC               = `import "C"`
	KindNotInImports          = "import not in Imports"
	KindNotInTable            = "import not in table"
	KindImportsMain           = "import of a main-module package"
	KindCycle                 = "import cycle"
	KindVersionUndeterminable = "language version undeterminable"
	KindVersionTooHigh        = "language version higher than toolchain"
	KindModuleMissing         = "module missing"
	KindSizesNil              = "sizes nil"
	KindTypeError             = "type error"
	KindTypeErrorSoft         = "type error (soft)"
	KindIncompletePackage     = "incomplete package"
	KindCheckerPanic          = "checker panic"
	KindPackageName           = "package name mismatch"
	KindDuplicateFile         = "duplicate file"
	KindNoFiles               = "no Go files"
	KindUnsafeNotStandard     = "unsafe not standard"
	KindImportsMismatch       = "imports mismatch"
	KindMultipleGoBuild       = "multiple //go:build"
	KindObjectMissing         = "object missing"
	KindOffPlatform           = "object off its table platforms"
	KindBudget                = "budget insufficient"
	KindDuplicatePackage      = "duplicate package"
	KindEmptyPath             = "empty import path"
	KindKindMismatch          = "kind mismatch"
	KindUnknownGOOS           = "unknown GOOS"
	KindToolchain             = "toolchain version undeterminable"
	KindNoInput               = "input function missing"
)

// Failure sites (judge.Failure.Site).
const (
	SiteSetup     = "typecheck.setup"
	SiteAdmission = "typecheck.admission"
	SiteSource    = "typecheck.source"
	SiteParse     = "go/parser"
	SiteImports   = "typecheck.imports"
	SiteOrder     = "typecheck.order"
	SiteVersion   = "typecheck.version"
	SiteTypes     = "go/types"
	SiteI6d       = "I-6(d)"
	SiteI7        = "I-7"
	SiteSelfCheck = "typecheck.selfcheck"
	SiteScan      = "typecheck.scan"
)

// Unevaluated reasons.
const (
	ReasonBudget  = "not evaluated: budget"
	ReasonSetup   = "not evaluated: setup failed"
	ReasonImports = "not evaluated: import incomplete: "
	ReasonSource  = "not evaluated: source unavailable"
	ReasonUnsafe  = `not evaluated: "unsafe" is never parsed`
)

// DefaultGoModVersion is cmd/go's go version for a module without one
// (cmd/go/internal/gover/version.go:24).
const DefaultGoModVersion = "1.16"

const parseMode = parser.SkipObjectResolution | parser.AllErrors

// Run loads, type-checks and scans one configuration.
func Run(in Input) *Report {
	r := newRunner(in)
	r.run()
	return r.rep
}

type runner struct {
	in      Input
	rep     *Report
	fset    *token.FileSet
	sizes   types.Sizes
	byPath  map[string]*Package
	res     map[string]*PackageResult
	done    map[string]*types.Package // packages whose Load holds ("unsafe": types.Unsafe)
	objects map[string]spec.Object    // ObjectsFor(GOOS) by full name
	all     map[string]spec.Object    // every table object by full name
	names   map[string]bool           // prefilter names
	stopped bool
	scan    judge.Acc // scan-level failures found while scanning
}

func newRunner(in Input) *runner {
	r := &runner{
		in:      in,
		rep:     &Report{Config: in.Config, Toolchain: in.Toolchain},
		fset:    token.NewFileSet(),
		byPath:  map[string]*Package{},
		res:     map[string]*PackageResult{},
		done:    map[string]*types.Package{},
		objects: map[string]spec.Object{},
		all:     map[string]spec.Object{},
		names:   spec.PrefilterNames(),
	}
	for _, o := range spec.Objects {
		r.all[o.FullName] = o
	}
	for _, o := range spec.ObjectsFor(in.Config.GOOS) {
		r.objects[o.FullName] = o
	}
	return r
}

func applies(k Kind) bool { return k == ThirdParty || k == XSys }

func expectedKind(p *Package) Kind {
	switch {
	case p.Standard:
		return Std
	case p.Module != nil && p.Module.Main:
		return Main
	case p.ImportPath == "golang.org/x/sys" || strings.HasPrefix(p.ImportPath, "golang.org/x/sys/"):
		return XSys
	}
	return ThirdParty
}

func (r *runner) run() {
	setup := r.setup()
	r.rep.Setup = setup.Result()
	r.rep.Budget = judge.Result{Status: judge.Hold}

	if !setup.Failed() {
		for i := range r.in.Packages {
			p := &r.in.Packages[i]
			if p.Kind == Main {
				continue
			}
			r.byPath[p.ImportPath] = p
			res := &PackageResult{ImportPath: p.ImportPath, Kind: p.Kind}
			if p.Kind == ThirdParty {
				res.Prefilter = PrefilterUnevaluated
			}
			r.res[p.ImportPath] = res
		}
		// Main packages are importable names (so an import of one is
		// recognised), but they are never processed.
		for i := range r.in.Packages {
			p := &r.in.Packages[i]
			if p.Kind == Main {
				r.byPath[p.ImportPath] = p
			}
		}
	} else {
		for i := range r.in.Packages {
			p := &r.in.Packages[i]
			if p.Kind == Main || r.res[p.ImportPath] != nil {
				continue
			}
			res := &PackageResult{ImportPath: p.ImportPath, Kind: p.Kind}
			res.Load = unevaluated(ReasonSetup)
			if applies(p.Kind) {
				res.ImportSet = unevaluated(ReasonSetup)
				res.Header = unevaluated(ReasonSetup)
			}
			if p.Kind == ThirdParty {
				res.Prefilter = PrefilterUnevaluated
			}
			r.res[p.ImportPath] = res
		}
		r.rep.Scan = unevaluated(ReasonSetup)
		r.collect()
		return
	}

	if res, ok := r.res["unsafe"]; ok {
		// Never read, parsed or scanned (loading step 3), whatever the record.
		res.Prefilter = PrefilterNA
		if applies(res.Kind) {
			res.ImportSet = unevaluated(ReasonUnsafe)
			res.Header = unevaluated(ReasonUnsafe)
		}
		if r.byPath["unsafe"].Standard {
			res.Load = judge.Result{Status: judge.Hold}
			r.done["unsafe"] = types.Unsafe
		} else {
			res.Load = fail(judge.Real(judge.J3, "unsafe", SiteImports, KindUnsafeNotStandard,
				`the table's "unsafe" is not Standard; it is never parsed and imports only as types.Unsafe`))
		}
	}

	order, cycles := r.order()
	r.rep.Order = order
	for _, path := range order {
		r.process(path, cycles[path])
	}
	r.selfCheck()
	r.finishScan()
	r.collect()
}

func (r *runner) setup() *judge.Acc {
	acc := &judge.Acc{}
	if r.in.Source == nil {
		acc.Add(judge.Real(judge.J1, "Source", SiteSetup, KindNoInput, "no byte source"))
	}
	if r.in.Admit == nil {
		acc.Add(judge.Real(judge.J1, "Admit", SiteSetup, KindNoInput, "no admission function"))
	}
	if _, ok := map[string]bool{"darwin": true, "linux": true, "windows": true}[r.in.Config.GOOS]; !ok {
		acc.Add(judge.Real(judge.J1, r.in.Config.String(), SiteSetup, KindUnknownGOOS,
			fmt.Sprintf("GOOS %q has no object table", r.in.Config.GOOS)))
	}
	if !version.IsValid(r.in.Toolchain) {
		acc.Add(judge.Real(judge.J3, r.in.Toolchain, SiteSetup, KindToolchain,
			fmt.Sprintf("toolchain version %q is not a valid Go version", r.in.Toolchain)))
	}
	r.sizes = types.SizesFor("gc", r.in.Config.GOARCH)
	if r.sizes == nil {
		acc.Add(judge.Real(judge.J3, r.in.Config.String(), SiteSetup, KindSizesNil,
			fmt.Sprintf(`types.SizesFor("gc", %q) is nil`, r.in.Config.GOARCH)))
	}
	seen := map[string]bool{}
	for i := range r.in.Packages {
		p := &r.in.Packages[i]
		if p.ImportPath == "" {
			acc.Add(judge.Real(judge.J3, fmt.Sprintf("package #%d", i), SiteSetup, KindEmptyPath, "empty ImportPath"))
			continue
		}
		if seen[p.ImportPath] {
			acc.Add(judge.Real(judge.J3, p.ImportPath, SiteSetup, KindDuplicatePackage, "ImportPath appears more than once"))
		}
		seen[p.ImportPath] = true
		if want := expectedKind(p); p.Kind != want {
			acc.Add(judge.Real(judge.J1, p.ImportPath, SiteSetup, KindKindMismatch,
				fmt.Sprintf("kind %s, record says %s", p.Kind, want)))
		}
	}
	return acc
}

// order returns the processing order (dependencies first) of every
// non-main package except "unsafe", and the cycle members with their cycle.
// Edges are Imports entries that are non-main packages of the table; other
// entries are failures of the importer (process). Tarjan's algorithm over
// sorted nodes and sorted edges: each strongly connected component is
// emitted after every component it reaches.
func (r *runner) order() ([]string, map[string][]string) {
	var nodes []string
	for path := range r.res {
		if path != "unsafe" {
			nodes = append(nodes, path)
		}
	}
	sort.Strings(nodes)
	edges := map[string][]string{}
	for _, n := range nodes {
		var es []string
		for _, imp := range uniqueSorted(r.byPath[n].Imports) {
			if dep, ok := r.res[imp]; ok && imp != "unsafe" && dep != nil {
				es = append(es, imp)
			}
		}
		edges[n] = es
	}

	index := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack, out []string
	cycles := map[string][]string{}
	next := 0
	var visit func(v string)
	visit = func(v string) {
		index[v], low[v] = next, next
		next++
		stack = append(stack, v)
		onStack[v] = true
		self := false
		for _, w := range edges[v] {
			if w == v {
				self = true
			}
			if _, seen := index[w]; !seen {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] != index[v] {
			return
		}
		var scc []string
		for {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[w] = false
			scc = append(scc, w)
			if w == v {
				break
			}
		}
		sort.Strings(scc)
		if len(scc) > 1 || self {
			for _, w := range scc {
				cycles[w] = scc
			}
		}
		out = append(out, scc...)
	}
	for _, n := range nodes {
		if _, seen := index[n]; !seen {
			visit(n)
		}
	}
	return out, cycles
}

func (r *runner) admit(unit string) bool {
	if r.stopped {
		return false
	}
	if r.in.Admit(unit) {
		return true
	}
	r.stopped = true
	r.rep.Refused = unit
	r.rep.Budget = fail(judge.Failure{
		Class: judge.J1, Item: unit, Site: SiteAdmission, Kind: KindBudget, Source: judge.SourceBudget,
		Detail: "admission refused; this unit and every package not yet finished are not evaluated",
	})
	return false
}

func (r *runner) markBudget(res *PackageResult) {
	res.Depth = Skipped
	res.Load = unevaluated(ReasonBudget)
	if applies(res.Kind) {
		res.ImportSet = unevaluated(ReasonBudget)
		res.Header = unevaluated(ReasonBudget)
	}
	if res.Kind == ThirdParty {
		res.Prefilter = PrefilterUnevaluated
	}
}

// process loads one package: every check whose input is available is
// evaluated, and the type check runs only when nothing of its own failed and
// every import loaded.
func (r *runner) process(path string, cycle []string) {
	p := r.byPath[path]
	res := r.res[path]
	if r.stopped || !r.admit(PackageUnit(path)) {
		r.markBudget(res)
		return
	}

	var own, header judge.Acc
	if cycle != nil {
		own.Add(judge.Real(judge.J3, path, SiteOrder, KindCycle, "import cycle among: "+strings.Join(cycle, ", ")))
	}
	lang, vf := r.langVersion(p)
	if vf != nil {
		own.Add(*vf)
	}
	res.GoVersion = lang

	names := append([]string(nil), p.GoFiles...)
	sort.Strings(names)
	if len(names) == 0 {
		own.Add(judge.Real(judge.J3, path, SiteSource, KindNoFiles, "GoFiles is empty"))
	}
	type parsed struct {
		file *ast.File
		fr   int // index into res.Files
	}
	var files []parsed
	seen := map[string]bool{}
	readFailed, parseFailed := false, false
	anyHit := false
	for _, name := range names {
		if seen[name] {
			own.Add(judge.Real(judge.J3, path+"/"+name, SiteSource, KindDuplicateFile, "GoFiles lists the file more than once"))
			continue
		}
		seen[name] = true
		if !r.admit(FileUnit(path, name)) {
			r.markBudget(res)
			return
		}
		src, err := r.in.Source(path, name)
		if err != nil {
			readFailed = true
			own.Add(judge.Real(judge.J1, path+"/"+name, SiteSource, KindReadError, err.Error()))
			res.Files = append(res.Files, FileResult{Name: name})
			continue
		}
		sum := sha256.Sum256(src)
		fr := FileResult{Name: name, Size: len(src), SHA256: hex.EncodeToString(sum[:])}
		if p.Kind == ThirdParty {
			fr.PrefilterHit = prefilterHit(src, r.names)
			anyHit = anyHit || fr.PrefilterHit
		}
		if applies(p.Kind) {
			line, err := ConstraintLine(src)
			if err != nil {
				header.Add(judge.Real(judge.J3, path+"/"+name, SiteI7, KindMultipleGoBuild, err.Error()))
			}
			fr.Constraint = line
		}
		res.Files = append(res.Files, fr)
		f, err := parser.ParseFile(r.fset, path+"/"+name, src, parseMode)
		if err != nil {
			parseFailed = true
			for _, msg := range parseErrors(err) {
				own.Add(judge.Real(judge.J3, path+"/"+name, SiteParse, KindParseError, msg))
			}
			continue
		}
		if f.Name.Name != p.Name {
			own.Add(judge.Real(judge.J3, path+"/"+name, SiteParse, KindPackageName,
				fmt.Sprintf("package clause %q, record Name %q", f.Name.Name, p.Name)))
		}
		files = append(files, parsed{f, len(res.Files) - 1})
	}

	if p.Kind == ThirdParty {
		switch {
		case readFailed:
			res.Prefilter = PrefilterUnevaluated
		case anyHit:
			res.Prefilter = PrefilterHit
		default:
			res.Prefilter = PrefilterMiss
		}
	}
	if applies(p.Kind) {
		switch {
		case header.Failed():
			res.Header = header.Result()
		case readFailed:
			res.Header = unevaluated(ReasonSource)
		default:
			res.Header = judge.Result{Status: judge.Hold}
		}
	}

	// Imports: what the files import, mapped through this package's
	// ImportMap, must be in Imports (loading step 5); every Imports entry
	// must be a non-main package of the table.
	inImports := map[string]bool{}
	for _, imp := range p.Imports {
		inImports[imp] = true
	}
	srcImports := map[string]bool{}
	for _, pf := range files {
		name := res.Files[pf.fr].Name
		for _, is := range pf.file.Imports {
			ip, err := strconv.Unquote(is.Path.Value)
			if err != nil {
				own.Add(judge.Real(judge.J3, path+"/"+name, SiteImports, KindParseError, "import path "+is.Path.Value))
				continue
			}
			mapped := ip
			if m, ok := p.ImportMap[ip]; ok {
				mapped = m
			}
			srcImports[mapped] = true
			if ip == "C" {
				own.Add(judge.Real(judge.J3, path+"/"+name, SiteImports, KindImportC, `the file imports "C" (FakeImportC is off)`))
				continue
			}
			if !inImports[mapped] {
				own.Add(judge.Real(judge.J3, path+"/"+name, SiteImports, KindNotInImports,
					fmt.Sprintf("import %q (mapped to %q) is not in Imports", ip, mapped)))
			}
		}
	}
	for _, imp := range uniqueSorted(p.Imports) {
		dep, ok := r.byPath[imp]
		switch {
		case !ok:
			own.Add(judge.Real(judge.J3, path, SiteImports, KindNotInTable, fmt.Sprintf("Imports entry %q is not in the package table", imp)))
		case dep.Kind == Main:
			own.Add(judge.Real(judge.J3, path, SiteImports, KindImportsMain,
				fmt.Sprintf("Imports entry %q is a main-module package, which is not type-checked", imp)))
		}
	}
	if !readFailed && !parseFailed {
		res.SourceImports = keys(srcImports)
	}
	if applies(p.Kind) {
		if readFailed || parseFailed {
			res.ImportSet = unevaluated(ReasonSource)
		} else {
			res.ImportSet = importSet(path, res.SourceImports, uniqueSorted(p.Imports))
		}
	}

	if own.Failed() {
		res.Depth = Skipped
		res.Load = own.Result()
		return
	}
	var incomplete []string
	for _, imp := range uniqueSorted(p.Imports) {
		if dep := r.res[imp]; dep == nil || dep.Load.Status != judge.Hold {
			incomplete = append(incomplete, imp)
		}
	}
	if len(incomplete) > 0 {
		res.Depth = Skipped
		res.Load = unevaluated(ReasonImports + strings.Join(incomplete, ", "))
		return
	}

	full := p.Kind == ThirdParty && res.Prefilter == PrefilterHit
	astFiles := make([]*ast.File, len(files))
	for i, pf := range files {
		if !full {
			StripBodies(pf.file)
		}
		astFiles[i] = pf.file
	}
	var info *types.Info
	if full {
		info = &types.Info{
			Types: map[ast.Expr]types.TypeAndValue{},
			Defs:  map[*ast.Ident]types.Object{},
			Uses:  map[*ast.Ident]types.Object{},
		}
	}
	tpkg, fails := r.check(p, lang, astFiles, full, info)
	if full {
		res.Depth = Full
	} else {
		res.Depth = Declarations
	}
	if len(fails) > 0 {
		res.Load = fail(fails...)
		return
	}
	res.Load = judge.Result{Status: judge.Hold}
	r.done[path] = tpkg
	if full {
		frs := make([]FileResult, len(files))
		for i, pf := range files {
			frs[i] = res.Files[pf.fr]
		}
		r.scanPackage(path, astFiles, frs, info)
		res.Scanned = true
	}
}

func (r *runner) langVersion(p *Package) (string, *judge.Failure) {
	if p.Module != nil {
		v := p.Module.GoVersion
		if v == "" {
			v = DefaultGoModVersion
		}
		gv := "go" + v
		if !version.IsValid(gv) {
			f := judge.Real(judge.J3, p.ImportPath, SiteVersion, KindVersionUndeterminable,
				fmt.Sprintf("Module.GoVersion %q is not a Go version", p.Module.GoVersion))
			return "", &f
		}
		if version.Compare(gv, r.in.Toolchain) > 0 {
			f := judge.Real(judge.J3, p.ImportPath, SiteVersion, KindVersionTooHigh,
				fmt.Sprintf("Module.GoVersion %q is above the toolchain %s", p.Module.GoVersion, r.in.Toolchain))
			return "", &f
		}
		return version.Lang(gv), nil
	}
	if !p.Standard {
		f := judge.Real(judge.J3, p.ImportPath, SiteVersion, KindModuleMissing, "Module is nil for a package that is not Standard")
		return "", &f
	}
	return version.Lang(r.in.Toolchain), nil
}

// importer is one package's importer (loading step 5): the path written in
// the source is mapped through this package's ImportMap, must then be in
// this package's Imports, and must be a package whose check completed
// without error. Nothing is loaded on demand.
type importer struct {
	from      *Package
	inImports map[string]bool
	done      map[string]*types.Package
}

func (im *importer) Import(path string) (*types.Package, error) {
	if path == "C" {
		return nil, errors.New(`import "C" is rejected (FakeImportC is off)`)
	}
	mapped := path
	if m, ok := im.from.ImportMap[path]; ok {
		mapped = m
	}
	if !im.inImports[mapped] {
		return nil, fmt.Errorf("%q (mapped to %q) is not in Imports", path, mapped)
	}
	p := im.done[mapped]
	if p == nil || !p.Complete() {
		return nil, fmt.Errorf("%q has not completed its check", mapped)
	}
	return p, nil
}

func (r *runner) check(p *Package, lang string, files []*ast.File, full bool, info *types.Info) (tpkg *types.Package, fails []judge.Failure) {
	inImports := map[string]bool{}
	for _, imp := range p.Imports {
		inImports[imp] = true
	}
	conf := types.Config{
		GoVersion:        lang,
		IgnoreFuncBodies: !full,
		FakeImportC:      false,
		Importer:         &importer{from: p, inImports: inImports, done: r.done},
		Sizes:            r.sizes,
		Error:            func(err error) { fails = append(fails, r.typeFailure(p.ImportPath, err)) },
	}
	defer func() {
		if v := recover(); v != nil {
			tpkg = nil
			fails = append(fails, judge.Real(judge.J1, p.ImportPath, SiteTypes, KindCheckerPanic, fmt.Sprint(v)))
		}
	}()
	tpkg, err := conf.Check(p.ImportPath, r.fset, files, info)
	if err != nil && len(fails) == 0 {
		fails = append(fails, r.typeFailure(p.ImportPath, err))
	}
	if len(fails) == 0 && (tpkg == nil || !tpkg.Complete()) {
		fails = append(fails, judge.Real(judge.J3, p.ImportPath, SiteTypes, KindIncompletePackage, "the checked package is not complete"))
	}
	return tpkg, fails
}

func (r *runner) typeFailure(path string, err error) judge.Failure {
	var te types.Error
	if errors.As(err, &te) {
		kind := KindTypeError
		if te.Soft {
			kind = KindTypeErrorSoft
		}
		detail := te.Msg
		if pos := r.fset.PositionFor(te.Pos, false); pos.IsValid() {
			detail = fmt.Sprintf("%s:%d:%d: %s", pos.Filename, pos.Line, pos.Column, te.Msg)
		}
		return judge.Real(judge.J3, path, SiteTypes, kind, detail)
	}
	return judge.Real(judge.J3, path, SiteTypes, KindTypeError, err.Error())
}

// importSet is I-6 (d): the parsed imports, mapped through ImportMap, equal
// the reported Imports as a set ("unsafe" included as it appears).
func importSet(path string, parsed, reported []string) judge.Result {
	ps, rs := setOf(parsed), setOf(reported)
	var onlyParsed, onlyReported []string
	for _, x := range parsed {
		if !rs[x] {
			onlyParsed = append(onlyParsed, x)
		}
	}
	for _, x := range reported {
		if !ps[x] {
			onlyReported = append(onlyReported, x)
		}
	}
	if len(onlyParsed) == 0 && len(onlyReported) == 0 {
		return judge.Result{Status: judge.Hold}
	}
	return fail(judge.Real(judge.J3, path, SiteI6d, KindImportsMismatch, fmt.Sprintf(
		"parsed (mapped) [%s]; Imports [%s]; only parsed [%s]; only Imports [%s]",
		strings.Join(parsed, " "), strings.Join(reported, " "),
		strings.Join(onlyParsed, " "), strings.Join(onlyReported, " "))))
}

// selfCheck is the table self-check of §12.9: every object of
// ObjectsFor(GOOS) whose package was type-checked must be in the type
// information as a package-level function, or as a method in the pointer
// method set of the named type, with exactly the table's full name.
func (r *runner) selfCheck() {
	for _, o := range spec.ObjectsFor(r.in.Config.GOOS) {
		item := SelfCheckItem{Object: o.FullName}
		res, inTable := r.res[o.PkgPath]
		switch {
		case !inTable:
			item.Status = SelfNotRequired
		case res.Load.Status != judge.Hold || r.done[o.PkgPath] == nil:
			item.Status = SelfUnevaluated
			item.Detail = "package " + o.PkgPath + " did not load"
		default:
			if got, ok := lookupObject(r.done[o.PkgPath], o); ok {
				item.Status = SelfFound
			} else {
				item.Status = SelfMissing
				item.Detail = got
				r.scan.Add(judge.Real(judge.J3, o.FullName, SiteSelfCheck, KindObjectMissing, got))
			}
		}
		r.rep.SelfCheck = append(r.rep.SelfCheck, item)
	}
}

// lookupObject finds a table object in a checked package. It returns the
// reason when the object is missing.
func lookupObject(pkg *types.Package, o spec.Object) (string, bool) {
	var f *types.Func
	if o.Recv == "" {
		obj := pkg.Scope().Lookup(o.Name)
		fn, ok := obj.(*types.Func)
		if !ok {
			return fmt.Sprintf("no package-level function %s in %s", o.Name, pkg.Path()), false
		}
		f = fn
	} else {
		tn, ok := pkg.Scope().Lookup(o.Recv).(*types.TypeName)
		if !ok {
			return fmt.Sprintf("no type %s in %s", o.Recv, pkg.Path()), false
		}
		named, ok := types.Unalias(tn.Type()).(*types.Named)
		if !ok {
			return fmt.Sprintf("%s.%s is not a named type", pkg.Path(), o.Recv), false
		}
		sel := types.NewMethodSet(types.NewPointer(named)).Lookup(pkg, o.Name)
		if sel == nil {
			return fmt.Sprintf("no method %s in the method set of *%s.%s", o.Name, pkg.Path(), o.Recv), false
		}
		fn, ok := sel.Obj().(*types.Func)
		if !ok {
			return fmt.Sprintf("%s of *%s.%s is not a function", o.Name, pkg.Path(), o.Recv), false
		}
		f = fn
	}
	if got := FuncName(f); got != o.FullName {
		return fmt.Sprintf("found %s, want %s", got, o.FullName), false
	}
	return "", true
}

func (r *runner) finishScan() {
	if r.scan.Failed() {
		r.rep.Scan = r.scan.Result()
		return
	}
	var pending []string
	if r.stopped {
		pending = append(pending, "budget")
	}
	for _, path := range sortedKeys(r.res) {
		res := r.res[path]
		if res.Kind != ThirdParty {
			continue
		}
		switch {
		case res.Prefilter == PrefilterUnevaluated:
			pending = append(pending, "prefilter of "+path)
		case res.Prefilter == PrefilterHit && !res.Scanned:
			pending = append(pending, "scan of "+path)
		}
	}
	for _, it := range r.rep.SelfCheck {
		if it.Status == SelfUnevaluated {
			pending = append(pending, "self-check of "+it.Object)
		}
	}
	if len(pending) > 0 {
		r.rep.Scan = unevaluated("not evaluated: " + strings.Join(pending, "; "))
		return
	}
	r.rep.Scan = judge.Result{Status: judge.Hold}
}

func (r *runner) collect() {
	for _, path := range sortedKeys(r.res) {
		r.rep.Packages = append(r.rep.Packages, *r.res[path])
	}
	sortRefs(r.rep.Findings)
	sortRefs(r.rep.Diagnostics)
	sort.SliceStable(r.rep.Signals, func(i, j int) bool { return refLess(r.rep.Signals[i].Ref, r.rep.Signals[j].Ref) })
}

func refLess(a, b Ref) bool {
	if a.Package != b.Package {
		return a.Package < b.Package
	}
	if a.File != b.File {
		return a.File < b.File
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	if a.Column != b.Column {
		return a.Column < b.Column
	}
	return a.Object < b.Object
}

func sortRefs(rs []Ref) { sort.SliceStable(rs, func(i, j int) bool { return refLess(rs[i], rs[j]) }) }

func fail(fs ...judge.Failure) judge.Result {
	return judge.Result{Status: judge.Fail, Failures: append([]judge.Failure(nil), fs...)}
}

func unevaluated(reason string) judge.Result {
	return judge.Result{Status: judge.Unevaluated, Reason: reason}
}

// parseErrors lists every syntax error (the parser runs with AllErrors and
// sorts them by position).
func parseErrors(err error) []string {
	var list scanner.ErrorList
	if errors.As(err, &list) && len(list) > 0 {
		out := make([]string, len(list))
		for i, e := range list {
			out[i] = e.Error()
		}
		return out
	}
	return []string{err.Error()}
}

func uniqueSorted(xs []string) []string {
	return keys(setOf(xs))
}

func setOf(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
