package checkrun_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/checkrun"
	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
	"github.com/XKHoshizora/specmetry/internal/testutil/dbdump"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/toderive"
)

var ctx = context.Background()

const (
	projectA  = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"
	projectB  = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A7"
	mainA     = "01J9Z3K5Q8R2T4V6W8X0Y2Z4B6"
	mainB     = "01J9Z3K5Q8R2T4V6W8X0Y2Z4B7"
	criterion = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N3"
	critB     = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N9"
	config    = "5d41402abc4b2a76b9719d911017c592aa5d41402abc4b2a76b9719d911017c5"
)

// Where project A's worktrees are: its main one, registered with the
// project, and a second one, not registered yet.
var (
	whereMain   = project.Where{Git: true, Root: "/r/a", Current: "/r/a", CommonDir: "/r/a/.git", GitDir: "/r/a/.git"}
	whereSecond = project.Where{Git: true, Root: "/r/a", Current: "/r/a-2", CommonDir: "/r/a/.git", GitDir: "/r/a/.git/worktrees/a-2"}
)

// store opens a fresh database with projects A and B, each with its main
// worktree, a requirement and an automated criterion with a check.
func store(t *testing.T) *sqlite.DB {
	t.Helper()
	db, _, err := sqlite.Open(ctx, testutil.TempHome(t), sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	exec(t, db, `
INSERT INTO projects (id, name, git_common_dir, main_worktree_id, stage, settings_json, created_at)
VALUES (?1, 'a', '/r/a/.git', ?3, 'in_development', '{}', 1), (?2, 'b', '/r/b/.git', ?4, 'in_development', '{}', 1);
INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES (?3, ?1, '/r/a', '/r/a/.git', 1, 1, 1), (?4, ?2, '/r/b', '/r/b/.git', 1, 2, 1);
INSERT INTO baseline_versions (id, seq, project_id, number, summary, actor, channel)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4M1', 3, ?1, 1, 'v1', 'human:me', 'human_cli'), ('01J9Z3K5Q8R2T4V6W8X0Y2Z4M2', 4, ?2, 1, 'v1', 'human:me', 'human_cli');
INSERT INTO requirements (id, project_id, key, status, revision, revision_seq, origin, created_version, updated_version, updated_seq,
    title, kind, priority)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4N2', ?1, 'REQ-001', 'active', 1, 3, 'extracted', 1, 1, 3, 'orders', 'functional', 'must'),
       ('01J9Z3K5Q8R2T4V6W8X0Y2Z4N8', ?2, 'REQ-001', 'active', 1, 4, 'extracted', 1, 1, 4, 'orders', 'functional', 'must');
INSERT INTO acceptance_criteria (id, project_id, key, status, revision, revision_seq, origin, created_version, updated_version,
    updated_seq, requirement_id, requirement_revision, statement, verification, check_command, check_rationale, check_config_hash)
VALUES (?5, ?1, 'AC-001-1', 'active', 1, 3, 'extracted', 1, 1, 3, '01J9Z3K5Q8R2T4V6W8X0Y2Z4N2', 1, 'orders are isolated', 'automated',
        'go test ./orders', 'it tests the criterion', ?7),
       (?6, ?2, 'AC-001-1', 'active', 1, 4, 'extracted', 1, 1, 4, '01J9Z3K5Q8R2T4V6W8X0Y2Z4N8', 1, 'orders are isolated', 'automated',
        'go test ./orders', 'it tests the criterion', ?7);
UPDATE meta SET seq_counter = 4 WHERE id = 1;`, projectA, projectB, mainA, mainB, criterion, critB, config)
	return db
}

func exec(t *testing.T, db *sqlite.DB, query string, args ...any) {
	t.Helper()
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func count(t *testing.T, db *sqlite.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error { return tx.QueryRowContext(ctx, query, args...).Scan(&n) }); err != nil {
		t.Fatal(err)
	}
	return n
}

func seqCounter(t *testing.T, db *sqlite.DB) int64 {
	t.Helper()
	return count(t, db, `SELECT seq_counter FROM meta WHERE id = 1`)
}

// passRun is a lawful observed pass of criterion on commit A.
func passRun() checkrun.Run {
	return checkrun.Run{
		CriterionID: criterion, CriterionRevision: 1, RequirementRevision: 1, CheckConfigHash: config, Workdir: ".",
		HeadStart: headA, HeadEnd: headA, TreeStateStart: checkrun.Clean, TreeStateEnd: checkrun.Clean,
		Result: checkrun.Pass, ExitCode: code(0), DurationMS: 8421, Channel: "agent_cli",
		StartedAt: 1793444000000, FinishedAt: 1793444008500,
	}
}

func write(t *testing.T, db *sqlite.DB, projectID string, where project.Where, run checkrun.Run) checkrun.Row {
	t.Helper()
	var row checkrun.Row
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		var err error
		row, err = checkrun.Write(ctx, tx, projectID, where, run)
		return err
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return row
}

// WP2.5 spec §10.3, data-model §4.11: in one write transaction, a worktree
// seen for the first time is registered with a seq of its own, then the run
// gets the next one; a registered worktree takes no seq. The row read back
// is the run as given, with the observed grade and isolation off, the input
// record sorted by path, and nothing in the isolation columns or
// session_id.
func TestWrite(t *testing.T) {
	db := store(t)
	before := seqCounter(t, db)
	run := passRun()
	run.Inputs = []checkrun.Input{
		{Path: "src/b.go", Mode: "100644", Type: "blob", OID: strings.Repeat("b", 40)},
		{Path: "src/a.go", Mode: "100755", Type: "blob", OID: strings.Repeat("a", 40)},
	}
	got := write(t, db, projectA, whereSecond, run)
	if got.Seq != before+2 || seqCounter(t, db) != before+2 {
		t.Fatalf("seq %d, counter %d; want the worktree at %d and the run at %d", got.Seq, seqCounter(t, db), before+1, before+2)
	}
	if n := count(t, db, `SELECT count(*) FROM worktrees WHERE id = ? AND project_id = ? AND path = '/r/a-2' AND registered_seq = ? AND is_main = 0`,
		got.WorktreeID, projectA, before+1); n != 1 {
		t.Errorf("the second worktree is not registered at seq %d: %d", before+1, n)
	}
	want := checkrun.Row{ID: got.ID, Seq: before + 2, WorktreeID: got.WorktreeID, Run: run, EvidenceGrade: checkrun.Observed, IsolationMode: checkrun.Off}
	want.Inputs = []checkrun.Input{run.Inputs[1], run.Inputs[0]}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("row read back:\n%+v\nwant\n%+v", got, want)
	}
	if n := count(t, db, `SELECT count(*) FROM check_runs WHERE id = ? AND isolation_backend IS NULL AND isolation_fallback_reason IS NULL
    AND session_id IS NULL AND inputs_json = '[{"path":"src/a.go","mode":"100755","type":"blob","oid":"`+strings.Repeat("a", 40)+
		`"},{"path":"src/b.go","mode":"100644","type":"blob","oid":"`+strings.Repeat("b", 40)+`"}]'`, got.ID); n != 1 {
		t.Errorf("the stored columns differ: %d", n)
	}

	// The same worktree again, then the main one: one seq each, no
	// registration.
	worktrees := count(t, db, `SELECT count(*) FROM worktrees`)
	again := write(t, db, projectA, whereSecond, passRun())
	if again.Seq != before+3 || again.WorktreeID != got.WorktreeID {
		t.Errorf("a second run in the worktree: seq %d in %s; want %d in %s", again.Seq, again.WorktreeID, before+3, got.WorktreeID)
	}
	main := write(t, db, projectA, whereMain, passRun())
	if main.Seq != before+4 || main.WorktreeID != mainA {
		t.Errorf("a run in the main worktree: seq %d in %s; want %d in %s", main.Seq, main.WorktreeID, before+4, mainA)
	}
	if n := count(t, db, `SELECT count(*) FROM worktrees`); n != worktrees {
		t.Errorf("worktrees %d, want %d", n, worktrees)
	}

	// The NULL columns: no HEAD, no failure kind, no exit code, no input
	// record, read back as their zero values.
	signal := passRun()
	signal.HeadStart, signal.HeadEnd, signal.TreeStateStart, signal.TreeStateEnd = "", "", checkrun.Unknown, checkrun.Unknown
	signal.Result, signal.FailureKind, signal.ExitCode = checkrun.Fail, checkrun.NonzeroExit, nil
	r := write(t, db, projectA, whereMain, signal)
	if r.HeadStart != "" || r.HeadEnd != "" || r.ExitCode != nil || r.Inputs != nil || r.FailureKind != checkrun.NonzeroExit {
		t.Errorf("NULL columns read back as %+v", r)
	}
	if n := count(t, db, `SELECT count(*) FROM check_runs WHERE id = ? AND head_start IS NULL AND head_end IS NULL AND exit_code IS NULL
    AND inputs_json IS NULL`, r.ID); n != 1 {
		t.Errorf("the NULL columns are not stored as NULL: %d", n)
	}
}

// WP2.5 spec §10.3, D13, CX-15: a run keeps the criterion revision,
// requirement revision and configuration read at its start; a later
// revision, or the criterion's retirement, does not refuse the write.
func TestWriteKeepsTheRevisionOfItsStart(t *testing.T) {
	db := store(t)
	exec(t, db, `UPDATE acceptance_criteria SET revision = 2, check_config_hash = ? WHERE id = ?`, strings.Repeat("c", 64), criterion)
	exec(t, db, `UPDATE requirements SET revision = 3 WHERE id = '01J9Z3K5Q8R2T4V6W8X0Y2Z4N2'`)
	r := write(t, db, projectA, whereMain, passRun())
	if r.CriterionRevision != 1 || r.RequirementRevision != 1 || r.CheckConfigHash != config {
		t.Errorf("the run binds %d, %d, %s", r.CriterionRevision, r.RequirementRevision, r.CheckConfigHash)
	}
	exec(t, db, `UPDATE acceptance_criteria SET status = 'retired', retired_version = 1 WHERE id = ?`, criterion)
	write(t, db, projectA, whereMain, passRun())
}

// WP2.5 spec §10.3, Review 120 §4.2: a failed first registration, a failed
// fact write and a rolled-back transaction each leave no run, no new
// worktree and the seq counter where it was. The registration failure is
// a trigger of the test database: SQLite itself refuses the insert, and the
// same write passes once it is gone.
func TestWriteFailuresLeaveNothing(t *testing.T) {
	db := store(t)
	state := func() [3]int64 {
		return [3]int64{count(t, db, `SELECT count(*) FROM check_runs`), count(t, db, `SELECT count(*) FROM worktrees`), seqCounter(t, db)}
	}
	attempt := func(projectID string, where project.Where, run checkrun.Run, after func() error) error {
		return db.WriteTx(ctx, func(tx *sql.Tx) error {
			if _, err := checkrun.Write(ctx, tx, projectID, where, run); err != nil {
				return err
			}
			return after()
		})
	}
	none := func() error { return nil }
	start := state()

	exec(t, db, `CREATE TRIGGER refuse_worktrees BEFORE INSERT ON worktrees BEGIN SELECT RAISE(ABORT, 'test: worktree registration refused'); END`)
	if err := attempt(projectA, whereSecond, passRun(), none); err == nil || !strings.Contains(err.Error(), "worktree registration refused") {
		t.Errorf("a failed first registration: %v", err)
	}
	if got := state(); got != start {
		t.Errorf("after a failed first registration: runs, worktrees, seq = %v; want %v", got, start)
	}
	exec(t, db, `DROP TRIGGER refuse_worktrees`)

	inconsistent := passRun()
	inconsistent.ExitCode = code(1)
	if err := attempt(projectA, whereSecond, inconsistent, none); err == nil || !strings.Contains(err.Error(), "CHECK constraint failed") {
		t.Errorf("a pass with exit code 1: %v", err)
	}
	if got := state(); got != start {
		t.Errorf("after a refused run: runs, worktrees, seq = %v; want %v", got, start)
	}

	foreign := passRun()
	foreign.CriterionID = critB
	if err := attempt(projectA, whereSecond, foreign, none); err == nil || !strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
		t.Errorf("a criterion of project B: %v", err)
	}
	if got := state(); got != start {
		t.Errorf("after a run refused at commit: runs, worktrees, seq = %v; want %v", got, start)
	}

	if err := attempt(projectA, whereSecond, passRun(), func() error { return errors.New("the caller rolls back") }); err == nil {
		t.Error("the caller's error was lost")
	}
	if got := state(); got != start {
		t.Errorf("after a rollback: runs, worktrees, seq = %v; want %v", got, start)
	}

	// The control: the same write, nothing in its way.
	write(t, db, projectA, whereSecond, passRun())
	if got, want := state(), [3]int64{start[0] + 1, start[1] + 1, start[2] + 2}; got != want {
		t.Errorf("after the write: runs, worktrees, seq = %v; want %v", got, want)
	}
}

// dbdump reads check_runs as stored (WP2.5 spec §10.2), and toderive hands
// the derive engine the fields it reads; nothing is computed on the way.
func TestDumpMapsCheckRuns(t *testing.T) {
	db := store(t)
	pass := passRun()
	pass.Inputs = []checkrun.Input{{Path: "src/a.go", Mode: "100644", Type: "blob", OID: strings.Repeat("a", 40)}}
	changed := passRun()
	changed.HeadEnd, changed.TreeStateEnd, changed.Result, changed.FailureKind, changed.ExitCode = "", checkrun.Unknown, checkrun.Error, checkrun.ContextChanged, nil
	rows := []checkrun.Row{write(t, db, projectA, whereMain, pass), write(t, db, projectA, whereSecond, changed)}
	write(t, db, projectB, project.Where{Git: true, Root: "/r/b", Current: "/r/b", CommonDir: "/r/b/.git", GitDir: "/r/b/.git"},
		checkrun.Run{CriterionID: critB, CriterionRevision: 1, RequirementRevision: 1, CheckConfigHash: config, Workdir: ".",
			TreeStateStart: checkrun.Unknown, TreeStateEnd: checkrun.Unknown, Result: checkrun.Pass, ExitCode: code(0), Channel: "human_cli"})
	var d factgen.Dump
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		var err error
		d, err = dbdump.Load(ctx, tx, projectA)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(d.CheckRuns) != len(rows) {
		t.Fatalf("dump has %d runs of A, want %d", len(d.CheckRuns), len(rows))
	}
	p := func(s string) *string { return &s }
	inputs := `[{"path":"src/a.go","mode":"100644","type":"blob","oid":"` + strings.Repeat("a", 40) + `"}]`
	want := []factgen.CheckRunRow{
		{ID: rows[0].ID, Seq: rows[0].Seq, CriterionID: criterion, CriterionRevision: 1, RequirementRevision: 1, CheckConfigHash: config,
			InputsJSON: &inputs, WorktreeID: mainA, EvidenceGrade: checkrun.Observed, HeadStart: p(headA), HeadEnd: p(headA),
			TreeStateStart: checkrun.Clean, TreeStateEnd: checkrun.Clean, Result: checkrun.Pass},
		{ID: rows[1].ID, Seq: rows[1].Seq, CriterionID: criterion, CriterionRevision: 1, RequirementRevision: 1, CheckConfigHash: config,
			WorktreeID: rows[1].WorktreeID, EvidenceGrade: checkrun.Observed, HeadStart: p(headA), TreeStateStart: checkrun.Clean,
			TreeStateEnd: checkrun.Unknown, Result: checkrun.Error, FailureKind: p(checkrun.ContextChanged)},
	}
	byID := map[string]factgen.CheckRunRow{}
	for _, r := range d.CheckRuns {
		byID[r.ID] = r
	}
	for _, w := range want {
		if got := byID[w.ID]; !reflect.DeepEqual(got, w) {
			t.Errorf("dump row:\n%+v\nwant\n%+v", got, w)
		}
	}
	facts := toderive.Facts(d)
	for _, r := range facts.CheckRuns {
		if r.ID == rows[0].ID {
			w := derive.CheckRun{ID: rows[0].ID, Seq: rows[0].Seq, CriterionID: criterion, CriterionRevision: 1, RequirementRevision: 1,
				CheckConfigHash: config, Inputs: derive.InputRecord{{Path: "src/a.go", Mode: "100644", Type: "blob", OID: strings.Repeat("a", 40)}},
				WorktreeID: mainA, EvidenceGrade: derive.Observed, HeadStart: headA, HeadEnd: headA, TreeStateStart: "clean", TreeStateEnd: "clean",
				Result: "pass"}
			if !reflect.DeepEqual(r, w) {
				t.Errorf("derive input:\n%+v\nwant\n%+v", r, w)
			}
		}
	}
}
