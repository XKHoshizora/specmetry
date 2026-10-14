package project_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/project"
)

func files(spec ...any) []project.File {
	var out []project.File
	for i := 0; i < len(spec); i += 2 {
		out = append(out, project.File{Path: spec[i].(string), Size: int64(spec[i+1].(int))})
	}
	return out
}

// The files of fx-docs-only (MVP-INIT-04).
var docsOnly = files(
	"CHANGELOG.md", 44,
	"README.md", 157,
	"docs/client-brief.md", 2255,
	"docs/empty.md", 0,
	"node_modules/x/README.md", 120,
)

// architecture §6.1.1, candidate sources.
func TestCandidates(t *testing.T) {
	cases := []struct {
		name  string
		files []project.File
		want  []string
	}{
		{"fx-docs-only", docsOnly, []string{"docs/client-brief.md", "README.md"}},
		{"document types only, any case", files(
			"a.md", 1, "b.MARKDOWN", 1, "c.txt", 1, "d.rst", 1, "e.adoc", 1, "f.pdf", 1, "g.docx", 1, "h.doc", 1,
			"i.odt", 1, "j.rtf", 1, "k.html", 1, "l.htm", 1, "m.eml", 1, "n.go", 1, "o.png", 1, "p.json", 1, "q", 1,
		), []string{"a.md", "b.MARKDOWN", "c.txt", "d.rst", "e.adoc", "f.pdf", "g.docx", "h.doc", "i.odt", "j.rtf", "k.html", "l.htm", "m.eml"}},
		{"empty and over 20 MiB", files(
			"empty.md", 0, "limit.md", 20<<20, "over.md", 20<<20+1,
		), []string{"limit.md"}},
		{"non-requirement files, any case", files(
			"CHANGELOG.md", 1, "changelog.rst", 1, "LICENSE.md", 1, "license-notes.txt", 1, "COPYING.txt", 1,
			"NOTICE.md", 1, "CODE_OF_CONDUCT.md", 1, "security.md", 1, "CONTRIBUTING.md", 1, "docs/LICENSE.md", 1,
			"guide.md", 1,
		), []string{"guide.md"}},
		{"excluded directories", files(
			".github/ISSUE.md", 1, "a/.hidden/x.md", 1, ".specmetry/x.md", 1, "node_modules/a.md", 1, "a/node_modules/b.md", 1,
			"vendor/a.md", 1, "third_party/a.md", 1, "dist/a.md", 1, "build/a.md", 1, "out/a.md", 1, "target/a.md", 1,
			"venv/a.md", 1, "pkg/__pycache__/a.md", 1, ".notes.md", 1, "builds/a.md", 1,
		), []string{".notes.md", "builds/a.md"}},
		{"project manifests are not documents", files(
			"requirements.txt", 1, "docs/requirements.md", 1,
		), []string{"docs/requirements.md"}},
		{"order: keywords, document directories, root README, the rest", files(
			"zz.md", 1,
			"sub/README.md", 1,
			"README.md", 1,
			"readme.txt", 1,
			"design/api.md", 1,
			"Docs/guide.md", 1,
			"docs/guide.md", 1,
			"spec/overview.md", 1,
			"docs/prd.md", 1,
			"ClientBrief.md", 1,
			"client-brief.md", 1,
			"req-001.md", 1,
			"SRS.docx", 1,
			"scope.md", 1,
			"proposal.md", 1,
			"specs.md", 1,
			"requirement.md", 1,
			"requirements/overview.md", 1,
			"old/specs/x.md", 1,
			"客户需求.md", 1,
			"规格.txt", 1,
			"项目要求.md", 1,
			"使用说明书.pdf", 1,
			"inspector.md", 1,
			"request.md", 1,
			"proposals.md", 1,
			"prescope.md", 1,
		), []string{
			"ClientBrief.md", "SRS.docx", "client-brief.md", "docs/prd.md", "old/specs/x.md", "proposal.md",
			"req-001.md", "requirement.md", "requirements/overview.md", "scope.md", "spec/overview.md", "specs.md",
			"使用说明书.pdf", "客户需求.md", "规格.txt", "项目要求.md",
			"Docs/guide.md", "design/api.md", "docs/guide.md",
			"README.md", "readme.txt",
			"inspector.md", "prescope.md", "proposals.md", "request.md", "sub/README.md", "zz.md",
		}},
		{"paths a relPath cannot carry", files(
			"line\nbreak.md", 1, strings.Repeat("a", 1022)+".md", 1, strings.Repeat("b", 1021)+".md", 1,
		), []string{strings.Repeat("b", 1021) + ".md"}},
	}
	for _, c := range cases {
		if got := project.Candidates(c.files); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

// At most 200 candidates, cut in order.
func TestCandidatesAreCapped(t *testing.T) {
	var fs []project.File
	for i := range 250 {
		fs = append(fs, project.File{Path: fmt.Sprintf("notes/n%03d.md", i), Size: 1})
	}
	fs = append(fs, project.File{Path: "brief.md", Size: 1})
	got := project.Candidates(fs)
	if len(got) != project.MaxCandidates || got[0] != "brief.md" || got[199] != "notes/n198.md" {
		t.Fatalf("got %d candidates: first %q, last %q", len(got), got[0], got[len(got)-1])
	}
}

// architecture §6.1.1, stage detection.
func TestDetectStage(t *testing.T) {
	brief := project.File{Path: "docs/client-brief.md", Size: 2255}
	cases := []struct {
		name  string
		files []project.File
		want  project.Stage
	}{
		{"fx-docs-only", docsOnly, project.DocsOnly},
		{"nothing", nil, project.DocsOnly},
		{"code and a brief", append(files("main.go", 10), brief), project.InDevelopment},
		{"code and a README only", files("main.go", 10, "README.md", 4096), project.IncompleteDocs},
		{"code and a small document", files("main.go", 10, "docs/notes.md", 511), project.IncompleteDocs},
		{"a document of exactly 512 bytes", files("main.go", 10, "docs/notes.md", 512), project.InDevelopment},
		{"a large excluded document", files("main.go", 10, "CHANGELOG.md", 9000, "node_modules/x/spec.md", 9000), project.IncompleteDocs},
		{"a sub-directory README counts", files("main.go", 10, "sub/README.md", 600), project.InDevelopment},
		{"code only in excluded directories", append(files("node_modules/x/index.js", 10, ".github/build.sh", 10, "vendor/a.go", 1), brief), project.DocsOnly},
		{"an empty code file is code", files("pkg/__init__.py", 0, "app.PY", 0), project.IncompleteDocs},
		{"a manifest is code", files("package.json", 10), project.IncompleteDocs},
		{"manifests by pattern", files("App.csproj", 10), project.IncompleteDocs},
		{"solution", files("App.sln", 10), project.IncompleteDocs},
		{"Swift package", files("Package.swift", 10), project.IncompleteDocs},
		{"not code", files("data.csv", 10, "image.png", 10, "notes.md", 600), project.DocsOnly},
	}
	for _, c := range cases {
		if got := project.DetectStage(c.files); got != c.want {
			t.Errorf("%s: DetectStage = %s, want %s", c.name, got, c.want)
		}
	}
	for _, ext := range []string{".go", ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".py", ".rb", ".java", ".kt", ".scala", ".rs", ".c", ".h", ".cc", ".cpp", ".hpp", ".cs", ".swift", ".m", ".php", ".dart", ".lua", ".ex", ".exs", ".erl", ".hs", ".clj", ".vue", ".svelte", ".sql", ".sh", ".ps1"} {
		if got := project.DetectStage(files("src/x"+ext, 1)); got != project.IncompleteDocs {
			t.Errorf("%s is not taken for code", ext)
		}
	}
	for _, name := range []string{"go.mod", "package.json", "pyproject.toml", "setup.py", "requirements.txt", "Cargo.toml", "pom.xml", "build.gradle", "build.gradle.kts", "x.csproj", "x.sln", "Gemfile", "composer.json", "mix.exs", "pubspec.yaml", "Package.swift", "CMakeLists.txt"} {
		if got := project.DetectStage(files("sub/"+name, 1)); got != project.IncompleteDocs {
			t.Errorf("%s is not taken for a manifest", name)
		}
	}
}

// The stage needs every candidate, not only the 200 that are listed.
func TestDetectStageLooksBeyondTheListedCandidates(t *testing.T) {
	fs := files("main.go", 1, "zz/notes.md", 4096)
	for i := range 250 {
		fs = append(fs, project.File{Path: fmt.Sprintf("brief-%03d.md", i), Size: 10})
	}
	if got := project.DetectStage(fs); got != project.InDevelopment {
		t.Fatalf("DetectStage = %s", got)
	}
}

// MVP-INIT-10 [P1]: test command hints from root manifests.
func TestTestCommandHints(t *testing.T) {
	got := project.TestCommandHints(files("go.mod", 1, "package.json", 1, "pytest.ini", 1, "Cargo.toml", 1, "sub/pom.xml", 1))
	want := []string{"go test ./...", "npm test", "pytest", "cargo test"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("hints = %q, want %q", got, want)
	}
	if got := project.TestCommandHints(files("README.md", 1)); len(got) != 0 {
		t.Fatalf("hints without manifests = %q", got)
	}
}

func TestParseStage(t *testing.T) {
	for _, s := range []string{"docs_only", "in_development", "incomplete_docs"} {
		if got, ok := project.ParseStage(s); !ok || string(got) != s {
			t.Errorf("ParseStage(%q) = %q, %v", s, got, ok)
		}
	}
	for _, s := range []string{"", "DOCS_ONLY", "in-development", "other"} {
		if _, ok := project.ParseStage(s); ok {
			t.Errorf("ParseStage(%q) accepted", s)
		}
	}
}
