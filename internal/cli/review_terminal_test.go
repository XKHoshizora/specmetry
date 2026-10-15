package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// MVP-BL-09, MVP-CH-01 and CH-02 with the real review command:
//   - in a terminal with an Agent's variable set (its value a canary): exit
//     3, no prompt, nothing written, the canary nowhere;
//   - not a terminal (stdin or stdout), with --json too: exit 3, error.v1;
//   - a person who declines: exit 3, nothing written;
//   - a person who confirms with the proposal's ID: the version is made.
func TestReviewInTerminal(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	r := proposeOK(t, dir, small(0, addReq("r1", "sign up", nil)))
	file := writeReview(t, reviewDoc(r.ProposalID, 0, decide(r.Items[0].ItemID, "accepted")))
	args := []string{"review", "--file", file}
	canary := testutil.NewCanary(t)
	before := state(t, home)

	for _, name := range []string{"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT"} {
		out := inTerminal(t, dir, args, noPrompt, name+"="+canary)
		if !strings.Contains(out, "exit=3") || strings.Contains(out, "to confirm") || strings.Contains(out, canary) {
			t.Errorf("with %s:\n%s", name, out)
		}
	}
	code, stdout, stderr := specmetry(t, dir, "", append(args, "--json")...)
	var env apperr.Envelope
	if err := json.Unmarshal([]byte(stdout), &env); code != apperr.ExitHumanRequired || err != nil || env.Error.Code != "human_required" {
		t.Errorf("not a terminal: exit %d\n%s%s", code, stdout, stderr)
	}
	raw, err := json.Marshal(append(args, "--json"))
	if err != nil {
		t.Fatal(err)
	}
	redirected := testutil.HelperCommand(t, "cli-redirected")
	out := testutil.RunInPTYSession(t, 60*time.Second, helperEnv(t, "cli-redirected", argsJSONEnv+"="+string(raw), dirEnv+"="+dir),
		nil, redirected.Path, redirected.Args[1:]...).Output
	if !strings.Contains(out, "exit=3") || strings.Contains(out, "to confirm") {
		t.Errorf("stdout redirected:\n%s", out)
	}
	if out := inTerminal(t, dir, args, ""); !strings.Contains(out, "exit=3") || !strings.Contains(out, "not confirmed") {
		t.Errorf("declined:\n%s", out)
	}
	if after := state(t, home); after != before {
		t.Fatalf("written: %s -> %s", before, after)
	}

	out = inTerminal(t, dir, args, r.ProposalID)
	if !strings.Contains(out, "exit=0") || !strings.Contains(out, "Baseline version 1 created") {
		t.Fatalf("confirmed:\n%s", out)
	}
	got := rows(t, home, `SELECT number, channel, substr(actor, 1, 6) FROM baseline_versions`)
	if want := [][]string{{"1", "human_cli", "human:"}}; !equalRows(got, want) {
		t.Fatalf("versions = %v", got)
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the canary reached the data directory: %v", hits)
	}
}
