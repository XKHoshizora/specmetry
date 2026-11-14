package typecheck

import (
	"errors"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// runInternal runs like Run and also returns the runner, so a test can look
// at the checked types.Package values.
func runInternal(in Input) (*Report, *runner) {
	r := newRunner(in)
	r.run()
	return r.rep, r
}

// The unsafe fixture of §12.12 ("装载的对照"): package-level Sizeof in a
// constant, Pointer conversions, unsafe.String and a package-level
// initialization through a function using unsafe. N is 16 on both 64-bit
// architectures, and the import is types.Unsafe itself.
const unsafeSrc = `package usf

import "unsafe"

const N = unsafe.Sizeof(struct {
	a int64
	b byte
}{})

var _ [N]byte

var x int64

var p = unsafe.Pointer(&x)

var q = (*int64)(unsafe.Pointer(&x))

var b = []byte("a")

var s = unsafe.String(&b[0], 1)

func size() uintptr { return unsafe.Sizeof(x) + unsafe.Alignof(x) }

var z = size()
`

func TestUnsafe(t *testing.T) {
	const p = "example.com/usf"
	for _, cfg := range []spec.Config{linuxAMD64, linuxARM64} {
		t.Run(cfg.String(), func(t *testing.T) {
			w := build(append(stdStubs("linux"), thirdParty(p, "1.22", []string{"unsafe"}, map[string]string{"usf.go": unsafeSrc}))...)
			rep, r := runInternal(w.input(cfg))
			res := pkgOf(t, rep, p)
			wantHold(t, "usf", res.Load)
			wantHold(t, "usf I-6(d)", res.ImportSet)
			if res.Prefilter != PrefilterMiss || res.Depth != Declarations {
				t.Errorf("usf: prefilter %s depth %s", res.Prefilter, res.Depth)
			}
			u := pkgOf(t, rep, "unsafe")
			wantHold(t, "unsafe", u.Load)
			if u.Depth != Skipped || len(u.Files) != 0 {
				t.Errorf("unsafe: depth %s files %v", u.Depth, u.Files)
			}
			if got := w.readsOf("unsafe"); len(got) != 0 {
				t.Errorf("unsafe was read: %v", got)
			}
			tp := r.done[p]
			n, ok := tp.Scope().Lookup("N").(*types.Const)
			if !ok {
				t.Fatalf("N is not a constant")
			}
			if v, exact := constant.Int64Val(n.Val()); !exact || v != 16 {
				t.Errorf("N = %v, want 16", n.Val())
			}
			if imps := tp.Imports(); len(imps) != 1 || imps[0] != types.Unsafe {
				t.Errorf("imports %v, want exactly types.Unsafe", imps)
			}
			for _, o := range rep.Order {
				if o == "unsafe" {
					t.Errorf("unsafe is in the processing order")
				}
			}
		})
	}
}

// The counter-example of the unsafe special rule: checking a source stub of
// unsafe as an ordinary package gives a different package, and the same
// constant declaration does not check (directly with go/types, not through
// the loader).
func TestUnsafeAsSourceIsNotUnsafe(t *testing.T) {
	fset := token.NewFileSet()
	stub, err := parser.ParseFile(fset, "unsafe.go", `package unsafe

type ArbitraryType int

type Pointer *ArbitraryType

func Sizeof(x ArbitraryType) uintptr
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	up, err := (&types.Config{}).Check("unsafe", fset, []*ast.File{stub}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if up == types.Unsafe {
		t.Fatal("source unsafe is types.Unsafe")
	}
	f, err := parser.ParseFile(fset, "usf.go", unsafeSrc, 0)
	if err != nil {
		t.Fatal(err)
	}
	var errs []error
	conf := types.Config{
		Importer: importerFunc(func(string) (*types.Package, error) { return up, nil }),
		Error:    func(err error) { errs = append(errs, err) },
	}
	_, _ = conf.Check("example.com/usf", fset, []*ast.File{f}, nil)
	if len(errs) == 0 {
		t.Fatal("the fixture checked against a source unsafe")
	}
}

type importerFunc func(string) (*types.Package, error)

func (f importerFunc) Import(p string) (*types.Package, error) { return f(p) }

func TestImportC(t *testing.T) {
	const p = "example.com/cg"
	w := build(thirdParty(p, "1.22", nil, map[string]string{"cg.go": "package cg\n\nimport \"C\"\n\nvar X = 1\n"}))
	rep := Run(w.input(linuxAMD64))
	res := pkgOf(t, rep, p)
	wantFail(t, "cg", res.Load, "J-3 "+KindImportC+" "+p+"/cg.go")
	wantFail(t, "cg I-6(d)", res.ImportSet, "J-3 "+KindImportsMismatch+" "+p)
	if res.Depth != Skipped {
		t.Errorf("depth %s", res.Depth)
	}
}

func TestImportNotInImports(t *testing.T) {
	const p = "example.com/miss"
	src := "package miss\n\nimport \"os\"\n\nvar I = os.Interrupt\n"
	w := build(append(stdStubs("linux"), thirdParty(p, "1.22", []string{"context"}, map[string]string{"miss.go": src}))...)
	rep := Run(w.input(linuxAMD64))
	res := pkgOf(t, rep, p)
	wantFail(t, "miss", res.Load, "J-3 "+KindNotInImports+" "+p+"/miss.go")
	wantFail(t, "miss I-6(d)", res.ImportSet, "J-3 "+KindImportsMismatch+" "+p)
	d := res.ImportSet.Failures[0].Detail
	want := "parsed (mapped) [os]; Imports [context]; only parsed [os]; only Imports [context]"
	if d != want {
		t.Errorf("I-6(d) detail %q, want %q", d, want)
	}
}

// ImportMap: a standard package's import of a vendored path resolves
// through the importing package's map; a missing map entry, a map to a path
// outside Imports, and a map to a path outside the table all fail.
func TestImportMap(t *testing.T) {
	const vend = "vendor/golang.org/x/net/idna"
	src := "package httpx\n\nimport \"golang.org/x/net/idna\"\n\nvar V = idna.Version\n"
	idna := std(vend, nil, map[string]string{"idna.go": "package idna\n\nconst Version = 1\n"})
	for _, tc := range []struct {
		name    string
		m       map[string]string
		imports []string
		table   bool
		want    []string
	}{
		{"mapped", map[string]string{"golang.org/x/net/idna": vend}, []string{vend}, true, nil},
		{"no map", nil, []string{vend}, true, []string{"J-3 " + KindNotInImports + " net/httpx/httpx.go"}},
		{"map outside Imports", map[string]string{"golang.org/x/net/idna": "vendor/other"}, []string{vend}, true,
			[]string{"J-3 " + KindNotInImports + " net/httpx/httpx.go"}},
		{"Imports outside table", map[string]string{"golang.org/x/net/idna": vend}, []string{vend}, false,
			[]string{"J-3 " + KindNotInTable + " net/httpx"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := std("net/httpx", tc.imports, map[string]string{"httpx.go": src})
			h.importMap = tc.m
			fxs := []fx{h}
			if tc.table {
				fxs = append(fxs, idna)
			}
			rep := Run(build(fxs...).input(linuxAMD64))
			res := pkgOf(t, rep, "net/httpx")
			if tc.want == nil {
				wantHold(t, "httpx", res.Load)
				if !reflect.DeepEqual(res.SourceImports, []string{vend}) {
					t.Errorf("source imports %v", res.SourceImports)
				}
				return
			}
			wantFail(t, "httpx", res.Load, tc.want...)
		})
	}
}

// A cycle fails its members; a package importing a member is not evaluated
// (import incomplete), not failed a second time.
func TestCycle(t *testing.T) {
	a := thirdParty("example.com/cyca", "1.22", []string{"example.com/cycb"}, map[string]string{"a.go": "package cyca\n\nimport _ \"example.com/cycb\"\n"})
	b := thirdParty("example.com/cycb", "1.22", []string{"example.com/cyca"}, map[string]string{"b.go": "package cycb\n\nimport _ \"example.com/cyca\"\n"})
	c := thirdParty("example.com/cycc", "1.22", []string{"example.com/cyca"}, map[string]string{"c.go": "package cycc\n\nimport _ \"example.com/cyca\"\n"})
	self := thirdParty("example.com/self", "1.22", []string{"example.com/self"}, map[string]string{"s.go": "package self\n"})
	rep := Run(build(a, b, c, self).input(linuxAMD64))
	for _, p := range []string{"example.com/cyca", "example.com/cycb"} {
		res := pkgOf(t, rep, p)
		wantFail(t, p, res.Load, "J-3 "+KindCycle+" "+p)
		if d := res.Load.Failures[0].Detail; d != "import cycle among: example.com/cyca, example.com/cycb" {
			t.Errorf("%s: %q", p, d)
		}
	}
	wantFail(t, "self", pkgOf(t, rep, "example.com/self").Load, "J-3 "+KindCycle+" example.com/self")
	wantUnevaluated(t, "cycc", pkgOf(t, rep, "example.com/cycc").Load, ReasonImports+"example.com/cyca")
	// The cycle members come before their importer in the order.
	pos := map[string]int{}
	for i, o := range rep.Order {
		pos[o] = i
	}
	if pos["example.com/cycc"] < pos["example.com/cyca"] {
		t.Errorf("order %v", rep.Order)
	}
}

// Type errors: every error is kept (soft ones too), and a dependent is not
// evaluated.
func TestTypeErrors(t *testing.T) {
	const p = "example.com/bad"
	src := `package bad

import "os/exec"

func F(c *exec.Cmd) error {
	var unused int
	var s string = 1
	_ = s
	return c.Run()
}
`
	dep := thirdParty("example.com/dep", "1.22", []string{p}, map[string]string{"dep.go": "package dep\n\nimport _ \"example.com/bad\"\n"})
	w := build(append(stdStubs("linux"), thirdParty(p, "1.22", []string{"os/exec"}, map[string]string{"bad.go": src}), dep)...)
	rep := Run(w.input(linuxAMD64))
	res := pkgOf(t, rep, p)
	if res.Depth != Full {
		t.Errorf("depth %s", res.Depth)
	}
	wantFail(t, "bad", res.Load, "J-3 "+KindTypeError+" "+p, "J-3 "+KindTypeErrorSoft+" "+p)
	var details []string
	for _, f := range res.Load.Failures {
		details = append(details, f.Detail)
	}
	joined := strings.Join(details, "\n")
	for _, want := range []string{p + "/bad.go:6:6: declared and not used: unused", p + "/bad.go:7:17: "} {
		if !strings.Contains(joined, want) {
			t.Errorf("type errors\n%s\ndo not contain %q", joined, want)
		}
	}
	if res.Scanned || len(rep.Findings) != 0 {
		t.Errorf("a failed package was scanned: %v", rep.Findings)
	}
	wantUnevaluated(t, "dep", pkgOf(t, rep, "example.com/dep").Load, ReasonImports+p)
	checkComplete(t, rep)
	wantUnevaluated(t, "scan", rep.Scan, "not evaluated: scan of "+p)
}

func TestLanguageVersion(t *testing.T) {
	generic := "package gen\n\nfunc Id[T any](x T) T { return x }\n"
	for _, tc := range []struct {
		name    string
		mod     *Module
		std     bool
		src     string
		version string
		kind    string // failure kind; "" for hold
	}{
		{"empty is go1.16", &Module{Path: "example.com/gen", Version: "v1.0.0"}, false, "package gen\n\nvar X = 1\n", "go1.16", ""},
		{"go1.16 rejects generics", &Module{Path: "example.com/gen", Version: "v1.0.0"}, false, generic, "go1.16", KindTypeError},
		{"1.27 is go1.27", &Module{Path: "example.com/gen", Version: "v1.0.0", GoVersion: "1.27"}, false, generic, "go1.27", ""},
		{"1.27.3 is go1.27", &Module{Path: "example.com/gen", Version: "v1.0.0", GoVersion: "1.27.3"}, false, generic, "go1.27", ""},
		{"1.27.4 is above go1.27.3", &Module{Path: "example.com/gen", Version: "v1.0.0", GoVersion: "1.27.4"}, false, generic, "", KindVersionTooHigh},
		{"1.28 is above", &Module{Path: "example.com/gen", Version: "v1.0.0", GoVersion: "1.28"}, false, generic, "", KindVersionTooHigh},
		{"not a version", &Module{Path: "example.com/gen", Version: "v1.0.0", GoVersion: "banana"}, false, generic, "", KindVersionUndeterminable},
		{"nil module, not standard", nil, false, generic, "", KindModuleMissing},
		{"standard is the toolchain's", nil, true, generic, "go1.27", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := fx{path: "example.com/gen", kind: ThirdParty, mod: tc.mod, files: map[string]string{"gen.go": tc.src}}
			if tc.std {
				f = std("gen", nil, map[string]string{"gen.go": tc.src})
			}
			rep := Run(build(f).input(linuxAMD64))
			res := pkgOf(t, rep, f.path)
			if res.GoVersion != tc.version {
				t.Errorf("GoVersion %q, want %q", res.GoVersion, tc.version)
			}
			if tc.kind == "" {
				wantHold(t, "load", res.Load)
				return
			}
			if tc.kind == KindTypeError {
				// go1.16 rejects both the type parameter and the
				// predeclared any; every failure is a type error and one
				// names go1.18.
				if res.Load.Status != judge.Fail {
					t.Fatalf("load %s", res.Load.Status)
				}
				var details []string
				for _, f := range res.Load.Failures {
					if f.Kind != KindTypeError || f.Class != judge.J3 {
						t.Errorf("failure %+v", f)
					}
					details = append(details, f.Detail)
				}
				if !strings.Contains(strings.Join(details, "\n"), "go1.18") {
					t.Errorf("details %q", details)
				}
				return
			}
			wantFail(t, "load", res.Load, "J-3 "+tc.kind+" "+f.path)
		})
	}
}

// Declarations only: a package-level function literal and a generic
// function with a body still check after stripping; the prefilter does not
// see names in comments, strings or raw strings; and a body is not checked
// (its error is invisible), which is what "declarations only" means.
func TestDeclarationsOnly(t *testing.T) {
	pool := std("pool", nil, map[string]string{"pool.go": `package pool

type Pool struct{ New func() any }

var P = Pool{New: func() any { return new(int) }}

func Map[T any](x T) T { return x }

var M = Map(3)
`})
	quiet := thirdParty("example.com/quiet", "1.22", []string{"os/exec"}, map[string]string{"quiet.go": "package quiet\n\n" +
		"import \"os/exec\"\n\n" +
		"// Run, Start, Wait and StartProcess appear only in comments.\n" +
		"/* Notify Reset Ignore */\n" +
		"const doc = \"Output CombinedOutput Exec\"\n\n" +
		"const raw = `ForkExec Wait4 Waitid`\n\n" +
		"var cmd = exec.Command\n\n" +
		"func helper() int { return undefinedName }\n"})
	w := build(append(stdStubs("linux"), pool, quiet)...)
	rep := Run(w.input(linuxAMD64))
	pr := pkgOf(t, rep, "pool")
	wantHold(t, "pool", pr.Load)
	if pr.Depth != Declarations {
		t.Errorf("pool depth %s", pr.Depth)
	}
	q := pkgOf(t, rep, "example.com/quiet")
	wantHold(t, "quiet", q.Load)
	if q.Prefilter != PrefilterMiss || q.Depth != Declarations || q.Scanned || q.Files[0].PrefilterHit {
		t.Errorf("quiet: prefilter %s depth %s scanned %v", q.Prefilter, q.Depth, q.Scanned)
	}
	if len(rep.Findings) != 0 {
		t.Errorf("findings %v", rep.Findings)
	}
	wantHold(t, "scan", rep.Scan)
}

func TestByteSourceError(t *testing.T) {
	w := build(append(stdStubs("linux"), lc7())...)
	w.errs[lc7Path+" forms.go"] = errors.New("boom")
	rep := Run(w.input(linuxAMD64))
	res := pkgOf(t, rep, lc7Path)
	wantFail(t, "lc7", res.Load, "J-1 "+KindReadError+" "+lc7Path+"/forms.go")
	checkComplete(t, rep)
	if res.Load.Failures[0].Detail != "boom" || res.Load.Failures[0].Source != judge.SourceReal {
		t.Errorf("failure %+v", res.Load.Failures[0])
	}
	if res.Prefilter != PrefilterUnevaluated {
		t.Errorf("prefilter %s", res.Prefilter)
	}
	wantUnevaluated(t, "I-6(d)", res.ImportSet, ReasonSource)
	wantUnevaluated(t, "I-7", res.Header, ReasonSource)
	if res.Files[0].SHA256 != "" || res.Files[1].SHA256 != sum(lc7Signals) {
		t.Errorf("files %+v", res.Files)
	}
	wantUnevaluated(t, "scan", rep.Scan, "not evaluated: prefilter of "+lc7Path)
}

// I-6 (d): the reported Imports carry an entry the files do not import.
// Loading holds (the importer only needs the source imports to be in
// Imports); I-6 (d) fails and lists both sides.
func TestImportSetMismatch(t *testing.T) {
	const p = "example.com/imix"
	src := "package imix\n\nimport \"os\"\n\nvar I = os.Interrupt\n"
	w := build(append(stdStubs("linux"), thirdParty(p, "1.22", []string{"context", "os"}, map[string]string{"imix.go": src}))...)
	rep := Run(w.input(linuxAMD64))
	res := pkgOf(t, rep, p)
	wantHold(t, "imix", res.Load)
	wantFail(t, "imix I-6(d)", res.ImportSet, "J-3 "+KindImportsMismatch+" "+p)
	want := "parsed (mapped) [os]; Imports [context os]; only parsed []; only Imports [context]"
	if d := res.ImportSet.Failures[0].Detail; d != want {
		t.Errorf("detail %q, want %q", d, want)
	}
	// golang.org/x/sys packages get I-6 (d) as well.
	x := xsysUnix(false)
	x.imports = []string{"syscall"}
	rep = Run(build(append(stdStubs("linux"), x)...).input(linuxAMD64))
	wantFail(t, "x/sys/unix I-6(d)", pkgOf(t, rep, "golang.org/x/sys/unix").ImportSet, "J-3 "+KindImportsMismatch+" golang.org/x/sys/unix")
}

// The repository's packages are not read, not checked and not listed; an
// import of one from a dependency fails.
func TestMainPackagesNotChecked(t *testing.T) {
	m := fx{path: spec.MainModule + "/cmd/x", kind: Main, files: map[string]string{"main.go": "package main\n\nfunc main() { undefined() }\n"}, imports: []string{lc7Path}}
	back := thirdParty("example.com/back", "1.22", []string{spec.MainModule + "/cmd/x"}, map[string]string{"b.go": "package back\n"})
	w := build(append(stdStubs("linux"), lc7(), m, back)...)
	rep := Run(w.input(linuxAMD64))
	if _, ok := rep.Package(spec.MainModule + "/cmd/x"); ok {
		t.Errorf("main package listed")
	}
	if got := w.readsOf(spec.MainModule + "/cmd/x"); len(got) != 0 {
		t.Errorf("main package read: %v", got)
	}
	wantHold(t, "lc7", pkgOf(t, rep, lc7Path).Load)
	wantFail(t, "back", pkgOf(t, rep, "example.com/back").Load, "J-3 "+KindImportsMain+" example.com/back")
}

func TestPackageNameAndFiles(t *testing.T) {
	f := thirdParty("example.com/nm", "1.22", nil, map[string]string{"a.go": "package other\n"})
	empty := thirdParty("example.com/empty", "1.22", nil, nil)
	broken := thirdParty("example.com/broken", "1.22", nil, map[string]string{"a.go": "package broken\n\nvar = \n", "b.go": "package broken\n\nfunc (\n"})
	rep := Run(build(f, empty, broken).input(linuxAMD64))
	wantFail(t, "nm", pkgOf(t, rep, "example.com/nm").Load, "J-3 "+KindPackageName+" example.com/nm/a.go")
	wantFail(t, "empty", pkgOf(t, rep, "example.com/empty").Load, "J-3 "+KindNoFiles+" example.com/empty")
	br := pkgOf(t, rep, "example.com/broken")
	if br.Load.Status != judge.Fail || len(br.Load.Failures) < 2 {
		t.Fatalf("broken: %s %v", br.Load.Status, br.Load.Failures)
	}
	files := map[string]bool{}
	for _, f := range br.Load.Failures {
		if f.Kind != KindParseError || f.Class != judge.J3 {
			t.Errorf("failure %+v", f)
		}
		files[f.Item] = true
	}
	if !files["example.com/broken/a.go"] || !files["example.com/broken/b.go"] {
		t.Errorf("parse errors of both files expected, got %v", br.Load.Failures)
	}
	wantUnevaluated(t, "broken I-6(d)", br.ImportSet, ReasonSource)
	wantHold(t, "broken I-7", br.Header)
}

func TestSetupFailures(t *testing.T) {
	base := func() *world { return build(append(stdStubs("linux"), lc7())...) }
	for _, tc := range []struct {
		name string
		edit func(*Input)
		want string
	}{
		{"no source", func(in *Input) { in.Source = nil }, "J-1 " + KindNoInput + " Source"},
		{"no admit", func(in *Input) { in.Admit = nil }, "J-1 " + KindNoInput + " Admit"},
		{"unknown GOOS", func(in *Input) { in.Config.GOOS = "plan9" }, "J-1 " + KindUnknownGOOS + " plan9/amd64"},
		{"toolchain", func(in *Input) { in.Toolchain = "devel" }, "J-3 " + KindToolchain + " devel"},
		{"sizes nil", func(in *Input) { in.Config.GOARCH = "bogus" }, "J-3 " + KindSizesNil + " linux/bogus"},
		{"duplicate", func(in *Input) { in.Packages = append(in.Packages, in.Packages[1]) }, "J-3 " + KindDuplicatePackage + " syscall"},
		{"kind", func(in *Input) {
			in.Packages = append([]Package(nil), in.Packages...)
			in.Packages[1].Kind = ThirdParty
		}, "J-1 " + KindKindMismatch + " syscall"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := base()
			in := w.input(linuxAMD64)
			tc.edit(&in)
			rep := Run(in)
			wantFail(t, "setup", rep.Setup, tc.want)
			checkComplete(t, rep)
			wantUnevaluated(t, "scan", rep.Scan, ReasonSetup)
			wantHold(t, "budget", rep.Budget)
			if len(w.reads) != 0 {
				t.Errorf("reads after a setup failure: %v", w.reads)
			}
			for _, p := range rep.Packages {
				wantUnevaluated(t, p.ImportPath, p.Load, ReasonSetup)
			}
			if len(rep.Packages) == 0 {
				t.Errorf("no package results")
			}
		})
	}
}

func TestDeterministic(t *testing.T) {
	w := build(append(stdStubs("linux"), lc7(), thirdParty("example.com/forms", "1.21", []string{"os", "os/exec"}, map[string]string{"forms.go": formsSrc}))...)
	a := Run(w.input(linuxAMD64))
	b := Run(w.input(linuxAMD64))
	if !reflect.DeepEqual(a, b) {
		t.Fatal("two runs on the same input differ")
	}
	// Tarjan over sorted nodes and sorted edges, each component after what
	// it reaches: context; then example.com/forms reaches os (which reaches
	// syscall) and os/exec; then example.com/lc7 reaches os/signal.
	wantOrder := []string{"context", "syscall", "os", "os/exec", "example.com/forms", "os/signal", lc7Path}
	if !reflect.DeepEqual(a.Order, wantOrder) {
		t.Errorf("order %v, want %v", a.Order, wantOrder)
	}
}

// "unsafe" is never parsed, whatever the record says; a record that is not
// Standard fails, and its importers are not evaluated.
func TestUnsafeNotStandard(t *testing.T) {
	u := thirdParty("unsafe", "1.22", nil, map[string]string{"unsafe.go": "package unsafe\n"})
	usf := thirdParty("example.com/usf", "1.22", []string{"unsafe"}, map[string]string{"usf.go": unsafeSrc})
	w := build(u, usf)
	rep := Run(w.input(linuxAMD64))
	wantHold(t, "setup", rep.Setup)
	res := pkgOf(t, rep, "unsafe")
	wantFail(t, "unsafe", res.Load, "J-3 "+KindUnsafeNotStandard+" unsafe")
	wantUnevaluated(t, "unsafe I-6(d)", res.ImportSet, ReasonUnsafe)
	wantUnevaluated(t, "unsafe I-7", res.Header, ReasonUnsafe)
	if res.Prefilter != PrefilterNA {
		t.Errorf("prefilter %s", res.Prefilter)
	}
	wantUnevaluated(t, "usf", pkgOf(t, rep, "example.com/usf").Load, ReasonImports+"unsafe")
	if got := w.readsOf("unsafe"); len(got) != 0 {
		t.Errorf("unsafe was read: %v", got)
	}
	checkComplete(t, rep)
}

// checkComplete asserts that every check that applies has a result and
// every check that does not apply has none.
func checkComplete(t *testing.T, rep *Report) {
	t.Helper()
	for _, r := range []struct {
		name string
		res  judge.Result
	}{{"setup", rep.Setup}, {"budget", rep.Budget}, {"scan", rep.Scan}} {
		if r.res.Status == judge.Unset {
			t.Errorf("%s has no result", r.name)
		}
	}
	for _, p := range rep.Packages {
		if p.Load.Status == judge.Unset {
			t.Errorf("%s: load has no result", p.ImportPath)
		}
		app := p.Kind == ThirdParty || p.Kind == XSys
		if app != (p.ImportSet.Status != judge.Unset) || app != (p.Header.Status != judge.Unset) {
			t.Errorf("%s (%s): I-6(d) %s, I-7 %s", p.ImportPath, p.Kind, p.ImportSet.Status, p.Header.Status)
		}
		if (p.Kind == ThirdParty && p.ImportPath != "unsafe") != (p.Prefilter != PrefilterNA) {
			t.Errorf("%s (%s): prefilter %s", p.ImportPath, p.Kind, p.Prefilter)
		}
	}
}
