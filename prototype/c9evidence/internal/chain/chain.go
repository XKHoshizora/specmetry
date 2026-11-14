// Package chain is the record chain of the C-9 limited evidence tool
// (docs/wp2.5-c9-limited-evidence-plan.md §6.4 – §6.6 and §7.1; the D exit
// and the diagnostic task of docs/wp2.5-implementation-plan.md §12.14
// item 13).
//
// The pieces, and who owns what:
//
//   - Log: the in-memory, append-only record items. Producers (R, F and O
//     tasks, test goroutines, the parent) only append; they never touch a
//     file. At the snapshot point the parent seals it; the writer writes
//     exactly the items up to the seal and stops reading, so the record
//     files have a fixed end. Items appended after the seal stay in memory.
//   - Outer: the outer registry. One mutex; the absolute deadline D; a
//     changed channel closed and replaced on every publication; the expiry
//     latch (set only); the records of the writer, the diagnostic task and
//     the generic outer (O-domain) operations. Every logical decision point
//     reads the clock under this mutex (checkLocked).
//   - Writer: the unique task that creates, writes and closes the record
//     directory, the record files and the manifest. External calls happen
//     outside every lock; each result is published, with its deadline
//     qualification, under the outer mutex. The writer's own publications
//     (bytes, hashes, qualifications, terminal state) are not log items
//     (plan §8.1, Review 144 §5).
//   - the diagnostic task: the unique task that writes the provisional
//     conclusion, the snapshot's failure and pending lines, the record
//     hashes and the terminator line to the diagnostic output (os.Stderr in
//     the tool).
//   - Closeout: the parent's five steps after m.Run returns (§6.4), ending
//     in the final consumption with its unique (a)/(b)/(c) order. It
//     returns the final conclusion and the exit code; it never calls
//     os.Exit and, once it has decided, does nothing else.
//
// Lock order: Outer.mu, then Log.mu (and a FakeClock's mutex). No code in
// this package takes Outer.mu while holding Log.mu. The functions a caller
// hands to Closeout that run under Outer.mu (Provisional, Level1) must not
// call methods of Outer or Op.
//
// Times: D = start + Budget; W = D - CleanupMargin (§12.14). A reading
// exactly equal to D is past the deadline.
package chain

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// The budget of one job and its cleanup margin (implementation plan
// §12.14): D = start + Budget, W = D - CleanupMargin.
const (
	Budget        = 120 * time.Second
	CleanupMargin = 10 * time.Second
)

// Deadlines derives the work deadline W and the absolute deadline D from the
// start instant. With a clock reading that carries a monotonic component
// (time.Now), both keep it.
func Deadlines(start time.Time) (w, d time.Time) {
	d = start.Add(Budget)
	return d.Add(-CleanupMargin), d
}

// Conclusion is one of the four conclusions of a job (plan §7.1), in
// priority order: a smaller value is the worse conclusion.
type Conclusion int

// The conclusions.
const (
	Invalid      Conclusion = 1 // 无效
	Incomplete   Conclusion = 2 // 未完成
	Inconsistent Conclusion = 3 // 与候选不一致
	Consistent   Conclusion = 4 // 与候选一致
)

// Valid reports whether c is one of the four conclusions.
func (c Conclusion) Valid() bool { return c >= Invalid && c <= Consistent }

// Name is the fixed name used in the diagnostic lines and the records.
func (c Conclusion) Name() string {
	switch c {
	case Invalid:
		return "invalid"
	case Incomplete:
		return "incomplete"
	case Inconsistent:
		return "inconsistent-with-candidates"
	case Consistent:
		return "consistent-with-candidates"
	}
	return "none"
}

func (c Conclusion) String() string { return fmt.Sprintf("%d %s", int(c), c.Name()) }

// ExitNonzero is the exit status of every final conclusion other than 3
// and 4.
const ExitNonzero = 1

// ExitCode maps a final conclusion to the exit status (LE-10): 0 if and only
// if the conclusion is 3 or 4.
func ExitCode(c Conclusion) int {
	if c == Inconsistent || c == Consistent {
		return 0
	}
	return ExitNonzero
}

// Qualification is the deadline qualification of a published result
// (plan §6.4): decided once, at the publishing lock's logical decision
// point, and never changed.
type Qualification int

// The qualifications. Timely: the latch was unset and the reading was
// before D. Late: otherwise (and the latch is then set).
const (
	Timely Qualification = iota + 1
	Late
)

func (q Qualification) String() string {
	switch q {
	case Timely:
		return "timely"
	case Late:
		return "late"
	}
	return "unqualified"
}

// Outcome is one published result: the real error of the external call
// (nil on success), the clock reading of the publishing decision point and
// the qualification decided there.
type Outcome struct {
	Err  error
	At   time.Time
	Qual Qualification
}

// OK reports whether the result succeeded and is timely.
func (o Outcome) OK() bool { return o.Err == nil && o.Qual == Timely }

// ChainError is one real error of the record chain, with the operation it
// came from ("mkdir <dir>", "create raw.jsonl", "write manifest.txt", ...).
type ChainError struct {
	Site string
	Err  error
}

func (e ChainError) Error() string { return e.Site + ": " + e.Err.Error() }

// ManifestName is the manifest's file name in the record directory.
const ManifestName = "manifest.txt"

// FormatVersion is the version in every record file's header line.
const FormatVersion = 1

// HeaderLine is the first line of a record file:
//
//	{"c9evidence":"<name>","version":1}
//
// followed by LF. Names are restricted by ValidFileName, so no escaping is
// needed for the line to be JSON.
func HeaderLine(name string) []byte {
	return []byte(`{"c9evidence":"` + name + `","version":` + strconv.Itoa(FormatVersion) + "}\n")
}

// ManifestLine is one line of the manifest: "<name> <bytes> <sha256>" and
// LF, bytes in decimal, the SHA-256 in lowercase hex. The manifest has one
// such line per record file, sorted by name, and nothing else.
func ManifestLine(name string, bytes int64, sha string) string {
	return name + " " + strconv.FormatInt(bytes, 10) + " " + sha + "\n"
}

// LinePrefix starts every diagnostic line.
const LinePrefix = "c9evidence: "

// The diagnostic lines, in the order the diagnostic task writes them. Each
// is one Write of exactly one line ending in LF. Free text is Go-quoted
// (strconv.Quote), so every line is a single line.
//
//	c9evidence: provisional <n> <name>
//	c9evidence: failure-count <k>
//	c9evidence: failure <quoted text>                (k lines)
//	c9evidence: pending-count <k>
//	c9evidence: pending <quoted text>                (k lines)
//
// then, when the writer's terminal state is success:
//
//	c9evidence: record <name> <bytes> <sha256>        (one per record file, sorted by name)
//	c9evidence: manifest manifest.txt <bytes> <sha256>
//	c9evidence: end manifest-sha256 <sha256> provisional <n> <name>
//
// the last one being the terminator line; or, when it is failure:
//
//	c9evidence: record-chain-error-count <k>
//	c9evidence: record-chain-error <quoted text>      (k lines; no terminator)

// ProvisionalLine is the provisional conclusion line.
func ProvisionalLine(c Conclusion) string {
	return LinePrefix + "provisional " + strconv.Itoa(int(c)) + " " + c.Name() + "\n"
}

// CountLine is "<kind>-count <n>"; kind is "failure", "pending" or
// "record-chain-error".
func CountLine(kind string, n int) string {
	return LinePrefix + kind + "-count " + strconv.Itoa(n) + "\n"
}

// TextLine is "<kind> <quoted text>".
func TextLine(kind, text string) string {
	return LinePrefix + kind + " " + strconv.Quote(text) + "\n"
}

// RecordLine is the hash line of one record file.
func RecordLine(name string, bytes int64, sha string) string {
	return LinePrefix + "record " + name + " " + strconv.FormatInt(bytes, 10) + " " + sha + "\n"
}

// ManifestHashLine is the hash line of the manifest.
func ManifestHashLine(bytes int64, sha string) string {
	return LinePrefix + "manifest " + ManifestName + " " + strconv.FormatInt(bytes, 10) + " " + sha + "\n"
}

// TerminatorLine is the last line of a complete diagnostic output: the
// manifest's SHA-256 and the provisional conclusion.
func TerminatorLine(manifestSHA string, c Conclusion) string {
	return LinePrefix + "end manifest-sha256 " + manifestSHA + " provisional " + strconv.Itoa(int(c)) + " " + c.Name() + "\n"
}

// errShortWrite is a write that returned fewer bytes than asked without an
// error, which breaks the io.Writer contract.
var errShortWrite = errors.New("short write")

// normWrite makes a write result usable: n is clamped to [0, want]; a short
// write without an error, or a count outside the range, becomes an error.
func normWrite(n int, err error, want int) (int, error) {
	if n < 0 || n > want {
		bad := fmt.Errorf("write reported %d of %d bytes", n, want)
		if n < 0 {
			n = 0
		} else {
			n = want
		}
		if err == nil {
			return n, bad
		}
		return n, errors.Join(err, bad)
	}
	if n < want && err == nil {
		return n, errShortWrite
	}
	return n, err
}
