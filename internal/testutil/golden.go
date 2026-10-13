package testutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// UpdateGoldenEnv rewrites golden files instead of comparing when set to 1.
const UpdateGoldenEnv = "SPECMETRY_UPDATE_GOLDEN"

// Golden compares got with testdata/<name>.golden (line endings normalized).
func Golden(t testing.TB, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	got = normalizeNewlines(got)
	if os.Getenv(UpdateGoldenEnv) == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file (run with %s=1 to create it): %v", UpdateGoldenEnv, err)
	}
	if want = normalizeNewlines(want); !bytes.Equal(got, want) {
		t.Fatalf("output does not match %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}

func normalizeNewlines(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}
