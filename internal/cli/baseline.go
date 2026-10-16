package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/termtext"
)

func newBaselineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "baseline",
		Short: "Show the confirmed baseline and its history",
	}
	cmd.AddCommand(newBaselineShowCmd(), newBaselineHistoryCmd())
	return cmd
}

func newBaselineShowCmd() *cobra.Command {
	var includeRetired, asJSON bool
	cmd := &cobra.Command{
		Use:   "show [--include-retired]",
		Short: "Show the confirmed baseline: goals, requirements and criteria",
		Long: `Show the baseline a person confirmed, with each entity's revision: the
base_version and target_revision values a proposal carries. Items waiting
for review are not part of it.`,
		Args: cobra.NoArgs,
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
			var doc baseline.ShowDoc
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				doc, err = baseline.Show(ctx, tx, p.ID, includeRetired)
				return err
			}); err != nil {
				return err
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), "baseline-show.v1", doc)
			}
			return printBaseline(cmd.OutOrStdout(), doc)
		}),
	}
	cmd.Flags().BoolVar(&includeRetired, "include-retired", false, "also show retired entities")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the baseline-show.v1 document")
	return cmd
}

func printBaseline(w io.Writer, doc baseline.ShowDoc) error {
	var b strings.Builder
	if doc.BaselineVersion == 0 {
		b.WriteString("No baseline version yet: nothing has been reviewed and accepted.\n")
	} else {
		fmt.Fprintf(&b, "Baseline version %d (as of seq %d)\n", doc.BaselineVersion, doc.AsOfSeq)
	}
	retired := func(status string) string {
		if status == "retired" {
			return " [retired]"
		}
		return ""
	}
	for _, g := range doc.Goals {
		fmt.Fprintf(&b, "%s r%d%s  %s\n", g.Key, g.Revision, retired(g.Status), g.Title)
	}
	for _, r := range doc.Requirements {
		goal := ""
		if r.Goal != nil {
			goal = " (" + *r.Goal + ")"
		}
		fmt.Fprintf(&b, "%s r%d%s  %s, %s%s  %s\n", r.Key, r.Revision, retired(r.Status), r.Kind, r.Priority, goal, r.Title)
		for _, c := range r.Criteria {
			note := ""
			if c.RequirementRevision < r.Revision {
				note = fmt.Sprintf(" [confirmed for %s r%d: parent revised]", r.Key, c.RequirementRevision)
			}
			fmt.Fprintf(&b, "  %s r%d%s  %s%s  %s\n", c.Key, c.Revision, retired(c.Status), c.Verification, note, indented(c.Statement))
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func newBaselineHistoryCmd() *cobra.Command {
	var sinceFlag string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "history [--since <version>]",
		Short: "Show every baseline version and the changes it made",
		Args:  cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			// Parsed here so that a bad value is not echoed, as a flag
			// parse error would (codex-review-025 B1).
			since := 0
			if sinceFlag != "" {
				n, err := strconv.Atoi(sinceFlag)
				if err != nil || n < 0 {
					return apperr.New(apperr.InvalidInput, "--since takes a baseline version number, 0 or more")
				}
				since = n
			}
			w, err := prepare(ctx)
			if err != nil {
				return err
			}
			defer w.close()
			if err := w.open(ctx); err != nil {
				return err
			}
			var doc baseline.HistoryDoc
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				doc, err = baseline.History(ctx, tx, p.ID, since)
				return err
			}); err != nil {
				return err
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), "baseline-history.v1", doc)
			}
			return printHistory(cmd.OutOrStdout(), doc)
		}),
	}
	cmd.Flags().StringVar(&sinceFlag, "since", "", "only versions after this one")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the baseline-history.v1 document")
	return cmd
}

func printHistory(w io.Writer, doc baseline.HistoryDoc) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Baseline version %d. %d versions shown.\n", doc.BaselineVersion, len(doc.Versions))
	for _, v := range doc.Versions {
		summary := ""
		if v.Summary != nil {
			summary = *v.Summary
		}
		fmt.Fprintf(&b, "\nversion %d  seq %d  by %s (%s)  %s\n", v.Number, v.Seq, termtext.Line(v.Actor), v.Channel, indented(summary))
		for _, c := range v.Changes {
			class := ""
			if c.ChangeClass != nil {
				class = " (" + *c.ChangeClass + ")"
			}
			cause := ""
			if c.ProposalItemID == nil {
				cause = "  [follow-up or cascade]"
			}
			var after struct {
				Revision int `json:"revision"`
			}
			_ = json.Unmarshal(c.After, &after)
			fmt.Fprintf(&b, "  %s %s %s%s -> r%d%s\n", c.Op, c.EntityType, c.Key, class, after.Revision, cause)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}
