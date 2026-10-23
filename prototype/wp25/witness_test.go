package wp25

// Instrumentation of the gate regressions (Review 076): what the outer
// gate check knows about an inner run without trusting the inner run's own
// account. Every helper writes one line to the witness file the outer
// check gives it when it starts, with its PID and the start time it reads
// for itself; an injected fault that reaches its site leaves a line too.
// Scenarios never read the witness file, and nothing here decides a
// scenario's verdict.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const witnessEnv = "WP25_WITNESS"

type witnessLine struct {
	kind   string // "launched" or "inject"
	pid    int
	start  string // the process's own start time, as procStart reads it
	detail string // the helper mode, or the fault
}

// witness appends one line about this process to the witness file, if the
// outer gate check gave one.
func witness(kind, detail string) {
	path := os.Getenv(witnessEnv)
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	line := fmt.Sprintf("%s %d %s %s\n", kind, os.Getpid(), selfStart(), detail)
	// Faults of the witness itself (Review 077 B1-R2-W), in helpers only.
	switch {
	case inHelper() && injected("witness-write-fail"):
		injectHit("witness", os.Getpid())
		_ = f.Close() // the write below then really fails
	case inHelper() && injected("witness-short-write"):
		injectHit("witness", os.Getpid())
		line = line[:len(line)/2] // only part of the record is written
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

func inHelper() bool { return os.Getenv(helperEnv) != "" }

// injectLogEnv names the file where helpers record that an injected fault
// reached its site; the inner test process prints a WP25INJECT line
// instead. Neither is the witness file, so a fault of the witness itself
// is still seen to have acted.
const injectLogEnv = "WP25_INJECT_LOG"

// injectHit records that an injected fault reached its site.
func injectHit(site string, pid int) {
	b, _ := json.Marshal(map[string]any{"fault": os.Getenv(injectEnv), "site": site, "pid": pid})
	if !inHelper() {
		fmt.Println("WP25INJECT " + string(b))
		return
	}
	if path := os.Getenv(injectLogEnv); path != "" {
		if f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			_, _ = f.Write(append(b, '\n'))
			_ = f.Close()
		}
	}
}

var (
	onceMu   sync.Mutex
	onceDone = map[string]bool{}
)

// injectOnce reports whether the named fault is injected and has not acted
// yet in this process; the call that says so records the hit.
func injectOnce(name, site string, pid int) bool {
	if !injected(name) {
		return false
	}
	onceMu.Lock()
	defer onceMu.Unlock()
	if onceDone[name] {
		return false
	}
	onceDone[name] = true
	injectHit(site, pid)
	return true
}

// tamperWitness damages the witness file at the end of an inner run, as a
// failure of the file itself would (Review 077 B1-R2-W): the end of the
// last record lost, a record twice, a record of an unknown kind, the file
// gone.
func tamperWitness() {
	path := os.Getenv(witnessEnv)
	if path == "" {
		return
	}
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		return
	}
	lines := strings.SplitAfter(strings.TrimSuffix(string(b), "\n"), "\n")
	last := strings.TrimSuffix(lines[len(lines)-1], "\n") + "\n"
	switch {
	case injected("witness-truncated"):
		injectHit("witness-file", os.Getpid())
		_ = os.Truncate(path, int64(len(b)-1))
	case injected("witness-duplicate"):
		injectHit("witness-file", os.Getpid())
		appendFile(path, last)
	case injected("witness-unknown-kind"):
		injectHit("witness-file", os.Getpid())
		appendFile(path, "bogus"+last[strings.IndexByte(last, ' '):])
	case injected("witness-deleted"):
		injectHit("witness-file", os.Getpid())
		_ = os.Remove(path)
	}
}

func appendFile(path, s string) {
	if f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
		_, _ = f.WriteString(s)
		_ = f.Close()
	}
}

func readWitness(path string) ([]witnessLine, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []witnessLine
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) != 4 {
			return nil, fmt.Errorf("malformed witness line %q", clip(sc.Text()))
		}
		pid, err := strconv.Atoi(fs[1])
		if err != nil || pid <= 0 {
			return nil, fmt.Errorf("malformed witness PID %q", clip(sc.Text()))
		}
		out = append(out, witnessLine{kind: fs[0], pid: pid, start: fs[2], detail: fs[3]})
	}
	return out, sc.Err()
}

// witnessEnded waits up to d for every launched helper to have ended, by
// the identity it witnessed. It returns the ones still alive, and an error
// if a state cannot be read.
func witnessEnded(ws []witnessLine, d time.Duration) ([]witnessLine, error) {
	deadline := time.Now().Add(d)
	for {
		var alive []witnessLine
		for _, w := range ws {
			if w.kind != "launched" {
				continue
			}
			a, err := witnessAlive(w)
			if err != nil {
				return nil, err
			}
			if a {
				alive = append(alive, w)
			}
		}
		if len(alive) == 0 || time.Now().After(deadline) {
			return alive, nil
		}
		time.Sleep(20 * time.Millisecond)
	}
}
