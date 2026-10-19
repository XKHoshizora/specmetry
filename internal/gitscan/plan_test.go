package gitscan

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/gitx"
)

func sha(c string) string { return strings.Repeat(c, 40) }

// WP2.3 spec §5.2 step 3, row by row: what each ref's strict read is,
// given the cursors at the plan's start, the classification of their old
// tips and the ancestry answers.
func TestPlanReads(t *testing.T) {
	main := Ref{Kind: KindBranch, Name: "refs/heads/main", Tip: sha("c")}
	feature := Ref{Kind: KindBranch, Name: "refs/heads/feature", Tip: sha("f")}
	detached := Ref{Kind: KindDetached, Name: "/r/wt", Tip: sha("d")}
	never := func(string, string) (bool, error) { return false, errors.New("not asked") }
	yes := func(string, string) (bool, error) { return true, nil }
	no := func(string, string) (bool, error) { return false, nil }
	for _, c := range []struct {
		name     string
		cursors  map[CursorKey]string
		refs     []Ref
		presence map[string]gitx.Presence
		ancestor func(a, b string) (bool, error)
		want     []Read
	}{
		{
			name:     "the first scan: every ref's whole history",
			refs:     []Ref{main, detached},
			ancestor: never,
			want:     []Read{{Ref: main}, {Ref: detached}},
		},
		{
			name:     "a cursor at the tip: nothing",
			cursors:  map[CursorKey]string{main.key(): main.Tip},
			refs:     []Ref{main},
			presence: map[string]gitx.Presence{main.Tip: gitx.Present},
			ancestor: never,
		},
		{
			name:     "the old tip present and an ancestor: the increment",
			cursors:  map[CursorKey]string{main.key(): sha("a")},
			refs:     []Ref{main},
			presence: map[string]gitx.Presence{sha("a"): gitx.Present},
			ancestor: yes,
			want:     []Read{{Ref: main, Exclude: []string{sha("a")}}},
		},
		{
			name:     "the old tip present, not an ancestor (rewritten): the whole history",
			cursors:  map[CursorKey]string{main.key(): sha("a")},
			refs:     []Ref{main},
			presence: map[string]gitx.Presence{sha("a"): gitx.Present},
			ancestor: no,
			want:     []Read{{Ref: main}},
		},
		{
			name:     "the old tip reliably absent: a rewrite, the whole history",
			cursors:  map[CursorKey]string{main.key(): sha("a")},
			refs:     []Ref{main},
			presence: map[string]gitx.Presence{sha("a"): gitx.Absent},
			ancestor: never,
			want:     []Read{{Ref: main}},
		},
		{
			name: "a new ref: excluding every present old tip, not the absent ones (D4)",
			cursors: map[CursorKey]string{
				main.key(): sha("a"),
				{Kind: KindBranch, Name: "refs/heads/old"}: sha("b"),
				{Kind: KindDetached, Name: "/gone"}:        sha("e"),
			},
			refs:     []Ref{main, feature},
			presence: map[string]gitx.Presence{sha("a"): gitx.Present, sha("b"): gitx.Present, sha("e"): gitx.Absent},
			ancestor: yes,
			want:     []Read{{Ref: main, Exclude: []string{sha("a")}}, {Ref: feature, Exclude: []string{sha("a"), sha("b")}}},
		},
		{
			name:     "a new ref whose tip a cursor already has: excluded, nothing new",
			cursors:  map[CursorKey]string{main.key(): main.Tip},
			refs:     []Ref{main, {Kind: KindBranch, Name: "refs/heads/copy", Tip: main.Tip}},
			presence: map[string]gitx.Presence{main.Tip: gitx.Present},
			ancestor: never,
			want:     []Read{{Ref: Ref{Kind: KindBranch, Name: "refs/heads/copy", Tip: main.Tip}, Exclude: []string{main.Tip}}},
		},
	} {
		got, err := planReads(c.cursors, c.refs, c.presence, c.ancestor)
		if err != nil || !slices.EqualFunc(got, c.want, func(a, b Read) bool { return a.Ref == b.Ref && slices.Equal(a.Exclude, b.Exclude) }) {
			t.Errorf("%s: planReads = %+v, %v; want %+v", c.name, got, err, c.want)
		}
	}
}

// An ancestry check that fails (K8: the old tip reclaimed after it was
// classified) fails the plan; an old tip with no classification is a
// mistake of the caller.
func TestPlanReadsFailures(t *testing.T) {
	main := Ref{Kind: KindBranch, Name: "refs/heads/main", Tip: sha("c")}
	cursors := map[CursorKey]string{main.key(): sha("a")}
	failure := apperr.New(apperr.Environment, "git merge-base failed")
	_, err := planReads(cursors, []Ref{main}, map[string]gitx.Presence{sha("a"): gitx.Present},
		func(string, string) (bool, error) { return false, failure })
	if !errors.Is(err, failure) {
		t.Errorf("a failed ancestry check: %v", err)
	}
	_, err = planReads(cursors, []Ref{main}, nil, func(string, string) (bool, error) { return true, nil })
	if apperr.ExitCode(err) != apperr.ExitInternal {
		t.Errorf("an unclassified old tip: %v", err)
	}
}
