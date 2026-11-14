package controls

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// FixtureRepo prepares the LC-5 baseline (R_f with its dependency Q) under
// root for job-level tests, and returns R_f, an in-process invoker that
// answers every configuration of the GOOS with LC5-0's healthy output, and
// the guard values a seam would hand over. The invoker publishes its calls on
// clock (the job's clock; nil means the real one). It starts no process.
func FixtureRepo(root, goos string, clock collect.Clock) (string, collect.Invoker, func() collect.GuardValues, error) {
	files := map[string]string{}
	for k, v := range rfFiles() {
		files["rf/"+k] = v
	}
	for k, v := range qFiles {
		files["modcache/example.com/q@v1.0.0/"+k] = v
	}
	if err := writeFiles(root, files); err != nil {
		return "", nil, nil, err
	}
	base := rfSetup{root: root, rf: filepath.Join(root, "rf"), q: filepath.Join(root, "modcache", "example.com", "q@v1.0.0"), goroot: filepath.Join(root, "goroot")}
	cfgs := spec.ConfigsFor(goos)
	if len(cfgs) == 0 {
		return "", nil, nil, errors.New("unknown GOOS " + goos)
	}
	base.cfg = cfgs[0]
	if clock == nil {
		clock = realClock{}
	}
	inv := perConfig{base: base, clock: clock}
	return base.rf, inv, func() collect.GuardValues { return base.guardValues(spec.ToolVersion) }, nil
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// perConfig answers each call with LC5-0's output for the call's
// configuration.
type perConfig struct {
	base  rfSetup
	clock collect.Clock
}

func (p perConfig) Invoke(c collect.CallSpec) collect.CallResult {
	s := p.base
	s.cfg = c.Config
	f := &collect.Fake{Clock: p.clock, Responses: s.responses(lc5Case{name: "job"})}
	return f.Invoke(c)
}
