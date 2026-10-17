package derive_test

import (
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The derivation is a pure function (architecture §7): its production code
// imports only these standard packages, none of which reads a database,
// git, a file, the network, the environment or the clock.
var allowedImports = []string{
	"cmp", "crypto/sha256", "encoding/hex", "errors", "fmt", "io", "slices", "strconv", "strings",
}

func TestDerivationImportsOnlyPureStandardPackages(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	checked := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(allowedImports, path) {
				t.Errorf("%s imports %s", name, path)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no production file checked")
	}
}
