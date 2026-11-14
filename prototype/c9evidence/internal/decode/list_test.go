package decode

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// Mirrors of the go1.27.3 definitions, with the same field names, types and
// tags, so encoding/json writes what go list writes for them.

type mOrigin struct {
	VCS       string `json:",omitempty"`
	URL       string `json:",omitempty"`
	Subdir    string `json:",omitempty"`
	Hash      string `json:",omitempty"`
	TagPrefix string `json:",omitempty"`
	TagSum    string `json:",omitempty"`
	Ref       string `json:",omitempty"`
	RepoSum   string `json:",omitempty"`
}

type mModuleError struct {
	Err string
}

type mModule struct {
	Path       string        `json:",omitempty"`
	Version    string        `json:",omitempty"`
	Query      string        `json:",omitempty"`
	Versions   []string      `json:",omitempty"`
	Replace    *mModule      `json:",omitempty"`
	Time       *time.Time    `json:",omitempty"`
	Update     *mModule      `json:",omitempty"`
	Main       bool          `json:",omitempty"`
	Indirect   bool          `json:",omitempty"`
	Dir        string        `json:",omitempty"`
	GoMod      string        `json:",omitempty"`
	GoVersion  string        `json:",omitempty"`
	Retracted  []string      `json:",omitempty"`
	Deprecated string        `json:",omitempty"`
	Error      *mModuleError `json:",omitempty"`
	Sum        string        `json:",omitempty"`
	GoModSum   string        `json:",omitempty"`
	Origin     *mOrigin      `json:",omitempty"`
	Reuse      bool          `json:",omitempty"`
}

// mPackageError marshals like load.PackageError (pkg.go:495-504).
type mPackageError struct {
	ImportStack []string
	Pos         string
	Err         string
}

func (e *mPackageError) MarshalJSON() ([]byte, error) {
	stack := e.ImportStack
	if stack == nil {
		stack = []string{} // ImportStack.Pkgs never returns nil
	}
	return json.Marshal(struct {
		ImportStack []string
		Pos         string
		Err         string
	}{stack, e.Pos, e.Err})
}

// mPackage has the requested PackagePublic fields in PackagePublic's order.
type mPackage struct {
	Dir            string            `json:",omitempty"`
	ImportPath     string            `json:",omitempty"`
	Name           string            `json:",omitempty"`
	ForTest        string            `json:",omitempty"`
	Module         *mModule          `json:",omitempty"`
	Standard       bool              `json:",omitempty"`
	Incomplete     bool              `json:",omitempty"`
	GoFiles        []string          `json:",omitempty"`
	CgoFiles       []string          `json:",omitempty"`
	IgnoredGoFiles []string          `json:",omitempty"`
	Imports        []string          `json:",omitempty"`
	ImportMap      map[string]string `json:",omitempty"`
	Error          *mPackageError    `json:",omitempty"`
	DepsErrors     []*mPackageError  `json:",omitempty"`
}

// encodeList writes records the way go list -json does (list.go:470-476).
func encodeList(t *testing.T, ps ...*mPackage) []byte {
	t.Helper()
	var buf bytes.Buffer
	for _, p := range ps {
		b, err := json.MarshalIndent(p, "", "\t")
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(b)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

// TestListEncoderOutput decodes the encoder's own output for records of the
// three branches and records with every nested definition, and compares
// every field, the zero values of omitted keys and the Present sets.
func TestListEncoderOutput(t *testing.T) {
	tm := time.Date(2026, 8, 1, 10, 20, 30, 123000000, time.FixedZone("x", -7*3600))
	std := &mPackage{
		Dir: "/go/src/net/http", ImportPath: "net/http", Name: "http", Standard: true,
		GoFiles: []string{"a.go"}, Imports: []string{"vendor/x"}, ImportMap: map[string]string{"x": "vendor/x"},
	}
	main := &mPackage{
		Dir: "/r/cmd", ImportPath: spec.MainModule + "/cmd", Name: "main",
		Module:  &mModule{Path: spec.MainModule, Main: true, Dir: "/r", GoMod: "/r/go.mod", GoVersion: "1.27"},
		GoFiles: []string{"main.go"}, Imports: []string{"modernc.org/libc"},
	}
	dep := &mPackage{
		Dir: "/m/libc", ImportPath: "modernc.org/libc", Name: "libc",
		Module: &mModule{
			Path: "modernc.org/libc", Version: "v1.77.1", Time: &tm, Indirect: true, Dir: "/m/libc",
			GoMod: "/m/cache/libc.mod", Sum: "h1:s<&>=", GoModSum: "h1:g=",
		},
		GoFiles: []string{"libc.go"}, IgnoredGoFiles: []string{"libc_unix.go"}, Imports: []string{"os/exec"},
	}
	odd := &mPackage{
		ImportPath: "p\xff", Name: "p", ForTest: "p", Incomplete: true,
		Module: &mModule{
			Path: "a", Version: "v1", Query: "q", Versions: []string{"v0", "v1"},
			Replace:   &mModule{Path: "../a", Dir: "/x", Error: &mModuleError{Err: "e"}},
			Update:    &mModule{Path: "a", Version: "v2"},
			Retracted: []string{"r"}, Deprecated: "d", Reuse: true,
			Error:  &mModuleError{},
			Origin: &mOrigin{VCS: "git", URL: "u", Subdir: "s", Hash: "h", TagPrefix: "tp", TagSum: "ts", Ref: "r", RepoSum: "rs"},
		},
		CgoFiles:   []string{"c.go"},
		Error:      &mPackageError{Err: "boom"},
		DepsErrors: []*mPackageError{{ImportStack: []string{"p", "q"}, Pos: "q.go:1:2", Err: "x"}, {Err: "y"}},
	}
	out := encodeList(t, std, main, dep, odd)
	got, err := DecodePackages(out)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}

	tmText := tm.Format(time.RFC3339Nano)
	want := []Package{
		{
			Dir: "/go/src/net/http", ImportPath: "net/http", Name: "http", Standard: true,
			GoFiles: []string{"a.go"}, Imports: []string{"vendor/x"}, ImportMap: map[string]string{"x": "vendor/x"},
			Present: present("Dir", "ImportPath", "Name", "Standard", "GoFiles", "Imports", "ImportMap"),
		},
		{
			Dir: "/r/cmd", ImportPath: spec.MainModule + "/cmd", Name: "main",
			Module: &Module{
				Path: spec.MainModule, Main: true, Dir: "/r", GoMod: "/r/go.mod", GoVersion: "1.27",
				Present: present("Path", "Main", "Dir", "GoMod", "GoVersion"),
			},
			GoFiles: []string{"main.go"}, Imports: []string{"modernc.org/libc"},
			Present: present("Dir", "ImportPath", "Name", "Module", "GoFiles", "Imports"),
		},
		{
			Dir: "/m/libc", ImportPath: "modernc.org/libc", Name: "libc",
			Module: &Module{
				Path: "modernc.org/libc", Version: "v1.77.1", Time: tmText, Indirect: true, Dir: "/m/libc",
				GoMod: "/m/cache/libc.mod", Sum: "h1:s<&>=", GoModSum: "h1:g=",
				Present: present("Path", "Version", "Time", "Indirect", "Dir", "GoMod", "Sum", "GoModSum"),
			},
			GoFiles: []string{"libc.go"}, IgnoredGoFiles: []string{"libc_unix.go"}, Imports: []string{"os/exec"},
			Present: present("Dir", "ImportPath", "Name", "Module", "GoFiles", "IgnoredGoFiles", "Imports"),
		},
		{
			ImportPath: "p\uFFFD", Name: "p", ForTest: "p", Incomplete: true,
			Module: &Module{
				Path: "a", Version: "v1", Query: "q", Versions: []string{"v0", "v1"},
				Replace: &Module{
					Path: "../a", Dir: "/x", Error: &ModuleError{Err: "e", Present: present("Err")},
					Present: present("Path", "Dir", "Error"),
				},
				Update:    &Module{Path: "a", Version: "v2", Present: present("Path", "Version")},
				Retracted: []string{"r"}, Deprecated: "d", Reuse: true,
				Error: &ModuleError{Err: "", Present: present("Err")},
				Origin: &Origin{
					VCS: "git", URL: "u", Subdir: "s", Hash: "h", TagPrefix: "tp", TagSum: "ts", Ref: "r", RepoSum: "rs",
					Present: present("VCS", "URL", "Subdir", "Hash", "TagPrefix", "TagSum", "Ref", "RepoSum"),
				},
				Present: present("Path", "Version", "Query", "Versions", "Replace", "Update", "Retracted", "Deprecated", "Error", "Origin", "Reuse"),
			},
			CgoFiles: []string{"c.go"},
			Error:    &PackageError{ImportStack: []string{}, Pos: "", Err: "boom", Present: present("ImportStack", "Pos", "Err")},
			DepsErrors: []PackageError{
				{ImportStack: []string{"p", "q"}, Pos: "q.go:1:2", Err: "x", Present: present("ImportStack", "Pos", "Err")},
				{ImportStack: []string{}, Pos: "", Err: "y", Present: present("ImportStack", "Pos", "Err")},
			},
			Present: present("ImportPath", "Name", "ForTest", "Module", "Incomplete", "CgoFiles", "Error", "DepsErrors"),
		},
	}
	if len(got) != len(want) {
		t.Fatalf("%d packages", len(got))
	}
	for i := range want {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Errorf("package %d:\ngot  %s\nwant %s", i, describe(got[i]), describe(want[i]))
		}
	}
	// Omitted keys are zero values (method record §3.1 step 4).
	if got[0].Module != nil || got[1].Module.Version != "" || got[2].Module.GoVersion != "" || got[0].CgoFiles != nil || got[1].ImportMap != nil {
		t.Error("omitted keys must decode to zero values")
	}
}

// TestPackageSchemaIsListFields: a record may carry exactly the requested
// fields (method record §3.1 step 2).
func TestPackageSchemaIsListFields(t *testing.T) {
	s, err := listSchemas()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range s.fields {
		names = append(names, f.name)
	}
	if !slices.Equal(names, spec.ListFields) || len(names) != 14 {
		t.Fatalf("package fields %v, want %v", names, spec.ListFields)
	}
	mod := s.lookup("Module").obj
	if len(mod.fields) != 19 || mod.lookup("Replace").obj != mod || mod.lookup("Update").obj != mod {
		t.Fatalf("ModulePublic schema: %d fields", len(mod.fields))
	}
	// Every field a decoded type stores is a schema field and vice versa.
	for _, c := range []struct {
		s *schema
		t reflect.Type
	}{
		{s, reflect.TypeOf(Package{})},
		{mod, reflect.TypeOf(Module{})},
		{mod.lookup("Error").obj, reflect.TypeOf(ModuleError{})},
		{mod.lookup("Origin").obj, reflect.TypeOf(Origin{})},
		{s.lookup("Error").obj, reflect.TypeOf(PackageError{})},
	} {
		var fields []string
		for i := 0; i < c.t.NumField(); i++ {
			if n := c.t.Field(i).Name; n != "Present" {
				fields = append(fields, n)
			}
		}
		var schemaNames []string
		for _, f := range c.s.fields {
			schemaNames = append(schemaNames, f.name)
		}
		slices.Sort(fields)
		slices.Sort(schemaNames)
		if !slices.Equal(fields, schemaNames) {
			t.Errorf("%s: type fields %v, schema %v", c.s.name, fields, schemaNames)
		}
	}
}

func listErr(t *testing.T, in string) *Error {
	t.Helper()
	_, err := DecodePackages([]byte(in))
	var de *Error
	if !errors.As(err, &de) {
		t.Fatalf("want *Error, got %v", err)
	}
	if de.Call != spec.CallList {
		t.Fatalf("call %v", de.Call)
	}
	return de
}

func TestListErrors(t *testing.T) {
	cases := []struct {
		name string
		in   string
		kind Kind
		at   string
	}{
		{"origin-unknown", `{"Module":{"Origin":{"VCS":"git","Branch":"main"}}}`, KindKey, "$[0].Module.Origin.Branch"},
		{"origin-null", `{"Module":{"Origin":{"VCS":null}}}`, KindNull, "$[0].Module.Origin.VCS"},
		{"replace-unknown", `{"Module":{"Replace":{"Path":"x","Bad":1}}}`, KindKey, "$[0].Module.Replace.Bad"},
		{"replace-null", `{"Module":{"Replace":null}}`, KindNull, "$[0].Module.Replace"},
		{"module-error-unknown", `{"Module":{"Error":{"Err":"e","Code":1}}}`, KindKey, "$[0].Module.Error.Code"},
		{"module-error-case", `{"Module":{"Error":{"err":"e"}}}`, KindCase, "$[0].Module.Error.err"},
		{"depserrors-null-element", `{"DepsErrors":[null]}`, KindNull, "$[0].DepsErrors[0]"},
		{"depserrors-element-string", `{"DepsErrors":["x"]}`, KindType, "$[0].DepsErrors[0]"},
		{"depserrors-object", `{"DepsErrors":{}}`, KindType, "$[0].DepsErrors"},
		{"depserrors-dup", `{"DepsErrors":[{"Err":"a","Err":"b"}]}`, KindDuplicate, "$[0].DepsErrors[0].Err"},
		{"importstack-number", `{"Error":{"ImportStack":[1]}}`, KindType, "$[0].Error.ImportStack[0]"},
		{"importstack-null", `{"Error":{"ImportStack":null}}`, KindNull, "$[0].Error.ImportStack"},
		{"importmap-null-value", `{"ImportMap":{"a":null}}`, KindNull, `$[0].ImportMap["a"]`},
		{"importmap-array", `{"ImportMap":["a"]}`, KindType, "$[0].ImportMap"},
		{"versions-string", `{"Module":{"Versions":"v1"}}`, KindType, "$[0].Module.Versions"},
		{"main-string", `{"Module":{"Main":"true"}}`, KindType, "$[0].Module.Main"},
		{"time-number", `{"Module":{"Time":0}}`, KindType, "$[0].Module.Time"},
		{"time-no-zone", `{"Module":{"Time":"2026-01-02T03:04:05"}}`, KindType, "$[0].Module.Time"},
		{"escaped-field-key", `{"Imp\u006frtPath":"p"}`, KindKey, "$[0].ImportPath"},
		{"unrequested-field", `{"ImportPath":"p","Match":["./..."]}`, KindKey, "$[0].Match"},
		{"unrequested-case", `{"ImportPath":"p","deps":[]}`, KindKey, "$[0].deps"},
		{"dup-after-type-error", `{"Standard":"x","Name":"a","Name":"b"}`, KindDuplicate, "$[0].Name"},
		{"null-in-second-record", "{}\n{\"Name\":null}", KindNull, "$[1].Name"},
		{"number-top-level", "1\n", KindForm, "$"},
		{"bad-literal", `{"Standard":tru}`, KindSyntax, "$[0]"},
		{"lone-surrogate", `{"Name":"\ud800"}`, KindSyntax, "$[0]"},
		{"invalid-utf8", "{\"Name\":\"\xff\"}", KindSyntax, "$[0]"},
		{"control-byte", "{\"Name\":\"a\tb\"}", KindSyntax, "$[0]"},
		{"missing-comma", "{\"Name\":\"a\" \"Dir\":\"b\"}", KindSyntax, "$[0]"},
		{"garbage-between", "{}\n,{}\n", KindTrailing, "$[1]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := listErr(t, c.in)
			if e.Kind != c.kind || e.At != c.at {
				t.Fatalf("got %s at %s (%s), want %s at %s", e.Kind, e.At, e.Detail, c.kind, c.at)
			}
		})
	}
}

// TestNestedOmissionsAreZero: omission inside nested definitions is a zero
// value too (method record §3.1 step 4), and an empty object is present.
func TestNestedOmissionsAreZero(t *testing.T) {
	got, err := DecodePackages([]byte(`{"Module":{"Error":{},"Origin":{"VCS":"git"},"Replace":{}},"Error":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	want := Package{
		Module: &Module{
			Error:   &ModuleError{Present: map[string]bool{}},
			Origin:  &Origin{VCS: "git", Present: present("VCS")},
			Replace: &Module{Present: map[string]bool{}},
			Present: present("Error", "Origin", "Replace"),
		},
		Error:   &PackageError{Present: map[string]bool{}},
		Present: present("Module", "Error"),
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("got %s\nwant %s", describe(got), describe(want))
	}
}

// TestListAcceptsStreamWhitespace: only whitespace around and between the
// objects, none required.
func TestListAcceptsStreamWhitespace(t *testing.T) {
	got, err := DecodePackages([]byte("\r\n\t {\"Name\":\"a\"}{\"Name\":\"b\"}  \r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "a" || got[1].Name != "b" {
		t.Fatalf("%+v", got)
	}
	if len(got[0].Present) != 1 || !got[0].Present["Name"] {
		t.Fatalf("present %v", got[0].Present)
	}
}
