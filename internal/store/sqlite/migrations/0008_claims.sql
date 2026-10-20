-- 0008_claims: claim batches and claims (M2 WP2.4; data-model §4.10, WP2.4
-- spec §8). One claim submission is one batch: the batch and all its
-- claims, unmapped work included, share the batch's seq, so seq is unique
-- in claim_batches and not in claims. The code context is the one the CLI
-- captured. A claim names an active criterion of its own project at the
-- revision current when it is written, or is unmapped work (target_type
-- none) with no criterion, revision or assertion. Summaries are stored;
-- references are paths, line ranges, test names and commit prefixes, and
-- their results, never file contents. Both tables are append-only.
CREATE TABLE claim_batches (
    id                  TEXT    PRIMARY KEY CHECK (length(id) = 26),
    seq                 INTEGER NOT NULL UNIQUE,
    project_id          TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    kind                TEXT    NOT NULL CHECK (kind IN ('sync', 'audit', 'manual')),
    audit_scope_json    TEXT    CHECK (audit_scope_json IS NULL OR (json_valid(audit_scope_json) AND json_type(audit_scope_json) = 'object')),
    base_version        INTEGER NOT NULL CHECK (base_version >= 0),
    worktree_id         TEXT    NOT NULL,
    head                TEXT    CHECK (head IS NULL OR (length(head) = 40 AND head NOT GLOB '*[^0-9a-f]*')),
    tree_state          TEXT    NOT NULL CHECK (tree_state IN ('clean', 'dirty', 'unknown')),
    compare_from        TEXT    CHECK (compare_from IS NULL OR (length(compare_from) = 40 AND compare_from NOT GLOB '*[^0-9a-f]*')),
    compare_from_source TEXT    CHECK (compare_from_source IN ('previous_batch', 'session_start', 'none')),
    session_id          TEXT,
    summary             TEXT    NOT NULL CHECK (length(summary) BETWEEN 1 AND 500),
    actor               TEXT    NOT NULL CHECK (actor GLOB 'agent:[a-z]*' OR actor GLOB 'human:?*'),
    channel             TEXT    NOT NULL CHECK (channel IN ('agent_cli', 'human_cli')),
    -- V-C05: an audit batch, and only one, declares its scope.
    CHECK ((kind = 'audit') = (audit_scope_json IS NOT NULL)),
    -- Only a sync batch has a comparison start; it is a commit unless its
    -- source is none.
    CHECK ((kind = 'sync') = (compare_from_source IS NOT NULL)),
    CHECK (compare_from IS NULL OR compare_from_source IS NOT NULL),
    CHECK (compare_from_source IS NULL OR (compare_from_source = 'none') = (compare_from IS NULL)),
    -- V-C01, M2-Q6 (WP2.4 spec E2): a manual batch only comes from a person
    -- at a terminal. A second line of defence; it proves nobody's identity.
    CHECK (kind <> 'manual' OR channel = 'human_cli'),
    CHECK ((channel = 'agent_cli') = (actor GLOB 'agent:*')),
    FOREIGN KEY (worktree_id, project_id) REFERENCES worktrees (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    UNIQUE (id, project_id)
);
-- compare_from reads the latest batch of a worktree (WP2.4 spec §6.3).
CREATE INDEX claim_batches_by_worktree ON claim_batches (project_id, worktree_id, seq);
CREATE TRIGGER claim_batches_append_only_update BEFORE UPDATE ON claim_batches
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table claim_batches'); END;
CREATE TRIGGER claim_batches_append_only_delete BEFORE DELETE ON claim_batches
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table claim_batches'); END;

-- refs_json holds the references as submitted (files with their line
-- ranges, test names, commit prefixes); refs_check_json their results in
-- the same order: {"files": [{path, lines, exists, checked_head}],
-- "commits": [{commit, resolved}]}.
CREATE TABLE claims (
    id                 TEXT    PRIMARY KEY CHECK (length(id) = 26),
    seq                INTEGER NOT NULL,
    project_id         TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    batch_id           TEXT    NOT NULL,
    target_type        TEXT    NOT NULL CHECK (target_type IN ('criterion', 'none')),
    criterion_id       TEXT    CHECK (length(criterion_id) = 26),
    criterion_revision INTEGER CHECK (criterion_revision >= 1),
    assertion          TEXT    CHECK (assertion IN ('done', 'partial', 'not_done', 'blocked', 'unknown')),
    summary            TEXT    NOT NULL CHECK (length(summary) BETWEEN 1 AND 500),
    refs_json          TEXT    NOT NULL CHECK (json_valid(refs_json) AND json_type(refs_json) = 'object'),
    refs_check_json    TEXT    NOT NULL CHECK (json_valid(refs_check_json) AND json_type(refs_check_json) = 'object'),
    -- Unmapped work names no criterion and asserts nothing.
    CHECK ((target_type = 'criterion') = (criterion_id IS NOT NULL)),
    CHECK ((target_type = 'criterion') = (criterion_revision IS NOT NULL)),
    CHECK ((target_type = 'criterion') = (assertion IS NOT NULL)),
    FOREIGN KEY (batch_id, project_id) REFERENCES claim_batches (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (criterion_id, project_id) REFERENCES acceptance_criteria (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    UNIQUE (id, project_id)
);
-- V-C03: a criterion at most once in a batch, so that the latest claim of
-- a criterion is never a tie (data-model §6.5).
CREATE UNIQUE INDEX claims_one_per_criterion ON claims (batch_id, criterion_id) WHERE criterion_id IS NOT NULL;
CREATE INDEX claims_by_criterion ON claims (project_id, criterion_id, seq);
-- A claim is written with its batch, in the batch's transaction (F4).
CREATE TRIGGER claims_share_their_batch_seq BEFORE INSERT ON claims
WHEN NOT EXISTS (SELECT 1 FROM claim_batches WHERE id = NEW.batch_id AND project_id = NEW.project_id AND seq = NEW.seq)
BEGIN SELECT RAISE(ABORT, 'a claim belongs to a batch of its project and shares its seq'); END;
-- V-C02, V-C03: the target is checked under the write lock; this is the
-- second line of defence.
CREATE TRIGGER claims_target BEFORE INSERT ON claims
WHEN NEW.criterion_id IS NOT NULL AND NEW.criterion_revision IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM acceptance_criteria
    WHERE id = NEW.criterion_id AND project_id = NEW.project_id AND status = 'active' AND revision = NEW.criterion_revision)
BEGIN SELECT RAISE(ABORT, 'a claim names an active criterion of its project at its current revision'); END;
-- V-C03: unknown is only an audit's answer.
CREATE TRIGGER claims_unknown_in_audit BEFORE INSERT ON claims
WHEN NEW.assertion = 'unknown' AND NOT EXISTS (SELECT 1 FROM claim_batches WHERE id = NEW.batch_id AND kind = 'audit')
BEGIN SELECT RAISE(ABORT, 'an unknown assertion is only made in an audit batch'); END;
CREATE TRIGGER claims_append_only_update BEFORE UPDATE ON claims
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table claims'); END;
CREATE TRIGGER claims_append_only_delete BEFORE DELETE ON claims
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table claims'); END;
