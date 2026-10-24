//go:build windows

package wp25

// Direct checks of the Windows stop's own calls (Review 082 B1-Rb), on a
// real os.Pipe whose read is held in its window as in TestStopRead: a
// cancellation that fails for a reason other than "nothing to cancel", and
// a close of the stop's own handle that fails, must reach the stop's
// result. The faults replace the calls; the test's own pipe and handles
// are the only ones touched, and a handle the fault left open is closed
// here.

import (
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestStopAPIErrors(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	t.Run("nothing-to-cancel", func(t *testing.T) {
		// The control: the real CancelIoEx, the read kept in its window until
		// the stop has met "nothing to cancel" (ERROR_NOT_FOUND) at least
		// once. That is the race the stop expects, not a failure.
		var notFound atomic.Int64
		cancelOwn = func(h windows.Handle, o *windows.Overlapped) error {
			err := windows.CancelIoEx(h, o)
			if errors.Is(err, windows.ERROR_NOT_FOUND) {
				notFound.Add(1)
			}
			return err
		}
		t.Cleanup(func() { cancelOwn = windows.CancelIoEx })
		g := newStopRig(t, true)
		select {
		case <-g.g.entered:
		case <-time.After(5 * time.Second):
			t.Fatal("the copier never took its read reference within 5 s")
		}
		g.start()
		deadline := time.Now().Add(5 * time.Second)
		for notFound.Load() == 0 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if notFound.Load() == 0 {
			t.Fatal("the stop never met nothing to cancel within 5 s")
		}
		g.g.open()
		r, ok := g.result(5 * time.Second)
		if !ok {
			t.Fatal("runS6On did not return within 5 s after the read left its window")
		}
		if r.stop["out"] != "done" || r.out["out"] != o2 {
			t.Errorf("stop %q, outcome %s; want done, O2", r.stop["out"], r.out["out"])
		}
		if err := r.stopErr["out"]; err != nil {
			t.Errorf("the stop failed after meeting nothing to cancel %d times: %v", notFound.Load(), err)
		}
	})
	t.Run("cancel-fails", func(t *testing.T) {
		var hits atomic.Int64
		cancelOwn = func(windows.Handle, *windows.Overlapped) error {
			hits.Add(1)
			return windows.ERROR_ACCESS_DENIED
		}
		t.Cleanup(func() { cancelOwn = windows.CancelIoEx }) // after the rig's own cleanup
		g := newStopRig(t, true)
		select {
		case <-g.g.entered:
		case <-time.After(5 * time.Second):
			t.Fatal("the copier never took its read reference within 5 s")
		}
		g.start()
		if d, ok := awaitAsked(g.clk); !ok || d != tJoin {
			t.Fatalf("phase J did not start (wake %v, asked %v)", d, ok)
		}
		deadline := time.Now().Add(5 * time.Second)
		for hits.Load() == 0 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if hits.Load() == 0 {
			t.Fatal("the stop never tried to cancel within 5 s")
		}
		// Nothing cancels the read now: the test's own write lets it return.
		g.g.open()
		if _, err := g.w.Write([]byte("x")); err != nil {
			t.Fatalf("writing to the test's own pipe: %v", err)
		}
		r, ok := g.result(5 * time.Second)
		if !ok {
			t.Fatal("runS6On did not return within 5 s after the read got its byte")
		}
		if err := r.stopErr["out"]; !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			t.Errorf("the stop's result is %v after its cancellations failed (%d times), want the failure", err, hits.Load())
		}
	})
	t.Run("own-handle-close-fails", func(t *testing.T) {
		var hits atomic.Int64
		var kept atomic.Uintptr
		closeOwn = func(h windows.Handle) error {
			hits.Add(1)
			kept.Store(uintptr(h)) // left open by the fault
			return windows.ERROR_ACCESS_DENIED
		}
		t.Cleanup(func() {
			closeOwn = windows.CloseHandle
			if h := kept.Load(); h != 0 {
				_ = windows.CloseHandle(windows.Handle(h))
			}
		})
		g := newStopRig(t, true)
		select {
		case <-g.g.entered:
		case <-time.After(5 * time.Second):
			t.Fatal("the copier never took its read reference within 5 s")
		}
		g.start()
		if d, ok := awaitAsked(g.clk); !ok || d != tJoin {
			t.Fatalf("phase J did not start (wake %v, asked %v)", d, ok)
		}
		g.g.open()
		r, ok := g.result(5 * time.Second)
		if !ok {
			t.Fatal("runS6On did not return within 5 s after the read left its window")
		}
		if hits.Load() == 0 {
			t.Fatal("the stop's own handle was never closed")
		}
		if err := r.stopErr["out"]; !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			t.Errorf("the stop's result is %v after its own handle could not be closed, want the failure", err)
		}
	})
}
