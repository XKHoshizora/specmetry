package factgen_test

import (
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/legality"
)

func base(t *testing.T) (h *factgen.History, req, ac string) {
	t.Helper()
	h = factgen.New("01JAAAAAAAPR0JECT000000000")
	h.SetTarget("default", "refs/heads/main")
	h.Apply(func(r *factgen.Review) {
		goal := r.AddGoal("g")
		req = r.AddRequirement(goal, "must")
		ac = r.AddCriterion(req, "manual", nil)
		r.Scope(req)
	})
	return h, req, ac
}

func legal(t *testing.T, h *factgen.History) {
	t.Helper()
	if p := legality.Check(h.Dump(), legality.Options{}); len(p) > 0 {
		t.Fatalf("not legal:\n%s", legality.Report(p, h.Trace()))
	}
}

// data-model §5.5: the changes apply by step and item_index, whatever
// order they are planned in; a cascade follows its cause.
func TestPlannedChangesApplyInTheOrderOfSection55(t *testing.T) {
	h, req, ac := base(t)
	var g2 string
	h.Apply(func(r *factgen.Review) { g2 = r.AddGoal("g2"); r.AddRequirement(g2, "should") })
	seq := h.Apply(func(r *factgen.Review) {
		r.Retire(ac)                       // a criterion retire
		r.Update(req, true, nil)           // a requirement
		r.AddCriterion(req, "manual", nil) // a criterion add
		r.Retire(g2)                       // a goal: other loses it in a cascade
		r.AddGoal("new")                   // a goal
	})
	var got []string
	for _, c := range h.Dump().BaselineChanges {
		if c.Seq == seq {
			cascade := ""
			if c.ProposalItemID == nil {
				cascade = " (cascade)"
			}
			got = append(got, c.EntityType+" "+c.Op+cascade)
		}
	}
	want := []string{
		"goal retire", "requirement update (cascade)", "goal add",
		"requirement update",
		"criterion add", "criterion retire",
	}
	if strings.Join(got, ", ") != strings.Join(want, ", ") {
		t.Errorf("changes = %v\nwant %v", got, want)
	}
	legal(t, h)
}

// Follow-ups (step 8) come after the items, with their rationale, for the
// criteria of a requirement the review revises materially.
func TestFollowUpsComeLast(t *testing.T) {
	h, req, ac := base(t)
	var second string
	h.Apply(func(r *factgen.Review) { second = r.AddCriterion(req, "manual", nil) })
	items, _ := h.Propose(1)
	seq := h.Review(func(r *factgen.Review) {
		r.FollowUp(ac, "reaffirm")
		r.Update(req, true, nil)
		r.FromItem(items[0])
		r.FollowUp(second, "retire")
	})
	var got []string
	for _, c := range h.Dump().BaselineChanges {
		if c.Seq == seq {
			got = append(got, c.EntityType+" "+c.Op)
			if (c.ProposalItemID == nil) != (c.Rationale != nil) {
				t.Errorf("change %s: item %v, rationale %v", c.ID, c.ProposalItemID, c.Rationale)
			}
		}
	}
	if want := "requirement update, criterion reaffirm, criterion retire"; strings.Join(got, ", ") != want {
		t.Errorf("changes = %v", got)
	}
	legal(t, h)
}

func mustPanic(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(r.(string), want) {
			t.Errorf("panic %v, want one about %q", r, want)
		}
	}()
	fn()
}

// A construction that breaks a write rule is refused, not written as a
// legal history.
func TestIllegalConstructionsPanic(t *testing.T) {
	t.Run("V-P03: a target named twice", func(t *testing.T) {
		h, req, _ := base(t)
		mustPanic(t, "V-P03", func() {
			h.Apply(func(r *factgen.Review) { r.Update(req, false, nil); r.Update(req, true, nil) })
		})
	})
	t.Run("V-P03: across two reviews of one proposal", func(t *testing.T) {
		h, req, _ := base(t)
		items, _ := h.Propose(2)
		h.Review(func(r *factgen.Review) { r.Update(req, false, nil); r.FromItem(items[0]) })
		mustPanic(t, "V-P03", func() {
			h.Review(func(r *factgen.Review) { r.Retire(req); r.FromItem(items[1]) })
		})
	})
	t.Run("a review's change without an item", func(t *testing.T) {
		h, req, _ := base(t)
		mustPanic(t, "needs a proposal item", func() {
			h.Review(func(r *factgen.Review) { r.Update(req, false, nil) })
		})
	})
	t.Run("two proposals in one review", func(t *testing.T) {
		h, _, _ := base(t)
		a, _ := h.Propose(1)
		b, _ := h.Propose(1)
		mustPanic(t, "one proposal", func() {
			h.Review(func(r *factgen.Review) { r.RejectItem(a[0]); r.RejectItem(b[0]) })
		})
	})
	t.Run("a change of an entity the write retires with its parent", func(t *testing.T) {
		h, req, ac := base(t)
		mustPanic(t, "retired by this write", func() {
			h.Apply(func(r *factgen.Review) { r.Retire(req); r.Update(ac, false, nil) })
		})
	})
	t.Run("a reference of an entity the write does not change", func(t *testing.T) {
		h, req, _ := base(t)
		mustPanic(t, "does not change", func() {
			h.Apply(func(r *factgen.Review) { r.AddGoal("x"); r.Ref(req, "01JAAAAAAASRC0000000000000", "verified") })
		})
	})
	t.Run("a follow-up of a requirement not revised materially", func(t *testing.T) {
		h, req, ac := base(t)
		items, _ := h.Propose(1)
		mustPanic(t, "follow-up", func() {
			h.Review(func(r *factgen.Review) { r.Update(req, false, nil); r.FromItem(items[0]); r.FollowUp(ac, "reaffirm") })
		})
	})
	t.Run("a follow-up in a person's edit", func(t *testing.T) {
		h, _, ac := base(t)
		mustPanic(t, "no follow-ups", func() {
			h.Apply(func(r *factgen.Review) { r.FollowUp(ac, "reaffirm") })
		})
	})
	t.Run("a question about an item accepted as no add", func(t *testing.T) {
		h, req, _ := base(t)
		items, _ := h.Propose(1, factgen.QuestionSpec{AboutItem: 1})
		mustPanic(t, "no add", func() {
			h.Review(func(r *factgen.Review) { r.Update(req, false, nil); r.FromItem(items[0]) })
		})
	})
}
