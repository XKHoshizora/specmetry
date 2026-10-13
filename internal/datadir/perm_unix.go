//go:build !windows

package datadir

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

func secureDir(path string) error { return os.Chmod(path, 0o700) }

func securePrivatePath(path string, isDir bool) error {
	if isDir {
		return os.Chmod(path, 0o700)
	}
	return os.Chmod(path, 0o600)
}

func checkPermissions(root string) ([]string, error) {
	var problems []string
	uid := os.Getuid()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		perm := info.Mode().Perm()
		switch {
		case d.IsDir() && perm&0o077 != 0:
			problems = append(problems, fmt.Sprintf("%s: mode %04o, want 0700", p, perm))
		case info.Mode().IsRegular() && perm&0o077 != 0:
			problems = append(problems, fmt.Sprintf("%s: mode %04o, want 0600", p, perm))
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok && int(st.Uid) != uid {
			problems = append(problems, fmt.Sprintf("%s: owned by uid %d, not the current user (uid %d)", p, st.Uid, uid))
		}
		return nil
	})
	return problems, err
}
