package gitscan

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/gitx"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/ulid"
)

// Options configures Scan.
type Options struct {
	// Secrets scans the names the scan would store in its cursors (D7).
	Secrets *secrets.Scanner
	// Scratch holds the private directories of the scan's git reads
	// (gitx.NewScratch on the data directory).
	Scratch *gitx.Scratch
	// Now is the clock of scanned_at; nil means time.Now.
	Now func() time.Time
}

// Result is what a scan did, in counts only.
type Result struct {
	// NotScanned: the project has no requirement or criterion yet, so
	// nothing was read or written (D5).
	NotScanned bool `json:"not_scanned,omitempty"`
	// Concurrent: the cursors changed between the plan and the write
	// (another scan), so nothing was written; the next scan does it again.
	Concurrent bool `json:"concurrent,omitempty"`
	// Read is how many commits the scan read, Stored how many it stored.
	Read   int `json:"read"`
	Stored int `json:"stored"`
}

// Test hooks: run after the plan, before the first commit is read, and
// before the write transaction.
var testHookBeforeRead, testHookBeforeWrite func()

// candidate is a commit read that mentions at least one key: its SHA, its
// committer time and the mentions, never its message.
type candidate struct {
	sha      string
	time     int64
	mentions []Mention
}

// Scan scans the commits of the repository of worktree for project
// (WP2.3 spec §5.5):
//
//  1. the plan, read under the store's lock: the project's cursors, and
//     whether it has any requirement or criterion (none: NotScanned, with
//     no git command run);
//  2. the reads, outside any transaction: the shallow check (P3), the
//     tips, the secret check of the names the cursors would store (D7),
//     the classification of the old tips and the detached HEADs, each
//     ref's range, and the strict logs; any failure fails the scan and
//     whatever was read is dropped;
//  3. one write transaction (BEGIN IMMEDIATE): if any cursor differs from
//     the plan, nothing is written (Concurrent); otherwise the commits not
//     yet stored whose mentions name keys of the project (retired ones
//     included) are stored with their links and one shared seq (none when
//     there are none), and every ref's cursor is written.
//
// Nothing is retried. A failure leaves no commit, link, seq or cursor.
func Scan(ctx context.Context, db *sqlite.DB, project, worktree string, opts Options) (Result, error) {
	if opts.Secrets == nil || opts.Scratch == nil {
		return Result{}, apperr.New(apperr.Internal, "Scan needs a secret scanner and a Scratch")
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	cursors, commonDir, keys, err := readPlan(ctx, db, project)
	if err != nil {
		return Result{}, err
	}
	if keys == 0 {
		return Result{NotScanned: true}, nil
	}
	loc, err := gitx.Locate(ctx, worktree)
	if err != nil {
		return Result{}, err
	}
	if loc.CommonDir != gitx.Canonical(commonDir) {
		return Result{}, apperr.New(apperr.Internal, "the worktree is not in the project's repository")
	}
	if err := gitx.CheckShallow(ctx, worktree); err != nil {
		return Result{}, err
	}
	tips, err := gitx.Tips(ctx, opts.Scratch, worktree)
	if err != nil {
		return Result{}, err
	}
	refs, err := refsOf(opts.Secrets, tips)
	if err != nil {
		return Result{}, err
	}
	ids := make([]string, 0, len(cursors)+len(refs))
	for _, tip := range cursors {
		ids = append(ids, tip)
	}
	for _, r := range refs {
		if r.Kind == KindDetached {
			ids = append(ids, r.Tip)
		}
	}
	presence, err := gitx.Classify(ctx, opts.Scratch, worktree, ids)
	if err != nil {
		return Result{}, err
	}
	for _, r := range refs {
		if r.Kind == KindDetached && presence[r.Tip] != gitx.Present {
			return Result{}, apperr.New(apperr.Environment,
				"the commit a worktree's detached HEAD points to cannot be read from the repository; nothing was scanned; run git log in that worktree to see why")
		}
	}
	reads, err := planReads(cursors, refs, presence, func(a, c string) (bool, error) {
		return gitx.IsAncestor(ctx, opts.Scratch, worktree, a, c)
	})
	if err != nil {
		return Result{}, err
	}
	if testHookBeforeRead != nil {
		testHookBeforeRead()
	}
	seen := map[string]bool{}
	var found []candidate
	for _, rd := range reads {
		err := gitx.Log(ctx, opts.Scratch, worktree, rd.Ref.Tip, rd.Exclude, func(c gitx.Commit) error {
			if seen[c.SHA] {
				return nil
			}
			seen[c.SHA] = true
			if m := Mentions(c.Trailers, c.Message); len(m) > 0 {
				found = append(found, candidate{sha: c.SHA, time: c.Time, mentions: m})
			}
			return nil
		})
		if err != nil {
			return Result{}, err // what was read is dropped (G3)
		}
	}
	if testHookBeforeWrite != nil {
		testHookBeforeWrite()
	}
	result := Result{Read: len(seen)}
	err = db.WriteTx(ctx, func(tx *sql.Tx) error {
		current, err := readCursors(ctx, tx, project)
		if err != nil {
			return err
		}
		if !sameCursors(current, cursors) {
			result.Concurrent = true
			return nil
		}
		stored, err := store(ctx, tx, project, found)
		if err != nil {
			return err
		}
		result.Stored = stored.n
		return writeCursors(ctx, tx, project, refs, stored.seq, now().UnixMilli())
	})
	if err != nil {
		return Result{}, err
	}
	if result.Concurrent {
		return Result{Concurrent: true, Read: result.Read}, nil
	}
	return result, nil
}

// readPlan reads the project's cursors, its git common directory and how
// many requirements and criteria it has.
func readPlan(ctx context.Context, db *sqlite.DB, project string) (map[CursorKey]string, string, int, error) {
	var cursors map[CursorKey]string
	var common sql.NullString
	var keys int
	err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `SELECT git_common_dir,
    (SELECT count(*) FROM requirements WHERE project_id = p.id) + (SELECT count(*) FROM acceptance_criteria WHERE project_id = p.id)
FROM projects p WHERE id = ?`, project).Scan(&common, &keys)
		if errors.Is(err, sql.ErrNoRows) {
			return apperr.New(apperr.Internal, "Scan was given a project that is not registered")
		}
		if err != nil {
			return err
		}
		cursors, err = readCursors(ctx, tx, project)
		return err
	})
	if err != nil {
		return nil, "", 0, err
	}
	if !common.Valid {
		return nil, "", 0, apperr.New(apperr.Internal, "Scan was given a project without a git repository")
	}
	return cursors, common.String, keys, nil
}

func readCursors(ctx context.Context, tx *sql.Tx, project string) (map[CursorKey]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT kind, name, last_tip FROM git_scan_cursors WHERE project_id = ?`, project)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[CursorKey]string{}
	for rows.Next() {
		var k CursorKey
		var tip string
		if err := rows.Scan(&k.Kind, &k.Name, &tip); err != nil {
			return nil, err
		}
		out[k] = tip
	}
	return out, rows.Err()
}

// sameCursors reports whether the cursors are the plan's: the same keys,
// each at the same last tip.
func sameCursors(a, b map[CursorKey]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// refsOf turns the tips into the refs to scan, after the secret check of
// the names their cursors would store (D7): any hit refuses the whole
// scan, naming the kind of name and the rule, never the name.
func refsOf(scanner *secrets.Scanner, tips []gitx.Tip) ([]Ref, error) {
	var refs []Ref
	var fields []secrets.Field
	for _, t := range tips {
		r := Ref{Kind: KindBranch, Name: t.Ref, Tip: t.Commit}
		pointer := "branch name"
		if t.Ref == "" {
			r = Ref{Kind: KindDetached, Name: t.WorktreePath, Tip: t.Commit}
			pointer = "worktree path (detached HEAD)"
		}
		refs = append(refs, r)
		fields = append(fields, secrets.Field{Pointer: pointer, Value: r.Name})
	}
	findings := scanner.Fields(fields...)
	if len(findings) == 0 {
		return refs, nil
	}
	details := make([]apperr.Detail, len(findings))
	for i, f := range findings {
		details[i] = apperr.Detail{Pointer: f.Pointer, Problem: "secret_detected: " + f.RuleID}
	}
	return nil, apperr.New(apperr.InvalidInput,
		"a branch name or a detached worktree's path that the commit scan would store looks like it contains a secret, so nothing was scanned; rename the branch, or move or remove the worktree").
		WithDetails(details...)
}

// storedCommits is what store wrote: how many commits, and the seq they
// share (0 for none).
type storedCommits struct {
	n   int
	seq int64
}

// store writes the candidates not yet stored that name at least one key
// of the project, with their links, under one new seq. A stored commit is
// skipped: its links were fixed when it was stored (WP2.3 spec §5.4), and
// this is the only place links are written.
func store(ctx context.Context, tx *sql.Tx, project string, found []candidate) (storedCommits, error) {
	targets := map[string]map[string]string{} // by type, then key: ID
	for t, table := range map[string]string{TargetRequirement: "requirements", TargetCriterion: "acceptance_criteria"} {
		targets[t] = map[string]string{}
		rows, err := tx.QueryContext(ctx, `SELECT key, id FROM `+table+` WHERE project_id = ?`, project) //nolint:gosec // a fixed table name
		if err != nil {
			return storedCommits{}, err
		}
		for rows.Next() {
			var key, id string
			if err := rows.Scan(&key, &id); err != nil {
				_ = rows.Close()
				return storedCommits{}, err
			}
			targets[t][key] = id
		}
		if err := rows.Close(); err != nil {
			return storedCommits{}, err
		}
	}
	type kept struct {
		c     candidate
		links []Mention
	}
	var keep []kept
	slices.SortFunc(found, func(a, b candidate) int { return strings.Compare(a.sha, b.sha) })
	for _, c := range found {
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM commits WHERE project_id = ? AND sha = ?`, project, c.sha).Scan(&n); err != nil {
			return storedCommits{}, err
		}
		if n > 0 {
			continue
		}
		var links []Mention
		for _, m := range c.mentions {
			if _, ok := targets[m.Type][m.Key]; ok {
				links = append(links, m)
			}
		}
		if len(links) > 0 {
			keep = append(keep, kept{c, links})
		}
	}
	if len(keep) == 0 {
		return storedCommits{}, nil
	}
	seq, err := sqlite.NextSeq(ctx, tx)
	if err != nil {
		return storedCommits{}, err
	}
	for _, k := range keep {
		id := ulid.New()
		if _, err := tx.ExecContext(ctx, `INSERT INTO commits (id, seq, project_id, sha, committed_at) VALUES (?, ?, ?, ?, ?)`,
			id, seq, project, k.c.sha, k.c.time*1000); err != nil {
			return storedCommits{}, err
		}
		for _, m := range k.links {
			if _, err := tx.ExecContext(ctx, `INSERT INTO commit_links (commit_id, project_id, target_type, target_id, link_source) VALUES (?, ?, ?, ?, ?)`,
				id, project, m.Type, targets[m.Type][m.Key], m.Source); err != nil {
				return storedCommits{}, err
			}
		}
	}
	return storedCommits{n: len(keep), seq: seq}, nil
}

// writeCursors writes every ref's cursor at its tip; scanned_seq becomes
// seq when the scan stored commits, and is kept (NULL for a new cursor)
// otherwise.
func writeCursors(ctx context.Context, tx *sql.Tx, project string, refs []Ref, seq, at int64) error {
	var scanned any
	if seq > 0 {
		scanned = seq
	}
	for _, r := range refs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO git_scan_cursors (project_id, kind, name, last_tip, scanned_seq, scanned_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (project_id, kind, name) DO UPDATE SET
    last_tip = excluded.last_tip,
    scanned_seq = coalesce(excluded.scanned_seq, git_scan_cursors.scanned_seq),
    scanned_at = excluded.scanned_at`, project, r.Kind, r.Name, r.Tip, scanned, at); err != nil {
			return err
		}
	}
	return nil
}
