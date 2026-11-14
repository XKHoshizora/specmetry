package collect

import (
	"fmt"
	"sort"
	"strings"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/cond"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/typecheck"
)

// TypeCheck is the TypeChecker of the tool: the typecheck package.
type TypeCheck struct{}

func tcKind(k cond.Kind) typecheck.Kind {
	switch k {
	case cond.Std:
		return typecheck.Std
	case cond.XSys:
		return typecheck.XSys
	case cond.Main:
		return typecheck.Main
	}
	return typecheck.ThirdParty
}

// Check runs the loader, the type check and the scan. A package whose
// prerequisites did not hold, and every package that imports one of them
// (transitively), is left out and reported not evaluated, never as an
// import failure of its importers (method record §6.1: a check that
// depends on an unevaluated one is unevaluated, not failed again).
func (TypeCheck) Check(in TCInput) TCOutput {
	var out TCOutput
	byPath := map[string]TCPkg{}
	for _, p := range in.Packages {
		byPath[p.ImportPath] = p
	}
	excluded := map[string]string{}
	for _, p := range in.Packages {
		if p.Unavailable {
			excluded[p.ImportPath] = "its prerequisites did not hold"
		}
	}
	for changed := true; changed; {
		changed = false
		for _, p := range in.Packages {
			if _, ok := excluded[p.ImportPath]; ok {
				continue
			}
			for _, imp := range p.Imports {
				if _, bad := excluded[imp]; bad {
					excluded[p.ImportPath] = "imports " + imp + ", which is not evaluated"
					changed = true
					break
				}
			}
		}
	}
	var pkgs []typecheck.Package
	for _, p := range in.Packages {
		if _, ok := excluded[p.ImportPath]; ok {
			continue
		}
		tp := typecheck.Package{ImportPath: p.ImportPath, Dir: p.Dir, Name: p.Name, Standard: p.Standard,
			GoFiles: p.GoFiles, Imports: p.Imports, ImportMap: p.ImportMap, Kind: tcKind(p.Kind)}
		if p.HasModule {
			tp.Module = &typecheck.Module{Path: p.ModulePath, Version: p.ModuleVersion, GoVersion: p.GoVersion}
		}
		pkgs = append(pkgs, tp)
	}
	paths := make([]string, 0, len(excluded))
	for p := range excluded {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		out.Unevaluated = append(out.Unevaluated, "type check "+p+": "+excluded[p])
	}

	rep := typecheck.Run(typecheck.Input{
		Config: in.Config, Toolchain: in.Toolchain, Packages: pkgs,
		Source: func(ip, file string) ([]byte, error) { return in.Source(byPath[ip], file) },
		Admit:  typecheck.Admit(in.Admit),
	})
	take := func(what string, r judge.Result) {
		switch r.Status {
		case judge.Fail:
			out.Failures = append(out.Failures, r.Failures...)
		case judge.Unevaluated:
			out.Unevaluated = append(out.Unevaluated, what+": "+r.Reason)
		}
	}
	take("type check setup", rep.Setup)
	if rep.Budget.Status == judge.Fail {
		// The refusal itself is already a J-1 of the shared budget.
		out.Unevaluated = append(out.Unevaluated, "type check: not admitted: "+rep.Refused)
	}
	for _, pr := range rep.Packages {
		take("type check "+pr.ImportPath, pr.Load)
		take("I-6 (d) "+pr.ImportPath, pr.ImportSet)
		take("I-7 "+pr.ImportPath, pr.Header)
		if pr.Load.Status == judge.Hold && (pr.Depth != typecheck.Skipped || pr.ImportPath == "unsafe") {
			out.Checked = append(out.Checked, pr.ImportPath)
		}
	}
	for _, r := range rep.Findings {
		out.Findings = append(out.Findings, Finding{Package: r.Package, File: r.File, FileSHA256: r.FileSHA256, Constraint: r.Constraint,
			Caller: r.Caller, Object: r.Object, Line: r.Line, Col: r.Column})
	}
	for _, r := range rep.Diagnostics {
		out.Diagnostics = append(out.Diagnostics, Diagnostic{Kind: string(r.Basis), Package: r.Package, File: r.File, Caller: r.Caller,
			Object: r.Object, Line: r.Line, Col: r.Column})
	}
	for _, s := range rep.Signals {
		detail := "not a direct call"
		if s.Direct {
			detail = "direct call; arguments: " + strings.Join(s.Classes, ", ")
			if s.Spread {
				detail += " (spread)"
			}
		}
		out.Diagnostics = append(out.Diagnostics, Diagnostic{Kind: "signal", Package: s.Package, File: s.File, Caller: s.Caller,
			Object: s.Object, Line: s.Line, Col: s.Column, Detail: detail})
	}
	switch rep.Scan.Status {
	case judge.Fail:
		out.SelfCheck = append(out.SelfCheck, rep.Scan.Failures...)
	case judge.Unevaluated:
		out.Unevaluated = append(out.Unevaluated, "scan: "+rep.Scan.Reason)
	}
	for _, it := range rep.SelfCheck {
		if it.Status == typecheck.SelfMissing {
			found := false
			for _, f := range out.SelfCheck {
				if strings.Contains(f.Detail, it.Object) || f.Item == it.Object {
					found = true
				}
			}
			if !found {
				out.SelfCheck = append(out.SelfCheck, judge.Real(judge.J3, it.Object, "S self-check", "object missing", fmt.Sprint(it.Detail)))
			}
		}
	}
	return out
}
