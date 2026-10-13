//go:build darwin

package datadir

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil/testdir"
)

// codex-review-004 B1: on macOS an extended ACL can grant access that the
// mode bits do not show; it is reported.
func TestDarwinCheckReportsACL(t *testing.T) {
	d := Dir{Path: filepath.Join(testdir.New(t), "home")}
	if err := d.Ensure(); err != nil {
		t.Fatal(err)
	}
	f, err := CreatePrivateFile(d.Join(DatabaseFile), true)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if problems, err := Check(d); err != nil || len(problems) != 0 {
		t.Fatalf("fresh directory: %v, %v", problems, err)
	}
	if out, err := exec.Command("/bin/chmod", "+a", "everyone allow read", d.Join(DatabaseFile)).CombinedOutput(); err != nil {
		t.Fatalf("chmod +a: %v\n%s", err, out)
	}
	problems, err := Check(d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(problems, "\n"), "access control list") {
		t.Fatalf("ACL not reported: %v", problems)
	}
}
