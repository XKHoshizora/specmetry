package derive_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/testutil/factgen"
)

func changesSince(t *testing.T, in derive.Input, since int64) []derive.Change {
	t.Helper()
	got, err := derive.ChangesSince(in, since)
	if err != nil {
		t.Fatalf("ChangesSince: %v", err)
	}
	return got
}

// MVP-ST-11: the changes the facts after <seq> bring, with this command's
// S and settings on both sides.
func TestChangesSince(t *testing.T) {
	f := newFixture(t)
	since := f.h.Seq()
	f.pass(f.ac, commitA)
	f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "done"})
	var added string
	f.h.Review(func(r *factgen.Review) { added = r.AddCriterion(f.req, "manual", nil) })
	got := changesSince(t, f.input(), since)
	want := []derive.Change{
		{Criterion: "AC-001-1", Field: "delivery_status", From: "in_progress", To: "ready_for_review"},
		{Criterion: "AC-001-1", Field: "progress", From: "unknown", To: "claimed"},
		{Criterion: f.h.Key(added), Field: "delivery_status", From: "", To: "in_progress"},
		{Criterion: f.h.Key(added), Field: "progress", From: "", To: "unknown"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("changes = %+v\nwant %+v", got, want)
	}
	if got := changesSince(t, f.input(), f.h.Seq()); len(got) != 0 {
		t.Errorf("since the cutoff itself: %+v", got)
	}
}

// MVP-ST-11: both sides use the same settings. With docs_only, a not_done
// claim after <seq> leaves the progress not_started: no change. With
// in_development the start is unknown, so it changes.
func TestChangesSinceSharesTheSettings(t *testing.T) {
	f := newFixture(t)
	since := f.h.Seq()
	f.h.Claims("sync", factgen.ClaimSpec{Criterion: f.ac, Assertion: "not_done"})
	f.stage = "docs_only"
	if got := changesSince(t, f.input(), since); len(got) != 0 {
		t.Errorf("docs_only: %+v", got)
	}
	f.stage = "in_development"
	want := []derive.Change{{Criterion: "AC-001-1", Field: "progress", From: "unknown", To: "not_started"}}
	if got := changesSince(t, f.input(), since); !reflect.DeepEqual(got, want) {
		t.Errorf("in_development: %+v", got)
	}
}

// MVP-ST-11, review 044 B1: a settings change alone allocates no seq and
// brings no change since the current cutoff, although a new query reports
// another progress.
func TestSettingsChangeAloneIsNoFactChange(t *testing.T) {
	f := newFixture(t)
	asOf := f.h.Seq()
	before := criterionOf(t, f.derive(t), "AC-001-1").Progress
	f.stage = "docs_only" // init --stage docs_only
	after := f.derive(t)
	if before != derive.ProgressUnknown || criterionOf(t, after, "AC-001-1").Progress != derive.ProgressNotStarted || after.AsOfSeq != asOf {
		t.Fatalf("progress %s then %s at %d", before, criterionOf(t, after, "AC-001-1").Progress, after.AsOfSeq)
	}
	if got := changesSince(t, f.input(), asOf); len(got) != 0 {
		t.Errorf("changes = %+v", got)
	}
}

// Review 045 §3.2: the start of --diff-since needs no observation of the
// sources only in its source set. The full derivation at that cutoff with
// the current observations is refused instead: a missing observation is
// never ignored or taken as NULL.
func TestChangesSinceAcrossASourceSetChange(t *testing.T) {
	f := newFixture(t)
	since := f.h.Seq()
	b := f.h.AddSource("docs/b.md", factgen.Hash("b"))
	f.h.RemoveSource(f.src)
	f.h.ConfirmScope(f.req)
	f.pass(f.ac, commitA)
	in := f.input()
	if len(in.Current.Observations) != 1 || in.Current.Observations[0].SourceID != b {
		t.Fatalf("observations = %+v, want only B", in.Current.Observations)
	}
	want := []derive.Change{{Criterion: "AC-001-1", Field: "delivery_status", From: "in_progress", To: "ready_for_review"}}
	if got := changesSince(t, in, since); !reflect.DeepEqual(got, want) {
		t.Errorf("changes = %+v", got)
	}
	start := in
	start.AsOfSeq = since
	var ie *derive.InputError
	if _, err := derive.Derive(start); !errors.As(err, &ie) {
		t.Errorf("full derivation at the start with B's observation: %v, want an *InputError", err)
	}
	// The full derivation now has its complete input.
	mustDerive(t, in)
}

// ChangesSince refuses what Derive refuses about S, the settings and the
// cutoffs, and needs no observation or workspace.
func TestChangesSinceInputs(t *testing.T) {
	f := newFixture(t)
	for name, edit := range map[string]func(in *derive.Input){
		"no S":        func(in *derive.Input) { in.Current.Snapshot = nil },
		"no settings": func(in *derive.Input) { in.Current.Settings = nil },
		"short facts": func(in *derive.Input) { in.Facts.LatestSeq = 0 },
	} {
		in := f.input()
		edit(&in)
		var ie *derive.InputError
		if _, err := derive.ChangesSince(in, 1); !errors.As(err, &ie) {
			t.Errorf("%s: %v, want an *InputError", name, err)
		}
	}
	in := f.input()
	for _, since := range []int64{-1, in.AsOfSeq + 1} {
		var ie *derive.InputError
		if _, err := derive.ChangesSince(in, since); !errors.As(err, &ie) {
			t.Errorf("since %d: %v, want an *InputError", since, err)
		}
	}
	in.Current.Observations, in.Current.Workspace = nil, nil
	changesSince(t, in, 1)
}
