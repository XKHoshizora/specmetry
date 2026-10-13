package sqlite

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/testutil"
)

const factsRow = `SELECT group_concat(id || ':' || seq || ':' || project_id || ':' || payload, ',') FROM facts`

func factsContent(t *testing.T, db *DB) string {
	t.Helper()
	var s string
	if err := db.sql.QueryRowContext(ctx, factsRow).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// codex-review-004 B4: REPLACE conflict resolution deletes the conflicting
// row; on every connection that delete must hit the append-only trigger, for
// primary-key and UNIQUE conflicts alike, and the original fact must survive.
func TestAppendOnlyRejectsReplace(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{Migrations: append(embedded(t), factsMigration(t))})
	insertFact(t, db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", "original")
	original := factsContent(t, db)
	cases := map[string]string{
		"primary key, INSERT OR REPLACE": `INSERT OR REPLACE INTO facts (id, seq, project_id, payload) VALUES (1, 99, 'x', 'overwritten')`,
		"primary key, REPLACE INTO":      `REPLACE INTO facts (id, seq, project_id, payload) VALUES (1, 99, 'x', 'overwritten')`,
		"unique seq, INSERT OR REPLACE":  `INSERT OR REPLACE INTO facts (id, seq, project_id, payload) VALUES (2, 1, 'x', 'overwritten')`,
		"primary key, upsert":            `INSERT INTO facts (id, seq, project_id, payload) VALUES (1, 99, 'x', 'new') ON CONFLICT (id) DO UPDATE SET payload = excluded.payload`,
		"unique seq, upsert":             `INSERT INTO facts (id, seq, project_id, payload) VALUES (2, 1, 'x', 'new') ON CONFLICT (seq) DO UPDATE SET payload = excluded.payload`,
		"primary key, UPDATE OR REPLACE": `UPDATE OR REPLACE facts SET payload = 'overwritten' WHERE id = 1`,
	}
	for name, stmt := range cases {
		if _, err := db.sql.ExecContext(ctx, stmt); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Errorf("%s: err = %v, want the append-only trigger to abort it", name, err)
		}
		if got := factsContent(t, db); got != original {
			t.Fatalf("%s: facts changed from %q to %q", name, original, got)
		}
	}
}

// codex-review-004 B4: a table whose constraint itself says ON CONFLICT
// REPLACE turns a plain INSERT into a replace; the trigger still aborts it.
func TestAppendOnlyRejectsDeclaredReplaceConflict(t *testing.T) {
	dir := testutil.TempHome(t)
	ddl, err := AppendOnlyTriggers("declared")
	if err != nil {
		t.Fatal(err)
	}
	m := Migration{Version: 2, Name: "declared", SQL: `
CREATE TABLE declared (id INTEGER PRIMARY KEY, k TEXT UNIQUE ON CONFLICT REPLACE, v TEXT);
` + ddl}
	db, _ := open(t, dir, Options{Migrations: append(embedded(t), m)})
	if _, err := db.sql.ExecContext(ctx, `INSERT INTO declared (k, v) VALUES ('a', 'original')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.sql.ExecContext(ctx, `INSERT INTO declared (k, v) VALUES ('a', 'overwritten')`); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("err = %v", err)
	}
	var v string
	if err := db.sql.QueryRowContext(ctx, `SELECT group_concat(v) FROM declared`).Scan(&v); err != nil || v != "original" {
		t.Fatalf("rows = %q, %v", v, err)
	}
}

// codex-review-004 B4: the connection setting that makes REPLACE fire delete
// triggers is on for every connection the pool creates, not just the first.
func TestEveryPoolConnectionEnablesRecursiveTriggers(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{})
	db.sql.SetMaxIdleConns(0) // every Conn below is a new connection
	for i := 0; i < 3; i++ {
		conn, err := db.sql.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var on int
		if err := conn.QueryRowContext(ctx, `PRAGMA recursive_triggers`).Scan(&on); err != nil || on != 1 {
			t.Fatalf("connection %d: recursive_triggers = %d, %v", i, on, err)
		}
		_ = conn.Close()
	}
}

// codex-review-004 B4: a connection that lost the setting cannot write.
func TestWriteTxRefusesConnectionWithoutRecursiveTriggers(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{Migrations: append(embedded(t), factsMigration(t))})
	if _, err := db.sql.ExecContext(ctx, `PRAGMA recursive_triggers = 0`); err != nil {
		t.Fatal(err)
	}
	assertWriteRefused(t, db, apperr.ExitInternal, "recursive_triggers")
}

var replaceSyntax = regexp.MustCompile(`(?i)\bOR\s+REPLACE\b|\bREPLACE\s+INTO\b|\bON\s+CONFLICT\s+REPLACE\b`)

// codex-review-004 B4 (application layer): production SQL never uses REPLACE
// conflict resolution, which would turn an insert into a delete-and-insert of
// an existing row. Go string literals outside tests and all migrations are
// checked.
func TestProductionSQLNeverUsesReplace(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	var hits []string
	scanned := map[string]bool{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "dist") {
			return filepath.SkipDir
		}
		switch {
		case strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go"):
			scanned[filepath.Base(p)] = true
			f, err := parser.ParseFile(token.NewFileSet(), p, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if v, err := strconv.Unquote(lit.Value); err == nil && replaceSyntax.MatchString(v) {
						hits = append(hits, p)
					}
				}
				return true
			})
		case strings.HasSuffix(p, ".sql"):
			scanned[filepath.Base(p)] = true
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			var code []string
			for _, line := range strings.Split(string(b), "\n") {
				if i := strings.Index(line, "--"); i >= 0 {
					line = line[:i]
				}
				code = append(code, line)
			}
			if replaceSyntax.MatchString(strings.Join(code, "\n")) {
				hits = append(hits, p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("REPLACE conflict resolution used in production SQL: %v", hits)
	}
	// The scan must actually have seen the migrations and the Go code that
	// writes, and the pattern must match what it is meant to.
	for _, name := range []string{"0001_init.sql", "store.go", "tombstone.go", "doctor.go"} {
		if !scanned[name] {
			t.Fatalf("%s was not scanned", name)
		}
	}
	for _, sql := range []string{"insert or replace into x", "REPLACE INTO x", "UPDATE OR REPLACE x", "k TEXT UNIQUE ON CONFLICT REPLACE"} {
		if !replaceSyntax.MatchString(sql) {
			t.Fatalf("pattern misses %q", sql)
		}
	}
}
