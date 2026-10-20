package cli

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/claim"
	"github.com/XKHoshizora/specmetry/internal/gitx"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/termtext"
)

// maxClaimsSize bounds a claim batch as a proposal is bounded.
const maxClaimsSize = maxProposalSize

// Test hooks: testHookAfterCapture runs once the code context is captured,
// testHookClaimReceipt changes the receipt before it is validated,
// testHookBeforeClaimWrite runs before the write transaction.
var (
	testHookAfterCapture     func()
	testHookClaimReceipt     func(*claim.Receipt)
	testHookBeforeClaimWrite func()
)

// errOutputCut marks an error raised once output has begun: the command's
// facts are committed and stdout may hold part of its document, so Execute
// writes the error to stderr only, never an error.v1 after that part.
var errOutputCut = errors.New("the output may be cut off")

func newClaimCmd() *cobra.Command {
	var file string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "claim --file <path|->",
		Short: "Record an Agent's claims about acceptance criteria",
		Long: `Record a claims.v1 batch: what the Agent says about acceptance criteria (sync,
an audit of a declared scope, or manual), and work that maps to no criterion.
The batch is checked against the current baseline version and each
criterion's revision, and stored with the code context of this worktree
(its HEAD and whether it is clean). A claim is a self-report: it never makes
a criterion accepted or ready, and its references are only marked as there
or not (a file of HEAD; a commit prefix naming exactly one readable commit),
which is not verification. A manual batch is a person's, at a terminal,
after confirming. Nothing is retried: on exit 5, read the baseline again.`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			d := channel.FromContext(ctx)
			if file == "" {
				return apperr.New(apperr.InvalidInput, "pass the claim batch with --file <path>, or --file - to read it from standard input")
			}
			raw, err := readClaims(file, cmd.InOrStdin())
			if err != nil {
				return err
			}
			val, err := validator()
			if err != nil {
				return apperr.Wrap(apperr.Internal, err, "cannot load the exchange schemas")
			}
			if err := val.ValidateJSON("claims.v1", raw); err != nil {
				return err
			}
			var doc any
			if err := json.Unmarshal(raw, &doc); err != nil {
				return apperr.New(apperr.InvalidInput, "the claim batch is not valid")
			}
			b, err := claim.Parse(raw)
			if err != nil {
				return err
			}
			// M2-Q6: a manual batch is a person's; the Agent channel is
			// refused before the configuration, the database or git is read.
			if b.Kind == claim.Manual {
				if err := d.RequireHuman("claim (manual)"); err != nil {
					return err
				}
			}
			if err := claim.CheckStatic(b); err != nil {
				return err
			}
			w, err := prepare(ctx)
			if err != nil {
				return err
			}
			defer w.close()
			// V-C04: every string of the batch, and the paths of a worktree
			// it may register.
			if err := secrets.Reject(w.scanner.JSON(doc)); err != nil {
				return err
			}
			if err := secrets.Reject(w.scanner.Fields(
				secrets.Field{Pointer: "worktree path", Value: w.where.Current},
				secrets.Field{Pointer: "worktree git directory", Value: w.where.GitDir})); err != nil {
				return err
			}
			if err := w.open(ctx); err != nil {
				return err
			}
			// An unregistered project is refused before the private area is
			// readied (WP2.4 spec §5 step 6).
			if err := w.tx(ctx, func(*sql.Tx, project.Registered) error { return nil }); err != nil {
				return err
			}
			var scratch *gitx.Scratch
			if w.where.Git {
				if scratch, err = gitx.NewScratch(w.dir); err != nil {
					return err
				}
				defer func() { _ = scratch.Close() }()
			}
			code, err := claim.Capture(ctx, scratch, w.where)
			if err != nil {
				return err
			}
			if testHookAfterCapture != nil {
				testHookAfterCapture()
			}
			// The file references are checked against the head captured,
			// not HEAD resolved again (WP2.4 spec §7).
			checked, err := claim.CheckRefs(ctx, scratch, w.where, code.Head, b)
			if err != nil {
				return err
			}
			if err := scratch.Close(); err != nil {
				return apperr.Wrap(apperr.Environment, err, "cannot close the data directory's tmp directory; nothing was written")
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
			sub := claim.Submission{Batch: b, Context: code, Checked: checked, Where: w.where, Actor: actor, Channel: string(d.Channel)}
			var receipt claim.Receipt
			var out []byte
			// The receipt is built from the rows written and validated, and
			// what is printed encoded, before COMMIT (WP2.4 spec §9.3).
			store := func(tx *sql.Tx, proj project.Registered) error {
				r, err := claim.Store(ctx, tx, proj.ID, sub)
				if err != nil {
					return err
				}
				if testHookClaimReceipt != nil {
					testHookClaimReceipt(&r)
				}
				if out, err = claimOutput(r, asJSON); err != nil {
					return err
				}
				receipt = r
				return nil
			}
			if b.Kind == claim.Manual {
				// A dry run first, rolled back: nobody is asked to confirm
				// what cannot be written.
				if err := dryRun(ctx, w, store); err != nil {
					return err
				}
				op := channel.Operation{Name: "claim (manual)", Summary: manualSummary(b, checked), Confirm: fmt.Sprintf("v%d", b.BaseVersion)}
				if err := confirm(cmd, d, op); err != nil {
					return err
				}
			}
			if testHookBeforeClaimWrite != nil {
				testHookBeforeClaimWrite()
			}
			if err := w.tx(ctx, store); err != nil {
				return err
			}
			// After COMMIT only the output is left: a write that fails or
			// comes up short cannot be undone, nor followed by a second
			// document.
			var dst io.Writer = cmd.OutOrStdout()
			if asJSON {
				dst = termtext.Raw(dst)
			}
			if n, err := dst.Write(out); err != nil || n != len(out) {
				return apperr.Wrap(apperr.Environment, errOutputCut, fmt.Sprintf(
					"the claim batch %s was stored at seq %d, but its output could not be written in full and may be cut off; do not submit it again before reading the status: a second submission stores another batch",
					receipt.BatchID, receipt.Seq))
			}
			return nil
		}),
	}
	cmd.Flags().StringVar(&file, "file", "", "the claims.v1 batch, or - for standard input")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the claim-receipt.v1 document")
	return cmd
}

// readClaims reads the batch from a file or, for "-", standard input.
// Errors do not name the file: its path has not been scanned for secrets.
func readClaims(file string, stdin io.Reader) ([]byte, error) {
	if file != "-" {
		return readInput(file, maxClaimsSize)
	}
	b, err := io.ReadAll(io.LimitReader(stdin, maxClaimsSize+1))
	if err != nil {
		return nil, apperr.New(apperr.InvalidInput, "cannot read the claim batch from standard input")
	}
	if len(b) > maxClaimsSize {
		return nil, apperr.Newf(apperr.InvalidInput, "the claim batch is larger than %d MiB", maxClaimsSize>>20)
	}
	return b, nil
}

// claimOutput is what the command prints on success: the claim-receipt.v1
// document, validated against its schema (a document that does not
// conform is an internal error), or the text.
func claimOutput(r claim.Receipt, asJSON bool) ([]byte, error) {
	if !asJSON {
		return []byte(claimText(r)), nil
	}
	val, err := validator()
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, err, "cannot load the exchange schemas")
	}
	if err := val.ValidateValue("claim-receipt.v1", r); err != nil {
		e := apperr.From(err)
		return nil, apperr.Newf(apperr.Internal, "the claim-receipt.v1 output does not conform to its schema; nothing was written: %s", e.Message).WithDetails(e.Details...)
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, err, "cannot encode the output")
	}
	return append(b, '\n'), nil
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// refCounts counts a reference result's marks.
type refCounts struct{ exists, missing, resolved, unknown, tests int }

func (c *refCounts) add(r claim.ReceiptRefs) {
	for _, f := range r.Files {
		if f.Exists {
			c.exists++
		} else {
			c.missing++
		}
	}
	for _, m := range r.Commits {
		if m.Resolved {
			c.resolved++
		} else {
			c.unknown++
		}
	}
	c.tests += r.Tests
}

const notVerification = "A reference that is there is not verification or acceptance: a claim alone never makes a criterion accepted or ready."

// claimText is the text output: the batch, its context and comparison
// start, the counts of the reference marks and where each mark is, and no
// summary or test name.
func claimText(r claim.Receipt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Claim batch %s stored at seq %d (%s, base version %d): %s, %s\n", r.BatchID, r.Seq, r.Kind, r.BaseVersion,
		plural(len(r.Claims), "claim"), plural(len(r.UnmappedWork), "unmapped work item"))
	head := "none"
	if r.Context.Head != nil {
		head = *r.Context.Head
	}
	fmt.Fprintf(&b, "Code context: worktree %s, HEAD %s, %s\n", r.Context.WorktreeID, head, r.Context.TreeState)
	if r.CompareFrom != nil {
		if r.CompareFrom.Commit != nil {
			fmt.Fprintf(&b, "Comparison start: %s (%s)\n", *r.CompareFrom.Commit, strings.ReplaceAll(r.CompareFrom.Source, "_", " "))
		} else {
			b.WriteString("Comparison start: none\n")
		}
	}
	var counts refCounts
	var notes []string
	mark := func(name string, refs claim.ReceiptRefs) {
		counts.add(refs)
		for k, f := range refs.Files {
			if !f.Exists {
				notes = append(notes, fmt.Sprintf("  %s, file reference %d: missing", name, k))
			}
		}
		for k, c := range refs.Commits {
			if !c.Resolved {
				notes = append(notes, fmt.Sprintf("  %s, commit reference %d: unknown_commit", name, k))
			}
		}
	}
	for _, c := range r.Claims {
		mark(fmt.Sprintf("claim %d (%s)", c.Index, c.Target), c.Refs)
	}
	for _, u := range r.UnmappedWork {
		mark(fmt.Sprintf("unmapped work %d", u.Index), u.Refs)
	}
	fmt.Fprintf(&b, "References: files %d exist, %d missing; commits %d resolved, %d unknown_commit; %s (not checked)\n",
		counts.exists, counts.missing, counts.resolved, counts.unknown, plural(counts.tests, "test name"))
	for _, n := range notes {
		b.WriteString(n + "\n")
	}
	b.WriteString(notVerification + "\n")
	return b.String()
}

// manualSummary is what a person confirms a manual batch from (E1): its
// kind, base version and counts, each claim's criterion, revision,
// assertion and reference marks, and the unmapped work's; no summary.
func manualSummary(b *claim.Batch, checked claim.Checked) []string {
	lines := []string{fmt.Sprintf("kind %s, base version %d: %s, %s", b.Kind, b.BaseVersion,
		plural(len(b.Claims), "claim"), plural(len(b.UnmappedWork), "unmapped work item"))}
	marks := func(r claim.RefResults, tests int) string {
		var parts []string
		if len(r.Files) > 0 {
			exists := 0
			for _, f := range r.Files {
				if f.Exists {
					exists++
				}
			}
			parts = append(parts, fmt.Sprintf("file references %d exists, %d missing", exists, len(r.Files)-exists))
		}
		if len(r.Commits) > 0 {
			resolved := 0
			for _, c := range r.Commits {
				if c.Resolved {
					resolved++
				}
			}
			parts = append(parts, fmt.Sprintf("commit references %d resolved, %d unknown_commit", resolved, len(r.Commits)-resolved))
		}
		if tests > 0 {
			parts = append(parts, plural(tests, "test name")+" (not checked)")
		}
		if len(parts) == 0 {
			return "no references"
		}
		return strings.Join(parts, "; ")
	}
	for i, c := range b.Claims {
		lines = append(lines, fmt.Sprintf("claim %d: %s revision %d, %s; %s", i, c.Target, c.TargetRevision, c.Assertion, marks(checked.Claims[i], len(c.Refs.Tests))))
	}
	for i, u := range b.UnmappedWork {
		lines = append(lines, fmt.Sprintf("unmapped work %d: %s", i, marks(checked.Unmapped[i], len(u.Refs.Tests))))
	}
	return append(lines, notVerification)
}
