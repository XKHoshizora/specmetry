package decode

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"regexp"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// TestLC4Variants runs every LC-4 variant and compares the complete
// outcome with the hand-written expectation.
func TestLC4Variants(t *testing.T) {
	vs := LC4Variants()
	if len(vs) == 0 {
		t.Fatal("no variants")
	}
	for _, v := range vs {
		t.Run(v.Name, func(t *testing.T) {
			if err := v.Match(v.Run()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestLC4VariantTable checks the table itself: unique names, a known call,
// a complete expectation of the right shape, and fresh values per call.
func TestLC4VariantTable(t *testing.T) {
	seen := map[string]bool{}
	for _, v := range LC4Variants() {
		if v.Name == "" || seen[v.Name] {
			t.Errorf("empty or duplicate name %q", v.Name)
		}
		seen[v.Name] = true
		if v.V == "" {
			t.Errorf("%s: no V row", v.Name)
		}
		prefix := map[spec.CallKind]string{spec.CallEnv: "env.", spec.CallContext: "ctx.", spec.CallList: "list."}[v.Call]
		if prefix == "" || !strings.HasPrefix(v.Name, prefix) {
			t.Errorf("%s: call %v does not match the name", v.Name, v.Call)
		}
		values := 0
		if v.WantEnv != nil {
			values++
		}
		if v.WantContext != nil {
			values++
		}
		if v.WantPackages != nil {
			values++
		}
		if v.WantOK {
			if v.WantKind != "" || v.WantAt != "" || values != 1 {
				t.Errorf("%s: a decoding variant needs exactly one expected value and no error", v.Name)
			}
		} else if v.WantKind == "" || v.WantAt == "" || values != 0 {
			t.Errorf("%s: a failing variant needs a kind and a location and no value", v.Name)
		}
	}

	// The variants LC-4 must cover (plan §5.2 row LC-4; the task list).
	for _, name := range []string{
		"env.missing-key", "env.extra-key", "env.case-key", "env.null", "env.duplicate-key", "env.type-mismatch", "env.trailing",
		"list.no-module", "list.extra-key", "list.case-key", "list.null-module", "list.duplicate-key", "list.type-standard-string", "list.trailing",
		"list.std", "list.main", "list.dep", "list.dep-omissions",
		"ctx.healthy", "ctx.missing-end", "ctx.count-short", "ctx.count-long", "ctx.noncanonical-x41", "ctx.crlf", "ctx.two-texts", "ctx.extra-line", "ctx.cgo-zero", "ctx.zero-bytes",
	} {
		if !seen[name] {
			t.Errorf("required variant %s is missing", name)
		}
	}

	// Fresh values: changing one table's input does not change another's.
	a, b := LC4Variants(), LC4Variants()
	a[0].Input[0] = 'X'
	if b[0].Input[0] == 'X' {
		t.Fatal("variants share input bytes")
	}
}

// TestLC4DocListsEveryVariant keeps the list in LC4Variants' doc comment
// (the list the implementation review fixes, plan §8.1) equal to the table.
func TestLC4DocListsEveryVariant(t *testing.T) {
	f, err := goparser.ParseFile(token.NewFileSet(), "variants.go", nil, goparser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var doc string
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "LC4Variants" && fd.Doc != nil {
			doc = fd.Doc.Text()
		}
	}
	listed := map[string]bool{}
	for _, m := range regexp.MustCompile(`\b(?:env|ctx|list)\.[a-z0-9-]+[a-z0-9]`).FindAllString(doc, -1) {
		if listed[m] {
			t.Errorf("%s listed twice", m)
		}
		listed[m] = true
	}
	table := map[string]bool{}
	for _, v := range LC4Variants() {
		table[v.Name] = true
		if !listed[v.Name] {
			t.Errorf("%s is not in the doc comment", v.Name)
		}
	}
	for name := range listed {
		if !table[name] {
			t.Errorf("%s is in the doc comment but not in the table", name)
		}
	}
}

// TestV34HealthyValues spells out V-34: the healthy text's "", false and
// count 0 are obtained values.
func TestV34HealthyValues(t *testing.T) {
	c, err := DecodeContext([]byte(ctxLinux))
	if err != nil {
		t.Fatal(err)
	}
	if c.CgoEnabled || c.UseAllFiles || c.InstallSuffix != "" || c.BuildTags == nil || len(c.BuildTags) != 0 {
		t.Fatalf("V-34 values: %+v", c)
	}
	if len(c.ToolTags) == 0 || len(c.ReleaseTags) == 0 {
		t.Fatalf("V-34 tags: %+v", c)
	}
}

// TestMatchRejectsWrongOutcomes makes sure Match is not vacuous.
func TestMatchRejectsWrongOutcomes(t *testing.T) {
	vs := LC4Variants()
	byName := map[string]Variant{}
	for _, v := range vs {
		byName[v.Name] = v
	}
	ok := byName["list.dep"]
	o := ok.Run()
	o.Packages[0].Module.Sum = "h1:other="
	if ok.Match(o) == nil {
		t.Error("a changed Sum matched")
	}
	o = ok.Run()
	delete(o.Packages[0].Module.Present, "GoMod")
	if ok.Match(o) == nil {
		t.Error("a changed Present matched")
	}
	bad := byName["env.null"]
	o = bad.Run()
	o.Err.Kind = KindType
	if bad.Match(o) == nil {
		t.Error("a different kind matched")
	}
	o = bad.Run()
	o.Err.At = "$.GOOS"
	if bad.Match(o) == nil {
		t.Error("a different location matched")
	}
	if bad.Match(byName["env.healthy"].Run()) == nil {
		t.Error("a success matched a failing variant")
	}
	if byName["ctx.healthy"].Match(byName["ctx.crlf"].Run()) == nil {
		t.Error("a failure matched a decoding variant")
	}
	if byName["ctx.healthy"].Match(byName["env.healthy"].Run()) == nil {
		t.Error("an outcome of another call matched")
	}
}
