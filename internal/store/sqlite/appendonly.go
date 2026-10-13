package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
)

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// AppendOnlyTriggers returns DDL that makes table append-only: UPDATE and
// DELETE abort unless the maintenance switch is on (data-model.md §1).
// Migrations that create fact tables append this DDL.
func AppendOnlyTriggers(table string) (string, error) {
	if !identifier.MatchString(table) {
		return "", fmt.Errorf("invalid table name %q", table)
	}
	return fmt.Sprintf(`
CREATE TRIGGER %[1]s_append_only_update BEFORE UPDATE ON %[1]s
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table %[1]s'); END;
CREATE TRIGGER %[1]s_append_only_delete BEFORE DELETE ON %[1]s
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table %[1]s'); END;
`, table), nil
}

// withMaintenance turns the maintenance switch on for the duration of fn
// inside tx. Other connections never observe the switch: they cannot write
// while tx holds the write lock, and the switch is off again before commit.
// If fn fails the caller must roll tx back.
func withMaintenance(ctx context.Context, tx *sql.Tx, fn func() error) error {
	if _, err := tx.ExecContext(ctx, `UPDATE maintenance SET active = 1 WHERE id = 1`); err != nil {
		return err
	}
	if err := fn(); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE maintenance SET active = 0 WHERE id = 1`)
	return err
}
