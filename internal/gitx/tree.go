package gitx

import (
	"bytes"
	"context"
	"path"
	"regexp"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// Entry is a tree entry's metadata (data-model §4.11.2): its mode, type
// and object ID as git gives them.
type Entry struct{ Mode, Type, OID string }

// commitID matches a full SHA-1 commit or object ID, the only form the
// WP2.3 reads accept (TD-26), so that no argument reaches git as an option.
var commitID = regexp.MustCompile(`^[0-9a-f]{40}$`)

var (
	treeMode = regexp.MustCompile(`^[0-7]{6}$`)
	treeType = map[string]bool{"blob": true, "tree": true, "commit": true}
)

// TreeEntry reads the entry of a lookup path (V-P04 normalized: forward
// slashes, cleaned, inside the repository) in the tree of a full commit ID
// (WP2.3 spec §4.1): "." is the root tree, recorded as mode 040000 (D3).
// found is false when the tree has no such path (L2); an error is an
// object the path needs that cannot be read (L1, exit 6), or an answer
// that is not exactly the one entry asked for (D8). Each path is its own
// literal ls-tree, run from the top of the tree; nothing (LFS, a
// submodule, a filter) runs.
func TreeEntry(ctx context.Context, worktree, commit, lookup string) (e Entry, found bool, err error) {
	if !commitID.MatchString(commit) {
		return Entry{}, false, apperr.New(apperr.Internal, "TreeEntry needs a full commit ID")
	}
	if !normalizedLookup(lookup) {
		return Entry{}, false, apperr.New(apperr.Internal, "TreeEntry needs a normalized lookup path")
	}
	if lookup == "." {
		out, err := call{label: "rev-parse", dir: worktree, objects: true,
			args: []string{"rev-parse", "--verify", commit + "^{tree}"}}.must(ctx)
		if err != nil {
			return Entry{}, false, err
		}
		tree := strings.TrimSpace(string(out))
		if !commitID.MatchString(tree) {
			return Entry{}, false, apperr.New(apperr.Environment, "git rev-parse printed something other than a tree ID")
		}
		return Entry{Mode: "040000", Type: "tree", OID: tree}, true, nil
	}
	out, err := call{label: "ls-tree", dir: worktree, objects: true,
		args: []string{"--literal-pathspecs", "ls-tree", "-z", "--full-tree", commit, "--", lookup}}.must(ctx)
	if err != nil {
		return Entry{}, false, err
	}
	return parseTreeEntry(out, lookup)
}

// normalizedLookup reports whether p is a lookup path as V-P04 makes it.
func normalizedLookup(p string) bool {
	return p != "" && !strings.Contains(p, `\`) && path.Clean(p) == p && !path.IsAbs(p) &&
		p != ".." && !strings.HasPrefix(p, "../")
}

// parseTreeEntry reads ls-tree -z's answer for lookup: nothing (no such
// path), or exactly one record "<mode> <type> <oid>\t<path>\x00" whose path
// is lookup. Anything else cannot be trusted (D8).
func parseTreeEntry(out []byte, lookup string) (Entry, bool, error) {
	if len(out) == 0 {
		return Entry{}, false, nil
	}
	untrusted := apperr.New(apperr.Environment, "git ls-tree gave an answer that is not the one entry asked for")
	record, rest, ok := bytes.Cut(out, []byte{0})
	if !ok || len(rest) != 0 {
		return Entry{}, false, untrusted
	}
	meta, p, ok := strings.Cut(string(record), "\t")
	fields := strings.Split(meta, " ")
	if !ok || p != lookup || len(fields) != 3 || !treeMode.MatchString(fields[0]) || !treeType[fields[1]] || !commitID.MatchString(fields[2]) {
		return Entry{}, false, untrusted
	}
	return Entry{Mode: fields[0], Type: fields[1], OID: fields[2]}, true, nil
}
