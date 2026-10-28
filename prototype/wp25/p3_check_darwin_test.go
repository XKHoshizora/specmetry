//go:build darwin

package wp25

// Direct checks of P-3's pure parts (docs/wp2.5-prototype.md §15.6): the
// decoder (M) and the test's own normalization (N) on §19.5's boundary
// values and on each gate regression's corruption, and the strictness of
// the probe's parser. They count no scenario and are no native evidence.

import (
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestP3Decode(t *testing.T) {
	const pid = 4242
	w := func(signo, code, status uint32) wRaw { return wRaw{Signo: signo, Code: code, Pid: pid, Status: status} }
	for _, c := range []struct {
		r    wRaw
		m, n string // the decoder's end or rule; N
	}{
		// §19.5's derivation, P1 to P4 holding.
		{w(20, 1, 0x7), "exit 7", "exit 7"},
		{w(20, 1, 0x100), "exit 0", "exit 0"},
		{w(20, 1, 0x107), "exit 7", "exit 7"},
		{w(20, 1, 0x00ffffff), "exit 255", "exit 255"},
		{w(20, 1, 0x0), "exit 0", "exit 0"},
		// The top byte kept (P2 not holding): W3a.
		{w(20, 1, 0xffffffff), "W3a", "none"},
		{w(20, 1, 0x7f000007), "W3a", "none"},
		{w(20, 2, 9), "signal 9", "signal 9"},
		{w(20, 3, 6), "signal 6 core", "signal 6 core"},
		{w(20, 2, 0), "W4", "none"},
		{w(20, 2, 32), "W4", "none"},
		{w(20, 2, 0x109), "W4", "none"},
		// The gate regressions' corruptions of w.exit7.
		{wRaw{Signo: 20, Code: 1, Pid: pid + 1, Status: 7}, "W1", "none"},
		{w(0, 1, 7), "W1", "none"},
		{w(20, 4, 7), "W1", "none"},
		{w(20, 1, 0x01000007), "W3a", "none"},
		{w(20, 1, 0xff0000ff), "W3a", "none"},
	} {
		e, err := decodeW(c.r, pid)
		got := ruleOf(err)
		if err == nil {
			got = e.String()
		}
		if got != c.m {
			t.Errorf("decodeW(%+v) = %s, want %s", c.r, got, c.m)
		}
		if n := p3NormW(c.r, pid); n != c.n {
			t.Errorf("p3NormW(%+v) = %s, want %s", c.r, n, c.n)
		}
	}
	a := w(20, 1, 7)
	b := a
	b.Status = 8 // p3.w-reread-once
	if _, err := decodeW2(a, b, pid); ruleOf(err) != "W5" {
		t.Errorf("decodeW2 of two different reads: %v, want W5", err)
	}
	if e, err := decodeW2(a, a, pid); err != nil || e.String() != "exit 7" {
		t.Errorf("decodeW2 of the same read twice: %v, %v", e, err)
	}
	// The first read decides: a corrupted first read is W1, not W5.
	c := a
	c.Pid = pid + 1
	if _, err := decodeW2(c, a, pid); ruleOf(err) != "W1" {
		t.Errorf("decodeW2 with a corrupted first read: %v, want W1", err)
	}

	const both = unix.NOTE_EXIT | unix.NOTE_EXITSTATUS
	k := func(fflags uint32, data int64) kRaw {
		return kRaw{Ident: pid, Filter: unix.EVFILT_PROC, Fflags: fflags, Data: data}
	}
	for _, c := range []struct {
		r    kRaw
		m, n string
	}{
		{k(both, 0x0700), "exit 7", "exit 7"},
		{k(both, 0x0000), "exit 0", "exit 0"},
		{k(both, 0xff00), "exit 255", "exit 255"},
		{k(both, 0x0009), "signal 9", "signal 9"},
		{k(both, 0x0086), "signal 6 core", "signal 6 core"},
		{k(both, 0x0109), "K2", "none"},
		{k(both, 0x0080), "K2", "none"},
		{k(both, -1), "K1", "none"},
		// The gate regressions' corruptions of k.exit7.
		{k(unix.NOTE_EXIT, 0x0700), "K1", "none"},
		{k(both, 0x10000), "K1", "none"},
		{k(both, 0x117f), "K2", "none"},
		{k(both, 0x0800), "exit 8", "exit 8"}, // p3.c-mismatch-once's K
	} {
		e, err := decodeK(c.r, pid)
		got := ruleOf(err)
		if err == nil {
			got = e.String()
		}
		if got != c.m {
			t.Errorf("decodeK(%+v) = %s, want %s", c.r, got, c.m)
		}
		if n := p3NormK(c.r, pid); n != c.n {
			t.Errorf("p3NormK(%+v) = %s, want %s", c.r, n, c.n)
		}
	}
	// An event of another process is an error of the call, not a rule.
	other := k(both, 0x0700)
	other.Ident = pid + 1
	if _, err := decodeK(other, pid); err == nil || ruleOf(err) != "" {
		t.Errorf("decodeK of another process's event: %v, want an error of no rule", err)
	}
	if n := p3NormK(other, pid); n != "none" {
		t.Errorf("p3NormK of another process's event: %s", n)
	}
	if err := sameEnd(exitInfo{"exit", 8}, exitInfo{"exit", 7}); ruleOf(err) != "C1" {
		t.Errorf("sameEnd of two ends: %v, want C1", err)
	}
	if err := sameEnd(exitInfo{"exit", 7}, exitInfo{"exit", 7}); err != nil {
		t.Errorf("sameEnd of one end: %v", err)
	}
}

func TestP3Parse(t *testing.T) {
	keys := []probeKey{{key: "a", bits: 64}, {key: "b", bits: 16, signed: true}, {key: "c", bits: 32}}
	got, err := parseProbe([]byte("a=1\nb=-5\nc=4294967295\n"), keys)
	if err != nil || got["a"] != 1 || got["b"] != -5 || got["c"] != 4294967295 {
		t.Errorf("a whole output: %v, %v", got, err)
	}
	for _, bad := range []string{
		"",
		"a=1\nb=-5\nc=1",              // the last line cut
		"a=1\nb=-5\nc=1\na=1\n",       // a key twice, the same value
		"a=1\nb=-5\nc=1\nd=1\n",       // an unknown key
		"a=1\nb=-5\n",                 // a key missing
		"a=1\nb=-5\nc=1\nnot a key\n", // a line that is not key=value
		"a=1\nb=-5\nc=1\n\n",          // an empty line
		"a=01\nb=-5\nc=1\n",           // not canonical
		"a=+1\nb=-5\nc=1\n",
		"a=1\nb=-05\nc=1\n",
		"a=1\nb=+5\nc=1\n",
		"a=1\nb=-32769\nc=1\n",      // beyond 16 signed bits
		"a=1\nb=-5\nc=4294967296\n", // beyond 32 unsigned bits
		"a=-1\nb=-5\nc=1\n",         // a negative unsigned value
		"a=1\nb=x\nc=1\n",
	} {
		if _, err := parseProbe([]byte(bad), keys); err == nil {
			t.Errorf("parseProbe(%q) accepted", bad)
		}
	}
	// The real key set: each key once, and one line of the probe for each.
	all := p3ProbeKeys()
	seen := map[string]bool{}
	for _, k := range all {
		if seen[k.key] {
			t.Errorf("the key %s twice", k.key)
		}
		seen[k.key] = true
	}
	if n := strings.Count(p3ProbeSource(all), "printf("); n != len(all) {
		t.Errorf("the probe prints %d lines for %d keys", n, len(all))
	}
}
