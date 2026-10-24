package wp25

// Destination scenarios without processes (T1, T6): the copier that does
// not return (O3), what it still owns, whether later runs may proceed, and
// the destination's rules under concurrency and failure.

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var errLingering = errors.New("a copier or a stop of an earlier run is not confirmed to have ended")

// lingering is the candidate owner of what a run left (s6Result.left): O3
// copiers, each keeping its reader and its sealed port until it returns,
// and stops not done (Review 081 B1), each keeping its read end until its
// close returns. admit is the policy for a later run in the same process.
type lingering struct {
	mu     sync.Mutex
	tasks  []leftover
	refuse bool
}

func (l *lingering) add(r s6Result) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tasks = append(l.tasks, r.left...)
}

// pending drops what has ended and counts the rest.
func (l *lingering) pending() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept := l.tasks[:0]
	for _, t := range l.tasks {
		if !t.until(time.Now()) {
			kept = append(kept, t)
		}
	}
	l.tasks = kept
	return len(kept)
}

func (l *lingering) admit() error {
	if l.refuse && l.pending() > 0 {
		return errLingering
	}
	return nil
}

func TestDestSealEndless(t *testing.T) {
	// A copier that never returns while its port takes writes: O3. After
	// the seal only the owner's document is added, and the copier returns
	// on its next write.
	s := begin(t, "T6", "T6.dest.seal-endless", normal, false, true)
	defer s.done()
	g0 := runtime.NumGoroutine()
	d := &dest{}
	e := &endless{stop: make(chan struct{})}
	p := d.copyPort("out")
	ss := []*stream{{name: "out", r: e, c: startCopy(e, p, false), p: p}}
	r := s.adopt(runS6(d, ss, 200*time.Millisecond, 100*time.Millisecond, false))
	atSeal, _ := d.snapshot()
	_, _ = d.ownerPort().Write(doc)
	time.Sleep(50 * time.Millisecond)
	after, log := d.snapshot()
	returned := ss[0].c.wait(time.Now().Add(time.Second))
	s.worked(int64(len(atSeal)))
	s.expect("outcome", o3, r.out["out"])
	s.expectTrue("after the seal only the document was added", bytes.Equal(after, append(atSeal, doc...)), "")
	s.expectTrue("document last and whole", lastIsWhole(after, log, doc), "")
	s.expect("copier returned on its next write", true, returned)
	s.expect("its end", endWrite, endOf(ss[0]))
	s.expectTrue("refused writes after the seal", d.rejectedBy(p) >= 1, fmt.Sprint(d.rejectedBy(p)))
	s.expectTrue("goroutines back", goroutinesSettle(g0, time.Second) <= g0, "")
}

func TestDestO3GateLingering(t *testing.T) {
	// The copier the spec cannot rule out: blocked in Read, ignoring Close.
	// From the seal on it cannot add output ("no further writes"); it ends
	// only when its Read returns ("resources ended"), here when the gate is
	// released. The two later-run policies are both exercised.
	s := begin(t, "T1", "T1.O3.gate-lingering", fault, false, true)
	defer s.done()
	g0 := runtime.NumGoroutine()
	d := &dest{}
	g := newGate()
	// o3-unreleased, a gate regression (Review 081 B1): the scenario never
	// releases its gate, so its O3 copier still runs when it ends; only the
	// wait for what its runs handed on can find that.
	unreleased := injected("o3-unreleased")
	if !unreleased {
		s.defer_(g.release)
	}
	p1 := d.copyPort("run1")
	ss := []*stream{{name: "out", r: g, c: startCopy(g, p1, false), p: p1}}
	r := s.adopt(runS6(d, ss, 100*time.Millisecond, 100*time.Millisecond, false))
	_, _ = d.ownerPort().Write(doc)
	reg := &lingering{}
	reg.add(r)

	reg.refuse = true
	refused := errors.Is(reg.admit(), errLingering)

	reg.refuse = false
	allowed := reg.admit() == nil
	p2 := d.copyPort("run2")
	src := strings.NewReader("second run\n")
	ss2 := []*stream{{name: "out", r: nopCloser{src}, c: startCopy(src, p2, false), p: p2}}
	r2 := s.adopt(runS6(d, ss2, time.Second, 100*time.Millisecond, false))
	doc2 := []byte(`{"run":2}` + "\n")
	_, _ = d.ownerPort().Write(doc2)
	beforeRelease, log := d.snapshot()
	pendingBefore := reg.pending()
	goroutinesBefore := runtime.NumGoroutine()

	if unreleased {
		injectHit("TestDestO3GateLingering", os.Getpid())
	} else {
		g.release()
	}
	ended := ss[0].c.wait(time.Now().Add(time.Second))
	afterRelease, _ := d.snapshot()
	s.worked(int64(len(beforeRelease)))
	s.expect("run 1", o3, r.out["out"])
	s.expect("policy refuse: later run refused", true, refused)
	s.expect("policy allow: later run admitted", true, allowed)
	s.expect("run 2", o1, r2.out["out"])
	s.expectTrue("run 2's document last and whole", lastIsWhole(beforeRelease, log, doc2), "")
	s.expect("pending copiers before the release", 1, pendingBefore)
	s.expectTrue("the copier still runs before the release", goroutinesBefore > g0, fmt.Sprintf("%d > %d", goroutinesBefore, g0))
	s.expect("copier ended after the release", true, ended)
	s.expect("its end", endWrite, endOf(ss[0]))
	s.expect("its late write refused", 1, d.rejectedBy(p1))
	s.expectTrue("nothing added after the release", bytes.Equal(beforeRelease, afterRelease), "")
	s.expect("pending copiers after the release", 0, reg.pending())
	s.expectTrue("goroutines back", goroutinesSettle(g0, time.Second) <= g0, "")
}

type nopCloser struct {
	r interface{ Read([]byte) (int, error) }
}

func (n nopCloser) Read(p []byte) (int, error) { return n.r.Read(p) }
func (nopCloser) Close() error                 { return nil }

func TestDestConcurrentWriters(t *testing.T) {
	// Eight copy ports and the owner write concurrently; every write is
	// one span, every port's bytes keep their order, the document is last.
	s := begin(t, "T6", "T6.dest.concurrent-writers", normal, false, true)
	defer s.done()
	d := &dest{}
	const ports, writes = 8, 500
	ps := make([]*port, ports)
	want := make([][]byte, ports)
	var wg sync.WaitGroup
	for i := range ps {
		ps[i] = d.copyPort(fmt.Sprintf("p%d", i))
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(i)))
			for j := 0; j < writes; j++ {
				b := bytes.Repeat([]byte{byte('a' + i)}, 1+rng.Intn(300))
				b[len(b)-1] = byte('0' + j%10)
				if _, err := ps[i].Write(b); err == nil {
					want[i] = append(want[i], b...)
				}
			}
		}(i)
	}
	owner := d.ownerPort()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 100; j++ {
			_, _ = fmt.Fprintf(owner, "progress %03d\n", j)
		}
	}()
	wg.Wait()
	d.seal(ps...)
	big := bytes.Repeat([]byte("D"), 256<<10)
	_, _ = owner.Write(big)
	buf, log := d.snapshot()
	s.worked(int64(len(buf)))
	contiguous := true
	off := 0
	for _, sp := range log {
		if sp.off != off {
			contiguous = false
		}
		off += sp.n
	}
	inOrder := true
	for i := range ps {
		if !bytes.Equal(streamOf(buf, log, ps[i].name), want[i]) {
			inOrder = false
		}
	}
	s.expectTrue("spans contiguous, none split", contiguous && off == len(buf), "")
	s.expectTrue("each port's bytes in order", inOrder, "")
	s.expectTrue("document last and whole", lastIsWhole(buf, log, big), "")
}

func TestDestOwnerShortWrite(t *testing.T) {
	// The owner's document does not fit: the write is short and reported
	// (§8.6 C2); no second document is appended.
	s := begin(t, "T6", "T6.dest.owner-short-write", fault, false, true)
	defer s.done()
	d := &dest{ownerCap: 100}
	big := bytes.Repeat([]byte("D"), 1000)
	n, err := d.ownerPort().Write(big)
	buf, log := d.snapshot()
	s.worked(int64(n))
	s.expect("written", 100, n)
	s.expectTrue("error reported", errors.Is(err, errCapacity), fmt.Sprint(err))
	s.expect("owner spans", 1, len(log))
	s.expect("destination bytes", 100, len(buf))
}

// blockingWriter is an arbitrary Writer whose Write blocks until released:
// what the controlled destination rules out.
type blockingWriter struct {
	mu   sync.Mutex
	buf  []byte
	gate chan struct{}
}

func (b *blockingWriter) Write(p []byte) (int, error) {
	<-b.gate
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *blockingWriter) ownerWrite(p []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
}

func TestArbitraryWriterCounterexample(t *testing.T) {
	// Review 072's counterexample, again: with a Writer that blocks, the
	// copier cannot be stopped; abandoning it after T_join lets its late
	// bytes land after the document. Hence D20 (i): such Writers are
	// refused before anything starts.
	s := begin(t, "T6", "T6.arbitrary-writer", counter, false, true)
	defer s.done()
	w := &blockingWriter{gate: make(chan struct{})}
	var once sync.Once
	release := func() { once.Do(func() { close(w.gate) }) }
	s.defer_(release)
	src := strings.NewReader("check output\n")
	c := startCopy(src, w, false)
	returned := c.wait(time.Now().Add(200 * time.Millisecond))
	w.ownerWrite(doc)
	release()
	c.wait(time.Now().Add(time.Second))
	w.mu.Lock()
	got := string(w.buf)
	w.mu.Unlock()
	s.worked(int64(len(got)))
	s.expect("copier returned within T_join", false, returned)
	s.expectTrue("late output after the document", strings.HasPrefix(got, string(doc)) && strings.HasSuffix(got, "check output\n"), fmt.Sprintf("%q", got))
}

func TestDestDocUnderLoad(t *testing.T) {
	// 50 times: a copier writing as fast as it can, the seal, a 64 KiB
	// document; the document is last and whole every time.
	s := begin(t, "T6", "T6.dest.doc-under-load", normal, false, true)
	defer s.done()
	big := bytes.Repeat([]byte("J"), 64<<10)
	good := 0
	for i := 0; i < 50; i++ {
		d := &dest{}
		e := &endless{stop: make(chan struct{})}
		p := d.copyPort("out")
		c := startCopy(e, p, false)
		time.Sleep(5 * time.Millisecond)
		d.seal(p)
		_, _ = d.ownerPort().Write(big)
		buf, log := d.snapshot()
		if lastIsWhole(buf, log, big) && c.wait(time.Now().Add(time.Second)) && c.res.end == endWrite {
			good++
		}
		close(e.stop)
	}
	s.worked(int64(good))
	s.expect("document last and whole, copier refused", 50, good)
}
