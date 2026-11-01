package checkrun_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/checkrun"
)

// WP2.5 spec §19.8 (G11): a candidate is excluded when its final path is a
// root or inside one, the prefix ending at a separator; letter case is
// folded on Windows and macOS and kept on Linux, which the caller passes.
func TestWithin(t *testing.T) {
	sep := string(filepath.Separator)
	root := sep + "r" + sep + "W"
	cases := []struct {
		p          string
		fold, want bool
	}{
		{root, false, true},
		{root + sep + "bin" + sep + "sh", false, true},
		{sep + "r" + sep + "Wx" + sep + "sh", false, false},
		{sep + "r" + sep + "Wx" + sep + "sh", true, false},
		{sep + "r" + sep + "w" + sep + "bin" + sep + "sh", false, false},
		{sep + "r" + sep + "w" + sep + "bin" + sep + "sh", true, true},
		{sep + "R" + sep + "W", true, true},
		{sep + "r", false, false},
	}
	for _, c := range cases {
		if got := checkrun.Within(c.p, root, c.fold); got != c.want {
			t.Errorf("Within(%q, %q, fold %v) = %v", c.p, root, c.fold, got)
		}
	}
	if !checkrun.Within(sep+"any"+sep+"sh", sep, false) {
		t.Error("a root that ends with a separator does not hold what is under it")
	}
}

// WP2.5 spec §19.8 (G11): when the final path of a worktree to exclude
// cannot be had, where an interpreter is cannot be verified, and no shell
// is available; the problem quotes no path.
func TestLookupWithAnUnresolvableWorktree(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "gone")
	f := checkrun.NewFinder([]string{t.TempDir(), gone}, os.Getenv("PATH"), func(context.Context) (string, error) {
		t.Fatal("the exec path was read")
		return "", nil
	})
	for _, shell := range []string{"sh", "bash", "cmd", "pwsh", "powershell"} {
		in, problem, err := checkrun.Lookup(ctx, f, shell)
		if err != nil || in.Path != "" || !strings.Contains(problem, "cannot be verified") || strings.Contains(problem, gone) {
			t.Errorf("%s: %+v, %q, %v", shell, in, problem, err)
		}
	}
}

// WP2.5 spec §4.1 step 7 (D1; MVP-EV-20): each item gets its shell's
// interpreter; when one cannot be used the batch is refused before
// anything runs, shell_unavailable at every item needing it, in key
// order, without a path, nothing written. A registered worktree whose
// path is gone makes every interpreter unavailable (spec §19.8; plan
// §11.11 item 1).
func TestPrepareInterpreters(t *testing.T) {
	w := newWorkspace(t)
	w.criteria(t, ac{key: "AC-001-1", check: sh("echo one")}, ac{key: "AC-001-2", check: shell("bash", "echo two")},
		ac{key: "AC-001-3", check: shell("sh", "echo three")})
	b, err := run(t, checkrun.NewHost(), all())
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range b.Items {
		want := "sh"
		if it.Criterion == "AC-001-2" {
			want = "bash"
		}
		fi, err := os.Stat(it.Interpreter.Path)
		if it.Interpreter.Shell != want || !filepath.IsAbs(it.Interpreter.Path) || err != nil || !fi.Mode().IsRegular() {
			t.Errorf("%s: %+v (%v)", it.Criterion, it.Interpreter, err)
		}
	}
	if b.Items[0].Interpreter != b.Items[2].Interpreter {
		t.Errorf("one shell, two interpreters: %+v, %+v", b.Items[0].Interpreter, b.Items[2].Interpreter)
	}
	closeBatch(t, b)

	gone := filepath.Join(filepath.Dir(w.repo.Dir), "gone-worktree")
	w.exec(t, `INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES ('01JC0000000000000000000W03', ?, ?, NULL, 0, 3, 1)`, wsProject, gone)
	before := w.state(t)
	h := checkrun.NewHost()
	_, err = run(t, h, all())
	doc := refused(t, h, err, apperr.ExitEnvironment, "AC-001-1", "AC-001-2", "AC-001-3")
	for _, d := range apperr.From(err).Details {
		if !strings.HasPrefix(d.Problem, "shell_unavailable: ") {
			t.Errorf("%s: %q", d.Pointer, d.Problem)
		}
	}
	if strings.Contains(doc, "gone-worktree") || strings.Contains(doc, filepath.Base(w.repo.Dir)) {
		t.Errorf("the refusal quotes a path: %s", doc)
	}
	if after := w.state(t); after != before {
		t.Errorf("a refusal wrote:\n%s\n->\n%s", before, after)
	}
}

// WP2.5 spec §4.1: step 6 is judged before step 7.
func TestPrepareStaticRulesBeforeInterpreters(t *testing.T) {
	w := newWorkspace(t)
	w.criteria(t, ac{key: "AC-001-1", check: sh("make\r")})
	w.exec(t, `INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES ('01JC0000000000000000000W03', ?, ?, NULL, 0, 3, 1)`, wsProject, filepath.Join(t.TempDir(), "gone"))
	h := checkrun.NewHost()
	_, err := run(t, h, all())
	refused(t, h, err, apperr.ExitInvalidInput, "AC-001-1")
}
