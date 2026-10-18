package derive_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen/toderive"
)

// MVP-ST-14 (pure part), data-model §6.13.1: the same complete input gives
// the same result, and the input is not modified.
func TestSameInputSameResult(t *testing.T) {
	f := readyFixture(t)
	f.h.Propose(2, factgen.QuestionSpec{Blocking: true})
	in := f.input()
	in.Current.Workspace = &derive.Workspace{WorktreeID: factgen.MainWorktree, Head: commitA, TreeState: "clean"}
	before := deepCopy(t, in)
	first := mustDerive(t, in)
	second := mustDerive(t, in)
	if !reflect.DeepEqual(first, second) {
		t.Error("two derivations of the same input differ")
	}
	if !reflect.DeepEqual(in, before) {
		t.Error("Derive modified its input")
	}
	if _, err := derive.ChangesSince(in, 1); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, before) {
		t.Error("ChangesSince modified its input")
	}
}

// deepCopy copies an input through the converter's own types: a fresh
// input with equal values.
func deepCopy(t *testing.T, in derive.Input) derive.Input {
	t.Helper()
	out := in
	f := in.Facts
	out.Facts = derive.Facts{
		ProjectID: f.ProjectID, LatestSeq: f.LatestSeq,
		Targets: clone(f.Targets), BaselineVersions: clone(f.BaselineVersions), Entities: clone(f.Entities),
		ScopeConfirmations: clone(f.ScopeConfirmations), ScopeItems: clone(f.ScopeItems), Sources: clone(f.Sources),
		SourceAcks: clone(f.SourceAcks), SourceRefs: clone(f.SourceRefs), Proposals: clone(f.Proposals),
		ProposalItems: clone(f.ProposalItems), Questions: clone(f.Questions), ClaimBatches: clone(f.ClaimBatches),
		Claims: clone(f.Claims), Commits: clone(f.Commits), CommitLinks: clone(f.CommitLinks),
	}
	for _, c := range f.BaselineChanges {
		if c.After.Check != nil {
			ch := *c.After.Check
			ch.Inputs = clone(ch.Inputs)
			c.After.Check = &ch
		}
		out.Facts.BaselineChanges = append(out.Facts.BaselineChanges, c)
	}
	for _, r := range f.CheckRuns {
		r.Inputs = clone(r.Inputs)
		out.Facts.CheckRuns = append(out.Facts.CheckRuns, r)
	}
	for _, d := range f.Decisions {
		d.Inputs = clone(d.Inputs)
		out.Facts.Decisions = append(out.Facts.Decisions, d)
	}
	s, st := *in.Current.Snapshot, *in.Current.Settings
	out.Current.Snapshot, out.Current.Settings = &s, &st
	out.Current.Observations = clone(in.Current.Observations)
	if in.Current.Workspace != nil {
		w := *in.Current.Workspace
		out.Current.Workspace = &w
	}
	return out
}

func clone[T any](s []T) []T {
	if s == nil {
		return nil
	}
	return append([]T{}, s...)
}

// MVP-ST-14, MVP-ST-15 (pure part), review 044 B1: a result records the
// current-value inputs it used; after the settings change, a replay with
// the recorded values gives the recorded result, while a new query with the
// new settings gives another one at the same as_of_seq.
func TestReplayWithTheRecordedSettings(t *testing.T) {
	h := factgen.New(projectID)
	h.SetTarget("default", "refs/heads/main") // a project only initialized
	d := legalDump(t, h)
	cur := h.Current(commitA, "in_development")
	original := mustDerive(t, toderive.Input(d, cur, h.Seq()))
	if original.InitProgress != derive.Initialized {
		t.Fatalf("init progress = %s", original.InitProgress)
	}
	if original.Settings != (derive.Settings{Stage: derive.InDevelopment}) || original.Snapshot.Commit != commitA {
		t.Errorf("recorded inputs: settings %+v, snapshot %+v", original.Settings, original.Snapshot)
	}

	// init --skip-sources: the settings change, no seq is allocated.
	changed := cur
	changed.SourcesSkipped = true
	now := mustDerive(t, toderive.Input(d, changed, h.Seq()))
	if now.InitProgress != derive.SourcesRegistered || now.AsOfSeq != original.AsOfSeq || !now.Settings.SourcesSkipped {
		t.Errorf("new query: %s at %d, settings %+v", now.InitProgress, now.AsOfSeq, now.Settings)
	}

	// The replay takes every current value from the original result.
	replay := derive.Input{
		Facts: toderive.Facts(d),
		Current: derive.Current{
			Snapshot: &original.Snapshot, Settings: &original.Settings, Observations: original.Observations,
		},
		AsOfSeq: original.AsOfSeq,
	}
	if again := mustDerive(t, replay); !reflect.DeepEqual(again, original) {
		t.Errorf("the replay differs from the original result: %s", again.InitProgress)
	}
}

// MVP-ST-15 (pure part): the result reports exactly the current values the
// derivation used.
func TestResultRecordsItsCurrentValues(t *testing.T) {
	f := readyFixture(t)
	other := f.h.AddSource("docs/z.md", factgen.Hash("z"))
	f.h.ConfirmScope(f.req)
	f.stage = "incomplete_docs"
	cur := f.current()
	cur.InventorySkipped = true
	cur.Observations = []factgen.Observation{{SourceID: other, SHA256: ""}, {SourceID: f.src, SHA256: factgen.Hash("brief v1")}}
	res := mustDerive(t, toderive.Input(legalDump(t, f.h), cur, f.h.Seq()))
	if res.Settings != (derive.Settings{Stage: derive.IncompleteDocs, InventorySkipped: true}) {
		t.Errorf("settings = %+v", res.Settings)
	}
	want := []derive.Observation{{SourceID: f.src, SHA256: factgen.Hash("brief v1")}, {SourceID: other, SHA256: ""}}
	if f.src > other {
		want[0], want[1] = want[1], want[0]
	}
	if !reflect.DeepEqual(res.Observations, want) {
		t.Errorf("observations = %+v, want %+v", res.Observations, want)
	}
	if res.Snapshot.Commit != commitA || res.Target == nil || res.Target.Snapshot != commitA {
		t.Errorf("snapshot = %+v, target %+v", res.Snapshot, res.Target)
	}
}

// MVP-ST-14, data-model §6.13.1: a missing input is refused, never filled
// in with a current value or a default.
func TestMissingInputs(t *testing.T) {
	cases := []struct {
		name    string
		edit    func(in *derive.Input)
		problem string
	}{
		{"no S", func(in *derive.Input) { in.Current.Snapshot = nil }, "snapshot"},
		{"no settings", func(in *derive.Input) { in.Current.Settings = nil }, "settings"},
		{"no stage", func(in *derive.Input) { in.Current.Settings = &derive.Settings{} }, "stage"},
		{"an unknown stage", func(in *derive.Input) { in.Current.Settings = &derive.Settings{Stage: "alpha"} }, "stage"},
		{"a source without an observation", func(in *derive.Input) { in.Current.Observations = nil }, "observation"},
		{"an observation of a source not in the set", func(in *derive.Input) {
			in.Current.Observations = append(in.Current.Observations, derive.Observation{SourceID: "01JAAAAAAAZZZZZZZZZZZZZZZZ", SHA256: ""})
		}, "observation"},
		{"two observations of a source", func(in *derive.Input) {
			in.Current.Observations = append(in.Current.Observations, in.Current.Observations[0])
		}, "observation"},
		{"facts that do not reach the cutoff", func(in *derive.Input) { in.Facts.LatestSeq = in.AsOfSeq - 1 }, "facts"},
		{"no project", func(in *derive.Input) { in.Facts.ProjectID = "" }, "project"},
		{"a negative cutoff", func(in *derive.Input) { in.AsOfSeq = -1 }, "cutoff"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := newFixture(t).input()
			c.edit(&in)
			res, err := derive.Derive(in)
			var ie *derive.InputError
			if !errors.As(err, &ie) || res != nil {
				t.Fatalf("Derive = %v, %v; want an *InputError", res, err)
			}
			if !strings.Contains(err.Error(), c.problem) {
				t.Errorf("error %q does not name the %s", err, c.problem)
			}
		})
	}
}

// data-model §6.13.1: a NULL observation is a file observed missing, a
// valid input that makes the source changed.
func TestNullObservationIsAnInput(t *testing.T) {
	f := readyFixture(t)
	in := f.input()
	in.Current.Observations[0].SHA256 = ""
	res := mustDerive(t, in)
	g := gapsOf(res, derive.GapSourceChanged)
	if len(g) != 1 || g[0].Reason != "missing" || res.Delivery.Status != derive.TargetNeedsReverification {
		t.Errorf("SOURCE_CHANGED = %+v, delivery %s", g, res.Delivery.Status)
	}
}
