package decode

import (
	"errors"
	"math/rand/v2"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/spec"
)

var knownKinds = map[Kind]bool{
	KindCount: true, KindForm: true, KindSyntax: true, KindTrailing: true, KindKey: true, KindCase: true,
	KindDuplicate: true, KindNull: true, KindType: true, KindMissing: true, KindLine: true, KindValue: true,
	KindCanonical: true,
}

// decodeAll runs every decoder on one input and checks the error contract:
// either success or an *Error of the right call with a known kind, a
// location and an offset inside the input. A panic fails the test.
func decodeAll(t *testing.T, in []byte) {
	t.Helper()
	check := func(call spec.CallKind, err error) {
		t.Helper()
		if err == nil {
			return
		}
		var de *Error
		if !errors.As(err, &de) {
			t.Fatalf("%v: error of type %T for %q", call, err, in)
		}
		if de.Call != call || !knownKinds[de.Kind] || de.At == "" || de.Offset < 0 || de.Offset > len(in) || de.Detail == "" {
			t.Fatalf("%v: bad error %+v for %q", call, de, in)
		}
	}
	_, err := DecodeEnv(in, spec.EnvNames)
	check(spec.CallEnv, err)
	_, err = DecodeContext(in)
	check(spec.CallContext, err)
	_, err = DecodePackages(in)
	check(spec.CallList, err)
	g, problems := ParseGoSum(in)
	if g.Size != len(in) {
		t.Fatalf("go.sum size %d for %d bytes", g.Size, len(in))
	}
	for _, p := range problems {
		if p.Line < 1 || p.Line > g.Lines || p.Detail == "" {
			t.Fatalf("bad go.sum problem %+v for %q", p, in)
		}
	}
}

// TestNoPanicOnTruncationsAndMutations feeds every prefix of every variant
// input, and deterministic byte mutations of it, to every decoder.
func TestNoPanicOnTruncationsAndMutations(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	repl := []byte{0, '\n', '\r', ' ', '"', '\\', '{', '}', '[', ']', ':', ',', '-', 'n', 0x7f, 0xc3, 0xff}
	for _, v := range LC4Variants() {
		in := v.Input
		for n := 0; n <= len(in); n++ {
			decodeAll(t, in[:n])
		}
		for i := 0; i < 64 && len(in) > 0; i++ {
			m := append([]byte(nil), in...)
			m[r.IntN(len(m))] = repl[r.IntN(len(repl))]
			decodeAll(t, m)
		}
	}
}

// TestNoPanicOnRandomBytes: random inputs, including short ones built from
// the structural bytes.
func TestNoPanicOnRandomBytes(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	alphabet := []byte("{}[]\":,\\ \t\r\nnulltruefalse0123456789-+.eEu-c9 GOOSendh1")
	for i := 0; i < 2000; i++ {
		b := make([]byte, r.IntN(64))
		for j := range b {
			if i%2 == 0 {
				b[j] = byte(r.IntN(256))
			} else {
				b[j] = alphabet[r.IntN(len(alphabet))]
			}
		}
		decodeAll(t, b)
	}
}

// TestTruncatedHealthyOutputsFail: a healthy first-call or second-call
// output cut short anywhere never decodes (it has exactly one object or
// text, ending in its final LF).
func TestTruncatedHealthyOutputsFail(t *testing.T) {
	for _, in := range []string{envLinux, envWindows} {
		for n := 0; n < len(in); n++ {
			if _, err := DecodeEnv([]byte(in[:n]), spec.EnvNames); err == nil {
				t.Fatalf("env prefix of %d bytes decoded", n)
			}
		}
	}
	for n := 0; n < len(ctxLinux); n++ {
		if _, err := DecodeContext([]byte(ctxLinux[:n])); err == nil {
			t.Fatalf("context prefix of %d bytes decoded", n)
		}
	}
	// A third-call prefix decodes only when it ends at a record boundary
	// (followed by whitespace only).
	stream := listStd + listMain + listDep
	ok := map[int]bool{len(listStd) - 1: true, len(listStd): true, len(listStd+listMain) - 1: true, len(listStd + listMain): true, len(stream) - 1: true}
	for n := 0; n < len(stream); n++ {
		_, err := DecodePackages([]byte(stream[:n]))
		if (err == nil) != ok[n] {
			t.Fatalf("list prefix of %d bytes: err %v", n, err)
		}
	}
}
