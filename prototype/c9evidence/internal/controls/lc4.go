package controls

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/decode"
	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
)

// LC4 are the decoder controls (plan §5.2 LC-4): every byte variant of
// decode.LC4Variants, each its own instance with the check "decode"
// entered with the input's SHA-256. A variant that must decode needs the
// decoded values to equal the ones written with the variant; one that must
// not needs the exact error kind (J-3).
func LC4() []Variant {
	var out []Variant
	for _, dv := range decode.LC4Variants() {
		dv := dv
		var want []string
		if dv.WantOK {
			want = []string{"decode hold", "values as expected"}
		} else {
			want = []string{"decode fail J-3 " + dv.Call.String() + " " + string(dv.WantKind) + " real @ decode at " + dv.WantAt}
		}
		out = append(out, Variant{
			Control: "LC-4", Name: dv.Name, Applies: always, MustEnter: []string{"decode"}, Want: want,
			Run: func(*Env) Outcome {
				ck := judge.NewChecks([]string{"decode"})
				sum := sha256.Sum256(dv.Input)
				ck.Enter("decode", hex.EncodeToString(sum[:]))
				o := dv.Run()
				lines := []string{}
				if o.OK {
					ck.Hold("decode")
					lines = append(lines, CheckLines(ck, noNorm)...)
					if err := dv.Match(o); err == nil {
						lines = append(lines, "values as expected")
					} else {
						lines = append(lines, "values differ: "+err.Error())
					}
				} else {
					kind, at, detail := "unknown", "", ""
					if o.Err != nil {
						kind, at, detail = string(o.Err.Kind), o.Err.At, o.Err.Error()
					}
					// The site is where the decoder stopped (a JSON path or a
					// line of the context text).
					ck.Fail("decode", judge.Real(judge.J3, dv.Call.String(), "decode at "+at, kind, detail))
					lines = append(lines, CheckLines(ck, noNorm)...)
				}
				inst := instanceOf(ck)
				inst.Extra["outcome"] = o
				return Outcome{Entered: Entered(ck), Lines: lines, Instance: inst, Terminal: true}
			},
		})
	}
	return out
}
