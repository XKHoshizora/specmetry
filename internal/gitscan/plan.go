package gitscan

import (
	"slices"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/gitx"
)

// Cursor kinds, as git_scan_cursors stores them.
const (
	KindBranch   = "branch"
	KindDetached = "detached"
)

// CursorKey is a cursor's identity in a project: a branch's full ref name,
// or a detached HEAD's canonical worktree path (D2).
type CursorKey struct{ Kind, Name string }

// Ref is a ref the scan reads, with the tip it pointed to when the scan
// began; later reads use the tip only, never the name.
type Ref struct{ Kind, Name, Tip string }

func (r Ref) key() CursorKey { return CursorKey{r.Kind, r.Name} }

// Read is one strict log of the scan: the commits reachable from the ref's
// tip and from none of Exclude (old tips shown present).
type Read struct {
	Ref     Ref
	Exclude []string
}

// planReads decides each ref's read (WP2.3 spec §5.2 step 3) from the
// cursors at the plan's start (last tips by key), the classification of
// every one of their old tips, and ancestry:
//
//   - a cursor at the tip: nothing to read;
//   - its old tip present and an ancestor: the increment old..tip;
//   - present and not an ancestor (a rewrite), or reliably absent (a
//     rewrite whose old tip was reclaimed): the ref's whole history;
//   - no cursor (a new ref, D4): the tip, excluding every old tip of the
//     cursors that is present; an absent one excludes nothing, so the read
//     only grows; with no cursor at all (the first scan), whole histories.
//
// Old tips that are present but unreadable or cannot be confirmed never
// get here: Classify fails the scan first.
func planReads(cursors map[CursorKey]string, refs []Ref, presence map[string]gitx.Presence, isAncestor func(ancestor, commit string) (bool, error)) ([]Read, error) {
	var present []string
	for _, tip := range cursors {
		p, ok := presence[tip]
		if !ok {
			return nil, apperr.New(apperr.Internal, "an old tip of the scan was not classified")
		}
		if p == gitx.Present && !slices.Contains(present, tip) {
			present = append(present, tip)
		}
	}
	slices.Sort(present)
	var reads []Read
	for _, r := range refs {
		old, ok := cursors[r.key()]
		switch {
		case !ok:
			reads = append(reads, Read{Ref: r, Exclude: present})
		case old == r.Tip:
		case presence[old] == gitx.Absent:
			reads = append(reads, Read{Ref: r})
		default:
			yes, err := isAncestor(old, r.Tip)
			if err != nil {
				return nil, err
			}
			if yes {
				reads = append(reads, Read{Ref: r, Exclude: []string{old}})
			} else {
				reads = append(reads, Read{Ref: r})
			}
		}
	}
	return reads, nil
}
