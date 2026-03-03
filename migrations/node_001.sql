-- Node schema v1
-- All timestamps are Unix epoch integers for portability.

CREATE TABLE IF NOT EXISTS actors (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    phone      TEXT    NOT NULL UNIQUE,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

CREATE TABLE IF NOT EXISTS offers (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    phone        TEXT    NOT NULL,
    qty          INTEGER NOT NULL,
    unit         TEXT    NOT NULL DEFAULT 'meals',
    window_start INTEGER,          -- Unix epoch; NULL = no window set
    window_end   INTEGER,
    location     TEXT    NOT NULL,
    notes        TEXT    NOT NULL DEFAULT '',
    allergens    TEXT    NOT NULL DEFAULT '',
    status       TEXT    NOT NULL DEFAULT 'awaiting_ready',
    -- status: awaiting_ready | ready | matched | done | canceled | expired
    created_at   INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at   INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

CREATE TABLE IF NOT EXISTS needs (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    phone        TEXT    NOT NULL,
    qty          INTEGER NOT NULL,
    unit         TEXT    NOT NULL DEFAULT 'meals',
    window_start INTEGER,
    window_end   INTEGER,
    location     TEXT    NOT NULL,
    priority     INTEGER NOT NULL DEFAULT 1,
    status       TEXT    NOT NULL DEFAULT 'open',
    -- status: open | matched | done | canceled | expired
    created_at   INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at   INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

CREATE TABLE IF NOT EXISTS jobs (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    offer_id     INTEGER NOT NULL REFERENCES offers(id),
    need_id      INTEGER NOT NULL REFERENCES needs(id),
    carrier      TEXT    NOT NULL DEFAULT 'manual',
    status       TEXT    NOT NULL DEFAULT 'matched',
    -- status: matched | dispatched | picked_up | delivered | failed | canceled
    match_score  REAL,
    match_reason TEXT    NOT NULL DEFAULT '',
    created_at   INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at   INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_offers_status ON offers(status);
CREATE INDEX IF NOT EXISTS idx_offers_phone  ON offers(phone);
CREATE INDEX IF NOT EXISTS idx_needs_status  ON needs(status);
CREATE INDEX IF NOT EXISTS idx_needs_phone   ON needs(phone);
CREATE INDEX IF NOT EXISTS idx_jobs_status   ON jobs(status);
