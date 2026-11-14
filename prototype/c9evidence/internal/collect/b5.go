package collect

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/cond"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/decode"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// TCPkg is one package handed to the type checker.
type TCPkg struct {
	ImportPath, Dir, Name string
	Standard              bool
	ModulePath            string
	ModuleVersion         string
	GoVersion             string // Module.GoVersion ("" when omitted)
	HasModule             bool
	GoFiles               []string
	Imports               []string
	ImportMap             map[string]string
	Kind                  cond.Kind
	// Unavailable marks a package whose prerequisites did not hold (its
	// record failed in B3 or its module's I-4 did not hold): it is not
	// checked, and what imports it is not evaluated (method record §6.1).
	Unavailable bool
}

// TCInput is the type checker's input for one configuration.
type TCInput struct {
	Config    spec.Config
	Toolchain string // GOVERSION of the first call, e.g. "go1.27.3"
	Packages  []TCPkg
	// Source is the group P read of a package's file.
	Source func(p TCPkg, name string) ([]byte, error)
	Admit  func(unit string) bool
}

// Finding is a source reference to a frozen object (plan §4.4), with the
// entry fields I-7 derives.
type Finding struct {
	Package    string `json:"package"`
	File       string `json:"file"`
	FileSHA256 string `json:"file_sha256"`
	Constraint string `json:"constraint"`
	Caller     string `json:"caller"`
	Object     string `json:"object"`
	Line       int    `json:"line"`
	Col        int    `json:"col"`
}

// Diagnostic is an X1 or X4 reference, or a signal argument class (LQ-5);
// diagnostics are never findings.
type Diagnostic struct {
	Kind    string `json:"kind"`
	Package string `json:"package"`
	File    string `json:"file"`
	Caller  string `json:"caller"`
	Object  string `json:"object"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Detail  string `json:"detail"`
}

// TCOutput is the type checker's result.
type TCOutput struct {
	// Failures are loading and type-check failures and I-6 (d) mismatches.
	Failures []judge.Failure
	// SelfCheck are the object table's self-check failures (scan, S).
	SelfCheck   []judge.Failure
	Findings    []Finding
	Diagnostics []Diagnostic
	// Unevaluated lists packages not evaluated (dependency or budget).
	Unevaluated []string
	// Checked lists packages type-checked to the end.
	Checked []string
}

// TypeChecker is the loader, type check and scan.
type TypeChecker interface {
	Check(in TCInput) TCOutput
}

// Analysis is B5's material for records and for S and K.
type Analysis struct {
	Cond     cond.Result
	TC       TCOutput
	Std      *modsnap.Plain
	Repo     *modsnap.Plain
	Coverage []judge.Failure // I-6 (b) and (c)
}

// consumptionParents are the consumption record's parents: the third
// call (it named the packages and files) and the raw I-4 record of every
// module whose bytes the controlled read handed out.
func consumptionParents(res *Result) []Ref {
	out := []Ref{res.Refs.Calls[spec.CallList]}
	mods := make([]string, 0, len(res.Refs.I4))
	for m := range res.Refs.I4 {
		mods = append(mods, m)
	}
	sort.Strings(mods)
	for _, m := range mods {
		out = append(out, res.Refs.I4[m])
	}
	return out
}

func isXSys(ip string) bool {
	return ip == "golang.org/x/sys" || strings.HasPrefix(ip, "golang.org/x/sys/")
}

// b5 is the controlled read, the conditional difference checks, I-5 to
// I-7, the loading and the type check (method record §6.2 B5).
func (res *Result) b5(o Options, mods map[string]*modInfo) {
	ck := res.Checks
	ck.Enter("B5", "")
	cfg := o.Config.String()
	var verified []*modsnap.Snapshot
	for _, s := range res.Snapshots {
		if s.Holds() {
			verified = append(verified, s)
		}
	}
	sort.Slice(verified, func(i, j int) bool { return verified[i].Path < verified[j].Path })
	reader := modsnap.NewReader(o.FS, verified, o.Inject)
	res.Reader = reader
	an := &Analysis{Std: modsnap.NewPlain(o.FS, "std"), Repo: modsnap.NewPlain(o.FS, "repo")}
	res.Analysis = an
	var acc judge.Acc
	var unevaluated []string

	byPath := map[string]*decode.Package{}
	for i := range res.Packages {
		byPath[res.Packages[i].ImportPath] = &res.Packages[i]
	}
	inClosure := map[string]bool{}
	for _, ip := range res.Closure {
		inClosure[ip] = true
	}

	// The packages of the conditional checks.
	in := cond.Input{
		Target: cond.Target{Config: o.Config, Compiler: res.Context.Compiler, ToolTags: res.Context.ToolTags,
			ReleaseTags: res.Context.ReleaseTags, ContextSHA: sha(res.callStdout(spec.CallContext))},
		RelDir:     map[*cond.Pkg]string{},
		Exceptions: o.Exceptions,
		Admit:      o.Budget.AdmitUnit,
	}
	dirs, walkFails := cond.RepoDirs(o.FS, o.R, o.SkipDirs)
	for _, f := range walkFails {
		acc.Add(f)
	}
	listedMain := map[string]*decode.Package{}
	for _, ip := range res.Closure {
		p := byPath[ip]
		if p != nil && p.Module != nil && p.Module.Main {
			listedMain[filepath.Clean(p.Dir)] = p
		}
	}
	foundMain := map[string]bool{}
	for _, rel := range dirs {
		abs := filepath.Join(o.R, filepath.FromSlash(rel))
		cp := &cond.Pkg{ImportPath: joinImport(o.MainModule, rel), Dir: abs, Kind: cond.Main, Src: cond.PlainSource{P: an.Repo, Dir: abs, RelDir: rel}}
		if lp, ok := listedMain[filepath.Clean(abs)]; ok {
			cp.Listed, cp.GoFiles, cp.CgoFiles, cp.ImportPath = true, lp.GoFiles, lp.CgoFiles, lp.ImportPath
			foundMain[filepath.Clean(abs)] = true
		}
		in.Packages = append(in.Packages, cp)
		in.RelDir[cp] = rel
	}
	for d, lp := range listedMain {
		if !foundMain[d] {
			acc.Add(judge.Real(judge.J3, lp.ImportPath, "B5 repository walk", "listed production package not found by the walk", d))
		}
	}

	var tcPkgs []TCPkg
	for _, ip := range res.Closure {
		p := byPath[ip]
		if p == nil || (p.Module != nil && p.Module.Main) {
			continue
		}
		tp := TCPkg{ImportPath: ip, Dir: p.Dir, Name: p.Name, Standard: p.Standard, GoFiles: p.GoFiles, Imports: p.Imports, ImportMap: p.ImportMap}
		if p.Module != nil {
			tp.HasModule, tp.ModulePath, tp.ModuleVersion, tp.GoVersion = true, p.Module.Path, p.Module.Version, p.Module.GoVersion
		}
		switch {
		case p.Standard:
			tp.Kind = cond.Std
			in.Packages = append(in.Packages, &cond.Pkg{ImportPath: ip, Dir: p.Dir, Kind: cond.Std, Src: cond.PlainSource{P: an.Std, Dir: p.Dir, RelDir: ip}})
		default:
			tp.Kind = cond.ThirdParty
			if isXSys(ip) {
				tp.Kind = cond.XSys
			}
			mi := mods[p.Module.Path]
			s := res.Snapshots[p.Module.Path]
			if mi == nil || !mi.I2 || !mi.I3 || s == nil || !s.Holds() {
				tp.Unavailable = true
				unevaluated = append(unevaluated, ip+": its module's I-2, I-3 or I-4 did not hold")
				break
			}
			rel, _ := filepath.Rel(s.Dir, p.Dir)
			rel = filepath.ToSlash(rel)
			if rel == "." {
				rel = ""
			}
			in.Packages = append(in.Packages, &cond.Pkg{ImportPath: ip, Dir: p.Dir, Kind: tp.Kind, Listed: true, GoFiles: p.GoFiles, CgoFiles: p.CgoFiles,
				Src: cond.ModuleSource{R: reader, M: s, RelDir: rel}})
		}
		tcPkgs = append(tcPkgs, tp)
	}

	an.Cond = cond.Check(in)
	for step := 1; step <= 5; step++ {
		for _, f := range an.Cond.Steps[step] {
			acc.Add(f)
		}
		if !an.Cond.Evaluated[step] {
			unevaluated = append(unevaluated, fmt.Sprintf("conditional step %d", step))
		}
	}
	for _, f := range an.Cond.Read {
		acc.Add(f)
	}
	for _, f := range an.Cond.Budget {
		unevaluated = append(unevaluated, f.Item)
	}
	for _, c := range an.Cond.Contexts {
		res.Refs.Contexts = append(res.Refs.Contexts, o.Rec.Derived(cfg+"/context/"+c.Condition, "B5 conditional", []Ref{res.Refs.Calls[spec.CallContext]}, c))
	}

	// I-6 (c): every candidate of a verified package was judged under every
	// condition.
	for cp, byCond := range an.Cond.Classes {
		if cp.Kind != cond.ThirdParty && cp.Kind != cond.XSys {
			continue
		}
		names, err := cp.Src.List()
		if err != nil {
			continue
		}
		var cands []string
		for _, n := range names {
			if cond.IsCandidate(n) {
				cands = append(cands, n)
			}
		}
		for cname, cl := range byCond {
			if strings.Join(cands, ",") != strings.Join(sortedCopy(cl.Judged), ",") {
				an.Coverage = append(an.Coverage, judge.Real(judge.J3, cp.ImportPath, "I-6 (c) "+cname, "a candidate was not judged", fmt.Sprintf("candidates %v judged %v", cands, cl.Judged)))
			}
		}
	}

	// Loading, type check and scan.
	if o.TypeCheck == nil {
		unevaluated = append(unevaluated, "type check not run")
	} else {
		an.TC = o.TypeCheck.Check(TCInput{
			Config: o.Config, Toolchain: res.Env["GOVERSION"], Packages: tcPkgs, Admit: o.Budget.AdmitUnit,
			Source: func(p TCPkg, name string) ([]byte, error) {
				if p.Standard {
					return an.Std.Read(filepath.Join(p.Dir, name), p.ImportPath+"/"+name, modsnap.GroupP, "parse")
				}
				s := res.Snapshots[p.ModulePath]
				rel, _ := filepath.Rel(s.Dir, filepath.Join(p.Dir, name))
				return reader.Read(s, filepath.ToSlash(rel), modsnap.GroupP, "parse")
			},
		})
		for _, f := range an.TC.Failures {
			acc.Add(f)
		}
		unevaluated = append(unevaluated, an.TC.Unevaluated...)
		// I-6 (b): every GoFiles name of a checked verified package was
		// consumed through the controlled read for group P.
		checked := map[string]bool{}
		for _, ip := range an.TC.Checked {
			checked[ip] = true
		}
		for _, tp := range tcPkgs {
			if tp.Standard || tp.Unavailable || !checked[tp.ImportPath] {
				continue
			}
			s := res.Snapshots[tp.ModulePath]
			for _, n := range tp.GoFiles {
				rel, _ := filepath.Rel(s.Dir, filepath.Join(tp.Dir, n))
				if !reader.WasConsumed(s, filepath.ToSlash(rel), modsnap.GroupP) {
					an.Coverage = append(an.Coverage, judge.Real(judge.J3, tp.ImportPath, "I-6 (b)", "a GoFiles file was not consumed", n))
				}
			}
		}
	}
	for _, f := range an.Coverage {
		acc.Add(f)
	}
	for _, f := range reader.Failures() {
		acc.Add(f)
	}
	for _, f := range an.Std.Failures() {
		acc.Add(f)
	}
	for _, f := range an.Repo.Failures() {
		acc.Add(f)
	}
	acc = dedupe(acc)
	res.Refs.Consumption = o.Rec.Derived(cfg+"/consumption", "B5 I-5", consumptionParents(res), map[string]any{
		"controlled": reader.Consumed(), "std": an.Std.Consumed(), "repo": an.Repo.Consumed(), "injected": reader.Injections(),
	})
	switch {
	case acc.Failed():
		ck.Set("B5", acc.Result())
	case len(unevaluated) > 0:
		sort.Strings(unevaluated)
		ck.Unevaluated("B5", strings.Join(unevaluated, "; "))
	default:
		ck.Hold("B5")
	}
}

// dedupe drops failures reported twice through two paths (a controlled
// read failure is both returned to the classifier and kept by the reader).
func dedupe(a judge.Acc) judge.Acc {
	seen := map[string]bool{}
	var out judge.Acc
	for _, f := range a.Failures() {
		k := fmt.Sprintf("%d|%s|%s|%s|%s|%d|%s", f.Class, f.Item, f.Site, f.Kind, f.Source, f.Inject, f.Detail)
		if !seen[k] {
			seen[k] = true
			out.Add(f)
		}
	}
	return out
}

func (res *Result) callStdout(k spec.CallKind) []byte {
	for _, r := range res.Calls {
		if r.Spec.Kind == k {
			return r.Stdout
		}
	}
	return nil
}

func joinImport(mod, rel string) string {
	if rel == "" {
		return mod
	}
	return mod + "/" + rel
}

func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

// scanAndCompare is S and K: S holds when the scan ran to the end and the
// table's self-check held; K compares the findings with the candidates and
// reports the closure's watched packages (plan §4.4, LQ-6).
func (res *Result) scanAndCompare(o Options) {
	ck := res.Checks
	cfg := o.Config.String()
	an := res.Analysis
	fref := o.Rec.Derived(cfg+"/findings", "S", []Ref{res.Refs.Consumption, res.Refs.Calls[spec.CallList]}, map[string]any{"findings": an.TC.Findings, "diagnostics": an.TC.Diagnostics})
	res.Refs.Findings = fref
	ck.Enter("S", fref.SHA256)
	if len(an.TC.SelfCheck) > 0 {
		ck.Fail("S", an.TC.SelfCheck...)
		ck.Unevaluated("K", "S failed")
		return
	}
	ck.Hold("S")

	ck.Enter("K", fref.SHA256)
	type key struct {
		Package, File, Caller, Object string
		Line                          int
	}
	want := map[key]string{}
	for _, c := range o.Candidates {
		want[key{c.Package, c.File, c.Caller, c.Object, c.Line}] = c.ID
	}
	got := map[key]bool{}
	var diff []string
	for _, f := range an.TC.Findings {
		k := key{f.Package, f.File, f.Caller, f.Object, f.Line}
		if got[k] {
			continue // one entry per position; columns and counts are not compared
		}
		got[k] = true
		if _, ok := want[k]; !ok {
			diff = append(diff, fmt.Sprintf("extra finding %s %s %s %s:%d", f.Object, f.Caller, f.Package, f.File, f.Line))
		}
	}
	for k, id := range want {
		if !got[k] {
			diff = append(diff, fmt.Sprintf("missing candidate %s (%s %s %s:%d)", id, k.Object, k.Caller, k.File, k.Line))
		}
	}
	for _, ip := range res.Closure {
		for _, w := range o.Watch {
			if ip == w || strings.HasPrefix(ip, w+"/") {
				diff = append(diff, "watched package in the closure: "+ip)
			}
		}
	}
	sort.Strings(diff)
	res.Diff = diff
	res.Consistent = len(diff) == 0
	res.Refs.Comparison = o.Rec.Derived(cfg+"/comparison", "K", []Ref{fref}, map[string]any{"consistent": res.Consistent, "differences": diff})
	ck.Hold("K")
}
