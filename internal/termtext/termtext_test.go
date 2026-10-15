package termtext_test

import (
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

func TestLines(t *testing.T) {
	got := termtext.Lines([]string{"a\x1b", "b\r\nc"})
	if len(got) != 2 || got[0] != `a\x1b` || got[1] != "b\nc" {
		t.Fatalf("Lines = %q", got)
	}
}
