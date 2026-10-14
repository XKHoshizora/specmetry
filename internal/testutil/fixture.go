package testutil

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// FixtureManifest describes how a project fixture is built: its
// fixture.json (acceptance-criteria §1.1).
type FixtureManifest struct {
	Description string `json:"description"`
	// Stage is the stage_detected the files are meant to produce
	// (architecture §6.1.1).
	Stage         string          `json:"stage"`
	DefaultBranch string          `json:"default_branch"`
	Brief         FixtureBrief    `json:"brief"`
	Commits       []FixtureCommit `json:"commits"`
	Branches      []FixtureBranch `json:"branches"`
	// ExpectedRefs pins every branch to its commit ID. A build that gives
	// other IDs is not reproducible, or the fixture changed without the
	// manifest.
	ExpectedRefs map[string]string `json:"expected_refs"`
}

// FixtureBrief notes the deliberate gaps of a fixture's requirement brief.
type FixtureBrief struct {
	WithoutAcceptance []string          `json:"without_acceptance"`
	Ambiguities       map[string]string `json:"ambiguities"`
}

// FixtureCommit is one commit: the listed files are copied from a
// directory of the fixture ("repo" unless From says otherwise) and
// committed at Date (RFC 3339).
type FixtureCommit struct {
	Message string   `json:"message"`
	Date    string   `json:"date"`
	From    string   `json:"from"`
	Paths   []string `json:"paths"`
}

// FixtureBranch is a branch created from another one, with its commits.
// After the branches are built the main worktree is back on the default
// branch.
type FixtureBranch struct {
	Name    string          `json:"name"`
	From    string          `json:"from"`
	Commits []FixtureCommit `json:"commits"`
}

// Fixture is a project fixture built as a git repository.
type Fixture struct {
	Name string
	// Source is the fixture's directory under testdata/fixtures, holding
	// fixture.json, repo/ and the recorded Agent outputs.
	Source   string
	Repo     *GitRepo
	Manifest FixtureManifest
}

// BuildFixture builds the named fixture (testdata/fixtures/<name>) as a git
// repository in a temporary directory, with the isolated configuration of
// NewGitRepo. Commit IDs are the same on every run and platform.
func BuildFixture(t testing.TB, name string) *Fixture {
	t.Helper()
	f := &Fixture{Name: name, Source: filepath.Join(fixturesRoot(t), name)}
	raw, err := os.ReadFile(filepath.Join(f.Source, "fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f.Manifest); err != nil {
		t.Fatalf("%s/fixture.json: %v", name, err)
	}
	if f.Manifest.DefaultBranch != "main" {
		t.Fatalf("%s: default_branch must be main (NewGitRepo initialises main)", name)
	}
	f.Repo = NewGitRepo(t)
	committed := map[string]bool{}
	for _, c := range f.Manifest.Commits {
		f.commit(t, c)
		if c.From == "" || c.From == "repo" {
			for _, p := range c.Paths {
				committed[p] = true
			}
		}
	}
	for _, br := range f.Manifest.Branches {
		f.Repo.Git("checkout", "-q", "-b", br.Name, br.From)
		for _, c := range br.Commits {
			f.commit(t, c)
		}
		f.Repo.Git("checkout", "-q", f.Manifest.DefaultBranch)
	}
	// Every file of repo/ must be committed on the default branch, so the
	// main worktree is clean and holds exactly repo/.
	_ = filepath.WalkDir(filepath.Join(f.Source, "repo"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Fatal(err)
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(filepath.Join(f.Source, "repo"), p)
			if !committed[filepath.ToSlash(rel)] {
				t.Fatalf("%s: repo/%s is not in any commit of fixture.json", name, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	return f
}

func (f *Fixture) commit(t testing.TB, c FixtureCommit) {
	t.Helper()
	from := c.From
	if from == "" {
		from = "repo"
	}
	date, err := time.Parse(time.RFC3339, c.Date)
	if err != nil {
		t.Fatalf("%s: commit %q: %v", f.Name, c.Message, err)
	}
	for _, p := range c.Paths {
		b, err := os.ReadFile(filepath.Join(f.Source, from, filepath.FromSlash(p)))
		if err != nil {
			t.Fatal(err)
		}
		f.Repo.WriteFile(p, string(b))
	}
	f.Repo.CommitAt(c.Message, date)
}

// Refs returns every local branch and the commit it points to.
func (f *Fixture) Refs() map[string]string {
	refs := map[string]string{}
	out := f.Repo.Git("for-each-ref", "--format=%(refname:short) %(objectname)", "refs/heads")
	for _, line := range strings.Split(out, "\n") {
		if name, sha, ok := strings.Cut(line, " "); ok {
			refs[name] = sha
		}
	}
	return refs
}

// Recorded returns the path of a recorded Agent output of the fixture, such
// as proposal.json.
func (f *Fixture) Recorded(file string) string {
	return filepath.Join(f.Source, filepath.FromSlash(file))
}

// fixturesRoot finds testdata/fixtures at the module root by walking up
// from the test's working directory (the package directory).
func fixturesRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		p := filepath.Join(dir, "testdata", "fixtures")
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if info, err := os.Stat(p); err == nil && info.IsDir() {
				return p
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("testdata/fixtures not found above the working directory")
		}
		dir = parent
	}
}
