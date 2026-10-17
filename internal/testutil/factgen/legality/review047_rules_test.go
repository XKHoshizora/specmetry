package legality_test

import (
	"slices"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/legality"
)

// The rules review 047 B1 is about, each with histories the checker must
// allow and histories that break only that rule.

func hasRule(ps []legality.Problem, rule string) bool {
	return slices.ContainsFunc(ps, func(p legality.Problem) bool { return p.Rule == rule })
}

// expect checks a dump: legal when rule is "", else refused for rule and
// for no rule but rule and also, so a broken history breaks only what it
// is about.
func expect(t *testing.T, name string, d factgen.Dump, rule string, also ...string) {
	t.Helper()
	got := legality.Check(d, legality.Options{})
	switch {
	case rule == "" && len(got) > 0:
		t.Errorf("%s: the legal history is refused:\n%s", name, legality.Report(got, nil))
	case rule != "" && !hasRule(got, rule):
		t.Errorf("%s: %s not found:\n%s", name, rule, legality.Report(got, nil))
	case slices.ContainsFunc(got, func(p legality.Problem) bool { return p.Rule != rule && !slices.Contains(also, p.Rule) }):
		t.Errorf("%s: other rules than %s %v:\n%s", name, rule, also, legality.Report(got, nil))
	}
}

// withSecondWrite is goalCascadeThenUpdate with other changes at seq 3.
func withSecondWrite(changes ...factgen.ChangeRow) factgen.Dump {
	d := goalCascadeThenUpdate()
	d.BaselineChanges = append(slices.DeleteFunc(d.BaselineChanges, func(c factgen.ChangeRow) bool { return c.Seq == 3 }), changes...)
	return d
}

// B1-R1: cascades and explicit changes of one entity in one write.
func TestCascadesAndExplicitChangesOfOneEntity(t *testing.T) {
	str := func(s string) *string { return &s }
	g, r, ac := literalID(1), literalID(2), literalID(3)
	row := func(index int, typ, entity, op string, class *string, after string, item *string) factgen.ChangeRow {
		return factgen.ChangeRow{ID: literalID(200 + index), Seq: 3, VersionID: literalID(7), ChangeIndex: index, EntityType: typ,
			EntityID: entity, Op: op, ChangeClass: class, AfterJSON: after, ProposalItemID: item}
	}
	const (
		goalRetired  = `{"revision":1,"status":"retired","values":{"title":"g"}}`
		reqNoGoal    = `{"revision":1,"status":"active","values":{"goal":null,"kind":"functional","priority":"must","title":"r"}}`
		reqRevised   = `{"revision":2,"status":"active","values":{"goal":null,"kind":"functional","priority":"must","title":"r, revised"}}`
		reqRetired   = `{"revision":1,"status":"retired","values":{"goal":null,"kind":"functional","priority":"must","title":"r"}}`
		reqUnderGoal = `{"revision":2,"status":"active","values":{"goal":"GOAL-1","kind":"functional","priority":"must","title":"r, revised"}}`
		acRetired    = `{"requirement_revision":1,"revision":1,"status":"retired","values":{"check":null,"requirement":"REQ-001","statement":"s","verification":"manual"}}`
		acEdited     = `{"requirement_revision":1,"revision":1,"status":"retired","values":{"check":null,"requirement":"REQ-001","statement":"edited","verification":"manual"}}`
		acReaffirmed = `{"requirement_revision":2,"revision":1,"status":"active","values":{"check":null,"requirement":"REQ-001","statement":"s","verification":"manual"}}`
	)
	item4, item5 := str(literalID(14)), str(literalID(15))
	cases := []struct {
		name, rule string
		also       []string // the rules the break also breaks, by structure
		d          factgen.Dump
	}{
		{"the goal's cascade, then an update (review 047)", "", nil, goalCascadeThenUpdate()},
		{"the goal's cascade, then the requirement's retirement and its cascade", "", nil, withSecondWrite(
			row(0, "goal", g, "retire", nil, goalRetired, item4),
			row(1, "requirement", r, "update", str("editorial"), reqNoGoal, nil),
			row(2, "requirement", r, "retire", nil, reqRetired, item5),
			row(3, "criterion", ac, "retire", nil, acRetired, nil),
		)},
		{"a second item of the proposal names the requirement", "V-P03", []string{"changes of an entity"}, func() factgen.Dump {
			d := goalCascadeThenUpdate()
			seq := int64(3)
			d.ProposalItems = append(d.ProposalItems, factgen.ProposalItemRow{ID: literalID(16), ProposalID: literalID(5), ItemIndex: 2,
				ReviewState: "accepted", ReviewedSeq: &seq})
			d.BaselineChanges = append(d.BaselineChanges, row(3, "requirement", r, "update", str("editorial"), reqRevised, str(literalID(16))))
			return d
		}()},
		{"no cascade: the requirement keeps the retired goal", "cascade", nil, withSecondWrite(
			row(0, "goal", g, "retire", nil, goalRetired, item4),
			row(1, "requirement", r, "update", str("material"), reqRevised, item5),
		)},
		{"the cascade after the requirement's own change", "cascade", []string{"explicit change"}, withSecondWrite(
			row(0, "goal", g, "retire", nil, goalRetired, item4),
			row(1, "requirement", r, "update", str("material"), reqRevised, item5),
			row(2, "requirement", r, "update", str("editorial"), reqRevised, nil),
		)},
		{"the update names the goal the write retires", "link", nil, withSecondWrite(
			row(0, "goal", g, "retire", nil, goalRetired, item4),
			row(1, "requirement", r, "update", str("editorial"), reqNoGoal, nil),
			row(2, "requirement", r, "update", str("material"), reqUnderGoal, item5),
		)},
		{"an item changes a criterion its requirement's cascade retired", "target", nil, withSecondWrite(
			row(0, "requirement", r, "retire", nil, `{"revision":1,"status":"retired","values":{"goal":"GOAL-1","kind":"functional","priority":"must","title":"r"}}`, item4),
			row(1, "criterion", ac, "retire", nil, acRetired, nil),
			row(2, "criterion", ac, "reaffirm", nil, acRetired, item5),
		)},
		{"a cascade that changes the values", "retire", nil, withSecondWrite(
			row(0, "goal", g, "retire", nil, goalRetired, item4),
			row(1, "requirement", r, "update", str("editorial"), reqNoGoal, nil),
			row(2, "requirement", r, "retire", nil, reqRetired, item5),
			row(3, "criterion", ac, "retire", nil, acEdited, nil),
		)},
		{"two follow-ups of one criterion", "changes of an entity", nil, func() factgen.Dump {
			// A review revises the requirement materially and reaffirms the
			// criterion as a follow-up; the follow-up row is written twice.
			h := factgen.New("01JAAAAAAAPR0JECT000000000")
			var req, crit string
			h.Apply(func(r *factgen.Review) { req = r.AddRequirement("", "must"); crit = r.AddCriterion(req, "manual", nil) })
			items, _ := h.Propose(1)
			h.Review(func(r *factgen.Review) { r.Update(req, true, nil); r.FromItem(items[0]); r.FollowUp(crit, "reaffirm") })
			d := h.Dump()
			again := d.BaselineChanges[len(d.BaselineChanges)-1]
			again.ID, again.ChangeIndex = literalID(300), again.ChangeIndex+1
			d.BaselineChanges = append(d.BaselineChanges, again)
			return d
		}()},
	}
	for _, c := range cases {
		expect(t, c.name, c.d, c.rule, c.also...)
	}
}

// runsHistory has three automated criteria: AC-001-1 declares "a",
// AC-001-2 "a" and "b", AC-001-3 "./a", the same lookup path as "a"
// (§4.11.2).
func runsHistory() (h *factgen.History, ac1, ac2, ac3 string) {
	h = factgen.New("01JAAAAAAAPR0JECT000000000")
	h.SetTarget("default", "refs/heads/main")
	check := func(inputs ...string) *baseline.Check {
		return &baseline.Check{Command: "go test ./...", Workdir: ".", TimeoutSec: 300, Inputs: inputs}
	}
	h.Apply(func(r *factgen.Review) {
		req := r.AddRequirement("", "must")
		ac1 = r.AddCriterion(req, "automated", check("a"))
		ac2 = r.AddCriterion(req, "automated", check("a", "b"))
		ac3 = r.AddCriterion(req, "automated", check("./a"))
	})
	return h, ac1, ac2, ac3
}

// B1-R2: one immutable tree per commit, read at head_start.
func TestOneTreePerCommit(t *testing.T) {
	x, y := factgen.Commit("x"), factgen.Commit("y")
	entry := func(p, oid string) factgen.InputEntry {
		return factgen.InputEntry{Path: p, Mode: "100644", Type: "blob", OID: factgen.Commit(oid)}
	}
	at := func(s factgen.RunSpec, head string, in ...factgen.InputEntry) factgen.RunSpec {
		s.Head, s.Inputs = head, in
		return s
	}
	missing := factgen.RunSpec{Result: "error", FailureKind: "input_missing"}
	pass := factgen.RunSpec{}
	cases := []struct {
		name, rule string
		runs       func(h *factgen.History, ac1, ac2, ac3 string)
	}{
		{"present at one commit, missing at another", "", func(h *factgen.History, ac1, _, _ string) {
			h.Run(ac1, at(pass, x, entry("a", "a@x")))
			h.Run(ac1, at(missing, y))
		}},
		{"another declared input is the missing one", "", func(h *factgen.History, ac1, ac2, _ string) {
			h.Run(ac1, at(pass, x, entry("a", "a@x")))
			h.Run(ac2, at(missing, x))
		}},
		{"a dirty worktree and a context change read the tree of head_start", "", func(h *factgen.History, ac1, _, ac3 string) {
			h.Run(ac1, at(factgen.RunSpec{TreeStart: "dirty", TreeEnd: "dirty"}, x, entry("a", "a@x")))
			h.Run(ac3, at(factgen.RunSpec{HeadEnd: y, Result: "error", FailureKind: "context_changed"}, x, entry("./a", "a@x")))
			h.Run(ac1, at(missing, y))
		}},
		{"missing where a record shows the only declared input (review 047, seed 4)", "commit tree", func(h *factgen.History, ac1, _, _ string) {
			h.Run(ac1, at(missing, x))
			h.Run(ac1, at(pass, x, entry("a", "a@x")))
		}},
		{"missing where records show every declared input", "commit tree", func(h *factgen.History, ac1, ac2, _ string) {
			h.Run(ac1, at(pass, x, entry("a", "a@x")))
			h.Run(ac2, at(factgen.RunSpec{Result: "fail", FailureKind: "nonzero_exit"}, y, entry("a", "a@y"), entry("b", "b@y")))
			h.Run(ac2, at(pass, x, entry("a", "a@x"), entry("b", "b@x")))
			h.Run(ac2, at(missing, x))
		}},
		{"two entries of one path at one commit", "commit tree", func(h *factgen.History, ac1, _, _ string) {
			h.Run(ac1, at(pass, x, entry("a", "a@x")))
			h.Run(ac1, at(pass, x, entry("a", "another a@x")))
		}},
		{"two entries of one lookup path at one commit", "commit tree", func(h *factgen.History, ac1, _, ac3 string) {
			h.Run(ac1, at(pass, x, entry("a", "a@x")))
			h.Run(ac3, at(pass, x, entry("./a", "another a@x")))
		}},
	}
	for _, c := range cases {
		h, ac1, ac2, ac3 := runsHistory()
		c.runs(h, ac1, ac2, ac3)
		expect(t, c.name, h.Dump(), c.rule)
	}
}

// B1-R3: a question's local about names a requirement or criterion add of
// its proposal; a question about an entity names an active requirement or
// criterion.
func TestQuestionsAboutItemsAndEntities(t *testing.T) {
	// items[0] becomes a requirement, items[1] its criterion, items[2] a
	// goal; items[3] is rejected and items[4] stays pending. The questions
	// are about items 1, 2, 4 and 5, and an existing requirement.
	build := func() (factgen.Dump, []string) {
		h := factgen.New("01JAAAAAAAPR0JECT000000000")
		h.SetTarget("default", "refs/heads/main")
		var existing string
		h.Apply(func(r *factgen.Review) { existing = r.AddRequirement("", "must") })
		items, _ := h.Propose(5,
			factgen.QuestionSpec{Blocking: true, AboutItem: 1}, factgen.QuestionSpec{Blocking: true, AboutItem: 2},
			factgen.QuestionSpec{Blocking: true, AboutItem: 4}, factgen.QuestionSpec{Blocking: true, AboutItem: 5},
			factgen.QuestionSpec{AboutEntity: existing})
		h.Review(func(r *factgen.Review) {
			req := r.AddRequirement("", "must")
			r.FromItem(items[0])
			r.AddCriterion(req, "manual", nil)
			r.FromItem(items[1])
			r.AddGoal("g")
			r.FromItem(items[2])
			r.RejectItem(items[3])
		})
		return h.Dump(), items
	}
	d, _ := build()
	expect(t, "about requirement and criterion items, a rejected and a pending item, an existing requirement", d, "")
	cases := []struct {
		name   string
		mutate func(d *factgen.Dump, items []string)
	}{
		{"about the item accepted as a goal (review 047, seed 7)", func(d *factgen.Dump, items []string) {
			d.Questions[0].ProposalItemID, d.Questions[0].RelatedType, d.Questions[0].RelatedID = &items[2], nil, nil
		}},
		{"about an existing goal", func(d *factgen.Dump, _ []string) {
			goal := "goal"
			d.Questions[4].RelatedType, d.Questions[4].RelatedID = &goal, &d.Goals[0].ID
		}},
		{"about an item of another proposal", func(d *factgen.Dump, _ []string) {
			d.Questions[2].ProposalItemID = &d.ProposalItems[0].ID // the edit's item
		}},
	}
	for _, c := range cases {
		d, items := build()
		c.mutate(&d, items)
		expect(t, c.name, d, "question")
	}
}
