package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/config"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
)

// initDoc is the init.v1 document.
type initDoc struct {
	Schema           string      `json:"schema"`
	Project          initProject `json:"project"`
	Created          bool        `json:"created"`
	InitProgress     string      `json:"init_progress"`
	BaselineVersion  int         `json:"baseline_version"`
	Target           *initTarget `json:"target"`
	CandidateSources []string    `json:"candidate_sources"`
	TestHints        []string    `json:"test_hints"`
	Warnings         []string    `json:"warnings"`
}

type initProject struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Stage         string `json:"stage"`
	StageDetected string `json:"stage_detected"`
}

type initTarget struct {
	ID       string  `json:"id"`
	Ref      string  `json:"ref"`
	Snapshot *string `json:"snapshot"`
}

func newInitCmd() *cobra.Command {
	var opts project.InitOptions
	var stage string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Register the project, or report where its onboarding stands",
		Long: `Register the project of the current directory, or report where its onboarding
stands if it is registered already (FR-INIT-01). Safe to run again.

init detects the starting stage and lists candidate sources from file names
and sizes only; it never reads file content and never registers a source.
It writes .specmetry/project.json in the main worktree and, unless
--no-exclude is given, adds /.specmetry/ to the repository's exclude file.
A new git project gets a default delivery target: the main worktree's
branch.`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("stage") {
				st, ok := project.ParseStage(stage)
				if !ok {
					return apperr.New(apperr.InvalidInput, "invalid stage").WithDetails(apperr.Detail{
						Pointer: "--stage", Problem: "must be docs_only, in_development or incomplete_docs"})
				}
				opts.Stage = st
			}
			opts.NameGiven = cmd.Flags().Changed("name")
			var err error
			if opts.Dir, err = os.Getwd(); err != nil {
				return apperr.Wrap(apperr.Environment, err, "cannot determine the current directory")
			}
			res, err := runInit(cmd, opts)
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), "init.v1", toInitDoc(res))
			}
			return printInit(cmd.OutOrStdout(), res)
		}),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the init.v1 document")
	cmd.Flags().StringVar(&opts.Name, "name", "", "name of a new project (default: the main worktree's directory name)")
	cmd.Flags().StringVar(&stage, "stage", "", "starting stage: docs_only, in_development or incomplete_docs (default: detected)")
	cmd.Flags().BoolVar(&opts.NoExclude, "no-exclude", false, "do not add /.specmetry/ to the repository's exclude file")
	cmd.Flags().BoolVar(&opts.SkipSources, "skip-sources", false, "record that the project has no source documents")
	cmd.Flags().BoolVar(&opts.SkipInventory, "skip-inventory", false, "record that the existing code will not be inventoried")
	return cmd
}

// runInit checks everything before the data directory is touched, then
// registers the project.
func runInit(cmd *cobra.Command, opts project.InitOptions) (project.Result, error) {
	ctx := cmd.Context()
	dir, err := datadir.Resolve()
	if err != nil {
		return project.Result{}, err
	}
	cfg, _, err := config.Load(dir.Join(datadir.ConfigFile))
	if err != nil {
		return project.Result{}, err
	}
	scanner, err := secrets.New(cfg.Secrets.ExtraRules)
	if err != nil {
		return project.Result{}, apperr.Wrap(apperr.Environment, err, "invalid secret rules")
	}
	plan, err := project.Prepare(ctx, opts, scanner)
	if err != nil {
		return project.Result{}, err
	}
	db, _, err := sqlite.Open(ctx, dir, sqlite.Options{})
	if err != nil {
		return project.Result{}, err
	}
	defer func() { _ = db.Close() }()
	return plan.Apply(ctx, db, time.Now())
}

func toInitDoc(res project.Result) initDoc {
	doc := initDoc{
		Schema: "specmetry.init/v1",
		Project: initProject{
			ID: res.ProjectID, Name: res.Name, Stage: string(res.Stage), StageDetected: string(res.StageDetected),
		},
		Created:          res.Created,
		InitProgress:     string(res.Progress),
		BaselineVersion:  res.BaselineVersion,
		CandidateSources: nonNil(res.Candidates),
		TestHints:        nonNil(res.TestHints),
		Warnings:         nonNil(res.Warnings),
	}
	if res.Target != nil {
		doc.Target = &initTarget{ID: res.Target.ID, Ref: res.Target.Ref}
		if res.Target.Snapshot != "" {
			snapshot := res.Target.Snapshot
			doc.Target.Snapshot = &snapshot
		}
	}
	return doc
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

var warningText = map[string]string{
	project.WarnNotGit:             "not a git repository: there is no delivery target, so delivery cannot be assessed",
	project.WarnDetachedNoTarget:   "the main worktree has a detached HEAD, so no default delivery target was created; a person can set one",
	project.WarnExcludeNotWritable: "could not add /.specmetry/ to the repository's exclude file; git may show the marker as untracked",
}

func printInit(w io.Writer, res project.Result) error {
	var b []byte
	line := func(format string, args ...any) { b = fmt.Appendf(b, format+"\n", args...) }
	if res.Created {
		line("Initialized project %s (%s)", res.Name, res.ProjectID)
	} else {
		line("Project %s (%s) is already initialized", res.Name, res.ProjectID)
	}
	line("  stage: %s (detected: %s)", res.Stage, res.StageDetected)
	line("  progress: %s", res.Progress)
	switch {
	case res.Target == nil:
		line("  delivery target: none")
	case res.Target.Snapshot == "":
		line("  delivery target: %s (no commit yet)", res.Target.Ref)
	default:
		line("  delivery target: %s at %s", res.Target.Ref, res.Target.Snapshot[:12])
	}
	if len(res.Candidates) == 0 {
		line("  candidate sources: none found")
	} else {
		line("  candidate sources (not registered):")
		for _, c := range res.Candidates {
			line("    %s", c)
		}
	}
	for _, h := range res.TestHints {
		line("  test command: %s", h)
	}
	for _, warn := range res.Warnings {
		line("warning: %s", warningText[warn])
	}
	if _, err := w.Write(b); err != nil {
		return apperr.Wrap(apperr.Environment, err, "cannot write the output")
	}
	return nil
}
