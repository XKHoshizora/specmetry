// Package testutil is the shared test harness (implementation-plan WP0.5):
// an isolated data directory, git fixture repositories, golden files,
// pseudo-terminals, canary search and helper subprocesses.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

// TempHome points SPECMETRY_HOME at a fresh, not yet created directory
// inside the test's temp dir and returns it.
func TempHome(t testing.TB) datadir.Dir {
	t.Helper()
	p := filepath.Join(testdir.New(t), "specmetry-home")
	t.Setenv(datadir.EnvHome, p)
	return datadir.Dir{Path: p, FromEnv: true}
}

const helperEnv = "SPECMETRY_TEST_HELPER"

// DispatchHelper must be called first in a package's TestMain. When the test
// binary was started by HelperCommand it runs the named helper and exits.
func DispatchHelper(helpers map[string]func() int) {
	name := os.Getenv(helperEnv)
	if name == "" {
		return
	}
	fn, ok := helpers[name]
	if !ok {
		_, _ = os.Stderr.WriteString("unknown test helper " + name + "\n")
		os.Exit(2)
	}
	os.Exit(fn())
}

// HelperCommand returns a command that re-executes the current test binary
// as the named helper process, with extra environment variables.
func HelperCommand(t testing.TB, name string, env ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(append(os.Environ(), helperEnv+"="+name), env...)
	return cmd
}
