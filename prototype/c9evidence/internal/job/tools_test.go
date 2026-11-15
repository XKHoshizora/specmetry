package job

import (
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
)

// stubTools stands for the tool registry: the closeout's consumption of
// its problems (§12.14 item 13) is checked here; the real registry is
// exercised by LC-9 and, on fixtures, by the real-call tests.
type stubTools struct{ fails, pend []string }

func (stubTools) Terminal() (bool, <-chan struct{}) { return true, nil }

func (s stubTools) Problems() ([]string, []string) { return s.fails, s.pend }

func TestJobToolProblems(t *testing.T) {
	cases := []struct {
		name  string
		tools stubTools
		want  chain.Conclusion
	}{
		{"clean", stubTools{}, chain.Consistent},
		{"failure", stubTools{fails: []string{"call 1 call1-env linux/amd64: cleanup unknown: x"}}, chain.Incomplete},
		{"pending", stubTools{pend: []string{"call 2 call3-list linux/amd64: not terminal: request running"}}, chain.Incomplete},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, errOut := newConfig(t, controlsWithout(t, "LC-9"))
			cfg.Tools = c.tools
			f := run(t, cfg, true, 0)
			if f.Conclusion != c.want {
				t.Fatalf("final %v %v", f.Conclusion, f.Reasons)
			}
			// The diagnostic task writes the provisional conclusion's
			// failure and pending lines.
			if c.want == chain.Incomplete && !strings.Contains(errOut.String(), "R tool control: ") {
				t.Errorf("no tool control line in the diagnostics:\n%s", errOut.String())
			}
		})
	}
}
