package cli

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
)

// controls lists the bytes and characters of s that a terminal may act on
// or that change the reading order: C0 controls other than LF and TAB,
// DEL, C1 controls, bidirectional controls, the line and paragraph
// separators, and bytes that are not UTF-8.
func controls(s string) []string {
	var found []string
	for i, r := range s {
		switch {
		case r == utf8.RuneError && !strings.HasPrefix(s[i:], "\uFFFD"):
			found = append(found, "byte "+strconv.Itoa(int(s[i])))
		case r < 0x20 && r != '\n' && r != '\t', r == 0x7f, r >= 0x80 && r <= 0x9f,
			r == 0x061c, r == 0x200e, r == 0x200f, r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069,
			r == 0x2028, r == 0x2029:
			found = append(found, strconv.QuoteRune(r))
		}
	}
	return found
}

// shownSafely checks that out holds no control character, and shows
// each of want in its visible form.
func shownSafely(t *testing.T, what, out string, want ...string) {
	t.Helper()
	if c := controls(out); len(c) > 0 {
		t.Errorf("%s writes control characters %v to the terminal:\n%q", what, c, out)
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("%s does not show %s:\n%s", what, w, out)
		}
	}
}

// codex-review-027 B1: text from a proposal (a question and its context),
// from the person (the answer, the reason) and from the configuration
// (the display name) reaches the terminal with its control characters
// shown, never as they are: in question list, and in the confirmation of
// question answer and question dismiss. A line feed stays a line break
// (CRLF one too) under the indentation; nothing is dropped. The stored
// values and question list --json keep the text as it is, and a
// cancelled confirmation writes nothing. Everything is read from memory,
// never from a real terminal.
func TestQuestionTextIsShownSafely(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	name := "ops\x1b]0;owned\x07\u202e"
	if err := os.WriteFile(home.Join(datadir.ConfigFile), []byte(`[user]
display_name = "ops\u001b]0;owned\u0007\u202e"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	text := "clear\x1b[2J\x1b[H here\rforged\x07 bell\b back\nnext line"
	context := "osc \x1b]0;title\x07 del\x7f c1\u009b[31m bidi \u202eevil crlf\r\nend"
	doc := small(0, addReq("r1", "export", nil))
	doc["questions"] = []any{
		map[string]any{"text": text, "context": context, "about": "r1", "blocking": true},
		map[string]any{"text": "up\x1b[1A", "blocking": false},
	}
	proposeOK(t, dir, doc)
	shownText := []string{`clear\x1b[2J\x1b[H here\rforged\a bell\b back`, "\n    next line", `osc \x1b]0;title\a del\x7f c1\u009b[31m bidi \u202eevil crlf`,
		"crlf\n    end", `up\x1b[1A`}

	code, out, stderr := specmetry(t, dir, "", "question", "list")
	if code != apperr.ExitOK {
		t.Fatalf("question list: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "question list", out+stderr, shownText...)

	answer := "yes\x1b[31m red\x1b[0m\rnope\x07"
	reason := "dup\x1b[2K\b\b"
	shownName := `human:ops\x1b]0;owned\a\u202e`
	before := closeState(t, home)
	code, out, stderr = asHuman(t, dir, "", "question", "answer", "Q-001", "-m", answer)
	if code != apperr.ExitHumanRequired || !strings.Contains(out, "to confirm") {
		t.Fatalf("cancelled answer: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "the answer's confirmation", out+stderr, append(shownText[:4:4], `yes\x1b[31m red\x1b[0m\rnope\a`, shownName)...)
	code, out, stderr = asHuman(t, dir, "", "question", "dismiss", "Q-002", "-m", reason)
	if code != apperr.ExitHumanRequired || !strings.Contains(out, "to confirm") {
		t.Fatalf("cancelled dismissal: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "the dismissal's confirmation", out+stderr, `up\x1b[1A`, `dup\x1b[2K\b\b`, shownName)
	if after := closeState(t, home); strings.Join(after, " ") != strings.Join(before, " ") {
		t.Fatalf("a cancelled confirmation wrote:\n%v\n%v", before, after)
	}

	code, out, stderr = asHuman(t, dir, "Q-001", "question", "answer", "Q-001", "-m", answer)
	if code != apperr.ExitOK {
		t.Fatalf("answer: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "question answer", out+stderr)
	if code, out, stderr = asHuman(t, dir, "Q-002", "question", "dismiss", "Q-002", "-m", reason); code != apperr.ExitOK {
		t.Fatalf("dismiss: exit %d\n%s%s", code, out, stderr)
	}
	code, out, stderr = specmetry(t, dir, "", "question", "list", "--all")
	if code != apperr.ExitOK {
		t.Fatalf("question list --all: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "question list --all", out+stderr, append(shownText, `yes\x1b[31m red\x1b[0m\rnope\a`, `dup\x1b[2K\b\b`, shownName)...)

	// The data keep the text as it is: question list --json (standard JSON
	// encoding) and the database.
	l, _ := questionList(t, dir, "--all")
	q1, q2 := l.Questions[0], l.Questions[1]
	if q1.Text != text || q1.Context == nil || *q1.Context != context || q1.Answer == nil || q1.Answer.Text != answer ||
		q1.Answer.By != "human:"+name || q2.Answer == nil || q2.Answer.Text != reason || q2.Text != "up\x1b[1A" {
		t.Fatalf("question list --json changed the text: %+v %+v", q1, q2)
	}
	got := rows(t, home, `SELECT text, context, answer, answered_by FROM questions WHERE key = 'Q-001'`)
	if want := [][]string{{text, context, answer, "human:" + name}}; !equalRows(got, want) {
		t.Fatalf("stored %q, want %q", got, want)
	}
}

// errorV1 checks that stdout is a whole error.v1 document with code, and
// decodes it.
func errorV1(t *testing.T, what, stdout string, code apperr.Code) apperr.Envelope {
	t.Helper()
	validator, err := schemas()
	if err != nil {
		t.Fatal(err)
	}
	var env apperr.Envelope
	if err := validator.ValidateJSON("error.v1", []byte(stdout)); err != nil {
		t.Errorf("%s: stdout is not error.v1: %v\n%s", what, err, stdout)
		return env
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil || env.Error.Code != code {
		t.Errorf("%s: error.v1 code %s, want %s (%v)", what, env.Error.Code, code, err)
	}
	return env
}

// codex-review-027 B2 and B3, through the real command entry:
//   - an unknown question command is invalid_input (exit 2), not help and
//     exit 0, and does not quote the name; with --json it is error.v1;
//   - question alone and --help still show the help (exit 0);
//   - an error before --json is parsed still prints error.v1 when the
//     arguments ask for JSON: --json and --json=true count, the last one
//     wins, a value of -m, --message or --file does not count, nor
//     anything after --, nor an unknown flag's own value;
//   - without JSON the error is text on stderr and stdout is empty.
//
// Nothing is read or written, and no seq is taken.
func TestQuestionCommandLineErrors(t *testing.T) {
	home, dir, _ := questionsProject(t)
	token := "ghp_" + strings.Repeat("A1b2", 9) // a GitHub token to the secret rules
	before := closeState(t, home)
	for _, c := range []struct {
		args   []string
		asJSON bool
		exit   int
	}{
		{[]string{"unknown"}, false, apperr.ExitInvalidInput},
		{[]string{"answr"}, false, apperr.ExitInvalidInput},
		{[]string{token}, false, apperr.ExitInvalidInput},
		{[]string{"unknown", "--json"}, true, apperr.ExitInvalidInput},
		{[]string{"answr", "--json"}, true, apperr.ExitInvalidInput},
		{[]string{"--json", token}, true, apperr.ExitInvalidInput},
		{[]string{"--json"}, true, apperr.ExitInvalidInput},
		{[]string{"--" + token, "unknown", "--json"}, true, apperr.ExitInvalidInput},
		{[]string{"list", "--all=not-bool", "--json"}, true, apperr.ExitInvalidInput},
		{[]string{"list", "--json", "--all=not-bool"}, true, apperr.ExitInvalidInput},
		{[]string{"list", "--all=" + token, "--json"}, true, apperr.ExitInvalidInput},
		{[]string{"list", "--json=true", "--all=not-bool"}, true, apperr.ExitInvalidInput},
		{[]string{"list", "--json=false", "--json", "--all=not-bool"}, true, apperr.ExitInvalidInput},
		{[]string{"answer", "Q-001", "-m", "x", "--unknown", "--json"}, true, apperr.ExitInvalidInput},
		{[]string{"answer", "Q-001", "-mx", "--" + token, "--json"}, true, apperr.ExitInvalidInput},
		{[]string{"dismiss", "--file", "x.json", "--unknown", "--json"}, true, apperr.ExitInvalidInput},
		{[]string{"list", "--json=false", "--all=not-bool"}, false, apperr.ExitInvalidInput},
		{[]string{"list", "--json", "--json=false", "--all=not-bool"}, false, apperr.ExitInvalidInput},
		{[]string{"list", "--json=maybe"}, false, apperr.ExitInvalidInput},
		{[]string{"answer", "Q-001", "-m=--json", "--unknown"}, false, apperr.ExitInvalidInput},
		{[]string{"answer", "Q-001", "--message", "--json", "--unknown"}, false, apperr.ExitInvalidInput},
		{[]string{"answer", "--file", "--json", "--unknown"}, false, apperr.ExitInvalidInput},
		{[]string{"answer", "Q-001", "-m", "x", "--unknown", "--", "--json"}, false, apperr.ExitInvalidInput},
		{[]string{"list", "--", "--json"}, false, apperr.ExitInvalidInput},
		{[]string{"list", "--unknown=--json"}, false, apperr.ExitInvalidInput},
		// -m takes --json as the answer: parsed, not JSON, and the Agent
		// channel needs a person.
		{[]string{"answer", "Q-001", "-m", "--json"}, false, apperr.ExitHumanRequired},
	} {
		args := append([]string{"question"}, c.args...)
		code, stdout, stderr := specmetry(t, dir, "", args...)
		what := strings.ReplaceAll(strings.Join(args, " "), token, "<token>")
		if code != c.exit {
			t.Errorf("%s: exit %d, want %d\n%s%s", what, code, c.exit, stdout, stderr)
		}
		if strings.Contains(stdout+stderr, token) {
			t.Errorf("%s echoes its input:\n%s%s", what, stdout, stderr)
		}
		if c.asJSON {
			errorV1(t, what, stdout, apperr.InvalidInput)
			if stderr != "" {
				t.Errorf("%s: stderr with error.v1:\n%s", what, stderr)
			}
			continue
		}
		if stdout != "" || !strings.HasPrefix(stderr, "error: ") {
			t.Errorf("%s: want a text error only, got stdout %q stderr %q", what, stdout, stderr)
		}
	}
	// A suggestion names an existing command only.
	if _, _, stderr := specmetry(t, dir, "", "question", "answr"); !strings.Contains(stderr, "did you mean answer?") {
		t.Errorf("question answr: %s", stderr)
	}
	for _, args := range [][]string{{"question"}, {"question", "--help"}, {"question", "list", "--help"}} {
		code, stdout, stderr := specmetry(t, dir, "", args...)
		if code != apperr.ExitOK || !strings.Contains(stdout, "Usage:") {
			t.Errorf("%v: exit %d\n%s%s", args, code, stdout, stderr)
		}
	}
	if after := closeState(t, home); strings.Join(after, " ") != strings.Join(before, " ") {
		t.Fatalf("written:\n%v\n%v", before, after)
	}
}
