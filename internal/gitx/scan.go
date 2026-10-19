package gitx

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// The reads of the commit scan (WP2.3 spec §5.1, §5.2). git can walk
// history without reading the commit objects, so that a missing or damaged
// object goes unnoticed (review 052 B1-R); every read here closes those
// ways: the commit-graph and the multi-pack-index are out of use, replace
// objects already are (globalArgs), and grafts and the shallow file point
// to a path that does not exist, so that neither a grafts file nor a
// shallow file that appears after CheckShallow cuts a walk short. Whatever
// git visits, it reads and parses. The scan starts with CheckShallow.

// scanArgs come first in every scan read.
var scanArgs = []string{"-c", "core.commitGraph=false", "-c", "core.multiPackIndex=false"}

// scanCall runs c as a scan read: an object read with scanArgs, and
// GIT_GRAFT_FILE and GIT_SHALLOW_FILE naming a file that does not exist,
// in a private directory of its own made for the call in s (only the
// current user can make anything there; its path is checked to lead to
// the directory made before git runs, and again as it is removed).
func scanCall(ctx context.Context, s *Scratch, c call) (result, error) {
	d, err := newPrivateDir(s)
	if err != nil {
		return result{}, err
	}
	if err := d.verify(); err != nil {
		_ = d.remove()
		return result{}, err
	}
	missing := d.join("none")
	c.objects = true
	c.args = append(slices.Clone(scanArgs), c.args...)
	c.env = append(c.env, "GIT_GRAFT_FILE="+missing, "GIT_SHALLOW_FILE="+missing)
	r, err := c.run(ctx)
	if rerr := d.remove(); err == nil {
		err = rerr
	}
	return r, err
}

// CheckShallow returns an environment error for a shallow repository, or
// one whose "shallow" entry in the common git directory cannot be shown
// absent (decision P3: the MVP scans no shallow clone). It reads that
// entry with lstat itself: git's --is-shallow-repository says false for a
// shallow file it cannot read or a directory (experiments H6, H7). The
// entry is never read or changed, nor is anything fetched.
func CheckShallow(ctx context.Context, worktree string) error {
	out, err := call{label: "rev-parse", dir: worktree,
		args: []string{"rev-parse", "--path-format=absolute", "--git-common-dir"}}.must(ctx)
	if err != nil {
		return err
	}
	common := filepath.FromSlash(strings.TrimRight(string(out), "\r\n"))
	cannot := apperr.New(apperr.Environment,
		"cannot tell whether this repository is a shallow clone, and Specmetry does not scan the commits of a shallow clone (WP2.3 decision P3); check the repository's git directory")
	if strings.ContainsAny(common, "\r\n") || !filepath.IsAbs(common) {
		return cannot
	}
	if fi, err := os.Lstat(common); err != nil || !fi.IsDir() {
		return cannot
	}
	_, err = os.Lstat(filepath.Join(common, "shallow"))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return cannot
	}
	return apperr.New(apperr.Environment,
		"this repository is a shallow clone, and Specmetry does not scan the commits of a shallow clone (WP2.3 decision P3): nothing was scanned or recorded; Specmetry fetches nothing, so make the repository complete (for example git fetch --unshallow) to scan it")
}

// Tip is a scanned ref: a local branch, or the detached HEAD of a worktree.
type Tip struct {
	Ref          string // "refs/heads/<name>", or "" for a detached HEAD
	WorktreePath string // a detached HEAD's worktree, as gitx.Canonical gives it
	Commit       string // full SHA-1
}

// Tips lists the local branches that point to commits, by name, and then
// the detached HEADs of the non-bare worktrees (prunable ones included),
// by path (WP2.3 spec §5.1, D2). A branch that points to another kind of
// object is left out; one whose object cannot be read fails the listing.
// A git without `worktree list -z` is an environment error: the list is
// never read line by line, since a path may hold a line break.
func Tips(ctx context.Context, s *Scratch, worktree string) ([]Tip, error) {
	r, err := scanCall(ctx, s, call{label: "for-each-ref", dir: worktree,
		args: []string{"for-each-ref", "--format=%(refname)%00%(objectname)%00%(objecttype)", "refs/heads"}})
	if err != nil {
		return nil, err
	}
	if r.code != 0 {
		return nil, failed("for-each-ref", r.code)
	}
	tips, err := parseBranches(r.out)
	if err != nil {
		return nil, err
	}
	wr, err := call{label: "worktree list", dir: worktree, args: []string{"worktree", "list", "--porcelain", "-z"}}.run(ctx)
	switch {
	case err != nil:
		return nil, err
	case wr.code == 129:
		return nil, apperr.New(apperr.Environment,
			"this git cannot list worktrees with `git worktree list -z` (git 2.36 or newer can), which the commit scan needs to read the detached HEADs safely; upgrade git")
	case wr.code != 0:
		return nil, failed("worktree list", wr.code)
	}
	detached, err := parseDetached(wr.out)
	if err != nil {
		return nil, err
	}
	return append(tips, detached...), nil
}

// parseBranches reads for-each-ref's lines "<ref>\0<id>\0<type>".
func parseBranches(out []byte) ([]Tip, error) {
	bad := apperr.New(apperr.Environment, "git for-each-ref printed something other than branches")
	var tips []Tip
	for _, line := range strings.Split(strings.TrimSuffix(string(out), "\n"), "\n") {
		if line == "" && len(out) == 0 {
			break
		}
		f := strings.Split(line, "\x00")
		if len(f) != 3 || !strings.HasPrefix(f[0], "refs/heads/") || !commitID.MatchString(f[1]) {
			return nil, bad
		}
		switch f[2] {
		case "commit":
			tips = append(tips, Tip{Ref: f[0], Commit: f[1]})
		case "tree", "blob", "tag":
		default:
			return nil, bad
		}
	}
	slices.SortFunc(tips, func(a, b Tip) int { return strings.Compare(a.Ref, b.Ref) })
	return tips, nil
}

// parseDetached reads `worktree list --porcelain -z`: records of
// NUL-terminated fields, each record ended by an empty field, its first
// field "worktree <path>". A detached, non-bare record must give its HEAD.
func parseDetached(out []byte) ([]Tip, error) {
	bad := apperr.New(apperr.Environment, "git worktree list printed something Specmetry cannot read")
	if len(out) == 0 {
		return nil, bad // there is always the main worktree
	}
	if !bytes.HasSuffix(out, []byte{0, 0}) {
		return nil, bad
	}
	seen := map[string]string{}
	var tips []Tip
	for _, rec := range strings.Split(strings.TrimSuffix(string(out), "\x00\x00"), "\x00\x00") {
		fields := strings.Split(rec, "\x00")
		p, ok := strings.CutPrefix(fields[0], "worktree ")
		if !ok || p == "" {
			return nil, bad
		}
		var head string
		var detached, bare bool
		for _, f := range fields[1:] {
			switch {
			case strings.HasPrefix(f, "HEAD "):
				head = strings.TrimPrefix(f, "HEAD ")
			case f == "detached":
				detached = true
			case f == "bare":
				bare = true
			}
		}
		if !detached || bare {
			continue
		}
		if !commitID.MatchString(head) {
			return nil, bad
		}
		name := Canonical(p)
		if prev, dup := seen[name]; dup {
			if prev != head {
				return nil, apperr.New(apperr.Environment, "git worktree list gave one worktree path two detached HEADs")
			}
			continue
		}
		seen[name] = head
		tips = append(tips, Tip{WorktreePath: name, Commit: head})
	}
	slices.SortFunc(tips, func(a, b Tip) int { return strings.Compare(a.WorktreePath, b.WorktreePath) })
	return tips, nil
}

// Presence is what the object database says of a commit ID.
type Presence int

const (
	Present Presence = iota // the object header says commit; the full read comes later
	Absent                  // reliably not in the object database (§5.2, step 2)
)

// Classify classifies commit IDs (WP2.3 spec §5.2 step 2): Present or
// Absent. Absent needs git's protocol answers (`missing` from
// --batch-check, then 1 from a bare-ID cat-file -e), no stderr output at
// all from either, and the storage check; anything else (present but
// unreadable, another type, an unexpected answer, any stderr output, a
// failed storage check) is an environment error.
func Classify(ctx context.Context, s *Scratch, worktree string, ids []string) (map[string]Presence, error) {
	var unique []string
	for _, id := range ids {
		if !commitID.MatchString(id) {
			return nil, apperr.New(apperr.Internal, "Classify needs full commit IDs")
		}
		if !slices.Contains(unique, id) {
			unique = append(unique, id)
		}
	}
	got := map[string]Presence{}
	if len(unique) == 0 {
		return got, nil
	}
	r, err := scanCall(ctx, s, call{label: "cat-file", dir: worktree,
		args: []string{"cat-file", "--batch-check"}, stdin: []byte(strings.Join(unique, "\n") + "\n")})
	if err != nil {
		return nil, err
	}
	if r.code != 0 {
		return nil, failed("cat-file", r.code)
	}
	lines := strings.Split(strings.TrimSuffix(string(r.out), "\n"), "\n")
	if len(lines) != len(unique) {
		return nil, apperr.New(apperr.Environment, "git cat-file --batch-check gave an answer Specmetry cannot read")
	}
	unconfirmed := apperr.New(apperr.Environment,
		"git says a commit the scan needs to know about is missing, but it cannot be shown to be gone rather than unreadable (a damaged or incomplete object storage, or one it cannot read); nothing was scanned; run git fsck in this directory")
	var missing []string
	for i, line := range lines {
		f := strings.Fields(line)
		switch {
		case len(f) == 3 && f[0] == unique[i] && f[1] == "commit" && isDecimal(f[2]):
			got[unique[i]] = Present
		case len(f) == 2 && f[0] == unique[i] && f[1] == "missing":
			missing = append(missing, unique[i])
		default:
			return nil, unconfirmed
		}
	}
	if len(missing) == 0 {
		return got, nil
	}
	if r.stderr {
		return nil, unconfirmed
	}
	for _, id := range missing {
		e, err := scanCall(ctx, s, call{label: "cat-file", dir: worktree, args: []string{"cat-file", "-e", id}})
		if err != nil {
			return nil, err
		}
		if e.code != 1 || e.stderr {
			return nil, unconfirmed // 0: present but unreadable (R6, K11)
		}
	}
	objects, err := call{label: "rev-parse", dir: worktree,
		args: []string{"rev-parse", "--path-format=absolute", "--git-path", "objects"}}.must(ctx)
	if err != nil {
		return nil, err
	}
	dir := filepath.FromSlash(strings.TrimRight(string(objects), "\r\n"))
	if strings.ContainsAny(dir, "\r\n") || !filepath.IsAbs(dir) || checkStorage(dir) != nil {
		return nil, unconfirmed
	}
	for _, id := range missing {
		got[id] = Absent
	}
	return got, nil
}

func isDecimal(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// IsAncestor reports whether ancestor, a Present commit, is an ancestor of
// commit; any answer but yes (0) or no (1) is an environment error.
func IsAncestor(ctx context.Context, s *Scratch, worktree, ancestor, commit string) (bool, error) {
	if !commitID.MatchString(ancestor) || !commitID.MatchString(commit) {
		return false, apperr.New(apperr.Internal, "IsAncestor needs full commit IDs")
	}
	r, err := scanCall(ctx, s, call{label: "merge-base", dir: worktree, args: []string{"merge-base", "--is-ancestor", ancestor, commit}})
	if err != nil {
		return false, err
	}
	switch r.code {
	case 0:
		return true, nil
	case 1:
		return false, nil
	}
	return false, failed("merge-base", r.code)
}

// Commit is one commit read by Log. Trailers and Message are only in
// memory: callers never store, log or echo them.
type Commit struct {
	SHA      string
	Time     int64 // committer time, seconds, as git gives it
	Trailers string
	Message  string
}

// logFormat gives a commit's SHA, committer time, Specmetry-Ref trailer
// values (git decides what a trailer is, its key in any case) and whole
// message, each NUL-terminated with -z.
const logFormat = "--format=%H%x00%ct%x00%(trailers:key=Specmetry-Ref,valueonly,separator=%x2C)%x00%B"

// Log strictly reads the commits reachable from tip and from none of
// exclude (Present commits), newest first, calling fn for each, with the
// commit-graph, grafts, the shallow file and the multi-pack-index out of
// use (WP2.3 spec §5.2 step 4): the tip, then ^<sha> per exclusion, on
// stdin, without --ignore-missing. Any object it cannot read is an
// environment error, and a caller keeps nothing fn received before it: git
// may print some commits before it fails (G3). log.showSignature is
// turned off, so that no gpg.program runs. fn's error stops the read and
// is returned.
func Log(ctx context.Context, s *Scratch, worktree, tip string, exclude []string, fn func(Commit) error) error {
	if !commitID.MatchString(tip) {
		return apperr.New(apperr.Internal, "Log needs full commit IDs")
	}
	in := tip + "\n"
	for _, e := range exclude {
		if !commitID.MatchString(e) {
			return apperr.New(apperr.Internal, "Log needs full commit IDs")
		}
		in += "^" + e + "\n"
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	lr := &logReader{fn: fn, stop: cancel}
	r, err := scanCall(ctx, s, call{label: "log", dir: worktree, stdin: []byte(in), stdout: lr,
		args: []string{"log", "-z", "--stdin", "--no-show-signature", logFormat}})
	if lr.err != nil {
		return lr.err
	}
	if err != nil {
		return err
	}
	if r.code != 0 {
		return failed("log", r.code)
	}
	return lr.close()
}

// maxCommitTime bounds a committer time so that it fits in milliseconds.
const maxCommitTime = math.MaxInt64 / 1000

// logReader parses log -z's output as it comes: four NUL-terminated fields
// a commit. It passes each commit to fn and keeps nothing of it.
type logReader struct {
	fn     func(Commit) error
	stop   func()
	buf    []byte
	fields []string
	err    error
}

func (r *logReader) Write(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	r.buf = append(r.buf, p...)
	for {
		i := bytes.IndexByte(r.buf, 0)
		if i < 0 {
			break
		}
		r.fields = append(r.fields, string(r.buf[:i]))
		r.buf = r.buf[i+1:]
		if len(r.fields) == 4 {
			if err := r.commit(); err != nil {
				r.err = err
				if r.stop != nil {
					r.stop()
				}
				return 0, err
			}
		}
	}
	return len(p), nil
}

func (r *logReader) commit() error {
	f := r.fields
	r.fields = r.fields[:0]
	t, err := strconv.ParseInt(f[1], 10, 64)
	if !commitID.MatchString(f[0]) || !isDecimal(f[1]) || err != nil || t > maxCommitTime {
		return apperr.New(apperr.Environment, "git log printed a commit Specmetry cannot read")
	}
	return r.fn(Commit{SHA: f[0], Time: t, Trailers: f[2], Message: f[3]})
}

// close checks that the output ended after a whole commit.
func (r *logReader) close() error {
	if r.err != nil {
		return r.err
	}
	if len(r.buf) != 0 || len(r.fields) != 0 {
		return apperr.New(apperr.Environment, "git log's output ended inside a commit")
	}
	return nil
}
