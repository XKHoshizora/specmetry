package typecheck

import (
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// refuseAt admits every unit until the given one, refuses it, and records
// every call.
type refuseAt struct {
	unit  string
	calls []string
}

func (a *refuseAt) admit(unit string) bool {
	a.calls = append(a.calls, unit)
	return unit != a.unit
}

// Admission refused in the middle (§12.14; method record §6.2): the refused
// unit and every package not yet finished are not evaluated ("budget"),
// nothing new starts (no further admission, no read), the packages finished
// before keep their results, and Budget fails with one J-1 whose source is
// the budget.
func TestAdmissionRefusedAtFile(t *testing.T) {
	w := build(append(stdStubs("linux"), lc7())...)
	refused := FileUnit("os/exec", "exec.go")
	a := &refuseAt{unit: refused}
	in := w.input(linuxAMD64)
	in.Admit = a.admit
	rep := Run(in)

	wantCalls := []string{
		PackageUnit("context"), FileUnit("context", "context.go"),
		PackageUnit("syscall"), FileUnit("syscall", "syscall.go"), FileUnit("syscall", "zsys.go"),
		PackageUnit("os"), FileUnit("os", "os.go"),
		PackageUnit("os/exec"), refused,
	}
	if !reflect.DeepEqual(a.calls, wantCalls) {
		t.Errorf("admission calls\n%s\nwant\n%s", strings.Join(a.calls, "\n"), strings.Join(wantCalls, "\n"))
	}
	if got := w.readsOf("os/exec"); len(got) != 0 {
		t.Errorf("read after the refusal: %v", got)
	}
	if got := w.readsOf(lc7Path); len(got) != 0 {
		t.Errorf("read after the refusal: %v", got)
	}
	if rep.Refused != refused {
		t.Errorf("refused %q", rep.Refused)
	}
	wantFail(t, "budget", rep.Budget, "J-1 "+KindBudget+" "+refused)
	checkComplete(t, rep)
	if f := rep.Budget.Failures[0]; f.Source != judge.SourceBudget || f.Site != SiteAdmission {
		t.Errorf("budget failure %+v", f)
	}
	for _, p := range []string{"context", "syscall", "os"} {
		wantHold(t, p, pkgOf(t, rep, p).Load)
	}
	for _, p := range []string{"os/exec", "os/signal", lc7Path} {
		res := pkgOf(t, rep, p)
		wantUnevaluated(t, p, res.Load, ReasonBudget)
		if res.Depth != Skipped {
			t.Errorf("%s depth %s", p, res.Depth)
		}
	}
	l := pkgOf(t, rep, lc7Path)
	wantUnevaluated(t, "lc7 I-6(d)", l.ImportSet, ReasonBudget)
	wantUnevaluated(t, "lc7 I-7", l.Header, ReasonBudget)
	if l.Prefilter != PrefilterUnevaluated {
		t.Errorf("lc7 prefilter %s", l.Prefilter)
	}
	if len(rep.Findings) != 0 {
		t.Errorf("findings %v", rep.Findings)
	}
	if rep.Scan.Status != judge.Unevaluated || !strings.HasPrefix(rep.Scan.Reason, "not evaluated: budget; prefilter of "+lc7Path) {
		t.Errorf("scan %s %q", rep.Scan.Status, rep.Scan.Reason)
	}
	// The self-check still reads what was checked before the refusal; it
	// starts no read, package or call.
	byPkg := map[string]string{
		"os": SelfFound, "syscall": SelfFound,
		"os/exec": SelfUnevaluated, "os/signal": SelfUnevaluated,
		"golang.org/x/sys/unix": SelfNotRequired,
	}
	objs := spec.ObjectsFor("linux")
	if len(rep.SelfCheck) != len(objs) {
		t.Fatalf("%d self-check items for %d objects", len(rep.SelfCheck), len(objs))
	}
	for i, it := range rep.SelfCheck {
		if want := byPkg[objs[i].PkgPath]; it.Object != objs[i].FullName || it.Status != want {
			t.Errorf("self-check %s: %s, want %s %s", it.Object, it.Status, objs[i].FullName, want)
		}
	}
}

func TestAdmissionRefusedAtPackage(t *testing.T) {
	w := build(append(stdStubs("linux"), lc7())...)
	a := &refuseAt{unit: PackageUnit(lc7Path)}
	in := w.input(linuxAMD64)
	in.Admit = a.admit
	rep := Run(in)
	if last := a.calls[len(a.calls)-1]; last != PackageUnit(lc7Path) {
		t.Errorf("last admission %q", last)
	}
	for _, p := range []string{"context", "syscall", "os", "os/exec", "os/signal"} {
		wantHold(t, p, pkgOf(t, rep, p).Load)
	}
	wantUnevaluated(t, "lc7", pkgOf(t, rep, lc7Path).Load, ReasonBudget)
	wantFail(t, "budget", rep.Budget, "J-1 "+KindBudget+" "+PackageUnit(lc7Path))
	checkComplete(t, rep)
	if got := w.readsOf(lc7Path); len(got) != 0 {
		t.Errorf("read after the refusal: %v", got)
	}
	// Every std object was found; only the scan of lc7 is missing.
	for _, it := range rep.SelfCheck {
		if it.Status != SelfFound && it.Status != SelfNotRequired {
			t.Errorf("self-check %s: %s", it.Object, it.Status)
		}
	}
	wantUnevaluated(t, "scan", rep.Scan, "not evaluated: budget; prefilter of "+lc7Path)
}

// The table self-check (§12.9): an object the table expects on this GOOS
// and that is missing from a checked package fails the scan; a method with
// the wrong receiver (value instead of pointer) has another full name and
// is missing too; an object of a package that did not load is not
// evaluated, and objects of packages outside the table are not required.
func TestSelfCheck(t *testing.T) {
	stubs := stdStubs("linux")
	for i := range stubs {
		if stubs[i].path == "os/exec" {
			src := strings.Replace(execSrc, "func (c *Cmd) CombinedOutput() ([]byte, error) { return nil, c.Run() }\n", "", 1)
			src = strings.Replace(src, "func (c *Cmd) Wait() error", "func (c Cmd) Wait() error", 1)
			stubs[i].files = map[string]string{"exec.go": src}
		}
		if stubs[i].path == "os/signal" {
			stubs[i].files = map[string]string{"signal.go": signalSrc + "\nvar broken int = \"x\"\n"}
		}
	}
	rep := Run(build(stubs...).input(linuxAMD64))
	wantFail(t, "scan", rep.Scan,
		"J-3 "+KindObjectMissing+" (*os/exec.Cmd).CombinedOutput",
		"J-3 "+KindObjectMissing+" (*os/exec.Cmd).Wait")
	got := map[string]SelfCheckItem{}
	for _, it := range rep.SelfCheck {
		got[it.Object] = it
	}
	if it := got["(*os/exec.Cmd).Wait"]; it.Detail != "found (os/exec.Cmd).Wait, want (*os/exec.Cmd).Wait" {
		t.Errorf("Wait: %+v", it)
	}
	if it := got["(*os/exec.Cmd).CombinedOutput"]; it.Status != SelfMissing {
		t.Errorf("CombinedOutput: %+v", it)
	}
	for _, o := range []string{"os/signal.Ignore", "os/signal.Notify", "os/signal.Reset", "os/signal.NotifyContext"} {
		if got[o].Status != SelfUnevaluated {
			t.Errorf("%s: %+v", o, got[o])
		}
	}
	for _, o := range []string{"golang.org/x/sys/unix.Exec", "golang.org/x/sys/unix.Wait4", "golang.org/x/sys/unix.Waitid"} {
		if got[o].Status != SelfNotRequired {
			t.Errorf("%s: %+v", o, got[o])
		}
	}
	for _, o := range []string{"(*os/exec.Cmd).Run", "os.StartProcess", "(*os.Process).Wait", "syscall.ForkExec", "syscall.Wait4"} {
		if got[o].Status != SelfFound {
			t.Errorf("%s: %+v", o, got[o])
		}
	}

	// Only unevaluated items: the scan is not evaluated, not failed.
	stubs = stdStubs("linux")
	for i := range stubs {
		if stubs[i].path == "os/signal" {
			stubs[i].files = map[string]string{"signal.go": signalSrc + "\nvar broken int = \"x\"\n"}
		}
	}
	rep = Run(build(stubs...).input(linuxAMD64))
	if rep.Scan.Status != judge.Unevaluated || !strings.Contains(rep.Scan.Reason, "self-check of os/signal.Ignore") {
		t.Errorf("scan %s %q", rep.Scan.Status, rep.Scan.Reason)
	}

	// x/sys/unix on linux must define Waitid.
	rep = Run(build(append(stdStubs("linux"), xsysUnix(false))...).input(linuxAMD64))
	wantFail(t, "scan", rep.Scan, "J-3 "+KindObjectMissing+" golang.org/x/sys/unix.Waitid")
}
