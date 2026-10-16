package cli

import (
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// codex-review-034 B1: the note of an interactive scope confirmation (-m)
// is checked as the scope.v1 document the confirmation stands for, like
// --file, before anybody is asked. On both entries, a note with a code
// fence (a synthetic one, a fence inside a line), of 1001 characters, or
// empty is exit 2 at /note with error.v1, no prompt, no seq, nothing
// written, and the note is in neither the output nor the data directory;
// a review's scope note is checked by review.v1 (/scope/note). A note of
// 1000 characters (code points, not bytes), Chinese, Japanese and emoji,
// and no note at all are confirmed on both entries, and stored as given
// (NULL for none).
func TestScopeNoteIsCheckedLikeAFile(t *testing.T) {
	home, f := scopeProject(t)
	dir := f.Repo.Dir
	canary := testutil.NewCanary(t)
	fence := "```go\nfunc " + strings.ReplaceAll(canary, "-", "_") + "() {}\n```"
	for _, c := range []struct{ what, note string }{
		{"a code fence", fence},
		{"a fence inside a line", "see ``` " + canary},
		{"1001 characters", strings.Repeat("备", 1001)},
		{"an empty note", ""},
	} {
		ts := targetScope(t, dir)
		file := scopeFile(t, ts, ts.ConfirmInputs.MustRequirements, func(d map[string]any) { d["note"] = c.note })
		for _, args := range [][]string{{"scope", "confirm", "--file", file}, {"scope", "confirm", "-m", c.note}} {
			t.Logf("%s: %s", c.what, args[2])
			humanRefused(t, home, dir, "v2", apperr.ExitInvalidInput, "/note", canary, args...)
		}
	}

	// In a review: the review.v1 schema checks the scope's note.
	r := proposeOK(t, dir, small(2, addReq("z", "export to CSV", nil)))
	doc := reviewDoc(r.ProposalID, 2, decide(r.Items[0].ItemID, "accepted"))
	scope := reviewScope(targetScope(t, dir), 2, "REQ-001")
	scope["note"] = fence
	doc["scope"] = scope
	reviewRefused(t, home, dir, doc, apperr.ExitInvalidInput, "/scope/note")
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the refused note reached the data directory: %v", hits)
	}

	last := func() string {
		t.Helper()
		return rows(t, home, `SELECT coalesce(note, 'NULL') FROM scope_confirmations ORDER BY seq DESC LIMIT 1`)[0][0]
	}
	for _, note := range []string{strings.Repeat("备", 1000), "范围已与客户确认：首期上线 🚀", "スコープを確認しました ✅"} {
		ts := targetScope(t, dir)
		file := scopeFile(t, ts, ts.ConfirmInputs.MustRequirements, func(d map[string]any) { d["note"] = note })
		for _, args := range [][]string{{"scope", "confirm", "--file", file}, {"scope", "confirm", "-m", note}} {
			if code, stdout, stderr := asHuman(t, dir, "v2", args...); code != apperr.ExitOK {
				t.Fatalf("%s %.20q: exit %d\n%s%s", args[2], note, code, stdout, stderr)
			}
			if got := last(); got != note {
				t.Fatalf("%s: stored note %.20q, want %.20q", args[2], got, note)
			}
		}
	}
	ts := targetScope(t, dir)
	for _, args := range [][]string{{"scope", "confirm", "--file", scopeFile(t, ts, ts.ConfirmInputs.MustRequirements, nil)}, {"scope", "confirm"}} {
		if code, stdout, stderr := asHuman(t, dir, "v2", args...); code != apperr.ExitOK {
			t.Fatalf("%v without a note: exit %d\n%s%s", args, code, stdout, stderr)
		}
		if got := last(); got != "NULL" {
			t.Fatalf("%v without a note: stored %q", args, got)
		}
	}
}

// codex-review-034 B2: a target name holds no code fence (architecture
// §10.2), whether given with --name or taken from the ref: exit 2 at
// --name before anybody is asked, with error.v1, nothing echoed, no target
// created or archived, the scope kept and no seq taken; also for the same
// ref renamed. A ref whose name would have a fence is set with a valid
// --name. Chinese, Japanese, emoji, and single and double backticks are
// valid names, set and read back as they are.
func TestTargetNameHasNoCodeFence(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	acceptAll(t, dir, proposeOK(t, dir, small(0, addReq("a", "sign up", nil))), 0, nil)
	if code, stdout, stderr := asHuman(t, dir, "v1", "scope", "confirm"); code != apperr.ExitOK {
		t.Fatalf("scope confirm: exit %d\n%s%s", code, stdout, stderr)
	}
	before := targetScope(t, dir)
	canary := testutil.NewCanary(t)
	branch := f.Repo.Git("symbolic-ref", "HEAD")
	fencedRef := "refs/heads/rel```x"
	for _, c := range []struct {
		args   []string
		hidden string
	}{
		{[]string{"target", "set", "--ref", "refs/heads/release", "--name", "```go func " + strings.ReplaceAll(canary, "-", "_") + "() {} ```"}, canary},
		{[]string{"target", "set", "--ref", "refs/heads/release", "--name", "release ``` " + canary}, canary},
		{[]string{"target", "set", "--ref", branch, "--name", "```" + canary}, canary},
		{[]string{"target", "set", "--ref", fencedRef}, "rel```x"},
	} {
		env := humanRefused(t, home, dir, c.args[3], apperr.ExitInvalidInput, "--name", c.hidden, c.args...)
		if len(c.args) == 4 && !strings.Contains(env.Error.Message, "--name") {
			t.Errorf("a default name that is not valid: the message does not point to --name: %s", env.Error.Message)
		}
	}
	if after := targetScope(t, dir); after.Target.ID != before.Target.ID || after.Scope == nil || after.Scope.ConfirmationID != before.Scope.ConfirmationID {
		t.Fatalf("the target or its scope changed: %+v %+v", after.Target, after.Scope)
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("a refused name reached the data directory: %v", hits)
	}

	if code, stdout, stderr := asHuman(t, dir, fencedRef, "target", "set", "--ref", fencedRef, "--name", "release candidate"); code != apperr.ExitOK {
		t.Fatalf("the ref with a valid --name: exit %d\n%s%s", code, stdout, stderr)
	}
	for _, name := range []string{"首期交付", "第一リリース", "🚀 launch", "`v1`", "``v1`` final"} {
		if code, stdout, stderr := asHuman(t, dir, "refs/heads/release", "target", "set", "--ref", "refs/heads/release", "--name", name); code != apperr.ExitOK {
			t.Fatalf("--name %q: exit %d\n%s%s", name, code, stdout, stderr)
		}
		if ts := targetScope(t, dir); ts.Target.Name != name || ts.Target.Ref != "refs/heads/release" {
			t.Fatalf("--name %q: target %+v", name, ts.Target)
		}
	}
}
