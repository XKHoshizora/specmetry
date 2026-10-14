package cli

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"strings"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/config"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/source"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
)

// The documents of source-list.v1 and source-add.v1.
type sourceDoc struct {
	ID            string          `json:"id"`
	Path          string          `json:"path"`
	Location      source.Location `json:"location"`
	Kind          source.Kind     `json:"kind"`
	AckedSHA256   string          `json:"acked_sha256"`
	CurrentSHA256 *string         `json:"current_sha256"`
	State         *source.State   `json:"state"`
	SizeBytes     int64           `json:"size_bytes"`
	RegisteredSeq int64           `json:"registered_seq"`
	RemovedSeq    *int64          `json:"removed_seq"`
	Refs          *[]any          `json:"refs,omitempty"`
}

type sourceListDoc struct {
	Schema         string      `json:"schema"`
	ExcerptStorage string      `json:"excerpt_storage"`
	SourceSetHash  string      `json:"source_set_hash"`
	Sources        []sourceDoc `json:"sources"`
}

type sourceAddDoc struct {
	Schema        string    `json:"schema"`
	Created       bool      `json:"created"`
	Source        sourceDoc `json:"source"`
	SourceSetHash string    `json:"source_set_hash"`
}

// sourceAckDoc is the source-ack.v1 input.
type sourceAckDoc struct {
	Schema         string `json:"schema"`
	Source         string `json:"source"`
	ObservedSHA256 string `json:"observed_sha256"`
	Rationale      string `json:"rationale"`
}

func newSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Register, list, acknowledge and remove requirement sources",
	}
	cmd.AddCommand(newSourceAddCmd(), newSourceListCmd(), newSourceAckCmd(), newSourceRemoveCmd())
	return cmd
}

// workspace is the project of the current directory, with its database.
type workspace struct {
	db      *sqlite.DB
	dir     datadir.Dir
	cfg     config.Config
	scanner *secrets.Scanner
	where   project.Where
	cwd     string
}

func (w *workspace) close() {
	if w.db != nil {
		_ = w.db.Close()
	}
}

// prepare loads the configuration and finds the project directory without
// touching the data directory.
func prepare(ctx context.Context) (*workspace, error) {
	w := &workspace{}
	var err error
	if w.dir, err = datadir.Resolve(); err != nil {
		return nil, err
	}
	if w.cfg, _, err = config.Load(w.dir.Join(datadir.ConfigFile)); err != nil {
		return nil, err
	}
	if w.scanner, err = secrets.New(w.cfg.Secrets.ExtraRules); err != nil {
		return nil, apperr.Wrap(apperr.Environment, err, "invalid secret rules")
	}
	if w.cwd, err = os.Getwd(); err != nil {
		return nil, apperr.Wrap(apperr.Environment, err, "cannot determine the current directory")
	}
	if w.where, err = project.Locate(ctx, w.cwd); err != nil {
		return nil, err
	}
	return w, nil
}

// open opens the database. Without one there is no project yet, and
// nothing is created.
func (w *workspace) open(ctx context.Context) error {
	if _, err := os.Lstat(w.dir.Join(datadir.DatabaseFile)); errors.Is(err, fs.ErrNotExist) {
		return project.ErrNotInitialized()
	}
	db, _, err := sqlite.Open(ctx, w.dir, sqlite.Options{})
	if err != nil {
		return err
	}
	w.db = db
	return nil
}

// tx runs fn with the registered project in one transaction.
func (w *workspace) tx(ctx context.Context, fn func(tx *sql.Tx, p project.Registered) error) error {
	return w.db.WriteTx(ctx, func(tx *sql.Tx) error {
		p, err := project.Lookup(ctx, tx, w.where)
		if err != nil {
			return err
		}
		return fn(tx, p)
	})
}

// actor is the person running a human-only command (data-model §1): the
// display name from config.toml, or the operating system user.
func (w *workspace) actor() (string, error) {
	name := strings.TrimSpace(w.cfg.User.DisplayName)
	if name == "" {
		if u, err := user.Current(); err == nil {
			name = u.Username
		}
	}
	if name == "" {
		name = "local"
	}
	if err := secrets.Reject(w.scanner.Fields(secrets.Field{Pointer: "user.display_name", Value: name})); err != nil {
		return "", err
	}
	return "human:" + name, nil
}

// doc is a source as source-list.v1 and source-add.v1 show it; refs, when
// not nil, are its references (--refs).
func doc(s source.Source, obs *source.Observation, refs []source.Ref) sourceDoc {
	d := sourceDoc{
		ID: s.ID, Path: s.Path, Location: s.Location, Kind: s.Kind, AckedSHA256: s.AckedSHA256,
		SizeBytes: s.Size, RegisteredSeq: s.RegisteredSeq, RemovedSeq: s.RemovedSeq,
	}
	if obs != nil {
		state := obs.State
		d.State = &state
		if obs.CurrentSHA256 != "" {
			current := obs.CurrentSHA256
			d.CurrentSHA256 = &current
		}
	}
	if refs != nil {
		list := make([]any, len(refs))
		for i, r := range refs {
			list[i] = r
		}
		d.Refs = &list
	}
	return d
}

// confirmText is what a person types to confirm an operation on a source:
// its path when it is in the repository, its ID otherwise (an absolute path
// is long and may need an input method).
func confirmText(s source.Source) string {
	if s.Location == source.InRepo && strings.TrimSpace(s.Path) == s.Path {
		return s.Path
	}
	return s.ID
}

func newSourceAddCmd() *cobra.Command {
	var kindFlag string
	var allowAny, asJSON bool
	cmd := &cobra.Command{
		Use:   "add <path>",
		Short: "Register a document as a requirement source",
		Long: `Register a document as a requirement source: its path, kind, size and sha256.
Only document types are accepted unless a person, at a terminal, passes
--allow-any-type and confirms. Registering an active source again changes
nothing, not even its acknowledged hash: only source ack does that.`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			kind, ok := source.ParseKind(kindFlag)
			if !ok {
				return apperr.New(apperr.InvalidInput, "invalid source kind").WithDetails(apperr.Detail{
					Pointer: "--kind", Problem: "must be client_requirement, product_doc, design_doc, issue, note or other"})
			}
			d := channel.FromContext(ctx)
			if allowAny {
				if err := d.RequireHuman("source add --allow-any-type"); err != nil {
					return err
				}
			}
			w, err := prepare(ctx)
			if err != nil {
				return err
			}
			defer w.close()
			// Errors below may name the path, and the path is stored.
			if err := secrets.Reject(w.scanner.Fields(secrets.Field{Pointer: "path", Value: args[0]})); err != nil {
				return err
			}
			r, err := source.Resolve(ctx, args[0], w.cwd, w.where)
			if err != nil {
				return err
			}
			if err := secrets.Reject(w.scanner.Fields(secrets.Field{Pointer: "path", Value: r.Path})); err != nil {
				return err
			}
			// The whitelist decides whether a source may be created. An
			// active source, which a person may have registered with
			// --allow-any-type, is returned whatever its type (MVP-SRC-07).
			mayCreate := allowAny || project.IsDocumentType(r.File)
			notDocument := apperr.Newf(apperr.InvalidInput,
				"%s is not a document type (data-model §4.3, V-S01); only a person at a terminal can register it, with --allow-any-type", r.Path)
			if err := w.open(ctx); err != nil {
				return err
			}
			var existing *source.Source
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				s, found, err := source.FindActive(ctx, tx, p.ID, r.Path)
				if found {
					existing = &s
				}
				return err
			}); err != nil {
				return err
			}
			if existing == nil && !mayCreate {
				return notDocument
			}
			if allowAny && existing == nil {
				op := channel.Operation{
					Name: "source add --allow-any-type",
					Summary: []string{
						"register " + r.Path + " as a source although it is not a document type",
						"its content is read only to hash it; excerpts cited from it are checked against it",
					},
					Confirm: confirmText(source.Source{Location: r.Location, Path: r.Path}),
				}
				if op.Confirm == "" {
					op.Confirm = r.Path
				}
				if err := channel.Guard(d, os.Stdin, cmd.OutOrStdout(), op); err != nil {
					return err
				}
			}
			sum, size, err := source.HashFile(ctx, r.File)
			if errors.Is(err, source.ErrMissing) {
				return apperr.Newf(apperr.InvalidInput, "%s cannot be read", r.Path)
			}
			if err != nil {
				return err
			}
			var out sourceAddDoc
			err = w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				// Checked again in the write transaction: the source may have
				// been removed meanwhile.
				if _, found, err := source.FindActive(ctx, tx, p.ID, r.Path); err != nil || (!found && !mayCreate) {
					if err == nil {
						err = notDocument
					}
					return err
				}
				s, created, err := source.Register(ctx, tx, p.ID, r, kind, sum, size)
				if err != nil {
					return err
				}
				obs := source.Observation{CurrentSHA256: sum, State: source.Changed}
				if sum == s.AckedSHA256 {
					obs.State = source.Unchanged
				}
				out = sourceAddDoc{Schema: "specmetry.source-add/v1", Created: created, Source: doc(s, &obs, nil)}
				out.SourceSetHash, err = source.ProjectSetHash(ctx, tx, p.ID)
				return err
			})
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), "source-add.v1", out)
			}
			if out.Created {
				return printf(cmd.OutOrStdout(), "Registered %s (%s) as a %s source\n  sha256 %s, %d bytes\n",
					out.Source.Path, out.Source.ID, out.Source.Kind, out.Source.AckedSHA256, out.Source.SizeBytes)
			}
			return printf(cmd.OutOrStdout(), "%s is already a source (%s); nothing changed (state: %s)\n",
				out.Source.Path, out.Source.ID, *out.Source.State)
		}),
	}
	cmd.Flags().StringVar(&kindFlag, "kind", "other", "client_requirement, product_doc, design_doc, issue, note or other")
	cmd.Flags().BoolVar(&allowAny, "allow-any-type", false, "register a file that is not a document type (a person at a terminal only)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the source-add.v1 document")
	return cmd
}

func newSourceListCmd() *cobra.Command {
	var includeRemoved, refs, asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the sources with their state on disk",
		Long: `List the sources. Every list hashes the files again: unchanged, changed (the
file differs from its acknowledged hash) or missing (it cannot be read). A
change is never acknowledged by listing it; only a person's source ack does
that.`,
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
			var list []source.Source
			var refsOf map[string][]source.Ref
			out := sourceListDoc{Schema: "specmetry.source-list/v1", Sources: []sourceDoc{}}
			var root string
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				out.ExcerptStorage, root = p.ExcerptStorage, p.Where.Root
				if list, err = source.List(ctx, tx, p.ID, includeRemoved); err != nil {
					return err
				}
				if refs {
					if refsOf, err = source.Refs(ctx, tx, p.ID); err != nil {
						return err
					}
				}
				out.SourceSetHash, err = source.ProjectSetHash(ctx, tx, p.ID)
				return err
			}); err != nil {
				return err
			}
			listed := func(s source.Source) []source.Ref {
				if !refs {
					return nil
				}
				if r := refsOf[s.ID]; r != nil {
					return r
				}
				return []source.Ref{}
			}
			// The files are hashed outside the transaction.
			for _, s := range list {
				if s.RemovedSeq != nil {
					out.Sources = append(out.Sources, doc(s, nil, listed(s)))
					continue
				}
				obs, err := source.Observe(ctx, root, s)
				if err != nil {
					return err
				}
				out.Sources = append(out.Sources, doc(s, &obs, listed(s)))
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), "source-list.v1", out)
			}
			return printSources(cmd.OutOrStdout(), out)
		}),
	}
	cmd.Flags().BoolVar(&includeRemoved, "include-removed", false, "also list removed sources")
	cmd.Flags().BoolVar(&refs, "refs", false, "list each source's references")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the source-list.v1 document")
	return cmd
}

func printSources(w io.Writer, out sourceListDoc) error {
	var b bytes.Buffer
	if len(out.Sources) == 0 {
		b.WriteString("No sources.\n")
	}
	for _, s := range out.Sources {
		state := "removed"
		if s.State != nil {
			state = string(*s.State)
		}
		fmt.Fprintf(&b, "%-9s %s (%s, %s)\n", state, s.Path, s.Kind, s.ID)
		if s.State != nil && *s.State == source.Changed {
			fmt.Fprintf(&b, "          acknowledged %s, on disk %s\n", s.AckedSHA256, *s.CurrentSHA256)
		}
	}
	fmt.Fprintf(&b, "source set hash %s (excerpt storage: %s)\n", out.SourceSetHash, out.ExcerptStorage)
	return printf(w, "%s", b.String())
}

func newSourceAckCmd() *cobra.Command {
	var observed, rationale, file string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "ack <path|id> --observed <sha256> -m <rationale>",
		Short: "Acknowledge that a changed source has been handled (a person only)",
		Long: `Acknowledge a change to a source: the baseline was updated, or the change
needs none. A person only, at a terminal, after confirming. --observed must
be the sha256 now on disk (see source list); the acknowledgement records
it, the rationale and who made it.`,
		Args: cobra.MaximumNArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			d := channel.FromContext(ctx)
			if err := d.RequireHuman("source ack"); err != nil {
				return err
			}
			input, err := ackInput(cmd, args, observed, rationale, file)
			if err != nil {
				return err
			}
			w, err := prepare(ctx)
			if err != nil {
				return err
			}
			defer w.close()
			if err := secrets.Reject(w.scanner.Fields(
				secrets.Field{Pointer: "/rationale", Value: input.Rationale},
				secrets.Field{Pointer: "/source", Value: input.Source})); err != nil {
				return err
			}
			actor, err := w.actor()
			if err != nil {
				return err
			}
			if err := w.open(ctx); err != nil {
				return err
			}
			var s source.Source
			var root string
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				var found bool
				s, found, err = source.FindActive(ctx, tx, p.ID, input.Source)
				if err == nil && !found {
					err = apperr.Newf(apperr.NotFound, "%s is not an active source (see source list)", input.Source)
				}
				root = p.Where.Root
				return err
			}); err != nil {
				return err
			}
			obs, err := source.Observe(ctx, root, s)
			if err != nil {
				return err
			}
			// A stale observed hash is stale whatever the state (MVP-SRC-03).
			if err := ackPrecondition(s, obs, input.ObservedSHA256); err != nil {
				return err
			}
			if obs.State == source.Unchanged {
				return printf(cmd.OutOrStdout(), "%s matches its acknowledged hash; nothing to acknowledge\n", s.Path)
			}
			op := channel.Operation{
				Name: "source ack",
				Summary: []string{
					"source: " + s.Path + " (" + s.ID + ")",
					"acknowledged hash: " + s.AckedSHA256,
					"new acknowledged hash: " + input.ObservedSHA256 + " (on disk now)",
					"rationale: " + input.Rationale,
					"the source set hash changes, so a confirmed delivery scope must be confirmed again",
				},
				Confirm: confirmText(s),
			}
			if err := channel.Guard(d, os.Stdin, cmd.OutOrStdout(), op); err != nil {
				return err
			}
			// What counts is the disk when the acknowledgement is written,
			// not when the prompt was shown.
			if obs, err = source.Observe(ctx, root, s); err != nil {
				return err
			}
			if err := ackPrecondition(s, obs, input.ObservedSHA256); err != nil {
				return err
			}
			var seq int64
			var setHash string
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				current, found, err := source.FindActive(ctx, tx, p.ID, s.ID)
				if err == nil && !found {
					err = apperr.Newf(apperr.NotFound, "source %s was removed meanwhile", s.ID)
				}
				if err != nil {
					return err
				}
				if seq, err = source.Acknowledge(ctx, tx, p.ID, current, input.ObservedSHA256, input.Rationale, actor, string(channel.HumanCLI)); err != nil {
					return err
				}
				setHash, err = source.ProjectSetHash(ctx, tx, p.ID)
				return err
			}); err != nil {
				return err
			}
			return printf(cmd.OutOrStdout(), "Acknowledged %s at %s (seq %d)\nsource set hash %s\n", s.Path, input.ObservedSHA256, seq, setHash)
		}),
	}
	cmd.Flags().StringVar(&observed, "observed", "", "the sha256 now on disk, as source list shows it")
	cmd.Flags().StringVarP(&rationale, "message", "m", "", "why the change needs no further action, or what was done")
	cmd.Flags().StringVar(&file, "file", "", "read a source-ack.v1 document from this file instead")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print errors as error.v1")
	return cmd
}

// ackInput builds the source-ack.v1 input from the flags or the file, and
// validates it against the schema.
func ackInput(cmd *cobra.Command, args []string, observed, rationale, file string) (sourceAckDoc, error) {
	var raw []byte
	if file != "" {
		if len(args) > 0 || cmd.Flags().Changed("observed") || cmd.Flags().Changed("message") {
			return sourceAckDoc{}, apperr.New(apperr.InvalidInput, "--file replaces the source, --observed and -m; give one or the other")
		}
		b, err := readSmallFile(file)
		if err != nil {
			return sourceAckDoc{}, err
		}
		raw = b
	} else {
		if len(args) != 1 {
			return sourceAckDoc{}, apperr.New(apperr.InvalidInput, "name the source to acknowledge (a path or an ID), or pass --file")
		}
		b, err := json.Marshal(sourceAckDoc{Schema: "specmetry.source-ack/v1", Source: args[0], ObservedSHA256: observed, Rationale: rationale})
		if err != nil {
			return sourceAckDoc{}, apperr.Wrap(apperr.Internal, err, "cannot encode the input")
		}
		raw = b
	}
	val, err := validator()
	if err != nil {
		return sourceAckDoc{}, apperr.Wrap(apperr.Internal, err, "cannot load the exchange schemas")
	}
	if err := val.ValidateJSON("source-ack.v1", raw); err != nil {
		return sourceAckDoc{}, err
	}
	var in sourceAckDoc
	if err := json.Unmarshal(raw, &in); err != nil {
		return sourceAckDoc{}, apperr.Wrap(apperr.InvalidInput, err, "the input is not valid")
	}
	return in, nil
}

// readSmallFile reads an input document of at most 1 MiB.
func readSmallFile(path string) ([]byte, error) { return readInput(path, 1<<20) }

// readInput reads an input document of at most limit bytes (a whole number
// of MiB). Errors do not name the file: its path has not been scanned for
// secrets.
func readInput(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, apperr.New(apperr.InvalidInput, "cannot read the --file input ("+osProblem(err)+")")
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, apperr.New(apperr.InvalidInput, "cannot read the --file input ("+osProblem(err)+")")
	}
	if int64(len(b)) > limit {
		return nil, apperr.Newf(apperr.InvalidInput, "the --file input is larger than %d MiB", limit>>20)
	}
	return b, nil
}

// osProblem describes a file system error without its path.
func osProblem(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return "read error"
}

// ackPrecondition checks that observed is the hash on disk (data-model
// §7.1): otherwise stale_precondition (exit code 5) with the current hash.
func ackPrecondition(s source.Source, obs source.Observation, observed string) error {
	if obs.CurrentSHA256 == observed {
		return nil
	}
	current := any(nil)
	what := "cannot be read now"
	if obs.CurrentSHA256 != "" {
		current = obs.CurrentSHA256
		what = "has sha256 " + obs.CurrentSHA256 + " on disk now"
	}
	e := apperr.Newf(apperr.StalePrecondition,
		"%s %s, not the observed %s; look at the change again and acknowledge the hash on disk", s.Path, what, observed)
	e.Current = map[string]any{"source": s.ID, "current_sha256": current}
	return e
}

func newSourceRemoveCmd() *cobra.Command {
	var erase, asJSON bool
	cmd := &cobra.Command{
		Use:   "remove <path|id>",
		Short: "Remove a source (a person only)",
		Long: `Remove a source. A person only, at a terminal, after confirming. Its
references stay and show the source as removed; registering the path again
creates a new source. --erase-excerpts also erases the excerpt text stored
for its references; a hash_only project stores none.`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			d := channel.FromContext(ctx)
			if err := d.RequireHuman("source remove"); err != nil {
				return err
			}
			w, err := prepare(ctx)
			if err != nil {
				return err
			}
			defer w.close()
			if err := secrets.Reject(w.scanner.Fields(secrets.Field{Pointer: "source", Value: args[0]})); err != nil {
				return err
			}
			if err := w.open(ctx); err != nil {
				return err
			}
			var s source.Source
			var storage string
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				var found bool
				s, found, err = source.FindActive(ctx, tx, p.ID, args[0])
				if err == nil && !found {
					err = apperr.Newf(apperr.NotFound, "%s is not an active source (see source list)", args[0])
				}
				storage = p.ExcerptStorage
				return err
			}); err != nil {
				return err
			}
			summary := []string{
				"source: " + s.Path + " (" + s.ID + ", " + string(s.Kind) + ")",
				"its references are kept and show the source as removed",
				"the source set hash changes, so a confirmed delivery scope must be confirmed again",
			}
			if erase {
				if storage != "hash_only" {
					// Erasing stored excerpt text, and compacting the files
					// afterwards, arrive with WP2.11.
					return apperr.New(apperr.Environment, "erasing stored excerpts is not available yet; nothing was changed")
				}
				summary = append(summary, "--erase-excerpts: this project stores excerpts as hash_only, so no excerpt text is stored and there is nothing to erase")
			}
			op := channel.Operation{Name: "source remove", Summary: summary, Confirm: confirmText(s)}
			if err := channel.Guard(d, os.Stdin, cmd.OutOrStdout(), op); err != nil {
				return err
			}
			var seq int64
			var setHash string
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				current, found, err := source.FindActive(ctx, tx, p.ID, s.ID)
				if err == nil && !found {
					err = apperr.Newf(apperr.NotFound, "source %s was removed meanwhile", s.ID)
				}
				if err != nil {
					return err
				}
				if seq, err = source.Remove(ctx, tx, p.ID, current); err != nil {
					return err
				}
				setHash, err = source.ProjectSetHash(ctx, tx, p.ID)
				return err
			}); err != nil {
				return err
			}
			return printf(cmd.OutOrStdout(), "Removed %s (seq %d)\nsource set hash %s\n", s.Path, seq, setHash)
		}),
	}
	cmd.Flags().BoolVar(&erase, "erase-excerpts", false, "also erase the excerpt text stored for its references")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print errors as error.v1")
	return cmd
}

// printf writes human-readable output.
func printf(w io.Writer, format string, args ...any) error {
	if _, err := fmt.Fprintf(w, format, args...); err != nil {
		return apperr.Wrap(apperr.Environment, err, "cannot write the output")
	}
	return nil
}
