// Package testdir provides a directory for tests to put data directories
// in. It has no dependencies inside the module, so every package's tests can
// use it.
package testdir

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// New returns a new empty directory that is removed when the test ends.
//
// Specmetry refuses data directories on a path that other users could
// change (datadir.Ensure). On Windows the temporary directory may grant
// other accounts write access (for example the sandbox accounts of some
// coding agents), so tests there use a directory under the user's local
// application data instead, which only the user, SYSTEM and Administrators
// can change. Elsewhere this is t.TempDir().
func New(t testing.TB) string {
	t.Helper()
	if runtime.GOOS != "windows" {
		return t.TempDir()
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(cache, "specmetry-test")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	name := strings.NewReplacer("/", "_", `\`, "_", ":", "_").Replace(t.Name())
	if len(name) > 40 {
		name = name[:40]
	}
	dir, err := os.MkdirTemp(base, name+"-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}
