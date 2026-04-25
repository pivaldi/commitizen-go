CREATE TABLE statuses (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);
INSERT INTO statuses (id, name) VALUES (1, 'in_progress'), (2, 'merged');

CREATE TABLE issues (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    id_slug   TEXT NOT NULL,
    title     TEXT NOT NULL,
    status_id INTEGER NOT NULL DEFAULT 1 REFERENCES statuses(id)
);

CREATE TABLE branches (
    uuid       TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    issue_id   INTEGER NOT NULL REFERENCES issues(id),
    type       TEXT NOT NULL,
    status_id  INTEGER NOT NULL DEFAULT 1 REFERENCES statuses(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    merged_at  DATETIME
);

-- SQLite CHECK constraints cannot reference other tables,
-- so the merged_at invariant is enforced via a trigger.
CREATE TRIGGER enforce_merged_at
BEFORE UPDATE OF status_id ON branches
WHEN NEW.status_id = 2
BEGIN
    SELECT CASE WHEN NEW.merged_at IS NULL
        THEN RAISE(ABORT, 'merged_at must not be null when status is merged')
    END;
END;
