package gitx

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
)

// Scheme B (owner approval in review-resolution §71): cooperating processes
// serialize a private directory's entire lifetime with a persistent namespace
// lock. v3 lives below tmp/git-reader-v3; v1/v2 content is never adopted.
// Other-user protection still comes from datadir. Uncooperative same-user
// namespace changes (including mistakes) are excluded, not prevented by these
// checks: Unix mkdir/open and final unlink windows remain, and Clean can consume
// replacement inputs. Observed anomalies fail and preserve the objects.
// Scan reads use empty graft/shallow overrides, never a private missing path.

const (
	privateNameLen = 32 // hex digits
	// ownerName is the owner file in each private directory; git never
	// makes a file of that name.
	ownerName        = "specmetry-owner"
	recordHeader     = "specmetry private directory 3"
	scratchNamespace = "git-reader-v3"
	leaseName        = "namespace.lock"
	removeAttempts   = 20
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
// data directory's v3 namespace, checked when it was made, and held open
// until Close.
type Scratch struct {
	path     string   // tmp/git-reader-v3, resolved
	chain    []dirKey // every directory from the volume root through the namespace
	root     *os.Root // namespace; nil once closed
	dir      *os.File // namespace itself, open
	witness  *os.File // persistent protocol lock, never unlinked
	leaseKey fileKey
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
	if _, err := d.EnsureSub(datadir.TmpDir); err != nil {
		return nil, err
	}
	tmp, err := d.EnsureSub(filepath.Join(datadir.TmpDir, scratchNamespace))
	if err != nil {
		return nil, err
	}
	s, err := openScratch(tmp)
	if err != nil {
		return nil, err
	}
	if err := s.initProtocol(); err != nil {
		_ = s.Close()
		return nil, err
	}
	if err := s.removeLeftovers(); err != nil {
		_ = s.Close()
		return nil, err
	}
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
	if s.witness != nil {
		err = s.witness.Close()
		s.witness = nil
	}
	if s.dir != nil {
		err = errors.Join(err, s.dir.Close())
	}
	err = errors.Join(err, s.root.Close())
	s.root, s.dir = nil, nil
	return err
}

// verify fails unless every directory on the path to tmp is, at its path
// and not through a link or junction, the one recorded.
func (s *Scratch) verify() error {
	if s == nil || s.root == nil {
		return apperr.New(apperr.Internal, "a git read needs an open Scratch")
	}
	for _, c := range s.chain {
		if key, dir, err := lstatKey(c.path); err != nil || !dir || key != c.key {
			return scratchProblem("the data directory's tmp directory, or a directory on the path to it, was moved or replaced since it was checked")
		}
	}
	if s.witness != nil {
		k, dir, err := lstatKey(filepath.Join(s.path, leaseName))
		if err != nil || dir || k != s.leaseKey {
			return scratchProblem("the protocol lock was replaced")
		}
	}
	return nil
}

func scratchProblem(what string) error {
	return apperr.New(apperr.Environment, what+"; the git operation did not complete and no result is accepted")
}

// initProtocol establishes a persistent, content-free coordination file. Its
// name is never removed on release, so cooperating processes cannot split locks.
func (s *Scratch) initProtocol() error {
	f, err := s.root.OpenFile(leaseName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		f, err = s.root.OpenFile(leaseName, os.O_RDWR, 0)
	}
	if err != nil {
		return scratchProblem("cannot open the v3 protocol lock")
	}
	s.witness = f
	k, dir, err := keyOf(f)
	if err != nil || dir {
		return scratchProblem("cannot identify the v3 protocol lock")
	}
	s.leaseKey = k
	// The empty file itself is the protocol identifier: no initialization write
	// can race a second cooperating process after exclusive creation.
	fi, err := f.Stat()
	if err != nil || fi.Size() != 0 {
		return scratchProblem("the v3 protocol lock has unknown content")
	}
	return s.verify()
}

// acquire returns a separately opened, nonblocking lease. The witness stays
// open for the Scratch lifetime; the lease spans one complete private read.
func (s *Scratch) acquire() (*os.File, error) {
	if err := s.verify(); err != nil {
		return nil, err
	}
	f, err := s.root.OpenFile(leaseName, os.O_RDWR, 0)
	if err != nil {
		return nil, scratchProblem("cannot open the protocol lease")
	}
	k, dir, err := keyOf(f)
	if err == nil && !dir && k == s.leaseKey {
		var ok bool
		ok, err = tryLock(f)
		if err == nil && !ok {
			err = errLeaseBusy
		}
	} else {
		err = errReplaced
	}
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if err = s.verify(); err != nil {
		releaseLease(f)
		return nil, err
	}
	return f, nil
}

var errLeaseBusy = errors.New("private namespace is busy")

func releaseLease(f *os.File) {
	if f != nil {
		_ = unlock(f)
		_ = f.Close()
	}
}

// removeLeftovers only adopts a complete v3 creation manifest, under the
// same lease as creators. A busy lease belongs to an active cooperating read.
func (s *Scratch) removeLeftovers() error {
	lease, err := s.acquire()
	if errors.Is(err, errLeaseBusy) {
		return nil
	}
	if err != nil {
		return scratchProblem("cannot acquire the protocol lease")
	}
	defer releaseLease(lease)
	return s.reapLocked()
}

// reapLocked is also run before every new lease creates content: a previous
// observed anomaly cannot be bypassed by reusing an already-open Scratch.
func (s *Scratch) reapLocked() error {
	entries, err := fs.ReadDir(s.root.FS(), ".")
	if err != nil {
		return scratchProblem("cannot read the private namespace")
	}
	var adopted []*privateDir
	defer func() {
		for _, d := range adopted {
			d.close()
		}
	}()
	for _, e := range entries {
		if e.Name() == leaseName {
			continue
		}
		if !privateName.MatchString(e.Name()) {
			return scratchProblem("an unknown entry is present in the private namespace; it was retained")
		}
		d := s.adopt(e.Name())
		if d == nil {
			return scratchProblem("a private directory has an unknown owner or manifest; it was retained")
		}
		adopted = append(adopted, d)
	}
	for _, d := range adopted {
		if err := d.remove(); err != nil {
			return err
		}
	}
	return nil
}
func (s *Scratch) adopt(name string) *privateDir {
	d, err := s.open(name)
	if err != nil {
		d.close()
		return nil
	}
	f, err := d.root.OpenFile(ownerName, os.O_RDWR, 0)
	if err != nil {
		d.close()
		return nil
	}
	d.lock = f
	k, dir, err := keyOf(f)
	if err != nil || dir {
		d.close()
		return nil
	}
	d.lockKey = k
	ok, err := tryLock(f)
	if err != nil || !ok || !d.loadRecord() || d.verify() != nil || d.checkContents() != nil {
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
	// consumerPending is durable before Start and cleared only after Wait (or
	// a failed Start) and complete input verification. Caller death leaves a
	// barrier: unlocked does not mean that every Git consumer has exited.
	consumerPending bool
	lease           *os.File
	entries         map[string]ownedEntry
	createHandle    *os.File
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

// ownedEntry records identity, not content. The manifest survives a crash.
type ownedEntry struct {
	Key fileKey `json:"key"`
	Dir bool    `json:"dir"`
}
type ownerRecord struct {
	Protocol        string                `json:"protocol"`
	Directory       fileKey               `json:"directory"`
	Owner           fileKey               `json:"owner"`
	Entries         map[string]ownedEntry `json:"entries"`
	ConsumerPending bool                  `json:"consumer_pending"`
}

func newPrivateDir(s *Scratch) (*privateDir, error) {
	if s == nil || s.root == nil {
		return nil, apperr.New(apperr.Internal, "a git read needs an open Scratch")
	}
	lease, err := s.acquire()
	if err != nil {
		if apperr.ExitCode(err) == apperr.ExitEnvironment {
			return nil, err
		}
		return nil, scratchProblem("cannot acquire the private namespace lease")
	}
	if err := s.reapLocked(); err != nil {
		releaseLease(lease)
		return nil, err
	}
	var b [privateNameLen / 2]byte
	if _, err = rand.Read(b[:]); err != nil {
		releaseLease(lease)
		return nil, scratchProblem("cannot name a private directory")
	}
	name := "read-" + hex.EncodeToString(b[:])
	created, err := createDirectory(s, name)
	if err != nil {
		releaseLease(lease)
		return nil, scratchProblem("cannot exclusively create a private directory; it may have been moved or replaced")
	}
	d, err := s.open(name)
	d.createHandle, d.lease = created, lease
	if err == nil {
		var k fileKey
		k, _, err = keyOf(created)
		if err == nil && k != d.key {
			err = errReplaced
		}
	}
	if err == nil {
		var es []fs.DirEntry
		es, err = fs.ReadDir(d.root.FS(), ".")
		if len(es) != 0 {
			err = errReplaced
		}
	}
	if err == nil {
		err = d.verify()
	}
	if err != nil {
		d.close()
		return nil, scratchProblem("a private directory was replaced as it was created")
	}
	d.entries = make(map[string]ownedEntry)
	d.lock, err = d.root.OpenFile(ownerName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err == nil {
		var ok bool
		ok, err = tryLock(d.lock)
		if !ok && err == nil {
			err = errLeaseBusy
		}
	}
	if err == nil {
		d.lockKey, _, err = keyOf(d.lock)
	}
	if err == nil {
		err = d.saveRecord()
	}
	if err != nil {
		d.close()
		return nil, scratchProblem("cannot establish the private directory manifest")
	}
	if testHookPrivateDir != nil {
		testHookPrivateDir(d.path)
	}
	return d, nil
}
func (d *privateDir) record() string {
	b, _ := json.Marshal(ownerRecord{Protocol: recordHeader, Directory: d.key, Owner: d.lockKey, Entries: d.entries, ConsumerPending: d.consumerPending})
	return string(b) + "\n"
}
func (d *privateDir) saveRecord() error {
	b := []byte(d.record())
	if err := d.lock.Truncate(0); err != nil {
		return err
	}
	n, err := d.lock.WriteAt(b, 0)
	if err == nil && n != len(b) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return err
	}
	return d.lock.Sync()
}
func (d *privateDir) loadRecord() bool {
	const limit = 1 << 20
	fi, err := d.lock.Stat()
	if err != nil || fi.Size() <= 0 || fi.Size() > limit {
		return false
	}
	b := make([]byte, int(fi.Size()))
	if _, err := d.lock.ReadAt(b, 0); err != nil {
		return false
	}
	var r ownerRecord
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if dec.Decode(&r) != nil || dec.Decode(new(any)) != io.EOF || r.Protocol != recordHeader || r.Directory != d.key || r.Owner != d.lockKey || r.Entries == nil || r.ConsumerPending {
		return false
	}
	for name, e := range r.Entries {
		if !fs.ValidPath(name) || name == "." || name == ownerName || strings.Contains(name, `\`) || e.Key == "" {
			return false
		}
	}
	d.entries = r.Entries
	return true
}

// register is called on the handle returned by exclusive creation, before
// data is written. Unknown partial creations are retained after a crash.
func (d *privateDir) register(name string, f *os.File) error {
	key, dir, err := keyOf(f)
	if err != nil {
		return err
	}
	d.entries[filepath.ToSlash(name)] = ownedEntry{key, dir}
	return d.saveRecord()
}
func (d *privateDir) mkdirAll(name string) error {
	var current string
	for _, part := range strings.Split(filepath.ToSlash(name), "/") {
		if current == "" {
			current = part
		} else {
			current += "/" + part
		}
		if _, ok := d.entries[current]; ok {
			continue
		}
		if err := d.checkOwner(); err != nil {
			return err
		}
		if err := d.root.Mkdir(filepath.FromSlash(current), 0o700); err != nil {
			return err
		}
		f, err := d.root.Open(filepath.FromSlash(current))
		if err != nil {
			return err
		}
		err = d.register(current, f)
		_ = f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
func (d *privateDir) create(name string) (*os.File, error) {
	if err := d.checkOwner(); err != nil {
		return nil, err
	}
	f, err := d.root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	if err = d.register(name, f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}
func (d *privateDir) write(name string, data []byte) error {
	f, err := d.create(filepath.FromSlash(name))
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return errors.Join(err, f.Close())
}

// checkOwner verifies the NAME before using the still-open owner handle. A
// missing name, another object, or a record that no longer matches is not owned.
func (d *privateDir) checkOwner() error {
	if d.ownerGone || d.lock == nil || d.root == nil {
		return errReplaced
	}
	if err := d.verify(); err != nil {
		return err
	}
	f, err := d.root.Open(ownerName)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	k, dir, err := keyOf(f)
	if err != nil || dir || k != d.lockKey {
		return errReplaced
	}
	pathKey, pathDir, err := lstatKey(d.join(ownerName))
	if err != nil || pathDir || pathKey != k {
		return errReplaced
	}
	want := d.record()
	b := make([]byte, len(want)+1)
	n, err := f.ReadAt(b, 0)
	if (err != nil && !errors.Is(err, io.EOF)) || string(b[:n]) != want {
		return errReplaced
	}
	return nil
}

// checkContents validates the WHOLE tree before deletion. No guessed recursive
// removal: extra files, links, identity changes and missing entries are errors.
func (d *privateDir) checkContents() error {
	if err := d.checkOwner(); err != nil {
		return err
	}
	seen := make(map[string]bool)
	ownerSeen := false
	err := fs.WalkDir(d.root.FS(), ".", func(name string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}
		if e.Type()&os.ModeSymlink != 0 || (e.Type() != 0 && !e.IsDir()) {
			return errReplaced
		}
		key, dir, err := lstatKey(d.join(filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		if name == ownerName {
			if key != d.lockKey || dir {
				return errReplaced
			}
			ownerSeen = true
			return nil
		}
		want, ok := d.entries[name]
		if !ok || key != want.Key || dir != want.Dir {
			return errReplaced
		}
		seen[name] = true
		return nil
	})
	if err != nil {
		return err
	}
	if !ownerSeen || len(seen) != len(d.entries) {
		return errReplaced
	}
	return nil
}

// beginConsumer establishes the conservative crash barrier before any child can
// consume this directory. A torn/failed write is also unadoptable; no child starts.
func (d *privateDir) beginConsumer() error {
	if d.consumerPending {
		return scratchProblem("a private Git consumer has no confirmed completion; its directory was retained")
	}
	if err := d.checkContents(); err != nil {
		return scratchProblem("private inputs changed; the directory was retained")
	}
	d.consumerPending = true
	if err := d.saveRecord(); err != nil {
		return scratchProblem("cannot persist the private Git consumer barrier; its directory was retained")
	}
	return nil
}

// finishConsumer is called only after commandIO has returned from Start failure
// or Wait. Failed verification leaves the barrier in place, never a false proof.
func (d *privateDir) finishConsumer() error {
	if err := d.checkContents(); err != nil {
		return scratchProblem("private inputs changed; the directory was retained")
	}
	d.consumerPending = false
	if err := d.saveRecord(); err != nil {
		return scratchProblem("cannot persist the private Git consumer completion; its directory was retained")
	}
	return nil
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
	if d.consumerPending {
		d.close()
		return scratchProblem("a private Git consumer has no confirmed completion; its directory was retained")
	}
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
		if err := d.checkContents(); err != nil {
			return errors.Join(errReplaced, err)
		}
		names := make([]string, 0, len(d.entries))
		for name := range d.entries {
			names = append(names, name)
		}
		// Descendants before parents; lexical tie breaking is not business order.
		slices.SortFunc(names, func(a, b string) int {
			if n := strings.Count(b, "/") - strings.Count(a, "/"); n != 0 {
				return n
			}
			return strings.Compare(a, b)
		})
		for _, name := range names {
			e := d.entries[name]
			parent := filepath.Dir(filepath.FromSlash(name))
			pf, err := d.root.Open(parent)
			if err != nil {
				return err
			}
			err = removeEntry(pf, d.join(parent), filepath.Base(name), e.Key, e.Dir)
			_ = pf.Close()
			if err != nil {
				return err
			}
			delete(d.entries, name)
			if err := d.saveRecord(); err != nil {
				return err
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
	err := removeDirectory(d)
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
	if d.createHandle != nil {
		_ = d.createHandle.Close()
		d.createHandle = nil
	}
	if d.dir != nil {
		_ = d.dir.Close()
		d.dir = nil
	}
	if d.root != nil {
		_ = d.root.Close()
		d.root = nil
	}
	releaseLease(d.lease)
	d.lease = nil
}

func (d *privateDir) closeLock() {
	if d.lock != nil {
		_ = unlock(d.lock)
		_ = d.lock.Close()
		d.lock = nil
	}
}

// removeOwned is for an intentional probe mutation, still under its lease.
func (d *privateDir) removeOwned(name string) error {
	if err := d.checkContents(); err != nil {
		return err
	}
	e, ok := d.entries[name]
	if !ok {
		return errReplaced
	}
	parent := filepath.Dir(filepath.FromSlash(name))
	f, err := d.root.Open(parent)
	if err != nil {
		return err
	}
	err = removeEntry(f, d.join(parent), filepath.Base(name), e.Key, e.Dir)
	_ = f.Close()
	if err != nil {
		return err
	}
	delete(d.entries, name)
	return d.saveRecord()
}
