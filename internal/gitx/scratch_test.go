package gitx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
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
	s, err := newScratch(t, d)
	if err != nil {
		t.Fatalf("NewScratch: %v (%v)", err, apperr.From(err).Details)
	}
	return s, d
}

// newScratch is NewScratch, the Scratch closed when the test ends.
func newScratch(t *testing.T, d datadir.Dir) (*Scratch, error) {
	t.Helper()
	s, err := NewScratch(d)
	if err == nil {
		t.Cleanup(func() { _ = s.Close() })
	}
	return s, err
}

// entries lists the names in the data directory's tmp directory.
func entries(t *testing.T, d datadir.Dir) []string {
	t.Helper()
	es, err := os.ReadDir(d.Join(filepath.Join(datadir.TmpDir, scratchNamespace)))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		if e.Name() != leaseName {
			out = append(out, e.Name())
		}
	}
	return out
}

func wantEmpty(t *testing.T, d datadir.Dir) {
	t.Helper()
	if left := entries(t, d); len(left) != 0 {
		t.Errorf("left in tmp: %v", left)
	}
}

// mkPrivate makes a directory as private as the data directory's.
func mkPrivate(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// wantFiles fails unless each file holds what it held.
func wantFiles(t *testing.T, files map[string]string) {
	t.Helper()
	for p, want := range files {
		if b, err := os.ReadFile(p); err != nil || string(b) != want {
			t.Errorf("%s was removed or changed: %v", p, err)
		}
	}
}

// wantReplaced fails unless err is the read's refusal of a directory moved
// or replaced.
func wantReplaced(t *testing.T, err error) {
	t.Helper()
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "replac") {
		t.Fatalf("the read = %v; want it to fail on the change", err)
	}
}

// moveAway renames from to to and reports true. On Windows it must fail
// instead, and reports false, when from is held: a directory cannot be
// renamed there while anything in it is open (a private directory's owner
// file, held from its making until its removal), nor tmp while the
// Scratch's handle on it is. That refusal is what the schedule meets on
// Windows.
func moveAway(t *testing.T, from, to string) bool {
	t.Helper()
	err := os.Rename(from, to)
	if runtime.GOOS == "windows" {
		if err == nil {
			t.Fatalf("Windows renamed %s while it was held open", from)
		}
		t.Logf("Windows refused the move: %v", err)
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	return true
}

// reads are the public reads that make private directories.
var reads = []struct {
	name string
	read func(t *testing.T, s *Scratch) error
}{{"Clean", readClean}} // Log no longer uses a private directory; see TestScanEmptyOverrides.

func readClean(t *testing.T, s *Scratch) error {
	repo := baseRepo(t)
	_, err := Clean(ctx, s, repo.Dir)
	return err
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
	if _, err := newScratch(t, d); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("a parent others can rename entries in: NewScratch = %v; want a refusal", err)
	}
	if _, err := os.Lstat(d.Path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the refused data directory was made: %v", err)
	}

	_, d = scratchIn(t)
	testutil.LoosenPermissions(t, d.Join(datadir.TmpDir))
	if _, err := newScratch(t, d); apperr.ExitCode(err) != apperr.ExitEnvironment {
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
	wantEmpty(t, d)
}

// Review 058 B1, the second counterexample: a private directory moved
// away and another put at its path (an empty directory, or a link to one)
// after it was made fails the read before git uses it, through Clean and
// through a scan read; what is at the path is not removed, nothing is made
// in it, and a later Scratch does not remove it either. On Windows the
// move is refused: the read holds the directory open; the read then goes
// on as usual.
func TestPrivateDirectoryReplaced(t *testing.T) {
	requireSafeGit(t)
	for _, c := range []struct {
		name string
		link bool
	}{{"by a directory", false}, {"by a link", true}} {
		for _, r := range reads {
			t.Run(c.name+" in "+r.name, func(t *testing.T) {
				s, d := scratchIn(t)
				target := filepath.Join(d.Path, "elsewhere")
				mkPrivate(t, target)
				var at, witness string
				moved := false
				testHookPrivateDir = func(path string) {
					testHookPrivateDir = nil
					at = path
					if moved = moveAway(t, path, path+".moved"); !moved {
						return
					}
					if c.link {
						link(t, target, path)
						witness = filepath.Join(target, "witness")
					} else {
						mkPrivate(t, path)
						witness = filepath.Join(path, "witness")
					}
					write(t, witness, "keep")
				}
				t.Cleanup(func() { testHookPrivateDir = nil })
				err := r.read(t, s)
				if !moved {
					if err != nil {
						t.Fatal(err)
					}
					wantEmpty(t, d)
					return
				}
				wantReplaced(t, err)
				wantFiles(t, map[string]string{witness: "keep"})
				if _, err := os.Lstat(at); err != nil {
					t.Errorf("what replaced the private directory was removed: %v", err)
				}
				if _, err := os.Lstat(filepath.Join(filepath.Dir(witness), ownerName)); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("an owner file is in the replacement: %v", err)
				}
				if !c.link { // with a link in tmp, the data directory is refused
					if _, err := newScratch(t, d); apperr.ExitCode(err) != apperr.ExitEnvironment {
						t.Fatalf("unknown replacement: %v", err)
					}
					wantFiles(t, map[string]string{witness: "keep"})
				}
			})
		}
	}
}

// Review 059 B1-R, the first counterexample: a removal attempt fails, and
// before the next one the private directory is moved away and another put
// at its path. Every attempt starts by checking the path: the read fails,
// makes no further attempt, and removes neither the replacement nor what
// it holds, nor does a later Scratch; the read's own directory is kept
// where it went. Through Clean and a scan read, the attempt failing before
// the directory's content is removed ("contents") and after it, before the
// directory's own entry ("entry"). On Windows the move before the content
// is removed is refused (the read holds the owner file in the directory
// open), and the next attempt removes the same directory; the move after
// it is not, and the read fails as elsewhere.
func TestRemovalAttemptsCheckTheDirectory(t *testing.T) {
	requireSafeGit(t)
	for _, step := range []string{"contents", "entry"} {
		for _, r := range reads {
			t.Run(r.name+" "+step, func(t *testing.T) {
				s, d := scratchIn(t)
				var at, witness string
				moved := false
				calls := 0
				testHookRemove = func(st, path string) error {
					if st != step {
						return nil
					}
					if calls++; calls > 1 {
						return nil
					}
					at = path
					if moved = moveAway(t, path, path+".moved"); moved {
						mkPrivate(t, path)
						witness = filepath.Join(path, "replacement-content")
						write(t, witness, "keep")
					}
					return errors.New("a transient removal failure")
				}
				t.Cleanup(func() { testHookRemove = nil })
				err := r.read(t, s)
				testHookRemove = nil
				if !moved {
					if err != nil || calls < 2 {
						t.Fatalf("the read = %v after %d attempts; want the next attempt to remove the same directory", err, calls)
					}
					wantEmpty(t, d)
					return
				}
				wantReplaced(t, err)
				if calls != 1 {
					t.Errorf("%d attempts reached the step; want none after the change", calls)
				}
				wantFiles(t, map[string]string{witness: "keep"})
				if _, err := os.Stat(at + ".moved"); err != nil {
					t.Errorf("the read's own directory, moved away, was removed: %v", err)
				}
				if _, err := newScratch(t, d); apperr.ExitCode(err) != apperr.ExitEnvironment {
					t.Fatalf("unknown replacement: %v", err)
				}
				wantFiles(t, map[string]string{witness: "keep"})
			})
		}
	}
}

// The private directory moved away and another put at its path after an
// attempt's check and before the step it guards, the attempt going on.
// Before the content is removed ("contents"), the other directory holds
// files of the names of the read's own (config, the owner file): the
// content and the owner file are removed through the read's handles, from
// its own directory, and none of the other's. Before the directory's own
// entry is removed ("entry"), the other directory is empty, which an rmdir
// alone would remove: the entry removed must be the directory made. Both
// keep the other directory, and the read fails. On Windows the move before
// the content is removed is refused, as above.
func TestRemovalChecksWhatItRemoves(t *testing.T) {
	requireSafeGit(t)
	for _, step := range []string{"contents", "entry"} {
		for _, r := range reads {
			t.Run(r.name+" "+step, func(t *testing.T) {
				s, d := scratchIn(t)
				var at string
				kept := map[string]string{}
				moved := false
				testHookRemove = func(st, path string) error {
					if st == step && at == "" {
						at = path
						if moved = moveAway(t, path, path+".moved"); moved {
							mkPrivate(t, path)
							if step == "contents" {
								kept[filepath.Join(path, "config")] = "keep"
								kept[filepath.Join(path, ownerName)] = "keep"
							}
							for p, c := range kept {
								write(t, p, c)
							}
						}
					}
					return nil
				}
				t.Cleanup(func() { testHookRemove = nil })
				err := r.read(t, s)
				testHookRemove = nil
				if !moved {
					if err != nil {
						t.Fatal(err)
					}
					wantEmpty(t, d)
					return
				}
				wantReplaced(t, err)
				if fi, err := os.Lstat(at); err != nil || !fi.IsDir() {
					t.Errorf("the directory put in its place was removed: %v", err)
				}
				wantFiles(t, kept)
				if _, err := os.Stat(at + ".moved"); err != nil {
					t.Errorf("the read's own directory, moved away, was removed: %v", err)
				}
			})
		}
	}
}

// The positive control: attempts that fail while the directory stays the
// same are tried again, and remove it.
func TestRemovalRetriesTheSameDirectory(t *testing.T) {
	requireSafeGit(t)
	for _, r := range reads {
		t.Run(r.name, func(t *testing.T) {
			s, d := scratchIn(t)
			failures := map[string]int{"contents": 2, "entry": 2}
			testHookRemove = func(step, _ string) error {
				if failures[step] > 0 {
					failures[step]--
					return errors.New("a transient removal failure")
				}
				return nil
			}
			t.Cleanup(func() { testHookRemove = nil })
			if err := r.read(t, s); err != nil {
				t.Fatal(err)
			}
			if failures["contents"]+failures["entry"] != 0 {
				t.Errorf("failures not reached: %v", failures)
			}
			wantEmpty(t, d)
		})
	}
}

// Review 059 B1-R, the second counterexample: while a read uses its
// private directory, tmp is moved away and rebuilt with a directory of the
// same name holding an owner file (a copy of the read's own, its record
// included) and a witness, and the lock file name of the earlier layout
// beside it. The read fails and removes none of it, nor does a later
// Scratch (the copied record names another directory and file); the
// read's own directory is kept with its owner file where tmp went. On
// Windows the move is refused: the Scratch holds tmp open.
func TestScratchRootReplacedWhileInUse(t *testing.T) {
	requireSafeGit(t)
	for _, r := range reads {
		t.Run(r.name, func(t *testing.T) {
			s, d := scratchIn(t)
			tmp := d.Join(datadir.TmpDir)
			kept := map[string]string{}
			var own string
			moved := false
			testHookPrivateDir = func(path string) {
				testHookPrivateDir = nil
				if moved = moveAway(t, tmp, tmp+".moved"); !moved {
					return
				}
				own = filepath.Join(tmp+".moved", scratchNamespace, filepath.Base(path))
				record, err := os.ReadFile(filepath.Join(own, ownerName))
				if err != nil {
					t.Fatal(err)
				}
				mkPrivate(t, tmp)
				mkPrivate(t, filepath.Join(tmp, scratchNamespace))
				mkPrivate(t, path)
				kept[filepath.Join(path, ownerName)] = string(record)
				kept[filepath.Join(path, "witness")] = "keep"
				kept[path+".lock"] = "keep-replacement-lock"
				for p, c := range kept {
					write(t, p, c)
				}
			}
			t.Cleanup(func() { testHookPrivateDir = nil })
			err := r.read(t, s)
			if !moved {
				if err != nil {
					t.Fatal(err)
				}
				wantEmpty(t, d)
				return
			}
			wantReplaced(t, err)
			wantFiles(t, kept)
			if _, err := os.Stat(filepath.Join(own, ownerName)); err != nil {
				t.Errorf("the read's own directory lost its owner file: %v", err)
			}
			if _, err := newScratch(t, d); apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Fatalf("unknown replacement: %v", err)
			}
			wantFiles(t, kept)
		})
	}
}

// Review 059 B1-R, the third counterexample: while a read uses its private
// directory, tmp, or the data directory above it, is moved away and its
// path made a link: to where it went, so that the private directory is
// still found below it, or to another directory holding one of the same
// name. The read fails and removes nothing; the link, the read's own
// directory and the other directory stay as they are. On Windows the move
// is refused (a junction cannot take the place of a directory held open).
func TestScratchRootRedirected(t *testing.T) {
	requireSafeGit(t)
	for _, c := range []struct {
		name            string
		data, elsewhere bool
	}{
		{"tmp to where it went", false, false},
		{"tmp elsewhere", false, true},
		{"the data directory to where it went", true, false},
	} {
		for _, r := range reads {
			t.Run(c.name+" in "+r.name, func(t *testing.T) {
				s, d := scratchIn(t)
				dir := d.Join(datadir.TmpDir)
				if c.data {
					dir = d.Path
				}
				target := dir + ".moved"
				var own, witness string
				moved := false
				testHookPrivateDir = func(path string) {
					testHookPrivateDir = nil
					if moved = moveAway(t, dir, dir+".moved"); !moved {
						return
					}
					own = filepath.Join(dir+".moved", scratchNamespace, filepath.Base(path))
					if c.data {
						own = filepath.Join(dir+".moved", datadir.TmpDir, scratchNamespace, filepath.Base(path))
					}
					if c.elsewhere {
						target = filepath.Join(d.Path, "elsewhere")
						mkPrivate(t, target)
						mkPrivate(t, filepath.Join(target, filepath.Base(path)))
						witness = filepath.Join(target, filepath.Base(path), "witness")
						write(t, witness, "keep")
					}
					link(t, target, dir)
				}
				t.Cleanup(func() { testHookPrivateDir = nil })
				err := r.read(t, s)
				if !moved {
					if err != nil {
						t.Fatal(err)
					}
					wantEmpty(t, d)
					return
				}
				wantReplaced(t, err)
				if got, err := os.Readlink(dir); err != nil || got != target {
					t.Errorf("the link is %q, %v; want it unchanged, to %s", got, err, target)
				}
				if _, err := os.Stat(filepath.Join(own, ownerName)); err != nil {
					t.Errorf("the read's own directory lost its owner file: %v", err)
				}
				if witness != "" {
					wantFiles(t, map[string]string{witness: "keep"})
				}
			})
		}
	}
}

// A private directory moved away as soon as it is made, before it is
// opened, and another put at its name: a directory holding a file, or a
// link to one. The read fails; neither is taken for the directory made,
// nothing is made in them or removed from them, and the directory made is
// kept where it went. Windows too: nothing holds the new directory yet.
// (An empty directory put there would be taken for the one made; see
// privdir.go.)
func TestPrivateDirectoryReplacedAsItIsMade(t *testing.T) {
	requireSafeGit(t)
	for _, c := range []struct {
		name string
		link bool
	}{{"by a directory", false}, {"by a link", true}} {
		for _, r := range reads {
			t.Run(c.name+" in "+r.name, func(t *testing.T) {
				s, d := scratchIn(t)
				var at, witness string
				testHookMkdir = func(path string) {
					testHookMkdir = nil
					at = path
					if !moveAway(t, path, path+".moved") {
						return
					}
					if c.link {
						target := filepath.Join(d.Path, "elsewhere")
						mkPrivate(t, target)
						link(t, target, path)
						witness = filepath.Join(target, "witness")
					} else {
						mkPrivate(t, path)
						witness = filepath.Join(path, "witness")
					}
					write(t, witness, "keep")
				}
				t.Cleanup(func() { testHookMkdir = nil })
				err := r.read(t, s)
				if witness == "" && runtime.GOOS == "windows" {
					if err != nil {
						t.Fatal(err)
					}
					wantEmpty(t, d)
					return
				}
				wantReplaced(t, err)
				wantFiles(t, map[string]string{witness: "keep"})
				if _, err := os.Lstat(at); err != nil {
					t.Errorf("what replaced the private directory was removed: %v", err)
				}
				if _, err := os.Lstat(filepath.Join(filepath.Dir(witness), ownerName)); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("an owner file was made in the replacement: %v", err)
				}
				if _, err := os.Stat(at + ".moved"); err != nil {
					t.Errorf("the directory made was removed: %v", err)
				}
			})
		}
	}
}

// The tmp directory itself moved away and another put in its place after
// the Scratch was made: no private directory is made in the replacement,
// and nothing in either is removed. On Windows the move is refused.
func TestScratchRootReplaced(t *testing.T) {
	s, d := scratchIn(t)
	root := d.Join(datadir.TmpDir)
	if !moveAway(t, root, root+".moved") {
		p, err := newPrivateDir(s)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.remove(); err != nil {
			t.Fatal(err)
		}
		return
	}
	mkPrivate(t, root)
	if _, err := newPrivateDir(s); apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "replac") {
		t.Errorf("newPrivateDir = %v; want a refusal", err)
	}
	wantEmpty(t, d)
}

// v3 retains unknown, copied, unmanifested and active content. Unlike the
// former cleanup, detecting an unknown object is an environment error.
func TestScratchRemovesOnlyItsOwnLeftovers(t *testing.T) {
	for _, kind := range []string{"crashed", "unknown-file", "copied-owner", "extra-content", "active"} {
		t.Run(kind, func(t *testing.T) {
			s, d := scratchIn(t)
			p, err := newPrivateDir(s)
			if err != nil {
				t.Fatal(err)
			}
			if err := p.write("index", []byte("private")); err != nil {
				t.Fatal(err)
			}
			path := p.path
			switch kind {
			case "unknown-file":
				p.close()
				write(t, filepath.Join(s.path, "unknown"), "keep")
			case "copied-owner":
				p.close()
				record, err := os.ReadFile(filepath.Join(path, ownerName))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(path, ownerName), d.Join("original-owner")); err != nil {
					t.Fatal(err)
				}
				write(t, filepath.Join(path, ownerName), string(record)) // new owner identity
			case "extra-content":
				p.close()
				write(t, filepath.Join(path, "unknown"), "keep")
			case "crashed":
				p.close()
			}
			next, err := newScratch(t, d)
			if kind == "crashed" {
				if err != nil {
					t.Fatal(err)
				}
				wantEmpty(t, d)
				return
			}
			if kind == "active" {
				if err != nil {
					t.Fatal(err)
				}
				if _, err := newPrivateDir(next); apperr.ExitCode(err) != apperr.ExitEnvironment {
					t.Fatalf("contention: %v", err)
				}
				if err := p.remove(); err != nil {
					t.Fatal(err)
				}
				return
			}
			if apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Fatalf("unknown content: %v", err)
			}
			wantFiles(t, map[string]string{filepath.Join(path, "index"): "private"})
		})
	}
}

// Review 052 N2, WP2.3 spec §4.3: after a read that succeeds, fails or is
// cancelled, nothing is left in tmp. A private directory that cannot be
// removed fails the read; one whose content cannot be removed keeps its
// owner file, unlocked, so that the next Scratch removes it; one that
// cannot itself be removed once its owner file is gone cannot be shown to
// be a crashed read's, and is kept. A missing or closed Scratch is a
// mistake of the caller.
func TestPrivateDirectoryLifecycle(t *testing.T) {
	requireSafeGit(t)
	s, d := scratchIn(t)
	super := subRepo(t)
	if got, err := Clean(ctx, s, super.Dir); err != nil || got != StateClean {
		t.Fatalf("Clean = %v, %v", got, err)
	}
	wantEmpty(t, d)
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
	wantEmpty(t, d)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := Clean(cancelled, s, super.Dir); err == nil {
		t.Error("a cancelled read succeeded")
	}
	if err := Log(cancelled, s, repo.Dir, shas[1], nil, func(Commit) error { return nil }); err == nil {
		t.Error("a cancelled log succeeded")
	}
	wantEmpty(t, d)

	failing := errors.New("cannot remove")
	testHookRemove = func(step, _ string) error {
		if step == "contents" {
			return failing
		}
		return nil
	}
	t.Cleanup(func() { testHookRemove = nil })
	got, err := Clean(ctx, s, baseRepo(t).Dir)
	testHookRemove = nil
	if apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("a directory that cannot be removed: Clean = %v, %v; want an environment error", got, err)
	}
	left := entries(t, d)
	if len(left) != 1 {
		t.Fatalf("after a failed removal: %v; want the directory", left)
	}
	if _, err := os.Stat(filepath.Join(d.Join(filepath.Join(datadir.TmpDir, scratchNamespace)), left[0], ownerName)); err != nil {
		t.Errorf("its owner file: %v", err)
	}
	if _, err := newScratch(t, d); err != nil {
		t.Fatal(err)
	}
	wantEmpty(t, d)

	testHookRemove = func(step, _ string) error {
		if step == "entry" {
			return failing
		}
		return nil
	}
	_, err = Clean(ctx, s, baseRepo(t).Dir)
	testHookRemove = nil
	if apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("a directory whose own entry cannot be removed: Clean = %v; want an environment error", err)
	}
	if _, err := newScratch(t, d); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("ownerless leftover: %v", err)
	}
	left = entries(t, d)
	if len(left) != 1 {
		t.Fatalf("after a failed removal of the entry: %v; want the empty directory, kept", left)
	}
	if es, err := os.ReadDir(filepath.Join(d.Join(filepath.Join(datadir.TmpDir, scratchNamespace)), left[0])); err != nil || len(es) != 0 {
		t.Errorf("the directory kept: %v, %v; want it empty", es, err)
	}

	if _, err := Clean(ctx, nil, super.Dir); apperr.ExitCode(err) != apperr.ExitInternal {
		t.Errorf("Clean without a Scratch = %v", err)
	}
	if _, err := Tips(ctx, nil, repo.Dir); apperr.ExitCode(err) != apperr.ExitInternal {
		t.Errorf("Tips without a Scratch = %v", err)
	}
	closed, _ := scratchIn(t)
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Clean(ctx, closed, super.Dir); apperr.ExitCode(err) != apperr.ExitInternal {
		t.Errorf("Clean with a closed Scratch = %v", err)
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
	if err != nil || len(es) < 6 {
		t.Fatalf("the private directory holds %d entries: %v", len(es), err)
	}
	problems, err := datadir.Check(d)
	if err != nil || len(problems) != 0 {
		t.Errorf("not private: %v, %v", problems, err)
	}
}
