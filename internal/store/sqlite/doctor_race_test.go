package sqlite_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/cli"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

func sum(t *testing.T, path string) [32]byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(b)
}

// codex-review-007 B2 through the public doctor command: the external-close
// race twice in a row must print "version unknown" both times — never
// "[ok] database schema v1" — and leave the main database untouched.
func TestDoctorRepeatedExternalCloseReportsUnknown(t *testing.T) {
	ctx := context.Background()
	home := testutil.TempHome(t)
	db, _, err := sqlite.Open(ctx, home, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	dbFile := home.Join(datadir.DatabaseFile)
	main := sum(t, dbFile)

	for cycle := 0; cycle < 2; cycle++ {
		happened := false
		sqlite.SetReadOnlyHook(func() {
			sqlite.SetReadOnlyHook(nil)
			if out, err := testutil.HelperCommand(t, "plainclose").CombinedOutput(); err != nil {
				t.Errorf("external close: %v\n%s", err, out)
			}
			for _, suffix := range []string{"-wal", "-shm"} {
				if _, err := os.Stat(dbFile + suffix); !os.IsNotExist(err) {
					t.Errorf("the external close left %s (%v)", suffix, err)
				}
			}
			happened = true
		})
		var out, errOut bytes.Buffer
		code := cli.Execute(ctx, []string{"doctor"}, &out, &errOut)
		sqlite.SetReadOnlyHook(nil)
		if code != apperr.ExitOK || !strings.Contains(out.String(), "[warn] database        version unknown") || strings.Contains(out.String(), "schema v") {
			t.Fatalf("cycle %d (outside close during the check: %v): exit %d\n%s%s", cycle, happened, code, out.String(), errOut.String())
		}
		if cycle == 0 && !happened {
			t.Fatal("the first cycle did not reproduce the race")
		}
		if sum(t, dbFile) != main {
			t.Fatalf("cycle %d: the main database file changed", cycle)
		}
		t.Logf("cycle %d: outside close during the check: %v", cycle, happened)
	}
}
