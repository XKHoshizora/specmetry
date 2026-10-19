package gitx

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
)

// Private directories (WP2.3 spec §4.3, "the temporary directory's
// lifecycle"; review 058 B1): Clean's private git directories, and the
// directory that holds the missing grafts and shallow files of the scan's
// reads. They are made in a Scratch: the tmp directory of the data
// directory, after datadir.Ensure has shown that no other user can replace
// anything on the data directory's path, and that the data directory and
// everything in it are private to the current user (TD-22), so that the
// whole name space a private directory lives in is the current user's.
// What each step adds, and only that:
//
//   - the random name keeps two reads of one user apart;
//   - the lock file, made before the directory and held while it is in
//     use, tells a directory in use from one a crashed read left; only a
//     directory whose lock can be taken is removed as left over;
//   - the identity recorded when the directory is made, checked before git
//     uses it and before it is removed, finds a directory that was moved or
//     replaced since: then nothing at that path is removed, and the read
//     fails;
//   - files in it are made new (O_EXCL).
//
// Nothing in them is a project fact; they never reach a log, an error or a
// backup. Leftovers of earlier builds in the system's temporary directory
// are not looked for.

const (
	privateNameLen = 32 // hex digits
	lockSuffix     = ".lock"
)

// privateName matches what newPrivateDir makes: "read-<hex>" and its lock.
var privateName = regexp.MustCompile(`^read-[0-9a-f]{32}(\.lock)?$`)

// removeAll removes a directory tree; tests replace it to make removal
// fail.
var removeAll = os.RemoveAll

// testHookPrivateDir runs once a private directory is made, for tests
// that move or replace it.
var testHookPrivateDir func(path string)

// Scratch is where a command's reads make their private directories: the
// data directory's tmp directory, checked when it was made.
type Scratch struct {
	root string
	id   os.FileInfo
}

// NewScratch readies the data directory d's tmp directory for private
// directories: datadir.Ensure first (the path to d not replaceable by
// other users, d and everything in it private; d made if missing), then
// tmp, its identity recorded, and the private directories crashed reads
// left in it removed. It opens no database and changes nothing else in d.
func NewScratch(d datadir.Dir) (*Scratch, error) {
	if err := d.Ensure(); err != nil {
		return nil, err
	}
	root, err := d.EnsureSub(datadir.TmpDir)
	if err != nil {
		return nil, err
	}
	id, err := identity(root)
	if err != nil {
		return nil, scratchProblem("cannot read the data directory's tmp directory")
	}
	s := &Scratch{root: root, id: id}
	s.removeLeftovers()
	return s, nil
}

// identity returns the identity of the directory at path, read from a
// handle opened on it, so that it is that directory's and not whatever is
// at the path later.
func identity(path string) (os.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.New("not a directory")
	}
	return fi, nil
}

// same reports whether path is still the directory id was read from: a
// directory, not a link, the same file.
func same(path string, id os.FileInfo) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.IsDir() && fi.Mode()&fs.ModeSymlink == 0 && os.SameFile(fi, id)
}

func scratchProblem(what string) error {
	return apperr.New(apperr.Environment, what+"; Specmetry reads git only through its own private directories, so nothing was read")
}

// removeLeftovers removes the private directories of reads that stopped:
// a lock file Specmetry made, whose lock no process holds, and the
// directory of the same name if it is a directory. A lock held, a name
// Specmetry does not make, or a directory without its lock file is left
// alone; failures are not errors (the lock file stays, to be tried again).
func (s *Scratch) removeLeftovers() {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, lockSuffix) || !privateName.MatchString(name) || !e.Type().IsRegular() {
			continue
		}
		lockPath := filepath.Join(s.root, name)
		lock, err := os.OpenFile(lockPath, os.O_RDWR, 0)
		if err != nil {
			continue
		}
		if locked, err := tryLock(lock); err != nil || !locked {
			_ = lock.Close()
			continue
		}
		dir := strings.TrimSuffix(lockPath, lockSuffix)
		fi, err := os.Lstat(dir)
		gone := errors.Is(err, fs.ErrNotExist)
		if err == nil && fi.IsDir() && fi.Mode()&fs.ModeSymlink == 0 {
			gone = removeAll(dir) == nil
		}
		_ = unlock(lock)
		_ = lock.Close()
		if gone {
			_ = os.Remove(lockPath)
		}
	}
}

// privateDir is a private directory in use.
type privateDir struct {
	path, lockPath string
	lock           *os.File
	id             os.FileInfo
}

// newPrivateDir makes and locks a private directory in s.
func newPrivateDir(s *Scratch) (*privateDir, error) {
	if s == nil {
		return nil, apperr.New(apperr.Internal, "a git read needs a Scratch")
	}
	if !same(s.root, s.id) {
		return nil, scratchProblem("the data directory's tmp directory was moved or replaced since it was checked")
	}
	var b [privateNameLen / 2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, scratchProblem("cannot name a private directory")
	}
	path := filepath.Join(s.root, "read-"+hex.EncodeToString(b[:]))
	d := &privateDir{path: path, lockPath: path + lockSuffix}
	lock, err := os.OpenFile(d.lockPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, scratchProblem("cannot make a private directory's lock (" + osProblem(err) + ")")
	}
	if locked, err := tryLock(lock); err != nil || !locked {
		_ = lock.Close()
		_ = os.Remove(d.lockPath)
		return nil, scratchProblem("cannot lock a private directory")
	}
	d.lock = lock
	if err := os.Mkdir(path, 0o700); err != nil {
		d.release(true)
		return nil, scratchProblem("cannot make a private directory (" + osProblem(err) + ")")
	}
	if d.id, err = identity(path); err != nil {
		d.release(true) // what is at the path is not known to be ours: no cleanup may remove it
		return nil, scratchProblem("cannot read a private directory it made")
	}
	if testHookPrivateDir != nil {
		testHookPrivateDir(path)
	}
	return d, nil
}

// join names a file in the directory.
func (d *privateDir) join(elem ...string) string {
	return filepath.Join(append([]string{d.path}, elem...)...)
}

// verify fails unless the directory is still the one newPrivateDir made.
func (d *privateDir) verify() error {
	if !same(d.path, d.id) {
		return scratchProblem("a private directory was moved or replaced while in use")
	}
	return nil
}

// remove removes the directory, if it is still the one made, then its
// lock. A directory moved or replaced is not removed: what is at its path
// now is not known to be ours, and its lock is removed too, so that no
// later cleanup removes it either; the read fails. On Windows a file a git
// process has just closed can stay in use for a moment, so a removal that
// fails is tried again briefly; one that still fails keeps the lock file,
// unlocked, so that a later Scratch removes the directory, and fails the
// read (the spec: a read whose directory cannot be removed is not
// reported clean).
func (d *privateDir) remove() error {
	if err := d.verify(); err != nil {
		d.release(true)
		return err
	}
	var err error
	for range 20 {
		if err = removeAll(d.path); err == nil {
			d.release(true)
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	d.release(false)
	return apperr.New(apperr.Environment, "cannot remove a private directory Specmetry read git with ("+osProblem(err)+")")
}

// release unlocks and closes the lock, and removes its file when asked.
func (d *privateDir) release(removeLock bool) {
	_ = unlock(d.lock)
	_ = d.lock.Close()
	if removeLock {
		_ = os.Remove(d.lockPath)
	}
}
