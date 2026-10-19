//go:build !windows

package gitx

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// fileKey is a file's identity: its device and inode, read once, not
// looked up again by path when compared.
type fileKey string

func statKey(st *unix.Stat_t) fileKey { return fileKey(fmt.Sprintf("%x:%x", st.Dev, st.Ino)) }

// keyOf returns the identity of the open file f, and whether it is a
// directory.
func keyOf(f *os.File) (fileKey, bool, error) {
	var st unix.Stat_t
	if err := unix.Fstat(int(f.Fd()), &st); err != nil {
		return "", false, err
	}
	return statKey(&st), st.Mode&unix.S_IFMT == unix.S_IFDIR, nil
}

// lstatKey returns the identity of what is at path, not following a link
// there, and whether it is a directory (a link is not).
func lstatKey(path string) (fileKey, bool, error) {
	var st unix.Stat_t
	if err := unix.Lstat(path, &st); err != nil {
		return "", false, err
	}
	return statKey(&st), st.Mode&unix.S_IFMT == unix.S_IFDIR, nil
}

// removeEntry removes the entry name of the directory open as parent if it
// is still the file key: a directory when dir is set, removed only if
// empty (rmdir: a file or a link at that name is never removed in its
// place), otherwise a file. The entry is checked, then removed, both
// relative to parent: a process of the same user that puts something else
// at the name between the two is not detected; for a directory, only an
// empty one can be removed that way.
func removeEntry(parent *os.File, _ string, name string, key fileKey, dir bool) error {
	fd := int(parent.Fd())
	var st unix.Stat_t
	if err := unix.Fstatat(fd, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	if statKey(&st) != key || (st.Mode&unix.S_IFMT == unix.S_IFDIR) != dir {
		return errReplaced
	}
	flags := 0
	if dir {
		flags = unix.AT_REMOVEDIR
	}
	return unix.Unlinkat(fd, name, flags)
}
