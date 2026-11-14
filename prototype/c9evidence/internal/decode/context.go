package decode

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// Context is the decoded "context text v1" of the second call (method
// record §3.3.2): the eleven fields of go list's Context
// (cmd/go/internal/list/context.go:11-23) as the template T2 writes them.
// Every field has a line, so every value is obtained: "" , false and a
// zero count are values, not omissions. The lists keep the output order
// and duplicates and are never nil (an empty list is []string{}).
type Context struct {
	GOARCH        string
	GOOS          string
	GOROOT        string
	GOPATH        string
	CgoEnabled    bool
	UseAllFiles   bool
	Compiler      string
	InstallSuffix string
	BuildTags     []string
	ToolTags      []string
	ReleaseTags   []string
}

// contextHeader is the first line of the text.
const contextHeader = "c9-context v1"

// textReader walks the context text one LF-terminated line at a time.
type textReader struct {
	data []byte
	pos  int // offset of the next line
	line int // 1-based number of the next line
}

// peek returns the next line without its LF, after the checks every line
// has: it is terminated by a LF (else KindForm, which also covers an output
// that ends early), it has no CR and it is not blank (else KindLine). At
// the end of the output it returns eof.
func (r *textReader) peek() (line string, eof bool, err *Error) {
	if r.pos >= len(r.data) {
		return "", true, nil
	}
	at := fmt.Sprintf("line %d", r.line)
	i := bytes.IndexByte(r.data[r.pos:], '\n')
	if i < 0 {
		return "", false, errorf(spec.CallContext, KindForm, at, r.pos, "line is not terminated by a LF (the text ends before its end line)")
	}
	s := string(r.data[r.pos : r.pos+i])
	if j := strings.IndexByte(s, '\r'); j >= 0 {
		return "", false, errorf(spec.CallContext, KindLine, at, r.pos+j, "line contains a CR")
	}
	if s == "" {
		return "", false, errorf(spec.CallContext, KindLine, at, r.pos, "blank line")
	}
	return s, false, nil
}

func (r *textReader) advance(line string) {
	r.pos += len(line) + 1
	r.line++
}

func (r *textReader) at() string { return fmt.Sprintf("line %d", r.line) }

// next returns the next line, which must exist.
func (r *textReader) next(what string) (string, *Error) {
	s, eof, err := r.peek()
	if err != nil {
		return "", err
	}
	if eof {
		return "", errorf(spec.CallContext, KindForm, r.at(), r.pos, "output ends before the %s line", what)
	}
	r.advance(s)
	return s, nil
}

// field reads a "<name> <value>" line and returns the value: the line must
// begin with the name and one space (else KindLine), and the value must be
// non-empty and have no leading or trailing space or TAB (else KindLine).
func (r *textReader) field(name string) (string, string, int, *Error) {
	at, off := r.at(), r.pos
	s, err := r.next(fmt.Sprintf("%q", name))
	if err != nil {
		return "", at, off, err
	}
	v, ok := strings.CutPrefix(s, name+" ")
	if !ok {
		return "", at, off, errorf(spec.CallContext, KindLine, at, off, "line %q, want the %q line", s, name)
	}
	if blankEdge(v) {
		return "", at, off, errorf(spec.CallContext, KindLine, at, off, "%q line: want exactly one space between name and value and no other blank", name)
	}
	return v, at, off + len(name) + 1, nil
}

// blankEdge reports whether a value is empty or begins or ends with a space
// or TAB, which the separator rule of §3.3.2 excludes.
func blankEdge(v string) bool {
	if v == "" {
		return true
	}
	first, last := v[0], v[len(v)-1]
	return first == ' ' || first == '\t' || last == ' ' || last == '\t'
}

// quoted decodes a <Q>: a Go interpreted string literal (KindValue
// otherwise) that is byte for byte strconv.Quote of the string it denotes
// (KindCanonical otherwise). The analyser's strconv is go1.27.3, the same
// as the tool's (A1 guard), so IsPrint agrees.
func quoted(v, at string, off int) (string, *Error) {
	if v[0] != '"' {
		return "", errorf(spec.CallContext, KindValue, at, off, "value %s is not a double-quoted string literal", v)
	}
	s, err := strconv.Unquote(v)
	if err != nil {
		return "", errorf(spec.CallContext, KindValue, at, off, "value %s is not a valid Go interpreted string literal", v)
	}
	if q := strconv.Quote(s); q != v {
		return "", errorf(spec.CallContext, KindCanonical, at, off, "value %s is not the strconv.Quote form %s", v, q)
	}
	return s, nil
}

// boolean decodes a <B>: exactly true or false.
func boolean(v, at string, off int) (bool, *Error) {
	switch v {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, errorf(spec.CallContext, KindValue, at, off, "value %q, want true or false", v)
}

// count decodes an <N>: "0" or [1-9][0-9]*, within int.
func count(v, at string, off int) (int, *Error) {
	ok := v == "0" || (v[0] >= '1' && v[0] <= '9')
	for i := 0; ok && i < len(v); i++ {
		ok = v[i] >= '0' && v[i] <= '9'
	}
	n, err := strconv.Atoi(v)
	if !ok || err != nil {
		return 0, errorf(spec.CallContext, KindValue, at, off, "value %q, want 0 or [1-9][0-9]*", v)
	}
	return n, nil
}

// DecodeContext decodes the output of the second call, go list -e -f <T2>
// unsafe, as "context text v1" (method record §3.3.2). It applies the
// syntax only; the requirements of §3.3.3 (GOOS equal to the
// configuration, CgoEnabled false, non-empty ToolTags, ...) are the
// caller's (stage B2).
//
// The lines are read in the fixed order and the first problem is returned:
//
//   - The output is not empty (else KindCount).
//   - Every line read is LF-terminated (an output that ends early, with or
//     without a final LF, is KindForm), has no CR and is not blank
//     (KindLine).
//   - The first line is "c9-context v1"; then GOARCH, GOOS, GOROOT, GOPATH
//     (<Q>), CgoEnabled, UseAllFiles (<B>), Compiler, InstallSuffix (<Q>),
//     each as name, one space, value (a wrong line is KindLine).
//   - Each list header BuildTags, ToolTags, ReleaseTags is name, one space,
//     <N>, followed by exactly N lines "- <Q>": a line within the N that
//     does not begin with "-", or a line beginning with "-" after the N, is
//     KindCount; a line beginning with "-" but not "- ", or whose string
//     has a leading or trailing space or TAB, is KindLine.
//   - Then "end". After its LF, nothing: a second text (bytes beginning
//     with "c9-context v1") is KindCount, anything else KindTrailing.
//   - <Q>, <B>, <N> that are not of their form are KindValue; a <Q> that is
//     a valid literal but not strconv.Quote's form is KindCanonical.
func DecodeContext(out []byte) (Context, error) {
	const call = spec.CallContext
	if len(out) == 0 {
		return Context{}, errorf(call, KindCount, "line 1", 0, "zero-byte output, want one context text")
	}
	r := &textReader{data: out, line: 1}
	at, off := r.at(), r.pos
	h, err := r.next("header")
	if err != nil {
		return Context{}, err
	}
	if h != contextHeader {
		return Context{}, errorf(call, KindLine, at, off, "line %q, want %q", h, contextHeader)
	}

	var c Context
	for _, f := range []struct {
		name string
		str  *string
		b    *bool
	}{
		{"GOARCH", &c.GOARCH, nil},
		{"GOOS", &c.GOOS, nil},
		{"GOROOT", &c.GOROOT, nil},
		{"GOPATH", &c.GOPATH, nil},
		{"CgoEnabled", nil, &c.CgoEnabled},
		{"UseAllFiles", nil, &c.UseAllFiles},
		{"Compiler", &c.Compiler, nil},
		{"InstallSuffix", &c.InstallSuffix, nil},
	} {
		v, at, voff, err := r.field(f.name)
		if err != nil {
			return Context{}, err
		}
		if f.str != nil {
			s, err := quoted(v, at, voff)
			if err != nil {
				return Context{}, err
			}
			*f.str = s
		} else {
			b, err := boolean(v, at, voff)
			if err != nil {
				return Context{}, err
			}
			*f.b = b
		}
	}

	for _, l := range []struct {
		name string
		dst  *[]string
	}{
		{"BuildTags", &c.BuildTags},
		{"ToolTags", &c.ToolTags},
		{"ReleaseTags", &c.ReleaseTags},
	} {
		v, at, voff, err := r.field(l.name)
		if err != nil {
			return Context{}, err
		}
		n, err := count(v, at, voff)
		if err != nil {
			return Context{}, err
		}
		list := []string{}
		for i := 0; i < n; i++ {
			at, off := r.at(), r.pos
			s, err := r.next(fmt.Sprintf("%s element %d of %d", l.name, i+1, n))
			if err != nil {
				return Context{}, err
			}
			if !strings.HasPrefix(s, "-") {
				return Context{}, errorf(call, KindCount, at, off, "%s declares %d elements, found %d", l.name, n, i)
			}
			e, ok := strings.CutPrefix(s, "- ")
			if !ok || blankEdge(e) {
				return Context{}, errorf(call, KindLine, at, off, "element line %q, want \"- \" and a string", s)
			}
			q, err := quoted(e, at, off+2)
			if err != nil {
				return Context{}, err
			}
			list = append(list, q)
		}
		s, eof, err := r.peek()
		if err != nil {
			return Context{}, err
		}
		if !eof && strings.HasPrefix(s, "-") {
			return Context{}, errorf(call, KindCount, r.at(), r.pos, "%s declares %d elements, found more", l.name, n)
		}
		*l.dst = list
	}

	at, off = r.at(), r.pos
	e, err := r.next("end")
	if err != nil {
		return Context{}, err
	}
	if e != "end" {
		return Context{}, errorf(call, KindLine, at, off, "line %q, want \"end\"", e)
	}
	if rest := out[r.pos:]; len(rest) > 0 {
		if bytes.HasPrefix(rest, []byte(contextHeader)) {
			return Context{}, errorf(call, KindCount, r.at(), r.pos, "a second context text follows the end line")
		}
		return Context{}, errorf(call, KindTrailing, r.at(), r.pos, "%d bytes after the end line", len(rest))
	}
	return c, nil
}
