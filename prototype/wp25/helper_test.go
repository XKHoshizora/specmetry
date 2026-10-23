package wp25

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Exit codes a helper uses for its own failures, so the scenario that
// checks the helper's status sees that its load was not built.
const (
	exitStart  = 90 // a child could not be started
	exitReport = 91 // a report record could not be written
	exitWait   = 92 // the children's records did not arrive as required
	exitWrite  = 93 // the output could not be written
)

// runHelper runs this binary as one of the scenarios' helper processes.
// Every helper has a hard limit: it reports "limit" and exits limitExit,
// which makes the scenario invalid. The limit acts only once the helper
// runs Go code; the scenarios' own waits are bounded without it.
func runHelper(mode string, args []string) int {
	limit := helperLimit
	if ms, err := strconv.Atoi(os.Getenv(limitEnv)); err == nil && ms > 0 {
		limit = time.Duration(ms) * time.Millisecond
	}
	time.AfterFunc(limit, func() {
		_ = report("limit")
		os.Exit(limitExit)
	})
	// Faults of the gate regressions (Review 077 B1-R2-W): a helper that
	// ends before it leaves its own record, or one that lives on without it.
	if (injected("helper-exit-at-start") && mode == "silent-parent") || (injected("unrecorded-ended") && mode == "linger") {
		injectHit("runHelper", os.Getpid())
		return 0
	}
	if injected("leak-unreceipted") && mode == "linger" {
		injectHit("runHelper", os.Getpid())
		select {}
	}
	witness("launched", mode)
	arg := func(i int) int {
		if i >= len(args) {
			return 0
		}
		n, _ := strconv.Atoi(args[i])
		return n
	}
	rep := os.Getenv(reportEnv)
	switch mode {
	case "write":
		// write <stdout bytes> <stderr bytes> <exit code>
		if !writePattern(os.Stdout, 'o', int64(arg(0))) || !writePattern(os.Stderr, 'e', int64(arg(1))) {
			return exitWrite
		}
		return arg(2)
	case "exit-after":
		// exit-after <ms> <code>
		time.Sleep(time.Duration(arg(0)) * time.Millisecond)
		return arg(1)
	case "sleep":
		if report("pid") != nil {
			return exitReport
		}
		select {}
	case "linger":
		// A process a gate regression's inner run leaves behind on purpose
		// (TestInnerLeak), ended by its own short limit at the latest.
		select {}
	case "marker":
		// The user code of the Windows failure chain: it leaves a marker,
		// so a process that never ran leaves none.
		_ = os.WriteFile(os.Getenv(markerEnv), []byte("ran\n"), 0o600)
		if report("pid") != nil {
			return exitReport
		}
		select {}
	case "holder":
		// A descendant that holds the output and keeps writing; it reports
		// the first write error (the read end was closed) and exits.
		ignoreSIGPIPE()
		if report("pid") != nil {
			return exitReport
		}
		for i := 0; ; i++ {
			if _, err := fmt.Fprintf(os.Stdout, "late-out %d\n", i); err != nil {
				_ = report("epipe")
				return 3
			}
			if _, err := fmt.Fprintf(os.Stderr, "late-err %d\n", i); err != nil {
				_ = report("epipe")
				return 3
			}
			time.Sleep(20 * time.Millisecond)
		}
	case "holder-parent":
		// Starts a holder that inherits its output, writes, waits for its
		// scenario's release, and exits 3.
		c := helper("holder", nil)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Start(); err != nil {
			return exitStart
		}
		if _, err := awaitRecords(rep, "pid", 1, 5*time.Second); err != nil {
			return exitWait
		}
		fmt.Fprintln(os.Stdout, "early-out")
		fmt.Fprintln(os.Stderr, "early-err")
		awaitRelease()
		return 3
	case "silent-parent":
		// Starts a descendant that holds the output without writing; exits
		// 0 once its scenario has verified the descendant and released it.
		c := helper("sleep", nil)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Start(); err != nil {
			return exitStart
		}
		if injected("parent-fail") {
			return 5 // the helper fails after starting its child
		}
		if _, err := awaitRecords(rep, "pid", 1, 5*time.Second); err != nil {
			return exitWait
		}
		awaitRelease()
		return 0
	case "fanout":
		// fanout <n>: n sleeping children, then a fanout record, then sleep.
		n := arg(0)
		switch {
		case injected("fanout-zero"):
			n = 0
		case injected("fanout-partial"):
			n /= 3
		}
		for i := 0; i < n; i++ {
			c := helper("sleep", nil)
			if err := c.Start(); err != nil {
				return exitStart
			}
		}
		if _, err := awaitRecords(rep, "pid", n, 10*time.Second); err != nil {
			return exitWait
		}
		if report("fanout") != nil {
			return exitReport
		}
		select {}
	}
	if code, ok := runHelperPlatform(mode, args); ok {
		return code
	}
	fmt.Fprintln(os.Stderr, "unknown helper mode")
	return 99
}

// awaitRelease blocks until the scenario closes the helper's stdin: it
// keeps a parent helper alive while its scenario verifies the descendants
// it reported, so they are still descendants when they are checked.
func awaitRelease() { _, _ = io.Copy(io.Discard, os.Stdin) }

// gated gives cmd a stdin the scenario closes to release it; the release
// also runs when the scenario ends.
func gated(s *scen, cmd *exec.Cmd) func() {
	w, err := cmd.StdinPipe()
	if err != nil {
		s.invalidf("stdin: %v", err)
	}
	release := func() { _ = w.Close() }
	s.defer_(release)
	return release
}

// report appends one record to the scenario's report file in one write
// (see report_test.go). Without a report file there is nothing to write.
func report(kind string) error {
	path := os.Getenv(reportEnv)
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	line := kind + " " + strconv.Itoa(os.Getpid())
	// Faults of the gate regressions (gate_test.go); none is set in a
	// normal run.
	payload := line + "\n"
	switch {
	case injected("report-skip") && kind == "pid":
		payload = ""
	case injected("report-malformed"):
		payload = line + "\nlimit not-a-pid\n" // Review 075's counterexample A
	case injected("report-unknown"):
		payload = line + "\nbogus " + strconv.Itoa(os.Getpid()) + "\n"
	case injected("report-dup"):
		payload = line + "\n" + line + "\n"
	case injected("report-extra") && kind == "pid":
		payload = line + "\nfanout " + strconv.Itoa(os.Getpid()) + "\n"
	case injected("report-truncated"):
		payload = line
	case injected("report-overlong"):
		payload = strings.Repeat("x", 70000) + "\n" + line + "\n"
	}
	_, werr := f.WriteString(payload)
	cerr := f.Close()
	if injected("report-delete") && kind == "pid" {
		_ = os.Remove(path)
	}
	if werr != nil {
		return werr
	}
	return cerr
}

// patternByte is the i-th byte a "write" helper sends on a stream.
func patternByte(seed byte, i int64) byte {
	if i%64 == 63 {
		return '\n'
	}
	return 'a' + byte((i/64+int64(seed))%26)
}

func writePattern(f *os.File, seed byte, n int64) bool {
	buf := make([]byte, 4096)
	for off := int64(0); off < n; {
		k := int64(len(buf))
		if n-off < k {
			k = n - off
		}
		for i := int64(0); i < k; i++ {
			buf[i] = patternByte(seed, off+i)
		}
		if _, err := f.Write(buf[:k]); err != nil {
			return false
		}
		off += k
	}
	return true
}

// isPattern reports whether b is exactly the first len(b) bytes a "write"
// helper sends with the seed.
func isPattern(b []byte, seed byte) bool {
	for i, c := range b {
		if c != patternByte(seed, int64(i)) {
			return false
		}
	}
	return true
}

// startHelper starts a helper with a null stdin (unless gated) and the
// given output, and makes the scenario own it.
func startHelper(s *scen, cmd *exec.Cmd) {
	if err := cmd.Start(); err != nil {
		s.invalidf("helper did not start: %v", err)
	}
	s.launched(1)
	s.own(ownChild(cmd))
}
