package checkrun

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/baseline"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/scope"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
)

// Item is one criterion of a batch with what the read transaction fixed
// for it (spec §3, D13): its revision, its requirement revision, its
// configuration hash and a copy of its check, which every later step and
// its run use, whatever the baseline says by then.
type Item struct {
	Criterion           string
	CriterionID         string
	CriterionRevision   int
	RequirementRevision int
	CheckConfigHash     string
	Check               baseline.Check
}

// fixed is what the one read transaction of steps 3 to 5 reads (spec §4.1,
// D13; plan §11.5): the project, whether W is registered and every
// registered worktree's path (step 7's exclusion), every criterion with
// its stored configuration hash, and the active delivery target (step 8
// resolves it).
type fixed struct {
	project    project.Registered
	worktreeID string
	worktrees  []string
	criteria   []baseline.Record
	hashes     map[string]string
	target     *scope.Target
}

// readFixed is the read transaction: a DB.WriteTx that writes nothing and
// allocates no seq, as baseline show reads. It ends before anything else
// is done; from then on only the copies it made are used.
func readFixed(ctx context.Context, db *sqlite.DB, where project.Where) (fixed, error) {
	var f fixed
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		var err error
		if f.project, err = project.Lookup(ctx, tx, where); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT id, path FROM worktrees WHERE project_id = ? ORDER BY path`, f.project.ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id, path string
			if err := rows.Scan(&id, &path); err != nil {
				_ = rows.Close()
				return err
			}
			f.worktrees = append(f.worktrees, path)
			if path == where.Current {
				f.worktreeID = id
			}
		}
		if err := errors.Join(rows.Err(), rows.Close()); err != nil {
			return err
		}
		if f.criteria, err = baseline.LoadAll(ctx, tx, f.project.ID, baseline.Criterion); err != nil {
			return err
		}
		f.hashes = map[string]string{}
		rows, err = tx.QueryContext(ctx, `SELECT id, check_config_hash FROM acceptance_criteria
WHERE project_id = ? AND check_config_hash IS NOT NULL`, f.project.ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id, hash string
			if err := rows.Scan(&id, &hash); err != nil {
				_ = rows.Close()
				return err
			}
			f.hashes[id] = hash
		}
		if err := errors.Join(rows.Err(), rows.Close()); err != nil {
			return err
		}
		f.target, err = scope.ActiveTarget(ctx, tx, f.project.ID)
		return err
	})
	return f, environment(err, "cannot read the baseline; no check was run")
}

// choose is step 5 (spec §3; D15, E1). Given criteria must each be an
// active automated criterion with a check, and given once; every problem
// is reported at its argument's position, none quoting it, and nothing is
// run. --all is every such criterion, the delivery scope aside. Either
// way they run in key order, whatever the order given.
func (f fixed) choose(opts Options) ([]Item, error) {
	runnable := func(r baseline.Record) bool {
		return r.Active() && r.Verification == "automated" && r.Check != nil
	}
	want := map[string]bool{}
	if !opts.All {
		byKey := map[string]baseline.Record{}
		for _, r := range f.criteria {
			byKey[r.Key] = r
		}
		var problems []apperr.Detail
		first := map[string]int{}
		for i, arg := range opts.Criteria {
			problem := ""
			r, found := byKey[arg]
			if j, again := first[arg]; again {
				problem = "repeats argument " + strconv.Itoa(j+1)
			} else {
				first[arg] = i
				switch {
				case !found:
					problem = "names no acceptance criterion of the baseline"
				case !r.Active():
					problem = "names a retired acceptance criterion"
				case r.Verification != "automated":
					problem = "names a manual acceptance criterion: only automated ones have a check to run"
				case r.Check == nil:
					problem = "names an acceptance criterion without a check"
				default:
					want[arg] = true
				}
			}
			if problem != "" {
				problems = append(problems, apperr.Detail{Pointer: "argument " + strconv.Itoa(i+1), Problem: problem})
			}
		}
		if len(problems) > 0 {
			return nil, apperr.New(apperr.InvalidInput, "the acceptance criteria given cannot all be checked; no check was run").
				WithDetails(problems...)
		}
	}
	var items []Item
	for _, r := range f.criteria {
		if !runnable(r) || !opts.All && !want[r.Key] {
			continue
		}
		hash, ok := f.hashes[r.ID]
		if !ok {
			return nil, apperr.Newf(apperr.Internal, "the check of %s has no configuration hash; no check was run", r.Key)
		}
		c := *r.Check
		c.Inputs = slices.Clone(c.Inputs)
		if c.Shell != nil {
			shell := *c.Shell
			c.Shell = &shell
		}
		items = append(items, Item{Criterion: r.Key, CriterionID: r.ID, CriterionRevision: r.Revision,
			RequirementRevision: r.RequirementRevision, CheckConfigHash: hash, Check: c})
	}
	return items, nil
}

// staticRules is step 6 (D3, D4): a fixed check whose command breaks the
// rule is refused before anything runs, every one at its criterion,
// without quoting the command. The saved check is left as it is; a person
// revises it with a material change.
func staticRules(items []Item) error {
	var problems []apperr.Detail
	for _, it := range items {
		if p := it.Check.LineBreakProblem(); p != "" {
			problems = append(problems, apperr.Detail{Pointer: it.Criterion, Problem: "the saved check's command " + p})
		}
	}
	if len(problems) > 0 {
		return apperr.New(apperr.InvalidInput,
			"a check cannot run as it is saved: a person revises it with a material change; no check was run").WithDetails(problems...)
	}
	return nil
}

// environment is err as it is when it is Specmetry's own error, and
// otherwise an environment error with message.
func environment(err error, message string) error {
	var e *apperr.Error
	if err == nil || errors.As(err, &e) {
		return err
	}
	return apperr.Wrap(apperr.Environment, err, message)
}
