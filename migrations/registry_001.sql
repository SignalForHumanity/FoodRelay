-- Registry schema v1
-- Stores ONLY node metadata — never offers, needs, or donor/recipient phones.

CREATE TABLE IF NOT EXISTS nodes (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id            TEXT    NOT NULL UNIQUE,
    public_phone       TEXT    NOT NULL,
    lat                REAL,
    lon                REAL,
    coverage_radius_km REAL    NOT NULL DEFAULT 10,
    capabilities       TEXT    NOT NULL DEFAULT '[]',  -- JSON array
    version            TEXT    NOT NULL DEFAULT '0.0.1',
    public_key         TEXT    NOT NULL,               -- hex-encoded ed25519 pubkey
    last_seen          INTEGER,                        -- Unix epoch
    created_at         INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at         INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_nodes_last_seen ON nodes(last_seen);
