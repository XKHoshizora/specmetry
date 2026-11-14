package decode

import (
	"fmt"
	"unicode/utf16"
	"unicode/utf8"
)

// This file is a small strict JSON parser (RFC 8259). It is used instead of
// encoding/json because the method record §3.1 step 2 needs what
// encoding/json hides: it matches keys without regard to case, keeps the
// last of duplicate keys, and turns null into a zero value. The parser
// builds a tree that keeps every member in output order with its raw key
// bytes, so the later walks see duplicates, case variants, escaped keys and
// nulls.
//
// Beyond RFC 8259 grammar it rejects, as syntax: bytes that are not valid
// UTF-8 inside or outside strings, and \u escapes that encode a lone UTF-16
// surrogate. The fixed encoder (encoding/json of go1.27.3) never writes
// either: it coerces invalid UTF-8 to U+FFFD and writes non-ASCII runes as
// UTF-8.

// maxDepth bounds the nesting of arrays and objects. The deepest values the
// fixed definitions produce in practice are four levels deep (record,
// Module, Replace, Origin; record, DepsErrors, element, ImportStack);
// deeper input is rejected as syntax rather than recursed into without
// bound.
const maxDepth = 64

type nodeKind uint8

const (
	nObject nodeKind = iota + 1
	nArray
	nString
	nNumber
	nBool
	nNull
)

func (k nodeKind) String() string {
	switch k {
	case nObject:
		return "object"
	case nArray:
		return "array"
	case nString:
		return "string"
	case nNumber:
		return "number"
	case nBool:
		return "bool"
	case nNull:
		return "null"
	}
	return "invalid"
}

// member is one key and value of an object, in output order.
type member struct {
	raw string // the key's bytes between the quotes, not unescaped
	key string // the key unescaped
	off int    // offset of the key's opening quote
	val *node
}

// node is one parsed JSON value.
type node struct {
	kind    nodeKind
	off     int // offset of the value's first byte
	str     string
	b       bool
	members []member
	elems   []*node
}

type syntaxError struct {
	off int
	msg string
}

type parser struct {
	data  []byte
	pos   int
	depth int
}

func (p *parser) fail(format string, args ...any) *syntaxError {
	return &syntaxError{off: p.pos, msg: fmt.Sprintf(format, args...)}
}

func isWS(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

func (p *parser) skipWS() {
	for p.pos < len(p.data) && isWS(p.data[p.pos]) {
		p.pos++
	}
}

// value parses one value at p.pos (after optional whitespace).
func (p *parser) value() (*node, *syntaxError) {
	p.skipWS()
	if p.pos >= len(p.data) {
		return nil, p.fail("unexpected end of output, want a value")
	}
	switch c := p.data[p.pos]; {
	case c == '{':
		return p.object()
	case c == '[':
		return p.array()
	case c == '"':
		off := p.pos
		s, _, err := p.str()
		if err != nil {
			return nil, err
		}
		return &node{kind: nString, off: off, str: s}, nil
	case c == 't':
		return p.literal("true", &node{kind: nBool, off: p.pos, b: true})
	case c == 'f':
		return p.literal("false", &node{kind: nBool, off: p.pos})
	case c == 'n':
		return p.literal("null", &node{kind: nNull, off: p.pos})
	case c == '-' || (c >= '0' && c <= '9'):
		return p.number()
	default:
		return nil, p.fail("unexpected byte %q, want a value", c)
	}
}

func (p *parser) literal(word string, n *node) (*node, *syntaxError) {
	if len(p.data)-p.pos < len(word) || string(p.data[p.pos:p.pos+len(word)]) != word {
		return nil, p.fail("invalid literal, want %s", word)
	}
	p.pos += len(word)
	return n, nil
}

// number accepts -?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?.
func (p *parser) number() (*node, *syntaxError) {
	start := p.pos
	digits := func() int {
		n := 0
		for p.pos < len(p.data) && p.data[p.pos] >= '0' && p.data[p.pos] <= '9' {
			p.pos++
			n++
		}
		return n
	}
	if p.data[p.pos] == '-' {
		p.pos++
	}
	if p.pos >= len(p.data) {
		return nil, p.fail("invalid number")
	}
	if p.data[p.pos] == '0' {
		p.pos++
	} else if digits() == 0 {
		return nil, p.fail("invalid number")
	}
	if p.pos < len(p.data) && p.data[p.pos] == '.' {
		p.pos++
		if digits() == 0 {
			return nil, p.fail("invalid number: no digits after the decimal point")
		}
	}
	if p.pos < len(p.data) && (p.data[p.pos] == 'e' || p.data[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.data) && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
			p.pos++
		}
		if digits() == 0 {
			return nil, p.fail("invalid number: no digits in the exponent")
		}
	}
	return &node{kind: nNumber, off: start, str: string(p.data[start:p.pos])}, nil
}

func hex4(b []byte) (rune, bool) {
	var r rune
	for _, c := range b {
		r <<= 4
		switch {
		case c >= '0' && c <= '9':
			r |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			r |= rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			r |= rune(c-'A') + 10
		default:
			return 0, false
		}
	}
	return r, true
}

// str parses a string at p.pos and returns it unescaped and its raw bytes
// between the quotes.
func (p *parser) str() (string, string, *syntaxError) {
	p.pos++ // opening quote
	start := p.pos
	var out []byte
	for {
		if p.pos >= len(p.data) {
			return "", "", p.fail("unterminated string")
		}
		c := p.data[p.pos]
		switch {
		case c == '"':
			raw := string(p.data[start:p.pos])
			p.pos++
			return string(out), raw, nil
		case c < 0x20:
			return "", "", p.fail("control byte 0x%02x in string", c)
		case c == '\\':
			if p.pos+1 >= len(p.data) {
				return "", "", p.fail("unterminated escape")
			}
			e := p.data[p.pos+1]
			switch e {
			case '"', '\\', '/':
				out = append(out, e)
				p.pos += 2
			case 'b':
				out = append(out, '\b')
				p.pos += 2
			case 'f':
				out = append(out, '\f')
				p.pos += 2
			case 'n':
				out = append(out, '\n')
				p.pos += 2
			case 'r':
				out = append(out, '\r')
				p.pos += 2
			case 't':
				out = append(out, '\t')
				p.pos += 2
			case 'u':
				r, err := p.unicodeEscape()
				if err != nil {
					return "", "", err
				}
				out = utf8.AppendRune(out, r)
			default:
				return "", "", p.fail("invalid escape \\%c", e)
			}
		case c < utf8.RuneSelf:
			out = append(out, c)
			p.pos++
		default:
			r, size := utf8.DecodeRune(p.data[p.pos:])
			if r == utf8.RuneError && size == 1 {
				return "", "", p.fail("invalid UTF-8 in string")
			}
			out = append(out, p.data[p.pos:p.pos+size]...)
			p.pos += size
		}
	}
}

// unicodeEscape parses \uXXXX at p.pos, and its low surrogate when it is a
// high surrogate.
func (p *parser) unicodeEscape() (rune, *syntaxError) {
	read := func() (rune, bool) {
		if len(p.data)-p.pos < 6 || p.data[p.pos] != '\\' || p.data[p.pos+1] != 'u' {
			return 0, false
		}
		r, ok := hex4(p.data[p.pos+2 : p.pos+6])
		if !ok {
			return 0, false
		}
		p.pos += 6
		return r, true
	}
	r, ok := read()
	if !ok {
		return 0, p.fail("invalid \\u escape")
	}
	switch {
	case r >= 0xDC00 && r <= 0xDFFF:
		return 0, p.fail("lone low surrogate \\u%04x", r)
	case r >= 0xD800 && r <= 0xDBFF:
		lo, ok := read()
		if !ok || lo < 0xDC00 || lo > 0xDFFF {
			return 0, p.fail("high surrogate \\u%04x without a low surrogate", r)
		}
		return utf16.DecodeRune(r, lo), nil
	}
	return r, nil
}

func (p *parser) enter() *syntaxError {
	p.depth++
	if p.depth > maxDepth {
		return p.fail("nesting deeper than %d", maxDepth)
	}
	return nil
}

func (p *parser) object() (*node, *syntaxError) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	n := &node{kind: nObject, off: p.pos}
	p.pos++ // {
	p.skipWS()
	if p.pos < len(p.data) && p.data[p.pos] == '}' {
		p.pos++
		p.depth--
		return n, nil
	}
	for {
		p.skipWS()
		if p.pos >= len(p.data) {
			return nil, p.fail("unexpected end of output in object")
		}
		if p.data[p.pos] != '"' {
			return nil, p.fail("unexpected byte %q, want a key", p.data[p.pos])
		}
		koff := p.pos
		key, raw, err := p.str()
		if err != nil {
			return nil, err
		}
		p.skipWS()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return nil, p.fail("want ':' after key")
		}
		p.pos++
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		n.members = append(n.members, member{raw: raw, key: key, off: koff, val: v})
		p.skipWS()
		if p.pos >= len(p.data) {
			return nil, p.fail("unexpected end of output in object")
		}
		switch p.data[p.pos] {
		case ',':
			p.pos++
		case '}':
			p.pos++
			p.depth--
			return n, nil
		default:
			return nil, p.fail("unexpected byte %q in object, want ',' or '}'", p.data[p.pos])
		}
	}
}

func (p *parser) array() (*node, *syntaxError) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	n := &node{kind: nArray, off: p.pos}
	p.pos++ // [
	p.skipWS()
	if p.pos < len(p.data) && p.data[p.pos] == ']' {
		p.pos++
		p.depth--
		return n, nil
	}
	for {
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		n.elems = append(n.elems, v)
		p.skipWS()
		if p.pos >= len(p.data) {
			return nil, p.fail("unexpected end of output in array")
		}
		switch p.data[p.pos] {
		case ',':
			p.pos++
		case ']':
			p.pos++
			p.depth--
			return n, nil
		default:
			return nil, p.fail("unexpected byte %q in array, want ',' or ']'", p.data[p.pos])
		}
	}
}
