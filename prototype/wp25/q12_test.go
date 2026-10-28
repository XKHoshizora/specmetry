package wp25

// The minimal model that P-1 and P-4 drive (docs/wp2.5-prototype.md §14;
// spec v7 §8.6, §19.1): the host's disabled state, its admission and its
// inventory of unconfirmed processes (Q-1), and the record decision of
// A12 and A6 (Q-2 (a)). Nothing here is product code: it implements only
// the branches the experiments exercise. TestQ12Classify checks the two
// decision tables on fixed inputs, without any process.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var errHostDisabled = errors.New("the host is disabled: no new check in this process")

// invItem is one entry of the host's inventory of unconfirmed processes:
// the very Cmd the inner call started, and on Windows the Runner's own
// process handle, held until the host exits (spec §19.3).
type invItem struct {
	kind   string // "unconfirmed-process"
	cmd    *exec.Cmd
	handle uintptr // the Runner's process handle (Windows); 0 elsewhere
	pid    int
}

// access counts what a check call did: a refused call must do none of it.
type access struct{ db, git, procs int }

// host is the per-process host object of spec §19.1, reduced to Q-1.
type host struct {
	mu       sync.Mutex
	disabled bool
	reason   string // the first reason; never reset
	active   int
	inv      []invItem
}

// admit is rule 1: under the lock, a disabled host refuses before
// anything is accessed; otherwise the call is counted as active.
func (h *host) admit() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.disabled {
		return fmt.Errorf("%w (%s)", errHostDisabled, h.reason)
	}
	h.active++
	return nil
}

func (h *host) leave() {
	h.mu.Lock()
	h.active--
	h.mu.Unlock()
}

// handOver is the F1 / A5 / A6 hand-over (rule 2, Q-1): the unconfirmed
// process enters the inventory and the host is disabled, in one critical
// section, before the call returns.
func (h *host) handOver(it invItem) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.inv = append(h.inv, it)
	if !h.disabled {
		h.disabled, h.reason = true, "unconfirmed_end"
	}
}

func (h *host) state() (bool, string, []invItem) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.disabled, h.reason, append([]invItem(nil), h.inv...)
}

// release empties the inventory, as the host's exit would; the disabled
// state does not change.
func (h *host) release() []invItem {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := h.inv
	h.inv = nil
	return out
}

// request is a new check call: admitted, its body runs and counts what it
// touches; refused, nothing runs.
func (h *host) request(body func(*access)) (access, error) {
	var a access
	if err := h.admit(); err != nil {
		return a, err
	}
	defer h.leave()
	body(&a)
	return a, nil
}

// caseOut names the §8.6 case a run ended in, for the Q-1 table.
type caseOut string

const (
	outNormal   caseOut = "normal"            // A10
	outF1       caseOut = "F1"                // A6 after a normal start
	outA4       caseOut = "A4"                // a failure before the resume, end confirmed
	outA5       caseOut = "A5"                // a failure before the resume, end not confirmed
	outA6Anom   caseOut = "A6-anomaly"        // a resume anomaly, end not confirmed
	outA12      caseOut = "A12"               // a resume anomaly, end confirmed
	outSpawnErr caseOut = "spawn_error"       // A3: no process
	outPreStart caseOut = "pre-start refusal" // A1: no process
	outO3       caseOut = "O3"
	outD21      caseOut = "D21"
)

// disables is Q-1 together with C-4 and D21: whether the caseOut disables
// the host, and the reason it records.
func disables(o caseOut) (bool, string) {
	switch o {
	case outF1, outA5, outA6Anom:
		return true, "unconfirmed_end"
	case outO3:
		return true, "o3"
	case outD21:
		return true, "stop_pending"
	}
	return false, ""
}

// record is the model's record decision for one run.
type record struct {
	Attempted   bool   `json:"attempted"`
	Result      string `json:"result,omitempty"`
	FailureKind string `json:"failure_kind,omitempty"`
	Qualifying  bool   `json:"qualifying"`
	ExitCode    *int   `json:"exit_code"`
	DurationMs  int64  `json:"duration_ms"`
	StartedAt   string `json:"started_at,omitempty"` // which clock reading it is
	HeadEnd     string `json:"head_end,omitempty"`
	TreeEnd     string `json:"tree_state_end,omitempty"`
	Batch       string `json:"batch"`
	Exit        int    `json:"exit"`
	Note        string `json:"note,omitempty"`
}

// anomalyRun is what a run of a Windows resume anomaly knows when it
// decides its record (A6 or A12).
type anomalyRun struct {
	treeConfirmed bool
	cancel        string // "", "before-write", "after-write"
	commit        string // after-write only: "committed", "not-committed", "uncertain"
	exitObtained  bool
	exitCode      int
	tAnom, tS3    time.Duration // on the run's monotonic clock
}

// decideAnomaly is Q-2 (a) for A12 and the D19 rule for A6, both with the
// resume-anomaly fields of spec §10.1: started_at sampled at the anomaly,
// duration t_S3 - t_anom, the exit code kept whenever it was reliably
// obtained, the end context not read. A cancel before the write
// transaction leaves A12 unrecorded (D8); A6 is recorded anyway (D19
// comes first). After the write began, the commit result decides.
func decideAnomaly(r anomalyRun) record {
	if r.treeConfirmed && r.cancel == "before-write" {
		return record{Batch: "stop", Exit: 6, Note: "cancelled before the write transaction: not recorded (D8)"}
	}
	rec := record{
		Attempted: true, Result: "error", FailureKind: "context_changed",
		DurationMs: (r.tS3 - r.tAnom).Milliseconds(), StartedAt: "t_anom",
		HeadEnd: "NULL", TreeEnd: "unknown", Batch: "stop", Exit: 6,
	}
	if r.exitObtained {
		c := r.exitCode
		rec.ExitCode = &c
	}
	switch {
	case r.cancel == "after-write" && r.commit == "uncertain":
		rec.Note = "the record may exist or not"
	case r.cancel == "after-write" && r.commit == "not-committed":
		rec.Attempted, rec.Note = true, "confirmed not committed: no record"
	}
	return rec
}

func TestQ12Classify(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	q1 := []struct {
		o      caseOut
		want   bool
		reason string
	}{
		{outF1, true, "unconfirmed_end"},
		{outA5, true, "unconfirmed_end"},
		{outA6Anom, true, "unconfirmed_end"},
		{outA4, false, ""},
		{outA12, false, ""},
		{outSpawnErr, false, ""},
		{outPreStart, false, ""},
		{outNormal, false, ""},
		{outO3, true, "o3"},
		{outD21, true, "stop_pending"},
	}
	for _, r := range q1 {
		t.Run("Q1/"+string(r.o), func(t *testing.T) {
			got, why := disables(r.o)
			if got != r.want || why != r.reason {
				t.Errorf("disables(%s) = %v %q, want %v %q", r.o, got, why, r.want, r.reason)
			}
		})
	}
	// The host: refused calls touch nothing; neither a release of the
	// inventory nor anything else lifts the disabled state.
	t.Run("Q1/host", func(t *testing.T) {
		h := &host{}
		if a, err := h.request(func(a *access) { a.db, a.git, a.procs = 1, 1, 1 }); err != nil || a != (access{1, 1, 1}) {
			t.Fatalf("a fresh host: %v %+v", err, a)
		}
		h.handOver(invItem{kind: "unconfirmed-process", pid: 1})
		h.handOver(invItem{kind: "unconfirmed-process", pid: 2})
		if a, err := h.request(func(a *access) { a.db = 1 }); !errors.Is(err, errHostDisabled) || a != (access{}) {
			t.Errorf("after the hand-over: %v %+v", err, a)
		}
		if _, why, items := h.state(); why != "unconfirmed_end" || len(items) != 2 {
			t.Errorf("state: %q, %d items", why, len(items))
		}
		h.release()
		if _, err := h.request(func(*access) {}); !errors.Is(err, errHostDisabled) {
			t.Errorf("after the release: %v", err)
		}
	})
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	seven := 7
	q2 := []struct {
		name string
		in   anomalyRun
		want record
	}{
		{"A12, exit obtained", anomalyRun{treeConfirmed: true, exitObtained: true, exitCode: 7, tAnom: ms(10), tS3: ms(25)},
			record{Attempted: true, Result: "error", FailureKind: "context_changed", ExitCode: &seven, DurationMs: 15, StartedAt: "t_anom", HeadEnd: "NULL", TreeEnd: "unknown", Batch: "stop", Exit: 6}},
		{"A12, exit not obtained", anomalyRun{treeConfirmed: true, tAnom: ms(10), tS3: ms(12)},
			record{Attempted: true, Result: "error", FailureKind: "context_changed", DurationMs: 2, StartedAt: "t_anom", HeadEnd: "NULL", TreeEnd: "unknown", Batch: "stop", Exit: 6}},
		{"A12, cancel before the write", anomalyRun{treeConfirmed: true, cancel: "before-write", exitObtained: true, exitCode: 7},
			record{Batch: "stop", Exit: 6, Note: "cancelled before the write transaction: not recorded (D8)"}},
		{"A12, cancel after the write, uncertain commit", anomalyRun{treeConfirmed: true, cancel: "after-write", commit: "uncertain"},
			record{Attempted: true, Result: "error", FailureKind: "context_changed", StartedAt: "t_anom", HeadEnd: "NULL", TreeEnd: "unknown", Batch: "stop", Exit: 6, Note: "the record may exist or not"}},
		{"A6, cancel before the write: D19 first", anomalyRun{cancel: "before-write", exitObtained: true, exitCode: 7, tAnom: ms(1), tS3: ms(3)},
			record{Attempted: true, Result: "error", FailureKind: "context_changed", ExitCode: &seven, DurationMs: 2, StartedAt: "t_anom", HeadEnd: "NULL", TreeEnd: "unknown", Batch: "stop", Exit: 6}},
	}
	for _, r := range q2 {
		t.Run("Q2/"+r.name, func(t *testing.T) {
			got, _ := json.Marshal(decideAnomaly(r.in))
			want, _ := json.Marshal(r.want)
			if string(got) != string(want) {
				t.Errorf("decideAnomaly:\n got %s\nwant %s", got, want)
			}
		})
	}
	// The goroutine evidence: a stack in a process-wait frame is found, a
	// plain one is not.
	t.Run("goroutines", func(t *testing.T) {
		gs := []gstack{
			{1, "goroutine 1 [running]:\nmain.main()\n"},
			{2, "goroutine 2 [syscall]:\nsyscall.Syscall6(...)\nos.(*Process).blockUntilWaitable(0xc0)\nos.(*Process).wait(0xc0)\nos.(*Process).Wait(...)\nos/exec.(*Cmd).Wait(0xc0)\n"},
			{9, "goroutine 9 [chan receive]:\ngithub.com/XKHoshizora/specmetry/prototype/wp25.(*scen).done(0xc0)\n"},
		}
		if w := waitingGoroutines(gs); len(w) != 1 || w[0] != 2 {
			t.Errorf("waiting: %v", w)
		}
		if f := freshProto(gs, 5, 0); len(f) != 1 || !strings.HasPrefix(f[0], "9 ") {
			t.Errorf("fresh: %v", f)
		}
	})
}

// gstack is one goroutine's stack, as runtime.Stack prints it.
type gstack struct {
	id    uint64
	stack string
}

// goroutines takes every goroutine's stack.
func goroutines() []gstack {
	buf := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	var out []gstack
	for _, part := range strings.Split(string(buf), "\n\n") {
		f := strings.Fields(strings.TrimPrefix(part, "goroutine "))
		if len(f) == 0 {
			continue
		}
		id, err := strconv.ParseUint(f[0], 10, 64)
		if err != nil {
			continue
		}
		out = append(out, gstack{id, part})
	}
	return out
}

// maxGoroutine is the highest goroutine ID now: goroutines created later
// have higher IDs.
func maxGoroutine() uint64 {
	var m uint64
	for _, g := range goroutines() {
		if g.id > m {
			m = g.id
		}
	}
	return m
}

// waitFrames are the frames of a goroutine blocked waiting for a process,
// which spec §19.3 forbids after F1.
var waitFrames = []string{
	"os.(*Process).Wait(", "os.(*Process).wait(", "os/exec.(*Cmd).Wait(",
	"syscall.Wait4(", "syscall.wait4(", "os.(*Process).blockUntilWaitable(",
	"os.(*Process).pidfdWait(",
}

// waitingGoroutines lists the goroutines in a process-wait frame.
func waitingGoroutines(gs []gstack) []uint64 {
	var out []uint64
	for _, g := range gs {
		for _, f := range waitFrames {
			if strings.Contains(g.stack, f) {
				out = append(out, g.id)
				break
			}
		}
	}
	return out
}

// freshProto lists the goroutines created after base that run this
// prototype's code, other than self: "id state first-prototype-frame".
func freshProto(gs []gstack, base, self uint64) []string {
	var out []string
	for _, g := range gs {
		if g.id <= base || g.id == self || !strings.Contains(g.stack, "specmetry/prototype/wp25.") {
			continue
		}
		lines := strings.Split(g.stack, "\n")
		state := ""
		if i := strings.IndexByte(lines[0], '['); i >= 0 {
			state = strings.TrimSuffix(lines[0][i:], ":")
		}
		frame := ""
		for _, l := range lines[1:] {
			if strings.Contains(l, "specmetry/prototype/wp25.") {
				frame = strings.TrimSpace(l)
				break
			}
		}
		out = append(out, fmt.Sprintf("%d %s %s", g.id, state, frame))
	}
	return out
}

// budget is an experiment's total deadline (docs/wp2.5-prototype.md
// §14.2): a run starts only while what is left covers its own limit.
type budget struct {
	name       string
	total, per time.Duration
	once       sync.Once
	start      time.Time
}

func (b *budget) allow() (bool, time.Duration) {
	b.once.Do(func() { b.start = time.Now() })
	left := b.total - time.Since(b.start)
	return left >= b.per, left
}

// pline prints one experiment's evidence before any remediation, so that
// it is kept whatever the remediation does.
func pline(id string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		b = []byte(strconv.Quote(err.Error()))
	}
	fmt.Println("WP25P " + id + " " + string(b))
}
