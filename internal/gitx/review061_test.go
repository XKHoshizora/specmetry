package gitx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// Review 061 B1-R1: an open owner handle does not prove that its name exists.
func TestOwnerRequiredBeforeCleanup(t *testing.T) {
	for _, content := range []bool{false, true} {
		for _, damage := range []string{"missing", "record", "directory", "replacement"} {
			t.Run(fmt.Sprintf("content=%v/%s", content, damage), func(t *testing.T) {
				s, home := scratchIn(t)
				d, err := newPrivateDir(s)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(d.close)
				if content {
					if err := d.write("config", []byte("preserve")); err != nil {
						t.Fatal(err)
					}
				}
				if damage != "record" {
					if err := os.Rename(d.join(ownerName), home.Join("saved-owner")); err != nil {
						if runtime.GOOS == "windows" {
							t.Skipf("OS blocked owner rename; missing-name schedule did not occur: %v", err)
						}
						t.Fatal(err)
					}
					if damage == "directory" {
						if err := os.Mkdir(d.join(ownerName), 0o700); err != nil {
							t.Fatal(err)
						}
					}
					if damage == "replacement" {
						write(t, d.join(ownerName), d.record())
					}
				} else {
					write(t, d.join(ownerName), "invalid owner record")
				}
				before := testutil.SettledSnapshot(t, home.Path)
				if err := d.checkContents(); err == nil {
					t.Error("owner anomaly passed complete preflight")
				}
				if err := d.remove(); apperr.ExitCode(err) != apperr.ExitEnvironment {
					t.Errorf("cleanup=%v", err)
				}
				testutil.AssertTreeUnchanged(t, home.Path, before)
				if _, err := newScratch(t, home); apperr.ExitCode(err) != apperr.ExitEnvironment {
					t.Errorf("leftover=%v", err)
				}
				testutil.AssertTreeUnchanged(t, home.Path, before)
			})
		}
	}
}

func TestCleanMissingOwnerPreservesInputs(t *testing.T) {
	requireSafeGit(t)
	for _, stage := range []string{"before-read", "after-read", "cleanup"} {
		t.Run(stage, func(t *testing.T) {
			s, home := scratchIn(t)
			var path string
			var before map[string]string
			moved := false
			damage := func(p string) {
				if moved {
					return
				}
				if err := os.Rename(filepath.Join(p, ownerName), home.Join("saved-owner")); err != nil {
					if runtime.GOOS == "windows" {
						t.Skipf("OS blocked owner rename; schedule not executed: %v", err)
					}
					t.Fatal(err)
				}
				moved, path = true, p
				before = testutil.SettledSnapshot(t, home.Path)
			}
			started := false
			testHookCommand = func(at string, _ int, args []string) {
				if !slices.Contains(args, "status") {
					return
				}
				if at == "started" {
					started = true
				}
				if at == "waited" && stage == "after-read" {
					for _, a := range args {
						if p, ok := strings.CutPrefix(a, "--git-dir="); ok {
							damage(p)
						}
					}
				}
			}
			if stage == "before-read" {
				testHookPrivateDir = damage
			}
			if stage == "cleanup" {
				testHookRemove = func(at, p string) error {
					if at == "contents" {
						damage(p)
					}
					return nil
				}
			}
			t.Cleanup(func() { testHookPrivateDir = nil; testHookRemove = nil; testHookCommand = nil })
			_, err := Clean(ctx, s, baseRepo(t).Dir)
			testHookPrivateDir = nil
			testHookRemove = nil
			testHookCommand = nil
			if apperr.ExitCode(err) != apperr.ExitEnvironment {
				t.Errorf("Clean=%v", err)
			}
			if !moved || path == "" {
				t.Fatal("owner schedule was not reached")
			}
			if stage == "before-read" && started {
				t.Error("Git ran after the owner was already missing")
			}
			testutil.AssertTreeUnchanged(t, home.Path, before)
		})
	}
}

// Completion is persisted only by a caller that actually waited (or could not
// start Git). A later Scratch may then reclaim this verified leftover.
func TestConfirmedConsumerCompletionAllowsReaping(t *testing.T) {
	requireSafeGit(t)
	git, err := objectGit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, end := range []string{"normal", "cancel-before-start", "start-failed"} {
		t.Run(end, func(t *testing.T) {
			s, home := scratchIn(t)
			p, err := prepare(ctx, s, baseRepo(t).Dir)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(p.dir.close)
			cctx, cancel := context.WithCancel(ctx)
			defer cancel()
			executable := git
			if end == "cancel-before-start" {
				cancel()
			}
			if end == "start-failed" {
				executable = filepath.Join(home.Path, "not-a-git-binary")
			}
			_, err = (call{label: "completion", dir: home.Path, args: []string{"--version"}, private: p.dir}).runWithGit(cctx, executable)
			if (err == nil) != (end == "normal") {
				t.Fatalf("%s: %v", end, err)
			}
			if p.dir.consumerPending || p.dir.checkContents() != nil {
				t.Fatal("confirmed completion was not persisted")
			}
			p.dir.close() // Simulate a later caller death before ordinary cleanup.
			if _, err := newScratch(t, home); err != nil {
				t.Fatal(err)
			}
			wantEmpty(t, home)
		})
	}
}

type consumerReady struct {
	Path string
	PID  int
}

type consumerConfig struct {
	Home, Repo, Tip, Kind string
}

// helperReview61Consumer owns a real Git child. Pause acknowledgement comes from
// the OS, not a sleep. Crashing here runs no defer and leaves that Git alive.
func helperReview61Consumer() int {
	input := json.NewDecoder(os.Stdin)
	var config consumerConfig
	if err := input.Decode(&config); err != nil {
		return 91
	}
	s, err := NewScratch(datadir.Dir{Path: config.Home})
	if err != nil {
		return apperr.ExitCode(err)
	}
	defer func() { _ = s.Close() }()
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mode := config.Kind
	end := ""
	testHookCommand = func(stage string, pid int, args []string) {
		if stage != "started" {
			return
		}
		if mode == "clean" && !slices.Contains(args, "status") {
			return
		}
		if mode == "probe" && !slices.Contains(args, "--format=%H") {
			return
		}
		var path string
		for _, arg := range args {
			if p, ok := strings.CutPrefix(arg, "--git-dir="); ok {
				path = p
			}
		}
		if path == "" {
			return
		}
		testHookCommand = nil
		if err := pauseConsumer(pid); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(94)
		}
		if err := json.NewEncoder(os.Stdout).Encode(consumerReady{path, pid}); err != nil {
			os.Exit(95)
		}
		if err := input.Decode(&end); err != nil {
			os.Exit(93)
		}
		if end == "crash" {
			os.Exit(99)
		}
		if end == "cancel" {
			cancel()
		} else if err := resumeConsumer(pid); err != nil {
			os.Exit(96)
		}
	}
	if mode == "clean" {
		_, err = Clean(cctx, s, config.Repo)
	} else {
		err = Log(cctx, s, config.Repo, config.Tip, nil, func(Commit) error { return nil })
	}
	if end == "cancel" && apperr.ExitCode(err) == apperr.ExitEnvironment {
		return 0
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return apperr.ExitCode(err)
	}
	if end != "release" {
		return 97
	}
	return 0
}

// Review 061 B1-R2. The reaper must preserve every byte while a real consumer
// is alive after caller death. When completion is unprovable, later exit alone
// does not authorize clearing the persistent barrier.
func TestGitConsumerLifetimeProcesses(t *testing.T) {
	requireSafeGit(t)
	for _, kind := range []string{"clean", "probe"} {
		for _, end := range []string{"release", "cancel", "crash"} {
			t.Run(kind+"/"+end, func(t *testing.T) {
				repo, ids := chain(t, 2)
				s, home := scratchIn(t)
				cmd := testutil.HelperCommand(t, "review61-consumer")
				input, err := cmd.StdinPipe()
				if err != nil {
					t.Fatal(err)
				}
				output, err := cmd.StdoutPipe()
				if err != nil {
					t.Fatal(err)
				}
				cmd.Stderr = os.Stderr
				if err = cmd.Start(); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = cmd.Process.Kill(); _ = input.Close() })
				commands := json.NewEncoder(input)
				if err := commands.Encode(consumerConfig{Home: home.Path, Repo: repo.Dir, Tip: ids[1], Kind: kind}); err != nil {
					t.Fatal(err)
				}
				ready := make(chan consumerReady, 1)
				go func() { var r consumerReady; _ = json.NewDecoder(output).Decode(&r); ready <- r }()
				var child consumerReady
				select {
				case child = <-ready:
				case <-time.After(30 * time.Second):
					t.Fatal("consumer did not acknowledge its stopped state")
				}
				if child.PID <= 0 || child.Path == "" {
					t.Fatal("helper did not start real Git")
				}
				proc, err := os.FindProcess(child.PID)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = proc.Kill(); _ = proc.Release() })
				wait, closeWatch, err := watchConsumerExit(child.PID)
				if err != nil {
					t.Fatal(err)
				}
				defer closeWatch()
				before := testutil.SettledSnapshot(t, child.Path)
				// A live caller's ordinary lease also remains effective.
				if _, err := newPrivateDir(s); apperr.ExitCode(err) != apperr.ExitEnvironment {
					t.Errorf("live lease=%v", err)
				}
				testutil.AssertTreeUnchanged(t, child.Path, before)
				if err = commands.Encode(end); err != nil {
					t.Fatal(err)
				}
				_ = input.Close()
				err = cmd.Wait()
				if end == "crash" {
					var exit *exec.ExitError
					if !errors.As(err, &exit) || exit.ExitCode() != 99 {
						t.Fatalf("caller did not crash: %v", err)
					}
					t.Logf("caller exited 99; real Git %d acknowledged paused and has not been resumed", child.PID)
					if next, err := NewScratch(home); apperr.ExitCode(err) != apperr.ExitEnvironment {
						if next != nil {
							_ = next.Close()
						}
						t.Errorf("crashed caller with live Git: %v", err)
					}
					testutil.AssertTreeUnchanged(t, child.Path, before)
					if err := resumeConsumer(child.PID); err != nil {
						t.Fatal(err)
					}
					if err := wait(); err != nil {
						t.Fatal(err)
					}
					// The test knows Git exited through a native exit notification. The
					// crashed producer did not persist that proof; production must refuse.
					if next, err := NewScratch(home); apperr.ExitCode(err) != apperr.ExitEnvironment {
						if next != nil {
							_ = next.Close()
						}
						t.Errorf("unconfirmed completion was guessed: %v", err)
					}
					testutil.AssertTreeUnchanged(t, child.Path, before)
					return
				}
				if err != nil {
					t.Fatalf("%s: %v", end, err)
				}
				if err := wait(); err != nil {
					t.Fatal(err)
				}
				wantEmpty(t, home)
				next, err := newPrivateDir(s)
				if err != nil {
					t.Fatal(err)
				}
				if err := next.remove(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
