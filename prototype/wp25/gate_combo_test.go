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
	"runtime"
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
}

type comboResult struct {
	ID       string     `json:"id"`
	Gate     string     `json:"gate"`
	Fault    string     `json:"fault,omitempty"`
	NoInject bool       `json:"no_inject,omitempty"`
	Strict   bool       `json:"strict,omitempty"`
	WantPass bool       `json:"want_pass"`
	Self     gateResult `json:"self"` // the self-check under test
	Pass     bool       `json:"pass"`
	Reason   string     `json:"reason,omitempty"`
}

var (
	comboMu      sync.Mutex
	comboResults []comboResult
)

var comboCases = []comboCase{
	// The expected rejection found, the remediation and its end confirmed.
	{id: "combo.reject-remedy-ok", gate: "witness.leak-unreceipted", wantPass: true},
	// The expected rejection found, the remediation fails: the left
	// process stays alive (the review's counterexample), with or without a
	// record of it.
	{id: "combo.reject-remedy-error", gate: "witness.leak-unreceipted", fault: "remedy-error"},
	{id: "combo.reject-unrecorded-remedy-error", gate: "witness.leak-unrecorded", fault: "remedy-error"},
	// The expected rejection found, the remediation acts but its end cannot
	// be confirmed.
	{id: "combo.reject-end-unconfirmed", gate: "witness.leak-unreceipted", fault: "remedy-unconfirmed"},
	// The expected rejection found, the outer check's census or a read of
	// a recorded process fails.
	{id: "combo.reject-census-error", gate: "witness.deleted", fault: "census-error"},
	{id: "combo.reject-state-unknown", gate: "witness.write-fail", fault: "record-unknown"},
	// The expected rejection found, with a problem of another class the
	// case does not allow.
	{id: "combo.reject-with-unexpected", gate: "witness.leak-unrecorded", strict: true},
	// The expected rejection missing, everything else normal.
	{id: "combo.expected-missing", gate: "witness.truncated", noInject: true},
	// A healthy run that needs no remediation; the same with a read of a
	// recorded process failing.
	{id: "combo.healthy", gate: "control.silent-holder", wantPass: true},
	{id: "combo.healthy-state-unknown", gate: "control.silent-holder", fault: "record-unknown"},
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
	self := runGate(t, c, cc.fault)
	res := comboResult{ID: cc.id, Gate: cc.gate, Fault: cc.fault, NoInject: cc.noInject, Strict: cc.strict, WantPass: cc.wantPass, Self: self}
	var why []string
	if self.Pass != cc.wantPass {
		why = append(why, fmt.Sprintf("the self-check gave pass=%v, want %v", self.Pass, cc.wantPass))
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
