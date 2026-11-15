package controls

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/toolctl"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

// The LC-9 regressions of Review 154 (review-resolution §202). R4-a and
// R4-b replace one real operation the close-real-error variant's seam calls
// with one that returns an error the plan did not expect; R4-c does the
// same for the real request of blocked-send (Linux only). The complete
// expectations of the variants are unchanged: each must be a miss.

func lc9CaseNamed(t *testing.T, name string) lc9Case {
	t.Helper()
	for _, k := range lc9Cases() {
		if k.name == name {
			return k
		}
	}
	t.Fatalf("no LC-9 case %q", name)
	return lc9Case{}
}

func judgeCase(t *testing.T, k lc9Case) verdict.Variant {
	t.Helper()
	v := k.variant(runtime.GOOS)
	ok, _ := v.Applies(runtime.GOOS)
	if !ok {
		t.Skipf("%s is not applicable on %s", k.name, runtime.GOOS)
	}
	env, _ := newEnv(t)
	o := v.Run(env)
	j := Judge(v, runtime.GOOS, o, true)
	t.Logf("%s: %s %v", k.name, j.Judgement, j.Reasons)
	return j
}

var errActual = errors.New("r154: a real result the plan did not expect")

// TestR154CloseRealErrorOtherSource is R4-a: at the site of the planned
// close failure (stdout-r), the real Close returns another error. The
// planned fault's source did not happen; the variant must not be a hit.
func TestR154CloseRealErrorOtherSource(t *testing.T) {
	k := lc9CaseNamed(t, "close-real-error")
	k.realClose = func(d int, f *os.File) error {
		err := f.Close()
		if d == toolctl.StdoutR && err == nil {
			return errActual
		}
		return err
	}
	if j := judgeCase(t, k); j.Judgement == verdict.Hit {
		t.Errorf("a real close error at the planned site was taken for the planned seam fault: %s", j.Judgement)
	}
}

// TestR154CloseRealErrorElsewhere is R4-b: another necessary operation
// (stderr-w's Close) has an extra real error; the variant must not be a
// hit.
func TestR154CloseRealErrorElsewhere(t *testing.T) {
	k := lc9CaseNamed(t, "close-real-error")
	k.realClose = func(d int, f *os.File) error {
		err := f.Close()
		if d == toolctl.StderrW && err == nil {
			return errActual
		}
		return err
	}
	if j := judgeCase(t, k); j.Judgement == verdict.Hit {
		t.Errorf("an extra real close error was not rejected: %s", j.Judgement)
	}
}

// TestR154RequestOtherKind is R4-c (Linux): the real request of
// blocked-send returns EBADF instead of the expected ESRCH (the process was
// reaped); a request error of another kind must not be a hit.
func TestR154RequestOtherKind(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the pidfd exists only on Linux")
	}
	k := lc9CaseNamed(t, "blocked-send")
	k.realSend = func(fd int) error {
		if err := toolctl.RealSendKill(fd); err == nil || !strings.Contains(err.Error(), "no such process") {
			return err
		}
		return syscall.EBADF
	}
	if j := judgeCase(t, k); j.Judgement == verdict.Hit {
		t.Errorf("a request error of another kind was taken for the expected one: %s", j.Judgement)
	}
}
