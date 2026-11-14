package chain

import (
	"errors"
	"testing"
)

// Review 147 B-147-2: the manifest handle has one explicit holder from the
// moment Create returns. Each case runs the real writer path with the fake
// file system; the expectations are written from the case, not read back
// from the tool.

// manifestAfterExpiry blocks the manifest's call of kind ("create" or
// "write"), moves the clock past D while it is blocked, then releases it:
// the writer must take the expiry exit at its next decision point, start no
// Close, and keep the object held.
func manifestAfterExpiry(t *testing.T, kind string) OuterSnapshot {
	h := newHarness(t)
	c := h.fs.set(t, kind, ManifestName, 1, blocking())
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	done := h.run(h.cfg(Consistent, nil, nil))
	waitCh(t, c.entered, "manifest "+kind)
	h.clk.Set(at(121))
	// Past D the parent's D timer and the diagnostic task's re-check race:
	// either the parent finds no diagnostic terminal state (step 5 branch
	// b) or it consumes "expired-stopped" (branch a). Both give 2; neither
	// is 3 or 4 (the five-state table).
	if f := h.final(done); f.Conclusion != Incomplete || (f.Path != PathExpiredFinal && f.Path != PathConsumed) {
		t.Fatalf("final %v via %v, want 2 via expired-final or consumed", f.Conclusion, f.Path)
	}
	c.free()
	s := h.waitFor("writer terminal", writerTerminal)
	if st := s.Writer.Terminal; st.State != WriterExpired || st.Qual != Late {
		t.Errorf("writer terminal %v/%v, want expired-incomplete/late", st.State, st.Qual)
	}
	if n := h.fs.count("close", ManifestName); n != 0 {
		t.Errorf("manifest closed %d times after D", n)
	}
	m := s.Writer.Manifest
	if m.Holding.Holder != HolderWriter || !m.Holding.Held() || m.Holding.CloseStarted || m.Holding.CloseReturned || m.Holding.Released {
		t.Errorf("manifest holding %+v, want held by the writer, no Close started, not released", m.Holding)
	}
	if m.Create == nil || m.Create.Err != nil {
		t.Errorf("manifest create %+v: the result must be kept", m.Create)
	}
	if !containsName(h.wr.Held(), ManifestName) {
		t.Errorf("Held() %v lacks the manifest", h.wr.Held())
	}
	return s
}

func containsName(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

func TestB1472ManifestCreateThenExpiry(t *testing.T) {
	s := manifestAfterExpiry(t, "create")
	if s.Writer.Manifest.Create.Qual != Late {
		t.Errorf("create returned after D: qualification %v, want late", s.Writer.Manifest.Create.Qual)
	}
	if s.Writer.Manifest.Write != nil {
		t.Errorf("no write may start after D: %+v", s.Writer.Manifest.Write)
	}
}

func TestB1472ManifestWriteThenExpiry(t *testing.T) {
	s := manifestAfterExpiry(t, "write")
	w := s.Writer.Manifest.Write
	if w == nil || w.Err != nil || w.Qual != Late || w.N == 0 {
		t.Errorf("manifest write %+v: the late successful write must be kept", w)
	}
}

func TestB1472ManifestCloseFails(t *testing.T) {
	h := newHarness(t)
	cerr := errors.New("manifest close failed")
	h.fs.set(t, "close", ManifestName, 1, failing(cerr, 0))
	f, s := runFailingChain(t, h)
	failedChainChecks(t, h, f, s, "close "+ManifestName)
	if n := h.fs.count("close", ManifestName); n != 1 {
		t.Errorf("manifest closed %d times, want exactly 1 (not retried)", n)
	}
	m := s.Writer.Manifest.Holding
	if !m.Held() || m.Holder != HolderWriter || !m.CloseStarted || !m.CloseReturned || m.Released || !errors.Is(m.CloseErr, cerr) {
		t.Errorf("manifest holding %+v, want held (release unknown) with the real close error", m)
	}
	if oc := s.Writer.Manifest.Close; oc == nil || !errors.Is(oc.Err, cerr) {
		t.Errorf("manifest close outcome %+v", oc)
	}
}

func TestB1472RecordCloseFailsKeepsTheHandle(t *testing.T) {
	h := newHarness(t)
	cerr := errors.New("record close failed")
	h.fs.set(t, "close", "derived.jsonl", 1, failing(cerr, 0))
	f, s := runFailingChain(t, h)
	failedChainChecks(t, h, f, s, "close derived.jsonl")
	for _, fs := range s.Writer.Files {
		hd := fs.Holding
		if fs.Name == "derived.jsonl" {
			if !hd.Held() || !hd.CloseStarted || !hd.CloseReturned || hd.Released || !errors.Is(hd.CloseErr, cerr) {
				t.Errorf("derived holding %+v, want held with the real error", hd)
			}
			continue
		}
		if hd.Held() || !hd.Released {
			t.Errorf("%s holding %+v, want released", fs.Name, hd)
		}
	}
	if got := h.wr.Held(); len(got) != 1 || got[0] != "derived.jsonl" {
		t.Errorf("Held() %v", got)
	}
}

func TestB1472NormalCloseReleases(t *testing.T) {
	h := newHarness(t)
	h.startWriter()
	h.appendItems()
	h.clk.Set(at(100))
	f := h.final(h.run(h.cfg(Consistent, nil, nil)))
	wantFinal(t, f, Consistent, PathConsumed)
	s := h.o.Snapshot()
	m := s.Writer.Manifest.Holding
	if m.Held() || !m.CloseStarted || !m.CloseReturned || !m.Released || m.CloseErr != nil {
		t.Errorf("manifest holding %+v, want released", m)
	}
	if n := h.fs.count("close", ManifestName); n != 1 {
		t.Errorf("manifest closed %d times", n)
	}
	if got := h.wr.Held(); len(got) != 0 {
		t.Errorf("Held() %v after a healthy run", got)
	}
}
