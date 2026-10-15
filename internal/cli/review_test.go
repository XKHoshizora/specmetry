package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// MVP-BL-06: five new items, three accepted and two rejected, make version
// 1 holding exactly the three; keys are given in order. The rejected items
// enter no baseline.
func TestReviewAcceptsSomeItems(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	r := proposeOK(t, dir, small(0, addGoal("g1", "launch"), addReq("r1", "sign up", "g1"), addReq("r2", "export", "g1"),
		addAC("c1", "r1", "a new account can sign in"), addAC("c2", "r2", "an export is a CSV file")))
	ids := itemIDs(r)
	out := reviewOK(t, dir, reviewDoc(r.ProposalID, 0,
		decide(ids["g1"], "accepted"), decide(ids["r1"], "accepted"), decide(ids["c1"], "accepted"),
		decide(ids["r2"], "rejected", "note", "later"), decide(ids["c2"], "rejected")))
	if !strings.Contains(out, "Baseline version 1 created") {
		t.Fatalf("output:\n%s", out)
	}
	if got := baselineShow(t, dir).keys(); got != "GOAL-1 REQ-001 AC-001-1" {
		t.Fatalf("baseline = %s", got)
	}
	if n := count(t, home, `SELECT count(*) FROM baseline_changes`); n != 3 {
		t.Errorf("%d changes, want 3", n)
	}
	got := map[string]string{}
	for _, it := range showProposal(t, dir, r.ProposalID).Items {
		got[it.ItemID] = it.ReviewState
	}
	want := map[string]string{ids["g1"]: "accepted", ids["r1"]: "accepted", ids["c1"]: "accepted", ids["r2"]: "rejected", ids["c2"]: "rejected"}
	for id, state := range want {
		if got[id] != state {
			t.Errorf("item %s is %s, want %s", id, got[id], state)
		}
	}
}

// MVP-BL-07: rejecting a requirement of the proposal while accepting a
// criterion of it fails the whole review, and nothing is written; the same
// for a criterion whose requirement is left undecided.
func TestReviewNeedsTheParentAccepted(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	r := proposeOK(t, dir, small(0, addReq("r1", "sign up", nil), addAC("c1", "r1", "a new account can sign in")))
	ids := itemIDs(r)
	reviewRefused(t, home, dir, reviewDoc(r.ProposalID, 0, decide(ids["r1"], "rejected"), decide(ids["c1"], "accepted")),
		apperr.ExitInvalidInput, "/items/1")
	reviewRefused(t, home, dir, reviewDoc(r.ProposalID, 0, decide(ids["c1"], "accepted")), apperr.ExitInvalidInput, "/items/0")
	// The parent accepted by an earlier review is in the baseline.
	reviewOK(t, dir, reviewDoc(r.ProposalID, 0, decide(ids["r1"], "accepted")))
	reviewOK(t, dir, reviewDoc(r.ProposalID, 1, decide(ids["c1"], "accepted")))
	if got := baselineShow(t, dir).keys(); got != "REQ-001 AC-001-1" {
		t.Fatalf("baseline = %s", got)
	}
}

// MVP-BL-08: accepted with edits keeps the proposed and the final values;
// the baseline takes the final ones.
func TestReviewAcceptsWithEdits(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	r := proposeOK(t, dir, small(0, addReq("r1", "sign up", nil)))
	ids := itemIDs(r)
	reviewOK(t, dir, reviewDoc(r.ProposalID, 0, decide(ids["r1"], "accepted_with_edits",
		"final_data", map[string]any{"title": "sign up with e-mail", "kind": "functional", "priority": "should"})))
	got := rows(t, home, `SELECT review_state, json_extract(payload_json, '$.title'), json_extract(final_payload_json, '$.title'),
    json_extract(final_payload_json, '$.priority') FROM proposal_items`)
	if want := [][]string{{"accepted_with_edits", "sign up", "sign up with e-mail", "should"}}; !equalRows(got, want) {
		t.Fatalf("item = %v, want %v", got, want)
	}
	b := baselineShow(t, dir)
	if len(b.Requirements) != 1 || b.Requirements[0].Title != "sign up with e-mail" {
		t.Fatalf("baseline = %+v", b.Requirements)
	}
	show := showProposal(t, dir, r.ProposalID)
	if rv := show.Items[0].Review; rv == nil || rv.Final["title"] != "sign up with e-mail" || show.Items[0].Proposed["title"] != "sign up" {
		t.Fatalf("proposal show = %+v", show.Items[0])
	}
}

func equalRows(a, b [][]string) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// MVP-BL-10: a stale seen_base_version is exit 5 with the current version,
// and nothing is written.
func TestReviewRefusesAStaleVersion(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	first := proposeOK(t, dir, small(0, addReq("r1", "sign up", nil)))
	second := proposeOK(t, dir, small(0, addReq("r1", "export", nil)))
	acceptAll(t, dir, first, 0, nil)
	env := reviewRefused(t, home, dir, reviewDoc(second.ProposalID, 0, decide(itemIDs(second)["r1"], "accepted")),
		apperr.ExitStalePrecondition, "/seen_base_version")
	if v, _ := env.Error.Current["baseline_version"].(float64); v != 1 {
		t.Fatalf("current = %v", env.Error.Current)
	}
	reviewOK(t, dir, reviewDoc(second.ProposalID, 1, decide(itemIDs(second)["r1"], "accepted")))
}

// MVP-BL-15: a retired key is never handed out again; criteria are numbered
// within their requirement.
func TestReviewNeverReusesKeys(t *testing.T) {
	_, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	r := proposeOK(t, dir, small(0, addReq("a", "a", nil), addReq("b", "b", nil), addReq("c", "c", nil),
		addAC("c1", "c", "first"), addAC("c2", "c", "second")))
	acceptAll(t, dir, r, 0, nil)
	if got := baselineShow(t, dir).keys(); got != "REQ-001 REQ-002 REQ-003 AC-003-1 AC-003-2" {
		t.Fatalf("baseline = %s", got)
	}
	acceptAll(t, dir, proposeOK(t, dir, small(1, retire("requirement", "REQ-003", 1))), 1, nil)
	acceptAll(t, dir, proposeOK(t, dir, small(2, addReq("d", "d", nil), addAC("d1", "d", "first"), addAC("a3", "REQ-001", "third"))), 2, nil)
	if got := baselineShow(t, dir, "--include-retired").keys(); got != "REQ-001 AC-001-1 REQ-002 REQ-003* AC-003-1* AC-003-2* REQ-004 AC-004-1" {
		t.Fatalf("baseline = %s", got)
	}
}

// MVP-BL-16: retiring a requirement with two criteria retires both in the
// same version, each change recorded; MVP-BL-11: after three versions
// baseline history lists every change with its before and after, actor,
// channel and item.
func TestReviewRetiresCriteriaWithTheirRequirement(t *testing.T) {
	_, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	acceptAll(t, dir, proposeOK(t, dir, small(0, addReq("r", "sign up", nil), addAC("c1", "r", "first"), addAC("c2", "r", "second"))), 0, nil)
	acceptAll(t, dir, proposeOK(t, dir, small(1, update("requirement", "REQ-001", 1, map[string]any{"title": "sign up with e-mail"}))),
		1, map[string]string{"REQ-001": "material"})
	retirement := proposeOK(t, dir, small(2, retire("requirement", "REQ-001", 2)))
	acceptAll(t, dir, retirement, 2, nil)

	code, stdout, stderr := specmetry(t, dir, "baseline-history.v1", "baseline", "history", "--json")
	if code != apperr.ExitOK {
		t.Fatalf("history: exit %d\n%s%s", code, stdout, stderr)
	}
	var h struct {
		BaselineVersion int `json:"baseline_version"`
		Versions        []struct {
			Number     int     `json:"number"`
			Actor      string  `json:"actor"`
			Channel    string  `json:"channel"`
			ProposalID *string `json:"proposal_id"`
			Changes    []struct {
				Key            string  `json:"key"`
				Op             string  `json:"op"`
				ChangeClass    *string `json:"change_class"`
				ProposalItemID *string `json:"proposal_item_id"`
				Before         *struct {
					Revision int            `json:"revision"`
					Status   string         `json:"status"`
					Values   map[string]any `json:"values"`
				} `json:"before"`
				After struct {
					Revision int            `json:"revision"`
					Status   string         `json:"status"`
					Values   map[string]any `json:"values"`
				} `json:"after"`
			} `json:"changes"`
		} `json:"versions"`
	}
	if err := json.Unmarshal([]byte(stdout), &h); err != nil {
		t.Fatal(err)
	}
	if h.BaselineVersion != 3 || len(h.Versions) != 3 {
		t.Fatalf("history = %+v", h)
	}
	var lines []string
	for _, v := range h.Versions {
		if !strings.HasPrefix(v.Actor, "human:") || v.Channel != "human_cli" || v.ProposalID == nil {
			t.Errorf("version %d: actor %s channel %s proposal %v", v.Number, v.Actor, v.Channel, v.ProposalID)
		}
		for _, c := range v.Changes {
			line := c.Op + " " + c.Key
			if c.ChangeClass != nil {
				line += " " + *c.ChangeClass
			}
			if c.ProposalItemID == nil {
				line += " cascade"
			}
			if c.Before != nil {
				line += " " + c.Before.Status
			}
			line += " -> " + c.After.Status
			lines = append(lines, line)
		}
	}
	want := "add REQ-001 -> active|add AC-001-1 -> active|add AC-001-2 -> active|update REQ-001 material active -> active|" +
		"retire REQ-001 active -> retired|retire AC-001-1 cascade active -> retired|retire AC-001-2 cascade active -> retired"
	if got := strings.Join(lines, "|"); got != want {
		t.Fatalf("changes:\n%s\nwant\n%s", got, want)
	}
	upd := h.Versions[1].Changes[0]
	if upd.Before.Values["title"] != "sign up" || upd.After.Values["title"] != "sign up with e-mail" || upd.After.Revision != 2 {
		t.Fatalf("update recorded %+v -> %+v", upd.Before, upd.After)
	}
	if got := baselineShow(t, dir).keys(); got != "" {
		t.Fatalf("active baseline = %q", got)
	}
}
