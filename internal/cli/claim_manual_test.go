package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/claim"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// manualDoc is a manual batch of two claims and unmapped work, with a file
// that exists and a commit prefix that names nothing.
func manualDoc() map[string]any {
	doc := claimDoc(claim.Manual, 1,
		claimOn("AC-001-1", 1, "done", map[string]any{"files": []any{fileRef("docs/client-brief.md", "")}, "commits": []any{"fedcba9"}}),
		claimOn("AC-002-1", 1, "blocked", nil))
	doc["unmapped_work"] = []any{map[string]any{"summary": "canary-unmapped-summary"}}
	return doc
}

// M2-Q6, WP2.4 spec §4 (MVP-CLM-05, E1, E2), a person at a terminal, in
// this process:
//   - declining, a wrong answer and the end of input are exit 3; with
//     --json and stderr not a terminal, or a confirmation that cannot be
//     shown whole, exit 6 with nothing read; none writes anything;
//   - a batch that cannot be written is refused before any prompt;
//   - confirmed with "v<base_version>", it is written with the person as
//     actor on human_cli; with --json the prompt is on stderr and stdout
//     holds the receipt only; the prompt names the kind, the base version,
//     each claim's criterion, revision, assertion and reference results
//     and the unmapped work, and no summary.
//
// The project is outside git (claimsProjectOutsideGit): the commit prefix
// is then unknown_commit too; TestClaimCompareFrom confirms a manual batch
// in a git project.
func TestClaimManualAsAPerson(t *testing.T) {
	home, dir := claimsProjectOutsideGit(t)
	doc := manualDoc()
	file := writeClaims(t, doc)
	before := factState(t, home)
	for _, answer := range []string{"", "v2", "V1", noPrompt} {
		code, stdout, stderr := asHuman(t, dir, answer, "claim", "--file", file)
		if code != apperr.ExitHumanRequired || !strings.Contains(stdout+stderr, "not confirmed") {
			t.Errorf("answer %q: exit %d\n%s%s", answer, code, stdout, stderr)
		}
	}
	t.Chdir(dir)
	var out, errOut bytes.Buffer
	in := &readCounter{r: strings.NewReader("v1\n")}
	if code := runAs(channel.Detection{Channel: channel.HumanCLI}, &out, &errOut, in, "claim", "--file", file, "--json"); code != apperr.ExitEnvironment ||
		in.reads != 0 || !strings.Contains(out.String(), "stderr is not a terminal") {
		t.Errorf("--json, stderr not a terminal: exit %d after %d reads\n%s%s", code, in.reads, out.String(), errOut.String())
	}
	half := &halfWriter{}
	in = &readCounter{r: strings.NewReader("v1\n")}
	if code := runAs(atTerminal, half, &errOut, in, "claim", "--file", file); code != apperr.ExitEnvironment || in.reads != 0 || half.got.Len() == 0 {
		t.Errorf("a prompt cut off: exit %d after %d reads", code, in.reads)
	}
	bad := claimDoc(claim.Manual, 1, claimOn("AC-009-1", 1, "done", nil))
	if code, stdout, stderr := asHuman(t, dir, noPrompt, "claim", "--file", writeClaims(t, bad)); code != apperr.ExitInvalidInput || strings.Contains(stdout+stderr, "to confirm") {
		t.Errorf("an invalid target: exit %d\n%s%s", code, stdout, stderr)
	}
	if after := factState(t, home); after != before {
		t.Fatalf("written before a confirmation:\n%s\n---\n%s", before, after)
	}

	code, stdout, stderr := asHuman(t, dir, "v1", "claim", "--file", file, "--json")
	if code != apperr.ExitOK {
		t.Fatalf("confirmed: exit %d\n%s%s", code, stdout, stderr)
	}
	val, err := schemas()
	if err != nil {
		t.Fatal(err)
	}
	if err := val.ValidateJSON("claim-receipt.v1", []byte(wholeJSON(t, stdout))); err != nil {
		t.Fatalf("stdout is not claim-receipt.v1: %v\n%s", err, stdout)
	}
	var r claimOut
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatal(err)
	}
	claimStored(t, home, doc, r)
	for _, s := range []string{"claim (manual)", "kind manual, base version 1: 2 claims, 1 unmapped work item",
		"claim 0: AC-001-1 revision 1, done; file references 1 exists, 0 missing; commit references 0 resolved, 1 unknown_commit",
		"claim 1: AC-002-1 revision 1, blocked; no references", "unmapped work 0: no references", `Type "v1" to confirm`} {
		if !strings.Contains(stderr, s) {
			t.Errorf("the prompt lacks %q:\n%s", s, stderr)
		}
	}
	noneEchoed(t, "manual", doc, stdout, stderr)
	got := rows(t, home, `SELECT kind, substr(actor, 1, 6), channel FROM claim_batches`)
	if !equalRows(got, [][]string{{"manual", "human:", "human_cli"}}) || r.CompareFrom != nil {
		t.Errorf("batches %v, compare_from %+v", got, r.CompareFrom)
	}

	// A display name with control characters is stored as it is; nothing
	// reaches the terminal unescaped.
	toml := "[user]\ndisplay_name = \"ops\\u001b]0;owned\\u0007\"\n"
	if err := os.WriteFile(home.Join(datadir.ConfigFile), []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = asHuman(t, dir, "v1", "claim", "--file", file)
	if code != apperr.ExitOK || !strings.Contains(stdout, "claim (manual)") || !strings.Contains(stdout, "stored at seq") {
		t.Errorf("confirmed without --json: exit %d\n%s%s", code, stdout, stderr)
	}
	if strings.ContainsAny(stdout+stderr, "\x1b\a") {
		t.Errorf("a control character reached the terminal:\n%q", stdout+stderr)
	}
	if got := rows(t, home, `SELECT actor FROM claim_batches ORDER BY seq DESC LIMIT 1`); !equalRows(got, [][]string{{"human:ops\x1b]0;owned\a"}}) {
		t.Errorf("actor %q", got)
	}
}

// M2-Q6, MVP-CLM-05 in this process: a manual batch outside human_cli is
// exit 3 (human_required) before the configuration is loaded, the
// database opened or git run: with an invalid configuration file it is
// still exit 3 (a sync batch reports the file), the data directory does
// not change, and git is never run. Invalid input is exit 2 first.
func TestClaimManualRefusedOnTheAgentChannel(t *testing.T) {
	home, _, dir := claimsProject(t)
	file := writeClaims(t, manualDoc())
	if err := os.WriteFile(home.Join(datadir.ConfigFile), []byte("not = toml = at all\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, stdout, _ := specmetry(t, dir, "", "claim", "--file", writeClaims(t, claimDoc(claim.Sync, 1)), "--json"); code != apperr.ExitEnvironment ||
		!strings.Contains(stdout, "invalid configuration file") {
		t.Fatalf("a sync batch with the invalid file: exit %d\n%s", code, stdout)
	}
	gitDir, env, calls := testutil.PassthroughGit(t, "")
	useGitFor(t, gitDir, env)
	snapshot := testutil.SettledSnapshot(t, home.Path)
	code, stdout, stderr := specmetry(t, dir, "", "claim", "--file", file, "--json")
	var e apperr.Envelope
	if err := json.Unmarshal([]byte(wholeJSON(t, stdout)), &e); code != apperr.ExitHumanRequired || err != nil || e.Error.Code != apperr.HumanRequired {
		t.Errorf("manual on the Agent channel: exit %d\n%s%s", code, stdout, stderr)
	}
	if code, stdout, _ := specmetry(t, dir, "", "claim", "--file", writeClaims(t, set(t, manualDoc(), "/claims/0/assertion", "unknown")), "--json"); code != apperr.ExitInvalidInput {
		t.Errorf("an invalid manual batch: exit %d\n%s", code, stdout)
	}
	testutil.AssertTreeUnchanged(t, home.Path, snapshot)
	if got := calls(); len(got) != 0 {
		t.Errorf("git ran: %v", got)
	}
}

// Review 067 B2 (WP2.4 spec §5 step 10, §9.3): the receipt is built from
// the rows written and validated against claim-receipt.v1 before COMMIT
// whatever the output; --json only chooses how it is shown.
//   - Sync, audit and manual batches are stored and reported, in text (which
//     names the stored seq) and with --json (the receipt, as stored).
//   - A receipt that does not conform (injected) is exit 1, internal, in
//     both modes: no batch, claim, unmapped work or seq, no success output,
//     the injected value not echoed. For a manual batch, a failing dry run
//     ends before the prompt; a final transaction failing after the
//     confirmation writes nothing either.
func TestClaimReceiptCheckedInEveryMode(t *testing.T) {
	home, dir := claimsProjectOutsideGit(t)
	docs := map[string]map[string]any{
		claim.Sync:   claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", nil)),
		claim.Audit:  claimDoc(claim.Audit, 1, claimOn("AC-001-2", 1, "unknown", nil)),
		claim.Manual: manualDoc(),
	}
	run := func(kind, mode string) (int, string, string) {
		t.Helper()
		args := []string{"claim", "--file", writeClaims(t, docs[kind])}
		if mode == "--json" {
			args = append(args, "--json")
		}
		if kind == claim.Manual {
			return asHuman(t, dir, "v1", args...)
		}
		return specmetry(t, dir, "", args...)
	}
	val, err := schemas()
	if err != nil {
		t.Fatal(err)
	}
	modes := []string{"text", "--json"}
	for _, kind := range []string{claim.Sync, claim.Audit, claim.Manual} {
		for _, mode := range modes {
			code, stdout, stderr := run(kind, mode)
			if code != apperr.ExitOK {
				t.Fatalf("%s, %s: exit %d\n%s%s", kind, mode, code, stdout, stderr)
			}
			last := rows(t, home, `SELECT seq, kind FROM claim_batches ORDER BY seq DESC LIMIT 1`)
			if mode == "text" {
				if !strings.Contains(stdout, "stored at seq "+last[0][0]+" ("+kind+",") || last[0][1] != kind {
					t.Errorf("%s, text: batch %v\n%s", kind, last, stdout)
				}
				continue
			}
			if err := val.ValidateJSON("claim-receipt.v1", []byte(wholeJSON(t, stdout))); err != nil {
				t.Fatalf("%s: stdout is not claim-receipt.v1: %v\n%s", kind, err, stdout)
			}
			var r claimOut
			if err := json.Unmarshal([]byte(stdout), &r); err != nil {
				t.Fatal(err)
			}
			claimStored(t, home, docs[kind], r)
		}
	}

	canary := testutil.NewCanary(t)
	calls, failOn := 0, 0
	testHookClaimReceipt = func(r *claim.Receipt) {
		calls++
		if calls == failOn {
			r.BatchID = canary
		}
	}
	t.Cleanup(func() { testHookClaimReceipt = nil })
	refused := func(name, kind, mode string, fail int, prompted bool) {
		t.Helper()
		calls, failOn = 0, fail
		before := factState(t, home)
		code, stdout, stderr := run(kind, mode)
		out := stdout + stderr
		if code != apperr.ExitInternal || strings.Contains(out, "stored at seq") || strings.Contains(out, canary) || strings.Contains(out, "to confirm") != prompted {
			t.Errorf("%s, %s: exit %d\n%s%s", name, mode, code, stdout, stderr)
		}
		if mode == "--json" {
			var env apperr.Envelope
			if err := json.Unmarshal([]byte(wholeJSON(t, stdout)), &env); err != nil || env.Error.Code != apperr.Internal || !strings.Contains(env.Error.Message, "claim-receipt.v1") {
				t.Errorf("%s, --json: %v\n%s", name, err, stdout)
			}
		} else if !strings.Contains(stderr, "claim-receipt.v1") {
			t.Errorf("%s, text: stderr %q", name, stderr)
		}
		if calls != fail {
			t.Errorf("%s, %s: the receipt was built %d times, want %d", name, mode, calls, fail)
		}
		if after := factState(t, home); after != before {
			t.Errorf("%s, %s wrote:\n%s\n---\n%s", name, mode, before, after)
		}
	}
	for _, mode := range modes {
		refused("sync", claim.Sync, mode, 1, false)
		refused("audit", claim.Audit, mode, 1, false)
		refused("manual, the dry run", claim.Manual, mode, 1, false)
		refused("manual, the final transaction", claim.Manual, mode, 2, true)
	}
}

func set(t *testing.T, doc map[string]any, pointer string, v any) map[string]any {
	t.Helper()
	return patched(t, doc, pointer, v).(map[string]any)
}

// beforeRead runs fn once, at the first read, before reading from r: a
// person's answer that arrives after something else happened.
type beforeRead struct {
	fn   func()
	r    io.Reader
	done bool
}

func (b *beforeRead) Read(p []byte) (int, error) {
	if !b.done {
		b.done = true
		b.fn()
	}
	return b.r.Read(p)
}

// Review 065 B1 and review 066 (WP2.4 spec §5 step 10, §15): the version
// precondition comes first. A claim read at version 1 waits, before its
// write lock, while another connection lawfully retires what it names and
// commits version 2 (the S1 snapshot): the claim is exit 5 with the
// current version, not exit 2, and the database is S1 exactly: nothing of
// the claim (no batch, claim, unmapped work or first worktree
// registration, no seq) and all of the other connection's facts, none
// undone. Three cases: a retired target, a retired requirement of an audit
// scope (D2), from a worktree not registered yet; and a manual batch whose
// baseline changes while the person is asked.
func TestClaimVersionFirstWhenTheBaselineChanges(t *testing.T) {
	home, f, dir := claimsProject(t)
	version := 1
	retireMeanwhile := func(item map[string]any) func() {
		return func() {
			testHookBeforeClaimWrite = nil
			acceptAll(t, dir, proposeOK(t, dir, small(version, item)), version, nil)
			version++
		}
	}
	stale := func(name string, env apperr.Envelope, s1 string) {
		t.Helper()
		if env.Error.Code != apperr.StalePrecondition || toFloat(env.Error.Current["base_version"]) != float64(version) {
			t.Errorf("%s: %+v; want stale at version %d", name, env.Error, version)
		}
		if got := factState(t, home); got != s1 {
			t.Errorf("%s: the database is not S1:\n%s\n---\n%s", name, s1, got)
		}
	}
	run := func(name, at string, doc map[string]any, meanwhile func()) {
		t.Helper()
		var s1 string
		testHookBeforeClaimWrite = func() { meanwhile(); s1 = factState(t, home) }
		t.Cleanup(func() { testHookBeforeClaimWrite = nil })
		code, stdout, stderr := specmetry(t, at, "claim-receipt.v1", "claim", "--file", writeClaims(t, doc), "--json")
		var env apperr.Envelope
		if err := json.Unmarshal([]byte(wholeJSON(t, stdout)), &env); code != apperr.ExitStalePrecondition || err != nil || s1 == "" {
			t.Fatalf("%s: exit %d\n%s%s", name, code, stdout, stderr)
		}
		stale(name, env, s1)
	}
	run("a retired target", dir, claimDoc(claim.Sync, 1, claimOn("AC-001-2", 1, "done", nil)), retireMeanwhile(retire("criterion", "AC-001-2", 1)))
	wt := f.Repo.AddWorktree("feature-b1")
	scope := claimDoc(claim.Audit, 2, claimOn("AC-002-1", 1, "done", nil))
	scope["audit_scope"] = map[string]any{"requirements": []any{"REQ-001", "REQ-002"}}
	run("a retired requirement of the scope, from a new worktree", wt, scope, retireMeanwhile(retire("requirement", "REQ-002", 1)))
	if got := rows(t, home, `SELECT count(*) FROM worktrees WHERE path = ?`, wt); !equalRows(got, [][]string{{"0"}}) {
		t.Errorf("the new worktree was registered: %v", got)
	}

	// A manual batch: the person answers after the baseline changed.
	acceptAll(t, dir, proposeOK(t, dir, small(version, addAC("c4", "REQ-001", "an admin can unlock"))), version, nil)
	version++
	doc := claimDoc(claim.Manual, version, claimOn("AC-001-3", 1, "done", nil))
	var s1 string
	t.Chdir(dir)
	var out, errOut bytes.Buffer
	in := &beforeRead{fn: func() { retireMeanwhile(retire("criterion", "AC-001-3", 1))(); s1 = factState(t, home) }, r: strings.NewReader("v" + strconv.Itoa(version) + "\n")}
	code := runAs(atTerminal, &out, &errOut, in, "claim", "--file", writeClaims(t, doc), "--json")
	var env apperr.Envelope
	if err := json.Unmarshal([]byte(wholeJSON(t, out.String())), &env); code != apperr.ExitStalePrecondition || err != nil {
		t.Fatalf("manual: exit %d\n%s%s", code, out.String(), errOut.String())
	}
	stale("manual, changed while asked", env, s1)
}

// WP2.4 spec §15: claims submitted at once by separate processes are each
// written whole or not at all, every batch with its own seq, the seqs of
// those written following one another.
//   - Outside git no claim reads git, and every one is written.
//   - In a git project each reads its worktree state through the data
//     directory's private namespace, whose lease is not shared: one that
//     finds it busy is exit 6 with nothing written and nothing retried
//     (WP2.3 spec §14.3, scheme B: "竞争环境错误，不自动重试"), and at
//     least one is written.
func TestClaimConcurrently(t *testing.T) {
	t.Run("outside git", func(t *testing.T) {
		noAgentVariables(t)
		home := testutil.TempHome(t)
		root, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		mustInit(t, root, "--name", "plain")
		acceptAll(t, root, proposeOK(t, root, small(0, addReq("r1", "sign up", nil), addAC("c1", "r1", "a new account can sign in"),
			addAC("c2", "r1", "a locked account cannot"))), 0, nil)
		if written := claimsAtOnce(t, home, root); written != 4 {
			t.Errorf("%d of 4 written", written)
		}
	})
	t.Run("git", func(t *testing.T) {
		home, _, dir := claimsProject(t)
		if written := claimsAtOnce(t, home, dir); written == 0 {
			t.Error("none written")
		}
	})
}

// claimsAtOnce runs four claim processes at once in dir and checks them:
// each is written whole with a seq of its own or is a busy private
// namespace (exit 6) that wrote nothing; the written seqs follow one
// another. It returns how many were written.
func claimsAtOnce(t *testing.T, home datadir.Dir, dir string) int {
	t.Helper()
	before, err := strconv.Atoi(seq(t, home))
	if err != nil {
		t.Fatal(err)
	}
	const n = 4
	cmds := make([]*exec.Cmd, n)
	outs, stderrs := make([]*bytes.Buffer, n), make([]*bytes.Buffer, n)
	for i := range n {
		doc := claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", nil), claimOn("AC-001-2", 1, "partial", nil))
		doc["summary"] = "concurrent " + strconv.Itoa(i)
		raw, _ := json.Marshal([]string{"claim", "--file", writeClaims(t, doc), "--json"})
		cmds[i] = testutil.HelperCommand(t, "cli")
		cmds[i].Env = helperEnv(t, "cli", argsJSONEnv+"="+string(raw), dirEnv+"="+dir)
		outs[i] = &bytes.Buffer{}
		cmds[i].Stdout = outs[i]
		stderrs[i] = &bytes.Buffer{}
		cmds[i].Stderr = stderrs[i]
	}
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Go(func() { errs[i] = cmds[i].Run() })
	}
	wg.Wait()
	var seqs []int
	for i := range n {
		if strings.Contains(stderrs[i].String(), "exit=6") && strings.Contains(outs[i].String(), "cannot acquire the private namespace lease") {
			continue
		}
		var r claimOut
		if err := json.Unmarshal(outs[i].Bytes(), &r); errs[i] != nil || err != nil {
			t.Fatalf("claim %d: %v %v\n%s%s", i, errs[i], err, outs[i].String(), stderrs[i].String())
		}
		seqs = append(seqs, int(r.Seq))
	}
	sort.Ints(seqs)
	for i, s := range seqs {
		if s != before+1+i {
			t.Fatalf("seqs %v after %d", seqs, before)
		}
	}
	if counter := seq(t, home); counter != strconv.Itoa(before+len(seqs)) {
		t.Errorf("seq counter %s after %d and %d batches", counter, before, len(seqs))
	}
	got := rows(t, home, `SELECT b.seq, count(c.id), min(c.seq) = b.seq AND max(c.seq) = b.seq FROM claim_batches b JOIN claims c ON c.batch_id = b.id GROUP BY b.id ORDER BY b.seq`)
	if len(got) != len(seqs) || count(t, home, `SELECT count(*) FROM claim_batches`) != len(seqs) {
		t.Fatalf("batches %v for %d written", got, len(seqs))
	}
	for _, row := range got {
		if row[1] != "2" || row[2] != "1" {
			t.Errorf("batch %v", row)
		}
	}
	return len(seqs)
}

// errWriter fails every write.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("the reader went away") }

// N2 (WP2.4 spec §9.3): after COMMIT only the output is left to write. A
// write that fails or comes up short is exit 6 with the batch stored: no
// second document follows what was written of the receipt (with --json,
// never an error.v1 after it), and stderr says that the batch may be
// stored and should not be submitted again before reading the status. In
// text alike. (Outside git: the output does not depend on it.)
func TestClaimOutputFailsAfterTheCommit(t *testing.T) {
	home, dir := claimsProjectOutsideGit(t)
	file := writeClaims(t, claimDoc(claim.Sync, 1, claimOn("AC-001-1", 1, "done", nil)))
	agent := channel.Detection{Channel: channel.AgentCLI, Reasons: []string{"not a terminal"}}
	t.Chdir(dir)
	stored := 0
	for _, c := range []struct {
		name string
		half bool
		json bool
	}{{"a short write, --json", true, true}, {"a write error, --json", false, true}, {"a short write, text", true, false}, {"a write error, text", false, false}} {
		var w io.Writer = errWriter{}
		half := &halfWriter{}
		if c.half {
			w = half
		}
		args := []string{"claim", "--file", file}
		if c.json {
			args = append(args, "--json")
		}
		var errOut bytes.Buffer
		code := runAs(agent, w, &errOut, strings.NewReader(""), args...)
		stored++
		if code != apperr.ExitEnvironment {
			t.Errorf("%s: exit %d\n%s", c.name, code, errOut.String())
		}
		if n := count(t, home, `SELECT count(*) FROM claim_batches`); n != stored {
			t.Errorf("%s: %d batches, want %d: the batch is stored", c.name, n, stored)
		}
		if msg := errOut.String(); !strings.Contains(msg, "was stored") || !strings.Contains(msg, "do not submit it again") {
			t.Errorf("%s: stderr says\n%s", c.name, msg)
		}
		if c.half {
			got := half.got.String()
			if got == "" || strings.Contains(got, "specmetry.error/v1") || strings.Contains(got, "error:") || (c.json && json.Valid([]byte(got))) {
				t.Errorf("%s: stdout holds\n%s", c.name, got)
			}
			if c.json && !strings.HasPrefix(got, "{") {
				t.Errorf("%s: stdout is not the start of the receipt:\n%s", c.name, got)
			}
		}
	}
}
