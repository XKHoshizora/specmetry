package derive_test

import (
	"testing"

	"github.com/XKHoshizora/specmetry/internal/domain/derive"
)

// data-model §6.12: the progress is the last step of the unbroken run of
// satisfied conditions, starting from the first.
func TestInitProgress(t *testing.T) {
	all := derive.InitFacts{
		HasSources: true, HasProposal: true, HasBaseline: true,
		ScopeConfirmed: true, InventoryComplete: true,
	}
	cases := []struct {
		name  string
		facts derive.InitFacts
		want  derive.InitProgress
	}{
		{"registered only", derive.InitFacts{}, derive.Initialized},
		{"a source", derive.InitFacts{HasSources: true}, derive.SourcesRegistered},
		{"sources skipped", derive.InitFacts{SourcesSkipped: true}, derive.SourcesRegistered},
		{"a proposal", derive.InitFacts{HasSources: true, HasProposal: true}, derive.BaselineProposed},
		{"a baseline", derive.InitFacts{HasSources: true, HasProposal: true, HasBaseline: true}, derive.BaselineConfirmed},
		{"scope confirmed", derive.InitFacts{HasSources: true, HasProposal: true, HasBaseline: true, ScopeConfirmed: true}, derive.ScopeConfirmed},
		{"everything", all, derive.Inventoried},
		{"inventory skipped", derive.InitFacts{HasSources: true, HasProposal: true, HasBaseline: true, ScopeConfirmed: true, InventorySkipped: true}, derive.Inventoried},
		{"docs_only needs no inventory", derive.InitFacts{SourcesSkipped: true, HasProposal: true, HasBaseline: true, ScopeConfirmed: true, DocsOnly: true}, derive.Inventoried},
		// Cumulative: a later condition does not count while an earlier one
		// is missing (MVP-INIT-07: --skip-inventory without a baseline does
		// not move past baseline_proposed).
		{"no source, everything else", derive.InitFacts{HasProposal: true, HasBaseline: true, ScopeConfirmed: true, InventoryComplete: true}, derive.Initialized},
		{"skip inventory without a baseline", derive.InitFacts{HasSources: true, HasProposal: true, InventorySkipped: true}, derive.BaselineProposed},
		{"docs_only without a proposal", derive.InitFacts{HasSources: true, DocsOnly: true}, derive.SourcesRegistered},
		{"scope without inventory", derive.InitFacts{HasSources: true, HasProposal: true, HasBaseline: true, ScopeConfirmed: true}, derive.ScopeConfirmed},
	}
	for _, c := range cases {
		if got := derive.InitProgressOf(c.facts); got != c.want {
			t.Errorf("%s: InitProgressOf(%+v) = %s, want %s", c.name, c.facts, got, c.want)
		}
	}
}
