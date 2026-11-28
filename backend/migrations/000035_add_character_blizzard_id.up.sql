-- Blizzard's character ID is expected to stay the same through a rename, so the roster
-- sync matches on it before the name. NULL until a sync has seen the character.
ALTER TABLE characters ADD COLUMN blizzard_id bigint;

CREATE UNIQUE INDEX characters_blizzard_id ON characters (blizzard_id) WHERE blizzard_id IS NOT NULL;
