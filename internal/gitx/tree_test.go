package gitx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// treeRepo builds the tree of WP2.3 spec §4.2's experiments by hand, so
// that it holds what a worktree cannot (a symbolic link on Windows, a
// gitlink to a commit that does not exist, paths with * and ?): every
// entry is written with update-index --cacheinfo, and the commit with
// commit-tree. It returns the repository, the commit and the object IDs by
// path.
func treeRepo(t *testing.T) (*testutil.GitRepo, string, map[string]string) {
	t.Helper()
	repo := testutil.NewGitRepo(t)
	blob := func(content string) string {
		f := filepath.Join(t.TempDir(), "blob")
		if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return repo.Git("hash-object", "-w", f)
	}
	a, x, y, target := blob("a\n"), blob("x\n"), blob("y\n"), blob("a\n")
	entries := []struct{ mode, oid, path string }{
		{"100644", a, "a"},
		{"100755", blob("#!/bin/sh\n"), "exec"},
		{"100644", x, "dir/x"},
		{"100644", y, "dir/sub/y"},
		{"120000", target, "link"},
		{"160000", "0123456789abcdef0123456789abcdef01234567", "gitlink"},
		{"100644", a, "a b"},
		{"100644", a, "sp ace/f.txt"},
		{"100644", a, "ü/n.txt"},
		{"100644", a, "[b]"},
		{"100644", a, "st*r"},
		{"100644", x, "stXr"},
		{"100644", a, "q?"},
	}
	for _, e := range entries {
		repo.Git("-c", "core.protectNTFS=false", "update-index", "--add", "--cacheinfo", e.mode+","+e.oid+","+e.path)
	}
	tree := repo.Git("write-tree")
	commit := repo.Git("commit-tree", tree, "-m", "tree")
	ids := map[string]string{".": tree}
	for _, p := range []string{"dir", "dir/sub"} {
		ids[p] = repo.Git("rev-parse", commit+":"+p)
	}
	for _, e := range entries {
		ids[e.path] = e.oid
	}
	return repo, commit, ids
}

// WP2.3 spec §4.1 and §4.2 T1 to T17, MVP-EV-10, MVP-EV-17: the entry of
// a path in a commit's tree, or "not found": the path's own entry (a
// directory's, not its content), compared byte for byte and case for case
// (T11 even with core.ignorecase), literally (T8, T9), from the top of the
// tree wherever it runs (T17); the root is a tree with mode 040000 (D3).
func TestTreeEntry(t *testing.T) {
	requireSafeGit(t)
	repo, commit, ids := treeRepo(t)
	sub := mkdir(t, repo.Dir, "dir", "sub")
	for _, c := range []struct {
		name, lookup, mode, typ string
		found                   bool
	}{
		{"T1 a file", "a", "100644", "blob", true},
		{"T2 an executable", "exec", "100755", "blob", true},
		{"T3 a directory", "dir", "040000", "tree", true},
		{"T3 a nested directory", "dir/sub", "040000", "tree", true},
		{"T4 a deep file", "dir/sub/y", "100644", "blob", true},
		{"T5 a symbolic link", "link", "120000", "blob", true},
		{"T6 a gitlink to a commit that is not there", "gitlink", "160000", "commit", true},
		{"T7 a space", "a b", "100644", "blob", true},
		{"T7 a space in a directory", "sp ace/f.txt", "100644", "blob", true},
		{"T7 not ASCII", "ü/n.txt", "100644", "blob", true},
		{"T7 brackets", "[b]", "100644", "blob", true},
		{"T8 a star, literally", "st*r", "100644", "blob", true},
		{"T8 a question mark, literally", "q?", "100644", "blob", true},
		{"T10 missing", "missing", "", "", false},
		{"T10 another case", "A", "", "", false},
		{"T10 another case in a directory", "DIR/x", "", "", false},
		{"T10 through a file", "a/b", "", "", false},
		{"T10 through a file, deeper", "dir/x/y", "", "", false},
		{"T10 inside a gitlink", "gitlink/inner", "", "", false},
		{"T16 the root", ".", "040000", "tree", true},
	} {
		for _, dir := range []string{repo.Dir, sub} { // T17
			e, found, err := TreeEntry(ctx, dir, commit, c.lookup)
			want := Entry{Mode: c.mode, Type: c.typ, OID: ids[c.lookup]}
			if !c.found {
				want = Entry{}
			}
			if err != nil || found != c.found || e != want {
				t.Errorf("%s (in %s): TreeEntry(%q) = %+v, %v, %v; want %+v, %v", c.name, filepath.Base(dir), c.lookup, e, found, err, want, c.found)
			}
		}
	}
	// T5: a link's object is a blob like a file with the same content.
	if ids["link"] != ids["a"] {
		t.Errorf("the link's blob %s, the file's %s", ids["link"], ids["a"])
	}
	// T11: core.ignorecase does not make the tree lookup ignore case.
	repo.Git("config", "core.ignorecase", "true")
	if _, found, err := TreeEntry(ctx, repo.Dir, commit, "A"); err != nil || found {
		t.Errorf("T11: TreeEntry(A) with core.ignorecase = %v, %v", found, err)
	}
	// T8: st*r does not match stXr, which has another object.
	if e, _, _ := TreeEntry(ctx, repo.Dir, commit, "st*r"); e.OID == ids["stXr"] {
		t.Error("T8: st*r matched stXr")
	}
}

// T12 to T15: the lookup path must be normalized first (a trailing slash,
// "./" and "." would list a directory's content, T12 to T14); TreeEntry
// refuses one that is not, as a mistake of its caller.
func TestTreeEntryNeedsANormalizedPath(t *testing.T) {
	repo, commit, _ := treeRepo(t)
	for _, p := range []string{"", "dir/", "./a", "a/../a", "/a", "..", "../a", `dir\x`, "dir//x"} {
		if _, _, err := TreeEntry(ctx, repo.Dir, commit, p); apperr.ExitCode(err) != apperr.ExitInternal {
			t.Errorf("TreeEntry(%q) = %v; want an internal error", p, err)
		}
	}
	for _, c := range []string{"HEAD", "--all", strings.ToUpper(commit), commit[:39]} {
		if _, _, err := TreeEntry(ctx, repo.Dir, c, "a"); apperr.ExitCode(err) != apperr.ExitInternal {
			t.Errorf("TreeEntry(commit %q) = %v; want an internal error", c, err)
		}
	}
}

// T18 to T23 (L1 and L3): an object the path needs that cannot be read is
// an environment error (exit 6), never "not found"; a missing leaf (the
// path's own object) is still listed, since only its entry is read.
func TestTreeEntryWithMissingObjects(t *testing.T) {
	requireSafeGit(t)
	for _, c := range []struct {
		name, missing, lookup string
		listed                bool
	}{
		{"T18 the tree of dir/sub, for dir/sub/y", "dir/sub", "dir/sub/y", false},
		{"T19 the tree of dir/sub, for dir/sub itself", "dir/sub", "dir/sub", true},
		{"T20 the blob of a", "a", "a", true},
		{"T21 the tree of dir, for dir/x", "dir", "dir/x", false},
		{"T21 the tree of dir, for dir", "dir", "dir", true},
		{"T22 the root tree, for a", ".", "a", false},
		{"T22 the root tree, for the root", ".", ".", false},
	} {
		repo, commit, ids := treeRepo(t)
		removeObject(t, repo, ids[c.missing])
		e, found, err := TreeEntry(ctx, repo.Dir, commit, c.lookup)
		switch {
		case c.listed && (err != nil || !found || e.OID != ids[c.lookup]):
			t.Errorf("%s: TreeEntry = %+v, %v, %v; want it listed", c.name, e, found, err)
		case !c.listed && (apperr.ExitCode(err) != apperr.ExitEnvironment || found):
			t.Errorf("%s: TreeEntry = %+v, %v, %v; want an environment error", c.name, e, found, err)
		}
	}
	// T23: the commit missing, or an ID that is a blob or names nothing.
	repo, commit, ids := treeRepo(t)
	for name, c := range map[string]string{"a blob": ids["a"], "nothing": strings.Repeat("e", 40)} {
		if _, found, err := TreeEntry(ctx, repo.Dir, c, "a"); apperr.ExitCode(err) != apperr.ExitEnvironment || found {
			t.Errorf("T23 %s: %v, %v; want an environment error", name, found, err)
		}
	}
	removeObject(t, repo, commit)
	if _, found, err := TreeEntry(ctx, repo.Dir, commit, "a"); apperr.ExitCode(err) != apperr.ExitEnvironment || found {
		t.Errorf("T23 the commit missing: %v, %v; want an environment error", found, err)
	}
}

// D8: ls-tree's answer is trusted only when it is nothing or exactly the
// entry asked for.
func TestParseTreeEntry(t *testing.T) {
	oid := strings.Repeat("a", 40)
	for _, c := range []struct {
		name, out string
		found     bool
		ok        bool
	}{
		{"nothing", "", false, true},
		{"the entry", "100644 blob " + oid + "\tdir/a\x00", true, true},
		{"two entries", "100644 blob " + oid + "\tdir/a\x00100644 blob " + oid + "\tdir/b\x00", false, false},
		{"another path", "100644 blob " + oid + "\tdir/b\x00", false, false},
		{"no NUL", "100644 blob " + oid + "\tdir/a", false, false},
		{"no tab", "100644 blob " + oid + " dir/a\x00", false, false},
		{"a short ID", "100644 blob " + oid[1:] + "\tdir/a\x00", false, false},
		{"another type", "100644 tag " + oid + "\tdir/a\x00", false, false},
		{"a bad mode", "10064 blob " + oid + "\tdir/a\x00", false, false},
		{"an extra field", "100644 blob " + oid + " 12\tdir/a\x00", false, false},
	} {
		e, found, err := parseTreeEntry([]byte(c.out), "dir/a")
		switch {
		case c.ok && (err != nil || found != c.found):
			t.Errorf("%s: %+v, %v, %v", c.name, e, found, err)
		case !c.ok && (apperr.ExitCode(err) != apperr.ExitEnvironment || found):
			t.Errorf("%s: %+v, %v, %v; want an environment error", c.name, e, found, err)
		}
	}
}
