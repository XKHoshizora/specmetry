//go:build darwin

package wp25

// Direct checks of the macOS census when a process's environment cannot be
// read (Review 083 N-MAC). The census must fail, not skip the process, and
// its failure must say what is known of that process without its
// arguments or environment: its identity, its state, whether it is
// exiting, its parent and group, and whether it runs this test's program.
// The process is this test's own helper; the failure is injected after the
// real read of its state; nothing else is read or touched.

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// unreadableCensus runs the census over the test's own helper only, its
// environment read failing with EINVAL, and its state read through recheck.
func unreadableCensus(t *testing.T, recheck func(int) (pinfo, presence)) (pid int, token string, err error) {
	t.Helper()
	token = "WP25_CENSUS_CHECK=" + strconv.Itoa(os.Getpid())
	cmd := helper("linger", []string{token, limitEnv + "=10000"})
	if err := start(cmd); err != nil {
		t.Fatalf("the helper did not start: %v", err)
	}
	pid = cmd.Process.Pid
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		done := make(chan struct{})
		go func() { _ = cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Errorf("the helper %d did not end within 5 s", pid)
		}
	})
	k, kerr := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if kerr != nil {
		t.Fatalf("the helper's state: %v", kerr)
	}
	since := fmt.Sprintf("%d.%06d", k.Proc.P_starttime.Sec, k.Proc.P_starttime.Usec)
	list, args, again := procList, procArgs, censusRecheck
	t.Cleanup(func() { procList, procArgs, censusRecheck = list, args, again })
	procList = func() ([]unix.KinfoProc, error) { return []unix.KinfoProc{*k}, nil }
	procArgs = func(p int) ([]byte, error) {
		if p == pid {
			return nil, unix.EINVAL
		}
		return nil, fmt.Errorf("the check reads only its own helper, not %d", p)
	}
	censusRecheck = recheck
	_, err = censusTag(token, since)
	return pid, token, err
}

func TestCensusUnreadable(t *testing.T) {
	if os.Getenv(innerEnv) != "" {
		t.Skip("inner run")
	}
	t.Run("state-read", func(t *testing.T) {
		pid, token, err := unreadableCensus(t, procInfo)
		if err == nil {
			t.Fatal("the census did not fail on an environment it could not read")
		}
		if !errors.Is(err, errCensus) || !errors.Is(err, unix.EINVAL) {
			t.Errorf("the failure %q is not a census error carrying EINVAL", err)
		}
		msg := err.Error()
		for _, want := range []string{"process " + strconv.Itoa(pid), "start ", "state ", "exiting ", "parent ", "group ", "same program as this test: yes"} {
			if !strings.Contains(msg, want) {
				t.Errorf("the failure does not say %q: %s", want, msg)
			}
		}
		if strings.Contains(msg, token) || strings.Contains(msg, "WP25_") {
			t.Errorf("the failure shows the process's environment: %s", msg)
		}
	})
	t.Run("state-unreadable", func(t *testing.T) {
		// The read of the state after the failure fails too: the census
		// still fails, and says the state is unknown.
		_, _, err := unreadableCensus(t, func(int) (pinfo, presence) { return pinfo{}, unknown })
		if err == nil {
			t.Fatal("the census did not fail when neither the environment nor the state could be read")
		}
		if !strings.Contains(err.Error(), "state after the failure: unreadable") {
			t.Errorf("the failure does not say the state could not be read: %s", err)
		}
	})
}
