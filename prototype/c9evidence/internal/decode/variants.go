package decode

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// Variant is one byte variant of the decoder control LC-4
// (docs/wp2.5-c9-limited-evidence-plan.md §5.2 row LC-4; method record §7
// rows V-34, V-34c, V-35 to V-39, V-41). Input is handed to the decoder of
// Call exactly as given. The complete expected outcome is: WantOK; for a
// failing variant the error's Call, WantKind and WantAt (and no decoded
// value); for a decoding variant the decoded value of its call (WantEnv,
// WantContext or WantPackages) and nothing else. The expected values are
// written by hand from the method record, not computed by the decoder.
type Variant struct {
	Name string
	// V names the method record §7 rows (or DS row) the variant stands for.
	V     string
	Call  spec.CallKind
	Input []byte

	WantOK   bool
	WantKind Kind
	WantAt   string

	WantEnv      map[string]string
	WantContext  *Context
	WantPackages []Package
}

// Outcome is what a decoder gave for a variant's input.
type Outcome struct {
	Call     spec.CallKind
	OK       bool
	Err      *Error
	Env      map[string]string
	Context  *Context
	Packages []Package
}

// Run decodes the variant's input with the decoder of its call. The first
// call is decoded with spec.EnvNames.
func (v Variant) Run() Outcome {
	o := Outcome{Call: v.Call}
	var err error
	switch v.Call {
	case spec.CallEnv:
		o.Env, err = DecodeEnv(v.Input, spec.EnvNames)
	case spec.CallContext:
		var c Context
		if c, err = DecodeContext(v.Input); err == nil {
			o.Context = &c
		}
	case spec.CallList:
		o.Packages, err = DecodePackages(v.Input)
	default:
		err = errorf(v.Call, KindUsage, "$", 0, "unknown call %d", int(v.Call))
	}
	if err != nil {
		var de *Error
		if !errors.As(err, &de) {
			de = &Error{Call: v.Call, Kind: KindUsage, At: "$", Detail: "error of type " + reflect.TypeOf(err).String() + ": " + err.Error()}
		}
		o.Err = de
		return o
	}
	o.OK = true
	return o
}

// Match reports whether an outcome equals the variant's complete expected
// outcome; the error says which part differs.
func (v Variant) Match(o Outcome) error {
	if o.Call != v.Call {
		return fmt.Errorf("%s: outcome of %s, want %s", v.Name, o.Call, v.Call)
	}
	if !v.WantOK {
		if o.OK || o.Err == nil {
			return fmt.Errorf("%s: decoded, want %s at %s", v.Name, v.WantKind, v.WantAt)
		}
		if o.Err.Call != v.Call || o.Err.Kind != v.WantKind || o.Err.At != v.WantAt {
			return fmt.Errorf("%s: got %s %s at %s (%s), want %s %s at %s", v.Name, o.Err.Call, o.Err.Kind, o.Err.At, o.Err.Detail, v.Call, v.WantKind, v.WantAt)
		}
		if o.Env != nil || o.Context != nil || o.Packages != nil {
			return fmt.Errorf("%s: a decoded value accompanies the error", v.Name)
		}
		return nil
	}
	if !o.OK || o.Err != nil {
		return fmt.Errorf("%s: got error %v, want a decoded value", v.Name, o.Err)
	}
	switch v.Call {
	case spec.CallEnv:
		if o.Context != nil || o.Packages != nil || !reflect.DeepEqual(o.Env, v.WantEnv) {
			return fmt.Errorf("%s: env %s, want %s", v.Name, describe(o.Env), describe(v.WantEnv))
		}
	case spec.CallContext:
		if o.Env != nil || o.Packages != nil || o.Context == nil || v.WantContext == nil || !reflect.DeepEqual(*o.Context, *v.WantContext) {
			return fmt.Errorf("%s: context %s, want %s", v.Name, describe(o.Context), describe(v.WantContext))
		}
	case spec.CallList:
		if o.Env != nil || o.Context != nil {
			return fmt.Errorf("%s: a value of another call", v.Name)
		}
		if len(o.Packages) != len(v.WantPackages) {
			return fmt.Errorf("%s: %d packages, want %d", v.Name, len(o.Packages), len(v.WantPackages))
		}
		for i := range o.Packages {
			if !reflect.DeepEqual(o.Packages[i], v.WantPackages[i]) {
				return fmt.Errorf("%s: package %d is %s, want %s", v.Name, i, describe(o.Packages[i]), describe(v.WantPackages[i]))
			}
		}
	default:
		return fmt.Errorf("%s: unknown call %d", v.Name, int(v.Call))
	}
	return nil
}

// describe renders a value deterministically for mismatch messages.
func describe(x any) string {
	b, err := json.Marshal(x)
	if err != nil {
		return fmt.Sprintf("<%v>", err)
	}
	return string(b)
}

func present(keys ...string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

// The healthy outputs. They follow the fixed encoders' layout (go env's
// json.Encoder and go list's json.MarshalIndent, both with a TAB indent and
// one LF after each object; T2's lines), with fixture values.

const envLinux = `{
	"CGO_ENABLED": "0",
	"GOARCH": "amd64",
	"GOFLAGS": "",
	"GOOS": "linux",
	"GOROOT": "/opt/hostedtoolcache/go/1.27.3/x64",
	"GOTOOLCHAIN": "local",
	"GOVERSION": "go1.27.3",
	"GOWORK": "off"
}
`

const envWindows = `{
	"CGO_ENABLED": "0",
	"GOARCH": "arm64",
	"GOFLAGS": "",
	"GOOS": "windows",
	"GOROOT": "C:\\hostedtoolcache\\windows\\go\\1.27.3\\x64",
	"GOTOOLCHAIN": "local",
	"GOVERSION": "go1.27.3",
	"GOWORK": "off"
}
`

func wantEnvLinux() map[string]string {
	return map[string]string{
		"CGO_ENABLED": "0",
		"GOARCH":      "amd64",
		"GOFLAGS":     "",
		"GOOS":        "linux",
		"GOROOT":      "/opt/hostedtoolcache/go/1.27.3/x64",
		"GOTOOLCHAIN": "local",
		"GOVERSION":   "go1.27.3",
		"GOWORK":      "off",
	}
}

const ctxLinux = `c9-context v1
GOARCH "amd64"
GOOS "linux"
GOROOT "/opt/hostedtoolcache/go/1.27.3/x64"
GOPATH "/home/runner/go"
CgoEnabled false
UseAllFiles false
Compiler "gc"
InstallSuffix ""
BuildTags 0
ToolTags 1
- "goamd64.v1"
ReleaseTags 3
- "go1.1"
- "go1.2"
- "go1.3"
end
`

func wantCtxLinux() *Context {
	return &Context{
		GOARCH:        "amd64",
		GOOS:          "linux",
		GOROOT:        "/opt/hostedtoolcache/go/1.27.3/x64",
		GOPATH:        "/home/runner/go",
		CgoEnabled:    false,
		UseAllFiles:   false,
		Compiler:      "gc",
		InstallSuffix: "",
		BuildTags:     []string{},
		ToolTags:      []string{"goamd64.v1"},
		ReleaseTags:   []string{"go1.1", "go1.2", "go1.3"},
	}
}

const listStd = `{
	"Dir": "/opt/hostedtoolcache/go/1.27.3/x64/src/net/http",
	"ImportPath": "net/http",
	"Name": "http",
	"Standard": true,
	"GoFiles": [
		"client.go",
		"server.go"
	],
	"IgnoredGoFiles": [
		"triv.go"
	],
	"Imports": [
		"errors",
		"vendor/golang.org/x/net/http/httpguts"
	],
	"ImportMap": {
		"golang.org/x/net/http/httpguts": "vendor/golang.org/x/net/http/httpguts"
	}
}
`

func wantStd() Package {
	return Package{
		Dir:            "/opt/hostedtoolcache/go/1.27.3/x64/src/net/http",
		ImportPath:     "net/http",
		Name:           "http",
		Standard:       true,
		GoFiles:        []string{"client.go", "server.go"},
		IgnoredGoFiles: []string{"triv.go"},
		Imports:        []string{"errors", "vendor/golang.org/x/net/http/httpguts"},
		ImportMap:      map[string]string{"golang.org/x/net/http/httpguts": "vendor/golang.org/x/net/http/httpguts"},
		Present:        present("Dir", "ImportPath", "Name", "Standard", "GoFiles", "IgnoredGoFiles", "Imports", "ImportMap"),
	}
}

const listMain = `{
	"Dir": "/home/runner/work/specmetry/specmetry/internal/cli",
	"ImportPath": "github.com/XKHoshizora/specmetry/internal/cli",
	"Name": "cli",
	"Module": {
		"Path": "github.com/XKHoshizora/specmetry",
		"Main": true,
		"Dir": "/home/runner/work/specmetry/specmetry",
		"GoMod": "/home/runner/work/specmetry/specmetry/go.mod",
		"GoVersion": "1.27"
	},
	"GoFiles": [
		"cli.go"
	],
	"Imports": [
		"errors",
		"modernc.org/libc"
	]
}
`

func wantMain() Package {
	return Package{
		Dir:        "/home/runner/work/specmetry/specmetry/internal/cli",
		ImportPath: "github.com/XKHoshizora/specmetry/internal/cli",
		Name:       "cli",
		Module: &Module{
			Path:      "github.com/XKHoshizora/specmetry",
			Main:      true,
			Dir:       "/home/runner/work/specmetry/specmetry",
			GoMod:     "/home/runner/work/specmetry/specmetry/go.mod",
			GoVersion: "1.27",
			Present:   present("Path", "Main", "Dir", "GoMod", "GoVersion"),
		},
		GoFiles: []string{"cli.go"},
		Imports: []string{"errors", "modernc.org/libc"},
		Present: present("Dir", "ImportPath", "Name", "Module", "GoFiles", "Imports"),
	}
}

const listDep = `{
	"Dir": "/home/runner/go/pkg/mod/modernc.org/libc@v1.77.1",
	"ImportPath": "modernc.org/libc",
	"Name": "libc",
	"Module": {
		"Path": "modernc.org/libc",
		"Version": "v1.77.1",
		"Time": "2026-08-01T10:20:30Z",
		"Indirect": true,
		"Dir": "/home/runner/go/pkg/mod/modernc.org/libc@v1.77.1",
		"GoMod": "/home/runner/go/pkg/mod/cache/download/modernc.org/libc/@v/v1.77.1.mod",
		"Sum": "h1:Qm9kdWxlQ29udGVudEZpeHR1cmVIYXNoMDAwMDA=",
		"GoModSum": "h1:R29Nb2RGaXh0dXJlSGFzaDAwMDAwMDAwMDAwMDA="
	},
	"GoFiles": [
		"libc.go",
		"libc_linux.go",
		"libc_musl.go"
	],
	"IgnoredGoFiles": [
		"libc_darwin.go",
		"libc_unix.go",
		"libc_windows.go"
	],
	"Imports": [
		"golang.org/x/sys/unix",
		"os/exec",
		"os/signal"
	]
}
`

func wantDep() Package {
	return Package{
		Dir:        "/home/runner/go/pkg/mod/modernc.org/libc@v1.77.1",
		ImportPath: "modernc.org/libc",
		Name:       "libc",
		Module: &Module{
			Path:     "modernc.org/libc",
			Version:  "v1.77.1",
			Time:     "2026-08-01T10:20:30Z",
			Indirect: true,
			Dir:      "/home/runner/go/pkg/mod/modernc.org/libc@v1.77.1",
			GoMod:    "/home/runner/go/pkg/mod/cache/download/modernc.org/libc/@v/v1.77.1.mod",
			Sum:      "h1:Qm9kdWxlQ29udGVudEZpeHR1cmVIYXNoMDAwMDA=",
			GoModSum: "h1:R29Nb2RGaXh0dXJlSGFzaDAwMDAwMDAwMDAwMDA=",
			Present:  present("Path", "Version", "Time", "Indirect", "Dir", "GoMod", "Sum", "GoModSum"),
		},
		GoFiles:        []string{"libc.go", "libc_linux.go", "libc_musl.go"},
		IgnoredGoFiles: []string{"libc_darwin.go", "libc_unix.go", "libc_windows.go"},
		Imports:        []string{"golang.org/x/sys/unix", "os/exec", "os/signal"},
		Present:        present("Dir", "ImportPath", "Name", "Module", "GoFiles", "IgnoredGoFiles", "Imports"),
	}
}

// LC4Variants returns the byte variants of LC-4, freshly built on every
// call. The list, by call:
//
// First call (go env -json, decoded with spec.EnvNames): env.healthy,
// env.healthy-escapes decode; env.missing-key, env.extra-key,
// env.case-key, env.escaped-key, env.null, env.duplicate-key,
// env.type-mismatch, env.trailing, env.crlf-end, env.second-object,
// env.no-final-lf, env.leading-space, env.top-level-array, env.malformed,
// env.zero-bytes fail.
//
// Second call (context text v1): ctx.healthy (V-34), ctx.healthy-escapes,
// ctx.order-and-duplicates, ctx.syntax-only decode; ctx.missing-end,
// ctx.count-short, ctx.count-long, ctx.noncanonical-x41, ctx.crlf,
// ctx.two-texts, ctx.extra-line, ctx.cgo-zero (the V-34c list),
// ctx.zero-bytes, ctx.after-end, ctx.no-final-lf, ctx.blank-line,
// ctx.double-space, ctx.unquoted, ctx.invalid-escape, ctx.leading-zero,
// ctx.element-no-space, ctx.json-shaped fail.
//
// Third call (go list -json): list.std (V-36), list.main (V-37), list.dep
// (V-35, V-35b), list.stream, list.no-module (V-38), list.dep-omissions
// (V-39), list.std-with-module (V-36b), list.error-fields,
// list.module-all-fields, list.importmap-case, list.empty-array decode;
// list.extra-key, list.extra-module-key, list.extra-packageerror-key,
// list.case-key, list.case-module-key, list.null-module,
// list.null-nested, list.null-element, list.duplicate-key,
// list.duplicate-module-key, list.duplicate-importmap-key,
// list.type-standard-string, list.type-gofiles-string,
// list.type-imports-element, list.type-importmap-value,
// list.type-module-array, list.type-module-error-string,
// list.type-time, list.type-number, list.order-keys-before-types,
// list.trailing, list.trailing-array, list.malformed,
// list.second-malformed, list.top-level-array, list.zero-bytes,
// list.whitespace-only fail.
func LC4Variants() []Variant {
	env := func(name, v string, in string, ok bool, kind Kind, at string, want map[string]string) Variant {
		return Variant{Name: name, V: v, Call: spec.CallEnv, Input: []byte(in), WantOK: ok, WantKind: kind, WantAt: at, WantEnv: want}
	}
	ctx := func(name, v string, in string, ok bool, kind Kind, at string, want *Context) Variant {
		return Variant{Name: name, V: v, Call: spec.CallContext, Input: []byte(in), WantOK: ok, WantKind: kind, WantAt: at, WantContext: want}
	}
	list := func(name, v string, in string, ok bool, kind Kind, at string, want []Package) Variant {
		return Variant{Name: name, V: v, Call: spec.CallList, Input: []byte(in), WantOK: ok, WantKind: kind, WantAt: at, WantPackages: want}
	}
	fail := false

	vs := []Variant{
		// First call.
		env("env.healthy", "DS-1", envLinux, true, "", "", wantEnvLinux()),
		env("env.healthy-escapes", "DS-1", envWindows, true, "", "", map[string]string{
			"CGO_ENABLED": "0",
			"GOARCH":      "arm64",
			"GOFLAGS":     "",
			"GOOS":        "windows",
			"GOROOT":      `C:\hostedtoolcache\windows\go\1.27.3\x64`,
			"GOTOOLCHAIN": "local",
			"GOVERSION":   "go1.27.3",
			"GOWORK":      "off",
		}),
		env("env.missing-key", "V-41 (DS-1 missing key)",
			"{\n\t\"CGO_ENABLED\": \"0\",\n\t\"GOARCH\": \"amd64\",\n\t\"GOFLAGS\": \"\",\n\t\"GOOS\": \"linux\",\n\t\"GOROOT\": \"/go\",\n\t\"GOTOOLCHAIN\": \"local\",\n\t\"GOVERSION\": \"go1.27.3\"\n}\n",
			fail, KindMissing, "$.GOWORK", nil),
		env("env.extra-key", "V-41",
			"{\n\t\"CGO_ENABLED\": \"0\",\n\t\"GOARCH\": \"amd64\",\n\t\"GOFLAGS\": \"\",\n\t\"GOOS\": \"linux\",\n\t\"GOPROXY\": \"off\",\n\t\"GOROOT\": \"/go\",\n\t\"GOTOOLCHAIN\": \"local\",\n\t\"GOVERSION\": \"go1.27.3\",\n\t\"GOWORK\": \"off\"\n}\n",
			fail, KindKey, "$.GOPROXY", nil),
		env("env.case-key", "V-41",
			"{\n\t\"CGO_ENABLED\": \"0\",\n\t\"GOARCH\": \"amd64\",\n\t\"GOFLAGS\": \"\",\n\t\"Goos\": \"linux\",\n\t\"GOROOT\": \"/go\",\n\t\"GOTOOLCHAIN\": \"local\",\n\t\"GOVERSION\": \"go1.27.3\",\n\t\"GOWORK\": \"off\"\n}\n",
			fail, KindCase, "$.Goos", nil),
		env("env.escaped-key", "V-41",
			"{\n\t\"CGO_ENABLED\": \"0\",\n\t\"GOARCH\": \"amd64\",\n\t\"GOFLAGS\": \"\",\n\t\"GO\\u004fS\": \"linux\",\n\t\"GOROOT\": \"/go\",\n\t\"GOTOOLCHAIN\": \"local\",\n\t\"GOVERSION\": \"go1.27.3\",\n\t\"GOWORK\": \"off\"\n}\n",
			fail, KindKey, "$.GOOS", nil),
		env("env.null", "V-41",
			"{\n\t\"CGO_ENABLED\": \"0\",\n\t\"GOARCH\": \"amd64\",\n\t\"GOFLAGS\": null,\n\t\"GOOS\": \"linux\",\n\t\"GOROOT\": \"/go\",\n\t\"GOTOOLCHAIN\": \"local\",\n\t\"GOVERSION\": \"go1.27.3\",\n\t\"GOWORK\": \"off\"\n}\n",
			fail, KindNull, "$.GOFLAGS", nil),
		env("env.duplicate-key", "V-41",
			"{\n\t\"CGO_ENABLED\": \"0\",\n\t\"GOARCH\": \"amd64\",\n\t\"GOFLAGS\": \"\",\n\t\"GOOS\": \"linux\",\n\t\"GOOS\": \"linux\",\n\t\"GOROOT\": \"/go\",\n\t\"GOTOOLCHAIN\": \"local\",\n\t\"GOVERSION\": \"go1.27.3\",\n\t\"GOWORK\": \"off\"\n}\n",
			fail, KindDuplicate, "$.GOOS", nil),
		env("env.type-mismatch", "V-41",
			"{\n\t\"CGO_ENABLED\": 0,\n\t\"GOARCH\": \"amd64\",\n\t\"GOFLAGS\": \"\",\n\t\"GOOS\": \"linux\",\n\t\"GOROOT\": \"/go\",\n\t\"GOTOOLCHAIN\": \"local\",\n\t\"GOVERSION\": \"go1.27.3\",\n\t\"GOWORK\": \"off\"\n}\n",
			fail, KindType, "$.CGO_ENABLED", nil),
		env("env.trailing", "V-41", envLinux+"x", fail, KindTrailing, "$", nil),
		env("env.crlf-end", "V-41", envLinux[:len(envLinux)-1]+"\r\n", fail, KindTrailing, "$", nil),
		env("env.second-object", "V-41", envLinux+envLinux, fail, KindCount, "$", nil),
		env("env.no-final-lf", "V-41", envLinux[:len(envLinux)-1], fail, KindForm, "$", nil),
		env("env.leading-space", "V-41", " "+envLinux, fail, KindForm, "$", nil),
		env("env.top-level-array", "V-41", "[]\n", fail, KindForm, "$", nil),
		env("env.malformed", "V-41", "{\n\t\"CGO_ENABLED\": \"0\",\n\t\"GOARCH\"\n}\n", fail, KindSyntax, "$", nil),
		env("env.zero-bytes", "V-41", "", fail, KindCount, "$", nil),

		// Second call.
		ctx("ctx.healthy", "V-34", ctxLinux, true, "", "", wantCtxLinux()),
		ctx("ctx.healthy-escapes", "V-34",
			"c9-context v1\nGOARCH \"arm64\"\nGOOS \"windows\"\nGOROOT \"C:\\\\hostedtoolcache\\\\windows\\\\go\\\\1.27.3\\\\x64\"\nGOPATH \"C:\\\\Users\\\\runneradmin\\\\go\"\nCgoEnabled false\nUseAllFiles false\nCompiler \"gc\"\nInstallSuffix \"\"\nBuildTags 0\nToolTags 1\n- \"goarm64.v8.0\"\nReleaseTags 2\n- \"go1.1\"\n- \"go1.27\"\nend\n",
			true, "", "", &Context{
				GOARCH:      "arm64",
				GOOS:        "windows",
				GOROOT:      `C:\hostedtoolcache\windows\go\1.27.3\x64`,
				GOPATH:      `C:\Users\runneradmin\go`,
				Compiler:    "gc",
				BuildTags:   []string{},
				ToolTags:    []string{"goarm64.v8.0"},
				ReleaseTags: []string{"go1.1", "go1.27"},
			}),
		ctx("ctx.order-and-duplicates", "V-34",
			"c9-context v1\nGOARCH \"amd64\"\nGOOS \"darwin\"\nGOROOT \"/go\"\nGOPATH \"/Users/r\u00fcnner/go\\t\"\nCgoEnabled false\nUseAllFiles false\nCompiler \"gc\"\nInstallSuffix \"\"\nBuildTags 0\nToolTags 2\n- \"goamd64.v1\"\n- \"goamd64.v1\"\nReleaseTags 3\n- \"go1.2\"\n- \"go1.1\"\n- \"go1.1\"\nend\n",
			true, "", "", &Context{
				GOARCH:      "amd64",
				GOOS:        "darwin",
				GOROOT:      "/go",
				GOPATH:      "/Users/r\u00fcnner/go\t",
				Compiler:    "gc",
				BuildTags:   []string{},
				ToolTags:    []string{"goamd64.v1", "goamd64.v1"},
				ReleaseTags: []string{"go1.2", "go1.1", "go1.1"},
			}),
		// Values §3.3.3 rejects in B2 (V-34d, V-40) still decode: decoding is
		// syntax only.
		ctx("ctx.syntax-only", "V-34d, V-40 (decode layer)",
			"c9-context v1\nGOARCH \"amd64\"\nGOOS \"\"\nGOROOT \"/go\"\nGOPATH \"\"\nCgoEnabled true\nUseAllFiles true\nCompiler \"\"\nInstallSuffix \"race\"\nBuildTags 1\n- \"nowebui\"\nToolTags 0\nReleaseTags 0\nend\n",
			true, "", "", &Context{
				GOARCH:        "amd64",
				GOROOT:        "/go",
				CgoEnabled:    true,
				UseAllFiles:   true,
				InstallSuffix: "race",
				BuildTags:     []string{"nowebui"},
				ToolTags:      []string{},
				ReleaseTags:   []string{},
			}),
		ctx("ctx.missing-end", "V-34c", ctxLinux[:len(ctxLinux)-len("end\n")], fail, KindForm, "line 17", nil),
		ctx("ctx.count-short", "V-34c",
			"c9-context v1\nGOARCH \"amd64\"\nGOOS \"linux\"\nGOROOT \"/go\"\nGOPATH \"/home/runner/go\"\nCgoEnabled false\nUseAllFiles false\nCompiler \"gc\"\nInstallSuffix \"\"\nBuildTags 0\nToolTags 2\n- \"goamd64.v1\"\nReleaseTags 1\n- \"go1.1\"\nend\n",
			fail, KindCount, "line 13", nil),
		ctx("ctx.count-long", "V-34c",
			"c9-context v1\nGOARCH \"amd64\"\nGOOS \"linux\"\nGOROOT \"/go\"\nGOPATH \"/home/runner/go\"\nCgoEnabled false\nUseAllFiles false\nCompiler \"gc\"\nInstallSuffix \"\"\nBuildTags 0\n- \"nowebui\"\nToolTags 1\n- \"goamd64.v1\"\nReleaseTags 1\n- \"go1.1\"\nend\n",
			fail, KindCount, "line 11", nil),
		ctx("ctx.noncanonical-x41", "V-34c",
			"c9-context v1\nGOARCH \"amd64\"\nGOOS \"linux\"\nGOROOT \"/go\"\nGOPATH \"\\x41\"\nCgoEnabled false\nUseAllFiles false\nCompiler \"gc\"\nInstallSuffix \"\"\nBuildTags 0\nToolTags 1\n- \"goamd64.v1\"\nReleaseTags 1\n- \"go1.1\"\nend\n",
			fail, KindCanonical, "line 5", nil),
		ctx("ctx.crlf", "V-34c",
			"c9-context v1\r\nGOARCH \"amd64\"\r\nGOOS \"linux\"\r\nGOROOT \"/go\"\r\nGOPATH \"/home/runner/go\"\r\nCgoEnabled false\r\nUseAllFiles false\r\nCompiler \"gc\"\r\nInstallSuffix \"\"\r\nBuildTags 0\r\nToolTags 1\r\n- \"goamd64.v1\"\r\nReleaseTags 1\r\n- \"go1.1\"\r\nend\r\n",
			fail, KindLine, "line 1", nil),
		ctx("ctx.two-texts", "V-34c", ctxLinux+ctxLinux, fail, KindCount, "line 18", nil),
		ctx("ctx.extra-line", "V-34c",
			"c9-context v1\nGOARCH \"amd64\"\nGOOS \"linux\"\nGOOS \"linux\"\nGOROOT \"/go\"\nGOPATH \"/home/runner/go\"\nCgoEnabled false\nUseAllFiles false\nCompiler \"gc\"\nInstallSuffix \"\"\nBuildTags 0\nToolTags 1\n- \"goamd64.v1\"\nReleaseTags 1\n- \"go1.1\"\nend\n",
			fail, KindLine, "line 4", nil),
		ctx("ctx.cgo-zero", "V-34c",
			"c9-context v1\nGOARCH \"amd64\"\nGOOS \"linux\"\nGOROOT \"/go\"\nGOPATH \"/home/runner/go\"\nCgoEnabled 0\nUseAllFiles false\nCompiler \"gc\"\nInstallSuffix \"\"\nBuildTags 0\nToolTags 1\n- \"goamd64.v1\"\nReleaseTags 1\n- \"go1.1\"\nend\n",
			fail, KindValue, "line 6", nil),
		ctx("ctx.zero-bytes", "V-34c (§3.3.2 count)", "", fail, KindCount, "line 1", nil),
		ctx("ctx.after-end", "V-34c (§3.3.2 count)", ctxLinux+"x\n", fail, KindTrailing, "line 18", nil),
		ctx("ctx.no-final-lf", "V-34c (§3.3.2 lines)", ctxLinux[:len(ctxLinux)-1], fail, KindForm, "line 17", nil),
		ctx("ctx.blank-line", "V-34c (§3.3.2 lines)", "c9-context v1\n\n"+ctxLinux[len("c9-context v1\n"):], fail, KindLine, "line 2", nil),
		ctx("ctx.double-space", "V-34c (§3.3.2 separator)",
			"c9-context v1\nGOARCH \"amd64\"\nGOOS  \"linux\"\nGOROOT \"/go\"\nGOPATH \"/home/runner/go\"\nCgoEnabled false\nUseAllFiles false\nCompiler \"gc\"\nInstallSuffix \"\"\nBuildTags 0\nToolTags 1\n- \"goamd64.v1\"\nReleaseTags 1\n- \"go1.1\"\nend\n",
			fail, KindLine, "line 3", nil),
		ctx("ctx.unquoted", "V-34c (§3.3.2 <Q>)",
			"c9-context v1\nGOARCH \"amd64\"\nGOOS linux\nGOROOT \"/go\"\nGOPATH \"/home/runner/go\"\nCgoEnabled false\nUseAllFiles false\nCompiler \"gc\"\nInstallSuffix \"\"\nBuildTags 0\nToolTags 1\n- \"goamd64.v1\"\nReleaseTags 1\n- \"go1.1\"\nend\n",
			fail, KindValue, "line 3", nil),
		ctx("ctx.invalid-escape", "V-34c (§3.3.2 <Q>)",
			"c9-context v1\nGOARCH \"amd64\"\nGOOS \"linux\"\nGOROOT \"/go\"\nGOPATH \"\\q\"\nCgoEnabled false\nUseAllFiles false\nCompiler \"gc\"\nInstallSuffix \"\"\nBuildTags 0\nToolTags 1\n- \"goamd64.v1\"\nReleaseTags 1\n- \"go1.1\"\nend\n",
			fail, KindValue, "line 5", nil),
		ctx("ctx.leading-zero", "V-34c (§3.3.2 <N>)",
			"c9-context v1\nGOARCH \"amd64\"\nGOOS \"linux\"\nGOROOT \"/go\"\nGOPATH \"/home/runner/go\"\nCgoEnabled false\nUseAllFiles false\nCompiler \"gc\"\nInstallSuffix \"\"\nBuildTags 00\nToolTags 1\n- \"goamd64.v1\"\nReleaseTags 1\n- \"go1.1\"\nend\n",
			fail, KindValue, "line 10", nil),
		ctx("ctx.element-no-space", "V-34c (§3.3.2 separator)",
			"c9-context v1\nGOARCH \"amd64\"\nGOOS \"linux\"\nGOROOT \"/go\"\nGOPATH \"/home/runner/go\"\nCgoEnabled false\nUseAllFiles false\nCompiler \"gc\"\nInstallSuffix \"\"\nBuildTags 0\nToolTags 1\n-\"goamd64.v1\"\nReleaseTags 1\n- \"go1.1\"\nend\n",
			fail, KindLine, "line 12", nil),
		ctx("ctx.json-shaped", "V-34 (not JSON)", "{\"GOARCH\":\"amd64\",\"GOOS\":\"linux\"}\n", fail, KindLine, "line 1", nil),

		// Third call: decoding variants.
		list("list.std", "V-36", listStd, true, "", "", []Package{wantStd()}),
		list("list.main", "V-37", listMain, true, "", "", []Package{wantMain()}),
		list("list.dep", "V-35, V-35b", listDep, true, "", "", []Package{wantDep()}),
		list("list.stream", "V-35, V-36, V-37", listStd+listMain+listDep, true, "", "", []Package{wantStd(), wantMain(), wantDep()}),
		list("list.no-module", "V-38 (decode layer)",
			"{\n\t\"Dir\": \"/x/p\",\n\t\"ImportPath\": \"example.com/p\",\n\t\"Name\": \"p\"\n}\n",
			true, "", "", []Package{{Dir: "/x/p", ImportPath: "example.com/p", Name: "p", Present: present("Dir", "ImportPath", "Name")}}),
		list("list.dep-omissions", "V-39 (decode layer)",
			"{\n\t\"ImportPath\": \"modernc.org/libc\",\n\t\"Name\": \"libc\",\n\t\"Module\": {\n\t\t\"Path\": \"modernc.org/libc\",\n\t\t\"Version\": \"v1.77.1\",\n\t\t\"Dir\": \"/m/libc\",\n\t\t\"GoModSum\": \"h1:x=\"\n\t}\n}\n",
			true, "", "", []Package{{
				ImportPath: "modernc.org/libc",
				Name:       "libc",
				Module:     &Module{Path: "modernc.org/libc", Version: "v1.77.1", Dir: "/m/libc", GoModSum: "h1:x=", Present: present("Path", "Version", "Dir", "GoModSum")},
				Present:    present("ImportPath", "Name", "Module"),
			}}),
		list("list.std-with-module", "V-36b (decode layer)",
			"{\"Dir\":\"/go/src/errors\",\"ImportPath\":\"errors\",\"Name\":\"errors\",\"Module\":{\"Path\":\"std\"},\"Standard\":true}\n",
			true, "", "", []Package{{
				Dir: "/go/src/errors", ImportPath: "errors", Name: "errors", Standard: true,
				Module:  &Module{Path: "std", Present: present("Path")},
				Present: present("Dir", "ImportPath", "Name", "Module", "Standard"),
			}}),
		list("list.error-fields", "§3.4 every record (decode layer)",
			"{\n\t\"Dir\": \"/x/p\",\n\t\"ImportPath\": \"example.com/p\",\n\t\"Name\": \"p\",\n\t\"ForTest\": \"example.com/p\",\n\t\"Incomplete\": true,\n\t\"Error\": {\n\t\t\"ImportStack\": [],\n\t\t\"Pos\": \"\",\n\t\t\"Err\": \"no Go files\"\n\t},\n\t\"DepsErrors\": [\n\t\t{\n\t\t\t\"ImportStack\": [\n\t\t\t\t\"example.com/p\",\n\t\t\t\t\"example.com/q\"\n\t\t\t],\n\t\t\t\"Pos\": \"q.go:3:8\",\n\t\t\t\"Err\": \"cannot find package\"\n\t\t}\n\t]\n}\n",
			true, "", "", []Package{{
				Dir: "/x/p", ImportPath: "example.com/p", Name: "p", ForTest: "example.com/p", Incomplete: true,
				Error: &PackageError{ImportStack: []string{}, Pos: "", Err: "no Go files", Present: present("ImportStack", "Pos", "Err")},
				DepsErrors: []PackageError{{
					ImportStack: []string{"example.com/p", "example.com/q"}, Pos: "q.go:3:8", Err: "cannot find package",
					Present: present("ImportStack", "Pos", "Err"),
				}},
				Present: present("Dir", "ImportPath", "Name", "ForTest", "Incomplete", "Error", "DepsErrors"),
			}}),
		list("list.module-all-fields", "§3.4 ModulePublic (decode layer)",
			"{\"ImportPath\":\"a.example/m\",\"Name\":\"m\",\"Module\":{\"Path\":\"a.example/m\",\"Version\":\"v1.0.0\",\"Query\":\"latest\",\"Versions\":[\"v0.9.0\",\"v1.0.0\"],\"Replace\":{\"Path\":\"../m\",\"Dir\":\"/r/m\",\"GoVersion\":\"1.21\"},\"Time\":\"2025-01-02T03:04:05.123456789+09:00\",\"Update\":{\"Path\":\"a.example/m\",\"Version\":\"v1.1.0\"},\"Main\":false,\"Indirect\":true,\"Dir\":\"/r/m\",\"GoMod\":\"/r/m/go.mod\",\"GoVersion\":\"1.21\",\"Retracted\":[\"bad\"],\"Deprecated\":\"use n\",\"Error\":{\"Err\":\"oops\"},\"Sum\":\"h1:s=\",\"GoModSum\":\"h1:g=\",\"Origin\":{\"VCS\":\"git\",\"URL\":\"https://a.example/m\",\"Subdir\":\"sub\",\"Hash\":\"abc\",\"TagPrefix\":\"sub/\",\"TagSum\":\"t1:x\",\"Ref\":\"refs/tags/v1.0.0\",\"RepoSum\":\"r1:y\"},\"Reuse\":true}}\n",
			true, "", "", []Package{{
				ImportPath: "a.example/m",
				Name:       "m",
				Module: &Module{
					Path:       "a.example/m",
					Version:    "v1.0.0",
					Query:      "latest",
					Versions:   []string{"v0.9.0", "v1.0.0"},
					Replace:    &Module{Path: "../m", Dir: "/r/m", GoVersion: "1.21", Present: present("Path", "Dir", "GoVersion")},
					Time:       "2025-01-02T03:04:05.123456789+09:00",
					Update:     &Module{Path: "a.example/m", Version: "v1.1.0", Present: present("Path", "Version")},
					Main:       false,
					Indirect:   true,
					Dir:        "/r/m",
					GoMod:      "/r/m/go.mod",
					GoVersion:  "1.21",
					Retracted:  []string{"bad"},
					Deprecated: "use n",
					Error:      &ModuleError{Err: "oops", Present: present("Err")},
					Sum:        "h1:s=",
					GoModSum:   "h1:g=",
					Origin: &Origin{
						VCS: "git", URL: "https://a.example/m", Subdir: "sub", Hash: "abc",
						TagPrefix: "sub/", TagSum: "t1:x", Ref: "refs/tags/v1.0.0", RepoSum: "r1:y",
						Present: present("VCS", "URL", "Subdir", "Hash", "TagPrefix", "TagSum", "Ref", "RepoSum"),
					},
					Reuse:   true,
					Present: present("Path", "Version", "Query", "Versions", "Replace", "Time", "Update", "Main", "Indirect", "Dir", "GoMod", "GoVersion", "Retracted", "Deprecated", "Error", "Sum", "GoModSum", "Origin", "Reuse"),
				},
				Present: present("ImportPath", "Name", "Module"),
			}}),
		list("list.importmap-case", "§3.1 step 2 (map keys are data)",
			"{\"ImportPath\":\"p\",\"Name\":\"p\",\"ImportMap\":{\"a\":\"x\",\"A\":\"y\",\"\\u0062\":\"z\"}}\n",
			true, "", "", []Package{{
				ImportPath: "p", Name: "p",
				ImportMap: map[string]string{"a": "x", "A": "y", "b": "z"},
				Present:   present("ImportPath", "Name", "ImportMap"),
			}}),
		list("list.empty-array", "§3.1 step 4", "{\"ImportPath\":\"p\",\"Name\":\"p\",\"CgoFiles\":[],\"ImportMap\":{}}\n",
			true, "", "", []Package{{
				ImportPath: "p", Name: "p", CgoFiles: []string{}, ImportMap: map[string]string{},
				Present: present("ImportPath", "Name", "CgoFiles", "ImportMap"),
			}}),

		// Third call: failing variants.
		list("list.extra-key", "V-41", "{\"ImportPath\":\"p\",\"Name\":\"p\",\"Deps\":[\"errors\"]}\n", fail, KindKey, "$[0].Deps", nil),
		list("list.extra-module-key", "V-41", "{\"ImportPath\":\"p\",\"Module\":{\"Path\":\"m\",\"Foo\":\"x\"}}\n", fail, KindKey, "$[0].Module.Foo", nil),
		list("list.extra-packageerror-key", "V-41", "{\"ImportPath\":\"p\",\"Error\":{\"ImportStack\":[],\"Pos\":\"\",\"Err\":\"e\",\"IsImportCycle\":true}}\n", fail, KindKey, "$[0].Error.IsImportCycle", nil),
		list("list.case-key", "V-41", "{\"importPath\":\"p\",\"Name\":\"p\"}\n", fail, KindCase, "$[0].importPath", nil),
		list("list.case-module-key", "V-41", "{\"ImportPath\":\"p\",\"Module\":{\"Path\":\"m\",\"sum\":\"h1:x=\"}}\n", fail, KindCase, "$[0].Module.sum", nil),
		list("list.null-module", "V-41", "{\"ImportPath\":\"p\",\"Name\":\"p\",\"Module\":null}\n", fail, KindNull, "$[0].Module", nil),
		list("list.null-nested", "V-41", "{\"ImportPath\":\"p\",\"Module\":{\"Path\":\"m\",\"Sum\":null}}\n", fail, KindNull, "$[0].Module.Sum", nil),
		list("list.null-element", "V-41", "{\"ImportPath\":\"p\",\"GoFiles\":[\"a.go\",null]}\n", fail, KindNull, "$[0].GoFiles[1]", nil),
		list("list.duplicate-key", "V-41", "{\"ImportPath\":\"p\",\"Name\":\"p\",\"Name\":\"p\"}\n", fail, KindDuplicate, "$[0].Name", nil),
		list("list.duplicate-module-key", "V-41", "{\"ImportPath\":\"p\",\"Module\":{\"Path\":\"m\",\"Path\":\"m\"}}\n", fail, KindDuplicate, "$[0].Module.Path", nil),
		list("list.duplicate-importmap-key", "V-41", "{\"ImportPath\":\"p\",\"ImportMap\":{\"a\":\"x\",\"\\u0061\":\"y\"}}\n", fail, KindDuplicate, `$[0].ImportMap["a"]`, nil),
		list("list.type-standard-string", "V-41", "{\"ImportPath\":\"errors\",\"Standard\":\"true\"}\n", fail, KindType, "$[0].Standard", nil),
		list("list.type-gofiles-string", "V-41", "{\"ImportPath\":\"p\",\"GoFiles\":\"a.go\"}\n", fail, KindType, "$[0].GoFiles", nil),
		list("list.type-imports-element", "V-41", "{\"ImportPath\":\"p\",\"Imports\":[\"errors\",1]}\n", fail, KindType, "$[0].Imports[1]", nil),
		list("list.type-importmap-value", "V-41", "{\"ImportPath\":\"p\",\"ImportMap\":{\"a\":true}}\n", fail, KindType, `$[0].ImportMap["a"]`, nil),
		list("list.type-module-array", "V-41", "{\"ImportPath\":\"p\",\"Module\":[]}\n", fail, KindType, "$[0].Module", nil),
		list("list.type-module-error-string", "V-41", "{\"ImportPath\":\"p\",\"Module\":{\"Path\":\"m\",\"Error\":\"boom\"}}\n", fail, KindType, "$[0].Module.Error", nil),
		list("list.type-time", "V-41", "{\"ImportPath\":\"p\",\"Module\":{\"Path\":\"m\",\"Time\":\"yesterday\"}}\n", fail, KindType, "$[0].Module.Time", nil),
		list("list.type-number", "V-41", "{\"ImportPath\":\"p\",\"Incomplete\":1}\n", fail, KindType, "$[0].Incomplete", nil),
		// Keys (step 2) are checked over the whole output before types (step
		// 3): the type mismatch in record 0 is not the reported error.
		list("list.order-keys-before-types", "V-41 (§3.1 order)", "{\"ImportPath\":\"p\",\"Standard\":\"x\"}\n{\"ImportPath\":\"q\",\"Foo\":1}\n", fail, KindKey, "$[1].Foo", nil),
		list("list.trailing", "V-41", listStd+"x", fail, KindTrailing, "$[1]", nil),
		list("list.trailing-array", "V-41", listStd+"[]\n", fail, KindTrailing, "$[1]", nil),
		list("list.malformed", "V-41", listStd[:40], fail, KindSyntax, "$[0]", nil),
		list("list.second-malformed", "V-41", listStd+"{\"ImportPath\": }\n", fail, KindSyntax, "$[1]", nil),
		list("list.top-level-array", "V-41", "["+listStd+"]\n", fail, KindForm, "$", nil),
		list("list.zero-bytes", "V-41 (§3.1 count)", "", fail, KindCount, "$", nil),
		list("list.whitespace-only", "V-41 (§3.1 count)", "\n \t\r\n", fail, KindCount, "$", nil),
	}
	return vs
}
