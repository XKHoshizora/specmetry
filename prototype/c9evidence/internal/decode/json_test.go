package decode

import (
	"encoding/json"
	"strings"
	"testing"
)

func parseOne(in string) (*node, *syntaxError, int) {
	p := &parser{data: []byte(in)}
	n, err := p.value()
	return n, err, p.pos
}

// TestStringsAgreeWithEncodingJSON: for valid strings the parser's
// unescaping equals encoding/json's.
func TestStringsAgreeWithEncodingJSON(t *testing.T) {
	for _, lit := range []string{
		`""`, `"a"`, `"\"\\\/\b\f\n\r\t"`, `"\u0000\u001f\u007f"`, `"üü"`,
		`"😀"`, `"  "`, `"<>&"`, `"r` + "ü" + `nner"`,
		`"` + "\U0001F600" + `"`, `"�"`, `"tab\tescaped"`,
	} {
		n, err, _ := parseOne(lit)
		if err != nil {
			t.Errorf("%s: %s", lit, err.msg)
			continue
		}
		var want string
		if jerr := json.Unmarshal([]byte(lit), &want); jerr != nil {
			t.Fatalf("%s: encoding/json: %v", lit, jerr)
		}
		if n.kind != nString || n.str != want {
			t.Errorf("%s: got %q want %q", lit, n.str, want)
		}
	}
}

// TestSyntaxRejections: inputs that are not RFC 8259 JSON, or that the
// fixed encoder cannot write (invalid UTF-8, lone surrogates).
func TestSyntaxRejections(t *testing.T) {
	for _, in := range []string{
		``, ` `, `"`, `"abc`, `"\`, `"\x41"`, `"\u12"`, `"\u12g4"`, `"\ud800"`, `"\udc00"`, `"\ud800A"`,
		`"\ud800x"`, "\"\x01\"", "\"\x7f\xff\"", "\"\xed\xa0\x80\"", "\"\xc3\"",
		`tru`, `nul`, `fals`, `True`, `NULL`,
		`01`, `-`, `1.`, `.5`, `+1`, `1e`, `1e+`, `--1`, `0x1`,
		`{`, `{"a"}`, `{"a":}`, `{"a":1,}`, `{,}`, `{a:1}`, `{'a':1}`, `{"a":1 "b":2}`,
		`[`, `[1,]`, `[,1]`, `[1 2]`, `]`, `}`, "\xef\xbb\xbf{}",
	} {
		// One whole value: a value followed by more bytes (01, 0x1) is not.
		p := &parser{data: []byte(in)}
		n, err := p.value()
		p.skipWS()
		if err == nil && p.pos == len(in) {
			t.Errorf("%q: accepted as %v", in, n.kind)
		}
	}
}

func TestNumbersAndLiterals(t *testing.T) {
	for in, want := range map[string]nodeKind{
		`0`: nNumber, `-0`: nNumber, `12`: nNumber, `1.5`: nNumber, `-1.5e-3`: nNumber, `1E+9`: nNumber,
		`true`: nBool, `false`: nBool, `null`: nNull, `[]`: nArray, `{}`: nObject, ` [ 1 , "a" ] `: nArray,
	} {
		n, err, _ := parseOne(in)
		if err != nil || n.kind != want {
			t.Errorf("%q: %v %v", in, n, err)
		}
	}
	// A literal is a token: the byte after it belongs to the caller.
	if _, err, pos := parseOne(`truex`); err != nil || pos != 4 {
		t.Errorf("truex: pos %d err %v", pos, err)
	}
}

func TestDepthLimit(t *testing.T) {
	ok := strings.Repeat("[", maxDepth) + strings.Repeat("]", maxDepth)
	if _, err, _ := parseOne(ok); err != nil {
		t.Fatalf("depth %d rejected: %s", maxDepth, err.msg)
	}
	deep := strings.Repeat("[", maxDepth+1) + strings.Repeat("]", maxDepth+1)
	if _, err, _ := parseOne(deep); err == nil || !strings.Contains(err.msg, "nesting") {
		t.Fatalf("depth %d accepted", maxDepth+1)
	}
	objs := strings.Repeat(`{"a":`, maxDepth+1) + "1" + strings.Repeat("}", maxDepth+1)
	if _, err, _ := parseOne(objs); err == nil {
		t.Fatal("deep objects accepted")
	}
	// Through the decoders: a deep value is syntax, not a crash.
	if e := listErr(t, `{"ImportMap":`+deep+`}`); e.Kind != KindSyntax {
		t.Fatalf("%v", e)
	}
}

func TestMembersKeepOrderRawKeysAndDuplicates(t *testing.T) {
	n, err, _ := parseOne(`{"b":1,"a":2,"b":3,"b":4}`)
	if err != nil {
		t.Fatal(err.msg)
	}
	var raws, keys []string
	for _, m := range n.members {
		raws = append(raws, m.raw)
		keys = append(keys, m.key)
	}
	if strings.Join(raws, ",") != `b,a,b,b` || strings.Join(keys, ",") != "b,a,b,b" {
		t.Fatalf("raw %v keys %v", raws, keys)
	}
}
