package gitx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

// scratchIn makes a Scratch in a data directory of the test's own and
// returns it and the data directory.
func scratchIn(t *testing.T) (*Scratch, datadir.Dir) {
	t.Helper()
	d := datadir.Dir{Path: filepath.Join(testdir.New(t), "home")}
	s, err := NewScratch(d)
	if err != nil {
		t.Fatalf("NewScratch: %v (%v)", err, apperr.From(err).Details)
	}
	return s, d
}

// entries lists the names in the data directory's tmp directory.
func entries(t *testing.T, d datadir.Dir) []string {
	t.Helper()
	es, err := os.ReadDir(d.Join(datadir.TmpDir))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

// Review 058 B1, the first counterexample: the private directories live
// only where datadir.Ensure has shown that no other user can replace
// anything on the path, and that everything is private. A data directory
// whose parent other users can rename entries in (FILE_DELETE_CHILD for
// Everyone on Windows, a non-sticky 0777 directory elsewhere) is refused
// before anything is made in it; so is one whose tmp directory others can
// read. The system's temporary directory is not used at all, whatever it
// allows.
func TestScratchNeedsAPrivateDataDirectory(t *testing.T) {
	parent := filepath.Join(testdir.New(t), "shared")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	makeReplaceable(t, parent)
	d := datadir.Dir{Path: filepath.Join(parent, "home")}
	if _, err := NewScratch(d); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("a parent others can rename entries in: NewScratch = %v; want a refusal", err)
	}
	if _, err := os.Lstat(d.Path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the refused data directory was made: %v", err)
	}

	_, d = scratchIn(t)
	testutil.LoosenPermissions(t, d.Join(datadir.TmpDir))
	if _, err := NewScratch(d); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("a tmp directory others can read: NewScratch = %v; want a refusal", err)
	}

	// A temporary directory others can rename entries in is not used.
	requireSafeGit(t)
	repo := baseRepo(t)
	tmp := filepath.Join(t.TempDir(), "tmp")
	if err := os.Mkdir(tmp, 0o700); err != nil {
		t.Fatal(err)
	}
	makeReplaceable(t, tmp)
	for _, v := range []string{"TMP", "TEMP", "TMPDIR"} {
		t.Setenv(v, tmp)
	}
	s, d := scratchIn(t)
	if got, err := Clean(ctx, s, repo.Dir); err != nil || got != StateClean {
		t.Errorf("Clean = %v, %v", got, err)
	}
	if es, _ := os.ReadDir(tmp); len(es) != 0 {
		t.Errorf("the system's temporary directory was used: %v", es)
	}
	if left := entries(t, d); len(left) != 0 {
		t.Errorf("left in tmp: %v", left)
	}
}

// Review 058 B1, the second counterexample: a private directory moved
// away and another put at its path (an empty directory, or a link to one)
// after it was made fails the read before git uses it, through Clean and
// through a scan read; what is at the path is not removed, and no lock is
// left that would let a later cleanup remove it.
func TestPrivateDirectoryReplaced(t *testing.T) {
	requireSafeGit(t)
	for _, c := range []struct {
		name string
		link bool
		read func(t *testing.T, s *Scratch) error
	}{
		{"by a directory in Clean", false, readClean},
		{"by a directory in a scan read", false, readLog},
		{"by a link in Clean", true, readClean},
		{"by a link in a scan read", true, readLog},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, d := scratchIn(t)
			target := filepath.Join(d.Path, "elsewhere")
			if err := os.Mkdir(target, 0o700); err != nil {
				t.Fatal(err)
			}
			var at, witness string
			testHookPrivateDir = func(path string) {
				testHookPrivateDir = nil
				at = path
				if err := os.Rename(path, path+".moved"); err != nil {
					t.Fatal(err)
				}
				if c.link {
					link(t, target, path)
					witness = filepath.Join(target, "witness")
				} else {
					if err := os.Mkdir(path, 0o700); err != nil {
						t.Fatal(err)
					}
					witness = filepath.Join(path, "witness")
				}
				if err := os.WriteFile(witness, []byte("keep"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			t.Cleanup(func() { testHookPrivateDir = nil })
			if err := c.read(t, s); apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "moved or replaced") {
				t.Fatalf("the read = %v; want it to fail on the replacement", err)
			}
			if _, err := os.Stat(witness); err != nil {
				t.Errorf("the replacement's content was removed: %v", err)
			}
			if _, err := os.Lstat(at); err != nil {
				t.Errorf("what replaced the private directory was removed: %v", err)
			}
			if _, err := os.Lstat(at + lockSuffix); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("its lock is left for a cleanup to act on: %v", err)
			}
			if _, err := NewScratch(d); err == nil {
				if _, err := os.Stat(witness); err != nil {
					t.Errorf("the next Scratch removed the replacement's content: %v", err)
				}
			}
		})
	}
}

func readClean(t *testing.T, s *Scratch) error {
	repo := baseRepo(t)
	_, err := Clean(ctx, s, repo.Dir)
	return err
}

func readLog(t *testing.T, s *Scratch) error {
	repo, shas := chain(t, 2)
	return Log(ctx, s, repo.Dir, shas[1], nil, func(Commit) error { return nil })
}

// The tmp directory itself moved away and another put in its place after
// the Scratch was made: no private directory is made in the replacement,
// and nothing in either is removed.
func TestScratchRootReplaced(t *testing.T) {
	s, d := scratchIn(t)
	root := d.Join(datadir.TmpDir)
	if err := os.Rename(root, root+".moved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := newPrivateDir(s); apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "moved or replaced") {
		t.Errorf("newPrivateDir = %v; want a refusal", err)
	}
	if left := entries(t, d); len(left) != 0 {
		t.Errorf("made in the replacement: %v", left)
	}
}

// Review 058 B1, the third counterexample, and the lifecycle: a new
// Scratch removes only what it can show is a crashed read's: a lock file
// Specmetry names, whose lock no process holds, and the directory of that
// name. A private directory in use (its lock held), a directory without
// its lock, a lock that is not a file, names Specmetry does not make, and
// anything in the system's temporary directory stay.
func TestScratchRemovesOnlyItsOwnLeftovers(t *testing.T) {
	s, d := scratchIn(t)
	root := d.Join(datadir.TmpDir)
	mk := func(name string, dir bool) {
		p := filepath.Join(root, name)
		var err error
		if dir {
			err = os.Mkdir(p, 0o700)
			if err == nil {
				err = os.WriteFile(filepath.Join(p, "witness"), []byte("x"), 0o600)
			}
		} else {
			err = os.WriteFile(p, nil, 0o600)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	const crashed, lockOnly, noLock = "read-11111111111111111111111111111111", "read-22222222222222222222222222222222", "read-33333333333333333333333333333333"
	mk(crashed+lockSuffix, false)
	mk(crashed, true)
	mk(lockOnly+lockSuffix, false)
	mk(noLock, true)
	mk("read-44444444444444444444444444444444.lock", true) // a lock that is a directory
	mk("read-not-hex", true)
	mk("read-not-hex.lock", false)
	mk("specmetry-private-old", true)
	mk("notes.txt", false)
	active, err := newPrivateDir(s)
	if err != nil {
		t.Fatal(err)
	}
	inUse := active.join("in-use")
	if err := os.WriteFile(inUse, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	system := filepath.Join(os.TempDir(), privateNameForTest())
	if err := os.Mkdir(system, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(system) })

	if _, err := NewScratch(d); err != nil {
		t.Fatal(err)
	}
	left := entries(t, d)
	for _, gone := range []string{crashed, crashed + lockSuffix, lockOnly + lockSuffix} {
		if slices.Contains(left, gone) {
			t.Errorf("not removed: %s", gone)
		}
	}
	for _, kept := range []string{noLock, "read-44444444444444444444444444444444.lock", "read-not-hex", "read-not-hex.lock",
		"specmetry-private-old", "notes.txt", filepath.Base(active.path), filepath.Base(active.lockPath)} {
		if !slices.Contains(left, kept) {
			t.Errorf("removed: %s", kept)
		}
	}
	if _, err := os.Stat(inUse); err != nil {
		t.Errorf("a file of the private directory in use was removed: %v", err)
	}
	if _, err := os.Stat(system); err != nil {
		t.Errorf("the system's temporary directory was cleaned: %v", err)
	}
	if err := active.remove(); err != nil {
		t.Fatal(err)
	}
}

// privateNameForTest is a name newPrivateDir could make.
func privateNameForTest() string { return "read-" + strings.Repeat("ab", privateNameLen/2) }

// Review 052 N2, WP2.3 spec §4.3: after a read that succeeds, fails or is
// cancelled, nothing is left in tmp; a private directory that cannot be
// removed fails the read and keeps its lock, unlocked, so that the next
// Scratch removes it; a missing Scratch is a mistake of the caller.
func TestPrivateDirectoryLifecycle(t *testing.T) {
	requireSafeGit(t)
	s, d := scratchIn(t)
	super := subRepo(t)
	if got, err := Clean(ctx, s, super.Dir); err != nil || got != StateClean {
		t.Fatalf("Clean = %v, %v", got, err)
	}
	if left := entries(t, d); len(left) != 0 {
		t.Errorf("left after success: %v", left)
	}
	broken := baseRepo(t)
	if err := os.Truncate(filepath.Join(broken.Dir, ".git", "index"), 30); err != nil {
		t.Fatal(err)
	}
	if _, err := Clean(ctx, s, broken.Dir); err == nil {
		t.Error("a truncated index was read")
	}
	repo, shas := chain(t, 2)
	if err := Log(ctx, s, repo.Dir, strings.Repeat("e", 40), nil, func(Commit) error { return nil }); err == nil {
		t.Error("a missing tip was read")
	}
	if left := entries(t, d); len(left) != 0 {
		t.Errorf("left after failures: %v", left)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := Clean(cancelled, s, super.Dir); err == nil {
		t.Error("a cancelled read succeeded")
	}
	if err := Log(cancelled, s, repo.Dir, shas[1], nil, func(Commit) error { return nil }); err == nil {
		t.Error("a cancelled log succeeded")
	}
	if left := entries(t, d); len(left) != 0 {
		t.Errorf("left after cancellation: %v", left)
	}

	failing := errors.New("cannot remove")
	removeAll = func(string) error { return failing }
	got, err := Clean(ctx, s, super.Dir)
	removeAll = os.RemoveAll
	if apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("a directory that cannot be removed: Clean = %v, %v; want an environment error", got, err)
	}
	left := entries(t, d)
	if len(left) != 2 {
		t.Fatalf("after a failed removal: %v; want the directory and its lock", left)
	}
	if _, err := NewScratch(d); err != nil {
		t.Fatal(err)
	}
	if left := entries(t, d); len(left) != 0 {
		t.Errorf("the next Scratch left: %v", left)
	}

	if _, err := Clean(ctx, nil, super.Dir); apperr.ExitCode(err) != apperr.ExitInternal {
		t.Errorf("Clean without a Scratch = %v", err)
	}
	if _, err := Tips(ctx, nil, repo.Dir); apperr.ExitCode(err) != apperr.ExitInternal {
		t.Errorf("Tips without a Scratch = %v", err)
	}
}

// WP2.3 spec §4.3 (only the current user can read or write a private
// directory): the data directory, private directories and what Clean
// writes in them included, passes datadir's whole check, the directories
// on its path included; nothing of it is filtered out.
func TestPrivateDirectoriesArePrivate(t *testing.T) {
	requireSafeGit(t)
	repo, _ := filterRepo(t)
	repo.Git("update-index", "--split-index")
	s, d := scratchIn(t)
	p, err := prepare(ctx, s, repo.Dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.close() }()
	es, err := os.ReadDir(p.gitDir)
	if err != nil || len(es) < 5 {
		t.Fatalf("the private directory holds %d entries: %v", len(es), err)
	}
	problems, err := datadir.Check(d)
	if err != nil || len(problems) != 0 {
		t.Errorf("not private: %v, %v", problems, err)
	}
}
