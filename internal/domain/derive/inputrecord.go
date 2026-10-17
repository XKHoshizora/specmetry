package derive

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
)

// CheckConfigHash is the check configuration hash of data-model §4.4: the
// sha256 of {"command","inputs","shell","workdir"} in JSON canonical form
// (RFC 8785), with the inputs sorted, absent inputs as an empty list and a
// shell that is not set as null. It is the formula M1 stores, so the hash
// of a historical revision can be computed from its snapshot.
func CheckConfigHash(c Check) string {
	inputs := slices.Clone(c.Inputs)
	slices.Sort(inputs)
	var b strings.Builder
	b.WriteString(`{"command":`)
	jcsString(&b, c.Command)
	b.WriteString(`,"inputs":[`)
	for i, in := range inputs {
		if i > 0 {
			b.WriteByte(',')
		}
		jcsString(&b, in)
	}
	b.WriteString(`],"shell":`)
	if c.Shell == nil {
		b.WriteString("null")
	} else {
		jcsString(&b, *c.Shell)
	}
	b.WriteString(`,"workdir":`)
	jcsString(&b, c.Workdir)
	b.WriteByte('}')
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// jcsString writes s as RFC 8785 serializes a string.
func jcsString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\f':
			b.WriteString(`\f`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r < 0x20 {
				fmt.Fprintf(b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}

// InputsEqual is the equality of two input records (data-model §4.11.2):
// their sets of {path, mode, type, oid} are the same. Comparing object IDs
// alone is not enough: one blob can be a file and a symbolic link. A NULL
// record equals nothing.
func InputsEqual(a, b InputRecord) bool {
	if a == nil || b == nil {
		return false
	}
	return slices.Equal(entrySet(a), entrySet(b))
}

func entrySet(r InputRecord) []InputEntry {
	s := slices.Clone(r)
	slices.SortFunc(s, func(x, y InputEntry) int {
		return cmp.Or(strings.Compare(x.Path, y.Path), strings.Compare(x.Mode, y.Mode),
			strings.Compare(x.Type, y.Type), strings.Compare(x.OID, y.OID))
	})
	return slices.Compact(s)
}
