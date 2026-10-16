package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/review"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/source"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
)

func newReviewCmd() *cobra.Command {
	var file string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "review --file <path>",
		Short: "Review a proposal's items: accept, edit, reject (a person only)",
		Long: `Apply a person's decisions on a proposal's items (a review.v1 document): accept,
accept with edits, or reject each item; an accepted update needs your
change_class (material or editorial), and an item in conflict needs
acknowledged_conflict after you looked at it (proposal show). For criteria of
a requirement you revise materially, criterion_followups reaffirm, retire or
defer them. Everything accepted becomes one new baseline version; nothing
accepted, no version. A person only, at a terminal, after confirming;
seen_base_version must be the current baseline version.`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			d := channel.FromContext(ctx)
			if err := d.RequireHuman("review"); err != nil {
				return err
			}
			if file == "" {
				return apperr.New(apperr.InvalidInput, "pass the review with --file <path> (a review.v1 document)")
			}
			raw, err := readProposal(file, cmd.InOrStdin())
			if err != nil {
				return err
			}
			w, err := prepare(ctx)
			if err != nil {
				return err
			}
			defer w.close()
			val, err := validator()
			if err != nil {
				return apperr.Wrap(apperr.Internal, err, "cannot load the exchange schemas")
			}
			if err := val.ValidateJSON("review.v1", raw); err != nil {
				return err
			}
			var doc any
			if err := json.Unmarshal(raw, &doc); err != nil {
				return apperr.New(apperr.InvalidInput, "the review is not valid")
			}
			// Notes, edits, summaries and rationales are free text.
			if err := secrets.Reject(w.scanner.JSON(doc)); err != nil {
				return err
			}
			rv, err := review.Parse(raw)
			if err != nil {
				return err
			}
			actor, err := w.actor()
			if err != nil {
				return err
			}
			if err := w.open(ctx); err != nil {
				return err
			}
			// review.Check compares seen_base_version with the current
			// version first, in each transaction: before the prompt, in the
			// dry run and when writing, so that a stale review is exit 5
			// even if the baseline moved while the person was confirming
			// (codex-review-025 B2).
			var checked review.Checked
			var sources map[string]source.Source
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				if checked, err = review.Check(ctx, tx, p.ID, rv, val); err != nil {
					return err
				}
				sources, err = review.CarriedSources(ctx, tx, p.ID, checked.Carry)
				return err
			}); err != nil {
				return err
			}
			carried, err := review.Recheck(ctx, w.where, sources, checked.Carry)
			if err != nil {
				return err
			}
			var result baseline.Result
			var rejected int
			apply := func(tx *sql.Tx, p project.Registered) error {
				c, err := review.Check(ctx, tx, p.ID, rv, val)
				if err != nil {
					return err
				}
				if err := c.Carried(carried); err != nil {
					return err
				}
				if c.Plan.Seq, err = sqlite.NextSeq(ctx, tx); err != nil {
					return err
				}
				c.Plan.Actor, c.Plan.Channel = actor, string(d.Channel)
				rejected = len(c.Plan.Rejected)
				result, err = baseline.Apply(ctx, tx, c.Plan)
				return err
			}
			// A dry run first, rolled back: nobody is asked to confirm
			// what cannot be applied.
			if err := dryRun(ctx, w, apply); err != nil {
				return err
			}
			op := channel.Operation{Name: "review", Summary: checked.Lines, Confirm: checked.Proposal.ID}
			if err := confirm(cmd, d, op); err != nil {
				return err
			}
			if err := w.tx(ctx, apply); err != nil {
				return err
			}
			return printReviewResult(cmd.OutOrStdout(), result, rejected)
		}),
	}
	cmd.Flags().StringVar(&file, "file", "", "the review.v1 document")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print errors as error.v1")
	return cmd
}

func printReviewResult(w io.Writer, r baseline.Result, rejected int) error {
	var b strings.Builder
	if r.Version == 0 {
		fmt.Fprintf(&b, "%d items rejected. Nothing was accepted, so no baseline version was created.\n", rejected)
	} else {
		fmt.Fprintf(&b, "Baseline version %d created: %d changes recorded, %d items rejected.\n", r.Version, r.Changes, rejected)
		keys := make([]string, 0, len(r.Keys))
		for _, k := range r.Keys {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		if len(keys) > 0 {
			fmt.Fprintf(&b, "New keys: %s\n", strings.Join(keys, ", "))
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// errDryRun rolls a dry run's transaction back.
var errDryRun = errors.New("dry run")

// dryRun runs fn in a write transaction that is always rolled back, and
// returns what fn returned.
func dryRun(ctx context.Context, w *workspace, fn func(tx *sql.Tx, p project.Registered) error) error {
	err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
		if err := fn(tx, p); err != nil {
			return err
		}
		return errDryRun
	})
	if errors.Is(err, errDryRun) {
		return nil
	}
	return err
}
