package controls

import (
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

func TestLC5(t *testing.T) {
	env, _ := newEnv(t)
	for _, j := range runVariants(t, LC5(), env) {
		if j.Judgement != verdict.Hit {
			t.Errorf("%s %s: %s %v", j.Control, j.Variant, j.Judgement, j.Reasons)
		}
	}
}
