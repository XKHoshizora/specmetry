package toolctl

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/pseudo"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// These unit tests run the protocol on the pseudo-tool (a re-execution of
// this test binary, see main_test.go). They are not formal LC evidence;
// LC-9 (internal/controls) is.

const guard = 20 * time.Second

func after(d time.Duration) <-chan struct{} {
	ch := make(chan struct{})
	time.AfterFunc(d, func() { close(ch) })
	return ch
}

func pseudoSpec(t *testing.T, args ...string) collect.CallSpec {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cfg := spec.Config{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
	return collect.CallSpec{Config: cfg, Kind: spec.CallContext, Exe: exe, Args: args, Dir: t.TempDir(), Env: spec.ToolEnv(cfg)}
}

func modeSeams(mode string, stdin *os.File) *Seams {
	return &Seams{Prepare: func(_ int, cmd *exec.Cmd) {
		cmd.Env = append(cmd.Env, pseudo.EnvMode+"="+mode)
		if stdin != nil {
			cmd.Stdin = stdin
		}
	}}
}

// TestConcurrentCalls runs eight calls on one registry at once: each is
// registered with its own ID, each meets item 11, and the registry ends
// terminal with no problem. Under -race it also checks the locking.
func TestConcurrentCalls(t *testing.T) {
	now := time.Now()
	reg := New(Options{Clock: chain.RealClock{}, W: now.Add(time.Hour), D: now.Add(time.Hour + 10*time.Second), Seams: modeSeams(pseudo.EchoArgs, nil)})
	const n = 8
	var wg sync.WaitGroup
	res := make([]collect.CallResult, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res[i] = reg.Invoke(pseudoSpec(t, "call", fmt.Sprint(i), spec.T2))
		}()
	}
	wg.Wait()
	for i, r := range res {
		if !r.Succeeded(reg.W()) {
			t.Errorf("call %d: %v", i, r.Problems(reg.W()))
		}
		if want := pseudo.EchoOutput([]string{"call", fmt.Sprint(i), spec.T2}); string(r.Stdout) != want {
			t.Errorf("call %d stdout %q, want %q", i, r.Stdout, want)
		}
	}
	if done, _ := reg.Terminal(); !done {
		t.Error("registry not terminal")
	}
	if f, p := reg.Problems(); len(f) > 0 || len(p) > 0 {
		t.Errorf("problems %v pending %v", f, p)
	}
	ids := map[int]bool{}
	for _, s := range reg.Snapshots() {
		ids[s.ID] = true
		if !s.Success || !s.Terminal {
			t.Errorf("call %d: success %v terminal %v", s.ID, s.Success, s.Terminal)
		}
	}
	if len(ids) != n {
		t.Errorf("%d distinct IDs, want %d", len(ids), n)
	}
}

// TestExpiredCallStaysHeld expires a call whose pseudo-tool blocks on its
// stdin. Until it is released the registry is not terminal on darwin (no
// termination); on Linux and Windows the one request ends the process. In
// every case the expiry is a failure that no later result clears, and the
// record stays in the registry.
func TestExpiredCallStaysHeld(t *testing.T) {
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	released := false
	t.Cleanup(func() {
		if !released {
			inW.Close()
		}
		inR.Close()
	})
	clk := chain.NewFakeClock(time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC))
	reg := New(Options{Clock: clk, W: clk.Now().Add(100 * time.Second), D: clk.Now().Add(110 * time.Second), Seams: modeSeams(pseudo.Block, inR)})
	got := make(chan collect.CallResult, 1)
	go func() { got <- reg.Invoke(pseudoSpec(t)) }()
	give := after(guard)
	if !reg.WaitFor(1, give, func(s Snapshot) bool {
		return s.State == StStarted && (s.Mode != ModePidfd || s.Pidfd.State == PidfdOpen)
	}) {
		t.Fatal("not started")
	}
	clk.Set(reg.W())
	var r collect.CallResult
	select {
	case r = <-got:
	case <-time.After(guard):
		t.Fatal("Invoke did not return at W")
	}
	if r.Succeeded(reg.W()) || !strings.Contains(strings.Join(r.Failures, ";"), "not completed before W") {
		t.Fatalf("result %v", r.Failures)
	}
	if reg.Mode() == ModeNone {
		if done, _ := reg.Terminal(); done {
			t.Fatal("terminal before the release on darwin")
		}
		if _, p := reg.Problems(); len(p) != 1 {
			t.Fatalf("pending %v", p)
		}
	}
	released = true
	if err := inW.Close(); err != nil {
		t.Fatal(err)
	}
	if !reg.WaitTerminal(1, after(guard)) {
		t.Fatal("not terminal after the release")
	}
	f, p := reg.Problems()
	if len(p) != 0 || len(f) != 1 || !strings.Contains(f[0], "not completed before W") {
		t.Fatalf("failures %v pending %v", f, p)
	}
	s, _ := reg.Snapshot(1)
	if s.Success || !s.Expired {
		t.Fatalf("snapshot %+v", s)
	}
	switch reg.Mode() {
	case ModePidfd, ModeKill:
		if s.Req.State != ReqDone || s.Req.Sends != 1 {
			t.Errorf("request %+v", s.Req)
		}
	default:
		if s.Req.State != ReqNone || len(s.Unknown) != 1 {
			t.Errorf("request %+v unknown %v", s.Req, s.Unknown)
		}
	}
}

// TestLinesOfAStartFailure checks the canonical lines of a start failure
// end to end (absolute path that does not exist; C, phase start).
func TestLinesOfAStartFailure(t *testing.T) {
	now := time.Now()
	reg := New(Options{Clock: chain.RealClock{}, W: now.Add(time.Hour), D: now.Add(time.Hour + 10*time.Second)})
	cs := pseudoSpec(t)
	cs.Exe = cs.Dir + string(os.PathSeparator) + "absent" + string(os.PathSeparator) + "tool.exe"
	r := reg.Invoke(cs)
	if r.Started || r.Succeeded(reg.W()) {
		t.Fatalf("%+v", r)
	}
	s, _ := reg.Snapshot(1)
	got := strings.Join(Lines(s), "\n")
	for _, want := range []string{"class C phase start", "state start-failed", "start error", "wait not-called", "terminal yes", "success no"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in\n%s", want, got)
		}
	}
}
