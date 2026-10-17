package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// projectDir is a new directory named base, outside git, holding a
// document, with its canonical path.
func projectDir(t *testing.T, base string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, base)
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "brief.md"), []byte("# brief\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// gitProjectDir is projectDir as a git repository with one commit, made
// with repo's isolated git.
func gitProjectDir(t *testing.T, repo *testutil.GitRepo, base string) string {
	t.Helper()
	dir := projectDir(t, base)
	repo.GitIn(dir, "init", "-q", "-b", "main")
	repo.GitIn(dir, "add", "-A")
	repo.GitIn(dir, "commit", "-q", "-m", "first")
	return dir
}

// markerName is the name in dir's marker, "" without one.
func markerName(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(project.MarkerPath(dir))
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	var m project.Marker
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m.Name
}

// The project name holds no code fence, as a target's name (architecture
// §10.2; implementation-plan M1, the security fix before M1's final
// acceptance). init refuses one, given with --name or taken from the main
// worktree's directory name, in a git project and outside git: exit 2 at
// --name or at the directory name (whose message says to pass --name),
// error.v1, the name not echoed, before anything is registered: no data
// directory on a first init, the data directory unchanged byte for byte
// when it holds other projects, no marker, the repository unchanged. A
// valid --name registers a directory whose name holds a fence. Single and
// double backticks, Chinese, Japanese, emoji and 200 characters are valid;
// 201 are not. Successful output conforms to init.v1.
func TestInitRefusesACodeFenceInTheProjectName(t *testing.T) {
	home := testutil.TempHome(t)
	canary := testutil.NewCanary(t)
	repo := committedRepo(t)
	fencedName := "```go func " + strings.ReplaceAll(canary, "-", "_") + "() {} ```"
	fencedPlain := projectDir(t, "rel```"+canary)
	fencedGit := gitProjectDir(t, repo, "app```"+canary)
	plain := projectDir(t, "plain")

	refused := func(what, dir, pointer string, args ...string) {
		t.Helper()
		before := testutil.SettledSnapshot(t, dir)
		homeBefore := ""
		if exists(home.Path) {
			homeBefore = treeDigest(t, home.Path)
		}
		t.Chdir(dir)
		code, stdout, stderr := execute(t, append([]string{"init", "--json"}, args...)...)
		env := errorV1(t, what, wholeJSON(t, stdout), apperr.InvalidInput)
		if code != apperr.ExitInvalidInput || strings.Contains(stdout+stderr, canary) || strings.Contains(stdout+stderr, "```") {
			t.Errorf("%s: exit %d\nstdout %s\nstderr %s", what, code, stdout, stderr)
		}
		if pointer == "--name" {
			if len(env.Error.Details) != 1 || env.Error.Details[0].Pointer != "--name" || !strings.Contains(env.Error.Details[0].Problem, "code fence") {
				t.Errorf("%s: details %+v", what, env.Error.Details)
			}
		} else if !strings.Contains(env.Error.Message, "code fence") || !strings.Contains(env.Error.Message, "pass --name") {
			// The directory name's error says why and what to do; it has
			// no details, as before.
			t.Errorf("%s: the message does not give the reason and say to pass --name: %s", what, env.Error.Message)
		}
		code, stdout, stderr = execute(t, append([]string{"init"}, args...)...)
		if code != apperr.ExitInvalidInput || stdout != "" || !strings.Contains(stderr, "code fence") || strings.Contains(stderr, canary) {
			t.Errorf("%s, text: exit %d\nstdout %q\nstderr %q", what, code, stdout, stderr)
		}
		if exists(project.MarkerPath(dir)) {
			t.Errorf("%s: a marker was written", what)
		}
		testutil.AssertTreeUnchanged(t, dir, before)
		if homeBefore == "" && exists(home.Path) {
			t.Errorf("%s: the data directory was created", what)
		}
		if homeBefore != "" && treeDigest(t, home.Path) != homeBefore {
			t.Errorf("%s: the data directory changed", what)
		}
	}
	// A first init: no data directory yet.
	refused("--name, git", repo.Dir, "--name", "--name", fencedName)
	refused("--name, a fence inside", repo.Dir, "--name", "--name", "demo ``` "+canary)
	refused("--name, outside git", plain, "--name", "--name", fencedName)
	refused("the directory name, outside git", fencedPlain, "name (the directory name)")
	refused("the directory name, git", fencedGit, "name (the directory name)")

	// With a project registered: the data directory does not change.
	mustInit(t, plain, "--name", "plain")
	refused("--name, with a project", repo.Dir, "--name", "--name", fencedName)
	refused("the directory name, with a project", fencedGit, "name (the directory name)")
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("a refused name reached the data directory: %v", hits)
	}

	// A valid --name registers the directory whose name holds a fence (its
	// path, canary included, is stored as the worktree's path).
	out := mustInit(t, fencedGit, "--name", "release candidate")
	if !out.Created || out.Project.Name != "release candidate" || markerName(t, fencedGit) != "release candidate" {
		t.Fatalf("a valid --name: %+v, marker %q", out, markerName(t, fencedGit))
	}
	for i, name := range []string{"`v1`", "``v1`` final", "首期项目", "プロジェクト", "🚀 launch", strings.Repeat("名", 200)} {
		dir := projectDir(t, "p"+string(rune('a'+i)))
		out := mustInit(t, dir, "--name", name)
		got := rows(t, home, `SELECT name FROM projects WHERE id = ?`, out.Project.ID)
		if out.Project.Name != name || !equalRows(got, [][]string{{name}}) || markerName(t, dir) != name {
			t.Errorf("--name %.12q: output %q, stored %v, marker %q", name, out.Project.Name, got, markerName(t, dir))
		}
	}
	long := projectDir(t, "long")
	t.Chdir(long)
	if code, stdout, _ := execute(t, "init", "--json", "--name", strings.Repeat("名", 201)); code != apperr.ExitInvalidInput {
		t.Errorf("201 characters: exit %d\n%s", code, stdout)
	}
}

// setRegisteredName stores name as the project's name in the database and
// its marker, as a version that accepted it would have.
func setRegisteredName(t *testing.T, home datadir.Dir, dir, projectID, name string) {
	t.Helper()
	ctx := context.Background()
	db, _, err := sqlite.Open(ctx, home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE projects SET name = ? WHERE id = ?`, name, projectID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(project.Marker{Schema: project.MarkerSchema, ProjectID: projectID, Name: name})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(project.MarkerPath(dir), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// A project registered by an earlier version under a name that holds a
// code fence (simulated by storing it). init does not rename a project and
// could not show the name in init.v1, so in that project it is refused,
// exit 2 with error.v1 and not an internal error: run again, with the
// registered name, with another name, with --stage, and from a new linked
// worktree. The name is not echoed, and nothing changes: not the name in
// the database or the marker, not a worktree, the stage or the seq. The
// other commands still work; they do not show the name.
func TestInitWithARegisteredNameThatHoldsACodeFence(t *testing.T) {
	home := testutil.TempHome(t)
	repo := committedRepo(t)
	out := mustInit(t, repo.Dir, "--name", "legacy")
	legacy := "legacy ```go x``` "
	setRegisteredName(t, home, repo.Dir, out.Project.ID, legacy)
	linked := repo.AddWorktree("feature")
	state := func() string {
		t.Helper()
		b, err := os.ReadFile(project.MarkerPath(repo.Dir))
		if err != nil {
			t.Fatal(err)
		}
		return strings.Join(rows(t, home, `SELECT (SELECT seq_counter FROM meta),
    (SELECT group_concat(name || ':' || stage, ' ') FROM projects), (SELECT count(*) FROM worktrees)`)[0], " ") + " | " + string(b)
	}
	before := state()
	for _, c := range []struct {
		dir     string
		args    []string
		pointer string
	}{
		{repo.Dir, nil, "name (registered)"},
		{repo.Dir, []string{"--name", "renamed"}, "name (registered)"},
		{repo.Dir, []string{"--stage", "incomplete_docs"}, "name (registered)"},
		{repo.Dir, []string{"--name", legacy}, "--name"},
		{linked, nil, "name (registered)"},
	} {
		what := strings.Join(append([]string{c.dir}, c.args...), " ")
		t.Chdir(c.dir)
		code, stdout, stderr := execute(t, append([]string{"init", "--json"}, c.args...)...)
		env := errorV1(t, what, wholeJSON(t, stdout), apperr.InvalidInput)
		if code != apperr.ExitInvalidInput || len(env.Error.Details) != 1 || env.Error.Details[0].Pointer != c.pointer ||
			strings.Contains(stdout+stderr, "```") {
			t.Errorf("%s: exit %d\nstdout %s\nstderr %s", what, code, stdout, stderr)
		}
		if c.pointer != "--name" && !strings.Contains(env.Error.Message, "does not rename") {
			t.Errorf("%s: %s", what, env.Error.Message)
		}
	}
	t.Chdir(repo.Dir)
	if code, stdout, stderr := execute(t, "init"); code != apperr.ExitInvalidInput || strings.Contains(stdout+stderr, "```") {
		t.Errorf("text: exit %d\nstdout %q\nstderr %q", code, stdout, stderr)
	}
	if after := state(); after != before {
		t.Fatalf("changed:\n%s\n%s", before, after)
	}
	for _, c := range []struct {
		schema string
		args   []string
	}{
		{"source-list.v1", []string{"source", "list", "--json"}},
		{"target-scope.v1", []string{"target", "show", "--json"}},
	} {
		if code, stdout, stderr := specmetry(t, repo.Dir, c.schema, c.args...); code != apperr.ExitOK {
			t.Errorf("%v: exit %d\n%s%s", c.args, code, stdout, stderr)
		}
	}
}
