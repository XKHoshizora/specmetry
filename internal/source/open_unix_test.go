//go:build !windows

package source_test

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/source"
)

// within runs fn and fails the test if it has not returned after a few
// seconds: opening a FIFO without a writer would block for ever.
func within(t *testing.T, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { fn(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not return: it blocked", what)
	}
}

// codex-review-020 B5: a FIFO (with no writer), a character device or a
// directory where a source should be is not a regular file. Opening it
// never blocks, and hashing it reports the source missing.
func TestSpecialFilesDoNotBlock(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "brief.md")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{fifo, "/dev/null", dir} {
		within(t, "hashing "+p, func() {
			if _, _, err := source.HashFile(ctx, p); !errors.Is(err, source.ErrMissing) {
				t.Errorf("HashFile(%s) = %v, want ErrMissing", p, err)
			}
		})
		within(t, "opening "+p, func() {
			f, err := source.OpenRegular(p)
			if err == nil {
				_ = f.Close()
				t.Errorf("OpenRegular(%s) opened it as a regular file", p)
			}
		})
	}
	// A regular file still opens, and reads blocking as usual.
	regular := filepath.Join(dir, "regular.md")
	if err := os.WriteFile(regular, []byte("订单"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := source.OpenRegular(regular)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	b := make([]byte, 16)
	if n, err := f.Read(b); err != nil || string(b[:n]) != "订单" {
		t.Fatalf("read %q, %v", b[:n], err)
	}
}
