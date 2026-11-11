DROP TABLE rank_change_requests;

ALTER TABLE professions
    ADD COLUMN source text NOT NULL DEFAULT 'officer' CHECK (source IN ('officer', 'blizzard'));

UPDATE characters SET role = 'dps' WHERE role IS NULL;
ALTER TABLE characters ALTER COLUMN role SET NOT NULL;
