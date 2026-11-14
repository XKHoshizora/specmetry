package typecheck

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// LC-7 (limited evidence plan §5.2): one direct call, one method value, one
// method expression, one use through a type alias and one through a dot
// import, each exactly one finding at the line and byte column written
// below; a call through an interface, no finding; signal calls with the
// SIGCHLD constant, an equal constant, another constant, a variable, an
// interface conversion, `sigs...`, no arguments and an indirect reference.
//
// The positions are counted by hand on the sources below (a tab is one
// byte, columns are 1-based).

const lc7Path = "example.com/lc7"

const lc7Forms = `package lc7

import (
	. "os"
	"os/exec"
)

// C is an alias of exec.Cmd.
type C = exec.Cmd

// Waiter is satisfied by *exec.Cmd.
type Waiter interface{ Wait() error }

var _ Waiter = (*exec.Cmd)(nil)

func Direct(c *exec.Cmd) error {
	return c.Run()
}

func MethodValue(c *exec.Cmd) func() error {
	return c.Start
}

func MethodExpr() func(*exec.Cmd) ([]byte, error) {
	return (*exec.Cmd).Output
}

func ViaAlias(c *C) ([]byte, error) {
	return c.CombinedOutput()
}

func ViaDot() (*Process, error) {
	return StartProcess("tool", nil, nil)
}

func ViaInterface(w Waiter) error {
	return w.Wait()
}
`

const lc7Signals = `//go:build !windows

package lc7

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

var sigs = []os.Signal{syscall.SIGINT}

func Signals(ctx context.Context, c chan os.Signal) {
	signal.Notify(c, syscall.SIGCHLD)
	signal.Ignore(syscall.SIGINT, syscall.Signal(0x11))
	var s os.Signal = syscall.SIGTERM
	signal.Reset(s)
	signal.Notify(c, sigs...)
	signal.Reset()
	signal.NotifyContext(ctx, os.Signal(syscall.SIGTERM))
	f := signal.Ignore
	f()
}
`

func lc7() fx {
	return thirdParty(lc7Path, "1.22", []string{"context", "os", "os/exec", "os/signal", "syscall"},
		map[string]string{"forms.go": lc7Forms, "signals.go": lc7Signals})
}

func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// lc7Expected is the complete expected output for a GOOS that has SIGCHLD.
// sigchld11 is the class of syscall.Signal(0x11) on that GOOS.
func lc7Expected(sigchld11 string) (findings, diags []Ref, signals []SignalUse) {
	forms := func(line, col int, caller, object string) Ref {
		return Ref{Basis: spec.Frozen, Package: lc7Path, File: "forms.go", Caller: lc7Path + "." + caller,
			Object: object, Line: line, Column: col, Constraint: "", FileSHA256: sum(lc7Forms)}
	}
	sig := func(basis spec.Basis, line, col int, object string) Ref {
		return Ref{Basis: basis, Package: lc7Path, File: "signals.go", Caller: lc7Path + ".Signals",
			Object: object, Line: line, Column: col, Constraint: "//go:build !windows", FileSHA256: sum(lc7Signals)}
	}
	findings = []Ref{
		forms(17, 11, "Direct", "(*os/exec.Cmd).Run"),
		forms(21, 11, "MethodValue", "(*os/exec.Cmd).Start"),
		forms(25, 21, "MethodExpr", "(*os/exec.Cmd).Output"),
		forms(29, 11, "ViaAlias", "(*os/exec.Cmd).CombinedOutput"),
		forms(33, 9, "ViaDot", "os.StartProcess"),
		sig(spec.Frozen, 15, 9, "os/signal.Notify"),
		sig(spec.Frozen, 16, 9, "os/signal.Ignore"),
		sig(spec.Frozen, 18, 9, "os/signal.Reset"),
		sig(spec.Frozen, 19, 9, "os/signal.Notify"),
		sig(spec.Frozen, 20, 9, "os/signal.Reset"),
		sig(spec.Frozen, 22, 14, "os/signal.Ignore"),
	}
	diags = []Ref{sig(spec.X4, 21, 9, "os/signal.NotifyContext")}
	signals = []SignalUse{
		{Ref: sig(spec.Frozen, 15, 9, "os/signal.Notify"), Direct: true, Classes: []string{SignalConstSIGCHLD}},
		{Ref: sig(spec.Frozen, 16, 9, "os/signal.Ignore"), Direct: true, Classes: []string{SignalOtherConst, sigchld11}},
		{Ref: sig(spec.Frozen, 18, 9, "os/signal.Reset"), Direct: true, Classes: []string{SignalRuntime}},
		{Ref: sig(spec.Frozen, 19, 9, "os/signal.Notify"), Direct: true, Spread: true, Classes: []string{SignalRuntime}},
		{Ref: sig(spec.Frozen, 20, 9, "os/signal.Reset"), Direct: true, Classes: []string{SignalNoArgs}},
		{Ref: sig(spec.X4, 21, 9, "os/signal.NotifyContext"), Direct: true, Classes: []string{SignalRuntime}},
		{Ref: sig(spec.Frozen, 22, 14, "os/signal.Ignore"), Direct: false},
	}
	return findings, diags, signals
}

func TestLC7(t *testing.T) {
	for _, tc := range []struct {
		cfg       spec.Config
		sigchld11 string
	}{
		{linuxAMD64, SignalConstSIGCHLD}, // 0x11 is linux's SIGCHLD
		{darwinARM64, SignalOtherConst},  // darwin's SIGCHLD is 0x14
	} {
		t.Run(tc.cfg.String(), func(t *testing.T) {
			w := build(append(stdStubs(tc.cfg.GOOS), lc7())...)
			rep := Run(w.input(tc.cfg))

			wantHold(t, "setup", rep.Setup)
			wantHold(t, "budget", rep.Budget)
			wantHold(t, "scan", rep.Scan)
			checkComplete(t, rep)
			p := pkgOf(t, rep, lc7Path)
			if p.Depth != Full || p.Prefilter != PrefilterHit || !p.Scanned || p.GoVersion != "go1.22" {
				t.Errorf("lc7: depth %s prefilter %s scanned %v version %q", p.Depth, p.Prefilter, p.Scanned, p.GoVersion)
			}
			wantHold(t, "lc7 load", p.Load)
			wantHold(t, "lc7 I-6(d)", p.ImportSet)
			wantHold(t, "lc7 I-7", p.Header)

			findings, diags, signals := lc7Expected(tc.sigchld11)
			if !reflect.DeepEqual(rep.Findings, findings) {
				t.Errorf("findings:\n%s\nwant\n%s", refsString(rep.Findings), refsString(findings))
			}
			if !reflect.DeepEqual(rep.Diagnostics, diags) {
				t.Errorf("diagnostics:\n%s\nwant\n%s", refsString(rep.Diagnostics), refsString(diags))
			}
			if !reflect.DeepEqual(rep.Signals, signals) {
				t.Errorf("signals:\n%+v\nwant\n%+v", rep.Signals, signals)
			}

			// Each of the five forms gives exactly one finding, and the
			// interface call (forms.go:37, (example.com/lc7.Waiter).Wait)
			// gives none.
			perCaller := map[string]int{}
			for _, f := range rep.Findings {
				if f.File == "forms.go" {
					perCaller[f.Caller]++
				}
			}
			for _, c := range []string{"Direct", "MethodValue", "MethodExpr", "ViaAlias", "ViaDot"} {
				if perCaller[lc7Path+"."+c] != 1 {
					t.Errorf("%s: %d findings, want 1", c, perCaller[lc7Path+"."+c])
				}
			}
			if n := perCaller[lc7Path+".ViaInterface"]; n != 0 {
				t.Errorf("interface call gave %d findings", n)
			}

			for _, s := range []string{"syscall", "os", "os/exec", "os/signal", "context"} {
				r := pkgOf(t, rep, s)
				if r.Depth != Declarations || r.Prefilter != PrefilterNA {
					t.Errorf("%s: depth %s prefilter %s", s, r.Depth, r.Prefilter)
				}
				wantHold(t, s, r.Load)
				if r.ImportSet.Status != judge.Unset || r.Header.Status != judge.Unset {
					t.Errorf("%s: I-6(d) %s, I-7 %s; they do not apply to the standard library", s, r.ImportSet.Status, r.Header.Status)
				}
			}
			if got := w.readsOf("unsafe"); len(got) != 0 {
				t.Errorf("unsafe was read: %v", got)
			}
		})
	}
}

func refsString(rs []Ref) string {
	var b strings.Builder
	for _, r := range rs {
		b.WriteString(r.String())
		b.WriteString(" [" + r.Constraint + "]\n")
	}
	return b.String()
}

// The forms the LC-7 list does not name: embedded promotion, a
// package-level function value (caller PackageLevel), a reference inside a
// function literal (caller: the enclosing top-level function), a method of
// a generic type (caller without type arguments), init, and a method named
// like a table object that is not one.
const formsSrc = `package forms

import (
	"os"
	"os/exec"
)

type Wrapped struct{ *exec.Cmd }

var startFn = os.StartProcess

func Promoted(w Wrapped) error {
	return w.Wait()
}

func Literal(c *exec.Cmd) func() error {
	return func() error { return c.Run() }
}

type Box[T any] struct{ v T }

func (b *Box[T]) Run(c *exec.Cmd) error {
	return c.Start()
}

func init() {
	_ = startFn
	_ = (&Box[int]{}).Run
	_ = exec.Command("x").Output
}
`

func TestOtherForms(t *testing.T) {
	const p = "example.com/forms"
	w := build(append(stdStubs("linux"), thirdParty(p, "1.21", []string{"os", "os/exec"}, map[string]string{"forms.go": formsSrc}))...)
	rep := Run(w.input(linuxAMD64))
	wantHold(t, "scan", rep.Scan)
	ref := func(line, col int, caller, object string) Ref {
		return Ref{Basis: spec.Frozen, Package: p, File: "forms.go", Caller: caller, Object: object,
			Line: line, Column: col, FileSHA256: sum(formsSrc)}
	}
	want := []Ref{
		ref(10, 18, PackageLevel, "os.StartProcess"),
		ref(13, 11, p+".Promoted", "(*os/exec.Cmd).Wait"),
		ref(17, 33, p+".Literal", "(*os/exec.Cmd).Run"),
		ref(23, 11, "(*"+p+".Box).Run", "(*os/exec.Cmd).Start"),
		ref(29, 24, p+".init", "(*os/exec.Cmd).Output"),
	}
	if !reflect.DeepEqual(rep.Findings, want) {
		t.Errorf("findings:\n%s\nwant\n%s", refsString(rep.Findings), refsString(want))
	}
	if len(rep.Diagnostics) != 0 || len(rep.Signals) != 0 {
		t.Errorf("diagnostics %v signals %v", rep.Diagnostics, rep.Signals)
	}
}

// Windows: no SIGCHLD, so an equal constant is another constant; the X1
// objects give diagnostics, not findings; an ordinary x/sys/windows wrapper
// gives nothing; the self-check covers syscall's and x/sys/windows' X1
// objects.
const winxSrc = `package winx

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/sys/windows"
)

func Spawn(c *exec.Cmd, ch chan os.Signal) error {
	_ = syscall.CreateProcess(nil, nil, false, 0, nil)
	_ = windows.ShellExecute(0, nil, nil, nil, nil, 1)
	_ = windows.GetCurrentProcessId()
	signal.Notify(ch, syscall.Signal(0x11), syscall.SIGINT)
	return c.Run()
}
`

func TestWindowsX1AndSignals(t *testing.T) {
	const p = "example.com/winx"
	fxs := append(stdStubs("windows"), xsysWindows(),
		thirdParty(p, "1.22", []string{"golang.org/x/sys/windows", "os", "os/exec", "os/signal", "syscall"}, map[string]string{"winx.go": winxSrc}))
	w := build(fxs...)
	rep := Run(w.input(windowsAMD64))
	wantHold(t, "scan", rep.Scan)
	ref := func(basis spec.Basis, line, col int, object string) Ref {
		return Ref{Basis: basis, Package: p, File: "winx.go", Caller: p + ".Spawn", Object: object,
			Line: line, Column: col, FileSHA256: sum(winxSrc)}
	}
	wantFindings := []Ref{ref(spec.Frozen, 16, 9, "os/signal.Notify"), ref(spec.Frozen, 17, 11, "(*os/exec.Cmd).Run")}
	wantDiags := []Ref{ref(spec.X1, 13, 14, "syscall.CreateProcess"), ref(spec.X1, 14, 14, "golang.org/x/sys/windows.ShellExecute")}
	wantSignals := []SignalUse{{Ref: ref(spec.Frozen, 16, 9, "os/signal.Notify"), Direct: true, Classes: []string{SignalOtherConst, SignalOtherConst}}}
	if !reflect.DeepEqual(rep.Findings, wantFindings) {
		t.Errorf("findings:\n%s\nwant\n%s", refsString(rep.Findings), refsString(wantFindings))
	}
	if !reflect.DeepEqual(rep.Diagnostics, wantDiags) {
		t.Errorf("diagnostics:\n%s\nwant\n%s", refsString(rep.Diagnostics), refsString(wantDiags))
	}
	if !reflect.DeepEqual(rep.Signals, wantSignals) {
		t.Errorf("signals %+v, want %+v", rep.Signals, wantSignals)
	}
	xw := pkgOf(t, rep, "golang.org/x/sys/windows")
	if xw.Depth != Declarations || xw.Files[0].Constraint != "//go:build windows" {
		t.Errorf("x/sys/windows: depth %s constraint %q", xw.Depth, xw.Files[0].Constraint)
	}
	wantHold(t, "x/sys/windows I-6(d)", xw.ImportSet)
	wantHold(t, "x/sys/windows I-7", xw.Header)
	var found []string
	for _, it := range rep.SelfCheck {
		if it.Status != SelfFound {
			t.Errorf("self-check %s: %s %s", it.Object, it.Status, it.Detail)
		}
		found = append(found, it.Object)
	}
	var want []string
	for _, o := range spec.ObjectsFor("windows") {
		want = append(want, o.FullName)
	}
	if !reflect.DeepEqual(found, want) {
		t.Errorf("self-check objects %v, want %v", found, want)
	}
}

// A reference to a table object on a GOOS the table does not list for it
// fails the scan instead of disappearing: a darwin x/sys/unix stub that
// (against the table) defines Waitid.
func TestOffPlatformReference(t *testing.T) {
	const p = "example.com/offp"
	src := `package offp

import "golang.org/x/sys/unix"

func W() error { return unix.Waitid(0, 0, nil, 0, nil) }
`
	w := build(append(stdStubs("darwin"), xsysUnix(true),
		thirdParty(p, "1.22", []string{"golang.org/x/sys/unix"}, map[string]string{"offp.go": src}))...)
	rep := Run(w.input(darwinARM64))
	wantFail(t, "scan", rep.Scan, "J-3 "+KindOffPlatform+" golang.org/x/sys/unix.Waitid")
	if len(rep.Findings) != 0 {
		t.Errorf("findings %v", rep.Findings)
	}
}
