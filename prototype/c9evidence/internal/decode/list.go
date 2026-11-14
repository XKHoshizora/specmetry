package decode

import (
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// Package is one decoded record of the third call: the fields spec.ListFields
// requests, with the types of load.PackagePublic (go1.27.3,
// cmd/go/internal/load/pkg.go:63-147). An omitted key leaves the Go zero
// value (method record §3.1 step 4): "", false, a nil slice, a nil map or a
// nil pointer. A key that was present with an empty array or object gives a
// non-nil empty value. Present records which keys the record had, so the
// caller can keep the raw facts.
type Package struct {
	ImportPath     string
	Dir            string
	Name           string
	Standard       bool
	Module         *Module
	GoFiles        []string
	CgoFiles       []string
	IgnoredGoFiles []string
	Imports        []string
	ImportMap      map[string]string
	ForTest        string
	Error          *PackageError
	DepsErrors     []PackageError
	Incomplete     bool

	Present map[string]bool
}

// Module is a decoded modinfo.ModulePublic (cmd/go/internal/modinfo/
// info.go:16-36): every field is allowed. Time is kept as the string the
// output had, after checking it parses as RFC 3339. Present records which
// keys the object had.
type Module struct {
	Path       string
	Version    string
	Query      string
	Versions   []string
	Replace    *Module
	Time       string
	Update     *Module
	Main       bool
	Indirect   bool
	Dir        string
	GoMod      string
	GoVersion  string
	Retracted  []string
	Deprecated string
	Error      *ModuleError
	Sum        string
	GoModSum   string
	Origin     *Origin
	Reuse      bool

	Present map[string]bool
}

// ModuleError is a decoded modinfo.ModuleError (info.go:38-40).
type ModuleError struct {
	Err     string
	Present map[string]bool
}

// Origin is a decoded codehost.Origin (cmd/go/internal/modfetch/codehost/
// codehost.go:93-126).
type Origin struct {
	VCS       string
	URL       string
	Subdir    string
	Hash      string
	TagPrefix string
	TagSum    string
	Ref       string
	RepoSum   string
	Present   map[string]bool
}

// PackageError is the encoded form of load.PackageError: its MarshalJSON
// writes ImportStack, Pos and Err (pkg.go:495-504).
type PackageError struct {
	ImportStack []string
	Pos         string
	Err         string
	Present     map[string]bool
}

// DecodePackages decodes the output of the third call, go list -e -deps
// -json=<spec.ListFields> -tags nowebui ./... (method record §3.4). The fixed
// cmd/go writes each record with json.MarshalIndent and a tab indent,
// followed by one LF (cmd/go/internal/list/list.go:456-476).
//
// The decode order is that of the method record §3.1, and the first
// problem found is returned:
//
//  1. Form, over the whole output: a stream of JSON objects up to EOF with
//     only JSON whitespace (space, TAB, LF, CR) around and between them.
//     A first non-whitespace byte that does not open an object is
//     KindForm; such a byte after at least one object is KindTrailing; a
//     malformed object is KindSyntax; an output with no object at all
//     (zero bytes or only whitespace) is KindCount, because the repository's
//     ./... always yields its own packages and an empty stream must not
//     read as "no packages observed".
//  2. Keys, records in output order and members in output order: a record
//     may have only the requested fields, every nested object only the
//     fields of its fixed definition (KindKey for other keys and escaped
//     keys, KindCase for case variants), each key at most once
//     (KindDuplicate, also for ImportMap keys), and no value anywhere may
//     be null (KindNull).
//  3. Types, in the same order (KindType).
//
// Omission is not an error at this layer, and no branch or semantic rule
// of §3.4 is applied: a package without Module, a dependency module
// without Sum, or a record with Error decodes successfully.
func DecodePackages(out []byte) ([]Package, error) {
	const call = spec.CallList
	pkgSchema, err := listSchemas()
	if err != nil {
		return nil, err
	}

	// Step 1: form.
	var objs []*node
	p := &parser{data: out}
	for {
		p.skipWS()
		if p.pos >= len(out) {
			break
		}
		if out[p.pos] != '{' {
			if len(objs) == 0 {
				return nil, errorf(call, KindForm, "$", p.pos, "output begins with %q, want '{'", out[p.pos])
			}
			return nil, errorf(call, KindTrailing, indexPath("$", len(objs)), p.pos, "byte %q after record %d, want '{' or the end", out[p.pos], len(objs)-1)
		}
		o, serr := p.value()
		if serr != nil {
			return nil, errorf(call, KindSyntax, indexPath("$", len(objs)), serr.off, "%s", serr.msg)
		}
		objs = append(objs, o)
	}
	if len(objs) == 0 {
		return nil, errorf(call, KindCount, "$", 0, "no JSON object in %d bytes of output", len(out))
	}

	// Steps 2 and 3.
	for i, o := range objs {
		if err := checkKeys(call, o, pkgSchema, indexPath("$", i)); err != nil {
			return nil, err
		}
	}
	for i, o := range objs {
		if err := checkTypes(call, o, pkgSchema, indexPath("$", i)); err != nil {
			return nil, err
		}
	}

	pkgs := make([]Package, 0, len(objs))
	for _, o := range objs {
		pkgs = append(pkgs, buildPackage(o))
	}
	return pkgs, nil
}

func presentKeys(n *node) map[string]bool {
	m := make(map[string]bool, len(n.members))
	for _, mm := range n.members {
		m[mm.raw] = true
	}
	return m
}

func buildPackage(n *node) Package {
	p := Package{Present: presentKeys(n)}
	for _, m := range n.members {
		v := m.val
		switch m.raw {
		case "ImportPath":
			p.ImportPath = v.str
		case "Dir":
			p.Dir = v.str
		case "Name":
			p.Name = v.str
		case "Standard":
			p.Standard = v.b
		case "Module":
			p.Module = buildModule(v)
		case "GoFiles":
			p.GoFiles = strs(v)
		case "CgoFiles":
			p.CgoFiles = strs(v)
		case "IgnoredGoFiles":
			p.IgnoredGoFiles = strs(v)
		case "Imports":
			p.Imports = strs(v)
		case "ImportMap":
			p.ImportMap = strMap(v)
		case "ForTest":
			p.ForTest = v.str
		case "Error":
			e := buildPackageError(v)
			p.Error = &e
		case "DepsErrors":
			p.DepsErrors = make([]PackageError, 0, len(v.elems))
			for _, e := range v.elems {
				p.DepsErrors = append(p.DepsErrors, buildPackageError(e))
			}
		case "Incomplete":
			p.Incomplete = v.b
		}
	}
	return p
}

func buildModule(n *node) *Module {
	m := &Module{Present: presentKeys(n)}
	for _, mm := range n.members {
		v := mm.val
		switch mm.raw {
		case "Path":
			m.Path = v.str
		case "Version":
			m.Version = v.str
		case "Query":
			m.Query = v.str
		case "Versions":
			m.Versions = strs(v)
		case "Replace":
			m.Replace = buildModule(v)
		case "Time":
			m.Time = v.str
		case "Update":
			m.Update = buildModule(v)
		case "Main":
			m.Main = v.b
		case "Indirect":
			m.Indirect = v.b
		case "Dir":
			m.Dir = v.str
		case "GoMod":
			m.GoMod = v.str
		case "GoVersion":
			m.GoVersion = v.str
		case "Retracted":
			m.Retracted = strs(v)
		case "Deprecated":
			m.Deprecated = v.str
		case "Error":
			m.Error = &ModuleError{Err: memberStr(v, "Err"), Present: presentKeys(v)}
		case "Sum":
			m.Sum = v.str
		case "GoModSum":
			m.GoModSum = v.str
		case "Origin":
			m.Origin = &Origin{
				VCS:       memberStr(v, "VCS"),
				URL:       memberStr(v, "URL"),
				Subdir:    memberStr(v, "Subdir"),
				Hash:      memberStr(v, "Hash"),
				TagPrefix: memberStr(v, "TagPrefix"),
				TagSum:    memberStr(v, "TagSum"),
				Ref:       memberStr(v, "Ref"),
				RepoSum:   memberStr(v, "RepoSum"),
				Present:   presentKeys(v),
			}
		case "Reuse":
			m.Reuse = v.b
		}
	}
	return m
}

func buildPackageError(n *node) PackageError {
	e := PackageError{Present: presentKeys(n)}
	for _, m := range n.members {
		switch m.raw {
		case "ImportStack":
			e.ImportStack = strs(m.val)
		case "Pos":
			e.Pos = m.val.str
		case "Err":
			e.Err = m.val.str
		}
	}
	return e
}

// memberStr returns the string value of a checked member, or "" when the
// key is absent.
func memberStr(n *node, key string) string {
	for _, m := range n.members {
		if m.raw == key {
			return m.val.str
		}
	}
	return ""
}
