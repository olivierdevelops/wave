CREATE TABLE IF NOT EXISTS pastes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    slug       TEXT    NOT NULL UNIQUE,
    content    TEXT    NOT NULL,
    language   TEXT    NOT NULL DEFAULT 'plaintext',
    created_at TEXT    NOT NULL DEFAULT (datetime('now')),
    expires_at TEXT,
    view_limit INTEGER,
    views      INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_pastes_slug ON pastes(slug);
