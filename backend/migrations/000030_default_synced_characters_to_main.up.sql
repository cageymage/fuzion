UPDATE characters SET is_main = true
WHERE is_main_set_by_hand = false AND guild_rank IS NOT NULL;
