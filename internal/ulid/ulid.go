// Package ulid generates ULIDs, the identifiers of Specmetry's rows
// (data-model.md §1). They identify; they never order facts, which is what
// seq is for.
package ulid

import (
	"crypto/rand"
	"encoding/binary"
	"strings"
	"time"
)

// crockford is the Crockford base32 alphabet used by ULIDs.
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// Make encodes a ULID from a Unix millisecond timestamp (its low 48 bits)
// and 80 bits of entropy: 26 characters, 5 bits each, most significant
// first (the 130-bit encoding starts with 2 zero bits).
func Make(ms uint64, entropy [10]byte) string {
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], ms)
	var raw [16]byte
	copy(raw[:6], ts[2:])
	copy(raw[6:], entropy[:])
	var out [26]byte
	for i := range out {
		// Bit offset of character i within the 130-bit value whose first
		// two bits are zero.
		bit := 5*i - 2
		var v uint
		for b := range 5 {
			pos := bit + b
			v <<= 1
			if pos >= 0 && raw[pos/8]&(0x80>>(pos%8)) != 0 {
				v |= 1
			}
		}
		out[i] = crockford[v]
	}
	return string(out[:])
}

// New returns a new ULID for the current time with random entropy.
func New() string {
	var entropy [10]byte
	_, _ = rand.Read(entropy[:]) // never fails (crypto/rand)
	return Make(uint64(time.Now().UnixMilli()), entropy)
}

// Valid reports whether s has the form of a ULID: 26 characters of the
// Crockford alphabet, as the exchange schemas require (common.v1 ulid).
func Valid(s string) bool {
	if len(s) != 26 {
		return false
	}
	for i := range len(s) {
		if strings.IndexByte(crockford, s[i]) < 0 {
			return false
		}
	}
	return true
}
