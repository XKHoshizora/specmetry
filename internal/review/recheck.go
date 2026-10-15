package review

import (
	"context"
	"database/sql"

	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/proposal"
	"github.com/XKHoshizora/specmetry/internal/source"
)

// CarriedSources reads the sources the carried references cite, removed
// ones included, in the caller's transaction.
func CarriedSources(ctx context.Context, tx *sql.Tx, projectID string, carry []Carry) (map[string]source.Source, error) {
	out := map[string]source.Source{}
	for _, c := range carry {
		for _, r := range c.SourceRefs {
			if _, ok := out[r.SourceID]; ok {
				continue
			}
			var s source.Source
			var location string
			if err := tx.QueryRowContext(ctx, `SELECT id, location, path FROM sources WHERE project_id = ? AND id = ?`,
				projectID, r.SourceID).Scan(&s.ID, &location, &s.Path); err != nil {
				return nil, err
			}
			s.Location = source.Location(location)
			out[s.ID] = s
		}
	}
	return out, nil
}

// Recheck checks carried references again, outside any transaction
// (data-model §4.5). Under hash_only no excerpt is kept to look for, so a
// source reference keeps its result while its source still has the hash
// it was checked against; once the source changed it is not_checkable, and
// source_missing when the source cannot be read. A code reference is
// checked against the HEAD commit of the worktree w, as a proposal's are.
func Recheck(ctx context.Context, w project.Where, sources map[string]source.Source, carry []Carry) (map[string]Carry, error) {
	hashes := map[string]string{}
	for id, s := range sources {
		sum, _, err := source.HashFile(ctx, s.File(w.Root))
		switch {
		case err == nil:
			hashes[id] = sum
		case ctx.Err() != nil:
			return nil, err
		}
	}
	out := map[string]Carry{}
	for _, c := range carry {
		got := Carry{ItemID: c.ItemID, EntityID: c.EntityID, Revision: c.Revision}
		for _, r := range c.SourceRefs {
			sum, readable := hashes[r.SourceID]
			switch {
			case !readable:
				r.Verification, r.SourceSHA256 = string(proposal.SourceMissing), nil
			case r.SourceSHA256 == nil || *r.SourceSHA256 != sum:
				r.Verification, r.SourceSHA256 = string(proposal.NotCheckable), &sum
			}
			got.SourceRefs = append(got.SourceRefs, r)
		}
		if len(c.CodeRefs) > 0 {
			refs := make([]proposal.CodeRef, len(c.CodeRefs))
			for k, cr := range c.CodeRefs {
				refs[k] = proposal.CodeRef{Path: cr.Path}
				if cr.Lines != nil {
					refs[k].Lines = *cr.Lines
				}
			}
			checked, err := proposal.CheckCodeRefs(ctx, w, [][]proposal.CodeRef{refs})
			if err != nil {
				return nil, err
			}
			for _, cr := range checked[0] {
				got.CodeRefs = append(got.CodeRefs, baseline.CodeRef{Path: cr.Path, Lines: cr.Lines, Exists: cr.Exists, CheckedHead: cr.CheckedHead})
			}
		}
		out[c.ItemID] = got
	}
	return out, nil
}
