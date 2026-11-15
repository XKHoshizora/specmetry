// Package toolctl is the control protocol of the tool's own calls
// (implementation plan §12.14 items 3 to 13; phase 2, review-resolution
// §200): a registry that registers every call before anything runs, a start
// task that owns the pipes until it hands the read ends over and is the
// only reaper, two read tasks, a parent that only queries published copies,
// and the termination at W by platform (Linux: one request through the
// pidfd the start task opened; Windows: one Kill of the process handle;
// darwin: none, the cleanup is unknown). Nothing is ever removed from the
// registry: a result that comes late is appended, and a failure is never
// cleared by a later success.
//
// It is the only code of the tool that starts a process. It is used for
// the three reviewed calls (E-10 (ii) accepted for phase 2's tests only)
// and, through the seams, for the pseudo-tools of LC-9.
package toolctl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/chain"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/collect"
)

// DefaultLimit is the output limit of each stream (§12.14 item 4 fixes no
// number; this value is the implementation's choice, registered for review
// in the phase 2 record §4 item 10). More than this many bytes is an
// overflow.
const DefaultLimit = 64 << 20

// The descriptors (§12.14 item 4).
const (
	StdoutW = iota
	StdoutR
	StderrW
	StderrR
)

// DescNames name the descriptors.
var DescNames = [4]string{"stdout-w", "stdout-r", "stderr-w", "stderr-r"}

// StreamNames name the two outputs.
var StreamNames = [2]string{"stdout", "stderr"}

// Descriptor states.
const (
	DescNone    = "none"
	DescOpen    = "open"
	DescClosing = "closing"
	DescClosed  = "closed"
)

// Record states (§12.14 item 3).
const (
	StRegistered  = "registered"
	StStarting    = "starting"
	StStarted     = "started"
	StStartFailed = "start-failed"
	StWaited      = "waited" // this call's one Wait returned; Class says whether the reap is confirmed
	StRefused     = "refused"
)

// pidfd states (§12.14 item 9).
const (
	PidfdNone        = "none" // not this platform, or no process
	PidfdUnpublished = "unpublished"
	PidfdOpen        = "open"
	PidfdOpenFailed  = "open-failed"
	PidfdClosing     = "closing"
	PidfdClosed      = "closed"
)

// Termination request states (§12.14 item 9).
const (
	ReqNone      = "none"
	ReqReserved  = "reserved"
	ReqRunning   = "running"
	ReqDone      = "done"
	ReqAbandoned = "abandoned"
)

// Termination modes, by platform (§12.14 item 8).
const (
	ModePidfd = "pidfd" // Linux
	ModeKill  = "kill"  // Windows: the process handle
	ModeNone  = "none"  // darwin: no termination request
)

// The named hook points the seams can block in to fix an interleaving.
const (
	HookAfterHandover    = "after handover"     // pipes built and handed over; before Start
	HookBeforeWait       = "before wait"        // after Start (and the pidfd); before the one Wait
	HookExpiryDecided    = "expiry decided"     // the parent decided to request; before the reservation
	HookRequestStart     = "request start"      // a request task starts; before its window check
	HookBeforePidfdClose = "before pidfd close" // the pidfd is sealed; before the closer waits and closes
)

// Desc is one descriptor's holder and close state.
type Desc struct {
	State    string
	Holder   string
	CloseErr string // the real Close result; "" is nil
	Injected []string
	Late     bool // the real result was published at or after W
}

// Stream is one read task's published state. Data is published only with
// Done (§12.14 item 7).
type Stream struct {
	Started  bool
	Done     bool
	Bytes    int64 // bytes read in total
	Data     []byte
	EOF      bool
	ReadErr  string
	Overflow bool
	Late     bool
}

// Pidfd is the pidfd the start task opened (Linux).
type Pidfd struct {
	State    string
	FD       int
	OpenErr  string
	CloseErr string
	Closes   int
}

// Request is the one termination request of a call.
type Request struct {
	Mode  string
	State string
	FD    int
	Err   string
	Sends int // calls of PidfdSendSignal or Kill
	Late  bool
}

// Exit is the copy of the ProcessState this call's Wait returned.
type Exit struct {
	Code   int
	Exited bool
	String string
}

// Event is one publication, in order.
type Event struct {
	At   time.Time
	What string
	Late bool // at or after W
}

// Record is one call as the registry holds it until the process ends.
type Record struct {
	ID         int
	Spec       collect.CallSpec
	State      string
	Class      string // A, B or C by §12.6; "" while unknown
	Phase      string // C's phase: start, wait, std-panic; A's: pipe, precondition
	StartErr   string
	WaitCalled bool
	WaitErr    string
	Exit       *Exit
	Pid        int
	Desc       [4]Desc
	Out        [2]Stream
	Pidfd      Pidfd
	Req        Request
	Expired    bool
	NotAdmit   []string // termination not admitted, and why
	Unknown    []string // cleanup unknown, and why
	Failures   []string // control failures, in order; never removed
	Injected   []string
	Events     []Event
	Tasks      int // tasks started and not returned
	Registered time.Time

	cmd     *exec.Cmd
	proc    *os.Process
	changed chan struct{}
}

// Snapshot is a copy of a record's published state.
type Snapshot struct {
	ID         int
	Kind       string
	Args       []string
	Dir        string
	State      string
	Class      string
	Phase      string
	StartErr   string
	WaitCalled bool
	WaitErr    string
	Exit       *Exit
	Pid        int
	Desc       [4]Desc
	Out        [2]Stream
	Pidfd      Pidfd
	Req        Request
	Expired    bool
	NotAdmit   []string
	Unknown    []string
	Failures   []string
	Injected   []string
	Events     []Event
	Tasks      int
	Terminal   bool
	Success    bool
	Mode       string
}

// Seams are the test seams; production leaves every field nil. Each one
// replaces exactly one operation; a seam may call the real operation.
type Seams struct {
	Pipe          func(id, n int) (r, w *os.File, err error) // n: 0 stdout, 1 stderr
	Read          func(id, stream int, f *os.File, p []byte) (int, error)
	CloseFile     func(id, desc int, f *os.File) error
	InjectClose   func(id, desc int) error // after the real result is published
	AfterHandover func(id int, stdoutW, stderrW *os.File)
	Prepare       func(id int, cmd *exec.Cmd) // completes the Cmd before Start (pseudo-tools)
	PidfdOpen     func(id, pid int) (int, error)
	SendSignal    func(id, fd int) error
	CloseFD       func(id, fd int) error
	Kill          func(id int, p *os.Process) error
	Hook          func(point string, id int)
}

// Options configure a registry.
type Options struct {
	Clock chain.Clock
	W, D  time.Time
	Limit int64 // per stream; 0 is DefaultLimit
	Seams *Seams
}

// Registry holds every call of one owner (the job, or one test case).
type Registry struct {
	mu      sync.Mutex // regMu
	o       Options
	s       Seams
	mode    string
	recs    []*Record
	changed chan struct{}
	refusal string // set when the registry refuses every call
}

// New makes a registry. On Linux it refuses every call when SIGCHLD is
// ignored: the pidfd's anchoring premise (§12.14 item 8) does not hold.
func New(o Options) *Registry {
	if o.Limit <= 0 {
		o.Limit = DefaultLimit
	}
	r := &Registry{o: o, mode: platformMode, changed: make(chan struct{})}
	if o.Seams != nil {
		r.s = *o.Seams
	}
	if platformMode == ModePidfd && sigchldIgnored() {
		r.refusal = "SIGCHLD is ignored: the pidfd's premise (no automatic reaping, §12.14 item 8) does not hold"
	}
	return r
}

// Mode is the platform's termination mode.
func (r *Registry) Mode() string { return r.mode }

// W and D are the registry's deadlines.
func (r *Registry) W() time.Time { return r.o.W }

// D is the registry's absolute deadline.
func (r *Registry) D() time.Time { return r.o.D }

func (r *Registry) hook(point string, id int) {
	if r.s.Hook != nil {
		r.s.Hook(point, id)
	}
}

// publishLocked records one publication and wakes every waiter (§12.14
// item 6: the channel is closed and renewed under the lock).
func (r *Registry) publishLocked(rec *Record, what string) {
	now := r.o.Clock.Now()
	rec.Events = append(rec.Events, Event{At: now, What: what, Late: !now.Before(r.o.W)})
	close(rec.changed)
	rec.changed = make(chan struct{})
	close(r.changed)
	r.changed = make(chan struct{})
}

// noProcessLocked marks that this call has no process to anchor a pidfd
// on (a start failure).
func (r *Registry) noProcessLocked(rec *Record) {
	if r.mode == ModePidfd {
		rec.Pidfd.State = PidfdNone
	}
}

func (r *Registry) lateLocked() bool { return !r.o.Clock.Now().Before(r.o.W) }

func (r *Registry) failLocked(rec *Record, s string) { rec.Failures = append(rec.Failures, s) }

// Invoke runs one call (collect.Invoker). It registers the record, starts
// the start task, and waits only on the record's published changes and W
// (§12.14 items 3, 6, 7). At W it decides the termination (item 8) and
// returns the published copy, which then fails; the tasks go on.
func (r *Registry) Invoke(s collect.CallSpec) collect.CallResult {
	rec := r.register(s)
	if rec.State == StRefused {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.resultLocked(rec)
	}
	go r.startTask(rec)
	wch, stop := r.o.Clock.Timer(r.o.W)
	defer stop()
	for {
		r.mu.Lock()
		if rec.Tasks == 0 {
			res := r.resultLocked(rec)
			r.mu.Unlock()
			return res
		}
		ch := rec.changed
		r.mu.Unlock()
		select {
		case <-ch:
		case <-wch:
			r.expire(rec)
			r.mu.Lock()
			res := r.resultLocked(rec)
			r.mu.Unlock()
			return res
		}
	}
}

func (r *Registry) register(s collect.CallSpec) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec := &Record{ID: len(r.recs) + 1, Spec: copySpec(s), State: StRegistered, Registered: r.o.Clock.Now(), changed: make(chan struct{})}
	rec.Pidfd.State = PidfdNone
	if r.mode == ModePidfd {
		rec.Pidfd.State = PidfdUnpublished
	}
	rec.Req = Request{Mode: r.mode, State: ReqNone}
	for i := range rec.Desc {
		rec.Desc[i].State = DescNone
	}
	r.recs = append(r.recs, rec)
	if r.refusal != "" {
		rec.State = StRefused
		r.failLocked(rec, "refused: "+r.refusal)
		r.publishLocked(rec, "refused")
		return rec
	}
	rec.Tasks = 1 // the start task, counted before it starts
	r.publishLocked(rec, "registered")
	return rec
}

func copySpec(s collect.CallSpec) collect.CallSpec {
	s.Args = append([]string(nil), s.Args...)
	s.Env = append(s.Env[:0:0], s.Env...)
	return s
}

// env is the inherited environment with the tool's variables set last
// (os/exec keeps the last value of a name).
func env(s collect.CallSpec) []string {
	out := os.Environ()
	for _, e := range s.Env {
		out = append(out, e.Name+"="+e.Value)
	}
	return out
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	if s := err.Error(); s != "" {
		return s
	}
	return "error"
}

func (r *Registry) pipe(id, n int) (*os.File, *os.File, error) {
	if r.s.Pipe != nil {
		return r.s.Pipe(id, n)
	}
	return os.Pipe()
}

func (r *Registry) closeFile(id, d int, f *os.File) error {
	if r.s.CloseFile != nil {
		return r.s.CloseFile(id, d, f)
	}
	return f.Close()
}

// startTask is the start task and the only reaper (§12.14 items 4 and 5).
func (r *Registry) startTask(rec *Record) {
	defer func() {
		if v := recover(); v != nil {
			r.mu.Lock()
			rec.Class, rec.Phase = "C", "std-panic"
			r.failLocked(rec, fmt.Sprintf("panic in the start task: %v", v))
			rec.Tasks--
			r.publishLocked(rec, "start task panicked")
			r.mu.Unlock()
		}
	}()
	id := rec.ID
	r.mu.Lock()
	rec.State = StStarting
	r.publishLocked(rec, "starting")
	r.mu.Unlock()

	outR, outW, err := r.pipe(id, 0)
	if err != nil {
		r.mu.Lock()
		rec.State, rec.Class, rec.Phase, rec.StartErr = StStartFailed, "A", "pipe", "stdout pipe: "+errText(err)
		r.noProcessLocked(rec)
		r.failLocked(rec, "stdout pipe: "+errText(err))
		rec.Tasks--
		r.publishLocked(rec, "start failed: stdout pipe")
		r.mu.Unlock()
		return
	}
	r.mu.Lock()
	rec.Desc[StdoutR] = Desc{State: DescOpen, Holder: "start task"}
	rec.Desc[StdoutW] = Desc{State: DescOpen, Holder: "start task"}
	r.publishLocked(rec, "stdout pipe built")
	r.mu.Unlock()
	errR, errW, err := r.pipe(id, 1)
	if err != nil {
		// Half built: the first pipe's two ends are closed once each by
		// the start task; no read task, no Start (§12.14 item 4).
		r.closeDesc(rec, StdoutR, outR)
		r.closeDesc(rec, StdoutW, outW)
		r.mu.Lock()
		rec.State, rec.Class, rec.Phase, rec.StartErr = StStartFailed, "A", "pipe", "stderr pipe: "+errText(err)
		r.noProcessLocked(rec)
		r.failLocked(rec, "stderr pipe: "+errText(err))
		rec.Tasks--
		r.publishLocked(rec, "start failed: stderr pipe")
		r.mu.Unlock()
		return
	}
	r.mu.Lock()
	rec.Desc[StderrR] = Desc{State: DescOpen, Holder: "start task"}
	rec.Desc[StderrW] = Desc{State: DescOpen, Holder: "start task"}
	// The hand-over: both pipes exist, so both read ends go to their read
	// tasks, counted before they start.
	rec.Desc[StdoutR].Holder, rec.Desc[StderrR].Holder = "read task", "read task"
	rec.Out[0].Started, rec.Out[1].Started = true, true
	rec.Tasks += 2
	r.publishLocked(rec, "read ends handed over")
	r.mu.Unlock()
	go r.readTask(rec, 0, outR)
	go r.readTask(rec, 1, errR)
	if r.s.AfterHandover != nil {
		r.s.AfterHandover(id, outW, errW)
	}

	cmd := exec.Command(rec.Spec.Exe, rec.Spec.Args...)
	cmd.Dir = rec.Spec.Dir
	cmd.Env = env(rec.Spec)
	cmd.Stdout, cmd.Stderr = outW, errW
	if r.s.Prepare != nil {
		r.s.Prepare(id, cmd)
	}
	// §12.6 A ②: Start returns before any descriptor or process when the
	// lookup failed or the path is empty.
	pre := cmd.Err != nil || cmd.Path == ""
	err = cmd.Start()
	r.mu.Lock()
	if err != nil {
		rec.State, rec.StartErr = StStartFailed, errText(err)
		if pre {
			rec.Class, rec.Phase = "A", "precondition"
		} else {
			rec.Class, rec.Phase = "C", "start"
		}
		r.failLocked(rec, fmt.Sprintf("start failed (class %s, %s): %s", rec.Class, rec.Phase, errText(err)))
		r.publishLocked(rec, "start failed")
	} else {
		rec.State, rec.cmd, rec.proc, rec.Pid = StStarted, cmd, cmd.Process, cmd.Process.Pid
		r.publishLocked(rec, "started")
	}
	r.mu.Unlock()
	r.closeDesc(rec, StdoutW, outW)
	r.closeDesc(rec, StderrW, errW)
	if err != nil {
		// Start failed: no Wait (§12.14 item 5); a C class's internal child
		// stays unknown and nothing is done about it.
		r.mu.Lock()
		r.noProcessLocked(rec)
		rec.Tasks--
		r.publishLocked(rec, "start task done")
		r.mu.Unlock()
		return
	}
	if r.mode == ModePidfd {
		fd, perr := r.pidfdOpen(id, rec.Pid)
		r.mu.Lock()
		if perr != nil {
			rec.Pidfd.State, rec.Pidfd.OpenErr = PidfdOpenFailed, errText(perr)
			r.failLocked(rec, "pidfd open: "+errText(perr))
			r.publishLocked(rec, "pidfd open failed")
		} else {
			rec.Pidfd.State, rec.Pidfd.FD = PidfdOpen, fd
			r.publishLocked(rec, "pidfd open")
		}
		r.mu.Unlock()
	}
	r.hook(HookBeforeWait, id)
	werr := cmd.Wait()
	r.mu.Lock()
	rec.WaitCalled, rec.WaitErr = true, errText(werr)
	rec.State = StWaited
	if ps := cmd.ProcessState; ps != nil {
		rec.Class, rec.Exit = "B", &Exit{Code: ps.ExitCode(), Exited: ps.Exited(), String: ps.String()}
	} else {
		rec.Class, rec.Phase = "C", "wait"
		r.failLocked(rec, "wait returned without a process state (class C, wait): "+errText(werr))
	}
	sealed := false
	if rec.Pidfd.State == PidfdOpen {
		rec.Pidfd.State, sealed = PidfdClosing, true
	}
	r.publishLocked(rec, "waited")
	r.mu.Unlock()
	if sealed {
		r.closePidfd(rec)
	}
	r.mu.Lock()
	rec.Tasks--
	r.publishLocked(rec, "start task done")
	r.mu.Unlock()
}

// closeDesc closes one descriptor once and publishes the real result, then
// lets the seam append an injection (§12.14 item 12).
func (r *Registry) closeDesc(rec *Record, d int, f *os.File) {
	r.mu.Lock()
	rec.Desc[d].State = DescClosing
	r.publishLocked(rec, DescNames[d]+" closing")
	r.mu.Unlock()
	err := r.closeFile(rec.ID, d, f)
	r.mu.Lock()
	rec.Desc[d].State, rec.Desc[d].CloseErr, rec.Desc[d].Late = DescClosed, errText(err), r.lateLocked()
	if err != nil {
		r.failLocked(rec, "close "+DescNames[d]+": "+errText(err))
	}
	r.publishLocked(rec, DescNames[d]+" closed")
	r.mu.Unlock()
	if r.s.InjectClose != nil {
		if e := r.s.InjectClose(rec.ID, d); e != nil {
			r.mu.Lock()
			rec.Desc[d].Injected = append(rec.Desc[d].Injected, errText(e))
			rec.Injected = append(rec.Injected, "close "+DescNames[d]+": "+errText(e))
			r.publishLocked(rec, DescNames[d]+" injected")
			r.mu.Unlock()
		}
	}
}

// readTask reads one output until EOF, a read error or an overflow, then
// closes its read end once (§12.14 item 4).
func (r *Registry) readTask(rec *Record, i int, f *os.File) {
	limit := r.o.Limit
	var data []byte
	var total int64
	var eof, overflow bool
	var rerr error
	chunk := make([]byte, 32<<10)
	for {
		var n int
		var err error
		if r.s.Read != nil {
			n, err = r.s.Read(rec.ID, i, f, chunk)
		} else {
			n, err = f.Read(chunk)
		}
		if n > 0 {
			total += int64(n)
			if total > limit {
				overflow = true
				if keep := limit - int64(len(data)); keep > 0 {
					data = append(data, chunk[:keep]...)
				}
				break
			}
			data = append(data, chunk[:n]...)
		}
		if errors.Is(err, io.EOF) {
			eof = true
			break
		}
		if err != nil {
			rerr = err
			break
		}
	}
	d := StdoutR
	if i == 1 {
		d = StderrR
	}
	r.mu.Lock()
	o := &rec.Out[i]
	o.Done, o.Bytes, o.Data, o.EOF, o.ReadErr, o.Overflow, o.Late = true, total, data, eof, errText(rerr), overflow, r.lateLocked()
	if rerr != nil {
		r.failLocked(rec, "read "+StreamNames[i]+": "+errText(rerr))
	}
	if overflow {
		r.failLocked(rec, fmt.Sprintf("overflow on %s: more than %d bytes", StreamNames[i], limit))
	}
	r.publishLocked(rec, StreamNames[i]+" read stopped")
	r.mu.Unlock()
	r.closeDesc(rec, d, f)
	r.mu.Lock()
	rec.Tasks--
	r.publishLocked(rec, StreamNames[i]+" read task done")
	r.mu.Unlock()
}

// RealPidfdOpen is the real pidfd_open(pid, 0) (Linux; an error
// elsewhere), for seams that delay and then perform the real operation.
func RealPidfdOpen(pid int) (int, error) { return pidfdOpen(pid) }

// RealSendKill is the real pidfd_send_signal(fd, SIGKILL) (Linux; an error
// elsewhere), for seams that delay and then perform the real operation.
func RealSendKill(fd int) error { return pidfdSendKill(fd) }

func (r *Registry) pidfdOpen(id, pid int) (int, error) {
	if r.s.PidfdOpen != nil {
		return r.s.PidfdOpen(id, pid)
	}
	return pidfdOpen(pid)
}

func (r *Registry) sendSignal(id, fd int) error {
	if r.s.SendSignal != nil {
		return r.s.SendSignal(id, fd)
	}
	return pidfdSendKill(fd)
}

func (r *Registry) closeFD(id, fd int) error {
	if r.s.CloseFD != nil {
		return r.s.CloseFD(id, fd)
	}
	return closeFD(fd)
}

func (r *Registry) kill(id int, p *os.Process) error {
	if r.s.Kill != nil {
		return r.s.Kill(id, p)
	}
	return p.Kill()
}

// closePidfd is rule 3 and 4 of §12.14 item 9: the start task, the only
// closer, waits outside the lock while a request is reserved or running,
// then closes once.
func (r *Registry) closePidfd(rec *Record) {
	r.hook(HookBeforePidfdClose, rec.ID)
	r.mu.Lock()
	for rec.Req.State == ReqReserved || rec.Req.State == ReqRunning {
		ch := rec.changed
		r.mu.Unlock()
		<-ch
		r.mu.Lock()
	}
	fd := rec.Pidfd.FD
	r.mu.Unlock()
	err := r.closeFD(rec.ID, fd)
	r.mu.Lock()
	rec.Pidfd.State, rec.Pidfd.CloseErr = PidfdClosed, errText(err)
	rec.Pidfd.Closes++
	if err != nil {
		r.failLocked(rec, "pidfd close: "+errText(err))
	}
	r.publishLocked(rec, "pidfd closed")
	r.mu.Unlock()
}

// expire is the parent's decision at W (§12.14 item 8): the call has
// failed; when its reap is not published the platform's termination is
// decided. Only the direct child is ever terminated.
func (r *Registry) expire(rec *Record) {
	r.mu.Lock()
	if rec.Tasks == 0 {
		r.mu.Unlock()
		return
	}
	if !rec.Expired {
		rec.Expired = true
		r.failLocked(rec, "not completed before W")
		r.publishLocked(rec, "expired at W")
	}
	need := rec.State != StWaited && rec.State != StStartFailed
	r.mu.Unlock()
	if !need {
		return
	}
	if r.mode == ModeNone {
		r.mu.Lock()
		rec.Unknown = append(rec.Unknown, "no termination request on this platform (darwin: os/wait_unimp.go); the tool may outlive the test process")
		r.publishLocked(rec, "cleanup unknown")
		r.mu.Unlock()
		return
	}
	r.hook(HookExpiryDecided, rec.ID)
	r.mu.Lock()
	defer r.mu.Unlock()
	ok, why, retry := r.reserveLocked(rec)
	switch {
	case ok:
	case retry:
		rec.Tasks++
		r.publishLocked(rec, "expiry task started: "+why)
		go r.expiryTask(rec)
	default:
		r.notAdmittedLocked(rec, why)
	}
}

func (r *Registry) notAdmittedLocked(rec *Record, why string) {
	rec.NotAdmit = append(rec.NotAdmit, why)
	r.failLocked(rec, "termination not admitted: "+why)
	if rec.State != StWaited && rec.State != StStartFailed {
		rec.Unknown = append(rec.Unknown, "no termination was sent and the reap is not published")
	}
	r.publishLocked(rec, "termination not admitted")
}

// Reserve is the reservation of rule 1 of §12.14 item 9 for a registered
// call, under the lock; it is the parent's entry, exported so the
// controls can try it with an old snapshot (rule 6). It reports whether a
// request was reserved and started, and why not.
func (r *Registry) Reserve(id int) (bool, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id < 1 || id > len(r.recs) {
		return false, "no such call"
	}
	ok, why, _ := r.reserveLocked(r.recs[id-1])
	return ok, why
}

// reserveLocked decides one reservation (rule 1): the clock re-read in
// [W, D), the pidfd open (or, on Windows, the process published and not
// waited) and no request yet. On success the request owns the pidfd's use
// and its task is counted and started.
func (r *Registry) reserveLocked(rec *Record) (ok bool, why string, retry bool) {
	now := r.o.Clock.Now()
	if now.Before(r.o.W) || !now.Before(r.o.D) {
		return false, "outside the request window [W, D)", false
	}
	if rec.Req.State != ReqNone {
		return false, "a request exists (" + rec.Req.State + ")", false
	}
	switch r.mode {
	case ModePidfd:
		switch rec.Pidfd.State {
		case PidfdOpen:
			rec.Req.State, rec.Req.FD = ReqReserved, rec.Pidfd.FD
			fd := rec.Pidfd.FD
			rec.Tasks++
			r.publishLocked(rec, "request reserved")
			go r.requestTask(rec, fd, nil)
			return true, "", false
		case PidfdUnpublished:
			return false, "the pidfd is not published yet", true
		case PidfdOpenFailed:
			return false, "the pidfd could not be opened", false
		case PidfdClosing, PidfdClosed:
			return false, "the pidfd is sealed (" + rec.Pidfd.State + ")", false
		default:
			return false, "no process (" + rec.State + ")", false
		}
	case ModeKill:
		switch rec.State {
		case StStarted:
			rec.Req.State = ReqReserved
			p := rec.proc
			rec.Tasks++
			r.publishLocked(rec, "request reserved")
			go r.requestTask(rec, 0, p)
			return true, "", false
		case StRegistered, StStarting:
			return false, "the process is not published yet", true
		default:
			return false, "the process is " + rec.State, false
		}
	}
	return false, "no termination on this platform", false
}

// expiryTask retries the reservation on every change of the record while
// the window lasts (§12.14 item 9, the parent's request decision).
func (r *Registry) expiryTask(rec *Record) {
	dch, stop := r.o.Clock.Timer(r.o.D)
	defer stop()
	r.mu.Lock()
	defer r.mu.Unlock()
	for {
		ok, why, retry := r.reserveLocked(rec)
		if ok || !retry {
			if !ok {
				r.notAdmittedLocked(rec, why)
			}
			rec.Tasks--
			r.publishLocked(rec, "expiry task done")
			return
		}
		ch := rec.changed
		r.mu.Unlock()
		select {
		case <-ch:
			r.mu.Lock()
		case <-dch:
			r.mu.Lock()
			if ok, _, _ := r.reserveLocked(rec); !ok {
				r.notAdmittedLocked(rec, "the window ended (D) before a reservation: "+why)
			}
			rec.Tasks--
			r.publishLocked(rec, "expiry task done")
			return
		}
	}
}

// requestTask is rule 5: the window is re-read under the lock; outside it
// the request is abandoned without any call, inside it one call is made.
func (r *Registry) requestTask(rec *Record, fd int, p *os.Process) {
	r.hook(HookRequestStart, rec.ID)
	r.mu.Lock()
	now := r.o.Clock.Now()
	if now.Before(r.o.W) || !now.Before(r.o.D) {
		rec.Req.State, rec.Req.Late = ReqAbandoned, r.lateLocked()
		r.failLocked(rec, "termination request abandoned: the window [W, D) ended before it ran")
		if rec.State != StWaited {
			rec.Unknown = append(rec.Unknown, "the termination request was abandoned; no termination was sent")
		}
		rec.Tasks--
		r.publishLocked(rec, "request abandoned")
		r.mu.Unlock()
		return
	}
	rec.Req.State = ReqRunning
	rec.Req.Sends++
	r.publishLocked(rec, "request running")
	r.mu.Unlock()
	var err error
	if r.mode == ModePidfd {
		err = r.sendSignal(rec.ID, fd)
	} else {
		err = r.kill(rec.ID, p)
	}
	r.mu.Lock()
	rec.Req.State, rec.Req.Err, rec.Req.Late = ReqDone, errText(err), r.lateLocked()
	if err != nil {
		r.failLocked(rec, "termination request: "+errText(err))
	}
	rec.Tasks--
	r.publishLocked(rec, "request done")
	r.mu.Unlock()
}

// successLocked is §12.14 item 11 for one record.
func (r *Registry) successLocked(rec *Record) bool {
	if rec.Tasks != 0 || rec.State != StWaited || rec.Class != "B" || rec.Exit == nil || rec.Exit.Code != 0 {
		return false
	}
	for _, d := range rec.Desc {
		if d.State != DescClosed || d.CloseErr != "" {
			return false
		}
	}
	for _, o := range rec.Out {
		if !o.Done || !o.EOF || o.ReadErr != "" || o.Overflow {
			return false
		}
	}
	if r.mode == ModePidfd && (rec.Pidfd.State != PidfdClosed || rec.Pidfd.CloseErr != "") {
		return false
	}
	if rec.Req.State != ReqNone || rec.Expired || len(rec.Unknown) > 0 || len(rec.NotAdmit) > 0 || len(rec.Failures) > 0 || len(rec.Injected) > 0 {
		return false
	}
	for _, e := range rec.Events {
		if e.Late {
			return false
		}
	}
	return true
}

func (r *Registry) snapshotLocked(rec *Record) Snapshot {
	s := Snapshot{ID: rec.ID, Kind: rec.Spec.Kind.String(), Args: append([]string(nil), rec.Spec.Args...), Dir: rec.Spec.Dir,
		State: rec.State, Class: rec.Class, Phase: rec.Phase, StartErr: rec.StartErr, WaitCalled: rec.WaitCalled, WaitErr: rec.WaitErr,
		Pid: rec.Pid, Desc: rec.Desc, Out: rec.Out, Pidfd: rec.Pidfd, Req: rec.Req, Expired: rec.Expired,
		NotAdmit: append([]string(nil), rec.NotAdmit...), Unknown: append([]string(nil), rec.Unknown...),
		Failures: append([]string(nil), rec.Failures...), Injected: append([]string(nil), rec.Injected...),
		Events: append([]Event(nil), rec.Events...), Tasks: rec.Tasks, Terminal: rec.Tasks == 0, Mode: r.mode}
	if rec.Exit != nil {
		e := *rec.Exit
		s.Exit = &e
	}
	for i := range s.Desc {
		s.Desc[i].Injected = append([]string(nil), rec.Desc[i].Injected...)
	}
	for i := range s.Out {
		s.Out[i].Data = append([]byte(nil), rec.Out[i].Data...)
	}
	s.Success = r.successLocked(rec)
	return s
}

// resultLocked is the published copy as the collector's CallResult. The
// outputs are given only when their read task published them (item 7).
func (r *Registry) resultLocked(rec *Record) collect.CallResult {
	s := r.snapshotLocked(rec)
	res := collect.CallResult{Spec: copySpec(rec.Spec), Started: rec.State == StStarted || rec.State == StWaited, StartErr: rec.StartErr,
		Reaped: rec.State == StWaited && rec.Class == "B", StdoutEOF: rec.Out[0].Done && rec.Out[0].EOF, StderrEOF: rec.Out[1].Done && rec.Out[1].EOF,
		Failures: r.problemsLocked(rec), Injected: append([]string(nil), rec.Injected...), Start: rec.Registered, Control: s}
	if rec.Exit != nil {
		res.ExitCode = rec.Exit.Code
	} else {
		res.ExitCode = -1
	}
	if rec.Out[0].Done {
		res.Stdout = append([]byte(nil), rec.Out[0].Data...)
	}
	if rec.Out[1].Done {
		res.Stderr = append([]byte(nil), rec.Out[1].Data...)
	}
	if n := len(rec.Events); n > 0 {
		res.End = rec.Events[n-1].At
	}
	return res
}

// problemsLocked is every reason a record does not meet item 11, in a
// stable order.
func (r *Registry) problemsLocked(rec *Record) []string {
	out := append([]string(nil), rec.Failures...)
	if rec.Tasks > 0 {
		out = append(out, "not terminal: "+r.pendingLocked(rec))
	}
	for _, u := range rec.Unknown {
		out = append(out, "cleanup unknown: "+u)
	}
	if rec.Tasks == 0 && rec.State == StWaited && rec.Class == "B" && rec.Exit != nil && rec.Exit.Code != 0 {
		out = append(out, fmt.Sprintf("exit code %d", rec.Exit.Code))
	}
	for i, o := range rec.Out {
		if o.Done && !o.EOF && o.ReadErr == "" && !o.Overflow {
			out = append(out, StreamNames[i]+" stopped without EOF")
		}
	}
	for _, e := range rec.Events {
		if e.Late {
			out = append(out, "late result: "+e.What)
			break
		}
	}
	return out
}

// pendingLocked names what a record is still waiting for.
func (r *Registry) pendingLocked(rec *Record) string {
	var p []string
	switch rec.State {
	case StRegistered, StStarting:
		p = append(p, "start "+rec.State)
	case StStarted:
		p = append(p, "reap not published")
	}
	for i, d := range rec.Desc {
		if d.State == DescOpen || d.State == DescClosing {
			p = append(p, DescNames[i]+" "+d.State)
		}
	}
	for i, o := range rec.Out {
		if o.Started && !o.Done {
			p = append(p, StreamNames[i]+" read in progress")
		}
	}
	if rec.Pidfd.State == PidfdOpen || rec.Pidfd.State == PidfdClosing {
		p = append(p, "pidfd "+rec.Pidfd.State)
	}
	if rec.Req.State == ReqReserved || rec.Req.State == ReqRunning {
		p = append(p, "request "+rec.Req.State)
	}
	if len(p) == 0 {
		p = append(p, fmt.Sprintf("%d tasks", rec.Tasks))
	}
	return strings.Join(p, ", ")
}

// Snapshot returns the published copy of one call.
func (r *Registry) Snapshot(id int) (Snapshot, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id < 1 || id > len(r.recs) {
		return Snapshot{}, false
	}
	return r.snapshotLocked(r.recs[id-1]), true
}

// Snapshots returns the published copies of every call, in order.
func (r *Registry) Snapshots() []Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Snapshot, 0, len(r.recs))
	for _, rec := range r.recs {
		out = append(out, r.snapshotLocked(rec))
	}
	return out
}

// Changed returns a channel closed at the registry's next publication.
func (r *Registry) Changed() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.changed
}

// Terminal is the registry as a closeout TerminalSource (§12.14 item 13):
// done when every record's tasks have returned.
func (r *Registry) Terminal() (bool, <-chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.recs {
		if rec.Tasks > 0 {
			return false, r.changed
		}
	}
	return true, nil
}

// Problems are the registry's failures and pending items for the closeout
// (§12.14 item 13): a record not terminal is pending; a terminal record
// that does not meet item 11 is a failure, its reasons listed.
func (r *Registry) Problems() (failures, pending []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.recs {
		name := fmt.Sprintf("call %d %s %s", rec.ID, rec.Spec.Kind, rec.Spec.Config)
		if rec.Tasks > 0 {
			pending = append(pending, name+": not terminal: "+r.pendingLocked(rec))
			continue
		}
		if !r.successLocked(rec) {
			ps := r.problemsLocked(rec)
			if len(ps) == 0 {
				ps = []string{"not successful (" + rec.State + ", class " + rec.Class + ")"}
			}
			failures = append(failures, name+": "+strings.Join(ps, "; "))
		}
	}
	sort.Strings(failures)
	return failures, pending
}

// WaitTerminal waits until the call is terminal or the channel closes,
// whichever comes first, and reports whether it is terminal. It is for
// the outer layer and tests; the parent never waits for it.
func (r *Registry) WaitTerminal(id int, give <-chan struct{}) bool {
	for {
		r.mu.Lock()
		if id < 1 {
			r.mu.Unlock()
			return false
		}
		if id > len(r.recs) {
			// Not registered yet: wait for the registry's next change.
			ch := r.changed
			r.mu.Unlock()
			select {
			case <-ch:
				continue
			case <-give:
				return false
			}
		}
		rec := r.recs[id-1]
		if rec.Tasks == 0 {
			r.mu.Unlock()
			return true
		}
		ch := rec.changed
		r.mu.Unlock()
		select {
		case <-ch:
		case <-give:
			return false
		}
	}
}

// WaitFor waits until cond holds on the call's published copy or the
// channel closes; it reports whether cond held. For the outer layer and
// tests.
func (r *Registry) WaitFor(id int, give <-chan struct{}, cond func(Snapshot) bool) bool {
	for {
		r.mu.Lock()
		if id < 1 {
			r.mu.Unlock()
			return false
		}
		if id > len(r.recs) {
			ch := r.changed
			r.mu.Unlock()
			select {
			case <-ch:
				continue
			case <-give:
				return false
			}
		}
		rec := r.recs[id-1]
		if cond(r.snapshotLocked(rec)) {
			r.mu.Unlock()
			return true
		}
		ch := rec.changed
		r.mu.Unlock()
		select {
		case <-ch:
		case <-give:
			return false
		}
	}
}

// Len is the number of registered calls.
func (r *Registry) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.recs)
}
