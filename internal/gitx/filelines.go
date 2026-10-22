package gitx

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// FileLines tells, for each of paths (repository paths from the top, with
// forward slashes), whether it is a regular file of the tree of commit in
// the repository of worktree and how many lines it has: its line breaks,
// and one more for a last line without one. A path that is not one (not
// in the tree; a directory, a symbolic link or a submodule) is not in the
// map. This is what code references are checked against (V-P08, WP2.4
// spec §7).
//
// Every object the answer rests on is read whole and must be the object
// its ID names (readVerified; review 068 B3): the commit, each tree on the
// way to a path, a level at a time, and each file's blob, once however
// many paths name it. The trees are parsed, and the lines counted, from
// that same checked read, never from another; nothing of a blob is kept.
// Only those objects are read: no other tree, blob or history. An object
// that does not read whole as its ID, or as the type its tree gives it,
// or that cannot be read, or any stderr output or failed status, is an
// environment error (exit 6) with no answer for any path. The reads are
// object reads (PD-26: a git shown never to fetch an object); a commit
// that is not a full commit ID is an internal error, and never reaches git.
func FileLines(ctx context.Context, worktree, commit string, paths []string) (map[string]int, error) {
	if !commitID.MatchString(commit) {
		return nil, apperr.New(apperr.Internal, "FileLines needs a full commit ID")
	}
	unreliable := apperr.New(apperr.Environment,
		"a code reference cannot be checked reliably: an object of the commit's tree does not read as the object its ID names, or cannot be read; nothing was written; run git fsck in this directory")
	run := func(c call) (result, error) {
		c.objects = true
		return c.run(ctx)
	}
	read := func(ids []string, want string, sink func(id string) io.Writer) error {
		types, ok, err := readVerified(run, worktree, ids, func(id, _ string) io.Writer { return sink(id) })
		if err != nil {
			return err
		}
		if !ok {
			return unreliable
		}
		for _, id := range ids {
			if types[id] != want {
				return unreliable
			}
		}
		return nil
	}

	// 1. The commit, for its tree.
	var body bytes.Buffer
	if err := read([]string{commit}, "commit", func(string) io.Writer { return &body }); err != nil {
		return nil, err
	}
	first, _, _ := strings.Cut(body.String(), "\n")
	root, ok := strings.CutPrefix(first, "tree ")
	if !ok || !commitID.MatchString(root) {
		return nil, unreliable
	}

	// 2. The trees on the way to each path, a level at a time.
	type walk struct {
		path string
		rest []string
		tree string
	}
	var walks []walk
	for _, p := range paths {
		parts := strings.Split(p, "/")
		if slices.ContainsFunc(parts, func(s string) bool { return s == "" || s == "." || s == ".." }) {
			continue // ls-tree never lists such a path: no file
		}
		walks = append(walks, walk{p, parts, root})
	}
	trees := map[string]map[string]treeEntry{}
	blobs := map[string]string{}
	for len(walks) > 0 {
		var need []string
		for _, w := range walks {
			if _, done := trees[w.tree]; !done && !slices.Contains(need, w.tree) {
				need = append(need, w.tree)
			}
		}
		bodies := map[string]*bytes.Buffer{}
		if err := read(need, "tree", func(id string) io.Writer { b := &bytes.Buffer{}; bodies[id] = b; return b }); err != nil {
			return nil, err
		}
		for _, id := range need {
			entries, ok := parseTree(bodies[id].Bytes())
			if !ok {
				return nil, unreliable
			}
			trees[id] = entries
		}
		var next []walk
		for _, w := range walks {
			e, found := trees[w.tree][w.rest[0]]
			switch {
			case !found:
			case len(w.rest) > 1 && e.kind == entryTree:
				next = append(next, walk{w.path, w.rest[1:], e.id})
			case len(w.rest) == 1 && e.kind == entryFile:
				blobs[w.path] = e.id
			}
		}
		walks = next
	}

	// 3. The files' blobs, counted as they stream past.
	var ids []string
	for _, id := range blobs {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	counters := map[string]*lineCounter{}
	if err := read(ids, "blob", func(id string) io.Writer { c := &lineCounter{}; counters[id] = c; return c }); err != nil {
		return nil, err
	}
	out := map[string]int{}
	for p, id := range blobs {
		out[p] = counters[id].lines()
	}
	return out, nil
}

// The kinds of tree entry, by the file type bits of the mode, as git
// canonicalizes them (tree-walk's canon_mode): a regular file (100644 or
// 100755 once canonical), a tree, or anything else (a symbolic link, a
// submodule).
type entryKind int

const (
	entryOther entryKind = iota
	entryFile
	entryTree
)

type treeEntry struct {
	kind entryKind
	id   string
}

// parseTree parses a tree object's content: entries "<mode> <name>\x00"
// and a 20-byte object ID (TD-26: SHA-1). Content that is not a sequence of
// such entries, or names an entry twice, cannot be relied on.
func parseTree(b []byte) (map[string]treeEntry, bool) {
	entries := map[string]treeEntry{}
	for len(b) > 0 {
		sp := bytes.IndexByte(b, ' ')
		if sp <= 0 {
			return nil, false
		}
		mode, err := strconv.ParseUint(string(b[:sp]), 8, 32)
		if err != nil {
			return nil, false
		}
		b = b[sp+1:]
		nul := bytes.IndexByte(b, 0)
		if nul <= 0 || len(b) < nul+1+20 {
			return nil, false
		}
		name := string(b[:nul])
		if strings.Contains(name, "/") {
			return nil, false
		}
		if _, twice := entries[name]; twice {
			return nil, false
		}
		e := treeEntry{id: hex.EncodeToString(b[nul+1 : nul+21])}
		switch mode & 0o170000 {
		case 0o100000:
			e.kind = entryFile
		case 0o040000:
			e.kind = entryTree
		}
		entries[name] = e
		b = b[nul+21:]
	}
	return entries, true
}
