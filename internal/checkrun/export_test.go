package checkrun

import (
	"context"
	"testing"
)

// Test access to the package's unexported parts. None of it is in the
// production build.

// Host is a check-run host of a test's own (spec §19.1), never the
// process's: each test starts from a host that was never disabled.
type Host = host

func NewHost() *Host { return &host{} }

// The reasons a host is disabled for.
const (
	DisabledO3          = disabledO3
	DisabledStopPending = disabledStopPending
	DisabledUnconfirmed = disabledUnconfirmed
)

// Held is an item of a host's resource list.
type Held = held

func CheckOptions(o Options) error { return o.check() }

func Admit(h *Host) (int, error) { return h.admit() }
func Release(h *Host)            { h.release() }
func Recheck(h *Host) error      { return h.recheck() }

// Disable is the test seam that disables a host (rule 2): in a later batch
// the execution's hand-over point calls disable; here only tests do. Each
// kind is one item held.
func Disable(h *Host, reason string, kinds ...string) []*Held {
	var items []*held
	for _, k := range kinds {
		kind := map[string]heldKind{"copy": heldCopy, "stop": heldStop, "process": heldProcess}[k]
		items = append(items, &held{kind: kind, criterion: "AC-001-1"})
	}
	h.disable(reason, items...)
	return items
}

func Remove(h *Host, item *Held) bool { return h.remove(item) }

// State is what a host's lock guards: why it was disabled, the calls
// admitted and not returned, and the items held.
func State(h *Host) (disabled string, active, held int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.disabled, h.active, len(h.held)
}

// PrepareWith is Prepare with a test's own host.
func PrepareWith(ctx context.Context, h *Host, opts Options, out Output) (*Batch, error) {
	return prepare(ctx, h, opts, out)
}

// OnOpen counts where Prepare opens the database (the counting seam of
// spec §19.1's "no database access").
func OnOpen(t testing.TB, f func()) {
	testHookOpen = f
	t.Cleanup(func() { testHookOpen = nil })
}

// AfterRead runs f once Prepare's read transaction has ended.
func AfterRead(t testing.TB, f func()) {
	testHookAfterRead = f
	t.Cleanup(func() { testHookAfterRead = nil })
}

// Finder looks interpreters up as step 7 does, with the worktrees, PATH
// and exec path a test gives.
type Finder = finder

func NewFinder(worktrees []string, path string, execPath func(context.Context) (string, error)) *Finder {
	return newFinder(worktrees, path, execPath)
}

func Lookup(ctx context.Context, f *Finder, shell string) (Interpreter, string, error) {
	return f.lookup(ctx, shell)
}

func Within(p, root string, fold bool) bool { return within(p, root, fold) }
func FinalPath(p string) (string, error)    { return finalPath(p) }
