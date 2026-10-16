package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// esc, rlo (a bidirectional override) and csi1 (the C1 CSI) are made at run
// time, and vis writes an escape the way termtext shows it, so that no
// such character, nor an escape a tool could turn into one, is in the
// sources.
var (
	rlo  = string(rune(0x202e))
	csi1 = string(rune(0x9b))
)

func vis(code string) string { return `\` + code }

// MVP-PRV-* display safety (implementation-plan M6; codex-review-027 B1 for
// the other commands): every text a command shows a person, from a
// proposal, a source path, the configuration or an argument, reaches the
// terminal with its control characters shown as escapes: proposal list and
// show, baseline show and history, the review and propose --apply
// confirmations and results, source add, list, ack and remove, init, and
// error text. Line breaks stay (CRLF as LF), continuation lines of a
// multi-line text are indented under their field, and Chinese, Japanese
// and emoji are shown as they are. --json and the database keep every
// value as it was; a cancelled confirmation writes nothing; confirmations
// still take the text they ask for. All of it is read from memory, never
// from a real terminal.
func TestTextOutputIsShownSafely(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	// The display name, for every human command.
	toml := "[user]\ndisplay_name = \"ops" + vis("u001b") + "]0;owned" + vis("u0007") + vis("u202e") + "\"\n"
	if err := os.WriteFile(home.Join(datadir.ConfigFile), []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}
	name := "ops\x1b]0;owned\a" + rlo
	shownName := "human:ops" + vis("x1b") + "]0;owned" + vis("a") + vis("u202e")
	// A source whose path holds a bidi override, Chinese, Japanese and emoji.
	path := "docs/需求-" + rlo + "-ブリーフ🚀.md"
	shownPath := "docs/需求-" + vis("u202e") + "-ブリーフ🚀.md"
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(path)), []byte("导出的 CSV 包含订单编号。\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, stderr := specmetry(t, dir, "", "source", "add", path)
	if code != apperr.ExitOK {
		t.Fatalf("source add: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "source add", out+stderr, shownPath)
	code, out, stderr = specmetry(t, dir, "", "source", "list")
	if code != apperr.ExitOK {
		t.Fatalf("source list: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "source list", out+stderr, shownPath)

	summary := "sum\x1b[2J\x1b[H fake\rover\a\b\x7f" + csi1 + "[31m" + rlo + " 中文 日本語 🚀\r\nsecond line"
	shownSummary := "sum" + vis("x1b") + "[2J" + vis("x1b") + "[H fake" + vis("r") + "over" + vis("a") + vis("b") + vis("x7f") +
		vis("u009b") + "[31m" + vis("u202e") + " 中文 日本語 🚀"
	statement := "statement \x1b[1A one\nline two\rx"
	goal := addGoal("g", "goal \x1b]0;title\a")
	req := addReq("r", "req "+rlo+"title", "g")
	req["sources"] = []any{map[string]any{"source": path, "excerpt": "导出的 CSV 包含订单编号。"}}
	req["code_refs"] = []any{map[string]any{"path": "src/\x1bx.go"}}
	doc := small(0, goal, req, addAC("c", "r", statement))
	doc["summary"] = summary
	r := proposeOK(t, dir, doc)

	code, out, stderr = specmetry(t, dir, "", "proposal", "list")
	if code != apperr.ExitOK {
		t.Fatalf("proposal list: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "proposal list", out+stderr, shownSummary, "\n    second line")
	code, out, stderr = specmetry(t, dir, "", "proposal", "show", r.ProposalID)
	if code != apperr.ExitOK {
		t.Fatalf("proposal show: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "proposal show", out+stderr, shownSummary, "\n    second line", shownPath, "src/"+vis("x1b")+"x.go", vis("u202e")+"title")

	// The review: its confirmation, cancelled (nothing written), then
	// confirmed with the proposal's ID.
	review := writeReview(t, reviewDoc(r.ProposalID, 0, decide(itemIDs(r)["g"], "accepted"), decide(itemIDs(r)["r"], "accepted"),
		decide(itemIDs(r)["c"], "accepted")))
	before := strings.Join(closeState(t, home), " ")
	code, out, stderr = asHuman(t, dir, "", "review", "--file", review)
	if code != apperr.ExitHumanRequired || !strings.Contains(out, "to confirm") {
		t.Fatalf("cancelled review: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "the review's confirmation", out+stderr, shownSummary, "\n    second line")
	if after := strings.Join(closeState(t, home), " "); after != before {
		t.Fatalf("a cancelled review wrote:\n%s\n%s", before, after)
	}
	code, out, stderr = asHuman(t, dir, r.ProposalID, "review", "--file", review)
	if code != apperr.ExitOK {
		t.Fatalf("review: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "review", out+stderr)

	code, out, stderr = specmetry(t, dir, "", "baseline", "show")
	if code != apperr.ExitOK {
		t.Fatalf("baseline show: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "baseline show", out+stderr, "goal "+vis("x1b")+"]0;title"+vis("a"), "req "+vis("u202e")+"title",
		"statement "+vis("x1b")+"[1A one", "\n    line two"+vis("r")+"x")

	// A person's edit: its confirmation, cancelled, then confirmed.
	edit := small(1, update("requirement", "REQ-001", 1, map[string]any{"title": "req, edited " + rlo}))
	edit["origin"] = "human_edit"
	edit["summary"] = "edit\x1b[2K" + rlo + "\r\nby hand"
	edit["items"].([]any)[0].(map[string]any)["change_class"] = "editorial"
	file := writeReview(t, edit)
	before = strings.Join(closeState(t, home), " ")
	code, out, stderr = asHuman(t, dir, "", "propose", "--apply", "--file", file)
	if code != apperr.ExitHumanRequired || !strings.Contains(out, "to confirm") {
		t.Fatalf("cancelled edit: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "the edit's confirmation", out+stderr, "edit"+vis("x1b")+"[2K"+vis("u202e"), "\n    by hand")
	if after := strings.Join(closeState(t, home), " "); after != before {
		t.Fatalf("a cancelled edit wrote:\n%s\n%s", before, after)
	}
	if code, out, stderr = asHuman(t, dir, "v2", "propose", "--apply", "--file", file); code != apperr.ExitOK {
		t.Fatalf("propose --apply: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "propose --apply", out+stderr)
	code, out, stderr = specmetry(t, dir, "", "baseline", "history")
	if code != apperr.ExitOK {
		t.Fatalf("baseline history: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "baseline history", out+stderr, shownName, shownSummary, "edit"+vis("x1b")+"[2K"+vis("u202e"))

	// The source changes, and is acknowledged and removed: the prompts
	// show the path and the reason escaped, and ask for the source's ID,
	// which can be typed, not for a path that cannot.
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(path)), []byte("导出的 CSV 包含订单编号和金额。\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	src := sourceList(t, dir).Sources
	var id, current string
	for _, s := range src {
		if s.Path == path {
			id, current = s.ID, *s.CurrentSHA256
		}
	}
	if id == "" {
		t.Fatalf("no source %q in %+v", path, src)
	}
	before = strings.Join(closeState(t, home), " ")
	ack := []string{"source", "ack", path, "--observed", current, "-m", "typo\x1b[1A fixed" + rlo}
	code, out, stderr = asHuman(t, dir, "", ack...)
	if code != apperr.ExitHumanRequired || !strings.Contains(out, `Type "`+id+`"`) {
		t.Fatalf("cancelled ack: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "the ack's confirmation", out+stderr, shownPath, "typo"+vis("x1b")+"[1A fixed"+vis("u202e"))
	code, out, stderr = asHuman(t, dir, "", "source", "remove", path)
	if code != apperr.ExitHumanRequired || !strings.Contains(out, `Type "`+id+`"`) {
		t.Fatalf("cancelled remove: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "the removal's confirmation", out+stderr, shownPath)
	if after := strings.Join(closeState(t, home), " "); after != before {
		t.Fatalf("a cancelled ack or removal wrote:\n%s\n%s", before, after)
	}
	if code, out, stderr = asHuman(t, dir, id, ack...); code != apperr.ExitOK {
		t.Fatalf("source ack: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "source ack", out+stderr, shownPath)

	// Error text quoting an argument (already scanned for secrets).
	code, out, stderr = specmetry(t, dir, "", "source", "add", "docs/"+rlo+"missing\x1b.md")
	if code != apperr.ExitInvalidInput {
		t.Fatalf("source add of a missing file: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "an error", out+stderr, "docs/"+vis("u202e")+"missing"+vis("x1b")+".md")

	// The data keep every value as it is.
	var show struct {
		Proposal struct {
			Summary string `json:"summary"`
		} `json:"proposal"`
		Items []struct {
			Proposed map[string]any `json:"proposed"`
			Sources  []struct {
				Path string `json:"path"`
			} `json:"sources"`
		} `json:"items"`
	}
	code, out, _ = specmetry(t, dir, "proposal-show.v1", "proposal", "show", r.ProposalID, "--json")
	if err := json.Unmarshal([]byte(out), &show); code != apperr.ExitOK || err != nil {
		t.Fatalf("proposal show --json: exit %d, %v", code, err)
	}
	if show.Proposal.Summary != summary || show.Items[1].Proposed["title"] != "req "+rlo+"title" || show.Items[2].Proposed["statement"] != statement ||
		show.Items[1].Sources[0].Path != path {
		t.Fatalf("proposal show --json changed the values: %+v", show)
	}
	got := rows(t, home, `SELECT (SELECT summary FROM proposals WHERE id = ?), (SELECT statement FROM acceptance_criteria),
    (SELECT actor FROM baseline_versions WHERE number = 2), (SELECT path FROM sources WHERE id = ?)`, r.ProposalID, id)
	if want := [][]string{{summary, statement, "human:" + name, path}}; !equalRows(got, want) {
		t.Fatalf("stored %q, want %q", got, want)
	}
}

// init's text output shows a project name with a bidi override escaped,
// and Chinese and emoji as they are; init.v1 keeps the name.
func TestInitTextIsShownSafely(t *testing.T) {
	testutil.TempHome(t)
	f := testutil.BuildFixture(t, "fx-docs-only")
	name := "项目 " + rlo + "evil 🚀"
	code, out, stderr := specmetry(t, f.Repo.Dir, "", "init", "--name", name)
	if code != apperr.ExitOK {
		t.Fatalf("init: exit %d\n%s%s", code, out, stderr)
	}
	shownSafely(t, "init", out+stderr, "项目 "+vis("u202e")+"evil 🚀")
	code, out, _ = specmetry(t, f.Repo.Dir, "init.v1", "init", "--json")
	var doc struct {
		Project struct {
			Name string `json:"name"`
		} `json:"project"`
	}
	if err := json.Unmarshal([]byte(out), &doc); code != apperr.ExitOK || err != nil || doc.Project.Name != name {
		t.Fatalf("init --json: exit %d, %v, %+v", code, err, doc)
	}
}
