package job

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/controls"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
)

// bound is the regressions' own failure exit: a wait that should end
// almost at once fails the test after it instead of hanging the process. It
// never orders events; ordering comes from the fake clock and the blocking
// fake's entered and release channels.
const bound = 10 * time.Second

// blockingTC is a TypeChecker that signals when the analysis entered it,
// then blocks until released; after the release it runs the real type
// checker, or returns a real failure when fail is set.
type blockingTC struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	rel     sync.Once
	before  func() // runs after the release, before returning
	fail    bool
}

func newBlockingTC() *blockingTC {
	return &blockingTC{entered: make(chan struct{}), release: make(chan struct{})}
}

func (b *blockingTC) free() { b.rel.Do(func() { close(b.release) }) }

func (b *blockingTC) Check(in collect.TCInput) collect.TCOutput {
	b.once.Do(func() { close(b.entered) })
	<-b.release
	if b.before != nil {
		b.before()
	}
	if b.fail {
		return collect.TCOutput{Failures: []judge.Failure{judge.Real(judge.J3, "example.com/q", "type check example.com/q", "type error", "a real late error")}}
	}
	return collect.TypeCheck{}.Check(in)
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) bool {
	t.Helper()
	select {
	case <-ch:
		return true
	case <-time.After(bound):
		t.Errorf("did not happen within %s: %s", bound, what)
		return false
	}
}

// Review 147 B-147-1: an admitted analysis that never returns must not keep
// the parent from W: RunDomains returns at W, and the closeout decides.
func TestB1471ParentLeavesAStuckAnalysisAtW(t *testing.T) {
	cfg, _ := newConfig(t, nil)
	clk := chain.NewFakeClock(time.Unix(1_000_000, 0))
	cfg.Clock, cfg.Start = clk, clk.Now()
	r, inv, guard, err := controls.FixtureRepo(filepath.Join(t.TempDir(), "fx"), runtime.GOOS, clk)
	if err != nil {
		t.Fatal(err)
	}
	cfg.R, cfg.Invoker, cfg.Guard = r, inv, guard
	tc := newBlockingTC()
	t.Cleanup(tc.free)
	cfg.TypeCheck = tc
	j, err := Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ran := make(chan struct{})
	go func() { j.RunDomains(); close(ran) }()
	if !waitFor(t, tc.entered, "the analysis entered the type check") {
		return
	}
	clk.Set(j.W)
	if !waitFor(t, ran, "RunDomains returned at W while the analysis is stuck") {
		tc.free()
		waitFor(t, ran, "RunDomains after the release")
		return
	}
	done := make(chan chain.Final, 1)
	go func() { done <- j.Finish(0) }()
	select {
	case f := <-done:
		if f.Conclusion == chain.Consistent || f.Conclusion == chain.Inconsistent || f.ExitCode == 0 {
			t.Errorf("a stuck analysis gave %v exit %d", f.Conclusion, f.ExitCode)
		}
	case <-time.After(bound):
		t.Error("Finish did not return")
	}
}

// Review 147 B-147-4: in a healthy job every derived record names its
// parents (name and SHA-256) and every parent resolves to an earlier record
// with that SHA-256.
func TestB1474DerivedRecordsNameResolvableParents(t *testing.T) {
	cfg, _ := newConfig(t, controlsWithout(t, "LC-9"))
	f := run(t, cfg, true, 0)
	if f.ExitCode != 0 {
		t.Fatalf("healthy job: %v %v", f.Conclusion, f.Reasons)
	}
	seen := map[string]string{}
	for _, name := range []string{FileRaw, FileDerived} {
		b, err := os.ReadFile(filepath.Join(cfg.RecordsDir, name))
		if err != nil {
			t.Fatal(err)
		}
		sc := bufio.NewScanner(bytes.NewReader(b))
		sc.Buffer(make([]byte, 1<<20), 1<<26)
		first := true
		for sc.Scan() {
			if first {
				first = false
				continue // the file header
			}
			line := sc.Bytes()
			var it struct {
				Name    string `json:"name"`
				Parents []struct {
					Name   string `json:"name"`
					SHA256 string `json:"sha256"`
				} `json:"parents"`
			}
			if err := json.Unmarshal(line, &it); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			sum := sha256.Sum256(line)
			if name == FileDerived {
				if len(it.Parents) == 0 {
					t.Errorf("derived %s has no parent", it.Name)
				}
				for _, p := range it.Parents {
					if got, ok := seen[p.Name]; !ok || got != p.SHA256 {
						t.Errorf("derived %s: parent %s %s does not resolve (have %q)", it.Name, p.Name, p.SHA256, got)
					}
				}
			}
			seen[it.Name] = hex.EncodeToString(sum[:])
		}
	}
}

// Review 147 B-147-4: the controls' records keep the sites of the tested
// failures, not only flattened lines.
func TestB1474ControlRecordsKeepSites(t *testing.T) {
	cfg, _ := newConfig(t, controls.LC3())
	run(t, cfg, true, 0)
	b, err := os.ReadFile(filepath.Join(cfg.RecordsDir, FileControls))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"Site":"I-5 example.com/lc3@v1.0.0/lc3.go (P parse)"`) {
		t.Errorf("the LC-3 records do not keep the failure site")
	}
}
