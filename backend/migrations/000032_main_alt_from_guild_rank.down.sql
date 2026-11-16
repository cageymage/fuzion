-- Hand-entered characters come back as hand-set, as migration 000025 marked them. The
-- hand-set choices on ranked characters were overwritten by the up migration, so those
-- come back as following their rank.
ALTER TABLE characters ADD COLUMN is_main_set_by_hand boolean NOT NULL DEFAULT false;
UPDATE characters SET is_main_set_by_hand = true WHERE guild_rank IS NULL;
