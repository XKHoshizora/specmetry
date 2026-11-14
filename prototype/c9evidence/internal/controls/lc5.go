package controls

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// The LC-5 baseline LC5-0 (plan §5.3): a fixture repository root R_f with
// one main package that imports one package of a dependency module Q; Q's
// module directory holds only regular files; go.sum has Q's two lines with
// the h1 computed independently of the tool.
const (
	rfModule = "example.com/rf"
	qModule  = "example.com/q"
	qH1      = "h1:dnwI/ZuKfmHPWyArCFBdzDu0EMVFu3RATvDJC6uc3pY="
	qGoModH1 = "h1:RM5+6DP3DKvetkVn8KYL41gwblKnUwbpZ2Oz6Fe/8/I="
)

var qFiles = map[string]string{
	"go.mod": "module example.com/q\n\ngo 1.22\n",
	"q.go":   "package q\n\n// V is the fixture's only declaration.\nconst V = 1\n",
}

func rfFiles() map[string]string {
	return map[string]string{
		"go.mod":  "module example.com/rf\n\ngo 1.22\n\nrequire example.com/q v1.0.0\n",
		"go.sum":  "example.com/q v1.0.0 " + qH1 + "\nexample.com/q v1.0.0/go.mod " + qGoModH1 + "\n",
		"main.go": "package main\n\nimport \"example.com/q\"\n\nfunc main() { _ = q.V }\n",
	}
}

// fakeClock is a settable clock for the controls.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// lc5Case is one row of the LC-5 table: the deviation from LC5-0.
type lc5Case struct {
	name       string
	goVersion  string // analyser and VERSION, for LC5-24
	env        func(map[string]string)
	ctx        func(*ctxText)
	listExit   int
	listQExtra string // extra JSON for Q's Module (LC5-42's Replace)
}

// ctxText renders the second call's output, context text v1 (method record
// §3.3.2), the way T2 writes it.
type ctxText struct {
	GOARCH, GOOS, GOROOT, GOPATH, Compiler, InstallSuffix string
	CgoEnabled, UseAllFiles                               bool
	BuildTags, ToolTags, ReleaseTags                      []string
}

func (c ctxText) bytes() []byte {
	var b strings.Builder
	q := func(name, v string) { fmt.Fprintf(&b, "%s %q\n", name, v) }
	b.WriteString("c9-context v1\n")
	q("GOARCH", c.GOARCH)
	q("GOOS", c.GOOS)
	q("GOROOT", c.GOROOT)
	q("GOPATH", c.GOPATH)
	fmt.Fprintf(&b, "CgoEnabled %t\nUseAllFiles %t\n", c.CgoEnabled, c.UseAllFiles)
	q("Compiler", c.Compiler)
	q("InstallSuffix", c.InstallSuffix)
	list := func(name string, xs []string) {
		fmt.Fprintf(&b, "%s %d\n", name, len(xs))
		for _, x := range xs {
			fmt.Fprintf(&b, "- %q\n", x)
		}
	}
	list("BuildTags", c.BuildTags)
	list("ToolTags", c.ToolTags)
	list("ReleaseTags", c.ReleaseTags)
	b.WriteString("end\n")
	return []byte(b.String())
}

// rfSetup is a prepared R_f and Q.
type rfSetup struct {
	root, rf, q, goroot string
	cfg                 spec.Config
}

func prepareRF(e *Env, name string) (*fixture, *rfSetup, error) {
	files := map[string]string{}
	for k, v := range rfFiles() {
		files["rf/"+k] = v
	}
	for k, v := range qFiles {
		files["modcache/example.com/q@v1.0.0/"+k] = v
	}
	fx, err := prepare(e, name, files)
	if err != nil {
		return nil, nil, err
	}
	cfg := spec.ConfigsFor(e.GOOS)[0]
	return fx, &rfSetup{
		root: fx.dir, rf: filepath.Join(fx.dir, "rf"), q: filepath.Join(fx.dir, "modcache", "example.com", "q@v1.0.0"),
		goroot: filepath.Join(fx.dir, "goroot"), cfg: cfg,
	}, nil
}

// guardValues are the A1-a values the seam hands over (plan §5.3 LC5-0).
func (s *rfSetup) guardValues(version string) collect.GuardValues {
	exe := "go"
	if s.cfg.GOOS == "windows" {
		exe = "go.exe"
	}
	return collect.GuardValues{AnalyzerVersion: version, Candidate: filepath.Join(s.goroot, "bin", exe), Root: s.goroot, ExeName: exe,
		ExeRegular: true, SrcCmdGo: true, VersionLine: version, ExeSHA256: strings.Repeat("0", 64)}
}

func (s *rfSetup) responses(c lc5Case) map[spec.CallKind]collect.FakeResponse {
	env := map[string]string{"GOROOT": s.goroot, "GOVERSION": "go1.27.3", "GOOS": s.cfg.GOOS, "GOARCH": s.cfg.GOARCH,
		"CGO_ENABLED": "0", "GOFLAGS": "", "GOWORK": "off", "GOTOOLCHAIN": "local"}
	if c.env != nil {
		c.env(env)
	}
	envOut, _ := json.MarshalIndent(env, "", "\t")
	ct := ctxText{GOARCH: s.cfg.GOARCH, GOOS: s.cfg.GOOS, GOROOT: s.goroot, Compiler: "gc",
		ToolTags: []string{s.cfg.GOARCH + ".v1"}, ReleaseTags: []string{"go1.1", "go1.27"}}
	if c.ctx != nil {
		c.ctx(&ct)
	}
	qMod := fmt.Sprintf(`{"Path":%q,"Version":"v1.0.0","Dir":%q,"GoVersion":"1.22","Sum":%q,"GoModSum":%q%s}`,
		qModule, s.q, qH1, qGoModH1, c.listQExtra)
	list := fmt.Sprintf("{\n\t\"ImportPath\": %q,\n\t\"Dir\": %q,\n\t\"Name\": \"q\",\n\t\"Module\": %s,\n\t\"GoFiles\": [\"q.go\"]\n}\n", qModule, s.q, qMod) +
		fmt.Sprintf("{\n\t\"ImportPath\": %q,\n\t\"Dir\": %q,\n\t\"Name\": \"main\",\n\t\"Module\": {\"Path\":%q,\"Main\":true,\"Dir\":%q,\"GoVersion\":\"1.22\"},\n\t\"GoFiles\": [\"main.go\"],\n\t\"Imports\": [%q]\n}\n",
			rfModule, s.rf, rfModule, s.rf, qModule)
	return map[spec.CallKind]collect.FakeResponse{
		spec.CallEnv:     {Stdout: append(envOut, '\n')},
		spec.CallContext: {Stdout: ct.bytes()},
		spec.CallList:    {Stdout: []byte(list), ExitCode: c.listExit},
	}
}

// projection builds the complete expectation of one case from its rules:
// the status of every check of E, the failures, the call counts and
// whether any derived record came from the third call.
func projection(over map[string][]string, def map[string]string, calls [3]int, fromCall3 bool) []string {
	var out []string
	for _, id := range collect.CheckIDs() {
		if lines, ok := over[id]; ok {
			out = append(out, lines...)
			continue
		}
		st := "hold"
		for p, s := range def {
			if id == p || strings.HasPrefix(id, p) {
				st = s
			}
		}
		out = append(out, id+" "+st)
	}
	out = append(out, fmt.Sprintf("calls env=%d context=%d list=%d", calls[0], calls[1], calls[2]))
	if fromCall3 {
		out = append(out, "derived from call 3: present")
	} else {
		out = append(out, "derived from call 3: none")
	}
	return out
}

var bscK = map[string]string{"B5": "unevaluated", "S": "unevaluated", "K": "unevaluated"}
var allBSK = map[string]string{"B": "unevaluated", "S": "unevaluated", "K": "unevaluated"}

// LC5Cases are the eight rows of plan §5.3, each with its projection.
func lc5Cases() []struct {
	c    lc5Case
	want []string
} {
	j := func(id, class, item, kind string) string {
		return fmt.Sprintf("%s fail %s %s %s real", id, class, item, kind)
	}
	return []struct {
		c    lc5Case
		want []string
	}{
		{lc5Case{name: "LC5-0 (V-1)"}, projection(nil, nil, [3]int{1, 1, 1}, true)},
		{lc5Case{name: "LC5-19b (V-19b)", env: func(m map[string]string) { m["GOWORK"] = "/rf/go.work" }},
			projection(map[string][]string{"B1/GOWORK": {j("B1/GOWORK", "J-2", "GOWORK", "mismatch")}}, bscK, [3]int{1, 1, 1}, true)},
		{lc5Case{name: "LC5-20a (V-20)", env: func(m map[string]string) { m["GOFLAGS"] = "-mod=mod" }},
			projection(map[string][]string{"B1/GOFLAGS": {j("B1/GOFLAGS", "J-2", "GOFLAGS", "mismatch")}}, bscK, [3]int{1, 1, 1}, true)},
		{lc5Case{name: "LC5-20b (V-20)", env: func(m map[string]string) { m["GOFLAGS"] = "-mod=mod" }, listExit: 1},
			projection(map[string][]string{"A3": {j("A3", "J-1", "call3-list", "call failed")}}, allBSK, [3]int{1, 1, 1}, false)},
		{lc5Case{name: "LC5-24 (V-24)", goVersion: "go1.27.4"},
			projection(map[string][]string{"A1-a": {j("A1-a", "J-1", "analyzer", "version"), j("A1-a", "J-1", "VERSION", "version")},
				"A1-b": {"A1-b unevaluated"}, "A1-c": {"A1-c unevaluated"}, "A2": {"A2 unevaluated"}, "A3": {"A3 unevaluated"}}, allBSK, [3]int{0, 0, 0}, false)},
		{lc5Case{name: "LC5-40a (V-40)", ctx: func(c *ctxText) { c.ToolTags = nil }},
			projection(map[string][]string{"B2/ToolTags": {j("B2/ToolTags", "J-3", "ToolTags", "mismatch")}}, bscK, [3]int{1, 1, 1}, true)},
		{lc5Case{name: "LC5-40b (V-40)", ctx: func(c *ctxText) { c.ToolTags = nil; c.BuildTags = []string{"x"} }},
			projection(map[string][]string{"B2/ToolTags": {j("B2/ToolTags", "J-3", "ToolTags", "mismatch")},
				"B2/BuildTags": {j("B2/BuildTags", "J-2", "BuildTags", "mismatch")}}, bscK, [3]int{1, 1, 1}, true)},
		{lc5Case{name: "LC5-42 (V-42)", listExit: 1, listQExtra: `,"Replace":{"Path":"example.com/q","Dir":"/elsewhere"}`},
			projection(map[string][]string{"A3": {j("A3", "J-1", "call3-list", "call failed")}}, allBSK, [3]int{1, 1, 1}, false)},
	}
}

// runCollect runs the tool's collection on R_f with the fake caller.
func runCollect(e *Env, s *rfSetup, c lc5Case, clock *fakeClock, b *collect.Budget, delay3 time.Duration) (*collect.Result, *collect.Fake, *collect.MemRecorder) {
	resp := s.responses(c)
	if delay3 > 0 {
		r := resp[spec.CallList]
		r.Delay = delay3
		resp[spec.CallList] = r
	}
	fake := &collect.Fake{Clock: clock, Advance: clock.Advance, Responses: resp}
	rec := &collect.MemRecorder{}
	ver := "go1.27.3"
	if c.goVersion != "" {
		ver = c.goVersion
	}
	res := collect.Collect(collect.Options{
		Config: s.cfg, R: s.rf, Invoker: fake, Budget: b, Guard: func() collect.GuardValues { return s.guardValues(ver) },
		FS: modsnap.OS{}, MainModule: rfModule, SkipDirs: spec.RepoSkipDirs, Rec: rec, TypeCheck: e.TypeCheck,
	})
	return res, fake, rec
}

func derivedFromCall3(rec *collect.MemRecorder, cfg string) bool {
	for _, it := range rec.Items() {
		if it.File == "derived" && strings.Contains(string(it.Line), `"name":"`+cfg+`/closure"`) {
			return true
		}
	}
	return false
}

// LC5 are the stage and dependency controls (plan §5.2 LC-5, §5.3).
func LC5() []Variant {
	var out []Variant
	for _, row := range lc5Cases() {
		row := row
		out = append(out, Variant{
			Control: "LC-5", Name: row.c.name, Applies: always, Want: row.want,
			Run: func(e *Env) Outcome {
				fx, s, err := prepareRF(e, "LC-5 "+row.c.name)
				if err != nil {
					return Outcome{PrepErr: err}
				}
				clock := &fakeClock{t: time.Unix(1_000_000, 0)}
				b := collect.NewBudget(clock, clock.Now(), 120*time.Second)
				res, fake, rec := runCollect(e, s, row.c, clock, b, 0)
				lines := CheckLines(res.Checks)
				lines = append(lines, fmt.Sprintf("calls env=%d context=%d list=%d", fake.Calls(spec.CallEnv), fake.Calls(spec.CallContext), fake.Calls(spec.CallList)))
				if derivedFromCall3(rec, s.cfg.String()) {
					lines = append(lines, "derived from call 3: present")
				} else {
					lines = append(lines, "derived from call 3: none")
				}
				for _, f := range b.Refused() {
					lines = append(lines, FailureLine("admission", f))
				}
				return Outcome{Entered: Entered(res.Checks), Lines: lines, Terminal: true, CleanupErr: fx.cleanup()}
			},
		})
	}
	return out
}

// LC8 are the budget and admission controls (plan §5.2 LC-8): a unit
// started after W is not admitted (J-1 "budget insufficient", source
// budget) and nothing depending on it is evaluated; a result published
// after W is a failure that no later success restores.
func LC8() []Variant {
	allUnevaluated := func() []string {
		var out []string
		for _, id := range collect.CheckIDs() {
			out = append(out, id+" unevaluated")
		}
		return out
	}
	return []Variant{
		{
			Control: "LC-8", Name: "unit after W", Applies: always, MustEnter: []string{"admission"},
			Want: append([]string{"admission fail J-1 probe budget insufficient budget"}, append(allUnevaluated(),
				"collect admission fail J-1 configuration CFG budget insufficient budget")...),
			Run: func(e *Env) Outcome {
				fx, s, err := prepareRF(e, "LC-8 unit after W")
				if err != nil {
					return Outcome{PrepErr: err}
				}
				clock := &fakeClock{t: time.Unix(1_000_000, 0)}
				b := collect.NewBudget(clock, clock.Now(), 120*time.Second)
				clock.Advance(111 * time.Second) // past W = D − 10 s
				ck := judge.NewChecks([]string{"admission"})
				ck.Enter("admission", "")
				if b.AdmitUnit("probe") {
					ck.Hold("admission")
				} else {
					ck.Fail("admission", b.Refused()...)
				}
				res, _, _ := runCollect(e, s, lc5Case{name: "after W"}, clock, b, 0)
				lines := append(CheckLines(ck), CheckLines(res.Checks)...)
				for _, f := range b.Refused()[1:] {
					lines = append(lines, strings.Replace(FailureLine("collect admission", f), s.cfg.String(), "CFG", 1))
				}
				return Outcome{Entered: Entered(ck), Lines: lines, Terminal: true, CleanupErr: fx.cleanup()}
			},
		},
		{
			Control: "LC-8", Name: "result after W", Applies: always, MustEnter: []string{"A3"},
			Want: projection(map[string][]string{"A3": {"A3 fail J-1 call3-list call failed real"}}, allBSK, [3]int{1, 1, 1}, false),
			Run: func(e *Env) Outcome {
				fx, s, err := prepareRF(e, "LC-8 result after W")
				if err != nil {
					return Outcome{PrepErr: err}
				}
				clock := &fakeClock{t: time.Unix(1_000_000, 0)}
				b := collect.NewBudget(clock, clock.Now(), 120*time.Second)
				// The third call is admitted (W − now ≥ 5 s) but publishes
				// its successful result after W.
				res, fake, rec := runCollect(e, s, lc5Case{name: "late"}, clock, b, 115*time.Second)
				lines := CheckLines(res.Checks)
				lines = append(lines, fmt.Sprintf("calls env=%d context=%d list=%d", fake.Calls(spec.CallEnv), fake.Calls(spec.CallContext), fake.Calls(spec.CallList)))
				if derivedFromCall3(rec, s.cfg.String()) {
					lines = append(lines, "derived from call 3: present")
				} else {
					lines = append(lines, "derived from call 3: none")
				}
				return Outcome{Entered: Entered(res.Checks), Lines: lines, Terminal: true, CleanupErr: fx.cleanup()}
			},
		},
	}
}

// LC9 is the control-protocol control (plan §5.2 LC-9). It needs the
// pseudo-tool (a re-execution of the test binary), which phase 1 does not
// authorize (LE-5 ①), and the control protocol of §12.14, which phase 1
// does not implement. It is applicable on all three platforms, so it is
// recorded as applicable and not executed: a miss (plan §5.1), and every
// formal run's conclusion is invalid until it exists.
func LC9() []Variant {
	return []Variant{{
		Control: "LC-9", Name: "control protocol (pseudo-tool)", Applies: always,
		Run: func(*Env) Outcome {
			return Outcome{NotExecuted: "phase 1: the pseudo-tool is not authorized (LE-5 ①) and the §12.14 control protocol is not implemented"}
		},
	}}
}
