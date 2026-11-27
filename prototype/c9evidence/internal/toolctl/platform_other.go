//go:build !linux && !windows

package toolctl

import "errors"

// platformMode elsewhere (darwin): no termination request; at W the call
// fails and the cleanup is unknown (§12.14 item 8; os/wait_unimp.go).
const platformMode = ModeNone

var errNoPidfd = errors.New("toolctl: no pidfd on this platform")

func pidfdOpen(int) (int, error) { return -1, errNoPidfd }

func pidfdSendKill(int) error { return errNoPidfd }

func closeFD(int) error { return errNoPidfd }

func sigchldIgnored() bool { return false }
