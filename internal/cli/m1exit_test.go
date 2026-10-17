package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// The M1 exit evidence (docs/m1-exit-evidence.md, items CH-01, CH-02,
// BL-21, BL-27, PRV-07): tests only, of behaviour already implemented.

const (
	// stdoutFileEnv names the file a helper sends stdout to, so that a
	// test reads exactly what the command printed there.
	stdoutFileEnv = "SPECMETRY_TEST_STDOUT_FILE"
	// execBinaryEnv names the binary the exec helper runs.
	execBinaryEnv = "SPECMETRY_TEST_EXEC"
)

// cliStdoutToFileHelper is cliRedirectedHelper with stdout going to the
// file stdoutFileEnv names, while stdin and stderr stay the terminal, as in
// "specmetry ... > file".
func cliStdoutToFileHelper() int {
	f, err := os.Create(os.Getenv(stdoutFileEnv))
	if err != nil {
		return 1
	}
	defer func() { _ = f.Close() }()
	terminal := os.Stdout
	os.Stdout = f
	root := newRoot(f, terminal)
	code := executeRoot(context.Background(), root, helperArgs(), terminal)
	fmt.Fprintf(terminal, "\nexit=%d\n", code)
	return code
}

// execHelper runs the binary execBinaryEnv names, with the helper's
// arguments and directory, as its own child process: stdin and stderr are
// the helper's (the terminal it runs in), and stdout too unless
// stdoutFileEnv names a file. It reports the child's exit code, which a
// terminal session does not.
func execHelper() int {
	args := helperArgs()
	cmd := exec.Command(os.Getenv(execBinaryEnv), args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if p := os.Getenv(stdoutFileEnv); p != "" {
		f, err := os.Create(p)
		if err != nil {
			return 1
		}
		defer func() { _ = f.Close() }()
		cmd.Stdout = f
	}
	code := 0
	var exit *exec.ExitError
	switch err := cmd.Run(); {
	case errors.As(err, &exit):
		code = exit.ExitCode()
	case err != nil:
		fmt.Fprintln(os.Stderr, err)
		code = -1
	}
	fmt.Fprintf(os.Stderr, "\nexit=%d\n", code)
	return 0
}

// inTerminalStdoutToFile runs the CLI in its own process under a PTY with
// stdout redirected to a file, and returns what reached the terminal
// (stderr, ending with "exit=<code>") and what reached stdout.
func inTerminalStdoutToFile(t *testing.T, dir string, args []string) (terminal, stdout string) {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "stdout")
	helper := testutil.HelperCommand(t, "cli-stdout-to-file")
	env := helperEnv(t, "cli-stdout-to-file", argsJSONEnv+"="+string(raw), dirEnv+"="+dir, stdoutFileEnv+"="+file)
	terminal = testutil.RunInPTYSession(t, 60*time.Second, env, nil, helper.Path, helper.Args[1:]...).Output
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return terminal, string(b)
}

// piped runs the CLI in its own process with stdin not a terminal (the
// null device) and its output to a pipe, and returns everything printed.
func piped(t *testing.T, dir string, args []string) string {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	cmd := testutil.HelperCommand(t, "cli")
	cmd.Env = helperEnv(t, "cli", argsJSONEnv+"="+string(raw), dirEnv+"="+dir)
	b, _ := cmd.CombinedOutput()
	return string(b)
}

// humanCommand is a human-only command line and the text that would
// confirm it.
type humanCommand struct {
	name    string
	args    []string
	confirm string
}

// exitEvidenceProject is questionsProject (baseline version 2, Q-001 to
// Q-004 open) with what the human-only commands of CH-01 and CH-02 act on:
// docs/client-brief.md changed since it was acknowledged, a non-document
// file, and a person's edit of REQ-001. It returns the command lines:
// propose --apply and question dismiss first.
func exitEvidenceProject(t *testing.T) (datadir.Dir, string, []humanCommand) {
	t.Helper()
	home, dir, _ := questionsProject(t)
	if err := os.WriteFile(filepath.Join(dir, "docs", "client-brief.md"), []byte("# brief, revised\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "logo.bin"), []byte{0, 1, 2}, 0o600); err != nil {
		t.Fatal(err)
	}
	var briefSHA string
	for _, s := range sourceList(t, dir).Sources {
		if s.Path == "docs/client-brief.md" {
			briefSHA = *s.CurrentSHA256
		}
	}
	version := baselineShow(t, dir).BaselineVersion
	item := update("requirement", "REQ-001", 1, map[string]any{"title": "sign up, edited"})
	item["change_class"] = "editorial"
	return home, dir, []humanCommand{
		{"propose --apply", []string{"propose", "--apply", "--file", writeProposal(t, editDoc(version, item))}, fmt.Sprintf("v%d", version+1)},
		{"question dismiss", []string{"question", "dismiss", "Q-002", "-m", "later"}, "Q-002"},
		{"source ack", []string{"source", "ack", "docs/client-brief.md", "--observed", briefSHA, "-m", "handled"}, "docs/client-brief.md"},
		{"source add --allow-any-type", []string{"source", "add", "assets/logo.bin", "--allow-any-type"}, "assets/logo.bin"},
	}
}

// MVP-CH-01 for propose --apply and question dismiss: the CLI in its own
// process at a real terminal (a PTY) with CLAUDECODE or
// CLAUDE_CODE_ENTRYPOINT set to a canary is exit 3, human_required (also
// as error.v1 with --json), before any prompt: the process ends without
// any input typed, and in this process, with the right confirmation
// waiting, the Agent channel reads none of it. Nothing is written: not a
// seq, a question, a proposal, a version or a source. The canary is in
// neither the output nor the data directory. The control: without the
// variable the same command line asks for confirmation, and declined,
// writes nothing.
func TestAgentVariablesRefuseTheRemainingHumanOnlyCommands(t *testing.T) {
	home, dir, commands := exitEvidenceProject(t)
	canary := testutil.NewCanary(t)
	before := confirmState(t, home)
	for _, c := range commands[:2] {
		for _, name := range []string{"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT"} {
			for _, args := range [][]string{c.args, append(slices.Clone(c.args), "--json")} {
				out := inTerminal(t, dir, args, noPrompt, name+"="+canary)
				refused := `"code": "human_required"`
				if len(args) == len(c.args) {
					refused = c.name + " is a human-only operation"
				}
				if !strings.Contains(out, "exit=3") || !strings.Contains(out, refused) || !strings.Contains(out, name+" is set") ||
					strings.Contains(out, "to confirm") || strings.Contains(out, canary) {
					t.Errorf("%s with %s, %v:\n%s", c.name, name, args[len(args)-1], out)
				}
			}
		}
		t.Chdir(dir)
		var stdout, stderr bytes.Buffer
		in := &readCounter{r: strings.NewReader(c.confirm + "\n")}
		agent := channel.Detection{Channel: channel.AgentCLI, Reasons: []string{"the Agent environment variable CLAUDECODE is set"}}
		if code := runAs(agent, &stdout, &stderr, in, append(slices.Clone(c.args), "--json")...); code != apperr.ExitHumanRequired || in.reads != 0 ||
			strings.Contains(stdout.String()+stderr.String(), "to confirm") {
			t.Errorf("%s from the Agent channel: exit %d after %d reads, want 3 and none\n%s%s", c.name, code, in.reads, stdout.String(), stderr.String())
		}
		errorV1(t, c.name, wholeJSON(t, stdout.String()), apperr.HumanRequired)
		if out := inTerminal(t, dir, c.args, ""); !strings.Contains(out, "exit=3") || !strings.Contains(out, "to confirm") {
			t.Errorf("%s at a person's terminal, declined:\n%s", c.name, out)
		}
	}
	if after := confirmState(t, home); after != before {
		t.Fatalf("written:\n%s\n%s", before, after)
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the canary reached the data directory: %v", hits)
	}
}

// MVP-CH-02 for source ack, source add --allow-any-type, propose --apply
// and question dismiss: the CLI in its own process with stdin the terminal
// and stdout redirected to a file is agent_cli: exit 3, the reason given,
// no prompt on the terminal or in the file, and with --json the file is
// exactly one error.v1 (human_required). With stdin not a terminal, also
// exit 3. Nothing is written.
func TestRedirectionRefusesTheRemainingHumanOnlyCommands(t *testing.T) {
	home, dir, commands := exitEvidenceProject(t)
	before := confirmState(t, home)
	for _, c := range commands {
		for _, args := range [][]string{c.args, append(slices.Clone(c.args), "--json")} {
			withJSON := len(args) > len(c.args)
			what := fmt.Sprintf("%s (--json %v)", c.name, withJSON)
			terminal, stdout := inTerminalStdoutToFile(t, dir, args)
			if !strings.Contains(terminal, "exit=3") || strings.Contains(terminal+stdout, "to confirm") {
				t.Errorf("%s, stdout redirected:\nterminal %q\nstdout %q", what, terminal, stdout)
			}
			if withJSON {
				errorV1(t, what+", stdout redirected", wholeJSON(t, stdout), apperr.HumanRequired)
			} else if stdout != "" || !strings.Contains(terminal, "stdout is not a terminal") {
				t.Errorf("%s, stdout redirected: want the reason on the terminal and nothing on stdout\nterminal %q\nstdout %q", what, terminal, stdout)
			}
			if out := piped(t, dir, args); !strings.Contains(out, "exit=3") || !strings.Contains(out, "stdin is not a terminal") ||
				strings.Contains(out, "to confirm") {
				t.Errorf("%s, stdin not a terminal:\n%s", what, out)
			}
		}
	}
	if after := confirmState(t, home); after != before {
		t.Fatalf("written:\n%s\n%s", before, after)
	}
}

// MVP-BL-21: a check command holding a secret is refused (exit 2) with the
// hint to reference an environment variable instead, as error.v1 with the
// pointer of the command and the rule (secret_detected: github_token), in
// JSON and in text; the secret is in neither output nor the data
// directory, and nothing is written.
func TestProposeSecretInACheckNamesAnEnvironmentVariable(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	secret := testutil.NewSecretCanary(t)
	doc := patched(t, recorded(t, f), "/items/2/data/check", map[string]any{
		"command": "curl -H 'Authorization: token " + secret + "' https://ci.example/run && go test ./internal/auth", "rationale": "the lockout"})
	const hint = "reference an environment variable by name instead"
	env := proposeRefused(t, home, dir, doc, apperr.ExitInvalidInput, "/items/2/data/check/command")
	if env.Error.Code != apperr.InvalidInput || !strings.Contains(env.Error.Message, hint) {
		t.Errorf("error = %+v", env.Error)
	}
	found := false
	for _, d := range env.Error.Details {
		found = found || (d.Pointer == "/items/2/data/check/command" && d.Problem == "secret_detected: github_token")
	}
	if b, _ := json.Marshal(env); !found || bytes.Contains(b, []byte(secret)) {
		t.Errorf("details %+v, or the secret echoed", env.Error.Details)
	}
	before := written(t, home)
	code, stdout, stderr := specmetry(t, dir, "", "propose", "--json", "--file", writeProposal(t, doc))
	errorV1(t, "propose --json", wholeJSON(t, stdout), apperr.InvalidInput)
	if code != apperr.ExitInvalidInput || strings.Contains(stdout+stderr, secret) {
		t.Errorf("--json: exit %d\n%s%s", code, stdout, stderr)
	}
	code, stdout, stderr = specmetry(t, dir, "", "propose", "--file", writeProposal(t, doc))
	if code != apperr.ExitInvalidInput || stdout != "" || !strings.Contains(stderr, hint) || strings.Contains(stderr, secret) {
		t.Errorf("text: exit %d\nstdout %q\nstderr %q", code, stdout, stderr)
	}
	if after := written(t, home); after != before {
		t.Errorf("written %s -> %s", before, after)
	}
	if hits := testutil.FindInTree(t, home.Path, secret); len(hits) != 0 {
		t.Fatalf("the secret reached the data directory: %v", hits)
	}
}

// MVP-BL-27 for an item in conflict: accepting an Agent's update that is
// in conflict, its conflict acknowledged, as accepted or as
// accepted_with_edits but without change_class, is exit 2 at
// /items/0/change_class: no seq, the item still in conflict, the baseline
// unchanged. With change_class the same decision is applied.
func TestReviewConflictItemNeedsAChangeClass(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	acceptAll(t, dir, proposeOK(t, dir, small(0, addReq("a", "sign up", nil))), 0, nil)
	first := proposeOK(t, dir, small(1, update("requirement", "REQ-001", 1, map[string]any{"title": "sign up with e-mail"})))
	second := proposeOK(t, dir, small(1, update("requirement", "REQ-001", 1, map[string]any{"priority": "should"})))
	acceptAll(t, dir, first, 1, nil)
	item := second.Items[0].ItemID
	inConflict := func() {
		t.Helper()
		got := rows(t, home, `SELECT review_state, (SELECT max(number) FROM baseline_versions) FROM proposal_items WHERE id = ?`, item)
		if !equalRows(got, [][]string{{"conflict", "2"}}) {
			t.Fatalf("item and version = %v, want conflict at version 2", got)
		}
	}
	inConflict()
	reviewRefused(t, home, dir, reviewDoc(second.ProposalID, 2, decide(item, "accepted", "acknowledged_conflict", true)),
		apperr.ExitInvalidInput, "/items/0/change_class")
	reviewRefused(t, home, dir, reviewDoc(second.ProposalID, 2, decide(item, "accepted_with_edits", "acknowledged_conflict", true,
		"final_data", map[string]any{"priority": "could"})), apperr.ExitInvalidInput, "/items/0/change_class")
	inConflict()
	reviewOK(t, dir, reviewDoc(second.ProposalID, 2, decide(item, "accepted_with_edits", "acknowledged_conflict", true,
		"final_data", map[string]any{"priority": "could"}, "change_class", "editorial")))
	got := rows(t, home, `SELECT review_state, change_class, (SELECT max(number) FROM baseline_versions),
    (SELECT priority FROM requirements WHERE key = 'REQ-001') FROM proposal_items WHERE id = ?`, item)
	if !equalRows(got, [][]string{{"accepted_with_edits", "editorial", "3", "could"}}) {
		t.Fatalf("after the review: %v", got)
	}
}

// MVP-PRV-07 for review: a review.v1 document is scanned whole. A secret
// in the summary, an item's note, an edited item's description (nested in
// final_data) or the scope's note is exit 2 with the detail at its pointer
// and rule, for a person at a terminal who would confirm (the scan runs
// after the channel check): no confirmation shown, the secret in neither
// output nor the data directory, nothing written (seq, versions, items,
// questions, scope). The same document without the secret is reviewed.
func TestReviewRefusesASecretAnywhereInTheDocument(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	secret := testutil.NewSecretCanary(t)
	r := proposeOK(t, dir, small(0, addReq("r1", "sign up", nil), addReq("r2", "export", nil)))
	ids := itemIDs(r)
	doc := func(secretAt string) map[string]any {
		text := func(at, plain string) string {
			if at == secretAt {
				return plain + " " + secret
			}
			return plain
		}
		d := reviewDoc(r.ProposalID, 0, decide(ids["r1"], "accepted", "note", text("/items/0/note", "as asked")),
			decide(ids["r2"], "accepted_with_edits", "final_data", map[string]any{"title": "export orders", "kind": "functional", "priority": "must",
				"description": text("/items/1/final_data/description", "as CSV")}))
		d["summary"] = text("/summary", "first review")
		if secretAt == "/scope/note" {
			d["scope"] = map[string]any{"target_id": r.ProposalID, "source_set_hash": strings.Repeat("0", 64), "requirements": []any{ids["r1"]},
				"attestations": map[string]any{"complete_scope": true, "pending_items_excluded": 0}, "acknowledged_waivers": []any{},
				"seen_scope_confirmation_id": nil, "note": text("/scope/note", "first scope")}
		}
		return d
	}
	for _, at := range []string{"/summary", "/items/0/note", "/items/1/final_data/description", "/scope/note"} {
		before := state(t, home)
		for _, mode := range []string{"--json", "--json=false"} {
			code, stdout, stderr := asHuman(t, dir, r.ProposalID, "review", mode, "--file", writeReview(t, doc(at)))
			what := at + " " + mode
			if code != apperr.ExitInvalidInput || strings.Contains(stdout+stderr, "to confirm") || strings.Contains(stdout+stderr, secret) {
				t.Errorf("%s: exit %d\nstdout %q\nstderr %q", what, code, stdout, stderr)
				continue
			}
			if mode == "--json=false" {
				if stdout != "" || !strings.Contains(stderr, "looks like a secret") {
					t.Errorf("%s: stdout %q, stderr %q", what, stdout, stderr)
				}
				continue
			}
			env := errorV1(t, what, wholeJSON(t, stdout), apperr.InvalidInput)
			if len(env.Error.Details) != 1 || env.Error.Details[0].Pointer != at || env.Error.Details[0].Problem != "secret_detected: github_token" {
				t.Errorf("%s: details %+v", what, env.Error.Details)
			}
		}
		if after := state(t, home); after != before {
			t.Errorf("%s: written %s -> %s", at, before, after)
		}
	}
	if hits := testutil.FindInTree(t, home.Path, secret); len(hits) != 0 {
		t.Fatalf("the secret reached the data directory: %v", hits)
	}
	reviewOK(t, dir, doc(""))
	if got := baselineShow(t, dir).keys(); got != "REQ-001 REQ-002" {
		t.Fatalf("baseline = %s", got)
	}
}
