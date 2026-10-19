package gitx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// linkIndex is an index of no entries whose only extension is a link
// with payload, its checksum right, or all zeros (index.skipHash): the
// outer layout is valid, so a parse reaches the link's bitmaps.
func linkIndex(payload []byte, zeroHash bool) []byte {
	return ixFile(2, -1, nil, []ixExt{{"link", payload}}, zeroHash)
}

// Review 058 B2, as reported: a link whose payload is the zero shared
// hash and 8 to 11 bytes of a bitmap header, in a 68- to 71-byte index
// with a zero (skipHash) trailer, made IndexIsSparse panic (slice bounds
// out of range [12:n]): the header check took 8 bytes for enough, and the
// slice that followed read past the payload into the trailer. Each is an
// environment error, with the checksum left out and with it right.
func TestIndexIsSparseShortBitmapHeader(t *testing.T) {
	dir := t.TempDir()
	for n := 8; n <= 11; n++ {
		for _, zero := range []bool{true, false} {
			data := linkIndex(make([]byte, 20+n), zero)
			if zero && len(data) != 60+n {
				t.Fatalf("the index is %d bytes, want %d", len(data), 60+n)
			}
			got, err := IndexIsSparse(writeIndex(t, dir, "index", data))
			if apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Errorf("a %d-byte bitmap header, zero trailer %v: IndexIsSparse = %v, %v; want an environment error", n, zero, got, err)
			}
		}
	}
}

// The same index as a repository's .git/index: Clean fails with an
// environment error before any git command reads the index, leaves the
// repository's files (content and modification time) as they were and its
// private directories removed.
func TestCleanWithAShortBitmapHeader(t *testing.T) {
	requireSafeGit(t)
	for n := 8; n <= 11; n++ {
		t.Run(fmt.Sprintf("%d bytes", n), func(t *testing.T) {
			repo := testutil.NewGitRepo(t)
			if err := os.WriteFile(filepath.Join(repo.Dir, ".git", "index"), linkIndex(make([]byte, 20+n), true), 0o644); err != nil {
				t.Fatal(err)
			}
			dir, env, calls := testutil.PassthroughGit(t, testutil.PassAsIs)
			useGit(t, dir, env)
			check := unchanged(t, filepath.Join(repo.Dir, ".git"))
			if got, err := cleanForTest(t, repo.Dir); apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Errorf("Clean = %v, %v; want an environment error", got, err)
			}
			check()
			for _, c := range calls() {
				if readsIndex(c.Args) {
					t.Errorf("a command that reads the index ran: %q", c.Args)
				}
			}
		})
	}
}

// Both bitmaps cut at every length, with the outer index valid (its link's
// size and the checksum made to agree, or the checksum left out): only the
// whole payload, or the bare hash, parses; every other length is an
// environment error, and nothing panics. Boundaries: a bitmap of no word
// is 12 bytes, so 12 bytes are one bitmap (still short of two), 24 two.
func TestIndexIsSparseCutBitmaps(t *testing.T) {
	dir := t.TempDir()
	hash := make([]byte, 20)
	for _, c := range []struct {
		name     string
		bitmaps  []byte
		complete map[int]bool // lengths of the bitmaps that parse
	}{
		{"two empty bitmaps", append(ewah(0, 0), ewah(0, 0)...), map[int]bool{0: true, 24: true}},
		{"a bitmap of one word, then an empty one", append(ewah(1, 0), ewah(0, 0)...), map[int]bool{0: true, 32: true}},
		{"an empty bitmap, then one of two words", append(ewah(0, 0), ewah(2, 1)...), map[int]bool{0: true, 40: true}},
	} {
		for n := 0; n <= len(c.bitmaps); n++ {
			for _, zero := range []bool{true, false} {
				payload := append(bytes.Clone(hash), c.bitmaps[:n]...)
				got, err := IndexIsSparse(writeIndex(t, dir, "index", linkIndex(payload, zero)))
				switch {
				case c.complete[n] && (err != nil || got):
					t.Errorf("%s, %d bytes, zero trailer %v: %v, %v; want it read", c.name, n, zero, got, err)
				case !c.complete[n] && apperr.ExitCode(err) != apperr.ExitEnvironment:
					t.Errorf("%s, %d bytes, zero trailer %v: %v, %v; want an environment error", c.name, n, zero, got, err)
				}
			}
		}
	}
}

// Word counts and running-length positions that do not fit, and the same
// in a link to a real shared index: errors, never a read past the payload.
func TestIndexIsSparseBadBitmapCounts(t *testing.T) {
	dir := t.TempDir()
	withWords := func(words uint32, present int, rlw uint32) []byte {
		var b bytes.Buffer
		_ = binary.Write(&b, binary.BigEndian, uint32(64))
		_ = binary.Write(&b, binary.BigEndian, words)
		b.Write(make([]byte, 8*present))
		_ = binary.Write(&b, binary.BigEndian, rlw)
		return b.Bytes()
	}
	for _, c := range []struct {
		name   string
		first  []byte
		second []byte
	}{
		{"a word count of 2^32-1", withWords(0xFFFFFFFF, 1, 0), ewah(0, 0)},
		{"a word count of 2^31", withWords(1<<31, 1, 0), ewah(0, 0)},
		{"two words counted, one present", withWords(2, 1, 0), ewah(0, 0)},
		{"one word counted, two present", withWords(1, 2, 0), ewah(0, 0)},
		{"the running-length word past the words", withWords(1, 1, 1), ewah(0, 0)},
		{"a running-length word in an empty bitmap", withWords(0, 0, 1), ewah(0, 0)},
		{"the second bitmap's count too large", ewah(0, 0), withWords(3, 1, 0)},
	} {
		for _, zero := range []bool{true, false} {
			payload := append(append(make([]byte, 20), c.first...), c.second...)
			got, err := IndexIsSparse(writeIndex(t, dir, "index", linkIndex(payload, zero)))
			if apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Errorf("%s, zero trailer %v: %v, %v; want an environment error", c.name, zero, got, err)
			}
		}
	}
	// A real shared index, the link's bitmaps cut short.
	shared := ixFile(2, -1, plain, nil, false)
	sum := shared[len(shared)-20:]
	writeIndex(t, dir, sharedIndexName([20]byte(sum)), shared)
	for n := 0; n < 12; n++ {
		payload := append(bytes.Clone(sum), ewah(0, 0)[:n]...)
		if n == 0 {
			continue // the bare hash: no bitmaps, valid
		}
		got, err := IndexIsSparse(writeIndex(t, dir, "index", ixFile(2, -1, plain[:1], []ixExt{{"link", payload}}, true)))
		if apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Errorf("a shared index, a %d-byte bitmap: %v, %v; want an environment error", n, got, err)
		}
	}
	got, err := IndexIsSparse(writeIndex(t, dir, "index", ixFile(2, -1, plain[:1],
		[]ixExt{{"link", append(append(bytes.Clone(sum), ewah(1, 0)...), ewah(0, 0)...)}}, true)))
	if err != nil || got {
		t.Errorf("a valid split index: %v, %v", got, err)
	}
}

// Bounded random damage to indexes with links (bit flips, cuts, inserted
// bytes inside the link, its size header kept in agreement and the
// checksum made right or left out, so that the outer parse passes): every
// result is a read or an environment error. A panic fails the test; the
// parser recovers from nothing.
func TestIndexIsSparseRandomDamage(t *testing.T) {
	dir := t.TempDir()
	rng := rand.New(rand.NewPCG(58, 2026))
	bases := [][]byte{
		append(append(make([]byte, 20), ewah(1, 0)...), ewah(0, 0)...),
		append(append(make([]byte, 20), ewah(2, 1)...), ewah(1, 0)...),
		make([]byte, 20),
	}
	for i := 0; i < 4000; i++ {
		p := bytes.Clone(bases[i%len(bases)])
		switch rng.IntN(3) {
		case 0:
			if len(p) > 0 {
				p[rng.IntN(len(p))] ^= byte(1 << rng.IntN(8))
			}
		case 1:
			p = p[:rng.IntN(len(p)+1)]
		case 2:
			at := rng.IntN(len(p) + 1)
			p = append(p[:at], append([]byte{byte(rng.IntN(256))}, p[at:]...)...)
		}
		entries := plain[:rng.IntN(len(plain)+1)]
		data := ixFile(uint32(2+rng.IntN(3)), -1, entries, []ixExt{{"link", p}}, rng.IntN(2) == 0)
		_, err := IndexIsSparse(writeIndex(t, dir, "index", data))
		if err != nil && apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Fatalf("damage %d: %v", i, err)
		}
	}
}

// FuzzIndexIsSparse: whatever the bytes, a read or an environment error;
// go test runs the seeds, go test -fuzz explores further.
func FuzzIndexIsSparse(f *testing.F) {
	for n := 8; n <= 11; n++ {
		f.Add(linkIndex(make([]byte, 20+n), true))
	}
	f.Add(linkIndex(append(append(make([]byte, 20), ewah(1, 0)...), ewah(0, 0)...), false))
	f.Add(ixFile(4, -1, plain, []ixExt{{"TREE", []byte("0 1\n")}}, false))
	f.Add(ixFile(3, -1, withSparseDir, []ixExt{{"sdir", nil}}, true))
	dir := f.TempDir()
	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := IndexIsSparse(writeIndex(t, dir, "index", data))
		if err != nil && apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Fatalf("%v", err)
		}
	})
}
