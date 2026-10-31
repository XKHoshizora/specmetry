-- 0009_check_runs: check runs (M2 WP2.5; data-model §4.11, WP2.5 spec §10,
-- §19.10 D14). One execution is one row with a seq of its own, unique in
-- the table, written after the run ends (§10.3). The row keeps the
-- criterion revision, requirement revision and check configuration read
-- when the run started: a later baseline change never refuses it (D13,
-- CX-15), so no trigger checks the current revision here. The command
-- text, the output and its hash are never stored. The observed runner of
-- the MVP writes evidence_grade observed and isolation_mode off only, and
-- no isolation column; a migration of M7 widens these. Append-only.
--
-- The CHECKs are the second line of defence for §6.2 and D14. A CHECK
-- passes when its expression is NULL, so every test below on a nullable
-- column uses IS / IS NOT, never = or IN alone, and a NULL can never let
-- an inconsistent row through. For the same reason id is NOT NULL: the
-- PRIMARY KEY of a rowid table allows NULL, and length(NULL) passes its
-- CHECK (Review 121 N-121-1).
CREATE TABLE check_runs (
    id                        TEXT    NOT NULL PRIMARY KEY CHECK (length(id) = 26),
    seq                       INTEGER NOT NULL UNIQUE,
    project_id                TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    criterion_id              TEXT    NOT NULL CHECK (length(criterion_id) = 26),
    criterion_revision        INTEGER NOT NULL CHECK (criterion_revision >= 1),
    requirement_revision      INTEGER NOT NULL CHECK (requirement_revision >= 1),
    check_config_hash         TEXT    NOT NULL CHECK (length(check_config_hash) = 64 AND check_config_hash NOT GLOB '*[^0-9a-f]*'),
    inputs_json               TEXT    CHECK (inputs_json IS NULL OR (json_valid(inputs_json) AND json_type(inputs_json) = 'array')),
    worktree_id               TEXT    NOT NULL,
    workdir                   TEXT    NOT NULL,
    evidence_grade            TEXT    NOT NULL CHECK (evidence_grade = 'observed'),
    isolation_mode            TEXT    NOT NULL CHECK (isolation_mode = 'off'),
    isolation_backend         TEXT    CHECK (isolation_backend IS NULL),
    isolation_fallback_reason TEXT    CHECK (isolation_fallback_reason IS NULL),
    head_start                TEXT    CHECK (head_start IS NULL OR (length(head_start) = 40 AND head_start NOT GLOB '*[^0-9a-f]*')),
    head_end                  TEXT    CHECK (head_end IS NULL OR (length(head_end) = 40 AND head_end NOT GLOB '*[^0-9a-f]*')),
    tree_state_start          TEXT    NOT NULL CHECK (tree_state_start IN ('clean', 'dirty', 'unknown')),
    tree_state_end            TEXT    NOT NULL CHECK (tree_state_end IN ('clean', 'dirty', 'unknown')),
    result                    TEXT    NOT NULL CHECK (result IN ('pass', 'fail', 'error')),
    -- D5: the observed runner never produces not_found; check-run.v1 and
    -- status.v1 keep it for readers.
    failure_kind              TEXT    CHECK (failure_kind IS NULL OR failure_kind IN ('nonzero_exit', 'timeout', 'spawn_error', 'input_missing', 'context_changed')),
    exit_code                 INTEGER,
    duration_ms               INTEGER NOT NULL CHECK (duration_ms >= 0),
    triggered_channel         TEXT    NOT NULL CHECK (triggered_channel IN ('human_cli', 'agent_cli')),
    session_id                TEXT,
    started_at                INTEGER NOT NULL,
    finished_at               INTEGER NOT NULL,
    -- §6.2, §10.1, D14: pass has no failure kind and exit code 0; fail is
    -- nonzero_exit with a nonzero code or NULL (a Unix signal, D12); error
    -- is one of the other kinds. timeout, spawn_error and input_missing
    -- have no exit code; context_changed may have any code, 0 included.
    -- Nothing here makes an error row non-qualifying (§7.4 decides that).
    CHECK ((result = 'pass') = (failure_kind IS NULL)),
    CHECK ((result = 'fail') = (failure_kind IS 'nonzero_exit')),
    CHECK (result IS NOT 'pass' OR exit_code IS 0),
    CHECK (result IS NOT 'fail' OR exit_code IS NOT 0),
    CHECK (failure_kind IS NULL OR failure_kind NOT IN ('timeout', 'spawn_error', 'input_missing') OR exit_code IS NULL),
    -- No process was started.
    CHECK (failure_kind IS NULL OR failure_kind NOT IN ('spawn_error', 'input_missing') OR duration_ms = 0),
    FOREIGN KEY (criterion_id, project_id) REFERENCES acceptance_criteria (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY (worktree_id, project_id) REFERENCES worktrees (id, project_id) DEFERRABLE INITIALLY DEFERRED,
    UNIQUE (id, project_id)
);
-- The derivation reads a criterion's runs in seq order (§10.1).
CREATE INDEX check_runs_by_criterion ON check_runs (project_id, criterion_id, seq);
CREATE TRIGGER check_runs_append_only_update BEFORE UPDATE ON check_runs
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table check_runs'); END;
CREATE TRIGGER check_runs_append_only_delete BEFORE DELETE ON check_runs
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table check_runs'); END;
