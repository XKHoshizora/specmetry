package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// inTerminalBinary runs bin with args in dir as the child of the exec
// helper, under a PTY: the binary's stdin and stderr are the terminal, and
// stdout too unless stdoutFile names a file. It returns the terminal's
// output, which ends with the binary's "exit=<code>".
func inTerminalBinary(t *testing.T, bin, dir string, args []string, steps []testutil.PTYStep, stdoutFile string, extra ...string) string {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	env := append(slices.Clone(extra), argsJSONEnv+"="+string(raw), dirEnv+"="+dir, execBinaryEnv+"="+bin)
	if stdoutFile != "" {
		env = append(env, stdoutFileEnv+"="+stdoutFile)
	}
	helper := testutil.HelperCommand(t, "exec")
	return testutil.RunInPTYSession(t, 60*time.Second, helperEnv(t, "exec", env...), steps, helper.Path, helper.Args[1:]...).Output
}

// The cmd/specmetry binary itself, built with go build from the main
// package as a release is, run as its own process (acceptance-criteria
// §1, 正式二进制). It is built once, without the embedded dashboard
// (-tags nowebui; the CI's dashboard build smoke-tests that build too).
//   - JSON reads: init --json, source add --json, source list --json and
//     target show --json, each validated against its schema;
//   - exit codes: version 0, an unknown command 2 (error.v1 with --json),
//     outside a project 4 (not_initialized), a human-only command with
//     stdin not a terminal 3 (human_required);
//   - the process boundaries of MVP-CH-01 and CH-02, the binary at a real
//     terminal (a PTY): a person is asked to confirm question dismiss and
//     declining writes nothing; with CLAUDECODE or CLAUDE_CODE_ENTRYPOINT
//     set to a canary question dismiss and propose --apply are exit 3
//     before any prompt, and the canary is in neither the output nor the
//     data directory; with stdout redirected it is exit 3 with no prompt,
//     and with --json stdout is one error.v1; confirmed, the binary
//     dismisses the question.
func TestBuiltBinary(t *testing.T) {
	bin := buildSpecmetry(t) // before any t.Chdir
	fresh := testutil.BuildFixture(t, "fx-docs-only")
	home, dir, _ := questionsProject(t)
	val, err := schemas()
	if err != nil {
		t.Fatal(err)
	}
	conforms := func(what, schema string, code int, stdout, stderr string) {
		t.Helper()
		if code != apperr.ExitOK {
			t.Fatalf("%s: exit %d\n%s%s", what, code, stdout, stderr)
		}
		if err := val.ValidateJSON(schema, []byte(wholeJSON(t, stdout))); err != nil {
			t.Fatalf("%s: stdout is not %s: %v\n%s", what, schema, err, stdout)
		}
	}

	code, stdout, stderr := runSpecmetry(t, bin, dir, nil, "version")
	if code != apperr.ExitOK || !strings.Contains(stdout, "specmetry") {
		t.Fatalf("version: exit %d\n%s%s", code, stdout, stderr)
	}
	code, stdout, stderr = runSpecmetry(t, bin, fresh.Repo.Dir, nil, "init", "--json")
	conforms("init --json", "init.v1", code, stdout, stderr)
	var initOut initOutput
	if err := json.Unmarshal([]byte(stdout), &initOut); err != nil || !initOut.Created || initOut.Target == nil {
		t.Fatalf("init = %+v (%v)", initOut, err)
	}
	code, stdout, stderr = runSpecmetry(t, bin, fresh.Repo.Dir, nil, "source", "add", "docs/client-brief.md", "--json")
	conforms("source add --json", "source-add.v1", code, stdout, stderr)
	code, stdout, stderr = runSpecmetry(t, bin, fresh.Repo.Dir, nil, "source", "list", "--json")
	conforms("source list --json", "source-list.v1", code, stdout, stderr)
	code, stdout, stderr = runSpecmetry(t, bin, fresh.Repo.Dir, nil, "target", "show", "--json")
	conforms("target show --json", "target-scope.v1", code, stdout, stderr)

	code, stdout, _ = runSpecmetry(t, bin, dir, nil, "--json", "nosuch")
	if code != apperr.ExitInvalidInput {
		t.Fatalf("an unknown command: exit %d", code)
	}
	errorV1(t, "an unknown command", wholeJSON(t, stdout), apperr.InvalidInput)
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, _ = runSpecmetry(t, bin, outside, nil, "source", "list", "--json")
	if code != apperr.ExitNotInitialized {
		t.Fatalf("outside a project: exit %d", code)
	}
	errorV1(t, "outside a project", wholeJSON(t, stdout), apperr.NotInitialized)

	before := confirmState(t, home)
	dismiss := []string{"question", "dismiss", "Q-002", "-m", "later"}
	code, stdout, _ = runSpecmetry(t, bin, dir, nil, append(dismiss, "--json")...)
	if code != apperr.ExitHumanRequired {
		t.Fatalf("stdin not a terminal: exit %d", code)
	}
	errorV1(t, "stdin not a terminal", wholeJSON(t, stdout), apperr.HumanRequired)

	// A person's terminal: the confirmation is asked for; declined.
	declined := []testutil.PTYStep{{Expect: "press Enter to cancel:", Send: "\r"}}
	if out := inTerminalBinary(t, bin, dir, dismiss, declined, ""); !strings.Contains(out, "exit=3") || !strings.Contains(out, "to confirm") {
		t.Fatalf("at a person's terminal, declined:\n%s", out)
	}
	// The Agent variables.
	version := baselineShow(t, dir).BaselineVersion
	item := update("requirement", "REQ-001", 1, map[string]any{"title": "sign up, edited"})
	item["change_class"] = "editorial"
	apply := []string{"propose", "--apply", "--file", writeProposal(t, editDoc(version, item))}
	canary := testutil.NewCanary(t)
	for _, name := range []string{"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT"} {
		for _, args := range [][]string{dismiss, apply} {
			out := inTerminalBinary(t, bin, dir, args, nil, "", name+"="+canary)
			if !strings.Contains(out, "exit=3") || !strings.Contains(out, "is a human-only operation") || !strings.Contains(out, name+" is set") ||
				strings.Contains(out, "to confirm") || strings.Contains(out, canary) {
				t.Errorf("%v with %s:\n%s", args[:2], name, out)
			}
		}
	}
	// stdout redirected.
	for _, args := range [][]string{dismiss, append(dismiss, "--json")} {
		file := filepath.Join(t.TempDir(), "stdout")
		out := inTerminalBinary(t, bin, dir, args, nil, file)
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "exit=3") || strings.Contains(out+string(b), "to confirm") {
			t.Errorf("stdout redirected %v:\nterminal %q\nstdout %q", args[len(args)-1], out, b)
		}
		if len(args) > len(dismiss) {
			errorV1(t, "stdout redirected, --json", wholeJSON(t, string(b)), apperr.HumanRequired)
		} else if len(b) != 0 || !strings.Contains(out, "stdout is not a terminal") {
			t.Errorf("stdout redirected: terminal %q, stdout %q", out, b)
		}
	}
	if after := confirmState(t, home); after != before {
		t.Fatalf("written:\n%s\n%s", before, after)
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the canary reached the data directory: %v", hits)
	}

	// Confirmed at a person's terminal, the binary dismisses the question.
	confirmed := []testutil.PTYStep{{Expect: "press Enter to cancel:", Send: "Q-002\r"}}
	if out := inTerminalBinary(t, bin, dir, dismiss, confirmed, ""); !strings.Contains(out, "exit=0") {
		t.Fatalf("confirmed:\n%s", out)
	}
	got := rows(t, home, `SELECT status, answered_channel FROM questions WHERE key = 'Q-002'`)
	if !equalRows(got, [][]string{{"dismissed", "human_cli"}}) {
		t.Fatalf("Q-002 = %v", got)
	}
}
