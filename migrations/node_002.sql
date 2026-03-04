-- Node schema v2: description-based offer/need model

-- Offers: description replaces qty + unit + notes + allergens
ALTER TABLE offers RENAME COLUMN notes TO description;
ALTER TABLE offers DROP COLUMN qty;
ALTER TABLE offers DROP COLUMN unit;
ALTER TABLE offers DROP COLUMN allergens;

-- Needs: remove qty/unit/time-window/priority — NEED is now a simple standing request
ALTER TABLE needs DROP COLUMN qty;
ALTER TABLE needs DROP COLUMN unit;
ALTER TABLE needs DROP COLUMN window_start;
ALTER TABLE needs DROP COLUMN window_end;
ALTER TABLE needs DROP COLUMN priority;
