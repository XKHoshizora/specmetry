//go:build !windows

package checkrun

import (
	"context"
	"io/fs"
	"path/filepath"
)

// finalPath is p with every symbolic link resolved.
func finalPath(p string) (string, error) { return filepath.EvalSymlinks(p) }

// executable reports whether a regular file may be run by anyone.
func executable(fi fs.FileInfo) bool { return fi.Mode().Perm()&0o111 != 0 }

// find is the Unix table of spec §5.3: sh is /bin/sh, a support policy and
// not POSIX's rule, never PATH's; bash and pwsh are the first acceptable
// on PATH; cmd and powershell are not available.
func (f *finder) find(_ context.Context, shell string) (Interpreter, bool, error) {
	var p string
	var ok bool
	switch shell {
	case "sh":
		p, ok = f.accept("/bin/sh")
	case "bash", "pwsh":
		p, ok = f.onPath(shell)
	}
	return Interpreter{Shell: shell, Path: p}, ok, nil
}
