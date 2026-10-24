//go:build linux

package wp25

import "golang.org/x/sys/unix"

// fionread asks how many bytes wait in a pipe (FIONREAD).
const fionread = unix.TIOCINQ
