package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestT2Binding(t *testing.T) {
	n, sum, ok := T2Binding(T2)
	if !ok || n != 688 || sum != T2SHA256 {
		t.Fatalf("T2 binding: len %d sha %s ok %v", n, sum, ok)
	}
	if _, _, ok := T2Binding(T2 + " "); ok {
		t.Fatal("a changed T2 must not bind")
	}
	if strings.ContainsAny(T2, "\"'\\\n\r\t") {
		t.Fatal("T2 must not contain quotes, backslashes or line breaks")
	}
}

// TestT2MatchesDocuments cross-checks the copy against the two reviewed
// documents that carry T2 (method record §3.3.1, implementation plan §12.14
// item 2). It reads the repository's documents only.
func TestT2MatchesDocuments(t *testing.T) {
	for _, doc := range []string{"wp2.5-c9-exception-identity-method.md", "wp2.5-implementation-plan.md"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "docs", doc))
		if err != nil {
			t.Fatal(err)
		}
		found := 0
		for _, line := range strings.Split(string(b), "\n") {
			s := strings.TrimSpace(line)
			if strings.HasPrefix(s, "{{with context}}c9-context v1") {
				found++
				if s != T2 {
					t.Errorf("%s: T2 differs from the document", doc)
				}
			}
		}
		if found != 1 {
			t.Errorf("%s: %d T2 lines", doc, found)
		}
	}
}

func TestArgs(t *testing.T) {
	a := Args(CallContext)
	if len(a) != 5 || a[3] != T2 || a[4] != "unsafe" {
		t.Fatalf("call 2 args: %q", a)
	}
	want := "list -e -deps -json=ImportPath,Dir,Name,Standard,Module,GoFiles,CgoFiles,IgnoredGoFiles,Imports,ImportMap,ForTest,Error,DepsErrors,Incomplete -tags nowebui ./..."
	if got := strings.Join(Args(CallList), " "); got != want {
		t.Fatalf("call 3 args: %s", got)
	}
	if got := strings.Join(Args(CallEnv), " "); got != "env -json GOROOT GOVERSION GOOS GOARCH CGO_ENABLED GOFLAGS GOWORK GOTOOLCHAIN" {
		t.Fatalf("call 1 args: %s", got)
	}
}

func TestCandidateCounts(t *testing.T) {
	counts := map[string]int{"linux": 4, "darwin": 7, "windows": 1}
	total := 0
	for goos, n := range counts {
		for _, c := range ConfigsFor(goos) {
			got := CandidatesFor(c)
			if len(got) != n {
				t.Errorf("%s: %d candidates, want %d", c, len(got), n)
			}
			total += len(got)
		}
	}
	if total != 24 {
		t.Fatalf("configuration entries: %d, want 24", total)
	}
}
