package source_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/source"
)

var ctx = context.Background()

func sum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

const (
	idA = "01J9Z3K5Q8R2T4V6W8X0Y2Z4E0"
	idB = "01J9Z3K5Q8R2T4V6W8X0Y2Z4E1"
	idC = "01J9Z3K5Q8R2T4V6W8X0Y2Z4E2"
)

var (
	hashA = strings.Repeat("a", 64)
	hashB = strings.Repeat("b", 64)
	hashC = strings.Repeat("c", 64)
)

// data-model §4.3: sha256 of "<source_id>:<acked_sha256>\n" for the active
// sources sorted by id.
func TestSetHash(t *testing.T) {
	set := []source.Member{{ID: idA, AckedSHA256: hashA}, {ID: idB, AckedSHA256: hashB}, {ID: idC, AckedSHA256: hashC}}
	want := sum([]byte(idA + ":" + hashA + "\n" + idB + ":" + hashB + "\n" + idC + ":" + hashC + "\n"))
	if got := source.SetHash(set); got != want {
		t.Fatalf("SetHash = %s, want %s", got, want)
	}
	if got := source.SetHash(nil); got != sum(nil) {
		t.Fatalf("SetHash of no sources = %s, want the hash of the empty string", got)
	}
}

// MVP-SRC-06: the same sources (the same IDs and acknowledged hashes), in
// any order, give the same hash. See PD-25 for what "the same sources"
// means when a file is registered again under a new ID.
func TestSetHashIgnoresOrder(t *testing.T) {
	a, b, c := source.Member{ID: idA, AckedSHA256: hashA}, source.Member{ID: idB, AckedSHA256: hashB}, source.Member{ID: idC, AckedSHA256: hashC}
	want := source.SetHash([]source.Member{a, b, c})
	for _, order := range [][]source.Member{{a, c, b}, {b, a, c}, {b, c, a}, {c, a, b}, {c, b, a}} {
		if got := source.SetHash(order); got != want {
			t.Errorf("order %v: %s, want %s", order, got, want)
		}
	}
	input := []source.Member{c, a, b}
	source.SetHash(input)
	if input[0] != c {
		t.Error("SetHash reordered its input")
	}
}

// Adding, removing or acknowledging a source changes the hash.
func TestSetHashChangesWithTheSet(t *testing.T) {
	base := []source.Member{{ID: idA, AckedSHA256: hashA}, {ID: idB, AckedSHA256: hashB}}
	h := source.SetHash(base)
	for name, other := range map[string][]source.Member{
		"added":         append(append([]source.Member{}, base...), source.Member{ID: idC, AckedSHA256: hashC}),
		"removed":       base[:1],
		"acknowledged":  {{ID: idA, AckedSHA256: hashC}, {ID: idB, AckedSHA256: hashB}},
		"re-registered": {{ID: idC, AckedSHA256: hashA}, {ID: idB, AckedSHA256: hashB}},
	} {
		if source.SetHash(other) == h {
			t.Errorf("%s: the hash did not change", name)
		}
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte(strings.Repeat("需求 requirement\n", 100000)) // larger than one read
	p := filepath.Join(dir, "brief.md")
	if err := os.WriteFile(p, content, 0o600); err != nil {
		t.Fatal(err)
	}
	got, size, err := source.HashFile(ctx, p)
	if err != nil || got != sum(content) || size != int64(len(content)) {
		t.Fatalf("HashFile = %s, %d, %v; want %s, %d", got, size, err, sum(content), len(content))
	}
	if _, _, err := source.HashFile(ctx, filepath.Join(dir, "gone.md")); !errors.Is(err, source.ErrMissing) {
		t.Fatalf("missing file: %v", err)
	}
	if _, _, err := source.HashFile(ctx, dir); !errors.Is(err, source.ErrMissing) {
		t.Fatalf("a directory: %v", err)
	}
}

// cancelAfter reports cancellation from its n-th Err call on.
type cancelAfter struct {
	context.Context
	n, calls int
}

func (c *cancelAfter) Err() error {
	c.calls++
	if c.calls >= c.n {
		return context.Canceled
	}
	return nil
}

// Hashing stops when the context is cancelled, before or while reading,
// and never returns a hash of part of the file.
func TestHashFileStopsWhenCanceled(t *testing.T) {
	p := filepath.Join(t.TempDir(), "big.md")
	if err := os.WriteFile(p, make([]byte, 8<<20), 0o600); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if h, _, err := source.HashFile(canceled, p); !errors.Is(err, context.Canceled) || h != "" || apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("cancelled before: %q, %v", h, err)
	}
	during := &cancelAfter{Context: ctx, n: 3}
	if h, _, err := source.HashFile(during, p); !errors.Is(err, context.Canceled) || h != "" {
		t.Fatalf("cancelled while reading: %q, %v", h, err)
	}
	if during.calls > during.n {
		t.Fatalf("read on after the cancellation (%d checks)", during.calls)
	}
}

func TestParseKind(t *testing.T) {
	for _, k := range []string{"client_requirement", "product_doc", "design_doc", "issue", "note", "other"} {
		if _, ok := source.ParseKind(k); !ok {
			t.Errorf("%s refused", k)
		}
	}
	for _, k := range []string{"", "Other", "spec"} {
		if _, ok := source.ParseKind(k); ok {
			t.Errorf("%q accepted", k)
		}
	}
}
