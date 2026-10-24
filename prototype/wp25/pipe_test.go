package wp25

// Pipe-path scenarios (T1, T6) with real helper processes on every
// platform: Specmetry's own os.Pipe, the write ends handed to the check and
// closed at once, one copier per stream into the controlled destination,
// and the S2 to S6 order of spec §8.5.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	tKill  = 5 * time.Second
	tDrain = 1 * time.Second
	tJoin  = 500 * time.Millisecond
)

var doc = []byte(`{"schema":"specmetry.check-run/v1","runs":[]}` + "\n")

// piped starts cmd with both streams on Specmetry's own pipes into d.
func piped(s *scen, cmd *exec.Cmd, d *dest, discard bool) (*os.File, []*stream) {
	ro, wo := pipePair(s)
	re, we := pipePair(s)
	cmd.Stdout, cmd.Stderr = wo, we
	return wo, startStreams(d, ro, re, we, discard)
}

func startStreams(d *dest, ro, re, we *os.File, discard bool) []*stream {
	po, pe := d.copyPort("out"), d.copyPort("err")
	return []*stream{
		{name: "out", r: ro, c: startCopy(ro, po, discard), p: po},
		{name: "err", r: re, c: startCopy(re, pe, discard), p: pe},
	}
}

func TestPipeNormalDrain(t *testing.T) {
	s := begin(t, "T1", "T1.pipe.normal-drain", normal, true, true)
	defer s.done()
	d := &dest{}
	cmd := helper("write", nil, "262144", "131072", "0")
	_, ss := piped(s, cmd, d, false)
	tr := startTree(s, cmd)
	closeWriteEnd(cmd)
	exited, status, _ := tr.waitExit(10 * time.Second)
	_ = tr.kill()
	conf := tr.confirm(tKill)
	if conf {
		tr.reap(tKill)
	}
	r := s.adopt(runS6(d, ss, tDrain, tJoin, !conf))
	_, _ = d.ownerPort().Write(doc)
	buf, log := d.snapshot()
	out, errb := streamOf(buf, log, "out"), streamOf(buf, log, "err")
	s.worked(int64(len(out) + len(errb)))
	s.needLoad(exited && status == "exit 0", "the helper wrote all its bytes and exited 0 (%v, %s)", exited, status)
	s.expect("tree confirmed", true, conf)
	s.expect("stdout", o1, r.out["out"])
	s.expect("stderr", o1, r.out["err"])
	s.expect("read stopped", false, r.stopped)
	s.expectTrue("stdout bytes intact", len(out) == 262144 && isPattern(out, 'o'), fmt.Sprintf("%d bytes", len(out)))
	s.expectTrue("stderr bytes intact", len(errb) == 131072 && isPattern(errb, 'e'), fmt.Sprintf("%d bytes", len(errb)))
	s.expectTrue("document last and whole", lastIsWhole(buf, log, doc), "")
}

// closeWriteEnd closes Specmetry's copies of the write ends handed to the
// check (S1): from then on only the check's processes hold them.
func closeWriteEnd(cmd *exec.Cmd) {
	for _, w := range []any{cmd.Stdout, cmd.Stderr} {
		if f, ok := w.(*os.File); ok {
			_ = f.Close()
		}
	}
}

func TestPipeEscapedHolder(t *testing.T) {
	// A writer the tree confirmation does not cover (here the scenario
	// itself keeps the stdout write end and writes): phase D ends at
	// T_drain without EOF, the read is stopped, phase J sees the copier
	// return, the ports are sealed. The holder's later writes fail.
	s := begin(t, "T1", "T1.pipe.escaped-holder", normal, true, true)
	defer s.done()
	d := &dest{}
	cmd := helper("write", nil, "1024", "1024", "0")
	wo, ss := piped(s, cmd, d, false)
	tr := startTree(s, cmd)
	_ = cmd.Stderr.(*os.File).Close()
	var holderErr error
	var holderWrites atomic.Int64
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			if _, err := fmt.Fprintf(wo, "late %d\n", i); err != nil {
				holderErr = err
				return
			}
			holderWrites.Add(1)
			time.Sleep(10 * time.Millisecond)
		}
	}()
	exited, status, _ := tr.waitExit(10 * time.Second)
	s.needLoad(exited && status == "exit 0", "the helper wrote and exited 0 (%v, %s)", exited, status)
	_ = tr.kill()
	conf := tr.confirm(tKill)
	if conf {
		tr.reap(tKill)
	}
	r := s.adopt(runS6(d, ss, tDrain, tJoin, !conf))
	s.needLoad(holderWrites.Load() > 0, "the holder wrote before the stop (%d writes)", holderWrites.Load())
	_, _ = d.ownerPort().Write(doc)
	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
	_ = wo.Close()
	buf, log := d.snapshot()
	s.worked(int64(len(buf)))
	s.expect("tree confirmed", true, conf)
	s.expect("stdout (holder alive)", o2, r.out["out"])
	s.expect("stderr", o1, r.out["err"])
	s.expect("read stopped", true, r.stopped)
	s.expect("copiers blocked at the stop", 1, r.pending)
	s.expectTrue("phase D used T_drain", r.drain >= tDrain, fmt.Sprintf("%v on its own clock (stopwatch %v)", r.drain, r.drainFine))
	s.expectTrue("phase J within T_join", r.join < tJoin, r.join.String())
	s.expectTrue("holder's writes fail after the stop", holderErr != nil, fmt.Sprint(holderErr))
	s.expectTrue("document last and whole", lastIsWhole(buf, log, doc), "")
}

func TestPipeF1SkipDrain(t *testing.T) {
	// F1 simulated: the tree is treated as not confirmed (no kill before
	// S6), so phase D is skipped. A descendant that inherited the pipes
	// keeps writing; after the stop and the seal, none of its output may
	// reach the destination, and its writes fail.
	s := begin(t, "T1", "T1.pipe.F1-skip-drain", fault, true, true)
	defer s.done()
	rep := s.newReport(reportSpec{"pid": 1, "epipe": -1, "ok": -1})
	d := &dest{}
	cmd := helper("holder-parent", []string{reportEnv + "=" + rep})
	_, ss := piped(s, cmd, d, false)
	release := gated(s, cmd)
	tr := startTree(s, cmd)
	closeWriteEnd(cmd)
	var holder owned
	s.awaitOwned(rep, 1, 5*time.Second, func(pid int) { holder = tr.ownReported(pid) })
	release()
	exited, status, _ := tr.waitExit(10 * time.Second)
	s.needLoad(exited && status == "exit 3", "the holder verified as the check's; its parent exited 3 (%v, %s)", exited, status)
	time.Sleep(100 * time.Millisecond) // the holder writes a few lines
	var keptEnds []*kept
	keep := ss
	mixed := injected("f1-mixed-fault")
	if mixed || injected("f1-one-kept") {
		// Review 082 N4-R: only stdout's read end is kept. With
		// f1-mixed-fault the holder's later write to stderr fails, and so
		// does its report; with f1-one-kept it writes only stdout.
		keep = ss[:1]
	}
	if mixed || injectedAny("read-end-kept", "f1-one-kept", "ok-report-fail", "fail-after-ok", "ok-open-output", "output-close-fails",
		"ok-epipe-reported", "epipe-then-ok", "ok-foreign", "ok-dup", "ok-truncated", "ok-malformed", "ok-limit", "f1-drain-fails",
		"ok-close-fails", "ok-limit-gated", "ok-limit-unwritable") {
		// A gate regression's healthy counter-case (Review 076 N4): the
		// scenario keeps a second handle on both read ends, so the stop
		// leaves the pipes open and the holder's writes do not fail. Its
		// late bytes then wait in the pipes, where the scenario sees them:
		// the evidence that the prediction is wrong while the tool works.
		for _, st := range keep {
			k, err := keepOpen(st.r.(*os.File))
			if err != nil {
				s.invalidf("a second handle on the read end: %v", err)
			}
			s.defer_(k.close)
			keptEnds = append(keptEnds, k)
		}
		injectHit("TestPipeF1SkipDrain", os.Getpid())
	}
	r := s.adopt(runS6(d, ss, tDrain, tJoin, true))
	atStop := waiting(s, keptEnds)
	if mixed {
		// The holder's round that writes stderr comes only now.
		if err := os.WriteFile(rep+".stopped", nil, 0o600); err != nil {
			s.invalidf("the stop marker: %v", err)
		}
	}
	_, _ = d.ownerPort().Write(doc)
	// Review 084 B-N4: the evidence is collected in full first, the report
	// read once more after the holder's end, and only then is the whole of
	// it classified; no record, and no answer, decides on its own.
	ev := observeF1(s, rep, holder, keptEnds, atStop, len(ss))
	buf, log := d.snapshot()
	s.worked(int64(len(buf)))
	_ = tr.kill()
	conf := tr.confirm(tKill)
	if conf {
		tr.reap(tKill)
	}
	ev.finalRead(rep, conf)
	found, why := classifyF1(ev)
	if len(why) > 0 {
		s.invalidf("observation failed: %s", strings.Join(why, "; "))
	}
	if found == f1Wrote {
		s.note("no epipe record; the holder's final answer came after its output closed, and late bytes reached every stream's kept read end from the stop to the end: %v", ev.late)
	}
	s.expect("stdout", o2, r.out["out"])
	s.expect("stderr", o2, r.out["err"])
	s.expect("phase D skipped", time.Duration(0), r.drain)
	s.expect("copiers blocked at the stop", 2, r.pending)
	s.expectTrue("phase J within T_join", r.join < tJoin, r.join.String())
	s.expectTrue("early output copied", bytes.Contains(buf, []byte("early-out")) && bytes.Contains(buf, []byte("early-err")), "")
	s.expectTrue("holder's late writes fail", found == f1Failed, ev.records.String())
	s.expectTrue("document last and whole", lastIsWhole(buf, log, doc), "")
	s.expect("cleanup kill confirmed", true, conf)
}

// f1Evidence is what the F1 scenario observed of its holder (Review 084
// B-N4), all of it collected before anything is classified.
type f1Evidence struct {
	holder   int       // the verified holder's PID
	state    string    // what was known of its state when the observation ended
	asked    bool      // the scenario asked for the holder's final answer
	kept     int       // streams with a read end the scenario kept
	streams  int       // the check's streams
	ended    bool      // after the question, EOF came through every kept read end
	late     []int     // bytes each kept stream carried from the stop to its end
	records  f1Records // the epipe and ok records of the final report, in order
	failures []string  // what failed while collecting: a read, a write, the report
}

type f1Records []reportLine

func (rs f1Records) String() string {
	if len(rs) == 0 {
		return "no epipe or ok record"
	}
	var b strings.Builder
	for i, r := range rs {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s %d", r.kind, r.pid)
	}
	return b.String()
}

// observeF1 collects the evidence while the holder may still run. Without
// a record of a failed write (Review 076 N4, Reviews 081 to 083 N4-R) the
// record may not have arrived, the holder may have failed to write it, or
// its late writes may not have failed; it is asked for its final answer,
// which ends the observation: it writes nothing more, closes its output,
// and answers "ok" only if none of its writes failed. The end is shown by
// EOF through each kept read end, within one 2 s budget for all of them;
// the late bytes are what each stream carried from the stop to that end.
// A failure on the way is recorded and the collection goes on.
func observeF1(s *scen, rep string, holder owned, keptEnds []*kept, atStop []int, streams int) f1Evidence {
	ev := f1Evidence{holder: holder.pidOf(), kept: len(keptEnds), streams: streams}
	failed := func(err error) {
		if err != nil && !errors.Is(err, errNotYet) {
			ev.failures = append(ev.failures, err.Error())
		}
	}
	epipe, err := awaitRecords(rep, "epipe", 1, 3*time.Second)
	failed(err)
	if len(epipe) == 1 && injectedAny("epipe-close-fails", "epipe-limit-gated", "epipe-limit-unwritable") {
		// Review 085's cases: the holder's failure after its record has
		// happened before the checks below (the test's own marker).
		awaitMarker(s, rep)
	}
	if len(epipe) == 0 {
		ev.asked = true
		if err := os.WriteFile(rep+".query", nil, 0o600); err != nil {
			failed(fmt.Errorf("asking the holder: %w", err))
		}
		ok, err := awaitRecords(rep, "ok", 1, 2*time.Second)
		failed(err)
		if injectedAny("fail-after-ok", "ok-epipe-reported", "ok-close-fails", "ok-limit-gated", "ok-limit-unwritable") && len(ok) == 1 {
			// The reviews' cases: the holder's failure after its answer has
			// happened before the checks below (the test's own marker).
			awaitMarker(s, rep)
		}
		ev.ended = true
		endBy := time.Now().Add(2 * time.Second)
		for i, k := range keptEnds {
			n, eof, err := k.drain(endBy)
			if err == nil && i == 0 && injected("f1-drain-fails") {
				// A tool failure beside a complete answer: the read of a
				// kept read end fails.
				injectHit("f1.drain", os.Getpid())
				err = errors.New("injected: the read failed")
			}
			if err != nil {
				failed(fmt.Errorf("reading a kept read end to its end: %w", err))
			}
			ev.late = append(ev.late, n-atStop[i])
			ev.ended = ev.ended && eof
		}
	}
	ev.state = stateOf(holder)
	return ev
}

// awaitMarker waits (bounded) for the test's own marker that a holder's
// failure happened; it only orders a gate regression, and decides nothing.
func awaitMarker(s *scen, rep string) {
	deadline := time.Now().Add(3 * time.Second)
	for !fileThere(rep+".failed") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	s.note("load: the holder's failure after its record happened: %v", fileThere(rep+".failed"))
}

// finalRead reads the report once more, after the holder's end: no writer
// is left, so it holds all the observation can have, and a record cut
// short is a truncation. An end not confirmed is no such read.
func (ev *f1Evidence) finalRead(rep string, ended bool) {
	if !ended {
		ev.failures = append(ev.failures, "the holder's end is not confirmed, so its report cannot be read as final")
	}
	lines, err := parseReport(rep, ended)
	if err == nil && injected("f1-final-read-fails") {
		// A tool failure beside the record of a failed write: the final
		// read fails.
		injectHit("f1.final-read", os.Getpid())
		err = errors.New("injected: the report could not be read")
	}
	if err == nil {
		_, err = tally(lines)
	}
	if err != nil {
		ev.failures = append(ev.failures, "report: "+err.Error())
		return
	}
	for _, l := range lines {
		if l.kind == "epipe" || l.kind == "ok" {
			ev.records = append(ev.records, l)
		}
	}
}

type f1Finding int

const (
	f1Invalid f1Finding = iota // the observation does not count
	f1Failed                   // the holder's late writes failed: the prediction holds
	f1Wrote                    // the tools worked, and the late writes did not fail
)

// classifyF1 decides on the whole of the evidence (Review 084 B-N4). By
// the protocol (report_test.go), "epipe" says a late write of the holder
// failed; "ok", given only when asked and only with the holder's output
// closed, says none did. They exclude each other: both, in either order,
// break the protocol, so neither counts. Each record must be the
// holder's. Every rule is checked before anything is decided; a success
// returns nothing early, and a failure while collecting outweighs both.
func classifyF1(ev f1Evidence) (f1Finding, []string) {
	var why []string
	for _, f := range ev.failures {
		if !slices.Contains(why, f) {
			why = append(why, f)
		}
	}
	epipe, ok := 0, 0
	for _, r := range ev.records {
		switch r.kind {
		case "epipe":
			epipe++
		case "ok":
			ok++
		default:
			continue
		}
		if r.pid != ev.holder {
			why = append(why, fmt.Sprintf("an %s record of process %d, not of the holder %d", r.kind, r.pid, ev.holder))
		}
	}
	if epipe > 1 || ok > 1 {
		why = append(why, fmt.Sprintf("%d epipe and %d ok records, where the protocol has at most one", epipe, ok))
	}
	switch {
	case epipe > 0 && ok > 0:
		why = append(why, fmt.Sprintf("the holder answered that none of its writes failed and reported a failed write (%s): the protocol has one or the other, so neither counts", ev.records))
	case epipe == 0 && ok == 0:
		why = append(why, fmt.Sprintf("no epipe record, and no final answer from the holder: its record was not written or has not arrived, or its tool failed (the holder %s)", ev.state))
	case ok > 0:
		if !ev.asked {
			why = append(why, "an answer nobody asked for")
		}
		if !ev.ended {
			why = append(why, fmt.Sprintf("the holder answered, but a stream still had a writer 2 s later: its answer was not the end of the observation (late bytes %v)", ev.late))
		}
		seen := ev.kept == ev.streams && len(ev.late) == ev.kept
		for _, n := range ev.late {
			if n <= 0 {
				seen = false
			}
		}
		if !seen {
			why = append(why, fmt.Sprintf("late bytes did not reach every stream through a kept read end (%d of %d kept, late bytes %v)", ev.kept, ev.streams, ev.late))
		}
	}
	if len(why) > 0 {
		return f1Invalid, why
	}
	if epipe > 0 {
		return f1Failed, nil
	}
	return f1Wrote, nil
}

// waiting is how many bytes wait in each kept read end; a read that fails
// stops the scenario as invalid.
func waiting(s *scen, ks []*kept) []int {
	n := make([]int, len(ks))
	for i, k := range ks {
		m, err := k.available()
		if err != nil {
			s.invalidf("observation failed: the bytes waiting in a kept read end could not be read: %v", err)
		}
		n[i] = m
	}
	return n
}

// stateOf says, for a reason, what is known of a process's state.
func stateOf(p owned) string {
	switch {
	case p.ended():
		return fmt.Sprintf("%d has ended", p.pidOf())
	case p.observeErr() != nil:
		return fmt.Sprintf("%d, whose state could not be read", p.pidOf())
	}
	return fmt.Sprintf("%d still runs", p.pidOf())
}

func TestPipeCapacityStop(t *testing.T) {
	// Counterexample to "a copier stops at its first write error": the
	// check then blocks on a full pipe and is forced into the timeout.
	s := begin(t, "T6", "T6.pipe.capacity-stop", counter, true, true)
	defer s.done()
	d := &dest{copyCap: 64 << 10}
	cmd := helper("write", nil, "1048576", "0", "0")
	_, ss := piped(s, cmd, d, false)
	tr := startTree(s, cmd)
	closeWriteEnd(cmd)
	exited, _, _ := tr.waitExit(2 * time.Second)
	_ = tr.kill()
	conf := tr.confirm(tKill)
	if conf {
		tr.reap(tKill)
	}
	r := s.adopt(runS6(d, ss, tDrain, tJoin, !conf))
	_, _ = d.ownerPort().Write(doc)
	buf, log := d.snapshot()
	out := streamOf(buf, log, "out")
	s.worked(int64(len(out)))
	s.expect("main exited before the timeout", false, exited)
	s.expect("stdout", o2, r.out["out"])
	s.expect("copied bytes", 64<<10, len(out))
	s.expect("tree confirmed after the kill", true, conf)
	s.expectTrue("document last and whole", lastIsWhole(buf, log, doc), "")
}

func TestPipeCapacityDiscard(t *testing.T) {
	// The candidate: after a write error the copier keeps reading and
	// drops the rest, so the check is not blocked; the run is O2.
	s := begin(t, "T6", "T6.pipe.capacity-discard", normal, true, true)
	defer s.done()
	d := &dest{copyCap: 64 << 10}
	cmd := helper("write", nil, "1048576", "0", "0")
	_, ss := piped(s, cmd, d, true)
	tr := startTree(s, cmd)
	closeWriteEnd(cmd)
	exited, status, _ := tr.waitExit(5 * time.Second)
	_ = tr.kill()
	conf := tr.confirm(tKill)
	if conf {
		tr.reap(tKill)
	}
	r := s.adopt(runS6(d, ss, tDrain, tJoin, !conf))
	_, _ = d.ownerPort().Write(doc)
	buf, log := d.snapshot()
	out := streamOf(buf, log, "out")
	s.worked(int64(len(out)))
	var dropped int64
	if ss[0].c.res != nil {
		dropped = ss[0].c.res.discarded
	}
	s.expect("main exited", true, exited)
	s.expect("status", "exit 0", status)
	s.expect("stdout", o2, r.out["out"])
	s.expect("stdout end", endWrite, endOf(ss[0]))
	s.expectTrue("copied bytes are the pattern's prefix", len(out) == 64<<10 && isPattern(out, 'o'), fmt.Sprintf("%d bytes", len(out)))
	s.expect("dropped bytes", 1048576-(64<<10), dropped)
	s.expect("read stopped", false, r.stopped)
	s.expectTrue("document last and whole", lastIsWhole(buf, log, doc), "")
}

func endOf(st *stream) copyEnd {
	if st.c.res == nil {
		return "none"
	}
	return st.c.res.end
}

func TestPipeAliasConcurrent(t *testing.T) {
	// Both streams into one destination, with Specmetry's own progress
	// writes going in concurrently through the owner port.
	s := begin(t, "T6", "T6.pipe.alias-concurrent", normal, true, true)
	defer s.done()
	d := &dest{}
	cmd := helper("write", nil, "200000", "200000", "0")
	_, ss := piped(s, cmd, d, false)
	tr := startTree(s, cmd)
	closeWriteEnd(cmd)
	owner := d.ownerPort()
	var wg sync.WaitGroup
	wg.Add(1)
	progress := 0
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			if _, err := fmt.Fprintf(owner, "progress %03d\n", i); err == nil {
				progress++
			}
			time.Sleep(time.Millisecond)
		}
	}()
	exited, status, _ := tr.waitExit(10 * time.Second)
	_ = tr.kill()
	conf := tr.confirm(tKill)
	if conf {
		tr.reap(tKill)
	}
	r := s.adopt(runS6(d, ss, tDrain, tJoin, !conf))
	wg.Wait()
	_, _ = owner.Write(doc)
	buf, log := d.snapshot()
	out, errb := streamOf(buf, log, "out"), streamOf(buf, log, "err")
	s.worked(int64(len(buf)))
	wholeOwner := true
	for _, sp := range log {
		if sp.port == "owner" && sp.n != len("progress 000\n") && sp.n != len(doc) {
			wholeOwner = false
		}
	}
	s.needLoad(exited && status == "exit 0", "the helper wrote all its bytes and exited 0 (%v, %s)", exited, status)
	s.needLoad(progress == 200, "the owner made its 200 progress writes (%d)", progress)
	s.expect("stdout", o1, r.out["out"])
	s.expect("stderr", o1, r.out["err"])
	s.expectTrue("stdout intact in order", len(out) == 200000 && isPattern(out, 'o'), fmt.Sprintf("%d bytes", len(out)))
	s.expectTrue("stderr intact in order", len(errb) == 200000 && isPattern(errb, 'e'), fmt.Sprintf("%d bytes", len(errb)))
	s.expectTrue("every owner write whole", wholeOwner, "")
	s.expectTrue("document last and whole", lastIsWhole(buf, log, doc), "")
}

func TestPipeStopReadLatency(t *testing.T) {
	// Closing the read end returns a Read that is blocked with no data,
	// while a writer still holds the pipe (in-process, 50 times).
	s := begin(t, "T1", "T1.pipe.stop-read-latency", normal, false, true)
	defer s.done()
	d := &dest{}
	var lat []time.Duration
	returned, blocked := 0, 0
	for i := 0; i < 50; i++ {
		r, w := pipePair(s)
		c := startCopy(r, d.copyPort("x"), false)
		time.Sleep(5 * time.Millisecond)
		if c.wait(time.Now()) {
			_ = w.Close()
			continue // not blocked: this repetition shows nothing
		}
		blocked++
		since := stopwatch()
		_ = r.Close()
		if c.wait(time.Now().Add(time.Second)) {
			returned++
			lat = append(lat, since())
		}
		_ = w.Close()
	}
	s.worked(int64(returned))
	s.needLoad(blocked == 50, "every copier was blocked in Read before the stop (%d of 50)", blocked)
	s.expect("copiers returned within 1 s", 50, returned)
	s.note("latency %s", durations(lat))
}

func TestPipeSilentHolderReal(t *testing.T) {
	// The same with a real process: a descendant holds the pipes without
	// writing, the main process has exited, the tree is not confirmed.
	s := begin(t, "T1", "T1.pipe.silent-holder", fault, true, true)
	defer s.done()
	var lat []time.Duration
	ok := 0
	for i := 0; i < 3; i++ {
		rep := s.newReport(reportSpec{"pid": 1})
		d := &dest{}
		cmd := helper("silent-parent", []string{reportEnv + "=" + rep})
		_, ss := piped(s, cmd, d, false)
		release := gated(s, cmd)
		tr := startTree(s, cmd)
		closeWriteEnd(cmd)
		s.awaitOwned(rep, 1, 5*time.Second, func(pid int) { tr.ownReported(pid) })
		release()
		exited, status, _ := tr.waitExit(10 * time.Second)
		s.needLoad(exited && status == "exit 0", "run %d: the holder verified as the check's; its parent exited 0 (%v, %s)", i+1, exited, status)
		since := stopwatch()
		r := s.adopt(runS6(d, ss, tDrain, tJoin, true))
		s.needLoad(r.pending == 2, "run %d: both copiers blocked at the stop (%d)", i+1, r.pending)
		if r.out["out"] == o2 && r.out["err"] == o2 {
			ok++
			lat = append(lat, since())
		}
		_ = tr.kill()
		if tr.confirm(tKill) {
			tr.reap(tKill)
		}
	}
	s.worked(int64(ok))
	s.expect("runs with both copiers returned (O2) in phase J", 3, ok)
	s.note("stop to return %s", durations(lat))
}
