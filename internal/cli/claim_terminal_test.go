package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// MVP-CLM-05 and M2-Q6 with the real channel detection, the CLI run as a
// process of its own in a pseudo-terminal (WP2.4 spec §15):
//   - with CLAUDECODE or CLAUDE_CODE_ENTRYPOINT set (to a canary) a manual
//     batch is exit 3 with no prompt, and the canary is nowhere;
//   - with stdout redirected, or the batch read from a pipe (--file -),
//     exit 3 with no prompt;
//   - with --json and stderr redirected, exit 6 before any prompt;
//   - declined, exit 3;
//   - none of these writes anything; confirmed, the batch is stored by a
//     person on human_cli.
//
// The channel does not depend on git: the project is outside git
// (TestClaimCompareFrom confirms a manual batch in a git project).
func TestClaimManualInTerminal(t *testing.T) {
	home, dir := claimsProjectOutsideGit(t)
	doc := manualDoc()
	file := writeClaims(t, doc)
	args := []string{"claim", "--file", file}
	canary := testutil.NewCanary(t)
	before := factState(t, home)

	for _, name := range []string{"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT"} {
		out := inTerminal(t, dir, args, noPrompt, name+"="+canary)
		if !strings.Contains(out, "exit=3") || strings.Contains(out, "to confirm") || strings.Contains(out, canary) {
			t.Errorf("with %s:\n%s", name, out)
		}
	}
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	redirected := testutil.HelperCommand(t, "cli-redirected")
	out := testutil.RunInPTYSession(t, 60*time.Second, helperEnv(t, "cli-redirected", argsJSONEnv+"="+string(raw), dirEnv+"="+dir),
		nil, redirected.Path, redirected.Args[1:]...).Output
	if !strings.Contains(out, "exit=3") || strings.Contains(out, "to confirm") {
		t.Errorf("stdout redirected:\n%s", out)
	}
	rawJSON, err := json.Marshal(append(args, "--json"))
	if err != nil {
		t.Fatal(err)
	}
	stderrRedirected := testutil.HelperCommand(t, "cli-stderr-redirected")
	out = testutil.RunInPTYSession(t, 60*time.Second, helperEnv(t, "cli-stderr-redirected", argsJSONEnv+"="+string(rawJSON), dirEnv+"="+dir),
		nil, stderrRedirected.Path, stderrRedirected.Args[1:]...).Output
	if !strings.Contains(out, "exit=6") || strings.Contains(out, "to confirm") || !strings.Contains(out, `"code": "environment"`) {
		t.Errorf("stderr redirected, with --json:\n%s", out)
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	piped, _ := json.Marshal([]string{"claim", "--file", "-", "--json"})
	cmd := testutil.HelperCommand(t, "cli")
	cmd.Env = helperEnv(t, "cli", argsJSONEnv+"="+string(piped), dirEnv+"="+dir)
	cmd.Stdin = bytes.NewReader(b)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run()
	if !strings.Contains(stderr.String(), "exit=3") || !strings.Contains(stdout.String(), `"code": "human_required"`) {
		t.Errorf("--file - from a pipe:\n%s%s", stdout.String(), stderr.String())
	}
	if out := inTerminal(t, dir, args, ""); !strings.Contains(out, "exit=3") || !strings.Contains(out, "not confirmed") {
		t.Errorf("declined:\n%s", out)
	}
	if after := factState(t, home); after != before {
		t.Fatalf("written:\n%s\n---\n%s", before, after)
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the canary reached the data directory: %v", hits)
	}

	out = inTerminal(t, dir, args, "v1")
	if !strings.Contains(out, "exit=0") || !strings.Contains(out, "stored at seq") {
		t.Fatalf("confirmed:\n%s", out)
	}
	noneEchoed(t, "in a terminal", doc, out)
	got := rows(t, home, `SELECT kind, substr(actor, 1, 6), channel FROM claim_batches`)
	if !equalRows(got, [][]string{{"manual", "human:", "human_cli"}}) {
		t.Fatalf("batches %v", got)
	}
}
