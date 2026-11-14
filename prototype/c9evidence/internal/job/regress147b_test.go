package job

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/controls"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

// fakeJob is a job on the fixture repository with a fake clock and a given
// type checker and controls.
func fakeJob(t *testing.T, tc collect.TypeChecker, vs []controls.Variant) (*Job, *chain.FakeClock) {
	t.Helper()
	cfg, _ := newConfig(t, vs)
	clk := chain.NewFakeClock(time.Unix(1_000_000, 0))
	cfg.Clock, cfg.Start = clk, clk.Now()
	r, inv, guard, err := controls.FixtureRepo(filepath.Join(t.TempDir(), "fx"), runtime.GOOS, clk)
	if err != nil {
		t.Fatal(err)
	}
	cfg.R, cfg.Invoker, cfg.Guard = r, inv, guard
	if tc != nil {
		cfg.TypeCheck = tc
	}
	j, err := Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return j, clk
}

func finish(t *testing.T, j *Job) (chain.Final, bool) {
	t.Helper()
	done := make(chan chain.Final, 1)
	go func() { done <- j.Finish(0) }()
	select {
	case f := <-done:
		return f, true
	case <-time.After(bound):
		t.Error("Finish did not return")
		return chain.Final{}, false
	}
}

func notValid(f chain.Final) bool {
	return f.Conclusion != chain.Consistent && f.Conclusion != chain.Inconsistent && f.ExitCode != 0
}

// Healthy and timely: everything is published before W; the conclusion is 4.
func TestB1471HealthyTimely(t *testing.T) {
	j, _ := fakeJob(t, nil, controlsWithout(t, "LC-9"))
	j.RunDomains()
	f, ok := finish(t, j)
	if !ok {
		return
	}
	if f.Conclusion != chain.Consistent || f.ExitCode != 0 {
		t.Fatalf("final %v exit %d %v", f.Conclusion, f.ExitCode, f.Reasons)
	}
	s := j.Snapshot()
	if s.Work == nil || s.Work.State != WorkDone || !s.Work.Timely || s.Work.Abandoned {
		t.Errorf("work %+v", s.Work)
	}
	for _, r := range s.RRecords {
		if !r.Timely {
			t.Errorf("%s late", r.Result.Config)
		}
	}
}

// The last analysis unit completes after W: admitted is not timely; the
// configuration's result is late and the job is not valid.
func TestB1471LastUnitCompletesAfterW(t *testing.T) {
	tc := newBlockingTC()
	j, clk := fakeJob(t, tc, nil)
	tc.before = func() { clk.Set(j.W.Add(time.Second)) }
	t.Cleanup(tc.free)
	ran := make(chan struct{})
	go func() { j.RunDomains(); close(ran) }()
	if !waitFor(t, tc.entered, "the analysis entered the type check") {
		return
	}
	tc.free() // it returns a healthy result, but only after W
	if !waitFor(t, ran, "RunDomains returned") {
		return
	}
	waitFor(t, j.WorkDone(), "the work task finished")
	f, ok := finish(t, j)
	if !ok {
		return
	}
	if !notValid(f) {
		t.Fatalf("a result after W gave %v exit %d", f.Conclusion, f.ExitCode)
	}
	s := j.Snapshot()
	if len(s.RRecords) == 0 || s.RRecords[0].Timely {
		t.Fatalf("the configuration completed after W must be late: %+v", s.RRecords)
	}
}

// After the parent stopped waiting, the work task publishes a real error:
// it is kept (late) and the decided conclusion does not change.
func TestB1471RealErrorAfterTheParentStopped(t *testing.T) {
	tc := newBlockingTC()
	tc.fail = true
	j, clk := fakeJob(t, tc, nil)
	t.Cleanup(tc.free)
	ran := make(chan struct{})
	go func() { j.RunDomains(); close(ran) }()
	if !waitFor(t, tc.entered, "the analysis entered the type check") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	f, ok := finish(t, j)
	if !ok {
		return
	}
	if !notValid(f) {
		t.Fatalf("final %v exit %d", f.Conclusion, f.ExitCode)
	}
	tc.free()
	if !waitFor(t, j.WorkDone(), "the abandoned work task finished") {
		return
	}
	s := j.Snapshot()
	if !s.Work.Abandoned || s.Work.Timely {
		t.Errorf("work %+v", s.Work)
	}
	found := false
	for _, r := range s.RRecords {
		b5, _ := r.Result.Checks.Get("B5")
		for _, x := range b5.Failures {
			found = found || (x.Detail == "a real late error" && !r.Timely)
		}
	}
	if !found {
		t.Error("the late real error is not kept")
	}
}

// blockingWriter blocks every write until released: the diagnostic task
// stays in its first write.
type blockingWriter struct {
	release chan struct{}
	once    sync.Once
}

func (b *blockingWriter) Write(p []byte) (int, error) { <-b.release; return len(p), nil }
func (b *blockingWriter) free()                       { b.once.Do(func() { close(b.release) }) }

// The analysis and the diagnostic output both stay stuck: Finish still
// returns at D (the expiry exit), not 3 or 4.
func TestB1471DExitReachable(t *testing.T) {
	tc := newBlockingTC()
	cfg, _ := newConfig(t, nil)
	clk := chain.NewFakeClock(time.Unix(1_000_000, 0))
	cfg.Clock, cfg.Start, cfg.TypeCheck = clk, clk.Now(), tc
	r, inv, guard, err := controls.FixtureRepo(filepath.Join(t.TempDir(), "fx"), runtime.GOOS, clk)
	if err != nil {
		t.Fatal(err)
	}
	cfg.R, cfg.Invoker, cfg.Guard = r, inv, guard
	bw := &blockingWriter{release: make(chan struct{})}
	cfg.Stderr = bw
	t.Cleanup(tc.free)
	t.Cleanup(bw.free)
	j, err := Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ran := make(chan struct{})
	go func() { j.RunDomains(); close(ran) }()
	if !waitFor(t, tc.entered, "the analysis entered the type check") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W") {
		return
	}
	done := make(chan chain.Final, 1)
	go func() { done <- j.Finish(0) }()
	// The parent waits for the diagnostic task, which is stuck in its first
	// write; nothing but D ends the wait. The short pause only gives a
	// wrong early return the chance to show; it does not order events.
	select {
	case f := <-done:
		t.Fatalf("Finish returned before D: %v %v", f.Conclusion, f.Path)
	case <-time.After(200 * time.Millisecond):
	}
	clk.Set(j.D)
	select {
	case f := <-done:
		if !notValid(f) || f.Path != chain.PathExpiredFinal {
			t.Errorf("final %v exit %d path %v", f.Conclusion, f.ExitCode, f.Path)
		}
	case <-time.After(bound):
		t.Error("Finish did not return at D")
	}
}

// The F records keep each instance whole through the job's wiring.
func TestB1474FRecordsKeepInstances(t *testing.T) {
	var vs []controls.Variant
	vs = append(vs, controls.LC3()...)
	vs = append(vs, controls.LC5()[0])
	cfg, _ := newConfig(t, vs)
	j, err := Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	j.RunDomains()
	j.Finish(0)
	s := j.Snapshot()
	if len(s.FRecords) != len(vs) {
		t.Fatalf("%d F records", len(s.FRecords))
	}
	for _, r := range s.FRecords {
		if r.Outcome.Instance == nil || len(r.Outcome.Instance.Checks) == 0 {
			t.Errorf("%s %s: no instance", r.Variant.Control, r.Variant.Variant)
		}
	}
	last := s.FRecords[len(s.FRecords)-1].Outcome.Instance
	if len(last.Calls) != 3 || len(last.Records) == 0 {
		t.Errorf("LC5-0 instance lost its calls or records: %d calls %d records", len(last.Calls), len(last.Records))
	}
	b, _ := os.ReadFile(filepath.Join(cfg.RecordsDir, FileControls))
	if !strings.Contains(string(b), `"instance":{"Checks"`) {
		t.Error("controls.jsonl does not carry the instances")
	}
}

// A broken source chain makes the job invalid even with complete files.
// Since Review 148 R-a the work task verifies the chain as its last unit, so
// this record, appended after that verification, is not covered by it: the
// closeout reports it as appended after the verification (a defect) and
// never reuses the earlier success. A broken record the verification itself
// finds is TestB148RaBrokenRecordFoundByTheWork.
func TestB1474BrokenProvenanceIsInvalid(t *testing.T) {
	cfg, _ := newConfig(t, controlsWithout(t, "LC-9"))
	j, err := Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	j.RunDomains()
	c := spec.ConfigsFor(runtime.GOOS)[0].String()
	j.rec.Derived(c+"/closure", "B3", nil, map[string]any{"closure": []string{"forged"}}) // no parent
	f := j.Finish(0)
	if f.Conclusion != chain.Invalid {
		t.Fatalf("final %v %v", f.Conclusion, f.Reasons)
	}
}

// N-147-3: the run record's W and D are the job's.
func TestN1473WRecord(t *testing.T) {
	cfg, _ := newConfig(t, nil)
	cfg.R = ""
	j, err := Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	j.Finish(0)
	b, err := os.ReadFile(filepath.Join(cfg.RecordsDir, FileRaw))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, `"name":"identity/run"`) {
			continue
		}
		var it struct {
			Value map[string]any `json:"value"`
		}
		if err := json.Unmarshal([]byte(line), &it); err != nil {
			t.Fatal(err)
		}
		found = true
		if it.Value["W_offset"] != "1m50s" || it.Value["D_offset"] != "2m0s" || it.Value["W"] != j.W.Format(time.RFC3339Nano) || it.Value["D"] != j.D.Format(time.RFC3339Nano) {
			t.Errorf("run record %v; W %v D %v", it.Value, j.W, j.D)
		}
	}
	if !found {
		t.Fatal("no identity/run record")
	}
}

// N-147-1: LC-1 looks at every module B4 had to snapshot, not only at the
// snapshots that were taken.
func TestN1471LC1UsesTheModulesToCheck(t *testing.T) {
	mk := func(b3, b4 judge.Status, modules []string, snaps map[string]string, timely bool) RRecord {
		ck := judge.NewChecks(collect.CheckIDs())
		for _, x := range []struct {
			id string
			st judge.Status
		}{{"B3", b3}, {"B4", b4}} {
			ck.Enter(x.id, "")
			switch x.st {
			case judge.Hold:
				ck.Hold(x.id)
			case judge.Fail:
				ck.Fail(x.id, judge.Real(judge.J4, "m", "I-4", "h1 differs from go.sum", ""))
			default:
				ck.Unevaluated(x.id, "x")
			}
		}
		res := &collect.Result{Config: spec.Config{GOOS: "linux", GOARCH: "amd64"}, Checks: ck, Modules: modules, Snapshots: map[string]*modsnap.Snapshot{}}
		for m, h1 := range snaps {
			res.Snapshots[m] = &modsnap.Snapshot{Path: m, H1: h1}
		}
		return RRecord{Result: res, Timely: timely}
	}
	cases := []struct {
		name string
		r    RRecord
		want verdict.Judgement
	}{
		{"every module computed", mk(judge.Hold, judge.Hold, []string{"a", "b"}, map[string]string{"a": "h1:a", "b": "h1:b"}, true), verdict.Hit},
		{"one h1 differs but all computed", mk(judge.Hold, judge.Fail, []string{"a", "b"}, map[string]string{"a": "h1:a", "b": "h1:x"}, true), verdict.Hit},
		{"module b never snapshotted", mk(judge.Fail, judge.Fail, []string{"a", "b"}, map[string]string{"a": "h1:x"}, true), verdict.Pending},
		{"closure unknown", mk(judge.Unevaluated, judge.Unevaluated, nil, nil, true), verdict.Pending},
		{"late", mk(judge.Hold, judge.Hold, []string{"a"}, map[string]string{"a": "h1:a"}, false), verdict.Pending},
	}
	for _, c := range cases {
		if got := lc1([]RRecord{c.r}); got.Judgement != c.want {
			t.Errorf("%s: %s %v", c.name, got.Judgement, got.Reasons)
		}
	}
}
