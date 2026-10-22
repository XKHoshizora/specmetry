package cli

import (
	"bytes"
	"compress/zlib"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/claim"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// cutBlob replaces the loose blob of path at HEAD with one whose header
// still inflates and states 500 bytes but holds only content: git reads it
// with exit 0. It returns a function that restores it.
func cutBlob(t *testing.T, f *testutil.Fixture, path, content string) func() {
	t.Helper()
	blob := f.Repo.Git("rev-parse", "HEAD:"+path)
	p := filepath.Join(f.Repo.Dir, ".git", "objects", blob[:2], blob[2:])
	original, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	z := zlib.NewWriter(&b)
	_, _ = z.Write([]byte("blob 500\x00" + content))
	_ = z.Close()
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return func() {
		t.Helper()
		if err := os.WriteFile(p, original, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// Review 068 B3 through the claim command: a file reference whose blob at
// the captured HEAD no longer reads as the object its ID names (cut short,
// git exits 0) is exit 6, not exists, with --json and in text, from a
// worktree the claim would register, alone or with healthy references: no
// batch, claim, unmapped work, worktree or seq; neither the blob's content
// nor git's stderr in any output or in the data directory. The batch is
// stored once the blob is repaired.
func TestClaimRefusesADamagedFileObject(t *testing.T) {
	home, f, dir := claimsProject(t)
	canary := testutil.NewCanary(t)
	doc := func(files ...any) map[string]any {
		d := claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", map[string]any{"files": files}))
		d["unmapped_work"] = []any{map[string]any{"summary": "canary-unmapped-summary"}}
		return d
	}
	if r := claimOK(t, home, dir, doc(fileRef("docs/client-brief.md", "1"))); r.Claims[0].Refs.marks() != "exists tests=0" {
		t.Fatalf("healthy: %s", r.Claims[0].Refs.marks())
	}
	wt := f.Repo.AddWorktree("feature-b3")
	repair := cutBlob(t, f, "docs/client-brief.md", canary+"\n")
	for _, files := range [][]any{{fileRef("docs/client-brief.md", "1")}, {fileRef("docs/none.md", ""), fileRef("docs/client-brief.md", "")}} {
		env := claimRefused(t, home, dir, doc(files...), 6, "")
		if strings.Contains(env.Error.Message, canary) || strings.Contains(env.Error.Message, "fatal") {
			t.Errorf("the error quotes the object or git: %s", env.Error.Message)
		}
	}
	before := factState(t, home)
	code, stdout, stderr := specmetry(t, wt, "", "claim", "--file", writeClaims(t, doc(fileRef("docs/client-brief.md", ""))))
	if code != apperr.ExitEnvironment || strings.Contains(stdout, "stored at seq") || strings.Contains(stdout+stderr, canary) || strings.Contains(stdout+stderr, "fatal") {
		t.Errorf("text, from a new worktree: exit %d\n%s%s", code, stdout, stderr)
	}
	if after := factState(t, home); after != before {
		t.Errorf("text, from a new worktree, wrote:\n%s\n---\n%s", before, after)
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Errorf("the blob's content reached the data directory: %v", hits)
	}
	repair()
	if r := claimOK(t, home, wt, doc(fileRef("docs/client-brief.md", "1"))); r.Claims[0].Refs.marks() != "exists tests=0" {
		t.Errorf("repaired: %s", r.Claims[0].Refs.marks())
	}
}

// Review 068 B3 on the path propose shares: a code reference whose blob at
// HEAD no longer reads as the object its ID names is exit 6, with --json
// and in text: nothing written, no seq, nothing of the blob or of git's
// stderr in the output. Repaired, the proposal is stored with the
// reference found.
func TestProposeRefusesADamagedFileObject(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	canary := testutil.NewCanary(t)
	item := addReq("r1", "sign up", nil)
	item["code_refs"] = []any{map[string]any{"path": "docs/client-brief.md", "lines": "1"}}
	doc := small(0, item)
	repair := cutBlob(t, f, "docs/client-brief.md", canary+"\n")
	env := proposeRefused(t, home, dir, doc, 6, "")
	if strings.Contains(env.Error.Message, canary) || strings.Contains(env.Error.Message, "fatal") {
		t.Errorf("the error quotes the object or git: %s", env.Error.Message)
	}
	before := written(t, home)
	code, stdout, stderr := specmetry(t, dir, "", "propose", "--file", writeProposal(t, doc))
	if code != apperr.ExitEnvironment || strings.Contains(stdout+stderr, canary) || strings.Contains(stdout+stderr, "fatal") || strings.Contains(stdout, "stored at seq") {
		t.Errorf("text: exit %d\n%s%s", code, stdout, stderr)
	}
	if after := written(t, home); after != before {
		t.Errorf("text wrote %s -> %s", before, after)
	}
	repair()
	r := proposeOK(t, dir, doc)
	if len(r.Items) != 1 || len(r.Items[0].CodeRefs) != 1 || !r.Items[0].CodeRefs[0].Exists {
		t.Errorf("repaired: %+v", r.Items)
	}
}
