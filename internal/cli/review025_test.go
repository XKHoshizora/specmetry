package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// codex-review-025 B1: proposal show and the other commands of WP1.6 never
// echo an argument they were given: not a malformed ID that matches a
// secret rule, nor a well-formed ID of no proposal, nor a bad --since; in
// text and in JSON (message, details, current), on stdout and stderr,
// nor in the data directory. Nothing is written; a real proposal still
// shows.
func TestProposalShowDoesNotEchoItsArgument(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	r := proposeOK(t, dir, small(0, addReq("r", "sign up", nil)))
	token := "ghp_" + strings.Repeat("A1b2", 9) // a GitHub token to the secret rules
	missing := "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"     // a ULID, of no proposal
	before := state(t, home)
	cases := [][]string{
		{"proposal", "show", token},
		{"proposal", "show", missing},
		{"baseline", "history", "--since", token},
	}
	for _, args := range cases {
		arg := args[len(args)-1]
		for _, asJSON := range []bool{false, true} {
			run := slicesWith(args, asJSON)
			code, stdout, stderr := specmetry(t, dir, "", run...)
			if code != apperr.ExitInvalidInput {
				t.Errorf("%v: exit %d\n%s%s", run, code, stdout, stderr)
			}
			if strings.Contains(stdout+stderr, arg) {
				t.Errorf("%v echoes its argument:\n%s%s", run, stdout, stderr)
			}
			if asJSON {
				var env apperr.Envelope
				if err := json.Unmarshal([]byte(stdout), &env); err != nil {
					t.Errorf("%v: no error.v1: %v\n%s", run, err, stdout)
				}
				if b, _ := json.Marshal(env.Error); bytes.Contains(b, []byte(arg)) {
					t.Errorf("%v: error.v1 holds the argument: %s", run, b)
				}
			}
		}
	}
	if after := state(t, home); after != before {
		t.Errorf("written: %s -> %s", before, after)
	}
	if hits := testutil.FindInTree(t, home.Path, token); len(hits) != 0 {
		t.Errorf("the argument reached the data directory: %v", hits)
	}
	if doc := showProposal(t, dir, r.ProposalID); len(doc.Items) != 1 {
		t.Fatalf("proposal show = %+v", doc)
	}
}

func slicesWith(args []string, asJSON bool) []string {
	out := append([]string{}, args...)
	if asJSON {
		out = append(out, "--json")
	}
	return out
}

// staleReview is a project at version 2 after a first proposal changed
// REQ-001, with a second proposal on REQ-001 made at version 1.
func staleReview(t *testing.T) (home datadir.Dir, dir string, first, second receiptOut) {
	t.Helper()
	home, f, _ := proposalsProject(t)
	dir = f.Repo.Dir
	acceptAll(t, dir, proposeOK(t, dir, small(0, addReq("r", "sign up", nil))), 0, nil)
	first = proposeOK(t, dir, small(1, update("requirement", "REQ-001", 1, map[string]any{"title": "sign up with e-mail"})))
	second = proposeOK(t, dir, small(1, update("requirement", "REQ-001", 1, map[string]any{"priority": "should"})))
	return home, dir, first, second
}

// codex-review-025 B2, counterexample A: a review read at version 1 of a
// target another review changed (version 2) is stale (exit 5, the current
// version, at /seen_base_version), not an unacknowledged conflict (exit
// 2), with or without acknowledged_conflict; no prompt, nothing written.
// At the current version the unacknowledged conflict is still exit 2.
func TestReviewIsStaleBeforeTheConflict(t *testing.T) {
	h, dir, first, second := staleReview(t)
	acceptAll(t, dir, first, 1, nil)
	for _, ack := range []bool{false, true} {
		d := decide(second.Items[0].ItemID, "accepted", "change_class", "editorial")
		if ack {
			d["acknowledged_conflict"] = true
		}
		env := reviewRefused(t, h, dir, reviewDoc(second.ProposalID, 1, d), apperr.ExitStalePrecondition, "/seen_base_version")
		if v, _ := env.Error.Current["baseline_version"].(float64); v != 2 {
			t.Errorf("acknowledged %v: current = %v", ack, env.Error.Current)
		}
	}
	reviewRefused(t, h, dir, reviewDoc(second.ProposalID, 2, decide(second.Items[0].ItemID, "accepted", "change_class", "editorial")),
		apperr.ExitInvalidInput, "/items/0/acknowledged_conflict")
}

// onFirstRead runs before when the confirmation is first read: what
// happens while the person reads the prompt.
type onFirstRead struct {
	before func()
	done   bool
	r      io.Reader
}

func (o *onFirstRead) Read(p []byte) (int, error) {
	if !o.done {
		o.done = true
		o.before()
	}
	return o.r.Read(p)
}

// codex-review-025 B2, counterexample B: the second review passes its
// checks and its dry run and waits for the person; meanwhile another
// connection applies the first. Confirmed, the second is stale (exit 5,
// version 2), not a conflict (exit 2); it is not applied and changes
// nothing the first wrote: versions, entities, keys, seq, items and
// references are as the first left them.
func TestReviewIsStaleAfterTheConfirmation(t *testing.T) {
	h, dir, first, second := staleReview(t)
	var afterFirst string
	in := &onFirstRead{r: strings.NewReader(second.ProposalID + "\n"), before: func() {
		// Another process, another connection: the in-process CLI opens
		// its own.
		reviewOK(t, dir, reviewDoc(first.ProposalID, 1, decide(first.Items[0].ItemID, "accepted", "change_class", "editorial")))
		afterFirst = state(t, h)
	}}
	doc := reviewDoc(second.ProposalID, 1, decide(second.Items[0].ItemID, "accepted", "change_class", "editorial"))
	code, stdout, stderr := asHumanWith(t, dir, in, "review", "--json", "--file", writeReview(t, doc))
	if code != apperr.ExitStalePrecondition || !strings.Contains(stdout, "to confirm") {
		t.Fatalf("exit %d, want 5 after the confirmation\n%s%s", code, stdout, stderr)
	}
	var env apperr.Envelope
	if err := json.Unmarshal([]byte(jsonPart(stdout)), &env); err != nil {
		t.Fatal(err)
	}
	if v, _ := env.Error.Current["baseline_version"].(float64); v != 2 || len(env.Error.Details) == 0 || env.Error.Details[0].Pointer != "/seen_base_version" {
		t.Fatalf("error = %+v", env.Error)
	}
	if after := state(t, h); after != afterFirst {
		t.Fatalf("written after the first review: %s -> %s", afterFirst, after)
	}
	b := baselineShow(t, dir)
	if b.BaselineVersion != 2 || b.Requirements[0].Title != "sign up with e-mail" {
		t.Fatalf("baseline = %+v", b)
	}
	got := rows(t, h, `SELECT review_state, reviewed_seq IS NULL FROM proposal_items WHERE id = ?`, second.Items[0].ItemID)
	if want := [][]string{{"conflict", "1"}}; !equalRows(got, want) {
		t.Fatalf("second item = %v, want %v", got, want)
	}
}

// asHumanWith is asHuman with the confirmation read from in.
func asHumanWith(t *testing.T, dir string, in io.Reader, args ...string) (int, string, string) {
	t.Helper()
	t.Chdir(dir)
	var out, errOut bytes.Buffer
	root := newRoot(&out, &errOut)
	root.SetIn(in)
	code := executeAs(context.Background(), channel.Detection{Channel: channel.HumanCLI}, root, args, &errOut)
	return code, out.String(), errOut.String()
}
