//go:build !windows

package proposal_test

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/proposal"
)

// codex-review-020 B5: a source replaced by a FIFO (with no writer) is
// source_missing, at once and whatever the context: opening it never
// blocks, so a deadline is not even needed to get out.
func TestCheckExcerptsOnAFIFO(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "brief.md")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	short, cancel := context.WithTimeout(ctx, 25*time.Millisecond)
	defer cancel()
	for _, c := range []context.Context{ctx, short} {
		done := make(chan struct{})
		var got []proposal.Verification
		var sum string
		var err error
		go func() {
			sum, got, err = proposal.CheckExcerpts(c, fifo, []string{"订单"})
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("checking an excerpt against a FIFO blocked")
		}
		if err == nil && (sum != "" || got[0] != proposal.SourceMissing) {
			t.Errorf("a FIFO: %q %v", sum, got)
		}
	}
}

// A source swapped between a regular file and a FIFO while it is checked
// is never read as anything but itself: every check returns at once, with
// verified (the file) or source_missing (the FIFO) and nothing else.
func TestCheckExcerptsWhileTheSourceIsSwapped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "brief.md")
	regular := filepath.Join(dir, "regular.md")
	fifo := filepath.Join(dir, "fifo.md")
	if err := os.WriteFile(regular, []byte("店员可以查看自己创建的订单"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	swapped := make(chan struct{})
	go func() {
		defer close(swapped)
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			// Hard links put either file at path; rename swaps atomically.
			from := regular
			if i%2 == 1 {
				from = fifo
			}
			tmp := filepath.Join(dir, "next")
			_ = os.Remove(tmp)
			if err := os.Link(from, tmp); err == nil {
				_ = os.Rename(tmp, path)
			}
		}
	}()
	deadline := time.Now().Add(2 * time.Second)
	seen := map[proposal.Verification]int{}
	for time.Now().Before(deadline) {
		done := make(chan struct{})
		var got []proposal.Verification
		var err error
		go func() {
			_, got, err = proposal.CheckExcerpts(ctx, path, []string{"查看自己创建的订单"})
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			close(stop)
			t.Fatal("a check blocked on the swapped source")
		}
		if err != nil {
			t.Fatal(err)
		}
		seen[got[0]]++
		if got[0] != proposal.Verified && got[0] != proposal.SourceMissing {
			t.Fatalf("result %s", got[0])
		}
	}
	close(stop)
	<-swapped
	t.Logf("results while swapping: %v", seen)
}
