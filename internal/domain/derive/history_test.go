package derive_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/toderive"
)

// at derives the fixture's facts at a cutoff, with the observations of the
// sources in the source set at that cutoff.
func (f *fixture) at(t *testing.T, asOf int64, observe map[string]string) *derive.Result {
	t.Helper()
	cur := f.current()
	cur.Observations = nil
	for id, hash := range observe {
		cur.Observations = append(cur.Observations, factgen.Observation{SourceID: id, SHA256: hash})
	}
	return mustDerive(t, toderive.Input(legalDump(t, f.h), cur, asOf))
}

// data-model §6.13: the state at as_of_seq is rebuilt from the facts; the
// current projection is never used for an earlier cutoff.
func TestStateAtCutoff(t *testing.T) {
	f := newFixture(t)
	brief := map[string]string{f.src: factgen.Hash("brief v1")}
	created := f.h.Seq()
	run := f.pass(f.ac, commitA)
	accepted := f.accept(f.ac, commitA, run)
	acceptedAt := f.h.Seq()
	f.h.Apply(func(r *factgen.Review) {
		r.Update(f.ac, true, func(x *factgen.Fields) { x.Statement = "stricter" })
		r.Scope(f.req)
	})
	revisedAt := f.h.Seq()

	before := f.at(t, created-1, brief)
	if len(before.Requirements) != 0 || before.BaselineVersion != 0 {
		t.Errorf("before the review: version %d, requirements %+v", before.BaselineVersion, before.Requirements)
	}
	atAccept := criterionOf(t, f.at(t, acceptedAt, brief), "AC-001-1")
	if atAccept.Revision != 1 || atAccept.Status != derive.CriterionAccepted || atAccept.CurrentDecision == nil || atAccept.CurrentDecision.ID != accepted {
		t.Errorf("at the acceptance: %+v", atAccept)
	}
	if atAccept.SnapshotRun == nil || atAccept.SnapshotRun.ID != run || !atAccept.SnapshotRun.Qualifying {
		t.Errorf("snapshot run = %+v", atAccept.SnapshotRun)
	}
	now := criterionOf(t, f.at(t, revisedAt, brief), "AC-001-1")
	if now.Revision != 2 || now.Status != derive.CriterionNeedsReverification || now.Reason != "ac_revised" || now.CurrentDecision != nil {
		t.Errorf("after the revision: %+v", now)
	}

	// Tampering with the current projection (the entity rows' mutable
	// columns, a question's backfilled link) changes nothing at any cutoff.
	// A source row's hash is not a projection: without acknowledgements it
	// is the hash at registration.
	items, _ := f.h.Propose(1, factgen.QuestionSpec{Blocking: true, AboutItem: 1})
	f.h.Review(func(r *factgen.Review) { r.AddGoal("later"); r.FromItem(items[0]) })
	d := legalDump(t, f.h)
	for i := range d.Criteria {
		d.Criteria[i].Status, d.Criteria[i].Revision, d.Criteria[i].RequirementRevision = "retired", 9, 9
	}
	for i := range d.Questions {
		d.Questions[i].RelatedType, d.Questions[i].RelatedID = ptr("requirement"), ptr(f.req)
	}
	for _, asOf := range []int64{acceptedAt, revisedAt, f.h.Seq()} {
		cur := f.current()
		got := mustDerive(t, toderive.Input(d, cur, asOf))
		want := mustDerive(t, toderive.Input(legalDump(t, f.h), cur, asOf))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("at %d the current projection changed the result", asOf)
		}
	}
}

// F5 positive case, data-model §6.13: a review shares one seq between the
// version, its changes, the references and the attached scope; as_of_seq
// at that seq keeps them all, one less keeps none.
func TestSharedSeqReview(t *testing.T) {
	h := factgen.New(projectID)
	h.SetTarget("default", "refs/heads/main")
	src := h.AddSource("docs/a.md", factgen.Hash("a"))
	items, _ := h.Propose(6)
	var reqs []string
	seq := h.Review(func(r *factgen.Review) {
		for i := range 3 {
			req := r.AddRequirement("", "must")
			r.FromItem(items[2*i])
			r.AddCriterion(req, "manual", nil)
			r.FromItem(items[2*i+1])
			r.Ref(req, src, "verified")
			reqs = append(reqs, req)
		}
		r.Scope(reqs...)
	})
	obs := h.Current(commitA, "in_development")
	at := mustDerive(t, toderive.Input(legalDump(t, h), obs, seq))
	if at.BaselineVersion != 1 || len(at.Requirements) != 3 || at.Scope == nil || at.Scope.Stale || len(at.Scope.Requirements) != 3 {
		t.Fatalf("at the review: version %d, %d requirements, scope %+v", at.BaselineVersion, len(at.Requirements), at.Scope)
	}
	before := mustDerive(t, toderive.Input(legalDump(t, h), obs, seq-1))
	if before.BaselineVersion != 0 || len(before.Requirements) != 0 || before.Scope != nil {
		t.Errorf("before the review: version %d, requirements %d, scope %+v", before.BaselineVersion, len(before.Requirements), before.Scope)
	}
}

// data-model §6.13: the acknowledged hash at a cutoff comes from the
// acknowledgements, and the source set hash is recomputed from it.
func TestSourcesAtCutoff(t *testing.T) {
	f := newFixture(t)
	f.accept(f.ac, commitA, f.pass(f.ac, commitA))
	ready := f.h.Seq()
	f.h.AckSource(f.src, factgen.Hash("brief v2"))
	f.h.ConfirmScope(f.req)
	acked := f.h.Seq()
	f.h.RemoveSource(f.src)
	removed := f.h.Seq()

	// At the first cutoff the source is unchanged against its first hash:
	// the row's latest hash would make it look changed and the scope stale.
	res := f.at(t, ready, map[string]string{f.src: factgen.Hash("brief v1")})
	if res.Delivery.Status != derive.TargetReady || res.Sources[0].AckedSHA256 != factgen.Hash("brief v1") {
		t.Errorf("at %d: %s, sources %+v, gaps %v", ready, res.Delivery.Status, res.Sources, gapTypes(res.Gaps))
	}
	res = f.at(t, acked, map[string]string{f.src: factgen.Hash("brief v2")})
	if res.Delivery.Status != derive.TargetReady || res.Sources[0].AckedSHA256 != factgen.Hash("brief v2") {
		t.Errorf("at %d: %s, sources %+v", acked, res.Delivery.Status, res.Sources)
	}
	res = f.at(t, removed, nil)
	if len(res.Sources) != 0 || res.Delivery.Status != derive.TargetNeedsReverification {
		t.Errorf("after the removal: sources %+v, %s", res.Sources, res.Delivery.Status)
	}
}

// data-model §6.13, §4.8: a question about a proposal item is linked to the
// entity only from the review that accepted the item, whatever the current
// (backfilled) link says.
func TestQuestionLinkAtCutoff(t *testing.T) {
	f := readyFixture(t)
	brief := map[string]string{f.src: factgen.Hash("brief v1")}
	items, _ := f.h.Propose(2, factgen.QuestionSpec{Blocking: true, AboutItem: 1})
	proposed := f.h.Seq()
	f.h.Review(func(r *factgen.Review) {
		n := r.AddRequirement(f.goal, "must")
		r.FromItem(items[0])
		r.AddCriterion(n, "manual", nil)
		r.FromItem(items[1])
		r.Scope(f.req, n)
	})
	reviewed := f.h.Seq()
	if g := gapsOf(f.at(t, proposed, brief), derive.GapBlockingQuestion); len(g) != 0 {
		t.Errorf("before the review: %+v", g)
	}
	if g := gapsOf(f.at(t, proposed, brief), derive.GapPendingReview); len(g) != 1 {
		t.Errorf("before the review the item is pending: %+v", g)
	}
	if g := gapsOf(f.at(t, reviewed, brief), derive.GapBlockingQuestion); len(g) != 1 {
		t.Errorf("after the review: %+v", g)
	}
	res := f.at(t, reviewed, brief)
	if len(res.OpenQuestions) != 1 || res.OpenQuestions[0].About != "REQ-002" || !res.OpenQuestions[0].Blocking {
		t.Errorf("open questions = %+v", res.OpenQuestions)
	}
}

// data-model §6.13: the active target is the one whose interval contains
// the cutoff, and a new target does not inherit the old one's scope.
func TestTargetAtCutoff(t *testing.T) {
	f := readyFixture(t)
	ready := f.h.Seq()
	f.h.SetTarget("release", "refs/heads/release")
	set := f.h.Seq()
	brief := map[string]string{f.src: factgen.Hash("brief v1")}
	if res := f.at(t, ready, brief); res.Target == nil || res.Target.Name != "default" || res.Delivery.Status != derive.TargetReady {
		t.Errorf("before: target %+v, %s", res.Target, res.Delivery.Status)
	}
	res := f.at(t, set, brief)
	if res.Target == nil || res.Target.Name != "release" || res.Scope != nil {
		t.Errorf("after: target %+v, scope %+v", res.Target, res.Scope)
	}
	if got := gapTypes(res.Delivery.BlockingGaps); !slices.Equal(got, []derive.GapType{derive.GapNoScope}) {
		t.Errorf("after: blocking %v", got)
	}
}

// data-model §6.13: facts after the cutoff do not count: a later decision,
// run, answer or review of an item.
func TestLaterFactsDoNotCount(t *testing.T) {
	f := newFixture(t)
	items, qs := f.h.Propose(1, factgen.QuestionSpec{Blocking: true})
	cutoff := f.h.Seq()
	f.h.Answer(qs[0])
	f.h.Review(func(r *factgen.Review) { r.RejectItem(items[0]) })
	f.accept(f.ac, commitA, f.pass(f.ac, commitA))
	res := f.at(t, cutoff, map[string]string{f.src: factgen.Hash("brief v1")})
	c := criterionOf(t, res, "AC-001-1")
	if c.Status != derive.CriterionInProgress || c.SnapshotRun != nil || c.CurrentDecision != nil {
		t.Errorf("criterion = %+v", c)
	}
	if res.Metrics.OpenQuestions != 1 || res.Metrics.PendingReviewItems != 1 || len(gapsOf(res, derive.GapBlockingQuestion)) != 1 {
		t.Errorf("metrics = %+v", res.Metrics)
	}
}
