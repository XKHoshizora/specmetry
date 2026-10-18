package gitx

import (
	"bytes"
	"crypto/sha1" //nolint:gosec // the pack index format's own checksum, not a security use
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The storage check (WP2.3 spec §5.2 step 2; reviews 052 B2-R, 055 B2-R2)
// tells a commit that was reclaimed from one that storage which is
// damaged, incomplete or unreadable hides from git: git answers `missing`,
// exit 1 and nothing on stderr for both (K3, K6, K12, K13, K15, K18, K19).
// Specmetry reads the object directories itself, without git, and never
// writes, rebuilds or repairs anything. It reads every pack index, so its
// cost grows with the objects; it runs only when an old tip is missing.

// maxAlternateDepth is as deep as git reads alternates (link_alt_odb_entries).
const maxAlternateDepth = 5

var (
	packFile      = regexp.MustCompile(`^pack-([0-9a-f]{40})\.(pack|idx|keep|promisor|bitmap|rev|mtimes)$`)
	midxFile      = regexp.MustCompile(`^multi-pack-index(-[0-9a-f]{40}\.(bitmap|rev))?$`)
	midxChainFile = regexp.MustCompile(`^(multi-pack-index-chain|multi-pack-index-[0-9a-f]{40}\.(midx|bitmap|rev))$`)
)

// errStorage is a failed storage check. It is never shown: the caller
// reports that the commit cannot be confirmed absent.
var errStorage = errors.New("the object storage is not complete")

// checkStorage checks the object directory dir and, recursively, its
// alternates.
func checkStorage(dir string) error {
	return checkObjects(dir, 0, map[string]bool{})
}

func checkObjects(dir string, depth int, seen map[string]bool) error {
	key := Canonical(dir)
	if seen[key] {
		return nil
	}
	seen[key] = true
	// 1. Every directory under objects/ can be listed, and every entry in
	// it can be looked at.
	if err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error { //nolint:gosec // the object directories git reads, alternates included; only read
		if err != nil {
			return err
		}
		_, err = d.Info()
		return err
	}); err != nil {
		return errStorage
	}
	// 2 to 4. The packs.
	if err := checkPacks(filepath.Join(dir, "pack")); err != nil {
		return err
	}
	// 5. The alternates, as deep as git reads them.
	alts, err := os.ReadFile(filepath.Join(dir, "info", "alternates")) //nolint:gosec // as above
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errStorage
	}
	for _, line := range strings.Split(string(alts), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if depth >= maxAlternateDepth || strings.HasPrefix(line, `"`) {
			return errStorage // nested deeper than git reads, or a quoted path
		}
		alt := filepath.FromSlash(line)
		if !filepath.IsAbs(alt) {
			alt = filepath.Join(dir, alt)
		}
		if fi, err := os.Stat(alt); err != nil || !fi.IsDir() { //nolint:gosec // an alternate git reads too
			return errStorage
		}
		if err := checkObjects(alt, depth+1, seen); err != nil {
			return err
		}
	}
	return nil
}

// checkPacks checks objects/pack: only files it recognizes; each .pack
// with its .idx and each .idx with its .pack (B2-R2); each .idx's trailing
// checksum right, and its pack's trailer the pack checksum it records.
func checkPacks(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errStorage
	}
	exts := map[string]map[string]bool{} // by pack hash
	for _, e := range entries {
		name := e.Name()
		if name == "multi-pack-index.d" && e.IsDir() {
			if err := checkMidxChain(filepath.Join(dir, name)); err != nil {
				return err
			}
			continue
		}
		if e.IsDir() || !e.Type().IsRegular() {
			return errStorage
		}
		if midxFile.MatchString(name) {
			continue // not read: the scan's reads turn the multi-pack-index off
		}
		m := packFile.FindStringSubmatch(name)
		if m == nil {
			return errStorage // tmp_pack_*, tmp_idx_*, ...: incomplete or being written
		}
		if exts[m[1]] == nil {
			exts[m[1]] = map[string]bool{}
		}
		exts[m[1]][m[2]] = true
	}
	for hash, has := range exts {
		if !has["pack"] || !has["idx"] {
			return errStorage // a companion without its pack, or half a pair
		}
		if err := checkPack(filepath.Join(dir, "pack-"+hash)); err != nil {
			return err
		}
	}
	return nil
}

// checkMidxChain accepts an incremental multi-pack-index's directory of
// recognized files only.
func checkMidxChain(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return errStorage
	}
	for _, e := range entries {
		if !e.Type().IsRegular() || !midxChainFile.MatchString(e.Name()) {
			return errStorage
		}
	}
	return nil
}

// idxMinSize is the smallest pack index: a version 1 index of no object
// (the fan-out table and two checksums); version 2 adds an 8-byte header.
const idxMinSize = 256*4 + 2*indexHashSize

// checkPack checks one pack's .idx and .pack (base is the path without the
// extension).
func checkPack(base string) error {
	f, err := os.Open(base + ".idx")
	if err != nil {
		return errStorage
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil || fi.Size() < idxMinSize {
		return errStorage
	}
	size := fi.Size()
	magic := make([]byte, 4)
	if _, err := f.ReadAt(magic, 0); err != nil || string(magic) == "ÿtOc" && size < idxMinSize+8 {
		return errStorage
	}
	// The index ends with its pack's checksum and its own, which covers
	// everything before it.
	h := sha1.New() //nolint:gosec // see the import
	if _, err := io.CopyN(h, f, size-indexHashSize); err != nil {
		return errStorage
	}
	sums := make([]byte, 2*indexHashSize)
	if _, err := f.ReadAt(sums, size-2*indexHashSize); err != nil || !bytes.Equal(h.Sum(nil), sums[indexHashSize:]) {
		return errStorage
	}
	p, err := os.Open(base + ".pack")
	if err != nil {
		return errStorage
	}
	defer func() { _ = p.Close() }()
	pi, err := p.Stat()
	if err != nil || pi.Size() < 12+indexHashSize {
		return errStorage
	}
	head := make([]byte, 4)
	trailer := make([]byte, indexHashSize)
	if _, err := p.ReadAt(head, 0); err != nil || string(head) != "PACK" {
		return errStorage
	}
	if _, err := p.ReadAt(trailer, pi.Size()-indexHashSize); err != nil || !bytes.Equal(trailer, sums[:indexHashSize]) {
		return errStorage
	}
	return nil
}
