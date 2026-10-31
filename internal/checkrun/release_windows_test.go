package checkrun_test

import (
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
