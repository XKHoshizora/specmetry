package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// targetScopeOut is target-scope.v1, decoded.
type targetScopeOut struct {
	Target *struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		Ref        string  `json:"ref"`
		Snapshot   *string `json:"snapshot"`
		CreatedSeq int64   `json:"created_seq"`
	} `json:"target"`
	Scope *struct {
		ConfirmationID  string `json:"confirmation_id"`
		Seq             int64  `json:"seq"`
		BaselineVersion int    `json:"baseline_version"`
		SourceSetHash   string `json:"source_set_hash"`
		Requirements    []struct {
			Key      string `json:"key"`
			Revision int    `json:"revision"`
		} `json:"requirements"`
		Attestations        map[string]any `json:"attestations"`
		AcknowledgedWaivers []string       `json:"acknowledged_waivers"`
		Stale               bool           `json:"stale"`
	} `json:"scope"`
	ConfirmInputs struct {
		TargetID                *string          `json:"target_id"`
		SeenScopeConfirmationID *string          `json:"seen_scope_confirmation_id"`
		BaselineVersion         int              `json:"baseline_version"`
		SourceSetHash           string           `json:"source_set_hash"`
		PendingItems            int              `json:"pending_items"`
		MustRequirements        []string         `json:"must_requirements"`
		EffectiveWaivers        []map[string]any `json:"effective_waivers"`
	} `json:"confirm_inputs"`
}

// scoped is the confirmed scope as KEY@revision, "" without one.
func (o targetScopeOut) scoped() string {
	if o.Scope == nil {
		return ""
	}
	var keys []string
	for _, r := range o.Scope.Requirements {
		keys = append(keys, r.Key+"@"+strconv.Itoa(r.Revision))
	}
	return strings.Join(keys, " ")
}

// targetScope reads target-scope.v1 from the Agent channel with target show
// --json and with scope show --json, checked against the schema; both print
// the same document. Then what the schema cannot express: confirm_inputs
// names the target and the confirmation shown.
func targetScope(t *testing.T, dir string) targetScopeOut {
	t.Helper()
	code, stdout, stderr := specmetry(t, dir, "target-scope.v1", "target", "show", "--json")
	if code != apperr.ExitOK {
		t.Fatalf("target show: exit %d\n%s%s", code, stdout, stderr)
	}
	code, again, stderr := specmetry(t, dir, "target-scope.v1", "scope", "show", "--json")
	if code != apperr.ExitOK || again != stdout {
		t.Fatalf("scope show: exit %d, not the target show document\n%s\n%s%s", code, stdout, again, stderr)
	}
	var out targetScopeOut
	if err := json.Unmarshal([]byte(wholeJSON(t, stdout)), &out); err != nil {
		t.Fatal(err)
	}
	ci := out.ConfirmInputs
	if (out.Target == nil) != (ci.TargetID == nil) || (out.Target != nil && *ci.TargetID != out.Target.ID) {
		t.Fatalf("confirm_inputs.target_id is not the target's ID:\n%s", stdout)
	}
	if (out.Scope == nil) != (ci.SeenScopeConfirmationID == nil) || (out.Scope != nil && *ci.SeenScopeConfirmationID != out.Scope.ConfirmationID) {
		t.Fatalf("confirm_inputs.seen_scope_confirmation_id is not the confirmation shown:\n%s", stdout)
	}
	return out
}

// scopeFields are the fields of a scope confirmation made from what ts
// says a new one must carry, for keys: the waivers in effect under them
// acknowledged, the confirmation shown as seen.
func scopeFields(ts targetScopeOut, keys ...any) map[string]any {
	ci := ts.ConfirmInputs
	acknowledged := []any{}
	for _, w := range ci.EffectiveWaivers {
		if slices.Contains(keys, w["requirement"]) {
			acknowledged = append(acknowledged, w["decision_id"])
		}
	}
	var target, seen any
	if ci.TargetID != nil {
		target = *ci.TargetID
	}
	if ci.SeenScopeConfirmationID != nil {
		seen = *ci.SeenScopeConfirmationID
	}
	return map[string]any{"target_id": target, "source_set_hash": ci.SourceSetHash, "requirements": keys,
		"attestations": map[string]any{"complete_scope": true, "pending_items_excluded": ci.PendingItems}, "acknowledged_waivers": acknowledged,
		"seen_scope_confirmation_id": seen}
}

// scopeFile is scopeFields as a scope.v1 document, changed by change (if
// given), written to a file.
func scopeFile(t *testing.T, ts targetScopeOut, keys []string, change func(map[string]any)) string {
	t.Helper()
	list := make([]any, len(keys))
	for i, k := range keys {
		list[i] = k
	}
	doc := scopeFields(ts, list...)
	doc["schema"] = "specmetry.scope/v1"
	doc["baseline_version"] = ts.ConfirmInputs.BaselineVersion
	if change != nil {
		change(doc)
	}
	return writeReview(t, doc)
}

// scopeState is everything target set and scope confirm may write: the
// seq counter, every target (ref, name, created and archived seq,
// channel), the confirmations and their items; and what they must not:
// the baseline and the pending items.
func scopeState(t *testing.T, home datadir.Dir) string {
	t.Helper()
	return strings.Join(rows(t, home, `SELECT (SELECT seq_counter FROM meta),
    (SELECT group_concat(d, ' ') FROM (SELECT ref || ':' || name || ':' || created_seq || ':' || coalesce(archived_seq, '-') || ':' || channel AS d
        FROM delivery_targets ORDER BY created_seq)),
    (SELECT count(*) FROM scope_confirmations), (SELECT count(*) FROM scope_items),
    (SELECT count(*) FROM baseline_versions), (SELECT count(*) || '/' || coalesce(sum(revision), 0) FROM requirements),
    (SELECT count(*) FROM proposal_items WHERE reviewed_seq IS NULL)`)[0], " ")
}

// humanRefused runs a human-only command with --json as a person at a
// terminal who would confirm with confirm: it must be refused with code
// before anybody is asked, with error.v1 holding a detail at pointer (if
// given) and none of hidden, and nothing written.
func humanRefused(t *testing.T, home datadir.Dir, dir, confirm string, code int, pointer string, hidden string, args ...string) apperr.Envelope {
	t.Helper()
	before := scopeState(t, home)
	got, stdout, stderr := asHuman(t, dir, confirm, append(args, "--json")...)
	what := strings.Join(args, " ")
	if got != code {
		t.Errorf("%s: exit %d, want %d\n%s%s", what, got, code, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "to confirm") {
		t.Errorf("%s: the person was asked to confirm what is refused", what)
	}
	if hidden != "" && strings.Contains(stdout+stderr, hidden) {
		t.Errorf("%s: the output echoes %q", what, hidden)
	}
	val, err := schemas()
	if err != nil {
		t.Fatal(err)
	}
	var env apperr.Envelope
	if err := val.ValidateJSON("error.v1", []byte(wholeJSON(t, stdout))); err != nil {
		t.Errorf("%s: stdout is not error.v1: %v\n%s", what, err, stdout)
	} else if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Error(err)
	}
	if pointer != "" {
		found := false
		for _, d := range env.Error.Details {
			found = found || d.Pointer == pointer
		}
		if !found {
			t.Errorf("%s: no detail at %s: %+v (%s)", what, pointer, env.Error.Details, env.Error.Message)
		}
	}
	if after := scopeState(t, home); after != before {
		t.Errorf("%s: written: %s -> %s", what, before, after)
	}
	return env
}

// confirmIn runs the CLI in this process in the current directory as a
// person who confirms with confirm; for a callback outside the test's
// goroutine (no t.Chdir, no t.Fatal).
func confirmIn(confirm string, args ...string) (int, string) {
	var out, errOut bytes.Buffer
	root := newRoot(&out, &errOut)
	root.SetIn(strings.NewReader(confirm + "\n"))
	code := executeAs(context.Background(), atTerminal, root, args, &errOut)
	return code, out.String() + errOut.String()
}

// scopeProject is a project at baseline version 2 with REQ-001 (must),
// REQ-002 (should) and REQ-003 (must), all at revision 1, and two items
// excluded from a scope: one in conflict (a second update of REQ-003,
// after the first was accepted) and one pending.
func scopeProject(t *testing.T) (datadir.Dir, *testutil.Fixture) {
	t.Helper()
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	should := addReq("b", "export", nil)
	should["data"].(map[string]any)["priority"] = "should"
	acceptAll(t, dir, proposeOK(t, dir, small(0, addReq("a", "sign up", nil), should, addReq("c", "audit log", nil))), 0, nil)
	first := proposeOK(t, dir, small(1, update("requirement", "REQ-003", 1, map[string]any{"title": "audit log of sign-ins"})))
	proposeOK(t, dir, small(1, update("requirement", "REQ-003", 1, map[string]any{"title": "audit trail"})))
	acceptAll(t, dir, first, 1, nil)
	proposeOK(t, dir, small(2, addReq("d", "import", nil)))
	return home, f
}

// MVP-TGT-01, TGT-02, FR-TGT-01, 02: init's default target is the main
// worktree's branch, named default, kept by init again. target show
// resolves the snapshot when read, never stores it (no column holds it):
// a ref that resolves to nothing shows null, then follows the branch, and
// an annotated tag shows its commit. target set is a person's at a
// terminal: the Agent channel and a declined or wrong confirmation are
// exit 3; a ref that is not a full branch or tag, an invalid ref name or
// target name, and a secret in either are exit 2 before anybody is asked,
// not echoed; none of them writes. Confirmed, the old target is archived
// and the new one created at one seq, by the person on human_cli; the
// same ref, not renamed, changes nothing. The scope confirmed for the old
// target does not carry over, and stays in the history.
func TestTargetShowAndSet(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	head := f.Repo.Git("rev-parse", "HEAD")
	branch := f.Repo.Git("symbolic-ref", "HEAD")
	ts := targetScope(t, dir)
	if ts.Target == nil || ts.Target.Ref != branch || ts.Target.Name != "default" || ts.Target.Snapshot == nil || *ts.Target.Snapshot != head ||
		ts.Scope != nil || ts.ConfirmInputs.BaselineVersion != 0 || len(ts.ConfirmInputs.MustRequirements) != 0 ||
		ts.ConfirmInputs.EffectiveWaivers == nil || len(ts.ConfirmInputs.EffectiveWaivers) != 0 {
		t.Fatalf("after init: %+v", ts)
	}
	initial, initialSeq := ts.Target.ID, ts.Target.CreatedSeq
	mustInit(t, dir)
	if again := targetScope(t, dir); again.Target.ID != initial {
		t.Fatalf("init again changed the target: %+v", again.Target)
	}
	code, stdout, stderr := specmetry(t, dir, "", "target", "show")
	if code != apperr.ExitOK || !strings.Contains(stdout, "Delivery target default: "+branch+", at "+head) ||
		!strings.Contains(stdout, "Scope: not confirmed") {
		t.Fatalf("target show: exit %d\n%s%s", code, stdout, stderr)
	}

	acceptAll(t, dir, proposeOK(t, dir, small(0, addReq("a", "sign up", nil))), 0, nil)
	if code, stdout, stderr := asHuman(t, dir, "v1", "scope", "confirm"); code != apperr.ExitOK {
		t.Fatalf("scope confirm: exit %d\n%s%s", code, stdout, stderr)
	}
	scopedFor := targetScope(t, dir).Scope.ConfirmationID

	secret := testutil.NewSecretCanary(t)
	release := []string{"target", "set", "--ref", "refs/heads/release"}
	for _, c := range []struct {
		args    []string
		pointer string
		hidden  string
	}{
		{[]string{"target", "set", "--ref", "refs/remotes/origin/main"}, "--ref", ""},
		{[]string{"target", "set", "--ref", "main"}, "--ref", ""},
		{[]string{"target", "set", "--ref", "refs/heads/"}, "--ref", ""},
		{[]string{"target", "set"}, "--ref", ""},
		{[]string{"target", "set", "--ref", "refs/heads/a..b"}, "--ref", ""},
		{[]string{"target", "set", "--ref", "refs/heads/rel@{1}"}, "--ref", ""},
		{[]string{"target", "set", "--ref", "refs/heads/" + secret}, "--ref", secret},
		{append(slices.Clone(release), "--name", ""), "--name", ""},
		{append(slices.Clone(release), "--name", "two\nlines"), "--name", ""},
		{append(slices.Clone(release), "--name", strings.Repeat("x", 201)), "--name", ""},
		{append(slices.Clone(release), "--name", "release "+secret), "--name", secret},
	} {
		humanRefused(t, home, dir, "refs/heads/release", apperr.ExitInvalidInput, c.pointer, c.hidden, c.args...)
	}
	before := scopeState(t, home)
	code, stdout, stderr = specmetry(t, dir, "error.v1", append(slices.Clone(release), "--json")...)
	if code != apperr.ExitHumanRequired || strings.Contains(stdout+stderr, "to confirm") {
		t.Fatalf("Agent: exit %d\n%s%s", code, stdout, stderr)
	}
	for _, answer := range []string{"", "refs/heads/main", "release", noPrompt} {
		if code, stdout, stderr := asHuman(t, dir, answer, release...); code != apperr.ExitHumanRequired {
			t.Fatalf("answered %q: exit %d\n%s%s", answer, code, stdout, stderr)
		}
	}
	if after := scopeState(t, home); after != before {
		t.Fatalf("written: %s -> %s", before, after)
	}

	code, stdout, stderr = asHuman(t, dir, "refs/heads/release", release...)
	if code != apperr.ExitOK || !strings.Contains(stdout, "new delivery target release: refs/heads/release; it resolves to no commit now") ||
		!strings.Contains(stdout, "the active target default ("+branch+") is archived") || !strings.Contains(stdout, "does not carry over") ||
		!strings.Contains(stdout, "Delivery target set to refs/heads/release (release)") {
		t.Fatalf("target set: exit %d\n%s%s", code, stdout, stderr)
	}
	seq0, _ := strconv.Atoi(strings.Fields(before)[0])
	got := rows(t, home, `SELECT ref, name, created_seq - ?, coalesce(archived_seq - ?, '-'), actor LIKE 'human:_%', channel FROM delivery_targets
ORDER BY created_seq`, seq0, seq0)
	if want := [][]string{{branch, "default", strconv.FormatInt(initialSeq-int64(seq0), 10), "1", "0", "system"},
		{"refs/heads/release", "release", "1", "-", "1", "human_cli"}}; !equalRows(got, want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
	if after := seq(t, home); after != strconv.Itoa(seq0+1) {
		t.Fatalf("seq %s, want %d", after, seq0+1)
	}
	ts = targetScope(t, dir)
	if ts.Target.Ref != "refs/heads/release" || ts.Target.Name != "release" || ts.Target.Snapshot != nil || ts.Target.CreatedSeq != int64(seq0+1) ||
		ts.Scope != nil || ts.ConfirmInputs.SeenScopeConfirmationID != nil {
		t.Fatalf("after target set: %+v %+v", ts.Target, ts.Scope)
	}
	if got := rows(t, home, `SELECT count(*) FROM scope_confirmations WHERE id = ? AND target_id = ?`, scopedFor, initial); got[0][0] != "1" {
		t.Fatalf("the old target's scope is gone: %v", got)
	}
	_, stdout, _ = specmetry(t, dir, "", "target", "show")
	if !strings.Contains(stdout, "refs/heads/release, which resolves to no commit now") {
		t.Fatalf("target show:\n%s", stdout)
	}
	f.Repo.Git("branch", "release")
	if ts := targetScope(t, dir); ts.Target.Snapshot == nil || *ts.Target.Snapshot != head {
		t.Fatalf("snapshot %v, want %s", ts.Target.Snapshot, head)
	}
	next := f.Repo.Commit("next")
	f.Repo.Git("branch", "-f", "release", next)
	if ts := targetScope(t, dir); *ts.Target.Snapshot != next {
		t.Fatalf("snapshot %v, want %s: it is resolved when read", *ts.Target.Snapshot, next)
	}

	before = scopeState(t, home)
	if code, stdout, stderr := asHuman(t, dir, noPrompt, release...); code != apperr.ExitOK || !strings.Contains(stdout, "nothing changed") {
		t.Fatalf("the same ref: exit %d\n%s%s", code, stdout, stderr)
	}
	if after := scopeState(t, home); after != before {
		t.Fatalf("the same ref wrote: %s -> %s", before, after)
	}
	f.Repo.Git("tag", "-a", "v1.0", "-m", "first release")
	if code, stdout, stderr := asHuman(t, dir, "refs/tags/v1.0", "target", "set", "--ref", "refs/tags/v1.0", "--name", "first release"); code != apperr.ExitOK ||
		!strings.Contains(stdout, "it resolves to "+next+" now") {
		t.Fatalf("a tag: exit %d\n%s%s", code, stdout, stderr)
	}
	if ts := targetScope(t, dir); ts.Target.Ref != "refs/tags/v1.0" || ts.Target.Name != "first release" || *ts.Target.Snapshot != next {
		t.Fatalf("tag target %+v", ts.Target)
	}
	if got := rows(t, home, `SELECT count(*), count(archived_seq) FROM delivery_targets`); !equalRows(got, [][]string{{"3", "2"}}) {
		t.Fatalf("targets: %v", got)
	}

	// Renamed, it is a new target. A name is shown with its control
	// characters escaped (a bidi override here) and kept as it is.
	renamed := "v1" + rlo + "-gold"
	code, stdout, stderr = asHuman(t, dir, "refs/tags/v1.0", "target", "set", "--ref", "refs/tags/v1.0", "--name", renamed)
	if code != apperr.ExitOK || strings.Contains(stdout, rlo) || !strings.Contains(stdout, "new delivery target v1"+vis("u202e")+"-gold") ||
		!strings.Contains(stdout, "the active target first release (refs/tags/v1.0) is archived") {
		t.Fatalf("renamed: exit %d\n%s%s", code, stdout, stderr)
	}
	_, stdout, _ = specmetry(t, dir, "", "target", "show")
	if strings.Contains(stdout, rlo) || !strings.Contains(stdout, "Delivery target v1"+vis("u202e")+"-gold: refs/tags/v1.0") {
		t.Fatalf("target show:\n%s", stdout)
	}
	if ts := targetScope(t, dir); ts.Target.Name != renamed {
		t.Fatalf("name %q, want %q", ts.Target.Name, renamed)
	}
}

// FR-INIT-09 and a detached HEAD: a project outside git has no target;
// target show says so, target set is exit 2, and scope confirm is exit 2
// with nothing to confirm for (exit 5 for a document naming a target:
// the target it was read for is not the active one, which is none). A
// git project initialized at a detached HEAD has none either, until a
// person sets the first one.
func TestTargetWithoutOne(t *testing.T) {
	f := testutil.BuildFixture(t, "fx-docs-only")
	home := testutil.TempHome(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "brief.md"), []byte("# brief\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustInit(t, root)
	ts := targetScope(t, root)
	if ts.Target != nil || ts.Scope != nil {
		t.Fatalf("outside git: %+v", ts)
	}
	if code, stdout, _ := specmetry(t, root, "", "scope", "show"); code != apperr.ExitOK || !strings.Contains(stdout, "No delivery target") {
		t.Fatalf("scope show: exit %d\n%s", code, stdout)
	}
	humanRefused(t, home, root, "refs/heads/main", apperr.ExitInvalidInput, "", "", "target", "set", "--ref", "refs/heads/main")
	humanRefused(t, home, root, "v0", apperr.ExitInvalidInput, "", "", "scope", "confirm", "--requirements", "REQ-001")
	env := humanRefused(t, home, root, "v1", apperr.ExitStalePrecondition, "/target_id", "", "scope", "confirm", "--file",
		scopeFile(t, ts, []string{"REQ-001"}, func(d map[string]any) { d["target_id"] = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"; d["baseline_version"] = 1 }))
	if v, ok := env.Error.Current["target_id"]; !ok || v != nil {
		t.Fatalf("current = %v", env.Error.Current)
	}

	home = testutil.TempHome(t)
	branch := f.Repo.Git("symbolic-ref", "HEAD")
	f.Repo.Git("checkout", "-q", "--detach")
	mustInit(t, f.Repo.Dir)
	if ts := targetScope(t, f.Repo.Dir); ts.Target != nil {
		t.Fatalf("detached: %+v", ts.Target)
	}
	code, stdout, stderr := asHuman(t, f.Repo.Dir, branch, "target", "set", "--ref", branch)
	if code != apperr.ExitOK || !strings.Contains(stdout, "the project has no delivery target now") {
		t.Fatalf("first target: exit %d\n%s%s", code, stdout, stderr)
	}
	if got := rows(t, home, `SELECT ref, count(archived_seq) FROM delivery_targets`); !equalRows(got, [][]string{{branch, "0"}}) {
		t.Fatalf("targets = %v", got)
	}
}

// Another connection changes the target while the person confirms: the
// confirmed change is stale (exit 5, the target now in error.current) and
// archives nothing; one seq is taken, by the other.
func TestTargetSetWhileConfirming(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	seq0, _ := strconv.Atoi(seq(t, home))
	var other int
	var otherOut string
	in := &onFirstRead{r: strings.NewReader("refs/heads/release\n"), before: func() {
		other, otherOut = confirmIn("refs/heads/hotfix", "target", "set", "--ref", "refs/heads/hotfix")
	}}
	code, stdout, stderr := asHumanWith(t, dir, in, "target", "set", "--ref", "refs/heads/release", "--json")
	if other != apperr.ExitOK {
		t.Fatalf("the other connection: exit %d\n%s", other, otherOut)
	}
	if code != apperr.ExitStalePrecondition || !strings.Contains(stderr, "to confirm") {
		t.Fatalf("exit %d, want 5\n%s%s", code, stdout, stderr)
	}
	env := errorV1(t, "target set", wholeJSON(t, stdout), apperr.StalePrecondition)
	if cur, ok := env.Error.Current["target"].(map[string]any); !ok || cur["ref"] != "refs/heads/hotfix" {
		t.Fatalf("current = %v", env.Error.Current)
	}
	got := rows(t, home, `SELECT ref, coalesce(archived_seq - ?, '-') FROM delivery_targets ORDER BY created_seq`, seq0)
	if want := [][]string{{f.Repo.Git("symbolic-ref", "HEAD"), "1"}, {"refs/heads/hotfix", "-"}}; !equalRows(got, want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
	if s := seq(t, home); s != strconv.Itoa(seq0+1) {
		t.Fatalf("seq %s, want %d", s, seq0+1)
	}
}

// MVP-TGT-03, 04, 05, 06, 08, 10 (M1): scope confirm is a person's; the
// interactive one selects every active must requirement by default and
// --requirements changes the selection; the confirmation is bound to the
// current version and source set hash, with the count of items pending or
// in conflict, the waivers in effect (none in M1) and the confirmation it
// replaces, all shown in the summary and stored; scope show reports what
// was confirmed and when it goes stale. Refused, and nothing written: the
// Agent channel and a declined confirmation (exit 3); an empty or unknown
// requirement, a secret in the note or the file, incomplete_docs without
// missing_goals_reviewed (exit 2); a stale version, source set hash,
// pending count or seen confirmation, or acknowledged waivers that are not
// exactly those in effect (exit 5, with the current value).
func TestScopeConfirm(t *testing.T) {
	home, f := scopeProject(t)
	dir := f.Repo.Dir
	ts := targetScope(t, dir)
	ci := ts.ConfirmInputs
	if ci.BaselineVersion != 2 || ci.PendingItems != 2 || !slices.Equal(ci.MustRequirements, []string{"REQ-001", "REQ-003"}) ||
		len(ci.EffectiveWaivers) != 0 || ts.Scope != nil {
		t.Fatalf("confirm_inputs = %+v", ci)
	}
	code, stdout, _ := specmetry(t, dir, "", "scope", "show")
	if code != apperr.ExitOK || !strings.Contains(stdout, "baseline version 2, source set hash "+ci.SourceSetHash) ||
		!strings.Contains(stdout, "2 proposal items pending or in conflict") || !strings.Contains(stdout, "(the default selection): REQ-001, REQ-003") ||
		!strings.Contains(stdout, "waivers in effect: none") {
		t.Fatalf("scope show:\n%s", stdout)
	}

	before := scopeState(t, home)
	code, stdout, stderr := specmetry(t, dir, "error.v1", "scope", "confirm", "--json")
	if code != apperr.ExitHumanRequired || strings.Contains(stdout+stderr, "to confirm") {
		t.Fatalf("Agent: exit %d\n%s%s", code, stdout, stderr)
	}
	for _, answer := range []string{"", "v1", "2", noPrompt} {
		code, stdout, stderr := asHuman(t, dir, answer, "scope", "confirm")
		if code != apperr.ExitHumanRequired || (answer != noPrompt && !strings.Contains(stdout, "REQ-001 (revision 1, must) sign up")) {
			t.Fatalf("answered %q: exit %d\n%s%s", answer, code, stdout, stderr)
		}
	}
	if after := scopeState(t, home); after != before {
		t.Fatalf("written: %s -> %s", before, after)
	}

	// The default: every must requirement, REQ-002 (should) not.
	code, stdout, stderr = asHuman(t, dir, "v2", "scope", "confirm", "-m", "first scope")
	if code != apperr.ExitOK || !strings.Contains(stdout, "the scope, 2 requirements with all their criteria:") ||
		!strings.Contains(stdout, "REQ-003 (revision 1, must) audit log of sign-ins") || strings.Contains(stdout, "REQ-002") ||
		!strings.Contains(stdout, "bound to baseline version 2 and source set hash "+ci.SourceSetHash) ||
		!strings.Contains(stdout, "2 proposal items pending or in conflict are not in it") || !strings.Contains(stdout, "waivers kept: none") ||
		!strings.Contains(stdout, "the target's first scope confirmation") || !strings.Contains(stdout, "note: first scope") ||
		!strings.Contains(stdout, "confirmed for refs/heads/") {
		t.Fatalf("scope confirm: exit %d\n%s%s", code, stdout, stderr)
	}
	seq0, _ := strconv.Atoi(strings.Fields(before)[0])
	c1 := targetScope(t, dir)
	if c1.scoped() != "REQ-001@1 REQ-003@1" || c1.Scope.BaselineVersion != 2 || c1.Scope.SourceSetHash != ci.SourceSetHash || c1.Scope.Stale ||
		c1.Scope.Seq != int64(seq0+1) || len(c1.Scope.AcknowledgedWaivers) != 0 || c1.Scope.Attestations["complete_scope"] != true ||
		c1.Scope.Attestations["pending_items_excluded"] != float64(2) {
		t.Fatalf("scope = %+v", c1.Scope)
	}
	got := rows(t, home, `SELECT c.target_id, c.baseline_version, c.attestations_json, c.acknowledged_waivers_json,
    coalesce(c.seen_scope_confirmation_id, '-'), c.note, c.actor LIKE 'human:_%', c.channel FROM scope_confirmations c`)
	if want := [][]string{{c1.Target.ID, "2", `{"complete_scope":true,"pending_items_excluded":2}`, "[]", "-", "first scope", "1", "human_cli"}}; !equalRows(got, want) {
		t.Fatalf("stored = %v, want %v", got, want)
	}

	// Changed: REQ-002 alone; it replaces the first.
	code, stdout, stderr = asHuman(t, dir, "v2", "scope", "confirm", "--requirements", "REQ-002")
	if code != apperr.ExitOK || !strings.Contains(stdout, "REQ-002 (revision 1, should) export") || strings.Contains(stdout, "REQ-001 (") ||
		!strings.Contains(stdout, "it replaces scope "+c1.Scope.ConfirmationID) {
		t.Fatalf("scope confirm --requirements: exit %d\n%s%s", code, stdout, stderr)
	}
	c2 := targetScope(t, dir)
	if c2.scoped() != "REQ-002@1" {
		t.Fatalf("scope = %s", c2.scoped())
	}
	if got := rows(t, home, `SELECT seen_scope_confirmation_id FROM scope_confirmations WHERE id = ?`, c2.Scope.ConfirmationID); got[0][0] != c1.Scope.ConfirmationID {
		t.Fatalf("seen = %v", got)
	}

	secret := testutil.NewSecretCanary(t)
	for _, c := range []struct {
		code    int
		pointer string
		hidden  string
		args    []string
	}{
		{apperr.ExitInvalidInput, "/requirements", "", []string{"scope", "confirm", "--requirements", ","}},
		{apperr.ExitInvalidInput, "/requirements/1", "", []string{"scope", "confirm", "--requirements", "REQ-001,REQ-009"}},
		{apperr.ExitInvalidInput, "/requirements/1", "", []string{"scope", "confirm", "--requirements", "REQ-001,REQ-001"}},
		{apperr.ExitInvalidInput, "-m", secret, []string{"scope", "confirm", "-m", "see " + secret}},
		{apperr.ExitInvalidInput, "", "", []string{"scope", "confirm", "--file", scopeFile(t, c2, []string{"REQ-001"}, nil), "-m", "x"}},
		{apperr.ExitInvalidInput, "/requirements", "", []string{"scope", "confirm", "--file", scopeFile(t, c2, []string{}, nil)}},
		{apperr.ExitInvalidInput, "/note", secret, []string{"scope", "confirm", "--file", scopeFile(t, c2, []string{"REQ-001"},
			func(d map[string]any) { d["note"] = secret })}},
		{apperr.ExitInvalidInput, "/attestations/complete_scope", "", []string{"scope", "confirm", "--file", scopeFile(t, c2, []string{"REQ-001"},
			func(d map[string]any) { d["attestations"].(map[string]any)["complete_scope"] = false })}},
		{apperr.ExitStalePrecondition, "/baseline_version", "", []string{"scope", "confirm", "--file", scopeFile(t, c2, []string{"REQ-001"},
			func(d map[string]any) { d["baseline_version"] = 1 })}},
		{apperr.ExitStalePrecondition, "/attestations/pending_items_excluded", "", []string{"scope", "confirm", "--file", scopeFile(t, c2, []string{"REQ-001"},
			func(d map[string]any) { d["attestations"].(map[string]any)["pending_items_excluded"] = 1 })}},
		{apperr.ExitStalePrecondition, "/seen_scope_confirmation_id", "", []string{"scope", "confirm", "--file", scopeFile(t, c2, []string{"REQ-001"},
			func(d map[string]any) { d["seen_scope_confirmation_id"] = c1.Scope.ConfirmationID })}},
		{apperr.ExitStalePrecondition, "/acknowledged_waivers", "", []string{"scope", "confirm", "--file", scopeFile(t, c2, []string{"REQ-001"},
			func(d map[string]any) { d["acknowledged_waivers"] = []any{"01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"} })}},
	} {
		env := humanRefused(t, home, dir, "v2", c.code, c.pointer, c.hidden, c.args...)
		if c.pointer == "/acknowledged_waivers" {
			// V-T01: the waivers in effect, none.
			if w, ok := env.Error.Current["effective_waivers"].([]any); !ok || len(w) != 0 {
				t.Errorf("current = %v", env.Error.Current)
			}
		}
	}

	// The source set changes: the confirmation is stale, and a document
	// read before is exit 5 with the current hash.
	stale := scopeFile(t, c2, []string{"REQ-001"}, nil)
	if err := os.WriteFile(filepath.Join(dir, "docs", "extra.md"), []byte("# extra\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sourceAdd(t, dir, "docs/extra.md")
	now := targetScope(t, dir)
	if !now.Scope.Stale || now.ConfirmInputs.SourceSetHash == c2.ConfirmInputs.SourceSetHash {
		t.Fatalf("after a new source: %+v", now.Scope)
	}
	env := humanRefused(t, home, dir, "v2", apperr.ExitStalePrecondition, "/source_set_hash", "", "scope", "confirm", "--file", stale)
	if env.Error.Current["source_set_hash"] != now.ConfirmInputs.SourceSetHash {
		t.Fatalf("current = %v", env.Error.Current)
	}
	if code, stdout, stderr := asHuman(t, dir, "v2", "scope", "confirm", "--file", scopeFile(t, now, []string{"REQ-001", "REQ-002"}, nil), "--json"); code != apperr.ExitOK {
		t.Fatalf("scope confirm --file: exit %d\n%s%s", code, stdout, stderr)
	}
	if got := targetScope(t, dir); got.scoped() != "REQ-001@1 REQ-002@1" || got.Scope.Stale {
		t.Fatalf("scope = %s, stale %v", got.scoped(), got.Scope.Stale)
	}

	// incomplete_docs: missing_goals_reviewed must be true.
	mustInit(t, dir, "--stage", "incomplete_docs")
	humanRefused(t, home, dir, "v2", apperr.ExitInvalidInput, "/attestations/missing_goals_reviewed", "", "scope", "confirm")
	humanRefused(t, home, dir, "v2", apperr.ExitInvalidInput, "/attestations/missing_goals_reviewed", "", "scope", "confirm", "--file",
		scopeFile(t, targetScope(t, dir), []string{"REQ-001"}, func(d map[string]any) { d["attestations"].(map[string]any)["missing_goals_reviewed"] = false }))
	code, stdout, stderr = asHuman(t, dir, "v2", "scope", "confirm", "--missing-goals-reviewed")
	if code != apperr.ExitOK || !strings.Contains(stdout, "the goals the code cannot reveal have been added (stage incomplete_docs)") {
		t.Fatalf("--missing-goals-reviewed: exit %d\n%s%s", code, stdout, stderr)
	}
	if got := targetScope(t, dir); got.Scope.Attestations["missing_goals_reviewed"] != true {
		t.Fatalf("attestations = %v", got.Scope.Attestations)
	}
	if n := count(t, home, `SELECT count(*) FROM scope_confirmations`); n != 4 {
		t.Fatalf("%d confirmations, want 4", n)
	}
}

// MVP-TGT-11, V-T02: two sessions read the same confirmation C1; B
// confirms (C2); A's confirmation naming C1 is stale (exit 5, C2 in
// error.current) and writes nothing; one naming none, now that one exists,
// too; read again, A's naming C2 is confirmed. The interactive confirm
// checks again under the write lock: another connection confirming while
// the person is asked, in this process or with the confirming command at a
// real terminal in another process, makes it stale.
func TestScopeConfirmedMeanwhile(t *testing.T) {
	home, f := scopeProject(t)
	dir := f.Repo.Dir
	if code, stdout, stderr := asHuman(t, dir, "v2", "scope", "confirm"); code != apperr.ExitOK {
		t.Fatalf("C1: exit %d\n%s%s", code, stdout, stderr)
	}
	readA := targetScope(t, dir)
	c1 := readA.Scope.ConfirmationID
	if code, stdout, stderr := asHuman(t, dir, "v2", "scope", "confirm", "--file", scopeFile(t, targetScope(t, dir), []string{"REQ-001"}, nil)); code != apperr.ExitOK {
		t.Fatalf("C2: exit %d\n%s%s", code, stdout, stderr)
	}
	c2 := targetScope(t, dir).Scope.ConfirmationID
	env := humanRefused(t, home, dir, "v2", apperr.ExitStalePrecondition, "/seen_scope_confirmation_id", "", "scope", "confirm", "--file",
		scopeFile(t, readA, []string{"REQ-003"}, nil))
	if env.Error.Current["scope_confirmation_id"] != c2 || c2 == c1 {
		t.Fatalf("current = %v, want %s", env.Error.Current, c2)
	}
	humanRefused(t, home, dir, "v2", apperr.ExitStalePrecondition, "/seen_scope_confirmation_id", "", "scope", "confirm", "--file",
		scopeFile(t, readA, []string{"REQ-003"}, func(d map[string]any) { d["seen_scope_confirmation_id"] = nil }))
	if code, stdout, stderr := asHuman(t, dir, "v2", "scope", "confirm", "--file", scopeFile(t, targetScope(t, dir), []string{"REQ-003"}, nil)); code != apperr.ExitOK {
		t.Fatalf("read again: exit %d\n%s%s", code, stdout, stderr)
	}
	c3 := targetScope(t, dir)
	if c3.scoped() != "REQ-003@1" {
		t.Fatalf("scope = %s", c3.scoped())
	}

	var other int
	var otherOut string
	in := &onFirstRead{r: strings.NewReader("v2\n"), before: func() {
		other, otherOut = confirmIn("v2", "scope", "confirm", "--requirements", "REQ-002")
	}}
	code, stdout, stderr := asHumanWith(t, dir, in, "scope", "confirm", "--json")
	if other != apperr.ExitOK {
		t.Fatalf("the other connection: exit %d\n%s", other, otherOut)
	}
	if code != apperr.ExitStalePrecondition || !strings.Contains(stderr, "to confirm") {
		t.Fatalf("exit %d, want 5\n%s%s", code, stdout, stderr)
	}
	c4 := targetScope(t, dir)
	env = errorV1(t, "scope confirm", wholeJSON(t, stdout), apperr.StalePrecondition)
	if env.Error.Current["scope_confirmation_id"] != c4.Scope.ConfirmationID || c4.scoped() != "REQ-002@1" {
		t.Fatalf("current = %v, scope %s", env.Error.Current, c4.scoped())
	}

	t.Chdir(dir)
	out := inTerminalChanging(t, dir, []string{"scope", "confirm"}, "v2", func() {
		if code, out := confirmIn("v2", "scope", "confirm", "--requirements", "REQ-001,REQ-002"); code != apperr.ExitOK {
			t.Errorf("the other process's confirmation was refused: exit %d\n%s", code, out)
		}
	})
	if !strings.Contains(out, "exit=5") {
		t.Fatalf("confirmed during the prompt:\n%s", out)
	}
	if got := targetScope(t, dir); got.scoped() != "REQ-001@1 REQ-002@1" {
		t.Fatalf("scope = %s", got.scoped())
	}
	got := rows(t, home, `SELECT count(*), count(DISTINCT seq), count(seen_scope_confirmation_id) FROM scope_confirmations`)
	if want := [][]string{{"5", "5", "4"}}; !equalRows(got, want) {
		t.Fatalf("confirmations = %v, want %v", got, want)
	}
}

// MVP-CH-01, CH-02 with target set and scope confirm: at a real terminal
// with CLAUDECODE or CLAUDE_CODE_ENTRYPOINT set to a canary, exit 3
// before any prompt, and the canary is in neither the output nor the data
// directory; without it, a declined confirmation writes nothing and a
// confirmed one is written. stdout redirected or stdin not a terminal is
// agent_cli, with or without --json: exit 3. target show and scope show
// work on either channel.
func TestTargetAndScopeNeedAPerson(t *testing.T) {
	home, f := scopeProject(t)
	dir := f.Repo.Dir
	canary := testutil.NewCanary(t)
	commands := []struct {
		args    []string
		confirm string
	}{
		{[]string{"scope", "confirm"}, "v2"},
		{[]string{"target", "set", "--ref", "refs/heads/release"}, "refs/heads/release"},
	}
	before := scopeState(t, home)
	for _, c := range commands {
		for _, name := range []string{"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT"} {
			out := inTerminal(t, dir, c.args, noPrompt, name+"="+canary)
			if !strings.Contains(out, "exit=3") || strings.Contains(out, "to confirm") || strings.Contains(out, canary) {
				t.Errorf("%v with %s:\n%s", c.args, name, out)
			}
		}
		if out := inTerminal(t, dir, c.args, ""); !strings.Contains(out, "exit=3") || !strings.Contains(out, "to confirm") {
			t.Errorf("%v declined:\n%s", c.args, out)
		}
		for _, args := range [][]string{c.args, append(slices.Clone(c.args), "--json")} {
			raw, err := json.Marshal(args)
			if err != nil {
				t.Fatal(err)
			}
			redirected := testutil.HelperCommand(t, "cli-redirected")
			out := testutil.RunInPTYSession(t, 60*time.Second, helperEnv(t, "cli-redirected", argsJSONEnv+"="+string(raw), dirEnv+"="+dir), nil,
				redirected.Path, redirected.Args[1:]...).Output
			if !strings.Contains(out, "exit=3") || strings.Contains(out, "to confirm") {
				t.Errorf("%v, stdout redirected:\n%s", args, out)
			}
			piped := testutil.HelperCommand(t, "cli")
			piped.Env = helperEnv(t, "cli", argsJSONEnv+"="+string(raw), dirEnv+"="+dir)
			if b, _ := piped.CombinedOutput(); !strings.Contains(string(b), "exit=3") || strings.Contains(string(b), "to confirm") {
				t.Errorf("%v, stdin not a terminal:\n%s", args, b)
			}
		}
	}
	if after := scopeState(t, home); after != before {
		t.Fatalf("written: %s -> %s", before, after)
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the value reached the data directory: %v", hits)
	}
	for _, c := range commands {
		if out := inTerminal(t, dir, c.args, c.confirm); !strings.Contains(out, "exit=0") || !strings.Contains(out, "to confirm") {
			t.Fatalf("%v confirmed:\n%s", c.args, out)
		}
	}
	got := rows(t, home, `SELECT (SELECT count(*) FROM scope_confirmations), (SELECT ref FROM delivery_targets WHERE archived_seq IS NULL)`)
	if want := [][]string{{"1", "refs/heads/release"}}; !equalRows(got, want) {
		t.Fatalf("after the confirmations: %v, want %v", got, want)
	}
	for _, args := range [][]string{{"target", "show", "--json"}, {"scope", "show", "--json"}} {
		raw, _ := json.Marshal(args)
		piped := testutil.HelperCommand(t, "cli")
		piped.Env = helperEnv(t, "cli", argsJSONEnv+"="+string(raw), dirEnv+"="+dir)
		if b, _ := piped.CombinedOutput(); !strings.Contains(string(b), "exit=0") || !strings.Contains(string(b), `"refs/heads/release"`) {
			t.Fatalf("%v in agent_cli:\n%s", args, b)
		}
	}
}

// codex-review-031 B1 for target set and scope confirm: a confirmation
// that cannot be shown whole (stdout takes half of each write, without an
// error) is exit 6, reads nothing and writes nothing.
func TestTargetAndScopeConfirmationNotShownWhole(t *testing.T) {
	home, f := scopeProject(t)
	dir := f.Repo.Dir
	t.Chdir(dir)
	before := scopeState(t, home)
	for _, c := range [][]string{{"scope", "confirm"}, {"target", "set", "--ref", "refs/heads/release"}} {
		out := &halfWriter{}
		var errOut bytes.Buffer
		in := &readCounter{r: strings.NewReader("v2\nrefs/heads/release\n")}
		code := runAs(atTerminal, out, &errOut, in, c...)
		if code != apperr.ExitEnvironment || in.reads != 0 || !strings.Contains(errOut.String(), "cannot show the confirmation") || out.got.Len() == 0 {
			t.Errorf("%v: exit %d after %d reads, want 6 and none\n%s", c, code, in.reads, errOut.String())
		}
	}
	if after := scopeState(t, home); after != before {
		t.Fatalf("written: %s -> %s", before, after)
	}
}
