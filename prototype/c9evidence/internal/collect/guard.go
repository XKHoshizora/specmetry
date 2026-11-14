package collect

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// GuardValues are what the A1-a guard reads (implementation plan §12.14
// item 1; plan §6.1 "工具链"): the analyser's own version, the candidate
// found as "go" on PATH after resolving symbolic links, its layout, the
// first line of <root>/VERSION, and the executable's SHA-256. Errors are
// kept as text; reading them is the guard's job, judging them is
// EvaluatePre's.
type GuardValues struct {
	AnalyzerVersion string
	Candidate       string // absolute, symbolic links resolved
	Root            string // <root>, the parent of the bin directory
	ExeName         string // "go" or "go.exe"
	ExeRegular      bool
	SrcCmdGo        bool // <root>/src/cmd/go is a directory
	VersionLine     string
	ExeSHA256       string
	Errors          []string // lookup, symbolic link, Lstat, read or close errors
}

// GuardSources are the guard's inputs; the real ones are ReadGuard's
// defaults, and tests and LC-5 replace them.
type GuardSources struct {
	Version      func() string
	LookPath     func(string) (string, error)
	EvalSymlinks func(string) (string, error)
	FS           modsnap.FS
	GOOS         string
}

// RealGuardSources are the real sources: runtime.Version, exec.LookPath
// (it only searches PATH; it starts nothing), filepath.EvalSymlinks and the
// real file system.
func RealGuardSources() GuardSources {
	return GuardSources{Version: runtime.Version, LookPath: exec.LookPath, EvalSymlinks: filepath.EvalSymlinks, FS: modsnap.OS{}, GOOS: runtime.GOOS}
}

// ReadGuard reads the guard values. It never runs the candidate.
func ReadGuard(src GuardSources) GuardValues {
	v := GuardValues{AnalyzerVersion: src.Version(), ExeName: "go"}
	if src.GOOS == "windows" {
		v.ExeName = "go.exe"
	}
	found, err := src.LookPath("go")
	if err != nil {
		v.Errors = append(v.Errors, "look up go: "+err.Error())
		return v
	}
	abs, err := filepath.Abs(found)
	if err == nil {
		abs, err = src.EvalSymlinks(abs)
	}
	if err != nil {
		v.Errors = append(v.Errors, "resolve candidate: "+err.Error())
		return v
	}
	v.Candidate = abs
	bin := filepath.Dir(abs)
	v.Root = filepath.Dir(bin)
	if fi, err := src.FS.Lstat(abs); err != nil {
		v.Errors = append(v.Errors, "lstat candidate: "+err.Error())
	} else {
		v.ExeRegular = fi.Mode().IsRegular()
	}
	if fi, err := src.FS.Lstat(filepath.Join(v.Root, "src", "cmd", "go")); err == nil && fi.IsDir() {
		v.SrcCmdGo = true
	}
	if b, err := readClosed(src.FS, filepath.Join(v.Root, "VERSION")); err != nil {
		v.Errors = append(v.Errors, "read VERSION: "+err.Error())
	} else {
		line, _, _ := bufio.NewReader(bytes.NewReader(b)).ReadLine()
		v.VersionLine = string(line)
	}
	if v.ExeRegular {
		if b, err := readClosed(src.FS, abs); err != nil {
			v.Errors = append(v.Errors, "read candidate: "+err.Error())
		} else {
			sum := sha256.Sum256(b)
			v.ExeSHA256 = hex.EncodeToString(sum[:])
		}
	}
	return v
}

// readClosed reads a whole file and returns the read or the real close
// error.
func readClosed(fsys modsnap.FS, path string) ([]byte, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return nil, err
	}
	b, rerr := io.ReadAll(f)
	cerr := f.Close()
	if rerr != nil {
		return nil, rerr
	}
	if cerr != nil {
		return nil, fmt.Errorf("close: %w", cerr)
	}
	return b, nil
}

// EvaluatePre is the A1-a guard: the analyser and <root>/VERSION are both
// go1.27.3 (not only equal to each other), the candidate is <root>/bin/go
// as a regular file with <root>/src/cmd/go present, and nothing failed to
// read. Every failure is J-1; nothing tries another candidate.
func EvaluatePre(v GuardValues) []judge.Failure {
	var fs []judge.Failure
	add := func(item, kind, detail string) {
		fs = append(fs, judge.Real(judge.J1, item, "A1-a guard", kind, detail))
	}
	if v.AnalyzerVersion != spec.ToolVersion {
		add("analyzer", "version", fmt.Sprintf("runtime.Version %q, want %q", v.AnalyzerVersion, spec.ToolVersion))
	}
	for _, e := range v.Errors {
		add("candidate", "read error", e)
	}
	if v.Candidate != "" {
		if filepath.Base(v.Candidate) != v.ExeName || filepath.Base(filepath.Dir(v.Candidate)) != "bin" || !v.ExeRegular {
			add("candidate", "layout", fmt.Sprintf("%s is not <root>/bin/%s as a regular file", v.Candidate, v.ExeName))
		}
		if !v.SrcCmdGo {
			add("candidate", "layout", "<root>/src/cmd/go is missing")
		}
		if v.VersionLine != spec.ToolVersion {
			add("VERSION", "version", fmt.Sprintf("first line %q, want %q", v.VersionLine, spec.ToolVersion))
		}
	} else if len(v.Errors) == 0 {
		add("candidate", "layout", "no candidate")
	}
	return fs
}

// EvaluatePost is the A1-c guard on the first call's decoded output:
// GOVERSION is go1.27.3 and GOROOT is <root>.
func EvaluatePost(v GuardValues, env map[string]string) []judge.Failure {
	var fs []judge.Failure
	if env["GOVERSION"] != spec.ToolVersion {
		fs = append(fs, judge.Real(judge.J1, "GOVERSION", "A1-c guard", "version", fmt.Sprintf("%q, want %q", env["GOVERSION"], spec.ToolVersion)))
	}
	if env["GOROOT"] == "" || env["GOROOT"] != v.Root {
		fs = append(fs, judge.Real(judge.J1, "GOROOT", "A1-c guard", "root", fmt.Sprintf("%q, want %q", env["GOROOT"], v.Root)))
	}
	return fs
}
