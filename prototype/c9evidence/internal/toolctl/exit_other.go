//go:build !unix

package toolctl

import "os"

// exitOf copies a ProcessState. Without signals (Windows) every end is an
// exit with its code.
func exitOf(ps *os.ProcessState) Exit {
	return Exit{Code: ps.ExitCode(), Exited: ps.Exited(), String: ps.String()}
}
