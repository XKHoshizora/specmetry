package typecheck

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// The expectations follow go/build's file-header rules (go/build/build.go
// parseFileHeader, Go 1.27.3), read case by case; the value is the raw line
// without the line end's LF and CR (method record §2.4 I-7).
func TestConstraintLine(t *testing.T) {
	for _, tc := range []struct {
		name, src, want string
		err             error
	}{
		{"none", "package x\n", "", nil},
		{"one", "//go:build linux\n\npackage x\n", "//go:build linux", nil},
		{"crlf", "//go:build linux && amd64\r\n\r\npackage x\r\n", "//go:build linux && amd64", nil},
		{"two", "//go:build linux\n//go:build amd64\n\npackage x\n", "", ErrMultipleGoBuild},
		{"two separated", "//go:build linux\n\n// doc\n//go:build amd64\n\npackage x\n", "", ErrMultipleGoBuild},
		{"after copyright", "// Copyright 2026\n\n//go:build linux\n\n// Package x.\npackage x\n", "//go:build linux", nil},
		{"no blank line before package", "//go:build linux\npackage x\n", "//go:build linux", nil},
		{"no final newline", "//go:build linux", "//go:build linux", nil},
		{"tab separated", "//go:build\tlinux\n\npackage x\n", "//go:build\tlinux", nil},
		{"raw bytes kept", "  //go:build linux  \n\npackage x\n", "  //go:build linux  ", nil},
		{"not a go:build line", "//go:buildlinux\n\npackage x\n", "", nil},
		{"inside block comment", "/*\n//go:build linux\n*/\n\npackage x\n", "", nil},
		{"second inside block comment", "//go:build linux\n/*\n//go:build amd64\n*/\n\npackage x\n", "//go:build linux", nil},
		{"after closed block comment", "/* a */\n//go:build linux\n\npackage x\n", "//go:build linux", nil},
		{"after block comment on the line", "/* c */ //go:build linux\n\npackage x\n", "", nil},
		{"after the header", "package x\n\n//go:build linux\n", "", nil},
		{"second after the header", "//go:build linux\n\npackage x\n\n//go:build ignore\n", "//go:build linux", nil},
		{"text after block comment ends header", "/* a */ var\n//go:build linux\n", "", nil},
		{"plus build is not go:build", "// +build linux\n\npackage x\n", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ConstraintLine([]byte(tc.src))
			if !errors.Is(err, tc.err) || got != tc.want {
				t.Errorf("ConstraintLine = %q, %v; want %q, %v", got, err, tc.want, tc.err)
			}
		})
	}
}

// I-7 in the scan: the constraint line of the file, and positions that a
// //line directive does not move (PositionFor(p, false), byte columns).
const linedSrc = `//go:build linux

package lined

import "os/exec"

//line generated.go:100:1
func Run(c *exec.Cmd) error {
	return c.Run()
}
`

func TestLinePositionsAndConstraint(t *testing.T) {
	const p = "example.com/lined"
	two := thirdParty("example.com/two", "1.22", nil, map[string]string{"two.go": "//go:build linux\n//go:build amd64\n\npackage two\n\nvar X = 1\n"})
	w := build(append(stdStubs("linux"), thirdParty(p, "1.22", []string{"os/exec"}, map[string]string{"lined.go": linedSrc}), two)...)
	rep := Run(w.input(linuxAMD64))
	want := []Ref{{Basis: spec.Frozen, Package: p, File: "lined.go", Caller: p + ".Run", Object: "(*os/exec.Cmd).Run",
		Line: 9, Column: 11, Constraint: "//go:build linux", FileSHA256: sum(linedSrc)}}
	if !reflect.DeepEqual(rep.Findings, want) {
		t.Errorf("findings:\n%s\nwant\n%s", refsString(rep.Findings), refsString(want))
	}
	// The directive is in effect: the adjusted position of the same
	// identifier is in generated.go. So the physical position above is not
	// an accident of an ignored directive.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "lined.go", linedSrc, 0)
	if err != nil {
		t.Fatal(err)
	}
	var sel *ast.SelectorExpr
	ast.Inspect(f, func(n ast.Node) bool {
		if s, ok := n.(*ast.SelectorExpr); ok && s.Sel.Name == "Run" {
			sel = s
		}
		return true
	})
	if adj := fset.PositionFor(sel.Sel.Pos(), true); adj.Filename != "generated.go" || adj.Line != 101 {
		t.Errorf("adjusted position %v; the //line directive is not in effect", adj)
	}

	tr := pkgOf(t, rep, "example.com/two")
	wantFail(t, "two I-7", tr.Header, "J-3 "+KindMultipleGoBuild+" example.com/two/two.go")
	wantHold(t, "two load", tr.Load)
	if tr.Files[0].Constraint != "" {
		t.Errorf("constraint %q after a failed derivation", tr.Files[0].Constraint)
	}
}

func TestPrefilterHit(t *testing.T) {
	names := spec.PrefilterNames()
	for _, tc := range []struct {
		src  string
		want bool
	}{
		{"package p\n\nfunc f(c interface{ Run() error }) { c.Run() }\n", true},
		{"package p\n\nvar x = y.CreateProcess\n", true},   // an X1 name counts
		{"package p\n\ntype T struct{ Wait int }\n", true}, // a token, not an object: over-approximation
		{"package p\n\n// Run Start Wait\n", false},
		{"package p\n\n/* Notify */ var x = 1\n", false},
		{"package p\n\nvar s = \"StartProcess\"\n", false},
		{"package p\n\nvar s = `Exec Wait4`\n", false},
		{"package p\n\nvar s = 'R'\n\nfunc Running() {}\n", false},
	} {
		if got := prefilterHit([]byte(tc.src), names); got != tc.want {
			t.Errorf("prefilterHit(%q) = %v, want %v", tc.src, got, tc.want)
		}
	}
}

const stripSrc = `package s

type Pool struct{ New func() any }

var P = Pool{New: func() any { return new(int) }}

func Map[T any](x T) T { return x }

func outer() func() int {
	g := func() int { return 1 }
	return g
}
`

// StripBodies keeps every Body block and its braces and empties the lists.
func TestStripBodies(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "s.go", stripSrc, 0)
	if err != nil {
		t.Fatal(err)
	}
	StripBodies(f)
	var decls, lits int
	ast.Inspect(f, func(n ast.Node) bool {
		var body *ast.BlockStmt
		switch n := n.(type) {
		case *ast.FuncDecl:
			decls++
			body = n.Body
		case *ast.FuncLit:
			lits++
			body = n.Body
		default:
			return true
		}
		if body == nil || body.List != nil || !body.Lbrace.IsValid() || !body.Rbrace.IsValid() {
			t.Errorf("body after stripping: %+v", body)
		}
		return true
	})
	// The literal inside outer's body is gone with the body.
	if decls != 2 || lits != 1 {
		t.Errorf("%d declarations, %d literals after stripping", decls, lits)
	}
	conf := types.Config{IgnoreFuncBodies: true, Error: func(err error) { t.Errorf("type error: %v", err) }}
	if _, err := conf.Check("s", fset, []*ast.File{f}, nil); err != nil {
		t.Errorf("check: %v", err)
	}
}

// The counter-examples of N-127-2, built directly with go/types: a generic
// function whose Body is nil is a soft error, and a function literal whose
// Body is nil makes the check panic when it takes the end position.
func TestStripBodiesCounterExamples(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "s.go", stripSrc, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "Map" {
			fd.Body = nil
		}
	}
	var soft []string
	conf := types.Config{IgnoreFuncBodies: true, Error: func(err error) {
		var te types.Error
		if errors.As(err, &te) && te.Soft {
			soft = append(soft, te.Msg)
		}
	}}
	_, _ = conf.Check("s", fset, []*ast.File{f}, nil)
	if len(soft) != 1 || !strings.Contains(soft[0], "missing function body") {
		t.Errorf("soft errors %q", soft)
	}

	f2, err := parser.ParseFile(fset, "s2.go", stripSrc, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(f2, func(n ast.Node) bool {
		if l, ok := n.(*ast.FuncLit); ok {
			l.Body = nil
			return false
		}
		return true
	})
	panicked := func() (p bool) {
		defer func() { p = recover() != nil }()
		_, _ = (&types.Config{IgnoreFuncBodies: true, Error: func(error) {}}).Check("s", fset, []*ast.File{f2}, nil)
		return false
	}()
	if !panicked {
		t.Errorf("a nil FuncLit.Body did not panic")
	}
}
