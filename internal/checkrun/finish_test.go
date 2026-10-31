package checkrun_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/checkrun"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// target makes ref the project's active delivery target.
func (w *workspace) target(t *testing.T, ref string) {
	t.Helper()
	w.exec(t, `INSERT INTO delivery_targets (id, project_id, name, ref, created_seq, actor, channel)
VALUES ('01JC0000000000000000000T01', ?, 'release', ?, 3, 'human:me', 'human_cli'); UPDATE meta SET seq_counter = 3 WHERE id = 1`,
		wsProject, ref)
}

// linked adds a worktree of the repository, not registered, and makes it
// the current directory.
func (w *workspace) linked(t *testing.T) project.Where {
	t.Helper()
	dir := filepath.Join(filepath.Dir(w.repo.Dir), "feature")
	w.repo.Git("worktree", "add", "-b", "feature", dir)
	t.Chdir(dir)
	where, err := project.Locate(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	return where
}

// without is a state with the lines of the tables named left out.
func without(state string, tables ...string) string {
	var out []string
	for _, line := range strings.Split(state, "\n") {
		if !slices.ContainsFunc(tables, func(t string) bool { return strings.HasPrefix(line, t+": ") }) {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// WP2.5 spec §4.1 step 8, §11.2 (D9): S is the commit the active target's
// ref resolves to; without a target, or with a ref that names no commit,
// there is none and it is not an error; a ref that cannot be read as a
// commit refuses the batch (exit 6) before anything runs.
func TestPrepareSnapshot(t *testing.T) {
	cases := []struct {
		name string
		ref  func(w *workspace) string
		want func(w *workspace) string
		exit int
	}{
		{"no target", nil, func(*workspace) string { return "" }, 0},
		{"the branch", func(w *workspace) string { return w.repo.Git("symbolic-ref", "HEAD") },
			func(w *workspace) string { return w.repo.Git("rev-parse", "HEAD") }, 0},
		{"a branch with no commit", func(*workspace) string { return "refs/heads/none" }, func(*workspace) string { return "" }, 0},
		{"a tag of a blob", func(w *workspace) string {
			w.repo.Git("tag", "a-blob", w.repo.Git("rev-parse", "HEAD:README.md"))
			return "refs/tags/a-blob"
		}, nil, apperr.ExitEnvironment},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorkspace(t)
			w.criteria(t, ac{key: "AC-001-1", check: sh("echo one")})
			if c.ref != nil {
				w.target(t, strings.TrimSpace(c.ref(w)))
			}
			h := checkrun.NewHost()
			b, err := run(t, h, all())
			if c.exit != 0 {
				refused(t, h, err, c.exit)
				return
			}
			if err != nil || b.Snapshot != strings.TrimSpace(c.want(w)) {
				t.Fatalf("S = %q, %v; want %q", b.Snapshot, err, c.want(w))
			}
		})
	}
}

// WP2.5 spec §4.1 step 8 (PD-26): a git that would fetch the objects a
// partial clone lacks refuses the batch (exit 6) before any command that
// reads an object, the private area and S; it is judged before the output
// (step 9) and an empty batch's registration (step 10).
func TestPrepareRefusesAGitThatFetches(t *testing.T) {
	w := newWorkspace(t)
	w.target(t, strings.TrimSpace(w.repo.Git("symbolic-ref", "HEAD")))
	w.linked(t)
	before := w.state(t)
	gitDir, env, calls := testutil.PassthroughGit(t, testutil.PassWithoutNoLazyFetch)
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		t.Setenv(k, v)
	}
	t.Setenv("PATH", gitDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	h := checkrun.NewHost()
	_, err := checkrun.PrepareWith(ctx, h, all(), checkrun.Output{Stdout: &bytes.Buffer{}, Stderr: os.Stderr})
	if doc := refused(t, h, err, apperr.ExitEnvironment); !strings.Contains(doc, "PD-26") {
		t.Fatalf("not the PD-26 refusal: %s", doc)
	}
	for _, c := range calls() {
		if slices.ContainsFunc(c.Args, func(a string) bool { return strings.HasPrefix(a, "refs/heads/") || a == "HEAD" }) {
			t.Errorf("a ref was read after the refusal: %v", c.Args)
		}
	}
	if _, err := os.Lstat(w.home.Join(datadir.TmpDir)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the private area was readied: %v", err)
	}
	if after := w.state(t); after != before {
		t.Errorf("the refusal wrote:\n%s\n->\n%s", before, after)
	}
}

// WP2.5 spec §4.1 step 9, §9.1, §19.9 (D20 (i)): a check's output goes to
// inherited files: the raw stdout and stderr, or with --json the raw
// stderr for both (stdout then carries only Specmetry's document). Any
// other destination, a nil *os.File too, is refused before a check starts
// (internal, exit 1), at the stream it would carry.
func TestPrepareOutput(t *testing.T) {
	newWorkspace(t).criteria(t, ac{key: "AC-001-1", check: sh("echo one")})
	var nilFile *os.File
	cases := []struct {
		name   string
		json   bool
		out    checkrun.Output
		refuse []string
	}{
		{"files", false, checkrun.Output{Stdout: os.Stdout, Stderr: os.Stderr}, nil},
		{"a buffer for stdout", false, checkrun.Output{Stdout: &bytes.Buffer{}, Stderr: os.Stderr}, []string{"stdout"}},
		{"buffers", false, checkrun.Output{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}, []string{"stdout", "stderr"}},
		{"a nil file", false, checkrun.Output{Stdout: os.Stdout, Stderr: nilFile}, []string{"stderr"}},
		{"json, a buffer for stdout", true, checkrun.Output{Stdout: &bytes.Buffer{}, Stderr: os.Stderr}, nil},
		{"json, a buffer for stderr", true, checkrun.Output{Stdout: os.Stdout, Stderr: &bytes.Buffer{}}, []string{"stderr"}},
	}
	for _, c := range cases {
		opts := all()
		opts.JSON = c.json
		h := checkrun.NewHost()
		b, err := checkrun.PrepareWith(ctx, h, opts, c.out)
		if c.refuse != nil {
			if doc := refused(t, h, err, apperr.ExitInternal, c.refuse...); !strings.Contains(doc, "the output cannot carry check output") {
				t.Errorf("%s: %s", c.name, doc)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		wantOut := os.Stdout
		if c.json {
			wantOut = os.Stderr
		}
		if b.Stdout != wantOut || b.Stderr != os.Stderr {
			t.Errorf("%s: the check's output goes to %v and %v", c.name, b.Stdout, b.Stderr)
		}
		_ = b.Close()
	}
}

// emptyDoc checks an empty batch's document against check-run.v1 and
// fixed values: W's worktree, no runs, isolation off and the D9 summary.
func emptyDoc(t *testing.T, b *checkrun.Batch, worktreeID string, onTarget bool) {
	t.Helper()
	d := b.Document
	if d == nil || d.WorktreeID != worktreeID || d.Runs == nil || len(d.Runs) != 0 || d.Isolation != "off" || d.IsTargetSnapshot != onTarget ||
		d.Schema != "specmetry.check-run/v1" || b.WorktreeID != worktreeID {
		t.Fatalf("document %+v; want worktree %s, target snapshot %v", d, worktreeID, onTarget)
	}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	val, err := validator()
	if err != nil {
		t.Fatal(err)
	}
	if err := val.ValidateJSON("check-run.v1", raw); err != nil || !strings.Contains(string(raw), `"runs":[]`) {
		t.Fatalf("not a check-run.v1 document with no runs: %v\n%s", err, raw)
	}
}

// WP2.5 spec §4.1 step 10, §8.6 D table, D18, E3, D9 (MVP-EV-27): --all
// with nothing to run runs nothing and has a document with no runs. A
// registered W is written nothing at all (no seq, no last_seen_at) and its
// worktree is given; an unregistered one is registered alone, with one
// seq, and the next call writes nothing. is_target_snapshot compares HEAD
// at the command's start with S.
func TestPrepareEmptyBatch(t *testing.T) {
	w := newWorkspace(t)
	w.criteria(t, ac{key: "AC-001-1", verification: "manual"}, ac{key: "AC-001-2", status: "retired", check: sh("echo two")})
	w.repo.Git("tag", "release")
	w.target(t, "refs/tags/release")

	before := w.state(t)
	b, err := run(t, checkrun.NewHost(), all())
	if err != nil {
		t.Fatal(err)
	}
	emptyDoc(t, b, wsMain, true)
	_ = b.Close()
	if after := w.state(t); after != before {
		t.Fatalf("an empty batch of a registered worktree wrote:\n%s\n->\n%s", before, after)
	}

	where := w.linked(t)
	if err := os.WriteFile(filepath.Join(where.Current, "b.md"), []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.repo.GitIn(where.Current, "add", "b.md")
	w.repo.GitIn(where.Current, "commit", "-m", "b")
	before = w.state(t)
	b, err = run(t, checkrun.NewHost(), all())
	if err != nil {
		t.Fatal(err)
	}
	registered := b.WorktreeID
	if registered == "" || registered == wsMain {
		t.Fatalf("the worktree was not registered: %q", registered)
	}
	emptyDoc(t, b, registered, false)
	_ = b.Close()
	after := w.state(t)
	if without(after, "worktrees", "meta") != without(before, "worktrees", "meta") {
		t.Fatalf("an empty batch wrote more than the registration:\n%s\n->\n%s", before, after)
	}
	got := rowsOf(t, w, `SELECT id, path, registered_seq, (SELECT seq_counter FROM meta), (SELECT count(*) FROM worktrees) FROM worktrees
WHERE path = ?`, where.Current)
	if want := [][]string{{registered, where.Current, "4", "4", "2"}}; !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("registration %v, want %v", got, want)
	}

	before = after
	b, err = run(t, checkrun.NewHost(), all())
	if err != nil {
		t.Fatal(err)
	}
	emptyDoc(t, b, registered, false)
	_ = b.Close()
	if after := w.state(t); after != before {
		t.Fatalf("the second empty batch wrote:\n%s\n->\n%s", before, after)
	}
}

// rowsOf reads query's rows as text.
func rowsOf(t *testing.T, w *workspace, query string, args ...any) [][]string {
	t.Helper()
	var out [][]string
	w.read(t, func(tx *sql.Tx) error {
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
			var row []string
			for _, v := range vals {
				row = append(row, v.String)
			}
			out = append(out, row)
		}
		return rs.Err()
	})
	return out
}

// WP2.5 spec §8.6 D table (MVP-EV-27): the registration's failures are
// told apart. Refused before its transaction's body returned: not
// registered (D3); refused by the commit: whether it was registered is
// not known (D4). Either way exit 6, no check run, and here nothing
// written. A failed read before it (D5: S, PD-26) registers nothing.
func TestPrepareEmptyBatchRegistrationFails(t *testing.T) {
	w := newWorkspace(t)
	w.linked(t)
	before := w.state(t)
	cases := []struct {
		name, message string
		hook          func(*sql.Tx) error
	}{
		{"not committed (D3)", "the worktree was not registered", func(*sql.Tx) error { return errors.New("injected") }},
		{"commit refused (D4)", "may have been registered, or not", func(tx *sql.Tx) error {
			// A row whose project does not exist, its check deferred to
			// the commit, which SQLite then refuses.
			_, err := tx.ExecContext(ctx, `PRAGMA defer_foreign_keys = ON;
INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES ('01JC0000000000000000000W09', '01JC0000000000000000000P09', '/nowhere', NULL, 0, 999, 1)`)
			return err
		}},
	}
	for _, c := range cases {
		checkrun.OnRegister(t, c.hook)
		h := checkrun.NewHost()
		_, err := run(t, h, all())
		if doc := refused(t, h, err, apperr.ExitEnvironment); !strings.Contains(doc, c.message) || !strings.Contains(doc, "no check was run") {
			t.Errorf("%s: %s", c.name, doc)
		}
		if after := w.state(t); after != before {
			t.Errorf("%s wrote:\n%s\n->\n%s", c.name, before, after)
		}
	}
	checkrun.OnRegister(t, nil)
	w.repo.Git("tag", "a-blob", w.repo.Git("rev-parse", "HEAD:README.md"))
	w.target(t, "refs/tags/a-blob")
	before = w.state(t)
	h := checkrun.NewHost()
	_, err := run(t, h, all())
	refused(t, h, err, apperr.ExitEnvironment)
	if after := w.state(t); after != before {
		t.Errorf("a failed read of S registered:\n%s\n->\n%s", before, after)
	}
}

// WP2.5 spec §4.1: step 9 is judged before step 10, so an unsupported
// destination refuses an empty batch (exit 1) and W is not registered.
func TestPrepareOutputBeforeRegistration(t *testing.T) {
	w := newWorkspace(t)
	w.linked(t)
	before := w.state(t)
	h := checkrun.NewHost()
	_, err := checkrun.PrepareWith(ctx, h, all(), checkrun.Output{Stdout: &bytes.Buffer{}, Stderr: os.Stderr})
	refused(t, h, err, apperr.ExitInternal, "stdout")
	if after := w.state(t); after != before {
		t.Errorf("the refusal registered:\n%s\n->\n%s", before, after)
	}
}

// The run items of check-run.v1 as the document holds them: a lawful pass
// and a signal's fail, written out by hand, conform to the schema.
func TestRunItemShape(t *testing.T) {
	head, kind := strings.Repeat("a", 40), "nonzero_exit"
	zero := int64(0)
	doc := checkrun.Document{Schema: "specmetry.check-run/v1", WorktreeID: wsMain, Isolation: "off", Runs: []checkrun.RunItem{
		{Criterion: "AC-001-1", CriterionRevision: 1, EvidenceID: "01JC0000000000000000000E01", Seq: 5, Result: "pass", ExitCode: &zero,
			DurationMS: 840, HeadStart: &head, TreeStateStart: "clean", EvidenceGrade: "observed", Qualifying: true},
		{Criterion: "AC-001-2", CriterionRevision: 2, EvidenceID: "01JC0000000000000000000E02", Seq: 6, Result: "fail", FailureKind: &kind,
			DurationMS: 12, TreeStateStart: "unknown", EvidenceGrade: "observed"},
	}}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	val, err := validator()
	if err != nil {
		t.Fatal(err)
	}
	if err := val.ValidateJSON("check-run.v1", raw); err != nil {
		t.Fatalf("%v\n%s", err, raw)
	}
}

// WP2.5 spec §4.1: step 7 is judged before step 8, so an interpreter that
// cannot be used refuses the batch (shell_unavailable) before a git that
// would fetch is.
func TestPrepareInterpretersBeforePD26(t *testing.T) {
	w := newWorkspace(t)
	w.criteria(t, ac{key: "AC-001-1", check: sh("echo one")})
	w.exec(t, `INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES ('01JC0000000000000000000W03', ?, ?, NULL, 0, 3, 1)`, wsProject, filepath.Join(t.TempDir(), "gone"))
	gitDir, env, _ := testutil.PassthroughGit(t, testutil.PassWithoutNoLazyFetch)
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		t.Setenv(k, v)
	}
	t.Setenv("PATH", gitDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	h := checkrun.NewHost()
	_, err := run(t, h, all())
	if doc := refused(t, h, err, apperr.ExitEnvironment, "AC-001-1"); !strings.Contains(doc, "shell_unavailable") {
		t.Fatalf("not step 7's refusal: %s", doc)
	}
}

// WP2.5 spec §10.3, §8.6 D1: W registered by another call between the
// read and the empty batch's registration is not registered again;
// nothing is written and its worktree is given.
func TestPrepareEmptyBatchRegisteredMeanwhile(t *testing.T) {
	w := newWorkspace(t)
	where := w.linked(t)
	checkrun.AfterRead(t, func() {
		w.exec(t, `INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES ('01JC0000000000000000000W05', ?, ?, ?, 0, 3, 7); UPDATE meta SET seq_counter = 3 WHERE id = 1`,
			wsProject, where.Current, where.GitDir)
	})
	var after string
	checkrun.OnRegister(t, func(*sql.Tx) error {
		after = "reached"
		return nil
	})
	b, err := run(t, checkrun.NewHost(), all())
	if err != nil {
		t.Fatal(err)
	}
	emptyDoc(t, b, "01JC0000000000000000000W05", false)
	_ = b.Close()
	got := rowsOf(t, w, `SELECT last_seen_at, (SELECT seq_counter FROM meta), (SELECT count(*) FROM worktrees) FROM worktrees WHERE path = ?`,
		where.Current)
	if want := [][]string{{"7", "3", "2"}}; !slices.EqualFunc(got, want, slices.Equal) || after != "reached" {
		t.Fatalf("worktree %v (want %v), registration transaction %q", got, want, after)
	}
}
