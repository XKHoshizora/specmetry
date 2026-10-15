package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/proposal"
)

func newProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proposal",
		Short: "List and show proposals and their review state",
	}
	cmd.AddCommand(newProposalListCmd(), newProposalShowCmd())
	return cmd
}

func newProposalListCmd() *cobra.Command {
	var pending, asJSON bool
	cmd := &cobra.Command{
		Use:   "list [--pending]",
		Short: "List the project's proposals, newest first",
		Args:  cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			w, err := prepare(ctx)
			if err != nil {
				return err
			}
			defer w.close()
			if err := w.open(ctx); err != nil {
				return err
			}
			var doc proposal.ListDoc
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				doc, err = proposal.ListDocument(ctx, tx, p.ID, pending)
				return err
			}); err != nil {
				return err
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), "proposal-list.v1", doc)
			}
			return printProposalList(cmd.OutOrStdout(), doc)
		}),
	}
	cmd.Flags().BoolVar(&pending, "pending", false, "only proposals with items still waiting for review")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the proposal-list.v1 document")
	return cmd
}

func printProposalList(w io.Writer, doc proposal.ListDoc) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Baseline version %d. %d proposals.\n", doc.BaselineVersion, len(doc.Proposals))
	for _, p := range doc.Proposals {
		c := p.Counts
		fmt.Fprintf(&b, "%s  seq %d  %s by %s  base v%d  pending %d, conflict %d, accepted %d, edited %d, rejected %d\n  %s\n",
			p.ID, p.Seq, p.Origin, p.Actor, p.BaseVersion, c["pending"], c["conflict"], c["accepted"], c["accepted_with_edits"],
			c["rejected"], p.Summary)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func newProposalShowCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show <proposal-id>",
		Short: "Show a proposal's items, their review state and conflicts",
		Long: `Show every item of a proposal with its review state. An item whose target
changed after the proposal's base version is in conflict: the comparison
shows the target's values then (base), now (current) and as proposed, and a
review must acknowledge it. baseline_version is the value a review sends
back as seen_base_version.`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w, err := prepare(ctx)
			if err != nil {
				return err
			}
			defer w.close()
			if err := w.open(ctx); err != nil {
				return err
			}
			var doc proposal.ShowDoc
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				doc, err = proposal.ShowDocument(ctx, tx, p.ID, args[0])
				return err
			}); err != nil {
				return err
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), "proposal-show.v1", doc)
			}
			return printProposalShow(cmd.OutOrStdout(), doc)
		}),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the proposal-show.v1 document")
	return cmd
}

func printProposalShow(w io.Writer, doc proposal.ShowDoc) error {
	var b strings.Builder
	p := doc.Proposal
	fmt.Fprintf(&b, "Proposal %s  seq %d  %s by %s (%s)\n  %s\n", p.ID, p.Seq, p.Origin, p.Actor, p.Channel, p.Summary)
	fmt.Fprintf(&b, "Proposed against baseline version %d; the baseline is at version %d (review with seen_base_version %d).\n",
		p.BaseVersion, doc.BaselineVersion, doc.BaselineVersion)
	for _, it := range doc.Items {
		subject := ""
		switch {
		case it.Ref != nil:
			subject = "ref " + *it.Ref
		case it.Target != nil:
			subject = fmt.Sprintf("%s (revision %d)", *it.Target, *it.TargetRevision)
		}
		fmt.Fprintf(&b, "\nitem %d  %s  %s %s %s\n", it.ItemIndex, it.ItemID, it.Op, it.Type, subject)
		fmt.Fprintf(&b, "  state: %s", it.ReviewState)
		if it.SuggestedChangeClass != nil {
			fmt.Fprintf(&b, "; suggested class: %s (a suggestion only; the reviewer chooses)", *it.SuggestedChangeClass)
		}
		b.WriteString("\n")
		if it.Proposed != nil {
			fmt.Fprintf(&b, "  proposed: %s\n", compact(it.Proposed))
		}
		if it.Rationale != "" {
			fmt.Fprintf(&b, "  rationale: %s\n", it.Rationale)
		}
		for _, s := range it.Sources {
			fmt.Fprintf(&b, "  source %s: %s\n", s.Path, s.Verification)
		}
		for _, c := range it.CodeRefs {
			state := "found"
			if !c.Exists {
				state = "not found"
			}
			fmt.Fprintf(&b, "  code %s: %s\n", c.Path, state)
		}
		if c := it.Conflict; c != nil {
			fmt.Fprintf(&b, "  CONFLICT: changed since the proposal was made (revision %d then, %d and %s now)\n", c.BaseRevision,
				c.CurrentRevision, c.CurrentStatus)
			fmt.Fprintf(&b, "    base:     %s\n    current:  %s\n    proposed: %s\n", compact(c.Base), compact(c.Current), compact(c.Proposed))
		}
		if r := it.Review; r != nil {
			fmt.Fprintf(&b, "  reviewed: %s by %s (%s) at seq %d", r.Decision, r.By, r.Channel, r.Seq)
			if r.ChangeClass != nil {
				fmt.Fprintf(&b, ", %s", *r.ChangeClass)
			}
			b.WriteString("\n")
		}
	}
	if len(doc.Questions) > 0 {
		b.WriteString("\nQuestions:\n")
		for _, q := range doc.Questions {
			about := "the project"
			if q.About != nil {
				about = *q.About
			}
			fmt.Fprintf(&b, "  %s  %s, blocking %v, about %s\n", q.Key, q.Status, q.Blocking, about)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// compact is a value on one line, its keys sorted.
func compact(v any) string {
	if m, ok := v.(map[string]any); ok {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = k + "=" + compact(m[k])
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}
