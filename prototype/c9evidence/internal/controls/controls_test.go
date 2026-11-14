package controls

import (
	"runtime"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

// runVariants runs each variant as the formal run would and returns the
// outer judgements. These unit tests are not formal LC evidence.
func runVariants(t *testing.T, vs []Variant, env *Env) []verdict.Variant {
	t.Helper()
	var out []verdict.Variant
	for _, v := range vs {
		ok, _ := v.Applies(env.GOOS)
		var o Outcome
		if ok {
			o = v.Run(env)
		}
		out = append(out, Judge(v, env.GOOS, o, ok))
	}
	return out
}

func newEnv(t *testing.T) (*Env, *MemOuter) {
	o := &MemOuter{}
	return &Env{GOOS: runtime.GOOS, TempRoot: t.TempDir(), Outer: o, TypeCheck: testTypeChecker()}, o
}

func TestControlsWithoutTypeCheck(t *testing.T) {
	env, outer := newEnv(t)
	var vs []Variant
	vs = append(vs, LC2()...)
	vs = append(vs, LC3()...)
	vs = append(vs, LC6()...)
	vs = append(vs, LC8()...)
	for _, j := range runVariants(t, vs, env) {
		if j.Judgement == verdict.Hit {
			continue
		}
		// A symbolic link this machine may not create is a failed outer
		// preparation: a miss, never "not applicable" (plan §5.6 item 4).
		if j.Control == "LC-2" && j.Variant == "symbolic link" && len(j.Reasons) == 1 && strings.HasPrefix(j.Reasons[0], "outer preparation failed") {
			t.Logf("LC-2 symbolic link: %s (miss on this machine)", j.Reasons[0])
			continue
		}
		t.Errorf("%s %s: %s %v", j.Control, j.Variant, j.Judgement, j.Reasons)
	}
	if _, errs, open := outer.State(); len(errs) > 0 && !strings.Contains(strings.Join(errs, ";"), "symbolic link") || open != 0 {
		t.Errorf("outer operations: errors %v open %d", errs, open)
	}
}

func TestLC9IsAMiss(t *testing.T) {
	env, _ := newEnv(t)
	js := runVariants(t, LC9(), env)
	if len(js) != 1 || js[0].Judgement != verdict.Miss {
		t.Fatalf("%v", js)
	}
}

// TestJudgeRejectsExtraRealErrors checks the outer rules on synthetic
// outcomes: an extra real failure next to the expected injected one is a
// miss; a cleanup failure after a matching result is pending; a missing
// entry record is a miss.
func TestJudgeRules(t *testing.T) {
	v := Variant{Control: "LC-x", Name: "v", Applies: always, MustEnter: []string{"I-5"},
		Want: []string{"I-5 fail J-1 f close error injected#1"}}
	cases := []struct {
		name string
		o    Outcome
		want verdict.Judgement
	}{
		{"hit", Outcome{Entered: []string{"I-5"}, Lines: []string{"I-5 fail J-1 f close error injected#1"}, Terminal: true}, verdict.Hit},
		{"extra real error at the same site", Outcome{Entered: []string{"I-5"}, Lines: []string{"I-5 fail J-1 f close error injected#1", "I-5 fail J-1 f close error real"}, Terminal: true}, verdict.Miss},
		{"real instead of injected", Outcome{Entered: []string{"I-5"}, Lines: []string{"I-5 fail J-1 f close error real"}, Terminal: true}, verdict.Miss},
		{"no entry", Outcome{Lines: []string{"I-5 fail J-1 f close error injected#1"}, Terminal: true}, verdict.Miss},
		{"not terminal", Outcome{Entered: []string{"I-5"}, Terminal: false}, verdict.Pending},
		{"cleanup failed", Outcome{Entered: []string{"I-5"}, Lines: []string{"I-5 fail J-1 f close error injected#1"}, Terminal: true, CleanupErr: errString("EBUSY")}, verdict.Pending},
		{"prep failed", Outcome{PrepErr: errString("EPERM")}, verdict.Miss},
	}
	for _, c := range cases {
		if got := Judge(v, "linux", c.o, true).Judgement; got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
	na := v
	na.Applies = func(goos string) (bool, string) { return goos != "windows", "no such thing on windows" }
	if got := Judge(na, "windows", Outcome{}, false).Judgement; got != verdict.NotApplicable {
		t.Errorf("not applicable: %s", got)
	}
	if got := Judge(na, "linux", Outcome{}, false).Judgement; got != verdict.Miss {
		t.Errorf("applicable but skipped: %s", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
