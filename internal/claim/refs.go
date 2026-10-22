package claim

import (
	"context"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/gitx"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/proposal"
)

// Tree states as a batch records them.
const (
	Clean   = "clean"
	Dirty   = "dirty"
	Unknown = "unknown"
)

// Context is the code context the CLI captures once, outside the write
// transaction (WP2.4 spec §6.1): the HEAD commit ("" outside git or before
// the first commit) and the worktree state.
type Context struct {
	Head      string
	TreeState string
}

// Capture reads the code context of w's current worktree: HEAD with
// gitx.CommitOf, the state with gitx.Clean. A state Clean cannot decide
// (P1) is unknown, never clean; outside git the head is "" and the state
// unknown, and git is not read. An error of either read is returned as it
// is (D1: an environment error is exit 6, and nothing is written).
func Capture(ctx context.Context, s *gitx.Scratch, w project.Where) (Context, error) {
	if !w.Git {
		return Context{TreeState: Unknown}, nil
	}
	head, err := gitx.CommitOf(ctx, w.Current, "HEAD")
	if err != nil {
		return Context{}, err
	}
	state, err := gitx.Clean(ctx, s, w.Current)
	if err != nil {
		return Context{}, err
	}
	c := Context{Head: head, TreeState: Unknown}
	switch state {
	case gitx.StateClean:
		c.TreeState = Clean
	case gitx.StateDirty:
		c.TreeState = Dirty
	}
	return c, nil
}

// Checked holds the reference results of each claim and unmapped work
// item, in input order (refs_check_json).
type Checked struct {
	Claims   []RefResults
	Unmapped []RefResults
}

// RefResults are one claim's reference results, as refs_check_json holds
// them: the files with the commit they were checked against, and the
// commit prefixes.
type RefResults struct {
	Files   []proposal.CheckedCodeRef `json:"files"`
	Commits []CommitResult            `json:"commits"`
}

// CheckRefs checks the references of b (V-C06): marks, never refusals. A
// file reference is checked against head, the commit Capture read, by the
// rules of a proposal's code references (proposal.CheckCodeRefsAt); a
// commit prefix resolves only if it names exactly one readable commit
// (gitx.ResolveCommits, E4), and never outside git, where no git object is
// read. PD-26: a git that could fetch an object, or a partial clone, reads
// none, and a batch with such references is exit 6. Test names are only
// stored.
func CheckRefs(ctx context.Context, s *gitx.Scratch, w project.Where, head string, b *Batch) (Checked, error) {
	all := make([]Refs, 0, len(b.Claims)+len(b.UnmappedWork))
	for _, c := range b.Claims {
		all = append(all, c.Refs)
	}
	for _, u := range b.UnmappedWork {
		all = append(all, u.Refs)
	}
	files := make([][]proposal.CodeRef, len(all))
	var prefixes []string
	for i, r := range all {
		files[i] = r.Files
		prefixes = append(prefixes, r.Commits...)
	}
	checkedFiles, err := proposal.CheckCodeRefsAt(ctx, w, head, files)
	if err != nil {
		return Checked{}, err
	}
	resolved := map[string]bool{}
	if w.Git && len(prefixes) > 0 {
		partial, err := gitx.PartialClone(ctx, w.Current)
		if err != nil {
			return Checked{}, err
		}
		if partial {
			return Checked{}, apperr.New(apperr.Environment,
				"this repository is a partial clone: git would fetch the objects it lacks from the promisor remote, which Specmetry never does, so commit references cannot be checked here; nothing was written; submit the batch in a full clone, or without commit references")
		}
		if resolved, err = gitx.ResolveCommits(ctx, s, w.Current, prefixes); err != nil {
			return Checked{}, err
		}
	}
	results := make([]RefResults, len(all))
	for i, r := range all {
		results[i] = RefResults{Files: checkedFiles[i], Commits: make([]CommitResult, len(r.Commits))}
		for k, p := range r.Commits {
			results[i].Commits[k] = CommitResult{Commit: p, Resolved: resolved[p]}
		}
	}
	return Checked{Claims: results[:len(b.Claims)], Unmapped: results[len(b.Claims):]}, nil
}
