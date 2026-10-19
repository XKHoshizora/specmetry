package gitx

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// No real scan read has a private guard path. Recreate Review 060's exact
// before-Git/after-Git roundtrip: a pathname mutant would silently cut history;
// the approved explicit-empty backend reads the whole graph instead.
func TestScanEmptyOverrides(t *testing.T) {
	requireSafeGit(t)
	repo, ids := chain(t, 2)
	s, d := scratchIn(t)
	if err := os.MkdirAll(filepath.Join(repo.Dir, ".git", "info"), 0o700); err != nil {
		t.Fatal(err)
	}
	dir, env, calls := testutil.PassthroughGit(t, testutil.PassAsIs)
	useGit(t, dir, env)
	first := true
	testHookScanCall = func(c call) func() {
		if !slices.Contains(c.args, "log") {
			return nil
		}
		var paths []string
		for _, name := range []string{"GIT_GRAFT_FILE", "GIT_SHALLOW_FILE"} {
			found := false
			for _, kv := range c.env {
				if key, value, ok := strings.Cut(kv, "="); ok && key == name {
					found = true
					if value != "" {
						paths = append(paths, value)
					}
				}
			}
			if !found {
				t.Errorf("missing explicit %s", name)
			}
		}
		if len(paths) == 0 {
			paths = []string{filepath.Join(repo.Dir, ".git", "info", "grafts"), filepath.Join(repo.Dir, ".git", "shallow")}
		}
		// Changing BOTH underlying files defeats an implementation that clears
		// only one override. Remove them before the next caller check.
		for _, p := range paths {
			write(t, p, ids[1]+"\n")
		}
		first = false
		return func() {
			for _, p := range paths {
				_ = os.Remove(p)
			}
		}
	}
	t.Cleanup(func() { testHookScanCall = nil })
	var got []string
	if err := Log(ctx, s, repo.Dir, ids[1], nil, func(c Commit) error { got = append(got, c.SHA); return nil }); err != nil {
		t.Fatal(err)
	}
	if first || !slices.Equal(got, []string{ids[1], ids[0]}) {
		t.Fatalf("roundtrip: %v", got)
	}
	testHookScanCall = nil
	actual := false
	for _, c := range calls() {
		if !slices.Contains(c.Args, "log") {
			continue
		}
		if slices.Contains(c.Args, "--stdin") {
			actual = true
		}
		for _, key := range []string{"GIT_GRAFT_FILE=", "GIT_SHALLOW_FILE="} {
			// Probe control is intentionally unset; real scan and override tests are not.
			if slices.Contains(c.Args, "--stdin") && !slices.Contains(c.Env, key) {
				t.Fatalf("not transmitted: %s", key)
			}
		}
	}
	if !actual {
		t.Fatal("did not run the real scan")
	}
	wantEmpty(t, d)
}

func TestScanRefusesUnsupportedEmptyOverrides(t *testing.T) {
	requireSafeGit(t)
	repo, ids := chain(t, 2)
	s, d := scratchIn(t)
	dir, env, calls := testutil.PassthroughGit(t, testutil.PassIgnoringEmptyHistory)
	useGit(t, dir, env)
	for range 2 {
		if err := Log(ctx, s, repo.Dir, ids[1], nil, func(Commit) error { t.Error("published unsupported read"); return nil }); apperr.ExitCode(err) != apperr.ExitEnvironment {
			t.Fatalf("unsupported: %v", err)
		}
		stateMu.Lock()
		cached := state.scanEmpty
		stateMu.Unlock()
		if cached {
			t.Fatal("failed probe was cached")
		}
	}
	for _, c := range calls() {
		if slices.Contains(c.Args, "--stdin") {
			t.Fatal("ran user read after failed probe")
		}
	}
	wantEmpty(t, d)
}

func TestScratchManifestAnomalyIsRetained(t *testing.T) {
	requireSafeGit(t)
	for _, kind := range []string{"extra", "replaced", "missing"} {
		t.Run(kind, func(t *testing.T) {
			s, _ := scratchIn(t)
			p, err := prepare(ctx, s, baseRepo(t).Dir)
			if err != nil {
				t.Fatal(err)
			}
			witness := filepath.Join(p.gitDir, "config")
			before, err := os.ReadFile(witness)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "extra":
				write(t, filepath.Join(p.gitDir, "unexpected"), "keep")
			case "replaced":
				if err := os.Rename(witness, witness+".old"); err != nil {
					t.Fatal(err)
				}
				write(t, witness, "replacement")
			case "missing":
				if err := os.Remove(filepath.Join(p.gitDir, "HEAD")); err != nil {
					t.Fatal(err)
				}
			}
			_, _, err = p.read(ctx, true)
			if apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Fatalf("read: %v", err)
			}
			if err := p.close(); apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Fatalf("cleanup: %v", err)
			}
			if kind != "replaced" {
				wantFiles(t, map[string]string{witness: string(before)})
			} else {
				wantFiles(t, map[string]string{witness: "replacement", witness + ".old": string(before)})
			}
		})
	}
}

func TestScratchVersionNamespacesDoNotMix(t *testing.T) {
	s, d := scratchIn(t)
	old := d.Join(filepath.Join(datadir.TmpDir, "read-"+strings.Repeat("f", 32)))
	mkPrivate(t, old)
	write(t, filepath.Join(old, ownerName), "specmetry private directory 1\n")
	oldV2 := d.Join(filepath.Join(datadir.TmpDir, "git-reader-v2", "read-"+strings.Repeat("e", 32)))
	mkPrivate(t, filepath.Dir(oldV2))
	mkPrivate(t, oldV2)
	write(t, filepath.Join(oldV2, ownerName), "specmetry private directory 2\n")
	write(t, filepath.Join(oldV2, "index"), "possibly consumed by an older Git")
	p, err := newPrivateDir(s)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.write("index", []byte("new")); err != nil {
		t.Fatal(err)
	}
	if err = p.remove(); err != nil {
		t.Fatal(err)
	}
	_, err = newScratch(t, d)
	if err != nil {
		t.Fatal(err)
	}
	wantFiles(t, map[string]string{
		filepath.Join(old, ownerName):   "specmetry private directory 1\n",
		filepath.Join(oldV2, ownerName): "specmetry private directory 2\n",
		filepath.Join(oldV2, "index"):   "possibly consumed by an older Git",
	})
	// An older owner INSIDE v3 cannot be adopted, even if its basename looks ours.
	bad := filepath.Join(s.path, filepath.Base(old))
	mkPrivate(t, bad)
	write(t, filepath.Join(bad, ownerName), "specmetry private directory 1\n")
	if _, err = newScratch(t, d); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("mixed owner: %v", err)
	}
}

func helperScratchLease() int {
	s, err := NewScratch(datadir.Dir{Path: os.Getenv("SCRATCH_HOME")})
	if err != nil {
		return apperr.ExitCode(err)
	}
	defer func() { _ = s.Close() }()
	p, err := newPrivateDir(s)
	if err != nil {
		return apperr.ExitCode(err)
	}
	if err = p.write("index", []byte("synthetic-private-content")); err != nil {
		return 97
	}
	fmt.Println(p.path)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return 98
	}
	if strings.TrimSpace(line) == "crash" {
		os.Exit(99) //nolint:gocritic // Intentional crash: no deferred handle or lock release.
	}
	if err = p.remove(); err != nil {
		return apperr.ExitCode(err)
	}
	return 0
}

func TestScratchProtocolProcesses(t *testing.T) {
	s, d := scratchIn(t)
	for _, end := range []string{"release", "crash"} {
		t.Run(end, func(t *testing.T) {
			cmd := testutil.HelperCommand(t, "scratch-lease", "SCRATCH_HOME="+d.Path)
			input, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			output, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill() })
			ready := make(chan string, 1)
			go func() { line, _ := bufio.NewReader(output).ReadString('\n'); ready <- strings.TrimSpace(line) }()
			var path string
			select {
			case path = <-ready:
			case <-time.After(20 * time.Second):
				t.Fatal("lease helper did not start")
			}
			if path == "" {
				_ = cmd.Wait()
				t.Fatalf("no helper workload: %s", stderr.String())
			}
			if _, err = newPrivateDir(s); apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Fatalf("lease contention: %v", err)
			}
			wantFiles(t, map[string]string{filepath.Join(path, "index"): "synthetic-private-content"})
			if _, err = fmt.Fprintln(input, end); err != nil {
				t.Fatal(err)
			}
			_ = input.Close()
			err = cmd.Wait()
			if end == "crash" {
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 99 {
					t.Fatalf("crash: %v", err)
				}
			} else if err != nil {
				t.Fatalf("release: %v %s", err, stderr.String())
			}
			next, err := newScratch(t, d)
			if err != nil {
				t.Fatal(err)
			}
			wantEmpty(t, d)
			p, err := newPrivateDir(next)
			if err != nil {
				t.Fatal(err)
			}
			if err = p.remove(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Preserve Review 060's empty mkdir/open counterexample with its approved
// boundary, rather than claiming Unix now prevents excluded interference.
func TestCreationWindowWithApprovedBoundary(t *testing.T) {
	requireSafeGit(t)
	s, _ := scratchIn(t)
	var original string
	blocked := false
	testHookMkdir = func(path string) {
		testHookMkdir = nil
		original = path + ".original"
		err := os.Rename(path, original)
		if runtime.GOOS == "windows" {
			if err == nil {
				t.Fatal("Windows atomic creation did not guard its directory")
			}
			blocked = true
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		mkPrivate(t, path) // excluded uncooperative same-user intervention
	}
	t.Cleanup(func() { testHookMkdir = nil })
	got, err := Clean(ctx, s, baseRepo(t).Dir)
	if err != nil || got != StateClean {
		t.Fatalf("normal or approved boundary read: %v %v", got, err)
	}
	if runtime.GOOS == "windows" {
		if !blocked {
			t.Fatal("window was not exercised")
		}
		return
	}
	if fi, err := os.Stat(original); err != nil || !fi.IsDir() {
		t.Fatalf("original object lost: %v", err)
	}
	t.Log("Unix accepted the empty replacement in the explicitly excluded window; this is not claimed prevented")
}

func TestScanCapabilityFollowsExecutableIdentity(t *testing.T) {
	requireSafeGit(t)
	repo, ids := chain(t, 2)
	s, _ := scratchIn(t)
	dir, env, calls := testutil.PassthroughGit(t, testutil.PassAsIs)
	useGit(t, dir, env)
	read := func() {
		t.Helper()
		if err := Log(ctx, s, repo.Dir, ids[1], nil, func(Commit) error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	read()
	before, err := findGit()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(before.path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(before.path)
	if err != nil {
		t.Fatal(err)
	}
	fresh := before.path + ".new"
	if err := os.WriteFile(fresh, data, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(fresh, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(before.path, before.path+".old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(fresh, before.path); err != nil {
		t.Fatal(err)
	}
	after, err := findGit()
	if err != nil {
		t.Fatal(err)
	}
	if after.identity == before.identity || after.size != before.size || after.modTime != before.modTime {
		t.Fatalf("replacement fixture: before %+v after %+v", before, after)
	}
	read()
	controls := 0
	for _, call := range calls() {
		if slices.Contains(call.Args, "--format=%H") && !slices.Contains(call.Env, "GIT_GRAFT_FILE=") {
			controls++
		}
	}
	if controls != 2 {
		t.Fatalf("empty-override controls=%d; want a new probe for the replacement binary", controls)
	}
}

func TestScanProbeCancellationReleasesLease(t *testing.T) {
	requireSafeGit(t)
	if err := CheckObjectReads(ctx); err != nil {
		t.Fatal(err)
	}
	// Force only the new capability to be probed, after the existing PD-26 gate.
	stateMu.Lock()
	state.scanEmpty = false
	stateMu.Unlock()
	repo, ids := chain(t, 2)
	s, d := scratchIn(t)
	cancelled, cancel := context.WithCancel(ctx)
	defer cancel()
	testHookPrivateDir = func(string) { testHookPrivateDir = nil; cancel() }
	t.Cleanup(func() { testHookPrivateDir = nil })
	if err := Log(cancelled, s, repo.Dir, ids[1], nil, func(Commit) error { t.Error("published cancelled scan"); return nil }); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("cancelled: %v", err)
	}
	wantEmpty(t, d)
	stateMu.Lock()
	cached := state.scanEmpty
	stateMu.Unlock()
	if cached {
		t.Fatal("cancelled probe cached")
	}
	p, err := newPrivateDir(s)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.remove(); err != nil {
		t.Fatal(err)
	}
}

// A real three-parent merge has non-ancestral parents. The probe alone does
// not substitute for this public Log regression or for scanner atomicity.
func TestScanEmptyMultiParentAndFailures(t *testing.T) {
	requireSafeGit(t)
	for _, damage := range []string{"none", "missing", "corrupt"} {
		t.Run(damage, func(t *testing.T) {
			repo, ids := chain(t, 2)
			var sides []string
			for _, branch := range []string{"side1", "side2"} {
				repo.Git("checkout", "-q", "-b", branch, ids[0])
				repo.WriteFile(branch+".txt", branch)
				sides = append(sides, repo.Commit(branch))
			}
			repo.Git("checkout", "-q", "main")
			repo.Git("merge", "--no-ff", "-m", "three-parent merge", "side1", "side2")
			tip := repo.Git("rev-parse", "HEAD")
			parents := strings.Fields(repo.Git("show", "-s", "--format=%P", tip))
			if len(parents) != 3 {
				t.Fatalf("not a real three-parent fixture: %v", parents)
			}
			s, _ := scratchIn(t)
			read := func(excludes ...string) ([]string, error) {
				var out []string
				err := Log(ctx, s, repo.Dir, tip, excludes, func(c Commit) error { out = append(out, c.SHA); return nil })
				slices.Sort(out)
				return out, err
			}
			want := append(append(slices.Clone(ids), sides...), tip)
			slices.Sort(want)
			got, err := read()
			if err != nil || !slices.Equal(got, want) {
				t.Fatalf("full merge: %v %v", got, err)
			}
			want = append(slices.Clone(sides), tip)
			slices.Sort(want)
			got, err = read(ids[1])
			if err != nil || !slices.Equal(got, want) {
				t.Fatalf("incremental merge: %v %v", got, err)
			}
			if damage == "none" {
				return
			}
			path := loose(repo, sides[0])
			writable(t, path)
			if damage == "missing" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			} else {
				write(t, path, "corrupt object")
			}
			if _, err := read(); apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Fatalf("%s parent: %v", damage, err)
			}
		})
	}
}

func TestScanScratchContract(t *testing.T) {
	requireSafeGit(t)
	repo, ids := chain(t, 2)
	closed, _ := scratchIn(t)
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	for _, s := range []*Scratch{nil, {}, closed} {
		if err := Log(ctx, s, repo.Dir, ids[1], nil, func(Commit) error { t.Error("read with unavailable Scratch"); return nil }); apperr.ExitCode(err) != apperr.ExitInternal {
			t.Fatalf("unavailable Scratch: %v", err)
		}
	}
	s, _ := scratchIn(t)
	if !moveAway(t, s.path, s.path+".moved") {
		if err := Log(ctx, s, repo.Dir, ids[1], nil, func(Commit) error { return nil }); err != nil {
			t.Fatal(err)
		}
		return
	}
	mkPrivate(t, s.path)
	if err := Log(ctx, s, repo.Dir, ids[1], nil, func(Commit) error { t.Error("read after observed Scratch replacement"); return nil }); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("replaced Scratch: %v", err)
	}
}
