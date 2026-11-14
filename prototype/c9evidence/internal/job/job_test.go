package job

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/controls"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// syncBuf is a goroutine-safe stderr for the diagnostic task.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// canSymlink reports whether this machine can create a symbolic link; the
// LC-2 symbolic link variant is a miss where it cannot (plan §5.6 item 4).
func canSymlink(t *testing.T) bool {
	d := t.TempDir()
	return os.Symlink("x", filepath.Join(d, "l")) == nil
}

// controlsWithout returns All(GOOS) minus the named controls, and minus the
// LC-2 symbolic link variant on a machine that cannot create links.
func controlsWithout(t *testing.T, drop ...string) []controls.Variant {
	skip := map[string]bool{}
	for _, d := range drop {
		skip[d] = true
	}
	link := canSymlink(t)
	var out []controls.Variant
	for _, v := range controls.All(runtime.GOOS) {
		if skip[v.Control] || (!link && v.Control == "LC-2" && v.Name == "symbolic link") {
			continue
		}
		out = append(out, v)
	}
	return out
}

func newConfig(t *testing.T, vs []controls.Variant) (Config, *syncBuf) {
	t.Helper()
	root := t.TempDir()
	r, inv, guard, err := controls.FixtureRepo(filepath.Join(root, "fx"), runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	errOut := &syncBuf{}
	return Config{
		GOOS: runtime.GOOS, R: r, RecordsDir: filepath.Join(root, "records"), Clock: chain.RealClock{}, Start: time.Now(),
		WriteFS: chain.OSFS{}, ReadFS: modsnap.OS{}, Stderr: errOut, Invoker: inv, Guard: guard, TypeCheck: collect.TypeCheck{},
		Controls: vs, FixtureRoot: "", MainModule: "example.com/rf",
		Candidates: func(spec.Config) []spec.Candidate { return nil }, Watch: spec.WatchPackages,
		Identity: map[string]string{"GITHUB_SHA": "test"}, T2: spec.T2,
	}, errOut
}

func run(t *testing.T, cfg Config, domains bool, mRun int) chain.Final {
	t.Helper()
	j, err := Begin(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if domains {
		j.RunDomains()
	}
	return j.Finish(mRun)
}

func TestJobConsistent(t *testing.T) {
	cfg, errOut := newConfig(t, controlsWithout(t, "LC-9"))
	f := run(t, cfg, true, 0)
	if f.Conclusion != chain.Consistent || f.ExitCode != 0 {
		t.Fatalf("final %v exit %d path %v reasons %v\nstderr:\n%s", f.Conclusion, f.ExitCode, f.Path, f.Reasons, errOut.String())
	}
	// The records are complete in the sense of plan §6.5: the manifest
	// lists every record file with its size and SHA-256, the directory has
	// nothing else, and the terminator line carries the manifest's SHA-256.
	man, err := os.ReadFile(filepath.Join(cfg.RecordsDir, "manifest.txt"))
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{"manifest.txt": true}
	sc := bufio.NewScanner(bytes.NewReader(man))
	for sc.Scan() {
		parts := strings.Fields(sc.Text())
		if len(parts) != 3 {
			t.Fatalf("manifest line %q", sc.Text())
		}
		b, err := os.ReadFile(filepath.Join(cfg.RecordsDir, parts[0]))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(b)
		if strconv.Itoa(len(b)) != parts[1] || hex.EncodeToString(sum[:]) != parts[2] {
			t.Errorf("manifest %q does not match the file", sc.Text())
		}
		listed[parts[0]] = true
	}
	ents, _ := os.ReadDir(cfg.RecordsDir)
	for _, e := range ents {
		if !listed[e.Name()] {
			t.Errorf("unlisted file %s", e.Name())
		}
	}
	msum := sha256.Sum256(man)
	if !strings.Contains(errOut.String(), hex.EncodeToString(msum[:])) || !strings.Contains(errOut.String(), chain.TerminatorLine(hex.EncodeToString(msum[:]), chain.Consistent)) {
		t.Errorf("no terminator line with the manifest SHA-256 in stderr:\n%s", errOut.String())
	}
	if len(listed) != 5 { // four record files and the manifest
		t.Errorf("record files: %v", listed)
	}
}

func TestJobPhase1IsInvalid(t *testing.T) {
	// With every control, LC-9 is applicable and not implemented: a miss.
	cfg, _ := newConfig(t, controls.All(runtime.GOOS))
	f := run(t, cfg, true, 0)
	if f.Conclusion != chain.Invalid || f.ExitCode == 0 {
		t.Fatalf("final %v exit %d %v", f.Conclusion, f.ExitCode, f.Reasons)
	}
}

func TestJobStoppedInvoker(t *testing.T) {
	// The real boundary of phase 1 starts nothing: every first call fails
	// (J-1) and R is incomplete.
	cfg, errOut := newConfig(t, controlsWithout(t, "LC-9"))
	cfg.Invoker = collect.Stopped{Clock: chain.RealClock{}}
	f := run(t, cfg, true, 0)
	if f.Conclusion != chain.Incomplete || f.ExitCode == 0 {
		t.Fatalf("final %v exit %d %v", f.Conclusion, f.ExitCode, f.Reasons)
	}
	if !strings.Contains(errOut.String(), "E-10 (ii)") {
		t.Errorf("the stopped boundary is not in the diagnostics:\n%s", errOut.String())
	}
}

func TestJobWithoutR(t *testing.T) {
	cfg, _ := newConfig(t, controlsWithout(t, "LC-9"))
	cfg.R = ""
	if f := run(t, cfg, true, 0); f.Conclusion != chain.Incomplete {
		t.Fatalf("final %v %v", f.Conclusion, f.Reasons)
	}
}

func TestJobMRunNonzero(t *testing.T) {
	cfg, _ := newConfig(t, controlsWithout(t, "LC-9"))
	if f := run(t, cfg, true, 1); f.Conclusion != chain.Invalid || f.ExitCode == 0 {
		t.Fatalf("final %v %v", f.Conclusion, f.Reasons)
	}
}

func TestJobDomainsNotRun(t *testing.T) {
	// The test function did not run: every applicable control is a miss.
	cfg, _ := newConfig(t, controlsWithout(t, "LC-9"))
	if f := run(t, cfg, false, 0); f.Conclusion != chain.Invalid {
		t.Fatalf("final %v %v", f.Conclusion, f.Reasons)
	}
}

func TestJobBadT2(t *testing.T) {
	cfg, _ := newConfig(t, controlsWithout(t, "LC-9"))
	cfg.T2 = spec.T2 + " "
	if f := run(t, cfg, true, 0); f.Conclusion != chain.Invalid {
		t.Fatalf("final %v %v", f.Conclusion, f.Reasons)
	}
}
