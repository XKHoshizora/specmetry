package wp25

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// runHelper runs this binary as one of the scenarios' helper processes.
// Every helper has a hard limit: it reports "limit" and exits limitExit,
// which makes the scenario invalid.
func runHelper(mode string, args []string) int {
	limit := helperLimit
	if ms, err := strconv.Atoi(os.Getenv(limitEnv)); err == nil && ms > 0 {
		limit = time.Duration(ms) * time.Millisecond
	}
	time.AfterFunc(limit, func() {
		report("limit")
		os.Exit(limitExit)
	})
	arg := func(i int) int {
		if i >= len(args) {
			return 0
		}
		n, _ := strconv.Atoi(args[i])
		return n
	}
	switch mode {
	case "write":
		// write <stdout bytes> <stderr bytes> <exit code>
		writePattern(os.Stdout, 'o', int64(arg(0)))
		writePattern(os.Stderr, 'e', int64(arg(1)))
		return arg(2)
	case "exit-after":
		// exit-after <ms> <code>
		time.Sleep(time.Duration(arg(0)) * time.Millisecond)
		return arg(1)
	case "sleep":
		report("pid")
		select {}
	case "marker":
		// The user code of the Windows failure chain: it leaves a marker,
		// so a process that never ran leaves none.
		_ = os.WriteFile(os.Getenv(markerEnv), []byte("ran\n"), 0o600)
		report("pid")
		select {}
	case "holder":
		// A descendant that holds the output and keeps writing; it reports
		// the first write error (the read end was closed) and exits.
		ignoreSIGPIPE()
		report("pid")
		for i := 0; ; i++ {
			if _, err := fmt.Fprintf(os.Stdout, "late-out %d\n", i); err != nil {
				report("epipe")
				return 3
			}
			if _, err := fmt.Fprintf(os.Stderr, "late-err %d\n", i); err != nil {
				report("epipe")
				return 3
			}
			time.Sleep(20 * time.Millisecond)
		}
	case "holder-parent":
		// Starts a holder that inherits its output, writes, and exits 3.
		c := helper("holder", nil)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Start(); err != nil {
			return 90
		}
		waitReport(os.Getenv(reportEnv), "pid", 1, 5*time.Second)
		fmt.Fprintln(os.Stdout, "early-out")
		fmt.Fprintln(os.Stderr, "early-err")
		return 3
	case "silent-parent":
		// Starts a descendant that holds the output without writing.
		c := helper("sleep", nil)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Start(); err != nil {
			return 90
		}
		if injected("parent-fail") {
			return 5 // the helper fails after starting its child
		}
		waitReport(os.Getenv(reportEnv), "pid", 1, 5*time.Second)
		return 0
	case "fanout":
		// fanout <n>: n sleeping children, then sleep.
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
				return 90
			}
		}
		waitReport(os.Getenv(reportEnv), "pid", n, 10*time.Second)
		report("fanout")
		select {}
	}
	if code, ok := runHelperPlatform(mode, args); ok {
		return code
	}
	fmt.Fprintln(os.Stderr, "unknown helper mode")
	return 99
}

// report appends one line to the scenario's report file in one write.
func report(kind string, rest ...string) {
	path := os.Getenv(reportEnv)
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return
	}
	line := kind + " " + strconv.Itoa(os.Getpid())
	for _, r := range rest {
		line += " " + r
	}
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
	case injected("report-truncated"):
		payload = line
	case injected("report-overlong"):
		payload = strings.Repeat("x", 70000) + "\n" + line + "\n"
	}
	_, _ = f.WriteString(payload)
	_ = f.Close()
	if injected("report-delete") && kind == "pid" {
		_ = os.Remove(path)
	}
}

// patternByte is the i-th byte a "write" helper sends on a stream.
func patternByte(seed byte, i int64) byte {
	if i%64 == 63 {
		return '\n'
	}
	return 'a' + byte((i/64+int64(seed))%26)
}

func writePattern(f *os.File, seed byte, n int64) {
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
			return
		}
		off += k
	}
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

// startHelper starts a helper with a null stdin and the given output, and
// makes the scenario own it.
func startHelper(s *scen, cmd *exec.Cmd) {
	if err := cmd.Start(); err != nil {
		s.invalidf("helper did not start: %v", err)
	}
	s.launched(1)
	s.own(ownChild(cmd))
}
