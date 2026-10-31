package checkrun

import (
	"fmt"
	"slices"
	"sync"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// Why a host was disabled (spec §19.1): the first reason is kept, for the
// message only; it is not a field or an enumeration of any contract.
const (
	// disabledO3: an output copy did not end (C-4).
	disabledO3 = "o3"
	// disabledStopPending: a stop task was not confirmed ended when J
	// ended (D21).
	disabledStopPending = "stop_pending"
	// disabledUnconfirmed: a consumer of the run was created and not
	// confirmed ended in the termination window (Q-1: F1, A5, A6).
	disabledUnconfirmed = "unconfirmed_end"
)

// heldKind is what an item of the resource list holds (spec §19.1).
type heldKind string

const (
	heldCopy    heldKind = "copy"
	heldStop    heldKind = "stop"
	heldProcess heldKind = "process"
)

// held is one item of the resource list: its kind, the call and criterion
// it came from (for diagnosis only) and what it holds, which the list keeps
// a strong reference to until it is removed.
type held struct {
	kind      heldKind
	call      int
	criterion string
	resource  any
}

// host is the check-run host of a process (spec §19.1; C-4, D21, Q-1). mu
// guards its state: whether it is disabled, and why first; the calls
// admitted and not returned (active); and the resource list. Once
// disabled it stays disabled until the process ends: removing items, or
// an empty list, never undoes it, and nothing resets it.
type host struct {
	mu       sync.Mutex
	disabled string
	active   int
	calls    int
	held     []*held
}

// process is this process's host, the only one production code uses.
var process host

// admit is rule 1: under mu, a disabled host refuses the call before any
// database, git or process is reached (spec §4.1 step 2); otherwise the
// call is admitted and counted in the same critical section, so no
// disabling falls between the check and the count. It returns the call's
// number.
func (h *host) admit() (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.disabled != "" {
		return 0, apperr.Newf(apperr.Environment,
			"this process runs no more checks: %s; it still holds %s of earlier check runs; no check was run",
			disabledCause(h.disabled), plural(len(h.held), "resource"))
	}
	h.active++
	h.calls++
	return h.calls, nil
}

// release ends an admitted call.
func (h *host) release() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.active--
}

// recheck is rule 3: before each criterion starts, an admitted call reads
// under mu whether the host was disabled meanwhile, and stops if it was;
// it is not interrupted otherwise.
func (h *host) recheck() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.disabled != "" {
		return apperr.Newf(apperr.Environment,
			"this process was disabled during this call: %s; the next criterion was not started", disabledCause(h.disabled))
	}
	return nil
}

// disable is rule 2 at the hand-over point: in one critical section the
// items left unended (copy and stop tasks, or a process not confirmed
// ended) join the resource list and the host is disabled for reason; the
// first reason is kept. A later batch calls it from the execution; until
// then only tests do.
func (h *host) disable(reason string, items ...*held) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.held = append(h.held, items...)
	if h.disabled == "" {
		h.disabled = reason
	}
}

// remove takes an ended item off the resource list (rule 4). Whether the
// host is disabled does not change.
func (h *host) remove(item *held) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	i := slices.Index(h.held, item)
	if i < 0 {
		return false
	}
	h.held = slices.Delete(h.held, i, i+1)
	return true
}

func disabledCause(reason string) string {
	switch reason {
	case disabledO3:
		return "an output copy of an earlier check run in this process did not end"
	case disabledStopPending:
		return "the stop of an earlier check run's output copy in this process was not confirmed to end"
	}
	return "the processes of an earlier check run in this process were not confirmed to have ended"
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
