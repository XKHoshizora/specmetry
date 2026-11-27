//go:build unix

package pseudo

import (
	"os"
	"syscall"
)

func raiseTerm() error { return syscall.Kill(os.Getpid(), syscall.SIGTERM) }
