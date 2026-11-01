package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
)

// MVP-EV-21, CX-29 (WP2.5 D3, D4): a check whose command a cmd check would
// cut at its first line (CR or LF), or with a CR in an sh check, is
// refused with exit 2 and nothing written on every path that writes a
// check: propose (add and update), review accepted_with_edits, review
// accepted of an item stored before the rule, and a person's edit. A saved
// check that breaks the rule stays readable and unchanged, an update
// without a check does not check it again, and a material edit replaces
// it. The refusal never quotes the command.
func TestCheckLineBreaksOnEveryWritePath(t *testing.T) {
	home, f, _ := proposalsProject(t)
	dir := f.Repo.Dir
	const canary = "CANARY-91d4"
	check := func(command, shell string) map[string]any {
		c := map[string]any{"command": command, "rationale": "it runs the tests"}
		if shell != "" {
			c["shell"] = shell
		}
		return c
	}
	automated := func(ref string, c map[string]any) map[string]any {
		return map[string]any{"op": "add", "type": "criterion", "ref": ref, "origin": "manual",
			"data": map[string]any{"requirement": "REQ-001", "statement": "orders are isolated", "verification": "automated", "check": c}}
	}
	material := func(item map[string]any) map[string]any {
		item["change_class"] = "material"
		return item
	}
	noQuote := func(env apperr.Envelope) {
		t.Helper()
		if b, _ := json.Marshal(env); strings.Contains(string(b), canary) {
			t.Errorf("the refusal quotes the command: %s", b)
		}
	}
	acceptAll(t, dir, proposeOK(t, dir, small(0, addReq("r", "orders", nil))), 0, nil)
	acceptAll(t, dir, proposeOK(t, dir, small(1, automated("c", check("echo one & echo two", "cmd")))), 1, nil)

	// propose: an add and an update.
	noQuote(proposeRefused(t, home, dir, small(2, automated("d", check("echo one\necho "+canary, "cmd"))), apperr.ExitInvalidInput,
		"/items/0/data/check/command"))
	noQuote(proposeRefused(t, home, dir, small(2, update("criterion", "AC-001-1", 1, map[string]any{"check": check("echo "+canary+"\r", "")})),
		apperr.ExitInvalidInput, "/items/0/data/check/command"))

	// review accepted_with_edits.
	r := proposeOK(t, dir, small(2, update("criterion", "AC-001-1", 1, map[string]any{"statement": "orders stay isolated"})))
	noQuote(reviewRefused(t, home, dir, reviewDoc(r.ProposalID, 2, decide(r.Items[0].ItemID, "accepted_with_edits", "change_class", "material",
		"final_data", map[string]any{"check": check("echo one\r\necho "+canary, "sh")})), apperr.ExitInvalidInput, "/items/0/final_data/check/command"))
	reviewOK(t, dir, reviewDoc(r.ProposalID, 2, decide(r.Items[0].ItemID, "rejected", "note", "not now")))

	// review accepted: an item stored before the rule (as an older binary
	// stored it), applied as proposed.
	r = proposeOK(t, dir, small(2, update("criterion", "AC-001-1", 1, map[string]any{"check": check("echo three", "cmd")})))
	maintain(t, home, `UPDATE proposal_items SET payload_json = json_set(payload_json, '$.check.command', ?) WHERE id = ?`,
		"echo one\necho "+canary, r.Items[0].ItemID)
	env := reviewRefused(t, home, dir, reviewDoc(r.ProposalID, 2, decide(r.Items[0].ItemID, "accepted", "change_class", "material")),
		apperr.ExitInvalidInput, "/items/0")
	noQuote(env)
	if len(env.Error.Details) != 1 || !strings.Contains(env.Error.Details[0].Problem, "cmd") {
		t.Errorf("review accepted: details %+v", env.Error.Details)
	}
	reviewOK(t, dir, reviewDoc(r.ProposalID, 2, decide(r.Items[0].ItemID, "rejected", "note", "not now")))

	// A person's edit.
	before := written(t, home) + " " + state(t, home)
	code, stdout, stderr := asHuman(t, dir, "v3", "propose", "--apply", "--json", "--file",
		writeProposal(t, editDoc(2, material(update("criterion", "AC-001-1", 1, map[string]any{"check": check(canary+"\r", "bash")})))))
	if code != apperr.ExitInvalidInput || !strings.Contains(stdout, "/items/0/data/check/command") || strings.Contains(stdout+stderr, canary) {
		t.Errorf("a person's edit: exit %d\n%s%s", code, stdout, stderr)
	}
	if after := written(t, home) + " " + state(t, home); after != before {
		t.Errorf("a person's edit wrote %s -> %s", before, after)
	}

	// A saved check that breaks the rule (written before it) stays as it
	// is: shown, not rewritten, and not checked by an update without a
	// check.
	cmd := "cmd"
	saved := baseline.Check{Command: "echo one\necho two", Workdir: ".", Shell: &cmd}
	maintain(t, home, `UPDATE acceptance_criteria SET check_command = ?, check_config_hash = ? WHERE key = 'AC-001-1'`,
		saved.Command, baseline.CheckConfigHash(saved))
	shown := func() (string, int) {
		t.Helper()
		_, stdout, _ := specmetry(t, dir, "baseline-show.v1", "baseline", "show", "--json")
		var doc struct {
			Requirements []struct {
				Criteria []struct {
					Revision int `json:"revision"`
					Check    struct {
						Command string `json:"command"`
					} `json:"check"`
				} `json:"criteria"`
			} `json:"requirements"`
		}
		if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
			t.Fatalf("baseline show: %v\n%s", err, stdout)
		}
		c := doc.Requirements[0].Criteria[0]
		return c.Check.Command, c.Revision
	}
	if command, rev := shown(); command != saved.Command || rev != 1 {
		t.Fatalf("the saved check shows as %q, revision %d", command, rev)
	}
	acceptAll(t, dir, proposeOK(t, dir, small(2, update("criterion", "AC-001-1", 1, map[string]any{"statement": "orders stay isolated"}))), 2, nil)
	if command, rev := shown(); command != saved.Command || rev != 1 {
		t.Errorf("after an update without a check: %q, revision %d", command, rev)
	}
	if got := rows(t, home, `SELECT check_config_hash FROM acceptance_criteria WHERE key = 'AC-001-1'`); got[0][0] != baseline.CheckConfigHash(saved) {
		t.Errorf("the saved hash changed: %v", got)
	}
	// A material edit replaces it.
	applyEdit(t, dir, editDoc(3, material(update("criterion", "AC-001-1", 1, map[string]any{"check": check("echo one & echo two", "cmd")}))))
	if command, rev := shown(); command != "echo one & echo two" || rev != 2 {
		t.Errorf("after the material edit: %q, revision %d", command, rev)
	}
}

// maintain runs query with append-only enforcement suspended, as a purge
// would: the test's stand-in for rows an older binary wrote.
func maintain(t *testing.T, home datadir.Dir, query string, args ...any) {
	t.Helper()
	ctx := context.Background()
	db, _, err := sqlite.Open(ctx, home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		return sqlite.Maintenance(ctx, tx, func() error {
			_, err := tx.ExecContext(ctx, query, args...)
			return err
		})
	}); err != nil {
		t.Fatal(err)
	}
}
