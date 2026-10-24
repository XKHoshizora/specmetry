package wp25

import (
	"errors"
	"os"
	"testing"
	"time"
)

// TestToolState: the order of a helper's tool state (Review 086 B-N4-R)
// on a fresh state per row, with operations that only count. The real
// helper's paths are the gate regressions (report.limit-race-*,
// report.ok-fails-epipe-reported, report.epipe-fails-then-ok,
// report.limit-after-done); these rows add the orders on their own.
func TestToolState(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	fail := errors.New("injected")
	ok := func() error { return nil }
	bad := func() error { return fail }
	t.Run("healthy", func(t *testing.T) {
		var ts toolState
		published := 0
		if err := ts.report(ok); err != nil {
			t.Fatal(err)
		}
		if err := ts.complete(func() error { published++; return nil }); err != nil || published != 1 || !ts.published {
			t.Fatalf("complete: %v, published %d", err, published)
		}
		if ts.limitEntered() {
			t.Error("the limit won over a published completion")
		}
		ran := false
		if err := ts.report(func() error { ran = true; return nil }); !errors.Is(err, errAfterCompletion) || ran {
			t.Errorf("a report after the completion: %v, ran %v", err, ran)
		}
	})
	t.Run("ignored failure, then a success", func(t *testing.T) {
		var ts toolState
		_ = ts.report(bad)
		_ = ts.report(ok)
		ts.failed(nil)
		published := 0
		if err := ts.complete(func() error { published++; return nil }); !errors.Is(err, fail) || published != 0 {
			t.Errorf("complete after a kept failure: %v, published %d", err, published)
		}
	})
	t.Run("limit entered first", func(t *testing.T) {
		var ts toolState
		if !ts.limitEntered() {
			t.Fatal("the limit lost before any completion")
		}
		_ = ts.report(ok)
		published := 0
		if err := ts.complete(func() error { published++; return nil }); err == nil || published != 0 || ts.published {
			t.Errorf("complete after the limit: %v, published %d", err, published)
		}
	})
	t.Run("publication fails", func(t *testing.T) {
		var ts toolState
		if err := ts.complete(bad); !errors.Is(err, fail) || ts.published {
			t.Fatalf("complete: %v", err)
		}
		if err := ts.complete(ok); err == nil || ts.published {
			t.Errorf("a second completion: %v", err)
		}
		if !ts.limitEntered() {
			t.Error("the limit lost to a completion that was not published")
		}
	})
	t.Run("limit enters during the publication", func(t *testing.T) {
		// The callback enters while the completion publishes: it waits for
		// the final state, and loses to the published completion.
		var ts toolState
		inside, release := make(chan struct{}), make(chan struct{})
		done := make(chan error, 1)
		go func() {
			done <- ts.complete(func() error { close(inside); <-release; return nil })
		}()
		select {
		case <-inside:
		case <-time.After(5 * time.Second):
			t.Fatal("the publication never started")
		}
		won := make(chan bool, 1)
		go func() { won <- ts.limitEntered() }()
		select {
		case w := <-won:
			close(release)
			t.Fatalf("the limit decided (%v) while the completion was publishing", w)
		case <-time.After(100 * time.Millisecond):
		}
		close(release)
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("complete: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the completion never returned")
		}
		select {
		case w := <-won:
			if w {
				t.Error("the limit won over the published completion")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the limit never decided")
		}
	})
}
