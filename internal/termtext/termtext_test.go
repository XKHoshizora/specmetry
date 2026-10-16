package termtext_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/XKHoshizora/specmetry/internal/termtext"
)

func TestEscape(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", ""},
		{"plain text, 中文, emoji 👩\u200d💻, a\\x1b backslash", "plain text, 中文, emoji 👩\u200d💻, a\\x1b backslash"},
		{"line one\nline two\tafter a tab", "line one\nline two\tafter a tab"},
		{"windows\r\nline", "windows\nline"},
		{"overwrite\rthis", `overwrite\rthis`},
		{"\r\r\n", `\r` + "\n"},
		{"bell\a back\b feed\f vtab\v", `bell\a back\b feed\f vtab\v`},
		{"nul\x00 esc\x1b us\x1f del\x7f", `nul\x00 esc\x1b us\x1f del\x7f`},
		{"\x1b[2J\x1b[H", `\x1b[2J\x1b[H`},
		{"\x1b]0;title\x07", `\x1b]0;title\a`},
		{"c1 \u0080 \u0085 \u009b[31m \u009f", `c1 \u0080 \u0085 \u009b[31m \u009f`},
		{"bidi \u202aa\u202bb\u202cc\u202dd\u202ee", `bidi \u202aa\u202bb\u202cc\u202dd\u202ee`},
		{"\u2066i\u2067r\u2068f\u2069 \u200el\u200fr \u061ca", `\u2066i\u2067r\u2068f\u2069 \u200el\u200fr \u061ca`},
		{"sep\u2028line\u2029para", `sep\u2028line\u2029para`},
		{"bad \x9b[31m utf8 \xff\xfe", `bad \x9b[31m utf8 \xff\xfe`},
		{"a real \uFFFD stays", "a real \uFFFD stays"},
	} {
		got := termtext.Escape(c.in)
		if got != c.want {
			t.Errorf("Escape(%q) = %q, want %q", c.in, got, c.want)
		}
		if again := termtext.Escape(got); again != got {
			t.Errorf("Escape is not idempotent on %q: %q", got, again)
		}
	}
}

// Every rune and every single byte: what Escape writes is UTF-8 with no
// control other than LF and TAB, no C1 control, no bidirectional control
// and no separator.
func TestEscapeLeavesNoControl(t *testing.T) {
	var all strings.Builder
	for r := rune(0); r <= 0x3000; r++ {
		all.WriteRune(r)
	}
	for b := 0; b < 256; b++ {
		all.WriteByte(byte(b))
	}
	out := termtext.Escape(all.String())
	if !utf8.ValidString(out) {
		t.Fatal("the escaped text is not UTF-8")
	}
	for _, r := range out {
		bad := r < 0x20 && r != '\n' && r != '\t' || r == 0x7f || r >= 0x80 && r <= 0x9f ||
			r == 0x061c || r == 0x200e || r == 0x200f || r >= 0x202a && r <= 0x202e || r >= 0x2066 && r <= 0x2069 ||
			r == 0x2028 || r == 0x2029
		if bad {
			t.Fatalf("Escape left %U", r)
		}
	}
}

// Line also shows line feeds and tabs: a path or a name stays one line.
func TestLine(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"docs/a.md", "docs/a.md"},
		{"docs/需求 🚀.md", "docs/需求 🚀.md"},
		{"a\nb\tc\r\nd\x1b", `a\nb\tc\nd\x1b`},
	} {
		if got := termtext.Line(c.in); got != c.want {
			t.Errorf("Line(%q) = %q, want %q", c.in, got, c.want)
		}
		if again := termtext.Line(termtext.Line(c.in)); again != c.want {
			t.Errorf("Line is not idempotent on %q: %q", c.in, again)
		}
	}
}

// A Writer escapes what is written to it and reports it written whole; Raw
// reaches past it to the writer underneath; NewWriter does not wrap twice.
func TestWriter(t *testing.T) {
	var buf bytes.Buffer
	w := termtext.NewWriter(&buf)
	n, err := io.WriteString(w, "a\x1b[2J\r\nb")
	if err != nil || n != len("a\x1b[2J\r\nb") || buf.String() != `a\x1b[2J`+"\nb" {
		t.Fatalf("Write: %d, %v, %q", n, err, buf.String())
	}
	if termtext.NewWriter(w) != w {
		t.Error("NewWriter wrapped a Writer again")
	}
	if termtext.Raw(termtext.NewWriter(w)) != io.Writer(&buf) || termtext.Raw(&buf) != io.Writer(&buf) {
		t.Error("Raw does not reach the writer underneath")
	}
	if _, err := termtext.NewWriter(failing{}).Write([]byte("x")); err == nil {
		t.Error("a failed write is reported written")
	}
}

type failing struct{}

func (failing) Write([]byte) (int, error) { return 0, io.ErrShortWrite }
