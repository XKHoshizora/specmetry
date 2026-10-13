//go:build windows

package datadir

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// secureDir replaces the directory's DACL with a protected one (inheritance
// from the parent disabled) granting full control to the current user and
// SYSTEM only; children inherit it (TD-22).
func secureDir(path string) error {
	user, system, err := allowedSIDs()
	if err != nil {
		return err
	}
	entries := []windows.EXPLICIT_ACCESS{
		grant(user, windows.TRUSTEE_IS_USER),
		grant(system, windows.TRUSTEE_IS_WELL_KNOWN_GROUP),
	}
	acl, err := windows.ACLFromEntries(entries, nil)
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, acl, nil)
}

// securePrivatePath relies on the DACL inherited from the data directory.
func securePrivatePath(string, bool) error { return nil }

func grant(sid *windows.SID, kind windows.TRUSTEE_TYPE) windows.EXPLICIT_ACCESS {
	return windows.EXPLICIT_ACCESS{
		AccessPermissions: windows.GENERIC_ALL,
		AccessMode:        windows.SET_ACCESS,
		Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  kind,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
}

func allowedSIDs() (user, system *windows.SID, err error) {
	tu, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, nil, err
	}
	user, err = tu.User.Sid.Copy()
	if err != nil {
		return nil, nil, err
	}
	system, err = windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	return user, system, err
}

func checkPermissions(root string) ([]string, error) {
	user, system, err := allowedSIDs()
	if err != nil {
		return nil, err
	}
	var problems []string
	err = filepath.WalkDir(root, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		sd, err := windows.GetNamedSecurityInfo(p, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			return err
		}
		if p == root {
			control, _, err := sd.Control()
			if err != nil {
				return err
			}
			if control&windows.SE_DACL_PROTECTED == 0 {
				problems = append(problems, p+": DACL inherits from the parent directory; want a protected DACL")
			}
		}
		dacl, _, err := sd.DACL()
		if err != nil {
			return err
		}
		if dacl == nil {
			problems = append(problems, p+": no DACL (everyone has full access)")
			return nil
		}
		for i := uint32(0); i < uint32(dacl.AceCount); i++ {
			var ace *windows.ACCESS_ALLOWED_ACE
			if err := windows.GetAce(dacl, i, &ace); err != nil {
				return err
			}
			switch ace.Header.AceType {
			case windows.ACCESS_DENIED_ACE_TYPE:
				// Deny entries only restrict access.
			case windows.ACCESS_ALLOWED_ACE_TYPE:
				// The SID is stored inline, starting at SidStart.
				sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)) //nolint:gosec // documented ACE layout
				if !sid.Equals(user) && !sid.Equals(system) {
					problems = append(problems, fmt.Sprintf("%s: grants access to %s; want only the current user and SYSTEM", p, describeSID(sid)))
				}
			default:
				problems = append(problems, fmt.Sprintf("%s: unexpected ACE type %d", p, ace.Header.AceType))
			}
		}
		return nil
	})
	return problems, err
}

func describeSID(sid *windows.SID) string {
	account, domain, _, err := sid.LookupAccount("")
	if err != nil || account == "" {
		return sid.String()
	}
	if domain != "" {
		return domain + `\` + account + " (" + sid.String() + ")"
	}
	return account + " (" + sid.String() + ")"
}
