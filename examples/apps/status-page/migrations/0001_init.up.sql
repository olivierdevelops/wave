CREATE TABLE IF NOT EXISTS services (
    name       TEXT PRIMARY KEY,
    status     TEXT NOT NULL DEFAULT 'operational',
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS log (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    service TEXT    NOT NULL,
    status  TEXT    NOT NULL,
    reason  TEXT    NOT NULL DEFAULT '',
    at      TEXT    NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_log_service ON log(service);
