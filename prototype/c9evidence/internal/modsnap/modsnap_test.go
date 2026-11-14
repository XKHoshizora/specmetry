package modsnap

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
)

// manualH1 computes Hash1 from its definition (method record §2.2 item 6),
// independently of dirhash.
func manualH1(files map[string]string) string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		s := sha256.Sum256([]byte(files[n]))
		fmt.Fprintf(&b, "%x  %s\n", s, n)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return "h1:" + base64.StdEncoding.EncodeToString(sum[:])
}

func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

var baseFiles = map[string]string{
	"go.mod":   "module example.com/q\n\ngo 1.22\n",
	"q.go":     "package q\n\nfunc F() int { return 1 }\n",
	"sub/s.go": "package sub\n",
}

func prefixed(prefix string, files map[string]string) map[string]string {
	out := map[string]string{}
	for rel, c := range files {
		out[prefix+"/"+rel] = c
	}
	return out
}

func classes(fs []judge.Failure) string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Class.String()+":"+f.Item+":"+f.Kind)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

func TestTakeHealthy(t *testing.T) {
	dir := writeModule(t, baseFiles)
	want := manualH1(prefixed("example.com/q@v1.0.0", baseFiles))
	s := Take(OS{}, "example.com/q", "v1.0.0", dir, want)
	if !s.Holds() || s.H1 != want || len(s.Failures) != 0 {
		t.Fatalf("h1 %s want %s failures %v", s.H1, want, s.Failures)
	}
	rec, ok := s.File("sub/s.go")
	sum := sha256.Sum256([]byte(baseFiles["sub/s.go"]))
	if !ok || rec.SHA256 != hex.EncodeToString(sum[:]) || rec.Size != int64(len(baseFiles["sub/s.go"])) || rec.Name != "example.com/q@v1.0.0/sub/s.go" {
		t.Fatalf("record %+v", rec)
	}
	if names, ok := s.DirNames(""); !ok || strings.Join(names, ",") != "go.mod,q.go" {
		t.Fatalf("root names %v", names)
	}
}

func TestTakeNegatives(t *testing.T) {
	want := manualH1(prefixed("example.com/q@v1.0.0", baseFiles))
	t.Run("one byte changed", func(t *testing.T) {
		files := map[string]string{}
		for k, v := range baseFiles {
			files[k] = v
		}
		files["q.go"] = strings.Replace(files["q.go"], "1", "2", 1)
		s := Take(OS{}, "example.com/q", "v1.0.0", writeModule(t, files), want)
		if got := classes(s.Failures); got != "J-4:.:h1 differs from go.sum" {
			t.Fatalf("failures %s", got)
		}
	})
	t.Run("one more file", func(t *testing.T) {
		files := map[string]string{"extra.txt": "x"}
		for k, v := range baseFiles {
			files[k] = v
		}
		s := Take(OS{}, "example.com/q", "v1.0.0", writeModule(t, files), want)
		if got := classes(s.Failures); got != "J-4:.:h1 differs from go.sum" {
			t.Fatalf("failures %s", got)
		}
	})
	t.Run("symbolic link", func(t *testing.T) {
		dir := writeModule(t, baseFiles)
		if err := os.Symlink(filepath.Join(dir, "q.go"), filepath.Join(dir, "link.go")); err != nil {
			t.Skipf("cannot create a symbolic link here: %v", err)
		}
		s := Take(OS{}, "example.com/q", "v1.0.0", dir, want)
		if got := classes(s.Failures); got != "J-2:link.go:not a regular file" || s.H1 != "" {
			t.Fatalf("failures %s h1 %q", got, s.H1)
		}
	})
}

// faultFS wraps OS and injects faults by path suffix.
type faultFS struct {
	OS
	openErr, readErr, closeErr map[string]error
	notRegular                 map[string]bool
}

func (f faultFS) Open(path string) (File, error) {
	for suffix, err := range f.openErr {
		if strings.HasSuffix(filepath.ToSlash(path), suffix) {
			return nil, err
		}
	}
	file, err := f.OS.Open(path)
	if err != nil {
		return nil, err
	}
	ff := &faultFile{File: file}
	for suffix, err := range f.readErr {
		if strings.HasSuffix(filepath.ToSlash(path), suffix) {
			ff.readErr = err
		}
	}
	for suffix, err := range f.closeErr {
		if strings.HasSuffix(filepath.ToSlash(path), suffix) {
			ff.closeErr = err
		}
	}
	for suffix := range f.notRegular {
		if strings.HasSuffix(filepath.ToSlash(path), suffix) {
			ff.notRegular = true
		}
	}
	return ff, nil
}

type faultFile struct {
	File
	readErr, closeErr error
	notRegular        bool
}

func (f *faultFile) Read(p []byte) (int, error) {
	if f.readErr != nil {
		return 0, f.readErr
	}
	return f.File.Read(p)
}

func (f *faultFile) Close() error {
	err := f.File.Close()
	if f.closeErr != nil {
		return f.closeErr
	}
	return err
}

type dirInfo struct{ fs.FileInfo }

func (dirInfo) Mode() fs.FileMode { return fs.ModeDir | 0o755 }

func (f *faultFile) Stat() (fs.FileInfo, error) {
	fi, err := f.File.Stat()
	if err != nil || !f.notRegular {
		return fi, err
	}
	return dirInfo{fi}, nil
}

func TestTakeFaults(t *testing.T) {
	want := manualH1(prefixed("example.com/q@v1.0.0", baseFiles))
	dir := writeModule(t, baseFiles)
	e := errors.New("EIO")
	cases := []struct {
		name string
		fsys faultFS
		want string
	}{
		{"open", faultFS{openErr: map[string]error{"q.go": e}}, "J-1:q.go:open error"},
		{"read", faultFS{readErr: map[string]error{"q.go": e}}, "J-1:q.go:read error"},
		{"close", faultFS{closeErr: map[string]error{"q.go": e}}, "J-1:q.go:close error"},
		{"handle", faultFS{notRegular: map[string]bool{"q.go": true}}, "J-2:q.go:handle not a regular file"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := Take(c.fsys, "example.com/q", "v1.0.0", dir, want)
			if got := classes(s.Failures); got != c.want {
				t.Fatalf("failures %s, want %s", got, c.want)
			}
			if c.name == "close" {
				// The bytes were read; the real close error is kept on the
				// record and h1 is still computed from the bytes read.
				rec, _ := s.File("q.go")
				if rec.CloseErr != "EIO" || s.H1 != want {
					t.Fatalf("record %+v h1 %s", rec, s.H1)
				}
			} else if s.H1 != "" {
				t.Fatalf("h1 computed despite %s", c.name)
			}
		})
	}
}

func TestControlledRead(t *testing.T) {
	want := manualH1(prefixed("example.com/q@v1.0.0", baseFiles))
	setup := func(t *testing.T, fsys FS, inject Injector) (*Reader, *Snapshot, string) {
		dir := writeModule(t, baseFiles)
		s := Take(OS{}, "example.com/q", "v1.0.0", dir, want)
		if !s.Holds() {
			t.Fatalf("snapshot %v", s.Failures)
		}
		return NewReader(fsys, []*Snapshot{s}, inject), s, dir
	}
	t.Run("healthy and same bytes", func(t *testing.T) {
		r, s, dir := setup(t, OS{}, nil)
		b, err := r.Read(s, "q.go", GroupC, "classify")
		if err != nil || string(b) != baseFiles["q.go"] {
			t.Fatalf("%q %v", b, err)
		}
		// A change after the first read is not seen: the same B is handed out.
		if err := os.WriteFile(filepath.Join(dir, "q.go"), []byte("changed"), 0o644); err != nil {
			t.Fatal(err)
		}
		b2, err := r.Read(s, "q.go", GroupP, "parse")
		if err != nil || string(b2) != string(b) || len(r.Consumed()) != 2 {
			t.Fatalf("%q %v %v", b2, err, r.Consumed())
		}
		if !r.WasConsumed(s, "q.go", GroupP) || r.WasConsumed(s, "go.mod", GroupP) {
			t.Fatal("consumption records")
		}
	})
	t.Run("name not in M", func(t *testing.T) {
		r, s, _ := setup(t, OS{}, nil)
		_, err := r.Read(s, "nope.go", GroupP, "parse")
		if f, _ := FailureOf(err); f.Class != judge.J4 {
			t.Fatalf("%v", err)
		}
	})
	t.Run("reread differs", func(t *testing.T) {
		r, s, dir := setup(t, OS{}, nil)
		if err := os.WriteFile(filepath.Join(dir, "q.go"), []byte("changed"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := r.Read(s, "q.go", GroupC, "classify")
		if f, _ := FailureOf(err); f.Class != judge.J4 || f.Kind != "sha256 differs from M" {
			t.Fatalf("%v", err)
		}
	})
	t.Run("handle not regular", func(t *testing.T) {
		r, s, _ := setup(t, faultFS{notRegular: map[string]bool{"q.go": true}}, nil)
		_, err := r.Read(s, "q.go", GroupC, "classify")
		if f, _ := FailureOf(err); f.Class != judge.J2 {
			t.Fatalf("%v", err)
		}
	})
	t.Run("injection after a real nil close", func(t *testing.T) {
		n := 0
		inj := func(string) (int, error) { n++; return n, errors.New("injected E") }
		r, s, _ := setup(t, OS{}, inj)
		_, err := r.Read(s, "q.go", GroupC, "classify")
		f, _ := FailureOf(err)
		if f.Class != judge.J1 || f.Source != judge.SourceInjected || f.Inject != 1 {
			t.Fatalf("%+v", f)
		}
		if got := r.Injections(); len(got) != 1 || got[0].ID != 1 {
			t.Fatalf("injections %v", got)
		}
		for _, x := range r.Failures() {
			if x.Source == judge.SourceReal {
				t.Fatalf("a real failure appeared: %v", x)
			}
		}
	})
	t.Run("real close error is never replaced by an injection", func(t *testing.T) {
		asked := 0
		inj := func(string) (int, error) { asked++; return asked, errors.New("injected E") }
		r, s, _ := setup(t, faultFS{closeErr: map[string]error{"q.go": errors.New("EIO")}}, inj)
		_, err := r.Read(s, "q.go", GroupC, "classify")
		f, _ := FailureOf(err)
		if f.Source != judge.SourceReal || f.Kind != "close error" || asked != 0 || len(r.Injections()) != 0 {
			t.Fatalf("%+v asked %d", f, asked)
		}
	})
	t.Run("view and locate", func(t *testing.T) {
		r, s, dir := setup(t, OS{}, nil)
		infos, err := r.ReadDir(dir)
		if err != nil || len(infos) != 2 || infos[0].Name() != "go.mod" || infos[1].Name() != "q.go" {
			t.Fatalf("%v %v", infos, err)
		}
		if infos[1].Size() != int64(len(baseFiles["q.go"])) || infos[1].Mode() != 0 || !infos[1].ModTime().IsZero() || infos[1].IsDir() || infos[1].Sys() != nil {
			t.Fatal("view metadata must come from H only")
		}
		if _, err := r.ReadDir(t.TempDir()); err == nil {
			t.Fatal("outside directory must fail")
		} else if f, _ := FailureOf(err); f.Class != judge.J3 {
			t.Fatalf("%v", f)
		}
		open := r.OpenFile(GroupC, "classify")
		rc, err := open(filepath.Join(dir, "sub", "s.go"))
		if err != nil {
			t.Fatal(err)
		}
		_ = rc.Close()
		if _, err := open(filepath.Join(t.TempDir(), "x.go")); err == nil {
			t.Fatal("outside path must fail")
		}
		m, rel, ok := r.Locate(filepath.Join(dir, "sub"))
		if !ok || m != s || rel != "sub" {
			t.Fatalf("locate %v %q", ok, rel)
		}
	})
}
