package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/proposal"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/source"
)

// maxProposalSize bounds a proposal document: 500 items with their
// references fit with room to spare.
const maxProposalSize = 8 << 20

func newProposeCmd() *cobra.Command {
	var file string
	var apply, asJSON bool
	cmd := &cobra.Command{
		Use:   "propose --file <path|->",
		Short: "Submit proposed baseline changes for a person's review",
		Long: `Submit a proposal.v1 document: goals, requirements and acceptance criteria to
add, update, retire or reaffirm, and questions to clarify. Nothing in the
baseline changes: the items wait as pending until a person reviews them, and
the questions are opened at once.

Every excerpt is checked against its registered source in memory only, and
its text is not stored (hash_only). "verified" means the excerpt appears in
the source (whitespace aside); it does not show that an item is right or
complete. Code references are checked against the HEAD commit of this
worktree; that a file is there is no evidence that a criterion is met.`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			d := channel.FromContext(ctx)
			if apply {
				if err := d.RequireHuman("propose --apply"); err != nil {
					return err
				}
			}
			if file == "" {
				return apperr.New(apperr.InvalidInput, "pass the proposal with --file <path>, or --file - to read it from standard input")
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
			if err := w.open(ctx); err != nil {
				return err
			}
			val, err := validator()
			if err != nil {
				return apperr.Wrap(apperr.Internal, err, "cannot load the exchange schemas")
			}
			if err := val.ValidateJSON("proposal.v1", raw); err != nil {
				return err
			}
			var doc any
			if err := json.Unmarshal(raw, &doc); err != nil {
				return apperr.New(apperr.InvalidInput, "the proposal is not valid")
			}
			p, err := proposal.Parse(raw)
			if err != nil {
				return err
			}
			// V-P01: human_edit is a person's edit, applied at once with
			// propose --apply; it is never taken as a pending proposal.
			if p.Origin == "human_edit" {
				if err := d.RequireHuman("a proposal with origin human_edit"); err != nil {
					return err
				}
			}
			if err := applyOrigin(p, apply); err != nil {
				return err
			}
			if err := proposal.CheckStatic(doc, p); err != nil {
				return err
			}
			// V-P05: every string of the proposal, excerpts included, and the
			// paths of a worktree this proposal may register.
			if err := secrets.Reject(w.scanner.JSON(doc)); err != nil {
				return err
			}
			if err := secrets.Reject(w.scanner.Fields(
				secrets.Field{Pointer: "worktree path", Value: w.where.Current},
				secrets.Field{Pointer: "worktree git directory", Value: w.where.GitDir})); err != nil {
				return err
			}
			actor := "agent:unknown"
			if d.Agent != "" {
				actor = "agent:" + d.Agent
			}
			if d.Channel == channel.HumanCLI {
				if actor, err = w.actor(); err != nil {
					return err
				}
			}

			var sources map[string]source.Source
			var root string
			if err := w.tx(ctx, func(tx *sql.Tx, proj project.Registered) error {
				root = proj.Where.Root
				sources, err = proposal.ResolveSources(ctx, tx, proj.ID, p)
				return err
			}); err != nil {
				return err
			}
			// The sources and the code are read outside the transaction.
			checked, err := proposal.CheckSources(ctx, root, p, sources)
			if err != nil {
				return err
			}
			codeRefs := make([][]proposal.CodeRef, len(p.Items))
			for i, it := range p.Items {
				codeRefs[i] = it.CodeRefs
			}
			if checked.CodeRefs, err = proposal.CheckCodeRefs(ctx, w.where, codeRefs); err != nil {
				return err
			}
			var edit *humanEdit
			var receipt proposal.Receipt
			store := func(tx *sql.Tx, proj project.Registered) error {
				receipt, err = proposal.Store(ctx, tx, proj.ID, proposal.Submission{
					Proposal: p, Checked: checked, Where: w.where, Actor: actor, Channel: string(d.Channel)})
				if err != nil || edit == nil {
					return err
				}
				return edit.apply(ctx, tx, proj.ID, p, checked, &receipt, actor, string(d.Channel))
			}
			if apply {
				if edit, err = prepareEdit(ctx, w, p); err != nil {
					return err
				}
				// A dry run first, rolled back: nobody is asked to confirm
				// what cannot be applied.
				if err := dryRun(ctx, w, store); err != nil {
					return err
				}
				op := channel.Operation{Name: "propose --apply", Summary: edit.lines, Confirm: fmt.Sprintf("v%d", p.BaseVersion+1)}
				if err := confirm(cmd, d, op); err != nil {
					return err
				}
			}
			if err := w.tx(ctx, store); err != nil {
				return err
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), "proposal-receipt.v1", receipt)
			}
			if apply {
				return printApplied(cmd.OutOrStdout(), receipt)
			}
			return printReceipt(cmd.OutOrStdout(), receipt)
		}),
	}
	cmd.Flags().StringVar(&file, "file", "", "the proposal.v1 document, or - for standard input")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply a person's edit (origin human_edit) at once, as a new baseline version (a person at a terminal only)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the proposal-receipt.v1 document")
	return cmd
}

// readProposal reads the proposal from a file or, for "-", standard input.
// Errors do not name the file: its path has not been scanned for secrets.
func readProposal(file string, stdin io.Reader) ([]byte, error) {
	if file != "-" {
		return readInput(file, maxProposalSize)
	}
	b, err := io.ReadAll(io.LimitReader(stdin, maxProposalSize+1))
	if err != nil {
		return nil, apperr.New(apperr.InvalidInput, "cannot read the proposal from standard input")
	}
	if len(b) > maxProposalSize {
		return nil, apperr.Newf(apperr.InvalidInput, "the proposal is larger than %d MiB", maxProposalSize>>20)
	}
	return b, nil
}

func printReceipt(w io.Writer, r proposal.Receipt) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Proposal %s stored at seq %d (base version %d): %d items pending review", r.ProposalID, r.Seq, r.BaseVersion, len(r.Items))
	if len(r.Questions) > 0 {
		keys := make([]string, len(r.Questions))
		for i, q := range r.Questions {
			keys[i] = q.Key
		}
		fmt.Fprintf(&b, ", %d questions opened (%s)", len(r.Questions), strings.Join(keys, ", "))
	}
	b.WriteString("\n")
	counts := map[proposal.Verification]int{}
	var notes []string
	found, missing := 0, 0
	for _, it := range r.Items {
		name := fmt.Sprintf("item %d", it.ItemIndex)
		if it.Ref != nil {
			name += " (" + *it.Ref + ")"
		} else if it.Target != nil {
			name += " (" + *it.Target + ")"
		}
		for k, s := range it.Sources {
			counts[s.Verification]++
			if s.Verification != proposal.Verified {
				notes = append(notes, fmt.Sprintf("  %s, source reference %d: %s", name, k, s.Verification))
			}
		}
		for k, c := range it.CodeRefs {
			if c.Exists {
				found++
			} else {
				missing++
				notes = append(notes, fmt.Sprintf("  %s, code reference %d: not in this worktree's HEAD commit", name, k))
			}
		}
	}
	if total := counts[proposal.Verified] + counts[proposal.NotFound] + counts[proposal.SourceMissing] + counts[proposal.NotCheckable]; total > 0 {
		fmt.Fprintf(&b, "Excerpts of items: %d verified, %d not found, %d source missing, %d not checkable\n",
			counts[proposal.Verified], counts[proposal.NotFound], counts[proposal.SourceMissing], counts[proposal.NotCheckable])
		b.WriteString("  (verified: the excerpt appears in its source; this does not show that an item is right or complete)\n")
	}
	if found+missing > 0 {
		fmt.Fprintf(&b, "Code references: %d found, %d not found (a file being there is no evidence that a criterion is met)\n", found, missing)
	}
	for _, n := range notes {
		b.WriteString(n + "\n")
	}
	b.WriteString("Nothing in the baseline changes until a person reviews the proposal.\n")
	_, err := io.WriteString(w, b.String())
	return err
}
