package readycheck_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const module = "github.com/XKHoshizora/specmetry/"

// imports are the imports of the Go files of a directory, test files
// excluded unless asked.
func imports(t *testing.T, dir string, tests bool) map[string][]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || (!tests && strings.HasSuffix(name, "_test.go")) {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			out[name] = append(out[name], p)
		}
	}
	if len(out) == 0 {
		t.Fatalf("no Go file in %s", dir)
	}
	return out
}

// F5 independence: the checker's logic uses only the standard library, so
// it imports neither the derivation engine nor a production package, not
// even through another package.
func TestCheckerImportsOnlyTheStandardLibrary(t *testing.T) {
	allowed := []string{"cmp", "crypto/sha256", "encoding/hex", "errors", "fmt", "slices", "strings"}
	for file, ps := range imports(t, ".", false) {
		for _, p := range ps {
			if !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s", file, p)
			}
		}
	}
}

// F5 independence: the checker's converter maps factgen rows only; it
// imports neither the engine nor the engine's converter.
func TestCheckerConverterIsIndependent(t *testing.T) {
	allowed := []string{"encoding/json", "fmt", module + "internal/testutil/factgen", module + "internal/testutil/readycheck"}
	for file, ps := range imports(t, "fromdump", false) {
		for _, p := range ps {
			if !slices.Contains(allowed, p) {
				t.Errorf("fromdump/%s imports %s", file, p)
			}
		}
	}
}

// F5 independence, the other direction: no production code imports the
// checker or its converter (the engine imports only pure standard
// packages, which its own test checks).
func TestNothingInProductionImportsTheChecker(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	checked := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "testdata", "testutil":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		checked++
		for _, imp := range f.Imports {
			if strings.Contains(imp.Path.Value, "internal/testutil") {
				t.Errorf("%s imports %s", path, imp.Path.Value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 50 {
		t.Fatalf("only %d production files checked", checked)
	}
}
