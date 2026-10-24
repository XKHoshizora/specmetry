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
	// exitWitness: under a gate check, the helper's records could not be
	// made (witness_test.go); it did nothing else.
	exitWitness = 94
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
	if mode == "holder" && injectedAny("limit-race-epipe", "limit-race-ok") {
		// Review 086 B-N4-R/T: the holder's real timer, early.
		limit = 100 * time.Millisecond
	}
	time.AfterFunc(limit, func() {
		limitHeld(mode)
		_ = report("limit")
		os.Exit(limitExit)
	})
	// Faults of the gate regressions (Review 077 B1-R2-W): a helper that
	// ends before it leaves its own record.
	if (injected("helper-exit-at-start") && mode == "silent-parent") || (injected("unrecorded-ended") && mode == "linger") {
		injectHit("runHelper", os.Getpid())
		return 0
	}
	// Under a gate check, nothing before the helper's own record, which
	// needs its creation record first (witness_test.go).
	if witnessOn() {
		if receipt(mode, !(injected("leak-unreceipted") && mode == "linger")) != nil {
			return exitWitness
		}
	}
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
		// the first write error (the read end was closed) and exits 3. A
		// record it cannot write is its own failure (Review 076 N4): it
		// exits exitReport, and its scenario finds it ended without the
		// record.
		ignoreSIGPIPE()
		if report("pid") != nil {
			return exitReport
		}
		if injected("ok-unasked") {
			// Review 084 B-N4: an answer nobody asked for, before any write;
			// the holder's writes then fail and it reports that too.
			injectHit("holder.unasked", os.Getpid())
			if report("ok") != nil {
				return exitReport
			}
		}
		// f1-mixed-fault (Review 082 N4-R): stderr is written only once the
		// scenario has counted what waits at the stop; the round that does
		// writes stdout after that count.
		mixed := injected("f1-mixed-fault")
		answered := false
		for i := 0; ; i++ {
			// Asked by its scenario (Reviews 082 and 083 N4-R), the holder
			// ends its observation: it writes nothing more, closes its output
			// first, so nothing it could do afterwards belongs to the
			// observation, and only then gives its one final answer, "ok":
			// none of its writes failed. An output it cannot close, or an
			// answer it cannot write, leaves the scenario without one. It
			// then waits to be ended.
			if !answered && fileThere(rep+".query") {
				answered = true
				switch {
				case injected("ok-open-output"):
					// Review 083 N4-R-S: an answer that is not the end: the
					// holder answers and stops writing, its output still open.
					injectHit("holder.answer-not-final", os.Getpid())
					if report("ok") == nil {
						_ = complete()
					}
					select {}
				case injectedAny("fail-after-ok", "ok-epipe-reported", "ok-fails-epipe-reported"):
					// Review 083 N4-R-S, the review's case: an answer in the
					// middle; then the holder's stderr breaks, its next write
					// to it fails, and its report of that fails too. Review
					// 084 B-N4: the same, but the report of that failure is
					// written.
					injectHit("holder.after-answer", os.Getpid())
					_ = report("ok")
					_ = os.Stderr.Close()
				default:
					oerr, eerr := os.Stdout.Close(), os.Stderr.Close()
					if injected("output-close-fails") {
						// The close of its output fails (a second close).
						injectHit("holder.close-fails", os.Getpid())
						eerr = os.Stderr.Close()
					}
					if oerr == nil && eerr == nil {
						if injectedAny("epipe-then-ok", "epipe-fails-then-ok") {
							// Review 084 B-N4, the other order: a write after
							// the end fails and is reported, and the holder
							// answers all the same.
							injectHit("holder.write-after-end", os.Getpid())
							if _, err := fmt.Fprintln(os.Stdout, "after the end"); err != nil {
								_ = report("epipe")
							}
						}
						err := report("ok")
						if err == nil {
							err = afterRecord(rep)
						}
						if err == nil {
							_ = complete()
						}
						if injectedAny("ok-close-fails", "ok-limit-unwritable") {
							// Tells the scenario the failure happened (the
							// test's own marker, not a report record).
							_ = os.WriteFile(rep+".failed", nil, 0o600)
						}
					}
					select {}
				}
			}
			both := !mixed && !injected("f1-one-kept") || mixed && fileThere(rep+".stopped")
			_, err := fmt.Fprintf(os.Stdout, "late-out %d\n", i)
			if err == nil && both {
				_, err = fmt.Fprintf(os.Stderr, "late-err %d\n", i)
			}
			if err != nil {
				if injected("epipe-report-late") {
					// The record comes only after the scenario's first wait for
					// it, while it waits for an answer (bounded).
					injectHit("holder.before-late-report", os.Getpid())
					time.Sleep(3500 * time.Millisecond)
				}
				rerr := report("epipe")
				if rerr == nil {
					rerr = afterRecord(rep)
				}
				if rerr == nil {
					rerr = complete()
				}
				if injectedAny("fail-after-ok", "ok-epipe-reported", "epipe-close-fails", "epipe-limit-unwritable") {
					// Tells the scenario the failure happened, and its report
					// was tried (the test's own marker, not a report record).
					_ = os.WriteFile(rep+".failed", nil, 0o600)
				}
				if rerr != nil {
					if injected("epipe-report-fail-alive") || mixed || injected("fail-after-ok") {
						// Review 081 N4-R: the record could not be written, and
						// the holder is still alive for a while (bounded), past
						// the scenario's decision; the scenario then ends it.
						injectHit("holder.after-report-failure", os.Getpid())
						time.Sleep(10 * time.Second)
					}
					return exitReport
				}
				return 3
			}
			time.Sleep(20 * time.Millisecond)
		}
	case "holder-parent":
		// Starts a holder that inherits its output, writes, waits for its
		// scenario's release, and exits 3.
		c := helper("holder", nil)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := start(c); err != nil {
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
		if err := start(c); err != nil {
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
			if err := start(c); err != nil {
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

// complete is a holder's last step (Review 085 B-N4-R), taken only once
// every record it had to write was written and closed and no tool failure
// came: it commits its completion, "done <pid>", written and closed in a
// file of its own and then renamed into place, so the scenario sees all
// of it or nothing. A failure on the way leaves no completion. Nothing the
// holder does afterwards belongs to the observation.
func complete() error {
	rep := os.Getenv(reportEnv)
	if rep == "" {
		return nil
	}
	if injectedAny("limit-race-epipe", "limit-race-ok") {
		// Review 086 B-N4-R/T: the completion comes only once the hard
		// limit's callback has entered (bounded).
		awaitFile(rep+".limit-entered", 3*time.Second)
	}
	pid := os.Getpid()
	if injected("done-foreign") {
		// A completion that names another process than its writer.
		injectHit("holder.done-foreign", pid)
		if pid = os.Getppid(); pid <= 0 {
			pid = 1
		}
	}
	f, err := os.OpenFile(rep+".done-part", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	_, werr := f.WriteString("done " + strconv.Itoa(pid) + "\n")
	cerr := f.Close()
	if injected("done-close-fails") {
		// The close of the completion fails (a second close).
		injectHit("holder.done-close-fails", os.Getpid())
		cerr = f.Close()
	}
	if werr != nil {
		return werr
	}
	if cerr != nil {
		return cerr
	}
	if injected("done-rename-fails") {
		// The rename fails: its target is a directory.
		injectHit("holder.done-rename-fails", os.Getpid())
		_ = os.Mkdir(rep+".done", 0o700)
	}
	return os.Rename(rep+".done-part", rep+".done")
}

// limitHeld is the gate regressions' hold on the hard limit's callback
// (Review 086 B-N4-R/T; nothing in a normal run): once it has entered, it
// leaves a marker and waits at a gate, released by its file or after
// 10 s, so a holder's completion can come while it runs.
func limitHeld(mode string) {
	if mode != "holder" || !injectedAny("limit-race-epipe", "limit-race-ok") {
		return
	}
	rep := os.Getenv(reportEnv)
	injectHit("helper.limit-entered", os.Getpid())
	_ = os.WriteFile(rep+".limit-entered", nil, 0o600)
	awaitFile(rep+".limit-release", 10*time.Second)
}

// awaitFile waits, bounded, for a file to exist.
func awaitFile(path string, d time.Duration) {
	deadline := time.Now().Add(d)
	for !fileThere(path) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
}

// afterRecord is what a holder does after its record under the gate
// regressions' faults (none in a normal run): a tool failure there is an
// error. The holder reaches its hard limit beside its record and reports
// it; with ...-unwritable that report fails (Review 085 B-N4-R); with
// ...-gated it comes only after 10 s (bounded), once the scenario has
// ended the holder, so the failure has happened but its report is cut off.
func afterRecord(rep string) error {
	switch {
	case injectedAny("epipe-limit", "ok-limit", "epipe-limit-unwritable", "ok-limit-unwritable"):
		injectHit("holder.limit-after-record", os.Getpid())
		if err := report("limit"); err != nil {
			return err
		}
	case injectedAny("epipe-limit-gated", "ok-limit-gated"):
		injectHit("holder.limit-gated", os.Getpid())
		_ = os.WriteFile(rep+".failed", nil, 0o600) // the failure point is reached
		time.Sleep(10 * time.Second)
		_ = report("limit")
	default:
		return nil
	}
	return fmt.Errorf("the holder reached its hard limit")
}

func fileThere(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
	flag := os.O_WRONLY | os.O_APPEND
	if (injected("epipe-report-fail") || injected("epipe-report-fail-alive") || injected("f1-mixed-fault") || injected("fail-after-ok")) && kind == "epipe" || injectedAny("ok-report-fail", "ok-fails-epipe-reported") && kind == "ok" ||
		injected("epipe-fails-then-ok") && kind == "epipe" ||
		injectedAny("epipe-limit-unwritable", "ok-limit-unwritable") && kind == "limit" {
		// Review 076 N4: a holder's record cannot be written. The report is
		// opened read-only, so the real write fails.
		injectHit("report", os.Getpid())
		flag = os.O_RDONLY
	}
	f, err := os.OpenFile(path, flag, 0o600)
	if err != nil {
		return err
	}
	pid := os.Getpid()
	if injected("epipe-foreign") && kind == "epipe" || injected("ok-foreign") && kind == "ok" {
		// Review 084: a record that names another process than its writer.
		injectHit("report.foreign", pid)
		if pid = os.Getppid(); pid <= 0 {
			pid = 1
		}
	}
	line := kind + " " + strconv.Itoa(pid)
	// Faults of the gate regressions (gate_test.go); none is set in a
	// normal run.
	payload := line + "\n"
	okFault := kind == "ok" && (injected("ok-dup") || injected("ok-truncated") || injected("ok-malformed"))
	if okFault {
		injectHit("report.ok-fault", os.Getpid())
	}
	switch {
	case okFault && injected("ok-dup"):
		payload = line + "\n" + line + "\n"
	case okFault && injected("ok-truncated"):
		payload = line
	case okFault && injected("ok-malformed"):
		payload = "ok not-a-pid\n"
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
	if injected("epipe-cut") && kind == "epipe" {
		// Review 085 B-N4-R: the holder is in the middle of its record when
		// the scenario ends it: the record's first part, then (bounded) the
		// rest.
		injectHit("report.cut", os.Getpid())
		_, _ = f.WriteString(line)
		time.Sleep(10 * time.Second)
		payload = "\n"
	}
	_, werr := f.WriteString(payload)
	cerr := f.Close()
	if injected("epipe-close-fails") && kind == "epipe" || injected("ok-close-fails") && kind == "ok" {
		// Review 085 B-N4-R: the record is written whole, then the close of
		// the report fails (a second close).
		injectHit("report.close-fails", os.Getpid())
		cerr = f.Close()
	}
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
	if err := start(cmd); err != nil {
		s.invalidf("helper did not start: %v", err)
	}
	s.launched(1)
	s.own(ownChild(cmd))
}
