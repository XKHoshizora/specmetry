// Package verdict decides a job's conclusion from the three evidence
// domains and the identity checks (plan §5.4, §5.5, §7.1): 1 invalid,
// 2 incomplete, 3 inconsistent with the candidates, 4 consistent with
// them, by priority. It only reads published state; it never waits.
package verdict

import (
	"fmt"
	"sort"

	"github.com/XKHoshizora/specmetry/prototype/c9evidence/internal/judge"
)

// Conclusion is one of the four conclusions.
type Conclusion int

// The conclusions (plan §7.1).
const (
	Invalid      Conclusion = 1
	Incomplete   Conclusion = 2
	Inconsistent Conclusion = 3
	Consistent   Conclusion = 4
)

func (c Conclusion) String() string {
	switch c {
	case Invalid:
		return "invalid"
	case Incomplete:
		return "incomplete"
	case Inconsistent:
		return "inconsistent-with-candidates"
	case Consistent:
		return "consistent-with-candidates"
	}
	return "unknown"
}

// ExitCode is LE-10: 0 exactly when the conclusion is 3 or 4.
func ExitCode(c Conclusion) int {
	if c == Inconsistent || c == Consistent {
		return 0
	}
	return 1
}

// Judgement is the outer layer's judgement of one control variant (plan
// §5.5).
type Judgement string

// The judgements.
const (
	Hit           Judgement = "hit"
	Miss          Judgement = "miss"
	Pending       Judgement = "pending"
	NotApplicable Judgement = "not-applicable"
)

// Variant is one judged control variant of the F domain.
type Variant struct {
	Control   string
	Variant   string
	Judgement Judgement
	Reasons   []string
}

// RConfig is one configuration of the R domain.
type RConfig struct {
	Config string
	// ABHold is "every check of A and B holds" in R (plan §7.1: levels 3
	// and 4 refer to R only).
	ABHold bool
	// Failures are every J failure of R in this configuration, the calls'
	// control failures included; K's comparison result is not a failure.
	Failures []judge.Failure
	// Unevaluated lists the checks of E without a hold or fail result.
	Unevaluated []string
	// Compared is K evaluated; Consistent is the findings equal to the
	// candidates with no new package (LQ-6).
	Compared   bool
	Consistent bool
}

// Inputs is the published state a conclusion is computed from.
type Inputs struct {
	T2OK     bool
	MRunCode int
	// Defects are tool defects that make results uninterpretable: unknown
	// check IDs (any result of the not-covered set N among them), second
	// results, evaluations without entry records.
	Defects  []string
	Variants []Variant
	R        []RConfig
	// O is the outer domain: cleanup failures or unknowns, operations not
	// terminal at the snapshot, and the record chain's published failures.
	OFailures []string
	OPending  []string
}

// Decide returns the conclusion and the reasons, by priority (plan §7.1).
func Decide(in Inputs) (Conclusion, []string) {
	var l1, l2 []string
	if !in.T2OK {
		l1 = append(l1, "T2 binding differs (LC-10)")
	}
	if in.MRunCode != 0 {
		l1 = append(l1, fmt.Sprintf("m.Run returned %d", in.MRunCode))
	}
	for _, d := range in.Defects {
		l1 = append(l1, "tool defect: "+d)
	}
	for _, v := range in.Variants {
		switch v.Judgement {
		case Miss:
			l1 = append(l1, fmt.Sprintf("%s %s miss: %v", v.Control, v.Variant, v.Reasons))
		case Pending:
			l2 = append(l2, fmt.Sprintf("%s %s pending: %v", v.Control, v.Variant, v.Reasons))
		}
	}
	if len(in.R) == 0 {
		l2 = append(l2, "R: no configuration was collected")
	}
	for _, r := range in.R {
		for _, f := range r.Failures {
			l2 = append(l2, "R "+r.Config+": "+f.String())
		}
		for _, u := range r.Unevaluated {
			l2 = append(l2, "R "+r.Config+": unevaluated "+u)
		}
		if !r.ABHold && len(r.Failures) == 0 && len(r.Unevaluated) == 0 {
			l2 = append(l2, "R "+r.Config+": A and B do not all hold")
		}
		if r.ABHold && !r.Compared && len(r.Failures) == 0 {
			l2 = append(l2, "R "+r.Config+": comparison with the candidates not evaluated")
		}
	}
	for _, o := range in.OFailures {
		l2 = append(l2, "O: "+o)
	}
	for _, o := range in.OPending {
		l2 = append(l2, "O pending: "+o)
	}
	if len(l1) > 0 {
		return Invalid, append(l1, l2...)
	}
	if len(l2) > 0 {
		return Incomplete, l2
	}
	var diff []string
	for _, r := range in.R {
		if !r.Consistent {
			diff = append(diff, "R "+r.Config+": findings differ from the candidates")
		}
	}
	if len(diff) > 0 {
		sort.Strings(diff)
		return Inconsistent, diff
	}
	return Consistent, nil
}

// Level1 reports whether the published state already makes the job invalid;
// the expiry exit before a provisional conclusion uses it (plan §7.1).
func Level1(in Inputs) bool {
	c, _ := Decide(in)
	return c == Invalid
}
