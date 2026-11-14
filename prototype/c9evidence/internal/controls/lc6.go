package controls

import (
	"path/filepath"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/cond"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// The LC-6 fixture: a main module without dependencies. Its pure-embed
// exception list is empty; the repository's web/embed.go exception applies
// to the repository only (plan §5.2 LC-6).
var lc6Base = map[string]string{
	"go.mod":  "module example.com/lc6\n\ngo 1.22\n",
	"main.go": "package main\n\nfunc main() {}\n",
}

// lc6 runs the conditional difference checks on the fixture as the tested
// instance with the checks cond/1 to cond/5. The third call's listed files
// are what a fake caller gives: the main package's GoFiles.
func lc6(name string, extra map[string]string, want []string) Variant {
	return Variant{
		Control: "LC-6", Name: name, Applies: always, MustEnter: []string{"cond/2"}, Want: want,
		Run: func(e *Env) Outcome {
			files := copyFiles(lc6Base)
			for k, v := range extra {
				files[k] = v
			}
			fx, err := prepare(e, "LC-6 "+name, files)
			if err != nil {
				return Outcome{PrepErr: err}
			}
			cfg := spec.ConfigsFor(e.GOOS)[0]
			plain := modsnap.NewPlain(modsnap.OS{}, "repo")
			dirs, walk := cond.RepoDirs(modsnap.OS{}, fx.dir, spec.RepoSkipDirs)
			in := cond.Input{
				Target: cond.Target{Config: cfg, Compiler: "gc", ToolTags: []string{cfg.GOARCH + ".v1"}, ReleaseTags: []string{"go1.1", "go1.27"}},
				RelDir: map[*cond.Pkg]string{},
			}
			for _, d := range dirs {
				abs := filepath.Join(fx.dir, filepath.FromSlash(d))
				p := &cond.Pkg{ImportPath: "example.com/lc6", Dir: abs, Kind: cond.Main, Src: cond.PlainSource{P: plain, Dir: abs, RelDir: d}}
				if d == "" {
					p.Listed, p.GoFiles = true, []string{"main.go"}
				}
				in.Packages = append(in.Packages, p)
				in.RelDir[p] = d
			}
			res := cond.Check(in)
			ck := judge.NewChecks([]string{"cond/1", "cond/2", "cond/3", "cond/4", "cond/5"})
			for step := 1; step <= 5; step++ {
				id := "cond/" + itoa(step)
				ck.Enter(id, "")
				switch {
				case len(res.Steps[step]) > 0:
					ck.Fail(id, res.Steps[step]...)
				case !res.Evaluated[step] || len(walk) > 0 || len(res.Read) > 0:
					ck.Unevaluated(id, "inputs incomplete")
				default:
					ck.Hold(id)
				}
			}
			inst := instanceOf(ck)
			inst.Extra["steps"], inst.Extra["read_failures"], inst.Extra["walk_failures"], inst.Extra["contexts"] = res.Steps, res.Read, walk, res.Contexts
			return Outcome{Entered: Entered(ck), Lines: CheckLines(ck, normalizer("", fx.dir)), Instance: inst, Terminal: true, CleanupErr: fx.cleanup()}
		},
	}
}

// LC6 are the conditional difference controls (plan §5.2 LC-6).
func LC6() []Variant {
	healthy := []string{"cond/1 hold", "cond/2 hold", "cond/3 hold", "cond/4 hold", "cond/5 hold"}
	return []Variant{
		lc6("healthy", nil, healthy),
		lc6("!nowebui file with exec", map[string]string{
			"extra.go": "//go:build !nowebui\n\npackage main\n\nimport \"os/exec\"\n\nfunc init() { _ = exec.Command(\"x\").Run() }\n",
		}, []string{"cond/1 hold", "cond/2 fail J-3 extra.go production-only file not in the exception list real @ step 2 extra.go", "cond/3 hold", "cond/4 hold", "cond/5 hold"}),
		lc6("cgo file", map[string]string{
			"cgo.go": "//go:build cgo\n\npackage main\n",
		}, []string{"cond/1 hold", "cond/2 fail J-3 cgo.go production-only file not in the exception list real @ step 2 cgo.go", "cond/3 hold", "cond/4 hold", "cond/5 hold"}),
	}
}
