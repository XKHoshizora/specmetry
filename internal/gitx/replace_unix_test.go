//go:build !windows

package gitx

import (
	"os"
	"testing"
)

// makeReplaceable makes dir writable by everyone and not sticky: anyone
// can rename or remove what is in it.
func makeReplaceable(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
}

// link makes path a symbolic link to target.
func link(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}
