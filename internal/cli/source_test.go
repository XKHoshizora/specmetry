package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

type sourceOut struct {
	ID            string             `json:"id"`
	Path          string             `json:"path"`
	Location      string             `json:"location"`
	Kind          string             `json:"kind"`
	AckedSHA256   string             `json:"acked_sha256"`
	CurrentSHA256 *string            `json:"current_sha256"`
	State         *string            `json:"state"`
	SizeBytes     int64              `json:"size_bytes"`
	RegisteredSeq int64              `json:"registered_seq"`
	RemovedSeq    *int64             `json:"removed_seq"`
	Refs          *[]json.RawMessage `json:"refs"`
}

type sourceAddOut struct {
	Created       bool      `json:"created"`
	Source        sourceOut `json:"source"`
	SourceSetHash string    `json:"source_set_hash"`
}

type sourceListOut struct {
	ExcerptStorage string      `json:"excerpt_storage"`
	SourceSetHash  string      `json:"source_set_hash"`
	Sources        []sourceOut `json:"sources"`
}

func fileSHA(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// setHash recomputes data-model §4.3 independently of the code under test.
func setHash(sources []sourceOut) string {
	var lines []string
	for _, s := range sources {
		if s.RemovedSeq == nil {
			lines = append(lines, s.ID+":"+s.AckedSHA256+"\n")
		}
	}
	sort.Strings(lines)
	h := sha256.Sum256([]byte(strings.Join(lines, "")))
	return hex.EncodeToString(h[:])
}

// specmetry runs the CLI in dir in this process, where stdin and stdout
// are not terminals (agent_cli), and checks --json output against schema
// (error.v1 on failure).
func specmetry(t *testing.T, dir, schema string, args ...string) (int, string, string) {
	t.Helper()
	t.Chdir(dir)
	code, stdout, stderr := execute(t, args...)
	if schema != "" && (code == apperr.ExitOK || strings.Contains(strings.Join(args, " "), "--json")) {
		validator, err := schemas()
		if err != nil {
			t.Fatal(err)
		}
		name := schema
		if code != apperr.ExitOK {
			name = "error.v1"
		}
		if err := validator.ValidateJSON(name, []byte(stdout)); err != nil {
			t.Fatalf("%v: stdout does not conform to %s: %v %+v\n%s", args, name, err, apperr.From(err).Details, stdout)
		}
	}
	return code, stdout, stderr
}

func sourceAdd(t *testing.T, dir string, args ...string) sourceAddOut {
	t.Helper()
	code, stdout, stderr := specmetry(t, dir, "source-add.v1", append([]string{"source", "add", "--json"}, args...)...)
	if code != apperr.ExitOK {
		t.Fatalf("source add %v: exit %d\n%s%s", args, code, stdout, stderr)
	}
	var out sourceAddOut
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func sourceList(t *testing.T, dir string, args ...string) sourceListOut {
	t.Helper()
	code, stdout, stderr := specmetry(t, dir, "source-list.v1", append([]string{"source", "list", "--json"}, args...)...)
	if code != apperr.ExitOK {
		t.Fatalf("source list %v: exit %d\n%s%s", args, code, stdout, stderr)
	}
	var out sourceListOut
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatal(err)
	}
	// Output consistency the schema cannot express (implementation-plan M1).
	if out.SourceSetHash != setHash(out.Sources) {
		t.Fatalf("source_set_hash %s does not match the listed active sources (%s)", out.SourceSetHash, setHash(out.Sources))
	}
	for _, s := range out.Sources {
		if s.RemovedSeq != nil {
			continue
		}
		unchanged := s.CurrentSHA256 != nil && *s.CurrentSHA256 == s.AckedSHA256
		if (*s.State == "unchanged") != unchanged {
			t.Fatalf("%s: state %s with acked %s and current %v", s.Path, *s.State, s.AckedSHA256, s.CurrentSHA256)
		}
	}
	return out
}

// noPrompt is the answer for a run that must end before any confirmation
// prompt; "" answers a prompt with Enter alone, which declines.
const noPrompt = "\x00"

// inTerminal runs the CLI in a pseudo-terminal in dir, answering the
// confirmation with answer (a person typing), and returns the output,
// which ends with "exit=<code>".
func inTerminal(t *testing.T, dir string, args []string, answer string, extra ...string) string {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	env := helperEnv(t, "cli", append(extra, argsJSONEnv+"="+string(raw), dirEnv+"="+dir)...)
	cmd := testutil.HelperCommand(t, "cli")
	var steps []testutil.PTYStep
	if answer != noPrompt {
		steps = []testutil.PTYStep{{Expect: "press Enter to cancel:", Send: answer + "\r"}}
	}
	return testutil.RunInPTYSession(t, 60*time.Second, env, steps, cmd.Path, cmd.Args[1:]...).Output
}

// inTerminalChanging is inTerminal that runs change while the confirmation
// prompt is shown, before the answer is typed.
func inTerminalChanging(t *testing.T, dir string, args []string, answer string, change func()) string {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	env := helperEnv(t, "cli", argsJSONEnv+"="+string(raw), dirEnv+"="+dir)
	cmd := testutil.HelperCommand(t, "cli")
	steps := []testutil.PTYStep{{Expect: "press Enter to cancel:", Before: change, Send: answer + "\r"}}
	return testutil.RunInPTYSession(t, 60*time.Second, env, steps, cmd.Path, cmd.Args[1:]...).Output
}

func seq(t *testing.T, home datadir.Dir) string {
	t.Helper()
	return rows(t, home, `SELECT seq_counter FROM meta`)[0][0]
}

// sourcesProject is fx-docs-only, initialized, with an external document
// whose path has spaces and Chinese characters.
func sourcesProject(t *testing.T) (home datadir.Dir, repo string, external string) {
	t.Helper()
	home = testutil.TempHome(t)
	f := testutil.BuildFixture(t, "fx-docs-only")
	mustInit(t, f.Repo.Dir)
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	external = filepath.Join(dir, "客户 资料", "补充 需求.md")
	if err := os.MkdirAll(filepath.Dir(external), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(external, []byte("# 补充需求\n\n导出订单。\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, f.Repo.Dir, external
}

// MVP-SRC-01: an in-repo and an external source (spaces, Chinese) are
// registered with path, kind, location, acknowledged hash and size.
func TestSourceAddRegisters(t *testing.T) {
	home, repo, external := sourcesProject(t)
	brief := filepath.Join(repo, "docs", "client-brief.md")

	a := sourceAdd(t, repo, "docs/client-brief.md", "--kind", "client_requirement")
	if !a.Created || a.Source.Location != "in_repo" || a.Source.Path != "docs/client-brief.md" || a.Source.Kind != "client_requirement" {
		t.Fatalf("in-repo source = %+v", a)
	}
	if a.Source.AckedSHA256 != fileSHA(t, brief) || a.Source.SizeBytes != 2255 || *a.Source.State != "unchanged" {
		t.Fatalf("in-repo source = %+v", a.Source)
	}
	b := sourceAdd(t, filepath.Join(repo, "docs"), external)
	if !b.Created || b.Source.Location != "external" || b.Source.Path != external || b.Source.Kind != "other" {
		t.Fatalf("external source = %+v", b)
	}
	if b.Source.AckedSHA256 != fileSHA(t, external) {
		t.Fatalf("external hash = %s", b.Source.AckedSHA256)
	}
	fi, err := os.Stat(external)
	if err != nil {
		t.Fatal(err)
	}
	got := rows(t, home, `SELECT path, location, kind, acked_sha256, size_bytes, removed_seq FROM sources ORDER BY registered_seq`)
	want := [][]string{
		{"docs/client-brief.md", "in_repo", "client_requirement", fileSHA(t, brief), "2255", "NULL"},
		{external, "external", "other", fileSHA(t, external), strconv.FormatInt(fi.Size(), 10), "NULL"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sources = %v, want %v", got, want)
	}
	list := sourceList(t, repo)
	if len(list.Sources) != 2 || list.SourceSetHash != b.SourceSetHash || list.ExcerptStorage != "hash_only" {
		t.Fatalf("list = %+v", list)
	}
}

// MVP-INIT-04 (the part left to WP1.4): candidates are not registered.
func TestInitRegistersNoSource(t *testing.T) {
	testutil.TempHome(t)
	f := testutil.BuildFixture(t, "fx-docs-only")
	mustInit(t, f.Repo.Dir)
	if list := sourceList(t, f.Repo.Dir); len(list.Sources) != 0 {
		t.Fatalf("sources after init = %+v", list.Sources)
	}
}

// MVP-SRC-07 and data-model §4.3: adding an active path again does
// nothing — no new fact, no seq, and the acknowledged hash stays, even
// after the file changed (only a person's source ack updates it).
func TestSourceAddAgainChangesNothing(t *testing.T) {
	home, repo, _ := sourcesProject(t)
	first := sourceAdd(t, repo, "docs/client-brief.md")
	before := seq(t, home)
	again := sourceAdd(t, filepath.Join(repo, "docs"), "client-brief.md")
	if again.Created || again.Source.ID != first.Source.ID || again.SourceSetHash != first.SourceSetHash {
		t.Fatalf("second add = %+v", again)
	}
	brief := filepath.Join(repo, "docs", "client-brief.md")
	if err := os.WriteFile(brief, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	third := sourceAdd(t, repo, "docs/client-brief.md")
	if third.Created || third.Source.AckedSHA256 != first.Source.AckedSHA256 || *third.Source.State != "changed" {
		t.Fatalf("add after a change = %+v", third.Source)
	}
	if after := seq(t, home); after != before {
		t.Fatalf("seq moved from %s to %s", before, after)
	}
	if n := count(t, home, `SELECT count(*) FROM sources`); n != 1 {
		t.Fatalf("%d sources", n)
	}
}

// Concurrent adds of one path register one source.
func TestSourceAddConcurrently(t *testing.T) {
	home, repo, _ := sourcesProject(t)
	t.Chdir(repo)
	const n = 4
	outs := make([]sourceAddOut, n)
	codes := make([]int, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			var stdout, stderr strings.Builder
			codes[i] = Execute(context.Background(), []string{"source", "add", "--json", "docs/client-brief.md"}, &stdout, &stderr)
			_ = json.Unmarshal([]byte(stdout.String()), &outs[i])
		})
	}
	wg.Wait()
	created := 0
	for i := range n {
		if codes[i] != apperr.ExitOK || outs[i].Source.ID != outs[0].Source.ID {
			t.Fatalf("run %d: exit %d, %+v", i, codes[i], outs[i])
		}
		if outs[i].Created {
			created++
		}
	}
	if created != 1 || count(t, home, `SELECT count(*) FROM sources`) != 1 {
		t.Fatalf("%d runs created a source", created)
	}
}

// MVP-SRC-05, MVP-SRC-07 (type) and the other refusals: exit 2 or 4, and
// nothing is written.
func TestSourceAddRefuses(t *testing.T) {
	home, repo, _ := sourcesProject(t)
	secret := "ghp_" + strings.Repeat("A1b2", 9)
	for _, f := range []string{"internal/auth/login.go", "docs/" + secret + ".md"} {
		p := filepath.Join(repo, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	before := seq(t, home)
	for _, c := range []struct {
		args []string
		code int
	}{
		{[]string{"docs/missing.md"}, apperr.ExitInvalidInput},
		{[]string{"docs"}, apperr.ExitInvalidInput},
		{[]string{"internal/auth/login.go"}, apperr.ExitInvalidInput},
		{[]string{"docs/" + secret + ".md"}, apperr.ExitInvalidInput},
		{[]string{"docs/missing-" + secret + ".md"}, apperr.ExitInvalidInput},
		{[]string{"docs/client-brief.md", "--kind", "spec"}, apperr.ExitInvalidInput},
		{[]string{".git/HEAD"}, apperr.ExitInvalidInput},
		{[]string{"docs/client-brief.md", "--allow-any-type"}, apperr.ExitHumanRequired},
	} {
		code, stdout, stderr := specmetry(t, repo, "source-add.v1", append([]string{"source", "add", "--json"}, c.args...)...)
		if code != c.code {
			t.Errorf("source add %v: exit %d, want %d\n%s%s", c.args, code, c.code, stdout, stderr)
		}
		if strings.Contains(stdout+stderr, secret) {
			t.Errorf("source add %v echoed the secret", c.args)
		}
	}
	if after := seq(t, home); after != before {
		t.Fatalf("seq moved from %s to %s", before, after)
	}
	if n := count(t, home, `SELECT count(*) FROM sources`); n != 0 {
		t.Fatalf("%d sources", n)
	}
}

// Outside an initialized project the source commands exit 4 and create
// nothing.
func TestSourceCommandsNeedAProject(t *testing.T) {
	home := testutil.TempHome(t)
	repo := committedRepo(t)
	for _, args := range [][]string{{"source", "list", "--json"}, {"source", "add", "--json", "README.md"}} {
		code, stdout, _ := specmetry(t, repo.Dir, "source-list.v1", args...)
		if code != apperr.ExitNotInitialized {
			t.Errorf("%v: exit %d\n%s", args, code, stdout)
		}
	}
	if exists(home.Path) {
		t.Fatal("the data directory was created")
	}
}

// MVP-SRC-02: every list observes the disk; the acknowledged hash stays.
func TestSourceListObservesTheDisk(t *testing.T) {
	_, repo, external := sourcesProject(t)
	brief := filepath.Join(repo, "docs", "client-brief.md")
	a := sourceAdd(t, repo, "docs/client-brief.md")
	sourceAdd(t, repo, external)
	if err := os.WriteFile(brief, []byte("# brief, revised\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(external); err != nil {
		t.Fatal(err)
	}
	list := sourceList(t, repo)
	states := map[string]sourceOut{}
	for _, s := range list.Sources {
		states[s.Path] = s
	}
	if s := states["docs/client-brief.md"]; *s.State != "changed" || *s.CurrentSHA256 != fileSHA(t, brief) || s.AckedSHA256 != a.Source.AckedSHA256 {
		t.Fatalf("changed source = %+v", s)
	}
	if s := states[external]; *s.State != "missing" || s.CurrentSHA256 != nil {
		t.Fatalf("missing source = %+v", s)
	}
	// --refs: no references exist before WP1.5 writes them.
	for _, s := range sourceList(t, repo, "--refs").Sources {
		if s.Refs == nil || len(*s.Refs) != 0 {
			t.Fatalf("refs of %s = %v", s.Path, s.Refs)
		}
	}
	for _, s := range list.Sources {
		if s.Refs != nil {
			t.Fatalf("refs listed without --refs: %+v", s)
		}
	}
}

// A source that cannot be read is reported as missing; its acknowledged
// hash and the source set are untouched.
func TestSourceListUnreadableSource(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a file the current user cannot read")
	}
	home, repo, _ := sourcesProject(t)
	a := sourceAdd(t, repo, "docs/client-brief.md")
	brief := filepath.Join(repo, "docs", "client-brief.md")
	if err := os.Chmod(brief, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(brief, 0o644) })
	list := sourceList(t, repo)
	if s := list.Sources[0]; *s.State != "missing" || s.AckedSHA256 != a.Source.AckedSHA256 || list.SourceSetHash != a.SourceSetHash {
		t.Fatalf("unreadable source = %+v", list)
	}
	if got := rows(t, home, `SELECT acked_sha256 FROM sources`); got[0][0] != a.Source.AckedSHA256 {
		t.Fatal("the acknowledged hash changed")
	}
}

// MVP-SRC-03 (non-TTY part) and CX-17: the Agent channel cannot
// acknowledge or remove; nothing is written and the change stays visible.
func TestSourceAckAndRemoveRefuseTheAgent(t *testing.T) {
	home, repo, _ := sourcesProject(t)
	a := sourceAdd(t, repo, "docs/client-brief.md")
	brief := filepath.Join(repo, "docs", "client-brief.md")
	if err := os.WriteFile(brief, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := seq(t, home)
	for _, args := range [][]string{
		{"source", "ack", "docs/client-brief.md", "--observed", fileSHA(t, brief), "-m", "typo", "--json"},
		{"source", "remove", "docs/client-brief.md", "--json"},
		{"source", "remove", "docs/client-brief.md", "--erase-excerpts"},
	} {
		code, stdout, stderr := specmetry(t, repo, "error.v1", args...)
		if code != apperr.ExitHumanRequired {
			t.Errorf("%v: exit %d\n%s%s", args, code, stdout, stderr)
		}
	}
	if after := seq(t, home); after != before {
		t.Fatalf("seq moved from %s to %s", before, after)
	}
	s := sourceList(t, repo).Sources[0]
	if *s.State != "changed" || s.AckedSHA256 != a.Source.AckedSHA256 || s.RemovedSeq != nil {
		t.Fatalf("source after refused writes = %+v", s)
	}
}

// MVP-SRC-03 (terminal part): a person acknowledges a change bound to the
// hash on disk; a stale observed hash is refused with exit 5.
func TestSourceAckInTerminal(t *testing.T) {
	home, repo, _ := sourcesProject(t)
	a := sourceAdd(t, repo, "docs/client-brief.md")
	brief := filepath.Join(repo, "docs", "client-brief.md")

	// Nothing to acknowledge while the file is unchanged.
	out := inTerminal(t, repo, []string{"source", "ack", "docs/client-brief.md", "--observed", a.Source.AckedSHA256, "-m", "typo"}, noPrompt)
	if !strings.Contains(out, "exit=0") || !strings.Contains(out, "nothing to acknowledge") {
		t.Fatalf("unchanged:\n%s", out)
	}

	if err := os.WriteFile(brief, []byte("# brief, revised\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	current := fileSHA(t, brief)
	before := seq(t, home)
	// An observed hash that is not on disk is refused before the prompt.
	stale := strings.Repeat("0", 64)
	out = inTerminal(t, repo, []string{"source", "ack", "docs/client-brief.md", "--observed", stale, "-m", "typo"}, noPrompt)
	if !strings.Contains(out, "exit=5") || !strings.Contains(out, current) {
		t.Fatalf("stale observed hash:\n%s", out)
	}
	// The file changes while the person reads the prompt: the hash on disk
	// when the acknowledgement is written is what counts.
	out = inTerminalChanging(t, repo, []string{"source", "ack", "docs/client-brief.md", "--observed", current, "-m", "typo"}, "docs/client-brief.md", func() {
		if err := os.WriteFile(brief, []byte("# brief, revised again\n"), 0o600); err != nil {
			t.Error(err)
		}
	})
	if !strings.Contains(out, "exit=5") {
		t.Fatalf("changed during the prompt:\n%s", out)
	}
	if err := os.WriteFile(brief, []byte("# brief, revised\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out = inTerminal(t, repo, []string{"source", "ack", "docs/client-brief.md", "--observed", current, "-m", "typo"}, "")
	if !strings.Contains(out, "exit=3") || !strings.Contains(out, "not confirmed") {
		t.Fatalf("declined:\n%s", out)
	}
	if after := seq(t, home); after != before {
		t.Fatalf("seq moved from %s to %s before the acknowledgement", before, after)
	}

	out = inTerminal(t, repo, []string{"source", "ack", "docs/client-brief.md", "--observed", current, "-m", "只修正了错别字，基准无需变更"}, "docs/client-brief.md")
	if !strings.Contains(out, "exit=0") {
		t.Fatalf("acknowledged:\n%s", out)
	}
	acks := rows(t, home, `SELECT k.source_id, k.previous_sha256, k.acked_sha256, k.rationale, k.channel, substr(k.actor, 1, 6), s.acked_sha256
FROM source_acks k JOIN sources s ON s.id = k.source_id`)
	want := [][]string{{a.Source.ID, a.Source.AckedSHA256, current, "只修正了错别字，基准无需变更", "human_cli", "human:", current}}
	if !reflect.DeepEqual(acks, want) {
		t.Fatalf("acknowledgement = %v, want %v", acks, want)
	}
	list := sourceList(t, repo)
	if *list.Sources[0].State != "unchanged" || list.SourceSetHash == a.SourceSetHash {
		t.Fatalf("after the acknowledgement: %+v", list)
	}
}

// A rationale that looks like a secret is refused before anything is
// written.
func TestSourceAckRefusesSecretRationale(t *testing.T) {
	home, repo, _ := sourcesProject(t)
	sourceAdd(t, repo, "docs/client-brief.md")
	brief := filepath.Join(repo, "docs", "client-brief.md")
	if err := os.WriteFile(brief, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	secret := "ghp_" + strings.Repeat("A1b2", 9)
	before := seq(t, home)
	out := inTerminal(t, repo, []string{"source", "ack", "docs/client-brief.md", "--observed", fileSHA(t, brief), "-m", "token " + secret}, noPrompt)
	if !strings.Contains(out, "exit=2") || strings.Contains(out, secret) {
		t.Fatalf("secret rationale:\n%s", out)
	}
	if after := seq(t, home); after != before {
		t.Fatal("written")
	}
}

// MVP-SRC-04 (without references, which arrive with WP1.5) and MVP-SRC-08
// (M1, hash_only): a person removes a source; it leaves the active set
// and the set hash changes; registering the path again makes a new source.
func TestSourceRemoveInTerminal(t *testing.T) {
	home, repo, external := sourcesProject(t)
	a := sourceAdd(t, repo, "docs/client-brief.md")
	b := sourceAdd(t, repo, external)

	out := inTerminal(t, repo, []string{"source", "remove", "docs/client-brief.md"}, "")
	if !strings.Contains(out, "exit=3") {
		t.Fatalf("declined:\n%s", out)
	}
	out = inTerminal(t, repo, []string{"source", "remove", "docs/client-brief.md", "--erase-excerpts"}, "docs/client-brief.md")
	if !strings.Contains(out, "exit=0") || !strings.Contains(out, "hash_only") {
		t.Fatalf("removed:\n%s", out)
	}
	list := sourceList(t, repo)
	if len(list.Sources) != 1 || list.Sources[0].ID != b.Source.ID || list.SourceSetHash == b.SourceSetHash {
		t.Fatalf("after removal: %+v", list)
	}
	all := sourceList(t, repo, "--include-removed", "--refs")
	if len(all.Sources) != 2 || all.SourceSetHash != list.SourceSetHash {
		t.Fatalf("--include-removed: %+v", all)
	}
	for _, s := range all.Sources {
		if s.ID == a.Source.ID && (s.RemovedSeq == nil || s.State != nil || s.CurrentSHA256 != nil || len(*s.Refs) != 0) {
			t.Fatalf("removed source = %+v", s)
		}
	}
	again := sourceAdd(t, repo, "docs/client-brief.md")
	if !again.Created || again.Source.ID == a.Source.ID {
		t.Fatalf("registered again = %+v", again)
	}
	if got := rows(t, home, `SELECT count(*) FROM sources WHERE path = 'docs/client-brief.md'`); got[0][0] != "2" {
		t.Fatalf("rows for the path = %v", got)
	}
	out = inTerminal(t, repo, []string{"source", "remove", a.Source.ID}, noPrompt)
	if !strings.Contains(out, "exit=2") {
		t.Fatalf("removing a removed source:\n%s", out)
	}
}

// MVP-SRC-07 (--allow-any-type): only a person, after confirming.
func TestSourceAddAnyTypeInTerminal(t *testing.T) {
	home, repo, _ := sourcesProject(t)
	login := filepath.Join(repo, "internal", "auth", "login.go")
	if err := os.MkdirAll(filepath.Dir(login), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(login, []byte("package auth\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := inTerminal(t, repo, []string{"source", "add", "internal/auth/login.go", "--allow-any-type"}, "")
	if !strings.Contains(out, "exit=3") || count(t, home, `SELECT count(*) FROM sources`) != 0 {
		t.Fatalf("declined:\n%s", out)
	}
	out = inTerminal(t, repo, []string{"source", "add", "internal/auth/login.go", "--allow-any-type"}, "internal/auth/login.go")
	if !strings.Contains(out, "exit=0") {
		t.Fatalf("confirmed:\n%s", out)
	}
	if got := rows(t, home, `SELECT path, location FROM sources`); !reflect.DeepEqual(got, [][]string{{"internal/auth/login.go", "in_repo"}}) {
		t.Fatalf("sources = %v", got)
	}
}

// MVP-CH-01 with the real commands: an Agent variable refuses source ack,
// source remove and source add --allow-any-type with exit 3, before any
// prompt; nothing is written; the variable's value is in neither the
// output nor the data directory.
func TestSourceHumanOnlyCommandsRefuseAgentInTerminal(t *testing.T) {
	home, repo, _ := sourcesProject(t)
	sourceAdd(t, repo, "docs/client-brief.md")
	brief := filepath.Join(repo, "docs", "client-brief.md")
	if err := os.WriteFile(brief, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	canary := testutil.NewCanary(t)
	before := seq(t, home)
	for _, args := range [][]string{
		{"source", "ack", "docs/client-brief.md", "--observed", fileSHA(t, brief), "-m", "typo"},
		{"source", "remove", "docs/client-brief.md"},
		{"source", "add", "README.md", "--allow-any-type", "--json"},
	} {
		for _, name := range []string{"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT"} {
			out := inTerminal(t, repo, args, noPrompt, name+"="+canary)
			if !strings.Contains(out, "exit=3") || strings.Contains(out, "to confirm") || strings.Contains(out, canary) {
				t.Errorf("%v with %s:\n%s", args, name, out)
			}
		}
	}
	if after := seq(t, home); after != before {
		t.Fatalf("seq moved from %s to %s", before, after)
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("the value reached the data directory: %v", hits)
	}
}

// MVP-CH-02 with the real commands: stdout redirected, or stdin not a
// terminal, is agent_cli; --json does not change it; source list still
// works.
func TestSourceHumanOnlyCommandsRefuseRedirection(t *testing.T) {
	home, repo, _ := sourcesProject(t)
	sourceAdd(t, repo, "docs/client-brief.md")
	before := seq(t, home)
	args := []string{"source", "remove", "docs/client-brief.md", "--json"}
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	env := helperEnv(t, "cli-redirected", argsJSONEnv+"="+string(raw), dirEnv+"="+repo)
	cmd := testutil.HelperCommand(t, "cli-redirected")
	out := testutil.RunInPTYSession(t, 60*time.Second, env, nil, cmd.Path, cmd.Args[1:]...).Output
	if !strings.Contains(out, "exit=3") || strings.Contains(out, "to confirm") {
		t.Fatalf("stdout redirected:\n%s", out)
	}
	piped := testutil.HelperCommand(t, "cli")
	piped.Env = helperEnv(t, "cli", argsJSONEnv+"="+string(raw), dirEnv+"="+repo)
	b, _ := piped.CombinedOutput()
	if !strings.Contains(string(b), "exit=3") {
		t.Fatalf("stdin not a terminal:\n%s", b)
	}
	if after := seq(t, home); after != before {
		t.Fatal("written")
	}
	listArgs, _ := json.Marshal([]string{"source", "list", "--json"})
	listed := testutil.HelperCommand(t, "cli")
	listed.Env = helperEnv(t, "cli", argsJSONEnv+"="+string(listArgs), dirEnv+"="+repo)
	if b, _ := listed.CombinedOutput(); !strings.Contains(string(b), "exit=0") || !strings.Contains(string(b), "client-brief.md") {
		t.Fatalf("source list in agent_cli:\n%s", b)
	}
}

// hash_only and the privacy contract: the content of a source never
// reaches the data directory or the output, through every command.
func TestSourceContentIsNeverStored(t *testing.T) {
	home, repo, external := sourcesProject(t)
	canary := testutil.NewCanary(t)
	if err := os.WriteFile(external, []byte("# 需求\n"+canary+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var outputs strings.Builder
	a := sourceAdd(t, repo, external)
	_, stdout, stderr := specmetry(t, repo, "", "source", "list", "--include-removed", "--refs")
	outputs.WriteString(stdout + stderr)
	if err := os.WriteFile(external, []byte("# 需求\n"+canary+" v2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	outputs.WriteString(inTerminal(t, repo, []string{"source", "ack", a.Source.ID, "--observed", fileSHA(t, external), "-m", "reviewed"}, a.Source.ID))
	outputs.WriteString(inTerminal(t, repo, []string{"source", "remove", a.Source.ID}, a.Source.ID))
	if n := count(t, home, `SELECT count(*) FROM source_acks`); n != 1 {
		t.Fatalf("%d acknowledgements", n)
	}
	raw, _ := json.Marshal(sourceList(t, repo, "--include-removed", "--refs"))
	outputs.Write(raw)
	if strings.Contains(outputs.String(), canary) {
		t.Fatal("source content reached the output")
	}
	if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
		t.Fatalf("source content reached the data directory: %v", hits)
	}
}

// A cancelled list or add stops, writes nothing and prints no document.
func TestSourceCommandsStopWhenCanceled(t *testing.T) {
	home, repo, _ := sourcesProject(t)
	sourceAdd(t, repo, "docs/client-brief.md")
	before := seq(t, home)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	t.Chdir(repo)
	for _, args := range [][]string{{"source", "list", "--json"}, {"source", "add", "--json", "README.md"}} {
		var stdout, stderr strings.Builder
		code := Execute(canceled, args, &stdout, &stderr)
		if code == apperr.ExitOK || strings.Contains(stdout.String(), "specmetry.source") {
			t.Fatalf("%v: exit %d\n%s", args, code, stdout.String())
		}
	}
	if after := seq(t, home); after != before {
		t.Fatal("written")
	}
}
