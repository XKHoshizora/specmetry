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
// scenario needs, a load that was not built, an observation that failed
// (any one, whatever followed), or no work done. The gate itself is tested
// in gate_test.go.
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
	for k, v := range metaExtra() {
		meta[k] = v
	}
	b, _ := json.Marshal(meta)
	fmt.Println("WP25META " + string(b))
	inner := os.Getenv(innerEnv) != "" && witnessOn()
	if inner {
		witnessRoot()
	}
	code := m.Run()
	if inner {
		tamperWitness() // a gate regression's fault of the witness file, if any
		printSeal()
	}
	if os.Getenv(innerEnv) == "" {
		p5Summary() // P-5's line of this job (darwin), none elsewhere
	}
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
	// The processes the scenario owned, as "pid@identity" (the identity is
	// known under a gate check), and the expected registrations it could
	// not make: a count, which includes expected processes never created.
	Owned     []string `json:"owned,omitempty"`
	Shortfall int      `json:"shortfall,omitempty"`
	// Invalid lists every reason the run cannot count, each on its own (a
	// P-3 gate check needs exactly one, spec §19.12).
	Invalid []string `json:"invalid,omitempty"`
	// P-5 (docs/wp2.5-prototype.md §16.11): the class of evidence (static,
	// layout, native, seam), absent elsewhere; and the run's calls of entry
	// E2, read after its end, a pointer so that absent is not 0.
	Evidence string `json:"evidence,omitempty"`
	Calls    *int   `json:"calls,omitempty"`
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
	// identity is "pid@identity", the identity empty where it is not known.
	identity() string
	// observeErr is the first observation of it that failed, if any. It is
	// kept: a later observation that succeeds does not undo it.
	observeErr() error
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
	left       []leftover // what the scenario's output steps handed on
	reports    []*reportFile
	baseKids   []int // this process's children when the scenario began (Linux)
	baseErr    error // the list of them could not be read
	after      []func()
	goid       uint64        // the test goroutine
	mu         sync.Mutex    // the scenario may be used from several goroutines
	cleanLimit time.Duration // the end's own limit, if any: over it, the run is invalid
	closeFail  []string      // what the scenario owned and could not close (closer)
	// A P-5 run's own deadline d (§16.8): the end's bounded waits take
	// min(their limit, d). Zero elsewhere.
	cap time.Time
	// final, if set, runs at the end once the cleanup is checked and before
	// the verdict (a P-5 run's calls and its deadline, §16.7, §16.8).
	final func()
}

// until is now plus d, but no later than the run's own deadline, if any.
func (s *scen) until(d time.Duration) time.Time {
	t := time.Now().Add(d)
	if !s.cap.IsZero() && s.cap.Before(t) {
		return s.cap
	}
	return t
}

// begin starts a scenario; the caller defers done. needLaunch and
// needWork say whether a run without launches or without work is invalid.
func begin(t *testing.T, gate, id string, k kind, needLaunch, needWork bool) *scen {
	s := &scen{
		t:          t,
		r:          result{ID: id, Gate: gate, Kind: k, Platform: runtime.GOOS + "/" + runtime.GOARCH},
		begin:      time.Now(),
		needLaunch: needLaunch,
		needWork:   needWork,
		goid:       goid(),
	}
	// A list that cannot be read is not an empty one (Review 077 B1-R1-S):
	// the leftover check at the end would have nothing to compare with.
	s.baseKids, s.baseErr = ownChildren()
	return s
}

// flag makes the run invalid without stopping it: for what is found
// while the scenario ends, or for an observation that failed but did not
// end the scenario's control flow. Once flagged, a run stays invalid.
func (s *scen) flag(f string, a ...any) {
	s.locked(func() { s.invalid = append(s.invalid, fmt.Sprintf(f, a...)) })
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

// adopt makes the scenario the owner of what a run of the output step
// handed on (Review 081 B1): a stop not done, a copier not returned. done
// waits for them once the scenario's processes are gone and reads their
// results. A stop or a close that failed is a failed observation of the
// run, whether its error came before the run returned (here) or after
// (in done).
func (s *scen) adopt(r s6Result) s6Result {
	s.locked(func() { s.left = append(s.left, r.left...) })
	for name, err := range r.stopErr {
		s.flag("observation failed: stopping the read of %s: %v", name, err)
	}
	return r
}

// closer makes the scenario the owner of one thing to close from the
// moment it exists (Review 101 B2). The returned func closes it once,
// whoever calls it first, the owner's code or the scenario's end, and
// gives that close's error. The scenario keeps the error as a failure of
// the run, whatever follows, and its cleanup is then not verified. A
// close that failed is never made again: what it left is not known, and
// the descriptor may name another file by then.
func (s *scen) closer(what string, close func() error) func() error {
	var once sync.Once
	var err error
	f := func() error {
		once.Do(func() {
			if err = close(); err != nil {
				s.flag("cleanup not verified: closing %s failed: %v", what, err)
				s.locked(func() { s.closeFail = append(s.closeFail, what) })
			}
		})
		return err
	}
	s.defer_(func() { f() })
	return f
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
	ending := time.Now()
	for i := len(s.after) - 1; i >= 0; i-- {
		s.after[i]()
	}
	s.r.Cleanup = s.cleanup()
	// A handle whose close failed is not known to be released (Review 101
	// B2): the processes' end does not verify it.
	if len(s.closeFail) > 0 {
		s.r.Cleanup = "NOT verified: closing failed (" + strings.Join(s.closeFail, ", ") + "); " + s.r.Cleanup
	}
	// What the output steps handed on ends once its reads return, at the
	// latest when the processes holding the write ends are gone, which the
	// cleanup has just confirmed (Review 081 B1).
	// Each is waited for, and its result read: an error it met, after the
	// run returned or while still running, fails the observation, and
	// nothing after it clears that (Review 082 B1-Ra).
	var unended []string
	deadline := s.until(5 * time.Second)
	for _, l := range s.left {
		ended, err := l.until(deadline)
		if err != nil {
			s.invalid = append(s.invalid, fmt.Sprintf("observation failed: %s: %v", l.what, err))
		}
		if !ended {
			unended = append(unended, l.what)
		}
	}
	if len(unended) > 0 {
		s.invalid = append(s.invalid, "cleanup not verified: still running: "+strings.Join(unended, ", "))
		s.r.Cleanup = "NOT verified: " + strings.Join(unended, ", ") + " still running; " + s.r.Cleanup
	}
	// The writers have ended (or the cleanup says they have not): every
	// report must now hold exactly what its scenario declared.
	for _, rf := range s.reports {
		if err := checkFinal(*rf); err != nil {
			s.invalid = append(s.invalid, "report: "+err.Error())
		}
		s.r.Shortfall += rf.shortfall
	}
	// Expected registrations a rejection could not make: processes never
	// reported (some never created), or reported in a report that could
	// not be read, so neither owned nor signalled by their PID.
	if s.r.Shortfall > 0 {
		s.invalid = append(s.invalid, fmt.Sprintf("cleanup not verified: %d expected processes were never registered", s.r.Shortfall))
		s.r.Cleanup = fmt.Sprintf("NOT verified: %d expected processes never registered; %s", s.r.Shortfall, s.r.Cleanup)
	}
	// Children of this process that the scenario left (Linux): not reaped,
	// or never owned. Without the list from the start, nothing can be
	// compared.
	if s.baseErr != nil {
		s.invalid = append(s.invalid, "observation failed: the children of this process at the start could not be read: "+s.baseErr.Error())
		s.r.Cleanup = "NOT verified: no list of children from the start; " + s.r.Cleanup
	} else if left := newChildren(s.baseKids); left != "" {
		s.invalid = append(s.invalid, "cleanup: children of this process left: "+left)
		s.r.Cleanup = "NOT verified: children left (" + left + "); " + s.r.Cleanup
	}
	for _, p := range s.procs {
		p.release()
		if p.pidOf() > 0 {
			s.r.Owned = append(s.r.Owned, p.identity())
		}
	}
	if d := time.Since(ending); s.cleanLimit > 0 && d > s.cleanLimit {
		s.invalid = append(s.invalid, fmt.Sprintf("cleanup not verified in time: the end took %v, over its %v", d.Round(time.Millisecond), s.cleanLimit))
	}
	if s.final != nil {
		s.final()
	}
	s.r.Invalid = append([]string(nil), s.invalid...)
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
	deadline := s.until(5 * time.Second)
	for {
		var left []string
		for _, p := range s.procs {
			if !p.ended() {
				left = append(left, p.String())
			}
		}
		if len(left) == 0 {
			names := make([]string, 0, len(s.procs))
			var failed []string
			for _, p := range s.procs {
				names = append(names, p.String())
				if err := p.observeErr(); err != nil {
					failed = append(failed, fmt.Sprintf("%s (%v)", p.String(), err))
				}
			}
			// The ends are confirmed now, but an observation that failed on
			// the way stays a failure of this run (Review 076 B1-R1): a
			// later success does not restore what was not observed.
			if len(failed) > 0 {
				s.invalid = append(s.invalid, "observation failed during the cleanup: "+strings.Join(failed, ", "))
				return fmt.Sprintf("ended, but observations failed: %d processes gone (%s)", len(s.procs), strings.Join(names, ", "))
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
	comboMu.Lock()
	comboPass := 0
	for _, c := range comboResults {
		if c.Pass {
			comboPass++
		}
	}
	combos := len(comboResults)
	comboMu.Unlock()
	b, _ := json.Marshal(map[string]any{
		"scenarios": len(results), "confirmed": count[confirmed], "refuted": count[refuted],
		"unsupported": count[unsupported], "invalid": count[invalid],
		"gate_checks": gates, "gate_passed": gatePass, "gate_failed": gates - gatePass,
		"combo_checks": combos, "combo_passed": comboPass, "combo_failed": combos - comboPass,
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
