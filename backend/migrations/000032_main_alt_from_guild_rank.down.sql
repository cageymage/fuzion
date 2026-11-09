-- The hand-set choices the up migration overwrote are gone, so every character comes
-- back as following its rank.
ALTER TABLE characters ADD COLUMN is_main_set_by_hand boolean NOT NULL DEFAULT false;
