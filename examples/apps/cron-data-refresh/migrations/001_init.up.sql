CREATE TABLE IF NOT EXISTS cached_payload (
    id           INTEGER PRIMARY KEY CHECK (id = 1),
    body         TEXT    NOT NULL,
    refreshed_at TEXT    NOT NULL DEFAULT (datetime('now'))
);
