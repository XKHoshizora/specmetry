package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/gitx"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/scope"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/termtext"
)

func newTargetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "target",
		Short: "Show and set the delivery target",
	}
	cmd.AddCommand(newTargetShowCmd(), newTargetSetCmd())
	return cmd
}

func newScopeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scope",
		Short: "Show and confirm the delivery scope",
	}
	cmd.AddCommand(newScopeShowCmd(), newScopeConfirmCmd())
	return cmd
}

// readScope reads the project's target and scope state, and the commit the
// target's ref resolves to now ("" when it resolves to none). Resolving
// reads the repository, so it is done outside the transaction; a commit
// that cannot be read is an error, never "" (PD-26), as for init.
func readScope(cmd *cobra.Command) (scope.State, string, error) {
	ctx := cmd.Context()
	w, err := prepare(ctx)
	if err != nil {
		return scope.State{}, "", err
	}
	defer w.close()
	if err := w.open(ctx); err != nil {
		return scope.State{}, "", err
	}
	var st scope.State
	if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
		st, err = scope.Read(ctx, tx, p.ID)
		return err
	}); err != nil {
		return scope.State{}, "", err
	}
	snapshot := ""
	if st.Target != nil && w.where.Git {
		if snapshot, err = gitx.CommitOf(ctx, w.where.Root, st.Target.Ref); err != nil {
			return scope.State{}, "", err
		}
	}
	return st, snapshot, nil
}

func newTargetShowCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the delivery target, its scope and what a confirmation needs",
		Long: `Show the active delivery target: the branch or tag it tracks, and the commit
that ref resolves to now (the snapshot, which is never stored). With --json,
the target-scope.v1 document, the same as scope show --json.`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			st, snapshot, err := readScope(cmd)
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), "target-scope.v1", scope.Document(st, snapshot))
			}
			return printf(cmd.OutOrStdout(), "%s", targetText(st, snapshot)+scopeText(st))
		}),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the target-scope.v1 document")
	return cmd
}

func newScopeShowCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the delivery scope and the values a new confirmation must carry",
		Long: `Show the delivery target's confirmed scope (whether it is stale: the baseline
version or the source set changed since), and what a new confirmation must
carry: the baseline version, the source set hash, the number of proposal
items pending or in conflict it excludes, the must requirements selected by
default, and every waiver in effect, to be acknowledged. With --json, the
target-scope.v1 document, the same as target show --json.`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			st, snapshot, err := readScope(cmd)
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), "target-scope.v1", scope.Document(st, snapshot))
			}
			return printf(cmd.OutOrStdout(), "%s", targetText(st, snapshot)+scopeText(st)+scopeConfirmText(st))
		}),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the target-scope.v1 document")
	return cmd
}

func targetText(st scope.State, snapshot string) string {
	if st.Target == nil {
		return "No delivery target (a person sets one: target set --ref <ref>).\n"
	}
	at := "which resolves to no commit now"
	if snapshot != "" {
		at = "at " + snapshot
	}
	return fmt.Sprintf("Delivery target %s: %s, %s (seq %d)\n", termtext.Line(st.Target.Name), termtext.Line(st.Target.Ref), at, st.Target.CreatedSeq)
}

func scopeText(st scope.State) string {
	if st.Target == nil {
		return ""
	}
	c := st.Latest
	if c == nil {
		return "Scope: not confirmed (a person confirms it: scope confirm)\n"
	}
	keys := make([]string, len(c.Requirements))
	for i, r := range c.Requirements {
		keys[i] = fmt.Sprintf("%s (revision %d)", r.Key, r.Revision)
	}
	stale := ""
	if st.Stale() {
		stale = " -- stale: the baseline version or the source set changed since; confirm it again"
	}
	return fmt.Sprintf("Scope %s confirmed at seq %d for baseline version %d: %s%s\n", c.ID, c.Seq, c.BaselineVersion, strings.Join(keys, ", "), stale)
}

func scopeConfirmText(st scope.State) string {
	var b strings.Builder
	b.WriteString("\nA new confirmation (scope confirm) is checked against:\n")
	fmt.Fprintf(&b, "  baseline version %d, source set hash %s\n", st.BaselineVersion, st.SourceSetHash)
	fmt.Fprintf(&b, "  %d proposal items pending or in conflict, which it excludes\n", st.PendingItems)
	must := st.MustRequirements()
	if len(must) == 0 {
		b.WriteString("  must requirements (the default selection): none\n")
	} else {
		fmt.Fprintf(&b, "  must requirements (the default selection): %s\n", strings.Join(must, ", "))
	}
	waivers := st.WaiversUnder(scopeKeys(st))
	if len(waivers) == 0 {
		b.WriteString("  waivers in effect: none\n")
	}
	for _, wv := range waivers {
		fmt.Fprintf(&b, "  waiver %s of %s (revision %d) under %s: %s\n", wv.DecisionID, wv.Criterion, wv.CriterionRevision, wv.Requirement, indented(wv.Rationale))
	}
	return b.String()
}

func scopeKeys(st scope.State) []string {
	keys := make([]string, 0, len(st.Requirements))
	for k := range st.Requirements {
		keys = append(keys, k)
	}
	return keys
}

func newTargetSetCmd() *cobra.Command {
	var ref, name string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "set --ref <ref>",
		Short: "Set the delivery target to a branch or tag (a person only)",
		Long: `Set the delivery target to a full branch or tag ref (refs/heads/... or
refs/tags/...). A person only, at a terminal, after confirming by typing the
ref. The active target is archived and a new one created; the scope
confirmed for the old one does not carry over, so confirm the scope again.
--name names the target (default: the ref without refs/heads/ or
refs/tags/). The ref need not resolve to a commit yet.`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			d := channel.FromContext(ctx)
			if err := d.RequireHuman("target set"); err != nil {
				return err
			}
			if !cmd.Flags().Changed("name") {
				name = scope.ShortName(ref)
			}
			w, err := prepare(ctx)
			if err != nil {
				return err
			}
			defer w.close()
			// Scanned before anything quotes them.
			if err := secrets.Reject(w.scanner.Fields(secrets.Field{Pointer: "--ref", Value: ref}, secrets.Field{Pointer: "--name", Value: name})); err != nil {
				return err
			}
			if _, ok := scope.RefKind(ref); !ok {
				return apperr.New(apperr.InvalidInput, "a delivery target tracks a branch or a tag: pass a full ref, refs/heads/<branch> or refs/tags/<tag>").
					WithDetails(apperr.Detail{Pointer: "--ref", Problem: "is not refs/heads/... or refs/tags/..."})
			}
			if err := scope.CheckName(name); err != nil {
				return err
			}
			actor, err := w.actor()
			if err != nil {
				return err
			}
			if err := w.open(ctx); err != nil {
				return err
			}
			var current *scope.Target
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				current, err = scope.ActiveTarget(ctx, tx, p.ID)
				return err
			}); err != nil {
				return err
			}
			if !w.where.Git {
				return apperr.New(apperr.InvalidInput, "the project is not a git repository, so it has no delivery target (FR-INIT-09)")
			}
			valid, err := gitx.ValidRefName(ctx, w.where.Root, ref)
			if err != nil {
				return err
			}
			if !valid {
				return apperr.New(apperr.InvalidInput, "the ref is not a valid ref name (see git check-ref-format)").
					WithDetails(apperr.Detail{Pointer: "--ref", Problem: "is not a valid ref name"})
			}
			// The same ref, not renamed, is the same target: its scope stays.
			if current != nil && current.Ref == ref && (!cmd.Flags().Changed("name") || current.Name == name) {
				return printf(cmd.OutOrStdout(), "%s (%s) is the delivery target already; nothing changed\n", termtext.Line(ref), termtext.Line(current.Name))
			}
			snapshot, err := gitx.CommitOf(ctx, w.where.Root, ref)
			if err != nil {
				return err
			}
			at := "it resolves to no commit now; the target is unresolved until it does"
			if snapshot != "" {
				at = "it resolves to " + snapshot + " now"
			}
			was := "the project has no delivery target now"
			var seen *string
			if current != nil {
				was = "the active target " + current.Name + " (" + current.Ref + ") is archived"
				id := current.ID
				seen = &id
			}
			op := channel.Operation{Name: "target set", Summary: []string{
				"new delivery target " + name + ": " + ref + "; " + at,
				was,
				"a scope confirmed for an earlier target does not carry over: confirm the scope for the new one (scope confirm)",
			}, Confirm: ref}
			if err := confirm(cmd, d, op); err != nil {
				return err
			}
			var made scope.Target
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				made, err = scope.SetTarget(ctx, tx, p.ID, seen, ref, name, actor, string(d.Channel))
				return err
			}); err != nil {
				return err
			}
			return printf(cmd.OutOrStdout(), "Delivery target set to %s (%s) at seq %d. Confirm its scope: scope confirm\n",
				termtext.Line(made.Ref), termtext.Line(made.Name), made.CreatedSeq)
		}),
	}
	cmd.Flags().StringVar(&ref, "ref", "", "the full branch or tag ref to deliver (refs/heads/... or refs/tags/...)")
	cmd.Flags().StringVar(&name, "name", "", "the target's name (default: the branch or tag name)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print errors as error.v1")
	return cmd
}

func newScopeConfirmCmd() *cobra.Command {
	var file, requirements, note string
	var missingGoals, asJSON bool
	cmd := &cobra.Command{
		Use:   "confirm [--requirements <keys>] [-m <note>] | --file <scope.v1>",
		Short: "Confirm the delivery scope of the target (a person only)",
		Long: `Confirm the complete delivery scope of the delivery target: the requirements
to deliver, each with all its criteria. A person only, at a terminal, after
confirming by typing the baseline version (for example v3).

By default the scope is every active must requirement; --requirements
REQ-001,REQ-004 names it instead. The confirmation binds to the current
baseline version and source set hash, excludes the proposal items pending or
in conflict, keeps every waiver in effect under its requirements, and
replaces the target's current confirmation; scope show lists all of these.
A project at stage incomplete_docs needs --missing-goals-reviewed: the goals
the code cannot reveal have been added. --file passes a scope.v1 document
instead. Anything that changes before it is written is stale (exit 5).`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			d := channel.FromContext(ctx)
			if err := d.RequireHuman("scope confirm"); err != nil {
				return err
			}
			var fromFile *scope.Input
			var doc any
			if file != "" {
				if cmd.Flags().Changed("requirements") || cmd.Flags().Changed("message") || cmd.Flags().Changed("missing-goals-reviewed") {
					return apperr.New(apperr.InvalidInput, "--file replaces --requirements, -m and --missing-goals-reviewed; give one or the other")
				}
				raw, err := readSmallFile(file)
				if err != nil {
					return err
				}
				val, err := validator()
				if err != nil {
					return apperr.Wrap(apperr.Internal, err, "cannot load the exchange schemas")
				}
				if err := val.ValidateJSON("scope.v1", raw); err != nil {
					return err
				}
				var in scope.Input
				if json.Unmarshal(raw, &in) != nil || json.Unmarshal(raw, &doc) != nil {
					return apperr.New(apperr.InvalidInput, "the scope is not valid")
				}
				fromFile = &in
			}
			w, err := prepare(ctx)
			if err != nil {
				return err
			}
			defer w.close()
			if fromFile != nil {
				err = secrets.Reject(w.scanner.JSON(doc))
			} else {
				err = secrets.Reject(w.scanner.Fields(secrets.Field{Pointer: "--requirements", Value: requirements}, secrets.Field{Pointer: "-m", Value: note}))
			}
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
			var st scope.State
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				st, err = scope.Read(ctx, tx, p.ID)
				return err
			}); err != nil {
				return err
			}
			in, err := scopeInput(st, fromFile, requirements, note, missingGoals)
			if err != nil {
				return err
			}
			reqs, err := scope.Check(st, in, "")
			if err != nil {
				return err
			}
			snapshot := ""
			if w.where.Git {
				if snapshot, err = gitx.CommitOf(ctx, w.where.Root, st.Target.Ref); err != nil {
					return err
				}
			}
			op := channel.Operation{Name: "scope confirm", Summary: confirmSummary(st, in, reqs, snapshot), Confirm: fmt.Sprintf("v%d", st.BaselineVersion)}
			if err := confirm(cmd, d, op); err != nil {
				return err
			}
			// Everything is checked again under the write lock: whatever
			// changed while the person confirmed is stale (V-T01, V-T02).
			var id string
			var seq int64
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				now, err := scope.Read(ctx, tx, p.ID)
				if err != nil {
					return err
				}
				reqs, err := scope.Check(now, in, "")
				if err != nil {
					return err
				}
				if seq, err = sqlite.NextSeq(ctx, tx); err != nil {
					return err
				}
				id, err = scope.Write(ctx, tx, p.ID, now, in, reqs, seq, actor, string(d.Channel), time.Now().Unix())
				return err
			}); err != nil {
				return err
			}
			return printf(cmd.OutOrStdout(), "Scope %s confirmed for %s at baseline version %d: %d requirements (seq %d).\n",
				id, termtext.Line(st.Target.Ref), st.BaselineVersion, len(reqs), seq)
		}),
	}
	cmd.Flags().StringVar(&file, "file", "", "read a scope.v1 document from this file instead")
	cmd.Flags().StringVar(&requirements, "requirements", "", "the requirements of the scope, separated by commas (default: every active must requirement)")
	cmd.Flags().BoolVar(&missingGoals, "missing-goals-reviewed", false, "state that the goals the code cannot reveal have been added (stage incomplete_docs)")
	cmd.Flags().StringVarP(&note, "message", "m", "", "a note kept with the confirmation")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print errors as error.v1")
	return cmd
}

// scopeInput is the confirmation to check: the --file document, or one
// built from st, the state the person is shown: the target, the version
// and the source set read, the pending count, the requirements named (by
// default every active must requirement, FR-TGT-07), the waivers in effect
// under them, and the target's current confirmation.
func scopeInput(st scope.State, fromFile *scope.Input, requirements, note string, missingGoals bool) (scope.Input, error) {
	if fromFile != nil {
		return *fromFile, nil
	}
	if st.Target == nil {
		return scope.Input{}, apperr.New(apperr.InvalidInput, "the project has no delivery target; a person sets one first (target set --ref <ref>)")
	}
	keys := st.MustRequirements()
	message := "there is no active must requirement to select by default; name the requirements with --requirements"
	if requirements != "" {
		keys = strings.FieldsFunc(requirements, func(r rune) bool { return r == ',' || r == ' ' })
		message = "--requirements names no requirement"
	}
	if len(keys) == 0 {
		return scope.Input{}, apperr.New(apperr.InvalidInput, message).
			WithDetails(apperr.Detail{Pointer: "/requirements", Problem: "is empty; a scope holds at least one requirement"})
	}
	version := st.BaselineVersion
	in := scope.Input{TargetID: st.Target.ID, BaselineVersion: &version, SourceSetHash: st.SourceSetHash, Requirements: keys,
		Attestations: scope.Attestations{CompleteScope: true, PendingItemsExcluded: st.PendingItems}, AcknowledgedWaivers: []string{}, Note: note}
	if missingGoals {
		in.Attestations.MissingGoalsReviewed = &missingGoals
	}
	for _, wv := range st.WaiversUnder(keys) {
		in.AcknowledgedWaivers = append(in.AcknowledgedWaivers, wv.DecisionID)
	}
	if st.Latest != nil {
		id := st.Latest.ID
		in.SeenScopeConfirmationID = &id
	}
	return in, nil
}

// confirmSummary is what a person confirms a scope on.
func confirmSummary(st scope.State, in scope.Input, reqs []scope.Requirement, snapshot string) []string {
	at := "which resolves to no commit now"
	if snapshot != "" {
		at = "at " + snapshot
	}
	lines := []string{
		fmt.Sprintf("delivery target %s: %s, %s", st.Target.Name, st.Target.Ref, at),
		fmt.Sprintf("bound to baseline version %d and source set hash %s", st.BaselineVersion, st.SourceSetHash),
		fmt.Sprintf("the scope, %d requirements with all their criteria:", len(reqs)),
	}
	for _, r := range reqs {
		lines = append(lines, fmt.Sprintf("  %s (revision %d, %s) %s", r.Key, r.Revision, r.Priority, r.Title))
	}
	lines = append(lines, "you confirm these are the complete delivery scope of the target",
		fmt.Sprintf("%d proposal items pending or in conflict are not in it", st.PendingItems))
	if in.Attestations.MissingGoalsReviewed != nil && *in.Attestations.MissingGoalsReviewed {
		lines = append(lines, "you state that the goals the code cannot reveal have been added (stage "+st.Stage+")")
	}
	waivers := st.WaiversUnder(keysOf(reqs))
	if len(waivers) == 0 {
		lines = append(lines, "waivers kept: none")
	}
	for _, wv := range waivers {
		lines = append(lines, fmt.Sprintf("waiver kept: %s of %s (revision %d) under %s: %s", wv.DecisionID, wv.Criterion, wv.CriterionRevision,
			wv.Requirement, wv.Rationale))
	}
	if st.Latest == nil {
		lines = append(lines, "the target's first scope confirmation")
	} else {
		lines = append(lines, fmt.Sprintf("it replaces scope %s (seq %d)", st.Latest.ID, st.Latest.Seq))
	}
	if in.Note != "" {
		lines = append(lines, "note: "+in.Note)
	}
	return lines
}

func keysOf(reqs []scope.Requirement) []string {
	keys := make([]string, len(reqs))
	for i, r := range reqs {
		keys[i] = r.Key
	}
	return keys
}
