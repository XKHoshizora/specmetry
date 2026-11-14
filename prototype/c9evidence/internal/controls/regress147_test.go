package controls

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

func lc5Row(t *testing.T, prefix string) (lc5Case, []string) {
	t.Helper()
	for _, r := range lc5Cases() {
		if strings.HasPrefix(r.c.name, prefix) {
			return r.c, r.want
		}
	}
	t.Fatalf("no LC-5 row %s", prefix)
	return lc5Case{}, nil
}

func variantNamed(t *testing.T, vs []Variant, name string) Variant {
	t.Helper()
	for _, v := range vs {
		if v.Name == name {
			return v
		}
	}
	t.Fatalf("no variant %q", name)
	return Variant{}
}

// Review 147 B-147-4: a failure of the same check, class and kind at
// another site does not hit. LC5-19b expects the J-2 mismatch on GOWORK;
// the instance that actually runs is LC5-20a's, whose J-2 mismatch is on
// GOFLAGS.
func TestB1474SameClassAtTheWrongSiteIsAMiss(t *testing.T) {
	env, _ := newEnv(t)
	_, want19b := lc5Row(t, "LC5-19b")
	v := variantNamed(t, LC5(), "LC5-20a (V-20)")
	v.Want = want19b
	j := Judge(v, env.GOOS, v.Run(env), true)
	if j.Judgement != verdict.Miss {
		t.Fatalf("%s %v, want a miss", j.Judgement, j.Reasons)
	}
}

// An extra real failure is a line of its own and makes the variant miss:
// the fixture holds both a !nowebui file with exec and a cgo file, and the
// expectation is the exec variant's (one step 2 failure).
func TestB1474ExtraRealFailureIsNotFiltered(t *testing.T) {
	env, _ := newEnv(t)
	want := variantNamed(t, LC6(), "!nowebui file with exec").Want
	v := lc6("both", map[string]string{
		"extra.go": "//go:build !nowebui\n\npackage main\n\nimport \"os/exec\"\n\nfunc init() { _ = exec.Command(\"x\").Run() }\n",
		"cgo.go":   "//go:build cgo\n\npackage main\n",
	}, want)
	o := v.Run(env)
	j := Judge(v, env.GOOS, o, true)
	if j.Judgement != verdict.Miss {
		t.Fatalf("%s %v, want a miss", j.Judgement, j.Reasons)
	}
	if !strings.Contains(strings.Join(o.Lines, "\n"), "cgo.go production-only file not in the exception list real @ step 2 cgo.go") {
		t.Fatalf("the extra failure is not among the lines: %q", o.Lines)
	}
}

// The structured instance keeps the whole result: every failure with its
// real site, detail, source and serial; the calls and the instance's own
// records.
func TestB1474InstanceKeepsWholeResults(t *testing.T) {
	env, _ := newEnv(t)
	o := variantNamed(t, LC3(), "injection after a real nil close").Run(env)
	if o.Instance == nil {
		t.Fatal("no instance")
	}
	var f judge.Failure
	for _, it := range o.Instance.Checks {
		if it.ID == "I-5" && len(it.Result.Failures) == 1 {
			f = it.Result.Failures[0]
		}
	}
	if f.Site != "I-5 example.com/lc3@v1.0.0/lc3.go (P parse)" || f.Detail != "injected close error" || f.Source != judge.SourceInjected || f.Inject != 1 {
		t.Errorf("failure %+v", f)
	}
	if o.Instance.Extra["injections"] == nil || len(o.Instance.Entries) != 1 {
		t.Errorf("instance %+v", o.Instance)
	}

	o5 := variantNamed(t, LC5(), "LC5-0 (V-1)").Run(env)
	if o5.Instance == nil || len(o5.Instance.Calls) != 3 || len(o5.Instance.Records) == 0 || len(o5.Instance.Checks) != len(collect.CheckIDs()) {
		t.Fatalf("LC5-0 instance %+v", o5.Instance)
	}
}

// Review 147 N-147-2: the target's tags from the second call reach the
// consumer that classifies the files (B5's step 1), through the real
// collection path with the fake caller. The main package has a file
// constrained by a tag that only the second call's ToolTags can carry.
func TestN1472TargetTagsReachTheConsumer(t *testing.T) {
	env, _ := newEnv(t)
	type tc struct {
		name     string
		toolTags func(arch string) []string
		listed   []string
		want     []string
	}
	healthy := projection(nil, nil, [3]int{1, 1, 1}, true)
	stepOne := projection(map[string][]string{"B5": {
		"B5 fail J-3 example.com/rf classification differs from the third call real @ step 1 example.com/rf",
	}}, map[string]string{"S": "unevaluated", "K": "unevaluated"}, [3]int{1, 1, 1}, true)
	cases := []tc{
		{"tag carried and file listed", func(a string) []string { return []string{a + ".v1", "c9tag"} }, []string{"main.go", "tagged.go"}, healthy},
		{"tag not carried, file listed", func(a string) []string { return []string{a + ".v1"} }, []string{"main.go", "tagged.go"}, stepOne},
		{"tag not carried, file not listed", func(a string) []string { return []string{a + ".v1"} }, []string{"main.go"}, healthy},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fx, s, err := prepareRF(env, "N-147-2 "+c.name)
			if err != nil {
				t.Fatal(err)
			}
			defer fx.cleanup()
			if err := os.WriteFile(filepath.Join(s.rf, "tagged.go"), []byte("//go:build c9tag\n\npackage main\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			tags := c.toolTags(s.cfg.GOARCH)
			lc := lc5Case{name: c.name, ctx: func(ct *ctxText) { ct.ToolTags = tags }, mainGoFiles: c.listed}
			clock := &fakeClock{t: time.Unix(1_000_000, 0)}
			b := collect.NewBudget(clock, clock.Now(), 120*time.Second)
			res, fake, rec := runCollect(env, s, lc, clock, b, 0)
			got := append(CheckLines(res.Checks, normalizer(s.cfg.String(), fx.dir)), instanceLines(fake, rec, s.cfg.String())...)
			if strings.Join(got, "\n") != strings.Join(c.want, "\n") {
				t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.want, "\n"))
			}
			// The Context handed to MatchFile carried the second call's tags.
			if res.Analysis == nil || len(res.Analysis.Cond.Contexts) == 0 || strings.Join(res.Analysis.Cond.Contexts[0].ToolTags, ",") != strings.Join(tags, ",") {
				t.Errorf("recorded Context tags differ from the second call's")
			}
			_ = spec.ToolVersion
		})
	}
}
