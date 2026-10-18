package sqlite

// Diagnostics for maintenance item M-14 (docs/m14-diagnostics.md). On macOS
// CI, a helper of TestSeqIsUniqueAcrossProcesses once failed in Open with
// "cannot read the database version: cannot read the database: attempt to
// write a readonly database (264)". These tests look for that failure and
// count what happens; they change no production code and decide nothing
// about a fix.
//
// They are off by default: each one skips unless SPECMETRY_M14=1, so the
// normal test run and CI never run them. Every run is bounded (seconds or
// rounds, each with a hard cap) and prints one line per variant,
//
//	M14-RESULT {...}
//
// with the platform, the Go, SQLite and driver versions, the commit under
// test (M14_SHA), the bounds, and every outcome counted, zero failures
// included.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	sqlitedrv "modernc.org/sqlite"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

const m14Project = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"

func m14Enabled(t *testing.T) {
	t.Helper()
	if os.Getenv("SPECMETRY_M14") != "1" {
		t.Skip("an M-14 diagnostic: set SPECMETRY_M14=1 to run it (docs/m14-diagnostics.md)")
	}
}

// m14Int reads a bound from the environment: def when unset, never above
// limit (a run is always bounded).
func m14Int(t *testing.T, name string, def, limit int) int {
	t.Helper()
	v := def
	if s := os.Getenv(name); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			t.Fatalf("%s=%q is not a non-negative integer", name, s)
		}
		v = n
	}
	return min(v, limit)
}

// m14Platform is what a result line records about the run.
type m14Platform struct {
	OS, Arch, Go, SQLite, Driver, SHA, Runner string
}

func m14PlatformOf(t *testing.T, db *DB) m14Platform {
	t.Helper()
	p := m14Platform{OS: runtime.GOOS, Arch: runtime.GOARCH, Go: runtime.Version(), SHA: os.Getenv("M14_SHA"),
		Runner: strings.TrimSpace(os.Getenv("ImageOS") + " " + os.Getenv("ImageVersion"))}
	if p.SHA == "" {
		p.SHA = "unknown (set M14_SHA)"
	}
	if err := db.sql.QueryRowContext(ctx, `SELECT sqlite_version()`).Scan(&p.SQLite); err != nil {
		t.Fatal(err)
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, m := range info.Deps {
			if m.Path == "modernc.org/sqlite" {
				p.Driver = m.Path + " " + m.Version
			}
		}
	}
	return p
}

func m14Report(t *testing.T, result map[string]any) {
	t.Helper()
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("M14-RESULT %s", b)
}

// m14Code is the SQLite (extended) result code in err, 0 when there is none.
func m14Code(err error) int {
	var se *sqlitedrv.Error
	if errors.As(err, &se) {
		return se.Code()
	}
	return 0
}

// m14OpenStage names the step of Open that err comes from, by the messages
// Open gives each step (store.go).
func m14OpenStage(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "cannot read the database version: cannot read the database:"):
		return "open: read-only version probe"
	case strings.Contains(msg, "cannot confirm that this specmetry binary may open the database"):
		return "open: read-only probe, uninspectable"
	case strings.Contains(msg, "cannot read the database version"):
		return "open: read-write version read"
	default:
		return "open: other"
	}
}

// m14Outcome is what a helper process reports, as one JSON line on stdout.
type m14Outcome struct {
	Stage     string         `json:"stage"`
	Code      int            `json:"code,omitempty"`
	ErrorCode string         `json:"error_code,omitempty"`
	Error     string         `json:"error,omitempty"`
	Committed int            `json:"committed"`
	Probes    int            `json:"probes,omitempty"`
	Outcomes  map[string]int `json:"outcomes,omitempty"`
	RetryOK   int            `json:"retry_ok,omitempty"`
	RetryFail int            `json:"retry_failed,omitempty"`
	AtMillis  []int64        `json:"at_ms,omitempty"`
}

// m14Print writes a helper's outcome; the helper exits 0 whatever it found,
// so that a crash is told apart from a failure it reports.
func m14Print(o m14Outcome) {
	b, _ := json.Marshal(o)
	fmt.Println(string(b))
}

func m14Until() time.Time {
	ms, _ := strconv.ParseInt(os.Getenv("M14_UNTIL"), 10, 64)
	return time.UnixMilli(ms)
}

// m14OpenHelper runs Open as every specmetry process does (the data
// directory check, the open lock, the read-only version probe, the
// read-write connection), then writes M14_FACTS facts, or facts until
// M14_UNTIL. It reports the step that failed, with its codes.
func m14OpenHelper() int {
	var o m14Outcome
	// Faults for the validity regressions (m14_validity_test.go): a helper
	// that crashes, prints a corrupt result or none, fails in its own setup,
	// or hangs. The diagnostics workflow never sets M14_FAULT.
	switch os.Getenv("M14_FAULT") {
	case "crash":
		os.Exit(42)
	case "corrupt":
		fmt.Println("not-json")
		return 0
	case "silent":
		return 0
	case "setup":
		o.Stage, o.Error = "resolve", "an injected setup failure"
		m14Print(o)
		return 0
	case "hang":
		time.Sleep(20 * time.Second)
		return 0
	}
	dir, err := datadir.Resolve()
	if err != nil {
		o.Stage, o.Error = "resolve", err.Error()
		m14Print(o)
		return 0
	}
	migs, err := helperMigrations(false)
	if err != nil {
		o.Stage, o.Error = "migrations", err.Error()
		m14Print(o)
		return 0
	}
	db, _, err := Open(context.Background(), dir, Options{Migrations: migs})
	if err != nil {
		o.Stage, o.Code, o.ErrorCode, o.Error = m14OpenStage(err), m14Code(err), string(apperr.From(err).Code), err.Error()
		m14Print(o)
		return 0
	}
	defer func() { _ = db.Close() }()
	facts, _ := strconv.Atoi(os.Getenv("M14_FACTS"))
	until := m14Until()
	for i := 0; facts > 0 && i < facts || facts == 0 && time.Now().Before(until); i++ {
		if _, err := writeFact(context.Background(), db, m14Project, "m14"); err != nil {
			o.Stage, o.Code, o.ErrorCode, o.Error = "write", m14Code(err), string(apperr.From(err).Code), err.Error()
			m14Print(o)
			return 0
		}
		o.Committed++
	}
	o.Stage = "done"
	m14Print(o)
	return 0
}

// m14ProbeHelper runs Open's read-only version probe (readMetaReadOnly, as
// Open calls it) in a loop until M14_UNTIL, outside Open: without the data
// directory check and without the open lock. After a SQLite error it
// probes once more at once.
func m14ProbeHelper() int {
	o := m14Outcome{Stage: "probe", Outcomes: map[string]int{}}
	dir, err := datadir.Resolve()
	if err != nil {
		o.Stage, o.Error = "resolve", err.Error()
		m14Print(o)
		return 0
	}
	dbPath := dir.Join(datadir.DatabaseFile)
	start, until := time.Now(), m14Until()
	for time.Now().Before(until) {
		_, err := readMetaReadOnly(context.Background(), dbPath, false)
		o.Probes++
		var u *uninspectableError
		switch {
		case err == nil:
			o.Outcomes["ok"]++
		case errors.As(err, &u):
			o.Outcomes["uninspectable"]++
		case m14Code(err) != 0:
			o.Outcomes[fmt.Sprintf("sqlite %d", m14Code(err))]++
			if len(o.AtMillis) < 50 {
				o.AtMillis = append(o.AtMillis, time.Since(start).Milliseconds())
			}
			if _, again := readMetaReadOnly(context.Background(), dbPath, false); again == nil {
				o.RetryOK++
			} else {
				o.RetryFail++
			}
		default:
			o.Outcomes["other"]++
			o.Error = err.Error()
		}
	}
	m14Print(o)
	return 0
}

// m14Spawn starts helper processes at once and returns their outcomes.
func m14Spawn(t *testing.T, cmds []*exec.Cmd) []m14Outcome {
	t.Helper()
	out := make([]m14Outcome, len(cmds))
	var wg sync.WaitGroup
	for i, cmd := range cmds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, err := cmd.Output()
			line := strings.TrimSpace(string(b))
			if err != nil || json.Unmarshal([]byte(line), &out[i]) != nil {
				out[i] = m14Outcome{Stage: "helper crashed", Error: fmt.Sprintf("%v: %s", err, line)}
			}
		}()
	}
	wg.Wait()
	return out
}

// m14Check checks the database after a round: SQLite's integrity check,
// the committed facts all there, their seqs unique and contiguous from 1,
// and the seq counter at the last one. A failed helper must have left
// nothing more than the facts it reported committed.
func m14Check(t *testing.T, db *DB, committed int) []string {
	t.Helper()
	var problems []string
	var integrity string
	if err := db.sql.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		problems = append(problems, fmt.Sprintf("integrity_check: %q %v", integrity, err))
	}
	var n, distinct, maxSeq, counter int
	if err := db.sql.QueryRowContext(ctx, `SELECT count(*), count(DISTINCT seq), coalesce(max(seq), 0) FROM facts`).Scan(&n, &distinct, &maxSeq); err != nil {
		return append(problems, err.Error())
	}
	if err := db.sql.QueryRowContext(ctx, `SELECT seq_counter FROM meta`).Scan(&counter); err != nil {
		return append(problems, err.Error())
	}
	if n != committed || distinct != n || maxSeq != n || counter != n {
		problems = append(problems, fmt.Sprintf("rows=%d distinct=%d max=%d counter=%d, committed=%d", n, distinct, maxSeq, counter, committed))
	}
	return problems
}

func m14Sorted(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k, v := range m {
		keys = append(keys, fmt.Sprintf("%d x %s", v, k))
	}
	sort.Strings(keys)
	return keys
}

// The read-only version probe, outside Open: probers run it in a loop while
// writers commit, and the test process keeps the database open, as in
// TestSeqIsUniqueAcrossProcesses. The control has no writers. This is an
// auxiliary probe: it shows whether the probe itself can meet 264, not that
// Open does.
func TestM14ReadOnlyProbe(t *testing.T) {
	m14Enabled(t)
	seconds := m14Int(t, "M14_SECONDS", 20, 300)
	writers := m14Int(t, "M14_WRITERS", 3, 16)
	probers := m14Int(t, "M14_PROBERS", 3, 16)
	for _, variant := range []struct {
		name    string
		writers int
	}{{"control, no commits", 0}, {"during commits", writers}} {
		t.Run(variant.name, func(t *testing.T) {
			dir := testutil.TempHome(t)
			db, _ := open(t, dir, Options{Migrations: append(foundation(t), factsMigration(t))})
			platform := m14PlatformOf(t, db)
			until := strconv.FormatInt(time.Now().Add(time.Duration(seconds)*time.Second).UnixMilli(), 10)
			var cmds []*exec.Cmd
			for range variant.writers {
				cmds = append(cmds, testutil.HelperCommand(t, "m14open", "M14_UNTIL="+until, "M14_FACTS=0"))
			}
			for range probers {
				cmds = append(cmds, testutil.HelperCommand(t, "m14probe", "M14_UNTIL="+until))
			}
			outcomes := m14Spawn(t, cmds)
			probes, retryOK, retryFail, committed := 0, 0, 0, 0
			counts, writerFailures := map[string]int{}, map[string]int{}
			var at []int64
			for i, o := range outcomes {
				if i < variant.writers {
					committed += o.Committed
					if o.Stage != "done" {
						writerFailures[fmt.Sprintf("%s (code %d): %s", o.Stage, o.Code, o.Error)]++
					}
					continue
				}
				if o.Stage != "probe" {
					t.Errorf("prober %d: %s: %s", i, o.Stage, o.Error)
				}
				probes, retryOK, retryFail = probes+o.Probes, retryOK+o.RetryOK, retryFail+o.RetryFail
				for k, v := range o.Outcomes {
					counts[k] += v
				}
				at = append(at, o.AtMillis...)
			}
			problems := m14Check(t, db, committed)
			for _, p := range problems {
				t.Errorf("database after the run: %s", p)
			}
			m14Report(t, map[string]any{
				"test": "read-only probe", "variant": variant.name, "platform": platform,
				"seconds": seconds, "writers": variant.writers, "probers": probers,
				"probes": probes, "outcomes": counts, "retry_ok": retryOK, "retry_failed": retryFail, "at_ms": at,
				"writer_commits": committed, "writer_failures": m14Sorted(writerFailures), "database_problems": problems,
			})
		})
	}
}

// Open as it runs, with its real locks and order: each round, a fresh
// database that the test process opens and keeps open, and M14_PROCS helper
// processes that each Open it and write M14_FACTS facts, all started at
// once: TestSeqIsUniqueAcrossProcesses (3 helpers, 100 facts each) by
// default, repeated M14_ROUNDS times.
func TestM14OpenPath(t *testing.T) {
	m14Enabled(t)
	rounds := m14Int(t, "M14_ROUNDS", 30, 1000)
	procs := m14Int(t, "M14_PROCS", 3, 16)
	facts := max(1, m14Int(t, "M14_FACTS", 100, 1000))
	base := testdir.New(t)
	var platform m14Platform
	failures := map[string]int{}
	failedRounds, failedHelpers := 0, 0
	var problems []string
	for r := range rounds {
		home := filepath.Join(base, fmt.Sprintf("r%04d", r))
		db, _, err := Open(ctx, datadir.Dir{Path: home, FromEnv: true}, Options{Migrations: append(foundation(t), factsMigration(t))})
		if err != nil {
			t.Fatalf("round %d: the test process's Open: %v", r, err)
		}
		if r == 0 {
			platform = m14PlatformOf(t, db)
		}
		// A validity regression: every helper's writes fail with a real
		// SQLite error (SQLITE_CONSTRAINT_TRIGGER), after a real Open.
		if os.Getenv("M14_INJECT") == "reject-writes" {
			if _, err := db.sql.ExecContext(ctx, `CREATE TRIGGER m14_reject BEFORE INSERT ON facts BEGIN SELECT RAISE(ABORT, 'm14: an injected rejection'); END`); err != nil {
				t.Fatal(err)
			}
		}
		var cmds []*exec.Cmd
		for range procs {
			cmds = append(cmds, testutil.HelperCommand(t, "m14open", datadir.EnvHome+"="+home, "M14_FACTS="+strconv.Itoa(facts)))
		}
		committed, failed := 0, false
		for _, o := range m14Spawn(t, cmds) {
			committed += o.Committed
			if o.Stage != "done" {
				failed = true
				failedHelpers++
				failures[fmt.Sprintf("%s, code %d, %s: %s", o.Stage, o.Code, o.ErrorCode, o.Error)]++
			}
		}
		if failed {
			failedRounds++
		}
		for _, p := range m14Check(t, db, committed) {
			problems = append(problems, fmt.Sprintf("round %d: %s", r, p))
			t.Errorf("round %d, database after the round: %s", r, p)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}
	m14Report(t, map[string]any{
		"test": "open path", "platform": platform, "rounds": rounds, "procs": procs, "facts": facts,
		"helper_opens": rounds * procs, "failed_helpers": failedHelpers, "failed_rounds": failedRounds,
		"failures": m14Sorted(failures), "database_problems": problems,
	})
}
