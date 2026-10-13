package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
)

// Migration is one forward-only schema step.
type Migration struct {
	Version int
	Name    string
	SQL     string
}

//go:embed migrations/*.sql
var migrationFS embed.FS

var migrationName = regexp.MustCompile(`^(\d{4})_([a-z0-9_]+)\.sql$`)

// Migrations returns the migrations embedded in the binary, in order.
func Migrations() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, err
	}
	var out []Migration
	for _, e := range entries {
		m := migrationName.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("bad migration file name %q", e.Name())
		}
		v, _ := strconv.Atoi(m[1])
		body, err := migrationFS.ReadFile(path.Join("migrations", e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, Migration{Version: v, Name: m[2], SQL: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	if err := validateSequence(out); err != nil {
		return nil, err
	}
	return out, nil
}

// SchemaVersion is the schema version produced by the embedded migrations.
func SchemaVersion() int {
	ms, err := Migrations()
	if err != nil || len(ms) == 0 {
		return 0
	}
	return ms[len(ms)-1].Version
}

func validateSequence(ms []Migration) error {
	for i, m := range ms {
		if m.Version != i+1 {
			return fmt.Errorf("migrations must be numbered 1..n without gaps; found %d at position %d", m.Version, i+1)
		}
	}
	return nil
}

// queryer is satisfied by *sql.DB, *sql.Conn and *sql.Tx.
type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// readMeta returns the schema version and minimum binary version, or (0, "")
// for a database that has never been migrated.
func readMeta(ctx context.Context, q queryer) (int, string, error) {
	var n int
	if err := q.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'meta'`).Scan(&n); err != nil {
		return 0, "", err
	}
	if n == 0 {
		return 0, "", nil
	}
	var v int
	var minBin string
	if err := q.QueryRowContext(ctx, `SELECT schema_version, min_binary_version FROM meta WHERE id = 1`).Scan(&v, &minBin); err != nil {
		return 0, "", err
	}
	return v, minBin, nil
}
