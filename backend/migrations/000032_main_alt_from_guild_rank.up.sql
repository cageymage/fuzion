-- Main/alt now follows the in-game guild rank, with no hand-set override. Ranks 3
-- (Officer Alt) and 5 (Alt) are the alt ranks in internal/roster/ranks.go; this applies
-- them once so the roster is right before the next sync runs.
UPDATE characters SET is_main = guild_rank NOT IN (3, 5)
WHERE guild_rank IS NOT NULL;

ALTER TABLE characters DROP COLUMN is_main_set_by_hand;
