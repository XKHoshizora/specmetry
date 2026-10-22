//go:build linux

package wp25

// T2 on Linux: the exclusive interval of spec §8.2, enforced rather than
// left to test discipline. A process-wide gate runs every child process
// (here the stand-ins for Specmetry's git calls and concurrent checks) as
// a shared lease that ends only after the child is reaped; a check takes
// the exclusive lease, finds no child at all (zombies included), sets the
// subreaper, and sweeps adopted descendants through pidfds.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

var errBaseline = errors.New("the process has children: the exclusive interval cannot start")

type interval struct{ from, to time.Time }

// procGate is the candidate enforcement: every child process goes
// through run (shared), a check through exclusive. Waiting exclusive
// requests keep new shared ones out, so a check is not starved.
type procGate struct {
	mu      sync.Mutex
	cond    *sync.Cond
	active  int
	waiting int
	excl    bool
	shared  []interval
	checks  []interval
}

func newGate2() *procGate {
	g := &procGate{}
	g.cond = sync.NewCond(&g.mu)
	return g
}

// run starts cmd under a shared lease and holds it until cmd is reaped.
func (g *procGate) run(cmd *exec.Cmd) error {
	g.mu.Lock()
	for g.excl || g.waiting > 0 {
		g.cond.Wait()
	}
	g.active++
	g.mu.Unlock()
	from := time.Now()
	err := cmd.Run()
	to := time.Now()
	g.mu.Lock()
	g.active--
	g.shared = append(g.shared, interval{from, to})
	g.cond.Broadcast()
	g.mu.Unlock()
	return err
}

// exclusive runs fn as the only source of children: no lease is held, the
// process has no child at all, and the subreaper is set for fn only.
func (g *procGate) exclusive(fn func() error) error {
	g.mu.Lock()
	g.waiting++
	for g.excl || g.active > 0 {
		g.cond.Wait()
	}
	g.waiting--
	g.excl = true
	from := time.Now()
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		g.excl = false
		g.checks = append(g.checks, interval{from, time.Now()})
		g.cond.Broadcast()
		g.mu.Unlock()
	}()
	if kids := childrenOf(os.Getpid()); len(kids) > 0 {
		return fmt.Errorf("%w: %s", errBaseline, describe(kids))
	}
	if err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); err != nil {
		return err
	}
	defer func() { _ = unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 0, 0, 0, 0) }()
	return fn()
}

// overlaps counts shared leases that overlapped a check's interval.
func (g *procGate) overlaps() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	n := 0
	for _, s := range g.shared {
		for _, c := range g.checks {
			if s.from.Before(c.to) && c.from.Before(s.to) {
				n++
			}
		}
	}
	return n
}

// describe gives PID, state and helper mode of this process's children,
// for a refusal's message.
func describe(pids []int) string {
	var out []string
	for _, p := range pids {
		st, _ := procInfo(p)
		mode := "?"
		if env, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", p)); err == nil {
			for _, kv := range strings.Split(string(env), "\x00") {
				if v, ok := strings.CutPrefix(kv, helperEnv+"="); ok {
					mode = v
				}
			}
		}
		out = append(out, fmt.Sprintf("%d zombie=%v mode=%s", p, st.zombie, mode))
	}
	return strings.Join(out, ", ")
}

func subreaper() int {
	var v int32
	_ = unix.Prctl(unix.PR_GET_CHILD_SUBREAPER, uintptr(unsafe.Pointer(&v)), 0, 0, 0)
	return int(v)
}

type sweepOpts struct {
	dryRun    bool
	maxRounds int // with limit set: rounds before giving up (F1)
	limit     bool
	open      func(pid int) (int, error)
	perturb   func(start string) string
	deadline  time.Duration
}

type sweepResult struct {
	rounds, reaped, mismatched int
	seen                       []int
	converged                  bool
	err                        error
}

// sweep ends the adopted descendants: every child of this process except
// the unreaped main process, round after round (an adopted process's
// children are adopted when it dies). Each candidate is bound by a pidfd
// and checked again (parent and start time) after the bind; it is killed
// and reaped only through its pidfd, with bounded waits.
func sweep(main int, o sweepOpts) sweepResult {
	self := os.Getpid()
	var r sweepResult
	open := o.open
	if open == nil {
		open = func(pid int) (int, error) { return unix.PidfdOpen(pid, 0) }
	}
	deadline := time.Now().Add(o.deadline)
	for {
		var cand []int
		for _, k := range childrenOf(self) {
			if k != main {
				cand = append(cand, k)
			}
		}
		if len(cand) == 0 {
			r.converged = true
			return r
		}
		if (o.limit && r.rounds >= o.maxRounds) || time.Now().After(deadline) {
			return r
		}
		r.rounds++
		progress := false
		for _, pid := range cand {
			r.seen = append(r.seen, pid)
			st, ok := procInfo(pid)
			if !ok || st.ppid != self || o.dryRun {
				continue
			}
			fd, err := open(pid)
			if err != nil {
				r.err = err // no fallback to signals by PID
				return r
			}
			st2, ok2 := procInfo(pid)
			start := st2.start
			if o.perturb != nil {
				start = o.perturb(start)
			}
			if !ok2 || st2.ppid != self || start != st.start {
				r.mismatched++
				_ = unix.Close(fd)
				continue
			}
			_ = unix.PidfdSendSignal(fd, unix.SIGKILL, nil, 0)
			if n, _ := unix.Poll([]unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}, pollMillis(deadline)); n == 1 {
				var info unix.Siginfo
				if unix.Waitid(unix.P_PIDFD, fd, &info, unix.WEXITED|unix.WNOHANG, nil) == nil {
					r.reaped++
					progress = true
				}
			}
			_ = unix.Close(fd)
		}
		if o.dryRun {
			return r
		}
		if !progress {
			time.Sleep(5 * time.Millisecond)
		}
	}
}

// checkRun is one check inside an interval: the main process in its own
// group, its exit observed without reaping, the group killed and
// confirmed, the adopted descendants swept, then the leader reaped.
type checkRun struct {
	tr       *unixTree
	reported []int
	sw       sweepResult
	conf     bool
}

func runCheck(s *scen, depth int, o sweepOpts, reapLeader bool) *checkRun {
	rep := s.newReport()
	cmd := helper("chain-main", []string{reportEnv + "=" + rep}, fmt.Sprint(depth))
	tr := startTree(s, cmd)
	cr := &checkRun{tr: tr}
	if ok, _, _ := tr.waitExit(10 * time.Second); !ok {
		s.invalidf("check main did not exit")
	}
	cr.reported = waitReport(rep, "pid", depth, 5*time.Second)
	for _, pid := range cr.reported {
		if p, err := ownPID(pid); err == nil {
			s.own(p)
		}
	}
	_ = tr.kill()
	cr.conf = tr.confirm(tKill)
	if o.deadline == 0 {
		o.deadline = tKill
	}
	cr.sw = sweep(tr.pid, o)
	if reapLeader && cr.conf && cr.sw.converged {
		tr.reap(time.Second)
	}
	return cr
}

func subset(a, b []int) bool {
	in := map[int]bool{}
	for _, x := range b {
		in[x] = true
	}
	for _, x := range a {
		if !in[x] {
			return false
		}
	}
	return true
}

func TestExclusiveGate(t *testing.T) {
	// Four goroutines start short children through the gate without
	// pause; three checks run in exclusive intervals among them.
	s := begin(t, "T2", "T2.gate.exclusive-interval", normal, true, true)
	defer s.done()
	g := newGate2()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	sharedRuns := 0
	// One of the four runs Specmetry's own kind of child, git, where git
	// is installed; the others run short helpers.
	git, _ := exec.LookPath("git")
	gitRuns := 0
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				cmd := helper("exit-after", nil, "5", "0")
				if i == 0 && git != "" {
					cmd = exec.Command(git, "--version")
				}
				if g.run(cmd) == nil {
					mu.Lock()
					sharedRuns++
					if i == 0 && git != "" {
						gitRuns++
					}
					mu.Unlock()
				}
			}
		}(i)
	}
	time.Sleep(200 * time.Millisecond)
	good := 0
	var notes []string
	for i := 0; i < 3; i++ {
		var cr *checkRun
		err := g.exclusive(func() error {
			cr = runCheck(s, 3, sweepOpts{}, true)
			return nil
		})
		if err == nil && cr != nil && cr.conf && cr.sw.converged && cr.sw.reaped == 3 && subset(cr.sw.seen, cr.reported) && cr.tr.c.reaped() {
			good++
		}
		if cr != nil {
			notes = append(notes, fmt.Sprintf("rounds=%d reaped=%d", cr.sw.rounds, cr.sw.reaped))
		}
	}
	close(stop)
	wg.Wait()
	s.launched(sharedRuns)
	s.worked(int64(sharedRuns))
	s.expect("checks with an empty baseline, every adopted process the check's, all reaped", 3, good)
	s.expect("shared leases overlapping an interval", 0, g.overlaps())
	s.expectTrue("shared children ran meanwhile", sharedRuns > 0, fmt.Sprint(sharedRuns))
	s.expectTrue("git ran through the gate", git == "" || gitRuns > 0, fmt.Sprintf("%d git runs (%s)", gitRuns, git))
	s.expect("subreaper after the intervals", 0, subreaper())
	s.expect("children left", 0, len(childrenOf(os.Getpid())))
	s.note("%v", notes)
}

func TestExclusiveConcurrentChecks(t *testing.T) {
	// Two in-process callers run checks at the same time, with shared
	// children around them: the checks' intervals never overlap, and each
	// finds an empty baseline.
	s := begin(t, "T2", "T2.gate.concurrent-checks", normal, true, true)
	defer s.done()
	g := newGate2()
	stop := make(chan struct{})
	var bg sync.WaitGroup
	bg.Add(1)
	go func() {
		defer bg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if g.run(helper("exit-after", nil, "5", "0")) == nil {
				s.launched(1)
			}
		}
	}()
	var wg sync.WaitGroup
	var mu sync.Mutex
	good, refused := 0, 0
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 2; j++ {
				var cr *checkRun
				err := g.exclusive(func() error {
					cr = runCheck(s, 2, sweepOpts{}, true)
					return nil
				})
				mu.Lock()
				switch {
				case err != nil || cr == nil:
					refused++
				case cr.conf && cr.sw.converged && cr.sw.reaped == 2 && subset(cr.sw.seen, cr.reported):
					good++
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	close(stop)
	bg.Wait()
	g.mu.Lock()
	checks := append([]interval(nil), g.checks...)
	g.mu.Unlock()
	overlap := 0
	for i := range checks {
		for j := i + 1; j < len(checks); j++ {
			if checks[i].from.Before(checks[j].to) && checks[j].from.Before(checks[i].to) {
				overlap++
			}
		}
	}
	s.worked(int64(good))
	s.expect("checks that ran clean", 4, good)
	s.expect("checks refused", 0, refused)
	s.expect("overlapping check intervals", 0, overlap)
	s.expect("shared leases overlapping a check", 0, g.overlaps())
	s.expect("children left", 0, len(childrenOf(os.Getpid())))
}

func TestExclusiveBypass(t *testing.T) {
	// Counterexample: a child started outside the gate during the interval
	// is a child of this process like an adopted one; a sweep by parent
	// alone would take it for the check's. Hence the gate must be the only
	// way to start a process (a static rule, below the runtime gate).
	s := begin(t, "T2", "T2.gate.bypass", counter, true, true)
	defer s.done()
	g := newGate2()
	var bypass *exec.Cmd
	var cr *checkRun
	_ = g.exclusive(func() error {
		bypass = helper("sleep", nil)
		startHelper(s, bypass)
		cr = runCheck(s, 1, sweepOpts{dryRun: true}, false)
		return nil
	})
	if cr == nil || bypass == nil {
		s.invalidf("interval did not run")
	}
	s.worked(int64(len(cr.sw.seen)))
	seenBypass := false
	for _, p := range cr.sw.seen {
		if p == bypass.Process.Pid {
			seenBypass = true
		}
	}
	s.expect("the bypassing child is a sweep candidate", true, seenBypass)
	s.expect("it is not among the check's processes", false, subset([]int{bypass.Process.Pid}, cr.reported))
	cr.tr.reap(time.Second)
}

func TestOtherReaper(t *testing.T) {
	// Counterexample to "the leader stays unreaped": other code in the
	// process calls wait(-1) and reaps the leader. The check before each
	// group signal (same start time, still this process's child) sees it
	// and refuses the signal; but a reaper acting between that check and
	// the signal cannot be seen. Hence the static rule: no wait(-1), no
	// reaping outside the runner.
	s := begin(t, "T2", "T2.other-reaper", counter, true, true)
	defer s.done()
	g := newGate2()
	var reapedBy int
	var held, refused bool
	_ = g.exclusive(func() error {
		cmd := helper("exit-after", nil, "50", "0")
		tr := startTree(s, cmd)
		if ok, _, _ := tr.waitExit(5 * time.Second); !ok {
			s.invalidf("main did not exit")
		}
		var ws syscall.WaitStatus
		reapedBy, _ = syscall.Wait4(-1, &ws, syscall.WNOHANG, nil) // the rogue reaper
		held = tr.leaderHeld()
		refused = errors.Is(tr.kill(), errGroupGone)
		tr.reap(time.Second) // Go's Wait now finds no child
		return nil
	})
	s.worked(1)
	s.expectTrue("the rogue wait(-1) reaped the leader", reapedBy > 0, fmt.Sprint(reapedBy))
	s.expect("leader still held", false, held)
	s.expect("group signal refused", true, refused)
	s.expect("children left", 0, len(childrenOf(os.Getpid())))
}

func TestExclusiveBaselineLive(t *testing.T) {
	s := begin(t, "T2", "T2.baseline.live-child", refusal, true, true)
	defer s.done()
	g := newGate2()
	pre := helper("sleep", nil)
	startHelper(s, pre)
	called := false
	err := g.exclusive(func() error { called = true; return nil })
	s.worked(1)
	s.expect("refused", true, errors.Is(err, errBaseline))
	s.expect("check started", false, called)
	s.expect("subreaper", 0, subreaper())
}

func TestExclusiveBaselineZombie(t *testing.T) {
	s := begin(t, "T2", "T2.baseline.zombie-child", refusal, true, true)
	defer s.done()
	g := newGate2()
	pre := helper("exit-after", nil, "10", "0")
	startHelper(s, pre)
	deadline := time.Now().Add(3 * time.Second)
	for {
		if st, ok := procInfo(pre.Process.Pid); ok && st.zombie {
			break
		}
		if time.Now().After(deadline) {
			s.invalidf("child did not become a zombie")
		}
		time.Sleep(5 * time.Millisecond)
	}
	called := false
	err := g.exclusive(func() error { called = true; return nil })
	s.worked(1)
	s.expect("refused", true, errors.Is(err, errBaseline))
	s.expect("check started", false, called)
	s.expect("subreaper", 0, subreaper())
}

func TestAdoptionChain(t *testing.T) {
	// Repeated adoption: a chain of four in a new session. Killing each
	// adopted process makes its child adopted; the sweep takes one round
	// per level and reaps each process once, through its pidfd.
	s := begin(t, "T2", "T2.adoption.chain", normal, true, true)
	defer s.done()
	g := newGate2()
	var cr *checkRun
	err := g.exclusive(func() error {
		cr = runCheck(s, 4, sweepOpts{}, true)
		return nil
	})
	if err != nil || cr == nil {
		s.invalidf("interval: %v", err)
	}
	s.worked(int64(cr.sw.reaped))
	s.expect("group confirmed", true, cr.conf)
	s.expect("converged", true, cr.sw.converged)
	s.expect("rounds", 4, cr.sw.rounds)
	s.expect("reaped through pidfds", 4, cr.sw.reaped)
	s.expect("identity mismatches", 0, cr.sw.mismatched)
	s.expectTrue("every candidate the check's", subset(cr.sw.seen, cr.reported), "")
	s.expect("leader reaped after the sweep", true, cr.tr.c.reaped())
	s.expect("children left", 0, len(childrenOf(os.Getpid())))
}

func TestSweepIdentityMismatch(t *testing.T) {
	// Fault injection: the identity read after the bind differs (as when
	// the PID was reused). The candidate is given up, never signalled, and
	// the sweep does not converge: F1.
	s := begin(t, "T2", "T2.identity.mismatch", fault, true, true)
	defer s.done()
	g := newGate2()
	var cr *checkRun
	_ = g.exclusive(func() error {
		cr = runCheck(s, 1, sweepOpts{perturb: func(st string) string { return st + "x" },
			deadline: 300 * time.Millisecond}, false)
		return nil
	})
	if cr == nil || len(cr.reported) != 1 {
		s.invalidf("interval did not run")
	}
	alive := false
	if st, ok := procInfo(cr.reported[0]); ok && !st.zombie {
		alive = true
	}
	s.worked(int64(cr.sw.mismatched))
	s.expectTrue("mismatches counted", cr.sw.mismatched >= 1, fmt.Sprint(cr.sw.mismatched))
	s.expect("reaped", 0, cr.sw.reaped)
	s.expect("converged", false, cr.sw.converged)
	s.expect("candidate left alive (not signalled)", true, alive)
	cr.tr.reap(time.Second)
}

func TestSweepPidfdUnsupported(t *testing.T) {
	// Refusal: pidfd_open is not available (injected ENOSYS). The sweep
	// stops with the error and signals nothing by PID: F1.
	s := begin(t, "T2", "T2.pidfd.unsupported", refusal, true, true)
	defer s.done()
	g := newGate2()
	var cr *checkRun
	_ = g.exclusive(func() error {
		cr = runCheck(s, 1, sweepOpts{open: func(int) (int, error) { return -1, unix.ENOSYS }}, false)
		return nil
	})
	if cr == nil || len(cr.reported) != 1 {
		s.invalidf("interval did not run")
	}
	alive := false
	if st, ok := procInfo(cr.reported[0]); ok && !st.zombie {
		alive = true
	}
	s.worked(1)
	s.expect("sweep error", "ENOSYS", errnoName(cr.sw.err))
	s.expect("converged", false, cr.sw.converged)
	s.expect("candidate left alive", true, alive)
	cr.tr.reap(time.Second)
}

func errnoName(err error) string {
	var en syscall.Errno
	if errors.As(err, &en) {
		return unix.ErrnoName(en)
	}
	return fmt.Sprint(err)
}

func TestF1Resources(t *testing.T) {
	// F1 (injected: the sweep gives up at once). The candidate rule for the
	// host's resources: pidfds closed, subreaper cleared, the leader left
	// unreaped, the lease released. The adopted process stays this
	// process's child, so the next check finds a non-empty baseline and is
	// refused: the "no reuse while residue may exist" mechanism.
	s := begin(t, "T2", "T2.F1.resources", fault, true, true)
	defer s.done()
	g := newGate2()
	var cr *checkRun
	_ = g.exclusive(func() error {
		cr = runCheck(s, 2, sweepOpts{limit: true, maxRounds: 0}, false)
		return nil
	})
	if cr == nil {
		s.invalidf("interval did not run")
	}
	kids := childrenOf(os.Getpid())
	sort.Ints(kids)
	called := false
	err := g.exclusive(func() error { called = true; return nil })
	s.worked(int64(len(kids)))
	s.expect("converged", false, cr.sw.converged)
	s.expect("subreaper cleared", 0, subreaper())
	s.expect("leader left unreaped", true, cr.tr.zombieLeader())
	s.expectTrue("adopted process still a child", subset(cr.reported[:1], kids), fmt.Sprint(kids))
	s.expect("next check refused", true, errors.Is(err, errBaseline))
	s.expect("next check started", false, called)
	s.defer_(func() { cr.tr.reap(time.Second) })
}

func TestSubreaperCleared(t *testing.T) {
	// Outside an interval the subreaper is off: an orphan goes to init or
	// another subreaper, never to this process.
	s := begin(t, "T2", "T2.subreaper.cleared", normal, true, true)
	defer s.done()
	rep := s.newReport()
	cmd := helper("chain-main", []string{reportEnv + "=" + rep}, "1")
	startHelper(s, cmd)
	childOf(cmd).reap(10 * time.Second)
	pids := waitReport(rep, "pid", 1, 5*time.Second)
	if len(pids) != 1 {
		s.invalidf("orphan did not report")
	}
	p, err := ownPID(pids[0])
	if err != nil {
		s.invalidf("orphan: %v", err)
	}
	s.own(p)
	time.Sleep(50 * time.Millisecond)
	st, _ := procInfo(pids[0])
	s.worked(1)
	s.expect("subreaper", 0, subreaper())
	s.expectTrue("orphan's parent is not this process", st.ppid != os.Getpid(), fmt.Sprintf("ppid %d", st.ppid))
}
