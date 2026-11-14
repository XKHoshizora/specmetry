package controls

import (
	"fmt"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/verdict"
)

// refusingOuter admits nothing after the first n preparations and no
// cleanup at all, as the outer layer does once its deadlines have passed.
type refusingOuter struct {
	MemOuter
	prepare int
}

func (r *refusingOuter) Prepare(name string) (func(error), bool) {
	if r.prepare <= 0 {
		return nil, false
	}
	r.prepare--
	return r.MemOuter.Prepare(name)
}

func (r *refusingOuter) Cleanup(string, string) (func(error), bool) { return nil, false }

// Review 148 R-c: a refused preparation is "not admitted" (a miss, never a
// preparation failure or a success), and a refused cleanup is "not
// admitted" (pending, never a success); the directory stays held.
func TestB148RefusalsInTheOuterJudgement(t *testing.T) {
	env, _ := newEnv(t)
	ro := &refusingOuter{}
	env.Outer = ro
	v := LC3()[0]
	j := Judge(v, env.GOOS, v.Run(env), true)
	if j.Judgement != verdict.Miss || len(j.Reasons) != 1 || !strings.HasPrefix(j.Reasons[0], "outer preparation not admitted") {
		t.Errorf("refused preparation: %s %v", j.Judgement, j.Reasons)
	}
	ro.prepare = 1
	o := v.Run(env)
	j = Judge(v, env.GOOS, o, true)
	if j.Judgement != verdict.Pending || len(j.Reasons) != 1 || !strings.HasPrefix(j.Reasons[0], "outer cleanup not admitted") {
		t.Errorf("refused cleanup: %s %v", j.Judgement, j.Reasons)
	}
	if held := ro.Held(); len(held) != 1 {
		t.Errorf("held %v, want the fixture whose cleanup was refused", held)
	}
	// A real failure keeps its own reason.
	j = Judge(v, env.GOOS, Outcome{PrepErr: fmt.Errorf("EPERM")}, true)
	if len(j.Reasons) != 1 || !strings.HasPrefix(j.Reasons[0], "outer preparation failed") {
		t.Errorf("real failure: %v", j.Reasons)
	}
}
