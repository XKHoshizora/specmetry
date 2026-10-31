package sqlite

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// sqlNull is NULL in a runRow, where nil takes the field's default.
type sqlNull struct{}

// runRow is a check_runs row for the constraint tests. Each field left
// zero takes the default of a lawful observed pass of criterion in
// project A: clean at both ends on one HEAD, exit code 0.
type runRow struct {
	id, project, criterion, worktree, hash, grade, isolation string
	treeStart, treeEnd, result, channel                      string
	inputs, backend, fallback, headStart, headEnd, kind      any
	code, revision                                           any
	seq, duration                                            int64
}

const runHead = "8f3c2a1b9d7e6f5a4b3c2d1e0f9a8b7c6d5e4f3a"

func insertRunWith(verb string, r runRow) func(tx *sql.Tx) error {
	or := func(s, def string) string {
		if s == "" {
			return def
		}
		return s
	}
	value := func(v, def any) any {
		switch v.(type) {
		case nil:
			return def
		case sqlNull:
			return nil
		}
		return v
	}
	return stmt(verb+` INTO check_runs (id, seq, project_id, criterion_id, criterion_revision, requirement_revision, check_config_hash,
    inputs_json, worktree_id, workdir, evidence_grade, isolation_mode, isolation_backend, isolation_fallback_reason, head_start, head_end,
    tree_state_start, tree_state_end, result, failure_kind, exit_code, duration_ms, triggered_channel, session_id, started_at, finished_at)
VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, '.', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, 1793444000000, 1793444008421)`,
		r.id, r.seq, or(r.project, projectA), or(r.criterion, criterion), value(r.revision, 1), or(r.hash, shaA), value(r.inputs, nil),
		or(r.worktree, worktreeA), or(r.grade, "observed"), or(r.isolation, "off"), value(r.backend, nil), value(r.fallback, nil),
		value(r.headStart, runHead), value(r.headEnd, runHead), or(r.treeStart, "clean"), or(r.treeEnd, "clean"), or(r.result, "pass"),
		value(r.kind, nil), value(r.code, 0), r.duration, or(r.channel, "agent_cli"))
}

func insertRun(r runRow) func(tx *sql.Tx) error { return insertRunWith("INSERT", r) }

// check_runs (0009; data-model §4.11, WP2.5 spec §10.1 and §19.10 D14):
// result, failure_kind, exit_code and duration_ms agree, and a NULL never
// lets a row through that breaks them; the new rows never hold not_found
// (D5); the observed runner's grade and isolation only, with no isolation
// column; one seq per run, unique in the table; the criterion and the
// worktree of the run's own project; no check of the current revision
// (CX-15); append-only, REPLACE and upserts included; purged with the
// project.
func TestCheckRunsConstraints(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{})
	defer func() { _ = db.Close() }()
	ok := func(name string, fn func(tx *sql.Tx) error) {
		t.Helper()
		if err := db.WriteTx(ctx, fn); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	refused := func(name, why string, fn func(tx *sql.Tx) error) {
		t.Helper()
		err := db.WriteTx(ctx, fn)
		switch {
		case err == nil:
			t.Errorf("%s was accepted", name)
		case !strings.Contains(err.Error(), why):
			t.Errorf("%s was refused for another reason: %v (want %q)", name, err, why)
		}
	}
	const check, unique, fk = "CHECK constraint failed", "UNIQUE constraint failed", "FOREIGN KEY constraint failed"
	const (
		reqB  = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N8"
		critB = "01J9Z3K5Q8R2T4V6W8X0Y2Z4N9"
	)
	ok("projects, a version, requirements and criteria", func(tx *sql.Tx) error {
		for _, f := range []func() error{
			func() error { return insertProject(tx, projectA, "/r/a/.git", worktreeA, "/r/a") },
			func() error { return insertProject(tx, projectB, "/r/b/.git", worktreeB, "/r/b") },
			func() error { return insertVersion(tx, versionA, 1, "human:me") },
			func() error { return insertRequirement(tx, reqA, "REQ-001", nil) },
			func() error { return insertCriterion(tx, criterion, "AC-001-1", reqA, "automated", "go test ./...") },
			func() error {
				return stmt(`INSERT INTO requirements (id, project_id, key, status, revision, revision_seq, origin, created_version,
    updated_version, updated_seq, title, kind, priority)
VALUES (?, ?, 'REQ-001', 'active', 1, 1, 'extracted', 1, 1, 1, 'log in', 'functional', 'must')`, reqB, projectB)(tx)
			},
			func() error {
				return stmt(`INSERT INTO acceptance_criteria (id, project_id, key, status, revision, revision_seq, origin,
    created_version, updated_version, updated_seq, requirement_id, requirement_revision, statement, verification)
VALUES (?, ?, 'AC-001-1', 'active', 1, 1, 'extracted', 1, 1, 1, ?, 1, 'in B', 'manual')`, critB, projectB, reqB)(tx)
			},
		} {
			if err := f(); err != nil {
				return err
			}
		}
		return nil
	})
	var ids rowIDs
	seq := int64(100)
	next := func(r runRow) runRow {
		seq++
		r.id, r.seq = ids.next(), seq
		return r
	}
	null := sqlNull{}
	other := strings.Repeat("b", 40)
	// The lawful rows of the truth table (§19.10), each accepted.
	for _, c := range []struct {
		name string
		r    runRow
	}{
		{"pass, exit 0", runRow{duration: 8421}},
		{"fail, nonzero_exit, 1", runRow{result: "fail", kind: "nonzero_exit", code: 1}},
		{"fail, nonzero_exit, NULL (a Unix signal, D12)", runRow{result: "fail", kind: "nonzero_exit", code: null}},
		{"fail, a Windows code above 255 (D17)", runRow{result: "fail", kind: "nonzero_exit", code: int64(3221225477)}},
		{"error, timeout, NULL", runRow{result: "error", kind: "timeout", code: null, duration: 300000}},
		{"error, spawn_error, NULL, 0 ms", runRow{result: "error", kind: "spawn_error", code: null}},
		{"error, input_missing, NULL, 0 ms", runRow{result: "error", kind: "input_missing", code: null}},
		{"error, context_changed, 0 (Review 091 N5)", runRow{result: "error", kind: "context_changed", treeEnd: "dirty"}},
		{"error, context_changed, NULL, end not read (D19)", runRow{result: "error", kind: "context_changed", code: null, headEnd: null,
			treeEnd: "unknown", duration: 300000}},
		{"error, context_changed, 5, end not read (A12)", runRow{result: "error", kind: "context_changed", code: 5, headEnd: null, treeEnd: "unknown"}},
		{"error, context_changed with no process", runRow{result: "error", kind: "context_changed", code: null, headEnd: other}},
		{"no commit at either end", runRow{headStart: null, headEnd: null, treeStart: "unknown", treeEnd: "unknown"}},
		{"an input record", runRow{inputs: `[{"path":"src/a.go","mode":"100644","type":"blob","oid":"` + strings.Repeat("5", 40) + `"}]`}},
		{"a person's run", runRow{channel: "human_cli"}},
	} {
		ok(c.name, insertRun(next(c.r)))
	}
	// Each constraint refuses, for its own reason. The rows with a NULL are
	// those a CHECK written with = or IN alone would let through.
	for _, c := range []struct {
		name, why string
		r         runRow
	}{
		{"pass with a NULL exit code", check, runRow{code: null}},
		{"pass with exit code 1", check, runRow{code: 1}},
		{"pass with a failure kind", check, runRow{kind: "nonzero_exit"}},
		{"pass, timeout, 0 (Review 089)", check, runRow{kind: "timeout"}},
		{"fail with no failure kind", check, runRow{result: "fail", code: 1}},
		{"fail as a timeout", check, runRow{result: "fail", kind: "timeout", code: null}},
		{"fail with exit code 0", check, runRow{result: "fail", kind: "nonzero_exit"}},
		{"error with no failure kind", check, runRow{result: "error", code: null}},
		{"error as nonzero_exit", check, runRow{result: "error", kind: "nonzero_exit", code: 1}},
		{"a new not_found row (D5)", check, runRow{result: "error", kind: "not_found", code: null}},
		{"a failure kind outside the enum", check, runRow{result: "error", kind: "killed", code: null}},
		{"timeout with exit code 0", check, runRow{result: "error", kind: "timeout"}},
		{"spawn_error with exit code 1", check, runRow{result: "error", kind: "spawn_error", code: 1}},
		{"input_missing with exit code 0", check, runRow{result: "error", kind: "input_missing"}},
		{"spawn_error that lasted", check, runRow{result: "error", kind: "spawn_error", code: null, duration: 5}},
		{"input_missing that lasted", check, runRow{result: "error", kind: "input_missing", code: null, duration: 1}},
		{"a result outside the enum", check, runRow{result: "skipped", code: null}},
		{"a negative duration", check, runRow{duration: -1}},
		{"revision 0", check, runRow{revision: 0}},
		{"an isolated grade (M7)", check, runRow{grade: "snapshot_isolated"}},
		{"isolation required", check, runRow{isolation: "required"}},
		{"isolation auto", check, runRow{isolation: "auto"}},
		{"an isolation backend", check, runRow{backend: "container"}},
		{"a fallback reason", check, runRow{fallback: "backend_unavailable"}},
		{"a short configuration hash", check, runRow{hash: "abc"}},
		{"an upper-case configuration hash", check, runRow{hash: strings.Repeat("A", 64)}},
		{"a short head", check, runRow{headStart: "abc", headEnd: "abc"}},
		{"an upper-case head", check, runRow{headStart: strings.Repeat("A", 40), headEnd: strings.Repeat("A", 40)}},
		{"a tree state outside the enum", check, runRow{treeEnd: "undecidable"}},
		{"a channel outside the enum", check, runRow{channel: "system"}},
		{"an input record that is an object", check, runRow{inputs: `{"path":"a"}`}},
		{"an input record that is not JSON", check, runRow{inputs: `[`}},
		{"a criterion of another project", fk, runRow{criterion: critB}},
		{"a worktree of another project", fk, runRow{worktree: worktreeB}},
		{"an unknown project", fk, runRow{project: "01J9Z3K5Q8R2T4V6W8X0Y2Z4ZZ"}},
	} {
		refused(c.name, c.why, insertRun(next(c.r)))
	}
	first := next(runRow{})
	ok("a run", insertRun(first))
	refused("a second run with its seq", unique, insertRun(runRow{id: ids.next(), seq: first.seq}))
	refused("a second run with its ID", unique, insertRun(runRow{id: first.id, seq: 999}))
	// CX-15, D13: a run keeps the revision of its start; neither a later
	// revision nor the criterion's retirement refuses it.
	ok("the criterion is revised and retired", stmt(`UPDATE acceptance_criteria SET revision = 3, status = 'retired', retired_version = 1 WHERE id = ?`, criterion))
	ok("a run of the revision of its start", insertRun(next(runRow{revision: 1})))
	for _, c := range []struct {
		name string
		fn   func(tx *sql.Tx) error
	}{
		{"changing a run", stmt(`UPDATE check_runs SET result = 'fail' WHERE id = ?`, first.id)},
		{"deleting a run", stmt(`DELETE FROM check_runs WHERE id = ?`, first.id)},
		{"replacing a run", insertRunWith("INSERT OR REPLACE", runRow{id: first.id, seq: 998})},
		{"replacing a run by its seq", insertRunWith("REPLACE", runRow{id: ids.next(), seq: first.seq})},
		{"an upsert of a run", func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `INSERT INTO check_runs (id, seq, project_id, criterion_id, criterion_revision, requirement_revision,
    check_config_hash, worktree_id, workdir, evidence_grade, isolation_mode, tree_state_start, tree_state_end, result, exit_code,
    duration_ms, triggered_channel, started_at, finished_at)
VALUES (?, 997, ?, ?, 1, 1, ?, ?, '.', 'observed', 'off', 'clean', 'clean', 'pass', 0, 0, 'agent_cli', 1, 1)
ON CONFLICT (id) DO UPDATE SET workdir = 'changed'`, first.id, projectA, criterion, shaA, worktreeA)
			return err
		}},
	} {
		refused(c.name, "append-only table check_runs", c.fn)
	}
	if n := countRows(t, db, `SELECT count(*) FROM check_runs WHERE id = ? AND seq = ? AND result = 'pass' AND exit_code IS 0 AND workdir = '.'`,
		first.id, first.seq); n != 1 {
		t.Errorf("the first run changed: %d", n)
	}
	if n := countRows(t, db, `SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = 'check_runs_by_criterion'`); n != 1 {
		t.Errorf("index check_runs_by_criterion: %d", n)
	}
	// The migration holds the append-only triggers themselves, the same as
	// AppendOnlyTriggers makes (data-model §1, development §5).
	ddl, err := AppendOnlyTriggers("check_runs")
	if err != nil {
		t.Fatal(err)
	}
	for _, trigger := range []string{"check_runs_append_only_update", "check_runs_append_only_delete"} {
		var stored string
		if err := db.sql.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type = 'trigger' AND name = ?`, trigger).Scan(&stored); err != nil {
			t.Fatalf("trigger %s: %v", trigger, err)
		}
		if !strings.Contains(strings.Join(strings.Fields(ddl), " "), strings.Join(strings.Fields(stored), " ")+";") {
			t.Errorf("trigger %s is not the one AppendOnlyTriggers makes:\n%s", trigger, stored)
		}
	}
	ok("a run of project B", insertRun(runRow{id: ids.next(), seq: 900, project: projectB, criterion: critB, worktree: worktreeB}))

	_ = db.Close()
	writeTombstones(t, dir, projectA)
	db, _ = open(t, dir, Options{})
	if n := countRows(t, db, `SELECT count(*) FROM check_runs WHERE project_id = ?`, projectA); n != 0 {
		t.Errorf("check_runs has %d rows of the purged project", n)
	}
	if n := countRows(t, db, `SELECT count(*) FROM check_runs WHERE project_id = ?`, projectB); n != 1 {
		t.Errorf("project B has %d runs after A was purged, want 1", n)
	}
}

// WP2.5 Review 120 §4.1: SQLite passes a CHECK whose expression is NULL,
// so a CHECK written with = alone lets a pass with a NULL exit code
// through; 0009 writes every nullable comparison with IS / IS NOT, which
// is never NULL. The first half pins the SQLite behaviour the second half
// relies on.
func TestCheckRunsNullCannotSlipThrough(t *testing.T) {
	db, _ := open(t, testutil.TempHome(t), Options{})
	var eqNull, isZero, isNotZero, inNull int
	if err := db.sql.QueryRowContext(ctx, `SELECT (NULL = 0) IS NULL, NULL IS 0, NULL IS NOT 0, (NULL IN ('timeout')) IS NULL`).
		Scan(&eqNull, &isZero, &isNotZero, &inNull); err != nil {
		t.Fatal(err)
	}
	if eqNull != 1 || isZero != 0 || isNotZero != 1 || inNull != 1 {
		t.Fatalf("NULL = 0 is NULL: %d, NULL IS 0: %d, NULL IS NOT 0: %d, NULL IN (...) is NULL: %d", eqNull, isZero, isNotZero, inNull)
	}
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `CREATE TEMP TABLE naive (result TEXT NOT NULL, exit_code INTEGER, CHECK (result <> 'pass' OR exit_code = 0))`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO naive (result, exit_code) VALUES ('pass', NULL)`); err != nil {
			t.Errorf("the naive CHECK refused a NULL exit code: %v", err)
		}
		_, err := tx.ExecContext(ctx, `DROP TABLE naive`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var checks []string
	rows, err := db.sql.QueryContext(ctx, `SELECT sql FROM sqlite_master WHERE name = 'check_runs'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		checks = append(checks, s)
	}
	_ = rows.Close()
	if len(checks) != 1 {
		t.Fatalf("check_runs DDL: %d", len(checks))
	}
	for _, naive := range []string{"exit_code = 0", "exit_code <> 0", "exit_code != 0", "failure_kind = 'nonzero_exit'"} {
		if strings.Contains(checks[0], naive) {
			t.Errorf("check_runs compares a nullable column with %q", naive)
		}
	}
}

// 0009 adds a table and changes nothing that exists: an upgrade from v8
// keeps every row, backs the database up first, and leaves
// min_binary_version empty (development §5: an older binary that knows only
// v8 already refuses v9 by its schema version, so nothing more is needed).
func TestCheckRunsMigrationUpgradesV8(t *testing.T) {
	dir := testutil.TempHome(t)
	all, err := Migrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 9 || all[8].Name != "check_runs" {
		t.Fatalf("migrations = %d, last %q", len(all), all[len(all)-1].Name)
	}
	v8 := all[:8]
	db, _ := open(t, dir, Options{Migrations: v8})
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		for _, f := range []func() error{
			func() error { return insertProject(tx, projectA, "/r/a/.git", worktreeA, "/r/a") },
			func() error { return insertVersion(tx, versionA, 1, "human:me") },
			func() error { return insertRequirement(tx, reqA, "REQ-001", nil) },
			func() error { return insertCriterion(tx, criterion, "AC-001-1", reqA, "automated", "go test ./...") },
		} {
			if err := f(); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, db, `SELECT count(*) FROM sqlite_master WHERE name = 'check_runs'`); n != 0 {
		t.Fatalf("check_runs at v8: %d", n)
	}
	_ = db.Close()
	if n := len(backups(t, dir)); n != 0 {
		t.Fatalf("backups before the upgrade: %d", n)
	}

	db, res := open(t, dir, Options{})
	if v, _ := db.SchemaVersion(ctx); v != 9 {
		t.Fatalf("schema v%d after the upgrade, want v9", v)
	}
	if n := len(backups(t, dir)); n != 1 {
		t.Errorf("backups after the upgrade: %d, want 1 (%+v)", n, res)
	}
	var seqCounter int64
	var minBin string
	if err := db.sql.QueryRowContext(ctx, `SELECT seq_counter, min_binary_version FROM meta WHERE id = 1`).Scan(&seqCounter, &minBin); err != nil {
		t.Fatal(err)
	}
	if minBin != "" {
		t.Errorf("min_binary_version = %q after 0009, want empty", minBin)
	}
	for _, q := range []string{
		`SELECT count(*) FROM projects WHERE id = '` + projectA + `'`,
		`SELECT count(*) FROM worktrees WHERE id = '` + worktreeA + `'`,
		`SELECT count(*) FROM acceptance_criteria WHERE id = '` + criterion + `' AND revision = 1`,
	} {
		if n := countRows(t, db, q); n != 1 {
			t.Errorf("%s = %d after the upgrade", q, n)
		}
	}
	if n := countRows(t, db, `SELECT count(*) FROM check_runs`); n != 0 {
		t.Errorf("check_runs has %d rows after the upgrade", n)
	}
	if err := db.WriteTx(ctx, insertRun(runRow{id: "01J9Z3K5Q8R2T4V6W8X0Y2Z5A1", seq: seqCounter + 1})); err != nil {
		t.Fatalf("a run after the upgrade: %v", err)
	}
	_ = db.Close()

	before := testutil.SnapshotTree(t, dir.Path)
	openRefused(t, dir, Options{Migrations: v8}, "upgrade specmetry")
	testutil.AssertTreeUnchanged(t, dir.Path, before)
}
