package wp25

// The outer check's own verdict under combinations (Review 078
// B1-R2-W-R): a gate regression whose expected rejection is found, while
// in the same run the outer check itself fails (its remediation, the end
// it confirms, its census, a read of a recorded process). The self-check
// under test must then not pass; this test, one level up, passes when the
// self-check correctly refused. Each combination is a real run: the inner
// run starts its helpers, the outer check takes its census, the injected
// fault acts on the tested path only, and the experiment's insurance
// cleanup ends what is left afterwards. Combinations are counted on
// WP25COMBO lines, apart from the gate regressions and the scenarios, and
// a self-check that refused here is never counted as a gate check passed.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
)

type comboCase struct {
	id       string
	gate     string // the gate regression it runs
	fault    string // the outer-side fault of the tested path
	noInject bool   // the gate regression's inner fault is not injected
	strict   bool   // no other class of problem is allowed alongside the expected one
	wantPass bool   // the self-check's own verdict
	// The self-check's dimensions this combination must show: the expected
	// rejection found or not, a tool error of a class ("" none), other
	// problems of the evidence, the remediation and the final state.
	// The dimensions are checked where remedy is set.
	expected   bool
	tool       string
	unexpected bool
	remedy     string
	final      string
	// fixture (Review 079 B2): a process this combination starts and holds
	// itself, outside the inner run, that a forged creation record names
	// ("owner": its real identity; "stale": its PID with another start
	// time). The outer check must hand it to no entry that acts by PID and
	// signal it never; it must still be alive afterwards. The combination
	// then ends it itself.
	fixture string
}

type comboResult struct {
	ID       string     `json:"id"`
	Gate     string     `json:"gate"`
	Fault    string     `json:"fault,omitempty"`
	NoInject bool       `json:"no_inject,omitempty"`
	Strict   bool       `json:"strict,omitempty"`
	WantPass bool       `json:"want_pass"`
	Self     gateResult `json:"self"` // the self-check under test
	// The fixture: handed to an acting entry, signals that would have been
	// sent to it (recorded, not sent), alive after the outer check, ended
	// by this combination's own cleanup.
	Fixture        string `json:"fixture,omitempty"`
	FixtureActed   bool   `json:"fixture_acted,omitempty"`
	FixtureSignals int    `json:"fixture_signals,omitempty"`
	FixtureAlive   bool   `json:"fixture_alive,omitempty"`
	FixtureEnded   bool   `json:"fixture_ended,omitempty"`
	Pass           bool   `json:"pass"`
	Reason         string `json:"reason,omitempty"`
}

var (
	comboMu      sync.Mutex
	comboResults []comboResult
)

var comboCases = []comboCase{
	// The expected rejection found, the remediation and its end confirmed.
	{id: "combo.reject-remedy-ok", gate: "witness.leak-unreceipted", wantPass: true,
		expected: true, remedy: "succeeded", final: "ended"},
	// The expected rejection found, the remediation fails: the left
	// process stays alive (the review's counterexample), with or without a
	// record of it.
	{id: "combo.reject-remedy-error", gate: "witness.leak-unreceipted", fault: "remedy-error",
		expected: true, tool: "still-alive", remedy: "failed", final: "alive"},
	{id: "combo.reject-unrecorded-remedy-error", gate: "witness.leak-unrecorded", fault: "remedy-error",
		expected: true, tool: "still-alive", remedy: "failed", final: "alive"},
	// The expected rejection found, the remediation acts but its end cannot
	// be confirmed.
	{id: "combo.reject-end-unconfirmed", gate: "witness.leak-unreceipted", fault: "remedy-unconfirmed",
		expected: true, tool: "still-alive", remedy: "failed", final: "unknown"},
	// The expected rejection found, the outer check's census or a read of
	// a recorded process fails.
	{id: "combo.reject-census-error", gate: "witness.deleted", fault: "census-error",
		expected: true, tool: "census", remedy: "not-needed", final: "unknown"},
	{id: "combo.reject-state-unknown", gate: "witness.write-fail", fault: "record-unknown",
		expected: true, tool: "state-unknown", remedy: "not-needed", final: "unknown"},
	// The expected rejection found, with a problem of another class the
	// case does not allow.
	{id: "combo.reject-with-unexpected", gate: "witness.leak-unrecorded", strict: true,
		expected: true, unexpected: true, remedy: "succeeded", final: "ended"},
	// The expected rejection missing, everything else normal.
	{id: "combo.expected-missing", gate: "witness.truncated", noInject: true,
		remedy: "not-needed", final: "ended"},
	// A healthy run that needs no remediation; the same with a read of a
	// recorded process failing.
	{id: "combo.healthy", gate: "control.silent-holder", wantPass: true,
		remedy: "not-needed", final: "ended"},
	{id: "combo.healthy-state-unknown", gate: "control.silent-holder", fault: "record-unknown",
		tool: "state-unknown", remedy: "not-needed", final: "unknown"},
	// Review 079 B2: a well-formed creation record names a live process
	// outside the inner run (the combination's own fixture): in a healthy
	// run; alongside one of the run's own processes left alive; and with
	// the fixture's PID but another start time (an identity that changed).
	{id: "combo.forge-owner", gate: "control.silent-holder", fault: "forge-owner", fixture: "owner"},
	{id: "combo.forge-owner-mixed", gate: "witness.leak-unreceipted", fault: "forge-owner", fixture: "owner"},
	{id: "combo.forge-stale", gate: "control.silent-holder", fault: "forge-owner", fixture: "stale"},
}

func TestGateCombos(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	byID := map[string]gateCase{}
	for _, c := range gateCases {
		byID[c.id] = c
	}
	for _, cc := range comboCases {
		cc := cc
		c, ok := byID[cc.gate]
		if !ok {
			t.Fatalf("%s: no gate regression %s", cc.id, cc.gate)
		}
		if len(c.goos) > 0 && !contains(c.goos, runtime.GOOS) {
			continue
		}
		t.Run(cc.id, func(t *testing.T) { checkCombo(t, cc, c) })
	}
}

func checkCombo(t *testing.T, cc comboCase, c gateCase) {
	if cc.noInject {
		c.inject, c.hit = "", false
	}
	if cc.strict {
		c.also = nil
	}
	f := outerFault{name: cc.fault}
	var fx *exec.Cmd
	var fxID survivor
	signals := func() int { return 0 }
	if cc.fixture != "" {
		// The fixture: started and held by this combination, never by the
		// inner run; its identity is read while it is this process's child.
		fx = helper("sleep", nil)
		if err := start(fx); err != nil {
			t.Fatalf("fixture: %v", err)
		}
		id, err := childIdentity(fx)
		if err != nil {
			_ = fx.Process.Kill()
			_ = fx.Wait()
			t.Fatalf("fixture identity: %v", err)
		}
		fxID = survivor{fx.Process.Pid, id}
		f.forge = fxID
		if cc.fixture == "stale" {
			f.forge.start = shiftIdentity(id)
		}
		var restore func()
		restore, signals = recordSignals(fxID.pid)
		defer restore()
	}
	self := runGate(t, c, f)
	res := comboResult{ID: cc.id, Gate: cc.gate, Fault: cc.fault, NoInject: cc.noInject, Strict: cc.strict, WantPass: cc.wantPass, Self: self, Fixture: cc.fixture}
	var why []string
	if self.Pass != cc.wantPass {
		why = append(why, fmt.Sprintf("the self-check gave pass=%v, want %v", self.Pass, cc.wantPass))
	}
	// Each dimension as the combination makes it, not only the verdict.
	if cc.remedy != "" {
		if self.Expected != cc.expected {
			why = append(why, fmt.Sprintf("expected rejection found %v, want %v", self.Expected, cc.expected))
		}
		if cc.tool == "" && len(self.ToolErrors) > 0 || cc.tool != "" && !hasClass(self.ToolErrors, cc.tool) {
			why = append(why, fmt.Sprintf("tool errors %v, want class %q", self.ToolErrors, cc.tool))
		}
		if (len(self.Unexpected) > 0) != cc.unexpected {
			why = append(why, fmt.Sprintf("other problems %v, want some: %v", self.Unexpected, cc.unexpected))
		}
		if self.Remedy != cc.remedy || self.Final != cc.final {
			why = append(why, fmt.Sprintf("remediation %s and final state %s, want %s and %s", self.Remedy, self.Final, cc.remedy, cc.final))
		}
	}
	if fx != nil {
		// Nothing outside the inner run may be acted on: not handed to an
		// entry that acts by PID, never signalled, still alive.
		res.FixtureActed = contains(self.Acted, fxID.key())
		res.FixtureSignals = signals()
		a, err := recordAlive(witnessRec{pid: fxID.pid, start: fxID.start})
		res.FixtureAlive = a && err == nil
		if res.FixtureActed || res.FixtureSignals > 0 || !res.FixtureAlive {
			why = append(why, fmt.Sprintf("the fixture outside the inner run was acted on: handed to an acting entry %v, %d signals, alive %v (%v)",
				res.FixtureActed, res.FixtureSignals, res.FixtureAlive, err))
		}
		// The combination's own cleanup of its own fixture, apart from the
		// outer check: it started the fixture and holds it.
		_ = fx.Process.Kill()
		_ = fx.Wait()
		a, err = recordAlive(witnessRec{pid: fxID.pid, start: fxID.start})
		res.FixtureEnded = !a && err == nil
		if !res.FixtureEnded {
			why = append(why, fmt.Sprintf("the fixture could not be confirmed ended (%v)", err))
		}
	}
	// Whatever the tested path did, the experiment's own cleanup must
	// leave nothing of this round behind.
	if !insured(self.Insurance) {
		why = append(why, "the insurance cleanup did not confirm the end: "+self.Insurance)
	}
	res.Pass, res.Reason = len(why) == 0, strings.Join(why, "; ")
	b, _ := json.Marshal(res)
	fmt.Println("WP25COMBO " + string(b))
	comboMu.Lock()
	comboResults = append(comboResults, res)
	comboMu.Unlock()
	if !res.Pass {
		t.Errorf("%s: %s; self-check problems %v", cc.id, res.Reason, self.Problems)
	}
}

func insured(s string) bool { return s == "nothing left" || strings.HasPrefix(s, "ended ") }

// shiftIdentity gives another identity of the same form: one second later
// on macOS, one tick later elsewhere.
func shiftIdentity(s string) string {
	sec, usec, ok := strings.Cut(s, ".")
	n, _ := strconv.ParseUint(sec, 10, 64)
	if ok {
		return strconv.FormatUint(n+1, 10) + "." + usec
	}
	return strconv.FormatUint(n+1, 10)
}
