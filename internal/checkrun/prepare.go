package checkrun

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"slices"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/config"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
)

// Batch is an admitted check run once every step before its first check
// has passed (spec §4.1): the project and W, W's worktree if it is
// registered, and the criteria to run in their order with what was fixed
// for them. Nothing has run and nothing is recorded; the caller closes it.
type Batch struct {
	ProjectID string
	Where     project.Where
	// WorktreeID is W's registered worktree, "" while W is not registered.
	WorktreeID string
	Items      []Item

	host   *host
	call   int
	db     *sqlite.DB
	closed bool
}

// Test hooks: testHookOpen runs where the database is opened, the one
// place Prepare opens it; testHookAfterRead runs once the read transaction
// has ended.
var (
	testHookOpen      func()
	testHookAfterRead func()
)

// Prepare runs, in order, the steps of spec §4.1 that come before the
// first check, with this process's host. On any failure it returns the
// error of the first step that failed, every problem of that step listed;
// nothing has run, nothing is recorded, and what it opened is closed.
func Prepare(ctx context.Context, opts Options, out Output) (*Batch, error) {
	return prepare(ctx, &process, opts, out)
}

func prepare(ctx context.Context, h *host, opts Options, _ Output) (_ *Batch, err error) {
	// Step 1, then step 2: refusals before any database, git or process.
	if err := opts.check(); err != nil {
		return nil, err
	}
	if opts.Isolation == "required" {
		return nil, isolationUnavailable()
	}
	call, err := h.admit()
	if err != nil {
		return nil, err
	}
	b := &Batch{host: h, call: call}
	defer func() {
		if err != nil {
			_ = b.Close()
		}
	}()

	// Step 3: the project and W, then the database.
	dir, err := datadir.Resolve()
	if err != nil {
		return nil, err
	}
	cfg, _, err := config.Load(dir.Join(datadir.ConfigFile))
	if err != nil {
		return nil, err
	}
	scanner, err := secrets.New(cfg.Secrets.ExtraRules)
	if err != nil {
		return nil, apperr.Wrap(apperr.Environment, err, "invalid secret rules")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, apperr.Wrap(apperr.Environment, err, "cannot determine the current directory")
	}
	if b.Where, err = project.Locate(ctx, cwd); err != nil {
		return nil, err
	}
	if _, err := os.Lstat(dir.Join(datadir.DatabaseFile)); errors.Is(err, fs.ErrNotExist) {
		return nil, project.ErrNotInitialized()
	}
	if testHookOpen != nil {
		testHookOpen()
	}
	if b.db, _, err = sqlite.Open(ctx, dir, sqlite.Options{}); err != nil {
		return nil, err
	}
	// Steps 3 to 5 read in one transaction; the steps are then judged in
	// their order, on what it fixed.
	f, err := readFixed(ctx, b.db, b.Where)
	if err != nil {
		return nil, err
	}
	if testHookAfterRead != nil {
		testHookAfterRead()
	}
	b.ProjectID, b.WorktreeID = f.project.ID, f.worktreeID

	// Step 4: a worktree to register has its paths scanned first.
	if b.WorktreeID == "" {
		if err := secrets.Reject(scanner.Fields(
			secrets.Field{Pointer: "worktree path", Value: b.Where.Current},
			secrets.Field{Pointer: "worktree git directory", Value: b.Where.GitDir})); err != nil {
			return nil, err
		}
	}
	// Step 5, then step 6.
	if b.Items, err = f.choose(opts); err != nil {
		return nil, err
	}
	if err := staticRules(b.Items); err != nil {
		return nil, err
	}
	// Step 7: the interpreters, outside every registered worktree and W.
	if len(b.Items) > 0 {
		roots := f.worktrees
		if b.WorktreeID == "" {
			roots = append(slices.Clone(roots), b.Where.Current)
		}
		if err := interpreters(ctx, b.Items, newFinder(roots, os.Getenv("PATH"), gitExecPath(b.Where.Current))); err != nil {
			return nil, err
		}
	}
	return b, nil
}

// Recheck is spec §19.1 rule 3, before each criterion starts: a host
// disabled since the call was admitted stops it.
func (b *Batch) Recheck() error { return b.host.recheck() }

// Close closes what the batch opened and ends its call on the host. It
// may be called again.
func (b *Batch) Close() error {
	if b.closed {
		return nil
	}
	b.closed = true
	var err error
	if b.db != nil {
		err = b.db.Close()
	}
	b.host.release()
	return err
}
