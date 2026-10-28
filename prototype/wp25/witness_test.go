package wp25

// The outer gate check's own evidence about an inner run (Review 076;
// Review 077 B1-R2-W). It never decides a scenario's verdict and the
// scenarios never read it: it is how the outer check knows, without the
// inner run's account, which processes the inner run created, whether
// they ended, and whether the evidence itself held. Under a gate check
// the inner run and every process it starts write records, one line each,
// in one write that is checked, to the witness file:
//
//	root     <pid> <start> <pid> inner    the inner test process, first
//	created  <pid> <start> <by>  <what>   its creator, right after the start
//	launched <pid> <start> <pid> <mode>   a helper, once it found its created record
//
// <start> is the identity of the process: the Linux /proc start ticks, the
// macOS sysctl start time, the Windows creation time. A creator reads it
// for the child it holds, a helper reads its own, and the two must agree.
//
// Every process the prototype starts goes through start (TestStartPoints
// checks the start points). A helper does nothing, no work and no child,
// before it has found its creation record and written its own. A failure
// to read an identity, to write or close a record, or to find the creation
// record is raised to the outer check out of band (a signal on Unix, an
// event on Windows), never only told to the file that failed; the process
// that meets it ends what it has just started and stops. The inner test
// process also prints a seal on its standard output: the records it wrote
// and the failures it met.
//
// The outer check rejects the evidence for a raise; a file that cannot be
// read; a record that is malformed, unknown, duplicated or cut, or an
// identity in the wrong form; a missing or different root; a seal that is
// missing, reports a failure or a different number of records; a helper's
// record without its creation record, or a creator that never ran; and on
// Windows a count of created processes, kept by the Job the inner run
// runs in, other than the records'. Apart from the records, it lists the
// processes still alive when the inner run is over (a census: on Linux and
// macOS by this check's tag in their environment, on Windows by its Job)
// before it ends any of them.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	witnessEnv   = "WP25_WITNESS"
	raiseEnv     = "WP25_WITNESS_RAISE" // the outer check's PID (Unix) or event name (Windows)
	injectLogEnv = "WP25_INJECT_LOG"

	// receiptWait bounds a helper's wait for its creation record.
	receiptWait = 10 * time.Second
	maxWitness  = 4 << 20
	maxRecord   = 160
)

var errWitness = errors.New("witness")

type witnessRec struct {
	kind   string // root, created or launched
	pid    int
	start  string // the identity of the process
	by     int    // the writer: the process itself, or its creator
	detail string // "inner", what was created, or the helper's mode
}

// key names one process: its PID and its identity.
func (w witnessRec) key() string { return idKey(w.pid, w.start) }

func idKey(pid int, start string) string { return strconv.Itoa(pid) + "@" + start }

func witnessOn() bool { return os.Getenv(witnessEnv) != "" }
func inHelper() bool  { return os.Getenv(helperEnv) != "" }

// witnessErr describes a failure of the witness without the file's path.
func witnessErr(op string, err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		err = pe.Err
	}
	return fmt.Errorf("%w: %s: %v", errWitness, op, err)
}

// The inner test process's account of its own records, for the seal.
var sealTally struct {
	sync.Mutex
	records, failed int
	first           string
}

// witnessFailed raises a failure of the witness to the outer check, and in
// the inner test process counts it for the seal.
func witnessFailed(err error) error {
	raiseWitness()
	if !inHelper() {
		sealTally.Lock()
		sealTally.failed++
		if sealTally.first == "" {
			sealTally.first = err.Error()
		}
		sealTally.Unlock()
	}
	return err
}

// writeRecord appends one record in one write and checks that all of it
// was written and the file closed. A failure is raised.
func writeRecord(r witnessRec) error {
	line := fmt.Sprintf("%s %d %s %d %s\n", r.kind, r.pid, r.start, r.by, r.detail)
	f, err := os.OpenFile(os.Getenv(witnessEnv), os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return witnessFailed(witnessErr("open", err))
	}
	// Faults of the gate regressions (Review 077 B1-R2-W): a write that
	// really fails, one that writes only part of the record.
	switch {
	case inHelper() && injected("witness-write-fail"),
		!inHelper() && r.kind == "created" && injectOnce("witness-root-write-fail", "writeRecord", os.Getpid()):
		if inHelper() {
			injectHit("writeRecord", os.Getpid())
		}
		_ = f.Close()
	case inHelper() && injected("witness-short-write"):
		injectHit("writeRecord", os.Getpid())
		line = line[:len(line)/2]
	}
	n, werr := f.WriteString(line)
	cerr := f.Close()
	switch {
	case werr != nil:
		return witnessFailed(witnessErr("write", werr))
	case n != len(line) || !strings.HasSuffix(line, "\n"):
		return witnessFailed(fmt.Errorf("%w: short write: %d bytes of the record written", errWitness, n))
	case cerr != nil:
		return witnessFailed(witnessErr("close", cerr))
	}
	if !inHelper() {
		sealTally.Lock()
		sealTally.records++
		sealTally.Unlock()
	}
	return nil
}

// identities holds the identity of each process start read for its
// creation record, so that what a scenario owns names the same process.
var identities sync.Map // *exec.Cmd -> string

func identityOf(cmd *exec.Cmd) string {
	if v, ok := identities.Load(cmd); ok {
		return v.(string)
	}
	return ""
}

// start is the only way the prototype starts a process: cmd.Start, then,
// under a gate check, the creation record, before anything else happens to
// the child. If the record cannot be made, the child is ended at once (its
// creator holds it), the failure is raised, and an error is returned, so
// no caller keeps a child the outer check does not know.
func start(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	if !witnessOn() {
		return nil
	}
	id, err := childIdentity(cmd)
	if err != nil {
		err = witnessFailed(witnessErr("the identity of the child", err))
	} else {
		err = writeRecord(witnessRec{"created", cmd.Process.Pid, id, os.Getpid(), what(cmd)})
	}
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}
	identities.Store(cmd, id)
	return nil
}

// what names a created process in its record: the helper mode, or the
// program's base name.
func what(cmd *exec.Cmd) string {
	w := ""
	for _, kv := range cmd.Env {
		if v, ok := strings.CutPrefix(kv, helperEnv+"="); ok {
			w = v
		}
	}
	if w == "" {
		w = strings.ToLower(strings.TrimSuffix(filepath.Base(cmd.Path), ".exe"))
	}
	if !validDetail(w) {
		return "exec"
	}
	return w
}

// receipt is a helper's part, before anything else it does: read its own
// identity, wait (at most receiptWait) until its creation record is in the
// witness file, and write its own record (unless write is false: a gate
// regression's helper that leaves no record of its own). A failure is
// raised; the helper then stops at once.
func receipt(mode string, write bool) error {
	id, err := selfIdentity()
	if err != nil {
		return witnessFailed(witnessErr("its own identity", err))
	}
	want := fmt.Sprintf("created %d %s ", os.Getpid(), id)
	deadline := time.Now().Add(receiptWait)
	for {
		b, err := os.ReadFile(os.Getenv(witnessEnv))
		if err != nil {
			return witnessFailed(witnessErr("read", err))
		}
		if hasRecord(b, want) {
			break
		}
		if time.Now().After(deadline) {
			return witnessFailed(fmt.Errorf("%w: no creation record within %v", errWitness, receiptWait))
		}
		time.Sleep(2 * time.Millisecond)
	}
	if !write {
		injectHit("receipt", os.Getpid())
		return nil
	}
	return writeRecord(witnessRec{"launched", os.Getpid(), id, os.Getpid(), mode})
}

// hasRecord reports whether b holds a whole line that begins with prefix.
func hasRecord(b []byte, prefix string) bool {
	for _, l := range strings.SplitAfter(string(b), "\n") {
		if strings.HasPrefix(l, prefix) && strings.HasSuffix(l, "\n") {
			return true
		}
	}
	return false
}

// witnessRoot writes the inner test process's own record, first.
func witnessRoot() {
	id, err := selfIdentity()
	if err != nil {
		_ = witnessFailed(witnessErr("its own identity", err))
		return
	}
	_ = writeRecord(witnessRec{"root", os.Getpid(), id, os.Getpid(), "inner"})
}

type seal struct {
	PID     int    `json:"pid"`
	Start   string `json:"start"`
	Records int    `json:"records"`
	Failed  int    `json:"failed"`
	First   string `json:"first,omitempty"`
}

// printSeal prints the inner test process's account of its records.
func printSeal() {
	id, err := selfIdentity()
	if err != nil {
		_ = witnessFailed(witnessErr("its own identity", err))
	}
	sealTally.Lock()
	sl := seal{PID: os.Getpid(), Start: id, Records: sealTally.records, Failed: sealTally.failed, First: sealTally.first}
	sealTally.Unlock()
	b, _ := json.Marshal(sl)
	fmt.Println("WP25WITNESS " + string(b))
}

// injectHit records that an injected fault reached its site: the inner
// test process prints a WP25INJECT line, a helper appends to the inject
// log. Neither is the witness file, so a fault of the witness itself is
// still seen to have acted.
func injectHit(site string, pid int) {
	b, _ := json.Marshal(map[string]any{"fault": os.Getenv(injectEnv), "site": site, "pid": pid})
	if !inHelper() {
		fmt.Println("WP25INJECT " + string(b))
		return
	}
	if path := os.Getenv(injectLogEnv); path != "" {
		if f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			_, _ = f.Write(append(b, '\n'))
			_ = f.Close()
		}
	}
}

var (
	onceMu   sync.Mutex
	onceDone = map[string]bool{}
)

// injectOnce reports whether the named fault is injected and has not acted
// yet in this process; the call that says so records the hit.
func injectOnce(name, site string, pid int) bool { return injectOnceAt(name, name, site, pid) }

// injectOnceAt is injectOnce for one of the sites of a fault that acts at
// several (Review 096 B1-R): each key acts once.
func injectOnceAt(name, key, site string, pid int) bool {
	if !injected(name) {
		return false
	}
	onceMu.Lock()
	defer onceMu.Unlock()
	if onceDone[key] {
		return false
	}
	onceDone[key] = true
	injectHit(site, pid)
	return true
}

// tamperWitness damages the witness file at the end of an inner run, as a
// failure of the file itself would (Review 077 B1-R2-W): the end of the
// last record lost, a record twice, a record of an unknown kind, the file
// gone.
func tamperWitness() {
	path := os.Getenv(witnessEnv)
	if path == "" {
		return
	}
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		return
	}
	lines := strings.SplitAfter(strings.TrimSuffix(string(b), "\n"), "\n")
	last := strings.TrimSuffix(lines[len(lines)-1], "\n") + "\n"
	switch {
	case injected("witness-truncated"):
		injectHit("witness-file", os.Getpid())
		_ = os.Truncate(path, int64(len(b)-1))
	case injected("witness-duplicate"):
		injectHit("witness-file", os.Getpid())
		appendFile(path, last)
	case injected("witness-unknown-kind"):
		injectHit("witness-file", os.Getpid())
		appendFile(path, "bogus"+last[strings.IndexByte(last, ' '):])
	case injected("witness-deleted"):
		injectHit("witness-file", os.Getpid())
		_ = os.Remove(path)
	}
}

func appendFile(path, s string) {
	if f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
		_, _ = f.WriteString(s)
		_ = f.Close()
	}
}

// validIdentity checks an identity's form on this platform: the Linux and
// Windows ones are positive decimal numbers in canonical form, the macOS
// one seconds and six digits of microseconds. "unknown", or anything else,
// is not an identity.
func validIdentity(s string) bool {
	canon := func(d string) bool {
		n, err := strconv.ParseUint(d, 10, 64)
		return err == nil && n > 0 && strconv.FormatUint(n, 10) == d
	}
	if runtime.GOOS == "darwin" {
		sec, usec, ok := strings.Cut(s, ".")
		if !ok || len(usec) != 6 || !canon(sec) {
			return false
		}
		_, err := strconv.ParseUint(usec, 10, 32)
		return err == nil && !strings.ContainsAny(usec, "+-")
	}
	return canon(s)
}

func validDetail(s string) bool {
	if s == "" || len(s) > 32 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '.' || c == '_') {
			return false
		}
	}
	return true
}

func canonPID(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	return n, err == nil && n > 0 && strconv.Itoa(n) == s
}

// parseWitness reads a witness file strictly: every record whole, of a
// known kind, with canonical PIDs, an identity of this platform's form,
// and no record twice. Nothing is skipped.
func parseWitness(b []byte) ([]witnessRec, error) {
	if len(b) > maxWitness {
		return nil, fmt.Errorf("file larger than %d bytes", maxWitness)
	}
	if len(b) > 0 && b[len(b)-1] != '\n' {
		return nil, errors.New("the last record is cut (no newline at the end)")
	}
	var out []witnessRec
	seen := map[string]bool{}
	for _, line := range strings.SplitAfter(string(b), "\n") {
		if line == "" {
			continue
		}
		line = strings.TrimSuffix(line, "\n")
		if len(line) > maxRecord {
			return nil, fmt.Errorf("record longer than %d bytes", maxRecord)
		}
		f := strings.Split(line, " ")
		if len(f) != 5 {
			return nil, fmt.Errorf("malformed record %q", clip(line))
		}
		switch f[0] {
		case "root", "created", "launched":
		default:
			return nil, fmt.Errorf("unknown record kind %q", clip(f[0]))
		}
		pid, ok1 := canonPID(f[1])
		by, ok2 := canonPID(f[3])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("malformed PID in %q", clip(line))
		}
		if !validIdentity(f[2]) {
			return nil, fmt.Errorf("not an identity of this platform in %q", clip(line))
		}
		if !validDetail(f[4]) {
			return nil, fmt.Errorf("malformed detail in %q", clip(line))
		}
		if f[0] != "created" && by != pid {
			return nil, fmt.Errorf("a %s record written by another process: %q", f[0], clip(line))
		}
		r := witnessRec{kind: f[0], pid: pid, start: f[2], by: by, detail: f[4]}
		if seen[r.kind+" "+r.key()] {
			return nil, fmt.Errorf("duplicate record %q", clip(line))
		}
		seen[r.kind+" "+r.key()] = true
		out = append(out, r)
	}
	return out, nil
}

// evidence is what the outer check gathered about one inner run.
type evidence struct {
	file    []byte
	readErr error
	seals   []string // the JSON of the inner run's WP25WITNESS lines
	inner   int      // the inner test process, as the outer check started it
	start   string   // and its identity, as the outer check read it
	raised  bool
	total   int // Windows: processes the Job saw created, the inner one included; -1 elsewhere
}

// judgeWitness checks the witness itself and returns its records and the
// problems that make it invalid, each "<class>: <detail>".
func judgeWitness(e evidence) ([]witnessRec, []string) {
	if e.readErr != nil {
		return nil, []string{"witness-unreadable: " + witnessErr("read", e.readErr).Error()}
	}
	recs, err := parseWitness(e.file)
	if err != nil {
		return nil, []string{"witness-malformed: " + err.Error()}
	}
	var probs []string
	if e.raised {
		probs = append(probs, "witness-raised: a process of the inner run could not make its record")
	}
	var roots, created int
	byRoot := 0
	running := map[int]bool{e.inner: true}
	creation := map[string]bool{}
	for _, r := range recs {
		switch r.kind {
		case "root":
			roots++
			if r.pid != e.inner || r.start != e.start {
				probs = append(probs, fmt.Sprintf("witness-root: a root record of %s, the inner run is %s", r.key(), idKey(e.inner, e.start)))
			}
		case "created":
			created++
			creation[r.key()] = true
		case "launched":
			running[r.pid] = true
		}
		if r.by == e.inner {
			byRoot++
		}
	}
	if roots != 1 {
		probs = append(probs, fmt.Sprintf("witness-root: %d root records, want 1", roots))
	}
	for _, r := range recs {
		switch {
		case r.kind == "launched" && !creation[r.key()]:
			probs = append(probs, fmt.Sprintf("witness-receipt: process %s ran without its creation record", r.key()))
		case r.kind == "created" && !running[r.by]:
			probs = append(probs, fmt.Sprintf("witness-creator: process %s was created by %d, which has no record of its own", r.key(), r.by))
		}
	}
	switch len(e.seals) {
	case 0:
		probs = append(probs, "witness-seal: the inner run printed no seal")
	case 1:
		var sl seal
		switch err := json.Unmarshal([]byte(e.seals[0]), &sl); {
		case err != nil:
			probs = append(probs, "witness-seal: unreadable: "+err.Error())
		case sl.PID != e.inner || sl.Start != e.start:
			probs = append(probs, fmt.Sprintf("witness-seal: printed by %s, the inner run is %s", idKey(sl.PID, sl.Start), idKey(e.inner, e.start)))
		case sl.Failed > 0:
			probs = append(probs, fmt.Sprintf("witness-seal: the inner run met %d failures of its records (first: %s)", sl.Failed, sl.First))
		case sl.Records != byRoot:
			probs = append(probs, fmt.Sprintf("witness-seal: the inner run wrote %d records, the file holds %d of them", sl.Records, byRoot))
		}
	default:
		probs = append(probs, fmt.Sprintf("witness-seal: %d seals, want 1", len(e.seals)))
	}
	if e.total >= 0 && e.total-1 != created {
		probs = append(probs, fmt.Sprintf("witness-count: the Job saw %d processes created besides the inner run, the records %d", e.total-1, created))
	}
	return recs, probs
}

// hasToken reports whether the NUL-separated strings of b include tag.
func hasToken(b []byte, tag string) bool {
	for _, t := range strings.Split(string(b), "\x00") {
		if t == tag {
			return true
		}
	}
	return false
}

// survivor is a process the census found alive after the inner run.
type survivor struct {
	pid   int
	start string
}

func (v survivor) key() string { return idKey(v.pid, v.start) }
