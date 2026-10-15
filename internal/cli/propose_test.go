package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

type receiptOut struct {
	ProposalID     string `json:"proposal_id"`
	Seq            int64  `json:"seq"`
	Origin         string `json:"origin"`
	BaseVersion    int    `json:"base_version"`
	AppliedVersion *int   `json:"applied_version"`
	Items          []struct {
		ItemID      string  `json:"item_id"`
		ItemIndex   int     `json:"item_index"`
		Op          string  `json:"op"`
		Type        string  `json:"type"`
		Ref         *string `json:"ref"`
		Target      *string `json:"target"`
		ReviewState string  `json:"review_state"`
		AssignedKey *string `json:"assigned_key"`
		Sources     []struct {
			SourceID     string `json:"source_id"`
			Verification string `json:"verification"`
		} `json:"sources"`
		CodeRefs []struct {
			Path   string `json:"path"`
			Exists bool   `json:"exists"`
		} `json:"code_refs"`
	} `json:"items"`
	Questions []struct {
		Key      string  `json:"key"`
		About    *string `json:"about"`
		Blocking bool    `json:"blocking"`
	} `json:"questions"`
}

// proposalsProject is fx-docs-only, initialized, with its client brief
// registered as a source.
func proposalsProject(t *testing.T) (home datadir.Dir, f *testutil.Fixture, brief string) {
	t.Helper()
	home = testutil.TempHome(t)
	f = testutil.BuildFixture(t, "fx-docs-only")
	mustInit(t, f.Repo.Dir)
	brief = sourceAdd(t, f.Repo.Dir, "docs/client-brief.md").Source.ID
	return home, f, brief
}

// recorded is the recorded fx-docs-only proposal (WP1.0), decoded.
func recorded(t *testing.T, f *testutil.Fixture) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(f.Source, "proposal.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

// clone copies a decoded document.
func clone(t *testing.T, doc any) any {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// patched returns a copy of doc with the value at pointer set. "-" as the
// last token appends to an array.
func patched(t *testing.T, doc any, pointer string, value any) any {
	t.Helper()
	out := clone(t, doc)
	tokens := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	parent := out
	for _, tok := range tokens[:len(tokens)-1] {
		switch p := parent.(type) {
		case map[string]any:
			parent = p[tok]
		case []any:
			i, _ := strconv.Atoi(tok)
			parent = p[i]
		}
	}
	last := tokens[len(tokens)-1]
	switch p := parent.(type) {
	case map[string]any:
		p[last] = value
	case []any:
		// Arrays are changed in place through their parent; rebuild it.
		if last != "-" {
			i, _ := strconv.Atoi(last)
			p[i] = value
			return out
		}
		return patched(t, out, strings.TrimSuffix(pointer, "/-"), append(p, value))
	}
	return out
}

func writeProposal(t *testing.T, doc any) string {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "proposal.json")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func proposeOK(t *testing.T, dir string, doc any) receiptOut {
	t.Helper()
	code, stdout, stderr := specmetry(t, dir, "proposal-receipt.v1", "propose", "--file", writeProposal(t, doc), "--json")
	if code != apperr.ExitOK {
		t.Fatalf("propose: exit %d\n%s%s", code, stdout, stderr)
	}
	var r receiptOut
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

// written counts the rows a proposal writes, and the seq counter.
func written(t *testing.T, home datadir.Dir) string {
	t.Helper()
	return strings.Join([]string{
		seq(t, home),
		strconv.Itoa(count(t, home, `SELECT count(*) FROM proposals`)),
		strconv.Itoa(count(t, home, `SELECT count(*) FROM proposal_items`)),
		strconv.Itoa(count(t, home, `SELECT count(*) FROM questions`)),
		strconv.Itoa(count(t, home, `SELECT count(*) FROM source_refs`)),
		strconv.Itoa(count(t, home, `SELECT count(*) FROM key_counters`)),
		strconv.Itoa(count(t, home, `SELECT count(*) FROM worktrees`)),
	}, "/")
}

// proposeRefused submits doc and checks the refusal: the exit code, an
// error.v1 with a detail at pointer (or the current values for a stale
// precondition), and nothing written.
func proposeRefused(t *testing.T, home datadir.Dir, dir string, doc any, code int, pointer string) apperr.Envelope {
	t.Helper()
	before := written(t, home)
	got, stdout, stderr := specmetry(t, dir, "proposal-receipt.v1", "propose", "--file", writeProposal(t, doc), "--json")
	if got != code {
		t.Errorf("exit %d, want %d (%s)\n%s%s", got, code, pointer, stdout, stderr)
	}
	var env apperr.Envelope
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Errorf("%s: no error.v1: %v\n%s", pointer, err, stdout)
	}
	if pointer != "" {
		found := false
		for _, d := range env.Error.Details {
			found = found || d.Pointer == pointer
		}
		if !found {
			t.Errorf("no detail at %s: %+v (%s)", pointer, env.Error.Details, env.Error.Message)
		}
	}
	if after := written(t, home); after != before {
		t.Errorf("%s: written %s -> %s (seq/proposals/items/questions/source refs/counters/worktrees)", pointer, before, after)
	}
	return env
}

func digest(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// noAgentVariables makes the test's channel agent_cli without an Agent
// variable, whatever environment the test runs in.
func noAgentVariables(t *testing.T) {
	t.Setenv("CLAUDECODE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
}

// MVP-BL-01 (M1), BL-28: the recorded fx-docs-only proposal is stored as
// one proposal of 18 pending items (1 goal, 8 requirements, 9 criteria)
// and 4 open questions, Q-001 to Q-004; no baseline entity or version is
// created; one seq is used. Every expected value is written out here, not
// derived from the proposal.
func TestProposeRecordedProposal(t *testing.T) {
	home, f, brief := proposalsProject(t)
	t.Setenv("CLAUDECODE", "1")
	before, err := strconv.Atoi(seq(t, home))
	if err != nil {
		t.Fatal(err)
	}
	r := proposeOK(t, f.Repo.Dir, recorded(t, f))

	if r.Seq != int64(before+1) || r.Origin != "skill_init" || r.BaseVersion != 0 || r.AppliedVersion != nil {
		t.Fatalf("receipt: seq %d (before %d), origin %s, base %d, applied %v", r.Seq, before, r.Origin, r.BaseVersion, r.AppliedVersion)
	}
	wantTypes := []string{"goal", "requirement", "criterion", "requirement", "criterion", "criterion", "requirement", "criterion",
		"requirement", "criterion", "requirement", "criterion", "criterion", "requirement", "criterion", "requirement", "criterion", "requirement"}
	wantRefs := []string{"g1", "r1", "c1", "r2", "c2", "c3", "r3", "c4", "r4", "c5", "r5", "c6", "c7", "r6", "c8", "r7", "c9", "r8"}
	if len(r.Items) != 18 {
		t.Fatalf("%d items, want 18", len(r.Items))
	}
	ids := map[string]bool{}
	for i, it := range r.Items {
		if it.ItemIndex != i || it.Op != "add" || it.Type != wantTypes[i] || it.Ref == nil || *it.Ref != wantRefs[i] ||
			it.Target != nil || it.ReviewState != "pending" || it.AssignedKey != nil {
			t.Errorf("item %d: %+v", i, it)
		}
		if len(it.Sources) != 1 || it.Sources[0].SourceID != brief || it.Sources[0].Verification != "verified" || len(it.CodeRefs) != 0 {
			t.Errorf("item %d references: %+v %+v", i, it.Sources, it.CodeRefs)
		}
		ids[it.ItemID] = true
	}
	if len(ids) != 18 {
		t.Errorf("%d distinct item IDs", len(ids))
	}
	wantAbout := []any{"r3", "r7", nil, "r8"}
	wantBlocking := []bool{false, false, true, false}
	if len(r.Questions) != 4 {
		t.Fatalf("%d questions, want 4", len(r.Questions))
	}
	for i, q := range r.Questions {
		about := any(nil)
		if q.About != nil {
			about = *q.About
		}
		if q.Key != "Q-00"+strconv.Itoa(i+1) || about != wantAbout[i] || q.Blocking != wantBlocking[i] {
			t.Errorf("question %d: %+v", i, q)
		}
	}

	// The database, read independently of the command.
	for query, want := range map[string]int{
		`SELECT count(*) FROM proposals WHERE origin = 'skill_init' AND actor = 'agent:claude_code' AND channel = 'agent_cli' AND base_version = 0`: 1,
		`SELECT count(*) FROM proposal_items WHERE review_state = 'pending' AND reviewed_seq IS NULL AND applied_version_id IS NULL`:                18,
		`SELECT count(*) FROM proposal_items WHERE entity_type = 'goal'`:                                                                            1,
		`SELECT count(*) FROM proposal_items WHERE entity_type = 'requirement'`:                                                                     8,
		`SELECT count(*) FROM proposal_items WHERE entity_type = 'criterion'`:                                                                       9,
		`SELECT count(*) FROM questions WHERE status = 'open' AND created_seq = ` + strconv.FormatInt(r.Seq, 10):                                    4,
		`SELECT count(*) FROM questions WHERE blocking = 1`:                                                                                         1,
		`SELECT count(*) FROM source_refs WHERE entity_type = 'question' AND excerpt IS NULL AND verification = 'verified'`:                         4,
		`SELECT count(*) FROM goals`:               0,
		`SELECT count(*) FROM requirements`:        0,
		`SELECT count(*) FROM acceptance_criteria`: 0,
		`SELECT count(*) FROM baseline_versions`:   0,
	} {
		if n := count(t, home, query); n != want {
			t.Errorf("%s: %d, want %d", query, n, want)
		}
	}
	// A question about a proposal-local ref names the item until a review
	// accepts it; one without about names nothing.
	for _, c := range []struct{ key, ref string }{{"Q-001", "r3"}, {"Q-002", "r7"}, {"Q-004", "r8"}, {"Q-003", ""}} {
		got := rows(t, home, `SELECT coalesce(i.ref, ''), coalesce(q.related_type, ''), coalesce(q.related_id, '')
FROM questions q LEFT JOIN proposal_items i ON i.id = q.proposal_item_id WHERE q.key = ?`, c.key)
		if len(got) != 1 || got[0][0] != c.ref || got[0][1] != "" || got[0][2] != "" {
			t.Errorf("%s relates to %v, want item %q", c.key, got, c.ref)
		}
	}
	// hash_only: the stored references hold the excerpt's hash, never its
	// text; the hashes are computed here from the expected excerpts.
	excerpts := map[string]string{
		"Q-001": "每页显示合适数量的订单", "Q-002": "系统应尽快通知创建订单的店员",
		"Q-003": "区域经理是否也可以创建订单，目前还没有结论。", "Q-004": "店员无需培训即可上手",
	}
	for key, excerpt := range excerpts {
		got := rows(t, home, `SELECT r.excerpt_sha256, r.locator FROM source_refs r JOIN questions q ON q.id = r.entity_id WHERE q.key = ?`, key)
		if len(got) != 1 || got[0][0] != digest(excerpt) {
			t.Errorf("%s reference %v, want the hash of %q", key, got, excerpt)
		}
	}
	for _, raw := range rows(t, home, `SELECT sources_json FROM proposal_items`) {
		if strings.Contains(raw[0], `"excerpt"`) || !strings.Contains(raw[0], `"excerpt_sha256"`) {
			t.Errorf("sources_json %s", raw[0])
		}
	}
	if got := rows(t, home, `SELECT sources_json FROM proposal_items WHERE ref = 'r1'`)[0][0]; !strings.Contains(got, digest("连续 5 次输错密码后，账号锁定 15 分钟。")) {
		t.Errorf("r1 sources_json %s lacks the excerpt's hash", got)
	}
}

// The actor names the Agent when one of its variables is set, and is
// agent:unknown in agent_cli otherwise.
func TestProposeActor(t *testing.T) {
	home, f, _ := proposalsProject(t)
	noAgentVariables(t)
	small := map[string]any{"schema": "specmetry.proposal/v1", "origin": "cli", "base_version": 0, "summary": "one goal",
		"items": []any{map[string]any{"op": "add", "type": "goal", "ref": "g1", "data": map[string]any{"title": "launch"}, "origin": "manual"}}}
	proposeOK(t, f.Repo.Dir, small)
	if got := rows(t, home, `SELECT actor, channel FROM proposals`); len(got) != 1 || got[0][0] != "agent:unknown" || got[0][1] != "agent_cli" {
		t.Fatalf("actor %v", got)
	}
}

// MVP-BL-02, BL-21, BL-23 (M1), BL-04 (length), BL-05 (M1), MVP-PRV-07
// (CLI): each refusal rule (V-P01, V-P03 – V-P06, V-P08) exits 2 with a
// JSON Pointer to the field and writes nothing.
func TestProposeRefuses(t *testing.T) {
	home, f, _ := proposalsProject(t)
	base := recorded(t, f)
	secret := "ghp_" + strings.Repeat("A1b2", 9)
	update := func(target, typ string) map[string]any {
		return map[string]any{"op": "update", "type": typ, "target": target, "target_revision": 1, "data": map[string]any{"title": "x"}}
	}
	check := func(c map[string]any) any { return patched(t, base, "/items/2/data/check", c) }
	withLines := func(lines string) any {
		inferred := patched(t, patched(t, base, "/items/1/origin", "inferred_from_code"), "/items/1/rationale", "seen in the code")
		return patched(t, inferred, "/items/1/code_refs", []any{map[string]any{"path": "docs/client-brief.md", "lines": lines}})
	}
	nines := strings.Repeat("9", 30)
	cases := []struct {
		name    string
		doc     any
		pointer string
	}{
		{"an unknown property", patched(t, base, "/extra", true), "/"},
		{"a summary with a blank line", patched(t, base, "/summary", "first\n\nsecond"), "/summary"},
		{"a multi-line title", patched(t, base, "/items/0/data/title", "first\nsecond"), "/items/0/data/title"},
		{"a code fence in a description", patched(t, base, "/items/0/data/description", "see\n```\nrm -rf /\n```"), "/items/0/data/description"},
		{"a code fence in an excerpt", patched(t, base, "/items/0/sources/0/excerpt", "```首期只覆盖下单和查询"), "/items/0/sources/0/excerpt"},
		{"a code fence in a locator", patched(t, base, "/items/0/sources/0/locator", "```"), "/items/0/sources/0/locator"},
		{"a secret in a statement", patched(t, base, "/items/2/data/statement", "use token "+secret), "/items/2/data/statement"},
		{"a secret in an excerpt", patched(t, base, "/items/0/sources/0/excerpt", secret), "/items/0/sources/0/excerpt"},
		{"a secret in the summary", patched(t, base, "/summary", "token "+secret), "/summary"},
		{"a duplicate ref", patched(t, base, "/items/2/ref", "r1"), "/items/2/ref"},
		{"a criterion of an unknown ref", patched(t, base, "/items/2/data/requirement", "r99"), "/items/2/data/requirement"},
		{"a criterion of a goal ref", patched(t, base, "/items/2/data/requirement", "g1"), "/items/2/data/requirement"},
		{"a requirement under a criterion ref", patched(t, base, "/items/1/data/goal", "c1"), "/items/1/data/goal"},
		{"a question about a goal ref", patched(t, base, "/questions/0/about", "g1"), "/questions/0/about"},
		{"a question about an unknown ref", patched(t, base, "/questions/0/about", "x9"), "/questions/0/about"},
		{"an update of an unknown key", patched(t, base, "/items/-", update("REQ-999", "requirement")), "/items/18/target"},
		{"a criterion of an unknown key", patched(t, base, "/items/2/data/requirement", "REQ-999"), "/items/2/data/requirement"},
		{"a question about an unknown key", patched(t, base, "/questions/0/about", "AC-001-1"), "/questions/0/about"},
		{"the same target twice", patched(t, patched(t, base, "/items/-", update("REQ-001", "requirement")), "/items/-",
			map[string]any{"op": "retire", "type": "requirement", "target": "REQ-001", "target_revision": 1, "rationale": "gone"}), "/items/19/target"},
		{"a target of another type", patched(t, base, "/items/-",
			map[string]any{"op": "retire", "type": "goal", "target": "REQ-001", "target_revision": 1, "rationale": "gone"}), "/items/18/target"},
		{"a check without a rationale", check(map[string]any{"command": "go test ./..."}), "/items/2/data/check"},
		{"a secret in a check command", check(map[string]any{"command": "curl -H 'Authorization: token " + secret + "' x", "rationale": "r"}),
			"/items/2/data/check/command"},
		{"a workdir outside the repository", check(map[string]any{"command": "go test ./...", "rationale": "r", "workdir": "../outside"}),
			"/items/2/data/check/workdir"},
		{"an input outside the repository", check(map[string]any{"command": "go test ./...", "rationale": "r", "inputs": []any{"a/../../b"}}),
			"/items/2/data/check/inputs/0"},
		{"a timeout of 0", check(map[string]any{"command": "go test ./...", "rationale": "r", "timeout_sec": 0}), "/items/2/data/check/timeout_sec"},
		{"a timeout over an hour", check(map[string]any{"command": "go test ./...", "rationale": "r", "timeout_sec": 3601}),
			"/items/2/data/check/timeout_sec"},
		{"a check on a manual criterion", patched(t, check(map[string]any{"command": "go test ./...", "rationale": "r"}),
			"/items/2/data/verification", "manual"), "/items/2/data/verification"},
		{"an unregistered source", patched(t, base, "/items/0/sources/0/source", "docs/unknown.md"), "/items/0/sources/0/source"},
		{"an excerpt over 500 characters", patched(t, base, "/items/0/sources/0/excerpt", strings.Repeat("订", 501)), "/items/0/sources/0/excerpt"},
		{"an excerpt of whitespace only", patched(t, base, "/items/0/sources/0/excerpt", " \r\n\t　"), "/items/0/sources/0/excerpt"},
		{"inferred_from_code without code references", patched(t, patched(t, base, "/items/1/origin", "inferred_from_code"),
			"/items/1/rationale", "seen in the code"), "/items/1"},
		{"a reversed line range", withLines("9-3"), "/items/1/code_refs/0/lines"},
		// codex-review-020 B4: a number too large is a malformed range,
		// never "no range". Lines are numbered up to 2147483647 (32 bits).
		{"a line number of 30 digits", withLines(nines), "/items/1/code_refs/0/lines"},
		{"a range starting past the limit", withLines(nines + "-1"), "/items/1/code_refs/0/lines"},
		{"a range ending past the limit", withLines("1-" + nines), "/items/1/code_refs/0/lines"},
		{"a line past the 32-bit limit", withLines("2147483648"), "/items/1/code_refs/0/lines"},
		{"a range ending past the 32-bit limit", withLines("1-2147483648"), "/items/1/code_refs/0/lines"},
		{"the 64-bit limit", withLines("9223372036854775807"), "/items/1/code_refs/0/lines"},
		{"past the 64-bit limit", withLines("9223372036854775808"), "/items/1/code_refs/0/lines"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := proposeRefused(t, home, f.Repo.Dir, c.doc, apperr.ExitInvalidInput, c.pointer)
			if env.Error.Code != "invalid_input" {
				t.Errorf("code %s", env.Error.Code)
			}
			if b, _ := json.Marshal(env); bytes.Contains(b, []byte(secret)) {
				t.Errorf("the error echoes the secret: %s", b)
			}
		})
	}
	if hits := testutil.FindInTree(t, home.Path, secret); len(hits) != 0 {
		t.Fatalf("the secret reached the data directory: %v", hits)
	}
}

// V-P01, MVP-BL-13 (Agent part): human_edit and propose --apply are a
// person's; from the Agent channel they exit 3 and write nothing. In a
// terminal, propose --apply asks for confirmation (WP1.6; not confirmed,
// exit 3 and nothing written; applying it: review_test), and human_edit
// without --apply is refused (exit 2) rather than taken as a pending
// proposal. A person's ordinary proposal stays pending, with a human actor.
func TestProposeHumanEdit(t *testing.T) {
	home, f, _ := proposalsProject(t)
	edit := patched(t, recorded(t, f), "/origin", "human_edit")
	proposeRefused(t, home, f.Repo.Dir, edit, apperr.ExitHumanRequired, "")
	before := written(t, home)
	code, stdout, stderr := specmetry(t, f.Repo.Dir, "proposal-receipt.v1", "propose", "--apply", "--file", writeProposal(t, recorded(t, f)), "--json")
	if code != apperr.ExitHumanRequired {
		t.Errorf("propose --apply from the Agent channel: exit %d\n%s%s", code, stdout, stderr)
	}
	for _, c := range []struct {
		args   []string
		answer string
		exit   string
	}{
		{[]string{"propose", "--apply", "--file", writeProposal(t, edit), "--json"}, "", "exit=3"},
		{[]string{"propose", "--file", writeProposal(t, edit), "--json"}, noPrompt, "exit=2"},
	} {
		if out := inTerminal(t, f.Repo.Dir, c.args, c.answer); !strings.Contains(out, c.exit) {
			t.Errorf("%v in a terminal:\n%s", c.args[:2], out)
		}
	}
	if after := written(t, home); after != before {
		t.Fatalf("written %s -> %s", before, after)
	}
	out := inTerminal(t, f.Repo.Dir, []string{"propose", "--file", writeProposal(t, patched(t, recorded(t, f), "/origin", "cli")), "--json"}, noPrompt)
	if !strings.Contains(out, "exit=0") {
		t.Fatalf("a person's proposal:\n%s", out)
	}
	got := rows(t, home, `SELECT actor, channel FROM proposals`)
	if len(got) != 1 || !strings.HasPrefix(got[0][0], "human:") || got[0][1] != "human_cli" {
		t.Fatalf("proposal by %v", got)
	}
	if n := count(t, home, `SELECT count(*) FROM proposal_items WHERE review_state = 'pending'`); n != 18 {
		t.Fatalf("%d pending items", n)
	}
}

// seedBaseline writes a confirmed baseline straight into the database, as
// a review (WP1.6) would leave it: versions 1 to n, GOAL-1, REQ-001 at
// revision 2, AC-001-1 confirmed for it, and REQ-002 retired.
func seedBaseline(t *testing.T, home datadir.Dir, versions int) map[string]string {
	t.Helper()
	db, _, err := sqlite.Open(context.Background(), home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	ids := map[string]string{"GOAL-1": "01J9Z3K5Q8R2T4V6W8X0Y2Z4N1", "REQ-001": "01J9Z3K5Q8R2T4V6W8X0Y2Z4N2",
		"REQ-002": "01J9Z3K5Q8R2T4V6W8X0Y2Z4N4", "AC-001-1": "01J9Z3K5Q8R2T4V6W8X0Y2Z4N3"}
	err = db.WriteTx(context.Background(), func(tx *sql.Tx) error {
		var project string
		if err := tx.QueryRow(`SELECT id FROM projects`).Scan(&project); err != nil {
			return err
		}
		for n := 1; n <= versions; n++ {
			s, err := sqlite.NextSeq(context.Background(), tx)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO baseline_versions (id, seq, project_id, number, summary, actor, channel)
VALUES (?, ?, ?, ?, 'reviewed', 'human:me', 'human_cli')`, "01J9Z3K5Q8R2T4V6W8X0Y2Z5M"+strconv.Itoa(n), s, project, n); err != nil {
				return err
			}
		}
		_, err := tx.Exec(`
INSERT INTO goals (id, project_id, key, status, revision, revision_seq, origin, created_version, updated_version, updated_seq, title)
VALUES (?2, ?1, 'GOAL-1', 'active', 1, 1, 'extracted', 1, 1, 1, 'launch');
INSERT INTO requirements (id, project_id, key, status, revision, revision_seq, origin, created_version, updated_version,
    retired_version, updated_seq, goal_id, title, kind, priority)
VALUES (?3, ?1, 'REQ-001', 'active', 2, 1, 'extracted', 1, 1, NULL, 1, ?2, 'log in', 'functional', 'must'),
       (?4, ?1, 'REQ-002', 'retired', 1, 1, 'extracted', 1, 1, 1, 1, NULL, 'old', 'functional', 'must');
INSERT INTO acceptance_criteria (id, project_id, key, status, revision, revision_seq, origin, created_version, updated_version,
    updated_seq, requirement_id, requirement_revision, statement, verification)
VALUES (?5, ?1, 'AC-001-1', 'active', 1, 1, 'extracted', 1, 1, 1, ?3, 2, 'locked after five failures', 'manual');`,
			project, ids["GOAL-1"], ids["REQ-001"], ids["REQ-002"], ids["AC-001-1"])
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

// baselineState is the baseline as stored: every entity's key, status,
// revision and requirement revision.
func baselineState(t *testing.T, home datadir.Dir) [][]string {
	t.Helper()
	return rows(t, home, `SELECT key, status, revision, '' FROM goals UNION ALL SELECT key, status, revision, '' FROM requirements
UNION ALL SELECT key, status, revision, requirement_revision FROM acceptance_criteria ORDER BY 1`)
}

// MVP-BL-03, CX-14: a stale base_version or target_revision exits 5 with
// the current values and writes nothing; with the current values the
// proposal is stored, pending, and the baseline itself does not change.
func TestProposeStale(t *testing.T) {
	home, f, _ := proposalsProject(t)
	ids := seedBaseline(t, home, 4)
	small := func(base int, items ...any) map[string]any {
		return map[string]any{"schema": "specmetry.proposal/v1", "origin": "skill_goals", "base_version": base, "summary": "client mail",
			"items": append([]any{}, items...)}
	}
	updateReq := func(rev int) map[string]any {
		return map[string]any{"op": "update", "type": "requirement", "target": "REQ-001", "target_revision": rev,
			"data": map[string]any{"title": "log in with a password"}, "change_class": "material", "rationale": "client mail"}
	}
	// CX-14: read v3, a review made v4, the Agent submits base_version 3.
	env := proposeRefused(t, home, f.Repo.Dir, small(3, updateReq(2)), apperr.ExitStalePrecondition, "/base_version")
	if env.Error.Code != "stale_precondition" || env.Error.Current["base_version"] != float64(4) {
		t.Errorf("stale base_version: %+v", env.Error)
	}
	env = proposeRefused(t, home, f.Repo.Dir, small(4, updateReq(1)), apperr.ExitStalePrecondition, "/items/0/target_revision")
	if b, _ := json.Marshal(env.Error.Current); !bytes.Contains(b, []byte(`"REQ-001"`)) || !bytes.Contains(b, []byte(`"revision":2`)) {
		t.Errorf("stale target_revision: current %s", b)
	}
	proposeRefused(t, home, f.Repo.Dir, small(4, map[string]any{"op": "update", "type": "requirement", "target": "REQ-002",
		"target_revision": 1, "data": map[string]any{"title": "x"}}), apperr.ExitInvalidInput, "/items/0/target")
	// An entity is changed at most once per proposal, also when every
	// revision is current.
	proposeRefused(t, home, f.Repo.Dir, small(4, updateReq(2), map[string]any{"op": "retire", "type": "requirement",
		"target": "REQ-001", "target_revision": 2, "rationale": "merged"}), apperr.ExitInvalidInput, "/items/1/target")
	// A criterion's check needs it to be automated: AC-001-1 is manual.
	proposeRefused(t, home, f.Repo.Dir, small(4, map[string]any{"op": "update", "type": "criterion", "target": "AC-001-1",
		"target_revision": 1, "data": map[string]any{"check": map[string]any{"command": "go test ./...", "rationale": "r"}}}),
		apperr.ExitInvalidInput, "/items/0/data/check")

	stateBefore := baselineState(t, home)
	r := proposeOK(t, f.Repo.Dir, small(4,
		updateReq(2),
		map[string]any{"op": "reaffirm", "type": "criterion", "target": "AC-001-1", "target_revision": 1, "rationale": "still applies"},
		map[string]any{"op": "add", "type": "criterion", "ref": "c1", "origin": "extracted",
			"data": map[string]any{"requirement": "REQ-001", "statement": "a wrong password is refused", "verification": "manual"}},
		map[string]any{"op": "retire", "type": "goal", "target": "GOAL-1", "target_revision": 1, "rationale": "merged"},
	))
	wantTargets := []any{"REQ-001", "AC-001-1", nil, "GOAL-1"}
	for i, it := range r.Items {
		target := any(nil)
		if it.Target != nil {
			target = *it.Target
		}
		if target != wantTargets[i] || it.ReviewState != "pending" {
			t.Errorf("item %d: %+v", i, it)
		}
	}
	for key, want := range map[string]string{"REQ-001": ids["REQ-001"], "AC-001-1": ids["AC-001-1"], "GOAL-1": ids["GOAL-1"]} {
		if n := count(t, home, `SELECT count(*) FROM proposal_items WHERE target_id = '`+want+`'`); n != 1 {
			t.Errorf("%s: %d items target it", key, n)
		}
	}
	if got := rows(t, home, `SELECT suggested_change_class, change_class IS NULL FROM proposal_items WHERE op = 'update'`); got[0][0] != "material" || got[0][1] != "1" {
		t.Errorf("the suggested class is kept apart from the reviewer's: %v", got)
	}
	if after := baselineState(t, home); !slices.EqualFunc(after, stateBefore, slices.Equal) {
		t.Fatalf("the proposal changed the baseline: %v -> %v", stateBefore, after)
	}
	if n := count(t, home, `SELECT count(*) FROM baseline_versions`); n != 4 {
		t.Fatalf("%d versions, want 4", n)
	}
	// A question about an existing key relates to that entity at once.
	q := small(4)
	q["questions"] = []any{map[string]any{"text": "how long is the lock?", "about": "REQ-001", "blocking": true}}
	proposeOK(t, f.Repo.Dir, q)
	if got := rows(t, home, `SELECT related_type, related_id FROM questions`); len(got) != 1 || got[0][0] != "requirement" || got[0][1] != ids["REQ-001"] {
		t.Fatalf("question relates to %v", got)
	}
}

// codex-review-020 B3: a requirement without a goal ("goal": null) is
// accepted, both added and when an update clears its goal; the null stays
// in the payload for the review to apply. Goal keys are still checked.
func TestProposeGoalNull(t *testing.T) {
	home, f, _ := proposalsProject(t)
	seedBaseline(t, home, 1)
	doc := map[string]any{"schema": "specmetry.proposal/v1", "origin": "skill_goals", "base_version": 1, "summary": "no goal",
		"items": []any{
			map[string]any{"op": "add", "type": "requirement", "ref": "r1", "origin": "manual",
				"data": map[string]any{"title": "export orders", "kind": "functional", "priority": "could", "goal": nil}},
			map[string]any{"op": "update", "type": "requirement", "target": "REQ-001", "target_revision": 2,
				"data": map[string]any{"goal": nil}},
		}}
	proposeRefused(t, home, f.Repo.Dir, patched(t, doc, "/items/0/data/goal", "GOAL-9"), apperr.ExitInvalidInput, "/items/0/data/goal")
	proposeRefused(t, home, f.Repo.Dir, patched(t, doc, "/items/1/data/goal", "REQ-001"), apperr.ExitInvalidInput, "/items/1/data/goal")
	r := proposeOK(t, f.Repo.Dir, doc)
	if len(r.Items) != 2 || r.Items[0].ReviewState != "pending" || r.Items[1].Target == nil || *r.Items[1].Target != "REQ-001" {
		t.Fatalf("receipt items %+v", r.Items)
	}
	for _, row := range rows(t, home, `SELECT payload_json FROM proposal_items ORDER BY item_index`) {
		var data map[string]any
		if err := json.Unmarshal([]byte(row[0]), &data); err != nil {
			t.Fatal(err)
		}
		if goal, ok := data["goal"]; !ok || goal != nil {
			t.Errorf("payload %s does not keep \"goal\": null", row[0])
		}
	}
}

// MVP-BL-04: an excerpt that differs from the source only in whitespace or
// line breaks (CRLF too) verifies; one changed character is not_found; a
// PDF is not_checkable; a deleted file is source_missing. The results are
// marks: the proposal is stored. Expected values are written out.
func TestProposeChecksExcerpts(t *testing.T) {
	home, f, brief := proposalsProject(t)
	repo := f.Repo.Dir
	if err := os.WriteFile(filepath.Join(repo, "docs", "crlf.md"), []byte("## R9\r\n\r\n导出的 CSV 包含\r\n订单编号和金额。\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "spec.pdf"), []byte("%PDF-1.7\n订单编号\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "gone.md"), []byte("要删除的需求"), 0o600); err != nil {
		t.Fatal(err)
	}
	crlf := sourceAdd(t, repo, "docs/crlf.md").Source.ID
	pdf := sourceAdd(t, repo, "docs/spec.pdf").Source.ID
	gone := sourceAdd(t, repo, "docs/gone.md").Source.ID
	if err := os.Remove(filepath.Join(repo, "docs", "gone.md")); err != nil {
		t.Fatal(err)
	}
	refs := []any{
		map[string]any{"source": "docs/crlf.md", "excerpt": "导出的 CSV 包含订单编号和金额。"},
		map[string]any{"source": crlf, "locator": "R9", "excerpt": "导出的CSV包含\n订单编号和金额。"},
		map[string]any{"source": "docs/crlf.md", "excerpt": "导出的 CSV 包含订单编号和数额。"},
		map[string]any{"source": "docs/spec.pdf", "excerpt": "订单编号"},
		map[string]any{"source": "docs/gone.md", "excerpt": "要删除的需求"},
		map[string]any{"source": "docs/client-brief.md", "excerpt": "首期只覆盖下单和查询，\r\n不包括支付和物流对接。"},
	}
	doc := map[string]any{"schema": "specmetry.proposal/v1", "origin": "skill_goals", "base_version": 0, "summary": "excerpts",
		"items": []any{map[string]any{"op": "add", "type": "goal", "ref": "g1", "data": map[string]any{"title": "export"},
			"origin": "extracted", "sources": refs}},
		"questions": []any{map[string]any{"text": "which columns?", "blocking": false, "sources": refs[2:4]}}}
	r := proposeOK(t, repo, doc)
	want := []struct{ source, verification string }{
		{crlf, "verified"}, {crlf, "verified"}, {crlf, "not_found"}, {pdf, "not_checkable"}, {gone, "source_missing"}, {brief, "verified"},
	}
	got := r.Items[0].Sources
	if len(got) != len(want) {
		t.Fatalf("%d sources", len(got))
	}
	for i, w := range want {
		if got[i].SourceID != w.source || got[i].Verification != w.verification {
			t.Errorf("reference %d: %+v, want %+v", i, got[i], w)
		}
	}
	var stored []map[string]any
	if err := json.Unmarshal([]byte(rows(t, home, `SELECT sources_json FROM proposal_items`)[0][0]), &stored); err != nil {
		t.Fatal(err)
	}
	for i, w := range want {
		if stored[i]["verification"] != w.verification || stored[i]["source_id"] != w.source {
			t.Errorf("stored reference %d: %v", i, stored[i])
		}
	}
	if got := rows(t, home, `SELECT verification, source_sha256 IS NULL FROM source_refs ORDER BY rowid`); len(got) != 2 ||
		got[0][0] != "not_found" || got[1][0] != "not_checkable" || got[0][1] != "0" {
		t.Fatalf("question references %v", got)
	}
}

// MVP-BL-05 (M1): code references are checked against the HEAD commit of
// the worktree: a committed file, with its line range inside the file,
// exists; a range past its end, a missing or uncommitted file and a
// directory do not. They are marks, never evidence. Without code
// references an inferred item is refused (TestProposeRefuses).
func TestProposeChecksCodeRefs(t *testing.T) {
	home, f, _ := proposalsProject(t)
	f.Repo.WriteFile("internal/auth/login.go", "package auth\n\nfunc Login() {}\n")
	f.Repo.WriteFile("internal/auth/empty.go", "")
	head := f.Repo.Commit("add login")
	f.Repo.WriteFile("internal/auth/later.go", "package auth\n")
	codeRefs := []any{
		map[string]any{"path": "internal/auth/login.go", "lines": "1-3"},
		map[string]any{"path": "internal/auth/login.go", "lines": "3"},
		map[string]any{"path": "internal/auth/login.go"},
		map[string]any{"path": "internal/auth/login.go", "lines": "2-4"},
		map[string]any{"path": "internal/auth/missing.go"},
		map[string]any{"path": "internal/auth/later.go"},
		map[string]any{"path": "internal/auth"},
		// Valid at the 32-bit limit, but past the end of the file.
		map[string]any{"path": "internal/auth/login.go", "lines": "2147483647"},
		map[string]any{"path": "internal/auth/login.go", "lines": "1-2147483647"},
		// An empty file is there, with no line in it.
		map[string]any{"path": "internal/auth/empty.go"},
		map[string]any{"path": "internal/auth/empty.go", "lines": "1"},
	}
	doc := map[string]any{"schema": "specmetry.proposal/v1", "origin": "skill_init", "base_version": 0, "summary": "inferred",
		"items": []any{map[string]any{"op": "add", "type": "requirement", "ref": "r1", "origin": "inferred_from_code",
			"rationale": "the login handler locks accounts", "code_refs": codeRefs,
			"data": map[string]any{"title": "log in", "kind": "functional", "priority": "must"}}}}
	r := proposeOK(t, f.Repo.Dir, doc)
	want := []bool{true, true, true, false, false, false, false, false, false, true, false}
	if len(r.Items[0].CodeRefs) != len(want) {
		t.Fatalf("%d code references", len(r.Items[0].CodeRefs))
	}
	for i, w := range want {
		if c := r.Items[0].CodeRefs[i]; c.Exists != w {
			t.Errorf("code reference %d: %+v, want exists %v", i, c, w)
		}
	}
	var stored []map[string]any
	if err := json.Unmarshal([]byte(rows(t, home, `SELECT code_refs_json FROM proposal_items`)[0][0]), &stored); err != nil {
		t.Fatal(err)
	}
	for i, c := range stored {
		if c["checked_head"] != head || c["exists"] != want[i] {
			t.Errorf("stored code reference %d: %v, want head %s", i, c, head)
		}
	}
	if got := rows(t, home, `SELECT origin FROM proposal_items`); got[0][0] != "inferred_from_code" {
		t.Fatalf("origin %v", got)
	}
}

// codex-review-020 B1: in a partial clone, reading the objects behind code
// references would make git fetch what it lacks from the promisor remote
// (a stand-in upload-pack here, which only leaves a marker). propose
// refuses (exit 6) before reading any object: the stand-in never runs,
// the repository is left as it was and nothing is written. A proposal
// without code references reads no object and is stored.
func TestProposeCodeRefsInAPartialClone(t *testing.T) {
	home, f, _ := proposalsProject(t)
	repo := f.Repo
	repo.WriteFile("internal/auth/login.go", "package auth\n\nfunc Login() {}\n")
	head := repo.Commit("add login")
	blob := repo.Git("rev-parse", head+":internal/auth/login.go")
	dir := t.TempDir()
	ran := filepath.Join(dir, "upload-pack-ran")
	script := filepath.Join(dir, "upload-pack.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ran >> '"+filepath.ToSlash(ran)+"'\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(script, " '") {
		t.Skipf("temporary path %q needs quoting", script)
	}
	repo.Git("config", "core.repositoryformatversion", "1")
	repo.Git("config", "extensions.partialClone", "origin")
	repo.Git("config", "remote.origin.url", filepath.ToSlash(t.TempDir()))
	repo.Git("config", "remote.origin.promisor", "true")
	repo.Git("config", "remote.origin.uploadpack", "sh "+filepath.ToSlash(script))
	if err := os.Remove(filepath.Join(repo.Dir, ".git", "objects", blob[:2], blob[2:])); err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{"schema": "specmetry.proposal/v1", "origin": "skill_init", "base_version": 0, "summary": "inferred",
		"items": []any{map[string]any{"op": "add", "type": "requirement", "ref": "r1", "origin": "inferred_from_code",
			"rationale": "the login handler locks accounts",
			"code_refs": []any{map[string]any{"path": "internal/auth/login.go", "lines": "1-3"}},
			"data":      map[string]any{"title": "log in", "kind": "functional", "priority": "must"}}}}
	gitDir := filepath.Join(repo.Dir, ".git")
	before := testutil.SettledSnapshot(t, gitDir)
	env := proposeRefused(t, home, repo.Dir, doc, apperr.ExitEnvironment, "")
	if env.Error.Code != "environment" || !strings.Contains(env.Error.Message, "partial clone") {
		t.Errorf("error %+v", env.Error)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Fatal("the promisor remote's upload-pack ran")
	}
	testutil.AssertTreeUnchanged(t, gitDir, before)
	proposeOK(t, repo.Dir, recorded(t, f))
	if _, err := os.Stat(ran); err == nil {
		t.Fatal("the promisor remote's upload-pack ran")
	}
}

// codex-review-021 B1, the combined case: a partial clone known only from
// remote.origin.promisor = 2 (no extensions.partialClone), a promisor URL
// for the remote helper named as the earlier allow list was, that helper
// on PATH (a stand-in that only leaves a marker), and a blob removed.
// propose refuses (exit 6) before reading an object: the helper never
// runs, and the git directory and the database stay as they were.
func TestProposeRefusesPartialCloneByAnyMeans(t *testing.T) {
	home, f, _ := proposalsProject(t)
	repo := f.Repo
	repo.WriteFile("internal/auth/login.go", "package auth\n\nfunc Login() {}\n")
	head := repo.Commit("add login")
	blob := repo.Git("rev-parse", head+":internal/auth/login.go")
	pathDir := t.TempDir()
	ran := testutil.FakeRemoteHelper(t, pathDir, "specmetry-no-transport")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	repo.Git("config", "remote.origin.url", "specmetry-no-transport::unused")
	repo.Git("config", "remote.origin.promisor", "2")
	if err := os.Remove(filepath.Join(repo.Dir, ".git", "objects", blob[:2], blob[2:])); err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{"schema": "specmetry.proposal/v1", "origin": "skill_init", "base_version": 0, "summary": "inferred",
		"items": []any{map[string]any{"op": "add", "type": "requirement", "ref": "r1", "origin": "inferred_from_code",
			"rationale": "the login handler locks accounts",
			"code_refs": []any{map[string]any{"path": "internal/auth/login.go", "lines": "1-3"}},
			"data":      map[string]any{"title": "log in", "kind": "functional", "priority": "must"}}}}
	gitDir := filepath.Join(repo.Dir, ".git")
	before := testutil.SettledSnapshot(t, gitDir)
	env := proposeRefused(t, home, repo.Dir, doc, apperr.ExitEnvironment, "")
	if env.Error.Code != "environment" || !strings.Contains(env.Error.Message, "partial clone") {
		t.Errorf("error %+v", env.Error)
	}
	if b, _ := json.Marshal(env); bytes.Contains(b, []byte("specmetry-no-transport")) || bytes.Contains(b, []byte(filepath.ToSlash(repo.Dir))) {
		t.Errorf("the error names the configuration or the repository: %s", b)
	}
	// The text error as well.
	code, stdout, stderr := specmetry(t, repo.Dir, "", "propose", "--file", writeProposal(t, doc))
	if code != apperr.ExitEnvironment || strings.Contains(stdout+stderr, "specmetry-no-transport") ||
		strings.Contains(filepath.ToSlash(stdout+stderr), filepath.ToSlash(repo.Dir)) {
		t.Errorf("text error: exit %d\n%s%s", code, stdout, stderr)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Fatal("the remote helper ran")
	}
	testutil.AssertTreeUnchanged(t, gitDir, before)
}

// MVP-BL-21: a check with its rationale, an environment variable for its
// secret and paths inside the repository is accepted and kept as proposed.
func TestProposeAcceptsAChecked(t *testing.T) {
	home, f, _ := proposalsProject(t)
	doc := patched(t, recorded(t, f), "/items/2/data/check", map[string]any{
		"command": "curl -H \"Authorization: Bearer $API_TOKEN\" localhost && go test ./internal/auth -run TestLock",
		"workdir": "./internal", "inputs": []any{"internal/auth/lock_test.go"}, "timeout_sec": 120,
		"rationale": "TestLock fails five logins and checks the sixth is refused"})
	proposeOK(t, f.Repo.Dir, doc)
	var data map[string]any
	if err := json.Unmarshal([]byte(rows(t, home, `SELECT payload_json FROM proposal_items WHERE ref = 'c1'`)[0][0]), &data); err != nil {
		t.Fatal(err)
	}
	if c, _ := data["check"].(map[string]any); c["timeout_sec"] != float64(120) || !strings.Contains(c["command"].(string), "$API_TOKEN") {
		t.Fatalf("payload %v", data)
	}
}

// MVP-PRV-08, BL-28: under hash_only an excerpt is only checked in memory:
// its text reaches neither the data directory (database, WAL, logs) nor
// the output nor an error, whether it verified or not.
func TestProposeNeverStoresExcerpts(t *testing.T) {
	home, f, _ := proposalsProject(t)
	canary := testutil.NewCanary(t)
	repo := f.Repo.Dir
	if err := os.WriteFile(filepath.Join(repo, "docs", "private.md"), []byte("# 私密\n\n客户口令规则："+canary+"，不得外传。\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sourceAdd(t, repo, "docs/private.md")
	refs := []any{
		map[string]any{"source": "docs/private.md", "locator": "私密", "excerpt": "客户口令规则：" + canary},
		map[string]any{"source": "docs/private.md", "excerpt": canary + "，可以外传。"},
	}
	doc := map[string]any{"schema": "specmetry.proposal/v1", "origin": "skill_init", "base_version": 0, "summary": "private",
		"items":     []any{map[string]any{"op": "add", "type": "goal", "ref": "g1", "data": map[string]any{"title": "rules"}, "origin": "extracted", "sources": refs}},
		"questions": []any{map[string]any{"text": "may it be shared?", "blocking": true, "sources": refs}}}
	var outputs []string
	for _, asJSON := range []bool{true, false} {
		args := []string{"propose", "--file", writeProposal(t, doc)}
		schema := ""
		if asJSON {
			args, schema = append(args, "--json"), "proposal-receipt.v1"
		}
		code, stdout, stderr := specmetry(t, repo, schema, args...)
		if code != apperr.ExitOK {
			t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
		}
		outputs = append(outputs, stdout, stderr)
	}
	// Refused proposals do not echo it either.
	for _, bad := range []any{
		patched(t, doc, "/items/0/sources/1/excerpt", strings.Repeat(canary, 30)),
		patched(t, doc, "/items/0/sources/1/source", "docs/unknown.md"),
		patched(t, doc, "/items/0/data/title", "a\nb"),
	} {
		code, stdout, stderr := specmetry(t, repo, "", "propose", "--file", writeProposal(t, bad), "--json")
		if code != apperr.ExitInvalidInput {
			t.Errorf("refused proposal: exit %d", code)
		}
		outputs = append(outputs, stdout, stderr)
	}
	for _, out := range outputs {
		if strings.Contains(out, canary) {
			t.Fatalf("the output holds the excerpt:\n%s", out)
		}
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the excerpt reached the data directory: %v", hits)
	}
	if n := count(t, home, `SELECT count(*) FROM source_refs WHERE excerpt IS NOT NULL`); n != 0 {
		t.Fatalf("%d stored excerpts", n)
	}
}

// codex-review-020 B2: the database refuses excerpt text in a stored
// proposal item under hash_only, through any write: a final payload
// holding an excerpt, inserted or updated in an ordinary write transaction,
// is refused and rolled back, and the text reaches neither the database
// nor its WAL. A review result without excerpt text is still written.
func TestHashOnlyDefendsStoredItems(t *testing.T) {
	home, f, _ := proposalsProject(t)
	canary := testutil.NewCanary(t)
	if err := os.WriteFile(filepath.Join(f.Repo.Dir, "docs", "private.md"), []byte("规则："+canary+"。\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	private := sourceAdd(t, f.Repo.Dir, "docs/private.md").Source.ID
	excerpt := "规则：" + canary
	doc := map[string]any{"schema": "specmetry.proposal/v1", "origin": "skill_init", "base_version": 0, "summary": "private",
		"items": []any{map[string]any{"op": "add", "type": "goal", "ref": "g1", "data": map[string]any{"title": "rules"},
			"origin": "extracted", "sources": []any{map[string]any{"source": "docs/private.md", "excerpt": excerpt}}}}}
	if r := proposeOK(t, f.Repo.Dir, doc); r.Items[0].Sources[0].Verification != "verified" {
		t.Fatalf("receipt %+v", r.Items[0])
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the submission stored the excerpt: %v", hits)
	}

	db, _, err := sqlite.Open(context.Background(), home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var proposalID, project string
	if err := db.WriteTx(context.Background(), func(tx *sql.Tx) error {
		return tx.QueryRow(`SELECT proposal_id, project_id FROM proposal_items`).Scan(&proposalID, &project)
	}); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{"title": "rules",
		"sources": []any{map[string]any{"source_id": private, "excerpt_sha256": digest(excerpt), "excerpt": excerpt}}})
	if err != nil {
		t.Fatal(err)
	}
	before := written(t, home)
	for name, query := range map[string]string{
		"an update of the pending item": `UPDATE proposal_items SET final_payload_json = ?1`,
		"a review with the excerpt": `UPDATE proposal_items SET final_payload_json = ?1, review_state = 'accepted_with_edits',
    change_class = 'material', reviewed_by = 'human:me', reviewed_channel = 'human_cli', reviewed_seq = 999`,
		"an inserted item": `INSERT INTO proposal_items (id, proposal_id, project_id, item_index, op, entity_type, ref, payload_json,
    sources_json, code_refs_json, final_payload_json)
VALUES ('01J9Z3K5Q8R2T4V6W8X0Y2Z4J8', ?2, ?3, 1, 'add', 'goal', 'g2', '{}', '[]', '[]', ?1)`,
	} {
		err := db.WriteTx(context.Background(), func(tx *sql.Tx) error {
			_, err := tx.Exec(query, string(payload), proposalID, project)
			return err
		})
		if err == nil || !strings.Contains(err.Error(), "excerpt text is not stored under hash_only") {
			t.Errorf("%s: %v", name, err)
		}
	}
	if after := written(t, home); after != before {
		t.Fatalf("written %s -> %s", before, after)
	}
	_ = db.Close()
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the excerpt reached the data directory: %v", hits)
	}

	// A review result without excerpt text is written.
	db, _, err = sqlite.Open(context.Background(), home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	clean, _ := json.Marshal(map[string]any{"title": "rules",
		"sources": []any{map[string]any{"source_id": private, "excerpt_sha256": digest(excerpt), "excerpt": nil}}})
	if err := db.WriteTx(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec(`UPDATE proposal_items SET final_payload_json = ?, review_state = 'accepted_with_edits',
    change_class = 'material', reviewed_by = 'human:me', reviewed_channel = 'human_cli', reviewed_seq = 999`, string(clean))
		return err
	}); err != nil {
		t.Fatalf("a review without excerpt text: %v", err)
	}
	_ = db.Close()
	if got := rows(t, home, `SELECT review_state, final_payload_json FROM proposal_items`); got[0][0] != "accepted_with_edits" ||
		strings.Contains(got[0][1], canary) {
		t.Fatalf("review %v", got)
	}
}

// Concurrent proposals each get their own seq, with no gap, and question
// keys are handed out once each.
func TestProposeConcurrently(t *testing.T) {
	home, f, _ := proposalsProject(t)
	before, err := strconv.Atoi(seq(t, home))
	if err != nil {
		t.Fatal(err)
	}
	const n = 6
	t.Chdir(f.Repo.Dir)
	paths := make([]string, n)
	for i := range n {
		paths[i] = writeProposal(t, map[string]any{"schema": "specmetry.proposal/v1", "origin": "cli", "base_version": 0,
			"summary": "proposal " + strconv.Itoa(i), "items": []any{},
			"questions": []any{map[string]any{"text": "question " + strconv.Itoa(i), "blocking": false}}})
	}
	codes, outs, errs := make([]int, n), make([]string, n), make([]string, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			var stdout, stderr bytes.Buffer
			codes[i] = Execute(context.Background(), []string{"propose", "--file", paths[i], "--json"}, &stdout, &stderr)
			outs[i], errs[i] = stdout.String(), stderr.String()
		})
	}
	wg.Wait()
	var seqs []int
	for i := range n {
		if codes[i] != apperr.ExitOK {
			t.Fatalf("proposal %d: exit %d\n%s%s", i, codes[i], outs[i], errs[i])
		}
		var r receiptOut
		if err := json.Unmarshal([]byte(outs[i]), &r); err != nil {
			t.Fatalf("proposal %d: %v\n%s", i, err, outs[i])
		}
		seqs = append(seqs, int(r.Seq))
	}
	sort.Ints(seqs)
	for i, s := range seqs {
		if s != before+1+i {
			t.Fatalf("seqs %v after %d", seqs, before)
		}
	}
	keys := rows(t, home, `SELECT key FROM questions ORDER BY key`)
	for i, k := range keys {
		if k[0] != "Q-00"+strconv.Itoa(i+1) {
			t.Fatalf("question keys %v", keys)
		}
	}
	if len(keys) != n {
		t.Fatalf("%d questions", len(keys))
	}
}

// A failure inside the write transaction (simulated with a trigger) leaves
// nothing behind: no proposal, item, question, reference, key or seq.
func TestProposeRollsBack(t *testing.T) {
	home, f, _ := proposalsProject(t)
	execSQL := func(query string) {
		t.Helper()
		db, _, err := sqlite.Open(context.Background(), home, sqlite.Options{})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = db.Close() }()
		if err := db.WriteTx(context.Background(), func(tx *sql.Tx) error { _, err := tx.Exec(query); return err }); err != nil {
			t.Fatal(err)
		}
	}
	execSQL(`CREATE TRIGGER test_fail_question BEFORE INSERT ON source_refs BEGIN SELECT RAISE(ABORT, 'simulated failure'); END`)
	before := written(t, home)
	code, stdout, stderr := specmetry(t, f.Repo.Dir, "", "propose", "--file", writeProposal(t, recorded(t, f)), "--json")
	if code == apperr.ExitOK {
		t.Fatalf("the proposal was stored\n%s%s", stdout, stderr)
	}
	if after := written(t, home); after != before {
		t.Fatalf("written %s -> %s", before, after)
	}
	execSQL(`DROP TRIGGER test_fail_question`)
	proposeOK(t, f.Repo.Dir, recorded(t, f))
	if got := rows(t, home, `SELECT key FROM questions ORDER BY key`); len(got) != 4 || got[0][0] != "Q-001" {
		t.Fatalf("questions after the rollback: %v", got)
	}
}

// A proposal from a linked worktree not seen before registers it (data-model
// §4.1) and names it; the next one reuses it.
func TestProposeRegistersTheWorktree(t *testing.T) {
	home, f, _ := proposalsProject(t)
	wt := f.Repo.AddWorktree("feature-x")
	small := map[string]any{"schema": "specmetry.proposal/v1", "origin": "cli", "base_version": 0, "summary": "from a worktree", "items": []any{}}
	before := count(t, home, `SELECT count(*) FROM worktrees`)
	proposeOK(t, wt, small)
	proposeOK(t, wt, small)
	if n := count(t, home, `SELECT count(*) FROM worktrees`); n != before+1 {
		t.Fatalf("%d worktrees, want %d", n, before+1)
	}
	got := rows(t, home, `SELECT DISTINCT w.is_main FROM proposals p JOIN worktrees w ON w.id = p.worktree_id`)
	if len(got) != 1 || got[0][0] != "0" {
		t.Fatalf("proposals name %v", got)
	}
}

// propose --file - reads the proposal from standard input.
func TestProposeFromStdin(t *testing.T) {
	home, f, _ := proposalsProject(t)
	b, err := json.Marshal(recorded(t, f))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal([]string{"propose", "--file", "-", "--json"})
	cmd := testutil.HelperCommand(t, "cli")
	cmd.Env = helperEnv(t, "cli", argsJSONEnv+"="+string(raw), dirEnv+"="+f.Repo.Dir)
	cmd.Stdin = bytes.NewReader(b)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run()
	if !strings.Contains(stderr.String(), "exit=0") {
		t.Fatalf("propose --file -:\n%s%s", stdout.String(), stderr.String())
	}
	validator, err := schemas()
	if err != nil {
		t.Fatal(err)
	}
	if err := validator.ValidateJSON("proposal-receipt.v1", stdout.Bytes()); err != nil {
		t.Fatal(err)
	}
	if n := count(t, home, `SELECT count(*) FROM proposal_items`); n != 18 {
		t.Fatalf("%d items", n)
	}
}

type refOut struct {
	EntityType     string  `json:"entity_type"`
	EntityKey      string  `json:"entity_key"`
	EntityRevision int     `json:"entity_revision"`
	Current        bool    `json:"current"`
	Locator        *string `json:"locator"`
	Verification   string  `json:"verification"`
	ExcerptSHA256  string  `json:"excerpt_sha256"`
	ExcerptState   string  `json:"excerpt_state"`
	Excerpt        *string `json:"excerpt"`
}

// MVP-SRC-04 and SRC-08 (M1, the parts waiting for references): source
// list --refs lists the references proposals wrote, with their locator and
// excerpt hash and excerpt_state not_stored under hash_only; removing the
// source (with --erase-excerpts) keeps them, listed under the removed
// source; a proposal can no longer cite it.
func TestSourceListRefs(t *testing.T) {
	home, f, _ := proposalsProject(t)
	repo := f.Repo.Dir
	proposeOK(t, repo, recorded(t, f))
	want := []struct{ key, locator, excerpt string }{
		{"Q-001", "R3", "每页显示合适数量的订单"}, {"Q-002", "R7", "系统应尽快通知创建订单的店员"},
		{"Q-003", "待确认", "区域经理是否也可以创建订单，目前还没有结论。"}, {"Q-004", "R8", "店员无需培训即可上手"},
	}
	checkRefs := func(refs []json.RawMessage) {
		t.Helper()
		if len(refs) != len(want) {
			t.Fatalf("%d references, want %d", len(refs), len(want))
		}
		for i, raw := range refs {
			var r refOut
			if err := json.Unmarshal(raw, &r); err != nil {
				t.Fatal(err)
			}
			w := want[i]
			if r.EntityType != "question" || r.EntityKey != w.key || r.EntityRevision != 1 || !r.Current || r.Locator == nil ||
				*r.Locator != w.locator || r.Verification != "verified" || r.ExcerptSHA256 != digest(w.excerpt) ||
				r.ExcerptState != "not_stored" || r.Excerpt != nil {
				t.Errorf("reference %d: %s", i, raw)
			}
		}
	}
	list := sourceList(t, repo, "--refs")
	if len(list.Sources) != 1 || list.Sources[0].Refs == nil {
		t.Fatalf("sources %+v", list.Sources)
	}
	checkRefs(*list.Sources[0].Refs)
	if list := sourceList(t, repo); list.Sources[0].Refs != nil {
		t.Fatal("references listed without --refs")
	}

	out := inTerminal(t, repo, []string{"source", "remove", "docs/client-brief.md", "--erase-excerpts"}, "docs/client-brief.md")
	if !strings.Contains(out, "exit=0") {
		t.Fatalf("source remove:\n%s", out)
	}
	removed := sourceList(t, repo, "--include-removed", "--refs")
	if len(removed.Sources) != 1 || removed.Sources[0].RemovedSeq == nil {
		t.Fatalf("sources %+v", removed.Sources)
	}
	checkRefs(*removed.Sources[0].Refs)
	if n := count(t, home, `SELECT count(*) FROM source_refs WHERE excerpt IS NULL AND excerpt_erased_seq IS NULL`); n != 4 {
		t.Fatalf("%d references kept as they were", n)
	}
	proposeRefused(t, home, repo, recorded(t, f), apperr.ExitInvalidInput, "/items/0/sources/0/source")
}
