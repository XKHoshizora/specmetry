// Package keep is the process-lifetime owner of the objects whose
// responsibility outlives the function and the tasks that made them
// (implementation plan §12.14 items 3 and 10: the registry holds every
// record and every unconfirmed responsibility until the test process ends;
// Review 155 B-154-2/3-R-H, review-resolution §204).
//
// Hold puts an object into a package-level table, a root of the program:
// the object, and everything it holds (its actual files, Cmd, Process and
// pidfd, its original errors, what is known of their release and reap),
// stay reachable until the process ends, whatever the function and the
// tasks that made it do. Nothing is ever removed from the table. keep never
// closes, waits for, retries, finalizes or releases anything, and it
// changes no state of the objects it holds: it only gives a responsibility
// an owner. It is not evidence of any cleanup.
package keep

import "sync"

// Entry is one hand-over: its order (1, 2, ...), what the object is, and
// a name for the records.
type Entry struct {
	Seq  int
	Kind string
	Name string
}

var table struct {
	mu   sync.Mutex
	objs []any
	ents []Entry
}

// Hold hands obj to the process owner and returns its entry. The object is
// held until the process ends; there is no release.
func Hold(kind, name string, obj any) Entry {
	table.mu.Lock()
	defer table.mu.Unlock()
	e := Entry{Seq: len(table.ents) + 1, Kind: kind, Name: name}
	table.objs = append(table.objs, obj)
	table.ents = append(table.ents, e)
	return e
}

// Get returns the object of the entry seq, the very value handed over.
func Get(seq int) (any, Entry, bool) {
	table.mu.Lock()
	defer table.mu.Unlock()
	if seq < 1 || seq > len(table.ents) {
		return nil, Entry{}, false
	}
	return table.objs[seq-1], table.ents[seq-1], true
}

// Entries returns every entry, in the order of Hold.
func Entries() []Entry {
	table.mu.Lock()
	defer table.mu.Unlock()
	return append([]Entry(nil), table.ents...)
}
