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

// RunInPTY runs name with args attached to a pseudo-terminal (ConPTY on
// Windows) and returns everything it printed. The process sees a terminal on
// stdin and stdout, which is how human_cli is detected (architecture.md §5).
func RunInPTY(t testing.TB, timeout time.Duration, env []string, name string, args ...string) PTYResult {
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
		for {
			n, err := p.Read(chunk)
			if n > 0 {
				mu.Lock()
				buf.Write(chunk[:n])
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
		t.Fatalf("%s did not exit within %s", name, timeout)
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
