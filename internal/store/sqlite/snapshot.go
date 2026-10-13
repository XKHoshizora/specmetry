package sqlite

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"
	"time"

	"modernc.org/sqlite/vfs"
)

// This file reads the database without writing anything — not the database,
// not its WAL or -shm, and no directory entry (codex-review-004 B2, B3).
//
// Opening the files through SQLite cannot give that guarantee. Measured with
// the driver in use (modernc.org/sqlite 1.60.1, SQLite 3.53.4) on Windows and
// Linux: a read-write connection with query_only=1 recovers a crashed WAL,
// checkpoints it into the database and deletes it on close; mode=ro creates
// -wal and -shm files and leaves them behind or rewrites the -shm; adding
// readonly_shm=1 avoids changes only when both -wal and -shm already exist.
// immutable=1 would tell SQLite that a database that may be changing cannot
// change, so it is not used either.
//
// Instead the files are opened read-only by this package. Committed WAL
// frames are applied in memory, following SQLite's WAL recovery rules
// (https://sqlite.org/fileformat2.html#walformat), and SQLite reads the
// resulting image through a read-only VFS as a rollback-journal database.
// Nothing is locked, so afterwards every page that was used is read again and
// the WAL is parsed again; if anything changed, the read is retried.

// uninspectableError explains why the database cannot be read without first
// being recovered, which would modify it.
type uninspectableError struct{ reason string }

func (e *uninspectableError) Error() string { return e.reason }

var errSnapshotChanged = errors.New("database changed while it was read")

// snapshotMeta is what a snapshot read reports.
type snapshotMeta struct {
	exists  bool
	version int
	minBin  string
}

const (
	walHeaderSize      = 32
	walFrameHeaderSize = 24
	walMagic           = 0x377f0682 // the low bit selects big-endian checksums
	walFormatVersion   = 3007000
	snapshotAttempts   = 20
	snapshotName       = "/specmetry-snapshot.db"
)

// journalMagic starts a rollback journal that may still need to be played
// back (a "hot" journal).
var journalMagic = []byte{0xd9, 0xd5, 0x05, 0xf9, 0x20, 0xa1, 0x63, 0xd7}

// readSnapshotMeta reads the schema version and minimum binary version from
// the database at dbPath without modifying any file. A missing database
// reports exists=false. It returns an *uninspectableError if that is not
// possible without recovery.
func readSnapshotMeta(ctx context.Context, dbPath string) (snapshotMeta, error) {
	var m snapshotMeta
	exists, err := readSnapshot(ctx, dbPath, func(sdb *sql.DB) error {
		var err error
		m.version, m.minBin, err = readMeta(ctx, sdb)
		return err
	})
	m.exists = exists
	return m, err
}

// readSnapshot runs query against a consistent read-only snapshot of the
// database, retrying while other processes change it. It reports whether the
// database exists; query is not called for a missing or empty database.
func readSnapshot(ctx context.Context, dbPath string, query func(*sql.DB) error) (bool, error) {
	for attempt := 1; ; attempt++ {
		exists, err := readSnapshotOnce(dbPath, query)
		if !errors.Is(err, errSnapshotChanged) {
			return exists, err
		}
		if attempt == snapshotAttempts {
			return true, &uninspectableError{"the database kept changing while it was being read; try again"}
		}
		select {
		case <-ctx.Done():
			return true, ctx.Err()
		case <-time.After(time.Duration(attempt) * 5 * time.Millisecond):
		}
	}
}

// snapshotHook, if set, runs after the query and before validation (tests).
var snapshotHook func()

func readSnapshotOnce(dbPath string, query func(*sql.DB) error) (bool, error) {
	dbFile, err := os.Open(dbPath)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer func() { _ = dbFile.Close() }()
	dbInfo, err := dbFile.Stat()
	if err != nil {
		return true, err
	}
	if !dbInfo.Mode().IsRegular() {
		return true, fmt.Errorf("%s is not a regular file", dbPath)
	}
	if hot, err := hotJournal(dbPath + "-journal"); err != nil {
		return true, err
	} else if hot {
		return true, &uninspectableError{"an interrupted write left a rollback journal (" + dbPath + "-journal) that must be played back first"}
	}

	dbPageSize, err := headerPageSize(dbFile, dbInfo.Size())
	if err != nil {
		return true, err
	}
	walFile, err := os.Open(dbPath + "-wal")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return true, err
	}
	var wal walIndex
	if walFile != nil {
		defer func() { _ = walFile.Close() }()
		if wal, err = parseWAL(walFile); err != nil {
			return true, err
		}
	}

	pageSize := dbPageSize
	if wal.nPages > 0 {
		if dbPageSize != 0 && wal.pageSize != dbPageSize {
			return true, &uninspectableError{"the WAL page size differs from the database page size"}
		}
		pageSize = wal.pageSize
	}
	if pageSize == 0 {
		// An empty file without committed WAL frames: a new database.
		return true, nil
	}
	size := dbInfo.Size()
	if wal.nPages > 0 {
		size = int64(wal.nPages) * int64(pageSize)
	}
	img := &snapshotImage{db: dbFile, wal: walFile, frames: wal.frames, pageSize: int64(pageSize), size: size,
		pages: map[uint32][]byte{}, sources: map[uint32][32]byte{}}

	err = queryImage(img, query)
	if snapshotHook != nil {
		snapshotHook()
	}
	if changed, verr := img.changed(dbPath, dbInfo, wal); verr != nil {
		return true, verr
	} else if changed {
		return true, errSnapshotChanged
	}
	if err != nil {
		return true, fmt.Errorf("cannot read the database: %w", err)
	}
	return true, nil
}

func queryImage(img *snapshotImage, query func(*sql.DB) error) error {
	name, fsys, err := vfs.New(snapshotFS{img})
	if err != nil {
		return err
	}
	defer func() { _ = fsys.Close() }()
	sdb, err := sql.Open("sqlite", "file:"+snapshotName+"?vfs="+name+"&mode=ro")
	if err != nil {
		return err
	}
	defer func() { _ = sdb.Close() }()
	sdb.SetMaxOpenConns(1)
	return query(sdb)
}

// hotJournal reports whether path is a rollback journal that SQLite would
// play back before reading.
func hotJournal(path string) (bool, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()
	head := make([]byte, len(journalMagic))
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return false, err
	}
	return n == len(head) && bytes.Equal(head, journalMagic), nil
}

// headerPageSize returns the page size recorded in the database header, or 0
// for an empty file.
func headerPageSize(f *os.File, size int64) (uint32, error) {
	if size == 0 {
		return 0, nil
	}
	hdr := make([]byte, 100)
	if _, err := f.ReadAt(hdr, 0); err != nil {
		return 0, fmt.Errorf("cannot read the database header: %w", err)
	}
	if string(hdr[:16]) != "SQLite format 3\x00" {
		return 0, errors.New("the database file is not a SQLite database")
	}
	ps := uint32(binary.BigEndian.Uint16(hdr[16:18]))
	if ps == 1 {
		ps = 65536
	}
	if ps < 512 || ps&(ps-1) != 0 {
		return 0, fmt.Errorf("invalid database page size %d", ps)
	}
	return ps, nil
}

// walIndex is the committed content of a WAL file.
type walIndex struct {
	pageSize uint32
	frames   map[uint32]int64 // page number -> offset of the newest committed copy
	nPages   uint32           // database size in pages after the last commit; 0 if none
	// signature identifies the committed content: header and the running
	// checksum after the last commit frame.
	signature [40]byte
}

// parseWAL applies SQLite's recovery rules: the header must be valid, frames
// are read in order while their salt matches the header and the cumulative
// checksum is correct, and only frames up to the last commit frame count.
func parseWAL(f *os.File) (walIndex, error) {
	var w walIndex
	hdr := make([]byte, walHeaderSize)
	if _, err := f.ReadAt(hdr, 0); err != nil {
		if errors.Is(err, io.EOF) {
			return w, nil // empty or truncated header: SQLite ignores the file
		}
		return w, err
	}
	magic := binary.BigEndian.Uint32(hdr[0:])
	pageSize := binary.BigEndian.Uint32(hdr[8:])
	if magic&^1 != walMagic || pageSize < 512 || pageSize > 65536 || pageSize&(pageSize-1) != 0 {
		return w, nil
	}
	bigEndian := magic&1 == 1
	s0, s1 := walChecksum(bigEndian, hdr[:24], 0, 0)
	if s0 != binary.BigEndian.Uint32(hdr[24:]) || s1 != binary.BigEndian.Uint32(hdr[28:]) {
		return w, nil
	}
	if v := binary.BigEndian.Uint32(hdr[4:]); v != walFormatVersion {
		return w, &uninspectableError{fmt.Sprintf("the WAL uses format version %d, which this binary cannot read", v)}
	}
	salt := hdr[16:24]
	committed := map[uint32]int64{}
	pending := map[uint32]int64{}
	frame := make([]byte, walFrameHeaderSize+int(pageSize))
	for off := int64(walHeaderSize); ; off += int64(len(frame)) {
		if _, err := f.ReadAt(frame, off); err != nil {
			if errors.Is(err, io.EOF) {
				break // a partly written last frame is not valid
			}
			return w, err
		}
		pgno := binary.BigEndian.Uint32(frame[0:])
		commit := binary.BigEndian.Uint32(frame[4:])
		if pgno == 0 || !bytes.Equal(frame[8:16], salt) {
			break
		}
		s0, s1 = walChecksum(bigEndian, frame[:8], s0, s1)
		s0, s1 = walChecksum(bigEndian, frame[walFrameHeaderSize:], s0, s1)
		if s0 != binary.BigEndian.Uint32(frame[16:]) || s1 != binary.BigEndian.Uint32(frame[20:]) {
			break
		}
		pending[pgno] = off + walFrameHeaderSize
		if commit != 0 {
			for p, o := range pending {
				committed[p] = o
			}
			clear(pending)
			w.nPages = commit
			copy(w.signature[:32], hdr)
			binary.BigEndian.PutUint32(w.signature[32:], s0)
			binary.BigEndian.PutUint32(w.signature[36:], s1)
		}
	}
	if w.nPages > 0 {
		w.pageSize = pageSize
		w.frames = committed
	}
	return w, nil
}

// walChecksum is SQLite's WAL checksum over 32-bit words.
func walChecksum(bigEndian bool, b []byte, s0, s1 uint32) (uint32, uint32) {
	order := binary.ByteOrder(binary.LittleEndian)
	if bigEndian {
		order = binary.BigEndian
	}
	for i := 0; i+8 <= len(b); i += 8 {
		s0 += order.Uint32(b[i:]) + s1
		s1 += order.Uint32(b[i+4:]) + s0
	}
	return s0, s1
}

// snapshotImage is the database as of the last committed WAL frame, read
// lazily one page at a time.
type snapshotImage struct {
	db, wal  *os.File
	frames   map[uint32]int64
	pageSize int64
	size     int64

	mu      sync.Mutex
	pages   map[uint32][]byte   // pages served, page 1 already patched
	sources map[uint32][32]byte // hash of each page's bytes as read from disk
}

func (img *snapshotImage) readSource(pgno uint32) ([]byte, error) {
	buf := make([]byte, img.pageSize)
	f, off := img.db, int64(pgno-1)*img.pageSize
	if o, ok := img.frames[pgno]; ok {
		f, off = img.wal, o
	}
	n, err := f.ReadAt(buf, off)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	clear(buf[n:]) // beyond the end of the database file
	return buf, nil
}

func (img *snapshotImage) page(pgno uint32) ([]byte, error) {
	img.mu.Lock()
	defer img.mu.Unlock()
	if p, ok := img.pages[pgno]; ok {
		return p, nil
	}
	src, err := img.readSource(pgno)
	if err != nil {
		return nil, err
	}
	img.sources[pgno] = sha256.Sum256(src)
	p := src
	if pgno == 1 && len(p) >= 20 {
		// Present a WAL database as a rollback-journal database (header
		// bytes 18 and 19), so SQLite reads only this image and never looks
		// for a WAL or shared memory.
		p = bytes.Clone(src)
		p[18], p[19] = 1, 1
	}
	img.pages[pgno] = p
	return p, nil
}

func (img *snapshotImage) readAt(b []byte, off int64) (int, error) {
	if off >= img.size {
		return 0, io.EOF
	}
	n := 0
	for n < len(b) && off < img.size {
		// off < size, and size is at most a uint32 page count times the
		// page size, so the page number fits.
		pgno := uint32(off/img.pageSize) + 1 //nolint:gosec // see above
		p, err := img.page(pgno)
		if err != nil {
			return n, err
		}
		c := copy(b[n:], p[off%img.pageSize:])
		if rest := img.size - off; int64(c) > rest {
			c = int(rest)
		}
		n += c
		off += int64(c)
	}
	if n < len(b) {
		return n, io.EOF
	}
	return n, nil
}

// changed reports whether any page that was read, the WAL, or the database
// size changed since the snapshot was taken.
func (img *snapshotImage) changed(dbPath string, dbInfo fs.FileInfo, wal walIndex) (bool, error) {
	img.mu.Lock()
	defer img.mu.Unlock()
	for pgno, sum := range img.sources {
		src, err := img.readSource(pgno)
		if err != nil {
			return false, err
		}
		if sha256.Sum256(src) != sum {
			return true, nil
		}
	}
	now, err := os.Stat(dbPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return true, nil
		}
		return false, err
	}
	if now.Size() != dbInfo.Size() {
		return true, nil
	}
	var again walIndex
	if f, err := os.Open(dbPath + "-wal"); err == nil {
		again, err = parseWAL(f)
		_ = f.Close()
		if err != nil {
			return false, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	return again.signature != wal.signature, nil
}

// snapshotFS serves the image under snapshotName and nothing else, so SQLite
// finds no journal and no WAL.
type snapshotFS struct{ img *snapshotImage }

func (s snapshotFS) Open(name string) (fs.File, error) {
	// The VFS prefixes the URI path with "/", so compare without slashes.
	if strings.TrimLeft(name, "/") == strings.TrimLeft(snapshotName, "/") {
		return &snapshotFile{img: s.img}, nil
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

type snapshotFile struct {
	img *snapshotImage
	off int64
}

func (f *snapshotFile) Read(b []byte) (int, error) {
	n, err := f.img.readAt(b, f.off)
	f.off += int64(n)
	return n, err
}

func (f *snapshotFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.off = offset
	case io.SeekCurrent:
		f.off += offset
	case io.SeekEnd:
		f.off = f.img.size + offset
	}
	return f.off, nil
}

func (f *snapshotFile) Stat() (fs.FileInfo, error) { return snapshotInfo{size: f.img.size}, nil }

func (f *snapshotFile) Close() error { return nil }

type snapshotInfo struct{ size int64 }

func (i snapshotInfo) Name() string       { return strings.TrimLeft(snapshotName, "/") }
func (i snapshotInfo) Size() int64        { return i.size }
func (i snapshotInfo) Mode() fs.FileMode  { return 0o400 }
func (i snapshotInfo) ModTime() time.Time { return time.Time{} }
func (i snapshotInfo) IsDir() bool        { return false }
func (i snapshotInfo) Sys() any           { return nil }
