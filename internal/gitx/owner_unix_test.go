//go:build !windows

package gitx

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// Reject a non-regular owner before opening it: a read-only FIFO open could
// otherwise wait for a writer instead of returning the required environment error.
func TestOwnerFIFORejectedBeforeOpen(t *testing.T) {
	s, home := scratchIn(t)
	d, err := newPrivateDir(s)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(d.close)
	if err := d.write("config", []byte("preserve")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(d.join(ownerName), home.Join("saved-owner")); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(d.join(ownerName), 0o600); err != nil {
		t.Fatal(err)
	}
	before := testutil.SettledSnapshot(t, home.Path)
	if err := d.checkContents(); err == nil {
		t.Error("non-regular owner passed preflight")
	}
	if err := d.remove(); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("cleanup=%v", err)
	}
	if _, err := newScratch(t, home); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("leftover=%v", err)
	}
	testutil.AssertTreeUnchanged(t, home.Path, before)
}
