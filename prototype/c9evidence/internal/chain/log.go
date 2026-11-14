package chain

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"unicode/utf8"
)

// Item is one record item: a global sequence number (1, 2, ... across all
// files, in append order), the record file it belongs to and the line
// (UTF-8, without the trailing LF, which the writer adds).
type Item struct {
	Seq  int
	File string
	Line []byte
}

// Log is the in-memory, append-only log of record items (plan §6.4
// 记录项). Producers only append; the writer is the only reader that turns
// items into file bytes. The parent seals it at the snapshot point (inside
// Closeout); after the seal the writer writes exactly the items up to the
// seal and stops reading, and later items stay in memory only (they are not
// guaranteed to reach a file, plan §6.4 晚到的结果).
type Log struct {
	mu      sync.Mutex
	files   []string
	known   map[string]bool
	items   []Item
	sealed  bool
	sealSeq int
	changed chan struct{}
	defects []string
}

// ValidFileName checks a record file name: 1 to 64 bytes of [a-z0-9._-],
// starting with a letter or digit, and not the manifest's name.
func ValidFileName(name string) error {
	if name == "" || len(name) > 64 {
		return fmt.Errorf("record file name %q: length must be 1 to 64", name)
	}
	if name == ManifestName {
		return fmt.Errorf("record file name %q is the manifest's name", name)
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		alnum := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
		if i == 0 && !alnum {
			return fmt.Errorf("record file name %q must start with a letter or digit", name)
		}
		if !alnum && c != '.' && c != '_' && c != '-' {
			return fmt.Errorf("record file name %q: byte %q is not allowed", name, c)
		}
	}
	return nil
}

// NewLog makes a log for this fixed, non-empty set of record files.
func NewLog(files ...string) (*Log, error) {
	if len(files) == 0 {
		return nil, errors.New("chain: a log needs at least one record file")
	}
	l := &Log{known: map[string]bool{}, changed: make(chan struct{})}
	for _, f := range files {
		if err := ValidFileName(f); err != nil {
			return nil, fmt.Errorf("chain: %w", err)
		}
		if l.known[f] {
			return nil, fmt.Errorf("chain: record file %q listed twice", f)
		}
		l.known[f] = true
		l.files = append(l.files, f)
	}
	return l, nil
}

// Files returns the record files in construction order.
func (l *Log) Files() []string { return append([]string(nil), l.files...) }

// checkItem reports why an item cannot be appended (file and line format).
func (l *Log) checkItem(file string, line []byte) error {
	switch {
	case !l.known[file]:
		return fmt.Errorf("item for unknown record file %q", file)
	case bytes.IndexByte(line, '\n') >= 0:
		return fmt.Errorf("item for %s contains LF", file)
	case !utf8.Valid(line):
		return fmt.Errorf("item for %s is not valid UTF-8", file)
	}
	return nil
}

// Append adds one item and returns its sequence number. It may be called
// from any goroutine and never does a file operation. An item for a file
// outside the set, or a line with LF or invalid UTF-8, is a tool defect: it
// is kept in Defects, nothing is appended, and the result is 0. Items
// appended after the seal are kept in memory with their sequence numbers
// (greater than the seal) and are never written.
func (l *Log) Append(file string, line []byte) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.checkItem(file, line); err != nil {
		l.defects = append(l.defects, err.Error())
		return 0
	}
	seq := len(l.items) + 1
	l.items = append(l.items, Item{Seq: seq, File: file, Line: append([]byte(nil), line...)})
	l.notifyLocked()
	return seq
}

func (l *Log) notifyLocked() {
	close(l.changed)
	l.changed = make(chan struct{})
}

// Items returns a copy of every item, including those after the seal.
func (l *Log) Items() []Item {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Item, len(l.items))
	for i, it := range l.items {
		it.Line = append([]byte(nil), it.Line...)
		out[i] = it
	}
	return out
}

// Sealed reports whether the log is sealed and the last sequence number
// inside the seal.
func (l *Log) Sealed() (sealed bool, sealSeq int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.sealed, l.sealSeq
}

// Defects returns the rejected appends.
func (l *Log) Defects() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.defects...)
}

// sealWith appends the closeout items and fixes the end of the record files
// right after them, in one critical section: no producer item can come
// between them, so the closeout items are the last items inside the seal.
// Items that fail checkItem are recorded as defects and skipped (Closeout
// checks them first). It returns the seal and, for each file, the last
// sequence number inside the seal (0 when the file has none). It is called
// once, by Closeout, with the outer mutex held; ok is false if the log was
// already sealed, and then nothing is appended.
func (l *Log) sealWith(recs []Record) (sealSeq int, lastPerFile map[string]int, ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sealed {
		return l.sealSeq, nil, false
	}
	for _, r := range recs {
		if err := l.checkItem(r.File, r.Line); err != nil {
			l.defects = append(l.defects, err.Error())
			continue
		}
		l.items = append(l.items, Item{Seq: len(l.items) + 1, File: r.File, Line: append([]byte(nil), r.Line...)})
	}
	l.sealed = true
	l.sealSeq = len(l.items)
	lastPerFile = map[string]int{}
	for _, f := range l.files {
		lastPerFile[f] = 0
	}
	for _, it := range l.items {
		lastPerFile[it.File] = it.Seq
	}
	l.notifyLocked()
	return l.sealSeq, lastPerFile, true
}

// take returns copies of the items from sequence number from on: every
// current item while the log is unsealed, only those up to the seal once it
// is sealed. It also returns the channel that is closed at the next append
// or seal, read in the same critical section so no change is missed.
func (l *Log) take(from int) (items []Item, sealed bool, sealSeq int, changed <-chan struct{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	end := len(l.items)
	if l.sealed {
		end = l.sealSeq
	}
	if from < 1 {
		from = 1
	}
	if from <= end {
		items = append([]Item(nil), l.items[from-1:end]...)
	}
	return items, l.sealed, l.sealSeq, l.changed
}
