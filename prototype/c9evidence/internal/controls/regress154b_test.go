package controls

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

// The holder H regressions of Review 154 (review-resolution §202: R3-a to
// R3-d). They need the new holder structure and never ran on the old
// implementation. Most drive the holder directly, with the test's own
// target pipe standing in for the tool's stdout write end; R3-e runs the
// whole output-not-closed variant.

var errSim154 = errors.New("r154: a holder operation error the seam stands in for")

type hlatch struct {
	entered, open chan struct{}
	eo, oo        sync.Once
}

func newHLatch() *hlatch { return &hlatch{entered: make(chan struct{}), open: make(chan struct{})} }

func (l *hlatch) wait() {
	l.eo.Do(func() { close(l.entered) })
	<-l.open
}

func (l *hlatch) release() { l.oo.Do(func() { close(l.open) }) }

func (l *hlatch) waitEntered(t *testing.T, what string) {
	t.Helper()
	select {
	case <-l.entered:
	case <-time.After(lc9Guard):
		t.Fatalf("%s: not entered within the guard", what)
	}
}

// holderHarness is an LC-9 run with only H, and the test's own target
// pipe whose write end H inherits as its stdout.
func holderHarness(t *testing.T, env *Env, seams func(h *holder)) (*lc9Run, *os.File) {
	t.Helper()
	c := &lc9Run{e: env, k: lc9Case{name: "r154-holder"}}
	c.h = newHolder(c)
	if seams != nil {
		seams(c.h)
	}
	tr, tw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		tw.Close()
		tr.Close()
	})
	return c, tw
}

func saveDecision(c *lc9Run) {
	c.mu.Lock()
	c.saved = true
	c.mu.Unlock()
}

func wantHolder(t *testing.T, c *lc9Run, lines, faults []string) {
	t.Helper()
	got := strings.Join(c.h.lines(), "\n")
	if want := strings.Join(lines, "\n"); got != want {
		t.Errorf("holder lines\n got  %q\n want %q", c.h.lines(), lines)
	}
	var fs []string
	for _, f := range c.h.faultSet() {
		fs = append(fs, f.Line())
	}
	if strings.Join(fs, ";") != strings.Join(faults, ";") {
		t.Errorf("holder faults %q, want %q", fs, faults)
	}
}

func opsWith(m *MemOuter, sub string) []string {
	ops, _, _ := m.State()
	var out []string
	for _, o := range ops {
		if strings.Contains(o, sub) {
			out = append(out, o)
		}
	}
	return out
}

// TestR154HolderHalfBuilt is R3-a: the release pipe fails after the ready
// pipe was built; the ready pipe's two ends are closed once each; the two
// pipes were two separate admitted preparations.
func TestR154HolderHalfBuilt(t *testing.T) {
	env, outer := newEnv(t)
	c, tw := holderHarness(t, env, func(h *holder) {
		h.pipeOp = func(n int) (*os.File, *os.File, error) {
			if n == 1 {
				return nil, nil, errSim154
			}
			return os.Pipe()
		}
	})
	c.startHolder(tw)
	if c.prepErr == nil || !c.h.waitTerminal(lc9Guard) {
		t.Fatalf("prep error %v terminal %v", c.prepErr, c.h.terminal())
	}
	wantHolder(t, c, []string{"holder start failed", "holder ready none", "holder wait none", "holder release none after-saved-decision no",
		"holder ready-r closed ok release confirmed", "holder ready-w closed ok release confirmed", "holder rel-r none", "holder rel-w none", "holder tasks 0"},
		[]string{"fault holder release pipe real other"})
	if p := opsWith(outer, "holder"); len(p) != 4 || !strings.HasSuffix(p[0], "holder ready pipe") || !strings.HasSuffix(p[1], "holder release pipe") {
		t.Errorf("outer operations %q", p)
	}
}

// refuseOuter refuses the preparations whose names end with refuse.
type refuseOuter struct {
	*MemOuter
	refuse string
}

func (o refuseOuter) Prepare(name string) (func(error), bool) {
	if strings.HasSuffix(name, o.refuse) {
		return nil, false
	}
	return o.MemOuter.Prepare(name)
}

// TestR154HolderSecondPipeRefused is R3-b: the outer layer refuses the
// release pipe's admission; it is never built and the ready pipe's ends
// are closed.
func TestR154HolderSecondPipeRefused(t *testing.T) {
	env, outer := newEnv(t)
	env.Outer = refuseOuter{MemOuter: outer, refuse: "holder release pipe"}
	c, tw := holderHarness(t, env, nil)
	c.startHolder(tw)
	if !errors.Is(c.prepErr, ErrNotAdmitted) || !c.h.waitTerminal(lc9Guard) {
		t.Fatalf("prep error %v terminal %v", c.prepErr, c.h.terminal())
	}
	wantHolder(t, c, []string{"holder start failed", "holder ready none", "holder wait none", "holder release none after-saved-decision no",
		"holder ready-r closed ok release confirmed", "holder ready-w closed ok release confirmed", "holder rel-r none", "holder rel-w none", "holder tasks 0"},
		[]string{"fault holder release pipe admission not-admitted"})
}

// TestR154HolderLateStart is R3-c: H's Start returned while the outer
// cleanup stopped; the cleanup does not release, the start path makes the
// one release request after it publishes; H exits, its Wait returns, every
// end is closed and H is terminal.
func TestR154HolderLateStart(t *testing.T) {
	env, _ := newEnv(t)
	l := newHLatch()
	t.Cleanup(l.release)
	c, tw := holderHarness(t, env, func(h *holder) { h.afterStart = l.wait })
	done := make(chan struct{})
	go func() {
		c.startHolder(tw)
		close(done)
	}()
	l.waitEntered(t, "after H's Start")
	saveDecision(c)
	c.h.stop()
	c.h.mu.Lock()
	start, rel := c.h.start, c.h.release
	c.h.mu.Unlock()
	if start != hStarting || rel != "none" {
		t.Errorf("at the stop: start %s release %s", start, rel)
	}
	l.release()
	select {
	case <-done:
	case <-time.After(lc9Guard):
		t.Fatal("startHolder did not return")
	}
	if !c.h.waitTerminal(lc9Guard) {
		t.Fatalf("not terminal: %q", c.h.lines())
	}
	wantHolder(t, c, []string{"holder start started", "holder ready ok", "holder wait done exited 0 reap confirmed", "holder release by-start after-saved-decision yes",
		"holder ready-r closed ok release confirmed", "holder ready-w closed ok release confirmed", "holder rel-r closed ok release confirmed", "holder rel-w closed ok release confirmed", "holder tasks 0"}, nil)
	if c.prepErr != nil {
		t.Errorf("prep error %v", c.prepErr)
	}
}

// TestR154HolderReadyCloseNotDone is R3-d: the ready line was read but the
// ready read end's close has not returned; H is not terminal (the instance
// is pending) until it returns.
func TestR154HolderReadyCloseNotDone(t *testing.T) {
	env, _ := newEnv(t)
	l := newHLatch()
	t.Cleanup(l.release)
	c, tw := holderHarness(t, env, func(h *holder) {
		h.closeOp = func(name string, f *os.File) error {
			if name == "ready-r" {
				l.wait()
			}
			return f.Close()
		}
	})
	c.startHolder(tw)
	if c.prepErr != nil {
		t.Fatal(c.prepErr)
	}
	l.waitEntered(t, "the ready read end's close")
	saveDecision(c)
	c.h.stop()
	c.h.mu.Lock()
	ready, readyEnd := c.h.ready, c.h.ends["ready-r"].state
	c.h.mu.Unlock()
	if ready != "ok" || readyEnd != endClosing {
		t.Errorf("ready %s, ready-r %s", ready, readyEnd)
	}
	if c.h.waitTerminal(300 * time.Millisecond) {
		t.Errorf("terminal while the ready close is in flight: %q", c.h.lines())
	}
	l.release()
	if !c.h.waitTerminal(lc9Guard) {
		t.Fatalf("not terminal after the release: %q", c.h.lines())
	}
	wantHolder(t, c, []string{"holder start started", "holder ready ok", "holder wait done exited 0 reap confirmed", "holder release by-cleanup after-saved-decision yes",
		"holder ready-r closed ok release confirmed", "holder ready-w closed ok release confirmed", "holder rel-r closed ok release confirmed", "holder rel-w closed ok release confirmed", "holder tasks 0"}, nil)
}

// TestR154HolderErrorJudged is R3-e: in the whole output-not-closed
// variant, the ready read end's real close returns an error after the
// ready line was sent; H's fault is in the judged lines, so the variant is
// not a hit.
func TestR154HolderErrorJudged(t *testing.T) {
	k := lc9CaseNamed(t, "output-not-closed")
	k.holderSeams = func(h *holder) {
		h.closeOp = func(name string, f *os.File) error {
			err := f.Close()
			if name == "ready-r" && err == nil {
				return errSim154
			}
			return err
		}
	}
	v := k.variant(runtime.GOOS)
	env, _ := newEnv(t)
	o := v.Run(env)
	j := Judge(v, runtime.GOOS, o, true)
	if j.Judgement == verdict.Hit || !strings.Contains(strings.Join(j.Reasons, " "), "fault holder close ready-r real other") {
		t.Errorf("%s %v", j.Judgement, j.Reasons)
	}
}
