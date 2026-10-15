package proposal

import (
	"strings"
	"testing"
)

// codex-review-020 B4: a code reference's lines are none, one line or an
// ordered range of lines from 1 to 2147483647; anything else is malformed,
// never "no range". Expected values are written out.
func TestParseLines(t *testing.T) {
	nines := strings.Repeat("9", 30)
	for _, c := range []struct {
		in   string
		want lineSpan
		ok   bool
	}{
		{"", lineSpan{}, true},
		{"1", lineSpan{set: true, from: 1, to: 1}, true},
		{"3-7", lineSpan{set: true, from: 3, to: 7}, true},
		{"7-7", lineSpan{set: true, from: 7, to: 7}, true},
		{"2147483647", lineSpan{set: true, from: 2147483647, to: 2147483647}, true},
		{"1-2147483647", lineSpan{set: true, from: 1, to: 2147483647}, true},
		{"7-3", lineSpan{}, false},
		{"0", lineSpan{}, false},
		{"0-3", lineSpan{}, false},
		{"2147483648", lineSpan{}, false},
		{"1-2147483648", lineSpan{}, false},
		{"9223372036854775807", lineSpan{}, false},
		{"9223372036854775808", lineSpan{}, false},
		{nines, lineSpan{}, false},
		{nines + "-1", lineSpan{}, false},
		{"1-" + nines, lineSpan{}, false},
		{"-3", lineSpan{}, false},
		{"3-", lineSpan{}, false},
		{"1-2-3", lineSpan{}, false},
	} {
		if got, ok := parseLines(c.in); got != c.want || ok != c.ok {
			t.Errorf("parseLines(%q) = %+v, %v; want %+v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}
