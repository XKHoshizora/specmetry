// Package collect runs the stages A and B, the third-party scan S and the
// comparison with the candidates K for one configuration (method record
// §6.2; plan §4.3, §5.3). Every check of the set E has exactly one result:
// hold, fail with J classes, or unevaluated; nothing outside E is ever
// recorded.
package collect

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/decode"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// B1Keys are the six keys B1 checks (method record §3.2).
var B1Keys = []string{"GOOS", "GOARCH", "CGO_ENABLED", "GOFLAGS", "GOWORK", "GOTOOLCHAIN"}

// B2Fields are the fields B2 checks (method record §3.3.3; GOPATH must be
// well-formed but is not consumed, so it is not a check).
var B2Fields = []string{"GOOS", "GOARCH", "GOROOT", "CgoEnabled", "UseAllFiles", "Compiler", "InstallSuffix", "BuildTags", "ToolTags", "ReleaseTags"}

// AIDs are the checks of stage A.
var AIDs = []string{"A1-a", "A1-b", "A1-c", "A2", "A3", "A4", "A5"}

// CheckIDs is the check set E of one configuration (plan §5.3 rule 1), in
// order: A, B1 per key, B2 per field, B3, B4, B5, S, K.
func CheckIDs() []string {
	ids := append([]string(nil), AIDs...)
	for _, k := range B1Keys {
		ids = append(ids, "B1/"+k)
	}
	for _, f := range B2Fields {
		ids = append(ids, "B2/"+f)
	}
	return append(ids, "B3", "B4", "B5", "S", "K")
}

// BIDs are the checks of stage B.
func BIDs() []string {
	var ids []string
	for _, k := range B1Keys {
		ids = append(ids, "B1/"+k)
	}
	for _, f := range B2Fields {
		ids = append(ids, "B2/"+f)
	}
	return append(ids, "B3", "B4", "B5")
}

// Options configure one configuration's collection.
type Options struct {
	Config  spec.Config
	R       string // the analysed repository root (not the tool's module)
	Invoker Invoker
	Budget  *Budget
	// Guard reads the A1-a values: ReadGuard(RealGuardSources()) in the
	// formal run; LC-5 hands fixed values (plan §5.3 LC5-0).
	Guard      func() GuardValues
	FS         modsnap.FS
	MainModule string
	Exceptions []spec.EmbedException
	SkipDirs   []string
	Candidates []spec.Candidate
	// Watch are import path prefixes of the decision draft §2.7 whose
	// presence in the closure is reported (LQ-6).
	Watch []string
	Rec   Recorder
	// Inject is the controlled read's seam (LC-3); nil in R.
	Inject modsnap.Injector
	// TypeCheck runs the loader, the type check and the scan; nil skips
	// them (B5 and S then record that the analysis was not run). The formal
	// run always sets it.
	TypeCheck TypeChecker
}

// Result is one configuration's outcome.
type Result struct {
	Config    spec.Config
	Checks    *judge.Checks
	Calls     []CallResult
	Guard     GuardValues
	Env       map[string]string
	Context   decode.Context
	Packages  []decode.Package
	GoSum     decode.GoSum
	Closure   []string
	Snapshots map[string]*modsnap.Snapshot // module path -> I-4
	Reader    *modsnap.Reader
	Analysis  *Analysis
	// K's outcome: whether the findings equal the candidates, and the
	// differences.
	Consistent bool
	Diff       []string
	// Modules are the dependency modules of the closure that B4 must
	// snapshot (whether or not their snapshot was taken), for LC-1.
	Modules []string
	// Refs are this configuration's record references (B-147-4): the raw
	// records the derived ones cite as their parents.
	Refs Refs
}

// Refs are a configuration's record references.
type Refs struct {
	RepoFiles   Ref
	Toolchain   Ref
	Calls       map[spec.CallKind]Ref
	I4          map[string]Ref // module path -> its raw I-4 record
	Closure     Ref
	Contexts    []Ref
	Consumption Ref
	Findings    Ref
	Comparison  Ref
}

// All returns every reference that was recorded, raw first, in a stable
// order: the parents of the configuration's checks record.
func (r Refs) All() []Ref {
	var out []Ref
	add := func(x Ref) {
		if x.Name != "" {
			out = append(out, x)
		}
	}
	add(r.RepoFiles)
	add(r.Toolchain)
	for _, k := range []spec.CallKind{spec.CallEnv, spec.CallContext, spec.CallList} {
		add(r.Calls[k])
	}
	mods := make([]string, 0, len(r.I4))
	for m := range r.I4 {
		mods = append(mods, m)
	}
	sort.Strings(mods)
	for _, m := range mods {
		add(r.I4[m])
	}
	add(r.Closure)
	for _, c := range r.Contexts {
		add(c)
	}
	add(r.Consumption)
	add(r.Findings)
	add(r.Comparison)
	return out
}

// Collect runs one configuration. It never starts a process itself: every
// call goes through o.Invoker.
func Collect(o Options) *Result {
	res := &Result{Config: o.Config, Checks: judge.NewChecks(CheckIDs()), Snapshots: map[string]*modsnap.Snapshot{},
		Refs: Refs{Calls: map[spec.CallKind]Ref{}, I4: map[string]Ref{}}}
	ck := res.Checks
	cfg := o.Config.String()
	if !o.Budget.AdmitUnit("configuration " + cfg) {
		ck.SetIfUnset("not admitted: budget insufficient for configuration " + cfg)
		return res
	}

	res.stageA(o)
	aHold := ck.AllHold(AIDs...)
	if !aHold {
		ck.SetIfUnset("stage A has a failed or unevaluated check")
		return res
	}
	res.stageB(o)
	if !ck.AllHold(BIDs()...) {
		ck.SetIfUnset("stage B has a failed or unevaluated check")
		return res
	}
	res.scanAndCompare(o)
	ck.SetIfUnset("not reached")
	return res
}

func (res *Result) stageA(o Options) {
	ck := res.Checks
	cfg := o.Config.String()

	// A4: R/go.sum by I-1; R/go.mod's identity is recorded with it.
	sum := ReadWhole(o.FS, filepath.Join(o.R, "go.sum"))
	ck.Enter("A4", sha(sum.Data))
	mod := ReadWhole(o.FS, filepath.Join(o.R, "go.mod"))
	res.Refs.RepoFiles = o.Rec.Raw(cfg+"/repo-files", map[string]any{
		"go.sum": fileIdentity(sum),
		"go.mod": fileIdentity(mod),
	})
	var a4 judge.Acc
	// Every real error of both reads is kept: an open, read or close error
	// is one J-1 each, and a read error does not hide the close error.
	for _, f := range []struct {
		name string
		r    ReadResult
	}{{"go.sum", sum}, {"go.mod", mod}} {
		for _, e := range f.r.Errors() {
			a4.Add(judge.Real(judge.J1, f.name, "A4 "+e.Op+" R/"+f.name, e.Op+" error", e.Err.Error()))
		}
	}
	if sum.OK() {
		gs, problems := decode.ParseGoSum(sum.Data)
		res.GoSum = gs
		for _, p := range problems {
			a4.Add(judge.Real(judge.J2, fmt.Sprintf("go.sum:%d", p.Line), "A4 I-1", string(p.Kind), p.Detail))
		}
	}
	ck.Set("A4", a4.Result())

	// A5: R/vendor and R/go.work must not exist.
	ck.Enter("A5", "")
	var a5 judge.Acc
	for _, name := range []string{"vendor", "go.work"} {
		_, err := o.FS.Lstat(filepath.Join(o.R, name))
		switch {
		case err == nil:
			a5.Add(judge.Real(judge.J2, name, "A5 Lstat R/"+name, "exists", ""))
		case !errors.Is(err, fs.ErrNotExist):
			a5.Add(judge.Real(judge.J1, name, "A5 Lstat R/"+name, "lstat error", err.Error()))
		}
	}
	ck.Set("A5", a5.Result())

	// A1-a: the guard before any call.
	res.Guard = o.Guard()
	res.Refs.Toolchain = o.Rec.Raw(cfg+"/toolchain", res.Guard)
	ck.Enter("A1-a", res.Refs.Toolchain.SHA256)
	if fs := EvaluatePre(res.Guard); len(fs) > 0 {
		ck.Fail("A1-a", fs...)
		ck.Unevaluated("A1-b", "A1-a failed: no call runs")
		ck.Unevaluated("A1-c", "A1-a failed: no call runs")
		ck.Unevaluated("A2", "A1-a failed: no call runs")
		ck.Unevaluated("A3", "A1-a failed: no call runs")
		return
	}
	ck.Hold("A1-a")

	// A1-b: the first call and its decoding.
	out, ok := res.call(o, spec.CallEnv, "A1-b")
	if ok {
		env, err := decode.DecodeEnv(out, spec.EnvNames)
		if err != nil {
			ck.Fail("A1-b", judge.Real(judge.J3, "call 1", "A1-b decode", "decode", err.Error()))
		} else {
			res.Env = env
			ck.Hold("A1-b")
		}
	}
	if ck.Status("A1-b") != judge.Hold {
		for _, id := range []string{"A1-c", "A2", "A3"} {
			ck.Unevaluated(id, "A1-b did not hold")
		}
		return
	}

	// A1-c: the guard after the first call.
	ck.Enter("A1-c", "")
	if fs := EvaluatePost(res.Guard, res.Env); len(fs) > 0 {
		ck.Fail("A1-c", fs...)
		ck.Unevaluated("A2", "A1-c failed: no further call runs")
		ck.Unevaluated("A3", "A1-c failed: no further call runs")
		return
	}
	ck.Hold("A1-c")

	// A2 and A3 do not depend on each other.
	if out, ok := res.call(o, spec.CallContext, "A2"); ok {
		c, err := decode.DecodeContext(out)
		if err != nil {
			ck.Fail("A2", judge.Real(judge.J3, "call 2", "A2 decode", "decode", err.Error()))
		} else {
			res.Context = c
			ck.Hold("A2")
		}
	}
	if out, ok := res.call(o, spec.CallList, "A3"); ok {
		pkgs, err := decode.DecodePackages(out)
		if err != nil {
			ck.Fail("A3", judge.Real(judge.J3, "call 3", "A3 decode", "decode", err.Error()))
		} else {
			res.Packages = pkgs
			ck.Hold("A3")
		}
	}
}

// call runs one call under admission. It returns the standard output only
// when the call succeeded by §12.14 item 11; otherwise the check is set:
// unevaluated when admission refused the call, J-1 when it failed. The
// output of a failed call is recorded raw and never consumed.
func (res *Result) call(o Options, k spec.CallKind, id string) ([]byte, bool) {
	ck := res.Checks
	cfg := o.Config.String()
	unit := fmt.Sprintf("%s %s", k, cfg)
	if !o.Budget.AdmitCall(unit) {
		ck.Unevaluated(id, "not admitted: budget insufficient for "+unit)
		return nil, false
	}
	exe := ""
	if res.Guard.Candidate != "" {
		exe = res.Guard.Candidate
	}
	cs := CallSpec{Config: o.Config, Kind: k, Exe: exe, Args: spec.Args(k), Dir: o.R, Env: spec.ToolEnv(o.Config)}
	r := o.Invoker.Invoke(cs)
	res.Calls = append(res.Calls, r)
	res.Refs.Calls[k] = o.Rec.Raw(fmt.Sprintf("%s/%s", cfg, k), rawCall(r))
	ck.Enter(id, sha(r.Stdout))
	if !r.Succeeded(o.Budget.W) {
		var fs []judge.Failure
		for _, p := range r.Problems(o.Budget.W) {
			fs = append(fs, judge.Real(judge.J1, k.String(), id+" "+unit, "call failed", p))
		}
		if len(fs) == 0 {
			fs = append(fs, judge.Real(judge.J1, k.String(), id+" "+unit, "call failed", "not successful"))
		}
		ck.Fail(id, fs...)
		return nil, false
	}
	return r.Stdout, true
}

func fileIdentity(r ReadResult) map[string]any {
	out := map[string]any{"bytes": len(r.Data), "sha256": sha(r.Data), "closed": r.Closed}
	for _, e := range r.Errors() {
		out[e.Op+"_error"] = e.Err.Error()
	}
	return out
}

// rawCall is a call's raw record (plan §6.1): the argument array as passed,
// the variables the tool set, the working directory, both outputs with
// their SHA-256, the exit status, the times and the control results.
func rawCall(r CallResult) map[string]any {
	env := map[string]string{}
	for _, e := range r.Spec.Env {
		env[e.Name] = e.Value
	}
	return map[string]any{
		"kind": r.Spec.Kind.String(), "exe": r.Spec.Exe, "args": r.Spec.Args, "dir": r.Spec.Dir, "env": env,
		"started": r.Started, "start_error": r.StartErr, "reaped": r.Reaped, "exit_code": r.ExitCode,
		"stdout": r.Stdout, "stdout_sha256": sha(r.Stdout), "stdout_eof": r.StdoutEOF,
		"stderr": r.Stderr, "stderr_sha256": sha(r.Stderr), "stderr_eof": r.StderrEOF,
		"control_failures": r.Failures, "injected": r.Injected,
		"start": r.Start.Format(time.RFC3339Nano), "end": r.End.Format(time.RFC3339Nano),
	}
}
