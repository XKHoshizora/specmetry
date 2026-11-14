// Package controls holds the controls LC-2 to LC-9 of the F domain (plan
// §5.2, §5.4, §5.5): each variant prepares its own fixture (outer
// preparation), runs the tested tool code on it as its own instance, and
// is judged by the outer layer against a complete expectation written
// here, independently of the tested code. LC-1 is judged on R's records
// and LC-10 by TestMain; they are not variants.
package controls

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

// Outer registers the outer layer's own operations (preparation and
// cleanup) in the O domain: Begin registers one before it runs, and the
// returned func publishes its result.
type Outer interface {
	Begin(name string) (done func(err error))
}

// Env is what every variant gets.
type Env struct {
	GOOS     string
	TempRoot string // fixtures are created below it
	Outer    Outer
	// TypeCheck is the tool's type checker, run by LC-5 and LC-7.
	TypeCheck collect.TypeChecker
}

// Outcome is one tested instance's result as canonical lines, plus the
// outer layer's own state.
type Outcome struct {
	Entered     []string // checks with an entry record
	Lines       []string // canonical results of the tested instance
	Terminal    bool     // every task and resource of the instance is terminal
	PrepErr     error
	CleanupErr  error
	NotExecuted string // an applicable variant that did not run, and why
}

// Variant is one variant of a control.
type Variant struct {
	Control string
	Name    string
	// Applies is the applicability table: false only on a platform written
	// down as not applicable, with the reason (plan §5.1).
	Applies func(goos string) (bool, string)
	// MustEnter are the checks that need an entry record.
	MustEnter []string
	// Want is the complete expectation (plan §5.5): every result line, with
	// item, class, kind, source and count.
	Want []string
	Run  func(e *Env) Outcome
}

func always(string) (bool, string) { return true, "" }

// Judge is the outer judgement of one variant (plan §5.5).
func Judge(v Variant, goos string, o Outcome, ran bool) verdict.Variant {
	res := verdict.Variant{Control: v.Control, Variant: v.Name}
	if ok, why := v.Applies(goos); !ok {
		res.Judgement, res.Reasons = verdict.NotApplicable, []string{why}
		return res
	}
	switch {
	case !ran || o.NotExecuted != "":
		res.Judgement, res.Reasons = verdict.Miss, []string{"applicable but not executed: " + o.NotExecuted}
		return res
	case o.PrepErr != nil:
		res.Judgement, res.Reasons = verdict.Miss, []string{"outer preparation failed: " + o.PrepErr.Error()}
		return res
	case !o.Terminal:
		res.Judgement, res.Reasons = verdict.Pending, []string{"tested instance not terminal"}
		return res
	}
	var why []string
	entered := map[string]bool{}
	for _, e := range o.Entered {
		entered[e] = true
	}
	for _, m := range v.MustEnter {
		if !entered[m] {
			why = append(why, "no entry record for "+m)
		}
	}
	got, want := sortedCopy(o.Lines), sortedCopy(v.Want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		why = append(why, fmt.Sprintf("results differ from the complete expectation:\n  got  %q\n  want %q", got, want))
	}
	if len(why) > 0 {
		res.Judgement, res.Reasons = verdict.Miss, why
		return res
	}
	if o.CleanupErr != nil {
		res.Judgement, res.Reasons = verdict.Pending, []string{"outer cleanup failed or unknown: " + o.CleanupErr.Error()}
		return res
	}
	res.Judgement = verdict.Hit
	return res
}

// CheckLines renders a check registry as canonical lines: "ID status" for
// hold and unevaluated, and one "ID fail J-n item kind source[#k]" line per
// failure; sites' paths and details are not part of the comparison. A
// defect of the registry (an unknown or repeated check) is a line too, so it
// can never match an expectation.
func CheckLines(c *judge.Checks) []string {
	var out []string
	for _, it := range c.Snapshot() {
		switch it.Result.Status {
		case judge.Fail:
			for _, f := range it.Result.Failures {
				out = append(out, FailureLine(it.ID, f))
			}
		default:
			out = append(out, it.ID+" "+it.Result.Status.String())
		}
	}
	for _, d := range c.Defects() {
		out = append(out, "defect "+d)
	}
	return out
}

// FailureLine is one failure's canonical line.
func FailureLine(id string, f judge.Failure) string {
	src := string(f.Source)
	if f.Source == judge.SourceInjected {
		src = fmt.Sprintf("injected#%d", f.Inject)
	}
	return fmt.Sprintf("%s fail %s %s %s %s", id, f.Class, f.Item, f.Kind, src)
}

// Entered lists the checks with entry records.
func Entered(c *judge.Checks) []string {
	var out []string
	for _, e := range c.Entries() {
		out = append(out, e.Check)
	}
	return out
}

func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

// fixture is the outer layer's temporary directory of one variant.
type fixture struct {
	dir  string
	e    *Env
	name string
}

// prepare creates a fixture directory with files (slash paths), as one
// registered outer operation.
func prepare(e *Env, name string, files map[string]string) (*fixture, error) {
	done := e.Outer.Begin("prepare " + name)
	dir, err := os.MkdirTemp(e.TempRoot, "fx-")
	if err == nil {
		err = writeFiles(dir, files)
	}
	done(err)
	if err != nil {
		return nil, err
	}
	return &fixture{dir: dir, e: e, name: name}, nil
}

func writeFiles(dir string, files map[string]string) error {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, rel := range names {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(files[rel]), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// cleanup removes the fixture, as one registered outer operation.
func (f *fixture) cleanup() error {
	done := f.e.Outer.Begin("cleanup " + f.name)
	err := os.RemoveAll(f.dir)
	done(err)
	return err
}

// MemOuter is an Outer that only remembers, for tests and for runs
// without the record chain.
type MemOuter struct {
	mu   sync.Mutex
	ops  []string
	errs []string
	open int
}

// Begin registers one operation.
func (m *MemOuter) Begin(name string) func(error) {
	m.mu.Lock()
	m.ops = append(m.ops, name)
	m.open++
	m.mu.Unlock()
	var once sync.Once
	return func(err error) {
		once.Do(func() {
			m.mu.Lock()
			m.open--
			if err != nil {
				m.errs = append(m.errs, name+": "+err.Error())
			}
			m.mu.Unlock()
		})
	}
}

// State returns the operations, the failures and how many are still open.
func (m *MemOuter) State() (ops, errs []string, open int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.ops...), append([]string(nil), m.errs...), m.open
}
