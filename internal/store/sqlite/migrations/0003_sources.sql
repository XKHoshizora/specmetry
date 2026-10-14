-- 0003_sources: requirement sources and their acknowledgements (M1 WP1.4;
-- data-model.md §4.3).

-- sources is controlled-mutable: a person's source ack updates acked_sha256,
-- a person's source remove sets removed_seq once. A removed source never
-- changes again; registering its path again creates a new source.
CREATE TABLE sources (
    id             TEXT    PRIMARY KEY CHECK (length(id) = 26),
    project_id     TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    kind           TEXT    NOT NULL CHECK (kind IN ('client_requirement', 'product_doc', 'design_doc', 'issue', 'note', 'other')),
    location       TEXT    NOT NULL CHECK (location IN ('in_repo', 'external')),
    path           TEXT    NOT NULL CHECK (length(path) BETWEEN 1 AND 4096),
    acked_sha256   TEXT    NOT NULL CHECK (length(acked_sha256) = 64 AND acked_sha256 NOT GLOB '*[^0-9a-f]*'),
    size_bytes     INTEGER NOT NULL CHECK (size_bytes >= 0),
    registered_seq INTEGER NOT NULL UNIQUE,
    removed_seq    INTEGER UNIQUE CHECK (removed_seq > registered_seq),
    UNIQUE (id, project_id)
);
-- One active source per path in a project.
CREATE UNIQUE INDEX sources_one_active_path ON sources (project_id, path) WHERE removed_seq IS NULL;

CREATE TRIGGER sources_removed_is_final BEFORE UPDATE ON sources
WHEN OLD.removed_seq IS NOT NULL AND (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'a removed source does not change'); END;
CREATE TRIGGER sources_never_deleted BEFORE DELETE ON sources
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'sources are removed, not deleted'); END;

-- source_acks is append-only, written only by a person (human channels).
CREATE TABLE source_acks (
    id              TEXT    PRIMARY KEY CHECK (length(id) = 26),
    seq             INTEGER NOT NULL UNIQUE,
    project_id      TEXT    NOT NULL,
    source_id       TEXT    NOT NULL,
    previous_sha256 TEXT    NOT NULL CHECK (length(previous_sha256) = 64 AND previous_sha256 NOT GLOB '*[^0-9a-f]*'),
    acked_sha256    TEXT    NOT NULL CHECK (length(acked_sha256) = 64 AND acked_sha256 NOT GLOB '*[^0-9a-f]*'),
    rationale       TEXT    NOT NULL CHECK (length(rationale) BETWEEN 1 AND 1000),
    actor           TEXT    NOT NULL CHECK (actor GLOB 'human:?*'),
    channel         TEXT    NOT NULL CHECK (channel IN ('human_cli', 'dashboard')),
    FOREIGN KEY (source_id, project_id) REFERENCES sources (id, project_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER source_acks_append_only_update BEFORE UPDATE ON source_acks
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table source_acks'); END;
CREATE TRIGGER source_acks_append_only_delete BEFORE DELETE ON source_acks
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table source_acks'); END;
