package datadir

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

func requireRefusal(t *testing.T, d Dir, want string) {
	t.Helper()
	err := d.Ensure()
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error()+detailText(err), want) {
		t.Fatalf("Ensure() = %v (%s), want a refusal mentioning %q", err, detailText(err), want)
	}
	if _, err := os.Lstat(d.Path); !os.IsNotExist(err) {
		t.Fatalf("the refused data directory was created (%v)", err)
	}
}

// grantEveryone adds an inheritable Everyone entry with the given rights to
// the directory's DACL.
func grantEveryone(t *testing.T, path string, rights windows.ACCESS_MASK) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	everyone, err := windows.CreateWellKnownSid(windows.WinWorldSid)
	if err != nil {
		t.Fatal(err)
	}
	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: rights,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_WELL_KNOWN_GROUP,
			TrusteeValue: windows.TrusteeValueFromSID(everyone),
		},
	}}, dacl)
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
}

// codex-review-005 B1: a directory on the path that lets other users delete
// or rename entries (DELETE on it, or FILE_DELETE_CHILD on its parent) lets
// them replace the data directory after the checks. Refused before anything
// is created.
func TestWindowsEnsureRefusesAncestorOthersCanReplace(t *testing.T) {
	for name, rights := range map[string]windows.ACCESS_MASK{
		"full control": windows.GENERIC_ALL,
		"delete":       windows.DELETE,
		"delete child": fileDeleteChild,
	} {
		t.Run(name, func(t *testing.T) {
			shared := filepath.Join(testdir.New(t), "shared")
			if err := os.Mkdir(shared, 0o700); err != nil {
				t.Fatal(err)
			}
			grantEveryone(t, shared, rights)
			requireRefusal(t, Dir{Path: filepath.Join(shared, "home")}, "other users can")
		})
	}
}

// Read access for others on an ancestor does not let them replace anything.
func TestWindowsEnsureAcceptsAncestorReadableByOthers(t *testing.T) {
	shared := filepath.Join(testdir.New(t), "shared")
	if err := os.Mkdir(shared, 0o700); err != nil {
		t.Fatal(err)
	}
	grantEveryone(t, shared, windows.GENERIC_READ|windows.GENERIC_EXECUTE)
	if err := (Dir{Path: filepath.Join(shared, "home")}).Ensure(); err != nil {
		t.Fatalf("Ensure() = %v (%s)", err, detailText(err))
	}
}

// A junction on the path is refused: what it points to can be changed by
// whoever controls it.
func TestWindowsEnsureRefusesJunctionOnThePath(t *testing.T) {
	target := testdir.New(t)
	link := filepath.Join(testdir.New(t), "link")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Fatalf("mklink: %v\n%s", err, out)
	}
	requireRefusal(t, Dir{Path: filepath.Join(link, "home")}, "symbolic link or junction")
}
