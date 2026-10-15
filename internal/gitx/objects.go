package gitx

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
)

// Object reads (PD-26). A git that reads an object a partial clone lacks
// fetches it from the promisor remote on demand, a lazy fetch, which may
// reach a network, run a program or write the repository's configuration.
// Specmetry never fetches, so a command that may read an object runs only
// with a git that running it has shown never to: one that honours
// GIT_NO_LAZY_FETCH, which controlledEnv sets for every command. Which git
// does is found out from its behaviour, not from its version or the
// options it knows (a distribution's git 2.43 honours the variable without
// knowing --no-lazy-fetch). With any other git, or when it cannot be found
// out, every such command is refused with an environment error (exit 6)
// before it runs, in every repository: whether a repository is a partial
// clone can change between a check and a read (codex-review-022).
//
// Every command that may read an object runs through runObjects,
// runObjectsCode or runObjectsTo, and future ones must too: rev-parse of a
// ref's commit, ls-tree, cat-file, and ls-files (git expands a sparse index
// from the trees it names). Commands that read only refs, the
// configuration or the git directory's layout (rev-parse --show-toplevel
// and --git-path, worktree list, symbolic-ref, config) run through run and
// runCode.

// objectSafety is what running git showed about missing objects.
type objectSafety int

const (
	// unprobed: nothing is known yet.
	unprobed objectSafety = iota
	// neverFetches: with GIT_NO_LAZY_FETCH set, git did not fetch.
	neverFetches
	// fetches: git fetched although GIT_NO_LAZY_FETCH was set.
	fetches
)

// The messages of a refused object read: built anew each time, since an
// apperr.Error can be added to.
const (
	msgFetches      = "this git does not honour GIT_NO_LAZY_FETCH: it would fetch the objects a partial clone lacks on demand, so Specmetry reads no git objects with it (PD-26); upgrade git to a release that honours GIT_NO_LAZY_FETCH"
	msgUnknown      = "cannot tell whether this git would fetch the objects a partial clone lacks on demand, so Specmetry reads no git objects with it (PD-26); upgrade git to a release that honours GIT_NO_LAZY_FETCH"
	msgProbeTimeout = "checking whether this git would fetch missing objects did not finish in time, so Specmetry reads no git objects with it (PD-26); run the command again"
)

func refused(message string) error { return apperr.New(apperr.Environment, message) }

// CheckObjectReads returns nil when git may read objects (PD-26); otherwise
// an environment error. It runs no command in a repository, so it can come
// before anything else that would read one.
func CheckObjectReads(ctx context.Context) error {
	_, err := objectGit(ctx)
	return err
}

// objectGit returns the git to run for a command that may read an object:
// one the probe has shown never to fetch. The probe runs the first time
// a git is asked for, and what it showed is kept while PATH finds the same
// git (gitBinary).
func objectGit(ctx context.Context) (string, error) {
	stateMu.Lock()
	defer stateMu.Unlock()
	return objectGitLocked(ctx)
}

// objectGitLocked is objectGit with stateMu held.
func objectGitLocked(ctx context.Context) (string, error) {
	git, err := gitToRunLocked(ctx)
	if err != nil {
		return "", err
	}
	if state.objects == unprobed {
		safety, err := probe(ctx, git)
		if err != nil {
			return "", err
		}
		state.objects = safety
	}
	if state.objects != neverFetches {
		return "", refused(msgFetches)
	}
	return git, nil
}

// sparseListing is whether git's ls-files knows --sparse.
type sparseListing int

const (
	sparseUnprobed sparseListing = iota
	sparseKnown
	sparseUnknown
)

const msgNoSparse = "this git's ls-files does not know --sparse, so listing a repository with a sparse index would expand the index and rewrite the timestamps of object files; Specmetry lists no files with it (PD-26); upgrade git"

// listGit returns the git to list a worktree's files with: one that never
// fetches (objectGit) and whose ls-files knows --sparse, so that a sparse
// index is listed without being expanded. Whether it knows the option is
// found out once per git (gitBinary), by running it outside any user's
// repository (probeSparse).
func listGit(ctx context.Context) (string, error) {
	stateMu.Lock()
	defer stateMu.Unlock()
	git, err := objectGitLocked(ctx)
	if err != nil {
		return "", err
	}
	if state.sparse == sparseUnprobed {
		known, err := probeSparse(ctx, git)
		if err != nil {
			return "", err
		}
		state.sparse = sparseUnknown
		if known {
			state.sparse = sparseKnown
		}
	}
	if state.sparse != sparseKnown {
		return "", refused(msgNoSparse)
	}
	return git, nil
}

// probeSparse finds out whether git's ls-files knows --sparse, by running
// `ls-files --sparse` in a worktree of its own: a new temporary directory
// holding a repository made by hand, with no index. It cannot be tried in
// a user's repository: git 2.32 reads (and expands) the index before it
// looks at its options, so a git that does not know the option would
// already have expanded a sparse index when it refuses it. Exit status 0
// means the option is known, 129 (a usage error) that it is not; anything
// else finds nothing out. The directory is removed whatever the outcome.
func probeSparse(ctx context.Context, git string) (bool, error) {
	parent := ctx
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	tmp, err := os.MkdirTemp("", "specmetry-git-probe-")
	if err != nil {
		return false, apperr.New(apperr.Environment,
			"cannot make a temporary directory to check git ("+osProblem(err)+"), so Specmetry lists no files with it")
	}
	defer removeProbeDir(tmp)
	dotGit := filepath.Join(tmp, ".git")
	for _, d := range []string{"objects/info", "objects/pack", "refs/heads", "refs/tags"} {
		if err := os.MkdirAll(filepath.Join(dotGit, filepath.FromSlash(d)), 0o700); err == nil {
			continue
		}
		return false, refused(msgUnknown)
	}
	if os.WriteFile(filepath.Join(dotGit, "HEAD"), []byte("ref: refs/heads/probe\n"), 0o600) != nil ||
		os.WriteFile(filepath.Join(dotGit, "config"), []byte("[core]\n\trepositoryformatversion = 0\n\tbare = false\n"), 0o600) != nil {
		return false, refused(msgUnknown)
	}
	code, err := command(ctx, git, tmp, filterEnv(os.Environ()), nil, append(slices.Clone(globalArgs), "ls-files", "-z", "--sparse")...)
	switch {
	case err != nil && parent.Err() == nil && errors.Is(ctx.Err(), context.DeadlineExceeded):
		return false, refused(msgProbeTimeout)
	case err != nil:
		return false, err
	case code == 0:
		return true, nil
	case code == 129:
		return false, nil
	}
	return false, refused(msgUnknown)
}

// runObjects is run for a command that may read an object.
func runObjects(ctx context.Context, dir string, args ...string) ([]byte, error) {
	out, code, err := runObjectsCode(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return nil, failed(args[0], code)
	}
	return out, nil
}

// runObjectsCode is runCode for a command that may read an object.
func runObjectsCode(ctx context.Context, dir string, args ...string) ([]byte, int, error) {
	git, err := objectGit(ctx)
	if err != nil {
		return nil, -1, err
	}
	return runWith(ctx, git, dir, args...)
}

// runObjectsTo runs a command that may read an object and must succeed,
// with its output written to stdout as it comes.
func runObjectsTo(ctx context.Context, dir string, stdout io.Writer, args ...string) error {
	git, err := objectGit(ctx)
	if err != nil {
		return err
	}
	code, err := execTo(ctx, git, dir, stdout, append(slices.Clone(globalArgs), args...)...)
	if err != nil {
		return err
	}
	if code != 0 {
		return failed(args[0], code)
	}
	return nil
}

// probeTimeout bounds the probe.
var probeTimeout = 30 * time.Second

// probeObject is the object the probe asks for, which its repository does
// not have.
const probeObject = "0123456789abcdef0123456789abcdef01234567"

// maxProbeEvents bounds the events read from one probe run.
const maxProbeEvents = 1 << 20

// probe finds out by running git whether it fetches a missing object
// although GIT_NO_LAZY_FETCH is set. In a new temporary directory, never in
// a user's repository, it makes a bare partial clone whose promisor remote
// is a directory that does not exist, and asks git for the type of an
// object the clone lacks, twice, each time in a fresh copy and with git's
// trace2 events (JSON, one event a line) written to a file:
//
//   - control, without GIT_NO_LAZY_FETCH: the events must show git
//     starting its fetch (a child_start event of its own whose argv holds
//     "fetch"), which shows that the repository makes git fetch and that
//     the events show a fetch;
//   - test, with GIT_NO_LAZY_FETCH=1: git never fetches when the events show
//     its own start and exit, with the failure status it exited with, and
//     no child process at all.
//
// Events are trusted only when they fit one consistent run of git
// (checkProbeEvents): any event of a child counts as a child, and a
// lifecycle event that is repeated, out of order or contradicts another
// makes the run show nothing. A test run whose events show a fetch is a git
// that fetches. Anything else (no events, a control without a fetch, events
// that are not trusted, a child that is not a fetch, an exit status that
// disagrees) shows nothing: an error, and nothing is kept. Both runs
// use the controlled environment otherwise, GIT_ALLOW_PROTOCOL= included,
// so the control's fetch is refused before it connects or runs a program.
// git's output is discarded, and the directory is removed whatever the
// outcome.
func probe(ctx context.Context, git string) (objectSafety, error) {
	parent := ctx
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	safety, err := probeIn(ctx, git)
	if err != nil && parent.Err() == nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return unprobed, refused(msgProbeTimeout)
	}
	return safety, err
}

func probeIn(ctx context.Context, git string) (objectSafety, error) {
	tmp, err := os.MkdirTemp("", "specmetry-git-probe-")
	if err != nil {
		return unprobed, apperr.New(apperr.Environment,
			"cannot make a temporary directory to check git ("+osProblem(err)+"), so Specmetry reads no git objects")
	}
	defer removeProbeDir(tmp)
	if tmp, err = filepath.Abs(tmp); err != nil {
		return unprobed, refused(msgUnknown)
	}
	control, err := probeRun(ctx, git, tmp, "control", false)
	if err != nil {
		return unprobed, err
	}
	if !control.fetched || !control.complete() {
		return unprobed, refused(msgUnknown)
	}
	test, err := probeRun(ctx, git, tmp, "test", true)
	switch {
	case err != nil:
		return unprobed, err
	case test.fetched:
		return fetches, nil
	case test.showsNoChild():
		return neverFetches, nil
	}
	return unprobed, refused(msgUnknown)
}

// probeResult is what one probe run's events show. readProbeEvents fills
// it only from a stream it trusts.
type probeResult struct {
	// started and exited: git's own start and exit events, once each and
	// in that order, the exit status the one git exited with, a failure.
	started, exited bool
	// children counts git's own child_start events; fetched is set when
	// one of them is a fetch. descendants is set when a process below git
	// wrote an event.
	children    int
	fetched     bool
	descendants bool
}

func (r probeResult) complete() bool { return r.started && r.exited }

// showsNoChild reports a complete run in which no child process shows in
// any form: the only evidence that git never fetches.
func (r probeResult) showsNoChild() bool {
	return r.complete() && r.children == 0 && !r.fetched && !r.descendants
}

// probeRun makes the repository name.git in tmp and asks git in it for
// probeObject, with GIT_NO_LAZY_FETCH=1 when noLazyFetch.
func probeRun(ctx context.Context, git, tmp, name string, noLazyFetch bool) (probeResult, error) {
	repo := filepath.Join(tmp, name+".git")
	if err := makeProbeRepo(repo, filepath.Join(tmp, "no-such-remote")); err != nil {
		return probeResult{}, apperr.New(apperr.Environment,
			"cannot make a repository to check git ("+osProblem(err)+"), so Specmetry reads no git objects")
	}
	events := filepath.Join(tmp, name+".events")
	args := append(slices.Clone(globalArgs), "--git-dir="+repo, "cat-file", "-t", probeObject)
	code, err := command(ctx, git, tmp, probeEnv(os.Environ(), events, noLazyFetch), nil, args...)
	if err != nil {
		return probeResult{}, err
	}
	r, ok := readProbeEvents(events, code)
	if !ok {
		return probeResult{}, refused(msgUnknown)
	}
	return r, nil
}

// makeProbeRepo writes a bare repository by hand, without running git: no
// object, and a promisor remote at a path that does not exist.
func makeProbeRepo(repo, remote string) error {
	for _, d := range []string{"objects/info", "objects/pack", "refs/heads", "refs/tags"} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.FromSlash(d)), 0o700); err != nil {
			return err
		}
	}
	url := filepath.ToSlash(remote)
	if strings.ContainsAny(url, "\n\r") {
		return errors.New("the temporary directory's path has a line break")
	}
	url = strings.ReplaceAll(strings.ReplaceAll(url, `\`, `\\`), `"`, `\"`)
	config := "[core]\n\trepositoryformatversion = 1\n\tbare = true\n" +
		"[extensions]\n\tpartialClone = origin\n" +
		"[remote \"origin\"]\n\turl = \"" + url + "\"\n\tpromisor = true\n"
	if err := os.WriteFile(filepath.Join(repo, "config"), []byte(config), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(repo, "HEAD"), []byte("ref: refs/heads/probe\n"), 0o600)
}

// probeEnv is the controlled environment with trace2 events written to
// events, and without GIT_NO_LAZY_FETCH unless noLazyFetch.
func probeEnv(env []string, events string, noLazyFetch bool) []string {
	var out []string
	for _, kv := range filterEnv(env) {
		if kv == "GIT_TRACE2_EVENT=0" || kv == "GIT_NO_LAZY_FETCH=1" && !noLazyFetch {
			continue
		}
		out = append(out, kv)
	}
	return append(out, "GIT_TRACE2_EVENT="+events)
}

// traceEvent holds the fields of a trace2 event the probe reads.
type traceEvent struct {
	Event   string   `json:"event"`
	SID     string   `json:"sid"`
	Argv    []string `json:"argv"`
	Code    *int     `json:"code"`
	ChildID *int     `json:"child_id"`
}

// readProbeEvents reads one probe run's events; code is the status git
// exited with. It reports false when the events cannot be trusted: a line
// that is not an event, too many events, or events that do not fit one
// consistent run (checkProbeEvents).
func readProbeEvents(path string, code int) (probeResult, bool) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return probeResult{}, true // no event: nothing shown
	}
	if err != nil {
		return probeResult{}, false
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxProbeEvents+1))
	if err != nil || len(data) > maxProbeEvents {
		return probeResult{}, false
	}
	var events []traceEvent
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	sc.Buffer(nil, maxProbeEvents)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e traceEvent
		if json.Unmarshal([]byte(line), &e) != nil || e.Event == "" || e.SID == "" {
			return probeResult{}, false
		}
		events = append(events, e)
	}
	if sc.Err() != nil {
		return probeResult{}, false
	}
	return checkProbeEvents(events, code)
}

// stage is how far git's own events have got.
type stage int

const (
	beforeStart stage = iota
	running           // after start
	exited            // after exit
	ended             // after atexit
)

// checkProbeEvents reads one probe run's events in the order they were
// written, and trusts them only when every event fits one consistent run
// of git, the process whose start event runs the probe's cat-file
// (codex-review-023 B1):
//
//   - git writes version at most once and before start; start once; exit
//     once, after start, with a status; atexit at most once, after exit,
//     with the same status; every other event of its own between start and
//     atexit;
//   - its child events come while it runs, before its exit: each
//     child_start with a child_id of its own; child_exit and child_ready
//     only for a child that started and has not exited, child_exit once;
//     no other child event; every child exits before git does;
//   - a process below git (its sid extends git's) writes only while git
//     runs, and there are no more such children than child_start events: a
//     child's events are never taken for no child;
//   - an event of any other process, an exit status other than the one git
//     exited with, or events without git's start are not trusted.
//
// code is the status git exited with. Anything a real git writes for the
// probe fits (git 2.32.0, 2.43, 2.56).
func checkProbeEvents(events []traceEvent, code int) (probeResult, bool) {
	root := ""
	for _, e := range events {
		if e.Event == "start" && slices.Contains(e.Argv, "cat-file") && slices.Contains(e.Argv, probeObject) {
			if root != "" {
				return probeResult{}, false
			}
			root = e.SID
		}
	}
	if root == "" {
		return probeResult{}, len(events) == 0 // no event: nothing shown
	}
	var r probeResult
	at, versions, status := beforeStart, 0, 0
	childExited := map[int]bool{} // by child_id
	below := map[string]bool{}    // git's children that wrote events, by sid
	for _, e := range events {
		switch {
		case e.SID == root:
			switch {
			case e.Event == "version":
				if at != beforeStart || versions > 0 {
					return probeResult{}, false
				}
				versions++
			case e.Event == "start":
				if at != beforeStart {
					return probeResult{}, false
				}
				at = running
			case e.Event == "exit":
				if at != running || e.Code == nil {
					return probeResult{}, false
				}
				for _, done := range childExited {
					if !done {
						return probeResult{}, false
					}
				}
				at, status = exited, *e.Code
			case e.Event == "atexit":
				if at != exited || e.Code == nil || *e.Code != status {
					return probeResult{}, false
				}
				at = ended
			case e.Event == "child_start":
				if at != running || e.ChildID == nil {
					return probeResult{}, false
				}
				if _, seen := childExited[*e.ChildID]; seen {
					return probeResult{}, false
				}
				childExited[*e.ChildID] = false
				r.children++
				r.fetched = r.fetched || slices.Contains(e.Argv, "fetch")
			case e.Event == "child_exit" || e.Event == "child_ready":
				if at != running || e.ChildID == nil {
					return probeResult{}, false
				}
				if done, seen := childExited[*e.ChildID]; !seen || done {
					return probeResult{}, false
				}
				childExited[*e.ChildID] = e.Event == "child_exit"
			case strings.HasPrefix(e.Event, "child_"):
				return probeResult{}, false
			default:
				if at != running && at != exited {
					return probeResult{}, false
				}
			}
		case strings.HasPrefix(e.SID, root+"/"):
			if at != running {
				return probeResult{}, false
			}
			r.descendants = true
			child, _, _ := strings.Cut(strings.TrimPrefix(e.SID, root+"/"), "/")
			below[child] = true
		default:
			return probeResult{}, false
		}
	}
	if len(below) > r.children || at >= exited && status != code {
		return probeResult{}, false
	}
	r.started = at != beforeStart
	r.exited = at >= exited && code != 0
	return r, true
}

// removeProbeDir removes the probe's directory. On Windows a file a git
// process has just closed can stay in use for a moment, so a failure is
// tried again briefly.
func removeProbeDir(dir string) {
	for range 20 {
		if os.RemoveAll(dir) == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
