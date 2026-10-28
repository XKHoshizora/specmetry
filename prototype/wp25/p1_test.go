package wp25

// P-1 (spec v7 §19.12, G3 and Q-1; plan fixed in docs/wp2.5-prototype.md
// §14.3): F1 with the main process still alive, or exited but its exit
// not observed, on the three platforms. The inner call is the model's
// F1 path; the outer, after it returns and before any remediation,
// confirms the target branch on its own, keeps the inner state, checks
// the host's admission, and only then remediates. A remediation that
// succeeds does not restore an inner finding.

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

// p1Common is what every platform's inner call reports.
type p1Common struct {
	Class    string   `json:"class"` // "F1" or "normal"
	Status   string   `json:"status,omitempty"`
	ObsHits  int      `json:"observe_fault_hits"`
	KillHits int      `json:"terminate_fault_hits"`
	PID      int      `json:"pid"`
	Facts    []string `json:"facts,omitempty"` // fixture events and constructions
	cmd      *exec.Cmd
}

func (c *p1Common) common() *p1Common { return c }

// p1Run is one platform's run of the inner call, and the outer checks
// that need the platform.
type p1Run interface {
	common() *p1Common
	// closed checks, first thing after the inner call, that the E-main
	// observer's descriptor is closed (Unix); "" where there is none.
	closed(s *scen) string
	// branch is the outer's own confirmation that the target branch was
	// hit, with its own reads, not the inner call's objects.
	branch(s *scen, variant string) (bool, string)
	// item says whether the inventory item is this run's.
	item(it invItem) (bool, string)
	// extra records the platform's own inner expectations.
	extra(s *scen, variant string)
	// remedy is the outer's remediation.
	remedy(s *scen, variant string) error
	// release is the host's exit for one inventory item.
	release(it invItem) error
}

var p1Budget = &budget{name: "P-1", total: 3 * time.Minute, per: time.Minute}

func TestP1(t *testing.T) {
	for _, v := range []string{"alive", "exited", "healthy"} {
		for n := 1; n <= 2; n++ {
			t.Run(fmt.Sprintf("%s.%d", v, n), func(t *testing.T) { runP1(t, v, n) })
		}
	}
}

func admission(a access, err error) string {
	switch {
	case errors.Is(err, errHostDisabled):
		return fmt.Sprintf("refused (%d db, %d git, %d processes)", a.db, a.git, a.procs)
	case err != nil:
		return "error: " + err.Error()
	}
	return fmt.Sprintf("admitted (%d db, %d git, %d processes)", a.db, a.git, a.procs)
}

func runP1(t *testing.T, variant string, n int) {
	id := fmt.Sprintf("P1.%s.%d", variant, n)
	k := fault
	if variant == "healthy" {
		k = normal
	}
	s := begin(t, "G3", id, k, true, true)
	defer s.done()
	if ok, left := p1Budget.allow(); !ok {
		s.invalidf("not executed: the total deadline of P-1 has %v left, less than one run's %v", left.Round(time.Second), p1Budget.per)
	}
	f1 := variant != "healthy"
	h := &host{}
	if err := h.admit(); err != nil {
		s.invalidf("a fresh host refused: %v", err)
	}
	base := maxGoroutine()
	began := time.Now()
	r := p1Inner(s, h, variant)
	h.leave()
	c := r.common()
	if d := time.Since(began); d > 30*time.Second {
		s.flag("the inner call took %v, over its 30 s", d)
	}

	// The outer, before any remediation.
	ev := time.Now()
	fdClosed := r.closed(s)
	var gs []gstack
	var waiting []uint64
	var fresh []string
	for i := 0; i < 5; i++ {
		// A goroutine that has just returned from its work may still be
		// on its way out; one blocked in a wait never leaves.
		gs = goroutines()
		waiting, fresh = waitingGoroutines(gs), freshProto(gs, base, goid())
		if len(waiting) == 0 && len(fresh) == 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	disabled, reason, items := h.state()
	hit, branch := r.branch(s, variant)
	acc, aerr := h.request(func(a *access) { a.db, a.git, a.procs = 1, 1, 1 })
	refusedByBarrier, barrier := p1Barrier(s)
	itemOK, itemDetail := false, fmt.Sprintf("%d items", len(items))
	if len(items) == 1 {
		itemOK, itemDetail = r.item(items[0])
	}
	if d := time.Since(ev); d > 10*time.Second {
		s.flag("the evidence took %v, over its 10 s", d)
	}
	pline(id, map[string]any{
		"inner": c, "inventory": len(items), "item": itemDetail, "disabled": disabled, "reason": reason,
		"waiting": waiting, "fresh": fresh, "observer": fdClosed, "branch": branch, "hit": hit,
		"admission": admission(acc, aerr), "barrier": barrier,
	})
	s.note("inner: class %s, status %q, observe faults %d, terminate faults %d", c.Class, c.Status, c.ObsHits, c.KillHits)
	for _, f := range c.Facts {
		s.note("%s", f)
	}
	s.note("branch: %s", branch)
	s.note("inventory: %s", itemDetail)
	s.note("barrier: %s", barrier)
	s.needLoad(hit, "the target branch (%s) was hit", variant)

	if f1 {
		s.expect("class", "F1", c.Class)
		s.expect("observation fault hits", 1, c.ObsHits)
		s.expectTrue("inventory: one unconfirmed process, this run's", len(items) == 1 && itemOK, itemDetail)
		s.expect("host disabled", "true unconfirmed_end", fmt.Sprint(disabled, " ", reason))
		s.expect("next check", "refused (0 db, 0 git, 0 processes)", admission(acc, aerr))
	} else {
		s.expect("class", "normal", c.Class)
		s.expect("status", "exit 0", c.Status)
		s.expect("observation fault hits", 0, c.ObsHits)
		s.expect("inventory items", 0, len(items))
		s.expect("host disabled", false, disabled)
		s.expect("next check", "admitted (1 db, 1 git, 1 processes)", admission(acc, aerr))
	}
	if variant == "alive" {
		s.expectTrue("terminate requests swallowed", c.KillHits >= 1, fmt.Sprint(c.KillHits))
	} else {
		s.expect("terminate fault hits", 0, c.KillHits)
	}
	s.expectTrue("no goroutine waits for a process", len(waiting) == 0, fmt.Sprint(waiting))
	s.expectTrue("no new prototype goroutine", len(fresh) == 0, fmt.Sprint(fresh))
	if runtime.GOOS == "linux" {
		s.expect("Linux baseline barrier refuses the next interval", f1, refusedByBarrier)
	}
	r.extra(s, variant)

	// The remediation, after the evidence is kept.
	rem := time.Now()
	rerr := r.remedy(s, variant)
	acc2, aerr2 := h.request(func(a *access) { a.db, a.git, a.procs = 1, 1, 1 })
	var relErrs []string
	released := h.release()
	for _, it := range released {
		if err := r.release(it); err != nil {
			relErrs = append(relErrs, err.Error())
		}
	}
	remedy := "not needed"
	switch {
	case rerr != nil:
		remedy = "failed: " + rerr.Error()
		s.flag("remediation failed: %v", rerr)
	case f1:
		remedy = "succeeded"
	}
	s.note("remedy: %s; %d inventory items released", remedy, len(released))
	if len(relErrs) > 0 {
		s.flag("the inventory could not be released: %v", relErrs)
	}
	if f1 {
		s.expect("next check after the end and the release", "refused (0 db, 0 git, 0 processes)", admission(acc2, aerr2))
	}
	if d := time.Since(rem); d > 20*time.Second {
		s.flag("the remediation took %v, over its 20 s", d)
	}
	s.worked(1)
}
