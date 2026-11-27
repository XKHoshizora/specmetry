package keep

import (
	"runtime"
	"testing"
	"weak"
)

type thing struct{ n int }

//go:noinline
func handOver(n int) (Entry, weak.Pointer[thing]) {
	x := &thing{n: n}
	return Hold("test thing", "h5", x), weak.Make(x)
}

// TestHoldKeepsUntilTheProcessEnds is H5 (review-resolution §204): the
// entries come in the order of Hold; Get returns the very object handed
// over, also after its maker returned and complete collections ran;
// nothing is ever removed.
func TestHoldKeepsUntilTheProcessEnds(t *testing.T) {
	before := len(Entries())
	e1, w1 := handOver(1)
	e2, w2 := handOver(2)
	for range 3 {
		runtime.GC()
	}
	if e1.Seq != before+1 || e2.Seq != before+2 || e1.Kind != "test thing" || e1.Name != "h5" {
		t.Fatalf("entries %+v %+v after %d", e1, e2, before)
	}
	for _, c := range []struct {
		e Entry
		w weak.Pointer[thing]
		n int
	}{{e1, w1, 1}, {e2, w2, 2}} {
		obj, e, ok := Get(c.e.Seq)
		x, isThing := obj.(*thing)
		if !ok || e != c.e || !isThing || x != c.w.Value() || x.n != c.n {
			t.Errorf("seq %d: ok %v entry %+v same object %v", c.e.Seq, ok, e, isThing && x == c.w.Value())
		}
	}
	all := Entries()
	if len(all) != before+2 || all[before] != e1 || all[before+1] != e2 {
		t.Errorf("entries %+v", all[before:])
	}
	if _, _, ok := Get(0); ok {
		t.Error("Get(0) found an entry")
	}
	if _, _, ok := Get(len(all) + 1); ok {
		t.Error("Get past the end found an entry")
	}
}
