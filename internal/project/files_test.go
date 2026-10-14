package project_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

var ctx = context.Background()

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func paths(fs []project.File) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Path)
	}
	slices.Sort(out)
	return out
}

// A git project lists tracked files and untracked files that are not
// ignored, with their sizes, from the given worktree.
func TestListFilesGit(t *testing.T) {
	repo := testutil.NewGitRepo(t)
	repo.WriteFile(".gitignore", "ignored.md\n")
	repo.WriteFile("docs/brief.md", "0123456789")
	repo.WriteFile("gone.md", "x")
	repo.Commit("init")
	if err := os.Remove(filepath.Join(repo.Dir, "gone.md")); err != nil {
		t.Fatal(err)
	}
	write(t, repo.Dir, "untracked.md", "abc")
	write(t, repo.Dir, "ignored.md", "abc")
	got, err := project.ListFiles(ctx, repo.Dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{".gitignore", "docs/brief.md", "untracked.md"}; !reflect.DeepEqual(paths(got), want) {
		t.Fatalf("files = %q, want %q", paths(got), want)
	}
	for _, f := range got {
		if f.Path == "docs/brief.md" && f.Size != 10 {
			t.Fatalf("size of docs/brief.md = %d", f.Size)
		}
	}
}

// A non-git project walks the directory, skipping the excluded
// directories without descending into them.
func TestListFilesWalk(t *testing.T) {
	root := t.TempDir()
	write(t, root, "README.md", "readme")
	write(t, root, "docs/brief.md", "0123456789")
	write(t, root, "node_modules/x/README.md", "x")
	write(t, root, ".cache/a.md", "x")
	write(t, root, "src/main.go", "package main")
	got, err := project.ListFiles(ctx, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"README.md", "docs/brief.md", "src/main.go"}; !reflect.DeepEqual(paths(got), want) {
		t.Fatalf("files = %q, want %q", paths(got), want)
	}
}

// Symbolic links are not followed, in either kind of project.
func TestListFilesSkipsSymlinks(t *testing.T) {
	outside := t.TempDir()
	write(t, outside, "secret-brief.md", "outside the project")
	for _, git := range []bool{false, true} {
		var root string
		if git {
			repo := testutil.NewGitRepo(t)
			repo.WriteFile("README.md", "readme")
			repo.Commit("init")
			root = repo.Dir
		} else {
			root = t.TempDir()
			write(t, root, "README.md", "readme")
		}
		if err := os.Symlink(filepath.Join(outside, "secret-brief.md"), filepath.Join(root, "linked-brief.md")); err != nil {
			if runtime.GOOS == "windows" {
				t.Skipf("cannot create symbolic links: %v", err)
			}
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(root, "linked-dir")); err != nil {
			t.Fatal(err)
		}
		got, err := project.ListFiles(ctx, root, git)
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"README.md"}; !reflect.DeepEqual(paths(got), want) {
			t.Fatalf("git=%v: files = %q, want %q", git, paths(got), want)
		}
	}
}

func TestMarkerRoundTrip(t *testing.T) {
	root := t.TempDir()
	if _, found, err := project.ReadMarker(root); err != nil || found {
		t.Fatalf("ReadMarker on an empty directory = %v, %v", found, err)
	}
	m := project.Marker{ProjectID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4C1", Name: "demo"}
	if err := project.WriteMarker(root, m); err != nil {
		t.Fatal(err)
	}
	got, found, err := project.ReadMarker(root)
	if err != nil || !found || got.ProjectID != m.ProjectID || got.Name != m.Name || got.Schema != project.MarkerSchema {
		t.Fatalf("ReadMarker = %+v, %v, %v", got, found, err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".specmetry", "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"schema":"specmetry.project/v1","project_id":"01J9Z3K5Q8R2T4V6W8X0Y2Z4C1","name":"demo"}` + "\n"; string(raw) != want {
		t.Fatalf("marker = %q, want %q", raw, want)
	}
	entries, _ := os.ReadDir(filepath.Join(root, ".specmetry"))
	if len(entries) != 1 {
		t.Fatalf(".specmetry holds %d entries, want only the marker", len(entries))
	}
}

// A marker that cannot be trusted is an error, never "no marker".
func TestReadMarkerRejectsBadMarkers(t *testing.T) {
	cases := map[string]string{
		"not JSON":       "{",
		"wrong schema":   `{"schema":"other/v1","project_id":"01J9Z3K5Q8R2T4V6W8X0Y2Z4C1","name":"demo"}`,
		"bad project id": `{"schema":"specmetry.project/v1","project_id":"not-a-ulid","name":"demo"}`,
		"unknown field":  `{"schema":"specmetry.project/v1","project_id":"01J9Z3K5Q8R2T4V6W8X0Y2Z4C1","name":"demo","x":1}`,
	}
	for name, content := range cases {
		root := t.TempDir()
		write(t, root, ".specmetry/project.json", content)
		if _, _, err := project.ReadMarker(root); err == nil {
			t.Errorf("%s: ReadMarker accepted %s", name, content)
		}
	}
	root := t.TempDir()
	write(t, root, ".specmetry", "a file, not a directory")
	if _, _, err := project.ReadMarker(root); err == nil {
		t.Error("ReadMarker accepted .specmetry as a file")
	}
	if err := project.WriteMarker(root, project.Marker{ProjectID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4C1", Name: "demo"}); err == nil {
		t.Error("WriteMarker wrote through .specmetry as a file")
	}
}

// The marker directory must not lead outside the project.
func TestMarkerRefusesSymlinkedDirectory(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".specmetry")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("cannot create symbolic links: %v", err)
		}
		t.Fatal(err)
	}
	if err := project.WriteMarker(root, project.Marker{ProjectID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4C1", Name: "demo"}); err == nil {
		t.Fatal("WriteMarker wrote through a symbolic link")
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("written outside the project: %v", entries)
	}
}

// FindMarker looks in the directory and its parents.
func TestFindMarker(t *testing.T) {
	root := t.TempDir()
	if err := project.WriteMarker(root, project.Marker{ProjectID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4C1", Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{root, sub} {
		got, found := project.FindMarker(dir)
		if !found || got != root {
			t.Errorf("FindMarker(%s) = %s, %v", dir, got, found)
		}
	}
	if _, found := project.FindMarker(t.TempDir()); found {
		t.Error("FindMarker found a marker in an empty directory")
	}
}

// MVP-INIT-08: the marker directory is added to the exclude file once,
// keeping what was there.
func TestEnsureExcluded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "info", "exclude")
	if err := project.EnsureExcluded(path); err != nil {
		t.Fatal(err)
	}
	if err := project.EnsureExcluded(path); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != "/.specmetry/\n" {
		t.Fatalf("new exclude file = %q", b)
	}

	if err := os.WriteFile(path, []byte("# mine\n*.log"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := project.EnsureExcluded(path); err != nil {
		t.Fatal(err)
	}
	if err := project.EnsureExcluded(path); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != "# mine\n*.log\n/.specmetry/\n" {
		t.Fatalf("exclude file = %q", b)
	}

	for _, existing := range []string{"/.specmetry/", "/.specmetry", ".specmetry/", ".specmetry", "  /.specmetry/  "} {
		content := "*.log\r\n" + existing + "\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := project.EnsureExcluded(path); err != nil {
			t.Fatal(err)
		}
		if b, _ := os.ReadFile(path); string(b) != content {
			t.Errorf("an existing entry %q was not recognised: %q", existing, b)
		}
	}
}

func TestEnsureExcludedReportsUnwritableFile(t *testing.T) {
	if runtime.GOOS != "windows" && os.Geteuid() == 0 {
		t.Skip("root can write read-only files")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "exclude")
	if err := os.WriteFile(path, []byte("*.log\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	err := project.EnsureExcluded(path)
	if err == nil {
		t.Fatal("EnsureExcluded wrote a read-only file")
	}
	if b, _ := os.ReadFile(path); string(b) != "*.log\n" {
		t.Fatalf("read-only exclude file changed: %q", b)
	}
	if strings.Contains(err.Error(), "*.log") {
		t.Fatalf("the error quotes the file: %v", err)
	}
}

// cancelAfter is a context whose Err reports cancellation from its n-th
// call on, as if it were cancelled while the walk runs.
type cancelAfter struct {
	context.Context
	n, calls int
}

func (c *cancelAfter) Err() error {
	c.calls++
	if c.calls >= c.n {
		return context.Canceled
	}
	return nil
}

// codex-review-015 N2: a cancelled scan stops, soon, and is reported as an
// error, never as a complete list.
func TestListFilesStopsWhenCanceled(t *testing.T) {
	root := t.TempDir()
	for i := range 50 {
		write(t, root, fmt.Sprintf("d%02d/f.md", i), "x")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	files, err := project.ListFiles(canceled, root, false)
	if !errors.Is(err, context.Canceled) || files != nil || apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("cancelled before the walk: %d files, %v", len(files), err)
	}

	during := &cancelAfter{Context: ctx, n: 10}
	files, err = project.ListFiles(during, root, false)
	if !errors.Is(err, context.Canceled) || files != nil {
		t.Fatalf("cancelled during the walk: %d files, %v", len(files), err)
	}
	if during.calls > during.n {
		t.Fatalf("the walk went on after the cancellation (%d checks)", during.calls)
	}

	repo := testutil.NewGitRepo(t)
	repo.WriteFile("README.md", "readme")
	repo.Commit("init")
	if files, err := project.ListFiles(canceled, repo.Dir, true); err == nil || files != nil {
		t.Fatalf("cancelled git listing: %d files, %v", len(files), err)
	}
}

// Concurrent init runs may all find the marker missing and write it at
// once (TestInitConcurrentRunsRegisterOneProject, seen failing on macOS
// CI): creating the directory must not fail because another writer just
// created it.
func TestWriteMarkerConcurrently(t *testing.T) {
	m := project.Marker{ProjectID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4C1", Name: "demo"}
	for round := range 20 {
		root := t.TempDir()
		var wg sync.WaitGroup
		errs := make([]error, 8)
		for i := range errs {
			wg.Go(func() { errs[i] = project.WriteMarker(root, m) })
		}
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Fatalf("round %d, writer %d: %v", round, i, err)
			}
		}
		if got, found, err := project.ReadMarker(root); err != nil || !found || got.ProjectID != m.ProjectID {
			t.Fatalf("round %d: ReadMarker = %+v, %v, %v", round, got, found, err)
		}
	}
}

// The marker is created, never replaced: an existing marker that holds the
// same project counts as written; a different one is left alone and is an
// error.
func TestWriteMarkerNeverReplaces(t *testing.T) {
	root := t.TempDir()
	m := project.Marker{ProjectID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4C1", Name: "demo"}
	if err := project.WriteMarker(root, m); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, ".specmetry", "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := project.WriteMarker(root, m); err != nil {
		t.Fatalf("writing the same marker again: %v", err)
	}
	other := project.Marker{ProjectID: "01J9Z3K5Q8R2T4V6W8X0Y2Z4C2", Name: "demo"}
	if err := project.WriteMarker(root, other); err == nil {
		t.Fatal("a different marker was replaced")
	}
	after, err := os.ReadFile(filepath.Join(root, ".specmetry", "project.json"))
	if err != nil || string(after) != string(before) {
		t.Fatalf("marker changed: %q -> %q (%v)", before, after, err)
	}
	if entries, _ := os.ReadDir(filepath.Join(root, ".specmetry")); len(entries) != 1 {
		t.Fatalf(".specmetry holds %d entries, want only the marker", len(entries))
	}
}
