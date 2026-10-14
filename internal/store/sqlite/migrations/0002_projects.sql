-- 0002_projects: project registry, worktrees and delivery targets (M1 WP1.1;
-- data-model.md §4.1, §4.2). All three are controlled-mutable tables.
-- References between them stay within one project (data-model.md §1.1); the
-- ones that form a cycle are checked at commit.

-- One row per project. A git project is identified by its git common
-- directory, so every worktree of a repository maps to the same project; a
-- project outside git has none and is identified by its main worktree path.
CREATE TABLE projects (
    id               TEXT    PRIMARY KEY CHECK (length(id) = 26),
    name             TEXT    NOT NULL CHECK (length(trim(name)) > 0 AND length(name) <= 200),
    git_common_dir   TEXT    UNIQUE,
    main_worktree_id TEXT    NOT NULL,
    stage            TEXT    NOT NULL CHECK (stage IN ('docs_only', 'in_development', 'incomplete_docs')),
    settings_json    TEXT    NOT NULL CHECK (json_valid(settings_json)),
    created_at       INTEGER NOT NULL,
    FOREIGN KEY (main_worktree_id, id) REFERENCES worktrees (id, project_id) DEFERRABLE INITIALLY DEFERRED
);

-- Each worktree a CLI command or Hook has run in. path is the canonical
-- absolute path; git_dir is the worktree's own git directory (NULL outside
-- git).
CREATE TABLE worktrees (
    id             TEXT    PRIMARY KEY CHECK (length(id) = 26),
    project_id     TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    path           TEXT    NOT NULL,
    git_dir        TEXT,
    is_main        INTEGER NOT NULL CHECK (is_main IN (0, 1)),
    registered_seq INTEGER NOT NULL UNIQUE,
    last_seen_at   INTEGER NOT NULL,
    UNIQUE (project_id, path),
    UNIQUE (id, project_id)
);
CREATE UNIQUE INDEX worktrees_one_main ON worktrees (project_id) WHERE is_main = 1;
-- Outside git the path is the project's identity.
CREATE UNIQUE INDEX worktrees_path_without_git ON worktrees (path) WHERE git_dir IS NULL;

-- Delivery targets: a branch or tag ref, changed only by a person except for
-- the default target init creates (actor and channel 'system'). Changing the
-- ref archives the old target and creates a new one; one is active (PD-19).
CREATE TABLE delivery_targets (
    id           TEXT    PRIMARY KEY CHECK (length(id) = 26),
    project_id   TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    name         TEXT    NOT NULL CHECK (length(trim(name)) > 0),
    ref          TEXT    NOT NULL CHECK (ref GLOB 'refs/heads/?*' OR ref GLOB 'refs/tags/?*'),
    created_seq  INTEGER NOT NULL UNIQUE,
    archived_seq INTEGER CHECK (archived_seq > created_seq),
    actor        TEXT    NOT NULL,
    channel      TEXT    NOT NULL CHECK (channel IN ('agent_cli', 'human_cli', 'dashboard', 'hook', 'system')),
    UNIQUE (id, project_id)
);
CREATE UNIQUE INDEX delivery_targets_one_active ON delivery_targets (project_id) WHERE archived_seq IS NULL;
