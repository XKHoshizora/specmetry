//go:build linux || darwin

package wp25

// The precondition of leader retention (T2, Review 073 §4): the kernel
// keeps an exited child unreaped only if SIGCHLD is not ignored. Each probe
// runs in its own helper process, so this test process is not changed.

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func probe(s *scen, viaShell bool, arg string) string {
	var out bytes.Buffer
	cmd := helper("sigchld-probe", nil, arg)
	if viaShell {
		// The shell sets SIGCHLD to SIG_IGN, then execs the helper, which
		// inherits the ignored disposition.
		self := cmd.Path
		args := append([]string{"-c", `trap '' CHLD; exec "$0" "$@"`, self}, cmd.Args[1:]...)
		cmd.Path, cmd.Args = "/bin/sh", append([]string{"/bin/sh"}, args...)
	}
	cmd.Stdout = &out
	startHelper(s, cmd)
	if !childOf(cmd).reap(10 * time.Second) {
		s.invalidf("probe did not end")
	}
	if cmd.ProcessState.ExitCode() != 0 {
		s.invalidf("probe exited %d", cmd.ProcessState.ExitCode())
	}
	line := strings.TrimSpace(out.String())
	if !strings.HasPrefix(line, "probe ") {
		s.invalidf("probe gave no result")
	}
	s.worked(1)
	return line
}

func TestSigchldIgnored(t *testing.T) {
	// Counterexample: a process that ignores SIGCHLD has its children
	// reaped by the kernel, so the leader cannot be kept unreaped. The
	// guard sees it through signal.Ignored.
	s := begin(t, "T2", "sigchld.ignored", counter, true, true)
	defer s.done()
	line := probe(s, false, "ignore")
	s.expectTrue("signal.Ignored reports it", strings.Contains(line, "ignored=true"), line)
	s.expectTrue("the child was reaped by the kernel", strings.Contains(line, "unreaped=false"), line)
}

func TestSigchldInherited(t *testing.T) {
	// An ignored SIGCHLD inherited across exec: the Go runtime installs
	// its own handler at start (only SIGHUP and SIGINT keep an inherited
	// SIG_IGN), so the child stays unreaped and signal.Ignored is false.
	s := begin(t, "T2", "sigchld.inherited", normal, true, true)
	defer s.done()
	line := probe(s, true, "inherit")
	s.expectTrue("signal.Ignored is false", strings.Contains(line, "ignored=false"), line)
	s.expectTrue("the child stayed unreaped", strings.Contains(line, "unreaped=true"), line)
}

func TestSigchldDefault(t *testing.T) {
	s := begin(t, "T2", "sigchld.default", normal, true, true)
	defer s.done()
	line := probe(s, false, "default")
	s.expectTrue("signal.Ignored is false", strings.Contains(line, "ignored=false"), line)
	s.expectTrue("the child stayed unreaped", strings.Contains(line, "unreaped=true"), line)
}
