//go:build !windows

package datadir

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

const systemNote = ""

// Directories need the execute bit to be entered, hence 0700 rather than
// the 0600 that gosec expects of files.
func secureDir(path string) error { return os.Chmod(path, 0o700) } //nolint:gosec // directory

func securePrivatePath(path string, isDir bool) error {
	if isDir {
		return os.Chmod(path, 0o700) //nolint:gosec // directory
	}
	return os.Chmod(path, 0o600)
}

// checkPermissions requires every entry to be owned by the current user with
// no group or other permission bits. On Linux the mode bits are sufficient:
// a POSIX ACL can only grant access through the group-class bits (its mask),
// so a mode without group and other bits leaves no ACL entry effective. On
// macOS extended ACLs are evaluated before the mode bits and are checked
// separately (extendedACLProblems).
func checkPermissions(root string) ([]string, error) {
	var problems []string
	var paths []string
	uid := os.Getuid()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if vanished(p, root, err) {
				return nil
			}
			return err
		}
		if beforeEntryCheck != nil && p != root {
			beforeEntryCheck(p)
		}
		if problem := entryProblem(p, d); problem != "" {
			problems = append(problems, problem)
			return nil
		}
		info, err := d.Info()
		if err != nil {
			if vanished(p, root, err) {
				return nil
			}
			return err
		}
		paths = append(paths, p)
		perm := info.Mode().Perm()
		switch {
		case d.IsDir() && perm&0o077 != 0:
			problems = append(problems, fmt.Sprintf("%s: mode %04o, want 0700", p, perm))
		case !d.IsDir() && perm&0o077 != 0:
			problems = append(problems, fmt.Sprintf("%s: mode %04o, want 0600", p, perm))
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok && int(st.Uid) != uid {
			problems = append(problems, fmt.Sprintf("%s: owned by uid %d, not the current user (uid %d)", p, st.Uid, uid))
		}
		return nil
	})
	if err != nil {
		return problems, err
	}
	aclProblems, err := extendedACLProblems(paths, func(p string, err error) bool { return vanished(p, root, err) })
	return append(problems, aclProblems...), err
}
