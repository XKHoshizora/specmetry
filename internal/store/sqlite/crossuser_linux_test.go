package sqlite

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

// Two unprivileged users for the cross-user tests. The test itself runs as
// root only to start processes as them; nothing it checks relies on root.
const (
	victimUID   = 4242
	attackerUID = 65534 // nobody
)

type crossUser struct {
	t    *testing.T
	base string
	bin  string
}

func newCrossUser(t *testing.T) *crossUser {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("cross-user tests need root to start processes as two different users (CI runs them with sudo)")
	}
	base, err := os.MkdirTemp("/tmp", "specmetry-crossuser-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	if err := os.Chmod(base, 0o755); err != nil {
		t.Fatal(err)
	}
	// The test binary is copied where both users can run it.
	bin := filepath.Join(base, "specmetry.test")
	src, err := os.Open(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.OpenFile(bin, os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatal(err)
	}
	_ = src.Close()
	_ = dst.Close()
	return &crossUser{t: t, base: base, bin: bin}
}

// mkdir creates a directory as root with the given owner and mode.
func (c *crossUser) mkdir(rel string, uid int, mode os.FileMode) string {
	c.t.Helper()
	p := filepath.Join(c.base, rel)
	if err := os.Mkdir(p, 0o700); err != nil {
		c.t.Fatal(err)
	}
	if err := os.Chown(p, uid, uid); err != nil {
		c.t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		c.t.Fatal(err)
	}
	return p
}

func (c *crossUser) command(uid int, helper string, env ...string) *exec.Cmd {
	cmd := exec.Command(c.bin, "-test.run=^$")
	cmd.Dir = "/"
	cmd.Env = append(append(os.Environ(), "SPECMETRY_TEST_HELPER="+helper), env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(uid)}}
	return cmd
}

// open runs a specmetry Open plus one write as the victim and returns the
// exit code and stderr.
func (c *crossUser) open(home, canary string) (int, string) {
	c.t.Helper()
	cmd := c.command(victimUID, "victimopen", datadir.EnvHome+"="+home, "CANARY="+canary)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), stderr.String()
	}
	if err != nil {
		c.t.Fatal(err)
	}
	return 0, stderr.String()
}

// attack runs one attacker operation and returns what it printed.
func (c *crossUser) attack(op, a, b string) string {
	c.t.Helper()
	out, err := c.command(attackerUID, "attacker", "ATTACK_OP="+op, "ATTACK_A="+a, "ATTACK_B="+b).Output()
	if err != nil {
		c.t.Fatalf("attacker %s: %v", op, err)
	}
	return strings.TrimSpace(string(out))
}

// prepareTrap makes a directory the attacker owns, holding a database file
// the attacker can read and that the victim can write.
func (c *crossUser) prepareTrap(rel string) string {
	c.t.Helper()
	trap := c.mkdir(rel, attackerUID, 0o777)
	db := filepath.Join(trap, datadir.DatabaseFile)
	if err := os.WriteFile(db, nil, 0o666); err != nil {
		c.t.Fatal(err)
	}
	if err := os.Chown(db, attackerUID, attackerUID); err != nil {
		c.t.Fatal(err)
	}
	if err := os.Chmod(db, 0o666); err != nil {
		c.t.Fatal(err)
	}
	return trap
}

// codex-review-005 B1, reproduced as reported: another user who can write
// to the data directory's parent keeps swapping the data directory for a
// link to a database they prepared. No write of the victim may land there.
func TestCrossUserPathSwapCannotRedirectWrites(t *testing.T) {
	c := newCrossUser(t)
	shared := c.mkdir("shared", 0, 0o777)
	trap := c.prepareTrap("trap")
	home := filepath.Join(shared, "home")
	stop := filepath.Join(c.base, "stop")

	swapper := c.command(attackerUID, "attacker", "ATTACK_OP=swap", "ATTACK_A="+home, "ATTACK_B="+trap, "ATTACK_STOP="+stop)
	if err := swapper.Start(); err != nil {
		t.Fatal(err)
	}
	canary := testutil.NewCanary(t)
	deadline := time.Now().Add(3 * time.Second)
	opens, refused := 0, 0
	for time.Now().Before(deadline) {
		_ = os.RemoveAll(home)
		code, stderr := c.open(home, canary)
		opens++
		if code == apperr.ExitEnvironment && strings.Contains(stderr, "writable by other users") {
			refused++
		}
	}
	_ = os.WriteFile(stop, nil, 0o600)
	_ = swapper.Wait()
	if hits := testutil.FindInTree(t, trap, canary); len(hits) != 0 {
		t.Fatalf("the victim's data reached the attacker's directory: %v", hits)
	}
	if refused != opens {
		t.Fatalf("%d of %d opens under an other-writable parent were not refused for that reason", opens-refused, opens)
	}
}

// With a parent only root can write to, the other user cannot move the data
// directory or read the database: the kernel refuses, and specmetry works.
func TestCrossUserCannotReplaceDataDirectoryUnderSafeParent(t *testing.T) {
	c := newCrossUser(t)
	safe := c.mkdir("safe", 0, 0o755)
	home := c.mkdir(filepath.Join("safe", "home"), victimUID, 0o700)
	canary := testutil.NewCanary(t)
	if code, stderr := c.open(home, canary); code != 0 {
		t.Fatalf("open under a safe parent: exit %d: %s", code, stderr)
	}
	if out := c.attack("rename", home, filepath.Join(safe, "stolen")); !strings.Contains(out, "permission denied") {
		t.Fatalf("attacker rename: %s", out)
	}
	if out := c.attack("read", filepath.Join(home, datadir.DatabaseFile), ""); !strings.Contains(out, "permission denied") {
		t.Fatalf("attacker read: %s", out)
	}
	if hits := testutil.FindInTree(t, home, canary); len(hits) == 0 {
		t.Fatal("the victim's write is missing")
	}
}

// A sticky, world-writable parent such as /tmp is accepted when the data
// directory belongs to the current user: only its owner can rename it.
func TestCrossUserStickyParent(t *testing.T) {
	c := newCrossUser(t)
	tmp := c.mkdir("tmp", 0, os.ModeSticky|0o777)
	home := filepath.Join(tmp, "home")
	if code, stderr := c.open(home, testutil.NewCanary(t)); code != 0 {
		t.Fatalf("open under a sticky parent: exit %d: %s", code, stderr)
	}
	if out := c.attack("rename", home, filepath.Join(tmp, "stolen")); !strings.Contains(out, "operation not permitted") {
		t.Fatalf("attacker rename in a sticky directory: %s", out)
	}
}

// A directory on the path owned by another user is refused: its owner can
// replace anything below it.
func TestCrossUserAncestorOwnedByAnotherUser(t *testing.T) {
	c := newCrossUser(t)
	c.mkdir("theirs", attackerUID, 0o755)
	home := c.mkdir(filepath.Join("theirs", "home"), victimUID, 0o700)
	canary := testutil.NewCanary(t)
	code, stderr := c.open(home, canary)
	if code != apperr.ExitEnvironment || !strings.Contains(stderr, "owned by uid 65534") {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if hits := testutil.FindInTree(t, home, canary); len(hits) != 0 {
		t.Fatalf("data was written: %v", hits)
	}
}

// repo makes a git repository with one commit, as root, and gives it to
// the victim (git reads only repositories their user owns).
func (c *crossUser) repo(rel string) string {
	c.t.Helper()
	p := filepath.Join(c.base, rel)
	env := append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_AUTHOR_NAME=v", "GIT_AUTHOR_EMAIL=v@v.invalid", "GIT_COMMITTER_NAME=v", "GIT_COMMITTER_EMAIL=v@v.invalid")
	for _, args := range [][]string{{"init", "-q", "-b", "main", p}, {"-C", p, "commit", "-q", "--allow-empty", "-m", "one"}} {
		cmd := exec.Command("git", args...)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			c.t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if out, err := exec.Command("chown", "-R", fmt.Sprintf("%d:%d", victimUID, victimUID), p).CombinedOutput(); err != nil {
		c.t.Fatalf("chown: %v: %s", err, out)
	}
	return p
}

// clean runs the victim's Clean of repo through a Scratch in home and
// returns the exit code, what it printed and its stderr.
func (c *crossUser) clean(home, repo string) (int, string, string) {
	c.t.Helper()
	victimHome := filepath.Join(c.base, "victim-home")
	if _, err := os.Stat(victimHome); err != nil {
		c.mkdir("victim-home", victimUID, 0o700)
	}
	cmd := c.command(victimUID, "victimclean", datadir.EnvHome+"="+home, "REPO="+repo, "HOME="+victimHome)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), stdout.String(), stderr.String()
	}
	if err != nil {
		c.t.Fatal(err)
	}
	return 0, strings.TrimSpace(stdout.String()), stderr.String()
}

// Review 058 B1 with two users: another user who can rename entries in the
// data directory's parent keeps swapping the data directory for a link to
// a directory of theirs while the victim reads git. Every read is refused
// before a private directory is made, and nothing reaches their directory.
func TestCrossUserScratchUnderAWritableParent(t *testing.T) {
	c := newCrossUser(t)
	shared := c.mkdir("shared", 0, 0o777)
	trap := c.mkdir("trap", attackerUID, 0o777)
	repo := c.repo("repo")
	home := filepath.Join(shared, "home")
	stop := filepath.Join(c.base, "stop")
	swapper := c.command(attackerUID, "attacker", "ATTACK_OP=swap", "ATTACK_A="+home, "ATTACK_B="+trap, "ATTACK_STOP="+stop)
	if err := swapper.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	reads, refused := 0, 0
	for time.Now().Before(deadline) {
		_ = os.RemoveAll(home)
		code, _, stderr := c.clean(home, repo)
		reads++
		if code == apperr.ExitEnvironment && strings.Contains(stderr, "writable by other users") {
			refused++
		}
	}
	_ = os.WriteFile(stop, nil, 0o600)
	_ = swapper.Wait()
	if refused != reads {
		t.Fatalf("%d of %d reads under an other-writable parent were not refused for that reason", reads-refused, reads)
	}
	if entries, _ := os.ReadDir(trap); len(entries) != 0 {
		t.Fatalf("the victim made something in the attacker's directory: %v", entries)
	}
}

// With a parent only root can write to, the victim's read works, and the
// other user can rename neither the data directory nor its tmp directory.
func TestCrossUserScratchUnderASafeParent(t *testing.T) {
	c := newCrossUser(t)
	safe := c.mkdir("safe", 0, 0o755)
	home := c.mkdir(filepath.Join("safe", "home"), victimUID, 0o700)
	repo := c.repo("repo")
	if code, out, stderr := c.clean(home, repo); code != 0 || out != "0" {
		t.Fatalf("the victim's read: exit %d, %q: %s", code, out, stderr)
	}
	tmp := filepath.Join(home, datadir.TmpDir)
	if entries, err := os.ReadDir(tmp); err != nil || len(entries) != 0 {
		t.Fatalf("tmp after the read: %v, %v", entries, err)
	}
	for _, a := range [][2]string{{tmp, filepath.Join(home, "stolen")}, {home, filepath.Join(safe, "stolen")}} {
		if out := c.attack("rename", a[0], a[1]); !strings.Contains(out, "permission denied") {
			t.Fatalf("attacker rename of %s: %s", a[0], out)
		}
	}
}

// A directory on the path owned by another user is refused before any
// private directory is made.
func TestCrossUserScratchAncestorOwnedByAnotherUser(t *testing.T) {
	c := newCrossUser(t)
	c.mkdir("theirs", attackerUID, 0o755)
	home := c.mkdir(filepath.Join("theirs", "home"), victimUID, 0o700)
	code, _, stderr := c.clean(home, c.repo("repo"))
	if code != apperr.ExitEnvironment || !strings.Contains(stderr, "owned by uid 65534") {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if _, err := os.Lstat(filepath.Join(home, datadir.TmpDir)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("tmp was made: %v", err)
	}
}

// A sticky, world-writable parent such as /tmp is accepted: only the
// victim can rename their data directory there.
func TestCrossUserScratchStickyParent(t *testing.T) {
	c := newCrossUser(t)
	tmp := c.mkdir("tmp", 0, os.ModeSticky|0o777)
	home := filepath.Join(tmp, "home")
	if code, out, stderr := c.clean(home, c.repo("repo")); code != 0 || out != "0" {
		t.Fatalf("the victim's read under a sticky parent: exit %d, %q: %s", code, out, stderr)
	}
	if out := c.attack("rename", home, filepath.Join(tmp, "stolen")); !strings.Contains(out, "operation not permitted") {
		t.Fatalf("attacker rename in a sticky directory: %s", out)
	}
}
