// Package wp25 holds test-only prototypes of the WP2.5 check runner's
// teardown mechanisms (docs/wp2.5-prototype.md). Nothing here is product
// code and nothing imports it. It starts only its own helper processes
// (this test binary, run again). Each helper has a hard limit of its own,
// effective only once it runs Go code (not while it is created
// suspended); the scenario's own waits are bounded and never rely on it.
// Every scenario ends by checking that the processes it started or
// verified as its own are gone.
//
// Each scenario prints one WP25RESULT line. Its verdict is "confirmed" when
// the run showed what the candidate predicts, "refuted" when it showed
// something else (a finding, not a pass), "unsupported" when the mechanism
// is not available here, and "invalid" when the run cannot count either
// way: a helper that did not start, crashed, reached its hard limit or was
// not cleaned up, a report that is missing, corrupt or not what the
// scenario needs, a load that was not built, or no work done. The gate
// itself is tested in gate_test.go.
package wp25

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	helperEnv = "WP25_HELPER"
	limitEnv  = "WP25_LIMIT_MS"
	reportEnv = "WP25_REPORT"
	markerEnv = "WP25_MARKER"

	// limitExit is the exit code of a helper that reached its hard limit.
	limitExit = 97
	// helperLimit is every helper's hard limit; scenarios end their
	// helpers within seconds, so reaching it means the cleanup failed.
	helperLimit = 60 * time.Second
)

func TestMain(m *testing.M) {
	if mode := os.Getenv(helperEnv); mode != "" {
		os.Exit(runHelper(mode, os.Args[1:]))
	}
	meta := map[string]string{
		"sha":      envOr("WP25_SHA", "local"),
		"platform": runtime.GOOS + "/" + runtime.GOARCH,
		"go":       runtime.Version(),
		"os":       osVersion(),
		"started":  time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(meta)
	fmt.Println("WP25META " + string(b))
	code := m.Run()
	os.Exit(summarize(code))
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type kind string

const (
	normal  kind = "normal"
	counter kind = "counterexample"
	fault   kind = "fault-injection"
	refusal kind = "refusal"
)

type verdict string

const (
	confirmed   verdict = "confirmed"
	refuted     verdict = "refuted"
	unsupported verdict = "unsupported"
	invalid     verdict = "invalid"
)

type result struct {
	ID       string  `json:"id"`
	Gate     string  `json:"gate"`
	Kind     kind    `json:"kind"`
	Platform string  `json:"platform"`
	Verdict  verdict `json:"verdict"`
	Expected string  `json:"expected"`
	Observed string  `json:"observed"`
	Launches int     `json:"launches"`
	Work     int64   `json:"work"`
	Cleanup  string  `json:"cleanup"`
	Reason   string  `json:"reason,omitempty"`
	Millis   int64   `json:"ms"`
	Owned    []int   `json:"owned,omitempty"`   // PIDs of the processes the scenario owned
	Missing  int     `json:"missing,omitempty"` // expected processes it could not account for
}

var (
	resultsMu sync.Mutex
	results   []result
)

// owned is a process a scenario started, or one its own helper reported
// and the scenario verified as its own (a descendant of the process it
// started, or in its group or Job): a reported PID alone is not enough.
// Its identity is checked again before every action.
type owned interface {
	String() string
	// ended reports whether the process has reliably ended; a zombie that
	// is not this process's child counts as ended. A state that cannot be
	// read is not an end.
	ended() bool
	// stop kills the process while its identity still holds, and reaps
	// it if it is this process's child.
	stop()
	// release gives up what observes it (a Windows handle); after ended.
	release()
	// pidOf is its PID, 0 if it never started.
	pidOf() int
}

type scen struct {
	t          *testing.T
	r          result
	begin      time.Time
	needLaunch bool
	needWork   bool
	want, got  []string
	mismatch   []string
	invalid    []string
	unsup      string
	procs      []owned
	reports    []reportFile
	after      []func()
	goid       uint64     // the test goroutine
	mu         sync.Mutex // the scenario may be used from several goroutines
}

// begin starts a scenario; the caller defers done. needLaunch and
// needWork say whether a run without launches or without work is invalid.
func begin(t *testing.T, gate, id string, k kind, needLaunch, needWork bool) *scen {
	return &scen{
		t:          t,
		r:          result{ID: id, Gate: gate, Kind: k, Platform: runtime.GOOS + "/" + runtime.GOARCH},
		begin:      time.Now(),
		needLaunch: needLaunch,
		needWork:   needWork,
		goid:       goid(),
	}
}

// goid gives the current goroutine's ID, read from its stack header.
func goid() uint64 {
	var b [64]byte
	n := runtime.Stack(b[:], false)
	f := strings.Fields(strings.TrimPrefix(string(b[:n]), "goroutine "))
	if len(f) == 0 {
		return 0
	}
	id, _ := strconv.ParseUint(f[0], 10, 64)
	return id
}

func (s *scen) locked(f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f()
}

func (s *scen) launched(n int)  { s.locked(func() { s.r.Launches += n }) }
func (s *scen) worked(n int64)  { s.locked(func() { s.r.Work += n }) }
func (s *scen) own(p owned)     { s.locked(func() { s.procs = append(s.procs, p) }) }
func (s *scen) defer_(f func()) { s.locked(func() { s.after = append(s.after, f) }) }

func (s *scen) note(f string, a ...any) {
	s.locked(func() { s.got = append(s.got, fmt.Sprintf(f, a...)) })
}

// expect records one prediction and what the run showed.
func (s *scen) expect(what string, want, got any) {
	w, g := fmt.Sprint(want), fmt.Sprint(got)
	s.locked(func() {
		s.want = append(s.want, what+"="+w)
		s.got = append(s.got, what+"="+g)
		if w != g {
			s.mismatch = append(s.mismatch, what)
		}
	})
}

// expectTrue records a prediction that holds when ok is true.
func (s *scen) expectTrue(what string, ok bool, detail string) {
	s.locked(func() {
		s.want = append(s.want, what)
		if detail != "" {
			s.got = append(s.got, what+": "+detail)
		} else {
			s.got = append(s.got, fmt.Sprintf("%s: %v", what, ok))
		}
		if !ok {
			s.mismatch = append(s.mismatch, what)
		}
	})
}

// invalidf marks the run as not counting and stops it: the test on the
// test goroutine, only the calling goroutine elsewhere.
func (s *scen) invalidf(f string, a ...any) {
	s.locked(func() { s.invalid = append(s.invalid, fmt.Sprintf(f, a...)) })
	if goid() == s.goid {
		s.t.FailNow()
	}
	s.t.Fail()
	runtime.Goexit()
}

// unsupported marks the mechanism as unavailable here and stops the run.
func (s *scen) unsupported(f string, a ...any) {
	s.unsup = fmt.Sprintf(f, a...)
	s.t.SkipNow()
}

// needLoad stops the scenario as invalid unless the load it needs was
// built: the roles, numbers and stages its prediction is about. A
// different result under the full load is refuted instead.
func (s *scen) needLoad(ok bool, f string, a ...any) {
	if !ok {
		s.invalidf("load not established: "+f, a...)
	}
	s.note("load: "+f, a...)
}

func (s *scen) done() {
	for i := len(s.after) - 1; i >= 0; i-- {
		s.after[i]()
	}
	s.r.Cleanup = s.cleanup()
	// The writers have ended (or the cleanup says they have not): every
	// report must now hold exactly what its scenario declared.
	for _, rf := range s.reports {
		if err := checkFinal(rf); err != nil {
			s.invalid = append(s.invalid, "report: "+err.Error())
		}
	}
	for _, p := range s.procs {
		p.release()
		if pid := p.pidOf(); pid > 0 {
			s.r.Owned = append(s.r.Owned, pid)
		}
	}
	switch {
	case len(s.invalid) > 0:
		s.r.Verdict, s.r.Reason = invalid, strings.Join(s.invalid, "; ")
	case s.unsup != "":
		s.r.Verdict, s.r.Reason = unsupported, s.unsup
	case s.needLaunch && s.r.Launches == 0:
		s.r.Verdict, s.r.Reason = invalid, "no process was launched"
	case s.needWork && s.r.Work == 0:
		s.r.Verdict, s.r.Reason = invalid, "no work was done"
	case s.t.Failed():
		s.r.Verdict, s.r.Reason = invalid, "the test failed before its predictions were checked"
	case len(s.want) == 0:
		s.r.Verdict, s.r.Reason = invalid, "no prediction was checked"
	case len(s.mismatch) > 0:
		s.r.Verdict, s.r.Reason = refuted, "differs: "+strings.Join(s.mismatch, ", ")
	default:
		s.r.Verdict = confirmed
	}
	s.r.Expected = strings.Join(s.want, "; ")
	s.r.Observed = strings.Join(s.got, "; ")
	s.r.Millis = time.Since(s.begin).Milliseconds()
	b, _ := json.Marshal(s.r)
	fmt.Println("WP25RESULT " + string(b))
	resultsMu.Lock()
	results = append(results, s.r)
	resultsMu.Unlock()
	switch s.r.Verdict {
	case refuted, invalid:
		if !s.t.Failed() {
			s.t.Errorf("%s: %s: %s", s.r.ID, s.r.Verdict, s.r.Reason)
		}
	}
}

// cleanup ends what is left of the scenario's processes and says whether
// all of them are confirmed gone.
func (s *scen) cleanup() string {
	if len(s.procs) == 0 {
		return "no processes"
	}
	for _, p := range s.procs {
		if !p.ended() {
			p.stop()
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		var left []string
		for _, p := range s.procs {
			if !p.ended() {
				left = append(left, p.String())
			}
		}
		if len(left) == 0 {
			names := make([]string, 0, len(s.procs))
			for _, p := range s.procs {
				names = append(names, p.String())
			}
			return fmt.Sprintf("verified: %d processes gone (%s)", len(s.procs), strings.Join(names, ", "))
		}
		if time.Now().After(deadline) {
			s.invalid = append(s.invalid, "cleanup not verified: "+strings.Join(left, ", "))
			return "NOT verified: " + strings.Join(left, ", ") + " left"
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func summarize(code int) int {
	resultsMu.Lock()
	defer resultsMu.Unlock()
	count := map[verdict]int{}
	for _, r := range results {
		count[r.Verdict]++
	}
	gateMu.Lock()
	gatePass := 0
	for _, g := range gateResults {
		if g.Pass {
			gatePass++
		}
	}
	gates := len(gateResults)
	gateMu.Unlock()
	b, _ := json.Marshal(map[string]any{
		"scenarios": len(results), "confirmed": count[confirmed], "refuted": count[refuted],
		"unsupported": count[unsupported], "invalid": count[invalid],
		"gate_checks": gates, "gate_passed": gatePass, "gate_failed": gates - gatePass,
	})
	fmt.Println("WP25SUMMARY " + string(b))
	if len(results) == 0 {
		fmt.Println("WP25SUMMARY no scenario ran: the run is invalid")
		return 1
	}
	return code
}

// helper builds a command that runs this test binary as a helper.
func helper(mode string, env []string, args ...string) *exec.Cmd {
	self, err := os.Executable()
	if err != nil {
		panic(err)
	}
	cmd := exec.Command(self, args...)
	cmd.Env = append(os.Environ(), helperEnv+"="+mode,
		limitEnv+"="+strconv.Itoa(int(helperLimit/time.Millisecond)))
	cmd.Env = append(cmd.Env, env...)
	return cmd
}

func durations(ds []time.Duration) string {
	if len(ds) == 0 {
		return "none"
	}
	s := append([]time.Duration(nil), ds...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return fmt.Sprintf("n=%d min=%v median=%v max=%v", len(s), s[0], s[len(s)/2], s[len(s)-1])
}
