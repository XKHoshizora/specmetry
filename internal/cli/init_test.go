package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/contract"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

type initOutput struct {
	Schema  string `json:"schema"`
	Project struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Stage         string `json:"stage"`
		StageDetected string `json:"stage_detected"`
	} `json:"project"`
	Created         bool   `json:"created"`
	InitProgress    string `json:"init_progress"`
	BaselineVersion int    `json:"baseline_version"`
	Target          *struct {
		ID       string  `json:"id"`
		Ref      string  `json:"ref"`
		Snapshot *string `json:"snapshot"`
	} `json:"target"`
	CandidateSources []string `json:"candidate_sources"`
	TestHints        []string `json:"test_hints"`
	Warnings         []string `json:"warnings"`
}

var (
	// schemas validates outputs in the tests independently of the
	// validator the CLI itself uses.
	schemas     = sync.OnceValues(contract.New)
	ulidPattern = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
)

// conforms checks stdout against init.v1 on success and error.v1 on
// failure: init --json always prints one of them.
func conforms(t *testing.T, code int, stdout string) {
	t.Helper()
	validator, err := schemas()
	if err != nil {
		t.Fatal(err)
	}
	schema := "init.v1"
	if code != apperr.ExitOK {
		schema = "error.v1"
	}
	if err := validator.ValidateJSON(schema, []byte(stdout)); err != nil {
		t.Fatalf("stdout does not conform to %s: %v %+v\n%s", schema, err, apperr.From(err).Details, stdout)
	}
}

// initIn runs `init --json` in dir.
func initIn(t *testing.T, dir string, args ...string) (int, initOutput, string) {
	t.Helper()
	t.Chdir(dir)
	code, stdout, stderr := execute(t, append([]string{"init", "--json"}, args...)...)
	conforms(t, code, stdout)
	var out initOutput
	if code == apperr.ExitOK {
		if err := json.Unmarshal([]byte(stdout), &out); err != nil {
			t.Fatal(err)
		}
	}
	return code, out, stdout + stderr
}

func mustInit(t *testing.T, dir string, args ...string) initOutput {
	t.Helper()
	code, out, text := initIn(t, dir, args...)
	if code != apperr.ExitOK {
		t.Fatalf("init %v in %s: exit %d\n%s", args, dir, code, text)
	}
	return out
}

// rows queries the database; NULL is returned as "NULL".
func rows(t *testing.T, home datadir.Dir, query string, args ...any) [][]string {
	t.Helper()
	ctx := context.Background()
	db, _, err := sqlite.Open(ctx, home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var out [][]string
	err = db.WriteTx(ctx, func(tx *sql.Tx) error {
		rs, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer func() { _ = rs.Close() }()
		cols, _ := rs.Columns()
		for rs.Next() {
			vals := make([]sql.NullString, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rs.Scan(ptrs...); err != nil {
				return err
			}
			row := make([]string, len(cols))
			for i, v := range vals {
				row[i] = "NULL"
				if v.Valid {
					row[i] = v.String
				}
			}
			out = append(out, row)
		}
		return rs.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func count(t *testing.T, home datadir.Dir, query string) int {
	t.Helper()
	r := rows(t, home, query)
	if len(r) != 1 || len(r[0]) != 1 {
		t.Fatalf("%s returned %v", query, r)
	}
	var n int
	if err := json.Unmarshal([]byte(r[0][0]), &n); err != nil {
		t.Fatal(err)
	}
	return n
}

func readMarker(t *testing.T, root string) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, ".specmetry", "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func committedRepo(t *testing.T) *testutil.GitRepo {
	t.Helper()
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("README.md", "# demo\n")
	repo.Commit("init")
	return repo
}

// MVP-INIT-01: init registers the project, writes the marker and returns
// a new ULID.
func TestInitRegistersProject(t *testing.T) {
	home := testutil.TempHome(t)
	repo := committedRepo(t)
	out := mustInit(t, repo.Dir, "--name", "demo")
	if !out.Created || !ulidPattern.MatchString(out.Project.ID) || out.Project.Name != "demo" {
		t.Fatalf("init = %+v", out)
	}
	if out.Schema != "specmetry.init/v1" || out.InitProgress != "initialized" || out.BaselineVersion != 0 {
		t.Fatalf("init = %+v", out)
	}
	m := readMarker(t, repo.Dir)
	if m["schema"] != "specmetry.project/v1" || m["project_id"] != out.Project.ID || m["name"] != "demo" {
		t.Fatalf("marker = %v", m)
	}
	got := rows(t, home, `SELECT id, name, git_common_dir, stage FROM projects`)
	want := [][]string{{out.Project.ID, "demo", filepath.Join(repo.Dir, ".git"), out.Project.Stage}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("projects = %v, want %v", got, want)
	}
	wts := rows(t, home, `SELECT w.path, w.git_dir, w.is_main, w.id = p.main_worktree_id FROM worktrees w JOIN projects p ON p.id = w.project_id`)
	if want := [][]string{{repo.Dir, filepath.Join(repo.Dir, ".git"), "1", "1"}}; !reflect.DeepEqual(wts, want) {
		t.Fatalf("worktrees = %v, want %v", wts, want)
	}
}

// MVP-INIT-02: init again changes nothing and keeps one project.
func TestInitIsIdempotent(t *testing.T) {
	home := testutil.TempHome(t)
	repo := committedRepo(t)
	first := mustInit(t, repo.Dir, "--name", "demo")
	marker, err := os.ReadFile(filepath.Join(repo.Dir, ".specmetry", "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	seq := rows(t, home, `SELECT seq_counter FROM meta`)
	second := mustInit(t, repo.Dir)
	if second.Created || second.Project.ID != first.Project.ID || second.Project.Name != "demo" {
		t.Fatalf("second init = %+v, first %+v", second, first)
	}
	if second.Target == nil || first.Target == nil || second.Target.ID != first.Target.ID {
		t.Fatalf("target changed: %+v -> %+v", first.Target, second.Target)
	}
	if again := mustInit(t, filepath.Join(repo.Dir, ".")); again.Project.ID != first.Project.ID {
		t.Fatal("init found another project")
	}
	for table, want := range map[string]int{"projects": 1, "worktrees": 1, "delivery_targets": 1} {
		if n := count(t, home, `SELECT count(*) FROM `+table); n != want {
			t.Errorf("%s has %d rows, want %d", table, n, want)
		}
	}
	if after := rows(t, home, `SELECT seq_counter FROM meta`); !reflect.DeepEqual(after, seq) {
		t.Errorf("seq moved from %v to %v on a re-run that registers nothing", seq, after)
	}
	if b, _ := os.ReadFile(filepath.Join(repo.Dir, ".specmetry", "project.json")); string(b) != string(marker) {
		t.Errorf("marker rewritten: %q -> %q", marker, b)
	}
}

// describeRun tells an init.v1 result from an error.v1 envelope and keeps
// stdout and stderr apart, for diagnosing a failed concurrent run.
func describeRun(stdout, stderr string) string {
	var head struct {
		Schema string      `json:"schema"`
		Error  apperr.Body `json:"error"`
	}
	kind := "unparsed stdout"
	if json.Unmarshal([]byte(stdout), &head) == nil {
		switch head.Schema {
		case "specmetry.init/v1":
			kind = "init.v1 result"
		case "specmetry.error/v1":
			kind = fmt.Sprintf("error.v1: code %s, message %q, details %+v", head.Error.Code, head.Error.Message, head.Error.Details)
		}
	}
	return fmt.Sprintf("  %s\n  stdout: %q\n  stderr: %q", kind, stdout, stderr)
}

// Concurrent first runs register one project.
func TestInitConcurrentRunsRegisterOneProject(t *testing.T) {
	home := testutil.TempHome(t)
	repo := committedRepo(t)
	t.Chdir(repo.Dir)
	const n = 4
	outs := make([]initOutput, n)
	codes := make([]int, n)
	stdouts, stderrs := make([]string, n), make([]string, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			var stdout, stderr strings.Builder
			codes[i] = Execute(context.Background(), []string{"init", "--json"}, &stdout, &stderr)
			stdouts[i], stderrs[i] = stdout.String(), stderr.String()
			_ = json.Unmarshal([]byte(stdouts[i]), &outs[i])
		})
	}
	wg.Wait()
	created := 0
	for i := range n {
		if codes[i] != apperr.ExitOK || outs[i].Project.ID != outs[0].Project.ID {
			t.Errorf("run %d: exit %d\n%s", i, codes[i], describeRun(stdouts[i], stderrs[i]))
		}
	}
	if t.Failed() {
		for i := range n {
			t.Logf("run %d: exit %d\n%s", i, codes[i], describeRun(stdouts[i], stderrs[i]))
		}
		t.FailNow()
	}
	for i := range n {
		if outs[i].Created {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("%d runs created the project", created)
	}
	if got := count(t, home, `SELECT count(*) FROM projects`); got != 1 {
		t.Fatalf("%d projects", got)
	}
}

// MVP-INIT-03: the stage of each fixture, the --stage override, and no file
// content read (canaries in files never reach the data directory or the
// output).
func TestInitDetectsStage(t *testing.T) {
	for _, c := range []struct{ fixture, stage string }{
		{"fx-docs-only", "docs_only"},
		{"fx-in-dev", "in_development"},
		{"fx-incomplete", "incomplete_docs"},
	} {
		t.Run(c.fixture, func(t *testing.T) {
			home := testutil.TempHome(t)
			f := testutil.BuildFixture(t, c.fixture)
			if f.Manifest.Stage != c.stage {
				t.Fatalf("fixture.json stage = %s", f.Manifest.Stage)
			}
			canary := testutil.NewCanary(t)
			readme := filepath.Join(f.Repo.Dir, "README.md")
			b, err := os.ReadFile(readme)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(readme, append(b, []byte("\n"+canary+"\n")...), 0o644); err != nil {
				t.Fatal(err)
			}
			f.Repo.WriteFile("notes/extra.md", canary+"\n")
			out := mustInit(t, f.Repo.Dir)
			if out.Project.StageDetected != c.stage || out.Project.Stage != c.stage {
				t.Fatalf("stage = %s, detected %s; want %s", out.Project.Stage, out.Project.StageDetected, c.stage)
			}
			code, stdout, stderr := execute(t, "init")
			if code != apperr.ExitOK {
				t.Fatalf("init without --json: exit %d\n%s%s", code, stdout, stderr)
			}
			if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
				t.Fatalf("file content reached the data directory: %v", hits)
			}
			text, _ := json.Marshal(out)
			if strings.Contains(string(text)+stdout+stderr, canary) {
				t.Fatal("file content reached the output")
			}
		})
	}
}

func TestInitStageOverride(t *testing.T) {
	home := testutil.TempHome(t)
	f := testutil.BuildFixture(t, "fx-in-dev")
	out := mustInit(t, f.Repo.Dir, "--stage", "docs_only")
	if out.Project.Stage != "docs_only" || out.Project.StageDetected != "in_development" {
		t.Fatalf("project = %+v", out.Project)
	}
	// The chosen stage is kept on later runs; detection is reported afresh.
	again := mustInit(t, f.Repo.Dir)
	if again.Project.Stage != "docs_only" || again.Project.StageDetected != "in_development" {
		t.Fatalf("re-run project = %+v", again.Project)
	}
	changed := mustInit(t, f.Repo.Dir, "--stage", "incomplete_docs")
	if changed.Project.Stage != "incomplete_docs" {
		t.Fatalf("project = %+v", changed.Project)
	}
	if got := rows(t, home, `SELECT stage FROM projects`); !reflect.DeepEqual(got, [][]string{{"incomplete_docs"}}) {
		t.Fatalf("stored stage = %v", got)
	}
	if code, _, _ := initIn(t, f.Repo.Dir, "--stage", "done"); code != apperr.ExitInvalidInput {
		t.Fatalf("--stage done: exit %d", code)
	}
}

// MVP-INIT-04: candidate sources of fx-docs-only, never registered.
func TestInitCandidateSources(t *testing.T) {
	testutil.TempHome(t)
	f := testutil.BuildFixture(t, "fx-docs-only")
	out := mustInit(t, f.Repo.Dir)
	if want := []string{"docs/client-brief.md", "README.md"}; !reflect.DeepEqual(out.CandidateSources, want) {
		t.Fatalf("candidate_sources = %q, want %q", out.CandidateSources, want)
	}
	if out.InitProgress != "initialized" {
		t.Fatalf("init_progress = %s: candidates must not count as sources", out.InitProgress)
	}
}

// MVP-INIT-11 (M1): the default target is the main worktree's branch, even
// when init first runs in a linked worktree on another branch.
func TestInitDefaultTarget(t *testing.T) {
	t.Run("main worktree on main", func(t *testing.T) {
		home := testutil.TempHome(t)
		f := testutil.BuildFixture(t, "fx-in-dev")
		out := mustInit(t, f.Repo.Dir)
		if out.Target == nil || out.Target.Ref != "refs/heads/main" || out.Target.Snapshot == nil || *out.Target.Snapshot != f.Manifest.ExpectedRefs["main"] {
			t.Fatalf("target = %+v", out.Target)
		}
		if len(out.Warnings) != 0 {
			t.Fatalf("warnings = %q", out.Warnings)
		}
		got := rows(t, home, `SELECT id, ref, archived_seq, actor, channel FROM delivery_targets`)
		if want := [][]string{{out.Target.ID, "refs/heads/main", "NULL", "system", "system"}}; !reflect.DeepEqual(got, want) {
			t.Fatalf("delivery_targets = %v, want %v", got, want)
		}
	})
	t.Run("first run in a linked worktree", func(t *testing.T) {
		testutil.TempHome(t)
		f := testutil.BuildFixture(t, "fx-in-dev")
		wt := f.Repo.WorktreeFor("feature/x")
		out := mustInit(t, wt)
		if out.Target == nil || out.Target.Ref != "refs/heads/main" || *out.Target.Snapshot != f.Manifest.ExpectedRefs["main"] {
			t.Fatalf("target = %+v", out.Target)
		}
		if out.Project.StageDetected != "in_development" {
			t.Fatalf("stage detected in %s", out.Project.StageDetected)
		}
		if _, err := os.Stat(filepath.Join(wt, ".specmetry")); !os.IsNotExist(err) {
			t.Fatalf("marker written in the linked worktree (%v)", err)
		}
		if readMarker(t, f.Repo.Dir)["project_id"] != out.Project.ID {
			t.Fatal("no marker in the main worktree")
		}
	})
	t.Run("detached main worktree", func(t *testing.T) {
		home := testutil.TempHome(t)
		f := testutil.BuildFixture(t, "fx-in-dev")
		f.Repo.Git("checkout", "-q", "--detach")
		out := mustInit(t, f.Repo.Dir)
		if out.Target != nil || !slices.Equal(out.Warnings, []string{"detached_head_no_target"}) {
			t.Fatalf("target = %+v, warnings %q", out.Target, out.Warnings)
		}
		if n := count(t, home, `SELECT count(*) FROM delivery_targets`); n != 0 {
			t.Fatalf("%d delivery targets", n)
		}
		// Only the human sets a target later (target set); init does not
		// add one once the project exists.
		f.Repo.Git("checkout", "-q", "main")
		again := mustInit(t, f.Repo.Dir)
		if again.Target != nil || len(again.Warnings) != 0 {
			t.Fatalf("re-run target = %+v, warnings %q", again.Target, again.Warnings)
		}
	})
	t.Run("branch without commits", func(t *testing.T) {
		testutil.TempHome(t)
		repo := testutil.NewGitRepo(t)
		out := mustInit(t, repo.Dir)
		if out.Target == nil || out.Target.Ref != "refs/heads/main" || out.Target.Snapshot != nil {
			t.Fatalf("target = %+v", out.Target)
		}
	})
}

// MVP-WT-01 (M1): a linked worktree shares the project and is registered
// on its own.
func TestInitRegistersLinkedWorktree(t *testing.T) {
	home := testutil.TempHome(t)
	f := testutil.BuildFixture(t, "fx-in-dev")
	main := mustInit(t, f.Repo.Dir)
	wt := f.Repo.WorktreeFor("feature/x")
	linked := mustInit(t, wt)
	if linked.Created || linked.Project.ID != main.Project.ID {
		t.Fatalf("linked init = %+v, main %+v", linked, main)
	}
	got := rows(t, home, `SELECT path, git_dir, is_main FROM worktrees WHERE project_id = ? ORDER BY is_main DESC`, main.Project.ID)
	want := [][]string{
		{f.Repo.Dir, filepath.Join(f.Repo.Dir, ".git"), "1"},
		{wt, filepath.Join(f.Repo.Dir, ".git", "worktrees", filepath.Base(wt)), "0"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("worktrees = %v, want %v", got, want)
	}
	if n := count(t, home, `SELECT count(DISTINCT id) FROM worktrees`); n != 2 {
		t.Fatalf("%d distinct worktree ids", n)
	}
	if again := mustInit(t, wt); again.Created || again.Project.ID != main.Project.ID {
		t.Fatalf("second linked init = %+v", again)
	}
	if n := count(t, home, `SELECT count(*) FROM worktrees`); n != 2 {
		t.Fatalf("%d worktrees after a re-run", n)
	}
}

// The project is found from any spelling of its path.
func TestInitFindsProjectFromAnotherPath(t *testing.T) {
	home := testutil.TempHome(t)
	repo := committedRepo(t)
	first := mustInit(t, repo.Dir)
	var others []string
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(repo.Dir, link); err == nil {
		others = append(others, link)
	} else if runtime.GOOS != "windows" {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		others = append(others, strings.ToLower(repo.Dir), strings.ToUpper(repo.Dir))
	}
	for _, dir := range others {
		if out := mustInit(t, dir); out.Created || out.Project.ID != first.Project.ID {
			t.Errorf("init from %s = %+v", dir, out)
		}
	}
	if n := count(t, home, `SELECT count(*) FROM worktrees`); n != 1 {
		t.Fatalf("%d worktrees: %v", n, rows(t, home, `SELECT path FROM worktrees`))
	}
}

// FR-INIT-09 (write side): a directory outside git is a project without a
// target, found again from its sub-directories through the marker.
func TestInitOutsideGit(t *testing.T) {
	home := testutil.TempHome(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "brief.md"), []byte(strings.Repeat("x", 600)), 0o644); err != nil {
		t.Fatal(err)
	}
	out := mustInit(t, root, "--name", "plain")
	if out.Target != nil || !slices.Equal(out.Warnings, []string{"not_a_git_repository"}) || out.Project.StageDetected != "docs_only" {
		t.Fatalf("init = %+v", out)
	}
	if !reflect.DeepEqual(out.CandidateSources, []string{"docs/brief.md"}) {
		t.Fatalf("candidate_sources = %q", out.CandidateSources)
	}
	if got := rows(t, home, `SELECT p.git_common_dir, w.path, w.git_dir FROM projects p JOIN worktrees w ON w.project_id = p.id`); !reflect.DeepEqual(got, [][]string{{"NULL", root, "NULL"}}) {
		t.Fatalf("registry = %v", got)
	}
	again := mustInit(t, filepath.Join(root, "docs", "deep"))
	if again.Created || again.Project.ID != out.Project.ID {
		t.Fatalf("init from a sub-directory = %+v", again)
	}
	if n := count(t, home, `SELECT count(*) FROM worktrees`); n != 1 {
		t.Fatalf("%d worktrees", n)
	}
}

func fileEntries(snapshot map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range snapshot {
		if !strings.HasPrefix(v, "d") {
			out[k] = v
		}
	}
	return out
}

// init reads the repository and adds only the marker and the exclude
// entry: no commit, ref, index, configuration or other file changes.
func TestInitLeavesTheRepositoryAlone(t *testing.T) {
	testutil.TempHome(t)
	f := testutil.BuildFixture(t, "fx-in-dev")
	wt := f.Repo.WorktreeFor("feature/x")
	before := fileEntries(testutil.SettledSnapshot(t, filepath.Dir(f.Repo.Dir)))
	mustInit(t, f.Repo.Dir)
	mustInit(t, wt)
	after := fileEntries(testutil.SnapshotTree(t, filepath.Dir(f.Repo.Dir)))
	added := map[string]bool{}
	for k := range after {
		if _, ok := before[k]; !ok {
			added[k] = true
		}
	}
	want := map[string]bool{"repo/.specmetry/project.json": true, "repo/.git/info/exclude": true}
	if !reflect.DeepEqual(added, want) {
		t.Errorf("files added: %v, want %v", added, want)
	}
	for k, v := range before {
		if after[k] != v {
			t.Errorf("%s changed or removed", k)
		}
	}
	if status := f.Repo.Git("status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Errorf("the worktree is not clean after init:\n%s", status)
	}
	if refs := f.Refs(); !reflect.DeepEqual(refs, f.Manifest.ExpectedRefs) {
		t.Errorf("refs = %v", refs)
	}
}

// MVP-INIT-08 [P1] and PD-07: the exclude entry is written once, keeps the
// user's entries, can be turned off, and an unwritable file only warns.
func TestInitExclude(t *testing.T) {
	t.Run("written once, keeping existing entries", func(t *testing.T) {
		testutil.TempHome(t)
		repo := committedRepo(t)
		exclude := filepath.Join(repo.Dir, ".git", "info", "exclude")
		if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(exclude, []byte("# mine\n*.log"), 0o644); err != nil {
			t.Fatal(err)
		}
		mustInit(t, repo.Dir)
		mustInit(t, repo.Dir)
		if b, _ := os.ReadFile(exclude); string(b) != "# mine\n*.log\n/.specmetry/\n" {
			t.Fatalf("exclude = %q", b)
		}
	})
	t.Run("--no-exclude", func(t *testing.T) {
		testutil.TempHome(t)
		repo := committedRepo(t)
		out := mustInit(t, repo.Dir, "--no-exclude")
		if len(out.Warnings) != 0 {
			t.Fatalf("warnings = %q", out.Warnings)
		}
		if _, err := os.Stat(filepath.Join(repo.Dir, ".git", "info", "exclude")); !os.IsNotExist(err) {
			t.Fatalf("exclude written (%v)", err)
		}
	})
	t.Run("read-only exclude file", func(t *testing.T) {
		if runtime.GOOS != "windows" && os.Geteuid() == 0 {
			t.Skip("root can write read-only files")
		}
		testutil.TempHome(t)
		repo := committedRepo(t)
		exclude := filepath.Join(repo.Dir, ".git", "info", "exclude")
		if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(exclude, []byte("*.log\n"), 0o444); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(exclude, 0o644) })
		out := mustInit(t, repo.Dir)
		if !slices.Equal(out.Warnings, []string{"exclude_not_writable"}) {
			t.Fatalf("warnings = %q", out.Warnings)
		}
		if b, _ := os.ReadFile(exclude); string(b) != "*.log\n" {
			t.Fatalf("exclude = %q", b)
		}
		if readMarker(t, repo.Dir)["project_id"] != out.Project.ID {
			t.Fatal("marker not written")
		}
	})
}

// MVP-INIT-10 [P1].
func TestInitTestHints(t *testing.T) {
	testutil.TempHome(t)
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("go.mod", "module x\n")
	repo.WriteFile("package.json", "{}\n")
	repo.WriteFile("pytest.ini", "[pytest]\n")
	repo.Commit("init")
	out := mustInit(t, repo.Dir)
	if want := []string{"go test ./...", "npm test", "pytest"}; !reflect.DeepEqual(out.TestHints, want) {
		t.Fatalf("test_hints = %q, want %q", out.TestHints, want)
	}
}

// --skip-sources and --skip-inventory are recorded and count towards the
// progress cumulatively (data-model §6.12).
func TestInitSkipFlags(t *testing.T) {
	testutil.TempHome(t)
	repo := committedRepo(t)
	if out := mustInit(t, repo.Dir, "--skip-inventory"); out.InitProgress != "initialized" {
		t.Fatalf("--skip-inventory alone: %s", out.InitProgress)
	}
	if out := mustInit(t, repo.Dir, "--skip-sources"); out.InitProgress != "sources_registered" {
		t.Fatalf("--skip-sources: %s", out.InitProgress)
	}
	if out := mustInit(t, repo.Dir); out.InitProgress != "sources_registered" {
		t.Fatalf("re-run: %s", out.InitProgress)
	}
}

// The default name is the main worktree's directory name; init never
// renames a project.
func TestInitName(t *testing.T) {
	testutil.TempHome(t)
	repo := committedRepo(t)
	out := mustInit(t, repo.Dir)
	if out.Project.Name != filepath.Base(repo.Dir) {
		t.Fatalf("name = %q", out.Project.Name)
	}
	if again := mustInit(t, repo.Dir, "--name", filepath.Base(repo.Dir)); again.Project.Name != out.Project.Name {
		t.Fatalf("name = %q", again.Project.Name)
	}
	code, _, text := initIn(t, repo.Dir, "--name", "other")
	if code != apperr.ExitInvalidInput || !strings.Contains(text, "does not rename") {
		t.Fatalf("rename: exit %d\n%s", code, text)
	}
}

// Invalid input is refused before anything is written: the data
// directory is not even created.
func TestInitRefusesInvalidInput(t *testing.T) {
	secret := "ghp_" + strings.Repeat("A1b2", 9)
	for name, args := range map[string][]string{
		"empty name":         {"--name", ""},
		"blank name":         {"--name", "  "},
		"multi-line name":    {"--name", "a\nb"},
		"control character":  {"--name", "a\x07b"},
		"name too long":      {"--name", strings.Repeat("n", 201)},
		"secret in the name": {"--name", "demo " + secret},
		"unknown stage":      {"--stage", "done"},
	} {
		t.Run(name, func(t *testing.T) {
			home := testutil.TempHome(t)
			repo := committedRepo(t)
			code, _, text := initIn(t, repo.Dir, args...)
			if code != apperr.ExitInvalidInput {
				t.Fatalf("exit %d\n%s", code, text)
			}
			if strings.Contains(text, secret) {
				t.Fatal("the secret was echoed")
			}
			if _, err := os.Stat(home.Path); !os.IsNotExist(err) {
				t.Fatalf("data directory created (%v)", err)
			}
			if _, err := os.Stat(filepath.Join(repo.Dir, ".specmetry")); !os.IsNotExist(err) {
				t.Fatalf("marker written (%v)", err)
			}
		})
	}
}

// A path Specmetry would store is scanned for secrets like the name.
func TestInitRefusesSecretInPath(t *testing.T) {
	home := testutil.TempHome(t)
	secret := "ghp_" + strings.Repeat("A1b2", 9)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, secret)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	code, _, text := initIn(t, dir, "--name", "demo")
	if code != apperr.ExitInvalidInput || strings.Contains(text, secret) {
		t.Fatalf("exit %d\n%s", code, text)
	}
	if _, err := os.Stat(home.Path); !os.IsNotExist(err) {
		t.Fatalf("data directory created (%v)", err)
	}
}

// Inside a git directory or a bare repository there is no worktree to
// register.
func TestInitRefusesGitDirectories(t *testing.T) {
	home := testutil.TempHome(t)
	repo := committedRepo(t)
	bare := filepath.Join(t.TempDir(), "bare.git")
	repo.Git("clone", "-q", "--bare", repo.Dir, bare)
	for _, dir := range []string{filepath.Join(repo.Dir, ".git"), bare} {
		code, _, text := initIn(t, dir)
		if code != apperr.ExitEnvironment {
			t.Errorf("init in %s: exit %d\n%s", dir, code, text)
		}
	}
	if _, err := os.Stat(home.Path); !os.IsNotExist(err) {
		t.Fatalf("data directory created (%v)", err)
	}
}

// TD-25: a marker that disagrees with the database is reported, and
// nothing is written.
func TestInitRefusesConflictingMarker(t *testing.T) {
	const other = "01J9Z3K5Q8R2T4V6W8X0Y2Z4C9"
	writeMarker := func(t *testing.T, root, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(root, ".specmetry"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".specmetry", "project.json"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("marker of a project this database does not have", func(t *testing.T) {
		home := testutil.TempHome(t)
		repo := committedRepo(t)
		writeMarker(t, repo.Dir, `{"schema":"specmetry.project/v1","project_id":"`+other+`","name":"elsewhere"}`)
		code, _, text := initIn(t, repo.Dir)
		if code != apperr.ExitEnvironment || !strings.Contains(text, other) {
			t.Fatalf("exit %d\n%s", code, text)
		}
		// codex-review-015 N1: checked before anything opens the database.
		if _, err := os.Stat(home.Path); !os.IsNotExist(err) {
			t.Fatalf("data directory created (%v)", err)
		}
	})
	t.Run("marker replaced after registration", func(t *testing.T) {
		home := testutil.TempHome(t)
		repo := committedRepo(t)
		mustInit(t, repo.Dir)
		seq := rows(t, home, `SELECT seq_counter FROM meta`)
		writeMarker(t, repo.Dir, `{"schema":"specmetry.project/v1","project_id":"`+other+`","name":"elsewhere"}`)
		wt := repo.AddWorktree("feature-x")
		if code, _, text := initIn(t, wt); code != apperr.ExitEnvironment {
			t.Fatalf("exit %d\n%s", code, text)
		}
		if after := rows(t, home, `SELECT seq_counter FROM meta`); !reflect.DeepEqual(after, seq) {
			t.Fatalf("written: seq %v -> %v", seq, after)
		}
	})
	t.Run("unreadable marker", func(t *testing.T) {
		testutil.TempHome(t)
		repo := committedRepo(t)
		writeMarker(t, repo.Dir, `{`)
		if code, _, text := initIn(t, repo.Dir); code != apperr.ExitEnvironment {
			t.Fatalf("exit %d\n%s", code, text)
		}
	})
}

// A project removed by purge stays removed: its marker does not bring it
// back under the same ID.
func TestInitDoesNotReviveTombstonedProject(t *testing.T) {
	home := testutil.TempHome(t)
	repo := committedRepo(t)
	out := mustInit(t, repo.Dir)
	b, err := json.Marshal([]sqlite.Tombstone{{ProjectID: out.Project.ID, PurgedAt: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(home.Join(datadir.TombstonesFile), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, text := initIn(t, repo.Dir); code != apperr.ExitEnvironment {
		t.Fatalf("exit %d\n%s", code, text)
	}
	if n := count(t, home, `SELECT count(*) FROM projects`); n != 0 {
		t.Fatalf("%d projects", n)
	}
}

// Without --json, init prints a summary for a person.
func TestInitHumanOutput(t *testing.T) {
	testutil.TempHome(t)
	f := testutil.BuildFixture(t, "fx-in-dev")
	t.Chdir(f.Repo.Dir)
	code, stdout, stderr := execute(t, "init")
	if code != apperr.ExitOK || stderr != "" {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	for _, want := range []string{"Initialized project", "stage: in_development", "refs/heads/main", "docs/client-brief.md", "go test ./..."} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output lacks %q:\n%s", want, stdout)
		}
	}
	_, stdout, _ = execute(t, "init")
	if !strings.Contains(stdout, "already initialized") {
		t.Errorf("re-run output:\n%s", stdout)
	}
}
