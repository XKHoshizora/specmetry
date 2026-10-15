package gitx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// Trace2 event streams for the probe's parser, modelled on what git 2.56,
// 2.43 and 2.32.0 write for the probe's two runs (codex-review-023 B1). No
// git runs here.

const (
	rootSID  = "20261015T000000.000001Z-H0000abcd-P00000001"
	childSID = rootSID + "/20261015T000000.000002Z-H0000abcd-P00000002"
)

var (
	probeArgv = []any{"git", "--no-pager", "--no-replace-objects", "-c", "core.fsmonitor=", "--git-dir=/tmp/x/test.git", "cat-file", "-t", probeObject}
	fetchArgv = []any{"git", "-c", "fetch.negotiationAlgorithm=noop", "fetch", "origin", "--no-tags", "--no-write-fetch-head", "--recurse-submodules=no", "--filter=blob:none", "--stdin"}
)

// event is one trace2 event line.
func event(name, sid string, fields map[string]any) string {
	e := map[string]any{"event": name, "sid": sid, "thread": "main", "time": "2026-10-15T00:00:00.000000Z", "file": "trace2.c", "line": 1}
	for k, v := range fields {
		e[k] = v
	}
	b, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func version(sid string) string {
	return event("version", sid, map[string]any{"evt": "4", "exe": "2.56.0"})
}
func start(sid string, argv []any) string {
	return event("start", sid, map[string]any{"t_abs": 0.001, "argv": argv})
}
func exitEvent(sid string, code int) string {
	return event("exit", sid, map[string]any{"t_abs": 0.01, "code": code})
}
func atexit(sid string, code int) string {
	return event("atexit", sid, map[string]any{"t_abs": 0.01, "code": code})
}
func other(name string) string { return event(name, rootSID, map[string]any{"msg": "x"}) }
func childStart(id int) string {
	return event("child_start", rootSID, map[string]any{"child_id": id, "child_class": "?", "use_shell": false, "argv": fetchArgv})
}
func childExit(id int) string {
	return event("child_exit", rootSID, map[string]any{"child_id": id, "pid": 2, "code": 128, "t_rel": 0.01})
}

// fetchChild is what the fetch git writes itself, under sid.
func fetchChild(sid string) []string {
	return []string{version(sid), start(sid, fetchArgv), event("cmd_name", sid, map[string]any{"name": "fetch"}),
		event("region_enter", sid, map[string]any{"category": "fetch"}), event("error", sid, map[string]any{"msg": "x"}),
		exitEvent(sid, 128), atexit(sid, 128)}
}

func join(parts ...[]string) []string { return slices.Concat(parts...) }

// The streams of real git: a test run without a child (2.56; 2.43 writes
// two errors and no data_json; 2.32.0 no cmd_ancestry either), and a run
// that fetches (any control; 2.32.0's test run too).
var (
	legitTest = []string{version(rootSID), start(rootSID, probeArgv), other("data_json"), other("cmd_ancestry"),
		other("cmd_name"), other("error"), other("data_json"), exitEvent(rootSID, 128), atexit(rootSID, 128)}
	legitTest243 = []string{version(rootSID), start(rootSID, probeArgv), other("cmd_ancestry"), other("cmd_name"),
		other("error"), other("error"), exitEvent(rootSID, 128), atexit(rootSID, 128)}
	legitTest232 = []string{version(rootSID), start(rootSID, probeArgv), other("cmd_name"), other("error"),
		exitEvent(rootSID, 128), atexit(rootSID, 128)}
	legitFetch = join([]string{version(rootSID), start(rootSID, probeArgv), other("data_json"), other("cmd_ancestry"),
		other("cmd_name"), childStart(0), other("data")}, fetchChild(childSID),
		[]string{childExit(0), other("error"), other("data_json"), exitEvent(rootSID, 128), atexit(rootSID, 128)})
)

// readEvents runs readProbeEvents on lines written to a file.
func readEvents(t *testing.T, code int, lines []string) (probeResult, bool) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "events")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return readProbeEvents(path, code)
}

// showsNoChild is what the probe takes for a git that never fetches: a
// complete run without a child.
func showsNoChild(r probeResult, ok bool) bool {
	return ok && r.complete() && r.children == 0 && !r.fetched
}

// Real git's streams are read as they are: a test run without a child
// (atexit after exit, or no atexit at all), and a fetch.
func TestProbeEventsOfRealGit(t *testing.T) {
	for name, lines := range map[string][]string{
		"2.56":            legitTest,
		"2.43":            legitTest243,
		"2.32.0":          legitTest232,
		"without atexit":  legitTest[:len(legitTest)-1],
		"an empty line":   append(slices.Clone(legitTest[:3]), append([]string{""}, legitTest[3:]...)...),
		"a thread's data": append(slices.Clone(legitTest[:3]), append([]string{event("data", rootSID, map[string]any{"thread": "preload-0"})}, legitTest[3:]...)...),
	} {
		if r, ok := readEvents(t, 128, lines); !showsNoChild(r, ok) {
			t.Errorf("%s: %+v, trusted %v; want a complete run without a child", name, r, ok)
		}
	}
	if r, ok := readEvents(t, 128, legitFetch); !ok || !r.complete() || !r.fetched || r.children != 1 {
		t.Errorf("a fetch: %+v, trusted %v; want a complete run with one fetch", r, ok)
	}
}

// codex-review-023 B1: a stream that is incomplete, out of order,
// contradictory, or holds an event no process of the run can account for,
// shows nothing: it is not trusted, or not complete, and never taken for a
// run without a child. The first three are Codex's counterexamples.
func TestProbeEventsThatShowNothing(t *testing.T) {
	head := []string{version(rootSID), start(rootSID, probeArgv), other("cmd_name")}
	tail := []string{other("error"), exitEvent(rootSID, 128), atexit(rootSID, 128)}
	grandchild := childSID + "/20261015T000000.000003Z-H0000abcd-P00000003"
	cases := []struct {
		name  string
		code  int
		lines []string
	}{
		{"A: a child's events without git's child_start", 128, join(head, fetchChild(childSID), tail)},
		{"B: exit before start", 128, []string{version(rootSID), exitEvent(rootSID, 128), start(rootSID, probeArgv), other("cmd_name"), atexit(rootSID, 128)}},
		{"C: exit 0, then exit 128", 128, join(head, []string{exitEvent(rootSID, 0), other("error"), exitEvent(rootSID, 128), atexit(rootSID, 128)})},
		{"the same exit twice", 128, join(head, []string{exitEvent(rootSID, 128)}, tail)},
		{"start twice", 128, join(head, []string{start(rootSID, probeArgv)}, tail)},
		{"start twice, the second of another command", 128, join(head, []string{start(rootSID, []any{"git", "version"})}, tail)},
		{"atexit twice", 128, append(join(head, tail), atexit(rootSID, 128))},
		{"atexit with another status", 128, join(head, []string{exitEvent(rootSID, 128), atexit(rootSID, 0)})},
		{"atexit without exit", 128, join(head, []string{atexit(rootSID, 128)})},
		{"an event after atexit", 128, append(join(head, tail), other("data"))},
		{"an event before start", 128, []string{version(rootSID), other("cmd_name"), start(rootSID, probeArgv), other("error"), exitEvent(rootSID, 128), atexit(rootSID, 128)}},
		{"version after start", 128, join(head, []string{version(rootSID)}, tail)},
		{"child_exit without child_start", 128, join(head, []string{childExit(2)}, tail)},
		{"child_ready without child_start", 128, join(head, []string{event("child_ready", rootSID, map[string]any{"child_id": 0, "ready": "ready"})}, tail)},
		{"another child event without child_start", 128, join(head, []string{event("child_signal", rootSID, map[string]any{"child_id": 0})}, tail)},
		{"child_start without child_id", 128, join(head, []string{event("child_start", rootSID, map[string]any{"argv": fetchArgv}), childExit(0)}, tail)},
		{"child_start without child_exit", 128, join(head, []string{childStart(0)}, fetchChild(childSID), tail)},
		{"the same child_id twice", 128, join(head, []string{childStart(0), childExit(0), childStart(0), childExit(0)}, tail)},
		{"child_exit after git's exit", 128, join(head, []string{childStart(0)}, fetchChild(childSID), []string{other("error"), exitEvent(rootSID, 128), childExit(0), atexit(rootSID, 128)})},
		{"a grandchild's events only", 128, join(head, fetchChild(grandchild), tail)},
		{"two children's events, one child_start", 128, join(head, []string{childStart(1)}, fetchChild(childSID), fetchChild(rootSID+"/20261015T000000.000004Z-H0000abcd-P00000004"), []string{childExit(1)}, tail)},
		{"a child's event before git's start", 128, join([]string{version(rootSID)}, fetchChild(childSID), []string{start(rootSID, probeArgv), other("cmd_name")}, tail)},
		{"a child's event after git's atexit", 128, join(head, []string{childStart(0), childExit(0)}, tail, []string{event("data", childSID, nil)})},
		{"exit status other than git's", 128, join(head, []string{other("error"), exitEvent(rootSID, 1), atexit(rootSID, 1)})},
		{"exit without a status", 128, join(head, []string{event("exit", rootSID, nil), atexit(rootSID, 128)})},
		{"git succeeded", 0, join(head, []string{exitEvent(rootSID, 0), atexit(rootSID, 0)})},
		{"no exit", 128, head},
		{"no start", 128, []string{version(rootSID), other("cmd_name"), exitEvent(rootSID, 128), atexit(rootSID, 128)}},
		{"no event", 128, nil},
		{"a line that is not JSON", 128, join(head, []string{`{"event":"error","sid":`}, tail)},
		{"an event without a sid", 128, join(head, []string{`{"event":"data"}`}, tail)},
		{"another process's events", 128, join(head, []string{event("data", "20261015T000000.000009Z-H0000abcd-P00000009", nil)}, tail)},
		{"a sid that only starts like git's", 128, join(head, []string{event("data", rootSID+"0", nil)}, tail)},
		{"two runs of the probe's command", 128, join(head, tail, []string{start("20261015T000000.000009Z-H0000abcd-P00000009", probeArgv)})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, ok := readEvents(t, c.code, c.lines)
			if showsNoChild(r, ok) {
				t.Fatalf("taken for a run without a child: %+v", r)
			}
			if ok && r.complete() {
				t.Errorf("trusted as a complete run: %+v", r)
			}
		})
	}
}

// codex-review-023 B1, through the probe and the gate: a stand-in git
// (no git runs) writes a fetching control and, for the test run, one of
// the streams above. Only real git's stream without a child gives a git
// that never fetches; a fetch in the test run gives one that fetches;
// every other stream finds nothing out: object reads are refused (exit 6),
// nothing is kept, and the next check probes again.
func TestProbeTrustsOnlyCompleteEvents(t *testing.T) {
	cases := []struct {
		name string
		test []string
		want objectSafety
	}{
		{"a run without a child", legitTest, neverFetches},
		{"a fetch", legitFetch, fetches},
		{"A: a child's events without git's child_start", join(legitTest[:5], fetchChild(childSID), legitTest[5:]), unprobed},
		{"B: exit before start", []string{version(rootSID), exitEvent(rootSID, 128), start(rootSID, probeArgv), other("cmd_name"), atexit(rootSID, 128)}, unprobed},
		{"C: exit 0, then exit 128", join(legitTest[:6], []string{exitEvent(rootSID, 0)}, legitTest[6:]), unprobed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir, env, calls := testutil.FakeTraceGit(t, strings.Join(legitFetch, "\n")+"\n", strings.Join(c.test, "\n")+"\n", 128)
			useGit(t, dir, env)
			err := CheckObjectReads(ctx)
			if got := kept(); got != c.want {
				t.Fatalf("kept %v, want %v (%v)", got, c.want, err)
			}
			switch c.want {
			case neverFetches:
				if err != nil {
					t.Fatal(err)
				}
			case fetches:
				if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "does not honour") {
					t.Fatalf("CheckObjectReads = %v", err)
				}
			default:
				if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "cannot tell") {
					t.Fatalf("CheckObjectReads = %v", err)
				}
				probes := func() int {
					n := 0
					for _, call := range calls() {
						if slices.Contains(call.Args, "cat-file") {
							n++
						}
					}
					return n
				}
				before := probes()
				if err := CheckObjectReads(ctx); apperr.ExitCode(err) != apperr.ExitEnvironment || kept() != unprobed {
					t.Fatalf("second check: %v, kept %v", err, kept())
				}
				if probes() != before+2 {
					t.Errorf("the second check ran the probe's runs %d times, want 2: nothing may be kept", probes()-before)
				}
			}
		})
	}
}
