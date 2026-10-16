package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

func editDoc(base int, items ...any) map[string]any {
	doc := small(base, items...)
	doc["origin"] = "human_edit"
	doc["summary"] = "a person's edit"
	return doc
}

// applyEdit runs propose --apply --json as a person who confirms with the
// next version, and returns the receipt.
func applyEdit(t *testing.T, dir string, doc map[string]any) receiptOut {
	t.Helper()
	confirm := "v" + string(rune('0'+doc["base_version"].(int)+1))
	code, stdout, stderr := asHuman(t, dir, confirm, "propose", "--apply", "--json", "--file", writeProposal(t, doc))
	if code != apperr.ExitOK {
		t.Fatalf("propose --apply: exit %d\n%s%s", code, stdout, stderr)
	}
	val, err := validator()
	if err != nil {
		t.Fatal(err)
	}
	body := wholeJSON(t, stdout)
	if err := val.ValidateJSON("proposal-receipt.v1", []byte(body)); err != nil {
		t.Fatalf("receipt does not conform: %v\n%s", err, body)
	}
	var r receiptOut
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

// MVP-BL-13: in a terminal, propose --apply stores a human_edit proposal
// and applies it in the same transaction, through the same mechanism as a
// review: the receipt reports the version and the keys, every item is
// accepted, and the version records the person. MVP-BL-23: an update
// without change_class is refused; so is another origin, and a person who
// does not confirm writes nothing.
func TestProposeApply(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	doc := editDoc(0, addReq("r", "sign up", nil), addAC("c", "r", "a new account can sign in"))
	doc["questions"] = []any{map[string]any{"text": "e-mail or phone?", "about": "r", "blocking": false}}
	r := applyEdit(t, dir, doc)
	if r.AppliedVersion == nil || *r.AppliedVersion != 1 || r.Origin != "human_edit" {
		t.Fatalf("receipt = %+v", r)
	}
	var keys []string
	for _, it := range r.Items {
		if it.ReviewState != "accepted" || it.AssignedKey == nil {
			t.Fatalf("item = %+v", it)
		}
		keys = append(keys, *it.AssignedKey)
	}
	if strings.Join(keys, " ") != "REQ-001 AC-001-1" || r.Questions[0].About == nil || *r.Questions[0].About != "REQ-001" {
		t.Fatalf("keys %v, questions %+v", keys, r.Questions)
	}
	got := rows(t, home, `SELECT p.origin, p.channel, v.number, v.channel, substr(v.actor, 1, 6), p.seq = v.seq FROM proposals p
JOIN proposal_items i ON i.proposal_id = p.id JOIN baseline_versions v ON v.id = i.applied_version_id GROUP BY p.id`)
	if want := [][]string{{"human_edit", "human_cli", "1", "human_cli", "human:", "1"}}; !equalRows(got, want) {
		t.Fatalf("stored %v, want %v", got, want)
	}
	if got := baselineShow(t, dir).keys(); got != "REQ-001 AC-001-1" {
		t.Fatalf("baseline = %s", got)
	}

	material := update("requirement", "REQ-001", 1, map[string]any{"title": "sign up with e-mail"})
	material["change_class"] = "material"
	if r := applyEdit(t, dir, editDoc(1, material)); *r.AppliedVersion != 2 {
		t.Fatalf("receipt = %+v", r)
	}
	if b := baselineShow(t, dir); b.Requirements[0].Revision != 2 || b.Requirements[0].Criteria[0].RequirementRevision != 1 {
		t.Fatalf("baseline = %+v", b.Requirements[0])
	}

	before := state(t, home)
	for _, c := range []struct {
		doc     map[string]any
		pointer string
	}{
		{editDoc(2, update("requirement", "REQ-001", 2, map[string]any{"title": "x"})), "/items/0/change_class"},
		{func() map[string]any { d := editDoc(2, addReq("x", "x", nil)); d["origin"] = "cli"; return d }(), "/origin"},
		{editDoc(2), "/items"},
	} {
		code, stdout, _ := asHuman(t, dir, "v3", "propose", "--apply", "--json", "--file", writeProposal(t, c.doc))
		var env apperr.Envelope
		_ = json.Unmarshal([]byte(wholeJSON(t, stdout)), &env)
		found := false
		for _, d := range env.Error.Details {
			found = found || d.Pointer == c.pointer
		}
		if code != apperr.ExitInvalidInput || !found {
			t.Errorf("%s: exit %d\n%s", c.pointer, code, stdout)
		}
	}
	code, stdout, stderr := asHuman(t, dir, "", "propose", "--apply", "--file", writeProposal(t, editDoc(2, addReq("x", "x", nil))))
	if code != apperr.ExitHumanRequired {
		t.Errorf("not confirmed: exit %d\n%s%s", code, stdout, stderr)
	}
	stale := material
	stale["target_revision"] = 1
	code, stdout, _ = asHuman(t, dir, "v2", "propose", "--apply", "--json", "--file", writeProposal(t, editDoc(1, stale)))
	if code != apperr.ExitStalePrecondition {
		t.Errorf("stale base_version: exit %d\n%s", code, stdout)
	}
	if after := state(t, home); after != before {
		t.Fatalf("written %s -> %s", before, after)
	}
}

// proposal list: newest first; --pending keeps the proposals with items
// still to decide.
func TestProposalList(t *testing.T) {
	_, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	first := proposeOK(t, dir, small(0, addReq("a", "a", nil)))
	second := proposeOK(t, dir, small(0, addReq("b", "b", nil)))
	acceptAll(t, dir, first, 0, nil)
	list := func(args ...string) []string {
		code, stdout, stderr := specmetry(t, dir, "proposal-list.v1", append([]string{"proposal", "list", "--json"}, args...)...)
		if code != apperr.ExitOK {
			t.Fatalf("proposal list: exit %d\n%s%s", code, stdout, stderr)
		}
		var doc struct {
			BaselineVersion int `json:"baseline_version"`
			Proposals       []struct {
				ID     string         `json:"id"`
				Counts map[string]int `json:"counts"`
			} `json:"proposals"`
		}
		if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, p := range doc.Proposals {
			out = append(out, p.ID)
		}
		return out
	}
	if got := list(); len(got) != 2 || got[0] != second.ProposalID || got[1] != first.ProposalID {
		t.Fatalf("list = %v", got)
	}
	if got := list("--pending"); len(got) != 1 || got[0] != second.ProposalID {
		t.Fatalf("pending = %v", got)
	}
	if code, out, _ := specmetry(t, dir, "", "proposal", "list"); code != apperr.ExitOK || !strings.Contains(out, second.ProposalID) {
		t.Fatalf("text list: exit %d\n%s", code, out)
	}
	if code, out, _ := specmetry(t, dir, "", "proposal", "show", first.ProposalID); code != apperr.ExitOK || !strings.Contains(out, "accepted") {
		t.Fatalf("text show: exit %d\n%s", code, out)
	}
	for _, args := range [][]string{{"baseline", "show"}, {"baseline", "history"}} {
		if code, out, _ := specmetry(t, dir, "", args...); code != apperr.ExitOK || !strings.Contains(out, "REQ-001") {
			t.Fatalf("%v: exit %d\n%s", args, code, out)
		}
	}
}
