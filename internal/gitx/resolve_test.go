package gitx

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha1" //nolint:gosec // git's own object IDs, to build a prefix collision
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// absentPrefix returns a seven-digit prefix that names no object of repo,
// as git itself lists them.
func absentPrefix(t *testing.T, repo *testutil.GitRepo) string {
	t.Helper()
	for _, p := range []string{"fedcba9", "0123456", "abcdef0", "9876543"} {
		if repo.Git("rev-parse", "--disambiguate="+p) == "" {
			return p
		}
	}
	t.Fatal("every candidate prefix names an object")
	return ""
}

// WP2.4 spec §7, E4: a well-formed prefix resolves (true) only when it
// names exactly one commit the repository can read; one that names no
// object, a tree or a blob, or a commit gc reclaimed, does not (false).
// The full ID and a longer prefix of a commit resolve alike, a prefix
// given twice is answered once, and nothing in the repository changes.
// A prefix that is not 7 to 40 lower-case hexadecimal digits is the
// caller's mistake (claims.v1 refuses it first).
func TestResolveCommits(t *testing.T) {
	requireSafeGit(t)
	repo, shas := chain(t, 2)
	gone := reclaim(t, repo)
	tree := repo.Git("rev-parse", shas[1]+"^{tree}")
	blob := repo.Git("rev-parse", shas[1]+":b.txt")
	absent := absentPrefix(t, repo)
	gitDir := filepath.Join(repo.Dir, ".git")
	before := testutil.SettledSnapshot(t, gitDir)
	asked := []string{shas[1], shas[1][:7], shas[0][:12], shas[1][:7], tree, blob[:9], gone, gone[:8], absent}
	got, err := ResolveCommits(ctx, testScratch(t), repo.Dir, asked)
	want := map[string]bool{shas[1]: true, shas[1][:7]: true, shas[0][:12]: true, tree: false, blob[:9]: false, gone: false, gone[:8]: false, absent: false}
	if err != nil || !resolvedEqual(got, want) {
		t.Errorf("ResolveCommits = %v, %v; want %v", got, err, want)
	}
	testutil.AssertTreeUnchanged(t, gitDir, before)
	if got, err := ResolveCommits(ctx, testScratch(t), repo.Dir, nil); err != nil || len(got) != 0 {
		t.Errorf("ResolveCommits(none) = %v, %v", got, err)
	}
	for _, p := range []string{"HEAD", strings.ToUpper(shas[0][:7]), shas[0][:6], shas[0] + "0", "--all", shas[0][:7] + "^{tree}", ""} {
		if got, err := ResolveCommits(ctx, testScratch(t), repo.Dir, []string{p}); apperr.ExitCode(err) != apperr.ExitInternal {
			t.Errorf("ResolveCommits(%q) = %v, %v; want an internal error", p, got, err)
		}
	}
}

func resolvedEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// objectID is git's SHA-1 object ID of an object of type typ.
func objectIDOf(typ, content string) string {
	h := sha1.New() //nolint:gosec // git's object ID
	fmt.Fprintf(h, "%s %d\x00%s", typ, len(content), content)
	return hex.EncodeToString(h.Sum(nil))
}

const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

func collisionCommit(i int) string {
	return fmt.Sprintf("tree %s\nauthor A <a@example.com> 1700000000 +0000\ncommitter A <a@example.com> 1700000000 +0000\n\ncollision %d\n", emptyTree, i)
}

// writeObject writes content as an object of type typ with git itself and
// checks the ID it gives.
func writeObject(t *testing.T, repo *testutil.GitRepo, typ, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "object")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	id := repo.Git("hash-object", "-t", typ, "-w", p)
	if want := objectIDOf(typ, content); id != want {
		t.Fatalf("git wrote %s as %s, not %s", typ, id, want)
	}
	return id
}

// collisions writes into repo a commit and a blob whose IDs share their
// first seven digits, and two commits that do too: real ambiguous
// prefixes, found by a birthday search over contents that do not depend
// on the run.
func collisions(t *testing.T, repo *testutil.GitRepo) (commitWithBlob, blob, commitA, commitB string) {
	t.Helper()
	const n = 1 << 16
	commits := map[string]int{}
	pairA, pairB := -1, -1
	for i := range n {
		p := objectIDOf("commit", collisionCommit(i))[:7]
		if j, ok := commits[p]; ok && pairA < 0 {
			pairA, pairB = j, i
		}
		commits[p] = i
	}
	withBlob, blobIndex := -1, -1
	for j := 0; j < n && withBlob < 0; j++ {
		p := objectIDOf("blob", fmt.Sprintf("blob %d\n", j))[:7]
		if i, ok := commits[p]; ok && pairA >= 0 && p != objectIDOf("commit", collisionCommit(pairA))[:7] {
			withBlob, blobIndex = i, j
		}
	}
	if pairA < 0 || withBlob < 0 {
		t.Fatal("the birthday search found no collision")
	}
	commitWithBlob = writeObject(t, repo, "commit", collisionCommit(withBlob))
	blob = writeObject(t, repo, "blob", fmt.Sprintf("blob %d\n", blobIndex))
	commitA = writeObject(t, repo, "commit", collisionCommit(pairA))
	commitB = writeObject(t, repo, "commit", collisionCommit(pairB))
	return commitWithBlob, blob, commitA, commitB
}

// distinct is the shortest prefix of a that b does not share.
func distinct(a, b string) string {
	k := 0
	for k < len(a) && a[k] == b[k] {
		k++
	}
	return a[:k+1]
}

// E4: a truly ambiguous prefix, a commit and a blob or two commits behind
// it, does not resolve (false, not an error); a longer prefix that tells
// the commit apart, or its full ID, does.
func TestResolveCommitsAmbiguous(t *testing.T) {
	requireSafeGit(t)
	repo, _ := chain(t, 1)
	commit, blob, a, b := collisions(t, repo)
	if commit[:7] != blob[:7] || a[:7] != b[:7] {
		t.Fatalf("no collision: %s %s, %s %s", commit, blob, a, b)
	}
	for _, p := range []string{commit[:7], a[:7]} {
		if listed := strings.Fields(repo.Git("rev-parse", "--disambiguate="+p)); len(listed) < 2 {
			t.Fatalf("git lists %v for %s", listed, p)
		}
	}
	asked := []string{commit[:7], distinct(commit, blob), commit, blob, a[:7], distinct(a, b), distinct(b, a), b}
	got, err := ResolveCommits(ctx, testScratch(t), repo.Dir, asked)
	want := map[string]bool{commit[:7]: false, distinct(commit, blob): true, commit: true, blob: false,
		a[:7]: false, distinct(a, b): true, distinct(b, a): true, b: true}
	if err != nil || !resolvedEqual(got, want) {
		t.Errorf("ResolveCommits = %v, %v; want %v", got, err, want)
	}
}

// E4: an object that cannot be read, storage that is damaged or
// incomplete, or one that cannot be checked, is no answer at all: an
// environment error (exit 6), never false, whether the prefix names a
// commit or nothing. git's own message is not in the error.
func TestResolveCommitsChecksTheStorage(t *testing.T) {
	requireSafeGit(t)
	type setup func(t *testing.T, repo *testutil.GitRepo, shas []string) (dir, prefix string)
	damage := func(content []byte) setup {
		return func(t *testing.T, repo *testutil.GitRepo, shas []string) (string, string) {
			p := loose(repo, shas[0])
			writable(t, p)
			if err := os.WriteFile(p, content, 0o644); err != nil {
				t.Fatal(err)
			}
			return repo.Dir, shas[0][:7]
		}
	}
	inPack := func(change func(t *testing.T, pack string)) setup {
		return func(t *testing.T, repo *testutil.GitRepo, shas []string) (string, string) {
			change(t, packOnly(t, repo))
			return repo.Dir, shas[0][:10]
		}
	}
	for _, c := range []struct {
		name  string
		setup setup
		ok    bool
	}{
		{"a damaged loose commit", damage([]byte("not zlib")), false},
		{"an empty loose commit", damage(nil), false},
		{"a loose commit whose header reads and whose content does not", damage(func() []byte {
			var b bytes.Buffer
			z := zlib.NewWriter(&b)
			_, _ = z.Write([]byte("commit 500\x00tree " + emptyTree + "\n"))
			_ = z.Close()
			return b.Bytes()
		}()), false},
		{"a damaged loose blob behind an ambiguous prefix", func(t *testing.T, repo *testutil.GitRepo, _ []string) (string, string) {
			commit, blob, _, _ := collisions(t, repo)
			p := loose(repo, blob)
			writable(t, p)
			if err := os.WriteFile(p, []byte("not zlib"), 0o644); err != nil {
				t.Fatal(err)
			}
			return repo.Dir, commit[:7]
		}, false},
		{"a .pack that lost its .idx", inPack(func(t *testing.T, pack string) {
			if err := os.Rename(pack+".idx", filepath.Join(t.TempDir(), "away.idx")); err != nil {
				t.Fatal(err)
			}
		}), false},
		{"a truncated .idx", inPack(func(t *testing.T, pack string) {
			writable(t, pack+".idx")
			if err := os.Truncate(pack+".idx", 1100); err != nil {
				t.Fatal(err)
			}
		}), false},
		{"a .pack whose trailer the .idx does not record", inPack(func(t *testing.T, pack string) {
			writable(t, pack+".pack")
			b, err := os.ReadFile(pack + ".pack")
			if err != nil {
				t.Fatal(err)
			}
			b[len(b)-1] ^= 0x01
			if err := os.WriteFile(pack+".pack", b, 0o644); err != nil {
				t.Fatal(err)
			}
		}), false},
		{"a leftover tmp_pack file", inPack(func(t *testing.T, pack string) {
			if err := os.WriteFile(filepath.Join(filepath.Dir(pack), "tmp_pack_XYZ"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}), false},
		{"an alternate that does not exist", func(t *testing.T, repo *testutil.GitRepo, shas []string) (string, string) {
			alt := filepath.Join(objectsDir(repo), "info", "alternates")
			if err := os.WriteFile(alt, []byte(filepath.ToSlash(filepath.Join(t.TempDir(), "none"))+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return repo.Dir, shas[0][:7]
		}, false},
		{"a .pack that lost its .idx in an alternate, the prefix naming nothing", func(t *testing.T, repo *testutil.GitRepo, _ []string) (string, string) {
			other := testutil.NewGitRepo(t)
			alt := filepath.Join(other.Dir, ".git", "objects", "info", "alternates")
			if err := os.WriteFile(alt, []byte(filepath.ToSlash(objectsDir(repo))+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			pack := packOnly(t, repo)
			if err := os.Rename(pack+".idx", filepath.Join(t.TempDir(), "away.idx")); err != nil {
				t.Fatal(err)
			}
			return other.Dir, absentPrefix(t, other)
		}, false},
		{"the objects of an alternate, read whole", func(t *testing.T, repo *testutil.GitRepo, shas []string) (string, string) {
			other := testutil.NewGitRepo(t)
			alt := filepath.Join(other.Dir, ".git", "objects", "info", "alternates")
			if err := os.WriteFile(alt, []byte(filepath.ToSlash(objectsDir(repo))+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return other.Dir, shas[0][:7]
		}, true},
		{".keep, .bitmap and .rev beside their pack", func(t *testing.T, repo *testutil.GitRepo, shas []string) (string, string) {
			repo.Git("-c", "pack.writeReverseIndex=true", "repack", "-adqb")
			repo.Git("prune-packed")
			packs, _ := filepath.Glob(filepath.Join(objectsDir(repo), "pack", "pack-*.pack"))
			if len(packs) != 1 {
				t.Fatalf("packs: %v", packs)
			}
			if err := os.WriteFile(strings.TrimSuffix(packs[0], ".pack")+".keep", nil, 0o644); err != nil {
				t.Fatal(err)
			}
			return repo.Dir, shas[0][:7]
		}, true},
		{"a multi-pack-index beside the packs", func(_ *testing.T, repo *testutil.GitRepo, shas []string) (string, string) {
			repo.Git("repack", "-adq")
			repo.Git("multi-pack-index", "write")
			return repo.Dir, shas[0][:7]
		}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, shas := chain(t, 2)
			dir, prefix := c.setup(t, repo, shas)
			gitDir := filepath.Join(dir, ".git")
			before := testutil.SettledSnapshot(t, gitDir)
			objects := testutil.SettledSnapshot(t, objectsDir(repo))
			got, err := ResolveCommits(ctx, testScratch(t), dir, []string{prefix})
			switch {
			case c.ok && (err != nil || !got[prefix]):
				t.Errorf("ResolveCommits = %v, %v; want resolved", got, err)
			case !c.ok && (apperr.ExitCode(err) != apperr.ExitEnvironment || got != nil):
				t.Errorf("ResolveCommits = %v, %v; want an environment error and no answer", got, err)
			case !c.ok && (strings.Contains(err.Error(), "fatal") || strings.Contains(err.Error(), "error:") || strings.Contains(err.Error(), dir)):
				t.Errorf("the error quotes git or the repository: %v", err)
			}
			testutil.AssertTreeUnchanged(t, gitDir, before)
			testutil.AssertTreeUnchanged(t, objectsDir(repo), objects)
		})
	}
}

// E4 on Linux and macOS, not as root: a loose object, or its directory,
// that cannot be read is an environment error, not a prefix naming
// nothing.
func TestResolveCommitsWithUnreadableStorage(t *testing.T) {
	requireSafeGit(t)
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("file modes do not keep root or Windows from reading")
	}
	for _, c := range []struct {
		name string
		path func(r *testutil.GitRepo, sha string) string
	}{
		{"a loose object's directory unreadable", func(r *testutil.GitRepo, sha string) string { return filepath.Dir(loose(r, sha)) }},
		{"a loose object unreadable", func(r *testutil.GitRepo, sha string) string { return loose(r, sha) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, shas := chain(t, 2)
			p := c.path(repo, shas[0])
			info, err := os.Stat(p)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(p, 0); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(p, info.Mode()) })
			if got, err := ResolveCommits(ctx, testScratch(t), repo.Dir, []string{shas[0][:7]}); apperr.ExitCode(err) != apperr.ExitEnvironment || got != nil {
				t.Errorf("ResolveCommits = %v, %v; want an environment error", got, err)
			}
		})
	}
}

// PD-26: with a git that would fetch a missing object, no prefix is
// resolved: the reads are refused before they run (exit 6), and no git
// command reads an object of the repository.
func TestResolveCommitsNeedsASafeGit(t *testing.T) {
	requireSafeGit(t)
	repo, shas := chain(t, 2)
	dir, env, calls := testutil.PassthroughGit(t, testutil.PassWithoutNoLazyFetch)
	useGit(t, dir, env)
	got, err := ResolveCommits(ctx, testScratch(t), repo.Dir, []string{shas[0][:7]})
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "GIT_NO_LAZY_FETCH") || got != nil {
		t.Errorf("ResolveCommits = %v, %v; want the object reads refused", got, err)
	}
	for _, c := range calls() {
		for _, sub := range []string{"--disambiguate=" + shas[0][:7], "--batch-check", "commit"} {
			if slices.Contains(c.Args, sub) && !slices.ContainsFunc(c.Args, func(a string) bool { return strings.HasPrefix(a, "--git-dir=") }) {
				t.Errorf("%s ran in the repository: %q", sub, c.Args)
			}
		}
	}
}

// PD-26 with the real git on PATH (SPECMETRY_TEST_GIT_OBJECTS=unsafe and
// a git that would fetch a missing object, git 2.32.0): refused, exit 6,
// and nothing in the repository changes.
func TestResolveCommitsWithAGitThatFetches(t *testing.T) {
	if os.Getenv(gitObjectsEnv) != "unsafe" {
		t.Skip("needs " + gitObjectsEnv + "=unsafe and a git that fetches")
	}
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("a.txt", "a\n")
	first := repo.Commit("one")
	gitDir := filepath.Join(repo.Dir, ".git")
	before := testutil.SettledSnapshot(t, gitDir)
	if got, err := ResolveCommits(ctx, testScratch(t), repo.Dir, []string{first[:7]}); apperr.ExitCode(err) != apperr.ExitEnvironment ||
		!strings.Contains(err.Error(), "GIT_NO_LAZY_FETCH") || got != nil {
		t.Errorf("ResolveCommits = %v, %v; want the object reads refused", got, err)
	}
	testutil.AssertTreeUnchanged(t, gitDir, before)
}

// A cancelled read gives no answer: an error, never a map of false.
func TestResolveCommitsCancelled(t *testing.T) {
	requireSafeGit(t)
	repo, shas := chain(t, 1)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if got, err := ResolveCommits(cancelled, testScratch(t), repo.Dir, []string{shas[0][:7]}); err == nil || got != nil {
		t.Errorf("ResolveCommits = %v, %v; want an error", got, err)
	}
}
