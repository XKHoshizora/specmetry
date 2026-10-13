//go:build !windows

package testutil

import (
	"os"
	"testing"
)

// LoosenPermissions makes path readable by other users (directories 0755,
// files 0644), so tests can check that it is detected and refused.
func LoosenPermissions(t testing.TB, path string) {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mode := os.FileMode(0o644)
	if fi.IsDir() {
		mode = 0o755
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}
