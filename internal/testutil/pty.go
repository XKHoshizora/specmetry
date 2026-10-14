package testutil

import (
	"bytes"
	"sync"
	"testing"
	"time"

	gopty "github.com/aymanbagabas/go-pty"
)

// PTYResult is the outcome of RunInPTY.
type PTYResult struct {
	Output  string
	WaitErr error
}

// PTYStep is one answer typed into the terminal: once the output contains
// Expect (after the previous step's match), Before runs if set (to change
// something while the prompt is shown), then Send is written as
// keystrokes. End a line with "\r", as the Enter key does. ConPTY drops
// trailing spaces from what it renders, so Expect should not end with a
// space.
type PTYStep struct {
	Expect string
	Before func()
	Send   string
}

// RunInPTY runs name with args attached to a pseudo-terminal (ConPTY on
// Windows) and returns everything it printed. The process sees a terminal on
// stdin and stdout, which is how human_cli is detected (architecture.md §5).
func RunInPTY(t testing.TB, timeout time.Duration, env []string, name string, args ...string) PTYResult {
	t.Helper()
	return RunInPTYSession(t, timeout, env, nil, name, args...)
}

// RunInPTYSession is RunInPTY that also types the steps' answers when
// their prompts appear, as a person at the terminal would.
func RunInPTYSession(t testing.TB, timeout time.Duration, env []string, steps []PTYStep, name string, args ...string) PTYResult {
	t.Helper()
	p, err := gopty.New()
	if err != nil {
		t.Fatalf("create pty: %v", err)
	}
	cmd := p.Command(name, args...)
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		_ = p.Close()
		t.Fatalf("start %s in pty: %v", name, err)
	}

	var (
		mu  sync.Mutex
		buf bytes.Buffer
	)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		chunk := make([]byte, 4096)
		next, from := 0, 0 // the next step, and where its search starts
		for {
			n, err := p.Read(chunk)
			if n > 0 {
				mu.Lock()
				buf.Write(chunk[:n])
				for next < len(steps) {
					i := bytes.Index(buf.Bytes()[from:], []byte(steps[next].Expect))
					if i < 0 {
						break
					}
					from += i + len(steps[next].Expect)
					if steps[next].Before != nil {
						steps[next].Before()
					}
					_, _ = p.Write([]byte(steps[next].Send))
					next++
				}
				mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	var waitErr error
	select {
	case waitErr = <-waitDone:
	case <-time.After(timeout):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = p.Close()
		mu.Lock()
		out := buf.String()
		mu.Unlock()
		t.Fatalf("%s did not exit within %s; output so far: %q", name, timeout, out)
	}
	// Unix: reads end with EIO after the child exits. Windows: the ConPTY
	// output pipe stays open until the pseudo console is closed.
	select {
	case <-readDone:
	case <-time.After(500 * time.Millisecond):
	}
	_ = p.Close()
	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
	}
	mu.Lock()
	defer mu.Unlock()
	return PTYResult{Output: buf.String(), WaitErr: waitErr}
}
