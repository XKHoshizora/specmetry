-- 0007_commits: the commit scan (M2 WP2.3; data-model §4.12, WP2.3 spec §7).
-- A commit is stored only when its message names at least one requirement
-- or criterion of the project, once per project, with its links, by the
-- scan that first finds it: the commits one scan stores share that scan's
-- seq, so seq is not unique here, and the links carry none (their commit's
-- seq is when they were found). Only the SHA, the committer time (UTC Unix
-- milliseconds, for display) and the links are kept, never the message.
-- Append-only.
CREATE TABLE commits (
    id           TEXT    PRIMARY KEY CHECK (length(id) = 26),
    seq          INTEGER NOT NULL,
    project_id   TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    sha          TEXT    NOT NULL CHECK (length(sha) = 40 AND sha NOT GLOB '*[^0-9a-f]*'),
    committed_at INTEGER NOT NULL,
    UNIQUE (project_id, sha),
    UNIQUE (id, project_id)
);
CREATE INDEX commits_by_seq ON commits (project_id, seq);
CREATE TRIGGER commits_append_only_update BEFORE UPDATE ON commits
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table commits'); END;
CREATE TRIGGER commits_append_only_delete BEFORE DELETE ON commits
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table commits'); END;

-- A commit's links: one per requirement or criterion it names, from a
-- Specmetry-Ref trailer or else from the message. They are written only in
-- the transaction that first stores their commit (the scan's single write
-- entry skips stored SHAs); nothing here can tell that, so the scan's
-- tests check it. The target must be a requirement or criterion of the
-- link's own project, of the kind target_type says.
CREATE TABLE commit_links (
    commit_id   TEXT NOT NULL,
    project_id  TEXT NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    target_type TEXT NOT NULL CHECK (target_type IN ('requirement', 'criterion')),
    target_id   TEXT NOT NULL CHECK (length(target_id) = 26),
    link_source TEXT NOT NULL CHECK (link_source IN ('trailer', 'message')),
    PRIMARY KEY (commit_id, target_type, target_id),
    FOREIGN KEY (commit_id, project_id) REFERENCES commits (id, project_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX commit_links_by_target ON commit_links (project_id, target_type, target_id);
CREATE TRIGGER commit_links_target BEFORE INSERT ON commit_links
WHEN NOT (
    (NEW.target_type = 'requirement'
        AND EXISTS (SELECT 1 FROM requirements WHERE id = NEW.target_id AND project_id = NEW.project_id))
    OR (NEW.target_type = 'criterion'
        AND EXISTS (SELECT 1 FROM acceptance_criteria WHERE id = NEW.target_id AND project_id = NEW.project_id)))
BEGIN SELECT RAISE(ABORT, 'commit_links target must be a requirement or criterion of the link''s project'); END;
CREATE TRIGGER commit_links_append_only_update BEFORE UPDATE ON commit_links
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table commit_links'); END;
CREATE TRIGGER commit_links_append_only_delete BEFORE DELETE ON commit_links
WHEN (SELECT active FROM maintenance WHERE id = 1) = 0
BEGIN SELECT RAISE(ABORT, 'append-only table commit_links'); END;

-- One cursor per scanned ref: a local branch (its full ref name) or the
-- detached HEAD of a worktree (its canonical path, D2). last_tip is the
-- commit the ref pointed to when it was last scanned; scanned_seq the seq
-- of the last scan that stored new commits and wrote this cursor (NULL if
-- none ever did), for diagnosis only; scanned_at (UTC Unix milliseconds)
-- for display and throttling only. Written only by the scan's write
-- transaction; a cursor whose ref is gone stays, and its last_tip still
-- bounds the scan of new refs.
CREATE TABLE git_scan_cursors (
    project_id  TEXT    NOT NULL REFERENCES projects (id) DEFERRABLE INITIALLY DEFERRED,
    kind        TEXT    NOT NULL CHECK (kind IN ('branch', 'detached')),
    name        TEXT    NOT NULL CHECK (length(name) >= 1),
    last_tip    TEXT    NOT NULL CHECK (length(last_tip) = 40 AND last_tip NOT GLOB '*[^0-9a-f]*'),
    scanned_seq INTEGER,
    scanned_at  INTEGER NOT NULL,
    PRIMARY KEY (project_id, kind, name)
);
