package typecheck

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// FuncName is the full name the object table uses for a function or
// method (implementation plan §12.9): types.Func.FullName of the origin
// (Origin(), so an instantiated function or method names its generic
// declaration), with aliases in the receiver expanded (types.Unalias).
//
// For a method whose receiver base is a named type the receiver is written
// "(pkgpath.T)" or "(*pkgpath.T)" without type arguments; that is exactly
// FullName for every non-generic receiver declared without an alias (all
// table objects and candidates), and the only spelling of a generic type's
// method that does not depend on the type parameters' names. Anything else
// (functions, interface methods) is FullName unchanged.
func FuncName(f *types.Func) string {
	f = f.Origin()
	sig, ok := f.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return f.FullName()
	}
	t := types.Unalias(sig.Recv().Type())
	star := ""
	if p, ok := t.(*types.Pointer); ok {
		star = "*"
		t = types.Unalias(p.Elem())
	}
	n, ok := t.(*types.Named)
	if !ok {
		return f.FullName()
	}
	obj := n.Origin().Obj()
	q := obj.Name()
	if obj.Pkg() != nil {
		q = obj.Pkg().Path() + "." + q
	}
	return "(" + star + q + ")." + f.Name()
}

// scanPackage scans one fully checked third-party package (§12.9 first
// class; limited evidence plan §4.4). Each identifier recorded in
// types.Info.Uses is one reference, so a selector counts once (its Sel) and
// a qualified identifier once (the object, not the package name). The
// object is resolved by go/types, so calls, method values, method
// expressions, function values, type aliases, dot imports and embedded
// promotion all name the declared object; a call through an interface
// names the interface's method and is not a reference to the table.
func (r *runner) scanPackage(path string, files []*ast.File, frs []FileResult, info *types.Info) {
	for i, f := range files {
		fr := frs[i]
		calls := map[*ast.Ident]*ast.CallExpr{}
		for _, decl := range f.Decls {
			caller := PackageLevel
			if fd, ok := decl.(*ast.FuncDecl); ok {
				if fn, ok := info.Defs[fd.Name].(*types.Func); ok {
					caller = FuncName(fn)
				} else {
					caller = path + "." + fd.Name.Name
				}
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CallExpr:
					// Recorded before the children are visited, so the
					// callee identifier finds its call.
					switch fun := ast.Unparen(n.Fun).(type) {
					case *ast.Ident:
						calls[fun] = n
					case *ast.SelectorExpr:
						calls[fun.Sel] = n
					}
				case *ast.Ident:
					r.reference(path, fr, caller, n, calls[n], info)
				}
				return true
			})
		}
	}
}

func (r *runner) reference(path string, fr FileResult, caller string, id *ast.Ident, call *ast.CallExpr, info *types.Info) {
	fn, ok := info.Uses[id].(*types.Func)
	if !ok {
		return
	}
	name := FuncName(fn)
	pos := r.fset.PositionFor(id.Pos(), false)
	o, ok := r.objects[name]
	if !ok {
		if _, inTable := r.all[name]; inTable {
			// The table says this object is not defined on this GOOS, yet
			// the source refers to it: the table's platform column is wrong
			// for this configuration. Fail closed instead of dropping it.
			r.scan.Add(judge.Real(judge.J3, name, SiteScan, KindOffPlatform, fmt.Sprintf(
				"%s/%s:%d:%d refers to %s, which the table does not list for %s",
				path, fr.Name, pos.Line, pos.Column, name, r.in.Config.GOOS)))
		}
		return
	}
	ref := Ref{
		Basis:      o.Basis,
		Package:    path,
		File:       fr.Name,
		Caller:     caller,
		Object:     name,
		Line:       pos.Line,
		Column:     pos.Column,
		Constraint: fr.Constraint,
		FileSHA256: fr.SHA256,
	}
	if o.Basis == spec.Frozen {
		r.rep.Findings = append(r.rep.Findings, ref)
	} else {
		r.rep.Diagnostics = append(r.rep.Diagnostics, ref)
	}
	if o.Signal {
		r.rep.Signals = append(r.rep.Signals, r.signalUse(ref, o, call, info))
	}
}

// signalUse classifies one reference to an os/signal object. Only a direct
// call has argument classes; the first argument of Notify and
// NotifyContext is the channel or the parent context, not a signal.
func (r *runner) signalUse(ref Ref, o spec.Object, call *ast.CallExpr, info *types.Info) SignalUse {
	su := SignalUse{Ref: ref}
	if call == nil {
		return su
	}
	su.Direct = true
	su.Spread = call.Ellipsis.IsValid()
	skip := 0
	if o.Name == "Notify" || o.Name == "NotifyContext" {
		skip = 1
	}
	if len(call.Args) <= skip {
		su.Classes = []string{SignalNoArgs}
		return su
	}
	for i, a := range call.Args[skip:] {
		spread := su.Spread && skip+i == len(call.Args)-1
		su.Classes = append(su.Classes, r.argClass(a, spread, info))
	}
	return su
}

func (r *runner) argClass(a ast.Expr, spread bool, info *types.Info) string {
	if spread {
		return SignalRuntime
	}
	tv, ok := info.Types[a]
	if !ok || tv.Value == nil || !isSyscallSignal(tv.Type) {
		return SignalRuntime
	}
	if want, ok := spec.SIGCHLD[r.in.Config.GOOS]; ok {
		if v := constant.ToInt(tv.Value); v.Kind() == constant.Int {
			if got, exact := constant.Int64Val(v); exact && got == want {
				return SignalConstSIGCHLD
			}
		}
	}
	return SignalOtherConst
}

func isSyscallSignal(t types.Type) bool {
	n, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := n.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "syscall" && obj.Name() == "Signal"
}
