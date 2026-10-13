//go:build windows

package datadir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// MVP-PRV-06: the data directory has a protected DACL granting access only to
// the current user and SYSTEM (TD-22), and files inherit it.
func TestWindowsDACL(t *testing.T) {
	d := Dir{Path: filepath.Join(t.TempDir(), "home")}
	if err := d.Ensure(); err != nil {
		t.Fatal(err)
	}
	f, err := CreatePrivateFile(d.Join(DatabaseFile), true)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	sd, err := windows.GetNamedSecurityInfo(d.Path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := sd.Control()
	if err != nil {
		t.Fatal(err)
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("data directory DACL is not protected")
	}
	user, system, err := allowedSIDs()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{d.Path, d.Join(DatabaseFile)} {
		sids := allowedACESIDs(t, p)
		if len(sids) == 0 {
			t.Fatalf("%s: no allow entries", p)
		}
		var sawUser, sawSystem bool
		for _, s := range sids {
			switch {
			case s.Equals(user):
				sawUser = true
			case s.Equals(system):
				sawSystem = true
			default:
				t.Fatalf("%s: unexpected allow entry for %s", p, s)
			}
		}
		if !sawUser || !sawSystem {
			t.Fatalf("%s: user=%v system=%v, want both", p, sawUser, sawSystem)
		}
	}
	if problems, err := Check(d); err != nil || len(problems) != 0 {
		t.Fatalf("Check() = %v, %v", problems, err)
	}
}

// MVP-PRV-06: doctor (via Check) detects a loosened DACL.
func TestWindowsCheckReportsLoosenedDACL(t *testing.T) {
	d := Dir{Path: filepath.Join(t.TempDir(), "home")}
	if err := d.Ensure(); err != nil {
		t.Fatal(err)
	}
	sd, err := windows.GetNamedSecurityInfo(d.Path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
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
	loose, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.GENERIC_READ,
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
	if err := windows.SetNamedSecurityInfo(d.Path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.UNPROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, loose, nil); err != nil {
		t.Fatal(err)
	}
	problems, err := Check(d)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(problems, "\n")
	if !strings.Contains(joined, everyone.String()) {
		t.Fatalf("Everyone entry not reported: %v", problems)
	}
	if !strings.Contains(joined, "protected DACL") {
		t.Fatalf("unprotected DACL not reported: %v", problems)
	}
	_ = os.RemoveAll(d.Path)
}

func allowedACESIDs(t *testing.T, path string) []*windows.SID {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	var out []*windows.SID
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			t.Fatal(err)
		}
		if ace.Header.AceType == windows.ACCESS_ALLOWED_ACE_TYPE {
			sid, err := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).Copy()
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, sid)
		}
	}
	return out
}
