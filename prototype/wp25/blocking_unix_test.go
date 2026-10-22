//go:build linux || darwin

package wp25

// A real O3 (T1, T6): a read end in blocking mode. Go cannot interrupt a
// blocking read, so stopping the read does not return the copier, and the
// descriptor stays open after Close until the read returns: "no further
// writes" holds from the seal, "resources ended" only when the last write
// end closes. os.Pipe gives non-blocking, pollable ends, where the stop
// works (T1.pipe.stop-read-latency); the counterexample is why the pipe
// path must use only such ends.

import (
	"bytes"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func fdOpen(fd int) bool {
	_, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0)
	return err == nil
}

func TestPipeBlockingReadEnd(t *testing.T) {
	s := begin(t, "T1", "T1.pipe.blocking-read-end", counter, false, true)
	defer s.done()
	var p [2]int
	if err := unix.Pipe(p[:]); err != nil {
		s.invalidf("pipe: %v", err)
	}
	rfd, wfd := p[0], p[1]
	r := os.NewFile(uintptr(rfd), "blocking-read-end")
	d := &dest{}
	port := d.copyPort("out")
	c := startCopy(r, port, false)
	time.Sleep(20 * time.Millisecond)
	if c.wait(time.Now()) {
		s.invalidf("copier returned before the stop")
	}
	ss := []*stream{{name: "out", r: r, c: c, p: port}}
	res := runS6(d, ss, 100*time.Millisecond, 300*time.Millisecond, false)
	_, _ = d.ownerPort().Write(doc)
	openAfterClose := fdOpen(rfd)
	// A holder still writes: the bytes reach the copier's next Read, its
	// write meets the sealed port.
	_, _ = unix.Write(wfd, []byte("late\n"))
	returned := c.wait(time.Now().Add(time.Second))
	_ = unix.Close(wfd)
	time.Sleep(20 * time.Millisecond)
	buf, log := d.snapshot()
	s.worked(1)
	s.expect("copiers blocked at the stop", 1, res.pending)
	s.expect("outcome", o3, res.out["out"])
	s.expect("descriptor still open after Close", true, openAfterClose)
	s.expect("copier returned once the holder wrote", true, returned)
	s.expect("its end", endWrite, endOf(ss[0]))
	s.expect("late write refused", 1, d.rejectedBy(port))
	s.expectTrue("nothing after the document", lastIsWhole(buf, log, doc) && !bytes.Contains(buf, []byte("late")), "")
	s.expect("descriptor closed after the read returned", false, fdOpen(rfd))
}
