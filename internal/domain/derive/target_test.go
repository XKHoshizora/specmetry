package derive_test

import (
	"slices"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/toderive"
)

// readyFixture is the fixture with its criterion accepted on S: ready.
func readyFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	f.accept(f.ac, commitA, f.pass(f.ac, commitA))
	return f
}

// MVP-ST-03: the six rules of the target delivery table (data-model §6.8),
// each with every blocking gap listed.
func TestTargetDeliveryStatus(t *testing.T) {
	cases := []struct {
		name     string
		setup    func(t *testing.T, f *fixture) *fixture
		edit     func(f *fixture, cur *factgen.Current)
		want     derive.TargetStatus
		blocking []derive.GapType
	}{
		{"1: no baseline", func(*testing.T, *fixture) *fixture {
			f := &fixture{t: t, h: factgen.New(projectID), s: commitA, stage: "in_development"}
			f.h.SetTarget("default", "refs/heads/main")
			return f
		}, nil, derive.TargetNotAssessed, []derive.GapType{derive.GapNoBaseline, derive.GapNoScope}},
		{"1: no target", func(*testing.T, *fixture) *fixture {
			f := &fixture{t: t, h: factgen.New(projectID), stage: "in_development"}
			f.h.Apply(func(r *factgen.Review) { r.AddCriterion(r.AddRequirement("", "must"), "manual", nil) })
			return f
		}, nil, derive.TargetNotAssessed, []derive.GapType{derive.GapNoTarget}},
		{"1: the target does not resolve", func(_ *testing.T, f *fixture) *fixture {
			f.s = ""
			return f
		}, nil, derive.TargetNotAssessed, []derive.GapType{derive.GapTargetUnresolved}},
		{"1: no scope for the new target", func(_ *testing.T, f *fixture) *fixture {
			f.h.SetTarget("release", "refs/heads/release")
			return f
		}, nil, derive.TargetNotAssessed, []derive.GapType{derive.GapNoScope}},
		{"1: every requirement of the scope is retired", func(_ *testing.T, f *fixture) *fixture {
			f.h.Apply(func(r *factgen.Review) { r.Retire(f.req) })
			return f
		}, nil, derive.TargetNotAssessed, []derive.GapType{derive.GapEmptyScope, derive.GapScopeStale}},
		{"2: a requirement of the scope without criteria", func(t *testing.T, _ *fixture) *fixture {
			f := readyFixture(t)
			var r string
			f.h.Apply(func(rv *factgen.Review) { r = rv.AddRequirement(f.goal, "must"); rv.Scope(f.req, r) })
			return f
		}, nil, derive.TargetInProgress, []derive.GapType{derive.GapNeedsCriteria}},
		{"2: a criterion in progress, with every other blocking gap", func(_ *testing.T, f *fixture) *fixture {
			f.h.Apply(func(r *factgen.Review) { r.AddCriterion(f.req, "manual", nil) }) // the scope is now stale
			return f
		}, editedSource, derive.TargetInProgress, []derive.GapType{derive.GapScopeStale, derive.GapSourceChanged, derive.GapNotVerified, derive.GapNotVerified}},
		{"2: a failing criterion", func(_ *testing.T, f *fixture) *fixture { f.fail(f.ac, commitA); return f }, nil,
			derive.TargetInProgress, []derive.GapType{derive.GapFailing}},
		{"2: a rejected criterion", func(_ *testing.T, f *fixture) *fixture {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA})
			return f
		}, nil, derive.TargetInProgress, []derive.GapType{derive.GapRejected}},
		{"2: a blocking question", func(t *testing.T, _ *fixture) *fixture {
			f := readyFixture(t)
			f.h.Propose(0, factgen.QuestionSpec{Blocking: true})
			return f
		}, nil, derive.TargetInProgress, []derive.GapType{derive.GapBlockingQuestion}},
		{"3: CX-18: the scope is stale", func(t *testing.T, _ *fixture) *fixture {
			f := readyFixture(t)
			f.h.Apply(func(r *factgen.Review) { r.AddRequirement(f.goal, "must") })
			return f
		}, nil, derive.TargetNeedsReverification, []derive.GapType{derive.GapScopeStale}},
		{"3: CX-18: a new source makes the scope stale", func(t *testing.T, _ *fixture) *fixture {
			f := readyFixture(t)
			f.h.AddSource("docs/more.md", factgen.Hash("more"))
			return f
		}, nil, derive.TargetNeedsReverification, []derive.GapType{derive.GapScopeStale}},
		{"3: a source changed on disk", func(t *testing.T, _ *fixture) *fixture { return readyFixture(t) }, editedSource,
			derive.TargetNeedsReverification, []derive.GapType{derive.GapSourceChanged}},
		{"3: a criterion needs reverification", func(t *testing.T, _ *fixture) *fixture {
			f := readyFixture(t)
			f.s = commitB
			return f
		}, nil, derive.TargetNeedsReverification, []derive.GapType{derive.GapNeedsReverification}},
		{"4: a criterion awaits review", func(_ *testing.T, f *fixture) *fixture { f.pass(f.ac, commitA); return f }, nil,
			derive.TargetReadyForReview, []derive.GapType{derive.GapAwaitingReview}},
		{"5: CX-06: the scope is all waived", func(_ *testing.T, f *fixture) *fixture {
			f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "waived"})
			return f
		}, nil, derive.TargetNotAssessed, []derive.GapType{derive.GapAllWaived}},
		{"6: ready", func(t *testing.T, _ *fixture) *fixture { return readyFixture(t) }, nil, derive.TargetReady, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := c.setup(t, newFixture(t))
			cur := f.current()
			if c.edit != nil {
				c.edit(f, &cur)
			}
			res := mustDerive(t, toderive.Input(legalDump(t, f.h), cur, f.h.Seq()))
			if res.Delivery.Status != c.want {
				t.Errorf("delivery = %s, want %s", res.Delivery.Status, c.want)
			}
			if got := gapTypes(res.Delivery.BlockingGaps); !slices.Equal(got, c.blocking) {
				t.Errorf("blocking gaps = %v, want %v", got, c.blocking)
			}
			var blocking []derive.Gap
			for _, g := range res.Gaps {
				if g.Blocking {
					blocking = append(blocking, g)
				}
			}
			if !slices.Equal(blocking, res.Delivery.BlockingGaps) {
				t.Errorf("blocking gaps %v are not the blocking ones of %v", res.Delivery.BlockingGaps, res.Gaps)
			}
			// The invariant of §6.8 (MVP-ST-13).
			if (res.Delivery.Status == derive.TargetReady) != (len(res.Delivery.BlockingGaps) == 0) {
				t.Errorf("ready = %v with %d blocking gaps", res.Delivery.Status == derive.TargetReady, len(res.Delivery.BlockingGaps))
			}
		})
	}
}

// editedSource observes the fixture's source with other content.
func editedSource(f *fixture, cur *factgen.Current) {
	for i := range cur.Observations {
		if cur.Observations[i].SourceID == f.src {
			cur.Observations[i].SHA256 = factgen.Hash("edited")
		}
	}
}

// data-model §6.8: the evidence grade of the delivery judgement.
func TestDeliveryEvidenceGrade(t *testing.T) {
	t.Run("observed", func(t *testing.T) {
		if g := readyFixture(t).derive(t).Delivery.EvidenceGrade; g != derive.Observed {
			t.Errorf("grade = %s, want observed", g)
		}
	})
	t.Run("CX-26: none before the acceptance", func(t *testing.T) {
		f := newFixture(t)
		f.pass(f.ac, commitA)
		if g := f.derive(t).Delivery.EvidenceGrade; g != derive.NoEvidence {
			t.Errorf("grade = %s, want none", g)
		}
	})
	t.Run("none for manual, exception and waived acceptances", func(t *testing.T) {
		f := newFixture(t)
		m, w := manualCriterion(f), manualCriterion(f)
		f.h.ConfirmScope(f.req)
		f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
		f.h.Decide(m, factgen.DecisionSpec{Decision: "accepted", Mode: "manual", Snapshot: commitA})
		f.h.Decide(w, factgen.DecisionSpec{Decision: "waived"})
		res := f.derive(t)
		if res.Delivery.Status != derive.TargetReady || res.Delivery.EvidenceGrade != derive.NoEvidence {
			t.Errorf("delivery = %s/%s, want ready/none", res.Delivery.Status, res.Delivery.EvidenceGrade)
		}
	})
	t.Run("the weakest grade wins", func(t *testing.T) {
		f := newFixture(t)
		f.accept(f.ac, commitA, f.h.Run(f.ac, factgen.RunSpec{Head: commitA, Grade: "snapshot_isolated"}))
		a := addAuto(f, f.req)
		f.accept(a, commitA, f.pass(a, commitA))
		if g := f.derive(t).Delivery.EvidenceGrade; g != derive.Observed {
			t.Errorf("grade = %s, want observed", g)
		}
	})
	t.Run("a run after an exception does not change its grade (PD-17)", func(t *testing.T) {
		f := newFixture(t)
		f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
		run := f.pass(f.ac, commitA)
		res := f.derive(t)
		c := criterionOf(t, res, f.h.Key(f.ac))
		if c.Status != derive.CriterionAccepted || c.Reason != "exception" || res.Delivery.EvidenceGrade != derive.NoEvidence {
			t.Errorf("criterion %s/%s, delivery grade %s", c.Status, c.Reason, res.Delivery.EvidenceGrade)
		}
		if c.SnapshotRun == nil || c.SnapshotRun.ID != run {
			t.Errorf("snapshot run = %+v, want %s", c.SnapshotRun, run)
		}
		if c.CurrentDecision == nil || c.CurrentDecision.EvidenceGrade != derive.NoEvidence || c.CurrentDecision.Mode != "exception" {
			t.Errorf("decision = %+v", c.CurrentDecision)
		}
	})
}

// data-model §6.10: gaps, their blocking, the scope filter and the order.
func TestGaps(t *testing.T) {
	t.Run("out of scope: no gaps; in scope: blocking", func(t *testing.T) {
		f := newFixture(t)
		var out string
		f.h.Apply(func(r *factgen.Review) { out = r.AddRequirement(f.goal, "should") })
		f.h.ConfirmScope(f.req)
		res := f.derive(t)
		for _, g := range res.Gaps {
			if g.TargetKey == f.h.Key(out) {
				t.Errorf("gap %+v of a requirement out of scope", g)
			}
		}
		g := gapsOf(res, derive.GapNotVerified)
		if len(g) != 1 || !g[0].Blocking || g[0].TargetKey != f.h.Key(f.ac) || g[0].Reason != "unknown" || g[0].NextAction != "实现并验证" {
			t.Errorf("NOT_VERIFIED = %+v", g)
		}
	})
	t.Run("without a scope: listed, not blocking", func(t *testing.T) {
		f := &fixture{t: t, h: factgen.New(projectID), s: commitA, stage: "in_development"}
		f.h.SetTarget("default", "refs/heads/main")
		var req string
		f.h.Apply(func(r *factgen.Review) {
			req = r.AddRequirement("", "must")
			r.AddRequirement("", "must")
			r.AddCriterion(req, "automated", nil)
		})
		res := f.derive(t)
		for _, g := range res.Gaps {
			if g.Type != derive.GapNoScope && g.Blocking {
				t.Errorf("gap %+v blocks without a scope", g)
			}
		}
		want := []derive.GapType{derive.GapNoScope, derive.GapInventoryIncomplete, derive.GapNeedsCriteria, derive.GapNoCheck}
		if got := gapTypes(res.Gaps); !slices.Equal(got, want) {
			t.Errorf("gaps = %v", got)
		}
	})
	t.Run("without a scope, project-level gaps keep their table's blocking (review 046 P1)", func(t *testing.T) {
		f := &fixture{t: t, h: factgen.New(projectID), s: commitA, stage: "in_development", inventorySkipped: true}
		f.h.SetTarget("default", "refs/heads/main")
		f.src = f.h.AddSource("docs/a.md", factgen.Hash("a"))
		f.h.Apply(func(r *factgen.Review) { r.AddCriterion(r.AddRequirement("", "must"), "manual", nil) })
		cur := f.current()
		cur.Observations[0].SHA256 = factgen.Hash("edited")
		res := mustDerive(t, toderive.Input(legalDump(t, f.h), cur, f.h.Seq()))
		want := []derive.GapType{derive.GapNoScope, derive.GapSourceChanged}
		if got := gapTypes(res.Delivery.BlockingGaps); !slices.Equal(got, want) {
			t.Errorf("blocking = %v, want %v", got, want)
		}
		if res.Delivery.Status != derive.TargetNotAssessed {
			t.Errorf("delivery = %s", res.Delivery.Status)
		}
	})
	t.Run("a stale scope still filters and blocks", func(t *testing.T) {
		f := newFixture(t)
		var out string
		f.h.Apply(func(r *factgen.Review) { out = r.AddRequirement(f.goal, "should") })
		res := f.derive(t)
		for _, g := range res.Gaps {
			if g.TargetKey == f.h.Key(out) {
				t.Errorf("gap %+v out of the stale scope", g)
			}
		}
		if g := gapsOf(res, derive.GapNotVerified); len(g) != 1 || !g[0].Blocking {
			t.Errorf("NOT_VERIFIED = %+v", g)
		}
	})
	t.Run("one state gap per criterion; none when accepted, waived or not assessed", func(t *testing.T) {
		f := readyFixture(t)
		f.inventorySkipped = true
		w := addAuto(f, f.req)
		f.h.ConfirmScope(f.req)
		f.h.Decide(w, factgen.DecisionSpec{Decision: "waived"})
		res := f.derive(t)
		if len(res.Gaps) != 0 {
			t.Errorf("gaps = %+v", res.Gaps)
		}
		f.s = ""
		res = f.derive(t)
		if got := gapTypes(res.Gaps); !slices.Equal(got, []derive.GapType{derive.GapTargetUnresolved}) {
			t.Errorf("gaps = %v", got)
		}
	})
	t.Run("the state gap types and reasons", func(t *testing.T) {
		f := newFixture(t)
		mk := func(fn func(a string)) string { a := addAuto(f, f.req); fn(a); return a }
		failing := mk(func(a string) { f.fail(a, commitA) })
		rejected := mk(func(a string) { f.h.Decide(a, factgen.DecisionSpec{Decision: "rejected", Snapshot: commitA}) })
		claimed := mk(func(a string) { f.h.Claims("sync", factgen.ClaimSpec{Criterion: a, Assertion: "done"}) })
		reverify := mk(func(a string) { f.pass(a, commitB) })
		review := mk(func(a string) { f.pass(a, commitA) })
		var nocheck string
		f.h.Apply(func(r *factgen.Review) { nocheck = r.AddCriterion(f.req, "automated", nil) })
		var revised string
		f.h.Apply(func(r *factgen.Review) {
			revised = r.AddRequirement(f.goal, "must")
			r.Scope(f.req, revised)
		})
		pr := addAuto(f, revised)
		f.h.Apply(func(r *factgen.Review) { r.Update(revised, true, nil) })
		f.h.ConfirmScope(f.req, revised)
		res := f.derive(t)
		want := map[string][2]string{
			failing: {"FAILING", "check_failed"}, rejected: {"REJECTED", "rework_requested"},
			claimed: {"CLAIMED_WITHOUT_EVIDENCE", "claimed_without_evidence"}, reverify: {"NEEDS_REVERIFICATION", "code_changed"},
			review: {"AWAITING_REVIEW", "first_review"}, nocheck: {"NO_CHECK", "no_check"}, pr: {"PARENT_REVISED", "parent_revised"},
			f.ac: {"NOT_VERIFIED", "unknown"},
		}
		for id, w := range want {
			var state []derive.Gap
			for _, g := range res.Gaps {
				if g.TargetKey == f.h.Key(id) && g.Type != derive.GapConflict {
					state = append(state, g)
				}
			}
			if len(state) != 1 || string(state[0].Type) != w[0] || state[0].Reason != w[1] {
				t.Errorf("%s: gaps %+v, want one %s/%s", f.h.Key(id), state, w[0], w[1])
			}
		}
	})
	t.Run("order: blocking first, then the table order, then the key", func(t *testing.T) {
		f := newFixture(t)
		f.stage = "docs_only"
		for range 10 {
			addAuto(f, f.req)
		}
		f.h.Propose(1)
		f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
		res := f.derive(t)
		var keys []string
		for _, g := range res.Gaps {
			keys = append(keys, string(g.Type)+" "+g.TargetKey)
		}
		// AC-001-2 .. AC-001-11 are NOT_VERIFIED in key order (2, 3, ..., 10,
		// 11), then the warnings.
		var gotNV []string
		for _, g := range res.Gaps {
			if g.Type == derive.GapNotVerified {
				gotNV = append(gotNV, g.TargetKey)
			}
		}
		wantNV := []string{"AC-001-2", "AC-001-3", "AC-001-4", "AC-001-5", "AC-001-6", "AC-001-7", "AC-001-8", "AC-001-9", "AC-001-10", "AC-001-11"}
		if !slices.Equal(gotNV, wantNV) {
			t.Errorf("NOT_VERIFIED order = %v", gotNV)
		}
		last := len(res.Gaps) - 1
		if res.Gaps[last-1].Type != derive.GapPendingReview || res.Gaps[last].Type != derive.GapExceptionAcceptance {
			t.Errorf("gaps = %v", keys)
		}
		if res.Gaps[last].Reason != "exception" || res.Gaps[last].NextAction != "在快照上补充自动化检查" || res.Gaps[last].Blocking {
			t.Errorf("EXCEPTION_ACCEPTANCE = %+v", res.Gaps[last])
		}
	})
	t.Run("warnings never block", func(t *testing.T) {
		f := newFixture(t)
		f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"})
		f.pass(f.ac, commitA)
		f.fail(f.ac, commitA)
		f.h.Propose(2)
		for _, typ := range []derive.GapType{derive.GapConflict, derive.GapInconsistentResults, derive.GapPendingReview, derive.GapInventoryIncomplete} {
			g := gapsOf(f.derive(t), typ)
			if len(g) != 1 || g[0].Blocking {
				t.Errorf("%s = %+v", typ, g)
			}
		}
	})
	t.Run("SOURCE_CHANGED names the source and the change", func(t *testing.T) {
		f := readyFixture(t)
		other := f.h.AddSource("docs/other.md", factgen.Hash("other"))
		f.h.ConfirmScope(f.req)
		cur := f.current()
		for i := range cur.Observations {
			switch cur.Observations[i].SourceID {
			case f.src:
				cur.Observations[i].SHA256 = ""
			case other:
				cur.Observations[i].SHA256 = factgen.Hash("other edited")
			}
		}
		res := mustDerive(t, toderive.Input(legalDump(t, f.h), cur, f.h.Seq()))
		got := gapsOf(res, derive.GapSourceChanged)
		if len(got) != 2 || got[0].TargetKey != "docs/brief.md" || got[0].Reason != "missing" ||
			got[1].TargetKey != "docs/other.md" || got[1].Reason != "changed" || !got[0].Blocking {
			t.Errorf("SOURCE_CHANGED = %+v", got)
		}
		if len(res.Sources) != 2 || res.Sources[0].State != derive.SourceMissing || res.Sources[1].State != derive.SourceChanged ||
			res.Sources[1].CurrentSHA256 != factgen.Hash("other edited") || res.Sources[0].CurrentSHA256 != "" {
			t.Errorf("sources = %+v", res.Sources)
		}
	})
	t.Run("INVENTORY_INCOMPLETE", func(t *testing.T) {
		for _, c := range []struct {
			name         string
			stage        string
			skip, audit  bool
			wantWarning  bool
			wantProgress derive.InitProgress
		}{
			{"in development, not inventoried", "in_development", false, false, true, derive.ScopeConfirmed},
			{"docs_only", "docs_only", false, false, false, derive.Inventoried},
			{"inventory skipped", "in_development", true, false, false, derive.Inventoried},
			{"fully audited", "in_development", false, true, false, derive.Inventoried},
		} {
			f := newFixture(t)
			f.stage = c.stage
			if c.audit {
				f.h.Claims("audit", factgen.ClaimSpec{Criterion: f.ac, Assertion: "unknown"})
			}
			cur := f.current()
			cur.InventorySkipped = c.skip
			res := mustDerive(t, toderive.Input(legalDump(t, f.h), cur, f.h.Seq()))
			if got := len(gapsOf(res, derive.GapInventoryIncomplete)) == 1; got != c.wantWarning {
				t.Errorf("%s: INVENTORY_INCOMPLETE = %v", c.name, got)
			}
			if res.InitProgress != c.wantProgress {
				t.Errorf("%s: init progress = %s, want %s", c.name, res.InitProgress, c.wantProgress)
			}
		}
	})
}

// data-model §6.8, §4.8: which open blocking questions block.
func TestBlockingQuestions(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(f *fixture)
		blocks bool
	}{
		{"about a requirement of the scope", func(f *fixture) {
			f.h.Propose(0, factgen.QuestionSpec{Blocking: true, AboutEntity: f.req})
		}, true},
		{"about a criterion of the scope", func(f *fixture) {
			f.h.Propose(0, factgen.QuestionSpec{Blocking: true, AboutEntity: f.ac})
		}, true},
		{"about nothing", func(f *fixture) { f.h.Propose(0, factgen.QuestionSpec{Blocking: true}) }, true},
		{"about a goal", func(f *fixture) {
			// The column allows it, so the derivation must treat it, but no
			// production write does: proposal.v1's about is a requirement or
			// criterion key or a local ref (review 047 B1-R3).
			f.outsideLegalDomain = "a question about a goal"
			f.h.Propose(0, factgen.QuestionSpec{Blocking: true, AboutEntity: f.goal})
		}, false},
		{"about a requirement out of the scope", func(f *fixture) {
			var out string
			f.h.Apply(func(r *factgen.Review) { out = r.AddRequirement("", "could") })
			f.h.ConfirmScope(f.req)
			f.h.Propose(0, factgen.QuestionSpec{Blocking: true, AboutEntity: out})
		}, false},
		{"not blocking", func(f *fixture) { f.h.Propose(0, factgen.QuestionSpec{AboutEntity: f.req}) }, false},
		{"answered", func(f *fixture) {
			_, qs := f.h.Propose(0, factgen.QuestionSpec{Blocking: true})
			f.h.Answer(qs[0])
		}, false},
		{"about a pending item", func(f *fixture) {
			f.h.Propose(1, factgen.QuestionSpec{Blocking: true, AboutItem: 1})
		}, false},
		{"about a rejected item", func(f *fixture) {
			items, _ := f.h.Propose(1, factgen.QuestionSpec{Blocking: true, AboutItem: 1})
			f.h.Review(func(r *factgen.Review) { r.RejectItem(items[0]) })
		}, false},
		{"about an accepted item now in the scope", func(f *fixture) {
			items, _ := f.h.Propose(2, factgen.QuestionSpec{Blocking: true, AboutItem: 1})
			f.h.Review(func(r *factgen.Review) {
				n := r.AddRequirement(f.goal, "must")
				r.FromItem(items[0])
				r.AddCriterion(n, "manual", nil)
				r.FromItem(items[1])
				r.Scope(f.req, n)
			})
		}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := readyFixture(t)
			c.setup(f)
			res := f.derive(t)
			if got := len(gapsOf(res, derive.GapBlockingQuestion)) == 1; got != c.blocks {
				t.Errorf("BLOCKING_QUESTION = %v, want %v (gaps %v)", got, c.blocks, gapTypes(res.Gaps))
			}
			if c.blocks {
				g := gapsOf(res, derive.GapBlockingQuestion)[0]
				if !g.Blocking || g.TargetKey != "Q-001" || g.NextAction != "回答问题" {
					t.Errorf("gap = %+v", g)
				}
			}
		})
	}
}

// M2-Q8 (data-model §6.10): SOURCE_REMOVED warnings, scope filtered, never
// blocking; SCOPE_STALE blocks until the scope is confirmed again.
func TestSourceRemoved(t *testing.T) {
	f := readyFixture(t)
	var out string
	f.h.Apply(func(r *factgen.Review) {
		r.Update(f.goal, false, func(x *factgen.Fields) { x.Title = "cited" })
		r.Ref(f.goal, f.src, "verified")
		r.Update(f.ac, false, func(x *factgen.Fields) { x.Statement = "cited" })
		r.Ref(f.ac, f.src, "verified")
		out = r.AddRequirement("", "could")
		r.Ref(out, f.src, "verified")
		r.Scope(f.req)
	})
	f.h.RemoveSource(f.src)
	res := f.derive(t)
	if got := gapTypes(res.Delivery.BlockingGaps); !slices.Equal(got, []derive.GapType{derive.GapScopeStale}) {
		t.Fatalf("blocking gaps = %v, want [SCOPE_STALE]", got)
	}
	f.h.ConfirmScope(f.req)
	res = f.derive(t)
	if res.Delivery.Status != derive.TargetReady {
		t.Fatalf("delivery = %s with gaps %v", res.Delivery.Status, res.Gaps)
	}
	var got []string
	for _, g := range gapsOf(res, derive.GapSourceRemoved) {
		got = append(got, g.TargetKey)
		if g.Blocking || g.Reason != "" || g.NextAction != "检查条目的来源引用" {
			t.Errorf("SOURCE_REMOVED = %+v", g)
		}
	}
	if want := []string{"GOAL-1", "REQ-001", "AC-001-1"}; !slices.Equal(got, want) {
		t.Errorf("SOURCE_REMOVED on %v, want %v", got, want)
	}
	// Without a scope, every active entity that cites the source.
	g := factgen.New(projectID)
	g.SetTarget("default", "refs/heads/main")
	src := g.AddSource("docs/a.md", factgen.Hash("a"))
	g.Apply(func(r *factgen.Review) {
		goal := r.AddGoal("G")
		r.Ref(goal, src, "verified")
		req := r.AddRequirement("", "must")
		r.Ref(req, src, "verified")
	})
	g.RemoveSource(src)
	res = mustDerive(t, toderive.Input(legalDump(t, g), g.Current(commitA, "in_development"), g.Seq()))
	if n := len(gapsOf(res, derive.GapSourceRemoved)); n != 2 {
		t.Errorf("SOURCE_REMOVED without a scope: %d, want 2", n)
	}
}

// data-model §6.11: the metrics.
func TestMetrics(t *testing.T) {
	f := newFixture(t)
	f.accept(f.ac, commitA, f.pass(f.ac, commitA))
	claimed := addAuto(f, f.req)
	f.h.Claims("sync", factgen.ClaimSpec{Criterion: claimed, Assertion: "done"})
	var r2, waivedReq string
	f.h.Apply(func(r *factgen.Review) {
		r2 = r.AddRequirement(f.goal, "must")
		waivedReq = r.AddRequirement(f.goal, "must")
	})
	e := addAuto(f, r2)
	f.h.Decide(e, factgen.DecisionSpec{Decision: "accepted", Mode: "exception", Snapshot: commitA})
	f.h.Decide(addAuto(f, waivedReq), factgen.DecisionSpec{Decision: "waived"})
	f.h.ConfirmScope(f.req, r2, waivedReq)
	f.h.Propose(3, factgen.QuestionSpec{Blocking: true}, factgen.QuestionSpec{})
	f.h.Claims("audit", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"})
	res := f.derive(t)
	m := res.Metrics
	// RS: REQ-001 in progress, REQ-002 accepted, REQ-003 waived: 1 / 2.
	if m.ScopeAcceptanceRatio == nil || *m.ScopeAcceptanceRatio != (derive.Ratio{Num: 1, Den: 2}) {
		t.Errorf("scope acceptance ratio = %+v, want 1/2", m.ScopeAcceptanceRatio)
	}
	wantCounts := map[derive.CriterionStatus]int{derive.CriterionAccepted: 2, derive.CriterionInProgress: 1, derive.CriterionWaived: 1}
	if len(m.ACStatusCounts) != len(wantCounts) {
		t.Errorf("status counts = %v, want %v", m.ACStatusCounts, wantCounts)
	}
	for k, v := range wantCounts {
		if m.ACStatusCounts[k] != v {
			t.Errorf("status counts = %v, want %v", m.ACStatusCounts, wantCounts)
		}
	}
	if m.ClaimedWithoutEvidence != 1 || m.PendingReviewItems != 3 || m.OpenQuestions != 2 || m.BlockingQuestions != 1 ||
		m.ExceptionAcceptances != 1 || m.ObservedEvidenceAcceptances != 1 {
		t.Errorf("metrics = %+v", m)
	}
	// One of four active criteria has an audit claim of its revision.
	if res.InventoryCoverage == nil || *res.InventoryCoverage != (derive.Ratio{Num: 1, Den: 4}) {
		t.Errorf("inventory coverage = %+v, want 1/4", res.InventoryCoverage)
	}

	t.Run("a null ratio when every requirement of the scope is waived", func(t *testing.T) {
		f := newFixture(t)
		f.h.Decide(f.ac, factgen.DecisionSpec{Decision: "waived"})
		if r := f.derive(t).Metrics.ScopeAcceptanceRatio; r != nil {
			t.Errorf("ratio = %+v, want null", r)
		}
	})
	t.Run("no coverage without an active criterion", func(t *testing.T) {
		g := factgen.New(projectID)
		g.Apply(func(r *factgen.Review) { r.AddRequirement("", "must") })
		res := mustDerive(t, toderive.Input(legalDump(t, g), g.Current("", "in_development"), g.Seq()))
		if res.InventoryCoverage != nil || len(gapsOf(res, derive.GapInventoryIncomplete)) != 0 {
			t.Errorf("coverage = %+v, gaps %v", res.InventoryCoverage, gapTypes(res.Gaps))
		}
	})
}

// data-model §6.12: the initialization progress at successive cutoffs.
func TestInitProgressFromFacts(t *testing.T) {
	h := factgen.New(projectID)
	h.SetTarget("default", "refs/heads/main")
	steps := []derive.InitProgress{derive.Initialized}
	h.AddSource("docs/a.md", factgen.Hash("a"))
	steps = append(steps, derive.SourcesRegistered)
	items, _ := h.Propose(2)
	steps = append(steps, derive.BaselineProposed)
	var req, ac string
	h.Review(func(r *factgen.Review) {
		req = r.AddRequirement("", "must")
		r.FromItem(items[0])
		ac = r.AddCriterion(req, "manual", nil)
		r.FromItem(items[1])
	})
	steps = append(steps, derive.BaselineConfirmed)
	h.ConfirmScope(req)
	steps = append(steps, derive.ScopeConfirmed)
	h.Claims("audit", factgen.ClaimSpec{Criterion: ac, Assertion: "unknown"})
	steps = append(steps, derive.Inventoried)
	for i, want := range steps {
		asOf := int64(i + 1)
		d := legalDump(t, h)
		cur := h.Current(commitA, "in_development")
		if asOf < 2 {
			cur.Observations = nil // the source does not exist yet
		}
		res := mustDerive(t, toderive.Input(d, cur, asOf))
		if res.InitProgress != want {
			t.Errorf("at seq %d: init progress = %s, want %s", asOf, res.InitProgress, want)
		}
	}
	// sources_skipped stands for a source, and a later step needs the
	// earlier ones.
	g := factgen.New(projectID)
	g.Propose(0)
	cur := g.Current("", "in_development")
	if res := mustDerive(t, toderive.Input(legalDump(t, g), cur, g.Seq())); res.InitProgress != derive.Initialized {
		t.Errorf("a proposal without a source: %s", res.InitProgress)
	}
	cur.SourcesSkipped = true
	if res := mustDerive(t, toderive.Input(legalDump(t, g), cur, g.Seq())); res.InitProgress != derive.BaselineProposed {
		t.Errorf("sources skipped: %s", res.InitProgress)
	}
}
