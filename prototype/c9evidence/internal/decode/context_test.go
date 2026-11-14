package decode

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"text/template"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// listContext mirrors go list's Context (go1.27.3 cmd/go/internal/list/
// context.go:11-23): the same eleven field names and types, no methods.
type listContext struct {
	GOARCH        string
	GOOS          string
	GOROOT        string
	GOPATH        string
	CgoEnabled    bool
	UseAllFiles   bool
	Compiler      string
	BuildTags     []string
	ToolTags      []string
	ReleaseTags   []string
	InstallSuffix string
}

// runT2 executes the template T2 the way go list does in-process
// (list.go:476-503): a FuncMap with context, one execution, and a LF added
// when the output does not end with one (TrackingWriter, :983-1009). No
// process is started.
func runT2(t *testing.T, c *listContext) []byte {
	t.Helper()
	tmpl, err := template.New("main").Funcs(template.FuncMap{
		"context": func() *listContext { return c },
	}).Parse(spec.T2)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{}{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// TestContextFixtureIsT2Output checks that the hand-written healthy text
// is exactly what T2 writes for those values.
func TestContextFixtureIsT2Output(t *testing.T) {
	got := runT2(t, &listContext{
		GOARCH: "amd64", GOOS: "linux", GOROOT: "/opt/hostedtoolcache/go/1.27.3/x64", GOPATH: "/home/runner/go",
		Compiler: "gc", ToolTags: []string{"goamd64.v1"}, ReleaseTags: []string{"go1.1", "go1.2", "go1.3"},
	})
	if string(got) != ctxLinux {
		t.Fatalf("T2 output differs from the fixture:\n%q\n%q", got, ctxLinux)
	}
}

// TestContextRoundTripThroughT2 decodes T2's own output for values that
// exercise every strconv.Quote rule (method record §2.2 item 12): quotes,
// backslashes, LF, CR, TAB, other control bytes, DEL, invalid UTF-8,
// printable and non-printable non-ASCII runes, empty strings, duplicates.
func TestContextRoundTripThroughT2(t *testing.T) {
	cases := []*listContext{
		{GOARCH: "amd64", GOOS: "linux", Compiler: "gc"},
		{
			GOARCH:        "a\"b\\c",
			GOOS:          "line1\nline2\rline3\tend",
			GOROOT:        "\x00\x01\x1f\x7f",
			GOPATH:        "bad\xffutf8\xc3",
			CgoEnabled:    true,
			UseAllFiles:   true,
			Compiler:      "r\u00fcnner \u00a0nbsp \u2028ls \U0001F600 \u200b",
			InstallSuffix: "- \"x\"",
			BuildTags:     []string{"", "a", "a"},
			ToolTags:      []string{"end", "c9-context v1", "- \"q\""},
			ReleaseTags:   []string{"go1.1"},
		},
		{GOROOT: `C:\Program Files\Go`, BuildTags: []string{"x"}, ReleaseTags: make([]string, 0)},
	}
	for i, c := range cases {
		out := runT2(t, c)
		got, err := DecodeContext(out)
		if err != nil {
			t.Fatalf("case %d: %v\n%q", i, err, out)
		}
		want := Context{
			GOARCH: c.GOARCH, GOOS: c.GOOS, GOROOT: c.GOROOT, GOPATH: c.GOPATH,
			CgoEnabled: c.CgoEnabled, UseAllFiles: c.UseAllFiles, Compiler: c.Compiler, InstallSuffix: c.InstallSuffix,
			BuildTags: nonNil(c.BuildTags), ToolTags: nonNil(c.ToolTags), ReleaseTags: nonNil(c.ReleaseTags),
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("case %d:\ngot  %#v\nwant %#v", i, got, want)
		}
	}
}

func ctxErr(t *testing.T, in string) *Error {
	t.Helper()
	_, err := DecodeContext([]byte(in))
	var de *Error
	if !errors.As(err, &de) {
		t.Fatalf("want *Error, got %v", err)
	}
	if de.Call != spec.CallContext {
		t.Fatalf("call %v", de.Call)
	}
	return de
}

// replaceLine returns ctxLinux with line n (1-based) replaced.
func replaceLine(n int, line string) string {
	ls := strings.SplitAfter(ctxLinux, "\n")
	ls[n-1] = line
	return strings.Join(ls, "")
}

func TestContextSyntaxErrors(t *testing.T) {
	cases := []struct {
		name string
		in   string
		kind Kind
		at   string
	}{
		{"header-wrong", replaceLine(1, "c9-context v2\n"), KindLine, "line 1"},
		{"header-only", "c9-context v1\n", KindForm, "line 2"},
		{"header-no-lf", "c9-context v1", KindForm, "line 1"},
		{"cr-inside-value", replaceLine(3, "GOOS \"li\rnux\"\n"), KindLine, "line 3"},
		{"trailing-space", replaceLine(3, "GOOS \"linux\" \n"), KindLine, "line 3"},
		{"leading-space", replaceLine(3, " GOOS \"linux\"\n"), KindLine, "line 3"},
		{"tab-separator", replaceLine(3, "GOOS\t\"linux\"\n"), KindLine, "line 3"},
		{"name-only", replaceLine(3, "GOOS\n"), KindLine, "line 3"},
		{"empty-value", replaceLine(3, "GOOS \n"), KindLine, "line 3"},
		{"lowercase-name", replaceLine(3, "goos \"linux\"\n"), KindLine, "line 3"},
		{"order-swapped", replaceLine(2, "GOOS \"linux\"\n"), KindLine, "line 2"},
		{"single-quoted", replaceLine(3, "GOOS 'l'\n"), KindValue, "line 3"},
		{"backquoted", replaceLine(3, "GOOS `linux`\n"), KindValue, "line 3"},
		{"unterminated", replaceLine(3, "GOOS \"linux\n"), KindValue, "line 3"},
		{"trailing-after-quote", replaceLine(3, "GOOS \"linux\"x\n"), KindValue, "line 3"},
		{"octal-escape", replaceLine(3, "GOOS \"\\101\"\n"), KindCanonical, "line 3"},
		{"unicode-escape-printable", replaceLine(3, "GOOS \"\\u00fc\"\n"), KindCanonical, "line 3"},
		{"upper-hex", replaceLine(3, "GOOS \"\\x7F\"\n"), KindCanonical, "line 3"},
		{"raw-tab", replaceLine(3, "GOOS \"a\tb\"\n"), KindCanonical, "line 3"},
		{"bool-True", replaceLine(6, "CgoEnabled True\n"), KindValue, "line 6"},
		{"bool-quoted", replaceLine(7, "UseAllFiles \"false\"\n"), KindValue, "line 7"},
		{"count-negative", replaceLine(10, "BuildTags -0\n"), KindValue, "line 10"},
		{"count-plus", replaceLine(10, "BuildTags +1\n"), KindValue, "line 10"},
		{"count-hex", replaceLine(10, "BuildTags 0x0\n"), KindValue, "line 10"},
		{"count-overflow", replaceLine(10, "BuildTags 99999999999999999999999\n"), KindValue, "line 10"},
		{"count-huge-short", replaceLine(10, "BuildTags 9000000000\n"), KindCount, "line 11"},
		{"element-unquoted", replaceLine(12, "- goamd64.v1\n"), KindValue, "line 12"},
		{"element-noncanonical", replaceLine(12, "- \"goamd64\\x2ev1\"\n"), KindCanonical, "line 12"},
		{"element-two-spaces", replaceLine(12, "-  \"goamd64.v1\"\n"), KindLine, "line 12"},
		{"element-dash-only", replaceLine(12, "-\n"), KindLine, "line 12"},
		{"element-dash-space", replaceLine(12, "- \n"), KindLine, "line 12"},
		{"end-wrong", replaceLine(17, "END\n"), KindLine, "line 17"},
		{"end-with-space", replaceLine(17, "end \n"), KindLine, "line 17"},
		{"after-end-blank", ctxLinux + "\n", KindTrailing, "line 18"},
		{"after-end-header-no-lf", ctxLinux + "c9-context v1", KindCount, "line 18"},
		{"truncated-in-list", ctxLinux[:strings.Index(ctxLinux, "- \"go1.2\"")], KindForm, "line 15"},
		{"truncated-mid-line", ctxLinux[:strings.Index(ctxLinux, "- \"go1.2\"")+3], KindForm, "line 15"},
		{"bom", "\ufeff" + ctxLinux, KindLine, "line 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := ctxErr(t, c.in)
			if e.Kind != c.kind || e.At != c.at {
				t.Fatalf("got %s at %s (%s), want %s at %s", e.Kind, e.At, e.Detail, c.kind, c.at)
			}
			if e.Offset < 0 || e.Offset > len(c.in) {
				t.Fatalf("offset %d outside the output", e.Offset)
			}
		})
	}
}

func TestContextErrorMessage(t *testing.T) {
	e := ctxErr(t, replaceLine(5, "GOPATH \"\\x41\"\n"))
	const want = `call2-context: canonical at line 5 (byte 93): value "\x41" is not the strconv.Quote form "A"`
	if e.Error() != want {
		t.Fatalf("message\n%s\nwant\n%s", e.Error(), want)
	}
}
