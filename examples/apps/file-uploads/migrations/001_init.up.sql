CREATE TABLE IF NOT EXISTS uploads (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    filename     TEXT NOT NULL,
    size         INTEGER NOT NULL,
    content_type TEXT,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
