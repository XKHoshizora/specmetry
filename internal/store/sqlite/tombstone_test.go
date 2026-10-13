package sqlite

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

const (
	purgedProject = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6"
	keptProject   = "01J9Z3K5Q8R2T4V6W8X0Y2Z4A7"
)

func writeTombstones(t *testing.T, dir datadir.Dir, projects ...string) {
	t.Helper()
	var ts []Tombstone
	for _, p := range projects {
		ts = append(ts, Tombstone{ProjectID: p, PurgedAt: 1})
	}
	b, err := json.Marshal(ts)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir.Join(datadir.TombstonesFile), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// relatedTables is the shape M1 tables will have: a project registry keyed by
// id, a parent fact table whose name sorts before its child, and a child that
// references the parent. Deleting by table name order would delete the
// parent first.
func relatedTables(t *testing.T) Migration {
	t.Helper()
	ddl := ""
	for _, table := range []string{"a_parent", "b_child"} {
		d, err := AppendOnlyTriggers(table)
		if err != nil {
			t.Fatal(err)
		}
		ddl += d
	}
	return Migration{Version: 2, Name: "related", SQL: `
CREATE TABLE projects (id TEXT PRIMARY KEY, name TEXT NOT NULL);
CREATE TABLE a_parent (id INTEGER PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects (id), payload TEXT NOT NULL);
CREATE TABLE b_child (id INTEGER PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects (id), parent_id INTEGER NOT NULL REFERENCES a_parent (id), payload TEXT NOT NULL);
` + ddl}
}

func insertRelated(t *testing.T, db *DB, project, payload string) {
	t.Helper()
	_, err := db.sql.ExecContext(ctx, `
INSERT INTO projects (id, name) VALUES (?1, 'p');
INSERT INTO a_parent (project_id, payload) VALUES (?1, ?2);
INSERT INTO b_child (project_id, parent_id, payload) VALUES (?1, last_insert_rowid(), ?2);`, project, payload)
	if err != nil {
		t.Fatal(err)
	}
}

// codex-review-004 N1: tombstone deletion must not depend on the order of
// tables when they reference each other, and it removes the project's
// registry row (projects.id) as well as every project_id row.
func TestTombstonesRespectForeignKeysAndRegistry(t *testing.T) {
	dir := testutil.TempHome(t)
	migs := append(embedded(t), relatedTables(t))
	canaryPurged, canaryKept := testutil.NewCanary(t), testutil.NewCanary(t)
	db, _ := open(t, dir, Options{Migrations: migs})
	insertRelated(t, db, purgedProject, canaryPurged)
	insertRelated(t, db, keptProject, canaryKept)
	_ = db.Close()

	writeTombstones(t, dir, purgedProject)
	db, res := open(t, dir, Options{Migrations: migs})
	if res.TombstoneRowsGone != 3 {
		t.Fatalf("deleted %d rows, want 3 (projects, a_parent, b_child)", res.TombstoneRowsGone)
	}
	for _, table := range []string{"projects", "a_parent", "b_child"} {
		if n := countRows(t, db, `SELECT count(*) FROM `+table); n != 1 {
			t.Fatalf("%s has %d rows, want only the kept project's", table, n)
		}
	}
	_ = db.Close()
	if hits := testutil.FindInTree(t, dir.Path, canaryPurged); len(hits) != 0 {
		t.Fatalf("purged data still on disk: %v", hits)
	}
}

// M1 table rule (codex-review-004 N1): every table except the storage
// bookkeeping tables and the project registry carries project_id, so purge
// and tombstones can find all of a project's rows. Runs on the real
// migrations; M1 tables are checked as they are added.
func TestEveryTableHasProjectID(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{})
	rows, err := db.sql.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, n)
	}
	_ = rows.Close()
	exempt := map[string]bool{"meta": true, "maintenance": true, "projects": true}
	for _, table := range tables {
		if exempt[table] {
			continue
		}
		if n := countRows(t, db, `SELECT count(*) FROM pragma_table_info(?) WHERE name = 'project_id'`, table); n != 1 {
			t.Errorf("table %s has no project_id column", table)
		}
	}
}
