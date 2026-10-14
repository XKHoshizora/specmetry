package ulid_test

import (
	"regexp"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/ulid"
)

var pattern = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

// The encoding of the ULID specification: 48-bit milliseconds then 80
// random bits, in Crockford base32.
func TestMake(t *testing.T) {
	cases := []struct {
		ms      uint64
		entropy [10]byte
		want    string
	}{
		{0, [10]byte{}, "00000000000000000000000000"},
		{1<<48 - 1, [10]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, "7ZZZZZZZZZZZZZZZZZZZZZZZZZ"},
		// The specification's example timestamp.
		{1469918176385, [10]byte{}, "01ARYZ6S410000000000000000"},
		{0, [10]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, "00000000000000000000000001"},
	}
	for _, c := range cases {
		if got := ulid.Make(c.ms, c.entropy); got != c.want {
			t.Errorf("Make(%d, %x) = %s, want %s", c.ms, c.entropy, got, c.want)
		}
	}
}

func TestNewIsValidAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		id := ulid.New()
		if !pattern.MatchString(id) {
			t.Fatalf("New() = %q is not a ULID", id)
		}
		if seen[id] {
			t.Fatalf("New() repeated %s", id)
		}
		seen[id] = true
	}
}
