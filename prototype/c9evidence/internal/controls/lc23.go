package controls

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/modsnap"
)

// The LC-2 fixture module and the h1 values computed independently of the
// tool (by the definition of Hash1, outside Go; see the implementation
// record): the go.sum record of the base content, and what I-4 must compute
// for each changed content.
var lc2Base = map[string]string{
	"go.mod":      "module example.com/lc2\n\ngo 1.22\n",
	"lc2.go":      "package lc2\n\nconst N = 1\n",
	"sub/doc.txt": "fixture\n",
}

const (
	lc2GoSum     = "h1:u344DAM58nzYOcttZ71LwT+pUl49rr53lCfGGtvTgAg="
	lc2ByteH1    = "h1:+EU9n9LZzOsUIzm4dQWa8SRl1fiD5u5koBlWIuQXXAo="
	lc2ExtraH1   = "h1:OiwaT/VEHVsvJ7RXTWCPr4q3mtjK0CeJ429NR5mc9DA="
	lc3GoSum     = "h1:4vbBQsx2OMuNRl1nzs6wGGckxEJzVw39cMAfIIM+z5E="
	lcModVersion = "v1.0.0"
)

func copyFiles(m map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

// snapshotInstance runs I-4 as a tested instance with the check B4.
func snapshotInstance(dir, mod, wantH1 string) (*judge.Checks, *modsnap.Snapshot) {
	ck := judge.NewChecks([]string{"B4"})
	ck.Enter("B4", "")
	s := modsnap.Take(modsnap.OS{}, mod, lcModVersion, dir, wantH1)
	var acc judge.Acc
	for _, f := range s.Failures {
		acc.Add(f)
	}
	ck.Set("B4", acc.Result())
	return ck, s
}

func h1Line(s *modsnap.Snapshot) string {
	if s.H1 == "" {
		return "h1 not computed"
	}
	return "h1 computed " + s.H1
}

func lc2(name string, mutate func(dir string) error, want []string) Variant {
	return Variant{
		Control: "LC-2", Name: name, Applies: always, MustEnter: []string{"B4"}, Want: want,
		Run: func(e *Env) Outcome {
			fx, err := prepare(e, "LC-2 "+name, lc2Base)
			if err == nil && mutate != nil {
				err = change(e, "prepare LC-2 "+name+" change", func() error { return mutate(fx.dir) })
			}
			if err != nil {
				var cerr error
				if fx != nil {
					cerr = fx.cleanup()
				}
				return Outcome{PrepErr: err, CleanupErr: cerr}
			}
			ck, s := snapshotInstance(fx.dir, "example.com/lc2", lc2GoSum)
			inst := instanceOf(ck)
			inst.Extra["files"], inst.Extra["h1"], inst.Extra["go.sum h1"] = s.Files, s.H1, lc2GoSum
			lines := append(CheckLines(ck, normalizer("", fx.dir)), h1Line(s))
			return Outcome{Entered: Entered(ck), Lines: lines, Instance: inst, Terminal: true, CleanupErr: fx.cleanup()}
		},
	}
}

// LC2 are the negative controls of Hash1 and I-4 (plan §5.2 LC-2).
func LC2() []Variant {
	return []Variant{
		lc2("one byte changed", func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "lc2.go"), []byte("package lc2\n\nconst N = 2\n"), 0o644)
		}, []string{"B4 fail J-4 . h1 differs from go.sum real @ I-4 example.com/lc2@v1.0.0", "h1 computed " + lc2ByteH1}),
		lc2("one more file", func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("x\n"), 0o644)
		}, []string{"B4 fail J-4 . h1 differs from go.sum real @ I-4 example.com/lc2@v1.0.0", "h1 computed " + lc2ExtraH1}),
		// Applicable on all three runners (plan §5.6 item 4): a link that
		// cannot be created is a failed preparation, so a miss, never "not
		// applicable" after the fact.
		lc2("symbolic link", func(dir string) error {
			return os.Symlink("lc2.go", filepath.Join(dir, "link.go"))
		}, []string{"B4 fail J-2 link.go not a regular file real @ I-4 example.com/lc2@v1.0.0", "h1 not computed"}),
	}
}

var lc3Base = map[string]string{
	"go.mod": "module example.com/lc3\n\ngo 1.22\n",
	"lc3.go": "package lc3\n\nconst N = 3\n",
}

// lc3 runs I-4 on the fixture (it must hold), applies the change, and then
// one controlled read as the tested instance with the check I-5.
func lc3(name, rel string, modify func(dir string) error, newInject func() modsnap.Injector, want []string) Variant {
	return Variant{
		Control: "LC-3", Name: name, Applies: always, MustEnter: []string{"I-5"}, Want: want,
		Run: func(e *Env) Outcome {
			fx, err := prepare(e, "LC-3 "+name, lc3Base)
			if err != nil {
				return Outcome{PrepErr: err}
			}
			s := modsnap.Take(modsnap.OS{}, "example.com/lc3", lcModVersion, fx.dir, lc3GoSum)
			if !s.Holds() {
				return Outcome{PrepErr: errors.New("the LC-3 fixture's I-4 does not hold"), CleanupErr: fx.cleanup()}
			}
			if modify != nil {
				if err := change(e, "prepare LC-3 "+name+" change", func() error { return modify(fx.dir) }); err != nil {
					return Outcome{PrepErr: err, CleanupErr: fx.cleanup()}
				}
			}
			var inject modsnap.Injector
			if newInject != nil {
				inject = newInject()
			}
			r := modsnap.NewReader(modsnap.OS{}, []*modsnap.Snapshot{s}, inject)
			ck := judge.NewChecks([]string{"I-5"})
			ck.Enter("I-5", "")
			_, rerr := r.Read(s, rel, modsnap.GroupP, "parse")
			var acc judge.Acc
			for _, f := range r.Failures() {
				acc.Add(f)
			}
			if rerr == nil && acc.Failed() {
				acc.Add(judge.Real(judge.J3, rel, "LC-3", "read returned bytes despite failures", ""))
			}
			ck.Set("I-5", acc.Result())
			lines := CheckLines(ck, normalizer("", fx.dir))
			for _, in := range r.Injections() {
				lines = append(lines, "injected kept apart #"+itoa(in.ID))
			}
			inst := instanceOf(ck)
			inst.Extra["reader_failures"], inst.Extra["injections"], inst.Extra["consumed"] = r.Failures(), r.Injections(), r.Consumed()
			return Outcome{Entered: Entered(ck), Lines: lines, Instance: inst, Terminal: true, CleanupErr: fx.cleanup()}
		},
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// LC3 are the controlled-read controls (plan §5.2 LC-3).
func LC3() []Variant {
	return []Variant{
		lc3("name not in M", "missing.go", nil, nil, []string{"I-5 fail J-4 missing.go name not in M real @ I-5 example.com/lc3@v1.0.0/missing.go (P parse)"}),
		lc3("reread differs", "lc3.go", func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "lc3.go"), []byte("package lc3\n\nconst N = 4\n"), 0o644)
		}, nil, []string{"I-5 fail J-4 lc3.go sha256 differs from M real @ I-5 example.com/lc3@v1.0.0/lc3.go (P parse)"}),
		lc3("handle not a regular file", "lc3.go", func(dir string) error {
			p := filepath.Join(dir, "lc3.go")
			if err := os.Remove(p); err != nil {
				return err
			}
			return os.Mkdir(p, 0o755)
		}, nil, []string{"I-5 fail J-2 lc3.go handle not a regular file real @ I-5 example.com/lc3@v1.0.0/lc3.go (P parse)"}),
		// The injection is only appended after the real Close returned nil;
		// the real result stays nil (no real failure line) and the injected
		// one is kept apart with its serial (V-29).
		lc3("injection after a real nil close", "lc3.go", nil, oneInjection,
			[]string{"I-5 fail J-1 lc3.go close error injected#1 @ I-5 example.com/lc3@v1.0.0/lc3.go (P parse)", "injected kept apart #1"}),
	}
}

func oneInjection() modsnap.Injector {
	n := 0
	return func(string) (int, error) {
		n++
		if n > 1 {
			return 0, nil
		}
		return n, errors.New("injected close error")
	}
}
