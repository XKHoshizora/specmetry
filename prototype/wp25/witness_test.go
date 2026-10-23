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
	_, _ = fmt.Fprintf(f, "%s %d %s %s\n", kind, os.Getpid(), selfStart(), detail)
	_ = f.Close()
}

// injectHit records, from the inner test process, that an injected fault
// reached its site.
func injectHit(site string, pid int) {
	b, _ := json.Marshal(map[string]any{"fault": os.Getenv(injectEnv), "site": site, "pid": pid})
	fmt.Println("WP25INJECT " + string(b))
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
