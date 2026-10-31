package checkrun_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/checkrun"
)

// available is a shell this machine has: sh (/bin/sh) on Unix, cmd (in
// the system directory) on Windows.
func available() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "sh"
}

func acKey(i int) string { return fmt.Sprintf("AC-001-%d", i+1) }

// manyItems is n items of AC-001-1 on; shell and command give each one's.
func manyItems(n int, shell, command func(i int) string) []checkrun.Item {
	items := make([]checkrun.Item, n)
	for i := range items {
		s := shell(i)
		items[i] = checkrun.Item{Criterion: acKey(i), CriterionID: fmt.Sprintf("01JC00000000000000000C%04d", i),
			CheckConfigHash: strings.Repeat("a", 64), Check: baseline.Check{Command: command(i), Shell: &s, Workdir: "."}}
	}
	return items
}

// manyCriteria is n active automated criteria with a check, AC-001-1 on,
// and their stored hashes.
func manyCriteria(n int) ([]baseline.Record, map[string]string) {
	records := make([]baseline.Record, n)
	hashes := map[string]string{}
	for i := range records {
		id := fmt.Sprintf("01JC00000000000000000C%04d", i)
		records[i] = baseline.Record{ID: id, Type: baseline.Criterion, Key: acKey(i), Status: "active", Revision: 1, RequirementRevision: 1,
			Verification: "automated", Check: &baseline.Check{Command: "echo ok", Workdir: ".", TimeoutSec: 300, Rationale: "r"}}
		hashes[id] = strings.Repeat("a", 64)
	}
	return records, hashes
}

func keysOf(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = acKey(i)
	}
	return out
}

// Review 123 B-123-1: each of the three refusals that list a problem per
// argument or criterion checks every one, keeps at most 200 real details
// in their order, at their own positions, and says in its message how
// many problems there are and how many are left out once there are more.
// The step's exit code stays. Each refusal is made into error.v1 by
// apperr.ToEnvelope, encoded and checked against the schema. A problem
// only at the 201st argument or criterion still refuses.
func TestRefusalDetailsBoundary(t *testing.T) {
	none := checkrun.NewFinder([]string{t.TempDir()}, "", nil)
	entries := []struct {
		name     string
		exit     int
		pointer  func(i int) string
		problems func(n int) error
		onlyLast func() error
	}{
		{
			"selection (step 5)", apperr.ExitInvalidInput, func(i int) string { return fmt.Sprintf("argument %d", i+1) },
			func(n int) error {
				_, err := checkrun.ChooseIn(nil, nil, given(keysOf(n)...))
				return err
			},
			func() error {
				records, hashes := manyCriteria(200)
				_, err := checkrun.ChooseIn(records, hashes, given(append(keysOf(200), "AC-999-1")...))
				return err
			},
		},
		{
			"static rules (step 6)", apperr.ExitInvalidInput, acKey,
			func(n int) error {
				return checkrun.StaticRules(manyItems(n, func(int) string { return "sh" }, func(int) string { return "make\r" }))
			},
			func() error {
				return checkrun.StaticRules(manyItems(201, func(int) string { return "sh" }, func(i int) string {
					if i == 200 {
						return "make\r"
					}
					return "make"
				}))
			},
		},
		{
			"interpreters (step 7)", apperr.ExitEnvironment, acKey,
			func(n int) error {
				return checkrun.Interpreters(ctx, manyItems(n, func(int) string { return "pwsh" }, func(int) string { return "echo" }), none)
			},
			func() error {
				return checkrun.Interpreters(ctx, manyItems(201, func(i int) string {
					if i == 200 {
						return "pwsh"
					}
					return available()
				}, func(int) string { return "echo" }), none)
			},
		},
	}
	for _, e := range entries {
		for _, n := range []int{199, 200, 201} {
			err := e.problems(n)
			if got := apperr.ExitCode(err); got != e.exit {
				t.Fatalf("%s, %d problems: exit %d (%v)", e.name, n, got, err)
			}
			doc := envelope(t, err)
			details := apperr.From(err).Details
			if want := min(n, checkrun.MaxDetails); len(details) != want {
				t.Fatalf("%s, %d problems: %d details, want %d", e.name, n, len(details), want)
			}
			for i, d := range details {
				if d.Pointer != e.pointer(i) {
					t.Fatalf("%s, %d problems: detail %d at %q, want %q", e.name, n, i, d.Pointer, e.pointer(i))
				}
			}
			omitted := strings.Contains(doc, "left out")
			if want := fmt.Sprintf("%d problems: the first 200 are listed, %d left out", n, n-200); n > 200 && !strings.Contains(doc, want) ||
				n <= 200 && omitted {
				t.Fatalf("%s, %d problems: %s", e.name, n, doc)
			}
		}
		err := e.onlyLast()
		envelope(t, err)
		details := apperr.From(err).Details
		if apperr.ExitCode(err) != e.exit || len(details) != 1 || details[0].Pointer != e.pointer(200) {
			t.Fatalf("%s, a problem at the 201st only: exit %d, details %v", e.name, apperr.ExitCode(err), details)
		}
	}
}

// Review 123 B-123-2: when a step refuses after Prepare opened the
// database and the private area, and both closes fail, the refusal keeps
// its step's code, exit code and details; its message notes that neither
// is confirmed released; both close errors stay reachable; each was
// closed once; the call is ended on the host. The control: the same
// refusal with healthy closes has no such note. The close failures are
// injected (added to each close's own result, which was nil): no native
// failure of these closes was made.
func TestPrepareRefusalKeepsCloseFailures(t *testing.T) {
	w := newWorkspace(t)
	w.criteria(t, ac{key: "AC-001-1", check: sh("echo one")})
	injected := map[string]error{"scratch": errors.New("injected: scratch close"), "database": errors.New("injected: database close")}
	calls := map[string]int{}
	inject := true
	checkrun.OnRelease(t, func(r string) error {
		calls[r]++
		if inject {
			return injected[r]
		}
		return nil
	})
	unsupported := checkrun.Output{Stdout: &bytes.Buffer{}, Stderr: os.Stderr}
	h := checkrun.NewHost()
	_, err := checkrun.PrepareWith(ctx, h, all(), unsupported)
	doc := refused(t, h, err, apperr.ExitInternal, "stdout")
	if !strings.Contains(doc, "the output cannot carry check output") ||
		!strings.Contains(doc, "cleanup: the data directory's tmp directory and the database closed with an error: not confirmed released") ||
		strings.Contains(doc, "injected") {
		t.Fatalf("the refusal: %s", doc)
	}
	var rel *checkrun.ReleaseError
	if !errors.Is(err, injected["scratch"]) || !errors.Is(err, injected["database"]) || !errors.As(err, &rel) || len(rel.Errs) != 2 {
		t.Fatalf("the close errors are not in the chain: %v", err)
	}
	if calls["scratch"] != 1 || calls["database"] != 1 {
		t.Fatalf("closes: %v", calls)
	}

	inject, calls = false, map[string]int{}
	_, err = checkrun.PrepareWith(ctx, h, all(), unsupported)
	if doc := refused(t, h, err, apperr.ExitInternal, "stdout"); strings.Contains(doc, "cleanup") || errors.As(err, &rel) {
		t.Fatalf("the control: %s", doc)
	}
	if calls["scratch"] != 1 || calls["database"] != 1 {
		t.Fatalf("the control's closes: %v", calls)
	}
}

// Review 123 B-123-2: a Batch's first Close result is kept and returned
// by every later call, which closes nothing again; a failed close is never
// followed by a nil. The control: healthy closes give nil, twice. The
// database close failure is injected.
func TestBatchCloseKeepsItsFirstResult(t *testing.T) {
	newWorkspace(t).criteria(t, ac{key: "AC-001-1", check: sh("echo one")})
	injected := errors.New("injected: database close")
	calls := map[string]int{}
	inject := true
	checkrun.OnRelease(t, func(r string) error {
		calls[r]++
		if inject && r == "database" {
			return injected
		}
		return nil
	})
	h := checkrun.NewHost()
	b, err := checkrun.PrepareWith(ctx, h, given("AC-001-1"), terminal)
	if err != nil {
		t.Fatal(err)
	}
	first := b.Close()
	var rel *checkrun.ReleaseError
	if !errors.As(first, &rel) || !slices.Equal(rel.Resources, []string{"the database"}) || !errors.Is(first, injected) {
		t.Fatalf("the first Close = %v", first)
	}
	if second := b.Close(); second != first {
		t.Fatalf("the second Close = %v, want the first's %v", second, first)
	}
	if calls["scratch"] != 1 || calls["database"] != 1 {
		t.Fatalf("closes: %v", calls)
	}
	if _, active, _ := checkrun.State(h); active != 0 {
		t.Fatalf("%d calls admitted after Close", active)
	}

	inject, calls = false, map[string]int{}
	b, err = checkrun.PrepareWith(ctx, h, given("AC-001-1"), terminal)
	if err != nil {
		t.Fatal(err)
	}
	if first, second := b.Close(), b.Close(); first != nil || second != nil || calls["scratch"] != 1 || calls["database"] != 1 {
		t.Fatalf("the control: %v, %v, closes %v", first, second, calls)
	}
}

// Review 123 B-123-1 with B-123-2: 201 problems at step 5 and a database
// close that fails make one refusal: 200 details, the count of problems
// and the cleanup note both in its message, a valid error.v1. The close
// failure is injected.
func TestPrepareManyProblemsAndACloseFailure(t *testing.T) {
	newWorkspace(t)
	injected := errors.New("injected: database close")
	checkrun.OnRelease(t, func(r string) error {
		if r == "database" {
			return injected
		}
		return nil
	})
	h := checkrun.NewHost()
	_, err := checkrun.PrepareWith(ctx, h, given(keysOf(201)...), terminal)
	pointers := make([]string, 200)
	for i := range pointers {
		pointers[i] = fmt.Sprintf("argument %d", i+1)
	}
	doc := refused(t, h, err, apperr.ExitInvalidInput, pointers...)
	count := strings.Index(doc, "201 problems: the first 200 are listed, 1 left out")
	cleanup := strings.Index(doc, "cleanup: the database closed with an error: not confirmed released")
	if count < 0 || cleanup < count || !errors.Is(err, injected) {
		t.Fatalf("the refusal: %s", doc)
	}
}
