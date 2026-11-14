package cond

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// Input is everything the five steps of one configuration need.
type Input struct {
	Target Target
	// Packages are the closure's third-party, standard library and x/sys
	// packages, and every repository production directory (Kind Main;
	// Listed when the third call reported it). RelDir names the
	// repository directories ("" for the root).
	Packages   []*Pkg
	RelDir     map[*Pkg]string
	Exceptions []spec.EmbedException
	// Admit is asked before each package's classification; false stops
	// that unit with "budget insufficient".
	Admit func(unit string) bool
}

// Result is the outcome of the five steps.
type Result struct {
	Contexts  []ContextRecord
	Classes   map[*Pkg]map[string]Classes // package -> condition -> classes
	Steps     [6][]judge.Failure          // Steps[1..5]
	Budget    []judge.Failure             // admission refusals (J-1)
	Read      []judge.Failure             // classification failures (reads, matches, parses)
	Complete  bool                        // every package was classified under every condition
	Evaluated [6]bool                     // a step was evaluated (all of its inputs were complete)
}

// Check classifies every package under every condition and evaluates the
// five steps. A step whose inputs are incomplete (a refused admission or a
// failed classification of a package it needs) is not evaluated; the
// failures that made it so are in Budget and Read.
func Check(in Input) Result {
	res := Result{Classes: map[*Pkg]map[string]Classes{}, Complete: true}
	conds := Conditions()
	recorded := map[string]bool{}
	incomplete := map[*Pkg]bool{}
	for _, p := range in.Packages {
		unit := "classify " + p.ImportPath
		if p.Kind == Main {
			unit = "classify repo dir " + relName(in.RelDir[p])
		}
		if in.Admit != nil && !in.Admit(unit) {
			res.Budget = append(res.Budget, judge.Failure{Class: judge.J1, Item: unit, Site: "admission " + unit, Kind: "budget insufficient", Source: judge.SourceBudget})
			incomplete[p] = true
			res.Complete = false
			continue
		}
		names, err := p.Src.List()
		if err != nil {
			res.Read = append(res.Read, readOrMatchFailure(err, p.ImportPath, "list "+p.ImportPath))
			incomplete[p] = true
			res.Complete = false
			continue
		}
		res.Classes[p] = map[string]Classes{}
		for _, c := range conds {
			ctx, rec := NewContext(in.Target, c, p.Src)
			if !recorded[c.Name] {
				recorded[c.Name] = true
				res.Contexts = append(res.Contexts, rec)
			}
			cl, fails := Classify(ctx, p, names)
			if len(fails) > 0 {
				res.Read = append(res.Read, fails...)
				incomplete[p] = true
				res.Complete = false
				continue
			}
			res.Classes[p][c.Name] = cl
		}
	}
	ready := func(p *Pkg) bool { return !incomplete[p] }

	// Step 1: consistency with the third call for listed repository
	// packages and third-party packages.
	res.Evaluated[1] = true
	for _, p := range in.Packages {
		if !(p.Kind == ThirdParty || (p.Kind == Main && p.Listed)) {
			continue
		}
		if !ready(p) {
			res.Evaluated[1] = false
			continue
		}
		a := res.Classes[p]["analysis"]
		if !equal(a.GoFiles, p.GoFiles) || !equal(a.CgoFiles, p.CgoFiles) {
			res.Steps[1] = append(res.Steps[1], judge.Real(judge.J3, p.ImportPath, "step 1 "+p.ImportPath, "classification differs from the third call",
				fmt.Sprintf("tool GoFiles %v CgoFiles %v; listed GoFiles %v CgoFiles %v", a.GoFiles, a.CgoFiles, sorted(p.GoFiles), sorted(p.CgoFiles))))
		}
	}

	// Steps 2 and 4: repository production directories.
	res.Evaluated[2], res.Evaluated[4] = true, true
	type prodFile struct {
		p    *Pkg
		name string
		path string
	}
	var prodOnly []prodFile
	for _, p := range in.Packages {
		if p.Kind != Main {
			continue
		}
		if !ready(p) {
			res.Evaluated[2], res.Evaluated[4] = false, false
			continue
		}
		a := res.Classes[p]["analysis"]
		inAnalysis := map[string]bool{}
		for _, n := range append(append([]string(nil), a.GoFiles...), a.CgoFiles...) {
			inAnalysis[n] = true
		}
		seen := map[string]bool{}
		for _, c := range conds[1:] {
			cl := res.Classes[p][c.Name]
			for _, n := range append(append([]string(nil), cl.GoFiles...), cl.CgoFiles...) {
				if !inAnalysis[n] && !seen[n] {
					seen[n] = true
					prodOnly = append(prodOnly, prodFile{p, n, joinRel(in.RelDir[p], n)})
				}
			}
		}
	}
	sort.Slice(prodOnly, func(i, j int) bool { return prodOnly[i].path < prodOnly[j].path })
	if res.Evaluated[2] {
		exc := map[string]spec.EmbedException{}
		for _, e := range in.Exceptions {
			exc[e.Path] = e
		}
		found := map[string]bool{}
		for _, f := range prodOnly {
			site := "step 2 " + f.path
			b, err := f.p.Src.Read(f.name)
			if err != nil {
				res.Read = append(res.Read, readOrMatchFailure(err, f.path, site))
				res.Evaluated[2], res.Evaluated[4] = false, false
				continue
			}
			imps, _ := Imports(b)
			for _, imp := range imps {
				// "C" is cgo's pseudo-package, not a package of the
				// closure; a production-only file importing it already
				// fails step 2.
				if imp != "C" && !IsStd(imp) {
					res.Steps[4] = append(res.Steps[4], judge.Real(judge.J3, f.path, "step 4 "+f.path, "production-only file imports a non-standard package", imp))
				}
			}
			e, ok := exc[f.path]
			if !ok {
				res.Steps[2] = append(res.Steps[2], judge.Real(judge.J3, f.path, site, "production-only file not in the exception list", ""))
				continue
			}
			found[f.path] = true
			sum := sha256.Sum256(b)
			if got := hex.EncodeToString(sum[:]); got != e.SHA256 || len(b) != e.Size {
				res.Steps[2] = append(res.Steps[2], judge.Real(judge.J3, f.path, site, "exception content differs", fmt.Sprintf("%d bytes %s, list %d bytes %s", len(b), got, e.Size, e.SHA256)))
			}
			line, _, err := Constraint(b)
			if err != nil || line != e.Constraint {
				detail := fmt.Sprintf("%q, list %q", line, e.Constraint)
				if err != nil {
					detail = err.Error()
				}
				res.Steps[2] = append(res.Steps[2], judge.Real(judge.J3, f.path, site, "exception constraint line differs", detail))
			}
			if got := uniqSorted(imps); strings.Join(got, ",") != strings.Join(e.Imports, ",") {
				res.Steps[2] = append(res.Steps[2], judge.Real(judge.J3, f.path, site, "exception imports differ", fmt.Sprintf("%v, list %v", got, e.Imports)))
			}
		}
		for _, e := range in.Exceptions {
			if !found[e.Path] && res.Evaluated[2] {
				res.Steps[2] = append(res.Steps[2], judge.Real(judge.J3, e.Path, "step 2 "+e.Path, "exception file is not a production-only file", ""))
			}
		}
	}

	// Step 3: third-party packages compile the same files in every
	// production variant, and no cgo files.
	res.Evaluated[3] = true
	for _, p := range in.Packages {
		if p.Kind != ThirdParty {
			continue
		}
		if !ready(p) {
			res.Evaluated[3] = false
			continue
		}
		a := res.Classes[p]["analysis"]
		for _, c := range conds[1:] {
			cl := res.Classes[p][c.Name]
			if !equal(cl.GoFiles, a.GoFiles) || len(cl.CgoFiles) > 0 {
				res.Steps[3] = append(res.Steps[3], judge.Real(judge.J3, p.ImportPath, "step 3 "+p.ImportPath+" "+c.Name, "production variant not analysed",
					fmt.Sprintf("GoFiles %v CgoFiles %v; analysis GoFiles %v", cl.GoFiles, cl.CgoFiles, a.GoFiles)))
			}
		}
	}

	// Step 5: standard library and x/sys import only standard library,
	// golang.org/x/sys/... and, with cgo, "C".
	res.Evaluated[5] = true
	for _, p := range in.Packages {
		if p.Kind != Std && p.Kind != XSys {
			continue
		}
		if !ready(p) {
			res.Evaluated[5] = false
			continue
		}
		for _, c := range conds[1:] {
			cl := res.Classes[p][c.Name]
			for _, n := range append(append([]string(nil), cl.GoFiles...), cl.CgoFiles...) {
				for _, imp := range cl.Imports[n] {
					ok := IsStd(imp) || imp == "golang.org/x/sys" || strings.HasPrefix(imp, "golang.org/x/sys/") || (imp == "C" && c.CgoEnabled)
					if !ok {
						res.Steps[5] = append(res.Steps[5], judge.Real(judge.J3, p.ImportPath, "step 5 "+p.ImportPath+" "+n+" "+c.Name, "import outside standard library and x/sys", imp))
					}
				}
			}
		}
	}
	return res
}

func uniqSorted(s []string) []string {
	m := map[string]bool{}
	var out []string
	for _, x := range s {
		if !m[x] {
			m[x] = true
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}

func joinRel(dir, name string) string {
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

func relName(rel string) string {
	if rel == "" {
		return "."
	}
	return rel
}
