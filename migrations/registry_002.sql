-- Registry schema v2
-- Adds source column to track whether a record came from a direct announce ('local')
-- or was replicated from a peer registry ('federation').

ALTER TABLE nodes ADD COLUMN source TEXT NOT NULL DEFAULT 'local';
