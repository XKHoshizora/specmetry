package sqlite

// Validity regressions for the M-14 diagnostics (codex-review-051 B1, N1,
// N3). Unlike the diagnostics, they run in the normal test run. Each case
// starts one diagnostic in a child test process, with small bounds (a few
// seconds, not the workflow's stress), and checks two things: the verdict,
// and what the result line records. A broken experiment (a helper that
// crashes, prints a corrupt result or none, fails in its own setup or
// hangs, or bounds that make no work) must fail the diagnostic. A valid one
// must pass, including one whose real Open succeeds and whose writes all
// meet a real SQLite error. Every child has a deadline and is reaped.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// m14ChildRun is what one child diagnostic run gave.
type m14ChildRun struct {
	passed  bool
	output  string
	results []map[string]any
	took    time.Duration
}

// m14Child runs the diagnostic test in a child test process, enabled, with
// env added to an environment cleared of M14_* settings.
func m14Child(t *testing.T, test string, env ...string) m14ChildRun {
	t.Helper()
	cctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cctx, os.Args[0], "-test.run=^"+test+"$", "-test.count=1", "-test.v", "-test.timeout=150s")
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, "M14_") || name == "SPECMETRY_M14" || name == "SPECMETRY_TEST_HELPER" {
			continue
		}
		cmd.Env = append(cmd.Env, kv)
	}
	cmd.Env = append(cmd.Env, "SPECMETRY_M14=1", "M14_SHA=validity-regression")
	cmd.Env = append(cmd.Env, env...)
	cmd.WaitDelay = 10 * time.Second
	start := time.Now()
	out, err := cmd.CombinedOutput()
	r := m14ChildRun{passed: err == nil, output: string(out), took: time.Since(start)}
	if cctx.Err() != nil {
		t.Fatalf("the child run of %s did not end within its deadline:\n%s", test, out)
	}
	if !strings.Contains(r.output, "=== RUN   "+test) || strings.Contains(r.output, "--- SKIP: "+test) {
		t.Fatalf("the child did not run %s:\n%s", test, out)
	}
	for _, line := range strings.Split(r.output, "\n") {
		_, js, ok := strings.Cut(line, "M14-RESULT ")
		if !ok {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(js)), &m); err != nil {
			t.Fatalf("a result line is not JSON: %q", line)
		}
		r.results = append(r.results, m)
	}
	return r
}

// m14Field is a number or flag of a result line, as a string ("<absent>"
// when the line has no such field), so a check can name what it found.
func m14Field(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return "<absent>"
	}
	return fmt.Sprint(v)
}

func m14Label(m map[string]any) string {
	if v, ok := m["variant"]; ok {
		return fmt.Sprint(v)
	}
	return fmt.Sprint(m["test"])
}

func m14Want(t *testing.T, m map[string]any, key, want string) {
	t.Helper()
	if got := m14Field(m, key); got != want {
		t.Errorf("%s: %s = %s, want %s", m14Label(m), key, got, want)
	}
}

func m14WantPositive(t *testing.T, m map[string]any, key string) {
	t.Helper()
	if n, ok := m[key].(float64); !ok || n <= 0 {
		t.Errorf("%s: %s = %s, want a positive number", m14Label(m), key, m14Field(m, key))
	}
}

func m14Variant(t *testing.T, r m14ChildRun, variant string) map[string]any {
	t.Helper()
	for _, m := range r.results {
		if variant == "" || m["variant"] == variant {
			return m
		}
	}
	t.Fatalf("no result line for %q:\n%s", variant, r.output)
	return nil
}

// m14RecordsInvalid: the run's record says it was not valid, and why.
func m14RecordsInvalid(t *testing.T, r m14ChildRun) {
	t.Helper()
	for _, m := range r.results {
		if inv, ok := m["invalid"].([]any); ok && len(inv) > 0 && m["valid"] == false {
			return
		}
	}
	t.Errorf("no result line records the run as invalid, with the reasons:\n%s", r.output)
}

// m14RecordsBound: a bound outside its range stopped the run before any
// work, naming the variable.
func m14RecordsBound(name string) func(*testing.T, m14ChildRun) {
	return func(t *testing.T, r m14ChildRun) {
		t.Helper()
		if !strings.Contains(r.output, name+"=") || len(r.results) > 0 {
			t.Errorf("want a refusal that names %s and no result line:\n%s", name, r.output)
		}
	}
}

func TestValidityOfM14Diagnostics(t *testing.T) {
	const probeTest, openTest = "TestM14ReadOnlyProbe", "TestM14OpenPath"
	probe := []string{"M14_SECONDS=2", "M14_WRITERS=2", "M14_PROBERS=1"}
	openPath := []string{"M14_ROUNDS=1", "M14_PROCS=2", "M14_FACTS=5"}
	with := func(base []string, extra ...string) []string { return append(append([]string(nil), base...), extra...) }
	cases := []struct {
		name, test string
		env        []string
		pass       bool
		within     time.Duration // the run must end within it; 0: the child's deadline only
		records    func(*testing.T, m14ChildRun)
	}{
		// Valid experiments, run first and one at a time.
		{"read-only probe, valid", probeTest, []string{"M14_SECONDS=3", "M14_WRITERS=2", "M14_PROBERS=1"}, true, 0,
			func(t *testing.T, r m14ChildRun) {
				control, during := m14Variant(t, r, "control, no commits"), m14Variant(t, r, "during commits")
				for _, m := range []map[string]any{control, during} {
					m14WantPositive(t, m, "probes")
					m14Want(t, m, "valid", "true")
					m14Want(t, m, "started_helpers", m14Field(m, "planned_helpers"))
				}
				m14Want(t, control, "writer_commits", "0")
				m14WantPositive(t, during, "writer_commits")
				m14WantPositive(t, during, "overlap_ms")
			}},
		{"open path, valid", openTest, openPath, true, 0, func(t *testing.T, r m14ChildRun) {
			m := m14Variant(t, r, "")
			for key, want := range map[string]string{"planned_helpers": "2", "started_helpers": "2", "open_calls": "2",
				"opens_succeeded": "2", "facts_committed": "10", "failed_helpers": "0", "valid": "true"} {
				m14Want(t, m, key, want)
			}
		}},
		{"open path, every write meets a SQLite error", openTest, with(openPath, "M14_INJECT=reject-writes"), true, 0,
			func(t *testing.T, r m14ChildRun) {
				m := m14Variant(t, r, "")
				for key, want := range map[string]string{"planned_helpers": "2", "started_helpers": "2", "open_calls": "2",
					"opens_succeeded": "2", "facts_committed": "0", "failed_helpers": "2", "valid": "true"} {
					m14Want(t, m, key, want)
				}
				if f := fmt.Sprint(m["failures"]); !strings.Contains(f, "write, code 1811") {
					t.Errorf("failures %s, want writes failed with SQLITE_CONSTRAINT_TRIGGER (1811)", f)
				}
			}},
		// Broken experiments.
		{"read-only probe, the writers crash", probeTest, with(probe, "M14_FAULT=crash"), false, 0, m14RecordsInvalid},
		{"read-only probe, the writers print a corrupt result", probeTest, with(probe, "M14_FAULT=corrupt"), false, 0, m14RecordsInvalid},
		{"read-only probe, the writers print no result", probeTest, with(probe, "M14_FAULT=silent"), false, 0, m14RecordsInvalid},
		{"open path, the helpers crash", openTest, with(openPath, "M14_FAULT=crash"), false, 0, m14RecordsInvalid},
		{"open path, the helpers print a corrupt result", openTest, with(openPath, "M14_FAULT=corrupt"), false, 0, m14RecordsInvalid},
		{"open path, the helpers print no result", openTest, with(openPath, "M14_FAULT=silent"), false, 0, m14RecordsInvalid},
		{"open path, the helpers fail in their setup", openTest, with(openPath, "M14_FAULT=setup"), false, 0, m14RecordsInvalid},
		{"open path, the helpers hang", openTest, with(openPath, "M14_FAULT=hang", "M14_ROUND_TIMEOUT=2"), false, 15 * time.Second, m14RecordsInvalid},
		// A valid result with a stray closing token after it: the helpers
		// really Open and commit (codex-review-054 B1-R).
		{"open path, the helpers append ] to a valid result", openTest, with(openPath, "M14_FAULT=trailing-bracket"), false, 0, m14RecordsInvalid},
		{"open path, the helpers append } to a valid result", openTest, with(openPath, "M14_FAULT=trailing-brace"), false, 0, m14RecordsInvalid},
		{"read-only probe, the helpers append ] to a valid result", probeTest, with(probe, "M14_FAULT=trailing-bracket"), false, 0, m14RecordsInvalid},
		// Bounds that would make no work, or other work than reported.
		{"read-only probe, zero seconds", probeTest, []string{"M14_SECONDS=0", "M14_WRITERS=2", "M14_PROBERS=1"}, false, 0, m14RecordsBound("M14_SECONDS")},
		{"read-only probe, zero probers", probeTest, []string{"M14_SECONDS=2", "M14_WRITERS=2", "M14_PROBERS=0"}, false, 0, m14RecordsBound("M14_PROBERS")},
		{"read-only probe, zero writers", probeTest, []string{"M14_SECONDS=2", "M14_WRITERS=0", "M14_PROBERS=1"}, false, 0, m14RecordsBound("M14_WRITERS")},
		{"open path, zero rounds", openTest, []string{"M14_ROUNDS=0", "M14_PROCS=2", "M14_FACTS=5"}, false, 0, m14RecordsBound("M14_ROUNDS")},
		{"open path, zero helpers", openTest, []string{"M14_ROUNDS=1", "M14_PROCS=0", "M14_FACTS=5"}, false, 0, m14RecordsBound("M14_PROCS")},
		{"open path, zero facts", openTest, []string{"M14_ROUNDS=1", "M14_PROCS=2", "M14_FACTS=0"}, false, 0, m14RecordsBound("M14_FACTS")},
		{"open path, helpers above the limit", openTest, []string{"M14_ROUNDS=1", "M14_PROCS=17", "M14_FACTS=5"}, false, 0, m14RecordsBound("M14_PROCS")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !c.pass {
				t.Parallel()
			}
			r := m14Child(t, c.test, c.env...)
			t.Run("verdict", func(t *testing.T) {
				if r.passed != c.pass {
					t.Errorf("the diagnostic passed = %v, want %v:\n%s", r.passed, c.pass, r.output)
				}
				if c.within > 0 && r.took > c.within {
					t.Errorf("the run took %v, want at most %v", r.took.Round(time.Millisecond), c.within)
				}
			})
			t.Run("records", func(t *testing.T) { c.records(t, r) })
		})
	}
}

// codex-review-051 N1: a seq gap below the rows (here 0 and 2) is a problem,
// though the count, the distinct count, the maximum and the counter agree.
func TestValidityOfM14CheckSeesASeqGap(t *testing.T) {
	for _, c := range []struct {
		name    string
		seqs    []int
		counter int
		problem bool
	}{
		{"contiguous from 1", []int{1, 2}, 2, false},
		{"0 and 2", []int{0, 2}, 2, true},
		{"empty, the counter at 0", nil, 0, false},
		{"empty, the counter at 1", nil, 1, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			db, _ := open(t, testutil.TempHome(t), Options{Migrations: append(foundation(t), factsMigration(t))})
			for _, s := range c.seqs {
				if _, err := db.sql.ExecContext(ctx, `INSERT INTO facts (seq, project_id, payload) VALUES (?, ?, 'x')`, s, m14Project); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := db.sql.ExecContext(ctx, `UPDATE meta SET seq_counter = ?`, c.counter); err != nil {
				t.Fatal(err)
			}
			if got := m14Check(t, db, len(c.seqs)); (len(got) > 0) != c.problem {
				t.Errorf("problems %q, want a problem: %v", got, c.problem)
			}
		})
	}
}

// codex-review-051 N3: only the read-only probe's query error is placed in
// the probe; another "cannot read the database version" may come from the
// probe's file checks or from the read-write connection.
func TestValidityOfM14OpenStage(t *testing.T) {
	for _, c := range []struct {
		err  error
		want string
	}{
		{apperr.Wrap(apperr.Environment, fmt.Errorf("cannot read the database: %w", errors.New("attempt to write a readonly database (264)")), "cannot read the database version"),
			"open: read-only version probe"},
		{apperr.Newf(apperr.Environment, "cannot confirm that this specmetry binary may open the database without modifying it: %s", "a reason"),
			"open: read-only probe, uninspectable"},
		{apperr.Wrap(apperr.Environment, errors.New("stat specmetry.db: permission denied"), "cannot read the database version"),
			"open: version read, step unknown"},
		{errors.New("the open lock is held by another process"), "open: other"},
	} {
		if got := m14OpenStage(c.err); got != c.want {
			t.Errorf("%q: stage %q, want %q", c.err, got, c.want)
		}
	}
}

// The overlap that "during commits" needs as its evidence: the time at
// least one writer and at least one prober were both between their first
// and their last call.
func TestValidityOfM14Overlap(t *testing.T) {
	for _, c := range []struct {
		name             string
		writers, probers [][2]int64
		want             int64
	}{
		{"none", nil, [][2]int64{{0, 10}}, 0},
		{"disjoint", [][2]int64{{0, 10}}, [][2]int64{{10, 20}}, 0},
		{"inside", [][2]int64{{0, 100}}, [][2]int64{{10, 20}}, 10},
		{"writers merged, probers merged", [][2]int64{{0, 50}, {40, 100}}, [][2]int64{{30, 60}, {50, 120}}, 70},
		{"a gap between writers", [][2]int64{{0, 10}, {90, 100}}, [][2]int64{{0, 100}}, 20},
	} {
		if got := m14Overlap(c.writers, c.probers); got != c.want {
			t.Errorf("%s: overlap %d, want %d", c.name, got, c.want)
		}
	}
}

// codex-review-054 B1-R: a helper's stdout is accepted only if it is
// exactly one JSON value, with only white space around it. A valid result
// followed by anything else (a stray ']' or '}', another character, a
// truncated or a second value) is malformed.
func TestValidityOfM14Parse(t *testing.T) {
	const valid = `{"helper":"m14open","stage":"done","open_called":true,"committed":1}`
	for _, c := range []struct {
		name, stdout string
		ok           bool
	}{
		{"a valid result", valid, true},
		{"with the newline that Println writes", valid + "\n", true},
		{"with white space around it", " \t\n" + valid + " \t\r\n", true},
		{"followed by ]", valid + "]", false},
		{"followed by }", valid + "}", false},
		{"followed by }]", valid + "}]", false},
		{"followed by another character", valid + "x", false},
		{"followed by a comma", valid + ",", false},
		{"followed by a truncated value", valid + `{"helper"`, false},
		{"followed by a second value", valid + valid, false},
		{"followed by a second value after a space", valid + " " + valid, false},
		{"followed by a number", valid + " 1", false},
		{"truncated", valid[:len(valid)-5], false},
		{"with an unknown field", `{"helper":"m14open","stage":"done","open_called":true,"committed":1,"extra":1}`, false},
		{"empty", "", false},
		{"not JSON", "not-json", false},
	} {
		o, invalid := m14Parse([]byte(c.stdout))
		if got := invalid == ""; got != c.ok {
			t.Errorf("%s: accepted = %v (%q), want %v", c.name, got, invalid, c.ok)
			continue
		}
		if c.ok && (o.Helper != "m14open" || o.Stage != "done" || !o.OpenCalled || o.Committed != 1) {
			t.Errorf("%s: decoded %+v", c.name, o)
		}
	}
}
