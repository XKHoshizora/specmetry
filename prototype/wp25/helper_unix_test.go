//go:build linux || darwin

package wp25

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// runHelperPlatform runs the Unix-only helper modes. The parent helpers
// wait for their children's records and then for their scenario's release
// (awaitRelease), so their children are still their children when the
// scenario verifies them.
func runHelperPlatform(mode string, args []string) (int, bool) {
	arg := func(i int) int {
		if i >= len(args) {
			return 0
		}
		n, _ := strconv.Atoi(args[i])
		return n
	}
	rep := os.Getenv(reportEnv)
	switch mode {
	case "member-parent":
		// One more member in the same group, then exit: the leader is
		// gone while the group still has a member.
		c := helper("sleep", nil)
		if err := c.Start(); err != nil {
			return exitStart, true
		}
		if _, err := awaitRecords(rep, "pid", 1, 5*time.Second); err != nil {
			return exitWait, true
		}
		awaitRelease()
		return 0, true
	case "escape-parent":
		// A descendant that leaves the group (setsid), then exit.
		c := helper("sleep", nil)
		c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := c.Start(); err != nil {
			return exitStart, true
		}
		if _, err := awaitRecords(rep, "pid", 1, 5*time.Second); err != nil {
			return exitWait, true
		}
		awaitRelease()
		return 0, true
	case "chain-main":
		// chain-main <depth>: a chain of depth processes in a new session,
		// each the parent of the next; exit once all have reported and the
		// scenario has released it.
		depth := arg(0)
		if injected("chain-short") {
			depth--
			witness("inject", "chain-short")
		}
		c := helper("chain", nil, strconv.Itoa(depth))
		c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := c.Start(); err != nil {
			return exitStart, true
		}
		if _, err := awaitRecords(rep, "pid", arg(0), 10*time.Second); err != nil {
			return exitWait, true
		}
		awaitRelease()
		return 0, true
	case "chain":
		if report("pid") != nil {
			return exitReport, true
		}
		if d := arg(0); d > 1 {
			c := helper("chain", nil, strconv.Itoa(d-1))
			if err := c.Start(); err != nil {
				return exitStart, true
			}
		}
		select {}
	case "sigchld-probe":
		// sigchld-probe ignore|inherit: does a child that exits stay
		// unreaped until this process waits for it?
		if len(args) > 0 && args[0] == "ignore" {
			signal.Ignore(syscall.SIGCHLD)
		}
		ign := signal.Ignored(syscall.SIGCHLD)
		c := helper("exit-after", nil, "100", "0")
		if err := c.Start(); err != nil {
			return exitStart, true
		}
		time.Sleep(600 * time.Millisecond)
		_, err := waitidNoWait(c.Process.Pid)
		held := err == nil
		_ = c.Wait()
		fmt.Printf("probe ignored=%v unreaped=%v kernel=%s err=%v\n", ign, held, sigchldDisposition(), err)
		return 0, true
	}
	return 0, false
}
