//go:build windows

package wp25

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// q quotes one argument by the MSVCRT rules, always.
func q(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			n++
			continue
		case '"':
			b.WriteString(strings.Repeat(`\`, 2*n+1))
		default:
			b.WriteString(strings.Repeat(`\`, n))
		}
		n = 0
		b.WriteByte(c)
	}
	b.WriteString(strings.Repeat(`\`, 2*n))
	b.WriteByte('"')
	return b.String()
}

// qm quotes for the MSYS runtime's own parser: inside double quotes it
// takes a backslash pair as one backslash, so every backslash is doubled
// and every quote escaped.
func qm(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

func withLine(exe, line string) *exec.Cmd {
	c := exec.Command(exe)
	c.SysProcAttr = &syscall.SysProcAttr{CmdLine: line}
	return c
}

// gitRoot finds the Git for Windows installation of the git on PATH.
func gitRoot() string {
	p := strings.TrimSpace(output("git", "--exec-path"))
	if p == "" {
		return ""
	}
	d := filepath.FromSlash(p)
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(d, `usr\bin\sh.exe`)); err == nil {
			return d
		}
		d = filepath.Dir(d)
	}
	return ""
}

func exists(p string) string {
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

func msysAdapter(name, exe string, quote func(string) string, k kind, changed []string) adapter {
	root := gitRoot()
	a := adapter{name: name, want: same, k: k, changed: changed,
		version: "git " + output("git", "--version") + "; root " + root}
	if exe != "" {
		a.version += "; " + exe + " " + output(exe, "-c", `printf %s "$BASH_VERSION"`)
		a.cmd = func(p string) *exec.Cmd { return withLine(exe, quote(exe)+" -c "+quote(shScript(p))) }
	}
	return a
}

func TestAdapterShMSYS(t *testing.T) {
	runAdapter(t, msysAdapter("sh-msys-quoting", exists(filepath.Join(gitRoot(), `usr\bin\sh.exe`)), qm, normal, nil))
}

func TestAdapterBashMSYS(t *testing.T) {
	runAdapter(t, msysAdapter("bash-msys-quoting", exists(filepath.Join(gitRoot(), `usr\bin\bash.exe`)), qm, normal, nil))
}

func TestAdapterShWrapperMSYS(t *testing.T) {
	runAdapter(t, msysAdapter("bin-sh-wrapper-msys-quoting", exists(filepath.Join(gitRoot(), `bin\sh.exe`)), qm, normal, nil))
}

func TestAdapterShMSVCRT(t *testing.T) {
	// Counterexample: MSVCRT quoting (Go's own rules) loses a backslash
	// of every pair under the MSYS parser.
	runAdapter(t, msysAdapter("sh-msvcrt-quoting", exists(filepath.Join(gitRoot(), `usr\bin\sh.exe`)), q, counter,
		[]string{`two\\`, `\\server\share\`}))
}

func psAdapter(name, exe string) adapter {
	a := adapter{name: name, want: hexOf, k: normal}
	if exe != "" {
		a.version = exe + " " + output(exe, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "$PSVersionTable.PSVersion.ToString()")
		a.cmd = func(p string) *exec.Cmd {
			return withLine(exe, q(exe)+" -NoLogo -NoProfile -NonInteractive -Command "+q(psScript(p)))
		}
	}
	return a
}

func sysDir() string {
	d, _ := windows.GetSystemDirectory()
	return d
}

func TestAdapterPowerShell(t *testing.T) {
	runAdapter(t, psAdapter("powershell-msvcrt-quoting", exists(filepath.Join(sysDir(), `WindowsPowerShell\v1.0\powershell.exe`))))
}

func TestAdapterPwsh(t *testing.T) {
	exe, _ := exec.LookPath("pwsh")
	runAdapter(t, psAdapter("pwsh-msvcrt-quoting", exe))
}

func TestAdapterCmd(t *testing.T) {
	// cmd gets the command raw in /d /s /c "...": the exit code comes back
	// as written, and a multi-line command runs its first line only (D3).
	s := begin(t, "T5", "T5.adapter.cmd-raw", normal, true, true)
	defer s.done()
	exe := exists(filepath.Join(sysDir(), "cmd.exe"))
	if exe == "" {
		s.unsupported("cmd.exe not found")
	}
	run := func(c string) (string, int) {
		cmd := withLine(exe, q(exe)+` /d /s /c "`+c+`"`)
		var out strings.Builder
		cmd.Stdout = &out
		startHelper(s, cmd)
		childOf(cmd).reap(30 * time.Second)
		s.worked(1)
		return strings.TrimSpace(out.String()), cmd.ProcessState.ExitCode()
	}
	if out, code := run("echo plain"); out != "plain" || code != 0 {
		s.invalidf("sanity run failed: %q %d", out, code)
	}
	_, c7 := run("exit /b 7")
	_, c9009 := run("exit /b 9009")
	lf, _ := run("echo one\necho two")
	crlf, _ := run("echo one\r\necho two")
	s.note("version %s", windowsVersion())
	s.expect("exit /b 7", 7, c7)
	s.expect("exit /b 9009", 9009, c9009)
	s.expect("LF: lines run", "one", lf)
	s.expect("CRLF: lines run", "one", crlf)
}

func windowsVersion() string { return osVersion() }
