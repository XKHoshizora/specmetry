package testutil

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// NewCanary returns a unique marker string to plant in data that must never
// be persisted (privacy contract, architecture.md §10).
func NewCanary(t testing.TB) string {
	t.Helper()
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	return "SPECMETRY-CANARY-" + hex.EncodeToString(b[:])
}

// FindInTree byte-searches every regular file under root (databases, WAL,
// spool, logs, backups) and returns "path: needle" for each hit.
func FindInTree(t testing.TB, root string, needles ...string) []string {
	t.Helper()
	var hits []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, n := range needles {
			if bytes.Contains(b, []byte(n)) {
				hits = append(hits, p+": "+n)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hits
}
