package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"path/filepath"
	"sort"
	"time"
)

// WriterConfig configures the writer: the file system, the record
// directory (which must not exist yet) and the log whose files it writes.
type WriterConfig struct {
	FS  FS
	Dir string
	Log *Log
}

// WriterState is the writer's terminal state.
type WriterState int

// The writer's terminal states.
const (
	WriterSuccess WriterState = iota + 1 // every record write and close and the manifest succeeded
	WriterFailure                        // some real error; Errors has all of them
	WriterExpired                        // expiry exit: D passed or the latch was set at a decision point
)

func (s WriterState) String() string {
	switch s {
	case WriterSuccess:
		return "success"
	case WriterFailure:
		return "failure"
	case WriterExpired:
		return "expired-incomplete"
	}
	return "none"
}

// WriterTerminal is the writer's published terminal state, with the
// qualification of the lock it was published in and every real error
// published until then.
type WriterTerminal struct {
	State  WriterState
	At     time.Time
	Qual   Qualification
	Errors []ChainError
}

// WriteOutcome is the published result of one Write: the bytes asked and
// the bytes the call reported (clamped to the request), and which items the
// write carried (FirstSeq, LastSeq; 0 when none) and whether it carried the
// header line.
type WriteOutcome struct {
	Outcome
	Requested int
	N         int
	Header    bool
	FirstSeq  int
	LastSeq   int
}

// FileSnapshot is the published state of one record file. Bytes and SHA256
// cover exactly the bytes the writes reported as written (the running
// SHA-256 of the file's content as far as this process knows it).
// WrittenThrough is the last item sequence number whose write succeeded.
type FileSnapshot struct {
	Name           string
	Create         *Outcome
	Writes         []WriteOutcome
	Bytes          int64
	SHA256         string
	HeaderWritten  bool
	WrittenThrough int
	Close          *Outcome
	Holding        Holding
}

// ManifestSnapshot is the published state of the manifest.
type ManifestSnapshot struct {
	Create  *Outcome
	Write   *WriteOutcome
	Close   *Outcome
	Bytes   int64
	SHA256  string
	Holding Holding
}

// HolderWriter is the only holder of the record files and the manifest.
const HolderWriter = "writer"

// Holding is the published ownership of one file handle (Review 147
// B-147-2). From the moment Create returns a handle until a Close returns
// nil, the writer holds the object itself (not only a result about it):
//
//   - Holder is HolderWriter while release is not confirmed, "" before the
//     handle exists and after a confirmed release;
//   - CloseStarted says the one Close was started; CloseReturned that it
//     returned, with its real result in CloseErr;
//   - Released is true only when Close returned nil.
//
// After an expiry exit no Close is started, and after a Close error the
// release is unknown and the Close is not repeated; in both cases the
// handle stays held by the writer until the process ends. Neither the GC
// nor the process exit counts as a confirmed release.
type Holding struct {
	Holder        string
	CloseStarted  bool
	CloseReturned bool
	CloseErr      error
	Released      bool
}

// Held reports whether the handle exists and is not confirmed released.
func (h Holding) Held() bool { return h.Holder != "" }

// WriterSnapshot is a copy of the writer's record in the outer registry.
// A nil Outcome means the operation was not attempted (or has not
// returned).
type WriterSnapshot struct {
	Dir      string
	Started  bool
	Mkdir    *Outcome
	Files    []FileSnapshot
	Manifest ManifestSnapshot
	Terminal *WriterTerminal
	Errors   []ChainError
}

// Writer is the unique writer task (plan §6.4 责任). It is registered by the
// parent before m.Run and then started. It alone creates, writes and closes
// the record directory, the record files and the manifest; every external
// call is outside every lock and is preceded by a deadline re-check under
// the outer mutex (admit). It does not wake for D by itself: while it waits
// for log items it holds nothing new, and at its next decision point after D
// it takes the expiry exit. If it is stuck in an external call it keeps its
// handles, its unwritten items and its running hashes; nobody waits for it.
type Writer struct {
	o     *Outer
	fs    FS
	dir   string
	log   *Log
	names []string
	idx   map[string]int

	// Published state, guarded by o.mu.
	started  bool
	mkdir    *Outcome
	files    []FileSnapshot
	manifest ManifestSnapshot
	terminal *WriterTerminal
	errs     []ChainError

	// Task state, used only by the writer goroutine.
	task writerTask
}

type writerTask struct {
	// handles keep every record file object until its Close returned nil;
	// a handle whose Close failed or never started stays here.
	handles []File
	open    []bool
	// manifest keeps the manifest object from Create until its Close
	// returned nil.
	manifest File
	hashes   []hash.Hash
	bytes    []int64
	pending  [][]byte // bytes not yet written: the header line, then item lines
	header   []bool   // pending starts with the header line
	first    []int    // first item sequence number in pending, 0 when none
	last     []int    // last item sequence number in pending, 0 when none
	next     int      // next item sequence number to take from the log
	failed   bool     // some real error happened
}

var emptySHA = hex.EncodeToString(sha256.New().Sum(nil))

// RegisterWriter registers the unique writer under the outer mutex. The
// record files are the log's files.
func (o *Outer) RegisterWriter(cfg WriterConfig) (*Writer, error) {
	if cfg.FS == nil || cfg.Log == nil || cfg.Dir == "" {
		return nil, errors.New("chain: writer needs an FS, a directory and a log")
	}
	names := cfg.Log.Files()
	n := len(names)
	w := &Writer{
		o:     o,
		fs:    cfg.FS,
		dir:   cfg.Dir,
		log:   cfg.Log,
		names: names,
		idx:   map[string]int{},
		files: make([]FileSnapshot, n),
		task: writerTask{
			handles: make([]File, n),
			open:    make([]bool, n),
			hashes:  make([]hash.Hash, n),
			bytes:   make([]int64, n),
			pending: make([][]byte, n),
			header:  make([]bool, n),
			first:   make([]int, n),
			last:    make([]int, n),
			next:    1,
		},
	}
	for i, name := range names {
		w.idx[name] = i
		w.files[i].Name = name
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.writer != nil {
		return nil, errors.New("chain: a writer is already registered")
	}
	o.writer = w
	o.publishLocked()
	return w, nil
}

// Start starts the writer task, once.
func (w *Writer) Start() error {
	o := w.o
	o.mu.Lock()
	if w.started {
		o.mu.Unlock()
		return errors.New("chain: writer already started")
	}
	w.started = true
	o.publishLocked()
	o.mu.Unlock()
	go w.run()
	return nil
}

// Snapshot returns a copy of the writer's published record.
func (w *Writer) Snapshot() WriterSnapshot {
	w.o.mu.Lock()
	defer w.o.mu.Unlock()
	return w.snapshotLocked()
}

func (w *Writer) snapshotLocked() WriterSnapshot {
	s := WriterSnapshot{
		Dir:     w.dir,
		Started: w.started,
		Mkdir:   copyOutcome(w.mkdir),
		Errors:  append([]ChainError(nil), w.errs...),
		Manifest: ManifestSnapshot{
			Create:  copyOutcome(w.manifest.Create),
			Close:   copyOutcome(w.manifest.Close),
			Bytes:   w.manifest.Bytes,
			SHA256:  w.manifest.SHA256,
			Holding: w.manifest.Holding,
		},
	}
	if w.manifest.Write != nil {
		c := *w.manifest.Write
		s.Manifest.Write = &c
	}
	for _, f := range w.files {
		f.Create = copyOutcome(f.Create)
		f.Close = copyOutcome(f.Close)
		f.Writes = append([]WriteOutcome(nil), f.Writes...)
		s.Files = append(s.Files, f)
	}
	if w.terminal != nil {
		t := *w.terminal
		t.Errors = append([]ChainError(nil), t.Errors...)
		s.Terminal = &t
	}
	return s
}

func (w *Writer) addErrLocked(site string, err error) {
	if err != nil {
		w.errs = append(w.errs, ChainError{Site: site, Err: err})
	}
}

// admit is the decision point before a new external call: under the outer
// mutex it reads the clock and checks the latch. Past D or with the latch
// set it takes the expiry exit (publishes expired-incomplete, late) and
// returns false; the caller then stops without any further external call.
func (w *Writer) admit() bool {
	o := w.o
	o.mu.Lock()
	defer o.mu.Unlock()
	now, q := o.checkLocked()
	if q == Timely {
		return true
	}
	if w.terminal == nil {
		w.terminal = &WriterTerminal{State: WriterExpired, At: now, Qual: Late, Errors: append([]ChainError(nil), w.errs...)}
		o.publishLocked()
	}
	return false
}

// pub publishes one result under the outer mutex; its qualification is
// decided at this lock's decision point. When term is not zero, the
// terminal state is published in the same lock (same reading and
// qualification).
func (w *Writer) pub(term WriterState, record func(now time.Time, q Qualification)) {
	o := w.o
	o.mu.Lock()
	defer o.mu.Unlock()
	now, q := o.checkLocked()
	record(now, q)
	if term != 0 && w.terminal == nil {
		w.terminal = &WriterTerminal{State: term, At: now, Qual: q, Errors: append([]ChainError(nil), w.errs...)}
	}
	o.publishLocked()
}

func failureIf(cond bool) WriterState {
	if cond {
		return WriterFailure
	}
	return 0
}

func (t *writerTask) anyOpen() bool {
	for _, open := range t.open {
		if open {
			return true
		}
	}
	return false
}

func (w *Writer) run() {
	t := &w.task
	if !w.admit() {
		return
	}
	err := w.fs.Mkdir(w.dir)
	if err != nil {
		t.failed = true
	}
	w.pub(failureIf(err != nil), func(now time.Time, q Qualification) {
		w.mkdir = &Outcome{Err: err, At: now, Qual: q}
		w.addErrLocked("mkdir "+w.dir, err)
	})
	if err != nil {
		return
	}
	if !w.createAll() || !w.writeUntilSeal() || !w.closeAll() || t.failed {
		return
	}
	w.writeManifest()
}

// createAll creates every record file exclusively, in order. When the k-th
// create fails, the first k-1 files are closed once each, nothing is
// written, and the writer fails (plan §6.4 半构造).
func (w *Writer) createAll() bool {
	t := &w.task
	for i, name := range w.names {
		if !w.admit() {
			return false
		}
		f, err := w.fs.Create(filepath.Join(w.dir, name))
		if err == nil && f == nil {
			err = errors.New("create returned neither a file nor an error")
		}
		if err != nil {
			t.failed = true
			last := !t.anyOpen()
			w.pub(failureIf(last), func(now time.Time, q Qualification) {
				w.files[i].Create = &Outcome{Err: err, At: now, Qual: q}
				w.addErrLocked("create "+name, err)
			})
			if !last {
				w.closeAll()
			}
			return false
		}
		t.handles[i] = f
		t.open[i] = true
		t.hashes[i] = sha256.New()
		t.pending[i] = HeaderLine(name)
		t.header[i] = true
		w.pub(0, func(now time.Time, q Qualification) {
			w.files[i].Create = &Outcome{At: now, Qual: q}
			w.files[i].SHA256 = emptySHA
			w.files[i].Holding = Holding{Holder: HolderWriter}
		})
	}
	return true
}

// writeUntilSeal copies unwritten items under the log's mutex and writes
// them, file by file, outside every lock, until the log is sealed and every
// item inside the seal is written; it never reads past the seal. A failed
// write stops all writing: the open files are closed once each and the
// writer fails, with the unwritten items left in memory.
func (w *Writer) writeUntilSeal() bool {
	t := &w.task
	for {
		items, sealed, _, changed := w.log.take(t.next)
		for _, it := range items {
			i := w.idx[it.File]
			if t.first[i] == 0 {
				t.first[i] = it.Seq
			}
			t.last[i] = it.Seq
			t.pending[i] = append(t.pending[i], it.Line...)
			t.pending[i] = append(t.pending[i], '\n')
			t.next = it.Seq + 1
		}
		for i := range w.names {
			if len(t.pending[i]) == 0 {
				continue
			}
			if !w.admit() {
				return false
			}
			if !w.writeFile(i) {
				w.closeAll()
				return false
			}
		}
		if sealed {
			return true
		}
		<-changed
	}
}

func (w *Writer) writeFile(i int) bool {
	t := &w.task
	buf := t.pending[i]
	n, err := t.handles[i].Write(buf)
	n, err = normWrite(n, err, len(buf))
	t.hashes[i].Write(buf[:n])
	t.bytes[i] += int64(n)
	sum := hex.EncodeToString(t.hashes[i].Sum(nil))
	total := t.bytes[i]
	wo := WriteOutcome{Requested: len(buf), N: n, Header: t.header[i], FirstSeq: t.first[i], LastSeq: t.last[i]}
	if err != nil {
		t.failed = true
	}
	name := w.names[i]
	w.pub(0, func(now time.Time, q Qualification) {
		f := &w.files[i]
		wo.Outcome = Outcome{Err: err, At: now, Qual: q}
		f.Writes = append(f.Writes, wo)
		f.Bytes = total
		f.SHA256 = sum
		if err == nil {
			if wo.Header {
				f.HeaderWritten = true
			}
			if wo.LastSeq > 0 {
				f.WrittenThrough = wo.LastSeq
			}
		}
		w.addErrLocked("write "+name, err)
	})
	if err != nil {
		return false
	}
	t.pending[i] = nil
	t.header[i] = false
	t.first[i], t.last[i] = 0, 0
	return true
}

// closeAll closes every open record file once, in order, each after its
// own decision point. A Close error means the release is unknown: it is
// recorded as a failure, not retried, and the other files are still closed.
// The last close's publication carries the failure terminal when anything
// has failed. It returns false if it took the expiry exit.
func (w *Writer) closeAll() bool {
	t := &w.task
	closed := false
	for i, name := range w.names {
		if !t.open[i] {
			continue
		}
		if !w.admit() {
			return false
		}
		w.markCloseStarted(&w.files[i].Holding)
		err := t.handles[i].Close()
		// The Close is the one attempt: it is never repeated. Only a nil
		// result confirms the release; otherwise the writer keeps the object.
		t.open[i] = false
		if err == nil {
			t.handles[i] = nil
		}
		closed = true
		if err != nil {
			t.failed = true
		}
		last := !t.anyOpen()
		w.pub(failureIf(last && t.failed), func(now time.Time, q Qualification) {
			w.files[i].Close = &Outcome{Err: err, At: now, Qual: q}
			w.files[i].Holding = closedHolding(err)
			w.addErrLocked("close "+name, err)
		})
	}
	if !closed && t.failed {
		w.pub(WriterFailure, func(time.Time, Qualification) {})
	}
	return true
}

func (w *Writer) manifestContent() []byte {
	t := &w.task
	order := append([]string(nil), w.names...)
	sort.Strings(order)
	var b []byte
	for _, name := range order {
		i := w.idx[name]
		b = append(b, ManifestLine(name, t.bytes[i], hex.EncodeToString(t.hashes[i].Sum(nil)))...)
	}
	return b
}

// writeManifest creates, writes and closes the manifest, each after its own
// decision point. It runs only after every record write and close
// succeeded. A failed write still closes the manifest once.
func (w *Writer) writeManifest() {
	t := &w.task
	content := w.manifestContent()
	if !w.admit() {
		return
	}
	mf, err := w.fs.Create(filepath.Join(w.dir, ManifestName))
	if err == nil && mf == nil {
		err = errors.New("create returned neither a file nor an error")
	}
	if err != nil {
		t.failed = true
		w.pub(WriterFailure, func(now time.Time, q Qualification) {
			w.manifest.Create = &Outcome{Err: err, At: now, Qual: q}
			w.addErrLocked("create "+ManifestName, err)
		})
		return
	}
	// From here the writer holds the manifest object itself: an expiry
	// exit below leaves it held (no Close is started after D), and only a
	// nil Close releases it (Review 147 B-147-2).
	t.manifest = mf
	w.pub(0, func(now time.Time, q Qualification) {
		w.manifest.Create = &Outcome{At: now, Qual: q}
		w.manifest.SHA256 = emptySHA
		w.manifest.Holding = Holding{Holder: HolderWriter}
	})
	if !w.admit() {
		return
	}
	n, err := mf.Write(content)
	n, err = normWrite(n, err, len(content))
	sum := sha256.Sum256(content[:n])
	if err != nil {
		t.failed = true
	}
	w.pub(0, func(now time.Time, q Qualification) {
		w.manifest.Write = &WriteOutcome{Outcome: Outcome{Err: err, At: now, Qual: q}, Requested: len(content), N: n}
		w.manifest.Bytes = int64(n)
		w.manifest.SHA256 = hex.EncodeToString(sum[:])
		w.addErrLocked("write "+ManifestName, err)
	})
	if !w.admit() {
		return
	}
	w.markCloseStarted(&w.manifest.Holding)
	cerr := t.manifest.Close()
	if cerr != nil {
		t.failed = true
	} else {
		t.manifest = nil
	}
	term := WriterSuccess
	if t.failed {
		term = WriterFailure
	}
	w.pub(term, func(now time.Time, q Qualification) {
		w.manifest.Close = &Outcome{Err: cerr, At: now, Qual: q}
		w.manifest.Holding = closedHolding(cerr)
		w.addErrLocked("close "+ManifestName, cerr)
	})
}

// markCloseStarted publishes that the one Close of a held handle starts
// (its result is published when it returns).
func (w *Writer) markCloseStarted(h *Holding) {
	w.o.mu.Lock()
	h.CloseStarted = true
	w.o.publishLocked()
	w.o.mu.Unlock()
}

// closedHolding is a handle's ownership after its one Close returned: a nil
// result confirms the release; an error leaves the release unknown and the
// object held by the writer, never closed again.
func closedHolding(err error) Holding {
	if err == nil {
		return Holding{CloseStarted: true, CloseReturned: true, Released: true}
	}
	return Holding{Holder: HolderWriter, CloseStarted: true, CloseReturned: true, CloseErr: err}
}

// Held lists the handles the writer still holds (created and not confirmed
// released), for the O domain's persistent-holder records.
func (w *Writer) Held() []string {
	w.o.mu.Lock()
	defer w.o.mu.Unlock()
	var out []string
	for _, f := range w.files {
		if f.Holding.Held() {
			out = append(out, f.Name)
		}
	}
	if w.manifest.Holding.Held() {
		out = append(out, ManifestName)
	}
	return out
}

func (s WriterSnapshot) String() string {
	term := "none"
	if s.Terminal != nil {
		term = fmt.Sprintf("%s/%s", s.Terminal.State, s.Terminal.Qual)
	}
	return fmt.Sprintf("writer %s terminal=%s errors=%d", s.Dir, term, len(s.Errors))
}
