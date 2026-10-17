package toderive_test

import (
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/toderive"
)

func str(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// F5: the converter keeps every fact row, including the rows that share a
// seq, and the fields F5 lists for decisions (evidence, snapshot, mode,
// seen_decision_id, seq) and runs (context, result, grade, seq).
func TestConverterKeepsTheRowsAndTheirTiming(t *testing.T) {
	var nullSeen, seen int
	for seed := range uint64(100) {
		c := factgen.Random(seed, factgen.Options{Steps: 40, Deliver: 0.5, Isolated: seed%2 == 1})
		d, f := c.Dump, toderive.Facts(c.Dump)
		counts := [][2]int{
			{len(d.DeliveryTargets), len(f.Targets)}, {len(d.BaselineVersions), len(f.BaselineVersions)},
			{len(d.BaselineChanges), len(f.BaselineChanges)}, {len(d.Goals) + len(d.Requirements) + len(d.Criteria), len(f.Entities)},
			{len(d.ScopeConfirmations), len(f.ScopeConfirmations)}, {len(d.ScopeItems), len(f.ScopeItems)},
			{len(d.Sources), len(f.Sources)}, {len(d.SourceAcks), len(f.SourceAcks)}, {len(d.SourceRefs), len(f.SourceRefs)},
			{len(d.Proposals), len(f.Proposals)}, {len(d.ProposalItems), len(f.ProposalItems)}, {len(d.Questions), len(f.Questions)},
			{len(d.ClaimBatches), len(f.ClaimBatches)}, {len(d.Claims), len(f.Claims)}, {len(d.CheckRuns), len(f.CheckRuns)},
			{len(d.Commits), len(f.Commits)}, {len(d.CommitLinks), len(f.CommitLinks)}, {len(d.Decisions), len(f.Decisions)},
		}
		for i, n := range counts {
			if n[0] != n[1] {
				t.Fatalf("seed %d, table %d: %d rows, %d converted", seed, i, n[0], n[1])
			}
		}
		for i, x := range d.Decisions {
			y := f.Decisions[i]
			if y.ID != x.ID || y.Seq != x.Seq || y.EvidenceID != str(x.EvidenceID) || y.Snapshot != str(x.Snapshot) ||
				y.Mode != str(x.Mode) || y.SeenDecisionID != str(x.SeenDecisionID) {
				t.Fatalf("seed %d: decision %s is not kept: %+v", seed, x.ID, y)
			}
			if x.SeenDecisionID == nil {
				nullSeen++
			} else {
				seen++
			}
		}
		for i, r := range d.CheckRuns {
			y := f.CheckRuns[i]
			if y.ID != r.ID || y.Seq != r.Seq || y.WorktreeID != r.WorktreeID || y.HeadStart != str(r.HeadStart) ||
				y.HeadEnd != str(r.HeadEnd) || y.TreeStateStart != r.TreeStateStart || y.TreeStateEnd != r.TreeStateEnd ||
				y.Result != r.Result || string(y.EvidenceGrade) != r.EvidenceGrade || y.FailureKind != str(r.FailureKind) {
				t.Fatalf("seed %d: run %s is not kept: %+v", seed, r.ID, y)
			}
		}
	}
	// seen_decision_id keeps NULL apart from a predecessor's ID.
	if nullSeen == 0 || seen == 0 {
		t.Errorf("decisions: %d with no predecessor, %d with one", nullSeen, seen)
	}
}
