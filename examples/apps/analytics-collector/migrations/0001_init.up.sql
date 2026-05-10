CREATE TABLE IF NOT EXISTS events (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id   TEXT    NOT NULL,
    path      TEXT    NOT NULL,
    referrer  TEXT    NOT NULL DEFAULT '',
    ua_hash   TEXT    NOT NULL DEFAULT '',
    day       TEXT    NOT NULL DEFAULT (date('now')),
    at        TEXT    NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_events_site ON events(site_id);
CREATE INDEX IF NOT EXISTS idx_events_day  ON events(day);
CREATE INDEX IF NOT EXISTS idx_events_path ON events(path);
