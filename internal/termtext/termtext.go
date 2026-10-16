// Package termtext makes dynamic text safe to write to a terminal
// (codex-review-027 B1). Text from a proposal, a person, the configuration
// or the environment may hold characters a terminal acts on: clear the
// screen, move the cursor, overwrite a line, set the title, ring the bell,
// or reverse the reading order. Escape writes them as visible escapes
// instead, so nothing is acted on and nothing is lost. It is for text
// output only: stored values and JSON output keep the text as it is.
package termtext

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Escape returns s as it is safe to show on a terminal:
//   - a line feed and a tab stay; CRLF becomes a line feed;
//   - every other C0 control, DEL and every C1 control is written as its
//     escape: \a, \b, \f, \r, \v, or \xNN (\x1b for ESC, \x7f for DEL),
//     and \u0080 to \u009f;
//   - the bidirectional controls (U+061C, U+200E, U+200F, U+202A to
//     U+202E, U+2066 to U+2069) and the line and paragraph separators
//     (U+2028, U+2029) are written as \uNNNN;
//   - a byte that is not UTF-8 is written as \xNN.
//
// A backslash stays as it is: what Escape writes is plain text either way.
// Escape(Escape(s)) == Escape(s).
func Escape(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			fmt.Fprintf(&b, `\x%02x`, s[i])
		case shown(r):
			b.WriteString(s[i : i+size])
		case r < 0x80:
			b.WriteString(ascii(r))
		default:
			fmt.Fprintf(&b, `\u%04x`, r)
		}
		i += size
	}
	return b.String()
}

// shown reports whether r is written as it is.
func shown(r rune) bool {
	switch {
	case r == '\n', r == '\t':
		return true
	case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
		return false
	case r == 0x061c, r == 0x200e, r == 0x200f, r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069:
		return false
	case r == 0x2028, r == 0x2029:
		return false
	}
	return true
}

// ascii is the escape of an ASCII control.
func ascii(r rune) string {
	switch r {
	case '\a':
		return `\a`
	case '\b':
		return `\b`
	case '\f':
		return `\f`
	case '\r':
		return `\r`
	case '\v':
		return `\v`
	}
	return fmt.Sprintf(`\x%02x`, r)
}

// Lines is Escape for each of lines.
func Lines(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = Escape(l)
	}
	return out
}
