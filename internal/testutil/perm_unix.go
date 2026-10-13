//go:build !windows

package testutil

import (
	"os"
	"testing"
)

// LoosenPermissions makes path readable by other users, so tests can check
// that doctor reports it.
func LoosenPermissions(t testing.TB, path string) {
	t.Helper()
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
