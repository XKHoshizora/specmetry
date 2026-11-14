package typecheck

import (
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
)

// Review 147 B-147-3, path two: an early exit must first keep every error
// already observed in the package (parse, constraint line, language version)
// and only mark what never ran as not evaluated; the budget failure stays a
// separate result and never replaces an earlier real failure. Expectations
// are written from the fixture, not from the tool's output.

const regressPath = "example.com/r147"

func regressWorld(gover string, files map[string]string) *world {
	return build(thirdParty(regressPath, gover, nil, files))
}

func runRefusing(w *world, unit string) *Report {
	a := &refuseAt{unit: unit}
	in := w.input(linuxAMD64)
	in.Admit = a.admit
	return Run(in)
}

func TestB1473ParseErrorKeptWhenTheNextFileIsRefused(t *testing.T) {
	w := regressWorld("1.22", map[string]string{
		"a.go": "package r147\n\nfunc broken( {\n",
		"b.go": "package r147\n",
	})
	rep := runRefusing(w, FileUnit(regressPath, "b.go"))
	res, _ := rep.Package(regressPath)
	if res.Load.Status != judge.Fail {
		t.Fatalf("load %s %q: the parse error of a.go was observed before b.go's refusal and must be kept", res.Load.Status, res.Load.Reason)
	}
	for _, f := range res.Load.Failures {
		if f.Kind != KindParseError || f.Item != regressPath+"/a.go" {
			t.Errorf("unexpected load failure %+v", f)
		}
	}
	wantFail(t, "budget", rep.Budget, "J-1 "+KindBudget+" "+FileUnit(regressPath, "b.go"))
	wantUnevaluated(t, "I-6 (d)", res.ImportSet, ReasonBudget)
}

func TestB1473ConstraintErrorKeptWhenTheNextFileIsRefused(t *testing.T) {
	w := regressWorld("1.22", map[string]string{
		"a.go": "//go:build linux\n//go:build amd64\n\npackage r147\n",
		"b.go": "package r147\n",
	})
	rep := runRefusing(w, FileUnit(regressPath, "b.go"))
	res, _ := rep.Package(regressPath)
	wantFail(t, "I-7", res.Header, "J-3 "+KindMultipleGoBuild+" "+regressPath+"/a.go")
	wantUnevaluated(t, "load", res.Load, ReasonBudget)
	wantFail(t, "budget", rep.Budget, "J-1 "+KindBudget+" "+FileUnit(regressPath, "b.go"))
}

func TestB1473VersionErrorKeptWhenAFileIsRefused(t *testing.T) {
	w := regressWorld("1.99", map[string]string{"a.go": "package r147\n"})
	rep := runRefusing(w, FileUnit(regressPath, "a.go"))
	res, _ := rep.Package(regressPath)
	wantFail(t, "load", res.Load, "J-3 "+KindVersionTooHigh+" "+regressPath)
	wantFail(t, "budget", rep.Budget, "J-1 "+KindBudget+" "+FileUnit(regressPath, "a.go"))
}

func TestB1473BudgetOnlyHealthyControl(t *testing.T) {
	w := regressWorld("1.22", map[string]string{"a.go": "package r147\n", "b.go": "package r147\n"})
	rep := runRefusing(w, FileUnit(regressPath, "b.go"))
	res, _ := rep.Package(regressPath)
	wantUnevaluated(t, "load", res.Load, ReasonBudget)
	wantUnevaluated(t, "I-7", res.Header, ReasonBudget)
	wantUnevaluated(t, "I-6 (d)", res.ImportSet, ReasonBudget)
	wantFail(t, "budget", rep.Budget, "J-1 "+KindBudget+" "+FileUnit(regressPath, "b.go"))
	checkComplete(t, rep)
}

// Review 147 §4 item 3: after a closed block comment, a //go:build on the
// next line is the constraint line; on the same line it is not.
func TestConstraintLineAfterABlockComment(t *testing.T) {
	next, err := ConstraintLine([]byte("/* x */\n//go:build a\npackage p\n"))
	if err != nil || next != "//go:build a" {
		t.Errorf("next line: %q %v", next, err)
	}
	same, err := ConstraintLine([]byte("/* x */ //go:build a\npackage p\n"))
	if err != nil || same != "" {
		t.Errorf("same line: %q %v", same, err)
	}
}
