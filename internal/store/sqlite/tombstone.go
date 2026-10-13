package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"github.com/XKHoshizora/specmetry/internal/datadir"
)

// Tombstone records a purged project (tombstones.json, architecture.md §11.4).
type Tombstone struct {
	ProjectID string `json:"project_id"`
	PurgedAt  int64  `json:"purged_at"`
}

var ulidPattern = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

// readTombstones reads tombstones.json. A missing file means no tombstones.
func readTombstones(path string) ([]Tombstone, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ts []Tombstone
	if err := json.Unmarshal(b, &ts); err != nil {
		return nil, fmt.Errorf("tombstones file %s is not valid: %w", path, err)
	}
	for _, t := range ts {
		if !ulidPattern.MatchString(t.ProjectID) {
			return nil, fmt.Errorf("tombstones file %s: invalid project_id %q", path, t.ProjectID)
		}
	}
	return ts, nil
}

// tombstoneTarget is a table and the column that holds a project's ID.
type tombstoneTarget struct{ table, column string }

// compactionHook, if set, runs after tombstoned rows are deleted and before
// the files are compacted (tests simulate a crash there).
var compactionHook func() error

// applyTombstones deletes every row that belongs to a tombstoned project:
// from every table with a project_id column, and the project's row in the
// projects registry (keyed by id). Tables are discovered from the database
// itself, so this works on any schema version and runs before any migration
// or migration backup. Foreign keys are checked at commit rather than per
// statement, so the order of tables does not matter (codex-review-004 N1).
//
// The deletion is then compacted out of the files (checkpoint and VACUUM)
// so the data is not left in free pages or the WAL. A marker file is written
// before the deletion commits and removed only after compaction, so a crash
// in between is finished on the next open even though no rows are left to
// delete then. It reports how many rows were deleted.
func applyTombstones(ctx context.Context, db *sql.DB, dir datadir.Dir, ts []Tombstone) (int64, error) {
	marker := dir.Join(datadir.CompactionFile)
	_, err := os.Lstat(marker)
	pending := err == nil
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}
	var deleted int64
	if len(ts) > 0 {
		targets, err := tombstoneTargets(ctx, db)
		if err != nil {
			return 0, err
		}
		found, err := anyTombstonedRows(ctx, db, targets, ts)
		if err != nil {
			return 0, err
		}
		if found {
			if err := writeMarker(marker); err != nil {
				return 0, err
			}
			pending = true
			if deleted, err = deleteTombstoned(ctx, db, targets, ts); err != nil {
				return 0, err
			}
		}
	}
	if !pending {
		return deleted, nil
	}
	if compactionHook != nil {
		if err := compactionHook(); err != nil {
			return deleted, err
		}
	}
	if _, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return deleted, err
	}
	if _, err := db.ExecContext(ctx, `VACUUM`); err != nil {
		return deleted, err
	}
	return deleted, os.Remove(marker)
}

func writeMarker(path string) error {
	f, err := datadir.CreatePrivateFile(path, false)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func tombstoneTargets(ctx context.Context, db *sql.DB) ([]tombstoneTarget, error) {
	tables, err := tablesWithProjectID(ctx, db)
	if err != nil {
		return nil, err
	}
	var targets []tombstoneTarget
	for _, t := range tables {
		targets = append(targets, tombstoneTarget{t, "project_id"})
	}
	var registry int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('projects') WHERE name = 'id'`).Scan(&registry); err != nil {
		return nil, err
	}
	if registry > 0 {
		targets = append(targets, tombstoneTarget{"projects", "id"})
	}
	return targets, nil
}

func anyTombstonedRows(ctx context.Context, db *sql.DB, targets []tombstoneTarget, ts []Tombstone) (bool, error) {
	for _, target := range targets {
		for _, t := range ts {
			var n int
			// table and column come from the database's own schema, not from input.
			q := `SELECT count(*) FROM (SELECT 1 FROM "` + target.table + `" WHERE "` + target.column + `" = ? LIMIT 1)`
			if err := db.QueryRowContext(ctx, q, t.ProjectID).Scan(&n); err != nil {
				return false, err
			}
			if n > 0 {
				return true, nil
			}
		}
	}
	return false, nil
}

func deleteTombstoned(ctx context.Context, db *sql.DB, targets []tombstoneTarget, ts []Tombstone) (int64, error) {
	var deleted int64
	err := withTx(ctx, db, func(tx *sql.Tx) error {
		// Reset automatically when the transaction ends.
		if _, err := tx.ExecContext(ctx, `PRAGMA defer_foreign_keys = ON`); err != nil {
			return err
		}
		return withMaintenance(ctx, tx, func() error {
			for _, target := range targets {
				for _, t := range ts {
					// table and column come from the database's own schema, not from input.
					q := `DELETE FROM "` + target.table + `" WHERE "` + target.column + `" = ?` //nolint:gosec // see above
					res, err := tx.ExecContext(ctx, q, t.ProjectID)
					if err != nil {
						return err
					}
					n, _ := res.RowsAffected()
					deleted += n
				}
			}
			return nil
		})
	})
	return deleted, err
}

func tablesWithProjectID(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			_ = rows.Close()
			return nil, err
		}
		names = append(names, n)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	var out []string
	for _, n := range names {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info(?) WHERE name = 'project_id'`, n).Scan(&count); err != nil {
			return nil, err
		}
		if count > 0 {
			if strings.ContainsRune(n, '"') {
				return nil, fmt.Errorf("unsupported table name %q", n)
			}
			out = append(out, n)
		}
	}
	return out, nil
}
