package typecheck

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// The fixtures are built in memory: stub standard-library packages
// (Standard, Module nil), a golang.org/x/sys stub, third-party packages and
// the package table that the third call would give for them. Imports are
// written by hand for every fixture; they are not computed from the
// sources, so the I-6 (d) check compares two independent inputs.

const toolchain = "go1.27.3"

var (
	linuxAMD64   = spec.Config{GOOS: "linux", GOARCH: "amd64"}
	linuxARM64   = spec.Config{GOOS: "linux", GOARCH: "arm64"}
	darwinARM64  = spec.Config{GOOS: "darwin", GOARCH: "arm64"}
	windowsAMD64 = spec.Config{GOOS: "windows", GOARCH: "amd64"}
)

// fx is one fixture package.
type fx struct {
	path      string
	name      string // package clause; default: the last path element
	kind      Kind
	mod       *Module // ThirdParty and XSys; nil for Std
	files     map[string]string
	imports   []string
	importMap map[string]string
}

func thirdParty(p, gover string, imports []string, files map[string]string) fx {
	return fx{path: p, kind: ThirdParty, mod: &Module{Path: p, Version: "v1.0.0", GoVersion: gover}, files: files, imports: imports}
}

func std(p string, imports []string, files map[string]string) fx {
	return fx{path: p, kind: Std, files: files, imports: imports}
}

// world is a package table with its byte source. The source logs every
// read and fails for "unsafe", for unknown files and for files in errs.
type world struct {
	pkgs  []Package
	src   map[string][]byte // "path name" → bytes
	errs  map[string]error
	reads []string
}

func build(fxs ...fx) *world {
	w := &world{src: map[string][]byte{}, errs: map[string]error{}}
	for _, f := range fxs {
		name := f.name
		if name == "" {
			name = path.Base(f.path)
		}
		p := Package{
			ImportPath: f.path,
			Dir:        "/fixture/" + f.path,
			Name:       name,
			Kind:       f.kind,
			Imports:    append([]string(nil), f.imports...),
			ImportMap:  f.importMap,
		}
		switch f.kind {
		case Std:
			p.Standard = true
		case Main:
			p.Module = &Module{Path: spec.MainModule, GoVersion: "1.27", Main: true}
		default:
			p.Module = f.mod
		}
		for n, body := range f.files {
			p.GoFiles = append(p.GoFiles, n)
			w.src[f.path+" "+n] = []byte(body)
		}
		sort.Strings(p.GoFiles)
		w.pkgs = append(w.pkgs, p)
	}
	return w
}

func (w *world) source(importPath, file string) ([]byte, error) {
	key := importPath + " " + file
	w.reads = append(w.reads, key)
	if importPath == "unsafe" {
		return nil, errors.New("unsafe must never be read")
	}
	if err := w.errs[key]; err != nil {
		return nil, err
	}
	b, ok := w.src[key]
	if !ok {
		return nil, fmt.Errorf("no file %s", key)
	}
	return b, nil
}

func (w *world) input(c spec.Config) Input {
	return Input{Config: c, Toolchain: toolchain, Packages: w.pkgs, Source: w.source, Admit: func(string) bool { return true }}
}

func (w *world) readsOf(importPath string) []string {
	var out []string
	for _, r := range w.reads {
		if strings.HasPrefix(r, importPath+" ") {
			out = append(out, r)
		}
	}
	return out
}

// stdStubs returns the stub standard library of a GOOS: syscall (with the
// GOOS's process objects and signals), os, os/exec, os/signal, context and
// unsafe. Bodies are present so that the declarations-only stripping runs.
func stdStubs(goos string) []fx {
	return []fx{
		std("unsafe", nil, map[string]string{"unsafe.go": "this is never read"}),
		std("syscall", nil, map[string]string{"syscall.go": syscallCommon, "zsys.go": syscallGOOS(goos)}),
		std("os", []string{"syscall"}, map[string]string{"os.go": osSrc}),
		std("os/exec", []string{"os"}, map[string]string{"exec.go": execSrc}),
		std("os/signal", []string{"context", "os"}, map[string]string{"signal.go": signalSrc}),
		std("context", nil, map[string]string{"context.go": contextSrc}),
	}
}

const syscallCommon = `package syscall

type Signal int

func (s Signal) Signal() {}

func (s Signal) String() string { return "signal " + string(rune('0'+int(s))) }

const (
	SIGINT  = Signal(0x2)
	SIGTERM = Signal(0xf)
)

type ProcAttr struct{ Dir string }

func StartProcess(argv0 string, argv []string, attr *ProcAttr) (pid int, handle uintptr, err error) {
	return 0, 0, nil
}

func Exec(argv0 string, argv []string, envv []string) error { return nil }
`

func syscallGOOS(goos string) string {
	switch goos {
	case "linux", "darwin":
		v := map[string]string{"linux": "0x11", "darwin": "0x14"}[goos]
		return `package syscall

const SIGCHLD = Signal(` + v + `)

type WaitStatus uint32

type Rusage struct{}

func ForkExec(argv0 string, argv []string, attr *ProcAttr) (pid int, err error) { return 0, nil }

func Wait4(pid int, wstatus *WaitStatus, options int, rusage *Rusage) (wpid int, err error) {
	return 0, nil
}
`
	case "windows":
		return `package syscall

type ProcessInformation struct{ Process uintptr }

func CreateProcess(appName *uint16, commandLine *uint16, inheritHandles bool, flags uint32, pi *ProcessInformation) error {
	return nil
}

func CreateProcessAsUser(token uintptr, appName *uint16, commandLine *uint16, pi *ProcessInformation) error {
	return nil
}
`
	}
	panic("no syscall stub for " + goos)
}

const osSrc = `package os

import "syscall"

type Signal interface {
	String() string
	Signal()
}

var (
	Interrupt Signal = syscall.SIGINT
	Kill      Signal = syscall.SIGTERM
)

type ProcAttr struct{ Dir string }

type ProcessState struct{ code int }

type Process struct{ Pid int }

func (p *Process) Wait() (*ProcessState, error) { return &ProcessState{}, nil }

func StartProcess(name string, argv []string, attr *ProcAttr) (*Process, error) {
	return &Process{}, nil
}
`

const execSrc = `package exec

import "os"

type Cmd struct {
	Path    string
	Process *os.Process
}

func Command(name string, arg ...string) *Cmd { return &Cmd{Path: name} }

func (c *Cmd) Start() error { return nil }

func (c *Cmd) Run() error { return c.Start() }

func (c *Cmd) Output() ([]byte, error) { return nil, c.Run() }

func (c *Cmd) CombinedOutput() ([]byte, error) { return nil, c.Run() }

func (c *Cmd) Wait() error { return nil }
`

const signalSrc = `package signal

import (
	"context"
	"os"
)

func Ignore(sig ...os.Signal) {}

func Ignored(sig os.Signal) bool { return false }

func Notify(c chan<- os.Signal, sig ...os.Signal) {}

func Reset(sig ...os.Signal) {}

func NotifyContext(parent context.Context, signals ...os.Signal) (ctx context.Context, stop context.CancelFunc) {
	Notify(nil, signals...)
	return parent, func() {}
}
`

const contextSrc = `package context

type Context interface {
	Done() <-chan struct{}
}

type CancelFunc func()

func Background() Context { return nil }
`

func xsysModule() *Module {
	return &Module{Path: "golang.org/x/sys", Version: "v0.49.0", GoVersion: "1.25"}
}

func xsysUnix(withWaitid bool) fx {
	src := `package unix

type WaitStatus uint32

type Rusage struct{}

func Exec(argv0 string, argv []string, envv []string) error { return nil }

func Wait4(pid int, wstatus *WaitStatus, options int, rusage *Rusage) (wpid int, err error) {
	return 0, nil
}
`
	if withWaitid {
		src += `
type Siginfo struct{}

func Waitid(idType int, id int, info *Siginfo, options int, rusage *Rusage) error { return nil }
`
	}
	return fx{path: "golang.org/x/sys/unix", kind: XSys, mod: xsysModule(), files: map[string]string{"syscall_unix.go": src}}
}

func xsysWindows() fx {
	return fx{path: "golang.org/x/sys/windows", kind: XSys, mod: xsysModule(), files: map[string]string{"zsyscall_windows.go": `//go:build windows

package windows

type ProcessInformation struct{ Process uintptr }

func CreateProcess(appName *uint16, commandLine *uint16, inheritHandles bool, flags uint32, pi *ProcessInformation) error {
	return nil
}

func CreateProcessAsUser(token uintptr, appName *uint16, commandLine *uint16, pi *ProcessInformation) error {
	return nil
}

func ShellExecute(hwnd uintptr, verb *uint16, file *uint16, args *uint16, cwd *uint16, showCmd int32) error {
	return nil
}

func GetCurrentProcessId() uint32 { return 0 }
`}}
}

// pkgOf returns a package's result or fails the test.
func pkgOf(t *testing.T, rep *Report, importPath string) PackageResult {
	t.Helper()
	res, ok := rep.Package(importPath)
	if !ok {
		t.Fatalf("no result for %s", importPath)
	}
	return res
}

// failureKinds renders a result's failures as "J-n kind item", sorted.
func failureKinds(r judge.Result) []string {
	var out []string
	for _, f := range r.Failures {
		out = append(out, fmt.Sprintf("%s %s %s", f.Class, f.Kind, f.Item))
	}
	sort.Strings(out)
	return out
}

func wantHold(t *testing.T, what string, r judge.Result) {
	t.Helper()
	if r.Status != judge.Hold {
		t.Errorf("%s: %s, want hold (reason %q, failures %v)", what, r.Status, r.Reason, r.Failures)
	}
}

func wantFail(t *testing.T, what string, r judge.Result, kinds ...string) {
	t.Helper()
	if r.Status != judge.Fail {
		t.Errorf("%s: %s, want fail (reason %q)", what, r.Status, r.Reason)
		return
	}
	sort.Strings(kinds)
	got := failureKinds(r)
	if strings.Join(got, "\n") != strings.Join(kinds, "\n") {
		t.Errorf("%s: failures\n%s\nwant\n%s", what, strings.Join(got, "\n"), strings.Join(kinds, "\n"))
	}
}

func wantUnevaluated(t *testing.T, what string, r judge.Result, reason string) {
	t.Helper()
	if r.Status != judge.Unevaluated || r.Reason != reason {
		t.Errorf("%s: %s %q (failures %v), want unevaluated %q", what, r.Status, r.Reason, r.Failures, reason)
	}
}
