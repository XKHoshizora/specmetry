-- 0006_scope: delivery scope confirmations (M1 WP1.8; data-model §4.2,
-- §5.3, §7.1). Append-only, made by a person only. A target's current
-- scope is its confirmation with the highest seq; whether it still holds
-- is derived (it is stale once the baseline version or the source set hash
-- moved on). A confirmation is bound to the target, the baseline version
-- and the source set hash it was made against, carries its attestations
-- (complete_scope true, pending_items_excluded, missing_goals_reviewed),
-- the waivers it acknowledged (decision IDs, V-T01) and the confirmation
-- it replaced (NULL for the first, V-T02). A review and the scope it
-- confirms share one seq; seq is unique among confirmations.
CREATE TABLE scope_confirmations (
    id                         TEXT    PRIMARY KEY CHECK (length(id) = 26),
    seq                        INTEGER NOT NULL UNIQUE,
    project_id                 TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    target_id                  TEXT    NOT NULL,
    baseline_version           INTEGER NOT NULL CHECK (baseline_version >= 1),
    source_set_hash            TEXT    NOT NULL CHECK (length(source_set_hash) = 64 AND source_set_hash NOT GLOB '*[^0-9a-f]*'),
    attestations_json          TEXT    NOT NULL CHECK (
        json_type(attestations_json) = 'object'
        AND json_type(attestations_json, '$.complete_scope') IS 'true'
        AND json_type(attestations_json, '$.pending_items_excluded') IS 'integer'
        AND json_extract(attestations_json, '$.pending_items_excluded') >= 0
        AND coalesce(json_type(attestations_json, '$.missing_goals_reviewed'), 'true') IN ('true', 'false')),
    acknowledged_waivers_json  TEXT    NOT NULL CHECK (json_type(acknowledged_waivers_json) = 'array'),
    seen_scope_confirmation_id TEXT    CHECK (length(seen_scope_confirmation_id) = 26),
    note                       TEXT    CHECK (length(note) BETWEEN 1 AND 1000),
    actor                      TEXT    NOT NULL CHECK (actor GLOB 'human:?*'),
    channel                    TEXT    NOT NULL CHECK (channel IN ('human_cli', 'dashboard')),
    confirmed_at               INTEGER NOT NULL,
    FOREIGN KEY (target_id, project_id) REFERENCES delivery_targets (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (seen_scope_confirmation_id, project_id) REFERENCES scope_confirmations (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    UNIQUE (id, project_id)
);
CREATE INDEX scope_confirmations_latest ON scope_confirmations (project_id, target_id, seq);
CREATE TRIGGER scope_confirmations_append_only_update BEFORE UPDATE ON scope_confirmations
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table scope_confirmations'); END;
CREATE TRIGGER scope_confirmations_append_only_delete BEFORE DELETE ON scope_confirmations
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table scope_confirmations'); END;

-- The requirements of a confirmation, each at the revision it had then; at
-- least one per confirmation, which the code writing them ensures.
CREATE TABLE scope_items (
    confirmation_id      TEXT    NOT NULL,
    project_id           TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    requirement_id       TEXT    NOT NULL,
    requirement_revision INTEGER NOT NULL CHECK (requirement_revision >= 1),
    PRIMARY KEY (confirmation_id, requirement_id),
    FOREIGN KEY (confirmation_id, project_id) REFERENCES scope_confirmations (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (requirement_id, project_id) REFERENCES requirements (id, project_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER scope_items_append_only_update BEFORE UPDATE ON scope_items
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table scope_items'); END;
CREATE TRIGGER scope_items_append_only_delete BEFORE DELETE ON scope_items
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table scope_items'); END;
