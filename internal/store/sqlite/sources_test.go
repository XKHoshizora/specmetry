package sqlite

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/XKHoshizora/specmetry/internal/testutil"
)

var (
	shaA = strings.Repeat("a", 64)
	shaB = strings.Repeat("b", 64)
)

func insertSource(tx *sql.Tx, id, project string) error {
	seq, err := NextSeq(ctx, tx)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO sources (id, project_id, kind, location, path, acked_sha256, size_bytes, registered_seq)
VALUES (?, ?, 'client_requirement', 'in_repo', 'docs/brief.md', ?, 10, ?)`, id, project, shaA, seq)
	return err
}

func removeSource(tx *sql.Tx, id string) error {
	seq, err := NextSeq(ctx, tx)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE sources SET removed_seq = ? WHERE id = ?`, seq, id)
	return err
}

func insertAck(tx *sql.Tx, id, source, channel string) error {
	seq, err := NextSeq(ctx, tx)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO source_acks (id, seq, project_id, source_id, previous_sha256, acked_sha256, rationale, actor, channel)
VALUES (?, ?, ?, ?, ?, ?, 'typo fix', 'human:me', ?)`, id, seq, projectA, source, shaA, shaB, channel)
	return err
}

const (
	sourceA  = "01J9Z3K5Q8R2T4V6W8X0Y2Z4F1"
	sourceA2 = "01J9Z3K5Q8R2T4V6W8X0Y2Z4F2"
	sourceB  = "01J9Z3K5Q8R2T4V6W8X0Y2Z4F3"
	ackA     = "01J9Z3K5Q8R2T4V6W8X0Y2Z4G1"
)

// data-model §4.3, enforced by the schema as well as by the commands: one
// active source per path; a removed source stays removed; acknowledgements
// are append-only facts made by a person, about a source of the same
// project.
func TestSourceTablesConstraints(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{})
	defer func() { _ = db.Close() }()
	ok := func(name string, fn func(tx *sql.Tx) error) {
		t.Helper()
		if err := db.WriteTx(ctx, fn); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	refused := func(name string, fn func(tx *sql.Tx) error) {
		t.Helper()
		err := db.WriteTx(ctx, fn)
		if err == nil {
			t.Errorf("%s was accepted", name)
		}
		t.Logf("%s: %v", name, err)
	}
	ok("projects", func(tx *sql.Tx) error {
		if err := insertProject(tx, projectA, "/r/a/.git", worktreeA, "/r/a"); err != nil {
			return err
		}
		return insertProject(tx, projectB, "/r/b/.git", worktreeB, "/r/b")
	})
	ok("a source", func(tx *sql.Tx) error { return insertSource(tx, sourceA, projectA) })
	ok("the same path in another project", func(tx *sql.Tx) error { return insertSource(tx, sourceB, projectB) })
	refused("the same active path twice", func(tx *sql.Tx) error { return insertSource(tx, sourceA2, projectA) })
	ok("an acknowledgement", func(tx *sql.Tx) error { return insertAck(tx, ackA, sourceA, "human_cli") })
	for _, channel := range []string{"agent_cli", "hook", "system"} {
		refused("an acknowledgement through "+channel, func(tx *sql.Tx) error {
			return insertAck(tx, "01J9Z3K5Q8R2T4V6W8X0Y2Z4G2", sourceA, channel)
		})
	}
	refused("an acknowledgement of another project's source", func(tx *sql.Tx) error {
		return insertAck(tx, "01J9Z3K5Q8R2T4V6W8X0Y2Z4G3", sourceB, "human_cli")
	})
	refused("changing an acknowledgement", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE source_acks SET rationale = 'other' WHERE id = ?`, ackA)
		return err
	})
	refused("deleting an acknowledgement", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM source_acks WHERE id = ?`, ackA)
		return err
	})
	refused("a hash that is not sha256", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE sources SET acked_sha256 = 'ABC' WHERE id = ?`, sourceA)
		return err
	})
	refused("an unknown kind", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE sources SET kind = 'spec' WHERE id = ?`, sourceA)
		return err
	})
	refused("removal before registration", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE sources SET removed_seq = registered_seq WHERE id = ?`, sourceA)
		return err
	})
	ok("removal", func(tx *sql.Tx) error { return removeSource(tx, sourceA) })
	ok("the removed path again, under a new ID", func(tx *sql.Tx) error { return insertSource(tx, sourceA2, projectA) })
	refused("bringing a removed source back", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE sources SET removed_seq = NULL WHERE id = ?`, sourceA)
		return err
	})
	refused("changing a removed source", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE sources SET acked_sha256 = ? WHERE id = ?`, shaB, sourceA)
		return err
	})
	refused("removing a source twice", func(tx *sql.Tx) error { return removeSource(tx, sourceA) })
	refused("deleting a source", func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM sources WHERE id = ?`, sourceA2)
		return err
	})
	if n := countRows(t, db, `SELECT count(*) FROM sources`); n != 3 {
		t.Fatalf("%d sources, want 3", n)
	}
}

// Tombstones remove a project's sources and acknowledgements too.
func TestTombstonesRemoveSources(t *testing.T) {
	dir := testutil.TempHome(t)
	db, _ := open(t, dir, Options{})
	if err := db.WriteTx(ctx, func(tx *sql.Tx) error {
		for _, p := range []struct{ id, wt, path, src string }{{projectA, worktreeA, "/r/a", sourceA}, {projectB, worktreeB, "/r/b", sourceB}} {
			if err := insertProject(tx, p.id, p.path+"/.git", p.wt, p.path); err != nil {
				return err
			}
			if err := insertSource(tx, p.src, p.id); err != nil {
				return err
			}
		}
		return insertAck(tx, ackA, sourceA, "human_cli")
	}); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	writeTombstones(t, dir, projectA)
	db, res := open(t, dir, Options{})
	defer func() { _ = db.Close() }()
	if res.TombstoneRowsGone != 4 {
		t.Fatalf("deleted %d rows, want 4 (project, worktree, source, acknowledgement)", res.TombstoneRowsGone)
	}
	for table, want := range map[string]int{"sources": 1, "source_acks": 0} {
		if n := countRows(t, db, `SELECT count(*) FROM `+table); n != want {
			t.Errorf("%s has %d rows, want %d", table, n, want)
		}
	}
}
