package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/datadir"
)

// atTerminal is the channel of a person at a terminal: stdin, stdout and
// stderr are all the terminal.
var atTerminal = channel.Detection{Channel: channel.HumanCLI, StderrTerminal: true}

// asHuman runs the CLI in-process as a person at a terminal does: the
// channel is human_cli, and confirm is the line typed at the confirmation
// prompt (noPrompt: nothing typed, the input ends). The real channel
// detection and terminal are covered by the PTY tests.
func asHuman(t *testing.T, dir, confirm string, args ...string) (int, string, string) {
	t.Helper()
	t.Chdir(dir)
	var out, errOut bytes.Buffer
	root := newRoot(&out, &errOut)
	in := ""
	if confirm != noPrompt {
		in = confirm + "\n"
	}
	root.SetIn(strings.NewReader(in))
	code := executeAs(context.Background(), atTerminal, root, args, &errOut)
	return code, out.String(), errOut.String()
}

func writeReview(t *testing.T, doc any) string {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "review.json")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// reviewDoc is a review.v1 document.
func reviewDoc(proposalID string, seen int, items ...map[string]any) map[string]any {
	list := make([]any, len(items))
	for i, it := range items {
		list[i] = it
	}
	return map[string]any{"schema": "specmetry.review/v1", "proposal_id": proposalID, "seen_base_version": seen, "items": list}
}

func decide(itemID, decision string, more ...any) map[string]any {
	d := map[string]any{"item_id": itemID, "decision": decision}
	for i := 0; i+1 < len(more); i += 2 {
		d[more[i].(string)] = more[i+1]
	}
	return d
}

// reviewOK reviews as a person who confirms, and returns the text output.
func reviewOK(t *testing.T, dir string, doc map[string]any) string {
	t.Helper()
	code, stdout, stderr := asHuman(t, dir, doc["proposal_id"].(string), "review", "--file", writeReview(t, doc))
	if code != apperr.ExitOK {
		t.Fatalf("review: exit %d\n%s%s", code, stdout, stderr)
	}
	return stdout
}

// state is everything a review may write, to compare before and after a
// refusal: the seq counter, versions, changes, entities, references, key
// counters, reviewed or conflicting items, and scope confirmations with
// their items.
func state(t *testing.T, home datadir.Dir) string {
	t.Helper()
	// One query: opening the database is what takes time.
	return strings.Join(rows(t, home, `SELECT
    (SELECT seq_counter FROM meta),
    (SELECT count(*) FROM baseline_versions),
    (SELECT count(*) FROM baseline_changes),
    (SELECT count(*) FROM goals),
    (SELECT count(*) || '/' || coalesce(sum(revision), 0) FROM requirements),
    (SELECT count(*) || '/' || coalesce(sum(revision), 0) || '/' || coalesce(sum(requirement_revision), 0) FROM acceptance_criteria),
    (SELECT count(*) FROM source_refs),
    (SELECT count(*) FROM code_refs),
    (SELECT coalesce(sum(next_value), 0) FROM key_counters),
    (SELECT count(*) FROM proposal_items WHERE review_state <> 'pending'),
    (SELECT count(*) FROM questions WHERE related_id IS NOT NULL),
    (SELECT count(*) FROM scope_confirmations), (SELECT count(*) FROM scope_items)`)[0], " ")
}

// reviewRefused runs a review that must be refused with code, with a
// detail at pointer (if given), and nothing written.
func reviewRefused(t *testing.T, home datadir.Dir, dir string, doc map[string]any, code int, pointer string) apperr.Envelope {
	t.Helper()
	before := state(t, home)
	got, stdout, stderr := asHuman(t, dir, doc["proposal_id"].(string), "review", "--json", "--file", writeReview(t, doc))
	if got != code {
		t.Errorf("exit %d, want %d (%s)\n%s%s", got, code, pointer, stdout, stderr)
	}
	// Refused before the person is asked anything, on either stream.
	if strings.Contains(stdout+stderr, "to confirm") {
		t.Errorf("%s: the person was asked to confirm a review that is refused", pointer)
	}
	var env apperr.Envelope
	if err := json.Unmarshal([]byte(wholeJSON(t, stdout)), &env); err != nil {
		t.Errorf("no error.v1: %v\n%s", err, stdout)
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
	if after := state(t, home); after != before {
		t.Errorf("written: %s -> %s", before, after)
	}
	return env
}

// showProposal reads proposal-show.v1.
func showProposal(t *testing.T, dir, id string) proposalShow {
	t.Helper()
	code, stdout, stderr := specmetry(t, dir, "proposal-show.v1", "proposal", "show", id, "--json")
	if code != apperr.ExitOK {
		t.Fatalf("proposal show: exit %d\n%s%s", code, stdout, stderr)
	}
	var doc proposalShow
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

type proposalShow struct {
	BaselineVersion int `json:"baseline_version"`
	Items           []struct {
		ItemID      string         `json:"item_id"`
		ReviewState string         `json:"review_state"`
		Proposed    map[string]any `json:"proposed"`
		Target      *string        `json:"target"`
		Conflict    *struct {
			BaseRevision    int            `json:"base_revision"`
			CurrentRevision int            `json:"current_revision"`
			CurrentStatus   string         `json:"current_status"`
			Base            map[string]any `json:"base"`
			Current         map[string]any `json:"current"`
			Proposed        map[string]any `json:"proposed"`
		} `json:"conflict"`
		Review *struct {
			Decision    string         `json:"decision"`
			ChangeClass *string        `json:"change_class"`
			Final       map[string]any `json:"final"`
			By          string         `json:"by"`
			Channel     string         `json:"channel"`
		} `json:"review"`
	} `json:"items"`
	Questions []struct {
		Key   string  `json:"key"`
		About *string `json:"about"`
	} `json:"questions"`
}

// baselineShow reads baseline-show.v1.
func baselineShow(t *testing.T, dir string, args ...string) baselineDoc {
	t.Helper()
	code, stdout, stderr := specmetry(t, dir, "baseline-show.v1", append([]string{"baseline", "show", "--json"}, args...)...)
	if code != apperr.ExitOK {
		t.Fatalf("baseline show: exit %d\n%s%s", code, stdout, stderr)
	}
	var doc baselineDoc
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

type baselineDoc struct {
	BaselineVersion int `json:"baseline_version"`
	Goals           []struct {
		Key      string `json:"key"`
		Revision int    `json:"revision"`
		Status   string `json:"status"`
		Title    string `json:"title"`
	} `json:"goals"`
	Requirements []struct {
		Key      string  `json:"key"`
		Revision int     `json:"revision"`
		Status   string  `json:"status"`
		Goal     *string `json:"goal"`
		Title    string  `json:"title"`
		Criteria []struct {
			Key                 string `json:"key"`
			Revision            int    `json:"revision"`
			Status              string `json:"status"`
			RequirementRevision int    `json:"requirement_revision"`
			Statement           string `json:"statement"`
			Verification        string `json:"verification"`
		} `json:"criteria"`
	} `json:"requirements"`
}

// keys lists the baseline's keys: goals, then each requirement and its
// criteria, with a * on retired ones.
func (d baselineDoc) keys() string {
	var out []string
	mark := func(key, status string) string {
		if status == "retired" {
			return key + "*"
		}
		return key
	}
	for _, g := range d.Goals {
		out = append(out, mark(g.Key, g.Status))
	}
	for _, r := range d.Requirements {
		out = append(out, mark(r.Key, r.Status))
		for _, c := range r.Criteria {
			out = append(out, mark(c.Key, c.Status))
		}
	}
	return strings.Join(out, " ")
}

// itemIDs returns a receipt's item IDs by ref or target.
func itemIDs(r receiptOut) map[string]string {
	out := map[string]string{}
	for _, it := range r.Items {
		switch {
		case it.Ref != nil:
			out[*it.Ref] = it.ItemID
		case it.Target != nil:
			out[*it.Target] = it.ItemID
		}
		out[strconv.Itoa(it.ItemIndex)] = it.ItemID
	}
	return out
}

// small is a cli proposal with the given items, against base.
func small(base int, items ...any) map[string]any {
	return map[string]any{"schema": "specmetry.proposal/v1", "origin": "cli", "base_version": base, "summary": "changes", "items": items}
}

func addGoal(ref, title string) map[string]any {
	return map[string]any{"op": "add", "type": "goal", "ref": ref, "origin": "manual", "data": map[string]any{"title": title}}
}

func addReq(ref, title string, goal any) map[string]any {
	data := map[string]any{"title": title, "kind": "functional", "priority": "must"}
	if goal != nil {
		data["goal"] = goal
	}
	return map[string]any{"op": "add", "type": "requirement", "ref": ref, "origin": "manual", "data": data}
}

func addAC(ref, requirement, statement string) map[string]any {
	return map[string]any{"op": "add", "type": "criterion", "ref": ref, "origin": "manual",
		"data": map[string]any{"requirement": requirement, "statement": statement, "verification": "manual"}}
}

func update(typ, target string, revision int, data map[string]any) map[string]any {
	return map[string]any{"op": "update", "type": typ, "target": target, "target_revision": revision, "data": data}
}

func retire(typ, target string, revision int) map[string]any {
	return map[string]any{"op": "retire", "type": typ, "target": target, "target_revision": revision, "rationale": "no longer needed"}
}

func reaffirm(target string, revision int) map[string]any {
	return map[string]any{"op": "reaffirm", "type": "criterion", "target": target, "target_revision": revision}
}

// acceptAll reviews a receipt's items, all accepted (updates editorial
// unless classes says otherwise), and returns the text output.
func acceptAll(t *testing.T, dir string, r receiptOut, seen int, classes map[string]string) string {
	t.Helper()
	var items []map[string]any
	for _, it := range r.Items {
		d := decide(it.ItemID, "accepted")
		if it.Op == "update" {
			class := "editorial"
			if it.Target != nil && classes[*it.Target] != "" {
				class = classes[*it.Target]
			}
			d["change_class"] = class
		}
		items = append(items, d)
	}
	return reviewOK(t, dir, reviewDoc(r.ProposalID, seen, items...))
}

// wholeJSON checks that stdout is one JSON document and nothing else: with
// --json a confirmation prompt goes to stderr (codex-review-031 N2), never
// before the document. It returns stdout.
func wholeJSON(t *testing.T, stdout string) string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(stdout))
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Errorf("stdout is not a JSON document: %v\n%s", err, stdout)
		return stdout
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		t.Errorf("stdout holds more than one JSON document:\n%s", stdout)
	}
	return stdout
}
