package decode

import (
	"bytes"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

// DecodeEnv decodes the output of the first call, go env -json with the
// requested names (spec.EnvNames; method record §3.2, §2.2 item 7).
//
// The fixed cmd/go writes a map[string]string through a json.Encoder with
// a tab indent (cmd/go/internal/envcmd/env.go:605-620), so the output is
// one JSON object followed by the encoder's single LF. The decode order is
// that of the method record §3.1, and the first problem found is returned:
//
//  1. Form: the output is not empty (else KindCount); its first byte opens
//     the object (no leading bytes, else KindForm); the object parses
//     (else KindSyntax); after it comes exactly one LF and nothing else: no
//     LF is KindForm, another object is KindCount, any other byte
//     (including CR, a second LF or whitespace) is KindTrailing.
//  2. Keys, in output order: every key byte-equal to a requested name
//     (KindKey for other names and escaped keys, KindCase for case
//     variants), at most once (KindDuplicate), not null (KindNull); then
//     every requested name present, in the order of names (KindMissing).
//  3. Types: every value is a string (KindType).
//
// The values are returned as decoded; an empty string is a value obtained,
// not a missing one. The requirements of the §3.2 table are the caller's.
func DecodeEnv(out []byte, names []string) (map[string]string, error) {
	const call = spec.CallEnv
	if len(names) == 0 {
		return nil, errorf(call, KindUsage, "$", 0, "no names requested")
	}
	want := map[string]bool{}
	for _, n := range names {
		if want[n] {
			return nil, errorf(call, KindUsage, "$", 0, "name %q requested twice", n)
		}
		want[n] = true
	}

	// Step 1: form.
	if len(out) == 0 {
		return nil, errorf(call, KindCount, "$", 0, "zero-byte output, want one JSON object")
	}
	if out[0] != '{' {
		return nil, errorf(call, KindForm, "$", 0, "output begins with %q, want '{'", out[0])
	}
	p := &parser{data: out}
	obj, serr := p.value()
	if serr != nil {
		return nil, errorf(call, KindSyntax, "$", serr.off, "%s", serr.msg)
	}
	rest := out[p.pos:]
	switch {
	case bytes.Equal(rest, []byte("\n")):
	case len(rest) == 0:
		return nil, errorf(call, KindForm, "$", p.pos, "the object is not followed by a LF")
	default:
		q := &parser{data: out, pos: p.pos}
		q.skipWS()
		if q.pos < len(out) && out[q.pos] == '{' {
			return nil, errorf(call, KindCount, "$", q.pos, "a second JSON object follows the first")
		}
		return nil, errorf(call, KindTrailing, "$", p.pos, "%d bytes after the object, want exactly one LF", len(rest))
	}

	// Steps 2 and 3.
	s := envSchema(names)
	if err := checkKeys(call, obj, s, "$"); err != nil {
		return nil, err
	}
	present := map[string]bool{}
	for _, m := range obj.members {
		present[m.raw] = true
	}
	for _, n := range names {
		if !present[n] {
			return nil, errorf(call, KindMissing, keyPath("$", n), len(out)-2, "requested name %q has no key", n)
		}
	}
	if err := checkTypes(call, obj, s, "$"); err != nil {
		return nil, err
	}

	env := make(map[string]string, len(names))
	for _, m := range obj.members {
		env[m.raw] = m.val.str
	}
	return env, nil
}
