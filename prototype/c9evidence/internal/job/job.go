// Package job wires one CI job of the limited evidence run (plan §4.1,
// §5.4, §6.4, §7.1): TestMain takes the single deadline D and checks T2
// (LC-10), the parent registers and starts the writer before m.Run, the one
// test function runs the F domain (the controls) and the R domain (the two
// configurations of this GOOS), and after m.Run the parent's closeout
// decides the conclusion and the exit status. Domain results are never
// reported through testing.T: the conclusion comes only from the published
// state of the three domains and the identity checks.
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
	running   bool
	ran       bool
	changed   chan struct{}
	variants  []verdict.Variant
	results   []*collect.Result
	refusals  map[string][]judge.Failure
	defects   []string
	rMissing  string
	lc1       verdict.Variant
	budget    *collect.Budget
	finalized bool
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
	j := &Job{cfg: cfg, changed: make(chan struct{}), refusals: map[string][]judge.Failure{}}
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
	j.rec.Raw("identity/run", map[string]any{"goos": cfg.GOOS, "R": cfg.R, "records": cfg.RecordsDir, "env": ids,
		"W": cfg.Start.Sub(cfg.Start).String(), "budget": chain.Budget.String(), "cleanup_margin": chain.CleanupMargin.String()})
	return j, nil
}

func (j *Job) publish() {
	close(j.changed)
	j.changed = make(chan struct{})
}

// terminal is the R and F registries' TerminalSource: they are terminal
// when the domains are not running (they ran, or never started).
func (j *Job) terminal() (bool, <-chan struct{}) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.running {
		return false, j.changed
	}
	return true, nil
}

// RunDomains runs the F domain, then the R domain. The test function calls
// it once.
func (j *Job) RunDomains() {
	j.mu.Lock()
	if j.running || j.ran {
		j.defects = append(j.defects, "RunDomains called twice")
		j.mu.Unlock()
		return
	}
	j.running = true
	j.budget = collect.NewBudget(j.cfg.Clock, j.cfg.Start, chain.Budget)
	j.publish()
	j.mu.Unlock()

	j.runControls()
	j.runR()

	j.mu.Lock()
	j.running, j.ran = false, true
	j.publish()
	j.mu.Unlock()
}

type outerAdapter struct{ o *chain.Outer }

func (a outerAdapter) Begin(name string) func(error) {
	op := a.o.RegisterOp(name)
	var once sync.Once
	return func(err error) { once.Do(func() { op.Done(err) }) }
}

func (j *Job) runControls() {
	root := j.cfg.FixtureRoot
	oa := outerAdapter{j.outer}
	if root == "" {
		done := oa.Begin("prepare fixture root")
		dir, err := os.MkdirTemp("", "c9evidence-fixtures-")
		done(err)
		if err != nil {
			j.mu.Lock()
			for _, v := range j.cfg.Controls {
				res := controls.Judge(v, j.cfg.GOOS, controls.Outcome{PrepErr: err}, true)
				j.variants = append(j.variants, res)
			}
			j.mu.Unlock()
			return
		}
		root = dir
		defer func() {
			done := oa.Begin("cleanup fixture root")
			done(os.RemoveAll(dir))
		}()
	}
	env := &controls.Env{GOOS: j.cfg.GOOS, TempRoot: root, Outer: oa, TypeCheck: j.cfg.TypeCheck}
	for _, v := range j.cfg.Controls {
		applicable, _ := v.Applies(j.cfg.GOOS)
		var o controls.Outcome
		ran := false
		if applicable {
			if j.budget.AdmitUnit("fixture " + v.Control + " " + v.Name) {
				o, ran = v.Run(env), true
			} else {
				o = controls.Outcome{NotExecuted: "not admitted: budget insufficient"}
			}
		}
		res := controls.Judge(v, j.cfg.GOOS, o, ran)
		j.log.Append(FileControls, mustLine(map[string]any{"control": v.Control, "variant": v.Name, "judgement": res.Judgement,
			"reasons": res.Reasons, "expectation": v.Want, "must_enter": v.MustEnter, "entered": o.Entered, "results": o.Lines,
			"terminal": o.Terminal, "prep_error": errText(o.PrepErr), "cleanup_error": errText(o.CleanupErr), "not_executed": o.NotExecuted}))
		j.mu.Lock()
		j.variants = append(j.variants, res)
		j.mu.Unlock()
	}
}

func (j *Job) runR() {
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
		res := collect.Collect(collect.Options{
			Config: c, R: j.cfg.R, Invoker: j.cfg.Invoker, Budget: j.budget, Guard: j.cfg.Guard, FS: j.cfg.ReadFS,
			MainModule: j.cfg.MainModule, Exceptions: j.cfg.Exceptions, SkipDirs: spec.RepoSkipDirs, Candidates: cands,
			Watch: j.cfg.Watch, Rec: j.rec, TypeCheck: j.cfg.TypeCheck,
		})
		refused := j.budget.Refused()[before:]
		items := res.Checks.Snapshot()
		j.rec.Derived(c.String()+"/checks", "A B S K", nil, map[string]any{"checks": items, "entries": res.Checks.Entries(),
			"defects": res.Checks.Defects(), "refused": refused, "consistent": res.Consistent, "differences": res.Diff})
		j.mu.Lock()
		j.results = append(j.results, res)
		j.refusals[c.String()] = refused
		j.mu.Unlock()
	}
	j.mu.Lock()
	j.lc1 = lc1(j.results)
	j.mu.Unlock()
}

// lc1 is the positive control of Hash1 on R's records (plan §5.2 LC-1):
// a hit when every configuration computed the h1 of every module of its
// closure; an h1 different from go.sum is R's J-4, not a miss. When R did
// not reach or complete I-4 the control is pending (R is incomplete
// anyway).
func lc1(results []*collect.Result) verdict.Variant {
	v := verdict.Variant{Control: "LC-1", Variant: "Hash1 positive control on R"}
	if len(results) == 0 {
		v.Judgement, v.Reasons = verdict.Pending, []string{"R collected no configuration"}
		return v
	}
	for _, r := range results {
		st := r.Checks.Status("B4")
		if st != judge.Hold && st != judge.Fail {
			v.Judgement, v.Reasons = verdict.Pending, []string{r.Config.String() + ": I-4 not evaluated"}
			return v
		}
		for p, s := range r.Snapshots {
			if s.H1 == "" {
				v.Judgement, v.Reasons = verdict.Pending, []string{r.Config.String() + ": no h1 computed for " + p}
				return v
			}
		}
	}
	v.Judgement = verdict.Hit
	return v
}

// state is the copy the closeout consumes.
type state struct {
	in      verdict.Inputs
	pending []string
}

func (j *Job) copyState(mRunCode int) state {
	j.mu.Lock()
	defer j.mu.Unlock()
	in := verdict.Inputs{T2OK: j.t2ok, MRunCode: mRunCode, Defects: append([]string(nil), j.defects...)}
	in.Defects = append(in.Defects, j.rec.Errors()...)
	judged := map[string]bool{}
	for _, v := range j.variants {
		judged[v.Control+"\x00"+v.Variant] = true
		in.Variants = append(in.Variants, v)
	}
	// An applicable variant that never ran (for example the test function
	// was not run) is a miss (plan §5.1).
	for _, v := range j.cfg.Controls {
		if judged[v.Control+"\x00"+v.Name] {
			continue
		}
		in.Variants = append(in.Variants, controls.Judge(v, j.cfg.GOOS, controls.Outcome{NotExecuted: "the domains did not run"}, false))
	}
	if j.ran && j.cfg.R != "" {
		in.Variants = append(in.Variants, j.lc1)
	}
	var pending []string
	if j.rMissing != "" {
		pending = append(pending, "R: "+j.rMissing)
	}
	for _, r := range j.results {
		rc := verdict.RConfig{Config: r.Config.String(), Compared: r.Checks.Status("K") == judge.Hold, Consistent: r.Consistent}
		rc.ABHold = r.Checks.AllHold(append(append([]string(nil), collect.AIDs...), collect.BIDs()...)...)
		for _, it := range r.Checks.Snapshot() {
			switch it.Result.Status {
			case judge.Fail:
				rc.Failures = append(rc.Failures, it.Result.Failures...)
			case judge.Unevaluated, judge.Unset:
				rc.Unevaluated = append(rc.Unevaluated, it.ID+": "+it.Result.Reason)
			}
		}
		rc.Failures = append(rc.Failures, j.refusals[r.Config.String()]...)
		for _, d := range r.Checks.Defects() {
			in.Defects = append(in.Defects, r.Config.String()+": "+d)
		}
		in.R = append(in.R, rc)
	}
	return state{in: in, pending: pending}
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
				case chain.OpFailed, chain.OpUnknown:
					in.OFailures = append(in.OFailures, fmt.Sprintf("%s: %s %s", op.Name, op.State, opDetail(op)))
				}
			}
			in.OPending = append(in.OPending, p.Copied.pending...)
			c, reasons := verdict.Decide(in)
			line := mustLine(map[string]any{"provisional": int(c), "name": c.String(), "reasons": reasons, "all_terminal": p.AllTerminal, "m.Run": p.MRunCode})
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
				if op.State == chain.OpFailed || op.State == chain.OpUnknown {
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
