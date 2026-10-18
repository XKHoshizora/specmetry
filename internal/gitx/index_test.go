package gitx

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// ixEntry is one entry of a synthetic index file.
type ixEntry struct {
	mode     uint32
	name     string
	extended uint16 // written only when the entry is extended
	ext      bool   // the extended flag
	nameLen  int    // the name length the flags give; -1 for len(name)
}

// ixExt is one extension of a synthetic index file.
type ixExt struct {
	sig  string
	data []byte
}

// ixFile writes a synthetic index file in the format of gitformat-index:
// version 2, 3 or 4 entries (version 4 prefix-compressed, without
// padding), then the extensions, then the SHA-1 of all that, or zeros when
// zeroHash. count overrides the entry count of the header when >= 0.
func ixFile(version uint32, count int, entries []ixEntry, exts []ixExt, zeroHash bool) []byte {
	var b bytes.Buffer
	b.WriteString("DIRC")
	_ = binary.Write(&b, binary.BigEndian, version)
	if count < 0 {
		count = len(entries)
	}
	_ = binary.Write(&b, binary.BigEndian, uint32(count))
	prev := ""
	for _, e := range entries {
		start := b.Len()
		b.Write(make([]byte, 24)) // ctime, mtime, dev, ino
		_ = binary.Write(&b, binary.BigEndian, e.mode)
		b.Write(make([]byte, 12))            // uid, gid, size
		b.Write(bytes.Repeat([]byte{7}, 20)) // the object ID
		n := e.nameLen
		if n < 0 {
			n = len(e.name)
		}
		flags := uint16(min(n, 0xFFF))
		if e.ext {
			flags |= 0x4000
		}
		_ = binary.Write(&b, binary.BigEndian, flags)
		if e.ext {
			_ = binary.Write(&b, binary.BigEndian, e.extended)
		}
		if version == 4 {
			common := 0
			for common < len(prev) && common < len(e.name) && prev[common] == e.name[common] {
				common++
			}
			b.Write(varint(uint64(len(prev) - common)))
			b.WriteString(e.name[common:])
			b.WriteByte(0)
			prev = e.name
			continue
		}
		b.WriteString(e.name)
		size := (b.Len() - start + 8) &^ 7
		b.Write(make([]byte, start+size-b.Len()))
	}
	for _, x := range exts {
		b.WriteString(x.sig)
		_ = binary.Write(&b, binary.BigEndian, uint32(len(x.data)))
		b.Write(x.data)
	}
	return withChecksum(b.Bytes(), zeroHash)
}

func withChecksum(body []byte, zero bool) []byte {
	sum := sha1.Sum(body)
	if zero {
		sum = [20]byte{}
	}
	return append(bytes.Clone(body), sum[:]...)
}

// varint is git's offset encoding (OFS_DELTA), which version 4 uses for
// the length to strip from the previous path.
func varint(v uint64) []byte {
	out := []byte{byte(v & 0x7F)}
	for v >>= 7; v != 0; v >>= 7 {
		v--
		out = append([]byte{0x80 | byte(v&0x7F)}, out...)
	}
	return out
}

// ewah writes an EWAH bitmap of 3 bits in words 64-bit words with its
// running-length word at rlw.
func ewah(words, rlw uint32) []byte {
	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, uint32(3))
	_ = binary.Write(&b, binary.BigEndian, words)
	b.Write(make([]byte, 8*int(words)))
	_ = binary.Write(&b, binary.BigEndian, rlw)
	return b.Bytes()
}

func writeIndex(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

var plain = []ixEntry{
	{mode: 0o100644, name: "a.txt", nameLen: -1},
	{mode: 0o100755, name: "dir/run.sh", nameLen: -1},
	{mode: 0o160000, name: "sub", nameLen: -1},
}

var withSparseDir = []ixEntry{
	{mode: 0o100644, name: "a.txt", nameLen: -1},
	{mode: 0o040000, name: "out/", nameLen: -1, ext: true, extended: 0x4000},
}

// WP2.3 spec §4.3 step 2 and review 057 §三: the gate parses an index in
// each version's own layout (versions 2 and 3 padded to 8 bytes, version 3
// extended flags, version 4 prefix-compressed and unpadded), skips the
// optional extensions, and reports a sparse index from a sparse directory
// entry (mode 040000) or the sdir extension.
func TestIndexIsSparseReadsEachVersion(t *testing.T) {
	dir := t.TempDir()
	ext3 := []ixEntry{plain[0], {mode: 0o100644, name: "skip.txt", nameLen: -1, ext: true, extended: 0x4000}}
	long := []ixEntry{{mode: 0o100644, name: strings.Repeat("x", 0x1100), nameLen: -1}, plain[0]}
	for _, c := range []struct {
		name string
		data []byte
		want bool
	}{
		{"version 2", ixFile(2, -1, plain, nil, false), false},
		{"version 3 with skip-worktree", ixFile(3, -1, ext3, nil, false), false},
		{"version 4", ixFile(4, -1, plain, nil, false), false},
		{"version 4, paths sharing prefixes", ixFile(4, -1, []ixEntry{
			{mode: 0o100644, name: "dir/a", nameLen: -1}, {mode: 0o100644, name: "dir/ab", nameLen: -1},
			{mode: 0o100644, name: "dir/b", nameLen: -1}, {mode: 0o100644, name: "e", nameLen: -1}}, nil, false), false},
		{"a path of 4352 bytes (flags say 0xFFF)", ixFile(2, -1, long, nil, false), false},
		{"a path of 4352 bytes, version 4", ixFile(4, -1, long, nil, false), false},
		{"no entries", ixFile(2, -1, nil, nil, false), false},
		{"optional extensions", ixFile(2, -1, plain, []ixExt{{"TREE", []byte("abc")}, {"UNTR", nil}, {"ZZZZ", []byte{1, 2}}}, false), false},
		{"no checksum (index.skipHash)", ixFile(2, -1, plain, nil, true), false},
		{"a sparse directory entry, version 3", ixFile(3, -1, withSparseDir, nil, false), true},
		{"a sparse directory entry, version 4", ixFile(4, -1, withSparseDir, nil, false), true},
		{"the sdir extension", ixFile(3, -1, plain, []ixExt{{"sdir", nil}}, false), true},
		{"both", ixFile(3, -1, withSparseDir, []ixExt{{"sdir", nil}}, false), true},
	} {
		got, err := IndexIsSparse(writeIndex(t, dir, "index", c.data))
		if err != nil || got != c.want {
			t.Errorf("%s: IndexIsSparse = %v, %v; want %v", c.name, got, err, c.want)
		}
	}
}

// Review 057 §三 1 and §5.2's "cannot be parsed completely": every count,
// integer, prefix, path end, padding and extension length is checked;
// anything the parser cannot read completely is an environment error
// (exit 6), never "not sparse", and so is a required (lower-case)
// extension it does not know.
func TestIndexIsSparseRefusesWhatItCannotRead(t *testing.T) {
	dir := t.TempDir()
	good := ixFile(2, -1, plain, nil, false)
	body := good[:len(good)-20]
	type badIndex struct {
		name, why string
		data      []byte
	}
	bad := func(name, why string, data []byte) badIndex { return badIndex{name, why, data} }
	const (
		short     = "the file is too short"
		notIndex  = "the file is not an index"
		version   = "an index version Specmetry does not read"
		checksum  = "its checksum is wrong"
		inEntry   = "it ends inside an entry"
		required  = "it has a required extension Specmetry does not know"
		pathEnd   = "an entry's path does not end where its length says"
		pastEnd   = "an extension runs past the end of the file"
		extHeader = "it ends inside an extension header"
	)
	flip := bytes.Clone(good)
	flip[20] ^= 1
	v4Strip := ixFile(4, -1, plain, nil, false)
	// The second entry's strip length: after the 62 fixed bytes of entry 1,
	// its varint, name and NUL, and the second entry's 62 fixed bytes.
	second := 12 + 62 + 1 + len("a.txt") + 1 + 62
	v4Strip[second] = 0x7F // strip 127 bytes from "a.txt"
	v4Strip = withChecksum(v4Strip[:len(v4Strip)-20], false)
	v4Over := ixFile(4, -1, plain, nil, false)
	over := append(bytes.Clone(v4Over[:second]), bytes.Repeat([]byte{0xFF}, 12)...)
	over = append(over, v4Over[second+1:len(v4Over)-20]...)
	v4Over = withChecksum(over, false)
	unterminated := ixFile(2, -1, []ixEntry{{mode: 0o100644, name: "abcdefgh", nameLen: 3}}, nil, false)
	nulInName := ixFile(2, -1, []ixEntry{{mode: 0o100644, name: "ab\x00cdefg", nameLen: 8}}, nil, false)
	for _, c := range []badIndex{
		bad("an empty file", short, nil),
		bad("shorter than a header", short, []byte("DIRC\x00\x00")),
		bad("not an index", notIndex, withChecksum([]byte("RIFC\x00\x00\x00\x02\x00\x00\x00\x00"), false)),
		bad("version 1", version, ixFile(1, -1, plain, nil, false)),
		bad("version 5", version, ixFile(5, -1, plain, nil, false)),
		bad("a wrong checksum", checksum, flip),
		bad("truncated", checksum, good[:len(good)-7]),
		bad("truncated in an entry, checksum recomputed", inEntry, withChecksum(body[:40], false)),
		bad("more entries counted than written", inEntry, ixFile(2, 4, plain, nil, false)),
		bad("fewer entries counted than written", required, ixFile(2, 2, plain, nil, false)),
		bad("an extended flag in version 2", "a version 2 entry has extended flags", ixFile(2, -1, []ixEntry{{mode: 0o100644, name: "a", nameLen: -1, ext: true}}, nil, false)),
		bad("unknown extended flags", "an entry has extended flags Specmetry does not know", ixFile(3, -1, []ixEntry{{mode: 0o100644, name: "a", nameLen: -1, ext: true, extended: 0x0100}}, nil, false)),
		bad("a path that does not end where its length says", pathEnd, unterminated),
		bad("a NUL inside a path", pathEnd, nulInName),
		bad("version 4: a strip longer than the previous path", "an entry strips more of the previous path than it has", v4Strip),
		bad("version 4: a strip length that overflows", "an entry's path prefix length is malformed", v4Over),
		bad("version 4: a path length the flags contradict", inEntry, ixFile(4, -1, []ixEntry{{mode: 0o100644, name: "abc", nameLen: 5}}, nil, false)),
		bad("an unknown required extension", required, ixFile(2, -1, plain, []ixExt{{"abcd", nil}}, false)),
		bad("an extension longer than the file", pastEnd, withChecksum(append(bytes.Clone(body), []byte("TREE\x00\x00\x10\x00ab")...), false)),
		bad("bytes after the last extension", extHeader, withChecksum(append(bytes.Clone(body), []byte("TRE")...), false)),
		bad("a link extension too short", "its link extension is too short", ixFile(2, -1, plain, []ixExt{{"link", make([]byte, 19)}}, false)),
	} {
		got, err := IndexIsSparse(writeIndex(t, dir, "index", c.data))
		if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), c.why) {
			t.Errorf("%s: IndexIsSparse = %v, %v; want an environment error: %s", c.name, got, err, c.why)
			continue
		}
		if strings.Contains(err.Error(), dir) {
			t.Errorf("%s: the error names the path: %v", c.name, err)
		}
	}
}

// No input makes the parser panic or read out of bounds: every prefix of
// a real-looking index, with its checksum made right, is an environment
// error, except the prefixes that end where an index may end (after the
// entries, or after an extension), which parse.
func TestIndexIsSparseOnEveryPrefix(t *testing.T) {
	dir := t.TempDir()
	tree := []byte("0 1\n")
	for _, version := range []uint32{2, 3, 4} {
		entries := append(append([]ixEntry{}, plain...), withSparseDir...)
		if version == 2 { // no extended flags in version 2
			entries[len(entries)-1].ext = false
		}
		full := ixFile(version, -1, entries, []ixExt{{"TREE", tree}, {"sdir", nil}}, false)
		body := full[:len(full)-20]
		endOfEntries := len(ixFile(version, -1, entries, nil, false)) - 20
		ends := map[int]bool{endOfEntries: true, endOfEntries + 8 + len(tree): true}
		for n := 0; n < len(body); n++ {
			got, err := IndexIsSparse(writeIndex(t, dir, "index", withChecksum(body[:n], false)))
			switch {
			case ends[n] && (err != nil || !got):
				t.Fatalf("version %d, %d of %d bytes (a complete index): %v, %v", version, n, len(body), got, err)
			case !ends[n] && apperr.ExitCode(err) != apperr.ExitEnvironment:
				t.Fatalf("version %d, %d of %d bytes: %v, %v; want an environment error", version, n, len(body), got, err)
			}
		}
		if got, err := IndexIsSparse(writeIndex(t, dir, "index", full)); err != nil || !got {
			t.Fatalf("version %d, whole: %v, %v", version, got, err)
		}
	}
}

// Review 057 §三 2 and N1: a split index's link names its shared index
// (sharedindex.<hash> beside it), whose entries count as well; a link of
// all zeros names none, and no file is looked for. A shared index that is
// missing, damaged, not the one named, or itself linked is an error.
func TestIndexIsSparseReadsTheSharedIndex(t *testing.T) {
	trailer := func(s []byte) []byte { return bytes.Clone(s[len(s)-20:]) }
	named := func(b byte) func([]byte) []byte { return func([]byte) []byte { return bytes.Repeat([]byte{b}, 20) } }
	shared := ixFile(2, -1, plain, nil, false)
	for _, c := range []struct {
		name   string
		shared []byte                     // the file the link names; nil: none
		link   func(shared []byte) []byte // the link's data; nil: the shared index's checksum
		want   bool
		fails  bool
	}{
		{name: "a plain shared index", shared: shared},
		{name: "a sparse directory entry in the shared index", shared: ixFile(3, -1, withSparseDir, nil, false), want: true},
		{name: "sdir in the shared index", shared: ixFile(2, -1, plain, []ixExt{{"sdir", nil}}, false), want: true},
		{name: "with the delete and replace bitmaps", shared: shared,
			link: func(s []byte) []byte { return append(append(trailer(s), ewah(1, 0)...), ewah(0, 0)...) }},
		{name: "the shared index missing", link: named(0xAB), fails: true},
		{name: "a damaged shared index", shared: shared[:30], link: named(0xCD), fails: true},
		{name: "a shared index with another checksum", shared: shared, link: named(0xEF), fails: true},
		{name: "a shared index that is linked itself", shared: ixFile(2, -1, plain, []ixExt{{"link", make([]byte, 20)}}, false), fails: true},
		{name: "a truncated bitmap", shared: shared, fails: true,
			link: func(s []byte) []byte { return append(trailer(s), ewah(2, 0)[:20]...) }},
		{name: "only the delete bitmap", shared: shared, fails: true,
			link: func(s []byte) []byte { return append(trailer(s), ewah(1, 0)...) }},
		{name: "bytes after the bitmaps", shared: shared, fails: true,
			link: func(s []byte) []byte {
				return append(append(append(trailer(s), ewah(1, 0)...), ewah(0, 0)...), 0)
			}},
		{name: "a running-length word out of the bitmap", shared: shared, fails: true,
			link: func(s []byte) []byte { return append(append(trailer(s), ewah(1, 1)...), ewah(0, 0)...) }},
	} {
		dir := t.TempDir()
		var link []byte
		if c.link != nil {
			link = c.link(c.shared)
		} else {
			link = trailer(c.shared)
		}
		if c.shared != nil {
			writeIndex(t, dir, "sharedindex."+hex.EncodeToString(link[:20]), c.shared)
		}
		got, err := IndexIsSparse(writeIndex(t, dir, "index", ixFile(2, -1, plain[:1], []ixExt{{"link", link}}, false)))
		switch {
		case c.fails && apperr.ExitCode(err) != apperr.ExitEnvironment:
			t.Errorf("%s: IndexIsSparse = %v, %v; want an environment error", c.name, got, err)
		case !c.fails && (err != nil || got != c.want):
			t.Errorf("%s: IndexIsSparse = %v, %v; want %v", c.name, got, err, c.want)
		}
	}

	// A link of zeros names no shared index: a sharedindex.000… beside it
	// is not read, even one that would make the index sparse.
	dir := t.TempDir()
	writeIndex(t, dir, "sharedindex."+strings.Repeat("0", 40), ixFile(3, -1, withSparseDir, nil, false))
	got, err := IndexIsSparse(writeIndex(t, dir, "index", ixFile(2, -1, plain, []ixExt{{"link", make([]byte, 20)}}, false)))
	if err != nil || got {
		t.Errorf("a link of zeros: IndexIsSparse = %v, %v; want false", got, err)
	}
}

// Review 057 N1: an index that does not exist is an empty index (a new
// repository has none, with or without a HEAD); one that cannot be read is
// an error, as is a directory where the index should be.
func TestIndexIsSparseWithoutAnIndex(t *testing.T) {
	dir := t.TempDir()
	if got, err := IndexIsSparse(filepath.Join(dir, "index")); err != nil || got {
		t.Errorf("no index: %v, %v; want false", got, err)
	}
	if err := os.Mkdir(filepath.Join(dir, "adir"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := IndexIsSparse(filepath.Join(dir, "adir")); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("a directory: %v; want an environment error", err)
	}
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		return // file modes do not keep root or Windows from reading
	}
	p := writeIndex(t, dir, "unreadable", ixFile(2, -1, plain, nil, false))
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := IndexIsSparse(p); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("an unreadable index: %v; want an environment error", err)
	}
}

// The indexes git itself writes: every layout the spec's experiments used
// (S22) reads as not sparse, and git's own sparse index as sparse, in
// versions 3 and 4.
func TestIndexIsSparseOnIndexesGitWrites(t *testing.T) {
	index := func(repo *testutil.GitRepo) string { return filepath.Join(repo.Dir, ".git", "index") }
	version := func(t *testing.T, repo *testutil.GitRepo) uint32 {
		b, err := os.ReadFile(index(repo))
		if err != nil {
			t.Fatal(err)
		}
		return binary.BigEndian.Uint32(b[4:8])
	}
	check := func(t *testing.T, name string, repo *testutil.GitRepo, want bool) {
		t.Helper()
		got, err := IndexIsSparse(index(repo))
		if err != nil || got != want {
			t.Errorf("%s: IndexIsSparse = %v, %v; want %v", name, got, err, want)
		}
	}
	repo := committedRepo(t)
	repo.WriteFile("dir/b.txt", "b\n")
	repo.Commit("b")
	check(t, "version 2", repo, false)
	repo.Git("update-index", "--skip-worktree", "README.md")
	if v := version(t, repo); v != 3 {
		t.Fatalf("skip-worktree gave version %d", v)
	}
	check(t, "version 3 with skip-worktree", repo, false)
	repo.Git("update-index", "--index-version", "4")
	check(t, "version 4", repo, false)
	repo.Git("-c", "core.untrackedCache=true", "update-index", "--untracked-cache")
	repo.WriteFile("untracked.txt", "u\n")
	repo.Git("-c", "core.untrackedCache=true", "status", "--porcelain")
	if b, _ := os.ReadFile(index(repo)); !bytes.Contains(b, []byte("UNTR")) {
		t.Error("git wrote no untracked cache")
	}
	check(t, "an untracked cache", repo, false)

	split := committedRepo(t)
	split.WriteFile("dir/b.txt", "b\n")
	split.Commit("b")
	split.Git("update-index", "--split-index")
	split.WriteFile("dir/b.txt", "changed\n")
	split.WriteFile("new.txt", "n\n")
	split.Git("add", "-A")
	b, err := os.ReadFile(index(split))
	if err != nil || !bytes.Contains(b, []byte("link")) {
		t.Fatalf("git wrote no split index: %v", err)
	}
	check(t, "a split index", split, false)
	split.Git("update-index", "--index-version", "4")
	check(t, "a split index, version 4", split, false)

	sparse, _, _ := sparseRepo(t)
	check(t, "a sparse index", sparse, true)
	sparse.Git("-c", "index.sparse=true", "update-index", "--index-version", "4")
	if version(t, sparse) == 4 {
		if b, _ := os.ReadFile(index(sparse)); bytes.Contains(b, []byte("sdir")) {
			check(t, "a sparse index, version 4", sparse, true)
		}
	}

	skip := committedRepo(t)
	skip.Git("-c", "index.skipHash=true", "update-index", "--force-write-index")
	if b, _ := os.ReadFile(index(skip)); bytes.Equal(b[len(b)-20:], make([]byte, 20)) {
		check(t, "index.skipHash", skip, false)
	} else {
		t.Log("this git wrote a checksum with index.skipHash=true")
	}
}
