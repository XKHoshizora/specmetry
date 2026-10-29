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
//
// Review 102 N-102-1: a self-check keeps every result of its inner run.
// P-3's target needs the layout probe of the same inner run; when the
// probe fails, the target is not executed and the self-check fails, and
// its result must still carry the probe's own reason and cleanup. These
// combinations check what is kept; they judge nothing anew.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	// Review 102 N-102-1, checked where keeps is set. inject replaces the
	// gate regression's inner fault ("" keeps it); stub puts a tool of
	// that name first on the inner run's PATH, a script that exits 1 (a
	// real failure, no seam). because must be part of the self-check's
	// reason; keeps are the inner run's results it must carry, all of them;
	// sites are the hits it must show, in order (none: no hit); ran says
	// whether the target launched and worked, or did neither.
	inject  string
	stub    string
	because string
	keeps   []innerKeep
	sites   []string
	ran     bool
}

// innerKeep is one result of the inner run a self-check must carry: its
// verdict, its reason naming because, its cleanup beginning cleanup.
type innerKeep struct {
	id      string
	verdict verdict
	because string
	cleanup string
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
	{id: "combo.forge-owner", gate: "control.silent-holder", fault: "forge-owner", fixture: "owner",
		tool: "ownership", unexpected: true, remedy: "not-needed", final: "unknown"},
	{id: "combo.forge-owner-mixed", gate: "witness.leak-unreceipted", fault: "forge-owner", fixture: "owner",
		expected: true, tool: "ownership", unexpected: true, remedy: "succeeded", final: "unknown"},
	{id: "combo.forge-stale", gate: "control.silent-holder", fault: "forge-owner", fixture: "stale",
		unexpected: true, remedy: "not-needed", final: "ended"},
	// Review 102 N-102-1 (macOS): the layout probe of the K1 regression's
	// inner run fails, by a real tool failure (a stub xcrun first on PATH)
	// with K1's seam armed and never reached, the shape of arm64's failure
	// in run 40817305419; or only at its end, after its evidence line, by
	// a close the seam p3-layout-close-late fails there. The self-check
	// must fail and carry the probe's own reason. The control: the healthy
	// run of the same scenarios.
	{id: "combo.trace-layout-tool", gate: "p3.k-nostatus-once", stub: "xcrun",
		tool: "hit", remedy: "not-needed", final: "ended",
		because: "not executed: the layout premise does not hold",
		keeps: []innerKeep{
			{"P3.layout.1", invalid, "the tool step xcrun-find-clang ended with exit 1", "verified"},
			{"P3.k.exit7.1", invalid, "not executed: the layout premise does not hold", "no processes"},
		}},
	{id: "combo.trace-layout-late", gate: "p3.k-nostatus-once", inject: "p3-layout-close-late",
		remedy: "not-needed", final: "ended",
		because: "not executed: the layout premise does not hold",
		keeps: []innerKeep{
			{"P3.layout.1", invalid, "closing the seam's late file", "NOT verified: closing failed"},
			{"P3.k.exit7.1", invalid, "not executed: the layout premise does not hold", "no processes"},
		},
		sites: []string{"the seam's late file"}},
	{id: "combo.trace-healthy", gate: "control.p3-k-exit7", wantPass: true,
		remedy: "not-needed", final: "ended",
		keeps: []innerKeep{
			{"P3.layout.1", confirmed, "", "verified"},
			{"P3.k.exit7.1", confirmed, "", "verified"},
		},
		ran: true},
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
	if cc.inject != "" {
		c.inject = cc.inject
	}
	f := outerFault{name: cc.fault}
	if cc.stub != "" {
		dir := t.TempDir()
		script := "#!/bin/sh\necho 'stub " + cc.stub + ": exits 1' >&2\nexit 1\n"
		if err := os.WriteFile(filepath.Join(dir, cc.stub), []byte(script), 0o755); err != nil {
			t.Fatalf("stub: %v", err)
		}
		f.env = []string{"PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")}
	}
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
	if len(cc.keeps) > 0 {
		why = append(why, traced(cc, self)...)
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

// traced checks what the self-check kept of its inner run (Review 102
// N-102-1): its own reason, each result the combination names exactly
// once and no other, the target's launches and work, the hits and their
// sites.
func traced(cc comboCase, g gateResult) (why []string) {
	if !strings.Contains(g.Reason, cc.because) {
		why = append(why, fmt.Sprintf("the self-check's reason does not name %q: %s", cc.because, g.Reason))
	}
	for _, k := range cc.keeps {
		var got []result
		for _, r := range g.Inner {
			if r.ID == k.id {
				got = append(got, r)
			}
		}
		switch {
		case len(got) != 1:
			why = append(why, fmt.Sprintf("the self-check keeps %d results of %s, want 1", len(got), k.id))
		case got[0].Verdict != k.verdict || !strings.Contains(got[0].Reason, k.because) || !strings.HasPrefix(got[0].Cleanup, k.cleanup):
			why = append(why, fmt.Sprintf("the kept result of %s is %s (%s), cleanup %q; want %s naming %q, cleanup beginning %q",
				k.id, got[0].Verdict, got[0].Reason, got[0].Cleanup, k.verdict, k.because, k.cleanup))
		}
	}
	if len(g.Inner) != len(cc.keeps) {
		why = append(why, fmt.Sprintf("the self-check keeps %d results of the inner run, want %d", len(g.Inner), len(cc.keeps)))
	}
	ran, none := g.Launches > 0 && g.Work > 0, g.Launches == 0 && g.Work == 0
	if cc.ran && !ran || !cc.ran && !none {
		why = append(why, fmt.Sprintf("the target launched %d and worked %d, want it run: %v", g.Launches, g.Work, cc.ran))
	}
	if len(g.Sites) != len(cc.sites) || g.Hits != len(cc.sites) {
		why = append(why, fmt.Sprintf("the self-check shows %d hits at %v, want %d at %v", g.Hits, g.Sites, len(cc.sites), cc.sites))
	} else {
		for i, s := range cc.sites {
			if !strings.Contains(g.Sites[i], s) {
				why = append(why, fmt.Sprintf("hit %d is at %q, want %q", i+1, g.Sites[i], s))
			}
		}
	}
	return why
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
