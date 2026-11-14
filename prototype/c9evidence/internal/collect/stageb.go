package collect

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/decode"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

func (res *Result) stageB(o Options) {
	res.b1(o)
	res.b2(o)
	mods := res.b3(o)
	res.b4(o, mods)
	if !res.Checks.AllHold(append(prefixed("B1/", B1Keys), prefixed("B2/", B2Fields)...)...) {
		res.Checks.Unevaluated("B5", "B1 or B2 did not all hold (configuration and tags)")
		return
	}
	res.b5(o, mods)
}

func prefixed(p string, xs []string) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = p + x
	}
	return out
}

// b1 is the semantic check of the first call (method record §3.2) except
// GOROOT and GOVERSION, which A1-c judged.
func (res *Result) b1(o Options) {
	want := map[string]string{"GOOS": o.Config.GOOS, "GOARCH": o.Config.GOARCH, "CGO_ENABLED": "0", "GOFLAGS": "", "GOWORK": "off", "GOTOOLCHAIN": "local"}
	in := sha(res.Calls[0].Stdout)
	for _, k := range B1Keys {
		id := "B1/" + k
		res.Checks.Enter(id, in)
		if got := res.Env[k]; got != want[k] {
			res.Checks.Fail(id, judge.Real(judge.J2, k, "B1 "+k, "mismatch", fmt.Sprintf("%q, want %q", got, want[k])))
			continue
		}
		res.Checks.Hold(id)
	}
}

// b2 is the semantic check of the second call (method record §3.3.3).
func (res *Result) b2(o Options) {
	c := res.Context
	in := ""
	for _, r := range res.Calls {
		if r.Spec.Kind == 2 {
			in = sha(r.Stdout)
		}
	}
	check := func(field string, c judge.Class, ok bool, detail string) {
		id := "B2/" + field
		res.Checks.Enter(id, in)
		if ok {
			res.Checks.Hold(id)
			return
		}
		res.Checks.Fail(id, judge.Real(c, field, "B2 "+field, "mismatch", detail))
	}
	check("GOOS", judge.J2, c.GOOS == o.Config.GOOS, fmt.Sprintf("%q", c.GOOS))
	check("GOARCH", judge.J2, c.GOARCH == o.Config.GOARCH, fmt.Sprintf("%q", c.GOARCH))
	check("GOROOT", judge.J2, c.GOROOT == res.Env["GOROOT"], fmt.Sprintf("%q, call 1 %q", c.GOROOT, res.Env["GOROOT"]))
	check("CgoEnabled", judge.J2, !c.CgoEnabled, "true")
	check("UseAllFiles", judge.J2, !c.UseAllFiles, "true")
	check("Compiler", judge.J2, c.Compiler == "gc", fmt.Sprintf("%q", c.Compiler))
	check("InstallSuffix", judge.J2, c.InstallSuffix == "", fmt.Sprintf("%q", c.InstallSuffix))
	check("BuildTags", judge.J2, len(c.BuildTags) == 0, fmt.Sprintf("%d elements", len(c.BuildTags)))
	check("ToolTags", judge.J3, len(c.ToolTags) != 0, "0 elements")
	check("ReleaseTags", judge.J3, len(c.ReleaseTags) != 0, "0 elements")
}

// modInfo is one dependency module of the closure, with its packages.
type modInfo struct {
	Path, Version, Dir string
	Sum, GoModSum      string
	Packages           []*decode.Package
	I2, I3             bool // I-2 and I-3 hold for the module and all its packages
}

// b3 is the semantic check of the third call (method record §3.4) with
// I-2 and I-3, and the closure (implementation plan §12.10 range).
func (res *Result) b3(o Options) map[string]*modInfo {
	ck := res.Checks
	var in string
	for _, r := range res.Calls {
		if r.Spec.Kind == 3 {
			in = sha(r.Stdout)
		}
	}
	ck.Enter("B3", in)
	var acc judge.Acc
	byPath := map[string]*decode.Package{}
	for i := range res.Packages {
		p := &res.Packages[i]
		item := p.ImportPath
		if item == "" {
			item = fmt.Sprintf("record %d", i)
		}
		bad := func(c judge.Class, kind, detail string) {
			acc.Add(judge.Real(c, item, "B3 "+item, kind, detail))
		}
		if p.ImportPath == "" {
			bad(judge.J3, "ImportPath missing", "")
		}
		if p.Name == "" {
			bad(judge.J3, "Name missing", "")
		}
		if p.ForTest != "" || strings.Contains(p.ImportPath, " [") {
			bad(judge.J3, "test variant", p.ForTest)
		}
		if p.Error != nil {
			bad(judge.J3, "Error present", p.Error.Err)
		}
		if len(p.DepsErrors) > 0 {
			bad(judge.J3, "DepsErrors present", fmt.Sprintf("%d", len(p.DepsErrors)))
		}
		if p.Incomplete {
			bad(judge.J3, "Incomplete", "")
		}
		if p.ImportPath != "" {
			if _, dup := byPath[p.ImportPath]; dup {
				bad(judge.J3, "duplicate ImportPath", "")
			}
			byPath[p.ImportPath] = p
		}
		if p.Dir == "" {
			bad(judge.J3, "Dir missing", "")
		}
		switch {
		case p.Standard:
			if p.Module != nil {
				bad(judge.J3, "standard package with Module", "")
			}
		case p.Module == nil:
			bad(judge.J3, "non-standard package without Module", "")
		case p.Module.Main:
			m := p.Module
			if m.Path == "" {
				bad(judge.J3, "main Module.Path missing", "")
			} else if m.Path != o.MainModule {
				bad(judge.J2, "main Module.Path differs", fmt.Sprintf("%q, want %q", m.Path, o.MainModule))
			}
			if filepath.Clean(m.Dir) != filepath.Clean(o.R) || m.Dir == "" {
				bad(judge.J2, "main Module.Dir is not R", fmt.Sprintf("%q, R %q", m.Dir, o.R))
			}
			for k, present := range map[string]bool{"Version": m.Version != "", "Sum": m.Sum != "", "GoModSum": m.GoModSum != "", "Replace": m.Replace != nil, "Error": m.Error != nil} {
				if present {
					bad(judge.J3, "main Module."+k+" present", "")
				}
			}
		}
	}

	// The closure: from the repository's production packages along Imports,
	// not through internal/testutil.
	isMain := func(p *decode.Package) bool { return p.Module != nil && p.Module.Main && !p.Standard }
	skip := o.MainModule + "/internal/testutil"
	var queue []string
	for _, p := range res.Packages {
		if isMain(&p) && p.ImportPath != skip && !strings.HasPrefix(p.ImportPath, skip+"/") {
			queue = append(queue, p.ImportPath)
		}
	}
	seen := map[string]bool{}
	for len(queue) > 0 {
		ip := queue[0]
		queue = queue[1:]
		if seen[ip] || ip == skip || strings.HasPrefix(ip, skip+"/") {
			continue
		}
		seen[ip] = true
		p, ok := byPath[ip]
		if !ok {
			acc.Add(judge.Real(judge.J3, ip, "B3 closure", "imported package not listed", ""))
			continue
		}
		queue = append(queue, p.Imports...)
	}
	for ip := range seen {
		res.Closure = append(res.Closure, ip)
	}
	sort.Strings(res.Closure)

	// The dependency-module branch, I-2 and I-3, for closure packages only.
	mods := map[string]*modInfo{}
	for _, ip := range res.Closure {
		p := byPath[ip]
		if p == nil || p.Standard || p.Module == nil || p.Module.Main {
			continue
		}
		m := p.Module
		bad := func(c judge.Class, kind, detail string) {
			acc.Add(judge.Real(c, ip, "B3 "+ip, kind, detail))
		}
		ok := true
		if m.Path == "" || m.Version == "" {
			bad(judge.J3, "Module.Path or Version missing", "")
			ok = false
		}
		if m.Replace != nil {
			bad(judge.J2, "Module.Replace present", "")
			ok = false
		}
		if m.Error != nil {
			bad(judge.J3, "Module.Error present", m.Error.Err)
			ok = false
		}
		if m.Dir == "" || !filepath.IsAbs(m.Dir) {
			bad(judge.J3, "Module.Dir missing or not absolute", m.Dir)
			ok = false
		}
		if m.Sum == "" || m.GoModSum == "" {
			bad(judge.J3, "Module.Sum or GoModSum missing", "")
			ok = false
		} else {
			h1, has := res.GoSum.H1(m.Path, m.Version)
			gm, hasGM := res.GoSum.GoModH1(m.Path, m.Version)
			switch {
			case !has || !hasGM:
				bad(judge.J3, "go.sum has no record for the module", m.Path+" "+m.Version)
				ok = false
			case m.Sum != h1:
				bad(judge.J2, "Module.Sum differs from go.sum", m.Sum+" vs "+h1)
				ok = false
			case m.GoModSum != gm:
				bad(judge.J2, "Module.GoModSum differs from go.sum", m.GoModSum+" vs "+gm)
				ok = false
			}
		}
		mi := mods[m.Path]
		if mi == nil {
			mi = &modInfo{Path: m.Path, Version: m.Version, Dir: m.Dir, Sum: m.Sum, GoModSum: m.GoModSum, I2: true, I3: true}
			mods[m.Path] = mi
		} else if mi.Version != m.Version || mi.Dir != m.Dir {
			bad(judge.J2, "one module path with two versions or directories", fmt.Sprintf("%s %s vs %s %s", mi.Version, mi.Dir, m.Version, m.Dir))
			mi.I2 = false
		}
		if !ok {
			mi.I2 = false
		}
		mi.Packages = append(mi.Packages, p)
		// I-3: lexical, no symbolic link resolved.
		r, err := filepath.Rel(m.Dir, p.Dir)
		i3 := err == nil && !filepath.IsAbs(r)
		if i3 {
			for _, part := range strings.Split(filepath.ToSlash(r), "/") {
				if part == ".." {
					i3 = false
				}
			}
		}
		if i3 {
			want := m.Path
			if r != "." {
				want = m.Path + "/" + filepath.ToSlash(r)
			}
			i3 = ip == want
		}
		if !i3 {
			bad(judge.J2, "I-3: package is not inside its module", fmt.Sprintf("Module.Dir %q, Dir %q", m.Dir, p.Dir))
			mi.I3 = false
		}
	}
	res.Refs.Closure = o.Rec.Derived(o.Config.String()+"/closure", "B3", []Ref{res.Refs.Calls[spec.CallList]}, map[string]any{"closure": res.Closure})
	ck.Set("B3", acc.Result())
	return mods
}

// b4 is I-4 for every dependency module of the closure whose I-2 and I-3
// hold (method record §6.2 B4).
func (res *Result) b4(o Options, mods map[string]*modInfo) {
	ck := res.Checks
	ck.Enter("B4", "")
	var acc judge.Acc
	var unevaluated []string
	paths := make([]string, 0, len(mods))
	for p := range mods {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	res.Modules = append([]string(nil), paths...)
	for _, p := range paths {
		mi := mods[p]
		if !mi.I2 || !mi.I3 {
			unevaluated = append(unevaluated, p+": I-2 or I-3 did not hold")
			continue
		}
		unit := "I-4 " + p + "@" + mi.Version
		if !o.Budget.AdmitUnit(unit) {
			unevaluated = append(unevaluated, p+": not admitted")
			continue
		}
		s := modsnap.Take(o.FS, mi.Path, mi.Version, mi.Dir, mi.Sum)
		if o.Budget.PastW(o.Budget.Clock.Now()) {
			s.Failures = append(s.Failures, judge.Failure{Class: judge.J1, Item: p, Site: unit, Kind: "finished after W", Source: judge.SourceBudget})
		}
		res.Snapshots[p] = s
		files := make([]map[string]any, 0, len(s.Files))
		for _, f := range s.Files {
			files = append(files, map[string]any{"name": f.Rel, "bytes": f.Size, "sha256": f.SHA256, "close_error": f.CloseErr})
		}
		res.Refs.I4[p] = o.Rec.Raw(o.Config.String()+"/I-4/"+p, map[string]any{"module": p, "version": mi.Version, "dir": mi.Dir, "files": files, "h1": s.H1, "go.sum h1": mi.Sum})
		for _, f := range s.Failures {
			acc.Add(f)
		}
	}
	switch {
	case acc.Failed():
		ck.Set("B4", acc.Result())
	case len(unevaluated) > 0:
		ck.Unevaluated("B4", strings.Join(unevaluated, "; "))
	default:
		ck.Hold("B4")
	}
}
