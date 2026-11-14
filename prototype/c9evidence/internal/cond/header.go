package cond

import (
	"bytes"
	"errors"
)

var (
	slashSlash   = []byte("//")
	slashStar    = []byte("/*")
	starSlash    = []byte("*/")
	goBuildBytes = []byte("//go:build")
)

// ErrMultipleGoBuild is go/build's errMultipleGoBuild.
var ErrMultipleGoBuild = errors.New("multiple //go:build comments")

// Constraint returns a file's //go:build line by go/build's header rules
// (go1.27.3 go/build/build.go:1627 parseFileHeader, :1539
// isGoBuildComment): the line as written, with only the trailing LF and CR
// removed (method record I-7). found is false when the header has none;
// two //go:build lines are an error, as in go/build (:1536).
func Constraint(content []byte) (line string, found bool, err error) {
	p := content
	ended := false
	inSlashStar := false
Lines:
	for len(p) > 0 {
		raw := p
		if i := bytes.IndexByte(raw, '\n'); i >= 0 {
			raw, p = raw[:i], p[i+1:]
		} else {
			p = p[len(p):]
		}
		ln := bytes.TrimSpace(raw)
		if len(ln) == 0 && !ended {
			continue Lines
		}
		if !bytes.HasPrefix(ln, slashSlash) {
			ended = true
		}
		if !inSlashStar && isGoBuildComment(ln) {
			if found {
				return "", false, ErrMultipleGoBuild
			}
			found = true
			line = string(bytes.TrimRight(raw, "\r\n"))
		}
	Comments:
		for len(ln) > 0 {
			if inSlashStar {
				if i := bytes.Index(ln, starSlash); i >= 0 {
					inSlashStar = false
					ln = bytes.TrimSpace(ln[i+len(starSlash):])
					continue Comments
				}
				continue Lines
			}
			if bytes.HasPrefix(ln, slashSlash) {
				continue Lines
			}
			if bytes.HasPrefix(ln, slashStar) {
				inSlashStar = true
				ln = bytes.TrimSpace(ln[len(slashStar):])
				continue Comments
			}
			break Lines
		}
	}
	return line, found, nil
}

func isGoBuildComment(line []byte) bool {
	if !bytes.HasPrefix(line, goBuildBytes) {
		return false
	}
	line = bytes.TrimSpace(line)
	rest := line[len(goBuildBytes):]
	return len(rest) == 0 || len(bytes.TrimSpace(rest)) < len(rest)
}
