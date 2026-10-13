//go:build !windows

package datadir

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// checkAncestors reports directories on the way to path that a user other
// than the current user and root could use to replace what comes after them
// (codex-review-005 B1). Whoever can write to a directory can rename an
// entry away and put a link in its place, so a check of the data directory
// alone would not stay true until the data is written. Such locations are
// refused rather than defended: the MVP does not support them.
//
// Directories are visited the way the kernel resolves the path, following
// symbolic links, so the directories a link leads through are checked too.
// A directory may be writable by others only if it is sticky (like /tmp) and
// the next entry belongs to the current user or root, because then only they
// can rename or remove it. Components that do not exist yet will be created
// by the current user under the last existing directory, which is checked.
// The data directory itself is checked by checkPermissions.
func checkAncestors(path string) ([]string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	uid := os.Getuid()
	cur := "/"
	curInfo, err := os.Lstat(cur)
	if err != nil {
		return nil, err
	}
	var problems, dirs []string
	dirs = append(dirs, cur)
	pending := components(abs)
	for links := 0; len(pending) > 0; {
		name := pending[0]
		pending = pending[1:]
		switch name {
		case ".":
			continue
		case "..":
			cur = filepath.Dir(cur)
			if curInfo, err = os.Lstat(cur); err != nil {
				return nil, err
			}
			continue
		}
		child := filepath.Join(cur, name)
		childInfo, err := os.Lstat(child)
		if errors.Is(err, fs.ErrNotExist) {
			// Created later by the current user, inside cur.
			problems = append(problems, replaceable(cur, curInfo, child, uid, uid)...)
			break
		}
		if err != nil {
			return nil, err
		}
		problems = append(problems, replaceable(cur, curInfo, child, ownerOf(childInfo), uid)...)
		if len(pending) == 0 {
			break // child is the data directory itself
		}
		if childInfo.Mode()&fs.ModeSymlink != 0 {
			if links++; links > 40 {
				return nil, fmt.Errorf("%s: too many symbolic links", abs)
			}
			target, err := os.Readlink(child)
			if err != nil {
				return nil, err
			}
			pending = append(components(target), pending...)
			if filepath.IsAbs(target) {
				cur = "/"
				if curInfo, err = os.Lstat(cur); err != nil {
					return nil, err
				}
			}
			continue
		}
		if !childInfo.IsDir() {
			return append(problems, child+": not a directory"), nil
		}
		cur, curInfo = child, childInfo
		dirs = append(dirs, cur)
	}
	aclProblems, err := extendedACLProblems(dirs)
	return append(problems, aclProblems...), err
}

// replaceable reports why users other than uid and root could replace child
// through its parent directory dir.
func replaceable(dir string, info fs.FileInfo, child string, childOwner, uid int) []string {
	var problems []string
	if owner := ownerOf(info); owner != 0 && owner != uid {
		problems = append(problems, fmt.Sprintf("%s: owned by uid %d, who could replace %s; every directory on the path must belong to you or root", dir, owner, child))
	}
	mode := info.Mode()
	if mode.Perm()&0o022 != 0 {
		sticky := mode&fs.ModeSticky != 0
		if !sticky || (childOwner != uid && childOwner != 0) {
			problems = append(problems, fmt.Sprintf("%s: writable by other users (mode %04o), who could replace %s; choose a location whose directories only you or root can write to", dir, mode.Perm(), child))
		}
	}
	return problems
}

func ownerOf(info fs.FileInfo) int {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return int(st.Uid)
	}
	return -1
}

func components(p string) []string {
	var out []string
	for _, c := range strings.Split(filepath.ToSlash(p), "/") {
		if c != "" {
			out = append(out, c)
		}
	}
	return out
}
