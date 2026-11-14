// Package controls holds the controls LC-2 to LC-9 of the F domain (plan
// §5.2, §5.4, §5.5): each variant prepares its own fixture (outer
// preparation), runs the tested tool code on it as its own instance, and
// is judged by the outer layer against a complete expectation written
// here, independently of the tested code. LC-1 is judged on R's records
// and LC-10 by TestMain; they are not variants.
package controls

import (
	"errors"
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

// Outer is the outer layer's own operations (preparation and cleanup) in
// the O domain (Review 148 R-c): registration is not admission. Prepare and
// Cleanup decide, right before the operation starts, whether it may start:
// a preparation only before the work deadline W, a cleanup only before D.
// When admitted, the operation is registered and done publishes its real
// result (also when it comes late). When refused (ok false), the refusal is
// recorded as such, never as a success or as the operation's failure, and
// nothing may start. One admission covers one external file call: a
// preparation or a change that makes several calls asks for each of them
// right before it starts (Review 149 B-147-1-R-c/P). Hold hands the outer
// layer a directory it created; the outer layer holds it until a cleanup
// confirms its removal.
type Outer interface {
	Prepare(name string) (done func(err error), ok bool)
	Hold(name, dir string)
	Cleanup(name, dir string) (done func(err error), ok bool)
}

// ErrNotAdmitted is the result of an outer operation that was not admitted:
// it never started.
var ErrNotAdmitted = errors.New("not admitted")

var (
	errPrepareNotAdmitted = fmt.Errorf("outer preparation %w: the work deadline W has passed", ErrNotAdmitted)
	errCleanupNotAdmitted = fmt.Errorf("outer cleanup %w: D has passed", ErrNotAdmitted)
)

// Env is what every variant gets.
type Env struct {
	GOOS     string
	TempRoot string // fixtures are created below it
	Outer    Outer
	// TypeCheck is the tool's type checker, run by LC-5 and LC-7.
	TypeCheck collect.TypeChecker
	// MkdirTemp and RemoveAll are the outer layer's directory operations
	// (nil: os.MkdirTemp and os.RemoveAll); tests replace them.
	MkdirTemp func(dir, pattern string) (string, error)
	RemoveAll func(path string) error
	// MkdirAll, WriteFile, Remove, Mkdir and Symlink are the outer layer's
	// other file operations on fixtures (nil: the os functions with modes
	// 0o755 and 0o644); tests replace them.
	MkdirAll  func(path string) error
	WriteFile func(path string, data []byte) error
	Remove    func(path string) error
	Mkdir     func(path string) error
	Symlink   func(oldname, newname string) error
}

func (e *Env) mkdirAll(path string) error {
	if e.MkdirAll != nil {
		return e.MkdirAll(path)
	}
	return os.MkdirAll(path, 0o755)
}

func (e *Env) writeFile(path string, data []byte) error {
	if e.WriteFile != nil {
		return e.WriteFile(path, data)
	}
	return os.WriteFile(path, data, 0o644)
}

func (e *Env) remove(path string) error {
	if e.Remove != nil {
		return e.Remove(path)
	}
	return os.Remove(path)
}

func (e *Env) mkdir(path string) error {
	if e.Mkdir != nil {
		return e.Mkdir(path)
	}
	return os.Mkdir(path, 0o755)
}

func (e *Env) symlink(oldname, newname string) error {
	if e.Symlink != nil {
		return e.Symlink(oldname, newname)
	}
	return os.Symlink(oldname, newname)
}

func (e *Env) mkdirTemp(dir, pattern string) (string, error) {
	if e.MkdirTemp != nil {
		return e.MkdirTemp(dir, pattern)
	}
	return os.MkdirTemp(dir, pattern)
}

func (e *Env) removeAll(path string) error {
	if e.RemoveAll != nil {
		return e.RemoveAll(path)
	}
	return os.RemoveAll(path)
}

// Outcome is one tested instance's result as canonical lines, plus the
// outer layer's own state.
type Outcome struct {
	Entered []string // checks with an entry record
	Lines   []string // canonical results, generated from Instance
	// Instance is the tested instance's structured result, kept whole
	// (Review 147 B-147-4); Lines never replace it.
	Instance    *Instance
	Terminal    bool // every task and resource of the instance is terminal
	PrepErr     error
	CleanupErr  error
	NotExecuted string // an applicable variant that did not run, and why
}

// Instance is a tested instance's structured result: every check result
// with each failure's site, kind, source and detail; the entry records; the
// calls of the fake caller; the instance's own record lines; the admission
// refusals; and any further raw outcome of the tested code.
type Instance struct {
	Checks  []judge.Item
	Entries []judge.Entry
	Calls   []collect.CallResult `json:",omitempty"`
	Records []string             `json:",omitempty"`
	Refused []judge.Failure      `json:",omitempty"`
	Extra   map[string]any       `json:",omitempty"`
}

// instanceOf starts an instance from one check registry.
func instanceOf(ck *judge.Checks) *Instance {
	return &Instance{Checks: ck.Snapshot(), Entries: ck.Entries(), Extra: map[string]any{}}
}

// recordLines are a recorder's lines as text, without the LF.
func recordLines(items []collect.Item) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, strings.TrimSuffix(string(it.Line), "\n"))
	}
	return out
}

// Norm rewrites the parts of a line that depend on where and for which
// configuration the fixture ran: a fixture directory becomes "<fx>" and the
// configuration "<cfg>". Nothing else is removed from the line.
type Norm func(string) string

func normalizer(cfg string, dirs ...string) Norm {
	return func(s string) string {
		for _, d := range dirs {
			if d != "" {
				s = strings.ReplaceAll(s, d, "<fx>")
			}
		}
		if cfg != "" {
			s = strings.ReplaceAll(s, cfg, "<cfg>")
		}
		return s
	}
}

func noNorm(s string) string { return s }

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
		why := "outer preparation failed: " + o.PrepErr.Error()
		if errors.Is(o.PrepErr, ErrNotAdmitted) {
			why = o.PrepErr.Error()
		}
		res.Judgement, res.Reasons = verdict.Miss, []string{why}
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
		why := "outer cleanup failed or unknown: " + o.CleanupErr.Error()
		if errors.Is(o.CleanupErr, ErrNotAdmitted) {
			why = o.CleanupErr.Error()
		}
		res.Judgement, res.Reasons = verdict.Pending, []string{why}
		return res
	}
	res.Judgement = verdict.Hit
	return res
}

// CheckLines renders a check registry as canonical lines: "ID status" for
// hold and unevaluated, and one "ID fail J-n item kind source[#k] @ site"
// line per failure, every failure kept (none is filtered). Only the
// failure's free-text detail is left out of the comparison; the site stays,
// normalized by norm. A defect of the registry (an unknown or repeated
// check) is a line too, so it can never match an expectation.
func CheckLines(c *judge.Checks, norm Norm) []string {
	var out []string
	for _, it := range c.Snapshot() {
		switch it.Result.Status {
		case judge.Fail:
			for _, f := range it.Result.Failures {
				out = append(out, FailureLine(it.ID, f, norm))
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

// FailureLine is one failure's canonical line: check, class, item, kind,
// source (with the injection serial) and site.
func FailureLine(id string, f judge.Failure, norm Norm) string {
	src := string(f.Source)
	if f.Source == judge.SourceInjected {
		src = fmt.Sprintf("injected#%d", f.Inject)
	}
	return norm(fmt.Sprintf("%s fail %s %s %s %s @ %s", id, f.Class, f.Item, f.Kind, src, f.Site))
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

// step runs one external file call of the outer layer's preparation as
// one outer operation, admitted against W right before it starts (Review
// 149 B-147-1-R-c/P). A refused step starts nothing and returns an error
// that wraps ErrNotAdmitted and names the step; it is registered as refused,
// never as a success or as a real failure. An admitted step's real result is
// published when the call returns, also when that is late, and is returned
// unchanged.
func step(e *Env, op string, run func() error) error {
	done, ok := e.Outer.Prepare(op)
	if !ok {
		return fmt.Errorf("%w (%s)", errPrepareNotAdmitted, op)
	}
	err := run()
	done(err)
	return err
}

// prepare creates a fixture directory with files (slash paths). Each
// external call is its own admitted step: the directory's creation
// ("prepare <name>"), then for each file in name order the creation of its
// parent ("prepare <name>: mkdir parent of <file>") and its write
// ("prepare <name>: write <file>"). It stops at the first refusal or real
// error, so nothing starts after it. The directory is held by the outer
// layer from the moment it exists; a preparation that stops part way leaves
// it, with the files written so far, held (it is not cleaned up by this
// variant; the fixture root's cleanup is admitted against D as before).
func prepare(e *Env, name string, files map[string]string) (*fixture, error) {
	op := "prepare " + name
	var dir string
	if err := step(e, op, func() error {
		d, err := e.mkdirTemp(e.TempRoot, "fx-")
		if err == nil {
			dir = d
			e.Outer.Hold(name, d)
		}
		return err
	}); err != nil {
		return nil, err
	}
	if err := e.writeFiles(op, dir, files); err != nil {
		return nil, err
	}
	return &fixture{dir: dir, e: e, name: name}, nil
}

// changeStep is one external file call of a fixture change.
type changeStep struct {
	name string // e.g. "write lc3.go"
	run  func(e *Env, dir string) error
}

// change applies a change to an existing fixture, each call its own
// admitted step ("<op>: <step>"), stopping at the first refusal or real
// error.
func change(e *Env, op, dir string, steps []changeStep) error {
	for _, s := range steps {
		if err := step(e, op+": "+s.name, func() error { return s.run(e, dir) }); err != nil {
			return err
		}
	}
	return nil
}

// writeFiles writes a fixture's files through the environment's file
// operations, each call its own admitted step named after op.
func (e *Env) writeFiles(op, dir string, files map[string]string) error {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, rel := range names {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := step(e, op+": mkdir parent of "+rel, func() error { return e.mkdirAll(filepath.Dir(p)) }); err != nil {
			return err
		}
		if err := step(e, op+": write "+rel, func() error { return e.writeFile(p, []byte(files[rel])) }); err != nil {
			return err
		}
	}
	return nil
}

// writeFiles writes files with the os functions. Only controls.FixtureRepo
// uses it, to prepare the test repository; it is not an outer operation of
// the job.
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

// cleanup removes the fixture, as one admitted outer operation. A refused
// cleanup removes nothing and returns an error that wraps ErrNotAdmitted;
// the directory stays held by the outer layer.
func (f *fixture) cleanup() error {
	done, ok := f.e.Outer.Cleanup(f.name, f.dir)
	if !ok {
		return errCleanupNotAdmitted
	}
	err := f.e.removeAll(f.dir)
	done(err)
	return err
}

// MemOuter is an Outer that admits every operation and only remembers, for
// tests and for runs without the record chain.
type MemOuter struct {
	mu   sync.Mutex
	ops  []string
	errs []string
	open int
	held map[string]bool // the directories created and not confirmed removed
}

// Prepare admits and registers one preparation.
func (m *MemOuter) Prepare(name string) (func(error), bool) { return m.begin(name, nil), true }

// Hold records a created directory.
func (m *MemOuter) Hold(_, dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.held == nil {
		m.held = map[string]bool{}
	}
	m.held[dir] = true
}

// Cleanup admits and registers one cleanup; a nil result releases the
// directory.
func (m *MemOuter) Cleanup(name, dir string) (func(error), bool) {
	return m.begin("cleanup "+name, func(err error) {
		if err == nil {
			delete(m.held, dir)
		}
	}), true
}

// begin registers an operation; after runs under the mutex with its result.
func (m *MemOuter) begin(name string, after func(error)) func(error) {
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
			if after != nil {
				after(err)
			}
			m.mu.Unlock()
		})
	}
}

// Held returns the directories still held (created and not confirmed
// removed), sorted.
func (m *MemOuter) Held() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.held))
	for d := range m.held {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// State returns the operations, the failures and how many are still open.
func (m *MemOuter) State() (ops, errs []string, open int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.ops...), append([]string(nil), m.errs...), m.open
}
