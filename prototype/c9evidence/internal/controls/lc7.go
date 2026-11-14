package controls

import (
	"fmt"
	"sort"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/cond"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// The LC-7 fixture: a third-party package checked in the tool against
// stub standard-library packages (no real GOROOT is read). Lines and
// columns of the expectation were computed independently of the tool, by
// searching the fixture text (see the implementation record).
var lc7Files = map[string]string{
	"lc7.go": "package lc7\n\nimport (\n\t\"os/exec\"\n)\n\n// Runner is satisfied by *exec.Cmd; a call through it is interface dispatch.\ntype Runner interface{ Run() error }\n\n// CmdAlias is an alias of exec.Cmd.\ntype CmdAlias = exec.Cmd\n\nfunc Direct() error { return exec.Command(\"x\").Run() }\n\nfunc Value() func() error { return exec.Command(\"x\").Start }\n\nfunc Expr() func(*exec.Cmd) error { return (*exec.Cmd).Wait }\n\nfunc Alias(c *CmdAlias) ([]byte, error) { return c.Output() }\n\nfunc Iface(r Runner) error { return r.Run() }\n",
	"dot.go": "package lc7\n\nimport . \"os/exec\"\n\nfunc Dot() ([]byte, error) { return Command(\"x\").CombinedOutput() }\n",
	"signals.go": "package lc7\n\nimport (\n\t\"os\"\n\t\"os/signal\"\n\t\"syscall\"\n)\n\nvar sigs = []os.Signal{syscall.SIGINT}\n\n" +
		"func Signals(c chan os.Signal, v os.Signal) {\n\tsignal.Notify(c, syscall.Signal(0x11))\n\tsignal.Ignore(syscall.SIGINT)\n\tsignal.Reset(v)\n\tsignal.Notify(c, sigs...)\n\tsignal.Reset()\n}\n",
}

func stubSyscall(goos string) string {
	s := "package syscall\n\ntype Signal int\n\nfunc (s Signal) Signal()        {}\nfunc (s Signal) String() string { return \"\" }\n\nconst SIGINT = Signal(0x2)\n\ntype ProcAttr struct{}\n\n" +
		"func StartProcess(argv0 string, argv []string, attr *ProcAttr) (int, uintptr, error) { return 0, 0, nil }\n" +
		"func Exec(argv0 string, argv []string, envv []string) error { return nil }\n"
	switch goos {
	case "windows":
		s += "func CreateProcess() error { return nil }\nfunc CreateProcessAsUser() error { return nil }\n"
	default:
		s += fmt.Sprintf("const SIGCHLD = Signal(0x%x)\n\ntype WaitStatus uint32\ntype Rusage struct{}\n\n", spec.SIGCHLD[goos]) +
			"func ForkExec(argv0 string, argv []string, attr *ProcAttr) (int, error) { return 0, nil }\n" +
			"func Wait4(pid int, wstatus *WaitStatus, options int, rusage *Rusage) (int, error) { return 0, nil }\n"
	}
	return s
}

var lc7Std = map[string]string{
	"context": "package context\n\ntype Context interface{}\ntype CancelFunc func()\n",
	"os": "package os\n\ntype Signal interface {\n\tString() string\n\tSignal()\n}\n\ntype ProcAttr struct{}\ntype ProcessState struct{}\ntype Process struct{}\n\n" +
		"func StartProcess(name string, argv []string, attr *ProcAttr) (*Process, error) { return nil, nil }\nfunc (p *Process) Wait() (*ProcessState, error) { return nil, nil }\n",
	"os/exec": "package exec\n\ntype Cmd struct{}\n\nfunc Command(name string, arg ...string) *Cmd { return &Cmd{} }\nfunc (c *Cmd) Start() error { return nil }\nfunc (c *Cmd) Run() error { return nil }\n" +
		"func (c *Cmd) Output() ([]byte, error) { return nil, nil }\nfunc (c *Cmd) CombinedOutput() ([]byte, error) { return nil, nil }\nfunc (c *Cmd) Wait() error { return nil }\n",
	"os/signal": "package signal\n\nimport (\n\t\"context\"\n\t\"os\"\n)\n\nfunc Ignore(sig ...os.Signal) {}\nfunc Notify(c chan<- os.Signal, sig ...os.Signal) {}\nfunc Reset(sig ...os.Signal) {}\n" +
		"func NotifyContext(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) { return parent, func() {} }\n",
}

func lc7Input(goos string) (collect.TCInput, map[string]map[string]string) {
	cfg := spec.ConfigsFor(goos)[0]
	src := map[string]map[string]string{
		"context": {"context.go": lc7Std["context"]}, "os": {"os.go": lc7Std["os"]}, "os/exec": {"exec.go": lc7Std["os/exec"]},
		"os/signal": {"signal.go": lc7Std["os/signal"]}, "syscall": {"syscall.go": stubSyscall(goos)}, "example.com/lc7": lc7Files,
	}
	std := func(ip string, imports ...string) collect.TCPkg {
		var files []string
		for f := range src[ip] {
			files = append(files, f)
		}
		return collect.TCPkg{ImportPath: ip, Dir: "/stub/" + ip, Name: lastElem(ip), Standard: true, GoFiles: files, Imports: imports, Kind: cond.Std}
	}
	in := collect.TCInput{
		Config: cfg, Toolchain: spec.ToolVersion,
		Packages: []collect.TCPkg{
			std("context"), std("os"), std("os/exec"), std("os/signal", "context", "os"), std("syscall"),
			{ImportPath: "example.com/lc7", Dir: "/stub/lc7", Name: "lc7", HasModule: true, ModulePath: "example.com/lc7", ModuleVersion: "v1.0.0",
				GoVersion: "1.22", GoFiles: []string{"dot.go", "lc7.go", "signals.go"}, Imports: []string{"os", "os/exec", "os/signal", "syscall"}, Kind: cond.ThirdParty},
		},
		Source: func(p collect.TCPkg, name string) ([]byte, error) {
			b, ok := src[p.ImportPath][name]
			if !ok {
				return nil, fmt.Errorf("no fixture file %s/%s", p.ImportPath, name)
			}
			return []byte(b), nil
		},
		Admit: func(string) bool { return true },
	}
	return in, src
}

func lastElem(ip string) string {
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == '/' {
			return ip[i+1:]
		}
	}
	return ip
}

// lc7Want is the complete expectation, by GOOS (the SIGCHLD value differs).
func lc7Want(goos string) []string {
	const p = "example.com/lc7."
	f := func(obj, caller, file string, line, col int) string {
		return fmt.Sprintf("finding %s %s %s:%d:%d", obj, p+caller, file, line, col)
	}
	sig := func(obj string, line int, detail string) string {
		return fmt.Sprintf("diagnostic signal %s %sSignals signals.go:%d:9 %s", obj, p, line, detail)
	}
	first := "other constant"
	if goos == "linux" {
		first = "constant SIGCHLD"
	}
	return []string{
		f("(*os/exec.Cmd).Run", "Direct", "lc7.go", 13, 48),
		f("(*os/exec.Cmd).Start", "Value", "lc7.go", 15, 54),
		f("(*os/exec.Cmd).Wait", "Expr", "lc7.go", 17, 56),
		f("(*os/exec.Cmd).Output", "Alias", "lc7.go", 19, 52),
		f("(*os/exec.Cmd).CombinedOutput", "Dot", "dot.go", 5, 50),
		f("os/signal.Notify", "Signals", "signals.go", 12, 9),
		f("os/signal.Ignore", "Signals", "signals.go", 13, 9),
		f("os/signal.Reset", "Signals", "signals.go", 14, 9),
		f("os/signal.Notify", "Signals", "signals.go", 15, 9),
		f("os/signal.Reset", "Signals", "signals.go", 16, 9),
		sig("os/signal.Notify", 12, "direct call; arguments: "+first),
		sig("os/signal.Ignore", 13, "direct call; arguments: other constant"),
		sig("os/signal.Reset", 14, "direct call; arguments: runtime value"),
		sig("os/signal.Notify", 15, "direct call; arguments: runtime value (spread)"),
		sig("os/signal.Reset", 16, "direct call; arguments: no signal arguments"),
		"checked context", "checked example.com/lc7", "checked os", "checked os/exec", "checked os/signal", "checked syscall",
		"S hold",
	}
}

// LC7 is the object-scan control (plan §5.2 LC-7): a direct call, a
// method value, a method expression, a type alias and a dot import each
// give exactly one finding at an exact position; a call through an
// interface gives none (U-1, recorded, not read as unreachability); the
// signal arguments get their classes. In-tool type check; no process.
func LC7(goos string) []Variant {
	return []Variant{{
		Control: "LC-7", Name: "object scan", Applies: always, MustEnter: []string{"S"},
		Want: lc7Want(goos),
		Run: func(e *Env) Outcome {
			in, _ := lc7Input(e.GOOS)
			tc := e.TypeCheck
			if tc == nil {
				return Outcome{NotExecuted: "no type checker"}
			}
			ck := judge.NewChecks([]string{"S"})
			ck.Enter("S", "")
			out := tc.Check(in)
			if len(out.Failures)+len(out.SelfCheck) > 0 {
				ck.Fail("S", append(append([]judge.Failure(nil), out.Failures...), out.SelfCheck...)...)
			} else {
				ck.Hold("S")
			}
			var lines []string
			for _, fd := range out.Findings {
				lines = append(lines, fmt.Sprintf("finding %s %s %s:%d:%d", fd.Object, fd.Caller, fd.File, fd.Line, fd.Col))
			}
			for _, d := range out.Diagnostics {
				lines = append(lines, fmt.Sprintf("diagnostic %s %s %s %s:%d:%d %s", d.Kind, d.Object, d.Caller, d.File, d.Line, d.Col, d.Detail))
			}
			for _, x := range out.Failures {
				lines = append(lines, FailureLine("typecheck", x, noNorm))
			}
			for _, x := range out.SelfCheck {
				lines = append(lines, FailureLine("S", x, noNorm))
			}
			for _, u := range out.Unevaluated {
				lines = append(lines, "unevaluated "+u)
			}
			checked := append([]string(nil), out.Checked...)
			sort.Strings(checked)
			for _, c := range checked {
				lines = append(lines, "checked "+c)
			}
			if r, _ := ck.Get("S"); r.Status == judge.Hold {
				lines = append(lines, "S hold")
			}
			inst := instanceOf(ck)
			inst.Extra["typecheck"] = out
			return Outcome{Entered: Entered(ck), Lines: lines, Instance: inst, Terminal: true}
		},
	}}
}

// All returns every variant of LC-2 to LC-9, in order, for a GOOS. LC-1 is
// judged on R and LC-10 by TestMain.
func All(goos string) []Variant {
	var out []Variant
	out = append(out, LC2()...)
	out = append(out, LC3()...)
	out = append(out, LC4()...)
	out = append(out, LC5()...)
	out = append(out, LC6()...)
	out = append(out, LC7(goos)...)
	out = append(out, LC8()...)
	out = append(out, LC9()...)
	return out
}
