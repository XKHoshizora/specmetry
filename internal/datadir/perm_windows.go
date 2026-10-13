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

const systemNote = " and SYSTEM"

// checkPermissions requires, for the data directory and everything in it:
//   - allow entries only for the current user and SYSTEM (deny entries only
//     restrict access and are accepted);
//   - an owner that is the current user, SYSTEM or the Administrators group
//     (an owner can always rewrite the DACL; administrators can take
//     ownership of anything anyway, and elevated processes create files
//     owned by that group);
//   - on the data directory itself, a protected DACL, so that nothing is
//     inherited from the parent;
//   - on every directory, an inheritable entry for the current user, so that
//     new files (the WAL, the -shm, backups) inherit the private DACL rather
//     than getting the creating process's default DACL.
func checkPermissions(root string) ([]string, error) {
	user, system, err := allowedSIDs()
	if err != nil {
		return nil, err
	}
	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, err
	}
	var problems []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if problem := entryProblem(p, d); problem != "" {
			problems = append(problems, problem)
			return nil
		}
		sd, err := windows.GetNamedSecurityInfo(p, windows.SE_FILE_OBJECT,
			windows.DACL_SECURITY_INFORMATION|windows.OWNER_SECURITY_INFORMATION)
		if err != nil {
			return err
		}
		owner, _, err := sd.Owner()
		if err != nil {
			return err
		}
		if owner == nil || (!owner.Equals(user) && !owner.Equals(system) && !owner.Equals(admins)) {
			problems = append(problems, fmt.Sprintf("%s: owned by %s; want the current user", p, describeSID(owner)))
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
		userInherits := false
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
				switch {
				case sid.Equals(user):
					const inherit = windows.OBJECT_INHERIT_ACE | windows.CONTAINER_INHERIT_ACE
					if ace.Header.AceFlags&inherit == inherit {
						userInherits = true
					}
				case sid.Equals(system):
				default:
					problems = append(problems, fmt.Sprintf("%s: grants access to %s; want only the current user and SYSTEM", p, describeSID(sid)))
				}
			default:
				problems = append(problems, fmt.Sprintf("%s: unexpected ACE type %d", p, ace.Header.AceType))
			}
		}
		if d.IsDir() && !userInherits {
			problems = append(problems, p+": does not pass its permissions on to new files (no inheritable entry for the current user)")
		}
		return nil
	})
	return problems, err
}

func describeSID(sid *windows.SID) string {
	if sid == nil {
		return "nobody"
	}
	account, domain, _, err := sid.LookupAccount("")
	if err != nil || account == "" {
		return sid.String()
	}
	if domain != "" {
		return domain + `\` + account + " (" + sid.String() + ")"
	}
	return account + " (" + sid.String() + ")"
}
