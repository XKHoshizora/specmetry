package wp25

// Regressions of the validity gate itself (Review 075 B1). Each one runs a
// real scenario again in a separate process of this test binary, with one
// fault injected through WP25_INJECT, and checks the verdict and the exit
// status the real diagnostic gives. A gate check passes when the real
// diagnostic rejects the broken run (invalid, non-zero exit), or, for a
// control, accepts the healthy one or, with a working tool and a wrong
// prediction, refutes it (non-zero exit). The outer check also judges its own
// evidence (witness_test.go): a case that injects a fault of the witness
// passes only when the outer check rejects that evidence for the named
// reason. A passing gate check is not a mechanism confirmed: it is
// reported on its own WP25GATE line and counted apart from the scenarios.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
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

// injectedAny says whether one of the faults is set.
func injectedAny(names ...string) bool {
	for _, n := range names {
		if injected(n) {
			return true
		}
	}
	return false
}

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
	// also: the other classes of problem the case's fault may bring along
	// (a deliberate leak is also unregistered, and on Windows uncounted).
	also []string
	// noReceipt: the outer check must see a process created without its
	// own record (one that ended before it ran), and confirm its end.
	noReceipt bool
	// because, when set, must be part of the inner run's reason (Review 095
	// B1): the run is rejected for the injected failure itself, not for a
	// consequence of it, such as a load the failure happened to break.
	because string
	// alsoBecause, when set, must be part of the reason too (Review 096
	// B1-R): a second failure the run met, kept beside the first.
	alsoBecause string
	// exact (P-3, spec §19.12): an invalid run has exactly one reason, its
	// own cleanup is verified, and the other scenarios of the inner run
	// (the layout probe it needs) are confirmed.
	exact bool
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
	Sites     []string `json:"inject_sites,omitempty"` // where the hits were, as each recorded it; a helper's prefixed "helper: "
	// Observed is what the inner run recorded, for a fault case: the state
	// it read after the fault, kept apart from the fault's own result; and
	// InnerCleanup, what its own cleanup found.
	Observed     string   `json:"inner_observed,omitempty"`
	InnerCleanup string   `json:"inner_cleanup,omitempty"`
	Reject       string   `json:"reject,omitempty"`
	Created      int      `json:"created"`    // creation records: processes the inner run started
	Receipts     int      `json:"receipts"`   // helpers' own records
	NoReceipt    int      `json:"no_receipt"` // created, ended without a record of their own
	Unowned      []string `json:"unowned"`    // created, not registered by the inner run: "pid:state"
	Shortfall    int      `json:"shortfall"`  // expected registrations the inner run declared missing
	Left         []int    `json:"left"`       // alive when the inner run was over, before any remediation
	Unrecord     []int    `json:"unrecorded"` // of those, without a creation record
	Still        []int    `json:"still_alive"`
	Total        int      `json:"job_total"` // Windows: processes the Job saw created (inner included); -1 elsewhere
	Raised       bool     `json:"raised"`
	Insurance    string   `json:"insurance"` // the experiment's own last cleanup, after the judgment
	// Acted: the identities handed to the entries that act on a process by
	// its PID (the remediation and the insurance cleanup), each of them
	// proven this round's. Refused: the ones alive by a record whose
	// ownership was not proven, never handed on.
	Acted    []string `json:"acted,omitempty"`
	Refused  []string `json:"refused,omitempty"`
	Problems []string `json:"problems,omitempty"`
	// The judgment's dimensions, kept apart (Review 078 B1-R2-W-R): the
	// expected rejection, other problems of the evidence, the outer
	// check's own failures, the remediation and the final state of this
	// round's processes, from the tested path (before the insurance).
	Expected   bool     `json:"expected_found"`
	Unexpected []string `json:"unexpected,omitempty"`
	ToolErrors []string `json:"tool_errors,omitempty"`
	Remedy     string   `json:"remedy"` // not-needed, succeeded, failed, not-run
	RemedyErr  string   `json:"remedy_error,omitempty"`
	Final      string   `json:"final"` // ended, alive, unknown
	Pass       bool     `json:"pass"`
	Millis     int64    `json:"ms"`
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
	// Review 081 B1: what an output step handed on (an O3 copier here) is
	// still running when the scenario ends: its cleanup is not verified.
	{id: "cleanup.leftover-running", test: "TestDestO3GateLingering", scenario: "T1.O3.gate-lingering", inject: "o3-unreleased", want: invalid, hit: true},
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

	// Review 076 N4: the F1 scenario's holder sees its writes fail but
	// cannot write its record: the observation failed, so the run is
	// invalid, not a finding. The controls: the healthy run is confirmed; a
	// second handle on the read ends keeps the writes from failing, a wrong
	// prediction with a working tool, which stays refuted.
	{id: "control.f1-skip-drain", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", want: confirmed},
	{id: "control.f1-read-kept", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "read-end-kept", want: refuted, hit: true},
	{id: "report.epipe-unwritable", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "epipe-report-fail", want: invalid, hit: true},
	// Review 081 N4-R: the same, but the holder stays alive for a while
	// after its report failed. Alive is no evidence that its writes
	// succeeded, nor that its tool worked.
	{id: "report.epipe-unwritable-alive", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "epipe-report-fail-alive", want: invalid, hit: true},
	// Review 082 N4-R: late bytes on the one kept read end, while the
	// holder's write to the other stream fails and its report fails too.
	// The bytes prove that one write, not that the tool worked.
	{id: "report.epipe-mixed-fault", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "f1-mixed-fault", want: invalid, hit: true},
	// Each part of the decision on its own: both read ends kept and every
	// write succeeding, but the holder's confirmation cannot be written;
	// the holder confirms, but only stdout was kept and written.
	{id: "report.ok-unwritable", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-report-fail", want: invalid, hit: true},
	{id: "report.one-stream-seen", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "f1-one-kept", want: invalid, hit: true},
	// A record that arrives late, while the scenario waits for an answer,
	// is still the observation: the prediction holds.
	{id: "report.epipe-late", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "epipe-report-late", want: confirmed, hit: true},
	// Review 083 N4-R-S: an answer is no proof of what comes after it. The
	// review's case: a true answer, then the holder's stderr breaks, its
	// write fails, and its report fails, all before the scenario's checks.
	// And an answer that is not the end: the holder stops writing but
	// keeps its output open.
	{id: "report.fail-after-ok", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "fail-after-ok", want: invalid, hit: true},
	{id: "report.ok-not-final", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-open-output", want: invalid, hit: true},
	// At the end itself: the holder cannot close its output, so it gives
	// no answer (added with the fix; the old code has no such step).
	{id: "report.close-fails", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "output-close-fails", want: invalid, hit: true},
	// Review 084 B-N4: an answer and a failure record together break the
	// protocol, in either order, and neither may count. The review's case:
	// a true answer, then a write fails and its report is written. The
	// other order: a write after the end fails and is reported, and the
	// holder answers all the same. And an answer nobody asked for, beside
	// the failure record the first read finds.
	{id: "report.ok-epipe-reported", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-epipe-reported", want: invalid, hit: true},
	{id: "report.epipe-then-ok", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "epipe-then-ok", want: invalid, hit: true},
	{id: "report.ok-unasked", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-unasked", want: invalid, hit: true},
	// Records the protocol does not allow: one that names another process
	// than the holder, a duplicate, a truncated or a malformed one.
	{id: "report.epipe-foreign", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "epipe-foreign", want: invalid, hit: true},
	{id: "report.ok-foreign", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-foreign", want: invalid, hit: true},
	{id: "report.ok-duplicate", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-dup", want: invalid, hit: true},
	{id: "report.ok-truncated", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-truncated", want: invalid, hit: true},
	{id: "report.ok-malformed", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-malformed", want: invalid, hit: true},
	// A tool failure beside either record: the holder reports reaching its
	// hard limit.
	{id: "report.epipe-limit", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "epipe-limit", want: invalid, hit: true},
	{id: "report.ok-limit", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-limit", want: invalid, hit: true},
	// Review 085 B-N4-R: a whole record is no proof that the holder's tool
	// completed. Beside a whole epipe or ok record: the close of the report
	// fails; a tool failure is reached and its report is cut off by the
	// scenario's end of the holder; that report is written and fails. And a
	// record itself cut off by that end.
	{id: "report.epipe-close-fails", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "epipe-close-fails", want: invalid, hit: true},
	{id: "report.epipe-limit-cut", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "epipe-limit-gated", want: invalid, hit: true},
	{id: "report.epipe-limit-unwritable", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "epipe-limit-unwritable", want: invalid, hit: true},
	{id: "report.ok-close-fails", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-close-fails", want: invalid, hit: true},
	{id: "report.ok-limit-cut", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-limit-gated", want: invalid, hit: true},
	{id: "report.ok-limit-unwritable", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-limit-unwritable", want: invalid, hit: true},
	{id: "report.epipe-cut", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "epipe-cut", want: invalid, hit: true},
	// The completion itself (added with the fix; the old code has none): its
	// close fails beside a record of a failed write; it names another
	// process than the holder beside a final answer.
	{id: "report.done-close-fails", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "done-close-fails", want: invalid, hit: true},
	{id: "report.done-foreign", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "done-foreign", want: invalid, hit: true},
	// Review 086 B-N4-R: a completion is published although another entry
	// of the same holder has failed. The hard limit's real callback has
	// entered and its report is still running when the holder completes,
	// on the epipe path and on the answer path (B-N4-R/T). A report fails
	// for real, its caller ignores that, and a later record succeeds and is
	// completed, on either path (B-N4-R/I). And the rename of the
	// completion fails.
	{id: "report.limit-race-epipe", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "limit-race-epipe", want: invalid, hit: true},
	{id: "report.limit-race-ok", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "limit-race-ok", want: invalid, hit: true},
	{id: "report.ok-fails-epipe-reported", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "ok-fails-epipe-reported", want: invalid, hit: true},
	{id: "report.epipe-fails-then-ok", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "epipe-fails-then-ok", want: invalid, hit: true},
	{id: "report.done-rename-fails", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "done-rename-fails", want: invalid, hit: true},
	// The other order (added with the fix): the completion is published
	// first, then the hard limit's real callback enters. It only ends the
	// holder; nothing of it belongs to the observation, which stays
	// confirmed. (The old code has no arbitration: its callback reports the
	// limit after the completion.)
	{id: "report.limit-after-done", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "limit-after-done", want: confirmed, hit: true},
	// The same in the scenario's own tools (added with the fix; the old code
	// has no final read): the final read of the report fails beside a record
	// of a failed write, the read of a kept read end beside a final answer.
	{id: "report.epipe-read-fails", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "f1-final-read-fails", want: invalid, hit: true},
	{id: "report.ok-drain-fails", test: "TestPipeF1SkipDrain", scenario: "T1.pipe.F1-skip-drain", inject: "f1-drain-fails", want: invalid, hit: true},

	// Review 082 B1-Ra: a stop's result, whenever it comes. Its error must
	// make the run invalid even when it comes after the run returned; a
	// stop that ends cleanly, early or late, must not.
	{id: "stop.healthy", test: "TestInnerStopError", scenario: "gate.stop-error", want: confirmed},
	{id: "stop.ok-after-run", test: "TestInnerStopError", scenario: "gate.stop-error", inject: "stop-ok-late", want: confirmed, hit: true},
	{id: "stop.error-in-run", test: "TestInnerStopError", scenario: "gate.stop-error", inject: "stop-error-early", want: invalid, hit: true},
	{id: "stop.error-after-run", test: "TestInnerStopError", scenario: "gate.stop-error", inject: "stop-error-late", want: invalid, hit: true},
	{id: "stop.never-done", test: "TestInnerStopError", scenario: "gate.stop-error", inject: "stop-never-done", want: invalid, hit: true},
	{id: "stop.error-after-copier", test: "TestInnerStopError", scenario: "gate.stop-error", inject: "stop-error-after-copier", want: invalid, hit: true},
	{id: "stop.error-one-of-two", test: "TestInnerStopError", scenario: "gate.stop-error", inject: "stop-error-one-of-two", want: invalid, hit: true},

	// Review 095 B1: one read of a new observation entry fails, the reads
	// after it succeed. The failure must make the run invalid, whatever the
	// legitimate value it could pass for (a NULL exit code, a refusal by
	// the baseline, an ended process). The controls: each experiment's own
	// healthy run and its planned faults stay confirmed.
	{id: "control.p1-alive", test: "TestP1/alive.1", scenario: "P1.alive.1", want: confirmed},
	{id: "control.p1-healthy", test: "TestP1/healthy.1", scenario: "P1.healthy.1", want: confirmed},
	{id: "p1.baseline-unread-once", test: "TestP1/exited.1", scenario: "P1.exited.1", inject: "p1-baseline-unread-once", want: invalid, goos: []string{"linux"}, hit: true, because: "observation failed: the barrier's baseline"},
	{id: "p1.observe-fail-once", test: "TestP1/healthy.1", scenario: "P1.healthy.1", inject: "p1-observe-fail-once", want: invalid, hit: true, because: "observation failed: the healthy control's"},
	{id: "p1.item-wait-fail-once", test: "TestP1/exited.1", scenario: "P1.exited.1", inject: "p1-item-wait-fail-once", want: invalid, goos: []string{"windows"}, hit: true, because: "observation failed: the wait on the handed handle"},
	{id: "control.p2-normal", test: "TestP2/normal.1", scenario: "P2.normal.1", want: confirmed, goos: []string{"linux"}},
	{id: "control.p2-denied", test: "TestP2/waitid.EPERM", scenario: "P2.waitid.EPERM", want: confirmed, goos: []string{"linux"}},
	{id: "control.p2-set0", test: "TestP2/prctl.set0", scenario: "P2.prctl.set0", want: confirmed, goos: []string{"linux"}},
	{id: "p2.check-poll-fail-once", test: "TestP2/normal.1", scenario: "P2.normal.1", inject: "p2-check-poll-fail-once", want: invalid, goos: []string{"linux"}, hit: true, because: "unplanned result of poll(check pidfd"},
	{id: "p2.close-fail-once", test: "TestP2/waitid.EPERM", scenario: "P2.waitid.EPERM", inject: "p2-close-fail-once", want: invalid, goos: []string{"linux"}, hit: true, because: "unplanned result of close(own pidfd)"},
	{id: "p2.status-unreadable-once", test: "TestP2/normal.1", scenario: "P2.normal.1", inject: "p2-status-unreadable-once", want: invalid, goos: []string{"linux"}, hit: true, because: "observation failed: the outer's read of the experiment process's status"},
	{id: "control.p4-resume-gt1", test: "TestP4/resume.gt1.1", scenario: "P4.resume.gt1.1", want: confirmed, goos: []string{"windows"}},
	{id: "control.p4-pre-unconfirmed", test: "TestP4/pre.unconfirmed.1", scenario: "P4.pre.unconfirmed.1", want: confirmed, goos: []string{"windows"}},
	{id: "control.p4-anomaly-unconfirmed", test: "TestP4/anomaly.unconfirmed.1", scenario: "P4.anomaly.unconfirmed.1", want: confirmed, goos: []string{"windows"}},
	{id: "p4.exitcode-fail-once", test: "TestP4/resume.gt1.1", scenario: "P4.resume.gt1.1", inject: "p4-exitcode-fail-once", want: invalid, goos: []string{"windows"}, hit: true, because: "observation failed: the exit code of the main process"},
	{id: "p4.exitcode-fail-once-healthy", test: "TestP4/healthy.1", scenario: "P4.healthy.1", inject: "p4-exitcode-fail-once", want: invalid, goos: []string{"windows"}, hit: true, because: "observation failed: the exit code of the main process"},
	{id: "p4.outer-exitcode-fail-once", test: "TestP4/pre.assign.1", scenario: "P4.pre.assign.1", inject: "p4-exitcode-fail-once", want: invalid, goos: []string{"windows"}, hit: true, because: "observation failed: the outer's read of the exit code"},
	{id: "p4.anomaly-wait-fail-once", test: "TestP4/resume.gt1.1", scenario: "P4.resume.gt1.1", inject: "p4-anomaly-wait-fail-once", want: invalid, goos: []string{"windows"}, hit: true, because: "observation failed: waiting for the main process"},
	{id: "p4.outer-wait-fail-once", test: "TestP4/pre.assign.1", scenario: "P4.pre.assign.1", inject: "p4-outer-wait-fail-once", want: invalid, goos: []string{"windows"}, hit: true, because: "observation failed: the outer's wait on"},
	{id: "p4.item-wait-fail-once", test: "TestP4/pre.unconfirmed.1", scenario: "P4.pre.unconfirmed.1", inject: "p4-item-wait-fail-once", want: invalid, goos: []string{"windows"}, hit: true, because: "observation failed: the wait on the handed handle"},

	// Review 096 B1-R: the clear of the subreaper at the end of an exclusive
	// interval fails once; only its result is replaced, the real clear is
	// made. The run must be invalid for that failure, whichever caller holds
	// the interval and whatever it does with the result: P-1's barrier and
	// its inner call, and T2's callers that count, check or drop it. With
	// it, the callback ending early through a failure of its own; and the
	// other steps of the interval, an unread baseline and a failed set. The
	// controls: the healthy runs, and a baseline read and found not empty.
	{id: "control.p1-exited", test: "TestP1/exited.1", scenario: "P1.exited.1", want: confirmed},
	{id: "control.t2-gate", test: "TestExclusiveGate", scenario: "T2.gate.exclusive-interval", want: confirmed, goos: []string{"linux"}},
	{id: "control.t2-bypass", test: "TestExclusiveBypass", scenario: "T2.gate.bypass", want: confirmed, goos: []string{"linux"}},
	{id: "p1.barrier-clear-fail-once", test: "TestP1/healthy.1", scenario: "P1.healthy.1", inject: "p1-barrier-clear-fail-once", want: invalid, goos: []string{"linux"}, hit: true, because: "the subreaper could not be cleared at the end of the interval (p1.barrier)", alsoBecause: "observation failed: the barrier's interval"},
	{id: "p1.inner-clear-fail-once", test: "TestP1/healthy.1", scenario: "P1.healthy.1", inject: "p1-inner-clear-fail-once", want: invalid, goos: []string{"linux"}, hit: true, because: "the subreaper could not be cleared at the end of the interval (p1.inner)", alsoBecause: "the exclusive interval: "},
	{id: "p1.observer-and-clear-fail-once", test: "TestP1/healthy.1", scenario: "P1.healthy.1", inject: "p1-observer-and-clear-fail-once", want: invalid, goos: []string{"linux"}, hit: true, because: "the subreaper could not be cleared at the end of the interval (p1.inner)", alsoBecause: "exit observer: injected"},
	{id: "t2.clear-fail-once-counted", test: "TestExclusiveGate", scenario: "T2.gate.exclusive-interval", inject: "t2-clear-fail-once", want: invalid, goos: []string{"linux"}, hit: true, because: "the subreaper could not be cleared at the end of the interval (T2)"},
	{id: "t2.clear-fail-once-checked", test: "TestAdoptionChain", scenario: "T2.adoption.chain", inject: "t2-clear-fail-once", want: invalid, goos: []string{"linux"}, hit: true, because: "the subreaper could not be cleared at the end of the interval (T2)"},
	{id: "t2.clear-fail-once-dropped", test: "TestExclusiveBypass", scenario: "T2.gate.bypass", inject: "t2-clear-fail-once", want: invalid, goos: []string{"linux"}, hit: true, because: "the subreaper could not be cleared at the end of the interval (T2)"},
	{id: "t2.baseline-unread-once", test: "TestExclusiveBaselineLive", scenario: "T2.baseline.live-child", inject: "t2-baseline-unread-once", want: invalid, goos: []string{"linux"}, hit: true, because: "the interval's baseline cannot be read (T2)"},
	{id: "t2.set1-fail-once", test: "TestExclusiveGate", scenario: "T2.gate.exclusive-interval", inject: "t2-set1-fail-once", want: invalid, goos: []string{"linux"}, hit: true, because: "the subreaper could not be set at the start of the interval (T2)"},

	// Review 096 N-096-2: P-2's own classification. The experiment reports
	// the filter as not installable without making the real install: a
	// valid report of it, while the outer waits for the check's
	// registration, is unsupported, not a registration that timed out. A
	// report that is withheld, truncated, another process's, or that
	// contradicts it (steps were made, a check was started) never passes
	// for one; a check that was started stays registered, judged and
	// cleaned up. The experiment's read of its own /proc entry that cannot
	// be read once is a failed observation, whatever succeeds after it, not
	// the capability found absent (refuted).
	{id: "control.p2-set1", test: "TestP2/prctl.set1", scenario: "P2.prctl.set1", want: confirmed, goos: []string{"linux"}},
	{id: "p2.unsupported-once", test: "TestP2/prctl.set0", scenario: "P2.prctl.set0", inject: "p2-unsupported-once", want: unsupported, goos: []string{"linux"}, hit: true, because: "the filter cannot be installed here: injected"},
	{id: "p2.unsupported-withheld-once", test: "TestP2/prctl.set0", scenario: "P2.prctl.set0", inject: "p2-unsupported-withheld-once", want: invalid, goos: []string{"linux"}, hit: true, because: "only 0 of 1 pid records"},
	{id: "p2.unsupported-truncated-once", test: "TestP2/prctl.set0", scenario: "P2.prctl.set0", inject: "p2-unsupported-truncated-once", want: invalid, goos: []string{"linux"}, hit: true, because: "the experiment's report does not parse"},
	{id: "p2.unsupported-foreign-once", test: "TestP2/prctl.set0", scenario: "P2.prctl.set0", inject: "p2-unsupported-foreign-once", want: invalid, goos: []string{"linux"}, hit: true, because: "load not established: a report of pid"},
	{id: "p2.unsupported-after-probe-once", test: "TestP2/prctl.set0", scenario: "P2.prctl.set0", inject: "p2-unsupported-after-probe-once", want: invalid, goos: []string{"linux"}, hit: true, because: "says unsupported but contradicts it"},
	{id: "p2.unsupported-after-check-once", test: "TestP2/prctl.set0", scenario: "P2.prctl.set0", inject: "p2-unsupported-after-check-once", want: invalid, goos: []string{"linux"}, hit: true, because: "says unsupported but contradicts it"},
	{id: "p2.self-read-unknown-once", test: "TestP2/prctl.set1", scenario: "P2.prctl.set1", inject: "p2-self-read-unknown-once", want: invalid, goos: []string{"linux"}, hit: true, because: "observation failed: the experiment process's read of its own /proc entry"},

	// P-3 (spec §19.12; plan in docs/wp2.5-prototype.md §15): one read of the
	// kernel's real answer is rewritten after the call. The candidate decoder
	// must reject it by the expected rule, for that reason alone, with its
	// cleanup verified and the layout probe of the same inner run confirmed.
	// The controls: the healthy runs of the same scenarios.
	{id: "control.p3-k-exit7", test: "TestP3$/^(layout.1|k.exit7.1)", scenario: "P3.k.exit7.1", want: confirmed, goos: []string{"darwin"}, exact: true},
	{id: "control.p3-w-exit7", test: "TestP3$/^(layout.1|w.exit7.1)", scenario: "P3.w.exit7.1", want: confirmed, goos: []string{"darwin"}, exact: true},
	{id: "control.p3-c-exit7", test: "TestP3$/^(layout.1|c.exit7.1)", scenario: "P3.c.exit7.1", want: confirmed, goos: []string{"darwin"}, exact: true},
	{id: "p3.w-sipid-once", test: "TestP3$/^(layout.1|w.exit7.1)", scenario: "P3.w.exit7.1", inject: "p3-w-sipid-once", want: invalid, goos: []string{"darwin"}, hit: true, exact: true, because: "planned corruption detected by W1"},
	{id: "p3.w-signo-once", test: "TestP3$/^(layout.1|w.exit7.1)", scenario: "P3.w.exit7.1", inject: "p3-w-signo-once", want: invalid, goos: []string{"darwin"}, hit: true, exact: true, because: "planned corruption detected by W1"},
	{id: "p3.w-code-once", test: "TestP3$/^(layout.1|w.exit7.1)", scenario: "P3.w.exit7.1", inject: "p3-w-code-once", want: invalid, goos: []string{"darwin"}, hit: true, exact: true, because: "planned corruption detected by W1"},
	{id: "p3.w-highbyte-once", test: "TestP3$/^(layout.1|w.exit7.1)", scenario: "P3.w.exit7.1", inject: "p3-w-highbyte-once", want: invalid, goos: []string{"darwin"}, hit: true, exact: true, because: "planned corruption detected by W3a"},
	{id: "p3.w-negative-once", test: "TestP3$/^(layout.1|w.exit7.1)", scenario: "P3.w.exit7.1", inject: "p3-w-negative-once", want: invalid, goos: []string{"darwin"}, hit: true, exact: true, because: "planned corruption detected by W3a"},
	{id: "p3.w-reread-once", test: "TestP3$/^(layout.1|w.exit7.1)", scenario: "P3.w.exit7.1", inject: "p3-w-reread-once", want: invalid, goos: []string{"darwin"}, hit: true, exact: true, because: "planned corruption detected by W5"},
	{id: "p3.k-nostatus-once", test: "TestP3$/^(layout.1|k.exit7.1)", scenario: "P3.k.exit7.1", inject: "p3-k-nostatus-once", want: invalid, goos: []string{"darwin"}, hit: true, exact: true, because: "planned corruption detected by K1"},
	{id: "p3.k-range-once", test: "TestP3$/^(layout.1|k.exit7.1)", scenario: "P3.k.exit7.1", inject: "p3-k-range-once", want: invalid, goos: []string{"darwin"}, hit: true, exact: true, because: "planned corruption detected by K1"},
	{id: "p3.k-stopped-once", test: "TestP3$/^(layout.1|k.exit7.1)", scenario: "P3.k.exit7.1", inject: "p3-k-stopped-once", want: invalid, goos: []string{"darwin"}, hit: true, exact: true, because: "planned corruption detected by K2"},
	{id: "p3.c-mismatch-once", test: "TestP3$/^(layout.1|c.exit7.1)", scenario: "P3.c.exit7.1", inject: "p3-c-mismatch-once", want: invalid, goos: []string{"darwin"}, hit: true, exact: true, because: "planned corruption detected by C1"},

	// Review 077 B1-R2-W: the witness itself fails. The outer check must
	// reject the evidence, whatever the inner run gives.
	{id: "witness.write-fail", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-write-fail", hit: true, reject: "witness-raised"},
	{id: "witness.short-write", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-short-write", hit: true, reject: "witness-malformed"},
	{id: "witness.identity-unknown", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-identity-unknown", hit: true, reject: "witness-raised"},
	{id: "witness.truncated", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-truncated", hit: true, reject: "witness-malformed"},
	{id: "witness.duplicate", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-duplicate", hit: true, reject: "witness-malformed"},
	{id: "witness.unknown-kind", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-unknown-kind", hit: true, reject: "witness-malformed"},
	{id: "witness.deleted", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-deleted", hit: true, reject: "witness-unreadable"},
	// The inner test process's own creation record cannot be written: its
	// seal reports the failure, and the scenario stops as invalid.
	{id: "witness.root-write-fail", test: "TestPipeSilentHolderReal", scenario: "T1.pipe.silent-holder", inject: "witness-root-write-fail", want: invalid, hit: true, reject: "witness-seal", also: []string{"witness-raised", "witness-count"}},
	// A process the inner run leaves alive without its own record, or
	// without any record; one that ends before any record (Windows: only
	// the Job's count of created processes can show it).
	{id: "witness.leak-unreceipted", test: "TestInnerLeak", scenario: "gate.leak", inject: "leak-unreceipted", want: invalid, hit: true, reject: "left", also: []string{"unaccounted"}},
	{id: "witness.leak-unrecorded", test: "TestInnerLeak", scenario: "gate.leak", inject: "leak-unrecorded", want: invalid, hit: true, reject: "left", also: []string{"unrecorded", "witness-count"}},
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
// The helper has a short limit of its own (8 s). With leak-unreceipted it
// is started as every process is (start), and leaves no record of its
// own; otherwise it is started past start, the way a start point missing
// from TestStartPoints would: no record at all. With unrecorded-ended it
// ends at once and is waited for here, so it is gone, unrecorded, before
// the outer check looks.
func TestInnerLeak(t *testing.T) {
	if os.Getenv(innerEnv) == "" {
		t.Skip("runs only as a gate regression")
	}
	s := begin(t, "gate", "gate.leak", fault, true, true)
	defer s.done()
	cmd := helper("linger", []string{limitEnv + "=8000"})
	injectHit("TestInnerLeak", os.Getpid())
	var err error
	if injected("leak-unreceipted") {
		err = start(cmd)
	} else {
		err = cmd.Start() // bypass: allowed in TestStartPoints for this line only
	}
	if err != nil {
		s.invalidf("helper did not start: %v", err)
	}
	s.launched(1)
	s.worked(1)
	if injected("unrecorded-ended") {
		s.expect("ended before any record", true, childOf(cmd).reap(10*time.Second))
	}
	s.flag("the scenario leaves its helper %d behind on purpose", cmd.Process.Pid)
	s.expect("left behind", true, true)
}

type innerRun struct {
	exit    int
	results map[string]result
	hits    int
	sites   []string // the site of each hit the inner test process printed
	seals   []string // the JSON of the WP25WITNESS lines
	proc    *innerProc
	err     error
}

// runInner runs one test function in a new process of this binary, with
// the fault injected, under a hard limit of its own and one of the
// parent's. env carries the witness file, the inject log and the raise.
func runInner(test, inject string, env []string) innerRun {
	self, err := os.Executable()
	if err != nil {
		return innerRun{err: err}
	}
	cmd := exec.Command(self, "-test.run", "^"+test+"$", "-test.count=1", "-test.v", "-test.timeout=150s")
	cmd.Env = append(append(os.Environ(), innerEnv+"=1", injectEnv+"="+inject), env...)
	cmd.WaitDelay = 5 * time.Second
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	p, err := launchInner(cmd, env[0])
	if err != nil {
		return innerRun{err: err}
	}
	r := innerRun{proc: p, results: map[string]result{}}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(180 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		r.err = fmt.Errorf("inner run did not end within 180 s")
	}
	r.exit = cmd.ProcessState.ExitCode()
	sc := bufio.NewScanner(bytes.NewReader(out.Bytes()))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if i := strings.Index(line, "WP25INJECT "); i >= 0 {
			r.hits++
			var h struct {
				Site string `json:"site"`
			}
			if json.Unmarshal([]byte(line[i+len("WP25INJECT "):]), &h) == nil {
				r.sites = append(r.sites, h.Site)
			}
		}
		if i := strings.Index(line, "WP25WITNESS "); i >= 0 {
			r.seals = append(r.seals, line[i+len("WP25WITNESS "):])
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

// outerFault is an outer-side fault of the tested path, for
// TestGateCombos; the zero value in TestGate.
type outerFault struct {
	name  string   // remedy-error, remedy-unconfirmed, census-error, record-unknown, forge-owner
	forge survivor // forge-owner: the process a forged creation record names
}

// The tested path's own steps, where a combination check (TestGateCombos)
// can inject an outer-side fault: the census, one read of a recorded
// process, the remediation. In TestGate the fault is "" and they are the
// plain calls.
func (p *innerProc) testedCensus() ([]survivor, int, error) {
	ss, total, err := p.census()
	if p.fault == "census-error" && err == nil {
		return nil, total, fmt.Errorf("%w: injected: the census could not be read", errCensus)
	}
	return ss, total, err
}

func (p *innerProc) testedAlive(w witnessRec) (bool, error) {
	a, err := recordAlive(w)
	if p.fault == "record-unknown" && !p.faulted {
		p.faulted = true
		return false, errors.New("injected: the state could not be read") // the real read was made; its answer is replaced
	}
	return a, err
}

func (p *innerProc) testedRemediate(ss []survivor) ([]survivor, error) {
	if p.fault == "remedy-error" && len(ss) > 0 {
		return ss, errors.New("injected: the remediation could not confirm its end") // nothing done
	}
	still, err := p.remediate(ss)
	if p.fault == "remedy-unconfirmed" && err == nil {
		return nil, errors.New("injected: the remediation's end could not be confirmed") // after the real one
	}
	return still, err
}

// toolClasses are failures of the outer check itself: the inner run it
// starts, its census and reads, the injection it needs, the remediation
// and the end it confirms, the insurance cleanup's contradiction. No
// expected rejection excuses them (Review 078 B1-R2-W-R).
var toolClasses = map[string]bool{"inner": true, "census": true, "state-unknown": true, "hit": true, "still-alive": true, "insurance": true, "ownership": true}

// sortCandidates decides which processes the outer check may act on by
// their PID (Review 079 B2). A process the census found is this round's by
// the census's own proof: this check's tag in its environment (Unix) or
// membership of the check's Job (Windows). A process alive only by a
// creation record is the inner run's claim, and the witness may be wrong:
// it is acted on only if verify proves it this round's now. A matching
// identity, a well-formed record, or its kind proves nothing about
// ownership. Every other claimed process is refused, never acted on, and
// each refusal is an "ownership" failure of the outer check, which no
// expected rejection excuses.
func sortCandidates(census, claimed []survivor, verify func(survivor) (bool, error)) (act []survivor, refused, problems []string) {
	act = append(act, census...)
	for _, v := range claimed {
		ok, err := verify(v)
		switch {
		case err != nil:
			refused = append(refused, v.key())
			problems = append(problems, fmt.Sprintf("ownership: process %s, alive by its record, cannot be shown to be this round's (%v): not acted on", v.key(), err))
		case !ok:
			refused = append(refused, v.key())
			problems = append(problems, fmt.Sprintf("ownership: process %s, alive by its record, is not shown to be this round's: not acted on", v.key()))
		default:
			act = append(act, v)
		}
	}
	return act, refused, problems
}

// judge decides a gate check on dimensions kept apart:
//
//   - the expected rejection: a problem of the class the case names (a
//     fault of the witness must be found, not just any rejection); with
//     no class named, no problem of the evidence at all;
//   - other problems of the evidence, of a class the case does not allow;
//   - the outer check's own failures (toolClasses);
//   - the remediation (not needed or succeeded; failed or not run) and
//     the final state of this round's processes (ended; alive or unknown).
//
// The check passes only when all of them hold, with the inner verdict and,
// where the case says so, a created process seen ending before its own
// record. Finding the expected rejection settles the first one only.
func judge(c gateCase, g *gateResult, verdictOK bool) {
	g.Expected, g.Unexpected, g.ToolErrors = false, nil, nil
	for _, p := range g.Problems {
		class, _, _ := strings.Cut(p, ":")
		switch {
		case toolClasses[class]:
			g.ToolErrors = append(g.ToolErrors, p)
		case c.reject != "" && class == c.reject:
			g.Expected = true
		case contains(c.also, class):
			// brought along by the case's own fault
		default:
			g.Unexpected = append(g.Unexpected, p)
		}
	}
	expected := c.reject == "" || g.Expected
	remedied := g.Remedy == "not-needed" || g.Remedy == "succeeded"
	g.Pass = verdictOK && expected && len(g.Unexpected) == 0 && len(g.ToolErrors) == 0 &&
		remedied && g.Final == "ended" && (!c.noReceipt || g.NoReceipt > 0)
}

func emptyFile(t *testing.T, dir, pattern string) string {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return f.Name()
}

// checkGate runs one gate regression and records it.
func checkGate(t *testing.T, c gateCase) {
	g := runGate(t, c, outerFault{})
	b, _ := json.Marshal(g)
	fmt.Println("WP25GATE " + string(b))
	gateMu.Lock()
	gateResults = append(gateResults, g)
	gateMu.Unlock()
	if !g.Pass {
		t.Errorf("%s: want %s (exit %s), reject %q, got %s (exit %d): %s; %v", c.id, c.want,
			map[bool]string{true: "0", false: "non-zero"}[c.want == confirmed], c.reject, g.Got, g.InnerExit, g.Reason, g.Problems)
	}
}

// runGate runs one gate regression and judges it in this order: the
// inner run's verdict; the state it left, read before anything is ended
// (the census, then each recorded process by its identity); only then the
// remediation of this round's own survivors, which never makes the check
// pass; then the witness itself and the inner run's registrations. f is an
// outer-side fault of the tested path (TestGateCombos), none in TestGate.
// After the judgment, the experiment's own insurance cleanup ends what the
// tested path left, and is recorded apart.
func runGate(t *testing.T, c gateCase, f outerFault) gateResult {
	begin := time.Now()
	dir := t.TempDir()
	wf := emptyFile(t, dir, "witness-*")
	il := emptyFile(t, dir, "inject-*")
	rw, err := newRaiseWatch()
	if err != nil {
		t.Fatal(err)
	}
	defer rw.close()
	r := runInner(c.test, c.inject, []string{witnessEnv + "=" + wf, injectLogEnv + "=" + il, rw.env()})
	g := gateResult{ID: c.id, Test: c.test, Inject: c.inject, Want: c.want, Reject: c.reject, InnerExit: r.exit, Hits: r.hits, Sites: r.sites, Total: -1}
	got, ok := r.results[c.scenario]
	verdictOK := false
	switch {
	case r.err != nil:
		g.Reason = "inner run: " + r.err.Error()
	case !ok:
		g.Reason = "the inner run gave no result for " + c.scenario
	default:
		g.Got, g.Reason, g.Launches, g.Work = got.Verdict, got.Reason, got.Launches, got.Work
		if c.hit {
			g.Observed, g.InnerCleanup = got.Observed, got.Cleanup
		}
		// A control must be accepted and exit 0; a broken run must be
		// rejected as invalid and make the diagnostic fail. A mechanism
		// verified as unavailable is unsupported and exits 0 (N-096-2):
		// it fails nothing, and is not counted as confirmed.
		verdictOK = c.want == "" || got.Verdict == c.want && (r.exit == 0) == (c.want == confirmed || c.want == unsupported)
		for _, b := range []string{c.because, c.alsoBecause} {
			if b != "" && !strings.Contains(got.Reason, b) {
				verdictOK = false
				g.Reason = fmt.Sprintf("the reason does not name %q: %s", b, got.Reason)
				break
			}
		}
		if c.exact && verdictOK {
			var others []string
			for id, o := range r.results {
				if id != c.scenario && o.Verdict != confirmed {
					others = append(others, id+" "+string(o.Verdict))
				}
			}
			sort.Strings(others)
			switch {
			case c.want == invalid && len(got.Invalid) != 1:
				verdictOK = false
				g.Reason = fmt.Sprintf("want exactly one reason, got %d: %s", len(got.Invalid), got.Reason)
			case !strings.HasPrefix(got.Cleanup, "verified"):
				verdictOK = false
				g.Reason = "the inner run's cleanup is not verified: " + got.Cleanup
			case len(others) > 0:
				verdictOK = false
				g.Reason = "other scenarios of the inner run are not confirmed: " + strings.Join(others, ", ")
			}
		}
	}
	if r.proc == nil {
		g.Problems = append(g.Problems, "inner: the inner run could not be started: "+fmt.Sprint(r.err))
		g.Remedy, g.Final = "not-run", "unknown"
		return finishGate(c, g, verdictOK, begin)
	}
	defer r.proc.close()
	r.proc.fault = f.name
	if f.name == "forge-owner" {
		// A well-formed creation record, written as the inner run's, that
		// names a process outside the inner run (Review 079 B2).
		appendFile(wf, fmt.Sprintf("created %d %s %d sleep\n", f.forge.pid, f.forge.start, r.proc.pid))
	}

	// The state the inner run left: first the census, independent of the
	// records; then every recorded process by its identity.
	survivors, total, cerr := r.proc.testedCensus()
	g.Total = total
	if cerr != nil {
		g.Problems = append(g.Problems, "census: "+cerr.Error())
	}
	file, rerr := os.ReadFile(wf)
	recs, _ := parseWitness(file)
	if rerr != nil {
		recs = nil
	}
	inCensus := map[string]bool{}
	for _, v := range survivors {
		inCensus[v.key()] = true
	}
	created := map[string]witnessRec{}
	receipts := map[string]bool{}
	for _, w := range recs {
		switch w.kind {
		case "created":
			created[w.key()] = w
		case "launched":
			receipts[w.key()] = true
			g.Receipts++
		}
	}
	g.Created = len(created)
	state := map[string]string{}
	var claimed []survivor // alive by a record only: the inner run's claim
	for k, w := range created {
		a, err := r.proc.testedAlive(w)
		switch {
		case err != nil:
			state[k] = "unknown"
			g.Problems = append(g.Problems, fmt.Sprintf("state-unknown: process %s: %v", k, err))
		case a:
			state[k] = "alive"
			if !inCensus[k] {
				g.Problems = append(g.Problems, fmt.Sprintf("census: process %s alive by its record, not found by the census", k))
				claimed = append(claimed, survivor{w.pid, w.start})
			}
		default:
			state[k] = "ended"
			if !receipts[k] {
				g.NoReceipt++
			}
		}
	}
	// Who may be acted on by PID (Review 079 B2): what the census found,
	// and a claimed process only once its ownership is proven now. The
	// others are refused, an ownership failure of the outer check.
	act, refused, oprobs := sortCandidates(survivors, claimed, r.proc.ownedNow)
	g.Problems = append(g.Problems, oprobs...)
	g.Refused = refused
	left := map[int]bool{}
	for _, v := range act {
		left[v.pid] = true
	}
	for _, v := range survivors {
		if _, ok := created[v.key()]; !ok {
			g.Unrecord = append(g.Unrecord, v.pid)
		}
	}
	for pid := range left {
		g.Left = append(g.Left, pid)
	}
	sort.Ints(g.Left)
	if len(g.Left) > 0 {
		g.Problems = append(g.Problems, fmt.Sprintf("left: %d processes of the inner run alive when it was over: %v", len(g.Left), g.Left))
	}
	if len(g.Unrecord) > 0 {
		g.Problems = append(g.Problems, fmt.Sprintf("unrecorded: %d of them without a creation record: %v", len(g.Unrecord), g.Unrecord))
	}

	// Remediation, after the state was recorded: this round's own
	// processes only, the ones whose ownership is proven (the Job's
	// members; on Unix, carrying this check's tag, with the same identity).
	// It never makes the check pass; whether it was needed, run, and
	// succeeded is kept as it is.
	known := act
	for _, v := range known {
		g.Acted = append(g.Acted, v.key())
	}
	sort.Strings(g.Acted)
	g.Remedy = "not-needed"
	var remedyErr error
	if len(g.Left) > 0 {
		g.Remedy = "not-run"
		still, err := r.proc.testedRemediate(known)
		remedyErr = err
		for _, v := range still {
			g.Still = append(g.Still, v.pid)
		}
		switch {
		case err != nil:
			g.Remedy, g.RemedyErr = "failed", err.Error()
		case len(g.Still) > 0:
			g.Remedy = "failed"
		default:
			g.Remedy = "succeeded"
		}
		if err != nil {
			g.Problems = append(g.Problems, "still-alive: the remediation could not confirm its end: "+err.Error())
		}
		if len(g.Still) > 0 {
			g.Problems = append(g.Problems, fmt.Sprintf("still-alive: %d processes still alive after the remediation: %v", len(g.Still), g.Still))
		}
	}
	// The final state of this round's processes, from the tested path's
	// own facts: alive if any is known alive, unknown if a read, the
	// census or the remediation's confirmation failed, or a claimed process
	// was refused (its ownership not proven), ended only if none of that
	// happened. An unknown state is never an end.
	unknown := 0
	for _, st := range state {
		if st == "unknown" {
			unknown++
		}
	}
	switch {
	case len(g.Still) > 0:
		g.Final = "alive"
	case cerr != nil || unknown > 0 || remedyErr != nil || len(refused) > 0:
		g.Final = "unknown"
	default:
		g.Final = "ended"
	}

	// The witness itself; a raise is read last, so a late one counts.
	g.Raised = rw.raised()
	_, wprobs := judgeWitness(evidence{file: file, readErr: rerr, seals: r.seals, inner: r.proc.pid, start: r.proc.start, raised: g.Raised, total: total})
	g.Problems = append(g.Problems, wprobs...)

	// The hits, from the inner test process's output and the helpers' own
	// log (not the witness file).
	if b, err := os.ReadFile(il); err == nil {
		g.Hits += strings.Count(string(b), "\n")
		for _, l := range strings.Split(string(b), "\n") {
			var h struct {
				Site string `json:"site"`
			}
			if l != "" && json.Unmarshal([]byte(l), &h) == nil {
				g.Sites = append(g.Sites, "helper: "+h.Site)
			}
		}
	}
	if c.hit && g.Hits == 0 {
		g.Problems = append(g.Problems, "hit: the fault never reached its site")
	}

	// Registration: each created process is registered by the inner run
	// (by PID and identity), or within the shortfall it declared. This is
	// a count, not a match of objects: the shortfall also counts expected
	// processes that were never created. An inner run of two scenarios (P-3's
	// layout probe and the case's own) registers through both.
	owned := map[string]bool{}
	for _, res := range r.results {
		for _, k := range res.Owned {
			owned[k] = true
		}
		g.Shortfall += res.Shortfall
	}
	for k, w := range created {
		if !owned[k] {
			g.Unowned = append(g.Unowned, fmt.Sprintf("%d:%s", w.pid, state[k]))
		}
	}
	sort.Strings(g.Unowned)
	if len(g.Unowned) > g.Shortfall {
		g.Problems = append(g.Problems, fmt.Sprintf("unaccounted: %d created processes not registered by the inner run, %d declared short", len(g.Unowned), g.Shortfall))
	}
	g = finishGate(c, g, verdictOK, begin)
	// The experiment's own insurance cleanup, after the judgment and apart
	// from the tested path: whatever of this round is still alive is ended.
	// It never makes a check pass. If it finds anything where the tested
	// path found the round ended, the tested path's account was wrong.
	g.Insurance = r.proc.insure(known)
	if g.Final == "ended" && g.Insurance != "nothing left" {
		g.Problems = append(g.Problems, "insurance: the tested path found the round ended, the insurance cleanup did not: "+g.Insurance)
		judge(c, &g, verdictOK)
	}
	return g
}

func finishGate(c gateCase, g gateResult, verdictOK bool, begin time.Time) gateResult {
	judge(c, &g, verdictOK)
	g.Millis = time.Since(begin).Milliseconds()
	return g
}
