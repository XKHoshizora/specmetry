package decode

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

const healthySum = `golang.org/x/mod v0.42.0 h1:AAAA=
golang.org/x/mod v0.42.0/go.mod h1:BBBB=
modernc.org/libc v1.77.1 h1:CCCC=
modernc.org/libc v1.77.1/go.mod h1:DDDD=
`

func TestGoSumHealthy(t *testing.T) {
	g, problems := ParseGoSum([]byte(healthySum))
	if len(problems) != 0 {
		t.Fatalf("problems %v", problems)
	}
	sum := sha256.Sum256([]byte(healthySum))
	if g.Size != len(healthySum) || g.SHA256 != hex.EncodeToString(sum[:]) || g.Lines != 4 || len(g.Duplicates) != 0 {
		t.Fatalf("%+v", g)
	}
	for _, c := range []struct {
		path, version, h1, gomod string
	}{
		{"golang.org/x/mod", "v0.42.0", "h1:AAAA=", "h1:BBBB="},
		{"modernc.org/libc", "v1.77.1", "h1:CCCC=", "h1:DDDD="},
	} {
		if h, ok := g.H1(c.path, c.version); !ok || h != c.h1 {
			t.Errorf("H1(%s %s) = %q %v", c.path, c.version, h, ok)
		}
		if h, ok := g.GoModH1(c.path, c.version); !ok || h != c.gomod {
			t.Errorf("GoModH1(%s %s) = %q %v", c.path, c.version, h, ok)
		}
	}
	if _, ok := g.H1("modernc.org/libc", "v1.77.2"); ok {
		t.Error("an absent version has a record")
	}
	if _, ok := g.H1("modernc.org/libc", "v1.77.1/go.mod"); ok {
		t.Error("H1 must not read a /go.mod key through its version")
	}
	want := []SumEntry{
		{SumKey{"golang.org/x/mod", "v0.42.0", false}, []string{"h1:AAAA="}, []int{1}, true},
		{SumKey{"golang.org/x/mod", "v0.42.0", true}, []string{"h1:BBBB="}, []int{2}, true},
		{SumKey{"modernc.org/libc", "v1.77.1", false}, []string{"h1:CCCC="}, []int{3}, true},
		{SumKey{"modernc.org/libc", "v1.77.1", true}, []string{"h1:DDDD="}, []int{4}, true},
	}
	if got := g.Entries(); !reflect.DeepEqual(got, want) {
		t.Fatalf("entries\n%+v\nwant\n%+v", got, want)
	}
}

// TestEmptyGoModHash recomputes cmd/go's constant with the Hash1 algorithm
// (method record §2.2 item 6) over a one-file tree "go.mod" with no bytes
// (fetch.go:842-846 names the file go.mod).
func TestEmptyGoModHash(t *testing.T) {
	empty := sha256.Sum256(nil)
	line := hex.EncodeToString(empty[:]) + "  go.mod\n"
	sum := sha256.Sum256([]byte(line))
	if want := "h1:" + base64.StdEncoding.EncodeToString(sum[:]); EmptyGoModHash != want {
		t.Fatalf("EmptyGoModHash %s, recomputed %s", EmptyGoModHash, want)
	}
}

func TestGoSumEachProblem(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []SumProblem // Detail is not compared
	}{
		{"two-fields", "a v1\n", []SumProblem{{Line: 1, Kind: SumFields}}},
		{"four-fields", "a v1 h1:x= extra\n", []SumProblem{{Line: 1, Kind: SumFields}}},
		{"one-field", "a\n", []SumProblem{{Line: 1, Kind: SumFields}}},
		{"other-algorithm", "a v1 h2:x=\n", []SumProblem{{Line: 1, Kind: SumHashPrefix}}},
		{"no-prefix", "a v1 x=\n", []SumProblem{{Line: 1, Kind: SumHashPrefix}}},
		{"uppercase-prefix", "a v1 H1:x=\n", []SumProblem{{Line: 1, Kind: SumHashPrefix}}},
		{"empty-gomod-hash", "a v1/go.mod " + EmptyGoModHash + "\n", []SumProblem{{Line: 1, Kind: SumEmptyGoMod}}},
		{"conflict", "a v1 h1:x=\na v1 h1:y=\n", []SumProblem{{Line: 2, Kind: SumConflict}}},
		{"conflict-gomod", "a v1/go.mod h1:x=\na v1/go.mod h1:y=\n", []SumProblem{{Line: 2, Kind: SumConflict}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, got := ParseGoSum([]byte(c.in))
			if len(got) != len(c.want) {
				t.Fatalf("got %v", got)
			}
			for i := range got {
				if got[i].Line != c.want[i].Line || got[i].Kind != c.want[i].Kind || got[i].Detail == "" {
					t.Fatalf("got %v want %v", got[i], c.want[i])
				}
			}
		})
	}
}

// TestGoSumAllProblemsKept: every line is evaluated and every problem kept
// in line order; the parse does not stop at the first.
func TestGoSumAllProblemsKept(t *testing.T) {
	in := strings.Join([]string{
		"a v1 h1:a1=",                   // 1
		"",                              // 2 (blank, skipped)
		"b v1",                          // 3 fields
		"c v1 sha256:x",                 // 4 hash-prefix
		"a v1 h1:a2=",                   // 5 conflict with line 1
		"   \t ",                        // 6 (blank)
		"d v1/go.mod " + EmptyGoModHash, // 7 empty-gomod-hash
		"a v1 h1:a1=",                   // 8 duplicate of line 1
		"a v1 h1:a3=",                   // 9 conflict with line 1
		"e v1 h1:e=",                    // 10 healthy
		"x y z w",                       // 11 fields
	}, "\n") + "\n"
	g, got := ParseGoSum([]byte(in))
	want := []struct {
		line int
		kind SumProblemKind
	}{{3, SumFields}, {4, SumHashPrefix}, {5, SumConflict}, {7, SumEmptyGoMod}, {9, SumConflict}, {11, SumFields}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i, w := range want {
		if got[i].Line != w.line || got[i].Kind != w.kind {
			t.Errorf("problem %d: %v, want line %d %s", i, got[i], w.line, w.kind)
		}
	}
	if !strings.Contains(got[2].Detail, "line 1") || !strings.Contains(got[4].Detail, "line 1") {
		t.Errorf("conflict details do not name the earlier line: %v", got)
	}
	if g.Lines != 11 {
		t.Errorf("lines %d", g.Lines)
	}
	if !reflect.DeepEqual(g.Duplicates, []SumDuplicate{{Line: 8, First: 1, Key: SumKey{"a", "v1", false}}}) {
		t.Errorf("duplicates %+v", g.Duplicates)
	}
	if _, ok := g.H1("a", "v1"); ok {
		t.Error("a conflicting key has a lookup result")
	}
	if _, ok := g.H1("c", "v1"); ok {
		t.Error("a key seen only on a hash-prefix line has a lookup result")
	}
	if _, ok := g.GoModH1("d", "v1"); ok {
		t.Error("a key seen only on an empty-gomod-hash line has a lookup result")
	}
	if h, ok := g.H1("e", "v1"); !ok || h != "h1:e=" {
		t.Error("a healthy key lost its record")
	}
	want2 := []SumEntry{
		{SumKey{"a", "v1", false}, []string{"h1:a1=", "h1:a2=", "h1:a3="}, []int{1, 5, 8, 9}, false},
		{SumKey{"e", "v1", false}, []string{"h1:e="}, []int{10}, true},
	}
	if e := g.Entries(); !reflect.DeepEqual(e, want2) {
		t.Errorf("entries %+v", e)
	}
}

func TestGoSumDuplicateIsNotConflict(t *testing.T) {
	g, problems := ParseGoSum([]byte("a v1 h1:x=\na  v1\th1:x=\na v1 h1:x=\n"))
	if len(problems) != 0 {
		t.Fatalf("problems %v", problems)
	}
	if h, ok := g.H1("a", "v1"); !ok || h != "h1:x=" {
		t.Fatal("a repeated identical line made the key unusable")
	}
	want := []SumDuplicate{{2, 1, SumKey{"a", "v1", false}}, {3, 1, SumKey{"a", "v1", false}}}
	if !reflect.DeepEqual(g.Duplicates, want) {
		t.Fatalf("%+v", g.Duplicates)
	}
}

// TestGoSumBadLineTaintsKey: a key with a valid hash and also a problem line
// is not usable, so a caller cannot take its hash past the failure.
func TestGoSumBadLineTaintsKey(t *testing.T) {
	g, problems := ParseGoSum([]byte("a v1/go.mod h1:x=\na v1/go.mod " + EmptyGoModHash + "\nb v1 h1:y=\nb v1 md5:z\n"))
	if len(problems) != 2 {
		t.Fatalf("problems %v", problems)
	}
	if _, ok := g.GoModH1("a", "v1"); ok {
		t.Error("a key with an empty-gomod-hash line is usable")
	}
	if _, ok := g.H1("b", "v1"); ok {
		t.Error("a key with a hash-prefix line is usable")
	}
}

// TestGoSumCRLF documents the choice: strings.Fields treats '\r' as white
// space, as cmd/go does, so CRLF parses like LF with no problem.
func TestGoSumCRLF(t *testing.T) {
	crlf := strings.ReplaceAll(healthySum, "\n", "\r\n")
	g1, p1 := ParseGoSum([]byte(healthySum))
	g2, p2 := ParseGoSum([]byte(crlf))
	if len(p1) != 0 || len(p2) != 0 {
		t.Fatalf("problems %v %v", p1, p2)
	}
	if !reflect.DeepEqual(g1.Entries(), g2.Entries()) || g1.Lines != g2.Lines {
		t.Fatal("CRLF changed the parse")
	}
	if g1.SHA256 == g2.SHA256 || g2.Size != len(crlf) {
		t.Fatal("the identity must still be of the bytes read")
	}
	// A lone CR is not a line break: the line has six fields.
	if _, p := ParseGoSum([]byte("a v1 h1:x=\rb v1 h1:y=\n")); len(p) != 1 || p[0].Kind != SumFields {
		t.Fatalf("lone CR: %v", p)
	}
}

func TestGoSumLineCounting(t *testing.T) {
	for in, lines := range map[string]int{
		"":                    0,
		"\n":                  1,
		"\n\n":                2,
		"a v1 h1:x=":          1,
		"a v1 h1:x=\n":        1,
		"a v1 h1:x=\n\n":      2,
		"\na v1 h1:x=\nb v1":  3,
		"a v1 h1:x=\r\n\r\n ": 3,
	} {
		g, _ := ParseGoSum([]byte(in))
		if g.Lines != lines {
			t.Errorf("%q: %d lines, want %d", in, g.Lines, lines)
		}
	}
	_, p := ParseGoSum([]byte("\na v1 h1:x=\nb v1"))
	if len(p) != 1 || p[0].Line != 3 {
		t.Fatalf("%v", p)
	}
}

func TestGoSumZeroValueIsSafe(t *testing.T) {
	var g GoSum
	if _, ok := g.H1("a", "v1"); ok {
		t.Fatal("zero GoSum has a record")
	}
	if len(g.Entries()) != 0 {
		t.Fatal("zero GoSum has entries")
	}
}
