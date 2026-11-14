package collect

import (
	"strings"
	"testing"
)

// chainItems builds a healthy chain for one configuration; mutate may
// change the parents of a derived item by name before it is appended.
func chainItems(t *testing.T, mutate func(name string, parents []Ref) []Ref) []Item {
	t.Helper()
	var items []Item
	refs := map[string]Ref{}
	add := func(file, name string, parents []Ref) {
		if mutate != nil && file == "derived" {
			parents = mutate(name, parents)
		}
		it, ref, err := Line(file, name, "step", parents, map[string]string{"x": name})
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, it)
		refs[name] = ref
	}
	c := "linux/amd64/"
	add("raw", c+"repo-files", nil)
	add("raw", c+"toolchain", nil)
	add("raw", c+"call1-env", nil)
	add("raw", c+"call2-context", nil)
	add("raw", c+"call3-list", nil)
	add("raw", "linux/arm64/I-4/example.com/q", nil)
	add("raw", c+"I-4/example.com/q", nil)
	add("derived", c+"closure", []Ref{refs[c+"call3-list"]})
	add("derived", c+"context/analysis", []Ref{refs[c+"call2-context"]})
	add("derived", c+"consumption", []Ref{refs[c+"call3-list"], refs[c+"I-4/example.com/q"]})
	add("derived", c+"findings", []Ref{refs[c+"consumption"], refs[c+"call3-list"]})
	add("derived", c+"comparison", []Ref{refs[c+"findings"]})
	add("derived", c+"checks", []Ref{refs[c+"repo-files"], refs[c+"toolchain"], refs[c+"call1-env"], refs[c+"call2-context"], refs[c+"call3-list"]})
	return items
}

func TestVerifyProvenanceHealthy(t *testing.T) {
	if p := VerifyProvenance(chainItems(t, nil)); len(p) != 0 {
		t.Fatalf("a healthy chain: %v", p)
	}
}

func TestVerifyProvenanceRejects(t *testing.T) {
	c := "linux/amd64/"
	cases := []struct {
		name   string
		mutate func(string, []Ref) []Ref
		want   string
	}{
		{"missing parent", func(n string, p []Ref) []Ref {
			if n == c+"closure" {
				return nil
			}
			return p
		}, c + "closure: no parent"},
		{"wrong digest", func(n string, p []Ref) []Ref {
			if n == c+"findings" {
				q := append([]Ref(nil), p...)
				q[0].SHA256 = strings.Repeat("0", 64)
				return q
			}
			return p
		}, c + "findings: parent " + c + "consumption digest"},
		{"another configuration's I-4", func(n string, p []Ref) []Ref {
			if n == c+"consumption" {
				return []Ref{p[0], {Name: "linux/arm64/I-4/example.com/q", SHA256: sha([]byte(""))}}
			}
			return p
		}, "belongs to another configuration"},
		{"wrong call", func(n string, p []Ref) []Ref {
			if n == c+"closure" {
				return []Ref{{Name: c + "call2-context"}}
			}
			return p
		}, c + "closure: required parent " + c + "call3-list is missing"},
		{"a later item as parent", func(n string, p []Ref) []Ref {
			if n == c+"context/analysis" {
				return []Ref{{Name: c + "comparison", SHA256: "x"}}
			}
			return p
		}, c + "context/analysis: parent " + c + "comparison is not an earlier record"},
	}
	for _, tc := range cases {
		got := strings.Join(VerifyProvenance(chainItems(t, tc.mutate)), "\n")
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: problems\n%s\nwant one containing %q", tc.name, got, tc.want)
		}
	}
	// A line changed after its reference was taken.
	items := chainItems(t, nil)
	items[0].Line = []byte(`{"name":"tampered"}` + "\n")
	if p := VerifyProvenance(items); len(p) == 0 {
		t.Error("a line that does not match its reference must be reported")
	}
}
