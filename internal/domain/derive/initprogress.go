// Package derive holds Specmetry's derivation rules (data-model.md §6) as
// pure functions: state is only ever derived from recorded facts, never set.
package derive

// InitProgress is a project's onboarding progress (data-model.md §6.12).
type InitProgress string

// Initialization progress values, in order.
const (
	Initialized       InitProgress = "initialized"
	SourcesRegistered InitProgress = "sources_registered"
	BaselineProposed  InitProgress = "baseline_proposed"
	BaselineConfirmed InitProgress = "baseline_confirmed"
	ScopeConfirmed    InitProgress = "scope_confirmed"
	Inventoried       InitProgress = "inventoried"
)

// InitFacts are the facts the progress of a registered project depends on.
type InitFacts struct {
	HasSources        bool
	SourcesSkipped    bool
	HasProposal       bool
	HasBaseline       bool
	ScopeConfirmed    bool
	InventoryComplete bool
	InventorySkipped  bool
	DocsOnly          bool
}

// InitProgressOf derives the progress of a registered project: the last
// step of the unbroken run of satisfied conditions, so a later condition
// never counts while an earlier one is missing.
func InitProgressOf(f InitFacts) InitProgress {
	steps := []struct {
		done  bool
		value InitProgress
	}{
		{f.HasSources || f.SourcesSkipped, SourcesRegistered},
		{f.HasProposal, BaselineProposed},
		{f.HasBaseline, BaselineConfirmed},
		{f.ScopeConfirmed, ScopeConfirmed},
		{f.InventoryComplete || f.InventorySkipped || f.DocsOnly, Inventoried},
	}
	progress := Initialized
	for _, s := range steps {
		if !s.done {
			break
		}
		progress = s.value
	}
	return progress
}
