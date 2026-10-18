package gitx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// old is the time the objects of a fixture are set to before a read, so
// that a read that touches one shows (the spec's S experiments).
var old = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

// ageObjects sets every file and directory under the object directories
// in roots to old.
func ageObjects(t *testing.T, roots ...string) {
	t.Helper()
	for _, root := range roots {
		err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && d.Name() == "objects" || strings.Contains(filepath.ToSlash(p), "/objects/") {
				return os.Chtimes(p, old, old)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// forget removes a marker the fixture's own git commands may have written
// while it was set up: only a run during Clean counts.
func forget(t *testing.T, marker string) {
	t.Helper()
	if err := os.Remove(marker); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}

// unchanged snapshots the git directories of a repository (objects aged
// first) and returns a check that nothing in them changed: no file's
// content or modification time, none created or removed.
func unchanged(t *testing.T, gitDirs ...string) func() {
	t.Helper()
	ageObjects(t, gitDirs...)
	var before []map[string]string
	for _, d := range gitDirs {
		before = append(before, testutil.SettledSnapshot(t, d))
	}
	return func() {
		t.Helper()
		for i, d := range gitDirs {
			testutil.AssertTreeUnchanged(t, d, before[i])
		}
	}
}

// baseRepo is a committed repository: README.md, dir/x.txt and a
// .gitignore that ignores *.log.
func baseRepo(t *testing.T) *testutil.GitRepo {
	t.Helper()
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("README.md", "readme\n")
	repo.WriteFile("dir/x.txt", "x\n")
	repo.WriteFile(".gitignore", "*.log\n")
	repo.Commit("init")
	return repo
}

func wantState(t *testing.T, name, dir string, want State) {
	t.Helper()
	got, err := Clean(ctx, dir)
	if err != nil || got != want {
		t.Errorf("%s: Clean = %v, %v; want %v", name, got, err, want)
	}
}

func (s State) String() string {
	return [...]string{"clean", "dirty", "undecidable"}[s]
}

func touch(t *testing.T, p string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(p, at, at); err != nil {
		t.Fatal(err)
	}
}

// data-model §1 (D1), WP2.3 spec §4.3 C2 to C8: the state the definition's
// status gives, read without changing anything in the repository.
func TestCleanStates(t *testing.T) {
	requireSafeGit(t)
	for _, c := range []struct {
		name   string
		change func(t *testing.T, repo *testutil.GitRepo)
		want   State
	}{
		{"nothing changed", func(*testing.T, *testutil.GitRepo) {}, StateClean},
		{"C8 only a modification time", func(t *testing.T, r *testutil.GitRepo) {
			touch(t, filepath.Join(r.Dir, "dir", "x.txt"), time.Now().Add(time.Hour))
		}, StateClean},
		{"C2 a modified file", func(_ *testing.T, r *testutil.GitRepo) { r.WriteFile("dir/x.txt", "changed\n") }, StateDirty},
		{"C3 an untracked file", func(_ *testing.T, r *testutil.GitRepo) { r.WriteFile("new/deep/u.txt", "u\n") }, StateDirty},
		{"C4 an ignored file only", func(_ *testing.T, r *testutil.GitRepo) { r.WriteFile("build.log", "l\n") }, StateClean},
		{"C5 a staged change", func(_ *testing.T, r *testutil.GitRepo) {
			r.WriteFile("dir/x.txt", "staged\n")
			r.Git("add", "dir/x.txt")
		}, StateDirty},
		{"C6 a deleted file", func(t *testing.T, r *testutil.GitRepo) {
			if err := os.Remove(filepath.Join(r.Dir, "README.md")); err != nil {
				t.Fatal(err)
			}
		}, StateDirty},
		{"Specmetry's own .specmetry/", func(_ *testing.T, r *testutil.GitRepo) { r.WriteFile(".specmetry/marker.json", "{}\n") }, StateClean},
		{"an untracked file beside .specmetry/", func(_ *testing.T, r *testutil.GitRepo) {
			r.WriteFile(".specmetry/marker.json", "{}\n")
			r.WriteFile(".specmetry-not/x", "x\n")
		}, StateDirty},
		{"an exclude in info/exclude", func(_ *testing.T, r *testutil.GitRepo) {
			r.WriteFile(".git/info/exclude", "secret.txt\n")
			r.WriteFile("secret.txt", "s\n")
		}, StateClean},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := baseRepo(t)
			c.change(t, repo)
			check := unchanged(t, filepath.Join(repo.Dir, ".git"))
			wantState(t, c.name, repo.Dir, c.want)
			check()
		})
	}
}

// markerProgram puts the marker stand-in in a new directory and returns
// its path, for a configuration value; it appends to the marker that
// SPECMETRY_FAKE_REMOTE_HELPER_MARKER names when run.
func markerProgram(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	marker := os.Getenv("SPECMETRY_FAKE_REMOTE_HELPER_MARKER")
	testutil.FakeRemoteHelper(t, dir, "marker")
	if marker != "" {
		t.Setenv("SPECMETRY_FAKE_REMOTE_HELPER_MARKER", marker)
	}
	exe := filepath.Join(dir, "git-remote-marker")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	return filepath.ToSlash(exe)
}

// filterRepo is baseRepo with f.txt and g.txt under filter=x, committed
// before any driver exists and settled (every tracked file older than the
// index, whose entries match them: nothing racy or stat-changed); then the
// local configuration defines driver x with a clean and a process filter
// that are the marker stand-in.
func filterRepo(t *testing.T) (repo *testutil.GitRepo, marker string) {
	t.Helper()
	dir := t.TempDir()
	marker = testutil.FakeRemoteHelper(t, dir, "filter")
	exe := filepath.ToSlash(filepath.Join(dir, "git-remote-filter"))
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	repo = baseRepo(t)
	repo.WriteFile(".gitattributes", "f.txt filter=x\ng.txt filter=x\n")
	repo.WriteFile("f.txt", "raw\n")
	repo.WriteFile("g.txt", "raw\n")
	repo.WriteFile("plain.txt", "p\n")
	repo.Commit("filtered")
	at := time.Now().Add(-time.Hour)
	for _, f := range strings.Split(repo.Git("ls-files"), "\n") {
		touch(t, filepath.Join(repo.Dir, filepath.FromSlash(f)), at)
	}
	repo.Git("update-index", "--refresh")
	repo.Git("config", "filter.x.clean", exe)
	repo.Git("config", "filter.x.process", exe)
	return repo, marker
}

// Review 050 B3, WP2.3 spec §4.3 C1, C7: the reading runs no program the
// repository configures. With a clean and a process filter configured
// after f.txt was committed and f.txt's modification time changed, the
// definition's status runs them (the control: plain git status writes
// the marker); Clean does not, and judges the state undecidable (step 5:
// a filtered file whose stat changed).
func TestCleanRunsNoFilter(t *testing.T) {
	requireSafeGit(t)
	control, controlMarker := filterRepo(t)
	touch(t, filepath.Join(control.Dir, "f.txt"), time.Now().Add(time.Hour))
	_, _ = control.GitResult("status", "--porcelain")
	if !ranAt(controlMarker) {
		t.Fatal("control: plain git status did not run the filter, so this test shows nothing")
	}

	repo, marker := filterRepo(t)
	touch(t, filepath.Join(repo.Dir, "f.txt"), time.Now().Add(time.Hour))
	forget(t, marker)
	check := unchanged(t, filepath.Join(repo.Dir, ".git"))
	wantState(t, "C1", repo.Dir, StateUndecidable)
	check()
	if ranAt(marker) {
		t.Error("Clean ran the repository's filter")
	}
}

// Review 052 B3-R1, WP2.3 spec §4.3 step 5 (F1 to F7): when a filter
// driver is defined in the local configuration, a filtered file the
// definition's status would have to compare (its stat changed, or racy: as
// new as the index or newer) makes the state undecidable, never clean;
// otherwise the private reading agrees with the definition.
func TestCleanFilterRule(t *testing.T) {
	requireSafeGit(t)
	future := time.Now().Add(time.Hour)
	for _, c := range []struct {
		name   string
		change func(t *testing.T, repo *testutil.GitRepo)
		want   State
	}{
		{"F1 stat changed, same raw content", func(t *testing.T, r *testutil.GitRepo) {
			touch(t, filepath.Join(r.Dir, "f.txt"), future)
		}, StateUndecidable},
		{"F2 stat changed, other raw content", func(_ *testing.T, r *testutil.GitRepo) {
			r.WriteFile("f.txt", "RAW\n")
		}, StateUndecidable},
		{"F3 stat as in the index, not racy", func(*testing.T, *testutil.GitRepo) {}, StateClean},
		{"F4 racy", func(t *testing.T, r *testutil.GitRepo) {
			idx, err := os.Stat(filepath.Join(r.Dir, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			touch(t, filepath.Join(r.Dir, "f.txt"), idx.ModTime())
		}, StateUndecidable},
		{"F7 racy: the entry as the file, the index no newer", func(t *testing.T, r *testutil.GitRepo) {
			// Refreshed before the driver exists, so the index has f.txt's
			// stat; diff-files compares a racy entry's raw content only
			// and does not list it. Only the racy rule finds it.
			clean := r.Git("config", "filter.x.clean")
			process := r.Git("config", "filter.x.process")
			r.Git("config", "--remove-section", "filter.x")
			at := time.Now().Truncate(time.Second)
			touch(t, filepath.Join(r.Dir, "f.txt"), at)
			r.Git("update-index", "--refresh")
			touch(t, filepath.Join(r.Dir, ".git", "index"), at)
			r.Git("config", "filter.x.clean", clean)
			r.Git("config", "filter.x.process", process)
			if out, _ := r.GitResult("diff-files", "--name-only"); out != "" {
				t.Fatalf("diff-files lists %q: the case does not hold", out)
			}
		}, StateUndecidable},
		{"F5 a driver the local configuration does not define", func(t *testing.T, r *testutil.GitRepo) {
			r.Git("config", "--remove-section", "filter.x")
			touch(t, filepath.Join(r.Dir, "f.txt"), future)
		}, StateClean},
		{"F6 only an unfiltered file changed", func(_ *testing.T, r *testutil.GitRepo) {
			r.WriteFile("plain.txt", "changed\n")
		}, StateDirty},
		{"a required driver without clean or process", func(t *testing.T, r *testutil.GitRepo) {
			r.Git("config", "--remove-section", "filter.x")
			r.Git("config", "filter.x.required", "true")
			touch(t, filepath.Join(r.Dir, "f.txt"), future)
		}, StateUndecidable},
		{"a driver with only a smudge filter", func(t *testing.T, r *testutil.GitRepo) {
			r.Git("config", "--remove-section", "filter.x")
			r.Git("config", "filter.x.smudge", "cat")
			touch(t, filepath.Join(r.Dir, "f.txt"), future)
		}, StateClean},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, marker := filterRepo(t)
			c.change(t, repo)
			forget(t, marker)
			check := unchanged(t, filepath.Join(repo.Dir, ".git"))
			wantState(t, c.name, repo.Dir, c.want)
			check()
			if ranAt(marker) {
				t.Error("Clean ran the repository's filter")
			}
		})
	}
}

// subRepo makes a superproject with an initialized submodule at sub (its
// git directory absorbed into .git/modules/sub, the .git a gitfile) and
// returns both.
func subRepo(t *testing.T) (super, sub *testutil.GitRepo) {
	t.Helper()
	sub = testutil.NewGitRepo(t)
	sub.WriteFile("a/s.txt", "s\n")
	sub.WriteFile("b/t.txt", "t\n")
	sub.Commit("sub")
	super = baseRepo(t)
	super.Git("-c", "protocol.file.allow=always", "submodule", "add", "-q", filepath.ToSlash(sub.Dir), "sub")
	super.Commit("add sub")
	if fi, err := os.Lstat(filepath.Join(super.Dir, "sub", ".git")); err != nil || !fi.Mode().IsRegular() {
		t.Fatalf("the submodule's .git is not a gitfile: %v", err)
	}
	return super, sub
}

// D1 and WP2.3 spec §4.3 step 4 (C9 to C11, N1-g): an initialized
// submodule counts: its HEAD against the gitlink, and its own state, read
// the same way, its git directory resolved from its gitfile. No process
// runs in it (C9: its filters do not run). The superproject's
// --ignore-submodules=all hides more than a submodule's own changes: a
// staged gitlink, a submodule directory removed or replaced by a file,
// a gitlink removed from the index; the definition (=none) shows them, and
// so does Clean.
func TestCleanSubmodules(t *testing.T) {
	requireSafeGit(t)
	for _, c := range []struct {
		name   string
		change func(t *testing.T, super *testutil.GitRepo)
		want   State
	}{
		{"nothing changed", func(*testing.T, *testutil.GitRepo) {}, StateClean},
		{"C10 the submodule's HEAD moved", func(_ *testing.T, s *testutil.GitRepo) {
			s.GitIn(filepath.Join(s.Dir, "sub"), "commit", "-q", "--allow-empty", "-m", "more")
		}, StateDirty},
		{"C11 the same with diff.ignoreSubmodules=all", func(_ *testing.T, s *testutil.GitRepo) {
			s.Git("config", "diff.ignoreSubmodules", "all")
			s.GitIn(filepath.Join(s.Dir, "sub"), "commit", "-q", "--allow-empty", "-m", "more")
		}, StateDirty},
		{"the submodule's HEAD moved and staged", func(_ *testing.T, s *testutil.GitRepo) {
			s.GitIn(filepath.Join(s.Dir, "sub"), "commit", "-q", "--allow-empty", "-m", "more")
			s.Git("add", "sub")
		}, StateDirty},
		{"a modified file in the submodule", func(_ *testing.T, s *testutil.GitRepo) { s.WriteFile("sub/a/s.txt", "changed\n") }, StateDirty},
		{"an untracked file in the submodule", func(_ *testing.T, s *testutil.GitRepo) { s.WriteFile("sub/new.txt", "n\n") }, StateDirty},
		{"the submodule's directory removed", func(t *testing.T, s *testutil.GitRepo) {
			if err := os.RemoveAll(filepath.Join(s.Dir, "sub")); err != nil {
				t.Fatal(err)
			}
		}, StateDirty},
		{"the submodule replaced by a file", func(t *testing.T, s *testutil.GitRepo) {
			if err := os.RemoveAll(filepath.Join(s.Dir, "sub")); err != nil {
				t.Fatal(err)
			}
			s.WriteFile("sub", "a file\n")
		}, StateDirty},
		{"the gitlink removed from the index, its directory too", func(t *testing.T, s *testutil.GitRepo) {
			s.Git("rm", "-q", "--cached", "sub")
			if err := os.RemoveAll(filepath.Join(s.Dir, "sub")); err != nil {
				t.Fatal(err)
			}
		}, StateDirty},
		{"deinitialized: an empty directory", func(_ *testing.T, s *testutil.GitRepo) {
			s.Git("submodule", "deinit", "-q", "-f", "sub")
		}, StateClean},
		{"C9 a filter in the submodule, only a modification time changed", func(t *testing.T, s *testutil.GitRepo) {
			subDir := filepath.Join(s.Dir, "sub")
			s.GitIn(subDir, "config", "filter.x.clean", markerProgram(t))
			s.WriteFile("sub/.gitattributes", "a/s.txt filter=x\n")
			s.GitIn(subDir, "add", ".gitattributes")
			s.GitIn(subDir, "commit", "-q", "-m", "attrs")
			s.Git("add", "sub")
			s.Git("commit", "-q", "-m", "bump")
			touch(t, filepath.Join(subDir, "a", "s.txt"), time.Now().Add(time.Hour))
		}, StateUndecidable},
	} {
		t.Run(c.name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "ran")
			t.Setenv("SPECMETRY_FAKE_REMOTE_HELPER_MARKER", marker)
			super, _ := subRepo(t)
			c.change(t, super)
			forget(t, marker)
			check := unchanged(t, filepath.Join(super.Dir, ".git"))
			wantState(t, c.name, super.Dir, c.want)
			check()
			if ranAt(marker) {
				t.Error("a filter of the submodule ran")
			}
		})
	}
}

// The definition's status (--ignore-submodules=none) on the cases above
// agrees with what TestCleanSubmodules expects, on this git: the
// reference the expectations are taken from (scratchpad experiments
// sub.sh and sub2.sh, git 2.56 and 2.43).
func TestCleanSubmodulesAgreeWithTheDefinition(t *testing.T) {
	requireSafeGit(t)
	definition := func(s *testutil.GitRepo) string {
		return s.Git("status", "--porcelain", "--untracked-files=all", "--ignore-submodules=none", "--", ".", ":(exclude).specmetry")
	}
	super, _ := subRepo(t)
	super.GitIn(filepath.Join(super.Dir, "sub"), "commit", "-q", "--allow-empty", "-m", "more")
	super.Git("add", "sub")
	if got := definition(super); got != "M  sub" {
		t.Errorf("a staged gitlink: %q", got)
	}
	if hidden := super.Git("status", "--porcelain", "--ignore-submodules=all"); hidden != "" {
		t.Logf("this git's --ignore-submodules=all shows a staged gitlink: %q", hidden)
	}
	super, _ = subRepo(t)
	if err := os.RemoveAll(filepath.Join(super.Dir, "sub")); err != nil {
		t.Fatal(err)
	}
	if got := definition(super); got != "D sub" && got != " D sub" {
		t.Errorf("a removed submodule directory: %q", got)
	}
	super, _ = subRepo(t)
	super.Git("submodule", "deinit", "-q", "-f", "sub")
	if got := definition(super); got != "" {
		t.Errorf("a deinitialized submodule: %q", got)
	}
}

// Review 055 B3-R2, WP2.3 spec §4.3 step 2 (S15 to S18b, X6, P2): a sparse
// index (sparse directory entries, sdir) is judged undecidable by parsing
// the private copy before any git command reads it: with a directory at
// the sparse path, git reading it would expand the index and rewrite the
// modification times of the repository's object files (the control,
// reading in the old order, shows it). After the gate no git command runs,
// in the superproject or in its submodules, and nothing in the repository
// changes. A submodule's own gate comes before its own commands.
func TestCleanSparseGate(t *testing.T) {
	requireSafeGit(t)
	withOut := func(_ *testing.T, repo *testutil.GitRepo) {
		repo.WriteFile("b/untracked.txt", "u\n") // b/ is the sparse directory
	}
	// The control: the old order (status on the private copy, no gate).
	control, _, _ := sparseRepo(t)
	withOut(t, control)
	gitDir := filepath.Join(control.Dir, ".git")
	ageObjects(t, gitDir)
	before := testutil.SettledSnapshot(t, gitDir)
	p, err := prepare(ctx, control.Dir)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = call{label: "status", dir: control.Dir, objects: true, args: p.args("status", "--porcelain")}.run(ctx)
	if err := p.close(); err != nil {
		t.Fatal(err)
	}
	if d := testutil.DiffSnapshots(before, testutil.SnapshotTree(t, gitDir)); d == "" {
		t.Log("control: this git did not touch the objects reading the sparse index in the old order")
	} else {
		t.Logf("control: reading in the old order changed %d entries of .git", strings.Count(d, "\n")+1)
	}

	for _, c := range []struct {
		name string
		out  bool
	}{{"S15/S16 a directory at the sparse path", true}, {"S17 none", false}} {
		t.Run(c.name, func(t *testing.T) {
			repo, _, _ := sparseRepo(t)
			if c.out {
				withOut(t, repo)
			}
			dir, env, calls := testutil.PassthroughGit(t, testutil.PassAsIs)
			useGit(t, dir, env)
			check := unchanged(t, filepath.Join(repo.Dir, ".git"))
			wantState(t, c.name, repo.Dir, StateUndecidable)
			check()
			for _, call := range calls() {
				if readsIndex(call.Args) {
					t.Errorf("a command that reads the index ran after the gate: %q", call.Args)
				}
			}
		})
	}

	t.Run("S18b a sparse submodule", func(t *testing.T) {
		super, _ := subRepo(t)
		subDir := filepath.Join(super.Dir, "sub")
		super.GitIn(subDir, "sparse-checkout", "init", "--cone", "--sparse-index")
		super.GitIn(subDir, "sparse-checkout", "set", "a")
		if b, err := os.ReadFile(filepath.Join(super.Dir, ".git", "modules", "sub", "index")); err != nil || !strings.Contains(string(b), "sdir") {
			t.Skip("this git keeps no sparse index")
		}
		super.WriteFile("sub/b/untracked.txt", "u\n")
		dir, env, calls := testutil.PassthroughGit(t, testutil.PassAsIs)
		useGit(t, dir, env)
		check := unchanged(t, filepath.Join(super.Dir, ".git"))
		wantState(t, "S18b", super.Dir, StateUndecidable)
		check()
		statuses := 0
		for _, call := range calls() {
			if slices.Contains(call.Args, "status") {
				statuses++
			}
		}
		if statuses != 1 {
			t.Errorf("status ran %d times, want once (the superproject's)", statuses)
		}
	})

	t.Run("X6 the superproject's diff-files and a sparse submodule", func(t *testing.T) {
		marker := filepath.Join(t.TempDir(), "ran")
		t.Setenv("SPECMETRY_FAKE_REMOTE_HELPER_MARKER", marker)
		super, _ := subRepo(t)
		super.WriteFile(".gitattributes", "f.txt filter=x\n")
		super.WriteFile("f.txt", "raw\n")
		super.Commit("filtered")
		super.Git("config", "filter.x.clean", markerProgram(t))
		subDir := filepath.Join(super.Dir, "sub")
		super.GitIn(subDir, "sparse-checkout", "init", "--cone", "--sparse-index")
		super.GitIn(subDir, "sparse-checkout", "set", "a")
		if b, err := os.ReadFile(filepath.Join(super.Dir, ".git", "modules", "sub", "index")); err != nil || !strings.Contains(string(b), "sdir") {
			t.Skip("this git keeps no sparse index")
		}
		super.WriteFile("sub/b/untracked.txt", "u\n")
		forget(t, marker)
		check := unchanged(t, filepath.Join(super.Dir, ".git"))
		wantState(t, "X6", super.Dir, StateUndecidable)
		check()
		if ranAt(marker) {
			t.Error("Clean ran the repository's filter")
		}
	})

	t.Run("S19 a normal index", func(t *testing.T) {
		repo := baseRepo(t)
		repo.WriteFile("dir/x.txt", "changed\n")
		repo.WriteFile("dir/u.txt", "u\n")
		dir, env, calls := testutil.PassthroughGit(t, testutil.PassAsIs)
		useGit(t, dir, env)
		check := unchanged(t, filepath.Join(repo.Dir, ".git"))
		wantState(t, "S19", repo.Dir, StateDirty)
		check()
		if !slices.ContainsFunc(calls(), func(c testutil.FakeGitCall) bool { return slices.Contains(c.Args, "status") }) {
			t.Error("status did not run")
		}
	})

	t.Run("S11 a sparse checkout without a sparse index", func(t *testing.T) {
		repo := baseRepo(t)
		repo.WriteFile("b/y.txt", "y\n")
		repo.Commit("b")
		repo.Git("sparse-checkout", "init", "--cone", "--no-sparse-index")
		repo.Git("sparse-checkout", "set", "dir")
		repo.WriteFile("b/untracked.txt", "u\n")
		check := unchanged(t, filepath.Join(repo.Dir, ".git"))
		wantState(t, "S11", repo.Dir, StateDirty)
		check()
	})
}

// readsIndex reports whether a git call is one of Clean's commands that
// read an index (the probes run elsewhere, without --git-dir).
func readsIndex(args []string) bool {
	for _, sub := range []string{"status", "diff-files", "diff-index", "check-attr"} {
		if slices.Contains(args, sub) {
			return true
		}
	}
	return slices.Contains(args, "ls-files") && slices.Contains(args, "--stage")
}

// S12 to S14: the whole reading on a normal index with an untracked, a
// modified and a filtered file, a split index and an untracked cache
// leaves the repository as it was.
func TestCleanLayouts(t *testing.T) {
	requireSafeGit(t)
	t.Run("S12", func(t *testing.T) {
		repo, marker := filterRepo(t)
		repo.WriteFile("dir/x.txt", "changed\n")
		repo.WriteFile("u.txt", "u\n")
		touch(t, filepath.Join(repo.Dir, "f.txt"), time.Now().Add(time.Hour))
		forget(t, marker)
		check := unchanged(t, filepath.Join(repo.Dir, ".git"))
		wantState(t, "S12", repo.Dir, StateUndecidable)
		check()
		if ranAt(marker) {
			t.Error("Clean ran the repository's filter")
		}
	})
	t.Run("S13 a split index", func(t *testing.T) {
		repo := baseRepo(t)
		repo.Git("update-index", "--split-index")
		repo.WriteFile("dir/x.txt", "changed\n")
		repo.Git("add", "dir/x.txt")
		check := unchanged(t, filepath.Join(repo.Dir, ".git"))
		wantState(t, "S13", repo.Dir, StateDirty)
		check()
		repo.Git("commit", "-q", "-m", "x")
		wantState(t, "S13 committed", repo.Dir, StateClean)
	})
	t.Run("S14 an untracked cache", func(t *testing.T) {
		repo := baseRepo(t)
		repo.Git("config", "core.untrackedCache", "true")
		repo.Git("update-index", "--untracked-cache")
		repo.WriteFile("u.txt", "u\n")
		repo.Git("status", "--porcelain")
		if b, _ := os.ReadFile(filepath.Join(repo.Dir, ".git", "index")); !strings.Contains(string(b), "UNTR") {
			t.Skip("git wrote no untracked cache")
		}
		check := unchanged(t, filepath.Join(repo.Dir, ".git"))
		wantState(t, "S14", repo.Dir, StateDirty)
		check()
	})
	t.Run("a linked worktree", func(t *testing.T) {
		repo := baseRepo(t)
		wt := repo.AddWorktree("side")
		wantState(t, "linked, clean", wt, StateClean)
		repo.WriteFile("../wt-side/dir/x.txt", "changed\n")
		check := unchanged(t, filepath.Join(repo.Dir, ".git"))
		wantState(t, "linked, modified", wt, StateDirty)
		check()
		wantState(t, "the main worktree", repo.Dir, StateClean)
	})
	t.Run("extensions.worktreeConfig", func(t *testing.T) {
		repo := baseRepo(t)
		repo.Git("config", "extensions.worktreeConfig", "true")
		repo.Git("config", "--worktree", "core.autocrlf", "false")
		wantState(t, "worktreeConfig", repo.Dir, StateClean)
	})
}

// Review 057 N1: a repository without an index is read as one with an
// empty index: a new repository is clean, with an untracked file dirty;
// one with commits whose index is gone is dirty (the definition shows its
// files deleted from the index and untracked).
func TestCleanWithoutAnIndex(t *testing.T) {
	requireSafeGit(t)
	repo := testutil.NewGitRepo(t)
	if _, err := os.Stat(filepath.Join(repo.Dir, ".git", "index")); err == nil {
		t.Fatal("a new repository has an index")
	}
	wantState(t, "N1-u a new repository", repo.Dir, StateClean)
	repo.WriteFile("u.txt", "u\n")
	wantState(t, "a new repository with an untracked file", repo.Dir, StateDirty)

	repo = baseRepo(t)
	if err := os.Remove(filepath.Join(repo.Dir, ".git", "index")); err != nil {
		t.Fatal(err)
	}
	wantState(t, "commits, no index", repo.Dir, StateDirty)
}

// WP2.3 spec §4.3 step 6, review 057 §三 4, S20: an index that cannot be
// read completely, or a shared index that is missing, is an environment
// error with no git command run on it; the repository is not repaired or
// changed.
func TestCleanFailures(t *testing.T) {
	requireSafeGit(t)
	for _, c := range []struct {
		name   string
		damage func(t *testing.T, repo *testutil.GitRepo)
	}{
		{"S20 a truncated index", func(t *testing.T, r *testutil.GitRepo) {
			p := filepath.Join(r.Dir, ".git", "index")
			if err := os.Truncate(p, 30); err != nil {
				t.Fatal(err)
			}
		}},
		{"a zero-byte index", func(t *testing.T, r *testutil.GitRepo) {
			if err := os.WriteFile(filepath.Join(r.Dir, ".git", "index"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"a split index without its shared index", func(t *testing.T, r *testutil.GitRepo) {
			r.Git("update-index", "--split-index")
			shared, _ := filepath.Glob(filepath.Join(r.Dir, ".git", "sharedindex.*"))
			for _, s := range shared {
				if err := os.Remove(s); err != nil {
					t.Fatal(err)
				}
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := baseRepo(t)
			c.damage(t, repo)
			dir, env, calls := testutil.PassthroughGit(t, testutil.PassAsIs)
			useGit(t, dir, env)
			check := unchanged(t, filepath.Join(repo.Dir, ".git"))
			if got, err := Clean(ctx, repo.Dir); apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Errorf("Clean = %v, %v; want an environment error", got, err)
			}
			check()
			for _, call := range calls() {
				if readsIndex(call.Args) {
					t.Errorf("a command that reads the index ran: %q", call.Args)
				}
			}
		})
	}
	t.Run("SHA-256", func(t *testing.T) {
		repo := testutil.NewSHA256GitRepo(t)
		repo.WriteFile("a.txt", "a\n")
		repo.Commit("a")
		if got, err := Clean(ctx, repo.Dir); apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Errorf("Clean = %v, %v; want an environment error (TD-26)", got, err)
		}
	})
	t.Run("a git that would fetch", func(t *testing.T) {
		repo := baseRepo(t)
		dir, env, calls := testutil.PassthroughGit(t, testutil.PassWithoutNoLazyFetch)
		useGit(t, dir, env)
		if got, err := Clean(ctx, repo.Dir); apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "GIT_NO_LAZY_FETCH") {
			t.Errorf("Clean = %v, %v; want the object reads refused", got, err)
		}
		for _, call := range calls() {
			if readsIndex(call.Args) {
				t.Errorf("a command that reads the index ran: %q", call.Args)
			}
		}
	})
}

// privateDirs lists the private directories in the temporary directory.
func privateDirs(t *testing.T) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(os.TempDir(), privatePrefix+"*"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// isolateTemp gives the test a temporary directory of its own.
func isolateTemp(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	for _, v := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(v, tmp)
	}
	if os.TempDir() != tmp {
		t.Skipf("the temporary directory is %s, not %s", os.TempDir(), tmp)
	}
}

// Review 052 N2, WP2.3 spec §4.3 "the temporary directory's lifecycle":
// the private directories are gone after a read that succeeds, fails or is
// cancelled; one that cannot be removed fails the read; the next read
// removes the ones crashed processes left, and leaves those in use.
func TestCleanPrivateDirectories(t *testing.T) {
	requireSafeGit(t)
	isolateTemp(t)
	super, _ := subRepo(t)
	wantState(t, "a read that succeeds", super.Dir, StateClean)
	if left := privateDirs(t); len(left) != 0 {
		t.Errorf("left after success: %v", left)
	}
	broken := baseRepo(t)
	if err := os.Truncate(filepath.Join(broken.Dir, ".git", "index"), 30); err != nil {
		t.Fatal(err)
	}
	if _, err := Clean(ctx, broken.Dir); err == nil {
		t.Error("a truncated index was read")
	}
	if left := privateDirs(t); len(left) != 0 {
		t.Errorf("left after failure: %v", left)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := Clean(cancelled, super.Dir); err == nil {
		t.Error("a cancelled read succeeded")
	}
	if left := privateDirs(t); len(left) != 0 {
		t.Errorf("left after cancellation: %v", left)
	}

	failing := errors.New("cannot remove")
	removeAll = func(string) error { return failing }
	got, err := Clean(ctx, super.Dir)
	removeAll = os.RemoveAll
	if apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Errorf("a directory that cannot be removed: Clean = %v, %v; want an environment error", got, err)
	}
	if len(privateDirs(t)) == 0 {
		t.Fatal("the directory that could not be removed is not there")
	}

	// Left behind by a process that crashed: unlocked, with a lock file
	// (the one above), and an old one without a lock file. In use: one
	// locked by a live process (this one), and a new one without a lock
	// file.
	stale := privateDirs(t)
	lockless := filepath.Join(os.TempDir(), privatePrefix+"lockless")
	if err := os.Mkdir(lockless, 0o700); err != nil {
		t.Fatal(err)
	}
	touch(t, lockless, time.Now().Add(-2*staleUnlocked))
	fresh := filepath.Join(os.TempDir(), privatePrefix+"fresh")
	if err := os.Mkdir(fresh, 0o700); err != nil {
		t.Fatal(err)
	}
	held, err := newPrivateDir()
	if err != nil {
		t.Fatal(err)
	}
	// A file in it besides its lock: on Windows the open lock alone keeps
	// the directory from being removed, not its other files.
	inUse := held.join("in-use")
	if err := os.WriteFile(inUse, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	wantState(t, "a read after a crash", super.Dir, StateClean)
	left := privateDirs(t)
	for _, d := range append(stale, lockless) {
		if slices.Contains(left, d) {
			t.Errorf("not removed: %s", filepath.Base(d))
		}
	}
	for _, d := range []string{fresh, held.path} {
		if !slices.Contains(left, d) {
			t.Errorf("removed while it may be in use: %s", filepath.Base(d))
		}
	}
	if _, err := os.Stat(inUse); err != nil {
		t.Errorf("a file of a directory in use was removed: %v", err)
	}
	if err := held.remove(); err != nil {
		t.Fatal(err)
	}
}
