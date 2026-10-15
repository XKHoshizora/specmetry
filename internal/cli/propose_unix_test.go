//go:build !windows

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// codex-review-020 B5: a registered source replaced by a FIFO (with no
// writer) does not stop the commands that read sources: propose returns
// at once and marks its excerpts source_missing, storing the proposal with
// one seq; source list reports the source missing.
func TestSourceReplacedByAFIFO(t *testing.T) {
	home, f, _ := proposalsProject(t)
	brief := filepath.Join(f.Repo.Dir, "docs", "client-brief.md")
	if err := os.Remove(brief); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(brief, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := strconv.Atoi(seq(t, home))
	if err != nil {
		t.Fatal(err)
	}
	path := writeProposal(t, recorded(t, f))
	t.Chdir(f.Repo.Dir)
	run := func(args ...string) (int, string) {
		t.Helper()
		type result struct {
			code   int
			stdout string
		}
		done := make(chan result, 1)
		go func() {
			var stdout, stderr bytes.Buffer
			code := Execute(context.Background(), args, &stdout, &stderr)
			done <- result{code, stdout.String() + stderr.String()}
		}()
		select {
		case r := <-done:
			return r.code, r.stdout
		case <-time.After(30 * time.Second):
			t.Fatalf("%v blocked on the FIFO", args)
		}
		return 0, ""
	}
	code, out := run("propose", "--file", path, "--json")
	if code != apperr.ExitOK {
		t.Fatalf("propose: exit %d\n%s", code, out)
	}
	var r receiptOut
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, it := range r.Items {
		if it.Sources[0].Verification != "source_missing" {
			t.Errorf("item %d: %s", it.ItemIndex, it.Sources[0].Verification)
		}
	}
	if s, _ := strconv.Atoi(seq(t, home)); s != before+1 || count(t, home, `SELECT count(*) FROM proposals`) != 1 {
		t.Fatalf("seq %d after %d", s, before)
	}
	code, out = run("source", "list", "--json")
	if code != apperr.ExitOK || !bytes.Contains([]byte(out), []byte(`"state": "missing"`)) {
		t.Fatalf("source list: exit %d\n%s", code, out)
	}
}
