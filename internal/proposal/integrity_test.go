package proposal_test

import (
	"bytes"
	"compress/zlib"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/proposal"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// integrityRepo has two commits: A, whose d/a.go holds a canary on its
// first of two lines, with an empty file and a file without a final line
// break; and B, which adds d/ghost.go and changes nothing else. HEAD is
// back at A, and both commits' objects are loose.
type integrityRepo struct {
	repo           *testutil.GitRepo
	a, b           string
	canary         string
	blob           string // d/a.go at A
	treeA, treeB   string // d at A and at B
	rootA, rootB   string
	ghostBlob      string
	otherBlob      string // a valid blob that is no file of A
	looseOf        func(oid string) string
	originalLooses map[string][]byte
}

func newIntegrityRepo(t *testing.T) *integrityRepo {
	t.Helper()
	if os.Getenv("SPECMETRY_TEST_GIT_OBJECTS") == "unsafe" {
		t.Skip("the git on PATH may not read objects")
	}
	r := &integrityRepo{repo: testutil.NewGitRepo(t), canary: testutil.NewCanary(t), originalLooses: map[string][]byte{}}
	r.repo.WriteFile("d/a.go", "one "+r.canary+"\ntwo\n")
	r.repo.WriteFile("empty.txt", "")
	r.repo.WriteFile("nolf.txt", "first\nsecond")
	r.a = r.repo.Commit("a")
	r.repo.WriteFile("d/ghost.go", "ghost\n")
	r.b = r.repo.Commit("b")
	r.repo.Git("reset", "-q", "--hard", r.a)
	r.blob = r.repo.Git("rev-parse", r.a+":d/a.go")
	r.treeA, r.treeB = r.repo.Git("rev-parse", r.a+":d"), r.repo.Git("rev-parse", r.b+":d")
	r.rootA, r.rootB = r.repo.Git("rev-parse", r.a+"^{tree}"), r.repo.Git("rev-parse", r.b+"^{tree}")
	r.ghostBlob = r.repo.Git("rev-parse", r.b+":d/ghost.go")
	other := filepath.Join(t.TempDir(), "other")
	if err := os.WriteFile(other, []byte("other\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.otherBlob = r.repo.Git("hash-object", "-w", other)
	r.looseOf = func(oid string) string { return filepath.Join(r.repo.Dir, ".git", "objects", oid[:2], oid[2:]) }
	return r
}

// replace puts content, compressed, in the loose file of oid; the original
// bytes are kept for restore.
func (r *integrityRepo) replace(t *testing.T, oid string, content []byte) {
	t.Helper()
	var b bytes.Buffer
	z := zlib.NewWriter(&b)
	_, _ = z.Write(content)
	_ = z.Close()
	r.write(t, oid, b.Bytes())
}

// substitute puts the loose bytes of another, valid object in the file of
// oid: git then reads oid as that object.
func (r *integrityRepo) substitute(t *testing.T, oid, with string) {
	t.Helper()
	b, err := os.ReadFile(r.looseOf(with))
	if err != nil {
		t.Fatal(err)
	}
	r.write(t, oid, b)
}

func (r *integrityRepo) remove(t *testing.T, oid string) {
	t.Helper()
	r.keep(t, oid)
	if err := os.Remove(r.looseOf(oid)); err != nil {
		t.Fatal(err)
	}
}

func (r *integrityRepo) keep(t *testing.T, oid string) {
	t.Helper()
	if _, kept := r.originalLooses[oid]; kept {
		return
	}
	b, err := os.ReadFile(r.looseOf(oid))
	if err != nil {
		t.Fatal(err)
	}
	r.originalLooses[oid] = b
	if err := os.Chmod(r.looseOf(oid), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (r *integrityRepo) write(t *testing.T, oid string, b []byte) {
	t.Helper()
	r.keep(t, oid)
	if err := os.WriteFile(r.looseOf(oid), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func (r *integrityRepo) restore(t *testing.T) {
	t.Helper()
	for oid, b := range r.originalLooses {
		if err := os.WriteFile(r.looseOf(oid), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r.originalLooses = map[string][]byte{}
}

var integrityRefs = [][]proposal.CodeRef{
	{{Path: "d/a.go", Lines: "1"}, {Path: "d/a.go", Lines: "999"}, {Path: "d/ghost.go"}, {Path: "d"}, {Path: "none.go"}},
	{{Path: "empty.txt"}, {Path: "empty.txt", Lines: "1"}, {Path: "nolf.txt", Lines: "2"}, {Path: "nolf.txt", Lines: "3"}},
}

const integrityHealthy = "[d/a.go@1:exists d/a.go@999:missing d/ghost.go@-:missing d@-:missing none.go@-:missing ]" +
	"[empty.txt@-:exists empty.txt@1:missing nolf.txt@2:exists nolf.txt@3:missing ]"

// marks is a check's results as path@lines:exists|missing, without the
// commit, which TestCheckCodeRefsAt checks.
func marks(refs [][]proposal.CheckedCodeRef) string {
	var b strings.Builder
	for _, list := range refs {
		b.WriteString("[")
		for _, r := range list {
			lines := "-"
			if r.Lines != nil {
				lines = *r.Lines
			}
			b.WriteString(r.Path + "@" + lines + ":" + map[bool]string{true: "exists", false: "missing"}[r.Exists] + " ")
		}
		b.WriteString("]")
	}
	return b.String()
}

// Review 068 B3 (WP2.4 spec §7, V-C06; propose shares the path): every
// object the file check reads to decide (the commit, each tree on the way
// to a path, the file's blob) must be read whole as the object its ID
// names. A blob cut short (git exits 0 and gives what is left), a blob
// holding another blob's bytes, a subtree holding another tree's (git then
// lists d/ghost.go, which A does not have), a root tree or commit holding
// another's, a removed blob or subtree: each is an environment error with
// no result, at the captured head and at HEAD (propose's), alone or among
// healthy references, and quotes nothing of the content. Healthy objects
// keep the existing marks: a file of A, a line range past its end, a path
// A does not have, a directory, an empty file and a last line without a
// line break. Repaired, the same check gives the healthy marks again.
func TestCheckCodeRefsVerifiesTheObjects(t *testing.T) {
	check := func(t *testing.T, r *integrityRepo, what string, healthy bool) {
		t.Helper()
		for name, run := range map[string]func() ([][]proposal.CheckedCodeRef, error){
			"CheckCodeRefsAt": func() ([][]proposal.CheckedCodeRef, error) {
				return proposal.CheckCodeRefsAt(ctx, where(t, r.repo.Dir), r.a, integrityRefs)
			},
			"CheckCodeRefs": func() ([][]proposal.CheckedCodeRef, error) {
				return proposal.CheckCodeRefs(ctx, where(t, r.repo.Dir), integrityRefs)
			},
		} {
			got, err := run()
			switch {
			case healthy && (err != nil || marks(got) != integrityHealthy):
				t.Errorf("%s, %s: %s, %v; want %s", what, name, marks(got), err, integrityHealthy)
			case !healthy && (apperr.ExitCode(err) != apperr.ExitEnvironment || got != nil):
				t.Errorf("%s, %s: %s, %v; want an environment error and no result", what, name, marks(got), err)
			case !healthy && (strings.Contains(err.Error(), r.canary) || strings.Contains(err.Error(), "fatal")):
				t.Errorf("%s, %s: the error quotes the object or git: %v", what, name, err)
			}
		}
	}
	t.Run("healthy", func(t *testing.T) {
		check(t, newIntegrityRepo(t), "healthy objects", true)
	})
	for _, c := range []struct {
		name   string
		damage func(t *testing.T, r *integrityRepo)
	}{
		{"a blob cut short", func(t *testing.T, r *integrityRepo) { r.replace(t, r.blob, []byte("blob 500\x00one\n")) }},
		{"a blob holding another blob's bytes", func(t *testing.T, r *integrityRepo) { r.substitute(t, r.blob, r.otherBlob) }},
		{"a subtree holding another tree's bytes", func(t *testing.T, r *integrityRepo) { r.substitute(t, r.treeA, r.treeB) }},
		{"a root tree holding another tree's bytes", func(t *testing.T, r *integrityRepo) { r.substitute(t, r.rootA, r.rootB) }},
		{"the commit holding another commit's bytes", func(t *testing.T, r *integrityRepo) { r.substitute(t, r.a, r.b) }},
		{"the commit cut short", func(t *testing.T, r *integrityRepo) { r.replace(t, r.a, []byte("commit 900\x00tree "+r.rootA+"\n")) }},
		{"a subtree cut short", func(t *testing.T, r *integrityRepo) { r.replace(t, r.treeA, []byte("tree 300\x00100644 a.go\x00")) }},
		{"the blob removed", func(t *testing.T, r *integrityRepo) { r.remove(t, r.blob) }},
		{"the subtree removed", func(t *testing.T, r *integrityRepo) { r.remove(t, r.treeA) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newIntegrityRepo(t)
			gitDir := filepath.Join(r.repo.Dir, ".git")
			c.damage(t, r)
			before := testutil.SettledSnapshot(t, gitDir)
			check(t, r, c.name, false)
			testutil.AssertTreeUnchanged(t, gitDir, before)
			r.restore(t)
			check(t, r, c.name+", repaired", true)
		})
	}
	t.Run("HEAD moved to a commit whose objects are damaged", func(t *testing.T) {
		r := newIntegrityRepo(t)
		r.repo.Git("reset", "-q", "--hard", r.b)
		r.replace(t, r.ghostBlob, []byte("blob 400\x00ghost\n"))
		got, err := proposal.CheckCodeRefsAt(ctx, where(t, r.repo.Dir), r.a, integrityRefs)
		if err != nil || marks(got) != integrityHealthy {
			t.Errorf("at the captured A: %s, %v; want %s", marks(got), err, integrityHealthy)
		}
		if got, err := proposal.CheckCodeRefs(ctx, where(t, r.repo.Dir), integrityRefs); apperr.ExitCode(err) != apperr.ExitEnvironment || got != nil {
			t.Errorf("at HEAD, B: %s, %v; want an environment error", marks(got), err)
		}
	})
}
