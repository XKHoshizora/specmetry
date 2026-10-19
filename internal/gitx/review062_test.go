package gitx

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// recordFault fires once at the record I/O boundary: before Start for the begin
// control, after actual Git exit for completion. It leaves the inputs intact.
type recordFault struct {
	d                      *privateDir
	stage, command         string
	begin, fired           bool
	started, waited, syncs int
	ioAfterFailure         int
	cmd                    *exec.Cmd
	before                 map[string]string
}

func (f *recordFault) install(t *testing.T) {
	t.Helper()
	testHookCommandSetup = func(cmd *exec.Cmd) {
		if slices.Contains(cmd.Args, f.command) {
			f.cmd = cmd
		}
	}
	testHookCommand = func(at string, pid int, args []string) {
		if !slices.Contains(args, f.command) {
			return
		}
		if f.cmd == nil || f.cmd.Process.Pid != pid {
			t.Fatal("did not observe the actual Git process")
		}
		switch at {
		case "started":
			f.started++
			if !f.d.consumerPending || f.d.checkContents() != nil {
				t.Fatal("Git started without complete pending inputs")
			}
		case "waited":
			if f.cmd.ProcessState == nil || !f.cmd.ProcessState.Exited() || !f.cmd.ProcessState.Success() {
				t.Fatal("real Git did not exit successfully before completion persistence")
			}
			f.waited++
			if f.d.checkContents() != nil {
				t.Fatal("complete inputs failed preflight before fault injection")
			}
		}
	}
	testHookRecordIO = func(stage string, d *privateDir) error {
		f.d = d
		if f.fired {
			f.ioAfterFailure++
			return nil
		}
		if (f.begin && !d.consumerPending) || (!f.begin && (d.consumerPending || f.waited == 0)) {
			return nil
		}
		if stage == "sync" {
			f.syncs++
		}
		if stage != f.stage {
			return nil
		}
		if f.begin {
			if f.started != 0 || f.waited != 0 {
				t.Fatal("begin fault happened after a consumer started")
			}
		} else if f.started != 1 || f.waited != 1 {
			t.Fatal("completion fault did not follow exactly one real Start/Wait")
		}
		if stage == "sync" {
			b, err := os.ReadFile(d.join(ownerName))
			var r ownerRecord
			if err != nil || json.Unmarshal(b, &r) != nil || r.ConsumerPending != f.begin || string(b) != d.record() || d.checkContents() != nil {
				t.Fatal("Sync fault did not follow a complete, valid record write")
			}
		}
		f.before = testutil.SettledSnapshot(t, d.path)
		f.fired = true
		t.Logf("inject %s: actual Git starts=%d waits=%d; record pending=%v", stage, f.started, f.waited, d.consumerPending)
		return errors.New("injected owner record I/O failure")
	}
	t.Cleanup(func() { testHookCommand = nil; testHookCommandSetup = nil; testHookRecordIO = nil })
}

func requireRecordEnvironment(t *testing.T, err error) {
	t.Helper()
	if apperr.ExitCode(err) != apperr.ExitEnvironment {
		t.Fatalf("got %v; want environment error", err)
	}
}

func checkRecordReleaseAndReap(t *testing.T, f *recordFault, s *Scratch, home datadir.Dir, reaps bool) {
	t.Helper()
	testutil.AssertTreeUnchanged(t, f.d.path, f.before)
	if f.ioAfterFailure != 0 {
		t.Fatal("failure cleanup/reuse attempted another record write or Sync")
	}
	if f.d.root != nil || f.d.lock != nil || f.d.lease != nil || f.d.dir != nil || f.d.createHandle != nil {
		t.Fatal("failure cleanup did not release its own resources")
	}
	lease, err := s.acquire()
	if err != nil {
		t.Fatalf("failure leaked the namespace lease: %v", err)
	}
	releaseLease(lease)
	// A new process has no memory of the earlier Sync return. v3 can adopt a
	// complete false record after full identity/manifest checks; pending or
	// incomplete records must still be retained. Never call this an old Sync ack.
	cmd := testutil.HelperCommand(t, "scratch-lease", "SCRATCH_HOME="+home.Path)
	cmd.Stdin = strings.NewReader("release\n")
	out, err := cmd.CombinedOutput()
	if reaps {
		if err != nil {
			t.Fatalf("later complete-record reap: %v %s", err, out)
		}
		wantEmpty(t, home)
		return
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != apperr.ExitEnvironment {
		t.Fatalf("later uncertain record: %v %s", err, out)
	}
	testutil.AssertTreeUnchanged(t, f.d.path, f.before)
}

func TestConsumerRecordFailureRetainsInputs(t *testing.T) {
	requireSafeGit(t)
	for _, phase := range []string{"begin-sync", "finish-sync", "finish-write", "finish-truncate"} {
		t.Run(phase, func(t *testing.T) {
			s, home := scratchIn(t)
			p, err := prepare(ctx, s, baseRepo(t).Dir)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(p.dir.close)
			if p.dir.checkContents() != nil || len(p.dir.entries) == 0 {
				t.Fatal("fixture is not a complete manifested repository")
			}
			begin, stage, _ := strings.Cut(phase, "-")
			f := &recordFault{d: p.dir, stage: stage, begin: begin == "begin", command: "status"}
			f.install(t)
			_, _, err = p.read(ctx, true)
			requireRecordEnvironment(t, err)
			if !f.fired || (stage == "sync" && f.syncs != 1) {
				t.Fatal("did not inject the intended single Sync/write failure")
			}
			if err := p.close(); apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Errorf("cleanup after failed persistence: %v; want environment error and retention", err)
			}
			checkRecordReleaseAndReap(t, f, s, home, phase == "finish-sync")
		})
	}
}

func TestPublicCompletionSyncFailure(t *testing.T) {
	requireSafeGit(t)
	if err := CheckObjectReads(ctx); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"clean", "probe"} {
		t.Run(mode, func(t *testing.T) {
			repo, ids := chain(t, 2)
			s, home := scratchIn(t)
			command := "status"
			if mode == "probe" {
				stateMu.Lock()
				state.scanEmpty = false
				stateMu.Unlock()
				command = "--format=%H"
			}
			f := &recordFault{stage: "sync", command: command}
			f.install(t)
			var err error
			if mode == "clean" {
				_, err = Clean(ctx, s, repo.Dir)
			} else {
				err = Log(ctx, s, repo.Dir, ids[1], nil, func(Commit) error { t.Error("published a result after probe failure"); return nil })
			}
			requireRecordEnvironment(t, err)
			if !f.fired || f.syncs != 1 || f.started != 1 || f.waited != 1 {
				t.Fatal("public call did not fail the completion Sync after real Git exit")
			}
			if mode == "probe" {
				stateMu.Lock()
				cached := state.scanEmpty
				stateMu.Unlock()
				if cached {
					t.Error("failed completion was cached as a supported probe")
				}
			}
			checkRecordReleaseAndReap(t, f, s, home, true)
		})
	}
}

func TestCompletionSyncFailureCannotRetry(t *testing.T) {
	requireSafeGit(t)
	s, home := scratchIn(t)
	p, err := prepare(ctx, s, baseRepo(t).Dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.dir.close)
	f := &recordFault{d: p.dir, stage: "sync", command: "status"}
	f.install(t)
	_, _, err = p.read(ctx, true)
	requireRecordEnvironment(t, err)
	if !f.fired || f.waited != 1 {
		t.Fatal("no completion failure")
	}
	// Do not let a later finish or begin silently supersede the known failure.
	requireRecordEnvironment(t, p.dir.finishConsumer())
	requireRecordEnvironment(t, p.dir.beginConsumer())
	testutil.AssertTreeUnchanged(t, p.dir.path, f.before)
	requireRecordEnvironment(t, p.close())
	checkRecordReleaseAndReap(t, f, s, home, true)
}
