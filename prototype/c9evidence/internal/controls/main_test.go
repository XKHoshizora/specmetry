package controls

import (
	"os"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/pseudo"
)

// TestMain lets the test binary act as the pseudo-tool of LC-9 when it is
// re-executed with the mode variable set; otherwise the tests run.
func TestMain(m *testing.M) {
	pseudo.MaybeRun()
	os.Exit(m.Run())
}
