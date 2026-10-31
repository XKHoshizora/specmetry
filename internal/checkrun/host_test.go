package checkrun_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/checkrun"
)

// WP2.5 spec §3, §4.1 step 1, D16: criteria or --all, not both and not
// neither; --isolation off or required, and auto is a command line error.
// required passes step 1 (step 2 refuses it). Nothing given is quoted.
func TestOptionsCheck(t *testing.T) {
	const canary = "CANARY-AC-0e7b"
	cases := []struct {
		name    string
		opts    checkrun.Options
		refused bool
		pointer string
	}{
		{"criteria", checkrun.Options{Criteria: []string{"AC-001-1"}, Isolation: "off"}, false, ""},
		{"all", checkrun.Options{All: true, Isolation: "off"}, false, ""},
		{"required", checkrun.Options{All: true, Isolation: "required"}, false, ""},
		{"both", checkrun.Options{Criteria: []string{canary}, All: true, Isolation: "off"}, true, ""},
		{"neither", checkrun.Options{Isolation: "off"}, true, ""},
		{"auto", checkrun.Options{All: true, Isolation: "auto"}, true, "--isolation"},
		{"another value", checkrun.Options{All: true, Isolation: canary}, true, "--isolation"},
		{"no value", checkrun.Options{All: true}, true, "--isolation"},
	}
	for _, c := range cases {
		err := checkrun.CheckOptions(c.opts)
		if !c.refused {
			if err != nil {
				t.Errorf("%s: %v", c.name, err)
			}
			continue
		}
		e := apperr.From(err)
		if e == nil || apperr.ExitCode(err) != apperr.ExitInvalidInput {
			t.Errorf("%s: %v; want exit 2", c.name, err)
			continue
		}
		var pointers []string
		for _, d := range e.Details {
			pointers = append(pointers, d.Pointer)
		}
		if strings.Join(pointers, ",") != c.pointer {
			t.Errorf("%s: details at %v, want %q", c.name, pointers, c.pointer)
		}
		if strings.Contains(envelope(t, err), canary) {
			t.Errorf("%s: the error quotes the command line", c.name)
		}
	}
}

// WP2.5 spec §19.1 rules 1 and 2, §8.6 E2, E3: a host disabled before a
// call refuses it (exit 6) without counting it; the items it holds can end
// and leave the list, and an empty list still refuses. A host never
// disabled admits and counts each call until it returns.
func TestHostRefusesOnceDisabled(t *testing.T) {
	h := checkrun.NewHost()
	call, err := checkrun.Admit(h)
	if err != nil || call != 1 {
		t.Fatalf("Admit = %d, %v", call, err)
	}
	if _, active, _ := checkrun.State(h); active != 1 {
		t.Fatalf("active %d after one call", active)
	}
	checkrun.Release(h)

	items := checkrun.Disable(h, checkrun.DisabledO3, "copy", "stop")
	refused := func(wantHeld string) {
		t.Helper()
		_, err := checkrun.Admit(h)
		if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "an output copy") ||
			!strings.Contains(err.Error(), wantHeld) {
			t.Fatalf("Admit on a disabled host = %v", err)
		}
		if disabled, active, _ := checkrun.State(h); disabled != checkrun.DisabledO3 || active != 0 {
			t.Fatalf("a refused call changed the host: %q, active %d", disabled, active)
		}
	}
	refused("2 resources")
	for _, it := range items {
		if !checkrun.Remove(h, it) {
			t.Fatal("an item held is not in the list")
		}
	}
	if _, _, held := checkrun.State(h); held != 0 {
		t.Fatalf("%d items held after their removal", held)
	}
	refused("0 resources")
}

// WP2.5 spec §19.1 rule 3: a call admitted before the host is disabled is
// not interrupted, but the next criterion's recheck stops it; a new call is
// refused. The first reason is kept.
func TestHostRecheck(t *testing.T) {
	h := checkrun.NewHost()
	if _, err := checkrun.Admit(h); err != nil {
		t.Fatal(err)
	}
	if err := checkrun.Recheck(h); err != nil {
		t.Fatalf("Recheck before any disabling: %v", err)
	}
	checkrun.Disable(h, checkrun.DisabledStopPending, "stop")
	checkrun.Disable(h, checkrun.DisabledUnconfirmed, "process")
	err := checkrun.Recheck(h)
	if apperr.ExitCode(err) != apperr.ExitEnvironment || !strings.Contains(err.Error(), "during this call") ||
		!strings.Contains(err.Error(), "stop of an earlier check run") {
		t.Fatalf("Recheck after disabling = %v", err)
	}
	if _, err := checkrun.Admit(h); apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("a new call = %v", err)
	}
	checkrun.Release(h)
	if disabled, active, held := checkrun.State(h); disabled != checkrun.DisabledStopPending || active != 0 || held != 2 {
		t.Fatalf("host: %q, active %d, held %d", disabled, active, held)
	}
}

// WP2.5 spec §19.1 rule 1: admission and disabling are one lock's critical
// sections, so a call racing the disabling is either refused, and never
// counted, or admitted and counted, and then stopped by its recheck (run
// with -race in CI).
func TestHostAdmissionRacesDisabling(t *testing.T) {
	for round := 0; round < 50; round++ {
		h := checkrun.NewHost()
		const calls = 16
		var wg sync.WaitGroup
		admitted := make([]bool, calls)
		start := make(chan struct{})
		for i := range calls {
			wg.Go(func() {
				<-start
				_, err := checkrun.Admit(h)
				admitted[i] = err == nil
				if err != nil && apperr.ExitCode(err) != apperr.ExitEnvironment {
					t.Errorf("Admit = %v", err)
				}
			})
		}
		wg.Go(func() {
			<-start
			checkrun.Disable(h, checkrun.DisabledUnconfirmed, "process")
		})
		close(start)
		wg.Wait()
		n := 0
		for _, ok := range admitted {
			if ok {
				n++
			}
		}
		if _, active, _ := checkrun.State(h); active != n {
			t.Fatalf("round %d: %d calls admitted, %d counted", round, n, active)
		}
		for range n {
			if err := checkrun.Recheck(h); apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Fatalf("round %d: an admitted call's recheck after the disabling = %v", round, err)
			}
			checkrun.Release(h)
		}
		if _, err := checkrun.Admit(h); err == nil {
			t.Fatalf("round %d: a call after the disabling was admitted", round)
		}
	}
}
