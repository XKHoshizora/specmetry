package modsnap

import (
	"bytes"
	"fmt"
	"io"

	"golang.org/x/mod/sumdb/dirhash"
)

// hash1 computes the h1 of names whose bytes are already in memory. The
// open function hands out those bytes only; it never touches the file
// system, so dirhash.Hash1 not checking Close (hash.go:58) hides no real
// result: the real reads and closes happened before, and their results are
// kept by the snapshot (plan §2.5.4; method record §2.5).
func hash1(names []string, data map[string][]byte) (string, error) {
	return dirhash.Hash1(names, func(name string) (io.ReadCloser, error) {
		b, ok := data[name]
		if !ok {
			return nil, fmt.Errorf("hash1: no bytes for %q", name)
		}
		return io.NopCloser(bytes.NewReader(b)), nil
	})
}
