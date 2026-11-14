package cond

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

func write(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, c := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const embedGo = "//go:build !nowebui\n\npackage web\n\nimport (\n\t\"embed\"\n\t\"io/fs\"\n)\n\nvar dist embed.FS\n\nfunc Assets() (fs.FS, error) { return fs.Sub(dist, \"dist\") }\n"

var repoFiles = map[string]string{
	"go.mod":                   "module example.com/r\n\ngo 1.27\n",
	"main.go":                  "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println() }\n",
	"main_test.go":             "package main\n",
	"web/embed_nowebui.go":     "//go:build nowebui\n\npackage web\n",
	"internal/testutil/t.go":   "//go:build !nowebui\n\npackage testutil\n",
	"testdata/d.go":            "//go:build !nowebui\n\npackage d\n",
	"nested/go.mod":            "module example.com/nested\n",
	"nested/n.go":              "//go:build !nowebui\n\npackage nested\n",
	"_ignored/i.go":            "//go:build !nowebui\n\npackage i\n",
	"lib/lib.go":               "package lib\n",
	"lib/lib_linux.go":         "package lib\n",
	"lib/lib_windows_arm64.go": "package lib\n",
}

func target(goos, goarch string) Target {
	return Target{
		Config:      spec.Config{GOOS: goos, GOARCH: goarch},
		Compiler:    "gc",
		ToolTags:    []string{goarch + ".v1"},
		ReleaseTags: []string{"go1.1", "go1.27"},
		ContextSHA:  "ctx",
	}
}

func classesOf(fs []judge.Failure) string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Class.String()+":"+f.Item+":"+f.Kind)
	}
	sort.Strings(out)
	return strings.Join(out, "|")
}

// repoInput builds the input for a repository fixture: every production
// directory, with the listed GoFiles taken from the expectation map.
func repoInput(t *testing.T, root string, exc []spec.EmbedException, listed map[string][]string) Input {
	t.Helper()
	plain := modsnap.NewPlain(modsnap.OS{}, "repo")
	dirs, fails := RepoDirs(modsnap.OS{}, root, spec.RepoSkipDirs)
	if len(fails) > 0 {
		t.Fatal(fails)
	}
	in := Input{Target: target("linux", "amd64"), RelDir: map[*Pkg]string{}, Exceptions: exc}
	for _, d := range dirs {
		abs := filepath.Join(root, filepath.FromSlash(d))
		p := &Pkg{ImportPath: "example.com/r/" + d, Dir: abs, Kind: Main, Src: PlainSource{P: plain, Dir: abs, RelDir: d}}
		if gf, ok := listed[d]; ok {
			p.Listed, p.GoFiles = true, gf
		}
		in.Packages = append(in.Packages, p)
		in.RelDir[p] = d
	}
	return in
}

func TestRepoDirs(t *testing.T) {
	root := t.TempDir()
	write(t, root, repoFiles)
	dirs, fails := RepoDirs(modsnap.OS{}, root, spec.RepoSkipDirs)
	if len(fails) > 0 || strings.Join(dirs, ",") != ",lib,web" {
		t.Fatalf("dirs %q fails %v", dirs, fails)
	}
}

func TestRepoSteps(t *testing.T) {
	sum := sha256.Sum256([]byte(embedGo))
	exc := []spec.EmbedException{{Path: "web/embed.go", Size: len(embedGo), SHA256: hex.EncodeToString(sum[:]), Constraint: "//go:build !nowebui", Imports: []string{"embed", "io/fs"}}}
	listed := map[string][]string{"": {"main.go"}, "lib": {"lib.go", "lib_linux.go"}, "web": {"embed_nowebui.go"}}
	cases := []struct {
		name  string
		extra map[string]string
		drop  string
		exc   []spec.EmbedException
		want  [6]string
	}{
		{name: "healthy with the exception", extra: map[string]string{"web/embed.go": embedGo}, exc: exc},
		{name: "healthy without exceptions (LC-6 baseline)", exc: nil},
		{name: "a !nowebui file with exec (Review 126 counterexample)", exc: nil,
			extra: map[string]string{"lib/bad.go": "//go:build !nowebui\n\npackage lib\n\nimport \"os/exec\"\n\nfunc F() { _ = exec.Command(\"x\").Run() }\n"},
			want:  [6]string{2: "J-3:lib/bad.go:production-only file not in the exception list"}},
		{name: "a cgo file", exc: nil,
			extra: map[string]string{"lib/c.go": "//go:build cgo\n\npackage lib\n"},
			want:  [6]string{2: "J-3:lib/c.go:production-only file not in the exception list"}},
		{name: "a file importing C", exc: nil,
			extra: map[string]string{"lib/cgo.go": "package lib\n\nimport \"C\"\n"},
			want:  [6]string{2: "J-3:lib/cgo.go:production-only file not in the exception list"}},
		{name: "exception changed one byte", extra: map[string]string{"web/embed.go": strings.Replace(embedGo, "dist", "dost", 1)}, exc: exc,
			want: [6]string{2: "J-3:web/embed.go:exception content differs"}},
		{name: "exception with another import", extra: map[string]string{"web/embed.go": strings.Replace(embedGo, "\"io/fs\"\n", "\"io/fs\"\n\t\"os\"\n", 1)}, exc: exc,
			want: [6]string{2: "J-3:web/embed.go:exception content differs|J-3:web/embed.go:exception imports differ"}},
		{name: "exception missing", exc: exc,
			want: [6]string{2: "J-3:web/embed.go:exception file is not a production-only file"}},
		{name: "production-only file with a non-standard import", exc: nil,
			extra: map[string]string{"lib/x.go": "//go:build !nowebui\n\npackage lib\n\nimport _ \"example.com/other\"\n"},
			want:  [6]string{2: "J-3:lib/x.go:production-only file not in the exception list", 4: "J-3:lib/x.go:production-only file imports a non-standard package"}},
		{name: "listed files differ", exc: nil, drop: "lib/lib_linux.go",
			want: [6]string{1: "J-3:example.com/r/lib:classification differs from the third call"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{}
			for k, v := range repoFiles {
				files[k] = v
			}
			for k, v := range c.extra {
				files[k] = v
			}
			delete(files, c.drop)
			write(t, root, files)
			res := Check(repoInput(t, root, c.exc, listed))
			if !res.Complete || len(res.Read) > 0 || len(res.Budget) > 0 {
				t.Fatalf("incomplete: %v %v", res.Read, res.Budget)
			}
			for step := 1; step <= 5; step++ {
				if !res.Evaluated[step] {
					t.Fatalf("step %d not evaluated", step)
				}
				if got := classesOf(res.Steps[step]); got != c.want[step] {
					t.Errorf("step %d: %q, want %q", step, got, c.want[step])
				}
			}
		})
	}
}

func moduleSnapshot(t *testing.T, files map[string]string) (*modsnap.Snapshot, string) {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, files)
	s0 := modsnap.Take(modsnap.OS{}, "example.com/q", "v1.0.0", dir, "")
	s := modsnap.Take(modsnap.OS{}, "example.com/q", "v1.0.0", dir, s0.H1)
	if !s.Holds() {
		t.Fatalf("snapshot: %v", s.Failures)
	}
	return s, dir
}

func TestThirdPartyAndStd(t *testing.T) {
	s, dir := moduleSnapshot(t, map[string]string{
		"go.mod":       "module example.com/q\n",
		"q.go":         "package q\n",
		"q_cgo.go":     "//go:build cgo\n\npackage q\n\nimport \"C\"\n",
		"q_v2.go":      "//go:build amd64.v2\n\npackage q\n",
		"q_test.go":    "package q\n",
		"sub/other.go": "package sub\n",
	})
	r := modsnap.NewReader(modsnap.OS{}, []*modsnap.Snapshot{s}, nil)
	pkg := func(listed []string) *Pkg {
		return &Pkg{ImportPath: "example.com/q", Dir: dir, Kind: ThirdParty, Listed: true, GoFiles: listed, Src: ModuleSource{R: r, M: s}}
	}
	stdDir := t.TempDir()
	write(t, stdDir, map[string]string{"s.go": "package s\n\nimport \"example.com/x\"\n", "s_cgo.go": "//go:build cgo\n\npackage s\n\nimport \"C\"\n"})
	std := &Pkg{ImportPath: "s", Dir: stdDir, Kind: Std, Src: PlainSource{P: modsnap.NewPlain(modsnap.OS{}, "std"), Dir: stdDir, RelDir: "s"}}

	res := Check(Input{Target: target("linux", "amd64"), Packages: []*Pkg{pkg([]string{"q.go"}), std}, RelDir: map[*Pkg]string{}})
	if got := classesOf(res.Steps[1]); got != "" {
		t.Fatalf("step 1: %s", got)
	}
	// The cgo variant compiles q_cgo.go as a cgo file: step 3 fails.
	if got := classesOf(res.Steps[3]); got != "J-3:example.com/q:production variant not analysed" {
		t.Fatalf("step 3: %s", got)
	}
	// The standard library fixture imports a non-standard path in every
	// production variant; "C" is allowed only with cgo.
	if got := classesOf(res.Steps[5]); got != "J-3:s:import outside standard library and x/sys|J-3:s:import outside standard library and x/sys" {
		t.Fatalf("step 5: %s", got)
	}
	if !r.WasConsumed(s, "q_cgo.go", modsnap.GroupC) || r.WasConsumed(s, "q_test.go", modsnap.GroupC) {
		t.Fatal("group C must read every candidate and no test file")
	}

	// Target tags matter (N-127-1): with the host's tags the amd64.v2 file
	// is classified differently and step 1 catches it.
	tv2 := target("linux", "amd64")
	tv2.ToolTags = []string{"amd64.v1", "amd64.v2"}
	res2 := Check(Input{Target: tv2, Packages: []*Pkg{pkg([]string{"q.go"})}, RelDir: map[*Pkg]string{}})
	if got := classesOf(res2.Steps[1]); got != "J-3:example.com/q:classification differs from the third call" {
		t.Fatalf("tag-sensitive step 1: %s", got)
	}
	res3 := Check(Input{Target: tv2, Packages: []*Pkg{pkg([]string{"q.go", "q_v2.go"})}, RelDir: map[*Pkg]string{}})
	if got := classesOf(res3.Steps[1]); got != "" {
		t.Fatalf("tag-sensitive healthy step 1: %s", got)
	}
}

func TestAdmissionAndReadFailures(t *testing.T) {
	s, dir := moduleSnapshot(t, map[string]string{"go.mod": "module example.com/q\n", "q.go": "package q\n"})
	r := modsnap.NewReader(modsnap.OS{}, []*modsnap.Snapshot{s}, nil)
	p := &Pkg{ImportPath: "example.com/q", Dir: dir, Kind: ThirdParty, Listed: true, GoFiles: []string{"q.go"}, Src: ModuleSource{R: r, M: s}}
	res := Check(Input{Target: target("linux", "amd64"), Packages: []*Pkg{p}, RelDir: map[*Pkg]string{}, Admit: func(string) bool { return false }})
	if res.Complete || res.Evaluated[1] || res.Evaluated[3] || classesOf(res.Budget) != "J-1:classify example.com/q:budget insufficient" {
		t.Fatalf("admission: %+v", res)
	}
	// The file changes after I-4: the controlled read refuses it (J-4) and
	// the steps that need the package are not evaluated.
	if err := os.WriteFile(filepath.Join(dir, "q.go"), []byte("package q // changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res = Check(Input{Target: target("linux", "amd64"), Packages: []*Pkg{p}, RelDir: map[*Pkg]string{}})
	if res.Complete || res.Evaluated[1] || !strings.Contains(classesOf(res.Read), "J-4:q.go:sha256 differs from M") {
		t.Fatalf("read failure: %v", res.Read)
	}
}

func TestConstraint(t *testing.T) {
	cases := []struct {
		in    string
		line  string
		found bool
		err   bool
	}{
		{"package p\n", "", false, false},
		{"// c\n\n//go:build linux\r\n\npackage p\n", "//go:build linux", true, false},
		{"//go:build a\n//go:build b\n\npackage p\n", "", false, true},
		// go/build keeps accepting //go:build after a closed /* */ block
		// (parseFileHeader checks only inSlashStar), and so does Constraint.
		{"/* x */\n//go:build a\npackage p\n", "//go:build a", true, false},
		{"package p\n//go:build a\n", "", false, false},
	}
	for _, c := range cases {
		line, found, err := Constraint([]byte(c.in))
		if line != c.line || found != c.found || (err != nil) != c.err {
			t.Errorf("%q: %q %v %v", c.in, line, found, err)
		}
	}
}
