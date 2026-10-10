-- guild_rank and left_guild_at are written only by the Blizzard roster sync.
-- A NULL guild_rank means the sync has never seen the character, so hand-entered
-- characters are never marked as having left. left_guild_at hides a character from
-- the roster instead of deleting it, so an officer can undo a bad sync.
ALTER TABLE characters
    ADD COLUMN guild_rank smallint CHECK (guild_rank >= 0),
    ADD COLUMN left_guild_at timestamptz;
