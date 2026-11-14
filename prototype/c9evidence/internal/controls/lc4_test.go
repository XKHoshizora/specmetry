package controls

import (
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

func TestLC4(t *testing.T) {
	env, _ := newEnv(t)
	vs := LC4()
	if len(vs) == 0 {
		t.Fatal("no LC-4 variants")
	}
	for _, j := range runVariants(t, vs, env) {
		if j.Judgement != verdict.Hit {
			t.Errorf("%s %s: %s %v", j.Control, j.Variant, j.Judgement, j.Reasons)
		}
	}
	t.Logf("%d LC-4 variants", len(vs))
}
