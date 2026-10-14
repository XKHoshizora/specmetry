package testutil

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/secrets"
)

// The M1 project fixtures (acceptance-criteria §1.1, WP1.0) and the stage
// each one is built for (architecture §6.1.1).
var fixtureStages = map[string]string{
	"fx-docs-only":  "docs_only",
	"fx-in-dev":     "in_development",
	"fx-incomplete": "incomplete_docs",
}

func fixtureNames() []string {
	names := make([]string, 0, len(fixtureStages))
	for n := range fixtureStages {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// WP1.0: each fixture is built once and checked by the subtests below,
// because building takes many git processes (slow on Windows).
func TestProjectFixtures(t *testing.T) {
	built := map[string]*Fixture{}
	for _, name := range fixtureNames() {
		built[name] = BuildFixture(t, name)
	}
	for _, name := range fixtureNames() {
		f := built[name]
		t.Run(name+"/reproducible", func(t *testing.T) { checkReproducible(t, f) })
		t.Run(name+"/git_state", func(t *testing.T) { checkGitState(t, f) })
		t.Run(name+"/stage", func(t *testing.T) { checkStage(t, f) })
	}
	t.Run("fx-docs-only/candidate_sources", func(t *testing.T) { checkCandidateSources(t, built["fx-docs-only"]) })
	t.Run("fx-docs-only/client_brief", func(t *testing.T) { checkClientBrief(t, built["fx-docs-only"], built["fx-in-dev"]) })
	t.Run("fx-docs-only/recorded_proposal", func(t *testing.T) { checkRecordedProposal(t, built["fx-docs-only"]) })
	t.Run("fx-in-dev/feature_branch", func(t *testing.T) { checkFeatureBranch(t, built["fx-in-dev"]) })
	t.Run("applications", func(t *testing.T) { checkApplications(t, built["fx-in-dev"], built["fx-incomplete"]) })
}

// A fixture builds into the same commits every time and on every platform:
// fixed dates and identity, isolated git configuration, and file bytes taken
// as they are. The expected commit IDs are in the manifest.
func checkReproducible(t *testing.T, f *Fixture) {
	again := BuildFixture(t, f.Name)
	if !reflect.DeepEqual(f.Refs(), again.Refs()) {
		t.Errorf("two builds differ: %v vs %v", f.Refs(), again.Refs())
	}
	if !reflect.DeepEqual(f.Refs(), f.Manifest.ExpectedRefs) {
		t.Errorf("refs %v, manifest expects %v", f.Refs(), f.Manifest.ExpectedRefs)
	}
}

// The main worktree is on the default branch, clean, and tracks exactly the
// files of repo/; branches are where the manifest puts them.
func checkGitState(t *testing.T, f *Fixture) {
	if got := f.Repo.Git("symbolic-ref", "HEAD"); got != "refs/heads/main" {
		t.Errorf("HEAD = %s", got)
	}
	if got := f.Repo.Git("status", "--porcelain", "--untracked-files=all"); got != "" {
		t.Errorf("worktree is not clean:\n%s", got)
	}
	if got := f.Repo.Git("rev-list", "--count", "main"); got != strconv.Itoa(len(f.Manifest.Commits)) {
		t.Errorf("main has %s commits, manifest lists %d", got, len(f.Manifest.Commits))
	}
	tracked := strings.Split(f.Repo.Git("ls-files"), "\n")
	if want := sourceFiles(t, filepath.Join(f.Source, "repo")); !reflect.DeepEqual(tracked, want) {
		t.Errorf("tracked files %v, repo/ has %v", tracked, want)
	}
	for _, br := range f.Manifest.Branches {
		ahead := f.Repo.Git("rev-list", "--count", br.From+".."+br.Name)
		behind := f.Repo.Git("rev-list", "--count", br.Name+".."+br.From)
		if ahead != strconv.Itoa(len(br.Commits)) || behind != "0" {
			t.Errorf("%s is %s ahead and %s behind %s", br.Name, ahead, behind, br.From)
		}
	}
}

// fx-in-dev: feature/x has one more feature than main, and a second
// worktree can check it out (MVP-WT-01, E2E-4).
func checkFeatureBranch(t *testing.T, f *Fixture) {
	if len(f.Manifest.Branches) != 1 || f.Manifest.Branches[0].Name != "feature/x" {
		t.Fatalf("branches = %+v", f.Manifest.Branches)
	}
	added := f.Repo.Git("diff", "--name-status", "main", "feature/x")
	for _, line := range strings.Split(added, "\n") {
		if !strings.HasPrefix(line, "A\t") {
			t.Errorf("feature/x should only add files, got %q", line)
		}
	}
	if _, err := os.Stat(filepath.Join(f.Repo.Dir, "orders", "export.go")); err == nil {
		t.Error("the feature/x file is present on main")
	}
	wt := f.Repo.WorktreeFor("feature/x")
	if got, want := f.Repo.GitIn(wt, "rev-parse", "HEAD"), f.Refs()["feature/x"]; got != want {
		t.Errorf("worktree HEAD %s, want %s", got, want)
	}
	if _, err := os.Stat(filepath.Join(wt, "orders", "export.go")); err != nil {
		t.Errorf("feature worktree lacks the feature: %v", err)
	}
}

// architecture §6.1.1, checked from the fixture side: each fixture has the
// files that make its stage (MVP-INIT-03). This is an independent reading of
// the rules, not the WP1.1 detector.
func checkStage(t *testing.T, f *Fixture) {
	if f.Manifest.Stage != fixtureStages[f.Name] {
		t.Errorf("manifest stage %q, want %q", f.Manifest.Stage, fixtureStages[f.Name])
	}
	if got := stageByDocumentedRules(t, f.Repo); got != fixtureStages[f.Name] {
		t.Errorf("files make stage %q, want %q", got, fixtureStages[f.Name])
	}
}

// MVP-INIT-04 needs these cases in fx-docs-only: the brief, a README to
// rank below it, and three files the candidate rules exclude.
func checkCandidateSources(t *testing.T, f *Fixture) {
	sizes := map[string]int64{}
	for _, p := range []string{"docs/client-brief.md", "README.md", "CHANGELOG.md", "node_modules/x/README.md", "docs/empty.md"} {
		info, err := os.Stat(filepath.Join(f.Repo.Dir, filepath.FromSlash(p)))
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		sizes[p] = info.Size()
	}
	if sizes["docs/client-brief.md"] < 512 || sizes["docs/empty.md"] != 0 || sizes["CHANGELOG.md"] == 0 || sizes["node_modules/x/README.md"] == 0 {
		t.Errorf("sizes = %v", sizes)
	}
}

var requirementHeading = regexp.MustCompile(`(?m)^### (R[0-9]+) `)

// The brief: 8 requirements, one without a decidable acceptance
// description, two ambiguities (acceptance-criteria §1.1). fx-in-dev uses
// the same bytes.
func checkClientBrief(t *testing.T, docs, inDev *Fixture) {
	brief := readFile(t, filepath.Join(docs.Repo.Dir, "docs", "client-brief.md"))
	if other := readFile(t, filepath.Join(inDev.Repo.Dir, "docs", "client-brief.md")); !bytes.Equal(brief, other) {
		t.Error("fx-in-dev's brief differs from fx-docs-only's")
	}
	sections := requirementHeading.FindAllSubmatchIndex(brief, -1)
	if len(sections) != 8 {
		t.Fatalf("brief has %d requirements, want 8", len(sections))
	}
	b := docs.Manifest.Brief
	// codex-review-012 N2: the same brief carries the same notes.
	if !reflect.DeepEqual(inDev.Manifest.Brief, b) {
		t.Errorf("fx-in-dev's brief notes %+v differ from fx-docs-only's %+v for the same brief", inDev.Manifest.Brief, b)
	}
	if len(b.WithoutAcceptance) != 1 || len(b.Ambiguities) != 2 {
		t.Fatalf("manifest brief notes = %+v", b)
	}
	for i, loc := range sections {
		id := string(brief[loc[2]:loc[3]])
		end := len(brief)
		if i+1 < len(sections) {
			end = sections[i+1][0]
		}
		hasAcceptance := bytes.Contains(brief[loc[0]:end], []byte("验收："))
		if hasAcceptance == slices.Contains(b.WithoutAcceptance, id) {
			t.Errorf("%s: has acceptance = %v, manifest says otherwise", id, hasAcceptance)
		}
	}
	for id, phrase := range b.Ambiguities {
		if !bytes.Contains(brief, []byte(phrase)) {
			t.Errorf("ambiguity %s: %q is not in the brief", id, phrase)
		}
	}
}

// MVP-BL-01 submits fx-docs-only/proposal.json: it must be a valid
// proposal.v1 from a fresh project whose content matches the brief
// (proposal_check_test.go).
func checkRecordedProposal(t *testing.T, f *Fixture) {
	raw := readFile(t, f.Recorded("proposal.json"))
	brief := string(readFile(t, filepath.Join(f.Repo.Dir, "docs", "client-brief.md")))
	for _, p := range proposalProblems(t, raw, brief) {
		t.Error(p)
	}
}

var (
	machinePath = regexp.MustCompile(`[A-Za-z]:[\\/]|/Users/|/home/`)
	anyURL      = regexp.MustCompile(`[a-z]+://[^\s"')]+`)
	anyEmail    = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]+`)
	safeHost    = regexp.MustCompile(`(^|[@./])(example\.(com|org|invalid)|localhost)(:|/|$)`)
)

// Fixtures hold no secrets, machine paths, real hosts or real addresses.
func TestFixturesAreSelfContained(t *testing.T) {
	s, err := secrets.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	root := fixturesRoot(t)
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		text := string(readFile(t, p))
		if id, ok := s.Match(text); ok {
			t.Errorf("%s: matches secret rule %s", rel, id)
		}
		if m := machinePath.FindString(text); m != "" {
			t.Errorf("%s: machine path %q", rel, m)
		}
		for _, u := range anyURL.FindAllString(text, -1) {
			if !safeHost.MatchString(strings.SplitN(strings.SplitN(u, "://", 2)[1], "/", 2)[0] + "/") {
				t.Errorf("%s: URL %q", rel, u)
			}
		}
		for _, e := range anyEmail.FindAllString(text, -1) {
			if _, host, _ := strings.Cut(e, "@"); !safeHost.MatchString("@" + host + "/") {
				t.Errorf("%s: address %q", rel, e)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The sample applications compile and their tests pass, so checks can run
// them later (M2). fx-in-dev gains its deliberately failing test in WP2.9.
func checkApplications(t *testing.T, inDev, incomplete *Fixture) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go is not installed")
	}
	feature := inDev.Repo.WorktreeFor("feature/x")
	for name, dir := range map[string]string{
		"fx-in-dev main":      inDev.Repo.Dir,
		"fx-in-dev feature/x": feature,
		"fx-incomplete":       incomplete.Repo.Dir,
	} {
		cmd := exec.Command("go", "test", "-count=1", "./...")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=local")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s: go test: %v\n%s", name, err, out)
		}
	}
}

// --- helpers -------------------------------------------------------------

func readFile(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// sourceFiles lists the files under dir as sorted slash paths, as git
// ls-files prints them.
func sourceFiles(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

func normalizeSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

var (
	codeExtensions = []string{".go", ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".py", ".rb", ".java", ".kt", ".scala", ".rs", ".c", ".h", ".cc", ".cpp", ".hpp", ".cs", ".swift", ".m", ".php", ".dart", ".lua", ".ex", ".exs", ".erl", ".hs", ".clj", ".vue", ".svelte", ".sql", ".sh", ".ps1"}
	manifestFiles  = []string{"go.mod", "package.json", "pyproject.toml", "setup.py", "requirements.txt", "Cargo.toml", "pom.xml", "build.gradle", "build.gradle.kts", "Gemfile", "composer.json", "mix.exs", "pubspec.yaml", "Package.swift", "CMakeLists.txt"}
	docExtensions  = []string{".md", ".markdown", ".txt", ".rst", ".adoc", ".pdf", ".docx", ".doc", ".odt", ".rtf", ".html", ".htm", ".eml"}
	excludedDirs   = []string{"node_modules", "vendor", "third_party", "dist", "build", "out", "target", "venv", "__pycache__"}
	nonRequirement = regexp.MustCompile(`(?i)^(changelog|license|copying|notice|code_of_conduct|security|contributing)`)
)

// stageByDocumentedRules applies architecture §6.1.1 to the tracked files.
func stageByDocumentedRules(t *testing.T, r *GitRepo) string {
	t.Helper()
	code, substantiveDoc := false, false
	for _, p := range strings.Split(r.Git("ls-files", "--cached", "--others", "--exclude-standard"), "\n") {
		parts := strings.Split(p, "/")
		skip := false
		for _, dir := range parts[:len(parts)-1] {
			skip = skip || strings.HasPrefix(dir, ".") || slices.Contains(excludedDirs, dir)
		}
		if skip {
			continue
		}
		base := parts[len(parts)-1]
		ext := strings.ToLower(filepath.Ext(base))
		if slices.Contains(codeExtensions, ext) || slices.Contains(manifestFiles, base) ||
			strings.HasSuffix(base, ".csproj") || strings.HasSuffix(base, ".sln") {
			code = true
		}
		if !slices.Contains(docExtensions, ext) || nonRequirement.MatchString(base) {
			continue
		}
		if len(parts) == 1 && strings.HasPrefix(strings.ToUpper(base), "README") {
			continue
		}
		info, err := os.Stat(filepath.Join(r.Dir, filepath.FromSlash(p)))
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() >= 512 {
			substantiveDoc = true
		}
	}
	switch {
	case !code:
		return "docs_only"
	case substantiveDoc:
		return "in_development"
	default:
		return "incomplete_docs"
	}
}
