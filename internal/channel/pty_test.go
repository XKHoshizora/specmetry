package channel_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

const markerEnv = "SPECMETRY_TEST_MARKER"

func TestMain(m *testing.M) {
	testutil.DispatchHelper(map[string]func() int{
		// detect prints the channel of the helper process itself.
		"detect": func() int {
			printDetection(channel.Detect(os.Stdin, os.Stdout))
			return 0
		},
		// detect-stdout-file is a process whose stdout goes to a file
		// while stdin stays the terminal (specmetry ... > file).
		"detect-stdout-file": func() int {
			f, err := os.CreateTemp("", "stdout")
			if err != nil {
				return 1
			}
			defer func() { _ = f.Close(); _ = os.Remove(f.Name()) }()
			printDetection(channel.Detect(os.Stdin, f))
			return 0
		},
		// detect-stdin-pipe is a process whose stdin is a pipe while stdout
		// stays the terminal (... | specmetry).
		"detect-stdin-pipe": func() int {
			r, w, err := os.Pipe()
			if err != nil {
				return 1
			}
			defer func() { _ = r.Close(); _ = w.Close() }()
			printDetection(channel.Detect(r, os.Stdout))
			return 0
		},
		// guard runs a human-only operation; it "writes" the marker file
		// only after the guard lets it proceed.
		"guard": func() int {
			err := channel.Guard(channel.Detect(os.Stdin, os.Stdout), os.Stdin, os.Stdout, op)
			if err == nil {
				if werr := os.WriteFile(os.Getenv(markerEnv), []byte("written"), 0o600); werr != nil {
					return 1
				}
			}
			fmt.Printf("\nresult: exit=%d err=%v\n", apperr.ExitCode(err), err)
			return apperr.ExitCode(err)
		},
	})
	os.Exit(m.Run())
}

func printDetection(d channel.Detection) {
	fmt.Printf("channel=%s reasons=%q\n", d.Channel, d.Reasons)
}

// helperEnv returns the environment of a helper process without the Agent
// variables (the test may itself run inside an Agent session), plus extra.
func helperEnv(t *testing.T, name string, extra ...string) []string {
	t.Helper()
	cmd := testutil.HelperCommand(t, name)
	var env []string
	for _, kv := range cmd.Env {
		key, _, _ := strings.Cut(kv, "=")
		if strings.EqualFold(key, "CLAUDECODE") || strings.EqualFold(key, "CLAUDE_CODE_ENTRYPOINT") {
			continue
		}
		env = append(env, kv)
	}
	return append(env, extra...)
}

func inPTY(t *testing.T, name string, steps []testutil.PTYStep, extra ...string) testutil.PTYResult {
	t.Helper()
	cmd := testutil.HelperCommand(t, name)
	return testutil.RunInPTYSession(t, 60*time.Second, helperEnv(t, name, extra...), steps, cmd.Path, cmd.Args[1:]...)
}

// MVP-CH-01 / MVP-CH-02 (component part): a real terminal (ConPTY on
// Windows) without Agent variables is human_cli.
func TestDetectInTerminal(t *testing.T) {
	res := inPTY(t, "detect", nil)
	if !strings.Contains(res.Output, "channel=human_cli") {
		t.Fatalf("output %q", res.Output)
	}
}

// MVP-CH-01 (component part): a verified Agent variable makes a terminal
// agent_cli; an empty one does not; the value never appears in output.
func TestDetectAgentVariablesInTerminal(t *testing.T) {
	canaryValue := testutil.NewCanary(t)
	for _, name := range channel.AgentVariables() {
		res := inPTY(t, "detect", nil, name+"="+canaryValue)
		if !strings.Contains(res.Output, "channel=agent_cli") || !strings.Contains(res.Output, name+" is set") {
			t.Errorf("%s set: output %q", name, res.Output)
		}
		if strings.Contains(res.Output, canaryValue) {
			t.Errorf("%s: the variable's value was printed", name)
		}
		if res := inPTY(t, "detect", nil, name+"="); !strings.Contains(res.Output, "channel=human_cli") {
			t.Errorf("%s empty: output %q", name, res.Output)
		}
	}
}

// MVP-CH-02 (component part): without a terminal, or with only one of
// stdin and stdout on the terminal, the channel is agent_cli.
func TestDetectWithoutTerminal(t *testing.T) {
	cmd := testutil.HelperCommand(t, "detect")
	cmd.Env = helperEnv(t, "detect")
	out, err := cmd.Output()
	if err != nil || !strings.Contains(string(out), `channel=agent_cli reasons=["stdin is not a terminal" "stdout is not a terminal"]`) {
		t.Errorf("pipes: %q, %v", out, err)
	}
	if res := inPTY(t, "detect-stdout-file", nil); !strings.Contains(res.Output, `channel=agent_cli reasons=["stdout is not a terminal"]`) {
		t.Errorf("stdout redirected: %q", res.Output)
	}
	if res := inPTY(t, "detect-stdin-pipe", nil); !strings.Contains(res.Output, `channel=agent_cli reasons=["stdin is not a terminal"]`) {
		t.Errorf("stdin piped: %q", res.Output)
	}
}

const prompt = "press Enter to cancel:"

// MVP-CH-01 (component part): in a terminal the human-only operation goes
// ahead only after the person types the confirmation text.
func TestGuardInTerminal(t *testing.T) {
	cases := []struct {
		name    string
		answer  string
		written bool
		result  string
	}{
		{"confirmed", "REQ-001\r", true, "result: exit=0"},
		{"wrong text", "yes\r", false, "result: exit=3"},
		{"cancelled with Enter", "\r", false, "result: exit=3"},
	}
	for _, c := range cases {
		marker := filepath.Join(t.TempDir(), "written")
		res := inPTY(t, "guard", []testutil.PTYStep{{Expect: prompt, Send: c.answer}}, markerEnv+"="+marker)
		_, statErr := os.Stat(marker)
		if (statErr == nil) != c.written || !strings.Contains(res.Output, c.result) {
			t.Errorf("%s: written=%v\n%s", c.name, statErr == nil, res.Output)
		}
		if !strings.Contains(res.Output, `Type "REQ-001" to confirm`) {
			t.Errorf("%s: no prompt\n%s", c.name, res.Output)
		}
	}
}

// MVP-CH-01 (component part): with an Agent variable set, even in a
// terminal, the operation is refused with exit code 3 before any prompt,
// nothing is written, and the value appears nowhere.
func TestGuardRefusesAgentInTerminal(t *testing.T) {
	canaryValue := testutil.NewCanary(t)
	for _, name := range channel.AgentVariables() {
		marker := filepath.Join(t.TempDir(), "written")
		res := inPTY(t, "guard", nil, markerEnv+"="+marker, name+"="+canaryValue)
		if _, err := os.Stat(marker); err == nil {
			t.Errorf("%s: the operation wrote", name)
		}
		if !strings.Contains(res.Output, "result: exit=3") || strings.Contains(res.Output, "to confirm") {
			t.Errorf("%s: output %q", name, res.Output)
		}
		if strings.Contains(res.Output, canaryValue) {
			t.Errorf("%s: the variable's value was printed", name)
		}
	}
}
