package checkrun_test

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/checkrun"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

func TestMain(m *testing.M) {
	testutil.DispatchFakeGit()
	os.Exit(m.Run())
}

const (
	wsProject = "01JC0000000000000000000P01"
	wsMain    = "01JC0000000000000000000W01"
)

// workspace is a registered git project whose main worktree is the test's
// current directory, at baseline version 1 and seq 2, with no criteria.
type workspace struct {
	home  datadir.Dir
	repo  *testutil.GitRepo
	where project.Where
	reqs  map[string]bool
	n     int
}

func newWorkspace(t *testing.T) *workspace {
	t.Helper()
	home := testutil.TempHome(t)
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("README.md", "readme")
	repo.Commit("init")
	where, err := project.Locate(ctx, repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	w := &workspace{home: home, repo: repo, where: where, reqs: map[string]bool{}}
	w.exec(t, `
INSERT INTO projects (id, name, git_common_dir, main_worktree_id, stage, settings_json, created_at)
VALUES (?1, 'p', ?3, ?2, 'in_development', '{}', 1);
INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at) VALUES (?2, ?1, ?4, ?5, 1, 1, 1);
INSERT INTO baseline_versions (id, seq, project_id, number, summary, actor, channel)
VALUES ('01JC0000000000000000000V01', 2, ?1, 1, 'v1', 'human:me', 'human_cli');
UPDATE meta SET seq_counter = 2 WHERE id = 1;`, wsProject, wsMain, where.CommonDir, where.Current, where.GitDir)
	t.Chdir(repo.Dir)
	return w
}

func (w *workspace) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	db, _, err := sqlite.Open(ctx, w.home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeDB(t, db)
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

// read runs fn in a transaction of the workspace's database.
func (w *workspace) read(t *testing.T, fn func(*sql.Tx) error) {
	t.Helper()
	db, _, err := sqlite.Open(ctx, w.home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeDB(t, db)
	if err := db.WriteTx(ctx, fn); err != nil {
		t.Fatal(err)
	}
}

// ac is a criterion to add: active and automated, at revision 1 of itself
// and its requirement, unless it says otherwise; hash, when set, is stored
// instead of the check's own.
type ac struct {
	key, status, verification string
	revision, reqRevision     int
	check                     *baseline.Check
	hash                      string
}

func sh(command string) *baseline.Check {
	return &baseline.Check{Command: command, Workdir: ".", TimeoutSec: 300, Rationale: "it tests the criterion"}
}

func shell(name, command string) *baseline.Check {
	c := sh(command)
	c.Shell = &name
	return c
}

func (w *workspace) criteria(t *testing.T, acs ...ac) {
	t.Helper()
	for _, a := range acs {
		parts := strings.Split(a.key, "-")
		req := "REQ-" + parts[1]
		reqID := "01JC000000000000000000R" + parts[1]
		if !w.reqs[req] {
			w.reqs[req] = true
			w.exec(t, `INSERT INTO requirements (id, project_id, key, status, revision, revision_seq, origin, created_version, updated_version,
    updated_seq, title, kind, priority) VALUES (?, ?, ?, 'active', 2, 2, 'extracted', 1, 1, 2, 'orders', 'functional', 'must')`,
				reqID, wsProject, req)
		}
		w.n++
		status, verification, revision, reqRevision := a.status, a.verification, a.revision, a.reqRevision
		if status == "" {
			status = "active"
		}
		if verification == "" {
			verification = "automated"
		}
		if revision == 0 {
			revision = 1
		}
		if reqRevision == 0 {
			reqRevision = 1
		}
		var retired, command, workdir, shellName, inputs, rationale, hash any = nil, nil, ".", nil, nil, nil, nil
		timeout := 300
		if status == "retired" {
			retired = 1
		}
		if c := a.check; c != nil {
			command, workdir, rationale, timeout, hash = c.Command, c.Workdir, c.Rationale, c.TimeoutSec, baseline.CheckConfigHash(*c)
			if c.Shell != nil {
				shellName = *c.Shell
			}
			if c.Inputs != nil {
				inputs = `["` + strings.Join(c.Inputs, `","`) + `"]`
			}
			if a.hash != "" {
				hash = a.hash
			}
		}
		w.exec(t, `INSERT INTO acceptance_criteria (id, project_id, key, status, revision, revision_seq, origin, created_version,
    updated_version, retired_version, updated_seq, requirement_id, requirement_revision, statement, verification, check_command,
    check_workdir, check_shell, check_inputs_json, check_rationale, check_timeout_sec, check_config_hash)
VALUES (?, ?, ?, ?, ?, 2, 'extracted', 1, 1, ?, 2, ?, ?, 'orders are isolated', ?, ?, ?, ?, ?, ?, ?, ?)`,
			fmt.Sprintf("01JC000000000000000000C%03d", w.n), wsProject, a.key, status, revision, retired, reqID, reqRevision,
			verification, command, workdir, shellName, inputs, rationale, timeout, hash)
	}
}

// state is every row of every table and the seq counter, to show that a
// refusal wrote nothing: no fact, no worktree, no seq, no last_seen_at.
func (w *workspace) state(t *testing.T) string {
	t.Helper()
	db, _, err := sqlite.Open(ctx, w.home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeDB(t, db)
	var out []string
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		var tables []string
		rs, err := tx.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name`)
		if err != nil {
			return err
		}
		for rs.Next() {
			var name string
			if err := rs.Scan(&name); err != nil {
				return err
			}
			tables = append(tables, name)
		}
		if err := rs.Close(); err != nil {
			return err
		}
		for _, table := range tables {
			rs, err := tx.QueryContext(ctx, `SELECT * FROM "`+table+`"`)
			if err != nil {
				return err
			}
			cols, _ := rs.Columns()
			var rows []string
			for rs.Next() {
				vals := make([]any, len(cols))
				ptrs := make([]any, len(cols))
				for i := range vals {
					ptrs[i] = &vals[i]
				}
				if err := rs.Scan(ptrs...); err != nil {
					return err
				}
				rows = append(rows, fmt.Sprintf("%v", vals))
			}
			if err := rs.Close(); err != nil {
				return err
			}
			slices.Sort(rows)
			out = append(out, table+": "+strings.Join(rows, "; "))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return strings.Join(out, "\n")
}

var terminal = checkrun.Output{Stdout: os.Stdout, Stderr: os.Stderr}

// run prepares opts with h and closes the batch at the test's end.
func run(t *testing.T, h *checkrun.Host, opts checkrun.Options) (*checkrun.Batch, error) {
	t.Helper()
	b, err := checkrun.PrepareWith(ctx, h, opts, terminal)
	if b != nil {
		t.Cleanup(func() { closeBatch(t, b) })
	}
	return b, err
}

func all() checkrun.Options { return checkrun.Options{All: true, Isolation: "off"} }

func given(keys ...string) checkrun.Options {
	return checkrun.Options{Criteria: keys, Isolation: "off"}
}

// refused checks a refusal: its exit code, its details' pointers in order,
// a message that says no check was run, an error.v1 document, and a host
// left with no call admitted. It returns the document.
func refused(t *testing.T, h *checkrun.Host, err error, exit int, pointers ...string) string {
	t.Helper()
	if got := apperr.ExitCode(err); got != exit {
		t.Fatalf("exit %d (%v), want %d", got, err, exit)
	}
	var got []string
	for _, d := range apperr.From(err).Details {
		got = append(got, d.Pointer)
	}
	if !slices.Equal(got, pointers) {
		t.Fatalf("details at %v, want %v (%v)", got, pointers, apperr.From(err).Details)
	}
	if _, active, _ := checkrun.State(h); active != 0 {
		t.Fatalf("%d calls still admitted after a refusal", active)
	}
	if !strings.Contains(apperr.From(err).Message, "no check was run") {
		t.Fatalf("the message does not say that no check was run: %q", apperr.From(err).Message)
	}
	return envelope(t, err)
}

func keys(b *checkrun.Batch) []string {
	var out []string
	for _, it := range b.Items {
		out = append(out, it.Criterion)
	}
	return out
}

// WP2.5 spec §4.1 steps 1 and 2, §19.1 (MVP-EV-13, CX-28(a)): a command
// line error, --isolation required and a disabled host are refused before
// any database or git access: no git process starts (a recording git at
// the process boundary), the database is not opened (the counting seam)
// and the data directory is not touched. A call that is admitted reaches
// both, which shows the seams count.
func TestPrepareRefusesBeforeReachingAnything(t *testing.T) {
	w := newWorkspace(t)
	w.criteria(t, ac{key: "AC-001-1", check: sh("echo one")})
	gitDir, env, calls := testutil.PassthroughGit(t, testutil.PassAsIs)
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		t.Setenv(k, v)
	}
	t.Setenv("PATH", gitDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	opened := 0
	checkrun.OnOpen(t, func() { opened++ })
	files := testutil.SnapshotTree(t, w.home.Path)
	disabled := checkrun.NewHost()
	checkrun.Disable(disabled, checkrun.DisabledO3, "copy")
	cases := []struct {
		name     string
		h        *checkrun.Host
		opts     checkrun.Options
		exit     int
		pointers []string
	}{
		{"required", checkrun.NewHost(), checkrun.Options{All: true, Isolation: "required"}, apperr.ExitEnvironment, []string{"--isolation"}},
		{"disabled host", disabled, all(), apperr.ExitEnvironment, nil},
		{"required on a disabled host", disabled, checkrun.Options{All: true, Isolation: "required"}, apperr.ExitEnvironment, []string{"--isolation"}},
		{"criteria and --all", disabled, checkrun.Options{Criteria: []string{"AC-001-1"}, All: true, Isolation: "off"}, apperr.ExitInvalidInput, nil},
		{"auto", checkrun.NewHost(), checkrun.Options{All: true, Isolation: "auto"}, apperr.ExitInvalidInput, []string{"--isolation"}},
	}
	for _, c := range cases {
		_, err := run(t, c.h, c.opts)
		doc := refused(t, c.h, err, c.exit, c.pointers...)
		if c.name == "required" && !strings.Contains(doc, "isolation_unavailable") {
			t.Errorf("%s: no isolation_unavailable in %s", c.name, doc)
		}
		if n := len(calls()); n != 0 || opened != 0 {
			t.Fatalf("%s: %d git processes, database opened %d times", c.name, n, opened)
		}
	}
	testutil.AssertTreeUnchanged(t, w.home.Path, files)
	if _, err := run(t, checkrun.NewHost(), all()); err != nil {
		t.Fatal(err)
	}
	if len(calls()) == 0 || opened != 1 {
		t.Fatalf("an admitted call: %d git processes, database opened %d times", len(calls()), opened)
	}
}

// WP2.5 spec §3, §4.1 step 5, E1, D13: given criteria run in key order,
// whatever the order given; --all is every active automated criterion
// with a check. Each item holds what the read fixed: the revisions, the
// stored configuration hash (not one computed again) and the check.
func TestPrepareSelects(t *testing.T) {
	w := newWorkspace(t)
	bash := shell("bash", "go test ./orders")
	bash.Inputs, bash.Workdir, bash.TimeoutSec = []string{"b.go", "a.go"}, "orders", 60
	stored := strings.Repeat("ab", 32)
	w.criteria(t,
		ac{key: "AC-010-1", check: sh("echo 10-1")},
		ac{key: "AC-002-10", check: sh("echo 2-10")},
		ac{key: "AC-002-9", check: sh("echo 2-9")},
		ac{key: "AC-001-1", check: sh("echo 1-1")},
		ac{key: "AC-001-2", status: "retired", check: sh("echo 1-2")},
		ac{key: "AC-001-3", verification: "manual"},
		ac{key: "AC-001-4"},
		ac{key: "AC-001-5", revision: 3, reqRevision: 2, check: bash, hash: stored},
	)
	b, err := run(t, checkrun.NewHost(), given("AC-010-1", "AC-002-10", "AC-001-5", "AC-001-1", "AC-002-9"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := keys(b), []string{"AC-001-1", "AC-001-5", "AC-002-9", "AC-002-10", "AC-010-1"}; !slices.Equal(got, want) {
		t.Fatalf("given: %v, want %v", got, want)
	}
	if b.ProjectID != wsProject || b.WorktreeID != wsMain || b.Where != w.where {
		t.Fatalf("batch of %s, worktree %s, %+v", b.ProjectID, b.WorktreeID, b.Where)
	}
	got := b.Items[1]
	if got.Interpreter.Shell != "bash" || got.Interpreter.Path == "" {
		t.Fatalf("AC-001-5's interpreter = %+v", got.Interpreter)
	}
	got.Interpreter = checkrun.Interpreter{} // step 7's, tested on its own
	want := checkrun.Item{Criterion: "AC-001-5", CriterionID: "01JC000000000000000000C008", CriterionRevision: 3, RequirementRevision: 2,
		CheckConfigHash: stored, Check: *bash}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AC-001-5 = %+v\nwant %+v", got, want)
	}
	if b, err = run(t, checkrun.NewHost(), all()); err != nil {
		t.Fatal(err)
	}
	if got, want := keys(b), []string{"AC-001-1", "AC-001-5", "AC-002-9", "AC-002-10", "AC-010-1"}; !slices.Equal(got, want) {
		t.Fatalf("--all: %v, want %v", got, want)
	}
}

// WP2.5 spec §3, D15: every criterion given that cannot be checked is
// refused at its argument's position, none quoted: one not in the
// baseline (or of another type, or misspelled), retired, manual, without
// a check, or repeated (at the later position). Nothing is written.
func TestPrepareRefusesCriteriaGiven(t *testing.T) {
	w := newWorkspace(t)
	w.criteria(t,
		ac{key: "AC-001-1", check: sh("echo one")},
		ac{key: "AC-001-2", status: "retired", check: sh("echo two")},
		ac{key: "AC-001-3", verification: "manual"},
		ac{key: "AC-001-4"},
	)
	const canary = "CANARY-arg-58e1"
	before := w.state(t)
	h := checkrun.NewHost()
	_, err := run(t, h, given("AC-001-1", "AC-099-1", "AC-001-2", "AC-001-3", "AC-001-4", "REQ-001", "AC-001-1", canary, "AC-001-3"))
	doc := refused(t, h, err, apperr.ExitInvalidInput,
		"argument 2", "argument 3", "argument 4", "argument 5", "argument 6", "argument 7", "argument 8", "argument 9")
	want := []string{
		"names no acceptance criterion of the baseline",
		"names a retired acceptance criterion",
		"names a manual acceptance criterion: only automated ones have a check to run",
		"names an acceptance criterion without a check",
		"names no acceptance criterion of the baseline",
		"repeats argument 1",
		"names no acceptance criterion of the baseline",
		"repeats argument 4",
	}
	for i, d := range apperr.From(err).Details {
		if d.Problem != want[i] {
			t.Errorf("%s: %q, want %q", d.Pointer, d.Problem, want[i])
		}
	}
	if strings.Contains(doc, canary) {
		t.Errorf("the refusal quotes an argument: %s", doc)
	}
	if after := w.state(t); after != before {
		t.Errorf("a refusal wrote:\n%s\n->\n%s", before, after)
	}
}

// WP2.5 spec §3, D13 (MVP-EV-26, CX-15, the fixing): what the read
// transaction fixed is what every later step and the batch use. A
// revision made once it has ended changes nothing: neither the item nor
// the static rule's verdict, either way.
func TestPrepareFixesTheConfiguration(t *testing.T) {
	w := newWorkspace(t)
	w.criteria(t, ac{key: "AC-001-1", check: sh("echo one")}, ac{key: "AC-001-2", check: shell("cmd", "echo one\necho two")})
	other := strings.Repeat("cd", 32)
	var revise func()
	checkrun.AfterRead(t, func() { revise() })
	revise = func() {
		w.exec(t, `UPDATE acceptance_criteria SET revision = 2, check_command = ?, check_shell = 'cmd', check_config_hash = ? WHERE key = 'AC-001-1'`,
			"echo one\recho two", other)
	}
	b, err := run(t, checkrun.NewHost(), given("AC-001-1"))
	if err != nil {
		t.Fatalf("refused on a revision made after the read: %v", err)
	}
	if it := b.Items[0]; it.CriterionRevision != 1 || it.Check.Command != "echo one" || it.Check.Shell != nil ||
		it.CheckConfigHash != baseline.CheckConfigHash(*sh("echo one")) {
		t.Fatalf("the item is not what was read: %+v", it)
	}
	revise = func() {
		w.exec(t, `UPDATE acceptance_criteria SET revision = 2, check_command = 'echo one', check_config_hash = ? WHERE key = 'AC-001-2'`, other)
	}
	h := checkrun.NewHost()
	_, err = run(t, h, given("AC-001-2"))
	refused(t, h, err, apperr.ExitInvalidInput, "AC-001-2")
}

// WP2.5 spec §4.1 step 6, D3, D4 (MVP-EV-21, CX-29): a saved check whose
// command breaks the rule refuses the whole batch at its criterion, in key
// order, without quoting the command, and leaves it as it is; a pwsh
// command is not concerned. Once a person revises it, the batch passes.
func TestPrepareStaticRules(t *testing.T) {
	w := newWorkspace(t)
	const canary = "CANARY-cmd-3b90"
	w.criteria(t,
		ac{key: "AC-001-1", check: shell("cmd", "echo one\necho "+canary)},
		ac{key: "AC-001-2", check: sh("make\r")},
		ac{key: "AC-001-3", check: shell("pwsh", "echo one\r\necho two")},
		ac{key: "AC-001-4", check: sh("echo one\necho two")},
	)
	before := w.state(t)
	h := checkrun.NewHost()
	_, err := run(t, h, all())
	if doc := refused(t, h, err, apperr.ExitInvalidInput, "AC-001-1", "AC-001-2"); strings.Contains(doc, canary) {
		t.Errorf("the refusal quotes the command: %s", doc)
	}
	if after := w.state(t); after != before {
		t.Errorf("a refusal wrote:\n%s\n->\n%s", before, after)
	}
	w.exec(t, `UPDATE acceptance_criteria SET revision = 2, check_command = 'echo one', check_shell = NULL, check_config_hash = ?
WHERE key = 'AC-001-1'`, baseline.CheckConfigHash(*sh("echo one")))
	w.exec(t, `UPDATE acceptance_criteria SET revision = 2, check_command = 'make', check_config_hash = ? WHERE key = 'AC-001-2'`,
		baseline.CheckConfigHash(*sh("make")))
	if _, err := run(t, checkrun.NewHost(), given("AC-001-1", "AC-001-2", "AC-001-4")); err != nil {
		t.Fatalf("after the revision: %v", err)
	}
}

// WP2.5 spec §4.1 step 4: W not registered yet has its path and git
// directory scanned for secrets before anything else of the batch is
// judged; a registered W is not scanned again.
func TestPrepareScansAWorktreeToRegister(t *testing.T) {
	w := newWorkspace(t)
	w.criteria(t, ac{key: "AC-001-1", check: sh("echo one")})
	secret := testutil.NewSecretCanary(t)
	linked := filepath.Join(filepath.Dir(w.repo.Dir), "wt-"+secret)
	w.repo.Git("worktree", "add", "-b", "feature", linked)
	t.Chdir(linked)
	h := checkrun.NewHost()
	_, err := run(t, h, given("AC-001-1", "AC-001-1"))
	doc := refused(t, h, err, apperr.ExitInvalidInput, "worktree path", "worktree git directory")
	if strings.Contains(doc, secret) {
		t.Fatalf("the refusal quotes the secret: %s", doc)
	}
	where, err := project.Locate(ctx, linked)
	if err != nil {
		t.Fatal(err)
	}
	w.exec(t, `INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES ('01JC0000000000000000000W02', ?, ?, ?, 0, 2, 1)`, wsProject, where.Current, where.GitDir)
	b, err := run(t, checkrun.NewHost(), given("AC-001-1"))
	if err != nil || b.WorktreeID != "01JC0000000000000000000W02" {
		t.Fatalf("a registered worktree: %v, %+v", err, b)
	}
}

// WP2.5 spec §4.1 step 3: without a database, or with no registered
// project here, the batch is not_initialized (exit 4), before its
// criteria are looked at; no database is created.
func TestPrepareNotInitialized(t *testing.T) {
	home := testutil.TempHome(t)
	repo := testutil.NewGitRepo(t)
	t.Chdir(repo.Dir)
	h := checkrun.NewHost()
	_, err := run(t, h, given("AC-001-1", "AC-001-1"))
	refused(t, h, err, apperr.ExitNotInitialized)
	if _, err := os.Lstat(home.Join(datadir.DatabaseFile)); err == nil {
		t.Fatal("a database was created")
	}
	db, _, err := sqlite.Open(ctx, home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	closeDB(t, db)
	_, err = run(t, h, given("AC-001-1", "AC-001-1"))
	refused(t, h, err, apperr.ExitNotInitialized)
}

// WP2.5 spec §4.1: the steps are judged in their order, whatever the one
// read found; the first that fails decides.
func TestPrepareStepOrder(t *testing.T) {
	w := newWorkspace(t)
	w.criteria(t, ac{key: "AC-001-1", check: shell("cmd", "echo one\necho two")})
	h := checkrun.NewHost()
	// Step 5 before step 6.
	_, err := run(t, h, given("AC-001-1", "AC-001-1"))
	refused(t, h, err, apperr.ExitInvalidInput, "argument 2")
	// Step 4 before step 5.
	linked := filepath.Join(filepath.Dir(w.repo.Dir), "wt-"+testutil.NewSecretCanary(t))
	w.repo.Git("worktree", "add", "-b", "feature", linked)
	t.Chdir(linked)
	_, err = run(t, h, given("AC-001-1", "AC-001-1"))
	refused(t, h, err, apperr.ExitInvalidInput, "worktree path", "worktree git directory")
}
