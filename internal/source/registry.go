package source

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/internal/ulid"
)

// Source is a registered source.
type Source struct {
	ID            string
	Kind          Kind
	Location      Location
	Path          string
	AckedSHA256   string
	Size          int64
	RegisteredSeq int64
	// RemovedSeq is nil while the source is active.
	RemovedSeq *int64
}

// Member returns the source as the set hash sees it.
func (s Source) Member() Member { return Member{ID: s.ID, AckedSHA256: s.AckedSHA256} }

// File is the file to read for the source of a project rooted at root.
func (s Source) File(root string) string {
	if s.Location == External {
		return s.Path
	}
	return filepath.Join(root, filepath.FromSlash(s.Path))
}

// State is a source's state on disk (sourceState in common.v1).
type State string

// States.
const (
	Unchanged State = "unchanged"
	Changed   State = "changed"
	Missing   State = "missing"
)

// Observation is what the disk holds for an active source now. It is
// never stored: every read observes again (data-model §4.3).
type Observation struct {
	// CurrentSHA256 is "" when the file cannot be read.
	CurrentSHA256 string
	State         State
}

// Observe hashes an active source's file. A file that cannot be read is
// missing; only cancellation is an error.
func Observe(ctx context.Context, root string, s Source) (Observation, error) {
	sum, _, err := HashFile(ctx, s.File(root))
	switch {
	case errors.Is(err, ErrMissing):
		return Observation{State: Missing}, nil
	case err != nil:
		return Observation{}, err
	case sum == s.AckedSHA256:
		return Observation{CurrentSHA256: sum, State: Unchanged}, nil
	default:
		return Observation{CurrentSHA256: sum, State: Changed}, nil
	}
}

const columns = `id, kind, location, path, acked_sha256, size_bytes, registered_seq, removed_seq`

func scan(rows interface{ Scan(...any) error }) (Source, error) {
	var s Source
	var removed sql.NullInt64
	err := rows.Scan(&s.ID, &s.Kind, &s.Location, &s.Path, &s.AckedSHA256, &s.Size, &s.RegisteredSeq, &removed)
	if removed.Valid {
		s.RemovedSeq = &removed.Int64
	}
	return s, err
}

// List returns a project's sources in registration order: the active
// ones, and the removed ones too if includeRemoved.
func List(ctx context.Context, tx *sql.Tx, projectID string, includeRemoved bool) ([]Source, error) {
	q := `SELECT ` + columns + ` FROM sources WHERE project_id = ?`
	if !includeRemoved {
		q += ` AND removed_seq IS NULL`
	}
	rows, err := tx.QueryContext(ctx, q+` ORDER BY registered_seq`, projectID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Source
	for rows.Next() {
		s, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ProjectSetHash is the source set hash of a project's active sources.
func ProjectSetHash(ctx context.Context, tx *sql.Tx, projectID string) (string, error) {
	active, err := List(ctx, tx, projectID, false)
	if err != nil {
		return "", err
	}
	members := make([]Member, len(active))
	for i, s := range active {
		members[i] = s.Member()
	}
	return SetHash(members), nil
}

// FindActive finds an active source by its stored path or its ID.
func FindActive(ctx context.Context, tx *sql.Tx, projectID, ref string) (Source, bool, error) {
	s, err := scan(tx.QueryRowContext(ctx, `SELECT `+columns+` FROM sources
WHERE project_id = ? AND removed_seq IS NULL AND (path = ? OR id = ?)`, projectID, ref, ref))
	if errors.Is(err, sql.ErrNoRows) {
		return Source{}, false, nil
	}
	return s, err == nil, err
}

// Ref is a reference to a source as source-list.v1 shows it (--refs).
// Under hash_only the excerpt is not stored (excerpt_state not_stored) and
// Excerpt stays null.
type Ref struct {
	Seq            int64   `json:"seq"`
	EntityType     string  `json:"entity_type"`
	EntityKey      string  `json:"entity_key"`
	EntityRevision int     `json:"entity_revision"`
	Current        bool    `json:"current"`
	Locator        *string `json:"locator"`
	Verification   string  `json:"verification"`
	ExcerptSHA256  string  `json:"excerpt_sha256"`
	ExcerptState   string  `json:"excerpt_state"`
	Excerpt        *string `json:"excerpt"`
}

// Refs returns every reference to the project's sources, current and
// historical, by source ID, in the order they were written. A reference
// is current when it is bound to its entity's current revision (a
// question has only one).
func Refs(ctx context.Context, tx *sql.Tx, projectID string) (map[string][]Ref, error) {
	rows, err := tx.QueryContext(ctx, `SELECT r.source_id, r.seq, r.entity_type, r.entity_revision, r.locator, r.verification,
    r.excerpt_sha256, r.excerpt, r.excerpt_erased_seq IS NOT NULL,
    coalesce(g.key, rq.key, ac.key, q.key, ''),
    CASE r.entity_type WHEN 'goal' THEN g.revision WHEN 'requirement' THEN rq.revision
        WHEN 'criterion' THEN ac.revision ELSE 1 END
FROM source_refs r
LEFT JOIN goals g ON r.entity_type = 'goal' AND g.id = r.entity_id
LEFT JOIN requirements rq ON r.entity_type = 'requirement' AND rq.id = r.entity_id
LEFT JOIN acceptance_criteria ac ON r.entity_type = 'criterion' AND ac.id = r.entity_id
LEFT JOIN questions q ON r.entity_type = 'question' AND q.id = r.entity_id
WHERE r.project_id = ?
ORDER BY r.seq, r.rowid`, projectID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string][]Ref{}
	for rows.Next() {
		var sourceID string
		var r Ref
		var erased bool
		var current sql.NullInt64
		if err := rows.Scan(&sourceID, &r.Seq, &r.EntityType, &r.EntityRevision, &r.Locator, &r.Verification,
			&r.ExcerptSHA256, &r.Excerpt, &erased, &r.EntityKey, &current); err != nil {
			return nil, err
		}
		r.Current = current.Valid && int(current.Int64) == r.EntityRevision
		switch {
		case erased:
			r.ExcerptState = "erased"
		case r.Excerpt != nil:
			r.ExcerptState = "stored"
		default:
			r.ExcerptState = "not_stored"
		}
		out[sourceID] = append(out[sourceID], r)
	}
	return out, rows.Err()
}

// FindRemoved reports whether ref is the stored path or the ID of a
// removed source of the project.
func FindRemoved(ctx context.Context, tx *sql.Tx, projectID, ref string) (bool, error) {
	var n int
	err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sources
WHERE project_id = ? AND removed_seq IS NOT NULL AND (path = ? OR id = ?)`, projectID, ref, ref).Scan(&n)
	return n > 0, err
}

// Register adds the source unless its path is already an active source,
// which is returned as it is: registering again changes nothing, not even
// the acknowledged hash (data-model §4.3). It reports whether the source
// was created.
func Register(ctx context.Context, tx *sql.Tx, projectID string, r Resolved, kind Kind, sha256 string, size int64) (Source, bool, error) {
	if s, found, err := FindActive(ctx, tx, projectID, r.Path); err != nil || found {
		return s, false, err
	}
	seq, err := sqlite.NextSeq(ctx, tx)
	if err != nil {
		return Source{}, false, err
	}
	s := Source{ID: ulid.New(), Kind: kind, Location: r.Location, Path: r.Path, AckedSHA256: sha256, Size: size, RegisteredSeq: seq}
	_, err = tx.ExecContext(ctx, `INSERT INTO sources (id, project_id, kind, location, path, acked_sha256, size_bytes, registered_seq)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, s.ID, projectID, s.Kind, s.Location, s.Path, s.AckedSHA256, s.Size, s.RegisteredSeq)
	return s, err == nil, err
}

// Acknowledge records a person's acknowledgement that the source now has
// the content hashed as observed, and makes that its acknowledged hash.
// The caller has checked observed against the disk.
func Acknowledge(ctx context.Context, tx *sql.Tx, projectID string, s Source, observed, rationale, actor, channel string) (int64, error) {
	seq, err := sqlite.NextSeq(ctx, tx)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO source_acks (id, seq, project_id, source_id, previous_sha256, acked_sha256, rationale, actor, channel)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, ulid.New(), seq, projectID, s.ID, s.AckedSHA256, observed, rationale, actor, channel); err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `UPDATE sources SET acked_sha256 = ? WHERE id = ? AND project_id = ? AND removed_seq IS NULL`, observed, s.ID, projectID)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return 0, apperr.Newf(apperr.NotFound, "source %s is no longer active", s.ID)
	}
	return seq, nil
}

// Remove marks an active source removed. Its references stay (data-model
// §4.3); registering its path again creates a new source.
func Remove(ctx context.Context, tx *sql.Tx, projectID string, s Source) (int64, error) {
	seq, err := sqlite.NextSeq(ctx, tx)
	if err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `UPDATE sources SET removed_seq = ? WHERE id = ? AND project_id = ? AND removed_seq IS NULL`, seq, s.ID, projectID)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return 0, apperr.Newf(apperr.NotFound, "source %s is no longer active", s.ID)
	}
	return seq, nil
}
