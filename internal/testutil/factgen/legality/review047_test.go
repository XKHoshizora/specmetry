package legality_test

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/legality"
)

// The regressions of review 047 B1. Each one is checked twice: by
// legality.Check, and by a scan written here from the rule alone, so that
// the checker and this test do not share one wrong assumption.

// literalID is an identifier of a history written row by row in a test.
func literalID(n int) string { return fmt.Sprintf("01JAAAAAAAREVIEW047%07d", n) }

// goalCascadeThenUpdate is review 047 B1-R1's history, written row by row
// without History: goal G, its requirement R and a criterion of R; then a
// person's edit with two items, the retirement of G (item 0) and a
// material update of R's title that does not name a goal (item 1). §5.5
// applies the goal step first, with its cascade (R loses its goal, an
// editorial change at R's revision), then the requirement step: R has two
// changes in one version, one of them a cascade (baseline.Apply records
// each change at its own change_index).
func goalCascadeThenUpdate() factgen.Dump {
	g, r, ac := literalID(1), literalID(2), literalID(3)
	p1, p2, v1, v2, target := literalID(4), literalID(5), literalID(6), literalID(7), literalID(8)
	item := func(n int) string { return literalID(10 + n) }
	change := func(seq int64, version string, index int, typ, entity, op string, class *string, after string, it *string) factgen.ChangeRow {
		return factgen.ChangeRow{ID: literalID(100 + int(seq)*10 + index), Seq: seq, VersionID: version, ChangeIndex: index,
			EntityType: typ, EntityID: entity, Op: op, ChangeClass: class, AfterJSON: after, ProposalItemID: it}
	}
	str := func(s string) *string { return &s }
	seq := func(n int64) *int64 { return &n }
	return factgen.Dump{
		ProjectID: "01JAAAAAAAPR0JECT000000000", LatestSeq: 3,
		DeliveryTargets:  []factgen.TargetRow{{ID: target, Name: "default", Ref: "refs/heads/main", CreatedSeq: 1}},
		Proposals:        []factgen.ProposalRow{{ID: p1, Seq: 2}, {ID: p2, Seq: 3}},
		BaselineVersions: []factgen.VersionRow{{ID: v1, Seq: 2, Number: 1}, {ID: v2, Seq: 3, Number: 2}},
		ProposalItems: []factgen.ProposalItemRow{
			{ID: item(1), ProposalID: p1, ItemIndex: 0, ReviewState: "accepted", ReviewedSeq: seq(2)},
			{ID: item(2), ProposalID: p1, ItemIndex: 1, ReviewState: "accepted", ReviewedSeq: seq(2)},
			{ID: item(3), ProposalID: p1, ItemIndex: 2, ReviewState: "accepted", ReviewedSeq: seq(2)},
			{ID: item(4), ProposalID: p2, ItemIndex: 0, ReviewState: "accepted", ReviewedSeq: seq(3)},
			{ID: item(5), ProposalID: p2, ItemIndex: 1, ReviewState: "accepted", ReviewedSeq: seq(3)},
		},
		BaselineChanges: []factgen.ChangeRow{
			change(2, v1, 0, "goal", g, "add", nil, `{"revision":1,"status":"active","values":{"title":"g"}}`, str(item(1))),
			change(2, v1, 1, "requirement", r, "add", nil,
				`{"revision":1,"status":"active","values":{"goal":"GOAL-1","kind":"functional","priority":"must","title":"r"}}`, str(item(2))),
			change(2, v1, 2, "criterion", ac, "add", nil,
				`{"requirement_revision":1,"revision":1,"status":"active","values":{"check":null,"requirement":"REQ-001","statement":"s","verification":"manual"}}`, str(item(3))),
			change(3, v2, 0, "goal", g, "retire", nil, `{"revision":1,"status":"retired","values":{"title":"g"}}`, str(item(4))),
			change(3, v2, 1, "requirement", r, "update", str("editorial"),
				`{"revision":1,"status":"active","values":{"goal":null,"kind":"functional","priority":"must","title":"r"}}`, nil),
			change(3, v2, 2, "requirement", r, "update", str("material"),
				`{"revision":2,"status":"active","values":{"goal":null,"kind":"functional","priority":"must","title":"r, revised"}}`, str(item(5))),
		},
		Goals:        []factgen.EntityRow{{ID: g, Key: "GOAL-1", Origin: "extracted", Status: "retired", Revision: 1, RevisionSeq: 2}},
		Requirements: []factgen.EntityRow{{ID: r, Key: "REQ-001", Origin: "extracted", Status: "active", Revision: 2, RevisionSeq: 3}},
		Criteria: []factgen.EntityRow{{ID: ac, Key: "AC-001-1", Origin: "extracted", RequirementID: r, Status: "active", Revision: 1,
			RevisionSeq: 2, RequirementRevision: 1}},
	}
}

// Review 047 B1-R1: a goal's cascade and an explicit change of the same
// requirement in one write is legal; V-P03 limits the explicit targets of
// a proposal, not the change rows of an entity.
func TestReview047GoalCascadeThenAnExplicitUpdate(t *testing.T) {
	if p := legality.Check(goalCascadeThenUpdate(), legality.Options{}); len(p) > 0 {
		t.Errorf("the legal history is refused:\n%s", legality.Report(p, nil))
	}
}

// treeConflicts finds, from the rows alone, input records that no single
// tree per commit explains (data-model §4.11.2, §7.4 step 2): a commit's
// tree is immutable, so a normalized path at one head_start has one entry,
// and an input_missing run at a head_start needs a declared path that no
// record shows at that commit.
func treeConflicts(d factgen.Dump) []string {
	lookup := func(p string) string { return path.Clean(strings.ReplaceAll(p, `\`, "/")) }
	// The declared inputs of each criterion configuration, from after_json.
	declared := map[string][]string{} // criterion/config hash -> inputs
	for _, ch := range d.BaselineChanges {
		var s struct {
			Values struct {
				Check *baseline.Check `json:"check"`
			} `json:"values"`
		}
		if json.Unmarshal([]byte(ch.AfterJSON), &s) == nil && s.Values.Check != nil {
			declared[ch.EntityID+"/"+baseline.CheckConfigHash(*s.Values.Check)] = s.Values.Check.Inputs
		}
	}
	type key struct{ commit, path string }
	present := map[key]factgen.InputEntry{}
	var out []string
	for _, r := range d.CheckRuns {
		if r.HeadStart == nil || r.InputsJSON == nil {
			continue
		}
		var entries []factgen.InputEntry
		_ = json.Unmarshal([]byte(*r.InputsJSON), &entries)
		for _, e := range entries {
			k := key{*r.HeadStart, lookup(e.Path)}
			was, seen := present[k]
			if seen && (was.Mode != e.Mode || was.Type != e.Type || was.OID != e.OID) {
				out = append(out, fmt.Sprintf("run %s: %s at %s has two entries", r.ID, k.path, prefix(k.commit)))
			}
			present[k] = e
		}
	}
	for _, r := range d.CheckRuns {
		if r.HeadStart == nil || r.FailureKind == nil || *r.FailureKind != "input_missing" {
			continue
		}
		inputs := declared[r.CriterionID+"/"+r.CheckConfigHash]
		if len(inputs) > 0 && !slices.ContainsFunc(inputs, func(p string) bool {
			_, ok := present[key{*r.HeadStart, lookup(p)}]
			return !ok
		}) {
			out = append(out, fmt.Sprintf("run %s at seq %d: input_missing at %s, where every declared input %v is recorded", r.ID, r.Seq,
				prefix(*r.HeadStart), inputs))
		}
	}
	return out
}

func prefix(s string) string { return s[:min(8, len(s))] }

// goalQuestions finds questions about a proposal item accepted as anything
// but a requirement or criterion add: a question's proposal-local about
// names a requirement or criterion item (proposal.CheckStatic).
func goalQuestions(d factgen.Dump) []string {
	var out []string
	for _, q := range d.Questions {
		if q.ProposalItemID == nil {
			continue
		}
		for _, ch := range d.BaselineChanges {
			if ch.ProposalItemID != nil && *ch.ProposalItemID == *q.ProposalItemID &&
				(ch.Op != "add" || (ch.EntityType != "requirement" && ch.EntityType != "criterion")) {
				out = append(out, fmt.Sprintf("%s is about item %s, accepted as a %s %s at seq %d", q.Key, *q.ProposalItemID, ch.EntityType, ch.Op, ch.Seq))
			}
		}
	}
	return out
}

// independent are the problems the scans of this file find.
func independent(d factgen.Dump) []string {
	return slices.Concat(treeConflicts(d), goalQuestions(d))
}

// Review 047 B1-R2, seed 4 of the comparison (MVP domain): a run recorded
// input_missing for the only declared input "a" at a commit, and another
// run recorded an entry for "a" at the same commit.
func TestReview047Seed4OneTreePerCommit(t *testing.T) {
	o := comparisonOptions(4)
	c := factgen.Random(4, o)
	if p := problemsOf(c, o, ""); len(p) > 0 {
		t.Errorf("seed 4:\n%s", legality.Report(p, nil))
	}
	if p := treeConflicts(c.Dump); len(p) > 0 {
		t.Errorf("seed 4: no one tree per commit explains the records:\n%s", strings.Join(p, "\n"))
	}
}

// Review 047 B1-R3, seed 7 of the comparison: a question about a proposal
// item whose item was accepted as a goal.
func TestReview047Seed7QuestionsAboutRequirementOrCriterionItems(t *testing.T) {
	o := comparisonOptions(7)
	c := factgen.Random(7, o)
	if p := problemsOf(c, o, ""); len(p) > 0 {
		t.Errorf("seed 7:\n%s", legality.Report(p, nil))
	}
	if p := goalQuestions(c.Dump); len(p) > 0 {
		t.Errorf("seed 7:\n%s", strings.Join(p, "\n"))
	}
}
