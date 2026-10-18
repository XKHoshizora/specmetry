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
// rounds, each between a lower bound and a hard cap, and every helper
// process has a deadline after which it is killed and waited for) and
// prints one line per variant,
//
//	M14-RESULT {...}
//
// with the platform, the Go, SQLite and driver versions, the commit under
// test (M14_SHA), the bounds, the work actually done, and every outcome
// counted, zero failures included.
//
// A run is valid only if the experiment really took place (codex-review-051
// B1). A helper that did not start, crashed, printed a corrupt result or
// none, failed in its own setup or reached its deadline is a broken
// experiment: the test fails and records why. So does a run without the
// work it claims: no probe, or, during commits, no commit or no probe while
// a writer was committing. An error that a diagnosed call (Open, the probe,
// a write) returns is an observation: it is counted and does not fail the
// test. m14_validity_test.go checks both, in the normal test run.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
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

const (
	m14Project = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"
	// m14KillGrace is how long a helper may outlive its deadline before it
	// is killed; Wait gets as long again for its output.
	m14KillGrace = 3 * time.Second
)

func m14Enabled(t *testing.T) {
	t.Helper()
	if os.Getenv("SPECMETRY_M14") != "1" {
		t.Skip("an M-14 diagnostic: set SPECMETRY_M14=1 to run it (docs/m14-diagnostics.md)")
	}
}

// m14Int reads a bound from the environment, def when unset. Every bound
// is at least 1: a value outside [1, limit] stops the run before any work,
// so a run never does less, or other, work than it reports.
func m14Int(t *testing.T, name string, def, limit int) int {
	t.Helper()
	v := def
	if s := os.Getenv(name); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("%s=%q is not an integer", name, s)
		}
		v = n
	}
	if v < 1 || v > limit {
		t.Fatalf("%s=%d is outside [1, %d]", name, v, limit)
	}
	return v
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

// m14Report prints a result line. Faults or injections that the validity
// regressions set are recorded in it, so such a run is never mistaken for
// a diagnosis.
func m14Report(t *testing.T, result map[string]any) {
	t.Helper()
	for _, name := range []string{"M14_FAULT", "M14_INJECT"} {
		if v := os.Getenv(name); v != "" {
			result[strings.ToLower(name)] = v
		}
	}
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
// Open gives each step (store.go), and only as far as they tell: "cannot
// read the database version" alone may come from the read-only probe's file
// checks or from the read-write connection (codex-review-051 N3).
func m14OpenStage(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "cannot read the database version: cannot read the database:"):
		return "open: read-only version probe"
	case strings.Contains(msg, "cannot confirm that this specmetry binary may open the database"):
		return "open: read-only probe, uninspectable"
	case strings.Contains(msg, "cannot read the database version"):
		return "open: version read, step unknown"
	default:
		return "open: other"
	}
}

// m14Outcome is what a helper process reports, as one JSON line on stdout.
// Times are Unix milliseconds of the machine's clock, which all the
// processes share.
type m14Outcome struct {
	Helper     string         `json:"helper"`
	Stage      string         `json:"stage"`
	OpenCalled bool           `json:"open_called"`
	Code       int            `json:"code,omitempty"`
	ErrorCode  string         `json:"error_code,omitempty"`
	Error      string         `json:"error,omitempty"`
	Committed  int            `json:"committed"`
	Probes     int            `json:"probes"` // first attempts; the retries are counted apart
	Outcomes   map[string]int `json:"outcomes,omitempty"`
	RetryOK    int            `json:"retry_ok,omitempty"`
	RetryFail  int            `json:"retry_failed,omitempty"`
	AtMillis   []int64        `json:"at_ms,omitempty"`
	StartMs    int64          `json:"start_ms,omitempty"` // before the first write or probe
	EndMs      int64          `json:"end_ms,omitempty"`   // after the last commit or probe
}

// m14Print writes a helper's outcome; the helper exits 0 whatever it found,
// so that a crash is told apart from a failure it reports.
func m14Print(o m14Outcome) {
	b, _ := json.Marshal(o)
	fmt.Println(string(b))
}

func m14UnixMilli(name string) (time.Time, error) {
	ms, err := strconv.ParseInt(os.Getenv(name), 10, 64)
	if err != nil || ms <= 0 {
		return time.Time{}, fmt.Errorf("%s is not set to a time", name)
	}
	return time.UnixMilli(ms), nil
}

// m14HelperContext ends at the helper's deadline, M14_DEADLINE, which the
// test also enforces by killing the helper.
func m14HelperContext() (context.Context, context.CancelFunc, error) {
	deadline, err := m14UnixMilli("M14_DEADLINE")
	if err != nil {
		return nil, nil, err
	}
	c, cancel := context.WithDeadline(context.Background(), deadline)
	return c, cancel, nil
}

// m14OpenHelper runs Open as every specmetry process does (the data
// directory check, the open lock, the read-only version probe, the
// read-write connection), then writes M14_FACTS facts, or, when it is 0,
// facts until M14_UNTIL (at least one attempt). It reports the step that
// failed, with its codes.
func m14OpenHelper() int {
	o := m14Outcome{Helper: "m14open"}
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
	hctx, cancel, err := m14HelperContext()
	if err != nil {
		o.Stage, o.Error = "setup", err.Error()
		m14Print(o)
		return 0
	}
	defer cancel()
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
	facts, err := strconv.Atoi(os.Getenv("M14_FACTS"))
	var until time.Time
	if err == nil && facts == 0 {
		until, err = m14UnixMilli("M14_UNTIL")
	}
	if err != nil || facts < 0 {
		o.Stage, o.Error = "setup", "M14_FACTS or M14_UNTIL is not usable"
		m14Print(o)
		return 0
	}
	o.OpenCalled = true
	db, _, err := Open(hctx, dir, Options{Migrations: migs})
	if err != nil {
		if hctx.Err() != nil {
			o.Stage, o.Error = "deadline", "in Open: "+err.Error()
		} else {
			o.Stage, o.Code, o.ErrorCode, o.Error = m14OpenStage(err), m14Code(err), string(apperr.From(err).Code), err.Error()
		}
		m14Print(o)
		return 0
	}
	defer func() { _ = db.Close() }()
	o.StartMs = time.Now().UnixMilli()
	for i := 0; facts > 0 && i < facts || facts == 0 && (i == 0 || time.Now().Before(until)); i++ {
		if _, err := writeFact(hctx, db, m14Project, "m14"); err != nil {
			if hctx.Err() != nil {
				o.Stage, o.Error = "deadline", "in a write: "+err.Error()
			} else {
				o.Stage, o.Code, o.ErrorCode, o.Error = "write", m14Code(err), string(apperr.From(err).Code), err.Error()
			}
			m14Print(o)
			return 0
		}
		o.Committed++
		o.EndMs = time.Now().UnixMilli()
	}
	o.Stage = "done"
	m14Print(o)
	return 0
}

// m14ProbeHelper runs Open's read-only version probe (readMetaReadOnly, as
// Open calls it) in a loop until M14_UNTIL (at least once), outside Open:
// without the data directory check and without the open lock. After a
// SQLite error it probes once more at once.
func m14ProbeHelper() int {
	o := m14Outcome{Helper: "m14probe", Stage: "probe", Outcomes: map[string]int{}}
	hctx, cancel, err := m14HelperContext()
	if err != nil {
		o.Stage, o.Error = "setup", err.Error()
		m14Print(o)
		return 0
	}
	defer cancel()
	until, err := m14UnixMilli("M14_UNTIL")
	if err != nil {
		o.Stage, o.Error = "setup", err.Error()
		m14Print(o)
		return 0
	}
	dir, err := datadir.Resolve()
	if err != nil {
		o.Stage, o.Error = "resolve", err.Error()
		m14Print(o)
		return 0
	}
	dbPath := dir.Join(datadir.DatabaseFile)
	o.StartMs = time.Now().UnixMilli()
	for first := true; first || time.Now().Before(until); first = false {
		_, err := readMetaReadOnly(hctx, dbPath, false)
		if err != nil && hctx.Err() != nil {
			o.Stage, o.Error = "deadline", "in a probe: "+err.Error()
			break
		}
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
				o.AtMillis = append(o.AtMillis, time.Now().UnixMilli())
			}
			if _, again := readMetaReadOnly(hctx, dbPath, false); again == nil {
				o.RetryOK++
			} else {
				o.RetryFail++
			}
		default:
			// An error of the probe that is not SQLite's: an observation too.
			o.Outcomes["other"]++
			o.Error = err.Error()
		}
		o.EndMs = time.Now().UnixMilli()
	}
	m14Print(o)
	return 0
}

// m14Run is a helper process as the test saw it. Invalid says why its
// result cannot be used ("" when it can); then Outcome means nothing.
type m14Run struct {
	Started bool
	Outcome m14Outcome
	Invalid string
}

// m14Spawn starts the helper commands at once, each with deadline: a helper
// stops by itself at its deadline, and one still running m14KillGrace later
// is killed. Every process is waited for before m14Spawn returns.
func m14Spawn(t *testing.T, cmds []*exec.Cmd, deadline time.Time) []m14Run {
	t.Helper()
	out := make([]m14Run, len(cmds))
	var wg sync.WaitGroup
	for i, h := range cmds {
		hctx, cancel := context.WithDeadline(context.Background(), deadline.Add(m14KillGrace))
		cmd := exec.CommandContext(hctx, h.Path, h.Args[1:]...)
		env := slices.Clone(h.Env)
		env = append(env, "M14_DEADLINE="+strconv.FormatInt(deadline.UnixMilli(), 10))
		cmd.Env = env
		cmd.WaitDelay = m14KillGrace
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()
			if err := cmd.Start(); err != nil {
				out[i].Invalid = "did not start: " + err.Error()
				return
			}
			out[i].Started = true
			err := cmd.Wait()
			switch {
			case hctx.Err() != nil:
				out[i].Invalid = fmt.Sprintf("killed: still running %v after its deadline", m14KillGrace)
			case err != nil:
				out[i].Invalid = fmt.Sprintf("exited abnormally (%v)%s", err, m14Tail(stderr.String()))
			default:
				out[i].Outcome, out[i].Invalid = m14Parse(stdout.Bytes())
			}
		}()
	}
	wg.Wait()
	return out
}

func m14Tail(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) > 300 {
		s = "..." + s[len(s)-300:]
	}
	return ": " + s
}

// m14Parse reads a helper's stdout: exactly one JSON outcome, with no field
// it does not know.
func m14Parse(stdout []byte) (m14Outcome, string) {
	var o m14Outcome
	line := bytes.TrimSpace(stdout)
	if len(line) == 0 {
		return o, "printed no result"
	}
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.DisallowUnknownFields()
	if bytes.ContainsRune(line, '\n') || dec.Decode(&o) != nil || dec.More() {
		if len(line) > 80 {
			line = line[:80]
		}
		return m14Outcome{}, fmt.Sprintf("printed a malformed result %q", line)
	}
	return o, ""
}

// m14CheckOpen decides whether an m14open helper's result is an observation
// of Open: facts is what it was to write, 0 for writes until a time. A
// setup failure, a deadline, or a report that does not add up is not.
func m14CheckOpen(r *m14Run, facts int) {
	if r.Invalid != "" {
		return
	}
	o := r.Outcome
	switch {
	case o.Helper != "m14open":
		r.Invalid = fmt.Sprintf("reported as helper %q", o.Helper)
	case o.Stage == "setup" || o.Stage == "resolve" || o.Stage == "migrations":
		r.Invalid = fmt.Sprintf("failed in its setup, before Open (%s): %s", o.Stage, o.Error)
	case o.Stage == "deadline":
		r.Invalid = "reached its deadline " + o.Error
	case !o.OpenCalled:
		r.Invalid = "reported stage " + o.Stage + " without calling Open"
	case strings.HasPrefix(o.Stage, "open: "):
		if o.Committed != 0 {
			r.Invalid = "reported commits after a failed Open"
		}
	case o.Stage == "write":
		if facts > 0 && o.Committed >= facts {
			r.Invalid = "reported a failed write after all its facts"
		}
	case o.Stage == "done":
		if facts > 0 && o.Committed != facts || facts == 0 && o.Committed == 0 {
			r.Invalid = fmt.Sprintf("reported done after %d commits", o.Committed)
		}
	default:
		r.Invalid = "reported an unknown stage " + o.Stage
	}
}

// m14CheckProbe decides whether an m14probe helper's result is usable: it
// probed at least once and its counts add up.
func m14CheckProbe(r *m14Run) {
	if r.Invalid != "" {
		return
	}
	o := r.Outcome
	sum := 0
	for _, v := range o.Outcomes {
		sum += v
	}
	switch {
	case o.Helper != "m14probe":
		r.Invalid = fmt.Sprintf("reported as helper %q", o.Helper)
	case o.Stage == "setup" || o.Stage == "resolve":
		r.Invalid = fmt.Sprintf("failed in its setup (%s): %s", o.Stage, o.Error)
	case o.Stage == "deadline":
		r.Invalid = "reached its deadline " + o.Error
	case o.Stage != "probe":
		r.Invalid = "reported an unknown stage " + o.Stage
	case o.Probes < 1:
		r.Invalid = "made no probe"
	case sum != o.Probes || o.StartMs == 0 || o.EndMs < o.StartMs:
		r.Invalid = "reported counts or times that do not add up"
	}
}

// m14Union merges time spans [start, end] into disjoint ones.
func m14Union(spans [][2]int64) [][2]int64 {
	s := slices.Clone(spans)
	sort.Slice(s, func(i, j int) bool { return s[i][0] < s[j][0] })
	var out [][2]int64
	for _, x := range s {
		if n := len(out); n > 0 && x[0] <= out[n-1][1] {
			out[n-1][1] = max(out[n-1][1], x[1])
			continue
		}
		out = append(out, x)
	}
	return out
}

// m14Overlap is how many milliseconds at least one writer was between its
// first and its last commit while at least one prober was between its first
// and its last probe. It is evidence that probes and commits overlapped,
// not that every writer committed throughout.
func m14Overlap(writers, probers [][2]int64) int64 {
	var total int64
	for _, w := range m14Union(writers) {
		for _, p := range m14Union(probers) {
			if lo, hi := max(w[0], p[0]), min(w[1], p[1]); hi > lo {
				total += hi - lo
			}
		}
	}
	return total
}

// m14Relative gives spans in milliseconds since start.
func m14Relative(spans [][2]int64, start time.Time) [][2]int64 {
	out := make([][2]int64, len(spans))
	for i, s := range spans {
		out[i] = [2]int64{s[0] - start.UnixMilli(), s[1] - start.UnixMilli()}
	}
	return out
}

// m14Check checks the database after a round: SQLite's integrity check,
// the committed facts all there, their seqs unique and exactly 1 to n
// (codex-review-051 N1), and the seq counter at the last one. A failed
// helper must have left nothing more than the facts it reported committed.
// It checks the database, not whether the experiment was valid.
func m14Check(t *testing.T, db *DB, committed int) []string {
	t.Helper()
	var problems []string
	var integrity string
	if err := db.sql.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		problems = append(problems, fmt.Sprintf("integrity_check: %q %v", integrity, err))
	}
	var n, distinct, minSeq, maxSeq, counter int
	if err := db.sql.QueryRowContext(ctx, `SELECT count(*), count(DISTINCT seq), coalesce(min(seq), 0), coalesce(max(seq), 0) FROM facts`).Scan(&n, &distinct, &minSeq, &maxSeq); err != nil {
		return append(problems, err.Error())
	}
	if err := db.sql.QueryRowContext(ctx, `SELECT seq_counter FROM meta`).Scan(&counter); err != nil {
		return append(problems, err.Error())
	}
	if n != committed || distinct != n || counter != n || n > 0 && (minSeq != 1 || maxSeq != n) {
		problems = append(problems, fmt.Sprintf("rows=%d distinct=%d min=%d max=%d counter=%d, committed=%d", n, distinct, minSeq, maxSeq, counter, committed))
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

// m14Invalid fails the test for each reason the experiment is not valid,
// and returns the reasons for the result line (at most 20, with the count).
func m14Invalid(t *testing.T, invalid []string) map[string]any {
	t.Helper()
	for i, s := range invalid {
		if i == 20 {
			t.Errorf("invalid experiment: %d more", len(invalid)-20)
			break
		}
		t.Errorf("invalid experiment: %s", s)
	}
	return map[string]any{"valid": len(invalid) == 0, "invalid": invalid[:min(len(invalid), 20)], "invalid_count": len(invalid)}
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
			start := time.Now()
			until := start.Add(time.Duration(seconds) * time.Second)
			// A writer's Open may wait up to 30 s for the open lock.
			deadline := until.Add(60 * time.Second)
			untilEnv := "M14_UNTIL=" + strconv.FormatInt(until.UnixMilli(), 10)
			var cmds []*exec.Cmd
			for range variant.writers {
				cmds = append(cmds, testutil.HelperCommand(t, "m14open", untilEnv, "M14_FACTS=0"))
			}
			for range probers {
				cmds = append(cmds, testutil.HelperCommand(t, "m14probe", untilEnv))
			}
			runs := m14Spawn(t, cmds, deadline)
			var invalid []string
			started, probes, retryOK, retryFail, committed := 0, 0, 0, 0, 0
			counts, writerFailures := map[string]int{}, map[string]int{}
			var at []int64
			var writerSpans, proberSpans [][2]int64
			for i, r := range runs {
				if r.Started {
					started++
				}
				if i < variant.writers {
					m14CheckOpen(&r, 0)
					if r.Invalid != "" {
						invalid = append(invalid, fmt.Sprintf("writer %d: %s", i, r.Invalid))
						continue
					}
					o := r.Outcome
					committed += o.Committed
					if o.Committed > 0 {
						writerSpans = append(writerSpans, [2]int64{o.StartMs, o.EndMs})
					}
					if o.Stage != "done" {
						writerFailures[fmt.Sprintf("%s (code %d): %s", o.Stage, o.Code, o.Error)]++
					}
					continue
				}
				m14CheckProbe(&r)
				if r.Invalid != "" {
					invalid = append(invalid, fmt.Sprintf("prober %d: %s", i-variant.writers, r.Invalid))
					continue
				}
				o := r.Outcome
				probes, retryOK, retryFail = probes+o.Probes, retryOK+o.RetryOK, retryFail+o.RetryFail
				for k, v := range o.Outcomes {
					counts[k] += v
				}
				for _, ms := range o.AtMillis {
					at = append(at, ms-start.UnixMilli())
				}
				proberSpans = append(proberSpans, [2]int64{o.StartMs, o.EndMs})
			}
			overlap := m14Overlap(writerSpans, proberSpans)
			if variant.writers > 0 {
				switch {
				case committed == 0:
					invalid = append(invalid, "no writer committed: there were no commits to probe during")
				case overlap == 0:
					invalid = append(invalid, "no probe ran while a writer was committing")
				}
			}
			problems := m14Check(t, db, committed)
			for _, p := range problems {
				t.Errorf("database after the run: %s", p)
			}
			result := m14Invalid(t, invalid)
			for k, v := range map[string]any{
				"test": "read-only probe", "variant": variant.name, "platform": platform,
				"loop_seconds": seconds, "deadline_seconds": int(deadline.Sub(start).Seconds()),
				"writers": variant.writers, "probers": probers,
				"planned_helpers": len(cmds), "started_helpers": started,
				"probes": probes, "retries": retryOK + retryFail, "probe_calls": probes + retryOK + retryFail,
				"outcomes": counts, "retry_ok": retryOK, "retry_failed": retryFail, "at_ms": at,
				"writer_commits": committed, "writer_failures": m14Sorted(writerFailures),
				"writer_spans_ms": m14Relative(writerSpans, start), "prober_spans_ms": m14Relative(proberSpans, start),
				"overlap_ms": overlap, "database_problems": problems,
			} {
				result[k] = v
			}
			m14Report(t, result)
		})
	}
}

// Open as it runs, with its real locks and order: each round, a fresh
// database that the test process opens and keeps open, and M14_PROCS helper
// processes that each Open it and write M14_FACTS facts, all started at
// once: TestSeqIsUniqueAcrossProcesses (3 helpers, 100 facts each) by
// default, repeated M14_ROUNDS times. Each round's helpers have
// M14_ROUND_TIMEOUT seconds.
func TestM14OpenPath(t *testing.T) {
	m14Enabled(t)
	rounds := m14Int(t, "M14_ROUNDS", 30, 1000)
	procs := m14Int(t, "M14_PROCS", 3, 16)
	facts := m14Int(t, "M14_FACTS", 100, 1000)
	roundTimeout := m14Int(t, "M14_ROUND_TIMEOUT", 120, 600)
	base := testdir.New(t)
	var platform m14Platform
	failures := map[string]int{}
	var invalid, problems []string
	planned, started, openCalls, opened, committed, failedRounds, failedHelpers := 0, 0, 0, 0, 0, 0, 0
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
		roundCommitted, roundFailed, roundValid := 0, false, true
		for i, run := range m14Spawn(t, cmds, time.Now().Add(time.Duration(roundTimeout)*time.Second)) {
			planned++
			if run.Started {
				started++
			}
			m14CheckOpen(&run, facts)
			if run.Invalid != "" {
				roundValid = false
				invalid = append(invalid, fmt.Sprintf("round %d, helper %d: %s", r, i, run.Invalid))
				continue
			}
			o := run.Outcome
			openCalls++
			roundCommitted += o.Committed
			if o.Stage == "done" || o.Stage == "write" {
				opened++
			}
			if o.Stage != "done" {
				roundFailed = true
				failedHelpers++
				failures[fmt.Sprintf("%s, code %d, %s: %s", o.Stage, o.Code, o.ErrorCode, o.Error)]++
			}
		}
		committed += roundCommitted
		if roundFailed {
			failedRounds++
		}
		// What a helper with an unusable result committed is unknown, so its
		// round's database cannot be checked against the reports.
		if roundValid {
			for _, p := range m14Check(t, db, roundCommitted) {
				problems = append(problems, fmt.Sprintf("round %d: %s", r, p))
				t.Errorf("round %d, database after the round: %s", r, p)
			}
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}
	result := m14Invalid(t, invalid)
	for k, v := range map[string]any{
		"test": "open path", "platform": platform, "rounds": rounds, "procs": procs, "facts": facts, "round_timeout_seconds": roundTimeout,
		"planned_helpers": planned, "started_helpers": started, "open_calls": openCalls, "opens_succeeded": opened,
		"facts_committed": committed, "failed_helpers": failedHelpers, "failed_rounds": failedRounds,
		"failures": m14Sorted(failures), "database_problems": problems,
	} {
		result[k] = v
	}
	m14Report(t, result)
}
