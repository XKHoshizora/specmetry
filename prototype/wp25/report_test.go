package wp25

// The helpers' report protocol (Review 075 B1). A helper appends one
// whole line per event, "<kind> <pid>\n", in a single write to the report
// file its scenario gave it:
//
//	pid     the helper is running (sleepers, holders, chain levels)
//	epipe   a holder's first write failed (the read end was closed)
//	fanout  all children of a fanout helper have reported
//	limit   the helper reached its hard limit
//
// Nothing else is valid: no other kind, no extra field, a positive PID in
// canonical decimal form, no line over maxLine bytes. While the helpers
// may still write, an unterminated last line is a write in progress and
// is left for the next read; once they have ended it is a truncation.
// Every error makes the scenario invalid; nothing is skipped.

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var reportKinds = map[string]bool{"pid": true, "epipe": true, "fanout": true, "limit": true}

// errNotYet is a wait that ended before the records arrived; any other
// error of awaitRecords means the report is not valid.
var errNotYet = errors.New("records not there in time")

const (
	maxReport = 64 << 10
	maxLine   = 64
)

type reportLine struct {
	kind string
	pid  int
}

// reportSpec is what a report must hold once its writers have ended:
// exactly n records of a kind (n >= 0), or at most -n (n < 0) for a record
// that is the result of the mechanism rather than part of the load; each
// (kind, PID) once. A kind it does not name must not appear; "limit"
// never may.
type reportSpec map[string]int

type reportFile struct {
	path string
	spec reportSpec
}

func clip(s string) string {
	if len(s) > 40 {
		return s[:40] + "..."
	}
	return s
}

// parseReport reads a report strictly; final says whether its writers have
// ended.
func parseReport(path string, final bool) ([]reportLine, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	if len(b) > maxReport {
		return nil, fmt.Errorf("report larger than %d bytes", maxReport)
	}
	text := string(b)
	var out []reportLine
	for len(text) > 0 {
		i := strings.IndexByte(text, '\n')
		if i < 0 {
			if final {
				return nil, fmt.Errorf("truncated record %q", clip(text))
			}
			if len(text) > maxLine {
				return nil, fmt.Errorf("record longer than %d bytes", maxLine)
			}
			break
		}
		line := text[:i]
		text = text[i+1:]
		if len(line) > maxLine {
			return nil, fmt.Errorf("record longer than %d bytes", maxLine)
		}
		f := strings.Split(line, " ")
		if len(f) != 2 {
			return nil, fmt.Errorf("malformed record %q", clip(line))
		}
		if !reportKinds[f[0]] {
			return nil, fmt.Errorf("unknown record kind %q", clip(f[0]))
		}
		pid, err := strconv.Atoi(f[1])
		if err != nil || pid <= 0 || strconv.Itoa(pid) != f[1] {
			return nil, fmt.Errorf("malformed PID in %q", clip(line))
		}
		out = append(out, reportLine{kind: f[0], pid: pid})
	}
	return out, nil
}

// tally counts the records by kind and refuses duplicates and limits.
func tally(lines []reportLine) (map[string][]int, error) {
	seen := map[reportLine]bool{}
	by := map[string][]int{}
	for _, l := range lines {
		if l.kind == "limit" {
			return nil, fmt.Errorf("helper %d reached its hard limit", l.pid)
		}
		if seen[l] {
			return nil, fmt.Errorf("duplicate record %s %d", l.kind, l.pid)
		}
		seen[l] = true
		by[l.kind] = append(by[l.kind], l.pid)
	}
	return by, nil
}

// checkFinal checks a report against its spec once its writers have ended.
func checkFinal(rf reportFile) error {
	lines, err := parseReport(rf.path, true)
	if err != nil {
		return err
	}
	by, err := tally(lines)
	if err != nil {
		return err
	}
	kinds := map[string]bool{}
	for k := range by {
		kinds[k] = true
	}
	for k := range rf.spec {
		kinds[k] = true
	}
	var bad []string
	for k := range kinds {
		want, got := rf.spec[k], len(by[k])
		switch {
		case want < 0 && got > -want:
			bad = append(bad, fmt.Sprintf("%d %s records, want at most %d", got, k, -want))
		case want >= 0 && got != want:
			bad = append(bad, fmt.Sprintf("%d %s records, want %d", got, k, want))
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		return errors.New(strings.Join(bad, ", "))
	}
	return nil
}

// newReport gives the scenario a report file for its helpers and what it
// must hold at the end.
func (s *scen) newReport(spec reportSpec) string {
	f, err := os.CreateTemp(s.t.TempDir(), "report-*")
	if err != nil {
		s.invalidf("report file: %v", err)
	}
	_ = f.Close()
	s.locked(func() { s.reports = append(s.reports, reportFile{path: f.Name(), spec: spec}) })
	return f.Name()
}

// awaitReport waits until the report holds exactly n records of the kind
// and returns their PIDs. A corrupt report, a limit record, a duplicate,
// more than n, or the deadline stops the scenario as invalid: the load it
// needs was not built.
func (s *scen) awaitReport(path, kind string, n int, d time.Duration) []int {
	pids, err := awaitRecords(path, kind, n, d)
	if err != nil {
		s.invalidf("load not established: %v", err)
	}
	return pids
}

// awaitRecords is awaitReport for helpers, which report the error.
func awaitRecords(path, kind string, n int, d time.Duration) ([]int, error) {
	deadline := time.Now().Add(d)
	for {
		lines, err := parseReport(path, false)
		if err != nil {
			return nil, fmt.Errorf("report: %w", err)
		}
		by, err := tally(lines)
		if err != nil {
			return nil, fmt.Errorf("report: %w", err)
		}
		got := by[kind]
		switch {
		case len(got) > n:
			return nil, fmt.Errorf("report: %d %s records, want %d", len(got), kind, n)
		case len(got) == n:
			return got, nil
		case time.Now().After(deadline):
			return nil, fmt.Errorf("report: only %d of %d %s records within %v: %w", len(got), n, kind, d, errNotYet)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
