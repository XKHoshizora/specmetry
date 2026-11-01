package checkrun_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/checkrun"
)

// handleProtectFromClose is HANDLE_FLAG_PROTECT_FROM_CLOSE: CloseHandle
// on such a handle fails (ERROR_INVALID_HANDLE) and leaves it open.
const handleProtectFromClose = 0x2

// protector makes finalPath's close of the handles of the paths chosen
// fail natively, by protecting them from close just before; the test's
// cleanup unprotects and closes each, checking both. The failure is made
// on purpose: it is a real CloseHandle failure, not one that happened on
// its own.
func protector(t *testing.T, chosen func(p string) bool) {
	t.Helper()
	var held []windows.Handle
	checkrun.OnHandle(t, func(p string, h windows.Handle) {
		if !chosen(p) {
			return
		}
		if err := windows.SetHandleInformation(h, handleProtectFromClose, handleProtectFromClose); err != nil {
			t.Errorf("protecting a handle: %v", err)
			return
		}
		held = append(held, h)
	})
	t.Cleanup(func() {
		for _, h := range held {
			if err := windows.SetHandleInformation(h, handleProtectFromClose, 0); err != nil {
				t.Errorf("unprotecting a handle: %v", err)
			}
			if err := windows.CloseHandle(h); err != nil {
				t.Errorf("closing a handle: %v", err)
			}
		}
	})
}

// Review 123 B-123-2: finalPath consumes its own handle's close. A close
// that fails after the path was read returns no path but a
// *ReleaseError: natively (the handle protected from close), injected, or
// both, neither covering the other. The control resolves the path.
func TestFinalPathConsumesItsClose(t *testing.T) {
	dir := t.TempDir()
	want, err := checkrun.FinalPath(dir)
	if err != nil || want == "" {
		t.Fatalf("the control: %q, %v", want, err)
	}
	var rel *checkrun.ReleaseError
	protect := true
	protector(t, func(string) bool { return protect })
	p, err := checkrun.FinalPath(dir)
	if p != "" || !errors.As(err, &rel) || !errors.Is(err, windows.ERROR_INVALID_HANDLE) {
		t.Fatalf("a native close failure: %q, %v", p, err)
	}

	injected := errors.New("injected: handle close")
	checkrun.OnRelease(t, func(r string) error {
		if r == "handle" {
			return injected
		}
		return nil
	})
	protect = false
	if p, err := checkrun.FinalPath(dir); p != "" || !errors.As(err, &rel) || !errors.Is(err, injected) || errors.Is(err, windows.ERROR_INVALID_HANDLE) {
		t.Fatalf("an injected close failure: %q, %v", p, err)
	}
	protect = true
	if p, err := checkrun.FinalPath(dir); p != "" || !errors.Is(err, injected) || !errors.Is(err, windows.ERROR_INVALID_HANDLE) {
		t.Fatalf("a native and an injected close failure: %q, %v", p, err)
	}
}

// Review 123 B-123-2: a candidate whose handle's close fails is not a
// candidate missing: the lookup fails (environment), even though the next
// candidate on PATH is good, and a root's is not a root unverified. The
// control finds the first candidate.
func TestLookupFailsOnACloseFailure(t *testing.T) {
	base := t.TempDir()
	first := file(t, filepath.Join(base, "one", "pwsh.exe"))
	file(t, filepath.Join(base, "two", "pwsh.exe"))
	path := filepath.Join(base, "one") + string(os.PathListSeparator) + filepath.Join(base, "two")
	in, problem, err := checkrun.Lookup(ctx, checkrun.NewFinder([]string{t.TempDir()}, path, nil), "pwsh")
	if err != nil || problem != "" || in.Path != final(t, first) {
		t.Fatalf("the control: %+v, %q, %v", in, problem, err)
	}
	protector(t, func(p string) bool { return p == first })
	in, problem, err = checkrun.Lookup(ctx, checkrun.NewFinder([]string{t.TempDir()}, path, nil), "pwsh")
	var rel *checkrun.ReleaseError
	if in.Path != "" || problem != "" || apperr.ExitCode(err) != apperr.ExitEnvironment || !errors.As(err, &rel) ||
		!strings.Contains(err.Error(), "not confirmed released") {
		t.Fatalf("a candidate's close failure: %+v, %q, %v", in, problem, err)
	}
	root := t.TempDir()
	protector(t, func(p string) bool { return p == root })
	in, problem, err = checkrun.Lookup(ctx, checkrun.NewFinder([]string{root}, path, nil), "pwsh")
	if in.Path != "" || problem != "" || apperr.ExitCode(err) != apperr.ExitEnvironment || !errors.As(err, &rel) {
		t.Fatalf("a root's close failure: %+v, %q, %v", in, problem, err)
	}
}

// Review 123 B-123-2: through Prepare, a handle close that fails at step 7
// refuses the batch with environment (exit 6), no detail, its controlled
// note, not shell_unavailable; the close error is in the chain. The close
// failure is injected; the other Prepare tests are the control.
func TestPrepareHandleCloseFailure(t *testing.T) {
	newWorkspace(t).criteria(t, ac{key: "AC-001-1", check: sh("echo one")})
	injected := errors.New("injected: handle close")
	checkrun.OnRelease(t, func(r string) error {
		if r == "handle" {
			return injected
		}
		return nil
	})
	h := checkrun.NewHost()
	_, err := checkrun.PrepareWith(ctx, h, all(), terminal)
	doc := refused(t, h, err, apperr.ExitEnvironment)
	if !strings.Contains(doc, "not confirmed released") || strings.Contains(doc, "shell_unavailable") || !errors.Is(err, injected) {
		t.Fatalf("the refusal: %s", doc)
	}
}

// closeFailures adds errs, in their order, to the next handle closes' own
// results, and counts every handle close from now on.
func closeFailures(t *testing.T, errs ...error) *int {
	t.Helper()
	closes := 0
	checkrun.OnRelease(t, func(r string) error {
		if r != "handle" {
			return nil
		}
		closes++
		if closes <= len(errs) {
			return errs[closes-1]
		}
		return nil
	})
	return &closes
}

// pathOf is a PATH of the directories named under base.
func pathOf(base string, dirs ...string) string {
	entries := make([]string, len(dirs))
	for i, d := range dirs {
		entries[i] = filepath.Join(base, d)
	}
	return strings.Join(entries, string(os.PathListSeparator))
}

// mentions reports whether an encoded document has s in it, as it is or
// escaped as a JSON string.
func mentions(doc, s string) bool {
	b, _ := json.Marshal(s)
	return strings.Contains(doc, s) || strings.Contains(doc, string(b[1:len(b)-1]))
}

// Review 124 B-123-2-R: a finder keeps every handle close that did not
// confirm its release, not only the first: two registered roots whose
// closes fail with E1 and E2, and two candidates on one PATH whose closes
// fail with E1 and E2. Each handle is closed once, no more; the lookup
// fails with environment (exit 6), no path and no problem; E1 and E2 are
// both in its chain; and step 7 fails the same way, without a further
// close, in a valid error.v1 that names neither a path nor the injected
// text. The close failures are injected (added to each close's own
// result, which was nil): no native failure was made.
func TestFinderKeepsEveryCloseFailure(t *testing.T) {
	e1, e2 := errors.New("injected: E1"), errors.New("injected: E2")
	pwsh := manyItems(1, func(int) string { return "pwsh" }, func(int) string { return "echo" })
	keepsBoth := func(t *testing.T, f *checkrun.Finder, closes *int, hidden ...string) {
		t.Helper()
		in, problem, err := checkrun.Lookup(ctx, f, "pwsh")
		var rel *checkrun.ReleaseError
		if in.Path != "" || problem != "" || apperr.ExitCode(err) != apperr.ExitEnvironment || !errors.As(err, &rel) ||
			!errors.Is(err, e1) || !errors.Is(err, e2) || *closes != 2 {
			t.Fatalf("the lookup: %+v, %q, %v; %d handle closes", in, problem, err, *closes)
		}
		err = checkrun.Interpreters(ctx, pwsh, f)
		if apperr.ExitCode(err) != apperr.ExitEnvironment || !errors.Is(err, e1) || !errors.Is(err, e2) || *closes != 2 {
			t.Fatalf("step 7: %v; %d handle closes", err, *closes)
		}
		doc := envelope(t, err)
		if !strings.Contains(doc, "not confirmed released") || strings.Contains(doc, "shell_unavailable") ||
			strings.Contains(doc, "injected") || len(apperr.From(err).Details) != 0 {
			t.Fatalf("step 7's error.v1: %s", doc)
		}
		for _, h := range hidden {
			if mentions(doc, h) {
				t.Fatalf("step 7's error.v1 names %s: %s", h, doc)
			}
		}
	}
	t.Run("two roots", func(t *testing.T) {
		a, b := t.TempDir(), t.TempDir()
		closes := closeFailures(t, e1, e2)
		keepsBoth(t, checkrun.NewFinder([]string{a, b}, "", nil), closes, a, b)
	})
	t.Run("two candidates", func(t *testing.T) {
		base := t.TempDir()
		file(t, filepath.Join(base, "one", "pwsh.exe"))
		file(t, filepath.Join(base, "two", "pwsh.exe"))
		f := checkrun.NewFinder([]string{t.TempDir()}, pathOf(base, "one", "two"), nil)
		keepsBoth(t, f, closeFailures(t, e1, e2), base)
	})
}

// Review 124 B-123-2-R, the controls with the same seam: one candidate's
// close that fails, the next one healthy, still fails the lookup; a PATH
// entry without the candidate is skipped and the next one chosen; two
// healthy candidates give the first. The close failure is injected.
func TestFinderCloseControls(t *testing.T) {
	e1 := errors.New("injected: E1")
	base := t.TempDir()
	one := final(t, file(t, filepath.Join(base, "one", "pwsh.exe")))
	two := final(t, file(t, filepath.Join(base, "two", "pwsh.exe")))
	if err := os.MkdirAll(filepath.Join(base, "none"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name   string
		dirs   []string
		errs   []error
		want   string // the path chosen; "" when the lookup fails
		closes int
	}{
		{"one close failure, the next healthy", []string{"one", "two"}, []error{e1}, "", 2},
		{"a candidate missing, the next healthy", []string{"none", "two"}, nil, two, 1},
		{"healthy", []string{"one", "two"}, nil, one, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := checkrun.NewFinder([]string{t.TempDir()}, pathOf(base, c.dirs...), nil)
			closes := closeFailures(t, c.errs...)
			in, problem, err := checkrun.Lookup(ctx, f, "pwsh")
			failed := apperr.ExitCode(err) == apperr.ExitEnvironment && errors.Is(err, e1)
			if in.Path != c.want || problem != "" || *closes != c.closes || c.want == "" && !failed || c.want != "" && err != nil {
				t.Fatalf("%+v, %q, %v; %d handle closes", in, problem, err, *closes)
			}
		})
	}
}
