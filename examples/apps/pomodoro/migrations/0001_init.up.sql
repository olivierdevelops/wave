CREATE TABLE IF NOT EXISTS rooms (
    name       TEXT    PRIMARY KEY,
    state      TEXT    NOT NULL DEFAULT 'idle',
    remaining  INTEGER NOT NULL DEFAULT 1500,
    started_at INTEGER NOT NULL DEFAULT 0
);
