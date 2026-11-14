//go:build c9evidence_run

package run

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/job"
)

var current *job.Job

// TestMain takes the single deadline D at its start, registers and starts
// the writer before m.Run, and exits only with the closeout's exit status
// (plan §5.4, §6.4, §7.1).
func TestMain(m *testing.M) {
	start := time.Now()
	cfg, err := job.FormalConfig(start)
	if err == nil {
		current, err = job.Begin(cfg)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "c9evidence: not started: "+err.Error())
		os.Exit(2)
	}
	code := m.Run()
	os.Exit(current.Finish(code).ExitCode)
}

// TestEvidence runs the F domain and the R domain. It reports nothing
// through testing.T: the conclusion comes from the published state only.
func TestEvidence(*testing.T) { current.RunDomains() }
