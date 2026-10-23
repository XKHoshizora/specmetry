package wp25

// Direct checks of the outer gate check's own rules (Review 077
// B1-R2-W), apart from any process: the judgment of a witness file, the
// identity of a recorded process, and the rule that every process is
// started through start. These are not gate regressions and not mechanism
// evidence; their expectations are fixed here, not computed by the code
// they check. The gate regressions (gate_test.go) show that real faults
// reach these rules.

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestStartPoints checks the source: a process is started only through
// start (witness_test.go), apart from the listed exceptions.
func TestStartPoints(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	allowed := map[string]string{
		"witness_test.go":         "if err := cmd.Start(); err != nil {", // start itself
		"witness_unix_test.go":    "if err := cmd.Start(); err != nil {", // the outer check starts the inner run
		"witness_windows_test.go": "if err := cmd.Start(); err != nil {", // the same, in its Job
		"gate_test.go":            "err = cmd.Start() // bypass: allowed in TestStartPoints for this line only",
	}
	pat := regexp.MustCompile(`\.Start\(\)|\.Run\(\)|\.Output\(\)|\.CombinedOutput\(\)|StartProcess|ForkExec|CreateProcess`)
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".go") || f.Name() == "witness_check_test.go" { // not this rule's own text
			continue
		}
		b, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		n++
		for i, l := range strings.Split(string(b), "\n") {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "//") || !pat.MatchString(l) || l == "code := m.Run()" || l == allowed[f.Name()] {
				continue
			}
			t.Errorf("%s:%d: a process started past start: %s", f.Name(), i+1, l)
		}
	}
	switch {
	case n == 0:
		t.Skip("the sources are not in the working directory (a copied test binary)")
	case n < 20:
		t.Fatalf("only %d source files read", n)
	}
}

// identity forms of this platform, for the judgment cases.
func ident(n int) string {
	if runtime.GOOS == "darwin" {
		return fmt.Sprintf("%d.%06d", 1700000000+n, n)
	}
	return fmt.Sprint(1000000 + n)
}

func TestWitnessJudge(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	const inner, helper1, child1 = 100, 200, 300
	root := fmt.Sprintf("root %d %s %d inner\n", inner, ident(1), inner)
	created := fmt.Sprintf("created %d %s %d silent-parent\n", helper1, ident(2), inner)
	launched := fmt.Sprintf("launched %d %s %d silent-parent\n", helper1, ident(2), helper1)
	created2 := fmt.Sprintf("created %d %s %d sleep\n", child1, ident(3), helper1)
	launched2 := fmt.Sprintf("launched %d %s %d sleep\n", child1, ident(3), child1)
	healthy := root + created + launched + created2 + launched2
	sealOf := func(records, failed int) string {
		return fmt.Sprintf(`{"pid":%d,"start":"%s","records":%d,"failed":%d}`, inner, ident(1), records, failed)
	}
	good := []string{sealOf(2, 0)}
	cases := []struct {
		name   string
		file   string
		err    error
		seals  []string
		raised bool
		total  int
		want   string // "" accepted, else the class of the first problem
	}{
		{name: "healthy", file: healthy, seals: good, total: -1},
		{name: "healthy, a child created and ended before its record", file: root + created + launched + created2, seals: good, total: -1},
		{name: "healthy, Job count agrees", file: healthy, seals: good, total: 3},
		{name: "empty file, empty seal", file: "", seals: []string{sealOf(0, 0)}, total: -1, want: "witness-root"},
		{name: "empty file, no seal", file: "", total: -1, want: "witness-root"},
		{name: "unreadable", err: errors.New("no such file"), seals: good, total: -1, want: "witness-unreadable"},
		{name: "last record cut", file: strings.TrimSuffix(healthy, "\n"), seals: good, total: -1, want: "witness-malformed"},
		{name: "unknown kind", file: healthy + "bogus 5 " + ident(5) + " 5 x\n", seals: good, total: -1, want: "witness-malformed"},
		{name: "duplicate", file: healthy + launched2, seals: good, total: -1, want: "witness-malformed"},
		{name: "unknown identity", file: root + created + fmt.Sprintf("launched %d unknown %d silent-parent\n", helper1, helper1), seals: good, total: -1, want: "witness-malformed"},
		{name: "PID not canonical", file: root + fmt.Sprintf("created 0%d %s %d x\n", helper1, ident(2), inner), seals: good, total: -1, want: "witness-malformed"},
		{name: "old four-field record", file: fmt.Sprintf("launched %d %s sleep\n", helper1, ident(2)), seals: good, total: -1, want: "witness-malformed"},
		{name: "helper record written by another", file: root + created + fmt.Sprintf("launched %d %s %d x\n", helper1, ident(2), inner), seals: good, total: -1, want: "witness-malformed"},
		{name: "ran without its creation record", file: root + launched, seals: []string{sealOf(1, 0)}, total: -1, want: "witness-receipt"},
		{name: "identity differs from the creation record", file: root + created + fmt.Sprintf("launched %d %s %d silent-parent\n", helper1, ident(9), helper1), seals: good, total: -1, want: "witness-receipt"},
		{name: "creator without a record", file: root + created2, seals: []string{sealOf(1, 0)}, total: -1, want: "witness-creator"},
		{name: "root of another process", file: strings.Replace(healthy, "root 100 "+ident(1), "root 100 "+ident(8), 1), seals: good, total: -1, want: "witness-root"},
		{name: "seal reports a failure", file: healthy, seals: []string{sealOf(2, 1)}, total: -1, want: "witness-seal"},
		{name: "a record of the inner run lost", file: root + launched2, seals: good, total: -1, want: "witness-receipt"},
		{name: "seal counts a lost record", file: root + created + launched, seals: []string{sealOf(3, 0)}, total: -1, want: "witness-seal"},
		{name: "no seal", file: healthy, total: -1, want: "witness-seal"},
		{name: "two seals", file: healthy, seals: []string{sealOf(2, 0), sealOf(2, 0)}, total: -1, want: "witness-seal"},
		{name: "raised", file: healthy, seals: good, raised: true, total: -1, want: "witness-raised"},
		{name: "Job saw a process without a record", file: healthy, seals: good, total: 4, want: "witness-count"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, probs := judgeWitness(evidence{file: []byte(c.file), readErr: c.err, seals: c.seals,
				inner: inner, start: ident(1), raised: c.raised, total: c.total})
			switch {
			case c.want == "" && len(probs) > 0:
				t.Errorf("rejected: %v", probs)
			case c.want != "" && !hasClass(probs, c.want):
				t.Errorf("want a %s problem, got %v", c.want, probs)
			}
		})
	}
}

func hasClass(probs []string, class string) bool {
	for _, p := range probs {
		if strings.HasPrefix(p, class+":") {
			return true
		}
	}
	return false
}

// TestGateJudge: the gate check's verdict from its dimensions, on fixed
// rows (Review 078 B1-R2-W-R). The real paths are in TestGateCombos; these
// rows also hold states the real paths do not reach easily (a remediation
// that was not run, an insurance contradiction).
func TestGateJudge(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	leak := gateCase{id: "leak", reject: "left", also: []string{"unaccounted"}}
	clean := gateCase{id: "clean"}
	rows := []struct {
		name     string
		c        gateCase
		problems []string
		remedy   string
		final    string
		want     bool
	}{
		{"expected, remedied, ended", leak, []string{"left: 1", "unaccounted: 1"}, "succeeded", "ended", true},
		{"expected, remediation failed", leak, []string{"left: 1", "still-alive: 1"}, "failed", "alive", false},
		{"expected, remediation not run", leak, []string{"left: 1"}, "not-run", "alive", false},
		{"expected, end unknown", leak, []string{"left: 1", "still-alive: unconfirmed"}, "failed", "unknown", false},
		{"expected, census failed", leak, []string{"left: 1", "census: x"}, "succeeded", "unknown", false},
		{"expected, injection not seen", leak, []string{"left: 1", "hit: never"}, "succeeded", "ended", false},
		{"expected, insurance contradicts", leak, []string{"left: 1", "insurance: found 1"}, "succeeded", "ended", false},
		{"expected, another class", leak, []string{"left: 1", "witness-malformed: x"}, "succeeded", "ended", false},
		{"expected missing", leak, []string{"unaccounted: 1"}, "not-needed", "ended", false},
		{"expected, but the final state alone unknown", leak, []string{"left: 1"}, "succeeded", "unknown", false},
		{"clean", clean, nil, "not-needed", "ended", true},
		{"clean with a tool error", clean, []string{"state-unknown: x"}, "not-needed", "unknown", false},
		{"clean with an evidence problem", clean, []string{"left: 1"}, "succeeded", "ended", false},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			g := gateResult{Problems: r.problems, Remedy: r.remedy, Final: r.final}
			judge(r.c, &g, true)
			if g.Pass != r.want {
				t.Errorf("pass %v, want %v (expected %v, unexpected %v, tool %v)", g.Pass, r.want, g.Expected, g.Unexpected, g.ToolErrors)
			}
		})
	}
}

// TestRecordAlive: a recorded identity that is this process's own says it
// runs; another identity of the same PID says it ended, which is reliable
// only because parseWitness admits identities of the platform's form
// alone ("unknown" never reaches recordAlive).
func TestRecordAlive(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	id, err := selfIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if a, err := recordAlive(witnessRec{pid: os.Getpid(), start: id}); err != nil || !a {
		t.Errorf("own identity: alive %v, %v", a, err)
	}
	if a, err := recordAlive(witnessRec{pid: os.Getpid(), start: ident(7)}); err != nil || a {
		t.Errorf("another identity: alive %v, %v", a, err)
	}
	line := fmt.Sprintf("launched %d unknown %d sleep\n", os.Getpid(), os.Getpid())
	if _, err := parseWitness([]byte(line)); err == nil {
		t.Errorf("an unknown identity was admitted")
	}
}
