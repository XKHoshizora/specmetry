-- 0001_init: storage foundation (M0). Business tables arrive in M1+.

-- Single-row metadata: schema version, minimum binary version able to open
-- this database ('' = no requirement; only a migration that needs it sets it),
-- and the global fact sequence counter (data-model.md §1, §4.14).
CREATE TABLE meta (
    id                 INTEGER PRIMARY KEY CHECK (id = 1),
    schema_version     INTEGER NOT NULL,
    min_binary_version TEXT    NOT NULL DEFAULT '',
    seq_counter        INTEGER NOT NULL DEFAULT 0 CHECK (seq_counter >= 0)
);
INSERT INTO meta (id, schema_version, min_binary_version, seq_counter) VALUES (1, 0, '', 0);

-- Single-row switch that lets purge, retention cleanup and excerpt erasure
-- bypass append-only triggers inside one write transaction (architecture.md §11.4).
CREATE TABLE maintenance (
    id     INTEGER PRIMARY KEY CHECK (id = 1),
    active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1))
);
INSERT INTO maintenance (id, active) VALUES (1, 0);
