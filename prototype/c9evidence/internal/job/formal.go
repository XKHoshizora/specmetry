package job

import (
	"errors"
	"os"
	"runtime"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/controls"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/toolctl"
)

// The workflow passes R and the record directory explicitly (plan §4.2:
// R is never inferred from the test's working directory).
const (
	EnvR       = "C9EVIDENCE_R"
	EnvRecords = "C9EVIDENCE_RECORDS"
)

// FormalConfig is the formal run's configuration. Phase 1 had the Stopped
// invoker (E-10 (ii) not confirmed). Phase 2 (review-resolution §200) wires
// the real caller, the control protocol's registry with the job's W and D,
// and hands it to the closeout as well; the guard reads the real toolchain
// without running it; the controls are all of LC-2 to LC-9 for this GOOS.
// Phase 2 only builds and vets this wiring: the formal run (R, the
// evidence workflow and its run branch) is not authorized and never runs.
func FormalConfig(start time.Time) (Config, error) {
	rec := os.Getenv(EnvRecords)
	if rec == "" {
		return Config{}, errors.New(EnvRecords + " is not set")
	}
	ids := map[string]string{}
	for _, k := range []string{"GITHUB_SHA", "GITHUB_REF", "GITHUB_WORKFLOW_REF", "GITHUB_WORKFLOW_SHA", "RUNNER_OS", "RUNNER_ARCH"} {
		ids[k] = os.Getenv(k)
	}
	clock := chain.RealClock{}
	w, d := chain.Deadlines(start)
	tools := toolctl.New(toolctl.Options{Clock: clock, W: w, D: d})
	return Config{
		GOOS: runtime.GOOS, R: os.Getenv(EnvR), RecordsDir: rec, Clock: clock, Start: start,
		WriteFS: chain.OSFS{}, ReadFS: modsnap.OS{}, Stderr: os.Stderr,
		Invoker:    tools,
		Tools:      tools,
		Guard:      func() collect.GuardValues { return collect.ReadGuard(collect.RealGuardSources()) },
		TypeCheck:  collect.TypeCheck{},
		Controls:   controls.All(runtime.GOOS),
		MainModule: spec.MainModule, Exceptions: spec.RepoEmbedExceptions, Candidates: spec.CandidatesFor,
		Watch: spec.WatchPackages, Identity: ids, T2: spec.T2,
	}, nil
}
