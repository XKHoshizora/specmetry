package typecheck

import (
	"bytes"
	"errors"
	"go/ast"
	"go/scanner"
	"go/token"
)

// ErrMultipleGoBuild is go/build's error for a file header with more than
// one //go:build line (go/build/build.go:1536).
var ErrMultipleGoBuild = errors.New("multiple //go:build comments")

var (
	slashSlash = []byte("//")
	slashStar  = []byte("/*")
	starSlash  = []byte("*/")
	goBuildPfx = []byte("//go:build")
)

// ConstraintLine derives a file's I-7 constraint line (method record §2.4):
// the //go:build line that go/build's file-header rules recognise
// (go/build/build.go:1627 parseFileHeader in Go 1.27.3), as the line's own
// bytes with the line end's LF and CR removed. It returns "" when the file
// has none, and ErrMultipleGoBuild when the header has more than one.
//
// The header rules, reimplemented here from that source: lines are split at
// LF and trimmed of white space; blank lines are skipped; a trimmed line
// that starts with "//go:build" followed by the end or white space is a
// //go:build line unless it is inside a /* */ comment; the header ends at the
// first text that is not a comment (a /* */ comment may span lines, and
// text after its end on the same line ends the header too).
//
// "Its own bytes" means the raw line: leading or trailing spaces and tabs
// are kept (go/build's own value is the trimmed line); only the LF the line
// was split at and trailing CRs are removed.
func ConstraintLine(src []byte) (string, error) {
	var found []byte
	have := false
	inSlashStar := false
	p := src
lines:
	for len(p) > 0 {
		raw := p
		if i := bytes.IndexByte(raw, '\n'); i >= 0 {
			raw, p = raw[:i], p[i+1:]
		} else {
			p = p[len(p):]
		}
		line := bytes.TrimSpace(raw)
		if len(line) == 0 {
			continue
		}
		if !inSlashStar && isGoBuildLine(line) {
			if have {
				return "", ErrMultipleGoBuild
			}
			found, have = raw, true
		}
		for len(line) > 0 {
			if inSlashStar {
				i := bytes.Index(line, starSlash)
				if i < 0 {
					continue lines
				}
				inSlashStar = false
				line = bytes.TrimSpace(line[i+len(starSlash):])
				continue
			}
			if bytes.HasPrefix(line, slashSlash) {
				continue lines
			}
			if bytes.HasPrefix(line, slashStar) {
				inSlashStar = true
				line = bytes.TrimSpace(line[len(slashStar):])
				continue
			}
			break lines // text that is not a comment: the header is over
		}
	}
	if !have {
		return "", nil
	}
	return string(bytes.TrimRight(found, "\r")), nil
}

// isGoBuildLine is go/build's isGoBuildComment on a trimmed line.
func isGoBuildLine(line []byte) bool {
	if !bytes.HasPrefix(line, goBuildPfx) {
		return false
	}
	rest := line[len(goBuildPfx):]
	return len(rest) == 0 || len(bytes.TrimSpace(rest)) < len(rest)
}

// prefilterHit is the lexical prefilter of §12.10: whether any identifier
// token of the file (go/scanner, comments not returned, string literals are
// STRING tokens) is the name of a table object. A scanning error is not
// handled here: the same bytes are parsed next and every syntax error fails
// the package.
func prefilterHit(src []byte, names map[string]bool) bool {
	fset := token.NewFileSet()
	file := fset.AddFile("", -1, len(src))
	var s scanner.Scanner
	s.Init(file, src, nil, 0)
	for {
		_, tok, lit := s.Scan()
		switch tok {
		case token.EOF:
			return false
		case token.IDENT:
			if names[lit] {
				return true
			}
		}
	}
}

// StripBodies empties the statement list of every function declaration's
// and every function literal's body (Body.List = nil), keeping the Body
// block with its braces (§12.10 loading step 9, N-127-2): go/types takes a
// function literal's end position from its body before it looks at
// IgnoreFuncBodies, and a generic function without a body is a soft error.
func StripBodies(f *ast.File) {
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			if n.Body != nil {
				n.Body.List = nil
			}
		case *ast.FuncLit:
			if n.Body != nil {
				n.Body.List = nil
			}
		}
		return true
	})
}
