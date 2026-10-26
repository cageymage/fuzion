-- Set when an officer chooses main or alt, so the Blizzard roster sync's rank mapping
-- never overrides that choice. Hand-entered characters were chosen by an officer, and
-- the sync only ever inserted alts, so an existing main was promoted by hand.
ALTER TABLE characters ADD COLUMN is_main_set_by_hand boolean NOT NULL DEFAULT false;

UPDATE characters SET is_main_set_by_hand = true WHERE guild_rank IS NULL OR is_main;
