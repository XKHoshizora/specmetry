package derive

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// InputError is a derivation refused because its input is incomplete or
// invalid: a missing current-value input, a missing observation, facts that
// do not reach the cutoff. The derivation never fills a missing input in
// (data-model §6.13.1).
type InputError struct {
	Problems []string
}

func (e *InputError) Error() string {
	return "derivation input: " + strings.Join(e.Problems, "; ")
}

var stages = map[Stage]bool{DocsOnly: true, InDevelopment: true, IncompleteDocs: true}

// checkCommon checks what both entry points need: the project, the facts
// up to the cutoff, S and the settings.
func checkCommon(in *Input) []string {
	var problems []string
	if in.Facts.ProjectID == "" {
		problems = append(problems, "no project")
	}
	if in.AsOfSeq < 0 {
		problems = append(problems, fmt.Sprintf("a negative cutoff %d", in.AsOfSeq))
	}
	if in.Facts.LatestSeq < in.AsOfSeq {
		problems = append(problems, fmt.Sprintf("the facts reach seq %d, before the cutoff %d", in.Facts.LatestSeq, in.AsOfSeq))
	}
	if in.Current.Snapshot == nil {
		problems = append(problems, "missing the snapshot S")
	}
	switch {
	case in.Current.Settings == nil:
		problems = append(problems, "missing the settings")
	case !stages[in.Current.Settings.Stage]:
		problems = append(problems, fmt.Sprintf("missing or unknown stage %q", in.Current.Settings.Stage))
	}
	return problems
}

// checkObservations checks that the observations cover the source set at
// the cutoff exactly: one for each member and no other.
func checkObservations(st *state, obs []Observation) []string {
	var problems []string
	count := map[string]int{}
	for _, o := range obs {
		count[o.SourceID]++
	}
	ids := make([]string, 0, len(count))
	for id := range count {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		switch {
		case !st.sourceByID[id]:
			problems = append(problems, fmt.Sprintf("an observation of source %s, which is not in the source set at the cutoff", id))
		case count[id] > 1:
			problems = append(problems, fmt.Sprintf("%d observations of source %s", count[id], id))
		}
	}
	for _, s := range st.sources {
		if count[s.ID] == 0 {
			problems = append(problems, fmt.Sprintf("no observation of source %s", s.ID))
		}
	}
	return problems
}

// effectiveS is S at a cutoff: NULL when no target is active there
// (data-model §6.13, the unknown states).
func effectiveS(st *state, s string) string {
	if st.target == nil {
		return ""
	}
	return s
}

// Derive is the full derivation at in.AsOfSeq (data-model §6): delivery
// statuses, progress, flags, gaps, metrics and the initialization progress.
// It needs the complete input; an incomplete one is refused with an
// *InputError.
func Derive(in Input) (*Result, error) {
	problems := checkCommon(&in)
	if len(problems) > 0 {
		return nil, &InputError{Problems: problems}
	}
	st := at(&in.Facts, in.AsOfSeq)
	if problems := checkObservations(st, in.Current.Observations); len(problems) > 0 {
		return nil, &InputError{Problems: problems}
	}
	return assemble(st, in), nil
}

// ChangesSince is the --diff-since projection (data-model §6.13.1): the
// criteria whose delivery status or progress differs between the cutoffs
// since and in.AsOfSeq, in criterion key order, delivery_status before
// progress. Both cutoffs use the same S and settings; only the fact cutoff
// changes. The projection reads no source observation and no workspace,
// which it does not depend on (§6.3, §6.5), so it needs neither.
func ChangesSince(in Input, since int64) ([]Change, error) {
	problems := checkCommon(&in)
	switch {
	case since < 0:
		problems = append(problems, fmt.Sprintf("a negative start %d", since))
	case since > in.AsOfSeq:
		problems = append(problems, fmt.Sprintf("the start %d is after the cutoff %d", since, in.AsOfSeq))
	}
	if len(problems) > 0 {
		return nil, &InputError{Problems: problems}
	}
	settings := *in.Current.Settings
	before := map[string]projected{}
	for _, c := range projection(at(&in.Facts, since), in.Current.Snapshot.Commit, settings) {
		before[c.id] = c
	}
	var out []Change
	for _, c := range projection(at(&in.Facts, in.AsOfSeq), in.Current.Snapshot.Commit, settings) {
		b, existed := before[c.id]
		from := func(v string) string {
			if existed {
				return v
			}
			return ""
		}
		if !existed || b.status != c.status {
			out = append(out, Change{Criterion: c.key, Field: "delivery_status", From: from(string(b.status)), To: string(c.status)})
		}
		if !existed || b.progress != c.progress {
			out = append(out, Change{Criterion: c.key, Field: "progress", From: from(string(b.progress)), To: string(c.progress)})
		}
	}
	return out, nil
}

type projected struct {
	id, key  string
	status   CriterionStatus
	progress Progress
}

// projection is the delivery status and progress of each active criterion
// at a cutoff, in key order (requirements, then their criteria; a
// criterion's key starts with its requirement's number).
func projection(st *state, s string, settings Settings) []projected {
	s = effectiveS(st, s)
	var out []projected
	for _, req := range st.activeRequirements() {
		for _, e := range st.activeCriteria(req) {
			c := evalCriterion(st, e, req, s, settings)
			out = append(out, projected{id: e.ID, key: e.Key, status: c.status, progress: c.progress})
		}
	}
	return out
}

// activeRequirements are the active requirements at the cutoff, in key
// order.
func (st *state) activeRequirements() []*entityAt {
	return st.activeOf(RequirementEntity, nil)
}

// activeCriteria are a requirement's active criteria, in key order.
func (st *state) activeCriteria(req *entityAt) []*entityAt {
	return st.activeOf(CriterionEntity, req)
}

func (st *state) activeGoals() []*entityAt { return st.activeOf(GoalEntity, nil) }

func (st *state) activeOf(t EntityType, parent *entityAt) []*entityAt {
	var out []*entityAt
	for _, e := range st.entities {
		if e.Type != t || !e.active() || (parent != nil && e.RequirementID != parent.ID) {
			continue
		}
		out = append(out, e)
	}
	slices.SortFunc(out, func(a, b *entityAt) int { return cmp.Or(compareKeys(a.Key, b.Key), strings.Compare(a.ID, b.ID)) })
	return out
}
