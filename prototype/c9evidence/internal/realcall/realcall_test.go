//go:build c9evidence_realcall

package realcall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/controls"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/job"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/pseudo"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/toolctl"
)

// TestMain acts as LC-9's pseudo-tool when re-executed (RC-4 runs LC-9),
// and otherwise refuses to run outside the dedicated test workflow on a
// GitHub-hosted runner (LE-5 ②: never on a development machine or a
// long-running self-hosted runner).
func TestMain(m *testing.M) {
	pseudo.MaybeRun()
	if os.Getenv("C9EVIDENCE_REALCALL") != "1" || os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" {
		fmt.Fprintln(os.Stderr, "realcall: refused: the real calls run only in the dedicated test workflow on a GitHub-hosted runner (LE-5 ②; review-resolution §200)")
		os.Exit(2)
	}
	os.Exit(m.Run())
}

// The fixture module: x/mod v0.42.0 only, with the root go.sum's lines 42
// and 43. The module is in the cache because the workflow downloaded the
// tool module's dependencies before the tests; the calls run with
// GOPROXY=off and download nothing.
const (
	fxModule = "example.com/c9fx"
	xmodSum  = "h1:ICeL3t2L30ljSPHHxDnR4fqVcBt/zR2zJAb9juavmQg="
	xmodMod  = "h1:u5fzIrRInkL8MaO1cbRkUHT64v303YIT5KEqpNr7jRs="
)

var fxFiles = map[string]string{
	"go.mod": "module example.com/c9fx\n\ngo 1.27\n\nrequire golang.org/x/mod v0.42.0\n",
	"go.sum": "golang.org/x/mod v0.42.0 " + xmodSum + "\ngolang.org/x/mod v0.42.0/go.mod " + xmodMod + "\n",
	"c9fx.go": "// Package c9fx is the fixture module of the real-call tests.\npackage c9fx\n\n" +
		"import \"golang.org/x/mod/semver\"\n\n// Valid reports whether v is a semantic version.\nfunc Valid(v string) bool { return semver.IsValid(v) }\n",
}

// fixture writes the fixture module below RUNNER_TEMP (no short names on
// Windows) and returns its directory with symbolic links resolved, as a
// clean absolute path: R for the calls.
func fixture(t *testing.T) string {
	t.Helper()
	base := os.Getenv("RUNNER_TEMP")
	if base == "" {
		t.Fatal("RUNNER_TEMP is not set")
	}
	d, err := os.MkdirTemp(base, "c9fx-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(d); err != nil {
			t.Errorf("remove the fixture: %v", err)
		}
	})
	if d, err = filepath.EvalSymlinks(d); err != nil {
		t.Fatal(err)
	}
	for name, body := range fxFiles {
		if err := os.WriteFile(filepath.Join(d, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Clean(d)
}

func guardValues() collect.GuardValues { return collect.ReadGuard(collect.RealGuardSources()) }

// evidence writes one test's structured outcome below C9EVIDENCE_OUT for
// the workflow's artifact (and the repository's evidence directory).
func evidence(t *testing.T, name string, v any) {
	t.Helper()
	dir := os.Getenv("C9EVIDENCE_OUT")
	if dir == "" {
		t.Error("C9EVIDENCE_OUT is not set: the outcome is not kept")
		return
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Errorf("marshal %s: %v", name, err)
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Error(err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, name), append(b, '\n'), 0o644); err != nil {
		t.Error(err)
	}
}

func hostConfig() spec.Config { return spec.Config{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH} }

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// TestRC0GuardPath is a diagnostic of the A1-a guard's input on this
// runner: what exec.LookPath("go") finds, its absolute form, what
// filepath.EvalSymlinks makes of it, and each component's Lstat mode and
// Readlink result. It starts no process and judges nothing.
func TestRC0GuardPath(t *testing.T) {
	found, err := exec.LookPath("go")
	out := map[string]any{"lookpath": found, "lookpath_error": errText(err)}
	abs, aerr := filepath.Abs(found)
	out["abs"], out["abs_error"] = abs, errText(aerr)
	ev, everr := filepath.EvalSymlinks(abs)
	out["evalsymlinks"], out["evalsymlinks_error"] = ev, errText(everr)
	var comps []map[string]any
	for p := abs; p != ""; p = filepath.Dir(p) {
		c := map[string]any{"path": p}
		if fi, err := os.Lstat(p); err != nil {
			c["lstat_error"] = err.Error()
		} else {
			c["mode"] = fi.Mode().String()
		}
		if l, err := os.Readlink(p); err == nil {
			c["readlink"] = l
		} else {
			c["readlink_error"] = err.Error()
		}
		if e, err := filepath.EvalSymlinks(p); err == nil {
			c["evalsymlinks"] = e
		} else {
			c["evalsymlinks_error"] = err.Error()
		}
		comps = append(comps, c)
		if filepath.Dir(p) == p {
			break
		}
	}
	out["components"] = comps
	out["GODEBUG"] = os.Getenv("GODEBUG")
	evidence(t, "rc0-guard-path.json", out)
	b, _ := json.MarshalIndent(out, "", "  ")
	t.Logf("%s", b)
}

// TestRC1Collect is RC-1: the full stages A and B, the scan and K on the
// fixture for both configurations of this GOOS, with the real caller, the
// real guard and the real type checker.
func TestRC1Collect(t *testing.T) {
	fx := fixture(t)
	for _, c := range spec.ConfigsFor(runtime.GOOS) {
		t.Run(strings.ReplaceAll(c.String(), "/", "-"), func(t *testing.T) {
			start := time.Now()
			w, d := chain.Deadlines(start)
			reg := toolctl.New(toolctl.Options{Clock: chain.RealClock{}, W: w, D: d})
			rec := &collect.MemRecorder{}
			res := collect.Collect(collect.Options{
				Config: c, R: fx, Invoker: reg, Budget: collect.NewBudget(chain.RealClock{}, start, chain.Budget), Guard: guardValues,
				FS: modsnap.OS{}, MainModule: fxModule, Rec: rec, TypeCheck: collect.TypeCheck{},
			})
			elapsed := time.Since(start)
			var lines []string
			for _, it := range rec.Items() {
				lines = append(lines, strings.TrimSuffix(string(it.Line), "\n"))
			}
			snaps := reg.Snapshots()
			fails, pend := reg.Problems()
			evidence(t, "rc1-"+strings.ReplaceAll(c.String(), "/", "-")+".json", map[string]any{
				"config": c.String(), "R": fx, "elapsed": elapsed.String(), "checks": res.Checks.Snapshot(), "entries": res.Checks.Entries(),
				"defects": res.Checks.Defects(), "modules": res.Modules, "consistent": res.Consistent, "differences": res.Diff,
				"calls": res.Calls, "registry": snaps, "registry_failures": fails, "registry_pending": pend, "records": lines,
				"recorder_errors": rec.Errors(), "guard": res.Guard,
			})
			t.Logf("%s: %s", c, elapsed)

			for _, it := range res.Checks.Snapshot() {
				if it.Result.Status != judge.Hold {
					t.Errorf("check %s: %s %s %v", it.ID, it.Result.Status, it.Result.Reason, it.Result.Failures)
				}
			}
			if d := res.Checks.Defects(); len(d) > 0 {
				t.Errorf("defects %v", d)
			}
			if !res.Consistent || len(res.Diff) > 0 {
				t.Errorf("K: consistent %v differences %v", res.Consistent, res.Diff)
			}
			if len(snaps) != 3 || len(fails) > 0 || len(pend) > 0 {
				t.Errorf("registry: %d records, failures %v, pending %v", len(snaps), fails, pend)
			}
			for _, s := range snaps {
				if !s.Success || !s.Terminal || len(s.Faults) > 0 {
					t.Errorf("call %d %s: success %v terminal %v %v faults %q", s.ID, s.Kind, s.Success, s.Terminal, s.Failures, faultSet(s))
				}
				healthyControl(t, s, toolctl.StdoutW, toolctl.StdoutR, toolctl.StderrW, toolctl.StderrR)
			}
			// T2 as one argument (the actual evidence of this platform):
			// the second call's argument array holds T2 whole at index 3.
			for _, r := range res.Calls {
				if r.Spec.Kind != spec.CallContext {
					continue
				}
				if len(r.Spec.Args) != 5 || r.Spec.Args[3] != spec.T2 {
					t.Errorf("call 2 arguments %q", r.Spec.Args)
				}
				if n, _, ok := spec.T2Binding(r.Spec.Args[3]); !ok || n != spec.T2Len {
					t.Errorf("call 2's T2 does not bind")
				}
			}
			if res.Context.GOOS != c.GOOS || res.Context.GOARCH != c.GOARCH || res.Context.GOROOT != res.Guard.Root || res.Context.CgoEnabled {
				t.Errorf("context %+v (guard root %q)", res.Context, res.Guard.Root)
			}
			var main, xmod, std bool
			for _, p := range res.Packages {
				switch {
				case p.ImportPath == fxModule:
					main = p.Module != nil && p.Module.Path == fxModule && p.Module.Main && p.Module.Dir == fx
				case p.ImportPath == "golang.org/x/mod/semver":
					xmod = p.Module != nil && p.Module.Version == "v0.42.0" && p.Module.Sum == xmodSum && p.Module.GoModSum == xmodMod
				case p.Standard:
					std = true
				}
			}
			if !main || !xmod || !std {
				t.Errorf("closure: main %v x/mod %v std %v", main, xmod, std)
			}
			if s := res.Snapshots["golang.org/x/mod"]; s == nil || s.H1 != xmodSum {
				t.Errorf("I-4 of x/mod: %+v", s)
			}
		})
	}
}

// listSpec is the third call for the host configuration on the fixture.
func listSpec(t *testing.T, fx string) collect.CallSpec {
	t.Helper()
	g := guardValues()
	if fs := collect.EvaluatePre(g); len(fs) > 0 {
		t.Fatalf("A1-a: %v", fs)
	}
	c := hostConfig()
	return collect.CallSpec{Config: c, Kind: spec.CallList, Exe: g.Candidate, Args: spec.Args(spec.CallList), Dir: fx, Env: spec.ToolEnv(c)}
}

func after(d time.Duration) <-chan struct{} {
	ch := make(chan struct{})
	time.AfterFunc(d, func() { close(ch) })
	return ch
}

// TestRC2Overflow is RC-2: the third call with a 1 KiB limit overflows its
// standard output; the read end is closed at once and the reap is recorded
// as it came.
func TestRC2Overflow(t *testing.T) {
	fx := fixture(t)
	now := time.Now()
	reg := toolctl.New(toolctl.Options{Clock: chain.RealClock{}, W: now.Add(110 * time.Second), D: now.Add(120 * time.Second), Limit: 1024})
	r := reg.Invoke(listSpec(t, fx))
	if !reg.WaitTerminal(1, after(120*time.Second)) {
		t.Fatal("not terminal")
	}
	s, _ := reg.Snapshot(1)
	faults := faultSet(s)
	evidence(t, "rc2-overflow.json", map[string]any{"result": r, "snapshot": s, "lines": toolctl.Lines(s), "faults": faults})
	t.Logf("%v %q", toolctl.Lines(s), faults)
	if r.Succeeded(reg.W()) || !s.Out[0].Overflow || s.State != toolctl.StWaited || s.Class != "B" || s.Success || s.Expired {
		t.Errorf("snapshot %v", toolctl.Lines(s))
	}
	// Review 154 B-154-4: the whole fault set is consumed. Besides the
	// overflow (the protocol's own finding), the only fault allowed is the
	// tool's own reap result, and only as follows (§12.14 item 4: a write
	// after the read end was closed may get EPIPE or SIGPIPE; that is the
	// tool's own result): on Unix, either go was ended by SIGPIPE (its Wait
	// returns an *exec.ExitError) or it had written everything before the
	// close and exited 0; on Windows the write error does not end go, which
	// exits 0. Any other fault is a failure. Review 155 B-154-4-R-S: the
	// end is judged by its structured identity (the signal's name), not by
	// its code or text.
	ended := toolctl.ExitKind(s.Exit) == "signaled SIGPIPE" && slices.Equal(faults, []string{"wait operation exit-status signaled SIGPIPE"})
	clean := toolctl.ExitKind(s.Exit) == "exited 0" && len(faults) == 0
	if runtime.GOOS == "windows" && !clean || runtime.GOOS != "windows" && !ended && !clean {
		t.Errorf("reap %+v faults %q: not one of the expected outcomes", s.Exit, faults)
	}
	healthyControl(t, s, toolctl.StdoutW, toolctl.StdoutR, toolctl.StderrW, toolctl.StderrR)
	if o := s.Out[1]; !o.Done || !o.EOF || o.ReadErr != "" || o.Overflow || s.Out[0].ReadErr != "" {
		t.Errorf("reads %+v %+v", s.Out[0], o)
	}
	if s.Req.State != toolctl.ReqNone || len(s.Unknown) > 0 || len(s.NotAdmit) > 0 || len(s.Injected) > 0 {
		t.Errorf("request %+v unknown %q not-admitted %q injected %q", s.Req, s.Unknown, s.NotAdmit, s.Injected)
	}
}

// healthyControl checks what every real call here must have, whatever its
// outcome: the four admissions admitted and nothing skipped; the Cmd and
// the process held; each named descriptor closed once with a nil result
// (its release confirmed) and its object held; no panic; Linux's pidfd
// closed once with a nil result.
func healthyControl(t *testing.T, s toolctl.Snapshot, ds ...int) {
	t.Helper()
	if len(s.Admissions) != 4 || len(s.Skipped) > 0 || !s.CmdHeld || !s.ProcessHeld || s.Panic != "" {
		t.Errorf("admissions %+v skipped %q cmd %v process %v panic %q", s.Admissions, s.Skipped, s.CmdHeld, s.ProcessHeld, s.Panic)
	}
	for _, a := range s.Admissions {
		if !a.Admitted {
			t.Errorf("admission %s refused", a.Point)
		}
	}
	for _, d := range ds {
		if x := s.Desc[d]; x.State != toolctl.DescClosed || x.CloseErr != "" || x.Release != toolctl.ReleaseConfirmed || !x.Held || len(x.Injected) > 0 {
			t.Errorf("%s: %+v", toolctl.DescNames[d], x)
		}
	}
	if s.Mode == toolctl.ModePidfd && (s.Pidfd.State != toolctl.PidfdClosed || s.Pidfd.CloseErr != "" || s.Pidfd.Closes != 1) {
		t.Errorf("pidfd %+v", s.Pidfd)
	}
}

// TestRC3Expiry is RC-3: the start task of the third call is held after
// Start and before its one Wait while the fake clock reaches W; the
// platform's termination is decided and recorded; then Wait proceeds. The
// hold stops the parent, not go: go may finish by itself before the
// request (Review 155 B-154-4-R-C). The facts are recorded apart and judged
// by judgeRC3 (rc3_test.go) against the legal branches only.
func TestRC3Expiry(t *testing.T) {
	fx := fixture(t)
	clk := chain.NewFakeClock(time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC))
	entered, release := make(chan struct{}), make(chan struct{})
	var once, ronce sync.Once
	t.Cleanup(func() { ronce.Do(func() { close(release) }) })
	reg := toolctl.New(toolctl.Options{Clock: clk, W: clk.Now().Add(100 * time.Second), D: clk.Now().Add(110 * time.Second), Seams: &toolctl.Seams{
		Hook: func(point string, _ int) {
			if point == toolctl.HookBeforeWait {
				once.Do(func() { close(entered) })
				<-release
			}
		},
	}})
	// The spec (and its guard) is taken here, on the test goroutine: a
	// failed guard must stop the test, not a background goroutine.
	cs := listSpec(t, fx)
	got := make(chan collect.CallResult, 1)
	go func() { got <- reg.Invoke(cs) }()
	select {
	case <-entered:
	case <-time.After(60 * time.Second):
		t.Fatal("the start task did not reach Wait")
	}
	clk.Set(reg.W())
	var r collect.CallResult
	select {
	case r = <-got:
	case <-time.After(60 * time.Second):
		t.Fatal("Invoke did not return at W")
	}
	if m := reg.Mode(); m == toolctl.ModePidfd || m == toolctl.ModeKill {
		if !reg.WaitFor(1, after(60*time.Second), func(s toolctl.Snapshot) bool { return s.Req.State == toolctl.ReqDone }) {
			t.Error("the request did not finish")
		}
	}
	ronce.Do(func() { close(release) })
	terminal := reg.WaitTerminal(1, after(120*time.Second))
	s, _ := reg.Snapshot(1)
	succeeded := r.Succeeded(reg.W())
	facts := rc3Facts(s, succeeded)
	evidence(t, "rc3-expiry.json", map[string]any{"result": r, "snapshot": s, "lines": toolctl.Lines(s), "terminal": terminal, "facts": facts})
	t.Logf("%v %q", toolctl.Lines(s), faultSet(s))
	t.Logf("RC-3 branch %q problems %q", facts["branch"], facts["problems"])
	if !terminal {
		t.Fatal("not terminal")
	}
	// What every real call here must have, whatever the branch.
	healthyControl(t, s, toolctl.StdoutW, toolctl.StdoutR, toolctl.StderrW, toolctl.StderrR)
	// Review 155 B-154-4-R-C: one of the legal branches, by the facts
	// recorded apart; never a liveness drawn from the held Wait.
	if branch, why := judgeRC3(s, succeeded); branch == "" {
		t.Errorf("RC-3 in no legal branch: %q", why)
	}
}

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// TestRC4JobDryRun is RC-4: a whole job on the fixture R, with the real
// caller handed to the closeout, every control of LC-2 to LC-9, and the
// records in a temporary directory. It is a dry run on a fixture, not the
// formal run: R is never the repository.
func TestRC4JobDryRun(t *testing.T) {
	fx := fixture(t)
	recs := filepath.Join(os.Getenv("RUNNER_TEMP"), "c9-rc4-records")
	if out := os.Getenv("C9EVIDENCE_OUT"); out != "" {
		recs = filepath.Join(out, "rc4-records")
	}
	start := time.Now()
	w, d := chain.Deadlines(start)
	reg := toolctl.New(toolctl.Options{Clock: chain.RealClock{}, W: w, D: d})
	errOut := &syncBuf{}
	ids := map[string]string{}
	for _, k := range []string{"GITHUB_SHA", "GITHUB_REF", "GITHUB_WORKFLOW_REF", "GITHUB_WORKFLOW_SHA", "RUNNER_OS", "RUNNER_ARCH"} {
		ids[k] = os.Getenv(k)
	}
	cfg := job.Config{
		GOOS: runtime.GOOS, R: fx, RecordsDir: recs, Clock: chain.RealClock{}, Start: start,
		WriteFS: chain.OSFS{}, ReadFS: modsnap.OS{}, Stderr: errOut, Invoker: reg, Tools: reg, Guard: guardValues,
		TypeCheck: collect.TypeCheck{}, Controls: controls.All(runtime.GOOS), MainModule: fxModule,
		Candidates: func(spec.Config) []spec.Candidate { return nil }, Identity: ids, T2: spec.T2,
	}
	j, err := job.Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	j.RunDomains()
	f := j.Finish(0)
	elapsed := time.Since(start)
	fails, pend := reg.Problems()
	evidence(t, "rc4-job.json", map[string]any{"final": f, "elapsed": elapsed.String(), "stderr": errOut.String(),
		"registry": reg.Snapshots(), "registry_failures": fails, "registry_pending": pend})
	t.Logf("final %v exit %d in %s", f.Conclusion, f.ExitCode, elapsed)
	t.Logf("diagnostics:\n%s", errOut.String())
	if f.Conclusion != chain.Consistent || f.ExitCode != 0 {
		t.Errorf("final %v exit %d %v", f.Conclusion, f.ExitCode, f.Reasons)
	}
	if n := reg.Len(); n != 6 || len(fails) > 0 || len(pend) > 0 {
		t.Errorf("registry: %d records, failures %v, pending %v", n, fails, pend)
	}
	names, err := os.ReadDir(recs)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range names {
		got = append(got, e.Name())
	}
	want := []string{"closeout.jsonl", "controls.jsonl", "derived.jsonl", "manifest.txt", "raw.jsonl"}
	if !slices.Equal(got, want) {
		t.Errorf("record files %v, want %v", got, want)
	}
}
