//go:build windows

package toolctl

import "errors"

// platformMode on Windows: one Kill of the published process (its handle
// refers to the same process object until it is released, §12.14 item 8).
const platformMode = ModeKill

var errNoPidfd = errors.New("toolctl: no pidfd on this platform")

func pidfdOpen(int) (int, error) { return -1, errNoPidfd }

func pidfdSendKill(int) error { return errNoPidfd }

func closeFD(int) error { return errNoPidfd }

func sigchldIgnored() bool { return false }
