package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// Every human-only command confirms through confirm, which picks the stream
// the confirmation is shown on; none calls channel.Guard itself, so that a
// new one (WP1.8) cannot miss the --json rule (codex-review-031 N2).
func TestHumanOnlyCommandsConfirmThroughConfirm(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no sources: %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == "confirm.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(b, []byte("channel.Guard(")) {
			t.Errorf("%s calls channel.Guard; confirm through confirm(cmd, d, op)", f)
		}
	}
}

// runAs runs the CLI in this process in the current directory with the
// channel d, stdout and stderr, and the confirmation read from in.
func runAs(d channel.Detection, stdout, stderr io.Writer, in io.Reader, args ...string) int {
	root := newRoot(stdout, stderr)
	root.SetIn(in)
	return executeAs(context.Background(), d, root, args, stderr)
}

// humanOp is a human-only command with what confirms it, and the document
// it prints on success with --json ("" for text).
type humanOp struct {
	name    string
	args    func() []string
	confirm func() string
	success string
}

// confirmState is everything the commands may write, read with SQL of its
// own: closeState, and every source's acknowledged hash and removal.
func confirmState(t *testing.T, home datadir.Dir) string {
	t.Helper()
	return strings.Join(closeState(t, home), " ") + " | " + strings.Join(rows(t, home, `SELECT
    (SELECT group_concat(path || ':' || acked_sha256 || ':' || coalesce(removed_seq, '-'), ' ') FROM (SELECT * FROM sources ORDER BY path)),
    (SELECT count(*) FROM source_acks)`)[0], " ")
}

// codex-review-031 N2, for every human-only confirmation, as a person at a
// terminal runs it (in this process):
//   - with --json the confirmation is on stderr and stdout holds only the
//     command's output: one whole error.v1 when it is cancelled, answered
//     wrong, or the input ends (exit 3) or cannot be read (exit 6); one
//     whole document (proposal-receipt.v1, source-add.v1) or the success
//     text when it is confirmed;
//   - with --json and stderr not a terminal, or cut off, it is exit 6 before
//     anything is read, with error.v1 on stdout;
//   - --json=false and no flag keep the confirmation on stdout;
//   - the Agent channel is exit 3 with no confirmation anywhere.
//
// Nothing is written until a confirmation; then exactly what it confirmed.
func TestConfirmationWithJSON(t *testing.T) {
	home, dir, _ := questionsProject(t)
	pending := proposeOK(t, dir, small(2, addReq("r9", "audit log", nil)))
	if err := os.WriteFile(filepath.Join(dir, "docs", "client-brief.md"), []byte("# brief, revised\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "notes.md"), []byte("# notes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sourceAdd(t, dir, "docs/notes.md")
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
	version := func() int { return baselineShow(t, dir).BaselineVersion }
	fixed := func(s string) func() string { return func() string { return s } }
	ops := []humanOp{
		{"question answer", func() []string { return []string{"question", "answer", "Q-001", "-m", "CSV"} }, fixed("Q-001"), ""},
		{"question dismiss", func() []string { return []string{"question", "dismiss", "Q-002", "-m", "later"} }, fixed("Q-002"), ""},
		{"review", func() []string {
			return []string{"review", "--file", writeReview(t, reviewDoc(pending.ProposalID, version(), decide(pending.Items[0].ItemID, "accepted")))}
		}, fixed(pending.ProposalID), ""},
		{"propose --apply", func() []string {
			item := update("requirement", "REQ-001", 1, map[string]any{"title": "sign up, edited"})
			item["change_class"] = "editorial"
			return []string{"propose", "--apply", "--file", writeProposal(t, editDoc(version(), item))}
		}, func() string { return fmt.Sprintf("v%d", version()+1) }, "proposal-receipt.v1"},
		{"source ack", func() []string {
			return []string{"source", "ack", "docs/client-brief.md", "--observed", briefSHA, "-m", "handled"}
		}, fixed("docs/client-brief.md"), ""},
		{"source remove", func() []string { return []string{"source", "remove", "docs/notes.md"} }, fixed("docs/notes.md"), ""},
		{"source add --allow-any-type", func() []string { return []string{"source", "add", "assets/logo.bin", "--allow-any-type"} }, fixed("assets/logo.bin"), "source-add.v1"},
	}
	before := confirmState(t, home)
	t.Chdir(dir)
	for _, op := range ops {
		withJSON := append(op.args(), "--json")
		// Cancelled, answered wrong, or at the end of the input: exit 3.
		for _, answer := range []string{"", "wrong", noPrompt} {
			code, stdout, stderr := asHuman(t, dir, answer, withJSON...)
			what := fmt.Sprintf("%s --json, answered %q", op.name, answer)
			if code != apperr.ExitHumanRequired {
				t.Errorf("%s: exit %d, want 3\n%s%s", what, code, stdout, stderr)
			}
			errorV1(t, what, wholeJSON(t, stdout), apperr.HumanRequired)
			if !strings.Contains(stderr, "to confirm") || strings.Contains(stdout, "to confirm") || strings.Contains(stderr, "error:") {
				t.Errorf("%s: the confirmation is not on stderr alone:\nstdout %q\nstderr %q", what, stdout, stderr)
			}
		}
		// The confirmation cannot be read: exit 6.
		var stdout, stderr bytes.Buffer
		code := runAs(atTerminal, &stdout, &stderr, iotest.ErrReader(errors.New("input closed")), withJSON...)
		if code != apperr.ExitEnvironment || !strings.Contains(stderr.String(), "to confirm") {
			t.Errorf("%s --json, unreadable input: exit %d\n%s%s", op.name, code, stdout.String(), stderr.String())
		}
		errorV1(t, op.name+" --json, unreadable input", wholeJSON(t, stdout.String()), apperr.Environment)
		// stderr is not a terminal, or is cut off: exit 6, nothing read.
		for _, c := range []struct {
			what   string
			d      channel.Detection
			stderr io.Writer
		}{
			{"stderr not a terminal", channel.Detection{Channel: channel.HumanCLI}, &bytes.Buffer{}},
			{"stderr cut off", atTerminal, &halfWriter{}},
		} {
			var stdout bytes.Buffer
			in := &readCounter{r: strings.NewReader(op.confirm() + "\n")}
			code := runAs(c.d, &stdout, c.stderr, in, withJSON...)
			what := op.name + " --json, " + c.what
			if code != apperr.ExitEnvironment || in.reads != 0 {
				t.Errorf("%s: exit %d after %d reads, want 6 and none\n%s", what, code, in.reads, stdout.String())
			}
			errorV1(t, what, wholeJSON(t, stdout.String()), apperr.Environment)
			if b, ok := c.stderr.(*bytes.Buffer); ok && b.Len() != 0 {
				t.Errorf("%s: stderr %q", what, b.String())
			}
		}
		// Without JSON, or with --json=false: the confirmation on stdout.
		for _, args := range [][]string{op.args(), append(op.args(), "--json=false")} {
			code, stdout, stderr := asHuman(t, dir, "", args...)
			if code != apperr.ExitHumanRequired || !strings.Contains(stdout, "to confirm") || strings.Contains(stderr, "to confirm") ||
				!strings.HasPrefix(stderr, "error: ") {
				t.Errorf("%s %v: want the confirmation on stdout and a text error\nstdout %q\nstderr %q", op.name, args[len(args)-1], stdout, stderr)
			}
		}
		// The Agent channel: exit 3, no confirmation anywhere.
		code, out, errOut := specmetry(t, dir, "", withJSON...)
		if code != apperr.ExitHumanRequired || strings.Contains(out+errOut, "to confirm") || errOut != "" {
			t.Errorf("%s --json, Agent: exit %d\n%s%s", op.name, code, out, errOut)
		}
		errorV1(t, op.name+" --json, Agent", wholeJSON(t, out), apperr.HumanRequired)
	}
	if after := confirmState(t, home); after != before {
		t.Fatalf("written without a confirmation:\n%s\n%s", before, after)
	}

	// Confirmed with --json: the confirmation on stderr, the output alone
	// on stdout, and the operation done.
	val, err := schemas()
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range ops {
		v := version()
		code, stdout, stderr := asHuman(t, dir, op.confirm(), append(op.args(), "--json")...)
		if code != apperr.ExitOK || !strings.Contains(stderr, "to confirm") || strings.Contains(stdout, "to confirm") {
			t.Fatalf("%s --json, confirmed: exit %d\nstdout %q\nstderr %q", op.name, code, stdout, stderr)
		}
		if op.success != "" {
			if err := val.ValidateJSON(op.success, []byte(wholeJSON(t, stdout))); err != nil {
				t.Errorf("%s: stdout is not %s: %v\n%s", op.name, op.success, err, stdout)
			}
		} else if strings.HasPrefix(strings.TrimSpace(stdout), "{") || stdout == "" {
			t.Errorf("%s: want the success text on stdout, got %q", op.name, stdout)
		}
		if (op.name == "review" || op.name == "propose --apply") && version() != v+1 {
			t.Errorf("%s: baseline version %d, want %d", op.name, version(), v+1)
		}
	}
	got := rows(t, home, `SELECT
    (SELECT group_concat(key || ':' || status, ' ') FROM (SELECT key, status FROM questions WHERE key IN ('Q-001', 'Q-002') ORDER BY key)),
    (SELECT acked_sha256 FROM sources WHERE path = 'docs/client-brief.md' AND removed_seq IS NULL),
    (SELECT count(*) FROM sources WHERE path = 'docs/notes.md' AND removed_seq IS NOT NULL),
    (SELECT count(*) FROM sources WHERE path = 'assets/logo.bin' AND removed_seq IS NULL),
    (SELECT max(number) FROM baseline_versions)`)
	if want := [][]string{{"Q-001:answered Q-002:dismissed", briefSHA, "1", "1", "4"}}; !equalRows(got, want) {
		t.Fatalf("after the confirmations: %v, want %v", got, want)
	}
}

// codex-review-031 N2 in a real terminal (the channel, stdin, stdout and
// stderr as the process has them):
//   - with --json the confirmation is shown (on stderr, which is the
//     terminal) and confirming answers the question; cancelling is exit 3
//     with error.v1;
//   - with stderr redirected to a file, --json is exit 6 before anything is
//     shown or read, with error.v1 on stdout and nothing on stderr;
//     without --json the confirmation is on stdout as before and confirms.
func TestConfirmationWithJSONInTerminal(t *testing.T) {
	home, dir, _ := questionsProject(t)
	out := inTerminal(t, dir, []string{"question", "answer", "Q-001", "-m", "CSV", "--json"}, "Q-001")
	if !strings.Contains(out, "exit=0") || !strings.Contains(out, "to confirm") {
		t.Fatalf("confirmed with --json:\n%s", out)
	}
	out = inTerminal(t, dir, []string{"question", "dismiss", "Q-002", "-m", "later", "--json"}, "")
	if !strings.Contains(out, "exit=3") || !strings.Contains(out, "to confirm") || !strings.Contains(out, `"code": "human_required"`) {
		t.Fatalf("cancelled with --json:\n%s", out)
	}
	stderrRedirected := func(args []string, answer string) string {
		t.Helper()
		raw, err := json.Marshal(args)
		if err != nil {
			t.Fatal(err)
		}
		helper := testutil.HelperCommand(t, "cli-stderr-redirected")
		var steps []testutil.PTYStep
		if answer != noPrompt {
			steps = []testutil.PTYStep{{Expect: "press Enter to cancel:", Send: answer + "\r"}}
		}
		return testutil.RunInPTYSession(t, 60*time.Second, helperEnv(t, "cli-stderr-redirected", argsJSONEnv+"="+string(raw), dirEnv+"="+dir),
			steps, helper.Path, helper.Args[1:]...).Output
	}
	before := confirmState(t, home)
	out = stderrRedirected([]string{"question", "answer", "Q-003", "-m", "never", "--json"}, noPrompt)
	if !strings.Contains(out, "exit=6") || strings.Contains(out, "to confirm") || !strings.Contains(out, "stderr=0 bytes") ||
		!strings.Contains(out, `"code": "environment"`) {
		t.Fatalf("stderr redirected, with --json:\n%s", out)
	}
	if after := confirmState(t, home); after != before {
		t.Fatalf("written:\n%s\n%s", before, after)
	}
	out = stderrRedirected([]string{"question", "answer", "Q-003", "-m", "kept"}, "Q-003")
	if !strings.Contains(out, "exit=0") || !strings.Contains(out, "to confirm") {
		t.Fatalf("stderr redirected, without --json:\n%s", out)
	}
	got := rows(t, home, `SELECT key, status FROM questions WHERE key IN ('Q-001', 'Q-002', 'Q-003') ORDER BY key`)
	if want := [][]string{{"Q-001", "answered"}, {"Q-002", "open"}, {"Q-003", "answered"}}; !equalRows(got, want) {
		t.Fatalf("questions = %v, want %v", got, want)
	}
}
