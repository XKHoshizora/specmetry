package decode

import (
	"strconv"
	"strings"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// fieldKind is the JSON type a field of a fixed definition takes.
type fieldKind uint8

const (
	fString    fieldKind = iota + 1 // string
	fBool                           // bool
	fTime                           // *time.Time: a string in RFC 3339 form
	fStrings                        // []string
	fStringMap                      // map[string]string
	fObject                         // pointer to a struct (obj)
	fObjects                        // slice of pointers to a struct (obj)
)

func (k fieldKind) want() string {
	switch k {
	case fString:
		return "string"
	case fBool:
		return "bool"
	case fTime:
		return "RFC 3339 time string"
	case fStrings:
		return "array of strings"
	case fStringMap:
		return "object of strings"
	case fObject:
		return "object"
	case fObjects:
		return "array of objects"
	}
	return "?"
}

type field struct {
	name string
	kind fieldKind
	obj  *schema
}

// schema is the field set of one fixed definition: the keys an object may
// have, each with its type.
type schema struct {
	name   string
	fields []field
}

func (s *schema) lookup(name string) *field {
	for i := range s.fields {
		if s.fields[i].name == name {
			return &s.fields[i]
		}
	}
	return nil
}

// listSchemas builds the fixed definitions of the third call (method record
// §3.1 step 2, §3.4), go1.27.3:
//
//   - package: only the fields spec.ListFields requests, with their
//     PackagePublic types (cmd/go/internal/load/pkg.go:63-147); go list
//     zeroes every other top-level field before encoding
//     (cmd/go/internal/list/list.go:458-469), and all are omitempty.
//   - module: every field of modinfo.ModulePublic
//     (cmd/go/internal/modinfo/info.go:16-36); Replace and Update are
//     ModulePublic again, Error is ModuleError, Origin is codehost.Origin,
//     Time is *time.Time.
//   - module error: modinfo.ModuleError {Err string} (info.go:38-40), in its
//     encoded object form only. Its UnmarshalJSON also accepts a bare string
//     (info.go:44-52), but the fixed encoder writes the object, so a string
//     is a type mismatch here.
//   - origin: codehost.Origin (cmd/go/internal/modfetch/codehost/
//     codehost.go:93-126).
//   - package error: the encoded form of load.PackageError, which
//     MarshalJSON writes as {ImportStack []string, Pos string, Err string}
//     (pkg.go:495-504); IsImportCycle and the unexported field are never
//     encoded and are not keys.
//
// It returns the package schema and builds fresh values on every call, so
// the package keeps no global state.
func listSchemas() (*schema, error) {
	module := &schema{name: "ModulePublic"}
	moduleError := &schema{name: "ModuleError", fields: []field{{"Err", fString, nil}}}
	origin := &schema{name: "Origin", fields: []field{
		{"VCS", fString, nil},
		{"URL", fString, nil},
		{"Subdir", fString, nil},
		{"Hash", fString, nil},
		{"TagPrefix", fString, nil},
		{"TagSum", fString, nil},
		{"Ref", fString, nil},
		{"RepoSum", fString, nil},
	}}
	module.fields = []field{
		{"Path", fString, nil},
		{"Version", fString, nil},
		{"Query", fString, nil},
		{"Versions", fStrings, nil},
		{"Replace", fObject, module},
		{"Time", fTime, nil},
		{"Update", fObject, module},
		{"Main", fBool, nil},
		{"Indirect", fBool, nil},
		{"Dir", fString, nil},
		{"GoMod", fString, nil},
		{"GoVersion", fString, nil},
		{"Retracted", fStrings, nil},
		{"Deprecated", fString, nil},
		{"Error", fObject, moduleError},
		{"Sum", fString, nil},
		{"GoModSum", fString, nil},
		{"Origin", fObject, origin},
		{"Reuse", fBool, nil},
	}
	packageError := &schema{name: "PackageError", fields: []field{
		{"ImportStack", fStrings, nil},
		{"Pos", fString, nil},
		{"Err", fString, nil},
	}}
	// The PackagePublic types of the requested fields.
	public := map[string]field{
		"ImportPath":     {"ImportPath", fString, nil},
		"Dir":            {"Dir", fString, nil},
		"Name":           {"Name", fString, nil},
		"Standard":       {"Standard", fBool, nil},
		"Module":         {"Module", fObject, module},
		"GoFiles":        {"GoFiles", fStrings, nil},
		"CgoFiles":       {"CgoFiles", fStrings, nil},
		"IgnoredGoFiles": {"IgnoredGoFiles", fStrings, nil},
		"Imports":        {"Imports", fStrings, nil},
		"ImportMap":      {"ImportMap", fStringMap, nil},
		"ForTest":        {"ForTest", fString, nil},
		"Error":          {"Error", fObject, packageError},
		"DepsErrors":     {"DepsErrors", fObjects, packageError},
		"Incomplete":     {"Incomplete", fBool, nil},
	}
	pkg := &schema{name: "Package"}
	for _, name := range spec.ListFields {
		f, ok := public[name]
		if !ok {
			return nil, errorf(spec.CallList, KindUsage, "$", 0, "requested field %q has no type in the decoder", name)
		}
		pkg.fields = append(pkg.fields, f)
	}
	return pkg, nil
}

// envSchema is the first call's definition: exactly the requested names,
// each a string (method record §3.2; envcmd/env.go:605-620 encodes a
// map[string]string).
func envSchema(names []string) *schema {
	s := &schema{name: "env"}
	for _, n := range names {
		s.fields = append(s.fields, field{n, fString, nil})
	}
	return s
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (i > 0 && c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// keyPath appends an object key to a path: ".Name" for identifiers,
// otherwise ["..."] with the key quoted.
func keyPath(path, key string) string {
	if isIdent(key) {
		return path + "." + key
	}
	return path + "[" + strconv.Quote(key) + "]"
}

func indexPath(path string, i int) string { return path + "[" + strconv.Itoa(i) + "]" }

// checkKeys is step 2 of the method record §3.1 for one object and,
// recursively, for the objects inside it whose values have the expected
// container type (a value of the wrong type is left to step 3): every key
// must be byte-equal to a field name (no escapes, no case variants), occur
// once, and no value anywhere may be an explicit null. Members are visited
// in output order; for each member the key is checked before its value.
// The first problem is returned.
func checkKeys(call spec.CallKind, n *node, s *schema, path string) *Error {
	seen := map[string]bool{}
	for _, m := range n.members {
		at := keyPath(path, m.key)
		f := s.lookup(m.raw)
		if f == nil {
			switch {
			case s.lookup(m.key) != nil:
				return errorf(call, KindKey, at, m.off, "key %q is written with escapes; %s field names are written literally", m.raw, s.name)
			case foldMatch(s, m.key) != "":
				return errorf(call, KindCase, at, m.off, "key %q differs only in case from the %s field %q", m.key, s.name, foldMatch(s, m.key))
			default:
				return errorf(call, KindKey, at, m.off, "key %q is not a field of %s", m.key, s.name)
			}
		}
		if seen[m.raw] {
			return errorf(call, KindDuplicate, at, m.off, "key %q appears more than once", m.raw)
		}
		seen[m.raw] = true
		if err := checkNulls(call, m.val, f, at); err != nil {
			return err
		}
	}
	return nil
}

// checkNulls rejects a null value and recurses into containers of the
// expected kind.
func checkNulls(call spec.CallKind, v *node, f *field, at string) *Error {
	if v.kind == nNull {
		return errorf(call, KindNull, at, v.off, "explicit null")
	}
	switch f.kind {
	case fObject:
		if v.kind == nObject {
			return checkKeys(call, v, f.obj, at)
		}
	case fObjects:
		if v.kind == nArray {
			for i, e := range v.elems {
				ep := indexPath(at, i)
				if e.kind == nNull {
					return errorf(call, KindNull, ep, e.off, "explicit null")
				}
				if e.kind == nObject {
					if err := checkKeys(call, e, f.obj, ep); err != nil {
						return err
					}
				}
			}
		}
	case fStrings:
		if v.kind == nArray {
			for i, e := range v.elems {
				if e.kind == nNull {
					return errorf(call, KindNull, indexPath(at, i), e.off, "explicit null")
				}
			}
		}
	case fStringMap:
		if v.kind == nObject {
			// Map keys are data (import paths), compared after unescaping
			// and with case: "A" and "a" are different keys.
			seen := map[string]bool{}
			for _, m := range v.members {
				mp := at + "[" + strconv.Quote(m.key) + "]"
				if seen[m.key] {
					return errorf(call, KindDuplicate, mp, m.off, "map key %q appears more than once", m.key)
				}
				seen[m.key] = true
				if m.val.kind == nNull {
					return errorf(call, KindNull, mp, m.val.off, "explicit null")
				}
			}
		}
	}
	return nil
}

func foldMatch(s *schema, key string) string {
	for _, f := range s.fields {
		if strings.EqualFold(f.name, key) {
			return f.name
		}
	}
	return ""
}

func typeErr(call spec.CallKind, at string, v *node, f *field) *Error {
	return errorf(call, KindType, at, v.off, "%s, want %s", v.kind, f.kind.want())
}

// checkTypes is step 3 of the method record §3.1: every value's JSON type
// matches its field's type. It runs only after checkKeys found nothing, so
// every key is known and no value is null.
func checkTypes(call spec.CallKind, n *node, s *schema, path string) *Error {
	for _, m := range n.members {
		at := keyPath(path, m.key)
		f := s.lookup(m.raw)
		v := m.val
		switch f.kind {
		case fString:
			if v.kind != nString {
				return typeErr(call, at, v, f)
			}
		case fBool:
			if v.kind != nBool {
				return typeErr(call, at, v, f)
			}
		case fTime:
			if v.kind != nString {
				return typeErr(call, at, v, f)
			}
			if _, err := time.Parse(time.RFC3339Nano, v.str); err != nil {
				return errorf(call, KindType, at, v.off, "string %q is not an RFC 3339 time", v.str)
			}
		case fStrings:
			if v.kind != nArray {
				return typeErr(call, at, v, f)
			}
			for i, e := range v.elems {
				if e.kind != nString {
					return errorf(call, KindType, indexPath(at, i), e.off, "%s, want string", e.kind)
				}
			}
		case fStringMap:
			if v.kind != nObject {
				return typeErr(call, at, v, f)
			}
			for _, mm := range v.members {
				if mm.val.kind != nString {
					return errorf(call, KindType, at+"["+strconv.Quote(mm.key)+"]", mm.val.off, "%s, want string", mm.val.kind)
				}
			}
		case fObject:
			if v.kind != nObject {
				return typeErr(call, at, v, f)
			}
			if err := checkTypes(call, v, f.obj, at); err != nil {
				return err
			}
		case fObjects:
			if v.kind != nArray {
				return typeErr(call, at, v, f)
			}
			for i, e := range v.elems {
				if e.kind != nObject {
					return errorf(call, KindType, indexPath(at, i), e.off, "%s, want object", e.kind)
				}
				if err := checkTypes(call, e, f.obj, indexPath(at, i)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Helpers that read a member checked by checkKeys and checkTypes.

func strs(v *node) []string {
	out := make([]string, 0, len(v.elems))
	for _, e := range v.elems {
		out = append(out, e.str)
	}
	return out
}

func strMap(v *node) map[string]string {
	out := make(map[string]string, len(v.members))
	for _, m := range v.members {
		out[m.key] = m.val.str
	}
	return out
}
