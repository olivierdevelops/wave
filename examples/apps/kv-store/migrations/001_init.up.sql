CREATE TABLE IF NOT EXISTS kv (
    k          TEXT PRIMARY KEY,
    v          BLOB,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
