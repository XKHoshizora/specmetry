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
)

// The workflow passes R and the record directory explicitly (plan §4.2:
// R is never inferred from the test's working directory).
const (
	EnvR       = "C9EVIDENCE_R"
	EnvRecords = "C9EVIDENCE_RECORDS"
)

// FormalConfig is the formal run's configuration. In phase 1 the invoker
// is Stopped: E-10 (ii) is not confirmed, so no process is started and
// every call fails (J-1). The guard reads the real toolchain without running
// it; the controls are all of LC-2 to LC-9 for this GOOS.
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
	return Config{
		GOOS: runtime.GOOS, R: os.Getenv(EnvR), RecordsDir: rec, Clock: clock, Start: start,
		WriteFS: chain.OSFS{}, ReadFS: modsnap.OS{}, Stderr: os.Stderr,
		Invoker:    collect.Stopped{Clock: clock},
		Guard:      func() collect.GuardValues { return collect.ReadGuard(collect.RealGuardSources()) },
		TypeCheck:  collect.TypeCheck{},
		Controls:   controls.All(runtime.GOOS),
		MainModule: spec.MainModule, Exceptions: spec.RepoEmbedExceptions, Candidates: spec.CandidatesFor,
		Watch: spec.WatchPackages, Identity: ids, T2: spec.T2,
	}, nil
}
