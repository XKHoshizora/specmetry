package derive

import "slices"

// criterionEval is one active criterion at a cutoff, with the symbols of
// data-model §6.1 and the results of §6.2 to §6.5.
type criterionEval struct {
	ent, parent *entityAt
	s           string // S at the cutoff; "" is NULL
	r           int    // revision
	h           string // check configuration hash; "" without a check
	automated   bool
	hasCheck    bool
	inputs      bool // the check declares inputs
	pr, ar      int

	runsS        []*CheckRun // runs(S), by seq
	v            string      // pass / fail / "" for none
	vRun         *CheckRun
	inconsistent bool
	d            *Decision // the current decision, or nil
	hist         bool
	histRevised  bool // HIST includes a record of an older revision or configuration
	progress     Progress

	status     CriterionStatus
	reason     string
	applicable bool // d is an acceptance that applies on S
	carried    bool
}

const (
	pass = "pass"
	fail = "fail"
)

// qualifying is a qualifying run (data-model §4.11.1): the same non-NULL
// HEAD at both ends, clean at both ends, no context change, and a grade an
// execution can have. Isolated runs are only recorded once verified.
func qualifying(r *CheckRun) bool {
	return r.HeadStart != "" && r.HeadStart == r.HeadEnd &&
		r.TreeStateStart == "clean" && r.TreeStateEnd == "clean" &&
		r.FailureKind != "context_changed" &&
		(r.EvidenceGrade == Observed || r.EvidenceGrade == SnapshotIsolated)
}

// evalCriterion evaluates a criterion: its runs, V, the current decision,
// HIST, the progress and its delivery status (data-model §6.1 to §6.5).
func evalCriterion(st *state, e, parent *entityAt, s string, settings Settings) *criterionEval {
	c := &criterionEval{ent: e, parent: parent, s: s, r: e.Revision, ar: e.RequirementRevision}
	c.automated = e.Verification == "automated"
	if e.Check != nil {
		c.hasCheck, c.h, c.inputs = true, CheckConfigHash(*e.Check), len(e.Check.Inputs) > 0
	}
	if parent != nil {
		c.pr = parent.Revision
	}
	runs := st.runs[e.ID]
	for _, run := range runs {
		if !qualifying(run) {
			continue
		}
		current := run.CriterionRevision == c.r && run.CheckConfigHash == c.h
		if s != "" && run.HeadStart == s && current {
			c.runsS = append(c.runsS, run)
		}
		if run.Result == pass && (run.HeadStart != s || !current) {
			c.hist = true
			if !current {
				c.histRevised = true
			}
		}
	}
	if n := len(c.runsS); n > 0 {
		c.vRun = c.runsS[n-1]
		c.v = fail
		if c.vRun.Result == pass {
			c.v = pass
		}
		for _, run := range c.runsS {
			if (run.Result == pass) != (c.v == pass) {
				c.inconsistent = true
			}
		}
	}
	for _, d := range st.decisions[e.ID] {
		if d.CriterionRevision == c.r {
			c.d = d // the latest by seq: the current decision (§5.4)
		}
		if d.Decision == "accepted" {
			c.hist = true
			if d.CriterionRevision != c.r {
				c.histRevised = true
			}
		}
	}
	c.progress = progressOf(st, e, settings)
	c.decide(st)
	return c
}

// progressOf is the progress table of data-model §6.5. The linked commits
// are those linked to the criterion itself.
func progressOf(st *state, e *entityAt, settings Settings) Progress {
	var latest *claimAt
	for i, cl := range st.claims[e.ID] {
		if cl.CriterionRevision == e.Revision {
			latest = &st.claims[e.ID][i]
		}
	}
	if latest != nil {
		switch latest.Assertion {
		case "done":
			return ProgressClaimed
		case "blocked":
			return ProgressBlocked
		case "partial":
			return ProgressInProgress
		case "not_done":
			return ProgressNotStarted
		default:
			return ProgressUnknown
		}
	}
	if slices.ContainsFunc(st.commitSeqs[e.ID], func(s int64) bool { return s > e.revisionSeq }) {
		return ProgressInProgress
	}
	if settings.Stage == DocsOnly {
		return ProgressNotStarted
	}
	return ProgressUnknown
}

// applies reports whether the current decision is an acceptance that
// applies on S (data-model §6.2), and whether by carry-forward.
func (c *criterionEval) applies() (applies, carried bool) {
	d := c.d
	if d == nil || d.Decision != "accepted" || c.s == "" {
		return false, false
	}
	if d.Snapshot == c.s {
		return true, false
	}
	if !c.automated || d.Mode != "normal" || c.v != pass || c.inconsistent || c.vRun.EvidenceGrade != SnapshotIsolated ||
		d.CheckConfigHash != c.h || (c.inputs && !InputsEqual(c.vRun.Inputs, d.Inputs)) {
		return false, false
	}
	return true, true
}

// liftedRejection reports whether a rejection is lifted (data-model §6.3
// rule 5): the code changed, and new evidence came after the rejection.
func (c *criterionEval) liftedRejection(st *state) bool {
	d := c.d
	if c.s == d.Snapshot {
		return false
	}
	if c.automated {
		return c.v == pass && c.vRun.Seq > d.Seq
	}
	return slices.ContainsFunc(st.claims[c.ent.ID], func(cl claimAt) bool {
		return cl.CriterionRevision == c.r && cl.Assertion == "done" && cl.Seq > d.Seq
	})
}

// decide applies the decision table of data-model §6.3: the first rule that
// holds.
func (c *criterionEval) decide(st *state) {
	c.applicable, c.carried = c.applies()
	d := c.d
	is := func(decision string) bool { return d != nil && d.Decision == decision }
	switch {
	case c.s == "": // 1
		c.status, c.reason = CriterionNotAssessed, "target_unresolved"
	case c.ar != c.pr: // 2
		c.status, c.reason = CriterionNeedsReverification, "parent_revised"
	case is("waived"): // 3
		c.status, c.reason = CriterionWaived, ""
	case c.automated && c.v == fail: // 4
		c.status, c.reason = CriterionFailing, "check_failed"
		if is("accepted") && d.Seq < c.vRun.Seq {
			c.reason = "regressed"
		}
	case is("rejected") && !c.liftedRejection(st): // 5
		c.status, c.reason = CriterionRejected, "rework_requested"
	case is("accepted") && c.applicable: // 6
		c.status = CriterionAccepted
		switch {
		case c.carried:
			c.reason = "carried_forward"
		case d.Mode == "exception":
			c.reason = "exception"
		case d.Mode == "manual":
			c.reason = "manual"
		default:
			c.reason = "direct"
		}
	case c.automated && c.v == pass: // 7
		c.status, c.reason = CriterionReadyForReview, c.reviewReason()
	case !c.automated && is("accepted"): // 8
		c.status, c.reason = CriterionNeedsReverification, "code_changed"
	case !c.automated && c.progress == ProgressClaimed: // 9
		c.status, c.reason = CriterionReadyForReview, "manual_claimed"
	case c.hist: // 10
		c.status, c.reason = CriterionNeedsReverification, "code_changed"
		if c.histRevised {
			c.reason = "ac_revised"
		}
	default: // 11
		c.status = CriterionInProgress
		if c.automated && !c.hasCheck {
			c.reason = "no_check"
		} else {
			c.reason = progressReason[c.progress]
		}
	}
}

var progressReason = map[Progress]string{
	ProgressClaimed:    "claimed_without_evidence",
	ProgressBlocked:    "blocked",
	ProgressInProgress: "agent_in_progress",
	ProgressNotStarted: "not_started",
	ProgressUnknown:    "unknown",
}

// reviewReason is rule 7's reason code: a lifted rejection; an acceptance
// on another snapshot that was not carried forward, by priority; or a first
// review. Legal facts always meet one of these; anything else is a first
// review.
func (c *criterionEval) reviewReason() string {
	d := c.d
	switch {
	case d != nil && d.Decision == "rejected":
		return "rework_submitted"
	case d == nil || d.Decision != "accepted":
		return "first_review"
	case d.Mode == "exception":
		return "exception_not_carried"
	case c.inconsistent:
		return "inconsistent_results"
	case c.inputs && !InputsEqual(c.vRun.Inputs, d.Inputs):
		return "inputs_changed"
	case c.vRun.EvidenceGrade == Observed:
		return "observed_evidence"
	}
	return "first_review"
}
