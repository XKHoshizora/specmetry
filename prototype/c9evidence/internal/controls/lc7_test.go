package controls

import (
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

func TestLC7AllGOOS(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		env, _ := newEnv(t)
		env.GOOS = goos
		for _, j := range runVariants(t, LC7(goos), env) {
			if j.Judgement != verdict.Hit {
				t.Errorf("%s: %s %v", goos, j.Judgement, j.Reasons)
			}
		}
	}
}
