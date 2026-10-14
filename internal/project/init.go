// Package project registers projects and their worktrees and runs init
// (implementation-plan WP1.1; architecture §6.1, §6.1.1, §10.6;
// data-model §4.1, §4.2).
package project

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/domain/derive"
	"github.com/XKHoshizora/specmetry/internal/gitx"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/ulid"
)

// InitOptions are the inputs of init.
type InitOptions struct {
	// Dir is the directory init runs in.
	Dir string
	// Name is the project name for a new project (--name, when NameGiven);
	// otherwise the main worktree's directory name. init never renames a
	// project.
	Name      string
	NameGiven bool
	// Stage overrides the detected stage (--stage); empty keeps it.
	Stage         Stage
	NoExclude     bool
	SkipSources   bool
	SkipInventory bool
}

// Warnings of init.v1.
const (
	WarnNotGit             = "not_a_git_repository"
	WarnDetachedNoTarget   = "detached_head_no_target"
	WarnExcludeNotWritable = "exclude_not_writable"
)

// Plan is what init found out before writing anything: where the project
// is, its files' names and sizes, and the main worktree's branch.
type Plan struct {
	opts InitOptions
	git  bool
	// root is the main worktree; current the worktree init runs in. Outside
	// git both are the project directory.
	root, current         string
	rootGitDir, curGitDir string
	commonDir             string
	name                  string
	marker                Marker
	markerFound           bool
	detected              Stage
	candidates, hints     []string
	// branch is the main worktree's branch, "" when detached or outside git.
	branch      string
	excludePath string
}

// Target is the active delivery target.
type Target struct {
	ID, Ref string
	// Snapshot is the commit the ref points at, "" if it has none (yet) or
	// it cannot be resolved.
	Snapshot string
}

// Result is the outcome of init (init.v1).
type Result struct {
	ProjectID, Name       string
	Stage, StageDetected  Stage
	Created               bool
	Progress              derive.InitProgress
	BaselineVersion       int
	Target                *Target
	Candidates, TestHints []string
	Warnings              []string
}

var commitID = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Prepare finds the project init is about and checks every input. It
// writes nothing: whatever it refuses leaves no trace.
func Prepare(ctx context.Context, opts InitOptions, scanner *secrets.Scanner) (*Plan, error) {
	p := &Plan{opts: opts}
	if opts.Stage != "" {
		if _, ok := ParseStage(string(opts.Stage)); !ok {
			return nil, apperr.New(apperr.InvalidInput, "invalid stage").WithDetails(apperr.Detail{
				Pointer: "--stage", Problem: "must be docs_only, in_development or incomplete_docs"})
		}
	}
	if opts.NameGiven {
		if err := validateName(opts.Name, "--name"); err != nil {
			return nil, err
		}
	}
	dir := gitx.Canonical(opts.Dir)
	loc, err := gitx.Locate(ctx, dir)
	switch {
	case errors.Is(err, gitx.ErrNotRepository):
		// Outside git the project directory is the nearest one with a
		// marker, so init finds the project from its sub-directories.
		p.root = dir
		if root, found := FindMarker(dir); found {
			p.root = root
		}
		p.current = p.root
	case err != nil:
		return nil, err
	default:
		p.git = true
		p.root, p.current = loc.Main, loc.Worktree
		p.curGitDir, p.commonDir = loc.GitDir, loc.CommonDir
		p.rootGitDir = loc.GitDir
		if p.current != p.root {
			main, err := gitx.Locate(ctx, p.root)
			if err != nil {
				return nil, err
			}
			p.rootGitDir = main.GitDir
		}
	}

	p.name = opts.Name
	pointer := "--name"
	if !opts.NameGiven {
		p.name, pointer = filepath.Base(p.root), "name (the directory name)"
		// The name is not quoted: it is checked for secrets only below.
		if err := validateName(p.name, pointer); err != nil {
			return nil, apperr.Newf(apperr.InvalidInput,
				"the main worktree's directory name cannot be the project name (%s); pass --name", apperr.From(err).Details[0].Problem)
		}
	}
	if err := secrets.Reject(scanner.Fields(secrets.Field{Pointer: pointer, Value: p.name})); err != nil {
		return nil, err
	}
	if err := rejectSecretPaths(scanner, p); err != nil {
		return nil, err
	}

	if p.marker, p.markerFound, err = ReadMarker(p.root); err != nil {
		return nil, err
	}
	files, err := ListFiles(ctx, p.root, p.git)
	if err != nil {
		return nil, err
	}
	p.detected = DetectStage(files)
	p.candidates = Candidates(files)
	p.hints = TestCommandHints(files)

	if p.git {
		if p.branch, err = gitx.HeadBranch(ctx, p.root); err != nil {
			return nil, err
		}
		// codex-review-015 B2: the branch is the ref a new project's default
		// target stores. It is scanned whether or not a target will be
		// created, since that is known only once the database is open.
		if err := rejectSecretRef(scanner, p.branch); err != nil {
			return nil, err
		}
		if !opts.NoExclude {
			// Not resolving it only costs the exclude entry: a warning.
			p.excludePath, _ = gitx.GitPath(ctx, p.root, "info/exclude")
		}
	}
	return p, nil
}

// validateName checks a project name: not blank, one line without control
// characters, at most 200 characters (init.v1).
func validateName(name, pointer string) error {
	problem := ""
	switch {
	case !utf8.ValidString(name):
		problem = "not valid UTF-8"
	case strings.TrimSpace(name) == "":
		problem = "empty"
	case utf8.RuneCountInString(name) > 200:
		problem = "longer than 200 characters"
	case strings.ContainsFunc(name, unicode.IsControl):
		problem = "contains a line break or another control character"
	default:
		return nil
	}
	return apperr.New(apperr.InvalidInput, "invalid project name").
		WithDetails(apperr.Detail{Pointer: pointer, Problem: problem})
}

// rejectSecretPaths scans the paths init stores (architecture §10.8). The
// error names which path and rule, never the text.
func rejectSecretPaths(scanner *secrets.Scanner, p *Plan) error {
	fields := []secrets.Field{
		{Pointer: "main worktree path", Value: p.root},
		{Pointer: "worktree path", Value: p.current},
	}
	if p.git {
		fields = append(fields,
			secrets.Field{Pointer: "git common directory", Value: p.commonDir},
			secrets.Field{Pointer: "main worktree git directory", Value: p.rootGitDir},
			secrets.Field{Pointer: "worktree git directory", Value: p.curGitDir})
	}
	findings := scanner.Fields(fields...)
	if len(findings) == 0 {
		return nil
	}
	details := make([]apperr.Detail, len(findings))
	for i, f := range findings {
		details[i] = apperr.Detail{Pointer: f.Pointer, Problem: "secret_detected: " + f.RuleID}
	}
	return apperr.New(apperr.InvalidInput,
		"a path Specmetry would store looks like it contains a secret; rename the directory or move the project").
		WithDetails(details...)
}

// rejectSecretRef scans the main worktree's branch (architecture §10.8).
// The error names the field and the rule, never the ref.
func rejectSecretRef(scanner *secrets.Scanner, ref string) error {
	findings := scanner.Fields(secrets.Field{Pointer: "default delivery target ref (the main worktree's branch)", Value: ref})
	if len(findings) == 0 {
		return nil
	}
	return apperr.New(apperr.InvalidInput,
		"the main worktree's branch name looks like it contains a secret, and Specmetry would store it as the default delivery target; rename the branch or check out another one").
		WithDetails(apperr.Detail{Pointer: findings[0].Pointer, Problem: "secret_detected: " + findings[0].RuleID})
}

// RefuseMarkerWithoutDatabase refuses a marker when the data directory has
// no database yet, so it cannot know the project the marker names: before
// the database is created, so the refusal leaves the data directory as it
// was (codex-review-015 N1).
func (p *Plan) RefuseMarkerWithoutDatabase() error {
	if !p.markerFound {
		return nil
	}
	return p.markerConflict("")
}

// settings is projects.settings_json (data-model §4.1). Keys added later
// are kept as they are.
type settings map[string]json.RawMessage

func (s settings) flag(key string) bool {
	var v bool
	_ = json.Unmarshal(s[key], &v)
	return v
}

func (s settings) set(key string, v any) {
	b, _ := json.Marshal(v)
	s[key] = b
}

type project struct {
	id, name, stage, mainWorktree string
	settings                      settings
}

// Apply registers the project and the worktree, or finds them, in one
// write transaction; then writes the marker and the exclude entry. A
// failure in the transaction writes nothing. A failure after it leaves a
// registered project that the next init completes.
func (p *Plan) Apply(ctx context.Context, db *sqlite.DB, now time.Time) (Result, error) {
	res := Result{
		StageDetected: p.detected,
		Candidates:    p.candidates,
		TestHints:     p.hints,
		Warnings:      []string{},
	}
	var target *Target
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		proj, found, err := p.find(ctx, tx)
		if err != nil {
			return err
		}
		if found {
			err = p.update(ctx, tx, &proj, now)
		} else {
			proj, err = p.create(ctx, tx, now)
		}
		if err != nil {
			return err
		}
		res.Created = !found
		res.ProjectID, res.Name, res.Stage = proj.id, proj.name, Stage(proj.stage)
		// Sources, proposals, baselines, scope confirmations and the
		// inventory arrive with later work packages; until then none
		// exists, so only the recorded skips count.
		res.Progress = derive.InitProgressOf(derive.InitFacts{
			SourcesSkipped:   proj.settings.flag("sources_skipped"),
			InventorySkipped: proj.settings.flag("inventory_skipped"),
			DocsOnly:         res.Stage == DocsOnly,
		})
		target, err = activeTarget(ctx, tx, proj.id)
		return err
	})
	if err != nil {
		return Result{}, err
	}

	if !p.markerFound {
		if err := WriteMarker(p.root, Marker{ProjectID: res.ProjectID, Name: res.Name}); err != nil {
			return Result{}, apperr.Newf(apperr.Environment,
				"project %s is registered, but its marker could not be written (%s); fix the problem and run init again",
				res.ProjectID, apperr.From(err).Message)
		}
	}
	if !p.git {
		res.Warnings = append(res.Warnings, WarnNotGit)
	} else if target == nil && p.branch == "" {
		res.Warnings = append(res.Warnings, WarnDetachedNoTarget)
	}
	if p.git && !p.opts.NoExclude {
		if p.excludePath == "" || EnsureExcluded(p.excludePath) != nil {
			res.Warnings = append(res.Warnings, WarnExcludeNotWritable)
		}
	}
	if target != nil {
		// A snapshot that cannot be resolved is unknown, not an error.
		if sha, err := gitx.CommitOf(ctx, p.root, target.Ref); err == nil && commitID.MatchString(sha) {
			target.Snapshot = sha
		}
	}
	res.Target = target
	return res, nil
}

// find looks the project up: a git project by its common directory, one
// outside git by its directory. The marker must agree with the database.
func (p *Plan) find(ctx context.Context, tx *sql.Tx) (project, bool, error) {
	var proj project
	var raw string
	var err error
	if p.git {
		err = tx.QueryRowContext(ctx, `SELECT id, name, stage, main_worktree_id, settings_json FROM projects WHERE git_common_dir = ?`,
			p.commonDir).Scan(&proj.id, &proj.name, &proj.stage, &proj.mainWorktree, &raw)
	} else {
		err = tx.QueryRowContext(ctx, `SELECT p.id, p.name, p.stage, p.main_worktree_id, p.settings_json
FROM projects p JOIN worktrees w ON w.id = p.main_worktree_id
WHERE p.git_common_dir IS NULL AND w.path = ?`, p.root).Scan(&proj.id, &proj.name, &proj.stage, &proj.mainWorktree, &raw)
	}
	found := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return project{}, false, err
	}
	if found {
		if err := json.Unmarshal([]byte(raw), &proj.settings); err != nil {
			return project{}, false, apperr.Wrap(apperr.Internal, err, "the project settings are not valid JSON")
		}
	}
	if p.markerFound && (!found || p.marker.ProjectID != proj.id) {
		return project{}, false, p.markerConflict(proj.id)
	}
	return proj, found, nil
}

// markerConflict reports a marker that names another project than the
// database (TD-25). Nothing is written: the marker may belong to a project
// that was purged, to another SPECMETRY_HOME, or to a copied or moved
// repository, and only a person can tell which.
func (p *Plan) markerConflict(registered string) error {
	where := "which this Specmetry data directory does not have; it may have been purged, or registered with another " +
		"SPECMETRY_HOME, or the directory may be a copy of another project"
	if registered != "" {
		where = "but this directory is registered as project " + registered
	}
	return apperr.Newf(apperr.Environment,
		"the marker %s names project %s, %s. No project was registered and the marker was left as it is. If the directory should be registered anew, remove the marker and run init again (TD-25)",
		MarkerPath(p.root), p.marker.ProjectID, where)
}

func (p *Plan) create(ctx context.Context, tx *sql.Tx, now time.Time) (project, error) {
	stage := p.detected
	if p.opts.Stage != "" {
		stage = p.opts.Stage
	}
	proj := project{id: ulid.New(), name: p.name, stage: string(stage), mainWorktree: ulid.New(), settings: settings{}}
	proj.settings.set("sources_skipped", p.opts.SkipSources)
	proj.settings.set("inventory_skipped", p.opts.SkipInventory)
	proj.settings.set("excerpt_storage", "hash_only") // PD-24
	raw, err := json.Marshal(proj.settings)
	if err != nil {
		return project{}, err
	}
	var common any
	if p.git {
		common = p.commonDir
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO projects (id, name, git_common_dir, main_worktree_id, stage, settings_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`, proj.id, proj.name, common, proj.mainWorktree, proj.stage, string(raw), now.UnixMilli()); err != nil {
		return project{}, err
	}
	if err := insertWorktree(ctx, tx, proj.mainWorktree, proj.id, p.root, p.gitDir(p.rootGitDir), true, now); err != nil {
		return project{}, err
	}
	if p.current != p.root {
		if err := insertWorktree(ctx, tx, ulid.New(), proj.id, p.current, p.gitDir(p.curGitDir), false, now); err != nil {
			return project{}, err
		}
	}
	// FR-INIT-10: the default target is the main worktree's branch. Only a
	// new project gets one; later, targets are a person's to set.
	if p.branch != "" {
		seq, err := sqlite.NextSeq(ctx, tx)
		if err != nil {
			return project{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO delivery_targets (id, project_id, name, ref, created_seq, actor, channel)
VALUES (?, ?, 'default', ?, ?, 'system', 'system')`, ulid.New(), proj.id, p.branch, seq); err != nil {
			return project{}, err
		}
	}
	return proj, nil
}

// update brings an existing project up to date: the stage if --stage
// changes it, the skip flags (only ever set), and the worktrees.
func (p *Plan) update(ctx context.Context, tx *sql.Tx, proj *project, now time.Time) error {
	if p.opts.NameGiven && p.opts.Name != proj.name {
		return apperr.Newf(apperr.InvalidInput,
			"project %s is registered as %q; init does not rename a project", proj.id, proj.name).
			WithDetails(apperr.Detail{Pointer: "--name", Problem: "differs from the registered name"})
	}
	if p.opts.Stage != "" && string(p.opts.Stage) != proj.stage {
		proj.stage = string(p.opts.Stage)
		if _, err := tx.ExecContext(ctx, `UPDATE projects SET stage = ? WHERE id = ?`, proj.stage, proj.id); err != nil {
			return err
		}
	}
	changed := false
	for key, skip := range map[string]bool{"sources_skipped": p.opts.SkipSources, "inventory_skipped": p.opts.SkipInventory} {
		if skip && !proj.settings.flag(key) {
			proj.settings.set(key, true)
			changed = true
		}
	}
	if changed {
		raw, err := json.Marshal(proj.settings)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE projects SET settings_json = ? WHERE id = ?`, string(raw), proj.id); err != nil {
			return err
		}
	}
	if err := p.ensureMain(ctx, tx, proj, now); err != nil {
		return err
	}
	if p.current != p.root {
		return p.ensureWorktree(ctx, tx, proj.id, now)
	}
	return nil
}

// ensureMain registers the main worktree's directory as the main worktree
// if it is not yet (a repository whose main worktree moved).
func (p *Plan) ensureMain(ctx context.Context, tx *sql.Tx, proj *project, now time.Time) error {
	var id string
	var isMain bool
	err := tx.QueryRowContext(ctx, `SELECT id, is_main FROM worktrees WHERE project_id = ? AND path = ?`, proj.id, p.root).Scan(&id, &isMain)
	switch {
	case err == nil && isMain:
		if p.current != p.root {
			return nil // init runs in a linked worktree; the main one was not seen
		}
		_, err = tx.ExecContext(ctx, `UPDATE worktrees SET last_seen_at = ? WHERE id = ?`, now.UnixMilli(), id)
		return err
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE worktrees SET is_main = 0 WHERE project_id = ? AND is_main = 1`, proj.id); err != nil {
		return err
	}
	if id == "" {
		id = ulid.New()
		if err := insertWorktree(ctx, tx, id, proj.id, p.root, p.gitDir(p.rootGitDir), true, now); err != nil {
			return err
		}
	} else if _, err := tx.ExecContext(ctx, `UPDATE worktrees SET is_main = 1, last_seen_at = ? WHERE id = ?`, now.UnixMilli(), id); err != nil {
		return err
	}
	proj.mainWorktree = id
	_, err = tx.ExecContext(ctx, `UPDATE projects SET main_worktree_id = ? WHERE id = ?`, id, proj.id)
	return err
}

// ensureWorktree registers the linked worktree init runs in, the first
// time it is seen (FR-WT-01).
func (p *Plan) ensureWorktree(ctx context.Context, tx *sql.Tx, projectID string, now time.Time) error {
	res, err := tx.ExecContext(ctx, `UPDATE worktrees SET last_seen_at = ? WHERE project_id = ? AND path = ?`, now.UnixMilli(), projectID, p.current)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	return insertWorktree(ctx, tx, ulid.New(), projectID, p.current, p.gitDir(p.curGitDir), false, now)
}

func (p *Plan) gitDir(dir string) any {
	if !p.git {
		return nil
	}
	return dir
}

// EnsureWorktree returns the ID of the worktree w.Current of a registered
// project, registering it the first time any command runs there
// (data-model §4.1); a known one only has last_seen_at updated, without a
// seq. The caller has scanned its path and git directory for secrets.
func EnsureWorktree(ctx context.Context, tx *sql.Tx, projectID string, w Where, now time.Time) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `SELECT id FROM worktrees WHERE project_id = ? AND path = ?`, projectID, w.Current).Scan(&id)
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE worktrees SET last_seen_at = ? WHERE id = ?`, now.UnixMilli(), id)
		return id, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	var gitDir any
	if w.Git {
		gitDir = w.GitDir
	}
	id = ulid.New()
	return id, insertWorktree(ctx, tx, id, projectID, w.Current, gitDir, false, now)
}

func insertWorktree(ctx context.Context, tx *sql.Tx, id, projectID, path string, gitDir any, isMain bool, now time.Time) error {
	seq, err := sqlite.NextSeq(ctx, tx)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO worktrees (id, project_id, path, git_dir, is_main, registered_seq, last_seen_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`, id, projectID, path, gitDir, isMain, seq, now.UnixMilli())
	if err != nil {
		return fmt.Errorf("register worktree %s: %w", path, err)
	}
	return nil
}

func activeTarget(ctx context.Context, tx *sql.Tx, projectID string) (*Target, error) {
	var t Target
	err := tx.QueryRowContext(ctx, `SELECT id, ref FROM delivery_targets WHERE project_id = ? AND archived_seq IS NULL`, projectID).Scan(&t.ID, &t.Ref)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}
