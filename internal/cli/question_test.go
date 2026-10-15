package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

type questionListOut struct {
	Questions []struct {
		Key        string  `json:"key"`
		Text       string  `json:"text"`
		Context    *string `json:"context"`
		Status     string  `json:"status"`
		Blocking   bool    `json:"blocking"`
		About      *string `json:"about"`
		ProposalID string  `json:"proposal_id"`
		CreatedSeq int64   `json:"created_seq"`
		Answer     *struct {
			Action  string `json:"action"`
			Text    string `json:"text"`
			By      string `json:"by"`
			Channel string `json:"channel"`
			Seq     int64  `json:"seq"`
		} `json:"answer"`
	} `json:"questions"`
}

// questionsProject is a project at baseline version 2: REQ-001 from a
// first proposal, then a second proposal (returned) adding r1 and r2,
// with four questions, reviewed: r1 accepted as REQ-002, r2 rejected.
//   - Q-001 about r1, so now about REQ-002, blocking, with a context;
//   - Q-002 about the project, not blocking;
//   - Q-003 about r2, rejected: it keeps naming r2, and stays open;
//   - Q-004 about REQ-001, blocking.
func questionsProject(t *testing.T) (datadir.Dir, string, receiptOut) {
	t.Helper()
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	acceptAll(t, dir, proposeOK(t, dir, small(0, addReq("r", "sign up", nil))), 0, nil)
	doc := small(1, addReq("r1", "export", nil), addReq("r2", "import", nil))
	doc["questions"] = []any{
		map[string]any{"text": "e-mail or phone?", "context": "the brief says both", "about": "r1", "blocking": true},
		map[string]any{"text": "how many per page?", "blocking": false},
		map[string]any{"text": "which format?", "about": "r2", "blocking": true},
		map[string]any{"text": "is sign-up open to all?", "about": "REQ-001", "blocking": true},
	}
	r := proposeOK(t, dir, doc)
	ids := itemIDs(r)
	reviewOK(t, dir, reviewDoc(r.ProposalID, 1, decide(ids["r1"], "accepted"), decide(ids["r2"], "rejected")))
	return home, dir, r
}

// closeState is what answering or dismissing writes, and what it must not:
// the seq counter, then every question's status, answer, who, channel,
// seq and relation, then the baseline (fields 2 on).
func closeState(t *testing.T, home datadir.Dir) []string {
	t.Helper()
	// One query: opening the database is what takes time.
	return rows(t, home, `SELECT (SELECT seq_counter FROM meta),
    (SELECT group_concat(q, ' ') FROM (SELECT key || ':' || status || ':' || coalesce(answer, '-') || ':' || coalesce(answered_by, '-') || ':' ||
        coalesce(answered_channel, '-') || ':' || coalesce(answered_seq, '-') || ':' || coalesce(related_type, '-') || ':' ||
        coalesce(related_id, '-') || ':' || coalesce(proposal_item_id, '-') AS q FROM questions ORDER BY key)),
    (SELECT count(*) FROM baseline_versions), (SELECT count(*) FROM baseline_changes), (SELECT count(*) FROM goals),
    (SELECT count(*) || '/' || coalesce(sum(revision), 0) FROM requirements), (SELECT count(*) FROM acceptance_criteria),
    (SELECT count(*) FROM source_refs), (SELECT count(*) FROM code_refs), (SELECT coalesce(sum(next_value), 0) FROM key_counters),
    (SELECT count(*) FROM proposal_items WHERE review_state <> 'pending'), (SELECT count(*) FROM proposals)`)[0]
}

func questionList(t *testing.T, dir string, args ...string) (questionListOut, string) {
	t.Helper()
	code, stdout, stderr := specmetry(t, dir, "question-list.v1", append([]string{"question", "list", "--json"}, args...)...)
	if code != apperr.ExitOK {
		t.Fatalf("question list %v: exit %d\n%s%s", args, code, stdout, stderr)
	}
	var out questionListOut
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatal(err)
	}
	return out, stdout
}

// listed is each listed question as key:status:about:blocking, and its
// answer as action/text/channel/seq.
func (l questionListOut) listed() string {
	var out []string
	for _, q := range l.Questions {
		about := "null"
		if q.About != nil {
			about = *q.About
		}
		s := q.Key + ":" + q.Status + ":" + about + ":" + strconv.FormatBool(q.Blocking)
		if a := q.Answer; a != nil {
			s += ":" + a.Action + "/" + a.Text + "/" + a.Channel + "/" + strconv.FormatInt(a.Seq, 10)
			if !strings.HasPrefix(a.By, "human:") {
				s += "/by " + a.By
			}
		}
		out = append(out, s)
	}
	return strings.Join(out, " ")
}

// MVP-BL-17, FR-BL-10: any channel lists the questions; a person answers
// one and dismisses another, each with one seq, and question list --all
// shows the answer, the reason, who gave them and on which channel. The
// baseline does not change. question list holds no internal IDs. Text
// output too.
func TestQuestionAnswerAndDismiss(t *testing.T) {
	home, dir, r := questionsProject(t)
	l, _ := questionList(t, dir)
	want := "Q-001:open:REQ-002:true Q-002:open:null:false Q-003:open:r2:true Q-004:open:REQ-001:true"
	if got := l.listed(); got != want {
		t.Fatalf("open questions:\n%s\nwant\n%s", got, want)
	}
	for _, q := range l.Questions {
		if q.ProposalID != r.ProposalID || q.CreatedSeq != r.Seq {
			t.Errorf("%s from %s at seq %d, want %s at %d", q.Key, q.ProposalID, q.CreatedSeq, r.ProposalID, r.Seq)
		}
	}
	if q := l.Questions[0]; q.Text != "e-mail or phone?" || q.Context == nil || *q.Context != "the brief says both" || l.Questions[1].Context != nil {
		t.Errorf("Q-001 = %+v", q)
	}
	before := closeState(t, home)
	seq0, _ := strconv.Atoi(before[0])

	code, stdout, stderr := asHuman(t, dir, "Q-001", "question", "answer", "Q-001", "-m", "e-mail only")
	if code != apperr.ExitOK || !strings.Contains(stdout, "Answered Q-001 (seq "+strconv.Itoa(seq0+1)+")") ||
		!strings.Contains(stdout, "answer: e-mail only") || !strings.Contains(stdout, "asks: e-mail or phone?") {
		t.Fatalf("answer: exit %d\n%s%s", code, stdout, stderr)
	}
	code, stdout, stderr = asHuman(t, dir, "Q-003", "question", "dismiss", "Q-003", "-m", "the import was rejected")
	if code != apperr.ExitOK || !strings.Contains(stdout, "Dismissed Q-003 (seq "+strconv.Itoa(seq0+2)+")") ||
		!strings.Contains(stdout, "dismissed because: the import was rejected") {
		t.Fatalf("dismiss: exit %d\n%s%s", code, stdout, stderr)
	}

	// The database, with SQL of its own.
	got := rows(t, home, `SELECT key, status, coalesce(answer, '-'), substr(coalesce(answered_by, '-'), 1, 6), coalesce(answered_channel, '-'),
    coalesce(answered_seq - ?, '-') FROM questions ORDER BY key`, seq0)
	wantRows := [][]string{
		{"Q-001", "answered", "e-mail only", "human:", "human_cli", "1"},
		{"Q-002", "open", "-", "-", "-", "-"},
		{"Q-003", "dismissed", "the import was rejected", "human:", "human_cli", "2"},
		{"Q-004", "open", "-", "-", "-", "-"},
	}
	if !equalRows(got, wantRows) {
		t.Fatalf("questions = %v, want %v", got, wantRows)
	}
	after := closeState(t, home)
	if after[0] != strconv.Itoa(seq0+2) || strings.Join(after[2:], " ") != strings.Join(before[2:], " ") {
		t.Fatalf("written besides the two questions and two seqs:\n%v\n%v", before, after)
	}

	l, raw := questionList(t, dir)
	if got := l.listed(); got != "Q-002:open:null:false Q-004:open:REQ-001:true" {
		t.Fatalf("open questions now: %s", got)
	}
	l, rawAll := questionList(t, dir, "--all")
	s := func(n int) string { return strconv.Itoa(seq0 + n) }
	want = "Q-001:answered:REQ-002:true:answer/e-mail only/human_cli/" + s(1) + " Q-002:open:null:false " +
		"Q-003:dismissed:r2:true:dismiss/the import was rejected/human_cli/" + s(2) + " Q-004:open:REQ-001:true"
	if got := l.listed(); got != want {
		t.Fatalf("all questions:\n%s\nwant\n%s", got, want)
	}

	code, text, stderr := specmetry(t, dir, "", "question", "list", "--all")
	if code != apperr.ExitOK {
		t.Fatalf("question list --all: exit %d\n%s%s", code, text, stderr)
	}
	for _, line := range []string{
		"4 questions: 2 open, 1 answered, 1 dismissed.",
		"Q-001  answered, blocking, about REQ-002 (proposal " + r.ProposalID + ", seq " + strconv.FormatInt(r.Seq, 10) + ")",
		"  e-mail or phone?\n  context: the brief says both\n  answered by human:",
		"(human_cli) at seq " + s(1) + ": e-mail only",
		"Q-002  open, not blocking, about the project",
		"Q-003  dismissed, blocking, about ref r2 of its proposal",
		"(human_cli) at seq " + s(2) + ": the import was rejected",
		"Only a person answers or dismisses a question",
	} {
		if !strings.Contains(text, line) {
			t.Errorf("question list --all has no %q:\n%s", line, text)
		}
	}
	code, open, _ := specmetry(t, dir, "", "question", "list")
	if code != apperr.ExitOK || !strings.HasPrefix(open, "2 open questions.") || strings.Contains(open, "\nQ-001  ") {
		t.Errorf("question list: exit %d\n%q", code, open)
	}

	// Internal IDs stay inside: the questions', their entities' and items'.
	for _, id := range rows(t, home, `SELECT id, coalesce(related_id, id), coalesce(proposal_item_id, id) FROM questions`) {
		for _, out := range []string{raw, rawAll, text, open} {
			for _, v := range id {
				if strings.Contains(out, v) {
					t.Errorf("question list shows the internal ID %s", v)
				}
			}
		}
	}
}

// A question is answered or dismissed once: then either action is stale
// (exit 5) before anybody is asked, with its status and seq but not its
// answer, and nothing is written. An unknown question is not_found (exit
// 2). The WP1.6 review still relates the open questions only.
func TestQuestionClosesOnce(t *testing.T) {
	home, dir, _ := questionsProject(t)
	if code, stdout, stderr := asHuman(t, dir, "Q-001", "question", "answer", "Q-001", "-m", "e-mail only"); code != apperr.ExitOK {
		t.Fatalf("answer: exit %d\n%s%s", code, stdout, stderr)
	}
	if code, stdout, stderr := asHuman(t, dir, "Q-002", "question", "dismiss", "Q-002", "-m", "twenty, as now"); code != apperr.ExitOK {
		t.Fatalf("dismiss: exit %d\n%s%s", code, stdout, stderr)
	}
	before := closeState(t, home)
	seq0, _ := strconv.Atoi(before[0])
	for _, c := range []struct {
		args   []string
		status string
		seq    int
	}{
		{[]string{"answer", "Q-001", "-m", "phone"}, "answered", seq0 - 1},
		{[]string{"dismiss", "Q-001", "-m", "never mind"}, "answered", seq0 - 1},
		{[]string{"answer", "Q-002", "-m", "ten"}, "dismissed", seq0},
		{[]string{"dismiss", "Q-002", "-m", "again"}, "dismissed", seq0},
	} {
		key := c.args[1]
		code, stdout, stderr := asHuman(t, dir, key, append(append([]string{"question"}, c.args...), "--json")...)
		if code != apperr.ExitStalePrecondition || strings.Contains(stdout, "to confirm") {
			t.Errorf("%v: exit %d, want 5 before any prompt\n%s%s", c.args, code, stdout, stderr)
			continue
		}
		var env apperr.Envelope
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("%v: no error.v1: %v\n%s", c.args, err, stdout)
		}
		cur := env.Error.Current
		if cur["question"] != key || cur["status"] != c.status || cur["answered_seq"] != float64(c.seq) {
			t.Errorf("%v: current = %v", c.args, cur)
		}
		if strings.Contains(stdout, "e-mail only") || strings.Contains(stdout, "twenty, as now") {
			t.Errorf("%v: the error shows the answer:\n%s", c.args, stdout)
		}
	}
	code, stdout, _ := asHuman(t, dir, noPrompt, "question", "answer", "Q-099", "-m", "x", "--json")
	var env apperr.Envelope
	if err := json.Unmarshal([]byte(stdout), &env); code != apperr.ExitInvalidInput || err != nil || env.Error.Code != "not_found" {
		t.Errorf("unknown question: exit %d\n%s", code, stdout)
	}
	if after := closeState(t, home); strings.Join(after, " ") != strings.Join(before, " ") {
		t.Fatalf("written:\n%v\n%v", before, after)
	}
}

// A question answered while its item waits for review keeps naming the
// item: accepting the item later relates only the open questions (the
// review is not stopped by the closed one), and the list shows the key the
// item got.
func TestQuestionAnsweredBeforeItsItem(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	doc := small(0, addReq("r1", "export", nil))
	doc["questions"] = []any{
		map[string]any{"text": "CSV or JSON?", "about": "r1", "blocking": true},
		map[string]any{"text": "how often?", "about": "r1", "blocking": false},
	}
	r := proposeOK(t, dir, doc)
	if code, stdout, stderr := asHuman(t, dir, "Q-001", "question", "answer", "Q-001", "-m", "CSV"); code != apperr.ExitOK {
		t.Fatalf("answer: exit %d\n%s%s", code, stdout, stderr)
	}
	if l, _ := questionList(t, dir, "--all"); !strings.HasPrefix(l.listed(), "Q-001:answered:r1:true") {
		t.Fatalf("before the review: %s", l.listed())
	}
	reviewOK(t, dir, reviewDoc(r.ProposalID, 0, decide(r.Items[0].ItemID, "accepted")))
	got := rows(t, home, `SELECT key, status, coalesce(related_type, '-') FROM questions ORDER BY key`)
	if want := [][]string{{"Q-001", "answered", "-"}, {"Q-002", "open", "requirement"}}; !equalRows(got, want) {
		t.Fatalf("questions = %v, want %v", got, want)
	}
	l, _ := questionList(t, dir, "--all")
	if got := l.listed(); !strings.HasPrefix(got, "Q-001:answered:REQ-001:true:answer/CSV/human_cli/") ||
		!strings.HasSuffix(got, "Q-002:open:REQ-001:false") {
		t.Fatalf("after the review: %s", got)
	}
}

// MVP-BL-17, PD-14: the Agent channel cannot answer or dismiss: exit 3,
// with --json error.v1, before anything given is read (a --file that
// does not exist is still exit 3), no prompt, nothing written; the -m
// text and the file's content (a canary) are nowhere.
func TestQuestionNeedsAPerson(t *testing.T) {
	home, dir, _ := questionsProject(t)
	canary := testutil.NewCanary(t)
	file := filepath.Join(t.TempDir(), "answer.json")
	if err := os.WriteFile(file, []byte(`{"schema":"specmetry.question-answer/v1","question":"Q-001","action":"answer","answer":"`+canary+`"}`),
		0o600); err != nil {
		t.Fatal(err)
	}
	before := closeState(t, home)
	for _, args := range [][]string{
		{"question", "answer", "Q-001", "-m", canary},
		{"question", "dismiss", "Q-001", "-m", canary},
		{"question", "answer", "--file", file},
		{"question", "dismiss", "--file", filepath.Join(t.TempDir(), canary+".json")},
	} {
		for _, asJSON := range []bool{false, true} {
			run := slicesWith(args, asJSON)
			code, stdout, stderr := specmetry(t, dir, "", run...)
			if code != apperr.ExitHumanRequired || strings.Contains(stdout+stderr, canary) || strings.Contains(stdout, "to confirm") {
				t.Errorf("%v: exit %d\n%s%s", run, code, stdout, stderr)
			}
			if asJSON {
				var env apperr.Envelope
				if err := json.Unmarshal([]byte(stdout), &env); err != nil || env.Error.Code != "human_required" {
					t.Errorf("%v: no human_required error.v1: %v\n%s", run, err, stdout)
				}
			}
		}
	}
	if after := closeState(t, home); strings.Join(after, " ") != strings.Join(before, " ") {
		t.Fatalf("written:\n%v\n%v", before, after)
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the canary reached the data directory: %v", hits)
	}
}

// MVP-BL-17, MVP-CH-01 and CH-02 with the real question commands in a
// pseudo-terminal:
//   - an Agent's variable set (its value a canary): exit 3, no prompt;
//   - stdout redirected: exit 3;
//   - Enter alone, or another question's key: exit 3, not confirmed;
//   - nothing written in all these cases, the canary nowhere;
//   - the question's key typed: answered, then another dismissed, by a
//     human: actor on human_cli.
func TestQuestionInTerminal(t *testing.T) {
	home, dir, _ := questionsProject(t)
	args := []string{"question", "answer", "Q-001", "-m", "e-mail only"}
	canary := testutil.NewCanary(t)
	before := closeState(t, home)
	out := inTerminal(t, dir, args, noPrompt, "CLAUDECODE="+canary)
	if !strings.Contains(out, "exit=3") || strings.Contains(out, "to confirm") || strings.Contains(out, canary) {
		t.Errorf("with CLAUDECODE:\n%s", out)
	}
	raw, err := json.Marshal(append(args, "--json"))
	if err != nil {
		t.Fatal(err)
	}
	redirected := testutil.HelperCommand(t, "cli-redirected")
	out = testutil.RunInPTYSession(t, 60*time.Second, helperEnv(t, "cli-redirected", argsJSONEnv+"="+string(raw), dirEnv+"="+dir),
		nil, redirected.Path, redirected.Args[1:]...).Output
	if !strings.Contains(out, "exit=3") || strings.Contains(out, "to confirm") {
		t.Errorf("stdout redirected:\n%s", out)
	}
	for _, answer := range []string{"", "Q-002"} {
		if out := inTerminal(t, dir, args, answer); !strings.Contains(out, "exit=3") || !strings.Contains(out, "not confirmed") {
			t.Errorf("answered %q at the prompt:\n%s", answer, out)
		}
	}
	if after := closeState(t, home); strings.Join(after, " ") != strings.Join(before, " ") {
		t.Fatalf("written:\n%v\n%v", before, after)
	}

	if out := inTerminal(t, dir, args, "Q-001"); !strings.Contains(out, "exit=0") || !strings.Contains(out, "Answered Q-001") {
		t.Fatalf("confirmed:\n%s", out)
	}
	out = inTerminal(t, dir, []string{"question", "dismiss", "Q-003", "-m", "the import was rejected"}, "Q-003")
	if !strings.Contains(out, "exit=0") || !strings.Contains(out, "Dismissed Q-003") {
		t.Fatalf("dismissed:\n%s", out)
	}
	got := rows(t, home, `SELECT key, status, substr(answered_by, 1, 6), answered_channel FROM questions WHERE status <> 'open' ORDER BY key`)
	if want := [][]string{{"Q-001", "answered", "human:", "human_cli"}, {"Q-003", "dismissed", "human:", "human_cli"}}; !equalRows(got, want) {
		t.Fatalf("questions = %v, want %v", got, want)
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the canary reached the data directory: %v", hits)
	}
}

// closeIn runs a question command in this process in the current
// directory as a person who confirms; for a callback outside the test's
// goroutine (no t.Chdir, no t.Fatal).
func closeIn(args ...string) (int, string) {
	var out, errOut bytes.Buffer
	root := newRoot(&out, &errOut)
	root.SetIn(strings.NewReader(args[2] + "\n"))
	code := executeAs(context.Background(), channel.Detection{Channel: channel.HumanCLI}, root, args, &errOut)
	return code, out.String() + errOut.String()
}

// Another connection closes the question while the person confirms: the
// confirmed answer is stale (exit 5) and does not overwrite it; one seq
// is taken, by the other. In this process with another connection, and
// with the confirming command in another process, at a real terminal.
func TestQuestionClosedWhileConfirming(t *testing.T) {
	home, dir, _ := questionsProject(t)
	before := closeState(t, home)
	seq0, _ := strconv.Atoi(before[0])
	var other int
	var otherOut string
	in := &onFirstRead{r: strings.NewReader("Q-001\n"), before: func() {
		other, otherOut = closeIn("question", "dismiss", "Q-001", "-m", "asked twice")
	}}
	code, stdout, stderr := asHumanWith(t, dir, in, "question", "answer", "Q-001", "-m", "e-mail only", "--json")
	if other != apperr.ExitOK {
		t.Fatalf("the other connection: exit %d\n%s", other, otherOut)
	}
	if code != apperr.ExitStalePrecondition || !strings.Contains(stdout, "to confirm") {
		t.Fatalf("exit %d, want 5 after the confirmation\n%s%s", code, stdout, stderr)
	}
	var env apperr.Envelope
	if err := json.Unmarshal([]byte(jsonPart(stdout)), &env); err != nil || env.Error.Current["status"] != "dismissed" {
		t.Fatalf("error = %+v (%v)", env.Error, err)
	}

	out := inTerminalChanging(t, dir, []string{"question", "answer", "Q-002", "-m", "twenty"}, "Q-002", func() {
		if code, out := closeIn("question", "dismiss", "Q-002", "-m", "as now"); code != apperr.ExitOK {
			t.Errorf("the other process's answer was refused: exit %d\n%s", code, out)
		}
	})
	if !strings.Contains(out, "exit=5") {
		t.Fatalf("closed during the prompt:\n%s", out)
	}
	got := rows(t, home, `SELECT key, status, answer, answered_seq - ? FROM questions WHERE status <> 'open' ORDER BY key`, seq0)
	if want := [][]string{{"Q-001", "dismissed", "asked twice", "1"}, {"Q-002", "dismissed", "as now", "2"}}; !equalRows(got, want) {
		t.Fatalf("questions = %v, want %v", got, want)
	}
	if after := closeState(t, home); after[0] != strconv.Itoa(seq0+2) || strings.Join(after[2:], " ") != strings.Join(before[2:], " ") {
		t.Fatalf("written:\n%v\n%v", before, after)
	}
}

// The input is checked before anybody is asked, and refused with exit 2,
// nothing written, its values not echoed (a token to the secret rules):
// a missing question or text, an empty text, a malformed or unknown key,
// a secret or a code fence in the text, a file whose action is the other
// command's or that mixes the two, a file together with an argument, a
// file that cannot be read, an unknown flag, an argument to question
// list. A file that is right is answered and dismissed.
func TestQuestionInputIsChecked(t *testing.T) {
	home, dir, _ := questionsProject(t)
	token := "ghp_" + strings.Repeat("A1b2", 9) // a GitHub token to the secret rules
	tmp := t.TempDir()
	write := func(name string, doc map[string]any) string {
		doc["schema"] = "specmetry.question-answer/v1"
		b, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(tmp, name)
		if err := os.WriteFile(p, b, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	answerQ1 := write("answer.json", map[string]any{"question": "Q-001", "action": "answer", "answer": "e-mail only"})
	dismissQ2 := write("dismiss.json", map[string]any{"question": "Q-002", "action": "dismiss", "rationale": "twenty, as now"})
	mixed := write("mixed.json", map[string]any{"question": "Q-001", "action": "answer", "answer": "x", "rationale": "y"})
	secret := write("secret.json", map[string]any{"question": "Q-001", "action": "answer", "answer": "use " + token})
	before := closeState(t, home)
	for _, c := range []struct {
		args    []string
		pointer string
	}{
		{[]string{"answer"}, ""},
		{[]string{"answer", "Q-001"}, ""},
		{[]string{"answer", "Q-001", "-m", ""}, "/answer"},
		{[]string{"dismiss", "Q-001"}, ""},
		{[]string{"dismiss", "Q-001", "-m", ""}, "/rationale"},
		{[]string{"answer", "Q-1", "-m", "x"}, "/question"},
		{[]string{"answer", token, "-m", "x"}, "/question"},
		{[]string{"answer", "Q-099", "-m", "x"}, ""},
		{[]string{"answer", "Q-001", "-m", "use " + token}, "/answer"},
		{[]string{"dismiss", "Q-001", "-m", "see\n```\ncode\n```"}, "/rationale"},
		{[]string{"answer", "--file", dismissQ2}, "/action"},
		{[]string{"dismiss", "--file", answerQ1}, "/action"},
		{[]string{"answer", "--file", mixed}, ""},
		{[]string{"answer", "Q-001", "--file", answerQ1}, ""},
		{[]string{"answer", "--file", answerQ1, "-m", "x"}, ""},
		{[]string{"answer", "--file", secret}, "/answer"},
		{[]string{"answer", "--file", filepath.Join(tmp, token+".json")}, ""},
		{[]string{"answer", "Q-001", "-m", "x", "--" + token}, ""},
		{[]string{"list", token}, ""},
		{[]string{"list", "--all=" + token}, ""},
	} {
		for _, asJSON := range []bool{false, true} {
			run := slicesWith(append([]string{"question"}, c.args...), asJSON)
			code, stdout, stderr := asHuman(t, dir, noPrompt, run...)
			if code != apperr.ExitInvalidInput || strings.Contains(stdout, "to confirm") {
				t.Errorf("%v: exit %d, want 2 before any prompt\n%s%s", run, code, stdout, stderr)
			}
			if strings.Contains(stdout+stderr, token) {
				t.Errorf("%v echoes its input:\n%s%s", run, stdout, stderr)
			}
			if !asJSON {
				continue
			}
			// Every --json run prints a whole error.v1, whether or not
			// the error points at a field (codex-review-027 B3).
			want := apperr.InvalidInput
			if slices.Contains(c.args, "Q-099") {
				want = apperr.NotFound
			}
			what := strings.ReplaceAll(strings.Join(run, " "), token, "<token>")
			env := errorV1(t, what, stdout, want)
			if c.pointer == "" {
				continue
			}
			found := false
			for _, d := range env.Error.Details {
				found = found || d.Pointer == c.pointer
			}
			if !found {
				t.Errorf("%v: no detail at %s: %+v", run, c.pointer, env.Error)
			}
		}
	}
	if after := closeState(t, home); strings.Join(after, " ") != strings.Join(before, " ") {
		t.Fatalf("written:\n%v\n%v", before, after)
	}
	if hits := testutil.FindInTree(t, home.Path, token); len(hits) != 0 {
		t.Fatalf("the token reached the data directory: %v", hits)
	}

	if code, stdout, stderr := asHuman(t, dir, "Q-001", "question", "answer", "--file", answerQ1); code != apperr.ExitOK {
		t.Fatalf("answer from a file: exit %d\n%s%s", code, stdout, stderr)
	}
	if code, stdout, stderr := asHuman(t, dir, "Q-002", "question", "dismiss", "--file", dismissQ2); code != apperr.ExitOK {
		t.Fatalf("dismiss from a file: exit %d\n%s%s", code, stdout, stderr)
	}
	l, _ := questionList(t, dir, "--all")
	if got := l.listed(); !strings.Contains(got, "Q-001:answered:REQ-002:true:answer/e-mail only/") ||
		!strings.Contains(got, "Q-002:dismissed:null:false:dismiss/twenty, as now/") {
		t.Fatalf("questions: %s", got)
	}
}
