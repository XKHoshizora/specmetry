package testutil

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// CI run 39121984050 (Ubuntu, git 2.55.0): right after a fixture commit,
// SettledSnapshot listed .git/objects/maintenance.lock and the file was
// gone when it read it. That is consistent with a race against git's
// automatic maintenance: a commit can start `git maintenance run --auto`,
// which takes that lock, and gits with maintenance.autoDetach may let it
// outlive the commit. Whether it did then was not observed, and the race
// was not reproduced. GitRepo's isolated configuration turns automatic
// maintenance and gc off. These tests show that a fixture commit starts
// neither, with a trace2 control showing that the check would see one.

// childProcesses runs git in r with a trace2 event target and returns the
// argv of every child process that git, or a child of it, started.
func childProcesses(t *testing.T, r *GitRepo, args ...string) [][]string {
	t.Helper()
	trace := filepath.Join(t.TempDir(), "trace2.json")
	r.gitEnvIn(r.Dir, []string{"GIT_TRACE2_EVENT=" + trace}, args...)
	f, err := os.Open(trace)
	if err != nil {
		t.Fatalf("git wrote no trace2 events: %v", err)
	}
	defer func() { _ = f.Close() }()
	var children [][]string
	events := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var e struct {
			Event string   `json:"event"`
			Argv  []string `json:"argv"`
		}
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("a trace2 event is not JSON: %q", sc.Text())
		}
		events++
		if e.Event == "child_start" {
			children = append(children, e.Argv)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if events == 0 {
		t.Fatal("the trace2 target is empty")
	}
	return children
}

// maintenanceChildren are the children that run automatic maintenance or gc.
func maintenanceChildren(children [][]string) [][]string {
	var out [][]string
	for _, argv := range children {
		if slices.Contains(argv, "maintenance") || slices.Contains(argv, "gc") {
			out = append(out, argv)
		}
	}
	return out
}

func TestFixtureCommitsStartNoAutomaticMaintenance(t *testing.T) {
	commit := func(t *testing.T, r *GitRepo, config ...string) [][]string {
		t.Helper()
		r.WriteFile("README.md", t.Name()+"\n")
		r.Git("add", "-A")
		return childProcesses(t, r, append(config, "commit", "-q", "-m", "a commit")...)
	}

	// The control: with automatic maintenance turned back on for this one
	// command (in the foreground, so that it ends with the commit), the
	// trace shows git starting it.
	t.Run("the trace sees automatic maintenance when it is on", func(t *testing.T) {
		r := NewGitRepo(t)
		got := maintenanceChildren(commit(t, r, "-c", "maintenance.auto=true",
			"-c", "maintenance.autoDetach=false", "-c", "gc.autoDetach=false"))
		if len(got) == 0 {
			t.Fatal("no maintenance child in the trace: the check cannot see one")
		}
		t.Logf("git started %q", got)
	})

	t.Run("a fixture commit starts none", func(t *testing.T) {
		r := NewGitRepo(t)
		if got := maintenanceChildren(commit(t, r)); len(got) > 0 {
			t.Fatalf("a fixture commit started %q", got)
		}
		for key, want := range map[string]string{"maintenance.auto": "false", "gc.auto": "0"} {
			if got := r.Git("config", "--get", key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
	})

	// Inherited variables are dropped (codex-review-012 B1), so they cannot
	// turn it back on either.
	for name, env := range map[string]map[string]string{
		"GIT_CONFIG_COUNT": {"GIT_CONFIG_COUNT": "2", "GIT_CONFIG_KEY_0": "maintenance.auto", "GIT_CONFIG_VALUE_0": "true",
			"GIT_CONFIG_KEY_1": "gc.auto", "GIT_CONFIG_VALUE_1": "1"},
		"GIT_CONFIG_PARAMETERS": {"GIT_CONFIG_PARAMETERS": "'maintenance.auto'='true' 'gc.auto'='1'"},
	} {
		t.Run("an inherited "+name+" cannot turn it on", func(t *testing.T) {
			for k, v := range env {
				t.Setenv(k, v)
			}
			r := NewGitRepo(t)
			if got := maintenanceChildren(commit(t, r)); len(got) > 0 {
				t.Fatalf("a fixture commit started %q", got)
			}
		})
	}
}

// The fixture's global configuration holds nothing but these two settings.
func TestFixtureGitConfigIsOnlyMaintenance(t *testing.T) {
	r := NewGitRepo(t)
	got := r.Git("config", "--global", "--list")
	want := "maintenance.auto=false\ngc.auto=0"
	if strings.ReplaceAll(got, "\r\n", "\n") != want {
		t.Fatalf("global configuration %q, want %q", got, want)
	}
}
