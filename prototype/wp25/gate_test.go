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
	"fmt"
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
	test     string // the real scenario's test function
	scenario string // its scenario ID
	inject   string // "" for a control
	want     verdict
	goos     []string // empty: every platform
}

type gateResult struct {
	ID        string  `json:"id"`
	Test      string  `json:"test"`
	Inject    string  `json:"inject"`
	Want      verdict `json:"want"`
	Got       verdict `json:"got"`
	Reason    string  `json:"reason,omitempty"`
	InnerExit int     `json:"inner_exit"`
	Launches  int     `json:"launches"`
	Work      int64   `json:"work"`
	Pass      bool    `json:"pass"`
	Millis    int64   `json:"ms"`
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
	{id: "report.truncated", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "report-truncated", want: invalid},
	{id: "report.overlong", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "report-overlong", want: invalid},
	{id: "report.deleted", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "report-delete", want: invalid},
	// The helper that builds the load fails, but the scenario goes on.
	{id: "load.parent-fails", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "parent-fail", want: invalid},
	// Review 075's counterexample B: no descendant at all; then only some.
	{id: "load.fanout-zero", test: "TestJobTerminateNotEnded", scenario: "T4.teardown.request-is-not-end", inject: "fanout-zero", want: invalid, goos: []string{"windows"}},
	{id: "load.fanout-partial", test: "TestJobTerminateNotEnded", scenario: "T4.teardown.request-is-not-end", inject: "fanout-partial", want: invalid, goos: []string{"windows"}},
	// A chain one level short: the load is not built, which is not the
	// mechanism giving a different result.
	{id: "load.chain-short", test: "TestAdoptionChain", scenario: "T2.adoption.chain", inject: "chain-short", want: invalid, goos: []string{"linux"}},
	// Cleanup and observation: an end that cannot be confirmed, a process
	// state that cannot be read, a process list that cannot be read.
	{id: "cleanup.unconfirmed", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "cleanup-unconfirmed", want: invalid},
	{id: "cleanup.state-unknown", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "procinfo-unknown", want: invalid, goos: []string{"linux", "darwin"}},
	{id: "observe.readdir-fails", test: "TestPipeNormalDrain", scenario: "T1.pipe.normal-drain", inject: "readdir-fail", want: invalid, goos: []string{"linux"}},
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

type innerRun struct {
	exit    int
	results map[string]result
	err     error
}

// runInner runs one test function in a new process of this binary, with
// the fault injected, under a hard limit of its own and one of the
// parent's.
func runInner(test, inject string) innerRun {
	self, err := os.Executable()
	if err != nil {
		return innerRun{err: err}
	}
	cmd := exec.Command(self, "-test.run", "^"+test+"$", "-test.count=1", "-test.v", "-test.timeout=150s")
	cmd.Env = append(os.Environ(), innerEnv+"=1", injectEnv+"="+inject)
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

func checkGate(t *testing.T, c gateCase) {
	begin := time.Now()
	r := runInner(c.test, c.inject)
	g := gateResult{ID: c.id, Test: c.test, Inject: c.inject, Want: c.want, InnerExit: r.exit}
	got, ok := r.results[c.scenario]
	switch {
	case r.err != nil:
		g.Reason = "inner run: " + r.err.Error()
	case !ok:
		g.Reason = "the inner run gave no result for " + c.scenario
	default:
		g.Got, g.Reason, g.Launches, g.Work = got.Verdict, got.Reason, got.Launches, got.Work
		// A control must be accepted and exit 0; a broken run must be
		// rejected as invalid and make the diagnostic fail.
		g.Pass = got.Verdict == c.want && (r.exit == 0) == (c.want == confirmed)
	}
	g.Millis = time.Since(begin).Milliseconds()
	b, _ := json.Marshal(g)
	fmt.Println("WP25GATE " + string(b))
	gateMu.Lock()
	gateResults = append(gateResults, g)
	gateMu.Unlock()
	if !g.Pass {
		t.Errorf("%s: want %s (exit %s), got %s (exit %d): %s", c.id, c.want,
			map[bool]string{true: "0", false: "non-zero"}[c.want == confirmed], g.Got, g.InnerExit, g.Reason)
	}
}
