-- Registry schema v3
-- Stores the original announce signature and its timestamp so that peer registries
-- can verify a node record before inserting it (federation propagation).

ALTER TABLE nodes ADD COLUMN announce_sig TEXT NOT NULL DEFAULT '';
ALTER TABLE nodes ADD COLUMN announce_ts  INTEGER NOT NULL DEFAULT 0;
