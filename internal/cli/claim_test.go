package cli

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/claim"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// claimOut is the claim-receipt.v1 document.
type claimOut struct {
	BatchID     string `json:"batch_id"`
	Seq         int64  `json:"seq"`
	Kind        string `json:"kind"`
	BaseVersion int    `json:"base_version"`
	Context     struct {
		WorktreeID string  `json:"worktree_id"`
		Head       *string `json:"head"`
		TreeState  string  `json:"tree_state"`
	} `json:"context"`
	CompareFrom *struct {
		Commit *string `json:"commit"`
		Source string  `json:"source"`
	} `json:"compare_from"`
	Claims []struct {
		Index          int     `json:"index"`
		ClaimID        string  `json:"claim_id"`
		Target         string  `json:"target"`
		TargetRevision int     `json:"target_revision"`
		Assertion      string  `json:"assertion"`
		Refs           refsOut `json:"refs"`
	} `json:"claims"`
	UnmappedWork []struct {
		Index   int     `json:"index"`
		ClaimID string  `json:"claim_id"`
		Refs    refsOut `json:"refs"`
	} `json:"unmapped_work"`
}

type refsOut struct {
	Files []struct {
		Path   string  `json:"path"`
		Lines  *string `json:"lines"`
		Exists bool    `json:"exists"`
	} `json:"files"`
	Commits []struct {
		Commit   string `json:"commit"`
		Resolved bool   `json:"resolved"`
	} `json:"commits"`
	Tests int `json:"tests"`
}

// marks are a receipt's reference results, as "exists" or "missing" and
// "resolved" or "unknown_commit", and the test count.
func (r refsOut) marks() string {
	var out []string
	for _, f := range r.Files {
		out = append(out, map[bool]string{true: "exists", false: "missing"}[f.Exists])
	}
	for _, c := range r.Commits {
		out = append(out, map[bool]string{true: "resolved", false: "unknown_commit"}[c.Resolved])
	}
	return strings.Join(append(out, "tests="+strconv.Itoa(r.Tests)), " ")
}

func addAuto(ref, requirement, statement string) map[string]any {
	return map[string]any{"op": "add", "type": "criterion", "ref": ref, "origin": "manual",
		"data": map[string]any{"requirement": requirement, "statement": statement, "verification": "automated",
			"check": map[string]any{"command": "go test ./...", "rationale": "the tests export every order"}}}
}

// claimsProject is fx-docs-only, initialized, with baseline version 1:
// REQ-001 with the manual criteria AC-001-1 and AC-001-2, and REQ-002 with
// the automated AC-002-1 (with a check). Its worktree is clean.
func claimsProject(t *testing.T) (datadir.Dir, *testutil.Fixture, string) {
	t.Helper()
	return claimsProjectOf(t, "fx-docs-only")
}

// claimsProjectOf is claimsProject from another fixture with the same
// client brief.
func claimsProjectOf(t *testing.T, fixture string) (datadir.Dir, *testutil.Fixture, string) {
	t.Helper()
	noAgentVariables(t)
	home := testutil.TempHome(t)
	f := testutil.BuildFixture(t, fixture)
	dir := f.Repo.Dir
	mustInit(t, dir)
	sourceAdd(t, dir, "docs/client-brief.md")
	acceptAll(t, dir, proposeOK(t, dir, small(0,
		addReq("r1", "sign up", nil), addAC("c1", "r1", "a new account can sign in"), addAC("c2", "r1", "a locked account cannot"),
		addReq("r2", "export", nil), addAuto("c3", "r2", "the export holds every order"))), 0, nil)
	if keys := baselineShow(t, dir).keys(); keys != "REQ-001 AC-001-1 AC-001-2 REQ-002 AC-002-1" {
		t.Fatalf("baseline = %s", keys)
	}
	if status := f.Repo.Git("status", "--porcelain", "--untracked-files=all", "--", ".", ":(exclude).specmetry"); status != "" {
		t.Fatalf("the worktree is not clean:\n%s", status)
	}
	return home, f, dir
}

// claimsProjectOutsideGit is claimsProject outside git: the same client
// brief, registered, and the same baseline version 1. For the tests whose
// subject does not depend on git (the input contract, the baseline under
// the write lock, a person's confirmation, the output), it saves the cost
// of a git fixture and of the worktree state read on every claim.
func claimsProjectOutsideGit(t *testing.T) (datadir.Dir, string) {
	t.Helper()
	noAgentVariables(t)
	home := testutil.TempHome(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "client-brief.md"), []byte("# brief\n\nsign up\nexport\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustInit(t, root, "--name", "plain")
	sourceAdd(t, root, "docs/client-brief.md")
	acceptAll(t, root, proposeOK(t, root, small(0,
		addReq("r1", "sign up", nil), addAC("c1", "r1", "a new account can sign in"), addAC("c2", "r1", "a locked account cannot"),
		addReq("r2", "export", nil), addAuto("c3", "r2", "the export holds every order"))), 0, nil)
	if keys := baselineShow(t, root).keys(); keys != "REQ-001 AC-001-1 AC-001-2 REQ-002 AC-002-1" {
		t.Fatalf("baseline = %s", keys)
	}
	return home, root
}

// claimDoc is a claims.v1 document; an audit batch declares all
// requirements unless the caller sets audit_scope.
func claimDoc(kind string, base int, claims ...any) map[string]any {
	doc := map[string]any{"schema": "specmetry.claims/v1", "kind": kind, "base_version": base, "summary": "canary-batch-summary", "claims": claims}
	if claims == nil {
		doc["claims"] = []any{}
	}
	if kind == claim.Audit {
		doc["audit_scope"] = map[string]any{"all": true}
	}
	return doc
}

func claimOn(target string, revision int, assertion string, refs map[string]any) map[string]any {
	c := map[string]any{"target": target, "target_revision": revision, "assertion": assertion, "summary": "canary-claim-summary " + target}
	if refs != nil {
		c["refs"] = refs
	}
	return c
}

func fileRef(path, lines string) map[string]any {
	if lines == "" {
		return map[string]any{"path": path}
	}
	return map[string]any{"path": path, "lines": lines}
}

func writeClaims(t *testing.T, doc any) string {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "claims.json")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// echoed lists what a claim's output must never hold (WP2.4 spec §9.1,
// §10): the summaries and test names of doc.
func echoed(doc map[string]any) []string {
	out := []string{doc["summary"].(string)}
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			if s, ok := v["summary"].(string); ok {
				out = append(out, s)
			}
			if refs, ok := v["refs"].(map[string]any); ok {
				if tests, ok := refs["tests"].([]any); ok {
					for _, name := range tests {
						out = append(out, name.(string))
					}
				}
			}
		case []any:
			for _, e := range v {
				walk(e)
			}
		}
	}
	walk(doc["claims"])
	walk(doc["unmapped_work"])
	return out
}

func noneEchoed(t *testing.T, what string, doc map[string]any, outputs ...string) {
	t.Helper()
	for _, s := range echoed(doc) {
		for _, out := range outputs {
			if strings.Contains(out, s) {
				t.Errorf("%s: the output echoes %q:\n%s", what, s, out)
			}
		}
	}
}

// claimOK submits doc with --json in dir and checks the receipt against
// the input and the stored rows (claimStored).
func claimOK(t *testing.T, home datadir.Dir, dir string, doc map[string]any) claimOut {
	t.Helper()
	code, stdout, stderr := specmetry(t, dir, "claim-receipt.v1", "claim", "--file", writeClaims(t, doc), "--json")
	if code != apperr.ExitOK {
		t.Fatalf("claim: exit %d\n%s%s", code, stdout, stderr)
	}
	var r claimOut
	if err := json.Unmarshal([]byte(wholeJSON(t, stdout)), &r); err != nil {
		t.Fatal(err)
	}
	noneEchoed(t, "claim", doc, stdout, stderr)
	claimStored(t, home, doc, r)
	return r
}

// storedRefs is the refs_json a claim's references are stored as: the
// files with their line ranges, the test names and the commit prefixes,
// each list there even when empty.
func storedRefs(refs any) any {
	files, tests, commits := []any{}, []any{}, []any{}
	if m, ok := refs.(map[string]any); ok {
		if v, ok := m["files"].([]any); ok {
			files = v
		}
		if v, ok := m["tests"].([]any); ok {
			tests = v
		}
		if v, ok := m["commits"].([]any); ok {
			commits = v
		}
	}
	return map[string]any{"files": files, "tests": tests, "commits": commits}
}

func decoded(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("%v: %s", err, s)
	}
	return v
}

func asJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// claimStored checks the business conditions of WP2.4 spec §9.2 that the
// schema cannot express: every input claim and unmapped work item is in
// the receipt, at its index, in input order, with its target, revision and
// assertion; the receipt's batch, seq, kind, base version, context and
// comparison start are the stored batch's; the batch and every claim share
// the seq; each claim ID is a stored row, distinct, whose criterion,
// revision, assertion, summary and references are the input's; the
// reference results are refs_check_json's, whose checked_head is the
// captured head, and tests counts the stored test names; M2 never compares
// from a session.
func claimStored(t *testing.T, home datadir.Dir, doc map[string]any, r claimOut) {
	t.Helper()
	in := doc["claims"].([]any)
	var unmapped []any
	if u, ok := doc["unmapped_work"].([]any); ok {
		unmapped = u
	}
	if len(r.Claims) != len(in) || len(r.UnmappedWork) != len(unmapped) {
		t.Fatalf("receipt has %d claims and %d unmapped items, the input %d and %d", len(r.Claims), len(r.UnmappedWork), len(in), len(unmapped))
	}
	batch := rows(t, home, `SELECT seq, kind, base_version, worktree_id, coalesce(head, '-'), tree_state, coalesce(compare_from, '-'),
    coalesce(compare_from_source, '-'), summary, coalesce(audit_scope_json, '-'), coalesce(session_id, '-'), actor, channel
FROM claim_batches WHERE id = ?`, r.BatchID)
	if len(batch) != 1 {
		t.Fatalf("batch %s: %v", r.BatchID, batch)
	}
	head, compareFrom, source := "-", "-", "-"
	if r.Context.Head != nil {
		head = *r.Context.Head
	}
	if r.CompareFrom != nil {
		source = r.CompareFrom.Source
		if r.CompareFrom.Commit != nil {
			compareFrom = *r.CompareFrom.Commit
		}
		if source == "session_start" {
			t.Errorf("M2 compared from a session: %+v", r.CompareFrom)
		}
	}
	scope := "-"
	if s, ok := doc["audit_scope"]; ok {
		scope = asJSON(t, s)
	}
	b := batch[0]
	want := []string{strconv.FormatInt(r.Seq, 10), r.Kind, strconv.Itoa(r.BaseVersion), r.Context.WorktreeID, head, r.Context.TreeState,
		compareFrom, source, doc["summary"].(string)}
	if !reflect.DeepEqual(b[:9], want) || r.Kind != doc["kind"] || float64(r.BaseVersion) != toFloat(doc["base_version"]) {
		t.Errorf("batch row %v, receipt %v, input kind %v base %v", b[:9], want, doc["kind"], doc["base_version"])
	}
	if scope == "-" && b[9] != "-" || scope != "-" && !reflect.DeepEqual(decoded(t, b[9]), decoded(t, scope)) {
		t.Errorf("audit_scope_json %s, input %s", b[9], scope)
	}
	if b[10] != "-" {
		t.Errorf("session_id %s before M3", b[10])
	}
	stored := rows(t, home, `SELECT c.id, c.seq, c.target_type, coalesce(a.key, '-'), coalesce(c.criterion_revision, '-'), coalesce(c.assertion, '-'),
    c.summary, c.refs_json, c.refs_check_json
FROM claims c LEFT JOIN acceptance_criteria a ON a.id = c.criterion_id WHERE c.batch_id = ?`, r.BatchID)
	byID := map[string][]string{}
	for _, row := range stored {
		byID[row[0]] = row
	}
	if len(byID) != len(in)+len(unmapped) {
		t.Fatalf("%d claim rows for %d claims and %d unmapped items", len(byID), len(in), len(unmapped))
	}
	seen := map[string]bool{}
	check := func(what string, index, i int, id string, input map[string]any, refs refsOut, criterion bool) {
		t.Helper()
		row, ok := byID[id]
		if index != i || !ok || seen[id] {
			t.Errorf("%s %d: index %d, claim %s stored %v, seen %v", what, i, index, id, ok, seen[id])
			return
		}
		seen[id] = true
		targetType, key, revision, assertion := "none", "-", "-", "-"
		if criterion {
			targetType, key = "criterion", input["target"].(string)
			revision, assertion = strconv.Itoa(int(toFloat(input["target_revision"]))), input["assertion"].(string)
		}
		if row[1] != strconv.FormatInt(r.Seq, 10) || row[2] != targetType || row[3] != key || row[4] != revision || row[5] != assertion || row[6] != input["summary"] {
			t.Errorf("%s %d: row %v, input %v", what, i, row[:7], input)
		}
		if !reflect.DeepEqual(decoded(t, row[7]), decoded(t, asJSON(t, storedRefs(input["refs"])))) {
			t.Errorf("%s %d: refs_json %s, input %v", what, i, row[7], input["refs"])
		}
		var checked struct {
			Files []struct {
				Path        string  `json:"path"`
				Lines       *string `json:"lines"`
				Exists      bool    `json:"exists"`
				CheckedHead *string `json:"checked_head"`
			} `json:"files"`
			Commits []struct {
				Commit   string `json:"commit"`
				Resolved bool   `json:"resolved"`
			} `json:"commits"`
		}
		if err := json.Unmarshal([]byte(row[8]), &checked); err != nil || checked.Files == nil || checked.Commits == nil {
			t.Fatalf("%s %d: refs_check_json %s (%v)", what, i, row[8], err)
		}
		var receiptFiles, storedFiles []string
		for _, f := range refs.Files {
			receiptFiles = append(receiptFiles, fmt.Sprintf("%s@%v:%v", f.Path, deref(f.Lines), f.Exists))
		}
		for _, f := range checked.Files {
			storedFiles = append(storedFiles, fmt.Sprintf("%s@%v:%v", f.Path, deref(f.Lines), f.Exists))
			if deref(f.CheckedHead) != deref(r.Context.Head) {
				t.Errorf("%s %d: %s checked against %v, the captured head is %v", what, i, f.Path, deref(f.CheckedHead), deref(r.Context.Head))
			}
		}
		stored := storedRefs(input["refs"]).(map[string]any)
		var inputFiles []string
		for _, f := range stored["files"].([]any) {
			m := f.(map[string]any)
			lines := "null"
			if l, ok := m["lines"].(string); ok {
				lines = l
			}
			inputFiles = append(inputFiles, m["path"].(string)+"@"+lines)
		}
		var receiptPaths []string
		for _, f := range receiptFiles {
			receiptPaths = append(receiptPaths, f[:strings.LastIndex(f, ":")])
		}
		if strings.Join(receiptFiles, " ") != strings.Join(storedFiles, " ") || strings.Join(receiptPaths, " ") != strings.Join(inputFiles, " ") {
			t.Errorf("%s %d: receipt files %v, stored %v, input %v", what, i, receiptFiles, storedFiles, inputFiles)
		}
		var receiptCommits, storedCommits, inputCommits []string
		for _, c := range refs.Commits {
			receiptCommits = append(receiptCommits, fmt.Sprintf("%s:%v", c.Commit, c.Resolved))
		}
		for _, c := range checked.Commits {
			storedCommits = append(storedCommits, fmt.Sprintf("%s:%v", c.Commit, c.Resolved))
		}
		for _, c := range stored["commits"].([]any) {
			inputCommits = append(inputCommits, c.(string))
		}
		if strings.Join(receiptCommits, " ") != strings.Join(storedCommits, " ") || len(receiptCommits) != len(inputCommits) {
			t.Errorf("%s %d: receipt commits %v, stored %v, input %v", what, i, receiptCommits, storedCommits, inputCommits)
		}
		for k, c := range refs.Commits {
			if c.Commit != inputCommits[k] {
				t.Errorf("%s %d: commit %d is %s, the input %s", what, i, k, c.Commit, inputCommits[k])
			}
		}
		if refs.Tests != len(stored["tests"].([]any)) {
			t.Errorf("%s %d: tests %d, the input has %d", what, i, refs.Tests, len(stored["tests"].([]any)))
		}
	}
	for i, c := range r.Claims {
		input := in[i].(map[string]any)
		if c.Target != input["target"] || float64(c.TargetRevision) != toFloat(input["target_revision"]) || c.Assertion != input["assertion"] {
			t.Errorf("claim %d: %s r%d %s, input %v", i, c.Target, c.TargetRevision, c.Assertion, input)
		}
		check("claim", c.Index, i, c.ClaimID, input, c.Refs, true)
	}
	for i, u := range r.UnmappedWork {
		check("unmapped work", u.Index, i, u.ClaimID, unmapped[i].(map[string]any), u.Refs, false)
	}
}

func toFloat(v any) float64 {
	switch v := v.(type) {
	case int:
		return float64(v)
	case float64:
		return v
	}
	return -1
}

func deref(p *string) string {
	if p == nil {
		return "null"
	}
	return *p
}

// factState is every row of every table, and the seq counter: a refused
// claim leaves all of it as it was (WP2.4 spec §5, §15), the first
// registration of a worktree included. One read, in one transaction.
func factState(t *testing.T, home datadir.Dir) string {
	t.Helper()
	ctx := context.Background()
	db, _, err := sqlite.Open(ctx, home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var b strings.Builder
	err = db.WriteTx(ctx, func(tx *sql.Tx) error {
		tables, err := tx.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
		if err != nil {
			return err
		}
		var names []string
		for tables.Next() {
			var n string
			if err := tables.Scan(&n); err != nil {
				return err
			}
			names = append(names, n)
		}
		if err := tables.Close(); err != nil {
			return err
		}
		for _, n := range names {
			rs, err := tx.QueryContext(ctx, `SELECT * FROM "`+n+`" ORDER BY rowid`)
			if err != nil {
				return err
			}
			cols, err := rs.Columns()
			if err != nil {
				return err
			}
			b.WriteString(n + ":\n")
			for rs.Next() {
				vals := make([]any, len(cols))
				ptrs := make([]any, len(cols))
				for i := range vals {
					ptrs[i] = &vals[i]
				}
				if err := rs.Scan(ptrs...); err != nil {
					return err
				}
				fmt.Fprintf(&b, "  %v\n", vals)
			}
			if err := rs.Close(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// claimRefused submits doc with --json and checks the refusal: the exit
// code, one error.v1 with a detail at pointer when one is given, nothing
// of the input echoed, and no row or seq written.
func claimRefused(t *testing.T, home datadir.Dir, dir string, doc map[string]any, code int, pointer string) apperr.Envelope {
	t.Helper()
	before := factState(t, home)
	got, stdout, stderr := specmetry(t, dir, "claim-receipt.v1", "claim", "--file", writeClaims(t, doc), "--json")
	if got != code {
		t.Errorf("exit %d, want %d (%s)\n%s%s", got, code, pointer, stdout, stderr)
	}
	var env apperr.Envelope
	if err := json.Unmarshal([]byte(wholeJSON(t, stdout)), &env); err != nil {
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
	if stderr != "" {
		t.Errorf("%s: stderr with --json:\n%s", pointer, stderr)
	}
	noneEchoed(t, "refused claim "+pointer, doc, stdout, stderr)
	if after := factState(t, home); after != before {
		t.Errorf("%s: the refusal wrote:\n%s\n---\n%s", pointer, before, after)
	}
	return env
}

func worktreeID(t *testing.T, home datadir.Dir, path string) string {
	t.Helper()
	got := rows(t, home, `SELECT id FROM worktrees WHERE path = ?`, path)
	if len(got) != 1 {
		t.Fatalf("worktree %s: %v", path, got)
	}
	return got[0][0]
}

// MVP-CLM-04, MVP-INV-02, MVP-INV-06, MVP-WT-03 and WP2.4 spec §9.2: a
// sync batch from the main worktree is stored with the code context the
// CLI captured (the worktree, its HEAD, clean) and no comparison start
// yet, using exactly one seq for the batch, its claims and its unmapped
// work. Every reference is marked, not refused: a file of HEAD exists, a
// file that is not there, or a range past its end, is missing; a commit
// prefix of HEAD resolves, one naming nothing does not. Tests are counted.
// The receipt and the rows match the input; no summary or test name is
// echoed; the actor is the Agent's.
func TestClaimRecordsASyncBatch(t *testing.T) {
	home, f, dir := claimsProject(t)
	head := f.Repo.Git("rev-parse", "HEAD")
	absent := "fedcba9"
	if f.Repo.Git("rev-parse", "--disambiguate="+absent) != "" {
		t.Fatalf("%s names an object", absent)
	}
	doc := claimDoc(claim.Sync, 1,
		claimOn("AC-001-1", 1, "done", map[string]any{
			"files":   []any{fileRef("docs/client-brief.md", "1-2"), fileRef("internal/none.go", ""), fileRef("docs/client-brief.md", "1-99999")},
			"tests":   []any{"TestCanaryOne", "TestCanaryTwo"},
			"commits": []any{head[:7], absent, head},
		}),
		claimOn("AC-002-1", 1, "partial", nil))
	doc["unmapped_work"] = []any{map[string]any{"summary": "canary-unmapped-summary", "refs": map[string]any{"files": []any{fileRef("docs/client-brief.md", "")}}}}
	before, _ := strconv.ParseInt(seq(t, home), 10, 64)
	r := claimOK(t, home, dir, doc)
	if r.Seq != before+1 || seq(t, home) != strconv.FormatInt(before+1, 10) {
		t.Errorf("seq %d, counter %s, before %d: one seq for the batch", r.Seq, seq(t, home), before)
	}
	if r.Kind != "sync" || r.BaseVersion != 1 || deref(r.Context.Head) != head || r.Context.TreeState != "clean" ||
		r.Context.WorktreeID != worktreeID(t, home, dir) {
		t.Errorf("receipt = %+v", r)
	}
	if r.CompareFrom == nil || r.CompareFrom.Source != "none" || r.CompareFrom.Commit != nil {
		t.Errorf("compare_from = %+v; want none", r.CompareFrom)
	}
	got := []string{r.Claims[0].Refs.marks(), r.Claims[1].Refs.marks(), r.UnmappedWork[0].Refs.marks()}
	want := []string{"exists missing missing resolved unknown_commit resolved tests=2", "tests=0", "exists tests=0"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marks %v; want %v", got, want)
	}
	if got := rows(t, home, `SELECT actor, channel FROM claim_batches`); !equalRows(got, [][]string{{"agent:unknown", "agent_cli"}}) {
		t.Errorf("actor and channel %v", got)
	}

	// The text output names the counts and the marks' places, and echoes
	// nothing either.
	code, stdout, stderr := specmetry(t, dir, "", "claim", "--file", writeClaims(t, doc))
	if code != apperr.ExitOK {
		t.Fatalf("text: exit %d\n%s%s", code, stdout, stderr)
	}
	for _, s := range []string{"seq " + strconv.FormatInt(before+2, 10), "2 claims", "1 unmapped work item", head, "previous batch",
		"claim 0 (AC-001-1), file reference 1: missing", "claim 0 (AC-001-1), commit reference 1: unknown_commit", "not verification"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("the text output lacks %q:\n%s", s, stdout)
		}
	}
	noneEchoed(t, "text", doc, stdout, stderr)
}

// WP2.4 spec §6.1, §7 (review 065): the file references are checked
// against the HEAD the CLI captured and the batch records, even when HEAD
// moves on before they are checked: a file of that commit exists, a file
// only the next commit has is missing, and each result names the captured
// commit.
func TestClaimChecksFilesAtTheCapturedHead(t *testing.T) {
	home, f, dir := claimsProject(t)
	captured := f.Repo.Git("rev-parse", "HEAD")
	testHookAfterCapture = func() {
		testHookAfterCapture = nil
		f.Repo.WriteFile("docs/later.md", "later\n")
		f.Repo.Commit("later")
	}
	t.Cleanup(func() { testHookAfterCapture = nil })
	r := claimOK(t, home, dir, claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", map[string]any{
		"files": []any{fileRef("docs/client-brief.md", ""), fileRef("docs/later.md", "")}})))
	if deref(r.Context.Head) != captured || r.Claims[0].Refs.marks() != "exists missing tests=0" {
		t.Errorf("head %s (captured %s), marks %s", deref(r.Context.Head), captured, r.Claims[0].Refs.marks())
	}
	if moved := f.Repo.Git("rev-parse", "HEAD"); moved == captured {
		t.Fatal("HEAD did not move, so this test shows nothing")
	}
}

// The positive cases of WP2.4 spec §13: an audit of all requirements with
// an unknown assertion; an audit declaring two requirements that claims
// one criterion of one of them (D3 asks no complete inventory); an audit
// and a sync with no claims; a batch of unmapped work only. An audit
// receipt has no comparison start; the declared scope is stored. (Outside
// git: none of it depends on it.)
func TestClaimPositiveCases(t *testing.T) {
	home, dir := claimsProjectOutsideGit(t)
	r := claimOK(t, home, dir, claimDoc(claim.Audit, 1, claimOn("AC-001-1", 1, "unknown", nil), claimOn("AC-001-2", 1, "done", nil)))
	if r.CompareFrom != nil || r.Kind != "audit" {
		t.Errorf("audit receipt = %+v", r)
	}
	some := claimDoc(claim.Audit, 1, claimOn("AC-001-1", 1, "not_done", nil))
	some["audit_scope"] = map[string]any{"requirements": []any{"REQ-001", "REQ-002"}}
	claimOK(t, home, dir, some)
	empty := claimDoc(claim.Audit, 1)
	empty["audit_scope"] = map[string]any{"requirements": []any{"REQ-002"}}
	claimOK(t, home, dir, empty)
	claimOK(t, home, dir, claimDoc(claim.Sync, 1))
	only := claimDoc(claim.Sync, 1)
	only["unmapped_work"] = []any{map[string]any{"summary": "canary-unmapped-a"}, map[string]any{"summary": "canary-unmapped-b",
		"refs": map[string]any{"tests": []any{"TestCanaryUnmapped"}}}}
	r = claimOK(t, home, dir, only)
	if len(r.Claims) != 0 || len(r.UnmappedWork) != 2 || r.UnmappedWork[1].Refs.marks() != "tests=1" {
		t.Errorf("unmapped only = %+v", r)
	}
	if got := rows(t, home, `SELECT count(*) FROM claims WHERE target_type = 'none' AND criterion_id IS NULL AND criterion_revision IS NULL AND assertion IS NULL`); !equalRows(got, [][]string{{"2"}}) {
		t.Errorf("unmapped rows %v", got)
	}
}

// E3 (WP2.4 spec §6.3, §14): a sync batch compares from the latest sync or
// audit batch of its own worktree when that one was clean with a head, and
// from none otherwise; an older clean batch is not looked for, a manual
// batch never counts, nor does another worktree's; audit and manual
// batches record no comparison start.
func TestClaimCompareFrom(t *testing.T) {
	home, f, dir := claimsProject(t)
	h0 := f.Repo.Git("rev-parse", "HEAD")
	compare := func(doc map[string]any, at string) string {
		t.Helper()
		r := claimOK(t, home, at, doc)
		if r.CompareFrom == nil {
			return "-"
		}
		return r.CompareFrom.Source + ":" + deref(r.CompareFrom.Commit)
	}
	sync := claimDoc(claim.Sync, 1)
	steps := []string{}
	steps = append(steps, compare(sync, dir))
	steps = append(steps, compare(sync, dir))
	f.Repo.WriteFile("scratch.txt", "x\n")
	steps = append(steps, compare(sync, dir)) // dirty itself, from the clean one before
	if err := os.Remove(filepath.Join(dir, "scratch.txt")); err != nil {
		t.Fatal(err)
	}
	steps = append(steps, compare(sync, dir)) // the latest is dirty
	f.Repo.WriteFile("docs/more.md", "more\n")
	h1 := f.Repo.Commit("more")
	steps = append(steps, compare(claimDoc(claim.Audit, 1), dir))
	steps = append(steps, compare(sync, dir)) // from the audit
	f.Repo.WriteFile("scratch.txt", "x\n")
	steps = append(steps, compare(sync, dir))
	if err := os.Remove(filepath.Join(dir, "scratch.txt")); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := asHuman(t, dir, "v1", "claim", "--file", writeClaims(t, claimDoc(claim.Manual, 1)))
	if code != apperr.ExitOK {
		t.Fatalf("manual: exit %d\n%s%s", code, stdout, stderr)
	}
	steps = append(steps, compare(sync, dir)) // the manual batch does not count
	wt := f.Repo.AddWorktree("feature")
	steps = append(steps, compare(sync, wt), compare(sync, wt))
	want := []string{"none:null", "previous_batch:" + h0, "previous_batch:" + h0, "none:null", "-", "previous_batch:" + h1,
		"previous_batch:" + h1, "none:null", "none:null", "previous_batch:" + h1}
	if !reflect.DeepEqual(steps, want) {
		t.Errorf("compare_from\n%v\nwant\n%v", steps, want)
	}
	got := rows(t, home, `SELECT kind, tree_state, coalesce(compare_from_source, '-') FROM claim_batches ORDER BY seq`)
	var states []string
	for _, row := range got {
		states = append(states, strings.Join(row, "/"))
	}
	wantStates := []string{"sync/clean/none", "sync/clean/previous_batch", "sync/dirty/previous_batch", "sync/clean/none", "audit/clean/-",
		"sync/clean/previous_batch", "sync/dirty/previous_batch", "manual/clean/-", "sync/clean/none", "sync/clean/none", "sync/clean/previous_batch"}
	if !reflect.DeepEqual(states, wantStates) {
		t.Errorf("batches\n%v\nwant\n%v", states, wantStates)
	}
}

// WP2.4 spec §13's counterexamples, each exit code with its pointer,
// nothing echoed and nothing written: the input contract (V-C01, V-C05),
// the rules without a database (V-C03, V-C04, V-P04), secrets in any
// string, then the baseline under the write lock: a target that does not
// exist, only a pending proposal would add, or was retired (V-C03,
// MVP-CLM-02), the audit scope's requirements (D2) and its criteria (D3),
// all with the current base version; a stale base version or revision
// (exit 5, CX-13); an uninitialized project (exit 4). None of it depends on
// git, so the project is outside git; the refusals that do are in
// TestClaimWorktreeState and TestClaimNeedsASafeGit.
func TestClaimRefuses(t *testing.T) {
	home, dir := claimsProjectOutsideGit(t)
	secret := testutil.NewSecretCanary(t)
	base := func() map[string]any { return claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", nil)) }
	set := func(doc map[string]any, pointer string, v any) map[string]any {
		return patched(t, doc, pointer, v).(map[string]any)
	}
	auditOf := func(scope []any, claims ...any) map[string]any {
		doc := claimDoc(claim.Audit, 1, claims...)
		doc["audit_scope"] = map[string]any{"requirements": scope}
		return doc
	}
	for _, c := range []struct {
		name    string
		doc     map[string]any
		code    int
		pointer string
	}{
		{"an unknown field", set(base(), "/claims/0/code", "x"), 2, ""},
		{"another kind", set(base(), "/kind", "review"), 2, ""},
		{"an audit without a scope", func() map[string]any { d := claimDoc(claim.Audit, 1); delete(d, "audit_scope"); return d }(), 2, ""},
		{"unknown in a sync batch", set(base(), "/claims/0/assertion", "unknown"), 2, ""},
		{"unknown in a manual batch", set(set(base(), "/kind", "manual"), "/claims/0/assertion", "unknown"), 2, ""},
		{"a scope on a sync batch", set(base(), "/audit_scope", map[string]any{"all": true}), 2, ""},
		{"a criterion claimed twice", claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", nil), claimOn("AC-001-2", 1, "done", nil),
			claimOn("AC-001-1", 1, "partial", nil)), 2, "/claims/2/target"},
		{"a summary of 501 characters", set(base(), "/claims/0/summary", strings.Repeat("x", 501)), 2, ""},
		{"a code fence", set(base(), "/summary", "see\n```\nx\n```"), 2, ""},
		{"a blank line", set(base(), "/claims/0/summary", "one\n\ntwo"), 2, ""},
		{"a secret in the batch summary", set(base(), "/summary", "token "+secret), 2, "/summary"},
		{"a secret in a claim summary", set(base(), "/claims/0/summary", "token "+secret), 2, "/claims/0/summary"},
		{"a secret in a test name", set(base(), "/claims/0/refs", map[string]any{"tests": []any{secret}}), 2, "/claims/0/refs/tests/0"},
		{"a secret in a path", set(base(), "/claims/0/refs", map[string]any{"files": []any{fileRef("docs/"+secret+".md", "")}}), 2, "/claims/0/refs/files/0/path"},
		{"a secret in unmapped work", set(base(), "/unmapped_work", []any{map[string]any{"summary": "token " + secret}}), 2, "/unmapped_work/0/summary"},
		{"a path with a NUL", set(base(), "/claims/0/refs", map[string]any{"files": []any{fileRef("a\x00b.go", "")}}), 2, "/claims/0/refs/files/0/path"},
		{"a reversed line range", set(base(), "/claims/0/refs", map[string]any{"files": []any{fileRef("a.go", "9-3")}}), 2, "/claims/0/refs/files/0/lines"},
		{"a line past the limit", set(base(), "/unmapped_work", []any{map[string]any{"summary": "canary-unmapped-lines", "refs": map[string]any{
			"files": []any{fileRef("a.go", "2147483648")}}}}), 2, "/unmapped_work/0/refs/files/0/lines"},
		{"an upper-case commit prefix", set(base(), "/claims/0/refs", map[string]any{"commits": []any{"ABCDEF1"}}), 2, ""},
		{"a short commit prefix", set(base(), "/claims/0/refs", map[string]any{"commits": []any{"abcdef"}}), 2, ""},
		{"a target that does not exist", claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", nil), claimOn("AC-009-1", 1, "done", nil)), 2, "/claims/1/target"},
		{"D2: a requirement that does not exist", auditOf([]any{"REQ-001", "REQ-009"}), 2, "/audit_scope/requirements/1"},
		{"D3: a criterion outside the declared requirements", auditOf([]any{"REQ-001"}, claimOn("AC-001-1", 1, "done", nil),
			claimOn("AC-002-1", 1, "done", nil)), 2, "/claims/1/target"},
		{"CX-13: a stale revision", claimDoc(claim.Sync, 1, claimOn("AC-001-2", 1, "done", nil), claimOn("AC-001-1", 2, "done", nil)), 5, "/claims/1/target_revision"},
		{"a stale base version", set(base(), "/base_version", 0), 5, "/base_version"},
		{"a base version from the future", set(base(), "/base_version", 7), 5, "/base_version"},
	} {
		t.Run(c.name, func(t *testing.T) {
			env := claimRefused(t, home, dir, c.doc, c.code, c.pointer)
			if c.code == apperr.ExitStalePrecondition && toFloat(env.Error.Current["base_version"]) != 1 {
				t.Errorf("current = %v", env.Error.Current)
			}
		})
	}
	stale := claimRefused(t, home, dir, claimDoc(claim.Sync, 1, claimOn("AC-001-1", 2, "done", nil)), 5, "/claims/0/target_revision")
	if got := asJSON(t, stale.Error.Current["targets"]); got != `[{"pointer":"/claims/0/target_revision","revision":1,"target":"AC-001-1"}]` {
		t.Errorf("current targets = %s", got)
	}

	// Not valid JSON at all, and no file.
	notJSON := filepath.Join(t.TempDir(), "claims.json")
	if err := os.WriteFile(notJSON, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := specmetry(t, dir, "", "claim", "--file", notJSON, "--json")
	if code != apperr.ExitInvalidInput || !strings.Contains(stdout, "invalid_input") {
		t.Errorf("not JSON: exit %d\n%s", code, stdout)
	}
	if code, stdout, _ := specmetry(t, dir, "", "claim", "--json"); code != apperr.ExitInvalidInput {
		t.Errorf("no --file: exit %d\n%s", code, stdout)
	}

	// Only in a pending proposal: AC-001-3 would be its key.
	proposeOK(t, dir, small(1, addAC("c9", "REQ-001", "an admin can unlock")))
	claimRefused(t, home, dir, claimDoc(claim.Sync, 1, claimOn("AC-001-3", 1, "done", nil)), 2, "/claims/0/target")

	// Retired, with the current base version (MVP-CLM-02): AC-001-2 at
	// version 2, then REQ-002 (and AC-002-1 with it) at version 3.
	acceptAll(t, dir, proposeOK(t, dir, small(1, retire("criterion", "AC-001-2", 1))), 1, nil)
	claimRefused(t, home, dir, claimDoc(claim.Sync, 2, claimOn("AC-001-1", 1, "done", nil), claimOn("AC-001-2", 1, "done", nil)), 2, "/claims/1/target")
	acceptAll(t, dir, proposeOK(t, dir, small(2, retire("requirement", "REQ-002", 1))), 2, nil)
	retired := claimDoc(claim.Audit, 3, claimOn("AC-002-1", 1, "done", nil))
	retired["audit_scope"] = map[string]any{"requirements": []any{"REQ-001", "REQ-002"}}
	env := claimRefused(t, home, dir, retired, 2, "/audit_scope/requirements/1")
	pointers := []string{}
	for _, d := range env.Error.Details {
		pointers = append(pointers, d.Pointer)
	}
	if !slices.Contains(pointers, "/claims/0/target") {
		t.Errorf("the retired target is not reported with the scope: %v", pointers)
	}

	// Not initialized: a directory of no registered project.
	fresh, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, _ = specmetry(t, fresh, "", "claim", "--file", writeClaims(t, base()), "--json")
	if code != apperr.ExitNotInitialized || !strings.Contains(stdout, "not_initialized") {
		t.Errorf("not initialized: exit %d\n%s", code, stdout)
	}
}

// WP2.4 spec §5 step 10 and §15: a worktree a claim names for the first
// time is registered in the claim's transaction with a seq of its own, just
// before the batch's; when the claim is refused there (a stale revision),
// or its receipt does not conform (exit 1: the receipt is validated before
// COMMIT, in text as with --json, review 067 B2), the registration is
// rolled back with everything else, nothing reports success, and the
// injected value is not echoed.
func TestClaimRegistersTheWorktreeOrNothing(t *testing.T) {
	home, f, _ := claimsProject(t)
	wt := f.Repo.AddWorktree("feature-a")
	claimRefused(t, home, wt, claimDoc(claim.Sync, 1, claimOn("AC-001-1", 3, "done", nil)), 5, "/claims/0/target_revision")

	canary := testutil.NewCanary(t)
	testHookClaimReceipt = func(r *claim.Receipt) { r.BatchID = canary }
	t.Cleanup(func() { testHookClaimReceipt = nil })
	doc := claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", nil))
	for _, mode := range []string{"--json", "text"} {
		before := factState(t, home)
		args := []string{"claim", "--file", writeClaims(t, doc)}
		if mode == "--json" {
			args = append(args, "--json")
		}
		code, stdout, stderr := specmetry(t, wt, "", args...)
		if code != apperr.ExitInternal || strings.Contains(stdout, "stored at seq") || strings.Contains(stdout+stderr, canary) {
			t.Errorf("%s, a receipt that does not conform: exit %d\n%s%s", mode, code, stdout, stderr)
		}
		if mode == "--json" {
			var env apperr.Envelope
			if err := json.Unmarshal([]byte(wholeJSON(t, stdout)), &env); err != nil || env.Error.Code != apperr.Internal || !strings.Contains(env.Error.Message, "claim-receipt.v1") {
				t.Errorf("--json: %v\n%s", err, stdout)
			}
		} else if !strings.Contains(stderr, "claim-receipt.v1") || stdout != "" {
			t.Errorf("text: stdout %q, stderr %q", stdout, stderr)
		}
		if after := factState(t, home); after != before {
			t.Errorf("%s, a receipt that does not conform, wrote:\n%s\n---\n%s", mode, before, after)
		}
	}
	testHookClaimReceipt = nil

	before2, _ := strconv.ParseInt(seq(t, home), 10, 64)
	r := claimOK(t, home, wt, doc)
	got := rows(t, home, `SELECT registered_seq, is_main FROM worktrees WHERE id = ?`, r.Context.WorktreeID)
	if r.Seq != before2+2 || !equalRows(got, [][]string{{strconv.FormatInt(before2+1, 10), "0"}}) || r.Context.WorktreeID != worktreeID(t, home, wt) {
		t.Errorf("seq %d after %d, worktree %v", r.Seq, before2, got)
	}
}

// WP2.4 spec §6.1, §7 outside git: the head is null and the state
// unknown; a file under the project root exists; a commit prefix never
// resolves and no git object is read for it (a git that would fetch is on
// PATH, and every object read would be refused). The next sync batch
// compares from none.
func TestClaimOutsideGit(t *testing.T) {
	noAgentVariables(t)
	home := testutil.TempHome(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "brief.md"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustInit(t, root, "--name", "plain")
	acceptAll(t, root, proposeOK(t, root, small(0, addReq("r1", "sign up", nil), addAC("c1", "r1", "a new account can sign in"))), 0, nil)
	calls := fetchingGit(t)
	doc := claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", map[string]any{
		"files": []any{fileRef("docs/brief.md", "2"), fileRef("docs/none.md", "")}, "commits": []any{"abcdef1"}}))
	r := claimOK(t, home, root, doc)
	if r.Context.Head != nil || r.Context.TreeState != "unknown" || r.Claims[0].Refs.marks() != "exists missing unknown_commit tests=0" {
		t.Errorf("receipt = %+v", r)
	}
	if r = claimOK(t, home, root, claimDoc(claim.Sync, 1)); r.CompareFrom == nil || r.CompareFrom.Source != "none" {
		t.Errorf("compare_from after an unknown batch = %+v", r.CompareFrom)
	}
	noObjectReads(t, calls())
}

// D1 (WP2.4 spec §6.2): a worktree state git cannot read for an actual
// environmental reason (here a git without ls-files --sparse, which the
// state read needs) is exit 6, with no batch, claim or worktree
// registration and no seq; the state Clean cannot decide (a sparse index,
// P1) is recorded as unknown and the batch is stored.
func TestClaimWorktreeState(t *testing.T) {
	home, f, _ := claimsProject(t)
	wt := f.Repo.AddWorktree("feature-d1")
	dir, env, _ := testutil.PassthroughGit(t, testutil.PassWithoutLsFilesSparse)
	path := os.Getenv("PATH")
	useGitFor(t, dir, env)
	claimRefused(t, home, wt, claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", nil)), 6, "")
	t.Setenv("PATH", path)

	f.Repo.GitIn(wt, "sparse-checkout", "init", "--cone", "--sparse-index")
	f.Repo.GitIn(wt, "sparse-checkout", "set", "docs")
	gitDir := strings.TrimSpace(f.Repo.GitIn(wt, "rev-parse", "--absolute-git-dir"))
	if b, err := os.ReadFile(filepath.Join(gitDir, "index")); err != nil || !strings.Contains(string(b), "sdir") {
		t.Skip("this git keeps no sparse index")
	}
	r := claimOK(t, home, wt, claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", nil)))
	if r.Context.TreeState != "unknown" || r.Context.Head == nil {
		t.Errorf("receipt = %+v", r)
	}
}

// Review 067 B1 through the claim command: a commit prefix that names a
// blob is marked unknown_commit and stored while the blob is healthy; once
// the blob's loose file still inflates a header but no longer holds the
// object its ID names, the same batch is exit 6 with nothing written (no
// batch, claim, unmapped work, worktree registered for it, or seq), with
// --json and in text, alone or among healthy references, and the blob's
// content is in no output and nowhere in the data directory. Repaired, the
// batch is stored again.
func TestClaimRefusesADamagedCommitCandidate(t *testing.T) {
	home, f, dir := claimsProject(t)
	canary := testutil.NewCanary(t)
	content := filepath.Join(t.TempDir(), "blob.txt")
	if err := os.WriteFile(content, []byte("healthy "+canary+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	blob := f.Repo.Git("hash-object", "-w", content)
	head := f.Repo.Git("rev-parse", "HEAD")
	doc := func(commits ...any) map[string]any {
		d := claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", map[string]any{"commits": commits}))
		d["unmapped_work"] = []any{map[string]any{"summary": "canary-unmapped-summary"}}
		return d
	}
	if r := claimOK(t, home, dir, doc(blob)); r.Claims[0].Refs.marks() != "unknown_commit tests=0" {
		t.Fatalf("a healthy blob: %s", r.Claims[0].Refs.marks())
	}

	p := filepath.Join(dir, ".git", "objects", blob[:2], blob[2:])
	original, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	z := zlib.NewWriter(&b)
	_, _ = z.Write([]byte("blob 500\x00cut " + canary))
	_ = z.Close()
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, refs := range [][]any{{blob}, {head[:7], blob}} {
		env := claimRefused(t, home, dir, doc(refs...), 6, "")
		if !strings.Contains(env.Error.Message, "cannot be resolved reliably") || strings.Contains(env.Error.Message, canary) {
			t.Errorf("refs %v: %s", refs, env.Error.Message)
		}
	}
	wt := f.Repo.AddWorktree("feature-b1")
	before := factState(t, home)
	code, stdout, stderr := specmetry(t, wt, "", "claim", "--file", writeClaims(t, doc(blob)))
	if code != apperr.ExitEnvironment || strings.Contains(stdout, "stored at seq") || !strings.Contains(stderr, "cannot be resolved reliably") {
		t.Errorf("text, from a new worktree: exit %d\n%s%s", code, stdout, stderr)
	}
	if after := factState(t, home); after != before {
		t.Errorf("text, from a new worktree, wrote:\n%s\n---\n%s", before, after)
	}
	if strings.Contains(stdout+stderr, canary) {
		t.Errorf("the blob's content reached the output:\n%s%s", stdout, stderr)
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Errorf("the blob's content reached the data directory: %v", hits)
	}

	if err := os.WriteFile(p, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if r := claimOK(t, home, wt, doc(head[:7], blob)); r.Claims[0].Refs.marks() != "resolved unknown_commit tests=0" {
		t.Errorf("repaired: %s", r.Claims[0].Refs.marks())
	}
}

// useGitFor puts dir first on PATH and sets env for the rest of the test.
func useGitFor(t *testing.T, dir string, env []string) {
	t.Helper()
	for _, kv := range env {
		name, value, _ := strings.Cut(kv, "=")
		t.Setenv(name, value)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// PD-26 (WP2.4 spec §7): with a git that would fetch a missing object, a
// batch is refused with exit 6 and nothing written, with references or
// without (its HEAD is a commit to read); in a partial clone a batch with
// file or commit references is refused too, and no object is read.
func TestClaimNeedsASafeGit(t *testing.T) {
	home, f, dir := claimsProject(t)
	head := f.Repo.Git("rev-parse", "HEAD")
	f.Repo.Git("config", "remote.origin.promisor", "true")
	for _, refs := range []map[string]any{
		{"files": []any{fileRef("docs/client-brief.md", "")}},
		{"commits": []any{head[:7]}},
	} {
		env := claimRefused(t, home, dir, claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", refs)), 6, "")
		if !strings.Contains(env.Error.Message, "partial clone") {
			t.Errorf("refs %v: %s", refs, env.Error.Message)
		}
	}
	f.Repo.Git("config", "--unset", "remote.origin.promisor")
	calls := fetchingGit(t)
	for _, refs := range []map[string]any{nil, {"files": []any{fileRef("docs/client-brief.md", "")}}, {"commits": []any{head[:7]}}} {
		env := claimRefused(t, home, dir, claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", refs)), 6, "")
		if !strings.Contains(env.Error.Message, "GIT_NO_LAZY_FETCH") {
			t.Errorf("refs %v: %s", refs, env.Error.Message)
		}
	}
	noObjectReads(t, calls())
}
