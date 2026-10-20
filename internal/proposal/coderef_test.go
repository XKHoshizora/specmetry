package proposal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/proposal"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

func where(t *testing.T, dir string) project.Where {
	t.Helper()
	w, err := project.Locate(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// WP2.4 spec §7 (review 065): a claim's file references are checked
// against the HEAD commit the command captured, never one resolved anew.
// After HEAD moves on, a file of the captured commit is still there and a
// file only the new HEAD has is not; a line range must lie within the file
// of that commit; each result names that commit. With no commit (head "")
// nothing is there and no commit is named. Outside git the files under the
// project root count and no commit is named.
func TestCheckCodeRefsAt(t *testing.T) {
	if os.Getenv("SPECMETRY_TEST_GIT_OBJECTS") == "unsafe" {
		t.Skip("the git on PATH may not read objects")
	}
	repo := testutil.NewGitRepo(t)
	repo.WriteFile("a.go", "one\ntwo\nthree\n")
	captured := repo.Commit("a")
	repo.Git("rm", "-q", "a.go")
	repo.WriteFile("b.go", "b\n")
	repo.Commit("b")
	refs := [][]proposal.CodeRef{{{Path: "a.go", Lines: "1-3"}, {Path: "b.go"}, {Path: "a.go", Lines: "4"}}, {}, {{Path: `.\a.go`}}}
	got, err := proposal.CheckCodeRefsAt(ctx, where(t, repo.Dir), captured, refs)
	if err != nil {
		t.Fatal(err)
	}
	lines13, lines4 := "1-3", "4"
	want := [][]proposal.CheckedCodeRef{
		{{Path: "a.go", Lines: &lines13, Exists: true, CheckedHead: &captured}, {Path: "b.go", CheckedHead: &captured}, {Path: "a.go", Lines: &lines4, CheckedHead: &captured}},
		{},
		{{Path: `.\a.go`, Exists: true, CheckedHead: &captured}},
	}
	if !checkedEqual(got, want) {
		t.Errorf("at the captured head = %s; want %s", show(got), show(want))
	}

	got, err = proposal.CheckCodeRefsAt(ctx, where(t, repo.Dir), "", refs[:1])
	if want := [][]proposal.CheckedCodeRef{{{Path: "a.go", Lines: &lines13}, {Path: "b.go"}, {Path: "a.go", Lines: &lines4}}}; err != nil || !checkedEqual(got, want) {
		t.Errorf("before any commit = %s, %v; want %s", show(got), err, show(want))
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = proposal.CheckCodeRefsAt(ctx, where(t, dir), "", [][]proposal.CodeRef{{{Path: "a.go", Lines: "1"}, {Path: "b.go"}}})
	lines1 := "1"
	if want := [][]proposal.CheckedCodeRef{{{Path: "a.go", Lines: &lines1, Exists: true}, {Path: "b.go"}}}; err != nil || !checkedEqual(got, want) {
		t.Errorf("outside git = %s, %v; want %s", show(got), err, show(want))
	}

	for name, c := range map[string]struct {
		dir, head string
	}{
		"a head outside git":       {dir, captured},
		"a head that is not an ID": {repo.Dir, "HEAD"},
		"an abbreviated head":      {repo.Dir, captured[:12]},
	} {
		if got, err := proposal.CheckCodeRefsAt(ctx, where(t, c.dir), c.head, refs); apperr.ExitCode(err) != apperr.ExitInternal {
			t.Errorf("%s: %s, %v; want an internal error", name, show(got), err)
		}
	}
}

func checkedEqual(a, b [][]proposal.CheckedCodeRef) bool {
	return show(a) == show(b)
}

func show(refs [][]proposal.CheckedCodeRef) string {
	var b strings.Builder
	for _, list := range refs {
		b.WriteString("[")
		for _, r := range list {
			lines, head := "-", "-"
			if r.Lines != nil {
				lines = *r.Lines
			}
			if r.CheckedHead != nil {
				head = *r.CheckedHead
			}
			b.WriteString(r.Path + "@" + lines + ":" + map[bool]string{true: "exists", false: "missing"}[r.Exists] + ":" + head + " ")
		}
		b.WriteString("]")
	}
	return b.String()
}

// WP2.4 spec §5 step 4: a claim's code reference is checked as a
// proposal's is (V-P04, V-P08), by the same rules: its path, normalized
// lexically with either separator, stays in the repository; its lines are
// none, a line, or an ordered range from 1 to 2147483647. Each problem is
// reported at the field it is about, without its value.
func TestCodeRefProblems(t *testing.T) {
	for _, c := range []struct {
		ref  proposal.CodeRef
		want []string
	}{
		{proposal.CodeRef{Path: "internal/a.go", Lines: "1-3"}, nil},
		{proposal.CodeRef{Path: "a/./b/../c.go"}, nil},
		{proposal.CodeRef{Path: "a/../../c.go"}, []string{"/r/path"}},
		{proposal.CodeRef{Path: `a\..\..\c.go`}, []string{"/r/path"}},
		{proposal.CodeRef{Path: "a.go", Lines: "3-1"}, []string{"/r/lines"}},
		{proposal.CodeRef{Path: "a.go", Lines: "2147483648"}, []string{"/r/lines"}},
		{proposal.CodeRef{Path: "../a.go", Lines: "0"}, []string{"/r/path", "/r/lines"}},
	} {
		var got []string
		for _, d := range proposal.CodeRefProblems("/r", c.ref) {
			got = append(got, d.Pointer)
			if strings.Contains(d.Problem, c.ref.Path) || (c.ref.Lines != "" && strings.Contains(d.Problem, c.ref.Lines)) {
				t.Errorf("%+v: the problem quotes the value: %s", c.ref, d.Problem)
			}
		}
		if strings.Join(got, " ") != strings.Join(c.want, " ") {
			t.Errorf("%+v: problems at %v; want %v", c.ref, got, c.want)
		}
	}
}
