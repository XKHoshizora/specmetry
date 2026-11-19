// Package pseudo is the pseudo-tool of LC-9 and of the control protocol's
// tests (implementation plan §12.12; LE-5 ①, review-resolution §200): a
// re-execution of the test binary that a variable selects. Every test
// binary that may start one calls MaybeRun first in its TestMain, before it
// takes any deadline or parses any flag; MaybeRun then acts as the selected
// mode and exits. It never starts a process itself, so it has no
// descendants.
package pseudo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

// The variables that select the mode and its size.
const (
	EnvMode  = "C9EVIDENCE_PSEUDO"
	EnvBytes = "C9EVIDENCE_PSEUDO_BYTES"
)

// The modes.
const (
	// EchoArgs writes every argument's length and SHA-256 to stdout, one
	// fixed line to stderr, and exits 0.
	EchoArgs = "echo-args"
	// Silent writes nothing and exits 0.
	Silent = "silent"
	// Flood writes EnvBytes bytes to stdout, ignoring write errors, and
	// exits 0.
	Flood = "flood"
	// Block reads stdin until EOF, writes nothing, and exits 0.
	Block = "block"
	// Hold is the controlled output holder H (§12.12 "输出不关闭的拒绝路
	// 径"): it never writes to stdout (the target pipe's write end it
	// inherited), writes one line "ready" to stderr, reads stdin until EOF
	// and exits 0.
	Hold = "hold"
	// FloodTerm writes EnvBytes bytes to stdout in one write and then ends
	// itself by SIGTERM (Unix; elsewhere it exits 98). It writes nothing
	// after, so no write of it can meet a closed read end (Review 155
	// B-154-4-R-S: another signal than the expected SIGPIPE).
	FloodTerm = "flood-term"
	// FloodExit writes EnvBytes bytes to stdout in one write and then exits
	// with FloodExitCode, a normal non-zero exit.
	FloodExit = "flood-exit"
	// Term ends itself by SIGTERM at once (Unix; elsewhere it exits 98).
	Term = "term"
)

// FloodExitCode is FloodExit's exit code: the code a shell reports for a
// SIGPIPE end (128 + 13), here a normal exit.
const FloodExitCode = 141

// StderrLine is what EchoArgs writes to stderr.
const StderrLine = "c9-pseudo stderr\n"

// ReadyLine is what Hold writes to stderr.
const ReadyLine = "ready\n"

// MaybeRun runs the pseudo-tool when EnvMode is set, and exits; otherwise
// it returns at once.
func MaybeRun() {
	mode, ok := os.LookupEnv(EnvMode)
	if !ok {
		return
	}
	os.Exit(run(mode, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// EchoOutput is EchoArgs' exact standard output for the arguments.
func EchoOutput(args []string) string {
	s := fmt.Sprintf("c9-pseudo argv %d\n", len(args))
	for i, a := range args {
		sum := sha256.Sum256([]byte(a))
		s += fmt.Sprintf("arg %d %d %s\n", i+1, len(a), hex.EncodeToString(sum[:]))
	}
	return s
}

func run(mode string, args []string, in io.Reader, out, errw io.Writer) int {
	switch mode {
	case EchoArgs:
		if _, err := io.WriteString(out, EchoOutput(args)); err != nil {
			return 3
		}
		if _, err := io.WriteString(errw, StderrLine); err != nil {
			return 3
		}
		return 0
	case Silent:
		return 0
	case Flood:
		n, err := strconv.Atoi(os.Getenv(EnvBytes))
		if err != nil || n < 0 {
			return 4
		}
		buf := make([]byte, 32<<10)
		for i := range buf {
			buf[i] = 'x'
		}
		for n > 0 {
			k := min(n, len(buf))
			_, _ = out.Write(buf[:k]) // the writes after the overflow may fail; ignored
			n -= k
		}
		return 0
	case FloodTerm, FloodExit:
		n, err := strconv.Atoi(os.Getenv(EnvBytes))
		if err != nil || n < 0 {
			return 4
		}
		buf := make([]byte, n)
		for i := range buf {
			buf[i] = 'x'
		}
		if _, err := out.Write(buf); err != nil {
			return 3
		}
		if mode == FloodExit {
			return FloodExitCode
		}
		return endByTerm()
	case Term:
		return endByTerm()
	case Block:
		_, _ = io.Copy(io.Discard, in)
		return 0
	case Hold:
		if _, err := io.WriteString(errw, ReadyLine); err != nil {
			return 3
		}
		_, _ = io.Copy(io.Discard, in)
		return 0
	}
	fmt.Fprintf(errw, "c9-pseudo: unknown mode %q\n", mode)
	return 97
}

// endByTerm sends SIGTERM to the pseudo-tool itself; the Go runtime then
// ends the process by the signal's default action (no handler is
// installed for it). Until that happens it only sleeps: on Unix it never
// returns. Elsewhere there is no signal and it returns 98.
func endByTerm() int {
	if err := raiseTerm(); err != nil {
		return 98
	}
	for {
		time.Sleep(time.Second)
	}
}
