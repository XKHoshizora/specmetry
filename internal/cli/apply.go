package cli

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/proposal"
	"github.com/XKHoshizora/specmetry/internal/review"
	"github.com/XKHoshizora/specmetry/internal/source"
)

// applyOrigin checks the origin against --apply (V-P01, V-P09): a person's
// edit (origin human_edit) is applied at once with --apply and is never a
// pending proposal; it needs an item, and every update item its
// change_class, since no reviewer chooses one later.
func applyOrigin(p *proposal.Proposal, apply bool) error {
	if !apply {
		if p.Origin == "human_edit" {
			return apperr.New(apperr.InvalidInput, "origin human_edit is a person's edit, applied at once with propose --apply").
				WithDetails(apperr.Detail{Pointer: "/origin", Problem: "human_edit needs propose --apply; use another origin to submit the proposal for review"})
		}
		return nil
	}
	var ps []apperr.Detail
	if p.Origin != "human_edit" {
		ps = append(ps, apperr.Detail{Pointer: "/origin", Problem: "propose --apply applies a person's edit: origin must be human_edit"})
	}
	if len(p.Items) == 0 {
		ps = append(ps, apperr.Detail{Pointer: "/items", Problem: "a person's edit changes the baseline: it needs at least one item"})
	}
	for i, it := range p.Items {
		if it.Op == "update" && it.ChangeClass == "" {
			ps = append(ps, apperr.Detail{Pointer: "/items/" + strconv.Itoa(i) + "/change_class",
				Problem: "a person's edit is applied at once, so every update needs change_class, material or editorial (V-P09)"})
		}
	}
	if len(ps) > 0 {
		return apperr.New(apperr.InvalidInput, "the edit is not valid; nothing was written").WithDetails(ps...)
	}
	return nil
}

// humanEdit is a person's edit checked outside the transaction: the
// summary to confirm, and the references material updates without new ones
// carry over, checked again.
type humanEdit struct {
	lines   []string
	carried map[string]review.Carry
}

func prepareEdit(ctx context.Context, w *workspace, p *proposal.Proposal) (*humanEdit, error) {
	var needs []review.Carry
	var sources map[string]source.Source
	if err := w.tx(ctx, func(tx *sql.Tx, proj project.Registered) error {
		for i, it := range p.Items {
			if it.Op != "update" || (len(it.Sources) > 0 && len(it.CodeRefs) > 0) {
				continue
			}
			r, found, err := baseline.LoadKey(ctx, tx, proj.ID, it.Target)
			if err != nil {
				return err
			}
			if !found {
				continue // storing the proposal refuses it
			}
			c, err := review.CurrentRefs(ctx, tx, proj.ID, strconv.Itoa(i), r.Type, r.ID)
			if err != nil {
				return err
			}
			if len(it.Sources) > 0 {
				c.SourceRefs = nil
			}
			if len(it.CodeRefs) > 0 {
				c.CodeRefs = nil
			}
			if len(c.SourceRefs)+len(c.CodeRefs) > 0 {
				needs = append(needs, c)
			}
		}
		var err error
		sources, err = review.CarriedSources(ctx, tx, proj.ID, needs)
		return err
	}); err != nil {
		return nil, err
	}
	carried, err := review.Recheck(ctx, w.where, sources, needs)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, it := range p.Items {
		counts[it.Op]++
	}
	e := &humanEdit{carried: carried, lines: []string{
		fmt.Sprintf("a person's edit: %s", p.Summary),
		fmt.Sprintf("applied at once as baseline version %d (now %d): %d to add, %d to update, %d to retire, %d to reaffirm",
			p.BaseVersion+1, p.BaseVersion, counts["add"], counts["update"], counts["retire"], counts["reaffirm"]),
	}}
	for i, it := range p.Items {
		subject := it.Target
		if it.Op == "add" {
			subject = "ref " + it.Ref
		}
		line := fmt.Sprintf("item %d: %s %s %s", i, it.Op, it.Type, subject)
		if it.ChangeClass != "" {
			line += " (" + it.ChangeClass + ")"
		}
		e.lines = append(e.lines, line)
	}
	return e, nil
}

// apply applies the stored edit in the transaction that stored it, with its
// seq, through the same mechanism as a review: every item accepted with
// the change_class it carries.
func (e *humanEdit) apply(ctx context.Context, tx *sql.Tx, projectID string, p *proposal.Proposal, checked proposal.Checked,
	r *proposal.Receipt, actor, channel string) error {
	plan := baseline.Plan{ProjectID: projectID, ProposalID: r.ProposalID, ExpectedVersion: p.BaseVersion, VersionPointer: "/base_version",
		Seq: r.Seq, Actor: actor, Channel: channel, Summary: p.Summary, Applied: map[string]string{}}
	refItems := map[string]string{}
	for i, it := range p.Items {
		pointer := "/items/" + strconv.Itoa(i)
		ch := baseline.Change{ItemID: r.Items[i].ItemID, Index: i, Pointer: pointer, DataPointer: pointer + "/data", Op: it.Op,
			Type: baseline.Type(it.Type), Ref: it.Ref, Data: it.Data, ChangeClass: it.ChangeClass, Origin: it.Origin,
			Confidence: it.Confidence}
		if it.Op == "add" {
			refItems[it.Ref] = ch.ItemID
		} else {
			target, found, err := baseline.LoadKey(ctx, tx, projectID, it.Target)
			if err != nil {
				return err
			}
			if !found {
				return apperr.New(apperr.Internal, "a stored item's target is not in the baseline")
			}
			ch.TargetID = target.ID
		}
		for _, s := range checked.Items[i] {
			ch.SourceRefs = append(ch.SourceRefs, baseline.SourceRef{SourceID: s.SourceID, Locator: s.Locator,
				ExcerptSHA256: s.ExcerptSHA256, Verification: string(s.Verification), SourceSHA256: s.SourceSHA256})
		}
		for _, c := range checked.CodeRefs[i] {
			ch.CodeRefs = append(ch.CodeRefs, baseline.CodeRef{Path: c.Path, Lines: c.Lines, Exists: c.Exists, CheckedHead: c.CheckedHead})
		}
		if c, ok := e.carried[strconv.Itoa(i)]; ok {
			if c.EntityID != ch.TargetID {
				return apperr.New(apperr.StalePrecondition, "the baseline changed while the edit was being confirmed; read it again")
			}
			ch.CarrySourceRefs, ch.CarryCodeRefs = c.SourceRefs, c.CodeRefs
		}
		plan.Accepted = append(plan.Accepted, ch)
	}
	res, err := baseline.Apply(ctx, tx, plan)
	if err != nil {
		return err
	}
	r.AppliedVersion = &res.Version
	for i := range r.Items {
		r.Items[i].ReviewState = "accepted"
		if key, ok := res.Keys[r.Items[i].ItemID]; ok {
			r.Items[i].AssignedKey = &key
		}
	}
	for i, q := range r.Questions {
		if q.About == nil {
			continue
		}
		if item, ok := refItems[*q.About]; ok {
			if key, ok := res.Keys[item]; ok {
				r.Questions[i].About = &key
			}
		}
	}
	return nil
}

func printApplied(w io.Writer, r proposal.Receipt) error {
	var keys []string
	for _, it := range r.Items {
		if it.AssignedKey != nil {
			keys = append(keys, *it.AssignedKey)
		}
	}
	slices.Sort(keys)
	var b strings.Builder
	fmt.Fprintf(&b, "Applied as baseline version %d (proposal %s, seq %d): %d items.\n", *r.AppliedVersion, r.ProposalID, r.Seq, len(r.Items))
	if len(keys) > 0 {
		fmt.Fprintf(&b, "New keys: %s\n", strings.Join(keys, ", "))
	}
	_, err := io.WriteString(w, b.String())
	return err
}
