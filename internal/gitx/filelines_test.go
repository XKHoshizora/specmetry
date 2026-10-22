package gitx

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// deepRepo has a commit A with a file three levels down, the same content
// at two paths (one blob), and a commit B that adds a file beside it; HEAD
// is back at A.
func deepRepo(t *testing.T) (repo *testutil.GitRepo, a, b string) {
	t.Helper()
	repo = testutil.NewGitRepo(t)
	repo.WriteFile("x/y/z/deep.go", "one\ntwo\nthree\n")
	repo.WriteFile("same/one.txt", "same\n")
	repo.WriteFile("same/two.txt", "same\n")
	a = repo.Commit("a")
	repo.WriteFile("x/y/z/ghost.go", "ghost\n")
	b = repo.Commit("b")
	repo.Git("reset", "-q", "--hard", a)
	return repo, a, b
}

var deepPaths = []string{"x/y/z/deep.go", "x/y/z/ghost.go", "same/one.txt", "same/two.txt", "x/y", "x/y/z/deep.go/w"}

// Review 068 B3: FileLines reads, and checks against its ID, every object
// it decides on: the commit, the trees on the way to each path, a level at
// a time, and each blob, once however many paths name it. A blob cut short,
// a blob or a subtree three levels down holding another object's bytes
// (which would make B's x/y/z/ghost.go a file of A), a commit or tree given
// for the wrong type: each is an environment error with no answer, the
// repository unchanged. A cancelled context gives no answer either.
// Healthy, a path A does not have, a directory and a path below a file are
// no files, and the two paths of one blob both have its count.
func TestFileLinesVerifiesTheObjects(t *testing.T) {
	requireSafeGit(t)
	repo, a, _ := deepRepo(t)
	want := map[string]int{"x/y/z/deep.go": 3, "same/one.txt": 1, "same/two.txt": 1}
	if got, err := FileLines(ctx, repo.Dir, a, deepPaths); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("healthy: FileLines = %v, %v; want %v", got, err, want)
	}
	if repo.Git("rev-parse", a+":same/one.txt") != repo.Git("rev-parse", a+":same/two.txt") {
		t.Fatal("the two paths do not share a blob")
	}
	tree := repo.Git("rev-parse", a+"^{tree}")
	if got, err := FileLines(ctx, repo.Dir, tree, deepPaths); apperr.ExitCode(err) != apperr.ExitEnvironment || got != nil {
		t.Errorf("a tree for the commit: %v, %v; want an environment error", got, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if got, err := FileLines(cancelled, repo.Dir, a, deepPaths); err == nil || got != nil {
		t.Errorf("cancelled: %v, %v; want an error and no answer", got, err)
	}

	for _, c := range []struct {
		name   string
		damage func(t *testing.T, repo *testutil.GitRepo, a, b string)
	}{
		{"the deep blob cut short", func(t *testing.T, repo *testutil.GitRepo, a, _ string) {
			damageLoose(t, repo, repo.Git("rev-parse", a+":x/y/z/deep.go"), []byte("blob 500\x00one\n"))
		}},
		{"the shared blob cut short", func(t *testing.T, repo *testutil.GitRepo, a, _ string) {
			damageLoose(t, repo, repo.Git("rev-parse", a+":same/one.txt"), []byte("blob 50\x00same\n"))
		}},
		{"the deepest subtree holding B's", func(t *testing.T, repo *testutil.GitRepo, a, b string) {
			substituteLoose(t, repo, repo.Git("rev-parse", a+":x/y/z"), repo.Git("rev-parse", b+":x/y/z"))
		}},
		{"a middle subtree holding B's", func(t *testing.T, repo *testutil.GitRepo, a, b string) {
			substituteLoose(t, repo, repo.Git("rev-parse", a+":x"), repo.Git("rev-parse", b+":x"))
		}},
		{"the commit holding B's", func(t *testing.T, repo *testutil.GitRepo, a, b string) { substituteLoose(t, repo, a, b) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, a, b := deepRepo(t)
			c.damage(t, repo, a, b)
			gitDir := filepath.Join(repo.Dir, ".git")
			before := testutil.SettledSnapshot(t, gitDir)
			got, err := FileLines(ctx, repo.Dir, a, deepPaths)
			if apperr.ExitCode(err) != apperr.ExitEnvironment || got != nil || strings.Contains(err.Error(), "fatal") {
				t.Errorf("FileLines = %v, %v; want an environment error and no answer", got, err)
			}
			testutil.AssertTreeUnchanged(t, gitDir, before)
		})
	}
}

// writeLooseObject writes an object of type typ with content straight into
// the loose object storage, as a broken tool might, and returns its ID: git
// itself would refuse a tree that names an object of another type.
func writeLooseObject(t *testing.T, repo *testutil.GitRepo, typ string, content []byte) string {
	t.Helper()
	full := append([]byte(fmt.Sprintf("%s %d\x00", typ, len(content))), content...)
	sum := sha1.Sum(full)
	id := hex.EncodeToString(sum[:])
	var b bytes.Buffer
	z := zlib.NewWriter(&b)
	_, _ = z.Write(full)
	_ = z.Close()
	p := loose(repo, id)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return id
}

func rawID(t *testing.T, id string) string {
	t.Helper()
	b, err := hex.DecodeString(id)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Review 068 B3: every object FileLines reads must also be the type its
// tree entry gives it, and every hash here is right. A tree entry for a
// directory that names a blob whose bytes happen to parse as a tree
// (listing B's ghost.go), and a file entry that names a tree, are
// environment errors, not a file and its count.
func TestFileLinesChecksTheTypes(t *testing.T) {
	requireSafeGit(t)
	repo, _, b := deepRepo(t)
	ghost := repo.Git("rev-parse", b+":x/y/z/ghost.go")
	disguised := writeLooseObject(t, repo, "blob", []byte("100644 ghost.go\x00"+rawID(t, ghost)))
	subtree := repo.Git("rev-parse", b+":x/y/z")
	root := writeLooseObject(t, repo, "tree", []byte("40000 fake\x00"+rawID(t, disguised)+"100644 tree.go\x00"+rawID(t, subtree)))
	commit := writeLooseObject(t, repo, "commit", []byte("tree "+root+"\nauthor A <a@example.com> 1700000000 +0000\ncommitter A <a@example.com> 1700000000 +0000\n\ntypes\n"))
	if got, err := FileLines(ctx, repo.Dir, commit, nil); err != nil || len(got) != 0 {
		t.Fatalf("the crafted commit reads: %v, %v", got, err)
	}
	for _, p := range []string{"fake/ghost.go", "tree.go"} {
		if got, err := FileLines(ctx, repo.Dir, commit, []string{p}); apperr.ExitCode(err) != apperr.ExitEnvironment || got != nil {
			t.Errorf("%s: FileLines = %v, %v; want an environment error", p, got, err)
		}
	}
}

// substituteLoose puts the loose bytes of another, valid object in the file
// of oid.
func substituteLoose(t *testing.T, repo *testutil.GitRepo, oid, with string) {
	t.Helper()
	b, err := os.ReadFile(loose(repo, with))
	if err != nil {
		t.Fatal(err)
	}
	writable(t, loose(repo, oid))
	if err := os.WriteFile(loose(repo, oid), b, 0o644); err != nil {
		t.Fatal(err)
	}
}
