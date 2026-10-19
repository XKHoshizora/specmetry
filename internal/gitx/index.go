package gitx

import (
	"bytes"
	"crypto/sha1" //nolint:gosec // the index format's own checksum, not a security use
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// Index files (gitformat-index), read by Specmetry itself, without git, for
// the sparse gate of Clean (WP2.3 spec §4.3 step 2): git cannot be asked
// whether an index is sparse, since git reading a sparse index may expand
// it and rewrite the timestamps of object files, ls-files --sparse
// included (experiments X2 to X4). The parse is complete or it fails:
// every count, integer, path, padding and extension length is checked
// against the file (review 057 §三), and a required extension it does not
// know is an error, never ignored. SHA-1 only (TD-26).

// indexHashSize is the size of a SHA-1 object ID and checksum.
const indexHashSize = 20

// indexFixedSize is the fixed part of an entry: ctime, mtime, dev, ino,
// mode, uid, gid and size (40 bytes), the object ID and the 16-bit flags.
const indexFixedSize = 40 + indexHashSize + 2

const (
	entryNameMask = 0x0FFF
	entryExtended = 0x4000
	// extendedKnown are the extended flags git defines: intent-to-add and
	// skip-worktree. Any other bit makes git refuse the index.
	extendedKnown = 0x2000 | 0x4000
	// sparseDirMode is the mode of a sparse directory entry (S_IFDIR).
	sparseDirMode = 0o040000
)

// indexInfo is what the gate needs from one index file.
type indexInfo struct {
	// sparse: a sparse directory entry or the sdir extension.
	sparse bool
	// linked: a link extension (a split index); shared is the shared
	// index it names, or nil when its hash is all zeros (none).
	linked bool
	shared *[indexHashSize]byte
	// checksum is the file's trailing hash, all zeros when git left it out
	// (index.skipHash).
	checksum [indexHashSize]byte
}

// indexProblem is an index that cannot be read completely. The problem is
// a fixed label: no path, no byte of the file.
func indexProblem(problem string) error {
	return apperr.New(apperr.Environment,
		"cannot read the git index ("+problem+"), so Specmetry cannot tell the state of the worktree; run git status in this directory to see whether git can read it")
}

// IndexIsSparse parses the index file at path without running git
// (versions 2 to 4, and the shared index a split index links to, which
// must be beside it) and reports whether it has a sparse directory entry
// or the sdir extension in either. An index that does not exist is an
// empty index, as in a new repository (review 057 N1); anything it cannot
// read or parse completely, a missing shared index included, is an
// environment error. Clean calls it on its private copy before any git
// command reads that copy.
func IndexIsSparse(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, indexProblem("the file cannot be read: " + osProblem(err))
	}
	info, err := parseIndex(data)
	if err != nil || info.shared == nil {
		return info.sparse, err
	}
	shared, err := os.ReadFile(filepath.Join(filepath.Dir(path), sharedIndexName(*info.shared)))
	if err != nil {
		return false, indexProblem("the shared index of a split index cannot be read: " + osProblem(err))
	}
	sinfo, err := parseIndex(shared)
	switch {
	case err != nil:
		return false, err
	case sinfo.linked:
		return false, indexProblem("the shared index of a split index is split itself")
	case sinfo.checksum != *info.shared:
		return false, indexProblem("the shared index is not the one the split index names")
	}
	return info.sparse || sinfo.sparse, nil
}

// sharedIndexName is the file name of the shared index a link names.
func sharedIndexName(hash [indexHashSize]byte) string {
	return "sharedindex." + hex.EncodeToString(hash[:])
}

// parseIndex parses one index file completely.
func parseIndex(data []byte) (indexInfo, error) {
	var info indexInfo
	if len(data) < 12+indexHashSize {
		return info, indexProblem("the file is too short")
	}
	if string(data[:4]) != "DIRC" {
		return info, indexProblem("the file is not an index")
	}
	version := binary.BigEndian.Uint32(data[4:8])
	if version < 2 || version > 4 {
		return info, indexProblem("an index version Specmetry does not read")
	}
	body := data[:len(data)-indexHashSize]
	copy(info.checksum[:], data[len(data)-indexHashSize:])
	if info.checksum != ([indexHashSize]byte{}) && sha1.Sum(body) != info.checksum { //nolint:gosec // see the import
		return info, indexProblem("its checksum is wrong")
	}
	count := int(binary.BigEndian.Uint32(data[8:12]))
	off := 12
	var prev []byte
	for i := 0; i < count; i++ {
		var mode uint32
		var err error
		mode, prev, off, err = indexEntry(body, off, version, i == 0, prev)
		if err != nil {
			return info, err
		}
		if mode == sparseDirMode {
			info.sparse = true
		}
	}
	for off < len(body) {
		if len(body)-off < 8 {
			return info, indexProblem("it ends inside an extension header")
		}
		sig := body[off : off+4]
		size := int(binary.BigEndian.Uint32(body[off+4 : off+8]))
		if size < 0 || size > len(body)-off-8 { // negative only where int has 32 bits
			return info, indexProblem("an extension runs past the end of the file")
		}
		ext := body[off+8 : off+8+size]
		switch string(sig) {
		case "sdir":
			info.sparse = true
		case "link":
			if info.linked {
				return info, indexProblem("it has two link extensions")
			}
			info.linked = true
			if err := readLink(ext, &info); err != nil {
				return info, err
			}
		default:
			// Only an extension whose signature begins with an upper-case
			// letter is optional; any other is required, and unknown.
			if sig[0] < 'A' || sig[0] > 'Z' {
				return info, indexProblem("it has a required extension Specmetry does not know")
			}
		}
		off += 8 + size
	}
	return info, nil
}

// indexEntry parses the entry at off and returns its mode, its path (for
// version 4's prefix compression) and the offset of the next entry.
func indexEntry(body []byte, off int, version uint32, first bool, prev []byte) (uint32, []byte, int, error) {
	truncated := indexProblem("it ends inside an entry")
	if len(body)-off < indexFixedSize {
		return 0, nil, 0, truncated
	}
	mode := binary.BigEndian.Uint32(body[off+24 : off+28])
	flags := binary.BigEndian.Uint16(body[off+indexFixedSize-2 : off+indexFixedSize])
	p := off + indexFixedSize
	if flags&entryExtended != 0 {
		if version < 3 {
			return 0, nil, 0, indexProblem("a version 2 entry has extended flags")
		}
		if len(body)-p < 2 {
			return 0, nil, 0, truncated
		}
		if binary.BigEndian.Uint16(body[p:p+2])&^extendedKnown != 0 {
			return 0, nil, 0, indexProblem("an entry has extended flags Specmetry does not know")
		}
		p += 2
	}
	nameLen := int(flags & entryNameMask)
	if version == 4 {
		strip, n, ok := indexVarint(body[p:])
		if !ok {
			return 0, nil, 0, indexProblem("an entry's path prefix length is malformed")
		}
		p += n
		kept := 0
		if !first {
			if strip > len(prev) {
				return 0, nil, 0, indexProblem("an entry strips more of the previous path than it has")
			}
			kept = len(prev) - strip
		}
		suffix, end, err := indexName(body, p, nameLen, kept)
		if err != nil {
			return 0, nil, 0, err
		}
		name := append(bytes.Clone(prev[:kept]), suffix...)
		return mode, name, end + 1, nil
	}
	_, end, err := indexName(body, p, nameLen, 0)
	if err != nil {
		return 0, nil, 0, err
	}
	// Versions 2 and 3 pad each entry with 1 to 8 NULs to a multiple of 8.
	size := (end - off + 8) &^ 7
	if size > len(body)-off {
		return 0, nil, 0, truncated
	}
	return mode, nil, off + size, nil
}

// indexName reads the path (or, in version 4, its suffix after kept bytes
// of the previous path) that starts at p, and returns it and the offset of
// its terminating NUL. A length below 0xFFF in the flags is the path's
// whole length and must end exactly at a NUL, with no NUL inside; 0xFFF
// means a path that long or longer, ended by its NUL.
func indexName(body []byte, p, nameLen, kept int) ([]byte, int, error) {
	if nameLen < entryNameMask {
		n := nameLen - kept
		if n < 0 {
			return nil, 0, indexProblem("an entry's path is shorter than the prefix it keeps")
		}
		if n >= len(body)-p {
			return nil, 0, indexProblem("it ends inside an entry")
		}
		name := body[p : p+n]
		if body[p+n] != 0 || bytes.IndexByte(name, 0) >= 0 {
			return nil, 0, indexProblem("an entry's path does not end where its length says")
		}
		return name, p + n, nil
	}
	n := bytes.IndexByte(body[p:], 0)
	if n < 0 {
		return nil, 0, indexProblem("it ends inside an entry")
	}
	return body[p : p+n], p + n, nil
}

// varintMax bounds a decoded prefix length: far above any path, and far
// below an overflow.
const varintMax = 1 << 30

// indexVarint decodes git's offset encoding (the one OFS_DELTA uses):
// returns the value and the bytes it took, or false when it grows past
// varintMax or runs past the end of the data.
func indexVarint(b []byte) (int, int, bool) {
	if len(b) == 0 {
		return 0, 0, false
	}
	c := b[0]
	v := int(c & 0x7F)
	n := 1
	for c&0x80 != 0 {
		v++
		if v > varintMax>>7 || n >= len(b) {
			return 0, 0, false
		}
		c = b[n]
		n++
		v = v<<7 | int(c&0x7F)
	}
	return v, n, true
}

// readLink reads a link extension: the shared index's hash (all zeros for
// none), then, when anything follows, the delete and the replace bitmaps
// (EWAH), which must take exactly the rest.
func readLink(ext []byte, info *indexInfo) error {
	if len(ext) < indexHashSize {
		return indexProblem("its link extension is too short")
	}
	var hash [indexHashSize]byte
	copy(hash[:], ext)
	if hash != ([indexHashSize]byte{}) {
		info.shared = &hash
	}
	rest := ext[indexHashSize:]
	if len(rest) == 0 {
		return nil
	}
	for range 2 {
		n, ok := ewahSize(rest)
		if !ok || n > len(rest) {
			return indexProblem("a bitmap of its link extension is malformed")
		}
		rest = rest[n:]
	}
	if len(rest) != 0 {
		return indexProblem("its link extension has bytes after its bitmaps")
	}
	return nil
}

// EWAH bitmaps: a 32-bit bit count and word count, the 64-bit words, and
// the 32-bit position of the running-length word.
const (
	ewahHeader = 8
	ewahFooter = 4
)

// ewahSize returns the size of the EWAH bitmap at the start of b, which
// must hold all of it: header, words and footer (a bitmap of no word is
// 12 bytes; review 058 B2: 8 to 11 bytes were taken for one). The
// running-length word must be one of the words.
func ewahSize(b []byte) (int, bool) {
	if len(b) < ewahHeader+ewahFooter {
		return 0, false
	}
	words := int(binary.BigEndian.Uint32(b[4:8]))
	if words < 0 || words > (len(b)-ewahHeader-ewahFooter)/8 {
		return 0, false
	}
	size := ewahHeader + 8*words + ewahFooter
	rlw := int(binary.BigEndian.Uint32(b[size-4 : size]))
	if words == 0 && rlw != 0 || words > 0 && rlw >= words {
		return 0, false
	}
	return size, true
}
