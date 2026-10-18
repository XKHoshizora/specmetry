package gitx

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// chain makes n commits on main, each with its own file, oldest first; the
// last names REQ-001 in a trailer and AC-001-1 in its subject.
func chain(t *testing.T, n int) (*testutil.GitRepo, []string) {
	t.Helper()
	repo := testutil.NewGitRepo(t)
	var shas []string
	for i := range n {
		name := string(rune('a' + i))
		repo.WriteFile(name+".txt", name+"\n")
		msg := "commit " + name
		if i == n-1 {
			msg = "fix AC-001-1\n\nbody\n\nSpecmetry-Ref: REQ-001"
		}
		shas = append(shas, repo.Commit(msg))
	}
	return repo, shas
}

func objectsDir(repo *testutil.GitRepo) string { return filepath.Join(repo.Dir, ".git", "objects") }

func loose(repo *testutil.GitRepo, oid string) string {
	return filepath.Join(objectsDir(repo), oid[:2], oid[2:])
}

// writable makes a file git wrote read-only writable again.
func writable(t *testing.T, p string) {
	t.Helper()
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
}

// packOnly packs every object into one pack, removes the loose copies and
// returns the pack's path without its extension.
func packOnly(t *testing.T, repo *testutil.GitRepo) string {
	t.Helper()
	repo.Git("repack", "-adq")
	repo.Git("prune-packed")
	packs, err := filepath.Glob(filepath.Join(objectsDir(repo), "pack", "pack-*.pack"))
	if err != nil || len(packs) != 1 {
		t.Fatalf("packs: %v, %v", packs, err)
	}
	return strings.TrimSuffix(packs[0], ".pack")
}

// reclaim makes a commit unreachable and prunes it, as a rewrite followed
// by gc does: a truly absent old tip. The commit is made on a branch of its
// own off main and named in no reflog.
func reclaim(t *testing.T, repo *testutil.GitRepo) string {
	t.Helper()
	repo.Git("checkout", "-q", "-b", "gone")
	repo.WriteFile("gone.txt", "gone\n")
	sha := repo.Commit("gone")
	repo.Git("checkout", "-q", "main")
	repo.Git("branch", "-D", "gone")
	repo.Git("reflog", "expire", "--expire=now", "--all")
	repo.Git("gc", "-q", "--prune=now")
	if _, err := os.Stat(loose(repo, sha)); err == nil {
		t.Fatal("gc kept the commit")
	}
	return sha
}

// P3 (WP2.3 spec §5.2 "shallow clones", H1, H2, H6, H7): a scan starts only
// when the common git directory's shallow entry is shown absent, read with
// lstat, not with git's --is-shallow-repository; a shallow file, a
// directory there, or one that cannot be checked is an environment error.
// Nothing is written.
func TestCheckShallow(t *testing.T) {
	repo := committedRepo(t)
	wt := repo.AddWorktree("side")
	gitDir := filepath.Join(repo.Dir, ".git")
	before := testutil.SettledSnapshot(t, gitDir)
	for _, dir := range []string{repo.Dir, wt} {
		if err := CheckShallow(ctx, dir); err != nil {
			t.Errorf("H1 a complete repository: %v", err)
		}
	}
	testutil.AssertTreeUnchanged(t, gitDir, before)
	shallow := filepath.Join(gitDir, "shallow")
	refused := func(name string) {
		t.Helper()
		for _, dir := range []string{repo.Dir, wt} {
			if err := CheckShallow(ctx, dir); apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "shallow") {
				t.Errorf("%s (from %s): %v; want an environment error", name, filepath.Base(dir), err)
			}
		}
	}
	if err := os.WriteFile(shallow, []byte(repo.Git("rev-parse", "HEAD")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	refused("H2 a shallow file")
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		if err := os.Chmod(shallow, 0); err != nil {
			t.Fatal(err)
		}
		// git itself then says it is not shallow.
		if out := repo.Git("rev-parse", "--is-shallow-repository"); out != "false" {
			t.Logf("git says --is-shallow-repository = %s for an unreadable shallow file", out)
		}
		refused("H6 an unreadable shallow file")
		if err := os.Chmod(shallow, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(shallow); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(shallow, 0o755); err != nil {
		t.Fatal(err)
	}
	if out := repo.Git("rev-parse", "--is-shallow-repository"); out != "false" {
		t.Logf("git says --is-shallow-repository = %s for a shallow directory", out)
	}
	refused("H7 a shallow directory")
	if err := os.Remove(shallow); err != nil {
		t.Fatal(err)
	}
	if err := CheckShallow(ctx, repo.Dir); err != nil {
		t.Errorf("made complete again: %v", err)
	}
}

// WP2.3 spec §5.1 (D2): the scanned refs are the local branches that point
// to commits (a branch pointing to another object is left out) and the
// detached HEADs of all worktrees, registered or not, prunable included,
// each named by gitx.Canonical of its path; a worktree on a branch, and a
// tag, are not tips of their own.
func TestTips(t *testing.T) {
	requireSafeGit(t)
	repo, shas := chain(t, 2)
	repo.Git("branch", "feature", shas[0])
	repo.Git("tag", "v1", shas[0])
	blob := repo.Git("rev-parse", shas[0]+":a.txt")
	// git refuses to point a branch at a blob, so the ref file is written.
	if err := os.WriteFile(filepath.Join(repo.Dir, ".git", "refs", "heads", "not-a-commit"), []byte(blob+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	detached := repo.AddWorktree("wt-detached")
	repo.GitIn(detached, "checkout", "-q", "--detach", shas[0])
	repo.AddWorktree("wt-branch")
	gone := repo.AddWorktree("wt-gone")
	repo.GitIn(gone, "checkout", "-q", "--detach", shas[1])
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(repo.Git("worktree", "list", "--porcelain"), "prunable") {
		t.Fatal("the removed worktree is not prunable")
	}
	tips, err := Tips(ctx, repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	branch := repo.Git("rev-parse", "refs/heads/wt-branch")
	want := []Tip{
		{Ref: "refs/heads/feature", Commit: shas[0]},
		{Ref: "refs/heads/main", Commit: shas[1]},
		{Ref: "refs/heads/wt-branch", Commit: branch},
		{Ref: "refs/heads/wt-detached", Commit: shas[1]},
		{Ref: "refs/heads/wt-gone", Commit: shas[1]},
	}
	detachedTips := []Tip{
		{WorktreePath: Canonical(detached), Commit: shas[0]},
		{WorktreePath: Canonical(gone), Commit: shas[1]},
	}
	slices.SortFunc(detachedTips, func(a, b Tip) int { return strings.Compare(a.WorktreePath, b.WorktreePath) })
	want = append(want, detachedTips...)
	if !slices.Equal(tips, want) {
		t.Errorf("Tips =\n%+v\nwant\n%+v", tips, want)
	}
	// The main worktree detached is a tip too.
	repo.Git("checkout", "-q", "--detach", shas[0])
	tips, err = Tips(ctx, repo.Dir)
	if err != nil || !slices.Contains(tips, Tip{WorktreePath: Canonical(repo.Dir), Commit: shas[0]}) {
		t.Errorf("the main worktree detached: %+v, %v", tips, err)
	}
}

// A branch whose commit cannot be read fails the listing: it is never
// left out as "not a commit".
func TestTipsWithAnUnreadableBranch(t *testing.T) {
	requireSafeGit(t)
	repo, shas := chain(t, 2)
	repo.Git("branch", "feature", shas[0])
	repo.Git("checkout", "-q", "--detach", shas[0])
	p := loose(repo, shas[1])
	writable(t, p)
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if tips, err := Tips(ctx, repo.Dir); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("Tips = %+v, %v; want an environment error", tips, err)
	}
}

// WP2.3 spec §5.1 and §13 (the owner's decision of 2026-10-19): a git that
// cannot list worktrees with -z (a usage error, as before 2.36) makes the
// scan an environment error; the list is never read line by line.
func TestTipsNeedsWorktreeListZ(t *testing.T) {
	requireSafeGit(t)
	repo, _ := chain(t, 1)
	dir, env, calls := testutil.PassthroughGit(t, testutil.PassWithoutWorktreeListZ)
	useGit(t, dir, env)
	tips, err := Tips(ctx, repo.Dir)
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "worktree list -z") {
		t.Fatalf("Tips = %+v, %v; want an environment error", tips, err)
	}
	for _, c := range calls() {
		if slices.Contains(c.Args, "worktree") && !slices.Contains(c.Args, "-z") {
			t.Errorf("the worktree list was read without -z: %q", c.Args)
		}
	}
}

// WP2.3 spec §5.2 step 2 (R5 to R9, K1, K11): a commit whose header reads
// is Present; one git says is missing is Absent only with no stderr and a
// storage check that passes; a loose object that is damaged or empty is
// present but unreadable, an object of another type cannot be confirmed:
// both are environment errors.
func TestClassify(t *testing.T) {
	requireSafeGit(t)
	repo, shas := chain(t, 2)
	gone := reclaim(t, repo)
	never := strings.Repeat("e", 40)
	gitDir := filepath.Join(repo.Dir, ".git")
	before := testutil.SettledSnapshot(t, gitDir)
	got, err := Classify(ctx, repo.Dir, []string{shas[0], gone, never, shas[1], shas[0]})
	want := map[string]Presence{shas[0]: Present, shas[1]: Present, gone: Absent, never: Absent}
	if err != nil || !mapsEqual(got, want) {
		t.Errorf("Classify = %v, %v; want %v", got, err, want)
	}
	testutil.AssertTreeUnchanged(t, gitDir, before)
	if got, err := Classify(ctx, repo.Dir, nil); err != nil || len(got) != 0 {
		t.Errorf("Classify(none) = %v, %v", got, err)
	}
	tree := repo.Git("rev-parse", shas[0]+"^{tree}")
	if _, err := Classify(ctx, repo.Dir, []string{tree}); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("R5 a tree: %v; want an environment error", err)
	}
	for _, id := range []string{"HEAD", strings.ToUpper(shas[0]), shas[0][:39], "--all"} {
		if _, err := Classify(ctx, repo.Dir, []string{id}); apperr.ExitCode(err) != apperr.ExitInternal {
			t.Errorf("Classify(%q) = %v; want an internal error", id, err)
		}
	}
	for _, c := range []struct {
		name    string
		content []byte
	}{{"R6 a damaged loose object", []byte("not zlib")}, {"R7 an empty loose object", nil}} {
		repo, shas := chain(t, 2)
		p := loose(repo, shas[0])
		writable(t, p)
		if err := os.WriteFile(p, c.content, 0o644); err != nil {
			t.Fatal(err)
		}
		if got, err := Classify(ctx, repo.Dir, []string{shas[0]}); apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Errorf("%s: Classify = %v, %v; want an environment error", c.name, got, err)
		}
	}
}

func mapsEqual(a, b map[string]Presence) bool {
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

// WP2.3 spec §5.2's storage check (K1 to K22, reviews 052 B2-R and 055
// B2-R2): a commit git calls missing, with exit 1 and no stderr, is Absent
// only if the object storage is complete: every directory listable, every
// .pack with its .idx and every .idx with its .pack, every .idx's checksum
// right and its pack's trailer the one it records, no file it does not
// recognize, and the same for every alternate. Otherwise it cannot be
// confirmed (exit 6). The check writes nothing.
func TestClassifyChecksTheStorage(t *testing.T) {
	requireSafeGit(t)
	type setup func(t *testing.T, repo *testutil.GitRepo, pack string) (dir string)
	moveIdx := func(t *testing.T, repo *testutil.GitRepo, pack string) string {
		if err := os.Rename(pack+".idx", filepath.Join(t.TempDir(), "away.idx")); err != nil {
			t.Fatal(err)
		}
		return repo.Dir
	}
	for _, c := range []struct {
		name string
		// reclaimed: the old tip asked about is a truly reclaimed commit;
		// otherwise it is a commit that only the pack holds.
		reclaimed bool
		setup     setup
		absent    bool
	}{
		{name: "K1 a reclaimed tip", reclaimed: true, absent: true,
			setup: func(_ *testing.T, repo *testutil.GitRepo, _ string) string { return repo.Dir }},
		{name: "K18 a .pack that lost its .idx", setup: moveIdx},
		{name: "K6 an .idx that lost its .pack", setup: func(t *testing.T, repo *testutil.GitRepo, pack string) string {
			writable(t, pack+".pack")
			if err := os.Remove(pack + ".pack"); err != nil {
				t.Fatal(err)
			}
			return repo.Dir
		}},
		{name: "K2 a truncated .idx", setup: func(t *testing.T, repo *testutil.GitRepo, pack string) string {
			writable(t, pack+".idx")
			if err := os.Truncate(pack+".idx", 1100); err != nil {
				t.Fatal(err)
			}
			return repo.Dir
		}},
		{name: "K3 a changed byte in the old tip's .idx entry", setup: func(t *testing.T, repo *testutil.GitRepo, pack string) string {
			writable(t, pack+".idx")
			b, err := os.ReadFile(pack + ".idx")
			if err != nil {
				t.Fatal(err)
			}
			id, err := hex.DecodeString(repo.Git("rev-parse", "main~1"))
			if err != nil {
				t.Fatal(err)
			}
			i := bytes.Index(b[8+256*4:], id)
			if i < 0 || i%20 != 0 {
				t.Fatal("the old tip is not in the .idx's object IDs")
			}
			b[8+256*4+i+19] ^= 0x01 // its last byte: the lookup misses it
			if err := os.WriteFile(pack+".idx", b, 0o644); err != nil {
				t.Fatal(err)
			}
			return repo.Dir
		}},
		{name: "a .pack whose trailer the .idx does not record", setup: func(t *testing.T, repo *testutil.GitRepo, pack string) string {
			writable(t, pack+".pack")
			b, err := os.ReadFile(pack + ".pack")
			if err != nil {
				t.Fatal(err)
			}
			b[len(b)-1] ^= 0x01
			if err := os.WriteFile(pack+".pack", b, 0o644); err != nil {
				t.Fatal(err)
			}
			return repo.Dir
		}},
		{name: "K20 a leftover tmp_pack file", reclaimed: true, setup: func(t *testing.T, repo *testutil.GitRepo, pack string) string {
			if err := os.WriteFile(filepath.Join(filepath.Dir(pack), "tmp_pack_XYZ"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			return repo.Dir
		}},
		{name: "K22 an .rev without its .pack", reclaimed: true, setup: func(t *testing.T, repo *testutil.GitRepo, pack string) string {
			if err := os.WriteFile(filepath.Join(filepath.Dir(pack), "pack-"+strings.Repeat("1", 40)+".rev"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			return repo.Dir
		}},
		{name: "K21 .keep, .bitmap and .rev beside their pack", reclaimed: true, absent: true,
			setup: func(t *testing.T, repo *testutil.GitRepo, pack string) string {
				if err := os.WriteFile(pack+".keep", nil, 0o644); err != nil {
					t.Fatal(err)
				}
				for _, ext := range []string{".bitmap", ".rev"} {
					if _, err := os.Stat(pack + ext); err != nil {
						t.Fatalf("git wrote no %s: %v", ext, err)
					}
				}
				return repo.Dir
			}},
		{name: "a multi-pack-index beside the packs", reclaimed: true, absent: true,
			setup: func(_ *testing.T, repo *testutil.GitRepo, _ string) string {
				repo.Git("multi-pack-index", "write")
				return repo.Dir
			}},
		{name: "K19 a .pack that lost its .idx in an alternate", setup: func(t *testing.T, repo *testutil.GitRepo, pack string) string {
			other := testutil.NewGitRepo(t)
			alt := filepath.Join(other.Dir, ".git", "objects", "info", "alternates")
			if err := os.WriteFile(alt, []byte(filepath.ToSlash(objectsDir(repo))+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			moveIdx(t, repo, pack)
			return other.Dir
		}},
		{name: "K17 an alternate that does not exist", reclaimed: true, setup: func(t *testing.T, repo *testutil.GitRepo, _ string) string {
			alt := filepath.Join(objectsDir(repo), "info", "alternates")
			if err := os.WriteFile(alt, []byte(filepath.ToSlash(filepath.Join(t.TempDir(), "none"))+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return repo.Dir
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, shas := chain(t, 2)
			old := shas[0]
			var pack string
			if c.reclaimed {
				old = reclaim(t, repo)
				repo.Git("-c", "pack.writeReverseIndex=true", "repack", "-adqb")
				repo.Git("prune-packed")
				packs, _ := filepath.Glob(filepath.Join(objectsDir(repo), "pack", "pack-*.pack"))
				if len(packs) != 1 {
					t.Fatalf("packs: %v", packs)
				}
				pack = strings.TrimSuffix(packs[0], ".pack")
			} else {
				pack = packOnly(t, repo)
			}
			dir := c.setup(t, repo, pack)
			gitDir := filepath.Join(dir, ".git")
			before := testutil.SettledSnapshot(t, gitDir)
			objects := testutil.SettledSnapshot(t, objectsDir(repo))
			got, err := Classify(ctx, dir, []string{old})
			switch {
			case c.absent && (err != nil || got[old] != Absent):
				t.Errorf("Classify = %v, %v; want Absent", got, err)
			case !c.absent && apperr.ExitCode(err) != apperr.ExitEnvironment:
				t.Errorf("Classify = %v, %v; want an environment error", got, err)
			}
			testutil.AssertTreeUnchanged(t, gitDir, before)
			testutil.AssertTreeUnchanged(t, objectsDir(repo), objects)
		})
	}

	// K18 undone: with the .idx back, the commit reads again, so it was
	// never a reclaimed one.
	repo, shas := chain(t, 2)
	pack := packOnly(t, repo)
	away := filepath.Join(t.TempDir(), "away.idx")
	if err := os.Rename(pack+".idx", away); err != nil {
		t.Fatal(err)
	}
	if _, err := Classify(ctx, repo.Dir, []string{shas[0]}); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("K18: %v", err)
	}
	if err := os.Rename(away, pack+".idx"); err != nil {
		t.Fatal(err)
	}
	if got, err := Classify(ctx, repo.Dir, []string{shas[0]}); err != nil || got[shas[0]] != Present {
		t.Errorf("K18 restored: %v, %v; want Present", got, err)
	}
}

// Alternates nest at most as deep as git reads them (5): a deeper chain
// cannot be checked, so the commit cannot be confirmed absent.
func TestClassifyAlternatesTooDeep(t *testing.T) {
	requireSafeGit(t)
	repo, _ := chain(t, 1)
	gone := reclaim(t, repo)
	prev := objectsDir(repo)
	for range 7 {
		next := filepath.Join(t.TempDir(), "objects")
		for _, d := range []string{"info", "pack"} {
			if err := os.MkdirAll(filepath.Join(next, d), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(prev, "info", "alternates"), []byte(filepath.ToSlash(next)+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		prev = next
	}
	if got, err := Classify(ctx, repo.Dir, []string{gone}); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("Classify = %v, %v; want an environment error", got, err)
	}
}

// K10 to K16 (Linux and macOS, not as root): directories and files that
// cannot be read make the storage check fail, so a missing commit is not
// taken for a reclaimed one.
func TestClassifyWithUnreadableStorage(t *testing.T) {
	requireSafeGit(t)
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("file modes do not keep root or Windows from reading")
	}
	for _, c := range []struct {
		name  string
		path  func(repo *testutil.GitRepo, pack, tip string) string
		mode  os.FileMode
		packd bool
	}{
		{"K10 a loose object's directory unreadable", func(r *testutil.GitRepo, _, tip string) string { return filepath.Dir(loose(r, tip)) }, 0, false},
		{"K11 a loose object unreadable (present, not readable)", func(r *testutil.GitRepo, _, tip string) string { return loose(r, tip) }, 0, false},
		{"K12 a .pack unreadable", func(_ *testutil.GitRepo, pack, _ string) string { return pack + ".pack" }, 0, true},
		{"K13 an .idx unreadable", func(_ *testutil.GitRepo, pack, _ string) string { return pack + ".idx" }, 0, true},
		{"K14 the pack directory unreadable", func(_ *testutil.GitRepo, pack, _ string) string { return filepath.Dir(pack) }, 0o300, true},
		{"K15 the pack directory listable but not searchable", func(_ *testutil.GitRepo, pack, _ string) string { return filepath.Dir(pack) }, 0o600, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, shas := chain(t, 2)
			pack := ""
			if c.packd {
				pack = packOnly(t, repo)
			}
			p := c.path(repo, pack, shas[0])
			info, err := os.Stat(p)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(p, c.mode); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(p, info.Mode()) })
			if got, err := Classify(ctx, repo.Dir, []string{shas[0]}); apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Errorf("Classify = %v, %v; want an environment error", got, err)
			}
		})
	}
	t.Run("K16 the alternates file unreadable", func(t *testing.T) {
		repo, shas := chain(t, 2)
		other := testutil.NewGitRepo(t)
		alt := filepath.Join(other.Dir, ".git", "objects", "info", "alternates")
		if err := os.WriteFile(alt, []byte(objectsDir(repo)+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(alt, 0); err != nil {
			t.Fatal(err)
		}
		if got, err := Classify(ctx, other.Dir, []string{shas[0]}); apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Errorf("Classify = %v, %v; want an environment error", got, err)
		}
	})
}

// WP2.3 spec §5.2 step 3: is-ancestor's 0 and 1 are yes and no, read
// with the commit-graph out of use; anything else is an error (G2: the
// ancestor's object gone). A commit missing on the way is never a yes:
// git 2.56 fails, while git 2.43 and 2.32 answer no (scratchpad experiment
// g2.sh); a no makes the scan read the ref's whole history, which the
// strict log then fails on (gitscan's TestScanFailuresWriteNothing).
func TestIsAncestor(t *testing.T) {
	requireSafeGit(t)
	repo, shas := chain(t, 3)
	repo.Git("commit-graph", "write", "--reachable")
	if yes, err := IsAncestor(ctx, repo.Dir, shas[0], shas[2]); err != nil || !yes {
		t.Errorf("IsAncestor(A, C) = %v, %v", yes, err)
	}
	if yes, err := IsAncestor(ctx, repo.Dir, shas[2], shas[0]); err != nil || yes {
		t.Errorf("IsAncestor(C, A) = %v, %v", yes, err)
	}
	if _, err := IsAncestor(ctx, repo.Dir, strings.Repeat("e", 40), shas[2]); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("R4 a missing ancestor: %v", err)
	}
	// B's object gone, the commit-graph still knows it.
	p := loose(repo, shas[1])
	writable(t, p)
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if yes, err := IsAncestor(ctx, repo.Dir, shas[0], shas[2]); yes {
		t.Errorf("a missing commit on the way: IsAncestor = yes, %v", err)
	} else {
		t.Logf("a missing commit on the way: IsAncestor = no, %v", err)
	}
	// G2: A's object gone.
	repo, shas = chain(t, 3)
	repo.Git("commit-graph", "write", "--reachable")
	p = loose(repo, shas[0])
	writable(t, p)
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if _, err := IsAncestor(ctx, repo.Dir, shas[0], shas[2]); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("G2 the ancestor's object gone: %v", err)
	}
	if _, err := IsAncestor(ctx, repo.Dir, "HEAD", shas[2]); apperr.ExitCode(err) != apperr.ExitInternal {
		t.Errorf("IsAncestor(HEAD) = %v; want an internal error", err)
	}
}

// logAll runs Log and returns what fn received.
func logAll(t *testing.T, dir, tip string, exclude ...string) ([]Commit, error) {
	t.Helper()
	var got []Commit
	err := Log(ctx, dir, tip, exclude, func(c Commit) error {
		got = append(got, c)
		return nil
	})
	return got, err
}

func shasOf(cs []Commit) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.SHA)
	}
	return out
}

// WP2.3 spec §5.2 step 4 (R11, R12, L1 to L5): the commits reachable from
// the tip and from no exclusion, newest first, each with its committer
// time, its Specmetry-Ref trailer values and its message, read with
// exclusions as ^<sha> lines on stdin.
func TestLog(t *testing.T) {
	requireSafeGit(t)
	repo, shas := chain(t, 3)
	got, err := logAll(t, repo.Dir, shas[2], shas[0])
	if err != nil || !slices.Equal(shasOf(got), []string{shas[2], shas[1]}) {
		t.Fatalf("Log(C ^A) = %v, %v", shasOf(got), err)
	}
	c := got[0]
	if c.Trailers != "REQ-001" || !strings.HasPrefix(c.Message, "fix AC-001-1\n") || c.Time <= 0 {
		t.Errorf("C = %+v", c)
	}
	if got, err := logAll(t, repo.Dir, shas[2]); err != nil || len(got) != 3 {
		t.Errorf("Log(C) = %v, %v", shasOf(got), err)
	}
	if got, err := logAll(t, repo.Dir, shas[2], shas[2]); err != nil || len(got) != 0 {
		t.Errorf("R12 Log(C ^C) = %v, %v", shasOf(got), err)
	}
	// L2: trailer keys in any case, with or without a space.
	repo.WriteFile("t.txt", "t\n")
	repo.Commit("trailers\n\nSpecmetry-Ref: REQ-010, AC-010-1\nspecmetry-ref: REQ-011 REQ-012\nSPECMETRY-REF:REQ-013")
	if got, err := logAll(t, repo.Dir, repo.Git("rev-parse", "HEAD"), shas[2]); err != nil || len(got) != 1 ||
		got[0].Trailers != "REQ-010, AC-010-1,REQ-011 REQ-012,REQ-013" {
		t.Errorf("L2: %+v, %v", got, err)
	}
	for _, args := range [][]string{{"HEAD"}, {shas[2], "--all"}, {strings.ToUpper(shas[2])}} {
		if err := Log(ctx, repo.Dir, args[0], args[1:], func(Commit) error { return nil }); apperr.ExitCode(err) != apperr.ExitInternal {
			t.Errorf("Log(%q) = %v; want an internal error", args, err)
		}
	}
	// fn's error stops the read and is returned.
	stop := errors.New("stop")
	if err := Log(ctx, repo.Dir, shas[2], nil, func(Commit) error { return stop }); !errors.Is(err, stop) {
		t.Errorf("Log with fn failing = %v", err)
	}
}

// WP2.3 spec §5.2 (review 050 B1, review 052 B1-R): the read is strict. A
// missing tip or exclusion (R1, R3), a missing commit on the way (G3, and
// git may print some commits first), or an exclusion that vanished while
// the commit-graph still knows it (G1) fails the read; grafts (G5) and a
// shallow file (H5) that appeared after the check do not cut it short;
// a damaged multi-pack-index (K7) is not used.
func TestLogIsStrict(t *testing.T) {
	requireSafeGit(t)
	failsWithout := func(name string, repo *testutil.GitRepo, oid string, read func() error) {
		t.Helper()
		p := loose(repo, oid)
		writable(t, p)
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		if err := read(); apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Errorf("%s: %v; want an environment error", name, err)
		}
	}
	repo, shas := chain(t, 3)
	if _, err := logAll(t, repo.Dir, strings.Repeat("e", 40)); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("R1 a missing tip: %v", err)
	}
	if _, err := logAll(t, repo.Dir, shas[2], strings.Repeat("e", 40)); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("R3 an exclusion that never existed: %v", err)
	}

	repo, shas = chain(t, 3)
	repo.Git("commit-graph", "write", "--reachable")
	failsWithout("G1 an exclusion gone, in the commit-graph", repo, shas[0], func() error {
		_, err := logAll(t, repo.Dir, shas[2], shas[0])
		return err
	})

	repo, shas = chain(t, 3)
	repo.Git("commit-graph", "write", "--reachable")
	failsWithout("G3 a commit on the way gone", repo, shas[1], func() error {
		_, err := logAll(t, repo.Dir, shas[2], shas[0])
		return err
	})

	repo, shas = chain(t, 3)
	grafts := filepath.Join(repo.Dir, ".git", "info", "grafts")
	if err := os.MkdirAll(filepath.Dir(grafts), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(grafts, []byte(shas[2]+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(repo.Git("log", "--format=%H", shas[2]), "\n") + 1; n != 1 {
		t.Fatalf("G5 control: plain git log reads %d commits, want 1: the grafts are not in effect", n)
	}
	if got, err := logAll(t, repo.Dir, shas[2]); err != nil || len(got) != 3 {
		t.Errorf("G5 grafts not applied: %v, %v", shasOf(got), err)
	}
	failsWithout("G5 grafts and the root gone", repo, shas[0], func() error {
		_, err := logAll(t, repo.Dir, shas[2])
		return err
	})

	repo, shas = chain(t, 3)
	if err := os.WriteFile(filepath.Join(repo.Dir, ".git", "shallow"), []byte(shas[1]+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(repo.Git("log", "--format=%H", shas[2]), "\n") + 1; n != 2 {
		t.Fatalf("H5 control: plain git log reads %d commits, want 2: the shallow file is not in effect", n)
	}
	if got, err := logAll(t, repo.Dir, shas[2]); err != nil || len(got) != 3 {
		t.Errorf("H5 a shallow file after the check: %v, %v", shasOf(got), err)
	}
	failsWithout("H5 a shallow file and the root gone", repo, shas[0], func() error {
		_, err := logAll(t, repo.Dir, shas[2])
		return err
	})

	repo, shas = chain(t, 3)
	packOnly(t, repo)
	repo.Git("multi-pack-index", "write")
	midx := filepath.Join(objectsDir(repo), "pack", "multi-pack-index")
	writable(t, midx)
	b, err := os.ReadFile(midx)
	if err != nil {
		t.Fatal(err)
	}
	for i := 12; i < len(b)-20; i += 7 {
		b[i] ^= 0x55
	}
	if err := os.WriteFile(midx, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := logAll(t, repo.Dir, shas[2]); err != nil || len(got) != 3 {
		t.Errorf("K7 a damaged multi-pack-index: %v, %v", shasOf(got), err)
	}
}

// G4 and G7: the read needs only the commits it visits (an ancestor of
// an exclusion may be missing), and with nothing missing a commit-graph and
// a multi-pack-index change nothing.
func TestLogReadsWhatItVisits(t *testing.T) {
	requireSafeGit(t)
	repo, shas := chain(t, 4)
	p := loose(repo, shas[1])
	writable(t, p)
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if got, err := logAll(t, repo.Dir, shas[3], shas[2]); err != nil || !slices.Equal(shasOf(got), []string{shas[3]}) {
		t.Errorf("G4 Log(Z ^Y), X gone: %v, %v", shasOf(got), err)
	}
	repo, shas = chain(t, 3)
	packOnly(t, repo)
	repo.Git("multi-pack-index", "write")
	repo.Git("commit-graph", "write", "--reachable")
	if got, err := logAll(t, repo.Dir, shas[2], shas[0]); err != nil || !slices.Equal(shasOf(got), []string{shas[2], shas[1]}) {
		t.Errorf("G7 with a commit-graph and a multi-pack-index: %v, %v", shasOf(got), err)
	}
}

// architecture §6.1.2 (no program runs): log.showSignature in the
// repository's configuration makes git log run gpg.program for a signed
// commit (the control, plain git log); the scan's read does not.
func TestLogRunsNoSignatureProgram(t *testing.T) {
	requireSafeGit(t)
	// The stand-in's marker reaches git through the environment, which a
	// repository takes when it is made.
	dir := t.TempDir()
	marker := testutil.FakeRemoteHelper(t, dir, "gpg")
	repo, shas := chain(t, 1)
	exe := filepath.Join(dir, "git-remote-gpg")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	repo.Git("config", "log.showSignature", "true")
	repo.Git("config", "gpg.program", filepath.ToSlash(exe))
	tree := repo.Git("rev-parse", shas[0]+"^{tree}")
	raw := "tree " + tree + "\nparent " + shas[0] + "\nauthor t <t@e> 1 +0000\ncommitter t <t@e> 1 +0000\n" +
		"gpgsig -----BEGIN PGP SIGNATURE-----\n \n abc\n -----END PGP SIGNATURE-----\n\nsigned\n"
	f := filepath.Join(t.TempDir(), "commit")
	if err := os.WriteFile(f, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	signed := repo.Git("hash-object", "-t", "commit", "-w", f)
	_, _ = repo.GitResult("log", "-1", "--format=%H", signed)
	if !ranAt(marker) {
		t.Fatal("control: plain git log did not run gpg.program, so this test shows nothing")
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if got, err := logAll(t, repo.Dir, signed); err != nil || len(got) != 2 {
		t.Errorf("Log = %v, %v", shasOf(got), err)
	}
	if ranAt(marker) {
		t.Error("the scan's read ran gpg.program")
	}
}

// The log's records are checked before fn sees them: four NUL-terminated
// fields, a full lower-case SHA-1 and a decimal time.
func TestLogRecords(t *testing.T) {
	sha := strings.Repeat("a", 40)
	for _, c := range []struct {
		name string
		out  string
		n    int
		ok   bool
	}{
		{"none", "", 0, true},
		{"one", sha + "\x001700000000\x00REQ-001\x00msg\n\x00", 1, true},
		{"two", sha + "\x001\x00\x00m\x00" + strings.Repeat("b", 40) + "\x002\x00\x00n\x00", 2, true},
		{"a short SHA", sha[1:] + "\x001\x00\x00m\x00", 0, false},
		{"an upper-case SHA", strings.ToUpper(sha) + "\x001\x00\x00m\x00", 0, false},
		{"a time that is not a number", sha + "\x00x1\x00\x00m\x00", 0, false},
		{"a negative time", sha + "\x00-1\x00\x00m\x00", 0, false},
		{"a time too large for milliseconds", sha + "\x0099999999999999999\x00\x00m\x00", 0, false},
		{"a record cut short", sha + "\x001\x00\x00m", 0, false},
		{"three fields", sha + "\x001\x00m\x00", 0, false},
	} {
		var got []Commit
		r := &logReader{fn: func(c Commit) error { got = append(got, c); return nil }}
		for _, chunk := range splitEvery([]byte(c.out), 3) { // any split of the stream
			if _, err := r.Write(chunk); err != nil {
				break
			}
		}
		err := r.close()
		switch {
		case c.ok && (err != nil || len(got) != c.n):
			t.Errorf("%s: %d commits, %v", c.name, len(got), err)
		case !c.ok && apperr.ExitCode(err) != apperr.ExitEnvironment:
			t.Errorf("%s: %v; want an environment error", c.name, err)
		}
	}
}

func splitEvery(b []byte, n int) [][]byte {
	var out [][]byte
	for len(b) > n {
		out = append(out, b[:n])
		b = b[n:]
	}
	return append(out, b)
}

// PD-26: with a git that would fetch a missing object, every scan read is
// refused before it runs, as every object read is.
func TestScanReadsNeedASafeGit(t *testing.T) {
	requireSafeGit(t)
	repo, shas := chain(t, 2)
	dir, env, calls := testutil.PassthroughGit(t, testutil.PassWithoutNoLazyFetch)
	useGit(t, dir, env)
	for name, read := range map[string]func() error{
		"TreeEntry": func() error { _, _, err := TreeEntry(ctx, repo.Dir, shas[1], "a.txt"); return err },
		"Tips":      func() error { _, err := Tips(ctx, repo.Dir); return err },
		"Classify":  func() error { _, err := Classify(ctx, repo.Dir, shas); return err },
		"IsAncestor": func() error {
			_, err := IsAncestor(ctx, repo.Dir, shas[0], shas[1])
			return err
		},
		"Log": func() error { _, err := logAll(t, repo.Dir, shas[1]); return err },
	} {
		if err := read(); apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "GIT_NO_LAZY_FETCH") {
			t.Errorf("%s = %v; want the object reads refused", name, err)
		}
	}
	for _, c := range calls() {
		for _, sub := range []string{"ls-tree", "for-each-ref", "merge-base", "log"} {
			if slices.Contains(c.Args, sub) {
				t.Errorf("%s ran: %q", sub, c.Args)
			}
		}
		if slices.Contains(c.Args, "cat-file") && !slices.ContainsFunc(c.Args, func(a string) bool { return strings.HasPrefix(a, "--git-dir=") }) {
			t.Errorf("cat-file ran in the repository: %q", c.Args)
		}
	}
}

// PD-26 with the real git on PATH (WP2.3 spec §11, maintenance item M-9):
// run with SPECMETRY_TEST_GIT_OBJECTS=unsafe and a git that would fetch a
// missing object (git 2.32.0), every WP2.3 read that may read an object is
// refused before it runs (exit 6), and nothing in the repository changes.
func TestWP23ReadsWithAGitThatFetches(t *testing.T) {
	if os.Getenv(gitObjectsEnv) != "unsafe" {
		t.Skip("needs " + gitObjectsEnv + "=unsafe and a git that fetches")
	}
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("a.txt", "a\n")
	first := repo.Commit("one")
	repo.WriteFile("b.txt", "b\n")
	second := repo.Commit("two REQ-001")
	gitDir := filepath.Join(repo.Dir, ".git")
	before := testutil.SettledSnapshot(t, gitDir)
	for name, read := range map[string]func() error{
		"TreeEntry":  func() error { _, _, err := TreeEntry(ctx, repo.Dir, second, "a.txt"); return err },
		"Tips":       func() error { _, err := Tips(ctx, repo.Dir); return err },
		"Classify":   func() error { _, err := Classify(ctx, repo.Dir, []string{first}); return err },
		"IsAncestor": func() error { _, err := IsAncestor(ctx, repo.Dir, first, second); return err },
		"Log":        func() error { _, err := logAll(t, repo.Dir, second); return err },
		"Clean":      func() error { _, err := Clean(ctx, repo.Dir); return err },
	} {
		if err := read(); apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "GIT_NO_LAZY_FETCH") {
			t.Errorf("%s = %v; want the object reads refused", name, err)
		}
	}
	testutil.AssertTreeUnchanged(t, gitDir, before)
	if err := CheckShallow(ctx, repo.Dir); err != nil {
		t.Errorf("CheckShallow reads no object and works: %v", err)
	}
}
