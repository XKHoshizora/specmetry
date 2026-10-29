//go:build darwin

package wp25

// P-3 (spec v8 second draft §19.5 and §19.12, G5; the plan fixed in
// docs/wp2.5-prototype.md §15 before this code): the macOS ABI of the
// E-main candidate, on arm64 and x86_64. Each run keeps its dimensions
// apart, none overwriting another:
//
//   - R, what the kernel returned, every read, before any decoding;
//   - N, the normalized end the test computes from R on its own, by the
//     spec's rules written apart from the decoder;
//   - M, the candidate decoder's answer: an end, or the rule it rejects by;
//   - T, the tool's validity: every failure, kept (s.flag);
//   - E, the environment: native macOS, and Xcode for the layout probe;
//   - F, a planned fault or corruption, and its hit;
//   - C, the cleanup (the result's cleanup).
//
// One verdict follows, in §19.12's order: a tool failure is invalid
// whatever else held; an environment without the mechanism is unsupported;
// a planned corruption the decoder rejects by the expected rule is invalid
// (its gate regression judges it), one it does not reject refuted; the
// planned fault and the native runs by their predictions. A decoder that
// rejects a native value gives the candidate's answer, which is compared
// with the prediction: it is never a tool failure.

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

var p3Budget = &budget{name: "P-3", total: 15 * time.Minute, per: time.Minute}

const (
	p3Inner   = 20 * time.Second  // a run's observation
	p3Outer   = 10 * time.Second  // its checks after the observation
	p3Clean   = 20 * time.Second  // its end
	p3Reserve = 120 * time.Second // to go test's deadline: this run and one run's cleanup
	p3Margin  = time.Minute       // what a tool step leaves to go test's deadline
	p3ToolEnv = "WP25_P3TOOL"
)

// p3Pred is §19.5's derivation for one exit argument: the waitid
// si_status, the kqueue data, the normalized exit value (Q-3 (a)) and the
// exit code Go reads after reaping. It is the spec's table, not computed.
var p3Pred = map[int]struct {
	w    uint32
	k    int64
	n    int
	code int
}{
	7:             {0x7, 0x0700, 7, 7},
	0x100:         {0x100, 0x0000, 0, 0},
	0x107:         {0x107, 0x0700, 7, 7},
	-1:            {0x00ffffff, 0xff00, 255, 255},
	0x7f000007:    {0x7, 0x0700, 7, 7},
	math.MinInt32: {0x0, 0x0000, 0, 0},
}

var p3Exits = []struct {
	name string
	code int
}{{"100", 0x100}, {"107", 0x107}, {"ff", -1}, {"7f", 0x7f000007}, {"80", math.MinInt32}}

type p3Case struct {
	name string
	kind kind
	run  func(*p3Run)
}

func p3Cases() []p3Case {
	cs := []p3Case{{"layout", normal, p3Layout}, {"k.exit7", normal, p3K(7)}, {"k.signal9", normal, p3KSignal}}
	for _, x := range p3Exits {
		cs = append(cs, p3Case{"k.exit-" + x.name, normal, p3K(x.code)})
	}
	cs = append(cs, p3Case{"k.esrch", normal, p3KEsrch}, p3Case{"w.exit7", normal, p3W(7)}, p3Case{"w.signal9", normal, p3WSignal})
	for _, x := range p3Exits {
		cs = append(cs, p3Case{"w.exit-" + x.name, normal, p3W(x.code)})
	}
	return append(cs,
		p3Case{"c.exit7", normal, p3C(7)}, p3Case{"c.exit-107", normal, p3C(0x107)}, p3Case{"c.exit-ff", normal, p3C(-1)},
		p3Case{"group.zombie-leader", normal, p3Group(false)}, p3Case{"group.enum-unknown", fault, p3Group(true)})
}

// TestP3 runs every scenario three times, the layout probe first.
func TestP3(t *testing.T) {
	for _, c := range p3Cases() {
		for n := 1; n <= 3; n++ {
			t.Run(fmt.Sprintf("%s.%d", c.name, n), func(t *testing.T) { runP3(t, c, n) })
		}
	}
}

// p3Run is one run and its dimensions.
type p3Run struct {
	s     *scen
	t     *testing.T
	began time.Time
	outer time.Time // the end of the observation
	seam  string    // a gate regression's planned corruption, "" otherwise
	rule  string    // and the rule expected to reject it
	hits  int       // of the corruption or the planned fault
	dir   string    // the layout probe's files
	tag   string    // its tools carry this in their environment
	since string    // and start no earlier than this
	ev    p3Evidence
}

// p3Evidence is what a run prints on its WP25P line, before its end. The
// final T, C and V are the result's (invalid, cleanup, verdict).
type p3Evidence struct {
	E      map[string]string `json:"E"`
	R      []p3Read          `json:"R,omitempty"`
	Layout map[string]int64  `json:"R_layout,omitempty"`
	N      []string          `json:"N,omitempty"`
	M      []string          `json:"M,omitempty"`
	F      []string          `json:"F,omitempty"`
	T      []string          `json:"T,omitempty"` // the tool failures found until the line
}

type p3Read struct {
	What string `json:"what"`
	W    *wRaw  `json:"waitid,omitempty"`
	K    *kRaw  `json:"kevent,omitempty"`
	Live string `json:"live,omitempty"` // a read of the members: the PIDs not zombies
	Err  string `json:"error,omitempty"`
}

// p3Layouts are the verdicts of the layout probes run in this process, in
// order; every other scenario needs them all confirmed.
var (
	p3LayoutMu sync.Mutex
	p3Layouts  []verdict
)

func runP3(t *testing.T, c p3Case, n int) {
	id := fmt.Sprintf("P3.%s.%d", c.name, n)
	s := begin(t, "G5", id, c.kind, true, true)
	s.cleanLimit = p3Clean
	r := &p3Run{s: s, t: t, began: time.Now(), ev: p3Evidence{E: map[string]string{}}}
	defer func() {
		// The run's evidence first, then its end (§19.12): nothing the end
		// does changes what was observed.
		r.limits()
		s.locked(func() { r.ev.T = append([]string(nil), s.invalid...) })
		pline(id, r.ev)
		s.done()
		// The seams stay set through the end: the observer's close is there.
		p3Seam.w, p3Seam.k, p3Seam.n, p3Seam.enum, p3Seam.close = nil, nil, 0, nil, nil
		if c.name == "layout" {
			p3LayoutMu.Lock()
			p3Layouts = append(p3Layouts, s.r.Verdict)
			p3LayoutMu.Unlock()
		}
	}()
	r.admit()
	r.environment()
	if c.name != "layout" {
		r.layoutPremise()
	}
	r.seamFor(c.name, n)
	r.closeFault()
	r.lateClose(c.name, n)
	c.run(r)
}

// admit runs the scenario only with one run's budget left and enough time
// to go test's deadline for this run and one run's cleanup; otherwise it
// is not executed, which is invalid.
func (r *p3Run) admit() {
	if ok, left := p3Budget.allow(); !ok {
		r.s.invalidf("not executed: the total budget of P-3 has %v left, less than one run's %v", left.Round(time.Second), p3Budget.per)
	}
	if dl, ok := r.t.Deadline(); ok {
		if left := time.Until(dl); left < p3Reserve {
			r.s.invalidf("not executed: go test's deadline is %v away, less than %v", left.Round(time.Second), p3Reserve)
		}
	}
}

// limits checks the run's own phases (§19.12): the observation and the
// checks after it; done checks the end.
func (r *p3Run) limits() {
	end := time.Now()
	if r.outer.IsZero() {
		r.outer = end
	}
	if d := r.outer.Sub(r.began); d > p3Inner {
		r.s.flag("the observation took %v, over its %v", d.Round(time.Millisecond), p3Inner)
	}
	if d := end.Sub(r.outer); d > p3Outer {
		r.s.flag("the checks after the observation took %v, over their %v", d.Round(time.Millisecond), p3Outer)
	}
}

// obsEnd is the end of the run's observation (§19.12): p3Inner from its
// start, and no later than go test's deadline less p3Margin.
func (r *p3Run) obsEnd() time.Time {
	end := r.began.Add(p3Inner)
	if dl, ok := r.t.Deadline(); ok && dl.Add(-p3Margin).Before(end) {
		end = dl.Add(-p3Margin)
	}
	return end
}

// mark ends the observation.
func (r *p3Run) mark() {
	if r.outer.IsZero() {
		r.outer = time.Now()
	}
}

// p3Meta reads the environment's metadata once (§19.12): the values as
// read, "absent" for a proc_translated that does not exist, "unknown" for
// a read that failed, which is also returned as a failure.
var (
	p3MetaOnce sync.Once
	p3MetaV    map[string]string
	p3MetaErr  []string
)

func p3Meta() (map[string]string, []string) {
	p3MetaOnce.Do(func() {
		m := map[string]string{}
		var errs []string
		for _, k := range []string{"kern.version", "kern.osproductversion", "kern.osversion", "hw.machine"} {
			v, err := unix.Sysctl(k)
			if err != nil {
				m[k] = "unknown"
				errs = append(errs, k+": "+err.Error())
				continue
			}
			m[k] = v
		}
		// macOS has no uname system call: x/sys's Uname, like the C
		// library's, reads the machine from sysctl (CTL_HW, HW_MACHINE).
		var u unix.Utsname
		if err := unix.Uname(&u); err != nil {
			m["uname.machine"] = "unknown"
			errs = append(errs, "uname: "+err.Error())
		} else {
			m["uname.machine"] = unix.ByteSliceToString(u.Machine[:])
		}
		v, err := unix.SysctlUint32("sysctl.proc_translated")
		switch {
		case errors.Is(err, unix.ENOENT):
			m["sysctl.proc_translated"] = "absent"
		case err != nil:
			m["sysctl.proc_translated"] = "unknown"
			errs = append(errs, "sysctl.proc_translated: "+err.Error())
		default:
			m["sysctl.proc_translated"] = strconv.FormatUint(uint64(v), 10)
		}
		p3MetaV, p3MetaErr = m, errs
	})
	return p3MetaV, p3MetaErr
}

func metaExtra() map[string]string {
	m, _ := p3Meta()
	return m
}

// environment decides E (§19.12): a native process of the architecture
// Go built for. A metadata read that failed is a tool failure.
func (r *p3Run) environment() {
	m, errs := p3Meta()
	for k, v := range m {
		r.ev.E[k] = v
	}
	r.ev.E["goarch"] = runtime.GOARCH
	for _, e := range errs {
		r.s.flag("metadata could not be read: %s", e)
	}
	want := map[string]string{"arm64": "arm64", "amd64": "x86_64"}[runtime.GOARCH]
	switch {
	case want == "":
		r.s.unsupported("not an architecture of P-3: %s", runtime.GOARCH)
	case m["sysctl.proc_translated"] == "1":
		r.s.unsupported("not native: the process is translated (sysctl.proc_translated 1)")
	case m["uname.machine"] != want:
		r.s.unsupported("not native: the machine is %q, GOARCH %s", m["uname.machine"], runtime.GOARCH)
	}
}

// layoutPremise needs every layout probe of this process confirmed; none,
// or one refuted or invalid, is a tool failure; all unsupported (no
// xcrun) leaves the scenario without its environment.
func (r *p3Run) layoutPremise() {
	p3LayoutMu.Lock()
	vs := append([]verdict(nil), p3Layouts...)
	p3LayoutMu.Unlock()
	count := map[verdict]int{}
	for _, v := range vs {
		count[v]++
	}
	switch {
	case len(vs) == 0:
		r.s.invalidf("not executed: the layout premise is not established: no layout probe ran in this process")
	case count[confirmed] == len(vs):
		r.ev.E["layout"] = fmt.Sprintf("%d probes of this process confirmed", len(vs))
	case count[unsupported] == len(vs):
		r.ev.E["layout"] = "unsupported"
		r.s.unsupported("no layout premise: the layout probe is unsupported here (no xcrun)")
	default:
		r.s.invalidf("not executed: the layout premise does not hold: the probes of this process gave %v", vs)
	}
}

// predict compares one native prediction; a gate regression's run makes
// none (its corruption is judged instead).
func (r *p3Run) predict(what string, want, got any) {
	if r.seam == "" {
		r.s.expect(what, want, got)
	}
}

func (r *p3Run) predictTrue(what string, ok bool, detail string) {
	if r.seam == "" {
		r.s.expectTrue(what, ok, detail)
	}
}

// answer records M, the decoder's answer, and gives the rule it rejected
// by: "" if it accepted. An error that is no rule's is an error of the
// call, a tool failure ("error").
func (r *p3Run) answer(what string, e *exitInfo, err error) string {
	rule := ruleOf(err)
	switch {
	case err == nil && e != nil:
		r.ev.M = append(r.ev.M, what+": "+e.String())
	case rule != "":
		r.ev.M = append(r.ev.M, what+": rejected, "+err.Error()+" (an observation failure: C-19, F1)")
	default:
		r.s.flag("observation failed: %s: %v", what, err)
		return "error"
	}
	return rule
}

// normal compares N with its prediction, and M with N (§19.12's P).
func (r *p3Run) normal(what, n, want string, e *exitInfo, err error) {
	r.ev.N = append(r.ev.N, what+": "+n)
	r.predict("N ("+what+")", want, n)
	m := "none"
	if err == nil && e != nil {
		m = e.String()
	}
	r.predict("M equals N ("+what+")", n, m)
}

// p3NormW is N for a waitid read: §19.5's W1, W3a, W3b and W4, written
// apart from decodeW. What they cannot explain is "none".
func p3NormW(r wRaw, pid int) string {
	if r.Signo != 20 || int64(r.Pid) != int64(pid) {
		return "none"
	}
	switch r.Code {
	case 1:
		if r.Status>>24 != 0 {
			return "none"
		}
		return fmt.Sprintf("exit %d", r.Status%256)
	case 2, 3:
		if r.Status == 0 || r.Status >= 32 {
			return "none"
		}
		if r.Code == 3 {
			return fmt.Sprintf("signal %d core", r.Status)
		}
		return fmt.Sprintf("signal %d", r.Status)
	}
	return "none"
}

// p3NormK is N for a kqueue event: §19.5's K1 and K2, written apart from
// decodeK, with the values of <sys/event.h> (EVFILT_PROC -5, NOTE_EXIT bit
// 31, NOTE_EXITSTATUS bit 26).
func p3NormK(r kRaw, pid int) string {
	if r.Ident != uint64(pid) || r.Filter != -5 || r.Fflags>>31 != 1 || r.Fflags>>26&1 != 1 {
		return "none"
	}
	if r.Data < 0 || r.Data >= 1<<16 {
		return "none"
	}
	lo, hi := r.Data%256, r.Data/256
	sig := lo % 128
	switch {
	case lo == 0:
		return fmt.Sprintf("exit %d", hi)
	case sig >= 1 && sig <= 31 && hi == 0:
		if lo >= 128 {
			return fmt.Sprintf("signal %d core", sig)
		}
		return fmt.Sprintf("signal %d", sig)
	}
	return "none"
}

// p3Seams are the gate regressions' planned corruptions (§19.12; values
// in docs/wp2.5-prototype.md §15.3): the scenario each acts in, and the
// rule expected to reject it.
var p3Seams = map[string]struct{ on, rule string }{
	"p3-w-sipid-once":    {"w.exit7", "W1"},
	"p3-w-signo-once":    {"w.exit7", "W1"},
	"p3-w-code-once":     {"w.exit7", "W1"},
	"p3-w-highbyte-once": {"w.exit7", "W3a"},
	"p3-w-negative-once": {"w.exit7", "W3a"},
	"p3-w-reread-once":   {"w.exit7", "W5"},
	"p3-k-nostatus-once": {"k.exit7", "K1"},
	"p3-k-range-once":    {"k.exit7", "K1"},
	"p3-k-stopped-once":  {"k.exit7", "K2"},
	"p3-c-mismatch-once": {"c.exit7", "C1"},
}

// seamFor sets a gate regression's corruption, in an inner run only, in
// the first run of its scenario.
func (r *p3Run) seamFor(name string, n int) {
	inj := os.Getenv(injectEnv)
	sm, ok := p3Seams[inj]
	if !ok || os.Getenv(innerEnv) == "" || sm.on != name || n != 1 {
		return
	}
	r.seam, r.rule = inj, sm.rule
	r.ev.F = append(r.ev.F, fmt.Sprintf("planned corruption %s, expected to be rejected by %s", inj, sm.rule))
	p3Seam.n = 0
	if strings.HasPrefix(inj, "p3-w-") {
		p3Seam.w = r.rewriteW
	} else {
		p3Seam.k = r.rewriteK
	}
}

// rewriteW rewrites one waitid read in its buffer, after the real call,
// and records the hit: the read, the field, the real and the new value.
func (r *p3Run) rewriteW(n int, si *[siSize]byte) {
	get := func(i int) uint32 { return binary.LittleEndian.Uint32(si[wOff[i]:]) }
	field, i, v := "si_status", 5, uint32(0)
	switch r.seam {
	case "p3-w-sipid-once":
		field, i, v = "si_pid", 3, get(3)+1
	case "p3-w-signo-once":
		field, i, v = "si_signo", 0, 0
	case "p3-w-code-once":
		field, i, v = "si_code", 2, 4
	case "p3-w-highbyte-once":
		v = get(5) + 0x01000000
	case "p3-w-negative-once":
		v = 0xff0000ff
	case "p3-w-reread-once":
		v = 8
	}
	want := 1 // the read it rewrites
	if r.seam == "p3-w-reread-once" {
		want = 2
	}
	if n != want {
		return
	}
	site := fmt.Sprintf("P-3 waitid read %d: %s %#x -> %#x", n, field, get(i), v)
	if injectOnce(r.seam, site, os.Getpid()) {
		binary.LittleEndian.PutUint32(si[wOff[i]:], v)
		r.hits++
		r.ev.F = append(r.ev.F, site)
	}
}

// rewriteK rewrites the kqueue event, after the real call, and records the
// hit.
func (r *p3Run) rewriteK(ev *unix.Kevent_t) {
	fflags, data := ev.Fflags, ev.Data
	switch r.seam {
	case "p3-k-nostatus-once":
		fflags &^= unix.NOTE_EXITSTATUS
	case "p3-k-range-once":
		data = 0x10000
	case "p3-k-stopped-once":
		data = 0x117f // stopped by SIGSTOP
	case "p3-c-mismatch-once":
		data = 0x0800 // an exit of 8
	}
	site := fmt.Sprintf("P-3 kevent: fflags %#x -> %#x, data %#x -> %#x", ev.Fflags, fflags, ev.Data, data)
	if injectOnce(r.seam, site, os.Getpid()) {
		ev.Fflags, ev.Data = fflags, data
		r.hits++
		r.ev.F = append(r.ev.F, site)
	}
}

// The seams of Review 101's gate regressions (gate_test.go), in an inner
// run only. Each answers in place of a real step, or replaces its result
// after it, and records its hit; none is a native failure.
//
//   - p3-enum-error-once: the first read of the members in confirmEnd is
//     answered by an injected error, the call not made (B1);
//   - p3-enum-enomem-3, p3-enum-enomem-4: its first 3 or 4 calls are
//     answered ENOMEM, the calls not made (B3);
//   - p3-close-kq-once: the first real close of an exit observer's kqueue
//     is made, its result replaced by an injected error (B2);
//   - p3-close-out-once: the same for the first output file of a tool
//     step (B2);
//   - p3-create-err-once: the first error file of a tool step is not
//     created; an injected error answers (B2);
//   - p3-enum-error-and-close-once: p3-enum-error-once and
//     p3-close-kq-once in one run.

// enumFault is the construction for the first read of the members in
// confirmEnd, or nil.
func (r *p3Run) enumFault(pgid int) func(string) error {
	inj := os.Getenv(injectEnv)
	if os.Getenv(innerEnv) == "" {
		return nil
	}
	switch inj {
	case "p3-enum-error-once", "p3-enum-error-and-close-once":
		return func(name string) error {
			if name != "kern.proc.pgrp" || !injectOnceAt(inj, inj+"/enum", fmt.Sprintf("P-3 kern.proc.pgrp %d: the read of the members, the call not made, is answered by an injected error", pgid), os.Getpid()) {
				return nil
			}
			r.ev.F = append(r.ev.F, fmt.Sprintf("the read of the members of %d is answered by an injected error, the call not made", pgid))
			return errors.New("injected: the read of the members failed (the call was not made)")
		}
	case "p3-enum-enomem-3", "p3-enum-enomem-4":
		limit := map[string]int{"p3-enum-enomem-3": 3, "p3-enum-enomem-4": 4}[inj]
		n := 0
		return func(name string) error {
			if name != "kern.proc.pgrp" || n == limit {
				return nil
			}
			n++
			injectHit(fmt.Sprintf("P-3 kern.proc.pgrp %d: call %d of %d answered ENOMEM, the call not made", pgid, n, limit), os.Getpid())
			r.ev.F = append(r.ev.F, fmt.Sprintf("constructed ENOMEM %d of %d for the read of the members of %d, the call not made", n, limit, pgid))
			return unix.ENOMEM
		}
	}
	return nil
}

// closeFault sets the construction for the observer's close, if any.
func (r *p3Run) closeFault() {
	inj := os.Getenv(injectEnv)
	if os.Getenv(innerEnv) == "" || inj != "p3-close-kq-once" && inj != "p3-enum-error-and-close-once" {
		return
	}
	p3Seam.close = func(err error) error {
		if err != nil || !injectOnceAt(inj, inj+"/close", "P-3 the real close of an exit observer's kqueue was made; its result is replaced by an injected error", os.Getpid()) {
			return err
		}
		r.ev.F = append(r.ev.F, "the real close of the exit observer's kqueue succeeded; its result is replaced by an injected error")
		return errors.New("injected: closing the kqueue failed (the real close was made)")
	}
}

// closeOut closes a tool step's output file, through the seam
// p3-close-out-once.
func closeOut(f *os.File, step string) error {
	err := f.Close()
	if err == nil && injectOnce("p3-close-out-once", "P-3 the real close of the output file of the tool step "+step+" was made; its result is replaced by an injected error", os.Getpid()) {
		return errors.New("injected: closing the output file failed (the real close was made)")
	}
	return err
}

// createErr creates a tool step's error file, through the seam
// p3-create-err-once.
func createErr(path string) (*os.File, error) {
	if injectOnce("p3-create-err-once", "P-3 the error file "+filepath.Base(path)+" of a tool step is not created; an injected error answers", os.Getpid()) {
		return nil, errors.New("injected: the error file could not be created (no call made)")
	}
	return os.Create(path)
}

// lateClose is the seam p3-layout-close-late of Review 102's regression
// (gate_combo_test.go), in an inner run's first layout probe only: the
// probe holds one more file, os.DevNull, left to its end, where the
// scenario's closer makes the real close and the seam replaces its result
// by an injected error. The failure comes after the run's evidence line
// (its T), in its final result only.
func (r *p3Run) lateClose(name string, n int) {
	if os.Getenv(innerEnv) == "" || !injected("p3-layout-close-late") || name != "layout" || n != 1 {
		return
	}
	f, err := os.Open(os.DevNull)
	if err != nil {
		r.s.invalidf("the seam's late file %s: %v", os.DevNull, err)
	}
	r.s.closer("the seam's late file "+os.DevNull, func() error {
		err := f.Close()
		if err == nil && injectOnce("p3-layout-close-late", "P-3 the real close of the seam's late file "+os.DevNull+", at the layout probe's end, was made; its result is replaced by an injected error", os.Getpid()) {
			return errors.New("injected: closing the late file failed at the end (the real close was made)")
		}
		return err
	})
}

// corruption judges a gate regression's run by the decoder's answer
// (§19.12, 3 and 4): rejected by the expected rule, the planned
// corruption is detected (invalid, the gate judges it); accepted, or
// rejected by another rule, the decoder differs from the spec (refuted).
func (r *p3Run) corruption(rule, detail string) {
	switch {
	case r.hits == 0:
		r.s.flag("the planned corruption %s never reached its site", r.seam)
	case rule == "error":
		// a tool failure, flagged by answer
	case rule == r.rule:
		r.s.flag("planned corruption detected by %s: %s", r.rule, detail)
	default:
		if rule == "" {
			rule = "accepted"
		}
		r.s.expect("the planned corruption is rejected by", r.rule, rule)
	}
}

// observer registers the candidate's kqueue observer of the leader.
func (r *p3Run) observer(tr *unixTree) exitObserver {
	obs, err := newExitObserver(tr.pid)
	if err != nil {
		r.ev.R = append(r.ev.R, p3Read{What: "kevent registration", Err: err.Error()})
		r.s.invalidf("observation failed: the kqueue registration: %v", err)
	}
	r.s.closer(fmt.Sprintf("the exit observer of %d", tr.pid), obs.close)
	return obs
}

// kread records the event the observer's wait got (R); none, or a failed
// call, is a tool failure.
func (r *p3Run) kread(obs exitObserver, err error) *kRaw {
	k := obs.s.lastK
	rd := p3Read{What: "kevent", K: k}
	if err != nil {
		rd.Err = err.Error()
	}
	r.ev.R = append(r.ev.R, rd)
	if k == nil {
		if err != nil {
			r.s.invalidf("observation failed: the kqueue wait: %v", err)
		}
		r.s.invalidf("load not established: no kqueue event within the wait")
	}
	return k
}

// readW is one waitid read of the leader, recorded (R); a failed call is a
// tool failure.
func (r *p3Run) readW(pid int, what string) wRaw {
	w, err := readWaitid(pid)
	if err != nil {
		r.ev.R = append(r.ev.R, p3Read{What: what, Err: err.Error()})
		r.s.invalidf("observation failed: %s: %v", what, err)
	}
	r.ev.R = append(r.ev.R, p3Read{What: what, W: &w})
	return w
}

// zombie waits (at most 3 s) until the leader is this process's own
// unreaped child and a zombie, with the start time it had: the identity
// premise of every read of it (a tool failure otherwise).
func (r *p3Run) zombie(tr *unixTree) {
	deadline := time.Now().Add(3 * time.Second)
	for {
		st, pr := procInfo(tr.pid)
		switch {
		case pr == unknown:
			r.s.invalidf("observation failed: the state of %d cannot be read", tr.pid)
		case pr == absent || st.start != tr.start || st.ppid != os.Getpid():
			r.s.invalidf("identity premise: %d is not this process's own unreaped child", tr.pid)
		case st.zombie:
			return
		case time.Now().After(deadline):
			r.s.invalidf("load not established: %d was not seen exited within 3 s", tr.pid)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// reap reaps the leader once its group is confirmed empty, and predicts
// its status as Go reads it (wait4, 8 bits).
func (r *p3Run) reap(tr *unixTree, cmd *exec.Cmd, want string) {
	r.predict("an unreaped zombie before the reap", true, tr.zombieLeader())
	if !tr.confirm(tKill) {
		r.s.flag("cleanup: the group %d could not be confirmed empty before the reap", tr.pid)
	}
	if !tr.reap(time.Second) {
		r.s.invalidf("cleanup: the leader %d was not reaped within 1 s", tr.pid)
	}
	ws, _ := cmd.ProcessState.Sys().(syscall.WaitStatus)
	r.predict("status after reaping (wait4, Go)", want, waitStatus(ws))
}

func (r *p3Run) predictK(k *kRaw, data int64) {
	const both = unix.NOTE_EXIT | unix.NOTE_EXITSTATUS
	r.predictTrue("kevent: the returned fflags include NOTE_EXIT and NOTE_EXITSTATUS", k.Fflags&both == both, fmt.Sprintf("%#x", k.Fflags))
	r.predict("kevent: data", fmt.Sprintf("%#x", data), fmt.Sprintf("%#x", k.Data))
}

// predictW predicts a waitid read's fields, and that the second read, if
// any, is the same.
func (r *p3Run) predictW(a wRaw, b *wRaw, pid int, code, status uint32) {
	r.predict("waitid: si_signo", 20, a.Signo)
	r.predict("waitid: si_pid", pid, a.Pid)
	r.predict("waitid: si_code", code, a.Code)
	r.predict("waitid: si_status", fmt.Sprintf("%#x", status), fmt.Sprintf("%#x", a.Status))
	if b != nil {
		r.predict("waitid: the second read is the same", fmt.Sprintf("%+v", a), fmt.Sprintf("%+v", *b))
	}
}

// p3K: the kqueue path, registered while the process runs (it exits after
// 200 ms with code).
func p3K(code int) func(*p3Run) {
	return func(r *p3Run) {
		pr := p3Pred[code]
		cmd := helper("exit-after", nil, "200", strconv.Itoa(code))
		tr := startGroup(r.s, cmd)
		obs := r.observer(tr)
		r.s.needLoad(!obs.s.early, "the kqueue registered while the process ran")
		_, _, err := obs.wait(3 * time.Second)
		k := r.kread(obs, err)
		rule := r.answer("kqueue", obs.s.end, err)
		r.mark()
		r.s.worked(1)
		if r.seam != "" {
			r.corruption(rule, fmt.Sprint(err))
		} else {
			r.predictK(k, pr.k)
			r.normal("kqueue", p3NormK(*k, tr.pid), fmt.Sprintf("exit %d", pr.n), obs.s.end, err)
		}
		r.zombie(tr)
		r.reap(tr, cmd, fmt.Sprintf("exit %d", pr.code))
	}
}

// p3KSignal: no event while the process runs; then SIGKILL.
func p3KSignal(r *p3Run) {
	cmd := helper("sleep", nil)
	tr := startGroup(r.s, cmd)
	obs := r.observer(tr)
	r.s.needLoad(!obs.s.early, "the kqueue registered while the process ran")
	t0 := time.Now()
	ok, _, err := obs.wait(300 * time.Millisecond)
	if err != nil {
		r.s.invalidf("observation failed: the bounded wait: %v", err)
	}
	r.ev.R = append(r.ev.R, p3Read{What: fmt.Sprintf("the bounded wait of 300 ms: an event %v, after %v", ok, time.Since(t0).Round(time.Millisecond))})
	r.predict("an event within 300 ms while the process runs", false, ok)
	if err := tr.kill(); err != nil {
		r.s.invalidf("load not established: kill(-pgid): %v", err)
	}
	_, _, err = obs.wait(3 * time.Second)
	k := r.kread(obs, err)
	r.answer("kqueue", obs.s.end, err)
	r.mark()
	r.s.worked(1)
	const both = unix.NOTE_EXIT | unix.NOTE_EXITSTATUS
	r.predictTrue("kevent: the returned fflags include NOTE_EXIT and NOTE_EXITSTATUS", k.Fflags&both == both, fmt.Sprintf("%#x", k.Fflags))
	r.predict("kevent: data, the low 7 bits", 9, k.Data&0x7f)
	r.normal("kqueue", p3NormK(*k, tr.pid), "signal 9", obs.s.end, err)
	r.zombie(tr)
	r.reap(tr, cmd, "signal 9")
}

// p3KEsrch: the registration comes after the exit; the candidate's
// observer takes its early path, one waitid read, and a second read
// follows.
func p3KEsrch(r *p3Run) {
	cmd := helper("exit-after", nil, "0", "7")
	tr := startGroup(r.s, cmd)
	r.zombie(tr)
	obs := r.observer(tr)
	r.ev.R = append(r.ev.R, p3Read{What: "kevent registration", Err: errText(obs.s.regErr)})
	r.predict("the registration of an unreaped zombie", "no such process", errText(obs.s.regErr))
	if obs.s.early {
		_, _, err := obs.wait(3 * time.Second)
		if obs.s.lastW == nil {
			r.s.invalidf("observation failed: the candidate's waitid read: %v", err)
		}
		a := *obs.s.lastW
		r.ev.R = append(r.ev.R, p3Read{What: "waitid 1 (the candidate's)", W: &a})
		b := r.readW(tr.pid, "waitid 2")
		r.answer("waitid (the candidate's early path)", obs.s.end, err)
		r.mark()
		r.s.worked(1)
		r.predictW(a, &b, tr.pid, cldExited, 0x7)
		r.normal("waitid", p3NormW(a, tr.pid), "exit 7", obs.s.end, err)
	} else {
		r.ev.M = append(r.ev.M, "the candidate did not take its early path (registered: "+errText(obs.s.regErr)+")")
		r.mark()
		r.s.worked(1)
	}
	r.reap(tr, cmd, "exit 7")
}

// p3W: waitid only, twice, once the process is an unreaped zombie.
func p3W(code int) func(*p3Run) {
	return func(r *p3Run) {
		pr := p3Pred[code]
		cmd := helper("exit-after", nil, "0", strconv.Itoa(code))
		tr := startGroup(r.s, cmd)
		r.zombie(tr)
		r.waitid2(tr, cldExited, pr.w, fmt.Sprintf("exit %d", pr.n))
		r.reap(tr, cmd, fmt.Sprintf("exit %d", pr.code))
	}
}

// p3WSignal: waitid only, twice, after SIGKILL.
func p3WSignal(r *p3Run) {
	cmd := helper("sleep", nil)
	tr := startGroup(r.s, cmd)
	if err := tr.kill(); err != nil {
		r.s.invalidf("load not established: kill(-pgid): %v", err)
	}
	r.zombie(tr)
	r.waitid2(tr, cldKilled, 9, "signal 9")
	r.reap(tr, cmd, "signal 9")
}

func (r *p3Run) waitid2(tr *unixTree, code, status uint32, want string) {
	a := r.readW(tr.pid, "waitid 1")
	b := r.readW(tr.pid, "waitid 2")
	e, err := decodeW2(a, b, tr.pid)
	rule := r.answer("waitid, two reads", &e, err)
	r.mark()
	r.s.worked(1)
	if r.seam != "" {
		r.corruption(rule, fmt.Sprint(err))
		return
	}
	r.predictW(a, &b, tr.pid, code, status)
	r.normal("waitid", p3NormW(a, tr.pid), want, &e, err)
}

// p3C: both paths for the same process: the kqueue event, then one waitid
// read of the unreaped zombie, and the two ends compared (C1).
func p3C(code int) func(*p3Run) {
	return func(r *p3Run) {
		pr := p3Pred[code]
		cmd := helper("exit-after", nil, "200", strconv.Itoa(code))
		tr := startGroup(r.s, cmd)
		obs := r.observer(tr)
		r.s.needLoad(!obs.s.early, "the kqueue registered while the process ran")
		_, _, kerr := obs.wait(3 * time.Second)
		k := r.kread(obs, kerr)
		krule := r.answer("kqueue", obs.s.end, kerr)
		r.zombie(tr)
		w := r.readW(tr.pid, "waitid 1")
		we, werr := decodeW(w, tr.pid)
		wrule := r.answer("waitid", &we, werr)
		crule, same := "", "not compared (a path did not give an end)"
		var cerr error
		if kerr == nil && werr == nil && obs.s.end != nil {
			if cerr = sameEnd(*obs.s.end, we); cerr != nil {
				crule, same = "C1", "different"
				r.ev.M = append(r.ev.M, "C1: rejected, "+cerr.Error()+" (an observation failure: C-19, F1)")
			} else {
				same = "the same"
				r.ev.M = append(r.ev.M, "C1: the same end")
			}
		}
		r.mark()
		r.s.worked(1)
		if r.seam != "" {
			// The candidate's answer is its first rejection.
			rule, detail := krule, fmt.Sprint(kerr)
			switch {
			case rule == "" && wrule != "":
				rule, detail = wrule, fmt.Sprint(werr)
			case rule == "":
				rule, detail = crule, fmt.Sprint(cerr)
			}
			r.corruption(rule, detail)
		} else {
			want := fmt.Sprintf("exit %d", pr.n)
			r.predictK(k, pr.k)
			r.predictW(w, nil, tr.pid, cldExited, pr.w)
			r.normal("kqueue", p3NormK(*k, tr.pid), want, obs.s.end, kerr)
			r.normal("waitid", p3NormW(w, tr.pid), want, &we, werr)
			r.predict("C1: the two ends", "the same", same)
		}
		r.reap(tr, cmd, fmt.Sprintf("exit %d", pr.code))
	}
}

// p3Group: the leader exits while a member lives (T3's group rule). With
// fail, the planned fault: the first read of the members is made, and its
// answer replaced by an error.
func p3Group(fail bool) func(*p3Run) {
	return func(r *p3Run) {
		s := r.s
		rep := s.newReport(reportSpec{"pid": 1})
		cmd := helper("member-parent", []string{reportEnv + "=" + rep})
		release := gated(s, cmd)
		tr := startTree(s, cmd)
		members := s.awaitOwned(rep, 1, 5*time.Second, func(pid int) { tr.ownReported(pid) })
		st, pr := procInfo(members[0])
		s.needLoad(pr == present && st.pgid == tr.pid, "the member verified as the check's, in its group (pgid %d)", st.pgid)
		release()
		ok, status, err := tr.obs.wait(5 * time.Second)
		if !ok && err == nil {
			s.invalidf("load not established: the leader's exit was not observed within 5 s")
		}
		rd := p3Read{What: "kevent (the leader)", K: tr.obs.s.lastK}
		if err != nil {
			rd.Err = err.Error()
		}
		r.ev.R = append(r.ev.R, rd)
		r.answer("kqueue (the leader)", tr.obs.s.end, err)
		r.predict("the leader's exit (kqueue)", "exit 0", status)
		r.zombie(tr)
		live, err := tr.live(r.obsEnd())
		if err != nil {
			s.invalidf("observation failed: the members of group %d: %v", tr.pid, err)
		}
		s.worked(int64(len(live)))
		r.predict("live members besides the leader", fmt.Sprint(members), fmt.Sprint(live))
		resp := r.confirmEnd(tr, fail)
		r.ev.M = append(r.ev.M, "the group's end: "+resp)
		r.mark()
		if fail {
			if r.hits != 1 {
				s.flag("the planned fault hit %d times, want 1", r.hits)
			}
			r.predict("the candidate's answer when the members cannot be read", "not confirmed: the member list is unknown", resp)
		} else {
			r.predict("the candidate confirms the group's end", "confirmed", resp)
		}
		// Not confirmed, by the planned fault or by a read that failed: the
		// group is ended before the leader is reaped.
		if resp != "confirmed" && !tr.confirm(tKill) {
			s.flag("cleanup: the group %d could not be confirmed empty after the candidate did not confirm its end", tr.pid)
		}
		r.predict("the leader still unreaped after the group's end", true, tr.zombieLeader())
		if !tr.reap(time.Second) {
			s.invalidf("cleanup: the leader %d was not reaped within 1 s", tr.pid)
		}
		ws, _ := cmd.ProcessState.Sys().(syscall.WaitStatus)
		r.predict("status after reaping (wait4, Go)", "exit 0", waitStatus(ws))
	}
}

// confirmEnd is the candidate's confirmation of a group's end (§19.5's
// enumeration rule): a member list that cannot be read is unknown, and
// unknown is no end; only ENOMEM is retried, within the budget
// (kinfoProcs). Each read's own result is kept first, apart from what the
// candidate makes of it (Review 101 B1): R records it, and a real read
// that fails is a tool failure, whatever follows. With fail, the planned
// fault then replaces the candidate's input of the first read by an
// injected error, recorded as its hit; it never replaces, overwrites or
// excuses the read's own result.
func (r *p3Run) confirmEnd(tr *unixTree, fail bool) string {
	deadline := time.Now().Add(tKill)
	for n := 1; ; n++ {
		if n == 1 {
			p3Seam.enum = r.enumFault(tr.pid)
		}
		m, err := groupLive(tr.pid, deadline)
		p3Seam.enum = nil
		rd := p3Read{What: fmt.Sprintf("kern.proc.pgrp %d, read %d", tr.pid, n), Live: fmt.Sprint(m)}
		if err != nil {
			rd.Live, rd.Err = "", err.Error()
			r.s.flag("observation failed: the members of group %d: %v", tr.pid, err)
		}
		in := err // the candidate's input
		if fail && n == 1 {
			r.hits++
			r.ev.F = append(r.ev.F, fmt.Sprintf("planned fault: the read of kern.proc.pgrp %d was made (its result is in R); the candidate's input is replaced by an injected error", tr.pid))
			in = errors.New("injected: the member list could not be read")
		}
		end := ""
		switch {
		case in != nil:
			end = "not confirmed: the member list is unknown"
		case len(m) == 0:
			end = "confirmed"
		case time.Now().After(deadline):
			end = "not confirmed: members alive at the deadline"
		}
		// R keeps the first read, every read that failed, and the last.
		if n == 1 || err != nil || end != "" {
			r.ev.R = append(r.ev.R, rd)
		}
		if end != "" {
			return end
		}
		_ = tr.kill()
		time.Sleep(5 * time.Millisecond)
	}
}

// The layout probe (§19.12): the SDK's siginfo_t, struct kevent, struct
// kinfo_proc and constants, compiled and run natively, read strictly, and
// compared with what the Go side uses.

type probeKey struct {
	key, expr string
	bits      int
	signed    bool
	want      int64
	from      string // where the Go side's value comes from
}

func p3ProbeKeys() []probeKey {
	var ev unix.Kevent_t
	var kp unix.KinfoProc
	u := func(key, expr string, want uintptr, from string) probeKey {
		return probeKey{key, expr, 64, false, int64(want), from}
	}
	c := func(key string, bits int, signed bool, want int64, from string) probeKey {
		return probeKey{key, key, bits, signed, want, from}
	}
	keys := []probeKey{u("siginfo.size", "sizeof(siginfo_t)", 104, "spec §19.5")}
	for i, f := range []string{"si_signo", "si_errno", "si_code", "si_pid", "si_uid", "si_status"} {
		keys = append(keys,
			u("siginfo."+f+".off", "offsetof(siginfo_t, "+f+")", uintptr(wOff[i]), "the decoder"),
			u("siginfo."+f+".size", "sizeof(((siginfo_t *)0)->"+f+")", 4, "the decoder"))
	}
	kf := []struct {
		name      string
		off, size uintptr
	}{
		{"ident", unsafe.Offsetof(ev.Ident), unsafe.Sizeof(ev.Ident)},
		{"filter", unsafe.Offsetof(ev.Filter), unsafe.Sizeof(ev.Filter)},
		{"flags", unsafe.Offsetof(ev.Flags), unsafe.Sizeof(ev.Flags)},
		{"fflags", unsafe.Offsetof(ev.Fflags), unsafe.Sizeof(ev.Fflags)},
		{"data", unsafe.Offsetof(ev.Data), unsafe.Sizeof(ev.Data)},
		{"udata", unsafe.Offsetof(ev.Udata), unsafe.Sizeof(ev.Udata)},
	}
	keys = append(keys, u("kevent.size", "sizeof(struct kevent)", unsafe.Sizeof(ev), "x/sys"))
	for _, f := range kf {
		keys = append(keys,
			u("kevent."+f.name+".off", "offsetof(struct kevent, "+f.name+")", f.off, "x/sys"),
			u("kevent."+f.name+".size", "sizeof(((struct kevent *)0)->"+f.name+")", f.size, "x/sys"))
	}
	p, e := unsafe.Offsetof(kp.Proc), unsafe.Offsetof(kp.Eproc)
	keys = append(keys,
		u("kinfo_proc.size", "sizeof(struct kinfo_proc)", unsafe.Sizeof(kp), "x/sys"),
		u("kinfo_proc.p_pid.off", "offsetof(struct kinfo_proc, kp_proc.p_pid)", p+unsafe.Offsetof(kp.Proc.P_pid), "x/sys"),
		u("kinfo_proc.p_stat.off", "offsetof(struct kinfo_proc, kp_proc.p_stat)", p+unsafe.Offsetof(kp.Proc.P_stat), "x/sys"),
		u("kinfo_proc.p_flag.off", "offsetof(struct kinfo_proc, kp_proc.p_flag)", p+unsafe.Offsetof(kp.Proc.P_flag), "x/sys"),
		u("kinfo_proc.p_starttime.off", "offsetof(struct kinfo_proc, kp_proc.p_starttime)", p+unsafe.Offsetof(kp.Proc.P_starttime), "x/sys"),
		u("kinfo_proc.p_comm.off", "offsetof(struct kinfo_proc, kp_proc.p_comm)", p+unsafe.Offsetof(kp.Proc.P_comm), "x/sys"),
		u("kinfo_proc.e_ppid.off", "offsetof(struct kinfo_proc, kp_eproc.e_ppid)", e+unsafe.Offsetof(kp.Eproc.Ppid), "x/sys"),
		u("kinfo_proc.e_pgid.off", "offsetof(struct kinfo_proc, kp_eproc.e_pgid)", e+unsafe.Offsetof(kp.Eproc.Pgid), "x/sys"),
		u("kinfo_proc.e_ucred.cr_uid.off", "offsetof(struct kinfo_proc, kp_eproc.e_ucred.cr_uid)",
			e+unsafe.Offsetof(kp.Eproc.Ucred)+unsafe.Offsetof(kp.Eproc.Ucred.Uid), "x/sys"),
		c("SYS_waitid", 32, true, unix.SYS_WAITID, "x/sys"),
		c("P_PID", 32, true, pPID, "the prototype"),
		c("WEXITED", 32, true, unix.WEXITED, "x/sys"),
		c("WNOWAIT", 32, true, unix.WNOWAIT, "x/sys"),
		c("SIGCHLD", 32, true, int64(unix.SIGCHLD), "x/sys"),
		c("CLD_EXITED", 32, true, cldExited, "the prototype"),
		c("CLD_KILLED", 32, true, cldKilled, "the prototype"),
		c("CLD_DUMPED", 32, true, cldDumped, "the prototype"),
		c("EVFILT_PROC", 16, true, unix.EVFILT_PROC, "x/sys"),
		c("NOTE_EXIT", 32, false, unix.NOTE_EXIT, "x/sys"),
		c("NOTE_EXITSTATUS", 32, false, unix.NOTE_EXITSTATUS, "x/sys"),
		c("NOTE_PDATAMASK", 32, false, unix.NOTE_PDATAMASK, "x/sys"),
		c("EV_ADD", 16, false, unix.EV_ADD, "x/sys"),
		c("EV_ONESHOT", 16, false, unix.EV_ONESHOT, "x/sys"),
		c("SZOMB", 32, true, sZomb, "the prototype"),
		c("P_WEXIT", 32, true, pWexit, "the prototype"),
		c("KERN_PROC_ALL", 32, true, 0, "spec §19.5"),
		c("KERN_PROC_PID", 32, true, 1, "spec §19.5"),
		c("KERN_PROC_PGRP", 32, true, 2, "spec §19.5"),
		c("KERN_PROCARGS2", 32, true, 49, "spec §19.5"),
	)
	return keys
}

// p3ProbeSource is the probe: one "key=value" line per key, in order.
func p3ProbeSource(keys []probeKey) string {
	var b strings.Builder
	b.WriteString(`#include <sys/types.h>
#include <sys/proc.h>
#include <sys/sysctl.h>
#include <sys/event.h>
#include <sys/wait.h>
#include <sys/syscall.h>
#include <signal.h>
#include <stddef.h>
#include <stdio.h>

int main(void) {
`)
	for _, k := range keys {
		if k.signed {
			fmt.Fprintf(&b, "\tprintf(\"%s=%%lld\\n\", (long long)(%s));\n", k.key, k.expr)
		} else {
			fmt.Fprintf(&b, "\tprintf(\"%s=%%llu\\n\", (unsigned long long)(%s));\n", k.key, k.expr)
		}
	}
	b.WriteString("\treturn fflush(stdout) != 0 || ferror(stdout) ? 2 : 0;\n}\n")
	return b.String()
}

// parseProbe reads the probe's output strictly: every line "key=value",
// every key of the set exactly once, nothing else, the last line whole; a
// value in canonical decimal within its key's sign and width.
func parseProbe(b []byte, keys []probeKey) (map[string]int64, error) {
	switch {
	case len(b) > 1<<16:
		return nil, fmt.Errorf("%d bytes, more than 64 KiB", len(b))
	case len(b) == 0 || b[len(b)-1] != '\n':
		return nil, errors.New("the last line is cut (no newline at the end)")
	}
	spec := map[string]probeKey{}
	for _, k := range keys {
		spec[k.key] = k
	}
	got := map[string]int64{}
	for i, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		k, v, ok := strings.Cut(line, "=")
		pk, known := spec[k]
		switch {
		case !ok:
			return nil, fmt.Errorf("line %d is not key=value: %q", i+1, clip(line))
		case !known:
			return nil, fmt.Errorf("line %d: unknown key %q", i+1, clip(k))
		}
		if _, dup := got[k]; dup {
			return nil, fmt.Errorf("line %d: the key %s again", i+1, k)
		}
		if pk.signed {
			n, err := strconv.ParseInt(v, 10, pk.bits)
			if err != nil || strconv.FormatInt(n, 10) != v {
				return nil, fmt.Errorf("%s=%q is not a %d-bit signed decimal", k, clip(v), pk.bits)
			}
			got[k] = n
			continue
		}
		n, err := strconv.ParseUint(v, 10, pk.bits)
		if err != nil || strconv.FormatUint(n, 10) != v || n > math.MaxInt64 {
			return nil, fmt.Errorf("%s=%q is not a %d-bit unsigned decimal", k, clip(v), pk.bits)
		}
		got[k] = int64(n)
	}
	for _, k := range keys {
		if _, ok := got[k.key]; !ok {
			return nil, fmt.Errorf("the key %s is missing", k.key)
		}
	}
	return got, nil
}

// p3Layout runs the probe with the runner's Xcode. Only a missing xcrun
// reliably shows that there is none (unsupported); any failure of the
// tools is a tool failure (docs/wp2.5-prototype.md §15.3).
func p3Layout(r *p3Run) {
	s := r.s
	xcrun, err := exec.LookPath("xcrun")
	if err != nil {
		r.ev.E["xcode"] = "xcrun not on PATH"
		s.unsupported("the layout probe: xcrun is not on PATH: no Xcode here")
	}
	r.toolSetup()
	clang := r.tool("xcrun-find-clang", 10*time.Second, xcrun, "--find", "clang")
	sdk := r.tool("xcrun-sdk-path", 10*time.Second, xcrun, "--sdk", "macosx", "--show-sdk-path")
	r.ev.E["sdk.version"] = r.tool("xcrun-sdk-version", 10*time.Second, xcrun, "--sdk", "macosx", "--show-sdk-version")
	r.ev.E["clang"], r.ev.E["sdk.path"] = clang, sdk
	if !filepath.IsAbs(clang) || !filepath.IsAbs(sdk) {
		s.invalidf("xcrun gave no absolute paths: %q, %q", clip(clang), clip(sdk))
	}
	keys := p3ProbeKeys()
	src, bin := filepath.Join(r.dir, "probe.c"), filepath.Join(r.dir, "probe")
	if err := os.WriteFile(src, []byte(p3ProbeSource(keys)), 0o600); err != nil {
		s.invalidf("the probe's source: %v", err)
	}
	arch := map[string]string{"arm64": "arm64", "amd64": "x86_64"}[runtime.GOARCH]
	r.toolRaw("compile", 30*time.Second, clang, "-arch", arch, "-isysroot", sdk, "-o", bin, src)
	out := r.toolRaw("probe", 5*time.Second, bin)
	r.mark()
	got, err := parseProbe(out, keys)
	if err != nil {
		r.ev.R = append(r.ev.R, p3Read{What: "the probe's output", Err: err.Error()})
		s.invalidf("the probe's output: %v", err)
	}
	r.ev.Layout = got
	s.worked(int64(len(got)))
	for _, k := range keys {
		s.expect(k.key+" ("+k.from+")", k.want, got[k.key])
	}
	s.expectTrue("siginfo_t fits the 128-byte buffer (capacity only)", got["siginfo.size"] <= siSize, strconv.FormatInt(got["siginfo.size"], 10))
}

// toolSetup makes the layout probe's directory and its tools' tag.
func (r *p3Run) toolSetup() {
	r.dir = r.t.TempDir()
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		r.s.invalidf("the tools' tag: %v", err)
	}
	r.tag = p3ToolEnv + "=" + hex.EncodeToString(b[:])
	r.since = fmt.Sprintf("%d.%06d", time.Now().Unix()-1, 0)
}

// tool is toolRaw's output as one trimmed, non-empty line.
func (r *p3Run) tool(step string, limit time.Duration, name string, args ...string) string {
	out := strings.TrimSpace(string(r.toolRaw(step, limit, name, args...)))
	if out == "" || strings.Contains(out, "\n") {
		r.s.invalidf("the tool step %s gave not one line: %q", step, clip(out))
	}
	return out
}

// toolRaw runs one tool step of the layout probe (§19.12): started through
// start and owned by the scenario, its output to files, within min(its
// limit, the observation's rest, go test's deadline less a minute). A
// census of this run's tag after it must find nothing alive. A step that
// fails stops the run as invalid.
func (r *p3Run) toolRaw(step string, limit time.Duration, name string, args ...string) []byte {
	s := r.s
	// One deadline for the whole step, fixed before it begins (Review 101
	// B3): the files, the start and the wait all come out of it.
	began := time.Now()
	end := began.Add(limit)
	if e := r.obsEnd(); e.Before(end) {
		end = e
	}
	if !began.Before(end) {
		s.invalidf("not executed: no time left for the tool step %s", step)
	}
	// Each file is the scenario's from the moment it exists (Review 101
	// B2). The parent closes its copies once the child has them; a file
	// made before a failure is closed at once; a close that fails is a tool
	// failure, kept.
	outPath, errPath := filepath.Join(r.dir, step+".out"), filepath.Join(r.dir, step+".err")
	stdout, err := os.Create(outPath)
	if err != nil {
		s.invalidf("the output files of the tool step %s: %v", step, err)
	}
	closeStdout := s.closer("the output file "+filepath.Base(outPath), func() error { return closeOut(stdout, step) })
	stderr, err := createErr(errPath)
	if err != nil {
		if closeStdout() == nil {
			s.note("the output file %s, created before the failure, closed", filepath.Base(outPath))
		}
		s.invalidf("the output files of the tool step %s: %v", step, err)
	}
	closeStderr := s.closer("the output file "+filepath.Base(errPath), stderr.Close)
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), r.tag)
	cmd.Dir = r.dir
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err = start(cmd)
	closeStdout()
	closeStderr()
	if err != nil {
		s.invalidf("the tool step %s did not start: %v", step, err)
	}
	s.launched(1)
	c := childOf(cmd)
	s.own(c)
	ended := c.reap(time.Until(end))
	if !ended {
		c.stop()
		s.flag("the tool step %s did not end within %v", step, end.Sub(began).Round(time.Millisecond))
	}
	r.census(step)
	if !ended {
		s.t.FailNow()
	}
	if ws, _ := cmd.ProcessState.Sys().(syscall.WaitStatus); !ws.Exited() || ws.ExitStatus() != 0 {
		eb, _ := r.readBack(errPath)
		s.invalidf("the tool step %s ended with %s: %s", step, waitStatus(ws), clip(strings.TrimSpace(string(eb))))
	}
	b, err := r.readBack(outPath)
	if err != nil {
		s.invalidf("the output of the tool step %s: %v", step, err)
	}
	return b
}

// readBack reads a tool step's file. Its handle is the scenario's while it
// is open (Review 101 B2): os.ReadFile would drop the close's error.
func (r *p3Run) readBack(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	closeF := r.s.closer("the file "+filepath.Base(path)+" read back", f.Close)
	b, err := io.ReadAll(f)
	closeF()
	return b, err
}

// census lists this run's tool processes still alive by their tag: Clang's
// own descendants are not started through start and carry only the tag.
// One found, or a census that fails, is a tool failure; the ones found are
// ended only while their identity and tag still hold, which does not make
// the run valid again.
func (r *p3Run) census(step string) {
	ss, err := censusTag(r.tag, r.since, r.obsEnd())
	if err != nil {
		r.s.flag("cleanup not verified: the census of the tools after %s: %v", step, err)
		return
	}
	if len(ss) == 0 {
		return
	}
	var keys []string
	for _, v := range ss {
		keys = append(keys, v.key())
	}
	r.s.flag("cleanup: tool processes alive after %s: %v", step, keys)
	for _, v := range ss {
		if st, pr := procInfo(v.pid); pr == present && st.start == v.start && !st.zombie && tagged(v.pid, r.tag) {
			_ = syscall.Kill(v.pid, syscall.SIGKILL)
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		var still []string
		for _, v := range ss {
			a, err := recordAlive(witnessRec{pid: v.pid, start: v.start})
			if err != nil {
				r.s.flag("cleanup not verified: the tool process %s: %v", v.key(), err)
				return
			}
			if a {
				still = append(still, v.key())
			}
		}
		switch {
		case len(still) == 0:
			r.s.note("remedy: the tool processes %v ended", keys)
			return
		case time.Now().After(deadline):
			r.s.flag("cleanup not verified: tool processes still alive: %v", still)
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}
