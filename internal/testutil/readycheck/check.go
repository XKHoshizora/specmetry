package readycheck

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
)

// snapshot is what the checker knows at a cutoff, rebuilt by itself from
// the facts with seq <= asOf.
type snapshot struct {
	asOf      int64
	version   int
	target    *Target
	entity    map[string]Entity
	state     map[string]Change // entity ID -> its latest change
	scope     *Scope
	rs        []string // requirement IDs of the scope that are active
	as        []string // active criteria of RS
	acked     map[string]string
	inSet     []string // source IDs of the source set
	itemState map[string]Item
	produced  map[string]string // item ID -> entity ID its review produced
	runs      []Run
	decisions []Decision
}

func before(asOf int64) func(int64) bool { return func(seq int64) bool { return seq <= asOf } }

func rebuild(f Facts, asOf int64) *snapshot {
	upTo := before(asOf)
	k := &snapshot{
		asOf: asOf, entity: map[string]Entity{}, state: map[string]Change{}, acked: map[string]string{},
		itemState: map[string]Item{}, produced: map[string]string{},
	}
	for _, v := range f.Versions {
		if upTo(v.Seq) {
			k.version = max(k.version, v.Number)
		}
	}
	for i, t := range f.Targets {
		if !upTo(t.Created) || (t.Archived != 0 && upTo(t.Archived)) {
			continue
		}
		if k.target == nil || t.Created > k.target.Created || (t.Created == k.target.Created && t.ID > k.target.ID) {
			k.target = &f.Targets[i]
		}
	}
	for _, e := range f.Entities {
		k.entity[e.ID] = e
	}
	later := func(a, b Change) bool { // a is after b
		return cmp.Or(cmp.Compare(a.Seq, b.Seq), cmp.Compare(a.Index, b.Index), strings.Compare(a.ID, b.ID)) > 0
	}
	for _, c := range f.Changes {
		if !upTo(c.Seq) {
			continue
		}
		if prev, ok := k.state[c.Entity]; !ok || later(c, prev) {
			k.state[c.Entity] = c
		}
	}
	if k.target != nil {
		for i, s := range f.Scopes {
			if s.Target == k.target.ID && upTo(s.Seq) && (k.scope == nil || s.Seq > k.scope.Seq || (s.Seq == k.scope.Seq && s.ID > k.scope.ID)) {
				k.scope = &f.Scopes[i]
			}
		}
	}
	if k.scope != nil {
		for _, it := range f.ScopeItems {
			if it.Scope == k.scope.ID && k.isActive(it.Requirement, "requirement") && !slices.Contains(k.rs, it.Requirement) {
				k.rs = append(k.rs, it.Requirement)
			}
		}
	}
	for id, e := range k.entity {
		if e.Kind == "criterion" && k.isActive(id, "criterion") && slices.Contains(k.rs, e.Requirement) {
			k.as = append(k.as, id)
		}
	}
	slices.Sort(k.rs)
	slices.Sort(k.as)
	k.rebuildSources(f)
	for _, it := range f.Items {
		k.itemState[it.ID] = it
	}
	produced := map[string]Change{}
	for _, c := range f.Changes {
		if c.Item != "" && upTo(c.Seq) {
			if prev, ok := produced[c.Item]; !ok || later(prev, c) {
				produced[c.Item] = c
			}
		}
	}
	for item, c := range produced {
		k.produced[item] = c.Entity
	}
	for _, r := range f.Runs {
		if upTo(r.Seq) {
			k.runs = append(k.runs, r)
		}
	}
	slices.SortFunc(k.runs, func(a, b Run) int { return cmp.Or(cmp.Compare(a.Seq, b.Seq), strings.Compare(a.ID, b.ID)) })
	for _, d := range f.Decisions {
		if upTo(d.Seq) {
			k.decisions = append(k.decisions, d)
		}
	}
	slices.SortFunc(k.decisions, func(a, b Decision) int { return cmp.Or(cmp.Compare(a.Seq, b.Seq), strings.Compare(a.ID, b.ID)) })
	return k
}

func (k *snapshot) isActive(id, kind string) bool {
	e, known := k.entity[id]
	c, exists := k.state[id]
	return known && exists && e.Kind == kind && c.Status == "active"
}

// rebuildSources finds the source set and the hashes acknowledged at the
// cutoff: the last acknowledgement up to it, else the registration hash
// (the first acknowledgement's previous hash, or the row's hash).
func (k *snapshot) rebuildSources(f Facts) {
	acks := map[string][]Ack{}
	for _, a := range f.Acks {
		acks[a.Source] = append(acks[a.Source], a)
	}
	for _, s := range f.Sources {
		if s.Registered > k.asOf || (s.Removed != 0 && s.Removed <= k.asOf) || slices.Contains(k.inSet, s.ID) {
			continue
		}
		hash := s.Hash
		list := acks[s.ID]
		slices.SortFunc(list, func(a, b Ack) int { return cmp.Or(cmp.Compare(a.Seq, b.Seq), strings.Compare(a.ID, b.ID)) })
		if len(list) > 0 {
			hash = list[0].Previous
		}
		for _, a := range list {
			if a.Seq <= k.asOf {
				hash = a.Hash
			}
		}
		k.inSet = append(k.inSet, s.ID)
		k.acked[s.ID] = hash
	}
	slices.Sort(k.inSet)
}

// sourceSetHash recomputes the source set hash (data-model §4.3).
func (k *snapshot) sourceSetHash() string {
	var b strings.Builder
	for _, id := range k.inSet { // sorted by ID
		fmt.Fprintf(&b, "%s:%s\n", id, k.acked[id])
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func qualifies(r Run) bool {
	return r.HeadStart != "" && r.HeadStart == r.HeadEnd && r.TreeStart == "clean" && r.TreeEnd == "clean" &&
		r.FailureKind != "context_changed"
}

// latestOnS is V's run: the criterion's latest qualifying run on S for its
// revision and configuration; inconsistent reports mixed results there.
func (k *snapshot) latestOnS(ac string, s string) (v *Run, inconsistent bool) {
	c := k.state[ac]
	hash := ""
	if c.Check != nil {
		hash = ConfigHash(*c.Check)
	}
	passed, failed := false, false
	for i, r := range k.runs {
		if r.Criterion != ac || !qualifies(r) || r.HeadStart != s || r.Revision != c.Revision || r.ConfigHash != hash {
			continue
		}
		v = &k.runs[i]
		if r.Result == "pass" {
			passed = true
		} else {
			failed = true
		}
	}
	return v, passed && failed
}

func (k *snapshot) currentDecision(ac string) *Decision {
	var d *Decision
	for i, x := range k.decisions {
		if x.Criterion == ac && x.Revision == k.state[ac].Revision {
			d = &k.decisions[i]
		}
	}
	return d
}

func (k *snapshot) run(id string) *Run {
	for i := range k.runs {
		if k.runs[i].ID == id {
			return &k.runs[i]
		}
	}
	return nil
}

func sameInputs(a, b []InputEntry) bool {
	if a == nil || b == nil {
		return false
	}
	key := func(e InputEntry) string { return e.Path + "\x00" + e.Mode + "\x00" + e.Type + "\x00" + e.OID }
	set := func(es []InputEntry) []string {
		var out []string
		for _, e := range es {
			if !slices.Contains(out, key(e)) {
				out = append(out, key(e))
			}
		}
		slices.Sort(out)
		return out
	}
	return slices.Equal(set(a), set(b))
}

// accepted checks R6 (c) for one criterion and returns the grade the
// acceptance relies on: "" when it is not a normal acceptance.
func (k *snapshot) accepted(ac, s string, d *Decision) (ok bool, grade string, why string) {
	c := k.state[ac]
	v, inconsistent := k.latestOnS(ac, s)
	switch {
	case d.Kind == "waived":
		return true, "", ""
	case d.Kind != "accepted":
		return false, "", "the current decision is " + d.Kind
	case c.Verification != "automated":
		if d.Mode == "manual" && s != "" && d.Snapshot == s {
			return true, "", ""
		}
		return false, "", "the manual acceptance is not on S"
	case d.Mode == "exception":
		if s != "" && d.Snapshot == s && (v == nil || v.Result == "pass") {
			return true, "", ""
		}
		return false, "", "the exception is not on S, or V is a failure"
	case d.Mode != "normal":
		return false, "", "an unknown mode"
	case v == nil || v.Result != "pass":
		return false, "", "V is not a pass"
	case d.Snapshot == s:
		evidence := k.run(d.EvidenceID)
		if evidence == nil {
			return true, "none", ""
		}
		return true, evidence.Grade, ""
	}
	// Carried forward (data-model §6.2): only an isolated, consistent pass
	// with the same configuration and equal declared inputs.
	declared := c.Check != nil && len(c.Check.Inputs) > 0
	if v.Grade == "snapshot_isolated" && !inconsistent && d.ConfigHash == v.ConfigHash && (!declared || sameInputs(v.Inputs, d.Inputs)) {
		return true, v.Grade, ""
	}
	return false, "", "the acceptance of another snapshot is not carried forward"
}

// Check evaluates the ready predicate at a cutoff (acceptance-criteria
// §3.1): it holds when every condition R1 to R9 holds. R10's grade is
// recomputed for the output consistency check.
func Check(f Facts, cur Current, asOf int64) Verdict {
	k := rebuild(f, asOf)
	v := Verdict{Failed: map[Condition][]string{}, Grade: "none"}
	fail := func(c Condition, why string) { v.Failed[c] = append(v.Failed[c], why) }
	s := cur.S
	if k.target == nil {
		s = ""
	}

	if k.version == 0 {
		fail(R1, "no baseline version")
	}
	switch {
	case k.target == nil:
		fail(R2, "no active target")
	case s == "":
		fail(R2, "S is NULL")
	}
	switch {
	case k.scope == nil:
		fail(R3, "no scope confirmation")
	case k.scope.Version != k.version:
		fail(R3, fmt.Sprintf("the scope is of version %d, the baseline is %d", k.scope.Version, k.version))
	case k.scope.SourceSetHash != k.sourceSetHash():
		fail(R3, "the source set changed since the scope was confirmed")
	}
	if len(k.rs) == 0 {
		fail(R4, "RS is empty")
	}
	for _, req := range k.rs {
		if !slices.ContainsFunc(k.as, func(ac string) bool { return k.entity[ac].Requirement == req }) {
			fail(R5, k.entity[req].Key)
		}
	}
	allWaived := len(k.as) > 0
	normalGrades := []string{}
	for _, ac := range k.as {
		key := k.entity[ac].Key
		parent := k.state[k.entity[ac].Requirement]
		if k.state[ac].RequirementRevision != parent.Revision {
			fail(R6, key+": the parent requirement is revised")
			allWaived = false
			continue
		}
		d := k.currentDecision(ac)
		if d == nil {
			fail(R6, key+": no decision on its revision")
			allWaived = false
			continue
		}
		if d.Kind != "waived" {
			allWaived = false
		}
		ok, grade, why := k.accepted(ac, s, d)
		if !ok {
			fail(R6, key+": "+why)
		} else if grade != "" {
			normalGrades = append(normalGrades, grade)
		}
	}
	if allWaived {
		fail(R7, "every criterion of AS is waived")
	}
	for _, q := range f.Questions {
		if q.Created > asOf || (q.Answered != 0 && q.Answered <= asOf) || !q.Blocking {
			continue
		}
		about, counts := q.About, true
		if q.Item != "" {
			item, known := k.itemState[q.Item]
			reviewed := known && item.Reviewed != 0 && item.Reviewed <= asOf
			switch {
			case reviewed && (item.State == "accepted" || item.State == "accepted_with_edits"):
				about = k.produced[q.Item]
				counts = about != ""
			default:
				counts = false // about a pending item, or a rejected one
			}
		}
		if counts && (about == "" || slices.Contains(k.rs, about) || slices.Contains(k.as, about)) {
			fail(R8, q.Key)
		}
	}
	for _, id := range k.inSet {
		if got, ok := cur.Observations[id]; !ok || got == "" || got != k.acked[id] {
			fail(R9, id)
		}
	}
	for _, why := range v.Failed {
		slices.Sort(why) // the verdict does not depend on the row order
	}
	for _, g := range normalGrades {
		switch {
		case g == "observed":
			v.Grade = "observed"
		case g == "snapshot_isolated" && v.Grade == "none":
			v.Grade = "snapshot_isolated"
		}
	}
	return v
}

// Consistent is R10 for a ready verdict: an output's ready verdict agrees
// with the checker's, and when ready its delivery evidence grade too. The
// grade of an output that is not ready is checked by GradeConsistent.
func Consistent(v Verdict, ready bool, grade string) error {
	if v.Ready() != ready {
		return fmt.Errorf("the output says ready = %v, the checker %v (failed %v)", ready, v.Ready(), v.Failed)
	}
	if ready {
		return GradeConsistent(v, grade)
	}
	return nil
}

// GradeConsistent compares an output's delivery evidence grade with the
// checker's for any output, ready or not: the grade of the runs the normal
// acceptances of AS that apply on S rely on (data-model §6.8), which a
// delivery that is not ready also reports.
func GradeConsistent(v Verdict, grade string) error {
	if grade != v.Grade {
		return fmt.Errorf("the output's evidence grade is %s, the checker recomputes %s", grade, v.Grade)
	}
	return nil
}

// ConfigHash is a check configuration's hash (data-model §4.4): the sha256
// of the RFC 8785 form of {"command","inputs","shell","workdir"}, inputs
// sorted, a shell not set as null.
func ConfigHash(c CheckConfig) string {
	inputs := slices.Clone(c.Inputs)
	slices.Sort(inputs)
	quoted := make([]string, len(inputs))
	for i, in := range inputs {
		quoted[i] = canonical(in)
	}
	shell := "null"
	if c.Shell != nil {
		shell = canonical(*c.Shell)
	}
	doc := `{"command":` + canonical(c.Command) + `,"inputs":[` + strings.Join(quoted, ",") + `],"shell":` + shell +
		`,"workdir":` + canonical(c.Workdir) + `}`
	sum := sha256.Sum256([]byte(doc))
	return hex.EncodeToString(sum[:])
}

// canonical is a JSON string as RFC 8785 writes it.
func canonical(s string) string {
	escapes := map[rune]string{'"': `\"`, '\\': `\\`, '\b': `\b`, '\f': `\f`, '\n': `\n`, '\r': `\r`, '\t': `\t`}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		e, escaped := escapes[r]
		switch {
		case escaped:
			b.WriteString(e)
		case r < 0x20:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
