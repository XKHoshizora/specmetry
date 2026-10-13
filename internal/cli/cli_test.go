package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

func execute(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = Execute(context.Background(), args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func openStore(t *testing.T, dir datadir.Dir) {
	t.Helper()
	db, _, err := sqlite.Open(context.Background(), dir, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func fileHash(t *testing.T, path string) [32]byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(b)
}

func TestVersion(t *testing.T) {
	testutil.TempHome(t)
	code, out, errOut := execute(t, "version")
	if code != apperr.ExitOK || errOut != "" {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"specmetry ", "database schema: v1", "dashboard:"} {
		if !strings.Contains(out, want) {
			t.Errorf("version output missing %q:\n%s", want, out)
		}
	}
}

func TestUnknownCommandAndFlagAreInvalidInput(t *testing.T) {
	testutil.TempHome(t)
	for _, args := range [][]string{{"no-such-command"}, {"version", "--no-such-flag"}, {"doctor", "extra"}} {
		code, out, errOut := execute(t, args...)
		if code != apperr.ExitInvalidInput {
			t.Errorf("%v: exit %d, want %d", args, code, apperr.ExitInvalidInput)
		}
		if out != "" || !strings.HasPrefix(errOut, "error: ") {
			t.Errorf("%v: stdout %q, stderr %q", args, out, errOut)
		}
	}
}

// doctor is read-only: in an empty home it reports and creates nothing.
func TestDoctorDoesNotCreateDataDirectory(t *testing.T) {
	home := testutil.TempHome(t)
	code, out, errOut := execute(t, "doctor")
	if code != apperr.ExitOK {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	if !strings.Contains(out, "does not exist yet") {
		t.Errorf("output:\n%s", out)
	}
	if _, err := os.Stat(home.Path); !os.IsNotExist(err) {
		t.Fatalf("doctor created %s (stat err %v)", home.Path, err)
	}
}

func TestDoctorHealthyHomeLeavesDatabaseUntouched(t *testing.T) {
	home := testutil.TempHome(t)
	openStore(t, home)
	before := fileHash(t, home.Join(datadir.DatabaseFile))
	code, out, errOut := execute(t, "doctor")
	if code != apperr.ExitOK {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	for _, want := range []string{"[ok]   permissions", "[ok]   database        schema v1", "[ok]   git"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if fileHash(t, home.Join(datadir.DatabaseFile)) != before {
		t.Fatal("doctor modified the database")
	}
}

func TestDoctorReportsNewerSchema(t *testing.T) {
	home := testutil.TempHome(t)
	openStore(t, home)
	raw, err := sql.Open("sqlite", home.Join(datadir.DatabaseFile))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec("UPDATE meta SET schema_version = 99 WHERE id = 1"); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()
	code, out, _ := execute(t, "doctor")
	if code != apperr.ExitEnvironment || !strings.Contains(out, "schema v99 is newer") {
		t.Fatalf("exit %d\n%s", code, out)
	}
}

func TestDoctorReportsInvalidConfigAndStaleLock(t *testing.T) {
	home := testutil.TempHome(t)
	if err := home.Ensure(); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		datadir.ConfigFile:      "[telemetry]\nenabled = true\n",
		datadir.MigrateLockFile: "",
	} {
		f, err := datadir.CreatePrivateFile(home.Join(name), true)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.WriteString(content)
		_ = f.Close()
	}
	code, out, errOut := execute(t, "doctor")
	if code != apperr.ExitEnvironment {
		t.Fatalf("exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "[fail] configuration") || !strings.Contains(out, "[fail] migration lock") {
		t.Errorf("output:\n%s", out)
	}
	if !strings.Contains(errOut, "doctor found 2 problem(s)") {
		t.Errorf("stderr %q", errOut)
	}
}

// MVP-PRV-05 / MVP-PRV-06: doctor reports a data directory whose
// permissions were loosened.
func TestDoctorReportsLoosenedPermissions(t *testing.T) {
	home := testutil.TempHome(t)
	openStore(t, home)
	testutil.LoosenPermissions(t, home.Path)
	code, out, _ := execute(t, "doctor")
	if code != apperr.ExitEnvironment || !strings.Contains(out, "[fail] permissions") {
		t.Fatalf("exit %d\n%s", code, out)
	}
}
