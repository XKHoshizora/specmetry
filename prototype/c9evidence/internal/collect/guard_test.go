package collect

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
)

// fakeRoot lays out <root>/bin/go(.exe), <root>/src/cmd/go and VERSION in
// a temporary directory. The "executable" is a few bytes that are hashed,
// never run.
func fakeRoot(t *testing.T, version string, exe string) (root, cand string) {
	t.Helper()
	root = t.TempDir()
	for _, d := range []string{"bin", filepath.Join("src", "cmd", "go")} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cand = filepath.Join(root, "bin", exe)
	if err := os.WriteFile(cand, []byte("not a program"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte(version+"\ntime 2026-10-19T22:20:36Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, cand
}

func sources(version, cand string) GuardSources {
	return GuardSources{
		Version:      func() string { return version },
		LookPath:     func(string) (string, error) { return cand, nil },
		EvalSymlinks: func(p string) (string, error) { return p, nil },
		FS:           modsnap.OS{},
		GOOS:         "linux",
	}
}

func kinds(fs []judge.Failure) string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Class.String()+":"+f.Item+":"+f.Kind)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

func TestGuard(t *testing.T) {
	root, cand := fakeRoot(t, "go1.27.3", "go")
	v := ReadGuard(sources("go1.27.3", cand))
	sum := sha256.Sum256([]byte("not a program"))
	if v.Root != root || v.VersionLine != "go1.27.3" || v.ExeSHA256 != hex.EncodeToString(sum[:]) || !v.SrcCmdGo || !v.ExeRegular {
		t.Fatalf("values %+v", v)
	}
	if got := EvaluatePre(v); len(got) != 0 {
		t.Fatalf("healthy guard: %v", got)
	}
	env := map[string]string{"GOVERSION": "go1.27.3", "GOROOT": root}
	if got := EvaluatePost(v, env); len(got) != 0 {
		t.Fatalf("healthy post guard: %v", got)
	}

	cases := []struct {
		name string
		v    func() GuardValues
		want string
	}{
		{"analyser go1.27.4", func() GuardValues { x := v; x.AnalyzerVersion = "go1.27.4"; return x }, "J-1:analyzer:version"},
		{"VERSION go1.27.4", func() GuardValues {
			_, c := fakeRoot(t, "go1.27.4", "go")
			return ReadGuard(sources("go1.27.3", c))
		}, "J-1:VERSION:version"},
		{"both go1.27.4 (equal to each other is not enough)", func() GuardValues {
			_, c := fakeRoot(t, "go1.27.4", "go")
			return ReadGuard(sources("go1.27.4", c))
		}, "J-1:VERSION:version,J-1:analyzer:version"},
		{"shim outside bin", func() GuardValues {
			dir := t.TempDir()
			shim := filepath.Join(dir, "shims", "go")
			_ = os.MkdirAll(filepath.Dir(shim), 0o755)
			_ = os.WriteFile(shim, []byte("#!/bin/sh\n"), 0o755)
			return ReadGuard(sources("go1.27.3", shim))
		}, "J-1:VERSION:version,J-1:candidate:layout,J-1:candidate:layout,J-1:candidate:read error"},
		{"lookup fails", func() GuardValues {
			s := sources("go1.27.3", cand)
			s.LookPath = func(string) (string, error) { return "", errors.New("not found") }
			return ReadGuard(s)
		}, "J-1:candidate:read error"},
	}
	for _, c := range cases {
		if got := kinds(EvaluatePre(c.v())); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
	if got := kinds(EvaluatePost(v, map[string]string{"GOVERSION": "go1.27.4", "GOROOT": "/elsewhere"})); got != "J-1:GOROOT:root,J-1:GOVERSION:version" {
		t.Errorf("post guard: %s", got)
	}
}
