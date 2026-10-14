-- 0004_proposals: the baseline tables, proposals and their items,
-- questions, source and code references, and key counters (M1 WP1.5;
-- data-model.md §4.4 – §4.8, §4.14). Proposals only read the baseline
-- tables; a person's review (WP1.6) writes them.

-- baseline_versions is append-only: one row for each applied review or
-- human edit, numbered from 1 within the project.
CREATE TABLE baseline_versions (
    id         TEXT    PRIMARY KEY CHECK (length(id) = 26),
    seq        INTEGER NOT NULL UNIQUE,
    project_id TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    number     INTEGER NOT NULL CHECK (number >= 1),
    summary    TEXT    NOT NULL CHECK (length(summary) BETWEEN 1 AND 500),
    actor      TEXT    NOT NULL CHECK (actor GLOB 'human:?*'),
    channel    TEXT    NOT NULL CHECK (channel IN ('human_cli', 'dashboard')),
    UNIQUE (project_id, number),
    UNIQUE (id, project_id)
);
CREATE TRIGGER baseline_versions_append_only_update BEFORE UPDATE ON baseline_versions
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table baseline_versions'); END;
CREATE TRIGGER baseline_versions_append_only_delete BEFORE DELETE ON baseline_versions
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table baseline_versions'); END;

-- The baseline entities are controlled-mutable: only applying a baseline
-- change modifies them. A key never changes, a retired entity never
-- changes again, an entity is never deleted, and a criterion never moves
-- to another requirement.
CREATE TABLE goals (
    id              TEXT    PRIMARY KEY CHECK (length(id) = 26),
    project_id      TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    key             TEXT    NOT NULL CHECK (key GLOB 'GOAL-[1-9]*' AND substr(key, 6) NOT GLOB '*[^0-9]*'),
    status          TEXT    NOT NULL CHECK (status IN ('active', 'retired')),
    revision        INTEGER NOT NULL CHECK (revision >= 1),
    revision_seq    INTEGER NOT NULL,
    origin          TEXT    NOT NULL CHECK (origin IN ('extracted', 'inferred_from_code', 'manual')),
    confidence      TEXT    CHECK (confidence IN ('high', 'medium', 'low')),
    created_version INTEGER NOT NULL CHECK (created_version >= 1),
    updated_version INTEGER NOT NULL CHECK (updated_version >= created_version),
    retired_version INTEGER CHECK (retired_version >= created_version),
    updated_seq     INTEGER NOT NULL,
    title           TEXT    NOT NULL CHECK (length(title) BETWEEN 1 AND 200 AND title NOT GLOB '*[' || char(10, 13) || ']*'),
    description     TEXT    CHECK (length(description) <= 4000),
    CHECK ((status = 'retired') = (retired_version IS NOT NULL)),
    UNIQUE (project_id, key),
    UNIQUE (id, project_id)
);

CREATE TABLE requirements (
    id              TEXT    PRIMARY KEY CHECK (length(id) = 26),
    project_id      TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    key             TEXT    NOT NULL CHECK (key GLOB 'REQ-[0-9][0-9][0-9]*' AND substr(key, 5) NOT GLOB '*[^0-9]*'),
    status          TEXT    NOT NULL CHECK (status IN ('active', 'retired')),
    revision        INTEGER NOT NULL CHECK (revision >= 1),
    revision_seq    INTEGER NOT NULL,
    origin          TEXT    NOT NULL CHECK (origin IN ('extracted', 'inferred_from_code', 'manual')),
    confidence      TEXT    CHECK (confidence IN ('high', 'medium', 'low')),
    created_version INTEGER NOT NULL CHECK (created_version >= 1),
    updated_version INTEGER NOT NULL CHECK (updated_version >= created_version),
    retired_version INTEGER CHECK (retired_version >= created_version),
    updated_seq     INTEGER NOT NULL,
    goal_id         TEXT,
    title           TEXT    NOT NULL CHECK (length(title) BETWEEN 1 AND 200 AND title NOT GLOB '*[' || char(10, 13) || ']*'),
    description     TEXT    CHECK (length(description) <= 4000),
    kind            TEXT    NOT NULL CHECK (kind IN ('functional', 'non_functional', 'constraint')),
    priority        TEXT    NOT NULL CHECK (priority IN ('must', 'should', 'could', 'wont')),
    CHECK ((status = 'retired') = (retired_version IS NOT NULL)),
    FOREIGN KEY (goal_id, project_id) REFERENCES goals (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    UNIQUE (project_id, key),
    UNIQUE (id, project_id)
);

CREATE TABLE acceptance_criteria (
    id                   TEXT    PRIMARY KEY CHECK (length(id) = 26),
    project_id           TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    key                  TEXT    NOT NULL CHECK (key GLOB 'AC-[0-9][0-9][0-9]*-[1-9]*' AND replace(substr(key, 4), '-', '') NOT GLOB '*[^0-9]*'),
    status               TEXT    NOT NULL CHECK (status IN ('active', 'retired')),
    revision             INTEGER NOT NULL CHECK (revision >= 1),
    revision_seq         INTEGER NOT NULL,
    origin               TEXT    NOT NULL CHECK (origin IN ('extracted', 'inferred_from_code', 'manual')),
    confidence           TEXT    CHECK (confidence IN ('high', 'medium', 'low')),
    created_version      INTEGER NOT NULL CHECK (created_version >= 1),
    updated_version      INTEGER NOT NULL CHECK (updated_version >= created_version),
    retired_version      INTEGER CHECK (retired_version >= created_version),
    updated_seq          INTEGER NOT NULL,
    requirement_id       TEXT    NOT NULL,
    requirement_revision INTEGER NOT NULL CHECK (requirement_revision >= 1),
    statement            TEXT    NOT NULL CHECK (length(statement) BETWEEN 1 AND 1000),
    verification         TEXT    NOT NULL CHECK (verification IN ('automated', 'manual')),
    check_command        TEXT    CHECK (length(check_command) BETWEEN 1 AND 2000),
    check_workdir        TEXT    NOT NULL DEFAULT '.',
    check_shell          TEXT    CHECK (check_shell IN ('sh', 'bash', 'cmd', 'pwsh', 'powershell')),
    check_inputs_json    TEXT    CHECK (check_inputs_json IS NULL OR json_type(check_inputs_json) = 'array'),
    check_rationale      TEXT,
    check_timeout_sec    INTEGER NOT NULL DEFAULT 300 CHECK (check_timeout_sec BETWEEN 1 AND 3600),
    check_config_hash    TEXT    CHECK (check_config_hash IS NULL OR (length(check_config_hash) = 64 AND check_config_hash NOT GLOB '*[^0-9a-f]*')),
    CHECK ((status = 'retired') = (retired_version IS NOT NULL)),
    -- A check makes the criterion automated and needs a rationale (data-model §4.4).
    CHECK (check_command IS NULL OR (verification = 'automated' AND check_rationale IS NOT NULL AND check_config_hash IS NOT NULL)),
    FOREIGN KEY (requirement_id, project_id) REFERENCES requirements (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    UNIQUE (project_id, key),
    UNIQUE (id, project_id)
);

CREATE TRIGGER goals_key_is_fixed BEFORE UPDATE OF key ON goals
WHEN NEW.key IS NOT OLD.key AND (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'a baseline key never changes'); END;
CREATE TRIGGER goals_retired_is_final BEFORE UPDATE ON goals
WHEN OLD.status = 'retired' AND (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'a retired baseline entity does not change'); END;
CREATE TRIGGER goals_never_deleted BEFORE DELETE ON goals
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'baseline entities are retired, not deleted'); END;

CREATE TRIGGER requirements_key_is_fixed BEFORE UPDATE OF key ON requirements
WHEN NEW.key IS NOT OLD.key AND (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'a baseline key never changes'); END;
CREATE TRIGGER requirements_retired_is_final BEFORE UPDATE ON requirements
WHEN OLD.status = 'retired' AND (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'a retired baseline entity does not change'); END;
CREATE TRIGGER requirements_never_deleted BEFORE DELETE ON requirements
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'baseline entities are retired, not deleted'); END;

CREATE TRIGGER acceptance_criteria_key_is_fixed BEFORE UPDATE OF key ON acceptance_criteria
WHEN NEW.key IS NOT OLD.key AND (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'a baseline key never changes'); END;
CREATE TRIGGER acceptance_criteria_stays_with_its_requirement BEFORE UPDATE OF requirement_id ON acceptance_criteria
WHEN NEW.requirement_id IS NOT OLD.requirement_id AND (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'a criterion does not move to another requirement'); END;
CREATE TRIGGER acceptance_criteria_retired_is_final BEFORE UPDATE ON acceptance_criteria
WHEN OLD.status = 'retired' AND (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'a retired baseline entity does not change'); END;
CREATE TRIGGER acceptance_criteria_never_deleted BEFORE DELETE ON acceptance_criteria
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'baseline entities are retired, not deleted'); END;

-- proposals never change once submitted. human_edit only comes from a
-- human channel (V-P01), and the actor matches the channel.
CREATE TABLE proposals (
    id           TEXT    PRIMARY KEY CHECK (length(id) = 26),
    seq          INTEGER NOT NULL UNIQUE,
    project_id   TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    origin       TEXT    NOT NULL CHECK (origin IN ('skill_init', 'skill_goals', 'skill_audit', 'skill_sync', 'cli', 'human_edit')),
    actor        TEXT    NOT NULL CHECK (actor GLOB 'agent:[a-z]*' OR actor GLOB 'human:?*'),
    channel      TEXT    NOT NULL CHECK (channel IN ('agent_cli', 'human_cli', 'dashboard')),
    session_id   TEXT,
    worktree_id  TEXT    NOT NULL,
    summary      TEXT    NOT NULL CHECK (length(summary) BETWEEN 1 AND 500),
    base_version INTEGER NOT NULL CHECK (base_version >= 0),
    CHECK (origin <> 'human_edit' OR channel IN ('human_cli', 'dashboard')),
    CHECK ((channel = 'agent_cli') = (actor GLOB 'agent:*')),
    FOREIGN KEY (worktree_id, project_id) REFERENCES worktrees (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    UNIQUE (id, project_id)
);
CREATE TRIGGER proposals_never_change BEFORE UPDATE ON proposals
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'a submitted proposal does not change'); END;
CREATE TRIGGER proposals_never_deleted BEFORE DELETE ON proposals
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'proposals are never deleted'); END;

-- proposal_items: what was submitted never changes; the review fields are
-- written once (WP1.6). sources_json and code_refs_json hold the checked
-- references; under hash_only, never an excerpt (trigger below).
CREATE TABLE proposal_items (
    id                     TEXT    PRIMARY KEY CHECK (length(id) = 26),
    proposal_id            TEXT    NOT NULL,
    project_id             TEXT    NOT NULL,
    item_index             INTEGER NOT NULL CHECK (item_index >= 0),
    op                     TEXT    NOT NULL CHECK (op IN ('add', 'update', 'retire', 'reaffirm')),
    entity_type            TEXT    NOT NULL CHECK (entity_type IN ('goal', 'requirement', 'criterion')),
    target_id              TEXT    CHECK (length(target_id) = 26),
    target_revision        INTEGER CHECK (target_revision >= 1),
    ref                    TEXT    CHECK (length(ref) BETWEEN 1 AND 32),
    payload_json           TEXT    NOT NULL CHECK (json_type(payload_json) IN ('object', 'null')),
    suggested_change_class TEXT    CHECK (suggested_change_class IN ('material', 'editorial')),
    sources_json           TEXT    NOT NULL CHECK (json_type(sources_json) = 'array'),
    code_refs_json         TEXT    NOT NULL CHECK (json_type(code_refs_json) = 'array'),
    rationale              TEXT    CHECK (length(rationale) BETWEEN 1 AND 1000),
    origin                 TEXT    CHECK (origin IN ('extracted', 'inferred_from_code', 'manual')),
    confidence             TEXT    CHECK (confidence IN ('high', 'medium', 'low')),
    review_state           TEXT    NOT NULL DEFAULT 'pending'
                                   CHECK (review_state IN ('pending', 'conflict', 'accepted', 'accepted_with_edits', 'rejected')),
    final_payload_json     TEXT    CHECK (final_payload_json IS NULL OR json_valid(final_payload_json)),
    change_class           TEXT    CHECK (change_class IN ('material', 'editorial')),
    reviewed_by            TEXT    CHECK (reviewed_by GLOB 'human:?*'),
    reviewed_channel       TEXT    CHECK (reviewed_channel IN ('human_cli', 'dashboard')),
    reviewed_seq           INTEGER,
    review_note            TEXT,
    applied_version_id     TEXT,
    CHECK ((op = 'add') = (ref IS NOT NULL)),
    CHECK ((op = 'add') = (target_id IS NULL)),
    CHECK ((target_id IS NULL) = (target_revision IS NULL)),
    CHECK (op <> 'reaffirm' OR entity_type = 'criterion'),
    CHECK (op <> 'retire' OR rationale IS NOT NULL),
    CHECK (suggested_change_class IS NULL OR op = 'update'),
    CHECK (origin IS NOT 'inferred_from_code' OR json_array_length(code_refs_json) >= 1),
    CHECK ((review_state IN ('pending', 'conflict')) = (reviewed_seq IS NULL)),
    CHECK ((reviewed_seq IS NULL) = (reviewed_by IS NULL) AND (reviewed_seq IS NULL) = (reviewed_channel IS NULL)),
    FOREIGN KEY (proposal_id, project_id) REFERENCES proposals (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (applied_version_id, project_id) REFERENCES baseline_versions (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    UNIQUE (proposal_id, item_index),
    UNIQUE (id, project_id)
);
-- A proposal-local ref names one item of its proposal.
CREATE UNIQUE INDEX proposal_items_one_ref ON proposal_items (proposal_id, ref) WHERE ref IS NOT NULL;
CREATE TRIGGER proposal_items_submission_is_fixed
BEFORE UPDATE OF id, proposal_id, project_id, item_index, op, entity_type, target_id, target_revision, ref,
    payload_json, suggested_change_class, sources_json, code_refs_json, rationale, origin, confidence ON proposal_items
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'a submitted proposal item does not change'); END;
CREATE TRIGGER proposal_items_reviewed_once BEFORE UPDATE ON proposal_items
WHEN OLD.reviewed_seq IS NOT NULL AND (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'a reviewed proposal item does not change'); END;
CREATE TRIGGER proposal_items_never_deleted BEFORE DELETE ON proposal_items
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'proposal items are never deleted'); END;
-- hash_only (the default): no excerpt text in a stored reference (PD-24).
CREATE TRIGGER proposal_items_hash_only BEFORE INSERT ON proposal_items
WHEN EXISTS (SELECT 1 FROM json_each(NEW.sources_json) WHERE json_type(value, '$.excerpt') IS NOT NULL)
    AND coalesce((SELECT json_extract(settings_json, '$.excerpt_storage') FROM projects WHERE id = NEW.project_id), 'hash_only') <> 'full'
BEGIN SELECT RAISE(ABORT, 'excerpt text is not stored under hash_only'); END;

-- questions are created when their proposal is submitted; only a person
-- answers or dismisses one (WP1.7), once. related_type and related_id name
-- an existing entity; a question about an item of its proposal names the
-- item until a review accepts it (data-model §4.8).
CREATE TABLE questions (
    id               TEXT    PRIMARY KEY CHECK (length(id) = 26),
    project_id       TEXT    NOT NULL,
    key              TEXT    NOT NULL CHECK (key GLOB 'Q-[0-9][0-9][0-9]*' AND substr(key, 3) NOT GLOB '*[^0-9]*'),
    text             TEXT    NOT NULL CHECK (length(text) BETWEEN 1 AND 1000),
    context          TEXT    CHECK (length(context) <= 4000),
    related_type     TEXT    CHECK (related_type IN ('goal', 'requirement', 'criterion')),
    related_id       TEXT    CHECK (length(related_id) = 26),
    proposal_item_id TEXT,
    blocking         INTEGER NOT NULL CHECK (blocking IN (0, 1)),
    status           TEXT    NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'answered', 'dismissed')),
    answer           TEXT,
    answered_by      TEXT    CHECK (answered_by GLOB 'human:?*'),
    answered_channel TEXT    CHECK (answered_channel IN ('human_cli', 'dashboard')),
    answered_seq     INTEGER,
    origin           TEXT    NOT NULL CHECK (origin IN ('skill_init', 'skill_goals', 'skill_audit', 'skill_sync', 'cli', 'human_edit')),
    proposal_id      TEXT    NOT NULL,
    created_seq      INTEGER NOT NULL,
    CHECK ((related_type IS NULL) = (related_id IS NULL)),
    CHECK ((status = 'open') = (answered_seq IS NULL)),
    CHECK ((answered_seq IS NULL) = (answered_by IS NULL) AND (answered_seq IS NULL) = (answered_channel IS NULL)),
    FOREIGN KEY (proposal_id, project_id) REFERENCES proposals (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (proposal_item_id, project_id) REFERENCES proposal_items (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    UNIQUE (project_id, key),
    UNIQUE (id, project_id)
);
CREATE TRIGGER questions_closed_is_final BEFORE UPDATE ON questions
WHEN OLD.status <> 'open' AND (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'an answered or dismissed question does not change'); END;
CREATE TRIGGER questions_never_deleted BEFORE DELETE ON questions
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'questions are never deleted'); END;

-- source_refs and code_refs are append-only and bound to an entity's
-- revision (data-model §4.5). The only controlled exceptions, in
-- maintenance mode, are erasing excerpts and purge.
CREATE TABLE source_refs (
    id                 TEXT    PRIMARY KEY CHECK (length(id) = 26),
    seq                INTEGER NOT NULL,
    project_id         TEXT    NOT NULL,
    source_id          TEXT    NOT NULL,
    entity_type        TEXT    NOT NULL CHECK (entity_type IN ('goal', 'requirement', 'criterion', 'question')),
    entity_id          TEXT    NOT NULL CHECK (length(entity_id) = 26),
    entity_revision    INTEGER NOT NULL CHECK (entity_revision >= 1),
    locator            TEXT    CHECK (length(locator) <= 100),
    excerpt            TEXT    CHECK (length(excerpt) BETWEEN 1 AND 500),
    excerpt_sha256     TEXT    NOT NULL CHECK (length(excerpt_sha256) = 64 AND excerpt_sha256 NOT GLOB '*[^0-9a-f]*'),
    excerpt_erased_seq INTEGER,
    verification       TEXT    NOT NULL CHECK (verification IN ('verified', 'not_found', 'source_missing', 'not_checkable')),
    source_sha256      TEXT    CHECK (length(source_sha256) = 64 AND source_sha256 NOT GLOB '*[^0-9a-f]*'),
    CHECK (excerpt_erased_seq IS NULL OR excerpt IS NULL),
    -- A source that could not be read has no hash to bind the result to.
    CHECK ((verification = 'source_missing') = (source_sha256 IS NULL)),
    FOREIGN KEY (source_id, project_id) REFERENCES sources (id, project_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER source_refs_append_only_update BEFORE UPDATE ON source_refs
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table source_refs'); END;
CREATE TRIGGER source_refs_append_only_delete BEFORE DELETE ON source_refs
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table source_refs'); END;
CREATE TRIGGER source_refs_hash_only BEFORE INSERT ON source_refs
WHEN NEW.excerpt IS NOT NULL
    AND coalesce((SELECT json_extract(settings_json, '$.excerpt_storage') FROM projects WHERE id = NEW.project_id), 'hash_only') <> 'full'
BEGIN SELECT RAISE(ABORT, 'excerpt text is not stored under hash_only'); END;

CREATE TABLE code_refs (
    id              TEXT    PRIMARY KEY CHECK (length(id) = 26),
    seq             INTEGER NOT NULL,
    project_id      TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    entity_type     TEXT    NOT NULL CHECK (entity_type IN ('goal', 'requirement', 'criterion')),
    entity_id       TEXT    NOT NULL CHECK (length(entity_id) = 26),
    entity_revision INTEGER NOT NULL CHECK (entity_revision >= 1),
    path            TEXT    NOT NULL CHECK (length(path) BETWEEN 1 AND 1024),
    lines           TEXT    CHECK (lines GLOB '[1-9]*' AND replace(lines, '-', '') NOT GLOB '*[^0-9]*'),
    checked_head    TEXT    CHECK (length(checked_head) = 40 AND checked_head NOT GLOB '*[^0-9a-f]*'),
    "exists"        INTEGER NOT NULL CHECK ("exists" IN (0, 1))
);
CREATE TRIGGER code_refs_append_only_update BEFORE UPDATE ON code_refs
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table code_refs'); END;
CREATE TRIGGER code_refs_append_only_delete BEFORE DELETE ON code_refs
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table code_refs'); END;

-- key_counters: the next number of each key sequence (TD-14). A number is
-- never handed out twice: counters only grow and are never deleted.
-- Criteria are numbered per requirement (scope = the requirement's ID).
CREATE TABLE key_counters (
    project_id  TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    entity_type TEXT    NOT NULL CHECK (entity_type IN ('goal', 'requirement', 'criterion', 'question')),
    scope       TEXT    NOT NULL,
    next_value  INTEGER NOT NULL CHECK (next_value >= 1),
    CHECK ((entity_type = 'criterion') = (scope <> '')),
    PRIMARY KEY (project_id, entity_type, scope)
);
CREATE TRIGGER key_counters_only_grow BEFORE UPDATE ON key_counters
WHEN (NEW.next_value <= OLD.next_value OR NEW.project_id IS NOT OLD.project_id
      OR NEW.entity_type IS NOT OLD.entity_type OR NEW.scope IS NOT OLD.scope)
    AND (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'a key counter only grows'); END;
CREATE TRIGGER key_counters_never_deleted BEFORE DELETE ON key_counters
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'key counters are never deleted'); END;
