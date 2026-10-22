package claim_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/claim"
	"github.com/XKHoshizora/specmetry/internal/proposal"
)

func example(t *testing.T, name string) *claim.Batch {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "schemas", "examples", "valid", name))
	if err != nil {
		t.Fatal(err)
	}
	b, err := claim.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The examples of claims.v1 parse as written, and pass the rules that need
// no database.
func TestParseExamples(t *testing.T) {
	sync := example(t, "claims.sync.json")
	if sync.Kind != claim.Sync || sync.BaseVersion != 4 || sync.AuditScope != nil || len(sync.Claims) != 2 || len(sync.UnmappedWork) != 1 {
		t.Fatalf("claims.sync.json = %+v", sync)
	}
	c := sync.Claims[0]
	if c.Target != "AC-002-2" || c.TargetRevision != 1 || c.Assertion != "done" ||
		len(c.Refs.Files) != 1 || c.Refs.Files[0] != (proposal.CodeRef{Path: "internal/orders/query.go", Lines: "40-62"}) ||
		strings.Join(c.Refs.Tests, ",") != "TestOrderIsolation" || strings.Join(c.Refs.Commits, ",") != "3f2a9c1" {
		t.Fatalf("claim 0 = %+v", c)
	}
	audit := example(t, "claims.audit.json")
	if audit.Kind != claim.Audit || audit.AuditScope == nil || !audit.AuditScope.All || len(audit.AuditScope.Requirements) != 0 || len(audit.Claims) != 3 {
		t.Fatalf("claims.audit.json = %+v", audit)
	}
	for _, b := range []*claim.Batch{sync, audit} {
		if err := claim.CheckStatic(b); err != nil {
			t.Errorf("%s: %v", b.Kind, err)
		}
	}
}

// WP2.4 spec §5 step 4 (V-C03, V-C04, V-P04): a criterion is claimed at
// most once in a batch, the later claim pointed at; no summary holds a
// code fence or a blank line, whether or not the schema saw it; a file
// reference stays in the repository and its line range is valid, in claims
// and unmapped work alike. Every problem is reported at once, at its JSON
// Pointer, without the value; the error is invalid input (exit 2).
func TestCheckStatic(t *testing.T) {
	fence := "see\n```go\nx()\n```"
	b := &claim.Batch{
		Kind: claim.Sync, Summary: fence,
		Claims: []claim.Claim{
			{Target: "AC-001-1", TargetRevision: 1, Assertion: "done", Summary: "one\n\ntwo"},
			{Target: "AC-001-2", TargetRevision: 1, Assertion: "done", Summary: "fine",
				Refs: claim.Refs{Files: []proposal.CodeRef{{Path: "a/../../secret.go"}, {Path: "ok.go", Lines: "9-3"}}}},
			{Target: "AC-001-1", TargetRevision: 2, Assertion: "partial", Summary: "again"},
			{Target: "AC-001-2", TargetRevision: 1, Assertion: "blocked", Summary: "and again"},
		},
		UnmappedWork: []claim.Unmapped{
			{Summary: "a \n \t\n gap"},
			{Summary: "fine", Refs: claim.Refs{Files: []proposal.CodeRef{{Path: "ok.go"}, {Path: `..\up.go`, Lines: "0"}}}},
		},
	}
	err := claim.CheckStatic(b)
	if apperr.ExitCode(err) != apperr.ExitInvalidInput {
		t.Fatalf("CheckStatic = %v; want invalid input", err)
	}
	var got []string
	for _, d := range apperr.From(err).Details {
		got = append(got, d.Pointer)
		for _, value := range []string{"```", "secret.go", "9-3", "up.go", "gap"} {
			if strings.Contains(d.Problem, value) {
				t.Errorf("%s: the problem quotes %q: %s", d.Pointer, value, d.Problem)
			}
		}
	}
	want := []string{"/summary", "/claims/0/summary", "/claims/1/refs/files/0/path", "/claims/1/refs/files/1/lines",
		"/claims/2/target", "/claims/3/target", "/unmapped_work/0/summary", "/unmapped_work/1/refs/files/1/path",
		"/unmapped_work/1/refs/files/1/lines"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("problems at\n%v\nwant\n%v", got, want)
	}
	if msg := apperr.From(err).Message; !strings.Contains(msg, "nothing was written") {
		t.Errorf("message = %q", msg)
	}
}
