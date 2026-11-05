-- Who last wrote a profession's skill level. The Blizzard sync overwrites a profession
-- it reports and marks it 'blizzard'; rows it never reports stay as officers left them.
ALTER TABLE professions
    ADD COLUMN source text NOT NULL DEFAULT 'officer' CHECK (source IN ('officer', 'blizzard'));
