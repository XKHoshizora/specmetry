package wp25

// T5: does a check's command text reach each interpreter unchanged? The
// 23 strings of the author's experiment X5; each is wrapped in a quoted
// here-document (or a PowerShell here-string) that prints it back, so it
// comes back intact only if the whole command line did. The interpreters,
// versions and layouts found are recorded; none is installed.

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

var payloads = []string{
	"plain", "with space", `quote"inside`, `single'q`, `back\slash`, `trail\`, `two\\`,
	`a\"b`, `*.go`, `{a,b}`, `[ab]`, `~/x`, `$HOME`, `%PATH%`, `!bang`, `^caret`,
	`a&b|c<d>e`, "tab\there", `中文 路径`, `\\server\share\`, `"`, `""`, `\"`,
}

// shScript prints the payload back with shell builtins only, so no tool
// on PATH is needed.
func shScript(p string) string {
	return "while IFS= read -r l; do printf '%s\\n' \"$l\"; done <<'WP25EOF'\n" + p + "\nWP25EOF"
}

// psScript prints the payload's UTF-8 bytes in hex.
func psScript(p string) string {
	return "$s=@'\n" + p + "\n'@; [Console]::Out.Write([BitConverter]::ToString([Text.Encoding]::UTF8.GetBytes($s)))"
}

func same(p string) string { return p }

func hexOf(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if i > 0 {
			b.WriteByte('-')
		}
		fmt.Fprintf(&b, "%02X", s[i])
	}
	return b.String()
}

type adapter struct {
	name    string
	cmd     func(p string) *exec.Cmd // nil: interpreter not found
	want    func(p string) string
	version string
	k       kind
	changed []string // payloads expected to change (a counterexample)
}

func runAdapter(t *testing.T, a adapter) {
	s := begin(t, "T5", "T5.adapter."+a.name, a.k, true, true)
	defer s.done()
	if a.cmd == nil {
		s.unsupported("interpreter not found")
	}
	run := func(p string) (string, error) {
		c := a.cmd(p)
		var out strings.Builder
		c.Stdout = &out
		if err := c.Start(); err != nil {
			return "", err
		}
		s.launched(1)
		s.own(ownChild(c))
		if !childOf(c).reap(30 * time.Second) {
			return "", fmt.Errorf("no exit within 30 s")
		}
		if code := c.ProcessState.ExitCode(); code != 0 {
			return out.String(), fmt.Errorf("exit %d", code)
		}
		return strings.TrimRight(out.String(), "\r\n"), nil
	}
	// A broken tool is not a finding about quoting: the plain string must
	// come back first.
	if got, err := run("plain"); err != nil || got != a.want("plain") {
		s.invalidf("sanity run failed: %v %q", err, got)
	}
	var changed []string
	for _, p := range payloads {
		got, err := run(p)
		if err != nil || got != a.want(p) {
			changed = append(changed, p)
		}
		s.worked(1)
	}
	s.note("version %s", a.version)
	s.expect("changed payloads", fmt.Sprintf("%q", a.changed), fmt.Sprintf("%q", changed))
}

// output runs a command for a version string; errors give "".
func output(name string, args ...string) string {
	c := exec.Command(name, args...)
	b, err := c.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0])
}
