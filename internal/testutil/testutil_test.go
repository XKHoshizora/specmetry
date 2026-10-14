package testutil

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/term"
)

func TestMain(m *testing.M) {
	DispatchHelper(map[string]func() int{
		"istty": func() int {
			fmt.Printf("stdin=%v stdout=%v\n", term.IsTerminal(int(os.Stdin.Fd())), term.IsTerminal(int(os.Stdout.Fd())))
			return 0
		},
		"exit7": func() int { return 7 },
		"readline": func() int {
			fmt.Print("ready> ")
			line, err := bufio.NewReader(os.Stdin).ReadString('\n')
			fmt.Printf("\ngot %q %v\n", strings.TrimSpace(line), err)
			return 0
		},
	})
	os.Exit(m.Run())
}

func TestTempHomeIsIsolated(t *testing.T) {
	d := TempHome(t)
	if got := os.Getenv("SPECMETRY_HOME"); got != d.Path {
		t.Fatalf("SPECMETRY_HOME = %q, want %q", got, d.Path)
	}
	if ok, _ := d.Exists(); ok {
		t.Fatal("TempHome must not create the directory")
	}
}

func TestGitRepoCommitsAndWorktrees(t *testing.T) {
	r := NewGitRepo(t)
	r.WriteFile("README.md", "hello\n")
	a := r.Commit("first")
	if len(a) != 40 {
		t.Fatalf("commit sha %q", a)
	}
	wt := r.AddWorktree("feature")
	if got := r.GitIn(wt, "rev-parse", "HEAD"); got != a {
		t.Fatalf("worktree HEAD %s, want %s", got, a)
	}
	common := r.GitIn(wt, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if filepath.Clean(common) != filepath.Clean(filepath.Join(r.Dir, ".git")) {
		t.Fatalf("worktree common dir %s", common)
	}
	if got := r.Git("config", "--get", "core.autocrlf"); got != "false" {
		t.Fatalf("core.autocrlf = %q", got)
	}
}

func TestCanarySearch(t *testing.T) {
	dir := t.TempDir()
	c1, c2 := NewCanary(t), NewCanary(t)
	if c1 == c2 {
		t.Fatal("canaries must be unique")
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "blob.bin"), []byte("\x00\x01"+c1+"\xff"), 0o600); err != nil {
		t.Fatal(err)
	}
	hits := FindInTree(t, dir, c1, c2)
	if len(hits) != 1 || !strings.Contains(hits[0], c1) {
		t.Fatalf("hits = %v", hits)
	}
}

func TestHelperProcessExitCode(t *testing.T) {
	err := HelperCommand(t, "exit7").Run()
	if err == nil || !strings.Contains(err.Error(), "7") {
		t.Fatalf("helper exit: %v", err)
	}
}

// RunInPTYSession types into the terminal once the program has printed what
// a step waits for, as a person answering a prompt would.
func TestRunInPTYSessionAnswersAPrompt(t *testing.T) {
	cmd := HelperCommand(t, "readline")
	res := RunInPTYSession(t, 30*time.Second, cmd.Env, []PTYStep{{Expect: "ready>", Send: "hello\r"}}, cmd.Path, cmd.Args[1:]...)
	if res.WaitErr != nil {
		t.Fatalf("helper failed: %v\n%s", res.WaitErr, res.Output)
	}
	if !strings.Contains(res.Output, `got "hello" <nil>`) {
		t.Fatalf("pty output = %q", res.Output)
	}
}

func TestRunInPTYProvidesATerminal(t *testing.T) {
	cmd := HelperCommand(t, "istty")
	res := RunInPTY(t, 30*time.Second, cmd.Env, cmd.Path, cmd.Args[1:]...)
	if res.WaitErr != nil {
		t.Fatalf("helper failed: %v\n%s", res.WaitErr, res.Output)
	}
	if !strings.Contains(res.Output, "stdin=true stdout=true") {
		t.Fatalf("pty output = %q", res.Output)
	}
	// Without a pty the same helper must not see a terminal.
	out, err := HelperCommand(t, "istty").Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "stdin=false stdout=false") {
		t.Fatalf("non-pty output = %q", out)
	}
}

func TestGoldenComparesNormalizedOutput(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Setenv(UpdateGoldenEnv, "1")
	Golden(t, "sample", []byte("a\r\nb\n"))
	t.Setenv(UpdateGoldenEnv, "")
	Golden(t, "sample", []byte("a\nb\n"))
}
