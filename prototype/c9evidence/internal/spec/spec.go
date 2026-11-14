// Package spec holds the fixed values of the C-9 limited evidence tool:
// the template T2, the configurations and conditions, the three calls'
// arguments and the environment the tool sets, the frozen object table
// with the X1 and X4 diagnostic objects, the static candidates of the
// decision draft §2.3, and the repository's pure-embed exception.
//
// Every value is copied from a reviewed document; the document and section
// are named next to it. None of them is computed from what the tool sees.
package spec

import (
	"crypto/sha256"
	"encoding/hex"
)

// T2 is the second call's template (method record §3.3.1; implementation
// plan §12.14 item 2). It is passed as one argument, byte for byte.
const T2 = "{{with context}}c9-context v1{{println}}GOARCH {{printf `%q` .GOARCH}}{{println}}GOOS {{printf `%q` .GOOS}}{{println}}GOROOT {{printf `%q` .GOROOT}}{{println}}GOPATH {{printf `%q` .GOPATH}}{{println}}CgoEnabled {{.CgoEnabled}}{{println}}UseAllFiles {{.UseAllFiles}}{{println}}Compiler {{printf `%q` .Compiler}}{{println}}InstallSuffix {{printf `%q` .InstallSuffix}}{{println}}BuildTags {{len .BuildTags}}{{println}}{{range .BuildTags}}- {{printf `%q` .}}{{println}}{{end}}ToolTags {{len .ToolTags}}{{println}}{{range .ToolTags}}- {{printf `%q` .}}{{println}}{{end}}ReleaseTags {{len .ReleaseTags}}{{println}}{{range .ReleaseTags}}- {{printf `%q` .}}{{println}}{{end}}end{{println}}{{end}}"

// The binding of T2 (LC-10): its length and SHA-256.
const (
	T2Len    = 688
	T2SHA256 = "5dfab1fd5637f6990adee91420b93e6dc23122951f55305eb70cc76597bb2ad7"
)

// T2Binding reports T2's length and SHA-256, and whether both equal the
// fixed values. LC-10 checks it at the start of TestMain.
func T2Binding(t2 string) (n int, sum string, ok bool) {
	h := sha256.Sum256([]byte(t2))
	sum = hex.EncodeToString(h[:])
	return len(t2), sum, len(t2) == T2Len && sum == T2SHA256
}

// ToolVersion is the only toolchain the reviewed sources apply to
// (implementation plan §12.14 item 1; E-10 (i)).
const ToolVersion = "go1.27.3"

// MainModule is the repository's module path (method record §3.4).
const MainModule = "github.com/XKHoshizora/specmetry"

// Config is one analysed configuration.
type Config struct{ GOOS, GOARCH string }

func (c Config) String() string { return c.GOOS + "/" + c.GOARCH }

// ConfigsFor returns the two configurations a job on this GOOS analyses
// (plan §3.3; implementation plan §12.10), in the order they run.
func ConfigsFor(goos string) []Config {
	switch goos {
	case "linux":
		return []Config{{"linux", "amd64"}, {"linux", "arm64"}}
	case "darwin":
		return []Config{{"darwin", "arm64"}, {"darwin", "amd64"}}
	case "windows":
		return []Config{{"windows", "amd64"}, {"windows", "arm64"}}
	}
	return nil
}

// AnalysisTags are the build tags of the analysis condition; it also has
// CGO_ENABLED=0 (implementation plan §12.10).
var AnalysisTags = []string{"nowebui"}

// ProductionCgo are the production variants' CGO_ENABLED values; production
// uses the default tags (none). Both are checked until E-8 (b) is decided.
var ProductionCgo = []bool{false, true}

// EnvVar is one variable the tool sets for a call.
type EnvVar struct{ Name, Value string }

// ToolEnv is the environment the tool sets for each call of a
// configuration (implementation plan §12.10 loading step 1; E-9). Only these
// names and values are recorded (plan §6.1, G4-P1 row).
func ToolEnv(c Config) []EnvVar {
	return []EnvVar{
		{"GOOS", c.GOOS},
		{"GOARCH", c.GOARCH},
		{"CGO_ENABLED", "0"},
		{"GOFLAGS", ""},
		{"GOWORK", "off"},
		{"GOTOOLCHAIN", "local"},
		{"GOPROXY", "off"},
	}
}

// EnvNames are the eight variables the first call asks for, in order.
var EnvNames = []string{"GOROOT", "GOVERSION", "GOOS", "GOARCH", "CGO_ENABLED", "GOFLAGS", "GOWORK", "GOTOOLCHAIN"}

// ListFields are the fields the third call asks for (§12.10 loading step 1).
var ListFields = []string{"ImportPath", "Dir", "Name", "Standard", "Module", "GoFiles", "CgoFiles", "IgnoredGoFiles", "Imports", "ImportMap", "ForTest", "Error", "DepsErrors", "Incomplete"}

// CallKind is one of the three calls.
type CallKind int

// The three calls.
const (
	CallEnv     CallKind = 1
	CallContext CallKind = 2
	CallList    CallKind = 3
)

func (k CallKind) String() string {
	switch k {
	case CallEnv:
		return "call1-env"
	case CallContext:
		return "call2-context"
	case CallList:
		return "call3-list"
	}
	return "call?"
}

// Args returns the arguments of a call after the executable, exactly as
// passed (method record §3.2 to §3.4; implementation plan §12.14 item 2).
func Args(k CallKind) []string {
	switch k {
	case CallEnv:
		return append([]string{"env", "-json"}, EnvNames...)
	case CallContext:
		return []string{"list", "-e", "-f", T2, "unsafe"}
	case CallList:
		fields := ""
		for i, f := range ListFields {
			if i > 0 {
				fields += ","
			}
			fields += f
		}
		return []string{"list", "-e", "-deps", "-json=" + fields, "-tags", "nowebui", "./..."}
	}
	return nil
}

// Basis says why an object is in the table.
type Basis string

// The bases (method record §4.3.3, O rows).
const (
	Frozen Basis = "frozen"
	X1     Basis = "x1"
	X4     Basis = "x4"
)

// Object is one entry of the object table: the types.Func full name, the
// GOOS values it is defined for, and its basis. Signal says the signal
// rule's argument classification applies (os/signal objects).
type Object struct {
	FullName  string
	PkgPath   string
	Recv      string // receiver type name for methods ("Cmd"), "" for functions
	Name      string
	Platforms []string
	Basis     Basis
	Signal    bool
}

var all = []string{"darwin", "linux", "windows"}

// Objects is the frozen object table of the third-party scan with the X1
// and X4 objects (implementation plan §12.9; decision draft §2.1). Frozen
// objects give findings; X1 and X4 objects only diagnostics (plan §4.4).
// The batch 5 e2waitid.Waitid is a repository package and not in the
// third-party scan.
var Objects = []Object{
	{"(*os/exec.Cmd).Start", "os/exec", "Cmd", "Start", all, Frozen, false},
	{"(*os/exec.Cmd).Run", "os/exec", "Cmd", "Run", all, Frozen, false},
	{"(*os/exec.Cmd).Output", "os/exec", "Cmd", "Output", all, Frozen, false},
	{"(*os/exec.Cmd).CombinedOutput", "os/exec", "Cmd", "CombinedOutput", all, Frozen, false},
	{"(*os/exec.Cmd).Wait", "os/exec", "Cmd", "Wait", all, Frozen, false},
	{"os.StartProcess", "os", "", "StartProcess", all, Frozen, false},
	{"(*os.Process).Wait", "os", "Process", "Wait", all, Frozen, false},
	{"syscall.ForkExec", "syscall", "", "ForkExec", []string{"darwin", "linux"}, Frozen, false},
	{"syscall.StartProcess", "syscall", "", "StartProcess", all, Frozen, false},
	{"syscall.Exec", "syscall", "", "Exec", all, Frozen, false},
	{"syscall.Wait4", "syscall", "", "Wait4", []string{"darwin", "linux"}, Frozen, false},
	{"syscall.CreateProcess", "syscall", "", "CreateProcess", []string{"windows"}, X1, false},
	{"syscall.CreateProcessAsUser", "syscall", "", "CreateProcessAsUser", []string{"windows"}, X1, false},
	{"golang.org/x/sys/unix.Exec", "golang.org/x/sys/unix", "", "Exec", []string{"darwin", "linux"}, Frozen, false},
	{"golang.org/x/sys/unix.Wait4", "golang.org/x/sys/unix", "", "Wait4", []string{"darwin", "linux"}, Frozen, false},
	{"golang.org/x/sys/unix.Waitid", "golang.org/x/sys/unix", "", "Waitid", []string{"linux"}, Frozen, false},
	{"golang.org/x/sys/windows.CreateProcess", "golang.org/x/sys/windows", "", "CreateProcess", []string{"windows"}, X1, false},
	{"golang.org/x/sys/windows.CreateProcessAsUser", "golang.org/x/sys/windows", "", "CreateProcessAsUser", []string{"windows"}, X1, false},
	{"golang.org/x/sys/windows.ShellExecute", "golang.org/x/sys/windows", "", "ShellExecute", []string{"windows"}, X1, false},
	{"os/signal.Ignore", "os/signal", "", "Ignore", all, Frozen, true},
	{"os/signal.Notify", "os/signal", "", "Notify", all, Frozen, true},
	{"os/signal.Reset", "os/signal", "", "Reset", all, Frozen, true},
	{"os/signal.NotifyContext", "os/signal", "", "NotifyContext", all, X4, true},
}

// ObjectsFor returns the objects defined on this GOOS.
func ObjectsFor(goos string) []Object {
	var out []Object
	for _, o := range Objects {
		for _, p := range o.Platforms {
			if p == goos {
				out = append(out, o)
				break
			}
		}
	}
	return out
}

// PrefilterNames are the identifier names of the lexical prefilter: the
// last component of every object (X1 included), implementation plan §12.10.
func PrefilterNames() map[string]bool {
	m := map[string]bool{}
	for _, o := range Objects {
		m[o.Name] = true
	}
	return m
}

// SIGCHLD is the value of syscall.SIGCHLD on the GOOS values that define it
// (implementation plan §12.9 signal rule item 4).
var SIGCHLD = map[string]int64{"linux": 0x11, "darwin": 0x14}

// Candidate is one static candidate of the decision draft §2.3: an
// expected finding of a configuration. Columns and counts are not part of
// the comparison (plan §4.4).
type Candidate struct {
	ID      string
	Package string
	File    string // file name inside the module
	Caller  string // full name of the top-level function
	Object  string // object full name
	Line    int
}

const libc = "modernc.org/libc"

// CandidateModule is the module the candidates belong to.
const (
	CandidateModule  = "modernc.org/libc"
	CandidateVersion = "v1.77.1"
)

var linuxCandidates = []Candidate{
	{"L-1", libc, "libc_musl.go", libc + ".Xsystem", "(*os/exec.Cmd).Run", 1199},
	{"L-2", libc, "libc_musl.go", libc + ".Xsignal", "os/signal.Reset", 1039},
	{"L-3", libc, "libc_musl.go", libc + ".Xsignal", "os/signal.Ignore", 1041},
	{"L-4", libc, "libc_musl.go", libc + ".Xsignal", "os/signal.Notify", 1045},
}

var darwinCandidates = []Candidate{
	{"D-1", libc, "libc_darwin.go", libc + ".Xwaitpid", "golang.org/x/sys/unix.Wait4", 917},
	{"D-2", libc, "libc_darwin.go", libc + ".Xsystem", "(*os/exec.Cmd).Run", 1130},
	{"D-3", libc, "libc_darwin.go", libc + ".Xpause", "os/signal.Notify", 2361},
	{"D-4", libc, "libc_darwin.go", libc + ".Xexecve", "golang.org/x/sys/unix.Exec", 2828},
	{"D-5", libc, "libc_unix.go", libc + ".Xsignal", "os/signal.Reset", 86},
	{"D-6", libc, "libc_unix.go", libc + ".Xsignal", "os/signal.Ignore", 88},
	{"D-7", libc, "libc_unix.go", libc + ".Xsignal", "os/signal.Notify", 90},
}

var windowsCandidates = []Candidate{
	{"W-1", libc, "libc_windows.go", libc + ".Xsystem", "(*os/exec.Cmd).Run", 1089},
}

// CandidatesFor returns the expected findings of a configuration.
func CandidatesFor(c Config) []Candidate {
	switch c.GOOS {
	case "linux":
		return append([]Candidate(nil), linuxCandidates...)
	case "darwin":
		return append([]Candidate(nil), darwinCandidates...)
	case "windows":
		return append([]Candidate(nil), windowsCandidates...)
	}
	return nil
}

// EmbedException is one file of a pure-embed exception list (implementation
// plan §12.10 step 2): checked by path, SHA-256, constraint line and import
// set.
type EmbedException struct {
	Path       string // slash-separated, relative to the module root
	Size       int
	SHA256     string
	Constraint string
	Imports    []string // sorted
}

// RepoEmbedExceptions is the repository's list: only web/embed.go, recorded
// at master 9137279 (533 bytes).
var RepoEmbedExceptions = []EmbedException{{
	Path:       "web/embed.go",
	Size:       533,
	SHA256:     "1115d9dd4ca8fb660b875c5eb75533dffbbe9fb45808eab61a62d97596fe286f",
	Constraint: "//go:build !nowebui",
	Imports:    []string{"embed", "io/fs"},
}}

// WatchPackages are the packages of the decision draft §2.7 whose presence
// in a production closure is a new fact (LQ-6): reported by K as a
// difference, never added to any exception.
var WatchPackages = []string{
	"golang.org/x/mod/zip",
	"golang.org/x/mod/gosumcheck",
	"golang.org/x/text/internal/testtext",
	"golang.org/x/crypto/acme/internal/acmeprobe",
	"golang.org/x/crypto/internal/testenv",
	"github.com/aymanbagabas/go-pty",
	"github.com/creack/pty",
	"github.com/u-root/u-root",
}

// RepoSkipDirs are the repository's directories outside the production
// packages (implementation plan §12.10 step 2 and range).
var RepoSkipDirs = []string{"internal/testutil"}
