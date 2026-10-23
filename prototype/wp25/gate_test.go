package wp25

// Regressions of the validity gate itself (Review 075 B1). Each one runs a
// real scenario again in a separate process of this test binary, with one
// fault injected through WP25_INJECT, and checks the verdict and the exit
// status the real diagnostic gives. A gate check passes when the real
// diagnostic rejects the broken run (invalid, non-zero exit), or, for a
// control, accepts the healthy one. A passing gate check is not a
// mechanism confirmed: it is reported on its own WP25GATE line and counted
// apart from the scenarios.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	injectEnv = "WP25_INJECT"
	innerEnv  = "WP25_INNER"
)

// injected reports whether this process runs with the named fault.
func injected(name string) bool { return os.Getenv(injectEnv) == name }

type gateCase struct {
	id       string
	test     string   // the real scenario's test function
	scenario string   // its scenario ID
	inject   string   // "" for a control
	want     verdict  // "": the inner verdict is recorded, not judged
	goos     []string // empty: every platform
	hit      bool     // the fault must be seen reaching its site
	// reject is the class of problem the outer check must find (Review 077
	// B1-R2-W): the gate check passes only if the evidence is rejected for
	// that reason. "": the evidence must be accepted, with no problem.
	reject string
	// noReceipt: the outer check must see a process created without its
	// own record (one that ended before it ran), and confirm its end.
	noReceipt bool
}

type gateResult struct {
	ID        string   `json:"id"`
	Test      string   `json:"test"`
	Inject    string   `json:"inject"`
	Want      verdict  `json:"want"`
	Got       verdict  `json:"got"`
	Reason    string   `json:"reason,omitempty"`
	InnerExit int      `json:"inner_exit"`
	Launches  int      `json:"launches"`
	Work      int64    `json:"work"`
	Hits      int      `json:"inject_hits"`
	Reject    string   `json:"reject,omitempty"`
	NoReceipt int      `json:"no_receipt"` // created, ended before their own record
	Launched  int      `json:"launched"`   // helpers the witness file saw start
	Unowned   int      `json:"unowned"`    // of those, not in the inner run's owned list
	Missing   int      `json:"missing"`    // processes the inner run declared unaccounted
	Left      []int    `json:"left"`       // still alive after the inner run, stopped by the gate check
	Problems  []string `json:"problems,omitempty"`
	Pass      bool     `json:"pass"`
	Millis    int64    `json:"ms"`
}

var (
	gateMu      sync.Mutex
	gateResults []gateResult
)

var gateCases = []gateCase{
	// Controls: a healthy run and the legitimate kinds stay accepted.
	{id: "control.silent-holder", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", want: confirmed},
	{id: "control.normal-drain", test: "TestPipeNormalDrain", scenario: "T1.pipe.normal-drain", want: confirmed},
	{id: "control.counterexample", test: "TestPipeCapacityStop", scenario: "T6.pipe.capacity-stop", want: confirmed},
	{id: "control.refusal", test: "TestExclusiveBaselineLive", scenario: "T2.baseline.live-child", want: confirmed, goos: []string{"linux"}},
	{id: "control.adoption-chain", test: "TestAdoptionChain", scenario: "T2.adoption.chain", want: confirmed, goos: []string{"linux"}},
	{id: "control.fault-injection", test: "TestJobFailAssign", scenario: "T4.start.fail-assign", want: confirmed, goos: []string{"windows"}},
	{id: "control.fanout", test: "TestJobTerminateNotEnded", scenario: "T4.teardown.request-is-not-end", want: confirmed, goos: []string{"windows"}},

	// Review 075's counterexample A: a malformed record next to the real one.
	{id: "report.malformed", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "report-malformed", want: invalid},
	// A required record missing, unknown, duplicated, truncated, too long,
	// or the report unreadable at the end.
	{id: "report.missing", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "report-skip", want: invalid},
	{id: "report.unknown-kind", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "report-unknown", want: invalid},
	{id: "report.duplicate", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "report-dup", want: invalid},
	// A well-formed record of a kind this scenario does not expect: only
	// the final check against the scenario's spec can reject it.
	{id: "report.unexpected-kind", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "report-extra", want: invalid},
	{id: "report.truncated", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "report-truncated", want: invalid},
	{id: "report.overlong", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "report-overlong", want: invalid},
	{id: "report.deleted", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "report-delete", want: invalid},
	// The helper that builds the load fails, but the scenario goes on.
	{id: "load.parent-fails", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "parent-fail", want: invalid},
	// Review 075's counterexample B: no descendant at all; then only some.
	{id: "load.fanout-zero", test: "TestJobTerminateNotEnded", scenario: "T4.teardown.request-is-not-end", inject: "fanout-zero", want: invalid, goos: []string{"windows"}},
	{id: "load.fanout-partial", test: "TestJobTerminateNotEnded", scenario: "T4.teardown.request-is-not-end", inject: "fanout-partial", want: invalid, goos: []string{"windows"}},
	// A chain one level short: the load is not built, which is not the
	// mechanism giving a different result. Review 076 B1-R2: the levels that
	// were started must still end when the scenario is rejected.
	{id: "load.chain-short", test: "TestAdoptionChain", scenario: "T2.adoption.chain", inject: "chain-short", want: invalid, goos: []string{"linux"}, hit: true},
	// The same chain with a report that cannot be read: the levels' PIDs
	// are not even known from it.
	{id: "load.chain-corrupt", test: "TestAdoptionChain", scenario: "T2.adoption.chain", inject: "report-malformed", want: invalid, goos: []string{"linux"}},
	// Ownership: a reported PID that is the scenario's own process but not
	// in the check's tree; only the ownership check can reject it.
	{id: "ownership.foreign-pid", test: "TestInnerForeignPID", scenario: "gate.foreign-pid", want: invalid},
	// Cleanup and observation: an end that cannot be confirmed, a process
	// state that cannot be read, a process list that cannot be read.
	{id: "cleanup.unconfirmed", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "cleanup-unconfirmed", want: invalid},
	// Review 076 B1-R1: one observation fails, the ones after it succeed.
	// The failure must still make the run invalid.
	{id: "observe.fail-once", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "observe-fail-once", want: invalid, hit: true},
	{id: "cleanup.state-unknown", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "procinfo-unknown", want: invalid, goos: []string{"linux", "darwin"}},
	{id: "observe.readdir-fails", test: "TestPipeNormalDrain", scenario: "T1.pipe.normal-drain", inject: "readdir-fail", want: invalid, goos: []string{"linux"}},

	// Review 077 B1-R1-S: one failed observation at the other shared
	// entries, the ones after it real. Each must make the run invalid.
	// After the pidfd bind in the sweep, one state read fails.
	{id: "observe.sweep-recheck-once", test: "TestExclusiveGate", scenario: "T2.gate.exclusive-interval", inject: "sweep-recheck-once", want: invalid, goos: []string{"linux"}, hit: true},
	// One waitid of the sweep fails without reaping; the next one reaps.
	{id: "observe.sweep-wait-once", test: "TestExclusiveGate", scenario: "T2.gate.exclusive-interval", inject: "sweep-wait-once", want: invalid, goos: []string{"linux"}, hit: true},
	// The scenario's first list of this process's children fails.
	{id: "observe.baseline-once", test: "TestPipeNormalDrain", scenario: "T1.pipe.normal-drain", inject: "readdir-fail-once", want: invalid, goos: []string{"linux"}, hit: true},
	// The read of the subreaper setting fails.
	{id: "observe.prctl-fails", test: "TestExclusiveBaselineLive", scenario: "T2.baseline.live-child", inject: "prctl-get-fail", want: invalid, goos: []string{"linux"}, hit: true},
	// Whether the process is in the Job cannot be read.
	{id: "observe.injob-once", test: "TestJobFailResume", scenario: "T4.start.fail-resume", inject: "injob-fail-once", want: invalid, goos: []string{"windows"}, hit: true},

	// Review 077 B1-R2-W: the witness itself fails. The outer check must
	// reject the evidence, whatever the inner run gives.
	{id: "witness.write-fail", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-write-fail", hit: true, reject: "witness-raised"},
	{id: "witness.short-write", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-short-write", hit: true, reject: "witness-malformed"},
	{id: "witness.identity-unknown", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-identity-unknown", hit: true, reject: "witness-raised"},
	{id: "witness.truncated", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-truncated", hit: true, reject: "witness-malformed"},
	{id: "witness.duplicate", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-duplicate", hit: true, reject: "witness-malformed"},
	{id: "witness.unknown-kind", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-unknown-kind", hit: true, reject: "witness-malformed"},
	{id: "witness.deleted", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-deleted", hit: true, reject: "witness-unreadable"},
	// A process the inner run leaves alive without its own record, or
	// without any record; one that ends before any record (Windows: only
	// the Job's count of created processes can show it).
	{id: "witness.leak-unreceipted", test: "TestInnerLeak", scenario: "gate.leak", inject: "leak-unreceipted", want: invalid, hit: true, reject: "left"},
	{id: "witness.leak-unrecorded", test: "TestInnerLeak", scenario: "gate.leak", inject: "leak-unrecorded", want: invalid, hit: true, reject: "left"},
	{id: "witness.unrecorded-ended", test: "TestInnerLeak", scenario: "gate.leak", inject: "unrecorded-ended", want: invalid, goos: []string{"windows"}, hit: true, reject: "witness-count"},
	// A positive case: a helper that ends before it leaves its own record
	// is not an omission, once its creation and its end are confirmed.
	{id: "witness.early-exit", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "helper-exit-at-start", want: invalid, hit: true, noReceipt: true},
}

func TestGate(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	for _, c := range gateCases {
		c := c
		if len(c.goos) > 0 && !contains(c.goos, runtime.GOOS) {
			continue
		}
		t.Run(c.id, func(t *testing.T) { checkGate(t, c) })
	}
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// TestInnerForeignPID runs only as an inner run of ownership.foreign-pid:
// a check's tree is asked to own a process outside it (a sibling helper
// this scenario started itself, so nothing unknown is touched). The
// ownership check must stop the scenario as invalid.
func TestInnerForeignPID(t *testing.T) {
	if os.Getenv(innerEnv) == "" {
		t.Skip("runs only as a gate regression")
	}
	s := begin(t, "gate", "gate.foreign-pid", fault, true, true)
	defer s.done()
	sibling := helper("sleep", nil)
	startHelper(s, sibling)
	tr := startTree(s, helper("sleep", nil))
	s.worked(1)
	tr.ownReported(sibling.Process.Pid)
	s.expect("got past the ownership check", true, true)
}

// TestInnerLeak runs only as an inner run of the witness regressions: it
// leaves one of its own helpers behind on purpose, as a broken scenario
// would, and does not own it, so only the outer gate check can find it.
// The helper has a short limit of its own (8 s).
func TestInnerLeak(t *testing.T) {
	if os.Getenv(innerEnv) == "" {
		t.Skip("runs only as a gate regression")
	}
	s := begin(t, "gate", "gate.leak", fault, true, true)
	defer s.done()
	cmd := helper("linger", []string{limitEnv + "=8000"})
	injectHit("TestInnerLeak", os.Getpid())
	if err := cmd.Start(); err != nil {
		s.invalidf("helper did not start: %v", err)
	}
	s.launched(1)
	s.worked(1)
	s.flag("the scenario leaves its helper %d behind on purpose", cmd.Process.Pid)
	s.expect("left behind", true, true)
}

type innerRun struct {
	exit    int
	results map[string]result
	hits    int
	err     error
}

// runInner runs one test function in a new process of this binary, with
// the fault injected, under a hard limit of its own and one of the
// parent's. The helpers it starts write to the witness file.
func runInner(test, inject, witnessFile, injectLog string) innerRun {
	self, err := os.Executable()
	if err != nil {
		return innerRun{err: err}
	}
	cmd := exec.Command(self, "-test.run", "^"+test+"$", "-test.count=1", "-test.v", "-test.timeout=150s")
	cmd.Env = append(os.Environ(), innerEnv+"=1", injectEnv+"="+inject, witnessEnv+"="+witnessFile, injectLogEnv+"="+injectLog)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Start(); err != nil {
		return innerRun{err: err}
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(180 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		return innerRun{err: fmt.Errorf("inner run did not end within 180 s")}
	}
	r := innerRun{exit: cmd.ProcessState.ExitCode(), results: map[string]result{}}
	sc := bufio.NewScanner(bytes.NewReader(out.Bytes()))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.Contains(line, "WP25INJECT ") {
			r.hits++
		}
		i := strings.Index(line, "WP25RESULT ")
		if i < 0 {
			continue
		}
		var res result
		if err := json.Unmarshal([]byte(line[i+len("WP25RESULT "):]), &res); err != nil {
			r.err = fmt.Errorf("inner result line: %w", err)
			continue
		}
		r.results[res.ID] = res
	}
	if err := sc.Err(); err != nil {
		r.err = err
	}
	return r
}

// judged says whether the outer check came to what the case requires: no
// problem at all, or a problem of the class the case names (a fault of the
// witness must be found, not just any rejection); and, where the case
// says so, a created process seen ending before its own record.
func judged(c gateCase, g gateResult) bool {
	if c.noReceipt && g.NoReceipt == 0 {
		return false
	}
	if c.reject == "" {
		return len(g.Problems) == 0
	}
	for _, p := range g.Problems {
		if strings.HasPrefix(p, c.reject+":") {
			return true
		}
	}
	return false
}

func checkGate(t *testing.T, c gateCase) {
	begin := time.Now()
	dir := t.TempDir()
	wf, err := os.CreateTemp(dir, "witness-*")
	if err != nil {
		t.Fatal(err)
	}
	_ = wf.Close()
	il, err := os.CreateTemp(dir, "inject-*")
	if err != nil {
		t.Fatal(err)
	}
	_ = il.Close()
	r := runInner(c.test, c.inject, wf.Name(), il.Name())
	g := gateResult{ID: c.id, Test: c.test, Inject: c.inject, Want: c.want, Reject: c.reject, InnerExit: r.exit, Hits: r.hits}
	got, ok := r.results[c.scenario]
	verdictOK := false
	switch {
	case r.err != nil:
		g.Reason = "inner run: " + r.err.Error()
	case !ok:
		g.Reason = "the inner run gave no result for " + c.scenario
	default:
		g.Got, g.Reason, g.Launches, g.Work = got.Verdict, got.Reason, got.Launches, got.Work
		// A control must be accepted and exit 0; a broken run must be
		// rejected as invalid and make the diagnostic fail.
		verdictOK = c.want == "" || got.Verdict == c.want && (r.exit == 0) == (c.want == confirmed)
	}
	// The helpers' hits, from their own log (not the witness file).
	if b, err := os.ReadFile(il.Name()); err == nil {
		g.Hits += strings.Count(string(b), "\n")
	}
	if c.hit && g.Hits == 0 {
		g.Problems = append(g.Problems, "hit: the fault never reached its site")
	}
	// What the witness file saw, independently of the inner run's account.
	ws, err := readWitness(wf.Name())
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			g.Problems = append(g.Problems, "witness-unreadable: "+pe.Op+": "+pe.Err.Error())
		} else {
			g.Problems = append(g.Problems, "witness-malformed: "+err.Error())
		}
	}
	// Every helper the inner run started must have ended once it is over.
	alive, err := witnessEnded(ws, 10*time.Second)
	if err != nil {
		g.Problems = append(g.Problems, "state-unknown: witness state: "+err.Error())
	}
	for _, w := range alive {
		g.Left = append(g.Left, w.pid)
		witnessStop(w) // this round's own helper, by its witnessed identity
	}
	if len(g.Left) > 0 {
		g.Problems = append(g.Problems, fmt.Sprintf("left: %d helpers left alive by the inner run", len(g.Left)))
		if still, _ := witnessEnded(alive, 10*time.Second); len(still) > 0 {
			g.Problems = append(g.Problems, fmt.Sprintf("still-alive: %d of them still alive after the gate check stopped them", len(still)))
		}
	}
	// Every helper started must be in the inner run's owned list, or be
	// declared unaccounted by it: a rejection must not skip objects.
	owned := map[int]bool{}
	for _, pid := range got.Owned {
		owned[pid] = true
	}
	for _, w := range ws {
		if w.kind != "launched" {
			continue
		}
		g.Launched++
		if !owned[w.pid] {
			g.Unowned++
		}
	}
	g.Missing = got.Missing
	if g.Unowned > g.Missing {
		g.Problems = append(g.Problems, fmt.Sprintf("unaccounted: %d started helpers neither owned nor declared unaccounted", g.Unowned-g.Missing))
	}
	g.Pass = verdictOK && judged(c, g)
	g.Millis = time.Since(begin).Milliseconds()
	b, _ := json.Marshal(g)
	fmt.Println("WP25GATE " + string(b))
	gateMu.Lock()
	gateResults = append(gateResults, g)
	gateMu.Unlock()
	if !g.Pass {
		t.Errorf("%s: want %s (exit %s), got %s (exit %d): %s; %v", c.id, c.want,
			map[bool]string{true: "0", false: "non-zero"}[c.want == confirmed], g.Got, g.InnerExit, g.Reason, g.Problems)
	}
}
