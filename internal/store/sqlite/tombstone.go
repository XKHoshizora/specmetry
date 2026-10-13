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

// applyTombstones deletes every row that belongs to a tombstoned project from
// every table that has a project_id column, then rewrites the file so the
// data is not left in free pages or the WAL. Tables are discovered from the
// database itself, so this works on any schema version and runs before any
// migration or migration backup. It reports how many rows were deleted.
func applyTombstones(ctx context.Context, db *sql.DB, path string) (int64, error) {
	ts, err := readTombstones(path)
	if err != nil || len(ts) == 0 {
		return 0, err
	}
	tables, err := tablesWithProjectID(ctx, db)
	if err != nil || len(tables) == 0 {
		return 0, err
	}
	var deleted int64
	err = withTx(ctx, db, func(tx *sql.Tx) error {
		return withMaintenance(ctx, tx, func() error {
			for _, table := range tables {
				for _, t := range ts {
					// table comes from the database's own schema, not from input.
					res, err := tx.ExecContext(ctx, `DELETE FROM "`+table+`" WHERE project_id = ?`, t.ProjectID) //nolint:gosec // see above
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
	if err != nil || deleted == 0 {
		return deleted, err
	}
	if _, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return deleted, err
	}
	_, err = db.ExecContext(ctx, `VACUUM`)
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
