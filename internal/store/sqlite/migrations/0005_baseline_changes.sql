-- 0005_baseline_changes: every change a baseline version makes (M1 WP1.6;
-- data-model.md §4.7). Append-only: one row per changed entity, in the
-- order the version applied them (change_index), with the entity before
-- and after the change. before_json is null for add; change_class is set
-- exactly for update. proposal_item_id names the reviewed item, and is
-- null for a cascade or a criterion follow-up of the review; rationale
-- holds a follow-up's reason.
CREATE TABLE baseline_changes (
    id               TEXT    PRIMARY KEY CHECK (length(id) = 26),
    seq              INTEGER NOT NULL,
    project_id       TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    version_id       TEXT    NOT NULL,
    change_index     INTEGER NOT NULL CHECK (change_index >= 0),
    entity_type      TEXT    NOT NULL CHECK (entity_type IN ('goal', 'requirement', 'criterion')),
    entity_id        TEXT    NOT NULL CHECK (length(entity_id) = 26),
    op               TEXT    NOT NULL CHECK (op IN ('add', 'update', 'retire', 'reaffirm')),
    change_class     TEXT    CHECK (change_class IN ('material', 'editorial')),
    before_json      TEXT    CHECK (before_json IS NULL OR json_type(before_json) = 'object'),
    after_json       TEXT    NOT NULL CHECK (json_type(after_json) = 'object'),
    proposal_item_id TEXT,
    rationale        TEXT    CHECK (length(rationale) BETWEEN 1 AND 1000),
    CHECK ((op = 'update') = (change_class IS NOT NULL)),
    CHECK ((op = 'add') = (before_json IS NULL)),
    FOREIGN KEY (version_id, project_id) REFERENCES baseline_versions (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (proposal_item_id, project_id) REFERENCES proposal_items (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    UNIQUE (version_id, change_index)
);
CREATE INDEX baseline_changes_entity ON baseline_changes (project_id, entity_id);
CREATE TRIGGER baseline_changes_append_only_update BEFORE UPDATE ON baseline_changes
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table baseline_changes'); END;
CREATE TRIGGER baseline_changes_append_only_delete BEFORE DELETE ON baseline_changes
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table baseline_changes'); END;
