package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// plainGit runs git as a user would, with the test's environment: it shows
// that an injection is live before init is shown to ignore it.
func plainGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	_ = cmd.Run() // the probes make git fail or complain; only their effects matter
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// probeScript writes a shell script that records that it ran in a file
// outside the repository and creates a branch, the effects
// codex-review-015 B1 observed. It returns the command that runs it and
// the file it writes. (git runs hook and fsmonitor commands with sh, Git
// for Windows included.)
func probeScript(t *testing.T) (command, ran string) {
	t.Helper()
	dir := t.TempDir()
	ran = filepath.Join(dir, "probe-ran")
	script := filepath.Join(dir, "probe.sh")
	body := "#!/bin/sh\necho ran >> '" + filepath.ToSlash(ran) + "'\ngit update-ref refs/heads/review-polluted HEAD\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(script, " '") {
		t.Skipf("temporary path %q needs quoting", script)
	}
	return "sh " + filepath.ToSlash(script), ran
}

// gitState is everything init must leave alone in the repository's git
// directory: every file with its content, and the refs.
func gitState(t *testing.T, repo *testutil.GitRepo, settled bool) (map[string]string, string) {
	t.Helper()
	dir := filepath.Join(repo.Dir, ".git")
	snapshot := testutil.SnapshotTree
	if settled {
		snapshot = testutil.SettledSnapshot
	}
	return fileEntries(snapshot(t, dir)), repo.Git("for-each-ref")
}

// codex-review-015 B1: the git commands init runs read only. Configuration
// from the environment, the user's global configuration or the repository
// cannot make them run a program, write a file outside the repository,
// create a ref, or write a trace — through the public init path.
func TestInitGitRunsNothingElse(t *testing.T) {
	listFiles := []string{"ls-files", "-z", "--cached", "--others", "--exclude-standard"}
	cases := []struct {
		name string
		// inject sets the injection up and returns the files it makes git
		// write and the git command that shows it is live.
		inject func(t *testing.T, repo *testutil.GitRepo) (effects []string, control []string)
	}{
		{"fsmonitor in the repository configuration", func(t *testing.T, repo *testutil.GitRepo) ([]string, []string) {
			command, ran := probeScript(t)
			repo.Git("config", "core.fsmonitor", command)
			return []string{ran}, listFiles
		}},
		{"fsmonitor through GIT_CONFIG_COUNT", func(t *testing.T, _ *testutil.GitRepo) ([]string, []string) {
			command, ran := probeScript(t)
			t.Setenv("GIT_CONFIG_COUNT", "1")
			t.Setenv("GIT_CONFIG_KEY_0", "core.fsmonitor")
			t.Setenv("GIT_CONFIG_VALUE_0", command)
			return []string{ran}, listFiles
		}},
		{"fsmonitor through GIT_CONFIG_PARAMETERS", func(t *testing.T, _ *testutil.GitRepo) ([]string, []string) {
			command, ran := probeScript(t)
			t.Setenv("GIT_CONFIG_PARAMETERS", "'core.fsmonitor'='"+command+"'")
			return []string{ran}, listFiles
		}},
		{"fsmonitor and trace2 in the global configuration", func(t *testing.T, _ *testutil.GitRepo) ([]string, []string) {
			command, ran := probeScript(t)
			home := t.TempDir()
			trace := filepath.Join(t.TempDir(), "trace2-global")
			config := "[core]\n\tfsmonitor = " + command + "\n[trace2]\n\tnormalTarget = " + filepath.ToSlash(trace) + "\n"
			if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(config), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", home)
			return []string{ran, trace}, listFiles
		}},
		{"global configuration named by GIT_CONFIG_GLOBAL", func(t *testing.T, _ *testutil.GitRepo) ([]string, []string) {
			command, ran := probeScript(t)
			config := filepath.Join(t.TempDir(), "gitconfig")
			if err := os.WriteFile(config, []byte("[core]\n\tfsmonitor = "+command+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GIT_CONFIG_GLOBAL", config)
			return []string{ran}, listFiles
		}},
		{"trace variables", func(t *testing.T, _ *testutil.GitRepo) ([]string, []string) {
			dir := t.TempDir()
			var effects []string
			for _, name := range []string{"GIT_TRACE", "GIT_TRACE2", "GIT_TRACE2_EVENT", "GIT_TRACE_SETUP", "GIT_TRACE_PERFORMANCE"} {
				p := filepath.Join(dir, strings.ToLower(name))
				t.Setenv(name, p)
				effects = append(effects, p)
			}
			return effects, listFiles
		}},
		{"hooks of the repository", func(t *testing.T, repo *testutil.GitRepo) ([]string, []string) {
			hooks := t.TempDir()
			ran := filepath.Join(t.TempDir(), "hook-ran")
			for _, hook := range []string{"reference-transaction", "post-index-change", "post-checkout", "pre-commit"} {
				body := "#!/bin/sh\necho " + hook + " >> '" + filepath.ToSlash(ran) + "'\n"
				if err := os.WriteFile(filepath.Join(hooks, hook), []byte(body), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			repo.Git("config", "core.hooksPath", filepath.ToSlash(hooks))
			// A ref update runs reference-transaction: the hooks are live.
			return []string{ran}, []string{"update-ref", "refs/heads/hook-probe", "HEAD"}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			testutil.TempHome(t)
			repo := committedRepo(t)
			effects, control := c.inject(t, repo)

			plainGit(t, repo.Dir, control...)
			live := false
			for _, e := range effects {
				live = live || exists(e)
			}
			if !live {
				t.Fatalf("the injection has no effect on plain git, so this test would prove nothing")
			}
			for _, e := range effects {
				_ = os.Remove(e)
			}
			for _, ref := range []string{"refs/heads/review-polluted", "refs/heads/hook-probe"} {
				plainGit(t, repo.Dir, "update-ref", "-d", ref)
			}
			for _, e := range effects {
				_ = os.Remove(e)
			}
			files, refs := gitState(t, repo, true)

			code, out, text := initIn(t, repo.Dir, "--no-exclude")
			if code != apperr.ExitOK || !out.Created {
				t.Fatalf("init: exit %d\n%s", code, text)
			}
			for _, e := range effects {
				if exists(e) {
					t.Errorf("init made git write %s", e)
				}
			}
			afterFiles, afterRefs := gitState(t, repo, false)
			if afterRefs != refs {
				t.Errorf("refs changed:\n%s\n->\n%s", refs, afterRefs)
			}
			if d := testutil.DiffSnapshots(files, afterFiles); d != "" {
				t.Errorf("the git directory changed:\n%s", d)
			}
		})
	}
}

// codex-review-015 B2: the main worktree's branch becomes the default
// target's ref, so it is scanned for secrets before anything is written.
func TestInitRefusesSecretInDefaultTargetRef(t *testing.T) {
	home := testutil.TempHome(t)
	repo := committedRepo(t)
	secret := "ghp_" + strings.Repeat("A1b2", 9)
	repo.Git("checkout", "-q", "-b", "feature/"+secret)
	t.Chdir(repo.Dir)
	code, stdout, stderr := execute(t, "init", "--json", "--name", "demo")
	if code != apperr.ExitInvalidInput {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	conforms(t, code, stdout)
	var env apperr.Envelope
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Error.Details) != 1 || env.Error.Details[0].Problem != "secret_detected: github_token" ||
		!strings.Contains(env.Error.Details[0].Pointer, "target") {
		t.Fatalf("details = %+v", env.Error.Details)
	}
	if strings.Contains(stdout+stderr, secret) {
		t.Fatal("the secret was echoed")
	}
	if code, out, err := execute(t, "init", "--name", "demo"); code != apperr.ExitInvalidInput || strings.Contains(out+err, secret) {
		t.Fatalf("without --json: exit %d\n%s%s", code, out, err)
	}
	if exists(home.Path) {
		t.Fatal("the data directory was created")
	}
	if exists(filepath.Join(repo.Dir, ".specmetry")) {
		t.Fatal("a marker was written")
	}
}

// codex-review-015 B3: git's own error messages, which quote configuration
// values, reach neither the output nor the data directory — whether the
// value comes from the environment or from the repository's
// configuration.
func TestInitDoesNotEchoGit(t *testing.T) {
	t.Run("invalid value in the environment", func(t *testing.T) {
		home := testutil.TempHome(t)
		repo := committedRepo(t)
		canary := testutil.NewCanary(t)
		t.Setenv("GIT_CONFIG_COUNT", "1")
		t.Setenv("GIT_CONFIG_KEY_0", "core.bare")
		t.Setenv("GIT_CONFIG_VALUE_0", canary)
		code, _, text := initIn(t, repo.Dir)
		// The variable never reaches git (B1), so init succeeds.
		if code != apperr.ExitOK {
			t.Fatalf("exit %d\n%s", code, text)
		}
		if strings.Contains(text, canary) {
			t.Fatalf("the canary was echoed:\n%s", text)
		}
		if hits := testutil.FindInTree(t, home.Path, canary); len(hits) != 0 {
			t.Fatalf("the canary was stored: %v", hits)
		}
		if b, _ := os.ReadFile(filepath.Join(repo.Dir, ".specmetry", "project.json")); strings.Contains(string(b), canary) {
			t.Fatal("the canary is in the marker")
		}
	})
	t.Run("invalid value in the repository configuration", func(t *testing.T) {
		home := testutil.TempHome(t)
		repo := committedRepo(t)
		canary := testutil.NewCanary(t)
		repo.Git("config", "core.bare", canary)
		for _, args := range [][]string{{"--json"}, {}} {
			t.Chdir(repo.Dir)
			code, stdout, stderr := execute(t, append([]string{"init"}, args...)...)
			if code != apperr.ExitEnvironment {
				t.Fatalf("init %v: exit %d\n%s%s", args, code, stdout, stderr)
			}
			if len(args) > 0 {
				conforms(t, code, stdout)
			}
			if strings.Contains(stdout+stderr, canary) || strings.Contains(stdout+stderr, repo.Dir) {
				t.Fatalf("init %v echoed git:\n%s%s", args, stdout, stderr)
			}
		}
		if exists(home.Path) {
			t.Fatal("the data directory was created")
		}
		if exists(filepath.Join(repo.Dir, ".specmetry")) {
			t.Fatal("a marker was written")
		}
	})
}

// codex-review-015 N1: a marker naming a project, with no database in this
// data directory, is refused before the database is created.
func TestInitRefusesOrphanMarkerBeforeCreatingDatabase(t *testing.T) {
	const other = "01J9Z3K5Q8R2T4V6W8X0Y2Z4C9"
	marker := `{"schema":"specmetry.project/v1","project_id":"` + other + `","name":"elsewhere"}` + "\n"
	setup := func(t *testing.T) *testutil.GitRepo {
		t.Helper()
		repo := committedRepo(t)
		if err := os.MkdirAll(filepath.Join(repo.Dir, ".specmetry"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo.Dir, ".specmetry", "project.json"), []byte(marker), 0o644); err != nil {
			t.Fatal(err)
		}
		return repo
	}
	check := func(t *testing.T, repo *testutil.GitRepo) {
		t.Helper()
		code, _, text := initIn(t, repo.Dir)
		if code != apperr.ExitEnvironment || !strings.Contains(text, other) || !strings.Contains(text, "No project was registered") {
			t.Fatalf("exit %d\n%s", code, text)
		}
		if b, _ := os.ReadFile(filepath.Join(repo.Dir, ".specmetry", "project.json")); string(b) != marker {
			t.Fatalf("marker changed: %q", b)
		}
	}
	t.Run("no data directory", func(t *testing.T) {
		home := testutil.TempHome(t)
		check(t, setup(t))
		if exists(home.Path) {
			t.Fatal("the data directory was created")
		}
	})
	t.Run("a data directory without a database", func(t *testing.T) {
		home := testutil.TempHome(t)
		if err := home.Ensure(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(home.Join(datadir.ConfigFile), []byte("[user]\ndisplay_name = \"me\"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		repo := setup(t)
		before := testutil.SettledSnapshot(t, home.Path)
		check(t, repo)
		testutil.AssertTreeUnchanged(t, home.Path, before)
	})
}

// codex-review-015 N2: a cancelled init stops, and registers nothing.
func TestInitStopsWhenCanceled(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	plain := t.TempDir()
	for i := range 20 {
		if err := os.WriteFile(filepath.Join(plain, "doc"+string(rune('a'+i))+".md"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for name, dir := range map[string]string{"outside git": plain, "git": committedRepo(t).Dir} {
		t.Run(name, func(t *testing.T) {
			home := testutil.TempHome(t)
			t.Chdir(dir)
			var stdout, stderr strings.Builder
			code := Execute(canceled, []string{"init", "--json"}, &stdout, &stderr)
			if code == apperr.ExitOK {
				t.Fatalf("a cancelled init succeeded:\n%s", stdout.String())
			}
			conforms(t, code, stdout.String())
			if exists(home.Path) {
				t.Fatal("the data directory was created")
			}
			if exists(filepath.Join(dir, ".specmetry")) {
				t.Fatal("a marker was written")
			}
		})
	}
}

// The refusals above leave the registry as it was.
func TestInitRefusalsRegisterNothing(t *testing.T) {
	home := testutil.TempHome(t)
	repo := committedRepo(t)
	first := mustInit(t, repo.Dir)
	seq := rows(t, home, `SELECT seq_counter FROM meta`)
	secret := "ghp_" + strings.Repeat("A1b2", 9)
	wt := repo.AddWorktree(secret)
	if code, _, text := initIn(t, wt); code != apperr.ExitInvalidInput || strings.Contains(text, secret) {
		t.Fatalf("exit %d\n%s", code, text)
	}
	if after := rows(t, home, `SELECT seq_counter FROM meta`); !reflect.DeepEqual(after, seq) {
		t.Fatalf("seq %v -> %v", seq, after)
	}
	if got := rows(t, home, `SELECT count(*) FROM worktrees WHERE project_id = ?`, first.Project.ID); !reflect.DeepEqual(got, [][]string{{"1"}}) {
		t.Fatalf("worktrees = %v", got)
	}
}
