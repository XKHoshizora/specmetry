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
	r := runS6(d, ss, tDrain, tJoin, !conf)
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
	r := runS6(d, ss, tDrain, tJoin, !conf)
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
	s.expectTrue("phase D used T_drain", r.drain >= tDrain, r.drain.String())
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
	rep := s.newReport(reportSpec{"pid": 1, "epipe": -1})
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
	if injected("read-end-kept") {
		// A gate regression's healthy counter-case (Review 076 N4): the
		// scenario keeps a second handle on both read ends, so the stop
		// leaves the pipes open and the holder's writes do not fail. The
		// tool still works: the prediction is wrong, which is refuted.
		for _, st := range ss {
			closeKept, err := keepOpen(st.r.(*os.File))
			if err != nil {
				s.invalidf("a second handle on the read end: %v", err)
			}
			s.defer_(closeKept)
		}
		injectHit("TestPipeF1SkipDrain", os.Getpid())
	}
	r := runS6(d, ss, tDrain, tJoin, true)
	_, _ = d.ownerPort().Write(doc)
	epipe, err := awaitRecords(rep, "epipe", 1, 3*time.Second)
	if err != nil && !errors.Is(err, errNotYet) {
		s.invalidf("%v", err)
	}
	if len(epipe) == 0 {
		// No record (Review 076 N4). While the holder still runs, its
		// writes have not failed: a finding. A holder that has ended
		// without its record could not report what it saw: the observation
		// failed. Its end is read first, then the report once more, so a
		// record written just before the end still counts.
		if holder.ended() {
			if epipe, err = awaitRecords(rep, "epipe", 1, 0); err != nil && !errors.Is(err, errNotYet) {
				s.invalidf("%v", err)
			}
			if len(epipe) == 0 {
				s.invalidf("observation failed: the holder %d ended without its epipe record, so it could not report what it saw", holder.pidOf())
			}
		} else {
			s.note("no epipe record; the holder %d still runs", holder.pidOf())
		}
	}
	buf, log := d.snapshot()
	s.worked(int64(len(buf)))
	_ = tr.kill()
	conf := tr.confirm(tKill)
	if conf {
		tr.reap(tKill)
	}
	s.expect("stdout", o2, r.out["out"])
	s.expect("stderr", o2, r.out["err"])
	s.expect("phase D skipped", time.Duration(0), r.drain)
	s.expect("copiers blocked at the stop", 2, r.pending)
	s.expectTrue("phase J within T_join", r.join < tJoin, r.join.String())
	s.expectTrue("early output copied", bytes.Contains(buf, []byte("early-out")) && bytes.Contains(buf, []byte("early-err")), "")
	s.expectTrue("holder's late writes fail", len(epipe) == 1, fmt.Sprintf("%d epipe reports", len(epipe)))
	s.expectTrue("document last and whole", lastIsWhole(buf, log, doc), "")
	s.expect("cleanup kill confirmed", true, conf)
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
	r := runS6(d, ss, tDrain, tJoin, !conf)
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
	r := runS6(d, ss, tDrain, tJoin, !conf)
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
	r := runS6(d, ss, tDrain, tJoin, !conf)
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
		r := runS6(d, ss, tDrain, tJoin, true)
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
