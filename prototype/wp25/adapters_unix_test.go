//go:build linux || darwin

package wp25

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func shAdapter(name, exe, version string) adapter {
	a := adapter{name: name, want: same, k: normal, version: version}
	if exe != "" {
		a.cmd = func(p string) *exec.Cmd { return exec.Command(exe, "-c", shScript(p)) }
	}
	return a
}

func lookPath(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

func TestAdapterSh(t *testing.T) {
	real, _ := filepath.EvalSymlinks("/bin/sh")
	v := "/bin/sh -> " + real + "; BASH_VERSION=" + output("/bin/sh", "-c", `printf %s "${BASH_VERSION:-none}"`)
	runAdapter(t, shAdapter("sh-argv", "/bin/sh", v))
}

func TestAdapterBash(t *testing.T) {
	exe := lookPath("bash")
	v := exe
	if exe != "" {
		v += " " + output(exe, "-c", `printf %s "$BASH_VERSION"`)
	}
	runAdapter(t, shAdapter("bash-argv", exe, v))
}

func TestAdapterDash(t *testing.T) {
	exe := lookPath("dash")
	runAdapter(t, shAdapter("dash-argv", exe, exe))
}

func TestAdapterPwsh(t *testing.T) {
	exe := lookPath("pwsh")
	a := adapter{name: "pwsh-argv", want: hexOf, k: normal}
	if exe != "" {
		a.version = exe + " " + output(exe, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "$PSVersionTable.PSVersion.ToString()")
		a.cmd = func(p string) *exec.Cmd {
			return exec.Command(exe, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", psScript(p))
		}
	}
	runAdapter(t, a)
}
