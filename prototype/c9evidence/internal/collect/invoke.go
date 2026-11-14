package collect

import (
	"fmt"
	"sync"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// CallSpec is one of the three calls of a configuration, as the tool sets
// it: the checked executable, the arguments after it, the working directory
// R, and only the variables the tool sets (plan §4.2, §6.1).
type CallSpec struct {
	Config spec.Config
	Kind   spec.CallKind
	Exe    string
	Args   []string
	Dir    string
	Env    []spec.EnvVar
}

// CallResult is what the control protocol published for one call
// (implementation plan §12.14 items 3 to 12): start, reap, exit code, both
// outputs with their EOF, every control failure (close errors, read errors,
// overflow, cleanup unknown, a stopped boundary) and the injected results
// kept apart.
type CallResult struct {
	Spec      CallSpec
	Started   bool
	StartErr  string
	Reaped    bool
	ExitCode  int
	Stdout    []byte
	Stderr    []byte
	StdoutEOF bool
	StderrEOF bool
	Failures  []string
	Injected  []string
	Start     time.Time
	End       time.Time
}

// Succeeded is §12.14 item 11: reaped with exit code 0, both outputs at
// EOF, no control failure and no injection, all published before W.
func (r CallResult) Succeeded(w time.Time) bool {
	return r.Started && r.Reaped && r.ExitCode == 0 && r.StdoutEOF && r.StderrEOF &&
		len(r.Failures) == 0 && len(r.Injected) == 0 && r.End.Before(w)
}

// Problems lists why a call did not succeed, for records and failures.
func (r CallResult) Problems(w time.Time) []string {
	var out []string
	if !r.Started {
		s := "not started"
		if r.StartErr != "" {
			s += ": " + r.StartErr
		}
		out = append(out, s)
	}
	if r.Started && !r.Reaped {
		out = append(out, "not reaped")
	}
	if r.Reaped && r.ExitCode != 0 {
		out = append(out, fmt.Sprintf("exit status %d", r.ExitCode))
	}
	if r.Started && (!r.StdoutEOF || !r.StderrEOF) {
		out = append(out, "output not at EOF")
	}
	out = append(out, r.Failures...)
	for _, s := range r.Injected {
		out = append(out, "injected: "+s)
	}
	if !r.End.IsZero() && !r.End.Before(w) {
		out = append(out, "published at or after W")
	}
	return out
}

// Invoker runs one call. The execution boundary of the tool: phase 1 has
// only Stopped and the in-process Fake.
type Invoker interface {
	Invoke(s CallSpec) CallResult
}

// StoppedMessage is the failure the Stopped invoker publishes.
const StoppedMessage = "real invocation stopped: E-10 (ii) is not confirmed; no process was started"

// Stopped is the real side of the boundary in phase 1. E-10 (ii) is not
// confirmed, so it starts nothing: no process, helper, script, subprocess
// or fallback. Every call it gets fails with StoppedMessage (J-1).
type Stopped struct{ Clock Clock }

// Invoke refuses the call.
func (s Stopped) Invoke(c CallSpec) CallResult {
	now := s.Clock.Now()
	return CallResult{Spec: c, Failures: []string{StoppedMessage}, Start: now, End: now}
}

// FakeResponse is what the in-process fake gives for one kind of call.
type FakeResponse struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Failures []string // control failures to publish
	// StartErr makes the call fail to start.
	StartErr string
	// Delay is added to the call's publication time (with a fake clock,
	// the fake advances it), to place the end after W in admission tests.
	Delay time.Duration
}

// Fake is the in-process fake caller (plan §5.1 "接缝"): it returns fixed
// bytes and counts the calls of each kind. It never starts a process.
type Fake struct {
	mu        sync.Mutex
	Clock     Clock
	Advance   func(time.Duration) // optional: advances a fake clock by Delay
	Responses map[spec.CallKind]FakeResponse
	calls     map[spec.CallKind]int
	specs     []CallSpec
}

// Invoke returns the fixed response of the call's kind.
func (f *Fake) Invoke(c CallSpec) CallResult {
	f.mu.Lock()
	if f.calls == nil {
		f.calls = map[spec.CallKind]int{}
	}
	f.calls[c.Kind]++
	f.specs = append(f.specs, c)
	resp, ok := f.Responses[c.Kind]
	f.mu.Unlock()
	start := f.Clock.Now()
	if resp.Delay > 0 && f.Advance != nil {
		f.Advance(resp.Delay)
	}
	end := f.Clock.Now()
	if !ok {
		return CallResult{Spec: c, Failures: []string{"fake: no response for " + c.Kind.String()}, Start: start, End: end}
	}
	if resp.StartErr != "" {
		return CallResult{Spec: c, StartErr: resp.StartErr, Start: start, End: end}
	}
	return CallResult{
		Spec: c, Started: true, Reaped: true, ExitCode: resp.ExitCode,
		Stdout: append([]byte(nil), resp.Stdout...), Stderr: append([]byte(nil), resp.Stderr...),
		StdoutEOF: true, StderrEOF: true, Failures: append([]string(nil), resp.Failures...),
		Start: start, End: end,
	}
}

// Calls returns how many calls of a kind the fake got.
func (f *Fake) Calls(k spec.CallKind) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[k]
}

// Specs returns the calls the fake got, in order.
func (f *Fake) Specs() []CallSpec {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]CallSpec(nil), f.specs...)
}
