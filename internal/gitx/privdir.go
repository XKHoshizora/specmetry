package gitx

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
)

// Private directories (WP2.3 spec §14.3; reviews 058 B1 and 059 B1-R):
// Clean's private git directories, and the directory that holds the
// missing grafts and shallow files of the scan's reads. They are made in a
// Scratch: the tmp directory of the data directory, after datadir.Ensure
// has shown that no other user can replace anything on the data
// directory's path, and that the data directory and everything in it are
// private to the current user (TD-22). Other users cannot reach that name
// space at all; what follows keeps a read from acting on anything it
// cannot show is its own when the name space changes otherwise (a process
// of the same user, or a trusted owner).
//
//   - Identity. The Scratch records the identity of every directory from
//     the volume's root down to tmp, its path resolved, with no link on it,
//     and keeps tmp open; a private directory is opened as it is made and
//     its identity read from that handle. Before a private directory is
//     used for anything, before git is given its path and before each
//     removal attempt, every directory on its path is checked to be, at
//     that path and not through a link or junction, the one recorded. A
//     change fails the read (exit 6); from then on the read removes
//     nothing, releases only its own handles, and keeps what is at the
//     path, and wherever its own directory went.
//   - Handles. Everything Specmetry writes in a private directory, and
//     everything it removes from it, goes through the directory's handle,
//     so that it acts on that directory whatever the path leads to by
//     then. Its own entry in tmp is removed only if it is still the
//     directory made: on Windows that identity is read from the handle the
//     deletion is set on; elsewhere it is read, then rmdir run relative to
//     tmp, which removes only an empty directory. On Windows a directory
//     cannot be renamed while anything in it is open, nor while a handle
//     on it that does not share deletion is (the Scratch's on tmp): so
//     tmp and the directories above it cannot be moved or replaced while
//     the Scratch is open, nor a private directory while its owner file
//     is; once that is removed, only the identity on the deleting handle
//     guards the directory's own entry. Elsewhere open handles keep
//     nothing in place, and the checks above are what find a change.
//   - Owner. A private directory holds its own lock and record, the owner
//     file: locked while the directory is in use, and naming the identity
//     of the directory and of itself. A new Scratch removes a directory in
//     tmp as a crashed read's only if its name is one Specmetry makes, it
//     is a directory, its owner file is a file whose lock no process
//     holds, and the record names that directory and that file: a copy, a
//     directory put in its place, or one Specmetry did not make is kept.
//     The owner file is removed last, still locked, after the directory's
//     other content.
//
// What the checks leave open, and only to a process of the same user (or
// a trusted owner): a change between a check and the step it guards that
// is undone before the next check (git may then have read another
// directory than its own; the next check finds a change that stays); on
// systems other than Windows, an empty directory put at a private
// directory's name between the check and the rmdir, and a file put at the
// owner file's name inside the private directory between its check and
// its removal; and on every system, an empty directory put at a new
// private directory's name between its mkdir and its opening, which is
// then taken for the one made.
//
// Nothing in them is a project fact; they never reach a log, an error or a
// backup. Leftovers of earlier builds in the system's temporary directory
// are not looked for.

const (
	privateNameLen = 32 // hex digits
	// ownerName is the owner file in each private directory; git never
	// makes a file of that name.
	ownerName      = "specmetry-owner"
	recordHeader   = "specmetry private directory 1\n"
	removeAttempts = 20
)

// privateName matches what newPrivateDir makes.
var privateName = regexp.MustCompile(`^read-[0-9a-f]{32}$`)

// errReplaced is removeEntry's: the entry is not the file it was to
// remove.
var errReplaced = errors.New("replaced")

// Test hooks.
var (
	// testHookMkdir runs once a private directory is made, before it is
	// opened.
	testHookMkdir func(path string)
	// testHookPrivateDir runs once a private directory is made, opened and
	// locked.
	testHookPrivateDir func(path string)
	// testHookRemove runs before a removal attempt's step: "contents"
	// (what is in the directory, then its owner file) or "entry" (the
	// directory itself). An error it returns fails the attempt.
	testHookRemove func(step, path string) error
)

// Scratch is where a command's reads make their private directories: the
// data directory's tmp directory, checked when it was made, and held open
// until Close.
type Scratch struct {
	path  string   // tmp, resolved
	chain []dirKey // every directory from the volume's root down to tmp
	root  *os.Root // tmp; nil once closed
	dir   *os.File // tmp itself, open
}

// dirKey is a directory's path and identity.
type dirKey struct {
	path string
	key  fileKey
}

// NewScratch readies the data directory d's tmp directory for private
// directories: datadir.Ensure first (the path to d not replaceable by
// other users, d and everything in it private; d made if missing), then
// tmp, its path's identities recorded and tmp opened, and the private
// directories crashed reads left in it removed. It opens no database and
// changes nothing else in d. The caller closes it once its reads have
// returned.
func NewScratch(d datadir.Dir) (*Scratch, error) {
	if err := d.Ensure(); err != nil {
		return nil, err
	}
	tmp, err := d.EnsureSub(datadir.TmpDir)
	if err != nil {
		return nil, err
	}
	s, err := openScratch(tmp)
	if err != nil {
		return nil, err
	}
	s.removeLeftovers()
	return s, nil
}

func openScratch(tmp string) (*Scratch, error) {
	path, err := filepath.Abs(tmp)
	if err == nil {
		path, err = filepath.EvalSymlinks(path)
	}
	if err != nil {
		return nil, scratchProblem("cannot resolve the path of the data directory's tmp directory")
	}
	chain, err := dirChain(path)
	if err != nil {
		return nil, scratchProblem("cannot read the directories on the path to the data directory's tmp directory")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, scratchProblem("cannot open the data directory's tmp directory")
	}
	s := &Scratch{path: path, chain: chain, root: root}
	if s.dir, err = root.Open("."); err == nil {
		var key fileKey
		var dir bool
		if key, dir, err = keyOf(s.dir); err == nil && (!dir || key != chain[len(chain)-1].key) {
			err = errReplaced
		}
	}
	if err != nil {
		_ = s.Close()
		return nil, scratchProblem("the data directory's tmp directory was moved or replaced while it was opened")
	}
	return s, nil
}

// dirChain returns the identity of every directory on path, from the
// volume's root down; each must be a directory, not a link.
func dirChain(path string) ([]dirKey, error) {
	var chain []dirKey
	for p := path; ; p = filepath.Dir(p) {
		key, dir, err := lstatKey(p)
		if err != nil {
			return nil, err
		}
		if !dir {
			return nil, errors.New("not a directory")
		}
		chain = append(chain, dirKey{p, key})
		if filepath.Dir(p) == p {
			break
		}
	}
	slices.Reverse(chain)
	return chain, nil
}

// Close closes tmp. The Scratch makes no private directory after it.
func (s *Scratch) Close() error {
	if s == nil || s.root == nil {
		return nil
	}
	var err error
	if s.dir != nil {
		err = s.dir.Close()
	}
	err = errors.Join(err, s.root.Close())
	s.root, s.dir = nil, nil
	return err
}

// verify fails unless every directory on the path to tmp is, at its path
// and not through a link or junction, the one recorded.
func (s *Scratch) verify() error {
	for _, c := range s.chain {
		if key, dir, err := lstatKey(c.path); err != nil || !dir || key != c.key {
			return scratchProblem("the data directory's tmp directory, or a directory on the path to it, was moved or replaced since it was checked")
		}
	}
	return nil
}

func scratchProblem(what string) error {
	return apperr.New(apperr.Environment, what+"; Specmetry reads git only through its own private directories, so nothing was read")
}

// removeLeftovers removes the private directories of reads that stopped
// (see adopt). Failures are not errors: what cannot be removed is kept.
func (s *Scratch) removeLeftovers() {
	entries, err := s.dir.ReadDir(-1)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !privateName.MatchString(e.Name()) {
			continue
		}
		if d := s.adopt(e.Name()); d != nil {
			_ = d.remove()
		}
	}
}

// adopt opens the entry name of tmp as a crashed read's private directory,
// or returns nil: it must be a directory (not a link or junction), opened
// as the one at the name; its owner file a file, its lock taken; and the
// record in it must name that directory and that file.
func (s *Scratch) adopt(name string) *privateDir {
	path := filepath.Join(s.path, name)
	key, dir, err := lstatKey(path)
	if err != nil || !dir {
		return nil
	}
	d, err := s.open(name)
	if err != nil || d.key != key {
		d.close()
		return nil
	}
	lock, err := d.root.OpenFile(ownerName, os.O_RDWR, 0)
	if err != nil {
		d.close()
		return nil
	}
	if locked, err := tryLock(lock); err != nil || !locked {
		_ = lock.Close()
		d.close()
		return nil
	}
	d.lock = lock
	var isDir bool
	if d.lockKey, isDir, err = keyOf(lock); err != nil || isDir || !d.recordMatches() || d.verify() != nil {
		d.close()
		return nil
	}
	return d
}

// privateDir is a private directory in use.
type privateDir struct {
	s       *Scratch
	name    string
	path    string // s.path/name
	key     fileKey
	root    *os.Root // the directory; nil once closed
	dir     *os.File // the directory itself, open; nil once closed
	lock    *os.File // its owner file, locked; nil once closed
	lockKey fileKey
	// ownerGone is set once the owner file is removed.
	ownerGone bool
}

// open opens the entry name of tmp as a directory, its identity read from
// the handle.
func (s *Scratch) open(name string) (*privateDir, error) {
	d := &privateDir{s: s, name: name, path: filepath.Join(s.path, name)}
	var err error
	if d.root, err = s.root.OpenRoot(name); err != nil {
		return d, err
	}
	if d.dir, err = d.root.Open("."); err != nil {
		return d, err
	}
	var dir bool
	if d.key, dir, err = keyOf(d.dir); err == nil && !dir {
		err = errReplaced
	}
	return d, err
}

// newPrivateDir makes, opens and locks a private directory in s.
func newPrivateDir(s *Scratch) (*privateDir, error) {
	if s == nil || s.root == nil {
		return nil, apperr.New(apperr.Internal, "a git read needs an open Scratch")
	}
	if err := s.verify(); err != nil {
		return nil, err
	}
	var b [privateNameLen / 2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, scratchProblem("cannot name a private directory")
	}
	name := "read-" + hex.EncodeToString(b[:])
	if err := s.root.Mkdir(name, 0o700); err != nil {
		return nil, scratchProblem("cannot make a private directory (" + osProblem(err) + ")")
	}
	if testHookMkdir != nil {
		testHookMkdir(filepath.Join(s.path, name))
	}
	// Taken for the directory made only if it is, at its path, the
	// directory opened, and empty.
	d, err := s.open(name)
	if err == nil {
		var es []fs.DirEntry
		if es, err = fs.ReadDir(d.root.FS(), "."); err == nil && len(es) > 0 {
			err = errReplaced
		}
	}
	if err == nil {
		err = d.verify()
	}
	if err != nil {
		d.close()
		return nil, scratchProblem("a private directory was moved or replaced as it was made")
	}
	lock, err := d.root.OpenFile(ownerName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		d.close()
		return nil, scratchProblem("cannot make a private directory's owner file (" + osProblem(err) + ")")
	}
	if locked, err := tryLock(lock); err != nil || !locked {
		_ = lock.Close()
		d.close()
		return nil, scratchProblem("cannot lock a private directory")
	}
	d.lock = lock
	if d.lockKey, _, err = keyOf(lock); err == nil {
		_, err = lock.WriteAt([]byte(d.record()), 0)
	}
	if err != nil {
		_ = d.remove()
		return nil, scratchProblem("cannot write a private directory's owner file (" + osProblem(err) + ")")
	}
	if testHookPrivateDir != nil {
		testHookPrivateDir(d.path)
	}
	return d, nil
}

// record is what the owner file says.
func (d *privateDir) record() string {
	return recordHeader + "directory " + string(d.key) + "\nowner " + string(d.lockKey) + "\n"
}

func (d *privateDir) recordMatches() bool {
	want := d.record()
	b := make([]byte, len(want)+1)
	n, err := d.lock.ReadAt(b, 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return false
	}
	return string(b[:n]) == want
}

// join names a file in the directory, for git.
func (d *privateDir) join(elem ...string) string {
	return filepath.Join(append([]string{d.path}, elem...)...)
}

// verify fails unless the path still leads, through the directories the
// Scratch recorded and without a link or junction, to this directory.
func (d *privateDir) verify() error {
	if err := d.s.verify(); err != nil {
		return err
	}
	if key, dir, err := lstatKey(d.path); err != nil || !dir || key != d.key {
		return scratchProblem("a private directory was moved or replaced while in use")
	}
	return nil
}

// remove removes the directory: what is in it, then its owner file, then
// the directory itself. Every attempt starts with verify; once that fails,
// or an entry to remove is not the file it was, nothing more is removed,
// the read's own handles are released, and the read fails. On Windows a
// file a git process has just closed can stay in use for a moment, so an
// attempt that fails otherwise is tried again briefly; when they all fail,
// the read fails too (the spec: a read whose directory cannot be removed
// is not reported clean), and a directory that still has its owner file
// is left for a later Scratch to remove.
func (d *privateDir) remove() error {
	var err error
	for attempt := range removeAttempts {
		if attempt > 0 {
			time.Sleep(50 * time.Millisecond)
		}
		if verr := d.verify(); verr != nil {
			d.close()
			return verr
		}
		if err = d.removeOnce(); err == nil {
			return nil
		}
		if errors.Is(err, errReplaced) {
			d.close()
			return scratchProblem("a private directory, or its owner file, was moved or replaced as it was removed")
		}
	}
	d.close()
	return apperr.New(apperr.Environment, "cannot remove a private directory Specmetry read git with ("+osProblem(err)+")")
}

func (d *privateDir) removeOnce() error {
	if !d.ownerGone {
		if err := d.hook("contents"); err != nil {
			return err
		}
		entries, err := fs.ReadDir(d.root.FS(), ".")
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.Name() != ownerName {
				if err := d.root.RemoveAll(e.Name()); err != nil {
					return err
				}
			}
		}
		if err := removeEntry(d.dir, d.path, ownerName, d.lockKey, false); err != nil {
			return err
		}
		d.ownerGone = true
		d.closeLock()
	}
	if err := d.hook("entry"); err != nil {
		return err
	}
	err := removeEntry(d.s.dir, d.s.path, d.name, d.key, true)
	if err == nil {
		d.close()
	}
	return err
}

func (d *privateDir) hook(step string) error {
	if testHookRemove == nil {
		return nil
	}
	return testHookRemove(step, d.path)
}

// close releases the read's handles: the lock, then the directory.
func (d *privateDir) close() {
	if d == nil {
		return
	}
	d.closeLock()
	if d.dir != nil {
		_ = d.dir.Close()
		d.dir = nil
	}
	if d.root != nil {
		_ = d.root.Close()
		d.root = nil
	}
}

func (d *privateDir) closeLock() {
	if d.lock != nil {
		_ = unlock(d.lock)
		_ = d.lock.Close()
		d.lock = nil
	}
}
