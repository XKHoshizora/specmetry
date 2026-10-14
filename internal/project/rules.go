package project

import (
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// File is a file of a project: its path relative to the main worktree,
// with forward slashes, and its size. Its content is never read.
type File struct {
	Path string
	Size int64
}

// Stage is a project's starting stage (architecture §6.1.1).
type Stage string

// Stages.
const (
	DocsOnly       Stage = "docs_only"
	InDevelopment  Stage = "in_development"
	IncompleteDocs Stage = "incomplete_docs"
)

// ParseStage parses a stage name.
func ParseStage(s string) (Stage, bool) {
	switch st := Stage(s); st {
	case DocsOnly, InDevelopment, IncompleteDocs:
		return st, true
	}
	return "", false
}

const (
	// MaxCandidates bounds the candidate sources init lists.
	MaxCandidates = 200
	// maxCandidateSize excludes larger files from the candidates.
	maxCandidateSize = 20 << 20
	// substantialSize is the size from which a candidate other than the
	// root README is a substantial document.
	substantialSize = 512
	// maxPathLength is the relPath limit of the exchange schemas.
	maxPathLength = 1024
)

// The rules of architecture §6.1.1.
var (
	// excludedDirs are skipped everywhere, as are directories whose name
	// starts with a dot.
	excludedDirs = set("node_modules", "vendor", "third_party", "dist", "build", "out", "target", "venv", "__pycache__")
	// documentExtensions is the source type whitelist (data-model §4.3).
	documentExtensions = set(".md", ".markdown", ".txt", ".rst", ".adoc", ".pdf", ".docx", ".doc", ".odt", ".rtf", ".html", ".htm", ".eml")
	// notRequirements are name prefixes of common files that are not
	// requirements, compared ignoring case.
	notRequirements = []string{"changelog", "license", "copying", "notice", "code_of_conduct", "security", "contributing"}
	// keywords mark requirement documents: English as whole words, Chinese
	// as substrings, both ignoring case.
	keywords        = set("brief", "requirement", "requirements", "req", "spec", "specs", "prd", "srs", "scope", "proposal")
	chineseKeywords = []string{"需求", "规格", "要求", "说明书"}
	// documentDirs rank the files under them, as top-level directories.
	documentDirs   = set("docs", "doc", "requirements", "spec", "specs", "design")
	codeExtensions = set(".go", ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".py", ".rb", ".java", ".kt", ".scala", ".rs",
		".c", ".h", ".cc", ".cpp", ".hpp", ".cs", ".swift", ".m", ".php", ".dart", ".lua", ".ex", ".exs", ".erl", ".hs",
		".clj", ".vue", ".svelte", ".sql", ".sh", ".ps1")
	manifests        = set("go.mod", "package.json", "pyproject.toml", "setup.py", "requirements.txt", "Cargo.toml", "pom.xml", "build.gradle", "build.gradle.kts", "Gemfile", "composer.json", "mix.exs", "pubspec.yaml", "Package.swift", "CMakeLists.txt")
	manifestSuffixes = []string{".csproj", ".sln"}
	// testCommands are suggested for manifests in the project root, in
	// this order (MVP-INIT-10).
	testCommands = []struct{ manifest, command string }{
		{"go.mod", "go test ./..."},
		{"package.json", "npm test"},
		{"pytest.ini", "pytest"},
		{"Cargo.toml", "cargo test"},
	}
)

func set(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, s := range items {
		m[s] = true
	}
	return m
}

// excludedDir reports whether a directory name is skipped.
func excludedDir(name string) bool {
	return strings.HasPrefix(name, ".") || excludedDirs[name]
}

// inExcludedDir reports whether a path lies in a skipped directory.
func inExcludedDir(p string) bool {
	dirs := strings.Split(p, "/")
	return slices.ContainsFunc(dirs[:len(dirs)-1], excludedDir)
}

func isManifest(name string) bool {
	if manifests[name] {
		return true
	}
	for _, suffix := range manifestSuffixes {
		if strings.HasSuffix(name, suffix) && len(name) > len(suffix) {
			return true
		}
	}
	return false
}

func isCode(f File) bool {
	if inExcludedDir(f.Path) {
		return false
	}
	name := path.Base(f.Path)
	return codeExtensions[strings.ToLower(path.Ext(name))] || isManifest(name)
}

// fitsRelPath reports whether the exchange schemas can carry p as a
// relPath: at most 1024 characters, and none that a JSON Schema "."
// does not match.
func fitsRelPath(p string) bool {
	return p != "" && utf8.ValidString(p) && utf8.RuneCountInString(p) <= maxPathLength &&
		!strings.ContainsAny(p, "\n\r  ")
}

// isCandidate applies the candidate filters: document type, size, the
// excluded directories and names. Project manifests such as
// requirements.txt are code, not documents.
func isCandidate(f File) bool {
	name := path.Base(f.Path)
	lower := strings.ToLower(name)
	if f.Size <= 0 || f.Size > maxCandidateSize || inExcludedDir(f.Path) || !fitsRelPath(f.Path) {
		return false
	}
	if !documentExtensions[strings.ToLower(path.Ext(name))] || isManifest(name) {
		return false
	}
	for _, prefix := range notRequirements {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}
	return true
}

func isRootReadme(p string) bool {
	return !strings.Contains(p, "/") && strings.HasPrefix(strings.ToLower(p), "readme")
}

// rank orders candidates: 1 a requirement keyword in the file name or in a
// directory name on its path, 2 under a top-level document directory, 3 the
// root README, 4 the rest.
func rank(p string) int {
	parts := strings.Split(p, "/")
	if slices.ContainsFunc(parts, hasKeyword) {
		return 1
	}
	if len(parts) > 1 && documentDirs[strings.ToLower(parts[0])] {
		return 2
	}
	if isRootReadme(p) {
		return 3
	}
	return 4
}

func hasKeyword(name string) bool {
	for _, w := range words(name) {
		if keywords[strings.ToLower(w)] {
			return true
		}
	}
	for _, k := range chineseKeywords {
		if strings.Contains(name, k) {
			return true
		}
	}
	return false
}

// words splits a name into runs of ASCII letters, also where a lower-case
// letter is followed by an upper-case one: "client-brief.md" and
// "ClientBrief.md" both hold the word "brief".
func words(name string) []string {
	var out []string
	start := -1
	for i, r := range name {
		letter := r < unicode.MaxASCII && unicode.IsLetter(r)
		switch {
		case !letter:
			if start >= 0 {
				out = append(out, name[start:i])
				start = -1
			}
		case start < 0:
			start = i
		case unicode.IsUpper(r) && unicode.IsLower(rune(name[i-1])):
			out = append(out, name[start:i])
			start = i
		}
	}
	if start >= 0 {
		out = append(out, name[start:])
	}
	return out
}

// candidates returns every candidate source in order.
func candidates(files []File) []string {
	type ranked struct {
		path string
		rank int
	}
	var list []ranked
	for _, f := range files {
		if isCandidate(f) {
			list = append(list, ranked{f.Path, rank(f.Path)})
		}
	}
	slices.SortFunc(list, func(a, b ranked) int {
		if a.rank != b.rank {
			return a.rank - b.rank
		}
		return strings.Compare(a.path, b.path)
	})
	out := make([]string, len(list))
	for i, c := range list {
		out[i] = c.path
	}
	return out
}

// Candidates lists the candidate sources among files, at most
// MaxCandidates, in order (architecture §6.1.1). They are never registered
// automatically.
func Candidates(files []File) []string {
	all := candidates(files)
	if len(all) > MaxCandidates {
		all = all[:MaxCandidates]
	}
	return all
}

// DetectStage detects the starting stage: without code files docs_only;
// with code files in_development if there is a substantial document (a
// candidate of at least 512 bytes other than the root README, among all
// candidates, not only the listed ones), incomplete_docs otherwise.
func DetectStage(files []File) Stage {
	if !slices.ContainsFunc(files, isCode) {
		return DocsOnly
	}
	for _, f := range files {
		if f.Size >= substantialSize && !isRootReadme(f.Path) && isCandidate(f) {
			return InDevelopment
		}
	}
	return IncompleteDocs
}

// TestCommandHints suggests test commands for the manifests in the project
// root (MVP-INIT-10). It looks at file names only.
func TestCommandHints(files []File) []string {
	root := map[string]bool{}
	for _, f := range files {
		if !strings.Contains(f.Path, "/") {
			root[f.Path] = true
		}
	}
	hints := []string{}
	for _, tc := range testCommands {
		if root[tc.manifest] {
			hints = append(hints, tc.command)
		}
	}
	return hints
}
