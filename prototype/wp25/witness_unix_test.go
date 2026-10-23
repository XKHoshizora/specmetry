//go:build linux || darwin

package wp25

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// selfIdentity is this process's start time as procInfo reads it.
func selfIdentity() (string, error) {
	st, pr := procInfo(syscall.Getpid())
	if inHelper() && injected("witness-identity-unknown") {
		injectHit("selfIdentity", syscall.Getpid())
		pr = unknown // the read of its own state failed
	}
	if pr != present {
		return "", errUnknownState
	}
	return st.start, nil
}

// childIdentity is the start time of a child this process has just
// started and not reaped, so its PID still names it.
func childIdentity(cmd *exec.Cmd) (string, error) {
	st, pr := procInfo(cmd.Process.Pid)
	if pr != present {
		return "", errUnknownState
	}
	return st.start, nil
}

// raiseWitness tells the outer check, by a signal to it, that a record
// could not be made.
func raiseWitness() {
	if pid, err := strconv.Atoi(os.Getenv(raiseEnv)); err == nil && pid > 0 {
		_ = syscall.Kill(pid, syscall.SIGUSR1)
	}
}

// The outer check's side: SIGUSR1 is caught for the whole run once the
// first gate check starts (a late raise must not end the test process),
// and drained before each one.
var (
	raiseOnce sync.Once
	raiseCh   = make(chan os.Signal, 1)
)

type raiseWatch struct{}

func newRaiseWatch() (raiseWatch, error) {
	raiseOnce.Do(func() { signal.Notify(raiseCh, syscall.SIGUSR1) })
	for {
		select {
		case <-raiseCh:
			continue
		default:
		}
		return raiseWatch{}, nil
	}
}

func (raiseWatch) env() string { return raiseEnv + "=" + strconv.Itoa(os.Getpid()) }

func (raiseWatch) raised() bool {
	select {
	case <-raiseCh:
		return true
	case <-time.After(200 * time.Millisecond):
		return false
	}
}

func (raiseWatch) close() {}

// innerProc is the inner test process as the outer check started it.
type innerProc struct {
	cmd     *exec.Cmd
	pid     int
	start   string
	tag     string // WP25_WITNESS=<this check's file>, in the environment of every process of the run
	fault   string // an outer-side fault of the tested path (TestGateCombos)
	faulted bool
}

func launchInner(cmd *exec.Cmd, tag string) (*innerProc, error) {
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &innerProc{cmd: cmd, pid: cmd.Process.Pid, tag: tag}
	st, pr := procInfo(p.pid)
	if pr != present {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("the inner run's identity: %w", errUnknownState)
	}
	p.start = st.start
	return p, nil
}

// census lists the processes still alive that carry this check's tag:
// the inner run's, wherever they were reparented. There is no count of
// processes created on Unix (total -1).
func (p *innerProc) census() ([]survivor, int, error) {
	ss, err := censusTag(p.tag, p.start)
	return ss, -1, err
}

// outerSignal is how the outer check ends a process (its remediation and
// its insurance cleanup). A combination check replaces it to record what
// would be signalled (Review 079 B2).
var outerSignal = func(pid int) error { return syscall.Kill(pid, syscall.SIGKILL) }

// remediate ends the survivors, each only while its identity holds and it
// still carries the tag (this round's own processes), and returns the ones
// still alive after a bounded wait.
func (p *innerProc) remediate(ss []survivor) ([]survivor, error) {
	for _, v := range ss {
		if st, pr := procInfo(v.pid); pr == present && st.start == v.start && !st.zombie && tagged(v.pid, p.tag) {
			_ = outerSignal(v.pid)
		}
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		var still []survivor
		for _, v := range ss {
			a, err := recordAlive(witnessRec{pid: v.pid, start: v.start})
			if err != nil {
				return nil, err
			}
			if a {
				still = append(still, v)
			}
		}
		if len(still) == 0 || time.Now().After(deadline) {
			return still, nil
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (p *innerProc) close() {}

// ownedNow proves, when asked, that a process is this round's: still the
// same process (its start time), alive, and carrying this check's tag in
// its environment, which only the inner run's processes inherit. A state
// or an environment that cannot be read is an error, never a yes.
func (p *innerProc) ownedNow(v survivor) (bool, error) {
	st, pr := procInfo(v.pid)
	switch {
	case pr == unknown:
		return false, errUnknownState
	case pr == absent, st.start != v.start, st.zombie:
		return false, nil
	}
	env, err := environOf(v.pid)
	if err != nil {
		return false, err
	}
	return hasToken(env, p.tag), nil
}

// insure is the experiment's own last cleanup, after the tested path was
// judged and apart from it. It ends first the processes the tested path
// already proved this round's (known: only what its census found, or what
// sortCandidates verified; never a record alone, Review 079 B2), each while
// its start time still holds, so it does not depend on reading their
// environment again; then whatever a fresh census of this round's tag
// finds. It reports only what was still alive, and confirms the end by
// identity and by a last census. Its result never makes a check pass.
func (p *innerProc) insure(known []survivor) string {
	alive := map[string]survivor{}
	for _, v := range known {
		if st, pr := procInfo(v.pid); pr == present && st.start == v.start && !st.zombie {
			alive[v.key()] = v
		}
	}
	add := func() error {
		ss, err := censusTag(p.tag, p.start)
		for _, v := range ss {
			alive[v.key()] = v
		}
		return err
	}
	cerr := add()
	if len(alive) == 0 {
		if cerr != nil {
			return "FAILED: " + cerr.Error()
		}
		return "nothing left"
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		var pids []int
		for _, v := range alive {
			pids = append(pids, v.pid)
			if st, pr := procInfo(v.pid); pr == present && st.start == v.start && !st.zombie {
				_ = outerSignal(v.pid)
			}
		}
		sort.Ints(pids)
		still, rerr := 0, error(nil)
		for _, v := range alive {
			a, err := recordAlive(witnessRec{pid: v.pid, start: v.start})
			if err != nil {
				rerr = err
			} else if a {
				still++
			}
		}
		n := len(alive)
		cerr = add()
		switch {
		case still == 0 && rerr == nil && cerr == nil && len(alive) == n:
			return fmt.Sprintf("ended %d: %v", n, pids)
		case time.Now().After(deadline):
			return fmt.Sprintf("FAILED: %d of %v still alive (read: %v; census: %v)", still, pids, rerr, cerr)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// recordSignals makes outerSignal record, and not send, every signal to
// pid (a combination's own fixture); other processes are signalled as
// before. It returns the restore and the count.
func recordSignals(pid int) (func(), func() int) {
	prev := outerSignal
	n := 0
	outerSignal = func(p int) error {
		if p == pid {
			n++
			return nil
		}
		return prev(p)
	}
	return func() { outerSignal = prev }, func() int { return n }
}

// recordAlive reports whether the process a record names still runs: the
// same PID with the same start time, not a zombie. The identity has the
// platform's form (parseWitness), so a different start time is reliably
// another process. A state that cannot be read is an error, never an end.
func recordAlive(w witnessRec) (bool, error) {
	st, pr := procInfo(w.pid)
	switch {
	case pr == unknown:
		return false, errUnknownState
	case pr == absent, st.start != w.start, st.zombie:
		return false, nil
	}
	return true, nil
}

var errCensus = errors.New("census")
