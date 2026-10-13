package datadir

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// replaceRights are the rights that let a user rename or replace a directory
// (DELETE, or FILE_DELETE_CHILD on its parent) or take control of it.
const replaceRights = windows.DELETE | windows.WRITE_DAC | windows.WRITE_OWNER | fileDeleteChild | windows.GENERIC_ALL

// checkAncestors reports directories on the way to path that a user other
// than the current user, SYSTEM, Administrators or TrustedInstaller could
// use to replace what comes after them (codex-review-005 B1): a directory
// that grants others DELETE can be renamed away and recreated, and one that
// grants FILE_DELETE_CHILD lets them remove any entry in it. Links and
// junctions on the path are refused, since they can be repointed. Such
// locations are refused rather than defended; the MVP does not support
// them. Directories that do not exist yet will be created by the current
// user. The data directory itself is checked by checkPermissions.
func checkAncestors(path string) ([]string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	vol := filepath.VolumeName(abs)
	if vol == "" {
		return nil, fmt.Errorf("%s: no volume", abs)
	}
	trusted, err := trustedSIDs()
	if err != nil {
		return nil, err
	}
	names := strings.FieldsFunc(abs[len(vol):], func(r rune) bool { return r == '\\' || r == '/' })
	if len(names) == 0 {
		return nil, nil
	}
	cur := vol + `\`
	problems, err := ancestorProblems(cur, trusted)
	if err != nil {
		return nil, err
	}
	for _, name := range names[:len(names)-1] {
		cur = filepath.Join(cur, name)
		info, err := os.Lstat(cur)
		if errors.Is(err, fs.ErrNotExist) {
			break
		}
		if err != nil {
			return nil, err
		}
		if isLink(info.Mode()) {
			return append(problems, cur+": "+linkProblem), nil
		}
		if !info.IsDir() {
			return append(problems, cur+": not a directory"), nil
		}
		p, err := ancestorProblems(cur, trusted)
		if err != nil {
			return nil, err
		}
		problems = append(problems, p...)
	}
	return problems, nil
}

func trustedSIDs() ([]*windows.SID, error) {
	user, system, err := allowedSIDs()
	if err != nil {
		return nil, err
	}
	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, err
	}
	installer, err := windows.StringToSid("S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464")
	if err != nil {
		return nil, err
	}
	return []*windows.SID{user, system, admins, installer}, nil
}

// isTrusted reports whether sid is one of the trusted principals, or an
// app package or capability SID (S-1-15-...). The latter never let another
// user in: an AppContainer process is granted access only if its user
// account is granted access as well.
func isTrusted(sid *windows.SID, trusted []*windows.SID) bool {
	if sid != nil && strings.HasPrefix(sid.String(), "S-1-15-") {
		return true
	}
	for _, t := range trusted {
		if sid != nil && sid.Equals(t) {
			return true
		}
	}
	return false
}

func ancestorProblems(dir string, trusted []*windows.SID) ([]string, error) {
	sd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return nil, err
	}
	var problems []string
	owner, _, err := sd.Owner()
	if err != nil {
		return nil, err
	}
	if !isTrusted(owner, trusted) {
		problems = append(problems, fmt.Sprintf("%s: owned by %s; other users can replace what is inside it", dir, describeSID(owner)))
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return nil, err
	}
	if dacl == nil {
		return append(problems, dir+": no DACL; other users can replace what is inside it"), nil
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return nil, err
		}
		if ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 {
			continue // applies only to entries created later, which are checked themselves
		}
		switch ace.Header.AceType {
		case windows.ACCESS_DENIED_ACE_TYPE:
		case windows.ACCESS_ALLOWED_ACE_TYPE:
			sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)) //nolint:gosec // documented ACE layout
			if !isTrusted(sid, trusted) && ace.Mask&replaceRights != 0 {
				problems = append(problems, fmt.Sprintf("%s: other users can delete or rename entries here (%s may delete or change permissions); choose a location only you can change", dir, describeSID(sid)))
			}
		default:
			problems = append(problems, fmt.Sprintf("%s: unexpected ACE type %d", dir, ace.Header.AceType))
		}
	}
	return problems, nil
}
