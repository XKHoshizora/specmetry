// Package job wires one CI job of the limited evidence run (plan §4.1,
// §5.4, §6.4, §7.1): TestMain takes the single deadline D and checks T2
// (LC-10), the parent registers and starts the writer before m.Run, the one
// test function registers the domains' work task, starts it and waits only
// for its completion or W, and after m.Run the parent's closeout decides the
// conclusion and the exit status. Domain results are never reported through
// testing.T: the conclusion comes only from the published state of the
// three domains and the identity checks.
package job

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/controls"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

// The record files (plan §6.4; LE-6).
const (
	FileRaw      = "raw.jsonl"
	FileDerived  = "derived.jsonl"
	FileControls = "controls.jsonl"
	FileCloseout = "closeout.jsonl"
)

// Config is everything a job needs; FormalConfig builds the real one.
type Config struct {
	GOOS       string
	R          string // the analysed repository root; "" when not given
	RecordsDir string // must not exist yet
	Clock      chain.Clock
	Start      time.Time // TestMain's start
	WriteFS    chain.FS
	ReadFS     modsnap.FS
	Stderr     io.Writer
	Invoker    collect.Invoker
	Guard      func() collect.GuardValues
	TypeCheck  collect.TypeChecker
	Controls   []controls.Variant
	// FixtureRoot is where the controls' fixtures go ("" makes a fresh
	// temporary directory, as an outer operation).
	FixtureRoot string
	MainModule  string
	Exceptions  []spec.EmbedException
	Candidates  func(spec.Config) []spec.Candidate
	Watch       []string
	// Identity are values recorded as read: GITHUB_SHA,
	// GITHUB_WORKFLOW_REF, GITHUB_WORKFLOW_SHA (plan §6.3).
	Identity map[string]string
	T2       string
}

// Work states.
const (
	WorkRegistered = "registered"
	WorkRunning    = "running"
	WorkDone       = "done"
)

// WorkRecord is the domains' work task (implementation plan §12.14
// "同步的类型检查"; Review 147 B-147-1). It is registered before it starts
// and runs in its own goroutine; nothing in it can be interrupted, so the
// parent only waits for its published completion or W. When W comes first
// the parent stops waiting (Abandoned) and the task keeps running: every
// unit it has not started is refused (the budget admits nothing at or after
// W), its unfinished unit, its inputs and its late results stay held by the
// job until the process ends, and nothing it publishes later can restore
// validity.
//
// The necessary work (Review 148 R-b) is everything the task publishes
// before done, in this order: every control variant's F record (with its
// outer preparation and cleanup), every configuration's R result and checks
// record, LC-1, and the source-chain verification. Done is published last,
// so a timely done (before W, the parent still waiting) means every one of
// them was published before W. The closeout consumes the task's completion
// qualification as a whole: not done, abandoned, or done at or after W each
// keep the job from a valid conclusion, whatever is published later; each
// publication also keeps its own qualification.
type WorkRecord struct {
	State       string
	Registered  time.Time
	Started     time.Time
	Done        time.Time
	Timely      bool // done was published before W and before the parent stopped waiting
	Abandoned   bool
	AbandonedAt time.Time
	Current     string // the unit in progress
	done        chan struct{}
}

// FRecord is one judged control variant as the F domain holds it: the
// judgement, the whole outcome with its structured instance, and the
// qualification of its publication (Review 147 B-147-4).
type FRecord struct {
	Variant verdict.Variant
	Outcome controls.Outcome
	At      time.Time
	Timely  bool
}

// LC1Record is LC-1 as the job holds it, with the qualification of its
// publication (Review 148 R-b): a hit published at or after W, or after the
// parent stopped waiting, is not a timely hit.
type LC1Record struct {
	Variant verdict.Variant
	At      time.Time
	Timely  bool
}

// The verification states.
const (
	ProvRegistered = "registered"
	ProvRunning    = "running"
	ProvDone       = "done"
	ProvRefused    = "not admitted"
)

// ProvRecord is the published source-chain verification (Review 148 R-a;
// Review 147 B-147-4). The work task runs it as its last unit, admitted only
// before W, over every record item the job holds at that moment; the
// parent's closeout only consumes this record. Covered says which items
// were verified: the first Covered items of the job's recorder (items are
// only appended), the last of them being Last. An item appended after the
// verification is not covered by it, and the old result is never reused for
// it.
type ProvRecord struct {
	State      string
	Registered time.Time
	Started    time.Time
	At         time.Time // when the result was published
	Covered    int
	Last       collect.Ref
	Problems   []string
	Timely     bool
}

// HeldDir is a directory the outer layer created (the fixture root or a
// fixture); the job holds it until a cleanup confirms its removal (Review
// 148 R-c). State is "held", "cleanup started", "removed", "cleanup failed:
// <error>" or "cleanup not admitted: <reason>".
type HeldDir struct {
	Name    string
	Dir     string
	At      time.Time
	State   string
	Removed bool
}

// RRecord is one configuration's result as the R domain holds it, with the
// qualification of its publication: a result published at or after W, or
// after the parent stopped waiting, is late and cannot make the job valid.
type RRecord struct {
	Result  *collect.Result
	Refused []judge.Failure
	At      time.Time
	Timely  bool
}

// Job is one job's state. Everything the closeout consumes is copied under
// mu.
type Job struct {
	cfg    Config
	W, D   time.Time
	outer  *chain.Outer
	log    *chain.Log
	writer *chain.Writer
	rec    *collect.MemRecorder

	mu        sync.Mutex
	t2ok      bool
	changed   chan struct{}
	work      *WorkRecord
	fRecords  []FRecord
	rRecords  []RRecord
	defects   []string
	rMissing  string
	lc1       *LC1Record
	prov      *ProvRecord
	held      []HeldDir
	budget    *collect.Budget
	finalized bool

	s seams
}

// seams are the job's test seams. Begin sets the real ones; only this
// package's tests replace them, after Begin and before RunDomains.
type seams struct {
	// hook runs at the named points of the work task and of the parent's
	// wait ("work start", "parent wait", "R published <cfg>", "R done",
	// "before done"); a test may block in it to order events.
	hook      func(point string)
	verify    func([]collect.Item) []string
	mkdirTemp func(dir, pattern string) (string, error)
	removeAll func(path string) error
}

func (j *Job) at(point string) {
	if j.s.hook != nil {
		j.s.hook(point)
	}
}

// Begin takes D, checks T2 (LC-10), and registers and starts the writer,
// all before m.Run.
func Begin(cfg Config) (*Job, error) {
	if cfg.Clock == nil || cfg.WriteFS == nil || cfg.ReadFS == nil || cfg.Stderr == nil || cfg.Invoker == nil || cfg.Guard == nil {
		return nil, errors.New("job: incomplete configuration")
	}
	if cfg.RecordsDir == "" {
		return nil, errors.New("job: no records directory")
	}
	j := &Job{cfg: cfg, changed: make(chan struct{})}
	j.s = seams{verify: collect.VerifyProvenance, mkdirTemp: os.MkdirTemp, removeAll: os.RemoveAll}
	j.W, j.D = chain.Deadlines(cfg.Start)
	j.outer = chain.NewOuter(cfg.Clock, j.D)
	log, err := chain.NewLog(FileRaw, FileDerived, FileControls, FileCloseout)
	if err != nil {
		return nil, err
	}
	j.log = log
	j.rec = &collect.MemRecorder{Sink: func(it collect.Item) {
		file := FileRaw
		if it.File == "derived" {
			file = FileDerived
		}
		j.log.Append(file, bytes.TrimSuffix(it.Line, []byte("\n"))) // the log adds the LF
	}}
	w, err := j.outer.RegisterWriter(chain.WriterConfig{FS: cfg.WriteFS, Dir: cfg.RecordsDir, Log: log})
	if err != nil {
		return nil, err
	}
	j.writer = w
	if err := w.Start(); err != nil {
		return nil, err
	}

	n, sum, ok := spec.T2Binding(cfg.T2)
	j.t2ok = ok
	j.rec.Raw("identity/T2", map[string]any{"bytes": n, "sha256": sum, "binds": ok, "entry": "LC-10"})
	ids := map[string]string{}
	for k, v := range cfg.Identity {
		ids[k] = v
	}
	// W and D as instants and as offsets from the start (Review 147
	// N-147-3); they are the same values the budget and the closeout use.
	j.rec.Raw("identity/run", map[string]any{"goos": cfg.GOOS, "R": cfg.R, "records": cfg.RecordsDir, "env": ids,
		"start": cfg.Start.Format(time.RFC3339Nano), "W": j.W.Format(time.RFC3339Nano), "D": j.D.Format(time.RFC3339Nano),
		"W_offset": j.W.Sub(cfg.Start).String(), "D_offset": j.D.Sub(cfg.Start).String(),
		"budget": chain.Budget.String(), "cleanup_margin": chain.CleanupMargin.String()})
	return j, nil
}

func (j *Job) publish() {
	close(j.changed)
	j.changed = make(chan struct{})
}

// terminal is the R and F registries' TerminalSource: they are terminal
// when the work task never registered or published done.
func (j *Job) terminal() (bool, <-chan struct{}) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.work != nil && j.work.State != WorkDone {
		return false, j.changed
	}
	return true, nil
}

// RunDomains registers the domains' work task, starts it, and waits only
// for its published completion or W, whichever comes first. The test
// function calls it once; it returns at W at the latest, so m.Run returns
// and the parent's closeout and its D exit stay reachable even when an
// admitted analysis never returns.
func (j *Job) RunDomains() {
	j.mu.Lock()
	if j.work != nil {
		j.defects = append(j.defects, "RunDomains called twice")
		j.mu.Unlock()
		return
	}
	j.budget = collect.NewBudget(j.cfg.Clock, j.cfg.Start, chain.Budget)
	w := &WorkRecord{State: WorkRegistered, Registered: j.cfg.Clock.Now(), done: make(chan struct{})}
	j.work = w // registered before it starts
	j.publish()
	j.mu.Unlock()

	go j.runWork(w)

	j.at("parent wait")
	wch, stop := j.cfg.Clock.Timer(j.W)
	defer stop()
	select {
	case <-w.done:
	case <-wch:
		j.mu.Lock()
		if w.State != WorkDone {
			w.Abandoned, w.AbandonedAt = true, j.cfg.Clock.Now()
			j.publish()
		}
		j.mu.Unlock()
	}
}

func (j *Job) runWork(w *WorkRecord) {
	j.at("work start")
	j.mu.Lock()
	w.State, w.Started = WorkRunning, j.cfg.Clock.Now()
	j.publish()
	j.mu.Unlock()

	j.runControls(w)
	j.runR(w)
	j.at("R done")
	j.verifyRecords(w)

	j.at("before done")
	j.mu.Lock()
	now := j.cfg.Clock.Now()
	w.State, w.Done, w.Current = WorkDone, now, ""
	w.Timely = now.Before(j.W) && !w.Abandoned
	close(w.done)
	j.publish()
	j.mu.Unlock()
}

// begin marks the unit in progress.
func (j *Job) begin(w *WorkRecord, unit string) {
	j.mu.Lock()
	w.Current = unit
	j.publish()
	j.mu.Unlock()
}

// timelyLocked decides a publication's qualification at its lock: before W
// and while the parent still waits. Admission is not completion: a unit
// admitted before W whose result is published at or after W is late.
func (j *Job) timelyLocked(now time.Time) bool {
	return now.Before(j.W) && (j.work == nil || !j.work.Abandoned)
}

// outerAdapter is the F domain's outer layer on the job's outer registry
// (Review 148 R-c). Registration is not admission: each preparation and
// each cleanup is decided by the registry right before it starts, a
// preparation against the work deadline W and a cleanup against D; a
// refusal is registered as refused (never as a success) and nothing starts.
// Every directory the outer layer created is held by the job until a
// cleanup confirms its removal; a refused or failed cleanup leaves it held.
// An admitted operation's real result is published when it returns, also
// when that is late.
type outerAdapter struct{ j *Job }

func (a outerAdapter) Prepare(name string) (func(error), bool) {
	op, ok := a.j.outer.AdmitOp(name, a.j.W)
	if !ok {
		return nil, false
	}
	return opDone(op), true
}

func (a outerAdapter) Hold(name, dir string) {
	a.j.mu.Lock()
	defer a.j.mu.Unlock()
	a.j.held = append(a.j.held, HeldDir{Name: name, Dir: dir, At: a.j.cfg.Clock.Now(), State: "held"})
	a.j.publish()
}

func (a outerAdapter) Cleanup(name, dir string) (func(error), bool) {
	op, ok := a.j.outer.AdmitOp("cleanup "+name, a.j.D)
	if !ok {
		a.j.setHeld(dir, "cleanup not admitted: D has passed", false)
		return nil, false
	}
	a.j.setHeld(dir, "cleanup started", false)
	done := opDone(op)
	return func(err error) {
		done(err)
		if err != nil {
			a.j.setHeld(dir, "cleanup failed: "+err.Error(), false)
			return
		}
		a.j.setHeld(dir, "removed", true)
	}, true
}

func opDone(op *chain.Op) func(error) {
	var once sync.Once
	return func(err error) { once.Do(func() { op.Done(err) }) }
}

// setHeld updates a held directory's state; only a confirmed removal
// releases it.
func (j *Job) setHeld(dir, state string, removed bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for i := range j.held {
		if j.held[i].Dir == dir && !j.held[i].Removed {
			j.held[i].State, j.held[i].Removed = state, removed
		}
	}
	j.publish()
}

func (j *Job) runControls(w *WorkRecord) {
	root := j.cfg.FixtureRoot
	oa := outerAdapter{j}
	if root == "" {
		var dir string
		var err error
		if done, ok := oa.Prepare("prepare fixture root"); ok {
			dir, err = j.s.mkdirTemp("", "c9evidence-fixtures-")
			if err == nil {
				oa.Hold("fixture root", dir)
			}
			done(err)
		} else {
			err = fmt.Errorf("outer preparation of the fixture root %w: the work deadline W has passed", controls.ErrNotAdmitted)
		}
		if err != nil {
			for _, v := range j.cfg.Controls {
				o := controls.Outcome{PrepErr: err}
				j.publishVariant(v, o, true)
			}
			return
		}
		root = dir
		// The root's cleanup is admitted against D when the controls are
		// done, like any other cleanup: a deferred call does not bypass the
		// deadline.
		defer func() {
			if done, ok := oa.Cleanup("fixture root", dir); ok {
				done(j.s.removeAll(dir))
			}
		}()
	}
	env := &controls.Env{GOOS: j.cfg.GOOS, TempRoot: root, Outer: oa, TypeCheck: j.cfg.TypeCheck, MkdirTemp: j.s.mkdirTemp, RemoveAll: j.s.removeAll}
	for _, v := range j.cfg.Controls {
		applicable, _ := v.Applies(j.cfg.GOOS)
		var o controls.Outcome
		ran := false
		if applicable {
			unit := "fixture " + v.Control + " " + v.Name
			if j.budget.AdmitUnit(unit) {
				j.begin(w, unit)
				o, ran = v.Run(env), true
			} else {
				o = controls.Outcome{NotExecuted: "not admitted: budget insufficient"}
			}
		}
		j.publishVariant(v, o, ran)
	}
}

// publishVariant judges a variant and publishes its record with the
// qualification decided at this lock. The whole outcome, with its
// structured instance, is held in memory and appended to controls.jsonl.
func (j *Job) publishVariant(v controls.Variant, o controls.Outcome, ran bool) {
	res := controls.Judge(v, j.cfg.GOOS, o, ran)
	j.mu.Lock()
	now := j.cfg.Clock.Now()
	timely := j.timelyLocked(now)
	j.fRecords = append(j.fRecords, FRecord{Variant: res, Outcome: o, At: now, Timely: timely})
	j.publish()
	j.mu.Unlock()
	j.log.Append(FileControls, mustLine(map[string]any{"control": v.Control, "variant": v.Name, "judgement": res.Judgement,
		"reasons": res.Reasons, "expectation": v.Want, "must_enter": v.MustEnter, "entered": o.Entered, "results": o.Lines,
		"instance": o.Instance, "terminal": o.Terminal, "prep_error": errText(o.PrepErr), "cleanup_error": errText(o.CleanupErr),
		"not_executed": o.NotExecuted, "published_at": now.Format(time.RFC3339Nano), "timely": timely}))
}

func (j *Job) runR(w *WorkRecord) {
	if j.cfg.R == "" || !filepath.IsAbs(j.cfg.R) || filepath.Clean(j.cfg.R) != j.cfg.R {
		j.mu.Lock()
		j.rMissing = fmt.Sprintf("the analysed repository root R is not given as a clean absolute path (%q); it is never inferred", j.cfg.R)
		j.mu.Unlock()
		return
	}
	for _, c := range spec.ConfigsFor(j.cfg.GOOS) {
		before := len(j.budget.Refused())
		var cands []spec.Candidate
		if j.cfg.Candidates != nil {
			cands = j.cfg.Candidates(c)
		}
		j.begin(w, "configuration "+c.String())
		res := collect.Collect(collect.Options{
			Config: c, R: j.cfg.R, Invoker: j.cfg.Invoker, Budget: j.budget, Guard: j.cfg.Guard, FS: j.cfg.ReadFS,
			MainModule: j.cfg.MainModule, Exceptions: j.cfg.Exceptions, SkipDirs: spec.RepoSkipDirs, Candidates: cands,
			Watch: j.cfg.Watch, Rec: j.rec, TypeCheck: j.cfg.TypeCheck,
		})
		refused := j.budget.Refused()[before:]
		j.mu.Lock()
		now := j.cfg.Clock.Now()
		timely := j.timelyLocked(now)
		j.rRecords = append(j.rRecords, RRecord{Result: res, Refused: refused, At: now, Timely: timely})
		j.publish()
		j.mu.Unlock()
		j.at("R published " + c.String())
		j.rec.Derived(c.String()+"/checks", "A B S K", res.Refs.All(), map[string]any{"checks": res.Checks.Snapshot(), "entries": res.Checks.Entries(),
			"defects": res.Checks.Defects(), "refused": refused, "modules": res.Modules, "consistent": res.Consistent, "differences": res.Diff,
			"published_at": now.Format(time.RFC3339Nano), "timely": timely})
	}
	j.mu.Lock()
	now := j.cfg.Clock.Now()
	j.lc1 = &LC1Record{Variant: lc1(j.rRecords), At: now, Timely: j.timelyLocked(now)}
	j.publish()
	j.mu.Unlock()
}

// verifyRecords is the work task's last unit (Review 148 R-a): the
// source-chain verification of every record item the job holds, admitted
// only before W. It is registered before it is admitted, runs outside every
// lock, and publishes its result, its coverage and its qualification under
// the job's lock; the record is held by the job until the process ends, so
// a late result (a real error included) is kept. A refusal is published as
// such; nothing is skipped silently.
func (j *Job) verifyRecords(w *WorkRecord) {
	j.mu.Lock()
	p := &ProvRecord{State: ProvRegistered, Registered: j.cfg.Clock.Now()}
	j.prov = p
	j.publish()
	j.mu.Unlock()
	if !j.budget.AdmitUnit("source-chain verification") {
		j.mu.Lock()
		p.State, p.At = ProvRefused, j.cfg.Clock.Now()
		j.publish()
		j.mu.Unlock()
		return
	}
	j.begin(w, "source-chain verification")
	items := j.rec.Items()
	j.mu.Lock()
	p.State, p.Started = ProvRunning, j.cfg.Clock.Now()
	j.publish()
	j.mu.Unlock()
	problems := j.s.verify(items)
	j.mu.Lock()
	now := j.cfg.Clock.Now()
	p.State, p.At, p.Covered, p.Problems, p.Timely = ProvDone, now, len(items), problems, j.timelyLocked(now)
	if len(items) > 0 {
		p.Last = items[len(items)-1].Ref
	}
	j.publish()
	j.mu.Unlock()
}

// lc1 is the positive control of Hash1 on R's records (plan §5.2 LC-1;
// Review 147 N-147-1): a hit when, in every configuration, every module of
// the closure that B4 had to snapshot (Result.Modules, whether or not its
// snapshot was taken) has a computed h1. An h1 different from go.sum is R's
// J-4, not a miss. A module without a computed h1, a configuration whose
// closure is unknown, or a late result leaves the control pending, with
// every such module listed.
func lc1(rs []RRecord) verdict.Variant {
	v := verdict.Variant{Control: "LC-1", Variant: "Hash1 positive control on R"}
	if len(rs) == 0 {
		v.Judgement, v.Reasons = verdict.Pending, []string{"R collected no configuration"}
		return v
	}
	var missing []string
	for _, r := range rs {
		res := r.Result
		if st := res.Checks.Status("B3"); st != judge.Hold && st != judge.Fail {
			missing = append(missing, res.Config.String()+": the closure is not known (B3 not evaluated)")
			continue
		}
		if !r.Timely {
			missing = append(missing, res.Config.String()+": the result was published at or after W")
		}
		for _, m := range res.Modules {
			s := res.Snapshots[m]
			if s == nil || s.H1 == "" {
				missing = append(missing, res.Config.String()+": no h1 computed for "+m)
			}
		}
	}
	if len(missing) > 0 {
		v.Judgement, v.Reasons = verdict.Pending, missing
		return v
	}
	v.Judgement = verdict.Hit
	return v
}

// State is a copy of the job's held state, for records and tests.
type State struct {
	Work     *WorkRecord
	FRecords []FRecord
	RRecords []RRecord
	LC1      *LC1Record
	Prov     *ProvRecord
	Held     []HeldDir
}

// Snapshot returns a copy of the held state.
func (j *Job) Snapshot() State {
	j.mu.Lock()
	defer j.mu.Unlock()
	s := State{FRecords: append([]FRecord(nil), j.fRecords...), RRecords: append([]RRecord(nil), j.rRecords...), Held: append([]HeldDir(nil), j.held...)}
	if j.work != nil {
		w := *j.work
		s.Work = &w
	}
	if j.lc1 != nil {
		l := *j.lc1
		s.LC1 = &l
	}
	if j.prov != nil {
		p := *j.prov
		p.Problems = append([]string(nil), p.Problems...)
		s.Prov = &p
	}
	return s
}

// WorkDone is closed when the work task published done (nil before it
// registered).
func (j *Job) WorkDone() <-chan struct{} {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.work == nil {
		return nil
	}
	return j.work.done
}

// state is the copy the closeout consumes.
type state struct {
	in      verdict.Inputs
	pending []string
	held    []HeldDir
	summary map[string]any // the work task's and the verification's published state, for the closeout record
}

// copyState copies the published state under the job's lock. It only reads
// what the work task published (Review 148 R-a): it does not hash, decode
// or walk the record items, and it starts no analysis, so the expiry exits
// can call it after D.
func (j *Job) copyState(mRunCode int) state {
	j.mu.Lock()
	defer j.mu.Unlock()
	in := verdict.Inputs{T2OK: j.t2ok, MRunCode: mRunCode, Defects: append([]string(nil), j.defects...)}
	in.Defects = append(in.Defects, j.rec.Errors()...)
	var pending []string
	items := j.rec.Len()
	summary := map[string]any{"record_items": items}
	// The source chain (Review 147 B-147-4) as the work task's verification
	// published it (Review 148 R-a). Its real problems are defects whenever
	// they were published; a verification not run, refused, unfinished or
	// late gives no valid conclusion; items appended after it are not
	// covered, and the old result is not reused for them.
	switch p := j.prov; {
	case p == nil:
		pending = append(pending, "the source-chain verification is not registered")
	default:
		summary["verification"] = map[string]any{"state": p.State, "covered": p.Covered, "last": p.Last, "problems": len(p.Problems),
			"published_at": p.At.Format(time.RFC3339Nano), "timely": p.Timely}
		for _, x := range p.Problems {
			in.Defects = append(in.Defects, "provenance: "+x)
		}
		switch {
		case p.State != ProvDone:
			pending = append(pending, "the source-chain verification is "+p.State)
		case !p.Timely:
			pending = append(pending, "the source-chain verification published at or after W ("+p.At.Format(time.RFC3339Nano)+")")
		}
		if p.State == ProvDone && items != p.Covered {
			in.Defects = append(in.Defects, fmt.Sprintf("provenance: %d record items were appended after the source-chain verification, which covered %d", items-p.Covered, p.Covered))
		}
	}
	current := ""
	if w := j.work; w != nil {
		summary["work"] = map[string]any{"state": w.State, "timely": w.Timely, "abandoned": w.Abandoned}
		// The work task's completion qualification (Review 148 R-b): an
		// established abandonment or late completion stays after done.
		switch {
		case w.State != WorkDone && w.Abandoned:
			current = w.Current
			pending = append(pending, "the domains' work task did not finish before W (the parent stopped waiting; unit in progress: "+w.Current+")")
		case w.State != WorkDone:
			current = w.Current
			pending = append(pending, "the domains' work task is not finished (unit in progress: "+w.Current+")")
		case w.Abandoned:
			pending = append(pending, "the domains' work task finished at "+w.Done.Format(time.RFC3339Nano)+", after the parent stopped waiting at "+w.AbandonedAt.Format(time.RFC3339Nano)+": a later completion does not restore validity")
		case !w.Timely:
			pending = append(pending, "the domains' work task finished at or after W ("+w.Done.Format(time.RFC3339Nano)+")")
		}
	}
	judged := map[string]bool{}
	for _, r := range j.fRecords {
		v := r.Variant
		judged[v.Control+"\x00"+v.Variant] = true
		if !r.Timely && v.Judgement == verdict.Hit {
			v.Judgement, v.Reasons = verdict.Pending, []string{"published at or after W: a late hit is not a hit"}
		}
		in.Variants = append(in.Variants, v)
	}
	for _, v := range j.cfg.Controls {
		if judged[v.Control+"\x00"+v.Name] {
			continue
		}
		unit := "fixture " + v.Control + " " + v.Name
		if unit == current {
			// The variant in progress has not reached a terminal state.
			in.Variants = append(in.Variants, verdict.Variant{Control: v.Control, Variant: v.Name, Judgement: verdict.Pending, Reasons: []string{"still running at the snapshot"}})
			continue
		}
		// An applicable variant that never ran is a miss (plan §5.1).
		in.Variants = append(in.Variants, controls.Judge(v, j.cfg.GOOS, controls.Outcome{NotExecuted: "the work task did not run it"}, false))
	}
	if l := j.lc1; l != nil {
		v := l.Variant
		if !l.Timely && v.Judgement == verdict.Hit {
			v.Judgement, v.Reasons = verdict.Pending, []string{"published at or after W: a late hit is not a hit"}
		}
		in.Variants = append(in.Variants, v)
	} else if j.work != nil && j.cfg.R != "" {
		in.Variants = append(in.Variants, verdict.Variant{Control: "LC-1", Variant: "Hash1 positive control on R", Judgement: verdict.Pending, Reasons: []string{"R did not finish"}})
	}
	if j.rMissing != "" {
		pending = append(pending, "R: "+j.rMissing)
	}
	for _, r := range j.rRecords {
		res := r.Result
		rc := verdict.RConfig{Config: res.Config.String(), Compared: res.Checks.Status("K") == judge.Hold, Consistent: res.Consistent}
		rc.ABHold = res.Checks.AllHold(append(append([]string(nil), collect.AIDs...), collect.BIDs()...)...)
		for _, it := range res.Checks.Snapshot() {
			switch it.Result.Status {
			case judge.Fail:
				rc.Failures = append(rc.Failures, it.Result.Failures...)
			case judge.Unevaluated, judge.Unset:
				rc.Unevaluated = append(rc.Unevaluated, it.ID+": "+it.Result.Reason)
			}
		}
		rc.Failures = append(rc.Failures, r.Refused...)
		if !r.Timely {
			rc.Failures = append(rc.Failures, judge.Failure{Class: judge.J1, Item: res.Config.String(), Site: "publication of the configuration result",
				Kind: "published at or after W", Source: judge.SourceBudget, Detail: r.At.Format(time.RFC3339Nano)})
		}
		for _, d := range res.Checks.Defects() {
			in.Defects = append(in.Defects, res.Config.String()+": "+d)
		}
		in.R = append(in.R, rc)
	}
	return state{in: in, pending: pending, held: append([]HeldDir(nil), j.held...), summary: summary}
}

// Finish runs the parent's closeout after m.Run returned (plan §6.4 steps
// 1 to 5) and returns its decision; TestMain exits with Final.ExitCode.
func (j *Job) Finish(mRunCode int) chain.Final {
	j.mu.Lock()
	if j.finalized {
		j.mu.Unlock()
		return chain.Final{Conclusion: chain.Invalid, ExitCode: chain.ExitNonzero, Reasons: []string{"Finish called twice"}}
	}
	j.finalized = true
	j.mu.Unlock()
	return chain.Closeout(chain.Config[state]{
		Outer:    j.outer,
		Log:      j.log,
		W:        j.W,
		MRunCode: mRunCode,
		Terminal: []chain.TerminalSource{j.terminal},
		Copy:     func() state { return j.copyState(mRunCode) },
		Provisional: func(p chain.ProvisionalInput[state]) chain.Snapshot {
			in := p.Copied.in
			for _, op := range p.Outer.Ops {
				switch op.State {
				case chain.OpPending:
					in.OPending = append(in.OPending, op.Name)
				case chain.OpFailed, chain.OpUnknown, chain.OpRefused:
					in.OFailures = append(in.OFailures, fmt.Sprintf("%s: %s %s", op.Name, op.State, opDetail(op)))
				}
			}
			// A directory the outer layer created and no cleanup confirmed
			// removed stays the job's responsibility (Review 148 R-c).
			for _, h := range p.Copied.held {
				if !h.Removed {
					in.OPending = append(in.OPending, fmt.Sprintf("%s directory %s not confirmed removed: %s", h.Name, h.Dir, h.State))
				}
			}
			in.OPending = append(in.OPending, p.Copied.pending...)
			c, reasons := verdict.Decide(in)
			line := mustLine(map[string]any{"provisional": int(c), "name": c.String(), "reasons": reasons, "all_terminal": p.AllTerminal, "m.Run": p.MRunCode,
				"published": p.Copied.summary})
			var fails, pend []string
			for _, r := range reasons {
				if strings.Contains(r, "pending") {
					pend = append(pend, r)
				} else {
					fails = append(fails, r)
				}
			}
			sort.Strings(fails)
			return chain.Snapshot{Conclusion: chain.Conclusion(c), Records: []chain.Record{{File: FileCloseout, Line: line}}, Failures: fails, Pending: pend}
		},
		Level1: func(o chain.OuterSnapshot) bool {
			s := j.copyState(mRunCode)
			for _, op := range o.Ops {
				if op.State == chain.OpFailed || op.State == chain.OpUnknown || op.State == chain.OpRefused {
					s.in.OFailures = append(s.in.OFailures, op.Name)
				}
			}
			return verdict.Level1(s.in)
		},
		CloseoutFile: FileCloseout,
		Stderr:       j.cfg.Stderr,
	})
}

func opDetail(op chain.OpSnapshot) string {
	if op.Outcome != nil && op.Outcome.Err != nil {
		return op.Outcome.Err.Error()
	}
	return op.Reason
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func mustLine(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(map[string]string{"marshal_error": err.Error()})
	}
	return b // the log adds the LF
}
