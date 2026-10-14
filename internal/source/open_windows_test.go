//go:build windows

package source_test

import (
	"errors"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/source"
)

// codex-review-020 B5 on Windows: a device (NUL) or a directory where a
// source should be is not a regular file; it is refused from its open
// handle before anything is read, and hashing reports the source missing.
func TestSpecialFilesAreRefused(t *testing.T) {
	for _, p := range []string{"NUL", `\\.\NUL`, t.TempDir()} {
		if f, err := source.OpenRegular(p); err == nil {
			_ = f.Close()
			t.Errorf("OpenRegular(%s) opened it as a regular file", p)
		}
		if _, _, err := source.HashFile(ctx, p); !errors.Is(err, source.ErrMissing) {
			t.Errorf("HashFile(%s) = %v, want ErrMissing", p, err)
		}
	}
}
