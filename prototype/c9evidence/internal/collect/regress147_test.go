package collect

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// readCloseFS fails both the read and the close of files whose name is in
// bad; every other file is the real one.
type readCloseFS struct {
	modsnap.OS
	bad map[string]bool
}

func (f readCloseFS) Open(path string) (modsnap.File, error) {
	file, err := f.OS.Open(path)
	if err != nil || !f.bad[filepath.Base(path)] {
		return file, err
	}
	return &readCloseFile{File: file}, nil
}

type readCloseFile struct{ modsnap.File }

var (
	errRead  = errors.New("EIO on read")
	errClose = errors.New("EBADF on close")
)

func (r *readCloseFile) Read([]byte) (int, error) { return 0, errRead }

func (r *readCloseFile) Close() error {
	_ = r.File.Close()
	return errClose
}

type stillClock struct{ t time.Time }

func (c stillClock) Now() time.Time { return c.t }

// Review 147 B-147-3, path one: a read error and a close error of the same
// read are both kept and both consumed by A4 (two J-1 failures, the close
// one not dropped).
func TestB1473ReadAndCloseErrorsBothKeptByA4(t *testing.T) {
	r := t.TempDir()
	for name, c := range map[string]string{"go.sum": "", "go.mod": "module example.com/r\n"} {
		if err := os.WriteFile(filepath.Join(r, name), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	clk := stillClock{time.Unix(1_000_000, 0)}
	res := Collect(Options{
		Config: spec.Config{GOOS: "linux", GOARCH: "amd64"}, R: r, Invoker: Stopped{Clock: clk},
		Budget: NewBudget(clk, clk.t, 120*time.Second),
		Guard:  func() GuardValues { return GuardValues{AnalyzerVersion: "go1.27.3"} },
		FS:     readCloseFS{bad: map[string]bool{"go.sum": true}}, MainModule: "example.com/r", Rec: &MemRecorder{},
	})
	a4, _ := res.Checks.Get("A4")
	var kinds []string
	for _, f := range a4.Failures {
		kinds = append(kinds, f.Class.String()+" "+f.Item+" "+f.Kind)
	}
	got := strings.Join(kinds, "|")
	for _, want := range []string{"J-1 go.sum read error", "J-1 go.sum close error"} {
		if !strings.Contains(got, want) {
			t.Errorf("A4 failures %q: missing %q", got, want)
		}
	}
	if a4.Status != judge.Fail {
		t.Errorf("A4 %s", a4.Status)
	}
}

// The guard keeps both errors of reading VERSION.
func TestB1473ReadAndCloseErrorsBothKeptByTheGuard(t *testing.T) {
	_, cand := fakeRoot(t, "go1.27.3", "go")
	src := sources("go1.27.3", cand)
	src.FS = readCloseFS{bad: map[string]bool{"VERSION": true}}
	v := ReadGuard(src)
	got := strings.Join(v.Errors, "|")
	for _, want := range []string{errRead.Error(), errClose.Error()} {
		if !strings.Contains(got, want) {
			t.Errorf("guard errors %q: missing %q", got, want)
		}
	}
	_ = fs.ErrNotExist
}
